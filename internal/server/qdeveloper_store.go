package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type qDeveloperStore struct {
	mu                 sync.Mutex
	nextID             int64
	accountPreferences map[string]any
	associations       map[string]map[string]any
	slackChannels      map[string]map[string]any
	slackUsers         map[string]map[string]any
	slackWorkspaces    map[string]map[string]any
	teamsChannels      map[string]map[string]any
	teamsConfigured    map[string]map[string]any
	teamsUsers         map[string]map[string]any
	chimeWebhooks      map[string]map[string]any
	customActions      map[string]map[string]any
	tags               map[string]map[string]string
}

func newQDeveloperStore() *qDeveloperStore {
	now := time.Now().UTC()
	store := &qDeveloperStore{
		nextID: 2,
		accountPreferences: map[string]any{
			"UserAuthorizationRequired":     true,
			"TrainingDataCollectionEnabled": false,
		},
		associations:    map[string]map[string]any{},
		slackChannels:   map[string]map[string]any{},
		slackUsers:      map[string]map[string]any{},
		slackWorkspaces: map[string]map[string]any{},
		teamsChannels:   map[string]map[string]any{},
		teamsConfigured: map[string]map[string]any{},
		teamsUsers:      map[string]map[string]any{},
		chimeWebhooks:   map[string]map[string]any{},
		customActions:   map[string]map[string]any{},
		tags:            map[string]map[string]string{},
	}

	slackArn := qDeveloperARN("slack-channel", "stackyard-slack-channel")
	store.slackChannels[slackArn] = map[string]any{
		"ChatConfigurationArn": slackArn,
		"ConfigurationName":    "stackyard-slack-channel",
		"IamRoleArn":           "arn:aws:iam::123456789012:role/stackyard-chatbot",
		"SlackWorkspaceId":     "T00000001",
		"SlackChannelId":       "C00000001",
		"SnsTopicArns":         []any{"arn:aws:sns:us-east-1:123456789012:stackyard-topic"},
		"LoggingLevel":         "ERROR",
		"State":                "ENABLED",
		"StateReason":          "",
	}
	store.slackWorkspaces["T00000001"] = map[string]any{
		"SlackWorkspaceId":   "T00000001",
		"SlackWorkspaceName": "stackyard-workspace",
	}
	store.slackUsers["stackyard-slack-user"] = map[string]any{
		"ChatConfigurationArn": slackArn,
		"SlackUserId":          "U00000001",
		"TeamId":               "T00000001",
		"AwsUserIdentityArn":   "arn:aws:iam::123456789012:user/stackyard",
	}

	teamsArn := qDeveloperARN("microsoft-teams-channel", "stackyard-teams-channel")
	store.teamsChannels[teamsArn] = map[string]any{
		"ChannelConfigurationArn": teamsArn,
		"ChannelId":               "19:stackyardchannel@thread.tacv2",
		"ChannelName":             "stackyard-channel",
		"TeamId":                  "00000000-0000-0000-0000-000000000000",
		"TeamName":                "stackyard-team",
		"TenantId":                "11111111-1111-1111-1111-111111111111",
		"IamRoleArn":              "arn:aws:iam::123456789012:role/stackyard-chatbot",
		"LoggingLevel":            "ERROR",
		"SnsTopicArns":            []any{"arn:aws:sns:us-east-1:123456789012:stackyard-topic"},
		"State":                   "ENABLED",
	}
	store.teamsConfigured["00000000-0000-0000-0000-000000000000"] = map[string]any{
		"TeamId":   "00000000-0000-0000-0000-000000000000",
		"TeamName": "stackyard-team",
		"TenantId": "11111111-1111-1111-1111-111111111111",
	}
	store.teamsUsers["stackyard-teams-user"] = map[string]any{
		"UserId":             "stackyard-teams-user",
		"TeamsChannelId":     "19:stackyardchannel@thread.tacv2",
		"AwsUserIdentityArn": "arn:aws:iam::123456789012:user/stackyard",
	}

	chimeArn := qDeveloperARN("chime-webhook", "stackyard-chime-webhook")
	store.chimeWebhooks[chimeArn] = map[string]any{
		"ChatConfigurationArn": chimeArn,
		"WebhookDescription":   "stackyard chime webhook",
		"IamRoleArn":           "arn:aws:iam::123456789012:role/stackyard-chatbot",
		"SnsTopicArns":         []any{"arn:aws:sns:us-east-1:123456789012:stackyard-topic"},
		"LoggingLevel":         "ERROR",
	}

	customActionArn := qDeveloperARN("custom-action", "stackyard-custom-action")
	store.customActions[customActionArn] = map[string]any{
		"CustomActionArn": customActionArn,
		"ActionName":      "stackyard-custom-action",
		"AliasName":       "stackyard",
		"Definition": map[string]any{
			"CommandText": "help",
		},
	}

	associationID := fmt.Sprintf("association-%06d", 1)
	store.associations[associationID] = map[string]any{
		"AssociationId":        associationID,
		"ChatConfigurationArn": slackArn,
		"Resource":             "arn:aws:codebuild:us-east-1:123456789012:project/stackyard",
		"AssociationType":      "AWS_RESOURCE",
		"Status":               "ASSOCIATED",
		"CreatedTimestamp":     now,
		"LastUpdatedTimestamp": now,
	}

	store.tags[slackArn] = map[string]string{"seed": "true", "service": "qdeveloper"}
	store.tags[teamsArn] = map[string]string{"seed": "true", "service": "qdeveloper"}
	store.tags[chimeArn] = map[string]string{"seed": "true", "service": "qdeveloper"}
	store.tags[customActionArn] = map[string]string{"seed": "true", "service": "qdeveloper"}
	return store
}

