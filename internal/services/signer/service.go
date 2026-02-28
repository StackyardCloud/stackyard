package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrConflict         = errors.New("conflict")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type SigningMaterial struct {
	CertificateARN string
}

type SignatureValidityPeriod struct {
	Value int32
	Type  string
}

type SigningConfigurationOverrides struct {
	EncryptionAlgorithm string
	HashAlgorithm       string
}

type SigningPlatformOverrides struct {
	SigningConfiguration *SigningConfigurationOverrides
	SigningImageFormat   string
}

type SigningProfileRevocationRecord struct {
	RevocationEffectiveFrom time.Time
	RevokedAt               time.Time
	RevokedBy               string
}

type SigningJobRevocationRecord struct {
	Reason    string
	RevokedAt time.Time
	RevokedBy string
}

type SigningProfile struct {
	ProfileName             string
	ProfileVersion          string
	ProfileVersionARN       string
	Arn                     string
	Status                  string
	StatusReason            string
	PlatformID              string
	PlatformDisplayName     string
	SigningMaterial         *SigningMaterial
	SignatureValidityPeriod *SignatureValidityPeriod
	Overrides               *SigningPlatformOverrides
	SigningParameters       map[string]string
	Tags                    map[string]string
	RevocationRecord        *SigningProfileRevocationRecord
	PermissionRevision      uint64
}

type ProfilePermission struct {
	Action         string
	Principal      string
	StatementID    string
	ProfileVersion string
}

type S3Source struct {
	BucketName string
	Key        string
	Version    string
}

type Source struct {
	S3 *S3Source
}

type S3Destination struct {
	BucketName string
	Prefix     string
}

type Destination struct {
	S3 *S3Destination
}

type S3SignedObject struct {
	BucketName string
	Key        string
}

type SignedObject struct {
	S3 *S3SignedObject
}

type SigningJob struct {
	JobID               string
	JobARN              string
	Source              *Source
	SignedObject        *SignedObject
	SigningMaterial     *SigningMaterial
	PlatformID          string
	PlatformDisplayName string
	ProfileName         string
	ProfileVersion      string
	Overrides           *SigningPlatformOverrides
	SigningParameters   map[string]string
	CreatedAt           time.Time
	CompletedAt         *time.Time
	SignatureExpiresAt  *time.Time
	RequestedBy         string
	Status              string
	StatusReason        string
	RevocationRecord    *SigningJobRevocationRecord
	IsRevoked           bool
	JobOwner            string
	JobInvoker          string
}

type EnumOptions struct {
	AllowedValues []string
	DefaultValue  string
}

type SigningConfiguration struct {
	EncryptionAlgorithmOptions *EnumOptions
	HashAlgorithmOptions       *EnumOptions
}

type SigningImageFormat struct {
	SupportedFormats []string
	DefaultFormat    string
}

type SigningPlatform struct {
	PlatformID           string
	DisplayName          string
	Partner              string
	Target               string
	Category             string
	SigningConfiguration *SigningConfiguration
	SigningImageFormat   *SigningImageFormat
	MaxSizeInMB          int32
	RevocationSupported  bool
}

type listJobsFilter struct {
	Status                 string
	PlatformID             string
	RequestedBy            string
	IsRevoked              *bool
	SignatureExpiresBefore *time.Time
	SignatureExpiresAfter  *time.Time
	JobInvoker             string
}

type Service struct {
	mu sync.Mutex

	profileSeq uint64
	jobSeq     uint64

	profiles            map[string]*SigningProfile
	profilePermissions  map[string]map[string]*ProfilePermission
	jobs                map[string]*SigningJob
	idempotencyTokenJob map[string]string
	resourceTags        map[string]map[string]string
	platforms           map[string]*SigningPlatform
}

func NewService() *Service {
	s := &Service{
		profiles:            map[string]*SigningProfile{},
		profilePermissions:  map[string]map[string]*ProfilePermission{},
		jobs:                map[string]*SigningJob{},
		idempotencyTokenJob: map[string]string{},
		resourceTags:        map[string]map[string]string{},
		platforms:           map[string]*SigningPlatform{},
	}
	s.seedPlatforms()
	return s
}

