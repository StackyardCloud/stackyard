package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)

	fmt.Printf("Stackyard GCP Text-to-Speech V1 texttospeech/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "texttospeech",
		},
	}

	client, err := texttospeech.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create texttospeech client: %v", err)
	}
	defer closeClient("texttospeech", client.Close)

	longAudioClient, err := texttospeech.NewTextToSpeechLongAudioSynthesizeRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create texttospeech long audio client: %v", err)
	}
	defer closeClient("texttospeech long audio", longAudioClient.Close)

	operationName := parent + "/operations/synthesizeLongAudio.op-1"

	calls := []callSpec{
		{
			name: "ListVoices",
			call: func(ctx context.Context) error {
				resp, err := client.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{
					LanguageCode: "en-US",
				})
				if err != nil {
					return err
				}
				if len(resp.GetVoices()) == 0 {
					return errors.New("ListVoices returned no voices")
				}
				return nil
			},
		},
		{
			name: "SynthesizeSpeech",
			call: func(ctx context.Context) error {
				resp, err := client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
					Input: &texttospeechpb.SynthesisInput{
						InputSource: &texttospeechpb.SynthesisInput_Text{Text: "hello from stackyard"},
					},
					Voice: &texttospeechpb.VoiceSelectionParams{
						LanguageCode: "en-US",
						SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
					},
					AudioConfig: &texttospeechpb.AudioConfig{
						AudioEncoding: texttospeechpb.AudioEncoding_MP3,
					},
				})
				if err != nil {
					return err
				}
				if len(resp.GetAudioContent()) == 0 {
					return errors.New("SynthesizeSpeech returned empty audio content")
				}
				return nil
			},
		},
		{
			name: "SynthesizeLongAudio",
			call: func(ctx context.Context) error {
				op, err := longAudioClient.SynthesizeLongAudio(ctx, &texttospeechpb.SynthesizeLongAudioRequest{
					Parent: parent,
					Input: &texttospeechpb.SynthesisInput{
						InputSource: &texttospeechpb.SynthesisInput_Text{Text: "stackyard long audio synthesis sample"},
					},
					AudioConfig: &texttospeechpb.AudioConfig{
						AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
					},
					OutputGcsUri: "gs://stackyard/output.wav",
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				operation, err := longAudioClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(operation.GetName()) == "" {
					return errors.New("GetOperation returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := longAudioClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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
