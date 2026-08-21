package docsync

import "testing"

func TestSuggestionMatchesAcceptFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		row    string
		filter string
		want   bool
	}{
		{row: "exact", filter: "high", want: true},
		{row: "high", filter: "high", want: true},
		{row: "medium", filter: "high", want: false},
		{row: "low", filter: "high", want: false},
		{row: "exact", filter: "all", want: true},
		{row: "medium", filter: "all", want: true},
		{row: "low", filter: "all", want: true},
		{row: "medium", filter: "medium", want: true},
		{row: "high", filter: "medium", want: false},
		{row: "exact", filter: "", want: true},
	}
	for _, tc := range cases {
		if got := suggestionMatchesAcceptFilter(tc.row, tc.filter); got != tc.want {
			t.Errorf("suggestionMatchesAcceptFilter(%q, %q) = %v, want %v", tc.row, tc.filter, got, tc.want)
		}
	}
}
