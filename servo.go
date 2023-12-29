package main

import (
  "fmt"
  "time"

  // rpio "github.com/stianeikeland/go-rpio"	
  "github.com/hjkoskel/govattu"
)


// 12, 13, 18 and 19 are Hardware PWM
const (
	SERVO_PIN = 18
)

func servo_reset(pos int) {
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(SERVO_PIN, govattu.ALT5)  // ALT5 function for 18 is PWM0
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

func open_servo(servoOpen int, servoClose int, waitSecs int) {
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(SERVO_PIN, govattu.ALT5)  // ALT5 function for 18 is PWM0
	hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
	hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
	hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width

	fmt.Println("Servo Opening XX.")
  servoFromTo(hw,servoClose,servoOpen)

	fmt.Println("Servo Pausing.")
	time.Sleep(time.Duration(waitSecs) * time.Second)

	fmt.Println("Servo Closing.")
  servoFromTo(hw,servoOpen,servoClose)


	fmt.Println("Servo End.")
	hw.Close()
}
