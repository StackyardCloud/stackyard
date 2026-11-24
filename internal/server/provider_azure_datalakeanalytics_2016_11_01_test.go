package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureDLATestVersion = "?api-version=" + azureDataLakeAnalytics20161101APIVersion

const azureDLATestAccount = `{"location":"eastus2","properties":{"defaultDataLakeStoreAccount":"lake1","dataLakeStoreAccounts":[{"name":"lake1"}],
"storageAccounts":[{"name":"blob1","properties":{"accessKey":"c2VjcmV0"}}]}}`

func TestAzureDataLakeAnalytics20161101Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dla")
	account := sub + "/resourceGroups/rg-dla/providers/Microsoft.DataLakeAnalytics/accounts/adla1"

	resp := do(http.MethodPut, account+azureDLATestVersion, azureDLATestAccount)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["endpoint"] != "adla1.azuredatalakeanalytics.net" || props["maxJobCount"] != float64(3) {
		t.Fatalf("unexpected account properties: %#v", props)
	}

	resp = do(http.MethodGet, account+"/storageAccounts/blob1"+azureDLATestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if sa := providerContractJSONMap(t, resp)["properties"].(map[string]any); sa["accessKey"] != nil || sa["suffix"] != "blob.core.windows.net" {
		t.Fatalf("storage account leaks its key or lacks a suffix: %#v", sa)
	}
	resp = do(http.MethodGet, account+"/storageAccounts/blob1/containers"+azureDLATestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/computePolicies/limits"+azureDLATestVersion,
		`{"properties":{"objectId":"00000000-0000-0000-0000-000000000001","objectType":"User","maxDegreeOfParallelismPerJob":5}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/firewallRules/office"+azureDLATestVersion,
		`{"properties":{"startIpAddress":"10.0.0.1","endIpAddress":"10.0.0.2"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/dataLakeStoreAccounts/lake2"+azureDLATestVersion, `{}`), http.StatusOK)
	resp = do(http.MethodGet, account+"/dataLakeStoreAccounts"+azureDLATestVersion, "")
	if stores := providerContractJSONMap(t, resp)["value"].([]any); len(stores) != 2 {
		t.Fatalf("want 2 Data Lake Store accounts, got %d", len(stores))
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.DataLakeAnalytics/locations/eastus2/capability"+azureDLATestVersion, "")
	if got := providerContractJSONMap(t, resp)["accountCount"]; got != float64(1) {
		t.Fatalf("capability accountCount = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, account+"/computePolicies/limits"+azureDLATestVersion, ""), http.StatusNotFound)
}

func TestAzureDataLakeAnalytics20161101Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dla")
	account := sub + "/resourceGroups/rg-dla/providers/Microsoft.DataLakeAnalytics/accounts/adla2"
	cases := []struct{ name, path, body string }{
		{"no default store", account, `{"location":"eastus","properties":{"dataLakeStoreAccounts":[{"name":"lake1"}]}}`},
		{"default not listed", account, `{"location":"eastus","properties":{"defaultDataLakeStoreAccount":"other","dataLakeStoreAccounts":[{"name":"lake1"}]}}`},
		{"storage without key", account, `{"location":"eastus","properties":{"defaultDataLakeStoreAccount":"lake1","dataLakeStoreAccounts":[{"name":"lake1"}],"storageAccounts":[{"name":"s"}]}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureDLATestVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, account+azureDLATestVersion, azureDLATestAccount), http.StatusCreated)
	resp := do(http.MethodPut, account+"/computePolicies/p"+azureDLATestVersion, `{"properties":{"objectId":"x","objectType":"User"}}`)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureDataLakeAnalytics20161101DataPlane(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-dla")
	const jobID = "6a2a2f2b-3b1f-4d5e-9f00-0000000000aa"
	job := `{"name":"count","type":"USql","properties":{"type":"USql","script":"@a = SELECT * FROM (VALUES (1)) AS T(x); OUTPUT @a TO \"/out.csv\" USING Outputters.Csv();"}}`

	resp := do(http.MethodPut, "/azure/Jobs/"+jobID+azureDLATestVersion, job)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if body := providerContractJSONMap(t, resp); body["state"] != "Ended" || body["result"] != "Succeeded" || body["jobId"] != jobID {
		t.Fatalf("unexpected job: %#v", body)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/Jobs/"+jobID+azureDLATestVersion, job), http.StatusConflict, "JobAlreadyExists")
	resp = do(http.MethodGet, "/azure/Jobs"+azureDLATestVersion, "")
	if jobs := providerContractJSONMap(t, resp)["value"].([]any); len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/Jobs/"+jobID+"/GetStatistics"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/Jobs/"+jobID+"/GetDebugDataPath"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/Jobs/"+jobID+"/CancelJob"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/BuildJob"+azureDLATestVersion, job), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/Jobs/not-a-guid"+azureDLATestVersion, job), http.StatusBadRequest, "InvalidJobId")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/Jobs/6a2a2f2b-3b1f-4d5e-9f00-0000000000ab"+azureDLATestVersion, `{"name":"x","type":"USql","properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/Jobs/6a2a2f2b-3b1f-4d5e-9f00-0000000000ac"+azureDLATestVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/pipelines"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/recurrences/r1"+azureDLATestVersion, ""), http.StatusNotFound)

	resp = do(http.MethodGet, "/azure/catalog/usql/databases"+azureDLATestVersion, "")
	if dbs := providerContractJSONMap(t, resp)["value"].([]any); len(dbs) != 1 || !strings.EqualFold(dbs[0].(map[string]any)["databaseName"].(string), "master") {
		t.Fatalf("unexpected databases: %#v", dbs)
	}
	cred := "/azure/catalog/usql/databases/master/credentials/sqlcred"
	azureResourcesExpectStatus(t, do(http.MethodPut, cred+azureDLATestVersion, `{"password":"p","uri":"https://sql.example:1433","userId":"u"}`), http.StatusOK)
	resp = do(http.MethodGet, cred+azureDLATestVersion, "")
	if body := providerContractJSONMap(t, resp); body["credentialName"] != "sqlcred" || body["password"] != nil {
		t.Fatalf("unexpected credential: %#v", body)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, cred+azureDLATestVersion, `{"password":"p","uri":"https://sql.example:1433","userId":"u"}`), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodDelete, cred+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, cred+azureDLATestVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/catalog/usql/databases/master/schemas/dbo"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/catalog/usql/databases/master/schemas/dbo/tables"+azureDLATestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/catalog/usql/databases/sales"+azureDLATestVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/catalog/usql/acl"+azureDLATestVersion+"&op=GrantAce", `{"aceType":"User","principalId":"00000000-0000-0000-0000-000000000001","permission":"Use"}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/catalog/usql/acl"+azureDLATestVersion+"&op=Nope", `{}`), http.StatusBadRequest)
}
