package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	wickrDefaultRegion    = "us-east-1"
	wickrDefaultAccountID = "123456789012"
)

type wickrStore struct {
	mu sync.Mutex

	nextNetwork int64
	nextUser    int64
	nextBot     int64
	nextGroup   int64

	networks              map[string]map[string]any
	networkSettings       map[string]map[string]any
	users                 map[string]map[string]map[string]any
	userDevices           map[string]map[string][]any
	bots                  map[string]map[string]map[string]any
	dataRetentionBots     map[string]map[string]any
	securityGroups        map[string]map[string]map[string]any
	securityGroupUsers    map[string]map[string][]string
	guestUsers            map[string]map[string]map[string]any
	blockedGuestUsers     map[string]map[string]map[string]any
	oidcConfigByNetworkID map[string]map[string]any
}

func newWickrStore() *wickrStore {
	now := time.Now().UTC().Format(time.RFC3339)

	s := &wickrStore{
		nextNetwork:           2,
		nextUser:              2,
		nextBot:               2,
		nextGroup:             2,
		networks:              map[string]map[string]any{},
		networkSettings:       map[string]map[string]any{},
		users:                 map[string]map[string]map[string]any{},
		userDevices:           map[string]map[string][]any{},
		bots:                  map[string]map[string]map[string]any{},
		dataRetentionBots:     map[string]map[string]any{},
		securityGroups:        map[string]map[string]map[string]any{},
		securityGroupUsers:    map[string]map[string][]string{},
		guestUsers:            map[string]map[string]map[string]any{},
		blockedGuestUsers:     map[string]map[string]map[string]any{},
		oidcConfigByNetworkID: map[string]map[string]any{},
	}

	networkID := "n-000001"
	userID := "u-000001"
	botID := "b-000001"
	groupID := "g-000001"
	usernameHash := "uh-000001"

	s.ensureNetworkLocked(networkID, now)
	s.ensureNetworkSettingsLocked(networkID, now)
	s.ensureUserLocked(networkID, userID, now)
	s.ensureUserDeviceLocked(networkID, userID, now)
	s.ensureBotLocked(networkID, botID, now)
	s.ensureSecurityGroupLocked(networkID, groupID, now)
	s.ensureGuestUserLocked(networkID, usernameHash, now)
	s.ensureOIDCConfigLocked(networkID, now)

	s.securityGroupUsers[networkID][groupID] = []string{userID}

	return s
}

