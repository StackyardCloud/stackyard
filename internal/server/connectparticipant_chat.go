package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var cnpMessageTypes = map[string]bool{
	"text/plain": true, "text/markdown": true, "application/json": true,
	"application/vnd.amazonaws.connect.message.interactive.response": true,
}

var cnpEventTypes = map[string]string{
	"application/vnd.amazonaws.connect.event.typing":                        "TYPING",
	"application/vnd.amazonaws.connect.event.connection.acknowledged":       "CONNECTION_ACK",
	"application/vnd.amazonaws.connect.event.message.delivered":             "MESSAGE_DELIVERED",
	"application/vnd.amazonaws.connect.event.message.read":                  "MESSAGE_READ",
	"application/vnd.amazonaws.connect.event.participant.idle":              "EVENT",
	"application/vnd.amazonaws.connect.event.participant.returned":          "EVENT",
	"application/vnd.amazonaws.connect.event.participant.autodisconnection": "EVENT",
}

func (st *cnpStore) createConnection(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p := st.byPartToken[req.bearer]
	if p == nil || p.disconnected {
		return nil, cnpDenied("the participant token is not valid")
	}
	if !p.chat.ended.IsZero() {
		return nil, cnpDenied("the chat has ended")
	}
	types, _ := req.in["Type"].([]any)
	connect, _ := req.in["ConnectParticipant"].(bool)
	if len(types) == 0 && !connect {
		return nil, cnpValidation("specify Type, ConnectParticipant, or both")
	}
	out := map[string]any{}
	for _, t := range types {
		switch t {
		case "WEBRTC_CONNECTION":
			return nil, cnpValidation("contact %s is a chat; it has no WebRTC connection", p.chat.contactID)
		case "CONNECTION_CREDENTIALS":
			if p.connToken == "" {
				p.connToken = cnpToken()
				st.byConnToken[p.connToken] = p
			}
			out["ConnectionCredentials"] = map[string]any{"ConnectionToken": p.connToken, "Expiry": cnpTime(time.Now().Add(cnpCredentialLifetime))}
		}
	}
	// ponytail: no WebSocket streaming; clients poll GetTranscript with the credentials.
	if len(types) > 0 && out["ConnectionCredentials"] == nil {
		return nil, cnpValidation("Stackyard serves chat through CONNECTION_CREDENTIALS only; it doesn't stream over WebSocket")
	}
	if connect && !p.connected {
		p.connected = true
		if p.role != "CUSTOMER" {
			_, _ = st.appendLocked(p.chat, &cnpItem{typ: "EVENT", contentType: "application/vnd.amazonaws.connect.event.participant.connected", participant: p})
		}
	}
	return out, nil
}

func (st *cnpStore) disconnect(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	token := cnpStr(req.in, "ClientToken")
	// A retry with the same client token succeeds after the disconnect.
	if p := st.byConnToken[req.bearer]; p != nil && p.disconnected && token != "" && token == p.disconnectToken {
		return nil, nil
	}
	p, err := st.byConnectionLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	chat := p.chat
	p.disconnected = true
	p.disconnectedAt = time.Now().UTC()
	p.disconnectToken = token
	delete(st.byPartToken, p.partToken)
	if chat.ended.IsZero() {
		_, _ = st.appendLocked(chat, &cnpItem{typ: "PARTICIPANT_LEFT", contentType: "application/vnd.amazonaws.connect.event.participant.left", participant: p})
		if p.role == "CUSTOMER" {
			st.endLocked(chat)
		}
	}
	return nil, nil
}

// idempotent returns the item an earlier call with the same client token made.
func (chat *cnpChat) idempotent(token string, p *cnpParticipant) *cnpItem {
	if it := chat.clientTokens[token]; token != "" && it != nil && it.participant == p {
		return it
	}
	return nil
}

func (st *cnpStore) sendMessage(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	ct := cnpStr(req.in, "ContentType")
	if !cnpMessageTypes[ct] {
		return nil, cnpValidation("ContentType must be text/plain, text/markdown, application/json or application/vnd.amazonaws.connect.message.interactive.response")
	}
	if ct == "application/json" && !json.Valid([]byte(cnpStr(req.in, "Content"))) {
		return nil, cnpValidation("Content isn't valid JSON")
	}
	token := cnpStr(req.in, "ClientToken")
	it := p.chat.idempotent(token, p)
	if it == nil {
		if it, err = st.appendLocked(p.chat, &cnpItem{typ: "MESSAGE", contentType: ct, content: cnpStr(req.in, "Content"), participant: p}); err != nil {
			return nil, err
		}
		if token != "" {
			p.chat.clientTokens[token] = it
		}
	}
	return map[string]any{"Id": it.id, "AbsoluteTime": cnpTime(it.at)}, nil
}

