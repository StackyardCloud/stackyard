package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	issueresolution "cloud.google.com/go/shopping/merchant/issueresolution/apiv1"
	"cloud.google.com/go/shopping/merchant/issueresolution/apiv1/issueresolutionpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_ISSUERESOLUTION_ACCOUNT_ID", "123456")
	productID := getenv("STACKYARD_GCP_MERCHANT_ISSUERESOLUTION_PRODUCT_ID", "sku-1001")
	accountName := fmt.Sprintf("accounts/%s", accountID)
	productName := fmt.Sprintf("%s/products/%s", accountName, productID)

	fmt.Printf("Stackyard GCP Shopping Merchant Issue Resolution shopping/merchant/issueresolution/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, accountName); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-issueresolution-apiv1",
		},
	}

	issueClient, err := issueresolution.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant issueresolution client: %v", err)
	}
	defer closeClient("merchant issueresolution", issueClient.Close)

	aggregateClient, err := issueresolution.NewAggregateProductStatusesRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create aggregate product statuses client: %v", err)
	}
	defer closeClient("aggregate product statuses", aggregateClient.Close)

	actionContext := "ctx-account-review"
	actionFlowID := "flow-review"

	calls := []callSpec{
		{
			name: "RenderAccountIssues",
			call: func(ctx context.Context) error {
				resp, err := issueClient.RenderAccountIssues(ctx, &issueresolutionpb.RenderAccountIssuesRequest{
					Name: accountName,
					Payload: &issueresolutionpb.RenderIssuesRequestPayload{
						ContentOption:         issueresolutionpb.ContentOption_CONTENT_OPTION_UNSPECIFIED.Enum(),
						UserInputActionOption: issueresolutionpb.UserInputActionRenderingOption_USER_INPUT_ACTION_RENDERING_OPTION_UNSPECIFIED.Enum(),
					},
				})
				if err != nil {
					return err
				}
				if ctxValue, flowValue := firstActionContextAndFlow(resp.GetRenderedIssues()); ctxValue != "" && flowValue != "" {
					actionContext, actionFlowID = ctxValue, flowValue
				}
				return nil
			},
		},
		{
			name: "RenderProductIssues",
			call: func(ctx context.Context) error {
				_, err := issueClient.RenderProductIssues(ctx, &issueresolutionpb.RenderProductIssuesRequest{
					Name: productName,
					Payload: &issueresolutionpb.RenderIssuesRequestPayload{
						ContentOption:         issueresolutionpb.ContentOption_CONTENT_OPTION_UNSPECIFIED.Enum(),
						UserInputActionOption: issueresolutionpb.UserInputActionRenderingOption_USER_INPUT_ACTION_RENDERING_OPTION_UNSPECIFIED.Enum(),
					},
				})
				return err
			},
		},
		{
			name: "ListAggregateProductStatuses",
			call: func(ctx context.Context) error {
				it := aggregateClient.ListAggregateProductStatuses(ctx, &issueresolutionpb.ListAggregateProductStatusesRequest{
					Parent:   accountName,
					PageSize: 1,
					Filter:   `reporting_context = "SHOPPING_ADS" AND country = "US"`,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "TriggerAction",
			call: func(ctx context.Context) error {
				_, err := issueClient.TriggerAction(ctx, &issueresolutionpb.TriggerActionRequest{
					Name: accountName,
					Payload: &issueresolutionpb.TriggerActionPayload{
						ActionContext: actionContext,
						ActionInput: &issueresolutionpb.ActionInput{
							ActionFlowId: actionFlowID,
							InputValues: []*issueresolutionpb.InputValue{
								{
									InputFieldId: "explanation",
									Value: &issueresolutionpb.InputValue_TextInputValue_{
										TextInputValue: &issueresolutionpb.InputValue_TextInputValue{
											Value: "All issues were fixed and validated.",
										},
									},
								},
							},
						},
					},
				})
				return err
			},
		},
	}

	for _, spec := range calls {
		if err := spec.call(ctx); err != nil {
			exitf("%s failed: %v", spec.name, err)
		}
		fmt.Printf("%s succeeded\n", spec.name)
	}

	fmt.Println("Done.")
}

func firstActionContextAndFlow(renderedIssues []*issueresolutionpb.RenderedIssue) (string, string) {
	for _, issue := range renderedIssues {
		for _, action := range issue.GetActions() {
			userInputAction := action.GetBuiltinUserInputAction()
			if userInputAction == nil {
				continue
			}
			actionContext := strings.TrimSpace(userInputAction.GetActionContext())
			for _, flow := range userInputAction.GetFlows() {
				flowID := strings.TrimSpace(flow.GetId())
				if actionContext != "" && flowID != "" {
					return actionContext, flowID
				}
			}
		}
	}
	return "", ""
}

func waitForStackyardReady(ctx context.Context, endpoint, accountName string) error {
	target := strings.TrimRight(endpoint, "/") + "/issueresolution/v1/" + accountName + "/aggregateProductStatuses?pageSize=1"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for stackyard at %s", target)
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
	if strings.TrimSpace(t.serviceName) != "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return base.RoundTrip(clone)
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
