package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	simSpaceWeaverDefaultRegion    = "us-east-1"
	simSpaceWeaverDefaultAccountID = "123456789012"
)

type simSpaceWeaverStore struct {
	mu sync.Mutex

	nextSnapshot int64

	simulations map[string]map[string]any
	apps        map[string]map[string]any
	tags        map[string]map[string]string
}

func newSimSpaceWeaverStore() *simSpaceWeaverStore {
	s := &simSpaceWeaverStore{
		nextSnapshot: 1,
		simulations:  map[string]map[string]any{},
		apps:         map[string]map[string]any{},
		tags:         map[string]map[string]string{},
	}
	now := time.Now().UTC()
	sim := s.ensureSimulationLocked("stackyard-sim-0001", now)
	app := s.ensureAppLocked("stackyard-sim-0001", "stackyard-domain", "stackyard-app-0001", now)
	s.ensureTagsLocked(simSpaceWeaverString(sim, []string{"arn"}, ""))
	s.ensureTagsLocked(simSpaceWeaverString(app, []string{"arn"}, ""))
	return s
}

func (s *simSpaceWeaverStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := simSpaceWeaverMergeContext(payload, pathParams, query)

	simulationName := simSpaceWeaverString(ctx, []string{"simulation", "Simulation", "name", "Name"}, "stackyard-sim-0001")
	domainName := simSpaceWeaverString(ctx, []string{"domain", "Domain"}, "stackyard-domain")
	appName := simSpaceWeaverString(ctx, []string{"app", "App", "name", "Name"}, "stackyard-app-0001")
	resourceARN := simSpaceWeaverString(ctx, []string{"ResourceArn", "resourceArn", "resourceARN"}, simSpaceWeaverSimulationARN(simulationName))

	switch action {
	case "StartSimulation":
		simName := simSpaceWeaverString(ctx, []string{"Name", "name", "simulation", "Simulation"}, simulationName)
		sim := s.ensureSimulationLocked(simName, now)
		sim["status"] = "RUNNING"
		sim["executionStatus"] = "STARTED"
		sim["lastUpdatedTime"] = now.Format(time.RFC3339)
		if role := simSpaceWeaverString(payload, []string{"RoleArn", "roleArn"}, ""); role != "" {
			sim["roleArn"] = role
		}
		if schema, ok := simSpaceWeaverAny(payload, []string{"SchemaS3Location", "schemaS3Location"}); ok {
			sim["schemaS3Location"] = schema
		}
		return map[string]any{
			"arn":          sim["arn"],
			"name":         sim["name"],
			"status":       sim["status"],
			"creationTime": sim["creationTime"],
		}

	case "StopSimulation":
		sim := s.ensureSimulationLocked(simulationName, now)
		sim["status"] = "STOPPED"
		sim["executionStatus"] = "STOPPED"
		sim["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "DeleteSimulation":
		s.deleteSimulationLocked(simulationName)
		return map[string]any{}

	case "DescribeSimulation":
		sim := s.ensureSimulationLocked(simulationName, now)
		return simSpaceWeaverCloneMap(sim)

	case "ListSimulations":
		items := []any{}
		for _, name := range s.sortedSimulationNamesLocked() {
			items = append(items, simSpaceWeaverCloneMap(s.simulations[name]))
		}
		return map[string]any{
			"simulations": items,
			"nextToken":   "",
		}

	case "CreateSnapshot":
		simName := simSpaceWeaverString(ctx, []string{"simulation", "Simulation", "name", "Name"}, simulationName)
		sim := s.ensureSimulationLocked(simName, now)
		snapshotName := fmt.Sprintf("snapshot-%06d", s.nextSnapshot)
		s.nextSnapshot++
		return map[string]any{
			"snapshot": map[string]any{
				"arn":          simSpaceWeaverSnapshotARN(snapshotName),
				"name":         snapshotName,
				"simulation":   sim["name"],
				"creationTime": now.Format(time.RFC3339),
			},
		}

	case "StartApp":
		simName := simSpaceWeaverString(ctx, []string{"simulation", "Simulation"}, simulationName)
		domain := simSpaceWeaverString(ctx, []string{"domain", "Domain"}, domainName)
		app := simSpaceWeaverString(ctx, []string{"name", "Name", "app", "App"}, appName)
		s.ensureSimulationLocked(simName, now)
		appMeta := s.ensureAppLocked(simName, domain, app, now)
		appMeta["status"] = "RUNNING"
		appMeta["targetStatus"] = "STARTED"
		appMeta["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"arn":    appMeta["arn"],
			"name":   appMeta["name"],
			"status": appMeta["status"],
		}

	case "StopApp":
		meta := s.ensureOrFindAppLocked(simulationName, domainName, appName, now)
		meta["status"] = "STOPPED"
		meta["targetStatus"] = "STOPPED"
		meta["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "DeleteApp":
		s.deleteAppLocked(simulationName, domainName, appName)
		return map[string]any{}

	case "DescribeApp":
		meta := s.ensureOrFindAppLocked(simulationName, domainName, appName, now)
		return simSpaceWeaverCloneMap(meta)

	case "ListApps":
		items := []any{}
		for _, key := range s.sortedAppKeysLocked() {
			item := s.apps[key]
			if item == nil {
				continue
			}
			if simulationName != "" && simulationName != "stackyard-sim-0001" {
				if simSpaceWeaverString(item, []string{"simulation"}, "") != simulationName {
					continue
				}
			}
			if domainName != "" && domainName != "stackyard-domain" {
				if simSpaceWeaverString(item, []string{"domain"}, "") != domainName {
					continue
				}
			}
			items = append(items, simSpaceWeaverCloneMap(item))
		}
		if len(items) == 0 {
			items = append(items, simSpaceWeaverCloneMap(s.ensureAppLocked(simulationName, domainName, appName, now)))
		}
		return map[string]any{
			"apps":      items,
			"nextToken": "",
		}

	case "StartClock":
		sim := s.ensureSimulationLocked(simulationName, now)
		sim["clockStatus"] = "RUNNING"
		sim["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"clock": map[string]any{
				"status":         "RUNNING",
				"simulationTime": now.Format(time.RFC3339),
			},
		}

	case "StopClock":
		sim := s.ensureSimulationLocked(simulationName, now)
		sim["clockStatus"] = "STOPPED"
		sim["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"clock": map[string]any{
				"status":         "STOPPED",
				"simulationTime": now.Format(time.RFC3339),
			},
		}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range simSpaceWeaverMapString(ctx["tags"]) {
			tags[key] = value
		}
		for key, value := range simSpaceWeaverMapString(ctx["Tags"]) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, tagKey := range simSpaceWeaverStringSlice(ctx["tagKeys"]) {
			delete(tags, tagKey)
		}
		for _, tagKey := range query["tagKeys"] {
			delete(tags, strings.TrimSpace(tagKey))
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{
			"tags": simSpaceWeaverCloneMapString(s.ensureTagsLocked(resourceARN)),
		}
	}

	return map[string]any{}
}

func (s *simSpaceWeaverStore) ensureSimulationLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-sim-0001"
	}
	if sim := s.simulations[name]; sim != nil {
		return sim
	}
	sim := map[string]any{
		"name":            name,
		"arn":             simSpaceWeaverSimulationARN(name),
		"status":          "STOPPED",
		"executionStatus": "STOPPED",
		"clockStatus":     "STOPPED",
		"creationTime":    now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
		"roleArn":         "arn:aws:iam::123456789012:role/stackyard-simspaceweaver",
		"schemaS3Location": map[string]any{
			"BucketName": "stackyard-simspaceweaver",
			"ObjectKey":  "schemas/simulation-schema.zip",
		},
	}
	s.simulations[name] = sim
	return sim
}

