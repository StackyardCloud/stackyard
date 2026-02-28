package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type ramStore struct {
	mu           sync.Mutex
	nextID       int64
	shares       map[string]map[string]any
	invitations  map[string]map[string]any
	permissions  map[string]map[string]any
	versions     map[string][]map[string]any
	tags         map[string]map[string]string
	replaceWorks map[string]map[string]any
}

func newRAMStore() *ramStore {
	now := time.Now().UTC().Format(time.RFC3339)
	shareArn := "arn:aws:ram:us-east-1:123456789012:resource-share/rs-000001"
	invitationArn := "arn:aws:ram:us-east-1:123456789012:resource-share-invitation/rsi-000001"
	permissionArn := "arn:aws:ram:us-east-1:123456789012:permission/perm-000001"

	share := map[string]any{
		"resourceShareArn":        shareArn,
		"name":                    "stackyard-share",
		"owningAccountId":         "123456789012",
		"allowExternalPrincipals": true,
		"status":                  "ACTIVE",
		"featureSet":              "CREATED_FROM_POLICY",
		"creationTime":            now,
		"lastUpdatedTime":         now,
	}

	invitation := map[string]any{
		"resourceShareInvitationArn": invitationArn,
		"resourceShareArn":           shareArn,
		"resourceShareName":          "stackyard-share",
		"receiverAccountId":          "210987654321",
		"senderAccountId":            "123456789012",
		"invitationTimestamp":        now,
		"status":                     "PENDING",
	}

	permission := map[string]any{
		"arn":             permissionArn,
		"name":            "stackyard-permission",
		"resourceType":    "ec2:Subnet",
		"status":          "ATTACHED",
		"permissionType":  "CUSTOMER_MANAGED",
		"version":         "1",
		"defaultVersion":  "1",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}

	return &ramStore{
		nextID: 2,
		shares: map[string]map[string]any{
			shareArn: share,
		},
		invitations: map[string]map[string]any{
			invitationArn: invitation,
		},
		permissions: map[string]map[string]any{
			permissionArn: permission,
		},
		versions: map[string][]map[string]any{
			permissionArn: {
				{
					"version":        "1",
					"creationTime":   now,
					"defaultVersion": true,
				},
			},
		},
		tags: map[string]map[string]string{
			shareArn:      {"stackyard": "true"},
			permissionArn: {"stackyard": "true"},
		},
		replaceWorks: map[string]map[string]any{},
	}
}

