package server

import (
	"net/http"
	"sort"
	"strings"
)

func (s *Server) handleCognitoUserPoolsStage7Action(w http.ResponseWriter, action string, payload map[string]any) bool {
	switch action {
	case "AddCustomAttributes":
		customAttributes, ok := cognitoUserPoolsCustomAttributes(payload["CustomAttributes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid CustomAttributes")
			return true
		}
		if err := s.cognitouserpools.AddCustomAttributes(cognitoUserPoolsString(payload["UserPoolId"]), customAttributes); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UpdateUserPoolDomain":
		version, versionSet, ok := cognitoUserPoolsIntPointer(payload, "ManagedLoginVersion")
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ManagedLoginVersion")
			return true
		}
		customDomainConfig, _ := payload["CustomDomainConfig"].(map[string]any)
		certificateARN := cognitoUserPoolsString(customDomainConfig["CertificateArn"])
		if !versionSet {
			version = nil
		}
		if _, err := s.cognitouserpools.UpdateUserPoolDomain(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Domain"]),
			version,
			certificateARN,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "SetUICustomization":
		record, err := s.cognitouserpools.SetUICustomization(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["CSS"]),
			cognitoUserPoolsString(payload["ImageFile"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UICustomization": cognitoUserPoolsUICustomizationPayload(record)})
		return true

	case "GetUICustomization":
		record, err := s.cognitouserpools.GetUICustomization(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UICustomization": cognitoUserPoolsUICustomizationPayload(record)})
		return true

	case "GetSigningCertificate":
		certificate, err := s.cognitouserpools.GetSigningCertificate(cognitoUserPoolsString(payload["UserPoolId"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Certificate": certificate})
		return true

	case "CreateManagedLoginBranding":
		useCognitoProvidedValues, useCognitoProvidedValuesSet := payload["UseCognitoProvidedValues"]
		useProvidedValues := false
		if useCognitoProvidedValuesSet {
			parsed, ok := cognitoUserPoolsBool(useCognitoProvidedValues)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UseCognitoProvidedValues")
				return true
			}
			useProvidedValues = parsed
		}
		settings, ok := cognitoUserPoolsMapAnyOrEmpty(payload["Settings"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Settings")
			return true
		}
		assets, ok := cognitoUserPoolsSliceMapAny(payload["Assets"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Assets")
			return true
		}
		record, err := s.cognitouserpools.CreateManagedLoginBranding(cognitoUserPoolsCreateManagedLoginBrandingInput{
			UserPoolID:               cognitoUserPoolsString(payload["UserPoolId"]),
			ClientID:                 cognitoUserPoolsString(payload["ClientId"]),
			UseCognitoProvidedValues: useProvidedValues,
			Settings:                 settings,
			Assets:                   assets,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ManagedLoginBranding": cognitoUserPoolsManagedLoginBrandingPayload(record)})
		return true

	case "UpdateManagedLoginBranding":
		input := cognitoUserPoolsUpdateManagedLoginBrandingInput{
			UserPoolID:             cognitoUserPoolsString(payload["UserPoolId"]),
			ManagedLoginBrandingID: cognitoUserPoolsString(payload["ManagedLoginBrandingId"]),
		}
		if raw, exists := cognitoUserPoolsField(payload, "ClientId"); exists {
			input.ClientIDSet = true
			input.ClientID = cognitoUserPoolsString(raw)
		}
		if raw, exists := cognitoUserPoolsField(payload, "UseCognitoProvidedValues"); exists {
			value, ok := cognitoUserPoolsBool(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UseCognitoProvidedValues")
				return true
			}
			input.UseCognitoProvidedValuesSet = true
			input.UseCognitoProvidedValues = value
		}
		if raw, exists := cognitoUserPoolsField(payload, "Settings"); exists {
			value, ok := cognitoUserPoolsMapAnyOrEmpty(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Settings")
				return true
			}
			input.SettingsSet = true
			input.Settings = value
		}
		if raw, exists := cognitoUserPoolsField(payload, "Assets"); exists {
			value, ok := cognitoUserPoolsSliceMapAny(raw)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Assets")
				return true
			}
			input.AssetsSet = true
			input.Assets = value
		}
		record, err := s.cognitouserpools.UpdateManagedLoginBranding(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ManagedLoginBranding": cognitoUserPoolsManagedLoginBrandingPayload(record)})
		return true

	case "DescribeManagedLoginBranding":
		record, err := s.cognitouserpools.DescribeManagedLoginBranding(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ManagedLoginBrandingId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ManagedLoginBranding": cognitoUserPoolsManagedLoginBrandingPayload(record)})
		return true

	case "DescribeManagedLoginBrandingByClient":
		record, err := s.cognitouserpools.DescribeManagedLoginBrandingByClient(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ClientId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"ManagedLoginBranding": cognitoUserPoolsManagedLoginBrandingPayload(record)})
		return true

	case "DeleteManagedLoginBranding":
		if err := s.cognitouserpools.DeleteManagedLoginBranding(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["ManagedLoginBrandingId"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateTerms":
		record, err := s.cognitouserpools.CreateTerms(cognitoUserPoolsCreateTermsInput{
			UserPoolID:   cognitoUserPoolsString(payload["UserPoolId"]),
			TermsID:      cognitoUserPoolsString(payload["TermsId"]),
			TermsName:    cognitoUserPoolsString(payload["TermsName"]),
			TermsDetails: cognitoUserPoolsTermsDetails(payload),
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Terms": cognitoUserPoolsTermsPayload(record)})
		return true

	case "UpdateTerms":
		input := cognitoUserPoolsUpdateTermsInput{
			UserPoolID:   cognitoUserPoolsString(payload["UserPoolId"]),
			TermsID:      cognitoUserPoolsString(payload["TermsId"]),
			TermsDetails: cognitoUserPoolsTermsDetails(payload),
		}
		if raw, exists := cognitoUserPoolsField(payload, "TermsName"); exists {
			input.TermsNameSet = true
			input.TermsName = cognitoUserPoolsString(raw)
		}
		record, err := s.cognitouserpools.UpdateTerms(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Terms": cognitoUserPoolsTermsPayload(record)})
		return true

	case "DescribeTerms":
		record, err := s.cognitouserpools.DescribeTerms(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["TermsId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Terms": cognitoUserPoolsTermsPayload(record)})
		return true

	case "ListTerms":
		maxResults, ok := cognitoUserPoolsIntDefault(payload, "MaxResults", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListTerms(
			cognitoUserPoolsString(payload["UserPoolId"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"TermsList": cognitoUserPoolsTermsListPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "DeleteTerms":
		if err := s.cognitouserpools.DeleteTerms(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["TermsId"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminDisableProviderForUser":
		user, ok := cognitoUserPoolsProviderUserIdentifierFromAny(payload["User"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid User")
			return true
		}
		if err := s.cognitouserpools.AdminDisableProviderForUser(cognitoUserPoolsString(payload["UserPoolId"]), user); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminLinkProviderForUser":
		destinationUser, ok := cognitoUserPoolsProviderUserIdentifierFromAny(payload["DestinationUser"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid DestinationUser")
			return true
		}
		sourceUser, ok := cognitoUserPoolsProviderUserIdentifierFromAny(payload["SourceUser"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid SourceUser")
			return true
		}
		if err := s.cognitouserpools.AdminLinkProviderForUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			destinationUser,
			sourceUser,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminSetUserSettings":
		mfaOptions, ok := cognitoUserPoolsMFAOptions(payload["MFAOptions"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MFAOptions")
			return true
		}
		if err := s.cognitouserpools.AdminSetUserSettings(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			mfaOptions,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "SetUserSettings":
		mfaOptions, ok := cognitoUserPoolsMFAOptions(payload["MFAOptions"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MFAOptions")
			return true
		}
		if err := s.cognitouserpools.SetUserSettings(
			cognitoUserPoolsString(payload["AccessToken"]),
			mfaOptions,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	return false
}

func cognitoUserPoolsCustomAttributes(value any) ([]cognitoUserPoolsCustomAttributeRecord, bool) {
	if value == nil {
		return []cognitoUserPoolsCustomAttributeRecord{}, true
	}
	rawItems, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]cognitoUserPoolsCustomAttributeRecord, 0, len(rawItems))
	for _, item := range rawItems {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		numberConstraints, ok := cognitoUserPoolsStringMap(obj["NumberAttributeConstraints"])
		if !ok {
			return nil, false
		}
		stringConstraints, ok := cognitoUserPoolsStringMap(obj["StringAttributeConstraints"])
		if !ok {
			return nil, false
		}
		developerOnly, developerOnlySet := cognitoUserPoolsBool(obj["DeveloperOnlyAttribute"])
		mutable, mutableSet := cognitoUserPoolsBool(obj["Mutable"])
		required, requiredSet := cognitoUserPoolsBool(obj["Required"])
		record := cognitoUserPoolsCustomAttributeRecord{
			Name:                       cognitoUserPoolsString(obj["Name"]),
			AttributeDataType:          cognitoUserPoolsString(obj["AttributeDataType"]),
			NumberAttributeConstraints: numberConstraints,
			StringAttributeConstraints: stringConstraints,
		}
		if developerOnlySet {
			record.DeveloperOnlyAttribute = developerOnly
		}
		if mutableSet {
			record.Mutable = mutable
		}
		if requiredSet {
			record.Required = required
		}
		out = append(out, record)
	}
	return out, true
}

func cognitoUserPoolsProviderUserIdentifierFromAny(value any) (cognitoUserPoolsProviderUserIdentifier, bool) {
	obj, ok := value.(map[string]any)
	if !ok {
		return cognitoUserPoolsProviderUserIdentifier{}, false
	}
	return cognitoUserPoolsProviderUserIdentifier{
		ProviderName:           cognitoUserPoolsString(obj["ProviderName"]),
		ProviderAttributeName:  cognitoUserPoolsString(obj["ProviderAttributeName"]),
		ProviderAttributeValue: cognitoUserPoolsString(obj["ProviderAttributeValue"]),
	}, true
}

func cognitoUserPoolsMFAOptions(value any) ([]cognitoUserPoolsMFAOption, bool) {
	if value == nil {
		return []cognitoUserPoolsMFAOption{}, true
	}
	rawItems, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]cognitoUserPoolsMFAOption, 0, len(rawItems))
	for _, item := range rawItems {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		out = append(out, cognitoUserPoolsMFAOption{
			DeliveryMedium: cognitoUserPoolsString(obj["DeliveryMedium"]),
			AttributeName:  cognitoUserPoolsString(obj["AttributeName"]),
		})
	}
	return out, true
}

func cognitoUserPoolsUICustomizationPayload(record cognitoUserPoolsUICustomizationRecord) map[string]any {
	return map[string]any{
		"UserPoolId":       record.UserPoolID,
		"ClientId":         record.ClientID,
		"CSS":              record.CSS,
		"ImageUrl":         record.ImageURL,
		"CSSVersion":       record.CSSVersion,
		"CreationDate":     float64(record.CreationDateUTC.Unix()),
		"LastModifiedDate": float64(record.LastModifiedAt.Unix()),
	}
}

func cognitoUserPoolsManagedLoginBrandingPayload(record cognitoUserPoolsManagedLoginBrandingRecord) map[string]any {
	return map[string]any{
		"UserPoolId":               record.UserPoolID,
		"ManagedLoginBrandingId":   record.ManagedLoginBrandingID,
		"ClientId":                 record.ClientID,
		"UseCognitoProvidedValues": record.UseCognitoProvidedValues,
		"Settings":                 cloneCognitoUserPoolsMapAny(record.Settings),
		"Assets":                   cloneCognitoUserPoolsSliceMapAny(record.Assets),
		"CreationDate":             float64(record.CreationDate.Unix()),
		"LastModifiedDate":         float64(record.LastModifiedDate.Unix()),
	}
}

func cognitoUserPoolsTermsPayload(record cognitoUserPoolsTermsRecord) map[string]any {
	out := map[string]any{
		"UserPoolId":       record.UserPoolID,
		"TermsId":          record.TermsID,
		"TermsName":        record.TermsName,
		"CreationDate":     float64(record.CreationDate.Unix()),
		"LastModifiedDate": float64(record.LastModifiedDate.Unix()),
	}
	if len(record.TermsDetails) > 0 {
		out["TermsDetails"] = cloneCognitoUserPoolsMapAny(record.TermsDetails)
	}
	return out
}

func cognitoUserPoolsTermsListPayload(records []cognitoUserPoolsTermsRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsTermsPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["TermsId"].(string)
		right, _ := out[j]["TermsId"].(string)
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return out
}

func cognitoUserPoolsTermsDetails(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	out := cloneCognitoUserPoolsMapAny(payload)
	delete(out, "UserPoolId")
	delete(out, "TermsId")
	delete(out, "TermsName")
	return out
}
