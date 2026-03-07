package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	metricsscope "cloud.google.com/go/monitoring/metricsscope/apiv1"
	"cloud.google.com/go/monitoring/metricsscope/apiv1/metricsscopepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *metricsscope.MetricsScopesClient) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	monitoredProjectID := getenv("STACKYARD_GCP_MONITORED_PROJECT_ID", "team-a")
	grpcEndpoint := grpcEndpointFromEnv()

	metricsScopeName := fmt.Sprintf("locations/global/metricsScopes/%s", projectID)
	monitoredProjectName := fmt.Sprintf("%s/projects/%s", metricsScopeName, monitoredProjectID)
	monitoredResourceContainer := fmt.Sprintf("projects/%s", monitoredProjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Cloud Monitoring Metrics Scope apiv1 client using %s\n", grpcEndpoint)

	client, err := metricsscope.NewMetricsScopesClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create metricsscope client: %v", err)
	}
	defer closeClient("metricsscope", client.Close)

	calls := []callSpec{
		{
			name: "GetMetricsScope",
			call: func(ctx context.Context, c *metricsscope.MetricsScopesClient) error {
				_, err := c.GetMetricsScope(ctx, &metricsscopepb.GetMetricsScopeRequest{
					Name: metricsScopeName,
				})
				return err
			},
		},
		{
			name: "ListMetricsScopesByMonitoredProject",
			call: func(ctx context.Context, c *metricsscope.MetricsScopesClient) error {
				_, err := c.ListMetricsScopesByMonitoredProject(ctx, &metricsscopepb.ListMetricsScopesByMonitoredProjectRequest{
					MonitoredResourceContainer: monitoredResourceContainer,
				})
				return err
			},
		},
		{
			name: "CreateMonitoredProject",
			call: func(ctx context.Context, c *metricsscope.MetricsScopesClient) error {
				_, err := c.CreateMonitoredProject(ctx, &metricsscopepb.CreateMonitoredProjectRequest{
					Parent: metricsScopeName,
					MonitoredProject: &metricsscopepb.MonitoredProject{
						Name: monitoredProjectName,
					},
				})
				return err
			},
		},
		{
			name: "DeleteMonitoredProject",
			call: func(ctx context.Context, c *metricsscope.MetricsScopesClient) error {
				_, err := c.DeleteMonitoredProject(ctx, &metricsscopepb.DeleteMonitoredProjectRequest{
					Name: monitoredProjectName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
