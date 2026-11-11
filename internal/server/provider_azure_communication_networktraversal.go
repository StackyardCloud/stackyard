package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationNetworkTraversalAPIVersion = "2022-03-01-preview"
const azureCommunicationNetworkTraversalPrefix = "/azure/communication-networktraversal/2022-03-01-preview"
const azureCommunicationNetworkTraversalDefaultTTLSeconds = 172800

type azureCommunicationNetworkTraversalConfiguration struct {
	RequestDigest  string
	IdentityID     string
	EffectiveTTL   int
	RequestedRoute string
	ExpiresOn      time.Time
	IceServers     []map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Server) handleAzureCommunicationNetworkTraversalRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationNetworkTraversalPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationNetworkTraversalAPIVersion {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2022-03-01-preview")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationNetworkTraversalPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) != 2 || !strings.EqualFold(segments[0], "networkTraversal") || !strings.EqualFold(segments[1], ":issueRelayConfiguration") {
		azureCommunicationNetworkTraversalError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	s.handleAzureCommunicationNetworkTraversalIssueRelayConfiguration(w, r)
	return true
}

func (s *Server) handleAzureCommunicationNetworkTraversalIssueRelayConfiguration(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationNetworkTraversalReadObject(w, r)
	if !ok {
		return
	}

	identityID := strings.TrimSpace(azureCommunicationNetworkTraversalString(payload["id"]))
	if identityID != "" && !azureCommunicationNetworkTraversalValidIdentityID(identityID) {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidRequest", "id must be a valid ACS identity")
		return
	}

	requestedRoute := azureCommunicationNetworkTraversalNormalizeRouteType(azureCommunicationNetworkTraversalString(payload["routeType"]))
	if routeRaw := strings.TrimSpace(azureCommunicationNetworkTraversalString(payload["routeType"])); routeRaw != "" && requestedRoute == "" {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidRequest", "routeType must be one of: any, nearest")
		return
	}

	effectiveTTL, ok := azureCommunicationNetworkTraversalParseTTL(payload["ttl"])
	if !ok {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidRequest", "ttl must be an integer between 0 and 172800")
		return
	}

	now := time.Now().UTC()
	requestDigest := azureCommunicationNetworkTraversalDigest(identityID + "|" + requestedRoute + "|" + strconv.Itoa(effectiveTTL) + "|" + string(body))

	s.providerStorageMu.Lock()
	config := s.azureCommunicationNetworkTraversalConfigurations[requestDigest]
	if config == nil {
		config = &azureCommunicationNetworkTraversalConfiguration{
			RequestDigest:  requestDigest,
			IdentityID:     identityID,
			EffectiveTTL:   effectiveTTL,
			RequestedRoute: requestedRoute,
			ExpiresOn:      now.Add(time.Duration(effectiveTTL) * time.Second),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		config.IceServers = azureCommunicationNetworkTraversalIssueIceServers(config)
		s.azureCommunicationNetworkTraversalConfigurations[requestDigest] = config
	} else {
		config.UpdatedAt = now
	}
	fixture := azureCommunicationNetworkTraversalConfigurationFixture(config)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, fixture)
}

func azureCommunicationNetworkTraversalConfigurationFixture(config *azureCommunicationNetworkTraversalConfiguration) map[string]any {
	servers := make([]any, 0, len(config.IceServers))
	for _, server := range config.IceServers {
		servers = append(servers, azureCommunicationNetworkTraversalCloneObject(server))
	}
	return map[string]any{
		"expiresOn":  config.ExpiresOn.Format(time.RFC3339Nano),
		"iceServers": servers,
	}
}

func azureCommunicationNetworkTraversalIssueIceServers(config *azureCommunicationNetworkTraversalConfiguration) []map[string]any {
	routeTypes := []string{"any", "nearest"}
	if config.RequestedRoute != "" {
		routeTypes = []string{config.RequestedRoute}
	}

	servers := make([]map[string]any, 0, len(routeTypes))
	for _, routeType := range routeTypes {
		digest := azureCommunicationNetworkTraversalDigest(config.RequestDigest + "|" + routeType)
		servers = append(servers, map[string]any{
			"urls": []any{
				"turn:relay.communication.azure.local:3478?transport=udp",
				"stun:relay.communication.azure.local:3478",
			},
			"username":   "relay-user-" + digest[:16],
			"credential": "relay-cred-" + digest[16:32],
			"routeType":  strings.ToLower(routeType),
		})
	}
	return servers
}

func azureCommunicationNetworkTraversalReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationNetworkTraversalError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationNetworkTraversalParseTTL(value any) (int, bool) {
	if value == nil {
		return azureCommunicationNetworkTraversalDefaultTTLSeconds, true
	}
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		if typed < 0 {
			return 0, false
		}
		ttl := int(typed)
		if ttl > azureCommunicationNetworkTraversalDefaultTTLSeconds {
			return azureCommunicationNetworkTraversalDefaultTTLSeconds, true
		}
		return ttl, true
	default:
		return 0, false
	}
}

func azureCommunicationNetworkTraversalNormalizeRouteType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "any":
		return "any"
	case "nearest":
		return "nearest"
	default:
		return ""
	}
}

func azureCommunicationNetworkTraversalValidIdentityID(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "8:acs:") && len(value) > len("8:acs:")
}

func azureCommunicationNetworkTraversalCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationNetworkTraversalCloneObject(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCommunicationNetworkTraversalCloneObject(object)
				} else {
					cloned[i] = item
				}
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func azureCommunicationNetworkTraversalString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationNetworkTraversalDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationNetworkTraversalError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
