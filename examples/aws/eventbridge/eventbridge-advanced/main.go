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
	busName := getenv("STACKYARD_EVENT_BUS", "demo-eventbridge-advanced")
	ruleName := getenv("STACKYARD_RULE", "demo-rule-advanced")
	targetArn := getenv("STACKYARD_TARGET_ARN", "arn:aws:lambda:us-east-1:123456789012:function:demo")

	ctx := context.Background()
	client := newEventBridgeClient(ctx, endpoint)

	fmt.Printf("Stackyard EventBridge advanced client using %s\n", endpoint)

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

	if err := listRules(ctx, client, busName); err != nil {
		exitf("list rules: %v", err)
	}

	if err := toggleRule(ctx, client, busName, ruleName); err != nil {
		exitf("toggle rule: %v", err)
	}

	if err := putTargets(ctx, client, busName, ruleName, targetArn); err != nil {
		exitf("put targets: %v", err)
	}

	if err := listTargets(ctx, client, busName, ruleName); err != nil {
		exitf("list targets: %v", err)
	}

	if err := listRuleNamesByTarget(ctx, client, busName, targetArn); err != nil {
		exitf("list rule names by target: %v", err)
	}

	if err := putEvent(ctx, client, busName); err != nil {
		exitf("put events: %v", err)
	}

	if err := tagResource(ctx, client, ruleArn); err != nil {
		exitf("tag resource: %v", err)
	}

	if err := testEventPattern(ctx, client); err != nil {
		exitf("test event pattern: %v", err)
	}

	archiveArn, err := createArchive(ctx, client, busArn)
	if err != nil {
		exitf("create archive: %v", err)
	}
	logf("archive arn: %s", archiveArn)

	connArn, err := createConnection(ctx, client)
	if err != nil {
		exitf("create connection: %v", err)
	}
	logf("connection arn: %s", connArn)

	apiArn, err := createApiDestination(ctx, client, connArn)
	if err != nil {
		exitf("create api destination: %v", err)
	}
	logf("api destination arn: %s", apiArn)

	endpointArn, err := createEndpoint(ctx, client, busArn)
	if err != nil {
		exitf("create endpoint: %v", err)
	}
	logf("endpoint arn: %s", endpointArn)

	if err := cleanupAdvanced(ctx, client, busName, ruleName, targetArn); err != nil {
		exitf("cleanup: %v", err)
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
		EventPattern: aws.String(`{"detail-type":["build"],"source":["stackyard.advanced"]}`),
		State:        types.RuleStateEnabled,
		Description:  aws.String("advanced rule"),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.RuleArn), nil
}

func listRules(ctx context.Context, client *eventbridge.Client, busName string) error {
	resp, err := client.ListRules(ctx, &eventbridge.ListRulesInput{
		EventBusName: aws.String(busName),
	})
	if err != nil {
		return err
	}
	logf("rules: %d", len(resp.Rules))
	return nil
}

func toggleRule(ctx context.Context, client *eventbridge.Client, busName, ruleName string) error {
	_, err := client.DisableRule(ctx, &eventbridge.DisableRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
	})
	if err != nil {
		return err
	}
	_, err = client.EnableRule(ctx, &eventbridge.EnableRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
	})
	return err
}

func putTargets(ctx context.Context, client *eventbridge.Client, busName, ruleName, targetArn string) error {
	_, err := client.PutTargets(ctx, &eventbridge.PutTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Targets: []types.Target{
			{
				Id:    aws.String("target-1"),
				Arn:   aws.String(targetArn),
				Input: aws.String(`{"action":"deploy"}`),
			},
		},
	})
	return err
}

func listTargets(ctx context.Context, client *eventbridge.Client, busName, ruleName string) error {
	resp, err := client.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
	})
	if err != nil {
		return err
	}
	logf("targets: %d", len(resp.Targets))
	return nil
}

func listRuleNamesByTarget(ctx context.Context, client *eventbridge.Client, busName, targetArn string) error {
	resp, err := client.ListRuleNamesByTarget(ctx, &eventbridge.ListRuleNamesByTargetInput{
		EventBusName: aws.String(busName),
		TargetArn:    aws.String(targetArn),
	})
	if err != nil {
		return err
	}
	logf("rule names by target: %d", len(resp.RuleNames))
	return nil
}

func putEvent(ctx context.Context, client *eventbridge.Client, busName string) error {
	_, err := client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(busName),
				Source:       aws.String("stackyard.advanced"),
				DetailType:   aws.String("build"),
				Detail:       aws.String(`{"status":"ok","duration":42}`),
				Time:         aws.Time(time.Now().UTC()),
			},
		},
	})
	return err
}

