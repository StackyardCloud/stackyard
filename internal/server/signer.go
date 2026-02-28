package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	signersvc "github.com/stackyard/stackyard/internal/services/signer"
)

type signerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

type signerPutSigningProfileRequest struct {
	SigningMaterial *struct {
		CertificateARN string `json:"certificateArn"`
	} `json:"signingMaterial"`
	SignatureValidityPeriod *struct {
		Value int32  `json:"value"`
		Type  string `json:"type"`
	} `json:"signatureValidityPeriod"`
	PlatformID string `json:"platformId"`
	Overrides  *struct {
		SigningConfiguration *struct {
			EncryptionAlgorithm string `json:"encryptionAlgorithm"`
			HashAlgorithm       string `json:"hashAlgorithm"`
		} `json:"signingConfiguration"`
		SigningImageFormat string `json:"signingImageFormat"`
	} `json:"overrides"`
	SigningParameters map[string]string `json:"signingParameters"`
	Tags              map[string]string `json:"tags"`
}

type signerAddProfilePermissionRequest struct {
	ProfileVersion string `json:"profileVersion"`
	Action         string `json:"action"`
	Principal      string `json:"principal"`
	RevisionID     string `json:"revisionId"`
	StatementID    string `json:"statementId"`
}

type signerStartSigningJobRequest struct {
	Source *struct {
		S3 *struct {
			BucketName string `json:"bucketName"`
			Key        string `json:"key"`
			Version    string `json:"version"`
		} `json:"s3"`
	} `json:"source"`
	Destination *struct {
		S3 *struct {
			BucketName string `json:"bucketName"`
			Prefix     string `json:"prefix"`
		} `json:"s3"`
	} `json:"destination"`
	ProfileName        string `json:"profileName"`
	ClientRequestToken string `json:"clientRequestToken"`
	ProfileOwner       string `json:"profileOwner"`
}

type signerRevokeSignatureRequest struct {
	JobOwner string `json:"jobOwner"`
	Reason   string `json:"reason"`
}

type signerRevokeSigningProfileRequest struct {
	ProfileVersion string `json:"profileVersion"`
	Reason         string `json:"reason"`
	EffectiveTime  any    `json:"effectiveTime"`
}

type signerSignPayloadRequest struct {
	ProfileName   string `json:"profileName"`
	ProfileOwner  string `json:"profileOwner"`
	Payload       []byte `json:"payload"`
	PayloadFormat string `json:"payloadFormat"`
}

type signerGetRevocationStatusRequest struct {
	SignatureTimestamp any      `json:"signatureTimestamp"`
	PlatformID         string   `json:"platformId"`
	ProfileVersionARN  string   `json:"profileVersionArn"`
	JobARN             string   `json:"jobArn"`
	CertificateHashes  []string `json:"certificateHashes"`
}

type signerTagResourceRequest struct {
	Tags map[string]string `json:"tags"`
}

func (s *Server) handleSignerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSignerRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "signer")
	if !ok {
		respondSignerError(w, status, code, msg)
		return true
	}

	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}

	switch segments[0] {
	case "signing-profiles":
		s.handleSignerSigningProfiles(w, r, segments)
		return true
	case "signing-jobs":
		s.handleSignerSigningJobs(w, r, segments)
		return true
	case "signing-platforms":
		s.handleSignerSigningPlatforms(w, r, segments)
		return true
	case "revocations":
		s.handleSignerRevocations(w, r, segments)
		return true
	case "tags":
		s.handleSignerTags(w, r, segments)
		return true
	default:
		respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}
}

