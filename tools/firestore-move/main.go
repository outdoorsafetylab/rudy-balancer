// Command firestore-move copies the balancer's Firestore data from one project
// to another, by path, for the move to rudymap (#32,
// docs/proposal/move_to_rudymap_plan.md D4). It is deleted once the move is
// done.
//
// It touches only the balancer's paths: the top-level collections in
// topLevel, and the DailyStats subcollection under each RudyMirrorSiteStats
// document. Another app in the same database has DailyStats subcollections of
// its own, which is why this does not use a collection-group export.
//
//	firestore-move copy   -from P -to Q -snapshot FILE
//	firestore-move verify -from P -to Q
//	firestore-move delta  -from P -to Q -snapshot FILE -since YYYY-MM-DD -done FILE
//
// copy refuses a destination that already holds any of these paths, and writes
// the copied DailyStats counts to FILE. delta adds to Q what P counted since the
// copy, per DailyStats document from -since on; run P=old, Q=new to finish a
// cutover, or P=new, Q=old to roll one back. Only one delta is ever applied:
// -done records it, and delta refuses to run once that file exists.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var topLevel = []string{"RudyMirrorSites", "RudyMirrorFiles", "RudyClientSalt", "RudyMirrorSiteStats"}

const (
	siteStats  = "RudyMirrorSiteStats"
	dailyStats = "DailyStats"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: firestore-move copy|verify|delta [flags]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	from := fs.String("from", "", "source project")
	to := fs.String("to", "", "destination project")
	snapshot := fs.String("snapshot", "", "file of the DailyStats counts as copied")
	since := fs.String("since", "", "first DailyStats day (UTC, YYYY-MM-DD) delta considers")
	done := fs.String("done", "", "file recording the one delta applied")
	_ = fs.Parse(os.Args[2:])
	if *from == "" || *to == "" || *from == *to {
		fmt.Fprintln(os.Stderr, "-from and -to must name two different projects")
		os.Exit(2)
	}
	ctx := context.Background()
	src, err := firestore.NewClient(ctx, *from)
	check(err)
	defer src.Close()
	dst, err := firestore.NewClient(ctx, *to)
	check(err)
	defer dst.Close()

	switch os.Args[1] {
	case "copy":
		need(*snapshot != "", "-snapshot")
		check(runCopy(ctx, src, dst, *snapshot))
	case "verify":
		check(runVerify(ctx, src, dst))
	case "delta":
		need(*snapshot != "" && *since != "" && *done != "", "-snapshot, -since and -done")
		check(runDelta(ctx, src, dst, *snapshot, *since, *done))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

func need(ok bool, what string) {
	if !ok {
		fmt.Fprintf(os.Stderr, "%s required\n", what)
		os.Exit(2)
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// docs reads every document in the balancer's paths, keyed by its path
// relative to the database root ("RudyMirrorSites/KC Wu",
// "RudyMirrorSiteStats/AllSites/DailyStats/2026-10-19").
func docs(ctx context.Context, c *firestore.Client) (map[string]map[string]interface{}, error) {
	out := make(map[string]map[string]interface{})
	add := func(it *firestore.DocumentIterator, prefix string) error {
		defer it.Stop()
		for {
			d, err := it.Next()
			if err == iterator.Done {
				return nil
			}
			if err != nil {
				return err
			}
			out[prefix+"/"+d.Ref.ID] = d.Data()
		}
	}
	for _, col := range topLevel {
		if err := add(c.Collection(col).Documents(ctx), col); err != nil {
			return nil, fmt.Errorf("%s: %w", col, err)
		}
	}
	// DocumentRefs also lists parents that hold only subcollections, which
	// is what most RudyMirrorSiteStats documents are.
	refs := c.Collection(siteStats).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", siteStats, err)
		}
		prefix := siteStats + "/" + ref.ID + "/" + dailyStats
		if err := add(ref.Collection(dailyStats).Documents(ctx), prefix); err != nil {
			return nil, fmt.Errorf("%s: %w", prefix, err)
		}
	}
	return out, nil
}

// counts is what delta works from: Count and Size of every DailyStats
// document.
type counts map[string]stat

type stat struct {
	Count int64
	Size  int64
}

func statsOf(all map[string]map[string]interface{}) counts {
	out := make(counts)
	for path, data := range all {
		if !isDaily(path) {
			continue
		}
		out[path] = stat{Count: asInt(data["Count"]), Size: asInt(data["Size"])}
	}
	return out
}

func isDaily(path string) bool {
	_, _, ok := dailyOf(path)
	return ok
}

// dailyOf splits "RudyMirrorSiteStats/{site}/DailyStats/{day}".
func dailyOf(path string) (site, day string, ok bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[0] != siteStats || parts[2] != dailyStats {
		return "", "", false
	}
	return parts[1], parts[3], true
}

func asInt(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	}
	return 0
}

// increments is what delta adds to the destination: for every source
// DailyStats document from since on, the source's count minus the count it
// had when copied (zero for a document created after the copy). A source
// count below its snapshot means the wrong snapshot or the wrong direction,
// and is an error.
func increments(src, snapshot counts, since string) (counts, error) {
	out := make(counts)
	for path, now := range src {
		_, day, _ := dailyOf(path)
		if day < since {
			continue
		}
		base := snapshot[path]
		inc := stat{Count: now.Count - base.Count, Size: now.Size - base.Size}
		if inc.Count < 0 || inc.Size < 0 {
			return nil, fmt.Errorf("%s: source %+v is below its snapshot %+v", path, now, base)
		}
		if inc != (stat{}) {
			out[path] = inc
		}
	}
	return out, nil
}

func runCopy(ctx context.Context, src, dst *firestore.Client, snapshot string) error {
	if _, err := os.Stat(snapshot); err == nil {
		return fmt.Errorf("%s exists: copy has already run", snapshot)
	}
	existing, err := docs(ctx, dst)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return fmt.Errorf("destination already holds %d documents in these paths; not overwriting", len(existing))
	}
	all, err := docs(ctx, src)
	if err != nil {
		return err
	}
	// The snapshot is written first: a copy that fails halfway leaves it
	// behind, and the next copy then refuses to run over a partial one.
	b, err := json.MarshalIndent(statsOf(all), "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(snapshot, b, 0o600); err != nil {
		return err
	}
	paths := sortedKeys(all)
	for start := 0; start < len(paths); start += 400 {
		end := start + 400
		if end > len(paths) {
			end = len(paths)
		}
		batch := dst.Batch()
		for _, p := range paths[start:end] {
			batch.Set(dst.Doc(p), all[p])
		}
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("writing %s…: %w", paths[start], err)
		}
	}
	fmt.Printf("copied %d documents (%d DailyStats)\n", len(all), len(statsOf(all)))
	return nil
}

