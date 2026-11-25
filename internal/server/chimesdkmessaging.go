package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Chime SDK Messaging: REST JSON signed chime, served from
// messaging-chime.<region>.amazonaws.com. Channels belong to an app
// instance; every channel call acts as the app instance user or bot named
// by the x-amz-chime-bearer header, which must belong to the channel's app
// instance. Stackyard doesn't emulate Chime SDK Identity, so any well-formed
// user or bot ARN of the app instance is accepted as an identity; the
// channel permissions (moderators, members, bans, privacy, restricted
// mode, message senders) are enforced on it.

const (
	cmsgMaxChannels    = 2000
	cmsgMaxFlows       = 100
	cmsgMaxApps        = 1000
	cmsgMaxMembers     = 1000
	cmsgMaxMemberships = 50000
	cmsgMaxModerators  = 100
	cmsgMaxBans        = 1000
	cmsgMaxMessages    = 500
	cmsgMaxBytes       = 16 << 20
	cmsgMaxContent     = 4096
	cmsgMaxControl     = 30
	cmsgMaxAttributes  = 10
	cmsgMaxAttrValues  = 10
	cmsgMaxTags        = 50
	cmsgMaxSubChannels = 1000
)

var (
	cmsgAppRE  = regexp.MustCompile(`^arn:aws:chime:[a-z0-9-]{1,32}:[0-9]{12}:app-instance/[A-Za-z0-9-]{1,64}$`)
	cmsgUserRE = regexp.MustCompile(`^(arn:aws:chime:[a-z0-9-]{1,32}:[0-9]{12}:app-instance/[A-Za-z0-9-]{1,64})/(user|bot)/[A-Za-z0-9._@:-]{1,64}$`)
)

type cmsgMember struct {
	arn, typ, invitedBy, sub string
	created, updated, read   time.Time
	prefs                    map[string]any
}

type cmsgRole struct {
	createdBy string
	created   time.Time
}

type cmsgMessage struct {
	id, content, metadata, typ, persistence, sender, sub, contentType, status, detail, callbackID, tokenKey string
	attrs                                                                                                   map[string]any
	target                                                                                                  []any
	created, updated, edited                                                                                time.Time
	redacted                                                                                                bool
	seq                                                                                                     int64
	size                                                                                                    int
}

type cmsgChannel struct {
	arn, appArn, name, mode, privacy, metadata, flowArn, createdBy, tokenKey string
	elastic, expiration                                                      map[string]any
	created, updated, lastMessage                                            time.Time
	tags                                                                     map[string]string
	members                                                                  map[string]*cmsgMember
	moderators, bans                                                         map[string]*cmsgRole
	subChannels                                                              map[string]int
	subSeq                                                                   int
	messages                                                                 []*cmsgMessage
	byID                                                                     map[string]*cmsgMessage
	msgTokens                                                                map[string]string
	seq                                                                      int64
}

type cmsgFlow struct {
	arn, appArn, name, tokenKey string
	processors                  []any
	created, updated            time.Time
	tags                        map[string]string
	channels                    map[string]struct{}
}

type cmsgStore struct {
	mu            sync.Mutex
	channels      map[string]*cmsgChannel
	byApp         map[string]map[string]struct{}
	userChannels  map[string]map[string]struct{}
	userModerated map[string]map[string]struct{}
	tokens        map[string]string
	flows         map[string]*cmsgFlow
	streaming     map[string][]any
	memberships   int
	bytes         int
	now           func() time.Time
}

func newChimeSDKMessagingStore() *cmsgStore {
	return &cmsgStore{
		channels: map[string]*cmsgChannel{}, byApp: map[string]map[string]struct{}{},
		userChannels: map[string]map[string]struct{}{}, userModerated: map[string]map[string]struct{}{},
		tokens: map[string]string{}, flows: map[string]*cmsgFlow{}, streaming: map[string][]any{}, now: time.Now,
	}
}

type cmsgHandler func(st *cmsgStore, in map[string]any, bearer string) (map[string]any, *chimeSDKErr)

func (s *Server) handleChimeSDKMessagingRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, params, ok := chimeSDKMatch(r, "messaging-chime.", chimeSDKMessagingOperations)
	if !ok || (chimeSDKIsTagOp(op) && chimeSDKTagOwner(r) != "messaging") {
		return false
	}
	model := awsmodels.ChimeSDKMessagingModel()
	in, ok := s.chimeSDKDecode(w, r, model, op, params)
	if !ok {
		return true
	}
	bearer := r.Header.Get("X-Amz-Chime-Bearer")
	if len(bearer) > 1600 {
		chimeSDKWriteError(w, chimeSDKBadRequest("x-amz-chime-bearer is too long."))
		return true
	}
	st := s.chimeSDKMessaging
	st.mu.Lock()
	out, cerr := cmsgHandlers[op.Name](st, in, bearer)
	var body []byte
	if cerr == nil && op.Code != http.StatusNoContent {
		if out == nil {
			out = map[string]any{}
		}
		body, _ = json.Marshal(model.Conform(model.Operations[op.Name].Output, out))
	}
	st.mu.Unlock()
	if cerr != nil {
		chimeSDKWriteError(w, cerr)
		return true
	}
	status := op.Code
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	return true
}

// cmsgHandlers run with the store lock held.
var cmsgHandlers = map[string]cmsgHandler{
	"CreateChannel":                               (*cmsgStore).createChannel,
	"DescribeChannel":                             (*cmsgStore).describeChannel,
	"UpdateChannel":                               (*cmsgStore).updateChannel,
	"DeleteChannel":                               (*cmsgStore).deleteChannel,
	"ListChannels":                                (*cmsgStore).listChannels,
	"SearchChannels":                              (*cmsgStore).searchChannels,
	"PutChannelExpirationSettings":                (*cmsgStore).putExpiration,
	"ListSubChannels":                             (*cmsgStore).listSubChannels,
	"ListChannelMembershipsForAppInstanceUser":    (*cmsgStore).listMembershipsForUser,
	"DescribeChannelMembershipForAppInstanceUser": (*cmsgStore).describeMembershipForUser,
	"ListChannelsModeratedByAppInstanceUser":      (*cmsgStore).listModeratedByUser,
	"DescribeChannelModeratedByAppInstanceUser":   (*cmsgStore).describeModeratedByUser,
	"CreateChannelMembership":                     (*cmsgStore).createMembership,
	"BatchCreateChannelMembership":                (*cmsgStore).batchCreateMembership,
	"DescribeChannelMembership":                   (*cmsgStore).describeMembership,
	"DeleteChannelMembership":                     (*cmsgStore).deleteMembership,
	"ListChannelMemberships":                      (*cmsgStore).listMemberships,
	"GetChannelMembershipPreferences":             (*cmsgStore).getPreferences,
	"PutChannelMembershipPreferences":             (*cmsgStore).putPreferences,
	"CreateChannelModerator":                      (*cmsgStore).createModerator,
	"DescribeChannelModerator":                    (*cmsgStore).describeModerator,
	"DeleteChannelModerator":                      (*cmsgStore).deleteModerator,
	"ListChannelModerators":                       (*cmsgStore).listModerators,
	"CreateChannelBan":                            (*cmsgStore).createBan,
	"DescribeChannelBan":                          (*cmsgStore).describeBan,
	"DeleteChannelBan":                            (*cmsgStore).deleteBan,
	"ListChannelBans":                             (*cmsgStore).listBans,
	"SendChannelMessage":                          (*cmsgStore).sendMessage,
	"GetChannelMessage":                           (*cmsgStore).getMessage,
	"GetChannelMessageStatus":                     (*cmsgStore).getMessageStatus,
	"ListChannelMessages":                         (*cmsgStore).listMessages,
	"UpdateChannelMessage":                        (*cmsgStore).updateMessage,
	"DeleteChannelMessage":                        (*cmsgStore).deleteMessage,
	"RedactChannelMessage":                        (*cmsgStore).redactMessage,
	"UpdateChannelReadMarker":                     (*cmsgStore).updateReadMarker,
	"CreateChannelFlow":                           (*cmsgStore).createFlow,
	"DescribeChannelFlow":                         (*cmsgStore).describeFlow,
	"UpdateChannelFlow":                           (*cmsgStore).updateFlow,
	"DeleteChannelFlow":                           (*cmsgStore).deleteFlow,
	"ListChannelFlows":                            (*cmsgStore).listFlows,
	"AssociateChannelFlow":                        (*cmsgStore).associateFlow,
	"DisassociateChannelFlow":                     (*cmsgStore).disassociateFlow,
	"ListChannelsAssociatedWithChannelFlow":       (*cmsgStore).listFlowChannels,
	"ChannelFlowCallback":                         (*cmsgStore).flowCallback,
	"PutMessagingStreamingConfigurations":         (*cmsgStore).putStreaming,
	"GetMessagingStreamingConfigurations":         (*cmsgStore).getStreaming,
	"DeleteMessagingStreamingConfigurations":      (*cmsgStore).deleteStreaming,
	"GetMessagingSessionEndpoint":                 (*cmsgStore).sessionEndpoint,
	"TagResource":                                 (*cmsgStore).tagResource,
	"UntagResource":                               (*cmsgStore).untagResource,
	"ListTagsForResource":                         (*cmsgStore).listTags,
}

