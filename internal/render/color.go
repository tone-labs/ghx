package render

import (
	"os"

	"golang.org/x/term"
)

// useColor gates ANSI escapes. Off by default so piped/redirected output stays
// clean; cli flips it on for an interactive terminal.
var useColor bool

// AutoColor enables color when w is a terminal and NO_COLOR is unset.
func AutoColor(fd uintptr) {
	useColor = os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(fd))
}

func dim(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}
