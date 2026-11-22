package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// glcMaxJSONBody bounds the JSON bodies (policies, job parameters, tags);
// only UploadArchive and UploadMultipartPart may send up to glcMaxBody.
const glcMaxJSONBody = 1 << 20

func (s *Server) handleGlacierRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGlacierRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > glcMaxBody {
		respondGlacierError(w, glcInvalid("Request bodies must declare a Content-Length of at most %d bytes", glcMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "glacier"); !ok {
		respondGlacierError(w, &glcErr{status, code, msg})
		return true
	}
	if v := r.Header.Get("x-amz-glacier-version"); v != "" && v != "2012-06-01" {
		respondGlacierError(w, glcInvalid("Unsupported x-amz-glacier-version %q. Use 2012-06-01", v))
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, p := range seg {
		u, err := url.PathUnescape(p)
		if err != nil {
			respondGlacierError(w, glcInvalid("Invalid path segment %q", p))
			return true
		}
		seg[i] = u
	}
	if seg[0] != "-" && seg[0] != glcAccountID {
		respondGlacierError(w, &glcErr{http.StatusForbidden, "AccessDeniedException", "Account " + seg[0] + " is not this account (" + glcAccountID + ")"})
		return true
	}
	if err := s.glacierRoute(w, r, seg); err != nil {
		respondGlacierError(w, err)
	}
	return true
}

func isGlacierRESTCandidate(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.TrimSpace(sigV4ServiceHint(r)) != "glacier" && !strings.HasPrefix(host, "glacier.") && !strings.Contains(host, ".glacier.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	if len(seg) < 2 {
		return false
	}
	switch seg[1] {
	case "vaults", "provisioned-capacity":
		return true
	case "policies":
		return len(seg) == 3 && seg[2] == "data-retrieval"
	}
	return false
}

func glcUnknown(r *http.Request) *glcErr {
	return &glcErr{http.StatusNotFound, "UnknownOperationException", "no Glacier operation matches " + r.Method + " " + rawRequestPath(r)}
}

func (s *Server) glacierRoute(w http.ResponseWriter, r *http.Request, seg []string) *glcErr {
	switch {
	case len(seg) == 3 && seg[1] == "policies":
		return s.glacierRetrievalPolicy(w, r)
	case len(seg) == 2 && seg[1] == "provisioned-capacity":
		switch r.Method {
		case http.MethodGet:
			respondGlacierJSON(w, s.glacier.listCapacity())
			return nil
		case http.MethodPost:
			id, err := s.glacier.purchaseCapacity()
			if err != nil {
				return err
			}
			w.Header().Set("x-amz-capacity-id", id)
			glcRespondStatus(w, http.StatusCreated)
			return nil
		}
	case len(seg) == 2 && r.Method == http.MethodGet:
		marker, limit, err := glcListQuery(r, 10)
		if err != nil {
			return err
		}
		out, err := s.glacier.listVaults(marker, limit)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case len(seg) == 3:
		return s.glacierVault(w, r, seg[2])
	case len(seg) >= 4:
		return s.glacierVaultResource(w, r, seg[2], seg[3:])
	}
	return glcUnknown(r)
}

func (s *Server) glacierVault(w http.ResponseWriter, r *http.Request, name string) *glcErr {
	switch r.Method {
	case http.MethodPut:
		if err := s.glacier.createVault(name); err != nil {
			return err
		}
		w.Header().Set("Location", glcVaultPath(name))
		glcRespondStatus(w, http.StatusCreated)
		return nil
	case http.MethodGet:
		out, err := s.glacier.describeVault(name)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case http.MethodDelete:
		if err := s.glacier.deleteVault(name); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	}
	return glcUnknown(r)
}

func (s *Server) glacierVaultResource(w http.ResponseWriter, r *http.Request, name string, rest []string) *glcErr {
	if err := glcCheckVaultName(name); err != nil {
		return err
	}
	switch rest[0] {
	case "archives":
		return s.glacierArchives(w, r, name, rest[1:])
	case "multipart-uploads":
		return s.glacierUploads(w, r, name, rest[1:])
	case "jobs":
		return s.glacierJobs(w, r, name, rest[1:])
	case "tags":
		return s.glacierTags(w, r, name, len(rest))
	case "access-policy":
		return s.glacierAccessPolicy(w, r, name, len(rest))
	case "notification-configuration":
		return s.glacierNotifications(w, r, name, len(rest))
	case "lock-policy":
		return s.glacierLock(w, r, name, rest[1:])
	}
	return glcUnknown(r)
}

func (s *Server) glacierArchives(w http.ResponseWriter, r *http.Request, name string, rest []string) *glcErr {
	switch {
	case len(rest) == 0 && r.Method == http.MethodPost:
		data, err := glcBody(r, glcMaxBody)
		if err != nil {
			return err
		}
		a, err := s.glacier.uploadArchive(name, r.Header.Get("x-amz-archive-description"), r.Header.Get("x-amz-sha256-tree-hash"), data)
		if err != nil {
			return err
		}
		w.Header().Set("Location", glcVaultPath(name)+"/archives/"+a.id)
		w.Header().Set("x-amz-archive-id", a.id)
		w.Header().Set("x-amz-sha256-tree-hash", a.hash)
		glcRespondStatus(w, http.StatusCreated)
		return nil
	case len(rest) == 1 && r.Method == http.MethodDelete:
		if err := s.glacier.deleteArchive(name, rest[0]); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	}
	return glcUnknown(r)
}

func (s *Server) glacierUploads(w http.ResponseWriter, r *http.Request, name string, rest []string) *glcErr {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodPost:
			raw := r.Header.Get("x-amz-part-size")
			if raw == "" {
				return glcMissing("Missing required header: x-amz-part-size")
			}
			size, perr := strconv.ParseInt(raw, 10, 64)
			if perr != nil {
				return glcInvalid("Invalid part size: %s", raw)
			}
			u, err := s.glacier.initiateUpload(name, r.Header.Get("x-amz-archive-description"), size)
			if err != nil {
				return err
			}
			w.Header().Set("Location", glcVaultPath(name)+"/multipart-uploads/"+u.id)
			w.Header().Set("x-amz-multipart-upload-id", u.id)
			glcRespondStatus(w, http.StatusCreated)
			return nil
		case http.MethodGet:
			marker, limit, err := glcListQuery(r, 50)
			if err != nil {
				return err
			}
			out, err := s.glacier.listUploads(name, marker, limit)
			if err != nil {
				return err
			}
			respondGlacierJSON(w, out)
			return nil
		}
		return glcUnknown(r)
	}
	if len(rest) != 1 {
		return glcUnknown(r)
	}
	id := rest[0]
	switch r.Method {
	case http.MethodPut:
		start, end, err := glcContentRange(r.Header.Get("Content-Range"))
		if err != nil {
			return err
		}
		data, err := glcBody(r, glcMaxBody)
		if err != nil {
			return err
		}
		checksum := r.Header.Get("x-amz-sha256-tree-hash")
		if err := s.glacier.uploadPart(name, id, checksum, start, end, data); err != nil {
			return err
		}
		w.Header().Set("x-amz-sha256-tree-hash", strings.ToLower(checksum))
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case http.MethodPost:
		raw := r.Header.Get("x-amz-archive-size")
		if raw == "" {
			return glcMissing("Missing required header: x-amz-archive-size")
		}
		size, perr := strconv.ParseInt(raw, 10, 64)
		if perr != nil {
			return glcInvalid("Invalid archive size: %s", raw)
		}
		a, err := s.glacier.completeUpload(name, id, r.Header.Get("x-amz-sha256-tree-hash"), size)
		if err != nil {
			return err
		}
		w.Header().Set("Location", glcVaultPath(name)+"/archives/"+a.id)
		w.Header().Set("x-amz-archive-id", a.id)
		w.Header().Set("x-amz-sha256-tree-hash", a.hash)
		glcRespondStatus(w, http.StatusCreated)
		return nil
	case http.MethodDelete:
		if err := s.glacier.abortUpload(name, id); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case http.MethodGet:
		marker, limit, err := glcListQuery(r, 50)
		if err != nil {
			return err
		}
		out, err := s.glacier.listParts(name, id, marker, limit)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	}
	return glcUnknown(r)
}

