package utils

import "testing"

func TestOptionalHTTPURLValidators(t *testing.T) {
	t.Parallel()

	type caseRow struct {
		raw  string
		want bool
	}

	run := func(name string, fn func(string) bool, cases []caseRow) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, tc := range cases {
				if got := fn(tc.raw); got != tc.want {
					t.Fatalf("%s(%q) = %v, want %v", name, tc.raw, got, tc.want)
				}
			}
		})
	}

	run("IsOptionalHTTPURL", IsOptionalHTTPURL, []caseRow{
		{"", true},
		{"   ", true},
		{"https://example.com/portfolio", true},
		{"http://example.com", true},
		{"not-a-url", false},
		{"ftp://example.com", false},
		{"javascript:alert(1)", false},
	})

	run("IsOptionalLinkedInURL", IsOptionalLinkedInURL, []caseRow{
		{"", true},
		{"https://www.linkedin.com/in/example", true},
		{"https://linkedin.com/in/example", true},
		{"https://vn.linkedin.com/in/example", true},
		{"https://github.com/example", false},
		{"https://example.com/in/example", false},
		{"not-a-url", false},
	})

	run("IsOptionalGitHubURL", IsOptionalGitHubURL, []caseRow{
		{"", true},
		{"https://github.com/example", true},
		{"https://www.github.com/example/repo", true},
		{"https://gist.github.com/example/abc", true},
		{"https://linkedin.com/in/example", false},
		{"https://example.com/example", false},
		{"not-a-url", false},
	})
}
