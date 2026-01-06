package eventpipe

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"goratt/video/screen"
)

// Config holds configuration for the event pipe.
type Config struct {
	Path string `yaml:"path"` // Path to named pipe (e.g., "/tmp/goratt-events")
}

// EventHandler is called when an event is received from the pipe.
type EventHandler func(screen.Event)

// EventPipe listens for events on a named pipe.
type EventPipe struct {
	path    string
	handler EventHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new EventPipe. Returns nil if path is empty.
func New(cfg Config, handler EventHandler) (*EventPipe, error) {
	if cfg.Path == "" {
		return nil, nil
	}

	// Remove existing pipe if it exists
	os.Remove(cfg.Path)

	// Create the named pipe
	if err := syscall.Mkfifo(cfg.Path, 0666); err != nil {
		return nil, fmt.Errorf("create named pipe %s: %w", cfg.Path, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ep := &EventPipe{
		path:    cfg.Path,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}

	return ep, nil
}

// Start begins listening for events on the pipe.
// This should be called as a goroutine.
func (ep *EventPipe) Start() {
	log.Printf("Event pipe listening on %s", ep.path)

	for {
		select {
		case <-ep.ctx.Done():
			return
		default:
		}

		// Open pipe for reading (blocks until writer connects)
		// We open in non-blocking mode first, then switch to blocking
		// This allows us to check for context cancellation
		file, err := os.OpenFile(ep.path, os.O_RDONLY, 0)
		if err != nil {
			if ep.ctx.Err() != nil {
				return
			}
			log.Printf("Event pipe open error: %v", err)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			select {
			case <-ep.ctx.Done():
				file.Close()
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			event, err := parseLine(line)
			if err != nil {
				log.Printf("Event pipe parse error: %v", err)
				continue
			}

			if ep.handler != nil {
				ep.handler(event)
			}
		}

		file.Close()
		// Writer closed the pipe, loop back to wait for next writer
	}
}

// Close stops the event pipe listener and removes the pipe.
func (ep *EventPipe) Close() error {
	ep.cancel()
	return os.Remove(ep.path)
}

// parseLine parses a command line into an Event.
// Command format:
//
//	rfid <tagid>                    - Raw RFID swipe
//	tag <tagid>                     - Alias for rfid
//	rotary <delta>                  - Rotary turn (+1 or -1)
//	rotary press                    - Rotary button press
//	pin <name> <0|1>                - Pin state change (0=released, 1=pressed)
//	screen <name>                   - Switch to screen (idle, granted, denied, etc.)
func parseLine(line string) (screen.Event, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return screen.Event{}, fmt.Errorf("empty command")
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "rfid", "tag":
		if len(parts) < 2 {
			return screen.Event{}, fmt.Errorf("rfid requires tag ID")
		}
		tagID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			// Try hex
			tagID, err = strconv.ParseUint(parts[1], 16, 64)
			if err != nil {
				return screen.Event{}, fmt.Errorf("invalid tag ID: %s", parts[1])
			}
		}
		return screen.Event{
			Type: screen.EventRFID,
			Data: screen.RFIDData{TagID: tagID},
		}, nil

	case "rotary":
		if len(parts) < 2 {
			return screen.Event{}, fmt.Errorf("rotary requires delta or 'press'")
		}
		if strings.ToLower(parts[1]) == "press" {
			return screen.Event{
				Type: screen.EventRotaryPress,
				Data: screen.RotaryData{ID: screen.RotaryMain},
			}, nil
		}
		delta, err := strconv.Atoi(parts[1])
		if err != nil {
			return screen.Event{}, fmt.Errorf("invalid rotary delta: %s", parts[1])
		}
		return screen.Event{
			Type: screen.EventRotaryTurn,
			Data: screen.RotaryData{ID: screen.RotaryMain, Delta: delta},
		}, nil

	case "pin":
		if len(parts) < 3 {
			return screen.Event{}, fmt.Errorf("pin requires <name> <0|1>")
		}
		pinID, err := parsePinID(parts[1])
		if err != nil {
			return screen.Event{}, err
		}
		pressed := parts[2] == "1" || strings.ToLower(parts[2]) == "true"
		return screen.Event{
			Type: screen.EventPin,
			Data: screen.PinData{ID: pinID, Pressed: pressed},
		}, nil

	default:
		return screen.Event{}, fmt.Errorf("unknown command: %s", cmd)
	}
}

// parsePinID converts a pin name to PinID.
func parsePinID(name string) (screen.PinID, error) {
	switch strings.ToLower(name) {
	case "button1", "btn1":
		return screen.PinButton1, nil
	case "button2", "btn2":
		return screen.PinButton2, nil
	case "sensor1":
		return screen.PinSensor1, nil
	case "sensor2":
		return screen.PinSensor2, nil
	case "estop":
		return screen.PinEstop, nil
	case "door":
		return screen.PinDoor, nil
	case "safelight":
		return screen.PinSafelight, nil
	case "activity":
		return screen.PinActivity, nil
	case "enable":
		return screen.PinEnable, nil
	default:
		// Try parsing as number
		id, err := strconv.Atoi(name)
		if err != nil {
			return 0, fmt.Errorf("unknown pin: %s", name)
		}
		return screen.PinID(id), nil
	}
}
