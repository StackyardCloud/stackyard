package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	imageID := getenv("STACKYARD_IMAGE_ID", "ami-12345678")

	ctx := context.Background()
	client := newEC2Client(ctx, endpoint)

	fmt.Printf("Stackyard EC2 basic client using %s\n", endpoint)

	runOut, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: types.InstanceTypeT3Micro,
	})
	if err != nil {
		exitf("run instances: %v", err)
	}
	if len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		exitf("run instances: missing instance id")
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)
	logf("created instance: %s", instanceID)

	describeOut, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		exitf("describe instances: %v", err)
	}
	logf("reservations: %d", len(describeOut.Reservations))

	if _, err := client.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		exitf("stop instances: %v", err)
	}
	if _, err := client.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		exitf("start instances: %v", err)
	}

	if _, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		exitf("terminate instances: %v", err)
	}
	logf("terminated instance: %s", instanceID)

	fmt.Println("Done.")
}

func newEC2Client(ctx context.Context, endpoint string) *ec2.Client {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func getenv(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
