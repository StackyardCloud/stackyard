package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	// kvsSignalingWSPath is the WSS endpoint GetSignalingChannelEndpoint hands
	// out (kinesisvideosignaling_relay.go).
	kvsSignalingWSPath = "/_stackyard/kinesisvideosignaling"
	// kvsSignalingMaxBody bounds data-plane request bodies; the largest
	// member (MessagePayload) is capped at 10,000 characters.
	kvsSignalingMaxBody = 64 << 10
	kvsIceServerTTL     = 300
	// kvsAlexaAnswerWait is how long SendAlexaOfferToMaster waits for a
	// connected master to answer over the relay.
	kvsAlexaAnswerWait = 3 * time.Second
)

var (
	kvsChannelARNPattern = regexp.MustCompile(`^arn:(aws[a-zA-Z-]*):kinesisvideo:[a-z0-9-]+:[0-9]+:[a-z]+/[a-zA-Z0-9_.-]+/[0-9]+$`)
	kvsClientIDPattern   = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,256}$`)
	kvsBase64Pattern     = regexp.MustCompile(`^[a-zA-Z0-9+/=]+$`)
)

// kvsValidMessagePayload checks the MessagePayload shape: base64, 1-10000
// characters.
func kvsValidMessagePayload(v string) bool {
	return len(v) <= 10000 && kvsBase64Pattern.MatchString(v)
}

// kvsSignalingRoutes maps the POST paths of Kinesis Video Signaling and
// WebRTC Storage to their operations. They share the "kinesisvideo" signing
// name with the control plane, so this router sits ahead of the media and
// control-plane routers.
var kvsSignalingRoutes = func() map[string]string {
	out := map[string]string{}
	for _, op := range kinesisVideoSignalingOperations {
		out[op.URI] = op.Name
	}
	for _, op := range kinesisVideoWebRTCStorageOperations {
		out[op.URI] = op.Name
	}
	return out
}()

func (s *Server) handleKinesisVideoSignalingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if r.Method == http.MethodGet && path == kvsSignalingWSPath {
		s.kvsSignalingConnect(w, r)
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	action, ok := kvsSignalingRoutes[path]
	if !ok {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "kinesisvideo" {
		return false
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one: chunked bodies of unknown length are refused.
	if r.ContentLength < 0 || r.ContentLength > kvsSignalingMaxBody {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "request body must declare a Content-Length of at most 64 KiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "kinesisvideo"); !ok {
		respondKinesisVideoStreamsError(w, status, code, msg)
		return true
	}
	payload, err := parseKinesisVideoStreamsPayload(r)
	if err != nil {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "invalid JSON body")
		return true
	}
	var fault *kvsFault
	switch action {
	case "GetIceServerConfig":
		fault = s.kvsGetIceServerConfig(w, r, payload)
	case "SendAlexaOfferToMaster":
		fault = s.kvsSendAlexaOfferToMaster(w, payload)
	case "JoinStorageSession", "JoinStorageSessionAsViewer":
		fault = s.kvsJoinStorageSession(w, action, payload)
	}
	if fault != nil {
		respondKinesisVideoStreamsError(w, fault.status, fault.code, fault.msg)
	}
	return true
}

// kvsChannelLocked returns an existing signaling channel by ARN. Unlike the
// control plane's ensureChannelLocked, it never creates one.
func (s *kinesisVideoStreamsStore) kvsChannelLocked(arn string) (map[string]any, *kvsFault) {
	if !kvsChannelARNPattern.MatchString(arn) || len(arn) > 1024 {
		return nil, kvsInvalidArgument("ChannelARN must be a valid signaling channel ARN")
	}
	channel := s.channels[s.channelsByARN[arn]]
	if channel == nil || kinesisVideoStreamsString(channel, "Status", "") != "ACTIVE" {
		return nil, kvsNotFound("The requested signaling channel was not found.")
	}
	return channel, nil
}

func (s *Server) kvsGetIceServerConfig(w http.ResponseWriter, r *http.Request, payload map[string]any) *kvsFault {
	arn := kinesisVideoStreamsPayloadString(payload, []string{"ChannelARN"}, "")
	clientID := kinesisVideoStreamsPayloadString(payload, []string{"ClientId"}, "")
	username := kinesisVideoStreamsPayloadString(payload, []string{"Username"}, "")
	if clientID != "" && !kvsClientIDPattern.MatchString(clientID) {
		return kvsInvalidArgument("ClientId must match [a-zA-Z0-9_.-]+ and be at most 256 characters")
	}
	if username != "" && !kvsClientIDPattern.MatchString(username) {
		return kvsInvalidArgument("Username must match [a-zA-Z0-9_.-]+ and be at most 256 characters")
	}
	if service := kinesisVideoStreamsPayloadString(payload, []string{"Service"}, "TURN"); service != "TURN" {
		return kvsInvalidArgument("Service must be TURN")
	}
	store := s.kinesisvideostreams
	store.mu.Lock()
	channel, fault := store.kvsChannelLocked(arn)
	var name string
	if channel != nil {
		name = kinesisVideoStreamsString(channel, "ChannelName", "")
	}
	store.mu.Unlock()
	if fault != nil {
		return fault
	}
	// ponytail: deterministic TURN credentials for a TURN server that does not
	// exist; clients get a valid IceServerList but relay candidates won't connect.
	user := name + "." + firstNonEmptyString(clientID, "master")
	if username != "" {
		user = username + "." + user
	}
	mac := hmac.New(sha256.New, []byte(s.secretKey))
	mac.Write([]byte(arn + "|" + user))
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = firstNonEmptyString(host, "localhost")
	respondKVSSignalingJSON(w, map[string]any{
		"IceServerList": []any{map[string]any{
			"Uris":     []string{"turn:" + host + ":3478?transport=udp", "turns:" + host + ":443?transport=tcp"},
			"Username": user,
			"Password": hex.EncodeToString(mac.Sum(nil))[:40],
			"Ttl":      kvsIceServerTTL,
		}},
	})
	return nil
}

// kvsDefaultAlexaAnswer is returned when no master answers over the relay.
var kvsDefaultAlexaAnswer = base64.StdEncoding.EncodeToString([]byte(
	"v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=stackyard\r\nt=0 0\r\na=group:BUNDLE 0\r\nm=video 9 UDP/TLS/RTP/SAVPF 96\r\nc=IN IP4 0.0.0.0\r\na=mid:0\r\na=recvonly\r\na=rtpmap:96 H264/90000\r\n",
))

func (s *Server) kvsSendAlexaOfferToMaster(w http.ResponseWriter, payload map[string]any) *kvsFault {
	arn := kinesisVideoStreamsPayloadString(payload, []string{"ChannelARN"}, "")
	sender := kinesisVideoStreamsPayloadString(payload, []string{"SenderClientId"}, "")
	offer := kinesisVideoStreamsPayloadString(payload, []string{"MessagePayload"}, "")
	if !kvsClientIDPattern.MatchString(sender) {
		return kvsInvalidArgument("SenderClientId is required and must match [a-zA-Z0-9_.-]+")
	}
	if !kvsValidMessagePayload(offer) {
		return kvsInvalidArgument("MessagePayload is required and must be base64, at most 10000 characters")
	}
	store := s.kinesisvideostreams
	store.mu.Lock()
	_, fault := store.kvsChannelLocked(arn)
	store.mu.Unlock()
	if fault != nil {
		return fault
	}
	// A master connected to the relay gets the offer and may answer it;
	// otherwise (or if it stays silent) the answer is a fixed recvonly SDP.
	answer, ok := s.kvsSignaling.offerToMaster(arn, sender, offer, kvsAlexaAnswerWait)
	if !ok {
		answer = kvsDefaultAlexaAnswer
	}
	respondKVSSignalingJSON(w, map[string]any{"Answer": answer})
	return nil
}

// kvsSignalingEndpoints builds GetSignalingChannelEndpoint's list: every
// protocol is served by this server (WSS by the relay path).
func kvsSignalingEndpoints(r *http.Request, payload map[string]any) ([]any, *kvsFault) {
	cfg, _ := payload["SingleMasterChannelEndpointConfiguration"].(map[string]any)
	role := strings.ToUpper(kinesisVideoStreamsPayloadString(cfg, []string{"Role"}, "MASTER"))
	if role != "MASTER" && role != "VIEWER" {
		return nil, kvsInvalidArgument("Role must be MASTER or VIEWER")
	}
	protocols := []string{"WSS", "HTTPS"}
	if raw, ok := cfg["Protocols"].([]any); ok && len(raw) > 0 {
		protocols = protocols[:0]
		for _, p := range raw {
			protocols = append(protocols, strings.ToUpper(kinesisVideoStreamsStringAny(p)))
		}
	}
	base := kvsRequestBaseURL(r)
	out := make([]any, 0, len(protocols))
	for _, p := range protocols {
		endpoint := base
		switch p {
		case "WSS":
			endpoint = "ws" + strings.TrimPrefix(base, "http") + kvsSignalingWSPath
		case "HTTPS", "WEBRTC":
		default:
			return nil, kvsInvalidArgument("Protocols must be WSS, HTTPS or WEBRTC")
		}
		out = append(out, map[string]any{"Protocol": p, "ResourceEndpoint": endpoint})
	}
	return out, nil
}

// respondKVSSignalingJSON writes a 200 signaling response; every member is
// server-built, and the content type is fixed.
func respondKVSSignalingJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
