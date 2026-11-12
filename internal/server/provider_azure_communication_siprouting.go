package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationSipRoutingAPIVersion = "2023-03-01"
const azureCommunicationSipRoutingPrefix = "/azure/communication-siprouting/2023-03-01"

type azureCommunicationSipRoutingConfiguration struct {
	Trunks    map[string]*azureCommunicationSipRoutingTrunk
	Routes    []*azureCommunicationSipRoutingRoute
	CreatedAt time.Time
	UpdatedAt time.Time
}

type azureCommunicationSipRoutingTrunk struct {
	FQDN             string
	SIPSignalingPort int
}

type azureCommunicationSipRoutingRoute struct {
	Name          string
	Description   string
	NumberPattern string
	Trunks        []string
}

func (s *Server) handleAzureCommunicationSipRoutingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationSipRoutingPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationSipRoutingAPIVersion {
		azureCommunicationSipRoutingError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2023-03-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationSipRoutingPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) != 1 || !strings.EqualFold(segments[0], "sip") {
		azureCommunicationSipRoutingError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureCommunicationSipRoutingGet(w)
	case http.MethodPatch:
		s.handleAzureCommunicationSipRoutingPatch(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func (s *Server) handleAzureCommunicationSipRoutingGet(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	fixture := azureCommunicationSipRoutingConfigurationFixture(s.azureCommunicationSipRoutingEnsureConfigurationLocked())
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationSipRoutingPatch(w http.ResponseWriter, r *http.Request) {
	if !azureCommunicationSipRoutingHasMergePatchContentType(r) {
		azureCommunicationSipRoutingError(w, http.StatusUnsupportedMediaType, "UnsupportedMediaType", "Content-Type must be application/merge-patch+json")
		return
	}

	payload, ok := azureCommunicationSipRoutingReadObject(w, r)
	if !ok {
		return
	}

	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	current := s.azureCommunicationSipRoutingEnsureConfigurationLocked()
	next := azureCommunicationSipRoutingCloneConfiguration(current)

	if raw, exists := payload["trunks"]; exists {
		trunks, errMessage := azureCommunicationSipRoutingParseTrunks(raw)
		if errMessage != "" {
			s.providerStorageMu.Unlock()
			azureCommunicationSipRoutingError(w, http.StatusBadRequest, "InvalidRequest", errMessage)
			return
		}
		next.Trunks = trunks
	}

	if raw, exists := payload["routes"]; exists {
		routes, errMessage := azureCommunicationSipRoutingParseRoutes(raw, next.Trunks)
		if errMessage != "" {
			s.providerStorageMu.Unlock()
			azureCommunicationSipRoutingError(w, http.StatusBadRequest, "InvalidRequest", errMessage)
			return
		}
		next.Routes = routes
	}

	next.UpdatedAt = now
	if next.CreatedAt.IsZero() {
		next.CreatedAt = now
	}
	s.azureCommunicationSipRoutingConfiguration = next
	fixture := azureCommunicationSipRoutingConfigurationFixture(next)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) azureCommunicationSipRoutingEnsureConfigurationLocked() *azureCommunicationSipRoutingConfiguration {
	if s.azureCommunicationSipRoutingConfiguration != nil {
		return s.azureCommunicationSipRoutingConfiguration
	}

	now := time.Now().UTC()
	s.azureCommunicationSipRoutingConfiguration = &azureCommunicationSipRoutingConfiguration{
		Trunks: map[string]*azureCommunicationSipRoutingTrunk{
			"vendor.xyz.biz": {
				FQDN:             "vendor.xyz.biz",
				SIPSignalingPort: 5061,
			},
			"vendor.abc.biz": {
				FQDN:             "vendor.abc.biz",
				SIPSignalingPort: 5061,
			},
		},
		Routes: []*azureCommunicationSipRoutingRoute{
			{
				Name:          "default-route",
				Description:   "Default PSTN route",
				NumberPattern: "^\\+1[2-9]\\d{9}$",
				Trunks:        []string{"vendor.xyz.biz"},
			},
			{
				Name:          "fallback-route",
				Description:   "Fallback international route",
				NumberPattern: "^\\+\\d+$",
				Trunks:        []string{"vendor.xyz.biz", "vendor.abc.biz"},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.azureCommunicationSipRoutingConfiguration
}

func azureCommunicationSipRoutingConfigurationFixture(config *azureCommunicationSipRoutingConfiguration) map[string]any {
	trunks := make(map[string]any, len(config.Trunks))
	for fqdn, trunk := range config.Trunks {
		trunks[fqdn] = map[string]any{
			"sipSignalingPort": trunk.SIPSignalingPort,
		}
	}

	routes := make([]any, 0, len(config.Routes))
	for _, route := range config.Routes {
		item := map[string]any{
			"name":          route.Name,
			"numberPattern": route.NumberPattern,
		}
		if route.Description != "" {
			item["description"] = route.Description
		}
		if len(route.Trunks) > 0 {
			trunksList := make([]any, 0, len(route.Trunks))
			for _, trunk := range route.Trunks {
				trunksList = append(trunksList, trunk)
			}
			item["trunks"] = trunksList
		}
		routes = append(routes, item)
	}

	return map[string]any{
		"trunks": trunks,
		"routes": routes,
	}
}

func azureCommunicationSipRoutingReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationSipRoutingError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationSipRoutingError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func azureCommunicationSipRoutingHasMergePatchContentType(r *http.Request) bool {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		return false
	}
	baseType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return baseType == "application/merge-patch+json"
}

func azureCommunicationSipRoutingParseTrunks(value any) (map[string]*azureCommunicationSipRoutingTrunk, string) {
	if value == nil {
		return map[string]*azureCommunicationSipRoutingTrunk{}, ""
	}

	items, ok := value.(map[string]any)
	if !ok {
		return nil, "trunks must be an object keyed by FQDN"
	}

	trunks := make(map[string]*azureCommunicationSipRoutingTrunk, len(items))
	for rawKey, rawValue := range items {
		fqdn := strings.TrimSpace(rawKey)
		if errMessage := azureCommunicationSipRoutingValidateFQDN(fqdn); errMessage != "" {
			return nil, errMessage
		}

		object, ok := rawValue.(map[string]any)
		if !ok {
			return nil, "each trunk must be an object"
		}

		port, ok := azureCommunicationSipRoutingParsePort(object["sipSignalingPort"])
		if !ok {
			return nil, "trunk sipSignalingPort must be an integer between 1 and 65535"
		}

		trunks[fqdn] = &azureCommunicationSipRoutingTrunk{
			FQDN:             fqdn,
			SIPSignalingPort: port,
		}
	}

	return trunks, ""
}

func azureCommunicationSipRoutingParseRoutes(value any, trunks map[string]*azureCommunicationSipRoutingTrunk) ([]*azureCommunicationSipRoutingRoute, string) {
	if value == nil {
		return []*azureCommunicationSipRoutingRoute{}, ""
	}

	items, ok := value.([]any)
	if !ok {
		return nil, "routes must be an array"
	}

	routes := make([]*azureCommunicationSipRoutingRoute, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, "each route must be an object"
		}

		name := strings.TrimSpace(azureCommunicationSipRoutingString(item["name"]))
		if name == "" {
			return nil, "route name is required"
		}
		if len(name) > 256 {
			return nil, "route name must be 256 characters or fewer"
		}

		description := strings.TrimSpace(azureCommunicationSipRoutingString(item["description"]))
		if len(description) > 1024 {
			return nil, "route description must be 1024 characters or fewer"
		}

		numberPattern := strings.TrimSpace(azureCommunicationSipRoutingString(item["numberPattern"]))
		if numberPattern == "" {
			return nil, "route numberPattern is required"
		}
		if len(numberPattern) > 1024 {
			return nil, "route numberPattern must be 1024 characters or fewer"
		}

		routeTrunks, errMessage := azureCommunicationSipRoutingParseRouteTrunks(item["trunks"], trunks)
		if errMessage != "" {
			return nil, errMessage
		}

		routes = append(routes, &azureCommunicationSipRoutingRoute{
			Name:          name,
			Description:   description,
			NumberPattern: numberPattern,
			Trunks:        routeTrunks,
		})
	}

	return routes, ""
}

func azureCommunicationSipRoutingParseRouteTrunks(value any, trunks map[string]*azureCommunicationSipRoutingTrunk) ([]string, string) {
	if value == nil {
		return nil, ""
	}

	items, ok := value.([]any)
	if !ok {
		return nil, "route trunks must be an array of FQDN strings"
	}

	routeTrunks := make([]string, 0, len(items))
	for _, rawItem := range items {
		fqdn := strings.TrimSpace(azureCommunicationSipRoutingString(rawItem))
		if errMessage := azureCommunicationSipRoutingValidateFQDN(fqdn); errMessage != "" {
			return nil, errMessage
		}
		if _, exists := trunks[fqdn]; !exists {
			return nil, "route trunk references must refer to configured trunks"
		}
		routeTrunks = append(routeTrunks, fqdn)
	}

	return routeTrunks, ""
}

func azureCommunicationSipRoutingParsePort(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		port := int(typed)
		return port, port >= 1 && port <= 65535
	case int:
		return typed, typed >= 1 && typed <= 65535
	default:
		return 0, false
	}
}

func azureCommunicationSipRoutingValidateFQDN(value string) string {
	if value == "" {
		return "trunk FQDN is required"
	}
	if len(value) > 249 {
		return "trunk FQDN must be 249 characters or fewer"
	}
	if strings.ContainsAny(value, " /") {
		return "trunk FQDN must be a valid hostname"
	}
	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return "trunk FQDN must be a valid hostname"
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return "trunk FQDN must be a valid hostname"
		}
		for i, ch := range label {
			isAlphaNum := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
			if isAlphaNum {
				continue
			}
			if ch == '-' && i > 0 && i < len(label)-1 {
				continue
			}
			return "trunk FQDN must be a valid hostname"
		}
	}
	return ""
}

func azureCommunicationSipRoutingString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationSipRoutingCloneConfiguration(input *azureCommunicationSipRoutingConfiguration) *azureCommunicationSipRoutingConfiguration {
	if input == nil {
		return &azureCommunicationSipRoutingConfiguration{
			Trunks: map[string]*azureCommunicationSipRoutingTrunk{},
			Routes: []*azureCommunicationSipRoutingRoute{},
		}
	}

	trunks := make(map[string]*azureCommunicationSipRoutingTrunk, len(input.Trunks))
	for key, trunk := range input.Trunks {
		if trunk == nil {
			continue
		}
		cloned := *trunk
		trunks[key] = &cloned
	}

	routes := make([]*azureCommunicationSipRoutingRoute, 0, len(input.Routes))
	for _, route := range input.Routes {
		if route == nil {
			continue
		}
		cloned := &azureCommunicationSipRoutingRoute{
			Name:          route.Name,
			Description:   route.Description,
			NumberPattern: route.NumberPattern,
		}
		if len(route.Trunks) > 0 {
			cloned.Trunks = append([]string(nil), route.Trunks...)
		}
		routes = append(routes, cloned)
	}

	return &azureCommunicationSipRoutingConfiguration{
		Trunks:    trunks,
		Routes:    routes,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
	}
}

func azureCommunicationSipRoutingError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"target":  strconv.Itoa(status),
		},
	})
}
