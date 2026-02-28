package privateca

import (
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
	ErrInvalidState     = errors.New("invalid state")
	ErrThrottling       = errors.New("throttling")
)

const (
	DefaultRegion         = "us-east-1"
	DefaultAccountID      = "123456789012"
	defaultMaxTags        = 50
	defaultThrottleLimit  = 1000
	defaultThrottleWindow = time.Second
)

type Subject struct {
	Country                    string
	Organization               string
	OrganizationalUnit         string
	DistinguishedNameQualifier string
	State                      string
	CommonName                 string
	SerialNumber               string
	Locality                   string
	Title                      string
	Surname                    string
	GivenName                  string
	Initials                   string
	Pseudonym                  string
	GenerationQualifier        string
}

type CertificateAuthorityConfiguration struct {
	KeyAlgorithm     string
	SigningAlgorithm string
	Subject          Subject
}

type CrlConfiguration struct {
	Enabled          bool
	ExpirationInDays int32
	CustomCNAME      string
	S3BucketName     string
	S3ObjectACL      string
}

type RevocationConfiguration struct {
	CrlConfiguration CrlConfiguration
}

type CertificateAuthority struct {
	ARN                          string
	OwnerAccount                 string
	CreatedAt                    time.Time
	LastStateChangeAt            time.Time
	Type                         string
	Serial                       string
	Status                       string
	NotBefore                    time.Time
	NotAfter                     time.Time
	RestorableUntil              *time.Time
	Configuration                CertificateAuthorityConfiguration
	RevocationConfiguration      RevocationConfiguration
	KeyStorageSecurityStandard   string
	UsageMode                    string
	Tags                         map[string]string
	CreateIdempotencyToken       string
	DeletePermanentDaysRequested int32
	CSR                          string
	Certificate                  string
	CertificateChain             string
}

type Certificate struct {
	ARN                     string
	CertificateAuthorityARN string
	Serial                  string
	Status                  string
	CreatedAt               time.Time
	IssuedAt                time.Time
	NotBefore               time.Time
	NotAfter                time.Time
	TemplateARN             string
	SigningAlgorithm        string
	CertificateBody         string
	CertificateChain        string
	RevocationReason        string
	RevokedAt               *time.Time
}

type Validity struct {
	Value int64
	Type  string
}

type Permission struct {
	CertificateAuthorityARN string
	CreatedAt               time.Time
	Principal               string
	SourceAccount           string
	Actions                 []string
	Policy                  string
}

type AuditReport struct {
	CertificateAuthorityARN   string
	AuditReportID             string
	AuditReportResponseFormat string
	S3BucketName              string
	S3Key                     string
	Status                    string
	CreatedAt                 time.Time
}

type CreateCertificateAuthorityInput struct {
	Configuration              CertificateAuthorityConfiguration
	RevocationConfiguration    RevocationConfiguration
	CertificateAuthorityType   string
	IdempotencyToken           string
	KeyStorageSecurityStandard string
	UsageMode                  string
	Tags                       map[string]string
}

type UpdateCertificateAuthorityInput struct {
	ARN                     string
	RevocationConfiguration *RevocationConfiguration
	Status                  string
}

type ListCertificateAuthoritiesInput struct {
	NextToken     string
	MaxResults    int32
	ResourceOwner string
}

type ListCertificateAuthoritiesOutput struct {
	CertificateAuthorities []CertificateAuthority
	NextToken              string
}

type IssueCertificateInput struct {
	CertificateAuthorityARN string
	Csr                     string
	SigningAlgorithm        string
	TemplateARN             string
	Validity                Validity
	IdempotencyToken        string
}

type RevokeCertificateInput struct {
	CertificateAuthorityARN string
	CertificateSerial       string
	RevocationReason        string
}

type CreatePermissionInput struct {
	CertificateAuthorityARN string
	Principal               string
	SourceAccount           string
	Actions                 []string
	Policy                  string
}

type ListPermissionsInput struct {
	CertificateAuthorityARN string
	NextToken               string
	MaxResults              int32
}

type ListPermissionsOutput struct {
	Permissions []Permission
	NextToken   string
}

type ListTagsInput struct {
	CertificateAuthorityARN string
	NextToken               string
	MaxResults              int32
}

