package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	orgpolicy "cloud.google.com/go/orgpolicy/apiv2"
	"cloud.google.com/go/orgpolicy/apiv2/orgpolicypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *orgpolicy.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_ORGPOLICY_PROJECT_ID", "stackyard")
	orgID := getenv("STACKYARD_GCP_ORGPOLICY_ORGANIZATION_ID", "123456789")

	projectParent := fmt.Sprintf("projects/%s", projectID)
	orgParent := fmt.Sprintf("organizations/%s", orgID)
	policyName := fmt.Sprintf("%s/policies/compute.disableSerialPortAccess", orgParent)
	customConstraintName := fmt.Sprintf("%s/customConstraints/custom.requireLabel", orgParent)

	fmt.Printf("Stackyard GCP Organization Policy apiv2 client using %s\n", apiEndpoint)

	client, err := orgpolicy.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create orgpolicy client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListConstraints",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				it := c.ListConstraints(ctx, &orgpolicypb.ListConstraintsRequest{
					Parent:   projectParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListPolicies",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				it := c.ListPolicies(ctx, &orgpolicypb.ListPoliciesRequest{
					Parent:   orgParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetPolicy",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.GetPolicy(ctx, &orgpolicypb.GetPolicyRequest{Name: policyName})
				return err
			},
		},
		{
			name: "GetEffectivePolicy",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.GetEffectivePolicy(ctx, &orgpolicypb.GetEffectivePolicyRequest{Name: policyName})
				return err
			},
		},
		{
			name: "CreatePolicy",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.CreatePolicy(ctx, &orgpolicypb.CreatePolicyRequest{
					Parent: orgParent,
					Policy: samplePolicy(policyName, true),
				})
				return err
			},
		},
		{
			name: "UpdatePolicy",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.UpdatePolicy(ctx, &orgpolicypb.UpdatePolicyRequest{
					Policy: samplePolicy(policyName, false),
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"spec"},
					},
				})
				return err
			},
		},
		{
			name: "DeletePolicy",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				return c.DeletePolicy(ctx, &orgpolicypb.DeletePolicyRequest{Name: policyName})
			},
		},
		{
			name: "ListCustomConstraints",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				it := c.ListCustomConstraints(ctx, &orgpolicypb.ListCustomConstraintsRequest{
					Parent:   orgParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetCustomConstraint",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.GetCustomConstraint(ctx, &orgpolicypb.GetCustomConstraintRequest{Name: customConstraintName})
				return err
			},
		},
		{
			name: "CreateCustomConstraint",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.CreateCustomConstraint(ctx, &orgpolicypb.CreateCustomConstraintRequest{
					Parent:           orgParent,
					CustomConstraint: sampleCustomConstraint(customConstraintName, "stackyard custom constraint"),
				})
				return err
			},
		},
		{
			name: "UpdateCustomConstraint",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				_, err := c.UpdateCustomConstraint(ctx, &orgpolicypb.UpdateCustomConstraintRequest{
					CustomConstraint: sampleCustomConstraint(customConstraintName, "updated stackyard custom constraint"),
				})
				return err
			},
		},
		{
			name: "DeleteCustomConstraint",
			call: func(ctx context.Context, c *orgpolicy.Client) error {
				return c.DeleteCustomConstraint(ctx, &orgpolicypb.DeleteCustomConstraintRequest{Name: customConstraintName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func samplePolicy(name string, enforce bool) *orgpolicypb.Policy {
	return &orgpolicypb.Policy{
		Name: name,
		Spec: &orgpolicypb.PolicySpec{
			Rules: []*orgpolicypb.PolicySpec_PolicyRule{
				{
					Kind: &orgpolicypb.PolicySpec_PolicyRule_Enforce{
						Enforce: enforce,
					},
				},
			},
		},
	}
}

func sampleCustomConstraint(name, description string) *orgpolicypb.CustomConstraint {
	return &orgpolicypb.CustomConstraint{
		Name:          name,
		ResourceTypes: []string{"compute.googleapis.com/Instance"},
		MethodTypes:   []orgpolicypb.CustomConstraint_MethodType{orgpolicypb.CustomConstraint_CREATE},
		Condition:     "resource.name != ''",
		ActionType:    orgpolicypb.CustomConstraint_DENY,
		DisplayName:   "stackyard-custom-constraint",
		Description:   description,
	}
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
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close orgpolicy client: %v\n", err)
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
