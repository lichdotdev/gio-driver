# gio-driver

Guided product tours for [Gio](https://gioui.org) applications, in the spirit of
[driver.js](https://driverjs.com): a scrim with a hole over one widget at a time, and a
popover beside it that walks the person through the steps.

It is built for [lich](https://github.com/lichdotdev/lich), its first user, and works on
plain `gioui.org`.

```sh
go get github.com/lichdotdev/gio-driver
```

## Use

Wrap the window's content with `Tour.Layout`, mark each widget a step points at with
`Tour.Anchor`, and read the end of the tour with `Tour.Update`:

```go
tour := driver.Tour{
	Steps: []driver.Step{
		{Anchor: "open", Title: "Open a project", Body: "Pick a folder to work in.",
			Hint: "Click Open to continue", Until: func() bool { return opened }},
		{Anchor: "sessions", Side: driver.Right, Title: "Your sessions", Body: "Each one gets a row."},
	},
	Popover: driver.DefaultPopover(driver.PopoverStyle{Theme: th}),
}
tour.Start()

// Every frame:
for {
	e, ok := tour.Update(gtx)
	if !ok {
		break
	}
	saveSeen(e.Completed)
}
tour.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
	// ...
	tour.Anchor(gtx, "open", openButton.Layout)
	// ...
})
```

`examples/basic` is a runnable window: `go run ./examples/basic`.

## How it behaves

- **Clicks pass through the hole** and are blocked everywhere else, scroll included.
- **`Until` holds a step** until the person does its action through the hole: the popover
  shows no Next, and Back is not offered into a step that cannot be undone.
- **The popover flips and shifts** to stay inside the window, like a tooltip.
- **Escape closes the tour** unless a focused widget takes it first, so a focused editor
  or terminal keeps its own Escape.
- **A step whose anchor is not on screen** shows the popover in the middle of the window
  over a full scrim.

## Your design system

`Spotlight` styles the scrim and the hole. The popover is yours: `DefaultPopover` takes
colors, sizes and translated labels, and `Tour.Popover` can be any widget built from
`StepState`, which hands over the step, its position in the tour and the clickables for
Next, Back and Close.

```go
tour.Popover = func(gtx layout.Context, s driver.StepState) layout.Dimensions {
	return myCoachMark(gtx, s.Step.Title, s.Step.Body, s.Index, s.Total, s.Next, s.Back, s.Close)
}
```

## Limits

- Gio does not tell a widget where it is in the window. The tour learns it from the
  pointer, so until the pointer has been inside the window, the popover waits in the
  middle of the window, with the hole already over its anchor, and moves beside the
  anchor at the first pointer event. A window that resizes under a still pointer keeps
  the old position until the pointer moves.
- The hole is not clipped by the anchor's parent: a widget half scrolled out of a list
  gets a hole over its full size. Scroll the anchor into view before its step.
- Gio blends in linear light, so a scrim's alpha reads lighter than the same value in a
  browser. Pick `Spotlight.Scrim` by eye.

## License

MIT
