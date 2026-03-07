package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	publishing "cloud.google.com/go/eventarc/publishing/apiv1"
	"cloud.google.com/go/eventarc/publishing/apiv1/publishingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *publishing.PublisherClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	channelID := getenv("STACKYARD_GCP_EVENTARC_CHANNEL_ID", "team-channel")
	channelConnectionProjectID := getenv("STACKYARD_GCP_EVENTARC_CONNECTION_PROJECT_ID", "partner")
	channelConnectionID := getenv("STACKYARD_GCP_EVENTARC_CONNECTION_ID", "conn-1")
	messageBusID := getenv("STACKYARD_GCP_EVENTARC_MESSAGE_BUS_ID", "team-bus")

	channelName := fmt.Sprintf("projects/%s/locations/%s/channels/%s", projectID, locationID, channelID)
	channelConnectionName := fmt.Sprintf("projects/%s/locations/%s/channelConnections/%s", channelConnectionProjectID, locationID, channelConnectionID)
	messageBusName := fmt.Sprintf("projects/%s/locations/%s/messageBuses/%s", projectID, locationID, messageBusID)

	jsonCloudEvent := `{"specversion":"1.0","id":"evt-1","source":"https://stackyard.dev/examples","type":"com.stackyard.event.created","datacontenttype":"application/json","data":{"hello":"world"}}`

	fmt.Printf("Stackyard GCP Eventarc Publishing apiv1 client using %s\n", apiEndpoint)

	client, err := publishing.NewPublisherRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create eventarc publishing client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "PublishChannelConnectionEvents",
			call: func(ctx context.Context, c *publishing.PublisherClient) error {
				_, err := c.PublishChannelConnectionEvents(ctx, &publishingpb.PublishChannelConnectionEventsRequest{
					ChannelConnection: channelConnectionName,
					TextEvents:        []string{jsonCloudEvent},
				})
				return err
			},
		},
		{
			name: "PublishEvents",
			call: func(ctx context.Context, c *publishing.PublisherClient) error {
				_, err := c.PublishEvents(ctx, &publishingpb.PublishEventsRequest{
					Channel:    channelName,
					TextEvents: []string{jsonCloudEvent},
				})
				return err
			},
		},
		{
			name: "Publish",
			call: func(ctx context.Context, c *publishing.PublisherClient) error {
				_, err := c.Publish(ctx, &publishingpb.PublishRequest{
					MessageBus: messageBusName,
					Format: &publishingpb.PublishRequest_JsonMessage{
						JsonMessage: jsonCloudEvent,
					},
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
		fmt.Fprintf(os.Stderr, "warning: close eventarc publishing client: %v\n", err)
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
