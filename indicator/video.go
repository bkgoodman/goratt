//go:build screen

package indicator

import (
	"goratt/video"
)

// VideoIndicator wraps the video.Video type to implement Indicator.
type VideoIndicator struct {
	v *video.Video
}

// NewVideo creates a new video-based indicator.
func NewVideo() (*VideoIndicator, error) {
	v, err := video.New()
	if err != nil {
		return nil, err
	}
	return &VideoIndicator{v: v}, nil
}

// Idle implements Indicator.Idle.
func (vi *VideoIndicator) Idle() {
	vi.v.Idle()
}

// Granted implements Indicator.Granted.
func (vi *VideoIndicator) Granted() {
	vi.v.Granted()
}

// Denied implements Indicator.Denied.
func (vi *VideoIndicator) Denied() {
	vi.v.Denied()
}

// Opening implements Indicator.Opening.
func (vi *VideoIndicator) Opening() {
	vi.v.Opening()
}

// ConnectionLost implements Indicator.ConnectionLost.
func (vi *VideoIndicator) ConnectionLost() {
	vi.v.ConnectionLost()
}

// Shutdown implements Indicator.Shutdown.
func (vi *VideoIndicator) Shutdown() {
	vi.v.Shutdown()
}

// Release implements Indicator.Release.
func (vi *VideoIndicator) Release() error {
	return vi.v.Release()
}

// Video returns the underlying video.Video for direct access (e.g., rotary display).
func (vi *VideoIndicator) Video() *video.Video {
	return vi.v
}
