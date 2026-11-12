package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSpeakerRecognitionRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "create text dependent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles?api-version=2021-09-05",
			body:   []byte(`{"locale":"en-us"}`),
		},
		{
			name:   "list text dependent profiles",
			method: http.MethodGet,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles?api-version=2021-09-05",
		},
		{
			name:   "get text dependent profile",
			method: http.MethodGet,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles/profile-a?api-version=2021-09-05",
		},
		{
			name:   "create text dependent enrollment",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles/profile-a/enrollments?api-version=2021-09-05",
			body:   []byte("binary-audio-data"),
		},
		{
			name:   "reset text dependent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles/profile-a:reset?api-version=2021-09-05",
		},
		{
			name:   "verify text dependent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles/profile-a:verify?api-version=2021-09-05",
			body:   []byte("binary-audio-data"),
		},
		{
			name:   "list text dependent phrases",
			method: http.MethodGet,
			path:   "/azure/speaker-recognition/verification/text-dependent/phrases/en-us?api-version=2021-09-05",
		},
		{
			name:   "create text independent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/identification/text-independent/profiles?api-version=2021-09-05",
			body:   []byte(`{"locale":"en-us"}`),
		},
		{
			name:   "identify single speaker",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/identification/text-independent/profiles:identifySingleSpeaker?api-version=2021-09-05&profileIds=profile-a,profile-b",
			body:   []byte("binary-audio-data"),
		},
		{
			name:   "list text independent activation phrases",
			method: http.MethodGet,
			path:   "/azure/speaker-recognition/identification/text-independent/phrases/en-us?api-version=2021-09-05",
		},
		{
			name:   "create verification text independent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-independent/profiles?api-version=2021-09-05",
			body:   []byte(`{"locale":"en-us"}`),
		},
		{
			name:   "verify verification text independent profile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-independent/profiles/profile-a:verify?api-version=2021-09-05",
			body:   []byte("binary-audio-data"),
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
				contentType := "application/json"
				lowerPath := strings.ToLower(tt.path)
				if strings.Contains(lowerPath, "enrollments") || strings.Contains(lowerPath, ":verify") || strings.Contains(lowerPath, "identifysinglespeaker") {
					contentType = "audio/wav; codecs=audio/pcm"
				}
				headers["Content-Type"] = contentType
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

func TestAzureSpeakerRecognitionInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/speaker-recognition/verification/text-dependent/profiles?api-version="
	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte(`{"locale":"en-us"}`), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/speaker-recognition/verification/text-dependent/profiles" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureSpeakerRecognitionUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/speaker-recognition/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown speaker-recognition nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
