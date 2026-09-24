package dao

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"service/config"
	"service/db"
)

func TestQuarter(t *testing.T) {
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
	ctx := context.Background()
	ref := db.Client().Collection(rudyClientSalt).Doc(currentSalt)
	if _, err := ref.Delete(ctx); err != nil {
		t.Fatal(err)
	}

	// Instances starting together on an empty store end up with one salt.
	const n = 8
	salts := make([][]byte, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			salts[i], errs[i] = loadSalt(ctx, "2026-Q3")
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
	q3 := salts[0]

	// A new quarter replaces the salt, destroying the old one.
	q4, err := loadSalt(ctx, "2026-Q4")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(q4, q3) {
		t.Fatal("new quarter kept the old salt")
	}
	if again, err := loadSalt(ctx, "2026-Q4"); err != nil || !bytes.Equal(again, q4) {
		t.Fatalf("same quarter should reuse the salt, got err %v", err)
	}

	// A lagging clock cannot roll it back.
	if _, err := loadSalt(ctx, "2026-Q3"); err == nil {
		t.Fatal("earlier quarter should be refused")
	}
	doc, err := ref.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stored := &clientSalt{}
	if err := doc.DataTo(stored); err != nil {
		t.Fatal(err)
	}
	if stored.Quarter != "2026-Q4" || !bytes.Equal(stored.Salt, q4) {
		t.Fatalf("stored salt changed to %s", stored.Quarter)
	}
}
