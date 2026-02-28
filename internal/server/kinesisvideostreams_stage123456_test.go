package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKinesisVideoStreamsStage12LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisVideoStreamsRequest(t, ts, "/createStream", []byte(`{"StreamName":"stage-stream","DataRetentionInHours":48}`))
	assertStatus(t, resp, http.StatusOK)
	createStreamPayload := decodeKinesisVideoStreamsPayload(t, resp)
	streamARN := kinesisVideoStreamsPayloadStringFromMap(createStreamPayload, "StreamARN")
	if streamARN == "" {
		t.Fatalf("expected CreateStream to return StreamARN")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/describeStream", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/listStreams", []byte(`{"MaxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	listStreamsPayload := decodeKinesisVideoStreamsPayload(t, resp)
	if _, ok := listStreamsPayload["StreamInfoList"].([]any); !ok {
		t.Fatalf("expected ListStreams to return StreamInfoList")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/updateStream", []byte(`{"StreamARN":"`+streamARN+`","MediaType":"video/h265"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/getDataEndpoint", []byte(`{"StreamARN":"`+streamARN+`","APIName":"GET_MEDIA"}`))
	assertStatus(t, resp, http.StatusOK)
	getDataEndpointPayload := decodeKinesisVideoStreamsPayload(t, resp)
	if kinesisVideoStreamsPayloadStringFromMap(getDataEndpointPayload, "DataEndpoint") == "" {
		t.Fatalf("expected GetDataEndpoint to return DataEndpoint")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/createSignalingChannel", []byte(`{"ChannelName":"stage-channel"}`))
	assertStatus(t, resp, http.StatusOK)
	createChannelPayload := decodeKinesisVideoStreamsPayload(t, resp)
	channelARN := kinesisVideoStreamsPayloadStringFromMap(createChannelPayload, "ChannelARN")
	if channelARN == "" {
		t.Fatalf("expected CreateSignalingChannel to return ChannelARN")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/describeSignalingChannel", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/listSignalingChannels", []byte(`{"MaxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	listChannelsPayload := decodeKinesisVideoStreamsPayload(t, resp)
	if _, ok := listChannelsPayload["ChannelInfoList"].([]any); !ok {
		t.Fatalf("expected ListSignalingChannels to return ChannelInfoList")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/updateSignalingChannel", []byte(`{"ChannelARN":"`+channelARN+`","SingleMasterConfiguration":{"MessageTtlSeconds":120}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/getSignalingChannelEndpoint", []byte(`{"ChannelARN":"`+channelARN+`","SingleMasterChannelEndpointConfiguration":{"Protocols":["WSS"],"Role":"MASTER"}}`))
	assertStatus(t, resp, http.StatusOK)
	endpointPayload := decodeKinesisVideoStreamsPayload(t, resp)
	if _, ok := endpointPayload["ResourceEndpointList"].([]any); !ok {
		t.Fatalf("expected GetSignalingChannelEndpoint to return ResourceEndpointList")
	}
}

func TestKinesisVideoStreamsStage34ConfigurationAndEdgeSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	streamARN := "arn:aws:kinesisvideo:us-east-1:123456789012:stream/stage-config-stream/000001"
	channelARN := "arn:aws:kinesisvideo:us-east-1:123456789012:channel/stage-config-channel/000001"

	resp := kinesisVideoStreamsRequest(t, ts, "/updateNotificationConfiguration", []byte(`{"StreamARN":"`+streamARN+`","NotificationDestinationConfig":{"Uri":"arn:aws:sns:us-east-1:123456789012:stackyard-kvs"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/describeNotificationConfiguration", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/updateMediaStorageConfiguration", []byte(`{"ChannelARN":"`+channelARN+`","Status":"ENABLED","StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/describeMediaStorageConfiguration", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/updateImageGenerationConfiguration", []byte(`{"StreamARN":"`+streamARN+`","Status":"ENABLED","ImageSelectorType":"SERVER_TIMESTAMP","DestinationConfig":{"Uri":"s3://stackyard-images"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/describeImageGenerationConfiguration", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/describeMappedResourceConfiguration", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/updateStreamStorageConfiguration", []byte(`{"StreamARN":"`+streamARN+`","Status":"ENABLED"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/describeStreamStorageConfiguration", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/startEdgeConfigurationUpdate", []byte(`{"ChannelARN":"`+channelARN+`","EdgeConfig":{"RecorderConfig":{},"UploaderConfig":{}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/describeEdgeConfiguration", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/listEdgeAgentConfigurations", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/deleteEdgeConfiguration", []byte(`{"ChannelARN":"`+channelARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestKinesisVideoStreamsStage56RetentionTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	streamARN := "arn:aws:kinesisvideo:us-east-1:123456789012:stream/stage-tag-stream/000001"

	resp := kinesisVideoStreamsRequest(t, ts, "/updateDataRetention", []byte(`{"StreamARN":"`+streamARN+`","Operation":"INCREASE_DATA_RETENTION","DataRetentionChangeInHours":4}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/tagStream", []byte(`{"StreamARN":"`+streamARN+`","Tags":{"owner":"qa","env":"stage"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/listTagsForStream", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	listTagsForStreamPayload := decodeKinesisVideoStreamsPayload(t, resp)
	if !strings.Contains(kinesisVideoStreamsCompactJSON(t, listTagsForStreamPayload), "owner") {
		t.Fatalf("expected ListTagsForStream to include owner tag")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/tagResource", []byte(`{"ResourceARN":"`+streamARN+`","Tags":{"app":"stackyard"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/listTagsForResource", []byte(`{"ResourceARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	listTagsForResourcePayload := decodeKinesisVideoStreamsPayload(t, resp)
	if !strings.Contains(kinesisVideoStreamsCompactJSON(t, listTagsForResourcePayload), "stackyard") {
		t.Fatalf("expected ListTagsForResource to include stackyard tag")
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/untagStream", []byte(`{"StreamARN":"`+streamARN+`","TagKeyList":["owner"]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/untagResource", []byte(`{"ResourceARN":"`+streamARN+`","TagKeyList":["app"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = kinesisVideoStreamsRequest(t, ts, "/unknown-kinesisvideo-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/listStreams",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"kinesisvideo",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = kinesisVideoStreamsRequest(t, ts, "/deleteStream", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/deleteStream", []byte(`{"StreamARN":"`+streamARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func decodeKinesisVideoStreamsPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func kinesisVideoStreamsPayloadStringFromMap(payload map[string]any, key string) string {
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

func kinesisVideoStreamsCompactJSON(t *testing.T, payload map[string]any) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return string(data)
}