func (s *qDeveloperStore) Handle(action string, payload map[string]any, _ map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "GetAccountPreferences":
		return map[string]any{"AccountPreferences": qDeveloperCloneMap(s.accountPreferences)}
	case "UpdateAccountPreferences":
		for key, value := range payload {
			s.accountPreferences[key] = value
		}
		return map[string]any{"AccountPreferences": qDeveloperCloneMap(s.accountPreferences)}

	case "ListAssociations":
		return map[string]any{"Associations": s.listMapValuesLocked(s.associations), "NextToken": ""}
	case "AssociateToConfiguration":
		id := fmt.Sprintf("association-%06d", s.nextLocked())
		association := map[string]any{
			"AssociationId":        id,
			"ChatConfigurationArn": qDeveloperDefaultString(payload, []string{"ChatConfigurationArn", "ChatConfiguration"}, s.firstSlackARNLocked()),
			"Resource":             qDeveloperDefaultString(payload, []string{"Resource"}, "arn:aws:codebuild:us-east-1:123456789012:project/stackyard"),
			"AssociationType":      qDeveloperDefaultString(payload, []string{"AssociationType"}, "AWS_RESOURCE"),
			"Status":               "ASSOCIATED",
			"CreatedTimestamp":     time.Now().UTC(),
		}
		s.associations[id] = association
		return map[string]any{"Association": qDeveloperCloneMap(association)}
	case "DisassociateFromConfiguration":
		associationID := qDeveloperDefaultString(payload, []string{"AssociationId"}, "")
		if associationID != "" {
			delete(s.associations, associationID)
		} else if id := s.firstAssociationIDLocked(); id != "" {
			delete(s.associations, id)
		}
		return map[string]any{}

	case "CreateSlackChannelConfiguration", "UpdateSlackChannelConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChatConfigurationArn"}, "")
		if arn == "" {
			name := qDeveloperDefaultString(payload, []string{"ConfigurationName"}, "stackyard-slack-channel")
			arn = qDeveloperARN("slack-channel", name)
		}
		conf := s.ensureSlackChannelLocked(arn)
		qDeveloperMergeKnown(conf, payload)
		return map[string]any{"ChannelConfiguration": qDeveloperCloneMap(conf)}
	case "DescribeSlackChannelConfigurations":
		return map[string]any{"SlackChannelConfigurations": s.listMapValuesLocked(s.slackChannels), "NextToken": ""}
	case "DeleteSlackChannelConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChatConfigurationArn"}, s.firstSlackARNLocked())
		delete(s.slackChannels, arn)
		return map[string]any{}

	case "DescribeSlackWorkspaces":
		return map[string]any{"SlackWorkspaces": s.listMapValuesLocked(s.slackWorkspaces), "NextToken": ""}
	case "DescribeSlackUserIdentities":
		return map[string]any{"SlackUserIdentities": s.listMapValuesLocked(s.slackUsers), "NextToken": ""}
	case "DeleteSlackUserIdentity", "DeleteSlackWorkspaceAuthorization":
		return map[string]any{}

	case "CreateMicrosoftTeamsChannelConfiguration", "UpdateMicrosoftTeamsChannelConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChannelConfigurationArn"}, "")
		if arn == "" {
			name := qDeveloperDefaultString(payload, []string{"ConfigurationName", "ChannelName"}, "stackyard-teams-channel")
			arn = qDeveloperARN("microsoft-teams-channel", name)
		}
		conf := s.ensureTeamsChannelLocked(arn)
		qDeveloperMergeKnown(conf, payload)
		return map[string]any{"ChannelConfiguration": qDeveloperCloneMap(conf)}
	case "GetMicrosoftTeamsChannelConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChannelConfigurationArn"}, s.firstTeamsARNLocked())
		return map[string]any{"ChannelConfiguration": qDeveloperCloneMap(s.ensureTeamsChannelLocked(arn))}
	case "ListMicrosoftTeamsChannelConfigurations":
		return map[string]any{"TeamChannelConfigurations": s.listMapValuesLocked(s.teamsChannels), "NextToken": ""}
	case "DeleteMicrosoftTeamsChannelConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChannelConfigurationArn"}, s.firstTeamsARNLocked())
		delete(s.teamsChannels, arn)
		return map[string]any{}
	case "ListMicrosoftTeamsConfiguredTeams":
		return map[string]any{"ConfiguredTeams": s.listMapValuesLocked(s.teamsConfigured), "NextToken": ""}
	case "DeleteMicrosoftTeamsConfiguredTeam":
		teamID := qDeveloperDefaultString(payload, []string{"TeamId"}, "")
		if teamID != "" {
			delete(s.teamsConfigured, teamID)
		}
		return map[string]any{}
	case "ListMicrosoftTeamsUserIdentities":
		return map[string]any{"UserIdentities": s.listMapValuesLocked(s.teamsUsers), "NextToken": ""}
	case "DeleteMicrosoftTeamsUserIdentity":
		userID := qDeveloperDefaultString(payload, []string{"UserId"}, "")
		if userID != "" {
			delete(s.teamsUsers, userID)
		}
		return map[string]any{}

	case "CreateChimeWebhookConfiguration", "UpdateChimeWebhookConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChatConfigurationArn"}, "")
		if arn == "" {
			name := qDeveloperDefaultString(payload, []string{"ConfigurationName"}, "stackyard-chime-webhook")
			arn = qDeveloperARN("chime-webhook", name)
		}
		conf := s.ensureChimeWebhookLocked(arn)
		qDeveloperMergeKnown(conf, payload)
		return map[string]any{"WebhookConfiguration": qDeveloperCloneMap(conf)}
	case "DescribeChimeWebhookConfigurations":
		return map[string]any{"WebhookConfigurations": s.listMapValuesLocked(s.chimeWebhooks), "NextToken": ""}
	case "DeleteChimeWebhookConfiguration":
		arn := qDeveloperDefaultString(payload, []string{"ChatConfigurationArn"}, s.firstChimeARNLocked())
		delete(s.chimeWebhooks, arn)
		return map[string]any{}

	case "CreateCustomAction", "UpdateCustomAction":
		arn := qDeveloperDefaultString(payload, []string{"CustomActionArn"}, "")
		if arn == "" {
			name := qDeveloperDefaultString(payload, []string{"ActionName"}, "stackyard-custom-action")
			arn = qDeveloperARN("custom-action", name)
		}
		actionItem := s.ensureCustomActionLocked(arn)
		qDeveloperMergeKnown(actionItem, payload)
		return map[string]any{"CustomActionArn": arn}
	case "GetCustomAction":
		arn := qDeveloperDefaultString(payload, []string{"CustomActionArn"}, s.firstCustomActionARNLocked())
		return map[string]any{"CustomAction": qDeveloperCloneMap(s.ensureCustomActionLocked(arn))}
	case "ListCustomActions":
		return map[string]any{"CustomActions": s.listMapValuesLocked(s.customActions), "NextToken": ""}
	case "DeleteCustomAction":
		arn := qDeveloperDefaultString(payload, []string{"CustomActionArn"}, s.firstCustomActionARNLocked())
		delete(s.customActions, arn)
		return map[string]any{}

	case "TagResource":
		arn := qDeveloperDefaultString(payload, []string{"ResourceArn"}, s.firstSlackARNLocked())
		existing := s.ensureTagsLocked(arn)
		for key, value := range qDeveloperStringMap(qDeveloperPayloadValue(payload, "Tags")) {
			existing[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		arn := qDeveloperDefaultString(payload, []string{"ResourceArn"}, s.firstSlackARNLocked())
		keys := qDeveloperStringSlice(qDeveloperPayloadValue(payload, "TagKeys"))
		existing := s.ensureTagsLocked(arn)
		for _, key := range keys {
			delete(existing, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		arn := qDeveloperDefaultString(payload, []string{"ResourceArn"}, s.firstSlackARNLocked())
		return map[string]any{"Tags": s.cloneTagsLocked(arn)}
	}

	return map[string]any{}
}

func (s *qDeveloperStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *qDeveloperStore) listMapValuesLocked(source map[string]map[string]any) []any {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, qDeveloperCloneMap(source[key]))
	}
	return out
}