func (s *Service) seedPlatforms() {
	s.platforms["AWSLambda-SHA384-ECDSA"] = &SigningPlatform{
		PlatformID:  "AWSLambda-SHA384-ECDSA",
		DisplayName: "AWS Lambda SHA384 ECDSA",
		Partner:     "AWS",
		Target:      "Lambda",
		Category:    "AWSIoT",
		SigningConfiguration: &SigningConfiguration{
			EncryptionAlgorithmOptions: &EnumOptions{
				AllowedValues: []string{"ECDSA", "RSA"},
				DefaultValue:  "ECDSA",
			},
			HashAlgorithmOptions: &EnumOptions{
				AllowedValues: []string{"SHA256"},
				DefaultValue:  "SHA256",
			},
		},
		SigningImageFormat: &SigningImageFormat{
			SupportedFormats: []string{"JSON", "JSONEmbedded", "JSONDetached"},
			DefaultFormat:    "JSON",
		},
		MaxSizeInMB:         10,
		RevocationSupported: true,
	}
}

func (s *Service) PutSigningProfile(
	profileName string,
	platformID string,
	signingMaterial *SigningMaterial,
	signatureValidityPeriod *SignatureValidityPeriod,
	overrides *SigningPlatformOverrides,
	signingParameters map[string]string,
	tags map[string]string,
) (SigningProfile, error) {
	profileName = strings.TrimSpace(profileName)
	platformID = strings.TrimSpace(platformID)
	if profileName == "" || platformID == "" {
		return SigningProfile{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	platform, ok := s.platforms[platformID]
	if !ok {
		return SigningProfile{}, ErrNotFound
	}

	s.profileSeq++
	version := fmt.Sprintf("%010d", s.profileSeq)
	arn := profileARN(profileName)
	versionARN := profileVersionARN(profileName, version)

	current := s.profiles[profileName]
	revision := uint64(0)
	if current != nil {
		revision = current.PermissionRevision
	}

	profile := &SigningProfile{
		ProfileName:             profileName,
		ProfileVersion:          version,
		ProfileVersionARN:       versionARN,
		Arn:                     arn,
		Status:                  "Active",
		PlatformID:              platformID,
		PlatformDisplayName:     platform.DisplayName,
		SigningMaterial:         cloneSigningMaterial(signingMaterial),
		SignatureValidityPeriod: cloneSignatureValidityPeriod(signatureValidityPeriod),
		Overrides:               cloneSigningPlatformOverrides(overrides),
		SigningParameters:       cloneStringMap(signingParameters),
		Tags:                    cloneStringMap(tags),
		PermissionRevision:      revision,
	}
	s.profiles[profileName] = profile

	tagBucket := cloneStringMap(profile.Tags)
	s.resourceTags[arn] = tagBucket
	s.resourceTags[versionARN] = cloneStringMap(tagBucket)

	return cloneSigningProfile(profile), nil
}

func (s *Service) GetSigningProfile(profileName, profileOwner string) (SigningProfile, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return SigningProfile{}, ErrInvalidParameter
	}
	profileOwner = strings.TrimSpace(profileOwner)
	if profileOwner != "" && profileOwner != DefaultAccountID {
		return SigningProfile{}, ErrNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return SigningProfile{}, ErrNotFound
	}
	return cloneSigningProfile(profile), nil
}

func (s *Service) ListSigningProfiles(
	includeCanceled bool,
	platformID string,
	statuses []string,
	maxResults int32,
	nextToken string,
) ([]SigningProfile, string, error) {
	if maxResults < 0 {
		return nil, "", ErrInvalidParameter
	}

	platformID = strings.TrimSpace(platformID)
	statusSet := map[string]struct{}{}
	for _, status := range statuses {
		trimmed := strings.TrimSpace(status)
		if trimmed == "" {
			continue
		}
		statusSet[trimmed] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]SigningProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		if !includeCanceled && profile.Status == "Canceled" {
			continue
		}
		if platformID != "" && profile.PlatformID != platformID {
			continue
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[profile.Status]; !ok {
				continue
			}
		}
		items = append(items, cloneSigningProfile(profile))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ProfileName < items[j].ProfileName
	})

	return paginateItems(items, maxResults, nextToken)
}

