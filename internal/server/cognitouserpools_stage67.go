package server

import (
	"net/http"
	"sort"
	"strings"
)

func (s *Server) handleCognitoUserPoolsStage6Action(w http.ResponseWriter, action string, payload map[string]any) bool {
	switch action {
	case "CreateIdentityProvider":
		providerDetails, ok := cognitoUserPoolsStringMap(payload["ProviderDetails"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ProviderDetails")
			return true
		}
		attributeMapping, ok := cognitoUserPoolsStringMap(payload["AttributeMapping"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid AttributeMapping")
			return true
		}
		idpIdentifiers, ok := cognitoUserPoolsStringSlice(payload["IdpIdentifiers"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid IdpIdentifiers")
			return true
		}
		record, err := s.cognitouserpools.CreateIdentityProvider(cognitoUserPoolsCreateIdentityProviderInput{
			UserPoolID:       cognitoUserPoolsString(payload["UserPoolId"]),
			ProviderName:     cognitoUserPoolsString(payload["ProviderName"]),
			ProviderType:     cognitoUserPoolsString(payload["ProviderType"]),
			ProviderDetails:  providerDetails,
			AttributeMapping: attributeMapping,
			IdpIdentifiers:   idpIdentifiers,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"IdentityProvider": cognitoUserPoolsIdentityProviderPayload(record)})
		return true

	case "DescribeIdentityProvider":
		record, err := s.cognitouserpools.DescribeIdentityProvider(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ProviderName"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"IdentityProvider": cognitoUserPoolsIdentityProviderPayload(record)})
		return true

	case "GetIdentityProviderByIdentifier":
		record, err := s.cognitouserpools.GetIdentityProviderByIdentifier(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["IdpIdentifier"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"IdentityProvider": cognitoUserPoolsIdentityProviderPayload(record)})
		return true

	case "ListIdentityProviders":
		maxResults, ok := cognitoUserPoolsIntDefault(payload, "MaxResults", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListIdentityProviders(
			cognitoUserPoolsString(payload["UserPoolId"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Providers": cognitoUserPoolsIdentityProvidersPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateIdentityProvider":
		input := cognitoUserPoolsUpdateIdentityProviderInput{
			UserPoolID:   cognitoUserPoolsString(payload["UserPoolId"]),
			ProviderName: cognitoUserPoolsString(payload["ProviderName"]),
		}
		if raw, exists := cognitoUserPoolsField(payload, "ProviderDetails"); exists {
			value, ok := cognitoUserPoolsStringMap(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ProviderDetails")
				return true
			}
			input.ProviderDetailsSet = true
			input.ProviderDetails = value
		}
		if raw, exists := cognitoUserPoolsField(payload, "AttributeMapping"); exists {
			value, ok := cognitoUserPoolsStringMap(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid AttributeMapping")
				return true
			}
			input.AttributeMappingSet = true
			input.AttributeMapping = value
		}
		if raw, exists := cognitoUserPoolsField(payload, "IdpIdentifiers"); exists {
			value, ok := cognitoUserPoolsStringSlice(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid IdpIdentifiers")
				return true
			}
			input.IdpIdentifiersSet = true
			input.IdpIdentifiers = value
		}
		record, err := s.cognitouserpools.UpdateIdentityProvider(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"IdentityProvider": cognitoUserPoolsIdentityProviderPayload(record)})
		return true

	case "DeleteIdentityProvider":
		if err := s.cognitouserpools.DeleteIdentityProvider(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ProviderName"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "SetRiskConfiguration":
		compromised, ok := cognitoUserPoolsMapAnyOrEmpty(payload["CompromisedCredentialsRiskConfiguration"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid CompromisedCredentialsRiskConfiguration")
			return true
		}
		takeover, ok := cognitoUserPoolsMapAnyOrEmpty(payload["AccountTakeoverRiskConfiguration"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid AccountTakeoverRiskConfiguration")
			return true
		}
		exceptions, ok := cognitoUserPoolsMapAnyOrEmpty(payload["RiskExceptionConfiguration"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid RiskExceptionConfiguration")
			return true
		}
		record, err := s.cognitouserpools.SetRiskConfiguration(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
			compromised,
			takeover,
			exceptions,
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"RiskConfiguration": cognitoUserPoolsRiskConfigurationPayload(record)})
		return true

	case "DescribeRiskConfiguration":
		record, err := s.cognitouserpools.DescribeRiskConfiguration(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"RiskConfiguration": cognitoUserPoolsRiskConfigurationPayload(record)})
		return true

	case "SetLogDeliveryConfiguration":
		logConfigurations, ok := cognitoUserPoolsSliceMapAny(payload["LogConfigurations"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid LogConfigurations")
			return true
		}
		record, err := s.cognitouserpools.SetLogDeliveryConfiguration(
			cognitoUserPoolsString(payload["UserPoolId"]),
			logConfigurations,
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsLogDeliveryConfigurationPayload(record))
		return true

	case "GetLogDeliveryConfiguration":
		record, err := s.cognitouserpools.GetLogDeliveryConfiguration(cognitoUserPoolsString(payload["UserPoolId"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsLogDeliveryConfigurationPayload(record))
		return true

	case "AdminListUserAuthEvents":
		maxResults, ok := cognitoUserPoolsIntDefault(payload, "MaxResults", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.AdminListUserAuthEvents(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"AuthEvents": cognitoUserPoolsAuthEventsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateAuthEventFeedback":
		if err := s.cognitouserpools.UpdateAuthEventFeedback(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["EventId"]),
			cognitoUserPoolsString(payload["FeedbackValue"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminUpdateAuthEventFeedback":
		if err := s.cognitouserpools.AdminUpdateAuthEventFeedback(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["EventId"]),
			cognitoUserPoolsString(payload["FeedbackValue"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	return false
}

func cognitoUserPoolsMapAnyOrEmpty(value any) (map[string]any, bool) {
	if value == nil {
		return map[string]any{}, true
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	return cloneCognitoUserPoolsMapAny(obj), true
}

func cognitoUserPoolsSliceMapAny(value any) ([]map[string]any, bool) {
	if value == nil {
		return []map[string]any{}, true
	}
	rawItems, ok := value.([]any)
	if !ok {
		switch typed := value.(type) {
		case []map[string]any:
			return cloneCognitoUserPoolsSliceMapAny(typed), true
		default:
			return nil, false
		}
	}
	out := make([]map[string]any, 0, len(rawItems))
	for _, item := range rawItems {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		out = append(out, cloneCognitoUserPoolsMapAny(obj))
	}
	return out, true
}

func cognitoUserPoolsIdentityProviderPayload(record cognitoUserPoolsIdentityProviderRecord) map[string]any {
	out := map[string]any{
		"UserPoolId":       record.UserPoolID,
		"ProviderName":     record.ProviderName,
		"ProviderType":     record.ProviderType,
		"ProviderDetails":  cloneStringMap(record.ProviderDetails),
		"AttributeMapping": cloneStringMap(record.AttributeMapping),
		"LastModifiedDate": float64(record.UpdatedAt.Unix()),
		"CreationDate":     float64(record.CreatedAt.Unix()),
	}
	if len(record.IdpIdentifiers) > 0 {
		out["IdpIdentifiers"] = cloneStringSlice(record.IdpIdentifiers)
	}
	return out
}

func cognitoUserPoolsIdentityProvidersPayload(records []cognitoUserPoolsIdentityProviderRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsIdentityProviderPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["ProviderName"].(string)
		right, _ := out[j]["ProviderName"].(string)
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return out
}

func cognitoUserPoolsRiskConfigurationPayload(record cognitoUserPoolsRiskConfigurationRecord) map[string]any {
	out := map[string]any{
		"UserPoolId": record.UserPoolID,
		"CompromisedCredentialsRiskConfiguration": cloneCognitoUserPoolsMapAny(record.CompromisedCredentialsRiskConfiguration),
		"AccountTakeoverRiskConfiguration":        cloneCognitoUserPoolsMapAny(record.AccountTakeoverRiskConfiguration),
		"RiskExceptionConfiguration":              cloneCognitoUserPoolsMapAny(record.RiskExceptionConfiguration),
	}
	if record.ClientID != "" {
		out["ClientId"] = record.ClientID
	}
	return out
}

func cognitoUserPoolsLogDeliveryConfigurationPayload(record cognitoUserPoolsLogDeliveryConfigurationRecord) map[string]any {
	return map[string]any{
		"UserPoolId":        record.UserPoolID,
		"LogConfigurations": cloneCognitoUserPoolsSliceMapAny(record.LogConfigurations),
	}
}

func cognitoUserPoolsAuthEventsPayload(records []cognitoUserPoolsAuthEventRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		event := map[string]any{
			"EventId":       record.EventID,
			"EventType":     record.EventType,
			"EventResponse": record.EventResponse,
			"CreationDate":  float64(record.CreationDate.Unix()),
			"EventRisk":     cloneCognitoUserPoolsMapAny(record.EventRisk),
		}
		if record.FeedbackValue != "" {
			feedback := map[string]any{
				"FeedbackValue": record.FeedbackValue,
				"Provider":      "COGNITO",
			}
			if record.FeedbackDate != nil {
				feedback["FeedbackDate"] = float64(record.FeedbackDate.Unix())
			}
			event["EventFeedback"] = feedback
		}
		out = append(out, event)
	}
	return out
}
