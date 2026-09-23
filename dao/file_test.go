package dao

import (
	"reflect"
	"testing"
	"time"

	"service/model"
)

func testSources(hidden map[string]bool, names ...string) []*model.Source {
	sources := make([]*model.Source, 0, len(names))
	for _, name := range names {
		site := &model.Site{Name: name, Hidden: hidden[name]}
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
		modified map[string]time.Time
		want     []string
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
		},
		{
			name:     "unknown modified is kept",
			modified: map[string]time.Time{"a": cur, "b": old},
			want:     []string{"a", "c"},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, _ := excludeLagging(testSources(tt.hidden, "a", "b", "c"), tt.modified)
			got := siteNames(kept)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
