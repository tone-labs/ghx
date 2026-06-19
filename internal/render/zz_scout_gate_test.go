package render

import "testing"

// TestScoutGateCanary passes — it gives a throwaway PR a clean check while we
// test-drive /scout's review-gate reporting. Delete with the branch.
func TestScoutGateCanary(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("arithmetic is broken")
	}
}
