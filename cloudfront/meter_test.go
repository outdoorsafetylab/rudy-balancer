package cloudfront

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"service/config"
	"service/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type fakeAPI struct {
	mu    sync.Mutex
	calls []*cloudwatch.GetMetricStatisticsInput
	sums  map[string][]float64
	err   error
}

func (f *fakeAPI) GetMetricStatistics(ctx context.Context, in *cloudwatch.GetMetricStatisticsInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	out := &cloudwatch.GetMetricStatisticsOutput{}
	for _, v := range f.sums[aws.ToString(in.Dimensions[0].Value)] {
		out.Datapoints = append(out.Datapoints, types.Datapoint{Sum: aws.Float64(v)})
	}
	return out, nil
}

func newMeter(api *fakeAPI, clock *time.Time) *Meter {
	return &Meter{api: api, distributions: []string{"DIST1", "DIST2"}, now: func() time.Time { return *clock }}
}

// read triggers any due refresh, waits for it, and reads again.
func read(m *Meter) (int64, bool) {
	m.MonthToDate()
	m.wg.Wait()
	return m.MonthToDate()
}

func TestMeter(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 9, 25, 3, 0, 0, 0, time.UTC)
	api := &fakeAPI{sums: map[string][]float64{"DIST1": {100, 200.5}, "DIST2": {50}}}
	m := newMeter(api, &clock)

	if _, ok := m.MonthToDate(); ok {
		t.Fatal("no reading may be reported before the first refresh")
	}
	m.wg.Wait()
	if got, ok := m.MonthToDate(); !ok || got != 350 {
		t.Fatalf("want 350 summed over both distributions, got %d, %v", got, ok)
	}
	if len(api.calls) != 2 {
		t.Fatalf("want one call per distribution, got %d", len(api.calls))
	}
	in := api.calls[0]
	if got := aws.ToString(in.Namespace) + "/" + aws.ToString(in.MetricName); got != "AWS/CloudFront/BytesDownloaded" {
		t.Errorf("metric %s", got)
	}
	if len(in.Dimensions) != 2 || aws.ToString(in.Dimensions[1].Name) != "Region" || aws.ToString(in.Dimensions[1].Value) != "Global" {
		t.Errorf("CloudFront metrics need the Region=Global dimension: %+v", in.Dimensions)
	}
	if want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC); !aws.ToTime(in.StartTime).Equal(want) {
		t.Errorf("start %s, want the first of the UTC month", aws.ToTime(in.StartTime))
	}
	if len(in.Statistics) != 1 || in.Statistics[0] != types.StatisticSum || aws.ToInt32(in.Period) != 86400 {
		t.Errorf("want daily sums, got %v every %d s", in.Statistics, aws.ToInt32(in.Period))
	}

	// Within refreshEvery the cached reading is reused.
	clock = clock.Add(refreshEvery - time.Second)
	api.sums["DIST1"] = []float64{1000}
	if got, _ := read(m); got != 350 || len(api.calls) != 2 {
		t.Fatalf("want cached 350 and no new calls, got %d after %d calls", got, len(api.calls))
	}
	clock = clock.Add(time.Second)
	if got, _ := read(m); got != 1050 || len(api.calls) != 4 {
		t.Fatalf("want refreshed 1050, got %d after %d calls", got, len(api.calls))
	}

	// A failing refresh keeps the last reading of the same month and is not
	// retried before refreshEvery.
	api.err = errors.New("throttled")
	clock = clock.Add(refreshEvery)
	if got, ok := read(m); !ok || got != 1050 {
		t.Fatalf("want last reading kept, got %d, %v", got, ok)
	}
	calls := len(api.calls)
	clock = clock.Add(time.Minute)
	read(m)
	if len(api.calls) != calls {
		t.Fatal("a failed refresh was retried before refreshEvery")
	}

	// Months are UTC, as AWS bills: 07:00 on 10/1 in Taipei is still
	// September, so September's reading stands.
	taipei := time.FixedZone("Asia/Taipei", 8*60*60)
	clock = time.Date(2026, 10, 1, 7, 0, 0, 0, taipei)
	if got, ok := m.MonthToDate(); !ok || got != 1050 {
		t.Fatalf("Taipei's October 1st is September in UTC; got %d, %v", got, ok)
	}
	m.wg.Wait()

	// A new UTC month has no reading until a refresh succeeds, so the old
	// month's usage cannot keep a site out.
	clock = time.Date(2026, 10, 1, 0, 5, 0, 0, time.UTC)
	if _, ok := read(m); ok {
		t.Fatal("last month's reading reported for the new month")
	}
	api.err = nil
	clock = clock.Add(refreshEvery)
	if _, ok := read(m); !ok {
		t.Fatal("want a reading once CloudWatch answers again")
	}
	if want := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC); !aws.ToTime(api.calls[len(api.calls)-1].StartTime).Equal(want) {
		t.Errorf("new month should start at %s", want)
	}
}
