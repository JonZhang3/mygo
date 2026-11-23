package ui_test

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A label its box cuts off reports it, and one that fits does not, so an app
// can show a tooltip with the whole text only while part of it is hidden.
func TestTruncatedTellsWhetherTheLastFrameCutTextShort(t *testing.T) {
	long := strings.Repeat("Hearth native workbench ", 8)
	var fits, cut, wraps bool
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(120).Children(func() {
			fits = ui.Text(c, "Hi").SingleLine().Truncated()
			cut = ui.Text(c, long).SingleLine().Truncated()
			wraps = ui.Text(c, long).Truncated()
		})
	}, 400, 400)
	tt.Frame()
	if fits {
		t.Error("a label that fits reports it was cut short")
	}
	if !cut {
		t.Error("a single line wider than its box does not report it was cut short")
	}
	if wraps {
		t.Error("text that wraps onto more lines reports it was cut short")
	}
}