func (s *Service) CancelSigningProfile(profileName string) error {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return ErrNotFound
	}
	profile.Status = "Canceled"
	profile.StatusReason = "Canceled by request"
	return nil
}

func (s *Service) AddProfilePermission(
	profileName string,
	profileVersion string,
	action string,
	principal string,
	revisionID string,
	statementID string,
) (string, error) {
	profileName = strings.TrimSpace(profileName)
	profileVersion = strings.TrimSpace(profileVersion)
	action = strings.TrimSpace(action)
	principal = strings.TrimSpace(principal)
	revisionID = strings.TrimSpace(revisionID)
	statementID = strings.TrimSpace(statementID)
	if profileName == "" || action == "" || principal == "" || statementID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return "", ErrNotFound
	}
	if profileVersion != "" && profile.ProfileVersion != profileVersion {
		return "", ErrNotFound
	}
	if revisionID != "" && revisionID != strconv.FormatUint(profile.PermissionRevision, 10) {
		return "", ErrConflict
	}

	bucket := s.profilePermissions[profileName]
	if bucket == nil {
		bucket = map[string]*ProfilePermission{}
		s.profilePermissions[profileName] = bucket
	}
	bucket[statementID] = &ProfilePermission{
		Action:         action,
		Principal:      principal,
		StatementID:    statementID,
		ProfileVersion: profile.ProfileVersion,
	}
	profile.PermissionRevision++

	return strconv.FormatUint(profile.PermissionRevision, 10), nil
}

func (s *Service) ListProfilePermissions(profileName, nextToken string) (string, int32, []ProfilePermission, string, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return "", 0, nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return "", 0, nil, "", ErrNotFound
	}
	bucket := s.profilePermissions[profileName]
	items := make([]ProfilePermission, 0, len(bucket))
	for _, item := range bucket {
		items = append(items, clonePermission(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].StatementID < items[j].StatementID
	})

	paged, outToken, err := paginateItems(items, 100, nextToken)
	if err != nil {
		return "", 0, nil, "", err
	}
	policySize := int32(len(items) * 120)
	return strconv.FormatUint(profile.PermissionRevision, 10), policySize, paged, outToken, nil
}

func (s *Service) RemoveProfilePermission(profileName, revisionID, statementID string) (string, error) {
	profileName = strings.TrimSpace(profileName)
	revisionID = strings.TrimSpace(revisionID)
	statementID = strings.TrimSpace(statementID)
	if profileName == "" || revisionID == "" || statementID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return "", ErrNotFound
	}
	if revisionID != strconv.FormatUint(profile.PermissionRevision, 10) {
		return "", ErrConflict
	}
	bucket := s.profilePermissions[profileName]
	if bucket == nil {
		return "", ErrNotFound
	}
	if _, ok := bucket[statementID]; !ok {
		return "", ErrNotFound
	}
	delete(bucket, statementID)
	profile.PermissionRevision++
	return strconv.FormatUint(profile.PermissionRevision, 10), nil
}

