package collector

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	
)

// check interface
var _ Scraper = ScrapeStacks{}

var (
	stackRefInfo = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "stacks", "total"),
		" number of stacks",
		[]string{"total"}, nil,
	)
)

type ScrapeStacks struct{}

// Name of the Scraper. Should be unique.
func (ScrapeStacks) Name() string {
	return "stacks"
}

// Help describes the role of the Scraper.
func (ScrapeStacks) Help() string {
	return "Collect the number of stacks in an org."
}

// Scrape collects data from client and sends it over channel as prometheus metric.
func (ScrapeStacks) Scrape(client *PulumiServiceClient, ch chan<- prometheus.Metric) error {
	var data stacksList
	url := fmt.Sprintf("/user/stacks?organization=%s", client.Opts.Org)
	body, err := client.request(url)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &data); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(stackRefInfo, prometheus.GaugeValue,
		float64(len(data.Stacks)), "stacks")

	return nil
}

type stacksList struct {
	Stacks []Stack `json:"stacks"`
}
type Stack struct {
	OrgName       string `json:"orgName"`
	ProjectName   string `json:"projectName"`
	StackName     string `json:"stackName"`
	LastUpdate    int    `json:"lastUpdate"`
	ResourceCount int    `json:"resourceCount"`
}

