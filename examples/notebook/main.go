// Command notebook is a small notes app with a guided tour over it: a
// centered welcome, a step that waits for its action, glides between
// anchors, a popover drawn by the app, and a light and a dark theme.
package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	driver "github.com/lichdotdev/gio-driver"
)

type note struct{ title, snippet string }

type notebook struct {
	th       *material.Theme
	pal      palette
	dark     bool
	notes    []note
	selected int
	created  bool

	newNote, share, help, theme widget.Clickable
	rows                        []widget.Clickable
	format                      [5]widget.Clickable

	tour driver.Tour
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Notebook"), app.Size(unit.Dp(1100), unit.Dp(700)))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	nb := newNotebook()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			nb.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func newNotebook() *notebook {
	nb := &notebook{th: material.NewTheme(), dark: true, notes: []note{
		{"Launch checklist", "Docs, changelog, the release notes and a tag."},
		{"Reading list", "Three papers on incremental layout."},
		{"Weekly sync", "Move the demo to Thursday."},
		{"Ideas", "A tour that waits for the person to act."},
	}}
	nb.tour = driver.Tour{Steps: nb.steps(), Popover: nb.coachMark}
	nb.tour.Start()
	return nb
}

func (nb *notebook) steps() []driver.Step {
	return []driver.Step{
		{Title: "Welcome to Notebook", Body: "A quick look at the essentials. It takes less than a minute."},
		{Anchor: "new", Side: driver.Right, Title: "Start a note", Body: "Every idea starts here.",
			Hint: "Click New note to continue", Until: func() bool { return nb.created }},
		{Anchor: "list", Side: driver.Right, Title: "Your notes", Body: "The newest note sits on top. Click any of them to open it."},
		{Anchor: "toolbar", Side: driver.Bottom, Title: "Format as you write", Body: "Bold, headings, lists and links, one click away."},
		{Anchor: "share", Side: driver.Bottom, Title: "Share a note", Body: "Send a read-only link to anyone, no account needed."},
		{Anchor: "help", Side: driver.Bottom, Title: "Come back any time", Body: "This button plays the tour again."},
	}
}

func (nb *notebook) frame(gtx layout.Context) {
	for {
		e, ok := nb.tour.Update(gtx)
		if !ok {
			break
		}
		log.Printf("tour ended at step %d, completed: %v", e.Step+1, e.Completed)
	}
	nb.update(gtx)
	nb.pal = palettes[nb.dark]
	nb.tour.Spotlight = nb.pal.spotlight()
	nb.tour.Layout(gtx, nb.layout)
}

func (nb *notebook) update(gtx layout.Context) {
	if nb.newNote.Clicked(gtx) {
		nb.notes = append([]note{{"Untitled", "Start writing…"}}, nb.notes...)
		nb.selected, nb.created = 0, true
	}
	if nb.theme.Clicked(gtx) {
		nb.dark = !nb.dark
	}
	if nb.help.Clicked(gtx) {
		nb.created = false
		nb.tour.Start()
	}
	nb.share.Clicked(gtx)
	for i := range nb.rows {
		if nb.rows[i].Clicked(gtx) {
			nb.selected = i
		}
	}
	for i := range nb.format {
		nb.format[i].Clicked(gtx)
	}
}
