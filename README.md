# Cloud Pricing Monitor

A tool to monitor AWS EC2 and GCP Compute Engine VM pricing and export metrics in Prometheus format.

## Features

- Tracks VM pricing for AWS EC2 and GCP Compute Engine instances
- Exports Prometheus metrics including:
  - Total cost per hour for each instance type
  - Cost per GB of RAM per hour
  - Cost per vCPU per hour
- Configurable polling interval (default: 1 hour)
- Support for multiple regions and instance types

## Quick Start with Docker Compose (Recommended)

The easiest way to run Cloud Pricing Monitor is using Docker Compose:

1. Copy the example environment file and configure it:
   ```bash
   cp .env.example .env
   ```

2. Edit [.env](.env) with your cloud provider settings:
   ```bash
   # AWS Configuration
   AWS_REGIONS=us-east-1,us-west-2
   AWS_INSTANCE_TYPES=t3.micro,t3.small,m5.large
   AWS_ACCESS_KEY_ID=your_access_key
   AWS_SECRET_ACCESS_KEY=your_secret_key

   # GCP Configuration
   GCP_REGIONS=us-central1,europe-west1
   GCP_INSTANCE_TYPES=e2-micro,n2-standard-2
   # Note this path is mounted into the container in the docker-compose from `./secrets/gcp-creds.json`
   GOOGLE_APPLICATION_CREDENTIALS=/app/gcp-creds.json

   # App Configuration
   POLL_INTERVAL=1h
   METRICS_LISTEN_ADDRESS=:6015
   LOG_LEVEL=info
   ```

3. If using GCP, place your service account key file:
   ```bash
   mkdir -p secrets
   cp /path/to/your/gcp-key.json secrets/gcp-creds.json
   ```

4. Start the service:
   ```bash
   docker compose up -d
   ```

5. View metrics:
   ```bash
   curl http://localhost:6015/metrics
   ```

6. View logs:
   ```bash
   docker compose logs -f
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
  --metrics-listen-address :6009
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--aws-regions` | `AWS_REGIONS` | - | Comma-separated list of AWS regions to monitor |
| `--aws-instance-types` | `AWS_INSTANCE_TYPES` | - | Comma-separated list of AWS EC2 instance types |
| `--gcp-regions` | `GCP_REGIONS` | - | Comma-separated list of GCP regions to monitor |
| `--gcp-instance-types` | `GCP_INSTANCE_TYPES` | - | Comma-separated list of GCP machine types |
| `--poll-interval` | `POLL_INTERVAL` | `1h` | How often to refresh pricing data |
| `--metrics-listen-address` | `METRICS_LISTEN_ADDRESS` | `:6009` | Address to serve Prometheus metrics |

### Using Environment Variables

```bash
export AWS_REGIONS=us-east-1,us-west-2
export AWS_INSTANCE_TYPES=t3.micro,t3.small,m5.large
export POLL_INTERVAL=30m

cloud-pricing-monitor
```

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

### Running with Plain Docker

If you need to use Docker without Docker Compose:

```bash
# Build the image
docker build -t cloud-pricing-monitor .

# Run the container
docker run -p 6015:6015 \
  -e AWS_REGIONS=us-east-1 \
  -e AWS_INSTANCE_TYPES=t3.micro,m5.large \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  cloud-pricing-monitor
```

**Note:** We recommend using Docker Compose (see Quick Start above) for easier configuration management.

## Notes

- AWS Pricing API is only available in `us-east-1` and `ap-south-1` regions, but returns pricing for all regions
- GCP pricing is fetched from the Cloud Billing API
- Pricing data is cached and refreshed at the configured poll interval
- For AWS, only Linux on-demand pricing with shared tenancy is tracked
- GCP pricing is calculated based on per-vCPU and per-GB-RAM pricing