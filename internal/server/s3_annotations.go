package server

import (
	"crypto/md5" //nolint:gosec // S3 ETags are MD5 by definition.
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/services/s3"
)

// Bounds for caller-controlled annotation state.
const (
	s3AnnMaxPayload      = 1 << 20  // bytes per annotation
	s3AnnMaxPerObject    = 1000     // annotations per object
	s3AnnMaxTotalBytes   = 64 << 20 // payload bytes across the store
	s3AnnMaxNameLen      = 1024
	s3AnnDefaultPageSize = 1000
)

type s3Annotation struct {
	payload      []byte
	etag         string
	lastModified time.Time
}

// s3AnnotationStore keeps object annotations and per-bucket annotation table configuration.
// ponytail: annotations are keyed by bucket+key and ignore versionId; key them by version if
// versioned-annotation behaviour matters.
type s3AnnotationStore struct {
	mu         sync.Mutex
	objects    map[string]map[string]s3Annotation
	tableState map[string]string
	totalBytes int
}

func newS3AnnotationStore() *s3AnnotationStore {
	return &s3AnnotationStore{objects: map[string]map[string]s3Annotation{}, tableState: map[string]string{}}
}

func s3AnnObjectKey(bucket, key string) string { return bucket + "\x00" + key }

var errS3AnnLimit = errors.New("annotation limits exceeded")

func (st *s3AnnotationStore) put(bucket, key, name string, payload []byte) (s3Annotation, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ok := s3AnnObjectKey(bucket, key)
	anns := st.objects[ok]
	prev, exists := anns[name]
	delta := len(payload) - len(prev.payload)
	if (!exists && len(anns) >= s3AnnMaxPerObject) || st.totalBytes+delta > s3AnnMaxTotalBytes {
		return s3Annotation{}, errS3AnnLimit
	}
	sum := md5.Sum(payload) //nolint:gosec // ETag, not a security control.
	a := s3Annotation{payload: append([]byte(nil), payload...), etag: `"` + hex.EncodeToString(sum[:]) + `"`, lastModified: time.Now().UTC()}
	if anns == nil {
		anns = map[string]s3Annotation{}
		st.objects[ok] = anns
	}
	anns[name] = a
	st.totalBytes += delta
	return a, nil
}

func (st *s3AnnotationStore) get(bucket, key, name string) (s3Annotation, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, ok := st.objects[s3AnnObjectKey(bucket, key)][name]
	return a, ok
}

func (st *s3AnnotationStore) remove(bucket, key, name string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	anns := st.objects[s3AnnObjectKey(bucket, key)]
	a, ok := anns[name]
	if ok {
		delete(anns, name)
		st.totalBytes -= len(a.payload)
	}
	return ok
}

type s3AnnotationEntry struct {
	AnnotationName string `xml:"AnnotationName"`
	LastModified   string `xml:"LastModified"`
	ETag           string `xml:"ETag"`
	Size           int    `xml:"Size"`
}