// glcContentRange parses UploadMultipartPart's "bytes start-end/*".
func glcContentRange(raw string) (int64, int64, *glcErr) {
	if raw == "" {
		return 0, 0, glcMissing("Missing required header: Content-Range")
	}
	spec, ok := strings.CutPrefix(raw, "bytes ")
	spec, total, ok2 := strings.Cut(spec, "/")
	a, b, ok3 := strings.Cut(spec, "-")
	start, err1 := strconv.ParseInt(a, 10, 64)
	end, err2 := strconv.ParseInt(b, 10, 64)
	if !ok || !ok2 || !ok3 || total != "*" || err1 != nil || err2 != nil || start < 0 || end < start {
		return 0, 0, glcInvalid("Invalid Content-Range: %s. Use bytes start-end/*", raw)
	}
	return start, end, nil
}

func (s *Server) glacierJobs(w http.ResponseWriter, r *http.Request, name string, rest []string) *glcErr {
	switch {
	case len(rest) == 0 && r.Method == http.MethodPost:
		p := &glcJobParams{}
		if err := glcDecodeJSON(r, p); err != nil {
			return err
		}
		if err := glcCheckJobParams(p); err != nil {
			return err
		}
		if p.SNSTopic != "" {
			if _, err := s.sns.GetTopic(p.SNSTopic); err != nil {
				return glcInvalid("SNS topic not found: %s", p.SNSTopic)
			}
		}
		if p.Type == "select" {
			if _, err := s.s3.GetBucket(glcSelectBucket(p)); err != nil {
				return glcInvalid("Output bucket not found: %s", glcSelectBucket(p))
			}
		}
		j, err := s.glacier.initiateJob(name, p)
		if err != nil {
			return err
		}
		w.Header().Set("Location", glcVaultPath(name)+"/jobs/"+j.id)
		w.Header().Set("x-amz-job-id", j.id)
		if j.outputPath != "" {
			w.Header().Set("x-amz-job-output-path", j.outputPath)
		}
		glcRespondStatus(w, http.StatusAccepted)
		return nil
	case len(rest) == 0 && r.Method == http.MethodGet:
		marker, limit, err := glcListQuery(r, 50)
		if err != nil {
			return err
		}
		q := r.URL.Query()
		status, completed := q.Get("statuscode"), q.Get("completed")
		if status != "" && status != "InProgress" && status != "Succeeded" && status != "Failed" {
			return glcInvalid("Invalid statuscode %q. Use InProgress, Succeeded or Failed", status)
		}
		if completed != "" && completed != "true" && completed != "false" {
			return glcInvalid("Invalid completed %q. Use true or false", completed)
		}
		out, err := s.glacier.listJobs(name, status, completed, marker, limit)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case len(rest) == 1 && r.Method == http.MethodGet:
		out, err := s.glacier.describeJob(name, rest[0])
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case len(rest) == 2 && rest[1] == "output" && r.Method == http.MethodGet:
		out, err := s.glacier.jobOutput(name, rest[0])
		if err != nil {
			return err
		}
		return glcWriteJobOutput(w, out, r.Header.Get("Range"))
	}
	return glcUnknown(r)
}

