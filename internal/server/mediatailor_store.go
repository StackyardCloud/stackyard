package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type mediaTailorStore struct {
	mu sync.Mutex

	channels              map[string]map[string]any
	channelPolicies       map[string]string
	programs              map[string]map[string]any
	sourceLocations       map[string]map[string]any
	liveSources           map[string]map[string]any
	vodSources            map[string]map[string]any
	playbackConfigs       map[string]map[string]any
	prefetchSchedules     map[string]map[string]any
	tags                  map[string]map[string]string
	channelLogConfig      map[string]map[string]any
	playbackLogConfig     map[string]map[string]any
	seededAlerts          []map[string]any
	nextProgramOrdinal    int64
	nextPrefetchOrdinal   int64
	nextLiveSourceOrdinal int64
	nextVodSourceOrdinal  int64
}

func newMediaTailorStore() *mediaTailorStore {
	s := &mediaTailorStore{
		channels:              map[string]map[string]any{},
		channelPolicies:       map[string]string{},
		programs:              map[string]map[string]any{},
		sourceLocations:       map[string]map[string]any{},
		liveSources:           map[string]map[string]any{},
		vodSources:            map[string]map[string]any{},
		playbackConfigs:       map[string]map[string]any{},
		prefetchSchedules:     map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
		channelLogConfig:      map[string]map[string]any{},
		playbackLogConfig:     map[string]map[string]any{},
		seededAlerts:          []map[string]any{},
		nextProgramOrdinal:    2,
		nextPrefetchOrdinal:   2,
		nextLiveSourceOrdinal: 2,
		nextVodSourceOrdinal:  2,
	}

	channel := s.ensureChannelLocked("channel-00000001")
	s.ensureProgramLocked(mtStringAny(channel, "ChannelName"), "program-00000001")
	s.ensureSourceLocationLocked("source-location-00000001")
	s.ensureLiveSourceLocked("source-location-00000001", "live-source-00000001")
	s.ensureVodSourceLocked("source-location-00000001", "vod-source-00000001")
	s.ensurePlaybackConfigurationLocked("playback-config-00000001")
	s.ensurePrefetchScheduleLocked("playback-config-00000001", "prefetch-00000001")

	seedARN := mtARN("channel", "channel-00000001")
	s.tags[seedARN] = map[string]string{"seed": "true"}
	s.seededAlerts = []map[string]any{
		{
			"AlertCode":        "SEED_ALERT",
			"Message":          "Stackyard seeded MediaTailor alert",
			"ResourceArn":      seedARN,
			"LastModifiedTime": time.Now().UTC().Format(time.RFC3339),
		},
	}

	return s
}

