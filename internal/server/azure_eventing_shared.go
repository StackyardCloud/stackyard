package server

import (
	"time"
)

// Helpers shared by the Event Hubs and Event Grid providers.

// azurePrivateEndpointConnectionStateValidate checks a private endpoint
// connection PUT, which approves, rejects or disconnects the connection.
func azurePrivateEndpointConnectionStateValidate(payload map[string]any) string {
	state := azureARMObject(azureARMObject(payload["properties"])["privateLinkServiceConnectionState"])
	switch state["status"] {
	case "Pending", "Approved", "Rejected", "Disconnected":
	default:
		return "properties.privateLinkServiceConnectionState.status must be Pending, Approved, Rejected or Disconnected."
	}
	if len(azureARMString(state["description"])) > 1024 {
		return "properties.privateLinkServiceConnectionState.description must be at most 1024 characters."
	}
	return ""
}

// azureARMUpdateProperties stores a copy of the resource with props merged
// into its properties, for actions that change state, and returns the new
// record, or nil when the resource is gone.
func (s *Server) azureARMUpdateProperties(id string, props map[string]any) *azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureARMUpdatePropertiesLocked(id, props)
}

// azureARMUpdatePropertiesLocked is azureARMUpdateProperties for callers
// holding the store lock.
func (s *Server) azureARMUpdatePropertiesLocked(id string, props map[string]any) *azureARMResource {
	existing := s.azureARMResources[azureARMKey(id)]
	if existing == nil {
		return nil
	}
	updated := *existing
	updated.Properties = azureARMMerge(existing.Properties, props)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	return &updated
}

// azureARMLookup returns the stored resource with the ID, or nil.
func (s *Server) azureARMLookup(id string) *azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureARMResources[azureARMKey(id)]
}
