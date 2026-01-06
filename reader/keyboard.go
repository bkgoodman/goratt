package reader

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/kenshaw/evdev"
)

// Keyboard implements TagReader for USB keyboard-style RFID readers
// that output digits followed by Enter.
type Keyboard struct {
	device    *evdev.Evdev
	numDigits int  // expected number of digits (0 = any)
	isHex     bool // true for hex input, false for decimal
	format    string
}

// NewKeyboard creates a new keyboard reader on the specified input device.
// Format specifies the input format: "10h" (10 hex digits), "10d" (10 decimal), "8h", "8d", etc.
// If format is empty, defaults to "10h" for backwards compatibility.
func NewKeyboard(device string, format string) (*Keyboard, error) {
	dev, err := evdev.OpenFile(device)
	if err != nil {
		return nil, fmt.Errorf("open evdev %s: %w", device, err)
	}

	log.Printf("Opened keyboard device: %s", dev.Name())
	log.Printf("Vendor: 0x%04x, Product: 0x%04x", dev.ID().Vendor, dev.ID().Product)

	// Parse format string (e.g., "10h", "10d", "8h", "8d")
	numDigits := 0
	isHex := true
	if format == "" {
		format = "10h"
	}
	format = strings.ToLower(format)

	if strings.HasSuffix(format, "h") {
		isHex = true
		numDigits, _ = strconv.Atoi(strings.TrimSuffix(format, "h"))
	} else if strings.HasSuffix(format, "d") {
		isHex = false
		numDigits, _ = strconv.Atoi(strings.TrimSuffix(format, "d"))
	} else {
		// Try to parse as just a number, assume hex
		numDigits, _ = strconv.Atoi(format)
		isHex = true
	}

	base := "hex"
	if !isHex {
		base = "decimal"
	}
	log.Printf("Keyboard reader format: %s (%d %s digits)", format, numDigits, base)

	return &Keyboard{
		device:    dev,
		numDigits: numDigits,
		isHex:     isHex,
		format:    format,
	}, nil
}

// Read implements TagReader.Read for keyboard readers.
// Reads digits until Enter is pressed, then parses according to configured format.
func (k *Keyboard) Read(ctx context.Context) (uint64, error) {
	ch := k.device.Poll(ctx)
	var strbuf string

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case event := <-ch:
			if event == nil {
				return 0, fmt.Errorf("keyboard device closed")
			}

			switch event.Type.(type) {
			case evdev.KeyType:
				if event.Value != 1 {
					continue
				}

				if event.Type == evdev.KeyEnter {
					if strbuf == "" {
						continue
					}

					// Check digit count if specified
					if k.numDigits > 0 && len(strbuf) != k.numDigits {
						log.Printf("Bad badge: expected %d digits, got %d (%q)", k.numDigits, len(strbuf), strbuf)
						strbuf = ""
						continue
					}

					// Parse based on format
					var base int
					if k.isHex {
						base = 16
					} else {
						base = 10
					}

					number, err := strconv.ParseUint(strbuf, base, 64)
					if err != nil {
						log.Printf("Bad badge line %q (base %d): %v", strbuf, base, err)
						strbuf = ""
						continue
					}

					number &= 0xffffffff
					log.Printf("Got %s String %s BadgeId %d", k.format, strbuf, number)
					return number, nil
				}

				s := evdev.KeyType(event.Code).String()
				strbuf += s
			}
		}
	}
}

// Close implements TagReader.Close.
func (k *Keyboard) Close() error {
	if k.device == nil {
		return nil
	}
	return k.device.Close()
}
