package main

import (
    "fmt"
    "time"

    "github.com/warthog618/go-gpiocdev"
)

var rotary_dtLine *gpiocdev.Line

func RotaryEvent(evt gpiocdev.LineEvent) {
    dtState, err := rotary_dtLine.Value()
    if err != nil {
        return
    }

    if evt.Type == gpiocdev.LineEventRisingEdge {
        if dtState == 0 {
            fmt.Println("Clockwise")
        } else {
            fmt.Println("Counter-clockwise")
        }
    }
}
func rotary_init() {
    // Named GPIO pins
    rotaryCLK := 23
    rotaryDT := 24
    rotaryKnob := 25
    chip := "gpiochip0"
    var err error

    // Debounce durations
    debounceRotary := 5 * time.Millisecond
    debounceButton := 10 * time.Millisecond

    // Request DT line for reading direction
    rotary_dtLine, err = gpiocdev.RequestLine(chip, rotaryDT,
        gpiocdev.AsInput,
        gpiocdev.WithPullUp)
    if err != nil {
        panic(err)
    }
    defer rotary_dtLine.Close()

    // Request CLK line with edge detection
    _, err = gpiocdev.RequestLine(chip, rotaryCLK,
        gpiocdev.WithPullUp,
        gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(debounceRotary),
        gpiocdev.WithEventHandler(RotaryEvent))
    if err != nil {
        panic(err)
    }

    // Request button line with falling edge detection
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

