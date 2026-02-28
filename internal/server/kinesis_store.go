package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type kinesisStore struct {
	mu sync.Mutex

	nextSequence int64
	streams      map[string]*kinesisStream
	consumers    map[string]*kinesisConsumer
	iterators    map[string]string
	policies     map[string]string
	tags         map[string]map[string]string
}

type kinesisStream struct {
	Name             string
	ARN              string
	Status           string
	Mode             string
	EncryptionType   string
	KeyID            string
	RetentionHours   int
	ShardCount       int
	MaxRecordSizeKiB int
	Shards           []string
}

type kinesisConsumer struct {
	Name      string
	ARN       string
	StreamARN string
	Status    string
	CreatedAt string
}

func newKinesisStore() *kinesisStore {
	s := &kinesisStore{
		nextSequence: 2,
		streams:      map[string]*kinesisStream{},
		consumers:    map[string]*kinesisConsumer{},
		iterators:    map[string]string{},
		policies:     map[string]string{},
		tags:         map[string]map[string]string{},
	}
	seed := s.ensureStreamLocked("stackyard-kinesis-stream")
	s.ensureTagMapLocked(seed.ARN)
	return s
}

func (s *kinesisStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	streamName := kinesisPayloadString(payload, "StreamName", "stackyard-kinesis-stream")
	streamARN := kinesisPayloadString(payload, "StreamARN", "")
	if streamARN == "" {
		streamARN = kinesisPayloadString(payload, "ResourceARN", "")
	}
	stream := s.resolveStreamLocked(streamName, streamARN)
	resourceARN := kinesisPayloadString(payload, "ResourceARN", stream.ARN)
	if resourceARN == "" {
		resourceARN = stream.ARN
	}
	s.ensureTagMapLocked(resourceARN)

	consumerARN := kinesisPayloadString(payload, "ConsumerARN", "")
	consumerName := kinesisPayloadString(payload, "ConsumerName", "stackyard-kinesis-consumer")
	if action == "RegisterStreamConsumer" {
		consumer := s.ensureConsumerLocked(stream.ARN, consumerName)
		consumerARN = consumer.ARN
	}
	consumer := s.resolveConsumerLocked(consumerARN, consumerName, stream.ARN)

	switch action {
	case "CreateStream":
		name := kinesisPayloadString(payload, "StreamName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-kinesis-stream-%06d", len(s.streams)+1)
		}
		stream = s.ensureStreamLocked(name)
		if mode := kinesisPayloadStringInMap(payload, "StreamMode", ""); mode != "" {
			stream.Mode = mode
		}
		if count := kinesisPayloadInt(payload, "ShardCount", 0); count > 0 {
			stream.ShardCount = count
			stream.Shards = kinesisShardIDs(count)
		}
		stream.Status = "ACTIVE"
		s.ensureTagMapLocked(stream.ARN)
		return map[string]any{}

	case "DeleteStream":
		stream.Status = "DELETING"
		return map[string]any{}

	case "DescribeStream":
		return map[string]any{
			"StreamDescription": map[string]any{
				"StreamName":              stream.Name,
				"StreamARN":               stream.ARN,
				"StreamStatus":            stream.Status,
				"Shards":                  kinesisShardsPayload(stream.Shards),
				"HasMoreShards":           false,
				"RetentionPeriodHours":    stream.RetentionHours,
				"EnhancedMonitoring":      []any{},
				"EncryptionType":          stream.EncryptionType,
				"KeyId":                   stream.KeyID,
				"StreamCreationTimestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}

	case "DescribeStreamSummary":
		return map[string]any{
			"StreamDescriptionSummary": map[string]any{
				"StreamName":           stream.Name,
				"StreamARN":            stream.ARN,
				"StreamStatus":         stream.Status,
				"RetentionPeriodHours": stream.RetentionHours,
				"OpenShardCount":       stream.ShardCount,
				"StreamModeDetails": map[string]any{
					"StreamMode": stream.Mode,
				},
			},
		}

	case "ListStreams":
		names := make([]string, 0, len(s.streams))
		for name := range s.streams {
			names = append(names, name)
		}
		sort.Strings(names)
		items := make([]any, 0, len(names))
		for _, name := range names {
			items = append(items, name)
		}
		return map[string]any{"StreamNames": items, "HasMoreStreams": false}

	case "ListShards":
		return map[string]any{"Shards": kinesisShardsPayload(stream.Shards), "NextToken": ""}

	case "PutRecord":
		return map[string]any{
			"ShardId":        stream.Shards[0],
			"SequenceNumber": s.nextSequenceLocked(),
		}

	case "PutRecords":
		recordCount := 1
		if raw, ok := payload["Records"].([]any); ok && len(raw) > 0 {
			recordCount = len(raw)
		}
		records := make([]any, 0, recordCount)
		for i := 0; i < recordCount; i++ {
			records = append(records, map[string]any{"ShardId": stream.Shards[0], "SequenceNumber": s.nextSequenceLocked()})
		}
		return map[string]any{"FailedRecordCount": 0, "Records": records}

	case "GetShardIterator":
		iterator := fmt.Sprintf("%s:%s:%s", stream.Name, kinesisPayloadString(payload, "ShardId", stream.Shards[0]), s.nextSequenceLocked())
		s.iterators[iterator] = stream.Name
		return map[string]any{"ShardIterator": iterator}

	case "GetRecords":
		next := fmt.Sprintf("iterator-%s", s.nextSequenceLocked())
		return map[string]any{
			"Records": []any{
				map[string]any{
					"SequenceNumber":              s.nextSequenceLocked(),
					"ApproximateArrivalTimestamp": time.Now().UTC().Format(time.RFC3339),
					"Data":                        "c3RhY2t5YXJk",
					"PartitionKey":                "coverage-partition-key",
				},
			},
			"MillisBehindLatest": 0,
			"NextShardIterator":  next,
		}

	case "SubscribeToShard":
		return map[string]any{"EventStream": []any{}, "ContinuationSequenceNumber": s.nextSequenceLocked()}

	case "RegisterStreamConsumer":
		return map[string]any{"Consumer": kinesisConsumerPayload(consumer)}

	case "DeregisterStreamConsumer":
		if consumer != nil {
			delete(s.consumers, consumer.ARN)
		}
		return map[string]any{}

	case "DescribeStreamConsumer":
		return map[string]any{"ConsumerDescription": kinesisConsumerPayload(consumer)}

	case "ListStreamConsumers":
		items := []any{}
		for _, key := range kinesisSortedConsumerKeys(s.consumers) {
			c := s.consumers[key]
			if c.StreamARN == stream.ARN {
				items = append(items, kinesisConsumerPayload(c))
			}
		}
		return map[string]any{"Consumers": items, "NextToken": ""}

	case "AddTagsToStream", "TagResource":
		targetARN := stream.ARN
		if action == "TagResource" {
			targetARN = resourceARN
		}
		tags := s.ensureTagMapLocked(targetARN)
		for key, value := range kinesisPayloadTags(payload) {
			tags[key] = value
		}
		return map[string]any{}

	case "RemoveTagsFromStream", "UntagResource":
		targetARN := stream.ARN
		if action == "UntagResource" {
			targetARN = resourceARN
		}
		tags := s.ensureTagMapLocked(targetARN)
		for _, key := range kinesisPayloadTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForStream", "ListTagsForResource":
		targetARN := stream.ARN
		if action == "ListTagsForResource" {
			targetARN = resourceARN
		}
		return map[string]any{"Tags": kinesisTagListPayload(s.ensureTagMapLocked(targetARN)), "HasMoreTags": false}

	case "GetResourcePolicy":
		policy := s.policies[resourceARN]
		if strings.TrimSpace(policy) == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
		}
		return map[string]any{"Policy": policy}

	case "PutResourcePolicy":
		s.policies[resourceARN] = kinesisPayloadString(payload, "Policy", `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{}

	case "DeleteResourcePolicy":
		delete(s.policies, resourceARN)
		return map[string]any{}

	case "DescribeLimits":
		open := 0
		for _, st := range s.streams {
			open += st.ShardCount
		}
		return map[string]any{"ShardLimit": 10000, "OpenShardCount": open}

	case "DescribeAccountSettings":
		return map[string]any{"AccountLimit": map[string]any{"ShardLimit": 10000, "OnDemandStreamCount": 50}}

	case "UpdateAccountSettings":
		return map[string]any{}

	case "EnableEnhancedMonitoring", "DisableEnhancedMonitoring":
		return map[string]any{"StreamName": stream.Name, "CurrentShardLevelMetrics": []any{"IncomingBytes"}, "DesiredShardLevelMetrics": []any{"IncomingBytes", "OutgoingBytes"}}

	case "IncreaseStreamRetentionPeriod":
		stream.RetentionHours += kinesisPayloadInt(payload, "RetentionPeriodHours", 24)
		return map[string]any{}

	case "DecreaseStreamRetentionPeriod":
		stream.RetentionHours -= kinesisPayloadInt(payload, "RetentionPeriodHours", 24)
		if stream.RetentionHours < 24 {
			stream.RetentionHours = 24
		}
		return map[string]any{}

	case "StartStreamEncryption":
		stream.EncryptionType = "KMS"
		stream.KeyID = kinesisPayloadString(payload, "KeyId", "alias/aws/kinesis")
		return map[string]any{}

	case "StopStreamEncryption":
		stream.EncryptionType = "NONE"
		stream.KeyID = ""
		return map[string]any{}

	case "SplitShard":
		stream.ShardCount++
		stream.Shards = kinesisShardIDs(stream.ShardCount)
		return map[string]any{}

	case "MergeShards":
		if stream.ShardCount > 1 {
			stream.ShardCount--
			stream.Shards = kinesisShardIDs(stream.ShardCount)
		}
		return map[string]any{}

	case "UpdateShardCount":
		target := kinesisPayloadInt(payload, "TargetShardCount", stream.ShardCount)
		if target < 1 {
			target = 1
		}
		stream.ShardCount = target
		stream.Shards = kinesisShardIDs(stream.ShardCount)
		return map[string]any{"CurrentShardCount": stream.ShardCount, "TargetShardCount": stream.ShardCount, "StreamName": stream.Name}

	case "UpdateStreamMode":
		mode := kinesisPayloadStringInMap(payload, "StreamMode", "PROVISIONED")
		if mode == "" {
			mode = "PROVISIONED"
		}
		stream.Mode = mode
		return map[string]any{}

	case "UpdateMaxRecordSize":
		v := kinesisPayloadInt(payload, "MaxRecordSizeInKiB", stream.MaxRecordSizeKiB)
		if v > 0 {
			stream.MaxRecordSizeKiB = v
		}
		return map[string]any{}

	case "UpdateStreamWarmThroughput":
		return map[string]any{"StreamARN": stream.ARN, "WarmThroughput": map[string]any{"Status": "ACTIVE", "ReadUnitsPerSecond": 1, "WriteUnitsPerSecond": 1}}
	}

	return map[string]any{}
}

func (s *kinesisStore) resolveStreamLocked(name, arn string) *kinesisStream {
	if strings.TrimSpace(arn) != "" {
		for _, stream := range s.streams {
			if stream.ARN == arn {
				return stream
			}
		}
		parts := strings.Split(arn, "/")
		if len(parts) >= 2 {
			name = parts[len(parts)-1]
		}
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-kinesis-stream"
	}
	return s.ensureStreamLocked(name)
}

func (s *kinesisStore) ensureStreamLocked(name string) *kinesisStream {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-kinesis-stream"
	}
	if existing := s.streams[name]; existing != nil {
		return existing
	}
	stream := &kinesisStream{
		Name:             name,
		ARN:              fmt.Sprintf("arn:aws:kinesis:us-east-1:123456789012:stream/%s", name),
		Status:           "ACTIVE",
		Mode:             "PROVISIONED",
		EncryptionType:   "NONE",
		RetentionHours:   24,
		ShardCount:       1,
		MaxRecordSizeKiB: 1024,
		Shards:           kinesisShardIDs(1),
	}
	s.streams[name] = stream
	return stream
}

func (s *kinesisStore) ensureConsumerLocked(streamARN, consumerName string) *kinesisConsumer {
	if strings.TrimSpace(consumerName) == "" {
		consumerName = "stackyard-kinesis-consumer"
	}
	for _, c := range s.consumers {
		if c.StreamARN == streamARN && strings.EqualFold(c.Name, consumerName) {
			return c
		}
	}
	arn := fmt.Sprintf("%s/consumer/%s:%d", streamARN, consumerName, time.Now().UTC().Unix())
	consumer := &kinesisConsumer{
		Name:      consumerName,
		ARN:       arn,
		StreamARN: streamARN,
		Status:    "ACTIVE",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.consumers[arn] = consumer
	return consumer
}

func (s *kinesisStore) resolveConsumerLocked(consumerARN, consumerName, streamARN string) *kinesisConsumer {
	if strings.TrimSpace(consumerARN) != "" {
		if consumer := s.consumers[consumerARN]; consumer != nil {
			return consumer
		}
	}
	if strings.TrimSpace(consumerName) == "" {
		consumerName = "stackyard-kinesis-consumer"
	}
	for _, c := range s.consumers {
		if c.StreamARN == streamARN && strings.EqualFold(c.Name, consumerName) {
			return c
		}
	}
	return s.ensureConsumerLocked(streamARN, consumerName)
}

func (s *kinesisStore) ensureTagMapLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = "arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-kinesis-stream"
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	tags := map[string]string{"stackyard": "true", "service": "kinesis"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *kinesisStore) nextSequenceLocked() string {
	n := s.nextSequence
	s.nextSequence++
	return fmt.Sprintf("%020d", n)
}

func kinesisPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			out := strings.TrimSpace(fmt.Sprintf("%v", value))
			if out != "" {
				return out
			}
		}
	}
	return fallback
}

func kinesisPayloadStringInMap(payload map[string]any, nestedKey, fallback string) string {
	for _, key := range []string{"StreamModeDetails", "streamModeDetails"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		nested, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for nk, nv := range nested {
			if strings.EqualFold(nk, nestedKey) {
				out := strings.TrimSpace(fmt.Sprintf("%v", nv))
				if out != "" {
					return out
				}
			}
		}
	}
	return fallback
}

func kinesisPayloadInt(payload map[string]any, key string, fallback int) int {
	for k, value := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float32:
			return int(v)
		case float64:
			return int(v)
		case jsonNumber:
			if n, err := v.Int64(); err == nil {
				return int(n)
			}
			if f, err := v.Float64(); err == nil {
				return int(f)
			}
		}
		var out int
		if _, err := fmt.Sscanf(strings.TrimSpace(fmt.Sprintf("%v", value)), "%d", &out); err == nil {
			return out
		}
	}
	return fallback
}

func kinesisPayloadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := payload["Tags"]
	if !ok {
		raw, ok = payload["tags"]
	}
	if !ok {
		return out
	}
	switch v := raw.(type) {
	case map[string]any:
		for key, val := range v {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	case []any:
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := kinesisPayloadString(entry, "Key", "")
			if key == "" {
				continue
			}
			value := kinesisPayloadString(entry, "Value", "")
			out[key] = value
		}
	}
	return out
}

func kinesisPayloadTagKeys(payload map[string]any) []string {
	keys := []string{}
	for _, field := range []string{"TagKeys", "tagKeys", "TagKeyList", "tagKeyList"} {
		raw, ok := payload[field]
		if !ok {
			continue
		}
		if list, ok := raw.([]any); ok {
			for _, item := range list {
				key := strings.TrimSpace(fmt.Sprintf("%v", item))
				if key != "" {
					keys = append(keys, key)
				}
			}
		}
	}
	if len(keys) == 0 {
		if key := kinesisPayloadString(payload, "TagKey", ""); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func kinesisTagListPayload(tags map[string]string) []any {
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

func kinesisShardIDs(count int) []string {
	if count < 1 {
		count = 1
	}
	shards := make([]string, 0, count)
	for i := 0; i < count; i++ {
		shards = append(shards, fmt.Sprintf("shardId-%012d", i))
	}
	return shards
}

func kinesisShardsPayload(ids []string) []any {
	if len(ids) == 0 {
		ids = []string{"shardId-000000000000"}
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{
			"ShardId": id,
			"HashKeyRange": map[string]any{
				"StartingHashKey": "0",
				"EndingHashKey":   "340282366920938463463374607431768211455",
			},
			"SequenceNumberRange": map[string]any{
				"StartingSequenceNumber": "1",
			},
		})
	}
	return out
}

func kinesisConsumerPayload(consumer *kinesisConsumer) map[string]any {
	if consumer == nil {
		return map[string]any{}
	}
	return map[string]any{
		"ConsumerName":              consumer.Name,
		"ConsumerARN":               consumer.ARN,
		"ConsumerStatus":            consumer.Status,
		"ConsumerCreationTimestamp": consumer.CreatedAt,
		"StreamARN":                 consumer.StreamARN,
	}
}

func kinesisSortedConsumerKeys(consumers map[string]*kinesisConsumer) []string {
	keys := make([]string, 0, len(consumers))
	for key := range consumers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
