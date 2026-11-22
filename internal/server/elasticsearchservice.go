package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// The legacy Elasticsearch Service API (es 2015-01-01) is a translation layer
// over the OpenSearch store: both APIs see the same domains, as in AWS.

const (
	eslPathPrefix = "/2015-01-01/"
	eslMaxBody    = 1 << 20
)

type eslError struct {
	status int
	code   string
	msg    string
}

func eslErr(status int, code, msg string) *eslError {
	return &eslError{status: status, code: code, msg: msg}
}

func eslValidation(msg string) *eslError {
	return eslErr(http.StatusBadRequest, "ValidationException", msg)
}

// botocore models these as 409 for the es API.
func eslNotFound(msg string) *eslError {
	return eslErr(http.StatusConflict, "ResourceNotFoundException", msg)
}

func eslConflict(code, msg string) *eslError { return eslErr(http.StatusConflict, code, msg) }

func (s *Server) handleElasticsearchServiceRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElasticsearchServiceCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one; r.Body is
	// not reassigned (CodeQL treats that as a fresh request source).
	if r.ContentLength < 0 || r.ContentLength > eslMaxBody {
		respondESLError(w, http.StatusBadRequest, "ValidationException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "es"); !ok {
		respondESLError(w, status, code, msg)
		return true
	}
	op, pathParams := eslMatchRoute(r.Method, rawRequestPath(r))
	if op == "" {
		respondESLError(w, http.StatusBadRequest, "UnknownOperationException", "no Elasticsearch Service operation matches "+r.Method+" "+rawRequestPath(r))
		return true
	}
	in, verr := eslInput(r, op, pathParams)
	if verr != nil {
		respondESLError(w, verr.status, verr.code, verr.msg)
		return true
	}
	out, herr := s.opensearch.eslHandle(op, in)
	if herr != nil {
		respondESLError(w, herr.status, herr.code, herr.msg)
		return true
	}
	respondESLJSON(w, http.StatusOK, out)
	return true
}

// isElasticsearchServiceCandidate claims /2015-01-01/ requests signed for es,
// or unsigned ones sent to an es host. OpenSearch owns /2021-01-01/.
func isElasticsearchServiceCandidate(r *http.Request) bool {
	if !strings.HasPrefix(rawRequestPath(r), eslPathPrefix) {
		return false
	}
	switch hint := strings.TrimSpace(sigV4ServiceHint(r)); hint {
	case "es":
		return true
	case "":
		host := strings.ToLower(r.Host)
		return strings.HasPrefix(host, "es.") || strings.Contains(host, ".es.")
	default:
		return false
	}
}

func eslMatchRoute(method, path string) (string, map[string]string) {
	for _, op := range elasticsearchserviceOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if params, ok := opensearchPathParams(op.URI, path); ok {
			return op.Name, params
		}
	}
	return "", nil
}

// eslInput decodes the body and merges the URI and query members, then checks
// every member against the model.
func eslInput(r *http.Request, op string, pathParams map[string]string) (map[string]any, *eslError) {
	m := awsmodels.ElasticsearchServiceModel()
	in := map[string]any{}
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, eslValidation("unable to read request body")
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &in); err != nil || in == nil {
			return nil, eslValidation("request body must be a JSON object")
		}
	}
	inputShape := m.Operations[op].Input
	if inputShape == "" {
		return in, nil
	}
	if problems := m.Validate(inputShape, in); len(problems) > 0 {
		return nil, eslValidation(strings.Join(problems, "; "))
	}
	shape := m.Shapes[inputShape]
	query := r.URL.Query()
	for name, ref := range shape.Members {
		var raw string
		switch ref.Location {
		case "uri":
			raw = pathParams[ref.LocationName]
		case "querystring":
			raw = query.Get(ref.LocationName)
		default:
			continue
		}
		if raw == "" {
			if slices.Contains(shape.Required, name) {
				return nil, eslValidation("missing required member " + name)
			}
			continue
		}
		var v any = raw
		if t := m.Shapes[ref.Shape]; t != nil && (t.Type == "integer" || t.Type == "long") {
			n, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return nil, eslValidation(name + " must be an integer")
			}
			v = float64(n)
		}
		if problems := m.Validate(ref.Shape, v); len(problems) > 0 {
			return nil, eslValidation(name + ": " + strings.TrimPrefix(problems[0], "input: "))
		}
		in[name] = v
	}
	return in, nil
}

// respondESLJSON writes an es response with a fixed JSON content type.
func respondESLJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondESLError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondESLJSON(w, status, map[string]any{"__type": code, "message": msg})
}
