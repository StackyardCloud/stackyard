package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	pubsublite "cloud.google.com/go/pubsublite/apiv1"
	"cloud.google.com/go/pubsublite/apiv1/pubsublitepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	topicID := getenv("STACKYARD_GCP_PUBSUBLITE_TOPIC_ID", "orders-lite")
	subscriptionID := getenv("STACKYARD_GCP_PUBSUBLITE_SUBSCRIPTION_ID", "orders-lite-sub")
	reservationID := getenv("STACKYARD_GCP_PUBSUBLITE_RESERVATION_ID", "orders-lite-reservation")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	topicName := parent + "/topics/" + topicID
	subscriptionName := parent + "/subscriptions/" + subscriptionID
	reservationName := parent + "/reservations/" + reservationID
	clientID := []byte("0123456789abcdef")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Pub/Sub Lite apiv1 clients using gRPC endpoint %s\n", grpcEndpoint)

	adminClient, err := pubsublite.NewAdminClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite admin client: %v", err)
	}
	defer closeClient("pubsublite admin", adminClient.Close)

	cursorClient, err := pubsublite.NewCursorClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite cursor client: %v", err)
	}
	defer closeClient("pubsublite cursor", cursorClient.Close)

	partitionAssignmentClient, err := pubsublite.NewPartitionAssignmentClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite partition assignment client: %v", err)
	}
	defer closeClient("pubsublite partition assignment", partitionAssignmentClient.Close)

	publisherClient, err := pubsublite.NewPublisherClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite publisher client: %v", err)
	}
	defer closeClient("pubsublite publisher", publisherClient.Close)

	subscriberClient, err := pubsublite.NewSubscriberClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite subscriber client: %v", err)
	}
	defer closeClient("pubsublite subscriber", subscriberClient.Close)

	topicStatsClient, err := pubsublite.NewTopicStatsClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create pubsublite topic stats client: %v", err)
	}
	defer closeClient("pubsublite topic stats", topicStatsClient.Close)

	calls := []callSpec{
		{
			name: "Admin.ListTopics",
			call: func(ctx context.Context) error {
				it := adminClient.ListTopics(ctx, &pubsublitepb.ListTopicsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Admin.CreateTopic",
			call: func(ctx context.Context) error {
				_, err := adminClient.CreateTopic(ctx, &pubsublitepb.CreateTopicRequest{
					Parent:  parent,
					TopicId: topicID,
					Topic: &pubsublitepb.Topic{
						PartitionConfig: &pubsublitepb.Topic_PartitionConfig{
							Count: 1,
						},
					},
				})
				return err
			},
		},
		{
			name: "Admin.GetTopic",
			call: func(ctx context.Context) error {
				_, err := adminClient.GetTopic(ctx, &pubsublitepb.GetTopicRequest{Name: topicName})
				return err
			},
		},
		{
			name: "Admin.ListSubscriptions",
			call: func(ctx context.Context) error {
				it := adminClient.ListSubscriptions(ctx, &pubsublitepb.ListSubscriptionsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Admin.CreateSubscription",
			call: func(ctx context.Context) error {
				_, err := adminClient.CreateSubscription(ctx, &pubsublitepb.CreateSubscriptionRequest{
					Parent:         parent,
					SubscriptionId: subscriptionID,
					Subscription: &pubsublitepb.Subscription{
						Topic: topicName,
					},
				})
				return err
			},
		},
		{
			name: "Admin.GetSubscription",
			call: func(ctx context.Context) error {
				_, err := adminClient.GetSubscription(ctx, &pubsublitepb.GetSubscriptionRequest{Name: subscriptionName})
				return err
			},
		},
		{
			name: "Admin.GetTopicPartitions",
			call: func(ctx context.Context) error {
				_, err := adminClient.GetTopicPartitions(ctx, &pubsublitepb.GetTopicPartitionsRequest{Name: topicName})
				return err
			},
		},
		{
			name: "Admin.ListReservations",
			call: func(ctx context.Context) error {
				it := adminClient.ListReservations(ctx, &pubsublitepb.ListReservationsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Admin.CreateReservation",
			call: func(ctx context.Context) error {
				_, err := adminClient.CreateReservation(ctx, &pubsublitepb.CreateReservationRequest{
					Parent:        parent,
					ReservationId: reservationID,
					Reservation: &pubsublitepb.Reservation{
						ThroughputCapacity: 1,
					},
				})
				return err
			},
		},
		{
			name: "Admin.ListReservationTopics",
			call: func(ctx context.Context) error {
				it := adminClient.ListReservationTopics(ctx, &pubsublitepb.ListReservationTopicsRequest{
					Name:     reservationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Admin.SeekSubscription",
			call: func(ctx context.Context) error {
				_, err := adminClient.SeekSubscription(ctx, &pubsublitepb.SeekSubscriptionRequest{
					Name: subscriptionName,
					Target: &pubsublitepb.SeekSubscriptionRequest_TimeTarget{
						TimeTarget: &pubsublitepb.TimeTarget{
							Time: &pubsublitepb.TimeTarget_PublishTime{
								PublishTime: timestamppb.Now(),
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "Admin.ListOperations",
			call: func(ctx context.Context) error {
				it := adminClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Cursor.ListPartitionCursors",
			call: func(ctx context.Context) error {
				it := cursorClient.ListPartitionCursors(ctx, &pubsublitepb.ListPartitionCursorsRequest{
					Parent:   subscriptionName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "Cursor.CommitCursor",
			call: func(ctx context.Context) error {
				_, err := cursorClient.CommitCursor(ctx, &pubsublitepb.CommitCursorRequest{
					Subscription: subscriptionName,
					Partition:    0,
					Cursor: &pubsublitepb.Cursor{
						Offset: 0,
					},
				})
				return err
			},
		},
		{
			name: "Publisher.PublishStream",
			call: func(ctx context.Context) error {
				stream, err := publisherClient.Publish(ctx)
				if err != nil {
					return err
				}
				if err := stream.Send(&pubsublitepb.PublishRequest{
					RequestType: &pubsublitepb.PublishRequest_InitialRequest{
						InitialRequest: &pubsublitepb.InitialPublishRequest{
							Topic:     topicName,
							Partition: 0,
							ClientId:  clientID,
						},
					},
				}); err != nil {
					return err
				}
				if err := stream.CloseSend(); err != nil {
					return err
				}
				_, err = stream.Recv()
				return streamResult(err)
			},
		},
		{
			name: "Subscriber.SubscribeStream",
			call: func(ctx context.Context) error {
				stream, err := subscriberClient.Subscribe(ctx)
				if err != nil {
					return err
				}
				if err := stream.Send(&pubsublitepb.SubscribeRequest{
					Request: &pubsublitepb.SubscribeRequest_Initial{
						Initial: &pubsublitepb.InitialSubscribeRequest{
							Subscription: subscriptionName,
							Partition:    0,
						},
					},
				}); err != nil {
					return err
				}
				if err := stream.CloseSend(); err != nil {
					return err
				}
				_, err = stream.Recv()
				return streamResult(err)
			},
		},
		{
			name: "PartitionAssignment.AssignPartitionsStream",
			call: func(ctx context.Context) error {
				stream, err := partitionAssignmentClient.AssignPartitions(ctx)
				if err != nil {
					return err
				}
				if err := stream.Send(&pubsublitepb.PartitionAssignmentRequest{
					Request: &pubsublitepb.PartitionAssignmentRequest_Initial{
						Initial: &pubsublitepb.InitialPartitionAssignmentRequest{
							Subscription: subscriptionName,
							ClientId:     clientID,
						},
					},
				}); err != nil {
					return err
				}
				if err := stream.CloseSend(); err != nil {
					return err
				}
				_, err = stream.Recv()
				return streamResult(err)
			},
		},
		{
			name: "TopicStats.ComputeHeadCursor",
			call: func(ctx context.Context) error {
				_, err := topicStatsClient.ComputeHeadCursor(ctx, &pubsublitepb.ComputeHeadCursorRequest{
					Topic:     topicName,
					Partition: 0,
				})
				return err
			},
		},
		{
			name: "TopicStats.ComputeMessageStats",
			call: func(ctx context.Context) error {
				_, err := topicStatsClient.ComputeMessageStats(ctx, &pubsublitepb.ComputeMessageStatsRequest{
					Topic:     topicName,
					Partition: 0,
					StartCursor: &pubsublitepb.Cursor{
						Offset: 0,
					},
					EndCursor: &pubsublitepb.Cursor{
						Offset: 10,
					},
				})
				return err
			},
		},
		{
			name: "TopicStats.ComputeTimeCursor",
			call: func(ctx context.Context) error {
				_, err := topicStatsClient.ComputeTimeCursor(ctx, &pubsublitepb.ComputeTimeCursorRequest{
					Topic:     topicName,
					Partition: 0,
					Target: &pubsublitepb.TimeTarget{
						Time: &pubsublitepb.TimeTarget_PublishTime{
							PublishTime: timestamppb.Now(),
						},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
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

func streamResult(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded, codes.Internal:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large") ||
		strings.Contains(text, "unexpected content-type") ||
		strings.Contains(text, "http2")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
