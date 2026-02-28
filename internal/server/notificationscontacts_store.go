package server

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type notificationsContactsStore struct {
	mu sync.Mutex

	counter int

	emailContacts   map[string]map[string]any
	activationCodes map[string]string
	tags            map[string]map[string]string
}

func newNotificationsContactsStore() *notificationsContactsStore {
	s := &notificationsContactsStore{
		counter:         1000,
		emailContacts:   map[string]map[string]any{},
		activationCodes: map[string]string{},
		tags:            map[string]map[string]string{},
	}

	arn := "arn:aws:notificationscontacts:us-east-1:123456789012:emailcontact/ec-000001"
	s.emailContacts[arn] = map[string]any{
		"arn":          arn,
		"name":         "stackyard-default-contact",
		"emailAddress": "stackyard@example.com",
		"status":       "ACTIVATED",
	}
	s.activationCodes[arn] = "123456"
	s.tags[arn] = map[string]string{"stackyard": "true"}

	return s
}

func (s *notificationsContactsStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	queryParams map[string][]string,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateEmailContact":
		arn := notificationsContactsPayloadString(payload, "arn", "")
		if arn == "" {
			arn = s.newContactArnLocked()
		}
		contact := s.ensureEmailContactLocked(arn)
		notificationsContactsMergeMap(contact, payload)
		if notificationsContactsMapString(contact, "name", "") == "" {
			contact["name"] = "stackyard-contact"
		}
		if notificationsContactsMapString(contact, "emailAddress", "") == "" {
			contact["emailAddress"] = "stackyard+" + strconv.Itoa(s.counter) + "@example.com"
		}
		if notificationsContactsMapString(contact, "status", "") == "" {
			contact["status"] = "PENDING_ACTIVATION"
		}
		if _, ok := s.activationCodes[arn]; !ok {
			s.activationCodes[arn] = "123456"
		}
		return map[string]any{
			"arn":          arn,
			"emailContact": notificationsContactsCloneMap(contact),
		}

	case "GetEmailContact":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		contact := s.ensureEmailContactLocked(arn)
		return map[string]any{"emailContact": notificationsContactsCloneMap(contact)}

	case "ListEmailContacts":
		return map[string]any{"emailContacts": notificationsContactsSortedValues(s.emailContacts), "nextToken": ""}

	case "SendActivationCode":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		contact := s.ensureEmailContactLocked(arn)
		code := notificationsContactsPayloadString(payload, "activationCode", "123456")
		s.activationCodes[arn] = code
		contact["status"] = "PENDING_ACTIVATION"
		return map[string]any{"arn": arn}

	case "ActivateEmailContact":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		code := notificationsContactsPathParam(pathParams, "code", "")
		contact := s.ensureEmailContactLocked(arn)
		if code == "" {
			code = notificationsContactsPayloadString(payload, "code", "")
		}
		if code == "" {
			code = notificationsContactsPayloadString(payload, "activationCode", "123456")
		}
		s.activationCodes[arn] = code
		contact["status"] = "ACTIVATED"
		return map[string]any{
			"arn":    arn,
			"status": "ACTIVATED",
		}

	case "DeleteEmailContact":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		delete(s.emailContacts, arn)
		delete(s.activationCodes, arn)
		delete(s.tags, arn)
		return map[string]any{}

	case "ListTagsForResource":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		tagMap := map[string]any{}
		for key, val := range s.tags[arn] {
			tagMap[key] = val
		}
		return map[string]any{"tags": tagMap}

	case "TagResource":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{}
		}
		for key, val := range notificationsContactsPayloadStringMap(payload, "tags") {
			s.tags[arn][key] = val
		}
		return map[string]any{}

	case "UntagResource":
		arn := notificationsContactsPathParam(pathParams, "arn", s.firstEmailContactArnLocked())
		tagKeys := notificationsContactsPayloadStringSlice(payload, "tagKeys")
		if len(tagKeys) == 0 {
			tagKeys = notificationsContactsQueryStringSlice(queryParams, "tagKeys")
		}
		for _, key := range tagKeys {
			delete(s.tags[arn], key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *notificationsContactsStore) newContactArnLocked() string {
	s.counter++
	return "arn:aws:notificationscontacts:us-east-1:123456789012:emailcontact/ec-" + strconv.Itoa(s.counter)
}

func (s *notificationsContactsStore) ensureEmailContactLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.newContactArnLocked()
	}
	if existing, ok := s.emailContacts[arn]; ok {
		return existing
	}
	contact := map[string]any{
		"arn":          arn,
		"name":         "stackyard-contact",
		"emailAddress": "stackyard@example.com",
		"status":       "PENDING_ACTIVATION",
	}
	s.emailContacts[arn] = contact
	if _, ok := s.activationCodes[arn]; !ok {
		s.activationCodes[arn] = "123456"
	}
	return contact
}

func (s *notificationsContactsStore) firstEmailContactArnLocked() string {
	return notificationsContactsFirstKey(s.emailContacts)
}

func notificationsContactsFirstKey(m map[string]map[string]any) string {
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

func notificationsContactsPathParam(params map[string]string, key, def string) string {
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

func notificationsContactsPayloadString(payload map[string]any, key, def string) string {
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

func notificationsContactsPayloadStringMap(payload map[string]any, key string) map[string]string {
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

func notificationsContactsPayloadStringSlice(payload map[string]any, key string) []string {
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

func notificationsContactsQueryStringSlice(query map[string][]string, key string) []string {
	if query == nil {
		return nil
	}
	values, ok := query[key]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				out = append(out, token)
			}
		}
	}
	return out
}

func notificationsContactsMapString(m map[string]any, key, def string) string {
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

func notificationsContactsMergeMap(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		dst[key] = notificationsContactsCloneValue(value)
	}
}

func notificationsContactsSortedValues(m map[string]map[string]any) []any {
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
		out = append(out, notificationsContactsCloneMap(m[key]))
	}
	return out
}

func notificationsContactsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = notificationsContactsCloneValue(value)
	}
	return out
}

func notificationsContactsCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return notificationsContactsCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = notificationsContactsCloneValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
