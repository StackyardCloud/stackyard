package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type cloudControlAPIError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudControlAPIJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudControlAPIJSONCandidate(r) {
		return false
	}

	action := parseCloudControlAPITarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := cloudControlAPIOperationByName[action]; !known {
		respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudcontrolapi")
	if !ok {
		respondCloudControlAPIError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudControlAPIPayload(r)
	if err != nil {
		respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateResource":
		event, err := s.cloudcontrolapi.CreateResource(
			cloudControlAPIString(payload["TypeName"]),
			cloudControlAPIString(payload["DesiredState"]),
			cloudControlAPIString(payload["ClientToken"]),
		)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{"ProgressEvent": cloudControlAPIProgressEventPayload(event)})
		return true

	case "GetResource":
		record, err := s.cloudcontrolapi.GetResource(
			cloudControlAPIString(payload["TypeName"]),
			cloudControlAPIString(payload["Identifier"]),
		)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{
			"TypeName": record.TypeName,
			"ResourceDescription": map[string]any{
				"Identifier": record.Identifier,
				"Properties": record.Properties,
			},
		})
		return true

	case "UpdateResource":
		event, err := s.cloudcontrolapi.UpdateResource(
			cloudControlAPIString(payload["TypeName"]),
			cloudControlAPIString(payload["Identifier"]),
			cloudControlAPIString(payload["PatchDocument"]),
			cloudControlAPIString(payload["ClientToken"]),
		)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{"ProgressEvent": cloudControlAPIProgressEventPayload(event)})
		return true

	case "DeleteResource":
		event, err := s.cloudcontrolapi.DeleteResource(
			cloudControlAPIString(payload["TypeName"]),
			cloudControlAPIString(payload["Identifier"]),
			cloudControlAPIString(payload["ClientToken"]),
		)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{"ProgressEvent": cloudControlAPIProgressEventPayload(event)})
		return true

	case "ListResources":
		maxResults, ok := cloudControlAPIOptionalInt(payload["MaxResults"])
		if !ok {
			respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		resources, nextToken, err := s.cloudcontrolapi.ListResources(
			cloudControlAPIString(payload["TypeName"]),
			maxResults,
			cloudControlAPIString(payload["NextToken"]),
		)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"TypeName": cloudControlAPIString(payload["TypeName"]),
		}
		descriptions := make([]map[string]any, 0, len(resources))
		for _, resource := range resources {
			descriptions = append(descriptions, map[string]any{
				"Identifier": resource.Identifier,
				"Properties": resource.Properties,
			})
		}
		response["ResourceDescriptions"] = descriptions
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondCloudControlAPIJSON(w, http.StatusOK, response)
		return true

	case "GetResourceRequestStatus":
		event, err := s.cloudcontrolapi.GetResourceRequestStatus(cloudControlAPIString(payload["RequestToken"]))
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{
			"ProgressEvent": cloudControlAPIProgressEventPayload(event),
		})
		return true

	case "CancelResourceRequest":
		event, err := s.cloudcontrolapi.CancelResourceRequest(cloudControlAPIString(payload["RequestToken"]))
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		respondCloudControlAPIJSON(w, http.StatusOK, map[string]any{"ProgressEvent": cloudControlAPIProgressEventPayload(event)})
		return true

	case "ListResourceRequests":
		maxResults, ok := cloudControlAPIOptionalInt(payload["MaxResults"])
		if !ok {
			respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		filter, ok := cloudControlAPIResourceRequestFilter(payload["ResourceRequestStatusFilter"])
		if !ok {
			respondCloudControlAPIError(w, http.StatusBadRequest, "ValidationException", "invalid ResourceRequestStatusFilter")
			return true
		}
		requests, nextToken, err := s.cloudcontrolapi.ListResourceRequests(maxResults, cloudControlAPIString(payload["NextToken"]), filter)
		if err != nil {
			respondCloudControlAPIErrorForErr(w, err)
			return true
		}
		summaries := make([]map[string]any, 0, len(requests))
		for _, summary := range requests {
			summaries = append(summaries, map[string]any{
				"Operation":       summary.Operation,
				"OperationStatus": summary.OperationStatus,
				"TypeName":        summary.TypeName,
				"Identifier":      summary.Identifier,
				"RequestToken":    summary.RequestToken,
				"EventTime":       summary.EventTime,
			})
		}
		response := map[string]any{"ResourceRequestStatusSummaries": summaries}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondCloudControlAPIJSON(w, http.StatusOK, response)
		return true
	}

	respondCloudControlAPIError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isCloudControlAPIJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "CloudApiService") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "CloudApiService")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "cloudcontrolapi" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".cloudcontrolapi.") || strings.HasPrefix(host, "cloudcontrolapi.")
}

func parseCloudControlAPITarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "CloudApiService.") {
		return strings.TrimPrefix(target, "CloudApiService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCloudControlAPIPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
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

func respondCloudControlAPIJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudControlAPIError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudControlAPIJSON(w, status, cloudControlAPIError{Type: code, Message: msg})
}

func respondCloudControlAPIErrorForErr(w http.ResponseWriter, err error) {
	if apiErr := asCloudControlAPIErrorInfo(err); apiErr != nil {
		respondCloudControlAPIError(w, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	respondCloudControlAPIError(w, http.StatusInternalServerError, "InternalErrorException", err.Error())
}

func cloudControlAPIString(value any) string {
	if value == nil {
		return ""
	}
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
}

func cloudControlAPIOptionalInt(value any) (int, bool) {
	if value == nil {
		return 0, true
	}
	switch typed := value.(type) {
	case int:
		if typed < 0 {
			return 0, false
		}
		return typed, true
	case int32:
		if typed < 0 {
			return 0, false
		}
		return int(typed), true
	case int64:
		if typed < 0 {
			return 0, false
		}
		return int(typed), true
	case float64:
		if typed < 0 || typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil || parsed < 0 {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func cloudControlAPIStringSlice(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	switch typed := value.(type) {
	case []string:
		return cloneStringSlice(typed), true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			asString, ok := item.(string)
			if !ok {
				return nil, false
			}
			asString = strings.TrimSpace(asString)
			if asString == "" {
				continue
			}
			out = append(out, asString)
		}
		return out, true
	default:
		return nil, false
	}
}

func cloudControlAPIResourceRequestFilter(value any) (cloudControlAPIResourceRequestStatusFilter, bool) {
	if value == nil {
		return cloudControlAPIResourceRequestStatusFilter{}, true
	}
	filterMap, ok := value.(map[string]any)
	if !ok {
		return cloudControlAPIResourceRequestStatusFilter{}, false
	}
	operations, ok := cloudControlAPIStringSlice(filterMap["Operations"])
	if !ok {
		return cloudControlAPIResourceRequestStatusFilter{}, false
	}
	operationStatuses, ok := cloudControlAPIStringSlice(filterMap["OperationStatuses"])
	if !ok {
		return cloudControlAPIResourceRequestStatusFilter{}, false
	}
	return cloudControlAPIResourceRequestStatusFilter{
		TypeName:          cloudControlAPIString(filterMap["TypeName"]),
		Operations:        operations,
		OperationStatuses: operationStatuses,
	}, true
}

func cloudControlAPIProgressEventPayload(event cloudControlAPIProgressEvent) map[string]any {
	out := map[string]any{
		"TypeName":        event.TypeName,
		"Identifier":      event.Identifier,
		"RequestToken":    event.RequestToken,
		"Operation":       event.Operation,
		"OperationStatus": event.OperationStatus,
		"EventTime":       event.EventTime,
	}
	if strings.TrimSpace(event.ResourceModel) != "" {
		out["ResourceModel"] = event.ResourceModel
	}
	if strings.TrimSpace(event.StatusMessage) != "" {
		out["StatusMessage"] = event.StatusMessage
	}
	if strings.TrimSpace(event.ErrorCode) != "" {
		out["ErrorCode"] = event.ErrorCode
	}
	return out
}
