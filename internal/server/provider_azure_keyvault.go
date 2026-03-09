package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type azureSecretVersion struct {
	Version   string
	Value     string
	CreatedAt time.Time
}

func (s *Server) handleAzureKeyVaultRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/keyvault/") {
		return false
	}
	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/keyvault/"))
	if len(segments) < 3 || segments[1] != "secrets" {
		respondAzureImplemented(w, path)
		return true
	}

	vault := segments[0]
	secretName := segments[2]

	if len(segments) == 3 {
		switch r.Method {
		case http.MethodPut:
			s.handleAzureKeyVaultSetSecret(w, r, vault, secretName)
		case http.MethodGet:
			s.handleAzureKeyVaultGetLatestSecret(w, vault, secretName)
		default:
			respondAzureImplemented(w, path)
		}
		return true
	}

	if len(segments) == 4 && segments[3] == "versions" && r.Method == http.MethodGet {
		s.handleAzureKeyVaultListSecretVersions(w, vault, secretName)
		return true
	}
	if len(segments) == 5 && segments[3] == "versions" && r.Method == http.MethodGet {
		s.handleAzureKeyVaultGetSecretVersion(w, vault, secretName, segments[4])
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureKeyVaultSetSecret(w http.ResponseWriter, r *http.Request, vault, secretName string) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return
	}

	value := strings.TrimSpace(string(body))
	if strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "invalid JSON payload"})
			return
		}
		if parsed, ok := payload["value"].(string); ok {
			value = strings.TrimSpace(parsed)
		}
	}
	if value == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "secret value is required"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	vaults := s.azureKeyVaults[vault]
	if vaults == nil {
		vaults = map[string][]azureSecretVersion{}
		s.azureKeyVaults[vault] = vaults
	}
	versions := vaults[secretName]
	version := fmt.Sprintf("v%d", len(versions)+1)
	entry := azureSecretVersion{
		Version:   version,
		Value:     value,
		CreatedAt: time.Now().UTC(),
	}
	vaults[secretName] = append(versions, entry)
	respondJSON(w, http.StatusOK, azureKeyVaultSecretResponse(vault, secretName, entry))
}

func (s *Server) handleAzureKeyVaultGetLatestSecret(w http.ResponseWriter, vault, secretName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	vaults := s.azureKeyVaults[vault]
	versions := vaults[secretName]
	if len(versions) == 0 {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "SecretNotFound", "message": "secret not found"})
		return
	}
	respondJSON(w, http.StatusOK, azureKeyVaultSecretResponse(vault, secretName, versions[len(versions)-1]))
}

func (s *Server) handleAzureKeyVaultListSecretVersions(w http.ResponseWriter, vault, secretName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	vaults := s.azureKeyVaults[vault]
	versions := vaults[secretName]
	if len(versions) == 0 {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "SecretNotFound", "message": "secret not found"})
		return
	}

	items := make([]map[string]any, 0, len(versions))
	for _, version := range versions {
		items = append(items, azureKeyVaultSecretResponse(vault, secretName, version))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureKeyVaultGetSecretVersion(w http.ResponseWriter, vault, secretName, versionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	vaults := s.azureKeyVaults[vault]
	versions := vaults[secretName]
	if len(versions) == 0 {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "SecretNotFound", "message": "secret not found"})
		return
	}
	for _, version := range versions {
		if version.Version == versionID {
			respondJSON(w, http.StatusOK, azureKeyVaultSecretResponse(vault, secretName, version))
			return
		}
	}
	respondJSON(w, http.StatusNotFound, map[string]any{"error": "SecretVersionNotFound", "message": "secret version not found"})
}

func azureKeyVaultSecretResponse(vault, name string, version azureSecretVersion) map[string]any {
	return map[string]any{
		"id":      azureKeyVaultSecretID(vault, name, version.Version),
		"vault":   vault,
		"name":    name,
		"value":   version.Value,
		"version": version.Version,
		"attributes": map[string]any{
			"created": version.CreatedAt.Unix(),
		},
	}
}

func azureKeyVaultSecretID(vault, name, version string) string {
	return fmt.Sprintf("https://%s.vault.azure.net/secrets/%s/%s", strings.TrimSpace(vault), strings.TrimSpace(name), strings.TrimSpace(version))
}