func (s *wickrStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	networkID := wickrLookupString(payload, pathParams, query, "networkId", "NetworkId")
	if networkID == "" {
		networkID = "n-000001"
	}
	userID := wickrLookupString(payload, pathParams, query, "userId", "UserId", "id", "Id")
	if userID == "" {
		userID = "u-000001"
	}
	botID := wickrLookupString(payload, pathParams, query, "botId", "BotId", "id", "Id")
	if botID == "" {
		botID = "b-000001"
	}
	groupID := wickrLookupString(payload, pathParams, query, "groupId", "GroupId", "id", "Id")
	if groupID == "" {
		groupID = "g-000001"
	}
	usernameHash := wickrLookupString(payload, pathParams, query, "usernameHash", "UsernameHash")
	if usernameHash == "" {
		usernameHash = "uh-000001"
	}

	s.ensureNetworkLocked(networkID, now)
	s.ensureNetworkSettingsLocked(networkID, now)
	s.ensureUserLocked(networkID, userID, now)
	s.ensureUserDeviceLocked(networkID, userID, now)
	s.ensureBotLocked(networkID, botID, now)
	s.ensureSecurityGroupLocked(networkID, groupID, now)
	s.ensureGuestUserLocked(networkID, usernameHash, now)
	s.ensureOIDCConfigLocked(networkID, now)

	switch action {
	case "CreateNetwork":
		createdID := wickrLookupString(payload, pathParams, query, "networkId", "NetworkId", "id", "Id")
		if createdID == "" {
			createdID = s.nextNetworkIDLocked()
		}
		item := s.ensureNetworkLocked(createdID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name", "networkName", "NetworkName"); name != "" {
			item["Name"] = name
		}
		item["UpdatedTimestamp"] = now
		return map[string]any{"Network": wickrCloneMap(item)}
	case "GetNetwork":
		return map[string]any{"Network": wickrCloneMap(s.ensureNetworkLocked(networkID, now))}
	case "ListNetworks":
		return map[string]any{"Networks": s.listNetworksLocked(), "NextToken": ""}
	case "UpdateNetwork":
		item := s.ensureNetworkLocked(networkID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name", "networkName", "NetworkName"); name != "" {
			item["Name"] = name
		}
		if status := wickrLookupString(payload, pathParams, query, "status", "Status"); status != "" {
			item["Status"] = strings.ToUpper(status)
		}
		item["UpdatedTimestamp"] = now
		return map[string]any{"Network": wickrCloneMap(item)}
	case "DeleteNetwork":
		delete(s.networks, networkID)
		delete(s.networkSettings, networkID)
		delete(s.users, networkID)
		delete(s.userDevices, networkID)
		delete(s.bots, networkID)
		delete(s.dataRetentionBots, networkID)
		delete(s.securityGroups, networkID)
		delete(s.securityGroupUsers, networkID)
		delete(s.guestUsers, networkID)
		delete(s.blockedGuestUsers, networkID)
		delete(s.oidcConfigByNetworkID, networkID)
		return map[string]any{}

	case "GetNetworkSettings":
		return map[string]any{"NetworkSettings": wickrCloneMap(s.ensureNetworkSettingsLocked(networkID, now))}
	case "UpdateNetworkSettings":
		settings := s.ensureNetworkSettingsLocked(networkID, now)
		if v := wickrLookupString(payload, pathParams, query, "DefaultClient", "defaultClient"); v != "" {
			settings["DefaultClient"] = v
		}
		if v := wickrLookupString(payload, pathParams, query, "EnableFederation", "enableFederation"); v != "" {
			settings["EnableFederation"] = strings.EqualFold(v, "true")
		}
		settings["UpdatedTimestamp"] = now
		return map[string]any{"NetworkSettings": wickrCloneMap(settings)}

	case "BatchCreateUser":
		createdUserID := wickrLookupString(payload, pathParams, query, "userId", "UserId", "id", "Id")
		if createdUserID == "" {
			createdUserID = s.nextUserIDLocked()
		}
		user := s.ensureUserLocked(networkID, createdUserID, now)
		if uname := wickrLookupString(payload, pathParams, query, "username", "Username", "uname", "Uname"); uname != "" {
			user["Username"] = uname
		}
		user["Status"] = "ACTIVE"
		user["UpdatedTimestamp"] = now
		s.ensureUserDeviceLocked(networkID, createdUserID, now)
		return map[string]any{"Users": []any{wickrCloneMap(user)}, "Errors": []any{}}
	case "BatchDeleteUser":
		delete(s.users[networkID], userID)
		delete(s.userDevices[networkID], userID)
		return map[string]any{"Users": []any{map[string]any{"UserId": userID}}, "Errors": []any{}}
	case "BatchLookupUserUname":
		user := s.ensureUserLocked(networkID, userID, now)
		return map[string]any{"Users": []any{wickrCloneMap(user)}, "Errors": []any{}}
	case "BatchReinviteUser":
		user := s.ensureUserLocked(networkID, userID, now)
		user["Status"] = "INVITED"
		user["UpdatedTimestamp"] = now
		return map[string]any{"Users": []any{wickrCloneMap(user)}, "Errors": []any{}}
	case "BatchToggleUserSuspendStatus":
		suspend := strings.EqualFold(wickrLookupString(payload, pathParams, query, "suspend", "Suspend"), "true")
		user := s.ensureUserLocked(networkID, userID, now)
		if suspend {
			user["Status"] = "SUSPENDED"
		} else {
			user["Status"] = "ACTIVE"
		}
		user["UpdatedTimestamp"] = now
		return map[string]any{"Users": []any{wickrCloneMap(user)}, "Errors": []any{}}
	case "BatchResetDevicesForUser":
		devices := s.ensureUserDeviceLocked(networkID, userID, now)
		return map[string]any{"Devices": wickrCloneSlice(devices), "Errors": []any{}}
	case "GetUser":
		return map[string]any{"User": wickrCloneMap(s.ensureUserLocked(networkID, userID, now))}
	case "ListUsers":
		return map[string]any{"Users": s.listUsersLocked(networkID, now), "NextToken": ""}
	case "GetUsersCount":
		return map[string]any{"UsersCount": len(s.users[networkID])}
	case "UpdateUser":
		user := s.ensureUserLocked(networkID, userID, now)
		if uname := wickrLookupString(payload, pathParams, query, "username", "Username", "uname", "Uname"); uname != "" {
			user["Username"] = uname
		}
		if status := wickrLookupString(payload, pathParams, query, "status", "Status"); status != "" {
			user["Status"] = strings.ToUpper(status)
		}
		user["UpdatedTimestamp"] = now
		return map[string]any{"User": wickrCloneMap(user)}
	case "ListDevicesForUser":
		return map[string]any{"Devices": wickrCloneSlice(s.ensureUserDeviceLocked(networkID, userID, now)), "NextToken": ""}

	case "CreateBot":
		createdBotID := wickrLookupString(payload, pathParams, query, "botId", "BotId", "id", "Id")
		if createdBotID == "" {
			createdBotID = s.nextBotIDLocked()
		}
		bot := s.ensureBotLocked(networkID, createdBotID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name", "displayName", "DisplayName"); name != "" {
			bot["DisplayName"] = name
		}
		bot["Status"] = "ACTIVE"
		bot["UpdatedTimestamp"] = now
		return map[string]any{"Bot": wickrCloneMap(bot)}
	case "GetBot":
		return map[string]any{"Bot": wickrCloneMap(s.ensureBotLocked(networkID, botID, now))}
	case "ListBots":
		return map[string]any{"Bots": s.listBotsLocked(networkID, now), "NextToken": ""}
	case "GetBotsCount":
		return map[string]any{"BotsCount": len(s.bots[networkID])}
	case "UpdateBot":
		bot := s.ensureBotLocked(networkID, botID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name", "displayName", "DisplayName"); name != "" {
			bot["DisplayName"] = name
		}
		if status := wickrLookupString(payload, pathParams, query, "status", "Status"); status != "" {
			bot["Status"] = strings.ToUpper(status)
		}
		bot["UpdatedTimestamp"] = now
		return map[string]any{"Bot": wickrCloneMap(bot)}
	case "DeleteBot":
		delete(s.bots[networkID], botID)
		return map[string]any{}

	case "CreateDataRetentionBot":
		item := s.ensureDataRetentionBotLocked(networkID, now)
		item["Status"] = "ACTIVE"
		item["UpdatedTimestamp"] = now
		return map[string]any{"DataRetentionBot": wickrCloneMap(item)}
	case "CreateDataRetentionBotChallenge":
		return map[string]any{"Token": "challenge-token", "ExpiresInSeconds": 300}
	case "GetDataRetentionBot":
		return map[string]any{"DataRetentionBot": wickrCloneMap(s.ensureDataRetentionBotLocked(networkID, now))}
	case "UpdateDataRetention":
		item := s.ensureDataRetentionBotLocked(networkID, now)
		item["Status"] = "ACTIVE"
		item["UpdatedTimestamp"] = now
		return map[string]any{"DataRetentionBot": wickrCloneMap(item)}
	case "DeleteDataRetentionBot":
		delete(s.dataRetentionBots, networkID)
		return map[string]any{}

	case "CreateSecurityGroup":
		createdGroupID := wickrLookupString(payload, pathParams, query, "groupId", "GroupId", "id", "Id")
		if createdGroupID == "" {
			createdGroupID = s.nextGroupIDLocked()
		}
		group := s.ensureSecurityGroupLocked(networkID, createdGroupID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name"); name != "" {
			group["Name"] = name
		}
		group["UpdatedTimestamp"] = now
		return map[string]any{"SecurityGroup": wickrCloneMap(group)}
	case "GetSecurityGroup":
		return map[string]any{"SecurityGroup": wickrCloneMap(s.ensureSecurityGroupLocked(networkID, groupID, now))}
	case "ListSecurityGroups":
		return map[string]any{"SecurityGroups": s.listSecurityGroupsLocked(networkID, now), "NextToken": ""}
	case "ListSecurityGroupUsers":
		userIDs := s.securityGroupUsers[networkID][groupID]
		users := make([]any, 0, len(userIDs))
		for _, id := range userIDs {
			users = append(users, wickrCloneMap(s.ensureUserLocked(networkID, id, now)))
		}
		return map[string]any{"Users": users, "NextToken": ""}
	case "UpdateSecurityGroup":
		group := s.ensureSecurityGroupLocked(networkID, groupID, now)
		if name := wickrLookupString(payload, pathParams, query, "name", "Name"); name != "" {
			group["Name"] = name
		}
		group["UpdatedTimestamp"] = now
		return map[string]any{"SecurityGroup": wickrCloneMap(group)}
	case "DeleteSecurityGroup":
		delete(s.securityGroups[networkID], groupID)
		delete(s.securityGroupUsers[networkID], groupID)
		return map[string]any{}

	case "ListGuestUsers":
		return map[string]any{"GuestUsers": s.listGuestUsersLocked(networkID), "NextToken": ""}
	case "ListBlockedGuestUsers":
		return map[string]any{"GuestUsers": s.listBlockedGuestUsersLocked(networkID), "NextToken": ""}
	case "GetGuestUserHistoryCount":
		return map[string]any{"GuestUserHistoryCount": map[string]any{"Count": len(s.guestUsers[networkID])}}
	case "UpdateGuestUser":
		guest := s.ensureGuestUserLocked(networkID, usernameHash, now)
		actionValue := strings.ToUpper(wickrLookupString(payload, pathParams, query, "action", "Action", "status", "Status"))
		if actionValue == "BLOCK" || actionValue == "BLOCKED" {
			guest["Status"] = "BLOCKED"
			s.ensureBlockedGuestUserLocked(networkID, usernameHash, now)
		} else if actionValue != "" {
			guest["Status"] = actionValue
			delete(s.blockedGuestUsers[networkID], usernameHash)
		}
		guest["UpdatedTimestamp"] = now
		return map[string]any{"GuestUser": wickrCloneMap(guest)}

	case "GetOidcInfo":
		return map[string]any{
			"OidcConfigInfo": wickrCloneMap(s.ensureOIDCConfigLocked(networkID, now)),
			"OidcTokenInfo": map[string]any{
				"TokenType":        "Bearer",
				"ExpiresInSeconds": 3600,
				"AccessToken":      "token-" + networkID,
			},
		}
	case "RegisterOidcConfig":
		oidc := s.ensureOIDCConfigLocked(networkID, now)
		for _, key := range []string{"Url", "ClientId", "RedirectUri", "GrantType"} {
			if value := wickrLookupString(payload, pathParams, query, key, strings.ToLower(key)); value != "" {
				oidc[key] = value
			}
		}
		oidc["UpdatedTimestamp"] = now
		return map[string]any{"OidcConfigInfo": wickrCloneMap(oidc)}
	case "RegisterOidcConfigTest":
		return map[string]any{
			"OidcTokenInfo": map[string]any{
				"TokenType":        "Bearer",
				"ExpiresInSeconds": 300,
				"AccessToken":      "test-token-" + networkID,
			},
		}
	}

	return map[string]any{}
}

