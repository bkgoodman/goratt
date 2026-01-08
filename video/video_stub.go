//go:build !screen

package video

import "goratt/video/screen"

// ScreenSupported returns whether screen support is compiled in.
func ScreenSupported() bool {
	return false
}

// Config holds video display configuration.
type Config struct {
	Rotation int `yaml:"rotation"` // 0, 90, 180, or 270 degrees
}

// Display is a stub when screen support is not compiled in.
type Display struct{}

// Video is kept for backward compatibility.
type Video = Display

// New returns an error when screen support is not compiled in.
func New(cfg ...Config) (*Display, error) {
	return nil, ErrScreenNotCompiled
}

func (v *Display) Idle()                                    {}
func (v *Display) Granted(member, nickname, warning string) {}
func (v *Display) Denied(member, nickname, warning string)  {}
func (v *Display) Opening(member, nickname, warning string) {}
func (v *Display) ConnectionLost()                          {}
func (v *Display) Shutdown()                                {}
func (v *Display) Release() error                           { return nil }
func (v *Display) Width() int                               { return 0 }
func (v *Display) Height() int                              { return 0 }
func (v *Display) Manager() *screen.Manager                 { return nil }
func (v *Display) SendEvent(event screen.Event) bool        { return false }
func (v *Display) SetMQTTConnected(connected bool)          {}
