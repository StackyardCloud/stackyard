package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type supportAppStore struct {
	mu sync.Mutex

	nextTeamID    int64
	nextChannelID int64
	accountAlias  string
	workspaces    map[string]map[string]any
	channels      map[string]map[string]any
}

func newSupportAppStore() *supportAppStore {
	seedTeamID := "T012ABCDEFG"
	seedChannelID := "C012ABCDEFG"

	s := &supportAppStore{
		nextTeamID:    2,
		nextChannelID: 2,
		accountAlias:  "stackyard-account",
		workspaces: map[string]map[string]any{
			seedTeamID: {
				"allowOrganizationMemberAccount": true,
				"teamId":                         seedTeamID,
				"teamName":                       "stackyard-workspace",
			},
		},
		channels: map[string]map[string]any{
			supportAppChannelKey(seedTeamID, seedChannelID): {
				"channelId":                       seedChannelID,
				"channelName":                     "stackyard-support",
				"channelRoleArn":                  "arn:aws:iam::123456789012:role/SupportAppChannelRole",
				"notifyOnAddCorrespondenceToCase": true,
				"notifyOnCaseSeverity":            "all",
				"notifyOnCreateOrReopenCase":      true,
				"notifyOnResolveCase":             true,
				"teamId":                          seedTeamID,
			},
		},
	}
	return s
}

func (s *supportAppStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateSlackChannelConfiguration":
		teamID := supportAppPayloadString(payload, "teamId", s.firstTeamIDLocked())
		workspace := s.ensureWorkspaceLocked(teamID)
		teamID = supportAppPayloadString(workspace, "teamId", teamID)
		channelID := supportAppPayloadString(payload, "channelId", s.nextChannelIdentifierLocked())
		cfg := s.ensureChannelLocked(teamID, channelID)

		cfg["channelId"] = channelID
		cfg["channelName"] = supportAppPayloadString(payload, "channelName", supportAppPayloadString(cfg, "channelName", "stackyard-support"))
		cfg["channelRoleArn"] = supportAppPayloadString(payload, "channelRoleArn", supportAppPayloadString(cfg, "channelRoleArn", "arn:aws:iam::123456789012:role/SupportAppChannelRole"))
		cfg["notifyOnAddCorrespondenceToCase"] = supportAppPayloadBool(payload, "notifyOnAddCorrespondenceToCase", supportAppPayloadBool(cfg, "notifyOnAddCorrespondenceToCase", true))
		cfg["notifyOnCaseSeverity"] = supportAppNormalizeSeverity(supportAppPayloadString(payload, "notifyOnCaseSeverity", supportAppPayloadString(cfg, "notifyOnCaseSeverity", "all")))
		cfg["notifyOnCreateOrReopenCase"] = supportAppPayloadBool(payload, "notifyOnCreateOrReopenCase", supportAppPayloadBool(cfg, "notifyOnCreateOrReopenCase", true))
		cfg["notifyOnResolveCase"] = supportAppPayloadBool(payload, "notifyOnResolveCase", supportAppPayloadBool(cfg, "notifyOnResolveCase", true))
		cfg["teamId"] = teamID
		return map[string]any{}

	case "DeleteAccountAlias":
		s.accountAlias = ""
		return map[string]any{}

	case "DeleteSlackChannelConfiguration":
		teamID := supportAppPayloadString(payload, "teamId", s.firstTeamIDLocked())
		channelID := supportAppPayloadString(payload, "channelId", s.firstChannelIDForTeamLocked(teamID))
		delete(s.channels, supportAppChannelKey(teamID, channelID))
		return map[string]any{}

	case "DeleteSlackWorkspaceConfiguration":
		teamID := supportAppPayloadString(payload, "teamId", s.firstTeamIDLocked())
		delete(s.workspaces, teamID)
		for key, cfg := range s.channels {
			if supportAppPayloadString(cfg, "teamId", "") == teamID {
				delete(s.channels, key)
			}
		}
		return map[string]any{}

	case "GetAccountAlias":
		return map[string]any{"accountAlias": s.accountAlias}

	case "ListSlackChannelConfigurations":
		out := make([]any, 0, len(s.channels))
		for _, cfg := range supportAppSortedChannels(s.channels) {
			out = append(out, supportAppCloneMap(cfg))
		}
		return map[string]any{
			"nextToken":                  "",
			"slackChannelConfigurations": out,
		}

	case "ListSlackWorkspaceConfigurations":
		out := make([]any, 0, len(s.workspaces))
		for _, ws := range supportAppSortedWorkspaces(s.workspaces) {
			out = append(out, supportAppCloneMap(ws))
		}
		return map[string]any{
			"nextToken":                    "",
			"slackWorkspaceConfigurations": out,
		}

	case "PutAccountAlias":
		s.accountAlias = supportAppPayloadString(payload, "accountAlias", s.accountAlias)
		return map[string]any{}

	case "RegisterSlackWorkspaceForOrganization":
		teamID := supportAppPayloadString(payload, "teamId", s.firstTeamIDLocked())
		workspace := s.ensureWorkspaceLocked(teamID)
		return map[string]any{
			"accountType": "management",
			"teamId":      supportAppPayloadString(workspace, "teamId", teamID),
			"teamName":    supportAppPayloadString(workspace, "teamName", "stackyard-workspace"),
		}

	case "UpdateSlackChannelConfiguration":
		teamID := supportAppPayloadString(payload, "teamId", s.firstTeamIDLocked())
		s.ensureWorkspaceLocked(teamID)
		channelID := supportAppPayloadString(payload, "channelId", s.firstChannelIDForTeamLocked(teamID))
		cfg := s.ensureChannelLocked(teamID, channelID)

		cfg["channelId"] = channelID
		cfg["channelName"] = supportAppPayloadString(payload, "channelName", supportAppPayloadString(cfg, "channelName", "stackyard-support"))
		cfg["channelRoleArn"] = supportAppPayloadString(payload, "channelRoleArn", supportAppPayloadString(cfg, "channelRoleArn", "arn:aws:iam::123456789012:role/SupportAppChannelRole"))
		cfg["notifyOnAddCorrespondenceToCase"] = supportAppPayloadBool(payload, "notifyOnAddCorrespondenceToCase", supportAppPayloadBool(cfg, "notifyOnAddCorrespondenceToCase", true))
		cfg["notifyOnCaseSeverity"] = supportAppNormalizeSeverity(supportAppPayloadString(payload, "notifyOnCaseSeverity", supportAppPayloadString(cfg, "notifyOnCaseSeverity", "all")))
		cfg["notifyOnCreateOrReopenCase"] = supportAppPayloadBool(payload, "notifyOnCreateOrReopenCase", supportAppPayloadBool(cfg, "notifyOnCreateOrReopenCase", true))
		cfg["notifyOnResolveCase"] = supportAppPayloadBool(payload, "notifyOnResolveCase", supportAppPayloadBool(cfg, "notifyOnResolveCase", true))
		cfg["teamId"] = teamID
		return supportAppCloneMap(cfg)
	}

	return map[string]any{}
}

