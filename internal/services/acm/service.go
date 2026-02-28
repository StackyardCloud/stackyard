package acm

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
	ErrInvalidState     = errors.New("invalid state")
	ErrThrottling       = errors.New("throttling")
	ErrLimitExceeded    = errors.New("limit exceeded")
)

const (
	DefaultRegion          = "us-east-1"
	DefaultAccountID       = "123456789012"
	defaultMaxTags         = 50
	defaultThrottleLimit   = 250
	defaultThrottleWindow  = time.Second
	defaultRenewalDays     = 365
	defaultNotAfterDays    = 365
	defaultExpiryEventDays = int32(45)
)

type CertificateOptions struct {
	CertificateTransparencyLoggingPreference string
}

type DomainValidation struct {
	DomainName       string
	ValidationDomain string
	ValidationMethod string
	ValidationStatus string
	ResourceRecord   ResourceRecord
}

type ResourceRecord struct {
	Name  string
	Type  string
	Value string
}

type RenewalSummary struct {
	RenewalStatus           string
	UpdatedAt               time.Time
	DomainValidationOptions []DomainValidation
}

type ExpiryEventsConfiguration struct {
	DaysBeforeExpiry int32
}

type AccountConfiguration struct {
	ExpiryEvents ExpiryEventsConfiguration
}

type Certificate struct {
	CertificateArn          string
	DomainName              string
	SubjectAlternativeNames []string
	Status                  string
	Type                    string
	ValidationMethod        string
	CreatedAt               time.Time
	IssuedAt                time.Time
	NotBefore               time.Time
	NotAfter                time.Time
	InUseBy                 []string
	FailureReason           string
	RevocationReason        string
	Options                 CertificateOptions
	DomainValidationOptions []DomainValidation
	RenewalSummary          RenewalSummary
	CertificateBody         string
	CertificateChain        string
	PrivateKey              string
	Tags                    map[string]string
	LastValidationEmailAt   time.Time
}

type CertificateSummary struct {
	CertificateArn                   string
	DomainName                       string
	Status                           string
	Type                             string
	HasAdditionalSubjectAlternatives bool
}

type Service struct {
	mu            sync.Mutex
	seq           uint64
	certificates  map[string]*Certificate
	requestTokens map[string]string
	accountConfig AccountConfiguration
	calls         []time.Time
}

func NewService() *Service {
	return &Service{
		certificates:  map[string]*Certificate{},
		requestTokens: map[string]string{},
		accountConfig: AccountConfiguration{
			ExpiryEvents: ExpiryEventsConfiguration{DaysBeforeExpiry: defaultExpiryEventDays},
		},
		calls: make([]time.Time, 0, defaultThrottleLimit),
	}
}

