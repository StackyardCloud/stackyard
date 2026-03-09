package server

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const azureContentModeratorImagePrefix = "/azure/contentmoderator/moderate/v1.0/ProcessImage/"

type azureContentModeratorImageInput struct {
	Overload   string
	Source     string
	SizeBytes  int
	CacheImage bool
}

func (s *Server) handleAzureContentModeratorImageModerationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentModeratorImagePrefix) {
		return false
	}

	operation := strings.TrimSpace(strings.TrimPrefix(path, azureContentModeratorImagePrefix))
	if operation == "" || strings.Contains(operation, "/") {
		respondAzureImplemented(w, path)
		return true
	}
	if r.Method != http.MethodPost {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(operation) {
	case "evaluate":
		s.handleAzureContentModeratorImageEvaluate(w, r, path)
	case "findfaces":
		s.handleAzureContentModeratorImageFindFaces(w, r, path)
	case "match":
		s.handleAzureContentModeratorImageMatch(w, r, path)
	case "ocr":
		s.handleAzureContentModeratorImageOCR(w, r, path)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func (s *Server) handleAzureContentModeratorImageEvaluate(w http.ResponseWriter, r *http.Request, path string) {
	input, ok := parseAzureContentModeratorImageInput(w, r, path)
	if !ok {
		return
	}

	adultScore, racyScore := azureContentModeratorClassificationScores(input.Source)
	result := adultScore >= 0.75 || racyScore >= 0.65
	respondJSON(w, http.StatusOK, map[string]any{
		"AdultClassificationScore": adultScore,
		"IsImageAdultClassified":   adultScore >= 0.75,
		"RacyClassificationScore":  racyScore,
		"IsImageRacyClassified":    racyScore >= 0.65,
		"AdvancedInfo":             azureContentModeratorAdvancedInfo(input),
		"Result":                   result,
		"Status":                   azureContentModeratorStatusOK(),
		"TrackingId":               azureContentModeratorTrackingID("evaluate", input.Source),
		"CacheID":                  azureContentModeratorCacheID("evaluate", input.Source),
		"provider":                 providerAzure,
		"path":                     path,
	})
}

func (s *Server) handleAzureContentModeratorImageFindFaces(w http.ResponseWriter, r *http.Request, path string) {
	input, ok := parseAzureContentModeratorImageInput(w, r, path)
	if !ok {
		return
	}

	faceCount := azureContentModeratorFaceCount(input.Source)
	faces := make([]map[string]any, 0, faceCount)
	seed := azureContentModeratorSeed("faces", input.Source)
	for i := 0; i < faceCount; i++ {
		left := 24 + int((seed+uint64(i*29))%110)
		top := 18 + int((seed+uint64(i*31))%95)
		width := 24 + int((seed+uint64(i*17))%38)
		height := 24 + int((seed+uint64(i*23))%42)
		faces = append(faces, map[string]any{
			"Left":   left,
			"Right":  left + width,
			"Top":    top,
			"Bottom": top + height,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"Faces":        faces,
		"Count":        faceCount,
		"AdvancedInfo": azureContentModeratorAdvancedInfo(input),
		"Result":       faceCount > 0,
		"Status":       azureContentModeratorStatusOK(),
		"TrackingId":   azureContentModeratorTrackingID("findfaces", input.Source),
		"CacheID":      azureContentModeratorCacheID("findfaces", input.Source),
		"provider":     providerAzure,
		"path":         path,
	})
}

func (s *Server) handleAzureContentModeratorImageMatch(w http.ResponseWriter, r *http.Request, path string) {
	listIDRaw := strings.TrimSpace(r.URL.Query().Get("listId"))
	if listIDRaw == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "listId query parameter is required", "provider": providerAzure, "path": path})
		return
	}
	listID, err := strconv.Atoi(listIDRaw)
	if err != nil || listID <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "listId must be a positive integer", "provider": providerAzure, "path": path})
		return
	}

	input, ok := parseAzureContentModeratorImageInput(w, r, path)
	if !ok {
		return
	}

	isMatch := !azureContentModeratorHasAny(input.Source, "nomatch", "no-match", "unknown")
	matches := make([]map[string]any, 0, 2)
	if isMatch {
		seed := azureContentModeratorSeed("match", input.Source)
		matchCount := 1
		if azureContentModeratorHasAny(input.Source, "multi", "crowd", "group") {
			matchCount = 2
		}
		for i := 0; i < matchCount; i++ {
			score := 0.99 - float64(i)*0.03
			matchID := listID*1000 + int((seed+uint64(i*13))%997)
			matches = append(matches, map[string]any{
				"Score":   score,
				"MatchId": matchID,
				"Source":  strconv.Itoa(listID),
				"Tags":    []int{101 + i},
				"Label":   fmt.Sprintf("Image-%d", i+1),
			})
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"IsMatch":    isMatch,
		"Matches":    matches,
		"Status":     azureContentModeratorStatusOK(),
		"TrackingId": azureContentModeratorTrackingID("match", input.Source),
		"CacheID":    azureContentModeratorCacheID("match", input.Source),
		"provider":   providerAzure,
		"path":       path,
	})
}

