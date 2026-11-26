package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon WorkDocs: REST JSON under /api/v1, signed workdocs. One
// organization (directory) with an administrator user; SigV4 callers act as
// that administrator, and an Authentication header names the acting user by
// user ID (Stackyard has no WorkDocs sign-in to issue real tokens).
// Document bytes go to pre-signed upload URLs that Stackyard serves under
// /_stackyard/workdocs/, and come back from its download URLs.

const (
	wdocSigningName = "workdocs"
	wdocMaxBody     = 1 << 20
	wdocContentType = "application/json"
	wdocLinkPrefix  = "/_stackyard/workdocs/"
	wdocUploadPath  = wdocLinkPrefix + "upload/"
	wdocFilePath    = wdocLinkPrefix + "download/"
)

type wdocErr struct {
	status    int
	code, msg string
}

func wdocError(status int, code, format string, args ...any) *wdocErr {
	return &wdocErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func wdocInvalid(format string, args ...any) *wdocErr {
	return wdocError(http.StatusBadRequest, "InvalidArgumentException", format, args...)
}

func wdocMissing(kind, id string) *wdocErr {
	return wdocError(http.StatusNotFound, "EntityNotExistsException", "The %s %s does not exist.", kind, id)
}

func wdocProhibited(format string, args ...any) *wdocErr {
	return wdocError(http.StatusConflict, "ProhibitedStateException", format, args...)
}

func wdocLimit(format string, args ...any) *wdocErr {
	return wdocError(http.StatusConflict, "LimitExceededException", format, args...)
}

func (s *Server) handleWorkDocsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, wdocLinkPrefix) {
		s.handleWorkDocsLinks(w, r)
		return true
	}
	if r.Header.Get("X-Amz-Target") != "" {
		return false
	}
	hint := sigV4ServiceHint(r)
	if hint != wdocSigningName && (hint != "" || !strings.HasPrefix(strings.ToLower(r.Host), wdocSigningName+".")) {
		return false
	}
	op, ok := wdocMatch(r.Method, r.URL.EscapedPath())
	if !ok {
		writeWDocError(w, wdocError(http.StatusNotFound, "EntityNotExistsException", "no such operation"))
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > wdocMaxBody {
		writeWDocError(w, wdocInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, wdocSigningName); !ok {
		writeWDocError(w, &wdocErr{status: status, code: code, msg: msg})
		return true
	}
	model := awsmodels.WorkDocsModel()
	in, problem := awsModelDecodeREST(r, model, model.Operations[op.op.Name].Input, op.params)
	if problem != "" {
		writeWDocError(w, wdocInvalid("%s", problem))
		return true
	}
	body, werr := s.workDocs.run(op.op.Name, in, r.Header.Get("Authentication"), mmgrBaseURL(r))
	if werr != nil {
		writeWDocError(w, werr)
		return true
	}
	if op.op.Status == http.StatusNoContent {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	w.Header().Set("Content-Type", wdocContentType)
	w.WriteHeader(op.op.Status)
	_, _ = w.Write(body)
	return true
}

type wdocMatched struct {
	op     wdocOperation
	params map[string]string
}

func wdocMatch(method, path string) (wdocMatched, bool) {
	for _, op := range workDocsOperations {
		if op.Method != method {
			continue
		}
		if params, ok := matchSESV2PathPattern(op.Pattern, path); ok {
			return wdocMatched{op: op, params: params}, true
		}
	}
	return wdocMatched{}, false
}

func writeWDocError(w http.ResponseWriter, e *wdocErr) {
	body, _ := json.Marshal(map[string]any{"Message": e.msg})
	w.Header().Set("Content-Type", wdocContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// handleWorkDocsLinks takes document uploads at the URLs
// InitiateDocumentVersionUpload hands out and serves version downloads.
func (s *Server) handleWorkDocsLinks(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	var prefix string
	switch {
	case r.Method == http.MethodPut && strings.HasPrefix(path, wdocUploadPath):
		prefix = wdocUploadPath
	case r.Method == http.MethodGet && strings.HasPrefix(path, wdocFilePath):
		prefix = wdocFilePath
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, verID, _ := strings.Cut(strings.TrimPrefix(path, prefix), "/")
	key := r.URL.Query().Get("key")
	if prefix == wdocFilePath {
		data, ok := s.workDocs.download(docID, verID, key)
		if !ok {
			http.Error(w, "the download URL is not valid", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "attachment")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	if r.ContentLength < 0 || r.ContentLength > wdocMaxUpload {
		http.Error(w, fmt.Sprintf("uploads must declare a Content-Length of at most %d MiB", wdocMaxUpload>>20), http.StatusRequestEntityTooLarge)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, wdocMaxUpload+1))
	if err != nil || len(data) > wdocMaxUpload {
		http.Error(w, "the upload can't be read", http.StatusBadRequest)
		return
	}
	status, msg := s.workDocs.upload(docID, verID, key, r.Header.Get("Content-Type"), data)
	if status != http.StatusOK {
		http.Error(w, msg, status)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func wdocKeyOK(got, want string) bool {
	return want != "" && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
