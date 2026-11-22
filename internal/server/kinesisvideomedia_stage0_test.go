package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// kvsTestMKV builds a minimal valid stream: one video track and one
// SimpleBlock (+33 ticks) per cluster at the given cluster timecodes (ms).
func kvsTestMKV(clusterMillis ...int64) []byte {
	ebml := mkvEncode(mkvIDEBML, append(mkvEncode(0x4282, []byte("matroska")), mkvEncode(0x4287, []byte{4})...))
	info := mkvEncode(mkvIDInfo, mkvEncode(mkvIDTimecodeScale, []byte{0x0F, 0x42, 0x40}))
	trackEntry := mkvEncode(0xAE, append(append(mkvEncode(0xD7, []byte{1}), mkvEncode(0x83, []byte{1})...), mkvEncode(0x86, []byte("V_MPEG4/ISO/AVC"))...))
	out := append([]byte{}, ebml...)
	out = append(out, 0x18, 0x53, 0x80, 0x67, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF) // unknown-size segment
	out = append(out, info...)
	out = append(out, mkvEncode(mkvIDTracks, trackEntry)...)
	for _, ms := range clusterMillis {
		tc := make([]byte, 8)
		binary.BigEndian.PutUint64(tc, uint64(ms))
		block := []byte{0x81, 0x00, 0x21, 0x80, 'f', 'r', 'a', 'm', 'e'}
		out = append(out, mkvEncode(mkvIDCluster, append(mkvEncode(mkvIDTimecode, tc), mkvEncode(mkvIDSimpleBlock, block)...))...)
	}
	return out
}

func newKVSMediaTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func kvsMediaJSON(t *testing.T, ts *httptest.Server, path, body string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "kinesisvideo")
}

func kvsCreateStream(t *testing.T, ts *httptest.Server, name string, retention int) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"StreamName": name, "DataRetentionInHours": retention, "MediaType": "video/h264"})
	resp := kinesisVideoStreamsRequest(t, ts, "/createStream", body)
	assertStatus(t, resp, http.StatusOK)
	var out struct{ StreamARN string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.StreamARN == "" {
		t.Fatalf("CreateStream: %v %+v", err, out)
	}
	return out.StreamARN
}

// kvsPutMedia sends an MKV body and returns the decoded ack events.
func kvsPutMedia(t *testing.T, ts *httptest.Server, stream string, headers map[string]string, mkv []byte) (int, []map[string]any) {
	t.Helper()
	h := map[string]string{"x-amzn-stream-name": stream, "x-amzn-fragment-timecode-type": "RELATIVE", "x-amzn-producer-start-timestamp": "1700000000"}
	for k, v := range headers {
		h[k] = v
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/putMedia", mkv, h, "kinesisvideo")
	body := mustBody(t, resp)
	var acks []map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	for dec.More() {
		var ack map[string]any
		if err := dec.Decode(&ack); err != nil {
			break
		}
		acks = append(acks, ack)
	}
	return resp.StatusCode, acks
}

func TestKinesisVideoMediaStage0Catalog(t *testing.T) {
	if len(kinesisVideoMediaOperations) != 2 || len(kinesisVideoMediaOperationByName) != 2 {
		t.Fatalf("expected GetMedia and PutMedia, got %d", len(kinesisVideoMediaOperations))
	}
	for _, name := range []string{"GetMedia", "PutMedia"} {
		if _, ok := kinesisVideoMediaOperationByName[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
	// The data planes sit ahead of the control plane and must not shadow it.
	ts := newKVSMediaTestServer(t)
	resp := kinesisVideoStreamsRequest(t, ts, "/listStreams", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = kinesisVideoStreamsRequest(t, ts, "/getMediaX", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	// Signed for another service: not claimed by the media router.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/getMedia", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "s3")
	if body := mustBody(t, resp); strings.Contains(string(body), "StartSelector") {
		t.Fatalf("s3-signed /getMedia should not reach the media plane: %s", body)
	}
}

func TestKinesisVideoMediaStage0MKVParser(t *testing.T) {
	clusters, err := parseKVSMKV(kvsTestMKV(0, 2000, 4000))
	if err != nil || len(clusters) != 3 {
		t.Fatalf("parse: %v, %d clusters", err, len(clusters))
	}
	if clusters[1].timecode != 2000 || clusters[1].scale != 1_000_000 || clusters[1].maxBlockTicks != 33 {
		t.Fatalf("cluster fields: %+v", clusters[1])
	}
	// A truncated body keeps the clusters before the damage.
	full := kvsTestMKV(0, 2000)
	clusters, err = parseKVSMKV(full[:len(full)-3])
	if err == nil || len(clusters) != 1 {
		t.Fatalf("truncated: err=%v clusters=%d", err, len(clusters))
	}
	if _, err := parseKVSMKV([]byte("not mkv at all")); err == nil {
		t.Fatalf("expected invalid MKV error")
	}
	// Each stored fragment is itself a parseable MKV with KVS tags.
	frag := kvsFragmentMKV(clusters[0], kvsFragmentNumber(7), time.Unix(1700000000, 0), time.Unix(1700000000, 0))
	again, err := parseKVSMKV(frag)
	if err != nil || len(again) != 1 || !strings.Contains(string(frag), "AWS_KINESISVIDEO_FRAGMENT_NUMBER") {
		t.Fatalf("fragment MKV did not round-trip: %v %d", err, len(again))
	}
}
