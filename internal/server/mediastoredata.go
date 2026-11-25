package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Elemental MediaStore Data Plane: REST, signed mediastore, sent to a
// container's endpoint. DescribeContainer hands out
// <server>/_stackyard/mediastore-data/<endpoint id>; a Host of
// <endpoint id>.data.mediastore.<region>... (through a proxy) works too.

const (
	mstdPathPrefix = "/_stackyard/mediastore-data/"
	// MediaStore objects are at most 25 MB; Stackyard keeps at most 256 MiB
	// and 10,000 objects across every container (metadata counts too).
	mstdMaxObject      = 25 << 20
	mstdMaxStoredBytes = 256 << 20
	mstdMaxObjects     = 10000
	mstdMaxCacheCtl    = 1024
)

type mstdObject struct {
	data                     []byte // replaced on overwrite, never mutated
	etag, contentType, cache string
	modified                 time.Time
}

func (o *mstdObject) cost(path string) int64 {
	return int64(len(o.data) + len(path) + len(o.contentType) + len(o.cache))
}

func mstdNoObject(path string) *mstErr {
	return &mstErr{http.StatusNotFound, "ObjectNotFoundException", fmt.Sprintf("Object %s does not exist", path)}
}

// mstdTarget finds the endpoint id and object path a request addresses.
func mstdTarget(r *http.Request) (id, path string, ok bool) {
	raw := rawRequestPath(r)
	if rest, found := strings.CutPrefix(raw, mstdPathPrefix); found {
		id, path, _ = strings.Cut(rest, "/")
		return id, path, id != ""
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if label, _, found := strings.Cut(host, ".data.mediastore."); found && label != "" && !strings.Contains(label, ".") {
		return label, strings.TrimPrefix(raw, "/"), true
	}
	return "", "", false
}

func (s *Server) handleMediaStoreDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Amz-Target") != "" {
		return false
	}
	id, rawPath, ok := mstdTarget(r)
	if !ok {
		return false
	}
	if svc := sigV4ServiceHint(r); svc != "" && svc != "mediastore" {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > mstdMaxObject {
		respondMSTDError(w, mstInvalid("request body must declare a Content-Length of at most %d bytes", mstdMaxObject))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "mediastore"); !ok {
		respondMSTDError(w, &mstErr{status, code, msg})
		return true
	}
	path, err := url.PathUnescape(rawPath)
	if err != nil {
		respondMSTDError(w, mstInvalid("invalid path"))
		return true
	}
	if merr := s.mstdRoute(w, r, id, path); merr != nil {
		respondMSTDError(w, merr)
	}
	return true
}

func (s *Server) mstdRoute(w http.ResponseWriter, r *http.Request, id, path string) *mstErr {
	if path == "" {
		if r.Method != http.MethodGet {
			return mstInvalid("no MediaStore Data operation matches %s /", r.Method)
		}
		return s.mstdListItems(w, r, id)
	}
	if problems := awsmodels.MediaStoreDataModel().Validate("PathNaming", path); len(problems) > 0 {
		return mstInvalid("Path %s", strings.TrimPrefix(problems[0], "input: "))
	}
	switch r.Method {
	case http.MethodPut:
		return s.mstdPutObject(w, r, id, path)
	case http.MethodGet, http.MethodHead:
		return s.mstdGetObject(w, r, id, path)
	case http.MethodDelete:
		if err := s.mediastore.deleteObject(id, path); err != nil {
			return err
		}
		respondMSTDJSON(w, map[string]any{})
		return nil
	}
	return mstInvalid("no MediaStore Data operation matches %s", r.Method)
}

