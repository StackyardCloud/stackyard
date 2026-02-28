package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	source := getenv("STACKYARD_SOURCE_EMAIL", "sender@example.com")
	destination := getenv("STACKYARD_DEST_EMAIL", "recipient@example.com")

	ctx := context.Background()
	client := newSESClient(ctx, endpoint)

	fmt.Printf("Stackyard SES basic client using %s\n", endpoint)

	if err := verifyEmailIdentity(ctx, client, source); err != nil {
		exitf("verify email identity: %v", err)
	}
	logf("verified identity: %s", source)

	messageID, err := sendEmail(ctx, client, source, destination)
	if err != nil {
		exitf("send email: %v", err)
	}
	logf("sent email message id: %s", messageID)

	fmt.Println("Done.")
}

func newSESClient(ctx context.Context, endpoint string) *ses.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == ses.ServiceID {
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

	return ses.NewFromConfig(cfg)
}

func verifyEmailIdentity(ctx context.Context, client *ses.Client, email string) error {
	_, err := client.VerifyEmailIdentity(ctx, &ses.VerifyEmailIdentityInput{EmailAddress: aws.String(email)})
	return err
}

func sendEmail(ctx context.Context, client *ses.Client, source, destination string) (string, error) {
	resp, err := client.SendEmail(ctx, &ses.SendEmailInput{
		Source: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: []string{destination},
		},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String("Stackyard SES Basic")},
			Body: &types.Body{
				Text: &types.Content{Data: aws.String("hello from stackyard ses basic example")},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.MessageId), nil
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