func (s *Service) StartSigningJob(
	source *Source,
	destination *Destination,
	profileName string,
	clientRequestToken string,
	profileOwner string,
) (string, string, error) {
	profileName = strings.TrimSpace(profileName)
	clientRequestToken = strings.TrimSpace(clientRequestToken)
	profileOwner = strings.TrimSpace(profileOwner)
	if profileName == "" || clientRequestToken == "" {
		return "", "", ErrInvalidParameter
	}
	if profileOwner != "" && profileOwner != DefaultAccountID {
		return "", "", ErrNotFound
	}
	if source == nil || source.S3 == nil || strings.TrimSpace(source.S3.BucketName) == "" || strings.TrimSpace(source.S3.Key) == "" || strings.TrimSpace(source.S3.Version) == "" {
		return "", "", ErrInvalidParameter
	}
	if destination == nil || destination.S3 == nil || strings.TrimSpace(destination.S3.BucketName) == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return "", "", ErrNotFound
	}
	if profile.Status != "Active" {
		return "", "", ErrConflict
	}

	if existingID, ok := s.idempotencyTokenJob[clientRequestToken]; ok {
		if existing, ok := s.jobs[existingID]; ok {
			return existing.JobID, existing.JobOwner, nil
		}
	}

	s.jobSeq++
	jobID := fmt.Sprintf("job-%016x", s.jobSeq)
	jobArn := jobARN(jobID)
	now := time.Now().UTC()
	completed := now
	expires := now.Add(24 * time.Hour)
	signedKey := strings.TrimSpace(destination.S3.Prefix)
	if signedKey != "" && !strings.HasSuffix(signedKey, "/") {
		signedKey += "/"
	}
	signedKey += profileName + "-" + jobID + ".sig"

	job := &SigningJob{
		JobID:               jobID,
		JobARN:              jobArn,
		Source:              cloneSource(source),
		SignedObject:        &SignedObject{S3: &S3SignedObject{BucketName: destination.S3.BucketName, Key: signedKey}},
		SigningMaterial:     cloneSigningMaterial(profile.SigningMaterial),
		PlatformID:          profile.PlatformID,
		PlatformDisplayName: profile.PlatformDisplayName,
		ProfileName:         profile.ProfileName,
		ProfileVersion:      profile.ProfileVersion,
		Overrides:           cloneSigningPlatformOverrides(profile.Overrides),
		SigningParameters:   cloneStringMap(profile.SigningParameters),
		CreatedAt:           now,
		CompletedAt:         &completed,
		SignatureExpiresAt:  &expires,
		RequestedBy:         DefaultAccountID,
		Status:              "Succeeded",
		JobOwner:            DefaultAccountID,
		JobInvoker:          DefaultAccountID,
	}
	s.jobs[jobID] = job
	s.idempotencyTokenJob[clientRequestToken] = jobID
	s.resourceTags[jobArn] = map[string]string{}

	return jobID, job.JobOwner, nil
}

func (s *Service) DescribeSigningJob(jobID string) (SigningJob, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return SigningJob{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return SigningJob{}, ErrNotFound
	}
	return cloneSigningJob(job), nil
}

func (s *Service) ListSigningJobs(
	status, platformID, requestedBy string,
	isRevoked *bool,
	signatureExpiresBefore, signatureExpiresAfter *time.Time,
	jobInvoker string,
	maxResults int32,
	nextToken string,
) ([]SigningJob, string, error) {
	if maxResults < 0 {
		return nil, "", ErrInvalidParameter
	}
	filter := s.parseListSigningJobsFilter(
		status,
		platformID,
		requestedBy,
		isRevoked,
		signatureExpiresBefore,
		signatureExpiresAfter,
		jobInvoker,
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]SigningJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		if filter.PlatformID != "" && job.PlatformID != filter.PlatformID {
			continue
		}
		if filter.RequestedBy != "" && job.RequestedBy != filter.RequestedBy {
			continue
		}
		if filter.JobInvoker != "" && job.JobInvoker != filter.JobInvoker {
			continue
		}
		if filter.IsRevoked != nil && job.IsRevoked != *filter.IsRevoked {
			continue
		}
		if filter.SignatureExpiresBefore != nil {
			if job.SignatureExpiresAt == nil || !job.SignatureExpiresAt.Before(*filter.SignatureExpiresBefore) {
				continue
			}
		}
		if filter.SignatureExpiresAfter != nil {
			if job.SignatureExpiresAt == nil || !job.SignatureExpiresAt.After(*filter.SignatureExpiresAfter) {
				continue
			}
		}
		items = append(items, cloneSigningJob(job))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return paginateItems(items, maxResults, nextToken)
}

