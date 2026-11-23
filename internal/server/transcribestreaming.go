package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Amazon Transcribe Streaming: five REST operations on the `transcribe`
// signing name. Four are bidirectional event streams (audio frames in,
// transcript frames out while the request is still open); the fifth reads a
// stored Medical Scribe session. Batch Transcribe (X-Amz-Target JSON on "/")
// never uses these paths, so the two never collide.
//
// There is no speech engine: transcripts are deterministic descriptions of the
// audio received (see tssSession.result).

const (
	tssMaxStreamBytes = 64 << 20  // audio bytes accepted on one stream
	tssMaxEventBytes  = 256 << 10 // payload of a single event frame
	tssMaxSessions    = 1000      // stored Medical Scribe sessions
	tssSegmentSeconds = 5.0       // a result is finalised every 5 s of audio
)

var (
	tssSessionIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	tssNamePattern      = regexp.MustCompile(`^[0-9a-zA-Z._-]{1,200}$`)
	tssRoleARNPattern   = regexp.MustCompile(`^arn:aws(-[^:]+)?:iam::[0-9]{0,63}:role/[A-Za-z0-9:_/+=,@.-]{1,512}$`)

	tssLanguageCodes          = strings.Fields(`en-US en-GB es-US fr-CA fr-FR en-AU it-IT de-DE pt-BR ja-JP ko-KR zh-CN th-TH es-ES ar-SA pt-PT ca-ES ar-AE hi-IN zh-HK nl-NL no-NO sv-SE pl-PL fi-FI zh-TW en-IN en-IE en-NZ en-AB en-ZA en-WL de-CH af-ZA eu-ES hr-HR cs-CZ da-DK fa-IR gl-ES el-GR he-IL id-ID lv-LV ms-MY ro-RO ru-RU sr-RS sk-SK so-SO tl-PH uk-UA vi-VN zu-ZA am-ET be-BY bg-BG bn-IN bs-BA ckb-IQ ckb-IR cy-WL es-MX et-ET fa-AF gu-IN ht-HT hu-HU hy-AM is-IS jv-ID ka-GE kab-DZ kk-KZ km-KH kn-IN lg-IN lt-LT mk-MK ml-IN mr-IN my-MM ne-NP or-IN pa-IN ps-AF si-LK sl-SI sq-AL su-ID sw-BI sw-KE sw-RW sw-TZ sw-UG ta-IN te-IN tr-TR uz-UZ`)
	tssCallAnalyticsLanguages = strings.Fields(`en-US en-GB es-US fr-CA fr-FR en-AU it-IT de-DE pt-BR`)
	tssMediaEncodings         = []string{"pcm", "ogg-opus", "flac", "g711-alaw", "g711-ulaw", "g729"}
	tssScribeEncodings        = []string{"pcm", "ogg-opus", "flac"}
	tssSpecialties            = []string{"PRIMARYCARE", "CARDIOLOGY", "NEUROLOGY", "ONCOLOGY", "RADIOLOGY", "UROLOGY"}
)

