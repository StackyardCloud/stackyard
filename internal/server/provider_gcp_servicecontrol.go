package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var gcpServiceControlReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPServiceControlRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_servicecontrol(w, r) {
		return true
	}

	path := normalizeGCPServiceControlPath(rawRequestPath(r))
	if !isGCPServiceControlPath(path, hasGCPServiceControlHint(r)) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	if handleGCPServiceControlCheck(w, r, path) {
		return true
	}
	if handleGCPServiceControlReport(w, r, path) {
		return true
	}
	if handleGCPServiceControlAllocateQuota(w, r, path) {
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPServiceControlPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPServiceControlHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "servicecontrol",
		"servicecontrol-apiv1",
		"servicecontrol_apiv1",
		"service-control",
		"service_control",
		"gcp-service-control":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-servicecontrol-apiv1") || strings.Contains(ua, "cloud.google.com/go/servicecontrol")
}

func isGCPServiceControlPath(path string, includeHint bool) bool {
	if _, action, ok := parseGCPServiceControlActionPath(path); ok {
		return action == "check" || action == "report" || action == "allocateQuota"
	}
	if isGCPServiceControlGRPCMethodPath(path) {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/v1/services/") {
		return true
	}
	return false
}

func isGCPServiceControlGRPCMethodPath(path string) bool {
	switch strings.TrimSpace(path) {
	case "/gcp/google.api.servicecontrol.v1.ServiceController/Check",
		"/gcp/google.api.servicecontrol.v1.ServiceController/Report",
		"/gcp/google.api.servicecontrol.v1.QuotaController/AllocateQuota":
		return true
	default:
		return false
	}
}

func handleGCPServiceControlCheck(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, action, ok := parseGCPServiceControlActionPath(path)
	if !ok || action != "check" {
		return false
	}

	body, valid := decodeGCPServiceControlJSONBody(w, r, path)
	if !valid {
		return true
	}
	if !validateGCPServiceControlBodyServiceName(w, path, body, serviceName) {
		return true
	}
	operation := gcpServiceControlBodyMap(body, "operation")
	if len(operation) == 0 {
		respondGCPServiceControlInvalidArgument(w, path, "operation is required")
		return true
	}

	operationID := strings.TrimSpace(gcpServiceControlString(operation, "operationId"))
	if operationID == "" {
		respondGCPServiceControlInvalidArgument(w, path, "operation.operationId is required")
		return true
	}
	startTime := strings.TrimSpace(gcpServiceControlString(operation, "startTime"))
	if startTime == "" {
		respondGCPServiceControlInvalidArgument(w, path, "operation.startTime is required")
		return true
	}
	if _, err := time.Parse(time.RFC3339, startTime); err != nil {
		respondGCPServiceControlInvalidArgument(w, path, "operation.startTime must be RFC3339")
		return true
	}

	consumerID := strings.TrimSpace(gcpServiceControlString(operation, "consumerId"))
	denied := gcpServiceControlDenied(r, consumerID)
	respondJSON(w, http.StatusOK, gcpServiceControlCheckResponse(operationID, denied))
	return true
}

func handleGCPServiceControlReport(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, action, ok := parseGCPServiceControlActionPath(path)
	if !ok || action != "report" {
		return false
	}

	body, valid := decodeGCPServiceControlJSONBody(w, r, path)
	if !valid {
		return true
	}
	if !validateGCPServiceControlBodyServiceName(w, path, body, serviceName) {
		return true
	}

	operations, ok := body["operations"].([]any)
	if !ok || len(operations) == 0 {
		respondGCPServiceControlInvalidArgument(w, path, "operations must include at least one entry")
		return true
	}

	reportErrors := make([]map[string]any, 0)
	for idx, raw := range operations {
		op, ok := raw.(map[string]any)
		if !ok || len(op) == 0 {
			respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d] must be an object", idx))
			return true
		}
		operationID := strings.TrimSpace(gcpServiceControlString(op, "operationId"))
		if operationID == "" {
			respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d].operationId is required", idx))
			return true
		}
		consumerID := strings.TrimSpace(gcpServiceControlString(op, "consumerId"))
		if consumerID == "" {
			respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d].consumerId is required", idx))
			return true
		}
		startTime := strings.TrimSpace(gcpServiceControlString(op, "startTime"))
		if startTime == "" {
			respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d].startTime is required", idx))
			return true
		}
		if _, err := time.Parse(time.RFC3339, startTime); err != nil {
			respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d].startTime must be RFC3339", idx))
			return true
		}
		if endTime := strings.TrimSpace(gcpServiceControlString(op, "endTime")); endTime != "" {
			if _, err := time.Parse(time.RFC3339, endTime); err != nil {
				respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("operations[%d].endTime must be RFC3339", idx))
				return true
			}
		}

		if gcpServiceControlDenied(r, consumerID) {
			reportErrors = append(reportErrors, map[string]any{
				"operationId": operationID,
				"status": map[string]any{
					"code":    7,
					"message": "permission denied by staged emulation",
				},
			})
		}
	}

	respondJSON(w, http.StatusOK, gcpServiceControlReportResponse(reportErrors))
	return true
}

