package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type cognitoUserPoolsCustomAttributeRecord struct {
	Name                       string
	AttributeDataType          string
	DeveloperOnlyAttribute     bool
	Mutable                    bool
	Required                   bool
	NumberAttributeConstraints map[string]string
	StringAttributeConstraints map[string]string
}

type cognitoUserPoolsUICustomizationRecord struct {
	UserPoolID      string
	ClientID        string
	CSS             string
	ImageFile       string
	ImageURL        string
	CSSVersion      string
	LastModifiedAt  time.Time
	CreationDateUTC time.Time
}

type cognitoUserPoolsManagedLoginBrandingRecord struct {
	UserPoolID               string
	ManagedLoginBrandingID   string
	ClientID                 string
	UseCognitoProvidedValues bool
	Settings                 map[string]any
	Assets                   []map[string]any
	CreationDate             time.Time
	LastModifiedDate         time.Time
}

type cognitoUserPoolsCreateManagedLoginBrandingInput struct {
	UserPoolID               string
	ClientID                 string
	UseCognitoProvidedValues bool
	Settings                 map[string]any
	Assets                   []map[string]any
}

type cognitoUserPoolsUpdateManagedLoginBrandingInput struct {
	UserPoolID                  string
	ManagedLoginBrandingID      string
	ClientID                    string
	ClientIDSet                 bool
	UseCognitoProvidedValues    bool
	UseCognitoProvidedValuesSet bool
	Settings                    map[string]any
	SettingsSet                 bool
	Assets                      []map[string]any
	AssetsSet                   bool
}

type cognitoUserPoolsTermsRecord struct {
	UserPoolID       string
	TermsID          string
	TermsName        string
	TermsDetails     map[string]any
	CreationDate     time.Time
	LastModifiedDate time.Time
}

type cognitoUserPoolsCreateTermsInput struct {
	UserPoolID   string
	TermsID      string
	TermsName    string
	TermsDetails map[string]any
}

type cognitoUserPoolsUpdateTermsInput struct {
	UserPoolID   string
	TermsID      string
	TermsName    string
	TermsNameSet bool
	TermsDetails map[string]any
}

type cognitoUserPoolsProviderUserIdentifier struct {
	ProviderName           string
	ProviderAttributeName  string
	ProviderAttributeValue string
}

