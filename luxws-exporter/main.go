package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	kitlog "github.com/go-kit/log"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var webConfig = webflag.AddFlags(kingpin.CommandLine, ":8081")
var metricsPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
var disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude metrics about the exporter itself").Bool()
var maxConcurrent = kingpin.Flag("web.max-requests", "Maximum number of concurrent scrape requests").Default("3").Uint()

var verbose = kingpin.Flag("verbose", "Log sent and received messages").Bool()
var timeout = kingpin.Flag("scrape-timeout", "Maximum duration for a scrape").Default("1m").Duration()

var target = kingpin.Flag("controller.address",
	`host:port for controller Websocket service (e.g. "192.0.2.1:8214")`).PlaceHolder("HOST:PORT").Required().String()
var httpTarget = kingpin.Flag("controller.address.http",
	`host:port for controller HTTP service; used to retrieve time (e.g. "192.0.2.1:80")`).PlaceHolder("HOST:PORT").String()
var timezone = kingpin.Flag("controller.timezone",
	"Timezone for parsing timestamps").Default(time.Local.String()).String()
var lang = kingpin.Flag("controller.language",
	fmt.Sprintf("Controller interface language (one of %q)", supportedLanguages())).PlaceHolder("NAME").Required().String()

func supportedLanguages() []string {
	result := []string{}

	for _, terms := range luxwslang.All() {
		result = append(result, terms.ID)
	}

	return result
}

func main() {
	kingpin.Parse()

	opts := collectorOpts{
		verbose:       *verbose,
		maxConcurrent: int64(*maxConcurrent),
		timeout:       *timeout,
		address:       *target,
		httpAddress:   *httpTarget,
	}

	if loc, err := time.LoadLocation(*timezone); err != nil {
		log.Fatalf("Loading timezone %q failed: %v", *timezone, err)
	} else {
		opts.loc = loc
	}

	if terms, err := luxwslang.LookupByID(*lang); err != nil {
		log.Fatalf("Unknown controller language: %v", err)
	} else {
		opts.terms = terms
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(newCollector(opts))
	if !*disableExporterMetrics {
		reg.MustRegister(
			collectors.NewBuildInfoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
			version.NewCollector("luxws_exporter"),
		)
	}

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>LuxWS Exporter</title></head>
			<body>
			<h1>LuxWS Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	logger := kitlog.NewLogfmtLogger(kitlog.StdlibWriter{})
	server := &http.Server{}

	if err := web.ListenAndServe(server, webConfig, logger); err != nil {
		log.Fatal(err)
	}
}
