# blockdev

A copy-on-write, in-memory block device backend in Go.

> **Status: v0.1.0 in development.** API stabilizing; behavior in flux.
> See [`IMPLEMENTATION_PLAN.md`](IMPLEMENTATION_PLAN.md) for the staged build.

## What

`blockdev` presents a fixed-size byte array as a sequence of 4096-byte blocks.
Writes are captured in an in-memory overlay layered above an immutable base.
Reads consult the overlay first, falling through to the base when a block has
not been written. `Serialize` produces a compact byte representation of just
the overlay; `Deserialize` reconstructs a device from those bytes plus the
original base.

A natural fit for sandbox snapshot/resume — think E2B microVMs where many
sandboxes share a base filesystem image and you only want to persist the
diffs.

## Install

```sh
go get github.com/codingminions/blockdev
```

Requires Go 1.22 or later. No external dependencies.

## Docs

See [`docs/`](docs/) — architecture, API reference, format spec, ADRs.

## License

[MIT](LICENSE).
