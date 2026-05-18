// Package blockdev is an in-memory copy-on-write block device. Many sandboxes
// share one read-only base in RAM while each captures its own writes in an
// overlay. Serialize emits only the overlay so per-sandbox snapshots stay
// small, regardless of base size.
//
// All public methods are safe for concurrent use.
package blockdev