func (s *supportAppStore) ensureWorkspaceLocked(teamID string) map[string]any {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		teamID = s.nextTeamIdentifierLocked()
	}
	if workspace, ok := s.workspaces[teamID]; ok {
		return workspace
	}
	workspace := map[string]any{
		"allowOrganizationMemberAccount": true,
		"teamId":                         teamID,
		"teamName":                       fmt.Sprintf("stackyard-%s", strings.ToLower(teamID)),
	}
	s.workspaces[teamID] = workspace
	return workspace
}

func (s *supportAppStore) ensureChannelLocked(teamID, channelID string) map[string]any {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		channelID = s.nextChannelIdentifierLocked()
	}
	key := supportAppChannelKey(teamID, channelID)
	if channel, ok := s.channels[key]; ok {
		return channel
	}
	channel := map[string]any{
		"channelId":                       channelID,
		"channelName":                     "stackyard-support",
		"channelRoleArn":                  "arn:aws:iam::123456789012:role/SupportAppChannelRole",
		"notifyOnAddCorrespondenceToCase": true,
		"notifyOnCaseSeverity":            "all",
		"notifyOnCreateOrReopenCase":      true,
		"notifyOnResolveCase":             true,
		"teamId":                          teamID,
	}
	s.channels[key] = channel
	return channel
}

func (s *supportAppStore) nextTeamIdentifierLocked() string {
	id := s.nextTeamID
	s.nextTeamID++
	return fmt.Sprintf("T%09d", id)
}

func (s *supportAppStore) nextChannelIdentifierLocked() string {
	id := s.nextChannelID
	s.nextChannelID++
	return fmt.Sprintf("C%09d", id)
}

func (s *supportAppStore) firstTeamIDLocked() string {
	if len(s.workspaces) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.workspaces))
	for key := range s.workspaces {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *supportAppStore) firstChannelIDForTeamLocked(teamID string) string {
	teamID = strings.TrimSpace(teamID)
	channels := supportAppSortedChannels(s.channels)
	for _, channel := range channels {
		if supportAppPayloadString(channel, "teamId", "") == teamID {
			return supportAppPayloadString(channel, "channelId", "")
		}
	}
	if len(channels) == 0 {
		return ""
	}
	return supportAppPayloadString(channels[0], "channelId", "")
}

func supportAppChannelKey(teamID, channelID string) string {
	return strings.TrimSpace(teamID) + "::" + strings.TrimSpace(channelID)
}

func supportAppPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return strings.TrimSpace(def)
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return strings.TrimSpace(def)
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return strings.TrimSpace(def)
		}
		return strings.TrimSpace(v)
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str == "" {
			return strings.TrimSpace(def)
		}
		return str
	}
}

func supportAppPayloadBool(payload map[string]any, key string, def bool) bool {
	if payload == nil {
		return def
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return def
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		if trimmed == "" {
			return def
		}
		if parsed, err := strconv.ParseBool(trimmed); err == nil {
			return parsed
		}
		return def
	default:
		return def
	}
}

func supportAppNormalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "all", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}

func supportAppSortedWorkspaces(in map[string]map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, supportAppCloneMap(in[key]))
	}
	return out
}

func supportAppSortedChannels(in map[string]map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, supportAppCloneMap(in[key]))
	}
	return out
}

func supportAppCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = supportAppCloneValue(value)
	}
	return out
}

func supportAppCloneSlice(in []any) []any {
	if in == nil {
		return nil
	}
	out := make([]any, len(in))
	for i := range in {
		out[i] = supportAppCloneValue(in[i])
	}
	return out
}

func supportAppCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return supportAppCloneMap(typed)
	case []any:
		return supportAppCloneSlice(typed)
	default:
		return typed
	}
}
