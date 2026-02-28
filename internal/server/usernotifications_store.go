package server

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type userNotificationsStore struct {
	mu sync.Mutex

	counter int

	orgAccessEnabled bool

	notificationConfigurations map[string]map[string]any
	eventRules                 map[string]map[string]any
	notificationEvents         map[string]map[string]any
	managedNotificationEvents  map[string]map[string]any
	managedChildEvents         map[string]map[string]any
	managedConfigurations      map[string]map[string]any
	channels                   map[string]map[string]any
	notificationHubs           map[string]map[string]any

	channelAssociations         map[string]string
	managedAdditionalChannels   map[string]bool
	managedAccountContacts      map[string]bool
	organizationalUnitAssociate map[string]bool

	memberAccounts []map[string]any
	tags           map[string]map[string]string
}

func newUserNotificationsStore() *userNotificationsStore {
	s := &userNotificationsStore{
		counter: 1000,

		orgAccessEnabled: false,

		notificationConfigurations: map[string]map[string]any{},
		eventRules:                 map[string]map[string]any{},
		notificationEvents:         map[string]map[string]any{},
		managedNotificationEvents:  map[string]map[string]any{},
		managedChildEvents:         map[string]map[string]any{},
		managedConfigurations:      map[string]map[string]any{},
		channels:                   map[string]map[string]any{},
		notificationHubs:           map[string]map[string]any{},

		channelAssociations:         map[string]string{},
		managedAdditionalChannels:   map[string]bool{},
		managedAccountContacts:      map[string]bool{},
		organizationalUnitAssociate: map[string]bool{},

		memberAccounts: []map[string]any{
			{"accountId": "123456789012"},
			{"accountId": "210987654321"},
		},
		tags: map[string]map[string]string{},
	}

	cfgArn := "arn:aws:notifications:us-east-1:123456789012:notification-configuration/nc-000001"
	s.notificationConfigurations[cfgArn] = map[string]any{
		"arn":    cfgArn,
		"name":   "stackyard-default-configuration",
		"status": "ACTIVE",
	}

	ruleArn := "arn:aws:notifications:us-east-1:123456789012:event-rule/er-000001"
	s.eventRules[ruleArn] = map[string]any{
		"arn":    ruleArn,
		"name":   "stackyard-default-rule",
		"status": "ACTIVE",
		"statusSummary": map[string]any{
			"status": "ACTIVE",
		},
	}

	eventArn := "arn:aws:notifications:us-east-1:123456789012:notification-event/ne-000001"
	s.notificationEvents[eventArn] = map[string]any{
		"arn":       eventArn,
		"source":    "stackyard",
		"eventType": "NotificationEvent",
	}

	managedEventArn := "arn:aws:notifications:us-east-1:123456789012:managed-notification-event/mne-000001"
	s.managedNotificationEvents[managedEventArn] = map[string]any{
		"arn":       managedEventArn,
		"source":    "aws.health",
		"eventType": "ManagedNotificationEvent",
	}

	childArn := "arn:aws:notifications:us-east-1:123456789012:managed-notification-child-event/mnce-000001"
	s.managedChildEvents[childArn] = map[string]any{
		"arn":          childArn,
		"parentArn":    managedEventArn,
		"source":       "aws.health",
		"eventType":    "ManagedNotificationChildEvent",
		"notification": "stackyard managed child event",
	}

	managedCfgArn := "arn:aws:notifications:us-east-1:123456789012:managed-notification-configuration/mnc-000001"
	s.managedConfigurations[managedCfgArn] = map[string]any{
		"arn":    managedCfgArn,
		"name":   "stackyard-managed-default",
		"status": "ACTIVE",
	}

	channelArn := "arn:aws:notifications:us-east-1:123456789012:channel/ch-000001"
	s.channels[channelArn] = map[string]any{
		"arn":  channelArn,
		"type": "EMAIL",
	}

	s.channelAssociations[channelArn] = cfgArn

	s.notificationHubs["us-east-1"] = map[string]any{
		"notificationHubRegion": "us-east-1",
		"statusSummary": map[string]any{
			"status": "REGISTERED",
		},
	}
	s.tags[cfgArn] = map[string]string{"stackyard": "true"}

	return s
}