func (s *Server) mstdPutObject(w http.ResponseWriter, r *http.Request, id, path string) *mstErr {
	m := awsmodels.MediaStoreDataModel()
	ctype := r.Header.Get("Content-Type")
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	if p := m.Validate("ContentType", ctype); len(p) > 0 {
		return mstInvalid("Content-Type %s", strings.TrimPrefix(p[0], "input: "))
	}
	if sc := r.Header.Get("x-amz-storage-class"); sc != "" && sc != "TEMPORAL" {
		return mstInvalid("x-amz-storage-class must be TEMPORAL")
	}
	if ua := r.Header.Get("x-amz-upload-availability"); ua != "" && ua != "STANDARD" && ua != "STREAMING" {
		return mstInvalid("x-amz-upload-availability must be STANDARD or STREAMING")
	}
	cache := r.Header.Get("Cache-Control")
	if len(cache) > mstdMaxCacheCtl {
		return mstInvalid("Cache-Control is longer than %d bytes", mstdMaxCacheCtl)
	}
	// ContentLength is declared and capped above; read exactly that much.
	data := make([]byte, r.ContentLength)
	if _, err := io.ReadFull(r.Body, data); err != nil {
		return mstInvalid("request body is shorter than its Content-Length")
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	obj := &mstdObject{data: data, etag: digest, contentType: ctype, cache: cache, modified: time.Now().UTC()}
	if err := s.mediastore.putObject(id, path, obj); err != nil {
		return err
	}
	respondMSTDJSON(w, map[string]any{"ContentSHA256": digest, "ETag": digest, "StorageClass": "TEMPORAL"})
	return nil
}

func (s *Server) mstdGetObject(w http.ResponseWriter, r *http.Request, id, path string) *mstErr {
	obj, err := s.mediastore.getObject(id, path)
	if err != nil {
		return err
	}
	// obj.data is never mutated after it's stored, so reading it here is safe.
	data, status := obj.data, http.StatusOK
	h := w.Header()
	if rng := r.Header.Get("Range"); rng != "" && r.Method == http.MethodGet {
		start, end, rerr := mstdRange(rng, int64(len(data)))
		if rerr != nil {
			return rerr
		}
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
		data, status = data[start:end+1], http.StatusPartialContent
	}
	h.Set("Content-Type", mstdSafeContentType(obj.contentType))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Length", strconv.Itoa(len(data)))
	h.Set("ETag", obj.etag)
	h.Set("Last-Modified", obj.modified.Format(http.TimeFormat))
	if obj.cache != "" {
		h.Set("Cache-Control", obj.cache)
	}
	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
	return nil
}

// mstdRange parses bytes=a-b, bytes=a- or bytes=-n against size.
func mstdRange(spec string, size int64) (int64, int64, *mstErr) {
	bad := &mstErr{http.StatusRequestedRangeNotSatisfiable, "RequestedRangeNotSatisfiableException", "The requested range is not satisfiable"}
	r, ok := strings.CutPrefix(spec, "bytes=")
	a, b, ok2 := strings.Cut(r, "-")
	if !ok || !ok2 || (a == "" && b == "") {
		return 0, 0, mstInvalid("Range must look like bytes=start-end")
	}
	parse := func(v string) (int64, bool) {
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil && n >= 0
	}
	if a == "" { // suffix: the last n bytes
		n, ok := parse(b)
		if !ok {
			return 0, 0, mstInvalid("Range must look like bytes=start-end")
		}
		if n == 0 || size == 0 {
			return 0, 0, bad
		}
		return max(size-n, 0), size - 1, nil
	}
	start, ok := parse(a)
	if !ok {
		return 0, 0, mstInvalid("Range must look like bytes=start-end")
	}
	end := size - 1
	if b != "" {
		if end, ok = parse(b); !ok || end < start {
			return 0, 0, mstInvalid("Range must look like bytes=start-end")
		}
	}
	if start >= size {
		return 0, 0, bad
	}
	return start, min(end, size-1), nil
}

// mstdSafeContentType echoes a stored content type only when browsers won't
// render it as a document; anything else is served as octet-stream.
func mstdSafeContentType(stored string) string {
	switch strings.ToLower(stored) {
	case "video/mp4":
		return "video/mp4"
	case "video/mp2t":
		return "video/MP2T"
	case "video/webm":
		return "video/webm"
	case "video/quicktime":
		return "video/quicktime"
	case "audio/mp4":
		return "audio/mp4"
	case "audio/mpeg":
		return "audio/mpeg"
	case "audio/aac":
		return "audio/aac"
	case "image/jpeg":
		return "image/jpeg"
	case "image/png":
		return "image/png"
	case "application/vnd.apple.mpegurl":
		return "application/vnd.apple.mpegurl"
	case "application/x-mpegurl":
		return "application/x-mpegURL"
	case "application/json":
		return "application/json"
	case "text/plain":
		return "text/plain"
	}
	return "application/octet-stream"
}

func (s *Server) mstdListItems(w http.ResponseWriter, r *http.Request, id string) *mstErr {
	q := r.URL.Query()
	folder := q.Get("Path")
	if p := awsmodels.MediaStoreDataModel().Validate("ListPathNaming", folder); len(p) > 0 {
		return mstInvalid("Path %s", strings.TrimPrefix(p[0], "input: "))
	}
	max := 1000
	if v := q.Get("MaxResults"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || n > 1000 {
			return mstInvalid("MaxResults must be between 1 and 1000")
		}
		max = int(n)
	}
	offset := 0
	if tok := q.Get("NextToken"); tok != "" {
		raw, err := base64.RawURLEncoding.DecodeString(tok)
		n, perr := strconv.ParseInt(string(raw), 10, 32)
		if err != nil || perr != nil || n < 0 {
			return mstInvalid("NextToken is not valid")
		}
		offset = int(n)
	}
	items, err := s.mediastore.listItems(id, strings.Trim(folder, "/"))
	if err != nil {
		return err
	}
	out := map[string]any{}
	if offset > len(items) {
		offset = len(items)
	}
	end := min(offset+max, len(items))
	if end < len(items) {
		out["NextToken"] = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	out["Items"] = items[offset:end]
	respondMSTDJSON(w, out)
	return nil
}

// ---- store (objects live in mstStore containers) ----

// container resolves an endpoint id. Callers hold st.mu.
func (st *mstStore) container(id string) (*mstContainer, *mstErr) {
	c := st.containers[st.endpoints[id]]
	if c == nil {
		return nil, &mstErr{http.StatusNotFound, "ContainerNotFoundException", "No container has endpoint " + id}
	}
	c.settle()
	return c, nil
}

func (st *mstStore) putObject(id, path string, obj *mstdObject) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.container(id)
	if err != nil {
		return err
	}
	// A path can't run through an existing object, nor be an existing folder.
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		if c.objects[strings.Join(parts[:i], "/")] != nil {
			return mstInvalid("%s is an object, not a folder", strings.Join(parts[:i], "/"))
		}
	}
	if st.isFolder(c, path) {
		return mstInvalid("%s is a folder", path)
	}
	bytes, count := st.storedBytes+obj.cost(path), st.storedObjects+1
	if old := c.objects[path]; old != nil {
		bytes, count = bytes-old.cost(path), count-1
	}
	if bytes > mstdMaxStoredBytes || count > mstdMaxObjects {
		return &mstErr{http.StatusBadRequest, "LimitExceededException", fmt.Sprintf("Stackyard keeps at most %d objects and %d bytes of MediaStore data", mstdMaxObjects, mstdMaxStoredBytes)}
	}
	c.objects[path] = obj
	st.storedBytes, st.storedObjects = bytes, count
	return nil
}