func handleGCPServiceControlAllocateQuota(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, action, ok := parseGCPServiceControlActionPath(path)
	if !ok || action != "allocateQuota" {
		return false
	}

	body, valid := decodeGCPServiceControlJSONBody(w, r, path)
	if !valid {
		return true
	}
	if !validateGCPServiceControlBodyServiceName(w, path, body, serviceName) {
		return true
	}

	allocateOperation := gcpServiceControlBodyMap(body, "allocateOperation")
	if len(allocateOperation) == 0 {
		respondGCPServiceControlInvalidArgument(w, path, "allocateOperation is required")
		return true
	}

	operationID := strings.TrimSpace(gcpServiceControlString(allocateOperation, "operationId"))
	if operationID == "" {
		respondGCPServiceControlInvalidArgument(w, path, "allocateOperation.operationId is required")
		return true
	}
	consumerID := strings.TrimSpace(gcpServiceControlString(allocateOperation, "consumerId"))
	if consumerID == "" {
		respondGCPServiceControlInvalidArgument(w, path, "allocateOperation.consumerId is required")
		return true
	}

	methodName := strings.TrimSpace(gcpServiceControlString(allocateOperation, "methodName"))
	quotaMetrics, _ := allocateOperation["quotaMetrics"].([]any)
	if methodName != "" && len(quotaMetrics) > 0 {
		respondGCPServiceControlInvalidArgument(w, path, "allocateOperation.methodName and allocateOperation.quotaMetrics are mutually exclusive")
		return true
	}
	if methodName == "" && len(quotaMetrics) == 0 {
		respondGCPServiceControlInvalidArgument(w, path, "allocateOperation.methodName or allocateOperation.quotaMetrics is required")
		return true
	}
	if len(quotaMetrics) > 0 {
		for idx, rawMetric := range quotaMetrics {
			metric, ok := rawMetric.(map[string]any)
			if !ok || len(metric) == 0 {
				respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("allocateOperation.quotaMetrics[%d] must be an object", idx))
				return true
			}
			if strings.TrimSpace(gcpServiceControlString(metric, "metricName")) == "" {
				respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("allocateOperation.quotaMetrics[%d].metricName is required", idx))
				return true
			}
			metricValues, _ := metric["metricValues"].([]any)
			if len(metricValues) == 0 {
				respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("allocateOperation.quotaMetrics[%d].metricValues must include at least one entry", idx))
				return true
			}
			for valueIdx, rawValue := range metricValues {
				value, ok := rawValue.(map[string]any)
				if !ok || len(value) == 0 {
					respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("allocateOperation.quotaMetrics[%d].metricValues[%d] must be an object", idx, valueIdx))
					return true
				}
				if !gcpServiceControlHasMetricScalarValue(value) {
					respondGCPServiceControlInvalidArgument(w, path, fmt.Sprintf("allocateOperation.quotaMetrics[%d].metricValues[%d] requires a scalar value", idx, valueIdx))
					return true
				}
			}
		}
	}

	denied := gcpServiceControlDenied(r, consumerID)
	respondJSON(w, http.StatusOK, gcpServiceControlAllocateQuotaResponse(operationID, denied))
	return true
}

func parseGCPServiceControlActionPath(path string) (serviceName, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "services" {
		return "", "", false
	}
	serviceName, action, ok = strings.Cut(strings.TrimSpace(parts[3]), ":")
	if !ok {
		return "", "", false
	}
	serviceName = strings.TrimSpace(serviceName)
	action = strings.TrimSpace(action)
	if serviceName == "" || action == "" {
		return "", "", false
	}
	return serviceName, action, true
}

func decodeGCPServiceControlJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceControlInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceControlInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func validateGCPServiceControlBodyServiceName(w http.ResponseWriter, path string, body map[string]any, expectedService string) bool {
	bodyServiceName := strings.TrimSpace(gcpServiceControlString(body, "serviceName"))
	if bodyServiceName == "" {
		return true
	}
	if bodyServiceName != expectedService {
		respondGCPServiceControlInvalidArgument(w, path, "serviceName must match requested service")
		return false
	}
	return true
}

func gcpServiceControlBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if nested == nil {
		return map[string]any{}
	}
	return nested
}

func gcpServiceControlString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpServiceControlHasMetricScalarValue(metricValue map[string]any) bool {
	if value, ok := metricValue["int64Value"]; ok {
		switch value.(type) {
		case float64, string:
			return true
		}
		return false
	}
	if _, ok := metricValue["doubleValue"]; ok {
		return true
	}
	if _, ok := metricValue["boolValue"]; ok {
		return true
	}
	if _, ok := metricValue["stringValue"]; ok {
		return true
	}
	return false
}

func gcpServiceControlDenied(r *http.Request, consumerID string) bool {
	if strings.TrimSpace(r.URL.Query().Get("deny")) == "1" {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(consumerID)), "deny")
}

func gcpServiceControlCheckResponse(operationID string, denied bool) map[string]any {
	checkErrors := []map[string]any{}
	if denied {
		checkErrors = append(checkErrors, map[string]any{
			"code":    "PERMISSION_DENIED",
			"subject": "operation.consumerId",
			"detail":  "permission denied by staged emulation",
		})
	}
	return map[string]any{
		"operationId":      operationID,
		"serviceConfigId":  "2026-01-01r0",
		"serviceRolloutId": "2026-01-01r0",
		"checkInfo": map[string]any{
			"unusedArguments": []any{},
			"consumerInfo": map[string]any{
				"projectNumber": "1234567890",
			},
		},
		"checkErrors": checkErrors,
	}
}

func gcpServiceControlReportResponse(reportErrors []map[string]any) map[string]any {
	return map[string]any{
		"serviceConfigId":  "2026-01-01r0",
		"serviceRolloutId": "2026-01-01r0",
		"reportErrors":     reportErrors,
	}
}

func gcpServiceControlAllocateQuotaResponse(operationID string, denied bool) map[string]any {
	allocateErrors := []map[string]any{}
	if denied {
		allocateErrors = append(allocateErrors, map[string]any{
			"code":        "RESOURCE_EXHAUSTED",
			"subject":     "allocateOperation.consumerId",
			"description": "quota exhausted by staged emulation",
		})
	}
	return map[string]any{
		"operationId":     operationID,
		"serviceConfigId": "2026-01-01r0",
		"allocateErrors":  allocateErrors,
		"quotaMetrics": []map[string]any{
			{
				"metricName": "serviceruntime.googleapis.com/api/consumer/quota_used_count",
				"metricValues": []map[string]any{
					{
						"int64Value": "1",
						"startTime":  gcpServiceControlReferenceTime.Format(time.RFC3339),
						"endTime":    gcpServiceControlReferenceTime.Add(time.Minute).Format(time.RFC3339),
					},
				},
			},
		},
	}
}

func respondGCPServiceControlInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_servicecontrol(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPServiceControlPath(rawRequestPath(r))
	serviceName, action, ok := parseGCPServiceControlActionPath(path)
	if !ok || (action != "check" && action != "report" && action != "allocateQuota") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPServiceControlInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	var payload map[string]any
	switch action {
	case "check":
		payload = gcpServiceControlCheckResponse("stackyard-check-op", false)
	case "report":
		payload = gcpServiceControlReportResponse([]map[string]any{})
	case "allocateQuota":
		payload = gcpServiceControlAllocateQuotaResponse("stackyard-quota-op", false)
	default:
		return false
	}
	payload["service"] = "servicecontrol"
	payload["provider"] = providerGCP
	payload["path"] = path
	payload["name"] = fmt.Sprintf("services/%s:%s", serviceName, action)
	respondJSON(w, http.StatusOK, payload)
	return true
}
