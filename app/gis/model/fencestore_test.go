package model

import "testing"

func TestGeohashLookupKeys(t *testing.T) {
	exactMatches, likePatterns := geohashLookupKeys([]string{"wx4g0"})

	wantExact := map[string]bool{
		"wx4":   false,
		"wx4g":  false,
		"wx4g0": false,
	}
	for _, match := range exactMatches {
		if _, ok := wantExact[match]; ok {
			wantExact[match] = true
		}
	}
	for match, found := range wantExact {
		if !found {
			t.Fatalf("missing exact match %q in %v", match, exactMatches)
		}
	}

	wantPattern := "wx4g0%"
	for _, pattern := range likePatterns {
		if pattern == wantPattern {
			return
		}
	}
	t.Fatalf("missing like pattern %q in %v", wantPattern, likePatterns)
}
