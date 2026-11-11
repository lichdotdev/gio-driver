// Package driver runs guided tours over a Gio window: a scrim with a hole over
// one widget at a time and a popover beside it, in the spirit of driver.js.
//
// A Tour wraps the window's content with Layout and marks each widget a step
// can point at with Anchor. The look is the application's: Spotlight styles
// the scrim, and Popover draws the step, either DefaultPopover or the
// application's own widget built from StepState.
package driver

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

// far bounds the scrim and the click blockers drawn in an anchor's
// coordinates: past any window edge, whatever the anchor's offset.
const far = 1 << 15

// Step is one stop of a tour.
type Step struct {
	// Anchor is the id given to Tour.Anchor. An empty or absent anchor
	// shows the popover centered over a full scrim.
	Anchor      string
	Title, Body string
	// Hint is a line under the body, typically the action Until waits for.
	Hint string
	// Side is where the popover goes; it flips to the opposite side when
	// that side fits and this one does not.
	Side Side
	// Until, when set, holds the step until it returns true: the popover
	// offers no Next, and the person does the step's action through the
	// hole. It is checked every frame the tour lays out.
	Until func() bool
}

// Event reports the end of a tour.
type Event struct {
	// Completed is true when the last step was finished, false when the
	// tour was closed early.
	Completed bool
	// Step is the index of the step the tour ended on.
	Step int
}

// StepState is what a Popover draws: the step, where it stands in the tour
// and the clickables that drive it.
type StepState struct {
	Step         Step
	Index, Total int
	// Waiting is a step with Until: show no Next, the action moves on.
	Waiting bool
	// CanBack is false on the first step and after a step with Until,
	// whose action going back cannot undo.
	CanBack bool
	// Last is the final step, whose Next finishes the tour.
	Last              bool
	Next, Back, Close *widget.Clickable
}

// Tour is a guided tour. Its zero value with Steps and Popover set is ready
// to Start.
type Tour struct {
	Steps     []Step
	Spotlight Spotlight
	Popover   func(gtx layout.Context, s StepState) layout.Dimensions

	active            bool
	index             int
	next, back, close widget.Clickable
	pending           []Event

	window image.Point
	// drawn is the current anchor laid out this frame; floating is its
	// popover left for Layout to draw in the middle of the window.
	drawn, floating bool

	// root sees the pointer in window coordinates; measure[i] sees it in
	// step i's anchor coordinates. Their difference is the anchor's window
	// origin, which Gio does not report. A tag per step, because a pointer
	// already inside a tag gets no Enter when the step changes.
	root           int
	measure        []int
	block          int
	pointer, local f32.Point
	hasPtr, hasLoc bool
}

// Start opens the tour at its first step.
func (t *Tour) Start() {
	if len(t.Steps) == 0 {
		return
	}
	t.active = true
	t.measure = make([]int, len(t.Steps))
	t.goTo(0)
}

// Stop closes the tour without an Event.
func (t *Tour) Stop() { t.active = false }

// Active reports whether the tour is showing.
func (t *Tour) Active() bool { return t.active }

// Index is the current step.
func (t *Tour) Index() int { return t.index }

// Update processes the tour's input and reports its end, once.
func (t *Tour) Update(gtx layout.Context) (Event, bool) {
	t.update(gtx)
	if len(t.pending) == 0 {
		return Event{}, false
	}
	e := t.pending[0]
	t.pending = t.pending[1:]
	return e, true
}

func (t *Tour) update(gtx layout.Context) {
	if !t.active {
		return
	}
	if t.close.Clicked(gtx) {
		t.end(false)
		return
	}
	s := t.Steps[t.index]
	switch {
	case s.Until != nil:
		if s.Until() {
			t.advance()
		}
	case t.next.Clicked(gtx):
		t.advance()
	case t.back.Clicked(gtx) && t.canBack():
		t.goTo(t.index - 1)
	}
}

// escaped is Escape that no widget took. Gio hands a key event to the first
// filter that asks for it in the frame, so Layout asks only after the
// window's content: a focused editor or terminal keeps its own Escape.
func (t *Tour) escaped(gtx layout.Context) bool {
	esc := false
	for {
		e, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			return esc
		}
		if k, isKey := e.(key.Event); isKey && k.State == key.Press {
			esc = true
		}
	}
}