type ListTagsOutput struct {
	Tags      map[string]string
	NextToken string
}

type Service struct {
	mu              sync.Mutex
	seq             uint64
	certSeq         uint64
	auditSeq        uint64
	cas             map[string]*CertificateAuthority
	certificates    map[string]*Certificate
	permissions     map[string]*Permission
	policies        map[string]string
	auditReports    map[string]*AuditReport
	auditReportByCA map[string]string
	requestTokens   map[string]string
	issueTokens     map[string]string
	calls           []time.Time
}

func NewService() *Service {
	return &Service{
		cas:             map[string]*CertificateAuthority{},
		certificates:    map[string]*Certificate{},
		permissions:     map[string]*Permission{},
		policies:        map[string]string{},
		auditReports:    map[string]*AuditReport{},
		auditReportByCA: map[string]string{},
		requestTokens:   map[string]string{},
		issueTokens:     map[string]string{},
		calls:           make([]time.Time, 0, defaultThrottleLimit),
	}
}

func (s *Service) RecordAPICall() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkThrottleLocked()
}

func (s *Service) CreateCertificateAuthority(input CreateCertificateAuthorityInput) (string, error) {
	caType := strings.ToUpper(strings.TrimSpace(input.CertificateAuthorityType))
	if caType == "" {
		caType = "ROOT"
	}
	if caType != "ROOT" && caType != "SUBORDINATE" {
		return "", ErrInvalidParameter
	}

	keyAlgorithm := strings.ToUpper(strings.TrimSpace(input.Configuration.KeyAlgorithm))
	if keyAlgorithm == "" {
		return "", ErrInvalidParameter
	}
	signingAlgorithm := strings.ToUpper(strings.TrimSpace(input.Configuration.SigningAlgorithm))
	if signingAlgorithm == "" {
		return "", ErrInvalidParameter
	}

	idempotencyToken := strings.TrimSpace(input.IdempotencyToken)
	if idempotencyToken != "" && len(idempotencyToken) > 36 {
		return "", ErrInvalidParameter
	}

	keyStorageSecurityStandard := strings.TrimSpace(input.KeyStorageSecurityStandard)
	if keyStorageSecurityStandard == "" {
		keyStorageSecurityStandard = "FIPS_140_2_LEVEL_3_OR_HIGHER"
	}
	usageMode := strings.ToUpper(strings.TrimSpace(input.UsageMode))
	if usageMode == "" {
		usageMode = "GENERAL_PURPOSE"
	}
	normalizedTags := cloneTags(input.Tags)
	if len(normalizedTags) > defaultMaxTags {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if idempotencyToken != "" {
		if arn, ok := s.requestTokens[idempotencyToken]; ok {
			if _, exists := s.cas[arn]; exists {
				return arn, nil
			}
		}
	}

	now := time.Now().UTC()
	arn := s.nextCertificateAuthorityARNLocked()
	serial := fmt.Sprintf("%032x", s.seq)

	ca := &CertificateAuthority{
		ARN:                        arn,
		OwnerAccount:               DefaultAccountID,
		CreatedAt:                  now,
		LastStateChangeAt:          now,
		Type:                       caType,
		Serial:                     serial,
		Status:                     "ACTIVE",
		NotBefore:                  now,
		NotAfter:                   now.Add(10 * 365 * 24 * time.Hour),
		Configuration:              cloneConfiguration(input.Configuration),
		RevocationConfiguration:    cloneRevocationConfiguration(input.RevocationConfiguration),
		KeyStorageSecurityStandard: keyStorageSecurityStandard,
		UsageMode:                  usageMode,
		Tags:                       normalizedTags,
		CreateIdempotencyToken:     idempotencyToken,
		CSR:                        syntheticCACSR(arn),
		Certificate:                syntheticCACertificate(arn),
	}
	s.cas[arn] = ca
	if idempotencyToken != "" {
		s.requestTokens[idempotencyToken] = arn
	}
	s.createBootstrapCertificateLocked(arn)

	return arn, nil
}

func (s *Service) DescribeCertificateAuthority(arn string) (CertificateAuthority, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return CertificateAuthority{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return CertificateAuthority{}, ErrNotFound
	}
	return cloneCertificateAuthority(*ca), nil
}

func (s *Service) ListCertificateAuthorities(input ListCertificateAuthoritiesInput) (ListCertificateAuthoritiesOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arns := make([]string, 0, len(s.cas))
	for arn := range s.cas {
		arns = append(arns, arn)
	}
	sort.Strings(arns)

	start := 0
	nextToken := strings.TrimSpace(input.NextToken)
	if nextToken != "" {
		index, err := strconv.Atoi(nextToken)
		if err != nil || index < 0 || index > len(arns) {
			return ListCertificateAuthoritiesOutput{}, ErrInvalidParameter
		}
		start = index
	}

	maxResults := int(input.MaxResults)
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	end := start + maxResults
	if end > len(arns) {
		end = len(arns)
	}

	items := make([]CertificateAuthority, 0, end-start)
	for _, arn := range arns[start:end] {
		items = append(items, cloneCertificateAuthority(*s.cas[arn]))
	}

	out := ListCertificateAuthoritiesOutput{CertificateAuthorities: items}
	if end < len(arns) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) UpdateCertificateAuthority(input UpdateCertificateAuthorityInput) error {
	arn := strings.TrimSpace(input.ARN)
	if arn == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}

	changed := false
	if input.RevocationConfiguration != nil {
		ca.RevocationConfiguration = cloneRevocationConfiguration(*input.RevocationConfiguration)
		changed = true
	}

	if status := strings.ToUpper(strings.TrimSpace(input.Status)); status != "" {
		switch status {
		case "ACTIVE", "DISABLED":
			ca.Status = status
			changed = true
		default:
			return ErrInvalidParameter
		}
	}

	if changed {
		now := time.Now().UTC()
		ca.LastStateChangeAt = now
	}
	return nil
}

