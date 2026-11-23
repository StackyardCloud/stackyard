package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Route 53 Domains: awsJson1_1, X-Amz-Target Route53Domains_v20140515.*,
// signed route53domains. The CLI 2.41 model lists smithy-rpc-v2-cbor first,
// but the CLI sends JSON 1.1, so that is what is served.

const (
	r53dTargetPrefix = "Route53Domains_v20140515."
	r53dMaxBody      = 256 << 10
)

func (s *Server) handleRoute53DomainsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), r53dTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > r53dMaxBody {
		respondRoute53DomainsError(w, http.StatusBadRequest, "InvalidInput", "request body must declare a Content-Length of at most 256 KiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53domains"); !ok {
		respondRoute53DomainsError(w, status, code, msg)
		return true
	}
	if _, known := route53domainsOperationByName[op]; !known {
		respondRoute53DomainsError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	in, err := decodeRoute53DomainsBody(r)
	if err != nil {
		respondRoute53DomainsError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	model := awsmodels.Route53DomainsModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondRoute53DomainsError(w, http.StatusBadRequest, "InvalidInput", strings.Join(problems, "; "))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, rerr := s.route53Domains.handle(s, op, shapes.Output, conformed)
	if rerr != nil {
		respondRoute53DomainsError(w, http.StatusBadRequest, rerr.code, rerr.msg)
		return true
	}
	// out was shaped to the output model under the store lock.
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func decodeRoute53DomainsBody(r *http.Request) (map[string]any, error) {
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

func respondRoute53DomainsError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
