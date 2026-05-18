// Package validate holds pure parameter predicates. Their errors are signals,
// not sentinels — callers wrap them with their own public errors so the
// internal package never appears in user-facing error chains.
package validate

import "errors"

const blockSize = 4096

var (
	errNotAligned = errors.New("validate: not block-aligned")
	errOutOfRange = errors.New("validate: out of range")
)

func Alignment(off int64, length int) error {
	if off%blockSize != 0 || int64(length)%blockSize != 0 {
		return errNotAligned
	}
	return nil
}

func Bounds(off int64, length int, deviceSize int64) error {
	if off < 0 || length < 0 || off+int64(length) > deviceSize {
		return errOutOfRange
	}
	return nil
}
