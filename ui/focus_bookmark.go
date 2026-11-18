package ui

// FocusBookmark retains focus identity across frames without retaining an Element.
// It cannot move focus into another window, an unmounted control or an inert exit.
type FocusBookmark struct {
	owner *elementOwner
	id    uint64
}

func (c *Context) SaveFocus() FocusBookmark {
	rt := c.runtime()
	if rt == nil {
		return FocusBookmark{}
	}
	return FocusBookmark{rt.arena, rt.focused}
}

// Live reports whether the focused control was built and remains interactive in
// this pass. Call after the application content has been built.
func (b FocusBookmark) Live(c *Context) bool { return b.resolve(c) != nil }
func (b FocusBookmark) resolve(c *Context) *node {
	rt := c.runtime()
	if rt == nil || b.owner != rt.arena || b.id == 0 {
		return nil
	}
	for i := uint32(0); i < uint32(rt.c.used); i++ {
		e := rt.nodeAt(i)
		if e.id != b.id {
			continue
		}
		if e.inHidden() || e.disabled() {
			return nil
		}
		for p := e; p != nil; p = p.parent {
			if p.ignoreInput || p.flags&flagInert != 0 {
				return nil
			}
		}
		return e
	}
	return nil
}
func (b FocusBookmark) Restore(c *Context) bool {
	e := b.resolve(c)
	if e == nil {
		return false
	}
	e.Focus()
	return true
}
