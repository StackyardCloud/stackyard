package server

import (
	"net/http"
	"testing"
	"time"
)

const eduTestAPI = "?api-version=" + azureEducation20211201PreviewAPIVersion

func TestAzureEducationLifecycle(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-edu")
	profile := "/azure/providers/Microsoft.Billing/billingAccounts/acct1/billingProfiles/bp1"
	lab := profile + "/invoiceSections/is1/providers/Microsoft.Education/labs/default"
	expires := time.Now().Add(240 * time.Hour).UTC().Format(time.RFC3339)

	resp := do(http.MethodPut, lab+eduTestAPI, `{"properties":{"displayName":"CS 101","description":"Intro","budgetPerStudent":{"currency":"USD","value":100},"expirationDate":"`+expires+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["status"] != "Active" || props["currency"] != "USD" {
		t.Fatalf("unexpected lab properties: %#v", props)
	}

	resp = do(http.MethodPost, lab+"/generateInviteCode"+eduTestAPI, `{"maxStudentCount":2}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	code, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["invitationCode"].(string)
	if code == "" {
		t.Fatal("no invitation code")
	}
	redeem := "/azure/providers/Microsoft.Education/redeemInvitationCode" + eduTestAPI
	azureResourcesExpectErrorCode(t, do(http.MethodPost, redeem, `{"redeemCode":"wrong","firstName":"Ada","lastName":"L"}`), http.StatusNotFound, "InvalidRedeemCode")
	azureResourcesExpectStatus(t, do(http.MethodPost, redeem, `{"redeemCode":"`+code+`","firstName":"Ada","lastName":"L"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, redeem, `{"redeemCode":"`+code+`","firstName":"Ada","lastName":"L"}`), http.StatusConflict, "Conflict")
	azureResourcesExpectStatus(t, do(http.MethodPost, redeem, `{"redeemCode":"`+code+`","firstName":"Bo","lastName":"M"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, redeem, `{"redeemCode":"`+code+`","firstName":"Cy","lastName":"N"}`), http.StatusConflict, "MaxStudentCountReached")

	resp = do(http.MethodGet, lab+"/joinRequests"+eduTestAPI, "")
	requests := providerContractJSONMap(t, resp)["value"].([]any)
	if len(requests) != 2 {
		t.Fatalf("%d join requests", len(requests))
	}
	first, second := requests[0].(map[string]any)["name"].(string), requests[1].(map[string]any)["name"].(string)
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/joinRequests/"+first+"/approve"+eduTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, lab+"/joinRequests/"+first+"/deny"+eduTestAPI, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/joinRequests/"+second+"/deny"+eduTestAPI, ""), http.StatusOK)

	resp = do(http.MethodPut, lab+"/students/grace"+eduTestAPI, `{"properties":{"firstName":"Grace","lastName":"H","email":"g@example.edu","role":"Admin","budget":{"currency":"USD","value":50},"expirationDate":"`+expires+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = do(http.MethodGet, lab+eduTestAPI, "")
	if total := providerContractJSONMap(t, resp)["properties"].(map[string]any)["totalBudget"].(map[string]any)["value"]; total != float64(150) {
		t.Fatalf("lab totalBudget = %v", total)
	}
	resp = do(http.MethodGet, lab+"/students"+eduTestAPI, "")
	if students := providerContractJSONMap(t, resp)["value"].([]any); len(students) != 2 {
		t.Fatalf("%d students", len(students))
	}

	resp = do(http.MethodGet, "/azure/providers/Microsoft.Education/studentLabs"+eduTestAPI, "")
	studentLabs := providerContractJSONMap(t, resp)["value"].([]any)
	if len(studentLabs) != 2 {
		t.Fatalf("%d student labs", len(studentLabs))
	}
	name := studentLabs[0].(map[string]any)["name"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Education/studentLabs/"+name+eduTestAPI, ""), http.StatusOK)

	resp = do(http.MethodGet, profile+"/providers/Microsoft.Education/grants/default"+eduTestAPI+"&includeAllocatedBudget=true", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["allocatedBudget"].(map[string]any)["value"]; got != float64(150) {
		t.Fatalf("grant allocatedBudget = %v", got)
	}
	for _, path := range []string{"/azure/providers/Microsoft.Education/grants", profile + "/providers/Microsoft.Education/labs", "/azure/providers/Microsoft.Education/operations"} {
		resp = do(http.MethodGet, path+eduTestAPI, "")
		if len(providerContractJSONMap(t, resp)["value"].([]any)) == 0 {
			t.Fatalf("%s listed nothing", path)
		}
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, lab+"/students/grace"+eduTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, lab+eduTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, lab+"/students/grace"+eduTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureEducationValidation(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-edu")
	section := "/azure/providers/Microsoft.Billing/billingAccounts/acct1/billingProfiles/bp1/invoiceSections/is1/providers/Microsoft.Education/labs"
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, section+"/default"+eduTestAPI, `{"properties":{"displayName":"x","description":"y","budgetPerStudent":{"currency":"USD","value":1},"expirationDate":"`+past+`"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, section+"/default"+eduTestAPI, `{"properties":{"displayName":"x","description":"y"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, section+"/other"+eduTestAPI, `{}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, section+"/default/students/s1"+eduTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, section+"/default/students/s1"+eduTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, section+"/default/generateInviteCode"+eduTestAPI, `{"maxStudentCount":1}`), http.StatusNotFound, "ResourceNotFound")
	// The billing provider still owns its own paths.
	resp := do(http.MethodGet, "/azure/providers/Microsoft.Billing/billingAccounts"+eduTestAPI, "")
	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("billing accounts route went missing")
	}
}
