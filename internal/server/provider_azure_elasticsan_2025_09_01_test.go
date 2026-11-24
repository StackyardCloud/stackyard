package server

import (
	"net/http"
	"strings"
	"testing"
)

const esanTestAPI = "?api-version=" + azureElasticSan20250901APIVersion

func TestAzureElasticSanLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-san")
	san := sub + "/resourceGroups/rg-san/providers/Microsoft.ElasticSan/elasticSans/san1"
	group := san + "/volumegroups/vg1"
	vol := group + "/volumes/vol1"

	resp := do(http.MethodPut, san+esanTestAPI, `{"location":"eastus","properties":{"sku":{"name":"Premium_LRS"},"baseSizeTiB":1,"extendedCapacitySizeTiB":1}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["totalSizeTiB"] != float64(2) || props["totalIops"] != float64(5000) || props["publicNetworkAccess"] != "Enabled" {
		t.Fatalf("unexpected SAN properties: %#v", props)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, group+esanTestAPI, `{"properties":{"protocolType":"Iscsi"}}`), http.StatusCreated)
	resp = do(http.MethodPut, vol+esanTestAPI, `{"properties":{"sizeGiB":100}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	target := providerContractJSONMap(t, resp)["properties"].(map[string]any)["storageTarget"].(map[string]any)
	if !strings.HasPrefix(target["targetIqn"].(string), "iqn.") || target["targetPortalPort"] != float64(esanISCSIPort) {
		t.Fatalf("unexpected storage target: %#v", target)
	}
	// 2 TiB holds 2048 GiB; 100 are used.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/volumes/vol2"+esanTestAPI, `{"properties":{"sizeGiB":2000}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vol+esanTestAPI, `{"properties":{"sizeGiB":50}}`), http.StatusBadRequest, "InvalidRequestContent")

	snap := group + "/snapshots/snap1"
	resp = do(http.MethodPut, snap+esanTestAPI, `{"properties":{"creationData":{"sourceId":"`+strings.TrimPrefix(vol, "/azure")+`"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["sourceVolumeSizeGiB"]; got != float64(100) {
		t.Fatalf("sourceVolumeSizeGiB = %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/snapshots/bad"+esanTestAPI, `{"properties":{"creationData":{"sourceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.ElasticSan/elasticSans/z/volumegroups/a/volumes/missing"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodDelete, vol+esanTestAPI, ""), http.StatusConflict, "Conflict")

	resp = do(http.MethodPost, group+"/preBackup"+esanTestAPI, `{"volumeNames":["vol1"]}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["validationStatus"] != "Success" {
		t.Fatal("preBackup did not validate")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, group+"/preBackup"+esanTestAPI, `{"volumeNames":["nope"]}`), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, group+"/preRestore"+esanTestAPI, `{"diskSnapshotIds":["not-an-id"]}`), http.StatusBadRequest, "InvalidRequestContent")

	for _, path := range []string{san + "/privateLinkResources", sub + "/providers/Microsoft.ElasticSan/skus", "/azure/providers/Microsoft.ElasticSan/operations"} {
		resp = do(http.MethodGet, path+esanTestAPI, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if len(providerContractJSONMap(t, resp)["value"].([]any)) == 0 {
			t.Fatalf("%s listed nothing", path)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, snap+esanTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, san+esanTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, vol+esanTestAPI, ""), http.StatusNotFound)
}

func TestAzureElasticSanValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-san")
	san := sub + "/resourceGroups/rg-san/providers/Microsoft.ElasticSan/elasticSans/san1"
	for name, body := range map[string]string{
		"no sku":         `{"location":"eastus","properties":{"baseSizeTiB":1,"extendedCapacitySizeTiB":0}}`,
		"bad sku":        `{"location":"eastus","properties":{"sku":{"name":"Standard_LRS"},"baseSizeTiB":1,"extendedCapacitySizeTiB":0}}`,
		"base too large": `{"location":"eastus","properties":{"sku":{"name":"Premium_LRS"},"baseSizeTiB":101,"extendedCapacitySizeTiB":0}}`,
		"fractional":     `{"location":"eastus","properties":{"sku":{"name":"Premium_LRS"},"baseSizeTiB":1.5,"extendedCapacitySizeTiB":0}}`,
	} {
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, do(http.MethodPut, san+esanTestAPI, body), http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, san+"/volumegroups/vg1"+esanTestAPI, `{}`), http.StatusNotFound, "ParentResourceNotFound")
}
