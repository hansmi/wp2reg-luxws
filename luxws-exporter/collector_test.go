package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hansmi/wp2reg-luxws/luxwsclient"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type adapter struct {
	t       *testing.T
	c       *collector
	fn      contentCollectFunc
	content *luxwsclient.ContentRoot
}

func (a *adapter) Describe(ch chan<- *prometheus.Desc) {
	a.c.Describe(ch)
}

func (a *adapter) Collect(ch chan<- prometheus.Metric) {
	if err := a.fn(ch, a.content); err != nil {
		a.t.Errorf("Collection failed: %v", err)
	}
}

func (a *adapter) collectAndCompare(t *testing.T, want string) {
	t.Helper()

	if err := testutil.CollectAndCompare(a, strings.NewReader(want)); err != nil {
		t.Error(err)
	}
}

func TestCollectWebSocketParts(t *testing.T) {
	c := newCollector(collectorOpts{
		terms: luxwslang.German,
		loc:   time.UTC,
	})

	for _, tc := range []struct {
		name  string
		fn    contentCollectFunc
		input *luxwsclient.ContentRoot
		want  string
	}{
		{
			name: "info empty",
			fn:   c.collectInfo,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Anlagenstatus",
					},
				},
			},
			want: `
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="",swversion=""} 1
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode=""} 1
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit=""} 0
`,
		},
		{
			name: "info full",
			fn:   c.collectInfo,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Anlagenstatus",
						Items: []luxwsclient.ContentItem{
							{Name: "Wärmepumpen Typ", Value: luxwsclient.String("typeA")},
							{Name: "Softwarestand", Value: luxwsclient.String("v1.2.3")},
							{Name: "Betriebszustand", Value: luxwsclient.String("running")},
							{Name: "Leistung Ist", Value: luxwsclient.String("999 kWh")},
							{Name: "Wärmepumpen Typ", Value: luxwsclient.String("typeB")},
						},
					},
				},
			},
			want: `
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="typeA, typeB",swversion="v1.2.3"} 1
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode="running"} 1
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit="kWh"} 999
`,
		},
		{
			name: "temperatures empty",
			fn:   c.collectTemperatures,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Temperaturen",
					},
				},
			},
			want: `
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="",unit=""} 0
`,
		},
		{
			name: "temperatures full",
			fn:   c.collectTemperatures,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Temperaturen",
						Items: []luxwsclient.ContentItem{
							{Name: "Wasser", Value: luxwsclient.String("10°C")},
							{Name: "Aussen", Value: luxwsclient.String("100°C")},
							{Name: "Etwas", Value: luxwsclient.String("1 K")},
						},
					},
				},
			},
			want: `
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="Aussen",unit="degC"} 100
luxws_temperature{name="Etwas",unit="K"} 1
luxws_temperature{name="Wasser",unit="degC"} 10
`,
		},
		{
			name: "op duration empty",
			fn:   c.collectOperatingDuration,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Betriebsstunden",
					},
				},
			},
			want: `
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name=""} 0
`,
		},
		{
			name: "op duration full",
			fn:   c.collectOperatingDuration,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Betriebsstunden",
						Items: []luxwsclient.ContentItem{
							{Name: "On\tspace", Value: luxwsclient.String("1h")},
							{Name: "Heat", Value: luxwsclient.String("1:2:3")},
							{Name: "Impulse xyz", Value: luxwsclient.String("")},
						},
					},
				},
			},
			want: `
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name="Heat"} 3723
luxws_operating_duration_seconds{name="On space"} 3600
`,
		},
		{
			name: "op elapsed empty",
			fn:   c.collectElapsedTime,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Ablaufzeiten",
					},
				},
			},
			want: `
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name=""} 0
`,
		},
		{
			name: "op elapsed full",
			fn:   c.collectElapsedTime,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Ablaufzeiten",
						Items: []luxwsclient.ContentItem{
							{Name: "a", Value: luxwsclient.String("1h")},
							{Name: "b", Value: luxwsclient.String("1:2")},
						},
					},
				},
			},
			want: `
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name="a"} 3600
luxws_elapsed_duration_seconds{name="b"} 3720
`,
		},
		{
			name: "inputs empty",
			fn:   c.collectInputs,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Eingänge",
					},
				},
			},
			want: `
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="",unit=""} 0
`,
		},
		{
			name: "inputs full",
			fn:   c.collectInputs,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Eingänge",
						Items: []luxwsclient.ContentItem{
							{Name: "temp a", Value: luxwsclient.String("20 °C")},
							{Name: "pressure", Value: luxwsclient.String("3 bar")},
						},
					},
				},
			},
			want: `
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="temp a",unit="degC"} 20
luxws_input{name="pressure",unit="bar"} 3
`,
		},
		{
			name: "outputs empty",
			fn:   c.collectOutputs,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Ausgänge",
					},
				},
			},
			want: `
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="",unit=""} 0
`,
		},
		{
			name: "outputs full",
			fn:   c.collectOutputs,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Ausgänge",
						Items: []luxwsclient.ContentItem{
							{Name: "rot", Value: luxwsclient.String("200 RPM")},
							{Name: "pwm", Value: luxwsclient.String("33 %")},
						},
					},
				},
			},
			want: `
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="pwm",unit="pct"} 33
luxws_output{name="rot",unit="rpm"} 200
`,
		},
		{
			name: "supplied heat empty",
			fn:   c.collectSuppliedHeat,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Wärmemenge",
					},
				},
			},
			want: `
# HELP luxws_supplied_heat Supplied heat
# TYPE luxws_supplied_heat gauge
luxws_supplied_heat{name="",unit=""} 0
`,
		},
		{
			name: "supplied heat full",
			fn:   c.collectSuppliedHeat,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Wärmemenge",
						Items: []luxwsclient.ContentItem{
							{Name: "water", Value: luxwsclient.String("200 kW")},
							{Name: "ice", Value: luxwsclient.String("100 kW")},
						},
					},
				},
			},
			want: `
# HELP luxws_supplied_heat Supplied heat
# TYPE luxws_supplied_heat gauge
luxws_supplied_heat{name="ice",unit="kW"} 100
luxws_supplied_heat{name="water",unit="kW"} 200
`,
		},
		{
			name: "latest error empty",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Fehlerspeicher",
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason=""} 0
`,
		},
		{
			name: "latest error",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Fehlerspeicher",
						Items: []luxwsclient.ContentItem{
							{Name: "02.02.11 08:00:00", Value: luxwsclient.String("aaa")},
							{Name: "03.04.14 23:00:00", Value: luxwsclient.String("bbb")},
							{Name: "01.01.10 09:00:11", Value: luxwsclient.String("aaa")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason="aaa"} 1296633600
luxws_latest_error{reason="bbb"} 1396566000
`,
		},
		{
			name: "latest error with empty rows",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Fehlerspeicher",
						Items: []luxwsclient.ContentItem{
							{Name: "----", Value: luxwsclient.String("placeholder")},
							{Name: "08.11.21 21:40:09", Value: luxwsclient.String("text")},
							{Name: "----", Value: luxwsclient.String("----")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason="text"} 1636407609
`,
		},
		{
			name: "latest switch-off empty",
			fn:   c.collectLatestSwitchOff,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Abschaltungen",
					},
				},
			},
			want: `
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason=""} 0
`,
		},
		{
			name: "latest switch-off",
			fn:   c.collectLatestSwitchOff,
			input: &luxwsclient.ContentRoot{
				Items: []luxwsclient.ContentItem{
					{
						Name: "Abschaltungen",
						Items: []luxwsclient.ContentItem{
							{Name: "02.02.19 08:00:00", Value: luxwsclient.String("aaa")},
							{Name: "03.04.20 23:00:00", Value: luxwsclient.String("bbb")},
							{Name: "01.01.20 09:00:11", Value: luxwsclient.String("aaa")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason="aaa"} 1577869211
luxws_latest_switchoff{reason="bbb"} 1585954800
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &adapter{t, c, tc.fn, tc.input}
			a.collectAndCompare(t, tc.want)
		})
	}
}

type httpAdapter struct {
	t       *testing.T
	c       *collector
	collect func(ch chan<- prometheus.Metric)
}

func (a *httpAdapter) Describe(ch chan<- *prometheus.Desc) {
	a.c.Describe(ch)
}

func (a *httpAdapter) Collect(ch chan<- prometheus.Metric) {
	a.collect(ch)
}

func (a *httpAdapter) collectAndCompare(t *testing.T, want string) {
	t.Helper()

	if err := testutil.CollectAndCompare(a, strings.NewReader(want)); err != nil {
		t.Error(err)
	}
}

func TestCollectHTTP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", "Mon, 02 Jan 2006 15:04:05 GMT")
	}))
	t.Cleanup(server.Close)

	c := newCollector(collectorOpts{
		terms: luxwslang.German,
		loc:   time.UTC,
	})

	if serverURL, err := url.Parse(server.URL); err != nil {
		t.Error(err)
	} else {
		c.httpAddress = serverURL.Host
	}

	a := &httpAdapter{t, c, func(ch chan<- prometheus.Metric) {
		if err := c.collectHTTP(ctx, ch); err != nil {
			t.Errorf("Collection failed: %v", err)
		}
	}}
	a.collectAndCompare(t, `
# HELP luxws_node_time_seconds System time in seconds since epoch (1970)
# TYPE luxws_node_time_seconds gauge
luxws_node_time_seconds 1136214245
`)
}
