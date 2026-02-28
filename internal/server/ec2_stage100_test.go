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

func TestEC2Stage100SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage100"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage100-source"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create source image: %v", err)
	}
	sourceImageID := aws.ToString(createImageOut.ImageId)

	out, err := client.CopyImage(ctx, &awsec2.CopyImageInput{
		Name:          aws.String("stage100-copy"),
		SourceImageId: aws.String(sourceImageID),
		SourceRegion:  aws.String("us-east-1"),
		CopyImageTags: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("copy image: %v", err)
	}
	if out.ImageId == nil || strings.TrimSpace(aws.ToString(out.ImageId)) == "" {
		t.Fatalf("expected image id")
	}
	if !strings.HasPrefix(aws.ToString(out.ImageId), "ami-") {
		t.Fatalf("unexpected copied image id: %q", aws.ToString(out.ImageId))
	}
}

func TestEC2Stage100ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CopyImage",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"Name":          "stage100-copy",
			"SourceImageId": "ami-00000000000000100",
			"SourceRegion":  "us-east-1",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
