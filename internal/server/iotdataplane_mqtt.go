package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// MQTT side of the IoT data plane. Stackyard runs no MQTT broker, so device
// clients are simulated through a Stackyard-only endpoint (as
// apigatewaymanagementapi does for WebSocket clients):
//
//	POST /_stackyard/iot-data/connections           connect a client with subscriptions
//	GET  /_stackyard/iot-data/connections/{id}/messages   drain its inbox
//
// Publish delivers to every connected client whose subscription matches (and
// queues QoS 1 messages for disconnected persistent sessions), keeps retained
// messages (an empty retained payload clears the topic), and applies shadow
// topics ($aws/things/{thing}/shadow[/name/{name}]/update|delete|get),
// answering on their /accepted or /rejected topics. SendDirectMessage
// delivers to one connected client.

const (
	iotdMaxTopicBytes     = 256
	iotdMaxTopicLevels    = 8 // AWS allows 7 forward slashes
	iotdMaxRetained       = 1000
	iotdMaxRetainedBytes  = 16 << 20
	iotdMaxClients        = 1000
	iotdMaxSubscriptions  = 50
	iotdMaxInbox          = 100
	iotdMaxPropertyHeader = 8 << 10
	iotdMaxQueryValue     = 1024
)

type iotdMessage struct {
	Topic           string `json:"topic"`
	Payload         []byte `json:"payload"`
	Qos             int    `json:"qos"`
	Retain          bool   `json:"retain,omitempty"`
	ContentType     string `json:"contentType,omitempty"`
	ResponseTopic   string `json:"responseTopic,omitempty"`
	UserProperties  []byte `json:"userProperties,omitempty"`
	CorrelationData []byte `json:"correlationData,omitempty"`
	PayloadFormat   string `json:"payloadFormatIndicator,omitempty"`
	MessageExpiry   *int64 `json:"messageExpiry,omitempty"`
	modified        time.Time
}

type iotdClient struct {
	id, thing                         string
	connected, cleanSession           bool
	keepAlive                         int
	sessionExpiry                     int64
	sourceIP                          string
	sourcePort                        int
	connectedSince, disconnectedSince time.Time
	disconnectReason                  string
	subs                              map[string]int // topic filter -> qos
	inbox                             []iotdMessage
}

// iotdValidTopic checks a topic name for publishing (no wildcards).
func iotdValidTopic(topic string) *iotdErr {
	if topic == "" || len(topic) > iotdMaxTopicBytes || !utf8.ValidString(topic) {
		return iotdInvalid("topic must be 1-%d bytes of UTF-8", iotdMaxTopicBytes)
	}
	if strings.ContainsAny(topic, "+#\x00") {
		return iotdInvalid("topic must not contain wildcards")
	}
	if strings.Count(topic, "/") >= iotdMaxTopicLevels {
		return iotdInvalid("topic can have at most %d forward slashes", iotdMaxTopicLevels-1)
	}
	if strings.HasPrefix(topic, "$") && !strings.HasPrefix(topic, "$aws/things/") && !strings.HasPrefix(topic, "$aws/rules/") {
		return iotdErrorf("InvalidRequestException", "reserved topic %s can't be published to", topic)
	}
	return nil
}

func iotdValidFilter(filter string) bool {
	if filter == "" || len(filter) > iotdMaxTopicBytes || !utf8.ValidString(filter) || strings.Count(filter, "/") >= iotdMaxTopicLevels {
		return false
	}
	levels := strings.Split(filter, "/")
	for i, l := range levels {
		if (strings.Contains(l, "+") && l != "+") || (strings.Contains(l, "#") && (l != "#" || i != len(levels)-1)) {
			return false
		}
	}
	return true
}

// iotdTopicMatch applies MQTT wildcard rules; filters starting with a
// wildcard don't match $-topics.
func iotdTopicMatch(filter, topic string) bool {
	if strings.HasPrefix(topic, "$") && (strings.HasPrefix(filter, "+") || strings.HasPrefix(filter, "#")) {
		return false
	}
	f, t := strings.Split(filter, "/"), strings.Split(topic, "/")
	for i, level := range f {
		if level == "#" {
			return true
		}
		if i >= len(t) || (level != "+" && level != t[i]) {
			return false
		}
	}
	return len(f) == len(t)
}

