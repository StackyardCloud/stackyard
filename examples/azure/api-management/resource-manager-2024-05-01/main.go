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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_APIM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APIM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-apim")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_APIM_SERVICE", "apim-a")
	apiID := getenv("STACKYARD_AZURE_APIM_API", "echo-api")
	operationID := getenv("STACKYARD_AZURE_APIM_OPERATION", "get-echo")
	productID := getenv("STACKYARD_AZURE_APIM_PRODUCT", "starter")
	userID := getenv("STACKYARD_AZURE_APIM_USER", "user-1")
	namedValueID := getenv("STACKYARD_AZURE_APIM_NAMED_VALUE", "kv-endpoint")
	gatewayID := getenv("STACKYARD_AZURE_APIM_GATEWAY", "gw-a")
	notificationID := getenv("STACKYARD_AZURE_APIM_NOTIFICATION", "NewCommentNotificationMessage")
	recipientID := getenv("STACKYARD_AZURE_APIM_RECIPIENT", "dev@example.com")

	fmt.Printf("Stackyard Azure API Management Resource Manager typed SDK example using %s\n", endpoint)

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

	operationsClient, err := armapimanagement.NewOperationsClient(credential, armOptions)
	if err != nil {
		exitf("create OperationsClient failed: %v", err)
	}
	skusClient, err := armapimanagement.NewSKUsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create SKUsClient failed: %v", err)
	}
	serviceClient, err := armapimanagement.NewServiceClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ServiceClient failed: %v", err)
	}
	apiClient, err := armapimanagement.NewAPIClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create APIClient failed: %v", err)
	}
	apiOperationClient, err := armapimanagement.NewAPIOperationClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create APIOperationClient failed: %v", err)
	}
	apiOperationPolicyClient, err := armapimanagement.NewAPIOperationPolicyClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create APIOperationPolicyClient failed: %v", err)
	}
	productClient, err := armapimanagement.NewProductClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ProductClient failed: %v", err)
	}
	userClient, err := armapimanagement.NewUserClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create UserClient failed: %v", err)
	}
	namedValueClient, err := armapimanagement.NewNamedValueClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create NamedValueClient failed: %v", err)
	}
	gatewayClient, err := armapimanagement.NewGatewayClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create GatewayClient failed: %v", err)
	}
	notificationRecipientEmailClient, err := armapimanagement.NewNotificationRecipientEmailClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create NotificationRecipientEmailClient failed: %v", err)
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
			name: "ListSKUs",
			run: func() error {
				pager := skusClient.NewListPager(nil)
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
				_, err := serviceClient.BeginCreateOrUpdate(ctx, resourceGroup, serviceName, armapimanagement.ServiceResource{
					Location: to.Ptr(location),
					SKU: &armapimanagement.ServiceSKUProperties{
						Name:     to.Ptr(armapimanagement.SKUTypeDeveloper),
						Capacity: to.Ptr[int32](1),
					},
					Properties: &armapimanagement.ServiceProperties{
						PublisherEmail: to.Ptr("dev@example.com"),
						PublisherName:  to.Ptr("Stackyard"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetService",
			run: func() error {
				_, err := serviceClient.Get(ctx, resourceGroup, serviceName, nil)
				return err
			},
		},
		{
			name: "ApplyNetworkConfigurationUpdates",
			run: func() error {
				_, err := serviceClient.BeginApplyNetworkConfigurationUpdates(ctx, resourceGroup, serviceName, &armapimanagement.ServiceClientBeginApplyNetworkConfigurationUpdatesOptions{
					Parameters: &armapimanagement.ServiceApplyNetworkConfigurationParameters{
						Location: to.Ptr(location),
					},
				})
				return err
			},
		},
		{
			name: "CreateAPI",
			run: func() error {
				_, err := apiClient.BeginCreateOrUpdate(ctx, resourceGroup, serviceName, apiID, armapimanagement.APICreateOrUpdateParameter{
					Properties: &armapimanagement.APICreateOrUpdateProperties{
						DisplayName: to.Ptr("Echo API"),
						Path:        to.Ptr("echo"),
						Protocols: []*armapimanagement.Protocol{
							to.Ptr(armapimanagement.ProtocolHTTPS),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAPIOperation",
			run: func() error {
				_, err := apiOperationClient.CreateOrUpdate(ctx, resourceGroup, serviceName, apiID, operationID, armapimanagement.OperationContract{
					Properties: &armapimanagement.OperationContractProperties{
						DisplayName: to.Ptr("Get Echo"),
						Method:      to.Ptr(http.MethodGet),
						URLTemplate: to.Ptr("/echo"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAPIOperationPolicy",
			run: func() error {
				_, err := apiOperationPolicyClient.CreateOrUpdate(ctx, resourceGroup, serviceName, apiID, operationID, armapimanagement.PolicyIDNamePolicy, armapimanagement.PolicyContract{
					Properties: &armapimanagement.PolicyContractProperties{
						Format: to.Ptr(armapimanagement.PolicyContentFormatRawxml),
						Value:  to.Ptr("<policies></policies>"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateProduct",
			run: func() error {
				_, err := productClient.CreateOrUpdate(ctx, resourceGroup, serviceName, productID, armapimanagement.ProductContract{
					Properties: &armapimanagement.ProductContractProperties{
						DisplayName:          to.Ptr("Starter"),
						SubscriptionRequired: to.Ptr(false),
						State:                to.Ptr(armapimanagement.ProductStatePublished),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateUser",
			run: func() error {
				_, err := userClient.CreateOrUpdate(ctx, resourceGroup, serviceName, userID, armapimanagement.UserCreateParameters{
					Properties: &armapimanagement.UserCreateParameterProperties{
						FirstName: to.Ptr("Dev"),
						LastName:  to.Ptr("User"),
						Email:     to.Ptr("dev@example.com"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateNamedValue",
			run: func() error {
				_, err := namedValueClient.BeginCreateOrUpdate(ctx, resourceGroup, serviceName, namedValueID, armapimanagement.NamedValueCreateContract{
					Properties: &armapimanagement.NamedValueCreateContractProperties{
						DisplayName: to.Ptr("KeyVaultEndpoint"),
						Value:       to.Ptr("https://example.vault.azure.net/"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateGateway",
			run: func() error {
				_, err := gatewayClient.CreateOrUpdate(ctx, resourceGroup, serviceName, gatewayID, armapimanagement.GatewayContract{
					Properties: &armapimanagement.GatewayContractProperties{
						Description: to.Ptr("gateway a"),
						LocationData: &armapimanagement.ResourceLocationDataContract{
							Name: to.Ptr(location),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "HeadNotificationRecipientEmail",
			run: func() error {
				_, err := notificationRecipientEmailClient.CheckEntityExists(ctx, resourceGroup, serviceName, armapimanagement.NotificationName(notificationID), recipientID, nil)
				return err
			},
		},
		{
			name: "DeleteService",
			run: func() error {
				_, err := serviceClient.BeginDelete(ctx, resourceGroup, serviceName, nil)
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
		fmt.Println("All api-management resource-manager routes are staged in this Stackyard build.")
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
