package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	logging "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	monitoredres "google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type clients struct {
	logging *logging.Client
	config  *logging.ConfigClient
	metrics *logging.MetricsClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	logID := getenv("STACKYARD_GCP_LOGGING_LOG_ID", "stackyard%2Fapp")
	sinkID := getenv("STACKYARD_GCP_LOGGING_SINK_ID", "export-a")
	exclusionID := getenv("STACKYARD_GCP_LOGGING_EXCLUSION_ID", "exclude-debug")
	metricID := getenv("STACKYARD_GCP_LOGGING_METRIC_ID", "error_count")

	projectName := fmt.Sprintf("projects/%s", projectID)
	logName := fmt.Sprintf("%s/logs/%s", projectName, logID)
	sinkName := fmt.Sprintf("%s/sinks/%s", projectName, sinkID)
	exclusionName := fmt.Sprintf("%s/exclusions/%s", projectName, exclusionID)
	metricName := fmt.Sprintf("%s/metrics/%s", projectName, metricID)

	fmt.Printf("Stackyard GCP Logging apiv2 clients using %s\n", apiEndpoint)

	loggingClient, err := logging.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create logging client: %v", err)
	}
	defer closeClient("logging", loggingClient.Close)

	configClient, err := logging.NewConfigRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create logging config client: %v", err)
	}
	defer closeClient("logging config", configClient.Close)

	metricsClient, err := logging.NewMetricsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create logging metrics client: %v", err)
	}
	defer closeClient("logging metrics", metricsClient.Close)

	allClients := &clients{
		logging: loggingClient,
		config:  configClient,
		metrics: metricsClient,
	}

	calls := []callSpec{
		{
			name: "ListLogs",
			call: func(ctx context.Context, c *clients) error {
				it := c.logging.ListLogs(ctx, &loggingpb.ListLogsRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListMonitoredResourceDescriptors",
			call: func(ctx context.Context, c *clients) error {
				it := c.logging.ListMonitoredResourceDescriptors(ctx, &loggingpb.ListMonitoredResourceDescriptorsRequest{
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListLogEntries",
			call: func(ctx context.Context, c *clients) error {
				it := c.logging.ListLogEntries(ctx, &loggingpb.ListLogEntriesRequest{
					ResourceNames: []string{projectName},
					PageSize:      1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "WriteLogEntries",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.logging.WriteLogEntries(ctx, &loggingpb.WriteLogEntriesRequest{
					LogName: logName,
					Resource: &monitoredres.MonitoredResource{
						Type: "global",
						Labels: map[string]string{
							"project_id": projectID,
						},
					},
					Entries: []*loggingpb.LogEntry{
						{
							Payload: &loggingpb.LogEntry_TextPayload{
								TextPayload: "stackyard logging apiv2 example entry",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteLog",
			call: func(ctx context.Context, c *clients) error {
				return c.logging.DeleteLog(ctx, &loggingpb.DeleteLogRequest{
					LogName: logName,
				})
			},
		},
		{
			name: "ListSinks",
			call: func(ctx context.Context, c *clients) error {
				it := c.config.ListSinks(ctx, &loggingpb.ListSinksRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetSink",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.config.GetSink(ctx, &loggingpb.GetSinkRequest{SinkName: sinkName})
				return err
			},
		},
		{
			name: "CreateSink",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.config.CreateSink(ctx, &loggingpb.CreateSinkRequest{
					Parent: projectName,
					Sink: &loggingpb.LogSink{
						Name:        sinkID,
						Destination: "storage.googleapis.com/stackyard-logs",
						Filter:      "severity>=ERROR",
						Description: "stackyard logging sink",
					},
				})
				return err
			},
		},
		{
			name: "DeleteSink",
			call: func(ctx context.Context, c *clients) error {
				return c.config.DeleteSink(ctx, &loggingpb.DeleteSinkRequest{SinkName: sinkName})
			},
		},
		{
			name: "ListExclusions",
			call: func(ctx context.Context, c *clients) error {
				it := c.config.ListExclusions(ctx, &loggingpb.ListExclusionsRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetExclusion",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.config.GetExclusion(ctx, &loggingpb.GetExclusionRequest{Name: exclusionName})
				return err
			},
		},
		{
			name: "CreateExclusion",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.config.CreateExclusion(ctx, &loggingpb.CreateExclusionRequest{
					Parent: projectName,
					Exclusion: &loggingpb.LogExclusion{
						Name:        exclusionID,
						Description: "stackyard exclusion",
						Filter:      "severity=DEBUG",
					},
				})
				return err
			},
		},
		{
			name: "DeleteExclusion",
			call: func(ctx context.Context, c *clients) error {
				return c.config.DeleteExclusion(ctx, &loggingpb.DeleteExclusionRequest{Name: exclusionName})
			},
		},
		{
			name: "ListLogMetrics",
			call: func(ctx context.Context, c *clients) error {
				it := c.metrics.ListLogMetrics(ctx, &loggingpb.ListLogMetricsRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLogMetric",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.metrics.GetLogMetric(ctx, &loggingpb.GetLogMetricRequest{MetricName: metricName})
				return err
			},
		},
		{
			name: "CreateLogMetric",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.metrics.CreateLogMetric(ctx, &loggingpb.CreateLogMetricRequest{
					Parent: projectName,
					Metric: &loggingpb.LogMetric{
						Name:        metricID,
						Description: "stackyard error counter",
						Filter:      "severity>=ERROR",
					},
				})
				return err
			},
		},
		{
			name: "DeleteLogMetric",
			call: func(ctx context.Context, c *clients) error {
				return c.metrics.DeleteLogMetric(ctx, &loggingpb.DeleteLogMetricRequest{MetricName: metricName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, allClients)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
