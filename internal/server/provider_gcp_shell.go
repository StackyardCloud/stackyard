package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	gcpShellOwnerRegex         = regexp.MustCompile(`^[A-Za-z0-9._%+\-@]+$`)
	gcpShellEnvironmentIDRegex = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)
)

func (s *Server) handleGCPShellRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shell(w, r) {
		return true
	}

	path := normalizeGCPShellPath(rawRequestPath(r))
	if !isGCPShellPath(path, hasGCPShellHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShellGetEnvironment(w, path) {
			return true
		}
		if handleGCPShellGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShellStartEnvironment(w, r, path) {
			return true
		}
		if handleGCPShellAuthorizeEnvironment(w, r, path) {
			return true
		}
		if handleGCPShellAddPublicKey(w, r, path) {
			return true
		}
		if handleGCPShellRemovePublicKey(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShellPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShellHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shell", "shell-apiv1", "shell_apiv1", "cloud-shell", "cloud_shell", "cloudshell", "gcp-cloud-shell":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shell-apiv1") || strings.Contains(ua, "cloud.google.com/go/shell")
}

func isGCPShellPath(path string, includeHint bool) bool {
	if isGCPShellGRPCPath(path, includeHint) {
		return true
	}
	if _, _, ok := parseGCPShellEnvironmentPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPShellEnvironmentActionPath(path); ok {
		return true
	}
	if operationID, ok := parseGCPShellOperationPath(path); ok {
		return includeHint || strings.HasPrefix(operationID, "shell-")
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/users/")
}

func isGCPShellGRPCPath(path string, includeHint bool) bool {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "/gcp/google.cloud.shell.v1.CloudShellService/") {
		return true
	}
	return includeHint && strings.HasPrefix(trimmed, "/gcp/google.longrunning.Operations/")
}

func handleGCPShellGetEnvironment(w http.ResponseWriter, path string) bool {
	owner, environmentID, ok := parseGCPShellEnvironmentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpShellEnvironment(owner, environmentID, []string{gcpShellDefaultPublicKey()}))
	return true
}

