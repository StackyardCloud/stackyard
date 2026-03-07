package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPStreetViewPublishRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)

	createPhotoPayload := []byte(`{
		"photoId":{"id":"photo-1"},
		"uploadReference":{"uploadUrl":"https://streetviewpublish.googleapis.com/media/user/stackyard/photo/photo-1"},
		"places":[{"placeId":"ChIJj61dQgK6j4AR4GeTYWZsKWw"}],
		"pose":{"latLngPair":{"latitude":37.422,"longitude":-122.084},"heading":90}
	}`)
	updatePhotoPayload := []byte(`{
		"photoId":{"id":"photo-1"},
		"pose":{"heading":120,"latLngPair":{"latitude":37.422,"longitude":-122.084}}
	}`)
	batchUpdatePayload := []byte(`{
		"updatePhotoRequests":[
			{"photo":{"photoId":{"id":"photo-1"},"pose":{"heading":135,"latLngPair":{"latitude":37.422,"longitude":-122.084}}},"updateMask":"pose.heading"}
		]
	}`)
	batchDeletePayload := []byte(`{"photoIds":["photo-1","missing-photo"]}`)
	createPhotoSequencePayload := []byte(`{
		"id":"sequence-1",
		"uploadReference":{"uploadUrl":"https://streetviewpublish.googleapis.com/media/user/stackyard/photo/sequence-sequence-1"}
	}`)

	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photo:startUpload", []byte(`{}`), "uploadUrl")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photo", createPhotoPayload, "photo-1")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/photo/photo-1?view=INCLUDE_DOWNLOAD_URL", nil, "downloadUrl")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/photos:batchGet?photoIds=photo-1&view=BASIC", nil, "results")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/photos?view=BASIC&pageSize=1", nil, "photos")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPut, "/gcp/v1/photo/photo-1?updateMask=pose.heading", updatePhotoPayload, "photo-1")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photos:batchUpdate", batchUpdatePayload, "results")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodDelete, "/gcp/v1/photo/photo-1", nil, "{}")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photos:batchDelete", batchDeletePayload, "status")

	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photoSequence:startUpload", []byte(`{}`), "uploadUrl")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodPost, "/gcp/v1/photoSequence?inputType=VIDEO", createPhotoSequencePayload, "operations/photoSequence.sequence-1")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/photoSequence/sequence-1?filter=published_status%3DPUBLISHED", nil, "operations/photoSequence.sequence-1")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/photoSequences?pageSize=1", nil, "photoSequences")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, "operations")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodGet, "/gcp/v1/operations/photoSequence.sequence-1", nil, "photoSequence.sequence-1")
	assertGCPStreetViewPublishSuccess(t, ts, http.MethodDelete, "/gcp/v1/photoSequence/sequence-1", nil, "{}")
}

