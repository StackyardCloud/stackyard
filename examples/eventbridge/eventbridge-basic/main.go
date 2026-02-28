package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	busName := getenv("STACKYARD_EVENT_BUS", "demo-eventbridge-basic")
	ruleName := getenv("STACKYARD_RULE", "demo-rule-basic")

	ctx := context.Background()
	client := newEventBridgeClient(ctx, endpoint)

	fmt.Printf("Stackyard EventBridge basic client using %s\n", endpoint)

	busArn, err := ensureEventBus(ctx, client, busName)
	if err != nil {
		exitf("create event bus: %v", err)
	}
	logf("event bus arn: %s", busArn)

	ruleArn, err := ensureRule(ctx, client, busName, ruleName)
	if err != nil {
		exitf("create rule: %v", err)
	}
	logf("rule arn: %s", ruleArn)

	if err := putTargets(ctx, client, busName, ruleName); err != nil {
		exitf("put targets: %v", err)
	}
	logf("targets attached")

	if err := putEvent(ctx, client, busName); err != nil {
		exitf("put events: %v", err)
	}
	logf("event sent")

	if err := cleanupRule(ctx, client, busName, ruleName); err != nil {
		exitf("cleanup rule: %v", err)
	}
	if err := cleanupBus(ctx, client, busName); err != nil {
		exitf("cleanup bus: %v", err)
	}

	fmt.Println("Done.")
}

func newEventBridgeClient(ctx context.Context, endpoint string) *eventbridge.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == eventbridge.ServiceID {
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
	return eventbridge.NewFromConfig(cfg)
}

func ensureEventBus(ctx context.Context, client *eventbridge.Client, name string) (string, error) {
	resp, err := client.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
		Name: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.EventBusArn), nil
}

func ensureRule(ctx context.Context, client *eventbridge.Client, busName, ruleName string) (string, error) {
	resp, err := client.PutRule(ctx, &eventbridge.PutRuleInput{
		Name:         aws.String(ruleName),
		EventBusName: aws.String(busName),
		EventPattern: aws.String(`{"source":["stackyard.basic"]}`),
		State:        types.RuleStateEnabled,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.RuleArn), nil
}

func putTargets(ctx context.Context, client *eventbridge.Client, busName, ruleName string) error {
	_, err := client.PutTargets(ctx, &eventbridge.PutTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Targets: []types.Target{
			{
				Id:  aws.String("target-1"),
				Arn: aws.String("arn:aws:lambda:us-east-1:123456789012:function:demo"),
			},
		},
	})
	return err
}

func putEvent(ctx context.Context, client *eventbridge.Client, busName string) error {
	_, err := client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(busName),
				Source:       aws.String("stackyard.basic"),
				DetailType:   aws.String("build"),
				Detail:       aws.String(`{"status":"ok"}`),
				Time:         aws.Time(time.Now().UTC()),
			},
		},
	})
	return err
}

func cleanupRule(ctx context.Context, client *eventbridge.Client, busName, ruleName string) error {
	_, _ = client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Ids:          []string{"target-1"},
	})
	_, err := client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
		Force:        true,
	})
	return err
}

func cleanupBus(ctx context.Context, client *eventbridge.Client, busName string) error {
	_, err := client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
		Name: aws.String(busName),
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
