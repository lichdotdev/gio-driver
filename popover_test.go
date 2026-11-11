package driver

import (
	"image"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func testTheme() *material.Theme {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	return th
}

func TestDefaultPopoverFillsZeroFields(t *testing.T) {
	st := PopoverStyle{Width: 280, Labels: Labels{Next: "Weiter"}}.withDefaults()
	if st.Width != 280 || st.Padding != defaultPopover.Padding {
		t.Errorf("sizes = %v/%v, want 280 and the default padding", st.Width, st.Padding)
	}
	if st.Labels.Next != "Weiter" || st.Labels.Done != "Done" || st.Labels.Progress(1, 6) != "2 of 6" {
		t.Errorf("labels = %+v", st.Labels)
	}
	if st.Background != defaultPopover.Background {
		t.Errorf("background = %v, want the default", st.Background)
	}
}

func TestDefaultPopoverIsItsWidth(t *testing.T) {
	draw := DefaultPopover(PopoverStyle{Theme: testTheme()})
	var next, back, closer widget.Clickable
	for _, s := range []StepState{
		{Step: Step{Title: "Open a project", Body: "Pick a folder.", Hint: "Click Open project"}, Total: 3, Waiting: true, Next: &next, Back: &back, Close: &closer},
		{Step: Step{Title: "Done", Body: "That is all."}, Index: 2, Total: 3, CanBack: true, Last: true, Next: &next, Back: &back, Close: &closer},
	} {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(800, 600)}}
		d := draw(gtx, s)
		if d.Size.X != 320 || d.Size.Y == 0 {
			t.Errorf("step %d: size = %v, want 320 wide", s.Index, d.Size)
		}
	}
}