func (s *Service) RequestCertificate(
	domainName string,
	subjectAlternativeNames []string,
	idempotencyToken string,
	validationMethod string,
	options CertificateOptions,
	domainValidationOptions []DomainValidation,
	tags map[string]string,
) (string, error) {
	domainName = strings.TrimSpace(strings.ToLower(domainName))
	if !isValidDomainName(domainName) {
		return "", ErrInvalidParameter
	}
	if validationMethod == "" {
		validationMethod = "DNS"
	}
	validationMethod = strings.ToUpper(strings.TrimSpace(validationMethod))
	if validationMethod != "DNS" && validationMethod != "EMAIL" {
		return "", ErrInvalidParameter
	}

	tokenKey := strings.TrimSpace(idempotencyToken)
	if tokenKey != "" {
		if len(tokenKey) < 1 || len(tokenKey) > 32 {
			return "", ErrInvalidParameter
		}
	}

	preference := strings.ToUpper(strings.TrimSpace(options.CertificateTransparencyLoggingPreference))
	if preference == "" {
		preference = "ENABLED"
	}
	if preference != "ENABLED" && preference != "DISABLED" {
		return "", ErrInvalidParameter
	}
	options.CertificateTransparencyLoggingPreference = preference

	normalizedTags := normalizeTags(tags)
	if len(normalizedTags) > defaultMaxTags {
		return "", ErrLimitExceeded
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return "", err
	}

	if tokenKey != "" {
		if existingARN, ok := s.requestTokens[tokenKey]; ok {
			if _, exists := s.certificates[existingARN]; exists {
				return existingARN, nil
			}
		}
	}

	now := time.Now().UTC()
	arn := s.nextCertificateARNLocked()
	entry := &Certificate{
		CertificateArn:          arn,
		DomainName:              domainName,
		SubjectAlternativeNames: dedupeNonEmptyStrings(subjectAlternativeNames),
		Status:                  "PENDING_VALIDATION",
		Type:                    "AMAZON_ISSUED",
		ValidationMethod:        validationMethod,
		CreatedAt:               now,
		NotBefore:               now,
		NotAfter:                now.Add(defaultNotAfterDays * 24 * time.Hour),
		Options:                 options,
		DomainValidationOptions: normalizeDomainValidationOptions(domainName, validationMethod, domainValidationOptions),
		Tags:                    normalizedTags,
	}
	s.certificates[arn] = entry
	if tokenKey != "" {
		s.requestTokens[tokenKey] = arn
	}
	return arn, nil
}

func (s *Service) ImportCertificate(
	certificateArn string,
	certificateBody string,
	privateKey string,
	certificateChain string,
	tags map[string]string,
) (string, error) {
	certificateBody = strings.TrimSpace(certificateBody)
	privateKey = strings.TrimSpace(privateKey)
	if certificateBody == "" || privateKey == "" {
		return "", ErrInvalidParameter
	}

	normalizedTags := normalizeTags(tags)
	if len(normalizedTags) > defaultMaxTags {
		return "", ErrLimitExceeded
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	arn := strings.TrimSpace(certificateArn)
	if arn == "" {
		arn = s.nextCertificateARNLocked()
	} else if !isValidCertificateARN(arn) {
		return "", ErrInvalidParameter
	}

	domainName := importedDomainNameForARN(arn)
	entry := &Certificate{
		CertificateArn:   arn,
		DomainName:       domainName,
		Status:           "ISSUED",
		Type:             "IMPORTED",
		ValidationMethod: "NONE",
		CreatedAt:        now,
		IssuedAt:         now,
		NotBefore:        now,
		NotAfter:         now.Add(defaultNotAfterDays * 24 * time.Hour),
		Options: CertificateOptions{
			CertificateTransparencyLoggingPreference: "ENABLED",
		},
		CertificateBody:  certificateBody,
		CertificateChain: strings.TrimSpace(certificateChain),
		PrivateKey:       privateKey,
		Tags:             normalizedTags,
	}
	s.certificates[arn] = entry
	return arn, nil
}

func (s *Service) DescribeCertificate(certificateArn string) (Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return Certificate{}, err
	}
	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return Certificate{}, ErrNotFound
	}
	return cloneCertificate(*entry), nil
}

func (s *Service) ListCertificates(nextToken string, maxItems int32, statuses []string) ([]CertificateSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return nil, "", err
	}

	if maxItems == 0 {
		maxItems = 100
	}
	if maxItems < 0 || maxItems > 1000 {
		return nil, "", ErrInvalidParameter
	}

	start, err := parseNextTokenOffset(nextToken)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}

	statusSet := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		normalized := strings.TrimSpace(strings.ToUpper(status))
		if normalized == "" {
			continue
		}
		if !isValidCertificateStatus(normalized) {
			return nil, "", ErrInvalidParameter
		}
		statusSet[normalized] = struct{}{}
	}

	arns := make([]string, 0, len(s.certificates))
	for arn, cert := range s.certificates {
		if len(statusSet) != 0 {
			if _, ok := statusSet[strings.ToUpper(cert.Status)]; !ok {
				continue
			}
		}
		arns = append(arns, arn)
	}
	sort.Strings(arns)

	if start >= len(arns) {
		return []CertificateSummary{}, "", nil
	}

	end := start + int(maxItems)
	if end > len(arns) {
		end = len(arns)
	}

	out := make([]CertificateSummary, 0, end-start)
	for _, arn := range arns[start:end] {
		cert := s.certificates[arn]
		out = append(out, CertificateSummary{
			CertificateArn:                   cert.CertificateArn,
			DomainName:                       cert.DomainName,
			Status:                           cert.Status,
			Type:                             cert.Type,
			HasAdditionalSubjectAlternatives: len(cert.SubjectAlternativeNames) > 0,
		})
	}

	if end < len(arns) {
		return out, fmt.Sprintf("%d", end), nil
	}
	return out, "", nil
}

