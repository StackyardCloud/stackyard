package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVisionRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	operationPath := "/gcp/v1/projects/stackyard/locations/us-central1/operations/vision-op-1"

	batchAnnotateImagesPayload := []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}]
	}`)
	batchAnnotateFilesPayload := []byte(`{
		"requests":[{
			"inputConfig":{"gcsSource":{"uri":"gs://stackyard-inputs/file-1.pdf"},"mimeType":"application/pdf"},
			"features":[{"type":"DOCUMENT_TEXT_DETECTION"}]
		}]
	}`)
	createProductSetPayload := []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"productSetId":"product-set-1",
		"productSet":{"displayName":"Stackyard Product Set 1"}
	}`)

	assertGCPVisionSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPVisionSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)

	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", batchAnnotateImagesPayload, `"responses":[`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateFiles", batchAnnotateFilesPayload, `"totalPages":1`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/AsyncBatchAnnotateImages", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}],
		"outputConfig":{"gcsDestination":{"uri":"gs://stackyard-outputs/vision/"}}
	}`), `asyncBatchAnnotateImages.op-1`)

	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/CreateProductSet", createProductSetPayload, `product-set-1`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/ListProductSets", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"productSets":[`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/GetProductSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/productSets/product-set-1"
	}`), `product-set-1`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/UpdateProductSet", []byte(`{
		"productSet":{"name":"projects/stackyard/locations/us-central1/productSets/product-set-1","displayName":"Updated"},
		"updateMask":{"paths":["display_name"]}
	}`), `"displayName":"Updated"`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/DeleteProductSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/productSets/product-set-1"
	}`), `{}`)

	assertGCPVisionSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, `"operations":[`)
	assertGCPVisionSuccess(t, ts, http.MethodGet, operationPath, nil, `"name":"projects/stackyard/locations/us-central1/operations/vision-op-1"`)
	assertGCPVisionSuccess(t, ts, http.MethodPost, operationPath+":cancel", []byte(`{}`), `{}`)
	assertGCPVisionSuccess(t, ts, http.MethodDelete, operationPath, nil, `{}`)
}

func TestGCPVisionRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", []byte(`{"requests"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vision invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_V2ServiceHintRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision-v2-apiv1",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision v2-hinted batch annotate images, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"responses":[`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_V2UserAgentHintRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, map[string]string{
		"User-Agent": "cloud.google.com/go/vision/v2/apiv1",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision v2 user-agent operation list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"operations":[`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_BatchAnnotateImagesRequiresRequests(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vision batch annotate images missing requests, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_BatchAnnotateFilesRequiresGCSInput(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateFiles", []byte(`{
		"requests":[{
			"inputConfig":{"gcsSource":{"uri":"https://example.com/file.pdf"}},
			"features":[{"type":"DOCUMENT_TEXT_DETECTION"}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vision batch annotate files invalid input uri, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_GetProductSetMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/GetProductSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/productSets/missing-product-set"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vision get product set missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	}

	asyncResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/AsyncBatchAnnotateImages", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}],
		"outputConfig":{"gcsDestination":{"uri":"gs://stackyard-outputs/vision/"}}
	}`), headers)
	if asyncResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision async batch annotate images, got %d body=%s", asyncResp.StatusCode, string(providerContractBody(t, asyncResp)))
	}
	asyncBody := providerContractJSONMap(t, asyncResp)
	if _, ok := asyncBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", asyncBody["name"])
	}
	if _, ok := asyncBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", asyncBody["done"])
	}
	metadata, ok := asyncBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata object, got %#v", asyncBody["metadata"])
	}
	if _, ok := metadata["@type"].(string); !ok {
		t.Fatalf("expected metadata @type string, got %#v", metadata["@type"])
	}
	response, ok := asyncBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected response object, got %#v", asyncBody["response"])
	}
	if _, ok := response["@type"].(string); !ok {
		t.Fatalf("expected response @type string, got %#v", response["@type"])
	}

	listProductSetsResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/ListProductSets", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), headers)
	if listProductSetsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision list product sets, got %d body=%s", listProductSetsResp.StatusCode, string(providerContractBody(t, listProductSetsResp)))
	}
	listProductSetsBody := providerContractJSONMap(t, listProductSetsResp)
	productSets, ok := listProductSetsBody["productSets"].([]any)
	if !ok || len(productSets) == 0 {
		t.Fatalf("expected non-empty productSets array, got %#v", listProductSetsBody["productSets"])
	}
	firstProductSet, ok := productSets[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first product set object, got %#v", productSets[0])
	}
	if _, ok := firstProductSet["name"].(string); !ok {
		t.Fatalf("expected productSet.name string, got %#v", firstProductSet["name"])
	}
	if _, ok := firstProductSet["indexTime"].(string); !ok {
		t.Fatalf("expected productSet.indexTime string, got %#v", firstProductSet["indexTime"])
	}
}

func TestGCPVisionRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vision?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "vision" {
		t.Fatalf("expected service=vision, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVisionRouter_OutputShapeContractProbeV2Selector(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vision-v2-apiv1?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	gotService, _ := body["service"].(string)
	if gotService != "vision" && gotService != "vision-v2-apiv1" {
		t.Fatalf("expected service to be vision or vision-v2-apiv1, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
	if gotProvider, _ := body["provider"].(string); gotProvider != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
}

func TestGCPVisionRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vision?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vision",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vision contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPVisionContractServer(t *testing.T) *httptest.Server {
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

func assertGCPVisionSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vision",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