func (s *userNotificationsStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ map[string][]string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "AssociateChannel":
		channelArn := userNotificationsPathParam(pathParams, "arn", s.firstChannelArnLocked())
		if channelArn == "" {
			channelArn = s.newArnLocked("channel", "ch")
		}
		cfgArn := userNotificationsPayloadString(payload, "notificationConfigurationArn", s.firstNotificationConfigurationArnLocked())
		cfg := s.ensureNotificationConfigurationLocked(cfgArn)
		s.ensureChannelLocked(channelArn)
		cfgArn = userNotificationsMapString(cfg, "arn", cfgArn)
		s.channelAssociations[channelArn] = cfgArn
		return map[string]any{
			"channelArn":                   channelArn,
			"notificationConfigurationArn": cfgArn,
		}

	case "AssociateManagedNotificationAccountContact":
		contactID := userNotificationsPathParam(pathParams, "contactIdentifier", "contact-000001")
		s.managedAccountContacts[contactID] = true
		return map[string]any{"contactIdentifier": contactID, "associated": true}

	case "AssociateManagedNotificationAdditionalChannel":
		channelArn := userNotificationsPathParam(pathParams, "channelArn", s.firstChannelArnLocked())
		if channelArn == "" {
			channelArn = s.newArnLocked("channel", "ch")
		}
		s.ensureChannelLocked(channelArn)
		s.managedAdditionalChannels[channelArn] = true
		return map[string]any{"channelArn": channelArn, "associated": true}

	case "AssociateOrganizationalUnit":
		ou := userNotificationsPathParam(pathParams, "organizationalUnitId", "ou-0000000000")
		s.organizationalUnitAssociate[ou] = true
		return map[string]any{"organizationalUnitId": ou, "associated": true}

	case "CreateEventRule":
		arn := userNotificationsPayloadString(payload, "arn", "")
		if arn == "" {
			arn = s.newArnLocked("event-rule", "er")
		}
		rule := s.ensureEventRuleLocked(arn)
		userNotificationsMergeMap(rule, payload)
		return map[string]any{"arn": rule["arn"], "eventRule": userNotificationsCloneMap(rule)}

	case "CreateNotificationConfiguration":
		arn := userNotificationsPayloadString(payload, "arn", "")
		if arn == "" {
			arn = s.newArnLocked("notification-configuration", "nc")
		}
		cfg := s.ensureNotificationConfigurationLocked(arn)
		userNotificationsMergeMap(cfg, payload)
		return map[string]any{"arn": cfg["arn"], "notificationConfiguration": userNotificationsCloneMap(cfg)}

	case "DeleteEventRule":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstEventRuleArnLocked())
		delete(s.eventRules, arn)
		return map[string]any{}

	case "DeleteNotificationConfiguration":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		delete(s.notificationConfigurations, arn)
		delete(s.tags, arn)
		for channelArn, cfgArn := range s.channelAssociations {
			if cfgArn == arn {
				delete(s.channelAssociations, channelArn)
			}
		}
		return map[string]any{}

	case "DeregisterNotificationHub":
		region := userNotificationsPathParam(pathParams, "notificationHubRegion", "us-east-1")
		delete(s.notificationHubs, region)
		return map[string]any{}

	case "DisableNotificationsAccessForOrganization":
		s.orgAccessEnabled = false
		return map[string]any{}

	case "DisassociateChannel":
		channelArn := userNotificationsPathParam(pathParams, "arn", s.firstChannelArnLocked())
		delete(s.channelAssociations, channelArn)
		return map[string]any{}

	case "DisassociateManagedNotificationAccountContact":
		contactID := userNotificationsPathParam(pathParams, "contactIdentifier", "contact-000001")
		delete(s.managedAccountContacts, contactID)
		return map[string]any{}

	case "DisassociateManagedNotificationAdditionalChannel":
		channelArn := userNotificationsPathParam(pathParams, "channelArn", s.firstChannelArnLocked())
		delete(s.managedAdditionalChannels, channelArn)
		return map[string]any{}

	case "DisassociateOrganizationalUnit":
		ou := userNotificationsPathParam(pathParams, "organizationalUnitId", "ou-0000000000")
		delete(s.organizationalUnitAssociate, ou)
		return map[string]any{}

	case "EnableNotificationsAccessForOrganization":
		s.orgAccessEnabled = true
		return map[string]any{}

	case "GetEventRule":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstEventRuleArnLocked())
		rule := s.ensureEventRuleLocked(arn)
		return map[string]any{"eventRule": userNotificationsCloneMap(rule)}

	case "GetManagedNotificationChildEvent":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstManagedChildEventArnLocked())
		child := s.ensureManagedChildEventLocked(arn, s.firstManagedNotificationEventArnLocked())
		return map[string]any{"managedNotificationChildEvent": userNotificationsCloneMap(child)}

	case "GetManagedNotificationConfiguration":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstManagedConfigurationArnLocked())
		cfg := s.ensureManagedConfigurationLocked(arn)
		return map[string]any{"managedNotificationConfiguration": userNotificationsCloneMap(cfg)}

	case "GetManagedNotificationEvent":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstManagedNotificationEventArnLocked())
		event := s.ensureManagedNotificationEventLocked(arn)
		return map[string]any{"managedNotificationEvent": userNotificationsCloneMap(event)}

	case "GetNotificationConfiguration":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		cfg := s.ensureNotificationConfigurationLocked(arn)
		return map[string]any{"notificationConfiguration": userNotificationsCloneMap(cfg)}

	case "GetNotificationEvent":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationEventArnLocked())
		event := s.ensureNotificationEventLocked(arn)
		return map[string]any{"notificationEvent": userNotificationsCloneMap(event)}

	case "GetNotificationsAccessForOrganization":
		status := "DISABLED"
		if s.orgAccessEnabled {
			status = "ENABLED"
		}
		return map[string]any{
			"notificationsAccessForOrganization": map[string]any{
				"status": status,
			},
		}

	case "ListChannels":
		return map[string]any{"channels": userNotificationsSortedValues(s.channels), "nextToken": ""}

	case "ListEventRules":
		return map[string]any{"eventRules": userNotificationsSortedValues(s.eventRules), "nextToken": ""}

	case "ListManagedNotificationChannelAssociations":
		out := make([]any, 0, len(s.channelAssociations)+len(s.managedAdditionalChannels))
		seen := map[string]bool{}
		for channelArn, cfgArn := range s.channelAssociations {
			out = append(out, map[string]any{
				"channelArn":                   channelArn,
				"notificationConfigurationArn": cfgArn,
				"associationType":              "NOTIFICATION_CONFIGURATION",
			})
			seen[channelArn] = true
		}
		for channelArn := range s.managedAdditionalChannels {
			if seen[channelArn] {
				continue
			}
			out = append(out, map[string]any{
				"channelArn":      channelArn,
				"associationType": "MANAGED_NOTIFICATION",
			})
		}
		userNotificationsSortSliceByArn(out)
		return map[string]any{"managedNotificationChannelAssociations": out, "nextToken": ""}

	case "ListManagedNotificationChildEvents":
		parentArn := userNotificationsPathParam(pathParams, "aggregateManagedNotificationEventArn", "")
		out := make([]any, 0)
		for _, child := range s.managedChildEvents {
			if parentArn != "" && userNotificationsMapString(child, "parentArn", "") != parentArn {
				continue
			}
			out = append(out, userNotificationsCloneMap(child))
		}
		userNotificationsSortSliceByArn(out)
		return map[string]any{"managedNotificationChildEvents": out, "nextToken": ""}

	case "ListManagedNotificationConfigurations":
		return map[string]any{"managedNotificationConfigurations": userNotificationsSortedValues(s.managedConfigurations), "nextToken": ""}

	case "ListManagedNotificationEvents":
		return map[string]any{"managedNotificationEvents": userNotificationsSortedValues(s.managedNotificationEvents), "nextToken": ""}

	case "ListMemberAccounts":
		return map[string]any{"memberAccounts": userNotificationsCloneMapSlice(s.memberAccounts), "nextToken": ""}

	case "ListNotificationConfigurations":
		return map[string]any{"notificationConfigurations": userNotificationsSortedValues(s.notificationConfigurations), "nextToken": ""}

	case "ListNotificationEvents":
		return map[string]any{"notificationEvents": userNotificationsSortedValues(s.notificationEvents), "nextToken": ""}

	case "ListNotificationHubs":
		return map[string]any{"notificationHubs": userNotificationsSortedValues(s.notificationHubs), "nextToken": ""}

	case "ListOrganizationalUnits":
		keys := make([]string, 0, len(s.organizationalUnitAssociate))
		for ou, assoc := range s.organizationalUnitAssociate {
			if assoc {
				keys = append(keys, ou)
			}
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, ou := range keys {
			out = append(out, map[string]any{"organizationalUnitId": ou})
		}
		return map[string]any{"organizationalUnits": out, "nextToken": ""}

	case "ListTagsForResource":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		tagMap := map[string]any{}
		for key, val := range s.tags[arn] {
			tagMap[key] = val
		}
		return map[string]any{"tags": tagMap}

	case "RegisterNotificationHub":
		region := userNotificationsPayloadString(payload, "notificationHubRegion", "us-east-1")
		s.notificationHubs[region] = map[string]any{
			"notificationHubRegion": region,
			"statusSummary": map[string]any{
				"status": "REGISTERED",
			},
		}
		return map[string]any{"notificationHubRegion": region}

	case "TagResource":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{}
		}
		for key, val := range userNotificationsPayloadStringMap(payload, "tags") {
			s.tags[arn][key] = val
		}
		return map[string]any{}

	case "UntagResource":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		tagKeys := userNotificationsPayloadStringSlice(payload, "tagKeys")
		for _, key := range tagKeys {
			delete(s.tags[arn], key)
		}
		return map[string]any{}

	case "UpdateEventRule":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstEventRuleArnLocked())
		rule := s.ensureEventRuleLocked(arn)
		userNotificationsMergeMap(rule, payload)
		return map[string]any{"eventRule": userNotificationsCloneMap(rule)}

	case "UpdateNotificationConfiguration":
		arn := userNotificationsPathParam(pathParams, "arn", s.firstNotificationConfigurationArnLocked())
		cfg := s.ensureNotificationConfigurationLocked(arn)
		userNotificationsMergeMap(cfg, payload)
		return map[string]any{"notificationConfiguration": userNotificationsCloneMap(cfg)}
	}

	return map[string]any{}
}

