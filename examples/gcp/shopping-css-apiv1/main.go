package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	css "cloud.google.com/go/shopping/css/apiv1"
	"cloud.google.com/go/shopping/css/apiv1/csspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
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

	accountID := getenv("STACKYARD_GCP_CSS_ACCOUNT_ID", "123456")
	rawProvidedID := getenv("STACKYARD_GCP_CSS_RAW_PROVIDED_ID", "sku-1")
	contentLanguage := getenv("STACKYARD_GCP_CSS_CONTENT_LANGUAGE", "en")
	feedLabel := strings.ToUpper(getenv("STACKYARD_GCP_CSS_FEED_LABEL", "US"))
	labelDisplayName := getenv("STACKYARD_GCP_CSS_LABEL_DISPLAY_NAME", "Summer 2026")

	accountName := fmt.Sprintf("accounts/%s", accountID)
	inputID := fmt.Sprintf("%s~%s~%s", contentLanguage, feedLabel, rawProvidedID)
	inputName := fmt.Sprintf("%s/cssProductInputs/%s", accountName, inputID)
	productName := fmt.Sprintf("%s/cssProducts/%s", accountName, inputID)

	fmt.Printf("Stackyard GCP Shopping CSS shopping/css/apiv1 clients using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, accountName); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-css-apiv1",
		},
	}

	accountsClient, err := css.NewAccountsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create css accounts client: %v", err)
	}
	defer closeClient("accounts", accountsClient.Close)

	labelsClient, err := css.NewAccountLabelsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create css account labels client: %v", err)
	}
	defer closeClient("account labels", labelsClient.Close)

	productsClient, err := css.NewCssProductsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create css products client: %v", err)
	}
	defer closeClient("css products", productsClient.Close)

	productInputsClient, err := css.NewCssProductInputsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create css product inputs client: %v", err)
	}
	defer closeClient("css product inputs", productInputsClient.Close)

	quotaClient, err := css.NewQuotaRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create css quota client: %v", err)
	}
	defer closeClient("quota", quotaClient.Close)

	createdLabelName := fmt.Sprintf("%s/labels/label-%s", accountName, slug(labelDisplayName))
	if strings.TrimSpace(createdLabelName) == fmt.Sprintf("%s/labels/label-", accountName) {
		createdLabelName = accountName + "/labels/label-1"
	}
	createdInputName := inputName
	finalProductName := productName

	calls := []callSpec{
		{
			name: "ListChildAccounts",
			call: func(ctx context.Context) error {
				it := accountsClient.ListChildAccounts(ctx, &csspb.ListChildAccountsRequest{
					Parent:   accountName,
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
			name: "GetAccount",
			call: func(ctx context.Context) error {
				_, err := accountsClient.GetAccount(ctx, &csspb.GetAccountRequest{Name: accountName})
				return err
			},
		},
		{
			name: "UpdateLabels",
			call: func(ctx context.Context) error {
				_, err := accountsClient.UpdateLabels(ctx, &csspb.UpdateAccountLabelsRequest{
					Name:     accountName,
					LabelIds: []int64{1001, 1002},
				})
				return err
			},
		},
		{
			name: "CreateAccountLabel",
			call: func(ctx context.Context) error {
				label, err := labelsClient.CreateAccountLabel(ctx, &csspb.CreateAccountLabelRequest{
					Parent: accountName,
					AccountLabel: &csspb.AccountLabel{
						DisplayName: proto.String(labelDisplayName),
						Description: proto.String("Seasonal promotional label"),
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(label.GetName()); name != "" {
					createdLabelName = name
				}
				return nil
			},
		},
		{
			name: "ListAccountLabels",
			call: func(ctx context.Context) error {
				it := labelsClient.ListAccountLabels(ctx, &csspb.ListAccountLabelsRequest{
					Parent:   accountName,
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
			name: "UpdateAccountLabel",
			call: func(ctx context.Context) error {
				_, err := labelsClient.UpdateAccountLabel(ctx, &csspb.UpdateAccountLabelRequest{
					AccountLabel: &csspb.AccountLabel{
						Name:        createdLabelName,
						DisplayName: proto.String(labelDisplayName + " Updated"),
						Description: proto.String("Updated seasonal promotional label"),
					},
				})
				return err
			},
		},
		{
			name: "InsertCssProductInput",
			call: func(ctx context.Context) error {
				input, err := productInputsClient.InsertCssProductInput(ctx, &csspb.InsertCssProductInputRequest{
					Parent: accountName,
					CssProductInput: &csspb.CssProductInput{
						RawProvidedId:   rawProvidedID,
						ContentLanguage: contentLanguage,
						FeedLabel:       feedLabel,
						Attributes:      &csspb.Attributes{Title: proto.String("Stackyard Tee")},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(input.GetName()); name != "" {
					createdInputName = name
				}
				if finalName := strings.TrimSpace(input.GetFinalName()); finalName != "" {
					finalProductName = finalName
				}
				return nil
			},
		},
		{
			name: "UpdateCssProductInput",
			call: func(ctx context.Context) error {
				_, err := productInputsClient.UpdateCssProductInput(ctx, &csspb.UpdateCssProductInputRequest{
					CssProductInput: &csspb.CssProductInput{
						Name:            createdInputName,
						RawProvidedId:   rawProvidedID,
						ContentLanguage: contentLanguage,
						FeedLabel:       feedLabel,
						Attributes:      &csspb.Attributes{Title: proto.String("Stackyard Tee Updated")},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"attributes.title"}},
				})
				return err
			},
		},
		{
			name: "GetCssProduct",
			call: func(ctx context.Context) error {
				_, err := productsClient.GetCssProduct(ctx, &csspb.GetCssProductRequest{Name: finalProductName})
				return err
			},
		},
		{
			name: "ListCssProducts",
			call: func(ctx context.Context) error {
				it := productsClient.ListCssProducts(ctx, &csspb.ListCssProductsRequest{
					Parent:   accountName,
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
			name: "DeleteCssProductInput",
			call: func(ctx context.Context) error {
				return productInputsClient.DeleteCssProductInput(ctx, &csspb.DeleteCssProductInputRequest{Name: createdInputName})
			},
		},
		{
			name: "DeleteAccountLabel",
			call: func(ctx context.Context) error {
				return labelsClient.DeleteAccountLabel(ctx, &csspb.DeleteAccountLabelRequest{Name: createdLabelName})
			},
		},
		{
			name: "ListQuotaGroups",
			call: func(ctx context.Context) error {
				it := quotaClient.ListQuotaGroups(ctx, &csspb.ListQuotaGroupsRequest{
					Parent:   accountName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, accountName string) error {
	readyURL := strings.TrimRight(apiEndpoint, "/") + "/v1/" + accountName
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "shopping-css-apiv1")
		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("ready probe status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
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
