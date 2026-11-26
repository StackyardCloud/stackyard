package server

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Channel messages, channel flows and streaming configurations.

// cmsgMessageSize counts the stored bytes of a message's caller content.
func cmsgMessageSize(content, metadata, contentType string, attrs map[string]any) int {
	raw, _ := json.Marshal(attrs)
	return len(content) + len(metadata) + len(contentType) + len(raw)
}

// cmsgCheckContent validates a message body as sent or edited.
func cmsgCheckContent(typ, content, metadata string, attrs map[string]any) *chimeSDKErr {
	switch {
	case len(content) > cmsgMaxContent:
		return chimeSDKBadRequest("Message content can be at most %d bytes.", cmsgMaxContent)
	case typ == "CONTROL" && (len(content) > cmsgMaxControl || metadata != ""):
		return chimeSDKBadRequest("CONTROL messages carry at most %d bytes of content and no metadata.", cmsgMaxControl)
	case len(attrs) > cmsgMaxAttributes:
		return chimeSDKBadRequest("A message can have at most %d attributes.", cmsgMaxAttributes)
	}
	for _, v := range attrs {
		if len(mmgrList(chimeSDKMapOf(v), "StringValues")) > cmsgMaxAttrValues {
			return chimeSDKBadRequest("A message attribute can have at most %d values.", cmsgMaxAttrValues)
		}
	}
	return nil
}

// reserve checks that the store can hold delta more message bytes.
func (st *cmsgStore) reserve(delta int) *chimeSDKErr {
	if st.bytes+delta > cmsgMaxBytes {
		return chimeSDKLimit("Stored channel messages are limited to %d bytes.", cmsgMaxBytes)
	}
	return nil
}

func (m *cmsgMessage) visibleTo(ch *cmsgChannel, who string) bool {
	if len(m.target) == 0 || who == m.sender || ch.isModerator(who) {
		return true
	}
	return mmgrStr(chimeSDKMapOf(m.target[0]), "MemberArn") == who
}

func (m *cmsgMessage) statusView() map[string]any {
	out := map[string]any{"Value": m.status}
	if m.detail != "" {
		out["Detail"] = m.detail
	}
	return out
}

func (m *cmsgMessage) view(ch *cmsgChannel) map[string]any {
	out := map[string]any{
		"ChannelArn": ch.arn, "MessageId": m.id, "Type": m.typ, "Persistence": m.persistence, "Redacted": m.redacted,
		"CreatedTimestamp": chimeSDKTime(m.created), "LastUpdatedTimestamp": chimeSDKTime(m.updated),
		"Sender": cmsgIdentityView(m.sender), "Status": m.statusView(),
	}
	if !m.redacted {
		out["Content"] = m.content
		if m.metadata != "" {
			out["Metadata"] = m.metadata
		}
		if m.attrs != nil {
			out["MessageAttributes"] = chimeSDKCopy(m.attrs)
		}
		if m.contentType != "" {
			out["ContentType"] = m.contentType
		}
	}
	if !m.edited.IsZero() {
		out["LastEditedTimestamp"] = chimeSDKTime(m.edited)
	}
	if m.sub != "" {
		out["SubChannelId"] = m.sub
	}
	if len(m.target) > 0 {
		out["Target"] = m.target
	}
	return out
}

// sub resolves the SubChannelId of a message call: required for elastic
// channels, refused for standard ones.
func (ch *cmsgChannel) sub(in map[string]any) (string, *chimeSDKErr) {
	sub := mmgrStr(in, "SubChannelId")
	switch {
	case ch.elastic == nil && sub != "":
		return "", chimeSDKBadRequest("SubChannelId applies only to elastic channels.")
	case ch.elastic != nil && sub == "":
		return "", chimeSDKBadRequest("Messages in elastic channel %s need a SubChannelId.", ch.arn)
	case ch.elastic != nil:
		if _, ok := ch.subChannels[sub]; !ok {
			return "", chimeSDKNotFound("Sub-channel %s doesn't exist.", sub)
		}
	}
	return sub, nil
}

