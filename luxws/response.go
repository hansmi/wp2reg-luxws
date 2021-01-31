package luxws

import (
	"errors"
	"sync"
)

// ErrIgnore is the error used by response handler callbacks when a given
// message needs to be ignored.
var ErrIgnore = errors.New("ignore response")

// ResponseHandlerFunc is the prototype for response handler callbacks.
type ResponseHandlerFunc func([]byte) error

type responseHandler struct {
	mu   sync.Mutex
	done chan struct{}
	err  error
	fn   ResponseHandlerFunc
}

func newResponseHandler(fn ResponseHandlerFunc) *responseHandler {
	return &responseHandler{
		done: make(chan struct{}),
		fn:   fn,
	}
}

func (h *responseHandler) Done() <-chan struct{} {
	h.mu.Lock()
	d := h.done
	h.mu.Unlock()
	return d
}

func (h *responseHandler) Err() error {
	h.mu.Lock()
	err := h.err
	h.mu.Unlock()
	return err
}

func (h *responseHandler) Handle(payload []byte) {
	h.mu.Lock()
	select {
	case <-h.done:
		h.mu.Unlock()
		return
	default:
	}

	fn := h.fn
	h.mu.Unlock()

	if err := fn(payload); err == nil || !errors.Is(err, ErrIgnore) {
		h.mu.Lock()
		h.err = err
		close(h.done)
		h.mu.Unlock()
	}
}
