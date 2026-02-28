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

func TestECRStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage4-raw"

	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	for _, tag := range []string{"v1", "v2"} {
		payload, err := json.Marshal(map[string]any{
			"repositoryName": repositoryName,
			"imageManifest":  `{"schemaVersion":2}`,
			"imageTag":       tag,
		})
		if err != nil {
			t.Fatalf("marshal put image payload: %v", err)
		}
		resp = ecrRequest(t, ts, "PutImage", payload)
		assertStatus(t, resp, http.StatusOK)
	}

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeImages", body: []byte(`{"repositoryName":"` + repositoryName + `","filter":{"tagStatus":"TAGGED"}}`)},
		{name: "BatchDeleteImage", body: []byte(`{"repositoryName":"` + repositoryName + `","imageIds":[{"imageTag":"v1"}]}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage4SDKImageCatalogAndDeletion(t *testing.T) {
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

	const repositoryName = "stage4-sdk"
	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	for _, tag := range []string{"v1", "v2"} {
		if _, err := client.PutImage(ctx, &awsecr.PutImageInput{
			RepositoryName: aws.String(repositoryName),
			ImageManifest:  aws.String(`{"schemaVersion":2}`),
			ImageTag:       aws.String(tag),
		}); err != nil {
			t.Fatalf("put image (%s): %v", tag, err)
		}
	}

	describeOut, err := client.DescribeImages(ctx, &awsecr.DescribeImagesInput{
		RepositoryName: aws.String(repositoryName),
		Filter:         &awsecrtypes.DescribeImagesFilter{TagStatus: awsecrtypes.TagStatusTagged},
	})
	if err != nil {
		t.Fatalf("describe images: %v", err)
	}
	if len(describeOut.ImageDetails) != 1 {
		t.Fatalf("expected 1 image detail, got %d", len(describeOut.ImageDetails))
	}

	deleteOut, err := client.BatchDeleteImage(ctx, &awsecr.BatchDeleteImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageIds: []awsecrtypes.ImageIdentifier{
			{ImageTag: aws.String("v1")},
		},
	})
	if err != nil {
		t.Fatalf("batch delete image by tag: %v", err)
	}
	if len(deleteOut.ImageIds) != 1 {
		t.Fatalf("expected 1 deleted image id, got %d", len(deleteOut.ImageIds))
	}

	describeOut, err = client.DescribeImages(ctx, &awsecr.DescribeImagesInput{
		RepositoryName: aws.String(repositoryName),
		Filter:         &awsecrtypes.DescribeImagesFilter{TagStatus: awsecrtypes.TagStatusTagged},
	})
	if err != nil {
		t.Fatalf("describe images after first delete: %v", err)
	}
	if len(describeOut.ImageDetails) != 1 {
		t.Fatalf("expected 1 image detail after first delete, got %d", len(describeOut.ImageDetails))
	}
	remainingDigest := aws.ToString(describeOut.ImageDetails[0].ImageDigest)
	if remainingDigest == "" {
		t.Fatalf("expected remaining digest")
	}

	deleteOut, err = client.BatchDeleteImage(ctx, &awsecr.BatchDeleteImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageIds: []awsecrtypes.ImageIdentifier{
			{ImageDigest: aws.String(remainingDigest)},
		},
	})
	if err != nil {
		t.Fatalf("batch delete image by digest: %v", err)
	}
	if len(deleteOut.ImageIds) != 1 {
		t.Fatalf("expected 1 deleted image id by digest, got %d", len(deleteOut.ImageIds))
	}

	describeOut, err = client.DescribeImages(ctx, &awsecr.DescribeImagesInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("describe images final: %v", err)
	}
	if len(describeOut.ImageDetails) != 0 {
		t.Fatalf("expected no images after deletes, got %d", len(describeOut.ImageDetails))
	}
}
