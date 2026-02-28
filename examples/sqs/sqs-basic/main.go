package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	queueName := getenv("STACKYARD_QUEUE", "jobs-basic")

	ctx := context.Background()
	client := newSQSClient(ctx, endpoint)

	fmt.Printf("Stackyard SQS basic client using %s\n", endpoint)

	queueURL, err := ensureQueue(ctx, client, endpoint, queueName)
	if err != nil {
		exitf("create queue: %v", err)
	}
	logf("queue url: %s", queueURL)

	if err := sendMessage(ctx, client, queueURL, "run build"); err != nil {
		exitf("send message: %v", err)
	}
	logf("sent message")

	msg, err := receiveMessage(ctx, client, queueURL)
	if err != nil {
		exitf("receive message: %v", err)
	}

	fmt.Printf("Received message: %q\n", msg)
	fmt.Println("Done.")
}

func newSQSClient(ctx context.Context, endpoint string) *sqs.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == sqs.ServiceID {
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

	return sqs.NewFromConfig(cfg)
}

func ensureQueue(ctx context.Context, client *sqs.Client, endpoint, name string) (string, error) {
	resp, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.QueueUrl), nil
}

func sendMessage(ctx context.Context, client *sqs.Client, queueURL, body string) error {
	_, err := client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	return err
}

func receiveMessage(ctx context.Context, client *sqs.Client, queueURL string) (string, error) {
	resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no messages received")
	}
	return aws.ToString(resp.Messages[0].Body), nil
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
