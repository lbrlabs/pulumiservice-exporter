package collector

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	
)

// check interface
var _ Scraper = ScrapeRum{}

var (
	rumRefInfo = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resources", "hourly"),
		" resources under management",
		[]string{"total"}, nil,
	)
)

type ScrapeRum struct{}

// Name of the Scraper. Should be unique.
func (ScrapeRum) Name() string {
	return "rum"
}

// Help describes the role of the Scraper.
func (ScrapeRum) Help() string {
	return "Collect the number of resources under management for an org."
}

// Scrape collects data from client and sends it over channel as prometheus metric.
func (ScrapeRum) Scrape(client *PulumiServiceClient, ch chan<- prometheus.Metric) error {
	var data rumSummary
	url := fmt.Sprintf("/orgs/%s/resources/summary?granularity=hourly&lookbackDays=1", client.Opts.Org)
	body, err := client.request(url)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &data); err != nil {
		return err
	}

	summary := data.Summary
	currentData := summary[len(summary)-1]

	ch <- prometheus.MustNewConstMetric(rumRefInfo, prometheus.GaugeValue,
		float64(currentData.Resources), "rum")

	return nil
}

type rumSummary struct {
	Summary []Summary `json:"summary"`
}

type Summary struct {
	Year      int `json:"year"`
	Month     int `json:"month"`
	Day       int `json:"day"`
	Hour      int `json:"hour"`
	Resources int `json:"resources"`
}
