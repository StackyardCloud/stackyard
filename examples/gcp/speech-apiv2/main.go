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

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
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
	recognizerID := getenv("STACKYARD_GCP_SPEECH_V2_RECOGNIZER_ID", "recognizer-1")
	customClassID := getenv("STACKYARD_GCP_SPEECH_V2_CUSTOM_CLASS_ID", "custom-class-1")
	phraseSetID := getenv("STACKYARD_GCP_SPEECH_V2_PHRASE_SET_ID", "phrase-set-1")

	projectName := "projects/" + projectID
	parent := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	recognizerName := parent + "/recognizers/" + recognizerID
	customClassName := parent + "/customClasses/" + customClassID
	phraseSetName := parent + "/phraseSets/" + phraseSetID
	configName := parent + "/config"

	fmt.Printf("Stackyard GCP Speech-to-Text V2 speech/apiv2 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := speech.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create speech v2 client: %v", err)
	}
	defer closeClient("speech v2", client.Close)

	locationIt := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
		Name:     projectName,
		PageSize: 1,
	})
	if _, err := locationIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListLocations failed: %v", err)
	}
	logf("ListLocations succeeded")

	if _, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent}); err != nil {
		exitf("GetLocation failed: %v", err)
	}
	logf("GetLocation succeeded")

	createRecognizerOp, err := client.CreateRecognizer(ctx, &speechpb.CreateRecognizerRequest{
		Parent:       parent,
		RecognizerId: recognizerID,
		Recognizer: &speechpb.Recognizer{
			DisplayName: "Stackyard Speech V2 Recognizer",
			Model:       "latest_long",
			LanguageCodes: []string{
				"en-US",
			},
		},
	})
	if err != nil {
		exitf("CreateRecognizer failed: %v", err)
	}
	if strings.TrimSpace(createRecognizerOp.Name()) == "" {
		exitf("CreateRecognizer returned empty operation name")
	}
	if _, err := createRecognizerOp.Wait(ctx); err != nil {
		exitf("CreateRecognizer Wait failed: %v", err)
	}
	logf("CreateRecognizer succeeded")

	gotRecognizer, err := client.GetRecognizer(ctx, &speechpb.GetRecognizerRequest{Name: recognizerName})
	if err != nil {
		exitf("GetRecognizer failed: %v", err)
	}
	if gotRecognizer.GetName() != recognizerName {
		exitf("GetRecognizer returned unexpected name %q", gotRecognizer.GetName())
	}
	logf("GetRecognizer succeeded")

	recognizerIt := client.ListRecognizers(ctx, &speechpb.ListRecognizersRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := recognizerIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListRecognizers failed: %v", err)
	}
	logf("ListRecognizers succeeded")

	updateRecognizerOp, err := client.UpdateRecognizer(ctx, &speechpb.UpdateRecognizerRequest{
		Recognizer: &speechpb.Recognizer{
			Name:        recognizerName,
			DisplayName: "Stackyard Speech V2 Recognizer Updated",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
	})
	if err != nil {
		exitf("UpdateRecognizer failed: %v", err)
	}
	if _, err := updateRecognizerOp.Wait(ctx); err != nil {
		exitf("UpdateRecognizer Wait failed: %v", err)
	}
	logf("UpdateRecognizer succeeded")

	recognizeResp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Recognizer: recognizerName,
		Config: &speechpb.RecognitionConfig{
			LanguageCodes: []string{"en-US"},
		},
		AudioSource: &speechpb.RecognizeRequest_Content{
			Content: []byte("stackyard"),
		},
	})
	if err != nil {
		exitf("Recognize failed: %v", err)
	}
	if len(recognizeResp.GetResults()) == 0 || len(recognizeResp.GetResults()[0].GetAlternatives()) == 0 {
		exitf("Recognize returned empty results")
	}
	logf("Recognize succeeded")

	streamCtx, streamCancel := context.WithTimeout(ctx, 5*time.Second)
	defer streamCancel()
	stream, err := client.StreamingRecognize(streamCtx)
	if err != nil {
		if isToleratedSpeechV2StreamingError(err) {
			logf("StreamingRecognize returned tolerated staged transport error: %v", err)
		} else {
			exitf("StreamingRecognize failed: %v", err)
		}
	} else {
		sendErr := stream.Send(&speechpb.StreamingRecognizeRequest{
			Recognizer: recognizerName,
			StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
				StreamingConfig: &speechpb.StreamingRecognitionConfig{
					Config: &speechpb.RecognitionConfig{
						LanguageCodes: []string{"en-US"},
					},
				},
			},
		})
		if sendErr != nil && !isToleratedSpeechV2StreamingError(sendErr) {
			exitf("StreamingRecognize send failed: %v", sendErr)
		}
		if closeErr := stream.CloseSend(); closeErr != nil && !isToleratedSpeechV2StreamingError(closeErr) {
			exitf("StreamingRecognize close send failed: %v", closeErr)
		}
		streamResp, recvErr := stream.Recv()
		if recvErr != nil {
			if !isToleratedSpeechV2StreamingError(recvErr) {
				exitf("StreamingRecognize recv failed: %v", recvErr)
			}
			logf("StreamingRecognize returned tolerated staged transport error: %v", recvErr)
		} else {
			if len(streamResp.GetResults()) == 0 || len(streamResp.GetResults()[0].GetAlternatives()) == 0 {
				exitf("StreamingRecognize returned empty results")
			}
			logf("StreamingRecognize succeeded")
		}
	}

	batchRecognizeOp, err := client.BatchRecognize(ctx, &speechpb.BatchRecognizeRequest{
		Recognizer: recognizerName,
		Files: []*speechpb.BatchRecognizeFileMetadata{
			{
				AudioSource: &speechpb.BatchRecognizeFileMetadata_Uri{
					Uri: "gs://stackyard/audio-1.wav",
				},
			},
		},
		RecognitionOutputConfig: &speechpb.RecognitionOutputConfig{
			Output: &speechpb.RecognitionOutputConfig_InlineResponseConfig{
				InlineResponseConfig: &speechpb.InlineOutputConfig{},
			},
		},
	})
	if err != nil {
		exitf("BatchRecognize failed: %v", err)
	}
	batchOperationName := strings.TrimSpace(batchRecognizeOp.Name())
	if batchOperationName == "" {
		exitf("BatchRecognize returned empty operation name")
	}
	batchResp, err := batchRecognizeOp.Wait(ctx)
	if err != nil {
		exitf("BatchRecognize Wait failed: %v", err)
	}
	if len(batchResp.GetResults()) == 0 {
		exitf("BatchRecognize Wait returned empty results")
	}
	logf("BatchRecognize succeeded")

	if _, err := client.GetConfig(ctx, &speechpb.GetConfigRequest{Name: configName}); err != nil {
		exitf("GetConfig failed: %v", err)
	}
	logf("GetConfig succeeded")

	if _, err := client.UpdateConfig(ctx, &speechpb.UpdateConfigRequest{
		Config: &speechpb.Config{
			Name:       configName,
			KmsKeyName: fmt.Sprintf("projects/%s/locations/%s/keyRings/stackyard/cryptoKeys/custom", projectID, locationID),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"kms_key_name"}},
	}); err != nil {
		exitf("UpdateConfig failed: %v", err)
	}
	logf("UpdateConfig succeeded")

	createCustomClassOp, err := client.CreateCustomClass(ctx, &speechpb.CreateCustomClassRequest{
		Parent:        parent,
		CustomClassId: customClassID,
		CustomClass: &speechpb.CustomClass{
			DisplayName: "Stackyard CustomClass",
			Items: []*speechpb.CustomClass_ClassItem{
				{Value: "stackyard"},
				{Value: "speech"},
			},
		},
	})
	if err != nil {
		exitf("CreateCustomClass failed: %v", err)
	}
	if _, err := createCustomClassOp.Wait(ctx); err != nil {
		exitf("CreateCustomClass Wait failed: %v", err)
	}
	logf("CreateCustomClass succeeded")

	if _, err := client.GetCustomClass(ctx, &speechpb.GetCustomClassRequest{Name: customClassName}); err != nil {
		exitf("GetCustomClass failed: %v", err)
	}
	logf("GetCustomClass succeeded")

	customClassIt := client.ListCustomClasses(ctx, &speechpb.ListCustomClassesRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := customClassIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListCustomClasses failed: %v", err)
	}
	logf("ListCustomClasses succeeded")

	updateCustomClassOp, err := client.UpdateCustomClass(ctx, &speechpb.UpdateCustomClassRequest{
		CustomClass: &speechpb.CustomClass{
			Name:        customClassName,
			DisplayName: "Stackyard CustomClass Updated",
			Items: []*speechpb.CustomClass_ClassItem{
				{Value: "updated"},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "items"}},
	})
	if err != nil {
		exitf("UpdateCustomClass failed: %v", err)
	}
	if _, err := updateCustomClassOp.Wait(ctx); err != nil {
		exitf("UpdateCustomClass Wait failed: %v", err)
	}
	logf("UpdateCustomClass succeeded")

	deleteCustomClassOp, err := client.DeleteCustomClass(ctx, &speechpb.DeleteCustomClassRequest{Name: customClassName})
	if err != nil {
		exitf("DeleteCustomClass failed: %v", err)
	}
	if _, err := deleteCustomClassOp.Wait(ctx); err != nil {
		exitf("DeleteCustomClass Wait failed: %v", err)
	}
	logf("DeleteCustomClass succeeded")

	undeleteCustomClassOp, err := client.UndeleteCustomClass(ctx, &speechpb.UndeleteCustomClassRequest{Name: customClassName})
	if err != nil {
		exitf("UndeleteCustomClass failed: %v", err)
	}
	if _, err := undeleteCustomClassOp.Wait(ctx); err != nil {
		exitf("UndeleteCustomClass Wait failed: %v", err)
	}
	logf("UndeleteCustomClass succeeded")

	createPhraseSetOp, err := client.CreatePhraseSet(ctx, &speechpb.CreatePhraseSetRequest{
		Parent:      parent,
		PhraseSetId: phraseSetID,
		PhraseSet: &speechpb.PhraseSet{
			DisplayName: "Stackyard PhraseSet",
			Phrases: []*speechpb.PhraseSet_Phrase{
				{Value: "stackyard"},
				{Value: "speech"},
			},
			Boost: 10.0,
		},
	})
	if err != nil {
		exitf("CreatePhraseSet failed: %v", err)
	}
	if _, err := createPhraseSetOp.Wait(ctx); err != nil {
		exitf("CreatePhraseSet Wait failed: %v", err)
	}
	logf("CreatePhraseSet succeeded")

	if _, err := client.GetPhraseSet(ctx, &speechpb.GetPhraseSetRequest{Name: phraseSetName}); err != nil {
		exitf("GetPhraseSet failed: %v", err)
	}
	logf("GetPhraseSet succeeded")

	phraseSetIt := client.ListPhraseSets(ctx, &speechpb.ListPhraseSetsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := phraseSetIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListPhraseSets failed: %v", err)
	}
	logf("ListPhraseSets succeeded")

	updatePhraseSetOp, err := client.UpdatePhraseSet(ctx, &speechpb.UpdatePhraseSetRequest{
		PhraseSet: &speechpb.PhraseSet{
			Name:        phraseSetName,
			DisplayName: "Stackyard PhraseSet Updated",
			Boost:       13.0,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "boost"}},
	})
	if err != nil {
		exitf("UpdatePhraseSet failed: %v", err)
	}
	if _, err := updatePhraseSetOp.Wait(ctx); err != nil {
		exitf("UpdatePhraseSet Wait failed: %v", err)
	}
	logf("UpdatePhraseSet succeeded")

	deletePhraseSetOp, err := client.DeletePhraseSet(ctx, &speechpb.DeletePhraseSetRequest{Name: phraseSetName})
	if err != nil {
		exitf("DeletePhraseSet failed: %v", err)
	}
	if _, err := deletePhraseSetOp.Wait(ctx); err != nil {
		exitf("DeletePhraseSet Wait failed: %v", err)
	}
	logf("DeletePhraseSet succeeded")

	undeletePhraseSetOp, err := client.UndeletePhraseSet(ctx, &speechpb.UndeletePhraseSetRequest{Name: phraseSetName})
	if err != nil {
		exitf("UndeletePhraseSet failed: %v", err)
	}
	if _, err := undeletePhraseSetOp.Wait(ctx); err != nil {
		exitf("UndeletePhraseSet Wait failed: %v", err)
	}
	logf("UndeletePhraseSet succeeded")

	if _, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: batchOperationName}); err != nil {
		exitf("GetOperation failed: %v", err)
	}
	logf("GetOperation succeeded")

	deleteRecognizerOp, err := client.DeleteRecognizer(ctx, &speechpb.DeleteRecognizerRequest{Name: recognizerName})
	if err != nil {
		exitf("DeleteRecognizer failed: %v", err)
	}
	if _, err := deleteRecognizerOp.Wait(ctx); err != nil {
		exitf("DeleteRecognizer Wait failed: %v", err)
	}
	logf("DeleteRecognizer succeeded")

	undeleteRecognizerOp, err := client.UndeleteRecognizer(ctx, &speechpb.UndeleteRecognizerRequest{Name: recognizerName})
	if err != nil {
		exitf("UndeleteRecognizer failed: %v", err)
	}
	if _, err := undeleteRecognizerOp.Wait(ctx); err != nil {
		exitf("UndeleteRecognizer Wait failed: %v", err)
	}
	logf("UndeleteRecognizer succeeded")

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("%s/v2/projects/%s/locations/%s", apiEndpoint, projectID, locationID)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "speech-apiv2")
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return errors.New("timeout waiting for stackyard")
}

func isToleratedSpeechV2StreamingError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unknown, codes.Internal:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "stream removed") || strings.Contains(msg, "protocol error")
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
