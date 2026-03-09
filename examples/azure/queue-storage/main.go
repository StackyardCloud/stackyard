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
	account := getenv("STACKYARD_AZURE_QUEUE_ACCOUNT", "devstoreaccount1")
	queue := getenv("STACKYARD_AZURE_QUEUE_NAME", "work-items")

	fmt.Printf("Stackyard Azure Queue example using %s\n", endpoint)

	basePath := "/azure/queue/" + account + "/" + queue
	mustStatus(request(endpoint, http.MethodPut, basePath, nil), http.StatusCreated, "CreateQueue")
	mustStatus(request(endpoint, http.MethodPost, basePath+"/messages", []byte("task-1")), http.StatusCreated, "EnqueueTask1")
	mustStatus(request(endpoint, http.MethodPost, basePath+"/messages", []byte("task-2")), http.StatusCreated, "EnqueueTask2")

	dequeueResp := mustStatus(request(endpoint, http.MethodPost, basePath+"/messages/dequeue?numofmessages=1", nil), http.StatusOK, "Dequeue")
	body := string(readBody(dequeueResp))
	if !strings.Contains(body, "task-1") {
		exitf("Dequeue expected task-1, got %s", body)
	}
	if !strings.Contains(body, `"messageId":"1"`) {
		exitf("Dequeue missing messageId in payload: %s", body)
	}
	if !strings.Contains(body, `"popReceipt":"`) {
		exitf("Dequeue missing popReceipt in payload: %s", body)
	}
	popReceipt := extractJSONValue(body, "popReceipt")
	if popReceipt == "" {
		exitf("failed extracting popReceipt from dequeue payload: %s", body)
	}

	mustStatus(request(endpoint, http.MethodDelete, basePath+"/messages/1?popreceipt="+popReceipt, nil), http.StatusNoContent, "DeleteDequeuedMessage")
	fmt.Println("Done.")
}

func request(endpoint, method, path string, body []byte) *http.Response {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint+path, reader)
	if err != nil {
		exitf("new request %s %s failed: %v", method, path, err)
	}
	req.Header.Set("Authorization", "SharedKey devstoreaccount1:signature")
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

func extractJSONValue(body, key string) string {
	prefix := `"` + key + `":"`
	start := strings.Index(body, prefix)
	if start < 0 {
		return ""
	}
	start += len(prefix)
	rest := body[start:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
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
