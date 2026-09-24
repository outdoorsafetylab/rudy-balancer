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

var saltCache struct {
	sync.Mutex
	quarter string
	salt    []byte
}

// ClientSalt returns the salt for quarter q, cached per instance. The salt
// never goes into a log.
func ClientSalt(ctx context.Context, q string) ([]byte, error) {
	saltCache.Lock()
	defer saltCache.Unlock()
	if saltCache.quarter == q {
		return saltCache.salt, nil
	}
	salt, err := loadSalt(ctx, q)
	if err != nil {
		return nil, err
	}
	saltCache.quarter, saltCache.salt = q, salt
	return salt, nil
}

// loadSalt reads the stored salt, creating one in the same transaction when it
// is missing or from an earlier quarter, so concurrent instances agree on one.
func loadSalt(ctx context.Context, q string) ([]byte, error) {
	ref := db.Client().Collection(rudyClientSalt).Doc(currentSalt)
	var salt []byte
	err := db.Client().RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil {
			stored := &clientSalt{}
			if err := doc.DataTo(stored); err != nil {
				return err
			}
			if stored.Quarter == q && len(stored.Salt) > 0 {
				salt = stored.Salt
				return nil
			}
			// A clock running behind must not roll the salt back and
			// destroy the current quarter's.
			if stored.Quarter > q {
				return fmt.Errorf("stored client salt is for %s, after %s", stored.Quarter, q)
			}
		}
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return err
		}
		return tx.Set(ref, &clientSalt{Quarter: q, Salt: salt})
	})
	if err != nil {
		return nil, err
	}
	return salt, nil
}
