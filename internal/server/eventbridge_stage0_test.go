package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func eventBridgeRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "AWSEvents." + action,
	}
	return signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", body, headers, "events", testRegion, "")
}

func TestEventBridgeStage0CoreFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := eventBridgeRequest(t, ts, "CreateEventBus", []byte(`{"Name":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
	var createBus struct {
		EventBusArn string `json:"EventBusArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createBus); err != nil {
		t.Fatalf("unmarshal create bus: %v", err)
	}
	if createBus.EventBusArn == "" {
		t.Fatalf("expected EventBusArn")
	}

	resp = eventBridgeRequest(t, ts, "DescribeEventBus", []byte(`{"Name":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeBus struct {
		Name             string `json:"Name"`
		Arn              string `json:"Arn"`
		Description      string `json:"Description"`
		KmsKeyIdentifier string `json:"KmsKeyIdentifier"`
		DeadLetterConfig struct {
			Arn string `json:"Arn"`
		} `json:"DeadLetterConfig"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeBus); err != nil {
		t.Fatalf("unmarshal describe bus: %v", err)
	}
	if describeBus.Name != "demo-bus" {
		t.Fatalf("expected bus name demo-bus")
	}

	resp = eventBridgeRequest(t, ts, "UpdateEventBus", []byte(`{
		"Name":"demo-bus",
		"Description":"updated bus",
		"KmsKeyIdentifier":"alias/aws/events",
		"DeadLetterConfig":{"Arn":"arn:aws:sqs:us-east-1:123456789012:dead-letter"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var updateBus struct {
		Arn              string `json:"Arn"`
		Name             string `json:"Name"`
		Description      string `json:"Description"`
		KmsKeyIdentifier string `json:"KmsKeyIdentifier"`
		DeadLetterConfig struct {
			Arn string `json:"Arn"`
		} `json:"DeadLetterConfig"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateBus); err != nil {
		t.Fatalf("unmarshal update bus: %v", err)
	}
	if updateBus.Name != "demo-bus" || updateBus.Arn == "" {
		t.Fatalf("expected updated bus identity")
	}
	if updateBus.Description != "updated bus" {
		t.Fatalf("expected updated description")
	}
	if updateBus.KmsKeyIdentifier != "alias/aws/events" {
		t.Fatalf("expected updated kms key identifier")
	}
	if updateBus.DeadLetterConfig.Arn != "arn:aws:sqs:us-east-1:123456789012:dead-letter" {
		t.Fatalf("expected updated dead-letter config arn")
	}

	resp = eventBridgeRequest(t, ts, "DescribeEventBus", []byte(`{"Name":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &describeBus); err != nil {
		t.Fatalf("unmarshal describe updated bus: %v", err)
	}
	if describeBus.Description != "updated bus" {
		t.Fatalf("expected updated description in DescribeEventBus")
	}
	if describeBus.KmsKeyIdentifier != "alias/aws/events" {
		t.Fatalf("expected updated kms key identifier in DescribeEventBus")
	}
	if describeBus.DeadLetterConfig.Arn != "arn:aws:sqs:us-east-1:123456789012:dead-letter" {
		t.Fatalf("expected updated dead-letter config arn in DescribeEventBus")
	}

	resp = eventBridgeRequest(t, ts, "ListEventBuses", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var listBus struct {
		EventBuses []struct {
			Name string `json:"Name"`
		} `json:"EventBuses"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listBus); err != nil {
		t.Fatalf("unmarshal list buses: %v", err)
	}
	found := false
	for _, bus := range listBus.EventBuses {
		if bus.Name == "demo-bus" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected demo-bus in list")
	}

	resp = eventBridgeRequest(t, ts, "PutRule", []byte(`{"Name":"demo-rule","EventBusName":"demo-bus","EventPattern":"{\"source\":[\"demo\"]}"}`))
	assertStatus(t, resp, http.StatusOK)
	var putRule struct {
		RuleArn string `json:"RuleArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putRule); err != nil {
		t.Fatalf("unmarshal put rule: %v", err)
	}
	if putRule.RuleArn == "" {
		t.Fatalf("expected RuleArn")
	}

	resp = eventBridgeRequest(t, ts, "DescribeRule", []byte(`{"Name":"demo-rule","EventBusName":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeRule struct {
		Name  string `json:"Name"`
		State string `json:"State"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeRule); err != nil {
		t.Fatalf("unmarshal describe rule: %v", err)
	}
	if describeRule.Name != "demo-rule" {
		t.Fatalf("expected rule name demo-rule")
	}

	resp = eventBridgeRequest(t, ts, "PutTargets", []byte(`{"Rule":"demo-rule","EventBusName":"demo-bus","Targets":[{"Id":"t1","Arn":"arn:aws:lambda:us-east-1:123456789012:function:demo"}]}`))
	assertStatus(t, resp, http.StatusOK)
	var putTargets struct {
		FailedEntryCount int `json:"FailedEntryCount"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putTargets); err != nil {
		t.Fatalf("unmarshal put targets: %v", err)
	}
	if putTargets.FailedEntryCount != 0 {
		t.Fatalf("expected 0 failed entries")
	}

	resp = eventBridgeRequest(t, ts, "ListTargetsByRule", []byte(`{"Rule":"demo-rule","EventBusName":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
	var listTargets struct {
		Targets []struct {
			Id string `json:"Id"`
		} `json:"Targets"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTargets); err != nil {
		t.Fatalf("unmarshal list targets: %v", err)
	}
	if len(listTargets.Targets) != 1 || listTargets.Targets[0].Id != "t1" {
		t.Fatalf("expected 1 target")
	}

	resp = eventBridgeRequest(t, ts, "ListRuleNamesByTarget", []byte(`{"EventBusName":"demo-bus","TargetArn":"arn:aws:lambda:us-east-1:123456789012:function:demo"}`))
	assertStatus(t, resp, http.StatusOK)
	var listRuleNames struct {
		RuleNames []string `json:"RuleNames"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listRuleNames); err != nil {
		t.Fatalf("unmarshal list rule names: %v", err)
	}
	if len(listRuleNames.RuleNames) != 1 || listRuleNames.RuleNames[0] != "demo-rule" {
		t.Fatalf("expected demo-rule in rule names")
	}

	resp = eventBridgeRequest(t, ts, "TagResource", []byte(`{"ResourceARN":"`+createBus.EventBusArn+`","Tags":[{"Key":"env","Value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eventBridgeRequest(t, ts, "ListTagsForResource", []byte(`{"ResourceARN":"`+createBus.EventBusArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var listTags struct {
		Tags []struct {
			Key string `json:"Key"`
		} `json:"Tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTags); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if len(listTags.Tags) != 1 || listTags.Tags[0].Key != "env" {
		t.Fatalf("expected env tag")
	}

	resp = eventBridgeRequest(t, ts, "UntagResource", []byte(`{"ResourceARN":"`+createBus.EventBusArn+`","TagKeys":["env"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eventBridgeRequest(t, ts, "PutEvents", []byte(`{"Entries":[{"EventBusName":"demo-bus","Source":"demo","DetailType":"demo","Detail":"{\"ok\":true}"}]}`))
	assertStatus(t, resp, http.StatusOK)
	var putEvents struct {
		FailedEntryCount int `json:"FailedEntryCount"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putEvents); err != nil {
		t.Fatalf("unmarshal put events: %v", err)
	}
	if putEvents.FailedEntryCount != 0 {
		t.Fatalf("expected 0 failed entries")
	}

	resp = eventBridgeRequest(t, ts, "RemoveTargets", []byte(`{"Rule":"demo-rule","EventBusName":"demo-bus","Ids":["t1"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eventBridgeRequest(t, ts, "DeleteRule", []byte(`{"Name":"demo-rule","EventBusName":"demo-bus","Force":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eventBridgeRequest(t, ts, "DeleteEventBus", []byte(`{"Name":"demo-bus"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestEventBridgeStage0NotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := eventBridgeRequest(t, ts, "NoSuchAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusNotImplemented)
	var errResp struct {
		Type string `json:"__type"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Type != "NotImplemented" {
		t.Fatalf("expected NotImplemented error, got %q", errResp.Type)
	}
}
