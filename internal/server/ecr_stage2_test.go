package server

import (
	"context"
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

func TestECRStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"stage2-raw"}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "SetRepositoryPolicy", body: []byte(`{"repositoryName":"stage2-raw","policyText":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)},
		{name: "GetRepositoryPolicy", body: []byte(`{"repositoryName":"stage2-raw"}`)},
		{name: "PutImageTagMutability", body: []byte(`{"repositoryName":"stage2-raw","imageTagMutability":"IMMUTABLE"}`)},
		{name: "PutImageScanningConfiguration", body: []byte(`{"repositoryName":"stage2-raw","imageScanningConfiguration":{"scanOnPush":true}}`)},
		{name: "BatchGetRepositoryScanningConfiguration", body: []byte(`{"repositoryNames":["stage2-raw","missing-repo"]}`)},
		{name: "DeleteRepositoryPolicy", body: []byte(`{"repositoryName":"stage2-raw"}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage2SDKRepositoryPolicyAndScanning(t *testing.T) {
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

	const repositoryName = "stage2-sdk"
	const policyText = `{"Version":"2012-10-17","Statement":[{"Sid":"AllowAll","Effect":"Allow","Principal":"*","Action":"ecr:*"}]}`

	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	setOut, err := client.SetRepositoryPolicy(ctx, &awsecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
		PolicyText:     aws.String(policyText),
	})
	if err != nil {
		t.Fatalf("set repository policy: %v", err)
	}
	if aws.ToString(setOut.PolicyText) != policyText {
		t.Fatalf("unexpected policy text: %q", aws.ToString(setOut.PolicyText))
	}

	getOut, err := client.GetRepositoryPolicy(ctx, &awsecr.GetRepositoryPolicyInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("get repository policy: %v", err)
	}
	if aws.ToString(getOut.PolicyText) != policyText {
		t.Fatalf("unexpected policy text from get: %q", aws.ToString(getOut.PolicyText))
	}

	mutOut, err := client.PutImageTagMutability(ctx, &awsecr.PutImageTagMutabilityInput{
		RepositoryName:     aws.String(repositoryName),
		ImageTagMutability: awsecrtypes.ImageTagMutabilityImmutable,
	})
	if err != nil {
		t.Fatalf("put image tag mutability: %v", err)
	}
	if mutOut.ImageTagMutability != awsecrtypes.ImageTagMutabilityImmutable {
		t.Fatalf("unexpected image tag mutability: %s", mutOut.ImageTagMutability)
	}

	scanOut, err := client.PutImageScanningConfiguration(ctx, &awsecr.PutImageScanningConfigurationInput{
		RepositoryName: aws.String(repositoryName),
		ImageScanningConfiguration: &awsecrtypes.ImageScanningConfiguration{
			ScanOnPush: true,
		},
	})
	if err != nil {
		t.Fatalf("put image scanning configuration: %v", err)
	}
	if scanOut.ImageScanningConfiguration == nil || !scanOut.ImageScanningConfiguration.ScanOnPush {
		t.Fatalf("expected scanOnPush true")
	}

	batchOut, err := client.BatchGetRepositoryScanningConfiguration(ctx, &awsecr.BatchGetRepositoryScanningConfigurationInput{
		RepositoryNames: []string{repositoryName, "missing-repo"},
	})
	if err != nil {
		t.Fatalf("batch get repository scanning configuration: %v", err)
	}
	if len(batchOut.ScanningConfigurations) != 1 {
		t.Fatalf("expected one scanning configuration, got %d", len(batchOut.ScanningConfigurations))
	}
	if len(batchOut.Failures) != 1 {
		t.Fatalf("expected one scanning failure, got %d", len(batchOut.Failures))
	}
	if string(batchOut.Failures[0].FailureCode) != "REPOSITORY_NOT_FOUND" {
		t.Fatalf("unexpected scanning failure code: %s", batchOut.Failures[0].FailureCode)
	}

	if _, err := client.DeleteRepositoryPolicy(ctx, &awsecr.DeleteRepositoryPolicyInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("delete repository policy: %v", err)
	}

	_, err = client.GetRepositoryPolicy(ctx, &awsecr.GetRepositoryPolicyInput{RepositoryName: aws.String(repositoryName)})
	if err == nil {
		t.Fatalf("expected get repository policy to fail after delete")
	}
	var policyErr *awsecrtypes.RepositoryPolicyNotFoundException
	if !errors.As(err, &policyErr) {
		t.Fatalf("expected RepositoryPolicyNotFoundException, got %v", err)
	}
}
