package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpSecurityPrivateCAReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSecurityPrivateCARouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_security_privateca(w, r) {
		return true
	}

	path := normalizeGCPSecurityPrivateCAPath(rawRequestPath(r))
	if isGCPSecurityPrivateCALocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecurityPrivateCAListLocations(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecurityPrivateCAPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecurityPrivateCAListCaPools(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetCaPool(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAFetchCaCerts(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAListCertificateAuthorities(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetCertificateAuthority(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAFetchCertificateAuthorityCSR(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAListCertificates(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetCertificate(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAListCertificateRevocationLists(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetCertificateRevocationList(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAListCertificateTemplates(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetCertificateTemplate(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAListOperations(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecurityPrivateCACreateCaPool(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAFetchCaCerts(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCACreateCertificateAuthority(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCACertificateAuthorityAction(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCACertificateAuthorityActionLoose(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCAFetchCertificateAuthorityCSR(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCACreateCertificate(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCARevokeCertificate(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCACreateCertificateTemplate(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCASetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCATestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCACancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecurityPrivateCAUpdateCaPool(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAUpdateCertificateAuthority(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAUpdateCertificate(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAUpdateCertificateRevocationList(w, r, path) {
			return true
		}
		if handleGCPSecurityPrivateCAUpdateCertificateTemplate(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecurityPrivateCADeleteCaPool(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCADeleteCertificateAuthority(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCADeleteCertificateTemplate(w, path) {
			return true
		}
		if handleGCPSecurityPrivateCADeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecurityPrivateCAPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecurityPrivateCAHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "security_privateca", "security-privateca", "security-privateca-apiv1", "security_privateca_apiv1", "privateca", "private-ca", "private_ca", "certificateauthority", "certificate-authority":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-security-privateca-apiv1") || strings.Contains(ua, "cloud.google.com/go/security/privateca")
}

func isGCPSecurityPrivateCALocationRequest(r *http.Request, path string) bool {
	if !hasGCPSecurityPrivateCAHint(r) {
		return false
	}
	_, _, _, ok := parseGCPSecurityPrivateCAProjectLocationPath(path)
	return ok
}

func isGCPSecurityPrivateCAPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.security.privateca.v1.CertificateAuthorityService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.longrunning.Operations/") {
		return true
	}
	if _, _, _, ok := parseGCPSecurityPrivateCACaPoolsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSecurityPrivateCACaPoolPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCACaPoolActionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCACertificateAuthoritiesCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCACertificateAuthorityPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPSecurityPrivateCACertificateAuthorityActionPath(path); ok {
		return true
	}
	if strings.Contains(path, "/caPools/") && strings.Contains(path, "/certificateAuthorities/") &&
		(strings.Contains(path, ":activate") || strings.Contains(path, ":enable") || strings.Contains(path, ":disable") || strings.Contains(path, ":undelete") || strings.Contains(path, ":fetch")) {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCACertificatesCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCACertificatePath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPSecurityPrivateCACertificateActionPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPSecurityPrivateCACertificateRevocationListsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPSecurityPrivateCACertificateRevocationListPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSecurityPrivateCACertificateTemplatesCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSecurityPrivateCACertificateTemplatePath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSecurityPrivateCAOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecurityPrivateCAOperationPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPSecurityPrivateCAIAMActionPath(path); ok {
		return true
	}
	return false
}

func handleGCPSecurityPrivateCAListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPSecurityPrivateCAProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCALocation(project, "us-central1"),
		gcpSecurityPrivateCALocation(project, "global"),
	}
	return respondGCPSecurityPrivateCAList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPSecurityPrivateCAProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCALocation(project, location))
	return true
}

func handleGCPSecurityPrivateCAListCaPools(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, ok := parseGCPSecurityPrivateCACaPoolsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCACaPool(project, location, "pool-1"),
		gcpSecurityPrivateCACaPool(project, location, "pool-2"),
	}
	return respondGCPSecurityPrivateCAList(w, "caPools", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetCaPool(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, ok := parseGCPSecurityPrivateCACaPoolPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACaPool(project, location, caPoolID))
	return true
}

func handleGCPSecurityPrivateCACreateCaPool(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, ok := parseGCPSecurityPrivateCACaPoolsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	caPool := gcpSecurityPrivateCABodyMap(body, "caPool")
	if len(caPool) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPrivateCAString(caPool, "tier")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool.tier is required")
		return true
	}
	caPoolID := strings.TrimSpace(r.URL.Query().Get("caPoolId"))
	if caPoolID == "" {
		if name := strings.TrimSpace(gcpSecurityPrivateCAString(caPool, "name")); name != "" {
			bodyProject, bodyLocation, bodyCaPoolID, nameOK := parseGCPSecurityPrivateCACaPoolResourceName(name)
			if !nameOK {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool.name is invalid")
				return true
			}
			if bodyProject != project || bodyLocation != location {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool.name must match parent")
				return true
			}
			caPoolID = bodyCaPoolID
		}
	}
	if caPoolID == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPoolId is required")
		return true
	}
	if !isGCPSecurityPrivateCAID(caPoolID) {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPoolId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "create-ca-pool-"+caPoolID, "create", gcpSecurityPrivateCACaPoolName(project, location, caPoolID)))
	return true
}

func handleGCPSecurityPrivateCAUpdateCaPool(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, ok := parseGCPSecurityPrivateCACaPoolPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	caPool := gcpSecurityPrivateCABodyMap(body, "caPool")
	if len(caPool) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool is required")
		return true
	}
	expectedName := gcpSecurityPrivateCACaPoolName(project, location, caPoolID)
	if name := strings.TrimSpace(gcpSecurityPrivateCAString(caPool, "name")); name == "" || name != expectedName {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "caPool.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "update-ca-pool-"+caPoolID, "update", expectedName))
	return true
}

func handleGCPSecurityPrivateCADeleteCaPool(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, ok := parseGCPSecurityPrivateCACaPoolPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "delete-ca-pool-"+caPoolID, "delete", gcpSecurityPrivateCACaPoolName(project, location, caPoolID)))
	return true
}

func handleGCPSecurityPrivateCAFetchCaCerts(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, action, ok := parseGCPSecurityPrivateCACaPoolActionPath(path)
	if !ok || action != "fetchCaCerts" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"caCerts": []any{
			map[string]any{
				"certificates": []any{
					"-----BEGIN CERTIFICATE-----\nSTACKYARD-CA-ROOT\n-----END CERTIFICATE-----",
				},
				"crlAccessUrls": []any{
					fmt.Sprintf("https://privateca.stackyard.local/%s/%s/%s/crl.pem", project, location, caPoolID),
				},
			},
		},
	})
	return true
}

func handleGCPSecurityPrivateCAListCertificateAuthorities(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, _, ok := parseGCPSecurityPrivateCACertificateAuthoritiesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCACertificateAuthority(project, location, caPoolID, "ca-1"),
		gcpSecurityPrivateCACertificateAuthority(project, location, caPoolID, "ca-disabled"),
		gcpSecurityPrivateCACertificateAuthority(project, location, caPoolID, "ca-awaiting"),
	}
	return respondGCPSecurityPrivateCAList(w, "certificateAuthorities", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetCertificateAuthority(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, ok := parseGCPSecurityPrivateCACertificateAuthorityPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificateAuthority(project, location, caPoolID, caID))
	return true
}

func handleGCPSecurityPrivateCACreateCertificateAuthority(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, _, ok := parseGCPSecurityPrivateCACertificateAuthoritiesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	ca := gcpSecurityPrivateCABodyMap(body, "certificateAuthority")
	if len(ca) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPrivateCAString(ca, "type")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority.type is required")
		return true
	}
	if _, ok := ca["config"]; !ok {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority.config is required")
		return true
	}
	caID := strings.TrimSpace(r.URL.Query().Get("certificateAuthorityId"))
	if caID == "" {
		if name := strings.TrimSpace(gcpSecurityPrivateCAString(ca, "name")); name != "" {
			bodyProject, bodyLocation, bodyCaPoolID, bodyCAID, nameOK := parseGCPSecurityPrivateCACertificateAuthorityResourceName(name)
			if !nameOK {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority.name is invalid")
				return true
			}
			if bodyProject != project || bodyLocation != location || bodyCaPoolID != caPoolID {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority.name must match parent")
				return true
			}
			caID = bodyCAID
		}
	}
	if caID == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthorityId is required")
		return true
	}
	if !isGCPSecurityPrivateCAID(caID) {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthorityId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "create-certificate-authority-"+caID, "create", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
	return true
}

func handleGCPSecurityPrivateCAUpdateCertificateAuthority(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, caID, ok := parseGCPSecurityPrivateCACertificateAuthorityPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	ca := gcpSecurityPrivateCABodyMap(body, "certificateAuthority")
	if len(ca) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority is required")
		return true
	}
	expectedName := gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)
	if name := strings.TrimSpace(gcpSecurityPrivateCAString(ca, "name")); name == "" || name != expectedName {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateAuthority.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "update-certificate-authority-"+caID, "update", expectedName))
	return true
}

func handleGCPSecurityPrivateCADeleteCertificateAuthority(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, ok := parseGCPSecurityPrivateCACertificateAuthorityPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "delete-certificate-authority-"+caID, "delete", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
	return true
}

func handleGCPSecurityPrivateCACertificateAuthorityAction(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, action, ok := parseGCPSecurityPrivateCACertificateAuthorityActionPath(path)
	if !ok {
		return false
	}
	state := gcpSecurityPrivateCACertificateAuthorityState(caID)
	switch action {
	case "activate":
		if state != "AWAITING_USER_ACTIVATION" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be in AWAITING_USER_ACTIVATION state")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "activate-certificate-authority-"+caID, "activate", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "enable":
		if state != "DISABLED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be DISABLED to enable")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "enable-certificate-authority-"+caID, "enable", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "disable":
		if state != "ENABLED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be ENABLED to disable")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "disable-certificate-authority-"+caID, "disable", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "undelete":
		if state != "DELETED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be DELETED to undelete")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "undelete-certificate-authority-"+caID, "undelete", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	default:
		return false
	}
}

func handleGCPSecurityPrivateCACertificateAuthorityActionLoose(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, action, ok := parseGCPSecurityPrivateCACertificateAuthorityActionPathLoose(path)
	if !ok {
		return false
	}
	state := gcpSecurityPrivateCACertificateAuthorityState(caID)
	switch action {
	case "activate":
		if state != "AWAITING_USER_ACTIVATION" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be in AWAITING_USER_ACTIVATION state")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "activate-certificate-authority-"+caID, "activate", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "enable":
		if state != "DISABLED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be DISABLED to enable")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "enable-certificate-authority-"+caID, "enable", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "disable":
		if state != "ENABLED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be ENABLED to disable")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "disable-certificate-authority-"+caID, "disable", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	case "undelete":
		if state != "DELETED" {
			respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate authority must be DELETED to undelete")
			return true
		}
		respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "undelete-certificate-authority-"+caID, "undelete", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID)))
		return true
	default:
		return false
	}
}

func handleGCPSecurityPrivateCAFetchCertificateAuthorityCSR(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, action, ok := parseGCPSecurityPrivateCACertificateAuthorityActionPath(path)
	if !ok || action != "fetch" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":   gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID),
		"pemCsr": "-----BEGIN CERTIFICATE REQUEST-----\nSTACKYARD-CSR\n-----END CERTIFICATE REQUEST-----",
	})
	return true
}

func handleGCPSecurityPrivateCAListCertificates(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, _, ok := parseGCPSecurityPrivateCACertificatesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCACertificate(project, location, caPoolID, "cert-1"),
		gcpSecurityPrivateCACertificate(project, location, caPoolID, "cert-revoked"),
	}
	return respondGCPSecurityPrivateCAList(w, "certificates", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetCertificate(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, certificateID, ok := parseGCPSecurityPrivateCACertificatePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificate(project, location, caPoolID, certificateID))
	return true
}

func handleGCPSecurityPrivateCACreateCertificate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, _, ok := parseGCPSecurityPrivateCACertificatesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	certificate := gcpSecurityPrivateCABodyMap(body, "certificate")
	if len(certificate) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPrivateCAString(certificate, "lifetime")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate.lifetime is required")
		return true
	}
	certificateID := strings.TrimSpace(r.URL.Query().Get("certificateId"))
	if certificateID == "" {
		if name := strings.TrimSpace(gcpSecurityPrivateCAString(certificate, "name")); name != "" {
			bodyProject, bodyLocation, bodyCaPoolID, bodyCertificateID, nameOK := parseGCPSecurityPrivateCACertificateResourceName(name)
			if !nameOK {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate.name is invalid")
				return true
			}
			if bodyProject != project || bodyLocation != location || bodyCaPoolID != caPoolID {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate.name must match parent")
				return true
			}
			certificateID = bodyCertificateID
		}
	}
	if certificateID == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateId is required")
		return true
	}
	if !isGCPSecurityPrivateCAID(certificateID) {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificate(project, location, caPoolID, certificateID))
	return true
}

