package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureWebPubSubDataPlaneAPIVersion = "2024-12-01"
const azureWebPubSubDataPlanePrefix = "/azure/webpubsub-data-plane/2024-12-01"

var azureWebPubSubDataPlaneHubPattern = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_`,.\\[\\]]{0,127}$")
var azureWebPubSubDataPlaneStartsWithFilter = regexp.MustCompile(`(?i)^startswith\((userId|connectionId)\s*,\s*'([^']*)'\)$`)
var azureWebPubSubDataPlaneEqualsFilter = regexp.MustCompile(`(?i)^(userId|connectionId)\s+eq\s+'([^']*)'$`)

type azureWebPubSubDataPlaneConnection struct {
	Hub          string
	ConnectionID string
	UserID       string
	Groups       map[string]struct{}
	Permissions  map[string]map[string]struct{}
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type azureWebPubSubDataPlaneUser struct {
	Hub       string
	UserID    string
	Groups    map[string]struct{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Server) handleAzureWebPubSubDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureWebPubSubDataPlanePrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureWebPubSubDataPlaneAPIVersion {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2024-12-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureWebPubSubDataPlanePrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
	if len(segments) == 2 && strings.EqualFold(segments[0], "api") && strings.EqualFold(segments[1], "health") {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		w.WriteHeader(http.StatusOK)
		return true
	}

	if len(segments) < 4 || !strings.EqualFold(segments[0], "api") || !strings.EqualFold(segments[1], "hubs") {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
	hub := strings.TrimSpace(segments[2])
	if !azureWebPubSubDataPlaneValidHub(hub) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "hub is invalid")
		return true
	}

	tail := segments[3:]
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case ":generatetoken":
		if len(tail) != 1 {
			azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureWebPubSubDataPlaneGenerateToken(w, r, hub)
		return true
	case ":addtogroups":
		if len(tail) != 1 {
			azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureWebPubSubDataPlaneAddConnectionsToGroups(w, r, hub)
		return true
	case ":removefromgroups":
		if len(tail) != 1 {
			azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureWebPubSubDataPlaneRemoveConnectionsFromGroups(w, r, hub)
		return true
	case ":send":
		if len(tail) != 1 {
			azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureWebPubSubDataPlaneSend(w, r)
		return true
	case ":closeconnections":
		if len(tail) != 1 {
			azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureWebPubSubDataPlaneCloseAllConnections(w, hub)
		return true
	case "connections":
		s.handleAzureWebPubSubDataPlaneConnections(w, r, hub, tail[1:])
		return true
	case "groups":
		s.handleAzureWebPubSubDataPlaneGroups(w, r, hub, tail[1:])
		return true
	case "users":
		s.handleAzureWebPubSubDataPlaneUsers(w, r, hub, tail[1:])
		return true
	case "permissions":
		s.handleAzureWebPubSubDataPlanePermissions(w, r, hub, tail[1:])
		return true
	default:
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureWebPubSubDataPlaneConnections(w http.ResponseWriter, r *http.Request, hub string, tail []string) {
	if len(tail) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	connectionID := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidEntityID(connectionID) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "connectionId is invalid")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodHead:
			s.handleAzureWebPubSubDataPlaneHeadConnection(w, hub, connectionID)
		case http.MethodDelete:
			s.handleAzureWebPubSubDataPlaneDeleteConnection(w, hub, connectionID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 2 && strings.EqualFold(tail[1], "groups") {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneRemoveConnectionFromAllGroups(w, hub, connectionID)
		return
	}

	if len(tail) == 2 && strings.EqualFold(tail[1], ":send") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneSendToConnection(w, r, hub, connectionID)
		return
	}

	azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureWebPubSubDataPlaneGroups(w http.ResponseWriter, r *http.Request, hub string, tail []string) {
	if len(tail) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	group := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidGroup(group) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "group is invalid")
		return
	}

	if len(tail) == 1 {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneHeadGroup(w, hub, group)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "connections":
		s.handleAzureWebPubSubDataPlaneGroupConnections(w, r, hub, group, tail[2:])
		return
	case ":send":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureWebPubSubDataPlaneSendToGroup(w, r, hub, group)
		return
	case ":closeconnections":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureWebPubSubDataPlaneCloseGroupConnections(w, hub, group)
		return
	default:
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureWebPubSubDataPlaneGroupConnections(w http.ResponseWriter, r *http.Request, hub, group string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneListGroupConnections(w, r, hub, group)
		return
	}
	connectionID := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidEntityID(connectionID) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "connectionId is invalid")
		return
	}
	if len(tail) != 1 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleAzureWebPubSubDataPlaneAddConnectionToGroup(w, r, hub, group, connectionID)
	case http.MethodDelete:
		s.handleAzureWebPubSubDataPlaneRemoveConnectionFromGroup(w, hub, group, connectionID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureWebPubSubDataPlaneUsers(w http.ResponseWriter, r *http.Request, hub string, tail []string) {
	if len(tail) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	userID := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidEntityID(userID) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "userId is invalid")
		return
	}

	if len(tail) == 1 {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneHeadUser(w, hub, userID)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "groups":
		s.handleAzureWebPubSubDataPlaneUserGroups(w, r, hub, userID, tail[2:])
	case ":send":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureWebPubSubDataPlaneSendToUser(w, r, hub, userID)
	case ":closeconnections":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureWebPubSubDataPlaneCloseUserConnections(w, hub, userID)
	default:
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureWebPubSubDataPlaneUserGroups(w http.ResponseWriter, r *http.Request, hub, userID string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubDataPlaneRemoveUserFromAllGroups(w, hub, userID)
		return
	}
	group := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidGroup(group) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "group is invalid")
		return
	}
	if len(tail) != 1 {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.handleAzureWebPubSubDataPlaneAddUserToGroup(w, hub, userID, group)
	case http.MethodDelete:
		s.handleAzureWebPubSubDataPlaneRemoveUserFromGroup(w, hub, userID, group)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureWebPubSubDataPlanePermissions(w http.ResponseWriter, r *http.Request, hub string, tail []string) {
	if len(tail) != 3 || !strings.EqualFold(tail[1], "connections") {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	permission := strings.TrimSpace(tail[0])
	if !azureWebPubSubDataPlaneValidPermission(permission) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "permission is invalid")
		return
	}
	connectionID := strings.TrimSpace(tail[2])
	if !azureWebPubSubDataPlaneValidEntityID(connectionID) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "connectionId is invalid")
		return
	}
	targetName := strings.TrimSpace(r.URL.Query().Get("targetName"))
	if targetName == "" {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "targetName is required")
		return
	}
	if !azureWebPubSubDataPlaneValidGroup(targetName) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "targetName is invalid")
		return
	}

	switch r.Method {
	case http.MethodHead:
		s.handleAzureWebPubSubDataPlaneCheckPermission(w, hub, permission, connectionID, targetName)
	case http.MethodPut:
		s.handleAzureWebPubSubDataPlaneGrantPermission(w, hub, permission, connectionID, targetName)
	case http.MethodDelete:
		s.handleAzureWebPubSubDataPlaneRevokePermission(w, hub, permission, connectionID, targetName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureWebPubSubDataPlaneGenerateToken(w http.ResponseWriter, r *http.Request, hub string) {
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if userID == "" {
		if payload, ok := azureWebPubSubDataPlaneRequestMap(w, r); ok {
			userID = strings.TrimSpace(azureWebPubSubDataPlaneString(payload["userId"]))
		} else {
			return
		}
	}
	roles := azureWebPubSubDataPlaneStringList(append(r.URL.Query()["roles"], r.URL.Query()["role"]...))
	groups := azureWebPubSubDataPlaneStringList(append(r.URL.Query()["groups"], r.URL.Query()["group"]...))
	clientProtocol := strings.TrimSpace(r.URL.Query().Get("clientProtocol"))
	if clientProtocol == "" {
		clientProtocol = "Default"
	}
	minutesToExpire := 60
	if raw := strings.TrimSpace(r.URL.Query().Get("minutesToExpire")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "minutesToExpire must be a positive integer")
			return
		}
		minutesToExpire = value
	}
	token := azureWebPubSubDataPlaneStableToken(hub, userID, clientProtocol, strings.Join(roles, ","), strings.Join(groups, ","), strconv.Itoa(minutesToExpire))
	respondJSON(w, http.StatusOK, map[string]any{"token": token})
}

