package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hansmi/wp2reg-luxws/luxwsclient"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
)

func findContentItem(r *luxwsclient.ContentRoot, name string) (*luxwsclient.ContentItem, error) {
	found := r.FindByName(name)
	if found == nil {
		return nil, fmt.Errorf("item with name %q not found", name)
	}

	return found, nil
}

type contentCollectFunc func(chan<- prometheus.Metric, *luxwsclient.ContentRoot) error

type collector struct {
	sem                   *semaphore.Weighted
	address               string
	loc                   *time.Location
	terms                 *luxwslang.Terminology
	upDesc                *prometheus.Desc
	infoDesc              *prometheus.Desc
	temperatureDesc       *prometheus.Desc
	operatingDurationDesc *prometheus.Desc
	elapsedDurationDesc   *prometheus.Desc
	inputDesc             *prometheus.Desc
	outputDesc            *prometheus.Desc
	opModeDesc            *prometheus.Desc
	heatQuantityDesc      *prometheus.Desc
	suppliedHeatDesc      *prometheus.Desc
	latestErrorDesc       *prometheus.Desc
	switchOffDesc         *prometheus.Desc
}

type collectorOpts struct {
	maxConcurrent int64
	address       string
	loc           *time.Location
	terms         *luxwslang.Terminology
}

func newCollector(opts collectorOpts) *collector {
	if opts.maxConcurrent < 1 {
		opts.maxConcurrent = 1
	}

	return &collector{
		sem:                   semaphore.NewWeighted(opts.maxConcurrent),
		address:               opts.address,
		loc:                   opts.loc,
		terms:                 opts.terms,
		upDesc:                prometheus.NewDesc("luxws_up", "Whether scrape was successful", []string{"status"}, nil),
		temperatureDesc:       prometheus.NewDesc("luxws_temperature", "Sensor temperature", []string{"name", "unit"}, nil),
		operatingDurationDesc: prometheus.NewDesc("luxws_operating_duration_seconds", "Operating time", []string{"name"}, nil),
		elapsedDurationDesc:   prometheus.NewDesc("luxws_elapsed_duration_seconds", "Elapsed time", []string{"name"}, nil),
		inputDesc:             prometheus.NewDesc("luxws_input", "Input values", []string{"name", "unit"}, nil),
		outputDesc:            prometheus.NewDesc("luxws_output", "Output values", []string{"name", "unit"}, nil),
		infoDesc:              prometheus.NewDesc("luxws_info", "Controller information", []string{"swversion", "hptype"}, nil),
		opModeDesc:            prometheus.NewDesc("luxws_operational_mode", "Operational mode", []string{"mode"}, nil),
		heatQuantityDesc:      prometheus.NewDesc("luxws_heat_quantity", "Heat quantity", []string{"unit"}, nil),
		suppliedHeatDesc:      prometheus.NewDesc("luxws_supplied_heat", "Supplied heat", []string{"name", "unit"}, nil),
		latestErrorDesc:       prometheus.NewDesc("luxws_latest_error", "Latest error", []string{"reason"}, nil),
		switchOffDesc:         prometheus.NewDesc("luxws_latest_switchoff", "Latest switch-off", []string{"reason"}, nil),
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upDesc
	ch <- c.infoDesc
	ch <- c.temperatureDesc
	ch <- c.operatingDurationDesc
	ch <- c.elapsedDurationDesc
	ch <- c.inputDesc
	ch <- c.outputDesc
	ch <- c.opModeDesc
	ch <- c.heatQuantityDesc
	ch <- c.suppliedHeatDesc
	ch <- c.latestErrorDesc
	ch <- c.switchOffDesc
}

func (c *collector) parseValue(text string) (float64, string, error) {
	text = strings.TrimSpace(text)

	switch text {
	case c.terms.BoolFalse:
		return 0, "bool", nil

	case c.terms.BoolTrue:
		return 1, "bool", nil
	}

	return c.terms.ParseMeasurement(text)
}

func (c *collector) collectInfo(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	var swVersion, opMode, heatOutputUnit string
	var heatOutputValue float64
	var hpType []string

	group, err := findContentItem(content, c.terms.NavSystemStatus)
	if err != nil {
		return err
	}

	for _, item := range group.Items {
		if item.Value == nil {
			continue
		}

		switch item.Name {
		case c.terms.StatusType:
			hpType = append(hpType, normalizeSpace(*item.Value))
		case c.terms.StatusSoftwareVersion:
			swVersion = normalizeSpace(*item.Value)
		case c.terms.StatusOperationMode:
			opMode = normalizeSpace(*item.Value)
		case c.terms.StatusPowerOutput:
			if heatOutputValue, heatOutputUnit, err = c.parseValue(*item.Value); err != nil {
				return fmt.Errorf("parsing heat output failed: %w", err)
			}
		}
	}

	sort.Strings(hpType)

	ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue,
		1, swVersion, strings.Join(hpType, ", "))

	ch <- prometheus.MustNewConstMetric(c.opModeDesc, prometheus.GaugeValue,
		1, opMode)

	ch <- prometheus.MustNewConstMetric(c.heatQuantityDesc, prometheus.GaugeValue,
		heatOutputValue, heatOutputUnit)

	return nil
}

