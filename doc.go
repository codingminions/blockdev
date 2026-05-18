// Package blockdev implements an in-memory block device backend with
// copy-on-write semantics.
//
// A BlockDevice presents a fixed-size byte array as a sequence of 4096-byte
// blocks. Writes are captured in an in-memory overlay layered above an
// immutable base. Reads consult the overlay first, falling through to the
// base when a block has not been written.
//
// Serialize produces a compact byte representation of just the overlay;
// Deserialize reconstructs a BlockDevice from a serialized overlay plus the
// original base. This makes blockdev a natural fit for sandbox snapshot and
// resume systems where many sandboxes share an identical base filesystem
// image and storing per-sandbox diffs is preferred over storing per-sandbox
// copies.
//
// Block size is fixed at 4096 bytes (see BlockSize). Offsets and lengths
// passed to ReadAt and WriteAt must be multiples of BlockSize. The initial
// base passed to New must be a multiple of BlockSize and must not be mutated
// by the caller after construction.
//
// All public methods are safe for concurrent use.
package blockdev
