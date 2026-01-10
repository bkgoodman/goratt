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
  msgsize int // 10 or 8 keys
  base int // Base 10 or Base 8
}

// NewKeyboard creates a new keyboard reader on the specified input device.
func NewKeyboard(kbdtype string,device string) (*Keyboard, error) {
	dev, err := evdev.OpenFile(device)
	if err != nil {
		return nil, fmt.Errorf("open evdev %s: %w", device, err)
	}

	log.Printf("Opened keyboard device: %s", dev.Name())
	log.Printf("Vendor: 0x%04x, Product: 0x%04x", dev.ID().Vendor, dev.ID().Product)

  kbd :=  Keyboard{device: dev}

  switch kbdtype {
    case "keyboard","10h-kbd":
      kbd.base=16
      kbd.msgsize = 10
    case "10d-kbd":
      kbd.base=10
      kbd.msgsize = 10
    case "8d-kbd":
      kbd.base=10
      kbd.msgsize = 8
    case "8h-kbd":
      kbd.base=16
      kbd.msgsize = 8
    default:
      return nil, fmt.Errorf("Invalid keyboard type \"%s\"",kbdtype)
  }
	log.Printf("Keyboard base %d keylength %d\n",kbd.base,kbd.msgsize)
	return &kbd, nil
}

// Read implements TagReader.Read for keyboard readers.
// Reads hex digits until Enter is pressed, then parses as hex.
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
					number, err := strconv.ParseUint(strbuf, k.base, 64)
					if err != nil {
						log.Printf("Bad badge line %q", strbuf)
						strbuf = ""
						continue
					}
					number &= 0xffffffff
					log.Printf("Got (Raw String %s) BadgeId %d", strbuf, number)
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
