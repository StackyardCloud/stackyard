package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// kvsSignalingMaxFrame bounds one relay message; a payload is at most
// 10,000 base64 characters plus the JSON envelope.
const kvsSignalingMaxFrame = 16 << 10

// kvsSignalingHub relays WebRTC signaling between the single master and the
// viewers of each channel, the way the service's WSS endpoint does.
type kvsSignalingHub struct {
	mu       sync.Mutex
	channels map[string]*kvsSignalingPeers // by channel ARN
}

type kvsSignalingPeers struct {
	master  *kvsSignalingPeer
	viewers map[string]*kvsSignalingPeer
	pending map[string]chan string // SendAlexaOfferToMaster sender -> answer
}

type kvsSignalingPeer struct {
	conn *websocket.Conn
	mu   sync.Mutex // serializes writes
}

// kvsSignalingInbound is what a client sends; kvsSignalingOutbound is what it
// receives (messageType SDP_OFFER/SDP_ANSWER/ICE_CANDIDATE/STATUS_RESPONSE).
type kvsSignalingInbound struct {
	Action            string `json:"action"`
	MessagePayload    string `json:"messagePayload"`
	RecipientClientID string `json:"recipientClientId"`
	CorrelationID     string `json:"correlationId"`
}

type kvsSignalingOutbound struct {
	MessageType    string         `json:"messageType"`
	MessagePayload string         `json:"messagePayload,omitempty"`
	SenderClientID string         `json:"senderClientId,omitempty"`
	StatusResponse map[string]any `json:"statusResponse,omitempty"`
}

func newKVSSignalingHub() *kvsSignalingHub {
	return &kvsSignalingHub{channels: map[string]*kvsSignalingPeers{}}
}

func (p *kvsSignalingPeer) send(msg kvsSignalingOutbound) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = websocket.JSON.Send(p.conn, msg)
}

func (h *kvsSignalingHub) peersLocked(arn string) *kvsSignalingPeers {
	peers := h.channels[arn]
	if peers == nil {
		peers = &kvsSignalingPeers{viewers: map[string]*kvsSignalingPeer{}, pending: map[string]chan string{}}
		h.channels[arn] = peers
	}
	return peers
}

// join registers a connection. A new master replaces (and disconnects) the
// previous one, as does a viewer reusing a client ID.
func (h *kvsSignalingHub) join(arn, clientID string, peer *kvsSignalingPeer) {
	h.mu.Lock()
	peers := h.peersLocked(arn)
	old := peers.master
	if clientID == "" {
		peers.master = peer
	} else {
		old = peers.viewers[clientID]
		peers.viewers[clientID] = peer
	}
	h.mu.Unlock()
	if old != nil {
		_ = old.conn.Close()
	}
}

func (h *kvsSignalingHub) leave(arn, clientID string, peer *kvsSignalingPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	peers := h.channels[arn]
	if peers == nil {
		return
	}
	if clientID == "" && peers.master == peer {
		peers.master = nil
	} else if clientID != "" && peers.viewers[clientID] == peer {
		delete(peers.viewers, clientID)
	}
	if peers.master == nil && len(peers.viewers) == 0 && len(peers.pending) == 0 {
		delete(h.channels, arn)
	}
}

// route forwards one message: viewers always talk to the master; the master
// addresses a viewer (or a pending Alexa offer) by recipientClientId.
func (h *kvsSignalingHub) route(arn, senderID string, msg kvsSignalingInbound) string {
	switch msg.Action {
	case "SDP_OFFER", "SDP_ANSWER", "ICE_CANDIDATE":
	default:
		return "invalid action: must be SDP_OFFER, SDP_ANSWER or ICE_CANDIDATE"
	}
	if !kvsValidMessagePayload(msg.MessagePayload) {
		return "invalid messagePayload: must be base64, at most 10000 characters"
	}
	h.mu.Lock()
	peers := h.peersLocked(arn)
	var target *kvsSignalingPeer
	if senderID != "" {
		target = peers.master
	} else if answer, ok := peers.pending[msg.RecipientClientID]; ok && msg.Action == "SDP_ANSWER" {
		delete(peers.pending, msg.RecipientClientID)
		h.mu.Unlock()
		answer <- msg.MessagePayload
		return ""
	} else {
		target = peers.viewers[msg.RecipientClientID]
	}
	h.mu.Unlock()
	if target == nil {
		return "the recipient is not connected"
	}
	target.send(kvsSignalingOutbound{MessageType: msg.Action, MessagePayload: msg.MessagePayload, SenderClientID: senderID})
	return ""
}