func (s *mediaTailorStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	channelName := mtFirstNonEmpty(mtPath(pathParams, "ChannelName"), mtStringAny(payload, "ChannelName", "channelName"), "channel-00000001")
	programName := mtFirstNonEmpty(mtPath(pathParams, "ProgramName"), mtStringAny(payload, "ProgramName", "programName"), "program-00000001")
	sourceLocationName := mtFirstNonEmpty(mtPath(pathParams, "SourceLocationName"), mtStringAny(payload, "SourceLocationName", "sourceLocationName"), "source-location-00000001")
	liveSourceName := mtFirstNonEmpty(mtPath(pathParams, "LiveSourceName"), mtStringAny(payload, "LiveSourceName", "liveSourceName"), "live-source-00000001")
	vodSourceName := mtFirstNonEmpty(mtPath(pathParams, "VodSourceName"), mtStringAny(payload, "VodSourceName", "vodSourceName"), "vod-source-00000001")
	playbackName := mtFirstNonEmpty(
		mtPath(pathParams, "PlaybackConfigurationName"),
		mtPath(pathParams, "Name"),
		mtStringAny(payload, "PlaybackConfigurationName", "Name", "name"),
		"playback-config-00000001",
	)
	prefetchName := mtFirstNonEmpty(mtPath(pathParams, "Name"), mtStringAny(payload, "Name", "name"), "prefetch-00000001")
	resourceARN := mtFirstNonEmpty(mtPath(pathParams, "ResourceArn"), mtStringAny(payload, "ResourceArn", "resourceArn"), mtARN("channel", channelName))
	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateChannel":
		channel := s.ensureChannelLocked(channelName)
		for k, v := range payload {
			channel[k] = v
		}
		channel["ChannelName"] = channelName
		channel["Arn"] = mtARN("channel", channelName)
		channel["State"] = "STOPPED"
		channel["LastModifiedTime"] = now
		return mtCloneMap(channel)
	case "DescribeChannel":
		return mtCloneMap(s.ensureChannelLocked(channelName))
	case "UpdateChannel":
		channel := s.ensureChannelLocked(channelName)
		for k, v := range payload {
			channel[k] = v
		}
		channel["LastModifiedTime"] = now
		return mtCloneMap(channel)
	case "DeleteChannel":
		delete(s.channels, channelName)
		for key := range s.programs {
			if strings.HasPrefix(key, channelName+"/") {
				delete(s.programs, key)
			}
		}
		delete(s.channelPolicies, channelName)
		return map[string]any{"ChannelName": channelName}
	case "ListChannels":
		return map[string]any{"Items": s.listChannelsLocked(), "NextToken": ""}
	case "StartChannel":
		channel := s.ensureChannelLocked(channelName)
		channel["State"] = "RUNNING"
		channel["LastModifiedTime"] = now
		return map[string]any{"ChannelName": channelName, "State": "RUNNING"}
	case "StopChannel":
		channel := s.ensureChannelLocked(channelName)
		channel["State"] = "STOPPED"
		channel["LastModifiedTime"] = now
		return map[string]any{"ChannelName": channelName, "State": "STOPPED"}
	case "ConfigureLogsForChannel":
		cfg := mtCloneMap(payload)
		if len(cfg) == 0 {
			cfg = map[string]any{"LogTypes": []any{"AS_RUN"}}
		}
		s.channelLogConfig[channelName] = cfg
		return map[string]any{"ChannelName": channelName, "LogConfiguration": mtCloneMap(cfg)}
	case "PutChannelPolicy":
		policy := mtFirstNonEmpty(mtStringAny(payload, "Policy", "policy"), "{}")
		s.channelPolicies[channelName] = policy
		return map[string]any{"ChannelName": channelName, "Policy": policy}
	case "GetChannelPolicy":
		policy := s.channelPolicies[channelName]
		if strings.TrimSpace(policy) == "" {
			policy = "{}"
		}
		return map[string]any{"ChannelName": channelName, "Policy": policy}
	case "DeleteChannelPolicy":
		delete(s.channelPolicies, channelName)
		return map[string]any{"ChannelName": channelName}

	case "CreateProgram":
		program := s.ensureProgramLocked(channelName, programName)
		for k, v := range payload {
			program[k] = v
		}
		program["ChannelName"] = channelName
		program["ProgramName"] = programName
		program["Arn"] = mtARN("program", channelName+"/"+programName)
		program["LastModifiedTime"] = now
		return mtCloneMap(program)
	case "DescribeProgram":
		return mtCloneMap(s.ensureProgramLocked(channelName, programName))
	case "UpdateProgram":
		program := s.ensureProgramLocked(channelName, programName)
		for k, v := range payload {
			program[k] = v
		}
		program["LastModifiedTime"] = now
		return mtCloneMap(program)
	case "DeleteProgram":
		delete(s.programs, mtProgramKey(channelName, programName))
		return map[string]any{"ChannelName": channelName, "ProgramName": programName}
	case "GetChannelSchedule":
		return map[string]any{"Items": s.listProgramsForChannelLocked(channelName), "NextToken": ""}

	case "CreateSourceLocation":
		sl := s.ensureSourceLocationLocked(sourceLocationName)
		for k, v := range payload {
			sl[k] = v
		}
		sl["SourceLocationName"] = sourceLocationName
		sl["Arn"] = mtARN("sourceLocation", sourceLocationName)
		sl["LastModifiedTime"] = now
		return mtCloneMap(sl)
	case "DescribeSourceLocation":
		return mtCloneMap(s.ensureSourceLocationLocked(sourceLocationName))
	case "UpdateSourceLocation":
		sl := s.ensureSourceLocationLocked(sourceLocationName)
		for k, v := range payload {
			sl[k] = v
		}
		sl["LastModifiedTime"] = now
		return mtCloneMap(sl)
	case "DeleteSourceLocation":
		delete(s.sourceLocations, sourceLocationName)
		for key := range s.liveSources {
			if strings.HasPrefix(key, sourceLocationName+"/") {
				delete(s.liveSources, key)
			}
		}
		for key := range s.vodSources {
			if strings.HasPrefix(key, sourceLocationName+"/") {
				delete(s.vodSources, key)
			}
		}
		return map[string]any{"SourceLocationName": sourceLocationName}
	case "ListSourceLocations":
		return map[string]any{"Items": s.listSourceLocationsLocked(), "NextToken": ""}

	case "CreateLiveSource":
		ls := s.ensureLiveSourceLocked(sourceLocationName, liveSourceName)
		for k, v := range payload {
			ls[k] = v
		}
		ls["SourceLocationName"] = sourceLocationName
		ls["LiveSourceName"] = liveSourceName
		ls["Arn"] = mtARN("liveSource", sourceLocationName+"/"+liveSourceName)
		ls["LastModifiedTime"] = now
		return mtCloneMap(ls)
	case "DescribeLiveSource":
		return mtCloneMap(s.ensureLiveSourceLocked(sourceLocationName, liveSourceName))
	case "UpdateLiveSource":
		ls := s.ensureLiveSourceLocked(sourceLocationName, liveSourceName)
		for k, v := range payload {
			ls[k] = v
		}
		ls["LastModifiedTime"] = now
		return mtCloneMap(ls)
	case "DeleteLiveSource":
		delete(s.liveSources, mtNestedKey(sourceLocationName, liveSourceName))
		return map[string]any{"SourceLocationName": sourceLocationName, "LiveSourceName": liveSourceName}
	case "ListLiveSources":
		return map[string]any{"Items": s.listLiveSourcesLocked(sourceLocationName), "NextToken": ""}

	case "CreateVodSource":
		vs := s.ensureVodSourceLocked(sourceLocationName, vodSourceName)
		for k, v := range payload {
			vs[k] = v
		}
		vs["SourceLocationName"] = sourceLocationName
		vs["VodSourceName"] = vodSourceName
		vs["Arn"] = mtARN("vodSource", sourceLocationName+"/"+vodSourceName)
		vs["LastModifiedTime"] = now
		return mtCloneMap(vs)
	case "DescribeVodSource":
		return mtCloneMap(s.ensureVodSourceLocked(sourceLocationName, vodSourceName))
	case "UpdateVodSource":
		vs := s.ensureVodSourceLocked(sourceLocationName, vodSourceName)
		for k, v := range payload {
			vs[k] = v
		}
		vs["LastModifiedTime"] = now
		return mtCloneMap(vs)
	case "DeleteVodSource":
		delete(s.vodSources, mtNestedKey(sourceLocationName, vodSourceName))
		return map[string]any{"SourceLocationName": sourceLocationName, "VodSourceName": vodSourceName}
	case "ListVodSources":
		return map[string]any{"Items": s.listVodSourcesLocked(sourceLocationName), "NextToken": ""}

	case "PutPlaybackConfiguration":
		pc := s.ensurePlaybackConfigurationLocked(playbackName)
		for k, v := range payload {
			pc[k] = v
		}
		pc["Name"] = playbackName
		pc["Arn"] = mtARN("playbackConfiguration", playbackName)
		pc["LastModifiedTime"] = now
		return mtCloneMap(pc)
	case "GetPlaybackConfiguration":
		return mtCloneMap(s.ensurePlaybackConfigurationLocked(playbackName))
	case "DeletePlaybackConfiguration":
		delete(s.playbackConfigs, playbackName)
		for key := range s.prefetchSchedules {
			if strings.HasPrefix(key, playbackName+"/") {
				delete(s.prefetchSchedules, key)
			}
		}
		return map[string]any{"Name": playbackName}
	case "ListPlaybackConfigurations":
		return map[string]any{"Items": s.listPlaybackConfigurationsLocked(), "NextToken": ""}
	case "ConfigureLogsForPlaybackConfiguration":
		cfg := mtCloneMap(payload)
		if len(cfg) == 0 {
			cfg = map[string]any{"PercentEnabled": 100}
		}
		s.playbackLogConfig[playbackName] = cfg
		return map[string]any{"Name": playbackName, "LogConfiguration": mtCloneMap(cfg)}

	case "CreatePrefetchSchedule":
		schedule := s.ensurePrefetchScheduleLocked(playbackName, prefetchName)
		for k, v := range payload {
			schedule[k] = v
		}
		schedule["Name"] = prefetchName
		schedule["PlaybackConfigurationName"] = playbackName
		schedule["Arn"] = mtARN("prefetchSchedule", playbackName+"/"+prefetchName)
		schedule["LastModifiedTime"] = now
		return mtCloneMap(schedule)
	case "GetPrefetchSchedule":
		return mtCloneMap(s.ensurePrefetchScheduleLocked(playbackName, prefetchName))
	case "DeletePrefetchSchedule":
		delete(s.prefetchSchedules, mtNestedKey(playbackName, prefetchName))
		return map[string]any{"PlaybackConfigurationName": playbackName, "Name": prefetchName}
	case "ListPrefetchSchedules":
		return map[string]any{"Items": s.listPrefetchSchedulesLocked(playbackName), "NextToken": ""}

	case "ListAlerts":
		items := make([]map[string]any, 0, len(s.seededAlerts))
		for _, alert := range s.seededAlerts {
			items = append(items, mtCloneMap(alert))
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range mtReadTags(payload) {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range mtReadTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"Tags": mtCloneTags(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *mediaTailorStore) ensureChannelLocked(channelName string) map[string]any {
	if channelName == "" {
		channelName = "channel-00000001"
	}
	if item, ok := s.channels[channelName]; ok {
		return item
	}
	item := map[string]any{
		"ChannelName":      channelName,
		"Arn":              mtARN("channel", channelName),
		"State":            "STOPPED",
		"LastModifiedTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.channels[channelName] = item
	return item
}

func (s *mediaTailorStore) ensureProgramLocked(channelName, programName string) map[string]any {
	if channelName == "" {
		channelName = "channel-00000001"
	}
	if programName == "" {
		programName = fmt.Sprintf("program-%08d", s.nextProgramOrdinal)
		s.nextProgramOrdinal++
	}
	key := mtProgramKey(channelName, programName)
	if item, ok := s.programs[key]; ok {
		return item
	}
	item := map[string]any{
		"ChannelName":      channelName,
		"ProgramName":      programName,
		"Arn":              mtARN("program", channelName+"/"+programName),
		"LastModifiedTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.programs[key] = item
	return item
}

func (s *mediaTailorStore) ensureSourceLocationLocked(name string) map[string]any {
	if name == "" {
		name = "source-location-00000001"
	}
	if item, ok := s.sourceLocations[name]; ok {
		return item
	}
	item := map[string]any{
		"SourceLocationName": name,
		"Arn":                mtARN("sourceLocation", name),
		"LastModifiedTime":   time.Now().UTC().Format(time.RFC3339),
	}
	s.sourceLocations[name] = item
	return item
}

func (s *mediaTailorStore) ensureLiveSourceLocked(sourceLocationName, liveSourceName string) map[string]any {
	s.ensureSourceLocationLocked(sourceLocationName)
	if liveSourceName == "" {
		liveSourceName = fmt.Sprintf("live-source-%08d", s.nextLiveSourceOrdinal)
		s.nextLiveSourceOrdinal++
	}
	key := mtNestedKey(sourceLocationName, liveSourceName)
	if item, ok := s.liveSources[key]; ok {
		return item
	}
	item := map[string]any{
		"SourceLocationName": sourceLocationName,
		"LiveSourceName":     liveSourceName,
		"Arn":                mtARN("liveSource", sourceLocationName+"/"+liveSourceName),
		"LastModifiedTime":   time.Now().UTC().Format(time.RFC3339),
	}
	s.liveSources[key] = item
	return item
}

func (s *mediaTailorStore) ensureVodSourceLocked(sourceLocationName, vodSourceName string) map[string]any {
	s.ensureSourceLocationLocked(sourceLocationName)
	if vodSourceName == "" {
		vodSourceName = fmt.Sprintf("vod-source-%08d", s.nextVodSourceOrdinal)
		s.nextVodSourceOrdinal++
	}
	key := mtNestedKey(sourceLocationName, vodSourceName)
	if item, ok := s.vodSources[key]; ok {
		return item
	}
	item := map[string]any{
		"SourceLocationName": sourceLocationName,
		"VodSourceName":      vodSourceName,
		"Arn":                mtARN("vodSource", sourceLocationName+"/"+vodSourceName),
		"LastModifiedTime":   time.Now().UTC().Format(time.RFC3339),
	}
	s.vodSources[key] = item
	return item
}

func (s *mediaTailorStore) ensurePlaybackConfigurationLocked(name string) map[string]any {
	if name == "" {
		name = "playback-config-00000001"
	}
	if item, ok := s.playbackConfigs[name]; ok {
		return item
	}
	item := map[string]any{
		"Name":             name,
		"Arn":              mtARN("playbackConfiguration", name),
		"LastModifiedTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.playbackConfigs[name] = item
	return item
}

func (s *mediaTailorStore) ensurePrefetchScheduleLocked(playbackName, prefetchName string) map[string]any {
	s.ensurePlaybackConfigurationLocked(playbackName)
	if prefetchName == "" {
		prefetchName = fmt.Sprintf("prefetch-%08d", s.nextPrefetchOrdinal)
		s.nextPrefetchOrdinal++
	}
	key := mtNestedKey(playbackName, prefetchName)
	if item, ok := s.prefetchSchedules[key]; ok {
		return item
	}
	item := map[string]any{
		"PlaybackConfigurationName": playbackName,
		"Name":                      prefetchName,
		"Arn":                       mtARN("prefetchSchedule", playbackName+"/"+prefetchName),
		"LastModifiedTime":          time.Now().UTC().Format(time.RFC3339),
	}
	s.prefetchSchedules[key] = item
	return item
}

func (s *mediaTailorStore) ensureTagsLocked(resourceARN string) map[string]string {
	if resourceARN == "" {
		resourceARN = mtARN("channel", "channel-00000001")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *mediaTailorStore) listChannelsLocked() []map[string]any {
	names := make([]string, 0, len(s.channels))
	for name := range s.channels {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, mtCloneMap(s.channels[name]))
	}
	return out
}

func (s *mediaTailorStore) listProgramsForChannelLocked(channelName string) []map[string]any {
	keys := make([]string, 0)
	prefix := channelName + "/"
	for key := range s.programs {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, mtCloneMap(s.programs[key]))
	}
	return out
}

func (s *mediaTailorStore) listSourceLocationsLocked() []map[string]any {
	names := make([]string, 0, len(s.sourceLocations))
	for name := range s.sourceLocations {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, mtCloneMap(s.sourceLocations[name]))
	}
	return out
}

func (s *mediaTailorStore) listLiveSourcesLocked(sourceLocationName string) []map[string]any {
	keys := make([]string, 0)
	prefix := sourceLocationName + "/"
	for key := range s.liveSources {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, mtCloneMap(s.liveSources[key]))
	}
	return out
}

func (s *mediaTailorStore) listVodSourcesLocked(sourceLocationName string) []map[string]any {
	keys := make([]string, 0)
	prefix := sourceLocationName + "/"
	for key := range s.vodSources {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, mtCloneMap(s.vodSources[key]))
	}
	return out
}

func (s *mediaTailorStore) listPlaybackConfigurationsLocked() []map[string]any {
	names := make([]string, 0, len(s.playbackConfigs))
	for name := range s.playbackConfigs {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, mtCloneMap(s.playbackConfigs[name]))
	}
	return out
}

func (s *mediaTailorStore) listPrefetchSchedulesLocked(playbackName string) []map[string]any {
	keys := make([]string, 0)
	prefix := playbackName + "/"
	for key := range s.prefetchSchedules {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, mtCloneMap(s.prefetchSchedules[key]))
	}
	return out
}