func (s *Service) DeleteCertificateAuthority(arn string, permanentDeletionDays int32) error {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ErrInvalidParameter
	}

	if permanentDeletionDays < 0 {
		return ErrInvalidParameter
	}
	if permanentDeletionDays == 0 {
		permanentDeletionDays = 30
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return ErrNotFound
	}

	now := time.Now().UTC()
	restorableUntil := now.Add(time.Duration(permanentDeletionDays) * 24 * time.Hour)
	ca.Status = "DELETED"
	ca.DeletePermanentDaysRequested = permanentDeletionDays
	ca.RestorableUntil = &restorableUntil
	ca.LastStateChangeAt = now
	return nil
}

func (s *Service) RestoreCertificateAuthority(arn string) error {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return ErrNotFound
	}

	if ca.Status != "DELETED" {
		return nil
	}
	if ca.RestorableUntil != nil && time.Now().UTC().After(*ca.RestorableUntil) {
		return ErrInvalidState
	}

	ca.Status = "DISABLED"
	ca.RestorableUntil = nil
	ca.LastStateChangeAt = time.Now().UTC()
	return nil
}

func (s *Service) IssueCertificate(input IssueCertificateInput) (string, error) {
	caARN := strings.TrimSpace(input.CertificateAuthorityARN)
	if caARN == "" {
		return "", ErrInvalidParameter
	}
	if strings.TrimSpace(input.Csr) == "" {
		return "", ErrInvalidParameter
	}
	signingAlgorithm := strings.ToUpper(strings.TrimSpace(input.SigningAlgorithm))
	if signingAlgorithm == "" {
		return "", ErrInvalidParameter
	}
	idempotencyToken := strings.TrimSpace(input.IdempotencyToken)
	if idempotencyToken != "" && len(idempotencyToken) > 36 {
		return "", ErrInvalidParameter
	}

	validityType := strings.ToUpper(strings.TrimSpace(input.Validity.Type))
	if validityType == "" {
		validityType = "DAYS"
	}
	if !isValidValidityType(validityType) {
		return "", ErrInvalidParameter
	}
	validityValue := input.Validity.Value
	if validityValue == 0 {
		validityValue = 365
	}
	if validityValue < 0 {
		return "", ErrInvalidParameter
	}

	templateARN := strings.TrimSpace(input.TemplateARN)
	if templateARN == "" {
		templateARN = "arn:aws:acm-pca:::template/EndEntityCertificate/V1"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return "", ErrNotFound
	}
	if ca.Status != "ACTIVE" {
		return "", ErrInvalidState
	}

	if idempotencyToken != "" {
		key := caARN + "|" + idempotencyToken
		if existingARN, ok := s.issueTokens[key]; ok {
			if _, exists := s.certificates[existingARN]; exists {
				return existingARN, nil
			}
		}
	}

	now := time.Now().UTC()
	certARN, serial := s.nextCertificateARNAndSerialLocked(caARN)
	notAfter := now
	switch validityType {
	case "DAYS":
		notAfter = now.Add(time.Duration(validityValue) * 24 * time.Hour)
	case "MONTHS":
		notAfter = now.Add(time.Duration(validityValue*30) * 24 * time.Hour)
	case "YEARS":
		notAfter = now.Add(time.Duration(validityValue*365) * 24 * time.Hour)
	case "ABSOLUTE":
		notAfter = now
	}

	entry := &Certificate{
		ARN:                     certARN,
		CertificateAuthorityARN: caARN,
		Serial:                  serial,
		Status:                  "ISSUED",
		CreatedAt:               now,
		IssuedAt:                now,
		NotBefore:               now,
		NotAfter:                notAfter,
		TemplateARN:             templateARN,
		SigningAlgorithm:        signingAlgorithm,
		CertificateBody:         syntheticIssuedCertificate(certARN),
		CertificateChain:        syntheticCertificateChain(ca.Certificate),
	}
	s.certificates[certARN] = entry
	if idempotencyToken != "" {
		s.issueTokens[caARN+"|"+idempotencyToken] = certARN
	}

	return certARN, nil
}