// iotdParseMessage builds a message from a Publish / SendDirectMessage request.
func iotdParseMessage(topic string, r *http.Request, body []byte, publish bool) (iotdMessage, *iotdErr) {
	if err := iotdValidTopic(topic); err != nil {
		return iotdMessage{}, err
	}
	q := r.URL.Query()
	msg := iotdMessage{Topic: topic, Payload: append([]byte(nil), body...), modified: time.Now().UTC()}
	for _, key := range []string{"contentType", "responseTopic"} {
		if len(q.Get(key)) > iotdMaxQueryValue {
			return msg, iotdInvalid("%s can be at most %d characters", key, iotdMaxQueryValue)
		}
	}
	msg.ContentType, msg.ResponseTopic = q.Get("contentType"), q.Get("responseTopic")
	if msg.ResponseTopic != "" {
		if err := iotdValidTopic(msg.ResponseTopic); err != nil {
			return msg, err
		}
	}
	if publish {
		if v := q.Get("qos"); v != "" {
			n, err := strconv.ParseInt(v, 10, 32)
			if err != nil || n < 0 || n > 1 {
				return msg, iotdInvalid("qos must be 0 or 1")
			}
			msg.Qos = int(n)
		}
		retain, err := iotdBool(q, "retain")
		if err != nil {
			return msg, err
		}
		msg.Retain = retain
		if v := q.Get("messageExpiry"); v != "" {
			n, perr := strconv.ParseInt(v, 10, 64)
			if perr != nil || n < 0 {
				return msg, iotdInvalid("messageExpiry must be a non-negative integer")
			}
			msg.MessageExpiry = &n
		}
	}
	if v := r.Header.Get("X-Amz-Mqtt5-Payload-Format-Indicator"); v != "" {
		if v != "UNSPECIFIED_BYTES" && v != "UTF8_DATA" {
			return msg, iotdInvalid("payloadFormatIndicator must be UNSPECIFIED_BYTES or UTF8_DATA")
		}
		if v == "UTF8_DATA" && !utf8.Valid(body) {
			return msg, iotdInvalid("the payload is not valid UTF-8")
		}
		msg.PayloadFormat = v
	}
	if v := r.Header.Get("X-Amz-Mqtt5-User-Properties"); v != "" {
		props, err := iotdHeaderBytes(v)
		var list []map[string]string
		if err != nil || json.Unmarshal(props, &list) != nil {
			return msg, iotdInvalid("userProperties must be a base64-encoded JSON array of {\"key\":\"value\"} objects")
		}
		msg.UserProperties = props
	}
	if v := r.Header.Get("X-Amz-Mqtt5-Correlation-Data"); v != "" {
		data, err := iotdHeaderBytes(v)
		if err != nil {
			return msg, iotdInvalid("correlationData must be base64-encoded")
		}
		msg.CorrelationData = data
	}
	return msg, nil
}

func iotdHeaderBytes(v string) ([]byte, error) {
	if len(v) > iotdMaxPropertyHeader {
		return nil, strconv.ErrRange
	}
	return base64.StdEncoding.DecodeString(v)
}

func (st *iotdStore) publish(msg iotdMessage) *iotdErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if msg.Retain {
		if err := st.retainLocked(msg); err != nil {
			return err
		}
	}
	st.deliverLocked(msg)
	st.shadowTopicLocked(msg)
	return nil
}

