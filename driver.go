// Package driver runs guided tours over a Gio window: a scrim with a hole over
// one widget at a time and a popover beside it, in the spirit of driver.js.
//
// A Tour wraps the window's content with Layout and marks each widget a step
// can point at with Anchor. The look is the application's: Spotlight styles
// the scrim, Motion times the animations, and Popover draws the step, either
// DefaultPopover or the application's own widget built from StepState.
package driver

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
)

// far bounds the scrim and the click blockers: past any window edge,
// whatever the offset they are drawn at.
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
	// Motion times the animations; nil is DefaultMotion, and a zero Motion
	// turns them off.
	Motion  *Motion
	Popover func(gtx layout.Context, s StepState) layout.Dimensions

	active, paused    bool
	index             int
	next, back, close widget.Clickable
	pending           []Event
	now               time.Time

	window image.Point
	// drawn is the current anchor laid out this frame, and mode how the
	// overlay is drawn this frame.
	drawn bool
	mode  mode

	// root sees the pointer in window coordinates; measure[i] sees it in
	// step i's anchor coordinates. Their difference is the anchor's window
	// origin, which Gio does not report. A tag per step, because a pointer
	// already inside a tag gets no Enter when the step changes.
	root           int
	measure        []int
	block          int
	pointer, local f32.Point
	hasPtr, hasLoc bool

	anim anim
}

// Start opens the tour at its first step.
func (t *Tour) Start() {
	if len(t.Steps) == 0 {
		return
	}
	t.active, t.index, t.hasLoc = true, 0, false
	t.measure = make([]int, len(t.Steps))
	t.anim = anim{fresh: true}
}

// Stop closes the tour without an Event.
func (t *Tour) Stop() {
	if t.active {
		t.active = false
		t.anim.exit(t.now)
	}
}

// Pause hides the tour and lets every click and key through, keeping its
// step: for a dialog that opens over the window mid-tour.
func (t *Tour) Pause() { t.paused = true }

// Resume shows a paused tour again, fading it back in.
func (t *Tour) Resume() {
	if t.paused {
		t.paused = false
		t.anim.fresh = true
	}
}

// GoTo shows step i of an active tour, for an application that sends it
// back to the step that opens a menu or dialog the person closed early.
func (t *Tour) GoTo(i int) {
	if t.active && i >= 0 && i < len(t.Steps) && i != t.index {
		t.goTo(i)
	}
}

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
	t.now = gtx.Now
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

// keys reads Escape, which closes the tour, and the arrows, which move it.
// Gio hands a key event to the first filter that asks for it in the frame,
// so Layout asks only after the window's content: a focused editor or
// terminal keeps its own keys.
func (t *Tour) keys(gtx layout.Context) {
	for {
		e, ok := gtx.Event(key.Filter{Name: key.NameEscape}, key.Filter{Name: key.NameRightArrow}, key.Filter{Name: key.NameLeftArrow})
		if !ok {
			return
		}
		k, isKey := e.(key.Event)
		if !isKey || k.State != key.Press || !t.active {
			continue
		}
		switch k.Name {
		case key.NameEscape:
			t.end(false)
		case key.NameRightArrow:
			if t.Steps[t.index].Until == nil {
				t.advance()
			}
		case key.NameLeftArrow:
			if t.canBack() {
				t.goTo(t.index - 1)
			}
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
	t.anim.step(t.now)
}

func (t *Tour) end(completed bool) {
	t.active = false
	t.anim.exit(t.now)
	t.pending = append(t.pending, Event{Completed: completed, Step: t.index})
}

func (t *Tour) motion() Motion {
	if t.Motion == nil {
		return DefaultMotion
	}
	return *t.Motion
}

// Layout lays out the window's content w with the tour over it. Call it at
// the window's origin, with the window's size as the maximum constraint.
func (t *Tour) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	if t.paused {
		t.now = gtx.Now
		return w(gtx)
	}
	t.update(gtx)
	t.window = gtx.Constraints.Max
	t.drawn, t.mode = false, modeCentered
	if t.active {
		t.anim.begin(t.now)
		t.readPointer(gtx)
	}
	d := w(gtx)
	if t.active {
		t.keys(gtx)
	}
	t.anim.settle(t.now, t.motion())
	if !t.active && !t.anim.exiting {
		return d
	}
	if t.active {
		t.anim.show(t.mode, t.now, t.hasPtr)
	}
	m := op.Record(gtx.Ops)
	t.layoutOverlay(gtx)
	if t.active {
		t.watch(gtx)
	}
	op.Defer(gtx.Ops, m.Stop())
	if t.anim.running(t.now, t.motion()) {
		gtx.Execute(op.InvalidateCmd{})
	}
	return d
}

// Anchor lays out w and, when the current step points at id, the tour over
// it. id names the widget for Step.Anchor.
func (t *Tour) Anchor(gtx layout.Context, id string, w layout.Widget) layout.Dimensions {
	d := w(gtx)
	if !t.active || t.paused || t.drawn || id == "" || t.Steps[t.index].Anchor != id {
		return d
	}
	t.drawn = true
	hole := image.Rectangle{Max: d.Size}.Inset(-gtx.Dp(t.Spotlight.orDefault().Padding))
	switch o, ok := t.origin(); {
	case ok:
		t.mode = modeWindow
		t.anim.aim(hole.Add(o), t.now, t.motion())
	case t.anim.pending(t.hasPtr, t.now):
		t.mode = modePending
	default:
		t.mode = modeLocal
		t.layoutLocal(gtx, hole)
	}
	t.deferMeasure(gtx)
	return d
}

func (t *Tour) readPointer(gtx layout.Context) {
	const kinds = pointer.Enter | pointer.Leave | pointer.Move | pointer.Drag | pointer.Press | pointer.Release
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &t.root, Kinds: kinds})
		if !ok {
			break
		}
		if p, isPtr := e.(pointer.Event); isPtr {
			t.pointer, t.hasPtr = p.Position, p.Kind != pointer.Leave
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
	all := pointer.ScrollRange{Min: -far, Max: far}
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

func (t *Tour) state() StepState {
	s := t.Steps[t.index]
	return StepState{
		Step: s, Index: t.index, Total: len(t.Steps),
		Waiting: s.Until != nil, CanBack: t.canBack(), Last: t.index == len(t.Steps)-1,
		Next: &t.next, Back: &t.back, Close: &t.close,
	}
}
