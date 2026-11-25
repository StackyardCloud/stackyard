package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/sesv2"
)

// Amazon Pinpoint Email: REST JSON under /v1/email, signed ses. It is the
// predecessor of SESv2 and shares its account state: identities,
// configuration sets and their event destinations, account sending, the
// send quota and resource tags live in the SESv2 service. Pinpoint Email
// adds configuration set options, dedicated IPs and pools, the
// deliverability dashboard and test reports (pinpointemail_store.go).

const ppeMaxBody = 2 << 20

type ppeErr struct {
	status    int
	code, msg string
}

func ppeError(status int, code, format string, args ...any) *ppeErr {
	return &ppeErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func ppeBadRequest(format string, args ...any) *ppeErr {
	return ppeError(http.StatusBadRequest, "BadRequestException", format, args...)
}

func ppeNotFound(format string, args ...any) *ppeErr {
	return ppeError(http.StatusNotFound, "NotFoundException", format, args...)
}

func ppeLimit(format string, args ...any) *ppeErr {
	return ppeError(http.StatusBadRequest, "LimitExceededException", format, args...)
}

// ppeFromSESv2 maps a SESv2 service error to its Pinpoint Email shape.
func ppeFromSESv2(err error, what string) *ppeErr {
	switch err {
	case nil:
		return nil
	case sesv2.ErrAlreadyExists:
		return ppeError(http.StatusBadRequest, "AlreadyExistsException", "%s already exists.", what)
	case sesv2.ErrNotFound:
		return ppeNotFound("%s doesn't exist.", what)
	case sesv2.ErrSendingDisabled:
		return ppeError(http.StatusBadRequest, "SendingPausedException", "Sending is paused for this account.")
	}
	return ppeBadRequest("%s is not valid.", what)
}

type ppeHandler func(s *Server, in map[string]any) (map[string]any, *ppeErr)

func (s *Server) handlePinpointEmailRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/v1/email/") {
		return false
	}
	hint := sigV4ServiceHint(r)
	if hint != "ses" && (hint != "" || !strings.HasPrefix(strings.ToLower(r.Host), "email.")) {
		return false
	}
	op, params, ok := matchSESV2OperationIn(pinpointEmailOperations, r.Method, r.URL.EscapedPath())
	if !ok {
		return false // e.g. SESv2's /v1/email/identities/{id}/dkim/signing
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > ppeMaxBody {
		respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", "request body must declare a Content-Length of at most 2 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "ses"); !ok {
		respondSESV2Error(w, status, code, msg)
		return true
	}
	model := awsmodels.PinpointEmailModel()
	in, problem := awsModelDecodeREST(r, model, model.Operations[op.Name].Input, params)
	if problem != "" {
		respondSESV2Error(w, http.StatusBadRequest, "BadRequestException", problem)
		return true
	}
	out, perr := ppeHandlers[op.Name](s, in)
	if perr != nil {
		respondSESV2Error(w, perr.status, perr.code, perr.msg)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	body, _ := json.Marshal(model.Conform(model.Operations[op.Name].Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// awsModelDecodeREST decodes a REST JSON request: the body through
// awsModelDecode, then URI and query string members bound from params and
// r's query, typed by the model. It returns a problem description on failure.
func awsModelDecodeREST(r *http.Request, model *awsmodels.Model, shape string, params map[string]string) (map[string]any, string) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, awsModelBadJSON
	}
	in, problem := awsModelDecode(raw, model, shape)
	if problem != "" {
		return nil, problem
	}
	s := model.Shapes[shape]
	if s == nil {
		return in, ""
	}
	query := r.URL.Query()
	for name, ref := range s.Members {
		var values []string
		switch ref.Location {
		case "uri":
			if v, ok := params[ref.LocationName]; ok {
				values = []string{v}
			}
		case "querystring":
			values = query[ref.LocationName]
		default:
			continue
		}
		if len(values) == 0 || values[0] == "" {
			if slices.Contains(s.Required, name) {
				return nil, "missing required parameter " + ref.LocationName
			}
			continue
		}
		v, problem := awsModelScalar(model, ref.Shape, values)
		if problem != "" {
			return nil, ref.LocationName + ": " + problem
		}
		if p := model.Validate(ref.Shape, v); len(p) > 0 {
			return nil, ref.LocationName + ": " + strings.Join(p, "; ")
		}
		in[name] = v
	}
	return in, ""
}

func awsModelScalar(model *awsmodels.Model, shape string, values []string) (any, string) {
	s := model.Shapes[shape]
	switch {
	case s == nil:
		return values[0], ""
	case s.Type == "list":
		out := make([]any, 0, len(values))
		for _, v := range values {
			out = append(out, v)
		}
		return out, ""
	case s.Type == "integer" || s.Type == "long":
		n, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			return nil, "must be an integer"
		}
		return float64(n), ""
	case s.Type == "timestamp":
		if t, err := time.Parse(time.RFC3339Nano, values[0]); err == nil {
			return float64(t.UnixMilli()) / 1000, ""
		}
		if n, err := strconv.ParseFloat(values[0], 64); err == nil {
			return n, ""
		}
		return nil, "must be a timestamp"
	case s.Type == "boolean":
		b, err := strconv.ParseBool(values[0])
		if err != nil {
			return nil, "must be a boolean"
		}
		return b, ""
	}
	return values[0], ""
}
