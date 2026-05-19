---

## Current state

**Done in v0.1.0**

- `New`, `ReadAt`, `WriteAt`, `Serialize`, `Deserialize` — functionally complete
- Implements `io.ReaderAt` and `io.WriterAt` (compile-time asserted)
- Functional options: `WithName`, `WithObserver`
- 28 black-box tests in `tests/e2e/`, all pass under `-race`
- 14 benchmarks in `tests/benchmarks/`
- Wire format locked by golden-bytes test
- CI: `build` + `vet` + `gofmt` + `test -race` on Go 1.22 + 1.23
- Nightly benchmark chart on GitHub Pages

**Deliberately out of scope**

- NBD server integration (caller wires this via `io.ReaderAt`/`io.WriterAt`)
- FUSE filesystem
- Sharded mutexes
- Overlay compression / encryption
- `Codec` interface for pluggable formats
- Fuzz tests for `Deserialize`
- Snapshot diff between two serialized blobs
- Baked-in Prometheus metrics — use `WithObserver`

---
