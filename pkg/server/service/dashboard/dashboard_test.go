package dashboard

import (
	"testing"
)

func TestBuildEmbeddingLink(t *testing.T) {
	testCases := []struct {
		desc     string
		scheme   string
		host     string
		boardurl string
		useRange bool
		from, to int64
		panel    uint
		wanted   string
	}{
		{
			desc:     "1",
			scheme:   "http",
			host:     "127.0.0.1:3000",
			boardurl: "/d/xxx",
			useRange: true,
			from:     123,
			to:       111,
			panel:    1,
			wanted:   "http://127.0.0.1:3000/d/xxx?from=123&to=111&viewPanel=1",
		},
		{
			desc:     "2",
			scheme:   "http",
			host:     "127.0.0.1:3000",
			boardurl: "/d/xxx",
			useRange: false,
			from:     123,
			to:       111,
			panel:    1,
			wanted:   "http://127.0.0.1:3000/d/xxx?viewPanel=1",
		},
	}
	for _, tc := range testCases {
		gotLink := BuildEmbeddingLink(tc.scheme, tc.host, tc.boardurl, tc.useRange, tc.from, tc.to, tc.panel)
		if gotLink != tc.wanted {
			t.Fatalf("tc %v, got %v, wanted: %v", tc.desc, gotLink, tc.wanted)
		}
	}
}
