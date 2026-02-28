package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type groundStationStore struct {
	mu sync.Mutex

	nextConfigID                int64
	nextDataflowEndpointGroupID int64
	nextEphemerisID             int64
	nextMissionProfileID        int64
	nextContactID               int64
	nextAgentID                 int64

	configs                map[string]map[string]map[string]any
	dataflowEndpointGroups map[string]map[string]any
	ephemerides            map[string]map[string]any
	missionProfiles        map[string]map[string]any
	contacts               map[string]map[string]any
	agents                 map[string]map[string]any
	satellites             map[string]map[string]any
	groundStations         map[string]map[string]any
	tags                   map[string]map[string]string
}

func newGroundStationStore() *groundStationStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &groundStationStore{
		nextConfigID:                2,
		nextDataflowEndpointGroupID: 2,
		nextEphemerisID:             2,
		nextMissionProfileID:        2,
		nextContactID:               2,
		nextAgentID:                 2,
		configs:                     map[string]map[string]map[string]any{},
		dataflowEndpointGroups:      map[string]map[string]any{},
		ephemerides:                 map[string]map[string]any{},
		missionProfiles:             map[string]map[string]any{},
		contacts:                    map[string]map[string]any{},
		agents:                      map[string]map[string]any{},
		satellites:                  map[string]map[string]any{},
		groundStations:              map[string]map[string]any{},
		tags:                        map[string]map[string]string{},
	}

	cfg := s.ensureConfigLocked("antenna-downlink", "cfg-00000001", now)
	dfg := s.ensureDataflowEndpointGroupLocked("deg-00000001", now)
	eph := s.ensureEphemerisLocked("eph-00000001", now)
	mp := s.ensureMissionProfileLocked("mp-00000001", now)
	contact := s.ensureContactLocked("contact-00000001", now)
	agent := s.ensureAgentLocked("agent-00000001", now)

	s.satellites["25544"] = map[string]any{
		"satelliteId":      "25544",
		"noradSatelliteID": 25544,
		"name":             "ISS (ZARYA)",
	}
	s.groundStations["us-east-1-1"] = map[string]any{
		"groundStationId":   "us-east-1-1",
		"groundStationName": "stackyard-east-1",
		"region":            "us-east-1",
	}

	s.tags[groundStationStringAny(cfg, "configArn")] = map[string]string{"seed": "true"}
	s.tags[groundStationStringAny(dfg, "dataflowEndpointGroupArn")] = map[string]string{"seed": "true"}
	s.tags[groundStationStringAny(eph, "ephemerisArn")] = map[string]string{"seed": "true"}
	s.tags[groundStationStringAny(mp, "missionProfileArn")] = map[string]string{"seed": "true"}
	s.tags[groundStationStringAny(contact, "contactArn")] = map[string]string{"seed": "true"}
	s.tags[groundStationStringAny(agent, "agentArn")] = map[string]string{"seed": "true"}

	return s
}