func (s *Server) handleAzureWebPubSubDataPlaneHeadConnection(w http.ResponseWriter, hub, connectionID string) {
	s.providerStorageMu.Lock()
	exists := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)] != nil
	s.providerStorageMu.Unlock()
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneDeleteConnection(w http.ResponseWriter, hub, connectionID string) {
	s.providerStorageMu.Lock()
	delete(s.azureWebPubSubDataPlaneConnections, azureWebPubSubDataPlaneConnectionKey(hub, connectionID))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneRemoveConnectionFromAllGroups(w http.ResponseWriter, hub, connectionID string) {
	s.providerStorageMu.Lock()
	connection := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)]
	if connection != nil {
		connection.Groups = map[string]struct{}{}
		connection.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneSendToConnection(w http.ResponseWriter, r *http.Request, hub, connectionID string) {
	if !s.azureWebPubSubDataPlaneConnectionExists(hub, connectionID) {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "ConnectionNotFound", "connection not found")
		return
	}
	s.handleAzureWebPubSubDataPlaneSend(w, r)
}

func (s *Server) handleAzureWebPubSubDataPlaneHeadGroup(w http.ResponseWriter, hub, group string) {
	if !s.azureWebPubSubDataPlaneGroupExists(hub, group) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneListGroupConnections(w http.ResponseWriter, r *http.Request, hub, group string) {
	if !s.azureWebPubSubDataPlaneGroupExists(hub, group) {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "GroupNotFound", "group not found")
		return
	}
	maxPageSize := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("maxpagesize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "maxpagesize must be between 1 and 200")
			return
		}
		maxPageSize = value
	}
	top := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("top")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "top must be a positive integer")
			return
		}
		top = value
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("continuationToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "continuationToken is invalid")
			return
		}
		offset = value
	}

	s.providerStorageMu.Lock()
	items := s.azureWebPubSubDataPlaneGroupMembersLocked(hub, group)
	s.providerStorageMu.Unlock()

	if top > 0 && top < len(items) {
		items = items[:top]
	}
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + maxPageSize
	if end > len(items) {
		end = len(items)
	}

	response := map[string]any{
		"value": items[offset:end],
	}
	if end < len(items) {
		nextURL := azureWebPubSubDataPlanePrefix + "/api/hubs/" + url.PathEscape(hub) + "/groups/" + url.PathEscape(group) + "/connections?api-version=" + azureWebPubSubDataPlaneAPIVersion
		nextURL += "&top=" + strconv.Itoa(len(items))
		nextURL += "&maxpagesize=" + strconv.Itoa(maxPageSize)
		nextURL += "&continuationToken=" + strconv.Itoa(end)
		response["nextLink"] = nextURL
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleAzureWebPubSubDataPlaneAddConnectionToGroup(w http.ResponseWriter, r *http.Request, hub, group, connectionID string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	connection := s.azureWebPubSubDataPlaneEnsureConnectionLocked(hub, connectionID, now)
	connection.Groups[group] = struct{}{}
	connection.UpdatedAt = now
	if userID := strings.TrimSpace(r.URL.Query().Get("userId")); userID != "" {
		connection.UserID = userID
		_ = s.azureWebPubSubDataPlaneEnsureUserLocked(hub, userID, now)
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneRemoveConnectionFromGroup(w http.ResponseWriter, hub, group, connectionID string) {
	s.providerStorageMu.Lock()
	if connection := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)]; connection != nil {
		delete(connection.Groups, group)
		connection.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneHeadUser(w http.ResponseWriter, hub, userID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(hub, userID)] != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub && strings.EqualFold(connection.UserID, userID) {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) handleAzureWebPubSubDataPlaneAddUserToGroup(w http.ResponseWriter, hub, userID, group string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	user := s.azureWebPubSubDataPlaneEnsureUserLocked(hub, userID, now)
	user.Groups[group] = struct{}{}
	user.UpdatedAt = now
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneRemoveUserFromGroup(w http.ResponseWriter, hub, userID, group string) {
	s.providerStorageMu.Lock()
	if user := s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(hub, userID)]; user != nil {
		delete(user.Groups, group)
		user.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneRemoveUserFromAllGroups(w http.ResponseWriter, hub, userID string) {
	s.providerStorageMu.Lock()
	if user := s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(hub, userID)]; user != nil {
		user.Groups = map[string]struct{}{}
		user.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneSendToUser(w http.ResponseWriter, r *http.Request, hub, userID string) {
	if !s.azureWebPubSubDataPlaneUserExists(hub, userID) {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "UserNotFound", "user not found")
		return
	}
	s.handleAzureWebPubSubDataPlaneSend(w, r)
}

func (s *Server) handleAzureWebPubSubDataPlaneCloseUserConnections(w http.ResponseWriter, hub, userID string) {
	s.providerStorageMu.Lock()
	for key, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub && strings.EqualFold(connection.UserID, userID) {
			delete(s.azureWebPubSubDataPlaneConnections, key)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneCheckPermission(w http.ResponseWriter, hub, permission, connectionID, targetName string) {
	s.providerStorageMu.Lock()
	connection := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)]
	s.providerStorageMu.Unlock()
	if connection == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	targets := connection.Permissions[strings.ToLower(permission)]
	if targets == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if _, ok := targets[targetName]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneGrantPermission(w http.ResponseWriter, hub, permission, connectionID, targetName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	connection := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)]
	if connection == nil {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "ConnectionNotFound", "connection not found")
		return
	}
	key := strings.ToLower(permission)
	if connection.Permissions[key] == nil {
		connection.Permissions[key] = map[string]struct{}{}
	}
	connection.Permissions[key][targetName] = struct{}{}
	connection.UpdatedAt = time.Now().UTC()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureWebPubSubDataPlaneRevokePermission(w http.ResponseWriter, hub, permission, connectionID, targetName string) {
	s.providerStorageMu.Lock()
	if connection := s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)]; connection != nil {
		if targets := connection.Permissions[strings.ToLower(permission)]; targets != nil {
			delete(targets, targetName)
		}
		connection.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneSend(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneSendToGroup(w http.ResponseWriter, r *http.Request, hub, group string) {
	if !s.azureWebPubSubDataPlaneGroupExists(hub, group) {
		azureWebPubSubDataPlaneError(w, http.StatusNotFound, "GroupNotFound", "group not found")
		return
	}
	s.handleAzureWebPubSubDataPlaneSend(w, r)
}

func (s *Server) handleAzureWebPubSubDataPlaneCloseAllConnections(w http.ResponseWriter, hub string) {
	s.providerStorageMu.Lock()
	for key, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub {
			delete(s.azureWebPubSubDataPlaneConnections, key)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneCloseGroupConnections(w http.ResponseWriter, hub, group string) {
	s.providerStorageMu.Lock()
	for key, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub && s.azureWebPubSubDataPlaneConnectionInGroupLocked(connection, group) {
			delete(s.azureWebPubSubDataPlaneConnections, key)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneAddConnectionsToGroups(w http.ResponseWriter, r *http.Request, hub string) {
	payload, ok := azureWebPubSubDataPlaneRequestMap(w, r)
	if !ok {
		return
	}
	groups := azureWebPubSubDataPlaneStringListFromAny(payload["groups"])
	if len(groups) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "groups is required")
		return
	}
	for _, group := range groups {
		if !azureWebPubSubDataPlaneValidGroup(group) {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "groups contains an invalid group")
			return
		}
	}
	filter := strings.TrimSpace(azureWebPubSubDataPlaneString(payload["filter"]))
	if filter == "" {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "filter is required")
		return
	}
	if !azureWebPubSubDataPlaneValidBulkFilter(filter) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "filter is invalid")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub != hub || !azureWebPubSubDataPlaneMatchesFilter(connection, filter) {
			continue
		}
		for _, group := range groups {
			connection.Groups[group] = struct{}{}
		}
		connection.UpdatedAt = time.Now().UTC()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureWebPubSubDataPlaneRemoveConnectionsFromGroups(w http.ResponseWriter, r *http.Request, hub string) {
	payload, ok := azureWebPubSubDataPlaneRequestMap(w, r)
	if !ok {
		return
	}
	groups := azureWebPubSubDataPlaneStringListFromAny(payload["groups"])
	if len(groups) == 0 {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "groups is required")
		return
	}
	for _, group := range groups {
		if !azureWebPubSubDataPlaneValidGroup(group) {
			azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "groups contains an invalid group")
			return
		}
	}
	filter := strings.TrimSpace(azureWebPubSubDataPlaneString(payload["filter"]))
	if filter == "" {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "filter is required")
		return
	}
	if !azureWebPubSubDataPlaneValidBulkFilter(filter) {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "filter is invalid")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub != hub || !azureWebPubSubDataPlaneMatchesFilter(connection, filter) {
			continue
		}
		for _, group := range groups {
			delete(connection.Groups, group)
		}
		connection.UpdatedAt = time.Now().UTC()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureWebPubSubDataPlaneConnectionExists(hub, connectionID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureWebPubSubDataPlaneConnections[azureWebPubSubDataPlaneConnectionKey(hub, connectionID)] != nil
}

func (s *Server) azureWebPubSubDataPlaneUserExists(hub, userID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(hub, userID)] != nil {
		return true
	}
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub && strings.EqualFold(connection.UserID, userID) {
			return true
		}
	}
	return false
}

func (s *Server) azureWebPubSubDataPlaneGroupExists(hub, group string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub == hub {
			if _, ok := connection.Groups[group]; ok {
				return true
			}
			if connection.UserID != "" {
				if user := s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(hub, connection.UserID)]; user != nil {
					if _, ok := user.Groups[group]; ok {
						return true
					}
				}
			}
		}
	}
	for _, user := range s.azureWebPubSubDataPlaneUsers {
		if user.Hub == hub {
			if _, ok := user.Groups[group]; ok {
				return true
			}
		}
	}
	return false
}

