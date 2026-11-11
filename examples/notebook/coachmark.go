package main

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget/material"

	driver "github.com/lichdotdev/gio-driver"
)

// coachMark is the app's own popover, built from the tour's StepState: what
// a design system plugs into Tour.Popover in place of DefaultPopover.
func (nb *notebook) coachMark(gtx layout.Context, s driver.StepState) layout.Dimensions {
	width := gtx.Dp(340)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return outlined(gtx, gtx.Constraints.Min, nb.pal.card, nb.pal.border, gtx.Dp(14))
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				rows := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.dots(gtx, s.Index, s.Total) }),
					layout.Rigid(layout.Spacer{Height: 16}.Layout),
					layout.Rigid(nb.label(s.Step.Title, 17, nb.pal.fg, font.SemiBold)),
					layout.Rigid(layout.Spacer{Height: 6}.Layout),
					layout.Rigid(nb.wrapped(s.Step.Body)),
				}
				if s.Step.Hint != "" {
					rows = append(rows, layout.Rigid(layout.Spacer{Height: 14}.Layout), layout.Rigid(nb.hint(s.Step.Hint)))
				}
				rows = append(rows, layout.Rigid(layout.Spacer{Height: 20}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return nb.coachActions(gtx, s)
				}))
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		}),
	)
}

// dots is the tour's progress: a pill for the current step, dots for the rest.
func (nb *notebook) dots(gtx layout.Context, index, total int) layout.Dimensions {
	children := make([]layout.FlexChild, 0, 2*total)
	for i := range total {
		c, w := nb.pal.border, gtx.Dp(6)
		if i == index {
			c, w = nb.pal.accent, gtx.Dp(20)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return box(gtx, image.Pt(w, gtx.Dp(6)), c, gtx.Dp(3))
		}), layout.Rigid(layout.Spacer{Width: 5}.Layout))
	}
	return layout.Flex{}.Layout(gtx, children...)
}

func (nb *notebook) hint(s string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return box(gtx, image.Pt(gtx.Dp(8), gtx.Dp(8)), nb.pal.accent, gtx.Dp(4))
			}),
			layout.Rigid(layout.Spacer{Width: 8}.Layout),
			layout.Rigid(nb.label(s, 13, nb.pal.fg, font.Medium)),
		)
	}
}

func (nb *notebook) coachActions(gtx layout.Context, s driver.StepState) layout.Dimensions {
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.textButton(gtx, s, "Skip") }),
		layout.Flexed(1, layout.Spacer{}.Layout),
	}
	if s.CanBack && !s.Waiting {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return nb.button(gtx, s.Back, "Back", false)
		}), layout.Rigid(layout.Spacer{Width: 8}.Layout))
	}
	if !s.Waiting {
		label := "Next"
		switch {
		case s.Index == 0:
			label = "Start the tour"
		case s.Last:
			label = "Done"
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return nb.button(gtx, s.Next, label, true)
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

func (nb *notebook) textButton(gtx layout.Context, s driver.StepState, txt string) layout.Dimensions {
	c := nb.pal.muted
	if s.Close.Hovered() {
		c = nb.pal.fg
	}
	return s.Close.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(4).Layout(gtx, nb.label(txt, 13, c, font.Medium))
	})
}

func (nb *notebook) wrapped(s string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := material.Label(nb.th, 14, s)
		l.Color, l.LineHeightScale = nb.pal.muted, 1.4
		return l.Layout(gtx)
	}
}
