package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKinesisVideoWebRTCStorageStage1JoinSession(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	arn := kvsCreateChannel(t, ts, "storage-chan")
	join := func(path, body string) (int, string) {
		resp := kvsMediaJSON(t, ts, path, body)
		return resp.StatusCode, string(mustBody(t, resp))
	}

	// Media storage is disabled until a stream is linked.
	resp := kinesisVideoStreamsRequest(t, ts, "/describeMediaStorageConfiguration", []byte(`{"ChannelARN":"`+arn+`"}`))
	var described struct{ MediaStorageConfiguration map[string]any }
	if err := json.Unmarshal(mustBody(t, resp), &described); err != nil || described.MediaStorageConfiguration["Status"] != "DISABLED" {
		t.Fatalf("default media storage: %v %+v", err, described)
	}
	if status, body := join("/joinStorageSession", `{"channelArn":"`+arn+`"}`); status != http.StatusBadRequest || !strings.Contains(body, "InvalidArgumentException") {
		t.Fatalf("join without storage: %d %s", status, body)
	}

	streamARN := kvsCreateStream(t, ts, "storage-stream", 24)
	resp = kinesisVideoStreamsRequest(t, ts, "/updateMediaStorageConfiguration", []byte(`{"ChannelARN":"`+arn+`","MediaStorageConfiguration":{"StreamARN":"`+streamARN+`","Status":"ENABLED"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	if status, body := join("/joinStorageSession", `{"channelArn":"`+arn+`"}`); status != http.StatusOK {
		t.Fatalf("JoinStorageSession: %d %s", status, body)
	}
	for _, id := range []string{"v1", "v2", "v3", "v1"} {
		if status, body := join("/joinStorageSessionAsViewer", `{"channelArn":"`+arn+`","clientId":"`+id+`"}`); status != http.StatusOK {
			t.Fatalf("viewer %s: %d %s", id, status, body)
		}
	}
	if status, body := join("/joinStorageSessionAsViewer", `{"channelArn":"`+arn+`","clientId":"v4"}`); status != http.StatusBadRequest || !strings.Contains(body, "ClientLimitExceededException") {
		t.Fatalf("fourth viewer: %d %s", status, body)
	}
	srv.kinesisvideostreams.mu.Lock()
	session := srv.kinesisvideostreams.storageSessions[arn]
	srv.kinesisvideostreams.mu.Unlock()
	if session == nil || session.MasterJoinedAt.IsZero() || len(session.Viewers) != 3 {
		t.Fatalf("recorded session: %+v", session)
	}

	missing := strings.Replace(arn, "storage-chan", "nope", 1)
	for name, tc := range map[string]struct {
		path, body string
		want       int
	}{
		"viewer without client id": {"/joinStorageSessionAsViewer", `{"channelArn":"` + arn + `"}`, http.StatusBadRequest},
		"bad arn":                  {"/joinStorageSession", `{"channelArn":"not-an-arn"}`, http.StatusBadRequest},
		"unknown channel":          {"/joinStorageSession", `{"channelArn":"` + missing + `"}`, http.StatusNotFound},
	} {
		if status, body := join(tc.path, tc.body); status != tc.want {
			t.Fatalf("%s: %d %s", name, status, body)
		}
	}

	// Disabling media storage (stream still linked) turns sessions away.
	resp = kinesisVideoStreamsRequest(t, ts, "/updateMediaStorageConfiguration", []byte(`{"ChannelARN":"`+arn+`","MediaStorageConfiguration":{"StreamARN":"`+streamARN+`","Status":"DISABLED"}}`))
	resp.Body.Close()
	if status, _ := join("/joinStorageSession", `{"channelArn":"`+arn+`"}`); status != http.StatusBadRequest {
		t.Fatalf("join with storage disabled: %d", status)
	}
	resp = kinesisVideoStreamsRequest(t, ts, "/updateMediaStorageConfiguration", []byte(`{"ChannelARN":"`+arn+`","MediaStorageConfiguration":{"StreamARN":"`+streamARN+`","Status":"ENABLED"}}`))
	resp.Body.Close()

	// Deleting the stream turns storage sessions away again.
	resp = kinesisVideoStreamsRequest(t, ts, "/deleteStream", []byte(`{"StreamARN":"`+streamARN+`"}`))
	resp.Body.Close()
	if status, _ := join("/joinStorageSession", `{"channelArn":"`+arn+`"}`); status != http.StatusBadRequest {
		t.Fatalf("join after stream delete: %d", status)
	}
}