func (s *simSpaceWeaverStore) ensureAppLocked(simulation, domain, name string, now time.Time) map[string]any {
	simulation = strings.TrimSpace(simulation)
	if simulation == "" {
		simulation = "stackyard-sim-0001"
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "stackyard-domain"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-app-0001"
	}

	key := simSpaceWeaverAppKey(simulation, domain, name)
	if app := s.apps[key]; app != nil {
		return app
	}
	app := map[string]any{
		"name":            name,
		"domain":          domain,
		"simulation":      simulation,
		"arn":             simSpaceWeaverAppARN(simulation, name),
		"status":          "STOPPED",
		"targetStatus":    "STOPPED",
		"launchStatus":    "COMPLETED",
		"creationTime":    now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
		"endpointInfo": []any{
			map[string]any{
				"address": "127.0.0.1",
				"ingressPortMappings": []any{
					map[string]any{
						"source":      8080,
						"destination": 8080,
						"protocol":    "TCP",
					},
				},
			},
		},
	}
	s.apps[key] = app
	return app
}

func (s *simSpaceWeaverStore) ensureOrFindAppLocked(simulation, domain, name string, now time.Time) map[string]any {
	simulation = strings.TrimSpace(simulation)
	domain = strings.TrimSpace(domain)
	name = strings.TrimSpace(name)

	if simulation != "" && domain != "" && name != "" {
		return s.ensureAppLocked(simulation, domain, name, now)
	}

	for _, app := range s.apps {
		if app == nil {
			continue
		}
		if name != "" && simSpaceWeaverString(app, []string{"name"}, "") != name {
			continue
		}
		if simulation != "" && simSpaceWeaverString(app, []string{"simulation"}, "") != simulation {
			continue
		}
		if domain != "" && simSpaceWeaverString(app, []string{"domain"}, "") != domain {
			continue
		}
		return app
	}
	return s.ensureAppLocked(simulation, domain, name, now)
}

