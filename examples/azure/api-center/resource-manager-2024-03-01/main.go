package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apicenter/armapicenter"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_APICENTER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APICENTER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-apic")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_APICENTER_SERVICE", "contoso")
	workspaceName := getenv("STACKYARD_AZURE_APICENTER_WORKSPACE", "default")
	apiName := getenv("STACKYARD_AZURE_APICENTER_API", "echo-api")
	versionName := getenv("STACKYARD_AZURE_APICENTER_VERSION", "2023-01-01")
	definitionName := getenv("STACKYARD_AZURE_APICENTER_DEFINITION", "openapi")
	deploymentName := getenv("STACKYARD_AZURE_APICENTER_DEPLOYMENT", "production")
	environmentName := getenv("STACKYARD_AZURE_APICENTER_ENVIRONMENT", "public")
	metadataSchemaName := getenv("STACKYARD_AZURE_APICENTER_METADATA_SCHEMA", "author")

	fmt.Printf("Stackyard Azure API Center Resource Manager typed SDK example using %s\n", endpoint)

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

	operationsClient, err := armapicenter.NewOperationsClient(credential, armOptions)
	if err != nil {
		exitf("create OperationsClient failed: %v", err)
	}
	servicesClient, err := armapicenter.NewServicesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ServicesClient failed: %v", err)
	}
	workspacesClient, err := armapicenter.NewWorkspacesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create WorkspacesClient failed: %v", err)
	}
	apisClient, err := armapicenter.NewApisClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ApisClient failed: %v", err)
	}
	apiVersionsClient, err := armapicenter.NewAPIVersionsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create APIVersionsClient failed: %v", err)
	}
	apiDefinitionsClient, err := armapicenter.NewAPIDefinitionsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create APIDefinitionsClient failed: %v", err)
	}
	deploymentsClient, err := armapicenter.NewDeploymentsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create DeploymentsClient failed: %v", err)
	}
	environmentsClient, err := armapicenter.NewEnvironmentsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create EnvironmentsClient failed: %v", err)
	}
	metadataSchemasClient, err := armapicenter.NewMetadataSchemasClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create MetadataSchemasClient failed: %v", err)
	}

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListOperations",
			run: func() error {
				pager := operationsClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateService",
			run: func() error {
				_, err := servicesClient.CreateOrUpdate(ctx, resourceGroup, serviceName, armapicenter.Service{
					Location: to.Ptr(location),
				}, nil)
				return err
			},
		},
		{
			name: "GetService",
			run: func() error {
				_, err := servicesClient.Get(ctx, resourceGroup, serviceName, nil)
				return err
			},
		},
		{
			name: "UpdateService",
			run: func() error {
				_, err := servicesClient.Update(ctx, resourceGroup, serviceName, armapicenter.ServiceUpdate{
					Tags: map[string]*string{
						"env": to.Ptr("dev"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListServicesByResourceGroup",
			run: func() error {
				pager := servicesClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListServicesBySubscription",
			run: func() error {
				pager := servicesClient.NewListBySubscriptionPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ExportMetadataSchema",
			run: func() error {
				_, err := servicesClient.BeginExportMetadataSchema(ctx, resourceGroup, serviceName, armapicenter.MetadataSchemaExportRequest{
					AssignedTo: to.Ptr(armapicenter.MetadataAssignmentEntityAPI),
				}, nil)
				return err
			},
		},
		{
			name: "CreateWorkspace",
			run: func() error {
				_, err := workspacesClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, armapicenter.Workspace{
					Properties: &armapicenter.WorkspaceProperties{
						Title: to.Ptr(workspaceName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAPI",
			run: func() error {
				_, err := apisClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, apiName, armapicenter.API{
					Properties: &armapicenter.APIProperties{
						Kind:  to.Ptr(armapicenter.APIKindRest),
						Title: to.Ptr(apiName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAPIVersion",
			run: func() error {
				_, err := apiVersionsClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, apiName, versionName, armapicenter.APIVersion{
					Properties: &armapicenter.APIVersionProperties{
						LifecycleStage: to.Ptr(armapicenter.LifecycleStageDesign),
						Title:          to.Ptr(versionName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAPIDefinition",
			run: func() error {
				_, err := apiDefinitionsClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, apiName, versionName, definitionName, armapicenter.APIDefinition{
					Properties: &armapicenter.APIDefinitionProperties{
						Title: to.Ptr(definitionName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ExportSpecification",
			run: func() error {
				_, err := apiDefinitionsClient.BeginExportSpecification(ctx, resourceGroup, serviceName, workspaceName, apiName, versionName, definitionName, nil)
				return err
			},
		},
		{
			name: "ImportSpecification",
			run: func() error {
				_, err := apiDefinitionsClient.BeginImportSpecification(ctx, resourceGroup, serviceName, workspaceName, apiName, versionName, definitionName, armapicenter.APISpecImportRequest{
					Format: to.Ptr(armapicenter.APISpecImportSourceFormatInline),
					Value:  to.Ptr("{\"openapi\":\"3.0.0\"}"),
				}, nil)
				return err
			},
		},
		{
			name: "CreateDeployment",
			run: func() error {
				_, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, apiName, deploymentName, armapicenter.Deployment{
					Properties: &armapicenter.DeploymentProperties{
						Title: to.Ptr(deploymentName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateEnvironment",
			run: func() error {
				_, err := environmentsClient.CreateOrUpdate(ctx, resourceGroup, serviceName, workspaceName, environmentName, armapicenter.Environment{
					Properties: &armapicenter.EnvironmentProperties{
						Kind:  to.Ptr(armapicenter.EnvironmentKindDevelopment),
						Title: to.Ptr(environmentName),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateMetadataSchema",
			run: func() error {
				_, err := metadataSchemasClient.CreateOrUpdate(ctx, resourceGroup, serviceName, metadataSchemaName, armapicenter.MetadataSchema{
					Properties: &armapicenter.MetadataSchemaProperties{
						Schema: to.Ptr("{\"type\":\"string\"}"),
					},
				}, nil)
				return err
			},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		err := call.run()
		switch {
		case err == nil:
			fmt.Printf("%s: ok\n", call.name)
		case isNotImplemented(err):
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All api-center resource-manager routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func isNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		if responseErr.StatusCode == http.StatusNotImplemented {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(responseErr.ErrorCode), "NotImplemented") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
