package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	translatepb "cloud.google.com/go/translate/apiv3/translatepb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpTranslateTranslateTextMethod           = "/google.cloud.translation.v3.TranslationService/TranslateText"
	gcpTranslateRomanizeTextMethod            = "/google.cloud.translation.v3.TranslationService/RomanizeText"
	gcpTranslateDetectLanguageMethod          = "/google.cloud.translation.v3.TranslationService/DetectLanguage"
	gcpTranslateGetSupportedLanguagesMethod   = "/google.cloud.translation.v3.TranslationService/GetSupportedLanguages"
	gcpTranslateTranslateDocumentMethod       = "/google.cloud.translation.v3.TranslationService/TranslateDocument"
	gcpTranslateBatchTranslateTextMethod      = "/google.cloud.translation.v3.TranslationService/BatchTranslateText"
	gcpTranslateBatchTranslateDocumentMethod  = "/google.cloud.translation.v3.TranslationService/BatchTranslateDocument"
	gcpTranslateCreateGlossaryMethod          = "/google.cloud.translation.v3.TranslationService/CreateGlossary"
	gcpTranslateUpdateGlossaryMethod          = "/google.cloud.translation.v3.TranslationService/UpdateGlossary"
	gcpTranslateListGlossariesMethod          = "/google.cloud.translation.v3.TranslationService/ListGlossaries"
	gcpTranslateGetGlossaryMethod             = "/google.cloud.translation.v3.TranslationService/GetGlossary"
	gcpTranslateDeleteGlossaryMethod          = "/google.cloud.translation.v3.TranslationService/DeleteGlossary"
	gcpTranslateGetGlossaryEntryMethod        = "/google.cloud.translation.v3.TranslationService/GetGlossaryEntry"
	gcpTranslateListGlossaryEntriesMethod     = "/google.cloud.translation.v3.TranslationService/ListGlossaryEntries"
	gcpTranslateCreateGlossaryEntryMethod     = "/google.cloud.translation.v3.TranslationService/CreateGlossaryEntry"
	gcpTranslateUpdateGlossaryEntryMethod     = "/google.cloud.translation.v3.TranslationService/UpdateGlossaryEntry"
	gcpTranslateDeleteGlossaryEntryMethod     = "/google.cloud.translation.v3.TranslationService/DeleteGlossaryEntry"
	gcpTranslateCreateDatasetMethod           = "/google.cloud.translation.v3.TranslationService/CreateDataset"
	gcpTranslateGetDatasetMethod              = "/google.cloud.translation.v3.TranslationService/GetDataset"
	gcpTranslateListDatasetsMethod            = "/google.cloud.translation.v3.TranslationService/ListDatasets"
	gcpTranslateDeleteDatasetMethod           = "/google.cloud.translation.v3.TranslationService/DeleteDataset"
	gcpTranslateCreateAdaptiveMtDatasetMethod = "/google.cloud.translation.v3.TranslationService/CreateAdaptiveMtDataset"
	gcpTranslateDeleteAdaptiveMtDatasetMethod = "/google.cloud.translation.v3.TranslationService/DeleteAdaptiveMtDataset"
	gcpTranslateGetAdaptiveMtDatasetMethod    = "/google.cloud.translation.v3.TranslationService/GetAdaptiveMtDataset"
	gcpTranslateListAdaptiveMtDatasetsMethod  = "/google.cloud.translation.v3.TranslationService/ListAdaptiveMtDatasets"
	gcpTranslateAdaptiveMtTranslateMethod     = "/google.cloud.translation.v3.TranslationService/AdaptiveMtTranslate"
	gcpTranslateGetAdaptiveMtFileMethod       = "/google.cloud.translation.v3.TranslationService/GetAdaptiveMtFile"
	gcpTranslateDeleteAdaptiveMtFileMethod    = "/google.cloud.translation.v3.TranslationService/DeleteAdaptiveMtFile"
	gcpTranslateImportAdaptiveMtFileMethod    = "/google.cloud.translation.v3.TranslationService/ImportAdaptiveMtFile"
	gcpTranslateListAdaptiveMtFilesMethod     = "/google.cloud.translation.v3.TranslationService/ListAdaptiveMtFiles"
	gcpTranslateListAdaptiveMtSentencesMethod = "/google.cloud.translation.v3.TranslationService/ListAdaptiveMtSentences"
	gcpTranslateImportDataMethod              = "/google.cloud.translation.v3.TranslationService/ImportData"
	gcpTranslateExportDataMethod              = "/google.cloud.translation.v3.TranslationService/ExportData"
	gcpTranslateListExamplesMethod            = "/google.cloud.translation.v3.TranslationService/ListExamples"
	gcpTranslateCreateModelMethod             = "/google.cloud.translation.v3.TranslationService/CreateModel"
	gcpTranslateListModelsMethod              = "/google.cloud.translation.v3.TranslationService/ListModels"
	gcpTranslateGetModelMethod                = "/google.cloud.translation.v3.TranslationService/GetModel"
	gcpTranslateDeleteModelMethod             = "/google.cloud.translation.v3.TranslationService/DeleteModel"
)

