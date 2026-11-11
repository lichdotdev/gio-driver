package driver

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

const (
	frameTime  = 16 * time.Millisecond
	winW, winH = 600, 400
	popW, popH = 120, 40
)

// Two anchors: a near the top-left, b near the bottom-right corner.
var (
	posA  = image.Pt(100, 80)
	posB  = image.Pt(440, 300)
	sizeW = image.Pt(100, 40)
)

type scene struct {
	r      input.Router
	clock  time.Time
	ops    op.Ops
	tour   Tour
	a, b   widget.Clickable
	hitsA  int
	hitsB  int
	events []Event
	last   StepState
	// focus, when set, is a widget that holds key focus and takes Escape.
	focus     bool
	focusTag  int
	focusEsc  int
	focusOnce bool
}

func newScene(steps ...Step) *scene {
	s := &scene{clock: time.Unix(1_000_000, 0)}
	s.tour.Steps = steps
	// Instant, so a click lands where the popover ends up; motion_test.go times the animations.
	s.tour.Motion = &Motion{}
	// The popover is all Next, so a click on it advances.
	s.tour.Popover = func(gtx layout.Context, st StepState) layout.Dimensions {
		s.last = st
		return st.Next.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return fill(gtx, color.NRGBA{A: 0xff}, image.Pt(popW, popH))
		})
	}
	return s
}

func fill(gtx layout.Context, c color.NRGBA, size image.Point) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func (s *scene) frame() {
	s.ops.Reset()
	s.clock = s.clock.Add(frameTime)
	gtx := layout.Context{Now: s.clock, Ops: &s.ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(winW, winH)), Source: s.r.Source()}
	for {
		e, ok := s.tour.Update(gtx)
		if !ok {
			break
		}
		s.events = append(s.events, e)
	}
	if s.a.Clicked(gtx) {
		s.hitsA++
	}
	if s.b.Clicked(gtx) {
		s.hitsB++
	}
	s.layoutFocus(gtx)
	s.tour.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		fill(gtx, color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff}, image.Pt(winW, winH))
		s.place(gtx, "a", posA, &s.a)
		s.place(gtx, "b", posB, &s.b)
		return layout.Dimensions{Size: image.Pt(winW, winH)}
	})
	s.r.Frame(gtx.Ops)
}

// place nests two offsets, so a hole in the right spot proves the anchor's
// transform came along.
func (s *scene) place(gtx layout.Context, id string, at image.Point, c *widget.Clickable) {
	defer op.Offset(image.Pt(at.X/2, at.Y/2)).Push(gtx.Ops).Pop()
	defer op.Offset(at.Sub(image.Pt(at.X/2, at.Y/2))).Push(gtx.Ops).Pop()
	s.tour.Anchor(gtx, id, func(gtx layout.Context) layout.Dimensions {
		return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return fill(gtx, color.NRGBA{G: 0xc0, A: 0xff}, sizeW) })
	})
}

func (s *scene) layoutFocus(gtx layout.Context) {
	if !s.focus {
		return
	}
	for {
		e, ok := gtx.Event(key.FocusFilter{Target: &s.focusTag}, key.Filter{Focus: &s.focusTag, Name: key.NameEscape})
		if !ok {
			break
		}
		if k, isKey := e.(key.Event); isKey && k.State == key.Press {
			s.focusEsc++
		}
	}
	event.Op(gtx.Ops, &s.focusTag)
	if !s.focusOnce {
		gtx.Execute(key.FocusCmd{Tag: &s.focusTag})
		s.focusOnce = true
	}
}

// wait runs frames until d has passed.
func (s *scene) wait(d time.Duration) {
	for end := s.clock.Add(d); s.clock.Before(end); {
		s.frame()
	}
}

func (s *scene) settle() {
	for range 3 {
		s.frame()
	}
}

func (s *scene) move(p image.Point) {
	s.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
	s.settle()
}

func (s *scene) click(p image.Point) {
	pos := f32.Pt(float32(p.X), float32(p.Y))
	s.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos},
	)
	s.settle()
}

func (s *scene) start() {
	s.settle()
	s.tour.Start()
	s.settle()
}

func center(at image.Point) image.Point { return at.Add(sizeW.Div(2)) }

func TestClickInTheHoleReachesTheAnchor(t *testing.T) {
	s := newScene(Step{Anchor: "a"})
	s.start()
	s.click(center(posA))
	if s.hitsA != 1 {
		t.Fatalf("anchor clicks = %d, want 1", s.hitsA)
	}
}

func TestClickOutsideTheHoleIsBlocked(t *testing.T) {
	s := newScene(Step{Anchor: "a"})
	s.start()
	s.click(center(posB))
	if s.hitsB != 0 {
		t.Fatalf("a widget outside the hole got %d clicks", s.hitsB)
	}
	if !s.tour.Active() {
		t.Fatal("a blocked click ended the tour")
	}
}

