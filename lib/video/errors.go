package video

import "errors"

// ErrScreenNotCompiled is returned when screen support was not compiled in.
var ErrScreenNotCompiled = errors.New("screen support not compiled in (build with -tags=screen)")
