package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	parametermanager "cloud.google.com/go/parametermanager/apiv1"
	"cloud.google.com/go/parametermanager/apiv1/parametermanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *parametermanager.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	parameterID := getenv("STACKYARD_GCP_PARAMETERMANAGER_PARAMETER_ID", "app-config")
	versionID := getenv("STACKYARD_GCP_PARAMETERMANAGER_VERSION_ID", "v1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	parameterName := fmt.Sprintf("%s/parameters/%s", parent, parameterID)
	versionName := fmt.Sprintf("%s/versions/%s", parameterName, versionID)

	fmt.Printf("Stackyard GCP Parameter Manager apiv1 client using %s\n", apiEndpoint)

	client, err := parametermanager.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create parametermanager client: %v", err)
	}
	defer closeClient("parametermanager", client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "CreateParameter",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.CreateParameter(ctx, &parametermanagerpb.CreateParameterRequest{
					Parent:      parent,
					ParameterId: parameterID,
					Parameter: &parametermanagerpb.Parameter{
						Name:   parameterName,
						Labels: map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "GetParameter",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.GetParameter(ctx, &parametermanagerpb.GetParameterRequest{Name: parameterName})
				return err
			},
		},
		{
			name: "ListParameters",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				it := c.ListParameters(ctx, &parametermanagerpb.ListParametersRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "UpdateParameter",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.UpdateParameter(ctx, &parametermanagerpb.UpdateParameterRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					Parameter: &parametermanagerpb.Parameter{
						Name:   parameterName,
						Labels: map[string]string{"env": "local-updated"},
					},
				})
				return err
			},
		},
		{
			name: "CreateParameterVersion",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.CreateParameterVersion(ctx, &parametermanagerpb.CreateParameterVersionRequest{
					Parent:             parameterName,
					ParameterVersionId: versionID,
					ParameterVersion: &parametermanagerpb.ParameterVersion{
						Name: versionName,
						Payload: &parametermanagerpb.ParameterVersionPayload{
							Data: []byte(`{"feature":"enabled"}`),
						},
					},
				})
				return err
			},
		},
		{
			name: "GetParameterVersion",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.GetParameterVersion(ctx, &parametermanagerpb.GetParameterVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "ListParameterVersions",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				it := c.ListParameterVersions(ctx, &parametermanagerpb.ListParameterVersionsRequest{
					Parent:   parameterName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "RenderParameterVersion",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.RenderParameterVersion(ctx, &parametermanagerpb.RenderParameterVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "UpdateParameterVersion",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				_, err := c.UpdateParameterVersion(ctx, &parametermanagerpb.UpdateParameterVersionRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"disabled"}},
					ParameterVersion: &parametermanagerpb.ParameterVersion{
						Name:     versionName,
						Disabled: false,
						Payload: &parametermanagerpb.ParameterVersionPayload{
							Data: []byte(`{"feature":"enabled"}`),
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteParameterVersion",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				return c.DeleteParameterVersion(ctx, &parametermanagerpb.DeleteParameterVersionRequest{Name: versionName})
			},
		},
		{
			name: "DeleteParameter",
			call: func(ctx context.Context, c *parametermanager.Client) error {
				return c.DeleteParameter(ctx, &parametermanagerpb.DeleteParameterRequest{Name: parameterName})
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

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
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
