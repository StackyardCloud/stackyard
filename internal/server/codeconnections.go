package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS CodeConnections (renamed from CodeStar Connections) and CodeStar
// Connections are one service under two names: the same awsJson1_0 operation
// set, signed codeconnections or codestar-connections, sharing one store. Each
// API renders ARNs with its own service prefix and accepts either prefix.

const (
	cxnMaxBody      = 256 << 10
	cxnStackyardAPI = "/_stackyard/codeconnections/"
)

// cxnAPI is one of the two names the service is reachable under.
type cxnAPI struct {
	service  string   // ARN service prefix and SigV4 signing name
	targets  []string // X-Amz-Target prefixes, including the trailing dot
	model    func() *awsmodels.Model
	knownOps map[string]bool
}

var codeConnectionsAPI = &cxnAPI{
	service: "codeconnections",
	targets: []string{"com.amazonaws.codeconnections.CodeConnections_20231201.", "CodeConnections_20231201."},
	model:   awsmodels.CodeConnectionsModel,
	knownOps: func() map[string]bool {
		out := map[string]bool{}
		for _, op := range codeconnectionsOperations {
			out[op.Name] = true
		}
		return out
	}(),
}

func (api *cxnAPI) action(r *http.Request) (string, bool) {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	for _, prefix := range api.targets {
		if op, ok := strings.CutPrefix(target, prefix); ok {
			return op, true
		}
	}
	return "", false
}

func (s *Server) handleCodeConnectionsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost && strings.HasPrefix(rawRequestPath(r), cxnStackyardAPI) {
		return s.handleCodeConnectionsStackyard(w, r)
	}
	if r.Method != http.MethodPost {
		return false
	}
	for _, api := range []*cxnAPI{codeConnectionsAPI, codeStarConnectionsAPI} {
		if op, ok := api.action(r); ok {
			s.serveCodeConnections(w, r, api, op)
			return true
		}
	}
	return false
}

func (s *Server) serveCodeConnections(w http.ResponseWriter, r *http.Request, api *cxnAPI, op string) {
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > cxnMaxBody {
		respondCodeConnectionsError(w, http.StatusBadRequest, "ValidationException", "request body must declare a Content-Length of at most 256 KiB")
		return
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, api.service); !ok {
		respondCodeConnectionsError(w, status, code, msg)
		return
	}
	if !api.knownOps[op] {
		respondCodeConnectionsError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return
	}
	in, err := decodeCodeConnectionsBody(r)
	if err != nil {
		respondCodeConnectionsError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return
	}
	model := api.model()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondCodeConnectionsError(w, http.StatusBadRequest, "ValidationException", strings.Join(problems, "; "))
		return
	}
	out, cerr := s.codeConnections.handle(api, op, in)
	if cerr != nil {
		respondCodeConnectionsError(w, cerr.status, cerr.code, cerr.msg)
		return
	}
	// out was shaped to the output model under the store lock.
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func decodeCodeConnectionsBody(r *http.Request) (map[string]any, error) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	in := map[string]any{}
	if strings.TrimSpace(string(raw)) == "" {
		return in, nil
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	return in, nil
}

func respondCodeConnectionsError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// handleCodeConnectionsStackyard serves the Stackyard-only endpoints that stand
// in for steps AWS completes outside the API (signed like the API itself):
//
//	POST /_stackyard/codeconnections/connections/{id}/activate   connection handshake done -> AVAILABLE
//	POST /_stackyard/codeconnections/hosts/{id}/activate         host set up -> AVAILABLE
//	POST /_stackyard/codeconnections/sync-configurations/{SyncType}/{ResourceName}/blockers
//	                                                             a sync failed -> new ACTIVE blocker
func (s *Server) handleCodeConnectionsStackyard(w http.ResponseWriter, r *http.Request) bool {
	if r.ContentLength < 0 || r.ContentLength > cxnMaxBody {
		respondCodeConnectionsError(w, http.StatusBadRequest, "ValidationException", "request body must declare a Content-Length of at most 256 KiB")
		return true
	}
	api := codeConnectionsAPI
	if strings.TrimSpace(sigV4ServiceHint(r)) == codeStarConnectionsAPI.service {
		api = codeStarConnectionsAPI
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, api.service); !ok {
		respondCodeConnectionsError(w, status, code, msg)
		return true
	}
	parts := strings.Split(strings.TrimPrefix(rawRequestPath(r), cxnStackyardAPI), "/")
	for i, p := range parts {
		unescaped, err := url.PathUnescape(p)
		if err != nil {
			respondCodeConnectionsError(w, http.StatusBadRequest, "ValidationException", "invalid path")
			return true
		}
		parts[i] = unescaped
	}
	var out map[string]any
	var cerr *cxnError
	switch {
	case len(parts) == 3 && parts[0] == "connections" && parts[2] == "activate":
		out, cerr = s.codeConnections.activateConnection(api, parts[1])
	case len(parts) == 3 && parts[0] == "hosts" && parts[2] == "activate":
		out, cerr = s.codeConnections.activateHost(api, parts[1])
	case len(parts) == 4 && parts[0] == "sync-configurations" && parts[3] == "blockers":
		in, err := decodeCodeConnectionsBody(r)
		if err != nil {
			respondCodeConnectionsError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
			return true
		}
		reason, _ := in["CreatedReason"].(string)
		out, cerr = s.codeConnections.addSyncBlocker(parts[1], parts[2], reason)
	default:
		respondCodeConnectionsError(w, http.StatusNotFound, "ResourceNotFoundException", "unknown Stackyard codeconnections endpoint")
		return true
	}
	if cerr != nil {
		respondCodeConnectionsError(w, cerr.status, cerr.code, cerr.msg)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}