func handleGCPSecurityPrivateCAUpdateCertificate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, certificateID, ok := parseGCPSecurityPrivateCACertificatePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	certificate := gcpSecurityPrivateCABodyMap(body, "certificate")
	if len(certificate) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate is required")
		return true
	}
	expectedName := gcpSecurityPrivateCACertificateName(project, location, caPoolID, certificateID)
	if name := strings.TrimSpace(gcpSecurityPrivateCAString(certificate, "name")); name == "" || name != expectedName {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificate.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificate(project, location, caPoolID, certificateID))
	return true
}

func handleGCPSecurityPrivateCARevokeCertificate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, certificateID, action, ok := parseGCPSecurityPrivateCACertificateActionPath(path)
	if !ok || action != "revoke" {
		return false
	}
	if gcpSecurityPrivateCACertificateState(certificateID) == "REVOKED" {
		respondGCPSecurityPrivateCAFailedPrecondition(w, path, "certificate is already revoked")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	reason := strings.TrimSpace(gcpSecurityPrivateCAString(body, "reason"))
	if reason == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "reason is required")
		return true
	}
	revoked := gcpSecurityPrivateCACertificate(project, location, caPoolID, certificateID)
	revoked["revocationDetails"] = map[string]any{
		"revocationState": "KEY_COMPROMISE",
		"revocationTime":  gcpSecurityPrivateCAReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
	}
	respondJSON(w, http.StatusOK, revoked)
	return true
}

