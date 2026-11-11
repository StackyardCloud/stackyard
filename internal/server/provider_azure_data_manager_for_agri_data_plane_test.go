package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestAzureDataManagerForAgriDataPlaneLifecycle(t *testing.T) {
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
		"Content-Type":              "application/json",
	}
	authHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}

	base := "/azure/data-manager-for-agri/dataplane/2023-11-01-preview"

	resp := providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Primary Party",
		"description":"Stackyard agriculture data-plane test"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating party, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	party := providerContractJSONMap(t, resp)
	if party["partyId"] != "party-a" || party["name"] != "Primary Party" {
		t.Fatalf("unexpected party payload: %#v", party)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/farms/farm-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Farm A",
		"status":"Active",
		"properties":{"acres":120}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating farm, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/farms?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing farms, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	farms := providerContractJSONMap(t, resp)
	if values, ok := farms["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected farm list payload, got %#v", farms)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/fields/field-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Field A",
		"status":"Active",
		"area":{"value":42,"unit":"acre"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating field, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/fields:search?api-version=2023-11-01-preview", []byte(`{
		"partyId":"party-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 searching fields, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	fieldSearch := providerContractJSONMap(t, resp)
	if values, ok := fieldSearch["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected field search results, got %#v", fieldSearch)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/sensor-partners/partner-a/devices/device-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Device A",
		"status":"Active",
		"deviceModelId":"device-model-a"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating device, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sensor-partners/partner-a/devices?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	attachmentBody, attachmentType := azureDataManagerForAgriDataPlaneMultipartBody(t, "field-notes.txt", "text/plain", []byte("field notes from stackyard"))
	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/attachments/attachment-a?api-version=2023-11-01-preview", attachmentBody, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              attachmentType,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating attachment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	attachment := providerContractJSONMap(t, resp)
	if attachment["fileName"] != "field-notes.txt" {
		t.Fatalf("unexpected attachment payload: %#v", attachment)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/parties/party-a/attachments/attachment-a/file?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 downloading attachment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if string(providerContractBody(t, resp)) != "field notes from stackyard" {
		t.Fatalf("unexpected attachment body")
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/models/model-a/resource-types/fields/resources/field-a/insights/insight-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Nitrogen alert",
		"status":"Active",
		"severity":"high"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating insight, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	insightAttachmentBody, insightAttachmentType := azureDataManagerForAgriDataPlaneMultipartBody(t, "insight.json", "application/json", []byte(`{"recommendation":"apply nitrogen"}`))
	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/models/model-a/resource-types/fields/resources/field-a/insight-attachments/insight-attachment-a?api-version=2023-11-01-preview", insightAttachmentBody, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              insightAttachmentType,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating insight attachment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/parties/party-a/models/model-a/resource-types/fields/resources/field-a/insight-attachments/insight-attachment-a/file?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 downloading insight attachment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/datasets/dataset-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Agronomy Dataset",
		"status":"Active"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating dataset, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/datasets/dataset-a/records/record-a?api-version=2023-11-01-preview", []byte(`{
		"name":"Record A",
		"partyId":"party-a",
		"status":"Active"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating dataset record, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/datasets/dataset-a/access:grant?api-version=2023-11-01-preview", []byte(`{
		"grantee":"grower@example.com",
		"permissions":["read","write"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 granting dataset access, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	grant := providerContractJSONMap(t, resp)
	if grant["datasetId"] != "dataset-a" {
		t.Fatalf("unexpected grant payload: %#v", grant)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/datasets/dataset-a/access?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing grants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	grants := providerContractJSONMap(t, resp)
	if values, ok := grants["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected at least one grant, got %#v", grants)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/weather-data:fetch?api-version=2023-11-01-preview", []byte(`{
		"partyId":"party-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 fetching weather data, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	weather := providerContractJSONMap(t, resp)
	if weather["status"] != "Succeeded" {
		t.Fatalf("unexpected weather payload: %#v", weather)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/oauth/tokens/:connect?api-version=2023-11-01-preview", []byte(`{
		"providerName":"demo-provider",
		"partyId":"party-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 connecting oauth token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	oauthConnect := providerContractJSONMap(t, resp)
	if oauthConnect["connectionLink"] == nil {
		t.Fatalf("unexpected oauth connect payload: %#v", oauthConnect)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/oauth/tokens?api-version=2023-11-01-preview", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing oauth tokens, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/solutions/crop-nitrogen:create?api-version=2023-11-01-preview", []byte(`{
		"partyId":"party-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating solution job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/solutions/crop-nitrogen:fetch?api-version=2023-11-01-preview", []byte(`{
		"partyId":"party-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 fetching solution output, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureDataManagerForAgriDataPlaneValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
	authHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}

	base := "/azure/data-manager-for-agri/dataplane/2023-11-01-preview"

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
			path:       base + "/parties/party-a?api-version=2023-01-01",
			headers:    authHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "empty create body",
			method:     http.MethodPatch,
			path:       base + "/parties/party-a?api-version=2023-11-01-preview",
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid search body",
			method:     http.MethodPost,
			path:       base + "/fields:search?api-version=2023-11-01-preview",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing resource",
			method:     http.MethodGet,
			path:       base + "/datasets/missing/records/record-a?api-version=2023-11-01-preview",
			headers:    authHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "ResourceNotFound",
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       base + "/unknown?api-version=2023-11-01-preview",
			headers:    authHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFound",
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
			errPayload, ok := payload["error"].(map[string]any)
			if !ok || errPayload["code"] != tt.wantCode {
				t.Fatalf("unexpected error payload: %#v", payload)
			}
		})
	}

	resp := providerContractRequest(t, ts, http.MethodPatch, base+"/parties/party-a/attachments/attachment-a?api-version=2023-11-01-preview", []byte("not-multipart"), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid attachment create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func azureDataManagerForAgriDataPlaneMultipartBody(t *testing.T, filename, contentType string, content []byte) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	_ = contentType
	return buf.Bytes(), writer.FormDataContentType()
}