func (s *groundStationStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	configType := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "configType"),
		groundStationStringAny(payload, "configType"),
		"antenna-downlink",
	)
	configID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "configId"),
		groundStationStringAny(payload, "configId"),
		"cfg-00000001",
	)
	dataflowEndpointGroupID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "dataflowEndpointGroupId"),
		groundStationStringAny(payload, "dataflowEndpointGroupId"),
		"deg-00000001",
	)
	ephemerisID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "ephemerisId"),
		groundStationStringAny(payload, "ephemerisId"),
		"eph-00000001",
	)
	missionProfileID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "missionProfileId"),
		groundStationStringAny(payload, "missionProfileId"),
		"mp-00000001",
	)
	contactID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "contactId"),
		groundStationStringAny(payload, "contactId"),
		"contact-00000001",
	)
	agentID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "agentId"),
		groundStationStringAny(payload, "agentId"),
		"agent-00000001",
	)
	satelliteID := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "satelliteId"),
		groundStationStringAny(payload, "satelliteId"),
		"25544",
	)
	resourceARN := groundStationFirstNonEmpty(
		groundStationPath(pathParams, "resourceArn"),
		groundStationStringAny(payload, "resourceArn"),
		groundStationStringAny(payload, "ResourceArn"),
		groundStationMissionProfileARN(missionProfileID),
	)

	s.mergeQueryIntoPayload(payload, query)

	switch action {
	case "CreateConfig":
		configType = groundStationFirstNonEmpty(configType, groundStationDetectConfigType(payload))
		configID = fmt.Sprintf("cfg-%08d", s.nextConfigIDLocked())
		cfg := s.ensureConfigLocked(configType, configID, now)
		groundStationMergeMap(cfg, payload)
		cfg["updatedAt"] = now
		return groundStationCloneMap(cfg)

	case "GetConfig", "UpdateConfig", "DeleteConfig":
		cfg := s.ensureConfigLocked(configType, configID, now)
		if action == "UpdateConfig" {
			groundStationMergeMap(cfg, payload)
			cfg["updatedAt"] = now
		}
		if action == "DeleteConfig" {
			cfg["status"] = "DELETED"
			cfg["updatedAt"] = now
			delete(s.ensureConfigTypeMapLocked(configType), configID)
		}
		return groundStationCloneMap(cfg)

	case "ListConfigs":
		items := make([]any, 0)
		for _, configTypeMap := range s.configs {
			for _, cfg := range configTypeMap {
				items = append(items, groundStationCloneMap(cfg))
			}
		}
		groundStationSortByStringField(items, "configId")
		return map[string]any{"configList": items, "nextToken": ""}

	case "CreateDataflowEndpointGroup", "CreateDataflowEndpointGroupV2":
		dataflowEndpointGroupID = fmt.Sprintf("deg-%08d", s.nextDataflowEndpointGroupIDLocked())
		dfg := s.ensureDataflowEndpointGroupLocked(dataflowEndpointGroupID, now)
		groundStationMergeMap(dfg, payload)
		dfg["updatedAt"] = now
		return map[string]any{
			"dataflowEndpointGroupId":  dataflowEndpointGroupID,
			"dataflowEndpointGroupArn": groundStationStringAny(dfg, "dataflowEndpointGroupArn"),
		}

	case "GetDataflowEndpointGroup", "DeleteDataflowEndpointGroup":
		dfg := s.ensureDataflowEndpointGroupLocked(dataflowEndpointGroupID, now)
		if action == "DeleteDataflowEndpointGroup" {
			dfg["status"] = "DELETED"
			dfg["updatedAt"] = now
			delete(s.dataflowEndpointGroups, dataflowEndpointGroupID)
		}
		return groundStationCloneMap(dfg)

	case "ListDataflowEndpointGroups":
		items := make([]any, 0, len(s.dataflowEndpointGroups))
		for _, dfg := range s.dataflowEndpointGroups {
			items = append(items, groundStationCloneMap(dfg))
		}
		groundStationSortByStringField(items, "dataflowEndpointGroupId")
		return map[string]any{"dataflowEndpointGroupList": items, "nextToken": ""}

	case "CreateEphemeris":
		ephemerisID = fmt.Sprintf("eph-%08d", s.nextEphemerisIDLocked())
		eph := s.ensureEphemerisLocked(ephemerisID, now)
		groundStationMergeMap(eph, payload)
		eph["updatedAt"] = now
		return map[string]any{"ephemerisId": ephemerisID, "ephemerisArn": groundStationStringAny(eph, "ephemerisArn")}

	case "DescribeEphemeris", "UpdateEphemeris", "DeleteEphemeris":
		eph := s.ensureEphemerisLocked(ephemerisID, now)
		if action == "UpdateEphemeris" {
			groundStationMergeMap(eph, payload)
			eph["updatedAt"] = now
		}
		if action == "DeleteEphemeris" {
			eph["status"] = "DELETED"
			eph["updatedAt"] = now
			delete(s.ephemerides, ephemerisID)
		}
		return groundStationCloneMap(eph)

	case "ListEphemerides":
		items := make([]any, 0, len(s.ephemerides))
		for _, eph := range s.ephemerides {
			items = append(items, groundStationCloneMap(eph))
		}
		groundStationSortByStringField(items, "ephemerisId")
		return map[string]any{"ephemerides": items, "nextToken": ""}

	case "CreateMissionProfile":
		missionProfileID = fmt.Sprintf("mp-%08d", s.nextMissionProfileIDLocked())
		mp := s.ensureMissionProfileLocked(missionProfileID, now)
		groundStationMergeMap(mp, payload)
		mp["updatedAt"] = now
		return map[string]any{"missionProfileId": missionProfileID, "missionProfileArn": groundStationStringAny(mp, "missionProfileArn")}

	case "GetMissionProfile", "UpdateMissionProfile", "DeleteMissionProfile":
		mp := s.ensureMissionProfileLocked(missionProfileID, now)
		if action == "UpdateMissionProfile" {
			groundStationMergeMap(mp, payload)
			mp["updatedAt"] = now
		}
		if action == "DeleteMissionProfile" {
			mp["status"] = "DELETED"
			mp["updatedAt"] = now
			delete(s.missionProfiles, missionProfileID)
		}
		return groundStationCloneMap(mp)

	case "ListMissionProfiles":
		items := make([]any, 0, len(s.missionProfiles))
		for _, mp := range s.missionProfiles {
			items = append(items, groundStationCloneMap(mp))
		}
		groundStationSortByStringField(items, "missionProfileId")
		return map[string]any{"missionProfileList": items, "nextToken": ""}

	case "ReserveContact":
		contactID = fmt.Sprintf("contact-%08d", s.nextContactIDLocked())
		contact := s.ensureContactLocked(contactID, now)
		groundStationMergeMap(contact, payload)
		contact["contactStatus"] = "SCHEDULED"
		contact["updatedAt"] = now
		return map[string]any{"contactId": contactID}

	case "DescribeContact", "CancelContact":
		contact := s.ensureContactLocked(contactID, now)
		if action == "CancelContact" {
			contact["contactStatus"] = "CANCELLED"
			contact["updatedAt"] = now
		}
		return groundStationCloneMap(contact)

	case "ListContacts":
		items := make([]any, 0, len(s.contacts))
		for _, contact := range s.contacts {
			items = append(items, groundStationCloneMap(contact))
		}
		groundStationSortByStringField(items, "contactId")
		return map[string]any{"contactList": items, "nextToken": ""}

	case "GetMinuteUsage":
		return map[string]any{
			"totalReservedMinuteUsage":  42,
			"totalScheduledMinuteUsage": 21,
			"estimatedMinutesRemaining": 1024,
			"isReserved":                true,
		}

	case "RegisterAgent":
		agentID = fmt.Sprintf("agent-%08d", s.nextAgentIDLocked())
		agent := s.ensureAgentLocked(agentID, now)
		groundStationMergeMap(agent, payload)
		agent["agentStatus"] = "ACTIVE"
		agent["updatedAt"] = now
		return map[string]any{"agentId": agentID, "agentArn": groundStationStringAny(agent, "agentArn")}

	case "GetAgentConfiguration":
		agent := s.ensureAgentLocked(agentID, now)
		return map[string]any{
			"agentId":             agentID,
			"agentStatus":         groundStationFirstNonEmpty(groundStationStringAny(agent, "agentStatus"), "ACTIVE"),
			"taskingDocument":     "{}",
			"missionProfileNames": []string{groundStationMissionProfileIDForARN(groundStationStringAny(agent, "missionProfileArn"))},
		}

	case "GetAgentTaskResponseUrl":
		s.ensureAgentLocked(agentID, now)
		return map[string]any{
			"agentId":              agentID,
			"agentTaskResponseUrl": fmt.Sprintf("https://stackyard.local/groundstation/agent/%s/task-response", agentID),
		}

	case "UpdateAgentStatus":
		agent := s.ensureAgentLocked(agentID, now)
		status := groundStationFirstNonEmpty(groundStationStringAny(payload, "agentStatus"), "ACTIVE")
		agent["agentStatus"] = status
		agent["updatedAt"] = now
		return map[string]any{"agentId": agentID, "agentStatus": status}

	case "GetSatellite":
		sat := s.ensureSatelliteLocked(satelliteID)
		return groundStationCloneMap(sat)

	case "ListSatellites":
		items := make([]any, 0, len(s.satellites))
		for _, sat := range s.satellites {
			items = append(items, groundStationCloneMap(sat))
		}
		groundStationSortByStringField(items, "satelliteId")
		return map[string]any{"satellites": items, "nextToken": ""}

	case "ListGroundStations":
		items := make([]any, 0, len(s.groundStations))
		for _, gs := range s.groundStations {
			items = append(items, groundStationCloneMap(gs))
		}
		groundStationSortByStringField(items, "groundStationId")
		return map[string]any{"groundStationList": items, "nextToken": ""}

	case "TagResource":
		tags := s.ensureTagMapLocked(resourceARN)
		for k, v := range groundStationReadTags(payload) {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagMapLocked(resourceARN)
		for _, key := range groundStationReadTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		tags := s.ensureTagMapLocked(resourceARN)
		return map[string]any{"tags": groundStationCloneStringMap(tags)}
	}

	return map[string]any{}
}

