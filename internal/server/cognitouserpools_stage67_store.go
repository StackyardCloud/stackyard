package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type cognitoUserPoolsIdentityProviderRecord struct {
	UserPoolID       string
	ProviderName     string
	ProviderType     string
	ProviderDetails  map[string]string
	AttributeMapping map[string]string
	IdpIdentifiers   []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type cognitoUserPoolsRiskConfigurationRecord struct {
	UserPoolID                              string
	ClientID                                string
	CompromisedCredentialsRiskConfiguration map[string]any
	AccountTakeoverRiskConfiguration        map[string]any
	RiskExceptionConfiguration              map[string]any
	UpdatedAt                               time.Time
}

type cognitoUserPoolsLogDeliveryConfigurationRecord struct {
	UserPoolID        string
	LogConfigurations []map[string]any
	UpdatedAt         time.Time
}

type cognitoUserPoolsAuthEventRecord struct {
	EventID       string
	EventType     string
	EventResponse string
	EventRisk     map[string]any
	CreationDate  time.Time
	FeedbackValue string
	FeedbackDate  *time.Time
}

type cognitoUserPoolsCreateIdentityProviderInput struct {
	UserPoolID       string
	ProviderName     string
	ProviderType     string
	ProviderDetails  map[string]string
	AttributeMapping map[string]string
	IdpIdentifiers   []string
}

type cognitoUserPoolsUpdateIdentityProviderInput struct {
	UserPoolID          string
	ProviderName        string
	ProviderDetails     map[string]string
	ProviderDetailsSet  bool
	AttributeMapping    map[string]string
	AttributeMappingSet bool
	IdpIdentifiers      []string
	IdpIdentifiersSet   bool
}

func (s *cognitoUserPoolsStore) CreateIdentityProvider(input cognitoUserPoolsCreateIdentityProviderInput) (cognitoUserPoolsIdentityProviderRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	providerName := strings.TrimSpace(input.ProviderName)
	providerType := strings.TrimSpace(input.ProviderType)
	if userPoolID == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if providerName == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("ProviderName is required")
	}
	if providerType == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("ProviderType is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsIdentityProviderRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.identityProviders[userPoolID] == nil {
		s.identityProviders[userPoolID] = map[string]cognitoUserPoolsIdentityProviderRecord{}
	}
	providerKey := cognitoUserPoolsIdentityProviderKey(providerName)
	if _, exists := s.identityProviders[userPoolID][providerKey]; exists {
		return cognitoUserPoolsIdentityProviderRecord{}, conflictCognitoUserPools("Identity provider already exists")
	}

	now := time.Now().UTC()
	record := cognitoUserPoolsIdentityProviderRecord{
		UserPoolID:       userPoolID,
		ProviderName:     providerName,
		ProviderType:     providerType,
		ProviderDetails:  cloneStringMap(input.ProviderDetails),
		AttributeMapping: cloneStringMap(input.AttributeMapping),
		IdpIdentifiers:   cloneStringSlice(input.IdpIdentifiers),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.identityProviders[userPoolID][providerKey] = record

	pool := s.pools[userPoolID]
	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeIdentityProvider(userPoolID, providerName string) (cognitoUserPoolsIdentityProviderRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	providerName = strings.TrimSpace(providerName)
	if userPoolID == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if providerName == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("ProviderName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.describeIdentityProviderLocked(userPoolID, providerName)
}

func (s *cognitoUserPoolsStore) describeIdentityProviderLocked(userPoolID, providerName string) (cognitoUserPoolsIdentityProviderRecord, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsIdentityProviderRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	record, ok := s.identityProviders[userPoolID][cognitoUserPoolsIdentityProviderKey(providerName)]
	if !ok {
		return cognitoUserPoolsIdentityProviderRecord{}, notFoundCognitoUserPools("Identity provider not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) GetIdentityProviderByIdentifier(userPoolID, identifier string) (cognitoUserPoolsIdentityProviderRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	identifier = strings.TrimSpace(identifier)
	if userPoolID == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if identifier == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("IdpIdentifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsIdentityProviderRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	providersByName := s.identityProviders[userPoolID]
	providerKey := cognitoUserPoolsIdentityProviderKey(identifier)
	if record, ok := providersByName[providerKey]; ok {
		return record, nil
	}
	identifierKey := strings.ToLower(identifier)
	for _, record := range providersByName {
		for _, idpIdentifier := range record.IdpIdentifiers {
			if strings.ToLower(strings.TrimSpace(idpIdentifier)) == identifierKey {
				return record, nil
			}
		}
	}
	return cognitoUserPoolsIdentityProviderRecord{}, notFoundCognitoUserPools("Identity provider not found")
}

func (s *cognitoUserPoolsStore) ListIdentityProviders(userPoolID string, maxResults int, nextToken string) ([]cognitoUserPoolsIdentityProviderRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if maxResults <= 0 {
		maxResults = 60
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}
	providersByName := s.identityProviders[userPoolID]
	keys := make([]string, 0, len(providersByName))
	for key := range providersByName {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	start, err := parseCognitoUserPoolsNextToken(nextToken, len(keys))
	if err != nil {
		return nil, "", err
	}
	end := start + maxResults
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]cognitoUserPoolsIdentityProviderRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, providersByName[key])
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) UpdateIdentityProvider(input cognitoUserPoolsUpdateIdentityProviderInput) (cognitoUserPoolsIdentityProviderRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	providerName := strings.TrimSpace(input.ProviderName)
	if userPoolID == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if providerName == "" {
		return cognitoUserPoolsIdentityProviderRecord{}, validationCognitoUserPools("ProviderName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.describeIdentityProviderLocked(userPoolID, providerName)
	if err != nil {
		return cognitoUserPoolsIdentityProviderRecord{}, err
	}
	if input.ProviderDetailsSet {
		record.ProviderDetails = cloneStringMap(input.ProviderDetails)
	}
	if input.AttributeMappingSet {
		record.AttributeMapping = cloneStringMap(input.AttributeMapping)
	}
	if input.IdpIdentifiersSet {
		record.IdpIdentifiers = cloneStringSlice(input.IdpIdentifiers)
	}
	record.UpdatedAt = time.Now().UTC()
	s.identityProviders[userPoolID][cognitoUserPoolsIdentityProviderKey(providerName)] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) DeleteIdentityProvider(userPoolID, providerName string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	providerName = strings.TrimSpace(providerName)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if providerName == "" {
		return validationCognitoUserPools("ProviderName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	providerKey := cognitoUserPoolsIdentityProviderKey(providerName)
	if _, ok := s.identityProviders[userPoolID][providerKey]; !ok {
		return notFoundCognitoUserPools("Identity provider not found")
	}
	delete(s.identityProviders[userPoolID], providerKey)
	return nil
}

func (s *cognitoUserPoolsStore) SetRiskConfiguration(userPoolID, clientID string, compromised, takeover, exceptions map[string]any) (cognitoUserPoolsRiskConfigurationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsRiskConfigurationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsRiskConfigurationRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.riskConfigs[userPoolID] == nil {
		s.riskConfigs[userPoolID] = map[string]cognitoUserPoolsRiskConfigurationRecord{}
	}
	now := time.Now().UTC()
	record := cognitoUserPoolsRiskConfigurationRecord{
		UserPoolID:                              userPoolID,
		ClientID:                                clientID,
		CompromisedCredentialsRiskConfiguration: cloneCognitoUserPoolsMapAny(compromised),
		AccountTakeoverRiskConfiguration:        cloneCognitoUserPoolsMapAny(takeover),
		RiskExceptionConfiguration:              cloneCognitoUserPoolsMapAny(exceptions),
		UpdatedAt:                               now,
	}
	s.riskConfigs[userPoolID][clientID] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeRiskConfiguration(userPoolID, clientID string) (cognitoUserPoolsRiskConfigurationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsRiskConfigurationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsRiskConfigurationRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if configsByClient := s.riskConfigs[userPoolID]; configsByClient != nil {
		if record, ok := configsByClient[clientID]; ok {
			return cloneCognitoUserPoolsRiskConfigurationRecord(record), nil
		}
	}
	return cognitoUserPoolsRiskConfigurationRecord{UserPoolID: userPoolID, ClientID: clientID}, nil
}

func (s *cognitoUserPoolsStore) SetLogDeliveryConfiguration(userPoolID string, logConfigurations []map[string]any) (cognitoUserPoolsLogDeliveryConfigurationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsLogDeliveryConfigurationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsLogDeliveryConfigurationRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	record := cognitoUserPoolsLogDeliveryConfigurationRecord{
		UserPoolID:        userPoolID,
		LogConfigurations: cloneCognitoUserPoolsSliceMapAny(logConfigurations),
		UpdatedAt:         time.Now().UTC(),
	}
	s.logDelivery[userPoolID] = record
	return cloneCognitoUserPoolsLogDeliveryConfigurationRecord(record), nil
}

func (s *cognitoUserPoolsStore) GetLogDeliveryConfiguration(userPoolID string) (cognitoUserPoolsLogDeliveryConfigurationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsLogDeliveryConfigurationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsLogDeliveryConfigurationRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if record, ok := s.logDelivery[userPoolID]; ok {
		return cloneCognitoUserPoolsLogDeliveryConfigurationRecord(record), nil
	}
	return cognitoUserPoolsLogDeliveryConfigurationRecord{UserPoolID: userPoolID, LogConfigurations: []map[string]any{}}, nil
}

func (s *cognitoUserPoolsStore) AdminListUserAuthEvents(userPoolID, username string, maxResults int, nextToken string) ([]cognitoUserPoolsAuthEventRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return nil, "", validationCognitoUserPools("Username is required")
	}
	if maxResults <= 0 {
		maxResults = 60
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}
	if _, _, err := s.getUserLocked(userPoolID, username); err != nil {
		return nil, "", err
	}
	usernameKey := cognitoUserPoolsUsernameKey(username)
	allEvents := cloneCognitoUserPoolsAuthEvents(s.authEvents[userPoolID][usernameKey])
	start, err := parseCognitoUserPoolsNextToken(nextToken, len(allEvents))
	if err != nil {
		return nil, "", err
	}
	end := start + maxResults
	if end > len(allEvents) {
		end = len(allEvents)
	}
	out := allEvents[start:end]
	outToken := ""
	if end < len(allEvents) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) AdminUpdateAuthEventFeedback(userPoolID, username, eventID, feedbackValue string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	eventID = strings.TrimSpace(eventID)
	feedbackValue = strings.TrimSpace(feedbackValue)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if eventID == "" {
		return validationCognitoUserPools("EventId is required")
	}
	if feedbackValue == "" {
		return validationCognitoUserPools("FeedbackValue is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	usernameKey := cognitoUserPoolsUsernameKey(username)
	eventsByUser := s.authEvents[userPoolID]
	for idx, event := range eventsByUser[usernameKey] {
		if strings.EqualFold(event.EventID, eventID) {
			now := time.Now().UTC()
			event.FeedbackValue = feedbackValue
			event.FeedbackDate = &now
			eventsByUser[usernameKey][idx] = event
			s.authEvents[userPoolID] = eventsByUser
			return nil
		}
	}
	return notFoundCognitoUserPools("Auth event not found")
}

func (s *cognitoUserPoolsStore) UpdateAuthEventFeedback(accessToken, eventID, feedbackValue string) error {
	accessToken = strings.TrimSpace(accessToken)
	eventID = strings.TrimSpace(eventID)
	feedbackValue = strings.TrimSpace(feedbackValue)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if eventID == "" {
		return validationCognitoUserPools("EventId is required")
	}
	if feedbackValue == "" {
		return validationCognitoUserPools("FeedbackValue is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	usernameKey := cognitoUserPoolsUsernameKey(user.Username)
	eventsByUser := s.authEvents[user.UserPoolID]
	for idx, event := range eventsByUser[usernameKey] {
		if strings.EqualFold(event.EventID, eventID) {
			now := time.Now().UTC()
			event.FeedbackValue = feedbackValue
			event.FeedbackDate = &now
			eventsByUser[usernameKey][idx] = event
			s.authEvents[user.UserPoolID] = eventsByUser
			return nil
		}
	}
	return notFoundCognitoUserPools("Auth event not found")
}

func (s *cognitoUserPoolsStore) recordAuthEventLocked(userPoolID, username, eventType, eventResponse string) string {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" || username == "" {
		return ""
	}
	if s.authEvents[userPoolID] == nil {
		s.authEvents[userPoolID] = map[string][]cognitoUserPoolsAuthEventRecord{}
	}
	usernameKey := cognitoUserPoolsUsernameKey(username)
	now := time.Now().UTC()
	eventID := "event-" + randomHex(10)
	event := cognitoUserPoolsAuthEventRecord{
		EventID:       eventID,
		EventType:     strings.TrimSpace(eventType),
		EventResponse: strings.TrimSpace(eventResponse),
		EventRisk: map[string]any{
			"RiskDecision": "NoRisk",
			"RiskLevel":    "Low",
		},
		CreationDate: now,
	}
	if strings.EqualFold(eventResponse, "Failure") {
		event.EventRisk["RiskDecision"] = "AccountTakeover"
		event.EventRisk["RiskLevel"] = "Medium"
	}
	s.authEvents[userPoolID][usernameKey] = append([]cognitoUserPoolsAuthEventRecord{event}, s.authEvents[userPoolID][usernameKey]...)
	return eventID
}

func cloneCognitoUserPoolsAuthEvents(in []cognitoUserPoolsAuthEventRecord) []cognitoUserPoolsAuthEventRecord {
	if len(in) == 0 {
		return []cognitoUserPoolsAuthEventRecord{}
	}
	out := make([]cognitoUserPoolsAuthEventRecord, 0, len(in))
	for _, record := range in {
		copied := record
		copied.EventRisk = cloneCognitoUserPoolsMapAny(record.EventRisk)
		if record.FeedbackDate != nil {
			t := *record.FeedbackDate
			copied.FeedbackDate = &t
		}
		out = append(out, copied)
	}
	return out
}

func cloneCognitoUserPoolsRiskConfigurationRecord(in cognitoUserPoolsRiskConfigurationRecord) cognitoUserPoolsRiskConfigurationRecord {
	out := in
	out.CompromisedCredentialsRiskConfiguration = cloneCognitoUserPoolsMapAny(in.CompromisedCredentialsRiskConfiguration)
	out.AccountTakeoverRiskConfiguration = cloneCognitoUserPoolsMapAny(in.AccountTakeoverRiskConfiguration)
	out.RiskExceptionConfiguration = cloneCognitoUserPoolsMapAny(in.RiskExceptionConfiguration)
	return out
}

func cloneCognitoUserPoolsLogDeliveryConfigurationRecord(in cognitoUserPoolsLogDeliveryConfigurationRecord) cognitoUserPoolsLogDeliveryConfigurationRecord {
	out := in
	out.LogConfigurations = cloneCognitoUserPoolsSliceMapAny(in.LogConfigurations)
	return out
}

func cloneCognitoUserPoolsMapAny(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneCognitoUserPoolsAny(value)
	}
	return out
}

func cloneCognitoUserPoolsSliceMapAny(in []map[string]any) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneCognitoUserPoolsMapAny(item))
	}
	return out
}

func cloneCognitoUserPoolsAny(in any) any {
	switch typed := in.(type) {
	case map[string]any:
		return cloneCognitoUserPoolsMapAny(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneCognitoUserPoolsAny(item))
		}
		return out
	default:
		return typed
	}
}

func cognitoUserPoolsIdentityProviderKey(providerName string) string {
	return strings.ToLower(strings.TrimSpace(providerName))
}