func gcpStage4GRPCTranslate(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTranslateTranslateTextMethod:
		return gcpStage4GRPCTranslateText(grpcReqBody)
	case gcpTranslateRomanizeTextMethod:
		return gcpStage4GRPCRomanizeText(grpcReqBody)
	case gcpTranslateDetectLanguageMethod:
		return gcpStage4GRPCDetectLanguage(grpcReqBody)
	case gcpTranslateGetSupportedLanguagesMethod:
		return gcpStage4GRPCGetSupportedLanguages(grpcReqBody)
	case gcpTranslateTranslateDocumentMethod:
		return gcpStage4GRPCTranslateDocument(grpcReqBody)
	case gcpTranslateBatchTranslateTextMethod:
		return gcpStage4GRPCBatchTranslateText(grpcReqBody)
	case gcpTranslateBatchTranslateDocumentMethod:
		return gcpStage4GRPCBatchTranslateDocument(grpcReqBody)
	case gcpTranslateCreateGlossaryMethod:
		return gcpStage4GRPCCreateGlossary(grpcReqBody)
	case gcpTranslateUpdateGlossaryMethod:
		return gcpStage4GRPCUpdateGlossary(grpcReqBody)
	case gcpTranslateListGlossariesMethod:
		return gcpStage4GRPCListGlossaries(grpcReqBody)
	case gcpTranslateGetGlossaryMethod:
		return gcpStage4GRPCGetGlossary(grpcReqBody)
	case gcpTranslateDeleteGlossaryMethod:
		return gcpStage4GRPCDeleteGlossary(grpcReqBody)
	case gcpTranslateGetGlossaryEntryMethod:
		return gcpStage4GRPCGetGlossaryEntry(grpcReqBody)
	case gcpTranslateListGlossaryEntriesMethod:
		return gcpStage4GRPCListGlossaryEntries(grpcReqBody)
	case gcpTranslateCreateGlossaryEntryMethod:
		return gcpStage4GRPCCreateGlossaryEntry(grpcReqBody)
	case gcpTranslateUpdateGlossaryEntryMethod:
		return gcpStage4GRPCUpdateGlossaryEntry(grpcReqBody)
	case gcpTranslateDeleteGlossaryEntryMethod:
		return gcpStage4GRPCDeleteGlossaryEntry(grpcReqBody)
	case gcpTranslateCreateDatasetMethod:
		return gcpStage4GRPCCreateDataset(grpcReqBody)
	case gcpTranslateGetDatasetMethod:
		return gcpStage4GRPCGetDataset(grpcReqBody)
	case gcpTranslateListDatasetsMethod:
		return gcpStage4GRPCListDatasets(grpcReqBody)
	case gcpTranslateDeleteDatasetMethod:
		return gcpStage4GRPCDeleteDataset(grpcReqBody)
	case gcpTranslateCreateAdaptiveMtDatasetMethod:
		return gcpStage4GRPCCreateAdaptiveMtDataset(grpcReqBody)
	case gcpTranslateDeleteAdaptiveMtDatasetMethod:
		return gcpStage4GRPCDeleteAdaptiveMtDataset(grpcReqBody)
	case gcpTranslateGetAdaptiveMtDatasetMethod:
		return gcpStage4GRPCGetAdaptiveMtDataset(grpcReqBody)
	case gcpTranslateListAdaptiveMtDatasetsMethod:
		return gcpStage4GRPCListAdaptiveMtDatasets(grpcReqBody)
	case gcpTranslateAdaptiveMtTranslateMethod:
		return gcpStage4GRPCAdaptiveMtTranslate(grpcReqBody)
	case gcpTranslateGetAdaptiveMtFileMethod:
		return gcpStage4GRPCGetAdaptiveMtFile(grpcReqBody)
	case gcpTranslateDeleteAdaptiveMtFileMethod:
		return gcpStage4GRPCDeleteAdaptiveMtFile(grpcReqBody)
	case gcpTranslateImportAdaptiveMtFileMethod:
		return gcpStage4GRPCImportAdaptiveMtFile(grpcReqBody)
	case gcpTranslateListAdaptiveMtFilesMethod:
		return gcpStage4GRPCListAdaptiveMtFiles(grpcReqBody)
	case gcpTranslateListAdaptiveMtSentencesMethod:
		return gcpStage4GRPCListAdaptiveMtSentences(grpcReqBody)
	case gcpTranslateImportDataMethod:
		return gcpStage4GRPCImportData(grpcReqBody)
	case gcpTranslateExportDataMethod:
		return gcpStage4GRPCExportData(grpcReqBody)
	case gcpTranslateListExamplesMethod:
		return gcpStage4GRPCListExamples(grpcReqBody)
	case gcpTranslateCreateModelMethod:
		return gcpStage4GRPCCreateModel(grpcReqBody)
	case gcpTranslateListModelsMethod:
		return gcpStage4GRPCListModels(grpcReqBody)
	case gcpTranslateGetModelMethod:
		return gcpStage4GRPCGetModel(grpcReqBody)
	case gcpTranslateDeleteModelMethod:
		return gcpStage4GRPCDeleteModel(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTranslateText(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.TranslateTextRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent())); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetTargetLanguageCode()) == "" {
		return grpcInvalidArgument("target_language_code-required")
	}
	if len(req.GetContents()) == 0 {
		return grpcInvalidArgument("contents-required")
	}
	items := make([]*translatepb.Translation, 0, len(req.GetContents()))
	for _, content := range req.GetContents() {
		if strings.TrimSpace(content) == "" {
			return grpcInvalidArgument("contents-empty")
		}
		items = append(items, &translatepb.Translation{
			TranslatedText:       fmt.Sprintf("[%s] %s", req.GetTargetLanguageCode(), content),
			DetectedLanguageCode: gcpStage4TranslateSourceLanguage(req.GetSourceLanguageCode()),
		})
	}
	return grpcProtoSuccess(&translatepb.TranslateTextResponse{Translations: items})
}

