package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	subscriptions "cloud.google.com/go/apps/events/subscriptions/apiv1"
	"cloud.google.com/go/apps/events/subscriptions/apiv1/subscriptionspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *subscriptions.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	subscriptionName := getenv("STACKYARD_GCP_SUBSCRIPTION", "subscriptions/team-subscription")
	eventType := getenv("STACKYARD_GCP_EVENT_TYPE", "google.workspace.chat.message.v1.created")
	targetResource := getenv("STACKYARD_GCP_TARGET_RESOURCE", "//chat.googleapis.com/spaces/AAAA123")
	pubsubTopic := getenv("STACKYARD_GCP_PUBSUB_TOPIC", "projects/stackyard/topics/workspace-events")
	filter := getenv("STACKYARD_GCP_SUBSCRIPTION_FILTER", `event_types:"google.workspace.chat.message.v1.created"`)

	fmt.Printf("Stackyard GCP Workspace Events Subscriptions apiv1 client using %s\n", apiEndpoint)

	client, err := subscriptions.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create subscriptions client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListSubscriptions",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				it := c.ListSubscriptions(ctx, &subscriptionspb.ListSubscriptionsRequest{
					Filter:   filter,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetSubscription",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				_, err := c.GetSubscription(ctx, &subscriptionspb.GetSubscriptionRequest{
					Name: subscriptionName,
				})
				return err
			},
		},
		{
			name: "CreateSubscription",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				_, err := c.CreateSubscription(ctx, &subscriptionspb.CreateSubscriptionRequest{
					Subscription: &subscriptionspb.Subscription{
						TargetResource: targetResource,
						EventTypes:     []string{eventType},
						NotificationEndpoint: &subscriptionspb.NotificationEndpoint{
							Endpoint: &subscriptionspb.NotificationEndpoint_PubsubTopic{
								PubsubTopic: pubsubTopic,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateSubscription",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				_, err := c.UpdateSubscription(ctx, &subscriptionspb.UpdateSubscriptionRequest{
					Subscription: &subscriptionspb.Subscription{
						Name: subscriptionName,
						Expiration: &subscriptionspb.Subscription_Ttl{
							Ttl: durationpb.New(2 * time.Hour),
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"ttl"}},
				})
				return err
			},
		},
		{
			name: "ReactivateSubscription",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				_, err := c.ReactivateSubscription(ctx, &subscriptionspb.ReactivateSubscriptionRequest{
					Name: subscriptionName,
				})
				return err
			},
		},
		{
			name: "DeleteSubscription",
			call: func(ctx context.Context, c *subscriptions.Client) error {
				_, err := c.DeleteSubscription(ctx, &subscriptionspb.DeleteSubscriptionRequest{
					Name: subscriptionName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
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

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close subscriptions client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
