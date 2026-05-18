package blockdev

// BlockSize is the unit of read, write, and tracking in bytes. All offsets
// and lengths passed to ReadAt and WriteAt must be multiples of BlockSize.
const BlockSize = 4096

// BlockDevice is an in-memory block device with copy-on-write semantics.
// Construct one with New. Safe for concurrent use.
type BlockDevice struct{}
