---

## Major design decisions

One-line rationale here; ADR linked where the full reasoning lives.

| Decision | Choice | Why |
|---|---|---|
| Block size | `4096` bytes | OS page + NBD sector convention |
| Overlay | `map[int64][]byte` + `sync.RWMutex` | O(1) lookup, parallel reads  |
| Base immutability | Caller contract, not enforced | Sharing one base across many sandboxes |
| Constructor | Functional options | Add knobs later without breaking callers |
| Serialization | `[8B BE block#][4096B data]`, sorted | Smallest overhead, deterministic |
| Errors | Sentinel + `fmt.Errorf` wrap | `errors.Is` works without parsing strings  |
| Validation order | Bounds → alignment | Range vs shape bugs map to different sentinels |
| Observer | Sync, post-op, outside locks | Honest latency, panic-safe  |
| `Deserialize` checks | Strict (length + range + dedup + order) | Catches storage corruption |
| Empty `Serialize()` | Returns `nil` | Zero-allocation fresh-device path |
| Dependencies | stdlib only | Clean module |