func (s *groundStationStore) ensureConfigTypeMapLocked(configType string) map[string]map[string]any {
	configType = groundStationFirstNonEmpty(strings.TrimSpace(configType), "antenna-downlink")
	configsByType, ok := s.configs[configType]
	if !ok {
		configsByType = map[string]map[string]any{}
		s.configs[configType] = configsByType
	}
	return configsByType
}

func (s *groundStationStore) ensureConfigLocked(configType, configID, now string) map[string]any {
	configType = groundStationFirstNonEmpty(strings.TrimSpace(configType), "antenna-downlink")
	configID = groundStationFirstNonEmpty(strings.TrimSpace(configID), "cfg-00000001")
	configsByType := s.ensureConfigTypeMapLocked(configType)
	cfg, ok := configsByType[configID]
	if !ok {
		cfg = map[string]any{
			"configId":   configID,
			"configType": configType,
			"name":       "stackyard-config-" + configID,
			"configArn":  groundStationConfigARN(configType, configID),
			"status":     "ACTIVE",
			"createdAt":  now,
			"updatedAt":  now,
		}
		configsByType[configID] = cfg
	}
	return cfg
}

func (s *groundStationStore) ensureDataflowEndpointGroupLocked(id, now string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "deg-00000001")
	dfg, ok := s.dataflowEndpointGroups[id]
	if !ok {
		dfg = map[string]any{
			"dataflowEndpointGroupId":        id,
			"dataflowEndpointGroupArn":       groundStationDataflowEndpointGroupARN(id),
			"endpointDetails":                []any{},
			"contactPrePassDurationSeconds":  120,
			"contactPostPassDurationSeconds": 60,
			"status":                         "ACTIVE",
			"createdAt":                      now,
			"updatedAt":                      now,
		}
		s.dataflowEndpointGroups[id] = dfg
	}
	return dfg
}

