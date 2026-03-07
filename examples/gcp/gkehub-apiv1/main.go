package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
	"cloud.google.com/go/gkehub/apiv1beta1/gkehubpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
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
	call func(context.Context, *gkehub.GkeHubMembershipClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	membershipID := getenv("STACKYARD_GCP_GKEHUB_MEMBERSHIP_ID", "cluster-a")
	operationID := getenv("STACKYARD_GCP_GKEHUB_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	membershipName := locationName + "/memberships/" + membershipID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP GKE Hub apiv1beta1 client using %s\n", apiEndpoint)

	client, err := gkehub.NewGkeHubMembershipRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create gkehub membership client: %v", err)
	}
	defer closeClient("gkehub membership client", client.Close)

	calls := []callSpec{
		{
			name: "ListMemberships",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				it := c.ListMemberships(ctx, &gkehubpb.ListMembershipsRequest{
					Parent:   locationName,
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
			name: "GetMembership",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GetMembership(ctx, &gkehubpb.GetMembershipRequest{Name: membershipName})
				return err
			},
		},
		{
			name: "CreateMembership",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.CreateMembership(ctx, &gkehubpb.CreateMembershipRequest{
					Parent:       locationName,
					MembershipId: membershipID,
					Resource: &gkehubpb.Membership{
						Name:        membershipName,
						Description: "stackyard gkehub membership",
						Labels: map[string]string{
							"team": "platform",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateMembership",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.UpdateMembership(ctx, &gkehubpb.UpdateMembershipRequest{
					Name:       membershipName,
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					Resource: &gkehubpb.Membership{
						Name:        membershipName,
						Description: "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteMembership",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.DeleteMembership(ctx, &gkehubpb.DeleteMembershipRequest{Name: membershipName})
				return err
			},
		},
		{
			name: "GenerateConnectManifest",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GenerateConnectManifest(ctx, &gkehubpb.GenerateConnectManifestRequest{
					Name:      membershipName,
					Version:   "v1",
					IsUpgrade: true,
					ConnectAgent: &gkehubpb.ConnectAgent{
						Namespace: "gke-connect",
					},
				})
				return err
			},
		},
		{
			name: "ValidateExclusivity",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.ValidateExclusivity(ctx, &gkehubpb.ValidateExclusivityRequest{
					Parent:             locationName,
					IntendedMembership: membershipID,
				})
				return err
			},
		},
		{
			name: "GenerateExclusivityManifest",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GenerateExclusivityManifest(ctx, &gkehubpb.GenerateExclusivityManifestRequest{
					Name: membershipName,
				})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
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
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: membershipName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: membershipName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    membershipName,
					Permissions: []string{"gkehub.memberships.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			name: "CancelOperation",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *gkehub.GkeHubMembershipClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
