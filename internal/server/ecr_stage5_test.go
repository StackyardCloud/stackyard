package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func TestECRStage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage5-raw"
	const lifecyclePolicyText = `{"rules":[{"rulePriority":1,"description":"expire untagged","selection":{"tagStatus":"untagged","countType":"imageCountMoreThan","countNumber":1},"action":{"type":"expire"}}]}`

	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	putImagePayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "stable",
	})
	if err != nil {
		t.Fatalf("marshal put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", putImagePayload)
	assertStatus(t, resp, http.StatusOK)

	putPolicyPayload, err := json.Marshal(map[string]any{
		"repositoryName":      repositoryName,
		"lifecyclePolicyText": lifecyclePolicyText,
	})
	if err != nil {
		t.Fatalf("marshal put lifecycle policy payload: %v", err)
	}

	actions := []struct {
		name string
		body []byte
	}{
		{name: "PutLifecyclePolicy", body: putPolicyPayload},
		{name: "GetLifecyclePolicy", body: []byte(`{"repositoryName":"` + repositoryName + `"}`)},
		{name: "StartLifecyclePolicyPreview", body: []byte(`{"repositoryName":"` + repositoryName + `"}`)},
		{name: "GetLifecyclePolicyPreview", body: []byte(`{"repositoryName":"` + repositoryName + `"}`)},
		{name: "DeleteLifecyclePolicy", body: []byte(`{"repositoryName":"` + repositoryName + `"}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage5SDKLifecyclePolicyFlow(t *testing.T) {
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

	client := awsecr.NewFromConfig(cfg, func(o *awsecr.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	const repositoryName = "stage5-sdk"
	const lifecyclePolicyText = `{"rules":[{"rulePriority":1,"description":"expire any","selection":{"tagStatus":"any","countType":"imageCountMoreThan","countNumber":0},"action":{"type":"expire"}}]}`

	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	if _, err := client.PutImage(ctx, &awsecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
		ImageTag:       aws.String("stable"),
	}); err != nil {
		t.Fatalf("put image: %v", err)
	}

	putOut, err := client.PutLifecyclePolicy(ctx, &awsecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repositoryName),
		LifecyclePolicyText: aws.String(lifecyclePolicyText),
	})
	if err != nil {
		t.Fatalf("put lifecycle policy: %v", err)
	}
	if aws.ToString(putOut.LifecyclePolicyText) != lifecyclePolicyText {
		t.Fatalf("unexpected lifecycle policy text from put")
	}

	getOut, err := client.GetLifecyclePolicy(ctx, &awsecr.GetLifecyclePolicyInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("get lifecycle policy: %v", err)
	}
	if aws.ToString(getOut.LifecyclePolicyText) != lifecyclePolicyText {
		t.Fatalf("unexpected lifecycle policy text from get")
	}
	if getOut.LastEvaluatedAt == nil {
		t.Fatalf("expected LastEvaluatedAt in get lifecycle policy output")
	}

	startOut, err := client.StartLifecyclePolicyPreview(ctx, &awsecr.StartLifecyclePolicyPreviewInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("start lifecycle policy preview: %v", err)
	}
	if startOut.Status != awsecrtypes.LifecyclePolicyPreviewStatusComplete {
		t.Fatalf("unexpected preview status: %s", startOut.Status)
	}

	previewOut, err := client.GetLifecyclePolicyPreview(ctx, &awsecr.GetLifecyclePolicyPreviewInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("get lifecycle policy preview: %v", err)
	}
	if previewOut.Status != awsecrtypes.LifecyclePolicyPreviewStatusComplete {
		t.Fatalf("unexpected get preview status: %s", previewOut.Status)
	}
	if previewOut.Summary == nil {
		t.Fatalf("expected preview summary")
	}
	if previewOut.Summary.ExpiringImageTotalCount == nil {
		t.Fatalf("expected expiring image total count in summary")
	}
	if *previewOut.Summary.ExpiringImageTotalCount < 1 {
		t.Fatalf("expected at least one preview result")
	}

	if _, err := client.DeleteLifecyclePolicy(ctx, &awsecr.DeleteLifecyclePolicyInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("delete lifecycle policy: %v", err)
	}

	_, err = client.GetLifecyclePolicy(ctx, &awsecr.GetLifecyclePolicyInput{RepositoryName: aws.String(repositoryName)})
	if err == nil {
		t.Fatalf("expected get lifecycle policy after delete to fail")
	}
	var lifecyclePolicyErr *awsecrtypes.LifecyclePolicyNotFoundException
	if !errors.As(err, &lifecyclePolicyErr) {
		t.Fatalf("expected LifecyclePolicyNotFoundException, got %v", err)
	}
}