func (s *wickrStore) nextNetworkIDLocked() string {
	id := s.nextNetwork
	s.nextNetwork++
	return fmt.Sprintf("n-%06d", id)
}

func (s *wickrStore) nextUserIDLocked() string {
	id := s.nextUser
	s.nextUser++
	return fmt.Sprintf("u-%06d", id)
}

func (s *wickrStore) nextBotIDLocked() string {
	id := s.nextBot
	s.nextBot++
	return fmt.Sprintf("b-%06d", id)
}

func (s *wickrStore) nextGroupIDLocked() string {
	id := s.nextGroup
	s.nextGroup++
	return fmt.Sprintf("g-%06d", id)
}

func (s *wickrStore) ensureNetworkLocked(networkID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if existing := s.networks[networkID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"Arn":              wickrNetworkARN(networkID),
		"Name":             "stackyard-network-" + networkID,
		"Status":           "ACTIVE",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.networks[networkID] = item
	return item
}

func (s *wickrStore) ensureNetworkSettingsLocked(networkID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if existing := s.networkSettings[networkID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"DefaultClient":    "web",
		"EnableFederation": true,
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.networkSettings[networkID] = item
	return item
}

func (s *wickrStore) ensureUserLocked(networkID, userID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "u-000001"
	}
	if s.users[networkID] == nil {
		s.users[networkID] = map[string]map[string]any{}
	}
	if existing := s.users[networkID][userID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":          networkID,
		"UserId":             userID,
		"Arn":                wickrUserARN(networkID, userID),
		"Username":           "stackyard-user-" + userID,
		"Status":             "ACTIVE",
		"CreatedTimestamp":   now,
		"UpdatedTimestamp":   now,
		"InviteStatus":       "ACCEPTED",
		"ConnectionStatus":   "CONNECTED",
		"DeviceCount":        1,
		"SecurityGroupCount": 1,
	}
	s.users[networkID][userID] = item
	return item
}

func (s *wickrStore) ensureUserDeviceLocked(networkID, userID, now string) []any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "u-000001"
	}
	if s.userDevices[networkID] == nil {
		s.userDevices[networkID] = map[string][]any{}
	}
	if existing := s.userDevices[networkID][userID]; len(existing) > 0 {
		return existing
	}
	devices := []any{
		map[string]any{
			"DeviceId":         "device-000001",
			"UserId":           userID,
			"NetworkId":        networkID,
			"Status":           "ACTIVE",
			"UpdatedTimestamp": now,
		},
	}
	s.userDevices[networkID][userID] = devices
	return devices
}

