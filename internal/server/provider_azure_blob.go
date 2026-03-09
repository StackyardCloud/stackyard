package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleAzureBlobRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(rawRequestPath(r), "/azure/")
	if path == rawRequestPath(r) {
		return false
	}
	segments := splitPathSegments(path)
	if len(segments) == 0 {
		respondAzureImplemented(w, rawRequestPath(r))
		return true
	}
	account := segments[0]
	query := r.URL.Query()

	if len(segments) == 1 && r.Method == http.MethodGet && strings.EqualFold(query.Get("comp"), "list") {
		s.handleAzureListContainers(w, r, account)
		return true
	}
	if len(segments) == 2 && strings.EqualFold(query.Get("restype"), "container") {
		container := segments[1]
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(query.Get("comp"), "list") {
			s.handleAzureListBlobs(w, r, account, container)
			return true
		}
		switch r.Method {
		case http.MethodPut:
			s.handleAzureCreateContainer(w, account, container)
		case http.MethodGet:
			s.handleAzureGetContainer(w, account, container)
		case http.MethodHead:
			s.handleAzureGetContainer(w, account, container)
		default:
			respondAzureImplemented(w, rawRequestPath(r))
		}
		return true
	}
	if len(segments) >= 3 {
		container := segments[1]
		blobName := strings.Join(segments[2:], "/")
		switch r.Method {
		case http.MethodPut:
			s.handleAzurePutBlob(w, r, account, container, blobName)
		case http.MethodGet:
			s.handleAzureGetBlob(w, r, account, container, blobName, true)
		case http.MethodHead:
			s.handleAzureGetBlob(w, r, account, container, blobName, false)
		default:
			respondAzureImplemented(w, rawRequestPath(r))
		}
		return true
	}
	return false
}

