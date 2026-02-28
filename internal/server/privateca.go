package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	privatecasvc "github.com/stackyard/stackyard/internal/services/privateca"
)

type privateCAError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePrivateCAJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPrivateCAJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "acm-pca")
	if !ok {
		respondPrivateCAError(w, status, code, msg)
		return true
	}

	action := parsePrivateCATarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := privateCAOperationByName[action]; !known {
		respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parsePrivateCAPayload(r)
	if err != nil {
		respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}
	if err := s.privateca.RecordAPICall(); err != nil {
		respondPrivateCAErrorForErr(w, err)
		return true
	}

	switch action {
	case "CreateCertificateAuthority":
		arn, err := s.privateca.CreateCertificateAuthority(privatecasvc.CreateCertificateAuthorityInput{
			Configuration:              privateCAConfigurationPayload(payload["CertificateAuthorityConfiguration"]),
			RevocationConfiguration:    privateCARevocationConfigurationPayload(payload["RevocationConfiguration"]),
			CertificateAuthorityType:   privateCAString(payload["CertificateAuthorityType"]),
			IdempotencyToken:           privateCAString(payload["IdempotencyToken"]),
			KeyStorageSecurityStandard: privateCAString(payload["KeyStorageSecurityStandard"]),
			UsageMode:                  privateCAString(payload["UsageMode"]),
			Tags:                       privateCATagsPayload(payload["Tags"]),
		})
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{"CertificateAuthorityArn": arn})
		return true

	case "DescribeCertificateAuthority":
		ca, err := s.privateca.DescribeCertificateAuthority(privateCAString(payload["CertificateAuthorityArn"]))
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{"CertificateAuthority": privateCADetailPayload(ca)})
		return true

	case "ListCertificateAuthorities":
		maxResults, ok := privateCAInt32(payload["MaxResults"])
		if !ok {
			respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		output, err := s.privateca.ListCertificateAuthorities(privatecasvc.ListCertificateAuthoritiesInput{
			NextToken:     privateCAString(payload["NextToken"]),
			MaxResults:    maxResults,
			ResourceOwner: privateCAString(payload["ResourceOwner"]),
		})
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"CertificateAuthorities": privateCASummaryListPayload(output.CertificateAuthorities),
		}
		if strings.TrimSpace(output.NextToken) != "" {
			response["NextToken"] = output.NextToken
		}
		respondPrivateCAJSON(w, http.StatusOK, response)
		return true

	case "UpdateCertificateAuthority":
		revocationConfig := privateCARevocationConfigurationPayload(payload["RevocationConfiguration"])
		input := privatecasvc.UpdateCertificateAuthorityInput{
			ARN:    privateCAString(payload["CertificateAuthorityArn"]),
			Status: privateCAString(payload["Status"]),
		}
		if privateCAHasMap(payload["RevocationConfiguration"]) {
			input.RevocationConfiguration = &revocationConfig
		}
		if err := s.privateca.UpdateCertificateAuthority(input); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "RestoreCertificateAuthority":
		if err := s.privateca.RestoreCertificateAuthority(privateCAString(payload["CertificateAuthorityArn"])); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "DeleteCertificateAuthority":
		permanentDeletionTimeInDays, ok := privateCAInt32(payload["PermanentDeletionTimeInDays"])
		if !ok {
			respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "invalid PermanentDeletionTimeInDays")
			return true
		}
		if err := s.privateca.DeleteCertificateAuthority(
			privateCAString(payload["CertificateAuthorityArn"]),
			permanentDeletionTimeInDays,
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "IssueCertificate":
		certARN, err := s.privateca.IssueCertificate(privatecasvc.IssueCertificateInput{
			CertificateAuthorityARN: privateCAString(payload["CertificateAuthorityArn"]),
			Csr:                     privateCAString(payload["Csr"]),
			SigningAlgorithm:        privateCAString(payload["SigningAlgorithm"]),
			TemplateARN:             privateCAString(payload["TemplateArn"]),
			Validity:                privateCAValidityPayload(payload["Validity"]),
			IdempotencyToken:        privateCAString(payload["IdempotencyToken"]),
		})
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{"CertificateArn": certARN})
		return true

	case "GetCertificate":
		certificate, certificateChain, err := s.privateca.GetCertificate(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCAString(payload["CertificateArn"]),
		)
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		response := map[string]any{"Certificate": certificate}
		if strings.TrimSpace(certificateChain) != "" {
			response["CertificateChain"] = certificateChain
		}
		respondPrivateCAJSON(w, http.StatusOK, response)
		return true

	case "GetCertificateAuthorityCsr":
		csr, err := s.privateca.GetCertificateAuthorityCSR(privateCAString(payload["CertificateAuthorityArn"]))
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{"Csr": csr})
		return true

	case "ImportCertificateAuthorityCertificate":
		if err := s.privateca.ImportCertificateAuthorityCertificate(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCAString(payload["Certificate"]),
			privateCAString(payload["CertificateChain"]),
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetCertificateAuthorityCertificate":
		certificate, certificateChain, err := s.privateca.GetCertificateAuthorityCertificate(privateCAString(payload["CertificateAuthorityArn"]))
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		response := map[string]any{"Certificate": certificate}
		if strings.TrimSpace(certificateChain) != "" {
			response["CertificateChain"] = certificateChain
		}
		respondPrivateCAJSON(w, http.StatusOK, response)
		return true

	case "RevokeCertificate":
		if err := s.privateca.RevokeCertificate(privatecasvc.RevokeCertificateInput{
			CertificateAuthorityARN: privateCAString(payload["CertificateAuthorityArn"]),
			CertificateSerial:       privateCAString(payload["CertificateSerial"]),
			RevocationReason:        privateCAString(payload["RevocationReason"]),
		}); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreatePermission":
		if err := s.privateca.CreatePermission(privatecasvc.CreatePermissionInput{
			CertificateAuthorityARN: privateCAString(payload["CertificateAuthorityArn"]),
			Principal:               privateCAString(payload["Principal"]),
			SourceAccount:           privateCAString(payload["SourceAccount"]),
			Actions:                 privateCAStringSlice(payload["Actions"]),
			Policy:                  privateCAString(payload["Policy"]),
		}); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListPermissions":
		maxResults, ok := privateCAInt32(payload["MaxResults"])
		if !ok {
			respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		output, err := s.privateca.ListPermissions(privatecasvc.ListPermissionsInput{
			CertificateAuthorityARN: privateCAString(payload["CertificateAuthorityArn"]),
			NextToken:               privateCAString(payload["NextToken"]),
			MaxResults:              maxResults,
		})
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		response := map[string]any{"Permissions": privateCAPermissionsPayload(output.Permissions)}
		if strings.TrimSpace(output.NextToken) != "" {
			response["NextToken"] = output.NextToken
		}
		respondPrivateCAJSON(w, http.StatusOK, response)
		return true

	case "DeletePermission":
		if err := s.privateca.DeletePermission(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCAString(payload["Principal"]),
			privateCAString(payload["SourceAccount"]),
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "PutPolicy":
		if err := s.privateca.PutPolicy(
			privateCAFirstString(payload, "CertificateAuthorityArn", "ResourceArn"),
			privateCAString(payload["Policy"]),
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetPolicy":
		policy, err := s.privateca.GetPolicy(privateCAFirstString(payload, "CertificateAuthorityArn", "ResourceArn"))
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{"Policy": policy})
		return true

	case "DeletePolicy":
		if err := s.privateca.DeletePolicy(privateCAFirstString(payload, "CertificateAuthorityArn", "ResourceArn")); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateCertificateAuthorityAuditReport":
		auditReportID, s3Key, err := s.privateca.CreateCertificateAuthorityAuditReport(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCAString(payload["S3BucketName"]),
			privateCAString(payload["AuditReportResponseFormat"]),
		)
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{
			"AuditReportId": auditReportID,
			"S3Key":         s3Key,
		})
		return true

	case "DescribeCertificateAuthorityAuditReport":
		report, err := s.privateca.DescribeCertificateAuthorityAuditReport(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCAString(payload["AuditReportId"]),
		)
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{
			"AuditReportStatus": report.Status,
			"S3BucketName":      report.S3BucketName,
			"S3Key":             report.S3Key,
			"CreatedAt":         report.CreatedAt,
		})
		return true

	case "ListTags":
		maxResults, ok := privateCAInt32(payload["MaxResults"])
		if !ok {
			respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		output, err := s.privateca.ListTags(privatecasvc.ListTagsInput{
			CertificateAuthorityARN: privateCAString(payload["CertificateAuthorityArn"]),
			NextToken:               privateCAString(payload["NextToken"]),
			MaxResults:              maxResults,
		})
		if err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"Tags": privateCATagsListPayload(output.Tags),
		}
		if strings.TrimSpace(output.NextToken) != "" {
			response["NextToken"] = output.NextToken
		}
		respondPrivateCAJSON(w, http.StatusOK, response)
		return true

	case "TagCertificateAuthority":
		if err := s.privateca.TagCertificateAuthority(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCATagsPayload(payload["Tags"]),
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagCertificateAuthority":
		if err := s.privateca.UntagCertificateAuthority(
			privateCAString(payload["CertificateAuthorityArn"]),
			privateCATagKeysPayload(payload["Tags"]),
		); err != nil {
			respondPrivateCAErrorForErr(w, err)
			return true
		}
		respondPrivateCAJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	respondPrivateCAError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isPrivateCAJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "ACMPrivateCA.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "ACMPrivateCA")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "acm-pca" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".acm-pca.") || strings.HasPrefix(host, "acm-pca.")
}

func parsePrivateCATarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ACMPrivateCA.") {
		return strings.TrimPrefix(target, "ACMPrivateCA.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parsePrivateCAPayload(r *http.Request) (map[string]any, error) {
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

func respondPrivateCAJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPrivateCAError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPrivateCAJSON(w, status, privateCAError{Type: code, Message: msg})
}

func respondPrivateCAErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, privatecasvc.ErrInvalidParameter):
		respondPrivateCAError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, privatecasvc.ErrNotFound):
		respondPrivateCAError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
	case errors.Is(err, privatecasvc.ErrInvalidState):
		respondPrivateCAError(w, http.StatusBadRequest, "InvalidStateException", err.Error())
	case errors.Is(err, privatecasvc.ErrThrottling):
		respondPrivateCAError(w, http.StatusTooManyRequests, "ThrottlingException", err.Error())
	default:
		respondPrivateCAError(w, http.StatusInternalServerError, "InternalFailure", err.Error())
	}
}

