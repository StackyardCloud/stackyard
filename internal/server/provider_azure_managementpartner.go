package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.ManagementPartner Partner Admin Link: the partner ID linked to
// the signed-in account in its tenant. Each account links at most one
// partner ID, changed with PATCH, so GET /partners returns that one link
// rather than a list. The emulator has a single caller, so it keeps a
// single link.

const azureManagementPartnerPath = "/providers/Microsoft.ManagementPartner/partners"

type azureManagementPartnerLink struct {
	PartnerID string
	Version   int
	Created   time.Time
	Updated   time.Time
}

// ponytail: partner IDs are numeric Microsoft Cloud Partner Program IDs;
// the emulator can't check that one exists, only its shape.
var azureManagementPartnerIDPattern = regexp.MustCompile(`^[0-9]{1,10}$`)

// azureManagementPartnerObjectID is the emulator caller's object ID.
var azureManagementPartnerObjectID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("stackyard/caller")).String()

func (s *Server) handleAzureManagementPartnerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	rest, ok := strings.CutPrefix(strings.ToLower(path), "/azure"+strings.ToLower(azureManagementPartnerPath))
	if !ok || (rest != "" && !strings.HasPrefix(rest, "/")) {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(seg) > 4 {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	if len(seg) == 3 {
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		s.providerStorageMu.Lock()
		link := s.azureManagementPartner
		s.providerStorageMu.Unlock()
		s.respondAzureManagementPartner(w, link, "")
		return true
	}
	id := seg[3]
	if r.Method == http.MethodPut || r.Method == http.MethodPatch {
		if !azureManagementPartnerIDPattern.MatchString(id) {
			azureARMError(w, http.StatusBadRequest, "BadRequest", "The partner ID must be your numeric Microsoft Cloud Partner Program ID.")
			return true
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	link := s.azureManagementPartner
	now := time.Now().UTC()
	switch r.Method {
	case http.MethodGet:
		s.respondAzureManagementPartner(w, link, id)
	case http.MethodPut:
		switch {
		case link == nil:
			link = &azureManagementPartnerLink{PartnerID: id, Version: 1, Created: now, Updated: now}
			s.azureManagementPartner = link
		case link.PartnerID != id:
			azureARMError(w, http.StatusConflict, "Conflict", "Partner ID "+link.PartnerID+" is already linked to this account. Update the link with PATCH.")
			return true
		}
		s.respondAzureManagementPartner(w, link, id)
	case http.MethodPatch:
		if link == nil {
			azureARMNotFound(w, "ResourceNotFound", azureManagementPartnerPath+"/"+id, "")
			return true
		}
		updated := azureManagementPartnerLink{PartnerID: id, Version: link.Version + 1, Created: link.Created, Updated: now}
		s.azureManagementPartner = &updated
		s.respondAzureManagementPartner(w, &updated, id)
	case http.MethodDelete:
		if link == nil || link.PartnerID != id {
			azureARMNotFound(w, "ResourceNotFound", azureManagementPartnerPath+"/"+id, "")
			return true
		}
		s.azureManagementPartner = nil
		w.WriteHeader(http.StatusOK)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
	return true
}

// respondAzureManagementPartner writes link, or 404 when there is none or,
// with id set, it links another partner ID.
func (s *Server) respondAzureManagementPartner(w http.ResponseWriter, link *azureManagementPartnerLink, id string) {
	if link == nil || (id != "" && link.PartnerID != id) {
		azureARMNotFound(w, "ResourceNotFound", azureManagementPartnerPath+"/"+id, "")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id": azureManagementPartnerPath + "/" + link.PartnerID, "name": link.PartnerID,
		"type": "Microsoft.ManagementPartner/partner", "etag": link.Version,
		"properties": map[string]any{
			"partnerId": link.PartnerID, "partnerName": "Partner " + link.PartnerID,
			"tenantId": azureDefaultTenantID, "objectId": azureManagementPartnerObjectID,
			"version": link.Version, "state": "Active",
			"createdTime": link.Created.Format(time.RFC3339), "updatedTime": link.Updated.Format(time.RFC3339),
		},
	})
}
