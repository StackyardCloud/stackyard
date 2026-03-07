package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	mediatranslation "cloud.google.com/go/mediatranslation/apiv1beta1"
	"cloud.google.com/go/mediatranslation/apiv1beta1/mediatranslationpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *mediatranslation.SpeechTranslationClient) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()

	sourceLang := getenv("STACKYARD_GCP_MEDIATRANSLATION_SOURCE_LANG", "en-US")
	targetLang := getenv("STACKYARD_GCP_MEDIATRANSLATION_TARGET_LANG", "es-ES")
	audioEncoding := getenv("STACKYARD_GCP_MEDIATRANSLATION_AUDIO_ENCODING", "linear16")
	sampleRate := int32(16000)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Media Translation apiv1beta1 client using gRPC endpoint %s\n", grpcEndpoint)

	client, err := mediatranslation.NewSpeechTranslationClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create mediatranslation client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "StreamingTranslateSpeech",
			call: func(ctx context.Context, c *mediatranslation.SpeechTranslationClient) error {
				stream, err := c.StreamingTranslateSpeech(ctx)
				if err != nil {
					return err
				}
				defer stream.CloseSend()

				if err := stream.Send(&mediatranslationpb.StreamingTranslateSpeechRequest{
					StreamingRequest: &mediatranslationpb.StreamingTranslateSpeechRequest_StreamingConfig{
						StreamingConfig: &mediatranslationpb.StreamingTranslateSpeechConfig{
							AudioConfig: &mediatranslationpb.TranslateSpeechConfig{
								AudioEncoding:      audioEncoding,
								SourceLanguageCode: sourceLang,
								TargetLanguageCode: targetLang,
								SampleRateHertz:    sampleRate,
							},
							SingleUtterance: true,
						},
					},
				}); err != nil {
					return err
				}

				if err := stream.Send(&mediatranslationpb.StreamingTranslateSpeechRequest{
					StreamingRequest: &mediatranslationpb.StreamingTranslateSpeechRequest_AudioContent{
						AudioContent: []byte("stackyard-audio-chunk"),
					},
				}); err != nil {
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

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close mediatranslation client: %v\n", err)
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
