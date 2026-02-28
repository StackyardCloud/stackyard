package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage9CompatibilityActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	if len(rdsStage9CompatActions) == 0 {
		t.Fatalf("expected stage9 compat action list to be populated")
	}

	for _, action := range rdsStage9CompatActions {
		status, body := rdsRequest(t, ts, url.Values{
			"Action": []string{action},
		})
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", action, string(body))
		}
	}
}
