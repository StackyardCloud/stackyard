package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type rolesAnywhereStore struct {
	mu sync.Mutex

	accountID string
	region    string
	nextID    int64

	profiles     map[string]map[string]any
	trustAnchors map[string]map[string]any
	crls         map[string]map[string]any
	subjects     map[string]map[string]any
	tags         map[string]map[string]string

	attributeMappings    map[string][]map[string]any
	notificationSettings map[string]any
}

func newRolesAnywhereStore() *rolesAnywhereStore {
	s := &rolesAnywhereStore{
		accountID:            "123456789012",
		region:               "us-east-1",
		nextID:               1,
		profiles:             map[string]map[string]any{},
		trustAnchors:         map[string]map[string]any{},
		crls:                 map[string]map[string]any{},
		subjects:             map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
		attributeMappings:    map[string][]map[string]any{},
		notificationSettings: map[string]any{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	profile := s.newProfile("profile-000001", "stackyard-profile", now)
	s.profiles["profile-000001"] = profile
	trustAnchor := s.newTrustAnchor("ta-000001", "stackyard-trust-anchor", now)
	s.trustAnchors["ta-000001"] = trustAnchor
	crl := s.newCRL("crl-000001", now)
	s.crls["crl-000001"] = crl
	subject := s.newSubject("subject-000001", "CN=stackyard", now)
	s.subjects["subject-000001"] = subject
	return s
}

func (s *rolesAnywhereStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	profileID := rolesAnywhereFirstNonEmpty(
		rolesAnywherePathParam(pathParams, "profileId"),
		rolesAnywherePayloadString(payload, "profileId"),
		"profile-000001",
	)
	trustAnchorID := rolesAnywhereFirstNonEmpty(
		rolesAnywherePathParam(pathParams, "trustAnchorId"),
		rolesAnywherePayloadString(payload, "trustAnchorId"),
		"ta-000001",
	)
	crlID := rolesAnywhereFirstNonEmpty(
		rolesAnywherePathParam(pathParams, "crlId"),
		rolesAnywherePayloadString(payload, "crlId"),
		"crl-000001",
	)
	subjectID := rolesAnywhereFirstNonEmpty(
		rolesAnywherePathParam(pathParams, "subjectId"),
		rolesAnywherePayloadString(payload, "subjectId"),
		"subject-000001",
	)
	resourceARN := rolesAnywhereFirstNonEmpty(
		rolesAnywherePayloadString(payload, "resourceArn"),
		strings.TrimSpace(query.Get("resourceArn")),
		rolesAnywherePathParam(pathParams, "resourceArn"),
		rolesAnywhereProfileARN(s.region, s.accountID, profileID),
	)

	switch action {
	case "CreateProfile":
		name := rolesAnywhereFirstNonEmpty(rolesAnywherePayloadString(payload, "name"), "stackyard-profile")
		id := s.nextProfileID()
		profile := s.newProfile(id, name, now)
		if enabled, ok := payload["enabled"].(bool); ok {
			profile["enabled"] = enabled
		}
		if roleArns, ok := payload["roleArns"].([]any); ok {
			profile["roleArns"] = rolesAnywhereAnySliceToStringSlice(roleArns)
		}
		s.profiles[id] = profile
		return map[string]any{"profile": rolesAnywhereCloneMap(profile)}

	case "CreateTrustAnchor":
		name := rolesAnywhereFirstNonEmpty(rolesAnywherePayloadString(payload, "name"), "stackyard-trust-anchor")
		id := s.nextTrustAnchorID()
		trustAnchor := s.newTrustAnchor(id, name, now)
		s.trustAnchors[id] = trustAnchor
		return map[string]any{"trustAnchor": rolesAnywhereCloneMap(trustAnchor)}

	case "DeleteAttributeMapping":
		delete(s.attributeMappings, profileID)
		return map[string]any{}

	case "DeleteCrl":
		delete(s.crls, crlID)
		return map[string]any{}

	case "DeleteProfile":
		delete(s.profiles, profileID)
		delete(s.attributeMappings, profileID)
		return map[string]any{}

	case "DeleteTrustAnchor":
		delete(s.trustAnchors, trustAnchorID)
		return map[string]any{}

	case "DisableCrl":
		crl := s.ensureCRLLocked(crlID, now)
		crl["enabled"] = false
		return map[string]any{"crl": rolesAnywhereCloneMap(crl)}

	case "DisableProfile":
		profile := s.ensureProfileLocked(profileID, now)
		profile["enabled"] = false
		return map[string]any{"profile": rolesAnywhereCloneMap(profile)}

	case "DisableTrustAnchor":
		trustAnchor := s.ensureTrustAnchorLocked(trustAnchorID, now)
		trustAnchor["enabled"] = false
		return map[string]any{"trustAnchor": rolesAnywhereCloneMap(trustAnchor)}

	case "EnableCrl":
		crl := s.ensureCRLLocked(crlID, now)
		crl["enabled"] = true
		return map[string]any{"crl": rolesAnywhereCloneMap(crl)}

	case "EnableProfile":
		profile := s.ensureProfileLocked(profileID, now)
		profile["enabled"] = true
		return map[string]any{"profile": rolesAnywhereCloneMap(profile)}

	case "EnableTrustAnchor":
		trustAnchor := s.ensureTrustAnchorLocked(trustAnchorID, now)
		trustAnchor["enabled"] = true
		return map[string]any{"trustAnchor": rolesAnywhereCloneMap(trustAnchor)}

	case "GetCrl":
		return map[string]any{"crl": rolesAnywhereCloneMap(s.ensureCRLLocked(crlID, now))}

	case "GetProfile":
		profile := rolesAnywhereCloneMap(s.ensureProfileLocked(profileID, now))
		if mappings, ok := s.attributeMappings[profileID]; ok {
			profile["attributeMappings"] = rolesAnywhereCloneMapSlice(mappings)
		}
		return map[string]any{"profile": profile}

	case "GetSubject":
		return map[string]any{"subject": rolesAnywhereCloneMap(s.ensureSubjectLocked(subjectID, now))}

	case "GetTrustAnchor":
		return map[string]any{"trustAnchor": rolesAnywhereCloneMap(s.ensureTrustAnchorLocked(trustAnchorID, now))}

	case "ImportCrl":
		id := s.nextCRLID()
		crl := s.newCRL(id, now)
		s.crls[id] = crl
		return map[string]any{"crl": rolesAnywhereCloneMap(crl)}

	case "ListCrls":
		return map[string]any{"crls": s.listCRLsLocked(now), "nextToken": ""}

	case "ListProfiles":
		return map[string]any{"profiles": s.listProfilesLocked(now), "nextToken": ""}

	case "ListSubjects":
		return map[string]any{"subjects": s.listSubjectsLocked(now), "nextToken": ""}

	case "ListTagsForResource":
		out := []any{}
		for k, v := range s.tags[resourceARN] {
			out = append(out, map[string]any{"key": k, "value": v})
		}
		sort.Slice(out, func(i, j int) bool {
			li := out[i].(map[string]any)["key"].(string)
			lj := out[j].(map[string]any)["key"].(string)
			return li < lj
		})
		return map[string]any{"tags": out}

	case "ListTrustAnchors":
		return map[string]any{"trustAnchors": s.listTrustAnchorsLocked(now), "nextToken": ""}

	case "PutAttributeMapping":
		current := []map[string]any{
			{
				"certificateField": "x509Subject",
				"mappingRules": []any{
					map[string]any{
						"specifier": "CN",
						"roleArn":   rolesAnywherePayloadString(payload, "roleArn"),
					},
				},
			},
		}
		if mappings, ok := payload["attributeMappings"].([]any); ok && len(mappings) > 0 {
			current = rolesAnywhereAnySliceToMapSlice(mappings)
		}
		s.attributeMappings[profileID] = current
		return map[string]any{"attributeMappings": rolesAnywhereCloneMapSlice(current)}

	case "PutNotificationSettings":
		if settings, ok := payload["notificationSettings"]; ok {
			s.notificationSettings["notificationSettings"] = settings
		} else {
			s.notificationSettings["notificationSettings"] = map[string]any{}
		}
		return map[string]any{"notificationSettings": s.notificationSettings["notificationSettings"]}

	case "ResetNotificationSettings":
		s.notificationSettings = map[string]any{}
		return map[string]any{}

	case "TagResource":
		tagSet := s.tags[resourceARN]
		if tagSet == nil {
			tagSet = map[string]string{}
			s.tags[resourceARN] = tagSet
		}
		switch t := payload["tags"].(type) {
		case map[string]any:
			for k, v := range t {
				tagSet[k] = rolesAnywhereAnyString(v)
			}
		case []any:
			for _, item := range t {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := rolesAnywhereFirstNonEmpty(rolesAnywherePayloadMapString(obj, "key"), rolesAnywherePayloadMapString(obj, "Key"))
				if key == "" {
					continue
				}
				val := rolesAnywhereFirstNonEmpty(rolesAnywherePayloadMapString(obj, "value"), rolesAnywherePayloadMapString(obj, "Value"))
				tagSet[key] = val
			}
		}
		return map[string]any{}

	case "UntagResource":
		tagSet := s.tags[resourceARN]
		if tagSet != nil {
			if keys, ok := payload["tagKeys"].([]any); ok {
				for _, key := range keys {
					delete(tagSet, strings.TrimSpace(rolesAnywhereAnyString(key)))
				}
			}
		}
		return map[string]any{}

	case "UpdateCrl":
		crl := s.ensureCRLLocked(crlID, now)
		crl["updatedAt"] = now
		name := rolesAnywherePayloadString(payload, "name")
		if name != "" {
			crl["name"] = name
		}
		return map[string]any{"crl": rolesAnywhereCloneMap(crl)}

	case "UpdateProfile":
		profile := s.ensureProfileLocked(profileID, now)
		profile["updatedAt"] = now
		name := rolesAnywherePayloadString(payload, "name")
		if name != "" {
			profile["name"] = name
		}
		if enabled, ok := payload["enabled"].(bool); ok {
			profile["enabled"] = enabled
		}
		return map[string]any{"profile": rolesAnywhereCloneMap(profile)}

	case "UpdateTrustAnchor":
		trustAnchor := s.ensureTrustAnchorLocked(trustAnchorID, now)
		trustAnchor["updatedAt"] = now
		name := rolesAnywherePayloadString(payload, "name")
		if name != "" {
			trustAnchor["name"] = name
		}
		if enabled, ok := payload["enabled"].(bool); ok {
			trustAnchor["enabled"] = enabled
		}
		return map[string]any{"trustAnchor": rolesAnywhereCloneMap(trustAnchor)}

	default:
		return map[string]any{}
	}
}

func (s *rolesAnywhereStore) nextProfileID() string {
	s.nextID++
	return fmt.Sprintf("profile-%06d", s.nextID)
}

func (s *rolesAnywhereStore) nextTrustAnchorID() string {
	s.nextID++
	return fmt.Sprintf("ta-%06d", s.nextID)
}

func (s *rolesAnywhereStore) nextCRLID() string {
	s.nextID++
	return fmt.Sprintf("crl-%06d", s.nextID)
}

func (s *rolesAnywhereStore) ensureProfileLocked(id, now string) map[string]any {
	id = rolesAnywhereFirstNonEmpty(id, "profile-000001")
	if existing, ok := s.profiles[id]; ok {
		return existing
	}
	profile := s.newProfile(id, "stackyard-profile", now)
	s.profiles[id] = profile
	return profile
}

func (s *rolesAnywhereStore) ensureTrustAnchorLocked(id, now string) map[string]any {
	id = rolesAnywhereFirstNonEmpty(id, "ta-000001")
	if existing, ok := s.trustAnchors[id]; ok {
		return existing
	}
	trustAnchor := s.newTrustAnchor(id, "stackyard-trust-anchor", now)
	s.trustAnchors[id] = trustAnchor
	return trustAnchor
}

func (s *rolesAnywhereStore) ensureCRLLocked(id, now string) map[string]any {
	id = rolesAnywhereFirstNonEmpty(id, "crl-000001")
	if existing, ok := s.crls[id]; ok {
		return existing
	}
	crl := s.newCRL(id, now)
	s.crls[id] = crl
	return crl
}

func (s *rolesAnywhereStore) ensureSubjectLocked(id, now string) map[string]any {
	id = rolesAnywhereFirstNonEmpty(id, "subject-000001")
	if existing, ok := s.subjects[id]; ok {
		return existing
	}
	subject := s.newSubject(id, "CN=stackyard", now)
	s.subjects[id] = subject
	return subject
}

func (s *rolesAnywhereStore) newProfile(id, name, now string) map[string]any {
	return map[string]any{
		"profileId": id,
		"name":      name,
		"enabled":   true,
		"createdAt": now,
		"updatedAt": now,
		"profileArn": rolesAnywhereProfileARN(
			s.region,
			s.accountID,
			id,
		),
		"roleArns": []string{
			"arn:aws:iam::123456789012:role/stackyard-role",
		},
	}
}

func (s *rolesAnywhereStore) newTrustAnchor(id, name, now string) map[string]any {
	return map[string]any{
		"trustAnchorId": id,
		"name":          name,
		"enabled":       true,
		"createdAt":     now,
		"updatedAt":     now,
		"trustAnchorArn": fmt.Sprintf(
			"arn:aws:rolesanywhere:%s:%s:trust-anchor/%s",
			s.region,
			s.accountID,
			id,
		),
	}
}

func (s *rolesAnywhereStore) newCRL(id, now string) map[string]any {
	return map[string]any{
		"crlId":     id,
		"name":      "stackyard-crl",
		"enabled":   true,
		"createdAt": now,
		"updatedAt": now,
		"crlArn":    fmt.Sprintf("arn:aws:rolesanywhere:%s:%s:crl/%s", s.region, s.accountID, id),
		"trustAnchorArn": fmt.Sprintf(
			"arn:aws:rolesanywhere:%s:%s:trust-anchor/%s",
			s.region,
			s.accountID,
			"ta-000001",
		),
	}
}

func (s *rolesAnywhereStore) newSubject(id, subject, now string) map[string]any {
	return map[string]any{
		"subjectId": id,
		"subject":   subject,
		"createdAt": now,
		"x509Subject": map[string]any{
			"commonName": "stackyard",
		},
	}
}

func (s *rolesAnywhereStore) listProfilesLocked(now string) []any {
	ids := make([]string, 0, len(s.profiles))
	for id := range s.profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, rolesAnywhereCloneMap(s.ensureProfileLocked(id, now)))
	}
	return out
}

