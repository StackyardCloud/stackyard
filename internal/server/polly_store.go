package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	pollyAccount        = "123456789012"
	pollyRegion         = "us-east-1"
	pollyMaxLexicons    = 100
	pollyMaxLexiconSize = 4000 // characters
	pollyMaxLexemeLen   = 100
	pollyMaxTasks       = 1000
	pollyMaxVoices      = 10
	pollyMaxTags        = 50
	pollyPageSize       = 100
)

var pollyAlphabets = map[string]bool{
	"ipa": true, "x-sampa": true, "x-amazon-pinyin": true, "x-amazon-jyutping": true,
	"x-amazon-pron-kana": true, "x-amazon-yomigana": true,
}

type pollyLexicon struct {
	name, content, alphabet, lang string
	lexemes                       int
	modified                      time.Time
}

func (l *pollyLexicon) arn() string {
	return "arn:aws:polly:" + pollyRegion + ":" + pollyAccount + ":lexicon/" + l.name
}

func (l *pollyLexicon) attributes() map[string]any {
	return map[string]any{
		"Alphabet": l.alphabet, "LanguageCode": l.lang, "LastModified": float64(l.modified.Unix()),
		"LexiconArn": l.arn(), "LexemesCount": l.lexemes, "Size": len(l.content),
	}
}

type pollyTask struct {
	id, status, uri, sns, format, rate, textType, voice, engine, lang string
	lexicons, marks                                                   []string
	chars                                                             int
	created                                                           time.Time
}

func (t *pollyTask) describe() map[string]any {
	out := map[string]any{
		"TaskId": t.id, "TaskStatus": t.status, "OutputUri": t.uri, "CreationTime": float64(t.created.Unix()),
		"RequestCharacters": t.chars, "OutputFormat": t.format, "SampleRate": t.rate, "TextType": t.textType,
		"VoiceId": t.voice, "Engine": t.engine, "LexiconNames": pollyAnyList(t.lexicons),
		"SpeechMarkTypes": pollyAnyList(t.marks),
	}
	if t.sns != "" {
		out["SnsTopicArn"] = t.sns
	}
	if t.lang != "" {
		out["LanguageCode"] = t.lang
	}
	return out
}

type pollyCustomVoice struct {
	id, name, lang, status, token string
	recordings                    []any // Format + S3Url only; recording bytes aren't kept
	kms                           map[string]any
	created                       time.Time
}

func (v *pollyCustomVoice) arn() string {
	return "arn:aws:polly:" + pollyRegion + ":" + pollyAccount + ":voice/" + v.id
}

func (v *pollyCustomVoice) describe() map[string]any {
	out := map[string]any{
		"Arn": v.arn(), "Id": v.id, "Name": v.name, "LanguageCode": v.lang, "LanguageName": pollyLanguageNames[v.lang],
		"Origin": "customer", "Status": v.status, "SupportedEngines": []any{"generative"},
		"VoiceCreationData": map[string]any{"Recordings": append([]any{}, v.recordings...)},
		"CreationTime":      float64(v.created.Unix()),
	}
	if v.kms != nil {
		out["CustomerManagedEncryption"] = v.kms
	}
	return out
}

// pollyStore holds lexicons, synthesis tasks, custom voices and tags. Every
// method returns freshly built maps, so callers can shape them after unlock.
type pollyStore struct {
	mu        sync.Mutex
	lexicons  map[string]*pollyLexicon
	tasks     map[string]*pollyTask
	taskOrder []string // oldest first
	voices    map[string]*pollyCustomVoice
	tags      map[string]map[string]string // by ARN
}

func newPollyStore() *pollyStore {
	return &pollyStore{
		lexicons: map[string]*pollyLexicon{}, tasks: map[string]*pollyTask{},
		voices: map[string]*pollyCustomVoice{}, tags: map[string]map[string]string{},
	}
}

