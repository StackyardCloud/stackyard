package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureStorageSyncTestAPIVersion = "?api-version=2022-09-01"
	azureStorageSyncTestServerID   = "3a048283-338f-4002-a9dd-a50fdadcb392"
)

func TestAzureStorageSyncLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sync")
	service := sub + "/resourceGroups/rg-sync/providers/Microsoft.StorageSync/storageSyncServices/files"
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-sync/providers"

	resp := do(http.MethodPut, service+azureStorageSyncTestAPIVersion, `{"location":"eastus","properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	uid := props["storageSyncServiceUid"]
	if props["incomingTrafficPolicy"] != "AllowAllTraffic" || uid == nil || props["useIdentity"] != false {
		t.Fatalf("unexpected sync service properties: %#v", props)
	}
	resp = do(http.MethodPost, sub+"/providers/Microsoft.StorageSync/locations/eastus/checkNameAvailability"+azureStorageSyncTestAPIVersion, `{"name":"files","type":"Microsoft.StorageSync/storageSyncServices"}`)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected the name taken: %#v", got)
	}

	server := service + "/registeredServers/" + azureStorageSyncTestServerID
	resp = do(http.MethodPut, server+azureStorageSyncTestAPIVersion, `{"properties":{"serverCertificate":"MIIDdzCCAl+gAwIBAgIQ","agentVersion":"18.0.0.0","serverOSVersion":"10.0.20348.0","friendlyName":"fs01.contoso.com"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["serverId"] != azureStorageSyncTestServerID || props["storageSyncServiceUid"] != uid || props["friendlyName"] != "fs01.contoso.com" {
		t.Fatalf("unexpected registered server properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, server+"/triggerRollover"+azureStorageSyncTestAPIVersion, `{"serverCertificate":"MIID"}`), http.StatusOK)

	group := service + "/syncGroups/shares"
	resp = do(http.MethodPut, group+azureStorageSyncTestAPIVersion, `{"properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["uniqueId"]; got == nil {
		t.Fatal("expected a sync group uniqueId")
	}
	cloud := group + "/cloudEndpoints/share1"
	resp = do(http.MethodPut, cloud+azureStorageSyncTestAPIVersion, `{"properties":{"storageAccountResourceId":"`+ids+`/Microsoft.Storage/storageAccounts/files","azureFileShareName":"share1"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["friendlyName"] != "share1" || props["storageAccountTenantId"] != azureDefaultTenantID {
		t.Fatalf("unexpected cloud endpoint properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, cloud+"/prebackup"+azureStorageSyncTestAPIVersion, `{"azureFileShare":"https://files.file.core.windows.net/share1"}`), http.StatusAccepted)
	resp = do(http.MethodPost, cloud+"/postbackup"+azureStorageSyncTestAPIVersion, `{"azureFileShare":"https://files.file.core.windows.net/share1"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, cloud+"/triggerChangeDetection"+azureStorageSyncTestAPIVersion, `{"directoryPath":"/","changeDetectionMode":"Recursive"}`), http.StatusAccepted)

	endpoint := group + "/serverEndpoints/fs01"
	resp = do(http.MethodPut, endpoint+azureStorageSyncTestAPIVersion, `{"properties":{"serverLocalPath":"D:\\shares\\share1","serverResourceId":"`+strings.TrimPrefix(server, "/azure")+`","cloudTiering":"on","volumeFreeSpacePercent":30}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["friendlyName"] != azureStorageSyncTestServerID || props["volumeFreeSpacePercent"] != float64(30) || props["initialDownloadPolicy"] != "NamespaceThenModifiedFiles" {
		t.Fatalf("unexpected server endpoint properties: %#v", props)
	}
	resp = do(http.MethodPut, group+"/cloudEndpoints/hr"+azureStorageSyncTestAPIVersion, `{"properties":{"storageAccountResourceId":"`+ids+`/Microsoft.Storage/storageAccounts/files","azureFileShareName":"hr","friendlyName":"HR share"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["friendlyName"]; got != "HR share" {
		t.Fatalf("expected the client friendlyName kept, got %v", got)
	}
	for name, want := range map[string]any{"new-files": true, "bad.": false} {
		resp = do(http.MethodPost, sub+"/providers/Microsoft.StorageSync/locations/eastus/checkNameAvailability"+azureStorageSyncTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.StorageSync/storageSyncServices"}`)
		if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != want {
			t.Fatalf("name %s: expected nameAvailable %v, got %v", name, want, got)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, endpoint+"/recallAction"+azureStorageSyncTestAPIVersion, `{"pattern":"","recallPath":""}`), http.StatusAccepted)

	azureResourcesExpectStatus(t, do(http.MethodDelete, service+azureStorageSyncTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, endpoint+azureStorageSyncTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureStorageSyncValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sync")
	service := sub + "/resourceGroups/rg-sync/providers/Microsoft.StorageSync/storageSyncServices/files"
	azureResourcesExpectStatus(t, do(http.MethodPut, service+azureStorageSyncTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, service+"/syncGroups/g"+azureStorageSyncTestAPIVersion, `{"properties":{}}`), http.StatusCreated)
	serverID := strings.TrimPrefix(service, "/azure") + "/registeredServers/" + azureStorageSyncTestServerID
	cases := []struct{ name, path, body string }{
		{"traffic policy", service, `{"location":"eastus","properties":{"incomingTrafficPolicy":"Deny"}}`},
		{"server certificate", service + "/registeredServers/" + azureStorageSyncTestServerID, `{"properties":{"agentVersion":"18.0.0.0"}}`},
		{"server agent", service + "/registeredServers/" + azureStorageSyncTestServerID, `{"properties":{"serverCertificate":"MIID"}}`},
		{"cloud account", service + "/syncGroups/g/cloudEndpoints/c", `{"properties":{"storageAccountResourceId":"files","azureFileShareName":"share1"}}`},
		{"cloud share", service + "/syncGroups/g/cloudEndpoints/c", `{"properties":{"storageAccountResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/files"}}`},
		{"cloud tenant", service + "/syncGroups/g/cloudEndpoints/c", `{"properties":{"storageAccountResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/files","azureFileShareName":"share1","storageAccountTenantId":"contoso"}}`},
		{"server path", service + "/syncGroups/g/serverEndpoints/s", `{"properties":{"serverResourceId":"` + serverID + `"}}`},
		{"server id", service + "/syncGroups/g/serverEndpoints/s", `{"properties":{"serverLocalPath":"D:\\share","serverResourceId":"fs01"}}`},
		{"tiering", service + "/syncGroups/g/serverEndpoints/s", `{"properties":{"serverLocalPath":"D:\\share","serverResourceId":"` + serverID + `","cloudTiering":"auto"}}`},
		{"free space", service + "/syncGroups/g/serverEndpoints/s", `{"properties":{"serverLocalPath":"D:\\share","serverResourceId":"` + serverID + `","volumeFreeSpacePercent":120}}`},
		{"download policy", service + "/syncGroups/g/serverEndpoints/s", `{"properties":{"serverLocalPath":"D:\\share","serverResourceId":"` + serverID + `","initialDownloadPolicy":"Everything"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureStorageSyncTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, service+"/registeredServers/fs01"+azureStorageSyncTestAPIVersion, `{"properties":{"serverCertificate":"MIID","agentVersion":"18.0.0.0"}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-sync/providers/Microsoft.StorageSync/storageSyncServices/files."+azureStorageSyncTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
}
