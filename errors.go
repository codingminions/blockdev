package blockdev

import "errors"

// Sentinels for errors.Is matching; methods wrap them with operation context
// so callers branch on category, not on parsed message strings.
var (
	ErrMisaligned  = errors.New("blockdev: offset or length not block-aligned")
	ErrOutOfBounds = errors.New("blockdev: read/write beyond device length")
	ErrBadFormat   = errors.New("blockdev: malformed serialized data")
)
