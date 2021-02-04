package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	kitlog "github.com/go-kit/kit/log"
)

var listenAddress = flag.String("web.listen-address", ":8081", "The address to listen on for HTTP requests")
var metricsPath = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
var disableExporterMetrics = flag.Bool("web.disable-exporter-metrics", false, "Exclude metrics about the exporter itself")
var maxRequests = flag.Uint("web.max-requests", 3, "Maximum number of concurrent scrape requests")
var configFile = flag.String("web.config", "", "Path to config yaml file that can enable TLS or authentication")

var timeout = flag.Duration("scrape-timeout", time.Minute, "Maximum duration for a scrape")

var target = flag.String("controller.address", "",
	`host:port for controller Websocket service (e.g. "192.0.2.1:8214")`)
var httpTarget = flag.String("controller.address.http", "",
	`host:port for controller HTTP service; used to retrieve time (e.g. "192.0.2.1:80")`)
var timezone = flag.String("controller.timezone", time.Local.String(),
	"Timezone for parsing timestamps")
var lang = flag.String("controller.language", "",
	fmt.Sprintf("Controller interface language (one of %q)", supportedLanguages()))

func supportedLanguages() []string {
	result := []string{}

	for _, terms := range luxwslang.All() {
		result = append(result, terms.ID)
	}

	return result
}

func main() {
	flag.Parse()

	if *target == "" {
		log.Fatal("Target address not specified")
	}

	opts := collectorOpts{
		maxConcurrent: int64(*maxRequests),
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
			prometheus.NewBuildInfoCollector(),
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
			prometheus.NewGoCollector(),
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

	log.Printf("Listening on %q", *listenAddress)

	logger := kitlog.NewLogfmtLogger(kitlog.StdlibWriter{})

	server := &http.Server{Addr: *listenAddress}

	if err := web.Listen(server, *configFile, logger); err != nil {
		log.Fatal(err)
	}
}
