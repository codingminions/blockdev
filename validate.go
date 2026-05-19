package blockdev

// Both checks return the public sentinel directly so the caller's fmt.Errorf
// wrap places the right ErrMisaligned/ErrOutOfBounds in the chain without
// having to map opaque internal errors.

func checkAlignment(off int64, length int) error {
	if off%BlockSize != 0 || int64(length)%BlockSize != 0 {
		return ErrMisaligned
	}
	return nil
}

func checkBounds(off int64, length int, deviceSize int64) error {
	if off < 0 || length < 0 || off+int64(length) > deviceSize {
		return ErrOutOfBounds
	}
	return nil
}
