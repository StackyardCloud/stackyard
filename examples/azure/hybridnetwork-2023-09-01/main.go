package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const (
	resourcesAPIVersion     = "2021-04-01"
	hybridNetworkAPIVersion = "2023-09-01"
)

type armClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newARMClient(endpoint, account, subscriptionKey string) *armClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"hybridnetwork-2023-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}
	if len(expectedStatuses) > 0 {
		ok := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-hybridnetwork-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	publisherName := getenv("STACKYARD_AZURE_AOSM_PUBLISHER", "stackyard-publisher")

	fmt.Printf("Stackyard Azure Hybrid Network (hybridnetwork-2023-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	providers := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.HybridNetwork/"
	armProviders := strings.TrimPrefix(providers, "/azure")
	publisher := providers + "publishers/" + publisherName
	armPublisher := armProviders + "publishers/" + publisherName
	manifest := publisher + "/artifactStores/acr/artifactManifests/nginx-manifest"
	nfdv := publisher + "/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0"
	nsdv := publisher + "/networkServiceDesignGroups/edge/networkServiceDesignVersions/1.0.0"
	cgs := publisher + "/configurationGroupSchemas/edge-config"
	customLocation := strings.TrimPrefix(base, "/azure") + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ExtendedLocation/customLocations/edge-cluster"
	resource := func(props map[string]any) map[string]any {
		return map[string]any{"location": region, "properties": props}
	}
	helm := map[string]any{
		"name": "nginx", "artifactType": "HelmPackage",
		"artifactProfile": map[string]any{
			"artifactStore":       map[string]any{"id": armPublisher + "/artifactStores/acr"},
			"helmArtifactProfile": map[string]any{"helmPackageName": "nginx", "helmPackageVersionRange": "~1.0.0"},
		},
		"deployParametersMappingRuleProfile": map[string]any{"applicationEnablement": "Enabled", "helmMappingRuleProfile": map[string]any{"releaseNamespace": "nginx", "releaseName": "nginx", "helmPackageVersion": "1.0.0", "values": "{}"}},
	}
	definition := resource(map[string]any{
		"networkFunctionType":     "ContainerizedNetworkFunction",
		"deployParameters":        `{"type":"object","properties":{"replicas":{"type":"integer"}},"required":["replicas"]}`,
		"networkFunctionTemplate": map[string]any{"nfviType": "AzureArcKubernetes", "networkFunctionApplications": []any{helm}},
	})
	design := resource(map[string]any{
		"configurationGroupSchemaReferences": map[string]any{"edge": map[string]any{"id": armPublisher + "/configurationGroupSchemas/edge-config"}},
		"nfvisFromSite":                      map[string]any{"edge": map[string]any{"name": "edge-cluster", "type": "AzureArcKubernetes"}},
		"resourceElementTemplates": []any{map[string]any{"name": "nginx-nf", "type": "NetworkFunctionDefinition", "configuration": map[string]any{
			"templateType": "ArmTemplate", "parameterValues": `{"replicas":"{configurationparameters('edge').replicas}"}`,
			"artifactProfile": map[string]any{"artifactName": "nginx-nf-template", "artifactVersion": "1.0.0", "artifactStoreReference": map[string]any{"id": armPublisher + "/artifactStores/acr"}},
		}}},
	})
	networkFunction := resource(map[string]any{
		"networkFunctionDefinitionVersionResourceReference": map[string]any{"id": strings.TrimPrefix(nfdv, "/azure"), "idType": "Open"},
		"nfviType": "AzureArcKubernetes", "nfviId": customLocation, "allowSoftwareUpdate": false,
		"configurationType": "Secret", "secretDeploymentValues": `{"replicas":2}`,
		"roleOverrideValues": []any{`{"name":"nginx","deployParametersMappingRuleProfile":{"helmMappingRuleProfile":{"releaseNamespace":"edge"}}}`},
	})
	siteNetworkService := map[string]any{"location": region, "sku": map[string]any{"name": "Standard"}, "properties": map[string]any{
		"siteReference": map[string]any{"id": armProviders + "sites/edge-site"},
		"networkServiceDesignVersionResourceReference":  map[string]any{"id": strings.TrimPrefix(nsdv, "/azure"), "idType": "Open"},
		"desiredStateConfigurationGroupValueReferences": map[string]any{"edge": map[string]any{"id": armProviders + "configurationGroupValues/edge-values"}},
	}}
	q := "?api-version=" + hybridNetworkAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreatePublisher", method: http.MethodPut, path: publisher + q, payload: resource(map[string]any{"scope": "Private"}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateArtifactStore", method: http.MethodPut, path: publisher + "/artifactStores/acr" + q, payload: resource(map[string]any{"storeType": "AzureContainerRegistry", "replicationStrategy": "SingleReplication"}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateArtifactManifest", method: http.MethodPut, path: manifest + q, payload: resource(map[string]any{"artifacts": []any{map[string]any{"artifactName": "nginx", "artifactType": "OCIArtifact", "artifactVersion": "1.0.0"}, map[string]any{"artifactName": "nginx-nf-template", "artifactType": "ArmTemplate", "artifactVersion": "1.0.0"}}}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListArtifactCredential", method: http.MethodPost, path: manifest + "/listCredential" + q, statuses: []int{http.StatusOK}},
		{name: "MarkManifestUploaded", method: http.MethodPost, path: manifest + "/updateState" + q, payload: map[string]any{"artifactManifestState": "Uploaded"}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "ListArtifactManifests", method: http.MethodGet, path: publisher + "/artifactStores/acr/artifactManifests" + q, statuses: []int{http.StatusOK}},
		{name: "CreateDefinitionGroup", method: http.MethodPut, path: publisher + "/networkFunctionDefinitionGroups/nginx" + q, payload: resource(map[string]any{"description": "nginx ingress"}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDefinitionVersion", method: http.MethodPut, path: nfdv + q, payload: definition, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ActivateDefinitionVersion", method: http.MethodPost, path: nfdv + "/updateState" + q, payload: map[string]any{"versionState": "Active"}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "CreateConfigurationGroupSchema", method: http.MethodPut, path: cgs + q, payload: resource(map[string]any{"description": "edge settings", "schemaDefinition": `{"type":"object","properties":{"replicas":{"type":"integer"}},"required":["replicas"]}`}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ActivateConfigurationGroupSchema", method: http.MethodPost, path: cgs + "/updateState" + q, payload: map[string]any{"versionState": "Active"}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "CreateDesignGroup", method: http.MethodPut, path: publisher + "/networkServiceDesignGroups/edge" + q, payload: resource(map[string]any{"description": "edge service"}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDesignVersion", method: http.MethodPut, path: nsdv + q, payload: design, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ActivateDesignVersion", method: http.MethodPost, path: nsdv + "/updateState" + q, payload: map[string]any{"versionState": "Active"}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "CreateSite", method: http.MethodPut, path: providers + "sites/edge-site" + q, payload: resource(map[string]any{"nfvis": []any{map[string]any{"nfviType": "AzureArcKubernetes", "name": "edge-cluster", "customLocationReference": map[string]any{"id": customLocation}}}}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateConfigurationGroupValue", method: http.MethodPut, path: providers + "configurationGroupValues/edge-values" + q, payload: resource(map[string]any{"configurationType": "Open", "configurationValue": `{"replicas":2}`, "configurationGroupSchemaResourceReference": map[string]any{"id": strings.TrimPrefix(cgs, "/azure"), "idType": "Open"}}), statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateSiteNetworkService", method: http.MethodPut, path: providers + "siteNetworkServices/edge-service" + q, payload: siteNetworkService, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "TagSiteNetworkService", method: http.MethodPatch, path: providers + "siteNetworkServices/edge-service" + q, payload: map[string]any{"tags": map[string]any{"owner": "stackyard"}}, statuses: []int{http.StatusOK}},
		{name: "CreateNetworkFunction", method: http.MethodPut, path: providers + "networkFunctions/nginx-nf" + q, payload: networkFunction, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ExecuteRequest", method: http.MethodPost, path: providers + "networkFunctions/nginx-nf/executeRequest" + q, payload: map[string]any{"serviceEndpoint": "nginx.edge.svc", "requestMetadata": map[string]any{"relativePath": "/reload", "httpMethod": "Post", "serializedBody": "{}", "apiVersion": "v1"}}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "ListNetworkFunctions", method: http.MethodGet, path: base + "/providers/Microsoft.HybridNetwork/networkFunctions" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteNetworkFunction", method: http.MethodDelete, path: providers + "networkFunctions/nginx-nf" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteSiteNetworkService", method: http.MethodDelete, path: providers + "siteNetworkServices/edge-service" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteConfigurationGroupValue", method: http.MethodDelete, path: providers + "configurationGroupValues/edge-values" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteSite", method: http.MethodDelete, path: providers + "sites/edge-site" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeletePublisher", method: http.MethodDelete, path: publisher + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if len(resp) != 0 {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
