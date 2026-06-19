package render

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/tone-labs/ghx/internal/model"
)

// ColorMode controls ANSI color in the human view.
type ColorMode int

const (
	ColorAuto   ColorMode = iota // color when the writer is a TTY (honors NO_COLOR)
	ColorAlways                  // force color even when piped (overrides NO_COLOR)
	ColorNever                   // never emit color
)

// Options controls the human (non-JSON) comment view.
type Options struct {
	Width            int       // explicit wrap width; <= 0 = detect the terminal (full width)
	BodyLines        int       // max wrapped lines per body; 0 = unlimited
	ShowConversation bool      // expand PR-level conversation (default: collapsed summary)
	Color            ColorMode // when to colorize (default: ColorAuto)
}

// styles bundles the lipgloss styles for one render, bound to the output writer
// (so color auto-detects per writer: a TTY gets color, a pipe or test buffer
// does not).
type styles struct {
	bold, faint, file, dir, idx, author lipgloss.Style
	green, yellow, red, purple          lipgloss.Style
}

func newStyles(w io.Writer, color ColorMode) styles {
	r := lipgloss.NewRenderer(w)
	// auto: lipgloss already detects TTY + honors NO_COLOR. Override only for the
	// explicit modes, where always wins even over NO_COLOR.
	switch color {
	case ColorAlways:
		r.SetColorProfile(termenv.TrueColor)
	case ColorNever:
		r.SetColorProfile(termenv.Ascii)
	}
	return styles{
		bold:   r.NewStyle().Bold(true),
		faint:  r.NewStyle().Faint(true),
		file:   r.NewStyle().Bold(true).Foreground(lipgloss.Color("6")), // cyan
		dir:    r.NewStyle().Faint(true),
		idx:    r.NewStyle().Bold(true).Foreground(lipgloss.Color("3")), // yellow
		author: r.NewStyle().Foreground(lipgloss.Color("4")),            // blue
		green:  r.NewStyle().Foreground(lipgloss.Color("2")),
		yellow: r.NewStyle().Foreground(lipgloss.Color("3")),
		red:    r.NewStyle().Foreground(lipgloss.Color("1")),
		purple: r.NewStyle().Foreground(lipgloss.Color("5")), // merged (GitHub-style)
	}
}

// Comments renders the PR review state, assuming pr is already filtered.
// rightGutter is a small margin kept clear of the terminal's right edge. It
// avoids last-column auto-wrap and absorbs width-estimation slop for
// ambiguous-width glyphs (e.g. "↳"), which some terminals render two columns
// wide while runewidth counts them as one.
const rightGutter = 2

func Comments(w io.Writer, pr *model.PR, opts Options) {
	s := newStyles(w, opts.Color)
	width := opts.Width
	if width <= 0 {
		width = contentWidth(w) - rightGutter
	}
	if width < 20 {
		width = 20
	}

	// Header + BLUF status line.
	fmt.Fprintln(w, s.bold.Render(fmt.Sprintf("#%d  %s", pr.Number, pr.Title)))
	if pr.URL != "" {
		fmt.Fprintln(w, s.faint.Render(pr.URL))
	}
	fmt.Fprintln(w, statusLine(s, pr))

	reviews := latestPerAuthor(pr.Reviews)
	if len(reviews) > 0 {
		fmt.Fprintln(w, "\n"+s.faint.Render("REVIEWS"))
		for _, r := range reviews {
			g := reviewGlyph(s, r.State)
			fmt.Fprintf(w, "  %s %s%s  %s\n", g, s.author.Render(r.Author), botTag(s, r.IsBot),
				s.faint.Render(strings.ToLower(r.State)))
			if body := flatten(r.Body); body != "" {
				writeBody(w, s, 6, "", body, width, opts.BodyLines)
			}
		}
	}

	if len(pr.Threads) > 0 {
		hdr := fmt.Sprintf("THREADS · %d", len(pr.Threads))
		fmt.Fprintln(w, "\n"+s.faint.Render(hdr))
		for i, t := range pr.Threads {
			renderThread(w, s, i+1, t, width, opts.BodyLines)
		}
	} else {
		fmt.Fprintln(w, "\n"+s.faint.Render("No threads."))
	}

	renderConversation(w, s, pr, width, opts)

	if len(pr.Threads) > 1 {
		fmt.Fprintln(w, "\n"+s.faint.Render(fmt.Sprintf("drill in:  ghx comments %d --thread <n>", pr.Number)))
	}
}

func statusLine(s styles, pr *model.PR) string {
	unresolved := 0
	for _, t := range pr.Threads {
		if !t.IsResolved {
			unresolved++
		}
	}
	parts := []string{decisionChip(s, pr.ReviewDecision)}
	parts = append(parts, s.faint.Render(fmt.Sprintf("%d unresolved", unresolved)))
	parts = append(parts, s.faint.Render(plural(len(latestPerAuthor(pr.Reviews)), "review")))
	state := strings.ToLower(pr.State)
	if pr.State == "MERGED" {
		parts = append(parts, s.purple.Render(state)) // GitHub-style merged
	} else {
		parts = append(parts, s.faint.Render(state))
	}
	if pr.IsDraft {
		parts = append(parts, s.faint.Render("draft"))
	}
	return strings.Join(parts, s.faint.Render(" · "))
}

func decisionChip(s styles, d string) string {
	switch d {
	case "APPROVED":
		return s.green.Render("APPROVED")
	case "CHANGES_REQUESTED":
		return s.red.Render("CHANGES REQUESTED")
	case "REVIEW_REQUIRED":
		return s.yellow.Render("REVIEW REQUIRED")
	default:
		return s.faint.Render("no decision")
	}
}