func handleGCPSecurityPrivateCAListCertificateRevocationLists(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, caID, _, ok := parseGCPSecurityPrivateCACertificateRevocationListsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCACertificateRevocationList(project, location, caPoolID, caID, "crl-1"),
	}
	return respondGCPSecurityPrivateCAList(w, "certificateRevocationLists", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetCertificateRevocationList(w http.ResponseWriter, path string) bool {
	project, location, caPoolID, caID, crlID, ok := parseGCPSecurityPrivateCACertificateRevocationListPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificateRevocationList(project, location, caPoolID, caID, crlID))
	return true
}

func handleGCPSecurityPrivateCAUpdateCertificateRevocationList(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, caPoolID, caID, crlID, ok := parseGCPSecurityPrivateCACertificateRevocationListPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	crl := gcpSecurityPrivateCABodyMap(body, "certificateRevocationList")
	if len(crl) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateRevocationList is required")
		return true
	}
	expectedName := gcpSecurityPrivateCACertificateRevocationListName(project, location, caPoolID, caID, crlID)
	if name := strings.TrimSpace(gcpSecurityPrivateCAString(crl, "name")); name == "" || name != expectedName {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateRevocationList.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "update-certificate-revocation-list-"+crlID, "update", expectedName))
	return true
}