func (s *userNotificationsStore) newArnLocked(resourceType, prefix string) string {
	s.counter++
	return "arn:aws:notifications:us-east-1:123456789012:" + resourceType + "/" + prefix + "-" + strconv.Itoa(s.counter)
}

func (s *userNotificationsStore) ensureChannelLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("channel", "ch")
	}
	if ch, ok := s.channels[arn]; ok {
		return ch
	}
	ch := map[string]any{
		"arn":  arn,
		"type": "EMAIL",
	}
	s.channels[arn] = ch
	return ch
}

func (s *userNotificationsStore) ensureNotificationConfigurationLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("notification-configuration", "nc")
	}
	if cfg, ok := s.notificationConfigurations[arn]; ok {
		return cfg
	}
	cfg := map[string]any{
		"arn":    arn,
		"name":   "stackyard-notification-configuration",
		"status": "ACTIVE",
	}
	s.notificationConfigurations[arn] = cfg
	return cfg
}

func (s *userNotificationsStore) ensureEventRuleLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("event-rule", "er")
	}
	if rule, ok := s.eventRules[arn]; ok {
		return rule
	}
	rule := map[string]any{
		"arn":    arn,
		"name":   "stackyard-event-rule",
		"status": "ACTIVE",
		"statusSummary": map[string]any{
			"status": "ACTIVE",
		},
	}
	s.eventRules[arn] = rule
	return rule
}

