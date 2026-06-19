package render

import "testing"

// TestScoutCanary intentionally fails to give /scout a red check to survey.
// This lives only on the throwaway collin/scout-test-drive branch — delete it.
func TestScoutCanary(t *testing.T) {
	t.Fatal("intentional failure: /scout test-drive canary — do not merge")
}
