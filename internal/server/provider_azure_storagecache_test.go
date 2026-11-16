package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureStorageCacheTestAPIVersion = "?api-version=2024-03-01"

func azureStorageCacheTestIDs(sub string) (subnet, container string) {
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-hpc/providers"
	return ids + "/Microsoft.Network/virtualNetworks/hpc/subnets/cache", ids + "/Microsoft.Storage/storageAccounts/hpcdata/blobServices/default/containers/data"
}

func TestAzureStorageCacheLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-hpc")
	subnet, container := azureStorageCacheTestIDs(sub)
	cache := sub + "/resourceGroups/rg-hpc/providers/Microsoft.StorageCache/caches/render"

	resp := do(http.MethodPut, cache+azureStorageCacheTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":3072,"subnet":"`+subnet+`","health":{"state":"Degraded"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["health"].(map[string]any)["state"] != "Healthy" || len(props["mountAddresses"].([]any)) != 3 {
		t.Fatalf("unexpected cache properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, cache+"/stop"+azureStorageCacheTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, cache+azureStorageCacheTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["health"].(map[string]any)["state"]; got != "Stopped" {
		t.Fatalf("expected a stopped cache, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, cache+"/start"+azureStorageCacheTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, cache+"/flush"+azureStorageCacheTestAPIVersion, ""), http.StatusAccepted)

	target := cache + "/storageTargets/assets"
	azureResourcesExpectStatus(t, do(http.MethodPut, target+azureStorageCacheTestAPIVersion, `{"properties":{"targetType":"blobNfs","blobNfs":{"target":"`+container+`","usageModel":"WRITE_AROUND"},"junctions":[{"namespacePath":"/assets","targetPath":"/"}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, target+"/suspend"+azureStorageCacheTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, target+azureStorageCacheTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["state"]; got != "Suspended" {
		t.Fatalf("expected a suspended target, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, cache+"/storageTargets/nas"+azureStorageCacheTestAPIVersion, `{"properties":{"targetType":"nfs3","nfs3":{"target":"10.0.0.20","usageModel":"READ_HEAVY_INFREQ"},"junctions":[{"namespacePath":"/nas","nfsExport":"/export"}]}}`), http.StatusCreated)

	fs := sub + "/resourceGroups/rg-hpc/providers/Microsoft.StorageCache/amlFilesystems/scratch"
	resp = do(http.MethodPut, fs+azureStorageCacheTestAPIVersion, `{"location":"eastus","zones":["1"],"sku":{"name":"AMLFS-Durable-Premium-250"},"properties":{"storageCapacityTiB":16,"filesystemSubnet":"`+subnet+`","maintenanceWindow":{"dayOfWeek":"Friday","timeOfDayUTC":"22:00"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	client := props["clientInfo"].(map[string]any)
	if props["throughputProvisionedMBps"] != float64(4000) || !strings.Contains(client["mountCommand"].(string), client["mgsAddress"].(string)+"@tcp:/lustrefs") {
		t.Fatalf("unexpected file system properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, fs+"/archive"+azureStorageCacheTestAPIVersion, `{"filesystemPath":"/"}`), http.StatusOK)

	resp = do(http.MethodPost, sub+"/providers/Microsoft.StorageCache/getRequiredAmlFSSubnetsSize"+azureStorageCacheTestAPIVersion, `{"sku":{"name":"AMLFS-Durable-Premium-250"},"storageCapacityTiB":16}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["filesystemSubnetSize"]; got != float64(18) {
		t.Fatalf("unexpected subnet size: %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.StorageCache/checkAmlFSSubnets"+azureStorageCacheTestAPIVersion, `{"filesystemSubnet":"`+subnet+`","sku":{"name":"AMLFS-Durable-Premium-250"},"storageCapacityTiB":16}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.StorageCache/checkAmlFSSubnets"+azureStorageCacheTestAPIVersion, `{"filesystemSubnet":"cache","sku":{"name":"AMLFS-Durable-Premium-250"},"storageCapacityTiB":16}`), http.StatusBadRequest, "InvalidParameter")
}

func TestAzureStorageCacheValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-hpc")
	subnet, container := azureStorageCacheTestIDs(sub)
	group := sub + "/resourceGroups/rg-hpc/providers/Microsoft.StorageCache"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/caches/c"+azureStorageCacheTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":3072,"subnet":"`+subnet+`"}}`), http.StatusCreated)
	window := `"maintenanceWindow":{"dayOfWeek":"Friday","timeOfDayUTC":"22:00"}`
	cases := []struct{ name, path, body string }{
		{"cache sku", "/caches/c", `{"location":"eastus","sku":{"name":"Standard_1G"},"properties":{"cacheSizeGB":3072,"subnet":"` + subnet + `"}}`},
		{"cache size", "/caches/c", `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":4096,"subnet":"` + subnet + `"}}`},
		{"cache subnet", "/caches/c", `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":3072,"subnet":"cache"}}`},
		{"cache mtu", "/caches/c", `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":3072,"subnet":"` + subnet + `","networkSettings":{"mtu":9000}}}`},
		{"target type", "/caches/c/storageTargets/t", `{"properties":{"targetType":"s3"}}`},
		{"target nfs host", "/caches/c/storageTargets/t", `{"properties":{"targetType":"nfs3","nfs3":{}}}`},
		{"target container", "/caches/c/storageTargets/t", `{"properties":{"targetType":"clfs","clfs":{"target":"data"}}}`},
		{"target junction", "/caches/c/storageTargets/t", `{"properties":{"targetType":"blobNfs","blobNfs":{"target":"` + container + `"},"junctions":[{"namespacePath":"assets"}]}}`},
		{"target duplicate junction", "/caches/c/storageTargets/t", `{"properties":{"targetType":"blobNfs","blobNfs":{"target":"` + container + `"},"junctions":[{"namespacePath":"/a"},{"namespacePath":"/a"}]}}`},
		{"target nfs export", "/caches/c/storageTargets/t", `{"properties":{"targetType":"nfs3","nfs3":{"target":"10.0.0.20"},"junctions":[{"namespacePath":"/nas"}]}}`},
		{"lustre 40 step", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-40"},"properties":{"storageCapacityTiB":16,"filesystemSubnet":"` + subnet + `",` + window + `}}`},
		{"lustre too large", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-500"},"properties":{"storageCapacityTiB":1e12,"filesystemSubnet":"` + subnet + `",` + window + `}}`},
		{"cache size type", "/caches/c", `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":"3072","subnet":"` + subnet + `"}}`},
		{"lustre sku", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Premium"},"properties":{"storageCapacityTiB":16,"filesystemSubnet":"` + subnet + `",` + window + `}}`},
		{"lustre capacity", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-250"},"properties":{"storageCapacityTiB":10,"filesystemSubnet":"` + subnet + `",` + window + `}}`},
		{"lustre subnet", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-250"},"properties":{"storageCapacityTiB":8,"filesystemSubnet":"x",` + window + `}}`},
		{"lustre window", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-250"},"properties":{"storageCapacityTiB":8,"filesystemSubnet":"` + subnet + `","maintenanceWindow":{"dayOfWeek":"Friday","timeOfDayUTC":"25:00"}}}`},
		{"lustre hsm", "/amlFilesystems/fs", `{"location":"eastus","sku":{"name":"AMLFS-Durable-Premium-250"},"properties":{"storageCapacityTiB":8,"filesystemSubnet":"` + subnet + `",` + window + `,"hsm":{"settings":{"container":"` + container + `","loggingContainer":"logs"}}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, group+tc.path+azureStorageCacheTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/caches/c/storageTargets/a.b"+azureStorageCacheTestAPIVersion, `{"properties":{"targetType":"unknown"}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/caches/render.cache"+azureStorageCacheTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard_2G"},"properties":{"cacheSizeGB":3072,"subnet":"`+subnet+`"}}`), http.StatusBadRequest, "InvalidResourceName")
}
