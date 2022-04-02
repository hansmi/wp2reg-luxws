package luxwsclient

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"github.com/hansmi/wp2reg-luxws/luxws"
)

func TestResponseUnmarshalIgnore(t *testing.T) {
	msg := struct {
		XMLName xml.Name
	}{}

	if err := responseUnmarshal([]byte("<valid></valid>"), &msg, "name"); err != luxws.ErrIgnore {
		t.Errorf("Valid XML wasn't ignored: %v", err)
	}
}

func newTestClient(t *testing.T, handleRoundTrip func(string) (string, error)) *Client {
	var upgrader websocket.Upgrader

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Connection upgrade failed: %v", err)
			return
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				t.Errorf("ReadMessage() failed: %v", err)
				break
			}

			response, err := handleRoundTrip(string(message))
			if err != nil {
				t.Errorf("handleRoundTrip(%q) failed: %v", message, err)
				break
			}

			if err = c.WriteMessage(mt, []byte(response)); err != nil {
				t.Errorf("WriteMessage(%q) failed: %v", message, err)
				break
			}
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	c, err := Dial(ctx, serverURL.Host, WithLogFunc(t.Logf))
	if err != nil {
		t.Fatalf("Dial(%q) failed: %v", serverURL.Host, err)
	}

	return c
}

func TestLogin(t *testing.T) {
	for _, tc := range []struct {
		name            string
		handleRoundTrip func(string) (string, error)
		want            *NavRoot
		wantErr         error
	}{
		{
			name: "simple",
			handleRoundTrip: func(req string) (string, error) {
				if req == "LOGIN;1234" {
					return `<Navigation id="0x41c123c8"><item id="0x41123678"><name>Test</name></item></Navigation>`, nil
				}

				return "<unknown></unknown>", nil
			},
			want: &NavRoot{
				XMLName: xml.Name{Local: "Navigation"},
				ID:      "0x41c123c8",
				Items: []NavItem{
					{
						ID:   "0x41123678",
						Name: "Test",
					},
				},
			},
		},
		{
			name: "wrong format",
			handleRoundTrip: func(string) (string, error) {
				return "<definitely<not<xml", nil
			},
			wantErr: &xml.SyntaxError{
				Line: 1,
				Msg:  "expected attribute name in element",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			t.Cleanup(cancel)

			c := newTestClient(t, tc.handleRoundTrip)

			if got, err := c.Login(ctx, "1234"); tc.wantErr != nil {
				if diff := cmp.Diff(tc.wantErr, err); diff != "" {
					t.Errorf("Login() error difference (-want +got):\n%s", diff)
				}
			} else if err != nil {
				t.Errorf("Login failed: %v", err)
			} else if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Navigation difference (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGet(t *testing.T) {
	for _, tc := range []struct {
		name            string
		handleRoundTrip func(string) (string, error)
		want            *ContentRoot
		wantErr         error
	}{
		{
			name: "simple",
			handleRoundTrip: func(req string) (string, error) {
				if req == "GET;0x1234" {
					return `
<?xml version="1.0"?>
<Content>
  <item id="0x41c14bc4">
    <name>Temperaturen</name>
    <item id="0x41c18d14">
      <name>Min. Rückl.Solltemp.</name>
      <min>150</min>
      <max>300</max>
      <step>5</step>
      <unit>°C</unit>
      <div>10.00</div>
      <raw>150</raw>
      <value>15.0°C</value>
    </item>
    <name>Temperaturen</name>
  </item>
  <item id="0x41c15184">
    <name>System Einstellung</name>
    <item id="0x41c17844">
      <name>Warmw. Nachh. max</name>
      <min>10</min>
      <max>100</max>
      <step>5</step>
      <unit> h</unit>
      <div>10.00</div>
      <raw>50</raw>
      <value>5.0 h</value>
    </item>
    <item id="0x41c1998c">
      <name>Smart Grid</name>
      <value>Nein</value>
    </item>
    <item id="0x41c19a84">
      <name>Regelung MK1</name>
      <option value="0">schnell</option>
      <option value="1">mittel</option>
      <option value="2">langsam</option>
      <raw>0</raw>
      <value>schnell</value>
    </item>
    <item id="0x41c19bc4">
      <name>Max Leistung ZWE</name>
      <value>9.0 kW</value>
    </item>
    <name>Einstellungen</name>
  </item>
</Content>`, nil
				}

				return "<unknown></unknown>", nil
			},
			want: &ContentRoot{
				XMLName: xml.Name{Local: "Content"},
				Items: []ContentItem{
					{
						ID:   "0x41c14bc4",
						Name: "Temperaturen",
						Items: []ContentItem{
							{
								ID:    "0x41c18d14",
								Name:  "Min. Rückl.Solltemp.",
								Min:   String("150"),
								Max:   String("300"),
								Step:  String("5"),
								Unit:  String("°C"),
								Div:   String("10.00"),
								Raw:   String("150"),
								Value: String("15.0°C"),
							},
						},
					},
					{
						ID:   "0x41c15184",
						Name: "Einstellungen",
						Items: []ContentItem{
							{
								ID:    "0x41c17844",
								Name:  "Warmw. Nachh. max",
								Min:   String("10"),
								Max:   String("100"),
								Step:  String("5"),
								Unit:  String(" h"),
								Div:   String("10.00"),
								Raw:   String("50"),
								Value: String("5.0 h"),
							},
							{
								ID:    "0x41c1998c",
								Name:  "Smart Grid",
								Value: String("Nein"),
							},
							{
								ID:   "0x41c19a84",
								Name: "Regelung MK1",
								Options: []ContentItemOption{
									{Value: "0", Name: "schnell"},
									{Value: "1", Name: "mittel"},
									{Value: "2", Name: "langsam"},
								},
								Raw:   String("0"),
								Value: String("schnell"),
							},
							{
								ID:    "0x41c19bc4",
								Name:  "Max Leistung ZWE",
								Value: String("9.0 kW"),
							},
						},
					},
				},
			},
		},
		{
			name: "wrong format",
			handleRoundTrip: func(string) (string, error) {
				return "<definitely<not<xml", nil
			},
			wantErr: &xml.SyntaxError{
				Line: 1,
				Msg:  "expected attribute name in element",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			t.Cleanup(cancel)

			c := newTestClient(t, tc.handleRoundTrip)

			if got, err := c.Get(ctx, "0x1234"); tc.wantErr != nil {
				if diff := cmp.Diff(tc.wantErr, err); diff != "" {
					t.Errorf("Get() error difference (-want +got):\n%s", diff)
				}
			} else if err != nil {
				t.Errorf("Get failed: %v", err)
			} else if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Content difference (-want +got):\n%s", diff)
			}
		})
	}
}
