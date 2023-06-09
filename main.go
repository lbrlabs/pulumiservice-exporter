package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/lbrlabs/pulumiservice-exporter/collector"
	"github.com/lbrlabs/pulumiservice-exporter/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	promver "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	"runtime"

	"github.com/alecthomas/kingpin/v2"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

type promHTTPLogger struct {
	logger log.Logger
}

func (l promHTTPLogger) Println(v ...interface{}) {
	level.Error(l.logger).Log("msg", fmt.Sprint(v...))
}

var (
	webConfig   = webflag.AddFlags(kingpin.CommandLine, ":9414")
	metricsPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
)

func init() {
	prometheus.MustRegister(promver.NewCollector("pulumiservice_exporter"))
}

func main() {

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	ver := version.Version
	if ver == "" {
		ver = "snapshot"
	}

	sha := version.Commit
	if sha == "" {
		sha = "unknown"
	}

	kingpin.Version(fmt.Sprintf("%s-%s", ver, sha))
	kingpin.HelpFlag.Short('h')

	opts := &collector.PulumiServiceOpts{}
	opts.AddFlag()

	// Generate ON/OFF flags for all scrapers.
	scraperFlags := map[collector.Scraper]*bool{}
	for scraper, enabledByDefault := range collector.Scrapers {
		defaultOn := false
		if enabledByDefault {
			defaultOn = true
		}
		f := kingpin.Flag("collect."+scraper.Name(), scraper.Help()).Default(fmt.Sprintf("%v", defaultOn)).Bool()
		scraperFlags[scraper] = f
	}

	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	// Register only scrapers enabled by flag.
	enabledScrapers := []collector.Scraper{}
	for scraper, enabled := range scraperFlags {
		if *enabled {
			level.Info(logger).Log("msg", "scraper enabled", "name", scraper.Name())
			enabledScrapers = append(enabledScrapers, scraper)
		}
	}

	exporter, err := collector.New(opts, collector.NewMetrics(), enabledScrapers, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Error starting collector", "err", err)
		os.Exit(1)
	}

	prometheus.MustRegister(exporter)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>` + collector.Name() + `</title></head>
             <body>
             <h1><a style="text-decoration:none" href='https://github.com/jaxxstorm/pulumiservice-exporter'>` + collector.Name() + `</a></h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             <h2>Build</h2>
             <pre>` + versionPrint() + `</pre>
             </body>
             </html>`))
	})

	http.Handle(*metricsPath, promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				ErrorLog: &promHTTPLogger{
					logger: logger,
				},
			},
		),
	),
	)

	http.HandleFunc("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok")
	})

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(1)
	}

}

func versionPrint() string {
	return fmt.Sprintf(`Name: %s
Version: %s
CommitID: %s
GoVersion: %s
Compiler: %s
Platform: %s/%s
`, collector.Name(), version.Version, version.Commit, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
}
