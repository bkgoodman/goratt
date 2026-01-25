//go:build screen

package screens

import (
	"image"
	"image/color"
	"math"
)

// Spinner pre-rendered frames (shared across instances)
var spinnerFrames []*image.RGBA
var spinnerSize = 40

func init() {
	// Pre-render 8 spinner frames
	spinnerFrames = make([]*image.RGBA, 8)
	for i := 0; i < 8; i++ {
		spinnerFrames[i] = renderSpinnerFrame(spinnerSize, i)
	}
}

// SpinnerFrames returns the pre-rendered spinner animation frames.
func SpinnerFrames() []*image.RGBA {
	return spinnerFrames
}

// SpinnerSize returns the size of the spinner in pixels.
func SpinnerSize() int {
	return spinnerSize
}

// renderSpinnerFrame creates a single frame of the spinner animation
func renderSpinnerFrame(size, frame int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size)/2 - 4

	// Draw 8 dots around the circle, with varying brightness
	for dot := 0; dot < 8; dot++ {
		angle := float64(dot) * math.Pi / 4
		x := cx + radius*math.Cos(angle)
		y := cy + radius*math.Sin(angle)

		// Calculate brightness based on distance from current frame
		dist := (dot - frame + 8) % 8
		brightness := uint8(255 - dist*28) // Fade from 255 to ~59

		// Draw a filled circle (5x5 pixels for 40px spinner)
		dotRadius := size / 10 // 4 pixels for 40px spinner
		for dy := -dotRadius; dy <= dotRadius; dy++ {
			for dx := -dotRadius; dx <= dotRadius; dx++ {
				if dx*dx+dy*dy <= dotRadius*dotRadius {
					px, py := int(x)+dx, int(y)+dy
					if px >= 0 && px < size && py >= 0 && py < size {
						img.Set(px, py, color.RGBA{brightness, brightness, brightness, 255})
					}
				}
			}
		}
	}

	return img
}