func (s *userNotificationsStore) ensureNotificationEventLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("notification-event", "ne")
	}
	if evt, ok := s.notificationEvents[arn]; ok {
		return evt
	}
	evt := map[string]any{
		"arn":       arn,
		"source":    "stackyard",
		"eventType": "NotificationEvent",
	}
	s.notificationEvents[arn] = evt
	return evt
}

func (s *userNotificationsStore) ensureManagedNotificationEventLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("managed-notification-event", "mne")
	}
	if evt, ok := s.managedNotificationEvents[arn]; ok {
		return evt
	}
	evt := map[string]any{
		"arn":       arn,
		"source":    "aws.health",
		"eventType": "ManagedNotificationEvent",
	}
	s.managedNotificationEvents[arn] = evt
	return evt
}

func (s *userNotificationsStore) ensureManagedChildEventLocked(arn, parentArn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("managed-notification-child-event", "mnce")
	}
	if evt, ok := s.managedChildEvents[arn]; ok {
		return evt
	}
	if strings.TrimSpace(parentArn) == "" {
		parentArn = s.firstManagedNotificationEventArnLocked()
	}
	evt := map[string]any{
		"arn":          arn,
		"parentArn":    parentArn,
		"source":       "aws.health",
		"eventType":    "ManagedNotificationChildEvent",
		"notification": "stackyard managed child event",
	}
	s.managedChildEvents[arn] = evt
	return evt
}

