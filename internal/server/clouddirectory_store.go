package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudDirectoryStore struct {
	mu sync.Mutex

	nextDirectory int64
	nextSchema    int64
	nextFacet     int64
	nextObject    int64

	directories     map[string]map[string]any
	schemas         map[string]map[string]any
	facets          map[string]map[string]any
	typedLinkFacets map[string]map[string]any
	objects         map[string]map[string]any
	tags            map[string]map[string]string
}

func newCloudDirectoryStore() *cloudDirectoryStore {
	s := &cloudDirectoryStore{
		nextDirectory:   2,
		nextSchema:      2,
		nextFacet:       2,
		nextObject:      2,
		directories:     map[string]map[string]any{},
		schemas:         map[string]map[string]any{},
		facets:          map[string]map[string]any{},
		typedLinkFacets: map[string]map[string]any{},
		objects:         map[string]map[string]any{},
		tags:            map[string]map[string]string{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	seedDirectoryARN := cloudDirectoryDirectoryARN("d-00000001")
	seedSchemaARN := cloudDirectorySchemaARN("s-00000001")
	seedObjectID := "root"
	s.directories[seedDirectoryARN] = map[string]any{
		"DirectoryArn":       seedDirectoryARN,
		"Name":               "stackyard-directory",
		"State":              "ENABLED",
		"CreationDateTime":   now,
		"ObjectIdentifier":   seedObjectID,
		"AppliedSchemaArn":   seedSchemaARN,
		"PublishedSchemaArn": seedSchemaARN + "/published",
	}
	s.schemas[seedSchemaARN] = map[string]any{
		"SchemaArn":            seedSchemaARN,
		"Name":                 "stackyard-schema",
		"Version":              "1.0",
		"DevelopmentSchemaArn": seedSchemaARN,
	}
	s.objects[seedObjectID] = map[string]any{
		"ObjectIdentifier": seedObjectID,
		"SchemaFacets":     []any{},
		"CreatedDateTime":  now,
	}
	s.tags[seedDirectoryARN] = map[string]string{"seed": "true"}
	return s
}

func (s *cloudDirectoryStore) Handle(action string, payload map[string]any, _ map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	directoryARN := cloudDirectoryFirstNonEmpty(
		cloudDirectoryString(payload, "DirectoryArn", ""),
		cloudDirectoryDirectoryARN("d-00000001"),
	)
	schemaARN := cloudDirectoryFirstNonEmpty(
		cloudDirectoryString(payload, "SchemaArn", ""),
		cloudDirectoryString(payload, "AppliedSchemaArn", ""),
		cloudDirectorySchemaARN("s-00000001"),
	)
	facetName := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "Name", ""), "stackyard-facet")
	typedLinkFacetName := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "Name", ""), "stackyard-typedlink-facet")
	objectIdentifier := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "ObjectIdentifier", ""), "root")
	resourceARN := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "ResourceArn", ""), directoryARN)

	switch action {
	case "CreateDirectory":
		name := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "Name", ""), fmt.Sprintf("stackyard-directory-%d", s.nextDirectory))
		id := fmt.Sprintf("d-%08d", s.nextDirectory)
		s.nextDirectory++
		directoryARN = cloudDirectoryDirectoryARN(id)
		entry := map[string]any{
			"DirectoryArn":       directoryARN,
			"Name":               name,
			"State":              "ENABLED",
			"CreationDateTime":   now,
			"ObjectIdentifier":   "root",
			"AppliedSchemaArn":   schemaARN,
			"PublishedSchemaArn": schemaARN + "/published",
		}
		s.directories[directoryARN] = entry
		return map[string]any{
			"DirectoryArn":     directoryARN,
			"Name":             name,
			"ObjectIdentifier": "root",
			"AppliedSchemaArn": schemaARN,
		}

	case "DeleteDirectory":
		delete(s.directories, directoryARN)
		return map[string]any{"DirectoryArn": directoryARN}

	case "GetDirectory":
		return map[string]any{"Directory": cloudDirectoryCloneMap(s.ensureDirectoryLocked(directoryARN, now))}

	case "ListDirectories":
		return map[string]any{"Directories": s.listDirectoriesLocked(), "NextToken": ""}

	case "EnableDirectory":
		dir := s.ensureDirectoryLocked(directoryARN, now)
		dir["State"] = "ENABLED"
		return map[string]any{}

	case "DisableDirectory":
		dir := s.ensureDirectoryLocked(directoryARN, now)
		dir["State"] = "DISABLED"
		return map[string]any{}

	case "CreateSchema":
		name := cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "Name", ""), fmt.Sprintf("stackyard-schema-%d", s.nextSchema))
		id := fmt.Sprintf("s-%08d", s.nextSchema)
		s.nextSchema++
		schemaARN = cloudDirectorySchemaARN(id)
		s.schemas[schemaARN] = map[string]any{
			"SchemaArn":            schemaARN,
			"Name":                 name,
			"Version":              "1.0",
			"DevelopmentSchemaArn": schemaARN,
			"CreatedDateTime":      now,
		}
		return map[string]any{"SchemaArn": schemaARN}

	case "DeleteSchema":
		delete(s.schemas, schemaARN)
		return map[string]any{"SchemaArn": schemaARN}

	case "UpdateSchema":
		s.ensureSchemaLocked(schemaARN, now)["UpdatedDateTime"] = now
		return map[string]any{"SchemaArn": schemaARN}

	case "PublishSchema":
		published := schemaARN + "/published"
		return map[string]any{"PublishedSchemaArn": published}

	case "PutSchemaFromJson":
		return map[string]any{"Arn": schemaARN}

	case "GetSchemaAsJson":
		return map[string]any{
			"Name":            cloudDirectoryFirstNonEmpty(cloudDirectoryString(payload, "SchemaName", ""), "stackyard-schema"),
			"Document":        "{}",
			"SchemaArn":       schemaARN,
			"CreatedDateTime": now,
		}

	case "GetAppliedSchemaVersion":
		return map[string]any{"AppliedSchemaArn": schemaARN}

	case "ListAppliedSchemaArns":
		return map[string]any{"SchemaArns": s.sortedSchemaARNsLocked(), "NextToken": ""}

	case "ListDevelopmentSchemaArns":
		return map[string]any{"SchemaArns": s.sortedSchemaARNsLocked(), "NextToken": ""}

	case "ListManagedSchemaArns":
		return map[string]any{"SchemaArns": s.sortedSchemaARNsLocked(), "NextToken": ""}

	case "ListPublishedSchemaArns":
		published := make([]any, 0, len(s.schemas))
		for arn := range s.schemas {
			published = append(published, arn+"/published")
		}
		return map[string]any{"SchemaArns": published, "NextToken": ""}

	case "CreateFacet":
		if facetName == "stackyard-facet" && len(s.facets) > 0 {
			for k := range s.facets {
				facetName = k
				break
			}
		}
		facet := map[string]any{
			"Name":            facetName,
			"SchemaArn":       schemaARN,
			"ObjectType":      "NODE",
			"FacetStyle":      "STATIC",
			"CreatedDateTime": now,
		}
		s.facets[facetName] = facet
		return map[string]any{"Facet": cloudDirectoryCloneMap(facet)}

	case "GetFacet":
		return map[string]any{"Facet": cloudDirectoryCloneMap(s.ensureFacetLocked(facetName, schemaARN, now))}

	case "UpdateFacet":
		facet := s.ensureFacetLocked(facetName, schemaARN, now)
		facet["UpdatedDateTime"] = now
		return map[string]any{"Facet": cloudDirectoryCloneMap(facet)}

	case "DeleteFacet":
		delete(s.facets, facetName)
		return map[string]any{}

	case "ListFacetNames":
		names := make([]any, 0, len(s.facets))
		keys := make([]string, 0, len(s.facets))
		for name := range s.facets {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			names = append(names, name)
		}
		return map[string]any{"FacetNames": names, "NextToken": ""}

	case "ListFacetAttributes":
		return map[string]any{"Attributes": []any{}, "NextToken": ""}

	case "CreateTypedLinkFacet":
		facet := map[string]any{
			"Name":            typedLinkFacetName,
			"SchemaArn":       schemaARN,
			"CreatedDateTime": now,
		}
		s.typedLinkFacets[typedLinkFacetName] = facet
		return map[string]any{"TypedLinkFacet": cloudDirectoryCloneMap(facet)}

	case "GetTypedLinkFacetInformation":
		facet := s.ensureTypedLinkFacetLocked(typedLinkFacetName, schemaARN, now)
		return map[string]any{
			"IdentityAttributeOrder": []any{},
			"TypedLinkFacet":         cloudDirectoryCloneMap(facet),
		}

	case "UpdateTypedLinkFacet":
		facet := s.ensureTypedLinkFacetLocked(typedLinkFacetName, schemaARN, now)
		facet["UpdatedDateTime"] = now
		return map[string]any{"TypedLinkFacet": cloudDirectoryCloneMap(facet)}

	case "DeleteTypedLinkFacet":
		delete(s.typedLinkFacets, typedLinkFacetName)
		return map[string]any{}

	case "ListTypedLinkFacetNames":
		names := make([]any, 0, len(s.typedLinkFacets))
		keys := make([]string, 0, len(s.typedLinkFacets))
		for name := range s.typedLinkFacets {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			names = append(names, name)
		}
		return map[string]any{"FacetNames": names, "NextToken": ""}

	case "ListTypedLinkFacetAttributes":
		return map[string]any{"Attributes": []any{}, "NextToken": ""}

	case "CreateObject":
		objectIdentifier = fmt.Sprintf("o-%08d", s.nextObject)
		s.nextObject++
		s.objects[objectIdentifier] = map[string]any{
			"ObjectIdentifier": objectIdentifier,
			"SchemaFacets":     []any{},
			"CreatedDateTime":  now,
		}
		return map[string]any{"ObjectIdentifier": objectIdentifier}

	case "DeleteObject":
		delete(s.objects, objectIdentifier)
		return map[string]any{}

	case "GetObjectInformation":
		return map[string]any{
			"SchemaFacets":     []any{},
			"ObjectIdentifier": objectIdentifier,
		}

	case "GetObjectAttributes":
		return map[string]any{"Attributes": map[string]any{}}

	case "ListObjectAttributes":
		return map[string]any{"Attributes": []any{}, "NextToken": ""}

	case "ListObjectChildren":
		return map[string]any{"Children": map[string]any{}, "NextToken": ""}

	case "ListObjectParentPaths":
		return map[string]any{"PathToObjectIdentifiersList": []any{}, "NextToken": ""}

	case "ListObjectParents":
		return map[string]any{"Parents": map[string]any{}, "NextToken": ""}

	case "ListObjectPolicies":
		return map[string]any{"AttachedPolicyIds": []any{}, "NextToken": ""}

	case "AttachObject", "DetachObject":
		return map[string]any{"AttachedObjectIdentifier": objectIdentifier}

	case "AttachToIndex", "DetachFromIndex":
		return map[string]any{"AttachedObjectIdentifier": objectIdentifier}

	case "ListAttachedIndices", "ListIndex":
		return map[string]any{"IndexAttachments": []any{}, "NextToken": ""}

	case "AttachPolicy", "DetachPolicy", "LookupPolicy":
		if action == "LookupPolicy" {
			return map[string]any{"PolicyToPathList": []any{}, "NextToken": ""}
		}
		return map[string]any{}

	case "ListPolicyAttachments":
		return map[string]any{"ObjectIdentifiers": []any{}, "NextToken": ""}

	case "AttachTypedLink", "DetachTypedLink":
		return map[string]any{"TypedLinkSpecifier": map[string]any{}}

	case "GetLinkAttributes":
		return map[string]any{"Attributes": map[string]any{}}

	case "UpdateLinkAttributes", "UpdateObjectAttributes":
		return map[string]any{"ObjectIdentifier": objectIdentifier}

	case "ListIncomingTypedLinks", "ListOutgoingTypedLinks":
		return map[string]any{"LinkSpecifiers": []any{}, "NextToken": ""}

	case "BatchRead":
		return map[string]any{"Responses": []any{}, "Exceptions": []any{}}

	case "BatchWrite":
		return map[string]any{"Responses": []any{}, "Exceptions": []any{}}

	case "ApplySchema", "AddFacetToObject", "RemoveFacetFromObject":
		return map[string]any{"ObjectIdentifier": objectIdentifier}

	case "TagResource":
		tagMap := s.ensureTagsLocked(resourceARN)
		tagsKey := cloudDirectoryFindKeyCaseInsensitive(payload, "Tags")
		if tagsKey != "" {
			if entries, ok := payload[tagsKey].([]any); ok {
				for _, entry := range entries {
					if tag, ok := entry.(map[string]any); ok {
						key := cloudDirectoryString(tag, "Key", "")
						value := cloudDirectoryString(tag, "Value", "")
						if key != "" {
							tagMap[key] = value
						}
					}
				}
			}
		}
		return map[string]any{}

	case "UntagResource":
		tagMap := s.ensureTagsLocked(resourceARN)
		keysKey := cloudDirectoryFindKeyCaseInsensitive(payload, "TagKeys")
		if keysKey != "" {
			if keys, ok := payload[keysKey].([]any); ok {
				for _, item := range keys {
					if key, ok := item.(string); ok {
						delete(tagMap, key)
					}
				}
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		tags := s.ensureTagsLocked(resourceARN)
		out := make([]any, 0, len(tags))
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": tags[k]})
		}
		return map[string]any{"Tags": out, "NextToken": ""}

	case "UpgradeAppliedSchema", "UpgradePublishedSchema":
		return map[string]any{"UpgradedSchemaArn": schemaARN}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}
	return map[string]any{}
}

