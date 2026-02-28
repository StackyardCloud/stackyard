package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

type s3AccelerateConfigurationTest struct {
	XMLName xml.Name `xml:"AccelerateConfiguration"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	Status  string   `xml:"Status"`
}

type s3RequestPaymentConfigurationTest struct {
	XMLName xml.Name `xml:"RequestPaymentConfiguration"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	Payer   string   `xml:"Payer"`
}

type s3OwnershipControlsTest struct {
	XMLName xml.Name              `xml:"OwnershipControls"`
	Xmlns   string                `xml:"xmlns,attr,omitempty"`
	Rules   []s3OwnershipRuleTest `xml:"Rule"`
}

type s3OwnershipRuleTest struct {
	ObjectOwnership string `xml:"ObjectOwnership"`
}

type s3BucketAbacConfigurationTest struct {
	XMLName xml.Name `xml:"BucketAbacConfiguration"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	Status  string   `xml:"Status"`
}

func TestS3BucketConfigEndpoints(t *testing.T) {
	srv := New(Config{
		Addr:      ":0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "info",
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	bucket := "config-bucket"
	resp := signedRequest(t, http.MethodPut, ts.URL+"/"+bucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?accelerate", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	var accel s3AccelerateConfigurationTest
	if err := xml.Unmarshal(mustBody(t, resp), &accel); err != nil {
		t.Fatalf("unmarshal accelerate config: %v", err)
	}
	if accel.Status != "Suspended" {
		t.Fatalf("expected accelerate status Suspended, got %q", accel.Status)
	}

	accelBody := `<AccelerateConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></AccelerateConfiguration>`
	resp = signedRequest(t, http.MethodPut, ts.URL+"/"+bucket+"?accelerate", []byte(accelBody), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?accelerate", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if err := xml.Unmarshal(mustBody(t, resp), &accel); err != nil {
		t.Fatalf("unmarshal accelerate config: %v", err)
	}
	if accel.Status != "Enabled" {
		t.Fatalf("expected accelerate status Enabled, got %q", accel.Status)
	}

	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?requestPayment", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	var payment s3RequestPaymentConfigurationTest
	if err := xml.Unmarshal(mustBody(t, resp), &payment); err != nil {
		t.Fatalf("unmarshal request payment: %v", err)
	}
	if payment.Payer != "BucketOwner" {
		t.Fatalf("expected request payment BucketOwner, got %q", payment.Payer)
	}

	paymentBody := `<RequestPaymentConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Payer>Requester</Payer></RequestPaymentConfiguration>`
	resp = signedRequest(t, http.MethodPut, ts.URL+"/"+bucket+"?requestPayment", []byte(paymentBody), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?requestPayment", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if err := xml.Unmarshal(mustBody(t, resp), &payment); err != nil {
		t.Fatalf("unmarshal request payment: %v", err)
	}
	if payment.Payer != "Requester" {
		t.Fatalf("expected request payment Requester, got %q", payment.Payer)
	}

	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?ownershipControls", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected ownership controls 404, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "OwnershipControlsNotFoundError")

	ownershipBody := `<OwnershipControls xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule><ObjectOwnership>BucketOwnerEnforced</ObjectOwnership></Rule></OwnershipControls>`
	resp = signedRequest(t, http.MethodPut, ts.URL+"/"+bucket+"?ownershipControls", []byte(ownershipBody), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?ownershipControls", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	var ownership s3OwnershipControlsTest
	if err := xml.Unmarshal(mustBody(t, resp), &ownership); err != nil {
		t.Fatalf("unmarshal ownership controls: %v", err)
	}
	if len(ownership.Rules) != 1 || ownership.Rules[0].ObjectOwnership != "BucketOwnerEnforced" {
		t.Fatalf("unexpected ownership controls %+v", ownership.Rules)
	}

	resp = signedRequest(t, http.MethodDelete, ts.URL+"/"+bucket+"?ownershipControls", nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?ownershipControls", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected ownership controls 404 after delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "OwnershipControlsNotFoundError")

	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?abac", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	var abac s3BucketAbacConfigurationTest
	if err := xml.Unmarshal(mustBody(t, resp), &abac); err != nil {
		t.Fatalf("unmarshal abac config: %v", err)
	}
	if abac.Status != "Disabled" {
		t.Fatalf("expected abac status Disabled, got %q", abac.Status)
	}

	abacBody := `<BucketAbacConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></BucketAbacConfiguration>`
	resp = signedRequest(t, http.MethodPut, ts.URL+"/"+bucket+"?abac", []byte(abacBody), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/"+bucket+"?abac", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if err := xml.Unmarshal(mustBody(t, resp), &abac); err != nil {
		t.Fatalf("unmarshal abac config: %v", err)
	}
	if abac.Status != "Enabled" {
		t.Fatalf("expected abac status Enabled, got %q", abac.Status)
	}
}
