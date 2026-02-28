package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	dsqlsvc "github.com/stackyard/stackyard/internal/services/dsql"
)

type dsqlError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

type dsqlCreateClusterRequest struct {
	ClientToken               string                         `json:"clientToken"`
	DeletionProtectionEnabled *bool                          `json:"deletionProtectionEnabled"`
	Identifier                string                         `json:"identifier"`
	KmsEncryptionKey          string                         `json:"kmsEncryptionKey"`
	MultiRegionProperties     *dsqlsvc.MultiRegionProperties `json:"multiRegionProperties"`
	Tags                      map[string]string              `json:"tags"`
}

type dsqlUpdateClusterRequest struct {
	ClientToken               string `json:"clientToken"`
	DeletionProtectionEnabled *bool  `json:"deletionProtectionEnabled"`
}

type dsqlPutClusterPolicyRequest struct {
	ClientToken string `json:"clientToken"`
	Policy      string `json:"policy"`
}

type dsqlTagResourceRequest struct {
	Tags map[string]string `json:"tags"`
}

func (s *Server) handleDSQLRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDSQLRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "dsql")
	if !ok {
		respondDSQLError(w, status, code, msg)
		return true
	}

	if s.handleDSQLStage0To2(w, r) {
		return true
	}
	if s.handleDSQLStage3To5(w, r) {
		return true
	}

	respondDSQLError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func (s *Server) handleDSQLStage0To2(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 1 && segments[0] == "cluster" {
		switch r.Method {
		case http.MethodPost:
			var req dsqlCreateClusterRequest
			if err := decodeDSQLBody(r, &req); err != nil {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return true
			}

			cluster, err := s.dsql.CreateCluster(dsqlsvc.CreateClusterInput{
				Identifier:                strings.TrimSpace(req.Identifier),
				ClientToken:               strings.TrimSpace(req.ClientToken),
				DeletionProtectionEnabled: req.DeletionProtectionEnabled,
				KmsEncryptionKey:          strings.TrimSpace(req.KmsEncryptionKey),
				MultiRegionProperties:     req.MultiRegionProperties,
				Tags:                      req.Tags,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, cluster)
			return true
		case http.MethodGet:
			maxResults, err := parseOptionalDSQLMaxResults(firstNonEmptyQuery(r.URL.Query(), "max-results", "maxResults"))
			if err != nil {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "invalid max-results")
				return true
			}
			nextToken := strings.TrimSpace(firstNonEmptyQuery(r.URL.Query(), "next-token", "nextToken"))
			clusters, outNextToken, err := s.dsql.ListClusters(dsqlsvc.ListClustersInput{
				MaxResults: maxResults,
				NextToken:  nextToken,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			out := map[string]any{"clusters": clusters}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondDSQLJSON(w, http.StatusOK, out)
			return true
		default:
			return false
		}
	}

	if segments[0] == "cluster" && len(segments) == 2 {
		identifier, ok := decodeDSQLPathSegment(segments[1])
		if !ok {
			respondDSQLError(w, http.StatusBadRequest, "ValidationException", "identifier is required")
			return true
		}

		switch r.Method {
		case http.MethodGet:
			cluster, err := s.dsql.GetCluster(identifier)
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, cluster)
			return true
		case http.MethodPost:
			var req dsqlUpdateClusterRequest
			if err := decodeDSQLBody(r, &req); err != nil {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return true
			}
			cluster, err := s.dsql.UpdateCluster(dsqlsvc.UpdateClusterInput{
				Identifier:                identifier,
				ClientToken:               strings.TrimSpace(req.ClientToken),
				DeletionProtectionEnabled: req.DeletionProtectionEnabled,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, cluster)
			return true
		case http.MethodDelete:
			clientToken := strings.TrimSpace(firstNonEmptyQuery(r.URL.Query(), "client-token", "clientToken"))
			cluster, err := s.dsql.DeleteCluster(dsqlsvc.DeleteClusterInput{
				Identifier:  identifier,
				ClientToken: clientToken,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, cluster)
			return true
		default:
			return false
		}
	}

	if len(segments) == 3 && segments[0] == "clusters" && segments[2] == "vpc-endpoint-service-name" && r.Method == http.MethodGet {
		identifier, ok := decodeDSQLPathSegment(segments[1])
		if !ok {
			respondDSQLError(w, http.StatusBadRequest, "ValidationException", "identifier is required")
			return true
		}
		serviceName, err := s.dsql.GetVpcEndpointServiceName(identifier)
		if err != nil {
			respondDSQLErrorForErr(w, err)
			return true
		}
		respondDSQLJSON(w, http.StatusOK, map[string]any{
			"vpcEndpointServiceName": serviceName,
		})
		return true
	}

	return false
}

func (s *Server) handleDSQLStage3To5(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 3 && segments[0] == "cluster" && segments[2] == "policy" {
		identifier, ok := decodeDSQLPathSegment(segments[1])
		if !ok {
			respondDSQLError(w, http.StatusBadRequest, "ValidationException", "identifier is required")
			return true
		}

		switch r.Method {
		case http.MethodPost:
			var req dsqlPutClusterPolicyRequest
			if err := decodeDSQLBody(r, &req); err != nil {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return true
			}
			if !json.Valid([]byte(strings.TrimSpace(req.Policy))) {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "policy must be valid JSON")
				return true
			}
			policy, err := s.dsql.PutClusterPolicy(dsqlsvc.PutClusterPolicyInput{
				Identifier:  identifier,
				ClientToken: strings.TrimSpace(req.ClientToken),
				Policy:      strings.TrimSpace(req.Policy),
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, policy)
			return true
		case http.MethodGet:
			policy, err := s.dsql.GetClusterPolicy(dsqlsvc.GetClusterPolicyInput{Identifier: identifier})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, policy)
			return true
		case http.MethodDelete:
			out, err := s.dsql.DeleteClusterPolicy(dsqlsvc.DeleteClusterPolicyInput{
				Identifier:            identifier,
				ClientToken:           strings.TrimSpace(firstNonEmptyQuery(r.URL.Query(), "client-token", "clientToken")),
				ExpectedPolicyVersion: strings.TrimSpace(firstNonEmptyQuery(r.URL.Query(), "expected-policy-version", "expectedPolicyVersion")),
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, out)
			return true
		default:
			return false
		}
	}

	if len(segments) == 2 && segments[0] == "tags" {
		resourceARN, ok := decodeDSQLPathSegment(segments[1])
		if !ok {
			respondDSQLError(w, http.StatusBadRequest, "ValidationException", "resource ARN is required")
			return true
		}
		switch r.Method {
		case http.MethodPost:
			var req dsqlTagResourceRequest
			if err := decodeDSQLBody(r, &req); err != nil {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return true
			}
			if len(req.Tags) == 0 {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "tags are required")
				return true
			}
			tags, err := s.dsql.TagResource(dsqlsvc.TagResourceInput{
				ResourceARN: resourceARN,
				Tags:        req.Tags,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, map[string]any{"tags": tags})
			return true
		case http.MethodGet:
			tags, err := s.dsql.ListTagsForResource(dsqlsvc.ListTagsForResourceInput{ResourceARN: resourceARN})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, map[string]any{"tags": tags})
			return true
		case http.MethodDelete:
			tagKeys := collectDSQLTagKeys(r.URL.Query())
			if len(tagKeys) == 0 {
				respondDSQLError(w, http.StatusBadRequest, "ValidationException", "tagKeys are required")
				return true
			}
			tags, err := s.dsql.UntagResource(dsqlsvc.UntagResourceInput{
				ResourceARN: resourceARN,
				TagKeys:     tagKeys,
			})
			if err != nil {
				respondDSQLErrorForErr(w, err)
				return true
			}
			respondDSQLJSON(w, http.StatusOK, map[string]any{"tags": tags})
			return true
		default:
			return false
		}
	}

	return false
}

func isDSQLRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "dsql" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	isDSQLHost := strings.Contains(host, ".dsql.") || strings.HasPrefix(host, "dsql.")
	path := rawRequestPath(r)

	prefixes := []string{"/cluster", "/clusters", "/tags"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return service == "dsql" || isDSQLHost
		}
	}

	if service == "dsql" {
		return true
	}
	return isDSQLHost
}

