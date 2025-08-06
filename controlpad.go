package main

import (
	"fmt"
	"github.com/warthog618/gpio"
	"os"
	"time"
)

var events chan int64 = make(chan int64, 10)
var state int = STATE_IDLE
var badge_no int64 = 0

func press_led1(pin *gpio.Pin) {
	events <- EVENT_BUTTON1
}
func press_led2(pin *gpio.Pin) {
	events <- EVENT_BUTTON2
}

const (
	STATE_IDLE = iota
	STATE_BADGED_IN
	STATE_BUTTON1
	STATE_BUTTON2
)

const (
	EVENT_BUTTON1 = -1
	EVENT_BUTTON2 = -2
	EVENT_TIMEOUT = -3
)

func rfidreader() {
	/* SIMULATE
	fd,err := os.Open("/dev/tty")
	if (err != nil) {panic(err)}
	defer fd.Close()

	for {
		test:=make([]byte,2)
		size,err := fd.Read(test)
		if (err != nil) {
			panic("Read error")
		}
		fmt.Println("GotScan",size)
		ch <- "SCAN"
	}
	*/
	for {
		tag := readrfid()
		if tag > 0 {
			events <- int64(tag)
		}
	}

}

func controlpad() {
	state = STATE_IDLE

	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	f, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	f.Write([]byte("18\n"))
	f.Close()
	f, err = os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	f.Write([]byte("25\n"))
	f.Close()

	sw1 := gpio.NewPin(18)
	sw2 := gpio.NewPin(25)
	led1 := gpio.NewPin(23)
	led2 := gpio.NewPin(24)
	sw1.Input()
	sw2.Input()
	sw1.PullUp()
	sw2.PullUp()
	led1.Output()
	led2.Output()
	err = sw1.Watch(gpio.EdgeFalling, press_led1)
	if err != nil {
		panic(fmt.Sprintf("sw1 %v", err))
	}
	defer sw1.Unwatch()
	err = sw2.Watch(gpio.EdgeFalling, press_led2)
	if err != nil {
		panic(fmt.Sprintf("sw2 %v", err))
	}
	defer sw2.Unwatch()

	go rfidreader()

	var duty = false
	var timeout time.Time = time.Now()
	for {

		WaitTime := 5000 * time.Millisecond
		if state == STATE_BADGED_IN {
			WaitTime = 100 * time.Millisecond
		}
		duty = !duty
		select {
                /*
		case r := <-requestChannel:
			fmt.Printf("Got aruco request\n")
			print_aruco_dymo(r.Number, r.Name)
            */
		case c := <-events:
			fmt.Println("Got ", c)
			switch c {
			case EVENT_BUTTON1:
				led2.Low()
				led1.High()
				state = STATE_BUTTON1
				if badge_no != 0 {
					//PrintBadge(badge_no, 2)
					timeout = time.Now().Add(time.Second * 5)
				}
				state = STATE_BADGED_IN
			case EVENT_BUTTON2:
				led1.Low()
				led2.High()
				state = STATE_BUTTON2
				if badge_no != 0 {
					// PRINT
					// PrintBadge(badge_no, 1)
					timeout = time.Now().Add(time.Second * 5)
				}
				state = STATE_BADGED_IN
			default:
				badge_no = c
				if c != 0 {
					if state == STATE_BUTTON1 {
						/* print */
						// PrintBadge(badge_no, 2)
					}
					if state == STATE_BUTTON2 {
						/* print */
						// PrintBadge(badge_no, 1)
					}
					timeout = time.Now().Add(time.Second * 5)
					state = STATE_BADGED_IN
				} else {
					state = STATE_IDLE
				}
			}

		case <-time.After(WaitTime):
			if state == STATE_BADGED_IN {
				if duty {
					led1.High()
					led2.Low()
				} else {
					led1.Low()
					led2.High()
				}
			}

			if (state == STATE_BADGED_IN) || (state == STATE_BUTTON1) || (state == STATE_BUTTON2) {
				if time.Now().After(timeout) {
					fmt.Println("End Timeout")
					badge_no = 0
					state = STATE_IDLE
					led1.Low()
					led2.Low()
				}
			}
		}

	}

}
