package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// azureARMTestClient sends ARM requests to a test server with the default
// Azure test headers.
type azureARMTestClient struct {
	t       *testing.T
	ts      *httptest.Server
	headers map[string]string
}

func newAzureARMTestClient(t *testing.T, group string) *azureARMTestClient {
	t.Helper()
	return newAzureARMTestClientOn(t, newAzureResourcesTestServer(t), group)
}

// newAzureARMTestClientNoAuth is newAzureARMTestClient on a server with
// Azure authentication disabled, so a data plane's own credentials (such as
// Event Grid's aeg-sas-key or SharedAccessKey) reach its handler.
func newAzureARMTestClientNoAuth(t *testing.T, group string) *azureARMTestClient {
	t.Helper()
	return newAzureARMTestClientOn(t, newProviderContractServer(t, Config{
		Addr: "127.0.0.1:0", Providers: []string{providerAzure}, AzureAuthMode: "disabled",
		AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error",
	}), group)
}

func newAzureARMTestClientOn(t *testing.T, ts *httptest.Server, group string) *azureARMTestClient {
	t.Helper()
	c := &azureARMTestClient{t: t, ts: ts, headers: azureResourcesTestHeaders()}
	c.expect(http.MethodPut, c.sub()+"/resourcegroups/"+group+"?api-version=2021-04-01", `{"location":"eastus"}`, http.StatusCreated)
	return c
}

func (c *azureARMTestClient) sub() string {
	return "/azure/subscriptions/" + azureResourcesTestSubscription
}

func (c *azureARMTestClient) do(method, path, body string) *http.Response {
	c.t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	return providerContractRequest(c.t, c.ts, method, path, raw, c.headers)
}

// expect sends the request, checks the status and returns the JSON body
// (nil when there is none).
func (c *azureARMTestClient) expect(method, path, body string, status int) map[string]any {
	c.t.Helper()
	resp := c.do(method, path, body)
	azureResourcesExpectStatus(c.t, resp, status)
	if resp.ContentLength == 0 || !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		_ = resp.Body.Close()
		return nil
	}
	return providerContractJSONMap(c.t, resp)
}

func (c *azureARMTestClient) expectError(method, path, body string, status int, code string) {
	c.t.Helper()
	azureResourcesExpectErrorCode(c.t, c.do(method, path, body), status, code)
}

func azureTestList(t *testing.T, payload map[string]any) []any {
	t.Helper()
	items, ok := payload["value"].([]any)
	if !ok {
		t.Fatalf("expected a value list, got %#v", payload)
	}
	return items
}

const evhTestAPI = "?api-version=2026-01-01"

