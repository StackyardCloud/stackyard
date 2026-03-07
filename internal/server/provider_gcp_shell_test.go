package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShellRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	environmentName := "users/me/environments/default"
	newKey := "ssh-rsa c3RhY2t5YXJkLW5ldy1rZXk= stackyard@example.com"

	assertGCPShellSuccess(t, ts, http.MethodGet, "/gcp/v1/"+environmentName, nil, environmentName)
	assertGCPShellSuccess(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":start", []byte(`{
		"name":"users/me/environments/default",
		"publicKeys":["`+newKey+`"]
	}`), `"type.googleapis.com/google.cloud.shell.v1.StartEnvironmentResponse"`)
	assertGCPShellSuccess(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":authorize", []byte(`{
		"name":"users/me/environments/default",
		"accessToken":"ya29.stackyard",
		"idToken":"stackyard-id-token"
	}`), `"type.googleapis.com/google.cloud.shell.v1.AuthorizeEnvironmentResponse"`)
	assertGCPShellSuccess(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":addPublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"`+newKey+`"
	}`), `"type.googleapis.com/google.cloud.shell.v1.AddPublicKeyResponse"`)
	assertGCPShellSuccess(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":removePublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"`+newKey+`"
	}`), `"type.googleapis.com/google.cloud.shell.v1.RemovePublicKeyResponse"`)

	operationID := gcpShellOperationID("start", environmentName, "")
	assertGCPShellSuccess(t, ts, http.MethodGet, "/gcp/v1/operations/"+operationID, nil, `"done":true`)
}

func TestGCPShellRouter_StartEnvironmentRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:start", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shell start environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShellRouter_StartEnvironmentRejectsMismatchedName(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:start", []byte(`{
		"name":"users/me/environments/dev"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shell start environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShellRouter_AuthorizeRequiresToken(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:authorize", []byte(`{
		"name":"users/me/environments/default"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shell authorize environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShellRouter_AddPublicKeyRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:addPublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"not-a-public-key"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shell add public key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShellRouter_AddPublicKeyRejectsDuplicate(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:addPublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"`+gcpShellDefaultPublicKey()+`"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shell add public key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"AlreadyExists"`) {
		t.Fatalf("expected AlreadyExists error in response")
	}
}

func TestGCPShellRouter_RemovePublicKeyRejectsMissingKey(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/users/me/environments/default:removePublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"ssh-rsa bWlzc2luZw== stackyard@example.com"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shell remove public key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShellRouter_GetOperationRejectsInvalidID(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations/not-shell-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shell",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shell get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShellRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShellContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shell",
	}
	environmentName := "users/me/environments/default"
	newKey := "ssh-rsa c3RhY2t5YXJkLW5ldy1rZXk= stackyard@example.com"

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+environmentName, nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell get environment, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["name"].(string); !ok {
		t.Fatalf("expected environment name string, got %#v", getBody["name"])
	}
	if _, ok := getBody["id"].(string); !ok {
		t.Fatalf("expected environment id string, got %#v", getBody["id"])
	}
	if _, ok := getBody["dockerImage"].(string); !ok {
		t.Fatalf("expected environment dockerImage string, got %#v", getBody["dockerImage"])
	}
	if _, ok := getBody["state"].(string); !ok {
		t.Fatalf("expected environment state string, got %#v", getBody["state"])
	}
	if _, ok := getBody["sshPort"].(float64); !ok {
		t.Fatalf("expected environment sshPort number, got %#v", getBody["sshPort"])
	}
	if _, ok := getBody["publicKeys"].([]any); !ok {
		t.Fatalf("expected environment publicKeys array, got %#v", getBody["publicKeys"])
	}

	startResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":start", []byte(`{
		"name":"users/me/environments/default",
		"publicKeys":["`+newKey+`"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if startResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell start environment, got %d body=%s", startResp.StatusCode, string(providerContractBody(t, startResp)))
	}
	startBody := providerContractJSONMap(t, startResp)
	startOperationName, _ := startBody["name"].(string)
	if startOperationName == "" {
		t.Fatalf("expected operation name string, got %#v", startBody["name"])
	}
	if _, ok := startBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", startBody["done"])
	}
	startMetadata, ok := startBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", startBody["metadata"])
	}
	if _, ok := startMetadata["@type"].(string); !ok {
		t.Fatalf("expected metadata @type string, got %#v", startMetadata["@type"])
	}
	startResponse, ok := startBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation response object, got %#v", startBody["response"])
	}
	if _, ok := startResponse["@type"].(string); !ok {
		t.Fatalf("expected response @type string, got %#v", startResponse["@type"])
	}
	startEnvironment, ok := startResponse["environment"].(map[string]any)
	if !ok {
		t.Fatalf("expected response environment object, got %#v", startResponse["environment"])
	}
	if _, ok := startEnvironment["name"].(string); !ok {
		t.Fatalf("expected response environment name string, got %#v", startEnvironment["name"])
	}

	addResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":addPublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"`+newKey+`"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if addResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell add public key, got %d body=%s", addResp.StatusCode, string(providerContractBody(t, addResp)))
	}
	addBody := providerContractJSONMap(t, addResp)
	addResponse, ok := addBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected add response object, got %#v", addBody["response"])
	}
	if _, ok := addResponse["key"].(string); !ok {
		t.Fatalf("expected add response key string, got %#v", addResponse["key"])
	}

	authResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":authorize", []byte(`{
		"name":"users/me/environments/default",
		"accessToken":"ya29.stackyard"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell authorize environment, got %d body=%s", authResp.StatusCode, string(providerContractBody(t, authResp)))
	}
	authBody := providerContractJSONMap(t, authResp)
	authResponse, ok := authBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected authorize response object, got %#v", authBody["response"])
	}
	if _, ok := authResponse["@type"].(string); !ok {
		t.Fatalf("expected authorize response @type string, got %#v", authResponse["@type"])
	}

	operationPath := "/gcp/v1/" + startOperationName
	pollResp := providerContractRequest(t, ts, http.MethodGet, operationPath, nil, headers)
	if pollResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell get operation, got %d body=%s", pollResp.StatusCode, string(providerContractBody(t, pollResp)))
	}
	pollBody := providerContractJSONMap(t, pollResp)
	if gotName, _ := pollBody["name"].(string); gotName != startOperationName {
		t.Fatalf("expected polled operation name %q, got %#v", startOperationName, pollBody["name"])
	}
	if _, ok := pollBody["done"].(bool); !ok {
		t.Fatalf("expected polled operation done bool, got %#v", pollBody["done"])
	}
}

func TestGCPShellRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/users/me/environments/default?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shell" {
		t.Fatalf("expected service=shell, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShellContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShellSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shell",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shell router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
