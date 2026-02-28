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

type cloudWatchSyntheticsStore struct {
	mu sync.Mutex

	nextID int64

	canaries       map[string]map[string]any
	groups         map[string]map[string]any
	groupResources map[string]map[string]bool
	tags           map[string]map[string]string
}

func newCloudWatchSyntheticsStore() *cloudWatchSyntheticsStore {
	store := &cloudWatchSyntheticsStore{
		nextID:         2,
		canaries:       map[string]map[string]any{},
		groups:         map[string]map[string]any{},
		groupResources: map[string]map[string]bool{},
		tags:           map[string]map[string]string{},
	}

	canary := store.ensureCanaryLocked("stackyard-canary")
	group := store.ensureGroupLocked("stackyard-group")
	groupID := syntheticsGroupID(group)
	if store.groupResources[groupID] == nil {
		store.groupResources[groupID] = map[string]bool{}
	}
	store.groupResources[groupID][syntheticsCanaryARN(canary)] = true

	store.tags[syntheticsCanaryARN(canary)] = map[string]string{"stackyard": "true"}
	store.tags[syntheticsGroupARN(groupID)] = map[string]string{"stackyard": "true"}
	return store
}

func (s *cloudWatchSyntheticsStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateCanary":
		name := syntheticsDefaultStringAny(payload, "Name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		s.applyCanaryPayloadLocked(canary, payload)
		return map[string]any{"Canary": syntheticsCloneMap(canary)}

	case "DeleteCanary":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		delete(s.canaries, name)
		for groupID, resources := range s.groupResources {
			delete(resources, syntheticsCanaryARNByName(name))
			s.groupResources[groupID] = resources
		}
		delete(s.tags, syntheticsCanaryARNByName(name))
		return map[string]any{}

	case "DescribeCanaries":
		return map[string]any{"Canaries": s.listCanariesLocked(), "NextToken": ""}

	case "DescribeCanariesLastRun":
		return map[string]any{"CanariesLastRun": s.listCanariesLastRunLocked(), "NextToken": ""}

	case "GetCanary":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		return map[string]any{"Canary": syntheticsCloneMap(s.ensureCanaryLocked(name))}

	case "GetCanaryRuns":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		return map[string]any{
			"CanaryRuns": []any{
				map[string]any{
					"Id":                 fmt.Sprintf("run-%06d", s.nextLocked()),
					"Name":               name,
					"Status":             map[string]any{"State": canary["Status"]},
					"Timeline":           map[string]any{"Started": time.Now().UTC().Add(-1 * time.Minute), "Completed": time.Now().UTC()},
					"ArtifactS3Location": fmt.Sprintf("s3://stackyard-synthetics-artifacts/%s/", name),
				},
			},
			"NextToken": "",
		}

	case "StartCanary":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		canary["Status"] = "RUNNING"
		canary["StateReason"] = "canary started"
		return map[string]any{}

	case "StartCanaryDryRun":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		canary["Status"] = "RUNNING"
		canary["StateReason"] = "dry run started"
		return map[string]any{
			"DryRunConfig": map[string]any{
				"Id":        fmt.Sprintf("dryrun-%06d", s.nextLocked()),
				"DryRunId":  fmt.Sprintf("dryrun-%06d", s.nextLocked()),
				"Status":    "RUNNING",
				"StartedOn": time.Now().UTC(),
			},
		}

	case "StopCanary":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		canary["Status"] = "STOPPED"
		canary["StateReason"] = "canary stopped"
		return map[string]any{}

	case "UpdateCanary":
		name := syntheticsDefaultString(pathParams, "name", "stackyard-canary")
		canary := s.ensureCanaryLocked(name)
		s.applyCanaryPayloadLocked(canary, payload)
		return map[string]any{"Canary": syntheticsCloneMap(canary)}

	case "CreateGroup":
		groupID := syntheticsDefaultStringAny(payload, "Name", "stackyard-group")
		group := s.ensureGroupLocked(groupID)
		return map[string]any{"Group": syntheticsCloneMap(group)}

	case "DeleteGroup":
		groupID := syntheticsDefaultString(pathParams, "groupIdentifier", "stackyard-group")
		delete(s.groups, groupID)
		delete(s.groupResources, groupID)
		delete(s.tags, syntheticsGroupARN(groupID))
		return map[string]any{}

	case "GetGroup":
		groupID := syntheticsDefaultString(pathParams, "groupIdentifier", "stackyard-group")
		return map[string]any{"Group": syntheticsCloneMap(s.ensureGroupLocked(groupID))}

	case "ListGroups":
		return map[string]any{"Groups": s.listGroupsLocked(), "NextToken": ""}

	case "AssociateResource":
		groupID := syntheticsDefaultString(pathParams, "groupIdentifier", "stackyard-group")
		s.ensureGroupLocked(groupID)
		resourceARN := syntheticsDefaultStringAny(payload, "ResourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		if s.groupResources[groupID] == nil {
			s.groupResources[groupID] = map[string]bool{}
		}
		s.groupResources[groupID][resourceARN] = true
		return map[string]any{}

	case "DisassociateResource":
		groupID := syntheticsDefaultString(pathParams, "groupIdentifier", "stackyard-group")
		resourceARN := syntheticsDefaultStringAny(payload, "ResourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		if s.groupResources[groupID] != nil {
			delete(s.groupResources[groupID], resourceARN)
		}
		return map[string]any{}

	case "ListAssociatedGroups":
		resourceARN := syntheticsDefaultString(pathParams, "resourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		out := []any{}
		groupIDs := make([]string, 0, len(s.groupResources))
		for groupID := range s.groupResources {
			groupIDs = append(groupIDs, groupID)
		}
		sort.Strings(groupIDs)
		for _, groupID := range groupIDs {
			if s.groupResources[groupID][resourceARN] {
				out = append(out, map[string]any{"GroupIdentifier": groupID})
			}
		}
		return map[string]any{"Groups": out, "NextToken": ""}

	case "ListGroupResources":
		groupID := syntheticsDefaultString(pathParams, "groupIdentifier", "stackyard-group")
		resources := s.groupResources[groupID]
		arns := make([]string, 0, len(resources))
		for arn := range resources {
			arns = append(arns, arn)
		}
		sort.Strings(arns)
		out := make([]any, 0, len(arns))
		for _, arn := range arns {
			out = append(out, map[string]any{"ResourceArn": arn})
		}
		return map[string]any{"Resources": out, "NextToken": ""}

	case "DescribeRuntimeVersions":
		return map[string]any{
			"RuntimeVersions": []any{
				map[string]any{
					"VersionName":     "syn-nodejs-puppeteer-6.2",
					"Description":     "Stackyard synthetic runtime",
					"DeprecationDate": time.Now().UTC().AddDate(1, 0, 0),
					"ReleaseDate":     time.Now().UTC().AddDate(0, -1, 0),
				},
			},
		}

	case "ListTagsForResource":
		resourceARN := syntheticsDefaultString(pathParams, "resourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		return map[string]any{"Tags": syntheticsCloneStringMap(s.tags[resourceARN])}

	case "TagResource":
		resourceARN := syntheticsDefaultString(pathParams, "resourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		newTags := syntheticsMapString(payload, "Tags")
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range newTags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := syntheticsDefaultString(pathParams, "resourceArn", syntheticsCanaryARNByName("stackyard-canary"))
		keys := syntheticsStringSlice(payload, "TagKeys")
		for _, k := range keys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], k)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *cloudWatchSyntheticsStore) ensureCanaryLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-canary"
	}
	canary := s.canaries[key]
	if canary != nil {
		return canary
	}
	canary = map[string]any{
		"Id":                           fmt.Sprintf("canary-%06d", s.nextLocked()),
		"Name":                         key,
		"Arn":                          syntheticsCanaryARNByName(key),
		"Status":                       "READY",
		"StateReason":                  "configured",
		"EngineArn":                    "arn:aws:lambda:us-east-1:123456789012:function:stackyard-synthetics-engine",
		"ExecutionRoleArn":             "arn:aws:iam::123456789012:role/stackyard-synthetics-role",
		"RuntimeVersion":               "syn-nodejs-puppeteer-6.2",
		"Schedule":                     map[string]any{"Expression": "rate(5 minutes)", "DurationInSeconds": 0},
		"SuccessRetentionPeriodInDays": 31,
		"FailureRetentionPeriodInDays": 31,
		"Code": map[string]any{
			"Handler": "index.handler",
		},
	}
	s.canaries[key] = canary
	return canary
}

