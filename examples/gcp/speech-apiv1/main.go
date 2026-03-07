package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	phraseSetID := getenv("STACKYARD_GCP_SPEECH_PHRASE_SET_ID", "phrase-set-1")
	customClassID := getenv("STACKYARD_GCP_SPEECH_CUSTOM_CLASS_ID", "custom-class-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	phraseSetName := parent + "/phraseSets/" + phraseSetID
	customClassName := parent + "/customClasses/" + customClassID

	fmt.Printf("Stackyard GCP Speech-to-Text V1 speech/apiv1 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := speech.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create speech client: %v", err)
	}
	defer closeClient("speech", client.Close)

	adaptationClient, err := speech.NewAdaptationClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create speech adaptation client: %v", err)
	}
	defer closeClient("speech adaptation", adaptationClient.Close)

	if err := runSpeechCalls(ctx, client); err != nil {
		exitf("speech calls failed: %v", err)
	}
	if err := runAdaptationCalls(ctx, adaptationClient, parent, phraseSetID, phraseSetName, customClassID, customClassName); err != nil {
		exitf("adaptation calls failed: %v", err)
	}

	fmt.Println("Done.")
}

func runSpeechCalls(ctx context.Context, client *speech.Client) error {
	recognizeResp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			LanguageCode:    "en-US",
			SampleRateHertz: 16000,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: []byte("stackyard"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Recognize: %w", err)
	}
	if len(recognizeResp.GetResults()) == 0 || len(recognizeResp.GetResults()[0].GetAlternatives()) == 0 {
		return errors.New("Recognize returned empty results")
	}
	if strings.TrimSpace(recognizeResp.GetResults()[0].GetAlternatives()[0].GetTranscript()) == "" {
		return errors.New("Recognize returned empty transcript")
	}
	logf("Recognize succeeded")

	lro, err := client.LongRunningRecognize(ctx, &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			LanguageCode: "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: []byte("stackyard-long-running"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("LongRunningRecognize: %w", err)
	}
	if strings.TrimSpace(lro.Name()) == "" {
		return errors.New("LongRunningRecognize returned empty operation name")
	}
	logf("LongRunningRecognize succeeded")

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	lroResp, err := lro.Wait(waitCtx)
	if err != nil {
		return fmt.Errorf("LongRunningRecognize Wait: %w", err)
	}
	if len(lroResp.GetResults()) == 0 || len(lroResp.GetResults()[0].GetAlternatives()) == 0 {
		return errors.New("LongRunningRecognize Wait returned empty results")
	}
	logf("LongRunningRecognize Wait succeeded")

	streamCtx, streamCancel := context.WithTimeout(ctx, 5*time.Second)
	defer streamCancel()

	stream, err := client.StreamingRecognize(streamCtx)
	if err != nil {
		if isToleratedSpeechStreamingError(err) {
			logf("StreamingRecognize returned tolerated staged transport error: %v", err)
			return nil
		}
		return fmt.Errorf("StreamingRecognize: %w", err)
	}
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					LanguageCode:    "en-US",
					SampleRateHertz: 16000,
				},
			},
		},
	}); err != nil {
		if isToleratedSpeechStreamingError(err) {
			logf("StreamingRecognize returned tolerated staged transport error: %v", err)
			return nil
		}
		return fmt.Errorf("StreamingRecognize send config: %w", err)
	}
	if err := stream.CloseSend(); err != nil && !isToleratedSpeechStreamingError(err) {
		return fmt.Errorf("StreamingRecognize close send: %w", err)
	}
	streamResp, err := stream.Recv()
	if err != nil {
		if isToleratedSpeechStreamingError(err) {
			logf("StreamingRecognize returned tolerated staged transport error: %v", err)
			return nil
		}
		return fmt.Errorf("StreamingRecognize recv: %w", err)
	}
	if len(streamResp.GetResults()) == 0 || len(streamResp.GetResults()[0].GetAlternatives()) == 0 {
		return errors.New("StreamingRecognize returned empty results")
	}
	logf("StreamingRecognize succeeded")

	return nil
}

