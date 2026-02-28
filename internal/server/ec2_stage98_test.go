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

func TestEC2Stage98SDKLifecycle(t *testing.T) {
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

	out, err := client.ConfirmProductInstance(ctx, &awsec2.ConfirmProductInstanceInput{
		InstanceId:  aws.String("i-00000000000000098"),
		ProductCode: aws.String("prod-00000000000000098"),
	})
	if err != nil {
		t.Fatalf("confirm product instance: %v", err)
	}
	if !aws.ToBool(out.Return) {
		t.Fatalf("expected return=true")
	}
	if aws.ToString(out.OwnerId) != "123456789012" {
		t.Fatalf("unexpected owner id: %q", aws.ToString(out.OwnerId))
	}
}

func TestEC2Stage98ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ConfirmProductInstance",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"InstanceId":  "i-00000000000000098",
			"ProductCode": "prod-00000000000000098",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