func (s *Server) handleSignerSigningProfiles(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			includeCanceled, err := parseSignerOptionalBool(r.URL.Query().Get("includeCanceled"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid includeCanceled")
				return
			}
			maxResults, err := parseSignerOptionalMaxResults(r.URL.Query().Get("maxResults"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid maxResults")
				return
			}
			nextToken := strings.TrimSpace(r.URL.Query().Get("nextToken"))
			platformID := strings.TrimSpace(r.URL.Query().Get("platformId"))
			statuses := signerQueryList(r.URL.Query(), "statuses")

			profiles, outNextToken, err := s.signer.ListSigningProfiles(
				includeCanceled,
				platformID,
				statuses,
				maxResults,
				nextToken,
			)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			outProfiles := make([]map[string]any, 0, len(profiles))
			for _, profile := range profiles {
				outProfiles = append(outProfiles, signerSigningProfileSummaryPayload(profile))
			}
			out := map[string]any{"profiles": outProfiles}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondSignerJSON(w, http.StatusOK, out)
			return
		default:
			respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	profileName, ok := decodeSignerPathSegment(segments[1])
	if !ok {
		respondSignerError(w, http.StatusBadRequest, "ValidationException", "profileName is required")
		return
	}

	if len(segments) == 2 {
		switch r.Method {
		case http.MethodPut:
			var req signerPutSigningProfileRequest
			if err := decodeSignerJSONBody(r, &req); err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return
			}
			var material *signersvc.SigningMaterial
			if req.SigningMaterial != nil {
				material = &signersvc.SigningMaterial{CertificateARN: strings.TrimSpace(req.SigningMaterial.CertificateARN)}
			}
			var validity *signersvc.SignatureValidityPeriod
			if req.SignatureValidityPeriod != nil {
				validity = &signersvc.SignatureValidityPeriod{
					Value: req.SignatureValidityPeriod.Value,
					Type:  strings.TrimSpace(req.SignatureValidityPeriod.Type),
				}
			}
			var overrides *signersvc.SigningPlatformOverrides
			if req.Overrides != nil {
				overrides = &signersvc.SigningPlatformOverrides{SigningImageFormat: strings.TrimSpace(req.Overrides.SigningImageFormat)}
				if req.Overrides.SigningConfiguration != nil {
					overrides.SigningConfiguration = &signersvc.SigningConfigurationOverrides{
						EncryptionAlgorithm: strings.TrimSpace(req.Overrides.SigningConfiguration.EncryptionAlgorithm),
						HashAlgorithm:       strings.TrimSpace(req.Overrides.SigningConfiguration.HashAlgorithm),
					}
				}
			}
			profile, err := s.signer.PutSigningProfile(
				profileName,
				strings.TrimSpace(req.PlatformID),
				material,
				validity,
				overrides,
				req.SigningParameters,
				req.Tags,
			)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			respondSignerJSON(w, http.StatusOK, map[string]any{
				"arn":               profile.Arn,
				"profileVersion":    profile.ProfileVersion,
				"profileVersionArn": profile.ProfileVersionARN,
			})
			return
		case http.MethodGet:
			profileOwner := strings.TrimSpace(r.URL.Query().Get("profileOwner"))
			profile, err := s.signer.GetSigningProfile(profileName, profileOwner)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			respondSignerJSON(w, http.StatusOK, signerSigningProfileDetailPayload(profile))
			return
		case http.MethodDelete:
			if err := s.signer.CancelSigningProfile(profileName); err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			respondSignerJSON(w, http.StatusOK, map[string]any{})
			return
		default:
			respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 3 && segments[2] == "permissions" {
		switch r.Method {
		case http.MethodPost:
			var req signerAddProfilePermissionRequest
			if err := decodeSignerJSONBody(r, &req); err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return
			}
			revisionID, err := s.signer.AddProfilePermission(
				profileName,
				strings.TrimSpace(req.ProfileVersion),
				strings.TrimSpace(req.Action),
				strings.TrimSpace(req.Principal),
				strings.TrimSpace(req.RevisionID),
				strings.TrimSpace(req.StatementID),
			)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			respondSignerJSON(w, http.StatusOK, map[string]any{"revisionId": revisionID})
			return
		case http.MethodGet:
			nextToken := strings.TrimSpace(r.URL.Query().Get("nextToken"))
			revisionID, policySizeBytes, permissions, outNextToken, err := s.signer.ListProfilePermissions(profileName, nextToken)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			outPermissions := make([]map[string]any, 0, len(permissions))
			for _, permission := range permissions {
				outPermissions = append(outPermissions, map[string]any{
					"action":         permission.Action,
					"principal":      permission.Principal,
					"statementId":    permission.StatementID,
					"profileVersion": permission.ProfileVersion,
				})
			}
			out := map[string]any{
				"revisionId":      revisionID,
				"policySizeBytes": policySizeBytes,
				"permissions":     outPermissions,
			}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondSignerJSON(w, http.StatusOK, out)
			return
		default:
			respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 4 && segments[2] == "permissions" && r.Method == http.MethodDelete {
		statementID, ok := decodeSignerPathSegment(segments[3])
		if !ok {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "statementId is required")
			return
		}
		revisionID := strings.TrimSpace(r.URL.Query().Get("revisionId"))
		newRevisionID, err := s.signer.RemoveProfilePermission(profileName, revisionID, statementID)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{"revisionId": newRevisionID})
		return
	}

	if len(segments) == 3 && segments[2] == "revoke" && r.Method == http.MethodPut {
		var req signerRevokeSigningProfileRequest
		if err := decodeSignerJSONBody(r, &req); err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		effectiveTime, err := parseSignerTimestampValue(req.EffectiveTime)
		if err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid effectiveTime")
			return
		}
		if err := s.signer.RevokeSigningProfile(
			profileName,
			strings.TrimSpace(req.ProfileVersion),
			strings.TrimSpace(req.Reason),
			effectiveTime,
		); err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{})
		return
	}

	respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleSignerSigningJobs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodPost:
			var req signerStartSigningJobRequest
			if err := decodeSignerJSONBody(r, &req); err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return
			}
			var source *signersvc.Source
			if req.Source != nil && req.Source.S3 != nil {
				source = &signersvc.Source{
					S3: &signersvc.S3Source{
						BucketName: strings.TrimSpace(req.Source.S3.BucketName),
						Key:        strings.TrimSpace(req.Source.S3.Key),
						Version:    strings.TrimSpace(req.Source.S3.Version),
					},
				}
			}
			var destination *signersvc.Destination
			if req.Destination != nil && req.Destination.S3 != nil {
				destination = &signersvc.Destination{
					S3: &signersvc.S3Destination{
						BucketName: strings.TrimSpace(req.Destination.S3.BucketName),
						Prefix:     strings.TrimSpace(req.Destination.S3.Prefix),
					},
				}
			}
			jobID, jobOwner, err := s.signer.StartSigningJob(
				source,
				destination,
				strings.TrimSpace(req.ProfileName),
				strings.TrimSpace(req.ClientRequestToken),
				strings.TrimSpace(req.ProfileOwner),
			)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			respondSignerJSON(w, http.StatusOK, map[string]any{"jobId": jobID, "jobOwner": jobOwner})
			return
		case http.MethodGet:
			maxResults, err := parseSignerOptionalMaxResults(r.URL.Query().Get("maxResults"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid maxResults")
				return
			}
			nextToken := strings.TrimSpace(r.URL.Query().Get("nextToken"))
			status := strings.TrimSpace(r.URL.Query().Get("status"))
			platformID := strings.TrimSpace(r.URL.Query().Get("platformId"))
			requestedBy := strings.TrimSpace(r.URL.Query().Get("requestedBy"))
			jobInvoker := strings.TrimSpace(r.URL.Query().Get("jobInvoker"))
			isRevoked, err := parseSignerOptionalBoolPtr(r.URL.Query().Get("isRevoked"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid isRevoked")
				return
			}
			signatureExpiresBefore, err := parseSignerOptionalTimestamp(r.URL.Query().Get("signatureExpiresBefore"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid signatureExpiresBefore")
				return
			}
			signatureExpiresAfter, err := parseSignerOptionalTimestamp(r.URL.Query().Get("signatureExpiresAfter"))
			if err != nil {
				respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid signatureExpiresAfter")
				return
			}

			jobs, outNextToken, err := s.signer.ListSigningJobs(
				status,
				platformID,
				requestedBy,
				isRevoked,
				signatureExpiresBefore,
				signatureExpiresAfter,
				jobInvoker,
				maxResults,
				nextToken,
			)
			if err != nil {
				respondSignerErrorForErr(w, err)
				return
			}
			outJobs := make([]map[string]any, 0, len(jobs))
			for _, job := range jobs {
				outJobs = append(outJobs, signerSigningJobSummaryPayload(job))
			}
			out := map[string]any{"jobs": outJobs}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondSignerJSON(w, http.StatusOK, out)
			return
		default:
			respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 2 && segments[1] == "with-payload" && r.Method == http.MethodPost {
		var req signerSignPayloadRequest
		if err := decodeSignerJSONBody(r, &req); err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		jobID, jobOwner, metadata, signature, err := s.signer.SignPayload(
			strings.TrimSpace(req.ProfileName),
			strings.TrimSpace(req.ProfileOwner),
			req.Payload,
			strings.TrimSpace(req.PayloadFormat),
		)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{
			"jobId":     jobID,
			"jobOwner":  jobOwner,
			"metadata":  metadata,
			"signature": signature,
		})
		return
	}

	jobID, ok := decodeSignerPathSegment(segments[1])
	if !ok {
		respondSignerError(w, http.StatusBadRequest, "ValidationException", "jobId is required")
		return
	}

	if len(segments) == 2 && r.Method == http.MethodGet {
		job, err := s.signer.DescribeSigningJob(jobID)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, signerSigningJobDetailPayload(job))
		return
	}

	if len(segments) == 3 && segments[2] == "revoke" && r.Method == http.MethodPut {
		var req signerRevokeSignatureRequest
		if err := decodeSignerJSONBody(r, &req); err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		if err := s.signer.RevokeSignature(jobID, strings.TrimSpace(req.JobOwner), strings.TrimSpace(req.Reason)); err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{})
		return
	}

	respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleSignerSigningPlatforms(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
		maxResults, err := parseSignerOptionalMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid maxResults")
			return
		}
		nextToken := strings.TrimSpace(r.URL.Query().Get("nextToken"))
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		partner := strings.TrimSpace(r.URL.Query().Get("partner"))
		target := strings.TrimSpace(r.URL.Query().Get("target"))

		platforms, outNextToken, err := s.signer.ListSigningPlatforms(category, partner, target, maxResults, nextToken)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		outPlatforms := make([]map[string]any, 0, len(platforms))
		for _, platform := range platforms {
			outPlatforms = append(outPlatforms, signerSigningPlatformPayload(platform))
		}
		out := map[string]any{"platforms": outPlatforms}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondSignerJSON(w, http.StatusOK, out)
		return
	}

	if len(segments) == 2 && r.Method == http.MethodGet {
		platformID, ok := decodeSignerPathSegment(segments[1])
		if !ok {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "platformId is required")
			return
		}
		platform, err := s.signer.GetSigningPlatform(platformID)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, signerSigningPlatformPayload(platform))
		return
	}

	respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleSignerRevocations(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 1 || r.Method != http.MethodGet {
		respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}

	signatureRaw := strings.TrimSpace(r.URL.Query().Get("signatureTimestamp"))
	platformID := strings.TrimSpace(r.URL.Query().Get("platformId"))
	profileVersionARN := strings.TrimSpace(r.URL.Query().Get("profileVersionArn"))
	jobARN := strings.TrimSpace(r.URL.Query().Get("jobArn"))
	certificateHashes := signerQueryList(r.URL.Query(), "certificateHashes")

	if signatureRaw == "" && platformID == "" && profileVersionARN == "" && jobARN == "" && len(certificateHashes) == 0 {
		var req signerGetRevocationStatusRequest
		if err := decodeSignerJSONBody(r, &req); err == nil {
			if value, ok := req.SignatureTimestamp.(string); ok {
				signatureRaw = strings.TrimSpace(value)
			}
			if signatureRaw == "" && req.SignatureTimestamp != nil {
				if parsed, err := parseSignerTimestampValue(req.SignatureTimestamp); err == nil {
					signatureRaw = parsed.Format(time.RFC3339Nano)
				}
			}
			platformID = strings.TrimSpace(req.PlatformID)
			profileVersionARN = strings.TrimSpace(req.ProfileVersionARN)
			jobARN = strings.TrimSpace(req.JobARN)
			certificateHashes = append(certificateHashes, req.CertificateHashes...)
		}
	}

	signatureTimestamp, err := parseSignerRequiredTimestamp(signatureRaw)
	if err != nil {
		respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid signatureTimestamp")
		return
	}

	revokedEntities, err := s.signer.GetRevocationStatus(
		signatureTimestamp,
		platformID,
		profileVersionARN,
		jobARN,
		certificateHashes,
	)
	if err != nil {
		respondSignerErrorForErr(w, err)
		return
	}
	respondSignerJSON(w, http.StatusOK, map[string]any{"revokedEntities": revokedEntities})
}

