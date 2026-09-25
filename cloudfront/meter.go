// Package cloudfront meters how much the AWS account's CloudFront has served
// this calendar month, which is what CloudFront's always-free tier (1 TB of
// data transfer a month) is measured on. A site with a MonthlyQuota stops
// receiving downloads once the meter passes it.
package cloudfront

import (
	"context"
	"strings"
	"sync"
	"time"

	"service/config"
	"service/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	// refreshEvery is how stale the reading may get. CloudWatch's free tier
	// covers a million GetMetricStatistics calls a month; one call per
	// distribution every 10 minutes per instance is far below it.
	refreshEvery = 10 * time.Minute
	// fetchTimeout bounds one refresh.
	fetchTimeout = 30 * time.Second
)

// statsAPI is the one CloudWatch call the meter makes; tests fake it.
// GetMetricStatistics, unlike GetMetricData, is covered by CloudWatch's free
// tier.
type statsAPI interface {
	GetMetricStatistics(ctx context.Context, in *cloudwatch.GetMetricStatisticsInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error)
}

// Meter caches the month-to-date BytesDownloaded summed over the account's
// distributions. Readers never wait on CloudWatch: a stale reading starts a
// refresh in the background and the old value is used meanwhile.
type Meter struct {
	api           statsAPI
	distributions []string
	now           func() time.Time

	mu        sync.Mutex
	month     string // UTC calendar month of bytes, e.g. "2026-09"
	bytes     int64
	attempted time.Time
	running   bool
	wg        sync.WaitGroup
}

// MonthToDate returns the bytes served this UTC calendar month (AWS bills by
// UTC month), and false when there is no reading for this month yet, e.g.
// right after start-up, at the turn of the month, or while CloudWatch keeps
// failing. Callers must not enforce a quota on a false reading.
func (m *Meter) MonthToDate() (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if !m.running && now.Sub(m.attempted) >= refreshEvery {
		m.running = true
		m.attempted = now
		m.wg.Add(1)
		go m.refresh()
	}
	if m.month != month(now) {
		return 0, false
	}
	return m.bytes, true
}

func (m *Meter) refresh() {
	defer m.wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	now := m.now()
	bytes, err := m.fetch(ctx, now)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	if err != nil {
		// Keep the last reading; if it is from an earlier month,
		// MonthToDate reports no reading and the quota is not enforced.
		log.Warningf("Failed to read CloudFront usage: %s", err.Error())
		return
	}
	m.month, m.bytes = month(now), bytes
}

func (m *Meter) fetch(ctx context.Context, now time.Time) (int64, error) {
	now = now.UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var total float64
	for _, id := range m.distributions {
		out, err := m.api.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/CloudFront"),
			MetricName: aws.String("BytesDownloaded"),
			Dimensions: []types.Dimension{
				{Name: aws.String("DistributionId"), Value: aws.String(id)},
				{Name: aws.String("Region"), Value: aws.String("Global")},
			},
			StartTime:  aws.Time(start),
			EndTime:    aws.Time(now),
			Period:     aws.Int32(86400),
			Statistics: []types.Statistic{types.StatisticSum},
		})
		if err != nil {
			return 0, err
		}
		for _, p := range out.Datapoints {
			if p.Sum != nil {
				total += *p.Sum
			}
		}
	}
	return int64(total), nil
}

func month(t time.Time) string {
	return t.UTC().Format("2006-01")
}

var (
	defaultOnce  sync.Once
	defaultMeter *Meter
)

// Default is the meter for the distributions listed in config
// cloudfront.distributions (comma-separated IDs, e.g. from the
// CLOUDFRONT_DISTRIBUTIONS environment variable), with credentials from the
// standard AWS environment. It is nil when nothing is configured, and quotas
// are then not enforced.
func Default() *Meter {
	defaultOnce.Do(func() {
		var ids []string
		for _, id := range strings.Split(config.Get().GetString("cloudfront.distributions"), ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			log.Warningf("No CloudFront distributions configured; monthly quotas are not enforced")
			return
		}
		cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion("us-east-1"))
		if err != nil {
			log.Errorf("Failed to load AWS config; monthly quotas are not enforced: %s", err.Error())
			return
		}
		defaultMeter = &Meter{api: cloudwatch.NewFromConfig(cfg), distributions: ids, now: time.Now}
	})
	return defaultMeter
}
