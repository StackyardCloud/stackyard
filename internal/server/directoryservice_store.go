package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type directoryServiceStore struct {
	mu          sync.Mutex
	nextID      int64
	tags        map[string]map[string]string
	directories map[string]map[string]any
	snapshots   map[string]map[string]any
	trusts      map[string]map[string]any
}

func newDirectoryServiceStore() *directoryServiceStore {
	directoryID := "d-0000000000"
	directoryARN := directoryServiceDirectoryARN(directoryID)
	return &directoryServiceStore{
		nextID: 1,
		tags: map[string]map[string]string{
			directoryID:  {"seed": "true"},
			directoryARN: {"seed": "true"},
		},
		directories: map[string]map[string]any{
			directoryID: {
				"DirectoryId":   directoryID,
				"Name":          "stackyard.local",
				"Stage":         "Active",
				"Type":          "SimpleAD",
				"DirectorySize": "Small",
				"Alias":         "stackyard",
			},
		},
		snapshots: map[string]map[string]any{},
		trusts:    map[string]map[string]any{},
	}
}

func (s *directoryServiceStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateDirectory", "CreateMicrosoftAD", "ConnectDirectory":
		directoryID := fmt.Sprintf("d-%010d", s.nextID)
		s.nextID++
		entry := map[string]any{
			"DirectoryId":   directoryID,
			"Name":          directoryServicePayloadString(payload, "Name", "stackyard.local"),
			"Stage":         "Requested",
			"Type":          "SimpleAD",
			"DirectorySize": directoryServicePayloadString(payload, "DirectorySize", "Small"),
		}
		s.directories[directoryID] = entry
		s.tags[directoryID] = map[string]string{"seed": "false"}
		s.tags[directoryServiceDirectoryARN(directoryID)] = map[string]string{"seed": "false"}
		return map[string]any{"DirectoryId": directoryID}
	case "DeleteDirectory":
		directoryID := directoryServicePayloadString(payload, "DirectoryId", "")
		if directoryID != "" {
			delete(s.directories, directoryID)
			delete(s.tags, directoryID)
			delete(s.tags, directoryServiceDirectoryARN(directoryID))
		}
		return map[string]any{"DirectoryId": directoryID}
	case "DescribeDirectories":
		return map[string]any{"DirectoryDescriptions": directoryServiceSortedMapValues(s.directories)}
	case "CreateSnapshot":
		snapshotID := fmt.Sprintf("s-%010d", s.nextID)
		s.nextID++
		s.snapshots[snapshotID] = map[string]any{
			"SnapshotId": snapshotID,
			"Status":     "Completed",
		}
		return map[string]any{"SnapshotId": snapshotID}
	case "DescribeSnapshots":
		return map[string]any{"Snapshots": directoryServiceSortedMapValues(s.snapshots)}
	case "GetSnapshotLimits":
		return map[string]any{
			"SnapshotLimits": map[string]any{
				"ManualSnapshotsLimit":        10,
				"ManualSnapshotsCurrentCount": len(s.snapshots),
				"ManualSnapshotsLimitReached": false,
			},
		}
	case "CreateTrust":
		trustID := fmt.Sprintf("t-%010d", s.nextID)
		s.nextID++
		s.trusts[trustID] = map[string]any{
			"TrustId": trustID,
			"State":   "Verified",
		}
		return map[string]any{"TrustId": trustID}
	case "DescribeTrusts":
		return map[string]any{"Trusts": directoryServiceSortedMapValues(s.trusts)}
	case "GetDirectoryLimits":
		return map[string]any{
			"DirectoryLimits": map[string]any{
				"CloudOnlyDirectoriesLimit":        10,
				"CloudOnlyDirectoriesCurrentCount": len(s.directories),
				"CloudOnlyDirectoriesLimitReached": false,
			},
		}
	case "ListTagsForResource":
		resourceID := directoryServicePayloadString(payload, "ResourceId", "d-0000000000")
		if resourceID == "" {
			resourceID = "d-0000000000"
		}
		tags := s.tags[resourceID]
		if tags == nil {
			tags = s.tags[directoryServiceDirectoryARN(resourceID)]
		}
		return map[string]any{"Tags": directoryServiceTagsToList(tags)}
	case "AddTagsToResource":
		resourceID := directoryServicePayloadString(payload, "ResourceId", "d-0000000000")
		if _, ok := s.tags[resourceID]; !ok {
			s.tags[resourceID] = map[string]string{}
		}
		for k, v := range directoryServiceTagsFromAny(payload["Tags"]) {
			s.tags[resourceID][k] = v
		}
		return map[string]any{}
	case "RemoveTagsFromResource":
		resourceID := directoryServicePayloadString(payload, "ResourceId", "d-0000000000")
		for _, key := range directoryServicePayloadStringSlice(payload, "TagKeys") {
			delete(s.tags[resourceID], key)
		}
		return map[string]any{}
	case "ListCertificates":
		return map[string]any{"CertificatesInfo": []any{}}
	case "ListIpRoutes":
		return map[string]any{"IpRoutesInfo": []any{}}
	case "ListLogSubscriptions":
		return map[string]any{"LogSubscriptions": []any{}}
	case "ListSchemaExtensions":
		return map[string]any{"SchemaExtensionsInfo": []any{}}
	case "DescribeEventTopics":
		return map[string]any{"EventTopics": []any{}}
	case "DescribeRegions":
		return map[string]any{"RegionsDescription": []any{}}
	case "DescribeSettings":
		return map[string]any{"DirectoryId": directoryServicePayloadString(payload, "DirectoryId", "d-0000000000"), "SettingEntries": []any{}}
	case "DescribeDomainControllers":
		return map[string]any{"DomainControllers": []any{}}
	case "DescribeConditionalForwarders":
		return map[string]any{"ConditionalForwarders": []any{}}
	case "DescribeClientAuthenticationSettings":
		return map[string]any{"ClientAuthenticationSettingsInfo": []any{}}
	case "DescribeLDAPSSettings":
		return map[string]any{"LDAPSSettingsInfo": []any{}}
	case "DescribeSharedDirectories":
		return map[string]any{"SharedDirectories": []any{}}
	case "DescribeCertificate":
		return map[string]any{"Certificate": map[string]any{}}
	case "DescribeDirectoryDataAccess":
		return map[string]any{"DirectoryDataAccessDescription": map[string]any{}}
	case "DescribeUpdateDirectory":
		return map[string]any{"UpdateActivities": []any{}}
	case "CreateAlias", "CreateComputer", "CreateConditionalForwarder", "CreateLogSubscription", "DeleteConditionalForwarder", "DeleteLogSubscription", "DeleteSnapshot", "DeleteTrust", "DeregisterCertificate", "DeregisterEventTopic", "DisableClientAuthentication", "DisableDirectoryDataAccess", "DisableLDAPS", "DisableRadius", "DisableSso", "EnableClientAuthentication", "EnableDirectoryDataAccess", "EnableLDAPS", "EnableRadius", "EnableSso", "RegisterCertificate", "RegisterEventTopic", "ResetUserPassword", "StartSchemaExtension", "CancelSchemaExtension", "VerifyTrust", "UpdateConditionalForwarder", "UpdateDirectorySetup", "UpdateNumberOfDomainControllers", "UpdateRadius", "UpdateSettings", "UpdateTrust", "AcceptSharedDirectory", "RejectSharedDirectory", "ShareDirectory", "UnshareDirectory", "AddIpRoutes", "RemoveIpRoutes", "AddRegion", "RemoveRegion", "RestoreFromSnapshot":
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func directoryServiceDirectoryARN(directoryID string) string {
	id := strings.TrimSpace(directoryID)
	if id == "" {
		id = "d-0000000000"
	}
	return fmt.Sprintf("arn:aws:ds:us-east-1:123456789012:directory/%s", id)
}

func directoryServicePayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return def
}

func directoryServicePayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	var raw any
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			raw = v
			break
		}
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func directoryServiceTagsFromAny(raw any) map[string]string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(directoryServicePayloadString(m, "Key", ""))
		if key == "" {
			continue
		}
		out[key] = directoryServicePayloadString(m, "Value", "")
	}
	return out
}

func directoryServiceTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
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

func directoryServiceSortedMapValues(m map[string]map[string]any) []any {
	if len(m) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}