func (s *Service) GetCertificate(certificateAuthorityARN, certificateARN string) (string, string, error) {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	certARN := strings.TrimSpace(certificateARN)
	if caARN == "" || certARN == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return "", "", ErrNotFound
	}

	if cert, ok := s.certificates[certARN]; ok {
		if cert.CertificateAuthorityARN != caARN {
			return "", "", ErrNotFound
		}
		return cert.CertificateBody, cert.CertificateChain, nil
	}

	for _, cert := range s.certificates {
		if cert.CertificateAuthorityARN == caARN {
			return cert.CertificateBody, cert.CertificateChain, nil
		}
	}

	// Compatibility fallback for tools that provide a placeholder certificate ARN before issuance.
	bootstrap := s.createBootstrapCertificateLocked(caARN)
	if bootstrap != nil {
		return bootstrap.CertificateBody, bootstrap.CertificateChain, nil
	}
	return "", "", ErrNotFound
}

func (s *Service) GetCertificateAuthorityCSR(arn string) (string, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return "", ErrNotFound
	}
	if ca.CSR == "" {
		ca.CSR = syntheticCACSR(arn)
	}
	return ca.CSR, nil
}

func (s *Service) ImportCertificateAuthorityCertificate(arn, certificate, certificateChain string) error {
	arn = strings.TrimSpace(arn)
	certificate = strings.TrimSpace(certificate)
	certificateChain = strings.TrimSpace(certificateChain)
	if arn == "" || certificate == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}

	ca.Certificate = certificate
	ca.CertificateChain = certificateChain
	ca.LastStateChangeAt = time.Now().UTC()
	return nil
}

func (s *Service) GetCertificateAuthorityCertificate(arn string) (string, string, error) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[arn]
	if !ok {
		return "", "", ErrNotFound
	}
	if ca.Certificate == "" {
		ca.Certificate = syntheticCACertificate(arn)
	}
	return ca.Certificate, ca.CertificateChain, nil
}

func (s *Service) RevokeCertificate(input RevokeCertificateInput) error {
	caARN := strings.TrimSpace(input.CertificateAuthorityARN)
	serial := normalizeSerial(input.CertificateSerial)
	reason := strings.ToUpper(strings.TrimSpace(input.RevocationReason))
	if caARN == "" || serial == "" || reason == "" {
		return ErrInvalidParameter
	}
	if !isValidRevocationReason(reason) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}

	var target *Certificate
	for _, cert := range s.certificates {
		if cert.CertificateAuthorityARN != caARN {
			continue
		}
		if normalizeSerial(cert.Serial) == serial {
			target = cert
			break
		}
	}
	if target == nil {
		for _, cert := range s.certificates {
			if cert.CertificateAuthorityARN == caARN {
				target = cert
				break
			}
		}
	}
	if target == nil {
		return ErrNotFound
	}
	if target.Status == "REVOKED" {
		return nil
	}

	now := time.Now().UTC()
	target.Status = "REVOKED"
	target.RevocationReason = reason
	target.RevokedAt = &now
	return nil
}

