package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage80SDKLifecycle(t *testing.T) {
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

	out, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceType:     aws.String("t3.micro"),
		AutoPlacement:    awsec2types.AutoPlacementOff,
		HostRecovery:     awsec2types.HostRecoveryOff,
		HostMaintenance:  awsec2types.HostMaintenanceOff,
		Quantity:         aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("allocate hosts: %v", err)
	}
	if len(out.HostIds) != 2 {
		t.Fatalf("expected 2 host ids, got %d", len(out.HostIds))
	}
	for _, hostID := range out.HostIds {
		if !strings.HasPrefix(hostID, "h-") {
			t.Fatalf("unexpected host id format: %q", hostID)
		}
	}
}

func TestEC2Stage80ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AllocateHosts",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"AvailabilityZone": "us-east-1a",
			"InstanceType":     "t3.micro",
			"Quantity":         "1",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