func (s *Service) RevokeSignature(jobID, jobOwner, reason string) error {
	jobID = strings.TrimSpace(jobID)
	jobOwner = strings.TrimSpace(jobOwner)
	reason = strings.TrimSpace(reason)
	if jobID == "" || reason == "" {
		return ErrInvalidParameter
	}
	if jobOwner != "" && jobOwner != DefaultAccountID {
		return ErrNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	job.IsRevoked = true
	job.RevocationRecord = &SigningJobRevocationRecord{
		Reason:    reason,
		RevokedAt: now,
		RevokedBy: DefaultAccountID,
	}
	return nil
}

func (s *Service) RevokeSigningProfile(profileName, profileVersion, reason string, effectiveTime time.Time) error {
	profileName = strings.TrimSpace(profileName)
	profileVersion = strings.TrimSpace(profileVersion)
	reason = strings.TrimSpace(reason)
	if profileName == "" || profileVersion == "" || reason == "" || effectiveTime.IsZero() {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return ErrNotFound
	}
	if profile.ProfileVersion != profileVersion {
		return ErrNotFound
	}
	now := time.Now().UTC()
	profile.Status = "Revoked"
	profile.StatusReason = reason
	profile.RevocationRecord = &SigningProfileRevocationRecord{
		RevocationEffectiveFrom: effectiveTime.UTC(),
		RevokedAt:               now,
		RevokedBy:               DefaultAccountID,
	}
	return nil
}

func (s *Service) GetRevocationStatus(signatureTimestamp time.Time, platformID, profileVersionARN, jobARN string, certificateHashes []string) ([]string, error) {
	platformID = strings.TrimSpace(platformID)
	profileVersionARN = strings.TrimSpace(profileVersionARN)
	jobARN = strings.TrimSpace(jobARN)
	if signatureTimestamp.IsZero() || platformID == "" || profileVersionARN == "" || jobARN == "" || len(certificateHashes) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	revoked := make([]string, 0, 2)
	if profileName, _, ok := parseProfileVersionARN(profileVersionARN); ok {
		if profile, found := s.profiles[profileName]; found && profile.RevocationRecord != nil {
			revoked = append(revoked, "SIGNING_PROFILE")
		}
	}
	if jobID, ok := parseJobARN(jobARN); ok {
		if job, found := s.jobs[jobID]; found && job.IsRevoked {
			revoked = append(revoked, "SIGNING_JOB")
		}
	}
	sort.Strings(revoked)
	return revoked, nil
}

func (s *Service) GetSigningPlatform(platformID string) (SigningPlatform, error) {
	platformID = strings.TrimSpace(platformID)
	if platformID == "" {
		return SigningPlatform{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	platform, ok := s.platforms[platformID]
	if !ok {
		return SigningPlatform{}, ErrNotFound
	}
	return cloneSigningPlatform(platform), nil
}

func (s *Service) ListSigningPlatforms(category, partner, target string, maxResults int32, nextToken string) ([]SigningPlatform, string, error) {
	if maxResults < 0 {
		return nil, "", ErrInvalidParameter
	}

	category = strings.TrimSpace(category)
	partner = strings.TrimSpace(partner)
	target = strings.TrimSpace(target)

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]SigningPlatform, 0, len(s.platforms))
	for _, platform := range s.platforms {
		if category != "" && !strings.EqualFold(platform.Category, category) {
			continue
		}
		if partner != "" && !strings.EqualFold(platform.Partner, partner) {
			continue
		}
		if target != "" && !strings.EqualFold(platform.Target, target) {
			continue
		}
		items = append(items, cloneSigningPlatform(platform))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].PlatformID < items[j].PlatformID
	})
	return paginateItems(items, maxResults, nextToken)
}

