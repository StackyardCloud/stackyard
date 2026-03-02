package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStatePersistence_ReplaysJournalOnStartup(t *testing.T) {
	stateDir := t.TempDir()
	cfg := Config{
		Addr:                 "127.0.0.1:0",
		AccessKey:            testAccessKey,
		SecretKey:            testSecretKey,
		LogLevel:             "error",
		PersistenceEnabled:   true,
		StateDir:             stateDir,
		SnapshotLoadStrategy: "on_startup",
		SnapshotSaveStrategy: "on_request",
	}

	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	createQueueForPersistenceTest(t, ts.URL, "persisted")
	ts.Close()

	restarted := New(cfg)
	restartedTS := httptest.NewServer(restarted.Handler())
	defer restartedTS.Close()

	names := listQueueNamesForPersistenceTest(t, restartedTS.URL)
	if !containsNameForPersistenceTest(names, "persisted") {
		t.Fatalf("expected persisted queue after restart, got %#v", names)
	}
}

func TestStatePersistence_SnapshotLifecycle(t *testing.T) {
	stateDir := t.TempDir()
	cfg := Config{
		Addr:                 "127.0.0.1:0",
		AccessKey:            testAccessKey,
		SecretKey:            testSecretKey,
		LogLevel:             "error",
		PersistenceEnabled:   true,
		StateDir:             stateDir,
		SnapshotLoadStrategy: "on_startup",
		SnapshotSaveStrategy: "on_request",
	}

	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createQueueForPersistenceTest(t, ts.URL, "alpha")
	callStateEndpointForPersistenceTest(t, http.MethodPost, ts.URL+"/_stackyard/state/snapshots/base", http.StatusCreated)

	snapshots := listSnapshotsForPersistenceTest(t, ts.URL)
	if !containsNameForPersistenceTest(snapshots, "base") {
		t.Fatalf("expected snapshot list to include base, got %#v", snapshots)
	}

	createQueueForPersistenceTest(t, ts.URL, "beta")
	callStateEndpointForPersistenceTest(t, http.MethodPost, ts.URL+"/_stackyard/state/snapshots/base/restore", http.StatusOK)
	ts.Close()

	restarted := New(cfg)
	restartedTS := httptest.NewServer(restarted.Handler())
	defer restartedTS.Close()

	names := listQueueNamesForPersistenceTest(t, restartedTS.URL)
	if !containsNameForPersistenceTest(names, "alpha") {
		t.Fatalf("expected alpha queue after restore+restart, got %#v", names)
	}
	if containsNameForPersistenceTest(names, "beta") {
		t.Fatalf("expected beta queue to be removed by snapshot restore, got %#v", names)
	}

	callStateEndpointForPersistenceTest(t, http.MethodDelete, restartedTS.URL+"/_stackyard/state/snapshots/base", http.StatusOK)
	snapshots = listSnapshotsForPersistenceTest(t, restartedTS.URL)
	if containsNameForPersistenceTest(snapshots, "base") {
		t.Fatalf("expected snapshot base to be deleted, got %#v", snapshots)
	}
}

func createQueueForPersistenceTest(t *testing.T, baseURL, queue string) {
	t.Helper()
	endpoint := baseURL + "/sqs/queues/" + url.PathEscape(queue)
	resp := callStateEndpointForPersistenceTest(t, http.MethodPut, endpoint, http.StatusCreated)
	resp.Body.Close()
}

func listQueueNamesForPersistenceTest(t *testing.T, baseURL string) []string {
	t.Helper()
	resp := callStateEndpointForPersistenceTest(t, http.MethodGet, baseURL+"/sqs/queues", http.StatusOK)
	defer resp.Body.Close()

	var payload struct {
		Queues []struct {
			Name string `json:"name"`
		} `json:"queues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode queues response: %v", err)
	}
	out := make([]string, 0, len(payload.Queues))
	for _, q := range payload.Queues {
		out = append(out, q.Name)
	}
	return out
}

func listSnapshotsForPersistenceTest(t *testing.T, baseURL string) []string {
	t.Helper()
	resp := callStateEndpointForPersistenceTest(t, http.MethodGet, baseURL+"/_stackyard/state/snapshots", http.StatusOK)
	defer resp.Body.Close()

	var payload struct {
		Snapshots []string `json:"snapshots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode snapshots response: %v", err)
	}
	return payload.Snapshots
}

func callStateEndpointForPersistenceTest(t *testing.T, method, endpoint string, wantStatus int) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, endpoint, err)
	}
	if resp.StatusCode != wantStatus {
		defer resp.Body.Close()
		t.Fatalf("unexpected status for %s %s: got %d want %d", method, endpoint, resp.StatusCode, wantStatus)
	}
	return resp
}

func containsNameForPersistenceTest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