func (st *cmsgStore) sendMessage(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	member := ch.members[bearer]
	if member == nil && !ch.isModerator(bearer) {
		return nil, chimeSDKForbidden("%s isn't a member of channel %s.", bearer, ch.arn)
	}
	sub, cerr := ch.sub(in)
	if cerr != nil {
		return nil, cerr
	}
	if sub != "" && !ch.isModerator(bearer) && member.sub != sub {
		return nil, chimeSDKForbidden("%s isn't a member of sub-channel %s.", bearer, sub)
	}
	typ, content, metadata, contentType := mmgrStr(in, "Type"), mmgrStr(in, "Content"), mmgrStr(in, "Metadata"), mmgrStr(in, "ContentType")
	attrs := mmgrMap(in, "MessageAttributes")
	if cerr := cmsgCheckContent(typ, content, metadata, attrs); cerr != nil {
		return nil, cerr
	}
	target := mmgrList(in, "Target")
	if len(target) > 0 && ch.members[mmgrStr(chimeSDKMapOf(target[0]), "MemberArn")] == nil {
		return nil, chimeSDKBadRequest("A targeted message must name a channel member.")
	}
	tokenKey := bearer + "|" + mmgrStr(in, "ClientRequestToken")
	if m := ch.byID[ch.msgTokens[tokenKey]]; m != nil {
		return cmsgSendResult(ch, m), nil
	}
	size := cmsgMessageSize(content, metadata, contentType, attrs)
	if cerr := st.reserve(size); cerr != nil {
		return nil, cerr
	}
	now := st.now()
	m := &cmsgMessage{
		id: strings.ReplaceAll(uuid.NewString(), "-", ""), content: content, metadata: metadata, typ: typ,
		persistence: mmgrStr(in, "Persistence"), sender: bearer, sub: sub, contentType: contentType, status: "SENT",
		attrs: attrs, target: target, created: now, updated: now, size: size, tokenKey: tokenKey,
	}
	ch.lastMessage = now
	// Channel flows hold messages PENDING until their processor calls back.
	if ch.flowArn != "" {
		m.status = "PENDING"
	} else if m.persistence == "NON_PERSISTENT" {
		return cmsgSendResult(ch, m), nil
	}
	ch.seq++
	m.seq = ch.seq
	ch.messages = append(ch.messages, m)
	ch.byID[m.id] = m
	ch.msgTokens[tokenKey] = m.id
	st.bytes += size
	for len(ch.messages) > cmsgMaxMessages {
		st.dropMessage(ch, 0)
	}
	return cmsgSendResult(ch, m), nil
}

func cmsgSendResult(ch *cmsgChannel, m *cmsgMessage) map[string]any {
	out := map[string]any{"ChannelArn": ch.arn, "MessageId": m.id, "Status": m.statusView()}
	if m.sub != "" {
		out["SubChannelId"] = m.sub
	}
	return out
}

func (st *cmsgStore) dropMessage(ch *cmsgChannel, i int) {
	m := ch.messages[i]
	ch.messages = append(ch.messages[:i:i], ch.messages[i+1:]...)
	delete(ch.byID, m.id)
	delete(ch.msgTokens, m.tokenKey)
	st.bytes -= m.size
}

// message returns the message a call names, checking its sub-channel.
func (ch *cmsgChannel) message(in map[string]any) (*cmsgMessage, *chimeSDKErr) {
	sub, cerr := ch.sub(in)
	if cerr != nil {
		return nil, cerr
	}
	m := ch.byID[mmgrStr(in, "MessageId")]
	if m == nil || m.sub != sub {
		return nil, chimeSDKNotFound("The message %s doesn't exist.", mmgrStr(in, "MessageId"))
	}
	return m, nil
}

func (st *cmsgStore) visibleMessage(in map[string]any, bearer string) (*cmsgChannel, *cmsgMessage, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, nil, cerr
	}
	m, cerr := ch.message(in)
	if cerr != nil {
		return nil, nil, cerr
	}
	if !m.visibleTo(ch, bearer) {
		return nil, nil, chimeSDKNotFound("The message %s doesn't exist.", m.id)
	}
	return ch, m, nil
}

func (st *cmsgStore) getMessage(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, m, cerr := st.visibleMessage(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"ChannelMessage": m.view(ch)}, nil
}

