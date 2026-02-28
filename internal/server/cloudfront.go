package server

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"
)

type cloudFrontErrorEnvelope struct {
	XMLName   xml.Name            `xml:"ErrorResponse"`
	Xmlns     string              `xml:"xmlns,attr,omitempty"`
	Error     cloudFrontErrorBody `xml:"Error"`
	RequestID string              `xml:"RequestId"`
}

type cloudFrontErrorBody struct {
	Type    string `xml:"Type,omitempty"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func (s *Server) handleCloudFrontRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudFrontRESTRouterCandidate(r) {
		return false
	}

	action := parseCloudFrontOperation(r)
	if action == "" {
		respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "unable to determine CloudFront action")
		return true
	}
	if _, known := cloudFrontOperationByName[action]; !known {
		respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "unknown CloudFront action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudfront")
	if !ok {
		respondCloudFrontError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudFrontPayload(r)
	if err != nil {
		respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "invalid request body")
		return true
	}

	pathParams := cloudFrontPathParams(rawRequestPath(r))
	result := s.cloudfront.Handle(action, payload, pathParams)
	if result.Status == 0 {
		result.Status = http.StatusOK
	}
	for key, value := range result.Headers {
		w.Header().Set(key, value)
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(result.Status)
	_, _ = w.Write(result.Body)
	return true
}

func isCloudFrontRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "cloudfront" {
		return false
	}
	if service == "cloudfront" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "cloudfront") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#cloudfront") || strings.Contains(userAgent, " cloudfront/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/2020-05-31")
}

func parseCloudFrontOperation(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Stackyard-Operation")); value != "" {
		return value
	}

	if target := strings.TrimSpace(r.Header.Get("X-Amz-Target")); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
			return strings.TrimSpace(target[dot+1:])
		}
		return target
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	path = strings.TrimSuffix(path, "/")

	if path == "/2020-05-31/distribution" {
		switch method {
		case http.MethodGet:
			return "ListDistributions"
		case http.MethodPost:
			return "CreateDistribution"
		}
	}
	if strings.HasPrefix(path, "/2020-05-31/distribution/") {
		rel := strings.TrimPrefix(path, "/2020-05-31/distribution/")
		parts := strings.Split(rel, "/")
		if len(parts) == 1 && parts[0] != "" {
			switch method {
			case http.MethodGet:
				return "GetDistribution"
			case http.MethodDelete:
				return "DeleteDistribution"
			case http.MethodPut:
				return "UpdateDistribution"
			}
		}
		if len(parts) == 2 && parts[1] == "config" {
			switch method {
			case http.MethodGet:
				return "GetDistributionConfig"
			case http.MethodPut:
				return "UpdateDistribution"
			}
		}
		if len(parts) == 2 && parts[1] == "invalidation" {
			switch method {
			case http.MethodGet:
				return "ListInvalidations"
			case http.MethodPost:
				return "CreateInvalidation"
			}
		}
		if len(parts) == 3 && parts[1] == "invalidation" {
			if method == http.MethodGet {
				return "GetInvalidation"
			}
		}
	}

	if method == http.MethodGet {
		return "ListDistributions"
	}
	return "CreateDistribution"
}

func cloudFrontPathParams(requestPath string) map[string]string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	path = strings.TrimSuffix(path, "/")
	params := map[string]string{}

	if strings.HasPrefix(path, "/2020-05-31/distribution/") {
		rel := strings.TrimPrefix(path, "/2020-05-31/distribution/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 && parts[0] != "" {
			params["distributionId"] = parts[0]
		}
		if len(parts) > 2 && parts[1] == "invalidation" && parts[2] != "" {
			params["invalidationId"] = parts[2]
		}
	}

	return params
}

func parseCloudFrontPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "json") {
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

	return map[string]any{"RawBody": string(body)}, nil
}

func respondCloudFrontError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	env := cloudFrontErrorEnvelope{
		Xmlns: "http://cloudfront.amazonaws.com/doc/2020-05-31/",
		Error: cloudFrontErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: msg,
		},
		RequestID: "stackyard-request",
	}
	_ = xml.NewEncoder(w).Encode(env)
}