func reviewGlyph(s styles, state string) string {
	switch state {
	case "APPROVED":
		return s.green.Render("✓")
	case "CHANGES_REQUESTED":
		return s.red.Render("✗")
	case "DISMISSED":
		return s.faint.Render("⊘")
	default: // COMMENTED, PENDING
		return s.faint.Render("○")
	}
}

func renderThread(w io.Writer, s styles, n int, t model.Thread, width, bodyLines int) {
	base, dir := splitPath(t.Path)
	loc := s.file.Render(base)
	if t.Line > 0 {
		loc += s.file.Render(fmt.Sprintf(":%d", t.Line))
	}
	head := fmt.Sprintf("  %s %s", s.idx.Render(fmt.Sprintf("[%d]", n)), loc)
	if dir != "" {
		head += "  " + s.dir.Render(elideDir(dir))
	}
	head += threadBadge(s, t)
	fmt.Fprintln(w, "\n"+head)
	for i, c := range t.Comments {
		marker := ""
		if i > 0 {
			marker = "↳ "
		}
		writeBody(w, s, 6, marker+c.Author+botTagPlain(c.IsBot), c.Body, width, bodyLines)
	}
}

// writeBody prints "<indent><label>  <wrapped body>" with continuation lines
// aligned under the body. label is the (plain) author/marker prefix; pass "" to
// indent the body with no label.
func writeBody(w io.Writer, s styles, indent int, label, body string, width, maxLines int) {
	pad := strings.Repeat(" ", indent)
	var headPlain, headStyled string
	if label != "" {
		headPlain = pad + label + "  "
		headStyled = pad + s.author.Render(label) + "  "
	} else {
		headPlain = pad
		headStyled = pad
	}
	// head is the label's display width; the body wraps in the space the label
	// leaves. When the label is so wide (or the terminal so narrow) that little
	// room remains, pull the continuation indent back in so re-indented body
	// lines never spill past the right edge. The body on the first line is then
	// clamped to whatever the label actually leaves it.
	const minBody = 10
	head := cellWidth(headPlain)
	cont := head
	bodyWidth := width - cont
	if bodyWidth < minBody {
		bodyWidth = min(minBody, width)
		cont = width - bodyWidth
	}
	lines := wrapBody(body, bodyWidth, maxLines)
	if len(lines) == 0 {
		return
	}
	fmt.Fprintln(w, headStyled+clampWidth(lines[0], width-head))
	for _, ln := range lines[1:] {
		fmt.Fprintln(w, strings.Repeat(" ", cont)+ln)
	}
}

func renderConversation(w io.Writer, s styles, pr *model.PR, width int, opts Options) {
	if len(pr.Conversation) == 0 {
		return
	}
	if !opts.ShowConversation {
		bots := 0
		for _, c := range pr.Conversation {
			if c.IsBot {
				bots++
			}
		}
		msg := fmt.Sprintf("CONVERSATION · %s", plural(len(pr.Conversation), "comment"))
		if bots > 0 {
			msg += fmt.Sprintf(" (%d from bots)", bots)
		}
		fmt.Fprintln(w, "\n"+s.faint.Render(msg+"   — --conversation to show"))
		return
	}
	fmt.Fprintln(w, "\n"+s.faint.Render("CONVERSATION"))
	for _, c := range pr.Conversation {
		writeBody(w, s, 2, c.Author+botTagPlain(c.IsBot), c.Body, width, opts.BodyLines)
	}
}

func splitPath(p string) (base, dir string) {
	if p == "" {
		return "(general)", ""
	}
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return p, ""
	}
	return p[i+1:], p[:i]
}

// elideDir keeps a long directory legible: the last two segments prefixed with
// an ellipsis. The full path is always available in --json.
func elideDir(dir string) string {
	parts := strings.Split(dir, "/")
	if len(parts) > 2 {
		return "…/" + strings.Join(parts[len(parts)-2:], "/")
	}
	return dir
}

func threadBadge(s styles, t model.Thread) string {
	var tags []string
	if t.IsResolved {
		tags = append(tags, "resolved")
	}
	if t.IsOutdated {
		tags = append(tags, "outdated")
	}
	if len(tags) == 0 {
		return ""
	}
	return "  " + s.faint.Render("("+strings.Join(tags, ", ")+")")
}

func botTag(s styles, b bool) string {
	if b {
		return " " + s.faint.Render("(bot)")
	}
	return ""
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func botTagPlain(b bool) string {
	if b {
		return " (bot)"
	}
	return ""
}

// latestPerAuthor keeps each author's most recently submitted review.
func latestPerAuthor(reviews []model.Review) []model.Review {
	latest := map[string]model.Review{}
	for _, r := range reviews {
		if cur, ok := latest[r.Author]; !ok || r.SubmittedAt > cur.SubmittedAt {
			latest[r.Author] = r
		}
	}
	out := make([]model.Review, 0, len(latest))
	for _, r := range latest {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubmittedAt < out[j].SubmittedAt })
	return out
}

// contentWidth returns the wrap width for the output: the full terminal width
// when writing to a TTY (measured once, at print time — like `gh pr checks`),
// else a fixed default for pipes and tests. Override via Options.Width.
func contentWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if wd, _, err := term.GetSize(int(f.Fd())); err == nil && wd > 0 {
			return wd
		}
	}
	return 100
}