func (st *cmsgStore) getMessageStatus(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	_, m, cerr := st.visibleMessage(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Status": m.statusView()}, nil
}

// listMessages pages a channel's messages (newest first by default). The
// next token is the sequence number of the last message returned.
func (st *cmsgStore) listMessages(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	sub, cerr := ch.sub(in)
	if cerr != nil {
		return nil, cerr
	}
	var after int64 = -1
	if tok := mmgrStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 64)
		if err != nil || n < 0 {
			return nil, chimeSDKBadRequest("NextToken is invalid.")
		}
		after = n
	}
	notBefore, hasBefore := in["NotBefore"].(float64)
	notAfter, hasAfter := in["NotAfter"].(float64)
	asc := mmgrStr(in, "SortOrder") == "ASCENDING"
	limit := mmgrInt(in, "MaxResults", 50)
	out, next, last := []any{}, "", int64(0)
	for i := range ch.messages {
		m := ch.messages[i]
		if !asc {
			m = ch.messages[len(ch.messages)-1-i]
		}
		ts := chimeSDKTime(m.created)
		if after >= 0 && ((asc && m.seq <= after) || (!asc && m.seq >= after)) ||
			m.sub != sub || !m.visibleTo(ch, bearer) || (hasBefore && ts < notBefore) || (hasAfter && ts > notAfter) {
			continue
		}
		if len(out) == limit {
			next = strconv.FormatInt(last, 10)
			break
		}
		out = append(out, m.view(ch))
		last = m.seq
	}
	resp := map[string]any{"ChannelArn": ch.arn, "ChannelMessages": out}
	if next != "" {
		resp["NextToken"] = next
	}
	if sub != "" {
		resp["SubChannelId"] = sub
	}
	return resp, nil
}

func (st *cmsgStore) updateMessage(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	m, cerr := ch.message(in)
	if cerr != nil {
		return nil, cerr
	}
	if m.sender != bearer {
		return nil, chimeSDKForbidden("Only the sender can edit message %s.", m.id)
	}
	if m.redacted {
		return nil, chimeSDKBadRequest("Message %s was redacted.", m.id)
	}
	content, metadata, contentType := mmgrStr(in, "Content"), mmgrStr(in, "Metadata"), mmgrStr(in, "ContentType")
	if cerr := cmsgCheckContent(m.typ, content, metadata, m.attrs); cerr != nil {
		return nil, cerr
	}
	size := cmsgMessageSize(content, metadata, contentType, m.attrs)
	if cerr := st.reserve(size - m.size); cerr != nil {
		return nil, cerr
	}
	st.bytes += size - m.size
	now := st.now()
	m.content, m.metadata, m.contentType, m.size, m.edited, m.updated = content, metadata, contentType, size, now, now
	if ch.flowArn != "" {
		m.status, m.detail, m.callbackID = "PENDING", "", ""
	}
	return cmsgSendResult(ch, m), nil
}

func (st *cmsgStore) deleteMessage(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	m, cerr := ch.message(in)
	if cerr != nil {
		return nil, cerr
	}
	i := sort.Search(len(ch.messages), func(i int) bool { return ch.messages[i].seq >= m.seq })
	st.dropMessage(ch, i)
	return nil, nil
}

func (st *cmsgStore) redactMessage(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	m, cerr := ch.message(in)
	if cerr != nil {
		return nil, cerr
	}
	if m.sender != bearer && !ch.isModerator(bearer) {
		return nil, chimeSDKForbidden("Only the sender or a moderator can redact message %s.", m.id)
	}
	st.bytes -= m.size
	m.content, m.metadata, m.contentType, m.attrs, m.size, m.redacted, m.updated = "", "", "", nil, 0, true, st.now()
	out := map[string]any{"ChannelArn": ch.arn, "MessageId": m.id}
	if m.sub != "" {
		out["SubChannelId"] = m.sub
	}
	return out, nil
}

func (st *cmsgStore) updateReadMarker(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	m, cerr := st.member(ch, bearer)
	if cerr != nil {
		return nil, cerr
	}
	m.read = st.now()
	return map[string]any{"ChannelArn": ch.arn}, nil
}

// ---- channel flows ----

func cmsgCheckProcessors(processors []any) *chimeSDKErr {
	seen := map[int]bool{}
	for _, raw := range processors {
		p := chimeSDKMapOf(raw)
		order := mmgrInt(p, "ExecutionOrder", 0)
		if seen[order] {
			return chimeSDKBadRequest("Processor ExecutionOrder values must be unique.")
		}
		seen[order] = true
		if arn := mmgrStr(mmgrMap(mmgrMap(p, "Configuration"), "Lambda"), "ResourceArn"); !strings.HasPrefix(arn, "arn:aws:lambda:") {
			return chimeSDKBadRequest("Processor %s needs a Lambda function ARN.", mmgrStr(p, "Name"))
		}
	}
	return nil
}