func (st *cnpStore) sendEvent(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	ct := cnpStr(req.in, "ContentType")
	typ, ok := cnpEventTypes[ct]
	if !ok {
		return nil, cnpValidation("ContentType %q isn't a participant event type", ct)
	}
	token := cnpStr(req.in, "ClientToken")
	if it := p.chat.idempotent(token, p); it != nil {
		return map[string]any{"Id": it.id, "AbsoluteTime": cnpTime(it.at)}, nil
	}
	content := cnpStr(req.in, "Content")
	var target *cnpItem
	if typ == "MESSAGE_DELIVERED" || typ == "MESSAGE_READ" {
		var receipt struct {
			MessageID string `json:"messageId"`
		}
		if json.Unmarshal([]byte(content), &receipt) != nil || receipt.MessageID == "" {
			return nil, cnpValidation(`a receipt's Content must be {"messageId":"..."}`)
		}
		for _, it := range p.chat.items {
			if it.id == receipt.MessageID && it.typ == "MESSAGE" {
				target = it
				break
			}
		}
		if target == nil {
			return nil, cnpValidation("message %s isn't in this chat", receipt.MessageID)
		}
		if target.participant == p {
			return nil, cnpConflict("a participant can't send receipts for its own message")
		}
	}
	it, err := st.appendLocked(p.chat, &cnpItem{typ: typ, contentType: ct, content: content, participant: p})
	if err != nil {
		return nil, err
	}
	if token != "" {
		p.chat.clientTokens[token] = it
	}
	if target != nil {
		if target.receipts == nil {
			target.receipts = map[string]*cnpReceipt{}
		}
		rc := target.receipts[p.id]
		if rc == nil {
			rc = &cnpReceipt{}
			target.receipts[p.id] = rc
		}
		if rc.delivered.IsZero() {
			rc.delivered = it.at
		}
		if typ == "MESSAGE_READ" {
			rc.read = it.at
		}
	}
	return map[string]any{"Id": it.id, "AbsoluteTime": cnpTime(it.at)}, nil
}

// getTranscript pages through the transcript. NextToken is the index the
// next page starts (FORWARD) or ends (BACKWARD) at.
func (st *cnpStore) getTranscript(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.byConnectionLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	chat := p.chat
	if id := cnpStr(req.in, "ContactId"); id != "" && id != chat.contactID {
		return nil, cnpValidation("contact %s isn't in this chat's contact chain", id)
	}
	n := len(chat.items)
	forward := cnpStr(req.in, "ScanDirection") == "FORWARD"
	limit := cnpInt(req.in, "MaxResults", 15)
	if limit == 0 {
		limit = 15
	}
	pos := -1
	if sp, _ := req.in["StartPosition"].(map[string]any); len(sp) > 0 {
		if len(sp) != 1 {
			return nil, cnpValidation("StartPosition takes exactly one of Id, AbsoluteTime and MostRecent")
		}
		var perr *cnpErr
		if pos, perr = chat.startPosition(sp); perr != nil {
			return nil, perr
		}
	}
	var lo, hi int
	if forward {
		lo = max(pos, 0)
		if tok := cnpStr(req.in, "NextToken"); tok != "" {
			v, err := strconv.ParseInt(tok, 10, 32)
			if err != nil || v < 0 || int(v) > n {
				return nil, cnpValidation("NextToken is not valid")
			}
			lo = int(v)
		}
		hi = min(lo+limit, n)
	} else {
		hi = n
		if pos >= 0 {
			hi = pos + 1
		}
		if tok := cnpStr(req.in, "NextToken"); tok != "" {
			v, err := strconv.ParseInt(tok, 10, 32)
			if err != nil || v < 0 || int(v) > n {
				return nil, cnpValidation("NextToken is not valid")
			}
			hi = int(v)
		}
		lo = max(hi-limit, 0)
	}
	items := make([]any, 0, max(hi-lo, 0))
	for i := lo; i < hi; i++ {
		items = append(items, chat.itemOut(chat.items[i]))
	}
	if cnpStr(req.in, "SortOrder") == "DESCENDING" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}
	out := map[string]any{"InitialContactId": chat.contactID, "Transcript": items}
	if forward && hi < n {
		out["NextToken"] = strconv.Itoa(hi)
	} else if !forward && lo > 0 {
		out["NextToken"] = strconv.Itoa(lo)
	}
	return out, nil
}

