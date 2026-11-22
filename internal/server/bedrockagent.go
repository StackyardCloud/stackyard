package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// bedrockAgentPathPrefixes are the first path segments of every bedrock-agent
// operation. The Bedrock control plane, Bedrock Runtime and Bedrock Agent
// Runtime all sign as "bedrock" too, so a request is only claimed when its
// path matches one of this service's URI templates.
var bedrockAgentPathPrefixes = []string{"/agents/", "/knowledgebases/", "/flows/", "/prompts/", "/retrievalinstructions/", "/resourcepolicy/", "/tags/"}

func (s *Server) handleBedrockAgentRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBedrockAgentRESTCandidate(r) {
		return false
	}
	op, params := matchBedrockAgentRoute(r.Method, rawRequestPath(r))
	if op == nil {
		// Same prefixes, other bedrock service (for example InvokeAgent on
		// Bedrock Agent Runtime): let later routers try.
		return false
	}
	if op.Name == "TagResource" || op.Name == "UntagResource" || op.Name == "ListTagsForResource" {
		// Bedrock Agent Runtime tags sessions through the same /tags/{arn} route.
		if strings.Contains(params["resourceArn"], ":session/") {
			return false
		}
	}

	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock"); !ok {
		respondBedrockAgentError(w, &bedrockAgentError{status: status, code: code, msg: msg})
		return true
	}

	in, err := decodeBedrockAgentBody(r)
	if err != nil {
		respondBedrockAgentError(w, baValidation("invalid JSON body"))
		return true
	}
	for k, v := range params {
		in[k] = v
	}
	out, apiErr := s.bedrockagent.handle(op.Name, in, r.URL.Query())
	if apiErr != nil {
		respondBedrockAgentError(w, apiErr)
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(op.Status)
	_, _ = w.Write(out)
	return true
}

func isBedrockAgentRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	path := rawRequestPath(r)
	if !hasAnyPrefix(path+"/", bedrockAgentPathPrefixes) {
		return false
	}
	if sigV4ServiceHint(r) == "bedrock" {
		return true
	}
	host := strings.ToLower(r.Host)
	return strings.HasPrefix(host, "bedrock-agent.") || strings.Contains(host, ".bedrock-agent.")
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// matchBedrockAgentRoute finds the operation whose URI template matches. A
// template with fewer variables wins, so POST /flows/validate-definition is
// ValidateFlowDefinition rather than PrepareFlow on a flow of that name.
func matchBedrockAgentRoute(method, path string) (*bedrockagentOperation, map[string]string) {
	var best *bedrockagentOperation
	var bestParams map[string]string
	for i := range bedrockagentOperations {
		op := &bedrockagentOperations[i]
		if op.Method != method {
			continue
		}
		params, ok := bedrockAgentCoreControlPathParams(op.URI, path)
		if !ok {
			continue
		}
		if best == nil || len(params) < len(bestParams) {
			best, bestParams = op, params
		}
	}
	return best, bestParams
}

func decodeBedrockAgentBody(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(body)) == 0 {
		return in, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&in); err != nil {
		return nil, err
	}
	if in == nil {
		in = map[string]any{}
	}
	return in, nil
}

type bedrockAgentError struct {
	status    int
	code, msg string
}

func respondBedrockAgentError(w http.ResponseWriter, e *bedrockAgentError) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_ = json.NewEncoder(w).Encode(map[string]string{"__type": e.code, "message": e.msg})
}

func baValidation(msg string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusBadRequest, code: "ValidationException", msg: msg}
}

func baNotFound(kind, id string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: kind + " " + id + " not found"}
}

func baConflict(msg string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusConflict, code: "ConflictException", msg: msg}
}

// baQueryOrBody reads a member that the model places in the query string for
// some operations and in the body for others.
func baQueryOrBody(in map[string]any, q url.Values, key string) any {
	if v := q.Get(key); v != "" {
		return v
	}
	return in[key]
}
