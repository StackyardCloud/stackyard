package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Chat contacts started with Amazon Connect's StartChatContact live here: the
// participants, their bearer tokens, the transcript and attachments. Connect
// (StartChatContact, CreateParticipant, StopContact, DescribeContact), Connect
// Participant, Contact Lens and Connect Cases all read this store. Lock order:
// connectStore.mu before cnpStore.mu; this store never calls out.

const (
	cnpMaxChats           = 500
	cnpMaxParticipants    = 6
	cnpMaxItemsPerChat    = 1000
	cnpMaxChatBytes       = 512 << 10
	cnpMaxStoreBytes      = 32 << 20
	cnpMaxAttachment      = 20 << 20
	cnpMaxAttachmentBytes = 64 << 20
	cnpMaxAttachments     = 20
	cnpMaxAuthSessions    = 10
	cnpTimeLayout         = "2006-01-02T15:04:05.000Z"
	cnpUploadWindow       = 5 * time.Minute
	cnpCredentialLifetime = 24 * time.Hour
)

type cnpErr struct {
	status    int
	code, msg string
}

func cnpValidation(format string, args ...any) *cnpErr {
	return &cnpErr{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func cnpDenied(format string, args ...any) *cnpErr {
	return &cnpErr{http.StatusForbidden, "AccessDeniedException", fmt.Sprintf(format, args...)}
}

func cnpConflict(format string, args ...any) *cnpErr {
	return &cnpErr{http.StatusConflict, "ConflictException", fmt.Sprintf(format, args...)}
}

func cnpQuota(format string, args ...any) *cnpErr {
	return &cnpErr{http.StatusPaymentRequired, "ServiceQuotaExceededException", fmt.Sprintf(format, args...)}
}

type cnpStore struct {
	mu          sync.Mutex
	chats       map[string]*cnpChat        // contact id
	byPartToken map[string]*cnpParticipant // participant token
	byConnToken map[string]*cnpParticipant // connection token
	attachments map[string]*cnpAttachment
	bytes       int   // transcript bytes, store-wide
	attBytes    int64 // reserved attachment bytes, store-wide
}

type cnpChat struct {
	instanceID, contactID, flowID string
	started, ended                time.Time
	participants                  []*cnpParticipant
	items                         []*cnpItem
	clientTokens                  map[string]*cnpItem
	attachments                   []*cnpAttachment
	auth                          map[string]*cnpAuthSession
	bytes                         int
}

type cnpParticipant struct {
	chat                     *cnpChat
	id, displayName, role    string
	partToken, connToken     string
	disconnectToken          string
	connected, disconnected  bool
	disconnectedAt, joinedAt time.Time
}

type cnpItem struct {
	id, typ, contentType, content string
	participant                   *cnpParticipant // nil for SYSTEM events
	at                            time.Time
	attachment                    *cnpAttachment
	receipts                      map[string]*cnpReceipt
}

type cnpReceipt struct{ delivered, read time.Time }

type cnpAttachment struct {
	id, name, contentType, clientToken string
	chat                               *cnpChat
	owner                              *cnpParticipant
	size                               int64
	data                               []byte
	uploadKey, downloadKey             string
	uploadExpiry, downloadExpiry       time.Time
	completed                          bool
}

type cnpAuthSession struct {
	redirect  string
	cancelled bool
}

func newConnectParticipantStore() *cnpStore {
	return &cnpStore{
		chats:       map[string]*cnpChat{},
		byPartToken: map[string]*cnpParticipant{},
		byConnToken: map[string]*cnpParticipant{},
		attachments: map[string]*cnpAttachment{},
	}
}

// cnpToken returns a fresh 256-bit bearer token.
func cnpToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func cnpTime(t time.Time) string { return t.UTC().Format(cnpTimeLayout) }

func cnpContactARN(instanceID, contactID string) string {
	return fmt.Sprintf("%s/contact/%s", connectInstanceARN(instanceID), contactID)
}

// cnpChatStart is what StartChatContact asks for.
type cnpChatStart struct {
	instanceID, flowID, displayName string
	initialType, initialContent     string
}

// startChat opens a chat contact with its customer participant and returns
// the contact id, participant id and participant token.
func (st *cnpStore) startChat(req cnpChatStart) (string, string, string, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.chats) >= cnpMaxChats && !st.evictEndedLocked() {
		return "", "", "", &cnpErr{http.StatusTooManyRequests, "LimitExceededException", fmt.Sprintf("at most %d chat contacts can be open", cnpMaxChats)}
	}
	now := time.Now().UTC()
	chat := &cnpChat{instanceID: req.instanceID, contactID: uuid.NewString(), flowID: req.flowID, started: now,
		clientTokens: map[string]*cnpItem{}, auth: map[string]*cnpAuthSession{}}
	p := st.addParticipantLocked(chat, req.displayName, "CUSTOMER")
	if p == nil {
		return "", "", "", cnpQuota("chat transcript storage is full")
	}
	st.chats[chat.contactID] = chat
	if req.initialContent != "" {
		if _, err := st.appendLocked(chat, &cnpItem{typ: "MESSAGE", contentType: req.initialType, content: req.initialContent, participant: p}); err != nil {
			st.dropChatLocked(chat)
			return "", "", "", err
		}
	}
	return chat.contactID, p.id, p.partToken, nil
}

// addParticipantLocked joins a participant and records the joined event.
func (st *cnpStore) addParticipantLocked(chat *cnpChat, name, role string) *cnpParticipant {
	p := &cnpParticipant{chat: chat, id: uuid.NewString(), displayName: name, role: role, partToken: cnpToken(), joinedAt: time.Now().UTC()}
	if _, err := st.appendLocked(chat, &cnpItem{typ: "PARTICIPANT_JOINED", contentType: "application/vnd.amazonaws.connect.event.participant.joined", participant: p}); err != nil {
		return nil
	}
	chat.participants = append(chat.participants, p)
	st.byPartToken[p.partToken] = p
	return p
}

// addParticipant is Connect's CreateParticipant.
func (st *cnpStore) addParticipant(instanceID, contactID, name, role string) (string, string, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	chat := st.chats[contactID]
	if chat == nil || chat.instanceID != instanceID {
		return "", "", &cnpErr{http.StatusNotFound, "ResourceNotFoundException", "chat contact not found"}
	}
	if !chat.ended.IsZero() {
		return "", "", &cnpErr{http.StatusBadRequest, "InvalidRequestException", "the chat has ended"}
	}
	if len(chat.participants) >= cnpMaxParticipants {
		return "", "", &cnpErr{http.StatusTooManyRequests, "ServiceQuotaExceededException", fmt.Sprintf("a chat holds at most %d participants", cnpMaxParticipants)}
	}
	p := st.addParticipantLocked(chat, name, role)
	if p == nil {
		return "", "", &cnpErr{http.StatusTooManyRequests, "ServiceQuotaExceededException", "chat transcript storage is full"}
	}
	return p.id, p.partToken, nil
}

// endChat is Connect's StopContact for a chat; it reports whether contactID
// names a chat of the instance.
func (st *cnpStore) endChat(instanceID, contactID string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	chat := st.chats[contactID]
	if chat == nil || chat.instanceID != instanceID {
		return false
	}
	st.endLocked(chat)
	return true
}

func (st *cnpStore) endLocked(chat *cnpChat) {
	if !chat.ended.IsZero() {
		return
	}
	// The ended event is always recorded, even past the transcript budget.
	item := &cnpItem{id: uuid.NewString(), typ: "CHAT_ENDED", contentType: "application/vnd.amazonaws.connect.event.chat.ended", at: time.Now().UTC()}
	chat.items = append(chat.items, item)
	chat.ended = item.at
}

// cnpContactView is a copy of what other services read about a chat contact.
type cnpContactView struct {
	instanceID, contactID string
	started, ended        time.Time
	messages              []cnpMessageView
}

type cnpMessageView struct {
	id, role, content string
	offset            time.Duration
}

// contact copies a chat contact's details (and text messages when withMessages).
func (st *cnpStore) contact(instanceID, contactID string, withMessages bool) (cnpContactView, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	chat := st.chats[contactID]
	if chat == nil || (instanceID != "" && chat.instanceID != instanceID) {
		return cnpContactView{}, false
	}
	v := cnpContactView{instanceID: chat.instanceID, contactID: chat.contactID, started: chat.started, ended: chat.ended}
	if withMessages {
		for _, it := range chat.items {
			if it.typ == "MESSAGE" && it.participant != nil && (it.contentType == "text/plain" || it.contentType == "text/markdown") {
				v.messages = append(v.messages, cnpMessageView{it.id, it.participant.role, it.content, it.at.Sub(chat.started)})
			}
		}
	}
	return v, true
}

// appendLocked adds an item to the transcript within the chat and store budgets.
func (st *cnpStore) appendLocked(chat *cnpChat, item *cnpItem) (*cnpItem, *cnpErr) {
	size := len(item.content) + len(item.contentType) + 128
	switch {
	case len(chat.items) >= cnpMaxItemsPerChat:
		return nil, cnpQuota("a chat transcript holds at most %d items", cnpMaxItemsPerChat)
	case chat.bytes+size > cnpMaxChatBytes:
		return nil, cnpQuota("a chat transcript holds at most %d bytes", cnpMaxChatBytes)
	case st.bytes+size > cnpMaxStoreBytes:
		return nil, cnpQuota("chat transcript storage is full")
	}
	item.id = uuid.NewString()
	item.at = time.Now().UTC()
	chat.items = append(chat.items, item)
	chat.bytes += size
	st.bytes += size
	return item, nil
}

// evictEndedLocked drops the chat that ended first; false when none has ended.
func (st *cnpStore) evictEndedLocked() bool {
	var oldest *cnpChat
	for _, c := range st.chats {
		if !c.ended.IsZero() && (oldest == nil || c.ended.Before(oldest.ended)) {
			oldest = c
		}
	}
	if oldest == nil {
		return false
	}
	st.dropChatLocked(oldest)
	return true
}

func (st *cnpStore) dropChatLocked(chat *cnpChat) {
	for _, p := range chat.participants {
		delete(st.byPartToken, p.partToken)
		delete(st.byConnToken, p.connToken)
	}
	for _, a := range chat.attachments {
		delete(st.attachments, a.id)
		st.attBytes -= a.size
	}
	st.bytes -= chat.bytes
	delete(st.chats, chat.contactID)
}

// byConnection resolves a connection token. Tokens are compared by map lookup
// and never echoed back in errors.
func (st *cnpStore) byConnectionLocked(token string) (*cnpParticipant, *cnpErr) {
	p := st.byConnToken[token]
	if p == nil || p.disconnected {
		return nil, cnpDenied("the connection token is not valid")
	}
	return p, nil
}

func (st *cnpStore) activeLocked(token string) (*cnpParticipant, *cnpErr) {
	p, err := st.byConnectionLocked(token)
	if err != nil {
		return nil, err
	}
	if !p.chat.ended.IsZero() {
		return nil, cnpDenied("the chat has ended")
	}
	return p, nil
}

// itemOut shapes a transcript item.
func (chat *cnpChat) itemOut(it *cnpItem) map[string]any {
	out := map[string]any{"Id": it.id, "Type": it.typ, "AbsoluteTime": cnpTime(it.at), "ContactId": chat.contactID}
	if it.contentType != "" {
		out["ContentType"] = it.contentType
	}
	if it.content != "" {
		out["Content"] = it.content
	}
	if it.participant == nil {
		out["ParticipantRole"] = "SYSTEM"
	} else {
		out["ParticipantId"] = it.participant.id
		out["DisplayName"] = it.participant.displayName
		out["ParticipantRole"] = it.participant.role
	}
	if a := it.attachment; a != nil {
		out["Attachments"] = []any{map[string]any{"AttachmentId": a.id, "AttachmentName": a.name, "ContentType": a.contentType, "Status": "APPROVED"}}
	}
	if it.typ == "MESSAGE" {
		meta := map[string]any{"MessageId": it.id}
		ids := make([]string, 0, len(it.receipts))
		for id := range it.receipts {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		receipts := []any{}
		for _, id := range ids {
			r := map[string]any{"RecipientParticipantId": id}
			if rc := it.receipts[id]; !rc.delivered.IsZero() {
				r["DeliveredTimestamp"] = cnpTime(rc.delivered)
			}
			if rc := it.receipts[id]; !rc.read.IsZero() {
				r["ReadTimestamp"] = cnpTime(rc.read)
			}
			receipts = append(receipts, r)
		}
		meta["Receipts"] = receipts
		out["MessageMetadata"] = meta
	}
	return out
}

// keyMatches compares an attachment URL key in constant time.
func cnpKeyMatches(want, got string) bool {
	return want != "" && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}