func tagResource(ctx context.Context, client *eventbridge.Client, arn string) error {
	_, err := client.TagResource(ctx, &eventbridge.TagResourceInput{
		ResourceARN: aws.String(arn),
		Tags: []types.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
			{Key: aws.String("team"), Value: aws.String("platform")},
		},
	})
	return err
}

func testEventPattern(ctx context.Context, client *eventbridge.Client) error {
	resp, err := client.TestEventPattern(ctx, &eventbridge.TestEventPatternInput{
		EventPattern: aws.String(`{"source":["stackyard.advanced"]}`),
		Event:        aws.String(`{"source":"stackyard.advanced","detail-type":"build","detail":{"status":"ok"}}`),
	})
	if err != nil {
		return err
	}
	logf("pattern matches: %t", resp.Result)
	return nil
}

func createArchive(ctx context.Context, client *eventbridge.Client, busArn string) (string, error) {
	resp, err := client.CreateArchive(ctx, &eventbridge.CreateArchiveInput{
		ArchiveName:    aws.String("demo-archive"),
		EventSourceArn: aws.String(busArn),
		EventPattern:   aws.String(`{"source":["stackyard.advanced"]}`),
		RetentionDays:  aws.Int32(7),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.ArchiveArn), nil
}

func createConnection(ctx context.Context, client *eventbridge.Client) (string, error) {
	resp, err := client.CreateConnection(ctx, &eventbridge.CreateConnectionInput{
		Name:              aws.String("demo-connection"),
		AuthorizationType: types.ConnectionAuthorizationTypeApiKey,
		Description:       aws.String("demo connection"),
		AuthParameters: &types.CreateConnectionAuthRequestParameters{
			ApiKeyAuthParameters: &types.CreateConnectionApiKeyAuthRequestParameters{
				ApiKeyName:  aws.String("x-api-key"),
				ApiKeyValue: aws.String("stackyard-demo"),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.ConnectionArn), nil
}

func createApiDestination(ctx context.Context, client *eventbridge.Client, connectionArn string) (string, error) {
	resp, err := client.CreateApiDestination(ctx, &eventbridge.CreateApiDestinationInput{
		Name:                         aws.String("demo-api-destination"),
		ConnectionArn:                aws.String(connectionArn),
		InvocationEndpoint:           aws.String("https://example.com/webhook"),
		HttpMethod:                   types.ApiDestinationHttpMethodPost,
		InvocationRateLimitPerSecond: aws.Int32(10),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.ApiDestinationArn), nil
}

func createEndpoint(ctx context.Context, client *eventbridge.Client, busArn string) (string, error) {
	_, err := client.CreateEndpoint(ctx, &eventbridge.CreateEndpointInput{
		Name:        aws.String("demo-endpoint"),
		Description: aws.String("demo endpoint"),
		EventBuses: []types.EndpointEventBus{
			{EventBusArn: aws.String(busArn)},
			{EventBusArn: aws.String(busArn)},
		},
		ReplicationConfig: &types.ReplicationConfig{
			State: types.ReplicationStateDisabled,
		},
		RoutingConfig: &types.RoutingConfig{
			FailoverConfig: &types.FailoverConfig{
				Primary: &types.Primary{
					HealthCheck: aws.String("arn:aws:route53:::healthcheck/stackyard-demo"),
				},
				Secondary: &types.Secondary{
					Route: aws.String("us-west-2"),
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return "demo-endpoint", nil
}

func cleanupAdvanced(ctx context.Context, client *eventbridge.Client, busName, ruleName, targetArn string) error {
	_, _ = client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Ids:          []string{"target-1"},
	})
	_, _ = client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
		Force:        true,
	})
	_, _ = client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
		Name: aws.String(busName),
	})
	_, _ = client.DeleteArchive(ctx, &eventbridge.DeleteArchiveInput{
		ArchiveName: aws.String("demo-archive"),
	})
	_, _ = client.DeleteApiDestination(ctx, &eventbridge.DeleteApiDestinationInput{
		Name: aws.String("demo-api-destination"),
	})
	_, _ = client.DeleteConnection(ctx, &eventbridge.DeleteConnectionInput{
		Name: aws.String("demo-connection"),
	})
	_, _ = client.DeleteEndpoint(ctx, &eventbridge.DeleteEndpointInput{
		Name: aws.String("demo-endpoint"),
	})
	_, _ = client.UntagResource(ctx, &eventbridge.UntagResourceInput{
		ResourceARN: aws.String(targetArn),
		TagKeys:     []string{"env", "team"},
	})
	return nil
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