func (s *simSpaceWeaverStore) deleteSimulationLocked(simulation string) {
	simulation = strings.TrimSpace(simulation)
	if simulation == "" {
		return
	}
	delete(s.simulations, simulation)
	for key, app := range s.apps {
		if app == nil {
			continue
		}
		if simSpaceWeaverString(app, []string{"simulation"}, "") == simulation {
			delete(s.apps, key)
		}
	}
}

func (s *simSpaceWeaverStore) deleteAppLocked(simulation, domain, name string) {
	simulation = strings.TrimSpace(simulation)
	domain = strings.TrimSpace(domain)
	name = strings.TrimSpace(name)

	if simulation != "" && domain != "" && name != "" {
		delete(s.apps, simSpaceWeaverAppKey(simulation, domain, name))
		return
	}

	for key, app := range s.apps {
		if app == nil {
			continue
		}
		if name != "" && simSpaceWeaverString(app, []string{"name"}, "") != name {
			continue
		}
		if simulation != "" && simSpaceWeaverString(app, []string{"simulation"}, "") != simulation {
			continue
		}
		if domain != "" && simSpaceWeaverString(app, []string{"domain"}, "") != domain {
			continue
		}
		delete(s.apps, key)
	}
}

func (s *simSpaceWeaverStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = simSpaceWeaverSimulationARN("stackyard-sim-0001")
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{
		"env":     "local",
		"service": "simspaceweaver",
	}
	s.tags[resourceARN] = tags
	return tags
}

func (s *simSpaceWeaverStore) sortedSimulationNamesLocked() []string {
	names := make([]string, 0, len(s.simulations))
	for name := range s.simulations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *simSpaceWeaverStore) sortedAppKeysLocked() []string {
	keys := make([]string, 0, len(s.apps))
	for key := range s.apps {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func simSpaceWeaverMergeContext(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
		out[strings.ToLower(strings.TrimSpace(key))] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			list := make([]any, 0, len(values))
			for _, value := range values {
				list = append(list, value)
			}
			out[key] = list
		}
	}
	return out
}

