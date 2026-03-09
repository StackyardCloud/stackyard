package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const azureContentModeratorImageListsPrefix = "/azure/contentmoderator/lists/v1.0/imagelists"

type azureContentModeratorImageList struct {
	ID          int64
	Name        string
	Description string
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Server) handleAzureContentModeratorListManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentModeratorImageListsPrefix) {
		return false
	}

	tail := strings.TrimPrefix(path, azureContentModeratorImageListsPrefix)
	segments := splitPathSegments(strings.TrimPrefix(tail, "/"))

	if len(segments) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureContentModeratorGetAllImageLists(w)
		case http.MethodPost:
			s.handleAzureContentModeratorCreateImageList(w, r, path)
		default:
			respondAzureImplemented(w, path)
		}
		return true
	}

	if len(segments) == 1 {
		listID, ok := parseAzureContentModeratorListID(w, path, segments[0])
		if !ok {
			return true
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAzureContentModeratorGetImageListDetails(w, listID)
		case http.MethodPut:
			s.handleAzureContentModeratorUpdateImageList(w, r, path, listID)
		case http.MethodDelete:
			s.handleAzureContentModeratorDeleteImageList(w, listID)
		default:
			respondAzureImplemented(w, path)
		}
		return true
	}

	if len(segments) == 2 {
		listID, ok := parseAzureContentModeratorListID(w, path, segments[0])
		if !ok {
			return true
		}
		if strings.EqualFold(segments[1], "RefreshIndex") && r.Method == http.MethodPost {
			s.handleAzureContentModeratorRefreshImageListIndex(w, listID)
			return true
		}
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureContentModeratorCreateImageList(w http.ResponseWriter, r *http.Request, path string) {
	payload, ok := parseAzureContentModeratorImageListPayload(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureContentModeratorNextImageListID++
	listID := s.azureContentModeratorNextImageListID
	if listID < 10000 {
		listID += 10000
		s.azureContentModeratorNextImageListID = listID
	}
	entry := &azureContentModeratorImageList{
		ID:          listID,
		Name:        payload.Name,
		Description: payload.Description,
		Metadata:    payload.Metadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	s.azureContentModeratorImageLists[listID] = entry
	respondJSON(w, http.StatusOK, azureContentModeratorImageListResponse(entry))
}

func (s *Server) handleAzureContentModeratorGetAllImageLists(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	rows := make([]map[string]any, 0, len(s.azureContentModeratorImageLists))
	for _, item := range s.azureContentModeratorImageLists {
		rows = append(rows, azureContentModeratorImageListResponse(item))
	}
	respondJSON(w, http.StatusOK, rows)
}

func (s *Server) handleAzureContentModeratorGetImageListDetails(w http.ResponseWriter, listID int64) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item := s.azureContentModeratorImageLists[listID]
	if item == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "image list not found"})
		return
	}
	respondJSON(w, http.StatusOK, azureContentModeratorImageListResponse(item))
}

func (s *Server) handleAzureContentModeratorUpdateImageList(w http.ResponseWriter, r *http.Request, path string, listID int64) {
	payload, ok := parseAzureContentModeratorImageListPayload(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item := s.azureContentModeratorImageLists[listID]
	if item == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "image list not found"})
		return
	}
	item.Name = payload.Name
	item.Description = payload.Description
	item.Metadata = payload.Metadata
	item.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureContentModeratorImageListResponse(item))
}

func (s *Server) handleAzureContentModeratorDeleteImageList(w http.ResponseWriter, listID int64) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if s.azureContentModeratorImageLists[listID] == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "image list not found"})
		return
	}
	delete(s.azureContentModeratorImageLists, listID)
	respondJSON(w, http.StatusOK, "")
}

func (s *Server) handleAzureContentModeratorRefreshImageListIndex(w http.ResponseWriter, listID int64) {
	s.providerStorageMu.Lock()
	item := s.azureContentModeratorImageLists[listID]
	s.providerStorageMu.Unlock()
	if item == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "image list not found"})
		return
	}

	trackingID := azureContentModeratorTrackingID("imagelists-refresh", fmt.Sprintf("%d|%s", item.ID, item.Name))
	respondJSON(w, http.StatusOK, map[string]any{
		"ContentSourceId": strconv.FormatInt(item.ID, 10),
		"IsUpdateSuccess": true,
		"AdvancedInfo":    []any{},
		"Status": map[string]any{
			"Code":        3000,
			"Description": "RefreshIndex successfully completed.",
			"Exception":   "",
		},
		"TrackingId": trackingID,
	})
}

type azureContentModeratorImageListPayload struct {
	Name        string
	Description string
	Metadata    map[string]string
}

func parseAzureContentModeratorImageListPayload(w http.ResponseWriter, r *http.Request, path string) (azureContentModeratorImageListPayload, bool) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "content type must be application/json", "provider": providerAzure, "path": path})
		return azureContentModeratorImageListPayload{}, false
	}

	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body", "provider": providerAzure, "path": path})
		return azureContentModeratorImageListPayload{}, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body is required", "provider": providerAzure, "path": path})
		return azureContentModeratorImageListPayload{}, false
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body must be valid JSON", "provider": providerAzure, "path": path})
		return azureContentModeratorImageListPayload{}, false
	}
	name := strings.TrimSpace(azureContentModeratorString(payload["Name"]))
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "Name is required", "provider": providerAzure, "path": path})
		return azureContentModeratorImageListPayload{}, false
	}
	description := strings.TrimSpace(azureContentModeratorString(payload["Description"]))
	metadata := map[string]string{}
	if rawMetadata, ok := payload["Metadata"].(map[string]any); ok {
		for key, value := range rawMetadata {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			metadata[trimmed] = strings.TrimSpace(fmt.Sprint(value))
		}
	}

	return azureContentModeratorImageListPayload{
		Name:        name,
		Description: description,
		Metadata:    metadata,
	}, true
}

func parseAzureContentModeratorListID(w http.ResponseWriter, path, raw string) (int64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "listId is required", "provider": providerAzure, "path": path})
		return 0, false
	}
	listID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || listID <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "listId must be a positive integer", "provider": providerAzure, "path": path})
		return 0, false
	}
	return listID, true
}

func azureContentModeratorImageListResponse(item *azureContentModeratorImageList) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	metadata := map[string]string{}
	for key, value := range item.Metadata {
		metadata[key] = value
	}
	return map[string]any{
		"Id":          item.ID,
		"Name":        item.Name,
		"Description": item.Description,
		"Metadata":    metadata,
	}
}
