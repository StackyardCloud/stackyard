package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type gameliftStreamsGroup struct {
	id          string
	description string
	streamClass string
	createdAt   time.Time
	updatedAt   time.Time

	applications map[string]struct{}
	locations    map[string]struct{}
}

type gameliftStreamsStore struct {
	mu sync.Mutex

	nextID int64

	applications map[string]map[string]any
	streamGroups map[string]*gameliftStreamsGroup
	sessions     map[string]map[string]map[string]any
	tags         map[string]map[string]string
}

func newGameLiftStreamsStore() *gameliftStreamsStore {
	s := &gameliftStreamsStore{
		nextID:       2,
		applications: map[string]map[string]any{},
		streamGroups: map[string]*gameliftStreamsGroup{},
		sessions:     map[string]map[string]map[string]any{},
		tags:         map[string]map[string]string{},
	}

	seedApp := s.ensureApplicationLocked("app-000001")
	seedGroup := s.ensureStreamGroupLocked("sg-000001")
	seedGroup.applications[seedApp["identifier"].(string)] = struct{}{}
	s.ensureSessionLocked("sg-000001", "ss-000001", "app-000001")
	s.tags[gameliftStreamsStreamGroupARN("sg-000001")] = map[string]string{
		"seed":    "true",
		"service": "gameliftstreams",
	}

	return s
}