func (s *Server) azureWebPubSubDataPlaneEnsureConnectionLocked(hub, connectionID string, now time.Time) *azureWebPubSubDataPlaneConnection {
	key := azureWebPubSubDataPlaneConnectionKey(hub, connectionID)
	if existing := s.azureWebPubSubDataPlaneConnections[key]; existing != nil {
		return existing
	}
	connection := &azureWebPubSubDataPlaneConnection{
		Hub:          hub,
		ConnectionID: connectionID,
		Groups:       map[string]struct{}{},
		Permissions:  map[string]map[string]struct{}{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.azureWebPubSubDataPlaneConnections[key] = connection
	return connection
}

func (s *Server) azureWebPubSubDataPlaneEnsureUserLocked(hub, userID string, now time.Time) *azureWebPubSubDataPlaneUser {
	key := azureWebPubSubDataPlaneUserKey(hub, userID)
	if existing := s.azureWebPubSubDataPlaneUsers[key]; existing != nil {
		return existing
	}
	user := &azureWebPubSubDataPlaneUser{
		Hub:       hub,
		UserID:    userID,
		Groups:    map[string]struct{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.azureWebPubSubDataPlaneUsers[key] = user
	return user
}

func (s *Server) azureWebPubSubDataPlaneGroupMembersLocked(hub, group string) []any {
	members := make([]map[string]any, 0)
	for _, connection := range s.azureWebPubSubDataPlaneConnections {
		if connection.Hub != hub || !s.azureWebPubSubDataPlaneConnectionInGroupLocked(connection, group) {
			continue
		}
		members = append(members, map[string]any{
			"connectionId": connection.ConnectionID,
			"userId":       connection.UserID,
		})
	}
	sort.Slice(members, func(i, j int) bool {
		left := azureWebPubSubDataPlaneString(members[i]["connectionId"])
		right := azureWebPubSubDataPlaneString(members[j]["connectionId"])
		return left < right
	})
	out := make([]any, 0, len(members))
	for _, member := range members {
		out = append(out, member)
	}
	return out
}

func (s *Server) azureWebPubSubDataPlaneConnectionInGroupLocked(connection *azureWebPubSubDataPlaneConnection, group string) bool {
	if _, ok := connection.Groups[group]; ok {
		return true
	}
	if connection.UserID == "" {
		return false
	}
	user := s.azureWebPubSubDataPlaneUsers[azureWebPubSubDataPlaneUserKey(connection.Hub, connection.UserID)]
	if user == nil {
		return false
	}
	_, ok := user.Groups[group]
	return ok
}

func azureWebPubSubDataPlaneConnectionKey(hub, connectionID string) string {
	return strings.ToLower(strings.TrimSpace(hub)) + "|" + strings.TrimSpace(connectionID)
}

func azureWebPubSubDataPlaneUserKey(hub, userID string) string {
	return strings.ToLower(strings.TrimSpace(hub)) + "|" + strings.TrimSpace(userID)
}

func azureWebPubSubDataPlaneValidHub(hub string) bool {
	return azureWebPubSubDataPlaneHubPattern.MatchString(strings.TrimSpace(hub))
}

func azureWebPubSubDataPlaneValidGroup(group string) bool {
	group = strings.TrimSpace(group)
	return group != "" && len(group) <= 1024
}

func azureWebPubSubDataPlaneValidEntityID(value string) bool {
	return strings.TrimSpace(value) != ""
}

func azureWebPubSubDataPlaneValidPermission(permission string) bool {
	switch strings.TrimSpace(permission) {
	case "joinLeaveGroup", "sendToGroup":
		return true
	default:
		return false
	}
}

func azureWebPubSubDataPlaneMatchesFilter(connection *azureWebPubSubDataPlaneConnection, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	if matches := azureWebPubSubDataPlaneStartsWithFilter.FindStringSubmatch(filter); len(matches) == 3 {
		field := strings.ToLower(matches[1])
		prefix := matches[2]
		if field == "userid" {
			return strings.HasPrefix(connection.UserID, prefix)
		}
		return strings.HasPrefix(connection.ConnectionID, prefix)
	}
	if matches := azureWebPubSubDataPlaneEqualsFilter.FindStringSubmatch(filter); len(matches) == 3 {
		field := strings.ToLower(matches[1])
		value := matches[2]
		if field == "userid" {
			return connection.UserID == value
		}
		return connection.ConnectionID == value
	}
	return false
}

func azureWebPubSubDataPlaneValidBulkFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	return azureWebPubSubDataPlaneStartsWithFilter.MatchString(filter) || azureWebPubSubDataPlaneEqualsFilter.MatchString(filter)
}

func azureWebPubSubDataPlaneStableToken(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	digest := hex.EncodeToString(hash[:])
	return fmt.Sprintf("stackyard.%s.%s", digest[:16], digest[16:32])
}

func azureWebPubSubDataPlaneRequestMap(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureWebPubSubDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func azureWebPubSubDataPlaneString(value any) string {
	typed, _ := value.(string)
	return typed
}

func azureWebPubSubDataPlaneStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func azureWebPubSubDataPlaneStringListFromAny(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(azureWebPubSubDataPlaneString(item)); text != "" {
				out = append(out, text)
			}
		}
		return azureWebPubSubDataPlaneStringList(out)
	case []string:
		return azureWebPubSubDataPlaneStringList(typed)
	case string:
		return azureWebPubSubDataPlaneStringList([]string{typed})
	default:
		return nil
	}
}

func azureWebPubSubDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-ms-error-code", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