func (s *qDeveloperStore) firstAssociationIDLocked() string {
	keys := make([]string, 0, len(s.associations))
	for key := range s.associations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func (s *qDeveloperStore) firstSlackARNLocked() string {
	keys := make([]string, 0, len(s.slackChannels))
	for key := range s.slackChannels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		arn := qDeveloperARN("slack-channel", "stackyard-slack-channel")
		s.slackChannels[arn] = map[string]any{"ChatConfigurationArn": arn, "ConfigurationName": "stackyard-slack-channel"}
		return arn
	}
	return keys[0]
}

func (s *qDeveloperStore) firstTeamsARNLocked() string {
	keys := make([]string, 0, len(s.teamsChannels))
	for key := range s.teamsChannels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		arn := qDeveloperARN("microsoft-teams-channel", "stackyard-teams-channel")
		s.teamsChannels[arn] = map[string]any{"ChannelConfigurationArn": arn, "ChannelName": "stackyard-teams-channel"}
		return arn
	}
	return keys[0]
}

func (s *qDeveloperStore) firstChimeARNLocked() string {
	keys := make([]string, 0, len(s.chimeWebhooks))
	for key := range s.chimeWebhooks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		arn := qDeveloperARN("chime-webhook", "stackyard-chime-webhook")
		s.chimeWebhooks[arn] = map[string]any{"ChatConfigurationArn": arn}
		return arn
	}
	return keys[0]
}

