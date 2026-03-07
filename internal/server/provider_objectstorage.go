package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultOCINamespace = "stackyard"

type providerBucket struct {
	Name              string
	CreatedAt         time.Time
	ProjectID         string
	Location          string
	StorageClass      string
	Metageneration    int64
	ETag              string
	VersioningEnabled bool
	Objects           map[string]providerObject
	ACL               map[string]string
	DefaultObjectACL  map[string]string
	IAMPolicy         map[string]any
	Notifications     map[string]map[string]any
}

type providerObject struct {
	Name           string
	UpdatedAt      time.Time
	CreatedAt      time.Time
	ContentType    string
	Metadata       map[string]string
	Generation     int64
	Metageneration int64
	ETag           string
	Deleted        bool
	ACL            map[string]string
	Data           []byte
}

func (s *Server) handleGCPObjectStorageRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.handleGCPStorageRouter(w, r)
}

func (s *Server) handleGCPCreateBucket(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string `json:"name"`
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "invalid JSON payload"})
			return
		}
	}
	bucket := strings.TrimSpace(payload.Name)
	if bucket == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "bucket name is required"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, exists := s.gcpStorageBuckets[bucket]; exists {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "Conflict", "message": "bucket already exists"})
		return
	}
	createdAt := time.Now().UTC()
	s.gcpStorageBuckets[bucket] = &providerBucket{
		Name:      bucket,
		CreatedAt: createdAt,
		Objects:   map[string]providerObject{},
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":        "storage#bucket",
		"id":          bucket,
		"name":        bucket,
		"timeCreated": createdAt.Format(time.RFC3339),
	})
}

func (s *Server) handleGCPListBuckets(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	keys := make([]string, 0, len(s.gcpStorageBuckets))
	for key := range s.gcpStorageBuckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		b := s.gcpStorageBuckets[key]
		items = append(items, map[string]any{
			"kind":        "storage#bucket",
			"id":          b.Name,
			"name":        b.Name,
			"timeCreated": b.CreatedAt.Format(time.RFC3339),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":  "storage#buckets",
		"items": items,
	})
}

func (s *Server) handleGCPUploadObject(w http.ResponseWriter, r *http.Request, path string) {
	const prefix = "/gcp/upload/storage/v1/b/"
	remainder := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(remainder, "/o") {
		respondProviderNotImplemented(w, providerGCP, path)
		return
	}
	bucketName := strings.TrimSuffix(remainder, "/o")
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "bucket is required"})
		return
	}
	objectName := strings.TrimSpace(r.URL.Query().Get("name"))
	if objectName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "object name is required"})
		return
	}
	payload, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "bucket not found"})
		return
	}
	updated := time.Now().UTC()
	bucket.Objects[objectName] = providerObject{Name: objectName, UpdatedAt: updated, Data: append([]byte(nil), payload...)}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":    "storage#object",
		"bucket":  bucketName,
		"name":    objectName,
		"size":    fmt.Sprintf("%d", len(payload)),
		"updated": updated.Format(time.RFC3339),
	})
}

func (s *Server) handleGCPBucketPath(w http.ResponseWriter, r *http.Request, path string) {
	const prefix = "/gcp/storage/v1/b/"
	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(remainder, "/", 3)
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "o" {
		respondProviderNotImplemented(w, providerGCP, path)
		return
	}
	bucket := strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		if r.Method == http.MethodGet {
			s.handleGCPListObjects(w, bucket)
			return
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return
	}
	objectName, err := url.PathUnescape(parts[2])
	if err != nil || strings.TrimSpace(objectName) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "invalid object path"})
		return
	}
	if r.Method == http.MethodGet {
		s.handleGCPGetObject(w, r, bucket, objectName)
		return
	}
	respondProviderNotImplemented(w, providerGCP, path)
}

func (s *Server) handleGCPListObjects(w http.ResponseWriter, bucketName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "bucket not found"})
		return
	}
	keys := make([]string, 0, len(bucket.Objects))
	for key := range bucket.Objects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		obj := bucket.Objects[key]
		items = append(items, map[string]any{
			"kind":    "storage#object",
			"bucket":  bucketName,
			"name":    obj.Name,
			"size":    fmt.Sprintf("%d", len(obj.Data)),
			"updated": obj.UpdatedAt.Format(time.RFC3339),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":  "storage#objects",
		"items": items,
	})
}

