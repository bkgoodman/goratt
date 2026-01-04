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
func (vi *VideoIndicator) Granted(info *AccessInfo) {
	var member, nickname, warning string
	if info != nil {
		member, nickname, warning = info.Member, info.Nickname, info.Warning
	}
	vi.v.Granted(member, nickname, warning)
}

// Denied implements Indicator.Denied.
func (vi *VideoIndicator) Denied(info *AccessInfo) {
	var member, nickname, warning string
	if info != nil {
		member, nickname, warning = info.Member, info.Nickname, info.Warning
	}
	vi.v.Denied(member, nickname, warning)
}

// Opening implements Indicator.Opening.
func (vi *VideoIndicator) Opening(info *AccessInfo) {
	var member, nickname, warning string
	if info != nil {
		member, nickname, warning = info.Member, info.Nickname, info.Warning
	}
	vi.v.Opening(member, nickname, warning)
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
