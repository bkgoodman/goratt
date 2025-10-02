
package main

import (
    "time"
    "sync/atomic"

    "github.com/warthog618/go-gpiocdev"
)

var rotary_dtLine *gpiocdev.Line
var rotary_clkLine *gpiocdev.Line

var lastCLK, lastDT int

var knobpos int64 = 0
func RotaryHandler(evt gpiocdev.LineEvent) {
    var newState int
    if evt.Type == gpiocdev.LineEventRisingEdge {
        newState = 1
    } else if evt.Type == gpiocdev.LineEventFallingEdge {
        newState = 0
    } else {
        return
    }

    switch evt.Offset {
    case rotary_clkLine.Offset():
        lastCLK = newState
    case rotary_dtLine.Offset():
        lastDT = newState
    }

    // Decode direction based on combined state
    // For example, on CLK edge:
    if evt.Offset == rotary_clkLine.Offset() && evt.Type == gpiocdev.LineEventRisingEdge {
        if lastDT == 0 {
            atomic.AddInt64(&knobpos, 1)
            uiEvent <- UIEvent { Event: Event_Encoderknob, Name:"cw" }
        } else {
            atomic.AddInt64(&knobpos, -1)
            uiEvent <- UIEvent { Event: Event_Encoderknob, Name:"ccw" }
        }
        //fmt.Printf("Knob %d\n",knobpos)
    }
}

func rotary_init() {
    rotaryCLK := 5
    rotaryDT := 6
    rotaryKnob := 13
    chip := "gpiochip0"

    debounceRotary := 250 * time.Microsecond
    debounceButton := 2 * time.Millisecond

    var err error

    // Request DT line for reading direction
    rotary_dtLine, err = gpiocdev.RequestLine(chip, rotaryDT,
        gpiocdev.WithPullUp,
        gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(debounceRotary),
        gpiocdev.WithEventHandler(RotaryHandler))
    if err != nil {
        panic(err)
    }
    //defer rotary_dtLine.Close()

    // Request CLK line with edge detection
    rotary_clkLine, err = gpiocdev.RequestLine(chip, rotaryCLK,
        gpiocdev.WithPullUp,
        gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(debounceRotary),
        gpiocdev.WithEventHandler(RotaryHandler))
    if err != nil {
        panic(err)
    }
    //defer rotary_clkLine.Close()

    // Request button line
    _, err = gpiocdev.RequestLine(chip, rotaryKnob,
        gpiocdev.WithPullUp,
        gpiocdev.WithFallingEdge,
        gpiocdev.WithDebounce(debounceButton),
        gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
            uiEvent <- UIEvent { Event: Event_Encoderknob, Name:"button" }
        }))
    if err != nil {
        panic(err)
    }

    // Keep the program alive
    //select {}
}

