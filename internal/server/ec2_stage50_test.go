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

func TestEC2Stage50SDKLifecycle(t *testing.T) {
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

	modifyOut, err := client.ModifyVpcEndpointConnectionNotification(ctx, &awsec2.ModifyVpcEndpointConnectionNotificationInput{
		ConnectionNotificationId:  aws.String("vpce-nfn-00000000"),
		ConnectionNotificationArn: aws.String("arn:aws:sns:us-east-1:123456789012:stage50-topic"),
		ConnectionEvents:          []string{"Accept", "Reject"},
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint connection notification: %v", err)
	}
	if !aws.ToBool(modifyOut.ReturnValue) {
		t.Fatalf("expected modify vpc endpoint connection notification return value true")
	}
}

func TestEC2Stage50ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcEndpointConnectionNotification",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ConnectionNotificationId":  "vpce-nfn-00000000",
			"ConnectionNotificationArn": "arn:aws:sns:us-east-1:123456789012:stage50-topic",
			"ConnectionEvents.1":        "Accept",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