func mtProgramKey(channelName, programName string) string {
	return mtNestedKey(channelName, programName)
}

func mtNestedKey(left, right string) string {
	return strings.TrimSpace(left) + "/" + strings.TrimSpace(right)
}

func mtARN(kind, name string) string {
	kind = strings.Trim(strings.TrimSpace(kind), "/")
	name = strings.Trim(strings.TrimSpace(name), "/")
	if kind == "" {
		kind = "resource"
	}
	if name == "" {
		name = "stackyard"
	}
	return fmt.Sprintf("arn:aws:mediatailor:us-east-1:123456789012:%s/%s", kind, name)
}

func mtPath(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func mtStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			continue
		}
		if v, ok := m[key]; ok {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func mtFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func mtCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mtReadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"Tags", "tags"} {
		if payload == nil {
			continue
		}
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch t := raw.(type) {
		case map[string]any:
			for k, v := range t {
				out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case map[string]string:
			for k, v := range t {
				out[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	return out
}

func mtReadTagKeys(payload map[string]any) []string {
	if payload == nil {
		return []string{"env"}
	}
	for _, key := range []string{"TagKeys", "tagKeys"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch t := raw.(type) {
		case []any:
			out := make([]string, 0, len(t))
			for _, item := range t {
				itemStr := strings.TrimSpace(fmt.Sprintf("%v", item))
				if itemStr != "" {
					out = append(out, itemStr)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			out := make([]string, 0, len(t))
			for _, item := range t {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return []string{"env"}
}

func mtCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
