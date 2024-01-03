package hubble

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	requiredMetrics = []string{
		"hubble_flows_processed_total",
		"hubble_tcp_flags_total",
	}
)

type ValidateHubbleMetrics struct {
	LocalPort string
}

func (v *ValidateHubbleMetrics) Run() error {
	promAddress := fmt.Sprintf("http://localhost:%s/metrics", v.LocalPort)

	metrics, err := getPrometheusMetrics(promAddress)
	if err != nil {
		return fmt.Errorf("failed to get prometheus metrics: %w", err)
	}

	for _, reqMetric := range requiredMetrics {
		if val, exists := metrics[reqMetric]; !exists {
			return fmt.Errorf("scraping %s, did not find metric %s", val, promAddress) //nolint:goerr113,gocritic
		}
	}
	fmt.Printf("all metrics validated: %+v", requiredMetrics)
	return nil
}

func (v *ValidateHubbleMetrics) ExpectError() bool {
	return false
}

func (c *ValidateHubbleMetrics) SaveParametersToJob() bool {
	return true
}

func (c *ValidateHubbleMetrics) Prevalidate() error {
	return nil
}

func (c *ValidateHubbleMetrics) Postvalidate() error {
	return nil
}

func getPrometheusMetrics(url string) (map[string]struct{}, error) {
	client := http.Client{}
	resp, err := client.Get(url) //nolint
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %v", resp.Status) //nolint:goerr113,gocritic
	}

	metricsData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response body failed: %w", err)
	}

	metrics := parseMetrics(string(metricsData))
	return metrics, nil
}

func parseMetrics(metricsData string) map[string]struct{} {
	// Create a map to store the strings before the first '{'.
	metrics := make(map[string]struct{})

	// sample metrics
	// hubble_tcp_flags_total{destination="",family="IPv4",flag="RST",source="kube-system/metrics-server"} 980
	// hubble_tcp_flags_total{destination="",family="IPv4",flag="SYN",source="kube-system/ama-metrics"} 1777
	// we only want the metric name for the time being
	// label order/parseing can happen later
	lines := strings.Split(metricsData, "\n")
	// Iterate through each line.
	for _, line := range lines {
		// Find the index of the first '{' character.
		index := strings.Index(line, "{")
		if index >= 0 {
			// Extract the string before the first '{'.
			str := strings.TrimSpace(line[:index])
			// Store the string in the map.
			metrics[str] = struct{}{}
		}
	}

	return metrics
}
