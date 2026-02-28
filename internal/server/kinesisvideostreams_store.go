package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	kinesisVideoStreamsDefaultRegion    = "us-east-1"
	kinesisVideoStreamsDefaultAccountID = "123456789012"
)

type kinesisVideoStreamsStore struct {
	mu sync.Mutex

	nextStreamSerial  int64
	nextChannelSerial int64

	streams               map[string]map[string]any
	streamsByARN          map[string]string
	channels              map[string]map[string]any
	channelsByARN         map[string]string
	edgeConfigs           map[string]map[string]any
	notificationConfigs   map[string]map[string]any
	mediaStorageConfigs   map[string]map[string]any
	imageGenerationConfig map[string]map[string]any
	mappedResourceConfig  map[string]map[string]any
	streamStorageConfig   map[string]map[string]any
	tags                  map[string]map[string]string
}

func newKinesisVideoStreamsStore() *kinesisVideoStreamsStore {
	s := &kinesisVideoStreamsStore{
		nextStreamSerial:      2,
		nextChannelSerial:     2,
		streams:               map[string]map[string]any{},
		streamsByARN:          map[string]string{},
		channels:              map[string]map[string]any{},
		channelsByARN:         map[string]string{},
		edgeConfigs:           map[string]map[string]any{},
		notificationConfigs:   map[string]map[string]any{},
		mediaStorageConfigs:   map[string]map[string]any{},
		imageGenerationConfig: map[string]map[string]any{},
		mappedResourceConfig:  map[string]map[string]any{},
		streamStorageConfig:   map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
	}
	now := time.Now().UTC()
	s.ensureStreamLocked("stackyard-kvs-stream", "", now)
	s.ensureChannelLocked("stackyard-kvs-channel", "", now)
	return s
}

