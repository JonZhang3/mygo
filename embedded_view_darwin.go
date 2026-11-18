//go:build darwin

package mygo

import (
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// EmbeddedView places an externally owned NSView in the same clipped,
// composited layer as a system WebView.
type EmbeddedView struct {
	host *Window
	view platform.WebView
}

// NewEmbeddedView creates a container for an external native view. The caller
// owns the guest and must close it before destroying the container.
func (w *Window) NewEmbeddedView(focus func()) (*EmbeddedView, uintptr, error) {
	var view *EmbeddedView
	var handle uintptr
	err := errNoContent
	onMain(func() {
		if w.conn == nil {
			return
		}
		if host, ok := w.conn.Surface.(interface {
			NewEmbeddedView(func()) (platform.WebView, uintptr)
		}); ok {
			native, pane := host.NewEmbeddedView(focus)
			view, handle, err = &EmbeddedView{host: w, view: native}, pane, nil
		}
	})
	return view, handle, err
}

func (v *EmbeddedView) SurfaceWebView(conn *surface.Conn) platform.WebView {
	if conn.Window != v.host {
		return nil
	}
	return v.view
}

// TabOut returns keyboard traversal from the guest to the host surface.
func (v *EmbeddedView) TabOut(back bool) {
	if native, ok := v.view.(interface{ EmbeddedTabOut(bool) }); ok {
		native.EmbeddedTabOut(back)
	}
}

// RestoreHostFocus sends subsequent keyboard input to MyGo.
func (v *EmbeddedView) RestoreHostFocus() {
	if native, ok := v.view.(interface{ EmbeddedRestoreHostFocus() }); ok {
		native.EmbeddedRestoreHostFocus()
	}
}

func (v *EmbeddedView) Destroy() {
	onMain(func() {
		if v.view != nil {
			v.view.Close()
			v.view = nil
		}
	})
}