func handleGCPShellStartEnvironment(w http.ResponseWriter, r *http.Request, path string) bool {
	owner, environmentID, action, ok := parseGCPShellEnvironmentActionPath(path)
	if !ok || action != "start" {
		return false
	}

	body, ok := decodeGCPShellJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := gcpShellEnvironmentName(owner, environmentID)
	name := gcpShellStringField(body, "name")
	if name == "" {
		respondGCPShellInvalidArgument(w, path, "name is required")
		return true
	}
	if name != expectedName {
		respondGCPShellInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	for idx, key := range gcpShellStringSlice(body["publicKeys"]) {
		if !isGCPShellPublicKey(key) {
			respondGCPShellInvalidArgument(w, path, fmt.Sprintf("publicKeys[%d] is invalid", idx))
			return true
		}
	}

	opID := gcpShellOperationID("start", expectedName, "")
	op, _ := gcpShellOperationFromID(opID)
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPShellAuthorizeEnvironment(w http.ResponseWriter, r *http.Request, path string) bool {
	owner, environmentID, action, ok := parseGCPShellEnvironmentActionPath(path)
	if !ok || action != "authorize" {
		return false
	}

	body, ok := decodeGCPShellJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := gcpShellEnvironmentName(owner, environmentID)
	name := gcpShellStringField(body, "name")
	if name == "" {
		respondGCPShellInvalidArgument(w, path, "name is required")
		return true
	}
	if name != expectedName {
		respondGCPShellInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	accessToken := gcpShellStringField(body, "accessToken", "access_token")
	idToken := gcpShellStringField(body, "idToken", "id_token")
	if accessToken == "" && idToken == "" {
		respondGCPShellInvalidArgument(w, path, "accessToken or idToken is required")
		return true
	}
	if expireTime := gcpShellStringField(body, "expireTime", "expire_time"); expireTime != "" {
		if _, err := time.Parse(time.RFC3339Nano, expireTime); err != nil {
			respondGCPShellInvalidArgument(w, path, "expireTime must be RFC3339")
			return true
		}
	}

	opID := gcpShellOperationID("authorize", expectedName, "")
	op, _ := gcpShellOperationFromID(opID)
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPShellAddPublicKey(w http.ResponseWriter, r *http.Request, path string) bool {
	owner, environmentID, action, ok := parseGCPShellEnvironmentActionPath(path)
	if !ok || action != "addPublicKey" {
		return false
	}

	body, ok := decodeGCPShellJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedEnvironment := gcpShellEnvironmentName(owner, environmentID)
	environment := gcpShellStringField(body, "environment")
	if environment == "" {
		respondGCPShellInvalidArgument(w, path, "environment is required")
		return true
	}
	if environment != expectedEnvironment {
		respondGCPShellInvalidArgument(w, path, "environment must match requested resource")
		return true
	}
	key := gcpShellStringField(body, "key")
	if key == "" {
		respondGCPShellInvalidArgument(w, path, "key is required")
		return true
	}
	if !isGCPShellPublicKey(key) {
		respondGCPShellInvalidArgument(w, path, "key format is invalid")
		return true
	}
	if isGCPShellDuplicateKey(key) {
		respondGCPShellAlreadyExists(w, path, "public key already exists")
		return true
	}

	opID := gcpShellOperationID("add-public-key", expectedEnvironment, key)
	op, _ := gcpShellOperationFromID(opID)
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPShellRemovePublicKey(w http.ResponseWriter, r *http.Request, path string) bool {
	owner, environmentID, action, ok := parseGCPShellEnvironmentActionPath(path)
	if !ok || action != "removePublicKey" {
		return false
	}

	body, ok := decodeGCPShellJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedEnvironment := gcpShellEnvironmentName(owner, environmentID)
	environment := gcpShellStringField(body, "environment")
	if environment == "" {
		respondGCPShellInvalidArgument(w, path, "environment is required")
		return true
	}
	if environment != expectedEnvironment {
		respondGCPShellInvalidArgument(w, path, "environment must match requested resource")
		return true
	}
	key := gcpShellStringField(body, "key")
	if key == "" {
		respondGCPShellInvalidArgument(w, path, "key is required")
		return true
	}
	if !isGCPShellPublicKey(key) {
		respondGCPShellInvalidArgument(w, path, "key format is invalid")
		return true
	}
	if isGCPShellMissingKey(key) {
		respondGCPShellNotFound(w, path, "public key was not found")
		return true
	}

	opID := gcpShellOperationID("remove-public-key", expectedEnvironment, key)
	op, _ := gcpShellOperationFromID(opID)
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPShellGetOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPShellOperationPath(path)
	if !ok {
		return false
	}
	operation, found := gcpShellOperationFromID(operationID)
	if !found {
		respondGCPShellInvalidArgument(w, path, "operation name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func parseGCPShellEnvironmentPath(path string) (owner, environmentID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "environments" {
		return "", "", false
	}
	owner = strings.TrimSpace(parts[3])
	environmentID = strings.TrimSpace(parts[5])
	if !isGCPShellOwner(owner) || !isGCPShellEnvironmentID(environmentID) {
		return "", "", false
	}
	return owner, environmentID, true
}

func parseGCPShellEnvironmentActionPath(path string) (owner, environmentID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "environments" {
		return "", "", "", false
	}
	owner = strings.TrimSpace(parts[3])
	actionSegment := strings.TrimSpace(parts[5])
	environmentID, action, ok = strings.Cut(actionSegment, ":")
	if !ok {
		return "", "", "", false
	}
	environmentID = strings.TrimSpace(environmentID)
	action = strings.TrimSpace(action)
	if !isGCPShellOwner(owner) || !isGCPShellEnvironmentID(environmentID) || action == "" {
		return "", "", "", false
	}
	return owner, environmentID, action, true
}

func parseGCPShellOperationPath(path string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "operations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[3])
	if operationID == "" {
		return "", false
	}
	return operationID, true
}

func parseGCPShellEnvironmentName(name string) (owner, environmentID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "users" || parts[2] != "environments" {
		return "", "", false
	}
	owner = strings.TrimSpace(parts[1])
	environmentID = strings.TrimSpace(parts[3])
	if !isGCPShellOwner(owner) || !isGCPShellEnvironmentID(environmentID) {
		return "", "", false
	}
	return owner, environmentID, true
}

func decodeGCPShellJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		respondGCPShellInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShellInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPShellInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPShellInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func isGCPShellOwner(owner string) bool {
	return gcpShellOwnerRegex.MatchString(strings.TrimSpace(owner))
}

func isGCPShellEnvironmentID(environmentID string) bool {
	return gcpShellEnvironmentIDRegex.MatchString(strings.TrimSpace(environmentID))
}

func isGCPShellPublicKey(key string) bool {
	parts := strings.Fields(strings.TrimSpace(key))
	if len(parts) < 2 {
		return false
	}
	switch parts[0] {
	case "ssh-dss", "ssh-rsa", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
	default:
		return false
	}
	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		if _, rawErr := base64.RawStdEncoding.DecodeString(parts[1]); rawErr != nil {
			return false
		}
	}
	return true
}

func isGCPShellDuplicateKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == gcpShellDefaultPublicKey() {
		return true
	}
	return strings.Contains(strings.ToLower(key), "existing-key")
}

func isGCPShellMissingKey(key string) bool {
	key = strings.TrimSpace(key)
	if strings.Contains(strings.ToLower(key), "missing-key") {
		return true
	}
	return key == "ssh-rsa bWlzc2luZw== stackyard@example.com"
}

func gcpShellEnvironment(owner, environmentID string, publicKeys []string) map[string]any {
	if len(publicKeys) == 0 {
		publicKeys = []string{gcpShellDefaultPublicKey()}
	}
	return map[string]any{
		"name":        gcpShellEnvironmentName(owner, environmentID),
		"id":          environmentID,
		"dockerImage": "gcr.io/dev-con/cloud-devshell:latest",
		"state":       "RUNNING",
		"webHost":     "ssh.cloud.google.com",
		"sshUsername": "stackyard",
		"sshHost":     "34.1.2.3",
		"sshPort":     6000,
		"publicKeys":  publicKeys,
	}
}

func gcpShellEnvironmentName(owner, environmentID string) string {
	return fmt.Sprintf("users/%s/environments/%s", owner, environmentID)
}

func gcpShellOperationID(action, environmentName, detail string) string {
	id := "shell-" + strings.TrimSpace(action) + "-" + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(environmentName)))
	if strings.TrimSpace(detail) != "" {
		id += "." + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(detail)))
	}
	return id
}

