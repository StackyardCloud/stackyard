package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func rdsRequest(t *testing.T, ts *httptest.Server, params url.Values) (int, []byte) {
	t.Helper()
	if params.Get("Version") == "" {
		params.Set("Version", "2014-10-31")
	}
	body := []byte(params.Encode())
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "rds")
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp.StatusCode, respBody
}

func TestRDSStage1DBInstanceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
		"Engine":               []string{"postgres"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret123!"},
	}
	status, body := rdsRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceIdentifier>rds-stage1-db</DBInstanceIdentifier>")) {
		t.Fatalf("missing created DB instance: %s", string(body))
	}

	describe := url.Values{"Action": []string{"DescribeDBInstances"}}
	status, body = rdsRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected describe 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstances>")) {
		t.Fatalf("expected DBInstances list: %s", string(body))
	}

	modify := url.Values{
		"Action":               []string{"ModifyDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
		"DBInstanceClass":      []string{"db.t3.small"},
	}
	status, body = rdsRequest(t, ts, modify)
	if status != http.StatusOK {
		t.Fatalf("expected modify 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceClass>db.t3.small</DBInstanceClass>")) {
		t.Fatalf("expected updated class in modify response: %s", string(body))
	}

	stop := url.Values{
		"Action":               []string{"StopDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
	}
	status, body = rdsRequest(t, ts, stop)
	if status != http.StatusOK {
		t.Fatalf("expected stop 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceStatus>stopped</DBInstanceStatus>")) {
		t.Fatalf("expected stopped status: %s", string(body))
	}

	start := url.Values{
		"Action":               []string{"StartDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
	}
	status, body = rdsRequest(t, ts, start)
	if status != http.StatusOK {
		t.Fatalf("expected start 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceStatus>available</DBInstanceStatus>")) {
		t.Fatalf("expected available status after start: %s", string(body))
	}

	reboot := url.Values{
		"Action":               []string{"RebootDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
	}
	status, body = rdsRequest(t, ts, reboot)
	if status != http.StatusOK {
		t.Fatalf("expected reboot 200, got %d: %s", status, string(body))
	}

	deleteReq := url.Values{
		"Action":               []string{"DeleteDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
		"SkipFinalSnapshot":    []string{"true"},
	}
	status, body = rdsRequest(t, ts, deleteReq)
	if status != http.StatusOK {
		t.Fatalf("expected delete 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceStatus>deleting</DBInstanceStatus>")) {
		t.Fatalf("expected deleting status: %s", string(body))
	}

	describeOne := url.Values{
		"Action":               []string{"DescribeDBInstances"},
		"DBInstanceIdentifier": []string{"rds-stage1-db"},
	}
	status, body = rdsRequest(t, ts, describeOne)
	if status != http.StatusNotFound {
		t.Fatalf("expected describe deleted instance 404, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("DBInstanceNotFound")) {
		t.Fatalf("expected DBInstanceNotFound after delete: %s", string(body))
	}
}

func TestRDSStage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage1-impl"},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret123!"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeDBInstances"}},
		{"Action": []string{"ModifyDBInstance"}, "DBInstanceIdentifier": []string{"rds-stage1-impl"}, "DBInstanceClass": []string{"db.t3.small"}},
		{"Action": []string{"StopDBInstance"}, "DBInstanceIdentifier": []string{"rds-stage1-impl"}},
		{"Action": []string{"StartDBInstance"}, "DBInstanceIdentifier": []string{"rds-stage1-impl"}},
		{"Action": []string{"RebootDBInstance"}, "DBInstanceIdentifier": []string{"rds-stage1-impl"}},
		{"Action": []string{"DeleteDBInstance"}, "DBInstanceIdentifier": []string{"rds-stage1-impl"}, "SkipFinalSnapshot": []string{"true"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