func (s *Service) DeleteCertificate(certificateArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	certificateArn = strings.TrimSpace(certificateArn)
	entry, ok := s.certificates[certificateArn]
	if !ok {
		return ErrNotFound
	}
	if len(entry.InUseBy) != 0 {
		return ErrInvalidState
	}

	delete(s.certificates, certificateArn)
	for token, arn := range s.requestTokens {
		if arn == certificateArn {
			delete(s.requestTokens, token)
		}
	}
	return nil
}

func (s *Service) GetCertificate(certificateArn string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return "", "", err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return "", "", ErrNotFound
	}

	if entry.Status == "REVOKED" {
		return "", "", ErrInvalidState
	}
	if entry.CertificateBody == "" {
		entry.CertificateBody = syntheticCertificatePEM(entry.DomainName)
	}
	if entry.CertificateChain == "" {
		entry.CertificateChain = syntheticCertificateChainPEM(entry.DomainName)
	}
	if entry.Status == "PENDING_VALIDATION" {
		entry.Status = "ISSUED"
		entry.IssuedAt = time.Now().UTC()
	}
	return entry.CertificateBody, entry.CertificateChain, nil
}

func (s *Service) ExportCertificate(certificateArn string, passphrase string) (string, string, string, error) {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" || len(passphrase) < 4 {
		return "", "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return "", "", "", err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return "", "", "", ErrNotFound
	}
	if entry.Status == "REVOKED" {
		return "", "", "", ErrInvalidState
	}
	if entry.Type != "IMPORTED" {
		return "", "", "", ErrInvalidState
	}
	if entry.CertificateBody == "" {
		entry.CertificateBody = syntheticCertificatePEM(entry.DomainName)
	}
	if entry.CertificateChain == "" {
		entry.CertificateChain = syntheticCertificateChainPEM(entry.DomainName)
	}
	if entry.PrivateKey == "" {
		entry.PrivateKey = syntheticPrivateKeyPEM(entry.DomainName)
	}
	if entry.Status == "PENDING_VALIDATION" {
		entry.Status = "ISSUED"
		entry.IssuedAt = time.Now().UTC()
	}
	return entry.CertificateBody, entry.CertificateChain, entry.PrivateKey, nil
}

func (s *Service) RenewCertificate(certificateArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	if entry.Status == "REVOKED" {
		return ErrInvalidState
	}

	now := time.Now().UTC()
	entry.Status = "ISSUED"
	entry.IssuedAt = now
	entry.NotAfter = now.Add(defaultRenewalDays * 24 * time.Hour)
	entry.RenewalSummary = RenewalSummary{
		RenewalStatus:           "SUCCESS",
		UpdatedAt:               now,
		DomainValidationOptions: cloneDomainValidations(entry.DomainValidationOptions),
	}
	return nil
}

func (s *Service) RevokeCertificate(certificateArn, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "UNSPECIFIED"
	}
	if !isValidRevocationReason(reason) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	if entry.Status == "REVOKED" {
		return ErrInvalidState
	}

	entry.Status = "REVOKED"
	entry.RevocationReason = reason
	entry.FailureReason = "REVOKED"
	return nil
}

