package collector

import (
	"fmt"
	"github.com/pkg/errors"
	kingpin "github.com/alecthomas/kingpin/v2"
	"io"
	"net/http"
	"time"
)

var (
	Scrapers = map[Scraper]bool{
		ScrapeRum{}: true,
		ScrapeStacks{}: true,
	}
)

type PulumiServiceOpts struct {
	Url         string
	AccessToken string
	UA          string
	Timeout     time.Duration
	Insecure    bool
	Org         string
}

type PulumiServiceClient struct {
	Client *http.Client
	Opts   *PulumiServiceOpts
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// use after set Opts
func (o *PulumiServiceOpts) AddFlag() {
	kingpin.Flag("address", "API address of the pulumi service. (prefix with https:// to connect over HTTPS)").Default("https://api.pulumi.com").StringVar(&o.Url)
	kingpin.Flag("access-token", "Pulumi access token.").Required().Envar("PULUMI_ACCESS_TOKEN").StringVar(&o.AccessToken)
	kingpin.Flag("user-agent", "user agent of the pulumiservice http client").Default("pulumiservice_exporter").StringVar(&o.UA)
	kingpin.Flag("timeout", "Timeout on HTTP requests to the pulumi service API.").Default("1600ms").DurationVar(&o.Timeout)
	kingpin.Flag("insecure", "Disable TLS host verification.").BoolVar(&o.Insecure)
	kingpin.Flag("org", "Pulumi organization.").Required().StringVar(&o.Org)
}

func (h *PulumiServiceClient) request(endpoint string) ([]byte, error) {
	url := h.Opts.Url + "/api" + endpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", h.Opts.AccessToken))
	req.Header.Set("User-Agent", h.Opts.UA)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pulumi+8")

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error handling request for %s http-statuscode: %s", endpoint, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (h *PulumiServiceClient) Ping() (bool, error) {
	req, err := http.NewRequest("GET", h.Opts.Url+"/api/user", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", h.Opts.AccessToken))
	req.Header.Set("User-Agent", h.Opts.UA)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pulumi+8")

	resp, err := h.Client.Do(req)
	if err != nil {
		return false, err
	}

	resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		return true, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return false, errors.New("invalid pulumi access token")
	default:
		return false, fmt.Errorf("error handling request, http-statuscode: %s", resp.Status)
	}
}