func pollyRandID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// pollyParseLexicon checks PLS content and returns its alphabet, language and
// lexeme count.
func pollyParseLexicon(content string) (alphabet, lang string, lexemes int, perr *pollyError) {
	if utf8.RuneCountInString(content) > pollyMaxLexiconSize {
		return "", "", 0, pollyBad("LexiconSizeExceededException", "A lexicon can be at most %d characters.", pollyMaxLexiconSize)
	}
	invalid := func(format string, args ...any) (string, string, int, *pollyError) {
		return "", "", 0, pollyBad("InvalidLexiconException", format, args...)
	}
	dec := xml.NewDecoder(strings.NewReader(content))
	depth, inGrapheme, grapheme := 0, false, 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return invalid("The lexicon isn't valid PLS: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				if t.Name.Local != "lexicon" {
					return invalid("The root element must be <lexicon>.")
				}
				for _, a := range t.Attr {
					switch {
					case a.Name.Local == "alphabet":
						alphabet = a.Value
					case a.Name.Local == "lang" && (a.Name.Space == "xml" || a.Name.Space == "http://www.w3.org/XML/1998/namespace"):
						lang = a.Value
					}
				}
			}
			switch t.Name.Local {
			case "lexeme":
				lexemes++
			case "grapheme", "alias":
				inGrapheme, grapheme = true, 0
			}
			depth++
		case xml.EndElement:
			depth--
			if t.Name.Local == "grapheme" || t.Name.Local == "alias" {
				inGrapheme = false
			}
		case xml.CharData:
			if inGrapheme {
				grapheme += utf8.RuneCount(t)
				if grapheme > pollyMaxLexemeLen {
					return "", "", 0, pollyBad("MaxLexemeLengthExceededException", "A lexeme can be at most %d characters.", pollyMaxLexemeLen)
				}
			}
		}
	}
	if depth != 0 || lang == "" && alphabet == "" && lexemes == 0 {
		return invalid("The lexicon isn't valid PLS.")
	}
	if !pollyAlphabets[alphabet] {
		return "", "", 0, pollyBad("UnsupportedPlsAlphabetException", "The alphabet %q isn't supported.", alphabet)
	}
	if pollyLanguageNames[lang] == "" {
		return "", "", 0, pollyBad("UnsupportedPlsLanguageException", "The language %q isn't supported.", lang)
	}
	return alphabet, lang, lexemes, nil
}

func (st *pollyStore) putLexicon(name, content string, tags map[string]any) (map[string]any, *pollyError) {
	alphabet, lang, lexemes, err := pollyParseLexicon(content)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.lexicons[name] == nil && len(st.lexicons) >= pollyMaxLexicons {
		return nil, pollyBad("MaxLexiconsNumberExceededException", "An account can have at most %d lexicons.", pollyMaxLexicons)
	}
	lex := &pollyLexicon{name: name, content: content, alphabet: alphabet, lang: lang, lexemes: lexemes, modified: time.Now().UTC()}
	if err := st.setTagsLocked(lex.arn(), tags); err != nil {
		return nil, err
	}
	st.lexicons[name] = lex
	return map[string]any{}, nil
}

func (st *pollyStore) getLexicon(name string) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	lex := st.lexicons[name]
	if lex == nil {
		return nil, pollyErr(http.StatusNotFound, "LexiconNotFoundException", "Lexicon not found: %s", name)
	}
	return map[string]any{
		"Lexicon":           map[string]any{"Name": lex.name, "Content": lex.content},
		"LexiconAttributes": lex.attributes(),
	}, nil
}

func (st *pollyStore) deleteLexicon(name string) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	lex := st.lexicons[name]
	if lex == nil {
		return nil, pollyErr(http.StatusNotFound, "LexiconNotFoundException", "Lexicon not found: %s", name)
	}
	delete(st.tags, lex.arn())
	delete(st.lexicons, name)
	return map[string]any{}, nil
}

func (st *pollyStore) listLexicons(start int) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := make([]string, 0, len(st.lexicons))
	for n := range st.lexicons {
		names = append(names, n)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, n := range names {
		items = append(items, map[string]any{"Name": n, "Attributes": st.lexicons[n].attributes()})
	}
	return pollyPage("Lexicons", items, start, pollyPageSize)
}

// missingLexicon returns the first name that isn't a stored lexicon.
func (st *pollyStore) missingLexicon(names []string) string {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, n := range names {
		if st.lexicons[n] == nil {
			return n
		}
	}
	return ""
}

