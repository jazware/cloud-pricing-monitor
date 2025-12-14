package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bluesky-social/go-util/pkg/telemetry"
	cli "github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:    "cloud-pricing-monitor",
		Usage:   "Monitor and export cloud VM pricing as Prometheus metrics",
		Version: version,
		Flags: []cli.Flag{
			telemetry.CLIFlagDebug,
			telemetry.CLIFlagMetricsListenAddress,
			&cli.StringSliceFlag{
				Name:     "aws-regions",
				Usage:    "AWS regions to monitor (e.g., us-east-1,us-west-2)",
				EnvVars:  []string{"AWS_REGIONS"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "aws-instance-types",
				Usage:    "AWS EC2 instance types to track (e.g., t3.micro,m5.large)",
				EnvVars:  []string{"AWS_INSTANCE_TYPES"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "gcp-regions",
				Usage:    "GCP regions to monitor (e.g., us-central1,us-east1)",
				EnvVars:  []string{"GCP_REGIONS"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "gcp-instance-types",
				Usage:    "GCP machine types to track (e.g., e2-micro,n2-standard-2)",
				EnvVars:  []string{"GCP_INSTANCE_TYPES"},
				Required: false,
			},
			&cli.DurationFlag{
				Name:    "poll-interval",
				Usage:   "How often to refresh pricing data",
				EnvVars: []string{"POLL_INTERVAL"},
				Value:   1 * time.Hour,
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cctx *cli.Context) error {
	ctx, cancel := context.WithCancel(cctx.Context)
	defer cancel()

	// Set up logging
	logger := telemetry.StartLogger(cctx)
	telemetry.StartMetrics(cctx)

	// Validate that at least one cloud provider is configured
	awsRegions := cctx.StringSlice("aws-regions")
	awsInstanceTypes := cctx.StringSlice("aws-instance-types")
	gcpRegions := cctx.StringSlice("gcp-regions")
	gcpInstanceTypes := cctx.StringSlice("gcp-instance-types")

	if len(awsRegions) == 0 && len(gcpRegions) == 0 {
		return fmt.Errorf("must specify at least one AWS or GCP region")
	}

	if len(awsRegions) > 0 && len(awsInstanceTypes) == 0 {
		return fmt.Errorf("aws-regions specified but no aws-instance-types provided")
	}

	if len(gcpRegions) > 0 && len(gcpInstanceTypes) == 0 {
		return fmt.Errorf("gcp-regions specified but no gcp-instance-types provided")
	}

	logger.Info("starting cloud pricing monitor",
		"version", version,
		"aws_regions", strings.Join(awsRegions, ","),
		"aws_instance_types", strings.Join(awsInstanceTypes, ","),
		"gcp_regions", strings.Join(gcpRegions, ","),
		"gcp_instance_types", strings.Join(gcpInstanceTypes, ","),
		"poll_interval", cctx.Duration("poll-interval"),
		"metrics_addr", cctx.String("metrics-addr"),
	)

	// Initialize metrics
	metrics := NewMetrics()

	// Create monitor
	monitor := &Monitor{
		awsRegions:       awsRegions,
		awsInstanceTypes: awsInstanceTypes,
		gcpRegions:       gcpRegions,
		gcpInstanceTypes: gcpInstanceTypes,
		pollInterval:     cctx.Duration("poll-interval"),
		metrics:          metrics,
	}

	// Start monitoring
	if err := monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down...")
	cancel()
	time.Sleep(1 * time.Second)

	return nil
}
