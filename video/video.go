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

	"goratt/video/screen"
	"goratt/video/screen/screens"
)

// ScreenSupported returns whether screen support is compiled in.
func ScreenSupported() bool {
	return true
}

// Rotation specifies display rotation in degrees.
type Rotation int

const (
	Rotation0   Rotation = 0   // No rotation
	Rotation90  Rotation = 90  // 90 degrees clockwise
	Rotation180 Rotation = 180 // 180 degrees (upside down)
	Rotation270 Rotation = 270 // 270 degrees clockwise (90 CCW)
)

// Config holds video display configuration.
type Config struct {
	Rotation int `yaml:"rotation"` // 0, 90, 180, or 270 degrees
}

// Display manages the framebuffer and screen system.
type Display struct {
	dc              *gg.Context
	pixBuffer       []byte
	backBuffer      []byte
	rgbaImage       *image.RGBA
	width           int // Logical width (after rotation)
	height          int // Logical height (after rotation)
	fbWidth         int // Actual framebuffer width
	fbHeight        int // Actual framebuffer height
	lineLengthBytes int
	initialized     bool
	rotation        Rotation

	// Screen management
	manager        *screen.Manager
	idleScreen     *screens.IdleScreen
	grantedScreen  *screens.GrantedScreen
	deniedScreen   *screens.DeniedScreen
	openingScreen  *screens.OpeningScreen
	connLostScreen *screens.ConnectionLostScreen
	shutdownScreen *screens.ShutdownScreen
}

// Video is kept for backward compatibility during transition.
// Deprecated: Use Display instead.
type Video = Display

// New creates a new Video indicator with optional configuration.
func New(cfg ...Config) (*Video, error) {
	v := &Video{}
	if len(cfg) > 0 {
		v.rotation = Rotation(cfg[0].Rotation)
	}
	if err := v.init(); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Display) init() error {
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

	v.fbWidth = int(varInfo.XRes)
	v.fbHeight = int(varInfo.YRes)
	v.lineLengthBytes = int(fixedInfo.LineLength)
	v.backBuffer = make([]byte, v.fbHeight*v.lineLengthBytes)

	// Validate rotation
	switch v.rotation {
	case Rotation0, Rotation180:
		v.width = v.fbWidth
		v.height = v.fbHeight
	case Rotation90, Rotation270:
		v.width = v.fbHeight
		v.height = v.fbWidth
	default:
		log.Printf("Video: invalid rotation %d, using 0", v.rotation)
		v.rotation = Rotation0
		v.width = v.fbWidth
		v.height = v.fbHeight
	}

	log.Printf("Video: framebuffer %dx%d, logical %dx%d, rotation %d, %d bpp, stride %d bytes",
		v.fbWidth, v.fbHeight, v.width, v.height, v.rotation, varInfo.BitsPerPixel, v.lineLengthBytes)

	// Create drawing surface at logical (rotated) dimensions
	v.rgbaImage = image.NewRGBA(image.Rect(0, 0, v.width, v.height))
	v.dc = gg.NewContextForRGBA(v.rgbaImage)
	v.initialized = true

	// Initialize screen manager with logical dimensions
	v.manager = screen.NewManager(v.dc, v.width, v.height, v.update)

	// Create and register screens
	v.idleScreen = screens.NewIdleScreen()
	v.grantedScreen = screens.NewGrantedScreen()
	v.deniedScreen = screens.NewDeniedScreen()
	v.openingScreen = screens.NewOpeningScreen()
	v.connLostScreen = screens.NewConnectionLostScreen()
	v.shutdownScreen = screens.NewShutdownScreen()

	v.manager.Register(screen.ScreenIdle, v.idleScreen)
	v.manager.Register(screen.ScreenGranted, v.grantedScreen)
	v.manager.Register(screen.ScreenDenied, v.deniedScreen)
	v.manager.Register(screen.ScreenOpening, v.openingScreen)
	v.manager.Register(screen.ScreenConnectionLost, v.connLostScreen)
	v.manager.Register(screen.ScreenShutdown, v.shutdownScreen)

	v.clear()
	return nil
}

func (v *Display) clear() {
	for i := range v.pixBuffer {
		v.pixBuffer[i] = 0
	}
}

func (v *Display) update() {
	if !v.initialized {
		return
	}
	// Iterate over logical pixels and map to framebuffer with rotation
	for y := 0; y < v.height; y++ {
		for x := 0; x < v.width; x++ {
			r, g, b, _ := v.rgbaImage.At(x, y).RGBA()
			r5 := uint16(r >> (16 - 5))
			g6 := uint16(g >> (16 - 6))
			b5 := uint16(b >> (16 - 5))
			pixel16 := (r5 << 11) | (g6 << 5) | b5

			// Map logical (x, y) to framebuffer (fbX, fbY) based on rotation
			var fbX, fbY int
			switch v.rotation {
			case Rotation0:
				fbX, fbY = x, y
			case Rotation90:
				// 90 CW: (x,y) -> (fbW-1-y, x)
				fbX = v.fbWidth - 1 - y
				fbY = x
			case Rotation180:
				// 180: (x,y) -> (fbW-1-x, fbH-1-y)
				fbX = v.fbWidth - 1 - x
				fbY = v.fbHeight - 1 - y
			case Rotation270:
				// 270 CW (90 CCW): (x,y) -> (y, fbH-1-x)
				fbX = y
				fbY = v.fbHeight - 1 - x
			}

			fbIdx := (fbY * v.lineLengthBytes) + (fbX * 2)
			if fbIdx+1 < len(v.backBuffer) {
				binary.LittleEndian.PutUint16(v.backBuffer[fbIdx:], pixel16)
			}
		}
	}
	copy(v.pixBuffer, v.backBuffer)
}

// Idle switches to the idle screen.
func (v *Display) Idle() {
	if !v.initialized {
		return
	}
	v.manager.SwitchTo(screen.ScreenIdle)
}

// Granted switches to the granted screen with member info.
func (v *Display) Granted(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.grantedScreen.SetInfo(member, nickname, warning)
	v.manager.SwitchTo(screen.ScreenGranted)
}

// Denied switches to the denied screen with member info.
func (v *Display) Denied(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.deniedScreen.SetInfo(member, nickname, warning)
	v.manager.SwitchTo(screen.ScreenDenied)
}

// Opening switches to the opening screen with member info.
func (v *Display) Opening(member, nickname, warning string) {
	if !v.initialized {
		return
	}
	v.openingScreen.SetInfo(member, nickname, warning)
	v.manager.SwitchTo(screen.ScreenOpening)
}

// ConnectionLost switches to the connection lost screen.
func (v *Display) ConnectionLost() {
	if !v.initialized {
		return
	}
	v.manager.SwitchTo(screen.ScreenConnectionLost)
}

// Shutdown switches to the shutdown screen.
func (v *Display) Shutdown() {
	if !v.initialized {
		return
	}
	v.manager.SwitchTo(screen.ScreenShutdown)
}

// Release releases display resources.
func (v *Display) Release() error {
	v.clear()
	v.initialized = false
	return nil
}

// Manager returns the screen manager for direct access.
func (v *Display) Manager() *screen.Manager {
	return v.manager
}

// SendEvent sends an event to the current screen.
func (v *Display) SendEvent(event screen.Event) bool {
	if v.manager == nil {
		return false
	}
	return v.manager.SendEvent(event)
}

// Width returns the display width.
func (v *Display) Width() int {
	return v.width
}

// Height returns the display height.
func (v *Display) Height() int {
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
