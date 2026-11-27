package model

import "testing"

func TestSiteURLs(t *testing.T) {
	s := &Site{RedirectScheme: "https", ProxyScheme: "http", Host: "map.example", Prefix: "/rudy"}
	tests := []struct {
		uri, redirect, proxy string
	}{
		// what every file in the mirror list looks like today: unchanged
		{"MOI_OSM_Taiwan_TOPO_Rudy.pmtiles", "https://map.example/rudy/MOI_OSM_Taiwan_TOPO_Rudy.pmtiles", "http://map.example/rudy/MOI_OSM_Taiwan_TOPO_Rudy.pmtiles"},
		{"/taiwan_topo.html", "https://map.example/rudy/taiwan_topo.html", "http://map.example/rudy/taiwan_topo.html"},
		{"rudymap_pmtiles_style/style.json", "https://map.example/rudy/rudymap_pmtiles_style/style.json", "http://map.example/rudy/rudymap_pmtiles_style/style.json"},
		{"", "https://map.example/rudy", "http://map.example/rudy"},
		// a space is escaped, slashes are kept (#57)
		{"rudymap_pmtiles_style/glyphs/Noto Sans Regular/0-255.pbf",
			"https://map.example/rudy/rudymap_pmtiles_style/glyphs/Noto%20Sans%20Regular/0-255.pbf",
			"http://map.example/rudy/rudymap_pmtiles_style/glyphs/Noto%20Sans%20Regular/0-255.pbf"},
	}
	for _, tt := range tests {
		if got := s.GetRedirectURL(tt.uri); got != tt.redirect {
			t.Errorf("GetRedirectURL(%q) = %q, want %q", tt.uri, got, tt.redirect)
		}
		if got := s.GetProxyURL(tt.uri); got != tt.proxy {
			t.Errorf("GetProxyURL(%q) = %q, want %q", tt.uri, got, tt.proxy)
		}
	}
	// a site with no prefix
	bare := &Site{RedirectScheme: "https", Host: "moi.example"}
	if got, want := bare.GetRedirectURL("a b.zip"), "https://moi.example/a%20b.zip"; got != want {
		t.Errorf("no prefix: %q, want %q", got, want)
	}
}
