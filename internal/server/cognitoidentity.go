package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type cognitoIdentityError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCognitoIdentityJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCognitoIdentityJSONCandidate(r) {
		return false
	}

	action := parseCognitoIdentityTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := cognitoIdentityOperationByName[action]; !known {
		respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if action != "UnlinkIdentity" || cognitoIdentityHasSigV4AuthMaterial(r) {
		ok, status, code, msg, _ := s.validateSigV4WithService(r, "cognito-identity")
		if !ok {
			respondCognitoIdentityError(w, status, code, msg)
			return true
		}
	}

	payload, err := parseCognitoIdentityPayload(r)
	if err != nil {
		respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	// Stage 1: identity pool lifecycle.
	case "CreateIdentityPool":
		allowUnauthenticatedRaw, ok := cognitoIdentityField(payload, "AllowUnauthenticatedIdentities")
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "AllowUnauthenticatedIdentities is required")
			return true
		}
		allowUnauthenticated, ok := cognitoIdentityBool(allowUnauthenticatedRaw)
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid AllowUnauthenticatedIdentities")
			return true
		}

		allowClassicFlow := false
		if allowClassicFlowRaw, ok := cognitoIdentityField(payload, "AllowClassicFlow"); ok {
			allowClassicFlow, ok = cognitoIdentityBool(allowClassicFlowRaw)
			if !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid AllowClassicFlow")
				return true
			}
		}

		supportedLoginProviders, ok := cognitoIdentityStringMap(payload["SupportedLoginProviders"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid SupportedLoginProviders")
			return true
		}
		openIDConnectProviderARNs, ok := cognitoIdentityStringSlice(payload["OpenIdConnectProviderARNs"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid OpenIdConnectProviderARNs")
			return true
		}
		cognitoProviders, ok := cognitoIdentityProviders(payload["CognitoIdentityProviders"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid CognitoIdentityProviders")
			return true
		}
		samlProviderARNs, ok := cognitoIdentityStringSlice(payload["SamlProviderARNs"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid SamlProviderARNs")
			return true
		}
		identityPoolTags, ok := cognitoIdentityStringMap(payload["IdentityPoolTags"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid IdentityPoolTags")
			return true
		}

		record, err := s.cognitoidentity.CreatePool(cognitoIdentityCreatePoolInput{
			Region:                         cognitoIdentityRegionFromRequest(r),
			IdentityPoolName:               cognitoIdentityString(payload["IdentityPoolName"]),
			AllowUnauthenticatedIdentities: allowUnauthenticated,
			AllowClassicFlow:               allowClassicFlow,
			SupportedLoginProviders:        supportedLoginProviders,
			DeveloperProviderName:          cognitoIdentityString(payload["DeveloperProviderName"]),
			OpenIDConnectProviderARNs:      openIDConnectProviderARNs,
			CognitoIdentityProviders:       cognitoProviders,
			SamlProviderARNs:               samlProviderARNs,
			IdentityPoolTags:               identityPoolTags,
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityPoolPayload(record))
		return true

	case "DescribeIdentityPool":
		record, err := s.cognitoidentity.DescribePool(cognitoIdentityString(payload["IdentityPoolId"]))
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityPoolPayload(record))
		return true

	case "ListIdentityPools":
		maxResults, ok := cognitoIdentityInt(payload["MaxResults"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitoidentity.ListPools(maxResults, cognitoIdentityString(payload["NextToken"]))
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		out := map[string]any{
			"IdentityPools": cognitoIdentityPoolShortDescriptionsPayload(records),
		}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoIdentityJSON(w, http.StatusOK, out)
		return true

	case "UpdateIdentityPool":
		allowUnauthenticatedRaw, ok := cognitoIdentityField(payload, "AllowUnauthenticatedIdentities")
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "AllowUnauthenticatedIdentities is required")
			return true
		}
		allowUnauthenticated, ok := cognitoIdentityBool(allowUnauthenticatedRaw)
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid AllowUnauthenticatedIdentities")
			return true
		}

		allowClassicFlow := false
		if allowClassicFlowRaw, ok := cognitoIdentityField(payload, "AllowClassicFlow"); ok {
			allowClassicFlow, ok = cognitoIdentityBool(allowClassicFlowRaw)
			if !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid AllowClassicFlow")
				return true
			}
		}

		supportedLoginProviders, ok := cognitoIdentityStringMap(payload["SupportedLoginProviders"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid SupportedLoginProviders")
			return true
		}
		openIDConnectProviderARNs, ok := cognitoIdentityStringSlice(payload["OpenIdConnectProviderARNs"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid OpenIdConnectProviderARNs")
			return true
		}
		cognitoProviders, ok := cognitoIdentityProviders(payload["CognitoIdentityProviders"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid CognitoIdentityProviders")
			return true
		}
		samlProviderARNs, ok := cognitoIdentityStringSlice(payload["SamlProviderARNs"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid SamlProviderARNs")
			return true
		}
		identityPoolTags, ok := cognitoIdentityStringMap(payload["IdentityPoolTags"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid IdentityPoolTags")
			return true
		}

		record, err := s.cognitoidentity.UpdatePool(cognitoIdentityUpdatePoolInput{
			IdentityPoolID:                 cognitoIdentityString(payload["IdentityPoolId"]),
			IdentityPoolName:               cognitoIdentityString(payload["IdentityPoolName"]),
			AllowUnauthenticatedIdentities: allowUnauthenticated,
			AllowClassicFlow:               allowClassicFlow,
			SupportedLoginProviders:        supportedLoginProviders,
			DeveloperProviderName:          cognitoIdentityString(payload["DeveloperProviderName"]),
			OpenIDConnectProviderARNs:      openIDConnectProviderARNs,
			CognitoIdentityProviders:       cognitoProviders,
			SamlProviderARNs:               samlProviderARNs,
			IdentityPoolTags:               identityPoolTags,
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityPoolPayload(record))
		return true

	case "DeleteIdentityPool":
		if err := s.cognitoidentity.DeletePool(cognitoIdentityString(payload["IdentityPoolId"])); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	// Stage 2: identity and credential retrieval.
	case "GetId":
		logins, ok := cognitoIdentityStringMap(payload["Logins"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Logins")
			return true
		}
		record, err := s.cognitoidentity.GetOrCreateIdentity(cognitoIdentityString(payload["IdentityPoolId"]), logins)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"IdentityId": record.IdentityID,
		})
		return true

	case "GetCredentialsForIdentity":
		if _, exists := cognitoIdentityField(payload, "Logins"); exists {
			if _, ok := cognitoIdentityStringMap(payload["Logins"]); !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Logins")
				return true
			}
		}
		identityID := cognitoIdentityString(payload["IdentityId"])
		credentials, err := s.cognitoidentity.GetCredentials(identityID)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"IdentityId": identityID,
			"Credentials": map[string]any{
				"AccessKeyId":  credentials.AccessKeyID,
				"SecretKey":    credentials.SecretKey,
				"SessionToken": credentials.SessionToken,
				"Expiration":   float64(credentials.Expiration),
			},
		})
		return true

	case "GetOpenIdToken":
		if _, exists := cognitoIdentityField(payload, "Logins"); exists {
			if _, ok := cognitoIdentityStringMap(payload["Logins"]); !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Logins")
				return true
			}
		}
		identityID := cognitoIdentityString(payload["IdentityId"])
		token, err := s.cognitoidentity.GetOpenIDToken(identityID)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"IdentityId": identityID,
			"Token":      token,
		})
		return true

	case "DescribeIdentity":
		record, err := s.cognitoidentity.DescribeIdentity(cognitoIdentityString(payload["IdentityId"]))
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityDescriptionPayload(record))
		return true

	case "ListIdentities":
		maxResults, ok := cognitoIdentityInt(payload["MaxResults"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		hideDisabled := false
		if hideDisabledRaw, exists := cognitoIdentityField(payload, "HideDisabled"); exists {
			hideDisabled, ok = cognitoIdentityBool(hideDisabledRaw)
			if !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid HideDisabled")
				return true
			}
		}
		identityPoolID := cognitoIdentityString(payload["IdentityPoolId"])
		records, nextToken, err := s.cognitoidentity.ListIdentities(
			identityPoolID,
			maxResults,
			cognitoIdentityString(payload["NextToken"]),
			hideDisabled,
		)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		out := map[string]any{
			"IdentityPoolId": identityPoolID,
			"Identities":     cognitoIdentityDescriptionsPayload(records),
		}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoIdentityJSON(w, http.StatusOK, out)
		return true

	// Stage 3: developer-authenticated identity workflows.
	case "GetOpenIdTokenForDeveloperIdentity":
		logins, ok := cognitoIdentityStringMap(payload["Logins"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Logins")
			return true
		}
		identityID, token, err := s.cognitoidentity.GetOpenIDTokenForDeveloperIdentity(cognitoIdentityGetOpenIDTokenForDeveloperIdentityInput{
			IdentityPoolID: cognitoIdentityString(payload["IdentityPoolId"]),
			IdentityID:     cognitoIdentityString(payload["IdentityId"]),
			Logins:         logins,
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"IdentityId": identityID,
			"Token":      token,
		})
		return true

	case "LookupDeveloperIdentity":
		maxResults := 0
		if raw, exists := cognitoIdentityField(payload, "MaxResults"); exists {
			value, ok := cognitoIdentityInt(raw)
			if !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
				return true
			}
			maxResults = value
		}
		out, err := s.cognitoidentity.LookupDeveloperIdentity(cognitoIdentityLookupDeveloperIdentityInput{
			IdentityPoolID:          cognitoIdentityString(payload["IdentityPoolId"]),
			IdentityID:              cognitoIdentityString(payload["IdentityId"]),
			DeveloperUserIdentifier: cognitoIdentityString(payload["DeveloperUserIdentifier"]),
			NextToken:               cognitoIdentityString(payload["NextToken"]),
			MaxResults:              maxResults,
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"IdentityId":                  out.IdentityID,
			"DeveloperUserIdentifierList": out.DeveloperUserIdentifierList,
		}
		if out.NextToken != "" {
			response["NextToken"] = out.NextToken
		}
		respondCognitoIdentityJSON(w, http.StatusOK, response)
		return true

	case "MergeDeveloperIdentities":
		identityID, err := s.cognitoidentity.MergeDeveloperIdentities(cognitoIdentityMergeDeveloperIdentitiesInput{
			IdentityPoolID:                    cognitoIdentityString(payload["IdentityPoolId"]),
			DeveloperProviderName:             cognitoIdentityString(payload["DeveloperProviderName"]),
			DestinationUserIdentifierForMerge: cognitoIdentityString(payload["DestinationUserIdentifierForMerge"]),
			SourceUserIdentifier:              cognitoIdentityString(payload["SourceUserIdentifier"]),
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"IdentityId": identityID,
		})
		return true

	case "UnlinkDeveloperIdentity":
		if err := s.cognitoidentity.UnlinkDeveloperIdentity(cognitoIdentityUnlinkDeveloperIdentityInput{
			IdentityPoolID:          cognitoIdentityString(payload["IdentityPoolId"]),
			IdentityID:              cognitoIdentityString(payload["IdentityId"]),
			DeveloperProviderName:   cognitoIdentityString(payload["DeveloperProviderName"]),
			DeveloperUserIdentifier: cognitoIdentityString(payload["DeveloperUserIdentifier"]),
		}); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteIdentities":
		identityIDs, ok := cognitoIdentityStringSlice(payload["IdentityIdsToDelete"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid IdentityIdsToDelete")
			return true
		}
		unprocessed, err := s.cognitoidentity.DeleteIdentities(identityIDs)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(unprocessed))
		for _, entry := range unprocessed {
			out = append(out, map[string]any{
				"IdentityId": entry.IdentityID,
				"ErrorCode":  entry.ErrorCode,
			})
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"UnprocessedIdentityIds": out,
		})
		return true

	// Stage 4: role and principal-tag mapping management.
	case "GetIdentityPoolRoles":
		roles, roleMappings, err := s.cognitoidentity.GetIdentityPoolRoles(cognitoIdentityString(payload["IdentityPoolId"]))
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"IdentityPoolId": cognitoIdentityString(payload["IdentityPoolId"]),
			"Roles":          roles,
		}
		if len(roleMappings) > 0 {
			response["RoleMappings"] = cognitoIdentityRoleMappingsPayload(roleMappings)
		}
		respondCognitoIdentityJSON(w, http.StatusOK, response)
		return true

	case "SetIdentityPoolRoles":
		roles, ok := cognitoIdentityStringMap(payload["Roles"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Roles")
			return true
		}
		roleMappings, ok := cognitoIdentityRoleMappings(payload["RoleMappings"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid RoleMappings")
			return true
		}
		if err := s.cognitoidentity.SetIdentityPoolRoles(
			cognitoIdentityString(payload["IdentityPoolId"]),
			roles,
			roleMappings,
		); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetPrincipalTagAttributeMap":
		value, err := s.cognitoidentity.GetPrincipalTagAttributeMap(
			cognitoIdentityString(payload["IdentityPoolId"]),
			cognitoIdentityString(payload["IdentityProviderName"]),
		)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityPrincipalTagAttributeMapPayload(value))
		return true

	case "SetPrincipalTagAttributeMap":
		principalTags, ok := cognitoIdentityStringMap(payload["PrincipalTags"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid PrincipalTags")
			return true
		}
		useDefaults := false
		if raw, exists := cognitoIdentityField(payload, "UseDefaults"); exists {
			value, ok := cognitoIdentityBool(raw)
			if !ok {
				respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid UseDefaults")
				return true
			}
			useDefaults = value
		}
		value, err := s.cognitoidentity.SetPrincipalTagAttributeMap(cognitoIdentityPrincipalTagAttributeMap{
			IdentityPoolID:       cognitoIdentityString(payload["IdentityPoolId"]),
			IdentityProviderName: cognitoIdentityString(payload["IdentityProviderName"]),
			PrincipalTags:        principalTags,
			UseDefaults:          useDefaults,
		})
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, cognitoIdentityPrincipalTagAttributeMapPayload(value))
		return true

	// Stage 5: identity unlink and resource-tag APIs.
	case "UnlinkIdentity":
		logins, ok := cognitoIdentityStringMap(payload["Logins"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Logins")
			return true
		}
		loginsToRemove, ok := cognitoIdentityStringSlice(payload["LoginsToRemove"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid LoginsToRemove")
			return true
		}
		if err := s.cognitoidentity.UnlinkIdentity(
			cognitoIdentityString(payload["IdentityId"]),
			logins,
			loginsToRemove,
		); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	case "TagResource":
		poolID, err := cognitoIdentityPoolIDFromResourceARN(cognitoIdentityString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoIdentityError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tags, ok := cognitoIdentityStringMap(payload["Tags"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid Tags")
			return true
		}
		if err := s.cognitoidentity.TagIdentityPool(poolID, tags); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		poolID, err := cognitoIdentityPoolIDFromResourceARN(cognitoIdentityString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoIdentityError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tagKeys, ok := cognitoIdentityStringSlice(payload["TagKeys"])
		if !ok {
			respondCognitoIdentityError(w, http.StatusBadRequest, "ValidationException", "invalid TagKeys")
			return true
		}
		if err := s.cognitoidentity.UntagIdentityPool(poolID, tagKeys); err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListTagsForResource":
		poolID, err := cognitoIdentityPoolIDFromResourceARN(cognitoIdentityString(payload["ResourceArn"]))
		if err != nil {
			respondCognitoIdentityError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
			return true
		}
		tags, err := s.cognitoidentity.ListIdentityPoolTags(poolID)
		if err != nil {
			respondCognitoIdentityErrorForErr(w, err)
			return true
		}
		respondCognitoIdentityJSON(w, http.StatusOK, map[string]any{
			"Tags": tags,
		})
		return true
	}

	respondCognitoIdentityError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isCognitoIdentityJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "com.amazonaws.cognito.identity.model.AWSCognitoIdentityService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSCognitoIdentityService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "cognito-identity" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".cognito-identity.") || strings.HasPrefix(host, "cognito-identity.")
}

func parseCognitoIdentityTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "com.amazonaws.cognito.identity.model.AWSCognitoIdentityService.") {
		return strings.TrimPrefix(target, "com.amazonaws.cognito.identity.model.AWSCognitoIdentityService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCognitoIdentityPayload(r *http.Request) (map[string]any, error) {
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

func respondCognitoIdentityJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCognitoIdentityError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCognitoIdentityJSON(w, status, cognitoIdentityError{Type: code, Message: msg})
}

func respondCognitoIdentityErrorForErr(w http.ResponseWriter, err error) {
	if apiErr := asCognitoIdentityAPIError(err); apiErr != nil {
		respondCognitoIdentityError(w, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	respondCognitoIdentityError(w, http.StatusInternalServerError, "InternalErrorException", err.Error())
}

func cognitoIdentityField(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	value, ok := payload[key]
	return value, ok
}

func cognitoIdentityString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func cognitoIdentityBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	default:
		return false, false
	}
}

func cognitoIdentityInt(value any) (int, bool) {
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
		parsed, err := strconv.Atoi(typed.String())
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func cognitoIdentityStringMap(value any) (map[string]string, bool) {
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
			rawValue, ok := raw.(string)
			if !ok {
				return nil, false
			}
			out[strings.TrimSpace(key)] = strings.TrimSpace(rawValue)
		}
		return out, true
	default:
		return nil, false
	}
}

func cognitoIdentityStringSlice(value any) ([]string, bool) {
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
			out = append(out, strings.TrimSpace(asString))
		}
		return out, true
	default:
		return nil, false
	}
}

func cognitoIdentityProviders(value any) ([]cognitoIdentityProvider, bool) {
	if value == nil {
		return nil, true
	}

	rawProviders, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]cognitoIdentityProvider); ok {
			return cloneCognitoIdentityProviders(typed), true
		}
		return nil, false
	}

	out := make([]cognitoIdentityProvider, 0, len(rawProviders))
	for _, raw := range rawProviders {
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		serverSideTokenCheck := false
		if value, ok := obj["ServerSideTokenCheck"]; ok {
			boolValue, ok := cognitoIdentityBool(value)
			if !ok {
				return nil, false
			}
			serverSideTokenCheck = boolValue
		}
		out = append(out, cognitoIdentityProvider{
			ProviderName:         cognitoIdentityString(obj["ProviderName"]),
			ClientID:             cognitoIdentityString(obj["ClientId"]),
			ServerSideTokenCheck: serverSideTokenCheck,
		})
	}
	return out, true
}