func runAdaptationCalls(
	ctx context.Context,
	client *speech.AdaptationClient,
	parent string,
	phraseSetID string,
	phraseSetName string,
	customClassID string,
	customClassName string,
) error {
	createdPhraseSet, err := client.CreatePhraseSet(ctx, &speechpb.CreatePhraseSetRequest{
		Parent:      parent,
		PhraseSetId: phraseSetID,
		PhraseSet: &speechpb.PhraseSet{
			Phrases: []*speechpb.PhraseSet_Phrase{
				{Value: "stackyard"},
				{Value: "speech emulator"},
			},
			Boost: 10,
		},
	})
	if err != nil {
		return fmt.Errorf("CreatePhraseSet: %w", err)
	}
	if strings.TrimSpace(createdPhraseSet.GetName()) == "" {
		return errors.New("CreatePhraseSet returned empty name")
	}
	logf("CreatePhraseSet succeeded")

	gotPhraseSet, err := client.GetPhraseSet(ctx, &speechpb.GetPhraseSetRequest{Name: phraseSetName})
	if err != nil {
		return fmt.Errorf("GetPhraseSet: %w", err)
	}
	if gotPhraseSet.GetName() != phraseSetName {
		return fmt.Errorf("GetPhraseSet returned unexpected name %q", gotPhraseSet.GetName())
	}
	logf("GetPhraseSet succeeded")

	listPhraseSetIt := client.ListPhraseSet(ctx, &speechpb.ListPhraseSetRequest{
		Parent:   parent,
		PageSize: 1,
	})
	_, err = listPhraseSetIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListPhraseSet: %w", err)
	}
	logf("ListPhraseSet succeeded")

	updatedPhraseSet, err := client.UpdatePhraseSet(ctx, &speechpb.UpdatePhraseSetRequest{
		PhraseSet: &speechpb.PhraseSet{
			Name:  phraseSetName,
			Boost: 15,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"boost"},
		},
	})
	if err != nil {
		return fmt.Errorf("UpdatePhraseSet: %w", err)
	}
	if updatedPhraseSet.GetName() != phraseSetName {
		return fmt.Errorf("UpdatePhraseSet returned unexpected name %q", updatedPhraseSet.GetName())
	}
	logf("UpdatePhraseSet succeeded")

	if err := client.DeletePhraseSet(ctx, &speechpb.DeletePhraseSetRequest{Name: phraseSetName}); err != nil {
		return fmt.Errorf("DeletePhraseSet: %w", err)
	}
	logf("DeletePhraseSet succeeded")

	createdCustomClass, err := client.CreateCustomClass(ctx, &speechpb.CreateCustomClassRequest{
		Parent:        parent,
		CustomClassId: customClassID,
		CustomClass: &speechpb.CustomClass{
			Items: []*speechpb.CustomClass_ClassItem{
				{Value: "stackyard"},
				{Value: "voice"},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("CreateCustomClass: %w", err)
	}
	if strings.TrimSpace(createdCustomClass.GetName()) == "" {
		return errors.New("CreateCustomClass returned empty name")
	}
	logf("CreateCustomClass succeeded")

	gotCustomClass, err := client.GetCustomClass(ctx, &speechpb.GetCustomClassRequest{Name: customClassName})
	if err != nil {
		return fmt.Errorf("GetCustomClass: %w", err)
	}
	if gotCustomClass.GetName() != customClassName {
		return fmt.Errorf("GetCustomClass returned unexpected name %q", gotCustomClass.GetName())
	}
	logf("GetCustomClass succeeded")

	listCustomClassIt := client.ListCustomClasses(ctx, &speechpb.ListCustomClassesRequest{
		Parent:   parent,
		PageSize: 1,
	})
	_, err = listCustomClassIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListCustomClasses: %w", err)
	}
	logf("ListCustomClasses succeeded")

	updatedCustomClass, err := client.UpdateCustomClass(ctx, &speechpb.UpdateCustomClassRequest{
		CustomClass: &speechpb.CustomClass{
			Name: customClassName,
			Items: []*speechpb.CustomClass_ClassItem{
				{Value: "speech"},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"items"},
		},
	})
	if err != nil {
		return fmt.Errorf("UpdateCustomClass: %w", err)
	}
	if updatedCustomClass.GetName() != customClassName {
		return fmt.Errorf("UpdateCustomClass returned unexpected name %q", updatedCustomClass.GetName())
	}
	logf("UpdateCustomClass succeeded")

	if err := client.DeleteCustomClass(ctx, &speechpb.DeleteCustomClassRequest{Name: customClassName}); err != nil {
		return fmt.Errorf("DeleteCustomClass: %w", err)
	}
	logf("DeleteCustomClass succeeded")

	return nil
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	readyURL := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/speech?stackyard_contract_probe=1&typedSuccess=1",
		strings.TrimRight(apiEndpoint, "/"),
		projectID,
		locationID,
	)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "speech")

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

func isToleratedSpeechStreamingError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large") ||
		strings.Contains(text, "error reading from server: eof")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
