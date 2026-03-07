package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	appengine "cloud.google.com/go/appengine/apiv1"
	"cloud.google.com/go/appengine/apiv1/appenginepb"
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

	appID := getenv("STACKYARD_GCP_APP_ID", "stackyard")
	serviceID := getenv("STACKYARD_GCP_SERVICE_ID", "default")
	versionID := getenv("STACKYARD_GCP_VERSION_ID", "v1")
	instanceID := getenv("STACKYARD_GCP_INSTANCE_ID", "i-1")

	appName := "apps/" + appID
	serviceName := appName + "/services/" + serviceID
	versionName := serviceName + "/versions/" + versionID
	instanceName := versionName + "/instances/" + instanceID

	fmt.Printf("Stackyard GCP App Engine Admin apiv1 client using %s\n", apiEndpoint)

	applicationsClient, err := appengine.NewApplicationsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create appengine applications client: %v", err)
	}
	defer closeClient("applications", applicationsClient.Close)

	servicesClient, err := appengine.NewServicesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create appengine services client: %v", err)
	}
	defer closeClient("services", servicesClient.Close)

	versionsClient, err := appengine.NewVersionsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create appengine versions client: %v", err)
	}
	defer closeClient("versions", versionsClient.Close)

	instancesClient, err := appengine.NewInstancesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create appengine instances client: %v", err)
	}
	defer closeClient("instances", instancesClient.Close)

	calls := []callSpec{
		{
			name: "GetApplication",
			call: func(ctx context.Context) error {
				_, err := applicationsClient.GetApplication(ctx, &appenginepb.GetApplicationRequest{Name: appName})
				return err
			},
		},
		{
			name: "ListServices",
			call: func(ctx context.Context) error {
				it := servicesClient.ListServices(ctx, &appenginepb.ListServicesRequest{
					Parent:   appName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetService",
			call: func(ctx context.Context) error {
				_, err := servicesClient.GetService(ctx, &appenginepb.GetServiceRequest{Name: serviceName})
				return err
			},
		},
		{
			name: "ListVersions",
			call: func(ctx context.Context) error {
				it := versionsClient.ListVersions(ctx, &appenginepb.ListVersionsRequest{
					Parent:   serviceName,
					PageSize: 1,
					View:     appenginepb.VersionView_BASIC,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetVersion",
			call: func(ctx context.Context) error {
				_, err := versionsClient.GetVersion(ctx, &appenginepb.GetVersionRequest{
					Name: versionName,
					View: appenginepb.VersionView_BASIC,
				})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context) error {
				it := instancesClient.ListInstances(ctx, &appenginepb.ListInstancesRequest{
					Parent:   versionName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context) error {
				_, err := instancesClient.GetInstance(ctx, &appenginepb.GetInstanceRequest{Name: instanceName})
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
