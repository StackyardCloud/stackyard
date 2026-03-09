package server

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const azureContentModeratorTextPrefix = "/azure/contentmoderator/moderate/v1.0/ProcessText/"

var azureContentModeratorPIIEmailRegexp = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

func (s *Server) handleAzureContentModeratorTextModerationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentModeratorTextPrefix) {
		return false
	}

	operation := strings.TrimSpace(strings.TrimPrefix(path, azureContentModeratorTextPrefix))
	operation = strings.TrimSuffix(operation, "/")
	if operation == "" || strings.Contains(operation, "/") {
		respondAzureImplemented(w, path)
		return true
	}
	if r.Method != http.MethodPost {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(operation) {
	case "detectlanguage":
		s.handleAzureContentModeratorTextDetectLanguage(w, r, path)
	case "screen":
		s.handleAzureContentModeratorTextScreen(w, r, path)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func (s *Server) handleAzureContentModeratorTextDetectLanguage(w http.ResponseWriter, r *http.Request, path string) {
	text, ok := parseAzureContentModeratorTextBody(w, r, path)
	if !ok {
		return
	}
	lang := azureContentModeratorDetectedLanguage(text)
	respondJSON(w, http.StatusOK, map[string]any{
		"DetectedLanguage": lang,
		"Status":           azureContentModeratorStatusOK(),
		"TrackingId":       azureContentModeratorTrackingID("text-detect-language", strings.ToLower(text)),
		"provider":         providerAzure,
		"path":             path,
	})
}

func (s *Server) handleAzureContentModeratorTextScreen(w http.ResponseWriter, r *http.Request, path string) {
	text, ok := parseAzureContentModeratorTextBody(w, r, path)
	if !ok {
		return
	}

	language := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("language")))
	if language == "" {
		language = azureContentModeratorDetectedLanguage(text)
	}
	autocorrect, ok := parseAzureContentModeratorBoolQuery(w, path, "autocorrect", r.URL.Query().Get("autocorrect"), false)
	if !ok {
		return
	}
	pii, ok := parseAzureContentModeratorBoolQuery(w, path, "PII", r.URL.Query().Get("PII"), false)
	if !ok {
		return
	}
	classify, ok := parseAzureContentModeratorBoolQuery(w, path, "classify", r.URL.Query().Get("classify"), false)
	if !ok {
		return
	}

	listID := 0
	listIDRaw := strings.TrimSpace(r.URL.Query().Get("listId"))
	if listIDRaw != "" {
		parsed, err := strconv.Atoi(listIDRaw)
		if err != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "listId must be a positive integer", "provider": providerAzure, "path": path})
			return
		}
		listID = parsed
	}

	normalized := azureContentModeratorNormalizeText(text, autocorrect)
	terms := azureContentModeratorExtractTerms(normalized, listID)

	resp := map[string]any{
		"OriginalText":   text,
		"NormalizedText": normalized,
		"Language":       language,
		"Terms":          terms,
		"Status":         azureContentModeratorStatusOK(),
		"TrackingId":     azureContentModeratorTrackingID("text-screen", strings.ToLower(normalized)),
		"provider":       providerAzure,
		"path":           path,
	}

	if pii {
		resp["PII"] = azureContentModeratorExtractPII(text)
	}
	if classify {
		resp["Classification"] = azureContentModeratorTextClassification(normalized)
	}

	respondJSON(w, http.StatusOK, resp)
}

func parseAzureContentModeratorTextBody(w http.ResponseWriter, r *http.Request, path string) (string, bool) {
	if !azureContentModeratorSupportedTextContentType(r.Header.Get("Content-Type")) {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unsupported content type for text moderation", "provider": providerAzure, "path": path})
		return "", false
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body", "provider": providerAzure, "path": path})
		return "", false
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "text body is required", "provider": providerAzure, "path": path})
		return "", false
	}
	return text, true
}

func azureContentModeratorSupportedTextContentType(raw string) bool {
	ct := strings.ToLower(strings.TrimSpace(raw))
	if ct == "" {
		return true
	}
	if strings.Contains(ct, ";") {
		ct = strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
	}
	supported := []string{"text/plain", "text/html", "text/xml", "text/markdown"}
	for _, candidate := range supported {
		if ct == candidate {
			return true
		}
	}
	return false
}

func azureContentModeratorDetectedLanguage(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "hola"), strings.Contains(lower, "gracias"), strings.Contains(lower, "español"):
		return "spa"
	case strings.Contains(lower, "bonjour"), strings.Contains(lower, "merci"), strings.Contains(lower, "français"):
		return "fra"
	case strings.Contains(lower, "hallo"), strings.Contains(lower, "danke"):
		return "deu"
	default:
		return "eng"
	}
}

func azureContentModeratorNormalizeText(text string, autocorrect bool) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if !autocorrect {
		return normalized
	}
	replacements := map[string]string{
		"teh":     "the",
		"recieve": "receive",
		"adress":  "address",
		"wierd":   "weird",
	}
	parts := strings.Split(normalized, " ")
	for i, part := range parts {
		stripped := strings.ToLower(strings.Trim(part, ",.!?;:"))
		if replacement, ok := replacements[stripped]; ok {
			parts[i] = replacement
		}
	}
	return strings.Join(parts, " ")
}

func azureContentModeratorExtractTerms(normalized string, listID int) []map[string]any {
	terms := []string{"hate", "violent", "idiot", "damn", "explicit"}
	lower := strings.ToLower(normalized)
	out := make([]map[string]any, 0, len(terms))
	for _, term := range terms {
		index := strings.Index(lower, term)
		if index < 0 {
			continue
		}
		entry := map[string]any{
			"Index":         index,
			"OriginalIndex": index,
			"Term":          term,
		}
		if listID > 0 {
			entry["ListId"] = listID
		}
		out = append(out, entry)
	}
	return out
}

func azureContentModeratorExtractPII(text string) map[string]any {
	emails := azureContentModeratorPIIEmailRegexp.FindAllString(text, -1)
	phone := ""
	compact := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(text)
	for i := 0; i < len(compact); i++ {
		for l := 10; l <= 12 && i+l <= len(compact); l++ {
			candidate := compact[i : i+l]
			if isDigits(candidate) {
				phone = candidate
				break
			}
		}
		if phone != "" {
			break
		}
	}
	phones := []string{}
	if phone != "" {
		phones = append(phones, phone)
	}
	return map[string]any{
		"Email": emails,
		"Phone": phones,
	}
}

func azureContentModeratorTextClassification(normalized string) map[string]any {
	lower := strings.ToLower(normalized)
	cat1 := 0.02
	cat2 := 0.03
	cat3 := 0.01
	if azureContentModeratorHasAny(lower, "damn", "idiot", "hate") {
		cat1 = 0.78
	}
	if azureContentModeratorHasAny(lower, "violent", "kill", "attack") {
		cat2 = 0.81
	}
	if azureContentModeratorHasAny(lower, "explicit", "nsfw") {
		cat3 = 0.76
	}
	review := cat1 >= 0.7 || cat2 >= 0.7 || cat3 >= 0.7
	return map[string]any{
		"Category1":         map[string]any{"Score": cat1},
		"Category2":         map[string]any{"Score": cat2},
		"Category3":         map[string]any{"Score": cat3},
		"ReviewRecommended": review,
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