func (s *groundStationStore) ensureEphemerisLocked(id, now string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "eph-00000001")
	eph, ok := s.ephemerides[id]
	if !ok {
		eph = map[string]any{
			"ephemerisId":  id,
			"ephemerisArn": groundStationEphemerisARN(id),
			"name":         "stackyard-ephemeris-" + id,
			"status":       "ENABLED",
			"priority":     1,
			"createdAt":    now,
			"updatedAt":    now,
		}
		s.ephemerides[id] = eph
	}
	return eph
}

func (s *groundStationStore) ensureMissionProfileLocked(id, now string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "mp-00000001")
	mp, ok := s.missionProfiles[id]
	if !ok {
		mp = map[string]any{
			"missionProfileId":                    id,
			"missionProfileArn":                   groundStationMissionProfileARN(id),
			"name":                                "stackyard-mission-profile-" + id,
			"trackingConfigArn":                   "arn:aws:groundstation:us-east-1:123456789012:config/tracking/trk-00000001",
			"minimumViableContactDurationSeconds": 60,
			"status":                              "ACTIVE",
			"createdAt":                           now,
			"updatedAt":                           now,
		}
		s.missionProfiles[id] = mp
	}
	return mp
}

func (s *groundStationStore) ensureContactLocked(id, now string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "contact-00000001")
	contact, ok := s.contacts[id]
	if !ok {
		contact = map[string]any{
			"contactId":          id,
			"contactArn":         groundStationContactARN(id),
			"contactStatus":      "SCHEDULED",
			"groundStation":      "us-east-1-1",
			"missionProfileArn":  groundStationMissionProfileARN("mp-00000001"),
			"satelliteArn":       "arn:aws:groundstation:us-east-1:123456789012:satellite/25544",
			"scheduledStartTime": now,
			"scheduledEndTime":   now,
			"createdAt":          now,
			"updatedAt":          now,
		}
		s.contacts[id] = contact
	}
	return contact
}

func (s *groundStationStore) ensureAgentLocked(id, now string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "agent-00000001")
	agent, ok := s.agents[id]
	if !ok {
		agent = map[string]any{
			"agentId":           id,
			"agentArn":          groundStationAgentARN(id),
			"agentStatus":       "ACTIVE",
			"missionProfileArn": groundStationMissionProfileARN("mp-00000001"),
			"instanceType":      "c5.large",
			"createdAt":         now,
			"updatedAt":         now,
		}
		s.agents[id] = agent
	}
	return agent
}

func (s *groundStationStore) ensureSatelliteLocked(id string) map[string]any {
	id = groundStationFirstNonEmpty(strings.TrimSpace(id), "25544")
	sat, ok := s.satellites[id]
	if !ok {
		sat = map[string]any{
			"satelliteId":      id,
			"noradSatelliteID": 25544,
			"name":             "stackyard-satellite-" + id,
		}
		s.satellites[id] = sat
	}
	return sat
}

