package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage8ReservedCapacityLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := rdsRequest(t, ts, url.Values{
		"Action": []string{"DescribeReservedDBInstancesOfferings"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe offerings 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ReservedDBInstancesOfferingId>offering-1yr-no-upfront-t3micro</ReservedDBInstancesOfferingId>")) {
		t.Fatalf("expected default offering in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                        []string{"PurchaseReservedDBInstancesOffering"},
		"ReservedDBInstancesOfferingId": []string{"offering-1yr-no-upfront-t3micro"},
		"ReservedDBInstanceId":          []string{"ri-stage8-1"},
		"DBInstanceCount":               []string{"2"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected purchase reserved instance 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ReservedDBInstanceId>ri-stage8-1</ReservedDBInstanceId>")) {
		t.Fatalf("expected purchased reserved instance id in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"DescribeReservedDBInstances"},
		"ReservedDBInstanceId": []string{"ri-stage8-1"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe reserved instances 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ReservedDBInstanceId>ri-stage8-1</ReservedDBInstanceId>")) {
		t.Fatalf("expected reserved instance in describe response: %s", string(body))
	}
}

func TestRDSStage8ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                        []string{"PurchaseReservedDBInstancesOffering"},
		"ReservedDBInstancesOfferingId": []string{"offering-1yr-no-upfront-t3micro"},
		"ReservedDBInstanceId":          []string{"ri-stage8-impl-1"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeReservedDBInstancesOfferings"}},
		{"Action": []string{"DescribeReservedDBInstances"}},
		{"Action": []string{"PurchaseReservedDBInstancesOffering"}, "ReservedDBInstancesOfferingId": []string{"offering-1yr-no-upfront-t3micro"}, "ReservedDBInstanceId": []string{"ri-stage8-impl-2"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