func (s *kinesisVideoStreamsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	stream := s.ensureStreamLocked(
		kinesisVideoStreamsPayloadString(payload, []string{"StreamName", "streamName"}, "stackyard-kvs-stream"),
		kinesisVideoStreamsPayloadString(payload, []string{"StreamARN", "StreamArn", "streamArn", "ResourceARN", "ResourceArn"}, ""),
		now,
	)
	channel := s.ensureChannelLocked(
		kinesisVideoStreamsPayloadString(payload, []string{"ChannelName", "channelName"}, "stackyard-kvs-channel"),
		kinesisVideoStreamsPayloadString(payload, []string{"ChannelARN", "ChannelArn", "channelArn"}, ""),
		now,
	)

	s.ensureConfigsLocked(stream, channel, now)

	streamARN := kinesisVideoStreamsString(stream, "StreamARN", "")
	channelARN := kinesisVideoStreamsString(channel, "ChannelARN", "")
	resourceARN := kinesisVideoStreamsPayloadString(
		payload,
		[]string{"ResourceARN", "ResourceArn", "resourceArn", "resourceARN"},
		"",
	)
	if resourceARN == "" {
		if strings.EqualFold(action, "TagStream") || strings.EqualFold(action, "ListTagsForStream") || strings.EqualFold(action, "UntagStream") {
			resourceARN = streamARN
		} else if strings.Contains(strings.ToLower(action), "signaling") {
			resourceARN = channelARN
		} else {
			resourceARN = streamARN
		}
	}

	s.ensureTagMapLocked(streamARN)
	s.ensureTagMapLocked(channelARN)
	s.ensureTagMapLocked(resourceARN)

	s.applyOperationMutationsLocked(action, payload, stream, channel, now)

	syncStatus := "IN_SYNC"
	if cfg := s.edgeConfigs[channelARN]; cfg != nil {
		syncStatus = kinesisVideoStreamsString(cfg, "SynchronizationStatus", "IN_SYNC")
	}

	switch action {
	case "CreateStream":
		return map[string]any{"StreamARN": streamARN}
	case "CreateSignalingChannel":
		return map[string]any{"ChannelARN": channelARN}
	case "DeleteStream", "DeleteSignalingChannel", "DeleteEdgeConfiguration":
		return map[string]any{}
	case "DescribeStream":
		return map[string]any{"StreamInfo": kinesisVideoStreamsCloneMap(stream)}
	case "DescribeSignalingChannel":
		return map[string]any{"ChannelInfo": kinesisVideoStreamsCloneMap(channel)}
	case "ListStreams":
		items := make([]any, 0, len(s.streams))
		for _, key := range kinesisVideoStreamsSortedKeys(s.streams) {
			items = append(items, kinesisVideoStreamsCloneMap(s.streams[key]))
		}
		return map[string]any{"StreamInfoList": items, "NextToken": ""}
	case "ListSignalingChannels":
		items := make([]any, 0, len(s.channels))
		for _, key := range kinesisVideoStreamsSortedKeys(s.channels) {
			items = append(items, kinesisVideoStreamsCloneMap(s.channels[key]))
		}
		return map[string]any{"ChannelInfoList": items, "NextToken": ""}
	case "UpdateStream", "UpdateSignalingChannel", "UpdateDataRetention", "TagStream", "TagResource", "UntagStream", "UntagResource", "UpdateMediaStorageConfiguration", "UpdateNotificationConfiguration", "UpdateImageGenerationConfiguration", "UpdateStreamStorageConfiguration":
		return map[string]any{}
	case "GetDataEndpoint":
		return map[string]any{
			"DataEndpoint": fmt.Sprintf(
				"https://stackyard-kinesisvideo.local/%s/data",
				strings.TrimPrefix(streamARN, "arn:aws:kinesisvideo:"),
			),
		}
	case "GetSignalingChannelEndpoint":
		return map[string]any{
			"ResourceEndpointList": []any{
				map[string]any{"Protocol": "WSS", "ResourceEndpoint": "wss://stackyard-kinesisvideo.local/signaling"},
				map[string]any{"Protocol": "HTTPS", "ResourceEndpoint": "https://stackyard-kinesisvideo.local/signaling"},
			},
		}
	case "DescribeEdgeConfiguration":
		cfg := kinesisVideoStreamsCloneMap(s.edgeConfigs[channelARN])
		return map[string]any{
			"ChannelName":           kinesisVideoStreamsString(channel, "ChannelName", ""),
			"ChannelARN":            channelARN,
			"EdgeConfig":            kinesisVideoStreamsAny(cfg, "EdgeConfig", map[string]any{}),
			"SynchronizationStatus": syncStatus,
		}
	case "StartEdgeConfigurationUpdate":
		return map[string]any{
			"SynchronizationStatus": syncStatus,
			"TimeStamp":             now.Format(time.RFC3339),
		}
	case "ListEdgeAgentConfigurations":
		cfg := kinesisVideoStreamsCloneMap(s.edgeConfigs[channelARN])
		return map[string]any{
			"EdgeConfigs": []any{
				map[string]any{
					"ChannelARN":            channelARN,
					"EdgeConfig":            kinesisVideoStreamsAny(cfg, "EdgeConfig", map[string]any{}),
					"SynchronizationStatus": syncStatus,
				},
			},
			"NextToken": "",
		}
	case "DescribeNotificationConfiguration":
		return map[string]any{"NotificationConfiguration": kinesisVideoStreamsCloneMap(s.notificationConfigs[streamARN])}
	case "DescribeMediaStorageConfiguration":
		return map[string]any{"MediaStorageConfiguration": kinesisVideoStreamsCloneMap(s.mediaStorageConfigs[channelARN])}
	case "DescribeImageGenerationConfiguration":
		return map[string]any{"ImageGenerationConfiguration": kinesisVideoStreamsCloneMap(s.imageGenerationConfig[streamARN])}
	case "DescribeMappedResourceConfiguration":
		return map[string]any{"MappedResourceConfigurationList": []any{kinesisVideoStreamsCloneMap(s.mappedResourceConfig[channelARN])}}
	case "DescribeStreamStorageConfiguration":
		return map[string]any{"StreamStorageConfiguration": kinesisVideoStreamsCloneMap(s.streamStorageConfig[streamARN])}
	case "ListTagsForResource", "ListTagsForStream":
		return map[string]any{"Tags": kinesisVideoStreamsTagsAsList(s.tags[resourceARN])}
	default:
		return map[string]any{}
	}
}

