package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

const cognitoIdentityMaxResults = 60

type cognitoIdentityGetOpenIDTokenForDeveloperIdentityInput struct {
	IdentityPoolID string
	IdentityID     string
	Logins         map[string]string
}

type cognitoIdentityLookupDeveloperIdentityInput struct {
	IdentityPoolID          string
	IdentityID              string
	DeveloperUserIdentifier string
	NextToken               string
	MaxResults              int
}

type cognitoIdentityLookupDeveloperIdentityOutput struct {
	IdentityID                  string
	DeveloperUserIdentifierList []string
	NextToken                   string
}

type cognitoIdentityMergeDeveloperIdentitiesInput struct {
	IdentityPoolID                    string
	DeveloperProviderName             string
	DestinationUserIdentifierForMerge string
	SourceUserIdentifier              string
}

type cognitoIdentityUnlinkDeveloperIdentityInput struct {
	IdentityPoolID          string
	IdentityID              string
	DeveloperProviderName   string
	DeveloperUserIdentifier string
}

type cognitoIdentityUnprocessedIdentity struct {
	IdentityID string
	ErrorCode  string
}

func (s *cognitoIdentityStore) GetOpenIDTokenForDeveloperIdentity(input cognitoIdentityGetOpenIDTokenForDeveloperIdentityInput) (string, string, error) {
	poolID := strings.TrimSpace(input.IdentityPoolID)
	if poolID == "" {
		return "", "", validationCognitoIdentity("IdentityPoolId is required")
	}
	if len(input.Logins) == 0 {
		return "", "", validationCognitoIdentity("Logins is required")
	}

	logins := normalizeCognitoIdentityStringMap(input.Logins)
	if len(logins) == 0 {
		return "", "", validationCognitoIdentity("Logins is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return "", "", notFoundCognitoIdentity("IdentityPool not found")
	}

	identityID := strings.TrimSpace(input.IdentityID)
	if identityID != "" {
		record, ok := s.identities[identityID]
		if !ok || record.IdentityPoolID != poolID {
			return "", "", notFoundCognitoIdentity("Identity not found")
		}
	} else {
		byPool := s.developerUserToIdentity[poolID]
		for provider, developerUser := range logins {
			key := cognitoIdentityDeveloperUserKey(provider, developerUser)
			if existingID, linked := byPool[key]; linked {
				if identityID == "" {
					identityID = existingID
					continue
				}
				if identityID != existingID {
					return "", "", conflictCognitoIdentity("logins are linked to different identities")
				}
			}
		}
		if identityID == "" {
			record := s.createIdentityLocked(poolID, pool, logins)
			identityID = record.IdentityID
		}
	}

	record, ok := s.identities[identityID]
	if !ok || record.IdentityPoolID != poolID {
		return "", "", notFoundCognitoIdentity("Identity not found")
	}

	byPool := s.developerUserToIdentity[poolID]
	if byPool == nil {
		byPool = map[string]string{}
		s.developerUserToIdentity[poolID] = byPool
	}
	for provider, developerUser := range logins {
		key := cognitoIdentityDeveloperUserKey(provider, developerUser)
		if existingID, linked := byPool[key]; linked && existingID != identityID {
			return "", "", developerUserAlreadyRegisteredCognitoIdentity("developer user is already registered")
		}
	}

	mergedLogins := cloneStringMap(record.Logins)
	if mergedLogins == nil {
		mergedLogins = map[string]string{}
	}
	for provider, developerUser := range logins {
		key := cognitoIdentityDeveloperUserKey(provider, developerUser)
		byPool[key] = identityID
		mergedLogins[provider] = developerUser
	}

	identityDeveloperUsers := s.identityDeveloperUsers[identityID]
	if identityDeveloperUsers == nil {
		identityDeveloperUsers = map[string]struct{}{}
		s.identityDeveloperUsers[identityID] = identityDeveloperUsers
	}
	for provider, developerUser := range logins {
		identityDeveloperUsers[cognitoIdentityDeveloperUserKey(provider, developerUser)] = struct{}{}
	}

	s.setIdentityLoginsLocked(identityID, mergedLogins)
	return identityID, "openid-dev-token-" + randomHex(16), nil
}

func (s *cognitoIdentityStore) LookupDeveloperIdentity(input cognitoIdentityLookupDeveloperIdentityInput) (cognitoIdentityLookupDeveloperIdentityOutput, error) {
	poolID := strings.TrimSpace(input.IdentityPoolID)
	if poolID == "" {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, validationCognitoIdentity("IdentityPoolId is required")
	}
	identityID := strings.TrimSpace(input.IdentityID)
	developerUserIdentifier := strings.TrimSpace(input.DeveloperUserIdentifier)
	if identityID == "" && developerUserIdentifier == "" {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, validationCognitoIdentity("IdentityId or DeveloperUserIdentifier is required")
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = cognitoIdentityMaxResults
	}
	if maxResults > cognitoIdentityMaxResults {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, validationCognitoIdentity("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, notFoundCognitoIdentity("IdentityPool not found")
	}

	if developerUserIdentifier != "" {
		resolvedIdentityID, found, err := s.lookupDeveloperIdentityIDLocked(poolID, pool.DeveloperProviderName, developerUserIdentifier)
		if err != nil {
			return cognitoIdentityLookupDeveloperIdentityOutput{}, err
		}
		if !found {
			return cognitoIdentityLookupDeveloperIdentityOutput{}, notFoundCognitoIdentity("Developer user not found")
		}
		if identityID != "" && identityID != resolvedIdentityID {
			return cognitoIdentityLookupDeveloperIdentityOutput{}, conflictCognitoIdentity("developer user is linked to a different identity")
		}
		return cognitoIdentityLookupDeveloperIdentityOutput{
			IdentityID:                  resolvedIdentityID,
			DeveloperUserIdentifierList: []string{developerUserIdentifier},
		}, nil
	}

	record, ok := s.identities[identityID]
	if !ok || record.IdentityPoolID != poolID {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, notFoundCognitoIdentity("Identity not found")
	}

	developerUsers := s.developerUsersForIdentityLocked(identityID)
	start, err := parseCognitoIdentityNextToken(input.NextToken, len(developerUsers))
	if err != nil {
		return cognitoIdentityLookupDeveloperIdentityOutput{}, err
	}

	end := start + maxResults
	if end > len(developerUsers) {
		end = len(developerUsers)
	}

	out := cognitoIdentityLookupDeveloperIdentityOutput{
		IdentityID:                  identityID,
		DeveloperUserIdentifierList: cloneStringSlice(developerUsers[start:end]),
	}
	if end < len(developerUsers) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *cognitoIdentityStore) MergeDeveloperIdentities(input cognitoIdentityMergeDeveloperIdentitiesInput) (string, error) {
	poolID := strings.TrimSpace(input.IdentityPoolID)
	if poolID == "" {
		return "", validationCognitoIdentity("IdentityPoolId is required")
	}
	providerName := strings.TrimSpace(input.DeveloperProviderName)
	if providerName == "" {
		return "", validationCognitoIdentity("DeveloperProviderName is required")
	}
	destinationUser := strings.TrimSpace(input.DestinationUserIdentifierForMerge)
	if destinationUser == "" {
		return "", validationCognitoIdentity("DestinationUserIdentifierForMerge is required")
	}
	sourceUser := strings.TrimSpace(input.SourceUserIdentifier)
	if sourceUser == "" {
		return "", validationCognitoIdentity("SourceUserIdentifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[poolID]; !ok {
		return "", notFoundCognitoIdentity("IdentityPool not found")
	}

	byPool := s.developerUserToIdentity[poolID]
	if byPool == nil {
		return "", notFoundCognitoIdentity("Developer user not found")
	}
	destKey := cognitoIdentityDeveloperUserKey(providerName, destinationUser)
	sourceKey := cognitoIdentityDeveloperUserKey(providerName, sourceUser)

	destinationIdentityID, ok := byPool[destKey]
	if !ok {
		return "", notFoundCognitoIdentity("Developer user not found")
	}
	sourceIdentityID, ok := byPool[sourceKey]
	if !ok {
		return "", notFoundCognitoIdentity("Developer user not found")
	}
	if destinationIdentityID == sourceIdentityID {
		return destinationIdentityID, nil
	}

	destination, ok := s.identities[destinationIdentityID]
	if !ok || destination.IdentityPoolID != poolID {
		return "", notFoundCognitoIdentity("Identity not found")
	}
	source, ok := s.identities[sourceIdentityID]
	if !ok || source.IdentityPoolID != poolID {
		return "", notFoundCognitoIdentity("Identity not found")
	}

	mergedLogins := cloneStringMap(destination.Logins)
	if mergedLogins == nil {
		mergedLogins = map[string]string{}
	}
	for provider, developerUser := range source.Logins {
		mergedLogins[provider] = developerUser
	}
	s.setIdentityLoginsLocked(destinationIdentityID, mergedLogins)

	destDeveloperUsers := s.identityDeveloperUsers[destinationIdentityID]
	if destDeveloperUsers == nil {
		destDeveloperUsers = map[string]struct{}{}
		s.identityDeveloperUsers[destinationIdentityID] = destDeveloperUsers
	}
	for key := range s.identityDeveloperUsers[sourceIdentityID] {
		byPool[key] = destinationIdentityID
		destDeveloperUsers[key] = struct{}{}
	}
	delete(s.identityDeveloperUsers, sourceIdentityID)

	s.removeIdentityLocked(sourceIdentityID)
	return destinationIdentityID, nil
}

func (s *cognitoIdentityStore) UnlinkDeveloperIdentity(input cognitoIdentityUnlinkDeveloperIdentityInput) error {
	poolID := strings.TrimSpace(input.IdentityPoolID)
	if poolID == "" {
		return validationCognitoIdentity("IdentityPoolId is required")
	}
	identityID := strings.TrimSpace(input.IdentityID)
	if identityID == "" {
		return validationCognitoIdentity("IdentityId is required")
	}
	providerName := strings.TrimSpace(input.DeveloperProviderName)
	if providerName == "" {
		return validationCognitoIdentity("DeveloperProviderName is required")
	}
	developerUserIdentifier := strings.TrimSpace(input.DeveloperUserIdentifier)
	if developerUserIdentifier == "" {
		return validationCognitoIdentity("DeveloperUserIdentifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pools[poolID]; !ok {
		return notFoundCognitoIdentity("IdentityPool not found")
	}
	record, ok := s.identities[identityID]
	if !ok || record.IdentityPoolID != poolID {
		return notFoundCognitoIdentity("Identity not found")
	}

	key := cognitoIdentityDeveloperUserKey(providerName, developerUserIdentifier)
	byPool := s.developerUserToIdentity[poolID]
	linkedIdentityID, linked := byPool[key]
	if !linked {
		return notFoundCognitoIdentity("Developer user not found")
	}
	if linkedIdentityID != identityID {
		return conflictCognitoIdentity("developer user is linked to a different identity")
	}

	delete(byPool, key)
	if len(byPool) == 0 {
		delete(s.developerUserToIdentity, poolID)
	}
	if byIdentity := s.identityDeveloperUsers[identityID]; byIdentity != nil {
		delete(byIdentity, key)
		if len(byIdentity) == 0 {
			delete(s.identityDeveloperUsers, identityID)
		}
	}

	logins := cloneStringMap(record.Logins)
	if logins != nil {
		if linkedUser := logins[providerName]; linkedUser == developerUserIdentifier {
			delete(logins, providerName)
			s.setIdentityLoginsLocked(identityID, logins)
		}
	}
	return nil
}

func (s *cognitoIdentityStore) DeleteIdentities(identityIDs []string) ([]cognitoIdentityUnprocessedIdentity, error) {
	if len(identityIDs) == 0 {
		return nil, validationCognitoIdentity("IdentityIdsToDelete is required")
	}
	if len(identityIDs) > 60 {
		return nil, validationCognitoIdentity("IdentityIdsToDelete must have 60 or fewer items")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	unprocessed := make([]cognitoIdentityUnprocessedIdentity, 0)
	for _, raw := range identityIDs {
		identityID := strings.TrimSpace(raw)
		if identityID == "" {
			unprocessed = append(unprocessed, cognitoIdentityUnprocessedIdentity{
				IdentityID: raw,
				ErrorCode:  "InvalidParameterException",
			})
			continue
		}
		if _, ok := s.identities[identityID]; !ok {
			unprocessed = append(unprocessed, cognitoIdentityUnprocessedIdentity{
				IdentityID: identityID,
				ErrorCode:  "ResourceNotFoundException",
			})
			continue
		}
		s.removeIdentityLocked(identityID)
	}
	return unprocessed, nil
}

func (s *cognitoIdentityStore) GetIdentityPoolRoles(identityPoolID string) (map[string]string, map[string]cognitoIdentityRoleMapping, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, nil, validationCognitoIdentity("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return nil, nil, notFoundCognitoIdentity("IdentityPool not found")
	}
	return cloneStringMap(record.Roles), cloneCognitoIdentityRoleMappings(record.RoleMappings), nil
}

func (s *cognitoIdentityStore) SetIdentityPoolRoles(identityPoolID string, roles map[string]string, roleMappings map[string]cognitoIdentityRoleMapping) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return validationCognitoIdentity("IdentityPoolId is required")
	}
	if len(roles) == 0 {
		return validationCognitoIdentity("Roles is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return notFoundCognitoIdentity("IdentityPool not found")
	}
	record.Roles = cloneStringMap(roles)
	record.RoleMappings = cloneCognitoIdentityRoleMappings(roleMappings)
	record.LastModifiedDate = time.Now().UTC()
	s.pools[identityPoolID] = record
	return nil
}

func (s *cognitoIdentityStore) GetPrincipalTagAttributeMap(identityPoolID, identityProviderName string) (cognitoIdentityPrincipalTagAttributeMap, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return cognitoIdentityPrincipalTagAttributeMap{}, validationCognitoIdentity("IdentityPoolId is required")
	}
	identityProviderName = strings.TrimSpace(identityProviderName)
	if identityProviderName == "" {
		return cognitoIdentityPrincipalTagAttributeMap{}, validationCognitoIdentity("IdentityProviderName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return cognitoIdentityPrincipalTagAttributeMap{}, notFoundCognitoIdentity("IdentityPool not found")
	}
	if record.PrincipalTagAttributeMaps == nil {
		return cognitoIdentityPrincipalTagAttributeMap{}, notFoundCognitoIdentity("Principal tag attribute map not found")
	}
	value, ok := record.PrincipalTagAttributeMaps[identityProviderName]
	if !ok {
		return cognitoIdentityPrincipalTagAttributeMap{}, notFoundCognitoIdentity("Principal tag attribute map not found")
	}
	return cognitoIdentityPrincipalTagAttributeMap{
		IdentityPoolID:       value.IdentityPoolID,
		IdentityProviderName: value.IdentityProviderName,
		PrincipalTags:        cloneStringMap(value.PrincipalTags),
		UseDefaults:          value.UseDefaults,
	}, nil
}

func (s *cognitoIdentityStore) SetPrincipalTagAttributeMap(value cognitoIdentityPrincipalTagAttributeMap) (cognitoIdentityPrincipalTagAttributeMap, error) {
	identityPoolID := strings.TrimSpace(value.IdentityPoolID)
	if identityPoolID == "" {
		return cognitoIdentityPrincipalTagAttributeMap{}, validationCognitoIdentity("IdentityPoolId is required")
	}
	identityProviderName := strings.TrimSpace(value.IdentityProviderName)
	if identityProviderName == "" {
		return cognitoIdentityPrincipalTagAttributeMap{}, validationCognitoIdentity("IdentityProviderName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return cognitoIdentityPrincipalTagAttributeMap{}, notFoundCognitoIdentity("IdentityPool not found")
	}
	if record.PrincipalTagAttributeMaps == nil {
		record.PrincipalTagAttributeMaps = map[string]cognitoIdentityPrincipalTagAttributeMap{}
	}
	record.PrincipalTagAttributeMaps[identityProviderName] = cognitoIdentityPrincipalTagAttributeMap{
		IdentityPoolID:       identityPoolID,
		IdentityProviderName: identityProviderName,
		PrincipalTags:        cloneStringMap(value.PrincipalTags),
		UseDefaults:          value.UseDefaults,
	}
	record.LastModifiedDate = time.Now().UTC()
	s.pools[identityPoolID] = record
	return record.PrincipalTagAttributeMaps[identityProviderName], nil
}

func (s *cognitoIdentityStore) TagIdentityPool(identityPoolID string, tags map[string]string) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return validationCognitoIdentity("IdentityPoolId is required")
	}
	if len(tags) == 0 {
		return validationCognitoIdentity("Tags is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return notFoundCognitoIdentity("IdentityPool not found")
	}

	existing := cloneStringMap(record.IdentityPoolTags)
	if existing == nil {
		existing = map[string]string{}
	}
	for key, value := range tags {
		existing[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if len(existing) > 50 {
		return validationCognitoIdentity("resource tags must have 50 or fewer entries")
	}

	record.IdentityPoolTags = existing
	record.LastModifiedDate = time.Now().UTC()
	s.pools[identityPoolID] = record
	return nil
}

func (s *cognitoIdentityStore) UntagIdentityPool(identityPoolID string, tagKeys []string) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return validationCognitoIdentity("IdentityPoolId is required")
	}
	if len(tagKeys) == 0 {
		return validationCognitoIdentity("TagKeys is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return notFoundCognitoIdentity("IdentityPool not found")
	}
	if len(record.IdentityPoolTags) == 0 {
		return nil
	}

	tags := cloneStringMap(record.IdentityPoolTags)
	for _, key := range tagKeys {
		delete(tags, strings.TrimSpace(key))
	}
	record.IdentityPoolTags = tags
	record.LastModifiedDate = time.Now().UTC()
	s.pools[identityPoolID] = record
	return nil
}

func (s *cognitoIdentityStore) ListIdentityPoolTags(identityPoolID string) (map[string]string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, validationCognitoIdentity("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.pools[identityPoolID]
	if !ok {
		return nil, notFoundCognitoIdentity("IdentityPool not found")
	}
	return cloneStringMap(record.IdentityPoolTags), nil
}

func (s *cognitoIdentityStore) UnlinkIdentity(identityID string, logins map[string]string, loginsToRemove []string) error {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return validationCognitoIdentity("IdentityId is required")
	}
	if len(logins) == 0 {
		return validationCognitoIdentity("Logins is required")
	}
	if len(loginsToRemove) == 0 {
		return validationCognitoIdentity("LoginsToRemove is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.identities[identityID]
	if !ok {
		return notFoundCognitoIdentity("Identity not found")
	}

	matched := false
	for provider, token := range logins {
		if strings.TrimSpace(record.Logins[provider]) == strings.TrimSpace(token) {
			matched = true
			break
		}
	}
	if !matched {
		return notAuthorizedCognitoIdentity("provided logins are not linked to the identity")
	}

	updated := cloneStringMap(record.Logins)
	if updated == nil {
		updated = map[string]string{}
	}

	for _, loginProvider := range loginsToRemove {
		provider := strings.TrimSpace(loginProvider)
		if provider == "" {
			continue
		}
		developerUser := strings.TrimSpace(updated[provider])
		delete(updated, provider)

		if developerUser == "" {
			continue
		}
		key := cognitoIdentityDeveloperUserKey(provider, developerUser)
		if byPool := s.developerUserToIdentity[record.IdentityPoolID]; byPool != nil {
			if linkedIdentityID, ok := byPool[key]; ok && linkedIdentityID == identityID {
				delete(byPool, key)
			}
			if len(byPool) == 0 {
				delete(s.developerUserToIdentity, record.IdentityPoolID)
			}
		}
		if byIdentity := s.identityDeveloperUsers[identityID]; byIdentity != nil {
			delete(byIdentity, key)
			if len(byIdentity) == 0 {
				delete(s.identityDeveloperUsers, identityID)
			}
		}
	}

	s.setIdentityLoginsLocked(identityID, updated)
	return nil
}

func (s *cognitoIdentityStore) createIdentityLocked(identityPoolID string, pool cognitoIdentityPoolRecord, logins map[string]string) cognitoIdentityRecord {
	now := time.Now().UTC()
	record := cognitoIdentityRecord{
		IdentityID:     cognitoIdentityRegionFromPoolID(pool.IdentityPoolID) + ":" + cognitoIdentityUUID(),
		IdentityPoolID: identityPoolID,
		Logins:         cloneStringMap(logins),
		CreationDate:   now,
		LastModified:   now,
	}
	s.identities[record.IdentityID] = record
	s.poolIdentityIDs[identityPoolID] = append(s.poolIdentityIDs[identityPoolID], record.IdentityID)
	s.setIdentityLoginIndexLocked(record)
	return record
}

func (s *cognitoIdentityStore) setIdentityLoginsLocked(identityID string, logins map[string]string) {
	record, ok := s.identities[identityID]
	if !ok {
		return
	}
	s.deleteIdentityLoginIndexLocked(record)
	record.Logins = cloneStringMap(logins)
	record.LastModified = time.Now().UTC()
	s.identities[identityID] = record
	s.setIdentityLoginIndexLocked(record)
}

func (s *cognitoIdentityStore) setIdentityLoginIndexLocked(record cognitoIdentityRecord) {
	if _, ok := s.identityByPoolLogins[record.IdentityPoolID]; !ok {
		s.identityByPoolLogins[record.IdentityPoolID] = map[string]string{}
	}
	s.identityByPoolLogins[record.IdentityPoolID][canonicalCognitoIdentityLoginsKey(record.Logins)] = record.IdentityID
}

func (s *cognitoIdentityStore) deleteIdentityLoginIndexLocked(record cognitoIdentityRecord) {
	byPool := s.identityByPoolLogins[record.IdentityPoolID]
	if byPool == nil {
		return
	}
	delete(byPool, canonicalCognitoIdentityLoginsKey(record.Logins))
	if len(byPool) == 0 {
		delete(s.identityByPoolLogins, record.IdentityPoolID)
	}
}

func (s *cognitoIdentityStore) lookupDeveloperIdentityIDLocked(poolID, providerHint, developerUserIdentifier string) (string, bool, error) {
	byPool := s.developerUserToIdentity[poolID]
	if byPool == nil {
		return "", false, nil
	}

	if provider := strings.TrimSpace(providerHint); provider != "" {
		identityID, ok := byPool[cognitoIdentityDeveloperUserKey(provider, developerUserIdentifier)]
		return identityID, ok, nil
	}

	matchIdentityID := ""
	for key, identityID := range byPool {
		_, user := cognitoIdentityDeveloperUserKeyParts(key)
		if user != developerUserIdentifier {
			continue
		}
		if matchIdentityID == "" {
			matchIdentityID = identityID
			continue
		}
		if matchIdentityID != identityID {
			return "", false, conflictCognitoIdentity("developer user is linked to multiple identities")
		}
	}
	if matchIdentityID == "" {
		return "", false, nil
	}
	return matchIdentityID, true, nil
}

func (s *cognitoIdentityStore) developerUsersForIdentityLocked(identityID string) []string {
	byIdentity := s.identityDeveloperUsers[identityID]
	if len(byIdentity) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for key := range byIdentity {
		_, developerUser := cognitoIdentityDeveloperUserKeyParts(key)
		if developerUser == "" {
			continue
		}
		set[developerUser] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for developerUser := range set {
		out = append(out, developerUser)
	}
	sort.Strings(out)
	return out
}

func normalizeCognitoIdentityStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for rawKey, rawValue := range in {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cognitoIdentityDeveloperUserKey(providerName, developerUserIdentifier string) string {
	return strings.TrimSpace(providerName) + "\x1f" + strings.TrimSpace(developerUserIdentifier)
}

func cognitoIdentityDeveloperUserKeyParts(value string) (string, string) {
	parts := strings.SplitN(value, "\x1f", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(value), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