func (st *iotdStore) retainLocked(msg iotdMessage) *iotdErr {
	old := st.retained[msg.Topic]
	oldSize := 0
	if old != nil {
		oldSize = iotdMessageSize(*old)
	}
	if len(msg.Payload) == 0 { // an empty retained message clears the topic
		delete(st.retained, msg.Topic)
		st.retainedSz -= oldSize
		return nil
	}
	if old == nil && len(st.retained) >= iotdMaxRetained {
		return iotdErrorf("ThrottlingException", "at most %d retained messages can be stored", iotdMaxRetained)
	}
	size := iotdMessageSize(msg)
	if st.retainedSz-oldSize+size > iotdMaxRetainedBytes {
		return iotdErrorf("ThrottlingException", "retained messages would exceed %d bytes", iotdMaxRetainedBytes)
	}
	st.retainedSz += size - oldSize
	stored := msg
	st.retained[msg.Topic] = &stored
	return nil
}

func iotdMessageSize(m iotdMessage) int {
	return len(m.Topic) + len(m.Payload) + len(m.ContentType) + len(m.ResponseTopic) + len(m.UserProperties) + len(m.CorrelationData)
}

// deliverLocked queues msg for every client subscribed to its topic; a
// full inbox drops its oldest message.
func (st *iotdStore) deliverLocked(msg iotdMessage) {
	for _, c := range st.clients {
		qos, ok := -1, false
		for filter, q := range c.subs {
			if iotdTopicMatch(filter, msg.Topic) && q > qos {
				qos, ok = q, true
			}
		}
		if !ok || (!c.connected && (qos < 1 || msg.Qos < 1)) {
			continue
		}
		out := msg
		out.Qos = min(msg.Qos, qos)
		c.push(out)
	}
}

func (c *iotdClient) push(msg iotdMessage) {
	inbox := append(c.inbox, msg)
	if len(inbox) > iotdMaxInbox {
		inbox = append([]iotdMessage(nil), inbox[len(inbox)-iotdMaxInbox:]...)
	}
	c.inbox = inbox
}

// shadowTopicLocked applies a publish to a shadow's update/delete/get topic
// and answers on /accepted or /rejected, as the MQTT shadow service does.
func (st *iotdStore) shadowTopicLocked(msg iotdMessage) {
	rest, ok := strings.CutPrefix(msg.Topic, "$aws/things/")
	if !ok {
		return
	}
	parts := strings.Split(rest, "/")
	var thing, name, action string
	switch {
	case len(parts) == 3 && parts[1] == "shadow":
		thing, action = parts[0], parts[2]
	case len(parts) == 5 && parts[1] == "shadow" && parts[2] == "name":
		thing, name, action = parts[0], parts[3], parts[4]
	default:
		return
	}
	if !iotdThingNameRe.MatchString(thing) || (name != "" && !iotdShadowNameRe.MatchString(name)) {
		return
	}
	var doc map[string]any
	var err *iotdErr
	switch action {
	case "update":
		doc, err = st.updateShadowLocked(thing, name, msg.Payload)
	case "delete":
		doc, err = st.deleteShadowLocked(thing, name)
	case "get":
		doc, err = st.getShadowLocked(thing, name)
	default:
		return
	}
	reply := msg.Topic + "/accepted"
	if err != nil {
		reply = msg.Topic + "/rejected"
		doc = map[string]any{"code": err.status, "message": err.msg, "timestamp": time.Now().Unix()}
	}
	payload, _ := json.Marshal(doc)
	st.deliverLocked(iotdMessage{Topic: reply, Payload: payload, Qos: msg.Qos, modified: time.Now().UTC()})
}

func (st *iotdStore) getRetained(topic string) (map[string]any, *iotdErr) {
	if err := iotdValidTopic(topic); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	m := st.retained[topic]
	if m == nil {
		return nil, iotdErrorf("ResourceNotFoundException", "No retained message exists for topic %s", topic)
	}
	out := map[string]any{"topic": m.Topic, "payload": m.Payload, "qos": m.Qos, "lastModifiedTime": m.modified.UnixMilli()}
	if len(m.UserProperties) > 0 {
		out["userProperties"] = m.UserProperties
	}
	return out, nil
}

