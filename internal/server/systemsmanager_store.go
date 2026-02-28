package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type systemsManagerStore struct {
	mu              sync.Mutex
	nextID          int64
	parameters      map[string]string
	documents       map[string]map[string]any
	serviceSettings map[string]map[string]any
	commands        map[string]map[string]any
	sessions        map[string]map[string]any
	opsItems        map[string]map[string]any
	resourceTags    map[string]map[string]string
}

func newSystemsManagerStore() *systemsManagerStore {
	now := time.Now().UTC().Format(time.RFC3339)
	seedDocName := "AWS-RunShellScript"
	seedDoc := map[string]any{
		"Name":            seedDocName,
		"DisplayName":     seedDocName,
		"DocumentVersion": "1",
		"DefaultVersion":  "1",
		"Owner":           "Amazon",
		"CreatedDate":     now,
		"Status":          "Active",
		"DocumentType":    "Command",
		"DocumentFormat":  "JSON",
		"SchemaVersion":   "2.2",
	}
	seedCommandID := "cmd-000001"
	seedSessionID := "session-000001"
	seedOpsID := "oi-000001"
	settingID := "/ssm/managed-instance/default-ec2-instance-management-role"
	return &systemsManagerStore{
		nextID:     2,
		parameters: map[string]string{"/stackyard/hello": "world"},
		documents: map[string]map[string]any{
			seedDocName: seedDoc,
		},
		serviceSettings: map[string]map[string]any{
			settingID: {
				"SettingId":        settingID,
				"SettingValue":     "arn:aws:iam::123456789012:role/stackyard-ssm-managed-instance",
				"LastModifiedDate": now,
				"LastModifiedUser": "arn:aws:iam::123456789012:user/stackyard",
				"Status":           "Customized",
			},
		},
		commands: map[string]map[string]any{
			seedCommandID: {
				"CommandId":         seedCommandID,
				"DocumentName":      seedDocName,
				"Status":            "Success",
				"RequestedDateTime": now,
				"TargetCount":       1,
				"CompletedCount":    1,
			},
		},
		sessions: map[string]map[string]any{
			seedSessionID: {
				"SessionId":    seedSessionID,
				"Status":       "Connected",
				"StartDate":    now,
				"Target":       "i-00000000000000001",
				"Owner":        "stackyard",
				"DocumentName": "SSM-SessionManagerRunShell",
			},
		},
		opsItems: map[string]map[string]any{
			seedOpsID: {
				"OpsItemId":        seedOpsID,
				"Status":           "Open",
				"Source":           "Stackyard",
				"CreatedTime":      now,
				"LastModifiedTime": now,
				"Title":            "Seeded Stackyard ops item",
				"Priority":         3,
			},
		},
		resourceTags: map[string]map[string]string{
			"Document:AWS-RunShellScript": {"stackyard": "true"},
			"Parameter:/stackyard/hello":  {"stackyard": "true"},
		},
	}
}

