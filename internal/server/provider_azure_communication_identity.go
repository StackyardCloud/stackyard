package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureCommunicationIdentityAPIVersion = "2025-06-30"
const azureCommunicationIdentityPrefix = "/azure/communication-identity/2025-06-30"

type azureCommunicationIdentityIdentity struct {
	ID         string
	Tokens     map[string]*azureCommunicationIdentityToken
	RevokedAt  *time.Time
	DeletedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	IssueCount int
}

type azureCommunicationIdentityToken struct {
	Token       string
	Scopes      []string
	ExpiresOn   time.Time
	Kind        string
	CreatedAt   time.Time
	Revoked     bool
	SourceToken string
}

type azureCommunicationIdentityAssignment struct {
	TenantID      string
	ObjectID      string
	PrincipalType string
	ClientIDs     []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (s *Server) handleAzureCommunicationIdentityRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationIdentityPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationIdentityAPIVersion {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-06-30")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationIdentityPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "identities":
		s.handleAzureCommunicationIdentityIdentities(w, r, segments[1:])
		return true
	case "teamsuser":
		if len(segments) != 2 || !strings.EqualFold(segments[1], ":exchangeAccessToken") {
			azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCommunicationIdentityExchangeTeamsUserAccessToken(w, r)
		return true
	case "access":
		s.handleAzureCommunicationIdentityAccess(w, r, segments[1:])
		return true
	default:
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureCommunicationIdentityIdentities(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationIdentityCreate(w, r)
		return
	}

	identityID := strings.TrimSpace(tail[0])
	if identityID == "" {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "id is required")
		return
	}

	if len(tail) == 1 {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationIdentityDelete(w, identityID)
		return
	}

	if len(tail) != 2 {
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case ":issueaccesstoken":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationIdentityIssueAccessToken(w, r, identityID)
	case ":revokeaccesstokens":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationIdentityRevokeAccessTokens(w, identityID)
	default:
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationIdentityAccess(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if !strings.EqualFold(tail[0], "teamsExtension") {
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], ":exchangeAccessToken") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationIdentityExchangeTeamsExtensionAccessToken(w, r)
		return
	}
	if len(tail) != 5 || !strings.EqualFold(tail[1], "tenants") || !strings.EqualFold(tail[3], "assignments") {
		azureCommunicationIdentityError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	tenantID := strings.TrimSpace(tail[2])
	objectID := strings.TrimSpace(tail[4])
	if tenantID == "" || objectID == "" {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "tenantId and objectId are required")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleAzureCommunicationIdentityUpsertAssignment(w, r, tenantID, objectID)
	case http.MethodGet:
		s.handleAzureCommunicationIdentityGetAssignment(w, tenantID, objectID)
	case http.MethodDelete:
		s.handleAzureCommunicationIdentityDeleteAssignment(w, tenantID, objectID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationIdentityCreate(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationIdentityReadObject(w, r)
	if !ok {
		return
	}

	scopes, hasScopes, ok := azureCommunicationIdentityScopes(payload["createTokenWithScopes"])
	if !ok {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "createTokenWithScopes must contain valid scopes")
		return
	}
	expiresInMinutes, ok := azureCommunicationIdentityParseExpiresInMinutes(payload["expiresInMinutes"], false)
	if !ok {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "expiresInMinutes must be between 60 and 1440")
		return
	}

	now := time.Now().UTC()
	identityID := azureCommunicationIdentityDeterministicID("8:acs", string(body), now)
	identity := &azureCommunicationIdentityIdentity{
		ID:        identityID,
		Tokens:    map[string]*azureCommunicationIdentityToken{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if hasScopes {
		token := azureCommunicationIdentityMintToken(identityID, scopes, expiresInMinutes, "create", "", now)
		identity.Tokens[token.Token] = token
	}

	s.providerStorageMu.Lock()
	s.azureCommunicationIdentityIdentities[identityID] = identity
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureCommunicationIdentityCreateResultFixture(identity, hasScopes))
}

func (s *Server) handleAzureCommunicationIdentityDelete(w http.ResponseWriter, identityID string) {
	s.providerStorageMu.Lock()
	identity := s.azureCommunicationIdentityIdentities[identityID]
	if identity == nil || identity.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationIdentityError(w, http.StatusNotFound, "IdentityNotFound", "identity not found")
		return
	}
	now := time.Now().UTC()
	identity.DeletedAt = &now
	identity.UpdatedAt = now
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationIdentityIssueAccessToken(w http.ResponseWriter, r *http.Request, identityID string) {
	payload, _, ok := azureCommunicationIdentityReadObject(w, r)
	if !ok {
		return
	}

	scopes, hasScopes, ok := azureCommunicationIdentityScopes(payload["scopes"])
	if !ok || !hasScopes {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "scopes must contain at least one valid scope")
		return
	}
	expiresInMinutes, ok := azureCommunicationIdentityParseExpiresInMinutes(payload["expiresInMinutes"], false)
	if !ok {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "expiresInMinutes must be between 60 and 1440")
		return
	}

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	identity := s.azureCommunicationIdentityIdentities[identityID]
	if identity == nil || identity.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationIdentityError(w, http.StatusNotFound, "IdentityNotFound", "identity not found")
		return
	}
	identity.IssueCount++
	token := azureCommunicationIdentityMintToken(identityID, scopes, expiresInMinutes, "issue", "", now)
	identity.Tokens[token.Token] = token
	identity.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureCommunicationIdentityAccessTokenFixture(token))
}

