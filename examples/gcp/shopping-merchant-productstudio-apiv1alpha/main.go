package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	productstudio "cloud.google.com/go/shopping/merchant/productstudio/apiv1alpha"
	"cloud.google.com/go/shopping/merchant/productstudio/apiv1alpha/productstudiopb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_PRODUCTSTUDIO_ACCOUNT_ID", "123456")
	name := fmt.Sprintf("accounts/%s", accountID)

	fmt.Printf("Stackyard GCP Shopping Merchant Product Studio shopping/merchant/productstudio/apiv1alpha clients using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-productstudio-apiv1alpha",
		},
	}

	imageClient, err := productstudio.NewImageRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant product studio image client: %v", err)
	}
	defer closeClient("merchant product studio image", imageClient.Close)

	textClient, err := productstudio.NewTextSuggestionsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant product studio text suggestions client: %v", err)
	}
	defer closeClient("merchant product studio text suggestions", textClient.Close)

	generateImageResp, err := imageClient.GenerateProductImageBackground(ctx, &productstudiopb.GenerateProductImageBackgroundRequest{
		Name: name,
		OutputConfig: &productstudiopb.OutputImageConfig{
			ReturnImageUri: true,
		},
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageUri{
				ImageUri: "https://example.com/products/sku-1001.jpg",
			},
		},
		Config: &productstudiopb.GenerateImageBackgroundConfig{
			ProductDescription:    "Stackyard red dress",
			BackgroundDescription: "Clean studio backdrop",
		},
	})
	if err != nil {
		exitf("GenerateProductImageBackground failed: %v", err)
	}
	if generateImageResp.GetGeneratedImage() == nil || strings.TrimSpace(generateImageResp.GetGeneratedImage().GetName()) == "" {
		exitf("GenerateProductImageBackground returned empty generated image")
	}
	fmt.Printf("GenerateProductImageBackground succeeded: %s\n", generateImageResp.GetGeneratedImage().GetName())

	removeImageResp, err := imageClient.RemoveProductImageBackground(ctx, &productstudiopb.RemoveProductImageBackgroundRequest{
		Name: name,
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageBytes{
				ImageBytes: []byte("stackyard-input-image"),
			},
		},
		Config: &productstudiopb.RemoveImageBackgroundConfig{
			BackgroundColor: &productstudiopb.RgbColor{
				Red:   255,
				Green: 255,
				Blue:  255,
			},
		},
	})
	if err != nil {
		exitf("RemoveProductImageBackground failed: %v", err)
	}
	if removeImageResp.GetGeneratedImage() == nil || strings.TrimSpace(removeImageResp.GetGeneratedImage().GetName()) == "" {
		exitf("RemoveProductImageBackground returned empty generated image")
	}
	fmt.Printf("RemoveProductImageBackground succeeded: %s\n", removeImageResp.GetGeneratedImage().GetName())

	upscaleImageResp, err := imageClient.UpscaleProductImage(ctx, &productstudiopb.UpscaleProductImageRequest{
		Name: name,
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageUri{
				ImageUri: "https://example.com/products/sku-1001.jpg",
			},
		},
	})
	if err != nil {
		exitf("UpscaleProductImage failed: %v", err)
	}
	if upscaleImageResp.GetGeneratedImage() == nil || strings.TrimSpace(upscaleImageResp.GetGeneratedImage().GetName()) == "" {
		exitf("UpscaleProductImage returned empty generated image")
	}
	fmt.Printf("UpscaleProductImage succeeded: %s\n", upscaleImageResp.GetGeneratedImage().GetName())

	textResp, err := textClient.GenerateProductTextSuggestions(ctx, &productstudiopb.GenerateProductTextSuggestionsRequest{
		Name: name,
		ProductInfo: &productstudiopb.ProductInfo{
			ProductAttributes: map[string]string{
				"title": "Red Dress",
				"brand": "Stackyard",
				"color": "red",
				"size":  "M",
			},
		},
		OutputSpec: &productstudiopb.OutputSpec{
			WorkflowId:     proto.String("title"),
			Tone:           proto.String("playful"),
			TargetLanguage: proto.String("en"),
		},
	})
	if err != nil {
		exitf("GenerateProductTextSuggestions failed: %v", err)
	}
	if textResp.GetTitle() == nil || strings.TrimSpace(textResp.GetTitle().GetText()) == "" {
		exitf("GenerateProductTextSuggestions returned empty title")
	}
	if textResp.GetDescription() == nil || strings.TrimSpace(textResp.GetDescription().GetText()) == "" {
		exitf("GenerateProductTextSuggestions returned empty description")
	}
	fmt.Printf("GenerateProductTextSuggestions succeeded: title=%q\n", textResp.GetTitle().GetText())
	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_productstudio/sample?stackyard_contract_probe=1"
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