func (s *kinesisVideoStreamsStore) applyOperationMutationsLocked(
	action string,
	payload map[string]any,
	stream map[string]any,
	channel map[string]any,
	now time.Time,
) {
	streamARN := kinesisVideoStreamsString(stream, "StreamARN", "")
	channelARN := kinesisVideoStreamsString(channel, "ChannelARN", "")

	switch action {
	case "CreateStream":
		if mediaType := kinesisVideoStreamsPayloadString(payload, []string{"MediaType", "mediaType"}, ""); mediaType != "" {
			stream["MediaType"] = mediaType
		}
		if deviceName := kinesisVideoStreamsPayloadString(payload, []string{"DeviceName", "deviceName"}, ""); deviceName != "" {
			stream["DeviceName"] = deviceName
		}
		if retention := kinesisVideoStreamsPayloadInt(payload, []string{"DataRetentionInHours", "dataRetentionInHours"}, -1); retention >= 0 {
			stream["DataRetentionInHours"] = retention
		}
		stream["Status"] = "ACTIVE"
		stream["Version"] = kinesisVideoStreamsVersion(now)
	case "CreateSignalingChannel":
		channel["Status"] = "ACTIVE"
		channel["Version"] = kinesisVideoStreamsVersion(now)
	case "DeleteStream":
		delete(s.streamsByARN, streamARN)
		delete(s.streams, kinesisVideoStreamsString(stream, "StreamName", ""))
		delete(s.notificationConfigs, streamARN)
		delete(s.imageGenerationConfig, streamARN)
		delete(s.streamStorageConfig, streamARN)
	case "DeleteSignalingChannel":
		delete(s.channelsByARN, channelARN)
		delete(s.channels, kinesisVideoStreamsString(channel, "ChannelName", ""))
		delete(s.edgeConfigs, channelARN)
		delete(s.mediaStorageConfigs, channelARN)
		delete(s.mappedResourceConfig, channelARN)
	case "UpdateStream":
		if mediaType := kinesisVideoStreamsPayloadString(payload, []string{"MediaType", "mediaType"}, ""); mediaType != "" {
			stream["MediaType"] = mediaType
		}
		if deviceName := kinesisVideoStreamsPayloadString(payload, []string{"DeviceName", "deviceName"}, ""); deviceName != "" {
			stream["DeviceName"] = deviceName
		}
		stream["Version"] = kinesisVideoStreamsVersion(now)
	case "UpdateSignalingChannel":
		if messageTtl := kinesisVideoStreamsPayloadInt(payload, []string{"SingleMasterConfiguration", "MessageTtlSeconds"}, -1); messageTtl > 0 {
			channel["SingleMasterConfiguration"] = map[string]any{"MessageTtlSeconds": messageTtl}
		}
		channel["Version"] = kinesisVideoStreamsVersion(now)
	case "UpdateDataRetention":
		change := kinesisVideoStreamsPayloadInt(payload, []string{"DataRetentionChangeInHours", "dataRetentionChangeInHours"}, 0)
		if change != 0 {
			current := kinesisVideoStreamsIntAny(stream["DataRetentionInHours"], 24)
			next := current + change
			if next < 0 {
				next = 0
			}
			stream["DataRetentionInHours"] = next
		}
	case "TagStream", "TagResource":
		resourceARN := kinesisVideoStreamsPayloadString(payload, []string{"ResourceARN", "ResourceArn", "resourceArn"}, streamARN)
		if strings.EqualFold(action, "TagStream") {
			resourceARN = kinesisVideoStreamsPayloadString(payload, []string{"StreamARN", "StreamArn"}, streamARN)
		}
		tags := s.ensureTagMapLocked(resourceARN)
		for key, value := range kinesisVideoStreamsExtractTags(payload) {
			tags[key] = value
		}
	case "UntagStream", "UntagResource":
		resourceARN := kinesisVideoStreamsPayloadString(payload, []string{"ResourceARN", "ResourceArn", "resourceArn"}, streamARN)
		if strings.EqualFold(action, "UntagStream") {
			resourceARN = kinesisVideoStreamsPayloadString(payload, []string{"StreamARN", "StreamArn"}, streamARN)
		}
		tags := s.ensureTagMapLocked(resourceARN)
		for _, key := range kinesisVideoStreamsExtractTagKeys(payload) {
			delete(tags, key)
		}
	case "UpdateNotificationConfiguration":
		cfg := s.ensureNotificationConfigLocked(streamARN)
		cfg["Status"] = "ENABLED"
		if destination := payload["NotificationDestinationConfig"]; destination != nil {
			cfg["NotificationDestinationConfig"] = destination
		}
	case "UpdateMediaStorageConfiguration":
		cfg := s.ensureMediaStorageConfigLocked(channelARN)
		cfg["Status"] = "ENABLED"
		if streamArn := kinesisVideoStreamsPayloadString(payload, []string{"StreamARN", "StreamArn"}, ""); streamArn != "" {
			cfg["StreamARN"] = streamArn
		}
	case "UpdateImageGenerationConfiguration":
		cfg := s.ensureImageGenerationConfigLocked(streamARN)
		cfg["Status"] = "ENABLED"
		for key, value := range payload {
			cfg[key] = value
		}
	case "UpdateStreamStorageConfiguration":
		cfg := s.ensureStreamStorageConfigLocked(streamARN)
		for key, value := range payload {
			cfg[key] = value
		}
	case "StartEdgeConfigurationUpdate":
		cfg := s.ensureEdgeConfigLocked(channelARN)
		if edge := payload["EdgeConfig"]; edge != nil {
			cfg["EdgeConfig"] = edge
		}
		cfg["SynchronizationStatus"] = "IN_SYNC"
	case "DeleteEdgeConfiguration":
		delete(s.edgeConfigs, channelARN)
	}
}

