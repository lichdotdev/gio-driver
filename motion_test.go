package driver

import (
	"image"
	"testing"
	"time"
)

func animated(steps ...Step) *scene {
	s := newScene(steps...)
	s.tour.Motion = nil
	return s
}

func TestScrimFadesIn(t *testing.T) {
	s := animated(Step{Anchor: "a"})
	s.settle()
	s.tour.Start()
	s.frame()
	s.frame()
	m := DefaultMotion
	if a := s.tour.anim.scrimAlpha(s.clock, m); a <= 0 || a >= 1 {
		t.Fatalf("scrim alpha a frame into the fade = %v, want between 0 and 1", a)
	}
	s.wait(m.Enter)
	if a := s.tour.anim.scrimAlpha(s.clock, m); a != 1 {
		t.Fatalf("scrim alpha after Enter = %v, want 1", a)
	}
}

func TestHoleGlidesToTheNextAnchorAndThePopoverFollows(t *testing.T) {
	s := animated(Step{Anchor: "a"}, Step{Anchor: "b"})
	s.start()
	s.move(image.Pt(300, 220))
	s.wait(time.Second)
	m := DefaultMotion
	s.tour.goTo(1)
	s.wait(m.Glide / 2)
	hole := s.tour.anim.hole(s.clock, m)
	if hole.Min.X <= posA.X-6 || hole.Min.X >= posB.X-6 {
		t.Fatalf("hole halfway through the glide at %v, want between a and b", hole)
	}
	if a := s.tour.anim.popAlpha(s.clock, m); a != 0 {
		t.Fatalf("popover alpha mid-glide = %v, want 0 until the hole lands", a)
	}
	s.wait(m.Glide + m.Enter)
	want := image.Rectangle{Min: posB, Max: posB.Add(sizeW)}.Inset(-6)
	if hole := s.tour.anim.hole(s.clock, m); hole != want {
		t.Fatalf("hole after the glide = %v, want %v", hole, want)
	}
	if a := s.tour.anim.popAlpha(s.clock, m); a != 1 {
		t.Fatalf("popover alpha after landing = %v, want 1", a)
	}
}

func TestEndingFadesOutAndLetsClicksThrough(t *testing.T) {
	s := animated(Step{Anchor: "a"})
	s.start()
	s.wait(time.Second)
	s.tour.close.Click()
	s.frame()
	if !s.tour.anim.exiting {
		t.Fatal("closing the tour did not fade it out")
	}
	s.click(center(posB))
	if s.hitsB != 1 {
		t.Fatalf("a click during the fade-out got %d hits, want 1", s.hitsB)
	}
	s.wait(DefaultMotion.Exit)
	if s.tour.anim.exiting {
		t.Fatal("the fade-out never ended")
	}
}

func TestPendingStepFallsBackWhenThePointerNeverArrives(t *testing.T) {
	a := anim{hasFrom: true, stepAt: time.Unix(0, 0)}
	if !a.pending(true, time.Unix(0, int64(pendingLimit)/2)) {
		t.Fatal("a fresh step change is not pending")
	}
	if a.pending(true, time.Unix(0, int64(pendingLimit))) {
		t.Fatal("a step change past pendingLimit still pending")
	}
	if a.pending(false, time.Unix(0, 0)) {
		t.Fatal("pending without a pointer to report the anchor")
	}
}
