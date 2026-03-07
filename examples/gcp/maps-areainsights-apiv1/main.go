package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	areainsights "cloud.google.com/go/maps/areainsights/apiv1"
	"cloud.google.com/go/maps/areainsights/apiv1/areainsightspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *areainsights.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	fmt.Printf("Stackyard GCP Maps Places Aggregate (Area Insights) apiv1 client using %s\n", apiEndpoint)

	client, err := areainsights.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create maps areainsights client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ComputeInsights",
			call: func(ctx context.Context, c *areainsights.Client) error {
				_, err := c.ComputeInsights(ctx, &areainsightspb.ComputeInsightsRequest{
					Insights: []areainsightspb.Insight{
						areainsightspb.Insight_INSIGHT_COUNT,
						areainsightspb.Insight_INSIGHT_PLACES,
					},
					Filter: &areainsightspb.Filter{
						LocationFilter: &areainsightspb.LocationFilter{
							Area: &areainsightspb.LocationFilter_Circle_{
								Circle: &areainsightspb.LocationFilter_Circle{
									Center: &areainsightspb.LocationFilter_Circle_LatLng{
										LatLng: &latlngpb.LatLng{
											Latitude:  37.7937,
											Longitude: -122.3965,
										},
									},
									Radius: 1200,
								},
							},
						},
						TypeFilter: &areainsightspb.TypeFilter{
							IncludedTypes: []string{"restaurant"},
						},
					},
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
		fmt.Fprintf(os.Stderr, "warning: close maps areainsights client: %v\n", err)
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