func (s *wickrStore) ensureBotLocked(networkID, botID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		botID = "b-000001"
	}
	if s.bots[networkID] == nil {
		s.bots[networkID] = map[string]map[string]any{}
	}
	if existing := s.bots[networkID][botID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"BotId":            botID,
		"Arn":              wickrBotARN(networkID, botID),
		"DisplayName":      "stackyard-bot-" + botID,
		"Status":           "ACTIVE",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.bots[networkID][botID] = item
	return item
}

func (s *wickrStore) ensureDataRetentionBotLocked(networkID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if existing := s.dataRetentionBots[networkID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"BotId":            "drb-" + strings.TrimPrefix(networkID, "n-"),
		"Status":           "ACTIVE",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.dataRetentionBots[networkID] = item
	return item
}

func (s *wickrStore) ensureSecurityGroupLocked(networkID, groupID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		groupID = "g-000001"
	}
	if s.securityGroups[networkID] == nil {
		s.securityGroups[networkID] = map[string]map[string]any{}
	}
	if s.securityGroupUsers[networkID] == nil {
		s.securityGroupUsers[networkID] = map[string][]string{}
	}
	if existing := s.securityGroups[networkID][groupID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"GroupId":          groupID,
		"Arn":              wickrSecurityGroupARN(networkID, groupID),
		"Name":             "stackyard-security-group-" + groupID,
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.securityGroups[networkID][groupID] = item
	s.securityGroupUsers[networkID][groupID] = []string{"u-000001"}
	return item
}

func (s *wickrStore) ensureGuestUserLocked(networkID, usernameHash, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	usernameHash = strings.TrimSpace(usernameHash)
	if usernameHash == "" {
		usernameHash = "uh-000001"
	}
	if s.guestUsers[networkID] == nil {
		s.guestUsers[networkID] = map[string]map[string]any{}
	}
	if existing := s.guestUsers[networkID][usernameHash]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"UsernameHash":     usernameHash,
		"Username":         "guest-" + usernameHash,
		"Status":           "ACTIVE",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.guestUsers[networkID][usernameHash] = item
	return item
}

func (s *wickrStore) ensureBlockedGuestUserLocked(networkID, usernameHash, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	usernameHash = strings.TrimSpace(usernameHash)
	if usernameHash == "" {
		usernameHash = "uh-000001"
	}
	if s.blockedGuestUsers[networkID] == nil {
		s.blockedGuestUsers[networkID] = map[string]map[string]any{}
	}
	if existing := s.blockedGuestUsers[networkID][usernameHash]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"UsernameHash":     usernameHash,
		"Username":         "guest-" + usernameHash,
		"Status":           "BLOCKED",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.blockedGuestUsers[networkID][usernameHash] = item
	return item
}

func (s *wickrStore) ensureOIDCConfigLocked(networkID, now string) map[string]any {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if existing := s.oidcConfigByNetworkID[networkID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"NetworkId":        networkID,
		"Url":              "https://idp.example.com",
		"ClientId":         "stackyard-client-id",
		"RedirectUri":      "https://localhost/callback",
		"GrantType":        "authorization_code",
		"CreatedTimestamp": now,
		"UpdatedTimestamp": now,
	}
	s.oidcConfigByNetworkID[networkID] = item
	return item
}

func (s *wickrStore) listNetworksLocked() []any {
	keys := make([]string, 0, len(s.networks))
	for k := range s.networks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.networks[k]))
	}
	return out
}

