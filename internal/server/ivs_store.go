package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type ivsStore struct {
	mu sync.Mutex

	nextChannelID         int64
	nextStreamKeyID       int64
	nextRecordingConfigID int64
	nextPlaybackKeyPairID int64
	nextPolicyID          int64
	nextSessionID         int64

	channels             map[string]map[string]any
	streamKeys           map[string]map[string]any
	recordingConfigs     map[string]map[string]any
	playbackKeyPairs     map[string]map[string]any
	playbackPolicies     map[string]map[string]any
	streams              map[string]map[string]any
	streamSessions       map[string][]map[string]any
	viewerRevocationJobs map[string]map[string]any
	tags                 map[string]map[string]string
}

func newIVSStore() *ivsStore {
	s := &ivsStore{
		nextChannelID:         2,
		nextStreamKeyID:       2,
		nextRecordingConfigID: 2,
		nextPlaybackKeyPairID: 2,
		nextPolicyID:          2,
		nextSessionID:         2,
		channels:              map[string]map[string]any{},
		streamKeys:            map[string]map[string]any{},
		recordingConfigs:      map[string]map[string]any{},
		playbackKeyPairs:      map[string]map[string]any{},
		playbackPolicies:      map[string]map[string]any{},
		streams:               map[string]map[string]any{},
		streamSessions:        map[string][]map[string]any{},
		viewerRevocationJobs:  map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
	}

	channel := s.ensureChannelLocked(ivsArn("channel", "channel-00000001"))
	s.ensureStreamKeyLocked(ivsArn("stream-key", "sk-00000001"), ivsStringAny(channel, "arn"))
	s.ensureRecordingConfigurationLocked(ivsArn("recording-configuration", "rc-00000001"))
	s.ensurePlaybackKeyPairLocked(ivsArn("playback-key-pair", "pk-00000001"))
	s.ensurePlaybackRestrictionPolicyLocked(ivsArn("playback-restriction-policy", "prp-00000001"))
	s.ensureStreamLocked(ivsStringAny(channel, "arn"))
	s.tags[ivsStringAny(channel, "arn")] = map[string]string{"seed": "true"}

	return s
}