func TestAzureEventHubNamespaceLifecycle(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-evh")
	ns := c.sub() + "/resourceGroups/rg-evh/providers/Microsoft.EventHub/namespaces/telemetry-ns"

	created := c.expect(http.MethodPut, ns+evhTestAPI, `{"location":"eastus","sku":{"name":"Standard","tier":"Standard","capacity":2}}`, http.StatusCreated)
	props := created["properties"].(map[string]any)
	if props["serviceBusEndpoint"] != "https://telemetry-ns.servicebus.windows.net:443/" || props["kafkaEnabled"] != true {
		t.Fatalf("unexpected namespace properties: %#v", props)
	}

	// New namespaces come with RootManageSharedAccessKey and the default network rule set.
	keys := c.expect(http.MethodPost, ns+"/authorizationRules/RootManageSharedAccessKey/listKeys"+evhTestAPI, "", http.StatusOK)
	if !strings.HasPrefix(keys["primaryConnectionString"].(string), "Endpoint=sb://telemetry-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;") {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	regenerated := c.expect(http.MethodPost, ns+"/authorizationRules/RootManageSharedAccessKey/regenerateKeys"+evhTestAPI, `{"keyType":"PrimaryKey"}`, http.StatusOK)
	if regenerated["primaryKey"] == keys["primaryKey"] || regenerated["secondaryKey"] != keys["secondaryKey"] {
		t.Fatalf("regenerating the primary key must change only it: before %#v after %#v", keys, regenerated)
	}
	if again := c.expect(http.MethodPost, ns+"/authorizationRules/RootManageSharedAccessKey/listKeys"+evhTestAPI, "", http.StatusOK); again["primaryKey"] != regenerated["primaryKey"] {
		t.Fatalf("listKeys must return the regenerated key")
	}
	rules := c.expect(http.MethodGet, ns+"/networkRuleSets/default"+evhTestAPI, "", http.StatusOK)
	if rules["properties"].(map[string]any)["defaultAction"] != "Allow" {
		t.Fatalf("unexpected default network rule set: %#v", rules)
	}

	hub := c.expect(http.MethodPut, ns+"/eventhubs/orders"+evhTestAPI, `{"properties":{"partitionCount":3,"messageRetentionInDays":2}}`, http.StatusCreated)
	if ids := hub["properties"].(map[string]any)["partitionIds"].([]any); len(ids) != 3 || ids[2] != "2" {
		t.Fatalf("unexpected partition IDs: %#v", hub)
	}
	c.expectError(http.MethodPut, ns+"/eventhubs/orders"+evhTestAPI, `{"properties":{"partitionCount":2}}`, http.StatusConflict, "Conflict")
	groups := azureTestList(t, c.expect(http.MethodGet, ns+"/eventhubs/orders/consumergroups"+evhTestAPI, "", http.StatusOK))
	if len(groups) != 1 || groups[0].(map[string]any)["name"] != "$Default" {
		t.Fatalf("expected the $Default consumer group, got %#v", groups)
	}
	c.expect(http.MethodPut, ns+"/eventhubs/orders/consumergroups/analytics"+evhTestAPI, `{"properties":{"userMetadata":"team=data"}}`, http.StatusCreated)
	c.expectError(http.MethodDelete, ns+"/eventhubs/orders/consumergroups/$Default"+evhTestAPI, "", http.StatusConflict, "Conflict")
	c.expect(http.MethodPut, ns+"/eventhubs/orders/authorizationRules/sender"+evhTestAPI, `{"properties":{"rights":["Send"]}}`, http.StatusCreated)
	hubKeys := c.expect(http.MethodPost, ns+"/eventhubs/orders/authorizationRules/sender/listKeys"+evhTestAPI, "", http.StatusOK)
	if !strings.HasSuffix(hubKeys["primaryConnectionString"].(string), ";EntityPath=orders") {
		t.Fatalf("event hub rule connection strings carry the entity path: %#v", hubKeys)
	}

	c.expect(http.MethodPut, ns+"/applicationGroups/ingest"+evhTestAPI,
		`{"properties":{"clientAppGroupIdentifier":"SASKeyName=sender","policies":[{"type":"ThrottlingPolicy","name":"cap","metricId":"IncomingBytes","rateLimitThreshold":1000}]}}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/schemagroups/contracts"+evhTestAPI, `{"properties":{"schemaCompatibility":"Backward","schemaType":"Avro"}}`, http.StatusCreated)
	links := azureTestList(t, c.expect(http.MethodGet, ns+"/privateLinkResources"+evhTestAPI, "", http.StatusOK))
	if len(links) != 1 {
		t.Fatalf("expected the namespace private link resource, got %#v", links)
	}

	// Name checks see namespaces across the Service Bus family.
	check := c.expect(http.MethodPost, c.sub()+"/providers/Microsoft.EventHub/checkNameAvailability"+evhTestAPI, `{"name":"telemetry-ns"}`, http.StatusOK)
	if check["nameAvailable"] != false || check["reason"] != "NameInUse" {
		t.Fatalf("unexpected name check: %#v", check)
	}

	c.expect(http.MethodDelete, ns+evhTestAPI, "", http.StatusOK)
	c.expectError(http.MethodGet, ns+"/eventhubs/orders"+evhTestAPI, "", http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureEventHubDisasterRecoveryAndClusters(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-evh-dr")
	base := c.sub() + "/resourceGroups/rg-evh-dr/providers/Microsoft.EventHub"
	primary, secondary := base+"/namespaces/orders-primary", base+"/namespaces/orders-secondary"
	for _, ns := range []string{primary, secondary} {
		c.expect(http.MethodPut, ns+evhTestAPI, `{"location":"eastus","sku":{"name":"Premium"}}`, http.StatusCreated)
	}
	partnerID := strings.TrimPrefix(secondary, "/azure")
	c.expectError(http.MethodPut, primary+"/disasterRecoveryConfigs/orders-alias"+evhTestAPI, `{"properties":{}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, primary+"/disasterRecoveryConfigs/orders-alias"+evhTestAPI, `{"properties":{"partnerNamespace":"`+partnerID+`"}}`, http.StatusCreated)

	name := c.expect(http.MethodPost, primary+"/disasterRecoveryConfigs/CheckNameAvailability"+evhTestAPI, `{"name":"orders-alias"}`, http.StatusOK)
	if name["nameAvailable"] != false {
		t.Fatalf("an alias name in use must be unavailable: %#v", name)
	}
	aliasRules := azureTestList(t, c.expect(http.MethodGet, primary+"/disasterRecoveryConfigs/orders-alias/authorizationRules"+evhTestAPI, "", http.StatusOK))
	if len(aliasRules) != 1 || !strings.Contains(aliasRules[0].(map[string]any)["id"].(string), "/disasterRecoveryConfigs/orders-alias/authorizationRules/RootManageSharedAccessKey") {
		t.Fatalf("unexpected alias rules: %#v", aliasRules)
	}
	aliasKeys := c.expect(http.MethodPost, primary+"/disasterRecoveryConfigs/orders-alias/authorizationRules/RootManageSharedAccessKey/listKeys"+evhTestAPI, "", http.StatusOK)
	if !strings.HasPrefix(aliasKeys["aliasPrimaryConnectionString"].(string), "Endpoint=sb://orders-alias.servicebus.windows.net/") {
		t.Fatalf("unexpected alias keys: %#v", aliasKeys)
	}
	c.expectError(http.MethodGet, primary+"/disasterRecoveryConfigs/missing/authorizationRules"+evhTestAPI, "", http.StatusNotFound, "ParentResourceNotFound")
	c.expect(http.MethodPost, primary+"/disasterRecoveryConfigs/orders-alias/breakPairing"+evhTestAPI, "", http.StatusOK)
	alias := c.expect(http.MethodGet, primary+"/disasterRecoveryConfigs/orders-alias"+evhTestAPI, "", http.StatusOK)
	if alias["properties"].(map[string]any)["partnerNamespace"] != "" {
		t.Fatalf("breaking the pairing clears the partner: %#v", alias)
	}

	cluster := base + "/clusters/dedicated-1"
	c.expectError(http.MethodPut, cluster+evhTestAPI, `{"location":"eastus","sku":{"name":"Standard"}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, cluster+evhTestAPI, `{"location":"eastus","sku":{"name":"Dedicated","capacity":1}}`, http.StatusCreated)
	hosted := base + "/namespaces/hosted-ns"
	c.expect(http.MethodPut, hosted+evhTestAPI, `{"location":"eastus","sku":{"name":"Premium"},"properties":{"clusterArmId":"`+strings.TrimPrefix(cluster, "/azure")+`"}}`, http.StatusCreated)
	if got := azureTestList(t, c.expect(http.MethodGet, cluster+"/namespaces"+evhTestAPI, "", http.StatusOK)); len(got) != 1 {
		t.Fatalf("expected the hosted namespace, got %#v", got)
	}
	c.expectError(http.MethodDelete, cluster+evhTestAPI, "", http.StatusConflict, "Conflict")
	quota := c.expect(http.MethodPatch, cluster+"/quotaConfiguration/default"+evhTestAPI, `{"properties":{"settings":{"eventhub-per-namespace-quota":"20"}}}`, http.StatusOK)
	if quota["properties"].(map[string]any)["settings"].(map[string]any)["eventhub-per-namespace-quota"] != "20" {
		t.Fatalf("unexpected quota configuration: %#v", quota)
	}
	c.expectError(http.MethodPatch, cluster+"/quotaConfiguration/default"+evhTestAPI, `{"properties":{"settings":{"x":"many"}}}`, http.StatusBadRequest, "InvalidRequestContent")
	if regions := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.EventHub/availableClusterRegions"+evhTestAPI, "", http.StatusOK)); len(regions) == 0 {
		t.Fatalf("expected available cluster regions")
	}
}

func TestAzureEventHubValidation(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-evh-bad")
	ns := c.sub() + "/resourceGroups/rg-evh-bad/providers/Microsoft.EventHub/namespaces"
	cases := []struct{ path, body string }{
		{"/bad", `{"location":"eastus","sku":{"name":"Standard"}}`},                         // name too short
		{"/valid-name-ns", `{"location":"eastus","sku":{"name":"Gold"}}`},                   // unknown sku
		{"/valid-name-ns", `{"location":"eastus","sku":{"name":"Standard","capacity":41}}`}, // capacity out of range
	}
	for _, tc := range cases {
		resp := c.do(http.MethodPut, ns+tc.path+evhTestAPI, tc.body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("PUT %s %s: expected 400, got %d", tc.path, tc.body, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
	c.expect(http.MethodPut, ns+"/valid-name-ns"+evhTestAPI, `{"location":"eastus","sku":{"name":"Basic"}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/valid-name-ns/eventhubs/big"+evhTestAPI, `{"properties":{"partitionCount":5000}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPost, ns+"/valid-name-ns/failover"+evhTestAPI, `{"properties":{"primaryLocation":"westus"}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPost, ns+"/valid-name-ns/authorizationRules/RootManageSharedAccessKey/regenerateKeys"+evhTestAPI, `{"keyType":"Tertiary"}`, http.StatusBadRequest, "InvalidRequestContent")
}