func (s *Server) handleAzureContentModeratorImageOCR(w http.ResponseWriter, r *http.Request, path string) {
	language := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("language")))
	if language == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "language query parameter is required", "provider": providerAzure, "path": path})
		return
	}

	enhanced, ok := parseAzureContentModeratorBoolQuery(w, path, "enhanced", r.URL.Query().Get("enhanced"), false)
	if !ok {
		return
	}
	input, ok := parseAzureContentModeratorImageInput(w, r, path)
	if !ok {
		return
	}
	if enhanced && input.Overload == "stream" && strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "image/tiff") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "enhanced OCR is not supported for tiff content", "provider": providerAzure, "path": path})
		return
	}

	text := azureContentModeratorOCRText(input.Source)
	candidates := []map[string]any{}
	if enhanced {
		candidates = []map[string]any{
			{"Text": "STACKYARD", "Confidence": 0.99},
			{"Text": "IMAGE", "Confidence": 0.95},
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"Metadata":   azureContentModeratorAdvancedInfo(input),
		"Language":   language,
		"Text":       text,
		"Candidates": candidates,
		"Status":     azureContentModeratorStatusOK(),
		"TrackingId": azureContentModeratorTrackingID("ocr", input.Source),
		"CacheID":    azureContentModeratorCacheID("ocr", input.Source),
		"provider":   providerAzure,
		"path":       path,
	})
}

func parseAzureContentModeratorImageInput(w http.ResponseWriter, r *http.Request, path string) (azureContentModeratorImageInput, bool) {
	overload := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("overload")))
	if overload == "" {
		overload = "stream"
	}
	if overload != "stream" && overload != "url" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "overload must be stream or url", "provider": providerAzure, "path": path})
		return azureContentModeratorImageInput{}, false
	}
	cacheImage, ok := parseAzureContentModeratorBoolQuery(w, path, "CacheImage", r.URL.Query().Get("CacheImage"), false)
	if !ok {
		return azureContentModeratorImageInput{}, false
	}

	if overload == "url" {
		body, ok := decodeAzureContentModeratorURLPayload(w, r, path)
		if !ok {
			return azureContentModeratorImageInput{}, false
		}
		representation := strings.ToLower(strings.TrimSpace(azureContentModeratorString(body["DataRepresentation"])))
		if representation != "" && representation != "url" {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "DataRepresentation must be URL when provided", "provider": providerAzure, "path": path})
			return azureContentModeratorImageInput{}, false
		}
		value := strings.TrimSpace(azureContentModeratorString(body["Value"]))
		if value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "Value is required for overload=url", "provider": providerAzure, "path": path})
			return azureContentModeratorImageInput{}, false
		}
		return azureContentModeratorImageInput{
			Overload:   overload,
			Source:     strings.ToLower(value),
			SizeBytes:  len(value),
			CacheImage: cacheImage,
		}, true
	}

	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body", "provider": providerAzure, "path": path})
		return azureContentModeratorImageInput{}, false
	}
	if len(body) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body is required for overload=stream", "provider": providerAzure, "path": path})
		return azureContentModeratorImageInput{}, false
	}

	source := strings.ToLower(strings.TrimSpace(string(body)))
	if source == "" {
		source = "binary:" + azureContentModeratorCacheID("stream", string(body))
	}
	return azureContentModeratorImageInput{
		Overload:   overload,
		Source:     source,
		SizeBytes:  len(body),
		CacheImage: cacheImage,
	}, true
}

func decodeAzureContentModeratorURLPayload(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body", "provider": providerAzure, "path": path})
		return nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body is required for overload=url", "provider": providerAzure, "path": path})
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body must be valid JSON", "provider": providerAzure, "path": path})
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func parseAzureContentModeratorBoolQuery(w http.ResponseWriter, path, key, raw string, fallback bool) (bool, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseBool(strings.ToLower(value))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": key + " must be a boolean", "provider": providerAzure, "path": path})
		return false, false
	}
	return parsed, true
}

func azureContentModeratorAdvancedInfo(input azureContentModeratorImageInput) []map[string]any {
	items := []map[string]any{{
		"Key":   "ImageSizeInBytes",
		"Value": strconv.Itoa(input.SizeBytes),
	}}
	if input.Overload == "url" {
		items = append(items, map[string]any{"Key": "ImageDownloadTimeInMs", "Value": "40"})
	}
	return items
}

func azureContentModeratorStatusOK() map[string]any {
	return map[string]any{
		"Code":        3000,
		"Description": "OK",
		"Exception":   "",
	}
}

func azureContentModeratorClassificationScores(source string) (float64, float64) {
	if azureContentModeratorHasAny(source, "adult", "nsfw", "explicit") {
		return 0.93, 0.74
	}
	if azureContentModeratorHasAny(source, "racy", "lingerie", "bikini") {
		return 0.22, 0.81
	}
	adult := 0.02 + float64(azureContentModeratorSeed("adult", source)%20)/100
	racy := 0.03 + float64(azureContentModeratorSeed("racy", source)%25)/100
	return adult, racy
}

func azureContentModeratorFaceCount(source string) int {
	if azureContentModeratorHasAny(source, "noface", "no-face", "blank") {
		return 0
	}
	if azureContentModeratorHasAny(source, "crowd", "group", "multi") {
		return 2
	}
	return 1
}

func azureContentModeratorOCRText(source string) string {
	if azureContentModeratorHasAny(source, "invoice") {
		return "INVOICE 1001 TOTAL 25.00"
	}
	if azureContentModeratorHasAny(source, "receipt") {
		return "RECEIPT SUBTOTAL 10.00"
	}
	return "STACKYARD OCR TEXT"
}

func azureContentModeratorTrackingID(operation, source string) string {
	sum := sha256.Sum256([]byte(operation + "|" + source))
	h := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

func azureContentModeratorCacheID(operation, source string) string {
	sum := sha256.Sum256([]byte("cache|" + operation + "|" + source))
	return strings.ToLower(hex.EncodeToString(sum[:8]))
}

func azureContentModeratorSeed(namespace, source string) uint64 {
	sum := sha256.Sum256([]byte(namespace + "|" + source))
	return binary.BigEndian.Uint64(sum[:8])
}

func azureContentModeratorHasAny(source string, needles ...string) bool {
	lower := strings.ToLower(source)
	for _, needle := range needles {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}

func azureContentModeratorString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
