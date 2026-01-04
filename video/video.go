//go:build screen

package video

import (
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/d21d3q/framebuffer"
	"github.com/fogleman/gg"
	"golang.org/x/image/draw"
)

// ScreenSupported returns whether screen support is compiled in.
func ScreenSupported() bool {
	return true
}

// Video implements indicator.Indicator using a framebuffer display.
type Video struct {
	dc              *gg.Context
	pixBuffer       []byte
	backBuffer      []byte
	rgbaImage       *image.RGBA
	width           int
	height          int
	lineLengthBytes int
	initialized     bool
}

// New creates a new Video indicator.
func New() (*Video, error) {
	v := &Video{}
	if err := v.init(); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Video) init() error {
	fbLowLevel, err := framebuffer.OpenFrameBuffer("/dev/fb0", os.O_RDWR)
	if err != nil {
		return fmt.Errorf("open framebuffer: %w", err)
	}

	varInfo, err := fbLowLevel.VarScreenInfo()
	if err != nil {
		return fmt.Errorf("get variable screen info: %w", err)
	}
	fixedInfo, err := fbLowLevel.FixScreenInfo()
	if err != nil {
		return fmt.Errorf("get fixed screen info: %w", err)
	}

	v.pixBuffer, err = fbLowLevel.Pixels()
	if err != nil {
		return fmt.Errorf("get pixel data: %w", err)
	}

	v.width = int(varInfo.XRes)
	v.height = int(varInfo.YRes)
	v.lineLengthBytes = int(fixedInfo.LineLength)
	v.backBuffer = make([]byte, v.height*v.lineLengthBytes)

	log.Printf("Video: framebuffer %dx%d, %d bpp, stride %d bytes",
		v.width, v.height, varInfo.BitsPerPixel, v.lineLengthBytes)

	v.rgbaImage = image.NewRGBA(image.Rect(0, 0, v.width, v.height))
	v.dc = gg.NewContextForRGBA(v.rgbaImage)
	v.initialized = true

	v.clear()
	return nil
}

func (v *Video) clear() {
	for i := range v.pixBuffer {
		v.pixBuffer[i] = 0
	}
}

func (v *Video) update() {
	if !v.initialized {
		return
	}
	for y := 0; y < v.height; y++ {
		for x := 0; x < v.width; x++ {
			r, g, b, _ := v.rgbaImage.At(x, y).RGBA()
			r5 := uint16(r >> (16 - 5))
			g6 := uint16(g >> (16 - 6))
			b5 := uint16(b >> (16 - 5))
			pixel16 := (r5 << 11) | (g6 << 5) | b5
			fbIdx := (y * v.lineLengthBytes) + (x * 2)
			if fbIdx+1 < len(v.backBuffer) {
				binary.LittleEndian.PutUint16(v.backBuffer[fbIdx:], pixel16)
			}
		}
	}
	copy(v.pixBuffer, v.backBuffer)
}

func (v *Video) setFontSize(size int) {
	fontPath := "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
	if err := v.dc.LoadFontFace(fontPath, float64(size)); err != nil {
		log.Printf("Video: failed to load font: %v", err)
	}
}

func (v *Video) drawCentered(text string, y float64, r, g, b float64) {
	v.dc.SetRGB(r, g, b)
	v.dc.DrawStringAnchored(text, float64(v.width/2), y, 0.5, 0.5)
}

// Idle implements indicator.Indicator.
func (v *Video) Idle() {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0, 0.5, 0) // Green background
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(64)
	v.drawCentered("Ready", float64(v.height/2), 1, 1, 1)
	v.update()
}

// Granted implements indicator.Indicator.
func (v *Video) Granted(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0, 0.7, 0) // Bright green
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(64)
	y := float64(v.height/2) - 40
	v.drawCentered("Access Granted", y, 1, 1, 1)

	// Display name (prefer nickname, fall back to member)
	v.setFontSize(48)
	displayName := nickname
	if displayName == "" {
		displayName = member
	}
	if displayName != "" {
		v.drawCentered(displayName, y+70, 1, 1, 1)
	}

	// Display warning if present
	if warning != "" {
		v.setFontSize(32)
		v.dc.SetRGB(1, 1, 0) // Yellow warning text
		v.dc.DrawStringAnchored(warning, float64(v.width/2), y+130, 0.5, 0.5)
	}

	v.update()
}

// Denied implements indicator.Indicator.
func (v *Video) Denied(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0.7, 0, 0) // Red
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(64)
	y := float64(v.height/2) - 40
	v.drawCentered("Access Denied", y, 1, 1, 1)

	// Display name if known
	displayName := nickname
	if displayName == "" {
		displayName = member
	}
	if displayName != "" {
		v.setFontSize(48)
		v.drawCentered(displayName, y+70, 1, 1, 1)
	}

	// Display warning/reason if present
	if warning != "" {
		v.setFontSize(32)
		v.dc.SetRGB(1, 1, 0) // Yellow warning text
		v.dc.DrawStringAnchored(warning, float64(v.width/2), y+130, 0.5, 0.5)
	}

	v.update()
}

// Opening implements indicator.Indicator.
func (v *Video) Opening(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0.7, 0.7, 0) // Yellow
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(64)
	y := float64(v.height/2) - 40
	v.drawCentered("Opening...", y, 0, 0, 0)

	// Display name
	displayName := nickname
	if displayName == "" {
		displayName = member
	}
	if displayName != "" {
		v.setFontSize(48)
		v.drawCentered(displayName, y+70, 0, 0, 0)
	}

	// Display warning if present
	if warning != "" {
		v.setFontSize(32)
		v.dc.SetRGB(0.7, 0, 0) // Red warning text on yellow background
		v.dc.DrawStringAnchored(warning, float64(v.width/2), y+130, 0.5, 0.5)
	}

	v.update()
}

// ConnectionLost implements indicator.Indicator.
func (v *Video) ConnectionLost() {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0.5, 0.3, 0) // Orange-ish
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(64)
	v.drawCentered("Connection Lost", float64(v.height/2), 1, 1, 1)
	v.update()
}

// Shutdown implements indicator.Indicator.
func (v *Video) Shutdown() {
	if !v.initialized {
		return
	}
	v.clear()
	v.update()
}

// Release implements indicator.Indicator.
func (v *Video) Release() error {
	v.clear()
	v.initialized = false
	return nil
}

// DisplayNumber shows a number on screen (for rotary encoder testing).
func (v *Video) DisplayNumber(n int64) {
	if !v.initialized {
		return
	}
	v.dc.SetRGB(0, 0, 0.3) // Dark blue background
	v.dc.DrawRectangle(0, 0, float64(v.width), float64(v.height))
	v.dc.Fill()

	v.setFontSize(128)
	v.drawCentered(fmt.Sprintf("%d", n), float64(v.height/2), 1, 1, 1)
	v.update()
}

// Width returns the display width.
func (v *Video) Width() int {
	return v.width
}

// Height returns the display height.
func (v *Video) Height() int {
	return v.height
}

// imageToRGBA converts an image.Image to *image.RGBA.
func imageToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}