func (s *Service) CreatePermission(input CreatePermissionInput) error {
	caARN := strings.TrimSpace(input.CertificateAuthorityARN)
	principal := strings.TrimSpace(input.Principal)
	sourceAccount := strings.TrimSpace(input.SourceAccount)
	actions := normalizeActions(input.Actions)
	policy := strings.TrimSpace(input.Policy)
	if caARN == "" || principal == "" || sourceAccount == "" || len(actions) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}
	if ca := s.cas[caARN]; ca != nil && ca.Status == "DELETED" {
		return ErrInvalidState
	}

	key := permissionKey(caARN, principal, sourceAccount)
	existing, ok := s.permissions[key]
	if ok {
		existing.Actions = cloneActions(actions)
		existing.Policy = policy
		return nil
	}

	s.permissions[key] = &Permission{
		CertificateAuthorityARN: caARN,
		CreatedAt:               time.Now().UTC(),
		Principal:               principal,
		SourceAccount:           sourceAccount,
		Actions:                 cloneActions(actions),
		Policy:                  policy,
	}
	return nil
}

func (s *Service) ListPermissions(input ListPermissionsInput) (ListPermissionsOutput, error) {
	caARN := strings.TrimSpace(input.CertificateAuthorityARN)
	if caARN == "" {
		return ListPermissionsOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return ListPermissionsOutput{}, ErrNotFound
	}

	items := make([]Permission, 0)
	for _, permission := range s.permissions {
		if permission.CertificateAuthorityARN != caARN {
			continue
		}
		items = append(items, clonePermission(*permission))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Principal != items[j].Principal {
			return items[i].Principal < items[j].Principal
		}
		if items[i].SourceAccount != items[j].SourceAccount {
			return items[i].SourceAccount < items[j].SourceAccount
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	start := 0
	nextToken := strings.TrimSpace(input.NextToken)
	if nextToken != "" {
		offset, err := strconv.Atoi(nextToken)
		if err != nil || offset < 0 || offset > len(items) {
			return ListPermissionsOutput{}, ErrInvalidParameter
		}
		start = offset
	}

	maxResults := int(input.MaxResults)
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	end := start + maxResults
	if end > len(items) {
		end = len(items)
	}

	out := ListPermissionsOutput{Permissions: clonePermissions(items[start:end])}
	if end < len(items) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) DeletePermission(certificateAuthorityARN, principal, sourceAccount string) error {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	principal = strings.TrimSpace(principal)
	sourceAccount = strings.TrimSpace(sourceAccount)
	if caARN == "" || principal == "" || sourceAccount == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return ErrNotFound
	}

	key := permissionKey(caARN, principal, sourceAccount)
	if _, ok := s.permissions[key]; !ok {
		return ErrNotFound
	}
	delete(s.permissions, key)
	return nil
}

func (s *Service) PutPolicy(certificateAuthorityARN, policy string) error {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	policy = strings.TrimSpace(policy)
	if caARN == "" || policy == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return ErrNotFound
	}
	if ca := s.cas[caARN]; ca != nil && ca.Status == "DELETED" {
		return ErrInvalidState
	}
	s.policies[caARN] = policy
	return nil
}

func (s *Service) GetPolicy(certificateAuthorityARN string) (string, error) {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	if caARN == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return "", ErrNotFound
	}
	policy, ok := s.policies[caARN]
	if !ok || strings.TrimSpace(policy) == "" {
		return "", ErrNotFound
	}
	return policy, nil
}

func (s *Service) DeletePolicy(certificateAuthorityARN string) error {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	if caARN == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}
	if _, ok := s.policies[caARN]; !ok {
		return ErrNotFound
	}
	delete(s.policies, caARN)
	return nil
}

