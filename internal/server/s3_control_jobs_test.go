package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3ControlJobsStage5(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	headers := map[string]string{
		"x-amz-account-id": accountID,
		"Content-Type":     "application/xml",
	}

	createBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateJobRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<ClientRequestToken>token-1</ClientRequestToken>` +
		`<ConfirmationRequired>false</ConfirmationRequired>` +
		`<Operation><S3PutObjectCopy></S3PutObjectCopy></Operation>` +
		`<Manifest><Spec></Spec></Manifest>` +
		`<Report><Enabled>true</Enabled></Report>` +
		`<Priority>10</Priority>` +
		`<RoleArn>arn:aws:iam::123456789012:role/S3Control</RoleArn>` +
		`</CreateJobRequest>`
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/jobs", []byte(createBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3ControlCreateJobResult
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create job response: %v", err)
	}
	if createResp.JobId == "" {
		t.Fatalf("expected job id")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/jobs/"+createResp.JobId, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var describeResp s3ControlDescribeJobResult
	if err := xml.Unmarshal(mustBody(t, resp), &describeResp); err != nil {
		t.Fatalf("parse describe job response: %v", err)
	}
	if describeResp.Job.JobId != createResp.JobId {
		t.Fatalf("unexpected job id in describe")
	}
	if describeResp.Job.Priority != 10 {
		t.Fatalf("unexpected job priority")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/jobs", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp s3ControlListJobsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list jobs response: %v", err)
	}
	if len(listResp.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(listResp.Jobs))
	}

	priorityBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<UpdateJobPriorityRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Priority>5</Priority>` +
		`</UpdateJobPriorityRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/priority", []byte(priorityBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	statusBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<UpdateJobStatusRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Status>Cancelled</Status>` +
		`</UpdateJobStatusRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/status", []byte(statusBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	tagBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutJobTaggingRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Tagging><Tag><Key>env</Key><Value>dev</Value></Tag></Tagging>` +
		`</PutJobTaggingRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/tagging", []byte(tagBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var taggingResp s3ControlGetJobTaggingResult
	if err := xml.Unmarshal(mustBody(t, resp), &taggingResp); err != nil {
		t.Fatalf("parse job tagging response: %v", err)
	}
	if len(taggingResp.Tagging.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(taggingResp.Tagging.Tags))
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/jobs/"+createResp.JobId+"/tagging", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchTagSet")
}
