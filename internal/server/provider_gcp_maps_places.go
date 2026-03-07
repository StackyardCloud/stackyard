package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMapsPlacesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMapsPlacesPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.maps.places.v1.Places/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPMapsPlacesSearchText(w, r, path) {
			return true
		}
		if handleGCPMapsPlacesSearchNearby(w, r, path) {
			return true
		}
		if handleGCPMapsPlacesAutocomplete(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodGet:
		if handleGCPMapsPlacesGetPlace(w, path) {
			return true
		}
		if handleGCPMapsPlacesGetPhotoMedia(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMapsPlacesPath(path string) bool {
	normalizedPath := normalizeGCPMapsPlacesPath(path)

	if strings.HasPrefix(normalizedPath, "/gcp/google.maps.places.v1.Places/") {
		return true
	}

	if normalizedPath == "/gcp/v1/places:searchNearby" ||
		normalizedPath == "/gcp/v1/places:searchText" ||
		normalizedPath == "/gcp/v1/places:autocomplete" {
		return true
	}

	return strings.HasPrefix(normalizedPath, "/gcp/v1/places/")
}

func handleGCPMapsPlacesSearchText(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsPlacesPath(path) != "/gcp/v1/places:searchText" {
		return false
	}

	body, valid := decodeGCPMapsPlacesJSONBody(w, r, path)
	if !valid {
		return true
	}

	textQuery, _ := body["textQuery"].(string)
	if strings.TrimSpace(textQuery) == "" {
		respondGCPMapsPlacesInvalidArgument(w, path, "textQuery is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"places": []any{
			gcpMapsPlacesPlace("ChIJj61dQgK6j4AR4GeTYWZsKWw"),
		},
	})
	return true
}

func handleGCPMapsPlacesSearchNearby(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsPlacesPath(path) != "/gcp/v1/places:searchNearby" {
		return false
	}

	body, valid := decodeGCPMapsPlacesJSONBody(w, r, path)
	if !valid {
		return true
	}

	locationRestriction, _ := body["locationRestriction"].(map[string]any)
	if len(locationRestriction) == 0 {
		respondGCPMapsPlacesInvalidArgument(w, path, "locationRestriction is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"places": []any{
			gcpMapsPlacesPlace("ChIJj61dQgK6j4AR4GeTYWZsKWw"),
		},
	})
	return true
}

func handleGCPMapsPlacesAutocomplete(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsPlacesPath(path) != "/gcp/v1/places:autocomplete" {
		return false
	}

	body, valid := decodeGCPMapsPlacesJSONBody(w, r, path)
	if !valid {
		return true
	}

	input, _ := body["input"].(string)
	if strings.TrimSpace(input) == "" {
		respondGCPMapsPlacesInvalidArgument(w, path, "input is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"suggestions": []any{
			map[string]any{
				"placePrediction": map[string]any{
					"place":   "places/ChIJj61dQgK6j4AR4GeTYWZsKWw",
					"placeId": "ChIJj61dQgK6j4AR4GeTYWZsKWw",
					"text": map[string]any{
						"text": "Stackyard Coffee",
					},
				},
			},
		},
	})
	return true
}

func handleGCPMapsPlacesGetPlace(w http.ResponseWriter, path string) bool {
	placeID, ok := parseGCPMapsPlacesResourceID(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpMapsPlacesPlace(placeID))
	return true
}

func handleGCPMapsPlacesGetPhotoMedia(w http.ResponseWriter, r *http.Request, path string) bool {
	placeID, photoRef, ok := parseGCPMapsPlacesPhotoPath(path)
	if !ok {
		return false
	}

	maxWidth := 400
	if raw := strings.TrimSpace(r.URL.Query().Get("maxWidthPx")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			respondGCPMapsPlacesInvalidArgument(w, path, "maxWidthPx must be a positive integer")
			return true
		}
		maxWidth = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("skipHttpRedirect")); raw != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			respondGCPMapsPlacesInvalidArgument(w, path, "skipHttpRedirect must be a boolean")
			return true
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name":     fmt.Sprintf("places/%s/photos/%s/media", placeID, photoRef),
		"photoUri": fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?photo_reference=%s&maxwidth=%d", photoRef, maxWidth),
	})
	return true
}

func decodeGCPMapsPlacesJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMapsPlacesInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPMapsPlacesResourceID(path string) (placeID string, ok bool) {
	normalizedPath := normalizeGCPMapsPlacesPath(path)
	if !strings.HasPrefix(normalizedPath, "/gcp/v1/places/") {
		return "", false
	}

	trimmed := strings.TrimPrefix(normalizedPath, "/gcp/v1/places/")
	if strings.Contains(trimmed, "/") || strings.TrimSpace(trimmed) == "" {
		return "", false
	}

	return strings.TrimSpace(trimmed), true
}

func parseGCPMapsPlacesPhotoPath(path string) (placeID, photoRef string, ok bool) {
	normalizedPath := normalizeGCPMapsPlacesPath(path)
	if !strings.HasPrefix(normalizedPath, "/gcp/v1/places/") {
		return "", "", false
	}

	trimmed := strings.TrimPrefix(normalizedPath, "/gcp/v1/places/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/places/{placeID}/photos/{photoRef}/media
	if len(parts) != 4 || parts[1] != "photos" || parts[3] != "media" {
		return "", "", false
	}

	placeID = strings.TrimSpace(parts[0])
	photoRef = strings.TrimSpace(parts[2])
	if placeID == "" || photoRef == "" {
		return "", "", false
	}
	return placeID, photoRef, true
}

func gcpMapsPlacesPlace(placeID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("places/%s", placeID),
		"id":   placeID,
		"displayName": map[string]any{
			"text":         "Stackyard Coffee",
			"languageCode": "en",
		},
		"types": []any{
			"cafe",
			"food",
		},
		"formattedAddress": "1 Stackyard Plaza, Local City, US",
		"googleMapsUri":    "https://maps.google.com/?cid=1234567890",
		"location": map[string]any{
			"latitude":  37.7937,
			"longitude": -122.3965,
		},
	}
}

func normalizeGCPMapsPlacesPath(path string) string {
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func respondGCPMapsPlacesInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
