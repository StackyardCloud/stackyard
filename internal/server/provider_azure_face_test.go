package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureFaceRoutesReturnFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "detect face",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detect?returnFaceId=true",
			body:   []byte("binary-image-data"),
		},
		{
			name:   "detect from url",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detect?_overload=detectFromUrl&returnFaceId=true",
			body:   []byte(`{"url":"https://example.com/face.jpg"}`),
		},
		{
			name:   "detect from session image id",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detect?_overload=detectFromSessionImageId&returnFaceId=true",
			body:   []byte(`{"sessionImageId":"session-image-1"}`),
		},
		{
			name:   "create face list",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/facelists/list-a",
			body:   []byte(`{"name":"stackyard-face-list","recognitionModel":"recognition_04"}`),
		},
		{
			name:   "add face list face",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/facelists/list-a/persistedfaces",
			body:   []byte("binary-image-data"),
		},
		{
			name:   "get face lists",
			method: http.MethodGet,
			path:   "/azure/face/v1.2/facelists?start=&top=10",
		},
		{
			name:   "create large face list",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/largefacelists/large-list-a",
			body:   []byte(`{"name":"stackyard-large-face-list","recognitionModel":"recognition_04"}`),
		},
		{
			name:   "train large face list",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/largefacelists/large-list-a/train",
		},
		{
			name:   "get large face list training status",
			method: http.MethodGet,
			path:   "/azure/face/v1.2/largefacelists/large-list-a/training",
		},
		{
			name:   "find similar",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/findsimilars",
			body:   []byte(`{"faceId":"face-a","faceIds":["face-b","face-c"],"maxNumOfCandidatesReturned":2}`),
		},
		{
			name:   "group faces",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/group",
			body:   []byte(`{"faceIds":["face-a","face-b","face-c"]}`),
		},
		{
			name:   "identify faces",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/identify",
			body:   []byte(`{"faceIds":["face-a"],"personGroupId":"pg-a","maxNumOfCandidatesReturned":1}`),
		},
		{
			name:   "verify faces",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/verify",
			body:   []byte(`{"faceId1":"face-a","faceId2":"face-b"}`),
		},
		{
			name:   "create liveness session",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detectLiveness-sessions",
			body:   []byte(`{"authTokenTimeToLiveInSeconds":300}`),
		},
		{
			name:   "get liveness session result",
			method: http.MethodGet,
			path:   "/azure/face/v1.2/detectLiveness-sessions/session-a",
		},
		{
			name:   "create liveness with verify session",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detectLivenessWithVerify-sessions",
			body:   []byte(`{"verifyImage":{"url":"https://example.com/verify.jpg"}}`),
		},
		{
			name:   "get session image",
			method: http.MethodGet,
			path:   "/azure/face/v1.2/sessionImages/session-image-1",
		},
		{
			name:   "create person group",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/persongroups/pg-a",
			body:   []byte(`{"name":"stackyard-person-group","recognitionModel":"recognition_04"}`),
		},
		{
			name:   "create person group person",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/persongroups/pg-a/persons",
			body:   []byte(`{"name":"Person A"}`),
		},
		{
			name:   "add person group person face",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/persongroups/pg-a/persons/person-a/persistedfaces",
			body:   []byte("binary-image-data"),
		},
		{
			name:   "train person group",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/persongroups/pg-a/train",
		},
		{
			name:   "get person group training status",
			method: http.MethodGet,
			path:   "/azure/face/v1.2/persongroups/pg-a/training",
		},
		{
			name:   "create large person group",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/largepersongroups/lpg-a",
			body:   []byte(`{"name":"stackyard-large-person-group","recognitionModel":"recognition_04"}`),
		},
		{
			name:   "create large person group person",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/largepersongroups/lpg-a/persons",
			body:   []byte(`{"name":"Large Person A"}`),
		},
		{
			name:   "train large person group",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/largepersongroups/lpg-a/train",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
			}
			if tt.body != nil {
				headers["Content-Type"] = "application/json"
			}

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s %s, got %d body=%s", tt.method, tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			if payload["status"] != "ok" {
				t.Fatalf("expected success payload, got %#v", payload)
			}
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider azure in payload, got %#v", payload)
			}

			expectedPath := tt.path
			if idx := strings.Index(expectedPath, "?"); idx >= 0 {
				expectedPath = expectedPath[:idx]
			}
			if payload["path"] != expectedPath {
				t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
			}
		})
	}
}

func TestAzureFaceInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/face/v1.2/detect?api-version="
	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte("binary-image-data"), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/octet-stream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/face/v1.2/detect" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureFaceUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/face/v1.2/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown face nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
