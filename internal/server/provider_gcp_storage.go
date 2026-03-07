package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var gcpStorageReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type gcpStorageHMACKey struct {
	AccessID            string
	ProjectID           string
	ServiceAccountEmail string
	Secret              string
	State               string
	CreateTime          time.Time
	UpdateTime          time.Time
	ETag                string
}

func (s *Server) handleGCPStorageRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_storage(w, r) {
		return true
	}

	path := normalizeGCPStoragePath(rawRequestPath(r))
	if !isGCPStoragePath(path, hasGCPStorageHint(r)) {
		return false
	}

	switch {
	case path == "/gcp/storage/v1/b":
		return s.handleGCPStorageBucketsCollection(w, r, path)
	case strings.HasPrefix(path, "/gcp/storage/v1/b/"):
		return s.handleGCPStorageBucketSubresource(w, r, path)
	case strings.HasPrefix(path, "/gcp/upload/storage/v1/b/"):
		return s.handleGCPStorageUploadObject(w, r, path)
	case strings.HasPrefix(path, "/gcp/download/storage/v1/b/"):
		return s.handleGCPStorageDownloadObject(w, r, path)
	case strings.HasPrefix(path, "/gcp/storage/v1/projects/"):
		return s.handleGCPStorageProjectsPath(w, r, path)
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPStoragePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPStorageHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "storage", "storage-apiv1", "storage_apiv1", "gcs", "cloud-storage", "cloud_storage", "gcp-cloud-storage":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-storage-apiv1") || strings.Contains(ua, "cloud.google.com/go/storage")
}

func isGCPStoragePath(path string, includeHint bool) bool {
	switch {
	case strings.HasPrefix(path, "/gcp/storage/v1/"):
		return true
	case strings.HasPrefix(path, "/gcp/upload/storage/v1/"):
		return true
	case strings.HasPrefix(path, "/gcp/download/storage/v1/"):
		return true
	default:
		return includeHint && strings.Contains(path, "/storage")
	}
}