func (s *qDeveloperStore) firstCustomActionARNLocked() string {
	keys := make([]string, 0, len(s.customActions))
	for key := range s.customActions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		arn := qDeveloperARN("custom-action", "stackyard-custom-action")
		s.customActions[arn] = map[string]any{"CustomActionArn": arn, "ActionName": "stackyard-custom-action"}
		return arn
	}
	return keys[0]
}

func (s *qDeveloperStore) ensureSlackChannelLocked(arn string) map[string]any {
	normalized := strings.TrimSpace(arn)
	if normalized == "" {
		normalized = s.firstSlackARNLocked()
	}
	if existing, ok := s.slackChannels[normalized]; ok {
		return existing
	}
	item := map[string]any{"ChatConfigurationArn": normalized, "ConfigurationName": qDeveloperNameFromARN(normalized)}
	s.slackChannels[normalized] = item
	return item
}

func (s *qDeveloperStore) ensureTeamsChannelLocked(arn string) map[string]any {
	normalized := strings.TrimSpace(arn)
	if normalized == "" {
		normalized = s.firstTeamsARNLocked()
	}
	if existing, ok := s.teamsChannels[normalized]; ok {
		return existing
	}
	item := map[string]any{"ChannelConfigurationArn": normalized, "ChannelName": qDeveloperNameFromARN(normalized)}
	s.teamsChannels[normalized] = item
	return item
}

