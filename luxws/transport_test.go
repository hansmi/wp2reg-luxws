package luxws

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestErrClosed(t *testing.T) {
	if !isErrClosed(ErrClosed) {
		t.Errorf("isErrClosed(ErrClosed) is not true")
	}
}

type cannedMessage struct {
	messageType int
	payload     []byte
	err         error
}

type handleWriteFunc func([]byte, chan<- cannedMessage) error

type fakeConn struct {
	mu          sync.Mutex
	logf        func(string, ...interface{})
	closed      chan struct{}
	handleWrite handleWriteFunc
	outgoing    chan cannedMessage
}

func newFakeConn(t *testing.T) *fakeConn {
	t.Helper()

	return &fakeConn{
		logf:     t.Logf,
		outgoing: make(chan cannedMessage, 16),
		closed:   make(chan struct{}),
	}
}

func (c *fakeConn) LocalAddr() net.Addr {
	return &net.IPAddr{}
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return &net.IPAddr{}
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	c.logf("SetWriteDeadline(%v)", t)

	select {
	case <-c.closed:
		return ErrClosed
	default:
	}

	return nil
}

func (c *fakeConn) WriteMessage(messageType int, payload []byte) error {
	c.logf("WriteMessage(%d, %q)", messageType, payload)

	select {
	case <-c.closed:
		return ErrClosed
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.handleWrite(payload, c.outgoing)
}

func (c *fakeConn) ReadMessage() (int, []byte, error) {
	c.logf("ReadMessage")

	select {
	case <-c.closed:
		return 0, nil, ErrClosed
	case msg := <-c.outgoing:
		return msg.messageType, msg.payload, msg.err
	}
}

func (c *fakeConn) Close() error {
	c.logf("Close")

	select {
	case <-c.closed:
		return ErrClosed
	default:
	}

	close(c.closed)

	return nil
}

func newFakeTransport(t *testing.T) (*fakeConn, *Transport) {
	t.Helper()

	fc := newFakeConn(t)
	tr := newTransport(fc)

	t.Cleanup(func() {
		tr.Close()
	})

	return fc, tr
}

func TestClose(t *testing.T) {
	_, tr := newFakeTransport(t)

	if err := tr.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if err := tr.Close(); !isErrClosed(err) {
		t.Errorf("second Close() returned unexpected value: %v", err)
	}
}

func TestRoundTripAfterClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	_, tr := newFakeTransport(t)

	if err := tr.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if err := tr.RoundTrip(ctx, "", nil); !isErrClosed(err) {
		t.Errorf("RoundTrip() after Close() didn't fail as expected: %v", err)
	}
}

func TestRoundTripAfterFailedRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc, tr := newFakeTransport(t)

	errTest := errors.New("test")

	fc.handleWrite = func(payload []byte, out chan<- cannedMessage) error {
		if string(payload) == "first" {
			out <- cannedMessage{
				messageType: websocket.TextMessage,
				err:         errTest,
			}
		}

		return nil
	}

	if err := tr.RoundTrip(ctx, "first", nil); !errors.Is(err, errTest) {
		t.Errorf("RoundTrip() failed: %v", err)
	}

	if err := tr.RoundTrip(ctx, "", nil); !errors.Is(err, errTest) {
		t.Errorf("RoundTrip() after failed read didn't fail as expected: %v", err)
	}
}

func TestRoundTripAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc, tr := newFakeTransport(t)

	fc.handleWrite = func(payload []byte, out chan<- cannedMessage) error {
		out <- cannedMessage{
			messageType: websocket.TextMessage,
		}

		return nil
	}

	cancel()

	if err := tr.RoundTrip(ctx, "", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("RoundTrip() didn't fail due to cancelled context: %v", err)
	}
}

func TestCancelDuringWrite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc, tr := newFakeTransport(t)

	fc.handleWrite = func(payload []byte, out chan<- cannedMessage) error {
		out <- cannedMessage{
			messageType: websocket.TextMessage,
		}

		cancel()

		return nil
	}

	if err := tr.RoundTrip(ctx, "", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("RoundTrip() didn't fail due to cancelled context: %v", err)
	}
}

func TestResponseHandlerError(t *testing.T) {
	errTest := errors.New("test error")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc, tr := newFakeTransport(t)

	fc.handleWrite = func(payload []byte, out chan<- cannedMessage) error {
		out <- cannedMessage{
			messageType: websocket.TextMessage,
			payload:     []byte("test"),
		}

		return nil
	}

	if err := tr.RoundTrip(ctx, "req", func(payload []byte) error {
		return errTest
	}); !errors.Is(err, errTest) {
		t.Errorf("RoundTrip() failed: %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc, tr := newFakeTransport(t)

	fc.handleWrite = func(payload []byte, out chan<- cannedMessage) error {
		out <- cannedMessage{
			messageType: websocket.PingMessage,
		}

		for _, i := range strings.Split(string(payload), ",") {
			out <- cannedMessage{
				messageType: websocket.TextMessage,
				payload:     []byte(i),
			}
		}

		return nil
	}

	if err := tr.RoundTrip(ctx, "foobar", func(payload []byte) error {
		switch resp := string(payload); resp {
		case "foobar":
			// nothing

		default:
			t.Errorf("Unexpected response %q", resp)
		}

		return nil
	}); err != nil {
		t.Errorf("RoundTrip() failed: %v", err)
	}

	if err := tr.RoundTrip(ctx, "ignore,ignore,response", func(payload []byte) error {
		switch resp := string(payload); resp {
		case "response":
			// nothing

		case "ignore":
			return ErrIgnore

		default:
			t.Errorf("Unexpected response %q", resp)
		}

		return nil
	}); err != nil {
		t.Errorf("RoundTrip() failed: %v", err)
	}
}
