package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"
)

type GCPPricingFetcher struct {
	service *cloudbilling.APIService
}

func NewGCPPricingFetcher(ctx context.Context) (*GCPPricingFetcher, error) {
	service, err := cloudbilling.NewService(ctx, option.WithScopes(cloudbilling.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP billing service: %w", err)
	}

	return &GCPPricingFetcher{
		service: service,
	}, nil
}

func (f *GCPPricingFetcher) FetchPricing(ctx context.Context, region, machineType string) (*VMPricing, error) {
	slog.Debug("fetching GCP pricing",
		"region", region,
		"machine_type", machineType,
	)

	// Parse machine type to get family and specs
	// GCP machine types follow patterns like: e2-micro, n2-standard-2, n1-standard-4
	family, vcpus, memoryGB, err := parseMachineType(machineType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine type: %w", err)
	}

	// Get the service for Compute Engine
	serviceId := "services/6F81-5844-456A" // Compute Engine service ID

	// Build a filter to find pricing for this machine type
	// GCP pricing is based on vCPU and memory separately
	vcpuPrice, err := f.getVCPUPrice(ctx, serviceId, region, family)
	if err != nil {
		return nil, fmt.Errorf("failed to get vCPU pricing: %w", err)
	}

	memoryPrice, err := f.getMemoryPrice(ctx, serviceId, region, family)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory pricing: %w", err)
	}

	totalCost := (vcpuPrice * float64(vcpus)) + (memoryPrice * memoryGB)

	slog.Debug("fetched GCP pricing",
		"region", region,
		"machine_type", machineType,
		"vcpu_price", vcpuPrice,
		"memory_price", memoryPrice,
		"total_cost", totalCost,
		"vcpus", vcpus,
		"memory_gb", memoryGB,
	)

	return &VMPricing{
		Provider:     "gcp",
		Region:       region,
		InstanceType: machineType,
		TotalCost:    totalCost,
		MemoryGB:     memoryGB,
		VCPUs:        vcpus,
	}, nil
}

