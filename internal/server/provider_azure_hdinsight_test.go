package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const azureHDInsightTestAPIVersion = "?api-version=2021-06-01"

const azureHDInsightTestCluster = `{"location":"eastus","properties":{"clusterVersion":"5.1","osType":"Linux","tier":"Standard",
"clusterDefinition":{"kind":"spark","configurations":{"gateway":{"restAuthCredential.isEnabled":true,"restAuthCredential.username":"admin","restAuthCredential.password":"Secret-123"}}},
"computeProfile":{"roles":[
 {"name":"headnode","targetInstanceCount":2,"hardwareProfile":{"vmSize":"Standard_E8_V3"},"osProfile":{"linuxOperatingSystemProfile":{"username":"sshuser","password":"Secret-123"}}},
 {"name":"workernode","targetInstanceCount":3,"hardwareProfile":{"vmSize":"Standard_E8_V3"},"osProfile":{"linuxOperatingSystemProfile":{"username":"sshuser","password":"Secret-123"}}}]},
"securityProfile":{"directoryType":"ActiveDirectory","domain":"corp.example","domainUsername":"hdiadmin","domainUserPassword":"Secret-123"},
"storageProfile":{"storageaccounts":[{"name":"lake.blob.core.windows.net","isDefault":true,"container":"spark","key":"Secret-123"}]}}}`

func TestAzureHDInsightLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-hdi")
	cluster := sub + "/resourceGroups/rg-hdi/providers/Microsoft.HDInsight/clusters/Spark-Prod"

	resp := do(http.MethodPut, cluster+azureHDInsightTestAPIVersion, azureHDInsightTestCluster)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	if raw, _ := json.Marshal(body); strings.Contains(string(raw), "Secret-123") {
		t.Fatalf("a password was returned: %s", raw)
	}
	props := body["properties"].(map[string]any)
	endpoints, _ := props["connectivityEndpoints"].([]any)
	if props["clusterState"] != "Running" || len(endpoints) != 2 || endpoints[0].(map[string]any)["location"] != "spark-prod.azurehdinsight.net" {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}
	roles := props["computeProfile"].(map[string]any)["roles"].([]any)
	if user := roles[0].(map[string]any)["osProfile"].(map[string]any)["linuxOperatingSystemProfile"].(map[string]any)["username"]; user != "sshuser" {
		t.Fatalf("expected the SSH user kept, got %#v", roles[0])
	}
	if definition := props["clusterDefinition"].(map[string]any); definition["configurations"] != nil || definition["kind"] != "spark" {
		t.Fatalf("unexpected cluster definition: %#v", definition)
	}

	resp = do(http.MethodGet, sub+"/providers/Microsoft.HDInsight/clusters"+azureHDInsightTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one cluster, got %#v", items)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureHDInsightTestAPIVersion, ""), http.StatusOK)
}

func TestAzureHDInsightValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-hdi")
	clusters := sub + "/resourceGroups/rg-hdi/providers/Microsoft.HDInsight/clusters/"
	replace := func(old, new string) string { return strings.Replace(azureHDInsightTestCluster, old, new, 1) }
	cases := []struct{ name, path, body, code string }{
		{"leading digit", clusters + "1spark", azureHDInsightTestCluster, "InvalidResourceName"},
		{"bad kind", clusters + "spark", replace(`"kind":"spark"`, `"kind":"storm"`), "InvalidRequestContent"},
		{"bad tier", clusters + "spark", replace(`"tier":"Standard"`, `"tier":"Gold"`), "InvalidRequestContent"},
		{"windows", clusters + "spark", replace(`"osType":"Linux"`, `"osType":"Windows"`), "InvalidRequestContent"},
		{"unknown role", clusters + "spark", replace(`"name":"headnode"`, `"name":"masternode"`), "InvalidRequestContent"},
		{"no worker", clusters + "spark", replace(`"name":"workernode"`, `"name":"edgenode"`), "InvalidRequestContent"},
		{"zero instances", clusters + "spark", replace(`"targetInstanceCount":3`, `"targetInstanceCount":0`), "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureHDInsightTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.HDInsight/locations/eastus/checkNameAvailability"+azureHDInsightTestAPIVersion, `{"name":"`+name+`","type":"clusters"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("spark"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, clusters+"spark"+azureHDInsightTestAPIVersion, azureHDInsightTestCluster), http.StatusCreated)
	if got := check("SPARK"); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("-spark"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
}
