package luxwsclient

import (
	"context"
	"encoding/xml"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hansmi/wp2reg-luxws/luxws"
)

type fakeTransport struct {
	handleRoundTrip func(string) (string, error)
}

func (*fakeTransport) Close() error {
	return nil
}

func (tr *fakeTransport) RoundTrip(ctx context.Context, req string, fn luxws.ResponseHandlerFunc) error {
	if err := fn([]byte("<valid></valid>")); err != luxws.ErrIgnore {
		return fmt.Errorf("valid XML wasn't ignored: %v", err)
	}

	response, err := tr.handleRoundTrip(req)
	if err != nil {
		return err
	}

	return fn([]byte(response))
}

func newFakeClient(t *testing.T, tr *fakeTransport) *Client {
	t.Helper()

	return &Client{
		t: tr,
	}
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

			c := newFakeClient(t, &fakeTransport{
				handleRoundTrip: tc.handleRoundTrip,
			})

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

			c := newFakeClient(t, &fakeTransport{
				handleRoundTrip: tc.handleRoundTrip,
			})

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