func cognitoIdentityPoolPayload(record cognitoIdentityPoolRecord) map[string]any {
	out := map[string]any{
		"IdentityPoolId":                 record.IdentityPoolID,
		"IdentityPoolName":               record.IdentityPoolName,
		"AllowUnauthenticatedIdentities": record.AllowUnauthenticatedIdentities,
		"AllowClassicFlow":               record.AllowClassicFlow,
	}
	if len(record.SupportedLoginProviders) > 0 {
		out["SupportedLoginProviders"] = cloneStringMap(record.SupportedLoginProviders)
	}
	if record.DeveloperProviderName != "" {
		out["DeveloperProviderName"] = record.DeveloperProviderName
	}
	if len(record.OpenIDConnectProviderARNs) > 0 {
		out["OpenIdConnectProviderARNs"] = cloneStringSlice(record.OpenIDConnectProviderARNs)
	}
	if len(record.CognitoIdentityProviders) > 0 {
		providers := make([]map[string]any, 0, len(record.CognitoIdentityProviders))
		for _, provider := range record.CognitoIdentityProviders {
			providers = append(providers, map[string]any{
				"ProviderName":         provider.ProviderName,
				"ClientId":             provider.ClientID,
				"ServerSideTokenCheck": provider.ServerSideTokenCheck,
			})
		}
		out["CognitoIdentityProviders"] = providers
	}
	if len(record.SamlProviderARNs) > 0 {
		out["SamlProviderARNs"] = cloneStringSlice(record.SamlProviderARNs)
	}
	if len(record.IdentityPoolTags) > 0 {
		out["IdentityPoolTags"] = cloneStringMap(record.IdentityPoolTags)
	}
	return out
}

