package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpTextToSpeechListVoicesMethod          = "/google.cloud.texttospeech.v1.TextToSpeech/ListVoices"
	gcpTextToSpeechSynthesizeSpeechMethod    = "/google.cloud.texttospeech.v1.TextToSpeech/SynthesizeSpeech"
	gcpTextToSpeechStreamingSynthesizeMethod = "/google.cloud.texttospeech.v1.TextToSpeech/StreamingSynthesize"

	gcpTextToSpeechSynthesizeLongAudioMethod = "/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/SynthesizeLongAudio"
	gcpTextToSpeechGetOperationMethod        = "/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/GetOperation"
	gcpTextToSpeechListOperationsMethod      = "/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/ListOperations"
)

func gcpStage4GRPCTextToSpeech(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTextToSpeechListVoicesMethod:
		return gcpStage4GRPCTextToSpeechListVoices(grpcReqBody)
	case gcpTextToSpeechSynthesizeSpeechMethod:
		return gcpStage4GRPCTextToSpeechSynthesizeSpeech(grpcReqBody)
	case gcpTextToSpeechStreamingSynthesizeMethod:
		return grpcUnimplemented("streaming-synthesize-unimplemented")
	case gcpTextToSpeechSynthesizeLongAudioMethod:
		return gcpStage4GRPCTextToSpeechSynthesizeLongAudio(grpcReqBody)
	case gcpTextToSpeechGetOperationMethod:
		return gcpStage4GRPCTextToSpeechGetOperation(grpcReqBody)
	case gcpTextToSpeechListOperationsMethod:
		return gcpStage4GRPCTextToSpeechListOperations(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTextToSpeechListVoices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &texttospeechpb.ListVoicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	items := gcpStage4TextToSpeechVoiceFixtures()
	languageCode := strings.ToLower(strings.TrimSpace(req.GetLanguageCode()))
	if languageCode != "" {
		filtered := make([]*texttospeechpb.Voice, 0, len(items))
		for _, voice := range items {
			if gcpStage4TextToSpeechVoiceMatchesLanguage(voice, languageCode) {
				filtered = append(filtered, voice)
			}
		}
		items = filtered
	}
	return grpcProtoSuccess(&texttospeechpb.ListVoicesResponse{Voices: items})
}

func gcpStage4GRPCTextToSpeechSynthesizeSpeech(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &texttospeechpb.SynthesizeSpeechRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !gcpStage4TextToSpeechValidateSynthesisInput(req.GetInput()) {
		return grpcInvalidArgument("input-required")
	}
	if !gcpStage4TextToSpeechValidateAudioConfig(req.GetAudioConfig()) {
		return grpcInvalidArgument("audio_config-invalid")
	}
	return grpcProtoSuccess(&texttospeechpb.SynthesizeSpeechResponse{
		AudioContent: []byte("stackyard-audio"),
	})
}

func gcpStage4GRPCTextToSpeechSynthesizeLongAudio(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &texttospeechpb.SynthesizeLongAudioRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTextToSpeechLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !gcpStage4TextToSpeechValidateSynthesisInput(req.GetInput()) {
		return grpcInvalidArgument("input-required")
	}
	if !gcpStage4TextToSpeechValidateAudioConfig(req.GetAudioConfig()) {
		return grpcInvalidArgument("audio_config-invalid")
	}
	outputURI := strings.TrimSpace(req.GetOutputGcsUri())
	if outputURI == "" {
		return grpcInvalidArgument("output_gcs_uri-required")
	}
	if !strings.HasPrefix(outputURI, "gs://") {
		return grpcInvalidArgument("output_gcs_uri-invalid")
	}
	return grpcProtoSuccess(gcpStage4TextToSpeechOperation(project, location, "synthesizeLongAudio.op-1"))
}

func gcpStage4GRPCTextToSpeechGetOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.GetOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, operationID, ok := parseGCPTextToSpeechOperationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTextToSpeechMissingID(operationID) {
		return grpcNotFound("operation-not-found")
	}
	return grpcProtoSuccess(gcpStage4TextToSpeechOperation(project, location, operationID))
}

