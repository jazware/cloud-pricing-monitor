package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

type AWSPricingFetcher struct {
	client *pricing.Client
}

func NewAWSPricingFetcher(ctx context.Context) (*AWSPricingFetcher, error) {
	// AWS Pricing API is only available in us-east-1 and ap-south-1
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSPricingFetcher{
		client: pricing.NewFromConfig(cfg),
	}, nil
}

func (f *AWSPricingFetcher) FetchPricing(ctx context.Context, region, instanceType string) (*VMPricing, error) {
	slog.Debug("fetching AWS pricing",
		"region", region,
		"instance_type", instanceType,
	)

	// Build filters for the pricing query
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("ServiceCode"),
			Value: aws.String("AmazonEC2"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("instanceType"),
			Value: aws.String(instanceType),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("regionCode"),
			Value: aws.String(region),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("operatingSystem"),
			Value: aws.String("Linux"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("tenancy"),
			Value: aws.String("Shared"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("capacitystatus"),
			Value: aws.String("Used"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("preInstalledSw"),
			Value: aws.String("NA"),
		},
	}

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int32(10),
	}

	output, err := f.client.GetProducts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS pricing: %w", err)
	}

	if len(output.PriceList) == 0 {
		return nil, fmt.Errorf("no pricing data found for instance type %s in region %s", instanceType, region)
	}

	// Parse the first result
	var priceData map[string]interface{}
	if err := json.Unmarshal([]byte(output.PriceList[0]), &priceData); err != nil {
		return nil, fmt.Errorf("failed to parse pricing data: %w", err)
	}

	// Extract instance attributes
	product, ok := priceData["product"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid product data structure")
	}

	attributes, ok := product["attributes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid attributes data structure")
	}

	// Extract memory and vCPU
	memoryStr, _ := attributes["memory"].(string)
	vcpuStr, _ := attributes["vcpu"].(string)

	memory, err := parseMemory(memoryStr)
	if err != nil {
		slog.Warn("failed to parse memory", "memory", memoryStr, "error", err)
	}

	vcpu, err := strconv.Atoi(vcpuStr)
	if err != nil {
		slog.Warn("failed to parse vcpu", "vcpu", vcpuStr, "error", err)
	}

	// Extract on-demand pricing
	terms, ok := priceData["terms"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid terms data structure")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no OnDemand pricing found")
	}

	// Get the first (and usually only) pricing term
	var hourlyPrice float64
	for _, termData := range onDemand {
		termMap, ok := termData.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := termMap["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		for _, dimension := range priceDimensions {
			dimMap, ok := dimension.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dimMap["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			usdPrice, ok := pricePerUnit["USD"].(string)
			if !ok {
				continue
			}

			hourlyPrice, err = strconv.ParseFloat(usdPrice, 64)
			if err != nil {
				continue
			}

			break
		}

		if hourlyPrice > 0 {
			break
		}
	}

	if hourlyPrice == 0 {
		return nil, fmt.Errorf("no valid pricing found")
	}

	slog.Debug("fetched AWS pricing",
		"region", region,
		"instance_type", instanceType,
		"hourly_price", hourlyPrice,
		"memory_gb", memory,
		"vcpus", vcpu,
	)

	return &VMPricing{
		Provider:     "aws",
		Region:       region,
		InstanceType: instanceType,
		TotalCost:    hourlyPrice,
		MemoryGB:     memory,
		VCPUs:        vcpu,
	}, nil
}

// parseMemory converts AWS memory strings like "8 GiB" to float64 in GB
func parseMemory(memStr string) (float64, error) {
	memStr = strings.TrimSpace(memStr)
	parts := strings.Fields(memStr)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid memory format: %s", memStr)
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	// Convert GiB to GB if needed
	unit := strings.ToUpper(parts[1])
	if unit == "GIB" {
		return value * 1.073741824, nil
	}

	return value, nil
}
