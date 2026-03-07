package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	datastream "cloud.google.com/go/datastream/apiv1"
	"cloud.google.com/go/datastream/apiv1/datastreampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *datastream.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	connectionProfileID := getenv("STACKYARD_GCP_DATASTREAM_CONNECTION_PROFILE_ID", "source-profile")
	streamID := getenv("STACKYARD_GCP_DATASTREAM_STREAM_ID", "orders-stream")
	streamObjectID := getenv("STACKYARD_GCP_DATASTREAM_OBJECT_ID", "orders")
	privateConnectionID := getenv("STACKYARD_GCP_DATASTREAM_PRIVATE_CONNECTION_ID", "private-link")
	routeID := getenv("STACKYARD_GCP_DATASTREAM_ROUTE_ID", "route-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	connectionProfileName := parent + "/connectionProfiles/" + connectionProfileID
	streamName := parent + "/streams/" + streamID
	streamObjectName := streamName + "/objects/" + streamObjectID
	privateConnectionName := parent + "/privateConnections/" + privateConnectionID
	routeName := privateConnectionName + "/routes/" + routeID

	fmt.Printf("Stackyard GCP Datastream apiv1 client using %s\n", apiEndpoint)

	client, err := datastream.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create datastream client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListConnectionProfiles",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.ListConnectionProfiles(ctx, &datastreampb.ListConnectionProfilesRequest{
					Parent:   parent,
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
			name: "GetConnectionProfile",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.GetConnectionProfile(ctx, &datastreampb.GetConnectionProfileRequest{Name: connectionProfileName})
				return err
			},
		},
		{
			name: "CreateConnectionProfile",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.CreateConnectionProfile(ctx, &datastreampb.CreateConnectionProfileRequest{
					Parent:              parent,
					ConnectionProfileId: connectionProfileID,
					ConnectionProfile: &datastreampb.ConnectionProfile{
						Name: connectionProfileName,
					},
				})
				return err
			},
		},
		{
			name: "DiscoverConnectionProfile",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.DiscoverConnectionProfile(ctx, &datastreampb.DiscoverConnectionProfileRequest{
					Parent: parent,
					Target: &datastreampb.DiscoverConnectionProfileRequest_ConnectionProfileName{
						ConnectionProfileName: connectionProfileName,
					},
				})
				return err
			},
		},
		{
			name: "ListStreams",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.ListStreams(ctx, &datastreampb.ListStreamsRequest{
					Parent:   parent,
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
			name: "GetStream",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.GetStream(ctx, &datastreampb.GetStreamRequest{Name: streamName})
				return err
			},
		},
		{
			name: "CreateStream",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.CreateStream(ctx, &datastreampb.CreateStreamRequest{
					Parent:   parent,
					StreamId: streamID,
					Stream: &datastreampb.Stream{
						Name: streamName,
					},
				})
				return err
			},
		},
		{
			name: "RunStream",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.RunStream(ctx, &datastreampb.RunStreamRequest{Name: streamName})
				return err
			},
		},
		{
			name: "ListStreamObjects",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.ListStreamObjects(ctx, &datastreampb.ListStreamObjectsRequest{
					Parent:   streamName,
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
			name: "GetStreamObject",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.GetStreamObject(ctx, &datastreampb.GetStreamObjectRequest{Name: streamObjectName})
				return err
			},
		},
		{
			name: "LookupStreamObject",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.LookupStreamObject(ctx, &datastreampb.LookupStreamObjectRequest{
					Parent:                 streamName,
					SourceObjectIdentifier: &datastreampb.SourceObjectIdentifier{},
				})
				return err
			},
		},
		{
			name: "StartBackfillJob",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.StartBackfillJob(ctx, &datastreampb.StartBackfillJobRequest{
					Object: streamObjectName,
				})
				return err
			},
		},
		{
			name: "StopBackfillJob",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.StopBackfillJob(ctx, &datastreampb.StopBackfillJobRequest{
					Object: streamObjectName,
				})
				return err
			},
		},
		{
			name: "FetchStaticIps",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.FetchStaticIps(ctx, &datastreampb.FetchStaticIpsRequest{
					Name:     parent,
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
			name: "ListPrivateConnections",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.ListPrivateConnections(ctx, &datastreampb.ListPrivateConnectionsRequest{
					Parent:   parent,
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
			name: "GetPrivateConnection",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.GetPrivateConnection(ctx, &datastreampb.GetPrivateConnectionRequest{Name: privateConnectionName})
				return err
			},
		},
		{
			name: "CreatePrivateConnection",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.CreatePrivateConnection(ctx, &datastreampb.CreatePrivateConnectionRequest{
					Parent:              parent,
					PrivateConnectionId: privateConnectionID,
					PrivateConnection: &datastreampb.PrivateConnection{
						Name: privateConnectionName,
					},
				})
				return err
			},
		},
		{
			name: "ListRoutes",
			call: func(ctx context.Context, c *datastream.Client) error {
				it := c.ListRoutes(ctx, &datastreampb.ListRoutesRequest{
					Parent:   privateConnectionName,
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
			name: "GetRoute",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.GetRoute(ctx, &datastreampb.GetRouteRequest{Name: routeName})
				return err
			},
		},
		{
			name: "CreateRoute",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.CreateRoute(ctx, &datastreampb.CreateRouteRequest{
					Parent:  privateConnectionName,
					RouteId: routeID,
					Route: &datastreampb.Route{
						Name: routeName,
					},
				})
				return err
			},
		},
		{
			name: "DeleteRoute",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.DeleteRoute(ctx, &datastreampb.DeleteRouteRequest{Name: routeName})
				return err
			},
		},
		{
			name: "DeletePrivateConnection",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.DeletePrivateConnection(ctx, &datastreampb.DeletePrivateConnectionRequest{Name: privateConnectionName})
				return err
			},
		},
		{
			name: "DeleteStream",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.DeleteStream(ctx, &datastreampb.DeleteStreamRequest{Name: streamName})
				return err
			},
		},
		{
			name: "DeleteConnectionProfile",
			call: func(ctx context.Context, c *datastream.Client) error {
				_, err := c.DeleteConnectionProfile(ctx, &datastreampb.DeleteConnectionProfileRequest{Name: connectionProfileName})
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
		fmt.Fprintf(os.Stderr, "warning: close datastream client: %v\n", err)
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