func handleGCPSecurityPrivateCAListCertificateTemplates(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, ok := parseGCPSecurityPrivateCACertificateTemplatesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCACertificateTemplate(project, location, "template-1"),
		gcpSecurityPrivateCACertificateTemplate(project, location, "template-2"),
	}
	return respondGCPSecurityPrivateCAList(w, "certificateTemplates", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetCertificateTemplate(w http.ResponseWriter, path string) bool {
	project, location, templateID, ok := parseGCPSecurityPrivateCACertificateTemplatePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCACertificateTemplate(project, location, templateID))
	return true
}

func handleGCPSecurityPrivateCACreateCertificateTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, ok := parseGCPSecurityPrivateCACertificateTemplatesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	template := gcpSecurityPrivateCABodyMap(body, "certificateTemplate")
	if len(template) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplate is required")
		return true
	}
	templateID := strings.TrimSpace(r.URL.Query().Get("certificateTemplateId"))
	if templateID == "" {
		if name := strings.TrimSpace(gcpSecurityPrivateCAString(template, "name")); name != "" {
			bodyProject, bodyLocation, bodyTemplateID, nameOK := parseGCPSecurityPrivateCACertificateTemplateResourceName(name)
			if !nameOK {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplate.name is invalid")
				return true
			}
			if bodyProject != project || bodyLocation != location {
				respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplate.name must match parent")
				return true
			}
			templateID = bodyTemplateID
		}
	}
	if templateID == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplateId is required")
		return true
	}
	if !isGCPSecurityPrivateCAID(templateID) {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplateId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "create-certificate-template-"+templateID, "create", gcpSecurityPrivateCACertificateTemplateName(project, location, templateID)))
	return true
}

func handleGCPSecurityPrivateCAUpdateCertificateTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, templateID, ok := parseGCPSecurityPrivateCACertificateTemplatePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	template := gcpSecurityPrivateCABodyMap(body, "certificateTemplate")
	if len(template) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplate is required")
		return true
	}
	expectedName := gcpSecurityPrivateCACertificateTemplateName(project, location, templateID)
	if name := strings.TrimSpace(gcpSecurityPrivateCAString(template, "name")); name == "" || name != expectedName {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "certificateTemplate.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "update-certificate-template-"+templateID, "update", expectedName))
	return true
}

func handleGCPSecurityPrivateCADeleteCertificateTemplate(w http.ResponseWriter, path string) bool {
	project, location, templateID, ok := parseGCPSecurityPrivateCACertificateTemplatePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, "delete-certificate-template-"+templateID, "delete", gcpSecurityPrivateCACertificateTemplateName(project, location, templateID)))
	return true
}

