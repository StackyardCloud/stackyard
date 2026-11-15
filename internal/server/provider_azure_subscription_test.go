package server

import (
	"net/http"
	"testing"
)

const azureSubscriptionTestAPIVersion = "?api-version=2021-10-01"

func TestAzureSubscriptionAliasLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sub")
	alias := "/azure/providers/Microsoft.Subscription/aliases/team-a"
	body := `{"properties":{"displayName":"Team A","workload":"DevTest","billingScope":"/providers/Microsoft.Billing/billingAccounts/1234/enrollmentAccounts/5678"}}`

	resp := do(http.MethodPut, alias+azureSubscriptionTestAPIVersion, body)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	id := created["properties"].(map[string]any)["subscriptionId"].(string)
	if created["id"] != "/providers/Microsoft.Subscription/aliases/team-a" || len(id) != 36 {
		t.Fatalf("unexpected alias: %#v", created)
	}
	// The same alias names the same subscription.
	resp = do(http.MethodPut, alias+azureSubscriptionTestAPIVersion, body)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["subscriptionId"]; got != id {
		t.Fatalf("expected %s again, got %v", id, got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure/providers/Microsoft.Subscription/aliases/existing"+azureSubscriptionTestAPIVersion, `{"properties":{"subscriptionId":"11111111-2222-3333-4444-555555555555"}}`), http.StatusCreated)

	// Aliases live at tenant scope only: the shared handler doesn't serve
	// them under a subscription, so this creates no alias.
	do(http.MethodPut, sub+"/providers/Microsoft.Subscription/aliases/other"+azureSubscriptionTestAPIVersion, body)
	resp = do(http.MethodGet, "/azure/providers/Microsoft.Subscription/aliases"+azureSubscriptionTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(got))
	}

	for _, action := range []string{"cancel", "enable"} {
		resp = do(http.MethodPost, "/azure/subscriptions/"+id+"/providers/Microsoft.Subscription/"+action+azureSubscriptionTestAPIVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp)["subscriptionId"]; got != id {
			t.Fatalf("%s: unexpected %v", action, got)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/subscriptions/"+id+"/providers/Microsoft.Subscription/rename"+azureSubscriptionTestAPIVersion, `{"subscriptionName":"Team A (renamed)"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, "/azure/subscriptions/"+id+"/providers/Microsoft.Subscription/rename"+azureSubscriptionTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-sub/providers/Microsoft.Subscription/cancel"+azureSubscriptionTestAPIVersion, ""), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodDelete, alias+azureSubscriptionTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, alias+azureSubscriptionTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureSubscriptionAliasValidation(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-sub")
	alias := "/azure/providers/Microsoft.Subscription/aliases/team-a" + azureSubscriptionTestAPIVersion
	cases := map[string]string{
		"bad subscription id": `{"properties":{"subscriptionId":"abc"}}`,
		"no display name":     `{"properties":{"workload":"Production","billingScope":"/providers/Microsoft.Billing/billingAccounts/1"}}`,
		"bad workload":        `{"properties":{"displayName":"A","workload":"Test","billingScope":"/providers/Microsoft.Billing/billingAccounts/1"}}`,
		"bad billing scope":   `{"properties":{"displayName":"A","workload":"Production","billingScope":"/subscriptions/x"}}`,
	}
	for name, body := range cases {
		resp := do(http.MethodPut, alias, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