func (s *gameliftStreamsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := gameliftStreamsMergeMaps(payload, pathParams, query)
	identifier := gameliftStreamsString(ctx, "identifier", "sg-000001")
	streamSessionID := gameliftStreamsString(ctx, "streamSessionIdentifier", "ss-000001")
	applicationID := gameliftStreamsString(ctx, "applicationIdentifier", "app-000001")
	resourceARN := gameliftStreamsString(ctx, "resourceArn", gameliftStreamsStreamGroupARN(identifier))

	switch action {
	case "ListApplications":
		return map[string]any{"items": s.listApplicationsLocked(), "nextToken": ""}
	case "CreateApplication":
		id := gameliftStreamsString(ctx, "identifier", "")
		if id == "" {
			id = s.nextIdentifierLocked("app")
		}
		app := s.ensureApplicationLocked(id)
		if desc := gameliftStreamsString(ctx, "description", ""); desc != "" {
			app["description"] = desc
		}
		app["lastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return gameliftStreamsCloneAnyMap(app)
	case "GetApplication":
		return gameliftStreamsCloneAnyMap(s.ensureApplicationLocked(identifier))
	case "UpdateApplication":
		app := s.ensureApplicationLocked(identifier)
		if desc := gameliftStreamsString(ctx, "description", ""); desc != "" {
			app["description"] = desc
		}
		app["lastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return gameliftStreamsCloneAnyMap(app)
	case "DeleteApplication":
		delete(s.applications, strings.TrimSpace(identifier))
		for _, group := range s.streamGroups {
			delete(group.applications, strings.TrimSpace(identifier))
		}
		return map[string]any{}

	case "ListStreamGroups":
		return map[string]any{"items": s.listStreamGroupsLocked(), "nextToken": ""}
	case "CreateStreamGroup":
		id := gameliftStreamsString(ctx, "identifier", "")
		if id == "" {
			id = s.nextIdentifierLocked("sg")
		}
		group := s.ensureStreamGroupLocked(id)
		if desc := gameliftStreamsString(ctx, "description", ""); desc != "" {
			group.description = desc
		}
		if streamClass := gameliftStreamsString(ctx, "streamClass", ""); streamClass != "" {
			group.streamClass = streamClass
		}
		if defaultApp := gameliftStreamsString(ctx, "defaultApplicationIdentifier", ""); defaultApp != "" {
			s.ensureApplicationLocked(defaultApp)
			group.applications[defaultApp] = struct{}{}
		}
		if locations := gameliftStreamsLocationNames(payload, query); len(locations) > 0 {
			group.locations = map[string]struct{}{}
			for _, location := range locations {
				group.locations[location] = struct{}{}
			}
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)
	case "GetStreamGroup":
		return s.streamGroupResponseLocked(s.ensureStreamGroupLocked(identifier))
	case "UpdateStreamGroup":
		group := s.ensureStreamGroupLocked(identifier)
		if desc := gameliftStreamsString(ctx, "description", ""); desc != "" {
			group.description = desc
		}
		if streamClass := gameliftStreamsString(ctx, "streamClass", ""); streamClass != "" {
			group.streamClass = streamClass
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)
	case "DeleteStreamGroup":
		delete(s.streamGroups, strings.TrimSpace(identifier))
		delete(s.sessions, strings.TrimSpace(identifier))
		return map[string]any{}

	case "AddStreamGroupLocations":
		group := s.ensureStreamGroupLocked(identifier)
		for _, location := range gameliftStreamsLocationNames(payload, query) {
			group.locations[location] = struct{}{}
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)
	case "RemoveStreamGroupLocations":
		group := s.ensureStreamGroupLocked(identifier)
		for _, location := range gameliftStreamsLocationNames(payload, query) {
			delete(group.locations, location)
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)
	case "AssociateApplications":
		group := s.ensureStreamGroupLocked(identifier)
		appIDs := gameliftStreamsApplicationIdentifiers(ctx)
		if len(appIDs) == 0 {
			appIDs = []string{"app-000001"}
		}
		for _, appID := range appIDs {
			s.ensureApplicationLocked(appID)
			group.applications[appID] = struct{}{}
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)
	case "DisassociateApplications":
		group := s.ensureStreamGroupLocked(identifier)
		for _, appID := range gameliftStreamsApplicationIdentifiers(ctx) {
			delete(group.applications, appID)
		}
		group.updatedAt = time.Now().UTC()
		return s.streamGroupResponseLocked(group)

	case "StartStreamSession":
		group := s.ensureStreamGroupLocked(identifier)
		if applicationID == "" {
			applicationID = s.defaultApplicationForGroupLocked(group)
		}
		s.ensureApplicationLocked(applicationID)
		group.applications[applicationID] = struct{}{}
		if streamSessionID == "" {
			streamSessionID = s.nextIdentifierLocked("ss")
		}
		session := s.ensureSessionLocked(group.id, streamSessionID, applicationID)
		if desc := gameliftStreamsString(ctx, "description", ""); desc != "" {
			session["description"] = desc
		}
		if protocol := gameliftStreamsString(ctx, "protocol", ""); protocol != "" {
			session["protocol"] = protocol
		}
		if signal := gameliftStreamsString(ctx, "signalRequest", ""); signal != "" {
			session["signalRequest"] = signal
		}
		session["status"] = "ACTIVE"
		session["lastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return gameliftStreamsCloneAnyMap(session)
	case "GetStreamSession":
		session := s.ensureSessionLocked(identifier, streamSessionID, applicationID)
		return gameliftStreamsCloneAnyMap(session)
	case "TerminateStreamSession":
		session := s.ensureSessionLocked(identifier, streamSessionID, applicationID)
		session["status"] = "TERMINATED"
		session["lastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}
	case "ListStreamSessions":
		return map[string]any{"items": s.listStreamSessionsForGroupLocked(identifier), "nextToken": ""}
	case "ListStreamSessionsByAccount":
		return map[string]any{"items": s.listAllStreamSessionsLocked(), "nextToken": ""}
	case "CreateStreamSessionConnection":
		session := s.ensureSessionLocked(identifier, streamSessionID, applicationID)
		return map[string]any{
			"streamSessionIdentifier": session["streamSessionIdentifier"],
			"connectionToken":         fmt.Sprintf("conn-%s", strings.TrimSpace(streamSessionID)),
			"signalRequest":           gameliftStreamsString(ctx, "signalRequest", "OFFER"),
			"webSdkProtocolUrl":       fmt.Sprintf("wss://%s.stackyard.gameliftstreams.local/connect/%s", strings.TrimSpace(identifier), strings.TrimSpace(streamSessionID)),
		}
	case "ExportStreamSessionFiles":
		session := s.ensureSessionLocked(identifier, streamSessionID, applicationID)
		session["exportFilesMetadata"] = map[string]any{
			"status":    "COMPLETED",
			"outputUri": gameliftStreamsString(ctx, "outputUri", "s3://stackyard-gameliftstreams/exports/"),
		}
		session["lastUpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return gameliftStreamsCloneAnyMap(session)

	case "TagResource":
		existing := s.ensureTagsLocked(resourceARN)
		for key, value := range gameliftStreamsMapString(payload["tags"]) {
			existing[key] = value
		}
		for key, value := range gameliftStreamsMapString(payload["Tags"]) {
			existing[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		existing := s.ensureTagsLocked(resourceARN)
		for _, key := range gameliftStreamsTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": gameliftStreamsCloneStringMap(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *gameliftStreamsStore) nextIdentifierLocked(prefix string) string {
	id := fmt.Sprintf("%s-%06d", strings.TrimSpace(prefix), s.nextID)
	s.nextID++
	return id
}

func (s *gameliftStreamsStore) ensureApplicationLocked(identifier string) map[string]any {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "app-000001"
	}
	if existing := s.applications[id]; existing != nil {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	app := map[string]any{
		"id":                 id,
		"identifier":         id,
		"arn":                gameliftStreamsApplicationARN(id),
		"description":        fmt.Sprintf("stackyard application %s", id),
		"status":             "ACTIVE",
		"runtimeEnvironment": map[string]any{"type": "WINDOWS", "version": "2022"},
		"createdAt":          now,
		"lastUpdatedAt":      now,
	}
	s.applications[id] = app
	return app
}

func (s *gameliftStreamsStore) ensureStreamGroupLocked(identifier string) *gameliftStreamsGroup {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "sg-000001"
	}
	if existing := s.streamGroups[id]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	group := &gameliftStreamsGroup{
		id:           id,
		description:  fmt.Sprintf("stackyard stream group %s", id),
		streamClass:  "general-purpose",
		createdAt:    now,
		updatedAt:    now,
		applications: map[string]struct{}{"app-000001": {}},
		locations:    map[string]struct{}{"us-east-1": {}},
	}
	s.streamGroups[id] = group
	return group
}

func (s *gameliftStreamsStore) ensureSessionLocked(groupID, streamSessionID, applicationID string) map[string]any {
	group := s.ensureStreamGroupLocked(groupID)
	id := strings.TrimSpace(streamSessionID)
	if id == "" {
		id = "ss-000001"
	}
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = s.defaultApplicationForGroupLocked(group)
	}
	if appID == "" {
		appID = "app-000001"
	}
	s.ensureApplicationLocked(appID)
	group.applications[appID] = struct{}{}

	if s.sessions[group.id] == nil {
		s.sessions[group.id] = map[string]map[string]any{}
	}
	if existing := s.sessions[group.id][id]; existing != nil {
		return existing
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session := map[string]any{
		"id":                      id,
		"identifier":              id,
		"streamSessionIdentifier": id,
		"arn":                     gameliftStreamsSessionARN(id),
		"streamGroupIdentifier":   group.id,
		"applicationIdentifier":   appID,
		"status":                  "ACTIVE",
		"protocol":                "WebRTC",
		"signalRequest":           "OFFER",
		"createdAt":               now,
		"lastUpdatedAt":           now,
	}
	s.sessions[group.id][id] = session
	return session
}

func (s *gameliftStreamsStore) listApplicationsLocked() []any {
	ids := make([]string, 0, len(s.applications))
	for id := range s.applications {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, gameliftStreamsCloneAnyMap(s.applications[id]))
	}
	return out
}

func (s *gameliftStreamsStore) listStreamGroupsLocked() []any {
	ids := make([]string, 0, len(s.streamGroups))
	for id := range s.streamGroups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.streamGroupResponseLocked(s.streamGroups[id]))
	}
	return out
}

func (s *gameliftStreamsStore) listStreamSessionsForGroupLocked(groupID string) []any {
	group := s.ensureStreamGroupLocked(groupID)
	groupSessions := s.sessions[group.id]
	if groupSessions == nil {
		return []any{}
	}
	ids := make([]string, 0, len(groupSessions))
	for id := range groupSessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, gameliftStreamsCloneAnyMap(groupSessions[id]))
	}
	return out
}

func (s *gameliftStreamsStore) listAllStreamSessionsLocked() []any {
	groupIDs := make([]string, 0, len(s.sessions))
	for groupID := range s.sessions {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Strings(groupIDs)
	out := make([]any, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		ids := make([]string, 0, len(s.sessions[groupID]))
		for sessionID := range s.sessions[groupID] {
			ids = append(ids, sessionID)
		}
		sort.Strings(ids)
		for _, sessionID := range ids {
			out = append(out, gameliftStreamsCloneAnyMap(s.sessions[groupID][sessionID]))
		}
	}
	return out
}

func (s *gameliftStreamsStore) streamGroupResponseLocked(group *gameliftStreamsGroup) map[string]any {
	if group == nil {
		group = s.ensureStreamGroupLocked("sg-000001")
	}
	appIDs := make([]string, 0, len(group.applications))
	for appID := range group.applications {
		appIDs = append(appIDs, appID)
	}
	sort.Strings(appIDs)

	locations := make([]string, 0, len(group.locations))
	for location := range group.locations {
		locations = append(locations, location)
	}
	sort.Strings(locations)

	locationStates := make([]any, 0, len(locations))
	for _, location := range locations {
		locationStates = append(locationStates, map[string]any{
			"locationName": location,
			"status":       "ACTIVE",
		})
	}

	return map[string]any{
		"id":                     group.id,
		"identifier":             group.id,
		"arn":                    gameliftStreamsStreamGroupARN(group.id),
		"description":            group.description,
		"status":                 "ACTIVE",
		"streamClass":            group.streamClass,
		"applicationIdentifiers": appIDs,
		"locationStates":         locationStates,
		"createdAt":              group.createdAt.Format(time.RFC3339),
		"lastUpdatedAt":          group.updatedAt.Format(time.RFC3339),
	}
}

func (s *gameliftStreamsStore) defaultApplicationForGroupLocked(group *gameliftStreamsGroup) string {
	if group == nil {
		return "app-000001"
	}
	ids := make([]string, 0, len(group.applications))
	for appID := range group.applications {
		ids = append(ids, appID)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "app-000001"
	}
	return ids[0]
}

func (s *gameliftStreamsStore) ensureTagsLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = gameliftStreamsStreamGroupARN("sg-000001")
	}
	if existing := s.tags[arn]; existing != nil {
		return existing
	}
	s.tags[arn] = map[string]string{"service": "gameliftstreams"}
	return s.tags[arn]
}

func gameliftStreamsApplicationARN(identifier string) string {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "app-000001"
	}
	return fmt.Sprintf("arn:aws:gameliftstreams:us-east-1:123456789012:application/%s", id)
}

