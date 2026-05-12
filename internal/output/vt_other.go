//go:build !windows

package output

func enableVTProcessing(_ uintptr) bool {
	return true
}