func (s *systemsManagerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "ListDocuments":
		return map[string]any{"DocumentIdentifiers": s.documentIdentifiersLocked(), "NextToken": ""}
	case "DescribeDocument":
		name := systemsManagerPayloadString(payload, "Name", "AWS-RunShellScript")
		doc := s.documentByNameLocked(name)
		return map[string]any{"Document": systemsManagerCloneMap(doc)}
	case "GetServiceSetting":
		settingID := systemsManagerPayloadString(payload, "SettingId", "/ssm/managed-instance/default-ec2-instance-management-role")
		setting := s.serviceSettingByIDLocked(settingID)
		return map[string]any{"ServiceSetting": systemsManagerCloneMap(setting)}
	case "ListCommands":
		return map[string]any{"Commands": s.sortedCommandsLocked(), "NextToken": ""}
	case "DescribeSessions":
		return map[string]any{"Sessions": s.sortedSessionsLocked(), "NextToken": ""}
	case "SendCommand":
		commandID := fmt.Sprintf("cmd-%06d", s.nextIDLocked())
		command := map[string]any{
			"CommandId":         commandID,
			"DocumentName":      systemsManagerPayloadString(payload, "DocumentName", "AWS-RunShellScript"),
			"Status":            "Pending",
			"RequestedDateTime": now,
			"TargetCount":       1,
			"CompletedCount":    0,
		}
		s.commands[commandID] = command
		return map[string]any{"Command": systemsManagerCloneMap(command)}
	case "CancelCommand":
		commandID := systemsManagerPayloadString(payload, "CommandId", s.firstCommandIDLocked())
		if cmd, ok := s.commands[commandID]; ok {
			cmd["Status"] = "Cancelled"
		}
		return map[string]any{}
	case "StartSession":
		sessionID := fmt.Sprintf("session-%06d", s.nextIDLocked())
		session := map[string]any{
			"SessionId":    sessionID,
			"Status":       "Connected",
			"StartDate":    now,
			"Target":       systemsManagerPayloadString(payload, "Target", "i-00000000000000001"),
			"Owner":        "stackyard",
			"DocumentName": systemsManagerPayloadString(payload, "DocumentName", "SSM-SessionManagerRunShell"),
		}
		s.sessions[sessionID] = session
		return map[string]any{
			"SessionId":  sessionID,
			"TokenValue": fmt.Sprintf("token-%s", sessionID),
			"StreamUrl":  fmt.Sprintf("wss://ssmmessages.us-east-1.amazonaws.com/v1/data-channel/%s", sessionID),
		}
	case "ResumeSession":
		sessionID := systemsManagerPayloadString(payload, "SessionId", s.firstSessionIDLocked())
		if _, ok := s.sessions[sessionID]; !ok {
			s.sessions[sessionID] = map[string]any{"SessionId": sessionID, "Status": "Connected", "StartDate": now}
		}
		return map[string]any{
			"SessionId":  sessionID,
			"TokenValue": fmt.Sprintf("token-%s", sessionID),
			"StreamUrl":  fmt.Sprintf("wss://ssmmessages.us-east-1.amazonaws.com/v1/data-channel/%s", sessionID),
		}
	case "TerminateSession":
		sessionID := systemsManagerPayloadString(payload, "SessionId", s.firstSessionIDLocked())
		if session, ok := s.sessions[sessionID]; ok {
			session["Status"] = "Terminated"
		}
		return map[string]any{}
	case "PutParameter":
		name := systemsManagerPayloadString(payload, "Name", "/stackyard/generated")
		value := systemsManagerPayloadString(payload, "Value", "")
		s.parameters[name] = value
		return map[string]any{"Version": 1}
	case "DeleteParameter":
		name := systemsManagerPayloadString(payload, "Name", "")
		delete(s.parameters, name)
		return map[string]any{}
	case "DeleteParameters":
		names := systemsManagerPayloadStrings(payload, "Names")
		for _, name := range names {
			delete(s.parameters, name)
		}
		return map[string]any{"DeletedParameters": names, "InvalidParameters": []any{}}
	case "GetParameter":
		name := systemsManagerPayloadString(payload, "Name", "/stackyard/hello")
		value := s.parameters[name]
		if value == "" {
			value = "world"
		}
		return map[string]any{"Parameter": map[string]any{"Name": name, "Type": "String", "Value": value, "Version": 1}}
	case "GetParameters":
		names := systemsManagerPayloadStrings(payload, "Names")
		if len(names) == 0 {
			names = []string{"/stackyard/hello"}
		}
		parameters := make([]any, 0, len(names))
		for _, name := range names {
			value := s.parameters[name]
			if value == "" {
				value = "world"
			}
			parameters = append(parameters, map[string]any{"Name": name, "Type": "String", "Value": value, "Version": 1})
		}
		return map[string]any{"Parameters": parameters, "InvalidParameters": []any{}}
	case "GetParametersByPath":
		path := systemsManagerPayloadString(payload, "Path", "/stackyard")
		parameters := []any{}
		for name, value := range s.parameters {
			if strings.HasPrefix(name, path) {
				parameters = append(parameters, map[string]any{"Name": name, "Type": "String", "Value": value, "Version": 1})
			}
		}
		sort.Slice(parameters, func(i, j int) bool {
			a := parameters[i].(map[string]any)
			b := parameters[j].(map[string]any)
			return fmt.Sprintf("%v", a["Name"]) < fmt.Sprintf("%v", b["Name"])
		})
		return map[string]any{"Parameters": parameters, "NextToken": ""}
	case "CreateDocument":
		name := systemsManagerPayloadString(payload, "Name", fmt.Sprintf("StackyardDocument-%06d", s.nextIDLocked()))
		doc := map[string]any{
			"Name":            name,
			"DocumentVersion": "1",
			"DefaultVersion":  "1",
			"Owner":           "stackyard",
			"CreatedDate":     now,
			"Status":          "Active",
			"DocumentType":    systemsManagerPayloadString(payload, "DocumentType", "Command"),
			"DocumentFormat":  systemsManagerPayloadString(payload, "DocumentFormat", "JSON"),
		}
		s.documents[name] = doc
		return map[string]any{"DocumentDescription": systemsManagerCloneMap(doc)}
	case "UpdateDocument":
		name := systemsManagerPayloadString(payload, "Name", "AWS-RunShellScript")
		doc := s.documentByNameLocked(name)
		doc["DocumentVersion"] = "2"
		doc["LatestVersion"] = "2"
		doc["Status"] = "Updating"
		doc["UpdatedDate"] = now
		s.documents[name] = doc
		return map[string]any{"DocumentDescription": systemsManagerCloneMap(doc)}
	case "DeleteDocument":
		name := systemsManagerPayloadString(payload, "Name", "")
		delete(s.documents, name)
		return map[string]any{}
	case "CreateMaintenanceWindow":
		windowID := fmt.Sprintf("mw-%06d", s.nextIDLocked())
		return map[string]any{"WindowId": windowID}
	case "CreateOpsItem":
		opsID := fmt.Sprintf("oi-%06d", s.nextIDLocked())
		op := map[string]any{
			"OpsItemId":        opsID,
			"Status":           "Open",
			"Source":           systemsManagerPayloadString(payload, "Source", "Stackyard"),
			"CreatedTime":      now,
			"LastModifiedTime": now,
			"Title":            systemsManagerPayloadString(payload, "Title", "Stackyard OpsItem"),
			"Priority":         3,
		}
		s.opsItems[opsID] = op
		return map[string]any{"OpsItemId": opsID}
	case "UpdateOpsItem":
		opsID := systemsManagerPayloadString(payload, "OpsItemId", s.firstOpsItemIDLocked())
		if item, ok := s.opsItems[opsID]; ok {
			item["LastModifiedTime"] = now
			status := systemsManagerPayloadString(payload, "Status", "")
			if status != "" {
				item["Status"] = status
			}
		}
		return map[string]any{}
	case "GetOpsItem":
		opsID := systemsManagerPayloadString(payload, "OpsItemId", s.firstOpsItemIDLocked())
		if item, ok := s.opsItems[opsID]; ok {
			return map[string]any{"OpsItem": systemsManagerCloneMap(item)}
		}
		return map[string]any{"OpsItem": map[string]any{"OpsItemId": opsID, "Status": "Open", "CreatedTime": now}}
	case "AddTagsToResource":
		resourceKey := systemsManagerResourceKey(payload)
		s.applyTagsLocked(resourceKey, payload)
		return map[string]any{}
	case "RemoveTagsFromResource":
		resourceKey := systemsManagerResourceKey(payload)
		s.removeTagsLocked(resourceKey, payload)
		return map[string]any{}
	case "ListTagsForResource":
		resourceKey := systemsManagerResourceKey(payload)
		return map[string]any{"TagList": s.tagsListLocked(resourceKey)}
	}

	if strings.HasPrefix(action, "List") {
		key := systemsManagerListKey(action)
		return map[string]any{key: []any{}, "NextToken": ""}
	}

	if strings.HasPrefix(action, "Describe") {
		key := strings.TrimPrefix(action, "Describe")
		if key == "" {
			key = "Items"
		}
		if strings.HasSuffix(key, "s") || strings.HasSuffix(key, "List") {
			return map[string]any{key: []any{}, "NextToken": ""}
		}
		return map[string]any{key: map[string]any{}}
	}

	if strings.HasPrefix(action, "Get") {
		key := strings.TrimPrefix(action, "Get")
		if key == "" {
			key = "Result"
		}
		return map[string]any{key: map[string]any{}}
	}

	if strings.HasPrefix(action, "Create") {
		resource := strings.TrimPrefix(action, "Create")
		if resource == "" {
			resource = "Resource"
		}
		id := fmt.Sprintf("%s-%06d", strings.ToLower(resource), s.nextIDLocked())
		return map[string]any{resource + "Id": id}
	}

	return map[string]any{}
}

