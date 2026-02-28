package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	acmsvc "github.com/stackyard/stackyard/internal/services/acm"
)

type acmError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleACMJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isACMJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "acm")
	if !ok {
		respondACMError(w, status, code, msg)
		return true
	}

	action := parseACMTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondACMError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := acmOperationByName[action]; !known {
		respondACMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseACMPayload(r)
	if err != nil {
		respondACMError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "RequestCertificate":
		arn, err := s.acm.RequestCertificate(
			acmString(payload["DomainName"]),
			acmStringSlice(payload["SubjectAlternativeNames"]),
			acmString(payload["IdempotencyToken"]),
			strings.ToUpper(acmString(payload["ValidationMethod"])),
			acmOptionsPayload(payload["Options"]),
			acmDomainValidationPayload(payload["DomainValidationOptions"]),
			acmTagsPayload(payload["Tags"]),
		)
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{"CertificateArn": arn})
		return true

	case "ImportCertificate":
		arn, err := s.acm.ImportCertificate(
			acmString(payload["CertificateArn"]),
			acmString(payload["Certificate"]),
			acmString(payload["PrivateKey"]),
			acmString(payload["CertificateChain"]),
			acmTagsPayload(payload["Tags"]),
		)
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{"CertificateArn": arn})
		return true

	case "DescribeCertificate":
		certificate, err := s.acm.DescribeCertificate(acmString(payload["CertificateArn"]))
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{"Certificate": acmCertificateDetailPayload(certificate)})
		return true

	case "ListCertificates":
		maxItems := acmInt32(payload["MaxItems"])
		summaries, nextToken, err := s.acm.ListCertificates(
			acmString(payload["NextToken"]),
			maxItems,
			acmStringSlice(payload["CertificateStatuses"]),
		)
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"CertificateSummaryList": acmCertificateSummaryListPayload(summaries),
		}
		if strings.TrimSpace(nextToken) != "" {
			response["NextToken"] = nextToken
		}
		respondACMJSON(w, http.StatusOK, response)
		return true

	case "DeleteCertificate":
		if err := s.acm.DeleteCertificate(acmString(payload["CertificateArn"])); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetCertificate":
		certificateBody, certificateChain, err := s.acm.GetCertificate(acmString(payload["CertificateArn"]))
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		response := map[string]any{"Certificate": certificateBody}
		if strings.TrimSpace(certificateChain) != "" {
			response["CertificateChain"] = certificateChain
		}
		respondACMJSON(w, http.StatusOK, response)
		return true

	case "ExportCertificate":
		certificateBody, certificateChain, privateKey, err := s.acm.ExportCertificate(
			acmString(payload["CertificateArn"]),
			acmString(payload["Passphrase"]),
		)
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Certificate": certificateBody,
			"PrivateKey":  privateKey,
		}
		if strings.TrimSpace(certificateChain) != "" {
			response["CertificateChain"] = certificateChain
		}
		respondACMJSON(w, http.StatusOK, response)
		return true

	case "RenewCertificate":
		if err := s.acm.RenewCertificate(acmString(payload["CertificateArn"])); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "RevokeCertificate":
		if err := s.acm.RevokeCertificate(
			acmString(payload["CertificateArn"]),
			strings.ToUpper(acmString(payload["RevocationReason"])),
		); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ResendValidationEmail":
		if err := s.acm.ResendValidationEmail(
			acmString(payload["CertificateArn"]),
			acmString(payload["Domain"]),
			acmString(payload["ValidationDomain"]),
		); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UpdateCertificateOptions":
		if err := s.acm.UpdateCertificateOptions(
			acmString(payload["CertificateArn"]),
			acmOptionsPayload(payload["Options"]),
		); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AddTagsToCertificate":
		if err := s.acm.AddTagsToCertificate(
			acmString(payload["CertificateArn"]),
			acmTagsPayload(payload["Tags"]),
		); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "RemoveTagsFromCertificate":
		if err := s.acm.RemoveTagsFromCertificate(
			acmString(payload["CertificateArn"]),
			acmTagKeysPayload(payload["Tags"]),
		); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListTagsForCertificate":
		tags, err := s.acm.ListTagsForCertificate(acmString(payload["CertificateArn"]))
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{
			"Tags": acmTagsListPayload(tags),
		})
		return true

	case "GetAccountConfiguration":
		config, err := s.acm.GetAccountConfiguration()
		if err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{
			"ExpiryEvents": map[string]any{
				"DaysBeforeExpiry": config.ExpiryEvents.DaysBeforeExpiry,
			},
		})
		return true

	case "PutAccountConfiguration":
		configuration := acmAccountConfigurationPayload(payload)
		if err := s.acm.PutAccountConfiguration(configuration); err != nil {
			respondACMErrorForErr(w, err)
			return true
		}
		respondACMJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	respondACMError(w, http.StatusBadRequest, "ValidationException", "unsupported action")
	return true
}

func isACMJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "CertificateManager.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "CertificateManager")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "acm" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".acm.") || strings.HasPrefix(host, "acm.")
}

func parseACMTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "CertificateManager.") {
		return strings.TrimPrefix(target, "CertificateManager.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func respondACMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondACMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondACMJSON(w, status, acmError{Type: code, Message: msg})
}

func respondACMErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, acmsvc.ErrInvalidParameter):
		respondACMError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, acmsvc.ErrNotFound):
		respondACMError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
	case errors.Is(err, acmsvc.ErrInvalidState):
		respondACMError(w, http.StatusBadRequest, "InvalidStateException", err.Error())
	case errors.Is(err, acmsvc.ErrLimitExceeded):
		respondACMError(w, http.StatusBadRequest, "TooManyTagsException", err.Error())
	case errors.Is(err, acmsvc.ErrThrottling):
		respondACMError(w, http.StatusTooManyRequests, "ThrottlingException", err.Error())
	default:
		respondACMError(w, http.StatusBadRequest, "ValidationException", err.Error())
	}
}

func parseACMPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return obj, nil
}

func acmString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func acmStringSlice(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		if text := acmString(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func acmInt32(value any) int32 {
	switch raw := value.(type) {
	case float64:
		return int32(raw)
	case float32:
		return int32(raw)
	case int:
		return int32(raw)
	case int32:
		return raw
	case int64:
		return int32(raw)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return 0
		}
		return int32(parsed)
	default:
		return 0
	}
}

func acmOptionsPayload(value any) acmsvc.CertificateOptions {
	options, ok := value.(map[string]any)
	if !ok {
		return acmsvc.CertificateOptions{}
	}
	return acmsvc.CertificateOptions{
		CertificateTransparencyLoggingPreference: strings.ToUpper(acmString(options["CertificateTransparencyLoggingPreference"])),
	}
}

func acmDomainValidationPayload(value any) []acmsvc.DomainValidation {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]acmsvc.DomainValidation, 0, len(values))
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		domainValidation := acmsvc.DomainValidation{
			DomainName:       acmString(entry["DomainName"]),
			ValidationDomain: acmString(entry["ValidationDomain"]),
			ValidationMethod: strings.ToUpper(acmString(entry["ValidationMethod"])),
			ValidationStatus: strings.ToUpper(acmString(entry["ValidationStatus"])),
		}
		resourceRecord, ok := entry["ResourceRecord"].(map[string]any)
		if ok {
			domainValidation.ResourceRecord = acmsvc.ResourceRecord{
				Name:  acmString(resourceRecord["Name"]),
				Type:  acmString(resourceRecord["Type"]),
				Value: acmString(resourceRecord["Value"]),
			}
		}
		out = append(out, domainValidation)
	}
	return out
}

func acmTagsPayload(value any) map[string]string {
	values, ok := value.([]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := acmString(entry["Key"])
		if key == "" {
			continue
		}
		out[key] = acmString(entry["Value"])
	}
	return out
}