// ---- identities and permissions ----

// cmsgUserApp returns the app instance ARN of a user or bot ARN, or "".
func cmsgUserApp(arn string) string {
	if m := cmsgUserRE.FindStringSubmatch(arn); m != nil {
		return m[1]
	}
	return ""
}

// identity checks that bearer is a user or bot of app.
func cmsgIdentity(bearer, app string) *chimeSDKErr {
	if bearer == "" {
		return chimeSDKBadRequest("The x-amz-chime-bearer header is required.")
	}
	if cmsgUserApp(bearer) == "" {
		return chimeSDKBadRequest("x-amz-chime-bearer %s isn't an app instance user or bot ARN.", bearer)
	}
	if cmsgUserApp(bearer) != app {
		return chimeSDKForbidden("%s isn't a user of app instance %s.", bearer, app)
	}
	return nil
}

func cmsgMemberARN(arn, app string) *chimeSDKErr {
	if cmsgUserApp(arn) != app {
		return chimeSDKBadRequest("%s isn't a user or bot of app instance %s.", arn, app)
	}
	return nil
}

func (ch *cmsgChannel) isModerator(who string) bool { return ch.moderators[who] != nil }

func (ch *cmsgChannel) canRead(who string) bool {
	return ch.privacy == "PUBLIC" || ch.members[who] != nil || ch.moderators[who] != nil
}

func (ch *cmsgChannel) expired(now time.Time) bool {
	days := mmgrInt(ch.expiration, "ExpirationDays", 0)
	if days <= 0 {
		return false
	}
	base := ch.created
	if mmgrStr(ch.expiration, "ExpirationCriterion") == "LAST_MESSAGE_TIMESTAMP" && !ch.lastMessage.IsZero() {
		base = ch.lastMessage
	}
	return now.After(base.Add(time.Duration(days) * 24 * time.Hour))
}

// channel returns a live channel (expired channels are deleted on access)
// after checking bearer belongs to its app instance.
func (st *cmsgStore) channel(arn, bearer string) (*cmsgChannel, *chimeSDKErr) {
	ch := st.channels[arn]
	if ch != nil && ch.expired(st.now()) {
		st.removeChannel(ch)
		ch = nil
	}
	if ch == nil {
		return nil, chimeSDKNotFound("The channel %s doesn't exist.", arn)
	}
	if cerr := cmsgIdentity(bearer, ch.appArn); cerr != nil {
		return nil, cerr
	}
	return ch, nil
}

// readable returns the channel when bearer may read it.
func (st *cmsgStore) readable(in map[string]any, bearer string) (*cmsgChannel, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	if !ch.canRead(bearer) {
		return nil, chimeSDKForbidden("%s isn't a member of private channel %s.", bearer, ch.arn)
	}
	return ch, nil
}

// moderated returns the channel when bearer moderates it.
func (st *cmsgStore) moderated(in map[string]any, bearer string) (*cmsgChannel, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	if !ch.isModerator(bearer) {
		return nil, chimeSDKForbidden("%s isn't a moderator of channel %s.", bearer, ch.arn)
	}
	return ch, nil
}

func (st *cmsgStore) removeChannel(ch *cmsgChannel) {
	for who := range ch.members {
		chimeSDKDrop(st.userChannels, who, ch.arn)
	}
	for who := range ch.moderators {
		chimeSDKDrop(st.userModerated, who, ch.arn)
	}
	st.memberships -= len(ch.members)
	for _, m := range ch.messages {
		st.bytes -= m.size
	}
	if f := st.flows[ch.flowArn]; f != nil {
		delete(f.channels, ch.arn)
	}
	chimeSDKDrop(st.byApp, ch.appArn, ch.arn)
	delete(st.tokens, ch.tokenKey)
	delete(st.channels, ch.arn)
}

// ---- channels ----

func cmsgIdentityView(arn string) map[string]any { return map[string]any{"Arn": arn} }

