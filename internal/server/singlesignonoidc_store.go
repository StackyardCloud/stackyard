package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type singleSignOnOIDCStore struct {
	mu sync.Mutex

	nextClient int64
	nextDevice int64
	nextToken  int64

	clients      map[string]map[string]any
	clientByName map[string]string
	deviceCodes  map[string]map[string]any
}

func newSingleSignOnOIDCStore() *singleSignOnOIDCStore {
	now := time.Now().UTC().Unix()
	s := &singleSignOnOIDCStore{
		nextClient:   1,
		nextDevice:   1,
		nextToken:    1,
		clients:      map[string]map[string]any{},
		clientByName: map[string]string{},
		deviceCodes:  map[string]map[string]any{},
	}

	clientID := "stackyard-client-id"
	clientSecret := "stackyard-client-secret"
	s.clients[clientID] = map[string]any{
		"clientName":            "stackyard-client",
		"clientType":            "public",
		"clientId":              clientID,
		"clientSecret":          clientSecret,
		"clientIdIssuedAt":      now,
		"clientSecretExpiresAt": now + int64((24*time.Hour).Seconds()*3650),
		"authorizationEndpoint": "https://device.sso.us-east-1.amazonaws.com/authorize",
		"tokenEndpoint":         "https://oidc.us-east-1.amazonaws.com/token",
		"scopes":                []any{"sso:account:access"},
		"grantTypes":            []any{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		"redirectUris":          []any{},
	}
	s.clientByName["stackyard-client"] = clientID
	s.deviceCodes["stackyard-device-code"] = map[string]any{
		"clientId":   clientID,
		"userCode":   "STACK-YARD",
		"deviceCode": "stackyard-device-code",
		"expiresAt":  now + 600,
		"expiresIn":  600,
		"interval":   5,
	}
	return s
}

func (s *singleSignOnOIDCStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Unix()

	switch action {
	case "RegisterClient":
		clientName := ssoOIDCPayloadString(payload, "clientName", fmt.Sprintf("stackyard-client-%06d", s.nextClient))
		clientType := ssoOIDCPayloadString(payload, "clientType", "public")

		clientID := s.clientByName[clientName]
		clientSecret := ""
		if clientID == "" {
			clientID = fmt.Sprintf("stackyard-client-id-%06d", s.nextClient)
			clientSecret = fmt.Sprintf("stackyard-client-secret-%06d", s.nextClient)
			s.nextClient++
		}
		if clientSecret == "" {
			if existing, ok := s.clients[clientID]; ok {
				clientSecret = ssoOIDCPayloadString(existing, "clientSecret", "")
			}
			if clientSecret == "" {
				clientSecret = fmt.Sprintf("stackyard-client-secret-%06d", s.nextClient)
			}
		}

		client := map[string]any{
			"clientName":            clientName,
			"clientType":            clientType,
			"clientId":              clientID,
			"clientSecret":          clientSecret,
			"clientIdIssuedAt":      now,
			"clientSecretExpiresAt": now + int64((24*time.Hour).Seconds()*3650),
			"authorizationEndpoint": "https://device.sso.us-east-1.amazonaws.com/authorize",
			"tokenEndpoint":         "https://oidc.us-east-1.amazonaws.com/token",
			"scopes":                ssoOIDCPayloadSlice(payload, "scopes"),
			"grantTypes":            ssoOIDCPayloadSlice(payload, "grantTypes"),
			"redirectUris":          ssoOIDCPayloadSlice(payload, "redirectUris"),
		}
		s.clients[clientID] = client
		s.clientByName[clientName] = clientID

		return map[string]any{
			"clientId":              clientID,
			"clientSecret":          clientSecret,
			"clientIdIssuedAt":      client["clientIdIssuedAt"],
			"clientSecretExpiresAt": client["clientSecretExpiresAt"],
			"authorizationEndpoint": client["authorizationEndpoint"],
			"tokenEndpoint":         client["tokenEndpoint"],
		}

	case "StartDeviceAuthorization":
		clientID := ssoOIDCPayloadString(payload, "clientId", "stackyard-client-id")
		clientSecret := ssoOIDCPayloadString(payload, "clientSecret", "stackyard-client-secret")
		startURL := ssoOIDCPayloadString(payload, "startUrl", "https://stackyard.awsapps.com/start")

		if _, ok := s.clients[clientID]; !ok {
			s.clients[clientID] = map[string]any{
				"clientName":            "stackyard-client",
				"clientType":            "public",
				"clientId":              clientID,
				"clientSecret":          clientSecret,
				"clientIdIssuedAt":      now,
				"clientSecretExpiresAt": now + int64((24*time.Hour).Seconds()*3650),
				"authorizationEndpoint": "https://device.sso.us-east-1.amazonaws.com/authorize",
				"tokenEndpoint":         "https://oidc.us-east-1.amazonaws.com/token",
			}
		}

		deviceCode := fmt.Sprintf("stackyard-device-code-%06d", s.nextDevice)
		userCode := fmt.Sprintf("STACK-%06d", s.nextDevice)
		s.nextDevice++

		expiresIn := int64(600)
		interval := int64(5)
		s.deviceCodes[deviceCode] = map[string]any{
			"clientId":   clientID,
			"userCode":   userCode,
			"deviceCode": deviceCode,
			"startUrl":   startURL,
			"expiresAt":  now + expiresIn,
			"expiresIn":  expiresIn,
			"interval":   interval,
		}

		return map[string]any{
			"deviceCode":              deviceCode,
			"userCode":                userCode,
			"verificationUri":         strings.TrimRight(startURL, "/") + "/verify",
			"verificationUriComplete": strings.TrimRight(startURL, "/") + "/verify?user_code=" + userCode,
			"expiresIn":               expiresIn,
			"interval":                interval,
		}

	case "CreateToken":
		clientID := ssoOIDCPayloadString(payload, "clientId", "stackyard-client-id")
		deviceCode := ssoOIDCPayloadString(payload, "deviceCode", "stackyard-device-code")
		if _, ok := s.deviceCodes[deviceCode]; !ok {
			s.deviceCodes[deviceCode] = map[string]any{
				"clientId":   clientID,
				"userCode":   "STACK-YARD",
				"deviceCode": deviceCode,
				"expiresAt":  now + 600,
				"expiresIn":  int64(600),
				"interval":   int64(5),
			}
		}
		tokenSuffix := fmt.Sprintf("%06d", s.nextToken)
		s.nextToken++
		return map[string]any{
			"accessToken":  "stackyard-access-token-" + tokenSuffix,
			"tokenType":    "Bearer",
			"expiresIn":    int64(3600),
			"refreshToken": "stackyard-refresh-token-" + tokenSuffix,
			"idToken":      "stackyard-id-token-" + tokenSuffix,
		}

	case "CreateTokenWithIAM":
		tokenSuffix := fmt.Sprintf("%06d", s.nextToken)
		s.nextToken++
		return map[string]any{
			"accessToken":     "stackyard-access-token-" + tokenSuffix,
			"tokenType":       "Bearer",
			"expiresIn":       int64(3600),
			"refreshToken":    "stackyard-refresh-token-" + tokenSuffix,
			"idToken":         "stackyard-id-token-" + tokenSuffix,
			"issuedTokenType": "urn:ietf:params:oauth:token-type:access_token",
			"scope":           []any{"sso:account:access"},
		}
	}

	return map[string]any{}
}

func ssoOIDCPayloadString(payload map[string]any, key, fallback string) string {
	for k, value := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s, ok := value.(string); ok {
			if strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
			return fallback
		}
	}
	return fallback
}

func ssoOIDCPayloadSlice(payload map[string]any, key string) []any {
	for k, value := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch v := value.(type) {
		case []any:
			out := make([]any, 0, len(v))
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					continue
				}
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
			return out
		case []string:
			out := make([]any, 0, len(v))
			for _, item := range v {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
	}
	return []any{}
}
