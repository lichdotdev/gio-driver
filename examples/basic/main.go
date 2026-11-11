// Command basic is a three-step tour over a small window: the first step
// waits for a click on its button, the others move on with Next.
package main

import (
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	driver "github.com/lichdotdev/gio-driver"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("gio-driver"), app.Size(unit.Dp(900), unit.Dp(560)))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	th := material.NewTheme()
	var create, save, start widget.Clickable
	created := false
	tour := driver.Tour{
		Steps: []driver.Step{
			{Anchor: "create", Title: "Create a document", Body: "Everything starts with a document.", Hint: "Click Create to continue", Until: func() bool { return created }},
			{Anchor: "save", Side: driver.Right, Title: "Save often", Body: "Your work is kept only once it is saved."},
			{Anchor: "start", Side: driver.Left, Title: "Come back any time", Body: "This button runs the tour again."},
		},
		Popover: driver.DefaultPopover(driver.PopoverStyle{Theme: th}),
	}
	tour.Start()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			for {
				end, ok := tour.Update(gtx)
				if !ok {
					break
				}
				log.Printf("tour ended at step %d, completed %v", end.Step, end.Completed)
			}
			if create.Clicked(gtx) {
				created = true
			}
			if start.Clicked(gtx) {
				created = false
				tour.Start()
			}
			save.Clicked(gtx)
			tour.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				paint.Fill(gtx.Ops, color.NRGBA{R: 0xf4, G: 0xf4, B: 0xf5, A: 0xff})
				at(gtx, image.Pt(40, 40), func(gtx layout.Context) layout.Dimensions {
					return tour.Anchor(gtx, "create", material.Button(th, &create, "Create").Layout)
				})
				at(gtx, image.Pt(40, 200), func(gtx layout.Context) layout.Dimensions {
					return tour.Anchor(gtx, "save", material.Button(th, &save, "Save").Layout)
				})
				at(gtx, image.Pt(gtx.Constraints.Max.X-gtx.Dp(120), 40), func(gtx layout.Context) layout.Dimensions {
					return tour.Anchor(gtx, "start", material.Button(th, &start, "Tour").Layout)
				})
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
			e.Frame(gtx.Ops)
		}
	}
}

func at(gtx layout.Context, p image.Point, w layout.Widget) {
	defer op.Offset(p).Push(gtx.Ops).Pop()
	gtx.Constraints.Min = image.Point{}
	w(gtx)
}