func (s *ivsStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	channelARN := ivsFirstNonEmpty(
		ivsStringAny(payload, "channelArn", "arn"),
		s.firstChannelARNLocked(),
		ivsArn("channel", "channel-00000001"),
	)
	streamKeyARN := ivsFirstNonEmpty(
		ivsStringAny(payload, "streamKeyArn", "arn"),
		s.firstStreamKeyARNLocked(),
		ivsArn("stream-key", "sk-00000001"),
	)
	recordingConfigARN := ivsFirstNonEmpty(
		ivsStringAny(payload, "recordingConfigurationArn", "arn"),
		s.firstRecordingConfigurationARNLocked(),
		ivsArn("recording-configuration", "rc-00000001"),
	)
	playbackKeyPairARN := ivsFirstNonEmpty(
		ivsStringAny(payload, "arn", "playbackKeyPairArn"),
		s.firstPlaybackKeyPairARNLocked(),
		ivsArn("playback-key-pair", "pk-00000001"),
	)
	policyARN := ivsFirstNonEmpty(
		ivsStringAny(payload, "arn", "playbackRestrictionPolicyArn"),
		s.firstPlaybackRestrictionPolicyARNLocked(),
		ivsArn("playback-restriction-policy", "prp-00000001"),
	)
	resourceARN := ivsFirstNonEmpty(
		ivsPath(pathParams, "resourceArn"),
		ivsStringAny(payload, "resourceArn", "arn"),
		channelARN,
	)

	switch action {
	case "CreateChannel":
		channelARN = ivsArn("channel", fmt.Sprintf("channel-%08d", s.nextChannelIDLocked()))
		channel := s.ensureChannelLocked(channelARN)
		for k, v := range payload {
			channel[k] = v
		}
		channel["arn"] = channelARN
		channel["name"] = ivsFirstNonEmpty(ivsStringAny(payload, "name"), fmt.Sprintf("stackyard-channel-%d", s.nextChannelID-1))
		channel["latencyMode"] = ivsFirstNonEmpty(ivsStringAny(payload, "latencyMode"), "LOW")
		channel["authorized"] = ivsBoolAny(payload, "authorized")
		channel["recordingConfigurationArn"] = recordingConfigARN
		channel["tags"] = ivsCloneTags(s.ensureTagsLocked(channelARN))
		channel["updatedAt"] = now
		stream := s.ensureStreamLocked(channelARN)
		stream["state"] = "LIVE"
		stream["channelArn"] = channelARN
		return map[string]any{"channel": ivsCloneMap(channel)}

	case "GetChannel":
		return map[string]any{"channel": ivsCloneMap(s.ensureChannelLocked(channelARN))}
	case "BatchGetChannel":
		arns := ivsStringSliceAny(payload, "arns")
		if len(arns) == 0 {
			arns = []string{channelARN}
		}
		channels := make([]map[string]any, 0, len(arns))
		for _, arn := range arns {
			channels = append(channels, ivsCloneMap(s.ensureChannelLocked(arn)))
		}
		return map[string]any{"channels": channels, "errors": []any{}}
	case "ListChannels":
		return map[string]any{"channels": s.listChannelsLocked(), "nextToken": ""}
	case "UpdateChannel":
		channel := s.ensureChannelLocked(channelARN)
		for k, v := range payload {
			channel[k] = v
		}
		channel["updatedAt"] = now
		return map[string]any{"channel": ivsCloneMap(channel)}
	case "DeleteChannel":
		delete(s.channels, channelARN)
		delete(s.streams, channelARN)
		delete(s.streamSessions, channelARN)
		return map[string]any{}

	case "CreateStreamKey":
		streamKeyARN = ivsArn("stream-key", fmt.Sprintf("sk-%08d", s.nextStreamKeyIDLocked()))
		sk := s.ensureStreamKeyLocked(streamKeyARN, channelARN)
		sk["channelArn"] = channelARN
		sk["arn"] = streamKeyARN
		sk["value"] = ivsFirstNonEmpty(ivsStringAny(payload, "value"), fmt.Sprintf("sk_%d", s.nextStreamKeyID-1))
		sk["tags"] = ivsCloneTags(s.ensureTagsLocked(streamKeyARN))
		sk["updatedAt"] = now
		return map[string]any{"streamKey": ivsCloneMap(sk)}
	case "GetStreamKey":
		return map[string]any{"streamKey": ivsCloneMap(s.ensureStreamKeyLocked(streamKeyARN, channelARN))}
	case "BatchGetStreamKey":
		arns := ivsStringSliceAny(payload, "arns")
		if len(arns) == 0 {
			arns = []string{streamKeyARN}
		}
		keys := make([]map[string]any, 0, len(arns))
		for _, arn := range arns {
			keys = append(keys, ivsCloneMap(s.ensureStreamKeyLocked(arn, channelARN)))
		}
		return map[string]any{"streamKeys": keys, "errors": []any{}}
	case "ListStreamKeys":
		return map[string]any{"streamKeys": s.listStreamKeysLocked(channelARN), "nextToken": ""}
	case "DeleteStreamKey":
		delete(s.streamKeys, streamKeyARN)
		return map[string]any{}

	case "CreateRecordingConfiguration":
		recordingConfigARN = ivsArn("recording-configuration", fmt.Sprintf("rc-%08d", s.nextRecordingConfigIDLocked()))
		rc := s.ensureRecordingConfigurationLocked(recordingConfigARN)
		for k, v := range payload {
			rc[k] = v
		}
		rc["arn"] = recordingConfigARN
		rc["name"] = ivsFirstNonEmpty(ivsStringAny(payload, "name"), fmt.Sprintf("stackyard-recording-%d", s.nextRecordingConfigID-1))
		rc["updatedAt"] = now
		return map[string]any{"recordingConfiguration": ivsCloneMap(rc)}
	case "GetRecordingConfiguration":
		return map[string]any{"recordingConfiguration": ivsCloneMap(s.ensureRecordingConfigurationLocked(recordingConfigARN))}
	case "ListRecordingConfigurations":
		return map[string]any{"recordingConfigurations": s.listRecordingConfigurationsLocked(), "nextToken": ""}
	case "DeleteRecordingConfiguration":
		delete(s.recordingConfigs, recordingConfigARN)
		return map[string]any{}

	case "CreatePlaybackRestrictionPolicy":
		policyARN = ivsArn("playback-restriction-policy", fmt.Sprintf("prp-%08d", s.nextPolicyIDLocked()))
		policy := s.ensurePlaybackRestrictionPolicyLocked(policyARN)
		for k, v := range payload {
			policy[k] = v
		}
		policy["arn"] = policyARN
		policy["name"] = ivsFirstNonEmpty(ivsStringAny(payload, "name"), fmt.Sprintf("stackyard-policy-%d", s.nextPolicyID-1))
		policy["updatedAt"] = now
		return map[string]any{"playbackRestrictionPolicy": ivsCloneMap(policy)}
	case "GetPlaybackRestrictionPolicy":
		return map[string]any{"playbackRestrictionPolicy": ivsCloneMap(s.ensurePlaybackRestrictionPolicyLocked(policyARN))}
	case "ListPlaybackRestrictionPolicies":
		return map[string]any{"playbackRestrictionPolicies": s.listPlaybackRestrictionPoliciesLocked(), "nextToken": ""}
	case "UpdatePlaybackRestrictionPolicy":
		policy := s.ensurePlaybackRestrictionPolicyLocked(policyARN)
		for k, v := range payload {
			policy[k] = v
		}
		policy["updatedAt"] = now
		return map[string]any{"playbackRestrictionPolicy": ivsCloneMap(policy)}
	case "DeletePlaybackRestrictionPolicy":
		delete(s.playbackPolicies, policyARN)
		return map[string]any{}

	case "ImportPlaybackKeyPair":
		playbackKeyPairARN = ivsArn("playback-key-pair", fmt.Sprintf("pk-%08d", s.nextPlaybackKeyPairIDLocked()))
		pair := s.ensurePlaybackKeyPairLocked(playbackKeyPairARN)
		for k, v := range payload {
			pair[k] = v
		}
		pair["arn"] = playbackKeyPairARN
		pair["name"] = ivsFirstNonEmpty(ivsStringAny(payload, "name"), fmt.Sprintf("stackyard-keypair-%d", s.nextPlaybackKeyPairID-1))
		pair["fingerprint"] = ivsFirstNonEmpty(ivsStringAny(payload, "fingerprint"), fmt.Sprintf("fingerprint-%d", s.nextPlaybackKeyPairID-1))
		pair["updatedAt"] = now
		return map[string]any{"playbackKeyPair": ivsCloneMap(pair)}
	case "GetPlaybackKeyPair":
		return map[string]any{"keyPair": ivsCloneMap(s.ensurePlaybackKeyPairLocked(playbackKeyPairARN))}
	case "ListPlaybackKeyPairs":
		return map[string]any{"keyPairs": s.listPlaybackKeyPairsLocked(), "nextToken": ""}
	case "DeletePlaybackKeyPair":
		delete(s.playbackKeyPairs, playbackKeyPairARN)
		return map[string]any{}

	case "GetStream":
		stream := s.ensureStreamLocked(channelARN)
		return map[string]any{"stream": ivsCloneMap(stream)}
	case "ListStreams":
		return map[string]any{"streams": s.listStreamsLocked(), "nextToken": ""}
	case "StopStream":
		stream := s.ensureStreamLocked(channelARN)
		stream["state"] = "OFFLINE"
		stream["updatedAt"] = now
		return map[string]any{}

	case "GetStreamSession":
		session := s.ensureStreamSessionLocked(channelARN)
		return map[string]any{"streamSession": ivsCloneMap(session)}
	case "ListStreamSessions":
		return map[string]any{"streamSessions": s.listStreamSessionsLocked(channelARN), "nextToken": ""}

	case "PutMetadata":
		stream := s.ensureStreamLocked(channelARN)
		events, _ := stream["events"].([]any)
		event := map[string]any{
			"name":      "METADATA",
			"type":      "METADATA",
			"time":      now,
			"errorCode": "",
			"metadata":  ivsFirstNonEmpty(ivsStringAny(payload, "metadata"), "stackyard"),
		}
		events = append(events, event)
		stream["events"] = events
		stream["updatedAt"] = now
		return map[string]any{}

	case "StartViewerSessionRevocation":
		job := map[string]any{
			"channelArn":                             channelARN,
			"viewerId":                               ivsFirstNonEmpty(ivsStringAny(payload, "viewerId"), "viewer-00000001"),
			"viewerSessionVersionsLessThanOrEqualTo": ivsFirstNonEmpty(ivsStringAny(payload, "viewerSessionVersionsLessThanOrEqualTo"), "1"),
			"timestamp":                              now,
		}
		s.viewerRevocationJobs[fmt.Sprintf("%s/%s", channelARN, job["viewerId"])] = job
		return map[string]any{}
	case "BatchStartViewerSessionRevocation":
		sessions := ivsMapSliceAny(payload, "viewerSessions")
		errors := make([]map[string]any, 0)
		for _, session := range sessions {
			viewerID := ivsFirstNonEmpty(ivsStringAny(session, "viewerId"), "viewer-00000001")
			s.viewerRevocationJobs[fmt.Sprintf("%s/%s", channelARN, viewerID)] = map[string]any{
				"channelArn": channelARN,
				"viewerId":   viewerID,
				"timestamp":  now,
			}
		}
		return map[string]any{"errors": errors}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range ivsReadTags(payload) {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range ivsReadTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": ivsCloneTags(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *ivsStore) ensureChannelLocked(channelARN string) map[string]any {
	channelARN = strings.TrimSpace(channelARN)
	if channelARN == "" {
		channelARN = ivsArn("channel", "channel-00000001")
	}
	if item, ok := s.channels[channelARN]; ok {
		return item
	}
	item := map[string]any{
		"arn":                       channelARN,
		"name":                      ivsResourceID(channelARN),
		"latencyMode":               "LOW",
		"authorized":                false,
		"playbackUrl":               "https://stackyard.example.com/playback/" + ivsResourceID(channelARN) + ".m3u8",
		"ingestEndpoint":            "a1b2c3d4e5.global-contribute.live-video.net",
		"recordingConfigurationArn": ivsArn("recording-configuration", "rc-00000001"),
		"updatedAt":                 time.Now().UTC().Format(time.RFC3339),
	}
	s.channels[channelARN] = item
	return item
}

func (s *ivsStore) ensureStreamKeyLocked(streamKeyARN, channelARN string) map[string]any {
	streamKeyARN = strings.TrimSpace(streamKeyARN)
	if streamKeyARN == "" {
		streamKeyARN = ivsArn("stream-key", "sk-00000001")
	}
	if item, ok := s.streamKeys[streamKeyARN]; ok {
		return item
	}
	item := map[string]any{
		"arn":        streamKeyARN,
		"channelArn": channelARN,
		"value":      "sk_seed_00000001",
		"tags":       map[string]string{},
		"updatedAt":  time.Now().UTC().Format(time.RFC3339),
	}
	s.streamKeys[streamKeyARN] = item
	return item
}

func (s *ivsStore) ensureRecordingConfigurationLocked(recordingConfigARN string) map[string]any {
	recordingConfigARN = strings.TrimSpace(recordingConfigARN)
	if recordingConfigARN == "" {
		recordingConfigARN = ivsArn("recording-configuration", "rc-00000001")
	}
	if item, ok := s.recordingConfigs[recordingConfigARN]; ok {
		return item
	}
	item := map[string]any{
		"arn":       recordingConfigARN,
		"name":      ivsResourceID(recordingConfigARN),
		"state":     "ACTIVE",
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}
	s.recordingConfigs[recordingConfigARN] = item
	return item
}

func (s *ivsStore) ensurePlaybackKeyPairLocked(playbackKeyPairARN string) map[string]any {
	playbackKeyPairARN = strings.TrimSpace(playbackKeyPairARN)
	if playbackKeyPairARN == "" {
		playbackKeyPairARN = ivsArn("playback-key-pair", "pk-00000001")
	}
	if item, ok := s.playbackKeyPairs[playbackKeyPairARN]; ok {
		return item
	}
	item := map[string]any{
		"arn":         playbackKeyPairARN,
		"name":        ivsResourceID(playbackKeyPairARN),
		"fingerprint": "fingerprint-seed",
		"updatedAt":   time.Now().UTC().Format(time.RFC3339),
	}
	s.playbackKeyPairs[playbackKeyPairARN] = item
	return item
}

func (s *ivsStore) ensurePlaybackRestrictionPolicyLocked(policyARN string) map[string]any {
	policyARN = strings.TrimSpace(policyARN)
	if policyARN == "" {
		policyARN = ivsArn("playback-restriction-policy", "prp-00000001")
	}
	if item, ok := s.playbackPolicies[policyARN]; ok {
		return item
	}
	item := map[string]any{
		"arn":       policyARN,
		"name":      ivsResourceID(policyARN),
		"state":     "ACTIVE",
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}
	s.playbackPolicies[policyARN] = item
	return item
}

func (s *ivsStore) ensureStreamLocked(channelARN string) map[string]any {
	channelARN = ivsFirstNonEmpty(channelARN, s.firstChannelARNLocked(), ivsArn("channel", "channel-00000001"))
	if item, ok := s.streams[channelARN]; ok {
		return item
	}
	item := map[string]any{
		"channelArn":  channelARN,
		"state":       "LIVE",
		"health":      "HEALTHY",
		"viewerCount": 0,
		"sessionId":   fmt.Sprintf("session-%08d", s.nextSessionIDLocked()),
		"startTime":   time.Now().UTC().Format(time.RFC3339),
		"updatedAt":   time.Now().UTC().Format(time.RFC3339),
		"events":      []any{},
	}
	s.streams[channelARN] = item
	s.ensureStreamSessionLocked(channelARN)
	return item
}

func (s *ivsStore) ensureStreamSessionLocked(channelARN string) map[string]any {
	channelARN = ivsFirstNonEmpty(channelARN, s.firstChannelARNLocked(), ivsArn("channel", "channel-00000001"))
	sessions := s.streamSessions[channelARN]
	if len(sessions) > 0 {
		return sessions[0]
	}
	session := map[string]any{
		"channel":             map[string]any{"arn": channelARN},
		"sessionId":           fmt.Sprintf("session-%08d", s.nextSessionIDLocked()),
		"startTime":           time.Now().UTC().Format(time.RFC3339),
		"endTime":             "",
		"ingestConfiguration": map[string]any{"audio": true, "video": true},
	}
	s.streamSessions[channelARN] = []map[string]any{session}
	return session
}

func (s *ivsStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = ivsArn("channel", "channel-00000001")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *ivsStore) listChannelsLocked() []map[string]any {
	arns := make([]string, 0, len(s.channels))
	for arn := range s.channels {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, ivsCloneMap(s.channels[arn]))
	}
	return out
}

func (s *ivsStore) listStreamKeysLocked(channelARN string) []map[string]any {
	keys := make([]string, 0)
	for arn, item := range s.streamKeys {
		if channelARN == "" || ivsStringAny(item, "channelArn") == channelARN {
			keys = append(keys, arn)
		}
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, arn := range keys {
		out = append(out, ivsCloneMap(s.streamKeys[arn]))
	}
	return out
}

func (s *ivsStore) listRecordingConfigurationsLocked() []map[string]any {
	arns := make([]string, 0, len(s.recordingConfigs))
	for arn := range s.recordingConfigs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, ivsCloneMap(s.recordingConfigs[arn]))
	}
	return out
}