func (t *Tour) canBack() bool {
	return t.index > 0 && t.Steps[t.index-1].Until == nil
}

func (t *Tour) advance() {
	if t.index == len(t.Steps)-1 {
		t.end(true)
		return
	}
	t.goTo(t.index + 1)
}

func (t *Tour) goTo(i int) {
	t.index, t.hasLoc = i, false
}

func (t *Tour) end(completed bool) {
	t.active = false
	t.pending = append(t.pending, Event{Completed: completed, Step: t.index})
}

// Layout lays out the window's content w with the tour over it. Call it at
// the window's origin, with the window's size as the maximum constraint.
func (t *Tour) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	t.update(gtx)
	t.window = gtx.Constraints.Max
	t.drawn, t.floating = false, false
	if t.active {
		t.readPointer(gtx)
	}
	d := w(gtx)
	if t.active && t.escaped(gtx) {
		t.end(false)
		gtx.Execute(op.InvalidateCmd{})
	}
	if !t.active {
		return d
	}
	m := op.Record(gtx.Ops)
	switch {
	case !t.drawn:
		t.layoutCentered(gtx)
	case t.floating:
		t.layoutMiddle(gtx)
	}
	t.watch(gtx)
	op.Defer(gtx.Ops, m.Stop())
	return d
}

// Anchor lays out w and, when the current step points at id, the tour over
// it. id names the widget for Step.Anchor.
func (t *Tour) Anchor(gtx layout.Context, id string, w layout.Widget) layout.Dimensions {
	d := w(gtx)
	if !t.active || t.drawn || id == "" || t.Steps[t.index].Anchor != id {
		return d
	}
	t.drawn = true
	m := op.Record(gtx.Ops)
	t.layoutSpotlight(gtx, d.Size)
	op.Defer(gtx.Ops, m.Stop())
	return d
}

func (t *Tour) readPointer(gtx layout.Context) {
	const kinds = pointer.Enter | pointer.Leave | pointer.Move | pointer.Drag | pointer.Press | pointer.Release
	all := pointer.ScrollRange{Min: -far, Max: far}
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &t.root, Kinds: kinds})
		if !ok {
			break
		}
		if p, isPtr := e.(pointer.Event); isPtr && p.Kind != pointer.Leave {
			t.pointer, t.hasPtr = p.Position, true
		}
	}
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &t.measure[t.index], Kinds: kinds})
		if !ok {
			break
		}
		if p, isPtr := e.(pointer.Event); isPtr && p.Kind != pointer.Leave {
			t.local, t.hasLoc = p.Position, true
		}
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &t.block, Kinds: pointer.Press | pointer.Release | pointer.Scroll, ScrollX: all, ScrollY: all}); !ok {
			break
		}
	}
}

// origin is the current anchor's top-left in window coordinates, once the
// pointer has been seen by both the window and the anchor.
func (t *Tour) origin() (image.Point, bool) {
	if !t.hasPtr || !t.hasLoc {
		return image.Point{}, false
	}
	return t.pointer.Sub(t.local).Round(), true
}

// layoutSpotlight draws, in the anchor's coordinates, the scrim with its
// hole, the blockers around the hole and the popover.
func (t *Tour) layoutSpotlight(gtx layout.Context, size image.Point) {
	sp := t.Spotlight.orDefault()
	hole := image.Rectangle{Max: size}.Inset(-gtx.Dp(sp.Padding))
	paintScrim(gtx, sp, scrimPath(gtx.Ops, hole, gtx.Dp(sp.Radius)))
	for _, r := range around(hole) {
		a := clip.Rect(r).Push(gtx.Ops)
		event.Op(gtx.Ops, &t.block)
		a.Pop()
	}
	// Without the anchor's window position nothing keeps the popover inside
	// the window, so it waits in the middle until the pointer is seen.
	if o, ok := t.origin(); ok {
		popover, size2 := t.recordPopover(gtx)
		window := image.Rectangle{Max: t.window}
		box, _ := place(t.Steps[t.index].Side, hole.Add(o), size2, window, gtx.Dp(sp.Offset), gtx.Dp(sp.Margin))
		off := op.Offset(box.Min.Sub(o)).Push(gtx.Ops)
		popover.Add(gtx.Ops)
		off.Pop()
	} else {
		t.floating = true
	}
	// Deferred again so it lands above a floating popover too, which Layout
	// defers after this; a popover under the pointer would hide it.
	m := op.Record(gtx.Ops)
	t.measureHere(gtx)
	op.Defer(gtx.Ops, m.Stop())
}

