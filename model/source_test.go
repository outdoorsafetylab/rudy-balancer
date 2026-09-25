package model

import (
	"net/http"
	"testing"
	"time"
)

func TestLastModified(t *testing.T) {
	release := time.Date(2025, 11, 12, 3, 51, 17, 0, time.UTC)
	upload := "Fri, 25 Sep 2026 03:51:38 GMT"
	tests := []struct {
		name  string
		mtime string
		lm    string
		want  time.Time
	}{
		{"plain mirror", "", "Wed, 12 Nov 2025 03:51:17 GMT", release},
		// S3 behind CloudFront: Last-Modified is the upload time.
		{"S3 with rclone mtime", "1762919477", upload, release},
		{"fraction cut to seconds", "1762919477.987654321", upload, release},
		{"bad mtime falls back", "yesterday", upload, time.Date(2026, 9, 25, 3, 51, 38, 0, time.UTC)},
		{"nothing", "", "", time.Time{}},
	}
	for _, tt := range tests {
		h := http.Header{}
		if tt.mtime != "" {
			h.Set("X-Amz-Meta-Mtime", tt.mtime)
		}
		if tt.lm != "" {
			h.Set("Last-Modified", tt.lm)
		}
		if got := lastModified(h); !got.Equal(tt.want) {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}
