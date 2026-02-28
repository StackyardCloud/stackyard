package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestXRayStage12GroupAndSamplingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := xrayRequest(t, ts, http.MethodPost, "/CreateGroup", `{"GroupName":"stage-xray-group","FilterExpression":"service(\"stage\")"}`)
	assertStatus(t, resp, http.StatusOK)
	groupPayload := decodeXRayPayload(t, resp)
	if !strings.Contains(xrayPayloadStringValue(groupPayload, "Group"), "stage-xray-group") {
		body, _ := json.Marshal(groupPayload)
		t.Fatalf("expected CreateGroup response to include stage-xray-group, got %s", string(body))
	}

	resp = xrayRequest(t, ts, http.MethodPost, "/GetGroup", `{"GroupName":"stage-xray-group"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/Groups", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/UpdateGroup", `{"GroupName":"stage-xray-group","FilterExpression":"service(\"stage-updated\")"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/CreateSamplingRule", `{"SamplingRule":{"RuleName":"stage-xray-rule","ResourceARN":"*","Priority":1,"FixedRate":0.05,"ReservoirSize":1,"ServiceName":"*","ServiceType":"*","Host":"*","HTTPMethod":"*","URLPath":"*","Version":1}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/GetSamplingRules", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/UpdateSamplingRule", `{"SamplingRuleUpdate":{"RuleName":"stage-xray-rule","FixedRate":0.1}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/SamplingStatisticSummaries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/SamplingTargets", `{"SamplingStatisticsDocuments":[{"RuleName":"stage-xray-rule","ClientID":"stage-client","Timestamp":1700000000,"RequestCount":1,"BorrowCount":0,"SampledCount":1}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/DeleteSamplingRule", `{"RuleName":"stage-xray-rule"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/DeleteGroup", `{"GroupName":"stage-xray-group"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestXRayStage34TraceRetrievalAndInsights(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	now := time.Now().UTC().Unix()
	start := now - 900
	traceID := "1-58406520-a006649127e371903a2de979"

	resp := xrayRequest(t, ts, http.MethodPost, "/TraceSegments", `{"TraceSegmentDocuments":["{\"trace_id\":\"1-58406520-a006649127e371903a2de979\",\"id\":\"stage-segment-000001\",\"name\":\"stage-xray-service\",\"start_time\":1700000000,\"end_time\":1700000001}"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/TelemetryRecords", `{"TelemetryRecords":[{"Timestamp":1700000000,"SegmentsReceivedCount":1}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/Traces", `{"TraceIds":["`+traceID+`"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/TraceGraph", `{"TraceIds":["`+traceID+`"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/ServiceGraph", `{"StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/TimeSeriesServiceStatistics", `{"StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/TraceSummaries", `{"StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/StartTraceRetrieval", `{"TraceIds":["`+traceID+`"],"StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeXRayPayload(t, resp)
	token := xrayPayloadStringValue(startPayload, "RetrievalToken")
	if strings.TrimSpace(token) == "" {
		t.Fatalf("expected StartTraceRetrieval to return RetrievalToken")
	}

	resp = xrayRequest(t, ts, http.MethodPost, "/ListRetrievedTraces", `{"RetrievalToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/GetRetrievedTracesGraph", `{"RetrievalToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/CancelTraceRetrieval", `{"RetrievalToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/InsightSummaries", `{"StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/Insight", `{"InsightId":"insight-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/InsightEvents", `{"InsightId":"insight-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/InsightImpactGraph", `{"InsightId":"insight-000001","StartTime":`+intString(start)+`,"EndTime":`+intString(now)+`}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestXRayStage56ConfigTaggingAndValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:xray:us-east-1:123456789012:group/stackyard-group"

	resp := xrayRequest(t, ts, http.MethodPost, "/EncryptionConfig", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/PutEncryptionConfig", `{"Type":"NONE"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/GetTraceSegmentDestination", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/UpdateTraceSegmentDestination", `{"Destination":"XRay"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/GetIndexingRules", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/UpdateIndexingRule", `{"Name":"Default","Rule":{"Probabilistic":{"DesiredSamplingPercentage":10}}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/PutResourcePolicy", `{"PolicyName":"stage-policy","PolicyDocument":"{}"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/ListResourcePolicies", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/DeleteResourcePolicy", `{"PolicyName":"stage-policy"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/TagResource", `{"ResourceARN":"`+resourceARN+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = xrayRequest(t, ts, http.MethodPost, "/ListTagsForResource", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = xrayRequest(t, ts, http.MethodPost, "/UntagResource", `{"ResourceARN":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = xrayRequest(t, ts, http.MethodPost, "/xray/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/Groups",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"xray",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeXRayPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func xrayPayloadStringValue(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		marshaled, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(marshaled)
	default:
		return ""
	}
}

func intString(v int64) string {
	return strconv.FormatInt(v, 10)
}
