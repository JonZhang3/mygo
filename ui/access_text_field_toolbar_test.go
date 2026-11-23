package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A custom text field that does not describe its text itself is a leaf to
// the platform, so a toolbar drawn over it would be lost: it shows beside
// the field instead, with its buttons, while the field's own text stays out.
func TestAccessToolbarOverCustomTextFieldShowsBesideIt(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Column(c).Role(RoleGroup).Label("Editor pane").Children(func() {
			Box(c).Role(RoleTextField).Label("Source").Size(300, 200).Focusable().Children(func() {
				Text(c, "package main")
				Row(c).Absolute().Top(4).Right(4).Role(RoleToolbar).Label("Format").Children(func() {
					Button(c, "Bold")
				})
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access

	field := accessNode(t, tree, platform.RoleTextField, "Source")
	toolbar := accessNode(t, tree, platform.RoleToolbar, "Format")
	if toolbar.Parent != field.Parent {
		t.Errorf("the toolbar's parent is node %d, want the field's parent %d", toolbar.Parent, field.Parent)
	}
	accessNode(t, tree, platform.RoleButton, "Bold")
	for _, n := range tree.Nodes {
		if n.Label == "package main" {
			t.Error("the leaf text field exposes its text as a node of its own")
		}
	}
}
