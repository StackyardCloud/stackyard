package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSageMakerStage12TrainingModelAndEndpointLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sagemakerRequest(t, ts, "CreateTrainingJob", `{"TrainingJobName":"stage-training-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DescribeTrainingJob", `{"TrainingJobName":"stage-training-job"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-training-job") {
		t.Fatalf("expected DescribeTrainingJob to include stage-training-job, got %q", body)
	}

	resp = sagemakerRequest(t, ts, "ListTrainingJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TrainingJobSummaries") {
		t.Fatalf("expected ListTrainingJobs to include TrainingJobSummaries, got %q", body)
	}

	resp = sagemakerRequest(t, ts, "StopTrainingJob", `{"TrainingJobName":"stage-training-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "CreateModel", `{"ModelName":"stage-model"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DescribeModel", `{"ModelName":"stage-model"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "ListModels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-model") {
		t.Fatalf("expected ListModels to include stage-model, got %q", body)
	}

	resp = sagemakerRequest(t, ts, "CreateEndpointConfig", `{"EndpointConfigName":"stage-endpoint-config"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "CreateEndpoint", `{"EndpointName":"stage-endpoint","EndpointConfigName":"stage-endpoint-config"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DescribeEndpoint", `{"EndpointName":"stage-endpoint"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "ListEndpoints", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-endpoint") {
		t.Fatalf("expected ListEndpoints to include stage-endpoint, got %q", body)
	}

	resp = sagemakerRequest(t, ts, "UpdateEndpoint", `{"EndpointName":"stage-endpoint","EndpointConfigName":"stage-endpoint-config"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DeleteEndpoint", `{"EndpointName":"stage-endpoint"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DeleteEndpointConfig", `{"EndpointConfigName":"stage-endpoint-config"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DeleteModel", `{"ModelName":"stage-model"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestSageMakerStage34NotebookAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sagemakerRequest(t, ts, "CreateNotebookInstance", `{"NotebookInstanceName":"stage-notebook","InstanceType":"ml.t3.medium"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DescribeNotebookInstance", `{"NotebookInstanceName":"stage-notebook"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "ListNotebookInstances", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "StopNotebookInstance", `{"NotebookInstanceName":"stage-notebook"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "StartNotebookInstance", `{"NotebookInstanceName":"stage-notebook"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "UpdateNotebookInstance", `{"NotebookInstanceName":"stage-notebook","InstanceType":"ml.m5.large"}`)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:sagemaker:us-east-1:123456789012:endpoint/stage-endpoint"
	resp = sagemakerRequest(t, ts, "AddTags", `{"ResourceArn":"`+resourceARN+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "ListTags", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTags to include owner tag, got %q", body)
	}

	resp = sagemakerRequest(t, ts, "DeleteTags", `{"ResourceArn":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DeleteNotebookInstance", `{"NotebookInstanceName":"stage-notebook"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestSageMakerStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sagemakerRequest(t, ts, "CreateModel", `{"ModelName":"stage-idempotent-model"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "CreateModel", `{"ModelName":"stage-idempotent-model"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "DescribeModel", `{"ModelName":"stage-idempotent-model"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = sagemakerRequest(t, ts, "TotallyUnknownAction", `{}`)
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
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "SageMaker.ListTrainingJobs",
		},
		"sagemaker",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
