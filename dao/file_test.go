package dao

import (
	"reflect"
	"testing"
	"time"

	"service/config"
	"service/log"
	"service/model"
)

func testSources(hidden map[string]bool, weight map[string]int, names ...string) []*model.Source {
	sources := make([]*model.Source, 0, len(names))
	for _, name := range names {
		w, ok := weight[name]
		if !ok {
			w = 1
		}
		site := &model.Site{Name: name, Hidden: hidden[name], Weight: w}
		sources = append(sources, &model.Source{Site: site, SiteName: name, URL: "https://" + name + "/f.zip"})
	}
	return sources
}

func siteNames(sources []*model.Source) []string {
	names := make([]string, 0, len(sources))
	for _, src := range sources {
		names = append(names, src.SiteName)
	}
	return names
}

func TestExcludeLagging(t *testing.T) {
	old := time.Date(2026, 9, 17, 4, 46, 47, 0, time.UTC)
	cur := old.Add(6 * time.Hour)
	tests := []struct {
		name     string
		hidden   map[string]bool
		weight   map[string]int
		modified map[string]time.Time
		want     []string
		lagging  []string
	}{
		{
			name: "no modified data keeps all",
			want: []string{"a", "b", "c"},
		},
		{
			name:     "all in sync keeps all",
			modified: map[string]time.Time{"a": cur, "b": cur, "c": cur},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "lagging site is excluded",
			modified: map[string]time.Time{"a": cur, "b": old, "c": cur},
			want:     []string{"a", "c"},
			lagging:  []string{"b"},
		},
		{
			name:     "unknown modified is kept",
			modified: map[string]time.Time{"a": cur, "b": old},
			want:     []string{"a", "c"},
			lagging:  []string{"b"},
		},
		{
			name:     "hidden site does not set newest",
			hidden:   map[string]bool{"c": true},
			modified: map[string]time.Time{"a": old, "b": old, "c": cur},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "only one site synced keeps just that site",
			modified: map[string]time.Time{"a": old, "b": cur, "c": old},
			want:     []string{"b"},
			lagging:  []string{"a", "c"},
		},
		{
			name:     "zero weight site does not set newest",
			weight:   map[string]int{"c": 0},
			modified: map[string]time.Time{"a": old, "b": old, "c": cur},
			want:     []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, lagging := excludeLagging(testSources(tt.hidden, tt.weight, "a", "b", "c"), tt.modified)
			if got := siteNames(kept); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("kept = %v, want %v", got, tt.want)
			}
			if got, want := siteNames(lagging), tt.lagging; len(got)+len(want) > 0 && !reflect.DeepEqual(got, want) {
				t.Errorf("lagging = %v, want %v", got, want)
			}
		})
	}
}

type fixedMeter struct {
	bytes int64
	ok    bool
}

func (m fixedMeter) MonthToDate() (int64, bool) { return m.bytes, m.ok }

func TestOverQuota(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	quota := &model.Site{Name: "OSL", MonthlyQuota: 900}
	tests := []struct {
		name  string
		site  *model.Site
		meter usageMeter
		want  bool
	}{
		{"over", quota, fixedMeter{901, true}, true},
		{"at the quota", quota, fixedMeter{900, true}, false},
		{"under", quota, fixedMeter{10, true}, false},
		{"no reading", quota, fixedMeter{5000, false}, false},
		{"no meter", quota, nil, false},
		{"no quota", &model.Site{Name: "kcwu"}, fixedMeter{5000, true}, false},
	}
	for _, tt := range tests {
		if got := overQuota(tt.site, tt.meter); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
