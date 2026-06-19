package render

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/tone-labs/ghx/internal/model"
)

// checkBucketOrder lists buckets worst-first, so failures lead the rollup.
var checkBucketOrder = []string{"fail", "cancel", "pending", "pass", "skipping"}

// ChecksView renders the CI status-check rollup: colored bucket counts
// (failures first) then any failing-check detail with workflow links.
func ChecksView(w io.Writer, pr int, ck *model.Checks, color ColorMode) {
	s := newStyles(w, color)
	fmt.Fprintln(w, s.bold.Render("checks")+s.faint.Render(fmt.Sprintf("  PR #%d  ·  %s", pr, plural(ck.Total, "check"))))
	if ck.Total == 0 {
		fmt.Fprintln(w, s.faint.Render("  (no checks)"))
		return
	}

	fmt.Fprintln(w)
	for _, b := range checkBucketOrder {
		n := ck.Counts[b]
		if n == 0 {
			continue
		}
		st := bucketStyle(s, b)
		fmt.Fprintf(w, "  %s %s\n", st.Render(bucketGlyph(b)), st.Render(fmt.Sprintf("%d %s", n, b)))
	}

	if len(ck.Failing) > 0 {
		fmt.Fprintln(w, "\n"+s.faint.Render("FAILING"))
		for _, c := range ck.Failing {
			head := "  " + s.red.Render(bucketGlyph(c.Bucket)) + " " + s.bold.Render(c.Name)
			if c.Workflow != "" {
				head += "  " + s.faint.Render(c.Workflow)
			}
			fmt.Fprintln(w, head)
			if c.Link != "" {
				fmt.Fprintln(w, "    "+s.faint.Render(c.Link))
			}
		}
	}
}

func bucketGlyph(bucket string) string {
	switch bucket {
	case "pass":
		return "✓"
	case "fail", "cancel":
		return "✗"
	case "pending":
		return "○"
	default: // skipping, unknown
		return "⊘"
	}
}

func bucketStyle(s styles, bucket string) lipgloss.Style {
	switch bucket {
	case "pass":
		return s.green
	case "fail", "cancel":
		return s.red
	case "pending":
		return s.yellow
	default:
		return s.faint
	}
}
