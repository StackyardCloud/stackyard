package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type cognitoUserPoolsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCognitoUserPoolsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCognitoUserPoolsJSONCandidate(r) {
		return false
	}

	action := parseCognitoUserPoolsTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := cognitoUserPoolsOperationByName[action]; !known {
		respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cognito-idp")
	if !ok {
		respondCognitoUserPoolsError(w, status, code, msg)
		return true
	}

	payload, err := parseCognitoUserPoolsPayload(r)
	if err != nil {
		respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	// Stage 1: user pool and domain lifecycle.
	case "CreateUserPool":
		tags, ok := cognitoUserPoolsStringMap(payload["UserPoolTags"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserPoolTags")
			return true
		}
		record, err := s.cognitouserpools.CreateUserPool(cognitoUserPoolsCreateUserPoolInput{
			Region:           cognitoUserPoolsRegionFromRequest(r),
			PoolName:         cognitoUserPoolsString(payload["PoolName"]),
			MFAConfiguration: cognitoUserPoolsString(payload["MfaConfiguration"]),
			Tags:             tags,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserPool": cognitoUserPoolsUserPoolPayload(record)})
		return true

	case "DescribeUserPool":
		record, err := s.cognitouserpools.DescribeUserPool(cognitoUserPoolsString(payload["UserPoolId"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserPool": cognitoUserPoolsUserPoolPayload(record)})
		return true

	case "ListUserPools":
		maxResults, ok := cognitoUserPoolsInt(payload["MaxResults"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListUserPools(maxResults, cognitoUserPoolsString(payload["NextToken"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"UserPools": cognitoUserPoolsUserPoolDescriptionsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateUserPool":
		input := cognitoUserPoolsUpdateUserPoolInput{
			UserPoolID: cognitoUserPoolsString(payload["UserPoolId"]),
		}
		if raw, exists := cognitoUserPoolsField(payload, "MfaConfiguration"); exists {
			input.MFAConfigurationSet = true
			input.MFAConfiguration = cognitoUserPoolsString(raw)
		}
		if raw, exists := cognitoUserPoolsField(payload, "UserPoolTags"); exists {
			tags, ok := cognitoUserPoolsStringMap(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserPoolTags")
				return true
			}
			input.TagsSet = true
			input.Tags = tags
		}
		if err := s.cognitouserpools.UpdateUserPool(input); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteUserPool":
		if err := s.cognitouserpools.DeleteUserPool(cognitoUserPoolsString(payload["UserPoolId"])); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateUserPoolDomain":
		record, err := s.cognitouserpools.CreateUserPoolDomain(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Domain"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CloudFrontDomain":    record.CloudFrontDomain,
			"ManagedLoginVersion": record.Version,
		})
		return true

	case "DescribeUserPoolDomain":
		record, err := s.cognitouserpools.DescribeUserPoolDomain(cognitoUserPoolsString(payload["Domain"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"DomainDescription": cognitoUserPoolsDomainDescriptionPayload(record)})
		return true

	case "DeleteUserPoolDomain":
		if err := s.cognitouserpools.DeleteUserPoolDomain(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Domain"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	// Stage 2: app clients, resource servers, and tagging.
	case "CreateUserPoolClient":
		explicitAuthFlows, ok := cognitoUserPoolsStringSlice(payload["ExplicitAuthFlows"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ExplicitAuthFlows")
			return true
		}
		generateSecret, ok := cognitoUserPoolsBoolDefaultFalse(payload, "GenerateSecret")
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid GenerateSecret")
			return true
		}
		refreshTokenValidity, ok := cognitoUserPoolsIntDefault(payload, "RefreshTokenValidity", 30)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid RefreshTokenValidity")
			return true
		}
		record, err := s.cognitouserpools.CreateUserPoolClient(cognitoUserPoolsCreateClientInput{
			UserPoolID:           cognitoUserPoolsString(payload["UserPoolId"]),
			ClientName:           cognitoUserPoolsString(payload["ClientName"]),
			GenerateSecret:       generateSecret,
			ExplicitAuthFlows:    explicitAuthFlows,
			RefreshTokenValidity: refreshTokenValidity,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserPoolClient": cognitoUserPoolsUserPoolClientPayload(record)})
		return true

	case "DescribeUserPoolClient":
		record, err := s.cognitouserpools.DescribeUserPoolClient(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserPoolClient": cognitoUserPoolsUserPoolClientPayload(record)})
		return true

	case "ListUserPoolClients":
		maxResults, ok := cognitoUserPoolsInt(payload["MaxResults"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListUserPoolClients(
			cognitoUserPoolsString(payload["UserPoolId"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"UserPoolClients": cognitoUserPoolsUserPoolClientDescriptionsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateUserPoolClient":
		input := cognitoUserPoolsUpdateClientInput{
			UserPoolID: cognitoUserPoolsString(payload["UserPoolId"]),
			ClientID:   cognitoUserPoolsString(payload["ClientId"]),
		}
		if raw, exists := cognitoUserPoolsField(payload, "ClientName"); exists {
			input.ClientNameSet = true
			input.ClientName = cognitoUserPoolsString(raw)
		}
		if raw, exists := cognitoUserPoolsField(payload, "ExplicitAuthFlows"); exists {
			flows, ok := cognitoUserPoolsStringSlice(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ExplicitAuthFlows")
				return true
			}
			input.ExplicitAuthFlowsSet = true
			input.ExplicitAuthFlows = flows
		}
		if raw, exists := cognitoUserPoolsField(payload, "RefreshTokenValidity"); exists {
			value, ok := cognitoUserPoolsInt(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid RefreshTokenValidity")
				return true
			}
			input.RefreshTokenValiditySet = true
			input.RefreshTokenValidity = value
		}
		record, err := s.cognitouserpools.UpdateUserPoolClient(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserPoolClient": cognitoUserPoolsUserPoolClientPayload(record)})
		return true

	case "DeleteUserPoolClient":
		if err := s.cognitouserpools.DeleteUserPoolClient(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "TagResource":
		userPoolID, err := cognitoUserPoolsIDFromResourceARN(cognitoUserPoolsString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tags, ok := cognitoUserPoolsStringMap(payload["Tags"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Tags")
			return true
		}
		if err := s.cognitouserpools.TagUserPool(userPoolID, tags); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		userPoolID, err := cognitoUserPoolsIDFromResourceARN(cognitoUserPoolsString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tagKeys, ok := cognitoUserPoolsStringSlice(payload["TagKeys"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid TagKeys")
			return true
		}
		if err := s.cognitouserpools.UntagUserPool(userPoolID, tagKeys); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListTagsForResource":
		userPoolID, err := cognitoUserPoolsIDFromResourceARN(cognitoUserPoolsString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tags, err := s.cognitouserpools.ListUserPoolTags(userPoolID)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Tags": tags})
		return true

	case "CreateResourceServer":
		scopes, ok := cognitoUserPoolsResourceServerScopes(payload["Scopes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Scopes")
			return true
		}
		record, err := s.cognitouserpools.CreateResourceServer(cognitoUserPoolsCreateResourceServerInput{
			UserPoolID: cognitoUserPoolsString(payload["UserPoolId"]),
			Identifier: cognitoUserPoolsString(payload["Identifier"]),
			Name:       cognitoUserPoolsString(payload["Name"]),
			Scopes:     scopes,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ResourceServer": cognitoUserPoolsResourceServerPayload(record)})
		return true

	case "DescribeResourceServer":
		record, err := s.cognitouserpools.DescribeResourceServer(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Identifier"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ResourceServer": cognitoUserPoolsResourceServerPayload(record)})
		return true

	case "ListResourceServers":
		maxResults, ok := cognitoUserPoolsInt(payload["MaxResults"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListResourceServers(
			cognitoUserPoolsString(payload["UserPoolId"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"ResourceServers": cognitoUserPoolsResourceServersPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateResourceServer":
		scopes, ok := cognitoUserPoolsResourceServerScopes(payload["Scopes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Scopes")
			return true
		}
		record, err := s.cognitouserpools.UpdateResourceServer(cognitoUserPoolsUpdateResourceServerInput{
			UserPoolID: cognitoUserPoolsString(payload["UserPoolId"]),
			Identifier: cognitoUserPoolsString(payload["Identifier"]),
			Name:       cognitoUserPoolsString(payload["Name"]),
			Scopes:     scopes,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ResourceServer": cognitoUserPoolsResourceServerPayload(record)})
		return true

	case "DeleteResourceServer":
		if err := s.cognitouserpools.DeleteResourceServer(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Identifier"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	if s.handleCognitoUserPoolsStage3Action(w, action, payload) {
		return true
	}
	if s.handleCognitoUserPoolsStage4Action(w, action, payload) {
		return true
	}
	if s.handleCognitoUserPoolsStage5Action(w, action, payload) {
		return true
	}
	if s.handleCognitoUserPoolsStage6Action(w, action, payload) {
		return true
	}
	if s.handleCognitoUserPoolsStage7Action(w, action, payload) {
		return true
	}

	respondCognitoUserPoolsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isCognitoUserPoolsJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSCognitoIdentityProviderService") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "IdentityProviderService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "cognito-idp" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".cognito-idp.") || strings.HasPrefix(host, "cognito-idp.")
}

func parseCognitoUserPoolsTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSCognitoIdentityProviderService.") {
		return strings.TrimPrefix(target, "AWSCognitoIdentityProviderService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCognitoUserPoolsPayload(r *http.Request) (map[string]any, error) {
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

func respondCognitoUserPoolsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCognitoUserPoolsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCognitoUserPoolsJSON(w, status, cognitoUserPoolsError{Type: code, Message: msg})
}

func respondCognitoUserPoolsErrorForErr(w http.ResponseWriter, err error) {
	if apiErr := asCognitoUserPoolsAPIError(err); apiErr != nil {
		respondCognitoUserPoolsError(w, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	respondCognitoUserPoolsError(w, http.StatusInternalServerError, "InternalErrorException", err.Error())
}

func cognitoUserPoolsField(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	value, ok := payload[key]
	return value, ok
}

func cognitoUserPoolsString(value any) string {
	if value == nil {
		return ""
	}
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
}

func cognitoUserPoolsBoolDefaultFalse(payload map[string]any, key string) (bool, bool) {
	raw, exists := cognitoUserPoolsField(payload, key)
	if !exists {
		return false, true
	}
	value, ok := raw.(bool)
	if !ok {
		return false, false
	}
	return value, true
}

func cognitoUserPoolsIntDefault(payload map[string]any, key string, fallback int) (int, bool) {
	raw, exists := cognitoUserPoolsField(payload, key)
	if !exists {
		return fallback, true
	}
	value, ok := cognitoUserPoolsInt(raw)
	if !ok {
		return 0, false
	}
	return value, true
}

func cognitoUserPoolsInt(value any) (int, bool) {
	if value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		if typed <= 0 {
			return 0, false
		}
		return typed, true
	case int32:
		if typed <= 0 {
			return 0, false
		}
		return int(typed), true
	case int64:
		if typed <= 0 {
			return 0, false
		}
		return int(typed), true
	case float64:
		if typed <= 0 || typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func cognitoUserPoolsStringMap(value any) (map[string]string, bool) {
	if value == nil {
		return nil, true
	}
	switch typed := value.(type) {
	case map[string]string:
		return cloneStringMap(typed), true
	case map[string]any:
		if len(typed) == 0 {
			return map[string]string{}, true
		}
		out := make(map[string]string, len(typed))
		for key, raw := range typed {
			asString, ok := raw.(string)
			if !ok {
				return nil, false
			}
			out[strings.TrimSpace(key)] = strings.TrimSpace(asString)
		}
		return out, true
	default:
		return nil, false
	}
}

func cognitoUserPoolsStringSlice(value any) ([]string, bool) {
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
			trimmed := strings.TrimSpace(asString)
			if trimmed == "" {
				continue
			}
			out = append(out, trimmed)
		}
		return out, true
	default:
		return nil, false
	}
}

func cognitoUserPoolsResourceServerScopes(value any) ([]cognitoUserPoolsResourceServerScope, bool) {
	if value == nil {
		return nil, true
	}
	rawScopes, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]cognitoUserPoolsResourceServerScope, 0, len(rawScopes))
	for _, item := range rawScopes {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		scopeName := cognitoUserPoolsString(obj["ScopeName"])
		if scopeName == "" {
			return nil, false
		}
		out = append(out, cognitoUserPoolsResourceServerScope{
			ScopeName:        scopeName,
			ScopeDescription: cognitoUserPoolsString(obj["ScopeDescription"]),
		})
	}
	return out, true
}

func cognitoUserPoolsUserPoolPayload(record cognitoUserPoolsUserPoolRecord) map[string]any {
	out := map[string]any{
		"Id":               record.ID,
		"Arn":              record.ARN,
		"Name":             record.Name,
		"Status":           record.Status,
		"CreationDate":     float64(record.CreatedAt.Unix()),
		"LastModifiedDate": float64(record.UpdatedAt.Unix()),
	}
	if record.MFAConfiguration != "" {
		out["MfaConfiguration"] = record.MFAConfiguration
	}
	if len(record.Tags) > 0 {
		out["UserPoolTags"] = cloneStringMap(record.Tags)
	}
	return out
}

func cognitoUserPoolsUserPoolDescriptionsPayload(records []cognitoUserPoolsUserPoolRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{
			"Id":               record.ID,
			"Name":             record.Name,
			"Status":           record.Status,
			"CreationDate":     float64(record.CreatedAt.Unix()),
			"LastModifiedDate": float64(record.UpdatedAt.Unix()),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["Id"].(string)
		right, _ := out[j]["Id"].(string)
		return left < right
	})
	return out
}

func cognitoUserPoolsDomainDescriptionPayload(record cognitoUserPoolsDomainRecord) map[string]any {
	return map[string]any{
		"Domain":           record.Domain,
		"UserPoolId":       record.UserPoolID,
		"CloudFrontDomain": record.CloudFrontDomain,
		"Version":          strconv.Itoa(record.Version),
		"Status":           "ACTIVE",
	}
}

func cognitoUserPoolsUserPoolClientPayload(record cognitoUserPoolsClientRecord) map[string]any {
	out := map[string]any{
		"UserPoolId":           record.UserPoolID,
		"ClientId":             record.ClientID,
		"ClientName":           record.ClientName,
		"RefreshTokenValidity": record.RefreshTokenValidity,
		"CreationDate":         float64(record.CreatedAt.Unix()),
		"LastModifiedDate":     float64(record.UpdatedAt.Unix()),
	}
	if len(record.ExplicitAuthFlows) > 0 {
		out["ExplicitAuthFlows"] = cloneStringSlice(record.ExplicitAuthFlows)
	}
	if record.ClientSecret != "" {
		out["ClientSecret"] = record.ClientSecret
	}
	return out
}

func cognitoUserPoolsUserPoolClientDescriptionsPayload(records []cognitoUserPoolsClientRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{
			"UserPoolId": record.UserPoolID,
			"ClientId":   record.ClientID,
			"ClientName": record.ClientName,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["ClientId"].(string)
		right, _ := out[j]["ClientId"].(string)
		return left < right
	})
	return out
}

func cognitoUserPoolsResourceServerPayload(record cognitoUserPoolsResourceServerRecord) map[string]any {
	scopes := make([]map[string]any, 0, len(record.Scopes))
	for _, scope := range record.Scopes {
		scopes = append(scopes, map[string]any{
			"ScopeName":        scope.ScopeName,
			"ScopeDescription": scope.ScopeDescription,
		})
	}
	return map[string]any{
		"UserPoolId": record.UserPoolID,
		"Identifier": record.Identifier,
		"Name":       record.Name,
		"Scopes":     scopes,
	}
}

func cognitoUserPoolsResourceServersPayload(records []cognitoUserPoolsResourceServerRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsResourceServerPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["Identifier"].(string)
		right, _ := out[j]["Identifier"].(string)
		return left < right
	})
	return out
}

func cognitoUserPoolsRegionFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		if parsed, err := parseSigV4Authorization(auth); err == nil {
			region := strings.TrimSpace(parsed.Region)
			if region != "" {
				return region
			}
		}
	}

	if parsed, err := parseSigV4Query(r.URL.Query()); err == nil {
		region := strings.TrimSpace(parsed.Region)
		if region != "" {
			return region
		}
	}
	return defaultSigV4Region
}

func cognitoUserPoolsIDFromResourceARN(resourceARN string) (string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return "", validationCognitoUserPools("ResourceArn is required")
	}

	if !strings.HasPrefix(resourceARN, "arn:aws:cognito-idp:") {
		return "", validationCognitoUserPools("ResourceArn is invalid")
	}

	parts := strings.Split(resourceARN, ":")
	if len(parts) < 6 {
		return "", validationCognitoUserPools("ResourceArn is invalid")
	}
	resource := strings.TrimSpace(parts[5])
	if !strings.HasPrefix(resource, "userpool/") {
		return "", validationCognitoUserPools("ResourceArn is invalid")
	}
	id := strings.TrimSpace(strings.TrimPrefix(resource, "userpool/"))
	if id == "" {
		return "", validationCognitoUserPools("ResourceArn is invalid")
	}
	return id, nil
}