func handleGCPSecurityPrivateCAGetIAMPolicy(w http.ResponseWriter, path string) bool {
	resource, action, ok := parseGCPSecurityPrivateCAIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAPolicy(resource, nil))
	return true
}

func handleGCPSecurityPrivateCASetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPSecurityPrivateCAIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpSecurityPrivateCABodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAPolicy(resource, policy))
	return true
}

func handleGCPSecurityPrivateCATestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, action, ok := parseGCPSecurityPrivateCAIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, valid := decodeGCPSecurityPrivateCAJSONBody(w, r, path)
	if !valid {
		return true
	}
	rawPermissions, _ := body["permissions"].([]any)
	if len(rawPermissions) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "permissions is required")
		return true
	}
	permissions := make([]string, 0, len(rawPermissions))
	for _, entry := range rawPermissions {
		if v, ok := entry.(string); ok && strings.TrimSpace(v) != "" {
			permissions = append(permissions, strings.TrimSpace(v))
		}
	}
	if len(permissions) == 0 {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "permissions is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
	return true
}

func handleGCPSecurityPrivateCAListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, ok := parseGCPSecurityPrivateCAOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPrivateCAPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPrivateCAOperation(project, location, "operation-1", "create", gcpSecurityPrivateCACaPoolName(project, location, "pool-1")),
		gcpSecurityPrivateCAOperation(project, location, "operation-2", "update", gcpSecurityPrivateCACertificateTemplateName(project, location, "template-1")),
	}
	return respondGCPSecurityPrivateCAList(w, "operations", items, pageSize, start, path)
}

func handleGCPSecurityPrivateCAGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, action, ok := parseGCPSecurityPrivateCAOperationPath(path)
	if !ok || action != "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPrivateCAOperation(project, location, operationID, "get", ""))
	return true
}

func handleGCPSecurityPrivateCACancelOperation(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPSecurityPrivateCAOperationPath(path)
	if !ok || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityPrivateCADeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPSecurityPrivateCAOperationPath(path)
	if !ok || action != "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPSecurityPrivateCAProjectLocationPath(path string) (project, location string, list bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" || parts[4] != "locations" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	if len(parts) == 6 {
		location = strings.TrimSpace(parts[5])
		if location == "" {
			return "", "", false, false
		}
		return project, location, false, true
	}
	return "", "", false, false
}

func parseGCPSecurityPrivateCALocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPSecurityPrivateCACaPoolsCollectionPath(path string) (project, location string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "caPools" {
		return "", "", false, false
	}
	return project, location, true, true
}

func parseGCPSecurityPrivateCACaPoolPath(path string) (project, location, caPoolID string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "caPools" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	if caPoolID == "" {
		return "", "", "", false
	}
	return project, location, caPoolID, true
}

func parseGCPSecurityPrivateCACaPoolActionPath(path string) (project, location, caPoolID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "caPools" {
		return "", "", "", "", false
	}
	caPoolID, action, ok = parseGCPSecurityPrivateCAResourceAction(tail[1])
	if !ok {
		return "", "", "", "", false
	}
	return project, location, caPoolID, action, true
}

func parseGCPSecurityPrivateCACertificateAuthoritiesCollectionPath(path string) (project, location, caPoolID string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "caPools" || tail[2] != "certificateAuthorities" {
		return "", "", "", false, false
	}
	caPoolID = strings.TrimSpace(tail[1])
	if caPoolID == "" {
		return "", "", "", false, false
	}
	return project, location, caPoolID, true, true
}

