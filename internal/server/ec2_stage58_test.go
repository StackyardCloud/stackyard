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

func TestEC2Stage58SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage58/1234567890abcdef"},
	})
	if err != nil {
		t.Fatalf("create vpc endpoint service configuration: %v", err)
	}
	if createOut.ServiceConfiguration == nil || createOut.ServiceConfiguration.ServiceId == nil {
		t.Fatalf("expected created service configuration id")
	}
	createdServiceID := aws.ToString(createOut.ServiceConfiguration.ServiceId)

	deleteOut, err := client.DeleteVpcEndpointServiceConfigurations(ctx, &awsec2.DeleteVpcEndpointServiceConfigurationsInput{
		ServiceIds: []string{
			createdServiceID,
			"vpce-svc-does-not-exist",
		},
	})
	if err != nil {
		t.Fatalf("delete vpc endpoint service configurations: %v", err)
	}
	if len(deleteOut.Unsuccessful) != 1 {
		t.Fatalf("expected one unsuccessful item, got %d", len(deleteOut.Unsuccessful))
	}
	if aws.ToString(deleteOut.Unsuccessful[0].ResourceId) != "vpce-svc-does-not-exist" {
		t.Fatalf("unexpected unsuccessful resource id: %q", aws.ToString(deleteOut.Unsuccessful[0].ResourceId))
	}
}

func TestEC2Stage58ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteVpcEndpointServiceConfigurations",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ServiceId.1": "vpce-svc-00000000",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
