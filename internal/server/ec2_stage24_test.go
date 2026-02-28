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

func TestEC2Stage24SDKLifecycle(t *testing.T) {
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

	allocateOut, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{Domain: awsec2types.DomainTypeVpc})
	if err != nil || allocateOut.AllocationId == nil || allocateOut.PublicIp == nil {
		t.Fatalf("allocate address: %v", err)
	}
	allocationID := aws.ToString(allocateOut.AllocationId)
	publicIP := aws.ToString(allocateOut.PublicIp)

	modifyAddressAttributeOut, err := client.ModifyAddressAttribute(ctx, &awsec2.ModifyAddressAttributeInput{
		AllocationId: aws.String(allocationID),
		DomainName:   aws.String("mail.example.com"),
	})
	if err != nil || modifyAddressAttributeOut.Address == nil || aws.ToString(modifyAddressAttributeOut.Address.PtrRecord) != "mail.example.com" {
		t.Fatalf("modify address attribute: %v", err)
	}

	resetAddressAttributeOut, err := client.ResetAddressAttribute(ctx, &awsec2.ResetAddressAttributeInput{
		AllocationId: aws.String(allocationID),
		Attribute:    awsec2types.AddressAttributeNameDomainName,
	})
	if err != nil || resetAddressAttributeOut.Address == nil || aws.ToString(resetAddressAttributeOut.Address.PtrRecord) == "" {
		t.Fatalf("reset address attribute: %v", err)
	}

	moveAddressToVpcOut, err := client.MoveAddressToVpc(ctx, &awsec2.MoveAddressToVpcInput{PublicIp: aws.String(publicIP)})
	if err != nil || moveAddressToVpcOut.AllocationId == nil || moveAddressToVpcOut.Status != awsec2types.StatusInVpc {
		t.Fatalf("move address to vpc: %v", err)
	}

	restoreAddressToClassicOut, err := client.RestoreAddressToClassic(ctx, &awsec2.RestoreAddressToClassicInput{PublicIp: aws.String(publicIP)})
	if err != nil || restoreAddressToClassicOut.PublicIp == nil || restoreAddressToClassicOut.Status != awsec2types.StatusInClassic {
		t.Fatalf("restore address to classic: %v", err)
	}

	moveAddressToVpcOut, err = client.MoveAddressToVpc(ctx, &awsec2.MoveAddressToVpcInput{PublicIp: aws.String(publicIP)})
	if err != nil || moveAddressToVpcOut.Status != awsec2types.StatusInVpc {
		t.Fatalf("move address to vpc after restore: %v", err)
	}
}

func TestEC2Stage24ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyAddressAttribute",
		"MoveAddressToVpc",
		"ResetAddressAttribute",
		"RestoreAddressToClassic",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "ModifyAddressAttribute":
			params["AllocationId"] = "eipalloc-00000001"
			params["DomainName"] = "mail.example.com"
		case "MoveAddressToVpc":
			params["PublicIp"] = "203.0.113.10"
		case "ResetAddressAttribute":
			params["AllocationId"] = "eipalloc-00000001"
			params["Attribute"] = "domain-name"
		case "RestoreAddressToClassic":
			params["PublicIp"] = "203.0.113.10"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
