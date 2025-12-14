# Contributing to Cloud Pricing Monitor

## Development Setup

1. Ensure you have Go 1.24+ installed
2. Clone the repository
3. Install dependencies:
   ```bash
   make deps
   ```

## Building

Build the application:
```bash
make build
```

Build for all platforms:
```bash
make build-all
```

## Testing

Run tests:
```bash
make test
```

## Code Quality

Format code:
```bash
make fmt
```

Lint code (requires golangci-lint):
```bash
make lint
```

## Adding New Cloud Providers

To add support for a new cloud provider:

1. Create a new file `<provider>.go` (e.g., `azure.go`)
2. Implement a fetcher struct with a `FetchPricing` method:
   ```go
   type AzurePricingFetcher struct {
       client *someAzureClient
   }

   func NewAzurePricingFetcher(ctx context.Context) (*AzurePricingFetcher, error) {
       // Initialize client
   }

   func (f *AzurePricingFetcher) FetchPricing(ctx context.Context, region, vmSize string) (*VMPricing, error) {
       // Fetch and return pricing data
   }
   ```

3. Add CLI flags in [main.go](main.go):
   ```go
   &cli.StringSliceFlag{
       Name:     "azure-regions",
       Usage:    "Azure regions to monitor",
       EnvVars:  []string{"AZURE_REGIONS"},
   },
   ```

4. Update the [Monitor](monitor.go) struct to include the new provider
5. Add fetching logic in `monitor.go`
6. Update the [README.md](README.md) with usage examples

## Project Structure

```
.
├── main.go          # CLI setup and application entry point
├── metrics.go       # Prometheus metrics definitions
├── monitor.go       # Polling and coordination logic
├── aws.go           # AWS pricing fetcher
├── gcp.go           # GCP pricing fetcher
├── Makefile         # Build and development tasks
├── Dockerfile       # Container image definition
└── README.md        # User documentation
```

## Pricing API Notes

### AWS
- Uses the AWS Price List API (Pricing service)
- Only available in `us-east-1` and `ap-south-1`
- Returns pricing for all regions
- Filters for Linux, on-demand, shared tenancy
- Returns JSON with nested structure

### GCP
- Uses Cloud Billing API
- Pricing is based on per-vCPU and per-GB-RAM
- Need to match SKU descriptions to instance families
- Total cost = (vCPU_price × vCPUs) + (RAM_price × RAM_GB)

## Testing with Real APIs

AWS testing:
```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
./cloud-pricing-monitor \
  --aws-regions us-east-1 \
  --aws-instance-types t3.micro \
  --log-level debug
```

GCP testing:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
./cloud-pricing-monitor \
  --gcp-regions us-central1 \
  --gcp-instance-types e2-micro \
  --log-level debug
```

View metrics:
```bash
curl http://localhost:9090/metrics | grep cloud_vm
```

## Debugging

Enable debug logging:
```bash
./cloud-pricing-monitor --log-level debug ...
```

The debug logs will show:
- API requests and responses
- Parsed pricing data
- Metric updates

## Release Process

1. Update version in build command
2. Build all platforms:
   ```bash
   make VERSION=v1.0.0 build-all
   ```
3. Create Git tag
4. Build and push Docker image:
   ```bash
   make VERSION=v1.0.0 docker-build
   docker tag cloud-pricing-monitor:v1.0.0 yourregistry/cloud-pricing-monitor:v1.0.0
   docker push yourregistry/cloud-pricing-monitor:v1.0.0
   ```