func (s *wickrStore) listUsersLocked(networkID, now string) []any {
	if s.users[networkID] == nil {
		_ = s.ensureUserLocked(networkID, "u-000001", now)
	}
	keys := make([]string, 0, len(s.users[networkID]))
	for k := range s.users[networkID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.users[networkID][k]))
	}
	return out
}

func (s *wickrStore) listBotsLocked(networkID, now string) []any {
	if s.bots[networkID] == nil {
		_ = s.ensureBotLocked(networkID, "b-000001", now)
	}
	keys := make([]string, 0, len(s.bots[networkID]))
	for k := range s.bots[networkID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.bots[networkID][k]))
	}
	return out
}

func (s *wickrStore) listSecurityGroupsLocked(networkID, now string) []any {
	if s.securityGroups[networkID] == nil {
		_ = s.ensureSecurityGroupLocked(networkID, "g-000001", now)
	}
	keys := make([]string, 0, len(s.securityGroups[networkID]))
	for k := range s.securityGroups[networkID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.securityGroups[networkID][k]))
	}
	return out
}

func (s *wickrStore) listGuestUsersLocked(networkID string) []any {
	keys := make([]string, 0, len(s.guestUsers[networkID]))
	for k := range s.guestUsers[networkID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.guestUsers[networkID][k]))
	}
	return out
}

