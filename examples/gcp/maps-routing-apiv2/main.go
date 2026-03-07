package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	routing "cloud.google.com/go/maps/routing/apiv2"
	"cloud.google.com/go/maps/routing/apiv2/routingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *routing.RoutesClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	fmt.Printf("Stackyard GCP Maps Routes apiv2 client using %s\n", apiEndpoint)

	client, err := routing.NewRoutesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create routes client: %v", err)
	}
	defer closeClient(client.Close)

	origin := waypoint(37.7937, -122.3965)
	destination := waypoint(37.7793, -122.4193)

	calls := []callSpec{
		{
			name: "ComputeRoutes",
			call: func(ctx context.Context, c *routing.RoutesClient) error {
				_, err := c.ComputeRoutes(ctx, &routingpb.ComputeRoutesRequest{
					Origin:      origin,
					Destination: destination,
					TravelMode:  routingpb.RouteTravelMode_DRIVE,
				})
				return err
			},
		},
		{
			name: "ComputeRouteMatrix",
			call: func(ctx context.Context, c *routing.RoutesClient) error {
				stream, err := c.ComputeRouteMatrix(ctx, &routingpb.ComputeRouteMatrixRequest{
					Origins: []*routingpb.RouteMatrixOrigin{
						{Waypoint: origin},
					},
					Destinations: []*routingpb.RouteMatrixDestination{
						{Waypoint: destination},
					},
					TravelMode: routingpb.RouteTravelMode_DRIVE,
				})
				if err != nil {
					return err
				}

				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", c.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", c.name)
		default:
			exitf("%s failed: %v", c.name, err)
		}
	}

	fmt.Println("Done.")
}

func waypoint(lat, lng float64) *routingpb.Waypoint {
	return &routingpb.Waypoint{
		LocationType: &routingpb.Waypoint_Location{
			Location: &routingpb.Location{
				LatLng: &latlngpb.LatLng{
					Latitude:  lat,
					Longitude: lng,
				},
			},
		},
	}
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
		fmt.Fprintf(os.Stderr, "warning: close routes client: %v\n", err)
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
