package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	errorreporting "cloud.google.com/go/errorreporting/apiv1beta1"
	"cloud.google.com/go/errorreporting/apiv1beta1/errorreportingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *errorreporting.ErrorStatsClient, *errorreporting.ErrorGroupClient, *errorreporting.ReportErrorsClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	groupID := getenv("STACKYARD_GCP_ERRORREPORTING_GROUP_ID", "group-1")
	serviceName := getenv("STACKYARD_GCP_ERRORREPORTING_SERVICE", "orders-api")
	serviceVersion := getenv("STACKYARD_GCP_ERRORREPORTING_VERSION", "v1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	groupName := fmt.Sprintf("%s/groups/%s", projectName, groupID)

	fmt.Printf("Stackyard GCP Error Reporting apiv1 client using %s\n", apiEndpoint)

	statsClient, err := errorreporting.NewErrorStatsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create error stats client: %v", err)
	}
	defer closeClient("error stats", statsClient.Close)

	groupClient, err := errorreporting.NewErrorGroupRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create error group client: %v", err)
	}
	defer closeClient("error group", groupClient.Close)

	reportClient, err := errorreporting.NewReportErrorsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create report errors client: %v", err)
	}
	defer closeClient("report errors", reportClient.Close)

	calls := []callSpec{
		{
			name: "ListGroupStats",
			call: func(ctx context.Context, statsClient *errorreporting.ErrorStatsClient, _ *errorreporting.ErrorGroupClient, _ *errorreporting.ReportErrorsClient) error {
				it := statsClient.ListGroupStats(ctx, &errorreportingpb.ListGroupStatsRequest{
					ProjectName: projectName,
					PageSize:    1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListEvents",
			call: func(ctx context.Context, statsClient *errorreporting.ErrorStatsClient, _ *errorreporting.ErrorGroupClient, _ *errorreporting.ReportErrorsClient) error {
				it := statsClient.ListEvents(ctx, &errorreportingpb.ListEventsRequest{
					ProjectName: projectName,
					GroupId:     groupID,
					PageSize:    1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeleteEvents",
			call: func(ctx context.Context, statsClient *errorreporting.ErrorStatsClient, _ *errorreporting.ErrorGroupClient, _ *errorreporting.ReportErrorsClient) error {
				_, err := statsClient.DeleteEvents(ctx, &errorreportingpb.DeleteEventsRequest{
					ProjectName: projectName,
				})
				return err
			},
		},
		{
			name: "GetGroup",
			call: func(ctx context.Context, _ *errorreporting.ErrorStatsClient, groupClient *errorreporting.ErrorGroupClient, _ *errorreporting.ReportErrorsClient) error {
				_, err := groupClient.GetGroup(ctx, &errorreportingpb.GetGroupRequest{
					GroupName: groupName,
				})
				return err
			},
		},
		{
			name: "UpdateGroup",
			call: func(ctx context.Context, _ *errorreporting.ErrorStatsClient, groupClient *errorreporting.ErrorGroupClient, _ *errorreporting.ReportErrorsClient) error {
				_, err := groupClient.UpdateGroup(ctx, &errorreportingpb.UpdateGroupRequest{
					Group: &errorreportingpb.ErrorGroup{
						Name:             groupName,
						ResolutionStatus: errorreportingpb.ResolutionStatus_RESOLVED,
					},
				})
				return err
			},
		},
		{
			name: "ReportErrorEvent",
			call: func(ctx context.Context, _ *errorreporting.ErrorStatsClient, _ *errorreporting.ErrorGroupClient, reportClient *errorreporting.ReportErrorsClient) error {
				_, err := reportClient.ReportErrorEvent(ctx, &errorreportingpb.ReportErrorEventRequest{
					ProjectName: projectName,
					Event: &errorreportingpb.ReportedErrorEvent{
						ServiceContext: &errorreportingpb.ServiceContext{
							Service: serviceName,
							Version: serviceVersion,
						},
						Message: "panic: staged stackyard errorreporting example",
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, statsClient, groupClient, reportClient)
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
