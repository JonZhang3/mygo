package ui

import (
	"fmt"
	"math"
	"slices"
)

// MultiSliderOptions describes the input contract of a range with any number
// of thumbs. Presentation remains with the caller, not the platform theme.
type MultiSliderOptions struct {
	Min, Max, Step                           float64
	MinStepsBetweenThumbs                    int
	Vertical, Inverted, Disabled, Controlled bool
	Label                                    string
	ThumbLabels                              []string
	OnValueChange, OnValueCommit             func([]float64)
}

// MultiSliderParts holds handles valid only in the current build.
type MultiSliderParts struct {
	Root   Element
	Thumbs []Element
}

type multiSliderState struct {
	active  int
	held    bool
	start   []float64
	pointer Point
}

// MultiSliderBase supports track presses, captured drags, crossing thumbs,
// keyboard stepping and accessibility actions. A rejected minimum-distance
// edit never mutates the owner's values or emits a commit.
func MultiSliderBase(c *Context, values *[]float64, o MultiSliderOptions) MultiSliderParts {
	ctx := c.build()
	if ctx == nil {
		return MultiSliderParts{}
	}
	ctx.rt.beginBuild()
	defer ctx.rt.endBuild()
	step := o.Step
	if step <= 0 {
		step = 1
	}
	lo, hi := o.Min, math.Max(o.Min, o.Max)
	root := coreBox(ctx).Role(RoleGroup).Label(o.Label).Disabled(o.Disabled).Shrink(0)
	root.flags |= flagDraggable | flagHover
	if o.Vertical {
		root.Width(16).MinHeight(176).PaddingY(8)
	} else {
		root.Height(6).MinWidth(32).PaddingX(8)
	}
	state := coreLocal(root, "range-input", func() multiSliderState { return multiSliderState{active: -1} })
	if o.Disabled {
		state.active, state.held = -1, false
	}
	count := len(*values)
	thumbs := make([]*node, count)
	result := MultiSliderParts{Root: wrapElement(root), Thumbs: make([]Element, count)}
	reversed := o.Inverted
	if !o.Vertical && root.rtl() {
		reversed = !reversed
	}
	set := func(index int, to float64, commit bool) {
		if index < 0 || index >= len(*values) || root.disabled() || math.IsNaN(to) {
			return
		}
		next := slices.Clone(*values)
		to = max(lo, min(hi, snap(to, lo, math.Inf(1), step)))
		if next[index] == to {
			return
		}
		next[index] = to
		slices.Sort(next)
		for i := 1; i < len(next); i++ {
			if next[i]-next[i-1]+1e-10 < float64(max(0, o.MinStepsBetweenThumbs))*step {
				return
			}
		}
		state.active = slices.Index(next, to)
		if !o.Controlled {
			*values = slices.Clone(next)
		}
		root.st.markChanged()
		ctx.rt.consumed = true
		if state.active >= 0 && state.active < len(thumbs) && thumbs[state.active] != nil {
			thumbs[state.active].Focus()
		}
		if o.OnValueChange != nil {
			o.OnValueChange(slices.Clone(next))
		}
		if commit && o.OnValueCommit != nil {
			o.OnValueCommit(slices.Clone(next))
		}
	}
	root.Children(func() {
		for i, value := range *values {
			f := fraction(value, lo, hi)
			if reversed {
				f = 1 - f
			}
			thumb := coreBox(ctx).Absolute().Size(16, 16).Focusable().FocusRing(false).Role(RoleSlider).Range(lo, hi, value).Step(step)
			thumb.flags |= flagDraggable | flagHover
			if o.Vertical {
				thumb.orient = Vertical
				thumb.BottomPercent(f*100).Margin(0, 0, -f*16, 0)
			} else {
				thumb.Top(-5).StartPercent(f * 100).MarginStart(-f * 16)
			}
			label := o.Label
			if i < len(o.ThumbLabels) {
				label = o.ThumbLabels[i]
			} else if count == 2 {
				label += []string{" minimum", " maximum"}[i]
			} else if count > 2 {
				label += fmt.Sprintf(" %d of %d", i+1, count)
			}
			thumb.Label(label)
			thumb.afterInput(func() {
				if i >= len(*values) {
					return
				}
				back, forth := KeyLeft, KeyRight
				if reversed && !o.Vertical {
					back, forth = forth, back
				}
				up, down := KeyUp, KeyDown
				if o.Vertical && o.Inverted {
					up, down = down, up
				}
				v := (*values)[i]
				switch {
				case thumb.Shortcut(0, KeyHome):
					set(0, lo, true)
				case thumb.Shortcut(0, KeyEnd):
					set(len(*values)-1, hi, true)
				case thumb.Shortcut(0, KeyPageUp), thumb.Shortcut(Shift, forth), thumb.Shortcut(Shift, up):
					set(i, v+10*step, true)
				case thumb.Shortcut(0, KeyPageDown), thumb.Shortcut(Shift, back), thumb.Shortcut(Shift, down):
					set(i, v-10*step, true)
				case thumb.Shortcut(0, forth), thumb.Shortcut(0, up):
					set(i, v+step, true)
				case thumb.Shortcut(0, back), thumb.Shortcut(0, down):
					set(i, v-step, true)
				}
			})
			thumbs[i], result.Thumbs[i] = thumb, wrapElement(thumb)
		}
	})
	root.afterInput(func() {
		pointer := root.st
		which := -1
		canceled := root.PressCanceled()
		for i, thumb := range thumbs {
			if thumb.st.pressedPrimary() {
				pointer, which = thumb.st, i
			}
			canceled = canceled || thumb.PressCanceled()
		}
		if !pointer.pressedPrimary() || root.disabled() || count == 0 {
			if state.held && !canceled && !root.disabled() && !slices.Equal(state.start, *values) && o.OnValueCommit != nil {
				o.OnValueCommit(slices.Clone(*values))
			}
			state.active, state.held = -1, false
			return
		}
		x, y := root.st.local(pointer.pointer(ctx.rt))
		x, y = x+root.st.x, y+root.st.y
		fraction := (x - root.st.x) / max(1, root.st.w)
		if o.Vertical {
			fraction = 1 - (y-root.st.y)/max(1, root.st.h)
		}
		if reversed {
			fraction = 1 - fraction
		}
		to := lo + float64(max(0, min(1, fraction)))*(hi-lo)
		firstPress := !state.held
		if firstPress {
			state.pointer = Point{X: x, Y: y}
			state.start = slices.Clone(*values)
			state.active = which
			if state.active < 0 {
				state.active = 0
				for i, v := range *values {
					if math.Abs(v-to) < math.Abs((*values)[state.active]-to) {
						state.active = i
					}
				}
			}
			state.held = true
		}
		if firstPress && which < 0 || !firstPress && state.pointer != (Point{X: x, Y: y}) {
			set(state.active, to, false)
		}
		state.pointer = Point{X: x, Y: y}
		if state.active >= 0 && state.active < len(thumbs) {
			thumbs[state.active].Focus()
		}
	})
	return result
}