func (f *GCPPricingFetcher) getVCPUPrice(ctx context.Context, serviceId, region, family string) (float64, error) {
	// Search for vCPU pricing SKUs
	call := f.service.Services.Skus.List(serviceId)
	call.CurrencyCode("USD")

	var price float64
	err := call.Pages(ctx, func(page *cloudbilling.ListSkusResponse) error {
		for _, sku := range page.Skus {
			// Look for vCPU pricing for the specified region and family
			if f.matchesVCPUSku(sku, region, family) {
				if len(sku.PricingInfo) > 0 && len(sku.PricingInfo[0].PricingExpression.TieredRates) > 0 {
					// Get the price from the first tier (usually the only tier for simple pricing)
					nanos := sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos
					units := sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Units
					price = float64(units) + (float64(nanos) / 1e9)
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	if price == 0 {
		return 0, fmt.Errorf("no vCPU pricing found for region %s and family %s", region, family)
	}

	return price, nil
}

func (f *GCPPricingFetcher) getMemoryPrice(ctx context.Context, serviceId, region, family string) (float64, error) {
	call := f.service.Services.Skus.List(serviceId)
	call.CurrencyCode("USD")

	var price float64
	err := call.Pages(ctx, func(page *cloudbilling.ListSkusResponse) error {
		for _, sku := range page.Skus {
			// Look for memory pricing for the specified region and family
			if f.matchesMemorySku(sku, region, family) {
				if len(sku.PricingInfo) > 0 && len(sku.PricingInfo[0].PricingExpression.TieredRates) > 0 {
					nanos := sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos
					units := sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Units
					price = float64(units) + (float64(nanos) / 1e9)
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	if price == 0 {
		return 0, fmt.Errorf("no memory pricing found for region %s and family %s", region, family)
	}

	return price, nil
}

func (f *GCPPricingFetcher) matchesVCPUSku(sku *cloudbilling.Sku, region, family string) bool {
	desc := strings.ToLower(sku.Description)

	// Check if it's a vCPU SKU
	if !strings.Contains(desc, "core") && !strings.Contains(desc, "vcpu") {
		return false
	}

	// Check if it's for the right family
	familyMatch := false
	switch family {
	case "e2":
		familyMatch = strings.Contains(desc, "e2 instance")
	case "n1":
		familyMatch = strings.Contains(desc, "n1 predefined") || strings.Contains(desc, "n1 instance")
	case "n2", "n2d":
		familyMatch = strings.Contains(desc, "n2 instance") || strings.Contains(desc, "n2d instance")
	case "n4", "n4d":
		familyMatch = strings.Contains(desc, "n4 instance") || strings.Contains(desc, "n4d instance")
	case "c2", "c2d", "c3":
		familyMatch = strings.Contains(desc, family+" instance")
	default:
		familyMatch = strings.Contains(desc, family)
	}

	if !familyMatch {
		return false
	}

	// Check region match
	return slices.Contains(sku.ServiceRegions, region)
}

func (f *GCPPricingFetcher) matchesMemorySku(sku *cloudbilling.Sku, region, family string) bool {
	desc := strings.ToLower(sku.Description)

	// Check if it's a memory SKU
	if !strings.Contains(desc, "ram") && !strings.Contains(desc, "memory") {
		return false
	}

	// Check if it's for the right family
	familyMatch := false
	switch family {
	case "e2":
		familyMatch = strings.Contains(desc, "e2 instance")
	case "n1":
		familyMatch = strings.Contains(desc, "n1 predefined") || strings.Contains(desc, "n1 instance")
	case "n2", "n2d":
		familyMatch = strings.Contains(desc, "n2 instance") || strings.Contains(desc, "n2d instance")
	case "n4", "n4d":
		familyMatch = strings.Contains(desc, "n4 instance") || strings.Contains(desc, "n4d instance")
	case "c2", "c2d", "c3":
		familyMatch = strings.Contains(desc, family+" instance")
	default:
		familyMatch = strings.Contains(desc, family)
	}

	if !familyMatch {
		return false
	}

	// Check region match
	for _, serviceRegion := range sku.ServiceRegions {
		if serviceRegion == region {
			return true
		}
	}

	return false
}

// parseMachineType extracts the machine family, vCPU count, and memory from GCP machine type
func parseMachineType(machineType string) (family string, vcpus int, memoryGB float64, err error) {
	// Standard machine types: e2-micro, e2-small, e2-medium, n1-standard-1, n2-standard-2, etc.
	parts := strings.Split(machineType, "-")
	if len(parts) < 2 {
		return "", 0, 0, fmt.Errorf("invalid machine type format: %s", machineType)
	}

	family = parts[0]
	machineClass := parts[1]

	// Handle predefined machine types
	switch machineType {
	case "e2-micro":
		return "e2", 2, 1.0, nil
	case "e2-small":
		return "e2", 2, 2.0, nil
	case "e2-medium":
		return "e2", 2, 4.0, nil
	case "f1-micro":
		return "f1", 1, 0.6, nil
	case "g1-small":
		return "g1", 1, 1.7, nil
	}

	// For standard machine types, extract vCPU count from the name
	var vcpuCount int
	if len(parts) >= 3 {
		_, err := fmt.Sscanf(parts[2], "%d", &vcpuCount)
		if err != nil {
			return "", 0, 0, fmt.Errorf("invalid vCPU count in machine type: %w", err)
		}
	}

	if vcpuCount == 0 {
		return "", 0, 0, fmt.Errorf("could not determine vCPU count for machine type: %s", machineType)
	}

	// Calculate memory based on machine class
	var memory float64
	switch machineClass {
	case "standard":
		memory = float64(vcpuCount) * 3.75 // 3.75 GB per vCPU
	case "highmem":
		memory = float64(vcpuCount) * 6.5 // 6.5 GB per vCPU
	case "highcpu":
		memory = float64(vcpuCount) * 0.9 // 0.9 GB per vCPU
	default:
		memory = float64(vcpuCount) * 4.0 // Default ratio
	}

	return family, vcpuCount, memory, nil
}
