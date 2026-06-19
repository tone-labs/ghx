package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/tone-labs/ghx/internal/render"
)

func TestResolveExit(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantShow bool
	}{
		{"nil is success", nil, 0, false},
		{"cmdError is runtime failure", fail(errors.New("boom")), 1, true},
		{"wrapped cmdError still 1", fmt.Errorf("ctx: %w", fail(errors.New("boom"))), 1, true},
		{"plain error is usage", errors.New("bad flag"), 2, true},
		{"statusError carries code, no output", statusExit(1), 1, false},
		{"statusError arbitrary code", statusExit(3), 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, show := resolveExit(tt.err)
			if code != tt.wantCode || show != tt.wantShow {
				t.Errorf("resolveExit = (%d, %v), want (%d, %v)", code, show, tt.wantCode, tt.wantShow)
			}
		})
	}
}

func TestColorFlag(t *testing.T) {
	var def colorFlag // zero value
	if def.String() != "auto" {
		t.Errorf("default String() = %q, want auto", def.String())
	}
	valid := map[string]render.ColorMode{
		"auto": render.ColorAuto, "always": render.ColorAlways, "never": render.ColorNever,
	}
	for in, want := range valid {
		var c colorFlag
		if err := c.Set(in); err != nil {
			t.Errorf("Set(%q) error: %v", in, err)
		}
		if c.mode != want {
			t.Errorf("Set(%q) mode = %v, want %v", in, c.mode, want)
		}
	}
	var bad colorFlag
	if err := bad.Set("rainbow"); err == nil {
		t.Error("Set(rainbow) should error")
	}
}

// sub builds a fresh root and returns it plus its named subcommand.
func sub(t *testing.T, name string) (root, cmd *cobra.Command) {
	t.Helper()
	root = newRootCmd()
	for _, c := range root.Commands() {
		if c.Name() == name {
			return root, c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil, nil
}

// TestInterspersedParsing is the regression guard for the bug this branch
// fixes: a flag value must never be mistaken for the positional PR. With pflag,
// `comments --width 100 42` consumes 100 as --width's value, leaving 42 as the
// sole positional — where the old hand-rolled splitPR grabbed 100 as the PR.
// RunE is stubbed so parsing is exercised without touching gh/network.
func TestInterspersedParsing(t *testing.T) {
	cases := []struct {
		args   []string
		wantPR string
	}{
		{[]string{"comments", "--width", "100", "42"}, "42"},
		{[]string{"comments", "42", "--lines", "4"}, "42"},
		{[]string{"comments", "--json"}, ""},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			root, cmd := sub(t, "comments")
			var got string
			ran := false
			cmd.RunE = func(_ *cobra.Command, args []string) error {
				ran, got = true, prArg(args)
				return nil
			}
			root.SetArgs(tc.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !ran {
				t.Fatal("RunE did not run")
			}
			if got != tc.wantPR {
				t.Errorf("PR arg = %q, want %q", got, tc.wantPR)
			}
		})
	}
}

// TestUsageErrors confirms flag/arg errors map to exit 2 (not the runtime-error
// exit 1). These all fail during parse/validation, before RunE, so no network.
func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{"comments", "1", "2"}, // too many positionals (MaximumNArgs(1))
		{"comments", "--nope"}, // unknown flag
		{"version", "extra"},   // NoArgs violated
		{"bogus"},              // unknown command
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := newRootCmd()
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			err := root.Execute()
			if err == nil {
				t.Fatal("expected an error")
			}
			if code, _ := resolveExit(err); code != 2 {
				t.Errorf("exit code = %d, want 2 (err: %v)", code, err)
			}
		})
	}
}

func TestFlagWiring(t *testing.T) {
	_, cmd := sub(t, "comments")
	if f := cmd.Flags().Lookup("lines"); f == nil || f.DefValue != "2" {
		t.Errorf("--lines default = %+v, want 2", f)
	}
	if f := cmd.Flags().Lookup("width"); f == nil || f.DefValue != "0" {
		t.Errorf("--width default = %+v, want 0", f)
	}
	if cmd.Flags().ShorthandLookup("R") == nil {
		t.Error("-R shorthand for --repo is missing")
	}
}

func TestVersionSubcommand(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetArgs([]string{"version"})
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got := out.String(); got != "ghx dev\n" {
		t.Errorf("version output = %q, want %q", got, "ghx dev\n")
	}
}