func (s *kinesisVideoStreamsStore) ensureConfigsLocked(stream, channel map[string]any, now time.Time) {
	streamARN := kinesisVideoStreamsString(stream, "StreamARN", "")
	channelARN := kinesisVideoStreamsString(channel, "ChannelARN", "")
	s.ensureEdgeConfigLocked(channelARN)
	s.ensureNotificationConfigLocked(streamARN)
	s.ensureMediaStorageConfigLocked(channelARN)
	s.ensureImageGenerationConfigLocked(streamARN)
	s.ensureMappedResourceConfigLocked(channelARN)
	s.ensureStreamStorageConfigLocked(streamARN)
	stream["CreationTime"] = kinesisVideoStreamsString(stream, "CreationTime", now.Format(time.RFC3339))
	channel["CreationTime"] = kinesisVideoStreamsString(channel, "CreationTime", now.Format(time.RFC3339))
}

func (s *kinesisVideoStreamsStore) ensureStreamLocked(name, arn string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	arn = strings.TrimSpace(arn)

	if arn != "" {
		if mappedName := strings.TrimSpace(s.streamsByARN[arn]); mappedName != "" {
			if stream := s.streams[mappedName]; stream != nil {
				return stream
			}
		}
		parsedName := kinesisVideoStreamsNameFromARN(arn)
		if parsedName != "" {
			name = parsedName
		}
	}
	if name == "" {
		name = "stackyard-kvs-stream"
	}

	if stream := s.streams[name]; stream != nil {
		if arn != "" {
			stream["StreamARN"] = arn
			s.streamsByARN[arn] = name
		}
		return stream
	}

	serial := s.nextStreamSerial
	if name == "stackyard-kvs-stream" {
		serial = 1
	} else {
		s.nextStreamSerial++
	}
	if arn == "" {
		arn = kinesisVideoStreamsStreamARN(name, serial)
	}

	stream := map[string]any{
		"StreamName":           name,
		"StreamARN":            arn,
		"Version":              kinesisVideoStreamsVersion(now),
		"Status":               "ACTIVE",
		"CreationTime":         now.Format(time.RFC3339),
		"DataRetentionInHours": 24,
		"MediaType":            "video/h264",
		"DeviceName":           "stackyard-device",
	}
	s.streams[name] = stream
	s.streamsByARN[arn] = name
	return stream
}

