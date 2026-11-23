package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// tssStore keeps Medical Scribe sessions for GetMedicalScribeStream and for
// resuming a paused session with the same SessionId.
type tssStore struct {
	mu       sync.Mutex
	sessions map[string]*tssScribe
}

type tssScribe struct {
	p       tssParams
	cfg     *tssScribeConfig
	status  string // IN_PROGRESS, PAUSED, FAILED, COMPLETED
	resumed bool   // the running stream continues a paused session
	created time.Time
	ended   time.Time
	seconds float64
}

func newTranscribeStreamingStore() *tssStore {
	return &tssStore{sessions: map[string]*tssScribe{}}
}

// claim starts a stream on a new or paused session. A non-empty code is the
// error to answer with before the stream starts.
func (st *tssStore) claim(p tssParams) (int, string, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if sess, ok := st.sessions[p.sessionID]; ok {
		switch sess.status {
		case "IN_PROGRESS":
			return http.StatusConflict, "ConflictException", "the Medical Scribe session is already streaming"
		case "PAUSED":
			if sess.p.rate != p.rate || sess.p.encoding != p.encoding || sess.p.language != p.language {
				return http.StatusBadRequest, "BadRequestException", "a resumed session must keep its language, sample rate and encoding"
			}
			sess.status, sess.resumed = "IN_PROGRESS", true
			return 0, "", ""
		}
		return http.StatusBadRequest, "BadRequestException", "the Medical Scribe session has ended"
	}
	if len(st.sessions) >= tssMaxSessions {
		return http.StatusTooManyRequests, "LimitExceededException", "too many Medical Scribe sessions"
	}
	st.sessions[p.sessionID] = &tssScribe{p: p, status: "IN_PROGRESS", created: time.Now().UTC()}
	return 0, "", ""
}

// resumable reports whether the running stream continues a configured session.
func (st *tssStore) resumable(id string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	sess := st.sessions[id]
	return sess != nil && sess.resumed
}

func (st *tssStore) configure(p tssParams, cfg *tssScribeConfig) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if sess := st.sessions[p.sessionID]; sess != nil {
		sess.cfg = cfg
	}
}

func (st *tssStore) fail(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if sess := st.sessions[id]; sess != nil {
		sess.status = "FAILED"
		sess.ended = time.Now().UTC()
	}
}

// end records how a stream finished: END_OF_SESSION completes the session, a
// dropped stream pauses it. A session that never got its configuration is
// forgotten.
func (st *tssStore) end(id string, done bool, seconds float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sess := st.sessions[id]
	if sess == nil {
		return
	}
	if sess.cfg == nil {
		delete(st.sessions, id)
		return
	}
	sess.seconds += seconds
	sess.resumed = false
	switch {
	case sess.status == "FAILED":
	case done:
		sess.status, sess.ended = "COMPLETED", time.Now().UTC()
	default:
		sess.status = "PAUSED"
	}
}

// details renders MedicalScribeStreamDetails under the lock.
func (st *tssStore) details(id string) (map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sess := st.sessions[id]
	if sess == nil || sess.cfg == nil {
		return nil, false
	}
	cfg := sess.cfg
	out := map[string]any{
		"SessionId":             id,
		"StreamCreatedAt":       float64(sess.created.UnixMilli()) / 1000,
		"LanguageCode":          sess.p.language,
		"MediaSampleRateHertz":  sess.p.rate,
		"MediaEncoding":         sess.p.encoding,
		"ResourceAccessRoleArn": cfg.ResourceAccessRoleArn,
		"StreamStatus":          sess.status,
		"PostStreamAnalyticsSettings": map[string]any{"ClinicalNoteGenerationSettings": map[string]any{
			"OutputBucketName": cfg.PostStreamAnalyticsSettings.ClinicalNoteGenerationSettings.OutputBucketName,
		}},
	}
	if t := cfg.PostStreamAnalyticsSettings.ClinicalNoteGenerationSettings.NoteTemplate; t != "" {
		out["PostStreamAnalyticsSettings"].(map[string]any)["ClinicalNoteGenerationSettings"].(map[string]any)["NoteTemplate"] = t
	}
	for key, value := range map[string]string{"VocabularyName": cfg.VocabularyName, "VocabularyFilterName": cfg.VocabularyFilterName, "VocabularyFilterMethod": cfg.VocabularyFilterMethod} {
		if value != "" {
			out[key] = value
		}
	}
	if len(cfg.ChannelDefinitions) > 0 {
		defs := make([]any, 0, len(cfg.ChannelDefinitions))
		for _, d := range cfg.ChannelDefinitions {
			defs = append(defs, map[string]any{"ChannelId": d.ChannelId, "ParticipantRole": d.ParticipantRole})
		}
		out["ChannelDefinitions"] = defs
	}
	if e := cfg.EncryptionSettings; e != nil {
		enc := map[string]any{"KmsKeyId": e.KmsKeyId}
		if len(e.KmsEncryptionContext) > 0 {
			ctx := make(map[string]any, len(e.KmsEncryptionContext))
			for k, v := range e.KmsEncryptionContext {
				ctx[k] = v
			}
			enc["KmsEncryptionContext"] = ctx
		}
		out["EncryptionSettings"] = enc
	}
	if !sess.ended.IsZero() {
		out["StreamEndedAt"] = float64(sess.ended.UnixMilli()) / 1000
	}
	if sess.status == "COMPLETED" {
		bucket := strings.TrimPrefix(cfg.PostStreamAnalyticsSettings.ClinicalNoteGenerationSettings.OutputBucketName, "s3://")
		out["PostStreamAnalyticsResult"] = map[string]any{"ClinicalNoteGenerationResult": map[string]any{
			"ClinicalNoteOutputLocation": "s3://" + bucket + "/" + id + "/summary.json",
			"TranscriptOutputLocation":   "s3://" + bucket + "/" + id + "/transcript.json",
			"Status":                     "COMPLETED",
		}}
	}
	return out, true
}
