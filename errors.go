package blockdev

import "errors"

var (
	// ErrMisaligned indicates an offset or length is not a multiple of BlockSize.
	ErrMisaligned = errors.New("blockdev: offset or length not block-aligned")

	// ErrOutOfBounds indicates a read or write extends beyond the device length.
	ErrOutOfBounds = errors.New("blockdev: read/write beyond device length")

	// ErrBadFormat indicates serialized data is malformed and cannot be deserialized.
	ErrBadFormat = errors.New("blockdev: malformed serialized data")
)