// offerToMaster sends an Alexa offer to the connected master and waits for
// its SDP_ANSWER addressed to sender. ok is false when no master is
// connected or it does not answer in time.
func (h *kvsSignalingHub) offerToMaster(arn, sender, offer string, wait time.Duration) (string, bool) {
	answer := make(chan string, 1)
	h.mu.Lock()
	peers := h.channels[arn]
	if peers == nil || peers.master == nil {
		h.mu.Unlock()
		return "", false
	}
	master := peers.master
	peers.pending[sender] = answer
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		if peers.pending[sender] == answer {
			delete(peers.pending, sender)
		}
		h.mu.Unlock()
	}()
	master.send(kvsSignalingOutbound{MessageType: "SDP_OFFER", MessagePayload: offer, SenderClientID: sender})
	select {
	case got := <-answer:
		return got, true
	case <-time.After(wait):
		return "", false
	}
}

// kvsHijackWriter gives x/net/websocket the http.Hijacker it type-asserts,
// reaching through Stackyard's wrapping writers via ResponseController.
type kvsHijackWriter struct{ http.ResponseWriter }

func (w kvsHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

// kvsSignalingConnect serves the WSS endpoint: a presigned (SigV4 query)
// upgrade with X-Amz-ChannelARN, plus X-Amz-ClientId for a viewer.
func (s *Server) kvsSignalingConnect(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "invalid request: the signaling endpoint requires a WebSocket upgrade")
		return
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "kinesisvideo"); !ok {
		respondKinesisVideoStreamsError(w, status, code, msg)
		return
	}
	query := r.URL.Query()
	arn := query.Get("X-Amz-ChannelARN")
	clientID := query.Get("X-Amz-ClientId")
	if clientID != "" && !kvsClientIDPattern.MatchString(clientID) {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "invalid X-Amz-ClientId: must match [a-zA-Z0-9_.-]+")
		return
	}
	store := s.kinesisvideostreams
	store.mu.Lock()
	_, fault := store.kvsChannelLocked(arn)
	store.mu.Unlock()
	if fault != nil {
		respondKinesisVideoStreamsError(w, fault.status, fault.code, fault.msg)
		return
	}
	server := websocket.Server{
		// SigV4 already authenticated the request; there is no browser Origin to check.
		Handshake: func(*websocket.Config, *http.Request) error { return nil },
		Handler: func(conn *websocket.Conn) {
			conn.MaxPayloadBytes = kvsSignalingMaxFrame
			peer := &kvsSignalingPeer{conn: conn}
			s.kvsSignaling.join(arn, clientID, peer)
			defer func() {
				s.kvsSignaling.leave(arn, clientID, peer)
				_ = conn.Close()
			}()
			for {
				var msg kvsSignalingInbound
				err := websocket.JSON.Receive(conn, &msg)
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				switch {
				case errors.As(err, &syntaxErr) || errors.As(err, &typeErr):
					msg = kvsSignalingInbound{Action: "invalid"}
				case err != nil:
					return
				}
				if problem := s.kvsSignaling.route(arn, clientID, msg); problem != "" && msg.CorrelationID != "" {
					peer.send(kvsSignalingOutbound{MessageType: "STATUS_RESPONSE", StatusResponse: map[string]any{
						"correlationId": msg.CorrelationID,
						"errorType":     "InvalidArgumentException",
						"statusCode":    "400",
						"description":   problem,
					}})
				}
			}
		},
	}
	server.ServeHTTP(kvsHijackWriter{w}, r)
}