func TestClicksLandAgainAfterTheTour(t *testing.T) {
	s := newScene(Step{Anchor: "a"})
	s.start()
	s.tour.Stop()
	s.settle()
	s.click(center(posB))
	if s.hitsB != 1 {
		t.Fatalf("after Stop clicks = %d, want 1", s.hitsB)
	}
}

func TestPopoverSitsOnItsSide(t *testing.T) {
	s := newScene(Step{Anchor: "a", Side: Right}, Step{Anchor: "a"})
	s.start()
	s.move(image.Pt(300, 220))
	// Hole is posA grown by 6; Right puts the popover 8 past it, centred.
	s.click(image.Pt(posA.X+sizeW.X+6+8+popW/2, center(posA).Y))
	if s.tour.Index() != 1 {
		t.Fatalf("index = %d, want 1 after clicking Next", s.tour.Index())
	}
}

func TestPopoverFlipsAtTheWindowEdge(t *testing.T) {
	s := newScene(Step{Anchor: "a", Side: Right}, Step{Anchor: "b", Side: Right})
	s.start()
	s.move(image.Pt(300, 220))
	s.click(image.Pt(posA.X+sizeW.X+6+8+popW/2, center(posA).Y))
	if s.tour.Index() != 1 {
		t.Fatalf("index = %d, want 1", s.tour.Index())
	}
	// Right of b overflows a 600 wide window; Left fits, without moving the pointer.
	flipped := image.Pt(posB.X-6-8-popW/2, center(posB).Y)
	s.click(flipped)
	if len(s.events) != 1 || !s.events[0].Completed {
		t.Fatalf("events = %+v, want the tour completed by the flipped Next", s.events)
	}
}

func TestPopoverWithoutAPointerUsesItsSide(t *testing.T) {
	s := newScene(Step{Anchor: "a", Side: Bottom})
	s.start()
	o, known := s.tour.origin()
	if known {
		t.Fatalf("origin %v known before the pointer was ever seen", o)
	}
}

func TestPopoverWaitsInTheMiddleUntilThePointerIsSeen(t *testing.T) {
	s := newScene(Step{Anchor: "a", Side: Bottom})
	s.start()
	if s.tour.anim.last != modeLocal || s.tour.hasPtr {
		t.Fatal("with no pointer seen, the popover is not in the middle of the window")
	}
	// Resting on the middle popover must not hide the pointer from the anchor.
	s.move(image.Pt(winW/2, winH/2))
	if s.tour.anim.last != modeWindow {
		t.Fatal("the pointer over the middle popover did not place it")
	}
	s.click(image.Pt(center(posA).X, posA.Y+sizeW.Y+6+8+popH/2))
	if s.tour.Active() {
		t.Fatal("Next below the anchor did not finish the tour")
	}
}

func TestOriginIsTheAnchorsWindowPosition(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"})
	s.start()
	s.move(image.Pt(300, 220))
	if o, ok := s.tour.origin(); !ok || o != posA {
		t.Fatalf("origin = %v, %v; want %v", o, ok, posA)
	}
	s.tour.goTo(1)
	s.settle()
	if o, ok := s.tour.origin(); !ok || o != posB {
		t.Fatalf("after a step change with the pointer still, origin = %v, %v; want %v", o, ok, posB)
	}
}

func TestMissingAnchorCentersThePopover(t *testing.T) {
	s := newScene(Step{Anchor: "nowhere"})
	s.start()
	s.click(center(posB))
	if s.hitsB != 0 {
		t.Fatal("a centred step let a click through")
	}
	s.click(image.Pt(winW/2, winH/2))
	if s.tour.Active() {
		t.Fatal("Next in the middle of the window did not finish the tour")
	}
}

func TestUntilHoldsTheStepUntilItsActionHappens(t *testing.T) {
	done := false
	s := newScene(Step{Anchor: "a", Until: func() bool { return done }}, Step{Anchor: "b"})
	s.start()
	if !s.last.Waiting {
		t.Fatal("a step with Until is not Waiting")
	}
	s.click(image.Pt(posA.X+sizeW.X/2, posA.Y+sizeW.Y+6+8+popH/2))
	if s.tour.Index() != 0 {
		t.Fatal("Next moved a step that waits on Until")
	}
	done = true
	s.settle()
	if s.tour.Index() != 1 {
		t.Fatalf("index = %d after Until held, want 1", s.tour.Index())
	}
	if s.last.CanBack {
		t.Fatal("Back offered into a step whose action cannot be undone")
	}
}

