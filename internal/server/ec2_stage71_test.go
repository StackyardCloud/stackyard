package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage71SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage71"),
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
		Name:       aws.String("stage71-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	getInitialStateOut, err := client.GetImageBlockPublicAccessState(ctx, &awsec2.GetImageBlockPublicAccessStateInput{})
	if err != nil {
		t.Fatalf("get image block public access state: %v", err)
	}
	if aws.ToString(getInitialStateOut.ImageBlockPublicAccessState) != "unblocked" {
		t.Fatalf("unexpected initial image block public access state: %q", aws.ToString(getInitialStateOut.ImageBlockPublicAccessState))
	}

	enableBlockOut, err := client.EnableImageBlockPublicAccess(ctx, &awsec2.EnableImageBlockPublicAccessInput{
		ImageBlockPublicAccessState: awsec2types.ImageBlockPublicAccessEnabledStateBlockNewSharing,
	})
	if err != nil {
		t.Fatalf("enable image block public access: %v", err)
	}
	if enableBlockOut.ImageBlockPublicAccessState != awsec2types.ImageBlockPublicAccessEnabledStateBlockNewSharing {
		t.Fatalf("unexpected enabled image block public access state: %q", enableBlockOut.ImageBlockPublicAccessState)
	}

	getEnabledStateOut, err := client.GetImageBlockPublicAccessState(ctx, &awsec2.GetImageBlockPublicAccessStateInput{})
	if err != nil {
		t.Fatalf("get image block public access state after enable: %v", err)
	}
	if aws.ToString(getEnabledStateOut.ImageBlockPublicAccessState) != "block-new-sharing" {
		t.Fatalf("unexpected enabled image block public access state: %q", aws.ToString(getEnabledStateOut.ImageBlockPublicAccessState))
	}

	disableBlockOut, err := client.DisableImageBlockPublicAccess(ctx, &awsec2.DisableImageBlockPublicAccessInput{})
	if err != nil {
		t.Fatalf("disable image block public access: %v", err)
	}
	if disableBlockOut.ImageBlockPublicAccessState != awsec2types.ImageBlockPublicAccessDisabledStateUnblocked {
		t.Fatalf("unexpected disabled image block public access state: %q", disableBlockOut.ImageBlockPublicAccessState)
	}

	deprecateAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	enableDeprecationOut, err := client.EnableImageDeprecation(ctx, &awsec2.EnableImageDeprecationInput{
		ImageId:     aws.String(imageID),
		DeprecateAt: aws.Time(deprecateAt),
	})
	if err != nil {
		t.Fatalf("enable image deprecation: %v", err)
	}
	if !aws.ToBool(enableDeprecationOut.Return) {
		t.Fatalf("expected true return value from enable image deprecation")
	}

	describeDeprecationEnabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{
		ImageIds: []string{imageID},
	})
	if err != nil {
		t.Fatalf("describe images after enabling deprecation: %v", err)
	}
	if len(describeDeprecationEnabledOut.Images) != 1 {
		t.Fatalf("expected one image after enabling deprecation, got %d", len(describeDeprecationEnabledOut.Images))
	}
	expectedDeprecationTime := deprecateAt.Format(timeRFC3339UTC)
	if aws.ToString(describeDeprecationEnabledOut.Images[0].DeprecationTime) != expectedDeprecationTime {
		t.Fatalf("unexpected deprecation time: %q", aws.ToString(describeDeprecationEnabledOut.Images[0].DeprecationTime))
	}

	disableDeprecationOut, err := client.DisableImageDeprecation(ctx, &awsec2.DisableImageDeprecationInput{
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("disable image deprecation: %v", err)
	}
	if !aws.ToBool(disableDeprecationOut.Return) {
		t.Fatalf("expected true return value from disable image deprecation")
	}

	describeDeprecationDisabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{
		ImageIds: []string{imageID},
	})
	if err != nil {
		t.Fatalf("describe images after disabling deprecation: %v", err)
	}
	if len(describeDeprecationDisabledOut.Images) != 1 {
		t.Fatalf("expected one image after disabling deprecation, got %d", len(describeDeprecationDisabledOut.Images))
	}
	if describeDeprecationDisabledOut.Images[0].DeprecationTime != nil {
		t.Fatalf("expected deprecation time to be cleared")
	}
}

func TestEC2Stage71ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableImageBlockPublicAccess",
		"DisableImageDeprecation",
		"EnableImageBlockPublicAccess",
		"EnableImageDeprecation",
		"GetImageBlockPublicAccessState",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisableImageDeprecation":
			params["ImageId"] = "ami-00000000"
		case "EnableImageBlockPublicAccess":
			params["ImageBlockPublicAccessState"] = "block-new-sharing"
		case "EnableImageDeprecation":
			params["ImageId"] = "ami-00000000"
			params["DeprecateAt"] = time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