func (s *cloudDirectoryStore) ensureDirectoryLocked(directoryARN, now string) map[string]any {
	if entry, ok := s.directories[directoryARN]; ok {
		return entry
	}
	entry := map[string]any{
		"DirectoryArn":     directoryARN,
		"Name":             "stackyard-directory",
		"State":            "ENABLED",
		"CreationDateTime": now,
		"ObjectIdentifier": "root",
	}
	s.directories[directoryARN] = entry
	return entry
}

func (s *cloudDirectoryStore) ensureSchemaLocked(schemaARN, now string) map[string]any {
	if entry, ok := s.schemas[schemaARN]; ok {
		return entry
	}
	entry := map[string]any{
		"SchemaArn":            schemaARN,
		"Name":                 "stackyard-schema",
		"Version":              "1.0",
		"DevelopmentSchemaArn": schemaARN,
		"CreatedDateTime":      now,
	}
	s.schemas[schemaARN] = entry
	return entry
}

func (s *cloudDirectoryStore) ensureFacetLocked(name, schemaARN, now string) map[string]any {
	if entry, ok := s.facets[name]; ok {
		return entry
	}
	entry := map[string]any{
		"Name":            name,
		"SchemaArn":       schemaARN,
		"ObjectType":      "NODE",
		"FacetStyle":      "STATIC",
		"CreatedDateTime": now,
	}
	s.facets[name] = entry
	return entry
}