func (s *Server) handleAzureCreateContainer(w http.ResponseWriter, account, container string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	containers := s.azureStorageAccounts[account]
	if containers == nil {
		containers = map[string]*providerBucket{}
		s.azureStorageAccounts[account] = containers
	}
	if _, exists := containers[container]; exists {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "ContainerAlreadyExists", "message": "container already exists"})
		return
	}
	containers[container] = &providerBucket{
		Name:      container,
		CreatedAt: time.Now().UTC(),
		Objects:   map[string]providerObject{},
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleAzureGetContainer(w http.ResponseWriter, account, container string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	containers := s.azureStorageAccounts[account]
	if containers == nil || containers[container] == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ContainerNotFound", "message": "container not found"})
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureListContainers(w http.ResponseWriter, r *http.Request, account string) {
	type azureContainer struct {
		Name string `xml:"Name"`
	}
	type azureContainerList struct {
		Container []azureContainer `xml:"Container"`
	}
	type azureEnumerationResults struct {
		XMLName    xml.Name           `xml:"EnumerationResults"`
		Containers azureContainerList `xml:"Containers"`
		NextMarker string             `xml:"NextMarker"`
	}

	s.providerStorageMu.Lock()
	containers := s.azureStorageAccounts[account]
	keys := make([]string, 0, len(containers))
	for key := range containers {
		keys = append(keys, key)
	}
	s.providerStorageMu.Unlock()
	sort.Strings(keys)

	marker := strings.TrimSpace(r.URL.Query().Get("marker"))
	maxResults, err := parseAzureMaxResults(r.URL.Query().Get("maxresults"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidQueryParameterValue", "message": err.Error()})
		return
	}
	visible, nextMarker := azurePaginate(keys, marker, maxResults)

	resp := azureEnumerationResults{
		Containers: azureContainerList{Container: make([]azureContainer, 0, len(visible))},
		NextMarker: nextMarker,
	}
	for _, key := range visible {
		resp.Containers.Container = append(resp.Containers.Container, azureContainer{Name: key})
	}
	payload, _ := xml.Marshal(resp)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (s *Server) handleAzureListBlobs(w http.ResponseWriter, r *http.Request, account, container string) {
	type azureBlob struct {
		Name string `xml:"Name"`
	}
	type azureBlobList struct {
		Blob []azureBlob `xml:"Blob"`
	}
	type azureEnumerationResults struct {
		XMLName    xml.Name      `xml:"EnumerationResults"`
		Blobs      azureBlobList `xml:"Blobs"`
		NextMarker string        `xml:"NextMarker"`
	}

	s.providerStorageMu.Lock()
	containers := s.azureStorageAccounts[account]
	bucket := containers[container]
	if bucket == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ContainerNotFound", "message": "container not found"})
		return
	}
	keys := make([]string, 0, len(bucket.Objects))
	for key := range bucket.Objects {
		keys = append(keys, key)
	}
	s.providerStorageMu.Unlock()
	sort.Strings(keys)

	marker := strings.TrimSpace(r.URL.Query().Get("marker"))
	maxResults, err := parseAzureMaxResults(r.URL.Query().Get("maxresults"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidQueryParameterValue", "message": err.Error()})
		return
	}
	visible, nextMarker := azurePaginate(keys, marker, maxResults)
	resp := azureEnumerationResults{
		Blobs:      azureBlobList{Blob: make([]azureBlob, 0, len(visible))},
		NextMarker: nextMarker,
	}
	for _, key := range visible {
		resp.Blobs.Blob = append(resp.Blobs.Blob, azureBlob{Name: key})
	}
	payload, _ := xml.Marshal(resp)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (s *Server) handleAzurePutBlob(w http.ResponseWriter, r *http.Request, account, container, blob string) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	containers := s.azureStorageAccounts[account]
	if containers == nil {
		containers = map[string]*providerBucket{}
		s.azureStorageAccounts[account] = containers
	}
	b := containers[container]
	if b == nil {
		b = &providerBucket{Name: container, CreatedAt: time.Now().UTC(), Objects: map[string]providerObject{}}
		containers[container] = b
	}
	existing := b.Objects[blob]
	created := existing.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	updated := time.Now().UTC()
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	metadata := map[string]string{}
	for key, values := range r.Header {
		if !strings.HasPrefix(strings.ToLower(key), "x-ms-meta-") || len(values) == 0 {
			continue
		}
		metaKey := strings.TrimSpace(key[len("x-ms-meta-"):])
		if metaKey == "" {
			continue
		}
		metadata[metaKey] = strings.TrimSpace(values[0])
	}
	etag := azureBlobETag(account, container, blob, body, updated)
	b.Objects[blob] = providerObject{
		Name:        blob,
		CreatedAt:   created,
		UpdatedAt:   updated,
		ContentType: contentType,
		Metadata:    metadata,
		ETag:        etag,
		Data:        append([]byte(nil), body...),
	}
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleAzureGetBlob(w http.ResponseWriter, r *http.Request, account, container, blob string, includeBody bool) {
	s.providerStorageMu.Lock()
	containers := s.azureStorageAccounts[account]
	var obj providerObject
	var ok bool
	if containers != nil {
		if b := containers[container]; b != nil {
			obj, ok = b.Objects[blob]
		}
	}
	s.providerStorageMu.Unlock()
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "BlobNotFound", "message": "blob not found"})
		return
	}
	if !azureIfMatchSatisfied(strings.TrimSpace(r.Header.Get("If-Match")), obj.ETag) {
		respondJSON(w, http.StatusPreconditionFailed, map[string]any{"error": "ConditionNotMet", "message": "If-Match condition failed"})
		return
	}
	if azureIfNoneMatchSatisfied(strings.TrimSpace(r.Header.Get("If-None-Match")), obj.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	contentType := strings.TrimSpace(obj.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.Data)))
	if strings.TrimSpace(obj.ETag) != "" {
		w.Header().Set("ETag", obj.ETag)
	}
	for key, value := range obj.Metadata {
		if strings.TrimSpace(key) == "" {
			continue
		}
		w.Header().Set("x-ms-meta-"+key, value)
	}
	w.WriteHeader(http.StatusOK)
	if includeBody {
		_, _ = w.Write(obj.Data)
	}
}

func parseAzureMaxResults(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("invalid maxresults %q", raw)
	}
	return count, nil
}

func azurePaginate(keys []string, marker string, maxResults int) ([]string, string) {
	start := 0
	if marker != "" {
		for i, key := range keys {
			if key > marker {
				start = i
				break
			}
			start = len(keys)
		}
	}
	if start >= len(keys) {
		return []string{}, ""
	}

	end := len(keys)
	if maxResults > 0 && start+maxResults < end {
		end = start + maxResults
	}
	nextMarker := ""
	if end < len(keys) && end > start {
		nextMarker = keys[end-1]
	}
	out := make([]string, end-start)
	copy(out, keys[start:end])
	return out, nextMarker
}

func azureBlobETag(account, container, blob string, body []byte, now time.Time) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		account,
		container,
		blob,
		fmt.Sprintf("%d", len(body)),
		fmt.Sprintf("%d", now.UnixNano()),
	}, "|")))
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}

func azureIfMatchSatisfied(ifMatch, etag string) bool {
	if ifMatch == "" || ifMatch == "*" {
		return true
	}
	return strings.TrimSpace(ifMatch) == strings.TrimSpace(etag)
}

func azureIfNoneMatchSatisfied(ifNoneMatch, etag string) bool {
	value := strings.TrimSpace(ifNoneMatch)
	if value == "" {
		return false
	}
	if value == "*" {
		return true
	}
	return value == strings.TrimSpace(etag)
}
