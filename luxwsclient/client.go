package luxwsclient

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"reflect"
	"strings"

	"github.com/hansmi/wp2reg-luxws/luxws"
	"golang.org/x/net/html/charset"
)

func xmlUnmarshal(data []byte, v any) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charset.NewReaderLabel
	return dec.Decode(v)
}

func responseUnmarshal(data []byte, v any, wantLocalName string) error {
	target := reflect.ValueOf(v)

	if target.Kind() != reflect.Ptr {
		return errors.New("value must be pointer")
	}

	doc := reflect.New(target.Elem().Type())

	if err := xmlUnmarshal(data, doc.Interface()); err != nil {
		return err
	}

	elemName := doc.Elem().FieldByName("XMLName").Interface().(xml.Name)

	// XML is case-sensitive, but the JavaScript code ignores casing in a few
	// places
	if strings.ToLower(elemName.Local) == wantLocalName {
		target.Elem().Set(doc.Elem())
		return nil
	}

	return luxws.ErrIgnore
}

// String stores s in a new string value and returns a pointer to it.
func String(s string) *string {
	return &s
}

type transport interface {
	RoundTrip(context.Context, string, luxws.ResponseHandlerFunc) error
	Close() error
}

// Option is the type of options for clients.
type Option func(*Client)

// LogFunc describes a logging function (e.g. log.Printf).
type LogFunc func(format string, v ...any)

// WithLogFunc supplies a logging function to the client.
func WithLogFunc(logf LogFunc) Option {
	return func(c *Client) {
		c.logf = logf
	}
}

// Client is a wrapper around an underlying LuxWS connection.
type Client struct {
	logf LogFunc
	t    transport
}

// Dial connects to a LuxWS server. The address must have the format
// "<host>:<port>" (see net.JoinHostPort). Use the context to establish
// a timeout.
//
// IDs returned by the server are unique to each connection.
func Dial(ctx context.Context, address string, opts ...Option) (*Client, error) {
	var err error

	c := &Client{
		logf: func(string, ...any) {},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.t, err = luxws.Dial(ctx, address, luxws.WithLogFunc(luxws.LogFunc(c.logf))); err != nil {
		return nil, err
	}

	return c, nil
}

// Close closes the underlying network connection.
func (c *Client) Close() error {
	return c.t.Close()
}

// Login sends a "LOGIN" command. The navigation structure is returned.
func (c *Client) Login(ctx context.Context, password string) (*NavRoot, error) {
	var result NavRoot

	return &result, c.t.RoundTrip(ctx, "LOGIN;"+password, func(payload []byte) error {
		return responseUnmarshal(payload, &result, "navigation")
	})
}

// Get sends a "GET" command. The page content is returned.
func (c *Client) Get(ctx context.Context, id string) (*ContentRoot, error) {
	var result ContentRoot

	return &result, c.t.RoundTrip(ctx, "GET;"+id, func(payload []byte) error {
		return responseUnmarshal(payload, &result, "content")
	})
}
