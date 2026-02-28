package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	zone := getenv("STACKYARD_AVAILABILITY_ZONE", "us-east-1a")
	instance := getenv("STACKYARD_INSTANCE_NAME", "lightsail-basic-instance")

	ctx := context.Background()
	client := newLightsailClient(ctx, endpoint)

	fmt.Printf("Stackyard Lightsail basic client using %s\n", endpoint)

	if err := createInstance(ctx, client, zone, instance); err != nil {
		exitf("create instance: %v", err)
	}
	logf("created instance: %s", instance)

	state, err := getInstanceState(ctx, client, instance)
	if err != nil {
		exitf("get instance state: %v", err)
	}
	logf("instance state: %s", state)

	count, err := listInstances(ctx, client)
	if err != nil {
		exitf("list instances: %v", err)
	}
	logf("instances: %d", count)

	if err := deleteInstance(ctx, client, instance); err != nil {
		exitf("delete instance: %v", err)
	}
	logf("deleted instance: %s", instance)

	fmt.Println("Done.")
}

func newLightsailClient(ctx context.Context, endpoint string) *lightsail.Client {
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

	return lightsail.NewFromConfig(cfg, func(o *lightsail.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func createInstance(ctx context.Context, client *lightsail.Client, zone, name string) error {
	_, err := client.CreateInstances(ctx, &lightsail.CreateInstancesInput{
		AvailabilityZone: aws.String(zone),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{name},
	})
	return err
}

func getInstanceState(ctx context.Context, client *lightsail.Client, name string) (string, error) {
	out, err := client.GetInstanceState(ctx, &lightsail.GetInstanceStateInput{
		InstanceName: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	if out.State == nil || out.State.Name == nil {
		return "", nil
	}
	return aws.ToString(out.State.Name), nil
}

func listInstances(ctx context.Context, client *lightsail.Client) (int, error) {
	out, err := client.GetInstances(ctx, &lightsail.GetInstancesInput{})
	if err != nil {
		return 0, err
	}
	return len(out.Instances), nil
}

func deleteInstance(ctx context.Context, client *lightsail.Client, name string) error {
	_, err := client.DeleteInstance(ctx, &lightsail.DeleteInstanceInput{
		InstanceName: aws.String(name),
	})
	return err
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
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
