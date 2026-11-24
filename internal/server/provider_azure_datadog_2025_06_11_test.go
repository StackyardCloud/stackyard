package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureDDOGTestVersion = "?api-version=" + azureDatadog20250611APIVersion

func TestAzureDatadog20250611Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ddog")
	monitor := sub + "/resourceGroups/rg-ddog/providers/Microsoft.Datadog/monitors/mon1"

	resp := do(http.MethodPut, monitor+azureDDOGTestVersion, `{"location":"eastus","sku":{"name":"payg_v2_Monthly"},"properties":{"userInfo":{"name":"Ada","emailAddress":"ada@example.com"},"datadogOrganizationProperties":{"apiKey":"secret","applicationKey":"secret2"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	org := props["datadogOrganizationProperties"].(map[string]any)
	if props["marketplaceSubscriptionStatus"] != "Active" || props["monitoringStatus"] != "Enabled" || org["apiKey"] != nil || org["applicationKey"] != nil || org["name"] != "mon1" {
		t.Fatalf("unexpected monitor properties: %#v", props)
	}

	resp = do(http.MethodPost, monitor+"/getDefaultKey"+azureDDOGTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	key, _ := providerContractJSONMap(t, resp)["key"].(string)
	if key == "" {
		t.Fatal("default key is empty")
	}
	resp = do(http.MethodPost, monitor+"/listApiKeys"+azureDDOGTestVersion, "")
	if keys := providerContractJSONMap(t, resp)["value"].([]any); len(keys) != 1 || keys[0].(map[string]any)["key"] != key {
		t.Fatalf("unexpected API keys: %#v", keys)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/setDefaultKey"+azureDDOGTestVersion, `{"key":"`+key+`"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, monitor+"/setDefaultKey"+azureDDOGTestVersion, `{"key":"other"}`), http.StatusBadRequest, "InvalidRequestContent")
	for _, action := range []string{"listHosts", "listMonitoredResources", "listLinkedResources"} {
		resp = do(http.MethodPost, monitor+"/"+action+azureDDOGTestVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
	}
	resp = do(http.MethodPost, monitor+"/refreshSetPasswordLink"+azureDDOGTestVersion, "")
	if link, _ := providerContractJSONMap(t, resp)["setPasswordLink"].(string); !strings.HasPrefix(link, "https://") {
		t.Fatalf("set password link = %q", link)
	}
	resp = do(http.MethodPost, monitor+"/getBillingInfo"+azureDDOGTestVersion, "")
	if saas := providerContractJSONMap(t, resp)["marketplaceSaasInfo"].(map[string]any); saas["subscribed"] != true {
		t.Fatalf("unexpected billing info: %#v", saas)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/resubscribe"+azureDDOGTestVersion, `{"sku":{"name":"payg_v2_Monthly"}}`), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/monitoredSubscriptions/default"+azureDDOGTestVersion, `{"properties":{"operation":"AddBegin","monitoredSubscriptionList":[]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/tagRules/default"+azureDDOGTestVersion, `{"properties":{"logRules":{"sendAadLogs":false}}}`), http.StatusOK)
	resp = do(http.MethodPut, monitor+"/singleSignOnConfigurations/sso"+azureDDOGTestVersion, `{"properties":{"singleSignOnState":"Enable","enterpriseAppId":"app"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if url, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["singleSignOnUrl"].(string); url == "" {
		t.Fatal("enabled single sign-on has no URL")
	}

	agreement := sub + "/providers/Microsoft.Datadog/agreements/default"
	azureResourcesExpectStatus(t, do(http.MethodPut, agreement+azureDDOGTestVersion, `{"properties":{"publisher":"datadog1591740804488","product":"dd_liftr_v2","plan":"payg_v2","accepted":true}}`), http.StatusOK)
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Datadog/agreements"+azureDDOGTestVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("want 1 agreement, got %d", len(items))
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Datadog/subscriptionStatuses/default"+azureDDOGTestVersion+"&datadogOrganizationId=org", "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["creationSupported"]; got != true {
		t.Fatalf("creationSupported = %v", got)
	}
	resp = do(http.MethodGet, "/azure/providers/Microsoft.Datadog/operations"+azureDDOGTestVersion, "")
	if ops := providerContractJSONMap(t, resp)["value"].([]any); len(ops) == 0 {
		t.Fatal("operations list is empty")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+azureDDOGTestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, monitor+"/tagRules/default"+azureDDOGTestVersion, ""), http.StatusNotFound)
}

func TestAzureDatadog20250611Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ddog")
	monitor := sub + "/resourceGroups/rg-ddog/providers/Microsoft.Datadog/monitors/mon2"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+azureDDOGTestVersion, `{"location":"eastus"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+azureDDOGTestVersion, `{"location":"eastus","sku":{"name":"p"},"properties":{"monitoringStatus":"Paused"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/listApiKeys"+azureDDOGTestVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+azureDDOGTestVersion, `{"location":"eastus","sku":{"name":"p"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/tagRules/custom"+azureDDOGTestVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/monitoredSubscriptions/default"+azureDDOGTestVersion, `{"properties":{"operation":"Pause"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+"/tagRules/default"+azureDDOGTestVersion, ""), http.StatusMethodNotAllowed)
}