// list returns entries with names after `after` matching prefix, sorted by name.
func (st *s3AnnotationStore) list(bucket, key, prefix, after string) []s3AnnotationEntry {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := []s3AnnotationEntry{}
	for name, a := range st.objects[s3AnnObjectKey(bucket, key)] {
		if strings.HasPrefix(name, prefix) && name > after {
			out = append(out, s3AnnotationEntry{name, a.lastModified.Format(time.RFC3339), a.etag, len(a.payload)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AnnotationName < out[j].AnnotationName })
	return out
}

func (st *s3AnnotationStore) setTableState(bucket, state string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.tableState[bucket] = state
}

// handleS3AnnotationRoutes serves object annotations (?annotation) and the bucket's annotation
// table configuration (?metadataAnnotationTable). It reports whether it handled the request.
func (s *Server) handleS3AnnotationRoutes(w http.ResponseWriter, r *http.Request) bool {
	q := r.URL.Query()
	if !q.Has("annotation") && !q.Has("metadataAnnotationTable") {
		return false
	}
	if !s.requireSigV4(w, r) {
		return true
	}
	bucket := bucketFromHost(r.Host)
	key := ""
	if bucket == "" {
		bucket, key = bucketAndKeyFromPath(r.URL.Path)
	} else {
		key = strings.TrimPrefix(r.URL.Path, "/")
	}
	if bucket == "" {
		respondS3ErrorXML(w, http.StatusBadRequest, "InvalidBucketName", "missing bucket name")
		return true
	}
	if q.Has("metadataAnnotationTable") {
		s.putS3AnnotationTableConfiguration(w, r, bucket, key)
		return true
	}
	if key == "" {
		respondS3ErrorXML(w, http.StatusBadRequest, "InvalidRequest", "annotations are object scoped")
		return true
	}
	obj, err := s.s3.GetObject(bucket, key)
	if err != nil {
		if errors.Is(err, s3.ErrBucketNotFound) {
			respondS3ErrorXML(w, http.StatusNotFound, "NoSuchBucket", "bucket not found")
		} else {
			respondS3ErrorXML(w, http.StatusNotFound, "NoSuchKey", "object not found")
		}
		return true
	}
	versionID := obj.VersionID
	if versionID == "" {
		versionID = "null"
	}
	w.Header().Set("x-amz-object-version-id", versionID)
	name := q.Get("annotationName")
	switch {
	case r.Method == http.MethodPut:
		s.putS3Annotation(w, r, bucket, key, name)
	case r.Method == http.MethodGet && q.Has("annotationName"):
		s.getS3Annotation(w, bucket, key, name)
	case r.Method == http.MethodGet:
		s.listS3Annotations(w, r, bucket, key)
	case r.Method == http.MethodDelete:
		if !s.s3Annotations.remove(bucket, key, name) {
			respondS3ErrorXML(w, http.StatusNotFound, "NoSuchAnnotation", "annotation not found")
			return true
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		respondS3ErrorXML(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
	return true
}

func s3AnnValidName(name string) bool {
	return name != "" && len(name) <= s3AnnMaxNameLen
}

func (s *Server) putS3Annotation(w http.ResponseWriter, r *http.Request, bucket, key, name string) {
	if !s3AnnValidName(name) {
		respondS3ErrorXML(w, http.StatusBadRequest, "InvalidArgument", "annotationName is required and at most 1024 bytes")
		return
	}
	if r.ContentLength < 0 || r.ContentLength > s3AnnMaxPayload {
		respondS3ErrorXML(w, http.StatusBadRequest, "EntityTooLarge", "annotation payload needs a Content-Length of at most 1 MiB")
		return
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, r.ContentLength))
	if err != nil {
		respondS3ErrorXML(w, http.StatusBadRequest, "IncompleteBody", "could not read annotation payload")
		return
	}
	a, err := s.s3Annotations.put(bucket, key, name, payload)
	if err != nil {
		respondS3ErrorXML(w, http.StatusBadRequest, "TooManyAnnotations", err.Error())
		return
	}
	w.Header().Set("ETag", a.etag)
	respondS3XML(w, http.StatusOK, struct {
		XMLName        xml.Name `xml:"PutObjectAnnotationResult"`
		Key            string   `xml:"Key"`
		AnnotationName string   `xml:"AnnotationName"`
	}{Key: key, AnnotationName: name})
}

func (s *Server) getS3Annotation(w http.ResponseWriter, bucket, key, name string) {
	a, ok := s.s3Annotations.get(bucket, key, name)
	if !ok {
		respondS3ErrorXML(w, http.StatusNotFound, "NoSuchAnnotation", "annotation not found")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", a.etag)
	w.Header().Set("Last-Modified", a.lastModified.Format(http.TimeFormat))
	w.Header().Set("Content-Length", strconv.Itoa(len(a.payload)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.payload)
}

func (s *Server) listS3Annotations(w http.ResponseWriter, r *http.Request, bucket, key string) {
	q := r.URL.Query()
	limit := s3AnnDefaultPageSize
	if v := q.Get("max-annotation-results"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || n > s3AnnDefaultPageSize {
			respondS3ErrorXML(w, http.StatusBadRequest, "InvalidArgument", "max-annotation-results must be between 1 and 1000")
			return
		}
		limit = int(n)
	}
	prefix, token := q.Get("annotation-prefix"), q.Get("continuation-token")
	entries := s.s3Annotations.list(bucket, key, prefix, token)
	type result struct {
		XMLName               xml.Name            `xml:"ListObjectAnnotationsResult"`
		Xmlns                 string              `xml:"xmlns,attr"`
		Bucket                string              `xml:"Bucket"`
		Key                   string              `xml:"Key"`
		AnnotationPrefix      string              `xml:"AnnotationPrefix,omitempty"`
		MaxAnnotationResults  int                 `xml:"MaxAnnotationResults"`
		AnnotationCount       int                 `xml:"AnnotationCount"`
		ContinuationToken     string              `xml:"ContinuationToken,omitempty"`
		NextContinuationToken string              `xml:"NextContinuationToken,omitempty"`
		Annotations           []s3AnnotationEntry `xml:"Annotations>AnnotationEntry"`
	}
	out := result{Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/", Bucket: bucket, Key: key, AnnotationPrefix: prefix,
		MaxAnnotationResults: limit, ContinuationToken: token}
	if len(entries) > limit {
		entries = entries[:limit]
		out.NextContinuationToken = entries[limit-1].AnnotationName
	}
	out.Annotations, out.AnnotationCount = entries, len(entries)
	respondS3XML(w, http.StatusOK, out)
}

func (s *Server) putS3AnnotationTableConfiguration(w http.ResponseWriter, r *http.Request, bucket, key string) {
	if key != "" || r.Method != http.MethodPut {
		respondS3ErrorXML(w, http.StatusBadRequest, "InvalidRequest", "annotation table configuration is a bucket-scoped PUT")
		return
	}
	if _, err := s.s3.GetBucket(bucket); err != nil {
		respondS3ErrorXML(w, http.StatusNotFound, "NoSuchBucket", "bucket not found")
		return
	}
	var cfg struct {
		XMLName            xml.Name `xml:"AnnotationTableConfiguration"`
		ConfigurationState string   `xml:"ConfigurationState"`
	}
	if err := xml.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg); err != nil {
		respondS3ErrorXML(w, http.StatusBadRequest, "MalformedXML", "invalid XML")
		return
	}
	if cfg.ConfigurationState != "ENABLED" && cfg.ConfigurationState != "DISABLED" {
		respondS3ErrorXML(w, http.StatusBadRequest, "InvalidArgument", "ConfigurationState must be ENABLED or DISABLED")
		return
	}
	s.s3Annotations.setTableState(bucket, cfg.ConfigurationState)
	w.WriteHeader(http.StatusOK)
}
