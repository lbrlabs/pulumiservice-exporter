package collector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	name      = "pulumiservice_exporter"
	namespace = "pulumiservice"
	//Subsystem(s).
	exporter = "exporter"
)

func Name() string {
	return name
}

// Verify if Exporter implements prometheus.Collector
var _ prometheus.Collector = (*Exporter)(nil)

// Metric descriptors.
var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "collector_duration_seconds"),
		"Collector time duration.",
		[]string{"collector"}, nil,
	)
)

type Exporter struct {
	//ctx      context.Context  //http timeout will work, don't need this
	client   *PulumiServiceClient
	scrapers []Scraper
	metrics  Metrics
	logger   log.Logger
}

func New(opts *PulumiServiceOpts, metrics Metrics, scrapers []Scraper, logger log.Logger) (*Exporter, error) {
	uri := opts.Url
	if !strings.Contains(uri, "://") {
		uri = "http://" + uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid pulumi service URL: %s", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid pulumi service URL: %s", uri)
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	tlsClientConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}

	if opts.Insecure {
		tlsClientConfig.InsecureSkipVerify = true
	}

	user := os.Getenv("PULUMI_ACCESS_TOKEN")
	if user != "" {
		opts.AccessToken = user
	}

	transport := &http.Transport{
		TLSClientConfig: tlsClientConfig,
	}

	hc := &PulumiServiceClient{
		Opts: opts,
		Client: &http.Client{
			Timeout:   opts.Timeout,
			Transport: transport,
		},
	}

	return &Exporter{
		client:   hc,
		metrics:  metrics,
		scrapers: scrapers,
		logger:   logger,
	}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.metrics.TotalScrapes.Desc()
	ch <- e.metrics.Error.Desc()
	e.metrics.ScrapeErrors.Describe(ch)
	ch <- e.metrics.PulumiServiceUp.Desc()
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	ch <- e.metrics.TotalScrapes
	ch <- e.metrics.Error
	e.metrics.ScrapeErrors.Collect(ch)
	ch <- e.metrics.PulumiServiceUp
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.metrics.TotalScrapes.Inc()

	scrapeTime := time.Now()

	if pong, err := e.client.Ping(); !pong || err != nil {
		level.Error(e.logger).Log("msg", "Pulumi Service ping failed", "err", err)
		e.metrics.PulumiServiceUp.Set(0)
		e.metrics.Error.Set(1)
	}
	e.metrics.PulumiServiceUp.Set(1)
	e.metrics.Error.Set(0)

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "reach")

	var wg sync.WaitGroup
	defer wg.Wait()
	for _, scraper := range e.scrapers {

		wg.Add(1)
		go func(scraper Scraper) {
			defer wg.Done()
			label := scraper.Name()
			scrapeTime := time.Now()
			if err := scraper.Scrape(e.client, ch); err != nil {
				level.Error(e.logger).Log("msg", "Scrape error", "name", scraper.Name(), "err", err)
				e.metrics.ScrapeErrors.WithLabelValues(label).Inc()
				e.metrics.Error.Set(1)
			}
			ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), label)
		}(scraper)
	}
}

// Metrics represents exporter metrics which values can be carried between http requests.
type Metrics struct {
	TotalScrapes    prometheus.Counter
	ScrapeErrors    *prometheus.CounterVec
	Error           prometheus.Gauge
	PulumiServiceUp prometheus.Gauge
}

// NewMetrics creates new Metrics instance.
func NewMetrics() Metrics {
	subsystem := exporter
	return Metrics{
		TotalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrapes_total",
			Help:      "Total number of times the pulumi service was scraped for metrics.",
		}),
		ScrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping the pulumi service.",
		}, []string{"collector"}),
		Error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from pulumi service resulted in an error (1 for error, 0 for success).",
		}),
		PulumiServiceUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether the pulumi service is up and responding.",
		}),
	}
}
