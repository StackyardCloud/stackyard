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

func TestECRStage7ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const prefix = "stage7-raw"
	resp := ecrRequest(t, ts, "CreatePullThroughCacheRule", []byte(`{"ecrRepositoryPrefix":"`+prefix+`","upstreamRegistryUrl":"registry-1.docker.io"}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribePullThroughCacheRules", body: []byte(`{"ecrRepositoryPrefixes":["` + prefix + `"]}`)},
		{name: "UpdatePullThroughCacheRule", body: []byte(`{"ecrRepositoryPrefix":"` + prefix + `","credentialArn":"arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr"}`)},
		{name: "ValidatePullThroughCacheRule", body: []byte(`{"ecrRepositoryPrefix":"` + prefix + `"}`)},
		{name: "DeletePullThroughCacheRule", body: []byte(`{"ecrRepositoryPrefix":"` + prefix + `"}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage7SDKPullThroughCacheRules(t *testing.T) {
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

	const prefix = "stage7-sdk"
	createOut, err := client.CreatePullThroughCacheRule(ctx, &awsecr.CreatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
		UpstreamRegistryUrl: aws.String("registry-1.docker.io"),
	})
	if err != nil {
		t.Fatalf("create pull through cache rule: %v", err)
	}
	if aws.ToString(createOut.EcrRepositoryPrefix) != prefix {
		t.Fatalf("unexpected prefix in create output: %q", aws.ToString(createOut.EcrRepositoryPrefix))
	}

	_, err = client.CreatePullThroughCacheRule(ctx, &awsecr.CreatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
		UpstreamRegistryUrl: aws.String("registry-1.docker.io"),
	})
	if err == nil {
		t.Fatalf("expected duplicate pull through cache rule create to fail")
	}
	var alreadyExistsErr *awsecrtypes.PullThroughCacheRuleAlreadyExistsException
	if !errors.As(err, &alreadyExistsErr) {
		t.Fatalf("expected PullThroughCacheRuleAlreadyExistsException, got %v", err)
	}

	describeOut, err := client.DescribePullThroughCacheRules(ctx, &awsecr.DescribePullThroughCacheRulesInput{
		EcrRepositoryPrefixes: []string{prefix},
	})
	if err != nil {
		t.Fatalf("describe pull through cache rules: %v", err)
	}
	if len(describeOut.PullThroughCacheRules) != 1 {
		t.Fatalf("expected one pull through cache rule, got %d", len(describeOut.PullThroughCacheRules))
	}

	updateOut, err := client.UpdatePullThroughCacheRule(ctx, &awsecr.UpdatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
		CredentialArn:       aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr"),
	})
	if err != nil {
		t.Fatalf("update pull through cache rule: %v", err)
	}
	if aws.ToString(updateOut.CredentialArn) == "" {
		t.Fatalf("expected credential arn in update output")
	}

	validateOut, err := client.ValidatePullThroughCacheRule(ctx, &awsecr.ValidatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
	})
	if err != nil {
		t.Fatalf("validate pull through cache rule: %v", err)
	}
	if !validateOut.IsValid {
		t.Fatalf("expected pull through cache rule validation to be valid")
	}

	if _, err := client.DeletePullThroughCacheRule(ctx, &awsecr.DeletePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
	}); err != nil {
		t.Fatalf("delete pull through cache rule: %v", err)
	}

	_, err = client.UpdatePullThroughCacheRule(ctx, &awsecr.UpdatePullThroughCacheRuleInput{
		EcrRepositoryPrefix: aws.String(prefix),
		CredentialArn:       aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:ecr"),
	})
	if err == nil {
		t.Fatalf("expected update after delete to fail")
	}
	var notFoundErr *awsecrtypes.PullThroughCacheRuleNotFoundException
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected PullThroughCacheRuleNotFoundException, got %v", err)
	}
}