func privateCAHasMap(value any) bool {
	_, ok := value.(map[string]any)
	return ok
}

func privateCAString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func privateCAFirstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := privateCAString(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func privateCAStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := privateCAString(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func privateCAInt32(value any) (int32, bool) {
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
		if strings.TrimSpace(v) == "" {
			return 0, true
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	default:
		return 0, false
	}
}

func privateCAInt64(value any) (int64, bool) {
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
		if strings.TrimSpace(v) == "" {
			return 0, true
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func privateCAMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if out, ok := value.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func privateCATagsPayload(value any) map[string]string {
	out := map[string]string{}
	items, ok := value.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		tag := privateCAMap(item)
		key := privateCAString(tag["Key"])
		if key == "" {
			continue
		}
		out[key] = privateCAString(tag["Value"])
	}
	return out
}

func privateCATagKeysPayload(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			key := strings.TrimSpace(typed)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, key)
		case map[string]any:
			key := privateCAString(typed["Key"])
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, key)
		}
	}
	return out
}

func privateCAConfigurationPayload(value any) privatecasvc.CertificateAuthorityConfiguration {
	raw := privateCAMap(value)
	subjectRaw := privateCAMap(raw["Subject"])
	return privatecasvc.CertificateAuthorityConfiguration{
		KeyAlgorithm:     privateCAString(raw["KeyAlgorithm"]),
		SigningAlgorithm: privateCAString(raw["SigningAlgorithm"]),
		Subject: privatecasvc.Subject{
			Country:                    privateCAString(subjectRaw["Country"]),
			Organization:               privateCAString(subjectRaw["Organization"]),
			OrganizationalUnit:         privateCAString(subjectRaw["OrganizationalUnit"]),
			DistinguishedNameQualifier: privateCAString(subjectRaw["DistinguishedNameQualifier"]),
			State:                      privateCAString(subjectRaw["State"]),
			CommonName:                 privateCAString(subjectRaw["CommonName"]),
			SerialNumber:               privateCAString(subjectRaw["SerialNumber"]),
			Locality:                   privateCAString(subjectRaw["Locality"]),
			Title:                      privateCAString(subjectRaw["Title"]),
			Surname:                    privateCAString(subjectRaw["Surname"]),
			GivenName:                  privateCAString(subjectRaw["GivenName"]),
			Initials:                   privateCAString(subjectRaw["Initials"]),
			Pseudonym:                  privateCAString(subjectRaw["Pseudonym"]),
			GenerationQualifier:        privateCAString(subjectRaw["GenerationQualifier"]),
		},
	}
}

