package server

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cognitoIdentityProvider struct {
	ProviderName         string
	ClientID             string
	ServerSideTokenCheck bool
}

type cognitoIdentityPoolRecord struct {
	IdentityPoolID                 string
	IdentityPoolName               string
	AllowUnauthenticatedIdentities bool
	AllowClassicFlow               bool
	SupportedLoginProviders        map[string]string
	DeveloperProviderName          string
	OpenIDConnectProviderARNs      []string
	CognitoIdentityProviders       []cognitoIdentityProvider
	SamlProviderARNs               []string
	IdentityPoolTags               map[string]string
	Roles                          map[string]string
	RoleMappings                   map[string]cognitoIdentityRoleMapping
	PrincipalTagAttributeMaps      map[string]cognitoIdentityPrincipalTagAttributeMap
	CreationDate                   time.Time
	LastModifiedDate               time.Time
}

type cognitoIdentityMappingRule struct {
	Claim     string
	MatchType string
	RoleARN   string
	Value     string
}

type cognitoIdentityRulesConfiguration struct {
	Rules []cognitoIdentityMappingRule
}

type cognitoIdentityRoleMapping struct {
	Type                    string
	AmbiguousRoleResolution string
	RulesConfiguration      *cognitoIdentityRulesConfiguration
}

type cognitoIdentityPrincipalTagAttributeMap struct {
	IdentityPoolID       string
	IdentityProviderName string
	PrincipalTags        map[string]string
	UseDefaults          bool
}

type cognitoIdentityRecord struct {
	IdentityID     string
	IdentityPoolID string
	Logins         map[string]string
	CreationDate   time.Time
	LastModified   time.Time
	Disabled       bool
}

type cognitoIdentityCredentials struct {
	AccessKeyID  string
	SecretKey    string
	SessionToken string
	Expiration   int64
}

type cognitoIdentityCreatePoolInput struct {
	Region                         string
	IdentityPoolName               string
	AllowUnauthenticatedIdentities bool
	AllowClassicFlow               bool
	SupportedLoginProviders        map[string]string
	DeveloperProviderName          string
	OpenIDConnectProviderARNs      []string
	CognitoIdentityProviders       []cognitoIdentityProvider
	SamlProviderARNs               []string
	IdentityPoolTags               map[string]string
}

type cognitoIdentityUpdatePoolInput struct {
	IdentityPoolID                 string
	IdentityPoolName               string
	AllowUnauthenticatedIdentities bool
	AllowClassicFlow               bool
	SupportedLoginProviders        map[string]string
	DeveloperProviderName          string
	OpenIDConnectProviderARNs      []string
	CognitoIdentityProviders       []cognitoIdentityProvider
	SamlProviderARNs               []string
	IdentityPoolTags               map[string]string
}

type cognitoIdentityStore struct {
	mu                      sync.Mutex
	pools                   map[string]cognitoIdentityPoolRecord
	identities              map[string]cognitoIdentityRecord
	poolIdentityIDs         map[string][]string
	identityByPoolLogins    map[string]map[string]string
	developerUserToIdentity map[string]map[string]string
	identityDeveloperUsers  map[string]map[string]struct{}
}

func newCognitoIdentityStore() *cognitoIdentityStore {
	return &cognitoIdentityStore{
		pools:                   map[string]cognitoIdentityPoolRecord{},
		identities:              map[string]cognitoIdentityRecord{},
		poolIdentityIDs:         map[string][]string{},
		identityByPoolLogins:    map[string]map[string]string{},
		developerUserToIdentity: map[string]map[string]string{},
		identityDeveloperUsers:  map[string]map[string]struct{}{},
	}
}

