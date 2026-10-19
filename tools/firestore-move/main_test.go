package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func TestDailyOf(t *testing.T) {
	for _, tc := range []struct {
		path, site, day string
		ok              bool
	}{
		{"RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-19", "KC Wu", "2026-10-19", true},
		{"RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19", "AllSites", "2026-10-19", true},
		// Another app's DailyStats must never count as the balancer's.
		{"Accounts/abc/DailyStats/2026-10-19", "", "", false},
		{"RudyMirrorSites/KC Wu", "", "", false},
		{"RudyMirrorSiteStats/KC Wu", "", "", false},
	} {
		site, day, ok := dailyOf(tc.path)
		if site != tc.site || day != tc.day || ok != tc.ok {
			t.Errorf("dailyOf(%q) = %q, %q, %v; want %q, %q, %v", tc.path, site, day, ok, tc.site, tc.day, tc.ok)
		}
	}
}

func TestIncrements(t *testing.T) {
	const d = "RudyMirrorSiteStats/AllSites/DailyStats/"
	src := counts{
		d + "2026-10-18": {Count: 10, Size: 100}, // before since: ignored
		d + "2026-10-19": {Count: 15, Size: 150}, // grew by 5 since the copy
		d + "2026-10-20": {Count: 3, Size: 30},   // created after the copy
		d + "2026-10-21": {Count: 7, Size: 70},   // unchanged since the copy
	}
	snapshot := counts{
		d + "2026-10-18": {Count: 1, Size: 10},
		d + "2026-10-19": {Count: 10, Size: 100},
		d + "2026-10-21": {Count: 7, Size: 70},
	}
	got, err := increments(src, snapshot, "2026-10-19")
	if err != nil {
		t.Fatal(err)
	}
	want := counts{
		d + "2026-10-19": {Count: 5, Size: 50},
		d + "2026-10-20": {Count: 3, Size: 30},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("increments = %v, want %v", got, want)
	}

	// A source below its snapshot: wrong snapshot or wrong direction.
	if _, err := increments(counts{d + "2026-10-19": {Count: 9, Size: 100}}, snapshot, "2026-10-19"); err == nil {
		t.Error("increments accepted a source count below its snapshot")
	}
}

func TestDeltaRejectsMalformedSince(t *testing.T) {
	done := filepath.Join(t.TempDir(), "done.json")
	for _, since := range []string{"2026-10-9", "20261019", "2026/10/19", ""} {
		if err := runDelta(context.Background(), nil, nil, "snapshot.json", since, done); err == nil {
			t.Errorf("runDelta accepted -since %q", since)
		}
	}
	if _, err := os.Stat(done); err == nil {
		t.Error("a rejected delta still recorded itself as done")
	}
}

// TestAgainstEmulator runs copy, verify, a rollback-style refusal and delta
// against the Firestore emulator, two projects in one emulator. It is
// skipped unless FIRESTORE_EMULATOR_HOST is set.
func TestAgainstEmulator(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set")
	}
	ctx := context.Background()
	suffix := time.Now().Format("150405.000000")
	src, err := firestore.NewClient(ctx, "src-"+suffix[len(suffix)-6:])
	if err != nil {
		t.Fatal(err)
	}
	dst, err := firestore.NewClient(ctx, "dst-"+suffix[len(suffix)-6:])
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	set := func(c *firestore.Client, path string, data map[string]interface{}) {
		t.Helper()
		if _, err := c.Doc(path).Set(ctx, data); err != nil {
			t.Fatal(err)
		}
	}
	set(src, "RudyMirrorSites/KC Wu", map[string]interface{}{"LastCheck": day})
	set(src, "RudyMirrorFiles/abc", map[string]interface{}{"File": "x.zip", "Sites": map[string]interface{}{"KC Wu": int64(5)}})
	set(src, "RudyClientSalt/Current", map[string]interface{}{"Quarter": "2026Q4", "Salt": []byte{1, 2, 3}})
	set(src, "RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19", map[string]interface{}{"Time": day, "Count": int64(10), "Size": int64(100)})
	set(src, "RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-19", map[string]interface{}{"Time": day, "Count": int64(4), "Size": int64(40)})
	// Another app's data with the same subcollection name: never copied.
	set(src, "Accounts/someone/DailyStats/2026-10-19", map[string]interface{}{"Count": int64(99)})

	dir := t.TempDir()
	snap := filepath.Join(dir, "snapshot.json")
	if err := runCopy(ctx, src, dst, snap); err != nil {
		t.Fatal(err)
	}
	if err := runVerify(ctx, src, dst); err != nil {
		t.Fatalf("verify after copy: %v", err)
	}
	// Independently of docs(), which copy and verify share: every seeded
	// document is in the destination with its contents, the salt's bytes
	// included.
	for path, want := range map[string]map[string]interface{}{
		"RudyMirrorSites/KC Wu":                           {"LastCheck": day},
		"RudyMirrorFiles/abc":                             {"File": "x.zip", "Sites": map[string]interface{}{"KC Wu": int64(5)}},
		"RudyClientSalt/Current":                          {"Quarter": "2026Q4", "Salt": []byte{1, 2, 3}},
		"RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-19": {"Time": day, "Count": int64(4), "Size": int64(40)},
	} {
		d, err := dst.Doc(path).Get(ctx)
		if err != nil {
			t.Fatalf("%s not copied: %v", path, err)
		}
		if got := d.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", path, got, want)
		}
	}
	if d, err := dst.Doc("Accounts/someone/DailyStats/2026-10-19").Get(ctx); err == nil && d.Exists() {
		t.Error("copied another app's DailyStats")
	}
	if err := runCopy(ctx, src, dst, filepath.Join(dir, "again.json")); err == nil {
		t.Error("copy ran over a destination that already holds data")
	}

	// Both sides keep counting after the copy.
	if _, err := src.Doc("RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19").Update(ctx, []firestore.Update{{Path: "Count", Value: firestore.Increment(3)}, {Path: "Size", Value: firestore.Increment(30)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.Doc("RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19").Update(ctx, []firestore.Update{{Path: "Count", Value: firestore.Increment(2)}, {Path: "Size", Value: firestore.Increment(20)}}); err != nil {
		t.Fatal(err)
	}
	set(src, "RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-20", map[string]interface{}{"Time": day.AddDate(0, 0, 1), "Count": int64(1), "Size": int64(10)})

	done := filepath.Join(dir, "done.json")
	if err := runDelta(ctx, src, dst, snap, "2026-10-19", done); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]stat{
		"RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19": {15, 150}, // copied + new + old's increase
		"RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-19":    {4, 40},   // unchanged
		"RudyMirrorSiteStats/KC Wu/DailyStats/2026-10-20":    {1, 10},   // created after the copy
	} {
		d, err := dst.Doc(path).Get(ctx)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if got := (stat{asInt(d.Data()["Count"]), asInt(d.Data()["Size"])}); got != want {
			t.Errorf("%s = %+v, want %+v", path, got, want)
		}
	}
	if err := runDelta(ctx, src, dst, snap, "2026-10-19", done); err == nil {
		t.Error("a second delta ran")
	}
}