func privateCARevocationConfigurationPayload(value any) privatecasvc.RevocationConfiguration {
	raw := privateCAMap(value)
	crlRaw := privateCAMap(raw["CrlConfiguration"])
	expirationInDays, _ := privateCAInt32(crlRaw["ExpirationInDays"])
	return privatecasvc.RevocationConfiguration{
		CrlConfiguration: privatecasvc.CrlConfiguration{
			Enabled:          privateCABool(crlRaw["Enabled"]),
			ExpirationInDays: expirationInDays,
			CustomCNAME:      privateCAString(crlRaw["CustomCname"]),
			S3BucketName:     privateCAString(crlRaw["S3BucketName"]),
			S3ObjectACL:      privateCAString(crlRaw["S3ObjectAcl"]),
		},
	}
}

func privateCAValidityPayload(value any) privatecasvc.Validity {
	raw := privateCAMap(value)
	validityValue, _ := privateCAInt64(raw["Value"])
	return privatecasvc.Validity{
		Value: validityValue,
		Type:  privateCAString(raw["Type"]),
	}
}

func privateCABool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func privateCADetailPayload(ca privatecasvc.CertificateAuthority) map[string]any {
	payload := map[string]any{
		"Arn":               ca.ARN,
		"OwnerAccount":      ca.OwnerAccount,
		"CreatedAt":         ca.CreatedAt,
		"LastStateChangeAt": ca.LastStateChangeAt,
		"Type":              ca.Type,
		"Serial":            ca.Serial,
		"Status":            ca.Status,
		"NotBefore":         ca.NotBefore,
		"NotAfter":          ca.NotAfter,
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm":     ca.Configuration.KeyAlgorithm,
			"SigningAlgorithm": ca.Configuration.SigningAlgorithm,
			"Subject": map[string]any{
				"Country":                    ca.Configuration.Subject.Country,
				"Organization":               ca.Configuration.Subject.Organization,
				"OrganizationalUnit":         ca.Configuration.Subject.OrganizationalUnit,
				"DistinguishedNameQualifier": ca.Configuration.Subject.DistinguishedNameQualifier,
				"State":                      ca.Configuration.Subject.State,
				"CommonName":                 ca.Configuration.Subject.CommonName,
				"SerialNumber":               ca.Configuration.Subject.SerialNumber,
				"Locality":                   ca.Configuration.Subject.Locality,
				"Title":                      ca.Configuration.Subject.Title,
				"Surname":                    ca.Configuration.Subject.Surname,
				"GivenName":                  ca.Configuration.Subject.GivenName,
				"Initials":                   ca.Configuration.Subject.Initials,
				"Pseudonym":                  ca.Configuration.Subject.Pseudonym,
				"GenerationQualifier":        ca.Configuration.Subject.GenerationQualifier,
			},
		},
		"RevocationConfiguration": map[string]any{
			"CrlConfiguration": map[string]any{
				"Enabled":          ca.RevocationConfiguration.CrlConfiguration.Enabled,
				"ExpirationInDays": ca.RevocationConfiguration.CrlConfiguration.ExpirationInDays,
				"CustomCname":      ca.RevocationConfiguration.CrlConfiguration.CustomCNAME,
				"S3BucketName":     ca.RevocationConfiguration.CrlConfiguration.S3BucketName,
				"S3ObjectAcl":      ca.RevocationConfiguration.CrlConfiguration.S3ObjectACL,
			},
		},
		"KeyStorageSecurityStandard": ca.KeyStorageSecurityStandard,
		"UsageMode":                  ca.UsageMode,
	}
	if ca.RestorableUntil != nil {
		payload["RestorableUntil"] = *ca.RestorableUntil
	}
	return payload
}

