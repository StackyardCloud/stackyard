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

func TestEC2Stage86SDKLifecycle(t *testing.T) {
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

	out, err := client.AssociateIpamResourceDiscovery(ctx, &awsec2.AssociateIpamResourceDiscoveryInput{
		IpamId:                  aws.String("ipam-00000000000000086"),
		IpamResourceDiscoveryId: aws.String("ipam-rd-00000000000000086"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeIpamResourceDiscoveryAssociation,
				Tags: []awsec2types.Tag{
					{Key: aws.String("env"), Value: aws.String("stage86")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("associate ipam resource discovery: %v", err)
	}
	if out.IpamResourceDiscoveryAssociation == nil {
		t.Fatalf("expected ipam resource discovery association in response")
	}
	assoc := out.IpamResourceDiscoveryAssociation
	if aws.ToString(assoc.IpamId) != "ipam-00000000000000086" {
		t.Fatalf("unexpected ipam id: %q", aws.ToString(assoc.IpamId))
	}
	if aws.ToString(assoc.IpamResourceDiscoveryId) != "ipam-rd-00000000000000086" {
		t.Fatalf("unexpected ipam resource discovery id: %q", aws.ToString(assoc.IpamResourceDiscoveryId))
	}
	if assoc.State != awsec2types.IpamResourceDiscoveryAssociationStateAssociateComplete {
		t.Fatalf("unexpected association state: %q", assoc.State)
	}
	if assoc.ResourceDiscoveryStatus != awsec2types.IpamAssociatedResourceDiscoveryStatusActive {
		t.Fatalf("unexpected resource discovery status: %q", assoc.ResourceDiscoveryStatus)
	}
	if len(assoc.Tags) != 1 || aws.ToString(assoc.Tags[0].Key) != "env" || aws.ToString(assoc.Tags[0].Value) != "stage86" {
		t.Fatalf("unexpected association tags: %#v", assoc.Tags)
	}
}

func TestEC2Stage86ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateIpamResourceDiscovery",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"IpamId":                          "ipam-00000000000000086",
			"IpamResourceDiscoveryId":         "ipam-rd-00000000000000086",
			"TagSpecification.1.ResourceType": "ipam-resource-discovery-association",
			"TagSpecification.1.Tag.1.Key":    "env",
			"TagSpecification.1.Tag.1.Value":  "stage86",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
