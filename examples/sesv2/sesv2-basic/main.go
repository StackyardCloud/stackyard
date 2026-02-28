package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	source := getenv("STACKYARD_SOURCE_EMAIL", "sender@example.com")
	destination := getenv("STACKYARD_DEST_EMAIL", "recipient@example.com")

	ctx := context.Background()
	client := newSESV2Client(ctx, endpoint)

	fmt.Printf("Stackyard SESv2 basic client using %s\n", endpoint)

	if err := createEmailIdentity(ctx, client, source); err != nil {
		exitf("create email identity: %v", err)
	}
	logf("created identity: %s", source)

	identityType, err := getEmailIdentity(ctx, client, source)
	if err != nil {
		exitf("get email identity: %v", err)
	}
	logf("identity type: %s", identityType)

	messageID, err := sendEmail(ctx, client, source, destination)
	if err != nil {
		exitf("send email: %v", err)
	}
	logf("message id: %s", messageID)

	count, err := listEmailIdentities(ctx, client)
	if err != nil {
		exitf("list email identities: %v", err)
	}
	logf("identities: %d", count)

	fmt.Println("Done.")
}

func newSESV2Client(ctx context.Context, endpoint string) *sesv2.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == sesv2.ServiceID {
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

	return sesv2.NewFromConfig(cfg)
}

func createEmailIdentity(ctx context.Context, client *sesv2.Client, identity string) error {
	_, err := client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(identity),
	})
	return err
}

func getEmailIdentity(ctx context.Context, client *sesv2.Client, identity string) (types.IdentityType, error) {
	resp, err := client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{EmailIdentity: aws.String(identity)})
	if err != nil {
		return "", err
	}
	return resp.IdentityType, nil
}

func sendEmail(ctx context.Context, client *sesv2.Client, source, destination string) (string, error) {
	resp, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: []string{destination},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Stackyard SESv2 Basic")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String("hello from stackyard sesv2 basic example")},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.MessageId), nil
}

func listEmailIdentities(ctx context.Context, client *sesv2.Client) (int, error) {
	resp, err := client.ListEmailIdentities(ctx, &sesv2.ListEmailIdentitiesInput{PageSize: aws.Int32(100)})
	if err != nil {
		return 0, err
	}
	return len(resp.EmailIdentities), nil
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
