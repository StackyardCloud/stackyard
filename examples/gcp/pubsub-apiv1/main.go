package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	topicID := getenv("STACKYARD_GCP_PUBSUB_TOPIC_ID", "orders")
	subscriptionID := getenv("STACKYARD_GCP_PUBSUB_SUBSCRIPTION_ID", "orders-sub")
	snapshotID := getenv("STACKYARD_GCP_PUBSUB_SNAPSHOT_ID", "orders-snap")
	schemaID := getenv("STACKYARD_GCP_PUBSUB_SCHEMA_ID", "orders-schema")

	projectName := fmt.Sprintf("projects/%s", projectID)
	topicName := fmt.Sprintf("%s/topics/%s", projectName, topicID)
	subscriptionName := fmt.Sprintf("%s/subscriptions/%s", projectName, subscriptionID)
	snapshotName := fmt.Sprintf("%s/snapshots/%s", projectName, snapshotID)
	schemaName := fmt.Sprintf("%s/schemas/%s", projectName, schemaID)

	fmt.Printf("Stackyard GCP Cloud Pub/Sub apiv1 clients using %s\n", apiEndpoint)

	publisherClient, err := pubsub.NewPublisherRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create pubsub publisher client: %v", err)
	}
	defer closeClient("pubsub publisher", publisherClient.Close)

	subscriberClient, err := pubsub.NewSubscriberRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create pubsub subscriber client: %v", err)
	}
	defer closeClient("pubsub subscriber", subscriberClient.Close)

	schemaClient, err := pubsub.NewSchemaRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create pubsub schema client: %v", err)
	}
	defer closeClient("pubsub schema", schemaClient.Close)

	calls := []callSpec{
		{
			name: "Publisher.ListTopics",
			call: func(ctx context.Context) error {
				it := publisherClient.ListTopics(ctx, &pubsubpb.ListTopicsRequest{
					Project:  projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Publisher.CreateTopic",
			call: func(ctx context.Context) error {
				_, err := publisherClient.CreateTopic(ctx, &pubsubpb.Topic{
					Name: topicName,
				})
				return err
			},
		},
		{
			name: "Publisher.GetTopic",
			call: func(ctx context.Context) error {
				_, err := publisherClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: topicName})
				return err
			},
		},
		{
			name: "Publisher.Publish",
			call: func(ctx context.Context) error {
				_, err := publisherClient.Publish(ctx, &pubsubpb.PublishRequest{
					Topic: topicName,
					Messages: []*pubsubpb.PubsubMessage{
						{Data: []byte(`{"orderId":"o-1"}`)},
					},
				})
				return err
			},
		},
		{
			name: "Publisher.ListTopicSubscriptions",
			call: func(ctx context.Context) error {
				it := publisherClient.ListTopicSubscriptions(ctx, &pubsubpb.ListTopicSubscriptionsRequest{
					Topic:    topicName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Publisher.ListTopicSnapshots",
			call: func(ctx context.Context) error {
				it := publisherClient.ListTopicSnapshots(ctx, &pubsubpb.ListTopicSnapshotsRequest{
					Topic:    topicName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Publisher.DeleteTopic",
			call: func(ctx context.Context) error {
				return publisherClient.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{Topic: topicName})
			},
		},
		{
			name: "Subscriber.ListSubscriptions",
			call: func(ctx context.Context) error {
				it := subscriberClient.ListSubscriptions(ctx, &pubsubpb.ListSubscriptionsRequest{
					Project:  projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Subscriber.CreateSubscription",
			call: func(ctx context.Context) error {
				_, err := subscriberClient.CreateSubscription(ctx, &pubsubpb.Subscription{
					Name:  subscriptionName,
					Topic: topicName,
				})
				return err
			},
		},
		{
			name: "Subscriber.GetSubscription",
			call: func(ctx context.Context) error {
				_, err := subscriberClient.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: subscriptionName})
				return err
			},
		},
		{
			name: "Subscriber.Pull",
			call: func(ctx context.Context) error {
				_, err := subscriberClient.Pull(ctx, &pubsubpb.PullRequest{
					Subscription: subscriptionName,
					MaxMessages:  1,
				})
				return err
			},
		},
		{
			name: "Subscriber.Acknowledge",
			call: func(ctx context.Context) error {
				return subscriberClient.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{
					Subscription: subscriptionName,
					AckIds:       []string{"ack-1"},
				})
			},
		},
		{
			name: "Subscriber.ModifyAckDeadline",
			call: func(ctx context.Context) error {
				return subscriberClient.ModifyAckDeadline(ctx, &pubsubpb.ModifyAckDeadlineRequest{
					Subscription:       subscriptionName,
					AckIds:             []string{"ack-1"},
					AckDeadlineSeconds: 30,
				})
			},
		},
		{
			name: "Subscriber.Seek",
			call: func(ctx context.Context) error {
				_, err := subscriberClient.Seek(ctx, &pubsubpb.SeekRequest{
					Subscription: subscriptionName,
					Target: &pubsubpb.SeekRequest_Snapshot{
						Snapshot: snapshotName,
					},
				})
				return err
			},
		},
		{
			name: "Subscriber.ListSnapshots",
			call: func(ctx context.Context) error {
				it := subscriberClient.ListSnapshots(ctx, &pubsubpb.ListSnapshotsRequest{
					Project:  projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Subscriber.DeleteSubscription",
			call: func(ctx context.Context) error {
				return subscriberClient.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{Subscription: subscriptionName})
			},
		},
		{
			name: "Schema.ListSchemas",
			call: func(ctx context.Context) error {
				it := schemaClient.ListSchemas(ctx, &pubsubpb.ListSchemasRequest{
					Parent:   projectName,
					PageSize: 1,
					View:     pubsubpb.SchemaView_BASIC,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Schema.CreateSchema",
			call: func(ctx context.Context) error {
				_, err := schemaClient.CreateSchema(ctx, &pubsubpb.CreateSchemaRequest{
					Parent:   projectName,
					SchemaId: schemaID,
					Schema: &pubsubpb.Schema{
						Type:       pubsubpb.Schema_PROTOCOL_BUFFER,
						Definition: `syntax = "proto3"; message Order { string id = 1; }`,
					},
				})
				return err
			},
		},
		{
			name: "Schema.GetSchema",
			call: func(ctx context.Context) error {
				_, err := schemaClient.GetSchema(ctx, &pubsubpb.GetSchemaRequest{
					Name: schemaName,
					View: pubsubpb.SchemaView_FULL,
				})
				return err
			},
		},
		{
			name: "Schema.ValidateSchema",
			call: func(ctx context.Context) error {
				_, err := schemaClient.ValidateSchema(ctx, &pubsubpb.ValidateSchemaRequest{
					Parent: projectName,
					Schema: &pubsubpb.Schema{
						Type:       pubsubpb.Schema_PROTOCOL_BUFFER,
						Definition: `syntax = "proto3"; message Order { string id = 1; }`,
					},
				})
				return err
			},
		},
		{
			name: "Schema.ValidateMessage",
			call: func(ctx context.Context) error {
				_, err := schemaClient.ValidateMessage(ctx, &pubsubpb.ValidateMessageRequest{
					Parent: projectName,
					SchemaSpec: &pubsubpb.ValidateMessageRequest_Name{
						Name: schemaName,
					},
					Encoding: pubsubpb.Encoding_JSON,
					Message:  []byte(`{"id":"o-1"}`),
				})
				return err
			},
		},
		{
			name: "Schema.DeleteSchema",
			call: func(ctx context.Context) error {
				return schemaClient.DeleteSchema(ctx, &pubsubpb.DeleteSchemaRequest{Name: schemaName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