func (s *Server) handleSignerTags(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 2 {
		respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}

	resourceARN, ok := decodeSignerPathSegment(segments[1])
	if !ok {
		respondSignerError(w, http.StatusBadRequest, "ValidationException", "resourceArn is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		tags, err := s.signer.ListTagsForResource(resourceARN)
		if err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{"tags": tags})
		return
	case http.MethodPost:
		var req signerTagResourceRequest
		if err := decodeSignerJSONBody(r, &req); err != nil {
			respondSignerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		if err := s.signer.TagResource(resourceARN, req.Tags); err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{})
		return
	case http.MethodDelete:
		tagKeys := signerQueryList(r.URL.Query(), "tagKeys")
		if err := s.signer.UntagResource(resourceARN, tagKeys); err != nil {
			respondSignerErrorForErr(w, err)
			return
		}
		respondSignerJSON(w, http.StatusOK, map[string]any{})
		return
	default:
		respondSignerError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}
}

func isSignerRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "signer" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	isSignerHost := strings.Contains(host, ".signer.") || strings.HasPrefix(host, "signer.")

	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = "/"
	}

	prefixes := []string{
		"/revocations",
		"/signing-jobs",
		"/signing-platforms",
		"/signing-profiles",
		"/tags",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	if service == "signer" {
		return true
	}

	return isSignerHost
}

func respondSignerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSignerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSignerJSON(w, status, signerError{Type: code, Message: msg})
}

func respondSignerErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, signersvc.ErrInvalidParameter):
		respondSignerError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, signersvc.ErrNotFound):
		respondSignerError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	case errors.Is(err, signersvc.ErrConflict), errors.Is(err, signersvc.ErrAlreadyExists):
		respondSignerError(w, http.StatusConflict, "ConflictException", err.Error())
	default:
		respondSignerError(w, http.StatusBadRequest, "ValidationException", err.Error())
	}
}

func decodeSignerJSONBody(r *http.Request, out any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func decodeSignerPathSegment(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func parseSignerOptionalBool(value string) (bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, err
	}
	return parsed, nil
}

func parseSignerOptionalBoolPtr(value string) (*bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseSignerOptionalMaxResults(value string) (int32, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid maxResults")
	}
	return int32(parsed), nil
}

func parseSignerRequiredTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("missing timestamp")
	}
	return parseSignerTimestamp(trimmed)
}

func parseSignerOptionalTimestamp(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := parseSignerTimestamp(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseSignerTimestamp(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			if layout == "2006-01-02T15:04:05" {
				return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), time.UTC), nil
			}
			return parsed.UTC(), nil
		}
	}
	if iv, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(iv, 0).UTC(), nil
	}
	if fv, err := strconv.ParseFloat(value, 64); err == nil {
		sec := int64(fv)
		ns := int64((fv - float64(sec)) * float64(time.Second))
		return time.Unix(sec, ns).UTC(), nil
	}
	return time.Time{}, errors.New("invalid timestamp")
}

