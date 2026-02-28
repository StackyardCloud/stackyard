package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func swfRequest(t *testing.T, ts *httptest.Server, target string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.0",
		"X-Amz-Target": target,
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, headers, "swf")
}

func TestSWFStage0DomainWorkflowLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	registerDomain := []byte(`{"name":"demo-domain","description":"demo","workflowExecutionRetentionPeriodInDays":"7"}`)
	resp := swfRequest(t, ts, "SimpleWorkflowService.RegisterDomain", registerDomain)
	assertStatus(t, resp, http.StatusOK)

	describeDomain := []byte(`{"name":"demo-domain"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.DescribeDomain", describeDomain)
	assertStatus(t, resp, http.StatusOK)
	var domainResp struct {
		DomainInfo struct {
			Name string `json:"name"`
		} `json:"domainInfo"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &domainResp); err != nil {
		t.Fatalf("describe domain unmarshal: %v", err)
	}
	if domainResp.DomainInfo.Name != "demo-domain" {
		t.Fatalf("expected domain name demo-domain, got %q", domainResp.DomainInfo.Name)
	}

	listDomains := []byte(`{"registrationStatus":"REGISTERED"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.ListDomains", listDomains)
	assertStatus(t, resp, http.StatusOK)
	var listDomainsResp struct {
		DomainInfos []struct {
			Name string `json:"name"`
		} `json:"domainInfos"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listDomainsResp); err != nil {
		t.Fatalf("list domains unmarshal: %v", err)
	}
	if len(listDomainsResp.DomainInfos) != 1 || listDomainsResp.DomainInfos[0].Name != "demo-domain" {
		t.Fatalf("expected domain in list")
	}

	registerWorkflow := []byte(`{"domain":"demo-domain","name":"demo-workflow","version":"1","description":"demo"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.RegisterWorkflowType", registerWorkflow)
	assertStatus(t, resp, http.StatusOK)

	startWorkflow := []byte(`{"domain":"demo-domain","workflowId":"wf-1","workflowType":{"name":"demo-workflow","version":"1"},"taskList":{"name":"main"}}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.StartWorkflowExecution", startWorkflow)
	assertStatus(t, resp, http.StatusOK)
	var startResp struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startResp); err != nil {
		t.Fatalf("start workflow unmarshal: %v", err)
	}
	if startResp.RunID == "" {
		t.Fatalf("expected runId")
	}

	describeExec := []byte(`{"domain":"demo-domain","execution":{"workflowId":"wf-1","runId":"` + startResp.RunID + `"}}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.DescribeWorkflowExecution", describeExec)
	assertStatus(t, resp, http.StatusOK)
	var describeResp struct {
		ExecutionInfo struct {
			Execution struct {
				WorkflowID string `json:"workflowId"`
			} `json:"execution"`
		} `json:"executionInfo"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeResp); err != nil {
		t.Fatalf("describe execution unmarshal: %v", err)
	}
	if describeResp.ExecutionInfo.Execution.WorkflowID != "wf-1" {
		t.Fatalf("expected workflowId wf-1")
	}

	listOpen := []byte(`{"domain":"demo-domain"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.ListOpenWorkflowExecutions", listOpen)
	assertStatus(t, resp, http.StatusOK)
	var listOpenResp struct {
		ExecutionInfos []struct {
			Execution struct {
				WorkflowID string `json:"workflowId"`
			} `json:"execution"`
		} `json:"executionInfos"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOpenResp); err != nil {
		t.Fatalf("list open unmarshal: %v", err)
	}
	if len(listOpenResp.ExecutionInfos) != 1 {
		t.Fatalf("expected 1 open execution")
	}

	countOpen := []byte(`{"domain":"demo-domain"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.CountOpenWorkflowExecutions", countOpen)
	assertStatus(t, resp, http.StatusOK)
	var countOpenResp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &countOpenResp); err != nil {
		t.Fatalf("count open unmarshal: %v", err)
	}
	if countOpenResp.Count != 1 {
		t.Fatalf("expected count 1, got %d", countOpenResp.Count)
	}

	cancel := []byte(`{"domain":"demo-domain","workflowId":"wf-1","runId":"` + startResp.RunID + `"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.RequestCancelWorkflowExecution", cancel)
	assertStatus(t, resp, http.StatusOK)

	listClosed := []byte(`{"domain":"demo-domain"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.ListClosedWorkflowExecutions", listClosed)
	assertStatus(t, resp, http.StatusOK)
	var listClosedResp struct {
		ExecutionInfos []struct {
			Execution struct {
				WorkflowID string `json:"workflowId"`
			} `json:"execution"`
		} `json:"executionInfos"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listClosedResp); err != nil {
		t.Fatalf("list closed unmarshal: %v", err)
	}
	if len(listClosedResp.ExecutionInfos) != 1 {
		t.Fatalf("expected 1 closed execution")
	}
}

func TestSWFStage0TagsAndActivityTypes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	registerDomain := []byte(`{"name":"tag-domain","description":"demo","workflowExecutionRetentionPeriodInDays":"7"}`)
	resp := swfRequest(t, ts, "SimpleWorkflowService.RegisterDomain", registerDomain)
	assertStatus(t, resp, http.StatusOK)

	tagReq := []byte(`{"resourceArn":"arn:aws:swf:us-east-1:123456789012:domain/tag-domain","tags":[{"key":"env","value":"test"}]}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.TagResource", tagReq)
	assertStatus(t, resp, http.StatusOK)

	listTags := []byte(`{"resourceArn":"arn:aws:swf:us-east-1:123456789012:domain/tag-domain"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.ListTagsForResource", listTags)
	assertStatus(t, resp, http.StatusOK)
	var listTagsResp struct {
		Tags []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsResp); err != nil {
		t.Fatalf("list tags unmarshal: %v", err)
	}
	if len(listTagsResp.Tags) != 1 || listTagsResp.Tags[0].Key != "env" {
		t.Fatalf("expected tag env")
	}

	untagReq := []byte(`{"resourceArn":"arn:aws:swf:us-east-1:123456789012:domain/tag-domain","tagKeys":["env"]}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.UntagResource", untagReq)
	assertStatus(t, resp, http.StatusOK)

	registerActivity := []byte(`{"domain":"tag-domain","name":"activity","version":"1","description":"demo"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.RegisterActivityType", registerActivity)
	assertStatus(t, resp, http.StatusOK)

	listActivities := []byte(`{"domain":"tag-domain","registrationStatus":"REGISTERED"}`)
	resp = swfRequest(t, ts, "SimpleWorkflowService.ListActivityTypes", listActivities)
	assertStatus(t, resp, http.StatusOK)
	var listActivitiesResp struct {
		TypeInfos []struct {
			ActivityType struct {
				Name string `json:"name"`
			} `json:"activityType"`
		} `json:"typeInfos"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listActivitiesResp); err != nil {
		t.Fatalf("list activity types unmarshal: %v", err)
	}
	if len(listActivitiesResp.TypeInfos) != 1 || listActivitiesResp.TypeInfos[0].ActivityType.Name != "activity" {
		t.Fatalf("expected activity type in list")
	}
}
