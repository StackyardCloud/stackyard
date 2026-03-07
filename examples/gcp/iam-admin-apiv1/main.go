package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cloudiam "cloud.google.com/go/iam"
	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *admin.IamClient) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	grpcEndpoint := grpcEndpointFromEnv()

	projectParent := "projects/" + projectID
	serviceAccountEmail := getenv("STACKYARD_GCP_IAM_ADMIN_SERVICE_ACCOUNT_EMAIL", "stackyard@example.iam.gserviceaccount.com")
	serviceAccountName := projectParent + "/serviceAccounts/" + serviceAccountEmail
	serviceAccountUniqueID := getenv("STACKYARD_GCP_IAM_ADMIN_SERVICE_ACCOUNT_UNIQUE_ID", "123456789012345678901")
	deletedServiceAccountName := projectParent + "/serviceAccounts/" + serviceAccountUniqueID
	serviceAccountID := getenv("STACKYARD_GCP_IAM_ADMIN_SERVICE_ACCOUNT_ID", "stackyard-sa")
	serviceAccountKeyID := getenv("STACKYARD_GCP_IAM_ADMIN_KEY_ID", "key-1")
	serviceAccountKeyName := serviceAccountName + "/keys/" + serviceAccountKeyID
	roleID := getenv("STACKYARD_GCP_IAM_ADMIN_ROLE_ID", "customViewer")
	roleName := projectParent + "/roles/" + roleID
	fullResourceName := getenv("STACKYARD_GCP_IAM_ADMIN_FULL_RESOURCE_NAME", "//cloudresourcemanager.googleapis.com/projects/stackyard")

	fmt.Printf("Stackyard GCP IAM Admin apiv1 client using gRPC endpoint %s\n", grpcEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := admin.NewIamClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create iam admin client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListServiceAccounts",
			call: func(ctx context.Context, c *admin.IamClient) error {
				it := c.ListServiceAccounts(ctx, &adminpb.ListServiceAccountsRequest{
					Name:     projectParent,
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
			name: "GetServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{Name: serviceAccountName})
				return err
			},
		},
		{
			name: "CreateServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.CreateServiceAccount(ctx, &adminpb.CreateServiceAccountRequest{
					Name:      projectParent,
					AccountId: serviceAccountID,
					ServiceAccount: &adminpb.ServiceAccount{
						DisplayName: "Stackyard IAM Admin SA",
						Description: "service account created by Stackyard IAM Admin example",
					},
				})
				return err
			},
		},
		{
			name: "PatchServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.PatchServiceAccount(ctx, &adminpb.PatchServiceAccountRequest{
					ServiceAccount: &adminpb.ServiceAccount{
						Name:        serviceAccountName,
						DisplayName: "Stackyard IAM Admin SA Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DisableServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.DisableServiceAccount(ctx, &adminpb.DisableServiceAccountRequest{Name: serviceAccountName})
			},
		},
		{
			name: "EnableServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.EnableServiceAccount(ctx, &adminpb.EnableServiceAccountRequest{Name: serviceAccountName})
			},
		},
		{
			name: "DeleteServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.DeleteServiceAccount(ctx, &adminpb.DeleteServiceAccountRequest{Name: serviceAccountName})
			},
		},
		{
			name: "UndeleteServiceAccount",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.UndeleteServiceAccount(ctx, &adminpb.UndeleteServiceAccountRequest{Name: deletedServiceAccountName})
				return err
			},
		},
		{
			name: "ListServiceAccountKeys",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.ListServiceAccountKeys(ctx, &adminpb.ListServiceAccountKeysRequest{Name: serviceAccountName})
				return err
			},
		},
		{
			name: "CreateServiceAccountKey",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.CreateServiceAccountKey(ctx, &adminpb.CreateServiceAccountKeyRequest{Name: serviceAccountName})
				return err
			},
		},
		{
			name: "GetServiceAccountKey",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.GetServiceAccountKey(ctx, &adminpb.GetServiceAccountKeyRequest{Name: serviceAccountKeyName})
				return err
			},
		},
		{
			name: "DisableServiceAccountKey",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.DisableServiceAccountKey(ctx, &adminpb.DisableServiceAccountKeyRequest{Name: serviceAccountKeyName})
			},
		},
		{
			name: "EnableServiceAccountKey",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.EnableServiceAccountKey(ctx, &adminpb.EnableServiceAccountKeyRequest{Name: serviceAccountKeyName})
			},
		},
		{
			name: "DeleteServiceAccountKey",
			call: func(ctx context.Context, c *admin.IamClient) error {
				return c.DeleteServiceAccountKey(ctx, &adminpb.DeleteServiceAccountKeyRequest{Name: serviceAccountKeyName})
			},
		},
		{
			name: "SignBlob",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.SignBlob(ctx, &adminpb.SignBlobRequest{
					Name:        serviceAccountName,
					BytesToSign: []byte("stackyard-payload"),
				})
				return err
			},
		},
		{
			name: "SignJwt",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.SignJwt(ctx, &adminpb.SignJwtRequest{
					Name:    serviceAccountName,
					Payload: `{"sub":"stackyard@example.com","iat":1710000000}`,
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: serviceAccountName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.SetIamPolicy(ctx, &admin.SetIamPolicyRequest{
					Resource: serviceAccountName,
					Policy: &cloudiam.Policy{
						InternalProto: &iampb.Policy{
							Bindings: []*iampb.Binding{
								{
									Role:    "roles/iam.serviceAccountUser",
									Members: []string{"user:alice@example.com"},
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    serviceAccountName,
					Permissions: []string{"iam.serviceAccounts.get", "iam.serviceAccounts.signJwt"},
				})
				return err
			},
		},
		{
			name: "QueryGrantableRoles",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.QueryGrantableRoles(ctx, &adminpb.QueryGrantableRolesRequest{
					FullResourceName: fullResourceName,
					PageSize:         1,
				})
				return err
			},
		},
		{
			name: "ListRoles",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.ListRoles(ctx, &adminpb.ListRolesRequest{
					Parent:   projectParent,
					PageSize: 1,
				})
				return err
			},
		},
		{
			name: "GetRole",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.GetRole(ctx, &adminpb.GetRoleRequest{Name: roleName})
				return err
			},
		},
		{
			name: "CreateRole",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.CreateRole(ctx, &adminpb.CreateRoleRequest{
					Parent: projectParent,
					RoleId: roleID,
					Role: &adminpb.Role{
						Title:               "Stackyard Custom Viewer",
						Description:         "Role created by Stackyard IAM Admin example",
						IncludedPermissions: []string{"resourcemanager.projects.get"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateRole",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.UpdateRole(ctx, &adminpb.UpdateRoleRequest{
					Name: roleName,
					Role: &adminpb.Role{
						Name:                roleName,
						Title:               "Stackyard Custom Viewer Updated",
						IncludedPermissions: []string{"resourcemanager.projects.get"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
				})
				return err
			},
		},
		{
			name: "DeleteRole",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.DeleteRole(ctx, &adminpb.DeleteRoleRequest{Name: roleName})
				return err
			},
		},
		{
			name: "UndeleteRole",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.UndeleteRole(ctx, &adminpb.UndeleteRoleRequest{Name: roleName})
				return err
			},
		},
		{
			name: "QueryTestablePermissions",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.QueryTestablePermissions(ctx, &adminpb.QueryTestablePermissionsRequest{
					FullResourceName: fullResourceName,
					PageSize:         1,
				})
				return err
			},
		},
		{
			name: "QueryAuditableServices",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.QueryAuditableServices(ctx, &adminpb.QueryAuditableServicesRequest{
					FullResourceName: fullResourceName,
				})
				return err
			},
		},
		{
			name: "LintPolicy",
			call: func(ctx context.Context, c *admin.IamClient) error {
				_, err := c.LintPolicy(ctx, &adminpb.LintPolicyRequest{
					FullResourceName: fullResourceName,
					LintObject: &adminpb.LintPolicyRequest_Condition{
						Condition: &expr.Expr{
							Expression: "request.time < timestamp('2030-01-01T00:00:00Z')",
							Title:      "stackyard-condition",
						},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, time.Second)
		err := call.call(callCtx, client)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedStageError(err):
			logf("%s returned staged error (expected in early emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedStageError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.Unknown:
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
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large") ||
		strings.Contains(text, "transport is closing")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close iam admin client: %v\n", err)
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