func parseSignerTimestampValue(value any) (time.Time, error) {
	switch typed := value.(type) {
	case nil:
		return time.Time{}, errors.New("missing timestamp")
	case string:
		return parseSignerTimestamp(strings.TrimSpace(typed))
	case float64:
		sec := int64(typed)
		ns := int64((typed - float64(sec)) * float64(time.Second))
		return time.Unix(sec, ns).UTC(), nil
	case int64:
		return time.Unix(typed, 0).UTC(), nil
	case int:
		return time.Unix(int64(typed), 0).UTC(), nil
	default:
		return time.Time{}, errors.New("invalid timestamp")
	}
}

func signerQueryList(values url.Values, key string) []string {
	rawValues, ok := values[key]
	if !ok || len(rawValues) == 0 {
		return nil
	}
	out := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			out = append(out, trimmed)
		}
	}
	return out
}

func signerSigningProfileSummaryPayload(profile signersvc.SigningProfile) map[string]any {
	out := map[string]any{
		"profileName":         profile.ProfileName,
		"profileVersion":      profile.ProfileVersion,
		"profileVersionArn":   profile.ProfileVersionARN,
		"platformId":          profile.PlatformID,
		"platformDisplayName": profile.PlatformDisplayName,
		"status":              profile.Status,
		"arn":                 profile.Arn,
	}
	if profile.SigningMaterial != nil {
		out["signingMaterial"] = map[string]any{"certificateArn": profile.SigningMaterial.CertificateARN}
	}
	if profile.SignatureValidityPeriod != nil {
		out["signatureValidityPeriod"] = map[string]any{
			"value": profile.SignatureValidityPeriod.Value,
			"type":  profile.SignatureValidityPeriod.Type,
		}
	}
	if len(profile.SigningParameters) > 0 {
		out["signingParameters"] = profile.SigningParameters
	}
	if len(profile.Tags) > 0 {
		out["tags"] = profile.Tags
	}
	return out
}

