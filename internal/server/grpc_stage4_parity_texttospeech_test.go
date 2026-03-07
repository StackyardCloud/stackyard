package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

func TestGCPStage4GRPCParity_TextToSpeech(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/voices?languageCode=en-US", nil, map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest texttospeech list voices, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restVoices, ok := restListBody["voices"].([]any)
	if !ok || len(restVoices) == 0 {
		t.Fatalf("expected voices list in rest payload, got %#v", restListBody["voices"])
	}
	restVoice, _ := restVoices[0].(map[string]any)
	restVoiceName, _ := restVoice["name"].(string)

	var listVoicesResp texttospeechpb.ListVoicesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechListVoicesMethod, &texttospeechpb.ListVoicesRequest{
		LanguageCode: "en-US",
	}, &listVoicesResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list voices, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listVoicesResp.GetVoices()) == 0 {
		t.Fatalf("expected grpc voices list")
	}
	if listVoicesResp.GetVoices()[0].GetName() != restVoiceName {
		t.Fatalf("expected grpc voice name %q to match rest %q", listVoicesResp.GetVoices()[0].GetName(), restVoiceName)
	}

	restSynthResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"input":{"text":"stackyard parity"},
		"audioConfig":{"audioEncoding":"MP3"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if restSynthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest texttospeech synthesize, got %d body=%s", restSynthResp.StatusCode, string(providerContractBody(t, restSynthResp)))
	}
	restSynthBody := providerContractJSONMap(t, restSynthResp)
	restAudioContent, _ := restSynthBody["audioContent"].(string)
	if strings.TrimSpace(restAudioContent) == "" {
		t.Fatalf("expected rest synth audioContent")
	}

	var synthResp texttospeechpb.SynthesizeSpeechResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechSynthesizeSpeechMethod, &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: "stackyard parity"},
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}, &synthResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for synthesize speech, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(synthResp.GetAudioContent()) == 0 {
		t.Fatalf("expected grpc synth audio content")
	}

	restLongAudioResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio", []byte(`{
		"input":{"text":"stackyard long audio"},
		"audioConfig":{"audioEncoding":"LINEAR16"},
		"outputGcsUri":"gs://stackyard/output.wav"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if restLongAudioResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest texttospeech synthesize long audio, got %d body=%s", restLongAudioResp.StatusCode, string(providerContractBody(t, restLongAudioResp)))
	}
	restLongAudioBody := providerContractJSONMap(t, restLongAudioResp)
	restOperationName, _ := restLongAudioBody["name"].(string)

	var longAudioOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechSynthesizeLongAudioMethod, &texttospeechpb.SynthesizeLongAudioRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: "stackyard long audio"},
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
		},
		OutputGcsUri: "gs://stackyard/output.wav",
	}, &longAudioOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for synthesize long audio, got %q message=%q", grpcStatus, grpcMessage)
	}
	if longAudioOp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", longAudioOp.GetName(), restOperationName)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechSynthesizeSpeechMethod, &texttospeechpb.SynthesizeSpeechRequest{
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "input-required") {
		t.Fatalf("expected grpc invalid argument for synthesize speech missing input, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechGetOperationMethod, &longrunningpb.GetOperationRequest{
		Name: "projects/stackyard/locations/us-central1/operations/missing-op",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "operation-not-found") {
		t.Fatalf("expected grpc not found for get operation, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTextToSpeechStreamingSynthesizeMethod, &texttospeechpb.StreamingSynthesizeRequest{}, nil)
	if grpcStatus != "12" || !strings.Contains(grpcMessage, "streaming-synthesize-unimplemented") {
		t.Fatalf("expected grpc unimplemented for streaming synthesize, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
