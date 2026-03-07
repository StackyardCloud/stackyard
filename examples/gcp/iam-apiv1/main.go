package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iam "cloud.google.com/go/iam/apiv1"
	"cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *iam.IamPolicyClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"
	resource := getenv("STACKYARD_GCP_IAM_RESOURCE", "projects/stackyard")

	fmt.Printf("Stackyard GCP Cloud IAM apiv1 client using %s\n", apiEndpoint)

	client, err := iam.NewIamPolicyRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iam policy client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *iam.IamPolicyClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: resource})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *iam.IamPolicyClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: resource,
					Policy: &iampb.Policy{
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/viewer",
								Members: []string{"user:alice@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *iam.IamPolicyClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    resource,
					Permissions: []string{"resourcemanager.projects.get", "resourcemanager.projects.getIamPolicy"},
				})
				return err
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close iam policy client: %v\n", err)
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
