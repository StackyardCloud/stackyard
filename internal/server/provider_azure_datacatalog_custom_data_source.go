package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const azureDataCatalogCustomDataSourceAPIVersion = "2016-03-30"

var azureDataCatalogCustomDataSourceNamespacePartPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,254}$`)
var azureDataCatalogCustomDataSourceNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,254}$`)
var azureDataCatalogCustomDataSourceIdentityPropertyNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,99}$`)

type azureDataCatalogCustomDataSourceProtocol struct {
	CatalogName        string
	Namespace          string
	Name               string
	IdentityProperties []map[string]any
	IdentitySets       []map[string]any
	ID                 string
	Timestamp          time.Time
}

func (s *Server) handleAzureDataCatalogCustomDataSourceRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/catalogs/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) != 3 || !strings.EqualFold(segments[0], "catalogs") || !strings.EqualFold(segments[2], "dataSourceProtocols") {
		return false
	}

	catalogName := strings.TrimSpace(segments[1])
	if catalogName == "" {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "catalog_name is required")
		return true
	}
	if !azureDataCatalogCustomDataSourceHasExpectedAPIVersion(r) {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "api-version must be 2016-03-30")
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureDataCatalogCustomDataSourceEnumerate(w, catalogName)
	case http.MethodPost:
		s.handleAzureDataCatalogCustomDataSourceRegister(w, r, path, catalogName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func (s *Server) handleAzureDataCatalogCustomDataSourceEnumerate(w http.ResponseWriter, catalogName string) {
	s.providerStorageMu.Lock()
	items := make([]*azureDataCatalogCustomDataSourceProtocol, 0)
	for _, protocol := range s.azureDataCatalogCustomDataSourceProtocols {
		if strings.EqualFold(protocol.CatalogName, catalogName) {
			items = append(items, protocol)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		left := strings.ToLower(items[i].Namespace + "." + items[i].Name)
		right := strings.ToLower(items[j].Namespace + "." + items[j].Name)
		return left < right
	})

	value := make([]any, 0, len(items))
	for _, protocol := range items {
		value = append(value, azureDataCatalogCustomDataSourceProtocolFixture(protocol))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureDataCatalogCustomDataSourceRegister(w http.ResponseWriter, r *http.Request, path, catalogName string) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "content type must be application/json")
		return
	}

	body, err := readBodyBytes(r)
	if err != nil {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "unable to read request body")
		return
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "request body is required")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", "request body must be valid JSON")
		return
	}

	input, err := azureDataCatalogCustomDataSourceNormalizeInput(payload)
	if err != nil {
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "BadRequest", err.Error())
		return
	}

	storageKey := azureDataCatalogCustomDataSourceStorageKey(catalogName, input.Namespace, input.Name)
	s.providerStorageMu.Lock()
	if existing := s.azureDataCatalogCustomDataSourceProtocols[storageKey]; existing != nil {
		s.providerStorageMu.Unlock()
		azureDataCatalogCustomDataSourceError(w, http.StatusBadRequest, path, "ImmutableViewItem", "data source protocol already exists and cannot be updated")
		return
	}

	now := time.Now().UTC()
	protocol := &azureDataCatalogCustomDataSourceProtocol{
		CatalogName:        catalogName,
		Namespace:          input.Namespace,
		Name:               input.Name,
		IdentityProperties: input.IdentityProperties,
		IdentitySets:       input.IdentitySets,
		ID:                 azureDataCatalogCustomDataSourceID(catalogName, input.Namespace, input.Name),
		Timestamp:          now,
	}
	s.azureDataCatalogCustomDataSourceProtocols[storageKey] = protocol
	s.providerStorageMu.Unlock()

	w.Header().Set("Location", protocol.ID)
	respondJSON(w, http.StatusOK, azureDataCatalogCustomDataSourceProtocolFixture(protocol))
}

type azureDataCatalogCustomDataSourceProtocolInput struct {
	Namespace          string
	Name               string
	IdentityProperties []map[string]any
	IdentitySets       []map[string]any
}

func azureDataCatalogCustomDataSourceNormalizeInput(payload map[string]any) (azureDataCatalogCustomDataSourceProtocolInput, error) {
	namespace := strings.TrimSpace(azureDataCatalogCustomDataSourceString(payload["namespace"]))
	if namespace == "" {
		return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("namespace is required")
	}
	if err := azureDataCatalogCustomDataSourceValidateNamespace(namespace); err != nil {
		return azureDataCatalogCustomDataSourceProtocolInput{}, err
	}
	name := strings.TrimSpace(azureDataCatalogCustomDataSourceString(payload["name"]))
	if name == "" {
		return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("name is required")
	}
	if len(name) > 255 || !azureDataCatalogCustomDataSourceNamePattern.MatchString(name) {
		return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("name must start with a letter and contain only letters, numbers, and dashes (1-255 chars)")
	}

	rawIdentityProperties, ok := payload["identityProperties"].([]any)
	if !ok || len(rawIdentityProperties) == 0 || len(rawIdentityProperties) > 20 {
		return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties must contain between 1 and 20 entries")
	}

	identityProperties := make([]map[string]any, 0, len(rawIdentityProperties))
	knownProperties := map[string]struct{}{}
	allowedTypes := map[string]struct{}{
		"bool":    {},
		"boolean": {},
		"byte":    {},
		"guid":    {},
		"int":     {},
		"integer": {},
		"long":    {},
		"string":  {},
		"url":     {},
	}
	for idx, raw := range rawIdentityProperties {
		entry, ok := raw.(map[string]any)
		if !ok {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d] must be an object", idx)
		}
		entryName := strings.TrimSpace(azureDataCatalogCustomDataSourceString(entry["name"]))
		if entryName == "" {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].name is required", idx)
		}
		if !azureDataCatalogCustomDataSourceIdentityPropertyNamePattern.MatchString(entryName) {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].name must start with a letter and contain only letters and numbers (1-100 chars)", idx)
		}
		entryType := strings.ToLower(strings.TrimSpace(azureDataCatalogCustomDataSourceString(entry["type"])))
		if entryType == "" {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].type is required", idx)
		}
		if _, allowed := allowedTypes[entryType]; !allowed {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].type is unsupported", idx)
		}
		propertyKey := strings.ToLower(entryName)
		if _, exists := knownProperties[propertyKey]; exists {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties contains duplicate property %q", entryName)
		}
		knownProperties[propertyKey] = struct{}{}

		normalized := map[string]any{
			"name": entryName,
			"type": entryType,
		}
		if rawIgnoreCase, exists := entry["ignoreCase"]; exists {
			ignoreCase, ok := azureDataCatalogCustomDataSourceBool(rawIgnoreCase)
			if !ok {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].ignoreCase must be a boolean", idx)
			}
			if entryType != "string" {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].ignoreCase can only be specified for type string", idx)
			}
			normalized["ignoreCase"] = ignoreCase
		}
		if rawURLPathSegmentsIgnoreCase, exists := entry["urlPathSegmentsIgnoreCase"]; exists {
			if entryType != "url" {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].urlPathSegmentsIgnoreCase can only be specified for type url", idx)
			}
			parsedValues, ok := azureDataCatalogCustomDataSourceBoolSlice(rawURLPathSegmentsIgnoreCase)
			if !ok {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identityProperties[%d].urlPathSegmentsIgnoreCase must be an array of booleans", idx)
			}
			normalized["urlPathSegmentsIgnoreCase"] = parsedValues
		}
		identityProperties = append(identityProperties, normalized)
	}

	rawIdentitySets, ok := payload["identitySets"].([]any)
	if !ok || len(rawIdentitySets) == 0 || len(rawIdentitySets) > 20 {
		return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets must contain between 1 and 20 entries")
	}

	identitySets := make([]map[string]any, 0, len(rawIdentitySets))
	seenSetKeys := map[string]struct{}{}
	for idx, raw := range rawIdentitySets {
		entry, ok := raw.(map[string]any)
		if !ok {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d] must be an object", idx)
		}
		rawProperties, ok := entry["properties"].([]any)
		if !ok || len(rawProperties) == 0 {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d].properties must contain at least one property", idx)
		}

		properties := make([]string, 0, len(rawProperties))
		seenPropertiesInSet := map[string]struct{}{}
		for propIdx, rawProperty := range rawProperties {
			propertyName := strings.TrimSpace(azureDataCatalogCustomDataSourceString(rawProperty))
			if propertyName == "" {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d].properties[%d] is required", idx, propIdx)
			}
			propertyKey := strings.ToLower(propertyName)
			if _, exists := knownProperties[propertyKey]; !exists {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d].properties[%d] references unknown identity property %q", idx, propIdx, propertyName)
			}
			if _, duplicate := seenPropertiesInSet[propertyKey]; duplicate {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d] contains duplicate property %q", idx, propertyName)
			}
			seenPropertiesInSet[propertyKey] = struct{}{}
			properties = append(properties, propertyName)
		}

		setKey := strings.ToLower(strings.Join(properties, "|"))
		if _, duplicate := seenSetKeys[setKey]; duplicate {
			return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets contains duplicate property set %q", strings.Join(properties, ","))
		}
		seenSetKeys[setKey] = struct{}{}

		normalized := map[string]any{"properties": properties}
		if nameValue, hasName := entry["name"]; hasName {
			identitySetName := strings.TrimSpace(azureDataCatalogCustomDataSourceString(nameValue))
			if identitySetName == "" {
				return azureDataCatalogCustomDataSourceProtocolInput{}, fmt.Errorf("identitySets[%d].name must not be empty when provided", idx)
			}
			normalized["name"] = identitySetName
		}
		identitySets = append(identitySets, normalized)
	}

	return azureDataCatalogCustomDataSourceProtocolInput{
		Namespace:          namespace,
		Name:               name,
		IdentityProperties: identityProperties,
		IdentitySets:       identitySets,
	}, nil
}

func azureDataCatalogCustomDataSourceProtocolFixture(protocol *azureDataCatalogCustomDataSourceProtocol) map[string]any {
	if protocol == nil {
		return map[string]any{}
	}

	identityProperties := make([]any, 0, len(protocol.IdentityProperties))
	for _, property := range protocol.IdentityProperties {
		identityProperties = append(identityProperties, azureDataCatalogCustomDataSourceCloneObject(property))
	}
	identitySets := make([]any, 0, len(protocol.IdentitySets))
	for _, set := range protocol.IdentitySets {
		identitySets = append(identitySets, azureDataCatalogCustomDataSourceCloneObject(set))
	}

	return map[string]any{
		"namespace":          protocol.Namespace,
		"name":               protocol.Name,
		"identityProperties": identityProperties,
		"identitySets":       identitySets,
		"id":                 protocol.ID,
		"timestamp":          protocol.Timestamp.Format(time.RFC3339Nano),
	}
}

func azureDataCatalogCustomDataSourceCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case []string:
			copyValues := make([]string, len(typed))
			copy(copyValues, typed)
			cloned[key] = copyValues
		case []any:
			copyValues := make([]any, len(typed))
			copy(copyValues, typed)
			cloned[key] = copyValues
		default:
			cloned[key] = value
		}
	}
	return cloned
}

func azureDataCatalogCustomDataSourceHasExpectedAPIVersion(r *http.Request) bool {
	values, ok := r.URL.Query()["api-version"]
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) != azureDataCatalogCustomDataSourceAPIVersion {
			return false
		}
	}
	return true
}

func azureDataCatalogCustomDataSourceStorageKey(catalogName, namespace, name string) string {
	return strings.ToLower(strings.TrimSpace(catalogName)) + "|" + strings.ToLower(strings.TrimSpace(namespace)) + "." + strings.ToLower(strings.TrimSpace(name))
}

func azureDataCatalogCustomDataSourceID(catalogName, namespace, name string) string {
	storageKey := azureDataCatalogCustomDataSourceStorageKey(catalogName, namespace, name)
	hash := sha1.Sum([]byte(storageKey))
	protocolID := hex.EncodeToString(hash[:])
	catalogSegment := strings.TrimSpace(catalogName)
	if catalogSegment == "" {
		catalogSegment = "DefaultCatalog"
	}
	return "https://stackyard.local/catalogs/" + catalogSegment + "/dataSourceProtocols/" + protocolID
}

func azureDataCatalogCustomDataSourceString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func azureDataCatalogCustomDataSourceBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "true" {
			return true, true
		}
		if trimmed == "false" {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

func azureDataCatalogCustomDataSourceBoolSlice(value any) ([]bool, bool) {
	rawValues, ok := value.([]any)
	if !ok {
		return nil, false
	}
	values := make([]bool, 0, len(rawValues))
	for _, raw := range rawValues {
		boolValue, ok := azureDataCatalogCustomDataSourceBool(raw)
		if !ok {
			return nil, false
		}
		values = append(values, boolValue)
	}
	return values, true
}

func azureDataCatalogCustomDataSourceValidateNamespace(namespace string) error {
	if len(namespace) > 255 {
		return fmt.Errorf("namespace must be at most 255 characters")
	}
	parts := strings.Split(namespace, ".")
	if len(parts) == 0 {
		return fmt.Errorf("namespace is invalid")
	}
	for _, part := range parts {
		if !azureDataCatalogCustomDataSourceNamespacePartPattern.MatchString(part) {
			return fmt.Errorf("namespace must contain one or more non-empty dot-separated parts; each part must start with a letter and contain only letters and numbers")
		}
	}
	return nil
}

func azureDataCatalogCustomDataSourceError(w http.ResponseWriter, status int, path, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "BadRequest"
	}
	if strings.TrimSpace(message) == "" {
		message = "request is invalid"
	}
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"target":  path,
		},
	})
}
