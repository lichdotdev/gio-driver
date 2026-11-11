package main

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	driver "github.com/lichdotdev/gio-driver"
)

type palette struct {
	canvas, panel, raised, border color.NRGBA
	fg, muted, accent, accentFg   color.NRGBA
	accentSoft, card, scrim       color.NRGBA
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

var palettes = map[bool]palette{
	true: {
		canvas: rgb(0x0b0b0e), panel: rgb(0x121216), raised: rgb(0x1c1c21), border: rgb(0x26262c),
		fg: rgb(0xf4f4f5), muted: rgb(0x9a9aa6), accent: rgb(0x3b82f6), accentFg: rgb(0xffffff),
		accentSoft: alpha(rgb(0x3b82f6), 0x2e), card: rgb(0x18181d), scrim: color.NRGBA{A: 0xa8},
	},
	false: {
		canvas: rgb(0xf4f4f6), panel: rgb(0xffffff), raised: rgb(0xf0f0f3), border: rgb(0xe3e3e8),
		fg: rgb(0x18181b), muted: rgb(0x6b6b76), accent: rgb(0x2563eb), accentFg: rgb(0xffffff),
		accentSoft: alpha(rgb(0x2563eb), 0x1c), card: rgb(0xffffff), scrim: color.NRGBA{R: 0x0b, G: 0x0b, B: 0x10, A: 0x80},
	},
}

func (p palette) spotlight() driver.Spotlight {
	return driver.Spotlight{Scrim: p.scrim, Padding: 6, Radius: 10, Offset: 12, Margin: 12}
}

func (nb *notebook) layout(gtx layout.Context) layout.Dimensions {
	paint.Fill(gtx.Ops, nb.pal.canvas)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(nb.topBar),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx, layout.Rigid(nb.sidebar), layout.Flexed(1, nb.editor))
		}),
		layout.Rigid(nb.statusBar),
	)
}

func (nb *notebook) topBar(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = gtx.Dp(56), gtx.Dp(56)
	gap := layout.Rigid(layout.Spacer{Width: 8}.Layout)
	return layout.Inset{Left: 20, Right: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(nb.logo),
			layout.Rigid(layout.Spacer{Width: 40}.Layout),
			layout.Rigid(nb.search),
			layout.Flexed(1, layout.Spacer{}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return nb.button(gtx, &nb.theme, map[bool]string{true: "Light", false: "Dark"}[nb.dark], false)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return nb.tour.Anchor(gtx, "help", func(gtx layout.Context) layout.Dimensions { return nb.button(gtx, &nb.help, "?", false) })
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return nb.tour.Anchor(gtx, "share", func(gtx layout.Context) layout.Dimensions { return nb.button(gtx, &nb.share, "Share", true) })
			}),
		)
	})
}

func (nb *notebook) logo(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = 0
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return box(gtx, image.Pt(gtx.Dp(22), gtx.Dp(22)), nb.pal.accent, gtx.Dp(6))
		}),
		layout.Rigid(layout.Spacer{Width: 10}.Layout),
		layout.Rigid(nb.label("Notebook", 16, nb.pal.fg, font.Bold)),
	)
}

func (nb *notebook) search(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Dp(320), gtx.Dp(34))
	return layout.Stack{Alignment: layout.W}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return outlined(gtx, size, nb.pal.raised, nb.pal.border, gtx.Dp(8))
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: 12}.Layout(gtx, nb.label("Search notes", 13, nb.pal.muted, font.Normal))
		}),
	)
}

func (nb *notebook) sidebar(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(272)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	return layout.Inset{Left: 12, Bottom: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return panel(gtx, nb.pal, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return nb.tour.Anchor(gtx, "new", func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return nb.button(gtx, &nb.newNote, "+  New note", true)
						})
					}),
					layout.Rigid(layout.Spacer{Height: 20}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: 10, Bottom: 8}.Layout(gtx, nb.label("NOTES", 11, nb.pal.muted, font.SemiBold))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.tour.Anchor(gtx, "list", nb.noteList) }),
				)
			})
		})
	})
}

