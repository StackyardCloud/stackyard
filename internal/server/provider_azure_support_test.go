package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const (
	azureSupportTestAPIVersion = "?api-version=2024-04-01"
	azureSupportTestService    = "/providers/Microsoft.Support/services/06bfd9d3-516b-d5c6-5802-169c800dec89"
)

func azureSupportTestTicket(problem string) string {
	return `{"properties":{"title":"Raise vCPU quota","description":"Need 200 DSv5 cores in eastus.","serviceId":"` + azureSupportTestService + `","problemClassificationId":"` + problem + `","severity":"moderate","advancedDiagnosticConsent":"Yes","contactDetails":{"firstName":"Ada","lastName":"Lovelace","preferredContactMethod":"email","primaryEmailAddress":"ada@contoso.com","preferredTimeZone":"Pacific Standard Time","country":"usa","preferredSupportLanguage":"en-US"}}}`
}

func TestAzureSupportTicketLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-support")

	resp := do(http.MethodGet, "/azure"+azureSupportTestService+"/problemClassifications"+azureSupportTestAPIVersion, "")
	problems := providerContractJSONMap(t, resp)["value"].([]any)
	if len(problems) == 0 {
		t.Fatal("expected problem classifications")
	}
	problem := problems[0].(map[string]any)["id"].(string)

	ticket := sub + "/providers/Microsoft.Support/supportTickets/quota-request"
	resp = do(http.MethodPut, ticket+azureSupportTestAPIVersion, azureSupportTestTicket(problem))
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["status"] != "Open" || len(props["supportTicketId"].(string)) != 16 || props["serviceDisplayName"] != "Service and subscription limits (quotas)" || props["serviceLevelAgreement"].(map[string]any)["slaMinutes"] != float64(240) {
		t.Fatalf("unexpected ticket properties: %#v", props)
	}
	resp = do(http.MethodPost, sub+"/providers/Microsoft.Support/checkNameAvailability"+azureSupportTestAPIVersion, `{"name":"quota-request","type":"Microsoft.Support/supportTickets"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("expected the ticket name taken, got %v", got)
	}

	msg := ticket + "/communications/followup"
	resp = do(http.MethodPut, msg+azureSupportTestAPIVersion, `{"properties":{"subject":"More detail","body":"We also need 100 FSv2 cores.","sender":"ada@contoso.com"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["communicationDirection"]; got != "inbound" {
		t.Fatalf("unexpected direction: %v", got)
	}
	resp = do(http.MethodPost, ticket+"/checkNameAvailability"+azureSupportTestAPIVersion, `{"name":"followup","type":"Microsoft.Support/communications"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("expected the communication name taken, got %v", got)
	}
	resp = do(http.MethodGet, ticket+"/communications"+azureSupportTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one communication, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, ticket+"/chatTranscripts"+azureSupportTestAPIVersion, ""), http.StatusOK)

	// The update model's fields are top-level, not under properties.
	resp = do(http.MethodPatch, ticket+azureSupportTestAPIVersion, `{"severity":"minimal","status":"closed","contactDetails":{"preferredTimeZone":"UTC"}}`)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["status"] != "Closed" || props["severity"] != "minimal" || props["contactDetails"].(map[string]any)["preferredTimeZone"] != "UTC" || props["contactDetails"].(map[string]any)["firstName"] != "Ada" {
		t.Fatalf("unexpected patched ticket: %#v", props)
	}
	// Only an update reopens a ticket, not a PUT.
	resp = do(http.MethodPut, ticket+azureSupportTestAPIVersion, azureSupportTestTicket(problem))
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Closed" {
		t.Fatalf("a PUT reopened the ticket: %v", got)
	}
	for _, bad := range []string{`{"status":"resolved"}`, `{"severity":"urgent"}`, `{"contactDetails":{"preferredContactMethod":"fax"}}`} {
		azureResourcesExpectErrorCode(t, do(http.MethodPatch, ticket+azureSupportTestAPIVersion, bad), http.StatusBadRequest, "InvalidRequestContent")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ticket+"/communications/late"+azureSupportTestAPIVersion, `{"properties":{"subject":"Again","body":"Hello?"}}`), http.StatusBadRequest, "InvalidRequestContent")

	// Tickets also live at tenant scope, listed apart from subscription ones.
	tenant := "/azure/providers/Microsoft.Support/supportTickets/billing-question"
	azureResourcesExpectStatus(t, do(http.MethodPut, tenant+azureSupportTestAPIVersion, azureSupportTestTicket(problem)), http.StatusOK)
	resp = do(http.MethodGet, "/azure/providers/Microsoft.Support/supportTickets"+azureSupportTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected only the tenant ticket, got %d", len(got))
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Support/supportTickets"+azureSupportTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected only the subscription ticket, got %d", len(got))
	}

	workspace := sub + "/providers/Microsoft.Support/fileWorkspaces/quota-logs_2026.11"
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+azureSupportTestAPIVersion, `{}`), http.StatusCreated)
	file := workspace + "/files/usage.csv"
	azureResourcesExpectStatus(t, do(http.MethodPut, file+azureSupportTestAPIVersion, `{"properties":{"fileSize":2048,"chunkSize":1024,"numberOfChunks":2}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, file+"/upload"+azureSupportTestAPIVersion, `{"content":"Y29yZXMsMjAw","chunkIndex":1}`), http.StatusNoContent)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, workspace+"/files/many"+azureSupportTestAPIVersion, `{"properties":{"fileSize":3000,"chunkSize":1000,"numberOfChunks":3}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, file+"/upload"+azureSupportTestAPIVersion, `{"content":"Y29yZXMsMjAw","chunkIndex":2}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureSupportValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-support")
	problem := azureSupportTestService + "/problemClassifications/e12e3d1d-7fa0-af33-c6d0-3c50df9658a3"
	body := azureSupportTestTicket(problem)
	cases := []struct{ name, old, new string }{
		{"title", `"title":"Raise vCPU quota",`, ``},
		{"service", `"serviceId":"` + azureSupportTestService + `"`, `"serviceId":"quotas"`},
		{"problem service", `"problemClassificationId":"` + problem + `"`, `"problemClassificationId":"/providers/Microsoft.Support/services/517f2da6-78fd-0498-4e22-ad26996b1dfc/problemClassifications/e12e3d1d-7fa0-af33-c6d0-3c50df9658a3"`},
		{"severity", `"severity":"moderate"`, `"severity":"urgent"`},
		{"consent", `"advancedDiagnosticConsent":"Yes"`, `"advancedDiagnosticConsent":"Maybe"`},
		{"contact name", `"firstName":"Ada",`, ``},
		{"contact email", `"primaryEmailAddress":"ada@contoso.com"`, `"primaryEmailAddress":"ada"`},
		{"contact phone", `"preferredContactMethod":"email"`, `"preferredContactMethod":"phone"`},
		{"critical phone", `"severity":"moderate"`, `"severity":"critical"`},
		{"start time", `"severity":"moderate"`, `"severity":"moderate","problemStartTime":"yesterday"`},
	}
	for _, tc := range cases {
		changed := strings.Replace(body, tc.old, tc.new, 1)
		if changed == body {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, sub+"/providers/Microsoft.Support/supportTickets/t1"+azureSupportTestAPIVersion, changed)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/supportTickets/t1"+azureSupportTestAPIVersion, body), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/supportTickets/t1/communications/c"+azureSupportTestAPIVersion, `{"properties":{"subject":"s"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/fileWorkspaces/w/files/small"+azureSupportTestAPIVersion, `{"properties":{"fileSize":10,"chunkSize":10,"numberOfChunks":1}}`), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/fileWorkspaces/w"+azureSupportTestAPIVersion, `{}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/fileWorkspaces/w/files/big"+azureSupportTestAPIVersion, `{"properties":{"fileSize":5000001,"chunkSize":1,"numberOfChunks":1}}`), http.StatusBadRequest, "InvalidRequestContent")
	// Resource groups don't hold support tickets, so the support handler
	// doesn't serve them there.
	resp := do(http.MethodPut, sub+"/resourceGroups/rg-support/providers/Microsoft.Support/supportTickets/t2"+azureSupportTestAPIVersion, body)
	if raw, _ := io.ReadAll(resp.Body); strings.Contains(string(raw), "supportTicketId") {
		t.Fatalf("a ticket was created in a resource group: %s", raw)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.Support/fileWorkspaces/w%21"+azureSupportTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidResourceName")
}
