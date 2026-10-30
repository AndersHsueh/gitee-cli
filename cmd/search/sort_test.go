package search

import "testing"

func TestCanonicalRepoSearchSort(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"stars":         "stars_count",
		"forks":         "forks_count",
		"updated":       "last_push_at",
		"stars_count":   "stars_count",
		"last_push_at":  "last_push_at",
		"watches_count": "watches_count",
	}
	for in, want := range cases {
		got, err := canonicalRepoSearchSort(in)
		if err != nil {
			t.Fatalf("sort %q: %v", in, err)
		}
		if got != want {
			t.Fatalf("sort %q = %q, want %q", in, got, want)
		}
	}
	if _, err := canonicalRepoSearchSort("nope"); err == nil {
		t.Fatal("expected invalid sort to fail locally")
	}
}

func TestCanonicalRepoSearchLanguage(t *testing.T) {
	if got := canonicalRepoSearchLanguage("go"); got != "Go" {
		t.Fatalf("go = %q", got)
	}
	if got := canonicalRepoSearchLanguage("Go"); got != "Go" {
		t.Fatalf("Go = %q", got)
	}
	if got := canonicalRepoSearchLanguage("JavaScript"); got != "JavaScript" {
		t.Fatalf("JavaScript = %q", got)
	}
	if got := canonicalRepoSearchLanguage("COBOL"); got != "COBOL" {
		t.Fatalf("unknown language should pass through, got %q", got)
	}
}
