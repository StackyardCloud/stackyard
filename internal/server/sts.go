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

const stsNamespace = "https://sts.amazonaws.com/doc/2011-06-15/"

func (s *Server) handleSTSQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSTSQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sts")
	if !ok {
		respondSTSError(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondSTSError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondSTSError(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondSTSError(w, http.StatusBadRequest, "MissingAction", "Action is required")
		return true
	}
	if _, ok := stsOperationByName[action]; !ok {
		respondSTSError(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2011-06-15" {
		respondSTSError(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	result := s.sts.Handle(action, r.Form)
	respondSTSXML(w, http.StatusOK, action, result)
	return true
}

func isSTSQueryCandidate(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".sts.") || strings.HasPrefix(host, "sts.") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "sts" {
		return false
	}
	if service == "sts" {
		return true
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	version := strings.TrimSpace(r.URL.Query().Get("Version"))
	if action != "" {
		if _, ok := stsOperationByName[action]; !ok {
			return false
		}
		if version != "" && version != "2011-06-15" {
			return false
		}
		return true
	}

	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}

	body, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return false
	}
	action = strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return false
	}
	if _, ok := stsOperationByName[action]; !ok {
		return false
	}
	version = strings.TrimSpace(values.Get("Version"))
	return version == "" || version == "2011-06-15"
}

func respondSTSXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, stsNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeSTSMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondSTSError(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", stsNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeSTSTextElement(&buf, "Code", code)
	writeSTSTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeSTSMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeSTSValue(buf, key, values[key])
	}
}

func writeSTSValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeSTSTextElement(buf, name, v)
	case bool:
		if v {
			writeSTSTextElement(buf, name, "true")
		} else {
			writeSTSTextElement(buf, name, "false")
		}
	case time.Time:
		writeSTSTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeSTSMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeSTSValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeSTSValue(buf, name, items)
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeSTSValue(buf, name, items)
	default:
		writeSTSTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeSTSTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
