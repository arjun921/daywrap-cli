package internal

// qr.go — renders each chunk as a QR code in the terminal at 5 fps,
// cycling indefinitely until Ctrl+C.

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	qrcode "github.com/skip2/go-qrcode"
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

// qrCapacityL is the maximum byte-mode capacity (error correction level L) for
// QR versions 1–40 (index 0 unused).
var qrCapacityL = [41]int{
	0,
	17, 32, 53, 78, 106, 134, 154, 192, 230, 271, // 1–10
	321, 367, 425, 458, 520, 586, 644, 718, 792, 858, // 11–20
	929, 1003, 1091, 1171, 1273, 1367, 1465, 1528, 1628, 1732, // 21–30
	1840, 1952, 2068, 2188, 2303, 2431, 2563, 2699, 2809, 2953, // 31–40
}

// framePrefix is the worst-case character overhead of the URL wrapper per frame:
// "daywrap://scan?d=DW:999:999:" = 28 chars.
const framePrefix = 28

// OptimalChunkSize returns the largest payload (in bytes) that produces a QR
// code fitting within the given terminal dimensions.
// It respects both width and height constraints so every frame renders without
// truncation or the need to zoom out.
func OptimalChunkSize(termCols, termRows int) int {
	// QR modules for version V: 4*V + 17.
	// With 4-module quiet zone each side: total = 4*V + 25.
	// ToSmallString: width  = 4*V + 25 chars (1 module = 1 char).
	//                height = ceil((4*V + 25) / 2) ≈ 2*V + 13 char rows.
	const statusLines = 4 // chunk counter + install URL + breathing room

	availRows := termRows - statusLines
	if availRows < 1 {
		availRows = 1
	}

	// Max QR version constrained by height: availRows >= 2*V + 13
	maxVh := (availRows - 13) / 2
	// Max QR version constrained by width: termCols >= 4*V + 25
	maxVw := (termCols - 25) / 4

	maxV := maxVh
	if maxVw < maxV {
		maxV = maxVw
	}
	if maxV < 1 {
		maxV = 1
	}
	if maxV > 40 {
		maxV = 40
	}

	payload := qrCapacityL[maxV] - framePrefix
	if payload < 10 {
		payload = 10
	}
	if payload > 2460 {
		payload = 2460
	}
	return payload
}

// equalisedChunks pads each frame with a '~' suffix (appended as '&p=~...~')
// until every chunk encodes to the same QR version as the highest-version chunk.
//
// Why '~': it is not in the QR numeric charset (0–9) or alphanumeric charset
// (0–9 A–Z SP $ % * + - . / :), so the encoder cannot optimise it to a denser
// mode. Equal-length strings in pure byte mode always produce identical QR
// versions, eliminating size-flicker during animation.
//
// The mobile parser uses /[?&]d=(DW:[^&\s]+)/ which stops at '&', so the
// '&p=~...' suffix is never seen by the decoder.
func equalisedChunks(chunks []string) ([]string, error) {
	if len(chunks) <= 1 {
		return chunks, nil
	}

	qrVersion := func(content string) (int, error) {
		qr, err := qrcode.New(content, qrcode.Low)
		if err != nil {
			return 0, err
		}
		b := qr.Bitmap()
		return (len(b) - 17) / 4, nil
	}

	// First pass: find the maximum QR version across all chunks.
	maxVer := 0
	for i, c := range chunks {
		v, err := qrVersion(c)
		if err != nil {
			return nil, fmt.Errorf("qr equalise probe chunk %d: %w", i+1, err)
		}
		if v > maxVer {
			maxVer = v
		}
	}

	// Second pass: pad under-version chunks with '~' until they reach maxVer.
	out := make([]string, len(chunks))
	const maxIter = 4000
	for i, c := range chunks {
		v, err := qrVersion(c)
		if err != nil {
			return nil, fmt.Errorf("qr equalise chunk %d: %w", i+1, err)
		}
		if v >= maxVer {
			out[i] = c
			continue
		}
		padded := c + "&p="
		for iter := 0; iter < maxIter; iter++ {
			v, err = qrVersion(padded)
			if err != nil {
				return nil, fmt.Errorf("qr equalise pad chunk %d: %w", i+1, err)
			}
			if v >= maxVer {
				break
			}
			padded += "~"
		}
		out[i] = padded
	}
	return out, nil
}

// Display renders chunks as an animated QR sequence in the terminal.
// It cycles through all chunks at 5 fps until the user presses Ctrl+C.
func Display(chunks []string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	// Equalise QR versions so every frame renders at the same pixel size.
	equalised, err := equalisedChunks(chunks)
	if err != nil {
		return err
	}

	// Hide cursor and do a single full clear at startup.
	// This is the only time we blank the screen; subsequent frames overwrite in place.
	fmt.Print("\033[?25l\033[2J\033[H")
	defer func() {
		// Erase screen and restore cursor on exit.
		fmt.Print("\033[2J\033[H\033[?25h")
	}()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	total := len(equalised)
	idx := 0

	if err := renderFrame(equalised[idx], idx+1, total); err != nil {
		return err
	}
	idx = (idx + 1) % total

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			if err := renderFrame(equalised[idx], idx+1, total); err != nil {
				return err
			}
			idx = (idx + 1) % total
		}
	}
}

// renderFrame overwrites the current terminal content with a new QR frame.
// It checks that the QR fits; if the terminal is too small it shows a resize
// hint instead of a garbled partial code.
func renderFrame(chunk string, index, total int) error {
	qr, err := qrcode.New(chunk, qrcode.Low)
	if err != nil {
		return fmt.Errorf("qr encode: %w", err)
	}

	termCols, termRows := GetTermSize()
	bitmap := qr.Bitmap()
	qrCharRows := (len(bitmap) + 1) / 2
	qrCharCols := len(bitmap[0])

	var buf bytes.Buffer
	// Move cursor to top-left — no clear, so there is no blank flash.
	buf.WriteString("\033[H")

	if qrCharRows+4 > termRows || qrCharCols > termCols {
		// Terminal shrank after chunking — show a helpful resize hint.
		fmt.Fprintf(&buf, "  Terminal too small for this QR code (chunk %d/%d).\n", index, total)
		fmt.Fprintf(&buf, "  Resize to at least %d cols × %d rows\n", qrCharCols, qrCharRows+4)
		fmt.Fprintf(&buf, "  (current: %d cols × %d rows)\n", termCols, termRows)
	} else {
		buf.WriteString(qr.ToSmallString(false))
		buf.WriteByte('\n')
		fmt.Fprintf(&buf, "  Chunk %d / %d   │   Scan with DayWrap   │   Ctrl+C to stop\n", index, total)
		buf.WriteString("  Not installed? Get it at: daywr.app\n")
	}
	// Erase from cursor to end of screen so no stale lines remain.
	buf.WriteString("\033[J")

	_, err = os.Stdout.Write(buf.Bytes())
	return err
}


