package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantProductstudioRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	name := "accounts/123456"

	assertGCPShoppingMerchantProductstudioSuccess(t, ts, "/gcp/productstudio/v1alpha/"+name+"/generatedImages:generateProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"outputConfig":{"returnImageUri":true},
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"},
		"config":{"productDescription":"Stackyard red dress","backgroundDescription":"Clean studio backdrop"}
	}`), `"generatedImage"`)

	assertGCPShoppingMerchantProductstudioSuccess(t, ts, "/gcp/productstudio/v1alpha/"+name+"/generatedImages:removeProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageBytes":"aW1hZ2UtYnl0ZXM="},
		"config":{"backgroundColor":{"red":"255","green":"255","blue":"255"}}
	}`), `"generatedImage"`)

	assertGCPShoppingMerchantProductstudioSuccess(t, ts, "/gcp/productstudio/v1alpha/"+name+"/generatedImages:upscaleProductImage", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"}
	}`), `"generatedImage"`)

	assertGCPShoppingMerchantProductstudioSuccess(t, ts, "/gcp/productstudio/v1alpha/"+name+":generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456",
		"productInfo":{"productAttributes":{"title":"Red Dress","brand":"Stackyard"}},
		"outputSpec":{"workflowId":"title","tone":"playful"}
	}`), `"title"`)
}

func TestGCPShoppingMerchantProductstudioRouter_GenerateImageRequiresInputImage(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:generateProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"config":{"productDescription":"dress","backgroundDescription":"studio"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant productstudio generate image without inputImage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_GenerateBackgroundRequiresConfigFields(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:generateProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"},
		"config":{"productDescription":"dress"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant productstudio generate background missing config fields, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_TextRequiresProductInfo(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456:generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant productstudio text suggestions without productInfo, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_TextRejectsUnsupportedTone(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456:generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456",
		"productInfo":{"productAttributes":{"title":"Red Dress","brand":"Stackyard"}},
		"outputSpec":{"workflowId":"title","tone":"dramatic"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant productstudio text suggestions unsupported tone, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_MissingAccountNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/missing/generatedImages:upscaleProductImage", []byte(`{
		"name":"accounts/missing",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant productstudio missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_PathBodyNameMismatchFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:upscaleProductImage", []byte(`{
		"name":"accounts/654321",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant productstudio path/body mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantProductstudioRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductstudioContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	}

	generateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:generateProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"outputConfig":{"returnImageUri":true},
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"},
		"config":{"productDescription":"Stackyard red dress","backgroundDescription":"Clean studio backdrop"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if generateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio generate background, got %d body=%s", generateResp.StatusCode, string(providerContractBody(t, generateResp)))
	}
	generateBody := providerContractJSONMap(t, generateResp)
	generatedImage, ok := generateBody["generatedImage"].(map[string]any)
	if !ok {
		t.Fatalf("expected generatedImage object, got %#v", generateBody["generatedImage"])
	}
	if _, ok := generatedImage["name"].(string); !ok {
		t.Fatalf("expected generatedImage.name string, got %#v", generatedImage["name"])
	}
	if _, ok := generatedImage["uri"].(string); !ok {
		t.Fatalf("expected generatedImage.uri string, got %#v", generatedImage["uri"])
	}
	if _, ok := generatedImage["generationTime"].(string); !ok {
		t.Fatalf("expected generatedImage.generationTime string, got %#v", generatedImage["generationTime"])
	}

	removeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:removeProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageBytes":"aW1hZ2UtYnl0ZXM="}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if removeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio remove background, got %d body=%s", removeResp.StatusCode, string(providerContractBody(t, removeResp)))
	}
	removeBody := providerContractJSONMap(t, removeResp)
	removeImage, ok := removeBody["generatedImage"].(map[string]any)
	if !ok {
		t.Fatalf("expected generatedImage object, got %#v", removeBody["generatedImage"])
	}
	if _, ok := removeImage["imageBytes"].(string); !ok {
		t.Fatalf("expected generatedImage.imageBytes string, got %#v", removeImage["imageBytes"])
	}

	textResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456:generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456",
		"productInfo":{"productAttributes":{"title":"Red Dress","brand":"Stackyard","color":"red"}},
		"outputSpec":{"workflowId":"title","tone":"playful","targetLanguage":"en"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if textResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio text suggestions, got %d body=%s", textResp.StatusCode, string(providerContractBody(t, textResp)))
	}
	textBody := providerContractJSONMap(t, textResp)
	title, ok := textBody["title"].(map[string]any)
	if !ok {
		t.Fatalf("expected title object, got %#v", textBody["title"])
	}
	if _, ok := title["text"].(string); !ok {
		t.Fatalf("expected title.text string, got %#v", title["text"])
	}
	if _, ok := title["score"].(float64); !ok {
		t.Fatalf("expected title.score float, got %#v", title["score"])
	}
	attributes, ok := textBody["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes object, got %#v", textBody["attributes"])
	}
	if _, ok := attributes["workflow"].(string); !ok {
		t.Fatalf("expected attributes.workflow string, got %#v", attributes["workflow"])
	}
	metadata, ok := textBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata object, got %#v", textBody["metadata"])
	}
	if _, ok := metadata["metadata"].(map[string]any); !ok {
		t.Fatalf("expected metadata.metadata object, got %#v", metadata["metadata"])
	}

	upscaleResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:upscaleProductImage", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if upscaleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio upscale image, got %d body=%s", upscaleResp.StatusCode, string(providerContractBody(t, upscaleResp)))
	}
	upscaleBody := providerContractJSONMap(t, upscaleResp)
	upscaleImage, ok := upscaleBody["generatedImage"].(map[string]any)
	if !ok {
		t.Fatalf("expected generatedImage object, got %#v", upscaleBody["generatedImage"])
	}
	if _, ok := upscaleImage["name"].(string); !ok {
		t.Fatalf("expected generatedImage.name string, got %#v", upscaleImage["name"])
	}
	if _, ok := upscaleImage["imageBytes"].(string); !ok {
		t.Fatalf("expected generatedImage.imageBytes string, got %#v", upscaleImage["imageBytes"])
	}

	healthResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456:generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456",
		"productInfo":{"productAttributes":{"title":"Red Dress","brand":"Stackyard"}}
	}`), headers)
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio text suggestions health response, got %d body=%s", healthResp.StatusCode, string(providerContractBody(t, healthResp)))
	}
}

func TestGCPShoppingMerchantProductstudioRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_productstudio/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_productstudio" {
		t.Fatalf("expected service=shopping_merchant_productstudio, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantProductstudioContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantProductstudioSuccess(t *testing.T, ts *httptest.Server, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, http.MethodPost, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant productstudio router for POST %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for POST %s to contain %q, got %s", path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