func (s *cognitoIdentityStore) CreatePool(input cognitoIdentityCreatePoolInput) (cognitoIdentityPoolRecord, error) {
	name := strings.TrimSpace(input.IdentityPoolName)
	if name == "" {
		return cognitoIdentityPoolRecord{}, validationCognitoIdentity("IdentityPoolName is required")
	}

	region := strings.TrimSpace(input.Region)
	if region == "" {
		region = defaultSigV4Region
	}

	now := time.Now().UTC()
	record := cognitoIdentityPoolRecord{
		IdentityPoolID:                 region + ":" + cognitoIdentityUUID(),
		IdentityPoolName:               name,
		AllowUnauthenticatedIdentities: input.AllowUnauthenticatedIdentities,
		AllowClassicFlow:               input.AllowClassicFlow,
		SupportedLoginProviders:        cloneStringMap(input.SupportedLoginProviders),
		DeveloperProviderName:          strings.TrimSpace(input.DeveloperProviderName),
		OpenIDConnectProviderARNs:      cloneStringSlice(input.OpenIDConnectProviderARNs),
		CognitoIdentityProviders:       cloneCognitoIdentityProviders(input.CognitoIdentityProviders),
		SamlProviderARNs:               cloneStringSlice(input.SamlProviderARNs),
		IdentityPoolTags:               cloneStringMap(input.IdentityPoolTags),
		CreationDate:                   now,
		LastModifiedDate:               now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools[record.IdentityPoolID] = record
	return record, nil
}

func (s *cognitoIdentityStore) DescribePool(identityPoolID string) (cognitoIdentityPoolRecord, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return cognitoIdentityPoolRecord{}, validationCognitoIdentity("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.pools[identityPoolID]
	if !ok {
		return cognitoIdentityPoolRecord{}, notFoundCognitoIdentity("IdentityPool not found")
	}
	return record, nil
}

func (s *cognitoIdentityStore) ListPools(maxResults int, nextToken string) ([]cognitoIdentityPoolRecord, string, error) {
	if maxResults <= 0 {
		return nil, "", validationCognitoIdentity("MaxResults is required")
	}
	if maxResults > 60 {
		return nil, "", validationCognitoIdentity("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.pools))
	for id := range s.pools {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseCognitoIdentityNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}

	items := make([]cognitoIdentityPoolRecord, 0, end-start)
	for _, id := range ids[start:end] {
		items = append(items, s.pools[id])
	}

	outNextToken := ""
	if end < len(ids) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cognitoIdentityStore) UpdatePool(input cognitoIdentityUpdatePoolInput) (cognitoIdentityPoolRecord, error) {
	identityPoolID := strings.TrimSpace(input.IdentityPoolID)
	if identityPoolID == "" {
		return cognitoIdentityPoolRecord{}, validationCognitoIdentity("IdentityPoolId is required")
	}

	name := strings.TrimSpace(input.IdentityPoolName)
	if name == "" {
		return cognitoIdentityPoolRecord{}, validationCognitoIdentity("IdentityPoolName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return cognitoIdentityPoolRecord{}, notFoundCognitoIdentity("IdentityPool not found")
	}

	record.IdentityPoolName = name
	record.AllowUnauthenticatedIdentities = input.AllowUnauthenticatedIdentities
	record.AllowClassicFlow = input.AllowClassicFlow
	record.SupportedLoginProviders = cloneStringMap(input.SupportedLoginProviders)
	record.DeveloperProviderName = strings.TrimSpace(input.DeveloperProviderName)
	record.OpenIDConnectProviderARNs = cloneStringSlice(input.OpenIDConnectProviderARNs)
	record.CognitoIdentityProviders = cloneCognitoIdentityProviders(input.CognitoIdentityProviders)
	record.SamlProviderARNs = cloneStringSlice(input.SamlProviderARNs)
	record.IdentityPoolTags = cloneStringMap(input.IdentityPoolTags)
	record.LastModifiedDate = time.Now().UTC()

	s.pools[identityPoolID] = record
	return record, nil
}

func (s *cognitoIdentityStore) DeletePool(identityPoolID string) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return validationCognitoIdentity("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[identityPoolID]; !ok {
		return notFoundCognitoIdentity("IdentityPool not found")
	}
	delete(s.pools, identityPoolID)

	for _, identityID := range s.poolIdentityIDs[identityPoolID] {
		s.removeIdentityLocked(identityID)
	}
	delete(s.poolIdentityIDs, identityPoolID)
	delete(s.identityByPoolLogins, identityPoolID)
	delete(s.developerUserToIdentity, identityPoolID)
	return nil
}

func (s *cognitoIdentityStore) GetOrCreateIdentity(identityPoolID string, logins map[string]string) (cognitoIdentityRecord, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return cognitoIdentityRecord{}, validationCognitoIdentity("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[identityPoolID]
	if !ok {
		return cognitoIdentityRecord{}, notFoundCognitoIdentity("IdentityPool not found")
	}

	loginKey := canonicalCognitoIdentityLoginsKey(logins)
	if byLogins, ok := s.identityByPoolLogins[identityPoolID]; ok {
		if existingID, found := byLogins[loginKey]; found {
			if existing, ok := s.identities[existingID]; ok {
				return existing, nil
			}
		}
	}

	region := cognitoIdentityRegionFromPoolID(pool.IdentityPoolID)
	now := time.Now().UTC()
	record := cognitoIdentityRecord{
		IdentityID:     region + ":" + cognitoIdentityUUID(),
		IdentityPoolID: identityPoolID,
		Logins:         cloneStringMap(logins),
		CreationDate:   now,
		LastModified:   now,
	}

	s.identities[record.IdentityID] = record
	s.poolIdentityIDs[identityPoolID] = append(s.poolIdentityIDs[identityPoolID], record.IdentityID)
	if _, ok := s.identityByPoolLogins[identityPoolID]; !ok {
		s.identityByPoolLogins[identityPoolID] = map[string]string{}
	}
	s.identityByPoolLogins[identityPoolID][loginKey] = record.IdentityID
	return record, nil
}

func (s *cognitoIdentityStore) DescribeIdentity(identityID string) (cognitoIdentityRecord, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return cognitoIdentityRecord{}, validationCognitoIdentity("IdentityId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.identities[identityID]
	if !ok {
		return cognitoIdentityRecord{}, notFoundCognitoIdentity("Identity not found")
	}
	return record, nil
}

func (s *cognitoIdentityStore) ListIdentities(identityPoolID string, maxResults int, nextToken string, hideDisabled bool) ([]cognitoIdentityRecord, string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, "", validationCognitoIdentity("IdentityPoolId is required")
	}
	if maxResults <= 0 {
		return nil, "", validationCognitoIdentity("MaxResults is required")
	}
	if maxResults > 60 {
		return nil, "", validationCognitoIdentity("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[identityPoolID]; !ok {
		return nil, "", notFoundCognitoIdentity("IdentityPool not found")
	}

	ids := cloneStringSlice(s.poolIdentityIDs[identityPoolID])
	sort.Strings(ids)

	filteredIDs := ids[:0]
	for _, id := range ids {
		record, ok := s.identities[id]
		if !ok {
			continue
		}
		if hideDisabled && record.Disabled {
			continue
		}
		filteredIDs = append(filteredIDs, id)
	}

	start, err := parseCognitoIdentityNextToken(nextToken, len(filteredIDs))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(filteredIDs) {
		end = len(filteredIDs)
	}

	items := make([]cognitoIdentityRecord, 0, end-start)
	for _, id := range filteredIDs[start:end] {
		items = append(items, s.identities[id])
	}

	outNextToken := ""
	if end < len(filteredIDs) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cognitoIdentityStore) GetCredentials(identityID string) (cognitoIdentityCredentials, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return cognitoIdentityCredentials{}, validationCognitoIdentity("IdentityId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.identities[identityID]; !ok {
		return cognitoIdentityCredentials{}, notFoundCognitoIdentity("Identity not found")
	}

	return cognitoIdentityCredentials{
		AccessKeyID:  "ASI" + strings.ToUpper(randomHex(8)),
		SecretKey:    randomHex(20),
		SessionToken: randomHex(32),
		Expiration:   time.Now().UTC().Add(1 * time.Hour).Unix(),
	}, nil
}

func (s *cognitoIdentityStore) GetOpenIDToken(identityID string) (string, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return "", validationCognitoIdentity("IdentityId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.identities[identityID]; !ok {
		return "", notFoundCognitoIdentity("Identity not found")
	}
	return "openid-token-" + randomHex(16), nil
}

func (s *cognitoIdentityStore) removeIdentityLocked(identityID string) {
	record, ok := s.identities[identityID]
	if !ok {
		return
	}

	if byLogins, ok := s.identityByPoolLogins[record.IdentityPoolID]; ok {
		loginKey := canonicalCognitoIdentityLoginsKey(record.Logins)
		delete(byLogins, loginKey)
		if len(byLogins) == 0 {
			delete(s.identityByPoolLogins, record.IdentityPoolID)
		}
	}

	if keys, ok := s.identityDeveloperUsers[identityID]; ok {
		if byPool, ok := s.developerUserToIdentity[record.IdentityPoolID]; ok {
			for key := range keys {
				delete(byPool, key)
			}
			if len(byPool) == 0 {
				delete(s.developerUserToIdentity, record.IdentityPoolID)
			}
		}
		delete(s.identityDeveloperUsers, identityID)
	}

	ids := s.poolIdentityIDs[record.IdentityPoolID]
	for i, candidate := range ids {
		if candidate != identityID {
			continue
		}
		ids = append(ids[:i], ids[i+1:]...)
		break
	}
	if len(ids) == 0 {
		delete(s.poolIdentityIDs, record.IdentityPoolID)
	} else {
		s.poolIdentityIDs[record.IdentityPoolID] = ids
	}

	delete(s.identities, identityID)
}

type cognitoIdentityAPIError struct {
	Status  int
	Code    string
	Message string
}

func (e *cognitoIdentityAPIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func validationCognitoIdentity(message string) error {
	return &cognitoIdentityAPIError{
		Status:  httpStatusBadRequest,
		Code:    "InvalidParameterException",
		Message: message,
	}
}

func notFoundCognitoIdentity(message string) error {
	return &cognitoIdentityAPIError{
		Status:  httpStatusBadRequest,
		Code:    "ResourceNotFoundException",
		Message: message,
	}
}

func conflictCognitoIdentity(message string) error {
	return &cognitoIdentityAPIError{
		Status:  httpStatusBadRequest,
		Code:    "ResourceConflictException",
		Message: message,
	}
}

func developerUserAlreadyRegisteredCognitoIdentity(message string) error {
	return &cognitoIdentityAPIError{
		Status:  httpStatusBadRequest,
		Code:    "DeveloperUserAlreadyRegisteredException",
		Message: message,
	}
}

func notAuthorizedCognitoIdentity(message string) error {
	return &cognitoIdentityAPIError{
		Status:  httpStatusBadRequest,
		Code:    "NotAuthorizedException",
		Message: message,
	}
}

const httpStatusBadRequest = 400

func parseCognitoIdentityNextToken(nextToken string, max int) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}

	start, err := strconv.Atoi(nextToken)
	if err != nil || start < 0 || start > max {
		return 0, validationCognitoIdentity("NextToken is invalid")
	}
	return start, nil
}

func canonicalCognitoIdentityLoginsKey(logins map[string]string) string {
	if len(logins) == 0 {
		return ""
	}
	keys := make([]string, 0, len(logins))
	for key := range logins {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+logins[key])
	}
	return strings.Join(parts, "|")
}

func cognitoIdentityUUID() string {
	value := randomHex(16)
	if len(value) != 32 {
		value = randomHex(16)
	}
	if len(value) != 32 {
		value = "00000000000000000000000000000000"
	}
	return value[0:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:32]
}

func cognitoIdentityRegionFromPoolID(identityPoolID string) string {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if idx := strings.Index(identityPoolID, ":"); idx > 0 {
		region := strings.TrimSpace(identityPoolID[:idx])
		if region != "" {
			return region
		}
	}
	return defaultSigV4Region
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneCognitoIdentityProviders(in []cognitoIdentityProvider) []cognitoIdentityProvider {
	if len(in) == 0 {
		return nil
	}
	out := make([]cognitoIdentityProvider, len(in))
	copy(out, in)
	return out
}

func cloneCognitoIdentityMappingRules(in []cognitoIdentityMappingRule) []cognitoIdentityMappingRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]cognitoIdentityMappingRule, len(in))
	copy(out, in)
	return out
}

func cloneCognitoIdentityRulesConfiguration(in *cognitoIdentityRulesConfiguration) *cognitoIdentityRulesConfiguration {
	if in == nil {
		return nil
	}
	return &cognitoIdentityRulesConfiguration{
		Rules: cloneCognitoIdentityMappingRules(in.Rules),
	}
}

func cloneCognitoIdentityRoleMappings(in map[string]cognitoIdentityRoleMapping) map[string]cognitoIdentityRoleMapping {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]cognitoIdentityRoleMapping, len(in))
	for key, mapping := range in {
		out[key] = cognitoIdentityRoleMapping{
			Type:                    mapping.Type,
			AmbiguousRoleResolution: mapping.AmbiguousRoleResolution,
			RulesConfiguration:      cloneCognitoIdentityRulesConfiguration(mapping.RulesConfiguration),
		}
	}
	return out
}

func cloneCognitoIdentityPrincipalTagAttributeMaps(in map[string]cognitoIdentityPrincipalTagAttributeMap) map[string]cognitoIdentityPrincipalTagAttributeMap {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]cognitoIdentityPrincipalTagAttributeMap, len(in))
	for key, mapping := range in {
		out[key] = cognitoIdentityPrincipalTagAttributeMap{
			IdentityPoolID:       mapping.IdentityPoolID,
			IdentityProviderName: mapping.IdentityProviderName,
			PrincipalTags:        cloneStringMap(mapping.PrincipalTags),
			UseDefaults:          mapping.UseDefaults,
		}
	}
	return out
}

func asCognitoIdentityAPIError(err error) *cognitoIdentityAPIError {
	if err == nil {
		return nil
	}
	var apiErr *cognitoIdentityAPIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
