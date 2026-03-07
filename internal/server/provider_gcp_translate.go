package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const gcpTranslateGRPCPathPrefix = "/gcp/google.cloud.translation.v3.TranslationService/"

var (
	gcpTranslateReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpTranslateProjectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,62}$`)
	gcpTranslateLocationPattern  = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)
)

type gcpTranslateRouteContext struct {
	Parent string
	Name   string
	Query  url.Values
}

func (s *Server) handleGCPTranslateRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_translate(w, r) {
		return true
	}

	path := normalizeGCPTranslatePath(rawRequestPath(r))
	if isGCPTranslateLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPTranslateListLocations(w, r, path) {
			return true
		}
		if handleGCPTranslateGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpTranslateGRPCPathPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPTranslateJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpTranslateGRPCPathPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPTranslateRPCMethod(w, path, method, body, gcpTranslateRouteContext{Query: r.URL.Query()}) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if !isGCPTranslatePath(path, hasGCPTranslateHint(r)) {
		return false
	}

	method, ctx, needsBody, ok := mapGCPTranslateRESTToMethod(r, path)
	if !ok {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body := map[string]any{}
	if needsBody {
		var valid bool
		body, valid = decodeGCPTranslateJSONBody(w, r, path)
		if !valid {
			return true
		}
	}
	if handleGCPTranslateRPCMethod(w, path, method, body, ctx) {
		return true
	}
	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPTranslatePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTranslateHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "translate",
		"translate-apiv3",
		"translate_apiv3",
		"cloud-translate",
		"cloud_translate",
		"cloud-translate-v3",
		"cloud_translate_v3",
		"gcp-translate",
		"gcp-translate-v3":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-translate-apiv3") || strings.Contains(ua, "cloud.google.com/go/translate/apiv3")
}

func isGCPTranslateLocationRequest(r *http.Request, path string) bool {
	if !hasGCPTranslateHint(r) {
		return false
	}
	_, _, _, ok := parseGCPTranslateProjectLocationsPath(path)
	return ok
}

func isGCPTranslatePath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpTranslateGRPCPathPrefix) {
		return true
	}
	if _, _, _, ok := parseGCPTranslateProjectLocationsPath(path); ok {
		return includeHint
	}
	if _, _, tail, ok := parseGCPTranslateLocationTail(path); ok {
		if isGCPTranslateLocationTailRecognized(tail) {
			return true
		}
		return includeHint && strings.HasPrefix(path, "/gcp/v3/projects/")
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v3/projects/")
}

func isGCPTranslateLocationTailRecognized(tail string) bool {
	switch tail {
	case ":translateText",
		":romanizeText",
		":detectLanguage",
		"/supportedLanguages",
		":translateDocument",
		":batchTranslateText",
		":batchTranslateDocument",
		"/glossaries",
		"/datasets",
		"/adaptiveMtDatasets",
		"/models",
		"/operations":
		return true
	}
	return strings.HasPrefix(tail, "/glossaries/") ||
		strings.HasPrefix(tail, "/datasets/") ||
		strings.HasPrefix(tail, "/adaptiveMtDatasets/") ||
		strings.HasPrefix(tail, "/models/") ||
		strings.HasPrefix(tail, "/operations/")
}

func mapGCPTranslateRESTToMethod(r *http.Request, path string) (string, gcpTranslateRouteContext, bool, bool) {
	ctx := gcpTranslateRouteContext{Query: r.URL.Query()}

	project, location, list, ok := parseGCPTranslateProjectLocationsPath(path)
	if ok {
		if list && r.Method == http.MethodGet {
			ctx.Name = "projects/" + project
			return "ListLocations", ctx, false, true
		}
		if !list && r.Method == http.MethodGet {
			ctx.Name = fmt.Sprintf("projects/%s/locations/%s", project, location)
			return "GetLocation", ctx, false, true
		}
		return "", gcpTranslateRouteContext{}, false, false
	}

	project, location, tail, ok := parseGCPTranslateLocationTail(path)
	if !ok {
		return "", gcpTranslateRouteContext{}, false, false
	}
	ctx.Parent = fmt.Sprintf("projects/%s/locations/%s", project, location)

	switch {
	case r.Method == http.MethodPost && tail == ":translateText":
		return "TranslateText", ctx, true, true
	case r.Method == http.MethodPost && tail == ":romanizeText":
		return "RomanizeText", ctx, true, true
	case r.Method == http.MethodPost && tail == ":detectLanguage":
		return "DetectLanguage", ctx, true, true
	case r.Method == http.MethodGet && tail == "/supportedLanguages":
		return "GetSupportedLanguages", ctx, false, true
	case r.Method == http.MethodPost && tail == ":translateDocument":
		return "TranslateDocument", ctx, true, true
	case r.Method == http.MethodPost && tail == ":batchTranslateText":
		return "BatchTranslateText", ctx, true, true
	case r.Method == http.MethodPost && tail == ":batchTranslateDocument":
		return "BatchTranslateDocument", ctx, true, true
	}

	if tail == "/glossaries" {
		switch r.Method {
		case http.MethodGet:
			return "ListGlossaries", ctx, false, true
		case http.MethodPost:
			return "CreateGlossary", ctx, true, true
		}
	}

	if strings.HasPrefix(tail, "/glossaries/") {
		rest := strings.TrimPrefix(tail, "/glossaries/")
		if strings.Contains(rest, "/glossaryEntries") {
			glossaryID, entryTail, ok := gcpTranslateSplitResourceTail(rest, "glossaryEntries")
			if !ok {
				return "", gcpTranslateRouteContext{}, false, false
			}
			glossaryName := fmt.Sprintf("%s/glossaries/%s", ctx.Parent, glossaryID)
			switch {
			case entryTail == "" && r.Method == http.MethodGet:
				ctx.Parent = glossaryName
				return "ListGlossaryEntries", ctx, false, true
			case entryTail == "" && r.Method == http.MethodPost:
				ctx.Parent = glossaryName
				return "CreateGlossaryEntry", ctx, true, true
			case strings.HasPrefix(entryTail, "/") && r.Method == http.MethodGet:
				ctx.Name = glossaryName + "/glossaryEntries/" + strings.TrimPrefix(entryTail, "/")
				return "GetGlossaryEntry", ctx, false, true
			case strings.HasPrefix(entryTail, "/") && r.Method == http.MethodPatch:
				ctx.Name = glossaryName + "/glossaryEntries/" + strings.TrimPrefix(entryTail, "/")
				return "UpdateGlossaryEntry", ctx, true, true
			case strings.HasPrefix(entryTail, "/") && r.Method == http.MethodDelete:
				ctx.Name = glossaryName + "/glossaryEntries/" + strings.TrimPrefix(entryTail, "/")
				return "DeleteGlossaryEntry", ctx, false, true
			}
			return "", gcpTranslateRouteContext{}, false, false
		}

		glossaryName := ctx.Parent + "/glossaries/" + rest
		switch r.Method {
		case http.MethodGet:
			ctx.Name = glossaryName
			return "GetGlossary", ctx, false, true
		case http.MethodPatch:
			ctx.Name = glossaryName
			return "UpdateGlossary", ctx, true, true
		case http.MethodDelete:
			ctx.Name = glossaryName
			return "DeleteGlossary", ctx, false, true
		default:
			return "", gcpTranslateRouteContext{}, false, false
		}
	}

	if tail == "/datasets" {
		switch r.Method {
		case http.MethodGet:
			return "ListDatasets", ctx, false, true
		case http.MethodPost:
			return "CreateDataset", ctx, true, true
		}
	}
	if strings.HasPrefix(tail, "/datasets/") {
		rest := strings.TrimPrefix(tail, "/datasets/")
		if strings.HasSuffix(rest, ":importData") && r.Method == http.MethodPost {
			ctx.Name = ctx.Parent + "/datasets/" + strings.TrimSuffix(rest, ":importData")
			return "ImportData", ctx, true, true
		}
		if strings.HasSuffix(rest, ":exportData") && r.Method == http.MethodPost {
			ctx.Name = ctx.Parent + "/datasets/" + strings.TrimSuffix(rest, ":exportData")
			return "ExportData", ctx, true, true
		}
		if strings.HasSuffix(rest, "/examples") && r.Method == http.MethodGet {
			ctx.Parent = ctx.Parent + "/datasets/" + strings.TrimSuffix(rest, "/examples")
			return "ListExamples", ctx, false, true
		}
		if strings.Contains(rest, "/") {
			return "", gcpTranslateRouteContext{}, false, false
		}
		ctx.Name = ctx.Parent + "/datasets/" + rest
		switch r.Method {
		case http.MethodGet:
			return "GetDataset", ctx, false, true
		case http.MethodDelete:
			return "DeleteDataset", ctx, false, true
		}
	}

	if tail == "/adaptiveMtDatasets" {
		switch r.Method {
		case http.MethodGet:
			return "ListAdaptiveMtDatasets", ctx, false, true
		case http.MethodPost:
			return "CreateAdaptiveMtDataset", ctx, true, true
		}
	}
	if strings.HasPrefix(tail, "/adaptiveMtDatasets/") {
		rest := strings.TrimPrefix(tail, "/adaptiveMtDatasets/")
		if strings.Contains(rest, "/adaptiveMtFiles") {
			datasetID, fileTail, ok := gcpTranslateSplitResourceTail(rest, "adaptiveMtFiles")
			if !ok {
				return "", gcpTranslateRouteContext{}, false, false
			}
			datasetName := fmt.Sprintf("%s/adaptiveMtDatasets/%s", ctx.Parent, datasetID)
			switch {
			case fileTail == "" && r.Method == http.MethodGet:
				ctx.Parent = datasetName
				return "ListAdaptiveMtFiles", ctx, false, true
			case strings.HasPrefix(fileTail, "/") && r.Method == http.MethodGet:
				ctx.Name = datasetName + "/adaptiveMtFiles/" + strings.TrimPrefix(fileTail, "/")
				return "GetAdaptiveMtFile", ctx, false, true
			case strings.HasPrefix(fileTail, "/") && r.Method == http.MethodDelete:
				ctx.Name = datasetName + "/adaptiveMtFiles/" + strings.TrimPrefix(fileTail, "/")
				return "DeleteAdaptiveMtFile", ctx, false, true
			}
			return "", gcpTranslateRouteContext{}, false, false
		}
		if strings.HasSuffix(rest, ":adaptiveMtTranslate") && r.Method == http.MethodPost {
			ctx.Parent = ctx.Parent
			ctx.Name = ctx.Parent + "/adaptiveMtDatasets/" + strings.TrimSuffix(rest, ":adaptiveMtTranslate")
			return "AdaptiveMtTranslate", ctx, true, true
		}
		if strings.HasSuffix(rest, ":importAdaptiveMtFile") && r.Method == http.MethodPost {
			ctx.Parent = ctx.Parent + "/adaptiveMtDatasets/" + strings.TrimSuffix(rest, ":importAdaptiveMtFile")
			return "ImportAdaptiveMtFile", ctx, true, true
		}
		if strings.HasSuffix(rest, "/adaptiveMtSentences") && r.Method == http.MethodGet {
			ctx.Parent = ctx.Parent + "/adaptiveMtDatasets/" + strings.TrimSuffix(rest, "/adaptiveMtSentences")
			return "ListAdaptiveMtSentences", ctx, false, true
		}
		if strings.Contains(rest, "/") {
			return "", gcpTranslateRouteContext{}, false, false
		}
		ctx.Name = ctx.Parent + "/adaptiveMtDatasets/" + rest
		switch r.Method {
		case http.MethodGet:
			return "GetAdaptiveMtDataset", ctx, false, true
		case http.MethodDelete:
			return "DeleteAdaptiveMtDataset", ctx, false, true
		}
	}

	if tail == "/models" {
		switch r.Method {
		case http.MethodGet:
			return "ListModels", ctx, false, true
		case http.MethodPost:
			return "CreateModel", ctx, true, true
		}
	}
	if strings.HasPrefix(tail, "/models/") {
		rest := strings.TrimPrefix(tail, "/models/")
		if strings.Contains(rest, "/") {
			return "", gcpTranslateRouteContext{}, false, false
		}
		ctx.Name = ctx.Parent + "/models/" + rest
		switch r.Method {
		case http.MethodGet:
			return "GetModel", ctx, false, true
		case http.MethodDelete:
			return "DeleteModel", ctx, false, true
		}
	}

	if tail == "/operations" && r.Method == http.MethodGet {
		ctx.Name = ctx.Parent + "/operations"
		return "ListOperations", ctx, false, true
	}
	if strings.HasPrefix(tail, "/operations/") {
		rest := strings.TrimPrefix(tail, "/operations/")
		switch {
		case strings.HasSuffix(rest, ":wait") && r.Method == http.MethodPost:
			ctx.Name = ctx.Parent + "/operations/" + strings.TrimSuffix(rest, ":wait")
			return "WaitOperation", ctx, true, true
		case strings.HasSuffix(rest, ":cancel") && r.Method == http.MethodPost:
			ctx.Name = ctx.Parent + "/operations/" + strings.TrimSuffix(rest, ":cancel")
			return "CancelOperation", ctx, true, true
		case !strings.Contains(rest, ":") && r.Method == http.MethodGet:
			ctx.Name = ctx.Parent + "/operations/" + rest
			return "GetOperation", ctx, false, true
		case !strings.Contains(rest, ":") && r.Method == http.MethodDelete:
			ctx.Name = ctx.Parent + "/operations/" + rest
			return "DeleteOperation", ctx, false, true
		}
	}

	return "", gcpTranslateRouteContext{}, false, false
}

func handleGCPTranslateRPCMethod(w http.ResponseWriter, path, method string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	switch method {
	case "TranslateText":
		return handleGCPTranslateTranslateText(w, path, body, ctx)
	case "RomanizeText":
		return handleGCPTranslateRomanizeText(w, path, body, ctx)
	case "DetectLanguage":
		return handleGCPTranslateDetectLanguage(w, path, body, ctx)
	case "GetSupportedLanguages":
		return handleGCPTranslateGetSupportedLanguages(w, path, body, ctx)
	case "TranslateDocument":
		return handleGCPTranslateTranslateDocument(w, path, body, ctx)
	case "BatchTranslateText":
		return handleGCPTranslateBatchTranslateText(w, path, body, ctx)
	case "BatchTranslateDocument":
		return handleGCPTranslateBatchTranslateDocument(w, path, body, ctx)
	case "CreateGlossary":
		return handleGCPTranslateCreateGlossary(w, path, body, ctx)
	case "UpdateGlossary":
		return handleGCPTranslateUpdateGlossary(w, path, body, ctx)
	case "ListGlossaries":
		return handleGCPTranslateListGlossaries(w, path, body, ctx)
	case "GetGlossary":
		return handleGCPTranslateGetGlossary(w, path, body, ctx)
	case "DeleteGlossary":
		return handleGCPTranslateDeleteGlossary(w, path, body, ctx)
	case "GetGlossaryEntry":
		return handleGCPTranslateGetGlossaryEntry(w, path, body, ctx)
	case "ListGlossaryEntries":
		return handleGCPTranslateListGlossaryEntries(w, path, body, ctx)
	case "CreateGlossaryEntry":
		return handleGCPTranslateCreateGlossaryEntry(w, path, body, ctx)
	case "UpdateGlossaryEntry":
		return handleGCPTranslateUpdateGlossaryEntry(w, path, body, ctx)
	case "DeleteGlossaryEntry":
		return handleGCPTranslateDeleteGlossaryEntry(w, path, body, ctx)
	case "CreateDataset":
		return handleGCPTranslateCreateDataset(w, path, body, ctx)
	case "GetDataset":
		return handleGCPTranslateGetDataset(w, path, body, ctx)
	case "ListDatasets":
		return handleGCPTranslateListDatasets(w, path, body, ctx)
	case "DeleteDataset":
		return handleGCPTranslateDeleteDataset(w, path, body, ctx)
	case "CreateAdaptiveMtDataset":
		return handleGCPTranslateCreateAdaptiveMtDataset(w, path, body, ctx)
	case "DeleteAdaptiveMtDataset":
		return handleGCPTranslateDeleteAdaptiveMtDataset(w, path, body, ctx)
	case "GetAdaptiveMtDataset":
		return handleGCPTranslateGetAdaptiveMtDataset(w, path, body, ctx)
	case "ListAdaptiveMtDatasets":
		return handleGCPTranslateListAdaptiveMtDatasets(w, path, body, ctx)
	case "AdaptiveMtTranslate":
		return handleGCPTranslateAdaptiveMtTranslate(w, path, body, ctx)
	case "GetAdaptiveMtFile":
		return handleGCPTranslateGetAdaptiveMtFile(w, path, body, ctx)
	case "DeleteAdaptiveMtFile":
		return handleGCPTranslateDeleteAdaptiveMtFile(w, path, body, ctx)
	case "ImportAdaptiveMtFile":
		return handleGCPTranslateImportAdaptiveMtFile(w, path, body, ctx)
	case "ListAdaptiveMtFiles":
		return handleGCPTranslateListAdaptiveMtFiles(w, path, body, ctx)
	case "ListAdaptiveMtSentences":
		return handleGCPTranslateListAdaptiveMtSentences(w, path, body, ctx)
	case "ImportData":
		return handleGCPTranslateImportData(w, path, body, ctx)
	case "ExportData":
		return handleGCPTranslateExportData(w, path, body, ctx)
	case "ListExamples":
		return handleGCPTranslateListExamples(w, path, body, ctx)
	case "CreateModel":
		return handleGCPTranslateCreateModel(w, path, body, ctx)
	case "ListModels":
		return handleGCPTranslateListModels(w, path, body, ctx)
	case "GetModel":
		return handleGCPTranslateGetModel(w, path, body, ctx)
	case "DeleteModel":
		return handleGCPTranslateDeleteModel(w, path, body, ctx)
	case "ListLocations":
		return handleGCPTranslateListLocationsByMethod(w, path, body, ctx)
	case "GetLocation":
		return handleGCPTranslateGetLocationByMethod(w, path, body, ctx)
	case "GetOperation":
		return handleGCPTranslateGetOperation(w, path, body, ctx)
	case "ListOperations":
		return handleGCPTranslateListOperations(w, path, body, ctx)
	case "WaitOperation":
		return handleGCPTranslateWaitOperation(w, path, body, ctx)
	case "CancelOperation":
		return handleGCPTranslateCancelOperation(w, path, body, ctx)
	case "DeleteOperation":
		return handleGCPTranslateDeleteOperation(w, path, body, ctx)
	default:
		return false
	}
}

func handleGCPTranslateListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPTranslateProjectLocationsPath(path)
	if !ok || !list {
		return false
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, nil, r.URL.Query())
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateLocationFixture(project, "global"),
		gcpTranslateLocationFixture(project, "us-central1"),
	}
	return respondGCPTranslateList(w, "locations", items, pageSize, offset, path)
}

func handleGCPTranslateGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPTranslateProjectLocationsPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpTranslateLocationFixture(project, location))
	return true
}

func handleGCPTranslateListLocationsByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := strings.TrimSpace(gcpTranslateString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, ok := gcpTranslateProjectFromProjectName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateLocationFixture(project, "global"),
		gcpTranslateLocationFixture(project, "us-central1"),
	}
	return respondGCPTranslateList(w, "locations", items, pageSize, offset, path)
}

func handleGCPTranslateGetLocationByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := strings.TrimSpace(gcpTranslateString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, location, ok := gcpTranslateProjectLocationFromName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateLocationFixture(project, location))
	return true
}

func handleGCPTranslateTranslateText(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	project, location, ok := gcpTranslateRequireParent(w, path, parent)
	if !ok {
		return true
	}
	target := strings.TrimSpace(gcpTranslateString(body, "targetLanguageCode", "target_language_code"))
	if target == "" {
		respondGCPTranslateInvalidArgument(w, path, "target_language_code is required")
		return true
	}
	contents := gcpTranslateStringSlice(body, "contents")
	if len(contents) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "contents is required")
		return true
	}
	translations := make([]any, 0, len(contents))
	for _, content := range contents {
		if strings.TrimSpace(content) == "" {
			respondGCPTranslateInvalidArgument(w, path, "contents must not contain empty strings")
			return true
		}
		translations = append(translations, map[string]any{
			"translatedText":       fmt.Sprintf("[%s] %s", target, content),
			"detectedLanguageCode": gcpTranslateSourceLanguage(body),
			"model":                fmt.Sprintf("projects/%s/locations/%s/models/general/nmt", project, location),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"translations": translations,
	})
	return true
}

func handleGCPTranslateRomanizeText(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	contents := gcpTranslateStringSlice(body, "contents")
	if len(contents) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "contents is required")
		return true
	}
	romanizations := make([]any, 0, len(contents))
	for _, content := range contents {
		if strings.TrimSpace(content) == "" {
			respondGCPTranslateInvalidArgument(w, path, "contents must not contain empty strings")
			return true
		}
		romanizations = append(romanizations, map[string]any{
			"romanizedText":        gcpTranslateRomanizeString(content),
			"detectedLanguageCode": gcpTranslateSourceLanguage(body),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"romanizations": romanizations,
	})
	return true
}

func handleGCPTranslateDetectLanguage(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	content := strings.TrimSpace(gcpTranslateString(body, "content"))
	if content == "" {
		content = strings.TrimSpace(gcpTranslateString(body, "text"))
	}
	if content == "" {
		respondGCPTranslateInvalidArgument(w, path, "content is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"languages": []any{
			map[string]any{
				"languageCode": "en",
				"confidence":   0.99,
			},
		},
	})
	return true
}

func handleGCPTranslateGetSupportedLanguages(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	languages := []any{
		map[string]any{
			"languageCode":  "en",
			"displayName":   "English",
			"supportSource": true,
			"supportTarget": true,
		},
		map[string]any{
			"languageCode":  "es",
			"displayName":   "Spanish",
			"supportSource": true,
			"supportTarget": true,
		},
	}
	sort.SliceStable(languages, func(i, j int) bool {
		left, _ := languages[i].(map[string]any)
		right, _ := languages[j].(map[string]any)
		return gcpTranslateString(left, "languageCode") < gcpTranslateString(right, "languageCode")
	})
	respondJSON(w, http.StatusOK, map[string]any{
		"languages": languages,
	})
	return true
}

func handleGCPTranslateTranslateDocument(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	project, location, ok := gcpTranslateRequireParent(w, path, parent)
	if !ok {
		return true
	}
	target := strings.TrimSpace(gcpTranslateString(body, "targetLanguageCode", "target_language_code"))
	if target == "" {
		respondGCPTranslateInvalidArgument(w, path, "target_language_code is required")
		return true
	}
	if len(gcpTranslateBodyMap(body, "documentInputConfig", "document_input_config")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "document_input_config is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"documentTranslation": map[string]any{
			"byteStreamOutputs":    []any{"c3RhY2t5YXJkLXRyYW5zbGF0ZWQtZG9jdW1lbnQ="},
			"mimeType":             "text/plain",
			"detectedLanguageCode": gcpTranslateSourceLanguage(body),
		},
		"model": fmt.Sprintf("projects/%s/locations/%s/models/general/nmt", project, location),
		"glossaryConfig": map[string]any{
			"glossary": fmt.Sprintf("projects/%s/locations/%s/glossaries/glossary-1", project, location),
		},
	})
	return true
}

func handleGCPTranslateBatchTranslateText(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	if strings.TrimSpace(gcpTranslateString(body, "sourceLanguageCode", "source_language_code")) == "" {
		respondGCPTranslateInvalidArgument(w, path, "source_language_code is required")
		return true
	}
	if len(gcpTranslateStringSlice(body, "targetLanguageCodes", "target_language_codes")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "target_language_codes is required")
		return true
	}
	if len(gcpTranslateBodySlice(body, "inputConfigs", "input_configs")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "input_configs is required")
		return true
	}
	if len(gcpTranslateBodyMap(body, "outputConfig", "output_config")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "output_config is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "batchTranslateText.stackyard"))
	return true
}

func handleGCPTranslateBatchTranslateDocument(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	if strings.TrimSpace(gcpTranslateString(body, "sourceLanguageCode", "source_language_code")) == "" {
		respondGCPTranslateInvalidArgument(w, path, "source_language_code is required")
		return true
	}
	if len(gcpTranslateStringSlice(body, "targetLanguageCodes", "target_language_codes")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "target_language_codes is required")
		return true
	}
	if len(gcpTranslateBodySlice(body, "inputConfigs", "input_configs")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "input_configs is required")
		return true
	}
	if len(gcpTranslateBodyMap(body, "outputConfig", "output_config")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "output_config is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "batchTranslateDocument.stackyard"))
	return true
}

func handleGCPTranslateCreateGlossary(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	glossary := gcpTranslateBodyMap(body, "glossary")
	if len(glossary) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "glossary is required")
		return true
	}
	if got := strings.TrimSpace(gcpTranslateString(glossary, "name")); got != "" {
		parsedParent, _, ok := gcpTranslateParseGlossaryName(got)
		if !ok || parsedParent != parent {
			respondGCPTranslateInvalidArgument(w, path, "glossary.name must match parent")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "createGlossary.glossary-1"))
	return true
}

func handleGCPTranslateUpdateGlossary(w http.ResponseWriter, path string, body map[string]any, _ gcpTranslateRouteContext) bool {
	glossary := gcpTranslateBodyMap(body, "glossary")
	if len(glossary) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "glossary is required")
		return true
	}
	name := strings.TrimSpace(gcpTranslateString(glossary, "name"))
	parent, _, ok := gcpTranslateParseGlossaryName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "glossary.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "updateGlossary.glossary-1"))
	return true
}

func handleGCPTranslateListGlossaries(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateGlossaryFixture(parent, "glossary-1"),
		gcpTranslateGlossaryFixture(parent, "glossary-2"),
	}
	return respondGCPTranslateList(w, "glossaries", items, pageSize, offset, path)
}

func handleGCPTranslateGetGlossary(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, glossaryID, ok := gcpTranslateParseGlossaryName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(glossaryID, "missing") {
		respondGCPTranslateNotFound(w, path, "glossary not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateGlossaryFixture(parent, glossaryID))
	return true
}

func handleGCPTranslateDeleteGlossary(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, _, ok := gcpTranslateParseGlossaryName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "deleteGlossary.glossary-1"))
	return true
}

func handleGCPTranslateGetGlossaryEntry(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, glossaryID, entryID, ok := gcpTranslateParseGlossaryEntryName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(entryID, "missing") {
		respondGCPTranslateNotFound(w, path, "glossary entry not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateGlossaryEntryFixture(parent, glossaryID, entryID))
	return true
}

func handleGCPTranslateListGlossaryEntries(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	glossaryParent, glossaryID, ok := gcpTranslateParseGlossaryName(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateGlossaryEntryFixture(glossaryParent, glossaryID, "entry-1"),
		gcpTranslateGlossaryEntryFixture(glossaryParent, glossaryID, "entry-2"),
	}
	return respondGCPTranslateList(w, "glossaryEntries", items, pageSize, offset, path)
}

func handleGCPTranslateCreateGlossaryEntry(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	glossaryParent, glossaryID, ok := gcpTranslateParseGlossaryName(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return true
	}
	entry := gcpTranslateBodyMap(body, "glossaryEntry", "glossary_entry")
	if len(entry) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "glossary_entry is required")
		return true
	}
	entryID := "entry-1"
	if name := strings.TrimSpace(gcpTranslateString(entry, "name")); name != "" {
		_, _, parsedEntryID, ok := gcpTranslateParseGlossaryEntryName(name)
		if !ok {
			respondGCPTranslateInvalidArgument(w, path, "glossary_entry.name is invalid")
			return true
		}
		entryID = parsedEntryID
	}
	respondJSON(w, http.StatusOK, gcpTranslateGlossaryEntryFixture(glossaryParent, glossaryID, entryID))
	return true
}

func handleGCPTranslateUpdateGlossaryEntry(w http.ResponseWriter, path string, body map[string]any, _ gcpTranslateRouteContext) bool {
	entry := gcpTranslateBodyMap(body, "glossaryEntry", "glossary_entry")
	if len(entry) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "glossary_entry is required")
		return true
	}
	name := strings.TrimSpace(gcpTranslateString(entry, "name"))
	parent, glossaryID, entryID, ok := gcpTranslateParseGlossaryEntryName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "glossary_entry.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateGlossaryEntryFixture(parent, glossaryID, entryID))
	return true
}

func handleGCPTranslateDeleteGlossaryEntry(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	if _, _, _, ok := gcpTranslateParseGlossaryEntryName(name); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTranslateCreateDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	if len(gcpTranslateBodyMap(body, "dataset")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "dataset is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "createDataset.dataset-1"))
	return true
}

func handleGCPTranslateGetDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, datasetID, ok := gcpTranslateParseDatasetName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(datasetID, "missing") {
		respondGCPTranslateNotFound(w, path, "dataset not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateDatasetFixture(parent, datasetID))
	return true
}

func handleGCPTranslateListDatasets(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateDatasetFixture(parent, "dataset-1"),
		gcpTranslateDatasetFixture(parent, "dataset-2"),
	}
	return respondGCPTranslateList(w, "datasets", items, pageSize, offset, path)
}

func handleGCPTranslateDeleteDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, _, ok := gcpTranslateParseDatasetName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "deleteDataset.dataset-1"))
	return true
}

func handleGCPTranslateCreateAdaptiveMtDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	item := gcpTranslateBodyMap(body, "adaptiveMtDataset", "adaptive_mt_dataset")
	if len(item) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "adaptive_mt_dataset is required")
		return true
	}
	id := "adaptive-dataset-1"
	if name := strings.TrimSpace(gcpTranslateString(item, "name")); name != "" {
		_, parsedID, ok := gcpTranslateParseAdaptiveDatasetName(name)
		if !ok {
			respondGCPTranslateInvalidArgument(w, path, "adaptive_mt_dataset.name is invalid")
			return true
		}
		id = parsedID
	}
	respondJSON(w, http.StatusOK, gcpTranslateAdaptiveDatasetFixture(parent, id))
	return true
}

func handleGCPTranslateDeleteAdaptiveMtDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(name); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTranslateGetAdaptiveMtDataset(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(datasetID, "missing") {
		respondGCPTranslateNotFound(w, path, "adaptive dataset not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateAdaptiveDatasetFixture(parent, datasetID))
	return true
}

func handleGCPTranslateListAdaptiveMtDatasets(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateAdaptiveDatasetFixture(parent, "adaptive-dataset-1"),
		gcpTranslateAdaptiveDatasetFixture(parent, "adaptive-dataset-2"),
	}
	return respondGCPTranslateList(w, "adaptiveMtDatasets", items, pageSize, offset, path)
}

func handleGCPTranslateAdaptiveMtTranslate(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	dataset := strings.TrimSpace(gcpTranslateString(body, "dataset"))
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(dataset); !ok {
		respondGCPTranslateInvalidArgument(w, path, "dataset is required")
		return true
	}
	content := gcpTranslateStringSlice(body, "content", "contents")
	if len(content) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "content is required")
		return true
	}
	items := make([]any, 0, len(content))
	for _, item := range content {
		items = append(items, map[string]any{"translatedText": "[adaptive] " + item})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"languageCode": "es",
		"translations": items,
	})
	return true
}

func handleGCPTranslateGetAdaptiveMtFile(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, datasetID, fileID, ok := gcpTranslateParseAdaptiveFileName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(fileID, "missing") {
		respondGCPTranslateNotFound(w, path, "adaptive file not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateAdaptiveFileFixture(parent, datasetID, fileID))
	return true
}

func handleGCPTranslateDeleteAdaptiveMtFile(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	if _, _, _, ok := gcpTranslateParseAdaptiveFileName(name); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTranslateImportAdaptiveMtFile(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	datasetParent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return true
	}
	fileSource := gcpTranslateBodyMap(body, "fileInputSource", "file_input_source")
	gcsSource := gcpTranslateBodyMap(body, "gcsInputSource", "gcs_input_source")
	if len(fileSource) == 0 && len(gcsSource) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "one source is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"adaptiveMtFile": gcpTranslateAdaptiveFileFixture(datasetParent, datasetID, "file-1"),
	})
	return true
}

func handleGCPTranslateListAdaptiveMtFiles(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	datasetParent, datasetID, ok := gcpTranslateParseAdaptiveDatasetName(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateAdaptiveFileFixture(datasetParent, datasetID, "file-1"),
		gcpTranslateAdaptiveFileFixture(datasetParent, datasetID, "file-2"),
	}
	return respondGCPTranslateList(w, "adaptiveMtFiles", items, pageSize, offset, path)
}

func handleGCPTranslateListAdaptiveMtSentences(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateParseAdaptiveDatasetName(parent); !ok {
		if _, _, _, ok := gcpTranslateParseAdaptiveFileName(parent); !ok {
			respondGCPTranslateInvalidArgument(w, path, "parent is required")
			return true
		}
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateAdaptiveSentenceFixture(parent, "sentence-1"),
		gcpTranslateAdaptiveSentenceFixture(parent, "sentence-2"),
	}
	return respondGCPTranslateList(w, "adaptiveMtSentences", items, pageSize, offset, path)
}

func handleGCPTranslateImportData(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	dataset := strings.TrimSpace(gcpTranslateString(body, "dataset"))
	if dataset == "" {
		dataset = strings.TrimSpace(ctx.Name)
	}
	parent, _, ok := gcpTranslateParseDatasetName(dataset)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "dataset is required")
		return true
	}
	if len(gcpTranslateBodyMap(body, "inputConfig", "input_config")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "input_config is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "importData.dataset-1"))
	return true
}

func handleGCPTranslateExportData(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	dataset := strings.TrimSpace(gcpTranslateString(body, "dataset"))
	if dataset == "" {
		dataset = strings.TrimSpace(ctx.Name)
	}
	parent, _, ok := gcpTranslateParseDatasetName(dataset)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "dataset is required")
		return true
	}
	if len(gcpTranslateBodyMap(body, "outputConfig", "output_config")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "output_config is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "exportData.dataset-1"))
	return true
}

func handleGCPTranslateListExamples(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	_, datasetID, ok := gcpTranslateParseDatasetName(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateExampleFixture(parent, "example-1", datasetID),
		gcpTranslateExampleFixture(parent, "example-2", datasetID),
	}
	return respondGCPTranslateList(w, "examples", items, pageSize, offset, path)
}

func handleGCPTranslateCreateModel(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	if len(gcpTranslateBodyMap(body, "model")) == 0 {
		respondGCPTranslateInvalidArgument(w, path, "model is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "createModel.model-1"))
	return true
}

func handleGCPTranslateListModels(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	parent := gcpTranslateResolveParent(body, ctx)
	if _, _, ok := gcpTranslateRequireParent(w, path, parent); !ok {
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateModelFixture(parent, "model-1"),
		gcpTranslateModelFixture(parent, "model-2"),
	}
	return respondGCPTranslateList(w, "models", items, pageSize, offset, path)
}

func handleGCPTranslateGetModel(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, modelID, ok := gcpTranslateParseModelName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(modelID, "missing") {
		respondGCPTranslateNotFound(w, path, "model not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateModelFixture(parent, modelID))
	return true
}

func handleGCPTranslateDeleteModel(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, _, ok := gcpTranslateParseModelName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, "deleteModel.model-1"))
	return true
}

func handleGCPTranslateGetOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, opID, ok := gcpTranslateParseOperationName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(opID, "missing") {
		respondGCPTranslateNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, opID))
	return true
}

func handleGCPTranslateListOperations(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := strings.TrimSpace(gcpTranslateString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	parent := strings.TrimSpace(name)
	if strings.HasSuffix(parent, "/operations") {
		parent = strings.TrimSuffix(parent, "/operations")
	}
	if _, _, ok := gcpTranslateProjectLocationFromParent(parent); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPTranslatePagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTranslateOperationFixture(parent, "batchTranslateText.stackyard"),
		gcpTranslateOperationFixture(parent, "createGlossary.glossary-1"),
	}
	return respondGCPTranslateList(w, "operations", items, pageSize, offset, path)
}

func handleGCPTranslateWaitOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	parent, opID, ok := gcpTranslateParseOperationName(name)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTranslateOperationFixture(parent, opID))
	return true
}

func handleGCPTranslateCancelOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	if _, _, ok := gcpTranslateParseOperationName(name); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTranslateDeleteOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpTranslateRouteContext) bool {
	name := gcpTranslateResolveName(body, ctx)
	if _, _, ok := gcpTranslateParseOperationName(name); !ok {
		respondGCPTranslateInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func decodeGCPTranslateJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.UseNumber()

	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPTranslateInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPTranslateProjectLocationsPath(path string) (project, location string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", false, false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if !gcpTranslateProjectIDPattern.MatchString(project) {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if !gcpTranslateLocationPattern.MatchString(location) {
		return "", "", false, false
	}
	return project, location, false, true
}

func parseGCPTranslateLocationTail(path string) (project, location, tail string, ok bool) {
	const prefix = "/gcp/v3/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}

	rest := strings.TrimPrefix(path, prefix)
	projectPart, afterProject, found := strings.Cut(rest, "/locations/")
	if !found {
		return "", "", "", false
	}
	project = strings.TrimSpace(projectPart)
	if !gcpTranslateProjectIDPattern.MatchString(project) {
		return "", "", "", false
	}

	afterProject = strings.TrimSpace(afterProject)
	if afterProject == "" {
		return "", "", "", false
	}

	split := strings.IndexAny(afterProject, "/:")
	if split < 0 {
		location = strings.TrimSpace(afterProject)
		tail = ""
	} else {
		location = strings.TrimSpace(afterProject[:split])
		tail = strings.TrimSpace(afterProject[split:])
	}
	if !gcpTranslateLocationPattern.MatchString(location) {
		return "", "", "", false
	}
	if tail == "/" || tail == ":" {
		return "", "", "", false
	}
	if tail != "" && !strings.HasPrefix(tail, "/") && !strings.HasPrefix(tail, ":") {
		return "", "", "", false
	}
	if strings.HasPrefix(tail, ":") && strings.Contains(tail, "/") {
		return "", "", "", false
	}
	return project, location, tail, true
}

func parseGCPTranslatePagination(w http.ResponseWriter, path string, body map[string]any, query url.Values) (int, int, bool) {
	pageSize := 50
	offset := 0

	rawPageSize := ""
	if body != nil {
		rawPageSize = strings.TrimSpace(gcpTranslateString(body, "pageSize", "page_size"))
	}
	if rawPageSize == "" && query != nil {
		rawPageSize = strings.TrimSpace(query.Get("pageSize"))
		if rawPageSize == "" {
			rawPageSize = strings.TrimSpace(query.Get("page_size"))
		}
	}
	if rawPageSize != "" {
		value, err := strconv.Atoi(rawPageSize)
		if err != nil || value < 0 || value > 1000 {
			respondGCPTranslateInvalidArgument(w, path, "pageSize must be between 0 and 1000")
			return 0, 0, false
		}
		pageSize = value
	}

	rawPageToken := ""
	if body != nil {
		rawPageToken = strings.TrimSpace(gcpTranslateString(body, "pageToken", "page_token"))
	}
	if rawPageToken == "" && query != nil {
		rawPageToken = strings.TrimSpace(query.Get("pageToken"))
		if rawPageToken == "" {
			rawPageToken = strings.TrimSpace(query.Get("page_token"))
		}
	}
	if rawPageToken != "" {
		value, err := strconv.Atoi(rawPageToken)
		if err != nil || value < 0 {
			respondGCPTranslateInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		offset = value
	}
	return pageSize, offset, true
}

func respondGCPTranslateList(w http.ResponseWriter, key string, items []map[string]any, pageSize, offset int, path string) bool {
	if offset > len(items) {
		respondGCPTranslateInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && offset+pageSize < end {
		end = offset + pageSize
	}
	response := map[string]any{
		key: items[offset:end],
	}
	if end < len(items) {
		response["nextPageToken"] = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func gcpTranslateResolveParent(body map[string]any, ctx gcpTranslateRouteContext) string {
	parent := strings.TrimSpace(gcpTranslateString(body, "parent"))
	if parent != "" {
		return parent
	}
	return strings.TrimSpace(ctx.Parent)
}

func gcpTranslateResolveName(body map[string]any, ctx gcpTranslateRouteContext) string {
	name := strings.TrimSpace(gcpTranslateString(body, "name"))
	if name != "" {
		return name
	}
	return strings.TrimSpace(ctx.Name)
}

func gcpTranslateRequireParent(w http.ResponseWriter, path, parent string) (string, string, bool) {
	project, location, ok := gcpTranslateProjectLocationFromParent(parent)
	if !ok {
		respondGCPTranslateInvalidArgument(w, path, "parent is required")
		return "", "", false
	}
	return project, location, true
}

func gcpTranslateProjectLocationFromParent(parent string) (string, string, bool) {
	value := strings.TrimSpace(parent)
	parts := strings.Split(value, "/")
	if len(parts) == 2 && parts[0] == "projects" && gcpTranslateProjectIDPattern.MatchString(parts[1]) {
		return parts[1], "global", true
	}
	if len(parts) == 4 &&
		parts[0] == "projects" &&
		parts[2] == "locations" &&
		gcpTranslateProjectIDPattern.MatchString(parts[1]) &&
		gcpTranslateLocationPattern.MatchString(parts[3]) {
		return parts[1], parts[3], true
	}
	return "", "", false
}

func gcpTranslateProjectFromProjectName(name string) (string, bool) {
	trimmed := strings.TrimSpace(name)
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] != "projects" || !gcpTranslateProjectIDPattern.MatchString(parts[1]) {
		return "", false
	}
	return parts[1], true
}

func gcpTranslateProjectLocationFromName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) {
		return "", "", false
	}
	return parts[1], parts[3], true
}

func gcpTranslateParseGlossaryName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "glossaries" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) || strings.TrimSpace(parts[5]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpTranslateParseGlossaryEntryName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "glossaries" || parts[6] != "glossaryEntries" {
		return "", "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) {
		return "", "", "", false
	}
	if strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], parts[7], true
}

func gcpTranslateParseDatasetName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "datasets" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) || strings.TrimSpace(parts[5]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpTranslateParseAdaptiveDatasetName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "adaptiveMtDatasets" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) || strings.TrimSpace(parts[5]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpTranslateParseAdaptiveFileName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "adaptiveMtDatasets" || parts[6] != "adaptiveMtFiles" {
		return "", "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) {
		return "", "", "", false
	}
	if strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], parts[7], true
}

func gcpTranslateParseModelName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "models" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) || strings.TrimSpace(parts[5]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpTranslateParseOperationName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", false
	}
	if !gcpTranslateProjectIDPattern.MatchString(parts[1]) || !gcpTranslateLocationPattern.MatchString(parts[3]) || strings.TrimSpace(parts[5]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpTranslateSplitResourceTail(rest, child string) (string, string, bool) {
	token := "/" + child
	idx := strings.Index(rest, token)
	if idx <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(rest[:idx]), strings.TrimSpace(rest[idx+len(token):]), true
}

func gcpTranslateLocationFixture(project, location string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s", project, location)
	return map[string]any{
		"name":        name,
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
		"metadata": map[string]any{
			"supportsBatchTranslation": true,
		},
	}
}

func gcpTranslateGlossaryFixture(parent, glossaryID string) map[string]any {
	name := fmt.Sprintf("%s/glossaries/%s", parent, glossaryID)
	return map[string]any{
		"name":        name,
		"displayName": "Stackyard Glossary " + glossaryID,
		"languagePair": map[string]any{
			"sourceLanguageCode": "en",
			"targetLanguageCode": "es",
		},
		"inputConfig": map[string]any{
			"gcsSource": map[string]any{
				"inputUri": "gs://stackyard/translate/glossary.csv",
			},
		},
		"entryCount": 2,
		"submitTime": gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"endTime":    gcpTranslateReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpTranslateGlossaryEntryFixture(parent, glossaryID, entryID string) map[string]any {
	name := fmt.Sprintf("%s/glossaries/%s/glossaryEntries/%s", parent, glossaryID, entryID)
	return map[string]any{
		"name":        name,
		"description": "Stackyard glossary entry " + entryID,
		"termsPair": map[string]any{
			"sourceTerm": "hello",
			"targetTerm": "hola",
		},
	}
}

func gcpTranslateDatasetFixture(parent, datasetID string) map[string]any {
	name := fmt.Sprintf("%s/datasets/%s", parent, datasetID)
	return map[string]any{
		"name":               name,
		"displayName":        "Stackyard Dataset " + datasetID,
		"sourceLanguageCode": "en",
		"targetLanguageCode": "es",
		"exampleCount":       2,
		"createTime":         gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"updateTime":         gcpTranslateReferenceTime.Add(5 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpTranslateAdaptiveDatasetFixture(parent, datasetID string) map[string]any {
	name := fmt.Sprintf("%s/adaptiveMtDatasets/%s", parent, datasetID)
	return map[string]any{
		"name":               name,
		"displayName":        "Stackyard Adaptive Dataset " + datasetID,
		"sourceLanguageCode": "en",
		"targetLanguageCode": "es",
		"exampleCount":       2,
		"createTime":         gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"updateTime":         gcpTranslateReferenceTime.Add(10 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpTranslateAdaptiveFileFixture(parent, datasetID, fileID string) map[string]any {
	name := fmt.Sprintf("%s/adaptiveMtDatasets/%s/adaptiveMtFiles/%s", parent, datasetID, fileID)
	return map[string]any{
		"name":        name,
		"displayName": "Stackyard Adaptive File " + fileID,
		"entryCount":  2,
		"createTime":  gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"updateTime":  gcpTranslateReferenceTime.Add(15 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpTranslateAdaptiveSentenceFixture(parent, sentenceID string) map[string]any {
	name := fmt.Sprintf("%s/adaptiveMtSentences/%s", parent, sentenceID)
	return map[string]any{
		"name":           name,
		"sourceSentence": "hello",
		"targetSentence": "hola",
		"createTime":     gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"updateTime":     gcpTranslateReferenceTime.Add(20 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpTranslateExampleFixture(parent, exampleID, datasetID string) map[string]any {
	name := fmt.Sprintf("%s/examples/%s", parent, exampleID)
	return map[string]any{
		"name":       name,
		"sourceText": "source sentence for " + datasetID,
		"targetText": "translated sentence for " + datasetID,
		"usage":      "TRAIN",
	}
}

func gcpTranslateModelFixture(parent, modelID string) map[string]any {
	name := fmt.Sprintf("%s/models/%s", parent, modelID)
	return map[string]any{
		"name":               name,
		"displayName":        "Stackyard Model " + modelID,
		"dataset":            fmt.Sprintf("%s/datasets/dataset-1", parent),
		"sourceLanguageCode": "en",
		"targetLanguageCode": "es",
		"trainExampleCount":  2,
		"createTime":         gcpTranslateReferenceTime.Format(time.RFC3339Nano),
		"updateTime":         gcpTranslateReferenceTime.Add(25 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpTranslateOperationFixture(parent, operationID string) map[string]any {
	name := fmt.Sprintf("%s/operations/%s", parent, strings.TrimSpace(operationID))
	return map[string]any{
		"name": name,
		"done": true,
		"metadata": map[string]any{
			"@type":     "type.googleapis.com/google.protobuf.Struct",
			"operation": operationID,
		},
		"response": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
}

func gcpTranslateRomanizeString(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"á", "a",
		"à", "a",
		"ä", "a",
		"é", "e",
		"è", "e",
		"ë", "e",
		"í", "i",
		"ì", "i",
		"ï", "i",
		"ó", "o",
		"ò", "o",
		"ö", "o",
		"ú", "u",
		"ù", "u",
		"ü", "u",
	)
	return replacer.Replace(s)
}

func gcpTranslateSourceLanguage(body map[string]any) string {
	source := strings.TrimSpace(gcpTranslateString(body, "sourceLanguageCode", "source_language_code"))
	if source == "" {
		return "en"
	}
	return source
}

func gcpTranslateString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func gcpTranslateBodyMap(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := body[key]; ok {
			if typed, ok := value.(map[string]any); ok {
				return typed
			}
		}
	}
	return map[string]any{}
}

func gcpTranslateBodySlice(body map[string]any, keys ...string) []any {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := body[key]; ok {
			if typed, ok := value.([]any); ok {
				return typed
			}
		}
	}
	return nil
}

func gcpTranslateStringSlice(body map[string]any, keys ...string) []string {
	items := gcpTranslateBodySlice(body, keys...)
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			out = append(out, typed)
		case json.Number:
			out = append(out, typed.String())
		}
	}
	return out
}

func respondGCPTranslateInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTranslateNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTranslateFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_translate(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "translate") && !isGCPTranslateContractProbeRequest(r, path) {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPTranslateInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/translate",
			"service":  "translate",
			"provider": providerGCP,
			"path":     path,
			"methods": []string{
				"TranslateText",
				"DetectLanguage",
				"GetSupportedLanguages",
				"BatchTranslateText",
				"CreateGlossary",
				"CreateDataset",
				"CreateAdaptiveMtDataset",
				"CreateModel",
			},
		})
		return true
	}
	return false
}

func isGCPTranslateContractProbeRequest(r *http.Request, path string) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return false
	}
	if parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "projects" || parts[4] != "locations" {
		return false
	}
	if !gcpTranslateProjectIDPattern.MatchString(strings.TrimSpace(parts[3])) {
		return false
	}
	if !gcpTranslateLocationPattern.MatchString(strings.TrimSpace(parts[5])) {
		return false
	}
	return strings.TrimSpace(parts[6]) == "translate"
}
