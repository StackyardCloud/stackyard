package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureCustomVisionRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "get domains",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/domains?api-version=3.3",
		},
		{
			name:   "get domains by type",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/domains/classification?api-version=3.3",
		},
		{
			name:   "create project",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects?api-version=3.3",
			body:   []byte(`{"name":"stackyard-project","classificationType":"Multiclass","domainId":"b30a91ae-e3c1-4f73-a81e-c270bff27c39"}`),
		},
		{
			name:   "list projects",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects?api-version=3.3",
		},
		{
			name:   "import project",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/import?api-version=3.3",
			body:   []byte(`{"token":"stackyard-project-token","name":"imported-project"}`),
		},
		{
			name:   "get project",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects/project-a?api-version=3.3",
		},
		{
			name:   "update project",
			method: http.MethodPatch,
			path:   "/azure/customvision/v3.3/training/projects/project-a?api-version=3.3",
			body:   []byte(`{"description":"updated project"}`),
		},
		{
			name:   "delete project",
			method: http.MethodDelete,
			path:   "/azure/customvision/v3.3/training/projects/project-a?api-version=3.3",
		},
		{
			name:   "train project",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/train?api-version=3.3",
			body:   []byte(`{"trainingType":"Regular"}`),
		},
		{
			name:   "export project",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/export?api-version=3.3",
			body:   []byte(`{"platform":"TensorFlow","flavor":"TensorFlowNormal"}`),
		},
		{
			name:   "list project export formats",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects/project-a/export/formats?api-version=3.3",
		},
		{
			name:   "get domain suggestion",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects/project-a/domains/suggest?api-version=3.3",
		},
		{
			name:   "create images from files",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/images/files?api-version=3.3",
			body:   []byte(`{"images":[{"name":"shelf-1.jpg","contents":"aGVsbG8="}]}`),
		},
		{
			name:   "query images",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/images/query?api-version=3.3",
			body:   []byte(`{"take":50,"orderBy":"Newest"}`),
		},
		{
			name:   "get image region suggestions",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects/project-a/images/image-a/regions/suggested?api-version=3.3",
		},
		{
			name:   "list iterations",
			method: http.MethodGet,
			path:   "/azure/customvision/v3.3/training/projects/project-a/iterations?api-version=3.3",
		},
		{
			name:   "export iteration",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/iterations/iteration-a/export?api-version=3.3",
			body:   []byte(`{"platform":"CoreML","flavor":"compact"}`),
		},
		{
			name:   "publish iteration",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/iterations/iteration-a/publish?api-version=3.3",
			body:   []byte(`{"publishName":"stackyard-publish","predictionId":"stackyard-resource"}`),
		},
		{
			name:   "query prediction results",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/predictions/query?api-version=3.3",
			body:   []byte(`{"maxCount":10}`),
		},
		{
			name:   "delete prediction",
			method: http.MethodDelete,
			path:   "/azure/customvision/v3.3/training/projects/project-a/predictions/prediction-a?api-version=3.3",
		},
		{
			name:   "create tag",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/tags?api-version=3.3",
			body:   []byte(`{"name":"stackyard-tag"}`),
		},
		{
			name:   "update tag",
			method: http.MethodPatch,
			path:   "/azure/customvision/v3.3/training/projects/project-a/tags/tag-a?api-version=3.3",
			body:   []byte(`{"name":"stackyard-tag-updated"}`),
		},
		{
			name:   "quick test image",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/quicktest/image?api-version=3.3",
			body:   []byte(`{"imageData":"aGVsbG8="}`),
		},
		{
			name:   "quick test url",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/quicktest/url?api-version=3.3",
			body:   []byte(`{"url":"https://example.com/image.jpg"}`),
		},
		{
			name:   "quick test image by iteration",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/project-a/quicktest/iterations/iteration-a/image?api-version=3.3",
			body:   []byte(`{"imageData":"aGVsbG8="}`),
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

func TestAzureCustomVisionInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/customvision/v3.3/training/projects?api-version="
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/customvision/v3.3/training/projects" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureCustomVisionUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/customvision/v3.3/training/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown custom vision nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
