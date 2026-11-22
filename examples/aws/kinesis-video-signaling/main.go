package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"golang.org/x/net/websocket"
)

type client struct {
	ctx    context.Context
	region string
	creds  aws.Credentials
}

type reply struct {
	status int
	body   []byte
}

// signalMessage is the WSS relay's message shape (both directions).
type signalMessage struct {
	Action            string `json:"action,omitempty"`
	MessageType       string `json:"messageType,omitempty"`
	MessagePayload    string `json:"messagePayload,omitempty"`
	RecipientClientID string `json:"recipientClientId,omitempty"`
	SenderClientID    string `json:"senderClientId,omitempty"`
}

// main creates a signaling channel, fetches its endpoints and ICE servers,
// relays an SDP offer/answer between a master and a viewer over WSS, and joins
// a WebRTC storage session linked to a stream.
func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	channel := getenv("STACKYARD_KVS_CHANNEL", "doorbell-1")
	provider := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	creds, err := provider.Retrieve(context.Background())
	if err != nil {
		exitf("credentials: %v", err)
	}
	c := client{ctx: context.Background(), region: getenv("AWS_REGION", "us-east-1"), creds: creds}
	fmt.Printf("Stackyard Kinesis Video signaling client using %s\n", endpoint)

	r := c.json(endpoint+"/createSignalingChannel", map[string]any{"ChannelName": channel})
	expect2xx("CreateSignalingChannel", r)
	var created struct{ ChannelARN string }
	_ = json.Unmarshal(r.body, &created)

	r = c.json(endpoint+"/getSignalingChannelEndpoint", map[string]any{
		"ChannelARN": created.ChannelARN,
		"SingleMasterChannelEndpointConfiguration": map[string]any{"Protocols": []string{"WSS", "HTTPS", "WEBRTC"}, "Role": "MASTER"},
	})
	expect2xx("GetSignalingChannelEndpoint", r)
	var eps struct {
		ResourceEndpointList []struct{ Protocol, ResourceEndpoint string }
	}
	_ = json.Unmarshal(r.body, &eps)
	byProtocol := map[string]string{}
	for _, ep := range eps.ResourceEndpointList {
		byProtocol[ep.Protocol] = ep.ResourceEndpoint
	}
	logf("endpoints: %v", byProtocol)

	r = c.json(byProtocol["HTTPS"]+"/v1/get-ice-server-config", map[string]any{"ChannelARN": created.ChannelARN, "ClientId": "viewer-1"})
	expect2xx("GetIceServerConfig", r)
	logf("ICE servers: %s", bytes.TrimSpace(r.body))

	master := c.dial(byProtocol["WSS"], created.ChannelARN, "")
	defer master.Close()
	viewer := c.dial(byProtocol["WSS"], created.ChannelARN, "viewer-1")
	defer viewer.Close()
	offer := base64.StdEncoding.EncodeToString([]byte("v=0\r\ns=viewer-offer\r\n"))
	send(viewer, signalMessage{Action: "SDP_OFFER", MessagePayload: offer})
	if got := receive(master); got.MessageType != "SDP_OFFER" || got.SenderClientID != "viewer-1" {
		exitf("master received %+v", got)
	}
	send(master, signalMessage{Action: "SDP_ANSWER", MessagePayload: offer, RecipientClientID: "viewer-1"})
	if got := receive(viewer); got.MessageType != "SDP_ANSWER" {
		exitf("viewer received %+v", got)
	}
	logf("relayed SDP offer and answer between master and viewer-1")

	r = c.json(endpoint+"/createStream", map[string]any{"StreamName": channel + "-storage", "DataRetentionInHours": 24})
	expect2xx("CreateStream", r)
	var stream struct{ StreamARN string }
	_ = json.Unmarshal(r.body, &stream)
	r = c.json(endpoint+"/updateMediaStorageConfiguration", map[string]any{
		"ChannelARN":                created.ChannelARN,
		"MediaStorageConfiguration": map[string]any{"StreamARN": stream.StreamARN, "Status": "ENABLED"},
	})
	expect2xx("UpdateMediaStorageConfiguration", r)
	r = c.json(byProtocol["WEBRTC"]+"/joinStorageSession", map[string]any{"channelArn": created.ChannelARN})
	expect2xx("JoinStorageSession", r)
	r = c.json(byProtocol["WEBRTC"]+"/joinStorageSessionAsViewer", map[string]any{"channelArn": created.ChannelARN, "clientId": "viewer-1"})
	expect2xx("JoinStorageSessionAsViewer", r)
	logf("Kinesis Video signaling example complete")
}

// dial opens the WSS relay with a SigV4-presigned URL, as the WebRTC SDKs do.
func (c client) dial(wss, channelARN, clientID string) *websocket.Conn {
	query := url.Values{"X-Amz-ChannelARN": {channelARN}, "X-Amz-Expires": {"299"}}
	if clientID != "" {
		query.Set("X-Amz-ClientId", clientID)
	}
	req, err := http.NewRequest(http.MethodGet, "http"+strings.TrimPrefix(wss, "ws")+"?"+query.Encode(), nil)
	if err != nil {
		exitf("build %s: %v", wss, err)
	}
	signed, _, err := v4.NewSigner().PresignHTTP(c.ctx, c.creds, req, "UNSIGNED-PAYLOAD", "kinesisvideo", c.region, time.Now())
	if err != nil {
		exitf("presign: %v", err)
	}
	cfg, err := websocket.NewConfig("ws"+strings.TrimPrefix(signed, "http"), "http://localhost")
	if err != nil {
		exitf("ws config: %v", err)
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		exitf("connect %s: %v", wss, err)
	}
	return conn
}

func send(conn *websocket.Conn, msg signalMessage) {
	if err := websocket.JSON.Send(conn, msg); err != nil {
		exitf("send: %v", err)
	}
}

func receive(conn *websocket.Conn) signalMessage {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var msg signalMessage
	if err := websocket.JSON.Receive(conn, &msg); err != nil {
		exitf("receive: %v", err)
	}
	return msg
}

func (c client) json(target string, payload any) reply {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("marshal %s: %v", target, err)
	}
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		exitf("build %s: %v", target, err)
	}
	req.Header.Set("Content-Type", "application/json")
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, c.creds, req, hex.EncodeToString(sum[:]), "kinesisvideo", c.region, time.Now()); err != nil {
		exitf("sign %s: %v", target, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", target, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s: %v", target, err)
	}
	return reply{status: resp.StatusCode, body: respBody}
}

func expect2xx(action string, r reply) {
	if r.status >= 200 && r.status < 300 {
		logf("%s returned %d", action, r.status)
		return
	}
	exitf("expected %s to return 2xx, got %d: %s", action, r.status, strings.TrimSpace(string(r.body)))
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