func (f *cmsgFlow) view() map[string]any {
	return map[string]any{
		"ChannelFlowArn": f.arn, "Name": f.name, "Processors": f.processors,
		"CreatedTimestamp": chimeSDKTime(f.created), "LastUpdatedTimestamp": chimeSDKTime(f.updated),
	}
}

func (st *cmsgStore) flow(arn string) (*cmsgFlow, *chimeSDKErr) {
	f := st.flows[arn]
	if f == nil {
		return nil, chimeSDKNotFound("The channel flow %s doesn't exist.", arn)
	}
	return f, nil
}

func (st *cmsgStore) createFlow(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	if !cmsgAppRE.MatchString(app) {
		return nil, chimeSDKBadRequest("AppInstanceArn %s isn't an app instance ARN.", app)
	}
	tokenKey := app + "|flow|" + mmgrStr(in, "ClientRequestToken")
	if arn, ok := st.tokens[tokenKey]; ok && st.flows[arn] != nil {
		return map[string]any{"ChannelFlowArn": arn}, nil
	}
	processors := mmgrList(in, "Processors")
	if cerr := cmsgCheckProcessors(processors); cerr != nil {
		return nil, cerr
	}
	if len(st.flows) >= cmsgMaxFlows {
		return nil, chimeSDKLimit("The account can have at most %d channel flows.", cmsgMaxFlows)
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmsgMaxTags, cmsgTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	now := st.now()
	f := &cmsgFlow{
		arn: app + "/channel-flow/" + uuid.NewString(), appArn: app, name: mmgrStr(in, "Name"), tokenKey: tokenKey,
		processors: processors, created: now, updated: now, tags: tags, channels: map[string]struct{}{},
	}
	st.flows[f.arn] = f
	st.tokens[tokenKey] = f.arn
	return map[string]any{"ChannelFlowArn": f.arn}, nil
}

func (st *cmsgStore) describeFlow(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	f, cerr := st.flow(mmgrStr(in, "ChannelFlowArn"))
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"ChannelFlow": f.view()}, nil
}

func (st *cmsgStore) updateFlow(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	f, cerr := st.flow(mmgrStr(in, "ChannelFlowArn"))
	if cerr != nil {
		return nil, cerr
	}
	processors := mmgrList(in, "Processors")
	if cerr := cmsgCheckProcessors(processors); cerr != nil {
		return nil, cerr
	}
	f.name, f.processors, f.updated = mmgrStr(in, "Name"), processors, st.now()
	return map[string]any{"ChannelFlowArn": f.arn}, nil
}

func (st *cmsgStore) deleteFlow(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	f, cerr := st.flow(mmgrStr(in, "ChannelFlowArn"))
	if cerr != nil {
		return nil, cerr
	}
	if len(f.channels) > 0 {
		return nil, chimeSDKConflict("Channel flow %s is associated with %d channels.", f.arn, len(f.channels))
	}
	delete(st.flows, f.arn)
	delete(st.tokens, f.tokenKey)
	return nil, nil
}

