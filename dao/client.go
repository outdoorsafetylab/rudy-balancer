package dao

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"service/db"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reports cut quarters on Asia/Taipei calendar days; Taiwan has no DST, so a
// fixed zone needs no tzdata in the image.
var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

// Quarter names the report quarter t falls in, e.g. "2026-Q4".
func Quarter(t time.Time) string {
	t = t.In(taipei)
	return fmt.Sprintf("%d-Q%d", t.Year(), (int(t.Month())-1)/3+1)
}

// clientSalt is the one Firestore document every instance shares. It holds
// only the current quarter's salt: rolling to a new quarter overwrites the
// old one, so a quarter's Client ids cannot be recomputed once it is over.
type clientSalt struct {
	Quarter string
	Salt    []byte
}

const (
	// saltTimeout bounds a Firestore round trip made while holding the cache
	// lock, so a stuck Firestore cannot pile up every logging goroutine.
	saltTimeout = 5 * time.Second
	// saltRetry is how long a failed quarter is not retried; its downloads
	// are logged without Client meanwhile.
	saltRetry = 30 * time.Second
)

var (
	saltCache struct {
		sync.Mutex
		quarter string
		salt    []byte
		failed  string
		retryAt time.Time
		err     error
	}
	// load and now are replaced in tests.
	load = loadSalt
	now  = time.Now
)

// ClientSalt returns the salt for quarter q, cached per instance. The salt
// never goes into a log.
func ClientSalt(ctx context.Context, q string) ([]byte, error) {
	saltCache.Lock()
	defer saltCache.Unlock()
	if saltCache.quarter == q {
		return saltCache.salt, nil
	}
	if saltCache.failed == q && now().Before(saltCache.retryAt) {
		return nil, saltCache.err
	}
	ctx, cancel := context.WithTimeout(ctx, saltTimeout)
	defer cancel()
	salt, err := load(ctx, q)
	if err != nil {
		saltCache.failed, saltCache.retryAt, saltCache.err = q, now().Add(saltRetry), err
		return nil, err
	}
	saltCache.quarter, saltCache.salt = q, append([]byte(nil), salt...)
	saltCache.failed = ""
	return saltCache.salt, nil
}

// serverTime is Firestore's clock as seen in a read; tests shift it.
var serverTime = func(readTime time.Time) time.Time { return readTime }

// loadSalt returns the salt for quarter q. Only Firestore's clock decides
// which quarter is current, so an instance whose clock runs ahead cannot
// destroy the salt early, nor one running behind roll it back; a request
// whose quarter is not the current one gets an error and no Client. The
// salt is created in the same transaction that finds it missing or stale,
// so concurrent instances agree on one.
func loadSalt(ctx context.Context, q string) ([]byte, error) {
	ref := db.Client().Collection(rudyClientSalt).Doc(currentSalt)
	var salt []byte
	err := db.Client().RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		salt = nil
		doc, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if doc == nil || doc.ReadTime.IsZero() {
			return fmt.Errorf("no read time for client salt")
		}
		current := Quarter(serverTime(doc.ReadTime))
		if q != current {
			return fmt.Errorf("quarter %s is not the current quarter %s", q, current)
		}
		if doc.Exists() {
			stored := &clientSalt{}
			if err := doc.DataTo(stored); err != nil {
				return err
			}
			if stored.Quarter == current && len(stored.Salt) > 0 {
				salt = stored.Salt
				return nil
			}
		}
		fresh := make([]byte, 32)
		if _, err := rand.Read(fresh); err != nil {
			return err
		}
		if err := tx.Set(ref, &clientSalt{Quarter: current, Salt: fresh}); err != nil {
			return err
		}
		salt = fresh
		return nil
	})
	if err != nil {
		return nil, err
	}
	return salt, nil
}