func (s *Server) handleAzureCommunicationIdentityRevokeAccessTokens(w http.ResponseWriter, identityID string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	identity := s.azureCommunicationIdentityIdentities[identityID]
	if identity == nil || identity.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationIdentityError(w, http.StatusNotFound, "IdentityNotFound", "identity not found")
		return
	}
	for _, token := range identity.Tokens {
		token.Revoked = true
	}
	identity.RevokedAt = &now
	identity.UpdatedAt = now
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationIdentityExchangeTeamsUserAccessToken(w http.ResponseWriter, r *http.Request) {
	payload, _, ok := azureCommunicationIdentityReadObject(w, r)
	if !ok {
		return
	}
	tokenValue := strings.TrimSpace(azureCommunicationIdentityString(payload["token"]))
	appID := strings.TrimSpace(azureCommunicationIdentityString(payload["appId"]))
	userID := strings.TrimSpace(azureCommunicationIdentityString(payload["userId"]))
	if tokenValue == "" || appID == "" || userID == "" {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "token, appId, and userId are required")
		return
	}

	now := time.Now().UTC()
	identityID := azureCommunicationIdentityDigest("teams-user|" + appID + "|" + userID)[:24]
	token := azureCommunicationIdentityMintToken("8:acs:"+identityID, []string{"chat"}, 1440, "teamsUser", tokenValue, now)
	respondJSON(w, http.StatusOK, azureCommunicationIdentityAccessTokenFixture(token))
}

func (s *Server) handleAzureCommunicationIdentityExchangeTeamsExtensionAccessToken(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := azureCommunicationIdentityReadObject(w, r); !ok {
		return
	}

	now := time.Now().UTC()
	identityID := "8:acs:" + azureCommunicationIdentityDigest("teams-extension|" + now.Format("20060102150405"))[:24]
	token := azureCommunicationIdentityMintToken(identityID, []string{"chat"}, 1440, "teamsExtension", "", now)
	identity := &azureCommunicationIdentityIdentity{
		ID:        identityID,
		Tokens:    map[string]*azureCommunicationIdentityToken{token.Token: token},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureCommunicationIdentityIdentities[identityID] = identity
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureCommunicationIdentityCreateResultFixture(identity, true))
}