func gameliftStreamsStreamGroupARN(identifier string) string {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "sg-000001"
	}
	return fmt.Sprintf("arn:aws:gameliftstreams:us-east-1:123456789012:streamgroup/%s", id)
}

func gameliftStreamsSessionARN(identifier string) string {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "ss-000001"
	}
	return fmt.Sprintf("arn:aws:gameliftstreams:us-east-1:123456789012:streamsession/%s", id)
}

func gameliftStreamsMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			out[key] = values[0]
			continue
		}
		dup := make([]string, len(values))
		copy(dup, values)
		out[key] = dup
	}
	return out
}

func gameliftStreamsString(values map[string]any, key, def string) string {
	if values == nil {
		return def
	}
	for candidate, raw := range values {
		if !strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(key)) {
			continue
		}
		if raw == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			return text
		}
	}
	return def
}

func gameliftStreamsMapString(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = strings.TrimSpace(raw)
			}
		}
	case map[string]any:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = strings.TrimSpace(fmt.Sprint(raw))
			}
		}
	}
	return out
}

func gameliftStreamsStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			token := strings.TrimSpace(item)
			if token != "" {
				out = append(out, token)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if item == nil {
				continue
			}
			token := strings.TrimSpace(fmt.Sprint(item))
			if token != "" {
				out = append(out, token)
			}
		}
		return out
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			token := strings.TrimSpace(part)
			if token != "" {
				out = append(out, token)
			}
		}
		return out
	default:
		return nil
	}
}