func (s *Service) SignPayload(profileName, profileOwner string, payload []byte, payloadFormat string) (string, string, map[string]string, []byte, error) {
	profileName = strings.TrimSpace(profileName)
	profileOwner = strings.TrimSpace(profileOwner)
	payloadFormat = strings.TrimSpace(payloadFormat)
	if profileName == "" || len(payload) == 0 || payloadFormat == "" {
		return "", "", nil, nil, ErrInvalidParameter
	}
	if profileOwner != "" && profileOwner != DefaultAccountID {
		return "", "", nil, nil, ErrNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[profileName]
	if !ok {
		return "", "", nil, nil, ErrNotFound
	}
	if profile.Status != "Active" {
		return "", "", nil, nil, ErrConflict
	}

	s.jobSeq++
	jobID := fmt.Sprintf("job-%016x", s.jobSeq)

	key := []byte("stackyard-signer:" + profile.ProfileVersionARN)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	signature := mac.Sum(nil)

	metadata := map[string]string{
		"profileVersionArn": profile.ProfileVersionARN,
		"signatureFormat":   strings.ToUpper(payloadFormat),
		"signatureB64":      base64.StdEncoding.EncodeToString(signature),
	}

	return jobID, DefaultAccountID, metadata, signature, nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" || len(tags) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	canonical, err := s.canonicalResourceARNLocked(resourceARN)
	if err != nil {
		return err
	}
	tagBucket := s.resourceTags[canonical]
	if tagBucket == nil {
		tagBucket = map[string]string{}
		s.resourceTags[canonical] = tagBucket
	}
	for key, value := range tags {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		tagBucket[key] = value
	}
	s.syncResourceTagsLocked(canonical, tagBucket)
	return nil
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	canonical, err := s.canonicalResourceARNLocked(resourceARN)
	if err != nil {
		return nil, err
	}
	return cloneStringMap(s.resourceTags[canonical]), nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" || len(tagKeys) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	canonical, err := s.canonicalResourceARNLocked(resourceARN)
	if err != nil {
		return err
	}
	tagBucket := s.resourceTags[canonical]
	if tagBucket == nil {
		tagBucket = map[string]string{}
		s.resourceTags[canonical] = tagBucket
	}
	for _, key := range tagKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(tagBucket, key)
	}
	s.syncResourceTagsLocked(canonical, tagBucket)
	return nil
}

func (s *Service) canonicalResourceARNLocked(resourceARN string) (string, error) {
	if profileName, version, ok := parseProfileVersionARN(resourceARN); ok {
		profile, exists := s.profiles[profileName]
		if !exists {
			return "", ErrNotFound
		}
		if version != "" && profile.ProfileVersion != version {
			return "", ErrNotFound
		}
		return profile.Arn, nil
	}
	if profileName, ok := parseProfileARN(resourceARN); ok {
		profile, exists := s.profiles[profileName]
		if !exists {
			return "", ErrNotFound
		}
		return profile.Arn, nil
	}
	if jobID, ok := parseJobARN(resourceARN); ok {
		job, exists := s.jobs[jobID]
		if !exists {
			return "", ErrNotFound
		}
		return job.JobARN, nil
	}
	return "", ErrNotFound
}

func (s *Service) syncResourceTagsLocked(canonicalARN string, tags map[string]string) {
	if profileName, ok := parseProfileARN(canonicalARN); ok {
		if profile, exists := s.profiles[profileName]; exists {
			profile.Tags = cloneStringMap(tags)
			s.resourceTags[profile.ProfileVersionARN] = cloneStringMap(tags)
		}
	}
}

func (s *Service) parseListSigningJobsFilter(
	status, platformID, requestedBy string,
	isRevoked *bool,
	signatureExpiresBefore, signatureExpiresAfter *time.Time,
	jobInvoker string,
) listJobsFilter {
	return listJobsFilter{
		Status:                 strings.TrimSpace(status),
		PlatformID:             strings.TrimSpace(platformID),
		RequestedBy:            strings.TrimSpace(requestedBy),
		IsRevoked:              isRevoked,
		SignatureExpiresBefore: signatureExpiresBefore,
		SignatureExpiresAfter:  signatureExpiresAfter,
		JobInvoker:             strings.TrimSpace(jobInvoker),
	}
}

func profileARN(profileName string) string {
	return fmt.Sprintf("arn:aws:signer:%s:%s:/signing-profiles/%s", DefaultRegion, DefaultAccountID, profileName)
}

func profileVersionARN(profileName, version string) string {
	return fmt.Sprintf("%s/%s", profileARN(profileName), version)
}

func jobARN(jobID string) string {
	return fmt.Sprintf("arn:aws:signer:%s:%s:/signing-jobs/%s", DefaultRegion, DefaultAccountID, jobID)
}

