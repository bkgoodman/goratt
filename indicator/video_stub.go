//go:build !screen

package indicator

import (
	"goratt/video"
)

// NewVideo returns an error when screen support is not compiled in.
func NewVideo() (*VideoIndicator, error) {
	return nil, video.ErrScreenNotCompiled
}

// VideoIndicator is a stub when screen support is not compiled in.
type VideoIndicator struct{}

func (vi *VideoIndicator) Idle()           {}
func (vi *VideoIndicator) Granted()        {}
func (vi *VideoIndicator) Denied()         {}
func (vi *VideoIndicator) Opening()        {}
func (vi *VideoIndicator) ConnectionLost() {}
func (vi *VideoIndicator) Shutdown()       {}
func (vi *VideoIndicator) Release() error  { return nil }
