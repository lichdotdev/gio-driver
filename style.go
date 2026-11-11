package driver

import (
	"image"
	"image/color"

	"gioui.org/unit"
)

// Side is where a popover sits around its anchor.
type Side uint8

const (
	Bottom Side = iota
	Top
	Right
	Left
)

func (s Side) opposite() Side {
	switch s {
	case Bottom:
		return Top
	case Top:
		return Bottom
	case Right:
		return Left
	}
	return Right
}

// Spotlight styles the scrim and the hole. Its zero value is
// DefaultSpotlight.
//
// Gio blends in linear light, so a scrim reads lighter than the same alpha
// in a browser: 50% black over a mid grey lands about a quarter darker, not
// half. Pick Scrim by eye in the application.
type Spotlight struct {
	Scrim color.NRGBA
	// Padding grows the hole past the anchor; Radius rounds its corners.
	Padding, Radius unit.Dp
	// Offset is the gap between the hole and the popover; Margin the room
	// the popover keeps from the window's edges.
	Offset, Margin unit.Dp
}

// DefaultSpotlight is the look of a zero Spotlight.
var DefaultSpotlight = Spotlight{Scrim: color.NRGBA{A: 0x80}, Padding: 6, Radius: 6, Offset: 8, Margin: 5}

func (s Spotlight) orDefault() Spotlight {
	if s == (Spotlight{}) {
		return DefaultSpotlight
	}
	return s
}

// beside is a box of size on side of anchor, centred along it.
func beside(side Side, anchor image.Rectangle, size image.Point, offset int) image.Rectangle {
	mid := anchor.Min.Add(anchor.Size().Sub(size).Div(2))
	var at image.Point
	switch side {
	case Top:
		at = image.Pt(mid.X, anchor.Min.Y-offset-size.Y)
	case Right:
		at = image.Pt(anchor.Max.X+offset, mid.Y)
	case Left:
		at = image.Pt(anchor.Min.X-offset-size.X, mid.Y)
	default:
		at = image.Pt(mid.X, anchor.Max.Y+offset)
	}
	return image.Rectangle{Min: at, Max: at.Add(size)}
}

// place puts a box of size beside anchor inside window, both in window
// coordinates: on side, or the opposite side when only that one fits, then
// shifted to keep margin from every edge, the top-left winning when the box
// cannot fit at all. It returns the side the box landed on.
func place(side Side, anchor image.Rectangle, size image.Point, window image.Rectangle, offset, margin int) (image.Rectangle, Side) {
	inside := window.Inset(margin)
	box := beside(side, anchor, size, offset)
	if flipped := beside(side.opposite(), anchor, size, offset); !fits(side, box, inside) && fits(side.opposite(), flipped, inside) {
		side, box = side.opposite(), flipped
	}
	return box.Add(image.Pt(shift(box.Min.X, size.X, inside.Min.X, inside.Max.X), shift(box.Min.Y, size.Y, inside.Min.Y, inside.Max.Y))), side
}

// fits is whether box stays inside on the axis side pushes it along.
func fits(side Side, box, inside image.Rectangle) bool {
	switch side {
	case Top:
		return box.Min.Y >= inside.Min.Y
	case Right:
		return box.Max.X <= inside.Max.X
	case Left:
		return box.Min.X >= inside.Min.X
	}
	return box.Max.Y <= inside.Max.Y
}

// shift moves a span of length n starting at start into [lo, hi], lo winning.
func shift(start, n, lo, hi int) int {
	d := 0
	if start+n > hi {
		d = hi - (start + n)
	}
	if start+d < lo {
		d = lo - start
	}
	return d
}
