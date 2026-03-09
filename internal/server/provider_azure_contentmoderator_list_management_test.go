package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestAzureContentModeratorImageListManagementLifecycle(t *testing.T) {
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
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}
	createBody := []byte(`{"Name":"blocked-images","Description":"seed list","Metadata":{"owner":"security"}}`)

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists", createBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating image list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	listIDFloat, ok := created["Id"].(float64)
	if !ok || listIDFloat <= 0 {
		t.Fatalf("expected positive Id in create response, got %#v", created)
	}
	listID := int64(listIDFloat)
	if got := listMgmtString(created["Name"]); got != "blocked-images" {
		t.Fatalf("expected list name blocked-images, got %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing image lists, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	rows := providerContractJSONList(t, resp)
	if len(rows) == 0 {
		t.Fatalf("expected non-empty image list collection")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists/"+itoa(listID), nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting image list details, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	details := providerContractJSONMap(t, resp)
	if got := listMgmtString(details["Description"]); got != "seed list" {
		t.Fatalf("expected description seed list, got %#v", details)
	}

	updateBody := []byte(`{"Name":"blocked-images-v2","Description":"updated list","Metadata":{"owner":"platform"}}`)
	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/contentmoderator/lists/v1.0/imagelists/"+itoa(listID), updateBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 updating image list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	updated := providerContractJSONMap(t, resp)
	if got := listMgmtString(updated["Name"]); got != "blocked-images-v2" {
		t.Fatalf("expected updated name blocked-images-v2, got %#v", updated)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists/"+itoa(listID)+"/RefreshIndex", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 refreshing image list index, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	refresh := providerContractJSONMap(t, resp)
	if ok, _ := refresh["IsUpdateSuccess"].(bool); !ok {
		t.Fatalf("expected IsUpdateSuccess=true, got %#v", refresh)
	}
	if code := textModerationStatusCode(refresh); code != 3000 {
		t.Fatalf("expected refresh status code 3000, got %#v", refresh)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/azure/contentmoderator/lists/v1.0/imagelists/"+itoa(listID), nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting image list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists/"+itoa(listID), nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureContentModeratorImageListManagementValidation(t *testing.T) {
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
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists", []byte(`{"Name":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := providerContractJSONMap(t, resp); body["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error, got %#v", body)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists", []byte(`{"Description":"missing-name"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing Name, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists", []byte(`{"Name":"x"}`), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid content type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists/not-a-number", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid list id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, "/azure/contentmoderator/lists/v1.0/imagelists", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, "/azure/contentmoderator/lists/v1.0/imagelists") {
		t.Fatalf("unexpected not-implemented payload: %s", body)
	}
}

func providerContractJSONList(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	body := providerContractBody(t, resp)
	if len(body) == 0 {
		return nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal JSON list failed: %v body=%s", err, string(body))
	}
	return rows
}

func listMgmtString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
