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
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	queueName := getenv("STACKYARD_QUEUE", "jobs-advanced")

	ctx := context.Background()
	client := newSQSClient(ctx, endpoint)

	fmt.Printf("Stackyard SQS advanced client using %s\n", endpoint)

	queueURL, err := ensureQueue(ctx, client, endpoint, queueName)
	if err != nil {
		exitf("create queue: %v", err)
	}
	logf("queue url: %s", queueURL)

	if err := updateQueueAttributes(ctx, client, queueURL); err != nil {
		exitf("update attributes: %v", err)
	}
	logf("updated queue attributes")

	if err := listQueues(ctx, client); err != nil {
		exitf("list queues: %v", err)
	}

	if err := sendMessage(ctx, client, queueURL, "task-1"); err != nil {
		exitf("send message 1: %v", err)
	}
	logf("sent message task-1")
	if err := sendMessage(ctx, client, queueURL, "task-2"); err != nil {
		exitf("send message 2: %v", err)
	}
	logf("sent message task-2")

	if err := sendBatch(ctx, client, queueURL); err != nil {
		exitf("send batch: %v", err)
	}
	logf("sent batch messages")

	msg, err := receiveMessage(ctx, client, queueURL)
	if err != nil {
		exitf("receive message: %v", err)
	}
	fmt.Printf("Received message: %q\n", msg)

	if err := changeVisibility(ctx, client, queueURL); err != nil {
		exitf("change visibility: %v", err)
	}
	logf("changed message visibility")

	if err := drainAndDelete(ctx, client, queueURL); err != nil {
		exitf("delete message: %v", err)
	}
	logf("deleted one message")

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
		Attributes: map[string]string{
			"VisibilityTimeout": "10",
		},
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
		MessageAttributes: map[string]types.MessageAttributeValue{
			"source": {
				DataType:    aws.String("String"),
				StringValue: aws.String("stackyard"),
			},
		},
	})
	return err
}

func receiveMessage(ctx context.Context, client *sqs.Client, queueURL string) (string, error) {
	resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		MessageAttributeNames: []string{
			"All",
		},
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no messages received")
	}
	return aws.ToString(resp.Messages[0].Body), nil
}

func updateQueueAttributes(ctx context.Context, client *sqs.Client, queueURL string) error {
	_, err := client.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		Attributes: map[string]string{
			"VisibilityTimeout":             "15",
			"ReceiveMessageWaitTimeSeconds": "1",
		},
	})
	return err
}

func listQueues(ctx context.Context, client *sqs.Client) error {
	resp, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return err
	}
	logf("queues: %d", len(resp.QueueUrls))
	return nil
}

func sendBatch(ctx context.Context, client *sqs.Client, queueURL string) error {
	_, err := client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries: []types.SendMessageBatchRequestEntry{
			{
				Id:          aws.String("batch-1"),
				MessageBody: aws.String("batch task 1"),
			},
			{
				Id:          aws.String("batch-2"),
				MessageBody: aws.String("batch task 2"),
			},
		},
	})
	return err
}

func changeVisibility(ctx context.Context, client *sqs.Client, queueURL string) error {
	resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		return err
	}
	if len(resp.Messages) == 0 {
		return nil
	}
	_, err = client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     resp.Messages[0].ReceiptHandle,
		VisibilityTimeout: 0,
	})
	return err
}

func drainAndDelete(ctx context.Context, client *sqs.Client, queueURL string) error {
	resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		return err
	}
	if len(resp.Messages) == 0 {
		logf("no messages to delete")
		return nil
	}
	_, err = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: resp.Messages[0].ReceiptHandle,
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