func (s *groundStationStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = groundStationFirstNonEmpty(strings.TrimSpace(resourceARN), groundStationMissionProfileARN("mp-00000001"))
	tags, ok := s.tags[resourceARN]
	if !ok {
		tags = map[string]string{}
		s.tags[resourceARN] = tags
	}
	return tags
}

func (s *groundStationStore) nextConfigIDLocked() int64 {
	id := s.nextConfigID
	s.nextConfigID++
	return id
}

func (s *groundStationStore) nextDataflowEndpointGroupIDLocked() int64 {
	id := s.nextDataflowEndpointGroupID
	s.nextDataflowEndpointGroupID++
	return id
}

func (s *groundStationStore) nextEphemerisIDLocked() int64 {
	id := s.nextEphemerisID
	s.nextEphemerisID++
	return id
}

func (s *groundStationStore) nextMissionProfileIDLocked() int64 {
	id := s.nextMissionProfileID
	s.nextMissionProfileID++
	return id
}

func (s *groundStationStore) nextContactIDLocked() int64 {
	id := s.nextContactID
	s.nextContactID++
	return id
}

func (s *groundStationStore) nextAgentIDLocked() int64 {
	id := s.nextAgentID
	s.nextAgentID++
	return id
}

func (s *groundStationStore) mergeQueryIntoPayload(payload map[string]any, query url.Values) {
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if _, ok := payload[key]; ok {
			continue
		}
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		list := make([]any, 0, len(values))
		for _, value := range values {
			list = append(list, value)
		}
		payload[key] = list
	}
}

func groundStationReadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case map[string]any:
			for k, v := range typed {
				value := strings.TrimSpace(fmt.Sprintf("%v", v))
				if strings.TrimSpace(k) != "" {
					out[strings.TrimSpace(k)] = value
				}
			}
		case map[string]string:
			for k, v := range typed {
				if strings.TrimSpace(k) != "" {
					out[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
		}
	}
	return out
}

func groundStationReadTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if raw, ok := payload[key]; ok {
			switch typed := raw.(type) {
			case []any:
				for _, item := range typed {
					value := strings.TrimSpace(fmt.Sprintf("%v", item))
					if value != "" {
						keys = append(keys, value)
					}
				}
			case []string:
				for _, item := range typed {
					value := strings.TrimSpace(item)
					if value != "" {
						keys = append(keys, value)
					}
				}
			case string:
				value := strings.TrimSpace(typed)
				if value != "" {
					keys = append(keys, value)
				}
			}
		}
	}
	for _, value := range query["tagKeys"] {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				keys = append(keys, part)
			}
		}
	}
	return keys
}

func groundStationDetectConfigType(payload map[string]any) string {
	if cfgDataRaw, ok := payload["configData"]; ok {
		if cfgData, ok := cfgDataRaw.(map[string]any); ok {
			for key := range cfgData {
				if strings.TrimSpace(key) != "" {
					return strings.TrimSpace(key)
				}
			}
		}
	}
	return ""
}

func groundStationSortByStringField(items []any, field string) {
	sort.SliceStable(items, func(i, j int) bool {
		mi, okI := items[i].(map[string]any)
		mj, okJ := items[j].(map[string]any)
		if !okI || !okJ {
			return i < j
		}
		return groundStationStringAny(mi, field) < groundStationStringAny(mj, field)
	})
}

func groundStationCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = groundStationCloneMap(typed)
		case []any:
			list := make([]any, len(typed))
			for i, item := range typed {
				if m, ok := item.(map[string]any); ok {
					list[i] = groundStationCloneMap(m)
					continue
				}
				list[i] = item
			}
			out[k] = list
		default:
			out[k] = v
		}
	}
	return out
}

func groundStationCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func groundStationMergeMap(dst map[string]any, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func groundStationStringAny(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		value := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if value != "" {
			return value
		}
	}
	return ""
}

func groundStationPath(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func groundStationFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func groundStationConfigARN(configType, configID string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:config/%s/%s", configType, configID)
}

func groundStationDataflowEndpointGroupARN(id string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:dataflow-endpoint-group/%s", id)
}

func groundStationEphemerisARN(id string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:ephemeris/%s", id)
}

func groundStationMissionProfileARN(id string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:mission-profile/%s", id)
}

func groundStationContactARN(id string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:contact/%s", id)
}

func groundStationAgentARN(id string) string {
	return fmt.Sprintf("arn:aws:groundstation:us-east-1:123456789012:agent/%s", id)
}

func groundStationMissionProfileIDForARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "mp-00000001"
	}
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return "mp-00000001"
	}
	id := strings.TrimSpace(parts[len(parts)-1])
	if id == "" {
		return "mp-00000001"
	}
	return id
}
