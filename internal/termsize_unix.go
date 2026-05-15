//go:build !windows

package internal

import (
	"os"
	"syscall"
	"unsafe"
)

// winsize mirrors the TIOCGWINSZ kernel struct.
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// GetTermSize returns the current terminal dimensions, defaulting to 80×24
// when stdout is not a TTY or the ioctl call fails.
func GetTermSize() (cols, rows int) {
	ws := new(winsize)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(ws)),
	)
	if errno != 0 || ws.Col == 0 || ws.Row == 0 {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}
