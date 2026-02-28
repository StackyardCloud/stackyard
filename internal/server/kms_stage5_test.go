package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKMSStage5CustomKeyStoreLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kmsRequest(t, ts, "CreateCustomKeyStore", mustJSON(t, map[string]any{
		"CustomKeyStoreName": "stage5-store",
		"CloudHsmClusterId":  "cluster-1234567890abcdef0",
	}))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CustomKeyStoreID string `json:"CustomKeyStoreId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create custom key store response: %v", err)
	}
	if createOut.CustomKeyStoreID == "" {
		t.Fatalf("expected custom key store id")
	}

	resp = kmsRequest(t, ts, "DescribeCustomKeyStores", mustJSON(t, map[string]any{
		"CustomKeyStoreId": createOut.CustomKeyStoreID,
		"Limit":            10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		CustomKeyStores []struct {
			CustomKeyStoreID string `json:"CustomKeyStoreId"`
			ConnectionState  string `json:"ConnectionState"`
		} `json:"CustomKeyStores"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe custom key stores response: %v", err)
	}
	if len(describeOut.CustomKeyStores) == 0 || describeOut.CustomKeyStores[0].ConnectionState != "DISCONNECTED" {
		t.Fatalf("expected disconnected custom key store after create")
	}

	resp = kmsRequest(t, ts, "ConnectCustomKeyStore", mustJSON(t, map[string]any{
		"CustomKeyStoreId": createOut.CustomKeyStoreID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "UpdateCustomKeyStore", mustJSON(t, map[string]any{
		"CustomKeyStoreId":      createOut.CustomKeyStoreID,
		"NewCustomKeyStoreName": "stage5-store-updated",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "DisconnectCustomKeyStore", mustJSON(t, map[string]any{
		"CustomKeyStoreId": createOut.CustomKeyStoreID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "DeleteCustomKeyStore", mustJSON(t, map[string]any{
		"CustomKeyStoreId": createOut.CustomKeyStoreID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "DescribeCustomKeyStores", mustJSON(t, map[string]any{
		"CustomKeyStoreId": createOut.CustomKeyStoreID,
		"Limit":            10,
	}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe custom key stores response after delete: %v", err)
	}
	if len(describeOut.CustomKeyStores) != 0 {
		t.Fatalf("expected custom key store to be deleted")
	}
}
