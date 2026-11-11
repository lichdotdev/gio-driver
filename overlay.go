package driver

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// popoverSlide is how far a popover travels toward its place as it comes in.
const popoverSlide = 6

// layoutOverlay draws the overlay in window coordinates, deferred by Layout
// over the window's content.
func (t *Tour) layoutOverlay(gtx layout.Context) {
	sp, m := t.Spotlight.orDefault(), t.motion()
	scrim, pop := t.anim.scrimAlpha(t.now, m), t.anim.popAlpha(t.now, m)
	window := image.Rectangle{Max: t.window}
	switch t.anim.last {
	case modeCentered:
		paintScrim(gtx, sp, clip.Rect(window).Path(), scrim)
		t.blockAround(gtx, image.Rectangle{})
		t.layoutPopover(gtx, t.middle, -1, pop)
	case modeWindow, modePending:
		hole := t.anim.hole(t.now, m)
		paintScrim(gtx, sp, scrimPath(gtx.Ops, hole, gtx.Dp(sp.Radius)), scrim)
		t.blockAround(gtx, hole)
		if t.anim.last == modeWindow {
			side := t.Steps[t.index].Side
			t.layoutPopover(gtx, func(size image.Point) (image.Point, Side) {
				box, s := place(side, t.anim.target, size, window, gtx.Dp(sp.Offset), gtx.Dp(sp.Margin))
				return box.Min, s
			}, 0, pop)
		}
	case modeLocal:
		if !t.hasPtr {
			t.layoutPopover(gtx, t.middle, -1, pop)
		}
	}
}

func (t *Tour) middle(size image.Point) (image.Point, Side) {
	return t.window.Sub(size).Div(2), Bottom
}

// layoutLocal is the overlay without the anchor's window position: the scrim
// and the blockers in the anchor's own coordinates, which need none.
func (t *Tour) layoutLocal(gtx layout.Context, hole image.Rectangle) {
	sp := t.Spotlight.orDefault()
	m := op.Record(gtx.Ops)
	paintScrim(gtx, sp, scrimPath(gtx.Ops, hole, gtx.Dp(sp.Radius)), t.anim.scrimAlpha(t.now, t.motion()))
	t.blockAround(gtx, hole)
	op.Defer(gtx.Ops, m.Stop())
}

// blockAround takes the clicks and scrolls outside hole while the tour is
// active; an empty hole blocks everything. An ending tour blocks nothing.
func (t *Tour) blockAround(gtx layout.Context, hole image.Rectangle) {
	if !t.active {
		return
	}
	rects := around(hole)
	if hole.Empty() {
		rects = [4]image.Rectangle{{Min: image.Pt(-far, -far), Max: image.Pt(far, far)}}
	}
	for _, r := range rects {
		if r.Empty() {
			continue
		}
		a := clip.Rect(r).Push(gtx.Ops)
		event.Op(gtx.Ops, &t.block)
		a.Pop()
	}
}

// layoutPopover draws the popover where at puts it, faded and scaled by
// alpha and sliding in toward its anchor. slide < 0 is a popover with no
// anchor to slide toward.
func (t *Tour) layoutPopover(gtx layout.Context, at func(size image.Point) (image.Point, Side), slide int, alpha float32) {
	if t.Popover == nil || alpha <= 0 {
		return
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: t.window}
	size := t.Popover(g, t.state()).Size
	call := m.Stop()
	pos, side := at(size)
	defer paint.PushOpacity(gtx.Ops, alpha).Pop()
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	shift := f32.Point{}
	if slide >= 0 {
		shift = toward(side).Mul(float32(gtx.Dp(popoverSlide)) * (1 - alpha))
	}
	scale := 0.96 + 0.04*alpha
	c := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
	defer op.Affine(f32.AffineId().Scale(c, f32.Pt(scale, scale)).Offset(shift)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// toward points from a popover on side back at its anchor.
func toward(side Side) f32.Point {
	switch side {
	case Top:
		return f32.Pt(0, 1)
	case Right:
		return f32.Pt(-1, 0)
	case Left:
		return f32.Pt(1, 0)
	}
	return f32.Pt(0, -1)
}

// deferMeasure lays the anchor's pass-through measuring area over everything,
// in the anchor's coordinates. Deferred twice, so it lands above the overlay
// Layout defers after it: a popover under the pointer would hide it.
func (t *Tour) deferMeasure(gtx layout.Context) {
	outer := op.Record(gtx.Ops)
	inner := op.Record(gtx.Ops)
	func() {
		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		defer clip.Rect{Min: image.Pt(-far, -far), Max: image.Pt(far, far)}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, &t.measure[t.index])
	}()
	op.Defer(gtx.Ops, inner.Stop())
	op.Defer(gtx.Ops, outer.Stop())
}

// watch is deferMeasure's counterpart in window coordinates.
func (t *Tour) watch(gtx layout.Context) {
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect{Max: t.window}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &t.root)
}

func paintScrim(gtx layout.Context, sp Spotlight, path clip.PathSpec, alpha float32) {
	c := sp.Scrim
	c.A = uint8(float32(c.A)*alpha + 0.5)
	if c.A == 0 {
		return
	}
	defer clip.Outline{Path: path}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
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
