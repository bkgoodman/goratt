package reader

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tarm/serial"
)

// Serial implements TagReader for serial RFID readers using a custom protocol.
// Protocol: [0x02][0x09][data...][checksum][0x03]
type Serial struct {
	port   *serial.Port
	device string
}

// NewSerial creates a new serial RFID reader.
func NewSerial(device string) (*Serial, error) {
	c := &serial.Config{
		Name:        device,
		Baud:        115200,
		ReadTimeout: time.Second,
	}
	port, err := serial.OpenPort(c)
	if err != nil {
		return nil, fmt.Errorf("open serial %s: %w", device, err)
	}

	return &Serial{port: port, device: device}, nil
}

// Read implements TagReader.Read for serial readers.
func (s *Serial) Read(ctx context.Context) (uint64, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		tag, err := s.readFrame()
		if err != nil {
			return 0, err
		}
		if tag != 0 {
			return tag, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Serial) readFrame() (uint64, error) {
	buff := make([]byte, 9)

	n, err := s.port.Read(buff)
	if err != nil {
		return 0, nil // Timeout, try again
	}
	if n == 0 {
		return 0, nil
	}
	if n != 9 {
		return 0, nil // Partial read
	}

	preambles := []byte{0x02, 0x09}
	terminator := []byte{0x03}

	if !bytes.Equal(buff[0:2], preambles) {
		return 0, nil
	}

	if !bytes.Equal(buff[8:9], terminator) {
		return 0, nil
	}

	data := buff[1:7]
	xor := data[0]
	for i := 1; i < len(data); i++ {
		xor ^= data[i]
	}

	tagno := (uint64(data[2]) << 24) | (uint64(data[3]) << 16) | (uint64(data[4]) << 8) | uint64(data[5])

	if xor != buff[7] {
		return 0, nil // Checksum mismatch
	}

	return tagno, nil
}

// Close implements TagReader.Close.
func (s *Serial) Close() error {
	if s.port == nil {
		return nil
	}
	return s.port.Close()
}