func privateCASummaryListPayload(items []privatecasvc.CertificateAuthority) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, ca := range items {
		summary := map[string]any{
			"Arn":               ca.ARN,
			"CreatedAt":         ca.CreatedAt,
			"LastStateChangeAt": ca.LastStateChangeAt,
			"Type":              ca.Type,
			"Serial":            ca.Serial,
			"Status":            ca.Status,
			"NotBefore":         ca.NotBefore,
			"NotAfter":          ca.NotAfter,
			"OwnerAccount":      ca.OwnerAccount,
		}
		if ca.RestorableUntil != nil {
			summary["RestorableUntil"] = *ca.RestorableUntil
		}
		out = append(out, summary)
	}
	return out
}

func privateCAPermissionsPayload(items []privatecasvc.Permission) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, permission := range items {
		entry := map[string]any{
			"CertificateAuthorityArn": permission.CertificateAuthorityARN,
			"CreatedAt":               permission.CreatedAt,
			"Principal":               permission.Principal,
			"SourceAccount":           permission.SourceAccount,
			"Actions":                 permission.Actions,
		}
		if strings.TrimSpace(permission.Policy) != "" {
			entry["Policy"] = permission.Policy
		}
		out = append(out, entry)
	}
	return out
}

func privateCATagsListPayload(tags map[string]string) []map[string]any {
	if len(tags) == 0 {
		return []map[string]any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{
			"Key":   key,
			"Value": tags[key],
		})
	}
	return out
}
