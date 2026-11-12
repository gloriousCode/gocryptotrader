package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var errChannelBufferFull = errors.New("channel buffer is full")

// Relay defines a channel relay for messages
type Relay struct {
	C      <-chan Payload
	comm   chan Payload
	direct atomic.Pointer[func(context.Context, any) (bool, error)]
}

// Payload represents a relayed message with a context
type Payload struct {
	Ctx  common.FrozenContext
	Data any
}

// NewRelay creates a new Relay instance with a specified buffer size
func NewRelay(buffer uint) *Relay {
	if buffer == 0 {
		panic("buffer size must be greater than 0")
	}
	comm := make(chan Payload, buffer)
	return &Relay{comm: comm, C: comm}
}

// Send sends a message to the channel receiver
// This is non-blocking and returns an error if the channel buffer is full
func (r *Relay) Send(ctx context.Context, data any) error {
	if handler := r.direct.Load(); handler != nil {
		if handled, err := (*handler)(ctx, data); handled || err != nil {
			return err
		}
	}
	select {
	case r.comm <- Payload{Ctx: common.FreezeContext(ctx), Data: data}:
		return nil
	default:
		return fmt.Errorf("%w: failed to relay <%T>", errChannelBufferFull, data)
	}
}

// Close closes the relay channel
func (r *Relay) Close() {
	close(r.comm)
}

// SetDirectHandler optionally routes selected notifications before the FIFO relay.
// The callback must be bounded and non-blocking. Returning false preserves normal
// FIFO delivery; errors propagate to the producer without silently dropping data.
// A nil handler restores the default relay behaviour.
func (r *Relay) SetDirectHandler(handler func(context.Context, any) (bool, error)) {
	if handler == nil {
		r.direct.Store(nil)
		return
	}
	r.direct.Store(&handler)
}