func (s *Server) handleAzureCommunicationIdentityUpsertAssignment(w http.ResponseWriter, r *http.Request, tenantID, objectID string) {
	payload, _, ok := azureCommunicationIdentityReadObject(w, r)
	if !ok {
		return
	}
	principalType := strings.TrimSpace(azureCommunicationIdentityString(payload["principalType"]))
	if principalType == "" {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "principalType is required")
		return
	}
	if !azureCommunicationIdentityValidPrincipalType(principalType) {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "principalType must be one of resourceAccount or user")
		return
	}
	clientIDs, ok := azureCommunicationIdentityStringList(payload["clientIds"])
	if !ok {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "clientIds must be an array of strings")
		return
	}

	now := time.Now().UTC()
	key := azureCommunicationIdentityAssignmentKey(tenantID, objectID)
	s.providerStorageMu.Lock()
	assignment := s.azureCommunicationIdentityAssignments[key]
	status := http.StatusCreated
	if assignment == nil {
		assignment = &azureCommunicationIdentityAssignment{
			TenantID:  tenantID,
			ObjectID:  objectID,
			CreatedAt: now,
		}
		s.azureCommunicationIdentityAssignments[key] = assignment
	} else {
		status = http.StatusOK
	}
	assignment.PrincipalType = principalType
	assignment.ClientIDs = append([]string(nil), clientIDs...)
	assignment.UpdatedAt = now
	fixture := azureCommunicationIdentityAssignmentFixture(assignment)
	s.providerStorageMu.Unlock()

	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationIdentityGetAssignment(w http.ResponseWriter, tenantID, objectID string) {
	key := azureCommunicationIdentityAssignmentKey(tenantID, objectID)
	s.providerStorageMu.Lock()
	assignment := s.azureCommunicationIdentityAssignments[key]
	s.providerStorageMu.Unlock()
	if assignment == nil {
		azureCommunicationIdentityError(w, http.StatusNotFound, "TeamsExtensionAssignmentNotFound", "teams extension assignment not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationIdentityAssignmentFixture(assignment))
}

func (s *Server) handleAzureCommunicationIdentityDeleteAssignment(w http.ResponseWriter, tenantID, objectID string) {
	key := azureCommunicationIdentityAssignmentKey(tenantID, objectID)
	s.providerStorageMu.Lock()
	if s.azureCommunicationIdentityAssignments[key] == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationIdentityError(w, http.StatusNotFound, "TeamsExtensionAssignmentNotFound", "teams extension assignment not found")
		return
	}
	delete(s.azureCommunicationIdentityAssignments, key)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func azureCommunicationIdentityCreateResultFixture(identity *azureCommunicationIdentityIdentity, includeToken bool) map[string]any {
	result := map[string]any{
		"identity": map[string]any{
			"id": identity.ID,
		},
	}
	if includeToken {
		token := azureCommunicationIdentityLatestToken(identity)
		if token != nil {
			result["accessToken"] = azureCommunicationIdentityAccessTokenFixture(token)
		}
	}
	return result
}

func azureCommunicationIdentityAssignmentFixture(assignment *azureCommunicationIdentityAssignment) map[string]any {
	clientIDs := make([]any, 0, len(assignment.ClientIDs))
	for _, clientID := range assignment.ClientIDs {
		clientIDs = append(clientIDs, clientID)
	}
	return map[string]any{
		"tenantId":      assignment.TenantID,
		"objectId":      assignment.ObjectID,
		"principalType": assignment.PrincipalType,
		"clientIds":     clientIDs,
	}
}

func azureCommunicationIdentityAccessTokenFixture(token *azureCommunicationIdentityToken) map[string]any {
	return map[string]any{
		"token":     token.Token,
		"expiresOn": token.ExpiresOn.Format(time.RFC3339Nano),
	}
}

func azureCommunicationIdentityLatestToken(identity *azureCommunicationIdentityIdentity) *azureCommunicationIdentityToken {
	items := make([]*azureCommunicationIdentityToken, 0, len(identity.Tokens))
	for _, token := range identity.Tokens {
		items = append(items, token)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if len(items) == 0 {
		return nil
	}
	return items[len(items)-1]
}

func azureCommunicationIdentityMintToken(identityID string, scopes []string, expiresInMinutes int, kind, sourceToken string, now time.Time) *azureCommunicationIdentityToken {
	if expiresInMinutes == 0 {
		expiresInMinutes = 1440
	}
	normalizedScopes := append([]string(nil), scopes...)
	sort.Strings(normalizedScopes)
	return &azureCommunicationIdentityToken{
		Token:       "token-" + azureCommunicationIdentityDigest(identityID + "|" + kind + "|" + strings.Join(normalizedScopes, ",") + "|" + sourceToken + "|" + now.Format(time.RFC3339Nano))[:32],
		Scopes:      normalizedScopes,
		ExpiresOn:   now.Add(time.Duration(expiresInMinutes) * time.Minute),
		Kind:        kind,
		CreatedAt:   now,
		SourceToken: sourceToken,
	}
}

func azureCommunicationIdentityScopes(value any) ([]string, bool, bool) {
	if value == nil {
		return nil, false, true
	}
	items, ok := azureCommunicationIdentityStringList(value)
	if !ok || len(items) == 0 {
		return nil, false, false
	}
	scopes := make([]string, 0, len(items))
	for _, item := range items {
		if !azureCommunicationIdentityValidScope(item) {
			return nil, false, false
		}
		scopes = append(scopes, item)
	}
	return scopes, true, true
}

func azureCommunicationIdentityParseExpiresInMinutes(value any, allowZero bool) (int, bool) {
	if value == nil {
		return 1440, true
	}
	switch typed := value.(type) {
	case float64:
		n := int(typed)
		if float64(n) != typed {
			return 0, false
		}
		if n == 0 && allowZero {
			return 0, true
		}
		if n < 60 || n > 1440 {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func azureCommunicationIdentityValidScope(value string) bool {
	switch strings.TrimSpace(value) {
	case "chat", "voip", "chat.join", "chat.join.limited", "voip.join":
		return true
	default:
		return false
	}
}

func azureCommunicationIdentityValidPrincipalType(value string) bool {
	switch strings.TrimSpace(value) {
	case "resourceAccount", "user":
		return true
	default:
		return false
	}
}

func azureCommunicationIdentityAssignmentKey(tenantID, objectID string) string {
	return strings.ToLower(strings.TrimSpace(tenantID) + "|" + strings.TrimSpace(objectID))
}

func azureCommunicationIdentityStringList(value any) ([]string, bool) {
	if value == nil {
		return []string{}, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		item := strings.TrimSpace(azureCommunicationIdentityString(raw))
		if item == "" {
			return nil, false
		}
		out = append(out, item)
	}
	return out, true
}

func azureCommunicationIdentityReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationIdentityError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationIdentityString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationIdentityDeterministicID(prefix, seed string, now time.Time) string {
	return prefix + ":" + azureCommunicationIdentityDigest(seed + "|" + now.Format(time.RFC3339Nano))[:24]
}

func azureCommunicationIdentityDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationIdentityError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