func (st *cmsgStore) listFlows(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	var arns []string
	for arn, f := range st.flows {
		if f.appArn == app {
			arns = append(arns, arn)
		}
	}
	page, next := chimeSDKPage(arns, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 50)
	out := make([]any, 0, len(page))
	for _, arn := range page {
		f := st.flows[arn]
		out = append(out, map[string]any{"ChannelFlowArn": f.arn, "Name": f.name, "Processors": f.processors})
	}
	resp := map[string]any{"ChannelFlows": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (st *cmsgStore) associateFlow(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	f, cerr := st.flow(mmgrStr(in, "ChannelFlowArn"))
	if cerr != nil {
		return nil, cerr
	}
	switch {
	case f.appArn != ch.appArn:
		return nil, chimeSDKBadRequest("Channel flow %s belongs to another app instance.", f.arn)
	case ch.flowArn == f.arn:
		return nil, nil
	case ch.flowArn != "":
		return nil, chimeSDKConflict("Channel %s already has channel flow %s.", ch.arn, ch.flowArn)
	}
	ch.flowArn = f.arn
	f.channels[ch.arn] = struct{}{}
	return nil, nil
}

func (st *cmsgStore) disassociateFlow(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	arn := mmgrStr(in, "ChannelFlowArn")
	if ch.flowArn != arn {
		return nil, chimeSDKNotFound("Channel flow %s isn't associated with channel %s.", arn, ch.arn)
	}
	ch.flowArn = ""
	if f := st.flows[arn]; f != nil {
		delete(f.channels, ch.arn)
	}
	return nil, nil
}

func (st *cmsgStore) listFlowChannels(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	f, cerr := st.flow(mmgrStr(in, "ChannelFlowArn"))
	if cerr != nil {
		return nil, cerr
	}
	return st.pageChannels(chimeSDKKeys(f.channels), in, "Channels", func(ch *cmsgChannel) map[string]any {
		out := ch.summary()
		delete(out, "LastMessageTimestamp")
		return out
	}), nil
}

// flowCallback completes channel flow processing of a PENDING message:
// DeleteResource denies it, otherwise its processed content is applied and
// it becomes SENT. Repeating a CallbackId returns the same result.
func (st *cmsgStore) flowCallback(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	arn := mmgrStr(in, "ChannelArn")
	ch := st.channels[arn]
	if ch == nil || ch.expired(st.now()) {
		return nil, chimeSDKNotFound("The channel %s doesn't exist.", arn)
	}
	cm := mmgrMap(in, "ChannelMessage")
	m, cerr := ch.message(cm)
	if cerr != nil {
		return nil, cerr
	}
	cb := mmgrStr(in, "CallbackId")
	out := map[string]any{"ChannelArn": ch.arn, "CallbackId": cb}
	if m.callbackID == cb {
		return out, nil
	}
	if m.status != "PENDING" {
		return nil, chimeSDKConflict("Message %s isn't pending channel flow processing.", m.id)
	}
	if denied, _ := in["DeleteResource"].(bool); denied {
		m.status, m.detail, m.callbackID = "DENIED", "Denied by the channel flow processor.", cb
		return out, nil
	}
	content, metadata, contentType, attrs := m.content, m.metadata, m.contentType, m.attrs
	if v := mmgrStr(cm, "Content"); v != "" {
		content = v
	}
	if v, ok := cm["Metadata"].(string); ok {
		metadata = v
	}
	if v, ok := cm["ContentType"].(string); ok {
		contentType = v
	}
	if v := mmgrMap(cm, "MessageAttributes"); v != nil {
		attrs = v
	}
	if cerr := cmsgCheckContent(m.typ, content, metadata, attrs); cerr != nil {
		return nil, cerr
	}
	size := cmsgMessageSize(content, metadata, contentType, attrs)
	if cerr := st.reserve(size - m.size); cerr != nil {
		return nil, cerr
	}
	st.bytes += size - m.size
	m.content, m.metadata, m.contentType, m.attrs, m.size = content, metadata, contentType, attrs, size
	m.status, m.detail, m.callbackID, m.updated = "SENT", "", cb, st.now()
	return out, nil
}

// ---- streaming configurations ----

func (st *cmsgStore) putStreaming(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	if !cmsgAppRE.MatchString(app) {
		return nil, chimeSDKBadRequest("AppInstanceArn %s isn't an app instance ARN.", app)
	}
	configs := mmgrList(in, "StreamingConfigurations")
	seen := map[string]bool{}
	for _, raw := range configs {
		c := chimeSDKMapOf(raw)
		if seen[mmgrStr(c, "DataType")] {
			return nil, chimeSDKBadRequest("Each DataType can be streamed once.")
		}
		seen[mmgrStr(c, "DataType")] = true
		if !strings.HasPrefix(mmgrStr(c, "ResourceArn"), "arn:aws:kinesis:") {
			return nil, chimeSDKBadRequest("ResourceArn must be a Kinesis data stream ARN.")
		}
	}
	if _, ok := st.streaming[app]; !ok && len(st.streaming) >= cmsgMaxApps {
		return nil, chimeSDKLimit("At most %d app instances can stream.", cmsgMaxApps)
	}
	st.streaming[app] = configs
	return map[string]any{"StreamingConfigurations": configs}, nil
}

func (st *cmsgStore) getStreaming(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	configs, ok := st.streaming[app]
	if !ok {
		return nil, chimeSDKNotFound("App instance %s has no streaming configurations.", app)
	}
	return map[string]any{"StreamingConfigurations": configs}, nil
}

func (st *cmsgStore) deleteStreaming(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	delete(st.streaming, mmgrStr(in, "AppInstanceArn"))
	return nil, nil
}