func (nb *notebook) noteList(gtx layout.Context) layout.Dimensions {
	for len(nb.rows) < len(nb.notes) {
		nb.rows = append(nb.rows, widget.Clickable{})
	}
	children := make([]layout.FlexChild, len(nb.notes))
	for i, n := range nb.notes {
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.noteRow(gtx, i, n) })
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (nb *notebook) noteRow(gtx layout.Context, i int, n note) layout.Dimensions {
	click := &nb.rows[i]
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				switch {
				case i == nb.selected:
					return box(gtx, gtx.Constraints.Min, nb.pal.accentSoft, gtx.Dp(8))
				case click.Hovered():
					return box(gtx, gtx.Constraints.Min, nb.pal.raised, gtx.Dp(8))
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(nb.label(n.title, 14, nb.pal.fg, font.SemiBold)),
						layout.Rigid(layout.Spacer{Height: 2}.Layout),
						layout.Rigid(nb.label(n.snippet, 12, nb.pal.muted, font.Normal)),
					)
				})
			}),
		)
	})
}

var formats = [5]string{"B", "I", "H1", "List", "Link"}

func (nb *notebook) editor(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: 12, Right: 12, Bottom: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return panel(gtx, nb.pal, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 20, Left: 40, Right: 40}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				n := nb.notes[nb.selected]
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.tour.Anchor(gtx, "toolbar", nb.toolbar) }),
					layout.Rigid(layout.Spacer{Height: 32}.Layout),
					layout.Rigid(nb.label(n.title, 30, nb.pal.fg, font.Bold)),
					layout.Rigid(layout.Spacer{Height: 6}.Layout),
					layout.Rigid(nb.label("Edited just now", 13, nb.pal.muted, font.Normal)),
					layout.Rigid(layout.Spacer{Height: 24}.Layout),
					layout.Rigid(nb.paragraph(n.snippet+" The rest of the note goes here, with room to think in full sentences and come back to them later.")),
					layout.Rigid(layout.Spacer{Height: 14}.Layout),
					layout.Rigid(nb.paragraph("Notes save as you type. Nothing to remember, nothing to lose.")),
				)
			})
		})
	})
}

func (nb *notebook) toolbar(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0, 2*len(formats))
	for i, f := range formats {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return nb.button(gtx, &nb.format[i], f, false) }))
		children = append(children, layout.Rigid(layout.Spacer{Width: 6}.Layout))
	}
	return layout.Flex{}.Layout(gtx, children[:len(children)-1]...)
}

func (nb *notebook) statusBar(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: 24, Right: 24, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Flexed(1, nb.label(itoa(len(nb.notes))+" notes", 12, nb.pal.muted, font.Normal)),
			layout.Rigid(nb.label("Saved", 12, nb.pal.muted, font.Normal)),
		)
	})
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func (nb *notebook) paragraph(s string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := material.Label(nb.th, 15, s)
		l.Color, l.LineHeightScale = nb.pal.fg, 1.5
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(620))
		return l.Layout(gtx)
	}
}

func (nb *notebook) label(s string, size unit.Sp, c color.NRGBA, w font.Weight) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := material.Label(nb.th, size, s)
		l.Color, l.Font.Weight, l.MaxLines, l.Truncator = c, w, 1, "…"
		return l.Layout(gtx)
	}
}

// button is a pill of text; primary paints it in the accent.
func (nb *notebook) button(gtx layout.Context, click *widget.Clickable, txt string, primary bool) layout.Dimensions {
	bg, fg, edge := nb.pal.raised, nb.pal.fg, nb.pal.border
	if primary {
		bg, fg, edge = nb.pal.accent, nb.pal.accentFg, nb.pal.accent
	}
	if click.Hovered() {
		bg = mix(bg, fg, 0.08)
	}
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return outlined(gtx, gtx.Constraints.Min, bg, edge, gtx.Dp(8))
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 8, Bottom: 8, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(nb.th, 13, txt)
					l.Color, l.Font.Weight, l.Alignment = fg, font.Medium, text.Middle
					return l.Layout(gtx)
				})
			}),
		)
	})
}

func panel(gtx layout.Context, p palette, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return outlined(gtx, gtx.Constraints.Max, p.panel, p.border, gtx.Dp(12))
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return w(gtx)
		}),
	)
}

func box(gtx layout.Context, size image.Point, c color.NRGBA, r int) layout.Dimensions {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(image.Rectangle{Max: size}, r).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

func outlined(gtx layout.Context, size image.Point, fill, edge color.NRGBA, r int) layout.Dimensions {
	box(gtx, size, edge, r)
	paint.FillShape(gtx.Ops, fill, clip.UniformRRect(image.Rectangle{Max: size}.Inset(1), r-1).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

func mix(a, b color.NRGBA, t float32) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return color.NRGBA{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: a.A}
}