func (s *qDeveloperStore) ensureChimeWebhookLocked(arn string) map[string]any {
	normalized := strings.TrimSpace(arn)
	if normalized == "" {
		normalized = s.firstChimeARNLocked()
	}
	if existing, ok := s.chimeWebhooks[normalized]; ok {
		return existing
	}
	item := map[string]any{"ChatConfigurationArn": normalized, "WebhookDescription": qDeveloperNameFromARN(normalized)}
	s.chimeWebhooks[normalized] = item
	return item
}

func (s *qDeveloperStore) ensureCustomActionLocked(arn string) map[string]any {
	normalized := strings.TrimSpace(arn)
	if normalized == "" {
		normalized = s.firstCustomActionARNLocked()
	}
	if existing, ok := s.customActions[normalized]; ok {
		return existing
	}
	item := map[string]any{"CustomActionArn": normalized, "ActionName": qDeveloperNameFromARN(normalized)}
	s.customActions[normalized] = item
	return item
}

func (s *qDeveloperStore) ensureTagsLocked(resourceARN string) map[string]string {
	normalized := strings.TrimSpace(resourceARN)
	if normalized == "" {
		normalized = s.firstSlackARNLocked()
	}
	if existing, ok := s.tags[normalized]; ok {
		return existing
	}
	s.tags[normalized] = map[string]string{}
	return s.tags[normalized]
}

func (s *qDeveloperStore) cloneTagsLocked(resourceARN string) map[string]string {
	existing := s.ensureTagsLocked(resourceARN)
	out := make(map[string]string, len(existing))
	for key, value := range existing {
		out[key] = value
	}
	return out
}

func qDeveloperPayloadValue(payload map[string]any, key string) any {
	for currentKey, value := range payload {
		if strings.EqualFold(currentKey, key) {
			return value
		}
	}
	return nil
}

func qDeveloperDefaultString(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if value := qDeveloperPayloadValue(payload, key); value != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", value))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return fallback
}

func qDeveloperStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", item)); trimmed != "" && trimmed != "<nil>" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return []string{}
	}
}

func qDeveloperStringMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, val := range typed {
			out[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	case map[string]any:
		for key, val := range typed {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			v := strings.TrimSpace(fmt.Sprintf("%v", val))
			if v == "<nil>" {
				v = ""
			}
			out[k] = v
		}
	}
	return out
}

func qDeveloperNameFromARN(arn string) string {
	trimmed := strings.TrimSpace(arn)
	if trimmed == "" {
		return "stackyard"
	}
	if slash := strings.LastIndex(trimmed, "/"); slash >= 0 && slash+1 < len(trimmed) {
		return trimmed[slash+1:]
	}
	if colon := strings.LastIndex(trimmed, ":"); colon >= 0 && colon+1 < len(trimmed) {
		return trimmed[colon+1:]
	}
	return trimmed
}

func qDeveloperARN(resourceType, name string) string {
	resource := strings.TrimSpace(resourceType)
	if resource == "" {
		resource = "resource"
	}
	item := strings.TrimSpace(name)
	if item == "" {
		item = "stackyard"
	}
	if strings.HasPrefix(item, "arn:") {
		return item
	}
	return fmt.Sprintf("arn:aws:chatbot:us-east-1:123456789012:%s/%s", resource, item)
}

func qDeveloperMergeKnown(target map[string]any, payload map[string]any) {
	for key, value := range payload {
		target[key] = value
	}
}

func qDeveloperCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = qDeveloperCloneMap(typed)
		case []any:
			out[key] = qDeveloperCloneSlice(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func qDeveloperCloneSlice(in []any) []any {
	if in == nil {
		return []any{}
	}
	out := make([]any, len(in))
	for i, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[i] = qDeveloperCloneMap(typed)
		case []any:
			out[i] = qDeveloperCloneSlice(typed)
		default:
			out[i] = typed
		}
	}
	return out
}
