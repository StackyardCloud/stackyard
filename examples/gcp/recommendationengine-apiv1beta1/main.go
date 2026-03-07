package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	recommendationengine "cloud.google.com/go/recommendationengine/apiv1beta1"
	"cloud.google.com/go/recommendationengine/apiv1beta1/recommendationenginepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	catalogID := getenv("STACKYARD_GCP_CATALOG_ID", "default_catalog")
	eventStoreID := getenv("STACKYARD_GCP_EVENT_STORE_ID", "default_event_store")
	placementID := getenv("STACKYARD_GCP_PLACEMENT_ID", "home_page")
	catalogItemID := getenv("STACKYARD_GCP_CATALOG_ITEM_ID", "item-1")
	predictionAPIKey := getenv("STACKYARD_GCP_PREDICTION_API_KEY", "stackyard-api-key")

	catalogParent := fmt.Sprintf("projects/%s/locations/global/catalogs/%s", projectID, catalogID)
	eventStoreParent := catalogParent + "/eventStores/" + eventStoreID
	catalogItemName := catalogParent + "/catalogItems/" + catalogItemID
	placementName := eventStoreParent + "/placements/" + placementID
	predictionAPIKeyRegistrationName := eventStoreParent + "/predictionApiKeyRegistrations/" + predictionAPIKey

	fmt.Printf("Stackyard GCP Recommendations AI recommendationengine/apiv1beta1 clients using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "recommendationengine",
		},
	}

	catalogClient, err := recommendationengine.NewCatalogRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create recommendationengine catalog client: %v", err)
	}
	defer closeClient("catalog", catalogClient.Close)

	userEventClient, err := recommendationengine.NewUserEventRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create recommendationengine user event client: %v", err)
	}
	defer closeClient("userEvent", userEventClient.Close)

	predictionClient, err := recommendationengine.NewPredictionRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create recommendationengine prediction client: %v", err)
	}
	defer closeClient("prediction", predictionClient.Close)

	registryClient, err := recommendationengine.NewPredictionApiKeyRegistryRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create recommendationengine prediction api key registry client: %v", err)
	}
	defer closeClient("predictionApiKeyRegistry", registryClient.Close)

	calls := []callSpec{
		{
			name: "CreateCatalogItem",
			call: func(ctx context.Context) error {
				_, err := catalogClient.CreateCatalogItem(ctx, &recommendationenginepb.CreateCatalogItemRequest{
					Parent: catalogParent,
					CatalogItem: &recommendationenginepb.CatalogItem{
						Id:    catalogItemID,
						Title: "Stackyard Item",
						CategoryHierarchies: []*recommendationenginepb.CatalogItem_CategoryHierarchy{
							{Categories: []string{"Books", "Fiction"}},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetCatalogItem",
			call: func(ctx context.Context) error {
				_, err := catalogClient.GetCatalogItem(ctx, &recommendationenginepb.GetCatalogItemRequest{Name: catalogItemName})
				return err
			},
		},
		{
			name: "ListCatalogItems",
			call: func(ctx context.Context) error {
				it := catalogClient.ListCatalogItems(ctx, &recommendationenginepb.ListCatalogItemsRequest{Parent: catalogParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateCatalogItem",
			call: func(ctx context.Context) error {
				_, err := catalogClient.UpdateCatalogItem(ctx, &recommendationenginepb.UpdateCatalogItemRequest{
					Name: catalogItemName,
					CatalogItem: &recommendationenginepb.CatalogItem{
						Id:    catalogItemID,
						Title: "Stackyard Item Updated",
						CategoryHierarchies: []*recommendationenginepb.CatalogItem_CategoryHierarchy{
							{Categories: []string{"Books", "Fiction"}},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
				})
				return err
			},
		},
		{
			name: "ImportCatalogItems",
			call: func(ctx context.Context) error {
				_, err := catalogClient.ImportCatalogItems(ctx, &recommendationenginepb.ImportCatalogItemsRequest{
					Parent:    catalogParent,
					RequestId: "req-1",
					InputConfig: &recommendationenginepb.InputConfig{
						Source: &recommendationenginepb.InputConfig_CatalogInlineSource{
							CatalogInlineSource: &recommendationenginepb.CatalogInlineSource{
								CatalogItems: []*recommendationenginepb.CatalogItem{{
									Id:    catalogItemID,
									Title: "Stackyard Item",
									CategoryHierarchies: []*recommendationenginepb.CatalogItem_CategoryHierarchy{
										{Categories: []string{"Books", "Fiction"}},
									},
								}},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "WriteUserEvent",
			call: func(ctx context.Context) error {
				_, err := userEventClient.WriteUserEvent(ctx, &recommendationenginepb.WriteUserEventRequest{
					Parent: eventStoreParent,
					UserEvent: &recommendationenginepb.UserEvent{
						EventType: "detail-page-view",
						UserInfo: &recommendationenginepb.UserInfo{
							VisitorId: "visitor-1",
							UserId:    "user-1",
						},
						ProductEventDetail: &recommendationenginepb.ProductEventDetail{
							ProductDetails: []*recommendationenginepb.ProductDetail{{
								Id:       catalogItemID,
								Quantity: 1,
							}},
						},
					},
				})
				return err
			},
		},
		{
			name: "CollectUserEvent",
			call: func(ctx context.Context) error {
				_, err := userEventClient.CollectUserEvent(ctx, &recommendationenginepb.CollectUserEventRequest{
					Parent:    eventStoreParent,
					UserEvent: "eventType=detail-page-view&visitorId=visitor-1",
					Uri:       "https://example.com/items/" + catalogItemID,
					Ets:       1700000000000,
				})
				return err
			},
		},
		{
			name: "ListUserEvents",
			call: func(ctx context.Context) error {
				it := userEventClient.ListUserEvents(ctx, &recommendationenginepb.ListUserEventsRequest{Parent: eventStoreParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "PurgeUserEvents",
			call: func(ctx context.Context) error {
				_, err := userEventClient.PurgeUserEvents(ctx, &recommendationenginepb.PurgeUserEventsRequest{
					Parent: eventStoreParent,
					Filter: `eventType = "detail-page-view"`,
					Force:  false,
				})
				return err
			},
		},
		{
			name: "ImportUserEvents",
			call: func(ctx context.Context) error {
				_, err := userEventClient.ImportUserEvents(ctx, &recommendationenginepb.ImportUserEventsRequest{
					Parent:    eventStoreParent,
					RequestId: "req-2",
					InputConfig: &recommendationenginepb.InputConfig{
						Source: &recommendationenginepb.InputConfig_UserEventInlineSource{
							UserEventInlineSource: &recommendationenginepb.UserEventInlineSource{
								UserEvents: []*recommendationenginepb.UserEvent{{
									EventType: "detail-page-view",
									UserInfo: &recommendationenginepb.UserInfo{
										VisitorId: "visitor-1",
										UserId:    "user-1",
									},
									EventTime: timestamppb.Now(),
								}},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "Predict",
			call: func(ctx context.Context) error {
				it := predictionClient.Predict(ctx, &recommendationenginepb.PredictRequest{
					Name: placementName,
					UserEvent: &recommendationenginepb.UserEvent{
						EventType: "detail-page-view",
						UserInfo: &recommendationenginepb.UserInfo{
							VisitorId: "visitor-1",
							UserId:    "user-1",
						},
					},
					PageSize: 1,
					DryRun:   true,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreatePredictionApiKeyRegistration",
			call: func(ctx context.Context) error {
				_, err := registryClient.CreatePredictionApiKeyRegistration(ctx, &recommendationenginepb.CreatePredictionApiKeyRegistrationRequest{
					Parent: eventStoreParent,
					PredictionApiKeyRegistration: &recommendationenginepb.PredictionApiKeyRegistration{
						ApiKey: predictionAPIKey,
					},
				})
				return err
			},
		},
		{
			name: "ListPredictionApiKeyRegistrations",
			call: func(ctx context.Context) error {
				it := registryClient.ListPredictionApiKeyRegistrations(ctx, &recommendationenginepb.ListPredictionApiKeyRegistrationsRequest{Parent: eventStoreParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeletePredictionApiKeyRegistration",
			call: func(ctx context.Context) error {
				return registryClient.DeletePredictionApiKeyRegistration(ctx, &recommendationenginepb.DeletePredictionApiKeyRegistrationRequest{Name: predictionAPIKeyRegistrationName})
			},
		},
		{
			name: "DeleteCatalogItem",
			call: func(ctx context.Context) error {
				return catalogClient.DeleteCatalogItem(ctx, &recommendationenginepb.DeleteCatalogItemRequest{Name: catalogItemName})
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close recommendationengine %s client: %v\n", name, err)
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