func (s *Service) ResendValidationEmail(certificateArn, domain, validationDomain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	if entry.Status != "PENDING_VALIDATION" {
		return ErrInvalidState
	}
	if entry.ValidationMethod != "EMAIL" {
		return ErrInvalidState
	}

	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain != "" && domain != strings.ToLower(entry.DomainName) {
		return ErrInvalidParameter
	}

	validationDomain = strings.TrimSpace(strings.ToLower(validationDomain))
	if validationDomain != "" {
		matched := false
		for _, option := range entry.DomainValidationOptions {
			if strings.EqualFold(option.ValidationDomain, validationDomain) {
				matched = true
				break
			}
		}
		if !matched {
			return ErrInvalidParameter
		}
	}

	entry.LastValidationEmailAt = time.Now().UTC()
	return nil
}

func (s *Service) UpdateCertificateOptions(certificateArn string, options CertificateOptions) error {
	preference := strings.TrimSpace(strings.ToUpper(options.CertificateTransparencyLoggingPreference))
	if preference == "" {
		return ErrInvalidParameter
	}
	if preference != "ENABLED" && preference != "DISABLED" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	entry.Options.CertificateTransparencyLoggingPreference = preference
	return nil
}

func (s *Service) AddTagsToCertificate(certificateArn string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	incoming := normalizeTags(tags)
	prospective := cloneStringMap(entry.Tags)
	for k, v := range incoming {
		prospective[k] = v
	}
	if len(prospective) > defaultMaxTags {
		return ErrLimitExceeded
	}
	entry.Tags = prospective
	return nil
}

func (s *Service) RemoveTagsFromCertificate(certificateArn string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return ErrNotFound
	}
	for _, key := range tagKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(entry.Tags, key)
	}
	return nil
}

func (s *Service) ListTagsForCertificate(certificateArn string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return nil, err
	}

	entry, ok := s.certificates[strings.TrimSpace(certificateArn)]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneStringMap(entry.Tags), nil
}

func (s *Service) GetAccountConfiguration() (AccountConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return AccountConfiguration{}, err
	}
	return s.accountConfig, nil
}

func (s *Service) PutAccountConfiguration(configuration AccountConfiguration) error {
	days := configuration.ExpiryEvents.DaysBeforeExpiry
	if days < 1 || days > 45 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkThrottleLocked(); err != nil {
		return err
	}
	s.accountConfig = configuration
	return nil
}

func (s *Service) nextCertificateARNLocked() string {
	s.seq++
	id := fmt.Sprintf("%012x", s.seq)
	return "arn:aws:acm:" + DefaultRegion + ":" + DefaultAccountID + ":certificate/" + id
}

func (s *Service) checkThrottleLocked() error {
	now := time.Now().UTC()
	floor := now.Add(-defaultThrottleWindow)
	trimmed := s.calls[:0]
	for _, item := range s.calls {
		if item.After(floor) {
			trimmed = append(trimmed, item)
		}
	}
	s.calls = trimmed
	if len(s.calls) >= defaultThrottleLimit {
		return ErrThrottling
	}
	s.calls = append(s.calls, now)
	return nil
}

func parseNextTokenOffset(nextToken string) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}
	value := 0
	for _, ch := range nextToken {
		if ch < '0' || ch > '9' {
			return 0, ErrInvalidParameter
		}
		value = value*10 + int(ch-'0')
	}
	return value, nil
}

func normalizeDomainValidationOptions(domainName, validationMethod string, supplied []DomainValidation) []DomainValidation {
	if len(supplied) == 0 {
		return []DomainValidation{defaultDomainValidation(domainName, validationMethod, domainName)}
	}
	out := make([]DomainValidation, 0, len(supplied))
	for _, item := range supplied {
		d := DomainValidation{
			DomainName:       normalizeOrDefault(strings.ToLower(item.DomainName), domainName),
			ValidationDomain: normalizeOrDefault(strings.ToLower(item.ValidationDomain), domainName),
			ValidationMethod: normalizeOrDefault(strings.ToUpper(item.ValidationMethod), validationMethod),
			ValidationStatus: "PENDING_VALIDATION",
			ResourceRecord:   item.ResourceRecord,
		}
		if d.ResourceRecord.Name == "" {
			d.ResourceRecord = ResourceRecord{
				Name:  "_acm-validation." + d.DomainName,
				Type:  "CNAME",
				Value: "validation-token." + d.DomainName,
			}
		}
		out = append(out, d)
	}
	return out
}