func (s *wickrStore) listBlockedGuestUsersLocked(networkID string) []any {
	keys := make([]string, 0, len(s.blockedGuestUsers[networkID]))
	for k := range s.blockedGuestUsers[networkID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, wickrCloneMap(s.blockedGuestUsers[networkID][k]))
	}
	return out
}

func wickrNetworkARN(networkID string) string {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		networkID = "n-000001"
	}
	return fmt.Sprintf("arn:aws:wickr:%s:%s:network/%s", wickrDefaultRegion, wickrDefaultAccountID, networkID)
}

func wickrUserARN(networkID, userID string) string {
	networkID = strings.TrimSpace(networkID)
	userID = strings.TrimSpace(userID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if userID == "" {
		userID = "u-000001"
	}
	return fmt.Sprintf("arn:aws:wickr:%s:%s:network/%s/user/%s", wickrDefaultRegion, wickrDefaultAccountID, networkID, userID)
}

func wickrBotARN(networkID, botID string) string {
	networkID = strings.TrimSpace(networkID)
	botID = strings.TrimSpace(botID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if botID == "" {
		botID = "b-000001"
	}
	return fmt.Sprintf("arn:aws:wickr:%s:%s:network/%s/bot/%s", wickrDefaultRegion, wickrDefaultAccountID, networkID, botID)
}

func wickrSecurityGroupARN(networkID, groupID string) string {
	networkID = strings.TrimSpace(networkID)
	groupID = strings.TrimSpace(groupID)
	if networkID == "" {
		networkID = "n-000001"
	}
	if groupID == "" {
		groupID = "g-000001"
	}
	return fmt.Sprintf("arn:aws:wickr:%s:%s:network/%s/security-group/%s", wickrDefaultRegion, wickrDefaultAccountID, networkID, groupID)
}

func wickrLookupString(payload map[string]any, pathParams map[string]string, query url.Values, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := pathParams[key]; ok {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
		for _, queryValue := range query[strings.ToLower(key)] {
			if value := strings.TrimSpace(queryValue); value != "" {
				return value
			}
		}
		if payload == nil {
			continue
		}
		for _, candidate := range []string{key, strings.ToLower(key), strings.ToUpper(key)} {
			raw, ok := payload[candidate]
			if !ok || raw == nil {
				continue
			}
			if value, ok := wickrStringFromAny(raw); ok && value != "" {
				return value
			}
		}
	}
	return ""
}

func wickrStringFromAny(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case json.Number:
		return strings.TrimSpace(typed.String()), true
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed)), true
	case int:
		return strings.TrimSpace(fmt.Sprintf("%d", typed)), true
	case int64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed)), true
	case uint64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed)), true
	case bool:
		if typed {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func wickrCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = wickrCloneValue(v)
	}
	return out
}

func wickrCloneSlice(in []any) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = wickrCloneValue(in[i])
	}
	return out
}

func wickrCloneValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return wickrCloneMap(typed)
	case []any:
		return wickrCloneSlice(typed)
	default:
		return typed
	}
}