// glcWriteJobOutput writes retrieved bytes. Archive content is caller data, so
// every content type is a constant chosen here, next to the write.
func glcWriteJobOutput(w http.ResponseWriter, out *glcJobOutput, rangeHeader string) *glcErr {
	data := out.data
	status := http.StatusOK
	total := len(data)
	if rangeHeader != "" {
		spec, ok := strings.CutPrefix(rangeHeader, "bytes=")
		a, b, ok2 := strings.Cut(spec, "-")
		start, err1 := strconv.ParseInt(a, 10, 32)
		end, err2 := strconv.ParseInt(b, 10, 32)
		if !ok || !ok2 || err1 != nil || err2 != nil || start < 0 || end < start || int(start) >= total {
			return glcInvalid("Invalid Range: %s. Use bytes=start-end within the job output", rangeHeader)
		}
		last := min(int(end), total-1)
		data, status = data[start:last+1], http.StatusPartialContent
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, last, total))
		// Glacier only returns a checksum for tree-hash-aligned ranges.
		if start%glcTreeChunk == 0 && ((last+1)%glcTreeChunk == 0 || last == total-1) {
			w.Header().Set("x-amz-sha256-tree-hash", glcTreeHash(data))
		}
	} else {
		w.Header().Set("x-amz-sha256-tree-hash", glcTreeHash(data))
	}
	h := w.Header()
	h.Set("Accept-Ranges", "bytes")
	h.Set("X-Content-Type-Options", "nosniff")
	switch out.kind {
	case "json":
		h.Set("Content-Type", "application/json")
	case "csv":
		h.Set("Content-Type", "text/csv")
	default:
		h.Set("Content-Type", "application/octet-stream")
		if out.desc != "" {
			h.Set("x-amz-archive-description", out.desc)
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return nil
}

func (s *Server) glacierTags(w http.ResponseWriter, r *http.Request, name string, n int) *glcErr {
	if n != 1 {
		return glcUnknown(r)
	}
	switch {
	case r.Method == http.MethodGet:
		out, err := s.glacier.listTags(name)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case r.Method == http.MethodPost && r.URL.Query().Get("operation") == "add":
		var in struct {
			Tags map[string]string `json:"Tags"`
		}
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		if len(in.Tags) > glcMaxTags {
			return glcLimit("A vault can have at most %d tags", glcMaxTags)
		}
		if err := s.glacier.addTags(name, in.Tags); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case r.Method == http.MethodPost && r.URL.Query().Get("operation") == "remove":
		var in struct {
			TagKeys []string `json:"TagKeys"`
		}
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		if err := s.glacier.removeTags(name, in.TagKeys); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	}
	return glcUnknown(r)
}

type glcPolicyBody struct {
	Policy string `json:"Policy"`
}

func (s *Server) glacierAccessPolicy(w http.ResponseWriter, r *http.Request, name string, n int) *glcErr {
	if n != 1 {
		return glcUnknown(r)
	}
	switch r.Method {
	case http.MethodPut:
		var in glcPolicyBody
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		if err := s.glacier.setAccessPolicy(name, in.Policy); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case http.MethodGet, http.MethodDelete:
		out, err := s.glacier.accessPolicy(name, r.Method == http.MethodDelete)
		if err != nil {
			return err
		}
		if out == nil {
			glcRespondStatus(w, http.StatusNoContent)
			return nil
		}
		respondGlacierJSON(w, out)
		return nil
	}
	return glcUnknown(r)
}

func (s *Server) glacierNotifications(w http.ResponseWriter, r *http.Request, name string, n int) *glcErr {
	if n != 1 {
		return glcUnknown(r)
	}
	switch r.Method {
	case http.MethodPut:
		var in glcNotify
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		if err := glcCheckNotify(in); err != nil {
			return err
		}
		if _, err := s.sns.GetTopic(in.Topic); err != nil {
			return glcInvalid("SNS topic not found: %s", in.Topic)
		}
		if err := s.glacier.setNotifications(name, in); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case http.MethodGet, http.MethodDelete:
		out, err := s.glacier.notifications(name, r.Method == http.MethodDelete)
		if err != nil {
			return err
		}
		if out == nil {
			glcRespondStatus(w, http.StatusNoContent)
			return nil
		}
		respondGlacierJSON(w, out)
		return nil
	}
	return glcUnknown(r)
}

func (s *Server) glacierLock(w http.ResponseWriter, r *http.Request, name string, rest []string) *glcErr {
	switch {
	case len(rest) == 0 && r.Method == http.MethodPost:
		var in glcPolicyBody
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		id, err := s.glacier.initiateLock(name, in.Policy)
		if err != nil {
			return err
		}
		w.Header().Set("x-amz-lock-id", id)
		glcRespondStatus(w, http.StatusCreated)
		return nil
	case len(rest) == 0 && r.Method == http.MethodGet:
		out, err := s.glacier.getLock(name)
		if err != nil {
			return err
		}
		respondGlacierJSON(w, out)
		return nil
	case len(rest) == 0 && r.Method == http.MethodDelete:
		if err := s.glacier.abortLock(name); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	case len(rest) == 1 && r.Method == http.MethodPost:
		if err := s.glacier.completeLock(name, rest[0]); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	}
	return glcUnknown(r)
}

func (s *Server) glacierRetrievalPolicy(w http.ResponseWriter, r *http.Request) *glcErr {
	switch r.Method {
	case http.MethodGet:
		respondGlacierJSON(w, s.glacier.retrievalPolicy())
		return nil
	case http.MethodPut:
		var in struct {
			Policy *struct {
				Rules []glcRule `json:"Rules"`
			} `json:"Policy"`
		}
		if err := glcDecodeJSON(r, &in); err != nil {
			return err
		}
		var rules []glcRule
		if in.Policy != nil {
			rules = in.Policy.Rules
		}
		if err := s.glacier.setRetrievalPolicy(rules); err != nil {
			return err
		}
		glcRespondStatus(w, http.StatusNoContent)
		return nil
	}
	return glcUnknown(r)
}

// glcListQuery reads the marker and a limit between 1 and 1000.
func glcListQuery(r *http.Request, def int) (string, int, *glcErr) {
	q := r.URL.Query()
	limit := def
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 1000 {
			return "", 0, glcInvalid("Invalid limit %q. Use 1 to 1000", raw)
		}
		limit = int(n)
	}
	return q.Get("marker"), limit, nil
}

func glcBody(r *http.Request, max int64) ([]byte, *glcErr) {
	if r.ContentLength > max {
		return nil, glcInvalid("Request body exceeds %d bytes", max)
	}
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, glcInvalid("unable to read request body")
	}
	return raw, nil
}

// glcDecodeJSON decodes a JSON body into a typed struct, rejecting unknown
// members so callers can't set anything the model doesn't define.
func glcDecodeJSON(r *http.Request, out any) *glcErr {
	raw, err := glcBody(r, glcMaxJSONBody)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil {
		return glcInvalid("Request body is not valid JSON for this operation")
	}
	return nil
}

func respondGlacierJSON(w http.ResponseWriter, body map[string]any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func glcRespondStatus(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

func respondGlacierError(w http.ResponseWriter, e *glcErr) {
	kind := "Client"
	if e.status >= 500 {
		kind = "Server"
	}
	raw, _ := json.Marshal(map[string]string{"code": e.code, "message": e.msg, "type": kind})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
