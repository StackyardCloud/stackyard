package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
)

func ecsRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonEC2ContainerService_V20141113." + action,
		},
		"ecs",
	)
}

func TestECSStage0OperationCoverage(t *testing.T) {
	if len(ecsOperations) != 70 {
		t.Fatalf("expected 70 ECS operations from docs, got %d", len(ecsOperations))
	}
	if len(ecsOperationByName) != len(ecsOperations) {
		t.Fatalf("expected unique ECS operation names")
	}

	required := []string{
		"PutAccountSetting",
		"PutAccountSettingDefault",
		"ListAccountSettings",
		"DeleteAccountSetting",
		"CreateCluster",
		"DescribeClusters",
		"ListClusters",
		"UpdateClusterSettings",
		"DeleteCluster",
	}
	for _, name := range required {
		if _, ok := ecsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestECSStage0AllDocumentedOperationsAreKnown(t *testing.T) {
	for i, op := range ecsOperations {
		if strings.TrimSpace(op.Name) == "" {
			t.Fatalf("empty operation at index %d", i)
		}
		if _, ok := ecsOperationByName[op.Name]; !ok {
			t.Fatalf("operation %s missing from lookup", op.Name)
		}
	}
}

func TestECSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecsRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestECSStage1AccountAndClusterCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "stage0-cluster"

	actions := []struct {
		name string
		body []byte
	}{
		{name: "PutAccountSetting", body: []byte(`{"name":"containerInsights","value":"enabled"}`)},
		{name: "PutAccountSettingDefault", body: []byte(`{"name":"serviceLongArnFormat","value":"enabled"}`)},
		{name: "ListAccountSettings", body: []byte(`{"effectiveSettings":true}`)},
		{name: "CreateCluster", body: []byte(`{"clusterName":"` + clusterName + `","settings":[{"name":"containerInsights","value":"enabled"}],"tags":[{"key":"env","value":"test"}]}`)},
		{name: "DescribeClusters", body: []byte(`{"clusters":["` + clusterName + `"]}`)},
		{name: "ListClusters", body: []byte(`{}`)},
		{name: "UpdateClusterSettings", body: []byte(`{"cluster":"` + clusterName + `","settings":[{"name":"containerInsights","value":"disabled"}]}`)},
		{name: "DeleteAccountSetting", body: []byte(`{"name":"containerInsights"}`)},
		{name: "DeleteCluster", body: []byte(`{"cluster":"` + clusterName + `"}`)},
	}

	for _, action := range actions {
		resp := ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}

	resp := ecsRequest(t, ts, "DescribeClusters", []byte(`{"clusters":["`+clusterName+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Failures []struct {
			Arn    string `json:"arn"`
			Reason string `json:"reason"`
		} `json:"failures"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe clusters response: %v", err)
	}
	if len(describeOut.Failures) != 1 {
		t.Fatalf("expected one failure after delete, got %d", len(describeOut.Failures))
	}
}

func TestECSStage1SDKCreateClusterResponseShape(t *testing.T) {
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
	client := awsecs.NewFromConfig(cfg, func(o *awsecs.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	out, err := client.CreateCluster(ctx, &awsecs.CreateClusterInput{
		ClusterName: aws.String("sdk-shape-cluster"),
	})
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	if out.Cluster == nil {
		t.Fatalf("create cluster: missing cluster in response")
	}
	if got := aws.ToString(out.Cluster.ClusterName); got != "sdk-shape-cluster" {
		t.Fatalf("unexpected cluster name: %q", got)
	}
	if len(out.Cluster.Statistics) != 0 {
		t.Fatalf("expected empty statistics list, got %d entries", len(out.Cluster.Statistics))
	}
}
