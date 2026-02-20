package output

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// SelectOption represents a single option in an interactive select prompt.
type SelectOption struct {
	Label string
	Value string
}

// Select shows an interactive selection prompt. Returns an error in
// non-interactive mode (CI or piped output).
func (w *Writer) Select(title string, options []SelectOption) (string, error) {
	if !w.interactive {
		return "", fmt.Errorf("cannot prompt for selection in non-interactive mode")
	}

	huhOpts := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOpts[i] = huh.NewOption(opt.Label, opt.Value)
	}

	var value string
	err := huh.NewSelect[string]().
		Title(title).
		Options(huhOpts...).
		Value(&value).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection prompt failed: %w", err)
	}

	return value, nil
}

// Input shows an interactive free-text input prompt. Returns an error in
// non-interactive mode (CI or piped output).
func (w *Writer) Input(title, placeholder string) (string, error) {
	if !w.interactive {
		return "", fmt.Errorf("cannot prompt for input in non-interactive mode")
	}

	var value string
	err := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&value).
		Run()
	if err != nil {
		return "", fmt.Errorf("input prompt failed: %w", err)
	}

	return value, nil
}

// SecureInput shows an interactive input prompt with masked characters.
// Use this for sensitive values like API tokens. Returns an error in
// non-interactive mode (CI or piped output).
func (w *Writer) SecureInput(title, placeholder string) (string, error) {
	if !w.interactive {
		return "", fmt.Errorf("cannot prompt for input in non-interactive mode")
	}

	var value string
	err := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		Run()
	if err != nil {
		return "", fmt.Errorf("input prompt failed: %w", err)
	}

	return value, nil
}
