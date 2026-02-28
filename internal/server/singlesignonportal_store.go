package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type singleSignOnPortalToken struct {
	Active    bool
	ExpiresAt int64
}

type singleSignOnPortalStore struct {
	mu sync.Mutex

	nextID int64

	tokens   map[string]singleSignOnPortalToken
	accounts map[string]map[string]any
	roles    map[string][]map[string]any
}

func newSingleSignOnPortalStore() *singleSignOnPortalStore {
	now := time.Now().UTC().Unix()
	// Keep the seeded coverage token effectively long-lived so full endpoint
	// sweeps do not start failing mid-run due to token expiry.
	defaultTokenExpiry := now + 10*365*24*60*60
	return &singleSignOnPortalStore{
		nextID: 1,
		tokens: map[string]singleSignOnPortalToken{
			"stackyard-access-token": {Active: true, ExpiresAt: defaultTokenExpiry},
		},
		accounts: map[string]map[string]any{
			"123456789012": {
				"accountId":    "123456789012",
				"accountName":  "stackyard-account",
				"emailAddress": "owner@stackyard.example",
			},
		},
		roles: map[string][]map[string]any{
			"123456789012": {
				{
					"roleName":  "stackyard-role",
					"accountId": "123456789012",
				},
				{
					"roleName":  "ReadOnly",
					"accountId": "123456789012",
				},
			},
		},
	}
}

func (s *singleSignOnPortalStore) Handle(action string, payload map[string]any) (map[string]any, int, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := portalPayloadString(payload, "accessToken", "")
	if token == "" {
		return nil, 401, "UnauthorizedException", "access token is required"
	}
	tokenState, ok := s.tokens[token]
	now := time.Now().UTC().Unix()
	if !ok || !tokenState.Active || tokenState.ExpiresAt <= now {
		return nil, 401, "UnauthorizedException", "invalid or expired access token"
	}

	switch action {
	case "ListAccounts":
		keys := make([]string, 0, len(s.accounts))
		for id := range s.accounts {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		accountList := make([]any, 0, len(keys))
		for _, id := range keys {
			accountList = append(accountList, cloneAnyMap(s.accounts[id]))
		}
		return map[string]any{
			"accountList": accountList,
			"nextToken":   "",
		}, 0, "", ""

	case "ListAccountRoles":
		accountID := portalPayloadString(payload, "accountId", "")
		if strings.TrimSpace(accountID) == "" {
			return nil, 400, "ValidationException", "accountId is required"
		}
		if _, ok := s.accounts[accountID]; !ok {
			return nil, 404, "ResourceNotFoundException", "account not found"
		}
		roles := s.roles[accountID]
		roleList := make([]any, 0, len(roles))
		for _, role := range roles {
			roleList = append(roleList, cloneAnyMap(role))
		}
		return map[string]any{
			"roleList":  roleList,
			"nextToken": "",
		}, 0, "", ""

	case "GetRoleCredentials":
		accountID := portalPayloadString(payload, "accountId", "")
		roleName := portalPayloadString(payload, "roleName", "")
		if strings.TrimSpace(accountID) == "" || strings.TrimSpace(roleName) == "" {
			return nil, 400, "ValidationException", "accountId and roleName are required"
		}
		if _, ok := s.accounts[accountID]; !ok {
			return nil, 404, "ResourceNotFoundException", "account not found"
		}
		if !s.accountHasRole(accountID, roleName) {
			return nil, 404, "ResourceNotFoundException", "role not found"
		}
		suffix := fmt.Sprintf("%06d", s.nextID)
		s.nextID++
		return map[string]any{
			"roleCredentials": map[string]any{
				"accessKeyId":     "ASIASTACKYARD" + suffix,
				"secretAccessKey": "stackyard-secret-" + suffix,
				"sessionToken":    "stackyard-session-" + suffix,
				"expiration":      (now + 3600) * 1000,
			},
		}, 0, "", ""

	case "Logout":
		tokenState.Active = false
		s.tokens[token] = tokenState
		return map[string]any{}, 0, "", ""
	}

	return map[string]any{}, 0, "", ""
}

func (s *singleSignOnPortalStore) accountHasRole(accountID, roleName string) bool {
	for _, role := range s.roles[accountID] {
		if strings.EqualFold(portalPayloadString(role, "roleName", ""), roleName) {
			return true
		}
	}
	return false
}

func portalPayloadString(payload map[string]any, key, fallback string) string {
	for k, value := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s, ok := value.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
			return fallback
		}
	}
	return fallback
}
