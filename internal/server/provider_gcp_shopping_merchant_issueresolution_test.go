package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantIssueresolutionRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	account := "accounts/123456"
	product := account + "/products/sku-1001"

	assertGCPShoppingMerchantIssueresolutionSuccess(t, ts, http.MethodPost, "/gcp/issueresolution/v1/"+account+":renderaccountissues", []byte(`{
		"contentOption":"CONTENT_OPTION_UNSPECIFIED",
		"userInputActionOption":"USER_INPUT_ACTION_RENDERING_OPTION_UNSPECIFIED"
	}`), "renderedIssues")
	assertGCPShoppingMerchantIssueresolutionSuccess(t, ts, http.MethodPost, "/gcp/issueresolution/v1/"+product+":renderproductissues", []byte(`{
		"contentOption":"CONTENT_OPTION_UNSPECIFIED",
		"userInputActionOption":"USER_INPUT_ACTION_RENDERING_OPTION_UNSPECIFIED"
	}`), "renderedIssues")
	assertGCPShoppingMerchantIssueresolutionSuccess(t, ts, http.MethodGet, "/gcp/issueresolution/v1/"+account+"/aggregateProductStatuses?pageSize=1", nil, "aggregateProductStatuses")
	assertGCPShoppingMerchantIssueresolutionSuccess(t, ts, http.MethodPost, "/gcp/issueresolution/v1/"+account+":triggeraction", []byte(`{
		"actionContext":"ctx-account-review",
		"actionInput":{
			"actionFlowId":"flow-review",
			"inputValues":[
				{"inputFieldId":"explanation","textInputValue":{"value":"All fields were corrected."}}
			]
		}
	}`), "action started")
}

func TestGCPShoppingMerchantIssueresolutionRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/issueresolution/v1/accounts/123456/aggregateProductStatuses?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_ListRejectsUnsupportedFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/issueresolution/v1/accounts/123456/aggregateProductStatuses?filter=channel=%22ONLINE%22", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution list filter validation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_RenderRejectsInvalidLanguageCode(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:renderaccountissues?languageCode=???", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution render invalid language code, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_TriggerActionRequiresPayload(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:triggeraction", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution triggeraction, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_TriggerActionRequiresActionInput(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:triggeraction", []byte(`{
		"actionContext":"ctx-account-review"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution triggeraction action input validation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_TriggerActionUnknownContextNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:triggeraction", []byte(`{
		"actionContext":"missing-context",
		"actionInput":{
			"actionFlowId":"flow-review",
			"inputValues":[
				{"inputFieldId":"explanation","textInputValue":{"value":"Fixed"}}
			]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant issueresolution triggeraction unknown context, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_TriggerActionLockedFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:triggeraction", []byte(`{
		"actionContext":"ctx-account-review-locked",
		"actionInput":{
			"actionFlowId":"flow-review",
			"inputValues":[
				{"inputFieldId":"explanation","textInputValue":{"value":"Fixed"}}
			]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant issueresolution triggeraction failed precondition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantIssueresolutionContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	}

	renderResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:renderaccountissues", []byte(`{
		"contentOption":"CONTENT_OPTION_UNSPECIFIED"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if renderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant issueresolution render account issues, got %d body=%s", renderResp.StatusCode, string(providerContractBody(t, renderResp)))
	}
	renderBody := providerContractJSONMap(t, renderResp)
	renderedIssues, ok := renderBody["renderedIssues"].([]any)
	if !ok || len(renderedIssues) == 0 {
		t.Fatalf("expected renderedIssues array, got %#v", renderBody["renderedIssues"])
	}
	firstIssue, ok := renderedIssues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected rendered issue object, got %#v", renderedIssues[0])
	}
	if _, ok := firstIssue["title"].(string); !ok {
		t.Fatalf("expected rendered issue title string, got %#v", firstIssue["title"])
	}
	if _, ok := firstIssue["prerenderedContent"].(string); !ok {
		t.Fatalf("expected rendered issue content string, got %#v", firstIssue["prerenderedContent"])
	}
	impact, ok := firstIssue["impact"].(map[string]any)
	if !ok {
		t.Fatalf("expected impact object, got %#v", firstIssue["impact"])
	}
	if _, ok := impact["message"].(string); !ok {
		t.Fatalf("expected impact message string, got %#v", impact["message"])
	}
	breakdowns, ok := impact["breakdowns"].([]any)
	if !ok || len(breakdowns) == 0 {
		t.Fatalf("expected impact breakdowns array, got %#v", impact["breakdowns"])
	}
	actions, ok := firstIssue["actions"].([]any)
	if !ok || len(actions) == 0 {
		t.Fatalf("expected actions array, got %#v", firstIssue["actions"])
	}
	secondAction, ok := actions[len(actions)-1].(map[string]any)
	if !ok {
		t.Fatalf("expected action object, got %#v", actions[len(actions)-1])
	}
	userInputAction, ok := secondAction["builtinUserInputAction"].(map[string]any)
	if !ok {
		t.Fatalf("expected builtinUserInputAction object, got %#v", secondAction["builtinUserInputAction"])
	}
	if _, ok := userInputAction["actionContext"].(string); !ok {
		t.Fatalf("expected actionContext string, got %#v", userInputAction["actionContext"])
	}
	flows, ok := userInputAction["flows"].([]any)
	if !ok || len(flows) == 0 {
		t.Fatalf("expected flows array, got %#v", userInputAction["flows"])
	}
	firstFlow, ok := flows[0].(map[string]any)
	if !ok {
		t.Fatalf("expected flow object, got %#v", flows[0])
	}
	if _, ok := firstFlow["id"].(string); !ok {
		t.Fatalf("expected flow id string, got %#v", firstFlow["id"])
	}
	inputs, ok := firstFlow["inputs"].([]any)
	if !ok || len(inputs) == 0 {
		t.Fatalf("expected flow inputs array, got %#v", firstFlow["inputs"])
	}
	firstInput, ok := inputs[0].(map[string]any)
	if !ok {
		t.Fatalf("expected input object, got %#v", inputs[0])
	}
	if _, ok := firstInput["id"].(string); !ok {
		t.Fatalf("expected input id string, got %#v", firstInput["id"])
	}
	if _, ok := firstInput["required"].(bool); !ok {
		t.Fatalf("expected input required bool, got %#v", firstInput["required"])
	}
	if _, ok := firstInput["textInput"].(map[string]any); !ok {
		t.Fatalf("expected input textInput object, got %#v", firstInput["textInput"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/issueresolution/v1/accounts/123456/aggregateProductStatuses?pageSize=1&filter=reporting_context%20%3D%20%22SHOPPING_ADS%22%20AND%20country%20%3D%20%22US%22", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant issueresolution list aggregate product statuses, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	statuses, ok := listBody["aggregateProductStatuses"].([]any)
	if !ok || len(statuses) == 0 {
		t.Fatalf("expected aggregateProductStatuses array, got %#v", listBody["aggregateProductStatuses"])
	}
	firstStatus, ok := statuses[0].(map[string]any)
	if !ok {
		t.Fatalf("expected aggregate status object, got %#v", statuses[0])
	}
	if _, ok := firstStatus["name"].(string); !ok {
		t.Fatalf("expected aggregate status name string, got %#v", firstStatus["name"])
	}
	if _, ok := firstStatus["country"].(string); !ok {
		t.Fatalf("expected aggregate status country string, got %#v", firstStatus["country"])
	}
	stats, ok := firstStatus["stats"].(map[string]any)
	if !ok {
		t.Fatalf("expected stats object, got %#v", firstStatus["stats"])
	}
	if _, ok := stats["activeCount"].(string); !ok {
		t.Fatalf("expected stats activeCount string, got %#v", stats["activeCount"])
	}
	itemLevelIssues, ok := firstStatus["itemLevelIssues"].([]any)
	if !ok || len(itemLevelIssues) == 0 {
		t.Fatalf("expected itemLevelIssues array, got %#v", firstStatus["itemLevelIssues"])
	}
	firstItemIssue, ok := itemLevelIssues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected item level issue object, got %#v", itemLevelIssues[0])
	}
	if _, ok := firstItemIssue["code"].(string); !ok {
		t.Fatalf("expected item level issue code string, got %#v", firstItemIssue["code"])
	}
	if _, ok := firstItemIssue["productCount"].(string); !ok {
		t.Fatalf("expected item level issue productCount string, got %#v", firstItemIssue["productCount"])
	}
}

func TestGCPShoppingMerchantIssueresolutionRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_issueresolution/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant issueresolution contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_issueresolution" {
		t.Fatalf("expected service=shopping_merchant_issueresolution, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantIssueresolutionContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPShoppingMerchantIssueresolutionSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant issueresolution router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
