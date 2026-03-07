package server

import (
	"net/http"
	"strings"
	"testing"

	transcoderpb "cloud.google.com/go/video/transcoder/apiv1/transcoderpb"
)

func TestGCPStage4GRPCParity_VideoTranscoder(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	parent := "projects/stackyard/locations/us-central1"
	jobName := parent + "/jobs/job-1"
	jobTemplateName := parent + "/jobTemplates/template-1"

	restListJobsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if restListJobsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video transcoder list jobs, got %d body=%s", restListJobsResp.StatusCode, string(providerContractBody(t, restListJobsResp)))
	}
	restListJobsBody := providerContractJSONMap(t, restListJobsResp)
	restJobs, ok := restListJobsBody["jobs"].([]any)
	if !ok || len(restJobs) == 0 {
		t.Fatalf("expected jobs list in rest payload, got %#v", restListJobsBody["jobs"])
	}
	restJob, _ := restJobs[0].(map[string]any)
	restJobName, _ := restJob["name"].(string)

	var listJobsResp transcoderpb.ListJobsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderListJobsMethod, &transcoderpb.ListJobsRequest{
		Parent:   parent,
		PageSize: 1,
	}, &listJobsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list jobs, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listJobsResp.GetJobs()) != 1 {
		t.Fatalf("expected one grpc job, got %d", len(listJobsResp.GetJobs()))
	}
	if listJobsResp.GetJobs()[0].GetName() != restJobName {
		t.Fatalf("expected grpc job name %q to match rest %q", listJobsResp.GetJobs()[0].GetName(), restJobName)
	}

	restCreateJobResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs", []byte(`{"job":{"name":"projects/stackyard/locations/us-central1/jobs/job-1","inputUri":"gs://stackyard-inputs/job-1.mp4","outputUri":"gs://stackyard-outputs/job-1/","templateId":"preset/web-hd"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if restCreateJobResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video transcoder create job, got %d body=%s", restCreateJobResp.StatusCode, string(providerContractBody(t, restCreateJobResp)))
	}
	restCreateJobBody := providerContractJSONMap(t, restCreateJobResp)
	restCreatedJobName, _ := restCreateJobBody["name"].(string)

	var createJobResp transcoderpb.Job
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderCreateJobMethod, &transcoderpb.CreateJobRequest{
		Parent: parent,
		Job: &transcoderpb.Job{
			Name:      jobName,
			InputUri:  "gs://stackyard-inputs/job-1.mp4",
			OutputUri: "gs://stackyard-outputs/job-1/",
			JobConfig: &transcoderpb.Job_TemplateId{TemplateId: "preset/web-hd"},
		},
	}, &createJobResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create job, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createJobResp.GetName() != restCreatedJobName {
		t.Fatalf("expected grpc job name %q to match rest %q", createJobResp.GetName(), restCreatedJobName)
	}

	restCreateTemplateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobTemplates?jobTemplateId=template-1", []byte(`{"jobTemplate":{"name":"projects/stackyard/locations/us-central1/jobTemplates/template-1","config":{"output":{"uri":"gs://stackyard-outputs/templates/template-1/"}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if restCreateTemplateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video transcoder create template, got %d body=%s", restCreateTemplateResp.StatusCode, string(providerContractBody(t, restCreateTemplateResp)))
	}
	restCreateTemplateBody := providerContractJSONMap(t, restCreateTemplateResp)
	restTemplateName, _ := restCreateTemplateBody["name"].(string)

	var createTemplateResp transcoderpb.JobTemplate
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderCreateJobTemplateMethod, &transcoderpb.CreateJobTemplateRequest{
		Parent:        parent,
		JobTemplateId: "template-1",
		JobTemplate: &transcoderpb.JobTemplate{
			Name: jobTemplateName,
			Config: &transcoderpb.JobConfig{
				Output: &transcoderpb.Output{Uri: "gs://stackyard-outputs/templates/template-1/"},
			},
		},
	}, &createTemplateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create job template, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createTemplateResp.GetName() != restTemplateName {
		t.Fatalf("expected grpc template name %q to match rest %q", createTemplateResp.GetName(), restTemplateName)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderCreateJobMethod, &transcoderpb.CreateJobRequest{
		Job: &transcoderpb.Job{
			Name:      parent + "/jobs/job-2",
			InputUri:  "gs://stackyard-inputs/job-2.mp4",
			OutputUri: "gs://stackyard-outputs/job-2/",
			JobConfig: &transcoderpb.Job_TemplateId{TemplateId: "preset/web-hd"},
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for create job missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderGetJobMethod, &transcoderpb.GetJobRequest{
		Name: parent + "/jobs/missing-job",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "job-not-found") {
		t.Fatalf("expected grpc not found for get job missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoTranscoderCreateJobTemplateMethod, &transcoderpb.CreateJobTemplateRequest{
		Parent:        parent,
		JobTemplateId: "1bad-template",
		JobTemplate: &transcoderpb.JobTemplate{
			Name: parent + "/jobTemplates/1bad-template",
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "job_template_id-invalid") {
		t.Fatalf("expected grpc invalid argument for create template invalid id, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
