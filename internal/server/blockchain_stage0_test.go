package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func blockchainRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "managedblockchain-query")
}

func TestBlockchainStage0CatalogCoverage(t *testing.T) {
	if len(blockchainOperations) != 9 {
		t.Fatalf("expected 9 Managed Blockchain Query operations from docs, got %d", len(blockchainOperations))
	}
	if len(blockchainOperationByName) != len(blockchainOperations) {
		t.Fatalf("expected unique Managed Blockchain Query operation names")
	}

	requiredActions := []string{
		"BatchGetTokenBalance",
		"GetAssetContract",
		"GetTokenBalance",
		"GetTransaction",
		"ListAssetContracts",
		"ListFilteredTransactionEvents",
		"ListTokenBalances",
		"ListTransactionEvents",
		"ListTransactions",
	}
	for _, action := range requiredActions {
		if _, ok := blockchainOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(blockchainDataTypes) != 23 {
		t.Fatalf("expected 23 Managed Blockchain Query data types from docs, got %d", len(blockchainDataTypes))
	}
	if len(blockchainDataTypeByName) != len(blockchainDataTypes) {
		t.Fatalf("expected unique Managed Blockchain Query data type names")
	}

	requiredTypes := []string{
		"AssetContract",
		"TokenBalance",
		"Transaction",
		"TransactionEvent",
		"ContractIdentifier",
	}
	for _, typeName := range requiredTypes {
		if _, ok := blockchainDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBlockchainStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/managed-blockchain-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBlockchainStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/list-token-balances", []byte(`{"tokenFilter":{"network":"ETHEREUM_MAINNET"}}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "tokenBalances") {
		t.Fatalf("expected ListTokenBalances response body to include tokenBalances, got %q", body)
	}
}

func TestBlockchainStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range blockchainOperations {
		resp := blockchainRequest(t, ts, op.Method, blockchainNormalizePath(op.URI), []byte(`{}`))
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
