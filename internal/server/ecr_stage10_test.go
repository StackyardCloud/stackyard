package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func TestECRStage10ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage10-raw"
	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	putImagePayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "latest",
	})
	if err != nil {
		t.Fatalf("marshal put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", putImagePayload)
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "StartImageScan", body: []byte(`{"repositoryName":"` + repositoryName + `","imageId":{"imageTag":"latest"}}`)},
		{name: "DescribeImageScanFindings", body: []byte(`{"repositoryName":"` + repositoryName + `","imageId":{"imageTag":"latest"}}`)},
	}
	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage10SDKScanExecution(t *testing.T) {
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

	const repositoryName = "stage10-sdk"
	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
	}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	if _, err := client.PutImage(ctx, &awsecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
		ImageTag:       aws.String("latest"),
	}); err != nil {
		t.Fatalf("put image: %v", err)
	}

	startOut, err := client.StartImageScan(ctx, &awsecr.StartImageScanInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &awsecrtypes.ImageIdentifier{
			ImageTag: aws.String("latest"),
		},
	})
	if err != nil {
		t.Fatalf("start image scan: %v", err)
	}
	if startOut.ImageScanStatus == nil || startOut.ImageScanStatus.Status != awsecrtypes.ScanStatusComplete {
		t.Fatalf("expected scan status COMPLETE, got %#v", startOut.ImageScanStatus)
	}

	findingsOut, err := client.DescribeImageScanFindings(ctx, &awsecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &awsecrtypes.ImageIdentifier{
			ImageTag: aws.String("latest"),
		},
	})
	if err != nil {
		t.Fatalf("describe image scan findings: %v", err)
	}
	if findingsOut.ImageScanStatus == nil || findingsOut.ImageScanStatus.Status != awsecrtypes.ScanStatusComplete {
		t.Fatalf("expected findings scan status COMPLETE, got %#v", findingsOut.ImageScanStatus)
	}
	if findingsOut.ImageScanFindings == nil || len(findingsOut.ImageScanFindings.Findings) == 0 {
		t.Fatalf("expected findings in describe image scan findings")
	}
}
