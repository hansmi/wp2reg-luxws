package luxwslang

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// Terminology describes the names and expressions used by a LuxWS-compatible
// heat pump controller. Member functions allow for parsing of timestamps,
// durations and measurements such as temperatures and pressures.
//
// The wp2reg-language-extractor tool
// (https://github.com/hansmi/wp2reg-language-extractor/) can be used to
// extract translation strings from language files shipped with firmware
// updates.
type Terminology struct {
	ID   string
	Name string

	timestampFormat string

	NavInformation  string
	NavTemperatures string
	NavElapsedTimes string
	NavInputs       string
	NavOutputs      string
	NavHeatQuantity string
	NavErrorMemory  string
	NavSwitchOffs   string

	NavOpHours      string
	HoursImpulsesRe *regexp.Regexp

	NavSystemStatus       string
	StatusType            string
	StatusSoftwareVersion string
	StatusOperationMode   string
	StatusPowerOutput     string

	BoolFalse string
	BoolTrue  string
}

// ParseTimestamp parses a formatted string and returns the time value it
// represents in the given location.
func (t *Terminology) ParseTimestamp(v string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(t.timestampFormat, v, loc)
}

// ParseDuration parses a duration string, e.g. "12:34:56" or "1h".
func (*Terminology) ParseDuration(v string) (time.Duration, error) {
	var hours, minutes, seconds int

	v = strings.TrimSpace(v)

	if n, err := fmt.Sscanf(v, "%d:%d:%d\n", &hours, &minutes, &seconds); err == nil && n == 3 {
	} else if n, err := fmt.Sscanf(v, "%d:%d\n", &hours, &minutes); err == nil && n == 2 {
	} else if n, err := fmt.Sscanf(v, "%dh\n", &hours); err == nil && n == 1 {
	} else if n, err := fmt.Sscanf(v, "%d\n", &hours); err == nil && n == 1 {
	} else {
		return math.MinInt64, fmt.Errorf("unrecognized duration format %q: %w", v, err)
	}

	// Let standard library deal with validation
	return time.ParseDuration(fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds))
}

// ParseMeasurement parses a string with a value and a physical unit such as
// degrees Celsius or kWh. The unit is case-sensitive. The returned unit string
// is in a normalized form.
func (*Terminology) ParseMeasurement(text string) (float64, string, error) {
	if len(text) > 2 {
		var value float64
		var unit string
		var ok bool

		text = strings.TrimSpace(strings.ReplaceAll(text, ",", "."))

		for _, format := range []string{
			"%f %s\n",
			"%f%s\n",
		} {
			if n, err := fmt.Sscanf(text, format, &value, &unit); err == nil && n == 2 {
				ok = true
				break
			}
		}

		if !ok {
			for _, format := range []string{"--- %s\n", "---%s\n"} {
				if n, err := fmt.Sscanf(text, format, &unit); err == nil && n == 1 {
					value = 0
					ok = true
					break
				}
			}
		}

		if ok {
			switch unit {
			case "K", "bar", "l/h", "kWh", "rpm", "V", "kW", "Hz", "mA", "s", "m³/h":
			case "°C":
				unit = "degC"
			case "%":
				unit = "pct"
			case "RPM":
				unit = "rpm"
			case "min":
				unit = "s"
				value *= 60
			default:
				return 0, "", fmt.Errorf("unrecognized unit %q", unit)
			}

			return value, unit, nil
		}
	}

	return 0, "", fmt.Errorf("unrecognized measurement format %q", text)
}
