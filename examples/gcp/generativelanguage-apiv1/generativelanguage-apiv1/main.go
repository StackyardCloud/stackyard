package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	generativelanguage "cloud.google.com/go/ai/generativelanguage/apiv1"
	"cloud.google.com/go/ai/generativelanguage/apiv1/generativelanguagepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *generativelanguage.ModelClient, *generativelanguage.GenerativeClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	model := getenv("STACKYARD_GCP_MODEL", "models/gemini-2.0-flash")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	fmt.Printf("Stackyard GCP Generative Language advanced client using %s\n", apiEndpoint)

	modelClient, err := generativelanguage.NewModelRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create model client: %v", err)
	}
	defer closeClient("model", modelClient.Close)

	generativeClient, err := generativelanguage.NewGenerativeRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create generative client: %v", err)
	}
	defer closeClient("generative", generativeClient.Close)

	calls := []callSpec{
		{
			name: "ListModels",
			call: func(ctx context.Context, modelClient *generativelanguage.ModelClient, _ *generativelanguage.GenerativeClient) error {
				it := modelClient.ListModels(ctx, &generativelanguagepb.ListModelsRequest{PageSize: 2})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetModel",
			call: func(ctx context.Context, modelClient *generativelanguage.ModelClient, _ *generativelanguage.GenerativeClient) error {
				_, err := modelClient.GetModel(ctx, &generativelanguagepb.GetModelRequest{
					Name: model,
				})
				return err
			},
		},
		{
			name: "CountTokens",
			call: func(ctx context.Context, _ *generativelanguage.ModelClient, generativeClient *generativelanguage.GenerativeClient) error {
				_, err := generativeClient.CountTokens(ctx, &generativelanguagepb.CountTokensRequest{
					Model: model,
					Contents: []*generativelanguagepb.Content{
						userContent("count these tokens"),
					},
				})
				return err
			},
		},
		{
			name: "EmbedContent",
			call: func(ctx context.Context, _ *generativelanguage.ModelClient, generativeClient *generativelanguage.GenerativeClient) error {
				_, err := generativeClient.EmbedContent(ctx, &generativelanguagepb.EmbedContentRequest{
					Model:   model,
					Content: userContent("embedding sample one"),
				})
				return err
			},
		},
		{
			name: "BatchEmbedContents",
			call: func(ctx context.Context, _ *generativelanguage.ModelClient, generativeClient *generativelanguage.GenerativeClient) error {
				_, err := generativeClient.BatchEmbedContents(ctx, &generativelanguagepb.BatchEmbedContentsRequest{
					Model: model,
					Requests: []*generativelanguagepb.EmbedContentRequest{
						{
							Model:   model,
							Content: userContent("embedding sample two"),
						},
						{
							Model:   model,
							Content: userContent("embedding sample three"),
						},
					},
				})
				return err
			},
		},
		{
			name: "GenerateContent",
			call: func(ctx context.Context, _ *generativelanguage.ModelClient, generativeClient *generativelanguage.GenerativeClient) error {
				_, err := generativeClient.GenerateContent(ctx, &generativelanguagepb.GenerateContentRequest{
					Model: model,
					Contents: []*generativelanguagepb.Content{
						userContent("generate a short hello message"),
					},
				})
				return err
			},
		},
		{
			name: "StreamGenerateContent",
			call: func(ctx context.Context, _ *generativelanguage.ModelClient, generativeClient *generativelanguage.GenerativeClient) error {
				stream, err := generativeClient.StreamGenerateContent(ctx, &generativelanguagepb.GenerateContentRequest{
					Model: model,
					Contents: []*generativelanguagepb.Content{
						userContent("stream a short response"),
					},
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, modelClient, generativeClient)
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

func userContent(text string) *generativelanguagepb.Content {
	return &generativelanguagepb.Content{
		Role: "user",
		Parts: []*generativelanguagepb.Part{
			{
				Data: &generativelanguagepb.Part_Text{Text: text},
			},
		},
	}
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
