package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/tone-labs/ghx/internal/model"
)

// ChecksView renders the CI status-check rollup: bucket counts then any
// failing-check detail with workflow links.
func ChecksView(w io.Writer, pr int, ck *model.Checks) {
	fmt.Fprintf(w, "Checks (PR #%d): %d total\n", pr, ck.Total)
	if ck.Total == 0 {
		fmt.Fprintln(w, "  (no checks)")
		return
	}

	buckets := make([]string, 0, len(ck.Counts))
	for b := range ck.Counts {
		buckets = append(buckets, b)
	}
	sort.Strings(buckets)
	for _, b := range buckets {
		fmt.Fprintf(w, "  %d %s\n", ck.Counts[b], upper(b))
	}

	if len(ck.Failing) > 0 {
		fmt.Fprintf(w, "\nFailing:\n")
		for _, c := range ck.Failing {
			fmt.Fprintf(w, "  %s (%s)\n    %s\n", c.Name, c.Bucket, c.Link)
		}
	}
}

func upper(s string) string {
	if s == "" {
		return s
	}
	out := []rune(s)
	for i, r := range out {
		if r >= 'a' && r <= 'z' {
			out[i] = r - 32
		}
	}
	return string(out)
}