func cognitoIdentityPoolShortDescriptionsPayload(records []cognitoIdentityPoolRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{
			"IdentityPoolId":   record.IdentityPoolID,
			"IdentityPoolName": record.IdentityPoolName,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["IdentityPoolId"].(string)
		right, _ := out[j]["IdentityPoolId"].(string)
		return left < right
	})
	return out
}

func cognitoIdentityDescriptionPayload(record cognitoIdentityRecord) map[string]any {
	out := map[string]any{
		"IdentityId":       record.IdentityID,
		"CreationDate":     float64(record.CreationDate.Unix()),
		"LastModifiedDate": float64(record.LastModified.Unix()),
	}
	if record.Disabled {
		out["Disabled"] = true
	}
	if len(record.Logins) > 0 {
		logins := make([]string, 0, len(record.Logins))
		for login := range record.Logins {
			logins = append(logins, login)
		}
		sort.Strings(logins)
		out["Logins"] = logins
	}
	return out
}

func cognitoIdentityDescriptionsPayload(records []cognitoIdentityRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoIdentityDescriptionPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["IdentityId"].(string)
		right, _ := out[j]["IdentityId"].(string)
		return left < right
	})
	return out
}

func cognitoIdentityRegionFromRequest(r *http.Request) string {
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

func cognitoIdentityHasSigV4AuthMaterial(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}
	query := r.URL.Query()
	return strings.TrimSpace(query.Get("X-Amz-Algorithm")) != "" ||
		strings.TrimSpace(query.Get("X-Amz-Credential")) != "" ||
		strings.TrimSpace(query.Get("X-Amz-Signature")) != ""
}