func (s *kinesisVideoStreamsStore) ensureChannelLocked(name, arn string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	arn = strings.TrimSpace(arn)

	if arn != "" {
		if mappedName := strings.TrimSpace(s.channelsByARN[arn]); mappedName != "" {
			if channel := s.channels[mappedName]; channel != nil {
				return channel
			}
		}
		parsedName := kinesisVideoStreamsNameFromARN(arn)
		if parsedName != "" {
			name = parsedName
		}
	}
	if name == "" {
		name = "stackyard-kvs-channel"
	}

	if channel := s.channels[name]; channel != nil {
		if arn != "" {
			channel["ChannelARN"] = arn
			s.channelsByARN[arn] = name
		}
		return channel
	}

	serial := s.nextChannelSerial
	if name == "stackyard-kvs-channel" {
		serial = 1
	} else {
		s.nextChannelSerial++
	}
	if arn == "" {
		arn = kinesisVideoStreamsChannelARN(name, serial)
	}

	channel := map[string]any{
		"ChannelName":               name,
		"ChannelARN":                arn,
		"Version":                   kinesisVideoStreamsVersion(now),
		"Status":                    "ACTIVE",
		"CreationTime":              now.Format(time.RFC3339),
		"SingleMasterConfiguration": map[string]any{"MessageTtlSeconds": 60},
		"SingleMasterChannelEndpointConfiguration": map[string]any{"Protocols": []any{"WSS", "HTTPS"}, "Role": "MASTER"},
	}
	s.channels[name] = channel
	s.channelsByARN[arn] = name
	return channel
}