// isFolder reports whether any object lives under path/. ponytail: linear
// scan, bounded by mstdMaxObjects; index by folder if that gets slow.
func (st *mstStore) isFolder(c *mstContainer, path string) bool {
	for p := range c.objects {
		if strings.HasPrefix(p, path+"/") {
			return true
		}
	}
	return false
}

// getObject returns a copy of the object's metadata sharing its immutable data.
func (st *mstStore) getObject(id, path string) (mstdObject, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.container(id)
	if err != nil {
		return mstdObject{}, err
	}
	obj := c.objects[path]
	if obj == nil {
		return mstdObject{}, mstdNoObject(path)
	}
	return *obj, nil
}

func (st *mstStore) deleteObject(id, path string) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.container(id)
	if err != nil {
		return err
	}
	obj := c.objects[path]
	if obj == nil {
		return mstdNoObject(path)
	}
	st.storedBytes -= obj.cost(path)
	st.storedObjects--
	delete(c.objects, path)
	return nil
}

// listItems returns the objects and folders directly inside folder ("" is
// the container root), sorted by name. Folders exist while they hold objects.
func (st *mstStore) listItems(id, folder string) ([]any, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.container(id)
	if err != nil {
		return nil, err
	}
	prefix := ""
	if folder != "" {
		prefix = folder + "/"
	}
	type entry struct {
		name string
		item map[string]any
	}
	folders := map[string]bool{}
	var entries []entry
	for p, obj := range c.objects {
		rest, ok := strings.CutPrefix(p, prefix)
		if !ok {
			continue
		}
		if sub, _, nested := strings.Cut(rest, "/"); nested {
			if !folders[sub] {
				folders[sub] = true
				entries = append(entries, entry{sub, map[string]any{"Name": sub, "Type": "FOLDER"}})
			}
			continue
		}
		entries = append(entries, entry{rest, map[string]any{
			"Name": rest, "Type": "OBJECT", "ETag": obj.etag, "ContentType": obj.contentType,
			"ContentLength": len(obj.data), "LastModified": float64(obj.modified.UnixMilli()) / 1000,
		}})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	items := make([]any, len(entries))
	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

func respondMSTDJSON(w http.ResponseWriter, body map[string]any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func respondMSTDError(w http.ResponseWriter, e *mstErr) {
	raw, _ := json.Marshal(map[string]string{"Message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
