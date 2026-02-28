package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type firehoseStore struct {
	mu              sync.Mutex
	nextID          int64
	deliveryStreams map[string]*firehoseDeliveryStream
}

type firehoseDeliveryStream struct {
	Name              string
	ARN               string
	Status            string
	VersionID         string
	DestinationID     string
	EncryptionEnabled bool
	Tags              map[string]string
}

func newFirehoseStore() *firehoseStore {
	seed := &firehoseDeliveryStream{
		Name:              "stackyard-firehose-stream",
		ARN:               firehoseDeliveryStreamARN("stackyard-firehose-stream"),
		Status:            "ACTIVE",
		VersionID:         "1",
		DestinationID:     "destinationId-000000000001",
		EncryptionEnabled: false,
		Tags: map[string]string{
			"stackyard": "true",
		},
	}

	return &firehoseStore{
		nextID: 1,
		deliveryStreams: map[string]*firehoseDeliveryStream{
			seed.Name: seed,
		},
	}
}

func (s *firehoseStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := firehosePayloadString(payload, "DeliveryStreamName", "stackyard-firehose-stream")
	if name == "" {
		name = "stackyard-firehose-stream"
	}

	switch action {
	case "CreateDeliveryStream":
		streamName := firehosePayloadString(payload, "DeliveryStreamName", "")
		if streamName == "" {
			streamName = s.nextTokenLocked("stackyard-firehose-stream", 6)
		}
		ds := s.ensureDeliveryStreamLocked(streamName)
		ds.Status = "ACTIVE"
		ds.VersionID = "1"
		s.applyTagsLocked(ds, payload)
		return map[string]any{}

	case "DeleteDeliveryStream":
		ds := s.ensureDeliveryStreamLocked(name)
		ds.Status = "DELETING"
		return map[string]any{}

	case "DescribeDeliveryStream":
		ds := s.ensureDeliveryStreamLocked(name)
		return map[string]any{
			"DeliveryStreamDescription": s.deliveryStreamDescriptionPayload(ds),
		}

	case "ListDeliveryStreams":
		names := s.sortedDeliveryStreamNamesLocked()
		limit := firehosePayloadInt(payload, "Limit", len(names))
		if limit > 0 && len(names) > limit {
			names = names[:limit]
		}
		list := make([]any, 0, len(names))
		for _, streamName := range names {
			list = append(list, streamName)
		}
		return map[string]any{
			"DeliveryStreamNames":    list,
			"HasMoreDeliveryStreams": false,
		}

	case "ListTagsForDeliveryStream":
		ds := s.ensureDeliveryStreamLocked(name)
		return map[string]any{
			"Tags":        firehoseTagsToList(ds.Tags),
			"HasMoreTags": false,
		}

	case "PutRecord":
		ds := s.ensureDeliveryStreamLocked(name)
		return map[string]any{
			"RecordId":  s.nextTokenLocked("record", 16),
			"Encrypted": ds.EncryptionEnabled,
		}

	case "PutRecordBatch":
		ds := s.ensureDeliveryStreamLocked(name)
		count := firehoseRecordCount(payload)
		if count <= 0 {
			count = 1
		}
		entries := make([]any, 0, count)
		for i := 0; i < count; i++ {
			entries = append(entries, map[string]any{"RecordId": s.nextTokenLocked("record", 16)})
		}
		return map[string]any{
			"FailedPutCount":   0,
			"RequestResponses": entries,
			"Encrypted":        ds.EncryptionEnabled,
		}

	case "StartDeliveryStreamEncryption":
		ds := s.ensureDeliveryStreamLocked(name)
		ds.EncryptionEnabled = true
		return map[string]any{
			"DeliveryStreamEncryptionConfiguration": map[string]any{
				"Status":  "ENABLING",
				"KeyType": "AWS_OWNED_CMK",
			},
		}

	case "StopDeliveryStreamEncryption":
		ds := s.ensureDeliveryStreamLocked(name)
		ds.EncryptionEnabled = false
		return map[string]any{
			"DeliveryStreamEncryptionConfiguration": map[string]any{
				"Status":  "DISABLING",
				"KeyType": "AWS_OWNED_CMK",
			},
		}

	case "TagDeliveryStream":
		ds := s.ensureDeliveryStreamLocked(name)
		s.applyTagsLocked(ds, payload)
		return map[string]any{}

	case "UntagDeliveryStream":
		ds := s.ensureDeliveryStreamLocked(name)
		tagKeys := firehosePayloadStringSlice(payload, "TagKeys")
		for _, key := range tagKeys {
			delete(ds.Tags, key)
		}
		return map[string]any{}

	case "UpdateDestination":
		ds := s.ensureDeliveryStreamLocked(name)
		ds.VersionID = firehosePayloadString(payload, "CurrentDeliveryStreamVersionId", ds.VersionID)
		if ds.VersionID == "" {
			ds.VersionID = "1"
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *firehoseStore) ensureDeliveryStreamLocked(name string) *firehoseDeliveryStream {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-firehose-stream"
	}
	if existing, ok := s.deliveryStreams[name]; ok {
		return existing
	}
	ds := &firehoseDeliveryStream{
		Name:              name,
		ARN:               firehoseDeliveryStreamARN(name),
		Status:            "ACTIVE",
		VersionID:         "1",
		DestinationID:     "destinationId-000000000001",
		EncryptionEnabled: false,
		Tags: map[string]string{
			"stackyard": "true",
		},
	}
	s.deliveryStreams[name] = ds
	return ds
}

func (s *firehoseStore) sortedDeliveryStreamNamesLocked() []string {
	names := make([]string, 0, len(s.deliveryStreams))
	for name := range s.deliveryStreams {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		names = append(names, "stackyard-firehose-stream")
	}
	return names
}

func (s *firehoseStore) deliveryStreamDescriptionPayload(ds *firehoseDeliveryStream) map[string]any {
	if ds == nil {
		ds = s.ensureDeliveryStreamLocked("stackyard-firehose-stream")
	}
	return map[string]any{
		"DeliveryStreamName":   ds.Name,
		"DeliveryStreamARN":    ds.ARN,
		"DeliveryStreamStatus": ds.Status,
		"DeliveryStreamType":   "DirectPut",
		"VersionId":            ds.VersionID,
		"HasMoreDestinations":  false,
		"Destinations": []any{
			map[string]any{
				"DestinationId": ds.DestinationID,
				"S3DestinationDescription": map[string]any{
					"BucketARN": "arn:aws:s3:::stackyard-firehose-bucket",
					"RoleARN":   "arn:aws:iam::123456789012:role/stackyard-firehose-role",
					"Prefix":    "firehose/",
				},
			},
		},
		"Tags": firehoseTagsToList(ds.Tags),
	}
}

func (s *firehoseStore) applyTagsLocked(ds *firehoseDeliveryStream, payload map[string]any) {
	if ds == nil {
		return
	}
	if ds.Tags == nil {
		ds.Tags = map[string]string{}
	}
	for key, value := range firehosePayloadTags(payload, "Tags") {
		ds.Tags[key] = value
	}
}

func (s *firehoseStore) nextTokenLocked(prefix string, width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func firehosePayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func firehosePayloadString(payload map[string]any, key, fallback string) string {
	value, ok := firehosePayloadValue(payload, key)
	if !ok {
		return fallback
	}
	out := strings.TrimSpace(fmt.Sprintf("%v", value))
	if out == "" {
		return fallback
	}
	return out
}

func firehosePayloadInt(payload map[string]any, key string, fallback int) int {
	value, ok := firehosePayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case jsonNumber:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	}
	parsed := strings.TrimSpace(fmt.Sprintf("%v", value))
	if parsed == "" {
		return fallback
	}
	var out int
	if _, err := fmt.Sscanf(parsed, "%d", &out); err == nil {
		return out
	}
	return fallback
}

type jsonNumber interface {
	Int64() (int64, error)
	Float64() (float64, error)
}

func firehosePayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := firehosePayloadValue(payload, key)
	if !ok {
		return nil
	}
	switch raw := value.(type) {
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", value))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func firehosePayloadTags(payload map[string]any, key string) map[string]string {
	value, ok := firehosePayloadValue(payload, key)
	if !ok {
		return nil
	}
	tags := map[string]string{}

	switch raw := value.(type) {
	case []any:
		for _, item := range raw {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k := firehosePayloadString(tag, "Key", "")
			v := firehosePayloadString(tag, "Value", "")
			if k == "" {
				continue
			}
			tags[k] = v
		}
	case map[string]any:
		for k, v := range raw {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}

	if len(tags) == 0 {
		return nil
	}
	return tags
}

func firehoseRecordCount(payload map[string]any) int {
	value, ok := firehosePayloadValue(payload, "Records")
	if !ok {
		return 0
	}
	if records, ok := value.([]any); ok {
		return len(records)
	}
	return 0
}

func firehoseTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func firehoseDeliveryStreamARN(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-firehose-stream"
	}
	return "arn:aws:firehose:us-east-1:123456789012:deliverystream/" + name
}