func parseGCPSecurityPrivateCACertificateAuthorityPath(path string) (project, location, caPoolID, caID string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "caPools" || tail[2] != "certificateAuthorities" || strings.Contains(tail[3], ":") {
		return "", "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	caID = strings.TrimSpace(tail[3])
	if caPoolID == "" || caID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, caID, true
}

func parseGCPSecurityPrivateCACertificateAuthorityActionPath(path string) (project, location, caPoolID, caID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "caPools" || tail[2] != "certificateAuthorities" {
		return "", "", "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	caID, action, ok = parseGCPSecurityPrivateCAResourceAction(tail[3])
	if !ok || caPoolID == "" {
		return "", "", "", "", "", false
	}
	return project, location, caPoolID, caID, action, true
}

func parseGCPSecurityPrivateCACertificatesCollectionPath(path string) (project, location, caPoolID string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "caPools" || tail[2] != "certificates" {
		return "", "", "", false, false
	}
	caPoolID = strings.TrimSpace(tail[1])
	if caPoolID == "" {
		return "", "", "", false, false
	}
	return project, location, caPoolID, true, true
}

func parseGCPSecurityPrivateCACertificatePath(path string) (project, location, caPoolID, certificateID string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "caPools" || tail[2] != "certificates" || strings.Contains(tail[3], ":") {
		return "", "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	certificateID = strings.TrimSpace(tail[3])
	if caPoolID == "" || certificateID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, certificateID, true
}

func parseGCPSecurityPrivateCACertificateActionPath(path string) (project, location, caPoolID, certificateID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "caPools" || tail[2] != "certificates" {
		return "", "", "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	certificateID, action, ok = parseGCPSecurityPrivateCAResourceAction(tail[3])
	if !ok || caPoolID == "" {
		return "", "", "", "", "", false
	}
	return project, location, caPoolID, certificateID, action, true
}

func parseGCPSecurityPrivateCACertificateRevocationListsCollectionPath(path string) (project, location, caPoolID, caID string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "caPools" || tail[2] != "certificateAuthorities" || tail[4] != "certificateRevocationLists" {
		return "", "", "", "", false, false
	}
	caPoolID = strings.TrimSpace(tail[1])
	caID = strings.TrimSpace(tail[3])
	if caPoolID == "" || caID == "" {
		return "", "", "", "", false, false
	}
	return project, location, caPoolID, caID, true, true
}

func parseGCPSecurityPrivateCACertificateRevocationListPath(path string) (project, location, caPoolID, caID, crlID string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "caPools" || tail[2] != "certificateAuthorities" || tail[4] != "certificateRevocationLists" || strings.Contains(tail[3], ":") || strings.Contains(tail[5], ":") {
		return "", "", "", "", "", false
	}
	caPoolID = strings.TrimSpace(tail[1])
	caID = strings.TrimSpace(tail[3])
	crlID = strings.TrimSpace(tail[5])
	if caPoolID == "" || caID == "" || crlID == "" {
		return "", "", "", "", "", false
	}
	return project, location, caPoolID, caID, crlID, true
}

func parseGCPSecurityPrivateCACertificateTemplatesCollectionPath(path string) (project, location string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificateTemplates" {
		return "", "", false, false
	}
	return project, location, true, true
}

func parseGCPSecurityPrivateCACertificateTemplatePath(path string) (project, location, templateID string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "certificateTemplates" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	templateID = strings.TrimSpace(tail[1])
	if templateID == "" {
		return "", "", "", false
	}
	return project, location, templateID, true
}

func parseGCPSecurityPrivateCAOperationsCollectionPath(path string) (project, location string, list bool, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return "", "", false, false
	}
	return project, location, true, true
}

func parseGCPSecurityPrivateCAOperationPath(path string) (project, location, operationID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecurityPrivateCALocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", "", false
	}
	segment := strings.TrimSpace(tail[1])
	if segment == "" {
		return "", "", "", "", false
	}
	if strings.Contains(segment, ":") {
		operationID, action, ok = parseGCPSecurityPrivateCAResourceAction(segment)
		if !ok {
			return "", "", "", "", false
		}
		return project, location, operationID, action, true
	}
	return project, location, segment, "", true
}

func parseGCPSecurityPrivateCAIAMActionPath(path string) (resource, action string, ok bool) {
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return "", "", false
	}
	resourceAndAction := strings.TrimPrefix(path, "/gcp/v1/")
	resource, action, found := strings.Cut(resourceAndAction, ":")
	if !found || strings.TrimSpace(resource) == "" {
		return "", "", false
	}
	if !strings.HasPrefix(resource, "projects/") {
		return "", "", false
	}
	switch action {
	case "getIamPolicy", "setIamPolicy", "testIamPermissions":
		return strings.TrimSpace(resource), action, true
	default:
		return "", "", false
	}
}

func parseGCPSecurityPrivateCAResourceAction(segment string) (resourceID, action string, ok bool) {
	resourceID, action, found := strings.Cut(strings.TrimSpace(segment), ":")
	if !found || strings.TrimSpace(resourceID) == "" || strings.TrimSpace(action) == "" {
		return "", "", false
	}
	return strings.TrimSpace(resourceID), strings.TrimSpace(action), true
}

func parseGCPSecurityPrivateCACertificateAuthorityActionPathLoose(path string) (project, location, caPoolID, caID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "caPools" || parts[8] != "certificateAuthorities" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	caPoolID = strings.TrimSpace(parts[7])
	caID, action, ok = parseGCPSecurityPrivateCAResourceAction(parts[9])
	if !ok || project == "" || location == "" || caPoolID == "" {
		return "", "", "", "", "", false
	}
	return project, location, caPoolID, caID, action, true
}

func parseGCPSecurityPrivateCACaPoolResourceName(name string) (project, location, caPoolID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || caPoolID == "" {
		return "", "", "", false
	}
	return project, location, caPoolID, true
}

