package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	language "cloud.google.com/go/language/apiv1"
	"cloud.google.com/go/language/apiv1/languagepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *language.Client, *languagepb.Document) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"
	text := getenv("STACKYARD_GCP_LANGUAGE_TEXT", "Stackyard emulates cloud APIs for deterministic local integration testing.")

	fmt.Printf("Stackyard GCP Cloud Natural Language apiv1 client using %s\n", apiEndpoint)

	client, err := language.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create language client: %v", err)
	}
	defer closeClient(client.Close)

	document := &languagepb.Document{
		Type: languagepb.Document_PLAIN_TEXT,
		Source: &languagepb.Document_Content{
			Content: text,
		},
		Language: "en",
	}

	calls := []callSpec{
		{
			name: "AnalyzeSentiment",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.AnalyzeSentiment(ctx, &languagepb.AnalyzeSentimentRequest{
					Document:     doc,
					EncodingType: languagepb.EncodingType_UTF8,
				})
				return err
			},
		},
		{
			name: "AnalyzeEntities",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.AnalyzeEntities(ctx, &languagepb.AnalyzeEntitiesRequest{
					Document:     doc,
					EncodingType: languagepb.EncodingType_UTF8,
				})
				return err
			},
		},
		{
			name: "AnalyzeEntitySentiment",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.AnalyzeEntitySentiment(ctx, &languagepb.AnalyzeEntitySentimentRequest{
					Document:     doc,
					EncodingType: languagepb.EncodingType_UTF8,
				})
				return err
			},
		},
		{
			name: "AnalyzeSyntax",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.AnalyzeSyntax(ctx, &languagepb.AnalyzeSyntaxRequest{
					Document:     doc,
					EncodingType: languagepb.EncodingType_UTF8,
				})
				return err
			},
		},
		{
			name: "ClassifyText",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.ClassifyText(ctx, &languagepb.ClassifyTextRequest{
					Document: doc,
				})
				return err
			},
		},
		{
			name: "ModerateText",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.ModerateText(ctx, &languagepb.ModerateTextRequest{
					Document: doc,
				})
				return err
			},
		},
		{
			name: "AnnotateText",
			call: func(ctx context.Context, c *language.Client, doc *languagepb.Document) error {
				_, err := c.AnnotateText(ctx, &languagepb.AnnotateTextRequest{
					Document: doc,
					Features: &languagepb.AnnotateTextRequest_Features{
						ExtractSyntax:            true,
						ExtractEntities:          true,
						ExtractDocumentSentiment: true,
						ExtractEntitySentiment:   true,
						ClassifyText:             true,
						ModerateText:             true,
					},
					EncodingType: languagepb.EncodingType_UTF8,
				})
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client, document)
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
		fmt.Fprintf(os.Stderr, "warning: close language client: %v\n", err)
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
