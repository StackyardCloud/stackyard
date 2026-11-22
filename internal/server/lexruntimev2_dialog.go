package server

import (
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Lex Runtime V2 conversation state. Sessions live in the Lex V2 models store
// (same mutex), keyed by bot, alias, locale and session ID, and expire after
// the bot's idleSessionTTLInSeconds.

type lexRTSessionState struct {
	DialogAction         *lexRTDialogAction `json:"dialogAction,omitempty"`
	Intent               *lexRTIntent       `json:"intent,omitempty"`
	ActiveContexts       json.RawMessage    `json:"activeContexts,omitempty"`
	SessionAttributes    map[string]string  `json:"sessionAttributes,omitempty"`
	OriginatingRequestID string             `json:"originatingRequestId,omitempty"`
	RuntimeHints         json.RawMessage    `json:"runtimeHints,omitempty"`
}

type lexRTDialogAction struct {
	Type                 string          `json:"type"`
	SlotToElicit         string          `json:"slotToElicit,omitempty"`
	SlotElicitationStyle string          `json:"slotElicitationStyle,omitempty"`
	SubSlotToElicit      json.RawMessage `json:"subSlotToElicit,omitempty"`
}

type lexRTIntent struct {
	Name              string                `json:"name"`
	Slots             map[string]*lexRTSlot `json:"slots,omitempty"`
	State             string                `json:"state,omitempty"`
	ConfirmationState string                `json:"confirmationState,omitempty"`
}

type lexRTSlot struct {
	Value    *lexRTValue           `json:"value,omitempty"`
	Shape    string                `json:"shape,omitempty"`
	Values   []*lexRTSlot          `json:"values,omitempty"`
	SubSlots map[string]*lexRTSlot `json:"subSlots,omitempty"`
}

type lexRTValue struct {
	OriginalValue    string   `json:"originalValue,omitempty"`
	InterpretedValue string   `json:"interpretedValue"`
	ResolvedValues   []string `json:"resolvedValues,omitempty"`
}

type lexRTMessage struct {
	Content           string          `json:"content,omitempty"`
	ContentType       string          `json:"contentType"`
	ImageResponseCard json.RawMessage `json:"imageResponseCard,omitempty"`
}

type lexRTScore struct {
	Score float64 `json:"score"`
}

type lexRTInterpretation struct {
	NluConfidence        *lexRTScore  `json:"nluConfidence,omitempty"`
	Intent               *lexRTIntent `json:"intent,omitempty"`
	InterpretationSource string       `json:"interpretationSource,omitempty"`
}

type lexRuntimeSession struct {
	State           lexRTSessionState
	Messages        []lexRTMessage
	Interpretations []lexRTInterpretation
	LastActive      time.Time
}

// lexRTTurn is one recognition result, copied out of the store so it can be
// encoded after the lock is released.
type lexRTTurn struct {
	Messages        []lexRTMessage
	Interpretations []lexRTInterpretation
	State           lexRTSessionState
}

type lexRTFault struct {
	status    int
	code, msg string
}

type lexRTTarget struct {
	bot, alias, locale, session string
}

func (t lexRTTarget) key() string {
	return t.bot + "\x00" + t.alias + "\x00" + t.locale + "\x00" + t.session
}

const lexRTFallbackIntent = "FallbackIntent"

var (
	lexRTBotIDPattern     = regexp.MustCompile(`^[0-9a-zA-Z]{10}$`)
	lexRTSessionIDPattern = regexp.MustCompile(`^[0-9a-zA-Z._:-]{2,100}$`)
)

func (t lexRTTarget) validate() *lexRTFault {
	switch {
	case !lexRTBotIDPattern.MatchString(t.bot):
		return &lexRTFault{http.StatusBadRequest, "ValidationException", "botId must be 10 alphanumeric characters"}
	case t.alias == "":
		return &lexRTFault{http.StatusBadRequest, "ValidationException", "botAliasId is required"}
	case t.locale == "":
		return &lexRTFault{http.StatusBadRequest, "ValidationException", "localeId is required"}
	case !lexRTSessionIDPattern.MatchString(t.session):
		return &lexRTFault{http.StatusBadRequest, "ValidationException", "sessionId must be 2-100 characters of [0-9a-zA-Z._:-]"}
	}
	return nil
}

// runtimeLocaleLocked resolves the bot, alias and locale a runtime call targets.
func (s *lexV2Store) runtimeLocaleLocked(t lexRTTarget) (*lexV2Locale, time.Duration, *lexRTFault) {
	bot := s.bots[t.bot]
	if bot == nil {
		return nil, 0, &lexRTFault{http.StatusNotFound, "ResourceNotFoundException", "bot " + t.bot + " not found"}
	}
	if _, ok := s.aliases[t.bot][t.alias]; !ok && t.alias != lexV2TestAliasID {
		return nil, 0, &lexRTFault{http.StatusNotFound, "ResourceNotFoundException", "bot alias " + t.alias + " not found"}
	}
	locale := s.locales[t.bot][t.locale]
	if locale == nil {
		return nil, 0, &lexRTFault{http.StatusNotFound, "ResourceNotFoundException", "locale " + t.locale + " not found for bot " + t.bot}
	}
	ttl := 300 * time.Second
	if v, ok := bot["idleSessionTTLInSeconds"].(float64); ok && v > 0 {
		ttl = time.Duration(v * float64(time.Second))
	}
	return locale, ttl, nil
}

// sessionLocked returns the live session, or nil when it never existed or has
// been idle longer than the TTL.
func (s *lexV2Store) sessionLocked(t lexRTTarget, ttl time.Duration, now time.Time) *lexRuntimeSession {
	sess := s.sessions[t.key()]
	if sess != nil && now.Sub(sess.LastActive) > ttl {
		delete(s.sessions, t.key())
		return nil
	}
	return sess
}

func (s *lexV2Store) dropLexRuntimeSessionsLocked(botID string) {
	prefix := botID + "\x00"
	for key := range s.sessions {
		if strings.HasPrefix(key, prefix) {
			delete(s.sessions, key)
		}
	}
}

// Recognize runs one user turn. reqState, when given, replaces the stored
// session state first, as the API documents.
func (s *lexV2Store) Recognize(t lexRTTarget, text string, reqState *lexRTSessionState) (lexRTTurn, *lexRTFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	locale, ttl, fault := s.runtimeLocaleLocked(t)
	if fault != nil {
		return lexRTTurn{}, fault
	}
	now := time.Now()
	sess := s.sessionLocked(t, ttl, now)
	if sess == nil {
		sess = &lexRuntimeSession{}
		s.sessions[t.key()] = sess
	}
	if reqState != nil {
		sess.State = *reqState
	}
	sess.Messages, sess.Interpretations = converseLexRT(locale, &sess.State, text)
	sess.State.OriginatingRequestID = uuid.NewString()
	sess.LastActive = now
	return cloneLexRT(lexRTTurn{Messages: sess.Messages, Interpretations: sess.Interpretations, State: sess.State}), nil
}

// PutSession replaces the session state and the messages last sent to the user.
func (s *lexV2Store) PutSession(t lexRTTarget, state lexRTSessionState, messages []lexRTMessage) (lexRTTurn, *lexRTFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, fault := s.runtimeLocaleLocked(t); fault != nil {
		return lexRTTurn{}, fault
	}
	sess := &lexRuntimeSession{State: state, Messages: messages, LastActive: time.Now()}
	s.sessions[t.key()] = sess
	return cloneLexRT(lexRTTurn{Messages: sess.Messages, State: sess.State}), nil
}

func (s *lexV2Store) GetSession(t lexRTTarget) (lexRTTurn, *lexRTFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ttl, fault := s.runtimeLocaleLocked(t)
	if fault != nil {
		return lexRTTurn{}, fault
	}
	sess := s.sessionLocked(t, ttl, time.Now())
	if sess == nil {
		return lexRTTurn{}, &lexRTFault{http.StatusNotFound, "ResourceNotFoundException", "session " + t.session + " not found"}
	}
	return cloneLexRT(lexRTTurn{Messages: sess.Messages, Interpretations: sess.Interpretations, State: sess.State}), nil
}

func (s *lexV2Store) DeleteSession(t lexRTTarget) *lexRTFault {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ttl, fault := s.runtimeLocaleLocked(t)
	if fault != nil {
		return fault
	}
	if s.sessionLocked(t, ttl, time.Now()) == nil {
		return &lexRTFault{http.StatusNotFound, "ResourceNotFoundException", "session " + t.session + " not found"}
	}
	delete(s.sessions, t.key())
	return nil
}

// converseLexRT advances the dialog with one utterance: it fills the slot being
// elicited (unless the user switched intents), or else recognizes an intent, then elicits the next required
// slot or closes. Intents with every required slot filled end
// ReadyForFulfillment, which is what Lex reports when no fulfillment Lambda
// is attached.
func converseLexRT(locale *lexV2Locale, state *lexRTSessionState, text string) ([]lexRTMessage, []lexRTInterpretation) {
	ranked := rankLexRTIntents(locale, text)
	// While a slot is elicited the reply is its value, unless it is an exact
	// sample utterance of another intent: then the user switched intents.
	switched := len(ranked) > 0 && ranked[0].score == 1 && state.Intent != nil && ranked[0].intent.Name != state.Intent.Name
	if intent := elicitingLexRTIntent(locale, state); intent != nil && !switched {
		state.Intent.Slots[state.DialogAction.SlotToElicit] = lexRTSlotValue(text)
		messages := advanceLexRT(intent, state)
		return messages, []lexRTInterpretation{{NluConfidence: &lexRTScore{1}, Intent: cloneLexRT(state.Intent), InterpretationSource: "Lex"}}
	}

	var interps []lexRTInterpretation
	if len(ranked) > 0 && ranked[0].score >= locale.Threshold {
		best := ranked[0]
		state.Intent = &lexRTIntent{Name: best.intent.Name, Slots: map[string]*lexRTSlot{}, State: "InProgress", ConfirmationState: "None"}
		for _, id := range best.intent.SlotOrder {
			if slot := best.intent.Slots[id]; slot != nil {
				state.Intent.Slots[slot.Name] = nil
			}
		}
		for name, value := range best.slots {
			state.Intent.Slots[name] = lexRTSlotValue(value)
		}
		messages := advanceLexRT(best.intent, state)
		interps = append(interps, lexRTInterpretation{NluConfidence: &lexRTScore{best.score}, Intent: cloneLexRT(state.Intent), InterpretationSource: "Lex"})
		interps = appendLexRTAlternatives(interps, ranked[1:])
		interps = append(interps, lexRTInterpretation{Intent: &lexRTIntent{Name: lexRTFallbackIntent, Slots: map[string]*lexRTSlot{}}, InterpretationSource: "Lex"})
		return messages, interps
	}

	state.Intent = &lexRTIntent{Name: lexRTFallbackIntent, Slots: map[string]*lexRTSlot{}, State: "ReadyForFulfillment", ConfirmationState: "None"}
	state.DialogAction = &lexRTDialogAction{Type: "Close"}
	interps = append(interps, lexRTInterpretation{Intent: cloneLexRT(state.Intent), InterpretationSource: "Lex"})
	return nil, appendLexRTAlternatives(interps, ranked)
}

// elicitingLexRTIntent returns the model intent when the session is waiting
// for a slot value of a known intent.
func elicitingLexRTIntent(locale *lexV2Locale, state *lexRTSessionState) *lexV2Intent {
	if state.Intent == nil || state.DialogAction == nil || state.DialogAction.Type != "ElicitSlot" || state.DialogAction.SlotToElicit == "" {
		return nil
	}
	for _, intent := range locale.Intents {
		if intent.Name == state.Intent.Name {
			if state.Intent.Slots == nil {
				state.Intent.Slots = map[string]*lexRTSlot{}
			}
			return intent
		}
	}
	return nil
}

func advanceLexRT(intent *lexV2Intent, state *lexRTSessionState) []lexRTMessage {
	for _, id := range intent.SlotOrder {
		slot := intent.Slots[id]
		if slot == nil || !slot.Required || state.Intent.Slots[slot.Name] != nil {
			continue
		}
		state.Intent.State = "InProgress"
		state.DialogAction = &lexRTDialogAction{Type: "ElicitSlot", SlotToElicit: slot.Name}
		if slot.Prompt == "" {
			return nil
		}
		return []lexRTMessage{{Content: slot.Prompt, ContentType: "PlainText"}}
	}
	state.Intent.State = "ReadyForFulfillment"
	state.DialogAction = &lexRTDialogAction{Type: "Close"}
	return nil
}

func appendLexRTAlternatives(interps []lexRTInterpretation, ranked []lexRTMatch) []lexRTInterpretation {
	for _, m := range ranked {
		if len(interps) >= 5 {
			break
		}
		interps = append(interps, lexRTInterpretation{NluConfidence: &lexRTScore{m.score}, Intent: &lexRTIntent{Name: m.intent.Name, Slots: map[string]*lexRTSlot{}}, InterpretationSource: "Lex"})
	}
	return interps
}

func lexRTSlotValue(v string) *lexRTSlot {
	return &lexRTSlot{Shape: "Scalar", Value: &lexRTValue{OriginalValue: v, InterpretedValue: v, ResolvedValues: []string{v}}}
}

type lexRTMatch struct {
	intent *lexV2Intent
	score  float64
	slots  map[string]string
}

// rankLexRTIntents scores every intent against the text by its best sample
// utterance, best first. ponytail: exact pattern match or word overlap stands
// in for NLU; swap in a real classifier if tests need synonyms.
func rankLexRTIntents(locale *lexV2Locale, text string) []lexRTMatch {
	var out []lexRTMatch
	for _, intent := range locale.Intents {
		best := lexRTMatch{intent: intent}
		for _, u := range intent.Utterances {
			if score, slots := matchLexRTUtterance(u, text); score > best.score {
				best.score, best.slots = score, slots
			}
		}
		if best.score > 0 {
			out = append(out, best)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].intent.created < out[j].intent.created
	})
	return out
}

