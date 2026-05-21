package door

import (
	"fmt"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
)

// Servo implements DoorOpener using PWM servo control.
type Servo struct {
	pin      gpio.PinIO
	openPos  int
	closePos int
	isOpen   bool
}

// NewServo creates a new servo-based door opener.
func NewServo(pinNum int, openPos, closePos int) (*Servo, error) {
	pinName := fmt.Sprintf("GPIO%d", pinNum)
	p := gpioreg.ByName(pinName)
	if p == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", pinName)
	}

	s := &Servo{
		pin:      p,
		openPos:  openPos,
		closePos: closePos,
		isOpen:   false,
	}

	// Start in closed position (in a goroutine to prevent periph.io PWM hangs from freezing startup)
	go s.moveTo(closePos)
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
	return s.pin.Halt()
}

// posToDuty converts a servo position (pulse width in microseconds) to a PWM duty cycle
// at 50Hz (20ms period). The position value represents microseconds of pulse width.
func posToDuty(pos int) gpio.Duty {
	// 50Hz = 20000us period. pos is in microseconds.
	// duty = pos / 20000 * DutyMax
	return gpio.Duty(int64(pos) * int64(gpio.DutyMax) / 20000)
}

func (s *Servo) moveTo(pos int) {
	duty := posToDuty(pos)
	// 50Hz for standard servo control
	_ = s.pin.PWM(duty, 50*physic.Hertz)
}

func (s *Servo) moveFromTo(from, to int) {
	inc := 1
	if to < from {
		inc = -1
	}
	for i := from; i != to; i += inc {
		duty := posToDuty(i)
		s.pin.PWM(duty, 50*physic.Hertz)
		time.Sleep(2 * time.Millisecond)
	}
}
