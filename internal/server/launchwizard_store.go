package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type launchWizardStore struct {
	mu sync.Mutex

	nextID int64

	workloads              map[string]map[string]any
	workloadPatterns       map[string]map[string]map[string]any
	patternVersions        map[string][]map[string]any
	deployments            map[string]map[string]any
	deploymentEvents       map[string][]map[string]any
	tags                   map[string]map[string]string
	createDeploymentTokens map[string]string
}

func newLaunchWizardStore() *launchWizardStore {
	now := time.Now().UTC().Format(time.RFC3339)
	workloadName := "SAP_HANA_SINGLE"
	patternName := "single-node-hana"
	versionName := "v1"
	deploymentID := "deployment-000001"
	deploymentARN := "arn:aws:launchwizard:us-east-1:123456789012:deployment/deployment-000001"

	s := &launchWizardStore{
		nextID: 2,
		workloads: map[string]map[string]any{
			workloadName: {
				"workloadName":        workloadName,
				"workloadDisplayName": "SAP HANA Single Node",
				"description":         "Seeded workload for deterministic Launch Wizard emulation",
				"status":              "ACTIVE",
			},
		},
		workloadPatterns: map[string]map[string]map[string]any{
			workloadName: {
				patternName: {
					"workloadName":                 workloadName,
					"deploymentPatternName":        patternName,
					"deploymentPatternDisplayName": "Single Node HANA",
					"description":                  "Seeded deployment pattern",
					"status":                       "ACTIVE",
				},
			},
		},
		patternVersions: map[string][]map[string]any{
			workloadName + "|" + patternName: {
				{
					"workloadName":                 workloadName,
					"deploymentPatternName":        patternName,
					"deploymentPatternVersionName": versionName,
					"status":                       "ACTIVE",
					"createdAt":                    now,
				},
			},
		},
		deployments: map[string]map[string]any{
			deploymentID: {
				"deploymentId":                  deploymentID,
				"deploymentArn":                 deploymentARN,
				"name":                          "stackyard-seeded-deployment",
				"status":                        "DEPLOYED",
				"workloadName":                  workloadName,
				"workloadDeploymentPatternName": patternName,
				"workloadVersionName":           versionName,
				"createdAt":                     now,
				"updatedAt":                     now,
			},
		},
		deploymentEvents: map[string][]map[string]any{
			deploymentID: {
				{
					"eventId":      "event-000001",
					"deploymentId": deploymentID,
					"status":       "SUCCEEDED",
					"timestamp":    now,
					"description":  "Seeded deployment completed",
				},
			},
		},
		tags: map[string]map[string]string{
			deploymentARN: {
				"stackyard": "true",
			},
		},
		createDeploymentTokens: map[string]string{},
	}
	return s
}

