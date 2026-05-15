//go:build windows

package internal

// GetTermSize returns a sensible default on Windows where TIOCGWINSZ is not
// available. The QR rendering path falls back to this when running on Windows.
func GetTermSize() (cols, rows int) {
	return 80, 24
}