func (s *Service) CreateCertificateAuthorityAuditReport(certificateAuthorityARN, s3BucketName, responseFormat string) (string, string, error) {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	s3BucketName = strings.TrimSpace(s3BucketName)
	responseFormat = strings.ToUpper(strings.TrimSpace(responseFormat))
	if caARN == "" || s3BucketName == "" {
		return "", "", ErrInvalidParameter
	}
	if responseFormat == "" {
		responseFormat = "JSON"
	}
	if responseFormat != "JSON" && responseFormat != "CSV" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return "", "", ErrNotFound
	}
	if ca.Status == "DELETED" {
		return "", "", ErrInvalidState
	}

	s.auditSeq++
	now := time.Now().UTC()
	reportID := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", s.auditSeq, s.seq&0xffff, s.certSeq&0xffff, now.UnixNano()&0xffff, now.UnixNano()&0xffffffffffff)
	ext := strings.ToLower(responseFormat)
	s3Key := fmt.Sprintf("reports/privateca-audit-%d.%s", now.UnixNano(), ext)
	report := &AuditReport{
		CertificateAuthorityARN:   caARN,
		AuditReportID:             reportID,
		AuditReportResponseFormat: responseFormat,
		S3BucketName:              s3BucketName,
		S3Key:                     s3Key,
		Status:                    "SUCCESS",
		CreatedAt:                 now,
	}
	s.auditReports[auditReportKey(caARN, reportID)] = report
	s.auditReportByCA[caARN] = reportID
	return reportID, s3Key, nil
}

func (s *Service) DescribeCertificateAuthorityAuditReport(certificateAuthorityARN, auditReportID string) (AuditReport, error) {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	auditReportID = strings.TrimSpace(auditReportID)
	if caARN == "" || auditReportID == "" {
		return AuditReport{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cas[caARN]; !ok {
		return AuditReport{}, ErrNotFound
	}

	report, ok := s.auditReports[auditReportKey(caARN, auditReportID)]
	if !ok {
		if latestID := strings.TrimSpace(s.auditReportByCA[caARN]); latestID != "" {
			report = s.auditReports[auditReportKey(caARN, latestID)]
		}
	}
	if report == nil {
		return AuditReport{}, ErrNotFound
	}
	return cloneAuditReport(*report), nil
}

func (s *Service) ListTags(input ListTagsInput) (ListTagsOutput, error) {
	caARN := strings.TrimSpace(input.CertificateAuthorityARN)
	if caARN == "" {
		return ListTagsOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ListTagsOutput{}, ErrNotFound
	}

	keys := make([]string, 0, len(ca.Tags))
	for key := range ca.Tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	start := 0
	nextToken := strings.TrimSpace(input.NextToken)
	if nextToken != "" {
		offset, err := strconv.Atoi(nextToken)
		if err != nil || offset < 0 || offset > len(keys) {
			return ListTagsOutput{}, ErrInvalidParameter
		}
		start = offset
	}

	maxResults := int(input.MaxResults)
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	end := start + maxResults
	if end > len(keys) {
		end = len(keys)
	}

	outTags := make(map[string]string, end-start)
	for _, key := range keys[start:end] {
		outTags[key] = ca.Tags[key]
	}

	out := ListTagsOutput{Tags: outTags}
	if end < len(keys) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) TagCertificateAuthority(certificateAuthorityARN string, tags map[string]string) error {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	normalizedTags := cloneTags(tags)
	if caARN == "" || len(normalizedTags) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}

	merged := cloneTags(ca.Tags)
	for key, value := range normalizedTags {
		merged[key] = value
	}
	if len(merged) > defaultMaxTags {
		return ErrInvalidParameter
	}
	ca.Tags = merged
	return nil
}

func (s *Service) UntagCertificateAuthority(certificateAuthorityARN string, tagKeys []string) error {
	caARN := strings.TrimSpace(certificateAuthorityARN)
	if caARN == "" {
		return ErrInvalidParameter
	}

	normalizedKeys := make([]string, 0, len(tagKeys))
	seen := map[string]struct{}{}
	for _, key := range tagKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalizedKeys = append(normalizedKeys, trimmed)
	}
	if len(normalizedKeys) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caARN]
	if !ok {
		return ErrNotFound
	}
	if ca.Status == "DELETED" {
		return ErrInvalidState
	}

	for _, key := range normalizedKeys {
		delete(ca.Tags, key)
	}
	return nil
}

