package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3OutpostsStage6InvalidEnumsAndConstraints(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}

	cases := []struct {
		name string
		body string
	}{
		{"invalid-access-type", `{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","AccessType":"Public"}`},
		{"invalid-network-type", `{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","NetworkType":"IPV6"}`},
		{"customer-owned-missing-pool", `{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","AccessType":"CustomerOwnedIp"}`},
		{"customer-owned-invalid-network", `{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","AccessType":"CustomerOwnedIp","NetworkType":"IPV6","CustomerOwnedIpv4Pool":"coip-pool-12345678"}`},
		{"pool-with-private-access", `{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","AccessType":"Private","CustomerOwnedIpv4Pool":"coip-pool-12345678"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(tc.body), headers, "s3-outposts")
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d", tc.name, resp.StatusCode)
			}
		})
	}
}

func TestS3OutpostsStage6MissingRequiredFields(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-0123456789abcdef0"}`), headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing security group and subnet, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0"}`), headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing outpost id, got %d", resp.StatusCode)
	}
}