func (st *iotdStore) listRetained(start, size int) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	topics := make([]string, 0, len(st.retained))
	for t := range st.retained {
		topics = append(topics, t)
	}
	sort.Strings(topics)
	from, to, next := iotdSlice(len(topics), start, size)
	list := make([]any, 0, to-from)
	for _, t := range topics[from:to] {
		m := st.retained[t]
		list = append(list, map[string]any{"topic": t, "payloadSize": len(m.Payload), "qos": m.Qos, "lastModifiedTime": m.modified.UnixMilli()})
	}
	out := map[string]any{"retainedTopics": list}
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

func iotdNoClient(id string) *iotdErr {
	return iotdErrorf("ResourceNotFoundException", "no connection for client %s", id)
}

func (st *iotdStore) getConnection(id string, socket bool) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.clients[id]
	if c == nil {
		return nil, iotdNoClient(id)
	}
	out := map[string]any{
		"clientId": c.id, "connected": c.connected, "cleanSession": c.cleanSession,
		"keepAliveDuration": c.keepAlive, "connectedSince": c.connectedSince.UnixMilli(),
	}
	if c.thing != "" {
		out["thingName"] = c.thing
	}
	if c.sessionExpiry > 0 {
		out["sessionExpiry"] = c.sessionExpiry
	}
	if !c.connected {
		out["disconnectedSince"], out["disconnectReason"] = c.disconnectedSince.UnixMilli(), c.disconnectReason
	}
	if socket {
		out["sourceIp"], out["sourcePort"], out["targetIp"], out["targetPort"] = c.sourceIP, c.sourcePort, "127.0.0.1", 8883
	}
	return out, nil
}

func (st *iotdStore) deleteConnection(id string, clean bool) *iotdErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.clients[id]
	if c == nil || !c.connected {
		return iotdNoClient(id)
	}
	if clean || c.cleanSession {
		delete(st.clients, id)
		return nil
	}
	c.connected, c.disconnectedSince, c.disconnectReason = false, time.Now().UTC(), "API_INITIATED_DISCONNECT"
	return nil
}

func (st *iotdStore) listSubscriptions(id string, start, size int) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.clients[id]
	if c == nil {
		return nil, iotdNoClient(id)
	}
	filters := make([]string, 0, len(c.subs))
	for f := range c.subs {
		filters = append(filters, f)
	}
	sort.Strings(filters)
	from, to, next := iotdSlice(len(filters), start, size)
	list := make([]any, 0, to-from)
	for _, f := range filters[from:to] {
		list = append(list, map[string]any{"topicFilter": f, "qos": c.subs[f]})
	}
	out := map[string]any{"subscriptions": list}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func (st *iotdStore) sendDirect(id string, msg iotdMessage) (map[string]any, *iotdErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.clients[id]
	if c == nil || !c.connected {
		return nil, iotdNoClient(id)
	}
	c.push(msg)
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return map[string]any{"message": "Message delivered to client " + id, "traceId": hex.EncodeToString(buf)}, nil
}

// iotdConnectRequest is the body of POST /_stackyard/iot-data/connections.
type iotdConnectRequest struct {
	ClientID          string `json:"clientId"`
	ThingName         string `json:"thingName"`
	CleanSession      *bool  `json:"cleanSession"`
	KeepAliveDuration int    `json:"keepAliveDuration"`
	SessionExpiry     int64  `json:"sessionExpiry"`
	Subscriptions     []struct {
		TopicFilter string `json:"topicFilter"`
		Qos         int    `json:"qos"`
	} `json:"subscriptions"`
}

