package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestImageBuilderStage12ComponentAndPipelineLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := imageBuilderRequest(t, ts, http.MethodPut, "/CreateComponent", []byte(`{"name":"stage-component-001"}`))
	assertStatus(t, resp, http.StatusOK)
	componentARN := imageBuilderPayloadString(t, resp, "componentBuildVersionArn")
	if componentARN == "" {
		t.Fatalf("expected componentBuildVersionArn from CreateComponent")
	}

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodGet,
		"/GetComponent?componentBuildVersionArn="+url.QueryEscape(componentARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-component-001") {
		t.Fatalf("expected GetComponent to include stage-component-001, got %q", body)
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/PutComponentPolicy", []byte(`{"componentArn":"`+componentARN+`","policy":"{}"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodGet,
		"/GetComponentPolicy?componentArn="+url.QueryEscape(componentARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListComponents", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-component-001") {
		t.Fatalf("expected ListComponents to include stage-component-001, got %q", body)
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateImageRecipe", []byte(`{"name":"stage-image-recipe-001"}`))
	assertStatus(t, resp, http.StatusOK)
	imageRecipeARN := imageBuilderPayloadString(t, resp, "imageRecipeArn")
	if imageRecipeARN == "" {
		t.Fatalf("expected imageRecipeArn from CreateImageRecipe")
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateInfrastructureConfiguration", []byte(`{"name":"stage-infra-config-001"}`))
	assertStatus(t, resp, http.StatusOK)
	infraARN := imageBuilderPayloadString(t, resp, "infrastructureConfigurationArn")
	if infraARN == "" {
		t.Fatalf("expected infrastructureConfigurationArn from CreateInfrastructureConfiguration")
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateDistributionConfiguration", []byte(`{"name":"stage-distribution-config-001"}`))
	assertStatus(t, resp, http.StatusOK)
	distributionARN := imageBuilderPayloadString(t, resp, "distributionConfigurationArn")
	if distributionARN == "" {
		t.Fatalf("expected distributionConfigurationArn from CreateDistributionConfiguration")
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateImagePipeline", []byte(`{"name":"stage-image-pipeline-001"}`))
	assertStatus(t, resp, http.StatusOK)
	imagePipelineARN := imageBuilderPayloadString(t, resp, "imagePipelineArn")
	if imagePipelineARN == "" {
		t.Fatalf("expected imagePipelineArn from CreateImagePipeline")
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/StartImagePipelineExecution", []byte(`{"imagePipelineArn":"`+imagePipelineARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodGet,
		"/GetImagePipeline?imagePipelineArn="+url.QueryEscape(imagePipelineARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-image-pipeline-001") {
		t.Fatalf("expected GetImagePipeline to include stage-image-pipeline-001, got %q", body)
	}

	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImagePipelines", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImages", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImagePackages", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImageScanFindings", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImageScanFindingAggregations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/DeleteImageRecipe?imageRecipeArn="+url.QueryEscape(imageRecipeARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/DeleteInfrastructureConfiguration?infrastructureConfigurationArn="+url.QueryEscape(infraARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/DeleteDistributionConfiguration?distributionConfigurationArn="+url.QueryEscape(distributionARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/DeleteImagePipeline?imagePipelineArn="+url.QueryEscape(imagePipelineARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/DeleteComponent?componentBuildVersionArn="+url.QueryEscape(componentARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
}

func TestImageBuilderStage3456LifecycleWorkflowTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := imageBuilderRequest(t, ts, http.MethodPut, "/CreateLifecyclePolicy", []byte(`{"name":"stage-lifecycle-policy-001"}`))
	assertStatus(t, resp, http.StatusOK)
	policyARN := imageBuilderPayloadString(t, resp, "lifecyclePolicyArn")
	if policyARN == "" {
		t.Fatalf("expected lifecyclePolicyArn from CreateLifecyclePolicy")
	}

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateLifecyclePolicy", []byte(`{"name":"stage-lifecycle-policy-001"}`))
	assertStatus(t, resp, http.StatusOK)
	policyARN2 := imageBuilderPayloadString(t, resp, "lifecyclePolicyArn")
	if policyARN2 != policyARN {
		t.Fatalf("expected idempotent lifecycle policy ARN, got %q and %q", policyARN, policyARN2)
	}

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodGet,
		"/GetLifecyclePolicy?lifecyclePolicyArn="+url.QueryEscape(policyARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListLifecyclePolicies", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodGet, "/GetLifecycleExecution?lifecycleExecutionId=lifecycle-execution-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListLifecycleExecutions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListLifecycleExecutionResources", []byte(`{"lifecycleExecutionId":"lifecycle-execution-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CancelLifecycleExecution", []byte(`{"lifecycleExecutionId":"lifecycle-execution-000001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(t, ts, http.MethodPut, "/CreateWorkflow", []byte(`{"name":"stage-workflow-001"}`))
	assertStatus(t, resp, http.StatusOK)
	workflowARN := imageBuilderPayloadString(t, resp, "workflowBuildVersionArn")
	if workflowARN == "" {
		t.Fatalf("expected workflowBuildVersionArn from CreateWorkflow")
	}

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodGet,
		"/GetWorkflow?workflowBuildVersionArn="+url.QueryEscape(workflowARN),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListWorkflows", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodGet, "/GetWorkflowExecution?workflowExecutionId=workflow-execution-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListWorkflowExecutions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodGet, "/GetWorkflowStepExecution?stepExecutionId=workflow-step-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListWorkflowStepExecutions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListWaitingWorkflowSteps", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = imageBuilderRequest(t, ts, http.MethodPut, "/SendWorkflowStepAction", []byte(`{"stepExecutionId":"workflow-step-000001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodPost,
		"/tags/",
		[]byte(`{"resourceArn":"`+workflowARN+`","tags":{"env":"stage","owner":"qa"}}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(t, ts, http.MethodGet, "/tags/?resourceArn="+url.QueryEscape(workflowARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = imageBuilderRequest(
		t,
		ts,
		http.MethodDelete,
		"/tags/",
		[]byte(`{"resourceArn":"`+workflowARN+`","tagKeys":["owner"]}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = imageBuilderRequest(t, ts, http.MethodPost, "/TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = imageBuilderRequest(t, ts, http.MethodPost, "/ListImages", []byte(`{"broken":`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func imageBuilderPayloadString(t *testing.T, resp *http.Response, key string) string {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
