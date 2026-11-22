package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestKinesisVideoSignalingStage1Endpoints(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateChannel(t, ts, "sig-endpoints")

	resp := kinesisVideoStreamsRequest(t, ts, "/getSignalingChannelEndpoint", []byte(`{"ChannelARN":"`+arn+`","SingleMasterChannelEndpointConfiguration":{"Protocols":["WSS","HTTPS","WEBRTC"],"Role":"VIEWER"}}`))
	assertStatus(t, resp, http.StatusOK)
	var out struct {
		ResourceEndpointList []struct{ Protocol, ResourceEndpoint string }
	}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || len(out.ResourceEndpointList) != 3 {
		t.Fatalf("endpoints: %v %+v", err, out)
	}
	want := map[string]string{"WSS": "ws" + strings.TrimPrefix(ts.URL, "http") + kvsSignalingWSPath, "HTTPS": ts.URL, "WEBRTC": ts.URL}
	for _, ep := range out.ResourceEndpointList {
		if want[ep.Protocol] != ep.ResourceEndpoint {
			t.Fatalf("%s endpoint = %q, want %q", ep.Protocol, ep.ResourceEndpoint, want[ep.Protocol])
		}
	}
	for _, bad := range []string{`"Role":"OWNER"`, `"Protocols":["FTP"],"Role":"MASTER"`} {
		resp = kinesisVideoStreamsRequest(t, ts, "/getSignalingChannelEndpoint", []byte(`{"ChannelARN":"`+arn+`","SingleMasterChannelEndpointConfiguration":{`+bad+`}}`))
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
}

func TestKinesisVideoSignalingStage1IceServerConfig(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateChannel(t, ts, "sig-ice")

	get := func(body string) (int, string) {
		resp := kvsMediaJSON(t, ts, "/v1/get-ice-server-config", body)
		return resp.StatusCode, string(mustBody(t, resp))
	}
	status, body := get(`{"ChannelARN":"` + arn + `","ClientId":"viewer-1","Service":"TURN"}`)
	var out struct {
		IceServerList []struct {
			Uris               []string
			Username, Password string
			Ttl                int
		}
	}
	if status != http.StatusOK || json.Unmarshal([]byte(body), &out) != nil || len(out.IceServerList) != 1 {
		t.Fatalf("GetIceServerConfig: %d %s", status, body)
	}
	ice := out.IceServerList[0]
	if ice.Ttl != kvsIceServerTTL || len(ice.Uris) != 2 || !strings.HasPrefix(ice.Uris[0], "turn:") || ice.Username != "sig-ice.viewer-1" || len(ice.Password) != 40 {
		t.Fatalf("ice server: %+v", ice)
	}
	if _, again := get(`{"ChannelARN":"` + arn + `","ClientId":"viewer-1"}`); again != body {
		t.Fatalf("credentials are not deterministic: %s vs %s", again, body)
	}

	missing := strings.Replace(arn, "sig-ice", "nope", 1)
	for name, tc := range map[string]struct {
		body string
		want int
	}{
		"unknown channel": {`{"ChannelARN":"` + missing + `"}`, http.StatusNotFound},
		"bad arn":         {`{"ChannelARN":"arn:aws:s3:::bucket"}`, http.StatusBadRequest},
		"no arn":          {`{}`, http.StatusBadRequest},
		"bad client id":   {`{"ChannelARN":"` + arn + `","ClientId":"has space"}`, http.StatusBadRequest},
		"bad service":     {`{"ChannelARN":"` + arn + `","Service":"STUN"}`, http.StatusBadRequest},
		"bad json":        {`{`, http.StatusBadRequest},
	} {
		if status, body := get(tc.body); status != tc.want {
			t.Fatalf("%s: %d %s", name, status, body)
		}
	}
}

func TestKinesisVideoSignalingStage1AlexaOfferWithoutMaster(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateChannel(t, ts, "sig-alexa")

	resp := kvsMediaJSON(t, ts, "/v1/send-alexa-offer-to-master", `{"ChannelARN":"`+arn+`","SenderClientId":"alexa","MessagePayload":"b2ZmZXI="}`)
	assertStatus(t, resp, http.StatusOK)
	var out struct{ Answer string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.Answer != kvsDefaultAlexaAnswer {
		t.Fatalf("answer: %v %q", err, out.Answer)
	}
	for _, body := range []string{
		`{"ChannelARN":"` + arn + `","MessagePayload":"b2ZmZXI="}`,
		`{"ChannelARN":"` + arn + `","SenderClientId":"alexa","MessagePayload":"<html>"}`,
		`{"ChannelARN":"` + arn + `","SenderClientId":"alexa","MessagePayload":"` + strings.Repeat("A", 10001) + `"}`,
	} {
		resp = kvsMediaJSON(t, ts, "/v1/send-alexa-offer-to-master", body)
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
	// Bodies past the cap are refused before parsing.
	resp = kvsMediaJSON(t, ts, "/v1/send-alexa-offer-to-master", `{"MessagePayload":"`+strings.Repeat("A", kvsSignalingMaxBody)+`"}`)
	if resp.StatusCode < 400 {
		t.Fatalf("oversized body: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func kvsReceive(t *testing.T, conn *websocket.Conn) kvsSignalingOutbound {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var msg kvsSignalingOutbound
	if err := websocket.JSON.Receive(conn, &msg); err != nil {
		t.Fatalf("receive: %v", err)
	}
	return msg
}

func TestKinesisVideoSignalingStage1Relay(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateChannel(t, ts, "sig-relay")

	master, err := kvsDialSignaling(t, ts, arn, "")
	if err != nil {
		t.Fatalf("master dial: %v", err)
	}
	defer master.Close()
	viewer, err := kvsDialSignaling(t, ts, arn, "viewer-1")
	if err != nil {
		t.Fatalf("viewer dial: %v", err)
	}
	defer viewer.Close()

	// viewer -> master carries the viewer's ID.
	_ = websocket.JSON.Send(viewer, kvsSignalingInbound{Action: "SDP_OFFER", MessagePayload: "b2ZmZXI="})
	if msg := kvsReceive(t, master); msg.MessageType != "SDP_OFFER" || msg.SenderClientID != "viewer-1" || msg.MessagePayload != "b2ZmZXI=" {
		t.Fatalf("master got %+v", msg)
	}
	// master -> viewer by recipientClientId.
	_ = websocket.JSON.Send(master, kvsSignalingInbound{Action: "SDP_ANSWER", MessagePayload: "YW5zd2Vy", RecipientClientID: "viewer-1"})
	if msg := kvsReceive(t, viewer); msg.MessageType != "SDP_ANSWER" || msg.MessagePayload != "YW5zd2Vy" || msg.SenderClientID != "" {
		t.Fatalf("viewer got %+v", msg)
	}
	// Unknown recipient or a bad action gets a status response when correlated.
	_ = websocket.JSON.Send(master, kvsSignalingInbound{Action: "ICE_CANDIDATE", MessagePayload: "Y2FuZA==", RecipientClientID: "ghost", CorrelationID: "c1"})
	if msg := kvsReceive(t, master); msg.MessageType != "STATUS_RESPONSE" || msg.StatusResponse["correlationId"] != "c1" {
		t.Fatalf("status response: %+v", msg)
	}
	_ = websocket.JSON.Send(viewer, kvsSignalingInbound{Action: "GO_AWAY", MessagePayload: "eA==", CorrelationID: "c2"})
	if msg := kvsReceive(t, viewer); msg.StatusResponse["statusCode"] != "400" {
		t.Fatalf("bad action: %+v", msg)
	}

	// An Alexa offer reaches the connected master, whose answer is returned.
	go func() {
		var offer kvsSignalingOutbound
		_ = master.SetReadDeadline(time.Now().Add(3 * time.Second))
		if websocket.JSON.Receive(master, &offer) == nil && offer.SenderClientID == "alexa-1" {
			_ = websocket.JSON.Send(master, kvsSignalingInbound{Action: "SDP_ANSWER", MessagePayload: "bWFzdGVyLWFuc3dlcg==", RecipientClientID: "alexa-1"})
		}
	}()
	resp := kvsMediaJSON(t, ts, "/v1/send-alexa-offer-to-master", `{"ChannelARN":"`+arn+`","SenderClientId":"alexa-1","MessagePayload":"b2ZmZXI="}`)
	assertStatus(t, resp, http.StatusOK)
	var out struct{ Answer string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.Answer != "bWFzdGVyLWFuc3dlcg==" {
		t.Fatalf("relayed answer: %v %q", err, out.Answer)
	}
}

func TestKinesisVideoSignalingStage1RelayRejects(t *testing.T) {
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateChannel(t, ts, "sig-reject")

	if _, err := kvsDialSignaling(t, ts, strings.Replace(arn, "sig-reject", "nope", 1), ""); err == nil {
		t.Fatalf("dial to an unknown channel succeeded")
	}
	if _, err := kvsDialSignaling(t, ts, arn, "bad%20id"); err == nil {
		t.Fatalf("dial with an invalid client id succeeded")
	}
	// An unsigned upgrade is refused.
	cfg, _ := websocket.NewConfig("ws"+strings.TrimPrefix(ts.URL, "http")+kvsSignalingWSPath+"?X-Amz-ChannelARN="+arn, ts.URL)
	if _, err := websocket.DialConfig(cfg); err == nil {
		t.Fatalf("unsigned dial succeeded")
	}
}
