package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Service Catalog: awsJson1_1, X-Amz-Target AWS242ServiceCatalogService.*,
// signed servicecatalog. The CLI 2.41 model lists smithy-rpc-v2-cbor first,
// but the CLI sends JSON 1.1, so that is what is served.

const (
	svcatTargetPrefix = "AWS242ServiceCatalogService."
	svcatMaxBody      = 1 << 20
)

func (s *Server) handleServiceCatalogJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), svcatTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > svcatMaxBody {
		respondServiceCatalogError(w, http.StatusBadRequest, "InvalidParametersException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "servicecatalog"); !ok {
		respondServiceCatalogError(w, status, code, msg)
		return true
	}
	if _, known := serviceCatalogOperationByName[op]; !known {
		respondServiceCatalogError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	in, err := decodeServiceCatalogBody(r)
	if err != nil {
		respondServiceCatalogError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	model := awsmodels.ServiceCatalogModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondServiceCatalogError(w, http.StatusBadRequest, "InvalidParametersException", strings.Join(problems, "; "))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, rerr := s.serviceCatalog.handle(s, r, op, shapes.Output, conformed)
	if rerr != nil {
		respondServiceCatalogError(w, http.StatusBadRequest, rerr.code, rerr.msg)
		return true
	}
	// out was shaped to the output model under the store lock.
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func decodeServiceCatalogBody(r *http.Request) (map[string]any, error) {
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

func respondServiceCatalogError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
