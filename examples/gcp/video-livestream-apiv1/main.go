package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	livestream "cloud.google.com/go/video/livestream/apiv1"
	livestreampb "cloud.google.com/go/video/livestream/apiv1/livestreampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	projectName := "projects/" + projectID
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	channelName := parent + "/channels/channel-1"
	inputName := parent + "/inputs/input-1"
	assetName := parent + "/assets/asset-1"

	fmt.Printf("Stackyard GCP Live Stream video/livestream/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "video-livestream",
		},
	}

	client, err := livestream.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create livestream client: %v", err)
	}
	defer closeClient("livestream", client.Close)

	operationName := parent + "/operations/createChannel.channel-1"

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				location, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListLocations returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(location.GetName()) == "" {
					return errors.New("ListLocations returned location without name")
				}
				return nil
			},
		},
		{
			name: "CreateChannel",
			call: func(ctx context.Context) error {
				op, err := client.CreateChannel(ctx, &livestreampb.CreateChannelRequest{
					Parent:    parent,
					ChannelId: "channel-1",
					Channel: &livestreampb.Channel{
						Name: channelName,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "ListChannels",
			call: func(ctx context.Context) error {
				it := client.ListChannels(ctx, &livestreampb.ListChannelsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				channel, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListChannels returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(channel.GetName()) == "" {
					return errors.New("ListChannels returned channel without name")
				}
				return nil
			},
		},
		{
			name: "GetChannel",
			call: func(ctx context.Context) error {
				channel, err := client.GetChannel(ctx, &livestreampb.GetChannelRequest{Name: channelName})
				if err != nil {
					return err
				}
				if channel.GetName() != channelName {
					return fmt.Errorf("GetChannel returned unexpected name: %q", channel.GetName())
				}
				return nil
			},
		},
		{
			name: "StartChannel",
			call: func(ctx context.Context) error {
				op, err := client.StartChannel(ctx, &livestreampb.StartChannelRequest{Name: channelName})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "StopChannel",
			call: func(ctx context.Context) error {
				op, err := client.StopChannel(ctx, &livestreampb.StopChannelRequest{Name: channelName})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "CreateInput",
			call: func(ctx context.Context) error {
				op, err := client.CreateInput(ctx, &livestreampb.CreateInputRequest{
					Parent:  parent,
					InputId: "input-1",
					Input: &livestreampb.Input{
						Name: inputName,
					},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "PreviewInput",
			call: func(ctx context.Context) error {
				resp, err := client.PreviewInput(ctx, &livestreampb.PreviewInputRequest{Name: inputName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(resp.GetUri()) == "" {
					return errors.New("PreviewInput returned empty uri")
				}
				return nil
			},
		},
		{
			name: "CreateEvent",
			call: func(ctx context.Context) error {
				event, err := client.CreateEvent(ctx, &livestreampb.CreateEventRequest{
					Parent:  channelName,
					EventId: "event-1",
					Event: &livestreampb.Event{
						Name: channelName + "/events/event-1",
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(event.GetName()) == "" {
					return errors.New("CreateEvent returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListEvents",
			call: func(ctx context.Context) error {
				it := client.ListEvents(ctx, &livestreampb.ListEventsRequest{
					Parent:   channelName,
					PageSize: 1,
				})
				event, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListEvents returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(event.GetName()) == "" {
					return errors.New("ListEvents returned event without name")
				}
				return nil
			},
		},
		{
			name: "CreateClip",
			call: func(ctx context.Context) error {
				op, err := client.CreateClip(ctx, &livestreampb.CreateClipRequest{
					Parent: channelName,
					ClipId: "clip-1",
					Clip: &livestreampb.Clip{
						Name: channelName + "/clips/clip-1",
					},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "ListClips",
			call: func(ctx context.Context) error {
				it := client.ListClips(ctx, &livestreampb.ListClipsRequest{
					Parent:   channelName,
					PageSize: 1,
				})
				clip, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListClips returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(clip.GetName()) == "" {
					return errors.New("ListClips returned clip without name")
				}
				return nil
			},
		},
		{
			name: "CreateAsset",
			call: func(ctx context.Context) error {
				op, err := client.CreateAsset(ctx, &livestreampb.CreateAssetRequest{
					Parent:  parent,
					AssetId: "asset-1",
					Asset: &livestreampb.Asset{
						Name: assetName,
					},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "GetAsset",
			call: func(ctx context.Context) error {
				asset, err := client.GetAsset(ctx, &livestreampb.GetAssetRequest{Name: assetName})
				if err != nil {
					return err
				}
				if asset.GetName() != assetName {
					return fmt.Errorf("GetAsset returned unexpected name: %q", asset.GetName())
				}
				return nil
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				operation, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(operation.GetName()) == "" {
					return errors.New("GetOperation returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				operation, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListOperations returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(operation.GetName()) == "" {
					return errors.New("ListOperations returned operation without name")
				}
				return nil
			},
		},
		{
			name: "NegativeValidation_CreateChannelMissingParent",
			call: func(ctx context.Context) error {
				_, err := client.CreateChannel(ctx, &livestreampb.CreateChannelRequest{
					ChannelId: "invalid-no-parent",
					Channel: &livestreampb.Channel{
						Name: parent + "/channels/invalid-no-parent",
					},
				})
				if isInvalidArgument(err) {
					return nil
				}
				if err == nil {
					return errors.New("expected invalid argument for create channel missing parent")
				}
				return fmt.Errorf("expected invalid argument, got: %w", err)
			},
		},
		{
			name: "UpdateInput",
			call: func(ctx context.Context) error {
				op, err := client.UpdateInput(ctx, &livestreampb.UpdateInputRequest{
					Input: &livestreampb.Input{
						Name: inputName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"tier"}},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func isInvalidArgument(err error) bool {
	if err == nil {
		return false
	}
	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.InvalidArgument {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "invalidargument")
}

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