func (s *ivsStore) listPlaybackRestrictionPoliciesLocked() []map[string]any {
	arns := make([]string, 0, len(s.playbackPolicies))
	for arn := range s.playbackPolicies {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, ivsCloneMap(s.playbackPolicies[arn]))
	}
	return out
}

func (s *ivsStore) listPlaybackKeyPairsLocked() []map[string]any {
	arns := make([]string, 0, len(s.playbackKeyPairs))
	for arn := range s.playbackKeyPairs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, ivsCloneMap(s.playbackKeyPairs[arn]))
	}
	return out
}

func (s *ivsStore) listStreamsLocked() []map[string]any {
	arns := make([]string, 0, len(s.streams))
	for arn := range s.streams {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, ivsCloneMap(s.streams[arn]))
	}
	return out
}

func (s *ivsStore) listStreamSessionsLocked(channelARN string) []map[string]any {
	sessions := s.streamSessions[channelARN]
	out := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, ivsCloneMap(session))
	}
	if len(out) == 0 {
		out = append(out, ivsCloneMap(s.ensureStreamSessionLocked(channelARN)))
	}
	return out
}

func (s *ivsStore) firstChannelARNLocked() string {
	if len(s.channels) == 0 {
		return ""
	}
	arns := make([]string, 0, len(s.channels))
	for arn := range s.channels {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *ivsStore) firstStreamKeyARNLocked() string {
	if len(s.streamKeys) == 0 {
		return ""
	}
	arns := make([]string, 0, len(s.streamKeys))
	for arn := range s.streamKeys {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *ivsStore) firstRecordingConfigurationARNLocked() string {
	if len(s.recordingConfigs) == 0 {
		return ""
	}
	arns := make([]string, 0, len(s.recordingConfigs))
	for arn := range s.recordingConfigs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *ivsStore) firstPlaybackKeyPairARNLocked() string {
	if len(s.playbackKeyPairs) == 0 {
		return ""
	}
	arns := make([]string, 0, len(s.playbackKeyPairs))
	for arn := range s.playbackKeyPairs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *ivsStore) firstPlaybackRestrictionPolicyARNLocked() string {
	if len(s.playbackPolicies) == 0 {
		return ""
	}
	arns := make([]string, 0, len(s.playbackPolicies))
	for arn := range s.playbackPolicies {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *ivsStore) nextChannelIDLocked() int64 {
	id := s.nextChannelID
	s.nextChannelID++
	return id
}

func (s *ivsStore) nextStreamKeyIDLocked() int64 {
	id := s.nextStreamKeyID
	s.nextStreamKeyID++
	return id
}

func (s *ivsStore) nextRecordingConfigIDLocked() int64 {
	id := s.nextRecordingConfigID
	s.nextRecordingConfigID++
	return id
}

func (s *ivsStore) nextPlaybackKeyPairIDLocked() int64 {
	id := s.nextPlaybackKeyPairID
	s.nextPlaybackKeyPairID++
	return id
}

func (s *ivsStore) nextPolicyIDLocked() int64 {
	id := s.nextPolicyID
	s.nextPolicyID++
	return id
}

func (s *ivsStore) nextSessionIDLocked() int64 {
	id := s.nextSessionID
	s.nextSessionID++
	return id
}

func ivsArn(kind, id string) string {
	kind = strings.Trim(strings.TrimSpace(kind), "/")
	id = strings.Trim(strings.TrimSpace(id), "/")
	if kind == "" {
		kind = "resource"
	}
	if id == "" {
		id = "stackyard"
	}
	return fmt.Sprintf("arn:aws:ivs:us-east-1:123456789012:%s/%s", kind, id)
}

func ivsResourceID(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "stackyard"
	}
	parts := strings.Split(arn, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func ivsPath(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func ivsStringAny(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
				continue
			}
			s := strings.TrimSpace(fmt.Sprintf("%v", value))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func ivsBoolAny(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if value, ok := m[key]; ok {
		s := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", value)))
		return s == "1" || s == "true" || s == "yes"
	}
	return false
}

func ivsStringSliceAny(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			itemStr := strings.TrimSpace(fmt.Sprintf("%v", item))
			if itemStr != "" && itemStr != "<nil>" {
				out = append(out, itemStr)
			}
		}
		return out
	default:
		return nil
	}
}

func ivsMapSliceAny(m map[string]any, key string) []map[string]any {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if mm, ok := item.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

func ivsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func ivsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ivsReadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload["tags"]
	if !ok {
		raw, ok = payload["Tags"]
	}
	if !ok {
		return out
	}
	switch tags := raw.(type) {
	case map[string]string:
		for k, v := range tags {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	case map[string]any:
		for k, v := range tags {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return out
}

func ivsReadTagKeys(payload map[string]any) []string {
	if payload == nil {
		return []string{"env"}
	}
	raw, ok := payload["tagKeys"]
	if !ok {
		raw, ok = payload["TagKeys"]
	}
	if !ok {
		return []string{"env"}
	}
	keys := make([]string, 0)
	switch v := raw.(type) {
	case []string:
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				keys = append(keys, item)
			}
		}
	case []any:
		for _, item := range v {
			itemStr := strings.TrimSpace(fmt.Sprintf("%v", item))
			if itemStr != "" && itemStr != "<nil>" {
				keys = append(keys, itemStr)
			}
		}
	}
	if len(keys) == 0 {
		return []string{"env"}
	}
	return keys
}

func ivsCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