func parseGCPSecurityPrivateCACertificateAuthorityResourceName(name string) (project, location, caPoolID, caID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" || parts[6] != "certificateAuthorities" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	caID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || caPoolID == "" || caID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, caID, true
}

func parseGCPSecurityPrivateCACertificateResourceName(name string) (project, location, caPoolID, certificateID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" || parts[6] != "certificates" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	certificateID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || caPoolID == "" || certificateID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, certificateID, true
}

func parseGCPSecurityPrivateCACertificateTemplateResourceName(name string) (project, location, templateID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "certificateTemplates" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	templateID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || templateID == "" {
		return "", "", "", false
	}
	return project, location, templateID, true
}

func parseGCPSecurityPrivateCAPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPSecurityPrivateCAInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPSecurityPrivateCAList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecurityPrivateCAOutOfRange(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPSecurityPrivateCAJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "request body is unreadable")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpSecurityPrivateCABodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	if nested, ok := body[key].(map[string]any); ok {
		return nested
	}
	if key == "policy" {
		return map[string]any{}
	}
	if _, ok := body["name"]; ok {
		return body
	}
	return map[string]any{}
}

func gcpSecurityPrivateCAString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func isGCPSecurityPrivateCAID(id string) bool {
	id = strings.TrimSpace(id)
	if len(id) == 0 || len(id) > 63 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func gcpSecurityPrivateCALocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Private CA " + location,
		"labels": map[string]string{
			"service": "security_privateca",
			"stage":   "emulated",
		},
	}
}

func gcpSecurityPrivateCACaPool(project, location, caPoolID string) map[string]any {
	return map[string]any{
		"name": gcpSecurityPrivateCACaPoolName(project, location, caPoolID),
		"tier": "ENTERPRISE",
		"publishingOptions": map[string]any{
			"publishCaCert": true,
			"publishCrl":    true,
		},
		"issuancePolicy": map[string]any{
			"allowedIssuanceModes": map[string]any{
				"allowConfigBasedIssuance": true,
				"allowCsrBasedIssuance":    true,
			},
		},
		"labels": map[string]string{
			"env": "staged",
		},
	}
}

func gcpSecurityPrivateCACertificateAuthority(project, location, caPoolID, caID string) map[string]any {
	return map[string]any{
		"name":     gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID),
		"type":     "SELF_SIGNED",
		"state":    gcpSecurityPrivateCACertificateAuthorityState(caID),
		"lifetime": "31536000s",
		"keySpec": map[string]any{
			"algorithm": "RSA_PKCS1_4096_SHA256",
		},
		"config": map[string]any{
			"subjectConfig": map[string]any{
				"subject": map[string]any{
					"commonName": fmt.Sprintf("Stackyard %s", caID),
				},
			},
		},
		"pemCaCertificates": []any{
			"-----BEGIN CERTIFICATE-----\nSTACKYARD-PRIVATECA\n-----END CERTIFICATE-----",
		},
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime": gcpSecurityPrivateCAReferenceTime.Format(time.RFC3339),
		"updateTime": gcpSecurityPrivateCAReferenceTime.Add(2 * time.Minute).Format(time.RFC3339),
	}
}

func gcpSecurityPrivateCACertificate(project, location, caPoolID, certificateID string) map[string]any {
	state := gcpSecurityPrivateCACertificateState(certificateID)
	resp := map[string]any{
		"name":                       gcpSecurityPrivateCACertificateName(project, location, caPoolID, certificateID),
		"issuerCertificateAuthority": gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, "ca-1"),
		"lifetime":                   "86400s",
		"pemCertificate":             "-----BEGIN CERTIFICATE-----\nSTACKYARD-CERT\n-----END CERTIFICATE-----",
		"certificateDescription": map[string]any{
			"subjectDescription": map[string]any{
				"subject": map[string]any{
					"commonName": fmt.Sprintf("stackyard-%s", certificateID),
				},
			},
			"subjectKeyId": map[string]any{
				"keyId": "stackyard-subject-key-id",
			},
		},
		"pemCertificateChain": []any{
			"-----BEGIN CERTIFICATE-----\nSTACKYARD-CERT-CHAIN\n-----END CERTIFICATE-----",
		},
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime": gcpSecurityPrivateCAReferenceTime.Add(1 * time.Hour).Format(time.RFC3339),
		"updateTime": gcpSecurityPrivateCAReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
	}
	if state == "REVOKED" {
		resp["revocationDetails"] = map[string]any{
			"revocationState": "KEY_COMPROMISE",
			"revocationTime":  gcpSecurityPrivateCAReferenceTime.Add(3 * time.Hour).Format(time.RFC3339),
		}
	}
	return resp
}

