package main

import (
  "fmt"
  "time"

  // rpio "github.com/stianeikeland/go-rpio"	
  "github.com/hjkoskel/govattu"
)


// 12, 13, 18 and 19 are Hardware PWM
func door_reset(pos bool) {
        if (cfg.DoorPin == nil) {
                return
        }
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(uint8(*cfg.DoorPin), govattu.ALToutput)  // ALT5 function for 18 is PWM0
	if (pos) {
		hw.PinSet(uint8(*cfg.DoorPin))
	} else  {
		hw.PinClear(uint8(*cfg.DoorPin))
	}
	hw.Close()
}

func servo_reset(pos int) {
        if (cfg.DoorPin == nil) {
                return
        }
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(uint8(*cfg.DoorPin), govattu.ALT5)  // ALT5 function for 18 is PWM0
	hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
	hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
	hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width
	hw.Pwm0Set(uint32(pos))  
	time.Sleep(5 * time.Second)
	hw.Close()
}


func servoFromTo(hw govattu.Vattu, from int, to int) {
  var inc int = 1
  if (to < from) { inc = -1 }
	fmt.Println("From",from,"To",to,"Inc",inc)
	for i := from; i != to; i += inc  {
		hw.Pwm0Set(uint32(i))  // Set pwm
		time.Sleep(2 * time.Millisecond)
	}
}

func servo_holdopen(servoOpen int, servoClose int, waitSecs int, mode string) {
        if (cfg.DoorPin == nil) {
                return
        }
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(uint8(*cfg.DoorPin), govattu.ALT5)  // ALT5 function for 18 is PWM0
	hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
	hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
	hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width

	fmt.Println("Servo Opening XX.")
	//hw.PinSet(25)
  servoFromTo(hw,servoClose,servoOpen)
	//hw.PinClear(25)
	fmt.Println("Servo Pausing.")
	//hw.PinSet(24)
	for {
		time.Sleep(time.Duration(waitSecs) * time.Second)
	}
}
func open_servo(servoOpen int, servoClose int, waitSecs int, mode string) {
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}
    if (cfg.DoorPin != nil) {

            hw.PinMode(uint8(*cfg.DoorPin), govattu.ALT5)  // ALT5 function for 18 is PWM0

            if (mode == "servo") {
                hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
                hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
                hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width
            } else {
                hw.PinMode(uint8(*cfg.DoorPin), govattu.ALToutput)  // ALT5 function for 18 is PWM0
            }

    }

    if (cfg.YellowLED != nil) {
            hw.PinSet(*cfg.YellowLED)
    }
	fmt.Println("Servo Opening XX.")
  LEDwriteString(LEDaccessGranted)

    if (cfg.DoorPin != nil) {
            switch (mode) {
                case "servo":
                    servoFromTo(hw,servoClose,servoOpen)
                case "openhigh":
                    hw.PinSet(uint8(*cfg.DoorPin))
                case "openlow":
                    hw.PinClear(uint8(*cfg.DoorPin))
            }
    }

	if (cfg.YellowLED != nil) { hw.PinClear(*cfg.YellowLED) }
	fmt.Println("Servo Pausing.")
	if (cfg.GreenLED != nil) { hw.PinSet(*cfg.GreenLED) }
	time.Sleep(time.Duration(waitSecs) * time.Second)
	if (cfg.GreenLED != nil) { hw.PinClear(*cfg.GreenLED) }
	if (cfg.YellowLED != nil) { hw.PinSet(*cfg.YellowLED) }

	fmt.Println("Servo Closing.")
	switch (mode) {
		case "servo":
			servoFromTo(hw,servoOpen,servoClose)
		case "openhigh":
			hw.PinClear(uint8(*cfg.DoorPin))
		case "openlow":
			hw.PinSet(uint8(*cfg.DoorPin))
		default:
			panic("Invalid mode in configu file")
	}
	if (cfg.YellowLED != nil) { hw.PinClear(*cfg.YellowLED) }
    LEDwriteString(LEDidleString) // Set LED to Idle
	fmt.Println("Servo End.")
	hw.Close()
}