func (ch *cmsgChannel) summary() map[string]any {
	out := map[string]any{"Name": ch.name, "ChannelArn": ch.arn, "Mode": ch.mode, "Privacy": ch.privacy}
	if ch.metadata != "" {
		out["Metadata"] = ch.metadata
	}
	if !ch.lastMessage.IsZero() {
		out["LastMessageTimestamp"] = chimeSDKTime(ch.lastMessage)
	}
	return out
}

func (ch *cmsgChannel) view() map[string]any {
	out := ch.summary()
	out["CreatedBy"] = cmsgIdentityView(ch.createdBy)
	out["CreatedTimestamp"] = chimeSDKTime(ch.created)
	out["LastUpdatedTimestamp"] = chimeSDKTime(ch.updated)
	if ch.flowArn != "" {
		out["ChannelFlowArn"] = ch.flowArn
	}
	if ch.elastic != nil {
		out["ElasticChannelConfiguration"] = chimeSDKCopy(ch.elastic)
	}
	if ch.expiration != nil {
		out["ExpirationSettings"] = chimeSDKCopy(ch.expiration)
	}
	return out
}

func (st *cmsgStore) createChannel(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	if !cmsgAppRE.MatchString(app) {
		return nil, chimeSDKBadRequest("AppInstanceArn %s isn't an app instance ARN.", app)
	}
	if cerr := cmsgIdentity(bearer, app); cerr != nil {
		return nil, cerr
	}
	tokenKey := app + "|" + bearer + "|" + mmgrStr(in, "ClientRequestToken")
	if arn, ok := st.tokens[tokenKey]; ok && st.channels[arn] != nil {
		return map[string]any{"ChannelArn": arn}, nil
	}
	id := mmgrStr(in, "ChannelId")
	if id == "" {
		id = uuid.NewString()
	}
	arn := app + "/channel/" + id
	if st.channels[arn] != nil {
		return nil, chimeSDKConflict("The channel %s already exists.", arn)
	}
	if len(st.channels) >= cmsgMaxChannels {
		return nil, chimeSDKLimit("The account can have at most %d channels.", cmsgMaxChannels)
	}
	elastic := mmgrMap(in, "ElasticChannelConfiguration")
	if elastic != nil && (mmgrInt(elastic, "MaximumSubChannels", 0) > cmsgMaxSubChannels || mmgrInt(elastic, "TargetMembershipsPerSubChannel", 0) > cmsgMaxMembers) {
		return nil, chimeSDKBadRequest("ElasticChannelConfiguration allows at most %d sub-channels of at most %d members.", cmsgMaxSubChannels, cmsgMaxMembers)
	}
	members, moderators := mmgrList(in, "MemberArns"), mmgrList(in, "ModeratorArns")
	for _, list := range [][]any{members, moderators} {
		for _, m := range list {
			if cerr := cmsgMemberARN(mmgrStrOf(m), app); cerr != nil {
				return nil, cerr
			}
		}
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmsgMaxTags, cmsgTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	now := st.now()
	ch := &cmsgChannel{
		arn: arn, appArn: app, name: mmgrStr(in, "Name"), mode: cmsgDefault(mmgrStr(in, "Mode"), "UNRESTRICTED"),
		privacy: cmsgDefault(mmgrStr(in, "Privacy"), "PUBLIC"), metadata: mmgrStr(in, "Metadata"), createdBy: bearer,
		tokenKey: tokenKey, elastic: elastic, expiration: mmgrMap(in, "ExpirationSettings"), created: now, updated: now,
		tags: tags, members: map[string]*cmsgMember{}, moderators: map[string]*cmsgRole{}, bans: map[string]*cmsgRole{},
		subChannels: map[string]int{}, byID: map[string]*cmsgMessage{}, msgTokens: map[string]string{},
	}
	st.channels[arn] = ch
	st.tokens[tokenKey] = arn
	chimeSDKPut(st.byApp, app, arn)
	// The creator moderates the channel; in a standard channel it's also a member.
	for _, who := range append([]any{bearer}, moderators...) {
		st.addModerator(ch, mmgrStrOf(who), bearer)
	}
	if elastic == nil {
		members = append([]any{bearer}, members...)
	}
	for _, who := range members {
		if _, cerr := st.addMember(ch, mmgrStrOf(who), "DEFAULT", bearer, ""); cerr != nil {
			st.removeChannel(ch)
			return nil, cerr
		}
	}
	return map[string]any{"ChannelArn": arn}, nil
}

func mmgrStrOf(v any) string { s, _ := v.(string); return s }

func cmsgDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func (st *cmsgStore) describeChannel(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Channel": ch.view()}, nil
}

func (st *cmsgStore) updateChannel(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	if v := mmgrStr(in, "Name"); v != "" {
		ch.name = v
	}
	if v := mmgrStr(in, "Mode"); v != "" {
		ch.mode = v
	}
	if v, ok := in["Metadata"].(string); ok {
		ch.metadata = v
	}
	ch.updated = st.now()
	return map[string]any{"ChannelArn": ch.arn}, nil
}

func (st *cmsgStore) deleteChannel(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	st.removeChannel(ch)
	return nil, nil
}

// pageChannels pages channel ARNs and shapes each with view.
func (st *cmsgStore) pageChannels(arns []string, in map[string]any, key string, view func(*cmsgChannel) map[string]any) map[string]any {
	page, next := chimeSDKPage(arns, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 50)
	out := make([]any, 0, len(page))
	now := st.now()
	for _, arn := range page {
		if ch := st.channels[arn]; ch != nil && !ch.expired(now) {
			out = append(out, view(ch))
		}
	}
	resp := map[string]any{key: out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp
}

func (st *cmsgStore) listChannels(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	app := mmgrStr(in, "AppInstanceArn")
	if cerr := cmsgIdentity(bearer, app); cerr != nil {
		return nil, cerr
	}
	privacy := cmsgDefault(mmgrStr(in, "Privacy"), "PUBLIC")
	var arns []string
	for arn := range st.byApp[app] {
		ch := st.channels[arn]
		// Private channels are listed only to their members and moderators.
		if ch.privacy == privacy && (privacy == "PUBLIC" || ch.members[bearer] != nil || ch.moderators[bearer] != nil) {
			arns = append(arns, arn)
		}
	}
	return st.pageChannels(arns, in, "Channels", (*cmsgChannel).summary), nil
}

// selfUser returns the AppInstanceUserArn a per-user call names (the bearer
// by default); users may only ask about themselves.
func cmsgSelfUser(in map[string]any, bearer string) (string, *chimeSDKErr) {
	user := cmsgDefault(mmgrStr(in, "AppInstanceUserArn"), bearer)
	if cerr := cmsgIdentity(bearer, cmsgUserApp(bearer)); cerr != nil {
		return "", cerr
	}
	if user != bearer {
		return "", chimeSDKForbidden("%s can only list its own channels.", bearer)
	}
	return user, nil
}

func (st *cmsgStore) listMembershipsForUser(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	user, cerr := cmsgSelfUser(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return st.pageChannels(chimeSDKKeys(st.userChannels[user]), in, "ChannelMemberships", func(ch *cmsgChannel) map[string]any {
		return ch.membershipSummary(user)
	}), nil
}

func (ch *cmsgChannel) membershipSummary(user string) map[string]any {
	m := ch.members[user]
	ms := map[string]any{"Type": m.typ}
	if !m.read.IsZero() {
		ms["ReadMarkerTimestamp"] = chimeSDKTime(m.read)
	}
	if m.sub != "" {
		ms["SubChannelId"] = m.sub
	}
	return map[string]any{"ChannelSummary": ch.summary(), "AppInstanceUserMembershipSummary": ms}
}

func (st *cmsgStore) describeMembershipForUser(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	user, cerr := cmsgSelfUser(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	if ch.members[user] == nil {
		return nil, chimeSDKNotFound("%s isn't a member of %s.", user, ch.arn)
	}
	return map[string]any{"ChannelMembership": ch.membershipSummary(user)}, nil
}

func (st *cmsgStore) listModeratedByUser(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	user, cerr := cmsgSelfUser(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return st.pageChannels(chimeSDKKeys(st.userModerated[user]), in, "Channels", func(ch *cmsgChannel) map[string]any {
		return map[string]any{"ChannelSummary": ch.summary()}
	}), nil
}

func (st *cmsgStore) describeModeratedByUser(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	user, cerr := cmsgSelfUser(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	if !ch.isModerator(user) {
		return nil, chimeSDKNotFound("%s doesn't moderate %s.", user, ch.arn)
	}
	return map[string]any{"Channel": map[string]any{"ChannelSummary": ch.summary()}}, nil
}

// searchChannels finds channels by members. Candidates come from the
// membership index of the first value searched for, so the work is bounded
// by that user's memberships.
func (st *cmsgStore) searchChannels(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	if bearer != "" {
		if cerr := cmsgIdentity(bearer, cmsgUserApp(bearer)); cerr != nil {
			return nil, cerr
		}
	}
	fields := mmgrList(in, "Fields")
	var first []any
	if len(fields) > 0 {
		first = mmgrList(chimeSDKMapOf(fields[0]), "Values")
	}
	if len(first) == 0 {
		return nil, chimeSDKBadRequest("Fields need at least one value.")
	}
	var arns []string
	for arn := range st.userChannels[mmgrStrOf(first[0])] {
		ch := st.channels[arn]
		if (bearer == "" || ch.canRead(bearer)) && ch.matchesSearch(fields) {
			arns = append(arns, arn)
		}
	}
	return st.pageChannels(arns, in, "Channels", (*cmsgChannel).summary), nil
}

func (ch *cmsgChannel) matchesSearch(fields []any) bool {
	for _, raw := range fields {
		f := chimeSDKMapOf(raw)
		values := mmgrList(f, "Values")
		for _, v := range values {
			if ch.members[mmgrStrOf(v)] == nil {
				return false
			}
		}
		if mmgrStr(f, "Operator") == "EQUALS" && len(ch.members) != len(values) {
			return false
		}
	}
	return true
}

func (st *cmsgStore) putExpiration(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	arn := mmgrStr(in, "ChannelArn")
	var ch *cmsgChannel
	var cerr *chimeSDKErr
	if bearer == "" {
		// An IAM caller without a bearer acts as the app instance admin.
		if ch = st.channels[arn]; ch == nil || ch.expired(st.now()) {
			return nil, chimeSDKNotFound("The channel %s doesn't exist.", arn)
		}
	} else if ch, cerr = st.moderated(in, bearer); cerr != nil {
		return nil, cerr
	}
	ch.expiration = mmgrMap(in, "ExpirationSettings")
	ch.updated = st.now()
	out := map[string]any{"ChannelArn": ch.arn}
	if ch.expiration != nil {
		out["ExpirationSettings"] = chimeSDKCopy(ch.expiration)
	}
	return out, nil
}

func (st *cmsgStore) listSubChannels(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	if ch.elastic == nil {
		return nil, chimeSDKBadRequest("Channel %s isn't an elastic channel.", ch.arn)
	}
	ids := make([]string, 0, len(ch.subChannels))
	for id := range ch.subChannels {
		ids = append(ids, id)
	}
	page, next := chimeSDKPage(ids, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 50)
	out := make([]any, 0, len(page))
	for _, id := range page {
		out = append(out, map[string]any{"SubChannelId": id, "MembershipCount": ch.subChannels[id]})
	}
	resp := map[string]any{"ChannelArn": ch.arn, "SubChannels": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// ---- tags ----

func cmsgTooManyTags() *chimeSDKErr {
	return chimeSDKLimit("A resource can have at most %d tags.", cmsgMaxTags)
}

func (st *cmsgStore) tagged(in map[string]any) (*map[string]string, *chimeSDKErr) {
	arn := mmgrStr(in, "ResourceARN")
	if ch := st.channels[arn]; ch != nil {
		return &ch.tags, nil
	}
	if f := st.flows[arn]; f != nil {
		return &f.tags, nil
	}
	return nil, chimeSDKNotFound("The resource %s doesn't exist.", arn)
}

func (st *cmsgStore) tagResource(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	merged, cerr := chimeSDKMergeTags(*tags, mmgrList(in, "Tags"), cmsgMaxTags, cmsgTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	*tags = merged
	return nil, nil
}

func (st *cmsgStore) untagResource(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	*tags = chimeSDKUntag(*tags, mmgrList(in, "TagKeys"))
	return nil, nil
}

func (st *cmsgStore) listTags(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Tags": chimeSDKTagList(*tags)}, nil
}

func (st *cmsgStore) sessionEndpoint(in map[string]any, _ string) (map[string]any, *chimeSDKErr) {
	host := "stackyard.ucws." + chimeSDKRegion + ".amazonaws.com"
	if mmgrStr(in, "NetworkType") == "DUAL_STACK" {
		host = "stackyard.ucws." + chimeSDKRegion + ".api.aws"
	}
	return map[string]any{"Endpoint": map[string]any{"Url": host}}, nil
}
