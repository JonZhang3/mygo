package text

import (
	"reflect"
	"testing"
)

// The native renderer must keep each selected font as one literal family. A
// comma or escaped quote must not select a different installed fallback font.
func TestHearthLiteralFontSelection(t *testing.T) {
	stack := `"Font, With Comma", "Quote\"Name", "Slash\\Name", "中文字体", system-ui`
	want := []string{"Font, With Comma", `Quote"Name`, `Slash\Name`, "中文字体", "system-ui"}
	if got := familyList(stack); !reflect.DeepEqual(got, want) {
		t.Fatalf("font selection changed: got %q, want %q", got, want)
	}
}
