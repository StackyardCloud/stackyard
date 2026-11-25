package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Chime SDK Meetings: REST JSON signed chime, served from
// meetings-chime.<region>.amazonaws.com. Meetings, their attendees (join
// tokens are returned only when an attendee is created, and never stored),
// live transcription and tags.

const (
	cmtgMaxMeetings     = 1000
	cmtgMaxTags         = 50
	cmtgDefaultMaxCount = 250
)

var cmtgMediaRegions = []string{
	"af-south-1", "ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2",
	"ca-central-1", "eu-central-1", "eu-north-1", "eu-south-1", "eu-west-1", "eu-west-2", "eu-west-3", "il-central-1",
	"sa-east-1", "us-east-1", "us-east-2", "us-west-1", "us-west-2", "us-gov-east-1", "us-gov-west-1",
}

type cmtgAttendee struct {
	id, externalID string
	caps           map[string]any
}

type cmtgMeeting struct {
	id, arn, externalID, hostID, region, primaryID, token, network string
	features, notifications                                        map[string]any
	tenantIDs                                                      []any
	maxAttendees                                                   int
	tags                                                           map[string]string
	attendees                                                      map[string]*cmtgAttendee
	byExternal                                                     map[string]string
	transcription                                                  map[string]any
}

type cmtgStore struct {
	mu       sync.Mutex
	meetings map[string]*cmtgMeeting
	tokens   map[string]string
}

func newChimeSDKMeetingsStore() *cmtgStore {
	return &cmtgStore{meetings: map[string]*cmtgMeeting{}, tokens: map[string]string{}}
}

func (s *Server) handleChimeSDKMeetingsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, params, ok := chimeSDKMatch(r, "meetings-chime.", chimeSDKMeetingsOperations)
	if !ok || (chimeSDKIsTagOp(op) && chimeSDKTagOwner(r) != "meetings") {
		return false
	}
	model := awsmodels.ChimeSDKMeetingsModel()
	in, ok := s.chimeSDKDecode(w, r, model, op, params)
	if !ok {
		return true
	}
	st := s.chimeSDKMeetings
	body, cerr := func() ([]byte, *chimeSDKErr) {
		st.mu.Lock()
		defer st.mu.Unlock()
		out, cerr := cmtgHandlers[op.Name](st, in)
		if cerr != nil || op.Code == http.StatusNoContent {
			return nil, cerr
		}
		if out == nil {
			out = map[string]any{}
		}
		body, _ := json.Marshal(model.Conform(model.Operations[op.Name].Output, out))
		return body, nil
	}()
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

// cmtgHandlers run with the store lock held.
var cmtgHandlers = map[string]func(st *cmtgStore, in map[string]any) (map[string]any, *chimeSDKErr){
	"CreateMeeting":                         (*cmtgStore).createMeeting,
	"CreateMeetingWithAttendees":            (*cmtgStore).createMeetingWithAttendees,
	"GetMeeting":                            (*cmtgStore).getMeeting,
	"DeleteMeeting":                         (*cmtgStore).deleteMeeting,
	"CreateAttendee":                        (*cmtgStore).createAttendee,
	"BatchCreateAttendee":                   (*cmtgStore).batchCreateAttendee,
	"GetAttendee":                           (*cmtgStore).getAttendee,
	"ListAttendees":                         (*cmtgStore).listAttendees,
	"DeleteAttendee":                        (*cmtgStore).deleteAttendee,
	"UpdateAttendeeCapabilities":            (*cmtgStore).updateAttendeeCapabilities,
	"BatchUpdateAttendeeCapabilitiesExcept": (*cmtgStore).batchUpdateCapabilitiesExcept,
	"StartMeetingTranscription":             (*cmtgStore).startTranscription,
	"StopMeetingTranscription":              (*cmtgStore).stopTranscription,
	"TagResource":                           (*cmtgStore).tagResource,
	"UntagResource":                         (*cmtgStore).untagResource,
	"ListTagsForResource":                   (*cmtgStore).listTags,
}

