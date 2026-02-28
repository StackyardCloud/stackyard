package server

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cognitoUserPoolsUserPoolRecord struct {
	ID                       string
	ARN                      string
	Name                     string
	Status                   string
	MFAConfiguration         string
	SoftwareTokenMFAEnabled  bool
	WebAuthnRelyingPartyID   string
	WebAuthnUserVerification string
	Tags                     map[string]string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type cognitoUserPoolsClientRecord struct {
	UserPoolID           string
	ClientID             string
	ClientName           string
	GenerateSecret       bool
	ClientSecret         string
	ExplicitAuthFlows    []string
	RefreshTokenValidity int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type cognitoUserPoolsDomainRecord struct {
	Domain           string
	UserPoolID       string
	CloudFrontDomain string
	Version          int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type cognitoUserPoolsResourceServerScope struct {
	ScopeName        string
	ScopeDescription string
}

type cognitoUserPoolsResourceServerRecord struct {
	UserPoolID string
	Identifier string
	Name       string
	Scopes     []cognitoUserPoolsResourceServerScope
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type cognitoUserPoolsCreateUserPoolInput struct {
	Region           string
	PoolName         string
	MFAConfiguration string
	Tags             map[string]string
}

type cognitoUserPoolsUpdateUserPoolInput struct {
	UserPoolID          string
	MFAConfiguration    string
	MFAConfigurationSet bool
	Tags                map[string]string
	TagsSet             bool
}

type cognitoUserPoolsCreateClientInput struct {
	UserPoolID           string
	ClientName           string
	GenerateSecret       bool
	ExplicitAuthFlows    []string
	RefreshTokenValidity int
}

type cognitoUserPoolsUpdateClientInput struct {
	UserPoolID              string
	ClientID                string
	ClientName              string
	ClientNameSet           bool
	ExplicitAuthFlows       []string
	ExplicitAuthFlowsSet    bool
	RefreshTokenValidity    int
	RefreshTokenValiditySet bool
}

type cognitoUserPoolsCreateResourceServerInput struct {
	UserPoolID string
	Identifier string
	Name       string
	Scopes     []cognitoUserPoolsResourceServerScope
}

type cognitoUserPoolsUpdateResourceServerInput struct {
	UserPoolID string
	Identifier string
	Name       string
	Scopes     []cognitoUserPoolsResourceServerScope
}

type cognitoUserPoolsStore struct {
	mu                sync.Mutex
	pools             map[string]cognitoUserPoolsUserPoolRecord
	domains           map[string]cognitoUserPoolsDomainRecord
	clients           map[string]map[string]cognitoUserPoolsClientRecord
	resourceServers   map[string]map[string]cognitoUserPoolsResourceServerRecord
	customAttributes  map[string]map[string]cognitoUserPoolsCustomAttributeRecord
	uiCustomizations  map[string]map[string]cognitoUserPoolsUICustomizationRecord
	managedBrandings  map[string]map[string]cognitoUserPoolsManagedLoginBrandingRecord
	terms             map[string]map[string]cognitoUserPoolsTermsRecord
	signingCerts      map[string]string
	providerLinks     map[string]map[string][]cognitoUserPoolsLinkedProviderRecord
	users             map[string]map[string]cognitoUserPoolsUserRecord
	groups            map[string]map[string]cognitoUserPoolsGroupRecord
	importJobs        map[string]map[string]cognitoUserPoolsImportJobRecord
	identityProviders map[string]map[string]cognitoUserPoolsIdentityProviderRecord
	riskConfigs       map[string]map[string]cognitoUserPoolsRiskConfigurationRecord
	logDelivery       map[string]cognitoUserPoolsLogDeliveryConfigurationRecord
	authEvents        map[string]map[string][]cognitoUserPoolsAuthEventRecord
	accessTokens      map[string]cognitoUserPoolsAccessTokenRecord
	refreshTokens     map[string]cognitoUserPoolsRefreshTokenRecord
	sessions          map[string]cognitoUserPoolsSessionRecord
}

func newCognitoUserPoolsStore() *cognitoUserPoolsStore {
	return &cognitoUserPoolsStore{
		pools:             map[string]cognitoUserPoolsUserPoolRecord{},
		domains:           map[string]cognitoUserPoolsDomainRecord{},
		clients:           map[string]map[string]cognitoUserPoolsClientRecord{},
		resourceServers:   map[string]map[string]cognitoUserPoolsResourceServerRecord{},
		customAttributes:  map[string]map[string]cognitoUserPoolsCustomAttributeRecord{},
		uiCustomizations:  map[string]map[string]cognitoUserPoolsUICustomizationRecord{},
		managedBrandings:  map[string]map[string]cognitoUserPoolsManagedLoginBrandingRecord{},
		terms:             map[string]map[string]cognitoUserPoolsTermsRecord{},
		signingCerts:      map[string]string{},
		providerLinks:     map[string]map[string][]cognitoUserPoolsLinkedProviderRecord{},
		users:             map[string]map[string]cognitoUserPoolsUserRecord{},
		groups:            map[string]map[string]cognitoUserPoolsGroupRecord{},
		importJobs:        map[string]map[string]cognitoUserPoolsImportJobRecord{},
		identityProviders: map[string]map[string]cognitoUserPoolsIdentityProviderRecord{},
		riskConfigs:       map[string]map[string]cognitoUserPoolsRiskConfigurationRecord{},
		logDelivery:       map[string]cognitoUserPoolsLogDeliveryConfigurationRecord{},
		authEvents:        map[string]map[string][]cognitoUserPoolsAuthEventRecord{},
		accessTokens:      map[string]cognitoUserPoolsAccessTokenRecord{},
		refreshTokens:     map[string]cognitoUserPoolsRefreshTokenRecord{},
		sessions:          map[string]cognitoUserPoolsSessionRecord{},
	}
}

func (s *cognitoUserPoolsStore) CreateUserPool(input cognitoUserPoolsCreateUserPoolInput) (cognitoUserPoolsUserPoolRecord, error) {
	name := strings.TrimSpace(input.PoolName)
	if name == "" {
		return cognitoUserPoolsUserPoolRecord{}, validationCognitoUserPools("PoolName is required")
	}

	region := strings.TrimSpace(input.Region)
	if region == "" {
		region = defaultSigV4Region
	}

	now := time.Now().UTC()
	id := cognitoUserPoolsID(region)
	record := cognitoUserPoolsUserPoolRecord{
		ID:               id,
		ARN:              cognitoUserPoolsARN(region, id),
		Name:             name,
		Status:           "ENABLED",
		MFAConfiguration: strings.TrimSpace(input.MFAConfiguration),
		Tags:             cloneStringMap(input.Tags),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools[id] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeUserPool(userPoolID string) (cognitoUserPoolsUserPoolRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsUserPoolRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsUserPoolRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) ListUserPools(maxResults int, nextToken string) ([]cognitoUserPoolsUserPoolRecord, string, error) {
	if maxResults <= 0 {
		return nil, "", validationCognitoUserPools("MaxResults is required")
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.pools))
	for id := range s.pools {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseCognitoUserPoolsNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}

	items := make([]cognitoUserPoolsUserPoolRecord, 0, end-start)
	for _, id := range ids[start:end] {
		items = append(items, s.pools[id])
	}

	outNextToken := ""
	if end < len(ids) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cognitoUserPoolsStore) UpdateUserPool(input cognitoUserPoolsUpdateUserPoolInput) error {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[userPoolID]
	if !ok {
		return notFoundCognitoUserPools("User pool not found")
	}

	if input.MFAConfigurationSet {
		record.MFAConfiguration = strings.TrimSpace(input.MFAConfiguration)
	}
	if input.TagsSet {
		record.Tags = cloneStringMap(input.Tags)
	}
	record.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = record
	return nil
}

func (s *cognitoUserPoolsStore) DeleteUserPool(userPoolID string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	delete(s.pools, userPoolID)
	delete(s.clients, userPoolID)
	delete(s.resourceServers, userPoolID)
	delete(s.customAttributes, userPoolID)
	delete(s.uiCustomizations, userPoolID)
	delete(s.managedBrandings, userPoolID)
	delete(s.terms, userPoolID)
	delete(s.signingCerts, userPoolID)
	delete(s.providerLinks, userPoolID)
	delete(s.users, userPoolID)
	delete(s.groups, userPoolID)
	delete(s.importJobs, userPoolID)
	delete(s.identityProviders, userPoolID)
	delete(s.riskConfigs, userPoolID)
	delete(s.logDelivery, userPoolID)
	delete(s.authEvents, userPoolID)
	for token, record := range s.accessTokens {
		if record.UserPoolID == userPoolID {
			delete(s.accessTokens, token)
		}
	}
	for token, record := range s.refreshTokens {
		if record.UserPoolID == userPoolID {
			delete(s.refreshTokens, token)
		}
	}
	for session, record := range s.sessions {
		if record.UserPoolID == userPoolID {
			delete(s.sessions, session)
		}
	}

	for domainKey, domain := range s.domains {
		if domain.UserPoolID == userPoolID {
			delete(s.domains, domainKey)
		}
	}
	return nil
}

func (s *cognitoUserPoolsStore) CreateUserPoolDomain(userPoolID, domain string) (cognitoUserPoolsDomainRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	domain = strings.TrimSpace(domain)
	if userPoolID == "" {
		return cognitoUserPoolsDomainRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if domain == "" {
		return cognitoUserPoolsDomainRecord{}, validationCognitoUserPools("Domain is required")
	}

	domainKey := strings.ToLower(domain)

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsDomainRecord{}, notFoundCognitoUserPools("User pool not found")
	}

	if existing, exists := s.domains[domainKey]; exists && existing.UserPoolID != userPoolID {
		return cognitoUserPoolsDomainRecord{}, conflictCognitoUserPools("Domain already exists")
	}

	now := time.Now().UTC()
	record := cognitoUserPoolsDomainRecord{
		Domain:           domain,
		UserPoolID:       userPoolID,
		CloudFrontDomain: domain + ".auth." + cognitoUserPoolsRegionFromPoolID(userPoolID) + ".amazoncognito.com",
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.domains[domainKey] = record

	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeUserPoolDomain(domain string) (cognitoUserPoolsDomainRecord, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return cognitoUserPoolsDomainRecord{}, validationCognitoUserPools("Domain is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.domains[strings.ToLower(domain)]
	if !ok {
		return cognitoUserPoolsDomainRecord{}, notFoundCognitoUserPools("Domain not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) DeleteUserPoolDomain(userPoolID, domain string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	domain = strings.TrimSpace(domain)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if domain == "" {
		return validationCognitoUserPools("Domain is required")
	}

	domainKey := strings.ToLower(domain)

	s.mu.Lock()
	defer s.mu.Unlock()

	domainRecord, ok := s.domains[domainKey]
	if !ok || domainRecord.UserPoolID != userPoolID {
		return notFoundCognitoUserPools("Domain not found")
	}
	delete(s.domains, domainKey)

	pool, exists := s.pools[userPoolID]
	if exists {
		pool.UpdatedAt = time.Now().UTC()
		s.pools[userPoolID] = pool
	}
	return nil
}

func (s *cognitoUserPoolsStore) CreateUserPoolClient(input cognitoUserPoolsCreateClientInput) (cognitoUserPoolsClientRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	clientName := strings.TrimSpace(input.ClientName)
	if userPoolID == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if clientName == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("ClientName is required")
	}

	if input.RefreshTokenValidity < 0 {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("RefreshTokenValidity is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsClientRecord{}, notFoundCognitoUserPools("User pool not found")
	}

	if s.clients[userPoolID] == nil {
		s.clients[userPoolID] = map[string]cognitoUserPoolsClientRecord{}
	}

	now := time.Now().UTC()
	clientID := "client-" + randomHex(8)
	record := cognitoUserPoolsClientRecord{
		UserPoolID:           userPoolID,
		ClientID:             clientID,
		ClientName:           clientName,
		GenerateSecret:       input.GenerateSecret,
		ExplicitAuthFlows:    cloneStringSlice(input.ExplicitAuthFlows),
		RefreshTokenValidity: input.RefreshTokenValidity,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if record.RefreshTokenValidity == 0 {
		record.RefreshTokenValidity = 30
	}
	if input.GenerateSecret {
		record.ClientSecret = randomHex(16)
	}

	s.clients[userPoolID][clientID] = record
	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeUserPoolClient(userPoolID, clientID string) (cognitoUserPoolsClientRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if clientID == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("ClientId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.describeUserPoolClientLocked(userPoolID, clientID)
}

func (s *cognitoUserPoolsStore) describeUserPoolClientLocked(userPoolID, clientID string) (cognitoUserPoolsClientRecord, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsClientRecord{}, notFoundCognitoUserPools("User pool not found")
	}

	poolClients := s.clients[userPoolID]
	record, ok := poolClients[clientID]
	if !ok {
		return cognitoUserPoolsClientRecord{}, notFoundCognitoUserPools("User pool client not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) ListUserPoolClients(userPoolID string, maxResults int, nextToken string) ([]cognitoUserPoolsClientRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if maxResults <= 0 {
		return nil, "", validationCognitoUserPools("MaxResults is required")
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}

	poolClients := s.clients[userPoolID]
	ids := make([]string, 0, len(poolClients))
	for clientID := range poolClients {
		ids = append(ids, clientID)
	}
	sort.Strings(ids)

	start, err := parseCognitoUserPoolsNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}

	items := make([]cognitoUserPoolsClientRecord, 0, end-start)
	for _, clientID := range ids[start:end] {
		items = append(items, poolClients[clientID])
	}

	outNextToken := ""
	if end < len(ids) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cognitoUserPoolsStore) UpdateUserPoolClient(input cognitoUserPoolsUpdateClientInput) (cognitoUserPoolsClientRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	clientID := strings.TrimSpace(input.ClientID)
	if userPoolID == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if clientID == "" {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("ClientId is required")
	}
	if input.RefreshTokenValiditySet && input.RefreshTokenValidity < 0 {
		return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("RefreshTokenValidity is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.describeUserPoolClientLocked(userPoolID, clientID)
	if err != nil {
		return cognitoUserPoolsClientRecord{}, err
	}

	if input.ClientNameSet {
		name := strings.TrimSpace(input.ClientName)
		if name == "" {
			return cognitoUserPoolsClientRecord{}, validationCognitoUserPools("ClientName is required")
		}
		record.ClientName = name
	}
	if input.ExplicitAuthFlowsSet {
		record.ExplicitAuthFlows = cloneStringSlice(input.ExplicitAuthFlows)
	}
	if input.RefreshTokenValiditySet {
		record.RefreshTokenValidity = input.RefreshTokenValidity
	}
	record.UpdatedAt = time.Now().UTC()

	s.clients[userPoolID][clientID] = record
	pool := s.pools[userPoolID]
	pool.UpdatedAt = record.UpdatedAt
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DeleteUserPoolClient(userPoolID, clientID string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if clientID == "" {
		return validationCognitoUserPools("ClientId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}

	poolClients := s.clients[userPoolID]
	if _, ok := poolClients[clientID]; !ok {
		return notFoundCognitoUserPools("User pool client not found")
	}
	delete(poolClients, clientID)
	if len(poolClients) == 0 {
		delete(s.clients, userPoolID)
	} else {
		s.clients[userPoolID] = poolClients
	}

	pool := s.pools[userPoolID]
	pool.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = pool
	return nil
}

func (s *cognitoUserPoolsStore) TagUserPool(userPoolID string, tags map[string]string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[userPoolID]
	if !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	if record.Tags == nil {
		record.Tags = map[string]string{}
	}
	for key, value := range tags {
		record.Tags[key] = value
	}
	record.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = record
	return nil
}

func (s *cognitoUserPoolsStore) UntagUserPool(userPoolID string, tagKeys []string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[userPoolID]
	if !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	for _, key := range tagKeys {
		delete(record.Tags, strings.TrimSpace(key))
	}
	record.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = record
	return nil
}

func (s *cognitoUserPoolsStore) ListUserPoolTags(userPoolID string) (map[string]string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[userPoolID]
	if !ok {
		return nil, notFoundCognitoUserPools("User pool not found")
	}
	return cloneStringMap(record.Tags), nil
}

func (s *cognitoUserPoolsStore) CreateResourceServer(input cognitoUserPoolsCreateResourceServerInput) (cognitoUserPoolsResourceServerRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	identifier := strings.TrimSpace(input.Identifier)
	name := strings.TrimSpace(input.Name)
	if userPoolID == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if identifier == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("Identifier is required")
	}
	if name == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("Name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsResourceServerRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.resourceServers[userPoolID] == nil {
		s.resourceServers[userPoolID] = map[string]cognitoUserPoolsResourceServerRecord{}
	}
	if _, exists := s.resourceServers[userPoolID][identifier]; exists {
		return cognitoUserPoolsResourceServerRecord{}, conflictCognitoUserPools("Resource server already exists")
	}

	now := time.Now().UTC()
	record := cognitoUserPoolsResourceServerRecord{
		UserPoolID: userPoolID,
		Identifier: identifier,
		Name:       name,
		Scopes:     cloneCognitoUserPoolsScopes(input.Scopes),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.resourceServers[userPoolID][identifier] = record
	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeResourceServer(userPoolID, identifier string) (cognitoUserPoolsResourceServerRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	identifier = strings.TrimSpace(identifier)
	if userPoolID == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if identifier == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("Identifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.describeResourceServerLocked(userPoolID, identifier)
}

func (s *cognitoUserPoolsStore) describeResourceServerLocked(userPoolID, identifier string) (cognitoUserPoolsResourceServerRecord, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsResourceServerRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	poolResourceServers := s.resourceServers[userPoolID]
	record, ok := poolResourceServers[identifier]
	if !ok {
		return cognitoUserPoolsResourceServerRecord{}, notFoundCognitoUserPools("Resource server not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) ListResourceServers(userPoolID string, maxResults int, nextToken string) ([]cognitoUserPoolsResourceServerRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if maxResults <= 0 {
		return nil, "", validationCognitoUserPools("MaxResults is required")
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}

	poolResourceServers := s.resourceServers[userPoolID]
	identifiers := make([]string, 0, len(poolResourceServers))
	for identifier := range poolResourceServers {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	start, err := parseCognitoUserPoolsNextToken(nextToken, len(identifiers))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(identifiers) {
		end = len(identifiers)
	}

	items := make([]cognitoUserPoolsResourceServerRecord, 0, end-start)
	for _, identifier := range identifiers[start:end] {
		items = append(items, poolResourceServers[identifier])
	}

	outNextToken := ""
	if end < len(identifiers) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cognitoUserPoolsStore) UpdateResourceServer(input cognitoUserPoolsUpdateResourceServerInput) (cognitoUserPoolsResourceServerRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	identifier := strings.TrimSpace(input.Identifier)
	name := strings.TrimSpace(input.Name)
	if userPoolID == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if identifier == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("Identifier is required")
	}
	if name == "" {
		return cognitoUserPoolsResourceServerRecord{}, validationCognitoUserPools("Name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.describeResourceServerLocked(userPoolID, identifier)
	if err != nil {
		return cognitoUserPoolsResourceServerRecord{}, err
	}
	record.Name = name
	record.Scopes = cloneCognitoUserPoolsScopes(input.Scopes)
	record.UpdatedAt = time.Now().UTC()
	s.resourceServers[userPoolID][identifier] = record

	pool := s.pools[userPoolID]
	pool.UpdatedAt = record.UpdatedAt
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DeleteResourceServer(userPoolID, identifier string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	identifier = strings.TrimSpace(identifier)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if identifier == "" {
		return validationCognitoUserPools("Identifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}

	poolResourceServers := s.resourceServers[userPoolID]
	if _, ok := poolResourceServers[identifier]; !ok {
		return notFoundCognitoUserPools("Resource server not found")
	}
	delete(poolResourceServers, identifier)
	if len(poolResourceServers) == 0 {
		delete(s.resourceServers, userPoolID)
	} else {
		s.resourceServers[userPoolID] = poolResourceServers
	}

	pool := s.pools[userPoolID]
	pool.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = pool
	return nil
}

type cognitoUserPoolsAPIError struct {
	Status  int
	Code    string
	Message string
}

func (e *cognitoUserPoolsAPIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func validationCognitoUserPools(message string) error {
	return &cognitoUserPoolsAPIError{
		Status:  httpStatusBadRequest,
		Code:    "InvalidParameterException",
		Message: message,
	}
}

func notFoundCognitoUserPools(message string) error {
	return &cognitoUserPoolsAPIError{
		Status:  httpStatusBadRequest,
		Code:    "ResourceNotFoundException",
		Message: message,
	}
}

func conflictCognitoUserPools(message string) error {
	return &cognitoUserPoolsAPIError{
		Status:  httpStatusBadRequest,
		Code:    "ResourceConflictException",
		Message: message,
	}
}

func parseCognitoUserPoolsNextToken(nextToken string, max int) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}

	start, err := strconv.Atoi(nextToken)
	if err != nil || start < 0 || start > max {
		return 0, validationCognitoUserPools("NextToken is invalid")
	}
	return start, nil
}

func cognitoUserPoolsID(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		region = defaultSigV4Region
	}
	return region + "_" + randomHex(8)
}

func cognitoUserPoolsARN(region, userPoolID string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		region = defaultSigV4Region
	}
	return "arn:aws:cognito-idp:" + region + ":123456789012:userpool/" + userPoolID
}

func cognitoUserPoolsRegionFromPoolID(userPoolID string) string {
	userPoolID = strings.TrimSpace(userPoolID)
	if idx := strings.Index(userPoolID, "_"); idx > 0 {
		region := strings.TrimSpace(userPoolID[:idx])
		if region != "" {
			return region
		}
	}
	return defaultSigV4Region
}

func cloneCognitoUserPoolsScopes(in []cognitoUserPoolsResourceServerScope) []cognitoUserPoolsResourceServerScope {
	if len(in) == 0 {
		return nil
	}
	out := make([]cognitoUserPoolsResourceServerScope, len(in))
	copy(out, in)
	return out
}

func asCognitoUserPoolsAPIError(err error) *cognitoUserPoolsAPIError {
	if err == nil {
		return nil
	}
	var apiErr *cognitoUserPoolsAPIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