var lexRTSlotRef = regexp.MustCompile(`^\{([^{}]+)\}$`)

// matchLexRTUtterance scores text against a sample utterance such as
// "book a {RoomType} room". A full match (slot references match any words) is
// 1.0 and yields the slot values; otherwise the score is the share of the
// utterance's literal words that appear in the text.
func matchLexRTUtterance(utterance, text string) (float64, map[string]string) {
	words := lexRTWords(utterance)
	input := lexRTWords(text)
	if len(words) == 0 || len(input) == 0 {
		return 0, nil
	}
	var pattern strings.Builder
	pattern.WriteString(`(?i)^`)
	var slotNames []string
	var literals []string
	for i, w := range words {
		if i > 0 {
			pattern.WriteString(` `)
		}
		if m := lexRTSlotRef.FindStringSubmatch(w); m != nil {
			slotNames = append(slotNames, m[1])
			pattern.WriteString(`(.+?)`)
			continue
		}
		literals = append(literals, strings.ToLower(w))
		pattern.WriteString(regexp.QuoteMeta(w))
	}
	pattern.WriteString(`$`)
	if re, err := regexp.Compile(pattern.String()); err == nil {
		if m := re.FindStringSubmatch(strings.Join(input, " ")); m != nil {
			slots := map[string]string{}
			for i, name := range slotNames {
				slots[name] = m[i+1]
			}
			return 1, slots
		}
	}
	if len(literals) == 0 {
		return 0, nil
	}
	seen := map[string]bool{}
	for _, w := range input {
		seen[strings.ToLower(w)] = true
	}
	hits := 0
	for _, w := range literals {
		if seen[w] {
			hits++
		}
	}
	// Scale below 1 so only a full match reaches full confidence.
	return math.Round(float64(hits)/float64(len(literals))*0.9*100) / 100, nil
}

