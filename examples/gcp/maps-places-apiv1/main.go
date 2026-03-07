package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	places "cloud.google.com/go/maps/places/apiv1"
	"cloud.google.com/go/maps/places/apiv1/placespb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *places.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	placeID := getenv("STACKYARD_GCP_MAPS_PLACE_ID", "ChIJj61dQgK6j4AR4GeTYWZsKWw")
	photoRef := getenv("STACKYARD_GCP_MAPS_PHOTO_REF", "AW123")
	searchTextQuery := getenv("STACKYARD_GCP_MAPS_TEXT_QUERY", "coffee")
	autocompleteInput := getenv("STACKYARD_GCP_MAPS_AUTOCOMPLETE_INPUT", "cof")

	placeName := "places/" + placeID
	photoMediaName := placeName + "/photos/" + photoRef + "/media"

	fmt.Printf("Stackyard GCP Maps Places apiv1 client using %s\n", apiEndpoint)

	client, err := places.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create maps places client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "SearchText",
			call: func(ctx context.Context, c *places.Client) error {
				_, err := c.SearchText(ctx, &placespb.SearchTextRequest{
					TextQuery:      searchTextQuery,
					MaxResultCount: 1,
				})
				return err
			},
		},
		{
			name: "SearchNearby",
			call: func(ctx context.Context, c *places.Client) error {
				_, err := c.SearchNearby(ctx, &placespb.SearchNearbyRequest{
					IncludedTypes:  []string{"restaurant"},
					MaxResultCount: 1,
					LocationRestriction: &placespb.SearchNearbyRequest_LocationRestriction{
						Type: &placespb.SearchNearbyRequest_LocationRestriction_Circle{
							Circle: &placespb.Circle{
								Center: &latlngpb.LatLng{
									Latitude:  37.7937,
									Longitude: -122.3965,
								},
								Radius: 1000,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "AutocompletePlaces",
			call: func(ctx context.Context, c *places.Client) error {
				_, err := c.AutocompletePlaces(ctx, &placespb.AutocompletePlacesRequest{
					Input: autocompleteInput,
				})
				return err
			},
		},
		{
			name: "GetPlace",
			call: func(ctx context.Context, c *places.Client) error {
				_, err := c.GetPlace(ctx, &placespb.GetPlaceRequest{Name: placeName})
				return err
			},
		},
		{
			name: "GetPhotoMedia",
			call: func(ctx context.Context, c *places.Client) error {
				_, err := c.GetPhotoMedia(ctx, &placespb.GetPhotoMediaRequest{
					Name:             photoMediaName,
					MaxWidthPx:       400,
					SkipHttpRedirect: true,
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
		fmt.Fprintf(os.Stderr, "warning: close maps places client: %v\n", err)
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