func decodeDSQLBody(r *http.Request, out any) error {
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		bodyBytes = []byte("{}")
	}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		return err
	}
	return nil
}

func decodeDSQLPathSegment(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func parseOptionalDSQLMaxResults(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil || n < 1 || n > 100 {
		return 0, errors.New("invalid max results")
	}
	return n, nil
}

func firstNonEmptyQuery(values url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func respondDSQLJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDSQLError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDSQLJSON(w, status, dsqlError{Type: code, Message: msg})
}

func respondDSQLErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dsqlsvc.ErrInvalidParameter):
		respondDSQLError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, dsqlsvc.ErrNotFound), errors.Is(err, dsqlsvc.ErrPolicyNotFound):
		respondDSQLError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	case errors.Is(err, dsqlsvc.ErrAlreadyExists), errors.Is(err, dsqlsvc.ErrConflict):
		respondDSQLError(w, http.StatusConflict, "ConflictException", err.Error())
	default:
		respondDSQLError(w, http.StatusInternalServerError, "InternalServerException", err.Error())
	}
}

func collectDSQLTagKeys(values url.Values) []string {
	keys := make([]string, 0)
	seen := map[string]struct{}{}

	appendKey := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		keys = append(keys, trimmed)
	}

	for _, key := range values["tagKeys"] {
		for _, part := range strings.Split(key, ",") {
			appendKey(part)
		}
	}
	for _, key := range values["tag-keys"] {
		for _, part := range strings.Split(key, ",") {
			appendKey(part)
		}
	}
	for name, entries := range values {
		if !strings.HasPrefix(name, "tagKeys.") {
			continue
		}
		for _, value := range entries {
			appendKey(value)
		}
	}
	return keys
}
