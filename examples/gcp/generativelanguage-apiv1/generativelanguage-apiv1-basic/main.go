package main

import (
	"context"
	"errors"
	"fmt"
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

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	model := getenv("STACKYARD_GCP_MODEL", "models/gemini-2.0-flash")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	fmt.Printf("Stackyard GCP Generative Language basic client using %s\n", apiEndpoint)

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

	modelIter := modelClient.ListModels(ctx, &generativelanguagepb.ListModelsRequest{
		PageSize: 1,
	})
	_, err = modelIter.Next()
	switch {
	case err == nil:
		logf("ListModels succeeded")
	case errors.Is(err, iterator.Done):
		logf("ListModels returned no models")
	case isToleratedNotImplemented(err):
		logf("ListModels returned NotImplemented (expected in staged emulation)")
	default:
		exitf("ListModels failed: %v", err)
	}

	_, err = generativeClient.GenerateContent(ctx, &generativelanguagepb.GenerateContentRequest{
		Model: model,
		Contents: []*generativelanguagepb.Content{
			userContent("Hello from Stackyard basic Generative Language example"),
		},
	})
	switch {
	case err == nil:
		logf("GenerateContent succeeded")
	case isToleratedNotImplemented(err):
		logf("GenerateContent returned NotImplemented (expected in staged emulation)")
	default:
		exitf("GenerateContent failed: %v", err)
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
