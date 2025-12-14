# Cloud Pricing Monitor

A Go-based tool to monitor AWS EC2 and GCP Compute Engine VM pricing and export metrics in Prometheus format.

## Features

- Tracks VM pricing for AWS EC2 and GCP Compute Engine instances
- Exports Prometheus metrics including:
  - Total cost per hour for each instance type
  - Cost per GB of RAM per hour
  - Cost per vCPU per hour
- Configurable polling interval (default: 1 hour)
- Support for multiple regions and instance types
- Built-in telemetry and structured logging

## Installation

```bash
go install github.com/jazware/cloud-pricing-monitor@latest
```

Or build from source:

```bash
git clone https://github.com/jazware/cloud-pricing-monitor
cd cloud-pricing-monitor
go build -o cloud-pricing-monitor
```

## Usage

### Basic Example

Monitor AWS EC2 pricing in us-east-1:

```bash
cloud-pricing-monitor \
  --aws-regions us-east-1 \
  --aws-instance-types t3.micro,t3.small,m5.large
```

Monitor GCP pricing in us-central1:

```bash
cloud-pricing-monitor \
  --gcp-regions us-central1 \
  --gcp-instance-types e2-micro,n2-standard-2
```

Monitor both AWS and GCP:

```bash
cloud-pricing-monitor \
  --aws-regions us-east-1,us-west-2 \
  --aws-instance-types t3.micro,m5.large \
  --gcp-regions us-central1,europe-west1 \
  --gcp-instance-types e2-micro,n2-standard-2 \
  --poll-interval 30m \
  --metrics-addr :9090
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--aws-regions` | `AWS_REGIONS` | - | Comma-separated list of AWS regions to monitor |
| `--aws-instance-types` | `AWS_INSTANCE_TYPES` | - | Comma-separated list of AWS EC2 instance types |
| `--gcp-regions` | `GCP_REGIONS` | - | Comma-separated list of GCP regions to monitor |
| `--gcp-instance-types` | `GCP_INSTANCE_TYPES` | - | Comma-separated list of GCP machine types |
| `--poll-interval` | `POLL_INTERVAL` | `1h` | How often to refresh pricing data |
| `--metrics-addr` | `METRICS_ADDR` | `:9090` | Address to serve Prometheus metrics |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Using Environment Variables

```bash
export AWS_REGIONS=us-east-1,us-west-2
export AWS_INSTANCE_TYPES=t3.micro,t3.small,m5.large
export POLL_INTERVAL=30m
export LOG_LEVEL=debug

cloud-pricing-monitor
```

## Authentication

### AWS

The tool uses the AWS SDK default credential chain. Configure credentials using one of:

1. Environment variables:
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   export AWS_REGION=us-east-1
   ```

2. AWS credentials file (`~/.aws/credentials`)
3. IAM role (when running on EC2)

Required IAM permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "pricing:GetProducts"
      ],
      "Resource": "*"
    }
  ]
}
```

### GCP

The tool uses Application Default Credentials. Configure using one of:

1. Service account key file:
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
   ```

2. `gcloud auth application-default login`
3. Compute Engine default service account (when running on GCE)

Required GCP permissions:
- `cloudbilling.skus.list` (typically included in the `roles/billing.viewer` role)

## Prometheus Metrics

The following metrics are exported:

### `cloud_vm_total_cost_per_hour`
Total cost per hour for the instance type in USD.

Labels:
- `provider`: Cloud provider (aws or gcp)
- `region`: Region name
- `instance_type`: Instance/machine type

### `cloud_vm_cost_per_gb_hour`
Cost per GB of RAM per hour in USD.

Labels:
- `provider`: Cloud provider (aws or gcp)
- `region`: Region name
- `instance_type`: Instance/machine type

### `cloud_vm_cost_per_vcpu_hour`
Cost per vCPU per hour in USD.

Labels:
- `provider`: Cloud provider (aws or gcp)
- `region`: Region name
- `instance_type`: Instance/machine type

### `cloud_vm_pricing_errors_total`
Total number of errors encountered while fetching pricing.

Labels:
- `provider`: Cloud provider (aws or gcp)
- `region`: Region name

### `cloud_vm_pricing_last_update_timestamp_seconds`
Unix timestamp of the last successful pricing update.

Labels:
- `provider`: Cloud provider (aws or gcp)
- `region`: Region name

## Example Prometheus Queries

Get the total cost per hour for all AWS t3.micro instances:
```promql
cloud_vm_total_cost_per_hour{provider="aws",instance_type="t3.micro"}
```

Compare costs across regions for the same instance type:
```promql
cloud_vm_total_cost_per_hour{provider="aws",instance_type="m5.large"}
```

Calculate total monthly cost (assuming 730 hours per month):
```promql
cloud_vm_total_cost_per_hour * 730
```

Find the cheapest instance per vCPU:
```promql
sort(cloud_vm_cost_per_vcpu_hour)
```

## Docker

Build the Docker image:

```bash
docker build -t cloud-pricing-monitor .
```

Run with Docker:

```bash
docker run -p 9090:9090 \
  -e AWS_REGIONS=us-east-1 \
  -e AWS_INSTANCE_TYPES=t3.micro,m5.large \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  cloud-pricing-monitor
```

## Development

Run tests:
```bash
go test ./...
```

Build:
```bash
go build -o cloud-pricing-monitor
```

Run locally:
```bash
go run . --aws-regions us-east-1 --aws-instance-types t3.micro
```

## Notes

- AWS Pricing API is only available in `us-east-1` and `ap-south-1` regions, but returns pricing for all regions
- GCP pricing is fetched from the Cloud Billing API
- Pricing data is cached and refreshed at the configured poll interval
- For AWS, only Linux on-demand pricing with shared tenancy is tracked
- GCP pricing is calculated based on per-vCPU and per-GB-RAM pricing

## License

MIT
