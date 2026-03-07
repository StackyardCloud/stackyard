package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dashboard "cloud.google.com/go/monitoring/dashboard/apiv1"
	"cloud.google.com/go/monitoring/dashboard/apiv1/dashboardpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *dashboard.DashboardsClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	dashboardID := getenv("STACKYARD_GCP_MONITORING_DASHBOARD_ID", "dashboard-1")

	parent := fmt.Sprintf("projects/%s", projectID)
	dashboardName := fmt.Sprintf("%s/dashboards/%s", parent, dashboardID)

	fmt.Printf("Stackyard GCP Cloud Monitoring Dashboard apiv1 client using %s\n", apiEndpoint)

	client, err := dashboard.NewDashboardsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create monitoring dashboard client: %v", err)
	}
	defer closeClient("monitoring dashboard", client.Close)

	calls := []callSpec{
		{
			name: "ListDashboards",
			call: func(ctx context.Context, c *dashboard.DashboardsClient) error {
				it := c.ListDashboards(ctx, &dashboardpb.ListDashboardsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetDashboard",
			call: func(ctx context.Context, c *dashboard.DashboardsClient) error {
				_, err := c.GetDashboard(ctx, &dashboardpb.GetDashboardRequest{
					Name: dashboardName,
				})
				return err
			},
		},
		{
			name: "CreateDashboard",
			call: func(ctx context.Context, c *dashboard.DashboardsClient) error {
				_, err := c.CreateDashboard(ctx, &dashboardpb.CreateDashboardRequest{
					Parent: parent,
					Dashboard: &dashboardpb.Dashboard{
						DisplayName: "Stackyard Dashboard",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateDashboard",
			call: func(ctx context.Context, c *dashboard.DashboardsClient) error {
				_, err := c.UpdateDashboard(ctx, &dashboardpb.UpdateDashboardRequest{
					Dashboard: &dashboardpb.Dashboard{
						Name:        dashboardName,
						DisplayName: "Stackyard Dashboard Updated",
						Labels:      map[string]string{"owner": "stackyard"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteDashboard",
			call: func(ctx context.Context, c *dashboard.DashboardsClient) error {
				return c.DeleteDashboard(ctx, &dashboardpb.DeleteDashboardRequest{
					Name: dashboardName,
				})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
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
