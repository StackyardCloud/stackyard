package server

import (
	"net/http"
	"testing"
)

const (
	azureSFClusterTestAPIVersion     = "?api-version=2023-11-01-preview"
	azureSFClusterTestNodeType       = `{"name":"primary","clientConnectionEndpointPort":19000,"httpGatewayEndpointPort":19080,"vmInstanceCount":5,"isPrimary":true}`
	azureSFClusterTestManagementSite = `"managementEndpoint":"https://apps.eastus.cloudapp.azure.com:19080"`
)

func TestAzureSFClusterLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sfc")
	cluster := sub + "/resourceGroups/rg-sfc/providers/Microsoft.ServiceFabric/clusters/apps"

	resp := do(http.MethodPut, cluster+azureSFClusterTestAPIVersion, `{"location":"eastus","properties":{`+azureSFClusterTestManagementSite+`,"nodeTypes":[`+azureSFClusterTestNodeType+`],"reliabilityLevel":"Bronze","clusterState":"Deploying"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	clusterID, _ := props["clusterId"].(string)
	if props["clusterState"] != "Ready" || props["reliabilityLevel"] != "Bronze" || props["upgradeMode"] != "Automatic" ||
		props["clusterCodeVersion"] != azureSFClusterCodeVersion || props["clusterEndpoint"] != "https://eastus.servicefabric.azure.com/runtime/clusters/"+clusterID {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}
	resp = do(http.MethodPost, cluster+"/listUpgradableVersions"+azureSFClusterTestAPIVersion, `{"targetVersion":"`+azureSFClusterCodeVersion+`"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	appType := cluster + "/applicationTypes/VotingType"
	azureResourcesExpectStatus(t, do(http.MethodPut, appType+azureSFClusterTestAPIVersion, `{"properties":{}}`), http.StatusCreated)
	resp = do(http.MethodPut, appType+"/versions/1.0.0"+azureSFClusterTestAPIVersion, `{"properties":{"appPackageUrl":"https://example.blob.core.windows.net/apps/Voting.sfpkg"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["defaultParameterList"]; got == nil {
		t.Fatal("expected defaultParameterList")
	}
	app := cluster + "/applications/voting"
	azureResourcesExpectStatus(t, do(http.MethodPut, app+azureSFClusterTestAPIVersion, `{"properties":{"typeName":"VotingType","typeVersion":"1.0.0","parameters":{"Web_InstanceCount":"-1"}}}`), http.StatusCreated)
	service := app + "/services/web"
	azureResourcesExpectStatus(t, do(http.MethodPut, service+azureSFClusterTestAPIVersion, `{"properties":{"serviceKind":"Stateless","serviceTypeName":"WebType","instanceCount":-1,"partitionDescription":{"partitionScheme":"Singleton"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, app+"/services/data"+azureSFClusterTestAPIVersion, `{"properties":{"serviceKind":"Stateful","serviceTypeName":"DataType","targetReplicaSetSize":3,"minReplicaSetSize":2,"partitionDescription":{"partitionScheme":"UniformInt64Range","count":2,"lowKey":"0","highKey":"9"}}}`), http.StatusCreated)
	resp = do(http.MethodGet, app+"/services"+azureSFClusterTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 2 {
		t.Fatalf("expected two services, got %d", len(got))
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureSFClusterTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, service+azureSFClusterTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSFClusterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sfc")
	cluster := sub + "/resourceGroups/rg-sfc/providers/Microsoft.ServiceFabric/clusters/apps"
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+azureSFClusterTestAPIVersion, `{"location":"eastus","properties":{`+azureSFClusterTestManagementSite+`,"nodeTypes":[`+azureSFClusterTestNodeType+`]}}`), http.StatusCreated)
	app := cluster + "/applications/voting"
	azureResourcesExpectStatus(t, do(http.MethodPut, app+azureSFClusterTestAPIVersion, `{"properties":{"typeName":"VotingType","typeVersion":"1.0.0"}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"management endpoint", cluster, `{"location":"eastus","properties":{"managementEndpoint":"apps","nodeTypes":[` + azureSFClusterTestNodeType + `]}}`},
		{"no node types", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[]}}`},
		{"no primary", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[{"name":"nt","clientConnectionEndpointPort":19000,"httpGatewayEndpointPort":19080,"vmInstanceCount":3,"isPrimary":false}]}}`},
		{"bad port", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[{"name":"nt","clientConnectionEndpointPort":0,"httpGatewayEndpointPort":19080,"vmInstanceCount":3,"isPrimary":true}]}}`},
		{"bad count", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[{"name":"nt","clientConnectionEndpointPort":19000,"httpGatewayEndpointPort":19080,"vmInstanceCount":0,"isPrimary":true}]}}`},
		{"primary flag", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[{"name":"nt","clientConnectionEndpointPort":19000,"httpGatewayEndpointPort":19080,"vmInstanceCount":3}]}}`},
		{"string count", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[{"name":"nt","clientConnectionEndpointPort":19000,"httpGatewayEndpointPort":19080,"vmInstanceCount":"3","isPrimary":true}]}}`},
		{"upgrade mode", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[` + azureSFClusterTestNodeType + `],"upgradeMode":"Rolling"}}`},
		{"reliability", cluster, `{"location":"eastus","properties":{` + azureSFClusterTestManagementSite + `,"nodeTypes":[` + azureSFClusterTestNodeType + `],"reliabilityLevel":"Diamond"}}`},
		{"package url", cluster + "/applicationTypes/VotingType/versions/1.0.0", `{"properties":{"appPackageUrl":"Voting.sfpkg"}}`},
		{"application type", app, `{"properties":{"typeVersion":"1.0.0"}}`},
		{"service type", app + "/services/web", `{"properties":{"serviceKind":"Stateless","partitionDescription":{"partitionScheme":"Singleton"}}}`},
		{"service kind", app + "/services/web", `{"properties":{"serviceKind":"Batch","serviceTypeName":"WebType","partitionDescription":{"partitionScheme":"Singleton"}}}`},
		{"partition scheme", app + "/services/web", `{"properties":{"serviceKind":"Stateless","serviceTypeName":"WebType","partitionDescription":{"partitionScheme":"Hash"}}}`},
		{"instance count", app + "/services/web", `{"properties":{"serviceKind":"Stateless","serviceTypeName":"WebType","instanceCount":-2,"partitionDescription":{"partitionScheme":"Singleton"}}}`},
		{"replica size", app + "/services/data", `{"properties":{"serviceKind":"Stateful","serviceTypeName":"DataType","targetReplicaSetSize":0,"partitionDescription":{"partitionScheme":"Singleton"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSFClusterTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