func gcpStage4GRPCRomanizeText(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.RomanizeTextRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent())); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if len(req.GetContents()) == 0 {
		return grpcInvalidArgument("contents-required")
	}
	items := make([]*translatepb.Romanization, 0, len(req.GetContents()))
	for _, content := range req.GetContents() {
		if strings.TrimSpace(content) == "" {
			return grpcInvalidArgument("contents-empty")
		}
		items = append(items, &translatepb.Romanization{
			RomanizedText:        gcpTranslateRomanizeString(content),
			DetectedLanguageCode: gcpStage4TranslateSourceLanguage(req.GetSourceLanguageCode()),
		})
	}
	return grpcProtoSuccess(&translatepb.RomanizeTextResponse{Romanizations: items})
}

func gcpStage4GRPCDetectLanguage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DetectLanguageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent())); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetContent()) == "" {
		return grpcInvalidArgument("content-required")
	}
	return grpcProtoSuccess(&translatepb.DetectLanguageResponse{
		Languages: []*translatepb.DetectedLanguage{
			{
				LanguageCode: "en",
				Confidence:   0.99,
			},
		},
	})
}

func gcpStage4GRPCGetSupportedLanguages(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetSupportedLanguagesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent())); !ok {
		return grpcInvalidArgument("parent-required")
	}
	return grpcProtoSuccess(&translatepb.SupportedLanguages{
		Languages: []*translatepb.SupportedLanguage{
			{
				LanguageCode:  "en",
				DisplayName:   "English",
				SupportSource: true,
				SupportTarget: true,
			},
			{
				LanguageCode:  "es",
				DisplayName:   "Spanish",
				SupportSource: true,
				SupportTarget: true,
			},
		},
	})
}