func (s *cloudDirectoryStore) ensureTypedLinkFacetLocked(name, schemaARN, now string) map[string]any {
	if entry, ok := s.typedLinkFacets[name]; ok {
		return entry
	}
	entry := map[string]any{
		"Name":            name,
		"SchemaArn":       schemaARN,
		"CreatedDateTime": now,
	}
	s.typedLinkFacets[name] = entry
	return entry
}

func (s *cloudDirectoryStore) ensureTagsLocked(resourceARN string) map[string]string {
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *cloudDirectoryStore) listDirectoriesLocked() []any {
	arns := make([]string, 0, len(s.directories))
	for arn := range s.directories {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, cloudDirectoryCloneMap(s.directories[arn]))
	}
	return out
}

func (s *cloudDirectoryStore) sortedSchemaARNsLocked() []any {
	arns := make([]string, 0, len(s.schemas))
	for arn := range s.schemas {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, arn)
	}
	return out
}

func cloudDirectoryDirectoryARN(id string) string {
	return "arn:aws:clouddirectory:us-east-1:123456789012:directory/" + id
}

func cloudDirectorySchemaARN(id string) string {
	return "arn:aws:clouddirectory:us-east-1:123456789012:schema/" + id
}

func cloudDirectoryString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return def
}

func cloudDirectoryFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func cloudDirectoryCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloudDirectoryFindKeyCaseInsensitive(payload map[string]any, key string) string {
	for k := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return k
		}
	}
	return ""
}
