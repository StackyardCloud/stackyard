package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"golang.org/x/net/websocket"
)

func kvsCreateChannel(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	resp := kinesisVideoStreamsRequest(t, ts, "/createSignalingChannel", []byte(`{"ChannelName":"`+name+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var out struct{ ChannelARN string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.ChannelARN == "" {
		t.Fatalf("CreateSignalingChannel: %v %+v", err, out)
	}
	return out.ChannelARN
}

// kvsDialSignaling opens the WSS relay with a SigV4-presigned URL, as the
// WebRTC SDKs do.
func kvsDialSignaling(t *testing.T, ts *httptest.Server, channelARN, clientID string) (*websocket.Conn, error) {
	t.Helper()
	target := ts.URL + kvsSignalingWSPath + "?X-Amz-Expires=299&X-Amz-ChannelARN=" + channelARN
	if clientID != "" {
		target += "&X-Amz-ClientId=" + clientID
	}
	req, _ := http.NewRequest(http.MethodGet, target, nil)
	creds := aws.Credentials{AccessKeyID: testAccessKey, SecretAccessKey: testSecretKey}
	signed, _, err := v4.NewSigner().PresignHTTP(context.Background(), creds, req, "UNSIGNED-PAYLOAD", "kinesisvideo", testRegion, time.Now())
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	cfg, err := websocket.NewConfig("ws"+strings.TrimPrefix(signed, "http"), ts.URL)
	if err != nil {
		t.Fatalf("ws config: %v", err)
	}
	return websocket.DialConfig(cfg)
}

func TestKinesisVideoSignalingStage0Catalog(t *testing.T) {
	if len(kinesisVideoSignalingOperations) != 2 || len(kinesisVideoSignalingOperationByName) != 2 {
		t.Fatalf("signaling ops: %d", len(kinesisVideoSignalingOperations))
	}
	if len(kinesisVideoWebRTCStorageOperations) != 2 || len(kinesisVideoWebRTCStorageOperationByName) != 2 {
		t.Fatalf("webrtc storage ops: %d", len(kinesisVideoWebRTCStorageOperations))
	}
	for _, name := range []string{"GetIceServerConfig", "SendAlexaOfferToMaster", "JoinStorageSession", "JoinStorageSessionAsViewer"} {
		if _, ok := kvsSignalingRoutes[map[string]string{
			"GetIceServerConfig":         "/v1/get-ice-server-config",
			"SendAlexaOfferToMaster":     "/v1/send-alexa-offer-to-master",
			"JoinStorageSession":         "/joinStorageSession",
			"JoinStorageSessionAsViewer": "/joinStorageSessionAsViewer",
		}[name]]; !ok {
			t.Fatalf("missing route for %s", name)
		}
	}
}

func TestKinesisVideoSignalingStage0Routing(t *testing.T) {
	ts := newKVSMediaTestServer(t)

	// The control plane still owns its own kinesisvideo-signed POSTs.
	resp := kinesisVideoStreamsRequest(t, ts, "/listStreams", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "StreamInfoList") {
		t.Fatalf("ListStreams lost to the signaling router")
	}

	// A signaling path signed for another service is not claimed.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v1/get-ice-server-config", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "s3")
	if strings.Contains(string(mustBody(t, resp)), "IceServerList") {
		t.Fatalf("s3-signed request was served as GetIceServerConfig")
	}

	// The WSS endpoint requires an upgrade.
	resp, err := http.Get(ts.URL + kvsSignalingWSPath)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()

	// Unsigned data-plane calls are rejected.
	resp, err = http.Post(ts.URL+"/v1/get-ice-server-config", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned GetIceServerConfig: %d", resp.StatusCode)
	}
	resp.Body.Close()
}
