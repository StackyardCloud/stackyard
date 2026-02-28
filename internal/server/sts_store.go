package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type stsStore struct {
	mu             sync.Mutex
	sessionCounter int
}

func newSTSStore() *stsStore {
	return &stsStore{}
}

func (s *stsStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "GetCallerIdentity":
		return map[string]any{
			"Account": "123456789012",
			"Arn":     "arn:aws:sts::123456789012:assumed-role/stackyard-role/stackyard-session",
			"UserId":  "AROASTACKYARDROLE:stackyard-session",
		}
	case "GetAccessKeyInfo":
		return map[string]any{
			"Account": "123456789012",
		}
	case "DecodeAuthorizationMessage":
		return map[string]any{
			"DecodedMessage": "{}",
		}
	case "GetWebIdentityToken":
		return map[string]any{
			"WebIdentityToken": "stackyard-web-identity-token",
		}
	case "GetDelegatedAccessToken":
		return map[string]any{
			"DelegatedAccessToken": "stackyard-delegated-access-token",
		}
	case "GetSessionToken":
		return map[string]any{
			"Credentials": s.issueCredentials("stackyard-session-token"),
		}
	case "GetFederationToken":
		name := strings.TrimSpace(form.Get("Name"))
		if name == "" {
			name = "stackyard"
		}
		return map[string]any{
			"Credentials": s.issueCredentials(name),
			"FederatedUser": map[string]any{
				"Arn":             "arn:aws:sts::123456789012:federated-user/" + name,
				"FederatedUserId": "123456789012:" + name,
			},
		}
	case "AssumeRole", "AssumeRoleWithSAML", "AssumeRoleWithWebIdentity", "AssumeRoot":
		sessionName := strings.TrimSpace(form.Get("RoleSessionName"))
		if sessionName == "" {
			sessionName = "stackyard-session"
		}
		return map[string]any{
			"Credentials": s.issueCredentials(sessionName),
			"AssumedRoleUser": map[string]any{
				"Arn":           "arn:aws:sts::123456789012:assumed-role/stackyard-role/" + sessionName,
				"AssumedRoleId": "AROASTACKYARDROLE:" + sessionName,
			},
		}
	default:
		return map[string]any{}
	}
}

func (s *stsStore) issueCredentials(name string) map[string]any {
	s.sessionCounter++
	index := s.sessionCounter
	serial := fmt.Sprintf("%06d", index)
	return map[string]any{
		"AccessKeyId":     "ASIASTACKYARD" + serial,
		"SecretAccessKey": "stackyard-secret-" + serial,
		"SessionToken":    "stackyard-token-" + serial,
		"Expiration":      "2026-12-31T23:59:59Z",
	}
}
