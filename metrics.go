package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TotalCostPerHour   *prometheus.GaugeVec
	CostPerGBPerHour   *prometheus.GaugeVec
	CostPerVCPUPerHour *prometheus.GaugeVec
	PricingErrors      *prometheus.CounterVec
	LastUpdateTime     *prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		TotalCostPerHour: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cloud_vm_total_cost_per_hour",
				Help: "Total cost per hour for the instance type in USD",
			},
			[]string{"provider", "region", "instance_type"},
		),
		CostPerGBPerHour: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cloud_vm_cost_per_gb_hour",
				Help: "Cost per GB of RAM per hour in USD",
			},
			[]string{"provider", "region", "instance_type"},
		),
		CostPerVCPUPerHour: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cloud_vm_cost_per_vcpu_hour",
				Help: "Cost per vCPU per hour in USD",
			},
			[]string{"provider", "region", "instance_type"},
		),
		PricingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cloud_vm_pricing_errors_total",
				Help: "Total number of errors encountered while fetching pricing",
			},
			[]string{"provider", "region"},
		),
		LastUpdateTime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cloud_vm_pricing_last_update_timestamp_seconds",
				Help: "Unix timestamp of the last successful pricing update",
			},
			[]string{"provider", "region"},
		),
	}
}

type VMPricing struct {
	Provider     string
	Region       string
	InstanceType string
	TotalCost    float64
	MemoryGB     float64
	VCPUs        int
}

func (m *Metrics) RecordPricing(p VMPricing) {
	labels := prometheus.Labels{
		"provider":      p.Provider,
		"region":        p.Region,
		"instance_type": p.InstanceType,
	}

	m.TotalCostPerHour.With(labels).Set(p.TotalCost)

	if p.MemoryGB > 0 {
		m.CostPerGBPerHour.With(labels).Set(p.TotalCost / p.MemoryGB)
	}

	if p.VCPUs > 0 {
		m.CostPerVCPUPerHour.With(labels).Set(p.TotalCost / float64(p.VCPUs))
	}
}