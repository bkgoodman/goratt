
package main

import (
    "fmt"
    "time"

    "github.com/warthog618/go-gpiocdev"
)

var rotary_dtLine *gpiocdev.Line
var rotary_clkLine *gpiocdev.Line

var lastCLK, lastDT int

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
            fmt.Println("Clockwise")
        } else {
            fmt.Println("Counter-clockwise")
        }
    }
}

func rotary_init() {
    rotaryCLK := 5
    rotaryDT := 6
    rotaryKnob := 13
    chip := "gpiochip0"

    debounceRotary := 5 * time.Millisecond
    debounceButton := 10 * time.Millisecond

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
    defer rotary_dtLine.Close()

    // Request CLK line with edge detection
    rotary_clkLine, err = gpiocdev.RequestLine(chip, rotaryCLK,
        gpiocdev.WithPullUp,
        gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(debounceRotary),
        gpiocdev.WithEventHandler(RotaryHandler))
    if err != nil {
        panic(err)
    }
    defer rotary_clkLine.Close()

    // Request button line
    _, err = gpiocdev.RequestLine(chip, rotaryKnob,
        gpiocdev.WithPullUp,
        gpiocdev.WithFallingEdge,
        gpiocdev.WithDebounce(debounceButton),
        gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
            fmt.Println("Button pressed")
        }))
    if err != nil {
        panic(err)
    }

    // Keep the program alive
    select {}
}

