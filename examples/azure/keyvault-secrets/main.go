package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	vault := getenv("STACKYARD_AZURE_KEYVAULT_NAME", "demo-vault")
	secret := getenv("STACKYARD_AZURE_KEYVAULT_SECRET_NAME", "api-token")

	fmt.Printf("Stackyard Azure Key Vault example using %s\n", endpoint)

	basePath := "/azure/keyvault/" + vault + "/secrets/" + secret
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	mustStatus(request(endpoint, http.MethodPut, basePath, []byte(`{"value":"token-v1"}`), headers), http.StatusOK, "SetSecretV1")
	mustStatus(request(endpoint, http.MethodPut, basePath, []byte(`{"value":"token-v2"}`), headers), http.StatusOK, "SetSecretV2")

	getResp := mustStatus(request(endpoint, http.MethodGet, basePath, nil, nil), http.StatusOK, "GetLatestSecret")
	latestBody := string(readBody(getResp))
	if !strings.Contains(latestBody, `"version":"v2"`) || !strings.Contains(latestBody, `"value":"token-v2"`) {
		exitf("GetLatestSecret response mismatch: %s", latestBody)
	}

	listResp := mustStatus(request(endpoint, http.MethodGet, basePath+"/versions", nil, nil), http.StatusOK, "ListSecretVersions")
	listBody := string(readBody(listResp))
	if !strings.Contains(listBody, `"version":"v1"`) || !strings.Contains(listBody, `"version":"v2"`) {
		exitf("ListSecretVersions response mismatch: %s", listBody)
	}

	versionResp := mustStatus(request(endpoint, http.MethodGet, basePath+"/versions/v1", nil, nil), http.StatusOK, "GetSecretV1")
	versionBody := string(readBody(versionResp))
	if !strings.Contains(versionBody, `"value":"token-v1"`) {
		exitf("GetSecretV1 response mismatch: %s", versionBody)
	}

	fmt.Println("Done.")
}

func request(endpoint, method, path string, body []byte, extraHeaders map[string]string) *http.Response {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint+path, reader)
	if err != nil {
		exitf("new request %s %s failed: %v", method, path, err)
	}
	req.Header.Set("Authorization", "SharedKey devstoreaccount1:signature")
	for key, value := range extraHeaders {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("request %s %s failed: %v", method, path, err)
	}
	return resp
}

func mustStatus(resp *http.Response, want int, name string) *http.Response {
	if resp.StatusCode != want {
		body := string(readBody(resp))
		exitf("%s expected status %d, got %d body=%s", name, want, resp.StatusCode, strings.TrimSpace(body))
	}
	return resp
}

func readBody(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read response body failed: %v", err)
	}
	_ = resp.Body.Close()
	return body
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
