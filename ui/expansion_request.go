package ui

// ExpansionRequest consumes an assistive-technology request to open or close a
// custom tree row and declares that it supports expansion. Expanded supplies
// its current state; this query supplies the requested action.
func (e Element) ExpansionRequest() (open, requested bool) {
	n := e.node()
	if n == nil {
		return false, false
	}
	n.expandable = true
	if n.st.expand == 0 {
		return false, false
	}
	open = n.st.expand > 0
	n.st.expand = 0
	n.c.rt.consumed = true
	return open, true
}