// handleIoTDataConnect simulates an MQTT CONNECT plus SUBSCRIBEs. A client
// id that is already connected is taken over (the old connection drops);
// a persistent session keeps its subscriptions and queued messages.
// Matching retained messages are delivered on subscribe.
func (s *Server) handleIoTDataConnect(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength < 0 || r.ContentLength > 64<<10 {
		iotdRespondError(w, iotdInvalid("request body must declare a Content-Length of at most %d bytes", 64<<10))
		return
	}
	var req iotdConnectRequest
	raw, err := readBodyBytes(r)
	if err != nil || json.Unmarshal(raw, &req) != nil {
		iotdRespondError(w, iotdInvalid("invalid JSON body"))
		return
	}
	if req.ClientID == "" || len(req.ClientID) > 128 || strings.HasPrefix(req.ClientID, "$") {
		iotdRespondError(w, iotdInvalid("clientId must be 1-128 characters and not start with $"))
		return
	}
	if req.ThingName != "" && !iotdThingNameRe.MatchString(req.ThingName) {
		iotdRespondError(w, iotdInvalid("thingName must be 1-128 characters of [a-zA-Z0-9:_-]"))
		return
	}
	if len(req.Subscriptions) > iotdMaxSubscriptions || req.KeepAliveDuration < 0 || req.KeepAliveDuration > 1200 || req.SessionExpiry < 0 {
		iotdRespondError(w, iotdInvalid("at most %d subscriptions; keepAliveDuration 0-1200", iotdMaxSubscriptions))
		return
	}
	for _, sub := range req.Subscriptions {
		if !iotdValidFilter(sub.TopicFilter) || sub.Qos < 0 || sub.Qos > 1 {
			iotdRespondError(w, iotdInvalid("invalid subscription %q (qos 0 or 1)", sub.TopicFilter))
			return
		}
	}
	host, portStr, _ := net.SplitHostPort(r.RemoteAddr)
	port, _ := strconv.Atoi(portStr)
	clean := req.CleanSession == nil || *req.CleanSession
	st := s.iotData
	st.mu.Lock()
	c := st.clients[req.ClientID]
	if c == nil && len(st.clients) >= iotdMaxClients {
		st.mu.Unlock()
		iotdRespondError(w, iotdErrorf("ThrottlingException", "at most %d clients can be connected", iotdMaxClients))
		return
	}
	next := &iotdClient{subs: map[string]int{}}
	if c != nil && !clean {
		next.inbox = c.inbox
		for f, q := range c.subs {
			next.subs[f] = q
		}
	}
	merged := len(next.subs)
	for _, sub := range req.Subscriptions {
		if _, dup := next.subs[sub.TopicFilter]; !dup {
			merged++
		}
	}
	if merged > iotdMaxSubscriptions {
		st.mu.Unlock()
		iotdRespondError(w, iotdInvalid("a client can have at most %d subscriptions", iotdMaxSubscriptions))
		return
	}
	next.id, next.thing, next.connected, next.cleanSession = req.ClientID, req.ThingName, true, clean
	next.keepAlive, next.sessionExpiry, next.sourceIP, next.sourcePort = req.KeepAliveDuration, req.SessionExpiry, host, port
	next.connectedSince = time.Now().UTC()
	for _, sub := range req.Subscriptions {
		next.subs[sub.TopicFilter] = sub.Qos
		for topic, m := range st.retained {
			if iotdTopicMatch(sub.TopicFilter, topic) {
				out := *m
				out.Qos = min(out.Qos, sub.Qos)
				next.push(out)
			}
		}
	}
	st.clients[req.ClientID] = next
	since := next.connectedSince
	st.mu.Unlock()
	body, _ := json.Marshal(map[string]any{"clientId": req.ClientID, "connectedSince": since.UnixMilli()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(body)
}

// handleIoTDataMessages returns and clears a simulated client's inbox.
// Payloads are base64 in the JSON body, so caller bytes are never echoed raw.
func (s *Server) handleIoTDataMessages(w http.ResponseWriter, r *http.Request) {
	st := s.iotData
	st.mu.Lock()
	c := st.clients[r.PathValue("clientId")]
	var msgs []iotdMessage
	if c != nil {
		msgs, c.inbox = c.inbox, nil
	}
	st.mu.Unlock()
	if c == nil {
		iotdRespondError(w, iotdNoClient(r.PathValue("clientId")))
		return
	}
	if msgs == nil {
		msgs = []iotdMessage{}
	}
	body, _ := json.Marshal(map[string]any{"messages": msgs})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
