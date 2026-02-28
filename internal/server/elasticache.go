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

const elastiCacheNamespace = "http://elasticache.amazonaws.com/doc/2015-02-02/"

func (s *Server) handleElastiCacheQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElastiCacheQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticache")
	if !ok {
		respondElastiCacheErrorXML(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondElastiCacheErrorXML(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondElastiCacheErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondElastiCacheErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, ok := elastiCacheOperationByName[action]; !ok {
		respondElastiCacheErrorXML(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2015-02-02" {
		respondElastiCacheErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	result := s.elasticache.Handle(action, r.Form)
	respondElastiCacheXML(w, http.StatusOK, action, result)
	return true
}

func isElastiCacheQueryCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "elasticache" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".elasticache.") || strings.HasPrefix(host, "elasticache.") {
		return true
	}

	path := strings.TrimSpace(r.URL.Path)
	if strings.HasPrefix(path, "/elasticache") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#elasticache") || strings.Contains(userAgent, " elasticache/") {
		return true
	}
	amzUserAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-User-Agent")))
	if strings.Contains(amzUserAgent, "command#elasticache") || strings.Contains(amzUserAgent, " elasticache/") {
		return true
	}

	if action := strings.TrimSpace(r.URL.Query().Get("Action")); action != "" {
		if _, ok := elastiCacheOperationByName[action]; !ok {
			return false
		}
		version := strings.TrimSpace(r.URL.Query().Get("Version"))
		if service == "elasticache" {
			return version == "" || version == "2015-02-02"
		}
		return version == "2015-02-02"
	}

	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}

	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return false
	}
	action := strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return false
	}
	if _, ok := elastiCacheOperationByName[action]; !ok {
		return false
	}
	version := strings.TrimSpace(values.Get("Version"))
	if service == "elasticache" {
		return version == "" || version == "2015-02-02"
	}
	return version == "2015-02-02"
}

func respondElastiCacheXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, elastiCacheNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeElastiCacheMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondElastiCacheErrorXML(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", elastiCacheNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeElastiCacheTextElement(&buf, "Code", code)
	writeElastiCacheTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeElastiCacheMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeElastiCacheValue(buf, key, values[key])
	}
}

func writeElastiCacheValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeElastiCacheTextElement(buf, name, v)
	case bool:
		if v {
			writeElastiCacheTextElement(buf, name, "true")
		} else {
			writeElastiCacheTextElement(buf, name, "false")
		}
	case time.Time:
		writeElastiCacheTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeElastiCacheMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeElastiCacheValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeElastiCacheValue(buf, name, items)
	default:
		writeElastiCacheTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeElastiCacheTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
