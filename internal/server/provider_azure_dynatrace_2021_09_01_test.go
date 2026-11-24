package server

import (
	"net/http"
	"strings"
	"testing"
)

const dynaTestAPIVersion = "?api-version=2021-09-01"

const dynaTestMonitorBody = `{"location":"eastus","properties":{"monitoringStatus":"Enabled",` +
	`"planData":{"planDetails":"dynatraceapitestplan","billingCycle":"MONTHLY","usageType":"COMMITTED"},` +
	`"userInfo":{"firstName":"Ada","lastName":"Lovelace","emailAddress":"ada@contoso.com"}}}`

func TestAzureDynatraceMonitorLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dyna")
	monitor := sub + "/resourceGroups/rg-dyna/providers/Dynatrace.Observability/monitors/obs"

	resp := do(http.MethodPut, monitor+dynaTestAPIVersion, dynaTestMonitorBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["userInfo"] != nil || props["marketplaceSubscriptionStatus"] != "Active" || props["dynatraceEnvironmentProperties"] == nil {
		t.Fatalf("unexpected monitor properties: %#v", props)
	}

	resp = do(http.MethodPost, monitor+"/getAccountCredentials"+dynaTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if key, _ := providerContractJSONMap(t, resp)["apiKey"].(string); !strings.HasPrefix(key, "dt0c01.") {
		t.Fatalf("unexpected credentials key %q", key)
	}
	resp = do(http.MethodPost, monitor+"/getVMHostPayload"+dynaTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["environmentId"] == nil {
		t.Fatal("expected an environment ID")
	}
	for _, action := range []string{"listHosts", "listAppServices"} {
		if items := adnsTestValue(t, do(http.MethodPost, monitor+"/"+action+dynaTestAPIVersion, "")); len(items) != 0 {
			t.Fatalf("%s: expected no items, got %#v", action, items)
		}
	}
	if items := adnsTestValue(t, do(http.MethodPost, monitor+"/listMonitoredResources"+dynaTestAPIVersion, "")); len(items) != 0 {
		t.Fatalf("expected no other resources, got %#v", items)
	}
	if envs := adnsTestValue(t, do(http.MethodPost, monitor+"/listLinkableEnvironments"+dynaTestAPIVersion, `{"userPrincipal":"ada@contoso.com","region":"eastus"}`)); len(envs) != 1 {
		t.Fatalf("unexpected linkable environments: %#v", envs)
	}

	resp = do(http.MethodPost, monitor+"/getSSODetails"+dynaTestAPIVersion, `{"userPrincipal":"ada@contoso.com"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["isSsoEnabled"]; got != "Disabled" {
		t.Fatalf("expected SSO disabled, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/singleSignOnConfigurations/default"+dynaTestAPIVersion,
		`{"properties":{"singleSignOnState":"Enable","singleSignOnUrl":"https://sso.dynatrace.com/app"}}`), http.StatusCreated)
	resp = do(http.MethodPost, monitor+"/getSSODetails"+dynaTestAPIVersion, `{"userPrincipal":"ada@contoso.com"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["isSsoEnabled"] != "Enabled" || got["singleSignOnUrl"] != "https://sso.dynatrace.com/app" {
		t.Fatalf("unexpected SSO details: %#v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/tagRules/default"+dynaTestAPIVersion,
		`{"properties":{"logRules":{"filteringTags":[{"name":"env","value":"prod","action":"Include"}]}}}`), http.StatusCreated)
	if ops := adnsTestValue(t, do(http.MethodGet, "/azure/providers/Dynatrace.Observability/operations"+dynaTestAPIVersion, "")); len(ops) == 0 {
		t.Fatal("expected provider operations")
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+dynaTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, monitor+"/tagRules/default"+dynaTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDynatraceValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dyna")
	monitors := sub + "/resourceGroups/rg-dyna/providers/Dynatrace.Observability/monitors/"

	cases := []struct{ name, body string }{
		{"missing plan", `{"location":"eastus","properties":{}}`},
		{"bad billing cycle", `{"location":"eastus","properties":{"planData":{"planDetails":"p","billingCycle":"DAILY"}}}`},
		{"bad email", `{"location":"eastus","properties":{"planData":{"planDetails":"p"},"userInfo":{"emailAddress":"not-an-email"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, monitors+"m1"+dynaTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	monitor := monitors + "m1"
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+dynaTestAPIVersion, dynaTestMonitorBody), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, monitor+"/getSSODetails"+dynaTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/tagRules/custom"+dynaTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/tagRules/default"+dynaTestAPIVersion,
		`{"properties":{"metricRules":{"filteringTags":[{"name":"env","action":"Maybe"}]}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/singleSignOnConfigurations/default"+dynaTestAPIVersion,
		`{"properties":{"singleSignOnState":"On"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+"/singleSignOnConfigurations/default"+dynaTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, monitors+"absent/getAccountCredentials"+dynaTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}
