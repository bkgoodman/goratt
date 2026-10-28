package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

type PadState int

const (
	StateIdle PadState = iota
	StateBadgedIn
	StateButton1
	StateButton2
)

// ControlPad manages the two buttons and two LEDs on the label printer.
type ControlPad struct {
	cfg     LabelPrinterConfig
	printer *Printer
	publish func(topic string, message string)

	btn1Pin gpio.PinIO
	btn2Pin gpio.PinIO
	led1Pin gpio.PinIO
	led2Pin gpio.PinIO

	mu            sync.Mutex
	state         PadState
	currentMember string
	timeout       time.Time

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// unexportGPIO releases a GPIO pin from Linux sysfs (/sys/class/gpio/unexport)
// if it was previously exported (e.g., by older software or startup scripts),
// allowing periph.io character device driver (/dev/gpiochip*) to claim it without EBUSY.
func unexportGPIO(pinNum int) {
	if pinNum <= 0 {
		return
	}
	f, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%d\n", pinNum)
}

// NewControlPad creates and initializes the button and LED controller.
func NewControlPad(cfg LabelPrinterConfig, printer *Printer, publishFn func(string, string)) *ControlPad {
	cp := &ControlPad{
		cfg:     cfg,
		printer: printer,
		publish: publishFn,
		state:   StateIdle,
		stopCh:  make(chan struct{}),
	}

	// Release pins from sysfs if previously exported, avoiding device-busy errors
	unexportGPIO(cfg.Button1Pin)
	unexportGPIO(cfg.Button2Pin)
	unexportGPIO(cfg.LED1Pin)
	unexportGPIO(cfg.LED2Pin)

	// Lookup and configure GPIO pins via periph.io
	cp.btn1Pin = lookupPin(cfg.Button1Pin)
	cp.btn2Pin = lookupPin(cfg.Button2Pin)
	cp.led1Pin = lookupPin(cfg.LED1Pin)
	cp.led2Pin = lookupPin(cfg.LED2Pin)

	if cp.btn1Pin != nil {
		if err := cp.btn1Pin.In(gpio.PullUp, gpio.BothEdges); err != nil {
			log.Printf("Warning: configure Button1 pin %d: %v", cfg.Button1Pin, err)
			cp.btn1Pin = nil
		}
	}
	if cp.btn2Pin != nil {
		if err := cp.btn2Pin.In(gpio.PullUp, gpio.BothEdges); err != nil {
			log.Printf("Warning: configure Button2 pin %d: %v", cfg.Button2Pin, err)
			cp.btn2Pin = nil
		}
	}
	if cp.btn1Pin == nil && cp.btn2Pin == nil {
		log.Printf("GPIO button pins not available; hardware button inputs disabled")
	}

	if cp.led1Pin != nil {
		if err := cp.led1Pin.Out(gpio.Low); err != nil {
			log.Printf("Warning: configure LED1 pin %d: %v", cfg.LED1Pin, err)
			cp.led1Pin = nil
		}
	}
	if cp.led2Pin != nil {
		if err := cp.led2Pin.Out(gpio.Low); err != nil {
			log.Printf("Warning: configure LED2 pin %d: %v", cfg.LED2Pin, err)
			cp.led2Pin = nil
		}
	}
	if cp.led1Pin == nil && cp.led2Pin == nil {
		log.Printf("GPIO LED pins not available; hardware LED outputs disabled")
	}

	return cp
}

func lookupPin(pinNum int) gpio.PinIO {
	if pinNum <= 0 {
		return nil
	}
	pinName := fmt.Sprintf("GPIO%d", pinNum)
	p := gpioreg.ByName(pinName)
	if p == nil {
		p = gpioreg.ByName(fmt.Sprintf("%d", pinNum))
	}
	return p
}

// Start launches the LED animation/timeout loop and button polling goroutines.
func (cp *ControlPad) Start() {
	cp.wg.Add(1)
	go cp.runStateLoop()

	if cp.btn1Pin != nil {
		cp.wg.Add(1)
		go cp.pollButton(cp.btn1Pin, func() {
			cp.onButtonPressed(1)
		})
	}

	if cp.btn2Pin != nil {
		cp.wg.Add(1)
		go cp.pollButton(cp.btn2Pin, func() {
			cp.onButtonPressed(2)
		})
	}
}

// Stop cleanly terminates the button watchers and turns off LEDs.
func (cp *ControlPad) Stop() {
	close(cp.stopCh)
	cp.wg.Wait()

	if cp.led1Pin != nil {
		cp.led1Pin.Out(gpio.Low)
	}
	if cp.led2Pin != nil {
		cp.led2Pin.Out(gpio.Low)
	}
}

// HandleBadgeScan is called when a valid RFID tag is swiped.
func (cp *ControlPad) HandleBadgeScan(member string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	log.Printf("ControlPad: badge scanned for member %q in state %v", member, cp.state)

	switch cp.state {
	case StateButton1:
		// Button 1 was already pressed, print Free Table immediately
		log.Printf("Button 1 was pre-selected; printing Free Table label for %s", member)
		cp.triggerPrintLocked(2, member)
		cp.state = StateBadgedIn
		cp.timeout = time.Now().Add(5 * time.Second)

	case StateButton2:
		// Button 2 was already pressed, print Storage Pass immediately
		log.Printf("Button 2 was pre-selected; printing Storage Pass label for %s", member)
		cp.triggerPrintLocked(1, member)
		cp.state = StateBadgedIn
		cp.timeout = time.Now().Add(5 * time.Second)

	default:
		// Transition to badged-in state, start alternating LED blink
		cp.currentMember = member
		cp.state = StateBadgedIn
		cp.timeout = time.Now().Add(5 * time.Second)
	}
}

func (cp *ControlPad) onButtonPressed(btnNum int) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	log.Printf("ControlPad: Button %d pressed in state %v", btnNum, cp.state)

	if cp.currentMember != "" {
		// User is badged in, print the requested label!
		if btnNum == 1 {
			log.Printf("Printing Free Table label for badged-in member %s", cp.currentMember)
			cp.triggerPrintLocked(2, cp.currentMember)
		} else {
			log.Printf("Printing Storage Pass label for badged-in member %s", cp.currentMember)
			cp.triggerPrintLocked(1, cp.currentMember)
		}
		cp.state = StateBadgedIn
		cp.timeout = time.Now().Add(5 * time.Second)
	} else {
		// User pressed button first before badging in
		if btnNum == 1 {
			cp.state = StateButton1
			cp.setLEDs(gpio.High, gpio.Low)
		} else {
			cp.state = StateButton2
			cp.setLEDs(gpio.Low, gpio.High)
		}
		cp.timeout = time.Now().Add(5 * time.Second)
	}
}

