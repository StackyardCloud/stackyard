package server

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpStreetViewPublishReferenceTime      = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpStreetViewPublishPhotoIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{0,127}$`)
	gcpStreetViewPublishSequenceIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{0,127}$`)
	gcpStreetViewPublishOperationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{0,255}$`)
)

func (s *Server) handleGCPStreetViewPublishRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_streetview_publish(w, r) {
		return true
	}

	path := normalizeGCPStreetViewPublishPath(rawRequestPath(r))
	if isGCPStreetViewPublishLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPStreetViewPublishListLocations(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPStreetViewPublishPath(path, hasGCPStreetViewPublishHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPStreetViewPublishGetPhoto(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishBatchGetPhotos(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishListPhotos(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishGetPhotoSequence(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishListPhotoSequences(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishGetOperation(w, path) {
			return true
		}
		if handleGCPStreetViewPublishListOperations(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPStreetViewPublishStartUpload(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishCreatePhoto(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishBatchUpdatePhotos(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishBatchDeletePhotos(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishStartPhotoSequenceUpload(w, r, path) {
			return true
		}
		if handleGCPStreetViewPublishCreatePhotoSequence(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPut:
		if handleGCPStreetViewPublishUpdatePhoto(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPStreetViewPublishDeletePhoto(w, path) {
			return true
		}
		if handleGCPStreetViewPublishDeletePhotoSequence(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPStreetViewPublishPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPStreetViewPublishHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "streetview_publish",
		"streetview-publish",
		"streetview-publish-apiv1",
		"streetview_publish_apiv1",
		"streetviewpublish",
		"streetviewpublish-apiv1",
		"street-view-publish",
		"street_view_publish",
		"gcp-street-view-publish":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-streetview-publish-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/streetview/publish/apiv1")
}

func isGCPStreetViewPublishLocationRequest(r *http.Request, path string) bool {
	if !hasGCPStreetViewPublishHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPStreetViewPublishPath(path string, includeHint bool) bool {
	if !includeHint {
		return false
	}
	if parseGCPStreetViewPublishStartUploadPath(path) ||
		parseGCPStreetViewPublishCreatePhotoPath(path) ||
		parseGCPStreetViewPublishBatchGetPhotosPath(path) ||
		parseGCPStreetViewPublishListPhotosPath(path) ||
		parseGCPStreetViewPublishBatchUpdatePhotosPath(path) ||
		parseGCPStreetViewPublishBatchDeletePhotosPath(path) ||
		parseGCPStreetViewPublishStartPhotoSequenceUploadPath(path) ||
		parseGCPStreetViewPublishCreatePhotoSequencePath(path) ||
		parseGCPStreetViewPublishListPhotoSequencesPath(path) ||
		parseGCPStreetViewPublishListOperationsPath(path) {
		return true
	}
	if _, ok := parseGCPStreetViewPublishPhotoPath(path); ok {
		return true
	}
	if _, ok := parseGCPStreetViewPublishPhotoSequencePath(path); ok {
		return true
	}
	if _, ok := parseGCPStreetViewPublishOperationPath(path); ok {
		return true
	}
	return false
}

func handleGCPStreetViewPublishListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPStreetViewPublishPagination(w, r, path, 256)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpStreetViewPublishLocation(project, "us-central1"),
		gcpStreetViewPublishLocation(project, "global"),
	}
	return respondGCPStreetViewPublishList(w, "locations", items, pageSize, start)
}

func handleGCPStreetViewPublishGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpStreetViewPublishLocation(project, location))
	return true
}

func handleGCPStreetViewPublishStartUpload(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishStartUploadPath(path) {
		return false
	}
	if _, valid := decodeGCPStreetViewPublishJSONBodyOptional(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpStreetViewPublishUploadRef("photo-upload-1"))
	return true
}

func handleGCPStreetViewPublishCreatePhoto(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishCreatePhotoPath(path) {
		return false
	}
	photo, valid := decodeGCPStreetViewPublishJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	photoID := gcpStreetViewPublishNestedString(photo, "photoId", "id")
	if photoID == "" {
		photoID = "photo-1"
	}
	if !isGCPStreetViewPublishPhotoID(photoID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photo.photoId.id is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(photoID), "existing") {
		respondGCPStreetViewPublishAlreadyExists(w, path, "photo already exists")
		return true
	}

	uploadURL := gcpStreetViewPublishNestedString(photo, "uploadReference", "uploadUrl")
	if !isGCPStreetViewPublishUploadURL(uploadURL) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photo.uploadReference.uploadUrl is required")
		return true
	}
	if strings.Contains(strings.ToLower(uploadURL), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "upload reference not found")
		return true
	}

	created := gcpStreetViewPublishPhoto(photoID, false)
	if pose, ok := photo["pose"].(map[string]any); ok {
		created["pose"] = pose
	}
	if places, ok := photo["places"].([]any); ok {
		created["places"] = places
	}
	if connections, ok := photo["connections"].([]any); ok {
		created["connections"] = connections
	}
	respondJSON(w, http.StatusOK, created)
	return true
}

func handleGCPStreetViewPublishGetPhoto(w http.ResponseWriter, r *http.Request, path string) bool {
	photoID, ok := parseGCPStreetViewPublishPhotoPath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishPhotoID(photoID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoId is invalid")
		return true
	}
	view, valid := parseGCPStreetViewPublishPhotoView(r.URL.Query().Get("view"), true)
	if !valid {
		respondGCPStreetViewPublishInvalidArgument(w, path, "view is required and must be BASIC or INCLUDE_DOWNLOAD_URL")
		return true
	}
	if strings.Contains(strings.ToLower(photoID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "photo not found")
		return true
	}
	_ = strings.TrimSpace(r.URL.Query().Get("languageCode"))
	respondJSON(w, http.StatusOK, gcpStreetViewPublishPhoto(photoID, view == "INCLUDE_DOWNLOAD_URL"))
	return true
}

func handleGCPStreetViewPublishBatchGetPhotos(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishBatchGetPhotosPath(path) {
		return false
	}
	view, valid := parseGCPStreetViewPublishPhotoView(r.URL.Query().Get("view"), true)
	if !valid {
		respondGCPStreetViewPublishInvalidArgument(w, path, "view is required and must be BASIC or INCLUDE_DOWNLOAD_URL")
		return true
	}
	photoIDs := r.URL.Query()["photoIds"]
	if len(photoIDs) == 0 {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoIds is required")
		return true
	}

	results := make([]any, 0, len(photoIDs))
	for _, rawID := range photoIDs {
		photoID := strings.TrimSpace(rawID)
		if !isGCPStreetViewPublishPhotoID(photoID) {
			respondGCPStreetViewPublishInvalidArgument(w, path, "photoIds contains an invalid id")
			return true
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			results = append(results, map[string]any{
				"status": gcpStreetViewPublishStatus(5, "photo not found"),
			})
			continue
		}
		results = append(results, map[string]any{
			"status": gcpStreetViewPublishStatus(0, "OK"),
			"photo":  gcpStreetViewPublishPhoto(photoID, view == "INCLUDE_DOWNLOAD_URL"),
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"results": results})
	return true
}

func handleGCPStreetViewPublishListPhotos(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishListPhotosPath(path) {
		return false
	}
	view, valid := parseGCPStreetViewPublishPhotoView(r.URL.Query().Get("view"), true)
	if !valid {
		respondGCPStreetViewPublishInvalidArgument(w, path, "view is required and must be BASIC or INCLUDE_DOWNLOAD_URL")
		return true
	}
	pageSize, start, valid := parseGCPStreetViewPublishPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	filter, err := parseGCPStreetViewPublishFilter(r.URL.Query().Get("filter"), map[string]string{
		"placeId":       "string",
		"min_latitude":  "float",
		"max_latitude":  "float",
		"min_longitude": "float",
		"max_longitude": "float",
	})
	if err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, err.Error())
		return true
	}

	items := []map[string]any{
		gcpStreetViewPublishPhoto("photo-1", view == "INCLUDE_DOWNLOAD_URL"),
		gcpStreetViewPublishPhoto("photo-2", view == "INCLUDE_DOWNLOAD_URL"),
	}
	if placeID := strings.TrimSpace(filter["placeId"]); placeID != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			places, _ := item["places"].([]any)
			matched := false
			for _, placeAny := range places {
				place, _ := placeAny.(map[string]any)
				if strings.EqualFold(strings.TrimSpace(gcpStreetViewPublishString(place, "placeId")), placeID) {
					matched = true
					break
				}
			}
			if matched {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	items = filterGCPStreetViewPublishByBounds(items, filter)
	sort.Slice(items, func(i, j int) bool {
		return gcpStreetViewPublishNestedString(items[i], "photoId", "id") < gcpStreetViewPublishNestedString(items[j], "photoId", "id")
	})
	respondGCPStreetViewPublishList(w, "photos", items, pageSize, start)
	return true
}

func handleGCPStreetViewPublishUpdatePhoto(w http.ResponseWriter, r *http.Request, path string) bool {
	photoID, ok := parseGCPStreetViewPublishPhotoPath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishPhotoID(photoID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(photoID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "photo not found")
		return true
	}

	photo, valid := decodeGCPStreetViewPublishJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	if bodyPhotoID := gcpStreetViewPublishNestedString(photo, "photoId", "id"); bodyPhotoID != "" && bodyPhotoID != photoID {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photo.photoId.id must match path")
		return true
	}
	mask, err := parseGCPStreetViewPublishUpdateMask(r.URL.Query().Get("updateMask"))
	if err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, err.Error())
		return true
	}
	if hasGCPStreetViewPublishMask(mask, "pose.altitude") && gcpStreetViewPublishNestedMap(photo, "pose") != nil {
		if gcpStreetViewPublishNestedMap(gcpStreetViewPublishNestedMap(photo, "pose"), "latLngPair") == nil {
			respondGCPStreetViewPublishInvalidArgument(w, path, "pose.altitude requires pose.latLngPair")
			return true
		}
	}

	updated := gcpStreetViewPublishPhoto(photoID, false)
	if len(mask) == 0 {
		applyGCPStreetViewPublishPhotoPatch(updated, photo)
	} else {
		applyGCPStreetViewPublishPhotoMaskPatch(updated, photo, mask)
	}
	respondJSON(w, http.StatusOK, updated)
	return true
}

func handleGCPStreetViewPublishBatchUpdatePhotos(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishBatchUpdatePhotosPath(path) {
		return false
	}
	body, valid := decodeGCPStreetViewPublishJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	reqItems, ok := body["updatePhotoRequests"].([]any)
	if !ok || len(reqItems) == 0 {
		respondGCPStreetViewPublishInvalidArgument(w, path, "updatePhotoRequests is required")
		return true
	}
	if len(reqItems) > 20 {
		respondGCPStreetViewPublishInvalidArgument(w, path, "updatePhotoRequests cannot exceed 20")
		return true
	}

	results := make([]any, 0, len(reqItems))
	for _, itemAny := range reqItems {
		item, _ := itemAny.(map[string]any)
		photo := gcpStreetViewPublishBodyMap(item, "photo")
		if len(photo) == 0 {
			results = append(results, map[string]any{"status": gcpStreetViewPublishStatus(3, "photo is required")})
			continue
		}
		photoID := gcpStreetViewPublishNestedString(photo, "photoId", "id")
		if !isGCPStreetViewPublishPhotoID(photoID) {
			results = append(results, map[string]any{"status": gcpStreetViewPublishStatus(3, "photoId is invalid")})
			continue
		}
		mask, err := parseGCPStreetViewPublishUpdateMaskFromAny(item["updateMask"])
		if err != nil {
			results = append(results, map[string]any{"status": gcpStreetViewPublishStatus(3, err.Error())})
			continue
		}
		if hasGCPStreetViewPublishMask(mask, "pose.altitude") && gcpStreetViewPublishNestedMap(photo, "pose") != nil {
			if gcpStreetViewPublishNestedMap(gcpStreetViewPublishNestedMap(photo, "pose"), "latLngPair") == nil {
				results = append(results, map[string]any{"status": gcpStreetViewPublishStatus(3, "pose.altitude requires pose.latLngPair")})
				continue
			}
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			results = append(results, map[string]any{"status": gcpStreetViewPublishStatus(5, "photo not found")})
			continue
		}

		updated := gcpStreetViewPublishPhoto(photoID, false)
		if len(mask) == 0 {
			applyGCPStreetViewPublishPhotoPatch(updated, photo)
		} else {
			applyGCPStreetViewPublishPhotoMaskPatch(updated, photo, mask)
		}
		results = append(results, map[string]any{
			"status": gcpStreetViewPublishStatus(0, "OK"),
			"photo":  updated,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"results": results})
	return true
}

func handleGCPStreetViewPublishDeletePhoto(w http.ResponseWriter, path string) bool {
	photoID, ok := parseGCPStreetViewPublishPhotoPath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishPhotoID(photoID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(photoID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "photo not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStreetViewPublishBatchDeletePhotos(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishBatchDeletePhotosPath(path) {
		return false
	}
	body, valid := decodeGCPStreetViewPublishJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	photoIDs, ok := body["photoIds"].([]any)
	if !ok || len(photoIDs) == 0 {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoIds is required")
		return true
	}
	statuses := make([]any, 0, len(photoIDs))
	for _, idAny := range photoIDs {
		photoID := strings.TrimSpace(fmt.Sprint(idAny))
		if !isGCPStreetViewPublishPhotoID(photoID) {
			respondGCPStreetViewPublishInvalidArgument(w, path, "photoIds contains an invalid id")
			return true
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			statuses = append(statuses, gcpStreetViewPublishStatus(5, "photo not found"))
			continue
		}
		statuses = append(statuses, gcpStreetViewPublishStatus(0, "OK"))
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": statuses})
	return true
}

func handleGCPStreetViewPublishStartPhotoSequenceUpload(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishStartPhotoSequenceUploadPath(path) {
		return false
	}
	if _, valid := decodeGCPStreetViewPublishJSONBodyOptional(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpStreetViewPublishUploadRef("photo-sequence-upload-1"))
	return true
}

func handleGCPStreetViewPublishCreatePhotoSequence(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishCreatePhotoSequencePath(path) {
		return false
	}
	photoSequence, valid := decodeGCPStreetViewPublishJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	inputType, ok := parseGCPStreetViewPublishInputType(r.URL.Query().Get("inputType"))
	if !ok {
		respondGCPStreetViewPublishInvalidArgument(w, path, "inputType is required and must be VIDEO or XDM")
		return true
	}

	sequenceID := strings.TrimSpace(gcpStreetViewPublishString(photoSequence, "id"))
	if sequenceID == "" {
		sequenceID = "sequence-1"
	}
	if !isGCPStreetViewPublishSequenceID(sequenceID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoSequence.id is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(sequenceID), "existing") {
		respondGCPStreetViewPublishAlreadyExists(w, path, "photo sequence already exists")
		return true
	}
	uploadURL := gcpStreetViewPublishNestedString(photoSequence, "uploadReference", "uploadUrl")
	if !isGCPStreetViewPublishUploadURL(uploadURL) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "photoSequence.uploadReference.uploadUrl is required")
		return true
	}

	done := !isGCPStreetViewPublishProcessingID(sequenceID)
	respondJSON(w, http.StatusOK, gcpStreetViewPublishSequenceOperation(sequenceID, done, inputType))
	return true
}

func handleGCPStreetViewPublishGetPhotoSequence(w http.ResponseWriter, r *http.Request, path string) bool {
	sequenceID, ok := parseGCPStreetViewPublishPhotoSequencePath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishSequenceID(sequenceID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "sequenceId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(sequenceID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "photo sequence not found")
		return true
	}
	if _, err := parseGCPStreetViewPublishFilter(r.URL.Query().Get("filter"), map[string]string{
		"published_status": "string",
	}); err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, err.Error())
		return true
	}
	if rawView := strings.TrimSpace(r.URL.Query().Get("view")); rawView != "" {
		if _, valid := parseGCPStreetViewPublishPhotoView(rawView, false); !valid {
			respondGCPStreetViewPublishInvalidArgument(w, path, "view is invalid")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpStreetViewPublishSequenceOperation(sequenceID, !isGCPStreetViewPublishProcessingID(sequenceID), "VIDEO"))
	return true
}

func handleGCPStreetViewPublishListPhotoSequences(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishListPhotoSequencesPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPStreetViewPublishPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	filter, err := parseGCPStreetViewPublishFilter(r.URL.Query().Get("filter"), map[string]string{
		"imagery_type":             "string",
		"processing_state":         "string",
		"min_latitude":             "float",
		"max_latitude":             "float",
		"min_longitude":            "float",
		"max_longitude":            "float",
		"filename_query":           "string",
		"min_capture_time_seconds": "int",
		"max_capture_time_seconds": "int",
	})
	if err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, err.Error())
		return true
	}

	items := []map[string]any{
		gcpStreetViewPublishSequenceOperation("sequence-1", true, "VIDEO"),
		gcpStreetViewPublishSequenceOperation("sequence-processing", false, "VIDEO"),
	}
	if state := strings.ToUpper(strings.TrimSpace(filter["processing_state"])); state != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			done, _ := item["done"].(bool)
			opState := "PROCESSING"
			if done {
				opState = "PROCESSED"
			}
			if opState == state {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpStreetViewPublishString(items[i], "name") < gcpStreetViewPublishString(items[j], "name")
	})
	respondGCPStreetViewPublishList(w, "photoSequences", items, pageSize, start)
	return true
}

func handleGCPStreetViewPublishDeletePhotoSequence(w http.ResponseWriter, path string) bool {
	sequenceID, ok := parseGCPStreetViewPublishPhotoSequencePath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishSequenceID(sequenceID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "sequenceId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(sequenceID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "photo sequence not found")
		return true
	}
	if isGCPStreetViewPublishProcessingID(sequenceID) {
		respondGCPStreetViewPublishFailedPrecondition(w, path, "photo sequence is still processing")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStreetViewPublishGetOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPStreetViewPublishOperationPath(path)
	if !ok {
		return false
	}
	if !isGCPStreetViewPublishOperationID(operationID) {
		respondGCPStreetViewPublishInvalidArgument(w, path, "operation name is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(operationID), "missing") {
		respondGCPStreetViewPublishNotFound(w, path, "operation not found")
		return true
	}
	sequenceID := gcpStreetViewPublishSequenceIDFromOperationID(operationID)
	respondJSON(w, http.StatusOK, gcpStreetViewPublishOperation(operationID, sequenceID, !isGCPStreetViewPublishProcessingID(sequenceID), "VIDEO"))
	return true
}

func handleGCPStreetViewPublishListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStreetViewPublishListOperationsPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPStreetViewPublishPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	if filterRaw := strings.TrimSpace(r.URL.Query().Get("filter")); filterRaw != "" {
		if filterRaw != "done=true" && filterRaw != "done=false" {
			respondGCPStreetViewPublishInvalidArgument(w, path, "filter must be done=true or done=false")
			return true
		}
	}

	items := []map[string]any{
		gcpStreetViewPublishOperation("photoSequence.sequence-1", "sequence-1", true, "VIDEO"),
		gcpStreetViewPublishOperation("photoSequence.sequence-processing", "sequence-processing", false, "VIDEO"),
	}
	if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" {
		wantDone := filter == "done=true"
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			done, _ := item["done"].(bool)
			if done == wantDone {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	respondGCPStreetViewPublishList(w, "operations", items, pageSize, start)
	return true
}

func parseGCPStreetViewPublishStartUploadPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photo:startUpload"
}

func parseGCPStreetViewPublishCreatePhotoPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photo"
}

func parseGCPStreetViewPublishPhotoPath(path string) (photoID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "photo" {
		return "", false
	}
	photoID = strings.TrimSpace(parts[3])
	if photoID == "" {
		return "", false
	}
	return photoID, true
}

func parseGCPStreetViewPublishBatchGetPhotosPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photos:batchGet"
}

func parseGCPStreetViewPublishListPhotosPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photos"
}

func parseGCPStreetViewPublishBatchUpdatePhotosPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photos:batchUpdate"
}

func parseGCPStreetViewPublishBatchDeletePhotosPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photos:batchDelete"
}

func parseGCPStreetViewPublishStartPhotoSequenceUploadPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photoSequence:startUpload"
}

func parseGCPStreetViewPublishCreatePhotoSequencePath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photoSequence"
}

func parseGCPStreetViewPublishPhotoSequencePath(path string) (sequenceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "photoSequence" {
		return "", false
	}
	sequenceID = strings.TrimSpace(parts[3])
	if sequenceID == "" {
		return "", false
	}
	return sequenceID, true
}

func parseGCPStreetViewPublishListPhotoSequencesPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "photoSequences"
}

func parseGCPStreetViewPublishOperationPath(path string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "operations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[3])
	if operationID == "" {
		return "", false
	}
	return operationID, true
}

func parseGCPStreetViewPublishListOperationsPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "operations"
}

func parseGCPStreetViewPublishPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPStreetViewPublishInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if n > maxPageSize {
			respondGCPStreetViewPublishOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPStreetViewPublishInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = n
	}
	return pageSize, start, true
}

func respondGCPStreetViewPublishList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int) bool {
	if start > len(items) {
		start = len(items)
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	values := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		values = append(values, item)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             values,
		"nextPageToken": next,
	})
	return true
}

func parseGCPStreetViewPublishFilter(raw string, allowed map[string]string) (map[string]string, error) {
	out := map[string]string{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return out, nil
	}
	segments := splitGCPStreetViewPublishFilterSegments(trimmed)
	if len(segments) == 0 {
		return out, fmt.Errorf("filter is invalid")
	}
	for _, segment := range segments {
		k, v, ok := strings.Cut(segment, "=")
		if !ok {
			return nil, fmt.Errorf("filter is invalid")
		}
		key := strings.TrimSpace(k)
		value := strings.Trim(strings.TrimSpace(v), "\"")
		if key == "" || value == "" {
			return nil, fmt.Errorf("filter is invalid")
		}
		kind, supported := allowed[key]
		if !supported {
			return nil, fmt.Errorf("filter key %q is not supported", key)
		}
		switch kind {
		case "float":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return nil, fmt.Errorf("filter value for %q must be a float", key)
			}
		case "int":
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return nil, fmt.Errorf("filter value for %q must be an integer", key)
			}
		}
		out[key] = value
	}
	return out, nil
}

func splitGCPStreetViewPublishFilterSegments(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		out = append(out, piece)
	}
	return out
}

func filterGCPStreetViewPublishByBounds(items []map[string]any, filter map[string]string) []map[string]any {
	minLat := parseGCPStreetViewPublishFloatOrNaN(filter["min_latitude"])
	maxLat := parseGCPStreetViewPublishFloatOrNaN(filter["max_latitude"])
	minLng := parseGCPStreetViewPublishFloatOrNaN(filter["min_longitude"])
	maxLng := parseGCPStreetViewPublishFloatOrNaN(filter["max_longitude"])
	if math.IsNaN(minLat) && math.IsNaN(maxLat) && math.IsNaN(minLng) && math.IsNaN(maxLng) {
		return items
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		pose, _ := item["pose"].(map[string]any)
		latLngPair, _ := pose["latLngPair"].(map[string]any)
		lat, _ := asFloat64GCPStreetViewPublish(latLngPair["latitude"])
		lng, _ := asFloat64GCPStreetViewPublish(latLngPair["longitude"])
		if !math.IsNaN(minLat) && lat < minLat {
			continue
		}
		if !math.IsNaN(maxLat) && lat > maxLat {
			continue
		}
		if !math.IsNaN(minLng) && lng < minLng {
			continue
		}
		if !math.IsNaN(maxLng) && lng > maxLng {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func parseGCPStreetViewPublishFloatOrNaN(raw string) float64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return math.NaN()
	}
	n, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return math.NaN()
	}
	return n
}

func parseGCPStreetViewPublishPhotoView(raw string, required bool) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if required {
			return "", false
		}
		return "BASIC", true
	}
	switch strings.ToUpper(trimmed) {
	case "0", "BASIC":
		return "BASIC", true
	case "1", "INCLUDE_DOWNLOAD_URL":
		return "INCLUDE_DOWNLOAD_URL", true
	default:
		return "", false
	}
}

func parseGCPStreetViewPublishInputType(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	switch strings.ToUpper(trimmed) {
	case "1", "VIDEO":
		return "VIDEO", true
	case "2", "XDM":
		return "XDM", true
	default:
		return "", false
	}
}

func parseGCPStreetViewPublishUpdateMask(raw string) ([]string, error) {
	allowed := map[string]struct{}{
		"pose.heading":      {},
		"pose.lat_lng_pair": {},
		"pose.pitch":        {},
		"pose.roll":         {},
		"pose.level":        {},
		"pose.altitude":     {},
		"connections":       {},
		"places":            {},
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		path := strings.TrimSpace(part)
		if path == "" {
			continue
		}
		if _, ok := allowed[path]; !ok {
			return nil, fmt.Errorf("updateMask contains unsupported field %q", path)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func parseGCPStreetViewPublishUpdateMaskFromAny(value any) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		return parseGCPStreetViewPublishUpdateMask(typed)
	case map[string]any:
		if pathsAny, ok := typed["paths"].([]any); ok {
			parts := make([]string, 0, len(pathsAny))
			for _, pathAny := range pathsAny {
				parts = append(parts, strings.TrimSpace(fmt.Sprint(pathAny)))
			}
			return parseGCPStreetViewPublishUpdateMask(strings.Join(parts, ","))
		}
		if single, ok := typed["paths"].(string); ok {
			return parseGCPStreetViewPublishUpdateMask(single)
		}
		return nil, fmt.Errorf("updateMask is invalid")
	default:
		return nil, fmt.Errorf("updateMask is invalid")
	}
}

func hasGCPStreetViewPublishMask(mask []string, field string) bool {
	for _, path := range mask {
		if path == field {
			return true
		}
	}
	return false
}

func applyGCPStreetViewPublishPhotoPatch(dst map[string]any, patch map[string]any) {
	for key, value := range patch {
		switch key {
		case "photoId", "uploadReference", "downloadUrl", "thumbnailUrl", "shareLink", "pose", "connections", "captureTime", "uploadTime", "places", "viewCount", "transferStatus", "mapsPublishStatus":
			dst[key] = value
		}
	}
}

func applyGCPStreetViewPublishPhotoMaskPatch(dst map[string]any, patch map[string]any, mask []string) {
	for _, path := range mask {
		switch path {
		case "connections":
			if v, ok := patch["connections"]; ok {
				dst["connections"] = v
			}
		case "places":
			if v, ok := patch["places"]; ok {
				dst["places"] = v
			}
		case "pose.heading", "pose.lat_lng_pair", "pose.pitch", "pose.roll", "pose.level", "pose.altitude":
			if posePatch, ok := patch["pose"].(map[string]any); ok {
				poseCurrent, _ := dst["pose"].(map[string]any)
				if poseCurrent == nil {
					poseCurrent = map[string]any{}
				}
				switch path {
				case "pose.heading":
					if v, ok := posePatch["heading"]; ok {
						poseCurrent["heading"] = v
					}
				case "pose.pitch":
					if v, ok := posePatch["pitch"]; ok {
						poseCurrent["pitch"] = v
					}
				case "pose.roll":
					if v, ok := posePatch["roll"]; ok {
						poseCurrent["roll"] = v
					}
				case "pose.level":
					if v, ok := posePatch["level"]; ok {
						poseCurrent["level"] = v
					}
				case "pose.altitude":
					if v, ok := posePatch["altitude"]; ok {
						poseCurrent["altitude"] = v
					}
				case "pose.lat_lng_pair":
					if v, ok := posePatch["latLngPair"]; ok {
						poseCurrent["latLngPair"] = v
					}
				}
				dst["pose"] = poseCurrent
			}
		}
	}
}

func decodeGCPStreetViewPublishJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, "request body is invalid")
		return nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
		respondGCPStreetViewPublishInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, "request body is invalid")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func decodeGCPStreetViewPublishJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, "request body is invalid")
		return nil, false
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondGCPStreetViewPublishInvalidArgument(w, path, "request body is invalid")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func isGCPStreetViewPublishPhotoID(photoID string) bool {
	photoID = strings.TrimSpace(photoID)
	if photoID == "" {
		return false
	}
	return gcpStreetViewPublishPhotoIDPattern.MatchString(photoID)
}

func isGCPStreetViewPublishSequenceID(sequenceID string) bool {
	sequenceID = strings.TrimSpace(sequenceID)
	if sequenceID == "" {
		return false
	}
	return gcpStreetViewPublishSequenceIDPattern.MatchString(sequenceID)
}

func isGCPStreetViewPublishOperationID(operationID string) bool {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return false
	}
	return gcpStreetViewPublishOperationIDPattern.MatchString(operationID)
}

func isGCPStreetViewPublishUploadURL(uploadURL string) bool {
	trimmed := strings.TrimSpace(uploadURL)
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "https://streetviewpublish.googleapis.com/media/user/")
}

func isGCPStreetViewPublishProcessingID(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(lower, "processing") || strings.Contains(lower, "running")
}

func gcpStreetViewPublishUploadRef(uploadID string) map[string]any {
	return map[string]any{
		"uploadUrl": gcpStreetViewPublishUploadURL(uploadID),
	}
}

func gcpStreetViewPublishUploadURL(uploadID string) string {
	return fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s", strings.TrimSpace(uploadID))
}

func gcpStreetViewPublishPhoto(photoID string, includeDownloadURL bool) map[string]any {
	photo := map[string]any{
		"photoId": map[string]any{
			"id": photoID,
		},
		"uploadReference": map[string]any{
			"uploadUrl": gcpStreetViewPublishUploadURL(photoID),
		},
		"thumbnailUrl":      fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s/thumbnail", photoID),
		"shareLink":         fmt.Sprintf("https://maps.google.com/?q=stackyard-photo-%s", photoID),
		"captureTime":       gcpStreetViewPublishReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"uploadTime":        gcpStreetViewPublishReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
		"viewCount":         float64(101),
		"transferStatus":    "NEVER_TRANSFERRED",
		"mapsPublishStatus": "PUBLISHED",
		"pose": map[string]any{
			"latLngPair": map[string]any{
				"latitude":  37.422,
				"longitude": -122.084,
			},
			"heading":  90.0,
			"pitch":    1.0,
			"roll":     0.0,
			"altitude": 5.0,
		},
		"connections": []any{
			map[string]any{
				"target": map[string]any{
					"id": "photo-2",
				},
			},
		},
		"places": []any{
			map[string]any{"placeId": "ChIJj61dQgK6j4AR4GeTYWZsKWw"},
		},
	}
	if includeDownloadURL {
		photo["downloadUrl"] = fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s/download", photoID)
	}
	return photo
}

func gcpStreetViewPublishPhotoSequence(sequenceID, inputType string) map[string]any {
	return map[string]any{
		"id": sequenceID,
		"uploadReference": map[string]any{
			"uploadUrl": gcpStreetViewPublishUploadURL("sequence-" + sequenceID),
		},
		"uploadTime":      gcpStreetViewPublishReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"processingState": map[string]string{"VIDEO": "PROCESSED", "XDM": "PROCESSED"}[inputType],
		"photos": []any{
			map[string]any{"photoId": map[string]any{"id": "photo-1"}},
			map[string]any{"photoId": map[string]any{"id": "photo-2"}},
		},
		"distanceMeters": 12.5,
		"viewCount":      float64(12),
		"filename":       sequenceID + ".mp4",
	}
}

func gcpStreetViewPublishOperation(operationID, sequenceID string, done bool, inputType string) map[string]any {
	operation := map[string]any{
		"name": fmt.Sprintf("operations/%s", operationID),
		"done": done,
		"metadata": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
	if done {
		operation["response"] = gcpStreetViewPublishAnyPhotoSequence(sequenceID, inputType)
	}
	return operation
}

func gcpStreetViewPublishSequenceOperation(sequenceID string, done bool, inputType string) map[string]any {
	return gcpStreetViewPublishOperation(gcpStreetViewPublishOperationID(sequenceID), sequenceID, done, inputType)
}

func gcpStreetViewPublishOperationID(sequenceID string) string {
	return "photoSequence." + strings.TrimSpace(sequenceID)
}

func gcpStreetViewPublishSequenceIDFromOperationID(operationID string) string {
	trimmed := strings.TrimSpace(operationID)
	if strings.HasPrefix(trimmed, "photoSequence.") {
		return strings.TrimPrefix(trimmed, "photoSequence.")
	}
	if _, sequenceID, ok := strings.Cut(trimmed, "."); ok {
		if strings.TrimSpace(sequenceID) != "" {
			return strings.TrimSpace(sequenceID)
		}
	}
	return trimmed
}

func gcpStreetViewPublishAnyPhotoSequence(sequenceID, inputType string) map[string]any {
	payload := gcpStreetViewPublishPhotoSequence(sequenceID, inputType)
	payload["@type"] = "type.googleapis.com/google.streetview.publish.v1.PhotoSequence"
	return payload
}

func gcpStreetViewPublishStatus(code int, message string) map[string]any {
	return map[string]any{
		"code":    code,
		"message": message,
		"details": []any{},
	}
}

func gcpStreetViewPublishLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Street View Publish " + location,
		"labels": map[string]string{
			"service": "streetviewpublish.googleapis.com",
		},
		"metadata": map[string]any{},
	}
}

func gcpStreetViewPublishBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return nil
	}
	value, ok := body[key]
	if !ok {
		return nil
	}
	mapped, _ := value.(map[string]any)
	return mapped
}

func gcpStreetViewPublishString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	value, ok := body[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func gcpStreetViewPublishNestedString(body map[string]any, keys ...string) string {
	current := body
	for i, key := range keys {
		if i == len(keys)-1 {
			return gcpStreetViewPublishString(current, key)
		}
		next, _ := current[key].(map[string]any)
		if next == nil {
			return ""
		}
		current = next
	}
	return ""
}

func gcpStreetViewPublishNestedMap(body map[string]any, keys ...string) map[string]any {
	current := body
	for _, key := range keys {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func asFloat64GCPStreetViewPublish(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		n, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return n, true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func respondGCPStreetViewPublishInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPStreetViewPublishError(w, http.StatusBadRequest, "InvalidArgument", message, path)
}

func respondGCPStreetViewPublishNotFound(w http.ResponseWriter, path, message string) {
	respondGCPStreetViewPublishError(w, http.StatusNotFound, "NotFound", message, path)
}

func respondGCPStreetViewPublishAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPStreetViewPublishError(w, http.StatusConflict, "AlreadyExists", message, path)
}

func respondGCPStreetViewPublishFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPStreetViewPublishError(w, http.StatusBadRequest, "FailedPrecondition", message, path)
}

func respondGCPStreetViewPublishOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPStreetViewPublishError(w, http.StatusBadRequest, "OutOfRange", message, path)
}

func respondGCPStreetViewPublishError(w http.ResponseWriter, status int, code, message, path string) {
	if strings.TrimSpace(message) == "" {
		message = strings.TrimSpace(code)
	}
	respondJSON(w, status, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_streetview_publish(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "streetview_publish") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPStreetViewPublishInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}

	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/streetview_publish/sample",
			"service":  "streetview_publish",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
