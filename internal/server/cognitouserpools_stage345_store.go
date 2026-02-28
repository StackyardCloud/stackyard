package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type cognitoUserPoolsUserRecord struct {
	UserPoolID                 string
	Username                   string
	Sub                        string
	Attributes                 map[string]string
	AttributeVerificationCodes map[string]string
	Enabled                    bool
	UserStatus                 string
	Password                   string
	PreferredMFA               string
	MFAOptions                 []cognitoUserPoolsMFAOption
	SoftwareTokenEnabled       bool
	SoftwareTokenVerified      bool
	Groups                     map[string]struct{}
	Devices                    map[string]cognitoUserPoolsDeviceRecord
	WebAuthnCredentials        map[string]cognitoUserPoolsWebAuthnCredentialRecord
	ConfirmationCode           string
	ResetCode                  string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type cognitoUserPoolsDeviceRecord struct {
	DeviceKey                 string
	DeviceName                string
	DeviceRememberedStatus    string
	DeviceCreateDate          time.Time
	DeviceLastAuthenticatedAt time.Time
	DeviceLastModifiedDate    time.Time
}

type cognitoUserPoolsGroupRecord struct {
	UserPoolID     string
	GroupName      string
	Description    string
	RoleARN        string
	Precedence     *int
	CreatedAt      time.Time
	LastModifiedAt time.Time
}

type cognitoUserPoolsImportJobRecord struct {
	UserPoolID            string
	JobID                 string
	JobName               string
	CloudWatchLogsRoleArn string
	Status                string
	PreSignedURL          string
	CreatedAt             time.Time
	LastModifiedAt        time.Time
	StartedAt             *time.Time
	CompletedAt           *time.Time
}

type cognitoUserPoolsAccessTokenRecord struct {
	Token      string
	UserPoolID string
	ClientID   string
	Username   string
	ExpiresAt  time.Time
	Revoked    bool
}

type cognitoUserPoolsRefreshTokenRecord struct {
	Token      string
	UserPoolID string
	ClientID   string
	Username   string
	ExpiresAt  time.Time
	Revoked    bool
}

type cognitoUserPoolsSessionRecord struct {
	SessionID     string
	UserPoolID    string
	ClientID      string
	Username      string
	SessionKind   string
	SecretCode    string
	ChallengeName string
	Challenge     string
	ExpiresAt     time.Time
}

type cognitoUserPoolsAuthResult struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
}

type cognitoUserPoolsAuthFlowResult struct {
	AuthenticationResult *cognitoUserPoolsAuthResult
	ChallengeName        string
	ChallengeParameters  map[string]string
	Session              string
}

type cognitoUserPoolsAdminCreateUserInput struct {
	UserPoolID         string
	Username           string
	TemporaryPassword  string
	UserAttributes     map[string]string
	DesiredDelivery    []string
	ForceAliasCreation bool
}

type cognitoUserPoolsCreateGroupInput struct {
	UserPoolID  string
	GroupName   string
	Description string
	RoleARN     string
	Precedence  *int
}

type cognitoUserPoolsUpdateGroupInput struct {
	UserPoolID     string
	GroupName      string
	Description    string
	DescriptionSet bool
	RoleARN        string
	RoleARNSet     bool
	Precedence     *int
	PrecedenceSet  bool
}

type cognitoUserPoolsInitiateAuthInput struct {
	UserPoolID     string
	ClientID       string
	AuthFlow       string
	AuthParameters map[string]string
}

type cognitoUserPoolsRespondAuthChallengeInput struct {
	UserPoolID         string
	ClientID           string
	ChallengeName      string
	ChallengeResponses map[string]string
	Session            string
}

type cognitoUserPoolsSetMFAPreferenceInput struct {
	UserPoolID        string
	Username          string
	SoftwareEnabled   *bool
	SoftwarePreferred *bool
	ByAccessToken     string
}

type cognitoUserPoolsSetPoolMFAConfigInput struct {
	UserPoolID                  string
	MFAConfiguration            string
	SoftwareTokenMFAEnabled     *bool
	WebAuthnRelyingPartyID      string
	WebAuthnRelyingPartyIDSet   bool
	WebAuthnUserVerification    string
	WebAuthnUserVerificationSet bool
}

type cognitoUserPoolsStartWebAuthnRegistrationResult struct {
	Challenge string
	Session   string
}

type cognitoUserPoolsWebAuthnCredentialRecord struct {
	CredentialID string
	FriendlyName string
	CreatedAt    time.Time
}