func gameliftStreamsTagKeys(payload map[string]any, query url.Values) []string {
	if keys := gameliftStreamsStringSlice(payload["tagKeys"]); len(keys) > 0 {
		return keys
	}
	if keys := gameliftStreamsStringSlice(payload["TagKeys"]); len(keys) > 0 {
		return keys
	}
	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		if values := query[queryKey]; len(values) > 0 {
			return gameliftStreamsStringSlice(strings.Join(values, ","))
		}
	}
	return nil
}

func gameliftStreamsApplicationIdentifiers(values map[string]any) []string {
	for _, key := range []string{"applicationIdentifiers", "ApplicationIdentifiers"} {
		if ids := gameliftStreamsStringSlice(values[key]); len(ids) > 0 {
			return ids
		}
	}
	if id := gameliftStreamsString(values, "applicationIdentifier", ""); id != "" {
		return []string{id}
	}
	return nil
}

func gameliftStreamsLocationNames(payload map[string]any, query url.Values) []string {
	for _, key := range []string{"locations", "Locations"} {
		if locations := gameliftStreamsStringSlice(payload[key]); len(locations) > 0 {
			return locations
		}
	}
	for _, key := range []string{"locationName", "LocationName"} {
		if location := gameliftStreamsString(payload, key, ""); location != "" {
			return []string{location}
		}
	}
	if cfg := payload["locationConfigurations"]; cfg != nil {
		if locations := gameliftStreamsLocationNamesFromConfiguration(cfg); len(locations) > 0 {
			return locations
		}
	}
	if cfg := payload["LocationConfigurations"]; cfg != nil {
		if locations := gameliftStreamsLocationNamesFromConfiguration(cfg); len(locations) > 0 {
			return locations
		}
	}
	if values := query["locations"]; len(values) > 0 {
		return gameliftStreamsStringSlice(strings.Join(values, ","))
	}
	return nil
}

func gameliftStreamsLocationNamesFromConfiguration(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		location := gameliftStreamsString(entry, "locationName", "")
		if location != "" {
			out = append(out, location)
		}
	}
	return out
}

func gameliftStreamsCloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func gameliftStreamsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
