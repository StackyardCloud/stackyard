package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage8SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage8"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instance: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId:  aws.String(instanceID),
		Name:        aws.String("stage8-image"),
		Description: aws.String("stage8 description"),
		NoReboot:    aws.Bool(true),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeImage,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage8")}},
			},
		},
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	describeImagesOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{
		ImageIds: []string{imageID},
		Owners:   []string{"self"},
	})
	if err != nil {
		t.Fatalf("describe images: %v", err)
	}
	if len(describeImagesOut.Images) != 1 {
		t.Fatalf("expected one image")
	}

	if _, err := client.ModifyImageAttribute(ctx, &awsec2.ModifyImageAttributeInput{
		ImageId:     aws.String(imageID),
		Attribute:   aws.String(string(awsec2types.ImageAttributeNameDescription)),
		Description: &awsec2types.AttributeValue{Value: aws.String("stage8 updated")},
	}); err != nil {
		t.Fatalf("modify image attribute description: %v", err)
	}
	if _, err := client.ModifyImageAttribute(ctx, &awsec2.ModifyImageAttributeInput{
		ImageId:   aws.String(imageID),
		Attribute: aws.String(string(awsec2types.ImageAttributeNameLaunchPermission)),
		LaunchPermission: &awsec2types.LaunchPermissionModifications{
			Add: []awsec2types.LaunchPermission{{UserId: aws.String("111122223333")}},
		},
	}); err != nil {
		t.Fatalf("modify image launch permission: %v", err)
	}

	describeDescriptionOut, err := client.DescribeImageAttribute(ctx, &awsec2.DescribeImageAttributeInput{
		ImageId:   aws.String(imageID),
		Attribute: awsec2types.ImageAttributeNameDescription,
	})
	if err != nil || describeDescriptionOut.Description == nil || aws.ToString(describeDescriptionOut.Description.Value) != "stage8 updated" {
		t.Fatalf("describe image attribute description: %v", err)
	}

	describeLaunchPermissionOut, err := client.DescribeImageAttribute(ctx, &awsec2.DescribeImageAttributeInput{
		ImageId:   aws.String(imageID),
		Attribute: awsec2types.ImageAttributeNameLaunchPermission,
	})
	if err != nil {
		t.Fatalf("describe image launch permission: %v", err)
	}
	if len(describeLaunchPermissionOut.LaunchPermissions) != 1 || aws.ToString(describeLaunchPermissionOut.LaunchPermissions[0].UserId) != "111122223333" {
		t.Fatalf("unexpected launch permissions: %+v", describeLaunchPermissionOut.LaunchPermissions)
	}

	if _, err := client.ResetImageAttribute(ctx, &awsec2.ResetImageAttributeInput{
		ImageId:   aws.String(imageID),
		Attribute: awsec2types.ResetImageAttributeNameLaunchPermission,
	}); err != nil {
		t.Fatalf("reset image attribute: %v", err)
	}
	describeLaunchPermissionOutAfterReset, err := client.DescribeImageAttribute(ctx, &awsec2.DescribeImageAttributeInput{
		ImageId:   aws.String(imageID),
		Attribute: awsec2types.ImageAttributeNameLaunchPermission,
	})
	if err != nil {
		t.Fatalf("describe image launch permission after reset: %v", err)
	}
	if len(describeLaunchPermissionOutAfterReset.LaunchPermissions) != 0 {
		t.Fatalf("expected no launch permissions after reset")
	}

	if _, err := client.DeregisterImage(ctx, &awsec2.DeregisterImageInput{ImageId: aws.String(imageID)}); err != nil {
		t.Fatalf("deregister image: %v", err)
	}
	describeImagesOutAfterDelete, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{ImageIds: []string{imageID}})
	if err != nil {
		t.Fatalf("describe images after deregister: %v", err)
	}
	if len(describeImagesOutAfterDelete.Images) != 0 {
		t.Fatalf("expected no images after deregister")
	}
}

func TestEC2Stage8ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateImage",
		"DescribeImages",
		"DeregisterImage",
		"DescribeImageAttribute",
		"ModifyImageAttribute",
		"ResetImageAttribute",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateImage":
			params["InstanceId"] = "i-00000001"
			params["Name"] = "stage8-image-" + strconv.Itoa(idx)
		case "DescribeImages":
			params["ImageId.1"] = "ami-00000001"
		case "DeregisterImage":
			params["ImageId"] = "ami-00000001"
		case "DescribeImageAttribute":
			params["ImageId"] = "ami-00000001"
			params["Attribute"] = "description"
		case "ModifyImageAttribute":
			params["ImageId"] = "ami-00000001"
			params["Attribute"] = "description"
			params["Description.Value"] = "updated"
		case "ResetImageAttribute":
			params["ImageId"] = "ami-00000001"
			params["Attribute"] = "launchPermission"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
