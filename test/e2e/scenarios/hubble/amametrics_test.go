//go:build e2e
// +build e2e

package hubble

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/Azure/azure-container-networking/test/e2e/framework/types"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const promHubbleJob = "hubble-pods"

var (
	ErrHubbleTargetNotUp = fmt.Errorf("hubble target not up")
	ErrNoActiveTargets   = fmt.Errorf("no active targets found")
)

func TestE2EPrometheusTargets(t *testing.T) {
	job := types.NewJob("Verify Prometheus targets are up")
	runner := types.NewRunner(t, job)
	defer runner.Run()

	job.AddStep(&VerifyPrometheusMetrics{
		Address: "http://localhost:9090",
	}, nil)
}

type VerifyPrometheusMetrics struct {
	Address string
}

func (v *VerifyPrometheusMetrics) Run() error {
	client, err := api.NewClient(api.Config{
		Address: v.Address,
	})
	if err != nil {
		return fmt.Errorf("failed to create prometheus client: %w", err)
	}

	promapi := promv1.NewAPI(client)
	ctx := context.Background()
	targets, err := promapi.Targets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get targets: %w", err)
	}

	if len(targets.Active) == 0 {
		return fmt.Errorf("no active targets found: %w", ErrNoActiveTargets)
	}

	validTarget := &promv1.ActiveTarget{
		ScrapePool: promHubbleJob,
		Health:     "up",
	}

	for i := range targets.Active {
		target := &targets.Active[i]
		if target.ScrapePool == validTarget.ScrapePool {
			if target.Health != validTarget.Health {
				return ErrHubbleTargetNotUp
			}
			break
		}
	}

	log.Printf("Verified Hubble Prometheus targets are up")
	return nil
}

func (v *VerifyPrometheusMetrics) Prevalidate() error {
	return nil
}

func (v *VerifyPrometheusMetrics) Postvalidate() error {
	return nil
}

func (v *VerifyPrometheusMetrics) Stop() {
}