func (s *systemsManagerStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *systemsManagerStore) documentByNameLocked(name string) map[string]any {
	if doc, ok := s.documents[name]; ok {
		return systemsManagerCloneMap(doc)
	}
	return map[string]any{
		"Name":            name,
		"DocumentVersion": "1",
		"DefaultVersion":  "1",
		"Owner":           "stackyard",
		"Status":          "Active",
		"DocumentType":    "Command",
		"DocumentFormat":  "JSON",
	}
}

func (s *systemsManagerStore) documentIdentifiersLocked() []any {
	names := make([]string, 0, len(s.documents))
	for name := range s.documents {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		doc := s.documents[name]
		out = append(out, map[string]any{
			"Name":            name,
			"DisplayName":     systemsManagerPayloadString(doc, "DisplayName", name),
			"DocumentVersion": systemsManagerPayloadString(doc, "DocumentVersion", "1"),
			"Owner":           systemsManagerPayloadString(doc, "Owner", "stackyard"),
			"DocumentType":    systemsManagerPayloadString(doc, "DocumentType", "Command"),
			"DocumentFormat":  systemsManagerPayloadString(doc, "DocumentFormat", "JSON"),
		})
	}
	return out
}

func (s *systemsManagerStore) serviceSettingByIDLocked(settingID string) map[string]any {
	if setting, ok := s.serviceSettings[settingID]; ok {
		return systemsManagerCloneMap(setting)
	}
	return map[string]any{
		"SettingId":        settingID,
		"SettingValue":     "",
		"Status":           "Default",
		"LastModifiedDate": time.Now().UTC().Format(time.RFC3339),
		"LastModifiedUser": "arn:aws:iam::123456789012:user/stackyard",
	}
}