func TestBackReturnsToAPlainStep(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "a"})
	s.start()
	if s.last.CanBack {
		t.Fatal("Back offered on the first step")
	}
	s.tour.goTo(1)
	s.settle()
	if !s.last.CanBack || !s.last.Last {
		t.Fatalf("state = %+v, want CanBack and Last", s.last)
	}
}

func TestEscapeClosesTheTour(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"})
	s.start()
	s.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	s.settle()
	if len(s.events) != 1 || s.events[0].Completed || s.events[0].Step != 0 {
		t.Fatalf("events = %+v, want one early close at step 0", s.events)
	}
}

func TestEscapeStaysWithAFocusedWidget(t *testing.T) {
	s := newScene(Step{Anchor: "a"})
	s.focus = true
	s.start()
	s.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	s.settle()
	if !s.tour.Active() {
		t.Fatal("Escape closed the tour while a focused widget takes it")
	}
	if s.focusEsc != 1 {
		t.Fatalf("focused widget got %d Escapes, want 1", s.focusEsc)
	}
}

func TestArrowsMoveTheTour(t *testing.T) {
	done := false
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"}, Step{Anchor: "a", Until: func() bool { return done }}, Step{Anchor: "b"})
	press := func(name key.Name) {
		s.r.Queue(key.Event{Name: name, State: key.Press})
		s.settle()
	}
	s.start()
	press(key.NameRightArrow)
	press(key.NameRightArrow)
	if s.tour.Index() != 2 {
		t.Fatalf("index = %d after two Rights, want 2", s.tour.Index())
	}
	press(key.NameRightArrow)
	if s.tour.Index() != 2 {
		t.Fatal("Right skipped a step that waits on Until")
	}
	press(key.NameLeftArrow)
	if s.tour.Index() != 1 {
		t.Fatalf("index = %d after Left, want 1", s.tour.Index())
	}
	done = true
	press(key.NameRightArrow)
	press(key.NameLeftArrow)
	if s.tour.Index() != 3 {
		t.Fatalf("index = %d, want Left not to go back into a step whose action is done", s.tour.Index())
	}
}

func TestPauseHidesTheTourAndKeepsItsStep(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"})
	s.start()
	s.tour.goTo(1)
	s.tour.Pause()
	s.settle()
	s.click(center(posA))
	if s.hitsA != 1 {
		t.Fatalf("a paused tour blocked a click: hits = %d", s.hitsA)
	}
	s.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	s.settle()
	if !s.tour.Active() {
		t.Fatal("Escape closed a paused tour")
	}
	s.tour.Resume()
	s.settle()
	s.click(center(posA))
	if s.hitsA != 1 || s.tour.Index() != 1 {
		t.Fatalf("after Resume: hits = %d, index = %d; want the click blocked on step 1", s.hitsA, s.tour.Index())
	}
}

func TestGoToShowsAStepOfAnActiveTour(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"}, Step{Anchor: "a"})
	s.tour.GoTo(1)
	if s.tour.Active() {
		t.Fatal("GoTo started a tour that was not running")
	}
	s.start()
	s.tour.GoTo(2)
	s.tour.GoTo(7)
	if s.tour.Index() != 2 {
		t.Fatalf("index = %d, want 2 and an out-of-range GoTo ignored", s.tour.Index())
	}
}

func TestCloseEndsEarly(t *testing.T) {
	s := newScene(Step{Anchor: "a"}, Step{Anchor: "b"})
	s.start()
	s.tour.close.Click()
	s.settle()
	if len(s.events) != 1 || s.events[0].Completed {
		t.Fatalf("events = %+v, want one early close", s.events)
	}
}

func TestPlace(t *testing.T) {
	window := image.Rect(0, 0, 600, 400)
	size := image.Pt(120, 40)
	cases := []struct {
		name     string
		side     Side
		anchor   image.Rectangle
		wantSide Side
		wantMin  image.Point
	}{
		{"fits below", Bottom, image.Rect(100, 100, 200, 140), Bottom, image.Pt(90, 148)},
		{"flips above at the bottom", Bottom, image.Rect(100, 350, 200, 390), Top, image.Pt(90, 302)},
		{"flips left at the right", Right, image.Rect(450, 100, 550, 140), Left, image.Pt(322, 100)},
		{"shifts in from the left edge", Bottom, image.Rect(0, 100, 40, 140), Bottom, image.Pt(5, 148)},
		{"stays when neither side fits", Bottom, image.Rect(0, 0, 600, 400), Bottom, image.Pt(240, 355)},
	}
	for _, c := range cases {
		box, side := place(c.side, c.anchor, size, window, 8, 5)
		if side != c.wantSide || box.Min != c.wantMin {
			t.Errorf("%s: got %v at %v, want %v at %v", c.name, side, box.Min, c.wantSide, c.wantMin)
		}
	}
}
