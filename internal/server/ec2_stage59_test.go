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
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage59SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateVpcEndpoint(ctx, &awsec2.CreateVpcEndpointInput{
		VpcId:             aws.String("vpc-00000001"),
		ServiceName:       aws.String("com.amazonaws.us-east-1.s3"),
		VpcEndpointType:   awsec2types.VpcEndpointTypeInterface,
		SubnetIds:         []string{"subnet-00000001"},
		SecurityGroupIds:  []string{"sg-00000000"},
		PrivateDnsEnabled: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("create vpc endpoint: %v", err)
	}
	if createOut.VpcEndpoint == nil || createOut.VpcEndpoint.VpcEndpointId == nil {
		t.Fatalf("expected created vpc endpoint id")
	}
	createdEndpointID := aws.ToString(createOut.VpcEndpoint.VpcEndpointId)

	deleteOut, err := client.DeleteVpcEndpoints(ctx, &awsec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{
			createdEndpointID,
			"vpce-does-not-exist",
		},
	})
	if err != nil {
		t.Fatalf("delete vpc endpoints: %v", err)
	}
	if len(deleteOut.Unsuccessful) != 1 {
		t.Fatalf("expected one unsuccessful item, got %d", len(deleteOut.Unsuccessful))
	}
	if aws.ToString(deleteOut.Unsuccessful[0].ResourceId) != "vpce-does-not-exist" {
		t.Fatalf("unexpected unsuccessful resource id: %q", aws.ToString(deleteOut.Unsuccessful[0].ResourceId))
	}
}

func TestEC2Stage59ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteVpcEndpoints",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"VpcEndpointId.1": "vpce-00000000",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