func (s *cloudWatchSyntheticsStore) ensureGroupLocked(groupID string) map[string]any {
	key := strings.TrimSpace(groupID)
	if key == "" {
		key = "stackyard-group"
	}
	group := s.groups[key]
	if group != nil {
		return group
	}
	group = map[string]any{
		"Id":        key,
		"Name":      key,
		"Arn":       syntheticsGroupARN(key),
		"CreatedOn": time.Now().UTC(),
	}
	s.groups[key] = group
	return group
}

func (s *cloudWatchSyntheticsStore) listCanariesLocked() []any {
	names := make([]string, 0, len(s.canaries))
	for name := range s.canaries {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, syntheticsCloneMap(s.canaries[name]))
	}
	return out
}

func (s *cloudWatchSyntheticsStore) listCanariesLastRunLocked() []any {
	names := make([]string, 0, len(s.canaries))
	for name := range s.canaries {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		canary := s.canaries[name]
		out = append(out, map[string]any{
			"CanaryName": name,
			"LastRun": map[string]any{
				"Id":       fmt.Sprintf("run-%06d", s.nextLocked()),
				"Status":   map[string]any{"State": canary["Status"]},
				"Timeline": map[string]any{"Started": time.Now().UTC().Add(-1 * time.Minute), "Completed": time.Now().UTC()},
			},
		})
	}
	return out
}

