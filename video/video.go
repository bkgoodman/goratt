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

// Display manages the framebuffer and screen system.
type Display struct {
	dc              *gg.Context
	pixBuffer       []byte
	backBuffer      []byte
	rgbaImage       *image.RGBA
	width           int
	height          int
	lineLengthBytes int
	initialized     bool

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

// New creates a new Video indicator.
func New() (*Video, error) {
	v := &Video{}
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

	v.width = int(varInfo.XRes)
	v.height = int(varInfo.YRes)
	v.lineLengthBytes = int(fixedInfo.LineLength)
	v.backBuffer = make([]byte, v.height*v.lineLengthBytes)

	log.Printf("Video: framebuffer %dx%d, %d bpp, stride %d bytes",
		v.width, v.height, varInfo.BitsPerPixel, v.lineLengthBytes)

	v.rgbaImage = image.NewRGBA(image.Rect(0, 0, v.width, v.height))
	v.dc = gg.NewContextForRGBA(v.rgbaImage)
	v.initialized = true

	// Initialize screen manager
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