func gcpStage4GRPCTextToSpeechListOperations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.ListOperationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTextToSpeechLocationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	items := []*longrunningpb.Operation{
		gcpStage4TextToSpeechOperation(project, location, "synthesizeLongAudio.op-1"),
		gcpStage4TextToSpeechOperation(project, location, "synthesizeLongAudio.op-2"),
	}

	start, end, nextPageToken, reason, ok := gcpStage4TextToSpeechPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&longrunningpb.ListOperationsResponse{
		Operations:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4TextToSpeechValidateSynthesisInput(input *texttospeechpb.SynthesisInput) bool {
	if input == nil {
		return false
	}
	text := strings.TrimSpace(input.GetText())
	ssml := strings.TrimSpace(input.GetSsml())
	if text == "" && ssml == "" {
		return false
	}
	if text != "" && ssml != "" {
		return false
	}
	return true
}

func gcpStage4TextToSpeechValidateAudioConfig(audioConfig *texttospeechpb.AudioConfig) bool {
	if audioConfig == nil {
		return false
	}
	if audioConfig.GetAudioEncoding() == texttospeechpb.AudioEncoding_AUDIO_ENCODING_UNSPECIFIED {
		return false
	}
	if audioConfig.GetSampleRateHertz() < 0 {
		return false
	}
	return true
}

func gcpStage4TextToSpeechVoiceFixtures() []*texttospeechpb.Voice {
	return []*texttospeechpb.Voice{
		{
			LanguageCodes:          []string{"en-US", "en"},
			Name:                   "en-US-Standard-A",
			SsmlGender:             texttospeechpb.SsmlVoiceGender_FEMALE,
			NaturalSampleRateHertz: 24000,
		},
		{
			LanguageCodes:          []string{"en-GB", "en"},
			Name:                   "en-GB-Standard-B",
			SsmlGender:             texttospeechpb.SsmlVoiceGender_MALE,
			NaturalSampleRateHertz: 22050,
		},
		{
			LanguageCodes:          []string{"es-ES"},
			Name:                   "es-ES-Standard-A",
			SsmlGender:             texttospeechpb.SsmlVoiceGender_FEMALE,
			NaturalSampleRateHertz: 24000,
		},
	}
}

func gcpStage4TextToSpeechVoiceMatchesLanguage(voice *texttospeechpb.Voice, languageCode string) bool {
	for _, candidate := range voice.GetLanguageCodes() {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		if normalized == languageCode || strings.HasPrefix(normalized, languageCode+"-") || strings.HasPrefix(languageCode, normalized+"-") {
			return true
		}
	}
	return false
}

func gcpStage4TextToSpeechOperation(project, location, operationID string) *longrunningpb.Operation {
	metadataAny, err := anypb.New(&texttospeechpb.SynthesizeLongAudioMetadata{
		StartTime:      timestamppb.New(gcpStage4ReferenceTime),
		LastUpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
	})
	if err != nil {
		metadataAny = nil
	}
	responseAny, err := anypb.New(&texttospeechpb.SynthesizeLongAudioResponse{})
	if err != nil {
		responseAny = nil
	}

	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: true,
	}
	if metadataAny != nil {
		out.Metadata = metadataAny
	}
	if responseAny != nil {
		out.Result = &longrunningpb.Operation_Response{Response: responseAny}
	}
	return out
}

func gcpStage4TextToSpeechPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 {
		return 0, 0, "", "page_size-negative", false
	}
	if pageSize > int32(max) {
		return 0, 0, "", "page_size-too-large", false
	}
	start = 0
	if strings.TrimSpace(pageToken) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(pageToken))
		if err != nil || parsed < 0 {
			return 0, 0, "", "page_token-invalid", false
		}
		start = parsed
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}
	end = total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	nextPageToken = ""
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, "", true
}
