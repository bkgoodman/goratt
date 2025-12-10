package wiegland

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
)

const (
	STX = 0x02
	ETX = 0x03
)

// RFIDReader encapsulates the serial port.
type RFIDReader struct {
	port serial.Port
}

// Initialize opens the serial port with the given baud rate.
// Equivalent to Python's serial.Serial(...); close(); open().
func (r *RFIDReader) Initialize(serialPort string, baudRate int) error {
	mode := &serial.Mode{
		BaudRate: baudRate,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	fmt.Printf("port %s %d baud\n", serialPort, baudRate)

	p, err := serial.Open(serialPort, mode)
	if err != nil {
		return fmt.Errorf("open serial: %w", err)
	}

	// Set a small read timeout so GetCard can return quickly if no data.
	if err := p.SetReadTimeout(50 * time.Millisecond); err != nil {
		// Not all drivers support this; it's okay to continue if it fails.
		// We'll still try to operate.
	}

	// Mimic the Python behaviour (close, then open) if desired:
	// (Not strictly necessary in Go; the handle we have is open.)
	// Just store and flush input.
	r.port = p
	r.flush()
	return nil
}

// Close closes the serial port.
func (r *RFIDReader) Close() error {
	if r.port == nil {
		return nil
	}
	return r.port.Close()
}

// GetCard reads a single card frame starting with STX and ending with ETX.
// On no data available, returns ("", nil). On parse issues, returns an error.
func (r *RFIDReader) GetCard() (uint64, error) {
	if r.port == nil {
		return 0, errors.New("port not initialized")
	}

	// Attempt to read one byte; if timeout (0 bytes), treat as no data.
	first := make([]byte, 1)
	n, err := r.port.Read(first)
	if err != nil {
		return 0, fmt.Errorf("read STX: %w", err)
	}
	if n == 0 {
		// No data available (timeout); mirror Python's None -> empty string here.
		return 0, nil
	}

	// Look for STX
	if first[0] != STX {
		// Noise or partial frame; flush and return none.
		r.flush()
		return 0, nil
	}

	// Read until ETX, building ASCII ID characters.
	var idBuilder strings.Builder
	buf := make([]byte, 1)

	for {
		n, err := r.port.Read(buf)
		if err != nil {
			return 0, fmt.Errorf("read body: %w", err)
		}
		if n == 0 {
			// Timeout mid-frame; treat as incomplete frame -> flush and return none.
			r.flush()
			return 0, nil
		}
		b := buf[0]
		if b == ETX {
			break
		}
		// Append literal character (Python used '%c' % buf[0])
		idBuilder.WriteByte(b)
	}

	ID := idBuilder.String()

	// Left-pad ID to length 10, same as Python loop `while len(ID) < 10`
	for len(ID) < 10 {
		ID = "0" + ID
	}

	fmt.Println(ID)

	// Compute checksum as XOR of each byte formed from hex pairs (positions 0..8 step 2)
	var checksum byte
	for i := 0; i <= 8; i += 2 {
		hi, err := hexCharToNibble(ID[i])
		if err != nil {
			return 0, fmt.Errorf("invalid hex at pos %d: %w", i, err)
		}
		lo, err := hexCharToNibble(ID[i+1])
		if err != nil {
			return 0, fmt.Errorf("invalid hex at pos %d: %w", i+1, err)
		}
		val := byte((hi << 4) | lo)
		checksum ^= val
	}
	checksumHex := fmt.Sprintf("0x%X", checksum)

	// Tag formed from digits 1,2,3: ((ID[1]<<8) + (ID[2]<<4) + (ID[3]<<0))
	d1, err := hexCharToNibble(ID[1])
	if err != nil {
		return 0, fmt.Errorf("invalid tag nibble 1: %w", err)
	}
	d2, err := hexCharToNibble(ID[2])
	if err != nil {
		return 0, fmt.Errorf("invalid tag nibble 2: %w", err)
	}
	d3, err := hexCharToNibble(ID[3])
	if err != nil {
		return 0, fmt.Errorf("invalid tag nibble 3: %w", err)
	}
	tagVal := (d1 << 8) + (d2 << 4) + d3
	tagHex := fmt.Sprintf("0x%X", tagVal)

	// Card is the decimal of hex substring ID[4:10]
	if len(ID) < 10 {
		return 0, fmt.Errorf("ID length < 10 after padding? got %d", len(ID))
	}
	cardHex := ID[4:10]
	cardInt, err := strconv.ParseUint(cardHex, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse card hex %q: %w", cardHex, err)
	}

	fmt.Println("------------------------------------------")
	fmt.Println("Data: ", ID)
	fmt.Println("Tag: ", tagHex)
	fmt.Println("ID: ", cardHex, " - ", cardInt)
	fmt.Println("Checksum: ", checksumHex)
	fmt.Println("------------------------------------------")

	return cardInt, nil
}

// flush drains the input buffer to discard any partial frames.
func (r *RFIDReader) flush() {
	if r.port == nil {
		return
	}
	// Temporarily use a very short timeout so flush returns quickly.
	_ = r.port.SetReadTimeout(10 * time.Millisecond)
	defer func() {
		_ = r.port.SetReadTimeout(50 * time.Millisecond)
	}()

	tmp := make([]byte, 64)
	for {
		n, err := r.port.Read(tmp)
		if err != nil || n == 0 {
			// Either timeout or error; stop flushing. Errors here are non-fatal.
			return
		}
		// Continue draining until timeout gives 0.
	}
}

// hexCharToNibble converts a single hex character to its numeric value (0..15).
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