func (s *systemsManagerStore) sortedCommandsLocked() []any {
	ids := make([]string, 0, len(s.commands))
	for id := range s.commands {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, systemsManagerCloneMap(s.commands[id]))
	}
	return out
}

func (s *systemsManagerStore) sortedSessionsLocked() []any {
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, systemsManagerCloneMap(s.sessions[id]))
	}
	return out
}

func (s *systemsManagerStore) firstCommandIDLocked() string {
	ids := make([]string, 0, len(s.commands))
	for id := range s.commands {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "cmd-000001"
	}
	return ids[0]
}

func (s *systemsManagerStore) firstSessionIDLocked() string {
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "session-000001"
	}
	return ids[0]
}

func (s *systemsManagerStore) firstOpsItemIDLocked() string {
	ids := make([]string, 0, len(s.opsItems))
	for id := range s.opsItems {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "oi-000001"
	}
	return ids[0]
}

func systemsManagerListKey(action string) string {
	switch action {
	case "ListTagsForResource":
		return "TagList"
	case "ListDocuments":
		return "DocumentIdentifiers"
	default:
		key := strings.TrimPrefix(action, "List")
		if key == "" {
			return "Items"
		}
		return key
	}
}

func systemsManagerResourceKey(payload map[string]any) string {
	resourceType := systemsManagerPayloadString(payload, "ResourceType", "Resource")
	resourceID := systemsManagerPayloadString(payload, "ResourceId", "default")
	return strings.TrimSpace(resourceType) + ":" + strings.TrimSpace(resourceID)
}

func (s *systemsManagerStore) applyTagsLocked(resourceKey string, payload map[string]any) {
	if s.resourceTags[resourceKey] == nil {
		s.resourceTags[resourceKey] = map[string]string{}
	}
	tagsRaw, ok := systemsManagerPayloadValue(payload, "Tags")
	if !ok {
		return
	}
	tags, ok := tagsRaw.([]any)
	if !ok {
		return
	}
	for _, item := range tags {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := systemsManagerPayloadString(m, "Key", "")
		if k == "" {
			continue
		}
		s.resourceTags[resourceKey][k] = systemsManagerPayloadString(m, "Value", "")
	}
}

func (s *systemsManagerStore) removeTagsLocked(resourceKey string, payload map[string]any) {
	tagKeys := systemsManagerPayloadStrings(payload, "TagKeys")
	for _, key := range tagKeys {
		delete(s.resourceTags[resourceKey], key)
	}
}

func (s *systemsManagerStore) tagsListLocked(resourceKey string) []any {
	tags := s.resourceTags[resourceKey]
	if tags == nil {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

func systemsManagerPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v, true
		}
	}
	return nil, false
}

func systemsManagerPayloadString(payload map[string]any, key, fallback string) string {
	v, ok := systemsManagerPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "%!v(<nil>)" {
		return fallback
	}
	return s
}

func systemsManagerPayloadStrings(payload map[string]any, key string) []string {
	v, ok := systemsManagerPayloadValue(payload, key)
	if !ok {
		return []string{}
	}
	arr, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s := strings.TrimSpace(fmt.Sprintf("%v", item))
		if s == "" || s == "%!v(<nil>)" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func systemsManagerCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
