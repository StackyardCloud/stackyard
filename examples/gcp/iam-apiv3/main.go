package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iam "cloud.google.com/go/iam/apiv3"
	"cloud.google.com/go/iam/apiv3/iampb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *iam.PolicyBindingsClient, *iam.PrincipalAccessBoundaryPoliciesClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	policyBindingID := getenv("STACKYARD_GCP_IAMV3_POLICY_BINDING_ID", "binding-a")
	organizationID := getenv("STACKYARD_GCP_IAMV3_ORGANIZATION_ID", "123456789012")
	pabID := getenv("STACKYARD_GCP_IAMV3_PAB_ID", "pab-a")
	operationName := getenv("STACKYARD_GCP_IAMV3_OPERATION_NAME", "operations/op-1")

	bindingParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	policyBindingName := bindingParent + "/policyBindings/" + policyBindingID
	pabParent := fmt.Sprintf("organizations/%s/locations/%s", organizationID, locationID)
	pabName := pabParent + "/principalAccessBoundaryPolicies/" + pabID

	fmt.Printf("Stackyard GCP Cloud IAM apiv3 clients using %s\n", apiEndpoint)

	bindingClient, err := iam.NewPolicyBindingsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iam v3 policy bindings client: %v", err)
	}
	defer closeClient("policy bindings", bindingClient.Close)

	pabClient, err := iam.NewPrincipalAccessBoundaryPoliciesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iam v3 principal access boundary client: %v", err)
	}
	defer closeClient("principal access boundary policies", pabClient.Close)

	calls := []callSpec{
		{
			name: "ListPolicyBindings",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				it := binding.ListPolicyBindings(ctx, &iampb.ListPolicyBindingsRequest{Parent: bindingParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPolicyBinding",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := binding.GetPolicyBinding(ctx, &iampb.GetPolicyBindingRequest{Name: policyBindingName})
				return err
			},
		},
		{
			name: "CreatePolicyBinding(validateOnly)",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := binding.CreatePolicyBinding(ctx, &iampb.CreatePolicyBindingRequest{
					Parent:          bindingParent,
					PolicyBindingId: policyBindingID,
					PolicyBinding:   samplePolicyBinding(policyBindingName, pabName),
					ValidateOnly:    true,
				})
				return err
			},
		},
		{
			name: "UpdatePolicyBinding(validateOnly)",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := binding.UpdatePolicyBinding(ctx, &iampb.UpdatePolicyBindingRequest{
					PolicyBinding: samplePolicyBinding(policyBindingName, pabName),
					ValidateOnly:  true,
					UpdateMask:    &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DeletePolicyBinding(validateOnly)",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := binding.DeletePolicyBinding(ctx, &iampb.DeletePolicyBindingRequest{Name: policyBindingName, ValidateOnly: true})
				return err
			},
		},
		{
			name: "SearchTargetPolicyBindings",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				it := binding.SearchTargetPolicyBindings(ctx, &iampb.SearchTargetPolicyBindingsRequest{
					Parent:   bindingParent,
					Target:   "//cloudresourcemanager.googleapis.com/projects/stackyard",
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
			name: "ListPrincipalAccessBoundaryPolicies",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				it := pab.ListPrincipalAccessBoundaryPolicies(ctx, &iampb.ListPrincipalAccessBoundaryPoliciesRequest{Parent: pabParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPrincipalAccessBoundaryPolicy",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := pab.GetPrincipalAccessBoundaryPolicy(ctx, &iampb.GetPrincipalAccessBoundaryPolicyRequest{Name: pabName})
				return err
			},
		},
		{
			name: "CreatePrincipalAccessBoundaryPolicy(validateOnly)",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := pab.CreatePrincipalAccessBoundaryPolicy(ctx, &iampb.CreatePrincipalAccessBoundaryPolicyRequest{
					Parent:                          pabParent,
					PrincipalAccessBoundaryPolicyId: pabID,
					PrincipalAccessBoundaryPolicy:   samplePABPolicy(pabName),
					ValidateOnly:                    true,
				})
				return err
			},
		},
		{
			name: "UpdatePrincipalAccessBoundaryPolicy(validateOnly)",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := pab.UpdatePrincipalAccessBoundaryPolicy(ctx, &iampb.UpdatePrincipalAccessBoundaryPolicyRequest{
					PrincipalAccessBoundaryPolicy: samplePABPolicy(pabName),
					ValidateOnly:                  true,
					UpdateMask:                    &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DeletePrincipalAccessBoundaryPolicy(validateOnly)",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := pab.DeletePrincipalAccessBoundaryPolicy(ctx, &iampb.DeletePrincipalAccessBoundaryPolicyRequest{Name: pabName, ValidateOnly: true, Force: true})
				return err
			},
		},
		{
			name: "SearchPrincipalAccessBoundaryPolicyBindings",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				it := pab.SearchPrincipalAccessBoundaryPolicyBindings(ctx, &iampb.SearchPrincipalAccessBoundaryPolicyBindingsRequest{Name: pabName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetOperation(policyBindingsClient)",
			call: func(ctx context.Context, binding *iam.PolicyBindingsClient, _ *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := binding.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "GetOperation(principalAccessBoundaryPoliciesClient)",
			call: func(ctx context.Context, _ *iam.PolicyBindingsClient, pab *iam.PrincipalAccessBoundaryPoliciesClient) error {
				_, err := pab.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, bindingClient, pabClient)
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

func samplePolicyBinding(name, policyName string) *iampb.PolicyBinding {
	return &iampb.PolicyBinding{
		Name:        name,
		DisplayName: "stackyard policy binding",
		Annotations: map[string]string{"env": "local", "owner": "stackyard"},
		Target: &iampb.PolicyBinding_Target{
			Target: &iampb.PolicyBinding_Target_PrincipalSet{PrincipalSet: "//cloudresourcemanager.googleapis.com/projects/stackyard"},
		},
		Policy: policyName,
	}
}

func samplePABPolicy(name string) *iampb.PrincipalAccessBoundaryPolicy {
	return &iampb.PrincipalAccessBoundaryPolicy{
		Name:        name,
		DisplayName: "stackyard pab policy",
		Annotations: map[string]string{"env": "local", "owner": "stackyard"},
		Details: &iampb.PrincipalAccessBoundaryPolicyDetails{
			Rules: []*iampb.PrincipalAccessBoundaryPolicyRule{
				{
					Description: "restrict access to stackyard project",
					Resources:   []string{"//cloudresourcemanager.googleapis.com/projects/stackyard"},
					Effect:      iampb.PrincipalAccessBoundaryPolicyRule_EFFECT_UNSPECIFIED,
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
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