func gcpShellOperationFromID(operationID string) (map[string]any, bool) {
	switch {
	case strings.HasPrefix(operationID, "shell-start-"):
		environmentName, _, ok := gcpShellDecodeOperationContext(strings.TrimPrefix(operationID, "shell-start-"))
		if !ok {
			return nil, false
		}
		owner, environmentID, ok := parseGCPShellEnvironmentName(environmentName)
		if !ok {
			return nil, false
		}
		return map[string]any{
			"name": "operations/" + operationID,
			"done": true,
			"metadata": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.StartEnvironmentMetadata",
				"state": "FINISHED",
			},
			"response": map[string]any{
				"@type":       "type.googleapis.com/google.cloud.shell.v1.StartEnvironmentResponse",
				"environment": gcpShellEnvironment(owner, environmentID, []string{gcpShellDefaultPublicKey()}),
			},
		}, true
	case strings.HasPrefix(operationID, "shell-authorize-"):
		environmentName, _, ok := gcpShellDecodeOperationContext(strings.TrimPrefix(operationID, "shell-authorize-"))
		if !ok {
			return nil, false
		}
		if _, _, ok := parseGCPShellEnvironmentName(environmentName); !ok {
			return nil, false
		}
		return map[string]any{
			"name": "operations/" + operationID,
			"done": true,
			"metadata": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.AuthorizeEnvironmentMetadata",
			},
			"response": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.AuthorizeEnvironmentResponse",
			},
		}, true
	case strings.HasPrefix(operationID, "shell-add-public-key-"):
		environmentName, detail, ok := gcpShellDecodeOperationContext(strings.TrimPrefix(operationID, "shell-add-public-key-"))
		if !ok {
			return nil, false
		}
		if _, _, ok := parseGCPShellEnvironmentName(environmentName); !ok {
			return nil, false
		}
		key := strings.TrimSpace(detail)
		if key == "" {
			key = gcpShellDefaultPublicKey()
		}
		return map[string]any{
			"name": "operations/" + operationID,
			"done": true,
			"metadata": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.AddPublicKeyMetadata",
			},
			"response": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.AddPublicKeyResponse",
				"key":   key,
			},
		}, true
	case strings.HasPrefix(operationID, "shell-remove-public-key-"):
		environmentName, _, ok := gcpShellDecodeOperationContext(strings.TrimPrefix(operationID, "shell-remove-public-key-"))
		if !ok {
			return nil, false
		}
		if _, _, ok := parseGCPShellEnvironmentName(environmentName); !ok {
			return nil, false
		}
		return map[string]any{
			"name": "operations/" + operationID,
			"done": true,
			"metadata": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.RemovePublicKeyMetadata",
			},
			"response": map[string]any{
				"@type": "type.googleapis.com/google.cloud.shell.v1.RemovePublicKeyResponse",
			},
		}, true
	default:
		return nil, false
	}
}

func gcpShellDecodeOperationContext(encoded string) (environmentName, detail string, ok bool) {
	environmentToken := encoded
	detailToken := ""
	if split := strings.SplitN(encoded, ".", 2); len(split) == 2 {
		environmentToken = strings.TrimSpace(split[0])
		detailToken = strings.TrimSpace(split[1])
	}
	if environmentToken == "" {
		return "", "", false
	}
	environmentBytes, err := base64.RawURLEncoding.DecodeString(environmentToken)
	if err != nil {
		return "", "", false
	}
	environmentName = strings.TrimSpace(string(environmentBytes))
	if environmentName == "" {
		return "", "", false
	}
	if detailToken == "" {
		return environmentName, "", true
	}
	detailBytes, err := base64.RawURLEncoding.DecodeString(detailToken)
	if err != nil {
		return "", "", false
	}
	return environmentName, strings.TrimSpace(string(detailBytes)), true
}

func gcpShellDefaultPublicKey() string {
	return "ssh-rsa c3RhY2t5YXJkLWRlZmF1bHQtaw== stackyard@example.com"
}

func gcpShellStringField(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := body[key].(string); ok {
			if trimmed := strings.TrimSpace(val); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func gcpShellStringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func respondGCPShellInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPShellError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPShellFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPShellError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPShellNotFound(w http.ResponseWriter, path, message string) {
	respondGCPShellError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPShellAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPShellError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPShellError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shell(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPShellPath(rawRequestPath(r))
	if !isGCPShellPath(path, true) {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPShellInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpShellEnvironment("me", "default", []string{gcpShellDefaultPublicKey()})
	payload["service"] = "shell"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}
