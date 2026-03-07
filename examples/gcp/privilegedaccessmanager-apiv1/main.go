package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	privilegedaccessmanager "cloud.google.com/go/privilegedaccessmanager/apiv1"
	"cloud.google.com/go/privilegedaccessmanager/apiv1/privilegedaccessmanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *privilegedaccessmanager.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	entitlementID := getenv("STACKYARD_GCP_PAM_ENTITLEMENT_ID", "entitlement-a")
	grantID := getenv("STACKYARD_GCP_PAM_GRANT_ID", "grant-a")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	entitlementName := fmt.Sprintf("%s/entitlements/%s", parent, entitlementID)
	grantName := fmt.Sprintf("%s/grants/%s", entitlementName, grantID)
	resourceName := fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", projectID)

	fmt.Printf("Stackyard GCP Privileged Access Manager apiv1 client using %s\n", apiEndpoint)

	client, err := privilegedaccessmanager.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create privilegedaccessmanager client: %v", err)
	}
	defer closeClient("privilegedaccessmanager", client.Close)

	calls := []callSpec{
		{
			name: "CheckOnboardingStatus",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.CheckOnboardingStatus(ctx, &privilegedaccessmanagerpb.CheckOnboardingStatusRequest{Parent: parent})
				return err
			},
		},
		{
			name: "ListEntitlements",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				it := c.ListEntitlements(ctx, &privilegedaccessmanagerpb.ListEntitlementsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "SearchEntitlements",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				it := c.SearchEntitlements(ctx, &privilegedaccessmanagerpb.SearchEntitlementsRequest{
					Parent:           parent,
					CallerAccessType: privilegedaccessmanagerpb.SearchEntitlementsRequest_CALLER_ACCESS_TYPE_UNSPECIFIED,
					PageSize:         1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetEntitlement",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.GetEntitlement(ctx, &privilegedaccessmanagerpb.GetEntitlementRequest{Name: entitlementName})
				return err
			},
		},
		{
			name: "CreateEntitlement",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.CreateEntitlement(ctx, &privilegedaccessmanagerpb.CreateEntitlementRequest{
					Parent:        parent,
					EntitlementId: entitlementID,
					Entitlement: &privilegedaccessmanagerpb.Entitlement{
						Name:               entitlementName,
						MaxRequestDuration: &durationpb.Duration{Seconds: 3600},
						PrivilegedAccess: &privilegedaccessmanagerpb.PrivilegedAccess{
							AccessType: &privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess_{
								GcpIamAccess: &privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess{
									ResourceType: "cloudresourcemanager.googleapis.com/Project",
									Resource:     resourceName,
									RoleBindings: []*privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess_RoleBinding{
										{Role: "roles/viewer"},
									},
								},
							},
						},
						RequesterJustificationConfig: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig{
							JustificationType: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig_Unstructured_{
								Unstructured: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig_Unstructured{},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateEntitlement",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.UpdateEntitlement(ctx, &privilegedaccessmanagerpb.UpdateEntitlementRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"max_request_duration"}},
					Entitlement: &privilegedaccessmanagerpb.Entitlement{
						Name:               entitlementName,
						MaxRequestDuration: &durationpb.Duration{Seconds: 1800},
						PrivilegedAccess: &privilegedaccessmanagerpb.PrivilegedAccess{
							AccessType: &privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess_{
								GcpIamAccess: &privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess{
									ResourceType: "cloudresourcemanager.googleapis.com/Project",
									Resource:     resourceName,
									RoleBindings: []*privilegedaccessmanagerpb.PrivilegedAccess_GcpIamAccess_RoleBinding{
										{Role: "roles/viewer"},
									},
								},
							},
						},
						RequesterJustificationConfig: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig{
							JustificationType: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig_Unstructured_{
								Unstructured: &privilegedaccessmanagerpb.Entitlement_RequesterJustificationConfig_Unstructured{},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteEntitlement",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.DeleteEntitlement(ctx, &privilegedaccessmanagerpb.DeleteEntitlementRequest{Name: entitlementName})
				return err
			},
		},
		{
			name: "ListGrants",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				it := c.ListGrants(ctx, &privilegedaccessmanagerpb.ListGrantsRequest{
					Parent:   entitlementName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "SearchGrants",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				it := c.SearchGrants(ctx, &privilegedaccessmanagerpb.SearchGrantsRequest{
					Parent:             entitlementName,
					CallerRelationship: privilegedaccessmanagerpb.SearchGrantsRequest_CALLER_RELATIONSHIP_TYPE_UNSPECIFIED,
					PageSize:           1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetGrant",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.GetGrant(ctx, &privilegedaccessmanagerpb.GetGrantRequest{Name: grantName})
				return err
			},
		},
		{
			name: "CreateGrant",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.CreateGrant(ctx, &privilegedaccessmanagerpb.CreateGrantRequest{
					Parent: entitlementName,
					Grant: &privilegedaccessmanagerpb.Grant{
						Name:              grantName,
						RequestedDuration: &durationpb.Duration{Seconds: 1200},
						Justification: &privilegedaccessmanagerpb.Justification{
							Justification: &privilegedaccessmanagerpb.Justification_UnstructuredJustification{
								UnstructuredJustification: "incident response task",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ApproveGrant",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.ApproveGrant(ctx, &privilegedaccessmanagerpb.ApproveGrantRequest{
					Name:   grantName,
					Reason: "approved for incident response",
				})
				return err
			},
		},
		{
			name: "DenyGrant",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.DenyGrant(ctx, &privilegedaccessmanagerpb.DenyGrantRequest{
					Name:   grantName,
					Reason: "denied for policy reasons",
				})
				return err
			},
		},
		{
			name: "RevokeGrant",
			call: func(ctx context.Context, c *privilegedaccessmanager.Client) error {
				_, err := c.RevokeGrant(ctx, &privilegedaccessmanagerpb.RevokeGrantRequest{
					Name:   grantName,
					Reason: "revoked after completion",
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
