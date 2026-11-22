package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func kvsFragmentNumbersIn(t *testing.T, mkv []byte) []string {
	t.Helper()
	var numbers []string
	for _, part := range strings.Split(string(mkv), "AWS_KINESISVIDEO_FRAGMENT_NUMBER")[1:] {
		// TagString element: 0x44 0x87, 8-byte size, then the number.
		idx := strings.Index(part, "\x44\x87")
		if idx < 0 || len(part) < idx+10 {
			t.Fatalf("malformed fragment tag")
		}
		rest := part[idx+10:]
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		numbers = append(numbers, rest[:end])
	}
	return numbers
}

func TestKinesisVideoMediaStage1PutAndGetMedia(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateStream(t, ts, "media-stream", 24)

	status, acks := kvsPutMedia(t, ts, "media-stream", nil, kvsTestMKV(0, 2000, 4000))
	if status != http.StatusOK || len(acks) != 9 {
		t.Fatalf("PutMedia: status %d, %d acks %v", status, len(acks), acks)
	}
	if acks[0]["EventType"] != "BUFFERING" || acks[1]["EventType"] != "RECEIVED" || acks[2]["EventType"] != "PERSISTED" {
		t.Fatalf("ack order: %v", acks[:3])
	}
	first, _ := acks[0]["FragmentNumber"].(string)
	second, _ := acks[3]["FragmentNumber"].(string)

	get := func(selector string) (*http.Response, []byte) {
		resp := kvsMediaJSON(t, ts, "/getMedia", `{"StreamARN":"`+arn+`","StartSelector":`+selector+`}`)
		return resp, mustBody(t, resp)
	}
	resp, body := get(`{"StartSelectorType":"EARLIEST"}`)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "video/webm" {
		t.Fatalf("GetMedia EARLIEST: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if got := kvsFragmentNumbersIn(t, body); len(got) != 3 || got[0] != first {
		t.Fatalf("EARLIEST fragments %v", got)
	}
	if clusters, err := parseKVSMKV(body); err != nil || len(clusters) != 3 {
		t.Fatalf("GetMedia output should parse as 3 fragments: %v %d", err, len(clusters))
	}
	_, body = get(`{"StartSelectorType":"FRAGMENT_NUMBER","AfterFragmentNumber":"` + second + `"}`)
	if got := kvsFragmentNumbersIn(t, body); len(got) != 2 || got[0] != second {
		t.Fatalf("FRAGMENT_NUMBER fragments %v", got)
	}
	_, body = get(`{"StartSelectorType":"CONTINUATION_TOKEN","ContinuationToken":"` + second + `"}`)
	if got := kvsFragmentNumbersIn(t, body); len(got) != 1 {
		t.Fatalf("CONTINUATION_TOKEN should resume after the token: %v", got)
	}
	// Producer time = start (1700000000) + cluster timecode; 1700000002 skips the first.
	_, body = get(`{"StartSelectorType":"PRODUCER_TIMESTAMP","StartTimestamp":1700000002}`)
	if got := kvsFragmentNumbersIn(t, body); len(got) != 2 || got[0] != second {
		t.Fatalf("PRODUCER_TIMESTAMP fragments %v", got)
	}

	// NOW waits for fragments that arrive after the call.
	done := make(chan []string, 1)
	go func() {
		resp := kvsMediaJSON(t, ts, "/getMedia", `{"StreamName":"media-stream","StartSelector":{"StartSelectorType":"NOW"}}`)
		done <- kvsFragmentNumbersIn(t, mustBody(t, resp))
	}()
	time.Sleep(200 * time.Millisecond)
	if status, _ := kvsPutMedia(t, ts, "media-stream", map[string]string{"x-amzn-producer-start-timestamp": "1700000100"}, kvsTestMKV(0)); status != http.StatusOK {
		t.Fatalf("second PutMedia: %d", status)
	}
	if got := <-done; len(got) != 1 {
		t.Fatalf("NOW should return only the new fragment, got %v", got)
	}

	// The control plane hands out this server as the data endpoint.
	resp = kinesisVideoStreamsRequest(t, ts, "/getDataEndpoint", []byte(`{"StreamARN":"`+arn+`","APIName":"GET_MEDIA"}`))
	assertStatus(t, resp, http.StatusOK)
	var endpoint struct{ DataEndpoint string }
	if err := json.Unmarshal(mustBody(t, resp), &endpoint); err != nil || endpoint.DataEndpoint != ts.URL {
		t.Fatalf("DataEndpoint = %q, want %q", endpoint.DataEndpoint, ts.URL)
	}
}

func TestKinesisVideoMediaStage1Errors(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	kvsCreateStream(t, ts, "err-stream", 24)

	for name, tc := range map[string]struct {
		body   string
		status int
		code   string
	}{
		"missing selector":    {`{"StreamName":"err-stream"}`, http.StatusBadRequest, "InvalidArgumentException"},
		"bad selector type":   {`{"StreamName":"err-stream","StartSelector":{"StartSelectorType":"LATEST"}}`, http.StatusBadRequest, "InvalidArgumentException"},
		"missing fragment no": {`{"StreamName":"err-stream","StartSelector":{"StartSelectorType":"FRAGMENT_NUMBER"}}`, http.StatusBadRequest, "InvalidArgumentException"},
		"unknown fragment":    {`{"StreamName":"err-stream","StartSelector":{"StartSelectorType":"FRAGMENT_NUMBER","AfterFragmentNumber":"1"}}`, http.StatusNotFound, "ResourceNotFoundException"},
		"no stream given":     {`{"StartSelector":{"StartSelectorType":"EARLIEST"}}`, http.StatusBadRequest, "InvalidArgumentException"},
		"unknown stream":      {`{"StreamName":"nope","StartSelector":{"StartSelectorType":"EARLIEST"}}`, http.StatusNotFound, "ResourceNotFoundException"},
		"bad json":            {`{`, http.StatusBadRequest, "InvalidArgumentException"},
	} {
		resp := kvsMediaJSON(t, ts, "/getMedia", tc.body)
		body := mustBody(t, resp)
		if resp.StatusCode != tc.status || !strings.Contains(string(body), tc.code) {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, body)
		}
	}

	if status, _ := kvsPutMedia(t, ts, "err-stream", map[string]string{"x-amzn-fragment-timecode-type": ""}, kvsTestMKV(0)); status != http.StatusBadRequest {
		t.Fatalf("missing timecode type: %d", status)
	}
	if status, _ := kvsPutMedia(t, ts, "nope", nil, kvsTestMKV(0)); status != http.StatusNotFound {
		t.Fatalf("unknown stream PutMedia: %d", status)
	}
	if _, acks := kvsPutMedia(t, ts, "err-stream", nil, []byte("garbage")); len(acks) != 1 || acks[0]["EventType"] != "ERROR" || acks[0]["ErrorId"] != float64(4006) {
		t.Fatalf("invalid MKV acks: %v", acks)
	}
	// A cluster earlier than the last stored one is rejected after the good ones.
	if _, acks := kvsPutMedia(t, ts, "err-stream", nil, kvsTestMKV(5000, 1000)); len(acks) != 4 || acks[3]["ErrorId"] != float64(4004) {
		t.Fatalf("decreasing timecode acks: %v", acks)
	}

	// Without retention nothing is PERSISTED, but live readers still get media.
	kvsCreateStream(t, ts, "live-only", 0)
	_, acks := kvsPutMedia(t, ts, "live-only", nil, kvsTestMKV(0))
	for _, ack := range acks {
		if ack["EventType"] == "PERSISTED" {
			t.Fatalf("retention 0 must not ack PERSISTED: %v", acks)
		}
	}
	resp := kvsMediaJSON(t, ts, "/getMedia", `{"StreamName":"live-only","StartSelector":{"StartSelectorType":"EARLIEST"}}`)
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("AWS_KINESISVIDEO_FRAGMENT_NUMBER")) {
		t.Fatalf("live GetMedia on retention-0 stream: %d", resp.StatusCode)
	}

	// Deleting the stream drops its media.
	resp = kinesisVideoStreamsRequest(t, ts, "/deleteStream", []byte(`{"StreamName":"err-stream"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kvsMediaJSON(t, ts, "/getMedia", `{"StreamName":"err-stream","StartSelector":{"StartSelectorType":"EARLIEST"}}`)
	assertStatus(t, resp, http.StatusNotFound)
}
