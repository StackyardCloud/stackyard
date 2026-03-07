package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iam "cloud.google.com/go/iam/apiv2"
	"cloud.google.com/go/iam/apiv2/iampb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *iam.PoliciesClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	attachmentPoint := getenv("STACKYARD_GCP_IAMV2_ATTACHMENT_POINT", "cloudresourcemanager.googleapis.com%2Fprojects%2Fstackyard")
	policyID := getenv("STACKYARD_GCP_IAMV2_POLICY_ID", "deny-read-access")
	operationName := getenv("STACKYARD_GCP_IAMV2_OPERATION_NAME", "operations/op-1")

	parent := "policies/" + attachmentPoint + "/denypolicies"
	policyName := parent + "/" + policyID

	fmt.Printf("Stackyard GCP Cloud IAM apiv2 client using %s\n", apiEndpoint)

	client, err := iam.NewPoliciesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iam v2 policies client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListPolicies",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				it := c.ListPolicies(ctx, &iampb.ListPoliciesRequest{
					Parent:   parent,
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
			name: "GetPolicy",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				_, err := c.GetPolicy(ctx, &iampb.GetPolicyRequest{Name: policyName})
				return err
			},
		},
		{
			name: "CreatePolicy",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				_, err := c.CreatePolicy(ctx, &iampb.CreatePolicyRequest{
					Parent:   parent,
					PolicyId: policyID,
					Policy:   samplePolicy(policyName, "stackyard deny policy"),
				})
				return err
			},
		},
		{
			name: "UpdatePolicy",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				_, err := c.UpdatePolicy(ctx, &iampb.UpdatePolicyRequest{
					Policy: samplePolicy(policyName, "updated stackyard deny policy"),
				})
				return err
			},
		},
		{
			name: "DeletePolicy",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				_, err := c.DeletePolicy(ctx, &iampb.DeletePolicyRequest{Name: policyName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *iam.PoliciesClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
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

func samplePolicy(name, displayName string) *iampb.Policy {
	return &iampb.Policy{
		Name:        name,
		DisplayName: displayName,
		Annotations: map[string]string{"env": "local", "owner": "stackyard"},
		Rules: []*iampb.PolicyRule{
			{
				Description: "deny broad project read",
				Kind: &iampb.PolicyRule_DenyRule{
					DenyRule: &iampb.DenyRule{
						DeniedPrincipals:  []string{"principalSet://goog/public:all"},
						DeniedPermissions: []string{"resourcemanager.googleapis.com/projects.get"},
					},
				},
			},
		},
	}
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
		fmt.Fprintf(os.Stderr, "warning: close iam v2 policies client: %v\n", err)
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
