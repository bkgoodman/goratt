package indicator

import (
	"fmt"
	"os"
)

// Neopixel command strings for the external neopixel tool.
const (
	neoConnectionLost = "@2 !150000 001010"
	neoNormalIdle     = "@3 !150000 400000"
	neoAccessGranted  = "@1 !50000 8000"
	neoAccessDenied   = "@2 !10000 ff"
	neoTerminated     = "@0 010101"
)

// Neopixel implements Indicator using an external neopixel tool via named pipe.
type Neopixel struct {
	pipe       *os.File
	idleString string
}

// NewNeopixel creates a new Neopixel indicator.
func NewNeopixel(pipePath string) (*Neopixel, error) {
	f, err := os.OpenFile(pipePath, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open neopixel pipe %s: %w", pipePath, err)
	}

	n := &Neopixel{
		pipe:       f,
		idleString: neoConnectionLost, // Start with connection lost until connected
	}
	return n, nil
}

// Idle implements Indicator.Idle.
func (n *Neopixel) Idle() {
	n.write(n.idleString)
}

// Granted implements Indicator.Granted.
func (n *Neopixel) Granted() {
	n.write(neoAccessGranted)
}

// Denied implements Indicator.Denied.
func (n *Neopixel) Denied() {
	n.write(neoAccessDenied)
}

// Opening implements Indicator.Opening.
func (n *Neopixel) Opening() {
	n.write(neoAccessGranted)
}

// ConnectionLost implements Indicator.ConnectionLost.
func (n *Neopixel) ConnectionLost() {
	n.idleString = neoConnectionLost
	n.write(neoConnectionLost)
}

// Shutdown implements Indicator.Shutdown.
func (n *Neopixel) Shutdown() {
	n.write(neoTerminated)
}

// Release implements Indicator.Release.
func (n *Neopixel) Release() error {
	if n.pipe == nil {
		return nil
	}
	return n.pipe.Close()
}

// SetConnected updates the idle string to normal when connected.
func (n *Neopixel) SetConnected() {
	n.idleString = neoNormalIdle
}

func (n *Neopixel) write(s string) {
	if n.pipe != nil {
		n.pipe.Write([]byte(s))
	}
}
