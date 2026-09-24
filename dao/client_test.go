package dao

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"service/config"
	"service/db"
)

func TestQuarter(t *testing.T) {
	// Quarter must not depend on the machine's zone.
	defer func(l *time.Location) { time.Local = l }(time.Local)
	time.Local = time.FixedZone("PDT", -7*60*60)
	tests := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 9, 30, 15, 59, 59, 0, time.UTC), "2026-Q3"}, // 23:59:59 in Taipei
		{time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC), "2026-Q4"},   // 00:00 on 10/1 in Taipei
		{time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC), "2027-Q1"},
		{time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC), "2027-Q2"},
		{time.Date(2027, 7, 15, 0, 0, 0, 0, time.UTC), "2027-Q3"},
	}
	for _, tt := range tests {
		if got := Quarter(tt.t); got != tt.want {
			t.Errorf("Quarter(%s) = %s, want %s", tt.t, got, tt.want)
		}
	}
}

func resetSaltCache() {
	saltCache.quarter, saltCache.salt, saltCache.failed, saltCache.err = "", nil, "", nil
}

func TestClientSalt(t *testing.T) {
	defer func(l func(context.Context, string) ([]byte, error), n func() time.Time) { load, now = l, n }(load, now)
	resetSaltCache()
	defer resetSaltCache()
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return clock }
	calls := 0
	fail := false
	load = func(ctx context.Context, q string) ([]byte, error) {
		calls++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("Firestore call made without a deadline")
		}
		if fail {
			return nil, errors.New("firestore down")
		}
		return []byte("salt " + q), nil
	}

	for i := 0; i < 3; i++ {
		if s, err := ClientSalt(context.Background(), "2026-Q3"); err != nil || string(s) != "salt 2026-Q3" {
			t.Fatalf("got %q, %v", s, err)
		}
	}
	if calls != 1 {
		t.Fatalf("want 1 load for a cached quarter, got %d", calls)
	}
	if s, _ := ClientSalt(context.Background(), "2026-Q4"); string(s) != "salt 2026-Q4" || calls != 2 {
		t.Fatalf("new quarter should load its own salt, got %q after %d loads", s, calls)
	}

	// A failure is not retried until saltRetry passes, so a stuck Firestore
	// costs one timeout per retry period, not one per download.
	fail = true
	for i := 0; i < 5; i++ {
		if _, err := ClientSalt(context.Background(), "2027-Q1"); err == nil {
			t.Fatal("want error")
		}
	}
	if calls != 3 {
		t.Fatalf("want 1 load while failing, got %d", calls-2)
	}
	fail = false
	clock = clock.Add(saltRetry)
	if s, err := ClientSalt(context.Background(), "2027-Q1"); err != nil || string(s) != "salt 2027-Q1" {
		t.Fatalf("retry after saltRetry: got %q, %v", s, err)
	}
}

// TestLoadSalt needs the Firestore emulator and never runs against a real
// project:
//
//	gcloud emulators firestore start --host-port=localhost:8681
//	FIRESTORE_EMULATOR_HOST=localhost:8681 go test ./dao -run TestLoadSalt
func TestLoadSalt(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set")
	}
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := db.Init(); err != nil {
		t.Fatal(err)
	}
	defer db.Deinit()
	defer func(f func(time.Time) time.Time) { serverTime = f }(serverTime)
	ctx := context.Background()
	ref := db.Client().Collection(rudyClientSalt).Doc(currentSalt)
	if _, err := ref.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	stored := func() *clientSalt {
		doc, err := ref.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := &clientSalt{}
		if err := doc.DataTo(s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	cur := Quarter(time.Now())
	next := Quarter(time.Now().AddDate(0, 3, 0))
	prev := Quarter(time.Now().AddDate(0, -3, 0))

	// Instances starting together on an empty store end up with the one
	// salt that was stored.
	const n = 8
	salts := make([][]byte, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			salts[i], errs[i] = loadSalt(ctx, cur)
		}(i)
	}
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("instance %d: %v", i, errs[i])
		}
		if len(salts[i]) != 32 || !bytes.Equal(salts[i], salts[0]) {
			t.Fatalf("instance %d got a different salt", i)
		}
	}
	if s := stored(); s.Quarter != cur || !bytes.Equal(s.Salt, salts[0]) {
		t.Fatalf("stored %s salt differs from the one handed out", s.Quarter)
	}
	old := salts[0]

	// Clocks off either way get no salt and change nothing.
	for _, q := range []string{next, prev} {
		if _, err := loadSalt(ctx, q); err == nil {
			t.Errorf("quarter %s accepted while Firestore says %s", q, cur)
		}
		if s := stored(); s.Quarter != cur || !bytes.Equal(s.Salt, old) {
			t.Fatalf("request for %s changed the stored salt", q)
		}
	}

	// Once Firestore's clock reaches the next quarter, the salt is replaced
	// and the old one is gone.
	serverTime = func(t time.Time) time.Time { return t.AddDate(0, 3, 0) }
	fresh, err := loadSalt(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(fresh, old) {
		t.Fatal("new quarter kept the old salt")
	}
	if s := stored(); s.Quarter != next || !bytes.Equal(s.Salt, fresh) {
		t.Fatalf("stored %s after rolling to %s", s.Quarter, next)
	}
	if again, err := loadSalt(ctx, next); err != nil || !bytes.Equal(again, fresh) {
		t.Fatalf("same quarter should reuse the salt, got err %v", err)
	}
	if _, err := loadSalt(ctx, cur); err == nil {
		t.Fatal("a request from the previous quarter should get no salt")
	}
}
