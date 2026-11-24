package server

import (
	"net/http"
	"net/url"
	"testing"
)

const azureEASMDTestBase = "/azure/defenderforeasm-dataplanepreview/2024-10-01-preview"

func azureEASMDTestPath(path string, query ...string) string {
	q := url.Values{"api-version": {"2024-10-01-preview"}}
	for i := 0; i+1 < len(query); i += 2 {
		q.Set(query[i], query[i+1])
	}
	return azureEASMDTestBase + path + "?" + q.Encode()
}

func TestAzureDefenderForEasmDataPlanePreview20241001Assets(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-easmd")

	page := providerContractJSONMap(t, do(http.MethodGet, azureEASMDTestPath("/assets", "filter", `state = "confirmed" AND kind = "host"`), ""))
	if page["totalElements"].(float64) != 2 {
		t.Fatalf("expected 2 confirmed hosts, got %v", page["totalElements"])
	}
	// aG9zdCQkY29udG9zby5jb20 is base64("host$$contoso.com").
	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestPath("/assets/aG9zdCQkY29udG9zby5jb20"), ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestPath("/assets/bm9wZQ"), ""), http.StatusNotFound)

	task := providerContractJSONMap(t, do(http.MethodPost, azureEASMDTestPath("/assets", "filter", `name ^= "www."`), `{"state":"dismissed","labels":{"triage":true}}`))
	if task["state"] != "complete" {
		t.Fatalf("update task: %#v", task)
	}
	asset := providerContractJSONMap(t, do(http.MethodGet, azureEASMDTestPath("/assets/aG9zdCQkd3d3LmNvbnRvc28uY29t"), ""))
	if asset["state"] != "dismissed" || len(asset["labels"].([]any)) != 1 {
		t.Fatalf("asset not updated: %#v", asset)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestPath("/tasks/"+task["id"].(string)), ""), http.StatusOK)

	for _, tc := range []struct{ name, method, path, body string }{
		{"unsupported filter field", http.MethodGet, azureEASMDTestPath("/assets", "filter", `color = "red"`), ""},
		{"update without filter", http.MethodPost, azureEASMDTestPath("/assets"), `{"state":"confirmed"}`},
		{"bad state", http.MethodPost, azureEASMDTestPath("/assets", "filter", `kind = "host"`), `{"state":"exploded"}`},
		{"export without columns", http.MethodPost, azureEASMDTestPath("/assets:export"), `{"fileName":"x.csv"}`},
		{"bad priorDays", http.MethodPost, azureEASMDTestPath("/assets:getDeltaSummary"), `{"priorDays":90}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectStatus(t, do(tc.method, tc.path, tc.body), http.StatusBadRequest)
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestBase+"/assets?api-version=2020-01-01", ""), http.StatusBadRequest)
}

func TestAzureDefenderForEasmDataPlanePreview20241001Collections(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-easmd")

	conn := `{"kind":"logAnalytics","properties":{"apiKey":"k","workspaceId":"w"},"frequency":"weekly","frequencyOffset":1}`
	resp := do(http.MethodPut, azureEASMDTestPath("/dataConnections/dc1"), conn)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if _, leaked := providerContractJSONMap(t, resp)["properties"].(map[string]any)["apiKey"]; leaked {
		t.Fatal("apiKey must not be returned")
	}
	if body := providerContractJSONMap(t, do(http.MethodPost, azureEASMDTestPath("/dataConnections:validate"), `{"kind":"other"}`)); body["error"] == nil {
		t.Fatal("validate should report an invalid connection")
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, azureEASMDTestPath("/dataConnections/dc1"), ""), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodDelete, azureEASMDTestPath("/dataConnections/dc1"), ""), http.StatusNotFound)

	group := `{"description":"d","tier":"advanced","seeds":[{"kind":"host","name":"a.contoso.com"}]}`
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/discoGroups/g1"), group), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/discoGroups/g2"), `{"seeds":[{"kind":"planet","name":"x"}]}`), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodPost, azureEASMDTestPath("/discoGroups/g1:run"), ""), http.StatusNoContent)
	if n := providerContractJSONMap(t, do(http.MethodGet, azureEASMDTestPath("/discoGroups/g1/runs"), ""))["totalElements"]; n != float64(1) {
		t.Fatalf("expected 1 run, got %v", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, azureEASMDTestPath("/discoGroups/nope:run"), ""), http.StatusNotFound)

	// Policies need an existing saved filter.
	policy := `{"filterName":"f1","action":"setState","actionParameters":{"value":"confirmed"}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/policies/p1"), policy), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/savedFilters/f1"), `{"filter":"state IN (\"confirmed\")","description":"d"}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/policies/p1"), policy), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestPath("/cisaCves/CVE-2023-12345"), ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, azureEASMDTestPath("/discoTemplates/12345"), ""), http.StatusOK)
	summary := providerContractJSONMap(t, do(http.MethodPost, azureEASMDTestPath("/reports/assets:getSummary"), `{"filters":["kind = \"domain\""]}`))
	if count := summary["assetSummaries"].([]any)[0].(map[string]any)["count"]; count != float64(1) {
		t.Fatalf("expected 1 domain, got %v", count)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, azureEASMDTestPath("/reports/assets:getSnapshot"), `{}`), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodPut, azureEASMDTestPath("/nothing/x"), `{}`), http.StatusNotFound)
}
