package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestEC2Stage94SDKLifecycle(t *testing.T) {
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
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	out, err := client.CancelImportTask(ctx, &awsec2.CancelImportTaskInput{
		ImportTaskId: aws.String("import-ami-00000000000000094"),
		CancelReason: aws.String("stage94 cancel"),
	})
	if err != nil {
		t.Fatalf("cancel import task: %v", err)
	}
	if aws.ToString(out.ImportTaskId) != "import-ami-00000000000000094" {
		t.Fatalf("unexpected import task id: %q", aws.ToString(out.ImportTaskId))
	}
	if aws.ToString(out.PreviousState) != "active" {
		t.Fatalf("unexpected previous state: %q", aws.ToString(out.PreviousState))
	}
	if aws.ToString(out.State) != "cancelled" {
		t.Fatalf("unexpected state: %q", aws.ToString(out.State))
	}
}

func TestEC2Stage94ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelImportTask",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"ImportTaskId": "import-ami-00000000000000094",
			"CancelReason": "stage94 cancel",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
