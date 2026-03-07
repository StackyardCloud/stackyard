package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	solar "cloud.google.com/go/maps/solar/apiv1"
	"cloud.google.com/go/maps/solar/apiv1/solarpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *solar.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	lat := parseFloat64(getenv("STACKYARD_GCP_SOLAR_LAT", "37.7937"), 37.7937)
	lng := parseFloat64(getenv("STACKYARD_GCP_SOLAR_LNG", "-122.3965"), -122.3965)
	radius := parseFloat32(getenv("STACKYARD_GCP_SOLAR_RADIUS_METERS", "100"), 100)
	geotiffID := getenv("STACKYARD_GCP_SOLAR_GEOTIFF_ID", "sample-geotiff-id")

	location := &latlngpb.LatLng{
		Latitude:  lat,
		Longitude: lng,
	}

	fmt.Printf("Stackyard GCP Maps Solar apiv1 client using %s\n", apiEndpoint)

	client, err := solar.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create solar client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "FindClosestBuildingInsights",
			call: func(ctx context.Context, c *solar.Client) error {
				_, err := c.FindClosestBuildingInsights(ctx, &solarpb.FindClosestBuildingInsightsRequest{
					Location: location,
				})
				return err
			},
		},
		{
			name: "GetDataLayers",
			call: func(ctx context.Context, c *solar.Client) error {
				_, err := c.GetDataLayers(ctx, &solarpb.GetDataLayersRequest{
					Location:     location,
					RadiusMeters: radius,
				})
				return err
			},
		},
		{
			name: "GetGeoTiff",
			call: func(ctx context.Context, c *solar.Client) error {
				_, err := c.GetGeoTiff(ctx, &solarpb.GetGeoTiffRequest{
					Id: geotiffID,
				})
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
		fmt.Fprintf(os.Stderr, "warning: close solar client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func parseFloat64(v string, fallback float64) float64 {
	var n float64
	if _, err := fmt.Sscanf(strings.TrimSpace(v), "%f", &n); err != nil {
		return fallback
	}
	return n
}

func parseFloat32(v string, fallback float32) float32 {
	var n float64
	if _, err := fmt.Sscanf(strings.TrimSpace(v), "%f", &n); err != nil {
		return fallback
	}
	return float32(n)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
