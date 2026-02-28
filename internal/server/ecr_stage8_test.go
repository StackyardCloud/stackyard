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

func TestECRStage8ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const prefix = "stage8-raw"
	resp := ecrRequest(t, ts, "CreateRepositoryCreationTemplate", []byte(`{"prefix":"`+prefix+`","appliedFor":["REPLICATION"]}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeRepositoryCreationTemplates", body: []byte(`{"prefixes":["` + prefix + `"]}`)},
		{name: "UpdateRepositoryCreationTemplate", body: []byte(`{"prefix":"` + prefix + `","description":"updated","imageTagMutability":"IMMUTABLE"}`)},
		{name: "DeleteRepositoryCreationTemplate", body: []byte(`{"prefix":"` + prefix + `"}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage8SDKRepositoryCreationTemplateLifecycle(t *testing.T) {
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

	const prefix = "stage8-sdk"
	createOut, err := client.CreateRepositoryCreationTemplate(ctx, &awsecr.CreateRepositoryCreationTemplateInput{
		Prefix:     aws.String(prefix),
		AppliedFor: []awsecrtypes.RCTAppliedFor{awsecrtypes.RCTAppliedForReplication},
		ResourceTags: []awsecrtypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	})
	if err != nil {
		t.Fatalf("create repository creation template: %v", err)
	}
	if createOut.RepositoryCreationTemplate == nil || aws.ToString(createOut.RepositoryCreationTemplate.Prefix) != prefix {
		t.Fatalf("unexpected create repository creation template output")
	}

	_, err = client.CreateRepositoryCreationTemplate(ctx, &awsecr.CreateRepositoryCreationTemplateInput{
		Prefix:     aws.String(prefix),
		AppliedFor: []awsecrtypes.RCTAppliedFor{awsecrtypes.RCTAppliedForReplication},
	})
	if err == nil {
		t.Fatalf("expected duplicate template create to fail")
	}
	var alreadyExistsErr *awsecrtypes.TemplateAlreadyExistsException
	if !errors.As(err, &alreadyExistsErr) {
		t.Fatalf("expected TemplateAlreadyExistsException, got %v", err)
	}

	describeOut, err := client.DescribeRepositoryCreationTemplates(ctx, &awsecr.DescribeRepositoryCreationTemplatesInput{
		Prefixes: []string{prefix},
	})
	if err != nil {
		t.Fatalf("describe repository creation templates: %v", err)
	}
	if len(describeOut.RepositoryCreationTemplates) != 1 {
		t.Fatalf("expected one repository creation template, got %d", len(describeOut.RepositoryCreationTemplates))
	}

	updateOut, err := client.UpdateRepositoryCreationTemplate(ctx, &awsecr.UpdateRepositoryCreationTemplateInput{
		Prefix:             aws.String(prefix),
		Description:        aws.String("updated"),
		ImageTagMutability: awsecrtypes.ImageTagMutabilityImmutable,
	})
	if err != nil {
		t.Fatalf("update repository creation template: %v", err)
	}
	if updateOut.RepositoryCreationTemplate == nil || aws.ToString(updateOut.RepositoryCreationTemplate.Description) != "updated" {
		t.Fatalf("unexpected template description after update")
	}

	if _, err := client.DeleteRepositoryCreationTemplate(ctx, &awsecr.DeleteRepositoryCreationTemplateInput{
		Prefix: aws.String(prefix),
	}); err != nil {
		t.Fatalf("delete repository creation template: %v", err)
	}

	_, err = client.UpdateRepositoryCreationTemplate(ctx, &awsecr.UpdateRepositoryCreationTemplateInput{
		Prefix:      aws.String(prefix),
		Description: aws.String("missing"),
	})
	if err == nil {
		t.Fatalf("expected update after delete to fail")
	}
	var notFoundErr *awsecrtypes.TemplateNotFoundException
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected TemplateNotFoundException, got %v", err)
	}
}