func (s *Server) handleGCPGetObject(w http.ResponseWriter, r *http.Request, bucketName, objectName string) {
	s.providerStorageMu.Lock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "bucket not found"})
		return
	}
	obj, ok := bucket.Objects[objectName]
	s.providerStorageMu.Unlock()
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "NotFound", "message": "object not found"})
		return
	}

	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("alt")), "media") {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.Data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.Data)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":    "storage#object",
		"bucket":  bucketName,
		"name":    objectName,
		"size":    fmt.Sprintf("%d", len(obj.Data)),
		"updated": obj.UpdatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleAzureBlobRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(rawRequestPath(r), "/azure/")
	if path == rawRequestPath(r) {
		return false
	}
	segments := splitPathSegments(path)
	if len(segments) == 0 {
		respondProviderNotImplemented(w, providerAzure, rawRequestPath(r))
		return true
	}
	account := segments[0]
	query := r.URL.Query()

	if len(segments) == 1 && r.Method == http.MethodGet && strings.EqualFold(query.Get("comp"), "list") {
		s.handleAzureListContainers(w, account)
		return true
	}
	if len(segments) == 2 && strings.EqualFold(query.Get("restype"), "container") {
		container := segments[1]
		switch r.Method {
		case http.MethodPut:
			s.handleAzureCreateContainer(w, account, container)
		case http.MethodGet:
			s.handleAzureGetContainer(w, account, container)
		case http.MethodHead:
			s.handleAzureGetContainer(w, account, container)
		default:
			respondProviderNotImplemented(w, providerAzure, rawRequestPath(r))
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
			s.handleAzureGetBlob(w, account, container, blobName, true)
		case http.MethodHead:
			s.handleAzureGetBlob(w, account, container, blobName, false)
		default:
			respondProviderNotImplemented(w, providerAzure, rawRequestPath(r))
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

func (s *Server) handleAzureListContainers(w http.ResponseWriter, account string) {
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

	resp := azureEnumerationResults{
		Containers: azureContainerList{Container: make([]azureContainer, 0, len(keys))},
		NextMarker: "",
	}
	for _, key := range keys {
		resp.Containers.Container = append(resp.Containers.Container, azureContainer{Name: key})
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
	b.Objects[blob] = providerObject{Name: blob, UpdatedAt: time.Now().UTC(), Data: append([]byte(nil), body...)}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleAzureGetBlob(w http.ResponseWriter, account, container, blob string, includeBody bool) {
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
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.Data)))
	w.WriteHeader(http.StatusOK)
	if includeBody {
		_, _ = w.Write(obj.Data)
	}
}

func (s *Server) handleOCIObjectStorageRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path == "/oci/n" && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, map[string]any{"value": defaultOCINamespace})
		return true
	}
	if !strings.HasPrefix(path, "/oci/n/") {
		return false
	}
	segments := splitPathSegments(strings.TrimPrefix(path, "/oci/"))
	if len(segments) < 3 || segments[0] != "n" {
		return false
	}
	namespace := segments[1]
	if segments[2] != "b" {
		return false
	}

	if len(segments) == 3 && r.Method == http.MethodGet {
		s.handleOCIListBuckets(w, namespace)
		return true
	}
	if len(segments) == 4 && r.Method == http.MethodPut {
		s.handleOCICreateBucket(w, namespace, segments[3])
		return true
	}
	if len(segments) >= 6 && segments[4] == "o" {
		objectName := strings.Join(segments[5:], "/")
		switch r.Method {
		case http.MethodPut:
			s.handleOCIPutObject(w, r, namespace, segments[3], objectName)
		case http.MethodGet:
			s.handleOCIGetObject(w, namespace, segments[3], objectName)
		default:
			respondProviderNotImplemented(w, providerOCI, path)
		}
		return true
	}
	return false
}

func (s *Server) handleOCIListBuckets(w http.ResponseWriter, namespace string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	buckets := s.ociStorageNamespaces[namespace]
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]any{"name": key, "namespace": namespace})
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleOCICreateBucket(w http.ResponseWriter, namespace, bucket string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	buckets := s.ociStorageNamespaces[namespace]
	if buckets == nil {
		buckets = map[string]*providerBucket{}
		s.ociStorageNamespaces[namespace] = buckets
	}
	if _, exists := buckets[bucket]; exists {
		respondJSON(w, http.StatusConflict, map[string]any{"code": "BucketAlreadyExists", "message": "bucket already exists"})
		return
	}
	buckets[bucket] = &providerBucket{Name: bucket, CreatedAt: time.Now().UTC(), Objects: map[string]providerObject{}}
	respondJSON(w, http.StatusOK, map[string]any{"name": bucket, "namespace": namespace})
}

func (s *Server) handleOCIPutObject(w http.ResponseWriter, r *http.Request, namespace, bucket, object string) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"code": "InvalidRequest", "message": "unable to read request body"})
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	buckets := s.ociStorageNamespaces[namespace]
	if buckets == nil {
		buckets = map[string]*providerBucket{}
		s.ociStorageNamespaces[namespace] = buckets
	}
	b := buckets[bucket]
	if b == nil {
		b = &providerBucket{Name: bucket, CreatedAt: time.Now().UTC(), Objects: map[string]providerObject{}}
		buckets[bucket] = b
	}
	b.Objects[object] = providerObject{Name: object, UpdatedAt: time.Now().UTC(), Data: append([]byte(nil), body...)}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleOCIGetObject(w http.ResponseWriter, namespace, bucket, object string) {
	s.providerStorageMu.Lock()
	buckets := s.ociStorageNamespaces[namespace]
	var obj providerObject
	var ok bool
	if buckets != nil {
		if b := buckets[bucket]; b != nil {
			obj, ok = b.Objects[object]
		}
	}
	s.providerStorageMu.Unlock()
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"code": "ObjectNotFound", "message": "object not found"})
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(obj.Data)
}