func (s *cognitoUserPoolsStore) AdminCreateUser(input cognitoUserPoolsAdminCreateUserInput) (cognitoUserPoolsUserRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	username := strings.TrimSpace(input.Username)
	if userPoolID == "" {
		return cognitoUserPoolsUserRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return cognitoUserPoolsUserRecord{}, validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsUserRecord{}, notFoundCognitoUserPools("User pool not found")
	}

	key := cognitoUserPoolsUsernameKey(username)
	if s.users[userPoolID] == nil {
		s.users[userPoolID] = map[string]cognitoUserPoolsUserRecord{}
	}
	if _, exists := s.users[userPoolID][key]; exists {
		return cognitoUserPoolsUserRecord{}, conflictCognitoUserPools("User already exists")
	}

	now := time.Now().UTC()
	tmpPassword := strings.TrimSpace(input.TemporaryPassword)
	if tmpPassword == "" {
		tmpPassword = "Temp!" + randomHex(6)
	}
	attributes := cloneStringMap(input.UserAttributes)
	if attributes == nil {
		attributes = map[string]string{}
	}
	sub := randomHex(16)
	attributes["sub"] = sub

	record := cognitoUserPoolsUserRecord{
		UserPoolID:                 userPoolID,
		Username:                   username,
		Sub:                        sub,
		Attributes:                 attributes,
		AttributeVerificationCodes: map[string]string{},
		Enabled:                    true,
		UserStatus:                 "FORCE_CHANGE_PASSWORD",
		Password:                   tmpPassword,
		Groups:                     map[string]struct{}{},
		Devices:                    map[string]cognitoUserPoolsDeviceRecord{},
		WebAuthnCredentials:        map[string]cognitoUserPoolsWebAuthnCredentialRecord{},
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	s.users[userPoolID][key] = record
	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return cloneCognitoUserPoolsUserRecord(record), nil
}

func (s *cognitoUserPoolsStore) SignUp(clientID, username, password string, attributes map[string]string) (cognitoUserPoolsUserRecord, bool, error) {
	clientID = strings.TrimSpace(clientID)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if clientID == "" {
		return cognitoUserPoolsUserRecord{}, false, validationCognitoUserPools("ClientId is required")
	}
	if username == "" {
		return cognitoUserPoolsUserRecord{}, false, validationCognitoUserPools("Username is required")
	}
	if password == "" {
		return cognitoUserPoolsUserRecord{}, false, validationCognitoUserPools("Password is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userPoolID, err := s.resolveUserPoolIDFromClientIDLocked(clientID)
	if err != nil {
		return cognitoUserPoolsUserRecord{}, false, err
	}

	if s.users[userPoolID] == nil {
		s.users[userPoolID] = map[string]cognitoUserPoolsUserRecord{}
	}
	key := cognitoUserPoolsUsernameKey(username)
	if _, exists := s.users[userPoolID][key]; exists {
		return cognitoUserPoolsUserRecord{}, false, conflictCognitoUserPools("User already exists")
	}

	now := time.Now().UTC()
	attrs := cloneStringMap(attributes)
	if attrs == nil {
		attrs = map[string]string{}
	}
	sub := randomHex(16)
	attrs["sub"] = sub

	record := cognitoUserPoolsUserRecord{
		UserPoolID:                 userPoolID,
		Username:                   username,
		Sub:                        sub,
		Attributes:                 attrs,
		AttributeVerificationCodes: map[string]string{},
		Enabled:                    true,
		UserStatus:                 "UNCONFIRMED",
		Password:                   password,
		Groups:                     map[string]struct{}{},
		Devices:                    map[string]cognitoUserPoolsDeviceRecord{},
		WebAuthnCredentials:        map[string]cognitoUserPoolsWebAuthnCredentialRecord{},
		ConfirmationCode:           cognitoUserPoolsCode(),
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	s.users[userPoolID][key] = record

	pool := s.pools[userPoolID]
	pool.UpdatedAt = now
	s.pools[userPoolID] = pool
	return cloneCognitoUserPoolsUserRecord(record), false, nil
}

func (s *cognitoUserPoolsStore) ConfirmSignUp(clientID, username, confirmationCode string) error {
	clientID = strings.TrimSpace(clientID)
	username = strings.TrimSpace(username)
	if clientID == "" {
		return validationCognitoUserPools("ClientId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userPoolID, err := s.resolveUserPoolIDFromClientIDLocked(clientID)
	if err != nil {
		return err
	}
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	if record.ConfirmationCode != "" && strings.TrimSpace(confirmationCode) != "" && strings.TrimSpace(confirmationCode) != record.ConfirmationCode {
		return validationCognitoUserPools("Confirmation code is invalid")
	}
	record.UserStatus = "CONFIRMED"
	record.ConfirmationCode = ""
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) AdminConfirmSignUp(userPoolID, username string) error {
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

	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	record.UserStatus = "CONFIRMED"
	record.ConfirmationCode = ""
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) ResendConfirmationCode(clientID, username string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	username = strings.TrimSpace(username)
	if clientID == "" {
		return "", validationCognitoUserPools("ClientId is required")
	}
	if username == "" {
		return "", validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	userPoolID, err := s.resolveUserPoolIDFromClientIDLocked(clientID)
	if err != nil {
		return "", err
	}
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return "", err
	}
	record.ConfirmationCode = cognitoUserPoolsCode()
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return cognitoUserPoolsDeliveryDestination(record), nil
}

func (s *cognitoUserPoolsStore) AdminGetUser(userPoolID, username string) (cognitoUserPoolsUserRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return cognitoUserPoolsUserRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return cognitoUserPoolsUserRecord{}, validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, _, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return cognitoUserPoolsUserRecord{}, err
	}
	return cloneCognitoUserPoolsUserRecord(record), nil
}

func (s *cognitoUserPoolsStore) ListUsers(userPoolID string, limit int, paginationToken string) ([]cognitoUserPoolsUserRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}

	usersByKey := s.users[userPoolID]
	keys := make([]string, 0, len(usersByKey))
	for key := range usersByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	start, err := parseCognitoUserPoolsNextToken(paginationToken, len(keys))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]cognitoUserPoolsUserRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, cloneCognitoUserPoolsUserRecord(usersByKey[key]))
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) AdminUpdateUserAttributes(userPoolID, username string, attributes map[string]string) error {
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
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	if record.Attributes == nil {
		record.Attributes = map[string]string{}
	}
	for k, v := range attributes {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		record.Attributes[name] = strings.TrimSpace(v)
	}
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) AdminDeleteUserAttributes(userPoolID, username string, attributeNames []string) error {
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
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	for _, name := range attributeNames {
		delete(record.Attributes, strings.TrimSpace(name))
		delete(record.AttributeVerificationCodes, strings.TrimSpace(name))
	}
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) AdminDeleteUser(userPoolID, username string) error {
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
	if _, ok := s.pools[userPoolID]; !ok {
		return notFoundCognitoUserPools("User pool not found")
	}
	key := cognitoUserPoolsUsernameKey(username)
	record, ok := s.users[userPoolID][key]
	if !ok {
		return notFoundCognitoUserPools("User not found")
	}
	delete(s.users[userPoolID], key)
	for groupName := range record.Groups {
		_ = groupName
	}
	s.revokeTokensForUserLocked(userPoolID, key)
	return nil
}

func (s *cognitoUserPoolsStore) AdminDisableUser(userPoolID, username string) error {
	return s.setUserEnabled(userPoolID, username, false)
}

func (s *cognitoUserPoolsStore) AdminEnableUser(userPoolID, username string) error {
	return s.setUserEnabled(userPoolID, username, true)
}

func (s *cognitoUserPoolsStore) setUserEnabled(userPoolID, username string, enabled bool) error {
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
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	record.Enabled = enabled
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) AdminSetUserPassword(userPoolID, username, password string, permanent bool) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if password == "" {
		return validationCognitoUserPools("Password is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	record.Password = password
	if permanent {
		record.UserStatus = "CONFIRMED"
	} else {
		record.UserStatus = "FORCE_CHANGE_PASSWORD"
	}
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) AdminResetUserPassword(userPoolID, username string) (string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return "", validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return "", validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return "", err
	}
	record.UserStatus = "RESET_REQUIRED"
	record.ResetCode = cognitoUserPoolsCode()
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return cognitoUserPoolsDeliveryDestination(record), nil
}

func (s *cognitoUserPoolsStore) ForgotPassword(clientID, username string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	username = strings.TrimSpace(username)
	if clientID == "" {
		return "", validationCognitoUserPools("ClientId is required")
	}
	if username == "" {
		return "", validationCognitoUserPools("Username is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	userPoolID, err := s.resolveUserPoolIDFromClientIDLocked(clientID)
	if err != nil {
		return "", err
	}
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return "", err
	}
	record.ResetCode = cognitoUserPoolsCode()
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return cognitoUserPoolsDeliveryDestination(record), nil
}

func (s *cognitoUserPoolsStore) ConfirmForgotPassword(clientID, username, confirmationCode, password string) error {
	clientID = strings.TrimSpace(clientID)
	username = strings.TrimSpace(username)
	confirmationCode = strings.TrimSpace(confirmationCode)
	password = strings.TrimSpace(password)
	if clientID == "" {
		return validationCognitoUserPools("ClientId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if confirmationCode == "" {
		return validationCognitoUserPools("ConfirmationCode is required")
	}
	if password == "" {
		return validationCognitoUserPools("Password is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	userPoolID, err := s.resolveUserPoolIDFromClientIDLocked(clientID)
	if err != nil {
		return err
	}
	record, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	if record.ResetCode == "" || confirmationCode != record.ResetCode {
		return validationCognitoUserPools("Confirmation code is invalid")
	}
	record.Password = password
	record.UserStatus = "CONFIRMED"
	record.ResetCode = ""
	record.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = record
	return nil
}

func (s *cognitoUserPoolsStore) InitiateAuth(input cognitoUserPoolsInitiateAuthInput) (cognitoUserPoolsAuthFlowResult, error) {
	clientID := strings.TrimSpace(input.ClientID)
	if clientID == "" {
		return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("ClientId is required")
	}
	authFlow := strings.ToUpper(strings.TrimSpace(input.AuthFlow))
	if authFlow == "" {
		authFlow = "USER_PASSWORD_AUTH"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userPoolID := strings.TrimSpace(input.UserPoolID)
	var err error
	if userPoolID == "" {
		userPoolID, err = s.resolveUserPoolIDFromClientIDLocked(clientID)
		if err != nil {
			return cognitoUserPoolsAuthFlowResult{}, err
		}
	}

	switch authFlow {
	case "REFRESH_TOKEN_AUTH", "REFRESH_TOKEN":
		refreshToken := strings.TrimSpace(input.AuthParameters["REFRESH_TOKEN"])
		auth, err := s.getTokensFromRefreshTokenLocked(clientID, refreshToken)
		if err != nil {
			return cognitoUserPoolsAuthFlowResult{}, err
		}
		return cognitoUserPoolsAuthFlowResult{AuthenticationResult: &auth}, nil
	case "USER_PASSWORD_AUTH", "ADMIN_USER_PASSWORD_AUTH", "ADMIN_NO_SRP_AUTH", "USER_AUTH":
		username := strings.TrimSpace(input.AuthParameters["USERNAME"])
		password := strings.TrimSpace(input.AuthParameters["PASSWORD"])
		if username == "" {
			return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("USERNAME is required")
		}
		if password == "" {
			return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("PASSWORD is required")
		}
		user, key, err := s.getUserLocked(userPoolID, username)
		if err != nil {
			return cognitoUserPoolsAuthFlowResult{}, err
		}
		if !user.Enabled {
			s.recordAuthEventLocked(userPoolID, username, "SignIn", "Failure")
			return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("User is disabled")
		}
		if user.UserStatus == "UNCONFIRMED" {
			s.recordAuthEventLocked(userPoolID, username, "SignIn", "Failure")
			return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("User is not confirmed")
		}
		if user.Password != password {
			s.recordAuthEventLocked(userPoolID, username, "SignIn", "Failure")
			return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("Incorrect username or password")
		}
		if user.UserStatus == "FORCE_CHANGE_PASSWORD" {
			s.recordAuthEventLocked(userPoolID, username, "SignIn", "Success")
			session := s.createSessionLocked(cognitoUserPoolsSessionRecord{
				UserPoolID:    userPoolID,
				ClientID:      clientID,
				Username:      user.Username,
				SessionKind:   "AUTH_CHALLENGE",
				ChallengeName: "NEW_PASSWORD_REQUIRED",
				ExpiresAt:     time.Now().UTC().Add(10 * time.Minute),
			})
			return cognitoUserPoolsAuthFlowResult{
				ChallengeName:       "NEW_PASSWORD_REQUIRED",
				ChallengeParameters: map[string]string{"USER_ID_FOR_SRP": user.Username},
				Session:             session,
			}, nil
		}
		if user.SoftwareTokenEnabled && user.SoftwareTokenVerified {
			s.recordAuthEventLocked(userPoolID, username, "SignIn", "Success")
			session := s.createSessionLocked(cognitoUserPoolsSessionRecord{
				UserPoolID:    userPoolID,
				ClientID:      clientID,
				Username:      user.Username,
				SessionKind:   "AUTH_CHALLENGE",
				ChallengeName: "SOFTWARE_TOKEN_MFA",
				ExpiresAt:     time.Now().UTC().Add(10 * time.Minute),
			})
			return cognitoUserPoolsAuthFlowResult{
				ChallengeName:       "SOFTWARE_TOKEN_MFA",
				ChallengeParameters: map[string]string{"USERNAME": user.Username},
				Session:             session,
			}, nil
		}
		auth := s.issueTokensLocked(userPoolID, clientID, user.Username, "")
		s.recordAuthEventLocked(userPoolID, username, "SignIn", "Success")
		user.UpdatedAt = time.Now().UTC()
		s.users[userPoolID][key] = user
		return cognitoUserPoolsAuthFlowResult{AuthenticationResult: &auth}, nil
	default:
		return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("AuthFlow is invalid")
	}
}

func (s *cognitoUserPoolsStore) RespondToAuthChallenge(input cognitoUserPoolsRespondAuthChallengeInput) (cognitoUserPoolsAuthFlowResult, error) {
	clientID := strings.TrimSpace(input.ClientID)
	sessionID := strings.TrimSpace(input.Session)
	if clientID == "" {
		return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("ClientId is required")
	}
	if sessionID == "" {
		return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("Session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || time.Now().UTC().After(session.ExpiresAt) {
		return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("Session is invalid")
	}
	if session.ClientID != clientID {
		return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("Session is invalid")
	}
	challengeName := strings.TrimSpace(input.ChallengeName)
	if challengeName == "" {
		challengeName = session.ChallengeName
	}
	challengeName = strings.ToUpper(challengeName)

	user, key, err := s.getUserLocked(session.UserPoolID, session.Username)
	if err != nil {
		return cognitoUserPoolsAuthFlowResult{}, err
	}

	switch challengeName {
	case "NEW_PASSWORD_REQUIRED":
		newPassword := strings.TrimSpace(input.ChallengeResponses["NEW_PASSWORD"])
		if newPassword == "" {
			newPassword = strings.TrimSpace(input.ChallengeResponses["NEW_PASSWORD_REQUIRED"])
		}
		if newPassword == "" {
			return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("NEW_PASSWORD is required")
		}
		user.Password = newPassword
		user.UserStatus = "CONFIRMED"
		user.UpdatedAt = time.Now().UTC()
		s.users[session.UserPoolID][key] = user
		delete(s.sessions, sessionID)
		auth := s.issueTokensLocked(session.UserPoolID, session.ClientID, user.Username, "")
		return cognitoUserPoolsAuthFlowResult{AuthenticationResult: &auth}, nil
	case "SOFTWARE_TOKEN_MFA":
		userCode := strings.TrimSpace(input.ChallengeResponses["SOFTWARE_TOKEN_MFA_CODE"])
		if userCode == "" {
			return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("SOFTWARE_TOKEN_MFA_CODE is required")
		}
		if !user.SoftwareTokenVerified {
			return cognitoUserPoolsAuthFlowResult{}, notAuthorizedCognitoUserPools("MFA token is not verified")
		}
		user.UpdatedAt = time.Now().UTC()
		s.users[session.UserPoolID][key] = user
		delete(s.sessions, sessionID)
		auth := s.issueTokensLocked(session.UserPoolID, session.ClientID, user.Username, "")
		return cognitoUserPoolsAuthFlowResult{AuthenticationResult: &auth}, nil
	default:
		return cognitoUserPoolsAuthFlowResult{}, validationCognitoUserPools("ChallengeName is invalid")
	}
}

func (s *cognitoUserPoolsStore) GetTokensFromRefreshToken(clientID, refreshToken string) (cognitoUserPoolsAuthResult, error) {
	clientID = strings.TrimSpace(clientID)
	refreshToken = strings.TrimSpace(refreshToken)
	if clientID == "" {
		return cognitoUserPoolsAuthResult{}, validationCognitoUserPools("ClientId is required")
	}
	if refreshToken == "" {
		return cognitoUserPoolsAuthResult{}, validationCognitoUserPools("RefreshToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getTokensFromRefreshTokenLocked(clientID, refreshToken)
}

func (s *cognitoUserPoolsStore) getTokensFromRefreshTokenLocked(clientID, refreshToken string) (cognitoUserPoolsAuthResult, error) {
	record, ok := s.refreshTokens[refreshToken]
	if !ok || record.Revoked || time.Now().UTC().After(record.ExpiresAt) {
		return cognitoUserPoolsAuthResult{}, notAuthorizedCognitoUserPools("Refresh token is invalid")
	}
	if record.ClientID != clientID {
		return cognitoUserPoolsAuthResult{}, notAuthorizedCognitoUserPools("Refresh token is invalid")
	}
	auth := s.issueTokensLocked(record.UserPoolID, record.ClientID, record.Username, refreshToken)
	return auth, nil
}

func (s *cognitoUserPoolsStore) RevokeToken(clientID, token string) error {
	clientID = strings.TrimSpace(clientID)
	token = strings.TrimSpace(token)
	if clientID == "" {
		return validationCognitoUserPools("ClientId is required")
	}
	if token == "" {
		return validationCognitoUserPools("Token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.refreshTokens[token]
	if !ok {
		return nil
	}
	if record.ClientID != clientID {
		return notAuthorizedCognitoUserPools("Token is invalid")
	}
	record.Revoked = true
	s.refreshTokens[token] = record
	return nil
}

func (s *cognitoUserPoolsStore) GlobalSignOut(accessToken string) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tokenRecord, ok := s.accessTokens[accessToken]
	if !ok || tokenRecord.Revoked || time.Now().UTC().After(tokenRecord.ExpiresAt) {
		return notAuthorizedCognitoUserPools("Access token is invalid")
	}
	s.revokeTokensForUserLocked(tokenRecord.UserPoolID, cognitoUserPoolsUsernameKey(tokenRecord.Username))
	return nil
}

func (s *cognitoUserPoolsStore) AdminUserGlobalSignOut(userPoolID, username string) error {
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
	if _, _, err := s.getUserLocked(userPoolID, username); err != nil {
		return err
	}
	s.revokeTokensForUserLocked(userPoolID, cognitoUserPoolsUsernameKey(username))
	return nil
}

func (s *cognitoUserPoolsStore) GetUser(accessToken string) (cognitoUserPoolsUserRecord, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return cognitoUserPoolsUserRecord{}, validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return cognitoUserPoolsUserRecord{}, err
	}
	return cloneCognitoUserPoolsUserRecord(user), nil
}

func (s *cognitoUserPoolsStore) DeleteUser(accessToken string) error {
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
	delete(s.users[user.UserPoolID], key)
	s.revokeTokensForUserLocked(user.UserPoolID, key)
	return nil
}

func (s *cognitoUserPoolsStore) ChangePassword(accessToken, previousPassword, proposedPassword string) error {
	accessToken = strings.TrimSpace(accessToken)
	previousPassword = strings.TrimSpace(previousPassword)
	proposedPassword = strings.TrimSpace(proposedPassword)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if previousPassword == "" {
		return validationCognitoUserPools("PreviousPassword is required")
	}
	if proposedPassword == "" {
		return validationCognitoUserPools("ProposedPassword is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	if user.Password != previousPassword {
		return notAuthorizedCognitoUserPools("Incorrect username or password")
	}
	user.Password = proposedPassword
	user.UserStatus = "CONFIRMED"
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) UpdateUserAttributes(accessToken string, attributes map[string]string) ([]string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return nil, err
	}
	if user.Attributes == nil {
		user.Attributes = map[string]string{}
	}
	if user.AttributeVerificationCodes == nil {
		user.AttributeVerificationCodes = map[string]string{}
	}

	needsVerification := make([]string, 0)
	for name, value := range attributes {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}
		user.Attributes[trimmedName] = strings.TrimSpace(value)
		if trimmedName == "email" || trimmedName == "phone_number" {
			code := cognitoUserPoolsCode()
			user.AttributeVerificationCodes[trimmedName] = code
			user.Attributes[trimmedName+"_verified"] = "false"
			needsVerification = append(needsVerification, trimmedName)
		}
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return needsVerification, nil
}

func (s *cognitoUserPoolsStore) DeleteUserAttributes(accessToken string, attributeNames []string) error {
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
	for _, name := range attributeNames {
		trimmed := strings.TrimSpace(name)
		delete(user.Attributes, trimmed)
		delete(user.Attributes, trimmed+"_verified")
		delete(user.AttributeVerificationCodes, trimmed)
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) GetUserAttributeVerificationCode(accessToken, attributeName string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	attributeName = strings.TrimSpace(attributeName)
	if accessToken == "" {
		return "", validationCognitoUserPools("AccessToken is required")
	}
	if attributeName == "" {
		return "", validationCognitoUserPools("AttributeName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return "", err
	}
	if user.AttributeVerificationCodes == nil {
		user.AttributeVerificationCodes = map[string]string{}
	}
	code := cognitoUserPoolsCode()
	user.AttributeVerificationCodes[attributeName] = code
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return cognitoUserPoolsDeliveryDestination(user), nil
}

func (s *cognitoUserPoolsStore) VerifyUserAttribute(accessToken, attributeName, code string) error {
	accessToken = strings.TrimSpace(accessToken)
	attributeName = strings.TrimSpace(attributeName)
	code = strings.TrimSpace(code)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if attributeName == "" {
		return validationCognitoUserPools("AttributeName is required")
	}
	if code == "" {
		return validationCognitoUserPools("Code is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	expected := strings.TrimSpace(user.AttributeVerificationCodes[attributeName])
	if expected == "" || expected != code {
		return validationCognitoUserPools("Code is invalid")
	}
	if user.Attributes == nil {
		user.Attributes = map[string]string{}
	}
	user.Attributes[attributeName+"_verified"] = "true"
	delete(user.AttributeVerificationCodes, attributeName)
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) ConfirmDevice(accessToken, deviceKey, deviceName string) error {
	accessToken = strings.TrimSpace(accessToken)
	deviceKey = strings.TrimSpace(deviceKey)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if deviceKey == "" {
		return validationCognitoUserPools("DeviceKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	if user.Devices == nil {
		user.Devices = map[string]cognitoUserPoolsDeviceRecord{}
	}
	now := time.Now().UTC()
	record := user.Devices[deviceKey]
	if record.DeviceCreateDate.IsZero() {
		record.DeviceCreateDate = now
	}
	record.DeviceKey = deviceKey
	record.DeviceName = strings.TrimSpace(deviceName)
	record.DeviceRememberedStatus = "remembered"
	record.DeviceLastAuthenticatedAt = now
	record.DeviceLastModifiedDate = now
	user.Devices[deviceKey] = record
	user.UpdatedAt = now
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) GetDeviceByAccessToken(accessToken, deviceKey string) (cognitoUserPoolsDeviceRecord, error) {
	accessToken = strings.TrimSpace(accessToken)
	deviceKey = strings.TrimSpace(deviceKey)
	if accessToken == "" {
		return cognitoUserPoolsDeviceRecord{}, validationCognitoUserPools("AccessToken is required")
	}
	if deviceKey == "" {
		return cognitoUserPoolsDeviceRecord{}, validationCognitoUserPools("DeviceKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return cognitoUserPoolsDeviceRecord{}, err
	}
	record, ok := user.Devices[deviceKey]
	if !ok {
		return cognitoUserPoolsDeviceRecord{}, notFoundCognitoUserPools("Device not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) ListDevicesByAccessToken(accessToken string, limit int, paginationToken string) ([]cognitoUserPoolsDeviceRecord, string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, "", validationCognitoUserPools("AccessToken is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return nil, "", err
	}
	return paginateCognitoUserPoolsDevices(user.Devices, limit, paginationToken)
}

func (s *cognitoUserPoolsStore) UpdateDeviceStatusByAccessToken(accessToken, deviceKey, status string) error {
	accessToken = strings.TrimSpace(accessToken)
	deviceKey = strings.TrimSpace(deviceKey)
	status = strings.ToLower(strings.TrimSpace(status))
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if deviceKey == "" {
		return validationCognitoUserPools("DeviceKey is required")
	}
	if status == "" {
		return validationCognitoUserPools("DeviceRememberedStatus is required")
	}
	if status != "remembered" && status != "not_remembered" {
		return validationCognitoUserPools("DeviceRememberedStatus is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	record, ok := user.Devices[deviceKey]
	if !ok {
		return notFoundCognitoUserPools("Device not found")
	}
	record.DeviceRememberedStatus = status
	record.DeviceLastModifiedDate = time.Now().UTC()
	user.Devices[deviceKey] = record
	user.UpdatedAt = record.DeviceLastModifiedDate
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) ForgetDeviceByAccessToken(accessToken, deviceKey string) error {
	accessToken = strings.TrimSpace(accessToken)
	deviceKey = strings.TrimSpace(deviceKey)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if deviceKey == "" {
		return validationCognitoUserPools("DeviceKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	if _, ok := user.Devices[deviceKey]; !ok {
		return notFoundCognitoUserPools("Device not found")
	}
	delete(user.Devices, deviceKey)
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) AdminGetDevice(userPoolID, username, deviceKey string) (cognitoUserPoolsDeviceRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	deviceKey = strings.TrimSpace(deviceKey)
	if userPoolID == "" {
		return cognitoUserPoolsDeviceRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return cognitoUserPoolsDeviceRecord{}, validationCognitoUserPools("Username is required")
	}
	if deviceKey == "" {
		return cognitoUserPoolsDeviceRecord{}, validationCognitoUserPools("DeviceKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return cognitoUserPoolsDeviceRecord{}, err
	}
	record, ok := user.Devices[deviceKey]
	if !ok {
		return cognitoUserPoolsDeviceRecord{}, notFoundCognitoUserPools("Device not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) AdminListDevices(userPoolID, username string, limit int, paginationToken string) ([]cognitoUserPoolsDeviceRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return nil, "", validationCognitoUserPools("Username is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return nil, "", err
	}
	return paginateCognitoUserPoolsDevices(user.Devices, limit, paginationToken)
}

func (s *cognitoUserPoolsStore) AdminUpdateDeviceStatus(userPoolID, username, deviceKey, status string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	deviceKey = strings.TrimSpace(deviceKey)
	status = strings.ToLower(strings.TrimSpace(status))
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if deviceKey == "" {
		return validationCognitoUserPools("DeviceKey is required")
	}
	if status == "" {
		return validationCognitoUserPools("DeviceRememberedStatus is required")
	}
	if status != "remembered" && status != "not_remembered" {
		return validationCognitoUserPools("DeviceRememberedStatus is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	record, ok := user.Devices[deviceKey]
	if !ok {
		return notFoundCognitoUserPools("Device not found")
	}
	record.DeviceRememberedStatus = status
	record.DeviceLastModifiedDate = time.Now().UTC()
	user.Devices[deviceKey] = record
	user.UpdatedAt = record.DeviceLastModifiedDate
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) AdminForgetDevice(userPoolID, username, deviceKey string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	deviceKey = strings.TrimSpace(deviceKey)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if deviceKey == "" {
		return validationCognitoUserPools("DeviceKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	if _, ok := user.Devices[deviceKey]; !ok {
		return notFoundCognitoUserPools("Device not found")
	}
	delete(user.Devices, deviceKey)
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) CreateGroup(input cognitoUserPoolsCreateGroupInput) (cognitoUserPoolsGroupRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	groupName := strings.TrimSpace(input.GroupName)
	if userPoolID == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if groupName == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsGroupRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.groups[userPoolID] == nil {
		s.groups[userPoolID] = map[string]cognitoUserPoolsGroupRecord{}
	}
	groupKey := cognitoUserPoolsGroupKey(groupName)
	if _, exists := s.groups[userPoolID][groupKey]; exists {
		return cognitoUserPoolsGroupRecord{}, conflictCognitoUserPools("Group already exists")
	}

	now := time.Now().UTC()
	record := cognitoUserPoolsGroupRecord{
		UserPoolID:     userPoolID,
		GroupName:      groupName,
		Description:    strings.TrimSpace(input.Description),
		RoleARN:        strings.TrimSpace(input.RoleARN),
		Precedence:     cloneCognitoUserPoolsIntPointer(input.Precedence),
		CreatedAt:      now,
		LastModifiedAt: now,
	}
	s.groups[userPoolID][groupKey] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) GetGroup(userPoolID, groupName string) (cognitoUserPoolsGroupRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	groupName = strings.TrimSpace(groupName)
	if userPoolID == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if groupName == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getGroupLocked(userPoolID, groupName)
}

func (s *cognitoUserPoolsStore) UpdateGroup(input cognitoUserPoolsUpdateGroupInput) (cognitoUserPoolsGroupRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	groupName := strings.TrimSpace(input.GroupName)
	if userPoolID == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if groupName == "" {
		return cognitoUserPoolsGroupRecord{}, validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.getGroupLocked(userPoolID, groupName)
	if err != nil {
		return cognitoUserPoolsGroupRecord{}, err
	}
	if input.DescriptionSet {
		record.Description = strings.TrimSpace(input.Description)
	}
	if input.RoleARNSet {
		record.RoleARN = strings.TrimSpace(input.RoleARN)
	}
	if input.PrecedenceSet {
		record.Precedence = cloneCognitoUserPoolsIntPointer(input.Precedence)
	}
	record.LastModifiedAt = time.Now().UTC()
	s.groups[userPoolID][cognitoUserPoolsGroupKey(groupName)] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) DeleteGroup(userPoolID, groupName string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	groupName = strings.TrimSpace(groupName)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if groupName == "" {
		return validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	groupKey := cognitoUserPoolsGroupKey(groupName)
	if _, ok := s.groups[userPoolID][groupKey]; !ok {
		return notFoundCognitoUserPools("Group not found")
	}
	delete(s.groups[userPoolID], groupKey)
	for userKey, user := range s.users[userPoolID] {
		if user.Groups != nil {
			delete(user.Groups, groupName)
			s.users[userPoolID][userKey] = user
		}
	}
	return nil
}

func (s *cognitoUserPoolsStore) ListGroups(userPoolID string, limit int, nextToken string) ([]cognitoUserPoolsGroupRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return nil, "", notFoundCognitoUserPools("User pool not found")
	}
	groupsByKey := s.groups[userPoolID]
	keys := make([]string, 0, len(groupsByKey))
	for key := range groupsByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start, err := parseCognitoUserPoolsNextToken(nextToken, len(keys))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	out := make([]cognitoUserPoolsGroupRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, groupsByKey[key])
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) AdminAddUserToGroup(userPoolID, username, groupName string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	groupName = strings.TrimSpace(groupName)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if groupName == "" {
		return validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.getGroupLocked(userPoolID, groupName); err != nil {
		return err
	}
	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	if user.Groups == nil {
		user.Groups = map[string]struct{}{}
	}
	user.Groups[groupName] = struct{}{}
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) AdminRemoveUserFromGroup(userPoolID, username, groupName string) error {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	groupName = strings.TrimSpace(groupName)
	if userPoolID == "" {
		return validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return validationCognitoUserPools("Username is required")
	}
	if groupName == "" {
		return validationCognitoUserPools("GroupName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return err
	}
	delete(user.Groups, groupName)
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) AdminListGroupsForUser(userPoolID, username string, limit int, nextToken string) ([]cognitoUserPoolsGroupRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	username = strings.TrimSpace(username)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if username == "" {
		return nil, "", validationCognitoUserPools("Username is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return nil, "", err
	}
	groupNames := make([]string, 0, len(user.Groups))
	for name := range user.Groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	start, err := parseCognitoUserPoolsNextToken(nextToken, len(groupNames))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(groupNames) {
		end = len(groupNames)
	}
	out := make([]cognitoUserPoolsGroupRecord, 0, end-start)
	for _, groupName := range groupNames[start:end] {
		if record, ok := s.groups[userPoolID][cognitoUserPoolsGroupKey(groupName)]; ok {
			out = append(out, record)
		}
	}
	outToken := ""
	if end < len(groupNames) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) ListUsersInGroup(userPoolID, groupName string, limit int, nextToken string) ([]cognitoUserPoolsUserRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	groupName = strings.TrimSpace(groupName)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if groupName == "" {
		return nil, "", validationCognitoUserPools("GroupName is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("Limit must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.getGroupLocked(userPoolID, groupName); err != nil {
		return nil, "", err
	}
	keys := make([]string, 0)
	for key, user := range s.users[userPoolID] {
		if _, ok := user.Groups[groupName]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	start, err := parseCognitoUserPoolsNextToken(nextToken, len(keys))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	out := make([]cognitoUserPoolsUserRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, cloneCognitoUserPoolsUserRecord(s.users[userPoolID][key]))
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) CreateUserImportJob(userPoolID, jobName, cloudWatchLogsRoleArn string) (cognitoUserPoolsImportJobRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	jobName = strings.TrimSpace(jobName)
	if userPoolID == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if jobName == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("JobName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsImportJobRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if s.importJobs[userPoolID] == nil {
		s.importJobs[userPoolID] = map[string]cognitoUserPoolsImportJobRecord{}
	}
	now := time.Now().UTC()
	record := cognitoUserPoolsImportJobRecord{
		UserPoolID:            userPoolID,
		JobID:                 "import-" + randomHex(10),
		JobName:               jobName,
		CloudWatchLogsRoleArn: strings.TrimSpace(cloudWatchLogsRoleArn),
		Status:                "Created",
		PreSignedURL:          "https://example.stackyard.local/cognito-import/" + randomHex(12),
		CreatedAt:             now,
		LastModifiedAt:        now,
	}
	s.importJobs[userPoolID][record.JobID] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) DescribeUserImportJob(userPoolID, jobID string) (cognitoUserPoolsImportJobRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	jobID = strings.TrimSpace(jobID)
	if userPoolID == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if jobID == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("JobId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.importJobs[userPoolID][jobID]
	if !ok {
		return cognitoUserPoolsImportJobRecord{}, notFoundCognitoUserPools("User import job not found")
	}
	return job, nil
}

func (s *cognitoUserPoolsStore) ListUserImportJobs(userPoolID string, limit int, paginationToken string) ([]cognitoUserPoolsImportJobRecord, string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, "", validationCognitoUserPools("UserPoolId is required")
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	jobsByID := s.importJobs[userPoolID]
	ids := make([]string, 0, len(jobsByID))
	for id := range jobsByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	start, err := parseCognitoUserPoolsNextToken(paginationToken, len(ids))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(ids) {
		end = len(ids)
	}
	out := make([]cognitoUserPoolsImportJobRecord, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, jobsByID[id])
	}
	outToken := ""
	if end < len(ids) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) StartUserImportJob(userPoolID, jobID string) (cognitoUserPoolsImportJobRecord, error) {
	return s.setUserImportJobStatus(userPoolID, jobID, "InProgress", true)
}

func (s *cognitoUserPoolsStore) StopUserImportJob(userPoolID, jobID string) (cognitoUserPoolsImportJobRecord, error) {
	return s.setUserImportJobStatus(userPoolID, jobID, "Stopped", false)
}

func (s *cognitoUserPoolsStore) setUserImportJobStatus(userPoolID, jobID, status string, setStarted bool) (cognitoUserPoolsImportJobRecord, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	jobID = strings.TrimSpace(jobID)
	if userPoolID == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("UserPoolId is required")
	}
	if jobID == "" {
		return cognitoUserPoolsImportJobRecord{}, validationCognitoUserPools("JobId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.importJobs[userPoolID][jobID]
	if !ok {
		return cognitoUserPoolsImportJobRecord{}, notFoundCognitoUserPools("User import job not found")
	}
	now := time.Now().UTC()
	job.Status = status
	if setStarted {
		job.StartedAt = &now
	}
	if status == "Stopped" {
		job.CompletedAt = &now
	}
	job.LastModifiedAt = now
	s.importJobs[userPoolID][jobID] = job
	return job, nil
}

func (s *cognitoUserPoolsStore) GetCSVHeader(userPoolID string) ([]string, error) {
	userPoolID = strings.TrimSpace(userPoolID)
	if userPoolID == "" {
		return nil, validationCognitoUserPools("UserPoolId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[userPoolID]; !ok {
		return nil, notFoundCognitoUserPools("User pool not found")
	}
	return []string{"username", "email", "phone_number"}, nil
}

func (s *cognitoUserPoolsStore) SetUserMFAPreference(input cognitoUserPoolsSetMFAPreferenceInput) error {
	if strings.TrimSpace(input.ByAccessToken) != "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		user, key, err := s.userFromAccessTokenLocked(input.ByAccessToken)
		if err != nil {
			return err
		}
		applyCognitoUserPoolsMFAPreference(&user, input.SoftwareEnabled, input.SoftwarePreferred)
		user.UpdatedAt = time.Now().UTC()
		s.users[user.UserPoolID][key] = user
		return nil
	}

	userPoolID := strings.TrimSpace(input.UserPoolID)
	username := strings.TrimSpace(input.Username)
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
	applyCognitoUserPoolsMFAPreference(&user, input.SoftwareEnabled, input.SoftwarePreferred)
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) SetUserPoolMFAConfig(input cognitoUserPoolsSetPoolMFAConfigInput) (cognitoUserPoolsUserPoolRecord, error) {
	userPoolID := strings.TrimSpace(input.UserPoolID)
	if userPoolID == "" {
		return cognitoUserPoolsUserPoolRecord{}, validationCognitoUserPools("UserPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.pools[userPoolID]
	if !ok {
		return cognitoUserPoolsUserPoolRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	if cfg := strings.TrimSpace(input.MFAConfiguration); cfg != "" {
		record.MFAConfiguration = cfg
	}
	if input.SoftwareTokenMFAEnabled != nil {
		record.SoftwareTokenMFAEnabled = *input.SoftwareTokenMFAEnabled
	}
	if input.WebAuthnRelyingPartyIDSet {
		record.WebAuthnRelyingPartyID = strings.TrimSpace(input.WebAuthnRelyingPartyID)
	}
	if input.WebAuthnUserVerificationSet {
		record.WebAuthnUserVerification = strings.TrimSpace(input.WebAuthnUserVerification)
	}
	record.UpdatedAt = time.Now().UTC()
	s.pools[userPoolID] = record
	return record, nil
}

func (s *cognitoUserPoolsStore) GetUserPoolMFAConfig(userPoolID string) (cognitoUserPoolsUserPoolRecord, error) {
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

func (s *cognitoUserPoolsStore) AssociateSoftwareToken(accessToken, sessionID string) (string, string, error) {
	accessToken = strings.TrimSpace(accessToken)
	sessionID = strings.TrimSpace(sessionID)
	if accessToken == "" && sessionID == "" {
		return "", "", validationCognitoUserPools("AccessToken or Session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var userPoolID, username string
	if accessToken != "" {
		user, _, err := s.userFromAccessTokenLocked(accessToken)
		if err != nil {
			return "", "", err
		}
		userPoolID = user.UserPoolID
		username = user.Username
	} else {
		session, ok := s.sessions[sessionID]
		if !ok {
			return "", "", notAuthorizedCognitoUserPools("Session is invalid")
		}
		userPoolID = session.UserPoolID
		username = session.Username
	}

	secret := strings.ToUpper(randomHex(8))
	newSession := s.createSessionLocked(cognitoUserPoolsSessionRecord{
		UserPoolID:  userPoolID,
		Username:    username,
		SessionKind: "SOFTWARE_TOKEN_SETUP",
		SecretCode:  secret,
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	})
	return secret, newSession, nil
}

func (s *cognitoUserPoolsStore) VerifySoftwareToken(accessToken, sessionID, userCode string) (string, string, error) {
	accessToken = strings.TrimSpace(accessToken)
	sessionID = strings.TrimSpace(sessionID)
	userCode = strings.TrimSpace(userCode)
	if accessToken == "" && sessionID == "" {
		return "", "", validationCognitoUserPools("AccessToken or Session is required")
	}
	if userCode == "" {
		return "", "", validationCognitoUserPools("UserCode is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var userPoolID, username string
	if accessToken != "" {
		user, _, err := s.userFromAccessTokenLocked(accessToken)
		if err != nil {
			return "", "", err
		}
		userPoolID = user.UserPoolID
		username = user.Username
	} else {
		session, ok := s.sessions[sessionID]
		if !ok {
			return "", "", notAuthorizedCognitoUserPools("Session is invalid")
		}
		userPoolID = session.UserPoolID
		username = session.Username
		delete(s.sessions, sessionID)
	}

	user, key, err := s.getUserLocked(userPoolID, username)
	if err != nil {
		return "", "", err
	}
	user.SoftwareTokenVerified = true
	if !user.SoftwareTokenEnabled {
		user.SoftwareTokenEnabled = true
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[userPoolID][key] = user

	newSession := s.createSessionLocked(cognitoUserPoolsSessionRecord{
		UserPoolID:  userPoolID,
		Username:    username,
		SessionKind: "SOFTWARE_TOKEN_VERIFIED",
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	})
	return "SUCCESS", newSession, nil
}

func (s *cognitoUserPoolsStore) StartWebAuthnRegistration(accessToken string) (cognitoUserPoolsStartWebAuthnRegistrationResult, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return cognitoUserPoolsStartWebAuthnRegistrationResult{}, validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return cognitoUserPoolsStartWebAuthnRegistrationResult{}, err
	}
	challenge := randomHex(24)
	session := s.createSessionLocked(cognitoUserPoolsSessionRecord{
		UserPoolID:  user.UserPoolID,
		Username:    user.Username,
		SessionKind: "WEBAUTHN_REGISTRATION",
		Challenge:   challenge,
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	})
	return cognitoUserPoolsStartWebAuthnRegistrationResult{Challenge: challenge, Session: session}, nil
}

func (s *cognitoUserPoolsStore) CompleteWebAuthnRegistration(accessToken, sessionID, credential, friendlyName string) (cognitoUserPoolsWebAuthnCredentialRecord, error) {
	accessToken = strings.TrimSpace(accessToken)
	sessionID = strings.TrimSpace(sessionID)
	if accessToken == "" {
		return cognitoUserPoolsWebAuthnCredentialRecord{}, validationCognitoUserPools("AccessToken is required")
	}
	if credential == "" {
		return cognitoUserPoolsWebAuthnCredentialRecord{}, validationCognitoUserPools("Credential is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return cognitoUserPoolsWebAuthnCredentialRecord{}, err
	}
	if sessionID != "" {
		session, ok := s.sessions[sessionID]
		if !ok || session.SessionKind != "WEBAUTHN_REGISTRATION" {
			return cognitoUserPoolsWebAuthnCredentialRecord{}, notAuthorizedCognitoUserPools("Session is invalid")
		}
		if session.Username != user.Username || session.UserPoolID != user.UserPoolID {
			return cognitoUserPoolsWebAuthnCredentialRecord{}, notAuthorizedCognitoUserPools("Session is invalid")
		}
		delete(s.sessions, sessionID)
	}

	if user.WebAuthnCredentials == nil {
		user.WebAuthnCredentials = map[string]cognitoUserPoolsWebAuthnCredentialRecord{}
	}
	credentialID := "webauthn-" + randomHex(10)
	record := cognitoUserPoolsWebAuthnCredentialRecord{
		CredentialID: credentialID,
		FriendlyName: strings.TrimSpace(friendlyName),
		CreatedAt:    time.Now().UTC(),
	}
	user.WebAuthnCredentials[credentialID] = record
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return record, nil
}

func (s *cognitoUserPoolsStore) ListWebAuthnCredentials(accessToken string, maxResults int, nextToken string) ([]cognitoUserPoolsWebAuthnCredentialRecord, string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, "", validationCognitoUserPools("AccessToken is required")
	}
	if maxResults <= 0 {
		maxResults = 60
	}
	if maxResults > 60 {
		return nil, "", validationCognitoUserPools("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return nil, "", err
	}
	keys := make([]string, 0, len(user.WebAuthnCredentials))
	for key := range user.WebAuthnCredentials {
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
	out := make([]cognitoUserPoolsWebAuthnCredentialRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, user.WebAuthnCredentials[key])
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *cognitoUserPoolsStore) DeleteWebAuthnCredential(accessToken, credentialID string) error {
	accessToken = strings.TrimSpace(accessToken)
	credentialID = strings.TrimSpace(credentialID)
	if accessToken == "" {
		return validationCognitoUserPools("AccessToken is required")
	}
	if credentialID == "" {
		return validationCognitoUserPools("CredentialId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, key, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return err
	}
	if _, ok := user.WebAuthnCredentials[credentialID]; !ok {
		return notFoundCognitoUserPools("WebAuthn credential not found")
	}
	delete(user.WebAuthnCredentials, credentialID)
	user.UpdatedAt = time.Now().UTC()
	s.users[user.UserPoolID][key] = user
	return nil
}

func (s *cognitoUserPoolsStore) GetUserAuthFactors(accessToken string) ([]string, string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, "", validationCognitoUserPools("AccessToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.userFromAccessTokenLocked(accessToken)
	if err != nil {
		return nil, "", err
	}
	factors := []string{"PASSWORD"}
	if user.SoftwareTokenEnabled && user.SoftwareTokenVerified {
		factors = append(factors, "SOFTWARE_TOKEN_MFA")
	}
	if len(user.WebAuthnCredentials) > 0 {
		factors = append(factors, "WEB_AUTHN")
	}
	sort.Strings(factors)
	return factors, strings.TrimSpace(user.PreferredMFA), nil
}

func (s *cognitoUserPoolsStore) getGroupLocked(userPoolID, groupName string) (cognitoUserPoolsGroupRecord, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsGroupRecord{}, notFoundCognitoUserPools("User pool not found")
	}
	record, ok := s.groups[userPoolID][cognitoUserPoolsGroupKey(groupName)]
	if !ok {
		return cognitoUserPoolsGroupRecord{}, notFoundCognitoUserPools("Group not found")
	}
	return record, nil
}

func (s *cognitoUserPoolsStore) getUserLocked(userPoolID, username string) (cognitoUserPoolsUserRecord, string, error) {
	if _, ok := s.pools[userPoolID]; !ok {
		return cognitoUserPoolsUserRecord{}, "", notFoundCognitoUserPools("User pool not found")
	}
	key := cognitoUserPoolsUsernameKey(username)
	if key == "" {
		return cognitoUserPoolsUserRecord{}, "", validationCognitoUserPools("Username is required")
	}
	record, ok := s.users[userPoolID][key]
	if !ok {
		return cognitoUserPoolsUserRecord{}, "", notFoundCognitoUserPools("User not found")
	}
	return record, key, nil
}

func (s *cognitoUserPoolsStore) userFromAccessTokenLocked(accessToken string) (cognitoUserPoolsUserRecord, string, error) {
	tokenRecord, ok := s.accessTokens[accessToken]
	if !ok || tokenRecord.Revoked || time.Now().UTC().After(tokenRecord.ExpiresAt) {
		return cognitoUserPoolsUserRecord{}, "", notAuthorizedCognitoUserPools("Access token is invalid")
	}
	user, key, err := s.getUserLocked(tokenRecord.UserPoolID, tokenRecord.Username)
	if err != nil {
		return cognitoUserPoolsUserRecord{}, "", err
	}
	return user, key, nil
}

func (s *cognitoUserPoolsStore) resolveUserPoolIDFromClientIDLocked(clientID string) (string, error) {
	for userPoolID, clients := range s.clients {
		if _, ok := clients[clientID]; ok {
			return userPoolID, nil
		}
	}
	return "", notFoundCognitoUserPools("User pool client not found")
}

func (s *cognitoUserPoolsStore) createSessionLocked(record cognitoUserPoolsSessionRecord) string {
	record.SessionID = "session-" + randomHex(16)
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	}
	s.sessions[record.SessionID] = record
	return record.SessionID
}

func (s *cognitoUserPoolsStore) issueTokensLocked(userPoolID, clientID, username, existingRefreshToken string) cognitoUserPoolsAuthResult {
	now := time.Now().UTC()
	accessToken := "access-" + randomHex(20)
	idToken := "id-" + randomHex(20)
	refreshToken := strings.TrimSpace(existingRefreshToken)
	if refreshToken == "" {
		refreshToken = "refresh-" + randomHex(24)
	}
	s.accessTokens[accessToken] = cognitoUserPoolsAccessTokenRecord{
		Token:      accessToken,
		UserPoolID: userPoolID,
		ClientID:   clientID,
		Username:   username,
		ExpiresAt:  now.Add(time.Hour),
	}
	if refreshRecord, ok := s.refreshTokens[refreshToken]; ok {
		refreshRecord.Revoked = false
		s.refreshTokens[refreshToken] = refreshRecord
	} else {
		s.refreshTokens[refreshToken] = cognitoUserPoolsRefreshTokenRecord{
			Token:      refreshToken,
			UserPoolID: userPoolID,
			ClientID:   clientID,
			Username:   username,
			ExpiresAt:  now.Add(30 * 24 * time.Hour),
		}
	}
	return cognitoUserPoolsAuthResult{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}
}

func (s *cognitoUserPoolsStore) revokeTokensForUserLocked(userPoolID, usernameKey string) {
	for token, record := range s.accessTokens {
		if record.UserPoolID == userPoolID && cognitoUserPoolsUsernameKey(record.Username) == usernameKey {
			record.Revoked = true
			s.accessTokens[token] = record
		}
	}
	for token, record := range s.refreshTokens {
		if record.UserPoolID == userPoolID && cognitoUserPoolsUsernameKey(record.Username) == usernameKey {
			record.Revoked = true
			s.refreshTokens[token] = record
		}
	}
}

func paginateCognitoUserPoolsDevices(devices map[string]cognitoUserPoolsDeviceRecord, limit int, paginationToken string) ([]cognitoUserPoolsDeviceRecord, string, error) {
	keys := make([]string, 0, len(devices))
	for key := range devices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start, err := parseCognitoUserPoolsNextToken(paginationToken, len(keys))
	if err != nil {
		return nil, "", err
	}
	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	out := make([]cognitoUserPoolsDeviceRecord, 0, end-start)
	for _, key := range keys[start:end] {
		out = append(out, devices[key])
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func cloneCognitoUserPoolsUserRecord(in cognitoUserPoolsUserRecord) cognitoUserPoolsUserRecord {
	out := in
	out.Attributes = cloneStringMap(in.Attributes)
	out.AttributeVerificationCodes = cloneStringMap(in.AttributeVerificationCodes)
	if len(in.Groups) > 0 {
		out.Groups = make(map[string]struct{}, len(in.Groups))
		for key := range in.Groups {
			out.Groups[key] = struct{}{}
		}
	} else {
		out.Groups = map[string]struct{}{}
	}
	if len(in.Devices) > 0 {
		out.Devices = make(map[string]cognitoUserPoolsDeviceRecord, len(in.Devices))
		for key, value := range in.Devices {
			out.Devices[key] = value
		}
	} else {
		out.Devices = map[string]cognitoUserPoolsDeviceRecord{}
	}
	if len(in.WebAuthnCredentials) > 0 {
		out.WebAuthnCredentials = make(map[string]cognitoUserPoolsWebAuthnCredentialRecord, len(in.WebAuthnCredentials))
		for key, value := range in.WebAuthnCredentials {
			out.WebAuthnCredentials[key] = value
		}
	} else {
		out.WebAuthnCredentials = map[string]cognitoUserPoolsWebAuthnCredentialRecord{}
	}
	out.MFAOptions = cloneCognitoUserPoolsMFAOptions(in.MFAOptions)
	return out
}

func applyCognitoUserPoolsMFAPreference(user *cognitoUserPoolsUserRecord, softwareEnabled, softwarePreferred *bool) {
	if user == nil {
		return
	}
	if softwareEnabled != nil {
		user.SoftwareTokenEnabled = *softwareEnabled
		if !*softwareEnabled && strings.EqualFold(user.PreferredMFA, "SOFTWARE_TOKEN_MFA") {
			user.PreferredMFA = ""
		}
	}
	if softwarePreferred != nil {
		if *softwarePreferred {
			user.PreferredMFA = "SOFTWARE_TOKEN_MFA"
		} else if strings.EqualFold(user.PreferredMFA, "SOFTWARE_TOKEN_MFA") {
			user.PreferredMFA = ""
		}
	}
}

func cloneCognitoUserPoolsIntPointer(in *int) *int {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}

func cognitoUserPoolsUsernameKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func cognitoUserPoolsGroupKey(groupName string) string {
	return strings.ToLower(strings.TrimSpace(groupName))
}

func cognitoUserPoolsCode() string {
	// Keep verification codes stable so coverage tooling can pre-seed dependent flows.
	return "123456"
}

func cognitoUserPoolsDeliveryDestination(user cognitoUserPoolsUserRecord) string {
	email := strings.TrimSpace(user.Attributes["email"])
	if email != "" {
		if idx := strings.Index(email, "@"); idx > 1 {
			return email[:1] + "***" + email[idx:]
		}
		return email
	}
	phone := strings.TrimSpace(user.Attributes["phone_number"])
	if phone != "" {
		if len(phone) > 4 {
			return "***" + phone[len(phone)-4:]
		}
		return phone
	}
	return "***"
}

func notAuthorizedCognitoUserPools(message string) error {
	return &cognitoUserPoolsAPIError{
		Status:  httpStatusBadRequest,
		Code:    "NotAuthorizedException",
		Message: message,
	}
}