func (s *Server) handleTranscribeStreamingRouter(w http.ResponseWriter, r *http.Request) bool {
	op, sessionID, ok := transcribeStreamingRoute(rawRequestPath(r))
	if !ok {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "transcribe" {
		return false
	}
	// Event streams keep the body open and are bounded while they are read; any
	// other body is read whole by the SigV4 check, so it must declare a length.
	if r.Header.Get("x-amz-content-sha256") != sigV4StreamingEventsPayload &&
		(r.ContentLength < 0 || r.ContentLength > tssMaxStreamBytes) {
		respondTranscribeStreamingError(w, http.StatusBadRequest, "BadRequestException", "the request body must declare a Content-Length of at most 64 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "transcribe"); !ok {
		respondTranscribeStreamingError(w, status, code, msg)
		return true
	}
	wantMethod := http.MethodPost
	if op == "GetMedicalScribeStream" {
		wantMethod = http.MethodGet
	}
	if r.Method != wantMethod {
		respondTranscribeStreamingError(w, http.StatusMethodNotAllowed, "BadRequestException", op+" requires "+wantMethod)
		return true
	}
	if op == "GetMedicalScribeStream" {
		s.transcribeStreamingGetScribe(w, sessionID)
		return true
	}
	p, msg := parseTranscribeStreamingParams(op, r.Header)
	if msg == "" {
		msg = s.transcribe.checkStreamingResources(p)
	}
	if msg != "" {
		respondTranscribeStreamingError(w, http.StatusBadRequest, "BadRequestException", msg)
		return true
	}
	if op == "StartMedicalScribeStream" {
		if status, code, msg := s.transcribeStreaming.claim(p); code != "" {
			respondTranscribeStreamingError(w, status, code, msg)
			return true
		}
	}
	s.transcribeStreamingRun(w, r, p)
	return true
}

func transcribeStreamingRoute(path string) (op, sessionID string, ok bool) {
	switch path {
	case "/stream-transcription":
		return "StartStreamTranscription", "", true
	case "/medical-stream-transcription":
		return "StartMedicalStreamTranscription", "", true
	case "/call-analytics-stream-transcription":
		return "StartCallAnalyticsStreamTranscription", "", true
	case "/medical-scribe-stream":
		return "StartMedicalScribeStream", "", true
	}
	if rest, found := strings.CutPrefix(path, "/medical-scribe-stream/"); found && rest != "" && !strings.Contains(rest, "/") {
		id, err := url.PathUnescape(rest)
		if err != nil {
			id = rest
		}
		return "GetMedicalScribeStream", id, true
	}
	return "", "", false
}

func (s *Server) transcribeStreamingGetScribe(w http.ResponseWriter, sessionID string) {
	if !tssSessionIDPattern.MatchString(sessionID) {
		respondTranscribeStreamingError(w, http.StatusBadRequest, "BadRequestException", "SessionId must be a UUID")
		return
	}
	details, found := s.transcribeStreaming.details(sessionID)
	if !found {
		respondTranscribeStreamingError(w, http.StatusNotFound, "ResourceNotFoundException", "the Medical Scribe stream session was not found")
		return
	}
	respondTranscribeStreamingJSON(w, http.StatusOK, map[string]any{"MedicalScribeStreamDetails": details})
}

// tssParams holds the validated request headers of one stream.
type tssParams struct {
	op            string
	language      string // resolved language (the preferred/first option when identifying)
	identify      bool
	sessionID     string
	encoding      string
	rate          int
	channels      int
	channelID     bool
	speakerLabels bool
	stabilize     bool
	vocabulary    string
	vocabularies  []string
	filter        string
	filters       []string
	languageModel string
	echo          map[string]string // response headers echoing the accepted settings
}

// parseTranscribeStreamingParams validates the header members of a start
// operation and returns a BadRequestException message when they are invalid.
// Messages never quote caller input.
func parseTranscribeStreamingParams(op string, h http.Header) (tssParams, string) {
	p := tssParams{op: op, channels: 1, echo: map[string]string{}}
	get := func(name string) string { return strings.TrimSpace(h.Get("x-amzn-transcribe-" + name)) }
	boolean := func(name string) (bool, bool) {
		switch strings.ToLower(get(name)) {
		case "":
			return false, true
		case "true":
			p.echo[name] = "true"
			return true, true
		case "false":
			p.echo[name] = "false"
			return false, true
		}
		return false, false
	}

	p.language = get("language-code")
	identify, ok := boolean("identify-language")
	if !ok {
		return p, "IdentifyLanguage must be true or false"
	}
	multi, ok := boolean("identify-multiple-languages")
	if !ok {
		return p, "IdentifyMultipleLanguages must be true or false"
	}
	languages := tssLanguageCodes
	switch op {
	case "StartCallAnalyticsStreamTranscription":
		languages = tssCallAnalyticsLanguages
		if multi {
			return p, "IdentifyMultipleLanguages isn't supported for Call Analytics"
		}
	case "StartMedicalStreamTranscription", "StartMedicalScribeStream":
		languages = []string{"en-US"}
		if identify || multi {
			return p, "language identification isn't supported for medical streams"
		}
	}
	chosen := 0
	for _, set := range []bool{p.language != "", identify, multi} {
		if set {
			chosen++
		}
	}
	if chosen != 1 {
		return p, "specify exactly one of LanguageCode, IdentifyLanguage or IdentifyMultipleLanguages"
	}
	if p.language != "" {
		if !slices.Contains(languages, p.language) {
			return p, "LanguageCode isn't supported for this operation"
		}
		p.echo["language-code"] = p.language
	} else {
		p.identify = true
		options := strings.Split(get("language-options"), ",")
		if len(options) < 2 || len(options) > 5 {
			return p, "LanguageOptions must list 2 to 5 language codes when identifying the language"
		}
		for i, o := range options {
			options[i] = strings.TrimSpace(o)
			if !slices.Contains(languages, options[i]) || slices.Index(options, options[i]) != i {
				return p, "LanguageOptions must be distinct, supported language codes"
			}
		}
		p.echo["language-options"] = strings.Join(options, ",")
		p.language = options[0]
		if pref := get("preferred-language"); pref != "" {
			if !slices.Contains(options, pref) {
				return p, "PreferredLanguage must be one of LanguageOptions"
			}
			p.language = pref
			p.echo["preferred-language"] = pref
		}
	}

	minRate, maxRate := 8000, 48000
	encodings := tssMediaEncodings
	switch op {
	case "StartMedicalStreamTranscription":
		minRate = 16000
		encodings = tssScribeEncodings
	case "StartMedicalScribeStream":
		minRate = 16000
		encodings = tssScribeEncodings
	}
	rate, err := strconv.ParseInt(get("sample-rate"), 10, 32)
	if err != nil || int(rate) < minRate || int(rate) > maxRate {
		return p, "MediaSampleRateHertz is required and must be within the supported range"
	}
	p.rate = int(rate)
	p.echo["sample-rate"] = strconv.Itoa(p.rate)
	p.encoding = get("media-encoding")
	if !slices.Contains(encodings, p.encoding) {
		return p, "MediaEncoding is required and must be a supported encoding"
	}
	p.echo["media-encoding"] = p.encoding

	if id := get("session-id"); id != "" {
		if !tssSessionIDPattern.MatchString(id) {
			return p, "SessionId must be a UUID"
		}
		p.sessionID = id
	} else {
		p.sessionID = uuid.NewString()
	}
	p.echo["session-id"] = p.sessionID

	if op == "StartMedicalScribeStream" {
		// Scribe settings travel in the ConfigurationEvent, not in headers.
		return p, ""
	}

	if p.channelID, ok = boolean("enable-channel-identification"); !ok {
		return p, "EnableChannelIdentification must be true or false"
	}
	if raw := get("number-of-channels"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 2 || n > 2 {
			return p, "NumberOfChannels must be 2"
		}
		p.channels = int(n)
		p.echo["number-of-channels"] = "2"
	}
	if p.channelID != (p.channels == 2) {
		return p, "EnableChannelIdentification and NumberOfChannels must be set together"
	}
	if p.speakerLabels, ok = boolean("show-speaker-label"); !ok {
		return p, "ShowSpeakerLabel must be true or false"
	}

	if op == "StartMedicalStreamTranscription" {
		specialty, typ := get("specialty"), get("type")
		if !slices.Contains(tssSpecialties, specialty) {
			return p, "Specialty is required and must be a supported specialty"
		}
		if typ != "CONVERSATION" && typ != "DICTATION" {
			return p, "Type is required and must be CONVERSATION or DICTATION"
		}
		p.echo["specialty"], p.echo["type"] = specialty, typ
		if ct := get("content-identification-type"); ct != "" {
			if ct != "PHI" {
				return p, "ContentIdentificationType must be PHI"
			}
			p.echo["content-identification-type"] = ct
		}
		if v := get("vocabulary-name"); v != "" {
			if !tssNamePattern.MatchString(v) {
				return p, "VocabularyName isn't valid"
			}
			p.vocabulary = v
			p.echo["vocabulary-name"] = v
		}
		return p, ""
	}

	// StartStreamTranscription and StartCallAnalyticsStreamTranscription.
	if p.stabilize, ok = boolean("enable-partial-results-stabilization"); !ok {
		return p, "EnablePartialResultsStabilization must be true or false"
	}
	if st := get("partial-results-stability"); st != "" {
		if !p.stabilize || (st != "high" && st != "medium" && st != "low") {
			return p, "PartialResultsStability needs EnablePartialResultsStabilization and must be high, medium or low"
		}
		p.echo["partial-results-stability"] = st
	}
	identification, redaction := get("content-identification-type"), get("content-redaction-type")
	if identification != "" && redaction != "" {
		return p, "ContentIdentificationType and ContentRedactionType can't be used together"
	}
	for name, v := range map[string]string{"content-identification-type": identification, "content-redaction-type": redaction} {
		if v != "" && v != "PII" {
			return p, "content identification and redaction only support PII"
		}
		if v != "" {
			p.echo[name] = v
		}
	}
	if pii := get("pii-entity-types"); pii != "" {
		if identification == "" && redaction == "" {
			return p, "PiiEntityTypes needs ContentIdentificationType or ContentRedactionType"
		}
		p.echo["pii-entity-types"] = "set"
	}
	for _, item := range []struct {
		header, label string
		single        *string
		multi         *[]string
	}{
		{"vocabulary-name", "VocabularyName", &p.vocabulary, nil},
		{"vocabulary-filter-name", "VocabularyFilterName", &p.filter, nil},
		{"language-model-name", "LanguageModelName", &p.languageModel, nil},
		{"vocabulary-names", "VocabularyNames", nil, &p.vocabularies},
		{"vocabulary-filter-names", "VocabularyFilterNames", nil, &p.filters},
	} {
		raw := get(item.header)
		if raw == "" {
			continue
		}
		if item.single != nil {
			if !tssNamePattern.MatchString(raw) {
				return p, item.label + " isn't valid"
			}
			*item.single = raw
			p.echo[item.header] = raw
			continue
		}
		if !p.identify {
			return p, item.label + " is only used with language identification"
		}
		for _, n := range strings.Split(raw, ",") {
			n = strings.TrimSpace(n)
			if !tssNamePattern.MatchString(n) {
				return p, item.label + " isn't valid"
			}
			*item.multi = append(*item.multi, n)
		}
		p.echo[item.header] = strings.Join(*item.multi, ",")
	}
	if p.identify && (p.vocabulary != "" || p.filter != "") {
		return p, "with language identification use VocabularyNames and VocabularyFilterNames"
	}
	if m := get("vocabulary-filter-method"); m != "" {
		if p.filter == "" && len(p.filters) == 0 {
			return p, "VocabularyFilterMethod needs a vocabulary filter"
		}
		if m != "remove" && m != "mask" && m != "tag" {
			return p, "VocabularyFilterMethod must be remove, mask or tag"
		}
		p.echo["vocabulary-filter-method"] = m
	}
	return p, ""
}

// checkStreamingResources checks that named vocabularies, filters and language
// models exist in the batch Transcribe store and match the stream's language.
func (s *transcribeStore) checkStreamingResources(p tssParams) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	vocabularies := s.vocabularies
	if p.op == "StartMedicalStreamTranscription" {
		vocabularies = s.medicalVocabularies
	}
	for _, name := range append([]string{p.vocabulary}, p.vocabularies...) {
		if name == "" {
			continue
		}
		v, ok := vocabularies[name]
		if !ok {
			return "the custom vocabulary doesn't exist"
		}
		if v.LanguageCode != "" && !p.identify && v.LanguageCode != p.language {
			return "the custom vocabulary's language doesn't match the stream's"
		}
	}
	for _, name := range append([]string{p.filter}, p.filters...) {
		if name == "" {
			continue
		}
		if _, ok := s.vocabularyFilters[name]; !ok {
			return "the vocabulary filter doesn't exist"
		}
	}
	if p.languageModel != "" {
		if _, ok := s.languageModels[p.languageModel]; !ok {
			return "the custom language model doesn't exist"
		}
	}
	return ""
}

func respondTranscribeStreamingJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-amzn-request-id", uuid.NewString())
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTranscribeStreamingError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTranscribeStreamingJSON(w, status, map[string]string{"Message": msg})
}
