package server

import (
	"net/http"
	"time"
)

// Microsoft.CustomerLockbox tenant opt-in and access requests. Microsoft
// engineers never ask for access in the emulator, so there are no requests
// to list or approve; only the tenant's opt-in is kept.
func init() {
	const ns = "Microsoft.CustomerLockbox"
	registerAzureARMProviderAction(ns, "enableLockbox", func(s *Server, _ map[string]any) (int, any) {
		s.setAzureLockboxOptIn(true)
		return http.StatusOK, nil
	})
	registerAzureARMProviderAction(ns, "disableLockbox", func(s *Server, _ map[string]any) (int, any) {
		s.setAzureLockboxOptIn(false)
		return http.StatusOK, nil
	})
	registerAzureARMView(ns, "tenantOptedIn", func(s *Server, scope string) []map[string]any {
		if scope != "" {
			return nil
		}
		s.providerStorageMu.Lock()
		rec := s.azureARMResources[azureARMKey(azureLockboxOptInID)]
		s.providerStorageMu.Unlock()
		optedIn := rec != nil && rec.Properties["isOptedIn"] == true
		return []map[string]any{{"id": azureLockboxOptInID, "isOptedIn": optedIn}}
	})
	registerAzureARMView(ns, "requests", func(*Server, string) []map[string]any { return nil })
}

// azureLockboxOptInID is where the default tenant's opt-in is stored.
const azureLockboxOptInID = "/providers/Microsoft.CustomerLockbox/tenantOptedIn/" + azureDefaultTenantID

func (s *Server) setAzureLockboxOptIn(optedIn bool) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	created := now
	if old := s.azureARMResources[azureARMKey(azureLockboxOptInID)]; old != nil {
		created = old.CreatedAt
	}
	s.azureARMResources[azureARMKey(azureLockboxOptInID)] = &azureARMResource{
		ID: azureLockboxOptInID, Name: azureDefaultTenantID, Type: "Microsoft.CustomerLockbox/tenantOptedIn",
		Properties: map[string]any{"isOptedIn": optedIn}, CreatedAt: created, UpdatedAt: now,
	}
}