func (c *collector) collectMeasurements(ch chan<- prometheus.Metric, desc *prometheus.Desc, content *luxwsclient.ContentRoot, groupName string) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	var found bool

	for _, item := range group.Items {
		if item.Value == nil {
			continue
		}

		value, unit, err := c.parseValue(*item.Value)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			value, normalizeSpace(item.Name), unit)

		found = true
	}

	if !found {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			0, "", "")
	}

	return nil
}

func (c *collector) collectDurations(ch chan<- prometheus.Metric, desc *prometheus.Desc, content *luxwsclient.ContentRoot, groupName string, ignoreRe *regexp.Regexp) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	var found bool

	for _, item := range group.Items {
		if item.Value == nil || (ignoreRe != nil && ignoreRe.MatchString(item.Name)) {
			continue
		}

		duration, err := c.terms.ParseDuration(*item.Value)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			duration.Seconds(), normalizeSpace(item.Name))

		found = true
	}

	if !found {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			0, "")
	}

	return nil
}

func (c *collector) collectTimetable(ch chan<- prometheus.Metric, desc *prometheus.Desc, content *luxwsclient.ContentRoot, groupName string) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	latest := map[string]time.Time{}

	for _, item := range group.Items {
		if item.Value == nil {
			continue
		}

		ts, err := c.terms.ParseTimestamp(item.Name, c.loc)
		if err != nil {
			return err
		}

		reason := normalizeSpace(*item.Value)

		// Use only the most recent timestamp per reason
		if prev, ok := latest[reason]; !ok || prev.IsZero() || prev.Before(ts) {
			latest[reason] = ts
		}
	}

	if len(latest) == 0 {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 0, "")
	} else {
		for reason, ts := range latest {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(ts.Unix()), reason)
		}
	}

	return nil
}

func (c *collector) collectTemperatures(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectMeasurements(ch, c.temperatureDesc, content, c.terms.NavTemperatures)
}

func (c *collector) collectOperatingDuration(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectDurations(ch, c.operatingDurationDesc, content, c.terms.NavOpHours, c.terms.HoursImpulsesRe)
}

func (c *collector) collectElapsedTime(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectDurations(ch, c.elapsedDurationDesc, content, c.terms.NavElapsedTimes, nil)
}

func (c *collector) collectInputs(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectMeasurements(ch, c.inputDesc, content, c.terms.NavInputs)
}

func (c *collector) collectOutputs(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectMeasurements(ch, c.outputDesc, content, c.terms.NavOutputs)
}

func (c *collector) collectSuppliedHeat(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectMeasurements(ch, c.suppliedHeatDesc, content, c.terms.NavHeatQuantity)
}

func (c *collector) collectLatestError(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectTimetable(ch, c.latestErrorDesc, content, c.terms.NavErrorMemory)
}

func (c *collector) collectLatestSwitchOff(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	return c.collectTimetable(ch, c.switchOffDesc, content, c.terms.NavSwitchOffs)
}

func (c *collector) collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	// Limit concurrent collections
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return err
	}

	defer c.sem.Release(1)

	cl, err := luxwsclient.Dial(ctx, c.address)
	if err != nil {
		return err
	}

	defer cl.Close()

	nav, err := cl.Login(ctx, "")
	if err != nil {
		return err
	}

	info := nav.FindByName(c.terms.NavInformation)
	if info == nil {
		return errors.New("information ID not found in response")
	}

	content, err := cl.Get(ctx, info.ID)
	if err != nil {
		return fmt.Errorf("fetching ID %q failed: %w", info.ID, err)
	}

	for _, fn := range []contentCollectFunc{
		c.collectInfo,
		c.collectTemperatures,
		c.collectOperatingDuration,
		c.collectElapsedTime,
		c.collectInputs,
		c.collectOutputs,
		c.collectSuppliedHeat,
		c.collectLatestError,
		c.collectLatestSwitchOff,
	} {
		if err := fn(ch, content); err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := c.collect(ctx, ch); err == nil {
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1, "")
	} else {
		log.Printf("Scrape failed: %v", err)
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0, err.Error())
	}
}
