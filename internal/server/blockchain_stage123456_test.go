package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBlockchainStage12TokenAndContractReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/get-token-balance", []byte(`{
		"tokenIdentifier":{"network":"ETHEREUM_MAINNET","contractAddress":"0x0000000000000000000000000000000000000000","tokenId":"1"},
		"ownerIdentifier":{"address":"0x1111111111111111111111111111111111111111"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "balance") {
		t.Fatalf("expected GetTokenBalance to include balance, got %q", body)
	}

	resp = blockchainRequest(t, ts, http.MethodPost, "/batch-get-token-balance", []byte(`{
		"getTokenBalanceInputs":[{"tokenIdentifier":{"network":"ETHEREUM_MAINNET","contractAddress":"0x0000000000000000000000000000000000000000","tokenId":"1"},"ownerIdentifier":{"address":"0x1111111111111111111111111111111111111111"}}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/get-asset-contract", []byte(`{
		"contractIdentifier":{"network":"ETHEREUM_MAINNET","contractAddress":"0x0000000000000000000000000000000000000000"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-asset-contracts", []byte(`{
		"contractFilter":{"network":"ETHEREUM_MAINNET","tokenStandard":"ERC20","deployerAddress":"0x1111111111111111111111111111111111111111"}
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestBlockchainStage34TransactionReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/get-transaction", []byte(`{
		"network":"ETHEREUM_MAINNET",
		"transactionHash":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-transactions", []byte(`{
		"address":"0x1111111111111111111111111111111111111111",
		"network":"ETHEREUM_MAINNET"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-transaction-events", []byte(`{
		"network":"ETHEREUM_MAINNET",
		"transactionHash":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-filtered-transaction-events", []byte(`{
		"network":"ETHEREUM_MAINNET",
		"addressIdentifierFilter":{"transactionEventToAddress":["0x1111111111111111111111111111111111111111"]}
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestBlockchainStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/list-token-balances", []byte(`{"tokenFilter":{"network":"ETHEREUM_MAINNET"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-token-balances", []byte(`{"tokenFilter":{"network":"ETHEREUM_MAINNET"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = blockchainRequest(t, ts, http.MethodPost, "/blockchain-unknown-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-token-balances", []byte(`{"tokenFilter":`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
