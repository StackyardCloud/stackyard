package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type paymentCryptographyDataError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePaymentCryptographyDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPaymentCryptographyDataRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "payment-cryptography")
	if !ok {
		respondPaymentCryptographyDataError(w, status, code, msg)
		return true
	}

	action, pathParams := parsePaymentCryptographyDataRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondPaymentCryptographyDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := paymentCryptographyDataOperationByName[action]; !known {
		respondPaymentCryptographyDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parsePaymentCryptographyDataPayload(r)
	if err != nil {
		respondPaymentCryptographyDataError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	pathParamsAny := map[string]any{}
	for k, v := range pathParams {
		pathParamsAny[k] = v
	}
	response := s.paymentcryptographydata.Handle(action, payload, pathParamsAny)
	if response == nil {
		response = map[string]any{}
	}
	respondPaymentCryptographyDataJSON(w, http.StatusOK, response)
	return true
}

func isPaymentCryptographyDataRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "payment-cryptography" {
		return false
	}
	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	action, _ := parsePaymentCryptographyDataRoute(method, path)
	host := strings.ToLower(strings.TrimSpace(r.Host))
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if service == "payment-cryptography" {
		return action != "" ||
			isPaymentCryptographyDataPathPrefix(path) ||
			strings.Contains(host, "dataplane.payment-cryptography.") ||
			strings.HasPrefix(host, "dataplane.payment-cryptography.") ||
			strings.Contains(userAgent, "command#payment-cryptography-data") ||
			strings.Contains(userAgent, " payment-cryptography-data/")
	}

	if strings.Contains(host, "dataplane.payment-cryptography.") || strings.HasPrefix(host, "dataplane.payment-cryptography.") {
		return true
	}

	if strings.Contains(userAgent, "command#payment-cryptography-data") || strings.Contains(userAgent, " payment-cryptography-data/") {
		return true
	}

	return action != ""
}

func isPaymentCryptographyDataPathPrefix(path string) bool {
	path = strings.ToLower(strings.TrimSpace(strings.SplitN(path, "?", 2)[0]))
	switch {
	case strings.HasPrefix(path, "/keys/"):
		return true
	case strings.HasPrefix(path, "/as2805kekvalidation/"):
		return true
	case strings.HasPrefix(path, "/cardvalidationdata/"):
		return true
	case strings.HasPrefix(path, "/mac/"):
		return true
	case strings.HasPrefix(path, "/macemvpinchange/"):
		return true
	case strings.HasPrefix(path, "/pindata/"):
		return true
	case strings.HasPrefix(path, "/keymaterial/"):
		return true
	case strings.HasPrefix(path, "/cryptogram/"):
		return true
	default:
		return false
	}
}

func parsePaymentCryptographyDataRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range paymentCryptographyDataOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := paymentCryptographyDataPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func paymentCryptographyDataPathParams(template, actual string) (map[string]string, bool) {
	tSegs := paymentCryptographyDataSplitPath(template)
	aSegs := paymentCryptographyDataSplitPath(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
				return nil, false
			}
			value, err := url.PathUnescape(a)
			if err != nil {
				value = a
			}
			params[name] = value
			continue
		}
		if t != a {
			return nil, false
		}
	}
	return params, true
}

func paymentCryptographyDataSplitPath(path string) []string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parsePaymentCryptographyDataPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.Contains(contentType, "json") {
		return map[string]any{}, nil
	}

	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondPaymentCryptographyDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPaymentCryptographyDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPaymentCryptographyDataJSON(w, status, paymentCryptographyDataError{Type: code, Message: msg})
}
