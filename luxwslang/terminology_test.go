package luxwslang

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestComplete(t *testing.T) {
	for _, terms := range All() {
		t.Run(terms.ID, func(t *testing.T) {
			v := reflect.ValueOf(terms).Elem()

			for idx := 0; idx < v.NumField(); idx++ {
				field := v.Field(idx)
				structField := v.Type().Field(idx)

				if !field.CanInterface() && structField.Name == "timestampFormat" {
					continue
				}

				var err error

				switch val := field.Interface().(type) {
				case string:
					if val == "" {
						err = errors.New("empty string")
					}
				case *regexp.Regexp:
					if val == nil {
						err = errors.New("nil regexp")
					}
				default:
					err = fmt.Errorf("unknown type %v", field.Type())
				}

				if err != nil {
					t.Errorf("field %q: %v", structField.Name, err)
				}
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	locBerlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Error(err)
	}

	for _, tc := range []struct {
		terms *Terminology
		input string
		loc   *time.Location
		want  time.Time
	}{
		{
			terms: German,
			input: "03.02.18 12:34:56",
			loc:   locBerlin,
			want:  time.Date(2018, time.March, 3, 12, 34, 56, 0, locBerlin),
		},
	} {
		t.Run(tc.terms.ID+" "+tc.input, func(t *testing.T) {
			if got, err := tc.terms.ParseTimestamp(tc.input, tc.loc); err != nil {
				t.Errorf("ParseTimestamp(%q, %v) failed: %v", tc.input, tc.loc, err)
			} else if got.Equal(tc.want) {
				t.Errorf("ParseTimestamp(%q, %v) = %v, want %v", tc.input, tc.loc, got, tc.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	for _, tc := range []struct {
		terms   *Terminology
		input   string
		want    string
		wantErr bool
	}{
		{terms: German, input: "1:2:3", want: "1h2m3s"},
		{terms: German, input: "23:59", want: "23h59m"},
		{terms: German, input: "12h", want: "12h"},
		{terms: German, input: "-100h", want: "-100h"},
		{terms: German, input: "  -23:1:2\n", want: "-23h1m2s"},
		{terms: German, input: "-1:0:0", want: "-1h"},
		{terms: German, input: "-100", want: "-100h"},
		{terms: German, input: "123", want: "123h"},
		{terms: German, input: "0:-1:0", wantErr: true},
		{terms: German, input: "0:0:-1", wantErr: true},
	} {
		t.Run(tc.terms.ID+" "+tc.input, func(t *testing.T) {
			got, err := tc.terms.ParseDuration(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("parseDuration(%q) didn't fail", tc.input)
				}
			} else if err != nil {
				t.Errorf("ParseDuration(%q) failed: %v", tc.input, err)
			} else if want, err := time.ParseDuration(tc.want); err != nil {
				t.Error(err)
			} else if got != want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseMeasurement(t *testing.T) {
	for _, tc := range []struct {
		terms    *Terminology
		input    string
		want     float64
		wantUnit string
		wantErr  bool
	}{
		{terms: German, input: "", wantErr: true},
		{terms: German, input: "1.23", wantErr: true},
		{terms: German, input: "100m", wantErr: true},
		{terms: German, input: "1l", wantErr: true},
		{terms: German, input: "1.11°C", want: 1.11, wantUnit: "degC"},
		{terms: German, input: "2.22 °C", want: 2.22, wantUnit: "degC"},
		{terms: German, input: "90 K", want: 90, wantUnit: "K"},
		{terms: German, input: "0.0 bar", want: 0, wantUnit: "bar"},
		{terms: German, input: "-100bar", want: -100, wantUnit: "bar"},
		{terms: German, input: "1,2\tbar", want: 1.2, wantUnit: "bar"},
		{terms: German, input: "100 l/h", want: 100, wantUnit: "l/h"},
		{terms: German, input: "400 RPM", want: 400, wantUnit: "rpm"},
		{terms: German, input: "-12.2 V", want: -12.2, wantUnit: "V"},
		{terms: German, input: "50%", want: 50, wantUnit: "pct"},
		{terms: German, input: "100000 kWh", want: 100000, wantUnit: "kWh"},
		{terms: German, input: "1 kW", want: 1, wantUnit: "kW"},
		{terms: German, input: "16.66 Hz", want: 16.66, wantUnit: "Hz"},
		{terms: English, input: "200 mA", want: 200, wantUnit: "mA"},
		{terms: English, input: "3600s", want: 3600, wantUnit: "s"},
		{terms: English, input: "36 m³/h", want: 36, wantUnit: "m³/h"},
		{terms: English, input: "18 min", want: 18 * 60, wantUnit: "s"},
	} {
		t.Run(tc.terms.ID+" "+tc.input, func(t *testing.T) {
			got, gotUnit, err := tc.terms.ParseMeasurement(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMeasurement(%q) didn't fail", tc.input)
				}
			} else if err != nil {
				t.Errorf("ParseMeasurement(%q) failed: %v", tc.input, err)
			} else if diff := cmp.Diff(tc.want, got, cmpopts.EquateApprox(0, 0.001)); diff != "" {
				t.Errorf("ParseMeasurement(%q) difference (-want +got):\n%s", tc.input, diff)
			} else if diff := cmp.Diff(tc.wantUnit, gotUnit); diff != "" {
				t.Errorf("ParseMeasurement(%q) difference (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}
