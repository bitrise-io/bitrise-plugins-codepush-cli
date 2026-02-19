package output

import (
	"github.com/charmbracelet/huh/spinner"
)

// Spinner runs an action while displaying a spinner animation.
// In non-interactive mode (CI, piped), it prints the title as a step
// and runs the action without animation.
func (w *Writer) Spinner(title string, action func() error) error {
	if !w.interactive {
		w.Step("%s...", title)
		return action()
	}

	var actionErr error
	if err := spinner.New().
		Title(" " + title + "...").
		Action(func() {
			actionErr = action()
		}).
		Run(); err != nil {
		return err
	}

	return actionErr
}
