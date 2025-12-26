package reader

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
)

const (
	stx = 0x02
	etx = 0x03
)

// Wiegand implements TagReader for Wiegand-style serial RFID readers.
type Wiegand struct {
	port serial.Port
}

// NewWiegand creates a new Wiegand reader on the specified serial port.
func NewWiegand(device string, baud int) (*Wiegand, error) {
	if baud == 0 {
		baud = 9600
	}

	mode := &serial.Mode{
		BaudRate: baud,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	p, err := serial.Open(device, mode)
	if err != nil {
		return nil, fmt.Errorf("open serial %s: %w", device, err)
	}

	_ = p.SetReadTimeout(50 * time.Millisecond)

	w := &Wiegand{port: p}
	w.flush()
	return w, nil
}

// Read implements TagReader.Read for Wiegand readers.
func (w *Wiegand) Read(ctx context.Context) (uint64, error) {
	if w.port == nil {
		return 0, errors.New("port not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		tag, err := w.readFrame()
		if err != nil {
			return 0, err
		}
		if tag != 0 {
			return tag, nil
		}
		// No data, brief sleep before retry
		time.Sleep(100 * time.Millisecond)
	}
}

// readFrame attempts to read a single card frame.
func (w *Wiegand) readFrame() (uint64, error) {
	first := make([]byte, 1)
	n, err := w.port.Read(first)
	if err != nil {
		return 0, fmt.Errorf("read STX: %w", err)
	}
	if n == 0 {
		return 0, nil
	}

	if first[0] != stx {
		w.flush()
		return 0, nil
	}

	var idBuilder strings.Builder
	buf := make([]byte, 1)

	for {
		n, err := w.port.Read(buf)
		if err != nil {
			return 0, fmt.Errorf("read body: %w", err)
		}
		if n == 0 {
			w.flush()
			return 0, nil
		}
		if buf[0] == etx {
			break
		}
		idBuilder.WriteByte(buf[0])
	}

	id := idBuilder.String()
	for len(id) < 10 {
		id = "0" + id
	}

	// Compute checksum
	var checksum byte
	for i := 0; i <= 8; i += 2 {
		hi, err := hexCharToNibble(id[i])
		if err != nil {
			return 0, fmt.Errorf("invalid hex at pos %d: %w", i, err)
		}
		lo, err := hexCharToNibble(id[i+1])
		if err != nil {
			return 0, fmt.Errorf("invalid hex at pos %d: %w", i+1, err)
		}
		checksum ^= byte((hi << 4) | lo)
	}

	// Parse card ID from hex substring
	if len(id) < 10 {
		return 0, fmt.Errorf("ID length < 10 after padding")
	}
	cardHex := id[4:10]
	cardInt, err := strconv.ParseUint(cardHex, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse card hex %q: %w", cardHex, err)
	}

	return cardInt, nil
}

// Close implements TagReader.Close.
func (w *Wiegand) Close() error {
	if w.port == nil {
		return nil
	}
	return w.port.Close()
}

func (w *Wiegand) flush() {
	if w.port == nil {
		return
	}
	_ = w.port.SetReadTimeout(10 * time.Millisecond)
	defer func() {
		_ = w.port.SetReadTimeout(50 * time.Millisecond)
	}()

	tmp := make([]byte, 64)
	for {
		n, err := w.port.Read(tmp)
		if err != nil || n == 0 {
			return
		}
	}
}

func hexCharToNibble(c byte) (int, error) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), nil
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10, nil
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10, nil
	default:
		return 0, fmt.Errorf("not a hex char: %q", c)
	}
}
