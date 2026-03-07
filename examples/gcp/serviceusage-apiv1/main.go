package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	serviceID := getenv("STACKYARD_GCP_SERVICE_ID", "serviceusage.googleapis.com")

	parent := "projects/" + projectID
	serviceName := fmt.Sprintf("%s/services/%s", parent, serviceID)

	fmt.Printf("Stackyard GCP Service Usage apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "serviceusage",
		},
	}

	client, err := serviceusage.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create serviceusage client: %v", err)
	}
	defer closeClient("serviceusage", client.Close)

	enableOperationName := ""
	disableOperationName := ""
	batchEnableOperationName := ""

	calls := []callSpec{
		{
			name: "ListServices",
			call: func(ctx context.Context) error {
				it := client.ListServices(ctx, &serviceusagepb.ListServicesRequest{
					Parent:   parent,
					PageSize: 1,
					Filter:   "state:ENABLED",
				})
				svc, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if svc != nil && strings.TrimSpace(svc.GetName()) != "" {
					serviceName = svc.GetName()
				}
				return nil
			},
		},
		{
			name: "GetService",
			call: func(ctx context.Context) error {
				_, err := client.GetService(ctx, &serviceusagepb.GetServiceRequest{
					Name: serviceName,
				})
				return err
			},
		},
		{
			name: "EnableService",
			call: func(ctx context.Context) error {
				op, err := client.EnableService(ctx, &serviceusagepb.EnableServiceRequest{
					Name: serviceName,
				})
				if err != nil {
					return err
				}
				enableOperationName = op.Name()
				return nil
			},
		},
		{
			name: "DisableService",
			call: func(ctx context.Context) error {
				op, err := client.DisableService(ctx, &serviceusagepb.DisableServiceRequest{
					Name:                   serviceName,
					CheckIfServiceHasUsage: serviceusagepb.DisableServiceRequest_SKIP,
				})
				if err != nil {
					return err
				}
				disableOperationName = op.Name()
				return nil
			},
		},
		{
			name: "BatchEnableServices",
			call: func(ctx context.Context) error {
				op, err := client.BatchEnableServices(ctx, &serviceusagepb.BatchEnableServicesRequest{
					Parent:     parent,
					ServiceIds: []string{serviceID, "stackyard.googleapis.com"},
				})
				if err != nil {
					return err
				}
				batchEnableOperationName = op.Name()
				return nil
			},
		},
		{
			name: "BatchGetServices",
			call: func(ctx context.Context) error {
				_, err := client.BatchGetServices(ctx, &serviceusagepb.BatchGetServicesRequest{
					Parent: parent,
					Names:  []string{serviceName},
				})
				return err
			},
		},
		{
			name: "GetOperation(enable)",
			call: func(ctx context.Context) error {
				if strings.TrimSpace(enableOperationName) == "" {
					return nil
				}
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: enableOperationName,
				})
				return err
			},
		},
		{
			name: "GetOperation(disable)",
			call: func(ctx context.Context) error {
				if strings.TrimSpace(disableOperationName) == "" {
					return nil
				}
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: disableOperationName,
				})
				return err
			},
		},
		{
			name: "GetOperation(batchEnable)",
			call: func(ctx context.Context) error {
				if strings.TrimSpace(batchEnableOperationName) == "" {
					return nil
				}
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: batchEnableOperationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     "operations",
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
