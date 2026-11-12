package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
)

func TestAzureContainerAppsDataPlaneLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
	identifier := "session-a"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/metadata?api-version=2025-10-02-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	metadata := providerContractJSONMap(t, resp)
	if metadata["jupyterKernel"] != "python3" {
		t.Fatalf("unexpected metadata payload: %#v", metadata)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/containerapps-data-plane/2025-10-02-preview/executions?api-version=2025-10-02-preview&identifier="+identifier, []byte(`{
		"timeoutInSeconds":30,
		"code":"print('hello')",
		"codeInputType":"inline",
		"executionType":"synchronous"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for execute, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if resp.Header.Get("Operation-Location") == "" {
		t.Fatalf("expected Operation-Location header, got headers=%v", resp.Header)
	}
	execution := providerContractJSONMap(t, resp)
	executionID, _ := execution["id"].(string)
	if execution["status"] != "Succeeded" || executionID == "" {
		t.Fatalf("unexpected execution payload: %#v", execution)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/executions/"+executionID+"?api-version=2025-10-02-preview&identifier="+identifier, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for execution get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	gotExecution := providerContractJSONMap(t, resp)
	if gotExecution["id"] != executionID {
		t.Fatalf("unexpected execution get payload: %#v", gotExecution)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/session?api-version=2025-10-02-preview&identifier="+identifier, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for session get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	session := providerContractJSONMap(t, resp)
	if session["identifier"] != identifier || session["etag"] == "" {
		t.Fatalf("unexpected session payload: %#v", session)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/listSessions?api-version=2025-10-02-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for session list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listPayload := providerContractJSONMap(t, resp)
	sessions, ok := listPayload["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected one session in list, got %#v", listPayload)
	}

	uploadBody, contentType := azureContainerAppsDataPlaneMultipartBody(t, "hello.txt", "text/plain", []byte("hello from stackyard"))
	uploadHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              contentType,
	}
	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/containerapps-data-plane/2025-10-02-preview/files?api-version=2025-10-02-preview&identifier="+identifier+"&path=workspace", uploadBody, uploadHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for file upload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	filePayload := providerContractJSONMap(t, resp)
	if filePayload["name"] != "hello.txt" || filePayload["directory"] != "/workspace" {
		t.Fatalf("unexpected uploaded file payload: %#v", filePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/files?api-version=2025-10-02-preview&identifier="+identifier+"&path=workspace", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for file list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	filesList := providerContractJSONMap(t, resp)
	value, ok := filesList["value"].([]any)
	if !ok || len(value) != 1 {
		t.Fatalf("expected one file entry, got %#v", filesList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/files/hello.txt?api-version=2025-10-02-preview&identifier="+identifier+"&path=workspace", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for file get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	fileGet := providerContractJSONMap(t, resp)
	if fileGet["contentType"] != "text/plain" {
		t.Fatalf("unexpected file metadata payload: %#v", fileGet)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/containerapps-data-plane/2025-10-02-preview/files/hello.txt/content?api-version=2025-10-02-preview&identifier="+identifier+"&path=workspace", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for file content, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractBody(t, resp)
	if resp.Header.Get("Content-Type") != "text/plain" || string(body) != "hello from stackyard" {
		t.Fatalf("unexpected file content response: status=%d content-type=%q body=%q", resp.StatusCode, resp.Header.Get("Content-Type"), string(body))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/azure/containerapps-data-plane/2025-10-02-preview/files/hello.txt?api-version=2025-10-02-preview&identifier="+identifier+"&path=workspace", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for file delete, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/azure/containerapps-data-plane/2025-10-02-preview/session?api-version=2025-10-02-preview&identifier="+identifier, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for session delete, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureContainerAppsDataPlaneValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/metadata?api-version=",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing identifier",
			method:     http.MethodGet,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/session?api-version=2025-10-02-preview",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "execute missing timeout",
			method:     http.MethodPost,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/executions?api-version=2025-10-02-preview&identifier=session-a",
			body:       []byte(`{"code":"print('hi')"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "execute conflicting shell fields",
			method:     http.MethodPost,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/executions?api-version=2025-10-02-preview&identifier=session-a",
			body:       []byte(`{"timeoutInSeconds":30,"shellCommand":"ls","execCommandAndArgs":["echo","hi"]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "get missing session",
			method:     http.MethodGet,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/session?api-version=2025-10-02-preview&identifier=missing-session",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "SessionNotFound",
		},
		{
			name:       "get missing execution",
			method:     http.MethodGet,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/executions/missing-execution?api-version=2025-10-02-preview&identifier=session-a",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "ExecutionNotFound",
		},
		{
			name:       "list sessions invalid skip",
			method:     http.MethodGet,
			path:       "/azure/containerapps-data-plane/2025-10-02-preview/listSessions?api-version=2025-10-02-preview&skip=bad",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(body))
			}
			payload := mustDecodeJSONMap(t, body)
			errorPayload, ok := payload["error"].(map[string]any)
			if !ok || errorPayload["code"] != tt.wantCode {
				t.Fatalf("unexpected error payload: %#v", payload)
			}
		})
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/containerapps-data-plane/2025-10-02-preview/files?api-version=2025-10-02-preview&identifier=session-a", []byte("not-multipart"), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing multipart file upload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if errPayload, ok := payload["error"].(map[string]any); !ok || errPayload["code"] != "InvalidRequest" {
		t.Fatalf("unexpected multipart validation payload: %#v", payload)
	}
}

func azureContainerAppsDataPlaneMultipartBody(t *testing.T, filename, contentType string, content []byte) ([]byte, string) {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	partHeaders.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return buffer.Bytes(), writer.FormDataContentType()
}