func gcpSecurityPrivateCACertificateRevocationList(project, location, caPoolID, caID, crlID string) map[string]any {
	return map[string]any{
		"name":           gcpSecurityPrivateCACertificateRevocationListName(project, location, caPoolID, caID, crlID),
		"sequenceNumber": "1",
		"pemCrl":         "-----BEGIN X509 CRL-----\nSTACKYARD-CRL\n-----END X509 CRL-----",
		"accessUrl":      fmt.Sprintf("https://privateca.stackyard.local/%s/%s/%s/%s/%s.crl", project, location, caPoolID, caID, crlID),
		"state":          "ACTIVE",
		"revisionId":     "00000001",
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime": gcpSecurityPrivateCAReferenceTime.Add(3 * time.Hour).Format(time.RFC3339),
		"updateTime": gcpSecurityPrivateCAReferenceTime.Add(4 * time.Hour).Format(time.RFC3339),
	}
}

func gcpSecurityPrivateCACertificateTemplate(project, location, templateID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityPrivateCACertificateTemplateName(project, location, templateID),
		"description":     "Stackyard template " + templateID,
		"maximumLifetime": "2592000s",
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime": gcpSecurityPrivateCAReferenceTime.Add(5 * time.Hour).Format(time.RFC3339),
		"updateTime": gcpSecurityPrivateCAReferenceTime.Add(6 * time.Hour).Format(time.RFC3339),
	}
}

func gcpSecurityPrivateCAPolicy(resource string, input map[string]any) map[string]any {
	policy := map[string]any{
		"version": 1,
		"etag":    "c3RhY2t5YXJkLXByaXZhdGVjYS1ldGFn",
		"bindings": []any{
			map[string]any{
				"role":    "roles/privateca.admin",
				"members": []any{"user:stackyard@example.com"},
			},
		},
	}
	if input != nil {
		if version, ok := input["version"]; ok {
			policy["version"] = version
		}
		if etag, ok := input["etag"]; ok {
			policy["etag"] = etag
		}
		if bindings, ok := input["bindings"]; ok {
			policy["bindings"] = bindings
		}
	}
	policy["resource"] = resource
	return policy
}

func gcpSecurityPrivateCAOperation(project, location, operationID, verb, target string) map[string]any {
	op := map[string]any{
		"name": gcpSecurityPrivateCAOperationName(project, location, operationID),
		"done": true,
		"metadata": map[string]any{
			"verb":   verb,
			"target": target,
		},
		"response": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
	if strings.TrimSpace(target) != "" {
		op["response"] = map[string]any{
			"@type": "type.googleapis.com/google.cloud.security.privateca.v1.OperationResponse",
			"name":  target,
		}
	}
	return op
}

func gcpSecurityPrivateCACertificateAuthorityState(caID string) string {
	caID = strings.ToLower(strings.TrimSpace(caID))
	switch {
	case strings.Contains(caID, "awaiting"):
		return "AWAITING_USER_ACTIVATION"
	case strings.Contains(caID, "disabled"):
		return "DISABLED"
	case strings.Contains(caID, "deleted"):
		return "DELETED"
	default:
		return "ENABLED"
	}
}

func gcpSecurityPrivateCACertificateState(certificateID string) string {
	certificateID = strings.ToLower(strings.TrimSpace(certificateID))
	if strings.Contains(certificateID, "revoked") {
		return "REVOKED"
	}
	return "ACTIVE"
}

func gcpSecurityPrivateCACaPoolName(project, location, caPoolID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/caPools/%s", project, location, caPoolID)
}

func gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID string) string {
	return fmt.Sprintf("%s/certificateAuthorities/%s", gcpSecurityPrivateCACaPoolName(project, location, caPoolID), caID)
}

func gcpSecurityPrivateCACertificateName(project, location, caPoolID, certificateID string) string {
	return fmt.Sprintf("%s/certificates/%s", gcpSecurityPrivateCACaPoolName(project, location, caPoolID), certificateID)
}

func gcpSecurityPrivateCACertificateRevocationListName(project, location, caPoolID, caID, crlID string) string {
	return fmt.Sprintf("%s/certificateRevocationLists/%s", gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID), crlID)
}

func gcpSecurityPrivateCACertificateTemplateName(project, location, templateID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/certificateTemplates/%s", project, location, templateID)
}

func gcpSecurityPrivateCAOperationName(project, location, operationID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
}

func respondGCPSecurityPrivateCAInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityPrivateCAFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityPrivateCAOutOfRange(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "OutOfRange",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_security_privateca(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "security_privateca") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSecurityPrivateCAInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/caPools/pool-1",
			"service":  "security_privateca",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
