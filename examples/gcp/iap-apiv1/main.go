package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	iap "cloud.google.com/go/iap/apiv1"
	"cloud.google.com/go/iap/apiv1/iappb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *iap.IdentityAwareProxyAdminClient, *iap.IdentityAwareProxyOAuthClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	projectNumber := getenv("STACKYARD_GCP_PROJECT_NUMBER", "123456789")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	brandID := getenv("STACKYARD_GCP_IAP_BRAND_ID", "brand-1")
	clientID := getenv("STACKYARD_GCP_IAP_CLIENT_ID", "client-1")
	destGroupID := getenv("STACKYARD_GCP_IAP_DEST_GROUP_ID", "corp-dest")

	iapResource := fmt.Sprintf("projects/%s/iap_web", projectNumber)
	projectParent := fmt.Sprintf("projects/%s", projectID)
	brandName := fmt.Sprintf("%s/brands/%s", projectParent, brandID)
	clientName := fmt.Sprintf("%s/identityAwareProxyClients/%s", brandName, clientID)
	tunnelParent := fmt.Sprintf("projects/%s/iap_tunnel/locations/%s", projectNumber, location)
	destGroupName := fmt.Sprintf("%s/destGroups/%s", tunnelParent, destGroupID)

	fmt.Printf("Stackyard GCP Cloud IAP apiv1 clients using %s\n", apiEndpoint)

	adminClient, err := iap.NewIdentityAwareProxyAdminRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iap admin client: %v", err)
	}
	defer closeClient("iap admin", adminClient.Close)

	oauthClient, err := iap.NewIdentityAwareProxyOAuthRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iap oauth client: %v", err)
	}
	defer closeClient("iap oauth", oauthClient.Close)

	calls := []callSpec{
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: iapResource})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: iapResource,
					Policy: &iampb.Policy{
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/iap.httpsResourceAccessor",
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
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    iapResource,
					Permissions: []string{"iap.web.getSettings", "iap.web.updateSettings"},
				})
				return err
			},
		},
		{
			name: "GetIapSettings",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.GetIapSettings(ctx, &iappb.GetIapSettingsRequest{Name: iapResource})
				return err
			},
		},
		{
			name: "UpdateIapSettings",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.UpdateIapSettings(ctx, &iappb.UpdateIapSettingsRequest{
					IapSettings: &iappb.IapSettings{
						Name: iapResource,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"access_settings"},
					},
				})
				return err
			},
		},
		{
			name: "ValidateIapAttributeExpression",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.ValidateIapAttributeExpression(ctx, &iappb.ValidateIapAttributeExpressionRequest{
					Name:       iapResource,
					Expression: "request.auth.access_levels.hasAny(['accessPolicies/123/accessLevels/corp'])",
				})
				return err
			},
		},
		{
			name: "ListTunnelDestGroups",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				it := adminClient.ListTunnelDestGroups(ctx, &iappb.ListTunnelDestGroupsRequest{
					Parent:   tunnelParent,
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
			name: "CreateTunnelDestGroup",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.CreateTunnelDestGroup(ctx, &iappb.CreateTunnelDestGroupRequest{
					Parent:            tunnelParent,
					TunnelDestGroupId: destGroupID,
					TunnelDestGroup: &iappb.TunnelDestGroup{
						Cidrs: []string{"10.0.0.0/24"},
						Fqdns: []string{"internal.example.local"},
					},
				})
				return err
			},
		},
		{
			name: "GetTunnelDestGroup",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.GetTunnelDestGroup(ctx, &iappb.GetTunnelDestGroupRequest{
					Name: destGroupName,
				})
				return err
			},
		},
		{
			name: "UpdateTunnelDestGroup",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				_, err := adminClient.UpdateTunnelDestGroup(ctx, &iappb.UpdateTunnelDestGroupRequest{
					TunnelDestGroup: &iappb.TunnelDestGroup{
						Name:  destGroupName,
						Fqdns: []string{"edge.internal.example.local"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"fqdns"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteTunnelDestGroup",
			call: func(ctx context.Context, adminClient *iap.IdentityAwareProxyAdminClient, _ *iap.IdentityAwareProxyOAuthClient) error {
				return adminClient.DeleteTunnelDestGroup(ctx, &iappb.DeleteTunnelDestGroupRequest{
					Name: destGroupName,
				})
			},
		},
		{
			name: "ListBrands",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.ListBrands(ctx, &iappb.ListBrandsRequest{
					Parent: projectParent,
				})
				return err
			},
		},
		{
			name: "CreateBrand",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.CreateBrand(ctx, &iappb.CreateBrandRequest{
					Parent: projectParent,
					Brand: &iappb.Brand{
						SupportEmail:     "stackyard@example.com",
						ApplicationTitle: "Stackyard IAP",
					},
				})
				return err
			},
		},
		{
			name: "GetBrand",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.GetBrand(ctx, &iappb.GetBrandRequest{Name: brandName})
				return err
			},
		},
		{
			name: "CreateIdentityAwareProxyClient",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.CreateIdentityAwareProxyClient(ctx, &iappb.CreateIdentityAwareProxyClientRequest{
					Parent: brandName,
					IdentityAwareProxyClient: &iappb.IdentityAwareProxyClient{
						DisplayName: "Stackyard IAP Client",
					},
				})
				return err
			},
		},
		{
			name: "ListIdentityAwareProxyClients",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				it := oauthClient.ListIdentityAwareProxyClients(ctx, &iappb.ListIdentityAwareProxyClientsRequest{
					Parent:   brandName,
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
			name: "GetIdentityAwareProxyClient",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.GetIdentityAwareProxyClient(ctx, &iappb.GetIdentityAwareProxyClientRequest{
					Name: clientName,
				})
				return err
			},
		},
		{
			name: "ResetIdentityAwareProxyClientSecret",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				_, err := oauthClient.ResetIdentityAwareProxyClientSecret(ctx, &iappb.ResetIdentityAwareProxyClientSecretRequest{
					Name: clientName,
				})
				return err
			},
		},
		{
			name: "DeleteIdentityAwareProxyClient",
			call: func(ctx context.Context, _ *iap.IdentityAwareProxyAdminClient, oauthClient *iap.IdentityAwareProxyOAuthClient) error {
				return oauthClient.DeleteIdentityAwareProxyClient(ctx, &iappb.DeleteIdentityAwareProxyClientRequest{
					Name: clientName,
				})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, adminClient, oauthClient)
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