func (chat *cnpChat) startPosition(sp map[string]any) (int, *cnpErr) {
	n := len(chat.items)
	if id, ok := sp["Id"].(string); ok {
		for i, it := range chat.items {
			if it.id == id {
				return i, nil
			}
		}
		return 0, cnpValidation("item %s isn't in this transcript", id)
	}
	if at, ok := sp["AbsoluteTime"].(string); ok {
		t, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return 0, cnpValidation("StartPosition.AbsoluteTime must be an ISO 8601 time")
		}
		for i, it := range chat.items {
			if !it.at.Before(t) {
				return i, nil
			}
		}
		return n, nil
	}
	recent := int(sp["MostRecent"].(float64))
	return max(n-1-recent, 0), nil
}

func (st *cnpStore) getAuthenticationURL(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	redirect := cnpStr(req.in, "RedirectUri")
	u, perr := url.Parse(redirect)
	if perr != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, cnpValidation("RedirectUri must be an absolute http(s) URL")
	}
	session := cnpStr(req.in, "SessionId")
	auth := p.chat.auth[session]
	if auth == nil {
		if len(p.chat.auth) >= cnpMaxAuthSessions {
			return nil, cnpValidation("a chat holds at most %d authentication sessions", cnpMaxAuthSessions)
		}
		auth = &cnpAuthSession{}
		p.chat.auth[session] = auth
	}
	if auth.cancelled {
		return nil, cnpValidation("authentication session %s was cancelled", session)
	}
	auth.redirect = redirect
	q := url.Values{"response_type": {"code"}, "client_id": {"stackyard-" + p.chat.instanceID}, "redirect_uri": {redirect}, "state": {session}}
	return map[string]any{"AuthenticationUrl": "https://stackyard-" + p.chat.instanceID + ".auth.us-east-1.amazoncognito.com/oauth2/authorize?" + q.Encode()}, nil
}

func (st *cnpStore) cancelAuthentication(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	session := cnpStr(req.in, "SessionId")
	auth := p.chat.auth[session]
	if auth == nil || auth.cancelled {
		return nil, cnpValidation("authentication session %s isn't active", session)
	}
	auth.cancelled = true
	_, _ = st.appendLocked(p.chat, &cnpItem{typ: "EVENT", contentType: "application/vnd.amazonaws.connect.event.authentication.cancelled", participant: p})
	return nil, nil
}

// cnpViews are the AWS managed views a ShowView block renders; their token is
// the view ARN.
var cnpViews = map[string]map[string]any{}

func init() {
	for name, actions := range map[string][]string{
		"cards":        {"ActionSelected", "Back"},
		"confirmation": {"Next"},
		"detail":       {"ActionSelected", "Back"},
		"form":         {"ActionSelected", "Submit", "Cancel"},
		"list":         {"ActionSelected", "Back"},
	} {
		arn := "arn:aws:connect:us-east-1:aws:view/" + name
		acts := make([]any, len(actions))
		for i, a := range actions {
			acts[i] = a
		}
		cnpViews[arn] = map[string]any{
			"Id": arn, "Arn": arn, "Name": strings.ToUpper(name[:1]) + name[1:], "Version": 1,
			"Content": map[string]any{
				"InputSchema": `{"type":"object","properties":{}}`,
				"Template":    `{"Head":{"Title":"` + name + `"},"Body":[]}`,
				"Actions":     acts,
			},
		}
	}
}

func (st *cnpStore) describeView(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	_, err := st.byConnectionLocked(req.bearer)
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	token, _ := url.PathUnescape(cnpStr(req.in, "ViewToken"))
	view := cnpViews[token]
	if view == nil {
		return nil, &cnpErr{http.StatusNotFound, "ResourceNotFoundException", "view not found"}
	}
	content := view["Content"].(map[string]any)
	return map[string]any{"View": map[string]any{
		"Id": view["Id"], "Arn": view["Arn"], "Name": view["Name"], "Version": view["Version"],
		"Content": map[string]any{"InputSchema": content["InputSchema"], "Template": content["Template"], "Actions": append([]any(nil), content["Actions"].([]any)...)},
	}}, nil
}