func gcpStage4GRPCTranslateDocument(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.TranslateDocumentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetTargetLanguageCode()) == "" {
		return grpcInvalidArgument("target_language_code-required")
	}
	if req.GetDocumentInputConfig() == nil {
		return grpcInvalidArgument("document_input_config-required")
	}
	return grpcProtoSuccess(&translatepb.TranslateDocumentResponse{
		DocumentTranslation: &translatepb.DocumentTranslation{
			ByteStreamOutputs:    [][]byte{[]byte("stackyard-translated-document")},
			MimeType:             "text/plain",
			DetectedLanguageCode: gcpStage4TranslateSourceLanguage(req.GetSourceLanguageCode()),
		},
		Model: fmt.Sprintf("projects/%s/locations/%s/models/general/nmt", project, location),
	})
}

func gcpStage4GRPCBatchTranslateText(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.BatchTranslateTextRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetSourceLanguageCode()) == "" {
		return grpcInvalidArgument("source_language_code-required")
	}
	if len(req.GetTargetLanguageCodes()) == 0 {
		return grpcInvalidArgument("target_language_codes-required")
	}
	if len(req.GetInputConfigs()) == 0 {
		return grpcInvalidArgument("input_configs-required")
	}
	if req.GetOutputConfig() == nil {
		return grpcInvalidArgument("output_config-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "batchTranslateText.stackyard"))
}

func gcpStage4GRPCBatchTranslateDocument(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.BatchTranslateDocumentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetSourceLanguageCode()) == "" {
		return grpcInvalidArgument("source_language_code-required")
	}
	if len(req.GetTargetLanguageCodes()) == 0 {
		return grpcInvalidArgument("target_language_codes-required")
	}
	if len(req.GetInputConfigs()) == 0 {
		return grpcInvalidArgument("input_configs-required")
	}
	if req.GetOutputConfig() == nil {
		return grpcInvalidArgument("output_config-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "batchTranslateDocument.stackyard"))
}

func gcpStage4GRPCCreateGlossary(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.CreateGlossaryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetGlossary() == nil {
		return grpcInvalidArgument("glossary-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "createGlossary.glossary-1"))
}

func gcpStage4GRPCUpdateGlossary(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.UpdateGlossaryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	glossary := req.GetGlossary()
	if glossary == nil {
		return grpcInvalidArgument("glossary-required")
	}
	parent, _, ok := gcpTranslateParseGlossaryName(strings.TrimSpace(glossary.GetName()))
	if !ok {
		return grpcInvalidArgument("glossary-name-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "updateGlossary.glossary-1"))
}

func gcpStage4GRPCListGlossaries(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListGlossariesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.Glossary{
		gcpStage4TranslateGlossary(parent, "glossary-1"),
		gcpStage4TranslateGlossary(parent, "glossary-2"),
	}
	return grpcProtoSuccess(&translatepb.ListGlossariesResponse{
		Glossaries:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCGetGlossary(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetGlossaryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, glossaryID, ok := gcpTranslateParseGlossaryName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(glossaryID, "missing") {
		return grpcNotFound("glossary-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateGlossary(parent, glossaryID))
}

func gcpStage4GRPCDeleteGlossary(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteGlossaryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, ok := gcpTranslateParseGlossaryName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "deleteGlossary.glossary-1"))
}

func gcpStage4GRPCGetGlossaryEntry(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetGlossaryEntryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, glossaryID, entryID, ok := gcpTranslateParseGlossaryEntryName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(entryID, "missing") {
		return grpcNotFound("glossary-entry-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateGlossaryEntry(parent, glossaryID, entryID))
}

func gcpStage4GRPCListGlossaryEntries(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListGlossaryEntriesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, glossaryID, ok := gcpTranslateParseGlossaryName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.GlossaryEntry{
		gcpStage4TranslateGlossaryEntry(parent, glossaryID, "entry-1"),
		gcpStage4TranslateGlossaryEntry(parent, glossaryID, "entry-2"),
	}
	return grpcProtoSuccess(&translatepb.ListGlossaryEntriesResponse{
		GlossaryEntries: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCCreateGlossaryEntry(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.CreateGlossaryEntryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, glossaryID, ok := gcpTranslateParseGlossaryName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	entry := req.GetGlossaryEntry()
	if entry == nil {
		return grpcInvalidArgument("glossary_entry-required")
	}
	entryID := "entry-1"
	if name := strings.TrimSpace(entry.GetName()); name != "" {
		_, _, parsedID, ok := gcpTranslateParseGlossaryEntryName(name)
		if !ok {
			return grpcInvalidArgument("glossary_entry-name-invalid")
		}
		entryID = parsedID
	}
	return grpcProtoSuccess(gcpStage4TranslateGlossaryEntry(parent, glossaryID, entryID))
}

func gcpStage4GRPCUpdateGlossaryEntry(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.UpdateGlossaryEntryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	entry := req.GetGlossaryEntry()
	if entry == nil {
		return grpcInvalidArgument("glossary_entry-required")
	}
	parent, glossaryID, entryID, ok := gcpTranslateParseGlossaryEntryName(strings.TrimSpace(entry.GetName()))
	if !ok {
		return grpcInvalidArgument("glossary_entry-name-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateGlossaryEntry(parent, glossaryID, entryID))
}

func gcpStage4GRPCDeleteGlossaryEntry(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteGlossaryEntryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := gcpTranslateParseGlossaryEntryName(strings.TrimSpace(req.GetName())); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCCreateDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.CreateDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetDataset() == nil {
		return grpcInvalidArgument("dataset-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "createDataset.dataset-1"))
}

func gcpStage4GRPCGetDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, datasetID, ok := gcpTranslateParseDatasetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(datasetID, "missing") {
		return grpcNotFound("dataset-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateDataset(parent, datasetID))
}

func gcpStage4GRPCListDatasets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListDatasetsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.Dataset{
		gcpStage4TranslateDataset(parent, "dataset-1"),
		gcpStage4TranslateDataset(parent, "dataset-2"),
	}
	return grpcProtoSuccess(&translatepb.ListDatasetsResponse{
		Datasets:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCDeleteDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, ok := gcpTranslateParseDatasetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "deleteDataset.dataset-1"))
}

func gcpStage4GRPCCreateAdaptiveMtDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.CreateAdaptiveMtDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	item := req.GetAdaptiveMtDataset()
	if item == nil {
		return grpcInvalidArgument("adaptive_mt_dataset-required")
	}
	id := "adaptive-dataset-1"
	if name := strings.TrimSpace(item.GetName()); name != "" {
		_, parsedID, ok := gcpTranslateParseAdaptiveDatasetName(name)
		if !ok {
			return grpcInvalidArgument("adaptive_mt_dataset-name-invalid")
		}
		id = parsedID
	}
	return grpcProtoSuccess(gcpStage4TranslateAdaptiveDataset(parent, id))
}

func gcpStage4GRPCDeleteAdaptiveMtDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteAdaptiveMtDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(strings.TrimSpace(req.GetName())); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCGetAdaptiveMtDataset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetAdaptiveMtDatasetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(datasetID, "missing") {
		return grpcNotFound("adaptive-dataset-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateAdaptiveDataset(parent, datasetID))
}

func gcpStage4GRPCListAdaptiveMtDatasets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListAdaptiveMtDatasetsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.AdaptiveMtDataset{
		gcpStage4TranslateAdaptiveDataset(parent, "adaptive-dataset-1"),
		gcpStage4TranslateAdaptiveDataset(parent, "adaptive-dataset-2"),
	}
	return grpcProtoSuccess(&translatepb.ListAdaptiveMtDatasetsResponse{
		AdaptiveMtDatasets: items[start:end],
		NextPageToken:      next,
	})
}

func gcpStage4GRPCAdaptiveMtTranslate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.AdaptiveMtTranslateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(strings.TrimSpace(req.GetParent())); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(strings.TrimSpace(req.GetDataset())); !ok {
		return grpcInvalidArgument("dataset-required")
	}
	if len(req.GetContent()) == 0 {
		return grpcInvalidArgument("content-required")
	}
	items := make([]*translatepb.AdaptiveMtTranslation, 0, len(req.GetContent()))
	for _, content := range req.GetContent() {
		items = append(items, &translatepb.AdaptiveMtTranslation{
			TranslatedText: "[adaptive] " + content,
		})
	}
	return grpcProtoSuccess(&translatepb.AdaptiveMtTranslateResponse{
		LanguageCode: "es",
		Translations: items,
	})
}

func gcpStage4GRPCGetAdaptiveMtFile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetAdaptiveMtFileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, datasetID, fileID, ok := gcpTranslateParseAdaptiveFileName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(fileID, "missing") {
		return grpcNotFound("adaptive-file-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateAdaptiveFile(parent, datasetID, fileID))
}

func gcpStage4GRPCDeleteAdaptiveMtFile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteAdaptiveMtFileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := gcpTranslateParseAdaptiveFileName(strings.TrimSpace(req.GetName())); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCImportAdaptiveMtFile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ImportAdaptiveMtFileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	rootParent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(parent)
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetFileInputSource() == nil && req.GetGcsInputSource() == nil {
		return grpcInvalidArgument("source-required")
	}
	return grpcProtoSuccess(&translatepb.ImportAdaptiveMtFileResponse{
		AdaptiveMtFile: gcpStage4TranslateAdaptiveFile(rootParent, datasetID, "file-1"),
	})
}

func gcpStage4GRPCListAdaptiveMtFiles(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListAdaptiveMtFilesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.AdaptiveMtFile{
		gcpStage4TranslateAdaptiveFile(parent, datasetID, "file-1"),
		gcpStage4TranslateAdaptiveFile(parent, datasetID, "file-2"),
	}
	return grpcProtoSuccess(&translatepb.ListAdaptiveMtFilesResponse{
		AdaptiveMtFiles: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCListAdaptiveMtSentences(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListAdaptiveMtSentencesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(parent); !ok {
		if _, _, _, ok := gcpTranslateParseAdaptiveFileName(parent); !ok {
			return grpcInvalidArgument("parent-required")
		}
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.AdaptiveMtSentence{
		gcpStage4TranslateAdaptiveSentence(parent, "sentence-1"),
		gcpStage4TranslateAdaptiveSentence(parent, "sentence-2"),
	}
	return grpcProtoSuccess(&translatepb.ListAdaptiveMtSentencesResponse{
		AdaptiveMtSentences: items[start:end],
		NextPageToken:       next,
	})
}

func gcpStage4GRPCImportData(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ImportDataRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, ok := gcpTranslateParseDatasetName(strings.TrimSpace(req.GetDataset()))
	if !ok {
		return grpcInvalidArgument("dataset-required")
	}
	if req.GetInputConfig() == nil {
		return grpcInvalidArgument("input_config-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "importData.dataset-1"))
}

func gcpStage4GRPCExportData(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ExportDataRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, ok := gcpTranslateParseDatasetName(strings.TrimSpace(req.GetDataset()))
	if !ok {
		return grpcInvalidArgument("dataset-required")
	}
	if req.GetOutputConfig() == nil {
		return grpcInvalidArgument("output_config-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "exportData.dataset-1"))
}

func gcpStage4GRPCListExamples(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListExamplesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	_, datasetID, ok := gcpTranslateParseDatasetName(parent)
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.Example{
		gcpStage4TranslateExample(parent, "example-1", datasetID),
		gcpStage4TranslateExample(parent, "example-2", datasetID),
	}
	return grpcProtoSuccess(&translatepb.ListExamplesResponse{
		Examples:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCCreateModel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.CreateModelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetModel() == nil {
		return grpcInvalidArgument("model-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "createModel.model-1"))
}

func gcpStage4GRPCListModels(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.ListModelsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, ok := gcpStage4TranslatePage(req.GetPageSize(), req.GetPageToken(), 2)
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*translatepb.Model{
		gcpStage4TranslateModel(parent, "model-1"),
		gcpStage4TranslateModel(parent, "model-2"),
	}
	return grpcProtoSuccess(&translatepb.ListModelsResponse{
		Models:        items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCGetModel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.GetModelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, modelID, ok := gcpTranslateParseModelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(modelID, "missing") {
		return grpcNotFound("model-not-found")
	}
	return grpcProtoSuccess(gcpStage4TranslateModel(parent, modelID))
}

func gcpStage4GRPCDeleteModel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &translatepb.DeleteModelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, ok := gcpTranslateParseModelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4TranslateOperation(parent, "deleteModel.model-1"))
}

func gcpStage4TranslatePage(pageSize int32, token string, total int) (int, int, string, bool) {
	if pageSize < 0 || pageSize > 1000 {
		return 0, 0, "", false
	}
	start := 0
	if strings.TrimSpace(token) != "" {
		value, err := strconv.Atoi(strings.TrimSpace(token))
		if err != nil || value < 0 || value > total {
			return 0, 0, "", false
		}
		start = value
	}
	end := total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	next := ""
	if end < total {
		next = strconv.Itoa(end)
	}
	return start, end, next, true
}

func gcpStage4TranslateOperation(parent, operationID string) *longrunningpb.Operation {
	anyResponse, _ := anypb.New(&emptypb.Empty{})
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("%s/operations/%s", parent, operationID),
		Done: true,
		Result: &longrunningpb.Operation_Response{
			Response: anyResponse,
		},
	}
}

func gcpStage4TranslateGlossary(parent, glossaryID string) *translatepb.Glossary {
	return &translatepb.Glossary{
		Name:        fmt.Sprintf("%s/glossaries/%s", parent, glossaryID),
		DisplayName: "Stackyard Glossary " + glossaryID,
		EntryCount:  2,
		SubmitTime:  timestamppb.New(gcpTranslateReferenceTime),
		EndTime:     timestamppb.New(gcpTranslateReferenceTime.Add(2 * time.Second)),
	}
}

func gcpStage4TranslateGlossaryEntry(parent, glossaryID, entryID string) *translatepb.GlossaryEntry {
	return &translatepb.GlossaryEntry{
		Name:        fmt.Sprintf("%s/glossaries/%s/glossaryEntries/%s", parent, glossaryID, entryID),
		Description: "Stackyard glossary entry " + entryID,
		Data: &translatepb.GlossaryEntry_TermsPair{
			TermsPair: &translatepb.GlossaryEntry_GlossaryTermsPair{
				SourceTerm: &translatepb.GlossaryTerm{
					LanguageCode: "en",
					Text:         "hello",
				},
				TargetTerm: &translatepb.GlossaryTerm{
					LanguageCode: "es",
					Text:         "hola",
				},
			},
		},
	}
}

func gcpStage4TranslateDataset(parent, datasetID string) *translatepb.Dataset {
	return &translatepb.Dataset{
		Name:               fmt.Sprintf("%s/datasets/%s", parent, datasetID),
		DisplayName:        "Stackyard Dataset " + datasetID,
		SourceLanguageCode: "en",
		TargetLanguageCode: "es",
		ExampleCount:       2,
		CreateTime:         timestamppb.New(gcpTranslateReferenceTime),
		UpdateTime:         timestamppb.New(gcpTranslateReferenceTime.Add(5 * time.Minute)),
	}
}

func gcpStage4TranslateAdaptiveDataset(parent, datasetID string) *translatepb.AdaptiveMtDataset {
	return &translatepb.AdaptiveMtDataset{
		Name:               fmt.Sprintf("%s/adaptiveMtDatasets/%s", parent, datasetID),
		DisplayName:        "Stackyard Adaptive Dataset " + datasetID,
		SourceLanguageCode: "en",
		TargetLanguageCode: "es",
		ExampleCount:       2,
		CreateTime:         timestamppb.New(gcpTranslateReferenceTime),
		UpdateTime:         timestamppb.New(gcpTranslateReferenceTime.Add(10 * time.Minute)),
	}
}

func gcpStage4TranslateAdaptiveFile(parent, datasetID, fileID string) *translatepb.AdaptiveMtFile {
	return &translatepb.AdaptiveMtFile{
		Name:        fmt.Sprintf("%s/adaptiveMtDatasets/%s/adaptiveMtFiles/%s", parent, datasetID, fileID),
		DisplayName: "Stackyard Adaptive File " + fileID,
		EntryCount:  2,
		CreateTime:  timestamppb.New(gcpTranslateReferenceTime),
		UpdateTime:  timestamppb.New(gcpTranslateReferenceTime.Add(15 * time.Minute)),
	}
}

func gcpStage4TranslateAdaptiveSentence(parent, sentenceID string) *translatepb.AdaptiveMtSentence {
	return &translatepb.AdaptiveMtSentence{
		Name:           fmt.Sprintf("%s/adaptiveMtSentences/%s", parent, sentenceID),
		SourceSentence: "hello",
		TargetSentence: "hola",
		CreateTime:     timestamppb.New(gcpTranslateReferenceTime),
		UpdateTime:     timestamppb.New(gcpTranslateReferenceTime.Add(20 * time.Minute)),
	}
}

func gcpStage4TranslateExample(parent, exampleID, datasetID string) *translatepb.Example {
	return &translatepb.Example{
		Name:       fmt.Sprintf("%s/examples/%s", parent, exampleID),
		SourceText: "source sentence for " + datasetID,
		TargetText: "translated sentence for " + datasetID,
		Usage:      "TRAIN",
	}
}

func gcpStage4TranslateModel(parent, modelID string) *translatepb.Model {
	return &translatepb.Model{
		Name:               fmt.Sprintf("%s/models/%s", parent, modelID),
		DisplayName:        "Stackyard Model " + modelID,
		Dataset:            fmt.Sprintf("%s/datasets/dataset-1", parent),
		SourceLanguageCode: "en",
		TargetLanguageCode: "es",
		TrainExampleCount:  2,
		CreateTime:         timestamppb.New(gcpTranslateReferenceTime),
		UpdateTime:         timestamppb.New(gcpTranslateReferenceTime.Add(25 * time.Minute)),
	}
}

func gcpStage4TranslateSourceLanguage(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "en"
	}
	return value
}
