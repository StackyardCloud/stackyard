package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	recommender "cloud.google.com/go/recommender/apiv1"
	"cloud.google.com/go/recommender/apiv1/recommenderpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type callSpec struct {
	name string
	call func(context.Context, *recommender.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	insightTypeID := getenv("STACKYARD_GCP_INSIGHT_TYPE", "google.iam.policy.Insight")
	recommenderID := getenv("STACKYARD_GCP_RECOMMENDER", "google.compute.instance.MachineTypeRecommender")
	insightID := getenv("STACKYARD_GCP_INSIGHT_ID", "insight-1")
	recommendationID := getenv("STACKYARD_GCP_RECOMMENDATION_ID", "recommendation-1")

	insightParent := fmt.Sprintf("projects/%s/locations/%s/insightTypes/%s", projectID, location, insightTypeID)
	recommenderParent := fmt.Sprintf("projects/%s/locations/%s/recommenders/%s", projectID, location, recommenderID)
	insightName := insightParent + "/insights/" + insightID
	recommendationName := recommenderParent + "/recommendations/" + recommendationID
	recommenderConfigName := recommenderParent + "/config"
	insightTypeConfigName := insightParent + "/config"

	fmt.Printf("Stackyard GCP Recommender apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "recommender",
		},
	}

	client, err := recommender.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create recommender client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListInsights",
			call: func(ctx context.Context, c *recommender.Client) error {
				it := c.ListInsights(ctx, &recommenderpb.ListInsightsRequest{Parent: insightParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInsight",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.GetInsight(ctx, &recommenderpb.GetInsightRequest{Name: insightName})
				return err
			},
		},
		{
			name: "MarkInsightAccepted",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.MarkInsightAccepted(ctx, &recommenderpb.MarkInsightAcceptedRequest{
					Name: insightName,
					Etag: "etag-" + insightID,
					StateMetadata: map[string]string{
						"ticket": "INC-100",
					},
				})
				return err
			},
		},
		{
			name: "ListRecommendations",
			call: func(ctx context.Context, c *recommender.Client) error {
				it := c.ListRecommendations(ctx, &recommenderpb.ListRecommendationsRequest{Parent: recommenderParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRecommendation",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.GetRecommendation(ctx, &recommenderpb.GetRecommendationRequest{Name: recommendationName})
				return err
			},
		},
		{
			name: "MarkRecommendationClaimed",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.MarkRecommendationClaimed(ctx, &recommenderpb.MarkRecommendationClaimedRequest{
					Name: recommendationName,
					Etag: "etag-" + recommendationID,
					StateMetadata: map[string]string{
						"change-request": "CR-200",
					},
				})
				return err
			},
		},
		{
			name: "MarkRecommendationSucceeded",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.MarkRecommendationSucceeded(ctx, &recommenderpb.MarkRecommendationSucceededRequest{
					Name: recommendationName,
					Etag: "etag-" + recommendationID,
				})
				return err
			},
		},
		{
			name: "GetRecommenderConfig",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.GetRecommenderConfig(ctx, &recommenderpb.GetRecommenderConfigRequest{Name: recommenderConfigName})
				return err
			},
		},
		{
			name: "UpdateRecommenderConfig",
			call: func(ctx context.Context, c *recommender.Client) error {
				params, err := structpb.NewStruct(map[string]any{
					"lookbackDays":          45,
					"minimumSavingsPercent": 12,
				})
				if err != nil {
					return err
				}
				_, err = c.UpdateRecommenderConfig(ctx, &recommenderpb.UpdateRecommenderConfigRequest{
					RecommenderConfig: &recommenderpb.RecommenderConfig{
						Name: recommenderConfigName,
						RecommenderGenerationConfig: &recommenderpb.RecommenderGenerationConfig{
							Params: params,
						},
						DisplayName: "Stackyard Recommender Config Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "GetInsightTypeConfig",
			call: func(ctx context.Context, c *recommender.Client) error {
				_, err := c.GetInsightTypeConfig(ctx, &recommenderpb.GetInsightTypeConfigRequest{Name: insightTypeConfigName})
				return err
			},
		},
		{
			name: "UpdateInsightTypeConfig",
			call: func(ctx context.Context, c *recommender.Client) error {
				params, err := structpb.NewStruct(map[string]any{
					"lookbackDays": 21,
				})
				if err != nil {
					return err
				}
				_, err = c.UpdateInsightTypeConfig(ctx, &recommenderpb.UpdateInsightTypeConfigRequest{
					InsightTypeConfig: &recommenderpb.InsightTypeConfig{
						Name: insightTypeConfigName,
						InsightTypeGenerationConfig: &recommenderpb.InsightTypeGenerationConfig{
							Params: params,
						},
						DisplayName: "Stackyard Insight Type Config Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
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
		fmt.Fprintf(os.Stderr, "warning: close recommender client: %v\n", err)
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
