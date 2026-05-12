//go:build windows

package output

import "golang.org/x/sys/windows"

// enableVTProcessing attempts to enable VT100 processing on the given console
// handle. Returns true on success, false if the console does not support VT
// sequences (e.g. classic cmd.exe without VT mode enabled).
func enableVTProcessing(fd uintptr) bool {
	var mode uint32
	if err := windows.GetConsoleMode(windows.Handle(fd), &mode); err != nil {
		return false
	}
	return windows.SetConsoleMode(windows.Handle(fd), mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
