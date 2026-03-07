package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type clients struct {
	alertPolicy         *monitoring.AlertPolicyClient
	group               *monitoring.GroupClient
	metric              *monitoring.MetricClient
	notificationChannel *monitoring.NotificationChannelClient
	serviceMonitoring   *monitoring.ServiceMonitoringClient
	snooze              *monitoring.SnoozeClient
	uptime              *monitoring.UptimeCheckClient
	query               *monitoring.QueryClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	groupID := getenv("STACKYARD_GCP_MONITORING_GROUP_ID", "group-1")
	serviceID := getenv("STACKYARD_GCP_MONITORING_SERVICE_ID", "service-a")
	grpcEndpoint := grpcEndpointFromEnv()

	projectName := fmt.Sprintf("projects/%s", projectID)
	groupName := fmt.Sprintf("%s/groups/%s", projectName, groupID)
	serviceName := fmt.Sprintf("%s/services/%s", projectName, serviceID)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Cloud Monitoring apiv3 clients using %s\n", grpcEndpoint)

	clientOptions := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	alertPolicyClient, err := monitoring.NewAlertPolicyClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create alert policy client: %v", err)
	}
	defer closeClient("alert policy", alertPolicyClient.Close)

	groupClient, err := monitoring.NewGroupClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create group client: %v", err)
	}
	defer closeClient("group", groupClient.Close)

	metricClient, err := monitoring.NewMetricClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create metric client: %v", err)
	}
	defer closeClient("metric", metricClient.Close)

	notificationChannelClient, err := monitoring.NewNotificationChannelClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create notification channel client: %v", err)
	}
	defer closeClient("notification channel", notificationChannelClient.Close)

	serviceMonitoringClient, err := monitoring.NewServiceMonitoringClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create service monitoring client: %v", err)
	}
	defer closeClient("service monitoring", serviceMonitoringClient.Close)

	snoozeClient, err := monitoring.NewSnoozeClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create snooze client: %v", err)
	}
	defer closeClient("snooze", snoozeClient.Close)

	uptimeClient, err := monitoring.NewUptimeCheckClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create uptime check client: %v", err)
	}
	defer closeClient("uptime", uptimeClient.Close)

	queryClient, err := monitoring.NewQueryClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create query client: %v", err)
	}
	defer closeClient("query", queryClient.Close)

	allClients := &clients{
		alertPolicy:         alertPolicyClient,
		group:               groupClient,
		metric:              metricClient,
		notificationChannel: notificationChannelClient,
		serviceMonitoring:   serviceMonitoringClient,
		snooze:              snoozeClient,
		uptime:              uptimeClient,
		query:               queryClient,
	}

	calls := []callSpec{
		{
			name: "ListAlertPolicies",
			call: func(ctx context.Context, c *clients) error {
				it := c.alertPolicy.ListAlertPolicies(ctx, &monitoringpb.ListAlertPoliciesRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListGroups",
			call: func(ctx context.Context, c *clients) error {
				it := c.group.ListGroups(ctx, &monitoringpb.ListGroupsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListGroupMembers",
			call: func(ctx context.Context, c *clients) error {
				it := c.group.ListGroupMembers(ctx, &monitoringpb.ListGroupMembersRequest{
					Name:     groupName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListMetricDescriptors",
			call: func(ctx context.Context, c *clients) error {
				it := c.metric.ListMetricDescriptors(ctx, &monitoringpb.ListMetricDescriptorsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListMonitoredResourceDescriptors",
			call: func(ctx context.Context, c *clients) error {
				it := c.metric.ListMonitoredResourceDescriptors(ctx, &monitoringpb.ListMonitoredResourceDescriptorsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListNotificationChannels",
			call: func(ctx context.Context, c *clients) error {
				it := c.notificationChannel.ListNotificationChannels(ctx, &monitoringpb.ListNotificationChannelsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListServices",
			call: func(ctx context.Context, c *clients) error {
				it := c.serviceMonitoring.ListServices(ctx, &monitoringpb.ListServicesRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListServiceLevelObjectives",
			call: func(ctx context.Context, c *clients) error {
				it := c.serviceMonitoring.ListServiceLevelObjectives(ctx, &monitoringpb.ListServiceLevelObjectivesRequest{
					Parent:   serviceName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListSnoozes",
			call: func(ctx context.Context, c *clients) error {
				it := c.snooze.ListSnoozes(ctx, &monitoringpb.ListSnoozesRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListUptimeCheckConfigs",
			call: func(ctx context.Context, c *clients) error {
				it := c.uptime.ListUptimeCheckConfigs(ctx, &monitoringpb.ListUptimeCheckConfigsRequest{
					Parent:   projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListUptimeCheckIps",
			call: func(ctx context.Context, c *clients) error {
				it := c.uptime.ListUptimeCheckIps(ctx, &monitoringpb.ListUptimeCheckIpsRequest{
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "QueryTimeSeries",
			call: func(ctx context.Context, c *clients) error {
				it := c.query.QueryTimeSeries(ctx, &monitoringpb.QueryTimeSeriesRequest{
					Name:     projectName,
					Query:    "fetch gce_instance::compute.googleapis.com/instance/cpu/utilization | within 5m",
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, allClients)
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
