package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	topicName := getenv("STACKYARD_TOPIC", "demo-sns-basic")

	ctx := context.Background()
	client := newSNSClient(ctx, endpoint)

	fmt.Printf("Stackyard SNS basic client using %s\n", endpoint)

	topicArn, err := createTopic(ctx, client, topicName)
	if err != nil {
		exitf("create topic: %v", err)
	}
	logf("created topic: %s", topicArn)

	if err := publishMessage(ctx, client, topicArn, "hello from stackyard sns basic example"); err != nil {
		exitf("publish message: %v", err)
	}
	logf("published message")

	fmt.Println("Done.")
}

func newSNSClient(ctx context.Context, endpoint string) *sns.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == sns.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return sns.NewFromConfig(cfg)
}

func createTopic(ctx context.Context, client *sns.Client, name string) (string, error) {
	resp, err := client.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(name)})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.TopicArn), nil
}

func publishMessage(ctx context.Context, client *sns.Client, topicArn, message string) error {
	_, err := client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicArn),
		Message:  aws.String(message),
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

func init() {
	_ = time.Now()
}
