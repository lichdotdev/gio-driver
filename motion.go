package driver

import (
	"image"
	"time"
)

// Motion times the tour's animations. A zero duration makes that animation
// instant.
type Motion struct {
	// Enter fades the scrim in and brings each step's popover in.
	Enter time.Duration
	// Exit fades the scrim and the popover out when the tour ends.
	Exit time.Duration
	// Glide moves the hole from one step's anchor to the next; the popover
	// comes in once it lands.
	Glide time.Duration
}

// DefaultMotion is the motion of a nil Tour.Motion.
var DefaultMotion = Motion{Enter: 160 * time.Millisecond, Exit: 180 * time.Millisecond, Glide: 260 * time.Millisecond}

// pendingLimit is how long a step change keeps the previous hole while the
// next anchor's position arrives, normally one frame. A pointer that left
// the window never sends it, and the step falls back to drawing in place.
const pendingLimit = 150 * time.Millisecond

// mode is how a frame draws the overlay.
type mode uint8

const (
	// modeCentered: no anchor on screen, a full scrim and a centered popover.
	modeCentered mode = iota
	// modeWindow: the anchor's window position is known; everything is
	// drawn in window coordinates and can animate.
	modeWindow
	// modePending: a step just changed and the new anchor's position is a
	// frame away; the previous hole holds and the popover waits.
	modePending
	// modeLocal: the position is unknown, so the scrim is drawn in the
	// anchor's own coordinates and the popover in the middle of the window.
	modeLocal
)

// anim is the tour's animation state, in window coordinates.
type anim struct {
	fresh              bool
	target, from       image.Rectangle
	hasTarget, hasFrom bool
	shownAt, stepAt    time.Time
	glideAt, popAt     time.Time
	exitAt             time.Time
	exiting            bool
	last               mode
}

// begin stamps a tour started or resumed between frames with the first
// frame's time; a popover that was showing comes back in with the scrim.
func (a *anim) begin(now time.Time) {
	if !a.fresh {
		return
	}
	a.fresh, a.shownAt, a.stepAt = false, now, now
	if !a.popAt.IsZero() {
		a.popAt = now
	}
}

// step starts a new step: the hole it leaves is where the next one glides
// from.
func (a *anim) step(now time.Time) {
	if a.hasTarget {
		a.from, a.hasFrom = a.target, true
	}
	a.hasTarget, a.stepAt, a.glideAt, a.popAt = false, now, time.Time{}, time.Time{}
}

// aim sets the current anchor's hole. The first aim of a step glides from
// the previous hole and brings the popover in after; later ones follow the
// anchor without animating.
func (a *anim) aim(hole image.Rectangle, now time.Time, m Motion) {
	if a.hasTarget {
		a.target = hole
		return
	}
	a.target, a.hasTarget = hole, true
	if a.hasFrom && a.from != hole {
		a.glideAt, a.popAt = now, now.Add(m.Glide)
		return
	}
	a.popAt = now
}

func (a *anim) pending(hasPtr bool, now time.Time) bool {
	return hasPtr && a.hasFrom && !a.hasTarget && now.Sub(a.stepAt) < pendingLimit
}

// show records the frame's mode for the exit animation, and starts the
// popover of a step whose popover sits in the middle of the window.
func (a *anim) show(m mode, now time.Time, hasPtr bool) {
	a.last = m
	if a.popAt.IsZero() && (m == modeCentered || m == modeLocal && !hasPtr) {
		a.popAt = now
	}
}

func (a *anim) exit(now time.Time) {
	a.exiting, a.exitAt = true, now
}

// settle ends a finished exit.
func (a *anim) settle(now time.Time, m Motion) {
	if a.exiting && progress(now, a.exitAt, m.Exit) >= 1 {
		a.exiting = false
	}
}

// hole is where the hole is drawn now: gliding toward the target, or the
// previous one while the target is pending.
func (a *anim) hole(now time.Time, m Motion) image.Rectangle {
	if !a.hasTarget {
		return a.from
	}
	if a.glideAt.IsZero() {
		return a.target
	}
	return lerpRect(a.from, a.target, ease(progress(now, a.glideAt, m.Glide)))
}

func (a *anim) scrimAlpha(now time.Time, m Motion) float32 {
	if a.exiting {
		return 1 - ease(progress(now, a.exitAt, m.Exit))
	}
	return ease(progress(now, a.shownAt, m.Enter))
}

func (a *anim) popAlpha(now time.Time, m Motion) float32 {
	if a.popAt.IsZero() {
		return 0
	}
	in := ease(progress(now, a.popAt, m.Enter))
	if a.exiting {
		return in * (1 - ease(progress(now, a.exitAt, m.Exit)))
	}
	return in
}

func (a *anim) running(now time.Time, m Motion) bool {
	busy := func(at time.Time, d time.Duration) bool { return !at.IsZero() && now.Before(at.Add(d)) }
	return a.exiting || busy(a.shownAt, m.Enter) || busy(a.glideAt, m.Glide) || busy(a.popAt, m.Enter) ||
		a.hasFrom && !a.hasTarget
}

// progress is how far now is into d from start, from 0 to 1.
func progress(now, start time.Time, d time.Duration) float32 {
	if d <= 0 {
		return 1
	}
	p := float32(now.Sub(start)) / float32(d)
	return min(max(p, 0), 1)
}

// ease is an ease-out cubic.
func ease(p float32) float32 {
	q := 1 - p
	return 1 - q*q*q
}

func lerpRect(a, b image.Rectangle, p float32) image.Rectangle {
	l := func(x, y int) int { return x + int(float32(y-x)*p+0.5) }
	return image.Rect(l(a.Min.X, b.Min.X), l(a.Min.Y, b.Min.Y), l(a.Max.X, b.Max.X), l(a.Max.Y, b.Max.Y))
}
