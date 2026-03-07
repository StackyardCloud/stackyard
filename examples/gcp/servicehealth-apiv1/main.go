package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	servicehealth "cloud.google.com/go/servicehealth/apiv1"
	"cloud.google.com/go/servicehealth/apiv1/servicehealthpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	project := getenv("STACKYARD_GCP_PROJECT", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "global")
	orgID := getenv("STACKYARD_GCP_ORGANIZATION", "123456789")

	projectParent := fmt.Sprintf("projects/%s", project)
	projectLocationName := fmt.Sprintf("projects/%s/locations/%s", project, location)
	orgParent := fmt.Sprintf("organizations/%s", orgID)
	orgLocationName := fmt.Sprintf("organizations/%s/locations/%s", orgID, location)

	eventParent := projectLocationName
	eventName := fmt.Sprintf("%s/events/event-1", projectLocationName)
	orgEventsParent := orgLocationName
	orgEventName := fmt.Sprintf("%s/organizationEvents/org-event-1", orgLocationName)
	orgImpactParent := orgLocationName
	orgImpactName := fmt.Sprintf("%s/organizationImpacts/impact-1", orgLocationName)

	fmt.Printf("Stackyard GCP Service Health apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "servicehealth",
		},
	}

	client, err := servicehealth.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create servicehealth client: %v", err)
	}
	defer closeClient("servicehealth", client.Close)

	calls := []callSpec{
		{
			name: "ListLocations(project)",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectParent,
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
			name: "GetLocation(project)",
			call: func(ctx context.Context) error {
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: projectLocationName})
				return err
			},
		},
		{
			name: "ListLocations(organization)",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     orgParent,
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
			name: "GetLocation(organization)",
			call: func(ctx context.Context) error {
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: orgLocationName})
				return err
			},
		},
		{
			name: "ListEvents",
			call: func(ctx context.Context) error {
				it := client.ListEvents(ctx, &servicehealthpb.ListEventsRequest{
					Parent:   eventParent,
					PageSize: 1,
					View:     servicehealthpb.EventView_EVENT_VIEW_FULL,
				})
				event, err := it.Next()
				if err == nil {
					eventName = event.GetName()
					return nil
				}
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetEvent",
			call: func(ctx context.Context) error {
				_, err := client.GetEvent(ctx, &servicehealthpb.GetEventRequest{Name: eventName})
				return err
			},
		},
		{
			name: "ListOrganizationEvents",
			call: func(ctx context.Context) error {
				it := client.ListOrganizationEvents(ctx, &servicehealthpb.ListOrganizationEventsRequest{
					Parent:   orgEventsParent,
					PageSize: 1,
					View:     servicehealthpb.OrganizationEventView_ORGANIZATION_EVENT_VIEW_FULL,
				})
				event, err := it.Next()
				if err == nil {
					orgEventName = event.GetName()
					return nil
				}
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetOrganizationEvent",
			call: func(ctx context.Context) error {
				_, err := client.GetOrganizationEvent(ctx, &servicehealthpb.GetOrganizationEventRequest{Name: orgEventName})
				return err
			},
		},
		{
			name: "ListOrganizationImpacts",
			call: func(ctx context.Context) error {
				it := client.ListOrganizationImpacts(ctx, &servicehealthpb.ListOrganizationImpactsRequest{
					Parent:   orgImpactParent,
					PageSize: 1,
				})
				impact, err := it.Next()
				if err == nil {
					orgImpactName = impact.GetName()
					return nil
				}
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetOrganizationImpact",
			call: func(ctx context.Context) error {
				_, err := client.GetOrganizationImpact(ctx, &servicehealthpb.GetOrganizationImpactRequest{Name: orgImpactName})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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