func signerSigningProfileDetailPayload(profile signersvc.SigningProfile) map[string]any {
	out := signerSigningProfileSummaryPayload(profile)
	if overrides := signerSigningOverridesPayload(profile.Overrides); overrides != nil {
		out["overrides"] = overrides
	}
	if profile.StatusReason != "" {
		out["statusReason"] = profile.StatusReason
	}
	if profile.RevocationRecord != nil {
		out["revocationRecord"] = map[string]any{
			"revocationEffectiveFrom": profile.RevocationRecord.RevocationEffectiveFrom,
			"revokedAt":               profile.RevocationRecord.RevokedAt,
			"revokedBy":               profile.RevocationRecord.RevokedBy,
		}
	}
	return out
}

func signerSigningJobSummaryPayload(job signersvc.SigningJob) map[string]any {
	out := map[string]any{
		"jobId":               job.JobID,
		"source":              signerSourcePayload(job.Source),
		"signedObject":        signerSignedObjectPayload(job.SignedObject),
		"signingMaterial":     signerSigningMaterialPayload(job.SigningMaterial),
		"createdAt":           job.CreatedAt,
		"status":              job.Status,
		"isRevoked":           job.IsRevoked,
		"profileName":         job.ProfileName,
		"profileVersion":      job.ProfileVersion,
		"platformId":          job.PlatformID,
		"platformDisplayName": job.PlatformDisplayName,
		"jobOwner":            job.JobOwner,
		"jobInvoker":          job.JobInvoker,
	}
	if job.SignatureExpiresAt != nil {
		out["signatureExpiresAt"] = *job.SignatureExpiresAt
	}
	return out
}