func (s *cloudWatchSyntheticsStore) listGroupsLocked() []any {
	ids := make([]string, 0, len(s.groups))
	for id := range s.groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, syntheticsCloneMap(s.groups[id]))
	}
	return out
}

func (s *cloudWatchSyntheticsStore) applyCanaryPayloadLocked(canary map[string]any, payload map[string]any) {
	if arn := syntheticsString(payload, "ArtifactS3Location"); arn != "" {
		canary["ArtifactS3Location"] = arn
	}
	if rv := syntheticsString(payload, "RuntimeVersion"); rv != "" {
		canary["RuntimeVersion"] = rv
	}
	if role := syntheticsString(payload, "ExecutionRoleArn"); role != "" {
		canary["ExecutionRoleArn"] = role
	}
	if schedule := syntheticsMapAny(payload, "Schedule"); len(schedule) != 0 {
		canary["Schedule"] = schedule
	}
	if code := syntheticsMapAny(payload, "Code"); len(code) != 0 {
		canary["Code"] = code
	}
	if tags := syntheticsMapString(payload, "Tags"); len(tags) != 0 {
		arn := syntheticsCanaryARN(canary)
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[arn][k] = v
		}
	}
}

func (s *cloudWatchSyntheticsStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func syntheticsCanaryARN(canary map[string]any) string {
	if v := syntheticsString(canary, "Arn"); v != "" {
		return v
	}
	return syntheticsCanaryARNByName(syntheticsDefaultStringAny(canary, "Name", "stackyard-canary"))
}

func syntheticsCanaryARNByName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		n = "stackyard-canary"
	}
	if strings.HasPrefix(n, "arn:") {
		return n
	}
	return fmt.Sprintf("arn:aws:synthetics:us-east-1:123456789012:canary:%s", n)
}

func syntheticsGroupARN(groupID string) string {
	g := strings.TrimSpace(groupID)
	if g == "" {
		g = "stackyard-group"
	}
	if strings.HasPrefix(g, "arn:") {
		return g
	}
	return fmt.Sprintf("arn:aws:synthetics:us-east-1:123456789012:group:%s", g)
}

func syntheticsGroupID(group map[string]any) string {
	if id := syntheticsString(group, "Id"); id != "" {
		return id
	}
	if name := syntheticsString(group, "Name"); name != "" {
		return name
	}
	return "stackyard-group"
}

func syntheticsString(payload map[string]any, key string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return ""
}

func syntheticsDefaultString(payload map[string]string, key, fallback string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func syntheticsDefaultStringAny(payload map[string]any, key, fallback string) string {
	if value := syntheticsString(payload, key); value != "" {
		return value
	}
	return fallback
}

// syntheticsDefaultString supports either map[string]string or map[string]any via wrappers.
func syntheticsMapAny(payload map[string]any, key string) map[string]any {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
		case map[string]any:
			return syntheticsCloneMap(typed)
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				return map[string]any{}
			}
			decoded := map[string]any{}
			if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
				return decoded
			}
		}
	}
	return map[string]any{}
}

func syntheticsMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if typed, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for mk, mv := range typed {
				out[mk] = strings.TrimSpace(fmt.Sprintf("%v", mv))
			}
			return out
		}
		if typed, ok := v.(map[string]string); ok {
			return syntheticsCloneStringMap(typed)
		}
	}
	return map[string]string{}
}

func syntheticsStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
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
				if trimmed := strings.TrimSpace(fmt.Sprintf("%v", item)); trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		}
	}
	return []string{}
}

func syntheticsCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func syntheticsCloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
