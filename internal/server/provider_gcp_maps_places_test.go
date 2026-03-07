package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsPlacesRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsPlacesContractServer(t)

	assertGCPMapsPlacesSuccess(t, ts, http.MethodPost, "/gcp/v1/places:searchText", []byte(`{"textQuery":"coffee","maxResultCount":1}`), "places")
	assertGCPMapsPlacesSuccess(t, ts, http.MethodPost, "/gcp/v1/places:searchNearby", []byte(`{
		"locationRestriction":{
			"circle":{
				"center":{"latitude":37.7937,"longitude":-122.3965},
				"radius":1000
			}
		}
	}`), "places")
	assertGCPMapsPlacesSuccess(t, ts, http.MethodPost, "/gcp/v1/places:autocomplete", []byte(`{"input":"cof"}`), "suggestions")
	assertGCPMapsPlacesSuccess(t, ts, http.MethodGet, "/gcp/v1/places/ChIJj61dQgK6j4AR4GeTYWZsKWw", nil, "places/ChIJj61dQgK6j4AR4GeTYWZsKWw")
	assertGCPMapsPlacesSuccess(t, ts, http.MethodGet, "/gcp/v1/places/ChIJj61dQgK6j4AR4GeTYWZsKWw/photos/AW123/media?maxWidthPx=400&skipHttpRedirect=true", nil, "photoUri")
}

func TestGCPMapsPlacesRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsPlacesContractServer(t)

	assertGCPMapsPlacesNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.places.v1.Places/SearchText", "Places/SearchText")
	assertGCPMapsPlacesNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.places.v1.Places/SearchNearby", "Places/SearchNearby")
	assertGCPMapsPlacesNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.places.v1.Places/GetPlace", "Places/GetPlace")
	assertGCPMapsPlacesNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.places.v1.Places/GetPhotoMedia", "Places/GetPhotoMedia")
	assertGCPMapsPlacesNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.places.v1.Places/AutocompletePlaces", "Places/AutocompletePlaces")
}

func TestGCPMapsPlacesRouter_SearchTextInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsPlacesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/places:searchText", []byte(`{"textQuery"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps places router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsPlacesRouter_SearchNearbyRequiresLocationRestriction(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsPlacesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/places:searchNearby", []byte(`{"includedTypes":["restaurant"]}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps places router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsPlacesRouter_GetPhotoMediaInvalidMaxWidth(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsPlacesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/places/ChIJj61dQgK6j4AR4GeTYWZsKWw/photos/AW123/media?maxWidthPx=invalid", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps places router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsPlacesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsPlacesNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps places router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMapsPlacesSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps places router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