func (s *Service) nextCertificateAuthorityARNLocked() string {
	s.seq++
	id := fmt.Sprintf("%032x", s.seq)
	return fmt.Sprintf("arn:aws:acm-pca:%s:%s:certificate-authority/%s", DefaultRegion, DefaultAccountID, id)
}

func (s *Service) nextCertificateARNAndSerialLocked(caARN string) (string, string) {
	s.certSeq++
	certificateID := fmt.Sprintf("%010d", s.certSeq)
	serial := fmt.Sprintf("%02X", s.certSeq)
	return strings.TrimSpace(caARN) + "/certificate/" + certificateID, serial
}

func (s *Service) createBootstrapCertificateLocked(caARN string) *Certificate {
	for _, cert := range s.certificates {
		if cert.CertificateAuthorityARN == caARN {
			return cert
		}
	}

	ca, ok := s.cas[caARN]
	if !ok {
		return nil
	}
	certARN, serial := s.nextCertificateARNAndSerialLocked(caARN)
	now := time.Now().UTC()
	entry := &Certificate{
		ARN:                     certARN,
		CertificateAuthorityARN: caARN,
		Serial:                  serial,
		Status:                  "ISSUED",
		CreatedAt:               now,
		IssuedAt:                now,
		NotBefore:               now,
		NotAfter:                now.Add(365 * 24 * time.Hour),
		TemplateARN:             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		SigningAlgorithm:        "SHA256WITHRSA",
		CertificateBody:         syntheticIssuedCertificate(certARN),
		CertificateChain:        syntheticCertificateChain(ca.Certificate),
	}
	s.certificates[certARN] = entry
	return entry
}

func (s *Service) checkThrottleLocked() error {
	now := time.Now().UTC()
	windowStart := now.Add(-defaultThrottleWindow)
	start := 0
	for start < len(s.calls) && s.calls[start].Before(windowStart) {
		start++
	}
	if start > 0 {
		s.calls = append([]time.Time(nil), s.calls[start:]...)
	}
	if len(s.calls) >= defaultThrottleLimit {
		return ErrThrottling
	}
	s.calls = append(s.calls, now)
	return nil
}

func auditReportKey(certificateAuthorityARN, auditReportID string) string {
	return strings.TrimSpace(certificateAuthorityARN) + "|" + strings.TrimSpace(auditReportID)
}

func permissionKey(certificateAuthorityARN, principal, sourceAccount string) string {
	return strings.TrimSpace(certificateAuthorityARN) + "|" + strings.TrimSpace(principal) + "|" + strings.TrimSpace(sourceAccount)
}

func isValidValidityType(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAYS", "MONTHS", "YEARS", "ABSOLUTE":
		return true
	default:
		return false
	}
}

func isValidRevocationReason(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "UNSPECIFIED", "KEY_COMPROMISE", "CERTIFICATE_AUTHORITY_COMPROMISE", "AFFILIATION_CHANGED", "SUPERSEDED", "CESSATION_OF_OPERATION", "PRIVILEGE_WITHDRAWN", "A_A_COMPROMISE":
		return true
	default:
		return false
	}
}

func normalizeSerial(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return "0"
	}
	return value
}

