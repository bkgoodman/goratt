package reader

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/kenshaw/evdev"
)

// Keyboard implements TagReader for USB keyboard-style RFID readers
// that output hex digits followed by Enter.
type Keyboard struct {
	device *evdev.Evdev
}

// NewKeyboard creates a new keyboard reader on the specified input device.
func NewKeyboard(device string) (*Keyboard, error) {
	dev, err := evdev.OpenFile(device)
	if err != nil {
		return nil, fmt.Errorf("open evdev %s: %w", device, err)
	}

	log.Printf("Opened keyboard device: %s", dev.Name())
	log.Printf("Vendor: 0x%04x, Product: 0x%04x", dev.ID().Vendor, dev.ID().Product)

	return &Keyboard{device: dev}, nil
}

// Read implements TagReader.Read for keyboard readers.
// Reads hex digits until Enter is pressed, then parses as hex.
func (k *Keyboard) Read(ctx context.Context) (uint64, error) {
	ch := k.device.Poll(ctx)
	var strbuf string

	for {
		select {
		case <-ctx.Done():
      fmt.Println("Keyboard done")
			return 0, ctx.Err()
		case event := <-ch:
      fmt.Println("Keyboard chan")
			if event == nil {
				return 0, fmt.Errorf("keyboard device closed")
			}

      fmt.Printf("Keyboard got event %v\n",event)
			switch event.Type.(type) {
			case evdev.KeyType:
				if event.Value != 1 {
					continue
				}

				if event.Type == evdev.KeyEnter {
					if strbuf == "" {
						continue
					}
					number, err := strconv.ParseUint(strbuf, 16, 64)
					if err != nil {
						log.Printf("Bad hex badge line %q", strbuf)
						strbuf = ""
						continue
					}
					number &= 0xffffffff
					log.Printf("Got 10h String %s BadgeId %d", strbuf, number)
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