// pollyStartTask renders the audio, writes it to the S3 store and records
// the task. Tasks report scheduled, then advance one step per read.
func (s *Server) pollyStartTask(r *http.Request, in map[string]any) (map[string]any, *pollyError) {
	bucket := pollyStr(in, "OutputS3BucketName")
	if _, err := s.s3.GetBucket(bucket); err != nil {
		return nil, pollyBad("InvalidS3BucketException", "The bucket %s doesn't exist.", bucket)
	}
	sns := pollyStr(in, "SnsTopicArn")
	if sns != "" {
		if _, err := s.sns.GetTopic(sns); err != nil {
			return nil, pollyBad("InvalidSnsTopicArnException", "The SNS topic %s doesn't exist.", sns)
		}
	}
	res, err := s.pollySynthesize(in, pollyTaskLimits)
	if err != nil {
		return nil, err
	}
	engine := pollyStr(in, "Engine")
	if engine == "" {
		engine = "standard"
	}
	rate, _ := pollySampleRate(res.format, engine, pollyStr(in, "SampleRate"))
	textType := pollyStr(in, "TextType")
	if textType == "" {
		textType = "text"
	}
	t := &pollyTask{
		id: pollyRandID(""), status: "scheduled", sns: sns, format: res.format, rate: strconv.Itoa(rate),
		textType: textType, voice: pollyStr(in, "VoiceId"), engine: engine, lang: pollyStr(in, "LanguageCode"),
		lexicons: pollyStrings(in, "LexiconNames"), marks: pollyStrings(in, "SpeechMarkTypes"),
		chars: res.chars, created: time.Now().UTC(),
	}
	key := pollyStr(in, "OutputS3KeyPrefix") + t.id + "." + pollyAudioExt(res.format)
	if !s.authorizeServiceS3(r, pollyServicePrincipal, bucket, key, "s3:PutObject") {
		return nil, pollyAccessDenied(bucket, key)
	}
	if _, err := s.s3.PutObject(bucket, key, pollyObjectType(res.format), res.audio, nil, "", ""); err != nil {
		return nil, pollyBad("InvalidS3KeyException", "The output object can't be written: %v", err)
	}
	t.uri = "https://s3." + pollyRegion + ".amazonaws.com/" + bucket + "/" + key
	// ponytail: SnsTopicArn is validated and recorded, but completion isn't
	// published; add a publish on the completed transition if tests need it.
	return map[string]any{"SynthesisTask": s.polly.addTask(t)}, nil
}

// pollyObjectType is the S3 content type for a task's output object.
func pollyObjectType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "ogg_vorbis", "ogg_opus":
		return "audio/ogg"
	case "pcm":
		return "audio/pcm"
	case "json":
		return "application/x-json-stream"
	}
	return "application/octet-stream"
}

func (st *pollyStore) addTask(t *pollyTask) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.taskOrder) >= pollyMaxTasks {
		delete(st.tasks, st.taskOrder[0])
		st.taskOrder = st.taskOrder[1:]
	}
	st.tasks[t.id] = t
	st.taskOrder = append(st.taskOrder, t.id)
	return t.describe()
}

// advance moves a task one step towards completed.
func (t *pollyTask) advance() {
	switch t.status {
	case "scheduled":
		t.status = "inProgress"
	case "inProgress":
		t.status = "completed"
	}
}

func (st *pollyStore) getTask(id string) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t := st.tasks[id]
	if t == nil {
		return nil, pollyBad("SynthesisTaskNotFoundException", "Synthesis task %s not found.", id)
	}
	t.advance()
	return map[string]any{"SynthesisTask": t.describe()}, nil
}

func (s *Server) pollyListTasks(q url.Values) (map[string]any, *pollyError) {
	size := pollyPageSize
	if raw := q.Get("MaxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return nil, pollyValidation("MaxResults must be between 1 and 100.")
		}
		size = int(n)
	}
	status := q.Get("Status")
	if status != "" {
		if err := pollyCheck("TaskStatus", "Status", status); err != nil {
			return nil, err
		}
	}
	start, err := pollyToken(q.Get("NextToken"))
	if err != nil {
		return nil, err
	}
	st := s.polly
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []any
	for i := len(st.taskOrder) - 1; i >= 0; i-- { // newest first
		t := st.tasks[st.taskOrder[i]]
		t.advance()
		if status == "" || t.status == status {
			items = append(items, t.describe())
		}
	}
	return pollyPage("SynthesisTasks", items, start, size)
}

// pollyCreateVoice records a custom voice. Recordings must name an S3 object
// that exists or carry inline bytes; only their format and S3 location are kept.
func (s *Server) pollyCreateVoice(req *http.Request, in map[string]any) (map[string]any, *pollyError) {
	if accepted, _ := in["TermsAccepted"].(bool); !accepted {
		return nil, pollyValidation("TermsAccepted must be true to create a voice.")
	}
	data, _ := in["VoiceCreationData"].(map[string]any)
	recs, _ := data["Recordings"].([]any)
	kept := make([]any, 0, len(recs))
	for i, r := range recs {
		rec, _ := r.(map[string]any)
		body, _ := rec["Data"].(map[string]any)
		s3url, hasURL := body["S3Url"].(string)
		_, hasBytes := body["Bytes"].(string)
		if hasURL == hasBytes {
			return nil, pollyValidation("Recordings[%d].Data must set exactly one of S3Url or Bytes.", i)
		}
		// Data is required, but Polly never returns recording bytes, so a
		// byte recording comes back with an empty Data.
		out := map[string]any{"Format": rec["Format"], "Data": map[string]any{}}
		if hasURL {
			bucket, key, _ := strings.Cut(strings.TrimPrefix(s3url, "s3://"), "/")
			if !s.authorizeServiceS3(req, pollyServicePrincipal, bucket, key, "s3:GetObject") {
				return nil, pollyAccessDenied(bucket, key)
			}
			if _, err := s.s3.GetObject(bucket, key); err != nil {
				return nil, pollyValidation("Recordings[%d]: the object %s doesn't exist.", i, s3url)
			}
			out["Data"] = map[string]any{"S3Url": s3url}
		}
		kept = append(kept, out)
	}
	tags, _ := in["Tags"].(map[string]any)
	kms, _ := in["CustomerManagedEncryption"].(map[string]any)
	v := &pollyCustomVoice{
		id: pollyRandID("cv-"), name: pollyStr(in, "Name"), lang: pollyStr(in, "LanguageCode"), status: "pending",
		token: pollyStr(in, "ClientToken"), recordings: kept, kms: kms, created: time.Now().UTC(),
	}
	return s.polly.addVoice(v, tags)
}