// layoutCentered is a step without an anchor on screen: a full scrim and the
// popover in the middle of the window.
func (t *Tour) layoutCentered(gtx layout.Context) {
	sp := t.Spotlight.orDefault()
	window := image.Rectangle{Max: t.window}
	paintScrim(gtx, sp, clip.Rect(window).Path())
	a := clip.Rect(window).Push(gtx.Ops)
	event.Op(gtx.Ops, &t.block)
	a.Pop()
	t.layoutMiddle(gtx)
}

// layoutMiddle draws the popover in the middle of the window.
func (t *Tour) layoutMiddle(gtx layout.Context) {
	popover, size := t.recordPopover(gtx)
	off := op.Offset(t.window.Sub(size).Div(2)).Push(gtx.Ops)
	popover.Add(gtx.Ops)
	off.Pop()
}

// measureHere lays a pass-through area over everything in the anchor's
// coordinates, so the anchor learns where the pointer is wherever it is.
func (t *Tour) measureHere(gtx layout.Context) {
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect{Min: image.Pt(-far, -far), Max: image.Pt(far, far)}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &t.measure[t.index])
}

// watch is measureHere's counterpart in window coordinates.
func (t *Tour) watch(gtx layout.Context) {
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect{Max: t.window}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &t.root)
}

func (t *Tour) recordPopover(gtx layout.Context) (op.CallOp, image.Point) {
	if t.Popover == nil {
		return op.CallOp{}, image.Point{}
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: t.window}
	d := t.Popover(g, t.state())
	return m.Stop(), d.Size
}

func (t *Tour) state() StepState {
	s := t.Steps[t.index]
	return StepState{
		Step: s, Index: t.index, Total: len(t.Steps),
		Waiting: s.Until != nil, CanBack: t.canBack(), Last: t.index == len(t.Steps)-1,
		Next: &t.next, Back: &t.back, Close: &t.close,
	}
}

func paintScrim(gtx layout.Context, sp Spotlight, path clip.PathSpec) {
	defer clip.Outline{Path: path}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: sp.Scrim}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// around is everything outside the hole, as four rectangles.
func around(h image.Rectangle) [4]image.Rectangle {
	return [4]image.Rectangle{
		{Min: image.Pt(-far, -far), Max: image.Pt(far, h.Min.Y)},
		{Min: image.Pt(-far, h.Max.Y), Max: image.Pt(far, far)},
		{Min: image.Pt(-far, h.Min.Y), Max: image.Pt(h.Min.X, h.Max.Y)},
		{Min: image.Pt(h.Max.X, h.Min.Y), Max: image.Pt(far, h.Max.Y)},
	}
}

// scrimPath is everything with the rounded hole cut out, the hole wound
// against the outside.
func scrimPath(ops *op.Ops, h image.Rectangle, radius int) clip.PathSpec {
	x0, y0, x1, y1 := float32(h.Min.X), float32(h.Min.Y), float32(h.Max.X), float32(h.Max.Y)
	r := min(float32(radius), (x1-x0)/2, (y1-y0)/2)
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(-far, -far))
	p.LineTo(f32.Pt(far, -far))
	p.LineTo(f32.Pt(far, far))
	p.LineTo(f32.Pt(-far, far))
	p.Close()
	p.MoveTo(f32.Pt(x0+r, y0))
	p.QuadTo(f32.Pt(x0, y0), f32.Pt(x0, y0+r))
	p.LineTo(f32.Pt(x0, y1-r))
	p.QuadTo(f32.Pt(x0, y1), f32.Pt(x0+r, y1))
	p.LineTo(f32.Pt(x1-r, y1))
	p.QuadTo(f32.Pt(x1, y1), f32.Pt(x1, y1-r))
	p.LineTo(f32.Pt(x1, y0+r))
	p.QuadTo(f32.Pt(x1, y0), f32.Pt(x1-r, y0))
	p.Close()
	return p.End()
}
