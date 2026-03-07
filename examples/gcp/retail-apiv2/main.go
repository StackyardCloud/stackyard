package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	retail "cloud.google.com/go/retail/apiv2"
	"cloud.google.com/go/retail/apiv2/retailpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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
	location := getenv("STACKYARD_GCP_LOCATION", "global")
	catalogID := getenv("STACKYARD_GCP_CATALOG_ID", "default_catalog")
	branchID := getenv("STACKYARD_GCP_BRANCH_ID", "default_branch")
	productID := getenv("STACKYARD_GCP_PRODUCT_ID", "product-1")
	servingConfigID := getenv("STACKYARD_GCP_SERVING_CONFIG_ID", "default_config")
	controlID := getenv("STACKYARD_GCP_CONTROL_ID", "control-1")
	modelID := getenv("STACKYARD_GCP_MODEL_ID", "model-1")
	placementID := getenv("STACKYARD_GCP_PLACEMENT_ID", "default_search")

	projectLocation := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	catalogName := fmt.Sprintf("%s/catalogs/%s", projectLocation, catalogID)
	branchName := fmt.Sprintf("%s/branches/%s", catalogName, branchID)
	productName := fmt.Sprintf("%s/products/%s", branchName, productID)
	servingConfigName := fmt.Sprintf("%s/servingConfigs/%s", catalogName, servingConfigID)
	controlName := fmt.Sprintf("%s/controls/%s", catalogName, controlID)
	modelName := fmt.Sprintf("%s/models/%s", catalogName, modelID)
	placementName := fmt.Sprintf("%s/placements/%s", catalogName, placementID)
	completionConfigName := catalogName + "/completionConfig"
	attributesConfigName := catalogName + "/attributesConfig"

	fmt.Printf("Stackyard GCP Vertex AI Search for commerce retail/apiv2 clients using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "retail",
		},
	}

	catalogClient, err := retail.NewCatalogRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail catalog client: %v", err)
	}
	defer closeClient("catalog", catalogClient.Close)

	productClient, err := retail.NewProductRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail product client: %v", err)
	}
	defer closeClient("product", productClient.Close)

	searchClient, err := retail.NewSearchRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail search client: %v", err)
	}
	defer closeClient("search", searchClient.Close)

	servingConfigClient, err := retail.NewServingConfigRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail serving config client: %v", err)
	}
	defer closeClient("serving config", servingConfigClient.Close)

	completionClient, err := retail.NewCompletionRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail completion client: %v", err)
	}
	defer closeClient("completion", completionClient.Close)

	predictionClient, err := retail.NewPredictionRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail prediction client: %v", err)
	}
	defer closeClient("prediction", predictionClient.Close)

	controlClient, err := retail.NewControlRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail control client: %v", err)
	}
	defer closeClient("control", controlClient.Close)

	userEventClient, err := retail.NewUserEventRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail user event client: %v", err)
	}
	defer closeClient("user event", userEventClient.Close)

	modelClient, err := retail.NewModelRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail model client: %v", err)
	}
	defer closeClient("model", modelClient.Close)

	analyticsClient, err := retail.NewAnalyticsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail analytics client: %v", err)
	}
	defer closeClient("analytics", analyticsClient.Close)

	generativeQuestionClient, err := retail.NewGenerativeQuestionRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create retail generative question client: %v", err)
	}
	defer closeClient("generative question", generativeQuestionClient.Close)

	calls := []callSpec{
		{
			name: "ListCatalogs",
			call: func(ctx context.Context) error {
				it := catalogClient.ListCatalogs(ctx, &retailpb.ListCatalogsRequest{Parent: projectLocation, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SetDefaultBranch",
			call: func(ctx context.Context) error {
				return catalogClient.SetDefaultBranch(ctx, &retailpb.SetDefaultBranchRequest{
					Catalog:  catalogName,
					BranchId: "0",
					Note:     "switch-to-default",
				})
			},
		},
		{
			name: "GetDefaultBranch",
			call: func(ctx context.Context) error {
				_, err := catalogClient.GetDefaultBranch(ctx, &retailpb.GetDefaultBranchRequest{Catalog: catalogName})
				return err
			},
		},
		{
			name: "GetCompletionConfig",
			call: func(ctx context.Context) error {
				_, err := catalogClient.GetCompletionConfig(ctx, &retailpb.GetCompletionConfigRequest{Name: completionConfigName})
				return err
			},
		},
		{
			name: "UpdateCompletionConfig",
			call: func(ctx context.Context) error {
				_, err := catalogClient.UpdateCompletionConfig(ctx, &retailpb.UpdateCompletionConfigRequest{
					CompletionConfig: &retailpb.CompletionConfig{
						Name:          completionConfigName,
						MatchingOrder: "exact-prefix",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"matching_order"}},
				})
				return err
			},
		},
		{
			name: "GetAttributesConfig",
			call: func(ctx context.Context) error {
				_, err := catalogClient.GetAttributesConfig(ctx, &retailpb.GetAttributesConfigRequest{Name: attributesConfigName})
				return err
			},
		},
		{
			name: "AddCatalogAttribute",
			call: func(ctx context.Context) error {
				_, err := catalogClient.AddCatalogAttribute(ctx, &retailpb.AddCatalogAttributeRequest{
					AttributesConfig: attributesConfigName,
					CatalogAttribute: &retailpb.CatalogAttribute{Key: "size"},
				})
				return err
			},
		},
		{
			name: "CreateProduct",
			call: func(ctx context.Context) error {
				_, err := productClient.CreateProduct(ctx, &retailpb.CreateProductRequest{
					Parent:    branchName,
					ProductId: productID,
					Product: &retailpb.Product{
						Title: "Stackyard Hoodie",
						Type:  retailpb.Product_PRIMARY,
					},
				})
				return err
			},
		},
		{
			name: "GetProduct",
			call: func(ctx context.Context) error {
				_, err := productClient.GetProduct(ctx, &retailpb.GetProductRequest{Name: productName})
				return err
			},
		},
		{
			name: "ListProducts",
			call: func(ctx context.Context) error {
				it := productClient.ListProducts(ctx, &retailpb.ListProductsRequest{Parent: branchName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateProduct",
			call: func(ctx context.Context) error {
				_, err := productClient.UpdateProduct(ctx, &retailpb.UpdateProductRequest{
					Product: &retailpb.Product{
						Name:  productName,
						Title: "Stackyard Hoodie Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
				})
				return err
			},
		},
		{
			name: "CreateServingConfig",
			call: func(ctx context.Context) error {
				_, err := servingConfigClient.CreateServingConfig(ctx, &retailpb.CreateServingConfigRequest{
					Parent:          catalogName,
					ServingConfigId: servingConfigID,
					ServingConfig: &retailpb.ServingConfig{
						DisplayName: "Default Serving Config",
					},
				})
				return err
			},
		},
		{
			name: "GetServingConfig",
			call: func(ctx context.Context) error {
				_, err := servingConfigClient.GetServingConfig(ctx, &retailpb.GetServingConfigRequest{Name: servingConfigName})
				return err
			},
		},
		{
			name: "ListServingConfigs",
			call: func(ctx context.Context) error {
				it := servingConfigClient.ListServingConfigs(ctx, &retailpb.ListServingConfigsRequest{Parent: catalogName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateControl",
			call: func(ctx context.Context) error {
				_, err := controlClient.CreateControl(ctx, &retailpb.CreateControlRequest{
					Parent:    catalogName,
					ControlId: controlID,
					Control: &retailpb.Control{
						DisplayName: "Typo Replacement",
					},
				})
				return err
			},
		},
		{
			name: "AddControl",
			call: func(ctx context.Context) error {
				_, err := servingConfigClient.AddControl(ctx, &retailpb.AddControlRequest{
					ServingConfig: servingConfigName,
					ControlId:     controlID,
				})
				return err
			},
		},
		{
			name: "CompleteQuery",
			call: func(ctx context.Context) error {
				_, err := completionClient.CompleteQuery(ctx, &retailpb.CompleteQueryRequest{
					Catalog:   catalogName,
					Query:     "hood",
					VisitorId: "visitor-1",
				})
				return err
			},
		},
		{
			name: "Search",
			call: func(ctx context.Context) error {
				it := searchClient.Search(ctx, &retailpb.SearchRequest{Placement: placementName, Query: "hoodie", PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "Predict",
			call: func(ctx context.Context) error {
				_, err := predictionClient.Predict(ctx, &retailpb.PredictRequest{
					Placement: placementName,
					UserEvent: &retailpb.UserEvent{
						EventType: "search",
						VisitorId: "visitor-1",
					},
				})
				return err
			},
		},
		{
			name: "WriteUserEvent",
			call: func(ctx context.Context) error {
				_, err := userEventClient.WriteUserEvent(ctx, &retailpb.WriteUserEventRequest{
					Parent: catalogName,
					UserEvent: &retailpb.UserEvent{
						EventType: "detail-page-view",
						VisitorId: "visitor-1",
					},
				})
				return err
			},
		},
		{
			name: "CollectUserEvent",
			call: func(ctx context.Context) error {
				_, err := userEventClient.CollectUserEvent(ctx, &retailpb.CollectUserEventRequest{
					Parent:    catalogName,
					UserEvent: "eventType=detail-page-view&visitorId=visitor-1",
				})
				return err
			},
		},
		{
			name: "ImportUserEvents",
			call: func(ctx context.Context) error {
				op, err := userEventClient.ImportUserEvents(ctx, &retailpb.ImportUserEventsRequest{
					Parent: catalogName,
					InputConfig: &retailpb.UserEventInputConfig{
						Source: &retailpb.UserEventInputConfig_UserEventInlineSource{
							UserEventInlineSource: &retailpb.UserEventInlineSource{
								UserEvents: []*retailpb.UserEvent{{
									EventType: "detail-page-view",
									VisitorId: "visitor-1",
								}},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "CreateModel",
			call: func(ctx context.Context) error {
				op, err := modelClient.CreateModel(ctx, &retailpb.CreateModelRequest{
					Parent: catalogName,
					Model: &retailpb.Model{
						Name:        modelName,
						DisplayName: "Stackyard Model",
						Type:        "recommended-for-you",
					},
				})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "GetModel",
			call: func(ctx context.Context) error {
				_, err := modelClient.GetModel(ctx, &retailpb.GetModelRequest{Name: modelName})
				return err
			},
		},
		{
			name: "TuneModel",
			call: func(ctx context.Context) error {
				op, err := modelClient.TuneModel(ctx, &retailpb.TuneModelRequest{Name: modelName})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "ExportAnalyticsMetrics",
			call: func(ctx context.Context) error {
				op, err := analyticsClient.ExportAnalyticsMetrics(ctx, &retailpb.ExportAnalyticsMetricsRequest{
					Catalog: catalogName,
					OutputConfig: &retailpb.OutputConfig{
						Destination: &retailpb.OutputConfig_GcsDestination_{
							GcsDestination: &retailpb.OutputConfig_GcsDestination{
								OutputUriPrefix: "gs://stackyard-retail/metrics",
							},
						},
					},
				})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "GetGenerativeQuestionsFeatureConfig",
			call: func(ctx context.Context) error {
				_, err := generativeQuestionClient.GetGenerativeQuestionsFeatureConfig(ctx, &retailpb.GetGenerativeQuestionsFeatureConfigRequest{Catalog: catalogName})
				return err
			},
		},
		{
			name: "UpdateGenerativeQuestionsFeatureConfig",
			call: func(ctx context.Context) error {
				_, err := generativeQuestionClient.UpdateGenerativeQuestionsFeatureConfig(ctx, &retailpb.UpdateGenerativeQuestionsFeatureConfigRequest{
					GenerativeQuestionsFeatureConfig: &retailpb.GenerativeQuestionsFeatureConfig{
						Catalog:        catalogName,
						FeatureEnabled: true,
					},
				})
				return err
			},
		},
		{
			name: "ListGenerativeQuestionConfigs",
			call: func(ctx context.Context) error {
				_, err := generativeQuestionClient.ListGenerativeQuestionConfigs(ctx, &retailpb.ListGenerativeQuestionConfigsRequest{Parent: catalogName})
				return err
			},
		},
		{
			name: "BatchUpdateGenerativeQuestionConfigs",
			call: func(ctx context.Context) error {
				_, err := generativeQuestionClient.BatchUpdateGenerativeQuestionConfigs(ctx, &retailpb.BatchUpdateGenerativeQuestionConfigsRequest{
					Parent: catalogName,
					Requests: []*retailpb.UpdateGenerativeQuestionConfigRequest{{
						GenerativeQuestionConfig: &retailpb.GenerativeQuestionConfig{
							Catalog:       catalogName,
							Facet:         "brand",
							FinalQuestion: "Preferred brand?",
						},
					}},
				})
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

	_ = controlName
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
		fmt.Fprintf(os.Stderr, "warning: close retail %s client: %v\n", label, err)
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