func (s *kinesisVideoStreamsStore) ensureEdgeConfigLocked(channelARN string) map[string]any {
	if cfg := s.edgeConfigs[channelARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"ChannelARN": channelARN,
		"EdgeConfig": map[string]any{
			"HubDeviceArn": "arn:aws:iot:us-east-1:123456789012:thing/stackyard-hub",
			"RecorderConfig": map[string]any{
				"MediaSourceConfig": []any{map[string]any{"Type": "RTSP_URI", "Uri": "rtsp://127.0.0.1/live"}},
			},
		},
		"SynchronizationStatus": "IN_SYNC",
	}
	s.edgeConfigs[channelARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureNotificationConfigLocked(streamARN string) map[string]any {
	if cfg := s.notificationConfigs[streamARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"Status": "ENABLED",
		"NotificationDestinationConfig": map[string]any{
			"Uri": "arn:aws:sns:us-east-1:123456789012:stackyard-kvs-notify",
		},
	}
	s.notificationConfigs[streamARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureMediaStorageConfigLocked(channelARN string) map[string]any {
	if cfg := s.mediaStorageConfigs[channelARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"ChannelARN": channelARN,
		"Status":     "ENABLED",
		"StreamARN":  kinesisVideoStreamsStreamARN("stackyard-kvs-stream", 1),
	}
	s.mediaStorageConfigs[channelARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureImageGenerationConfigLocked(streamARN string) map[string]any {
	if cfg := s.imageGenerationConfig[streamARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"Status":                           "ENABLED",
		"ImageSelectorType":                "SERVER_TIMESTAMP",
		"SamplingInterval":                 200,
		"Format":                           "JPEG",
		"WidthPixels":                      1280,
		"HeightPixels":                     720,
		"ImageGenerationDestinationConfig": map[string]any{"DestinationRegion": kinesisVideoStreamsDefaultRegion, "Uri": "s3://stackyard-kvs-images"},
	}
	s.imageGenerationConfig[streamARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureMappedResourceConfigLocked(channelARN string) map[string]any {
	if cfg := s.mappedResourceConfig[channelARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"Type":                "S3_DESTINATION",
		"ResourceArn":         "arn:aws:s3:::stackyard-kvs-archive",
		"ConfigurationStatus": "ENABLED",
	}
	s.mappedResourceConfig[channelARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureStreamStorageConfigLocked(streamARN string) map[string]any {
	if cfg := s.streamStorageConfig[streamARN]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"StreamARN":    streamARN,
		"Status":       "ENABLED",
		"StorageClass": "STANDARD",
	}
	s.streamStorageConfig[streamARN] = cfg
	return cfg
}

func (s *kinesisVideoStreamsStore) ensureTagMapLocked(resourceARN string) map[string]string {
	if resourceARN == "" {
		resourceARN = kinesisVideoStreamsStreamARN("stackyard-kvs-stream", 1)
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{"stackyard": "true", "service": "kinesisvideostreams"}
	s.tags[resourceARN] = tags
	return tags
}

func kinesisVideoStreamsPayloadString(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			s := strings.TrimSpace(kinesisVideoStreamsStringAny(value))
			if s != "" {
				return s
			}
		}
	}
	if len(keys) == 2 {
		if nested, ok := payload[keys[0]].(map[string]any); ok {
			if value, ok := nested[keys[1]]; ok {
				s := strings.TrimSpace(kinesisVideoStreamsStringAny(value))
				if s != "" {
					return s
				}
			}
		}
	}
	return def
}

func kinesisVideoStreamsPayloadInt(payload map[string]any, keys []string, def int) int {
	if len(keys) == 2 {
		if nested, ok := payload[keys[0]].(map[string]any); ok {
			if value, ok := nested[keys[1]]; ok {
				return kinesisVideoStreamsIntAny(value, def)
			}
		}
	}
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return kinesisVideoStreamsIntAny(value, def)
		}
	}
	return def
}

func kinesisVideoStreamsExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if raw, ok := payload["Tags"]; ok {
		switch v := raw.(type) {
		case map[string]any:
			for key, val := range v {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				out[key] = kinesisVideoStreamsStringAny(val)
			}
		case []any:
			for _, item := range v {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := strings.TrimSpace(kinesisVideoStreamsPayloadString(entry, []string{"Key", "TagKey"}, ""))
				if key == "" {
					continue
				}
				out[key] = kinesisVideoStreamsPayloadString(entry, []string{"Value", "TagValue"}, "")
			}
		}
	}
	return out
}

func kinesisVideoStreamsExtractTagKeys(payload map[string]any) []string {
	var keys []string
	if raw, ok := payload["TagKeyList"]; ok {
		if list, ok := raw.([]any); ok {
			for _, item := range list {
				key := strings.TrimSpace(kinesisVideoStreamsStringAny(item))
				if key != "" {
					keys = append(keys, key)
				}
			}
		}
	}
	if raw, ok := payload["TagKeys"]; ok {
		if list, ok := raw.([]any); ok {
			for _, item := range list {
				key := strings.TrimSpace(kinesisVideoStreamsStringAny(item))
				if key != "" {
					keys = append(keys, key)
				}
			}
		}
	}
	if key := strings.TrimSpace(kinesisVideoStreamsPayloadString(payload, []string{"TagKey", "tagKey"}, "")); key != "" {
		keys = append(keys, key)
	}
	return keys
}

func kinesisVideoStreamsNameFromARN(arn string) string {
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-2])
}

func kinesisVideoStreamsStreamARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kinesisvideo:%s:%s:stream/%s/%06d",
		kinesisVideoStreamsDefaultRegion,
		kinesisVideoStreamsDefaultAccountID,
		name,
		serial,
	)
}

func kinesisVideoStreamsChannelARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kinesisvideo:%s:%s:channel/%s/%06d",
		kinesisVideoStreamsDefaultRegion,
		kinesisVideoStreamsDefaultAccountID,
		name,
		serial,
	)
}

func kinesisVideoStreamsVersion(now time.Time) string {
	return now.UTC().Format("20060102150405")
}

func kinesisVideoStreamsStringAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func kinesisVideoStreamsString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	if value, ok := m[key]; ok {
		s := strings.TrimSpace(kinesisVideoStreamsStringAny(value))
		if s != "" {
			return s
		}
	}
	return def
}

func kinesisVideoStreamsIntAny(value any, def int) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case jsonNumber:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}

func kinesisVideoStreamsAny(m map[string]any, key string, def any) any {
	if m == nil {
		return def
	}
	if value, ok := m[key]; ok {
		return value
	}
	return def
}

func kinesisVideoStreamsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func kinesisVideoStreamsTagsAsList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func kinesisVideoStreamsSortedKeys(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