func cognitoIdentityRoleMappings(value any) (map[string]cognitoIdentityRoleMapping, bool) {
	if value == nil {
		return nil, true
	}
	rawMappings, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	if len(rawMappings) == 0 {
		return map[string]cognitoIdentityRoleMapping{}, true
	}

	out := make(map[string]cognitoIdentityRoleMapping, len(rawMappings))
	for rawKey, rawValue := range rawMappings {
		key := strings.TrimSpace(rawKey)
		obj, ok := rawValue.(map[string]any)
		if !ok {
			return nil, false
		}
		roleMapping := cognitoIdentityRoleMapping{
			Type:                    cognitoIdentityString(obj["Type"]),
			AmbiguousRoleResolution: cognitoIdentityString(obj["AmbiguousRoleResolution"]),
		}
		if rawRulesConfig, exists := obj["RulesConfiguration"]; exists {
			rulesConfig, ok := rawRulesConfig.(map[string]any)
			if !ok {
				return nil, false
			}
			rawRules, ok := rulesConfig["Rules"].([]any)
			if !ok {
				return nil, false
			}
			rules := make([]cognitoIdentityMappingRule, 0, len(rawRules))
			for _, rawRule := range rawRules {
				ruleObject, ok := rawRule.(map[string]any)
				if !ok {
					return nil, false
				}
				rules = append(rules, cognitoIdentityMappingRule{
					Claim:     cognitoIdentityString(ruleObject["Claim"]),
					MatchType: cognitoIdentityString(ruleObject["MatchType"]),
					RoleARN:   cognitoIdentityString(ruleObject["RoleARN"]),
					Value:     cognitoIdentityString(ruleObject["Value"]),
				})
			}
			roleMapping.RulesConfiguration = &cognitoIdentityRulesConfiguration{
				Rules: rules,
			}
		}
		out[key] = roleMapping
	}
	return out, true
}

