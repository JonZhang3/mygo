package terminal

import (
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/ui"
)

func TestTerminalLinksPreserveSoftWrappedTargets(t *testing.T) {
	if os.Getenv("MYGO_GHOSTTY_VT") == "" {
		t.Skip("set MYGO_GHOSTTY_VT to the bundled library")
	}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	const width, rows = 32, 3
	const url = "http://127.0.0.1:48765/index.html?native-final=terminal"
	const path = "./workspace/changes/中/terminal-fresh.md:42:3"
	for _, tc := range []struct {
		name, output, target string
		isPath               bool
	}{
		{"URL", url, url, false},
		{"path and source location", path, path, true},
		{"wide glyph at wrap boundary", strings.Repeat(" ", width-len("./workspace/changes/")-1) + path, path, true},
		{"hard newline", "https://example.com/a\r\ncontinuation", "https://example.com/a", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grid, err := vt.NewTerminal(width, rows, vt.Effects{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(grid.Free)
			v := &view{t: &Terminal{term: grid, opts: Options{OpenLink: func(string) bool { return true }}}, cols: width, rows: rows}
			grid.Write([]byte(tc.output + "\r\n"))
			grid.ScrollToTop()
			column := len(tc.output) - len(tc.target)
			if tc.name == "hard newline" {
				column = 0
			}
			check := func(col, row int, want string, isPath bool) {
				t.Helper()
				got, pathLink, _ := v.linkAt(col, row)
				if got != want || pathLink != isPath {
					t.Fatalf("link at (%d, %d) = %q (path %v), want %q (path %v)", col, row, got, pathLink, want, isPath)
				}
			}
			check(column, 0, tc.target, tc.isPath)
			if tc.name == "hard newline" {
				check(0, 1, "", false)
				return
			}
			check(0, 1, tc.target, tc.isPath)
			grid.Write([]byte("\r\nnext\r\n"))
			grid.ScrollToRow(1)
			check(0, 0, tc.target, tc.isPath)
		})
	}
}

func TestTerminalLinksOnListingNamesAndPrimaryClicks(t *testing.T) {
	if os.Getenv("MYGO_GHOSTTY_VT") == "" {
		t.Skip("set MYGO_GHOSTTY_VT to the bundled library")
	}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	grid, err := vt.NewTerminal(100, 6, vt.Effects{})
	if err != nil {
		t.Fatal(err)
	}
	defer grid.Free()
	v := &view{t: &Terminal{term: grid, opts: Options{
		OnLinkClick:   func(string, ui.InputEvent) {},
		LinkAvailable: func(link string) bool { return link != "MISSING" },
	}}, cols: 100, rows: 6, cellW: 8, cellH: 16, scale: 1}
	const header = "-rw-r--r--@ 1 user staff 21 Nov 17 13:00 "
	grid.Write([]byte(header + "LICENSE\r\n" + header + "some file.md\r\n" +
		"See LICENSE and MISSING\r\n" + "some\\ file.md\r\n" + "./ ../\r\n" +
		"\x1b]8;;https://example.test/path\x07web\x1b]8;;\x07"))
	for _, tc := range []struct {
		col, row int
		want     string
	}{
		{len(header), 0, "LICENSE"},
		{len(header) + 6, 1, "some file.md"},
		{5, 1, ""},
		{4, 2, "LICENSE"},
		{6, 3, "some file.md"},
		{0, 4, "./"},
		{4, 4, "../"},
		{0, 5, "https://example.test/path"},
	} {
		if got, _, _ := v.linkAt(tc.col, tc.row); got != tc.want {
			t.Fatalf("link at (%d, %d) = %q, want %q", tc.col, tc.row, got, tc.want)
		}
	}
	if got, _, _ := v.linkUnder(0, 4*8+2, 2*16+2); got != "LICENSE" {
		t.Fatalf("ordinary hover on LICENSE returned %q", got)
	}
	if got, _, _ := v.linkUnder(0, 16*8+2, 2*16+2); got != "" {
		t.Fatalf("unavailable file got hover decoration: %q", got)
	}
	press := ui.InputEvent{Kind: ui.InputPointerDown, X: 4*8 + 2, Y: 2*16 + 2}
	release := press
	release.Kind = ui.InputPointerUp
	v.trackLinkClick(press)
	v.trackLinkClick(release)
	if v.activatedLink != "LICENSE" {
		t.Fatalf("primary activation returned %q", v.activatedLink)
	}
	v.activatedLink = ""
	v.trackLinkClick(press)
	move := press
	move.Kind, move.X = ui.InputPointerMove, press.X+18
	v.trackLinkClick(move)
	v.trackLinkClick(release)
	if v.activatedLink != "" {
		t.Fatal("dragging a selection activated the link")
	}
	grid.Write([]byte("\x1b[?1000h"))
	if got, _, span := v.linkUnder(0, 2, 5*16+2); got != "https://example.test/path" || !span.on() {
		t.Fatalf("TUI mouse tracking hid the URL's unmodified hover underline: %q", got)
	}
	for _, tc := range []struct {
		col, row int
		want     string
	}{
		{4, 2, "LICENSE"},
		{4, 4, "../"},
		{0, 5, "https://example.test/path"},
		{5, 1, ""},
		{16, 2, ""},
	} {
		v.activatedLink = ""
		press.X, press.Y = float32(tc.col*8+2), float32(tc.row*16+2)
		release.X, release.Y = press.X, press.Y
		down, up := v.trackLinkClick(press), v.trackLinkClick(release)
		if v.activatedLink != tc.want || down != (tc.want != "") || up != down {
			t.Fatalf("TUI link click at (%d, %d) activated %q, owned %v/%v; want %q", tc.col, tc.row, v.activatedLink, down, up, tc.want)
		}
	}
}