func (s *rolesAnywhereStore) listTrustAnchorsLocked(now string) []any {
	ids := make([]string, 0, len(s.trustAnchors))
	for id := range s.trustAnchors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, rolesAnywhereCloneMap(s.ensureTrustAnchorLocked(id, now)))
	}
	return out
}

func (s *rolesAnywhereStore) listCRLsLocked(now string) []any {
	ids := make([]string, 0, len(s.crls))
	for id := range s.crls {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, rolesAnywhereCloneMap(s.ensureCRLLocked(id, now)))
	}
	return out
}

func (s *rolesAnywhereStore) listSubjectsLocked(now string) []any {
	ids := make([]string, 0, len(s.subjects))
	for id := range s.subjects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, rolesAnywhereCloneMap(s.ensureSubjectLocked(id, now)))
	}
	return out
}

func rolesAnywhereProfileARN(region, accountID, profileID string) string {
	return fmt.Sprintf("arn:aws:rolesanywhere:%s:%s:profile/%s", region, accountID, profileID)
}

func rolesAnywherePathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v, ok := pathParams[key]; ok {
		return strings.TrimSpace(v)
	}
	for k, v := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func rolesAnywherePayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key]; ok {
		return strings.TrimSpace(rolesAnywhereAnyString(v))
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(rolesAnywhereAnyString(v))
		}
	}
	return ""
}

func rolesAnywherePayloadMapString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key]; ok {
		return strings.TrimSpace(rolesAnywhereAnyString(v))
	}
	return ""
}

func rolesAnywhereAnyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func rolesAnywhereAnySliceToStringSlice(values []any) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		s := strings.TrimSpace(rolesAnywhereAnyString(v))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func rolesAnywhereAnySliceToMapSlice(values []any) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, v := range values {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, rolesAnywhereCloneMap(m))
	}
	return out
}

func rolesAnywhereFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func rolesAnywhereCloneMap(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func rolesAnywhereCloneMapSlice(src []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(src))
	for _, item := range src {
		out = append(out, rolesAnywhereCloneMap(item))
	}
	return out
}
