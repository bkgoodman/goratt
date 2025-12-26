package door

import (
	"time"

	"github.com/hjkoskel/govattu"
)

// Servo implements DoorOpener using PWM servo control.
type Servo struct {
	hw       govattu.Vattu
	pin      uint8
	openPos  int
	closePos int
	isOpen   bool
}

// NewServo creates a new servo-based door opener.
func NewServo(hw govattu.Vattu, pin uint8, openPos, closePos int) (*Servo, error) {
	hw.PinMode(pin, govattu.ALT5) // ALT5 for PWM0
	hw.PwmSetMode(true, true, false, false)
	hw.PwmSetClock(19)
	hw.Pwm0SetRange(20000)

	s := &Servo{
		hw:       hw,
		pin:      pin,
		openPos:  openPos,
		closePos: closePos,
		isOpen:   false,
	}

	// Start in closed position
	s.moveTo(closePos)
	return s, nil
}

// Open implements DoorOpener.Open.
func (s *Servo) Open() error {
	s.moveFromTo(s.closePos, s.openPos)
	s.isOpen = true
	return nil
}

// Close implements DoorOpener.Close.
func (s *Servo) Close() error {
	s.moveFromTo(s.openPos, s.closePos)
	s.isOpen = false
	return nil
}

// Release implements DoorOpener.Release.
func (s *Servo) Release() error {
	return s.hw.Close()
}

func (s *Servo) moveTo(pos int) {
	s.hw.Pwm0Set(uint32(pos))
}

func (s *Servo) moveFromTo(from, to int) {
	inc := 1
	if to < from {
		inc = -1
	}
	for i := from; i != to; i += inc {
		s.hw.Pwm0Set(uint32(i))
		time.Sleep(2 * time.Millisecond)
	}
}
