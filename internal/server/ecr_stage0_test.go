package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func ecrRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonEC2ContainerRegistry_V20150921." + action,
		},
		"ecr",
	)
}

func TestECRStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecrRequest(t, ts, "GetAuthorizationToken", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var authOut struct {
		AuthorizationData []struct {
			AuthorizationToken string `json:"authorizationToken"`
			ProxyEndpoint      string `json:"proxyEndpoint"`
		} `json:"authorizationData"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &authOut); err != nil {
		t.Fatalf("unmarshal auth token response: %v", err)
	}
	if len(authOut.AuthorizationData) != 1 {
		t.Fatalf("expected one authorization entry")
	}
	decoded, err := base64.StdEncoding.DecodeString(authOut.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		t.Fatalf("decode authorization token: %v", err)
	}
	if !strings.HasPrefix(string(decoded), "AWS:") {
		t.Fatalf("expected authorization token to decode to AWS credentials payload")
	}
	if authOut.AuthorizationData[0].ProxyEndpoint == "" {
		t.Fatalf("expected proxy endpoint")
	}

	resp = ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"demo-repo","tags":[{"Key":"env","Value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		Repository struct {
			RepositoryArn string `json:"repositoryArn"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create repository: %v", err)
	}
	if createOut.Repository.RepositoryArn == "" {
		t.Fatalf("expected repository arn")
	}

	resp = ecrRequest(t, ts, "DescribeRepositories", []byte(`{"repositoryNames":["demo-repo"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecrRequest(t, ts, "TagResource", []byte(`{"resourceArn":"`+createOut.Repository.RepositoryArn+`","tags":[{"Key":"team","Value":"platform"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecrRequest(t, ts, "ListTagsForResource", []byte(`{"resourceArn":"`+createOut.Repository.RepositoryArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		Tags []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if len(listTagsOut.Tags) < 2 {
		t.Fatalf("expected at least two tags")
	}

	resp = ecrRequest(t, ts, "UntagResource", []byte(`{"resourceArn":"`+createOut.Repository.RepositoryArn+`","tagKeys":["team"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecrRequest(t, ts, "DeleteRepository", []byte(`{"repositoryName":"demo-repo","force":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecrRequest(t, ts, "DescribeRepositories", []byte(`{"repositoryNames":["demo-repo"]}`))
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestECRStage0OperationCoverage(t *testing.T) {
	if len(ecrOperations) != 58 {
		t.Fatalf("expected 58 ECR operations from docs, got %d", len(ecrOperations))
	}
	if len(ecrOperationByName) != len(ecrOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"GetAuthorizationToken",
		"CreateRepository",
		"DescribeRepositories",
		"DeleteRepository",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
		"PutImage",
		"DescribeImages",
		"BatchGetImage",
	}
	for _, name := range required {
		if _, ok := ecrOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestECRStage0AllDocumentedOperationsAreKnown(t *testing.T) {
	for i, op := range ecrOperations {
		if strings.TrimSpace(op.Name) == "" {
			t.Fatalf("empty operation at index %d", i)
		}
		if _, ok := ecrOperationByName[op.Name]; !ok {
			t.Fatalf("operation %s missing from lookup", op.Name)
		}
	}
}

func TestECRStage0ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"implemented-repo"}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		Repository struct {
			RepositoryArn string `json:"repositoryArn"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create repository: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "GetAuthorizationToken", Body: []byte(`{}`)},
		{Name: "DescribeRepositories", Body: []byte(`{"repositoryNames":["implemented-repo"]}`)},
		{Name: "ListTagsForResource", Body: []byte(`{"resourceArn":"` + createOut.Repository.RepositoryArn + `"}`)},
		{Name: "TagResource", Body: []byte(`{"resourceArn":"` + createOut.Repository.RepositoryArn + `","tags":[{"Key":"env","Value":"test"}]}`)},
		{Name: "UntagResource", Body: []byte(`{"resourceArn":"` + createOut.Repository.RepositoryArn + `","tagKeys":["env"]}`)},
		{Name: "DeleteRepository", Body: []byte(`{"repositoryName":"implemented-repo","force":true}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}

func TestECRStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecrRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestECRStage0SDKClientLifecycle(t *testing.T) {
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

	createOut, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{
		RepositoryName: aws.String("sdk-repo"),
		Tags: []awsecrtypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if createOut.Repository == nil || createOut.Repository.RepositoryArn == nil {
		t.Fatalf("expected repository arn")
	}
	repositoryArn := aws.ToString(createOut.Repository.RepositoryArn)

	authOut, err := client.GetAuthorizationToken(ctx, &awsecr.GetAuthorizationTokenInput{})
	if err != nil {
		t.Fatalf("get authorization token: %v", err)
	}
	if len(authOut.AuthorizationData) == 0 {
		t.Fatalf("expected authorization data")
	}

	describeOut, err := client.DescribeRepositories(ctx, &awsecr.DescribeRepositoriesInput{
		RepositoryNames: []string{"sdk-repo"},
	})
	if err != nil {
		t.Fatalf("describe repositories: %v", err)
	}
	if len(describeOut.Repositories) != 1 {
		t.Fatalf("expected one repository")
	}

	if _, err := client.TagResource(ctx, &awsecr.TagResourceInput{
		ResourceArn: aws.String(repositoryArn),
		Tags: []awsecrtypes.Tag{
			{Key: aws.String("team"), Value: aws.String("platform")},
		},
	}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}

	listTagsOut, err := client.ListTagsForResource(ctx, &awsecr.ListTagsForResourceInput{ResourceArn: aws.String(repositoryArn)})
	if err != nil {
		t.Fatalf("list tags for resource: %v", err)
	}
	if len(listTagsOut.Tags) < 2 {
		t.Fatalf("expected at least two tags")
	}

	if _, err := client.UntagResource(ctx, &awsecr.UntagResourceInput{
		ResourceArn: aws.String(repositoryArn),
		TagKeys:     []string{"team"},
	}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	if _, err := client.DeleteRepository(ctx, &awsecr.DeleteRepositoryInput{
		RepositoryName: aws.String("sdk-repo"),
		Force:          true,
	}); err != nil {
		t.Fatalf("delete repository: %v", err)
	}
}