func (s *ramStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "AcceptResourceShareInvitation":
		invArn := ramPayloadString(payload, "resourceShareInvitationArn", s.firstInvitationArnLocked())
		inv := s.ensureInvitationLocked(invArn)
		inv["status"] = "ACCEPTED"
		return map[string]any{"resourceShareInvitation": ramCloneMap(inv), "clientToken": "stackyard"}
	case "AssociateResourceShare":
		shareArn := ramPayloadString(payload, "resourceShareArn", s.firstShareArnLocked())
		assoc := map[string]any{
			"resourceShareArn": shareArn,
			"associatedEntity": ramPayloadString(payload, "principal", "123456789012"),
			"associationType":  "PRINCIPAL",
			"status":           "ASSOCIATED",
			"statusMessage":    "",
			"creationTime":     now,
			"lastUpdatedTime":  now,
			"external":         false,
		}
		return map[string]any{"resourceShareAssociations": []any{assoc}, "clientToken": "stackyard"}
	case "AssociateResourceSharePermission":
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "CreatePermission":
		id := s.nextIDLocked()
		arn := fmt.Sprintf("arn:aws:ram:us-east-1:123456789012:permission/perm-%06d", id)
		name := ramPayloadString(payload, "name", fmt.Sprintf("stackyard-permission-%06d", id))
		perm := map[string]any{
			"arn":             arn,
			"name":            name,
			"resourceType":    ramPayloadString(payload, "resourceType", "ec2:Subnet"),
			"status":          "ATTACHED",
			"permissionType":  "CUSTOMER_MANAGED",
			"version":         "1",
			"defaultVersion":  "1",
			"creationTime":    now,
			"lastUpdatedTime": now,
		}
		s.permissions[arn] = perm
		s.versions[arn] = []map[string]any{{"version": "1", "creationTime": now, "defaultVersion": true}}
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{"stackyard": "true"}
		}
		return map[string]any{"permission": ramCloneMap(perm), "clientToken": "stackyard"}
	case "CreatePermissionVersion":
		permArn := ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked())
		perm := s.ensurePermissionLocked(permArn)
		version := fmt.Sprintf("%d", len(s.versions[permArn])+1)
		entry := map[string]any{"version": version, "creationTime": now, "defaultVersion": false}
		s.versions[permArn] = append(s.versions[permArn], entry)
		perm["version"] = version
		perm["lastUpdatedTime"] = now
		return map[string]any{"permission": ramCloneMap(perm), "clientToken": "stackyard"}
	case "CreateResourceShare":
		id := s.nextIDLocked()
		arn := fmt.Sprintf("arn:aws:ram:us-east-1:123456789012:resource-share/rs-%06d", id)
		name := ramPayloadString(payload, "name", fmt.Sprintf("stackyard-share-%06d", id))
		share := map[string]any{
			"resourceShareArn":        arn,
			"name":                    name,
			"owningAccountId":         "123456789012",
			"allowExternalPrincipals": ramPayloadBool(payload, "allowExternalPrincipals", true),
			"status":                  "ACTIVE",
			"featureSet":              "CREATED_FROM_POLICY",
			"creationTime":            now,
			"lastUpdatedTime":         now,
		}
		s.shares[arn] = share
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		s.applyTagsLocked(arn, payload)
		share["tags"] = s.tagsListLocked(arn)
		return map[string]any{"resourceShare": ramCloneMap(share), "clientToken": "stackyard"}
	case "DeletePermission":
		delete(s.permissions, ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked()))
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "DeletePermissionVersion":
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "DeleteResourceShare":
		delete(s.shares, ramPayloadString(payload, "resourceShareArn", s.firstShareArnLocked()))
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "DisassociateResourceShare":
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "DisassociateResourceSharePermission":
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "EnableSharingWithAwsOrganization":
		return map[string]any{"returnValue": true}
	case "GetPermission":
		permArn := ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked())
		return map[string]any{"permission": ramCloneMap(s.ensurePermissionLocked(permArn))}
	case "GetResourcePolicies":
		return map[string]any{
			"policies":  []any{"{\"Version\":\"2012-10-17\",\"Statement\":[]}"},
			"nextToken": "",
		}
	case "GetResourceShareAssociations":
		shareArn := ramPayloadString(payload, "resourceShareArn", s.firstShareArnLocked())
		return map[string]any{
			"resourceShareAssociations": []any{
				map[string]any{
					"resourceShareArn": shareArn,
					"associatedEntity": "123456789012",
					"associationType":  "PRINCIPAL",
					"status":           "ASSOCIATED",
					"creationTime":     now,
					"lastUpdatedTime":  now,
					"external":         false,
				},
			},
			"nextToken": "",
		}
	case "GetResourceShareInvitations":
		return map[string]any{"resourceShareInvitations": s.sortedInvitationsLocked(), "nextToken": ""}
	case "GetResourceShares":
		return map[string]any{"resourceShares": s.sortedSharesLocked(), "nextToken": ""}
	case "ListPendingInvitationResources":
		return map[string]any{
			"resources": []any{
				map[string]any{
					"arn":              "arn:aws:ec2:us-east-1:123456789012:subnet/subnet-000001",
					"type":             "ec2:Subnet",
					"resourceShareArn": ramPayloadString(payload, "resourceShareInvitationArn", s.firstShareArnLocked()),
				},
			},
			"nextToken": "",
		}
	case "ListPermissionAssociations":
		return map[string]any{
			"permissionAssociations": []any{
				map[string]any{
					"associatedEntity": "arn:aws:ram:us-east-1:123456789012:resource-share/rs-000001",
					"resourceType":     "ec2:Subnet",
					"status":           "ASSOCIATED",
					"external":         false,
					"lastUpdatedTime":  now,
				},
			},
			"nextToken": "",
		}
	case "ListPermissionVersions":
		permArn := ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked())
		versions := make([]any, 0, len(s.versions[permArn]))
		for _, version := range s.versions[permArn] {
			versions = append(versions, ramCloneMap(version))
		}
		return map[string]any{"permissionVersions": versions, "nextToken": ""}
	case "ListPermissions":
		return map[string]any{"permissions": s.sortedPermissionSummariesLocked(), "nextToken": ""}
	case "ListPrincipals":
		return map[string]any{
			"principals": []any{
				map[string]any{
					"id":               "123456789012",
					"resourceShareArn": s.firstShareArnLocked(),
					"creationTime":     now,
					"lastUpdatedTime":  now,
					"external":         false,
				},
			},
			"nextToken": "",
		}
	case "ListReplacePermissionAssociationsWork":
		return map[string]any{"replacePermissionAssociationsWorks": s.sortedReplaceWorksLocked(), "nextToken": ""}
	case "ListResourceSharePermissions":
		return map[string]any{"permissions": s.sortedPermissionSummariesLocked(), "nextToken": ""}
	case "ListResourceTypes":
		return map[string]any{"resourceTypes": []any{"ec2:Subnet", "ec2:VPC", "s3:Bucket"}, "nextToken": ""}
	case "ListResources":
		return map[string]any{
			"resources": []any{
				map[string]any{
					"arn":                 "arn:aws:ec2:us-east-1:123456789012:subnet/subnet-000001",
					"type":                "ec2:Subnet",
					"resourceShareArn":    s.firstShareArnLocked(),
					"status":              "AVAILABLE",
					"statusMessage":       "",
					"creationTime":        now,
					"lastUpdatedTime":     now,
					"resourceGroupArn":    "",
					"resourceRegionScope": "REGIONAL",
				},
			},
			"nextToken": "",
		}
	case "ListSourceAssociations":
		return map[string]any{
			"sourceAssociations": []any{
				map[string]any{
					"sourceArn":        "arn:aws:iam::123456789012:policy/stackyard-ram-policy",
					"associatedEntity": s.firstShareArnLocked(),
					"associationType":  "RESOURCE_SHARE",
					"status":           "ASSOCIATED",
					"lastUpdatedTime":  now,
				},
			},
			"nextToken": "",
		}
	case "PromotePermissionCreatedFromPolicy":
		permArn := ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked())
		perm := s.ensurePermissionLocked(permArn)
		perm["permissionType"] = "CUSTOMER_MANAGED"
		perm["lastUpdatedTime"] = now
		return map[string]any{"permission": ramCloneMap(perm), "clientToken": "stackyard"}
	case "PromoteResourceShareCreatedFromPolicy":
		shareArn := ramPayloadString(payload, "resourceShareArn", s.firstShareArnLocked())
		share := s.ensureShareLocked(shareArn)
		share["featureSet"] = "PROMOTING_TO_STANDARD"
		share["lastUpdatedTime"] = now
		return map[string]any{"resourceShare": ramCloneMap(share), "clientToken": "stackyard"}
	case "RejectResourceShareInvitation":
		invArn := ramPayloadString(payload, "resourceShareInvitationArn", s.firstInvitationArnLocked())
		inv := s.ensureInvitationLocked(invArn)
		inv["status"] = "REJECTED"
		return map[string]any{"resourceShareInvitation": ramCloneMap(inv), "clientToken": "stackyard"}
	case "ReplacePermissionAssociations":
		workID := fmt.Sprintf("rpaw-%06d", s.nextIDLocked())
		workArn := fmt.Sprintf("arn:aws:ram:us-east-1:123456789012:replace-permission-associations-work/%s", workID)
		work := map[string]any{
			"id":                                   workID,
			"fromPermissionArn":                    ramPayloadString(payload, "fromPermissionArn", s.firstPermissionArnLocked()),
			"toPermissionArn":                      ramPayloadString(payload, "toPermissionArn", s.firstPermissionArnLocked()),
			"status":                               "IN_PROGRESS",
			"statusMessage":                        "",
			"creationTime":                         now,
			"lastUpdatedTime":                      now,
			"replacePermissionAssociationsWorkArn": workArn,
		}
		s.replaceWorks[workID] = work
		return map[string]any{"replacePermissionAssociationsWork": ramCloneMap(work), "clientToken": "stackyard"}
	case "SetDefaultPermissionVersion":
		permArn := ramPayloadString(payload, "permissionArn", s.firstPermissionArnLocked())
		version := ramPayloadString(payload, "permissionVersion", "1")
		perm := s.ensurePermissionLocked(permArn)
		perm["defaultVersion"] = version
		perm["lastUpdatedTime"] = now
		for _, v := range s.versions[permArn] {
			v["defaultVersion"] = fmt.Sprintf("%v", v["version"]) == version
		}
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "TagResource":
		arn := ramPayloadString(payload, "resourceArn", s.firstShareArnLocked())
		s.applyTagsLocked(arn, payload)
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "UntagResource":
		arn := ramPayloadString(payload, "resourceArn", s.firstShareArnLocked())
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		if raw, ok := payload["tagKeys"]; ok {
			if keys, ok := raw.([]any); ok {
				for _, key := range keys {
					delete(s.tags[arn], strings.TrimSpace(fmt.Sprintf("%v", key)))
				}
			}
		}
		return map[string]any{"returnValue": true, "clientToken": "stackyard"}
	case "UpdateResourceShare":
		shareArn := ramPayloadString(payload, "resourceShareArn", s.firstShareArnLocked())
		share := s.ensureShareLocked(shareArn)
		if name := ramPayloadString(payload, "name", ""); name != "" {
			share["name"] = name
		}
		share["allowExternalPrincipals"] = ramPayloadBool(payload, "allowExternalPrincipals", true)
		share["lastUpdatedTime"] = now
		share["tags"] = s.tagsListLocked(shareArn)
		return map[string]any{"resourceShare": ramCloneMap(share), "clientToken": "stackyard"}
	}

	return map[string]any{}
}

