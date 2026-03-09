package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_STORAGE_ACCOUNT", "devstoreaccount1")
	container := getenv("STACKYARD_AZURE_STORAGE_CONTAINER", "ingress")
	blobPath := getenv("STACKYARD_AZURE_STORAGE_BLOB", "orders/2026-03-07.json")
	blobBody := []byte(`{"orderId":"o-1","status":"created"}`)

	fmt.Printf("Stackyard Azure Blob example using %s\n", endpoint)

	mustStatus(request(endpoint, http.MethodPut, "/azure/"+account+"/"+container+"?restype=container", nil, nil), http.StatusCreated, "CreateContainer")

	putHeaders := map[string]string{
		"Content-Type":  "application/json",
		"x-ms-meta-env": "demo",
	}
	putResp := mustStatus(request(endpoint, http.MethodPut, "/azure/"+account+"/"+container+"/"+blobPath, blobBody, putHeaders), http.StatusCreated, "PutBlob")
	etag := strings.TrimSpace(putResp.Header.Get("ETag"))
	if etag == "" {
		exitf("PutBlob missing ETag header")
	}

	getResp := mustStatus(request(endpoint, http.MethodGet, "/azure/"+account+"/"+container+"/"+blobPath, nil, nil), http.StatusOK, "GetBlob")
	if got := strings.TrimSpace(getResp.Header.Get("x-ms-meta-env")); got != "demo" {
		exitf("GetBlob metadata mismatch: %q", got)
	}
	if payload := string(readBody(getResp)); payload != string(blobBody) {
		exitf("GetBlob payload mismatch: %q", payload)
	}

	notModifiedHeaders := map[string]string{"If-None-Match": etag}
	mustStatus(request(endpoint, http.MethodGet, "/azure/"+account+"/"+container+"/"+blobPath, nil, notModifiedHeaders), http.StatusNotModified, "GetBlobIfNoneMatch")

	type containerEntry struct {
		Name string `xml:"Name"`
	}
	type listEnvelope struct {
		Containers struct {
			Container []containerEntry `xml:"Container"`
		} `xml:"Containers"`
	}

	listResp := mustStatus(request(endpoint, http.MethodGet, "/azure/"+account+"/?comp=list&maxresults=10", nil, nil), http.StatusOK, "ListContainers")
	var list listEnvelope
	if err := xml.Unmarshal(readBody(listResp), &list); err != nil {
		exitf("ListContainers XML parse failed: %v", err)
	}
	if len(list.Containers.Container) == 0 {
		exitf("ListContainers returned no containers")
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
