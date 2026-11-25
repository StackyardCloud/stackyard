package server

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Chat attachments: StartAttachmentUpload reserves the declared size and
// returns a URL on this server that takes exactly that many bytes (the stand-in
// for S3's pre-signed PUT); CompleteAttachmentUpload posts it to the
// transcript and GetAttachment hands out a short-lived download URL.

const cnpAttachmentPath = "/_stackyard/connectparticipant/attachments/"

// cnpAttachmentTypes are the MIME types Amazon Connect chat accepts.
var cnpAttachmentTypes = map[string]bool{
	"text/csv": true, "text/plain": true, "application/pdf": true, "application/json": true, "application/rtf": true,
	"application/msword": true, "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.ms-powerpoint": true, "application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"image/jpeg": true, "image/png": true, "image/heic": true, "audio/wav": true, "audio/x-wav": true,
	"video/mp4": true, "video/quicktime": true,
}

func (st *cnpStore) startAttachmentUpload(req cnpRequest) (map[string]any, *cnpErr) {
	ct := strings.ToLower(cnpStr(req.in, "ContentType"))
	if !cnpAttachmentTypes[ct] {
		return nil, cnpValidation("attachments of type %q aren't supported", ct)
	}
	size, _ := req.in["AttachmentSizeInBytes"].(float64)
	if size > cnpMaxAttachment {
		return nil, cnpValidation("attachments are at most %d bytes", cnpMaxAttachment)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	chat := p.chat
	token := cnpStr(req.in, "ClientToken")
	var a *cnpAttachment
	for _, prev := range chat.attachments {
		if prev.clientToken == token && prev.owner == p {
			a = prev
		}
	}
	if a == nil {
		if len(chat.attachments) >= cnpMaxAttachments {
			return nil, cnpQuota("a chat holds at most %d attachments", cnpMaxAttachments)
		}
		if st.attBytes+int64(size) > cnpMaxAttachmentBytes {
			return nil, cnpQuota("attachment storage is full")
		}
		a = &cnpAttachment{id: uuid.NewString(), name: cnpStr(req.in, "AttachmentName"), contentType: ct, clientToken: token,
			chat: chat, owner: p, size: int64(size), uploadKey: cnpToken(), uploadExpiry: time.Now().Add(cnpUploadWindow)}
		chat.attachments = append(chat.attachments, a)
		st.attachments[a.id] = a
		st.attBytes += a.size
	}
	return map[string]any{"AttachmentId": a.id, "UploadMetadata": map[string]any{
		"Url":              req.base + cnpAttachmentPath + a.id + "?key=" + url.QueryEscape(a.uploadKey),
		"UrlExpiry":        cnpTime(a.uploadExpiry),
		"HeadersToInclude": map[string]any{"Content-Type": a.contentType, "Content-Length": strconv.FormatInt(a.size, 10)},
	}}, nil
}

// chatAttachmentLocked finds an attachment of the caller's chat.
func (st *cnpStore) chatAttachmentLocked(p *cnpParticipant, id string) (*cnpAttachment, *cnpErr) {
	a := st.attachments[id]
	if a == nil || a.chat != p.chat {
		return nil, cnpValidation("attachment %s isn't in this chat", id)
	}
	return a, nil
}

func (st *cnpStore) completeAttachmentUpload(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.activeLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	ids, _ := req.in["AttachmentIds"].([]any)
	for _, raw := range ids {
		a, err := st.chatAttachmentLocked(p, raw.(string))
		if err != nil {
			return nil, err
		}
		if a.owner != p {
			return nil, cnpDenied("attachment %s was started by another participant", a.id)
		}
		if a.data == nil {
			return nil, cnpConflict("attachment %s hasn't been uploaded", a.id)
		}
		if a.completed {
			continue
		}
		if _, err := st.appendLocked(p.chat, &cnpItem{typ: "ATTACHMENT", contentType: a.contentType, participant: p, attachment: a}); err != nil {
			return nil, err
		}
		a.completed = true
	}
	return nil, nil
}

func (st *cnpStore) getAttachment(req cnpRequest) (map[string]any, *cnpErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.byConnectionLocked(req.bearer)
	if err != nil {
		return nil, err
	}
	a, err := st.chatAttachmentLocked(p, cnpStr(req.in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if !a.completed {
		return nil, cnpValidation("attachment %s isn't approved", a.id)
	}
	secs := cnpInt(req.in, "UrlExpiryInSeconds", 300)
	a.downloadKey = cnpToken()
	a.downloadExpiry = time.Now().Add(time.Duration(secs) * time.Second)
	disposition := "attachment"
	if cnpStr(req.in, "UrlContentDisposition") == "INLINE" {
		disposition = "inline"
	}
	return map[string]any{
		"Url":                   req.base + cnpAttachmentPath + a.id + "?key=" + url.QueryEscape(a.downloadKey) + "&disposition=" + disposition,
		"UrlExpiry":             cnpTime(a.downloadExpiry),
		"AttachmentSizeInBytes": float64(a.size),
	}, nil
}

// handleConnectParticipantUpload takes the bytes of a started attachment.
func (s *Server) handleConnectParticipantUpload(w http.ResponseWriter, r *http.Request) {
	st := s.connectParticipant
	id, key := r.PathValue("id"), r.URL.Query().Get("key")
	st.mu.Lock()
	a := st.attachments[id]
	ok := a != nil && cnpKeyMatches(a.uploadKey, key) && time.Now().Before(a.uploadExpiry) && a.data == nil
	var size int64
	var ct string
	if ok {
		size, ct = a.size, a.contentType
	}
	st.mu.Unlock()
	if !ok {
		cnpPlainError(w, http.StatusForbidden, "the upload URL is not valid or has expired")
		return
	}
	if r.ContentLength != size {
		cnpPlainError(w, http.StatusBadRequest, "the body must be exactly the attachment's declared size")
		return
	}
	if got, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); got != ct {
		cnpPlainError(w, http.StatusBadRequest, "Content-Type must match the attachment's declared type")
		return
	}
	// The body is read outside the lock; its size was reserved at start.
	data, err := io.ReadAll(io.LimitReader(r.Body, size))
	if err != nil || int64(len(data)) != size {
		cnpPlainError(w, http.StatusBadRequest, "the body ended early")
		return
	}
	st.mu.Lock()
	stored := st.attachments[id] == a && a.data == nil
	if stored {
		a.data = data
		a.uploadKey = ""
	}
	st.mu.Unlock()
	if !stored {
		cnpPlainError(w, http.StatusConflict, "the attachment was already uploaded or removed")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleConnectParticipantDownload serves an approved attachment. Caller bytes
// always go out as application/octet-stream with nosniff.
func (s *Server) handleConnectParticipantDownload(w http.ResponseWriter, r *http.Request) {
	st := s.connectParticipant
	id, key := r.PathValue("id"), r.URL.Query().Get("key")
	st.mu.Lock()
	a := st.attachments[id]
	var data []byte
	var name string
	if a != nil && a.completed && cnpKeyMatches(a.downloadKey, key) && time.Now().Before(a.downloadExpiry) {
		data, name = a.data, a.name // stored bytes are never mutated after upload
	}
	st.mu.Unlock()
	if data == nil {
		cnpPlainError(w, http.StatusForbidden, "the download URL is not valid or has expired")
		return
	}
	disposition := "attachment"
	if r.URL.Query().Get("disposition") == "inline" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func cnpPlainError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, msg+"\n")
}
