package blockdev

import "time"

// Option is the functional-option type. Variadic in New keeps the constructor
// extensible without forcing a breaking change every time a knob is added.
type Option func(*config)

type config struct {
	name     string
	observer func(Event)
}

// WithName tags the device so a single observer can distinguish events from
// many devices (one per agent in the demo).
func WithName(name string) Option {
	return func(c *config) { c.name = name }
}

// WithObserver installs a synchronous callback invoked after every public
// operation. Panics from the callback are recovered so user instrumentation
// cannot crash the I/O path.
func WithObserver(fn func(Event)) Option {
	return func(c *config) { c.observer = fn }
}

// Op identifies the operation that produced an Event.
type Op uint8

const (
	OpRead Op = iota + 1
	OpWrite
	OpSerialize
	OpDeserialize
)

func (o Op) String() string {
	switch o {
	case OpRead:
		return "read"
	case OpWrite:
		return "write"
	case OpSerialize:
		return "serialize"
	case OpDeserialize:
		return "deserialize"
	default:
		return "unknown"
	}
}

// Event is the per-operation record passed to WithObserver's callback.
// Length and Blocks have op-specific meaning: for Read/Write they describe
// the request; for Serialize/Deserialize they describe the blob size and the
// number of changed-block entries.
type Event struct {
	Op       Op
	Device   string
	Offset   int64
	Length   int
	Blocks   int
	Duration time.Duration
	Err      error
}
