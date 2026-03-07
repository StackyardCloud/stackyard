package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingCSSRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	accountName := "accounts/123456"
	labelName := accountName + "/labels/label-summer-2026"
	inputName := accountName + "/cssProductInputs/en~US~sku-1"

	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName+":listChildAccounts?pageSize=1", nil, "accounts")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName, nil, `"accountType":"CSS_DOMAIN"`)
	assertGCPShoppingCSSSuccess(t, ts, http.MethodPost, "/gcp/v1/"+accountName+":updateLabels", []byte(`{
		"name":"accounts/123456",
		"labelIds":[1001,1002]
	}`), `"labelIds":["1001","1002"]`)

	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/labels?pageSize=1", nil, "accountLabels")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodPost, "/gcp/v1/"+accountName+"/labels", []byte(`{
		"displayName":"Summer 2026",
		"description":"Seasonal label"
	}`), "label-summer-2026")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodPatch, "/gcp/v1/"+labelName+"?updateMask=displayName,description", []byte(`{
		"name":"accounts/123456/labels/label-summer-2026",
		"displayName":"Summer 2026 Updated",
		"description":"Updated seasonal label"
	}`), "Summer 2026 Updated")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodDelete, "/gcp/v1/"+labelName, nil, "{}")

	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/cssProducts/en~US~sku-1", nil, `"rawProvidedId":"sku-1"`)
	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/cssProducts?pageSize=1", nil, "cssProducts")

	assertGCPShoppingCSSSuccess(t, ts, http.MethodPost, "/gcp/v1/"+accountName+"/cssProductInputs:insert", []byte(`{
		"rawProvidedId":"sku-1",
		"contentLanguage":"en",
		"feedLabel":"US",
		"attributes":{"title":"Stackyard Tee"}
	}`), "cssProductInputs/en~US~sku-1")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodPatch, "/gcp/v1/"+inputName+"?updateMask=attributes.title", []byte(`{
		"name":"accounts/123456/cssProductInputs/en~US~sku-1",
		"rawProvidedId":"sku-1",
		"contentLanguage":"en",
		"feedLabel":"US",
		"attributes":{"title":"Stackyard Tee Updated"}
	}`), "Stackyard Tee Updated")
	assertGCPShoppingCSSSuccess(t, ts, http.MethodDelete, "/gcp/v1/"+inputName, nil, "{}")

	assertGCPShoppingCSSSuccess(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/quotas?pageSize=1", nil, "quotaGroups")
}

func TestGCPShoppingCSSRouter_ListChildAccountsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456:listChildAccounts?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css list child accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_UpdateLabelsNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/accounts/123456:updateLabels", []byte(`{
		"name":"accounts/999999",
		"labelIds":[1001]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css update labels, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_CreateAccountLabelRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/accounts/123456/labels", []byte(`{
		"description":"no display name"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css create account label, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_CreateAccountLabelRejectsDuplicate(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/accounts/123456/labels", []byte(`{
		"displayName":"Existing Label"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shopping css create account label, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"AlreadyExists"`) {
		t.Fatalf("expected AlreadyExists error in response")
	}
}

func TestGCPShoppingCSSRouter_UpdateAccountLabelRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/accounts/123456/labels/label-1", []byte(`{
		"name":"accounts/123456/labels/label-1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css update account label, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_DeleteAccountLabelMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/accounts/123456/labels/missing-label", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping css delete account label, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingCSSRouter_InsertCssProductInputRequiresFields(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/accounts/123456/cssProductInputs:insert", []byte(`{
		"contentLanguage":"en"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css insert product input, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_UpdateCssProductInputNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/accounts/123456/cssProductInputs/en~US~sku-1?updateMask=attributes.title", []byte(`{
		"name":"accounts/123456/cssProductInputs/en~US~sku-2",
		"rawProvidedId":"sku-1",
		"contentLanguage":"en",
		"feedLabel":"US",
		"attributes":{"title":"Mismatch"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping css update product input, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingCSSRouter_DeleteCssProductInputMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/accounts/123456/cssProductInputs/missing-input", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping css delete product input, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingCSSRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingCSSContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	}

	accountResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456", nil, headers)
	if accountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css get account, got %d body=%s", accountResp.StatusCode, string(providerContractBody(t, accountResp)))
	}
	accountBody := providerContractJSONMap(t, accountResp)
	if _, ok := accountBody["name"].(string); !ok {
		t.Fatalf("expected account name string, got %#v", accountBody["name"])
	}
	if _, ok := accountBody["labelIds"].([]any); !ok {
		t.Fatalf("expected account labelIds array, got %#v", accountBody["labelIds"])
	}
	if _, ok := accountBody["accountType"].(string); !ok {
		t.Fatalf("expected account type string, got %#v", accountBody["accountType"])
	}

	labelsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456/labels?pageSize=1", nil, headers)
	if labelsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css list account labels, got %d body=%s", labelsResp.StatusCode, string(providerContractBody(t, labelsResp)))
	}
	labelsBody := providerContractJSONMap(t, labelsResp)
	accountLabels, ok := labelsBody["accountLabels"].([]any)
	if !ok || len(accountLabels) == 0 {
		t.Fatalf("expected accountLabels array, got %#v", labelsBody["accountLabels"])
	}
	firstLabel, ok := accountLabels[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first account label object, got %#v", accountLabels[0])
	}
	if _, ok := firstLabel["name"].(string); !ok {
		t.Fatalf("expected account label name string, got %#v", firstLabel["name"])
	}
	if _, ok := firstLabel["displayName"].(string); !ok {
		t.Fatalf("expected account label displayName string, got %#v", firstLabel["displayName"])
	}

	productResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456/cssProducts/en~US~sku-1", nil, headers)
	if productResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css get css product, got %d body=%s", productResp.StatusCode, string(providerContractBody(t, productResp)))
	}
	productBody := providerContractJSONMap(t, productResp)
	if _, ok := productBody["rawProvidedId"].(string); !ok {
		t.Fatalf("expected css product rawProvidedId string, got %#v", productBody["rawProvidedId"])
	}
	attributes, ok := productBody["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected css product attributes object, got %#v", productBody["attributes"])
	}
	if _, ok := attributes["title"].(string); !ok {
		t.Fatalf("expected css product attributes.title string, got %#v", attributes["title"])
	}

	quotaResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456/quotas?pageSize=1", nil, headers)
	if quotaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css list quota groups, got %d body=%s", quotaResp.StatusCode, string(providerContractBody(t, quotaResp)))
	}
	quotaBody := providerContractJSONMap(t, quotaResp)
	quotaGroups, ok := quotaBody["quotaGroups"].([]any)
	if !ok || len(quotaGroups) == 0 {
		t.Fatalf("expected quotaGroups array, got %#v", quotaBody["quotaGroups"])
	}
	firstQuotaGroup, ok := quotaGroups[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first quota group object, got %#v", quotaGroups[0])
	}
	if _, ok := firstQuotaGroup["name"].(string); !ok {
		t.Fatalf("expected quota group name string, got %#v", firstQuotaGroup["name"])
	}
	if _, ok := firstQuotaGroup["methodDetails"].([]any); !ok {
		t.Fatalf("expected quota group methodDetails array, got %#v", firstQuotaGroup["methodDetails"])
	}
}

func TestGCPShoppingCSSRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/accounts/123456?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_css" {
		t.Fatalf("expected service=shopping_css, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingCSSContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingCSSSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping css router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
