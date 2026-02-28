package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestB2BIStage12CapabilityProfilePartnershipLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := b2biRequest(t, ts, "CreateCapability", `{"capabilityId":"cap-stage-001","name":"stage capability"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GetCapability", `{"capabilityId":"cap-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "UpdateCapability", `{"capabilityId":"cap-stage-001","name":"stage capability updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "ListCapabilities", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "CreateProfile", `{"profileId":"prof-stage-001","name":"stage profile"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GetProfile", `{"profileId":"prof-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "UpdateProfile", `{"profileId":"prof-stage-001","name":"stage profile updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "ListProfiles", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(
		t,
		ts,
		"CreatePartnership",
		`{"partnershipId":"part-stage-001","profileId":"prof-stage-001","capabilityId":"cap-stage-001","name":"stage partnership"}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GetPartnership", `{"partnershipId":"part-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "UpdatePartnership", `{"partnershipId":"part-stage-001","name":"stage partnership updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "ListPartnerships", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "DeletePartnership", `{"partnershipId":"part-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "DeleteProfile", `{"profileId":"prof-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "DeleteCapability", `{"capabilityId":"cap-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestB2BIStage34TransformerLifecycleAndMappingSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := b2biRequest(t, ts, "CreateTransformer", `{"transformerId":"trf-stage-001","name":"stage transformer"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GetTransformer", `{"transformerId":"trf-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "UpdateTransformer", `{"transformerId":"trf-stage-001","name":"stage transformer updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "ListTransformers", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "CreateStarterMappingTemplate", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GenerateMapping", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "TestParsing", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "TestConversion", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "TestMapping", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "StartTransformerJob", `{"transformerId":"trf-stage-001","transformerJobId":"job-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "transformerJob") {
		t.Fatalf("expected StartTransformerJob response to include transformerJob, got %q", body)
	}
	resp = b2biRequest(t, ts, "GetTransformerJob", `{"transformerJobId":"job-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "DeleteTransformer", `{"transformerId":"trf-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestB2BIStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:b2bi:us-east-1:123456789012:transformer/trf-000001"

	resp := b2biRequest(t, ts, "TagResource", `{"resourceArn":"`+resourceARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = b2biRequest(t, ts, "UntagResource", `{"resourceArn":"`+resourceARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = b2biRequest(t, ts, "CreateTransformer", `{"transformerId":"trf-idempotent-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "CreateTransformer", `{"transformerId":"trf-idempotent-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = b2biRequest(t, ts, "GetTransformer", `{"transformerId":"trf-idempotent-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = b2biRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "B2BI.ListCapabilities",
		},
		"b2bi",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