func (s *ramStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *ramStore) firstShareArnLocked() string {
	if len(s.shares) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.shares))
	for arn := range s.shares {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *ramStore) firstInvitationArnLocked() string {
	if len(s.invitations) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.invitations))
	for arn := range s.invitations {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *ramStore) firstPermissionArnLocked() string {
	if len(s.permissions) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.permissions))
	for arn := range s.permissions {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *ramStore) ensureShareLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.firstShareArnLocked()
	}
	if share := s.shares[arn]; share != nil {
		if s.tags[arn] != nil {
			share["tags"] = s.tagsListLocked(arn)
		}
		return share
	}
	now := time.Now().UTC().Format(time.RFC3339)
	share := map[string]any{
		"resourceShareArn":        arn,
		"name":                    "stackyard-share",
		"owningAccountId":         "123456789012",
		"allowExternalPrincipals": true,
		"status":                  "ACTIVE",
		"featureSet":              "CREATED_FROM_POLICY",
		"creationTime":            now,
		"lastUpdatedTime":         now,
	}
	s.shares[arn] = share
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true"}
	}
	share["tags"] = s.tagsListLocked(arn)
	return share
}

func (s *ramStore) ensureInvitationLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.firstInvitationArnLocked()
	}
	if inv := s.invitations[arn]; inv != nil {
		return inv
	}
	now := time.Now().UTC().Format(time.RFC3339)
	inv := map[string]any{
		"resourceShareInvitationArn": arn,
		"resourceShareArn":           s.firstShareArnLocked(),
		"resourceShareName":          "stackyard-share",
		"receiverAccountId":          "210987654321",
		"senderAccountId":            "123456789012",
		"invitationTimestamp":        now,
		"status":                     "PENDING",
	}
	s.invitations[arn] = inv
	return inv
}

