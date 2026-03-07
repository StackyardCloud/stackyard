package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	translatepb "cloud.google.com/go/translate/apiv3/translatepb"
)

func TestGCPStage4GRPCParity_Translate(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restTranslateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:translateText", []byte(`{
		"contents":["parity translate"],
		"targetLanguageCode":"es"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if restTranslateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest translate translateText, got %d body=%s", restTranslateResp.StatusCode, string(providerContractBody(t, restTranslateResp)))
	}
	restTranslateBody := providerContractJSONMap(t, restTranslateResp)
	restTranslations, ok := restTranslateBody["translations"].([]any)
	if !ok || len(restTranslations) == 0 {
		t.Fatalf("expected translations in rest payload, got %#v", restTranslateBody["translations"])
	}
	restFirstTranslation, _ := restTranslations[0].(map[string]any)
	restTranslatedText, _ := restFirstTranslation["translatedText"].(string)

	var grpcTranslateResp translatepb.TranslateTextResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTranslateTranslateTextMethod, &translatepb.TranslateTextRequest{
		Parent:             "projects/stackyard/locations/us-central1",
		Contents:           []string{"parity translate"},
		TargetLanguageCode: "es",
	}, &grpcTranslateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for translateText, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcTranslateResp.GetTranslations()) == 0 {
		t.Fatalf("expected grpc translations list")
	}
	if grpcTranslateResp.GetTranslations()[0].GetTranslatedText() != restTranslatedText {
		t.Fatalf("expected grpc translated text %q to match rest %q", grpcTranslateResp.GetTranslations()[0].GetTranslatedText(), restTranslatedText)
	}

	restListGlossariesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "translate",
	})
	if restListGlossariesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest translate list glossaries, got %d body=%s", restListGlossariesResp.StatusCode, string(providerContractBody(t, restListGlossariesResp)))
	}
	restListGlossariesBody := providerContractJSONMap(t, restListGlossariesResp)
	restGlossaries, ok := restListGlossariesBody["glossaries"].([]any)
	if !ok || len(restGlossaries) == 0 {
		t.Fatalf("expected glossaries list in rest payload, got %#v", restListGlossariesBody["glossaries"])
	}
	restGlossary, _ := restGlossaries[0].(map[string]any)
	restGlossaryName, _ := restGlossary["name"].(string)

	var grpcListGlossariesResp translatepb.ListGlossariesResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTranslateListGlossariesMethod, &translatepb.ListGlossariesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListGlossariesResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list glossaries, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListGlossariesResp.GetGlossaries()) == 0 {
		t.Fatalf("expected grpc glossaries list")
	}
	if grpcListGlossariesResp.GetGlossaries()[0].GetName() != restGlossaryName {
		t.Fatalf("expected grpc glossary name %q to match rest %q", grpcListGlossariesResp.GetGlossaries()[0].GetName(), restGlossaryName)
	}

	restBatchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:batchTranslateText", []byte(`{
		"sourceLanguageCode":"en",
		"targetLanguageCodes":["es"],
		"inputConfigs":[{"mimeType":"text/plain"}],
		"outputConfig":{"gcsDestination":{"outputUriPrefix":"gs://stackyard/output/"}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if restBatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest translate batchTranslateText, got %d body=%s", restBatchResp.StatusCode, string(providerContractBody(t, restBatchResp)))
	}
	restBatchBody := providerContractJSONMap(t, restBatchResp)
	restOperationName, _ := restBatchBody["name"].(string)

	var grpcBatchResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTranslateBatchTranslateTextMethod, &translatepb.BatchTranslateTextRequest{
		Parent:             "projects/stackyard/locations/us-central1",
		SourceLanguageCode: "en",
		TargetLanguageCodes: []string{
			"es",
		},
		InputConfigs: []*translatepb.InputConfig{
			{
				MimeType: "text/plain",
			},
		},
		OutputConfig: &translatepb.OutputConfig{},
	}, &grpcBatchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for batchTranslateText, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcBatchResp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", grpcBatchResp.GetName(), restOperationName)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTranslateTranslateTextMethod, &translatepb.TranslateTextRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		Contents: []string{"missing target"},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "target_language_code-required") {
		t.Fatalf("expected grpc invalid argument for translate missing target, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTranslateGetGlossaryMethod, &translatepb.GetGlossaryRequest{
		Name: "projects/stackyard/locations/us-central1/glossaries/missing-glossary",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "glossary-not-found") {
		t.Fatalf("expected grpc not found for get glossary missing, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