// cmtgMeetingIDFromARN extracts the meeting ID from a meeting ARN, in
// either the meeting/<id> or the meeting:<id> (media pipelines) form.
func cmtgMeetingIDFromARN(arn string) string {
	if !strings.HasPrefix(arn, "arn:aws:chime:") {
		return ""
	}
	for _, sep := range []string{":meeting/", ":meeting:"} {
		if i := strings.LastIndex(arn, sep); i >= 0 {
			return arn[i+len(sep):]
		}
	}
	return ""
}

// meetingExists reports whether the meeting an ARN names exists. It takes
// the meetings lock; callers must not hold another store's lock.
func (st *cmtgStore) meetingExists(arn string) bool {
	id := cmtgMeetingIDFromARN(arn)
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.meetings[id] != nil
}

func (st *cmtgStore) meeting(in map[string]any) (*cmtgMeeting, *chimeSDKErr) {
	id := mmgrStr(in, "MeetingId")
	m := st.meetings[id]
	if m == nil {
		return nil, chimeSDKNotFound("The meeting %s doesn't exist.", id)
	}
	return m, nil
}

func (st *cmtgStore) createMeeting(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.newMeeting(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Meeting": m.view()}, nil
}

func (st *cmtgStore) newMeeting(in map[string]any) (*cmtgMeeting, *chimeSDKErr) {
	token := mmgrStr(in, "ClientRequestToken")
	if id, ok := st.tokens[token]; ok {
		return st.meetings[id], nil
	}
	region := mmgrStr(in, "MediaRegion")
	if !slices.Contains(cmtgMediaRegions, region) {
		return nil, chimeSDKBadRequest("MediaRegion %s is not supported.", region)
	}
	if primary := mmgrStr(in, "PrimaryMeetingId"); primary != "" && st.meetings[primary] == nil {
		return nil, chimeSDKNotFound("The primary meeting %s doesn't exist.", primary)
	}
	if n := mmgrMap(in, "NotificationsConfiguration"); n != nil && len(n) > 1 {
		return nil, chimeSDKBadRequest("NotificationsConfiguration takes one of LambdaFunctionArn, SnsTopicArn or SqsQueueArn.")
	}
	if len(st.meetings) >= cmtgMaxMeetings {
		return nil, chimeSDKError(http.StatusBadRequest, "LimitExceededException", "The account can have at most %d active meetings.", cmtgMaxMeetings)
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmtgMaxTags, cmtgTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	features := mmgrMap(in, "MeetingFeatures")
	id := uuid.NewString()
	m := &cmtgMeeting{
		id: id, arn: "arn:aws:chime:" + chimeSDKRegion + ":" + chimeSDKAccount + ":meeting/" + id,
		externalID: mmgrStr(in, "ExternalMeetingId"), hostID: mmgrStr(in, "MeetingHostId"), region: region,
		primaryID: mmgrStr(in, "PrimaryMeetingId"), token: token, network: mmgrStr(in, "MediaPlacementNetworkType"),
		features: features, notifications: mmgrMap(in, "NotificationsConfiguration"), tenantIDs: mmgrList(in, "TenantIds"),
		maxAttendees: mmgrInt(mmgrMap(features, "Attendee"), "MaxCount", cmtgDefaultMaxCount),
		tags:         tags, attendees: map[string]*cmtgAttendee{}, byExternal: map[string]string{},
	}
	st.meetings[id] = m
	st.tokens[token] = id
	return m, nil
}

func (m *cmtgMeeting) view() map[string]any {
	host := "app.chime.aws"
	if m.network == "DualStack" {
		host = "app.dualstack.chime.aws"
	}
	base := "https://" + m.region + "." + host
	out := map[string]any{
		"MeetingId": m.id, "MeetingArn": m.arn, "ExternalMeetingId": m.externalID, "MediaRegion": m.region,
		"MediaPlacement": map[string]any{
			"AudioHostUrl":      m.id + ".k.m1." + m.region + "." + host + ":3478",
			"AudioFallbackUrl":  "wss://wss." + m.region + "." + host + ":443/calls/" + m.id,
			"SignalingUrl":      "wss://signal." + m.region + "." + host + "/control/" + m.id,
			"TurnControlUrl":    base + "/v2/turn_sessions",
			"ScreenDataUrl":     "wss://bitpw." + m.region + "." + host + ":443/v2/screen/" + m.id,
			"ScreenViewingUrl":  "wss://bitpw." + m.region + "." + host + ":443/ws/connect?passcode=null&viewer_uuid=null&X-BitHub-Call-Id=" + m.id,
			"ScreenSharingUrl":  "wss://bitpw." + m.region + "." + host + ":443/v2/screen/" + m.id,
			"EventIngestionUrl": "https://data.svc." + m.region + ".ingest." + host + "/v1/client-events",
		},
	}
	if m.hostID != "" {
		out["MeetingHostId"] = m.hostID
	}
	if m.primaryID != "" {
		out["PrimaryMeetingId"] = m.primaryID
	}
	if m.features != nil {
		out["MeetingFeatures"] = chimeSDKCopy(m.features)
	}
	if len(m.tenantIDs) > 0 {
		out["TenantIds"] = slices.Clone(m.tenantIDs)
	}
	return out
}

func (st *cmtgStore) getMeeting(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Meeting": m.view()}, nil
}

func (st *cmtgStore) deleteMeeting(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	delete(st.meetings, m.id)
	delete(st.tokens, m.token)
	return nil, nil
}

func (st *cmtgStore) createMeetingWithAttendees(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.newMeeting(in)
	if cerr != nil {
		return nil, cerr
	}
	attendees, errs := m.addAttendees(mmgrList(in, "Attendees"))
	return map[string]any{"Meeting": m.view(), "Attendees": attendees, "Errors": errs}, nil
}

// cmtgCapabilities returns the requested capabilities (SendReceive by
// default, None where the meeting disables video or content), checked
// against the meeting features and the content-needs-video rule.
func (m *cmtgMeeting) capabilities(req map[string]any) (map[string]any, *chimeSDKErr) {
	videoOff := mmgrStr(mmgrMap(m.features, "Video"), "MaxResolution") == "None"
	contentOff := mmgrStr(mmgrMap(m.features, "Content"), "MaxResolution") == "None"
	if req == nil {
		req = map[string]any{"Audio": "SendReceive", "Video": "SendReceive", "Content": "SendReceive"}
		if videoOff {
			req["Video"] = "None"
		}
		if contentOff || videoOff {
			req["Content"] = "None"
		}
	}
	video, content := mmgrStr(req, "Video"), mmgrStr(req, "Content")
	switch {
	case videoOff && video != "None":
		return nil, chimeSDKBadRequest("Video capabilities must be None when the meeting's video MaxResolution is None.")
	case contentOff && content != "None":
		return nil, chimeSDKBadRequest("Content capabilities must be None when the meeting's content MaxResolution is None.")
	case (content == "SendReceive" || content == "Receive") && video != "SendReceive" && video != "Receive":
		return nil, chimeSDKBadRequest("Content capabilities SendReceive or Receive require video capabilities SendReceive or Receive.")
	}
	return map[string]any{"Audio": mmgrStr(req, "Audio"), "Video": video, "Content": content}, nil
}

// addAttendee adds (or, for a known ExternalUserId, returns) an attendee
// and a fresh join token.
func (m *cmtgMeeting) addAttendee(item map[string]any) (map[string]any, *chimeSDKErr) {
	ext := mmgrStr(item, "ExternalUserId")
	caps, cerr := m.capabilities(mmgrMap(item, "Capabilities"))
	if cerr != nil {
		return nil, cerr
	}
	a := m.attendees[m.byExternal[ext]]
	if a == nil {
		if len(m.attendees) >= m.maxAttendees {
			return nil, chimeSDKError(http.StatusBadRequest, "LimitExceededException", "The meeting has its maximum of %d attendees.", m.maxAttendees)
		}
		a = &cmtgAttendee{id: uuid.NewString(), externalID: ext, caps: caps}
		m.attendees[a.id] = a
		m.byExternal[ext] = a.id
	}
	out := a.view()
	out["JoinToken"] = chimeSDKSecret(96)
	return out, nil
}

func (m *cmtgMeeting) addAttendees(items []any) ([]any, []any) {
	attendees, errs := []any{}, []any{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		a, cerr := m.addAttendee(item)
		if cerr != nil {
			errs = append(errs, map[string]any{"ExternalUserId": mmgrStr(item, "ExternalUserId"), "ErrorCode": chimeSDKErrorCodes[cerr.code], "ErrorMessage": cerr.msg})
			continue
		}
		attendees = append(attendees, a)
	}
	return attendees, errs
}

func (a *cmtgAttendee) view() map[string]any {
	return map[string]any{"AttendeeId": a.id, "ExternalUserId": a.externalID, "Capabilities": chimeSDKCopy(a.caps)}
}

func (st *cmtgStore) createAttendee(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	a, cerr := m.addAttendee(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Attendee": a}, nil
}

func (st *cmtgStore) batchCreateAttendee(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	attendees, errs := m.addAttendees(mmgrList(in, "Attendees"))
	return map[string]any{"Attendees": attendees, "Errors": errs}, nil
}

func (st *cmtgStore) attendee(in map[string]any) (*cmtgMeeting, *cmtgAttendee, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, nil, cerr
	}
	a := m.attendees[mmgrStr(in, "AttendeeId")]
	if a == nil {
		return nil, nil, chimeSDKNotFound("The attendee %s doesn't exist.", mmgrStr(in, "AttendeeId"))
	}
	return m, a, nil
}

func (st *cmtgStore) getAttendee(in map[string]any) (map[string]any, *chimeSDKErr) {
	_, a, cerr := st.attendee(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Attendee": a.view()}, nil
}

func (st *cmtgStore) listAttendees(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	ids := make([]string, 0, len(m.attendees))
	for id := range m.attendees {
		ids = append(ids, id)
	}
	page, next := chimeSDKPage(ids, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 100)
	out := make([]any, 0, len(page))
	for _, id := range page {
		out = append(out, m.attendees[id].view())
	}
	resp := map[string]any{"Attendees": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (st *cmtgStore) deleteAttendee(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, a, cerr := st.attendee(in)
	if cerr != nil {
		return nil, cerr
	}
	delete(m.attendees, a.id)
	delete(m.byExternal, a.externalID)
	return nil, nil
}

func (st *cmtgStore) updateAttendeeCapabilities(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, a, cerr := st.attendee(in)
	if cerr != nil {
		return nil, cerr
	}
	caps, cerr := m.capabilities(mmgrMap(in, "Capabilities"))
	if cerr != nil {
		return nil, cerr
	}
	a.caps = caps
	return map[string]any{"Attendee": a.view()}, nil
}

func (st *cmtgStore) batchUpdateCapabilitiesExcept(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	caps, cerr := m.capabilities(mmgrMap(in, "Capabilities"))
	if cerr != nil {
		return nil, cerr
	}
	excluded := map[string]bool{}
	for _, raw := range mmgrList(in, "ExcludedAttendeeIds") {
		id := mmgrStr(chimeSDKMapOf(raw), "AttendeeId")
		if m.attendees[id] == nil {
			return nil, chimeSDKNotFound("The attendee %s doesn't exist.", id)
		}
		excluded[id] = true
	}
	for id, a := range m.attendees {
		if !excluded[id] {
			a.caps = chimeSDKCopy(caps)
		}
	}
	return nil, nil
}

func (st *cmtgStore) startTranscription(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	cfg := mmgrMap(in, "TranscriptionConfiguration")
	std, medical := mmgrMap(cfg, "EngineTranscribeSettings"), mmgrMap(cfg, "EngineTranscribeMedicalSettings")
	if (std == nil) == (medical == nil) {
		return nil, chimeSDKBadRequest("TranscriptionConfiguration takes exactly one of EngineTranscribeSettings or EngineTranscribeMedicalSettings.")
	}
	if std != nil {
		if cerr := cmtgCheckTranscribe(std); cerr != nil {
			return nil, cerr
		}
	}
	if m.transcription != nil {
		return nil, chimeSDKConflict("Transcription is already running for meeting %s.", m.id)
	}
	m.transcription = cfg
	return nil, nil
}

func cmtgCheckTranscribe(s map[string]any) *chimeSDKErr {
	identify, _ := s["IdentifyLanguage"].(bool)
	switch {
	case identify && mmgrStr(s, "LanguageCode") != "":
		return chimeSDKBadRequest("LanguageCode can't be combined with IdentifyLanguage.")
	case !identify && mmgrStr(s, "LanguageCode") == "":
		return chimeSDKBadRequest("EngineTranscribeSettings needs LanguageCode or IdentifyLanguage.")
	case identify && mmgrStr(s, "LanguageOptions") == "":
		return chimeSDKBadRequest("IdentifyLanguage needs LanguageOptions.")
	case mmgrStr(s, "ContentIdentificationType") != "" && mmgrStr(s, "ContentRedactionType") != "":
		return chimeSDKBadRequest("ContentIdentificationType and ContentRedactionType can't both be set.")
	case mmgrStr(s, "PiiEntityTypes") != "" && mmgrStr(s, "ContentIdentificationType") == "" && mmgrStr(s, "ContentRedactionType") == "":
		return chimeSDKBadRequest("PiiEntityTypes needs ContentIdentificationType or ContentRedactionType.")
	case mmgrStr(s, "VocabularyFilterMethod") != "" && mmgrStr(s, "VocabularyFilterName") == "" && mmgrStr(s, "VocabularyFilterNames") == "":
		return chimeSDKBadRequest("VocabularyFilterMethod needs a vocabulary filter.")
	}
	return nil
}

func (st *cmtgStore) stopTranscription(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.meeting(in)
	if cerr != nil {
		return nil, cerr
	}
	if m.transcription == nil {
		return nil, chimeSDKNotFound("Transcription isn't running for meeting %s.", m.id)
	}
	m.transcription = nil
	return nil, nil
}

func cmtgTooManyTags() *chimeSDKErr {
	return chimeSDKError(http.StatusBadRequest, "TooManyTagsException", "A meeting can have at most %d tags.", cmtgMaxTags)
}

func (st *cmtgStore) taggedMeeting(in map[string]any) (*cmtgMeeting, *chimeSDKErr) {
	arn := mmgrStr(in, "ResourceARN")
	m := st.meetings[cmtgMeetingIDFromARN(arn)]
	if m == nil || m.arn != arn {
		return nil, chimeSDKError(http.StatusNotFound, "ResourceNotFoundException", "The resource %s doesn't exist.", arn)
	}
	return m, nil
}

func (st *cmtgStore) tagResource(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.taggedMeeting(in)
	if cerr != nil {
		return nil, cerr
	}
	tags, cerr := chimeSDKMergeTags(m.tags, mmgrList(in, "Tags"), cmtgMaxTags, cmtgTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	m.tags = tags
	return nil, nil
}

func (st *cmtgStore) untagResource(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.taggedMeeting(in)
	if cerr != nil {
		return nil, cerr
	}
	m.tags = chimeSDKUntag(m.tags, mmgrList(in, "TagKeys"))
	return nil, nil
}

func (st *cmtgStore) listTags(in map[string]any) (map[string]any, *chimeSDKErr) {
	m, cerr := st.taggedMeeting(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Tags": chimeSDKTagList(m.tags)}, nil
}