type cognitoUserPoolsLinkedProviderRecord struct {
	Destination cognitoUserPoolsProviderUserIdentifier
	Source      cognitoUserPoolsProviderUserIdentifier
	Disabled    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type cognitoUserPoolsMFAOption struct {
	DeliveryMedium string
	AttributeName  string
}

func (s *cognitoUserPoolsStore) AddCustomAttributes(userPoolID string, attributes []cognitoUserPoolsCustomAttributeRecord) error {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	if s.customAttributes[userPoolID] == nil {
		s.customAttributes[userPoolID] = map[string]cognitoUserPoolsCustomAttributeRecord{}
	}
	for _, attribute := range attributes {
		name := strings.TrimSpace(attribute.Name)
		if name == "" {
			return validationCognitoUserPools("CustomAttributes.Name is required")
		}
		attribute.Name = name
		if strings.TrimSpace(attribute.AttributeDataType) == "" {
			attribute.AttributeDataType = "String"
		}
		s.customAttributes[userPoolID][strings.ToLower(name)] = cloneCognitoUserPoolsCustomAttributeRecord(attribute)
	}

	pool := s.pools[userPoolID]
	pool.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = pool
	return nil
}

func (s *cognitoUserPoolsStore) UpdateUserPoolDomain(userPoolID, domain string, managedLoginVersion *int, certificateARN string) (cognitoUserPoolsDomainRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	domain = strings.TrimSpace(domain)
	if userPoolID == "" {
		return cognitoUserPoolsDomainRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if domain == "" {
		return cognitoUserPoolsDomainRecord{}, validationCognitoUserPools("Domain is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.domains[strings.ToLower(domain)]
	if !ok || record.UserPoolID != userPoolID {
		return cognitoUserPoolsDomainRecord{}, notFoundCognitoUserPools("Domain not found")
	}
	if managedLoginVersion != nil && *managedLoginVersion > 0 {
		record.Version = *managedLoginVersion
	}
	if strings.TrimSpace(certificateARN) != "" {
		record.CloudFrontDomain = domain + ".custom." + cognitoUserPoolsRegionFromPoolID(userPoolID) + ".amazoncognito.com"
	}
	record.UpdatedAt = time.Now().UTC()
	s.domains[strings.ToLower(domain)] = record

	pool := s.pools[userPoolID]
	pool.UpdatedAt = record.UpdatedAt
	s.pools[userPoolID] = pool
	return record, nil
}

func (s *cognitoUserPoolsStore) SetUICustomization(userPoolID, clientID, css, imageFile string) (cognitoUserPoolsUICustomizationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsUICustomizationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsUICustomizationRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if clientID != "" {
		if _, ok := s.clients[userPoolID][clientID]; !ok {
			return cognitoUserPoolsUICustomizationRecord{}, notFoundCognitoUserPools("User pool client not found")
		}
	}
	if s.uiCustomizations[userPoolID] == nil {
		s.uiCustomizations[userPoolID] = map[string]cognitoUserPoolsUICustomizationRecord{}
	}

	key := cognitoUserPoolsUICustomizationKey(clientID)
	now := time.Now().UTC()
	record, exists := s.uiCustomizations[userPoolID][key]
	if !exists {
		record = cognitoUserPoolsUICustomizationRecord{
			UserPoolID:      userPoolID,
			ClientID:        clientID,
			CSSVersion:      "0",
			CreationDateUTC: now,
		}
	}
	record.UserPoolID = userPoolID
	record.ClientID = clientID
	record.CSS = css
	record.ImageFile = imageFile
	record.ImageURL = cognitoUserPoolsUICustomizationImageURL(userPoolID, clientID)
	record.LastModifiedAt = now
	record.CSSVersion = strconv.Itoa(cognitoUserPoolsVersionNumber(record.CSSVersion) + 1)
	s.uiCustomizations[userPoolID][key] = record
	return cloneCognitoUserPoolsUICustomizationRecord(record), nil
}

func (s *cognitoUserPoolsStore) GetUICustomization(userPoolID, clientID string) (cognitoUserPoolsUICustomizationRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsUICustomizationRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsUICustomizationRecord{}, notFoundCognitoUserPools("User pool not found")
	}

	key := cognitoUserPoolsUICustomizationKey(clientID)
	record, ok := s.uiCustomizations[userPoolID][key]
	if !ok {
		record = cognitoUserPoolsUICustomizationRecord{
			UserPoolID:      userPoolID,
			ClientID:        clientID,
			CSSVersion:      "1",
			ImageURL:        cognitoUserPoolsUICustomizationImageURL(userPoolID, clientID),
			CreationDateUTC: time.Now().UTC(),
			LastModifiedAt:  time.Now().UTC(),
		}
	}
	return cloneCognitoUserPoolsUICustomizationRecord(record), nil
}

func (s *cognitoUserPoolsStore) GetSigningCertificate(userPoolID string) (string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return "", validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return "", notFoundCognitoUserPools("User pool not found")
	}
	certificate := strings.TrimSpace(s.signingCerts[userPoolID])
	if certificate == "" {
		certificate = cognitoUserPoolsDefaultSigningCertificate(userPoolID)
		s.signingCerts[userPoolID] = certificate
	}
	return certificate, nil
}

func (s *cognitoUserPoolsStore) CreateManagedLoginBranding(input cognitoUserPoolsCreateManagedLoginBrandingInput) (cognitoUserPoolsManagedLoginBrandingRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	clientID := strings.TrimSpace(input.ClientID)
	if userPoolID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if clientID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("ClientId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if _, ok := s.clients[userPoolID][clientID]; !ok {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("User pool client not found")
	}
	if s.managedBrandings[userPoolID] == nil {
		s.managedBrandings[userPoolID] = map[string]cognitoUserPoolsManagedLoginBrandingRecord{}
	}
	now := time.Now().UTC()
	record := cognitoUserPoolsManagedLoginBrandingRecord{
		UserPoolID:               userPoolID,
		ManagedLoginBrandingID:   "branding-" + randomHex(10),
		ClientID:                 clientID,
		UseCognitoProvidedValues: input.UseCognitoProvidedValues,
		Settings:                 cloneCognitoUserPoolsMapAny(input.Settings),
		Assets:                   cloneCognitoUserPoolsSliceMapAny(input.Assets),
		CreationDate:             now,
		LastModifiedDate:         now,
	}
	s.managedBrandings[userPoolID][record.ManagedLoginBrandingID] = record
	return cloneCognitoUserPoolsManagedLoginBrandingRecord(record), nil
}

func (s *cognitoUserPoolsStore) UpdateManagedLoginBranding(input cognitoUserPoolsUpdateManagedLoginBrandingInput) (cognitoUserPoolsManagedLoginBrandingRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	brandingID := strings.TrimSpace(input.ManagedLoginBrandingID)
	if userPoolID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if brandingID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("ManagedLoginBrandingId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.describeManagedLoginBrandingLocked(userPoolID, brandingID)
	if err != nil {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, err
	}
	if input.ClientIDSet {
		clientID := strings.TrimSpace(input.ClientID)
		if clientID == "" {
			return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("ClientId is required")
		}
		if _, ok := s.clients[userPoolID][clientID]; !ok {
			return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("User pool client not found")
		}
		record.ClientID = clientID
	}
	if input.UseCognitoProvidedValuesSet {
		record.UseCognitoProvidedValues = input.UseCognitoProvidedValues
	}
	if input.SettingsSet {
		record.Settings = cloneCognitoUserPoolsMapAny(input.Settings)
	}
	if input.AssetsSet {
		record.Assets = cloneCognitoUserPoolsSliceMapAny(input.Assets)
	}
	record.LastModifiedDate = time.Now().UTC()
	s.managedBrandings[userPoolID][brandingID] = record
	return cloneCognitoUserPoolsManagedLoginBrandingRecord(record), nil
}

func (s *cognitoUserPoolsStore) DescribeManagedLoginBranding(userPoolID, brandingID string) (cognitoUserPoolsManagedLoginBrandingRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	brandingID = strings.TrimSpace(brandingID)
	if userPoolID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if brandingID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("ManagedLoginBrandingId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.describeManagedLoginBrandingLocked(userPoolID, brandingID)
}

func (s *cognitoUserPoolsStore) DescribeManagedLoginBrandingByClient(userPoolID, clientID string) (cognitoUserPoolsManagedLoginBrandingRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if userPoolID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if clientID == "" {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, validationCognitoUserPools("ClientId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	for _, record := range s.managedBrandings[userPoolID] {
		if strings.EqualFold(record.ClientID, clientID) {
			return cloneCognitoUserPoolsManagedLoginBrandingRecord(record), nil
		}
	}
	return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("Managed login branding not found")
}

func (s *cognitoUserPoolsStore) DeleteManagedLoginBranding(userPoolID, brandingID string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	brandingID = strings.TrimSpace(brandingID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if brandingID == "" {
		return validationCognitoUserPools("ManagedLoginBrandingId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	if _, ok := s.managedBrandings[userPoolID][brandingID]; !ok {
		return notFoundCognitoUserPools("Managed login branding not found")
	}
	delete(s.managedBrandings[userPoolID], brandingID)
	return nil
}

func (s *cognitoUserPoolsStore) CreateTerms(input cognitoUserPoolsCreateTermsInput) (cognitoUserPoolsTermsRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsTermsRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsTermsRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.terms[userPoolID] == nil {
		s.terms[userPoolID] = map[string]cognitoUserPoolsTermsRecord{}
	}

	termsID := strings.TrimSpace(input.TermsID)
	if termsID == "" {
		termsID = "terms-" + randomHex(10)
	}
	if _, ok := s.terms[userPoolID][termsID]; ok {
		return cognitoUserPoolsTermsRecord{}, conflictCognitoUserPools("Terms already exists")
	}

	now := time.Now().UTC()
	record := cognitoUserPoolsTermsRecord{
		UserPoolID:       userPoolID,
		TermsID:          termsID,
		TermsName:        strings.TrimSpace(input.TermsName),
		TermsDetails:     cloneCognitoUserPoolsMapAny(input.TermsDetails),
		CreationDate:     now,
		LastModifiedDate: now,
	}
	s.terms[userPoolID][termsID] = record
	return cloneCognitoUserPoolsTermsRecord(record), nil
}

func (s *cognitoUserPoolsStore) UpdateTerms(input cognitoUserPoolsUpdateTermsInput) (cognitoUserPoolsTermsRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsTermsRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, termsID, err := s.describeTermsLocked(userPoolID, input.TermsID)
	if err != nil {
		return cognitoUserPoolsTermsRecord{}, err
	}
	if input.TermsNameSet {
		record.TermsName = strings.TrimSpace(input.TermsName)
	}
	if len(input.TermsDetails) > 0 {
		record.TermsDetails = cloneCognitoUserPoolsMapAny(input.TermsDetails)
	}
	record.LastModifiedDate = time.Now().UTC()
	s.terms[userPoolID][termsID] = record
	return cloneCognitoUserPoolsTermsRecord(record), nil
}

func (s *cognitoUserPoolsStore) DescribeTerms(userPoolID, termsID string) (cognitoUserPoolsTermsRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsTermsRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, _, err := s.describeTermsLocked(userPoolID, termsID)
	if err != nil {
		return cognitoUserPoolsTermsRecord{}, err
	}
	return cloneCognitoUserPoolsTermsRecord(record), nil
}

func (s *cognitoUserPoolsStore) ListTerms(userPoolID string, maxResults int, nextToken string) ([]cognitoUserPoolsTermsRecord, string, error) {
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
	recordsByID := s.terms[userPoolID]
	ids := make([]string, 0, len(recordsByID))
	for id := range recordsByID {
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
	out := make([]cognitoUserPoolsTermsRecord, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, cloneCognitoUserPoolsTermsRecord(recordsByID[id]))
	}
	outToken := ""
	if end < len(ids) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) DeleteTerms(userPoolID, termsID string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	_, resolvedTermsID, err := s.describeTermsLocked(userPoolID, termsID)
	if err != nil {
		return err
	}
	delete(s.terms[userPoolID], resolvedTermsID)
	return nil
}

func (s *cognitoUserPoolsStore) AdminDisableProviderForUser(userPoolID string, user cognitoUserPoolsProviderUserIdentifier) error {
	userPoolID = strings.TrimSpace(userPoolID)
	user.ProviderName = strings.TrimSpace(user.ProviderName)
	user.ProviderAttributeName = strings.TrimSpace(user.ProviderAttributeName)
	user.ProviderAttributeValue = strings.TrimSpace(user.ProviderAttributeValue)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if user.ProviderName == "" {
		return validationCognitoUserPools("User.ProviderName is required")
	}
	if user.ProviderAttributeName == "" {
		return validationCognitoUserPools("User.ProviderAttributeName is required")
	}
	if user.ProviderAttributeValue == "" {
		return validationCognitoUserPools("User.ProviderAttributeValue is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}

	key := cognitoUserPoolsProviderUserKey(user)
	now := time.Now().UTC()
	for destinationKey, links := range s.providerLinks[userPoolID] {
		for idx, link := range links {
			if destinationKey == key || cognitoUserPoolsProviderUserKey(link.Source) == key {
				link.Disabled = true
				link.UpdatedAt = now
				links[idx] = link
			}
		}
		s.providerLinks[userPoolID][destinationKey] = links
	}
	return nil
}

func (s *cognitoUserPoolsStore) AdminLinkProviderForUser(
	userPoolID string,
	destinationUser, sourceUser cognitoUserPoolsProviderUserIdentifier,
) error {
	userPoolID = strings.TrimSpace(userPoolID)
	destinationUser = normalizeCognitoUserPoolsProviderUserIdentifier(destinationUser)
	sourceUser = normalizeCognitoUserPoolsProviderUserIdentifier(sourceUser)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if destinationUser.ProviderName == "" || destinationUser.ProviderAttributeName == "" || destinationUser.ProviderAttributeValue == "" {
		return validationCognitoUserPools("DestinationUser is required")
	}
	if sourceUser.ProviderName == "" || sourceUser.ProviderAttributeName == "" || sourceUser.ProviderAttributeValue == "" {
		return validationCognitoUserPools("SourceUser is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	if s.providerLinks[userPoolID] == nil {
		s.providerLinks[userPoolID] = map[string][]cognitoUserPoolsLinkedProviderRecord{}
	}

	destinationKey := cognitoUserPoolsProviderUserKey(destinationUser)
	now := time.Now().UTC()
	links := s.providerLinks[userPoolID][destinationKey]
	for idx, existing := range links {
		if cognitoUserPoolsProviderUserKey(existing.Source) == cognitoUserPoolsProviderUserKey(sourceUser) {
			existing.Disabled = false
			existing.UpdatedAt = now
			links[idx] = existing
			s.providerLinks[userPoolID][destinationKey] = links
			return nil
		}
	}
	links = append(links, cognitoUserPoolsLinkedProviderRecord{
		Destination: destinationUser,
		Source:      sourceUser,
		Disabled:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	s.providerLinks[userPoolID][destinationKey] = links
	return nil
}

func (s *cognitoUserPoolsStore) AdminSetUserSettings(userPoolID, username string, mfaOptions []cognitoUserPoolsMFAOption) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	user.MFAOptions = cloneCognitoUserPoolsMFAOptions(mfaOptions)
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) SetUserSettings(accessToken string, mfaOptions []cognitoUserPoolsMFAOption) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	user.MFAOptions = cloneCognitoUserPoolsMFAOptions(mfaOptions)
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) describeManagedLoginBrandingLocked(userPoolID, brandingID string) (cognitoUserPoolsManagedLoginBrandingRecord, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	record, ok := s.managedBrandings[userPoolID][brandingID]
	if !ok {
		return cognitoUserPoolsManagedLoginBrandingRecord{}, notFoundCognitoUserPools("Managed login branding not found")
	}
	return cloneCognitoUserPoolsManagedLoginBrandingRecord(record), nil
}

func (s *cognitoUserPoolsStore) describeTermsLocked(userPoolID, termsID string) (cognitoUserPoolsTermsRecord, string, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsTermsRecord{}, "", notFoundCognitoUserPools("User pool not found")
	}
	recordsByID := s.terms[userPoolID]
	resolvedTermsID := strings.TrimSpace(termsID)
	if resolvedTermsID == "" {
		if len(recordsByID) == 0 {
			return cognitoUserPoolsTermsRecord{}, "", notFoundCognitoUserPools("Terms not found")
		}
		ids := make([]string, 0, len(recordsByID))
		for id := range recordsByID {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		resolvedTermsID = ids[0]
	}
	record, ok := recordsByID[resolvedTermsID]
	if !ok {
		return cognitoUserPoolsTermsRecord{}, "", notFoundCognitoUserPools("Terms not found")
	}
	return cloneCognitoUserPoolsTermsRecord(record), resolvedTermsID, nil
}

func cloneCognitoUserPoolsCustomAttributeRecord(in cognitoUserPoolsCustomAttributeRecord) cognitoUserPoolsCustomAttributeRecord {
	out := in
	out.NumberAttributeConstraints = cloneStringMap(in.NumberAttributeConstraints)
	out.StringAttributeConstraints = cloneStringMap(in.StringAttributeConstraints)
	return out
}

func cloneCognitoUserPoolsUICustomizationRecord(in cognitoUserPoolsUICustomizationRecord) cognitoUserPoolsUICustomizationRecord {
	return in
}

func cloneCognitoUserPoolsManagedLoginBrandingRecord(in cognitoUserPoolsManagedLoginBrandingRecord) cognitoUserPoolsManagedLoginBrandingRecord {
	out := in
	out.Settings = cloneCognitoUserPoolsMapAny(in.Settings)
	out.Assets = cloneCognitoUserPoolsSliceMapAny(in.Assets)
	return out
}

func cloneCognitoUserPoolsTermsRecord(in cognitoUserPoolsTermsRecord) cognitoUserPoolsTermsRecord {
	out := in
	out.TermsDetails = cloneCognitoUserPoolsMapAny(in.TermsDetails)
	return out
}

func cloneCognitoUserPoolsMFAOptions(in []cognitoUserPoolsMFAOption) []cognitoUserPoolsMFAOption {
	if len(in) == 0 {
		return []cognitoUserPoolsMFAOption{}
	}
	out := make([]cognitoUserPoolsMFAOption, len(in))
	copy(out, in)
	return out
}

func normalizeCognitoUserPoolsProviderUserIdentifier(in cognitoUserPoolsProviderUserIdentifier) cognitoUserPoolsProviderUserIdentifier {
	return cognitoUserPoolsProviderUserIdentifier{
		ProviderName:           strings.TrimSpace(in.ProviderName),
		ProviderAttributeName:  strings.TrimSpace(in.ProviderAttributeName),
		ProviderAttributeValue: strings.TrimSpace(in.ProviderAttributeValue),
	}
}

func cognitoUserPoolsProviderUserKey(in cognitoUserPoolsProviderUserIdentifier) string {
	return strings.ToLower(strings.Join(
		[]string{
			strings.TrimSpace(in.ProviderName),
			strings.TrimSpace(in.ProviderAttributeName),
			strings.TrimSpace(in.ProviderAttributeValue),
		},
		"|",
	))
}

func cognitoUserPoolsUICustomizationKey(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "__default__"
	}
	return strings.ToLower(clientID)
}

func cognitoUserPoolsUICustomizationImageURL(userPoolID, clientID string) string {
	userPoolID = strings.TrimSpace(userPoolID)
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "https://cognito.stackyard.local/" + userPoolID + "/ui/default/logo.png"
	}
	return "https://cognito.stackyard.local/" + userPoolID + "/ui/" + clientID + "/logo.png"
}

func cognitoUserPoolsVersionNumber(version string) int {
	value, err := strconv.Atoi(strings.TrimSpace(version))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func cognitoUserPoolsDefaultSigningCertificate(userPoolID string) string {
	return "-----BEGIN CERTIFICATE-----\n" +
		"MIIBZjCCAQ2gAwIBAgIUDGVtb3N0YWNr\n" +
		"eWFyZC11c2VyLXBvb2wt" + strings.ToUpper(randomHex(6)) + "\n" +
		"MAoGCCqGSM49BAMCMBUxEzARBgNVBAMMCnN0YWNreWFyZDAeFw0yNTAxMDEwMDAwMDBa\n" +
		"Fw0zNTAxMDEwMDAwMDBaMBUxEzARBgNVBAMMCnN0YWNreWFyZDBZMBMGByqGSM49AgEG\n" +
		"CCqGSM49AwEHA0IABJjE7rU0dQ6F0g8s8tBo6l1jVJxF9mDkZkQH5tDw4aR0u8Bq\n" +
		"Vw4eV5sF6f2gQ8d9n+J1Lk9Y2x7K6vA8i6hQ7tKjUzBRMB0GA1UdDgQWBBRzdGFj\n" +
		"a3lhcmQtdXNlci1wb29sLWtleTAfBgNVHSMEGDAWgBRzdGFja3lhcmQtdXNlci1w\n" +
		"b29sLWtleTAPBgNVHRMBAf8EBTADAQH/MAoGCCqGSM49BAMCA0cAMEQCIBW4dV0N\n" +
		"a0hYbWJQeGZsY2x4ZG1vY2tzaWduYXR1cmV0ZXN0AiA4Y3ZtY2Y0dGx4aW9hY2N0\n" +
		"b2tlbmNlcnRzaWduYXR1cmU=\n" +
		"-----END CERTIFICATE-----\n"
}
