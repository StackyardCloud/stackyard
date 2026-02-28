package server

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const elasticLoadBalancingNamespace = "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/"

func (s *Server) handleElasticLoadBalancingQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElasticLoadBalancingQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticloadbalancing")
	if !ok {
		respondElasticLoadBalancingError(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondElasticLoadBalancingError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondElasticLoadBalancingError(w, http.StatusBadRequest, "ValidationError", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondElasticLoadBalancingError(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, exists := elasticLoadBalancingOperationByName[action]; !exists {
		respondElasticLoadBalancingError(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2015-12-01" {
		respondElasticLoadBalancingError(w, http.StatusBadRequest, "ValidationError", "unsupported Version")
		return true
	}

	result := s.elasticloadbalancing.Handle(action, r.Form)
	respondElasticLoadBalancingXML(w, http.StatusOK, action, result)
	return true
}

func isElasticLoadBalancingQueryCandidate(r *http.Request) bool {
	return isElasticLoadBalancingQueryCandidateForVersion(
		r,
		"2015-12-01",
		func(action string) bool {
			_, ok := elasticLoadBalancingOperationByName[action]
			return ok
		},
	)
}

func isElasticLoadBalancingQueryCandidateForVersion(r *http.Request, version string, opKnown func(string) bool) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	hostHint := strings.Contains(host, "elasticloadbalancing") || strings.Contains(host, "elbv2")
	pathHint := strings.HasPrefix(r.URL.Path, "/elasticloadbalancing") || strings.HasPrefix(r.URL.Path, "/elbv2")
	hinted := hostHint || pathHint

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "elasticloadbalancing" {
		return false
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	requestVersion := strings.TrimSpace(r.URL.Query().Get("Version"))

	if action == "" && r.Method == http.MethodPost &&
		strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		body, err := readBodyBytes(r)
		if err != nil {
			return false
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return false
		}
		action = strings.TrimSpace(values.Get("Action"))
		if requestVersion == "" {
			requestVersion = strings.TrimSpace(values.Get("Version"))
		}
	}

	if requestVersion != "" && requestVersion != version {
		return false
	}

	if action != "" {
		if opKnown(action) {
			return true
		}
		return service == "elasticloadbalancing" || hinted
	}

	return service == "elasticloadbalancing" || hinted
}

func respondElasticLoadBalancingXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, elasticLoadBalancingNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeElasticLoadBalancingMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondElasticLoadBalancingError(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", elasticLoadBalancingNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeElasticLoadBalancingTextElement(&buf, "Code", code)
	writeElasticLoadBalancingTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeElasticLoadBalancingMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeElasticLoadBalancingValue(buf, key, values[key])
	}
}

func writeElasticLoadBalancingValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeElasticLoadBalancingTextElement(buf, name, v)
	case bool:
		if v {
			writeElasticLoadBalancingTextElement(buf, name, "true")
		} else {
			writeElasticLoadBalancingTextElement(buf, name, "false")
		}
	case int:
		writeElasticLoadBalancingTextElement(buf, name, fmt.Sprintf("%d", v))
	case int64:
		writeElasticLoadBalancingTextElement(buf, name, fmt.Sprintf("%d", v))
	case float64:
		writeElasticLoadBalancingTextElement(buf, name, fmt.Sprintf("%v", v))
	case time.Time:
		writeElasticLoadBalancingTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeElasticLoadBalancingMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeElasticLoadBalancingValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeElasticLoadBalancingValue(buf, name, items)
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeElasticLoadBalancingValue(buf, name, items)
	default:
		writeElasticLoadBalancingTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeElasticLoadBalancingTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