// lexRTWords splits on spaces and strips surrounding punctuation, keeping
// {SlotName} references whole.
func lexRTWords(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		if !lexRTSlotRef.MatchString(w) {
			w = strings.TrimFunc(w, func(r rune) bool { return unicode.IsPunct(r) && r != '{' && r != '}' })
		}
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// cloneLexRT deep-copies through JSON, so results handed out of the store
// never alias session state.
func cloneLexRT[T any](v T) T {
	var out T
	raw, err := json.Marshal(v)
	if err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

var (
	lexRTDialogActionTypes = map[string]bool{"Close": true, "ConfirmIntent": true, "Delegate": true, "ElicitIntent": true, "ElicitSlot": true, "None": true}
	lexRTIntentStates      = map[string]bool{"Failed": true, "Fulfilled": true, "InProgress": true, "ReadyForFulfillment": true, "Waiting": true, "FulfillmentInProgress": true}
	lexRTConfirmStates     = map[string]bool{"Confirmed": true, "Denied": true, "None": true}
	lexRTStyles            = map[string]bool{"Default": true, "SpellByLetter": true, "SpellByWord": true}
	lexRTMessageTypes      = map[string]bool{"CustomPayload": true, "ImageResponseCard": true, "PlainText": true, "SSML": true}
)

// validateLexRTState checks the enum and required members of a caller's
// sessionState; it returns "" when the state is acceptable.
func validateLexRTState(st *lexRTSessionState) string {
	if st == nil {
		return ""
	}
	if da := st.DialogAction; da != nil {
		if !lexRTDialogActionTypes[da.Type] {
			return "sessionState.dialogAction.type must be one of Close, ConfirmIntent, Delegate, ElicitIntent, ElicitSlot, None"
		}
		if da.SlotElicitationStyle != "" && !lexRTStyles[da.SlotElicitationStyle] {
			return "sessionState.dialogAction.slotElicitationStyle is invalid"
		}
		if da.Type == "ElicitSlot" && da.SlotToElicit == "" {
			return "sessionState.dialogAction.slotToElicit is required for ElicitSlot"
		}
	}
	if in := st.Intent; in != nil {
		if in.Name == "" {
			return "sessionState.intent.name is required"
		}
		if in.State != "" && !lexRTIntentStates[in.State] {
			return "sessionState.intent.state is invalid"
		}
		if in.ConfirmationState != "" && !lexRTConfirmStates[in.ConfirmationState] {
			return "sessionState.intent.confirmationState is invalid"
		}
		for name, slot := range in.Slots {
			if slot != nil && slot.Value != nil && slot.Value.InterpretedValue == "" {
				return "sessionState.intent.slots." + name + ".value.interpretedValue is required"
			}
		}
	}
	return ""
}

func validateLexRTMessages(messages []lexRTMessage) string {
	if len(messages) > 10 {
		return "messages may hold at most 10 items"
	}
	for _, m := range messages {
		if !lexRTMessageTypes[m.ContentType] {
			return "messages[].contentType must be one of CustomPayload, ImageResponseCard, PlainText, SSML"
		}
	}
	return ""
}