func parseProfileARN(arn string) (string, bool) {
	prefix := fmt.Sprintf("arn:aws:signer:%s:%s:/signing-profiles/", DefaultRegion, DefaultAccountID)
	if !strings.HasPrefix(arn, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(arn, prefix))
	if rest == "" || strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

func parseProfileVersionARN(arn string) (string, string, bool) {
	prefix := fmt.Sprintf("arn:aws:signer:%s:%s:/signing-profiles/", DefaultRegion, DefaultAccountID)
	if !strings.HasPrefix(arn, prefix) {
		return "", "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(arn, prefix))
	parts := strings.Split(rest, "/")
	if len(parts) == 1 {
		if strings.TrimSpace(parts[0]) == "" {
			return "", "", false
		}
		return strings.TrimSpace(parts[0]), "", true
	}
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func parseJobARN(arn string) (string, bool) {
	prefix := fmt.Sprintf("arn:aws:signer:%s:%s:/signing-jobs/", DefaultRegion, DefaultAccountID)
	if !strings.HasPrefix(arn, prefix) {
		return "", false
	}
	jobID := strings.TrimSpace(strings.TrimPrefix(arn, prefix))
	if jobID == "" || strings.Contains(jobID, "/") {
		return "", false
	}
	return jobID, true
}

func parseNextToken(nextToken string, total int) (int, error) {
	trimmed := strings.TrimSpace(nextToken)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 || value > total {
		return 0, ErrInvalidParameter
	}
	return value, nil
}

func paginateItems[T any](items []T, maxResults int32, nextToken string) ([]T, string, error) {
	start, err := parseNextToken(nextToken, len(items))
	if err != nil {
		return nil, "", err
	}
	limit := len(items)
	if maxResults > 0 && int(maxResults) < limit-start {
		limit = start + int(maxResults)
	}
	if limit < start {
		limit = start
	}

	page := make([]T, 0, limit-start)
	if limit > start {
		page = append(page, items[start:limit]...)
	}
	outToken := ""
	if limit < len(items) {
		outToken = strconv.Itoa(limit)
	}
	return page, outToken, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneSigningMaterial(in *SigningMaterial) *SigningMaterial {
	if in == nil {
		return nil
	}
	return &SigningMaterial{CertificateARN: in.CertificateARN}
}

func cloneSignatureValidityPeriod(in *SignatureValidityPeriod) *SignatureValidityPeriod {
	if in == nil {
		return nil
	}
	return &SignatureValidityPeriod{
		Value: in.Value,
		Type:  in.Type,
	}
}

func cloneSigningPlatformOverrides(in *SigningPlatformOverrides) *SigningPlatformOverrides {
	if in == nil {
		return nil
	}
	out := &SigningPlatformOverrides{
		SigningImageFormat: in.SigningImageFormat,
	}
	if in.SigningConfiguration != nil {
		out.SigningConfiguration = &SigningConfigurationOverrides{
			EncryptionAlgorithm: in.SigningConfiguration.EncryptionAlgorithm,
			HashAlgorithm:       in.SigningConfiguration.HashAlgorithm,
		}
	}
	return out
}

func cloneSource(in *Source) *Source {
	if in == nil {
		return nil
	}
	out := &Source{}
	if in.S3 != nil {
		out.S3 = &S3Source{
			BucketName: in.S3.BucketName,
			Key:        in.S3.Key,
			Version:    in.S3.Version,
		}
	}
	return out
}

func cloneSigningProfile(in *SigningProfile) SigningProfile {
	if in == nil {
		return SigningProfile{}
	}
	return SigningProfile{
		ProfileName:             in.ProfileName,
		ProfileVersion:          in.ProfileVersion,
		ProfileVersionARN:       in.ProfileVersionARN,
		Arn:                     in.Arn,
		Status:                  in.Status,
		StatusReason:            in.StatusReason,
		PlatformID:              in.PlatformID,
		PlatformDisplayName:     in.PlatformDisplayName,
		SigningMaterial:         cloneSigningMaterial(in.SigningMaterial),
		SignatureValidityPeriod: cloneSignatureValidityPeriod(in.SignatureValidityPeriod),
		Overrides:               cloneSigningPlatformOverrides(in.Overrides),
		SigningParameters:       cloneStringMap(in.SigningParameters),
		Tags:                    cloneStringMap(in.Tags),
		RevocationRecord:        cloneProfileRevocationRecord(in.RevocationRecord),
		PermissionRevision:      in.PermissionRevision,
	}
}

func clonePermission(in *ProfilePermission) ProfilePermission {
	if in == nil {
		return ProfilePermission{}
	}
	return ProfilePermission{
		Action:         in.Action,
		Principal:      in.Principal,
		StatementID:    in.StatementID,
		ProfileVersion: in.ProfileVersion,
	}
}

func cloneSigningJob(in *SigningJob) SigningJob {
	if in == nil {
		return SigningJob{}
	}
	return SigningJob{
		JobID:               in.JobID,
		JobARN:              in.JobARN,
		Source:              cloneSource(in.Source),
		SignedObject:        cloneSignedObject(in.SignedObject),
		SigningMaterial:     cloneSigningMaterial(in.SigningMaterial),
		PlatformID:          in.PlatformID,
		PlatformDisplayName: in.PlatformDisplayName,
		ProfileName:         in.ProfileName,
		ProfileVersion:      in.ProfileVersion,
		Overrides:           cloneSigningPlatformOverrides(in.Overrides),
		SigningParameters:   cloneStringMap(in.SigningParameters),
		CreatedAt:           in.CreatedAt,
		CompletedAt:         cloneTimePtr(in.CompletedAt),
		SignatureExpiresAt:  cloneTimePtr(in.SignatureExpiresAt),
		RequestedBy:         in.RequestedBy,
		Status:              in.Status,
		StatusReason:        in.StatusReason,
		RevocationRecord:    cloneJobRevocationRecord(in.RevocationRecord),
		IsRevoked:           in.IsRevoked,
		JobOwner:            in.JobOwner,
		JobInvoker:          in.JobInvoker,
	}
}

func cloneSignedObject(in *SignedObject) *SignedObject {
	if in == nil {
		return nil
	}
	out := &SignedObject{}
	if in.S3 != nil {
		out.S3 = &S3SignedObject{
			BucketName: in.S3.BucketName,
			Key:        in.S3.Key,
		}
	}
	return out
}

func cloneSigningPlatform(in *SigningPlatform) SigningPlatform {
	if in == nil {
		return SigningPlatform{}
	}
	out := SigningPlatform{
		PlatformID:          in.PlatformID,
		DisplayName:         in.DisplayName,
		Partner:             in.Partner,
		Target:              in.Target,
		Category:            in.Category,
		MaxSizeInMB:         in.MaxSizeInMB,
		RevocationSupported: in.RevocationSupported,
	}
	if in.SigningConfiguration != nil {
		out.SigningConfiguration = &SigningConfiguration{
			EncryptionAlgorithmOptions: cloneEnumOptions(in.SigningConfiguration.EncryptionAlgorithmOptions),
			HashAlgorithmOptions:       cloneEnumOptions(in.SigningConfiguration.HashAlgorithmOptions),
		}
	}
	if in.SigningImageFormat != nil {
		out.SigningImageFormat = &SigningImageFormat{
			SupportedFormats: append([]string(nil), in.SigningImageFormat.SupportedFormats...),
			DefaultFormat:    in.SigningImageFormat.DefaultFormat,
		}
	}
	return out
}

func cloneEnumOptions(in *EnumOptions) *EnumOptions {
	if in == nil {
		return nil
	}
	return &EnumOptions{
		AllowedValues: append([]string(nil), in.AllowedValues...),
		DefaultValue:  in.DefaultValue,
	}
}

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	out := in.UTC()
	return &out
}

func cloneProfileRevocationRecord(in *SigningProfileRevocationRecord) *SigningProfileRevocationRecord {
	if in == nil {
		return nil
	}
	return &SigningProfileRevocationRecord{
		RevocationEffectiveFrom: in.RevocationEffectiveFrom,
		RevokedAt:               in.RevokedAt,
		RevokedBy:               in.RevokedBy,
	}
}

func cloneJobRevocationRecord(in *SigningJobRevocationRecord) *SigningJobRevocationRecord {
	if in == nil {
		return nil
	}
	return &SigningJobRevocationRecord{
		Reason:    in.Reason,
		RevokedAt: in.RevokedAt,
		RevokedBy: in.RevokedBy,
	}
}
