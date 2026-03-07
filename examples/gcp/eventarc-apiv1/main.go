package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	eventarc "cloud.google.com/go/eventarc/apiv1"
	"cloud.google.com/go/eventarc/apiv1/eventarcpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
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
	call func(context.Context, *eventarc.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	triggerID := getenv("STACKYARD_GCP_EVENTARC_TRIGGER_ID", "team-trigger")
	channelID := getenv("STACKYARD_GCP_EVENTARC_CHANNEL_ID", "team-channel")
	channelConnectionID := getenv("STACKYARD_GCP_EVENTARC_CONNECTION_ID", "team-connection")
	messageBusID := getenv("STACKYARD_GCP_EVENTARC_MESSAGE_BUS_ID", "team-bus")
	enrollmentID := getenv("STACKYARD_GCP_EVENTARC_ENROLLMENT_ID", "team-enrollment")
	pipelineID := getenv("STACKYARD_GCP_EVENTARC_PIPELINE_ID", "team-pipeline")
	googleAPISourceID := getenv("STACKYARD_GCP_EVENTARC_GOOGLE_API_SOURCE_ID", "team-source")
	operationID := getenv("STACKYARD_GCP_EVENTARC_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	triggerName := locationName + "/triggers/" + triggerID
	channelName := locationName + "/channels/" + channelID
	providerName := locationName + "/providers/google-cloudevents"
	channelConnectionName := locationName + "/channelConnections/" + channelConnectionID
	googleChannelConfigName := locationName + "/googleChannelConfig"
	messageBusName := locationName + "/messageBuses/" + messageBusID
	enrollmentName := locationName + "/enrollments/" + enrollmentID
	pipelineName := locationName + "/pipelines/" + pipelineID
	googleAPISourceName := locationName + "/googleApiSources/" + googleAPISourceID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Eventarc apiv1 client using %s\n", apiEndpoint)

	client, err := eventarc.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create eventarc client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListTriggers",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListTriggers(ctx, &eventarcpb.ListTriggersRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetTrigger",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetTrigger(ctx, &eventarcpb.GetTriggerRequest{Name: triggerName})
				return err
			},
		},
		{
			name: "CreateTrigger",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateTrigger(ctx, &eventarcpb.CreateTriggerRequest{
					Parent:    locationName,
					TriggerId: triggerID,
					Trigger:   &eventarcpb.Trigger{Name: triggerName},
				})
				return err
			},
		},
		{
			name: "UpdateTrigger",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateTrigger(ctx, &eventarcpb.UpdateTriggerRequest{
					Trigger:    &eventarcpb.Trigger{Name: triggerName},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteTrigger",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteTrigger(ctx, &eventarcpb.DeleteTriggerRequest{Name: triggerName})
				return err
			},
		},
		{
			name: "ListChannels",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListChannels(ctx, &eventarcpb.ListChannelsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetChannel",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetChannel(ctx, &eventarcpb.GetChannelRequest{Name: channelName})
				return err
			},
		},
		{
			name: "CreateChannel",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateChannel(ctx, &eventarcpb.CreateChannelRequest{
					Parent:    locationName,
					ChannelId: channelID,
					Channel:   &eventarcpb.Channel{Name: channelName},
				})
				return err
			},
		},
		{
			name: "UpdateChannel",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateChannel(ctx, &eventarcpb.UpdateChannelRequest{
					Channel:    &eventarcpb.Channel{Name: channelName},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteChannel",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteChannel(ctx, &eventarcpb.DeleteChannelRequest{Name: channelName})
				return err
			},
		},
		{
			name: "ListProviders",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListProviders(ctx, &eventarcpb.ListProvidersRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetProvider",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetProvider(ctx, &eventarcpb.GetProviderRequest{Name: providerName})
				return err
			},
		},
		{
			name: "ListChannelConnections",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListChannelConnections(ctx, &eventarcpb.ListChannelConnectionsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetChannelConnection",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetChannelConnection(ctx, &eventarcpb.GetChannelConnectionRequest{Name: channelConnectionName})
				return err
			},
		},
		{
			name: "CreateChannelConnection",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateChannelConnection(ctx, &eventarcpb.CreateChannelConnectionRequest{
					Parent:              locationName,
					ChannelConnectionId: channelConnectionID,
					ChannelConnection:   &eventarcpb.ChannelConnection{Name: channelConnectionName},
				})
				return err
			},
		},
		{
			name: "DeleteChannelConnection",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteChannelConnection(ctx, &eventarcpb.DeleteChannelConnectionRequest{Name: channelConnectionName})
				return err
			},
		},
		{
			name: "GetGoogleChannelConfig",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetGoogleChannelConfig(ctx, &eventarcpb.GetGoogleChannelConfigRequest{Name: googleChannelConfigName})
				return err
			},
		},
		{
			name: "UpdateGoogleChannelConfig",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateGoogleChannelConfig(ctx, &eventarcpb.UpdateGoogleChannelConfigRequest{
					GoogleChannelConfig: &eventarcpb.GoogleChannelConfig{Name: googleChannelConfigName},
					UpdateMask:          &fieldmaskpb.FieldMask{Paths: []string{"crypto_key_name"}},
				})
				return err
			},
		},
		{
			name: "ListMessageBuses",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListMessageBuses(ctx, &eventarcpb.ListMessageBusesRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetMessageBus",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetMessageBus(ctx, &eventarcpb.GetMessageBusRequest{Name: messageBusName})
				return err
			},
		},
		{
			name: "ListMessageBusEnrollments",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListMessageBusEnrollments(ctx, &eventarcpb.ListMessageBusEnrollmentsRequest{Parent: messageBusName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateMessageBus",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateMessageBus(ctx, &eventarcpb.CreateMessageBusRequest{
					Parent:       locationName,
					MessageBusId: messageBusID,
					MessageBus:   &eventarcpb.MessageBus{Name: messageBusName},
				})
				return err
			},
		},
		{
			name: "UpdateMessageBus",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateMessageBus(ctx, &eventarcpb.UpdateMessageBusRequest{
					MessageBus:   &eventarcpb.MessageBus{Name: messageBusName},
					UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					AllowMissing: true,
				})
				return err
			},
		},
		{
			name: "DeleteMessageBus",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteMessageBus(ctx, &eventarcpb.DeleteMessageBusRequest{Name: messageBusName, AllowMissing: true})
				return err
			},
		},
		{
			name: "ListEnrollments",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListEnrollments(ctx, &eventarcpb.ListEnrollmentsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetEnrollment",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetEnrollment(ctx, &eventarcpb.GetEnrollmentRequest{Name: enrollmentName})
				return err
			},
		},
		{
			name: "CreateEnrollment",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateEnrollment(ctx, &eventarcpb.CreateEnrollmentRequest{
					Parent:       locationName,
					EnrollmentId: enrollmentID,
					Enrollment:   &eventarcpb.Enrollment{Name: enrollmentName},
				})
				return err
			},
		},
		{
			name: "UpdateEnrollment",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateEnrollment(ctx, &eventarcpb.UpdateEnrollmentRequest{
					Enrollment:   &eventarcpb.Enrollment{Name: enrollmentName},
					UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					AllowMissing: true,
				})
				return err
			},
		},
		{
			name: "DeleteEnrollment",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteEnrollment(ctx, &eventarcpb.DeleteEnrollmentRequest{Name: enrollmentName, AllowMissing: true})
				return err
			},
		},
		{
			name: "ListPipelines",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListPipelines(ctx, &eventarcpb.ListPipelinesRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPipeline",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetPipeline(ctx, &eventarcpb.GetPipelineRequest{Name: pipelineName})
				return err
			},
		},
		{
			name: "CreatePipeline",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreatePipeline(ctx, &eventarcpb.CreatePipelineRequest{
					Parent:     locationName,
					PipelineId: pipelineID,
					Pipeline:   &eventarcpb.Pipeline{Name: pipelineName},
				})
				return err
			},
		},
		{
			name: "UpdatePipeline",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdatePipeline(ctx, &eventarcpb.UpdatePipelineRequest{
					Pipeline:     &eventarcpb.Pipeline{Name: pipelineName},
					UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					AllowMissing: true,
				})
				return err
			},
		},
		{
			name: "DeletePipeline",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeletePipeline(ctx, &eventarcpb.DeletePipelineRequest{Name: pipelineName, AllowMissing: true})
				return err
			},
		},
		{
			name: "ListGoogleApiSources",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListGoogleApiSources(ctx, &eventarcpb.ListGoogleApiSourcesRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetGoogleApiSource",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetGoogleApiSource(ctx, &eventarcpb.GetGoogleApiSourceRequest{Name: googleAPISourceName})
				return err
			},
		},
		{
			name: "CreateGoogleApiSource",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.CreateGoogleApiSource(ctx, &eventarcpb.CreateGoogleApiSourceRequest{
					Parent:            locationName,
					GoogleApiSourceId: googleAPISourceID,
					GoogleApiSource:   &eventarcpb.GoogleApiSource{Name: googleAPISourceName},
				})
				return err
			},
		},
		{
			name: "UpdateGoogleApiSource",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.UpdateGoogleApiSource(ctx, &eventarcpb.UpdateGoogleApiSourceRequest{
					GoogleApiSource: &eventarcpb.GoogleApiSource{Name: googleAPISourceName},
					UpdateMask:      &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteGoogleApiSource",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.DeleteGoogleApiSource(ctx, &eventarcpb.DeleteGoogleApiSourceRequest{Name: googleAPISourceName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectName})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: triggerName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: triggerName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    triggerName,
					Permissions: []string{"eventarc.triggers.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *eventarc.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *eventarc.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *eventarc.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *eventarc.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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
		fmt.Fprintf(os.Stderr, "warning: close eventarc client: %v\n", err)
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
