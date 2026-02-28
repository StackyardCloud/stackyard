package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsbatch "github.com/aws/aws-sdk-go-v2/service/batch"
	awsbatchtypes "github.com/aws/aws-sdk-go-v2/service/batch/types"
)

func batchRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if method == http.MethodPost {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "batch")
}

func TestBatchStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := batchRequest(t, ts, http.MethodPost, "/v1/createcomputeenvironment", []byte(`{"computeEnvironmentName":"demo-ce","type":"UNMANAGED","state":"ENABLED","unmanagedvCpus":16,"tags":{"env":"test"}}`))
	assertStatus(t, resp, http.StatusOK)
	var createCE map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &createCE); err != nil {
		t.Fatalf("unmarshal create compute env: %v", err)
	}
	if createCE["computeEnvironmentArn"] == "" {
		t.Fatalf("expected computeEnvironmentArn")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describecomputeenvironments", []byte(`{"computeEnvironments":["demo-ce"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/createjobqueue", []byte(`{"jobQueueName":"demo-queue","state":"ENABLED","priority":1,"computeEnvironmentOrder":[{"order":1,"computeEnvironment":"demo-ce"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/registerjobdefinition", []byte(`{"jobDefinitionName":"demo-jobdef","type":"container","parameters":{"mode":"fast"},"tags":{"team":"platform"}}`))
	assertStatus(t, resp, http.StatusOK)
	var registerOut struct {
		JobDefinitionArn string `json:"jobDefinitionArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut); err != nil {
		t.Fatalf("unmarshal register job definition: %v", err)
	}
	if registerOut.JobDefinitionArn == "" {
		t.Fatalf("expected jobDefinitionArn")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/submitjob", []byte(`{"jobName":"demo-job","jobQueue":"demo-queue","jobDefinition":"demo-jobdef"}`))
	assertStatus(t, resp, http.StatusOK)
	var submitOut struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &submitOut); err != nil {
		t.Fatalf("unmarshal submit job: %v", err)
	}
	if submitOut.JobID == "" {
		t.Fatalf("expected job id")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/listjobs", []byte(`{"jobQueue":"demo-queue"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describejobs", []byte(`{"jobs":["`+submitOut.JobID+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/canceljob", []byte(`{"jobId":"`+submitOut.JobID+`","reason":"cancel"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/terminatejob", []byte(`{"jobId":"`+submitOut.JobID+`","reason":"terminate"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/tags/"+url.PathEscape(registerOut.JobDefinitionArn), []byte(`{"tags":{"owner":"ops"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodGet, "/v1/tags/"+url.PathEscape(registerOut.JobDefinitionArn), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodDelete, "/v1/tags/"+url.PathEscape(registerOut.JobDefinitionArn)+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describeservicejob", []byte(`{}`))
	assertStatus(t, resp, http.StatusNotFound)
}

func TestBatchStage0ExtendedOperations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := batchRequest(t, ts, http.MethodPost, "/v1/createcomputeenvironment", []byte(`{"computeEnvironmentName":"ext-ce","type":"UNMANAGED","state":"ENABLED"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/createjobqueue", []byte(`{"jobQueueName":"ext-queue","state":"ENABLED","priority":1,"computeEnvironmentOrder":[{"order":1,"computeEnvironment":"ext-ce"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/registerjobdefinition", []byte(`{"jobDefinitionName":"ext-jobdef","type":"container"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/createconsumableresource", []byte(`{"consumableResourceName":"gpu-hours","resourceType":"REPLENISHABLE","totalQuantity":100}`))
	assertStatus(t, resp, http.StatusOK)
	var createCR struct {
		ConsumableResourceArn  string `json:"consumableResourceArn"`
		ConsumableResourceName string `json:"consumableResourceName"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createCR); err != nil {
		t.Fatalf("unmarshal create consumable resource: %v", err)
	}
	if createCR.ConsumableResourceArn == "" || createCR.ConsumableResourceName == "" {
		t.Fatalf("expected consumable resource identifiers")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describeconsumableresource", []byte(`{"consumableResource":"gpu-hours"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/listconsumableresources", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/updateconsumableresource", []byte(`{"consumableResource":"gpu-hours","operation":"ADD","quantity":5}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/submitjob", []byte(`{
		"jobName":"ext-job",
		"jobQueue":"ext-queue",
		"jobDefinition":"ext-jobdef",
		"consumableResourcePropertiesOverride":{"consumableResourceList":[{"consumableResource":"gpu-hours","quantity":2}]},
		"shareIdentifier":"team-a",
		"quantity":2
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/listjobsbyconsumableresource", []byte(`{"consumableResource":"gpu-hours","maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var jobsByConsumable struct {
		Jobs []any `json:"jobs"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &jobsByConsumable); err != nil {
		t.Fatalf("unmarshal list jobs by consumable: %v", err)
	}
	if len(jobsByConsumable.Jobs) == 0 {
		t.Fatalf("expected jobs by consumable to include submitted job")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/getjobqueuesnapshot", []byte(`{"jobQueue":"ext-queue"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/createserviceenvironment", []byte(`{
		"serviceEnvironmentName":"svc-env",
		"serviceEnvironmentType":"SAGEMAKER_TRAINING",
		"state":"ENABLED",
		"capacityLimits":[{"capacityUnit":"GPU","maxCapacity":10}],
		"tags":{"env":"test"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describeserviceenvironments", []byte(`{"serviceEnvironments":["svc-env"],"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/updateserviceenvironment", []byte(`{"serviceEnvironment":"svc-env","state":"DISABLED","capacityLimits":[{"capacityUnit":"GPU","maxCapacity":5}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/submitservicejob", []byte(`{
		"jobName":"svc-job",
		"jobQueue":"ext-queue",
		"serviceJobType":"SAGEMAKER_TRAINING",
		"serviceRequestPayload":"{}",
		"schedulingPriority":1,
		"retryStrategy":{"attempts":2},
		"timeoutConfig":{"attemptDurationSeconds":60}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var submitServiceOut struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &submitServiceOut); err != nil {
		t.Fatalf("unmarshal submit service job: %v", err)
	}
	if submitServiceOut.JobID == "" {
		t.Fatalf("expected service job id")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describeservicejob", []byte(`{"jobId":"`+submitServiceOut.JobID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/listservicejobs", []byte(`{"jobQueue":"ext-queue","maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/terminateservicejob", []byte(`{"jobId":"`+submitServiceOut.JobID+`","reason":"cleanup"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/deleteserviceenvironment", []byte(`{"serviceEnvironment":"svc-env"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/deleteconsumableresource", []byte(`{"consumableResource":"gpu-hours"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestBatchStage0OperationCoverage(t *testing.T) {
	if len(batchOperations) != 39 {
		t.Fatalf("expected 39 Batch operations from docs, got %d", len(batchOperations))
	}
	if len(batchOperationByName) != len(batchOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateComputeEnvironment",
		"CreateConsumableResource",
		"CreateServiceEnvironment",
		"GetJobQueueSnapshot",
		"ListTagsForResource",
		"SubmitJob",
		"SubmitServiceJob",
		"TerminateServiceJob",
		"UpdateServiceEnvironment",
	}
	for _, name := range required {
		if _, ok := batchOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestBatchStage0SDKClientLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awsbatch.NewFromConfig(cfg, func(o *awsbatch.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	createCE, err := client.CreateComputeEnvironment(ctx, &awsbatch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String("sdk-ce"),
		Type:                   awsbatchtypes.CETypeUnmanaged,
		State:                  awsbatchtypes.CEStateEnabled,
		UnmanagedvCpus:         aws.Int32(8),
	})
	if err != nil {
		t.Fatalf("create compute environment: %v", err)
	}
	if createCE.ComputeEnvironmentArn == nil {
		t.Fatalf("expected compute environment arn")
	}

	createSP, err := client.CreateSchedulingPolicy(ctx, &awsbatch.CreateSchedulingPolicyInput{Name: aws.String("sdk-sp")})
	if err != nil {
		t.Fatalf("create scheduling policy: %v", err)
	}

	createJQ, err := client.CreateJobQueue(ctx, &awsbatch.CreateJobQueueInput{
		JobQueueName: aws.String("sdk-queue"),
		Priority:     aws.Int32(1),
		State:        awsbatchtypes.JQStateEnabled,
		ComputeEnvironmentOrder: []awsbatchtypes.ComputeEnvironmentOrder{{
			Order:              aws.Int32(1),
			ComputeEnvironment: aws.String("sdk-ce"),
		}},
		SchedulingPolicyArn: createSP.Arn,
	})
	if err != nil {
		t.Fatalf("create job queue: %v", err)
	}
	if createJQ.JobQueueArn == nil {
		t.Fatalf("expected job queue arn")
	}

	register, err := client.RegisterJobDefinition(ctx, &awsbatch.RegisterJobDefinitionInput{
		JobDefinitionName: aws.String("sdk-jd"),
		Type:              awsbatchtypes.JobDefinitionTypeContainer,
	})
	if err != nil {
		t.Fatalf("register job definition: %v", err)
	}
	if register.JobDefinitionArn == nil {
		t.Fatalf("expected job definition arn")
	}

	submit, err := client.SubmitJob(ctx, &awsbatch.SubmitJobInput{
		JobName:       aws.String("sdk-job"),
		JobQueue:      aws.String("sdk-queue"),
		JobDefinition: aws.String("sdk-jd"),
	})
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}
	if submit.JobId == nil {
		t.Fatalf("expected job id")
	}

	if _, err := client.DescribeJobs(ctx, &awsbatch.DescribeJobsInput{Jobs: []string{aws.ToString(submit.JobId)}}); err != nil {
		t.Fatalf("describe jobs: %v", err)
	}
	if _, err := client.ListJobs(ctx, &awsbatch.ListJobsInput{JobQueue: aws.String("sdk-queue")}); err != nil {
		t.Fatalf("list jobs: %v", err)
	}

	if _, err := client.TagResource(ctx, &awsbatch.TagResourceInput{ResourceArn: register.JobDefinitionArn, Tags: map[string]string{"env": "test"}}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	if _, err := client.ListTagsForResource(ctx, &awsbatch.ListTagsForResourceInput{ResourceArn: register.JobDefinitionArn}); err != nil {
		t.Fatalf("list tags for resource: %v", err)
	}
	if _, err := client.UntagResource(ctx, &awsbatch.UntagResourceInput{ResourceArn: register.JobDefinitionArn, TagKeys: []string{"env"}}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	if _, err := client.CancelJob(ctx, &awsbatch.CancelJobInput{JobId: submit.JobId, Reason: aws.String("cancel")}); err != nil {
		t.Fatalf("cancel job: %v", err)
	}
	if _, err := client.TerminateJob(ctx, &awsbatch.TerminateJobInput{JobId: submit.JobId, Reason: aws.String("terminate")}); err != nil {
		t.Fatalf("terminate job: %v", err)
	}

	if _, err := client.DeregisterJobDefinition(ctx, &awsbatch.DeregisterJobDefinitionInput{JobDefinition: register.JobDefinitionArn}); err != nil {
		t.Fatalf("deregister job definition: %v", err)
	}
	if _, err := client.DeleteJobQueue(ctx, &awsbatch.DeleteJobQueueInput{JobQueue: createJQ.JobQueueArn}); err != nil {
		t.Fatalf("delete job queue: %v", err)
	}
	if _, err := client.DeleteComputeEnvironment(ctx, &awsbatch.DeleteComputeEnvironmentInput{ComputeEnvironment: createCE.ComputeEnvironmentArn}); err != nil {
		t.Fatalf("delete compute environment: %v", err)
	}
	if _, err := client.DeleteSchedulingPolicy(ctx, &awsbatch.DeleteSchedulingPolicyInput{Arn: createSP.Arn}); err != nil {
		t.Fatalf("delete scheduling policy: %v", err)
	}
}