func simSpaceWeaverString(source map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		for sourceKey, raw := range source {
			if !strings.EqualFold(strings.TrimSpace(sourceKey), strings.TrimSpace(key)) {
				continue
			}
			switch value := raw.(type) {
			case string:
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					return trimmed
				}
			case fmt.Stringer:
				if trimmed := strings.TrimSpace(value.String()); trimmed != "" {
					return trimmed
				}
			case []any:
				for _, entry := range value {
					if text, ok := entry.(string); ok {
						if trimmed := strings.TrimSpace(text); trimmed != "" {
							return trimmed
						}
					}
				}
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func simSpaceWeaverAny(source map[string]any, keys []string) (any, bool) {
	for _, key := range keys {
		for sourceKey, raw := range source {
			if strings.EqualFold(strings.TrimSpace(sourceKey), strings.TrimSpace(key)) {
				return raw, true
			}
		}
	}
	return nil, false
}

func simSpaceWeaverStringSlice(raw any) []string {
	out := []string{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}

	switch value := raw.(type) {
	case string:
		for _, part := range strings.Split(value, ",") {
			appendValue(part)
		}
	case []string:
		for _, part := range value {
			appendValue(part)
		}
	case []any:
		for _, part := range value {
			if text, ok := part.(string); ok {
				appendValue(text)
			}
		}
	}
	return out
}

func simSpaceWeaverMapString(raw any) map[string]string {
	out := map[string]string{}
	switch value := raw.(type) {
	case map[string]string:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(entry)
		}
	case map[string]any:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text, _ := entry.(string)
			out[key] = strings.TrimSpace(text)
		}
	}
	return out
}

func simSpaceWeaverCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = simSpaceWeaverCloneAny(value)
	}
	return out
}

func simSpaceWeaverCloneAny(raw any) any {
	switch value := raw.(type) {
	case map[string]any:
		return simSpaceWeaverCloneMap(value)
	case map[string]string:
		return simSpaceWeaverCloneMapString(value)
	case []any:
		out := make([]any, 0, len(value))
		for _, entry := range value {
			out = append(out, simSpaceWeaverCloneAny(entry))
		}
		return out
	default:
		return raw
	}
}

func simSpaceWeaverCloneMapString(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func simSpaceWeaverSimulationARN(simulation string) string {
	simulation = strings.TrimSpace(simulation)
	if simulation == "" {
		simulation = "stackyard-sim-0001"
	}
	return fmt.Sprintf("arn:aws:simspaceweaver:%s:%s:simulation/%s", simSpaceWeaverDefaultRegion, simSpaceWeaverDefaultAccountID, simulation)
}

func simSpaceWeaverSnapshotARN(snapshot string) string {
	snapshot = strings.TrimSpace(snapshot)
	if snapshot == "" {
		snapshot = "snapshot-000001"
	}
	return fmt.Sprintf("arn:aws:simspaceweaver:%s:%s:snapshot/%s", simSpaceWeaverDefaultRegion, simSpaceWeaverDefaultAccountID, snapshot)
}

func simSpaceWeaverAppARN(simulation, app string) string {
	simulation = strings.TrimSpace(simulation)
	if simulation == "" {
		simulation = "stackyard-sim-0001"
	}
	app = strings.TrimSpace(app)
	if app == "" {
		app = "stackyard-app-0001"
	}
	return fmt.Sprintf(
		"arn:aws:simspaceweaver:%s:%s:simulation/%s/app/%s",
		simSpaceWeaverDefaultRegion,
		simSpaceWeaverDefaultAccountID,
		simulation,
		app,
	)
}

func simSpaceWeaverAppKey(simulation, domain, app string) string {
	return strings.ToLower(strings.TrimSpace(simulation)) + "|" + strings.ToLower(strings.TrimSpace(domain)) + "|" + strings.ToLower(strings.TrimSpace(app))
}