func signerSigningJobDetailPayload(job signersvc.SigningJob) map[string]any {
	out := map[string]any{
		"jobId":               job.JobID,
		"source":              signerSourcePayload(job.Source),
		"signingMaterial":     signerSigningMaterialPayload(job.SigningMaterial),
		"platformId":          job.PlatformID,
		"platformDisplayName": job.PlatformDisplayName,
		"profileName":         job.ProfileName,
		"profileVersion":      job.ProfileVersion,
		"signingParameters":   job.SigningParameters,
		"createdAt":           job.CreatedAt,
		"requestedBy":         job.RequestedBy,
		"status":              job.Status,
		"signedObject":        signerSignedObjectPayload(job.SignedObject),
		"jobOwner":            job.JobOwner,
		"jobInvoker":          job.JobInvoker,
	}
	if job.JobARN != "" {
		out["jobArn"] = job.JobARN
	}
	if overrides := signerSigningOverridesPayload(job.Overrides); overrides != nil {
		out["overrides"] = overrides
	}
	if job.CompletedAt != nil {
		out["completedAt"] = *job.CompletedAt
	}
	if job.SignatureExpiresAt != nil {
		out["signatureExpiresAt"] = *job.SignatureExpiresAt
	}
	if job.StatusReason != "" {
		out["statusReason"] = job.StatusReason
	}
	if job.RevocationRecord != nil {
		out["revocationRecord"] = map[string]any{
			"reason":    job.RevocationRecord.Reason,
			"revokedAt": job.RevocationRecord.RevokedAt,
			"revokedBy": job.RevocationRecord.RevokedBy,
		}
	}
	return out
}

func signerSigningPlatformPayload(platform signersvc.SigningPlatform) map[string]any {
	out := map[string]any{
		"platformId":          platform.PlatformID,
		"displayName":         platform.DisplayName,
		"partner":             platform.Partner,
		"target":              platform.Target,
		"category":            platform.Category,
		"maxSizeInMB":         platform.MaxSizeInMB,
		"revocationSupported": platform.RevocationSupported,
	}
	if platform.SigningConfiguration != nil {
		signingConfig := map[string]any{}
		if platform.SigningConfiguration.EncryptionAlgorithmOptions != nil {
			signingConfig["encryptionAlgorithmOptions"] = map[string]any{
				"allowedValues": platform.SigningConfiguration.EncryptionAlgorithmOptions.AllowedValues,
				"defaultValue":  platform.SigningConfiguration.EncryptionAlgorithmOptions.DefaultValue,
			}
		}
		if platform.SigningConfiguration.HashAlgorithmOptions != nil {
			signingConfig["hashAlgorithmOptions"] = map[string]any{
				"allowedValues": platform.SigningConfiguration.HashAlgorithmOptions.AllowedValues,
				"defaultValue":  platform.SigningConfiguration.HashAlgorithmOptions.DefaultValue,
			}
		}
		out["signingConfiguration"] = signingConfig
	}
	if platform.SigningImageFormat != nil {
		out["signingImageFormat"] = map[string]any{
			"supportedFormats": platform.SigningImageFormat.SupportedFormats,
			"defaultFormat":    platform.SigningImageFormat.DefaultFormat,
		}
	}
	return out
}

func signerSigningMaterialPayload(material *signersvc.SigningMaterial) map[string]any {
	if material == nil {
		return nil
	}
	return map[string]any{"certificateArn": material.CertificateARN}
}

func signerSourcePayload(source *signersvc.Source) map[string]any {
	if source == nil {
		return nil
	}
	out := map[string]any{}
	if source.S3 != nil {
		out["s3"] = map[string]any{
			"bucketName": source.S3.BucketName,
			"key":        source.S3.Key,
			"version":    source.S3.Version,
		}
	}
	return out
}

func signerSignedObjectPayload(object *signersvc.SignedObject) map[string]any {
	if object == nil {
		return nil
	}
	out := map[string]any{}
	if object.S3 != nil {
		out["s3"] = map[string]any{
			"bucketName": object.S3.BucketName,
			"key":        object.S3.Key,
		}
	}
	return out
}

func signerSigningOverridesPayload(overrides *signersvc.SigningPlatformOverrides) map[string]any {
	if overrides == nil {
		return nil
	}
	out := map[string]any{}
	if overrides.SigningConfiguration != nil {
		out["signingConfiguration"] = map[string]any{
			"encryptionAlgorithm": overrides.SigningConfiguration.EncryptionAlgorithm,
			"hashAlgorithm":       overrides.SigningConfiguration.HashAlgorithm,
		}
	}
	if overrides.SigningImageFormat != "" {
		out["signingImageFormat"] = overrides.SigningImageFormat
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