func (s *launchWizardStore) Handle(action string, payload map[string]any, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateDeployment":
		name := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "name", "deploymentName"),
			"stackyard-deployment",
		)
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			s.firstWorkloadNameLocked(),
		)
		patternName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadDeploymentPatternName", "deploymentPatternName"),
			s.firstPatternNameLocked(workloadName),
		)
		versionName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadVersionName", "deploymentPatternVersionName"),
			s.firstPatternVersionLocked(workloadName, patternName),
		)
		clientToken := launchWizardPayloadString(payload, "clientToken", "ClientToken")
		if clientToken != "" {
			if existingID := s.createDeploymentTokens[clientToken]; existingID != "" {
				return map[string]any{
					"deployment": launchWizardCloneMap(s.deployments[existingID]),
				}
			}
		}

		id := "deployment-" + fmt.Sprintf("%06d", s.nextID)
		s.nextID++
		arn := "arn:aws:launchwizard:us-east-1:123456789012:deployment/" + id
		deployment := map[string]any{
			"deploymentId":                  id,
			"deploymentArn":                 arn,
			"name":                          name,
			"status":                        "IN_PROGRESS",
			"workloadName":                  workloadName,
			"workloadDeploymentPatternName": patternName,
			"workloadVersionName":           versionName,
			"createdAt":                     now,
			"updatedAt":                     now,
		}
		s.deployments[id] = deployment
		s.deploymentEvents[id] = append(s.deploymentEvents[id], map[string]any{
			"eventId":      "event-" + fmt.Sprintf("%06d", s.nextID),
			"deploymentId": id,
			"status":       "IN_PROGRESS",
			"timestamp":    now,
			"description":  "Deployment started",
		})
		if clientToken != "" {
			s.createDeploymentTokens[clientToken] = id
		}
		s.ensureTagsLocked(arn)["stackyard"] = "true"
		return map[string]any{"deployment": launchWizardCloneMap(deployment)}

	case "UpdateDeployment":
		deployment := s.ensureDeploymentByIDLocked(s.deploymentIdentifier(payload, query), now)
		if v := launchWizardPayloadString(payload, "name", "deploymentName"); v != "" {
			deployment["name"] = v
		}
		if v := launchWizardPayloadString(payload, "status"); v != "" {
			deployment["status"] = v
		} else {
			deployment["status"] = "UPDATED"
		}
		deployment["updatedAt"] = now
		s.deploymentEvents[launchWizardPayloadString(deployment, "deploymentId")] = append(
			s.deploymentEvents[launchWizardPayloadString(deployment, "deploymentId")],
			map[string]any{
				"eventId":      "event-" + fmt.Sprintf("%06d", s.nextID),
				"deploymentId": launchWizardPayloadString(deployment, "deploymentId"),
				"status":       "UPDATED",
				"timestamp":    now,
				"description":  "Deployment updated",
			},
		)
		s.nextID++
		return map[string]any{"deployment": launchWizardCloneMap(deployment)}

	case "DeleteDeployment":
		deployment := s.ensureDeploymentByIDLocked(s.deploymentIdentifier(payload, query), now)
		deploymentID := launchWizardPayloadString(deployment, "deploymentId")
		deploymentARN := launchWizardPayloadString(deployment, "deploymentArn")
		delete(s.deployments, deploymentID)
		delete(s.deploymentEvents, deploymentID)
		delete(s.tags, deploymentARN)
		return map[string]any{}

	case "GetDeployment":
		deployment := s.ensureDeploymentByIDLocked(s.deploymentIdentifier(payload, query), now)
		return map[string]any{"deployment": launchWizardCloneMap(deployment)}

	case "ListDeployments":
		items := make([]any, 0, len(s.deployments))
		for _, deployment := range launchWizardSortedMaps(s.deployments) {
			items = append(items, launchWizardDeploymentSummary(deployment))
		}
		return map[string]any{"deployments": items, "nextToken": ""}

	case "ListDeploymentEvents":
		deployment := s.ensureDeploymentByIDLocked(s.deploymentIdentifier(payload, query), now)
		deploymentID := launchWizardPayloadString(deployment, "deploymentId")
		events := s.deploymentEvents[deploymentID]
		if len(events) == 0 {
			events = []map[string]any{
				{
					"eventId":      "event-000000",
					"deploymentId": deploymentID,
					"status":       launchWizardPayloadString(deployment, "status"),
					"timestamp":    now,
					"description":  "No events recorded",
				},
			}
			s.deploymentEvents[deploymentID] = events
		}
		return map[string]any{"deploymentEvents": launchWizardCloneListOfMaps(events), "nextToken": ""}

	case "GetWorkload":
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			launchWizardLookupString(nil, payload, query, "workloadName", "WorkloadName"),
			s.firstWorkloadNameLocked(),
		)
		workload := s.ensureWorkloadLocked(workloadName)
		return map[string]any{"workload": launchWizardCloneMap(workload)}

	case "ListWorkloads":
		items := make([]any, 0, len(s.workloads))
		for _, workload := range launchWizardSortedMaps(s.workloads) {
			items = append(items, launchWizardWorkloadSummary(workload))
		}
		return map[string]any{"workloads": items, "nextToken": ""}

	case "GetWorkloadDeploymentPattern":
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			s.firstWorkloadNameLocked(),
		)
		patternName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "deploymentPatternName", "workloadDeploymentPatternName"),
			s.firstPatternNameLocked(workloadName),
		)
		pattern := s.ensurePatternLocked(workloadName, patternName)
		return map[string]any{"workloadDeploymentPattern": launchWizardCloneMap(pattern)}

	case "ListWorkloadDeploymentPatterns":
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			s.firstWorkloadNameLocked(),
		)
		patterns := s.workloadPatterns[workloadName]
		items := make([]any, 0, len(patterns))
		for _, pattern := range launchWizardSortedMaps(patterns) {
			items = append(items, launchWizardPatternSummary(pattern))
		}
		return map[string]any{"workloadDeploymentPatterns": items, "nextToken": ""}

	case "GetDeploymentPatternVersion":
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			s.firstWorkloadNameLocked(),
		)
		patternName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "deploymentPatternName", "workloadDeploymentPatternName"),
			s.firstPatternNameLocked(workloadName),
		)
		versionName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "deploymentPatternVersionName", "workloadVersionName"),
			s.firstPatternVersionLocked(workloadName, patternName),
		)
		version := s.ensurePatternVersionLocked(workloadName, patternName, versionName, now)
		return map[string]any{"deploymentPatternVersion": launchWizardCloneMap(version)}

	case "ListDeploymentPatternVersions":
		workloadName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "workloadName"),
			s.firstWorkloadNameLocked(),
		)
		patternName := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "deploymentPatternName", "workloadDeploymentPatternName"),
			s.firstPatternNameLocked(workloadName),
		)
		key := workloadName + "|" + patternName
		versions := s.patternVersions[key]
		if len(versions) == 0 {
			versions = []map[string]any{s.ensurePatternVersionLocked(workloadName, patternName, "v1", now)}
			s.patternVersions[key] = versions
		}
		return map[string]any{"deploymentPatternVersions": launchWizardCloneListOfMaps(versions), "nextToken": ""}

	case "TagResource":
		resourceARN := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "resourceArn", "ResourceArn", "deploymentArn"),
			strings.TrimSpace(query.Get("resourceArn")),
			s.firstDeploymentARNLocked(),
		)
		incoming := launchWizardExtractTags(payload)
		if len(incoming) == 0 {
			incoming = map[string]string{"env": "test"}
		}
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range incoming {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "resourceArn", "ResourceArn", "deploymentArn"),
			strings.TrimSpace(query.Get("resourceArn")),
			s.firstDeploymentARNLocked(),
		)
		tagKeys := launchWizardExtractTagKeys(payload, query)
		tags := s.ensureTagsLocked(resourceARN)
		for _, tagKey := range tagKeys {
			delete(tags, tagKey)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := launchWizardFirstNonEmpty(
			launchWizardPayloadString(payload, "resourceArn", "ResourceArn", "deploymentArn"),
			strings.TrimSpace(query.Get("resourceArn")),
			s.firstDeploymentARNLocked(),
		)
		tags := map[string]any{}
		for k, v := range s.ensureTagsLocked(resourceARN) {
			tags[k] = v
		}
		return map[string]any{"tags": tags}

	default:
		return map[string]any{}
	}
}

