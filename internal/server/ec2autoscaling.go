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

const ec2AutoScalingNamespace = "http://autoscaling.amazonaws.com/doc/2011-01-01/"

func (s *Server) handleEC2AutoScalingQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEC2AutoScalingQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "autoscaling")
	if !ok {
		respondEC2AutoScalingError(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondEC2AutoScalingError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondEC2AutoScalingError(w, http.StatusBadRequest, "ValidationError", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondEC2AutoScalingError(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, ok := ec2AutoScalingOperationByName[action]; !ok {
		respondEC2AutoScalingError(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2011-01-01" {
		respondEC2AutoScalingError(w, http.StatusBadRequest, "ValidationError", "unsupported Version")
		return true
	}

	result := s.ec2autoscaling.Handle(action, r.Form)
	respondEC2AutoScalingXML(w, http.StatusOK, action, result)
	return true
}

func isEC2AutoScalingQueryCandidate(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "autoscaling") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/autoscaling") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "autoscaling" {
		return false
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if service == "" {
			if _, ok := ec2AutoScalingOperationByName[action]; !ok {
				return false
			}
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2011-01-01" {
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
		if _, ok := ec2AutoScalingOperationByName[action]; !ok {
			return false
		}
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2011-01-01" {
		return false
	}
	return true
}

func respondEC2AutoScalingXML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, ec2AutoScalingNamespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeEC2AutoScalingMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondEC2AutoScalingError(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", ec2AutoScalingNamespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeEC2AutoScalingTextElement(&buf, "Code", code)
	writeEC2AutoScalingTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeEC2AutoScalingMap(buf *bytes.Buffer, values map[string]any) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeEC2AutoScalingValue(buf, key, values[key])
	}
}

func writeEC2AutoScalingValue(buf *bytes.Buffer, name string, value any) {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "<%s/>", name)
	case string:
		writeEC2AutoScalingTextElement(buf, name, v)
	case bool:
		if v {
			writeEC2AutoScalingTextElement(buf, name, "true")
		} else {
			writeEC2AutoScalingTextElement(buf, name, "false")
		}
	case int:
		writeEC2AutoScalingTextElement(buf, name, fmt.Sprintf("%d", v))
	case int64:
		writeEC2AutoScalingTextElement(buf, name, fmt.Sprintf("%d", v))
	case float64:
		writeEC2AutoScalingTextElement(buf, name, fmt.Sprintf("%v", v))
	case time.Time:
		writeEC2AutoScalingTextElement(buf, name, v.UTC().Format(time.RFC3339))
	case map[string]any:
		fmt.Fprintf(buf, "<%s>", name)
		writeEC2AutoScalingMap(buf, v)
		fmt.Fprintf(buf, "</%s>", name)
	case []any:
		fmt.Fprintf(buf, "<%s>", name)
		for _, item := range v {
			writeEC2AutoScalingValue(buf, "member", item)
		}
		fmt.Fprintf(buf, "</%s>", name)
	case []string:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeEC2AutoScalingValue(buf, name, items)
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		writeEC2AutoScalingValue(buf, name, items)
	default:
		writeEC2AutoScalingTextElement(buf, name, fmt.Sprintf("%v", v))
	}
}

func writeEC2AutoScalingTextElement(buf *bytes.Buffer, name, value string) {
	fmt.Fprintf(buf, "<%s>", name)
	_ = xml.EscapeText(buf, []byte(value))
	fmt.Fprintf(buf, "</%s>", name)
}
