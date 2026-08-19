package docsync

import "testing"

func TestDocsOverlap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		slugA  string
		pathA  string
		slugB  string
		pathB  string
		expect bool
	}{
		{
			name:   "same slug",
			slugA:  "foo-bar",
			pathA:  "docs/Foo/Bar",
			slugB:  "foo-bar",
			pathB:  "docs/Other",
			expect: true,
		},
		{
			name:   "equivalent paths with spacing",
			slugA:  "create-audience-group",
			pathA:  "docs/Affinity Service/AudienceGroups/Create Audience Group",
			slugB:  "createaudiencegroup",
			pathB:  "docs/Affinity Service/AudienceGroups/CreateAudienceGroup",
			expect: true,
		},
		{
			name:   "different docs",
			slugA:  "alpha",
			pathA:  "docs/Alpha",
			slugB:  "beta",
			pathB:  "docs/Beta",
			expect: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := docsOverlap(tc.slugA, tc.pathA, tc.slugB, tc.pathB); got != tc.expect {
				t.Fatalf("docsOverlap() = %v, want %v", got, tc.expect)
			}
		})
	}
}
