package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKinesisStage12StreamLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisRequest(t, ts, "CreateStream", `{"StreamName":"stage-kinesis-stream","ShardCount":2}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "DescribeStreamSummary", `{"StreamName":"stage-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "DescribeStream", `{"StreamName":"stage-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "ListStreams", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "ListShards", `{"StreamName":"stage-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "DescribeLimits", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "DeleteStream", `{"StreamName":"stage-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestKinesisStage34DataAndConsumerSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisRequest(t, ts, "PutRecord", `{"StreamName":"stackyard-kinesis-stream","PartitionKey":"p","Data":"c3RhY2t5YXJk"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "PutRecords", `{"StreamName":"stackyard-kinesis-stream","Records":[{"PartitionKey":"p","Data":"c3RhY2t5YXJk"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "GetShardIterator", `{"StreamName":"stackyard-kinesis-stream","ShardId":"shardId-000000000000","ShardIteratorType":"TRIM_HORIZON"}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodeKinesisPayload(t, resp)
	it := kinesisPayloadStringFromMap(payload, "ShardIterator")
	if it == "" {
		t.Fatalf("expected GetShardIterator to return ShardIterator")
	}

	resp = kinesisRequest(t, ts, "GetRecords", `{"ShardIterator":"`+it+`","Limit":10}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "RegisterStreamConsumer", `{"StreamARN":"arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-kinesis-stream","ConsumerName":"stage-consumer"}`)
	assertStatus(t, resp, http.StatusOK)
	consumerPayload := decodeKinesisPayload(t, resp)
	consumer := kinesisPayloadMap(consumerPayload, "Consumer")
	consumerARN := kinesisPayloadStringFromMap(consumer, "ConsumerARN")
	if consumerARN == "" {
		t.Fatalf("expected RegisterStreamConsumer to return Consumer.ConsumerARN")
	}

	resp = kinesisRequest(t, ts, "DescribeStreamConsumer", `{"ConsumerARN":"`+consumerARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "ListStreamConsumers", `{"StreamARN":"arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "DeregisterStreamConsumer", `{"ConsumerARN":"`+consumerARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestKinesisStage56AdminTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-kinesis-stream"

	resp := kinesisRequest(t, ts, "AddTagsToStream", `{"StreamName":"stackyard-kinesis-stream","Tags":{"owner":"qa","env":"stage"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "ListTagsForStream", `{"StreamName":"stackyard-kinesis-stream"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "TagResource", `{"ResourceARN":"`+resourceARN+`","Tags":{"app":"stackyard"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "ListTagsForResource", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stackyard") {
		t.Fatalf("expected ListTagsForResource to include stackyard tag, got %q", body)
	}

	resp = kinesisRequest(t, ts, "RemoveTagsFromStream", `{"StreamName":"stackyard-kinesis-stream","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "UntagResource", `{"ResourceARN":"`+resourceARN+`","TagKeys":["app"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "PutResourcePolicy", `{"ResourceARN":"`+resourceARN+`","Policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "GetResourcePolicy", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "DeleteResourcePolicy", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "EnableEnhancedMonitoring", `{"StreamName":"stackyard-kinesis-stream","ShardLevelMetrics":["IncomingBytes"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "DisableEnhancedMonitoring", `{"StreamName":"stackyard-kinesis-stream","ShardLevelMetrics":["IncomingBytes"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "UpdateShardCount", `{"StreamName":"stackyard-kinesis-stream","TargetShardCount":3,"ScalingType":"UNIFORM_SCALING"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "SplitShard", `{"StreamName":"stackyard-kinesis-stream","ShardToSplit":"shardId-000000000000","NewStartingHashKey":"170141183460469231731687303715884105728"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "MergeShards", `{"StreamName":"stackyard-kinesis-stream","ShardToMerge":"shardId-000000000000","AdjacentShardToMerge":"shardId-000000000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "IncreaseStreamRetentionPeriod", `{"StreamName":"stackyard-kinesis-stream","RetentionPeriodHours":24}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "DecreaseStreamRetentionPeriod", `{"StreamName":"stackyard-kinesis-stream","RetentionPeriodHours":24}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "StartStreamEncryption", `{"StreamName":"stackyard-kinesis-stream","EncryptionType":"KMS","KeyId":"alias/aws/kinesis"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "StopStreamEncryption", `{"StreamName":"stackyard-kinesis-stream","EncryptionType":"KMS","KeyId":"alias/aws/kinesis"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "UpdateAccountSettings", `{"MinimumThroughputBillingCommitmentInput":{"Enabled":true}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "DescribeAccountSettings", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "UpdateMaxRecordSize", `{"StreamName":"stackyard-kinesis-stream","MaxRecordSizeInKiB":1024}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "UpdateStreamMode", `{"StreamName":"stackyard-kinesis-stream","StreamModeDetails":{"StreamMode":"ON_DEMAND"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisRequest(t, ts, "UpdateStreamWarmThroughput", `{"StreamARN":"`+resourceARN+`","CurrentWarmThroughput":{"ReadUnitsPerSecond":1,"WriteUnitsPerSecond":1},"DesiredWarmThroughput":{"ReadUnitsPerSecond":2,"WriteUnitsPerSecond":2}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Kinesis_20131202.ListStreams",
		},
		"kinesis",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeKinesisPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func kinesisPayloadMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func kinesisPayloadStringFromMap(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