func (st *pollyStore) addVoice(v *pollyCustomVoice, tags map[string]any) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if v.token != "" {
		for _, existing := range st.voices {
			if existing.token == v.token {
				if existing.name != v.name {
					return nil, pollyErr(http.StatusConflict, "ConflictException", "ClientToken %s was already used for another request.", v.token)
				}
				return map[string]any{"Voice": existing.describe()}, nil
			}
		}
	}
	if len(st.voices) >= pollyMaxVoices {
		e := pollyErr(http.StatusPaymentRequired, "ServiceQuotaExceededException", "An account can have at most %d custom voices.", pollyMaxVoices)
		e.extra = map[string]any{"quotaCode": "custom-voice-max-count", "serviceCode": "polly"}
		return nil, e
	}
	if err := st.setTagsLocked(v.arn(), tags); err != nil {
		return nil, err
	}
	st.voices[v.id] = v
	return map[string]any{"Voice": v.describe()}, nil
}

// getVoice describes a custom voice (pending until first read) or a built-in one.
func (st *pollyStore) getVoice(id string) (map[string]any, *pollyError) {
	if b, ok := pollyVoiceByID[id]; ok {
		out := b.describe()
		out["Origin"], out["Status"] = "aws", "active"
		return map[string]any{"Voice": out}, nil
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v := st.voices[id]
	if v == nil {
		return nil, pollyNotFound("voice", id)
	}
	out := v.describe()
	v.status = "active"
	return map[string]any{"Voice": out}, nil
}

func (st *pollyStore) deleteVoice(id string) (map[string]any, *pollyError) {
	if _, ok := pollyVoiceByID[id]; ok {
		return nil, pollyValidation("Voice %s is provided by Amazon Polly and can't be deleted.", id)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v := st.voices[id]
	if v == nil {
		return nil, pollyNotFound("voice", id)
	}
	if v.status == "pending" {
		e := pollyErr(http.StatusConflict, "ConflictException", "Voice %s is still being created.", id)
		e.extra = map[string]any{"resourceId": id, "resourceType": "voice"}
		return nil, e
	}
	delete(st.tags, v.arn())
	delete(st.voices, id)
	return map[string]any{}, nil
}

// resourceExistsLocked reports whether arn is a lexicon or custom voice.
func (st *pollyStore) resourceExistsLocked(arn string) bool {
	prefix := "arn:aws:polly:" + pollyRegion + ":" + pollyAccount + ":"
	switch rest, ok := strings.CutPrefix(arn, prefix); {
	case !ok:
		return false
	case strings.HasPrefix(rest, "lexicon/"):
		return st.lexicons[strings.TrimPrefix(rest, "lexicon/")] != nil
	case strings.HasPrefix(rest, "voice/"):
		return st.voices[strings.TrimPrefix(rest, "voice/")] != nil
	}
	return false
}

func (st *pollyStore) setTagsLocked(arn string, add map[string]any) *pollyError {
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range add {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return pollyBad("InvalidTagParameterException", "Tag keys can't start with aws:.")
		}
		merged[k], _ = v.(string)
	}
	if len(merged) > pollyMaxTags {
		return pollyBad("TooManyTagsException", "A resource can have at most %d tags.", pollyMaxTags)
	}
	if len(merged) > 0 {
		st.tags[arn] = merged
	}
	return nil
}

func (st *pollyStore) listTags(arn string) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return nil, pollyNotFound("resource", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"Tags": tags}, nil
}

func (st *pollyStore) tagResource(arn string, tags map[string]any) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return nil, pollyNotFound("resource", arn)
	}
	if err := st.setTagsLocked(arn, tags); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *pollyStore) untagResource(arn string, keys []any) (map[string]any, *pollyError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return nil, pollyNotFound("resource", arn)
	}
	for _, k := range keys {
		if key, ok := k.(string); ok {
			delete(st.tags[arn], key)
		}
	}
	return map[string]any{}, nil
}
