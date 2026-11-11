package driver

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Labels are the default popover's words, for the application to translate.
type Labels struct {
	Next, Back, Done, Close string
	// Progress names where a step stands, index from 0.
	Progress func(index, total int) string
}

// PopoverStyle is DefaultPopover's look. Zero colors, sizes and labels take
// the defaults.
type PopoverStyle struct {
	Theme                           *material.Theme
	Background, Border, Title, Body color.NRGBA
	Width, Padding, Radius          unit.Dp
	Labels                          Labels
}

var defaultPopover = PopoverStyle{
	Background: color.NRGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff},
	Border:     color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a},
	Title:      color.NRGBA{R: 0xfa, G: 0xfa, B: 0xfa, A: 0xff},
	Body:       color.NRGBA{R: 0xa1, G: 0xa1, B: 0xaa, A: 0xff},
	Width:      320, Padding: 16, Radius: 8,
	Labels: Labels{Next: "Next", Back: "Back", Done: "Done", Close: "Skip tour",
		Progress: func(i, n int) string { return fmt.Sprintf("%d of %d", i+1, n) }},
}

// DefaultPopover is a plain popover drawn with st.Theme's buttons: title,
// body, hint, progress, and Close, Back and Next as the step allows.
func DefaultPopover(st PopoverStyle) func(gtx layout.Context, s StepState) layout.Dimensions {
	st = st.withDefaults()
	return func(gtx layout.Context, s StepState) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(st.Width)
		gtx.Constraints.Max.X = gtx.Constraints.Min.X
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions { return st.card(gtx) }),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(st.Padding).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return st.content(gtx, s)
				})
			}),
		)
	}
}

func (st PopoverStyle) withDefaults() PopoverStyle {
	d := defaultPopover
	pick := func(c *color.NRGBA, def color.NRGBA) {
		if *c == (color.NRGBA{}) {
			*c = def
		}
	}
	pick(&st.Background, d.Background)
	pick(&st.Border, d.Border)
	pick(&st.Title, d.Title)
	pick(&st.Body, d.Body)
	for _, p := range []struct {
		v   *unit.Dp
		def unit.Dp
	}{{&st.Width, d.Width}, {&st.Padding, d.Padding}, {&st.Radius, d.Radius}} {
		if *p.v == 0 {
			*p.v = p.def
		}
	}
	for _, p := range []struct {
		v   *string
		def string
	}{{&st.Labels.Next, d.Labels.Next}, {&st.Labels.Back, d.Labels.Back}, {&st.Labels.Done, d.Labels.Done}, {&st.Labels.Close, d.Labels.Close}} {
		if *p.v == "" {
			*p.v = p.def
		}
	}
	if st.Labels.Progress == nil {
		st.Labels.Progress = d.Labels.Progress
	}
	return st
}

func (st PopoverStyle) card(gtx layout.Context) layout.Dimensions {
	size, r := gtx.Constraints.Min, gtx.Dp(st.Radius)
	paint.FillShape(gtx.Ops, st.Border, clip.UniformRRect(image.Rectangle{Max: size}, r).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, st.Background, clip.UniformRRect(image.Rectangle{Max: size}.Inset(1), r-1).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

func (st PopoverStyle) content(gtx layout.Context, s StepState) layout.Dimensions {
	text := func(txt string, c color.NRGBA, w font.Weight) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(st.Theme, txt)
			l.Color, l.Font.Weight = c, w
			return l.Layout(gtx)
		})
	}
	gap := layout.Rigid(layout.Spacer{Height: 8}.Layout)
	rows := []layout.FlexChild{text(s.Step.Title, st.Title, font.SemiBold), gap, text(s.Step.Body, st.Body, font.Normal)}
	if s.Step.Hint != "" {
		rows = append(rows, gap, text(s.Step.Hint, st.Title, font.Medium))
	}
	rows = append(rows, layout.Rigid(layout.Spacer{Height: 12}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return st.actions(gtx, s)
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (st PopoverStyle) actions(gtx layout.Context, s StepState) layout.Dimensions {
	progress := material.Caption(st.Theme, st.Labels.Progress(s.Index, s.Total))
	progress.Color = st.Body
	children := []layout.FlexChild{layout.Flexed(1, progress.Layout)}
	button := func(b material.ButtonStyle) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: 4}.Layout(gtx, b.Layout)
		})
	}
	quiet := func(b material.ButtonStyle) material.ButtonStyle {
		b.Background, b.Color = color.NRGBA{}, st.Title
		return b
	}
	children = append(children, button(quiet(material.Button(st.Theme, s.Close, st.Labels.Close))))
	if s.CanBack && !s.Waiting {
		children = append(children, button(quiet(material.Button(st.Theme, s.Back, st.Labels.Back))))
	}
	if !s.Waiting {
		label := st.Labels.Next
		if s.Last {
			label = st.Labels.Done
		}
		children = append(children, button(material.Button(st.Theme, s.Next, label)))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}