func runVerify(ctx context.Context, src, dst *firestore.Client) error {
	a, err := docs(ctx, src)
	if err != nil {
		return err
	}
	b, err := docs(ctx, dst)
	if err != nil {
		return err
	}
	diffs := 0
	for _, p := range sortedKeys(a) {
		if _, ok := b[p]; !ok {
			fmt.Println("missing in destination:", p)
			diffs++
		} else if !reflect.DeepEqual(a[p], b[p]) {
			fmt.Println("differs:", p)
			diffs++
		}
	}
	for _, p := range sortedKeys(b) {
		if _, ok := a[p]; !ok {
			fmt.Println("only in destination:", p)
			diffs++
		}
	}
	fmt.Printf("%d documents in source, %d in destination, %d differences\n", len(a), len(b), diffs)
	if diffs > 0 {
		return errors.New("source and destination differ")
	}
	return nil
}

func runDelta(ctx context.Context, src, dst *firestore.Client, snapshot, since, done string) error {
	// Days compare as strings, so a malformed -since ("2026-10-9") would skip
	// days silently and still mark the one delta as done.
	if _, err := time.Parse("2006-01-02", since); err != nil {
		return fmt.Errorf("-since %q is not YYYY-MM-DD", since)
	}
	if _, err := os.Stat(done); err == nil {
		return fmt.Errorf("%s exists: a delta has already been applied", done)
	}
	b, err := os.ReadFile(snapshot)
	if err != nil {
		return err
	}
	base := make(counts)
	if err := json.Unmarshal(b, &base); err != nil {
		return err
	}
	all, err := docs(ctx, src)
	if err != nil {
		return err
	}
	inc, err := increments(statsOf(all), base, since)
	if err != nil {
		return err
	}
	// Recorded before writing, so an interrupted run cannot be repeated
	// over itself; the file says what it was going to add.
	rec, err := json.MarshalIndent(inc, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(done, rec, 0o600); err != nil {
		return err
	}
	for _, p := range sortedKeys(inc) {
		update := map[string]interface{}{
			"Count": firestore.Increment(inc[p].Count),
			"Size":  firestore.Increment(inc[p].Size),
		}
		if t, ok := all[p]["Time"]; ok {
			update["Time"] = t
		}
		if _, err := dst.Doc(p).Set(ctx, update, firestore.MergeAll); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		fmt.Printf("%s +%d (%d bytes)\n", p, inc[p].Count, inc[p].Size)
	}
	fmt.Printf("applied %d increments\n", len(inc))
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