func acmTagKeysPayload(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		switch typed := item.(type) {
		case string:
			if key := strings.TrimSpace(typed); key != "" {
				out = append(out, key)
			}
		case map[string]any:
			if key := acmString(typed["Key"]); key != "" {
				out = append(out, key)
			}
		}
	}
	return out
}

func acmTagsListPayload(tags map[string]string) []map[string]string {
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

func acmAccountConfigurationPayload(payload map[string]any) acmsvc.AccountConfiguration {
	out := acmsvc.AccountConfiguration{}
	expiryEvents, ok := payload["ExpiryEvents"].(map[string]any)
	if !ok {
		return out
	}
	out.ExpiryEvents = acmsvc.ExpiryEventsConfiguration{
		DaysBeforeExpiry: acmInt32(expiryEvents["DaysBeforeExpiry"]),
	}
	return out
}

func acmCertificateSummaryListPayload(items []acmsvc.CertificateSummary) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"CertificateArn":                       item.CertificateArn,
			"DomainName":                           item.DomainName,
			"Status":                               item.Status,
			"Type":                                 item.Type,
			"HasAdditionalSubjectAlternativeNames": item.HasAdditionalSubjectAlternatives,
		})
	}
	return out
}

func acmCertificateDetailPayload(certificate acmsvc.Certificate) map[string]any {
	out := map[string]any{
		"CertificateArn":          certificate.CertificateArn,
		"DomainName":              certificate.DomainName,
		"SubjectAlternativeNames": certificate.SubjectAlternativeNames,
		"Status":                  certificate.Status,
		"Type":                    certificate.Type,
		"CreatedAt":               certificate.CreatedAt,
		"NotBefore":               certificate.NotBefore,
		"NotAfter":                certificate.NotAfter,
		"InUseBy":                 certificate.InUseBy,
		"Options": map[string]any{
			"CertificateTransparencyLoggingPreference": certificate.Options.CertificateTransparencyLoggingPreference,
		},
	}
	if !certificate.IssuedAt.IsZero() {
		out["IssuedAt"] = certificate.IssuedAt
	}
	if certificate.ValidationMethod != "" && certificate.ValidationMethod != "NONE" {
		out["DomainValidationOptions"] = acmDomainValidationListPayload(certificate.DomainValidationOptions)
	}
	if certificate.FailureReason != "" {
		out["FailureReason"] = certificate.FailureReason
	}
	if certificate.RevocationReason != "" {
		out["RevocationReason"] = certificate.RevocationReason
	}
	if certificate.RenewalSummary.RenewalStatus != "" {
		renewal := map[string]any{
			"RenewalStatus": certificate.RenewalSummary.RenewalStatus,
		}
		if !certificate.RenewalSummary.UpdatedAt.IsZero() {
			renewal["UpdatedAt"] = certificate.RenewalSummary.UpdatedAt
		}
		if len(certificate.RenewalSummary.DomainValidationOptions) != 0 {
			renewal["DomainValidationOptions"] = acmDomainValidationListPayload(certificate.RenewalSummary.DomainValidationOptions)
		}
		out["RenewalSummary"] = renewal
	}
	return out
}

func acmDomainValidationListPayload(values []acmsvc.DomainValidation) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		item := map[string]any{
			"DomainName":       value.DomainName,
			"ValidationDomain": value.ValidationDomain,
			"ValidationMethod": value.ValidationMethod,
			"ValidationStatus": value.ValidationStatus,
		}
		if value.ResourceRecord.Name != "" || value.ResourceRecord.Type != "" || value.ResourceRecord.Value != "" {
			item["ResourceRecord"] = map[string]any{
				"Name":  value.ResourceRecord.Name,
				"Type":  value.ResourceRecord.Type,
				"Value": value.ResourceRecord.Value,
			}
		}
		out = append(out, item)
	}
	return out
}