func TestGCPStreetViewPublishRouter_ListPhotosInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photos?view=BASIC&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish list photos, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_ListPhotosOversizedPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photos?view=BASIC&pageSize=5001", nil, map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish list photos, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_GetPhotoRequiresView(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photo/photo-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish get photo, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_CreatePhotoRequiresUploadReference(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photo", []byte(`{
		"photoId":{"id":"photo-1"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish create photo, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_UpdatePhotoInvalidMask(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPut, "/gcp/v1/photo/photo-1?updateMask=photoId", []byte(`{
		"photoId":{"id":"photo-1"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish update photo, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_BatchUpdatePhotosTooManyRequests(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photos:batchUpdate", []byte(`{
		"updatePhotoRequests":[
			{"photo":{"photoId":{"id":"photo-01"}}},
			{"photo":{"photoId":{"id":"photo-02"}}},
			{"photo":{"photoId":{"id":"photo-03"}}},
			{"photo":{"photoId":{"id":"photo-04"}}},
			{"photo":{"photoId":{"id":"photo-05"}}},
			{"photo":{"photoId":{"id":"photo-06"}}},
			{"photo":{"photoId":{"id":"photo-07"}}},
			{"photo":{"photoId":{"id":"photo-08"}}},
			{"photo":{"photoId":{"id":"photo-09"}}},
			{"photo":{"photoId":{"id":"photo-10"}}},
			{"photo":{"photoId":{"id":"photo-11"}}},
			{"photo":{"photoId":{"id":"photo-12"}}},
			{"photo":{"photoId":{"id":"photo-13"}}},
			{"photo":{"photoId":{"id":"photo-14"}}},
			{"photo":{"photoId":{"id":"photo-15"}}},
			{"photo":{"photoId":{"id":"photo-16"}}},
			{"photo":{"photoId":{"id":"photo-17"}}},
			{"photo":{"photoId":{"id":"photo-18"}}},
			{"photo":{"photoId":{"id":"photo-19"}}},
			{"photo":{"photoId":{"id":"photo-20"}}},
			{"photo":{"photoId":{"id":"photo-21"}}}
		]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish batch update photos, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_CreatePhotoSequenceInvalidInputType(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photoSequence?inputType=INPUT_TYPE_UNSPECIFIED", []byte(`{
		"id":"sequence-1",
		"uploadReference":{"uploadUrl":"https://streetviewpublish.googleapis.com/media/user/stackyard/photo/sequence-sequence-1"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish create photo sequence, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_DeletePhotoSequenceProcessingFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/photoSequence/sequence-processing", nil, map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp streetview_publish delete photo sequence, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp streetview_publish get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStreetViewPublishRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPStreetViewPublishContractServer(t)
	headers := map[string]string{"X-Stackyard-GCP-Service": "streetview_publish"}

	startUploadResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photo:startUpload", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if startUploadResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish start upload, got %d body=%s", startUploadResp.StatusCode, string(providerContractBody(t, startUploadResp)))
	}
	startUploadBody := providerContractJSONMap(t, startUploadResp)
	if _, ok := startUploadBody["uploadUrl"].(string); !ok {
		t.Fatalf("expected uploadUrl string, got %#v", startUploadBody["uploadUrl"])
	}

	photoResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photo/photo-1?view=INCLUDE_DOWNLOAD_URL", nil, headers)
	if photoResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish get photo, got %d body=%s", photoResp.StatusCode, string(providerContractBody(t, photoResp)))
	}
	photoBody := providerContractJSONMap(t, photoResp)
	if _, ok := photoBody["photoId"].(map[string]any); !ok {
		t.Fatalf("expected photoId object, got %#v", photoBody["photoId"])
	}
	if _, ok := photoBody["uploadReference"].(map[string]any); !ok {
		t.Fatalf("expected uploadReference object, got %#v", photoBody["uploadReference"])
	}
	if _, ok := photoBody["pose"].(map[string]any); !ok {
		t.Fatalf("expected pose object, got %#v", photoBody["pose"])
	}
	if _, ok := photoBody["places"].([]any); !ok {
		t.Fatalf("expected places array, got %#v", photoBody["places"])
	}

	batchGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photos:batchGet?photoIds=photo-1&photoIds=missing-photo&view=BASIC", nil, headers)
	if batchGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish batch get photos, got %d body=%s", batchGetResp.StatusCode, string(providerContractBody(t, batchGetResp)))
	}
	batchGetBody := providerContractJSONMap(t, batchGetResp)
	results, ok := batchGetBody["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected batch get results array, got %#v", batchGetBody["results"])
	}
	firstResult, _ := results[0].(map[string]any)
	if _, ok := firstResult["status"].(map[string]any); !ok {
		t.Fatalf("expected first result status object, got %#v", firstResult["status"])
	}

	listPhotosResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photos?view=BASIC&pageSize=1", nil, headers)
	if listPhotosResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish list photos, got %d body=%s", listPhotosResp.StatusCode, string(providerContractBody(t, listPhotosResp)))
	}
	listPhotosBody := providerContractJSONMap(t, listPhotosResp)
	if _, ok := listPhotosBody["photos"].([]any); !ok {
		t.Fatalf("expected photos array, got %#v", listPhotosBody["photos"])
	}
	if _, ok := listPhotosBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listPhotosBody["nextPageToken"])
	}

	batchUpdateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photos:batchUpdate", []byte(`{
		"updatePhotoRequests":[
			{"photo":{"photoId":{"id":"photo-1"},"pose":{"heading":135,"latLngPair":{"latitude":37.422,"longitude":-122.084}}},"updateMask":"pose.heading"}
		]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if batchUpdateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish batch update photos, got %d body=%s", batchUpdateResp.StatusCode, string(providerContractBody(t, batchUpdateResp)))
	}
	batchUpdateBody := providerContractJSONMap(t, batchUpdateResp)
	if _, ok := batchUpdateBody["results"].([]any); !ok {
		t.Fatalf("expected batch update results array, got %#v", batchUpdateBody["results"])
	}

	batchDeleteResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/photos:batchDelete", []byte(`{"photoIds":["photo-1","missing-photo"]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "streetview_publish",
	})
	if batchDeleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish batch delete photos, got %d body=%s", batchDeleteResp.StatusCode, string(providerContractBody(t, batchDeleteResp)))
	}
	batchDeleteBody := providerContractJSONMap(t, batchDeleteResp)
	if _, ok := batchDeleteBody["status"].([]any); !ok {
		t.Fatalf("expected batch delete status array, got %#v", batchDeleteBody["status"])
	}

	photoSequenceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photoSequence/sequence-1?filter=published_status%3DPUBLISHED", nil, headers)
	if photoSequenceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish get photo sequence, got %d body=%s", photoSequenceResp.StatusCode, string(providerContractBody(t, photoSequenceResp)))
	}
	photoSequenceBody := providerContractJSONMap(t, photoSequenceResp)
	if _, ok := photoSequenceBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", photoSequenceBody["name"])
	}
	if _, ok := photoSequenceBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", photoSequenceBody["done"])
	}
	if _, ok := photoSequenceBody["response"].(map[string]any); !ok {
		t.Fatalf("expected operation response object, got %#v", photoSequenceBody["response"])
	}

	listPhotoSequencesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photoSequences?pageSize=1", nil, headers)
	if listPhotoSequencesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish list photo sequences, got %d body=%s", listPhotoSequencesResp.StatusCode, string(providerContractBody(t, listPhotoSequencesResp)))
	}
	listPhotoSequencesBody := providerContractJSONMap(t, listPhotoSequencesResp)
	if _, ok := listPhotoSequencesBody["photoSequences"].([]any); !ok {
		t.Fatalf("expected photoSequences array, got %#v", listPhotoSequencesBody["photoSequences"])
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations/photoSequence.sequence-1", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
	}
	operationBody := providerContractJSONMap(t, operationResp)
	if _, ok := operationBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", operationBody["metadata"])
	}

	listOperationsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, headers)
	if listOperationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish list operations, got %d body=%s", listOperationsResp.StatusCode, string(providerContractBody(t, listOperationsResp)))
	}
	listOperationsBody := providerContractJSONMap(t, listOperationsResp)
	if _, ok := listOperationsBody["operations"].([]any); !ok {
		t.Fatalf("expected operations array, got %#v", listOperationsBody["operations"])
	}
	if _, ok := listOperationsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listOperationsBody["nextPageToken"])
	}
}

func TestGCPStreetViewPublishRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/streetview_publish?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "streetview_publish" {
		t.Fatalf("expected service=streetview_publish, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPStreetViewPublishContractServer(t *testing.T) *httptest.Server {
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

func assertGCPStreetViewPublishSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp streetview_publish router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