func (s *ramStore) ensurePermissionLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = s.firstPermissionArnLocked()
	}
	if perm := s.permissions[arn]; perm != nil {
		return perm
	}
	now := time.Now().UTC().Format(time.RFC3339)
	perm := map[string]any{
		"arn":             arn,
		"name":            "stackyard-permission",
		"resourceType":    "ec2:Subnet",
		"status":          "ATTACHED",
		"permissionType":  "CUSTOMER_MANAGED",
		"version":         "1",
		"defaultVersion":  "1",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	s.permissions[arn] = perm
	s.versions[arn] = []map[string]any{{"version": "1", "creationTime": now, "defaultVersion": true}}
	return perm
}

func (s *ramStore) sortedSharesLocked() []any {
	keys := make([]string, 0, len(s.shares))
	for arn := range s.shares {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		share := s.ensureShareLocked(arn)
		out = append(out, ramCloneMap(share))
	}
	return out
}

func (s *ramStore) sortedInvitationsLocked() []any {
	keys := make([]string, 0, len(s.invitations))
	for arn := range s.invitations {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		out = append(out, ramCloneMap(s.invitations[arn]))
	}
	return out
}

func (s *ramStore) sortedPermissionSummariesLocked() []any {
	keys := make([]string, 0, len(s.permissions))
	for arn := range s.permissions {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		perm := s.permissions[arn]
		out = append(out, map[string]any{
			"arn":             perm["arn"],
			"name":            perm["name"],
			"resourceType":    perm["resourceType"],
			"status":          perm["status"],
			"permissionType":  perm["permissionType"],
			"version":         perm["version"],
			"defaultVersion":  perm["defaultVersion"],
			"creationTime":    perm["creationTime"],
			"lastUpdatedTime": perm["lastUpdatedTime"],
		})
	}
	return out
}

func (s *ramStore) sortedReplaceWorksLocked() []any {
	keys := make([]string, 0, len(s.replaceWorks))
	for id := range s.replaceWorks {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		out = append(out, ramCloneMap(s.replaceWorks[id]))
	}
	return out
}

func (s *ramStore) applyTagsLocked(arn string, payload map[string]any) {
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
	raw, ok := payload["tags"]
	if !ok {
		return
	}
	switch typed := raw.(type) {
	case map[string]any:
		for k, v := range typed {
			s.tags[arn][k] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case []any:
		for _, item := range typed {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := ramPayloadString(m, "key", "")
			if key == "" {
				continue
			}
			s.tags[arn][key] = ramPayloadString(m, "value", "")
		}
	}
}

func (s *ramStore) tagsListLocked(arn string) []any {
	t := s.tags[arn]
	if len(t) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(t))
	for key := range t {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"key": key, "value": t[key]})
	}
	return out
}

func ramPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "%!v(<nil>)" {
				return s
			}
		}
	}
	return fallback
}

func ramPayloadBool(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch typed := v.(type) {
		case bool:
			return typed
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(typed))
			if trimmed == "true" {
				return true
			}
			if trimmed == "false" {
				return false
			}
		}
	}
	return fallback
}

func ramCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = ramCloneAny(value)
	}
	return out
}

func ramCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ramCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ramCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
