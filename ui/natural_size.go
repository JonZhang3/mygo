package ui

// NaturalHeight clears a fixed/percentage height while retaining minimum and
// maximum constraints. Auto is reserved for margins and is not a size value.
func (e Element) NaturalHeight() Element {
	if node := e.node(); node != nil {
		node.height = length{}
	}
	return e
}

// NaturalWidth restores intrinsic width after a shared control supplied a size.
func (e Element) NaturalWidth() Element {
	if node := e.node(); node != nil {
		node.width = length{}
	}
	return e
}
