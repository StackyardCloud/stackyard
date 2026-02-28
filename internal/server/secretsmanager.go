package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	secretsmgrsvc "github.com/stackyard/stackyard/internal/services/secretsmanager"
)

type secretsManagerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSecretsManagerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSecretsManagerJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "secretsmanager")
	if !ok {
		respondSecretsManagerError(w, status, code, msg)
		return true
	}

	action := parseSecretsManagerTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := secretsManagerOperationByName[action]; !known {
		respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSecretsManagerPayload(r)
	if err != nil {
		respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	if err := s.secretsmgr.RecordAPICall(); err != nil {
		respondSecretsManagerErrorForErr(w, err)
		return true
	}

	switch action {
	case "CreateSecret":
		out, err := s.secretsmgr.CreateSecret(secretsmgrsvc.CreateSecretInput{
			Name:               secretsManagerString(payload["Name"]),
			ClientRequestToken: secretsManagerString(payload["ClientRequestToken"]),
			Description:        secretsManagerString(payload["Description"]),
			KmsKeyID:           secretsManagerFirstString(payload, "KmsKeyId", "KMSKeyID"),
			OwningService:      secretsManagerString(payload["OwningService"]),
			SecretBinary:       secretsManagerString(payload["SecretBinary"]),
			SecretString:       secretsManagerString(payload["SecretString"]),
			Tags:               secretsManagerTagsPayload(payload["Tags"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		}
		if strings.TrimSpace(out.VersionID) != "" {
			response["VersionId"] = out.VersionID
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "DescribeSecret":
		out, err := s.secretsmgr.DescribeSecret(secretsManagerString(payload["SecretId"]))
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, secretsManagerDescribeSecretPayload(out))
		return true

	case "ListSecrets":
		maxResults, ok := secretsManagerInt32(payload["MaxResults"])
		if !ok {
			respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		out, err := s.secretsmgr.ListSecrets(secretsmgrsvc.ListSecretsInput{
			NextToken:              secretsManagerString(payload["NextToken"]),
			MaxResults:             maxResults,
			IncludePlannedDeletion: secretsManagerBool(payload["IncludePlannedDeletion"]),
			SortOrder:              strings.ToLower(secretsManagerString(payload["SortOrder"])),
			Filters:                secretsManagerFiltersPayload(payload["Filters"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"SecretList": secretsManagerSecretSummaryListPayload(out.SecretList),
		}
		if strings.TrimSpace(out.NextToken) != "" {
			response["NextToken"] = out.NextToken
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "UpdateSecret":
		out, err := s.secretsmgr.UpdateSecret(secretsmgrsvc.UpdateSecretInput{
			SecretID:           secretsManagerString(payload["SecretId"]),
			ClientRequestToken: secretsManagerString(payload["ClientRequestToken"]),
			Description:        secretsManagerString(payload["Description"]),
			KmsKeyID:           secretsManagerFirstString(payload, "KmsKeyId", "KMSKeyID"),
			SecretBinary:       secretsManagerString(payload["SecretBinary"]),
			SecretString:       secretsManagerString(payload["SecretString"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		}
		if strings.TrimSpace(out.VersionID) != "" {
			response["VersionId"] = out.VersionID
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "DeleteSecret":
		recoveryWindowInDays, ok := secretsManagerInt64(payload["RecoveryWindowInDays"])
		if !ok {
			respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid RecoveryWindowInDays")
			return true
		}
		out, err := s.secretsmgr.DeleteSecret(secretsmgrsvc.DeleteSecretInput{
			SecretID:                   secretsManagerString(payload["SecretId"]),
			RecoveryWindowInDays:       recoveryWindowInDays,
			ForceDeleteWithoutRecovery: secretsManagerBool(payload["ForceDeleteWithoutRecovery"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":          out.ARN,
			"Name":         out.Name,
			"DeletionDate": out.DeletionDate,
		})
		return true

	case "RestoreSecret":
		out, err := s.secretsmgr.RestoreSecret(secretsManagerString(payload["SecretId"]))
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		})
		return true

	case "PutSecretValue":
		out, err := s.secretsmgr.PutSecretValue(secretsmgrsvc.PutSecretValueInput{
			SecretID:           secretsManagerString(payload["SecretId"]),
			ClientRequestToken: secretsManagerString(payload["ClientRequestToken"]),
			SecretBinary:       secretsManagerString(payload["SecretBinary"]),
			SecretString:       secretsManagerString(payload["SecretString"]),
			VersionStages:      secretsManagerStringSlice(payload["VersionStages"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":           out.ARN,
			"Name":          out.Name,
			"VersionId":     out.VersionID,
			"VersionStages": append([]string(nil), out.VersionStages...),
		})
		return true

	case "GetSecretValue":
		out, err := s.secretsmgr.GetSecretValue(secretsmgrsvc.GetSecretValueInput{
			SecretID:     secretsManagerString(payload["SecretId"]),
			VersionID:    secretsManagerString(payload["VersionId"]),
			VersionStage: secretsManagerString(payload["VersionStage"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, secretsManagerGetSecretValuePayload(out))
		return true

	case "ListSecretVersionIds":
		maxResults, ok := secretsManagerInt32(payload["MaxResults"])
		if !ok {
			respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		out, err := s.secretsmgr.ListSecretVersionIDs(secretsmgrsvc.ListSecretVersionIDsInput{
			SecretID:          secretsManagerString(payload["SecretId"]),
			NextToken:         secretsManagerString(payload["NextToken"]),
			MaxResults:        maxResults,
			IncludeDeprecated: secretsManagerBool(payload["IncludeDeprecated"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"ARN":      out.ARN,
			"Name":     out.Name,
			"Versions": secretsManagerSecretVersionListPayload(out.Versions),
		}
		if strings.TrimSpace(out.NextToken) != "" {
			response["NextToken"] = out.NextToken
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "UpdateSecretVersionStage":
		out, err := s.secretsmgr.UpdateSecretVersionStage(secretsmgrsvc.UpdateSecretVersionStageInput{
			SecretID:            secretsManagerString(payload["SecretId"]),
			VersionStage:        secretsManagerString(payload["VersionStage"]),
			RemoveFromVersionID: secretsManagerString(payload["RemoveFromVersionId"]),
			MoveToVersionID:     secretsManagerString(payload["MoveToVersionId"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		})
		return true

	case "BatchGetSecretValue":
		maxResults, ok := secretsManagerInt32(payload["MaxResults"])
		if !ok {
			respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		out, err := s.secretsmgr.BatchGetSecretValue(secretsmgrsvc.BatchGetSecretValueInput{
			SecretIDList: secretsManagerStringSlice(payload["SecretIdList"]),
			NextToken:    secretsManagerString(payload["NextToken"]),
			MaxResults:   maxResults,
			Filters:      secretsManagerFiltersPayload(payload["Filters"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"SecretValues": secretsManagerGetSecretValuesPayload(out.SecretValues),
			"Errors":       secretsManagerBatchErrorsPayload(out.Errors),
		}
		if strings.TrimSpace(out.NextToken) != "" {
			response["NextToken"] = out.NextToken
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "GetRandomPassword":
		length, ok := secretsManagerInt64(payload["PasswordLength"])
		if !ok {
			respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid PasswordLength")
			return true
		}
		password, err := s.secretsmgr.GetRandomPassword(secretsmgrsvc.GetRandomPasswordInput{
			PasswordLength:          length,
			ExcludeCharacters:       secretsManagerString(payload["ExcludeCharacters"]),
			ExcludeNumbers:          secretsManagerBool(payload["ExcludeNumbers"]),
			ExcludePunctuation:      secretsManagerBool(payload["ExcludePunctuation"]),
			ExcludeUppercase:        secretsManagerBool(payload["ExcludeUppercase"]),
			ExcludeLowercase:        secretsManagerBool(payload["ExcludeLowercase"]),
			IncludeSpace:            secretsManagerBool(payload["IncludeSpace"]),
			RequireEachIncludedType: secretsManagerBool(payload["RequireEachIncludedType"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"RandomPassword": password,
		})
		return true

	case "RotateSecret":
		var automaticallyAfterDays int64
		if rotationRules, ok := payload["RotationRules"].(map[string]any); ok {
			parsed, valid := secretsManagerInt64(rotationRules["AutomaticallyAfterDays"])
			if !valid {
				respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", "invalid RotationRules.AutomaticallyAfterDays")
				return true
			}
			automaticallyAfterDays = parsed
		}
		rotateImmediately := true
		if raw, exists := payload["RotateImmediately"]; exists {
			rotateImmediately = secretsManagerBool(raw)
		}
		out, err := s.secretsmgr.RotateSecret(secretsmgrsvc.RotateSecretInput{
			SecretID:               secretsManagerString(payload["SecretId"]),
			ClientRequestToken:     secretsManagerString(payload["ClientRequestToken"]),
			RotationLambdaARN:      secretsManagerFirstString(payload, "RotationLambdaARN", "RotationLambdaArn"),
			AutomaticallyAfterDays: automaticallyAfterDays,
			RotateImmediately:      rotateImmediately,
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		}
		if strings.TrimSpace(out.VersionID) != "" {
			response["VersionId"] = out.VersionID
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "CancelRotateSecret":
		out, err := s.secretsmgr.CancelRotateSecret(secretsManagerString(payload["SecretId"]))
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		}
		if strings.TrimSpace(out.VersionID) != "" {
			response["VersionId"] = out.VersionID
		}
		respondSecretsManagerJSON(w, http.StatusOK, response)
		return true

	case "ReplicateSecretToRegions":
		out, err := s.secretsmgr.ReplicateSecretToRegions(secretsmgrsvc.ReplicateSecretToRegionsInput{
			SecretID:                    secretsManagerString(payload["SecretId"]),
			AddReplicaRegions:           secretsManagerReplicaRegionsPayload(payload["AddReplicaRegions"]),
			ForceOverwriteReplicaSecret: secretsManagerBool(payload["ForceOverwriteReplicaSecret"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":               out.ARN,
			"ReplicationStatus": secretsManagerReplicationStatusPayload(out.ReplicationStatus),
		})
		return true

	case "RemoveRegionsFromReplication":
		out, err := s.secretsmgr.RemoveRegionsFromReplication(secretsmgrsvc.RemoveRegionsFromReplicationInput{
			SecretID:             secretsManagerString(payload["SecretId"]),
			RemoveReplicaRegions: secretsManagerStringSlice(payload["RemoveReplicaRegions"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":               out.ARN,
			"ReplicationStatus": secretsManagerReplicationStatusPayload(out.ReplicationStatus),
		})
		return true

	case "StopReplicationToReplica":
		out, err := s.secretsmgr.StopReplicationToReplica(secretsmgrsvc.StopReplicationToReplicaInput{
			SecretID:      secretsManagerString(payload["SecretId"]),
			ReplicaRegion: secretsManagerFirstString(payload, "ReplicaRegion", "Region"),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN": out.ARN,
		})
		return true

	case "PutResourcePolicy":
		out, err := s.secretsmgr.PutResourcePolicy(secretsmgrsvc.PutResourcePolicyInput{
			SecretID:          secretsManagerString(payload["SecretId"]),
			ResourcePolicy:    secretsManagerString(payload["ResourcePolicy"]),
			BlockPublicPolicy: secretsManagerBool(payload["BlockPublicPolicy"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		})
		return true

	case "GetResourcePolicy":
		out, err := s.secretsmgr.GetResourcePolicy(secretsManagerString(payload["SecretId"]))
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":            out.ARN,
			"Name":           out.Name,
			"ResourcePolicy": out.ResourcePolicy,
		})
		return true

	case "DeleteResourcePolicy":
		out, err := s.secretsmgr.DeleteResourcePolicy(secretsManagerString(payload["SecretId"]))
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"ARN":  out.ARN,
			"Name": out.Name,
		})
		return true

	case "ValidateResourcePolicy":
		out, err := s.secretsmgr.ValidateResourcePolicy(secretsmgrsvc.ValidateResourcePolicyInput{
			SecretID:          secretsManagerString(payload["SecretId"]),
			ResourcePolicy:    secretsManagerString(payload["ResourcePolicy"]),
			BlockPublicPolicy: secretsManagerBool(payload["BlockPublicPolicy"]),
		})
		if err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{
			"PolicyValidationPassed": out.PolicyValidationPassed,
			"ValidationErrors":       secretsManagerPolicyValidationErrorsPayload(out.ValidationErrors),
		})
		return true

	case "TagResource":
		if err := s.secretsmgr.TagResource(secretsmgrsvc.TagResourceInput{
			SecretID: secretsManagerString(payload["SecretId"]),
			Tags:     secretsManagerTagsPayload(payload["Tags"]),
		}); err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		if err := s.secretsmgr.UntagResource(secretsmgrsvc.UntagResourceInput{
			SecretID: secretsManagerString(payload["SecretId"]),
			TagKeys:  secretsManagerStringSlice(payload["TagKeys"]),
		}); err != nil {
			respondSecretsManagerErrorForErr(w, err)
			return true
		}
		respondSecretsManagerJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	respondSecretsManagerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isSecretsManagerJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	targetLower := strings.ToLower(target)
	if strings.HasPrefix(targetLower, "secretsmanager.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(targetLower, "secretsmanager")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "secretsmanager" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".secretsmanager.") || strings.HasPrefix(host, "secretsmanager.")
}

func parseSecretsManagerTarget(target string) string {
	if target == "" {
		return ""
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSecretsManagerPayload(r *http.Request) (map[string]any, error) {
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
		return map[string]any{}, nil
	}
	return payload, nil
}

func respondSecretsManagerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSecretsManagerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSecretsManagerJSON(w, status, secretsManagerError{Type: code, Message: msg})
}

func respondSecretsManagerErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, secretsmgrsvc.ErrInvalidParameter):
		respondSecretsManagerError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, secretsmgrsvc.ErrNotFound):
		respondSecretsManagerError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
	case errors.Is(err, secretsmgrsvc.ErrInvalidState):
		respondSecretsManagerError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
	case errors.Is(err, secretsmgrsvc.ErrLimitExceeded):
		respondSecretsManagerError(w, http.StatusBadRequest, "LimitExceededException", err.Error())
	case errors.Is(err, secretsmgrsvc.ErrThrottling):
		respondSecretsManagerError(w, http.StatusTooManyRequests, "ThrottlingException", err.Error())
	default:
		respondSecretsManagerError(w, http.StatusInternalServerError, "InternalServiceError", err.Error())
	}
}

func secretsManagerString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func secretsManagerFirstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := secretsManagerString(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func secretsManagerBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err == nil {
			return parsed
		}
	}
	return false
}

func secretsManagerInt32(value any) (int32, bool) {
	if value == nil {
		return 0, true
	}
	switch v := value.(type) {
	case json.Number:
		i64, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(i64), true
	case float64:
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case string:
		i64, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return 0, false
		}
		return int32(i64), true
	default:
		return 0, false
	}
}

func secretsManagerInt64(value any) (int64, bool) {
	if value == nil {
		return 0, true
	}
	switch v := value.(type) {
	case json.Number:
		i64, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return i64, true
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case string:
		i64, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return i64, true
	default:
		return 0, false
	}
}

func secretsManagerStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := secretsManagerString(item)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}

func secretsManagerTagsPayload(value any) map[string]string {
	items, ok := value.([]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		tag, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := secretsManagerFirstString(tag, "Key", "key")
		if key == "" {
			continue
		}
		out[key] = secretsManagerFirstString(tag, "Value", "value")
	}
	return out
}

func secretsManagerFiltersPayload(value any) []secretsmgrsvc.SecretFilter {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]secretsmgrsvc.SecretFilter, 0, len(items))
	for _, item := range items {
		filter, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := secretsManagerFirstString(filter, "Key", "key")
		values := secretsManagerStringSlice(filter["Values"])
		out = append(out, secretsmgrsvc.SecretFilter{
			Key:    key,
			Values: values,
		})
	}
	return out
}

func secretsManagerReplicaRegionsPayload(value any) []secretsmgrsvc.ReplicaRegionInput {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]secretsmgrsvc.ReplicaRegionInput, 0, len(items))
	for _, item := range items {
		region, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, secretsmgrsvc.ReplicaRegionInput{
			Region:   secretsManagerFirstString(region, "Region", "region"),
			KmsKeyID: secretsManagerFirstString(region, "KmsKeyId", "KMSKeyID"),
		})
	}
	return out
}

func secretsManagerTagsListPayload(tags map[string]string) []map[string]string {
	if len(tags) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{
			"Key":   key,
			"Value": tags[key],
		})
	}
	return out
}

func secretsManagerReplicationStatusPayload(value any) []map[string]any {
	statuses := make([]secretsmgrsvc.ReplicationStatus, 0)
	switch typed := value.(type) {
	case []secretsmgrsvc.ReplicationStatus:
		statuses = append(statuses, typed...)
	case map[string]*secretsmgrsvc.ReplicationStatus:
		regions := make([]string, 0, len(typed))
		for region := range typed {
			regions = append(regions, region)
		}
		sort.Strings(regions)
		for _, region := range regions {
			status := typed[region]
			if status == nil {
				continue
			}
			statuses = append(statuses, *status)
		}
	default:
		return []map[string]any{}
	}
	if len(statuses) == 0 {
		return []map[string]any{}
	}

	out := make([]map[string]any, 0, len(statuses))
	for _, status := range statuses {
		item := map[string]any{
			"Region":        status.Region,
			"KmsKeyId":      status.KmsKeyID,
			"Status":        status.Status,
			"StatusMessage": status.StatusMessage,
		}
		if status.LastAccessedDate != nil {
			item["LastAccessedDate"] = *status.LastAccessedDate
		}
		out = append(out, item)
	}
	return out
}

func secretsManagerPolicyValidationErrorsPayload(values []secretsmgrsvc.ValidateResourcePolicyError) []map[string]string {
	if len(values) == 0 {
		return []map[string]string{}
	}
	out := make([]map[string]string, 0, len(values))
	for _, value := range values {
		out = append(out, map[string]string{
			"CheckName":    value.CheckName,
			"ErrorMessage": value.ErrorMessage,
		})
	}
	return out
}

func secretsManagerDescribeSecretPayload(secret secretsmgrsvc.Secret) map[string]any {
	response := map[string]any{
		"ARN":               secret.ARN,
		"Name":              secret.Name,
		"Description":       secret.Description,
		"KmsKeyId":          secret.KmsKeyID,
		"OwningService":     secret.OwningService,
		"PrimaryRegion":     secret.PrimaryRegion,
		"CreatedDate":       secret.CreatedDate,
		"LastChangedDate":   secret.LastChangedDate,
		"RotationEnabled":   secret.RotationEnabled,
		"RotationLambdaARN": secret.RotationLambdaARN,
		"Tags":              secretsManagerTagsListPayload(secret.Tags),
		"ReplicationStatus": secretsManagerReplicationStatusPayload(secret.ReplicationStatus),
	}
	if secret.LastAccessedDate != nil {
		response["LastAccessedDate"] = *secret.LastAccessedDate
	}
	if secret.AutomaticallyAfterDays > 0 {
		response["RotationRules"] = map[string]any{
			"AutomaticallyAfterDays": secret.AutomaticallyAfterDays,
		}
	}
	if secret.LastRotatedDate != nil {
		response["LastRotatedDate"] = *secret.LastRotatedDate
	}
	if secret.NextRotationDate != nil {
		response["NextRotationDate"] = *secret.NextRotationDate
	}
	if secret.DeletedDate != nil {
		response["DeletedDate"] = *secret.DeletedDate
	}

	versionIDsToStages := map[string][]string{}
	for versionID, version := range secret.Versions {
		stages := append([]string(nil), version.Stages...)
		sort.Strings(stages)
		versionIDsToStages[versionID] = stages
	}
	response["VersionIdsToStages"] = versionIDsToStages
	return response
}

func secretsManagerSecretSummaryListPayload(summaries []secretsmgrsvc.SecretSummary) []map[string]any {
	if len(summaries) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(summaries))
	for _, summary := range summaries {
		item := map[string]any{
			"ARN":               summary.ARN,
			"Name":              summary.Name,
			"Description":       summary.Description,
			"KmsKeyId":          summary.KmsKeyID,
			"OwningService":     summary.OwningService,
			"PrimaryRegion":     summary.PrimaryRegion,
			"CreatedDate":       summary.CreatedDate,
			"LastChangedDate":   summary.LastChangedDate,
			"RotationEnabled":   summary.RotationEnabled,
			"Tags":              secretsManagerTagsListPayload(summary.Tags),
			"ReplicationStatus": secretsManagerReplicationStatusPayload(summary.ReplicationStatus),
		}
		if summary.LastAccessedDate != nil {
			item["LastAccessedDate"] = *summary.LastAccessedDate
		}
		if summary.LastRotatedDate != nil {
			item["LastRotatedDate"] = *summary.LastRotatedDate
		}
		if summary.NextRotationDate != nil {
			item["NextRotationDate"] = *summary.NextRotationDate
		}
		if summary.DeletedDate != nil {
			item["DeletedDate"] = *summary.DeletedDate
		}
		out = append(out, item)
	}
	return out
}

func secretsManagerGetSecretValuePayload(value secretsmgrsvc.GetSecretValueOutput) map[string]any {
	response := map[string]any{
		"ARN":           value.ARN,
		"Name":          value.Name,
		"VersionId":     value.VersionID,
		"VersionStages": append([]string(nil), value.VersionStages...),
		"CreatedDate":   value.CreatedDate,
	}
	if strings.TrimSpace(value.SecretString) != "" {
		response["SecretString"] = value.SecretString
	}
	if strings.TrimSpace(value.SecretBinary) != "" {
		response["SecretBinary"] = value.SecretBinary
	}
	return response
}

func secretsManagerGetSecretValuesPayload(values []secretsmgrsvc.GetSecretValueOutput) []map[string]any {
	if len(values) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		out = append(out, secretsManagerGetSecretValuePayload(value))
	}
	return out
}

func secretsManagerSecretVersionListPayload(versions []secretsmgrsvc.SecretVersionListItem) []map[string]any {
	if len(versions) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(versions))
	for _, version := range versions {
		out = append(out, map[string]any{
			"VersionId":     version.VersionID,
			"VersionStages": append([]string(nil), version.VersionStages...),
			"CreatedDate":   version.CreatedDate,
		})
	}
	return out
}

func secretsManagerBatchErrorsPayload(errorsList []secretsmgrsvc.BatchGetSecretValueError) []map[string]any {
	if len(errorsList) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(errorsList))
	for _, entry := range errorsList {
		out = append(out, map[string]any{
			"SecretId":  entry.SecretID,
			"ErrorCode": entry.ErrorCode,
			"Message":   entry.Message,
		})
	}
	return out
}
