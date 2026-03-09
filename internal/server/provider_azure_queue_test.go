package server

import (
	"net/http"
	"testing"
)

func TestAzureQueueLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	authHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/azure/queue/devstoreaccount1/work-items", nil, authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating queue, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/work-items/messages", []byte("task-1"), authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 enqueue task-1, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/work-items/messages", []byte("task-2"), authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 enqueue task-2, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/work-items/messages/dequeue?numofmessages=1", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 dequeue one, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	dequeued := providerContractJSONMap(t, resp)
	items, ok := dequeued["messages"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one dequeued message, got %#v", dequeued["messages"])
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected message map, got %#v", items[0])
	}
	if first["messageText"] != "task-1" {
		t.Fatalf("expected first dequeued message task-1, got %#v", first["messageText"])
	}
	messageID, _ := first["messageId"].(string)
	popReceipt, _ := first["popReceipt"].(string)
	if messageID == "" || popReceipt == "" {
		t.Fatalf("expected messageId/popReceipt in dequeue response, got %#v", first)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/azure/queue/devstoreaccount1/work-items/messages/"+messageID+"?popreceipt="+popReceipt, nil, authHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting dequeued message, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/work-items/messages/dequeue?numofmessages=5", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 dequeue remaining messages, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	dequeued = providerContractJSONMap(t, resp)
	items, ok = dequeued["messages"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one remaining message, got %#v", dequeued["messages"])
	}
}

func TestAzureQueueValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	authHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/missing/messages", []byte("task"), authHeaders)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 enqueueing to missing queue, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/queue/devstoreaccount1/work-items", nil, authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating queue, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/queue/devstoreaccount1/work-items/messages/dequeue?numofmessages=bad", nil, authHeaders)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid numofmessages, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}
