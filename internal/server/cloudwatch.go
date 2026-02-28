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

const cloudWatchNamespace = "http://monitoring.amazonaws.com/doc/2010-08-01/"

func (s *Server) handleCloudWatchQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "monitoring")
	if !ok {
		respondCloudWatchError(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondCloudWatchError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondCloudWatchError(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondCloudWatchError(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, ok := cloudWatchOperationByName[action]; !ok {
		respondCloudWatchError(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2010-08-01" {
		respondCloudWatchError(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	result := s.cloudwatch.Handle(action, r.Form)
	respondCloudWatchXML(w, http.StatusOK, action, result)
	return true
}

func isCloudWatchQueryCandidate(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "cloudwatch") || strings.Contains(host, "monitoring") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/cloudwatch") || strings.HasPrefix(r.URL.Path, "/monitoring") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "monitoring" && service != "cloudwatch" {
		return false
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if service == "" {
			if _, ok := cloudWatchOperationByName[action]; !ok {
				return false
			}
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2010-08-01" {
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
	if service == "" {
		if _, ok := cloudWatchOperationByName[action]; !ok {
			return false
		}
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2010-08-01" {
		return false
	}
	return true
}

func respondCloudWatchXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, cloudWatchNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeCloudWatchMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondCloudWatchError(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", cloudWatchNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeCloudWatchTextElement(&buf, "Code", code)
	writeCloudWatchTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeCloudWatchMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeCloudWatchValue(buf, key, values[key])
	}
}

func writeCloudWatchValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeCloudWatchTextElement(buf, name, v)
	case bool:
		if v {
			writeCloudWatchTextElement(buf, name, "true")
		} else {
			writeCloudWatchTextElement(buf, name, "false")
		}
	case time.Time:
		writeCloudWatchTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeCloudWatchMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeCloudWatchValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeCloudWatchValue(buf, name, items)
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeCloudWatchValue(buf, name, items)
	default:
		writeCloudWatchTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeCloudWatchTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