func defaultDomainValidation(domainName, validationMethod, validationDomain string) DomainValidation {
	return DomainValidation{
		DomainName:       domainName,
		ValidationDomain: validationDomain,
		ValidationMethod: validationMethod,
		ValidationStatus: "PENDING_VALIDATION",
		ResourceRecord: ResourceRecord{
			Name:  "_acm-validation." + domainName,
			Type:  "CNAME",
			Value: "validation-token." + domainName,
		},
	}
}

func syntheticCertificatePEM(domainName string) string {
	if strings.TrimSpace(domainName) == "" {
		domainName = "stackyard.local"
	}
	return "-----BEGIN CERTIFICATE-----\n" + domainName + "\n-----END CERTIFICATE-----"
}

func syntheticCertificateChainPEM(domainName string) string {
	if strings.TrimSpace(domainName) == "" {
		domainName = "stackyard.local"
	}
	return "-----BEGIN CERTIFICATE-----\nchain." + domainName + "\n-----END CERTIFICATE-----"
}

func syntheticPrivateKeyPEM(domainName string) string {
	if strings.TrimSpace(domainName) == "" {
		domainName = "stackyard.local"
	}
	return "-----BEGIN PRIVATE KEY-----\n" + domainName + "\n-----END PRIVATE KEY-----"
}

func importedDomainNameForARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "imported.stackyard.example.com"
	}
	if idx := strings.LastIndex(arn, "/"); idx >= 0 && idx+1 < len(arn) {
		return strings.ToLower(strings.TrimSpace(arn[idx+1:])) + ".imported.example.com"
	}
	return "imported.stackyard.example.com"
}

func dedupeNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeTags(in map[string]string) map[string]string {
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

func cloneCertificate(in Certificate) Certificate {
	out := in
	out.SubjectAlternativeNames = append([]string(nil), in.SubjectAlternativeNames...)
	out.InUseBy = append([]string(nil), in.InUseBy...)
	out.DomainValidationOptions = cloneDomainValidations(in.DomainValidationOptions)
	out.RenewalSummary = RenewalSummary{
		RenewalStatus:           in.RenewalSummary.RenewalStatus,
		UpdatedAt:               in.RenewalSummary.UpdatedAt,
		DomainValidationOptions: cloneDomainValidations(in.RenewalSummary.DomainValidationOptions),
	}
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneDomainValidations(in []DomainValidation) []DomainValidation {
	if len(in) == 0 {
		return nil
	}
	out := make([]DomainValidation, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func normalizeOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func isValidCertificateStatus(value string) bool {
	switch value {
	case "PENDING_VALIDATION", "ISSUED", "INACTIVE", "EXPIRED", "VALIDATION_TIMED_OUT", "REVOKED", "FAILED":
		return true
	default:
		return false
	}
}

func isValidRevocationReason(value string) bool {
	switch value {
	case "UNSPECIFIED", "KEY_COMPROMISE", "CA_COMPROMISE", "AFFILIATION_CHANGED", "SUPERCEDED", "CESSATION_OF_OPERATION", "CERTIFICATE_HOLD", "REMOVE_FROM_CRL", "PRIVILEGE_WITHDRAWN", "A_A_COMPROMISE":
		return true
	default:
		return false
	}
}

func isValidDomainName(value string) bool {
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, " /\\") {
		return false
	}
	return strings.Contains(value, ".")
}

func isValidCertificateARN(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "arn:aws:acm:") && strings.Contains(value, ":certificate/")
}