func (cp *ControlPad) triggerPrintLocked(printType int, member string) {
	// Launch printing in background goroutine to not block state machine
	go func() {
		var err error
		if printType == 2 {
			err = cp.printer.PrintFreeTable(member)
		} else {
			err = cp.printer.PrintStoragePass(member)
		}
		if err != nil {
			log.Printf("Error printing label: %v", err)
			return
		}

		if cp.publish != nil {
			msg := fmt.Sprintf(`{"allowed":true,"member":%q}`, member)
			cp.publish(cp.cfg.Personality, msg)
		}
	}()
}

func (cp *ControlPad) runStateLoop() {
	defer cp.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	duty := false

	for {
		select {
		case <-cp.stopCh:
			return
		case <-ticker.C:
			cp.mu.Lock()
			state := cp.state
			timeout := cp.timeout

			if state == StateBadgedIn {
				duty = !duty
				if duty {
					cp.setLEDs(gpio.High, gpio.Low)
				} else {
					cp.setLEDs(gpio.Low, gpio.High)
				}
			}

			if state != StateIdle && time.Now().After(timeout) {
				log.Println("ControlPad timeout reached, returning to idle")
				cp.state = StateIdle
				cp.currentMember = ""
				cp.setLEDs(gpio.Low, gpio.Low)
			}
			cp.mu.Unlock()
		}
	}
}

func (cp *ControlPad) setLEDs(led1, led2 gpio.Level) {
	if cp.led1Pin != nil {
		cp.led1Pin.Out(led1)
	}
	if cp.led2Pin != nil {
		cp.led2Pin.Out(led2)
	}
}

// pollButton implements level-change detection with debounce for mechanical switches.
func (cp *ControlPad) pollButton(pin gpio.PinIO, onPress func()) {
	defer cp.wg.Done()
	lastStableLevel := pin.Read()
	const debounceDelay = 15 * time.Millisecond

	for {
		select {
		case <-cp.stopCh:
			return
		default:
		}

		if !pin.WaitForEdge(100 * time.Millisecond) {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		level := pin.Read()
		if level == lastStableLevel {
			continue
		}

		time.Sleep(debounceDelay)
		confirmedLevel := pin.Read()
		if confirmedLevel == lastStableLevel {
			continue
		}

		lastStableLevel = confirmedLevel
		if confirmedLevel == gpio.Low { // Active-low with internal pull-up
			onPress()
		}
	}
}
