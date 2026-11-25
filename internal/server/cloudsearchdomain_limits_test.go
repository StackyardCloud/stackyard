package server

import (
	"net/url"
	"strings"
	"testing"
)

func csdNaiveDistance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

func TestCSDWithinDistanceMatchesFullLevenshtein(t *testing.T) {
	words := []string{"kitten", "sitten", "sittin", "kitxen", "ktiten", "abcdef", "kitten"}
	for _, a := range words {
		for _, b := range words {
			for k := 0; k <= 2; k++ {
				want := csdNaiveDistance([]rune(a), []rune(b)) <= k
				if got := csdWithinDistance([]rune(a), []rune(b), k); got != want {
					t.Fatalf("within(%q,%q,%d)=%v want %v", a, b, k, got, want)
				}
			}
		}
	}
}

func TestCSDExpressionNodeBudget(t *testing.T) {
	long := strings.TrimSuffix(strings.Repeat("1+", csdMaxExprNodes+1), "+")
	if _, err := csdExprParam(&csDomain{}, `{"e":"`+long+`"}`); err == nil {
		t.Fatal("expected a chain longer than the node budget to be rejected")
	}
	if _, err := csdExprParam(&csDomain{}, `{"e":"_score*2+1"}`); err != nil {
		t.Fatalf("small expression rejected: %v", err.msg)
	}
}

func TestCSDSearchRejectsTooManyTerms(t *testing.T) {
	st := newCloudSearchStore()
	p := url.Values{"q": {strings.TrimSpace(strings.Repeat("a ", csdMaxQueryTerms+1))}}
	if _, err := st.search("any", p); err == nil || !strings.Contains(err.msg, "terms") {
		t.Fatalf("want term-cap error, got %v", err)
	}
}
