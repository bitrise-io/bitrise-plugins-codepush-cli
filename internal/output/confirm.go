package output

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// ConfirmDestructive asks for confirmation before a destructive operation.
// If yesFlag is true (user passed --yes), it proceeds without asking.
// In non-interactive mode without --yes, it returns an error with a hint.
// In interactive mode without --yes, it shows a y/N prompt.
func (w *Writer) ConfirmDestructive(msg string, yesFlag bool) error {
	if yesFlag {
		return nil
	}

	if !w.interactive {
		return fmt.Errorf("%s; use --yes to confirm", msg)
	}

	w.Warning("%s", msg)

	var confirmed bool
	err := huh.NewConfirm().
		Title("Continue?").
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()
	if err != nil {
		return fmt.Errorf("confirmation prompt failed: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("cancelled by user")
	}

	return nil
}
