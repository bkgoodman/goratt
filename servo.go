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
	SERVO_START = 1000
	SERVO_END = 1700
)

func servo_reset() {
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(SERVO_PIN, govattu.ALT5)  // ALT5 function for 18 is PWM0
	hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
	hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
	hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width
		hw.Pwm0Set(uint32(SERVO_END))  
	time.Sleep(5 * time.Second)
	hw.Close()
}


func open_servo() {
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}

	hw.PinMode(SERVO_PIN, govattu.ALT5)  // ALT5 function for 18 is PWM0
	hw.PwmSetMode(true, true, false, false)  // PwmSetMode(en0 bool, ms0 bool, en1 bool, ms1 bool) enable and set to mark-space mode for pwm0 and pwm1
	hw.PwmSetClock(19)  // Set clock divisor to get 50Hz frequency
	hw.Pwm0SetRange(20000)  // SET RANGE to get 1ms - 2ms pulse width

	hw.Pwm0Set(uint32(1000))  // Set pwm
	fmt.Println("Servo Start.")
	for i := SERVO_END; i > SERVO_START; i -= 10 {
		hw.Pwm0Set(uint32(i))  // Set pwm
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(5 * time.Second)

	for i := SERVO_START; i <= SERVO_END; i += 10 {
		hw.Pwm0Set(uint32(i))  // Set pwm
		time.Sleep(10 * time.Millisecond)
	}


	fmt.Println("Servo End.")
	hw.Close()
}