func (s *Server) handleGCPStorageBucketsCollection(w http.ResponseWriter, r *http.Request, path string) bool {
	switch r.Method {
	case http.MethodPost:
		return s.handleGCPStorageCreateBucket(w, r, path)
	case http.MethodGet:
		return s.handleGCPStorageListBuckets(w, r, path)
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func (s *Server) handleGCPStorageBucketSubresource(w http.ResponseWriter, r *http.Request, path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/gcp/storage/v1/b/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		respondGCPStorageInvalidArgument(w, path, "bucket is required")
		return true
	}
	bucketName, err := url.PathUnescape(strings.TrimSpace(parts[0]))
	if err != nil || !isGCPStorageBucketName(bucketName) {
		respondGCPStorageInvalidArgument(w, path, "bucket name is invalid")
		return true
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return s.handleGCPStorageGetBucket(w, path, bucketName)
		case http.MethodPatch:
			return s.handleGCPStorageUpdateBucket(w, r, path, bucketName)
		case http.MethodDelete:
			return s.handleGCPStorageDeleteBucket(w, path, bucketName)
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	switch parts[1] {
	case "o":
		return s.handleGCPStorageObjectsPath(w, r, path, bucketName, parts[2:])
	case "acl":
		return s.handleGCPStorageBucketACLPath(w, r, path, bucketName, parts[2:])
	case "iam":
		return s.handleGCPStorageBucketIAMPath(w, r, path, bucketName, parts[2:])
	case "notificationConfigs":
		return s.handleGCPStorageNotificationsPath(w, r, path, bucketName, parts[2:])
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func (s *Server) handleGCPStorageProjectsPath(w http.ResponseWriter, r *http.Request, path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/gcp/storage/v1/projects/"), "/")
	if len(parts) < 2 {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	projectID, err := url.PathUnescape(strings.TrimSpace(parts[0]))
	if err != nil || strings.TrimSpace(projectID) == "" {
		respondGCPStorageInvalidArgument(w, path, "project is required")
		return true
	}
	switch parts[1] {
	case "serviceAccount":
		if r.Method != http.MethodGet {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"kind":          "storage#serviceAccount",
			"email_address": gcpStorageServiceAccountEmail(projectID),
			"emailAddress":  gcpStorageServiceAccountEmail(projectID),
			"projectId":     projectID,
		})
		return true
	case "hmacKeys":
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodPost:
				return s.handleGCPStorageCreateHMACKey(w, r, path, projectID)
			case http.MethodGet:
				return s.handleGCPStorageListHMACKeys(w, r, path, projectID)
			default:
				respondProviderNotImplemented(w, providerGCP, path)
				return true
			}
		}
		accessID, err := url.PathUnescape(strings.TrimSpace(parts[2]))
		if err != nil || strings.TrimSpace(accessID) == "" {
			respondGCPStorageInvalidArgument(w, path, "accessId is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			return s.handleGCPStorageGetHMACKey(w, path, projectID, accessID)
		case http.MethodPut:
			return s.handleGCPStorageUpdateHMACKey(w, r, path, projectID, accessID)
		case http.MethodDelete:
			return s.handleGCPStorageDeleteHMACKey(w, path, projectID, accessID)
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func (s *Server) handleGCPStorageUploadObject(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	remainder := strings.TrimPrefix(path, "/gcp/upload/storage/v1/b/")
	if !strings.HasSuffix(remainder, "/o") {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	bucketName := strings.TrimSuffix(remainder, "/o")
	bucketName = strings.TrimSpace(bucketName)
	if !isGCPStorageBucketName(bucketName) {
		respondGCPStorageInvalidArgument(w, path, "bucket name is invalid")
		return true
	}
	uploadType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("uploadType")))
	if uploadType == "" {
		respondGCPStorageInvalidArgument(w, path, "uploadType is required")
		return true
	}
	switch uploadType {
	case "media", "multipart", "resumable":
	default:
		respondGCPStorageInvalidArgument(w, path, "uploadType is invalid")
		return true
	}

	objectName := strings.TrimSpace(r.URL.Query().Get("name"))
	if objectName == "" {
		respondGCPStorageInvalidArgument(w, path, "object name is required")
		return true
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		respondGCPStorageInvalidArgument(w, path, "request body must be readable")
		return true
	}
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	if bucket.Objects == nil {
		bucket.Objects = map[string]providerObject{}
	}
	now := time.Now().UTC()
	generation := s.nextGCPStorageGenerationLocked()
	existing := bucket.Objects[objectName]
	metadata := map[string]string{}
	for k, v := range existing.Metadata {
		metadata[k] = v
	}
	obj := providerObject{
		Name:           objectName,
		CreatedAt:      now,
		UpdatedAt:      now,
		ContentType:    contentType,
		Metadata:       metadata,
		Generation:     generation,
		Metageneration: 1,
		ETag:           gcpStorageETag(bucketName + "/" + objectName + "/" + strconv.FormatInt(generation, 10)),
		Deleted:        false,
		ACL:            existing.ACL,
		Data:           append([]byte(nil), payload...),
	}
	if existing.Name != "" {
		obj.CreatedAt = existing.CreatedAt
	}
	if len(obj.ACL) == 0 {
		obj.ACL = map[string]string{
			"project-owners-stackyard": "OWNER",
		}
	}
	bucket.Objects[objectName] = obj
	bucket.Metageneration++
	bucket.ETag = gcpStorageETag(bucket.Name + ":" + strconv.FormatInt(bucket.Metageneration, 10))

	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, obj))
	return true
}

func (s *Server) handleGCPStorageDownloadObject(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodGet {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	remainder := strings.TrimPrefix(path, "/gcp/download/storage/v1/b/")
	parts := strings.SplitN(remainder, "/o/", 2)
	if len(parts) != 2 {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	bucketName := strings.TrimSpace(parts[0])
	objectName, err := url.PathUnescape(strings.TrimSpace(parts[1]))
	if !isGCPStorageBucketName(bucketName) || err != nil || strings.TrimSpace(objectName) == "" {
		respondGCPStorageInvalidArgument(w, path, "invalid bucket or object path")
		return true
	}

	s.providerStorageMu.Lock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		s.providerStorageMu.Unlock()
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, ok := bucket.Objects[objectName]
	s.providerStorageMu.Unlock()
	if !ok || obj.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(obj.Data)))
	w.Header().Set("ETag", obj.ETag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(obj.Data)
	return true
}

func (s *Server) handleGCPStorageCreateBucket(w http.ResponseWriter, r *http.Request, path string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	bucketName := strings.TrimSpace(gcpStorageString(body, "name"))
	if bucketName == "" {
		respondGCPStorageInvalidArgument(w, path, "bucket name is required")
		return true
	}
	if !isGCPStorageBucketName(bucketName) {
		respondGCPStorageInvalidArgument(w, path, "bucket name is invalid")
		return true
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project"))
	if projectID == "" {
		projectID = "stackyard"
	}
	location := strings.TrimSpace(gcpStorageString(body, "location"))
	if location == "" {
		location = "US"
	}
	storageClass := strings.ToUpper(strings.TrimSpace(gcpStorageString(body, "storageClass")))
	if storageClass == "" {
		storageClass = "STANDARD"
	}
	versioningEnabled := false
	if versioning, ok := body["versioning"].(map[string]any); ok {
		versioningEnabled = gcpStorageBool(versioning, "enabled")
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, exists := s.gcpStorageBuckets[bucketName]; exists {
		respondGCPStorageAlreadyExists(w, path, "bucket already exists")
		return true
	}

	now := time.Now().UTC()
	bucket := &providerBucket{
		Name:              bucketName,
		CreatedAt:         now,
		ProjectID:         projectID,
		Location:          location,
		StorageClass:      storageClass,
		Metageneration:    1,
		ETag:              gcpStorageETag(bucketName + ":1"),
		VersioningEnabled: versioningEnabled,
		Objects:           map[string]providerObject{},
		ACL: map[string]string{
			"project-owners-stackyard": "OWNER",
		},
		DefaultObjectACL: map[string]string{
			"project-owners-stackyard": "OWNER",
		},
		Notifications: map[string]map[string]any{},
	}
	bucket.IAMPolicy = gcpStorageDefaultIAMPolicy(bucketName)
	s.gcpStorageBuckets[bucketName] = bucket

	respondJSON(w, http.StatusOK, gcpStorageBucketResponse(bucket))
	return true
}

func (s *Server) handleGCPStorageListBuckets(w http.ResponseWriter, r *http.Request, path string) bool {
	pageSize, start, ok := parseGCPStoragePagination(w, r, path)
	if !ok {
		return true
	}
	projectFilter := strings.TrimSpace(r.URL.Query().Get("project"))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	names := make([]string, 0, len(s.gcpStorageBuckets))
	for name, bucket := range s.gcpStorageBuckets {
		if projectFilter != "" && bucket.ProjectID != projectFilter {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	if start > len(names) {
		respondGCPStorageInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(names)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(names) {
		nextPageToken = strconv.Itoa(end)
	}

	items := make([]map[string]any, 0, end-start)
	for _, name := range names[start:end] {
		items = append(items, gcpStorageBucketResponse(s.gcpStorageBuckets[name]))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":          "storage#buckets",
		"items":         items,
		"nextPageToken": nextPageToken,
	})
	return true
}

func (s *Server) handleGCPStorageGetBucket(w http.ResponseWriter, path, bucketName string) bool {
	s.providerStorageMu.Lock()
	bucket := s.gcpStorageBuckets[bucketName]
	s.providerStorageMu.Unlock()
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageBucketResponse(bucket))
	return true
}

func (s *Server) handleGCPStorageUpdateBucket(w http.ResponseWriter, r *http.Request, path, bucketName string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, true)
	if !ok {
		return true
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	if location := strings.TrimSpace(gcpStorageString(body, "location")); location != "" {
		bucket.Location = location
	}
	if storageClass := strings.ToUpper(strings.TrimSpace(gcpStorageString(body, "storageClass"))); storageClass != "" {
		bucket.StorageClass = storageClass
	}
	if versioning, ok := body["versioning"].(map[string]any); ok {
		bucket.VersioningEnabled = gcpStorageBool(versioning, "enabled")
	}
	bucket.Metageneration++
	bucket.ETag = gcpStorageETag(bucket.Name + ":" + strconv.FormatInt(bucket.Metageneration, 10))
	respondJSON(w, http.StatusOK, gcpStorageBucketResponse(bucket))
	return true
}

func (s *Server) handleGCPStorageDeleteBucket(w http.ResponseWriter, path, bucketName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	for _, obj := range bucket.Objects {
		if !obj.Deleted {
			respondGCPStorageFailedPrecondition(w, path, "bucket must be empty before delete")
			return true
		}
	}
	delete(s.gcpStorageBuckets, bucketName)
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func (s *Server) handleGCPStorageObjectsPath(w http.ResponseWriter, r *http.Request, path, bucketName string, tail []string) bool {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		return s.handleGCPStorageListObjects(w, r, path, bucketName)
	}

	objectName, err := url.PathUnescape(strings.TrimSpace(tail[0]))
	if err != nil || strings.TrimSpace(objectName) == "" {
		respondGCPStorageInvalidArgument(w, path, "object name is invalid")
		return true
	}
	remaining := tail[1:]

	if len(remaining) == 0 {
		switch r.Method {
		case http.MethodGet:
			return s.handleGCPStorageGetObject(w, r, path, bucketName, objectName)
		case http.MethodPatch:
			return s.handleGCPStorageUpdateObject(w, r, path, bucketName, objectName)
		case http.MethodDelete:
			return s.handleGCPStorageDeleteObject(w, path, bucketName, objectName)
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	switch remaining[0] {
	case "copyTo":
		if r.Method == http.MethodPost {
			return s.handleGCPStorageCopyObject(w, path, bucketName, objectName, remaining[1:])
		}
	case "rewriteTo":
		if r.Method == http.MethodPost {
			return s.handleGCPStorageRewriteObject(w, path, bucketName, objectName, remaining[1:])
		}
	case "compose":
		if r.Method == http.MethodPost {
			return s.handleGCPStorageComposeObject(w, r, path, bucketName, objectName)
		}
	case "moveTo":
		if r.Method == http.MethodPost {
			return s.handleGCPStorageMoveObject(w, path, bucketName, objectName, remaining[1:])
		}
	case "restore":
		if r.Method == http.MethodPost {
			return s.handleGCPStorageRestoreObject(w, path, bucketName, objectName)
		}
	case "acl":
		return s.handleGCPStorageObjectACLPath(w, r, path, bucketName, objectName, remaining[1:])
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func (s *Server) handleGCPStorageListObjects(w http.ResponseWriter, r *http.Request, path, bucketName string) bool {
	pageSize, start, ok := parseGCPStoragePagination(w, r, path)
	if !ok {
		return true
	}
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))

	s.providerStorageMu.Lock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		s.providerStorageMu.Unlock()
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	names := make([]string, 0, len(bucket.Objects))
	for name, obj := range bucket.Objects {
		if obj.Deleted {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if start > len(names) {
		s.providerStorageMu.Unlock()
		respondGCPStorageInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(names)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(names) {
		nextToken = strconv.Itoa(end)
	}
	items := make([]map[string]any, 0, end-start)
	for _, name := range names[start:end] {
		items = append(items, gcpStorageObjectResponse(bucketName, bucket.Objects[name]))
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"kind":          "storage#objects",
		"items":         items,
		"nextPageToken": nextToken,
	})
	return true
}

func (s *Server) handleGCPStorageGetObject(w http.ResponseWriter, r *http.Request, path, bucketName, objectName string) bool {
	s.providerStorageMu.Lock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		s.providerStorageMu.Unlock()
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, ok := bucket.Objects[objectName]
	s.providerStorageMu.Unlock()
	if !ok || obj.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}

	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("alt")), "media") {
		w.Header().Set("Content-Type", obj.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(obj.Data)))
		w.Header().Set("ETag", obj.ETag)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.Data)
		return true
	}

	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, obj))
	return true
}

func (s *Server) handleGCPStorageUpdateObject(w http.ResponseWriter, r *http.Request, path, bucketName, objectName string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, true)
	if !ok {
		return true
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, exists := bucket.Objects[objectName]
	if !exists || obj.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	if contentType := strings.TrimSpace(gcpStorageString(body, "contentType")); contentType != "" {
		obj.ContentType = contentType
	}
	if metadataAny, ok := body["metadata"].(map[string]any); ok {
		metadata := map[string]string{}
		for k, v := range metadataAny {
			if val := strings.TrimSpace(fmt.Sprintf("%v", v)); val != "" {
				metadata[k] = val
			}
		}
		obj.Metadata = metadata
	}
	obj.Metageneration++
	obj.UpdatedAt = time.Now().UTC()
	obj.ETag = gcpStorageETag(bucketName + "/" + objectName + ":" + strconv.FormatInt(obj.Metageneration, 10))
	bucket.Objects[objectName] = obj
	bucket.Metageneration++
	bucket.ETag = gcpStorageETag(bucket.Name + ":" + strconv.FormatInt(bucket.Metageneration, 10))

	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, obj))
	return true
}

func (s *Server) handleGCPStorageDeleteObject(w http.ResponseWriter, path, bucketName, objectName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, exists := bucket.Objects[objectName]
	if !exists {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	obj.Deleted = true
	obj.UpdatedAt = time.Now().UTC()
	obj.Metageneration++
	bucket.Objects[objectName] = obj
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func (s *Server) handleGCPStorageCopyObject(w http.ResponseWriter, path, srcBucket, srcObject string, tail []string) bool {
	if len(tail) != 4 || tail[0] != "b" || tail[2] != "o" {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	dstBucket := strings.TrimSpace(tail[1])
	dstObject, err := url.PathUnescape(strings.TrimSpace(tail[3]))
	if !isGCPStorageBucketName(dstBucket) || err != nil || strings.TrimSpace(dstObject) == "" {
		respondGCPStorageInvalidArgument(w, path, "destination bucket/object is invalid")
		return true
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	srcB := s.gcpStorageBuckets[srcBucket]
	dstB := s.gcpStorageBuckets[dstBucket]
	if srcB == nil || dstB == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	src, ok := srcB.Objects[srcObject]
	if !ok || src.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	generation := s.nextGCPStorageGenerationLocked()
	now := time.Now().UTC()
	copied := src
	copied.Name = dstObject
	copied.CreatedAt = now
	copied.UpdatedAt = now
	copied.Generation = generation
	copied.Metageneration = 1
	copied.ETag = gcpStorageETag(dstBucket + "/" + dstObject + "/" + strconv.FormatInt(generation, 10))
	copied.Deleted = false
	copied.Data = append([]byte(nil), src.Data...)
	dstB.Objects[dstObject] = copied
	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(dstBucket, copied))
	return true
}

func (s *Server) handleGCPStorageRewriteObject(w http.ResponseWriter, path, srcBucket, srcObject string, tail []string) bool {
	return s.handleGCPStorageCopyObject(w, path, srcBucket, srcObject, tail)
}

func (s *Server) handleGCPStorageComposeObject(w http.ResponseWriter, r *http.Request, path, bucketName, dstObject string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, true)
	if !ok {
		return true
	}
	sourceRaw, ok := body["sourceObjects"].([]any)
	if !ok || len(sourceRaw) == 0 {
		respondGCPStorageInvalidArgument(w, path, "sourceObjects is required")
		return true
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}

	var composed []byte
	for _, item := range sourceRaw {
		srcMap, _ := item.(map[string]any)
		name := strings.TrimSpace(gcpStorageString(srcMap, "name"))
		if name == "" {
			respondGCPStorageInvalidArgument(w, path, "sourceObjects.name is required")
			return true
		}
		obj, exists := bucket.Objects[name]
		if !exists || obj.Deleted {
			respondGCPStorageNotFound(w, path, "compose source object not found")
			return true
		}
		composed = append(composed, obj.Data...)
	}

	now := time.Now().UTC()
	generation := s.nextGCPStorageGenerationLocked()
	obj := providerObject{
		Name:           dstObject,
		CreatedAt:      now,
		UpdatedAt:      now,
		ContentType:    "application/octet-stream",
		Generation:     generation,
		Metageneration: 1,
		ETag:           gcpStorageETag(bucketName + "/" + dstObject + "/" + strconv.FormatInt(generation, 10)),
		ACL: map[string]string{
			"project-owners-stackyard": "OWNER",
		},
		Data: composed,
	}
	bucket.Objects[dstObject] = obj
	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, obj))
	return true
}

func (s *Server) handleGCPStorageMoveObject(w http.ResponseWriter, path, bucketName, srcObject string, tail []string) bool {
	if len(tail) != 2 || tail[0] != "o" {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	dstObject, err := url.PathUnescape(strings.TrimSpace(tail[1]))
	if err != nil || strings.TrimSpace(dstObject) == "" {
		respondGCPStorageInvalidArgument(w, path, "destination object is invalid")
		return true
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	src, exists := bucket.Objects[srcObject]
	if !exists || src.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	now := time.Now().UTC()
	generation := s.nextGCPStorageGenerationLocked()
	moved := src
	moved.Name = dstObject
	moved.CreatedAt = now
	moved.UpdatedAt = now
	moved.Generation = generation
	moved.Metageneration = 1
	moved.ETag = gcpStorageETag(bucketName + "/" + dstObject + "/" + strconv.FormatInt(generation, 10))
	moved.Deleted = false
	moved.Data = append([]byte(nil), src.Data...)
	bucket.Objects[dstObject] = moved
	src.Deleted = true
	src.UpdatedAt = now
	src.Metageneration++
	bucket.Objects[srcObject] = src
	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, moved))
	return true
}

func (s *Server) handleGCPStorageRestoreObject(w http.ResponseWriter, path, bucketName, objectName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, ok := bucket.Objects[objectName]
	if !ok {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	if !obj.Deleted {
		respondGCPStorageFailedPrecondition(w, path, "object is not soft-deleted")
		return true
	}
	obj.Deleted = false
	obj.Generation = s.nextGCPStorageGenerationLocked()
	obj.Metageneration = 1
	obj.UpdatedAt = time.Now().UTC()
	obj.ETag = gcpStorageETag(bucketName + "/" + objectName + "/" + strconv.FormatInt(obj.Generation, 10))
	bucket.Objects[objectName] = obj
	respondJSON(w, http.StatusOK, gcpStorageObjectResponse(bucketName, obj))
	return true
}

func (s *Server) handleGCPStorageBucketACLPath(w http.ResponseWriter, r *http.Request, path, bucketName string, tail []string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	if bucket.ACL == nil {
		bucket.ACL = map[string]string{}
	}

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, map[string]any{
				"kind":  "storage#bucketAccessControls",
				"items": gcpStorageACLItems(bucketName, "", bucket.ACL),
			})
			return true
		case http.MethodPost, http.MethodPut:
			body, ok := decodeGCPStorageJSONBody(w, r, path, true)
			if !ok {
				return true
			}
			entity := strings.TrimSpace(gcpStorageString(body, "entity"))
			role := strings.ToUpper(strings.TrimSpace(gcpStorageString(body, "role")))
			if entity == "" || role == "" {
				respondGCPStorageInvalidArgument(w, path, "entity and role are required")
				return true
			}
			bucket.ACL[entity] = role
			respondJSON(w, http.StatusOK, gcpStorageACLItem(bucketName, "", entity, role))
			return true
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	entity, err := url.PathUnescape(strings.TrimSpace(tail[0]))
	if err != nil || entity == "" {
		respondGCPStorageInvalidArgument(w, path, "entity is invalid")
		return true
	}
	if r.Method != http.MethodDelete {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	delete(bucket.ACL, entity)
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func (s *Server) handleGCPStorageObjectACLPath(w http.ResponseWriter, r *http.Request, path, bucketName, objectName string, tail []string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	obj, exists := bucket.Objects[objectName]
	if !exists || obj.Deleted {
		respondGCPStorageNotFound(w, path, "object not found")
		return true
	}
	if obj.ACL == nil {
		obj.ACL = map[string]string{}
	}

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, map[string]any{
				"kind":  "storage#objectAccessControls",
				"items": gcpStorageACLItems(bucketName, objectName, obj.ACL),
			})
			return true
		case http.MethodPost, http.MethodPut:
			body, ok := decodeGCPStorageJSONBody(w, r, path, true)
			if !ok {
				return true
			}
			entity := strings.TrimSpace(gcpStorageString(body, "entity"))
			role := strings.ToUpper(strings.TrimSpace(gcpStorageString(body, "role")))
			if entity == "" || role == "" {
				respondGCPStorageInvalidArgument(w, path, "entity and role are required")
				return true
			}
			obj.ACL[entity] = role
			obj.Metageneration++
			obj.UpdatedAt = time.Now().UTC()
			bucket.Objects[objectName] = obj
			respondJSON(w, http.StatusOK, gcpStorageACLItem(bucketName, objectName, entity, role))
			return true
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	entity, err := url.PathUnescape(strings.TrimSpace(tail[0]))
	if err != nil || entity == "" {
		respondGCPStorageInvalidArgument(w, path, "entity is invalid")
		return true
	}
	if r.Method != http.MethodDelete {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	delete(obj.ACL, entity)
	obj.Metageneration++
	obj.UpdatedAt = time.Now().UTC()
	bucket.Objects[objectName] = obj
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func (s *Server) handleGCPStorageBucketIAMPath(w http.ResponseWriter, r *http.Request, path, bucketName string, tail []string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	if bucket.IAMPolicy == nil {
		bucket.IAMPolicy = gcpStorageDefaultIAMPolicy(bucketName)
	}

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, bucket.IAMPolicy)
			return true
		case http.MethodPut, http.MethodPost:
			body, ok := decodeGCPStorageJSONBody(w, r, path, true)
			if !ok {
				return true
			}
			policy := body
			if wrapped, ok := body["policy"].(map[string]any); ok {
				policy = wrapped
			}
			if _, ok := policy["bindings"]; !ok {
				respondGCPStorageInvalidArgument(w, path, "policy.bindings is required")
				return true
			}
			if _, ok := policy["version"]; !ok {
				policy["version"] = float64(1)
			}
			if _, ok := policy["etag"]; !ok {
				policy["etag"] = gcpStorageETag(bucketName + ":iam")
			}
			bucket.IAMPolicy = policy
			respondJSON(w, http.StatusOK, policy)
			return true
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	if len(tail) == 1 && tail[0] == "testPermissions" {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		permissions := gcpStoragePermissionsFromRequest(r)
		if len(permissions) == 0 {
			respondGCPStorageInvalidArgument(w, path, "permissions are required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"kind":        "storage#testIamPermissionsResponse",
			"permissions": permissions,
		})
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func (s *Server) handleGCPStorageNotificationsPath(w http.ResponseWriter, r *http.Request, path, bucketName string, tail []string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	bucket := s.gcpStorageBuckets[bucketName]
	if bucket == nil {
		respondGCPStorageNotFound(w, path, "bucket not found")
		return true
	}
	if bucket.Notifications == nil {
		bucket.Notifications = map[string]map[string]any{}
	}

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			ids := make([]string, 0, len(bucket.Notifications))
			for id := range bucket.Notifications {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, bucket.Notifications[id])
			}
			respondJSON(w, http.StatusOK, map[string]any{
				"kind":  "storage#notifications",
				"items": items,
			})
			return true
		case http.MethodPost:
			body, ok := decodeGCPStorageJSONBody(w, r, path, true)
			if !ok {
				return true
			}
			topic := strings.TrimSpace(gcpStorageString(body, "topic"))
			if topic == "" {
				respondGCPStorageInvalidArgument(w, path, "topic is required")
				return true
			}
			id := strconv.FormatInt(s.nextGCPStorageGenerationLocked(), 10)
			notification := map[string]any{
				"kind":              "storage#notification",
				"id":                id,
				"topic":             topic,
				"event_types":       body["event_types"],
				"payload_format":    gcpStorageFallback(gcpStorageString(body, "payload_format"), "JSON_API_V1"),
				"custom_attributes": body["custom_attributes"],
			}
			bucket.Notifications[id] = notification
			respondJSON(w, http.StatusOK, notification)
			return true
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	if len(tail) == 1 && r.Method == http.MethodDelete {
		id := strings.TrimSpace(tail[0])
		delete(bucket.Notifications, id)
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func (s *Server) handleGCPStorageCreateHMACKey(w http.ResponseWriter, r *http.Request, path, projectID string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	serviceAccountEmail := strings.TrimSpace(gcpStorageString(body, "serviceAccountEmail"))
	if serviceAccountEmail == "" {
		serviceAccountEmail = gcpStorageServiceAccountEmail(projectID)
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.gcpStorageHMACKeys == nil {
		s.gcpStorageHMACKeys = map[string]map[string]*gcpStorageHMACKey{}
	}
	if s.gcpStorageHMACKeys[projectID] == nil {
		s.gcpStorageHMACKeys[projectID] = map[string]*gcpStorageHMACKey{}
	}
	s.gcpStorageNextHMACKeyID++
	id := fmt.Sprintf("GOOG1STACKYARD%08d", s.gcpStorageNextHMACKeyID)
	secret := fmt.Sprintf("stackyard-secret-%08d", s.gcpStorageNextHMACKeyID)
	now := time.Now().UTC()
	key := &gcpStorageHMACKey{
		AccessID:            id,
		ProjectID:           projectID,
		ServiceAccountEmail: serviceAccountEmail,
		Secret:              secret,
		State:               "ACTIVE",
		CreateTime:          now,
		UpdateTime:          now,
		ETag:                gcpStorageETag(projectID + ":" + id),
	}
	s.gcpStorageHMACKeys[projectID][id] = key

	respondJSON(w, http.StatusOK, map[string]any{
		"kind":     "storage#hmacKey",
		"secret":   secret,
		"metadata": gcpStorageHMACMetadata(key),
	})
	return true
}

func (s *Server) handleGCPStorageListHMACKeys(w http.ResponseWriter, r *http.Request, path, projectID string) bool {
	pageSize, start, ok := parseGCPStoragePagination(w, r, path)
	if !ok {
		return true
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	projectKeys := s.gcpStorageHMACKeys[projectID]
	ids := make([]string, 0, len(projectKeys))
	for id := range projectKeys {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if start > len(ids) {
		respondGCPStorageInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(ids)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(ids) {
		nextToken = strconv.Itoa(end)
	}
	items := make([]map[string]any, 0, end-start)
	for _, id := range ids[start:end] {
		items = append(items, gcpStorageHMACMetadata(projectKeys[id]))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"kind":          "storage#hmacKeys",
		"items":         items,
		"nextPageToken": nextToken,
	})
	return true
}

func (s *Server) handleGCPStorageGetHMACKey(w http.ResponseWriter, path, projectID, accessID string) bool {
	s.providerStorageMu.Lock()
	key := s.lookupGCPStorageHMACKeyLocked(projectID, accessID)
	s.providerStorageMu.Unlock()
	if key == nil {
		respondGCPStorageNotFound(w, path, "hmac key not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageHMACMetadata(key))
	return true
}

func (s *Server) handleGCPStorageUpdateHMACKey(w http.ResponseWriter, r *http.Request, path, projectID, accessID string) bool {
	body, ok := decodeGCPStorageJSONBody(w, r, path, true)
	if !ok {
		return true
	}
	newState := strings.ToUpper(strings.TrimSpace(gcpStorageString(body, "state")))
	if newState == "" {
		newState = strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
	}
	switch newState {
	case "ACTIVE", "INACTIVE", "DELETED":
	default:
		respondGCPStorageInvalidArgument(w, path, "state must be ACTIVE, INACTIVE, or DELETED")
		return true
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := s.lookupGCPStorageHMACKeyLocked(projectID, accessID)
	if key == nil {
		respondGCPStorageNotFound(w, path, "hmac key not found")
		return true
	}
	key.State = newState
	key.UpdateTime = time.Now().UTC()
	key.ETag = gcpStorageETag(projectID + ":" + accessID + ":" + newState)
	respondJSON(w, http.StatusOK, gcpStorageHMACMetadata(key))
	return true
}

func (s *Server) handleGCPStorageDeleteHMACKey(w http.ResponseWriter, path, projectID, accessID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := s.lookupGCPStorageHMACKeyLocked(projectID, accessID)
	if key == nil {
		respondGCPStorageNotFound(w, path, "hmac key not found")
		return true
	}
	key.State = "DELETED"
	key.UpdateTime = time.Now().UTC()
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func (s *Server) lookupGCPStorageHMACKeyLocked(projectID, accessID string) *gcpStorageHMACKey {
	if s.gcpStorageHMACKeys == nil {
		return nil
	}
	return s.gcpStorageHMACKeys[projectID][accessID]
}

func (s *Server) nextGCPStorageGenerationLocked() int64 {
	s.gcpStorageNextGeneration++
	if s.gcpStorageNextGeneration <= 0 {
		s.gcpStorageNextGeneration = 1
	}
	return s.gcpStorageNextGeneration
}

func parseGCPStoragePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("maxResults")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 1000 {
			respondGCPStorageInvalidArgument(w, path, "maxResults must be a non-negative integer <= 1000")
			return 0, 0, false
		}
		pageSize = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPStorageInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = n
	}
	return pageSize, start, true
}

func decodeGCPStorageJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPStorageInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		if required {
			respondGCPStorageInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPStorageInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func isGCPStorageBucketName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func gcpStorageString(body map[string]any, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func gcpStorageBool(body map[string]any, key string) bool {
	raw, ok := body[key]
	if !ok {
		return false
	}
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func gcpStorageBucketResponse(bucket *providerBucket) map[string]any {
	return map[string]any{
		"kind":           "storage#bucket",
		"id":             bucket.Name,
		"name":           bucket.Name,
		"projectNumber":  "1000000000000",
		"location":       gcpStorageFallback(bucket.Location, "US"),
		"storageClass":   gcpStorageFallback(bucket.StorageClass, "STANDARD"),
		"metageneration": strconv.FormatInt(bucket.Metageneration, 10),
		"etag":           bucket.ETag,
		"timeCreated":    bucket.CreatedAt.Format(time.RFC3339),
		"updated":        bucket.CreatedAt.Format(time.RFC3339),
		"versioning": map[string]any{
			"enabled": bucket.VersioningEnabled,
		},
	}
}

func gcpStorageObjectResponse(bucketName string, obj providerObject) map[string]any {
	hash := md5.Sum(obj.Data)
	size := strconv.Itoa(len(obj.Data))
	return map[string]any{
		"kind":           "storage#object",
		"id":             fmt.Sprintf("%s/%s/%d", bucketName, obj.Name, obj.Generation),
		"bucket":         bucketName,
		"name":           obj.Name,
		"size":           size,
		"contentType":    gcpStorageFallback(obj.ContentType, "application/octet-stream"),
		"generation":     strconv.FormatInt(obj.Generation, 10),
		"metageneration": strconv.FormatInt(obj.Metageneration, 10),
		"etag":           obj.ETag,
		"timeCreated":    obj.CreatedAt.Format(time.RFC3339),
		"updated":        obj.UpdatedAt.Format(time.RFC3339),
		"md5Hash":        base64.StdEncoding.EncodeToString(hash[:]),
		"metadata":       obj.Metadata,
	}
}

func gcpStorageACLItems(bucketName, objectName string, acl map[string]string) []map[string]any {
	entities := make([]string, 0, len(acl))
	for entity := range acl {
		entities = append(entities, entity)
	}
	sort.Strings(entities)
	items := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		items = append(items, gcpStorageACLItem(bucketName, objectName, entity, acl[entity]))
	}
	return items
}

func gcpStorageACLItem(bucketName, objectName, entity, role string) map[string]any {
	item := map[string]any{
		"kind":   "storage#bucketAccessControl",
		"entity": entity,
		"role":   role,
		"bucket": bucketName,
	}
	if objectName != "" {
		item["kind"] = "storage#objectAccessControl"
		item["object"] = objectName
	}
	return item
}

func gcpStoragePermissionsFromRequest(r *http.Request) []string {
	perms := make([]string, 0)
	for _, value := range r.URL.Query()["permissions"] {
		for _, token := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(token)
			if trimmed != "" {
				perms = append(perms, trimmed)
			}
		}
	}
	if len(perms) > 0 {
		return perms
	}
	if r.Method == http.MethodPost {
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if len(strings.TrimSpace(string(data))) > 0 {
			var body map[string]any
			if err := json.Unmarshal(data, &body); err == nil {
				if list, ok := body["permissions"].([]any); ok {
					for _, raw := range list {
						token := strings.TrimSpace(fmt.Sprintf("%v", raw))
						if token != "" {
							perms = append(perms, token)
						}
					}
				}
			}
		}
		r.Body = io.NopCloser(strings.NewReader(string(data)))
	}
	return perms
}

func gcpStorageDefaultIAMPolicy(bucketName string) map[string]any {
	return map[string]any{
		"version": float64(1),
		"etag":    gcpStorageETag(bucketName + ":iam"),
		"bindings": []map[string]any{
			{
				"role":    "roles/storage.objectViewer",
				"members": []string{"user:stackyard@example.com"},
			},
		},
	}
}

func gcpStorageServiceAccountEmail(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = "stackyard"
	}
	return fmt.Sprintf("service-%s@stackyard.iam.gserviceaccount.com", projectID)
}

func gcpStorageHMACMetadata(key *gcpStorageHMACKey) map[string]any {
	return map[string]any{
		"kind":                "storage#hmacKeyMetadata",
		"accessId":            key.AccessID,
		"projectId":           key.ProjectID,
		"state":               key.State,
		"etag":                key.ETag,
		"timeCreated":         key.CreateTime.Format(time.RFC3339),
		"updated":             key.UpdateTime.Format(time.RFC3339),
		"serviceAccountEmail": key.ServiceAccountEmail,
	}
}

func gcpStorageETag(seed string) string {
	sum := md5.Sum([]byte(seed))
	return fmt.Sprintf(`"%x"`, sum[:8])
}

func gcpStorageFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func respondGCPStorageInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPStorageError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPStorageFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPStorageError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPStorageNotFound(w http.ResponseWriter, path, message string) {
	respondGCPStorageError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPStorageAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPStorageError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPStorageError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_storage(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}
	path := normalizeGCPStoragePath(rawRequestPath(r))
	if !isGCPStoragePath(path, hasGCPStorageHint(r)) {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPStorageInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("exists") == "1" {
		respondGCPStorageAlreadyExists(w, path, "bucket already exists")
		return true
	}
	if r.URL.Query().Get("precondition") == "1" {
		respondGCPStorageFailedPrecondition(w, path, "precondition failed")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":      "projects/stackyard/buckets/stackyard-bucket",
		"bucket":    "stackyard-bucket",
		"object":    "orders/2026-01-01.json",
		"size":      "17",
		"service":   "storage",
		"provider":  providerGCP,
		"timestamp": gcpStorageReferenceTime.Format(time.RFC3339),
	})
	return true
}