func (s *userNotificationsStore) ensureManagedConfigurationLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newArnLocked("managed-notification-configuration", "mnc")
	}
	if cfg, ok := s.managedConfigurations[arn]; ok {
		return cfg
	}
	cfg := map[string]any{
		"arn":    arn,
		"name":   "stackyard-managed-notification-configuration",
		"status": "ACTIVE",
	}
	s.managedConfigurations[arn] = cfg
	return cfg
}

func (s *userNotificationsStore) firstChannelArnLocked() string {
	return userNotificationsFirstKey(s.channels)
}

func (s *userNotificationsStore) firstNotificationConfigurationArnLocked() string {
	return userNotificationsFirstKey(s.notificationConfigurations)
}

func (s *userNotificationsStore) firstEventRuleArnLocked() string {
	return userNotificationsFirstKey(s.eventRules)
}

func (s *userNotificationsStore) firstNotificationEventArnLocked() string {
	return userNotificationsFirstKey(s.notificationEvents)
}

func (s *userNotificationsStore) firstManagedNotificationEventArnLocked() string {
	return userNotificationsFirstKey(s.managedNotificationEvents)
}

func (s *userNotificationsStore) firstManagedChildEventArnLocked() string {
	return userNotificationsFirstKey(s.managedChildEvents)
}

func (s *userNotificationsStore) firstManagedConfigurationArnLocked() string {
	return userNotificationsFirstKey(s.managedConfigurations)
}

func userNotificationsFirstKey(m map[string]map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func userNotificationsPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return strings.TrimSpace(def)
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return strings.TrimSpace(def)
	}
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return strings.TrimSpace(def)
		}
		return trimmed
	default:
		return strings.TrimSpace(def)
	}
}

func userNotificationsPayloadStringMap(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return out
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for mk, mv := range m {
		if str, ok := mv.(string); ok {
			out[mk] = str
		}
	}
	return out
}

func userNotificationsPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		str, ok := item.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)
		if str != "" {
			out = append(out, str)
		}
	}
	return out
}

func userNotificationsPathParam(params map[string]string, key, def string) string {
	if params == nil {
		return strings.TrimSpace(def)
	}
	value, ok := params[key]
	if !ok {
		return strings.TrimSpace(def)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(def)
	}
	return value
}

func userNotificationsMapString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	value, ok := m[key]
	if !ok || value == nil {
		return def
	}
	str, ok := value.(string)
	if !ok {
		return def
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return def
	}
	return str
}

func userNotificationsMergeMap(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		dst[key] = userNotificationsCloneValue(value)
	}
}

func userNotificationsSortedValues(m map[string]map[string]any) []any {
	if len(m) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, userNotificationsCloneMap(m[key]))
	}
	return out
}

func userNotificationsSortSliceByArn(items []any) {
	sort.Slice(items, func(i, j int) bool {
		li, _ := items[i].(map[string]any)
		lj, _ := items[j].(map[string]any)
		return userNotificationsMapString(li, "arn", "") < userNotificationsMapString(lj, "arn", "")
	})
}

func userNotificationsCloneMapSlice(in []map[string]any) []any {
	if len(in) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, userNotificationsCloneMap(item))
	}
	return out
}

func userNotificationsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = userNotificationsCloneValue(value)
	}
	return out
}

func userNotificationsCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return userNotificationsCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = userNotificationsCloneValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