func cognitoIdentityRoleMappingsPayload(mappings map[string]cognitoIdentityRoleMapping) map[string]any {
	if len(mappings) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(mappings))
	for key, mapping := range mappings {
		entry := map[string]any{
			"Type":                    mapping.Type,
			"AmbiguousRoleResolution": mapping.AmbiguousRoleResolution,
		}
		if mapping.RulesConfiguration != nil {
			rules := make([]map[string]any, 0, len(mapping.RulesConfiguration.Rules))
			for _, rule := range mapping.RulesConfiguration.Rules {
				rules = append(rules, map[string]any{
					"Claim":     rule.Claim,
					"MatchType": rule.MatchType,
					"RoleARN":   rule.RoleARN,
					"Value":     rule.Value,
				})
			}
			entry["RulesConfiguration"] = map[string]any{
				"Rules": rules,
			}
		}
		out[key] = entry
	}
	return out
}

func cognitoIdentityPrincipalTagAttributeMapPayload(value cognitoIdentityPrincipalTagAttributeMap) map[string]any {
	out := map[string]any{
		"IdentityPoolId":       value.IdentityPoolID,
		"IdentityProviderName": value.IdentityProviderName,
		"UseDefaults":          value.UseDefaults,
	}
	if len(value.PrincipalTags) > 0 {
		out["PrincipalTags"] = cloneStringMap(value.PrincipalTags)
	} else {
		out["PrincipalTags"] = map[string]string{}
	}
	return out
}

func cognitoIdentityPoolIDFromResourceARN(resourceARN string) (string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return "", validationCognitoIdentity("ResourceArn is required")
	}

	parts := strings.SplitN(resourceARN, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || strings.TrimSpace(parts[2]) != "cognito-identity" {
		return "", validationCognitoIdentity("ResourceArn is invalid")
	}
	resource := strings.TrimSpace(parts[5])
	if !strings.HasPrefix(resource, "identitypool/") {
		return "", validationCognitoIdentity("ResourceArn is invalid")
	}
	identityPoolID := strings.TrimSpace(strings.TrimPrefix(resource, "identitypool/"))
	if identityPoolID == "" {
		return "", validationCognitoIdentity("ResourceArn is invalid")
	}
	return identityPoolID, nil
}
