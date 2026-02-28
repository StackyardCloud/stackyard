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

func TestEC2Stage72SDKLifecycle(t *testing.T) {
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

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage72"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instance: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage72-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	enableDeregOut, err := client.EnableImageDeregistrationProtection(ctx, &awsec2.EnableImageDeregistrationProtectionInput{
		ImageId:      aws.String(imageID),
		WithCooldown: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("enable image deregistration protection: %v", err)
	}
	if aws.ToString(enableDeregOut.Return) != "true" {
		t.Fatalf("unexpected enable deregistration return: %q", aws.ToString(enableDeregOut.Return))
	}

	describeEnabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{ImageIds: []string{imageID}})
	if err != nil {
		t.Fatalf("describe images after enabling deregistration protection: %v", err)
	}
	if len(describeEnabledOut.Images) != 1 {
		t.Fatalf("expected one image after enabling deregistration protection, got %d", len(describeEnabledOut.Images))
	}
	if aws.ToString(describeEnabledOut.Images[0].DeregistrationProtection) != "enabled" {
		t.Fatalf("unexpected deregistration protection after enable: %q", aws.ToString(describeEnabledOut.Images[0].DeregistrationProtection))
	}

	disableDeregOut, err := client.DisableImageDeregistrationProtection(ctx, &awsec2.DisableImageDeregistrationProtectionInput{
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("disable image deregistration protection: %v", err)
	}
	if aws.ToString(disableDeregOut.Return) != "true" {
		t.Fatalf("unexpected disable deregistration return: %q", aws.ToString(disableDeregOut.Return))
	}

	describeDisabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{ImageIds: []string{imageID}})
	if err != nil {
		t.Fatalf("describe images after disabling deregistration protection: %v", err)
	}
	if len(describeDisabledOut.Images) != 1 {
		t.Fatalf("expected one image after disabling deregistration protection, got %d", len(describeDisabledOut.Images))
	}
	if aws.ToString(describeDisabledOut.Images[0].DeregistrationProtection) != "disabled" {
		t.Fatalf("unexpected deregistration protection after disable: %q", aws.ToString(describeDisabledOut.Images[0].DeregistrationProtection))
	}

	getSnapshotStateOut, err := client.GetSnapshotBlockPublicAccessState(ctx, &awsec2.GetSnapshotBlockPublicAccessStateInput{})
	if err != nil {
		t.Fatalf("get snapshot block public access state: %v", err)
	}
	if getSnapshotStateOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected initial snapshot managed by: %q", getSnapshotStateOut.ManagedBy)
	}
	if getSnapshotStateOut.State != awsec2types.SnapshotBlockPublicAccessStateUnblocked {
		t.Fatalf("unexpected initial snapshot state: %q", getSnapshotStateOut.State)
	}

	enableSnapshotStateOut, err := client.EnableSnapshotBlockPublicAccess(ctx, &awsec2.EnableSnapshotBlockPublicAccessInput{
		State: awsec2types.SnapshotBlockPublicAccessStateBlockAllSharing,
	})
	if err != nil {
		t.Fatalf("enable snapshot block public access: %v", err)
	}
	if enableSnapshotStateOut.State != awsec2types.SnapshotBlockPublicAccessStateBlockAllSharing {
		t.Fatalf("unexpected enabled snapshot state: %q", enableSnapshotStateOut.State)
	}

	getEnabledSnapshotStateOut, err := client.GetSnapshotBlockPublicAccessState(ctx, &awsec2.GetSnapshotBlockPublicAccessStateInput{})
	if err != nil {
		t.Fatalf("get snapshot block public access state after enable: %v", err)
	}
	if getEnabledSnapshotStateOut.State != awsec2types.SnapshotBlockPublicAccessStateBlockAllSharing {
		t.Fatalf("unexpected snapshot state after enable: %q", getEnabledSnapshotStateOut.State)
	}
	if getEnabledSnapshotStateOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected snapshot managed by after enable: %q", getEnabledSnapshotStateOut.ManagedBy)
	}

	disableSnapshotStateOut, err := client.DisableSnapshotBlockPublicAccess(ctx, &awsec2.DisableSnapshotBlockPublicAccessInput{})
	if err != nil {
		t.Fatalf("disable snapshot block public access: %v", err)
	}
	if disableSnapshotStateOut.State != awsec2types.SnapshotBlockPublicAccessStateUnblocked {
		t.Fatalf("unexpected snapshot state after disable: %q", disableSnapshotStateOut.State)
	}
}

func TestEC2Stage72ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableImageDeregistrationProtection",
		"EnableImageDeregistrationProtection",
		"DisableSnapshotBlockPublicAccess",
		"EnableSnapshotBlockPublicAccess",
		"GetSnapshotBlockPublicAccessState",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisableImageDeregistrationProtection":
			params["ImageId"] = "ami-00000000"
		case "EnableImageDeregistrationProtection":
			params["ImageId"] = "ami-00000000"
			params["WithCooldown"] = "true"
		case "EnableSnapshotBlockPublicAccess":
			params["State"] = "block-new-sharing"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
