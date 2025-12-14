package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	awsRegions       []string
	awsInstanceTypes []string
	gcpRegions       []string
	gcpInstanceTypes []string
	pollInterval     time.Duration
	metrics          *Metrics

	awsFetcher *AWSPricingFetcher
	gcpFetcher *GCPPricingFetcher
}

func (m *Monitor) Start(ctx context.Context) error {
	// Initialize fetchers
	if len(m.awsRegions) > 0 {
		awsFetcher, err := NewAWSPricingFetcher(ctx)
		if err != nil {
			return err
		}
		m.awsFetcher = awsFetcher
	}

	if len(m.gcpRegions) > 0 {
		gcpFetcher, err := NewGCPPricingFetcher(ctx)
		if err != nil {
			return err
		}
		m.gcpFetcher = gcpFetcher
	}

	// Perform initial fetch
	if err := m.fetchAllPricing(ctx); err != nil {
		slog.Error("initial pricing fetch failed", "error", err)
	}

	// Start polling goroutine
	go m.pollPricing(ctx)

	return nil
}

func (m *Monitor) pollPricing(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping pricing monitor")
			return
		case <-ticker.C:
			if err := m.fetchAllPricing(ctx); err != nil {
				slog.Error("pricing fetch failed", "error", err)
			}
		}
	}
}

func (m *Monitor) fetchAllPricing(ctx context.Context) error {
	slog.Info("fetching pricing data")

	var wg sync.WaitGroup

	// Fetch AWS pricing
	if m.awsFetcher != nil {
		for _, region := range m.awsRegions {
			for _, instanceType := range m.awsInstanceTypes {
				wg.Add(1)
				go func(region, instanceType string) {
					defer wg.Done()
					m.fetchAWSPricing(ctx, region, instanceType)
				}(region, instanceType)
			}
		}
	}

	// Fetch GCP pricing
	if m.gcpFetcher != nil {
		for _, region := range m.gcpRegions {
			for _, instanceType := range m.gcpInstanceTypes {
				wg.Add(1)
				go func(region, instanceType string) {
					defer wg.Done()
					m.fetchGCPPricing(ctx, region, instanceType)
				}(region, instanceType)
			}
		}
	}

	wg.Wait()
	slog.Info("pricing data fetch complete")
	return nil
}

func (m *Monitor) fetchAWSPricing(ctx context.Context, region, instanceType string) {
	pricing, err := m.awsFetcher.FetchPricing(ctx, region, instanceType)
	if err != nil {
		slog.Error("failed to fetch AWS pricing",
			"region", region,
			"instance_type", instanceType,
			"error", err,
		)
		m.metrics.PricingErrors.With(prometheus.Labels{
			"provider": "aws",
			"region":   region,
		}).Inc()
		return
	}

	m.metrics.RecordPricing(*pricing)
	m.metrics.LastUpdateTime.With(prometheus.Labels{
		"provider": "aws",
		"region":   region,
	}).Set(float64(time.Now().Unix()))

	slog.Info("updated AWS pricing",
		"region", region,
		"instance_type", instanceType,
		"cost_per_hour", pricing.TotalCost,
	)
}

func (m *Monitor) fetchGCPPricing(ctx context.Context, region, instanceType string) {
	pricing, err := m.gcpFetcher.FetchPricing(ctx, region, instanceType)
	if err != nil {
		slog.Error("failed to fetch GCP pricing",
			"region", region,
			"instance_type", instanceType,
			"error", err,
		)
		m.metrics.PricingErrors.With(prometheus.Labels{
			"provider": "gcp",
			"region":   region,
		}).Inc()
		return
	}

	m.metrics.RecordPricing(*pricing)
	m.metrics.LastUpdateTime.With(prometheus.Labels{
		"provider": "gcp",
		"region":   region,
	}).Set(float64(time.Now().Unix()))

	slog.Info("updated GCP pricing",
		"region", region,
		"instance_type", instanceType,
		"cost_per_hour", pricing.TotalCost,
	)
}