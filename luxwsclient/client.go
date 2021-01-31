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

func xmlUnmarshal(data []byte, v interface{}) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charset.NewReaderLabel
	return dec.Decode(v)
}

func responseUnmarshal(data []byte, v interface{}, wantLocalName string) error {
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

// Client is a wrapper around an underlying LuxWS connection.
type Client struct {
	t transport
}

// Dial connects to a LuxWS server. The address must have the format
// "<host>:<port>" (see net.JoinHostPort). Use the context to establish
// a timeout.
//
// IDs returned by the server are unique to each connection.
func Dial(ctx context.Context, address string) (*Client, error) {
	t, err := luxws.Dial(ctx, address)
	if err != nil {
		return nil, err
	}

	return &Client{
		t: t,
	}, nil
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