func (s *launchWizardStore) firstWorkloadNameLocked() string {
	for k := range s.workloads {
		return k
	}
	return "SAP_HANA_SINGLE"
}

func (s *launchWizardStore) firstPatternNameLocked(workloadName string) string {
	if patterns := s.workloadPatterns[workloadName]; len(patterns) > 0 {
		for k := range patterns {
			return k
		}
	}
	return "single-node-hana"
}

func (s *launchWizardStore) firstPatternVersionLocked(workloadName, patternName string) string {
	key := workloadName + "|" + patternName
	if versions := s.patternVersions[key]; len(versions) > 0 {
		for _, v := range versions {
			if name := launchWizardPayloadString(v, "deploymentPatternVersionName", "workloadVersionName"); name != "" {
				return name
			}
		}
	}
	return "v1"
}

func (s *launchWizardStore) firstDeploymentIDLocked() string {
	for k := range s.deployments {
		return k
	}
	return "deployment-000001"
}

func (s *launchWizardStore) firstDeploymentARNLocked() string {
	for _, d := range s.deployments {
		if arn := launchWizardPayloadString(d, "deploymentArn", "resourceArn"); arn != "" {
			return arn
		}
	}
	return "arn:aws:launchwizard:us-east-1:123456789012:deployment/deployment-000001"
}

func (s *launchWizardStore) ensureWorkloadLocked(workloadName string) map[string]any {
	if w := s.workloads[workloadName]; w != nil {
		return w
	}
	w := map[string]any{
		"workloadName":        workloadName,
		"workloadDisplayName": strings.ReplaceAll(strings.ToUpper(workloadName), "_", " "),
		"description":         "Auto-created workload",
		"status":              "ACTIVE",
	}
	s.workloads[workloadName] = w
	return w
}

func (s *launchWizardStore) ensurePatternLocked(workloadName, patternName string) map[string]any {
	patterns := s.workloadPatterns[workloadName]
	if patterns == nil {
		patterns = map[string]map[string]any{}
		s.workloadPatterns[workloadName] = patterns
	}
	if p := patterns[patternName]; p != nil {
		return p
	}
	p := map[string]any{
		"workloadName":                 workloadName,
		"deploymentPatternName":        patternName,
		"deploymentPatternDisplayName": strings.ReplaceAll(patternName, "-", " "),
		"description":                  "Auto-created deployment pattern",
		"status":                       "ACTIVE",
	}
	patterns[patternName] = p
	return p
}

func (s *launchWizardStore) ensurePatternVersionLocked(workloadName, patternName, versionName, now string) map[string]any {
	key := workloadName + "|" + patternName
	versions := s.patternVersions[key]
	for _, v := range versions {
		if launchWizardPayloadString(v, "deploymentPatternVersionName", "workloadVersionName") == versionName {
			return v
		}
	}
	v := map[string]any{
		"workloadName":                   workloadName,
		"deploymentPatternName":          patternName,
		"deploymentPatternVersionName":   versionName,
		"workloadVersionName":            versionName,
		"status":                         "ACTIVE",
		"createdAt":                      now,
		"deploymentSpecificationsFields": []any{},
	}
	s.patternVersions[key] = append(versions, v)
	return v
}

