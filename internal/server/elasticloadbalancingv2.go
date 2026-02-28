package server

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

const elasticLoadBalancingV2Namespace = "http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/"

func (s *Server) handleElasticLoadBalancingV2QueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElasticLoadBalancingV2QueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticloadbalancing")
	if !ok {
		respondElasticLoadBalancingV2Error(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondElasticLoadBalancingV2Error(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondElasticLoadBalancingV2Error(w, http.StatusBadRequest, "ValidationError", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondElasticLoadBalancingV2Error(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if _, exists := elasticLoadBalancingV2OperationByName[action]; !exists {
		respondElasticLoadBalancingV2Error(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}

	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2012-06-01" {
		respondElasticLoadBalancingV2Error(w, http.StatusBadRequest, "ValidationError", "unsupported Version")
		return true
	}

	result := s.elasticloadbalancingv2.Handle(action, r.Form)
	respondElasticLoadBalancingV2XML(w, http.StatusOK, action, result)
	return true
}

func isElasticLoadBalancingV2QueryCandidate(r *http.Request) bool {
	return isElasticLoadBalancingQueryCandidateForVersion(
		r,
		"2012-06-01",
		func(action string) bool {
			_, ok := elasticLoadBalancingV2OperationByName[action]
			return ok
		},
	)
}

func respondElasticLoadBalancingV2XML(w http.ResponseWriter, status int, action string, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<%sResponse xmlns=\"%s\">", action, elasticLoadBalancingV2Namespace)
	fmt.Fprintf(&buf, "<%sResult>", action)
	writeElasticLoadBalancingMap(&buf, result)
	fmt.Fprintf(&buf, "</%sResult>", action)
	buf.WriteString("<ResponseMetadata><RequestId>stackyard-request</RequestId></ResponseMetadata>")
	fmt.Fprintf(&buf, "</%sResponse>", action)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func respondElasticLoadBalancingV2Error(w http.ResponseWriter, status int, code, message string) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, "<ErrorResponse xmlns=\"%s\">", elasticLoadBalancingV2Namespace)
	buf.WriteString("<Error><Type>Sender</Type>")
	writeElasticLoadBalancingTextElement(&buf, "Code", code)
	writeElasticLoadBalancingTextElement(&buf, "Message", message)
	buf.WriteString("</Error><RequestId>stackyard-request</RequestId></ErrorResponse>")

	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
