package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AUTHORIZATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AUTHORIZATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-authz")
	roleDefinitionName := getenv("STACKYARD_AZURE_AUTHORIZATION_ROLE_DEFINITION", "rd-storage-reader")
	resourceRoleAssignmentName := getenv("STACKYARD_AZURE_AUTHORIZATION_RESOURCE_ASSIGNMENT", "ra-storage-reader")
	subscriptionRoleAssignmentName := getenv("STACKYARD_AZURE_AUTHORIZATION_SUBSCRIPTION_ASSIGNMENT", "ra-subscription-reader")

	subscriptionScope := "/subscriptions/" + subscriptionID
	resourceGroupScope := subscriptionScope + "/resourceGroups/" + resourceGroup
	resourceScope := resourceGroupScope + "/providers/Microsoft.Storage/storageAccounts/saauth"
	roleDefinitionID := subscriptionScope + "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionName
	subscriptionRoleAssignmentID := subscriptionScope + "/providers/Microsoft.Authorization/roleAssignments/" + subscriptionRoleAssignmentName

	fmt.Printf("Stackyard Azure Authorization typed SDK example using %s\n", endpoint)

	credential := azsdkshim.StaticTokenCredential{}
	armOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			InsecureAllowCredentialWithHTTP: true,
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: endpoint,
				Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
					cloud.ResourceManager: {
						Endpoint: endpoint,
						Audience: endpoint,
					},
				},
			},
			Transport: azsdkshim.NewTransport("/azure", account, subscriptionKey),
		},
	}

	permissionsClient, err := armauthorization.NewPermissionsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create PermissionsClient failed: %v", err)
	}
	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create RoleAssignmentsClient failed: %v", err)
	}
	roleDefinitionsClient, err := armauthorization.NewRoleDefinitionsClient(credential, armOptions)
	if err != nil {
		exitf("create RoleDefinitionsClient failed: %v", err)
	}

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "CreateRoleDefinition",
			run: func() error {
				_, err := roleDefinitionsClient.CreateOrUpdate(ctx, subscriptionScope, roleDefinitionName, armauthorization.RoleDefinition{
					Properties: &armauthorization.RoleDefinitionProperties{
						RoleName:    to.Ptr("Storage Reader"),
						Description: to.Ptr("Read access to one storage account"),
						RoleType:    to.Ptr("CustomRole"),
						Permissions: []*armauthorization.Permission{
							{
								Actions:    []*string{to.Ptr("Microsoft.Storage/storageAccounts/read")},
								NotActions: []*string{},
							},
						},
						AssignableScopes: []*string{
							to.Ptr(subscriptionScope),
							to.Ptr(resourceGroupScope),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetRoleDefinition",
			run: func() error {
				_, err := roleDefinitionsClient.Get(ctx, subscriptionScope, roleDefinitionName, nil)
				return err
			},
		},
		{
			name: "GetRoleDefinitionByID",
			run: func() error {
				_, err := roleDefinitionsClient.GetByID(ctx, roleDefinitionID, nil)
				return err
			},
		},
		{
			name: "ListRoleDefinitions",
			run: func() error {
				pager := roleDefinitionsClient.NewListPager(subscriptionScope, &armauthorization.RoleDefinitionsClientListOptions{
					Filter: to.Ptr("roleName eq 'Storage Reader'"),
				})
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateRoleAssignment",
			run: func() error {
				_, err := roleAssignmentsClient.Create(ctx, resourceScope, resourceRoleAssignmentName, armauthorization.RoleAssignmentCreateParameters{
					Properties: &armauthorization.RoleAssignmentProperties{
						PrincipalID:      to.Ptr("11111111-1111-1111-1111-111111111111"),
						RoleDefinitionID: to.Ptr(roleDefinitionID),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateRoleAssignmentByID",
			run: func() error {
				_, err := roleAssignmentsClient.CreateByID(ctx, subscriptionRoleAssignmentID, armauthorization.RoleAssignmentCreateParameters{
					Properties: &armauthorization.RoleAssignmentProperties{
						PrincipalID:      to.Ptr("22222222-2222-2222-2222-222222222222"),
						RoleDefinitionID: to.Ptr(roleDefinitionID),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetRoleAssignment",
			run: func() error {
				_, err := roleAssignmentsClient.Get(ctx, resourceScope, resourceRoleAssignmentName, nil)
				return err
			},
		},
		{
			name: "GetRoleAssignmentByID",
			run: func() error {
				_, err := roleAssignmentsClient.GetByID(ctx, subscriptionRoleAssignmentID, nil)
				return err
			},
		},
		{
			name: "ListRoleAssignmentsForResource",
			run: func() error {
				pager := roleAssignmentsClient.NewListForResourcePager(resourceGroup, "Microsoft.Storage", "", "storageAccounts", "saauth", nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListRoleAssignmentsForResourceGroup",
			run: func() error {
				pager := roleAssignmentsClient.NewListForResourceGroupPager(resourceGroup, &armauthorization.RoleAssignmentsClientListForResourceGroupOptions{
					Filter: to.Ptr("principalId eq '11111111-1111-1111-1111-111111111111'"),
				})
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListRoleAssignmentsForScope",
			run: func() error {
				pager := roleAssignmentsClient.NewListForScopePager(resourceScope, &armauthorization.RoleAssignmentsClientListForScopeOptions{
					Filter: to.Ptr("atScope()"),
				})
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListRoleAssignmentsForSubscription",
			run: func() error {
				pager := roleAssignmentsClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListPermissionsForResourceGroup",
			run: func() error {
				pager := permissionsClient.NewListForResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListPermissionsForResource",
			run: func() error {
				pager := permissionsClient.NewListForResourcePager(resourceGroup, "Microsoft.Storage", "", "storageAccounts", "saauth", nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "DeleteRoleAssignmentByID",
			run: func() error {
				_, err := roleAssignmentsClient.DeleteByID(ctx, subscriptionRoleAssignmentID, nil)
				return err
			},
		},
		{
			name: "DeleteRoleAssignment",
			run: func() error {
				_, err := roleAssignmentsClient.Delete(ctx, resourceScope, resourceRoleAssignmentName, nil)
				return err
			},
		},
		{
			name: "DeleteRoleDefinition",
			run: func() error {
				_, err := roleDefinitionsClient.Delete(ctx, subscriptionScope, roleDefinitionName, nil)
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.run(); err != nil {
			var responseErr *azcore.ResponseError
			if errors.As(err, &responseErr) {
				exitf("%s failed: status=%d code=%s err=%v", call.name, responseErr.StatusCode, responseErr.ErrorCode, err)
			}
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: ok\n", call.name)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