func (s *launchWizardStore) ensureDeploymentByIDLocked(deploymentID, now string) map[string]any {
	if deploymentID == "" {
		deploymentID = s.firstDeploymentIDLocked()
	}
	if d := s.deployments[deploymentID]; d != nil {
		return d
	}
	arn := "arn:aws:launchwizard:us-east-1:123456789012:deployment/" + deploymentID
	d := map[string]any{
		"deploymentId":                  deploymentID,
		"deploymentArn":                 arn,
		"name":                          "auto-created-deployment",
		"status":                        "DEPLOYED",
		"workloadName":                  s.firstWorkloadNameLocked(),
		"workloadDeploymentPatternName": s.firstPatternNameLocked(s.firstWorkloadNameLocked()),
		"workloadVersionName":           s.firstPatternVersionLocked(s.firstWorkloadNameLocked(), s.firstPatternNameLocked(s.firstWorkloadNameLocked())),
		"createdAt":                     now,
		"updatedAt":                     now,
	}
	s.deployments[deploymentID] = d
	return d
}

func (s *launchWizardStore) deploymentIdentifier(payload map[string]any, query url.Values) string {
	deploymentID := launchWizardFirstNonEmpty(
		launchWizardPayloadString(payload, "deploymentId", "DeploymentId"),
		strings.TrimSpace(query.Get("deploymentId")),
	)
	if deploymentID != "" {
		return deploymentID
	}
	arn := launchWizardFirstNonEmpty(
		launchWizardPayloadString(payload, "deploymentArn", "resourceArn", "ResourceArn"),
		strings.TrimSpace(query.Get("resourceArn")),
	)
	if arn != "" {
		for id, d := range s.deployments {
			if launchWizardPayloadString(d, "deploymentArn") == arn {
				return id
			}
		}
	}
	return s.firstDeploymentIDLocked()
}

func (s *launchWizardStore) ensureTagsLocked(resourceARN string) map[string]string {
	t := s.tags[resourceARN]
	if t == nil {
		t = map[string]string{}
		s.tags[resourceARN] = t
	}
	return t
}

func launchWizardExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if m, ok := payload["tags"].(map[string]any); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	if m, ok := payload["Tags"].(map[string]any); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func launchWizardExtractTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	appendFrom := func(v any) {
		if list, ok := v.([]any); ok {
			for _, item := range list {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					keys = append(keys, strings.TrimSpace(s))
				}
			}
		}
	}
	appendFrom(payload["tagKeys"])
	appendFrom(payload["TagKeys"])
	if len(keys) == 0 {
		for _, k := range query["tagKeys"] {
			for _, p := range strings.Split(k, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					keys = append(keys, p)
				}
			}
		}
	}
	if len(keys) == 0 {
		keys = append(keys, "env")
	}
	return keys
}

func launchWizardPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func launchWizardFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func launchWizardCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		switch t := v.(type) {
		case map[string]any:
			out[k] = launchWizardCloneMap(t)
		case []any:
			c := make([]any, len(t))
			copy(c, t)
			out[k] = c
		default:
			out[k] = t
		}
	}
	return out
}

func launchWizardCloneListOfMaps(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, launchWizardCloneMap(item))
	}
	return out
}

func launchWizardSortedMaps(src map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, src[k])
	}
	return out
}

func launchWizardDeploymentSummary(in map[string]any) map[string]any {
	return map[string]any{
		"deploymentId":                  launchWizardPayloadString(in, "deploymentId"),
		"deploymentArn":                 launchWizardPayloadString(in, "deploymentArn"),
		"name":                          launchWizardPayloadString(in, "name"),
		"status":                        launchWizardPayloadString(in, "status"),
		"workloadName":                  launchWizardPayloadString(in, "workloadName"),
		"workloadDeploymentPatternName": launchWizardPayloadString(in, "workloadDeploymentPatternName"),
		"workloadVersionName":           launchWizardPayloadString(in, "workloadVersionName"),
	}
}

func launchWizardWorkloadSummary(in map[string]any) map[string]any {
	return map[string]any{
		"workloadName":        launchWizardPayloadString(in, "workloadName"),
		"workloadDisplayName": launchWizardPayloadString(in, "workloadDisplayName"),
		"status":              launchWizardPayloadString(in, "status"),
	}
}

func launchWizardPatternSummary(in map[string]any) map[string]any {
	return map[string]any{
		"workloadName":                 launchWizardPayloadString(in, "workloadName"),
		"deploymentPatternName":        launchWizardPayloadString(in, "deploymentPatternName"),
		"deploymentPatternDisplayName": launchWizardPayloadString(in, "deploymentPatternDisplayName"),
		"status":                       launchWizardPayloadString(in, "status"),
	}
}
