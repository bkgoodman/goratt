//go:build !screen

package video

// ScreenSupported returns whether screen support is compiled in.
func ScreenSupported() bool {
	return false
}

// Video is a stub when screen support is not compiled in.
type Video struct{}

// New returns an error when screen support is not compiled in.
func New() (*Video, error) {
	return nil, ErrScreenNotCompiled
}

func (v *Video) Idle()                 {}
func (v *Video) Granted()              {}
func (v *Video) Denied()               {}
func (v *Video) Opening()              {}
func (v *Video) ConnectionLost()       {}
func (v *Video) Shutdown()             {}
func (v *Video) Release() error        { return nil }
func (v *Video) DisplayNumber(n int64) {}
func (v *Video) Width() int            { return 0 }
func (v *Video) Height() int           { return 0 }

// Rotary is a stub when screen support is not compiled in.
type Rotary struct{}

// RotaryConfig holds configuration for a rotary encoder.
type RotaryConfig struct {
	Chip      string
	CLKPin    int
	DTPin     int
	ButtonPin int
	OnTurn    func(delta int)
	OnPress   func()
}

// NewRotary returns an error when screen support is not compiled in.
func NewRotary(cfg RotaryConfig) (*Rotary, error) {
	return nil, ErrScreenNotCompiled
}

func (r *Rotary) Position() int64 { return 0 }
func (r *Rotary) Release() error  { return nil }