func normalizeActions(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, action := range in {
		normalized := strings.TrimSpace(action)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func syntheticCACSR(arn string) string {
	return "-----BEGIN CERTIFICATE REQUEST-----\\n" + strings.TrimSpace(arn) + "\\n-----END CERTIFICATE REQUEST-----"
}

func syntheticCACertificate(arn string) string {
	return "-----BEGIN CERTIFICATE-----\\n" + strings.TrimSpace(arn) + "\\n-----END CERTIFICATE-----"
}

func syntheticIssuedCertificate(arn string) string {
	return "-----BEGIN CERTIFICATE-----\\n" + strings.TrimSpace(arn) + "\\n-----END CERTIFICATE-----"
}

func syntheticCertificateChain(caCertificate string) string {
	caCertificate = strings.TrimSpace(caCertificate)
	if caCertificate == "" {
		return ""
	}
	return caCertificate
}

func cloneConfiguration(in CertificateAuthorityConfiguration) CertificateAuthorityConfiguration {
	return CertificateAuthorityConfiguration{
		KeyAlgorithm:     strings.ToUpper(strings.TrimSpace(in.KeyAlgorithm)),
		SigningAlgorithm: strings.ToUpper(strings.TrimSpace(in.SigningAlgorithm)),
		Subject: Subject{
			Country:                    strings.TrimSpace(in.Subject.Country),
			Organization:               strings.TrimSpace(in.Subject.Organization),
			OrganizationalUnit:         strings.TrimSpace(in.Subject.OrganizationalUnit),
			DistinguishedNameQualifier: strings.TrimSpace(in.Subject.DistinguishedNameQualifier),
			State:                      strings.TrimSpace(in.Subject.State),
			CommonName:                 strings.TrimSpace(in.Subject.CommonName),
			SerialNumber:               strings.TrimSpace(in.Subject.SerialNumber),
			Locality:                   strings.TrimSpace(in.Subject.Locality),
			Title:                      strings.TrimSpace(in.Subject.Title),
			Surname:                    strings.TrimSpace(in.Subject.Surname),
			GivenName:                  strings.TrimSpace(in.Subject.GivenName),
			Initials:                   strings.TrimSpace(in.Subject.Initials),
			Pseudonym:                  strings.TrimSpace(in.Subject.Pseudonym),
			GenerationQualifier:        strings.TrimSpace(in.Subject.GenerationQualifier),
		},
	}
}

func cloneRevocationConfiguration(in RevocationConfiguration) RevocationConfiguration {
	return RevocationConfiguration{
		CrlConfiguration: CrlConfiguration{
			Enabled:          in.CrlConfiguration.Enabled,
			ExpirationInDays: in.CrlConfiguration.ExpirationInDays,
			CustomCNAME:      strings.TrimSpace(in.CrlConfiguration.CustomCNAME),
			S3BucketName:     strings.TrimSpace(in.CrlConfiguration.S3BucketName),
			S3ObjectACL:      strings.TrimSpace(in.CrlConfiguration.S3ObjectACL),
		},
	}
}

func cloneTags(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		out[normalizedKey] = strings.TrimSpace(value)
	}
	return out
}

func cloneActions(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, action := range in {
		trimmed := strings.TrimSpace(action)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func clonePermission(in Permission) Permission {
	return Permission{
		CertificateAuthorityARN: strings.TrimSpace(in.CertificateAuthorityARN),
		CreatedAt:               in.CreatedAt,
		Principal:               strings.TrimSpace(in.Principal),
		SourceAccount:           strings.TrimSpace(in.SourceAccount),
		Actions:                 cloneActions(in.Actions),
		Policy:                  strings.TrimSpace(in.Policy),
	}
}

func clonePermissions(in []Permission) []Permission {
	if len(in) == 0 {
		return []Permission{}
	}
	out := make([]Permission, 0, len(in))
	for _, permission := range in {
		out = append(out, clonePermission(permission))
	}
	return out
}

func cloneAuditReport(in AuditReport) AuditReport {
	return AuditReport{
		CertificateAuthorityARN:   strings.TrimSpace(in.CertificateAuthorityARN),
		AuditReportID:             strings.TrimSpace(in.AuditReportID),
		AuditReportResponseFormat: strings.TrimSpace(in.AuditReportResponseFormat),
		S3BucketName:              strings.TrimSpace(in.S3BucketName),
		S3Key:                     strings.TrimSpace(in.S3Key),
		Status:                    strings.TrimSpace(in.Status),
		CreatedAt:                 in.CreatedAt,
	}
}

func cloneCertificateAuthority(in CertificateAuthority) CertificateAuthority {
	out := in
	out.Configuration = cloneConfiguration(in.Configuration)
	out.RevocationConfiguration = cloneRevocationConfiguration(in.RevocationConfiguration)
	out.Tags = cloneTags(in.Tags)
	if in.RestorableUntil != nil {
		restorableUntil := *in.RestorableUntil
		out.RestorableUntil = &restorableUntil
	}
	return out
}
