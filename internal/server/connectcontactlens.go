package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Connect Contact Lens: one REST JSON operation, signed "connect" like
// Amazon Connect itself, so this router runs before Connect's and claims only
// its own path. Real-time segments are analyzed from the text messages of the
// chat contacts Connect started (see connectparticipant_store.go): each message
// is a Transcript segment with lexicon sentiment and detected issues. No
// Contact Lens rules exist, so no Categories segments are produced.

const (
	cnlPath    = "/realtime-contact-analysis/analysis-segments"
	cnlMaxBody = 1 << 20
)

type cnlErr struct {
	status    int
	code, msg string
}

func (s *Server) handleConnectContactLensRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || rawRequestPath(r) != cnlPath {
		return false
	}
	hint := sigV4ServiceHint(r)
	if hint != "connect" && !(hint == "" && strings.HasPrefix(strings.ToLower(r.Host), "contact-lens.")) {
		return false
	}
	if r.ContentLength < 0 || r.ContentLength > cnlMaxBody {
		respondCnlError(w, &cnlErr{http.StatusBadRequest, "InvalidRequestException", "request body must declare a Content-Length of at most 1 MiB"})
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "connect"); !ok {
		respondCnlError(w, &cnlErr{status, code, msg})
		return true
	}
	model := awsmodels.ConnectContactLensModel()
	shapes := model.Operations["ListRealtimeContactAnalysisSegments"]
	raw, err := io.ReadAll(io.LimitReader(r.Body, cnlMaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondCnlError(w, &cnlErr{http.StatusBadRequest, "InvalidRequestException", "invalid JSON body"})
		return true
	}
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondCnlError(w, &cnlErr{http.StatusBadRequest, "InvalidRequestException", strings.Join(problems, "; ")})
		return true
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, cerr := s.cnlListSegments(conformed)
	if cerr != nil {
		respondCnlError(w, cerr)
		return true
	}
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondCnlError(w http.ResponseWriter, e *cnlErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func (s *Server) cnlListSegments(in map[string]any) (map[string]any, *cnlErr) {
	if f, _ := in["ListFilter"].(map[string]any); f != nil {
		if cf, _ := f["CategoriesAttributeFilter"].(map[string]any); len(cf) != 1 {
			return nil, &cnlErr{http.StatusBadRequest, "InvalidRequestException", "CategoriesAttributeFilter takes exactly one of OrConditions, AndCondition and TagCondition"}
		}
	}
	instanceID, contactID := cnpStr(in, "InstanceId"), cnpStr(in, "ContactId")
	// Connect's lock first, then the chat store's (taken inside contact()).
	s.connect.mu.Lock()
	known := s.connect.instances[instanceID] != nil
	var view cnpContactView
	var found bool
	if known {
		view, found = s.connectParticipant.contact(instanceID, contactID, true)
	}
	s.connect.mu.Unlock()
	if !known {
		return nil, &cnlErr{http.StatusNotFound, "ResourceNotFoundException", "instance " + instanceID + " not found"}
	}
	if !found {
		return nil, &cnlErr{http.StatusNotFound, "ResourceNotFoundException", "contact " + contactID + " not found"}
	}
	start := 0
	if tok := cnpStr(in, "NextToken"); tok != "" {
		v, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || v < 0 || int(v) > len(view.messages) {
			return nil, &cnlErr{http.StatusBadRequest, "InvalidRequestException", "NextToken is not valid"}
		}
		start = int(v)
	}
	limit := cnpInt(in, "MaxResults", 100)
	end := min(start+limit, len(view.messages))
	segments := make([]any, 0, end-start)
	for _, m := range view.messages[start:end] {
		segments = append(segments, map[string]any{"Transcript": cnlTranscript(m)})
	}
	out := map[string]any{"Segments": segments}
	if end < len(view.messages) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}

var (
	cnlPositive = []string{"thank", "great", "good", "happy", "love", "excellent", "perfect", "awesome", "appreciate", "resolved"}
	cnlNegative = []string{"bad", "angry", "terrible", "awful", "hate", "unhappy", "frustrated", "upset", "worst", "disappointed"}
	cnlIssues   = []string{"not working", "broken", "error", "problem", "issue", "can't", "cannot", "failed", "refund"}
)

// cnlTranscript analyzes one message: sentiment from a word lexicon, issues at
// the first occurrence of each issue phrase.
func cnlTranscript(m cnpMessageView) map[string]any {
	lower := strings.ToLower(m.content)
	score := 0
	for _, w := range cnlPositive {
		score += strings.Count(lower, w)
	}
	for _, w := range cnlNegative {
		score -= strings.Count(lower, w)
	}
	sentiment := "NEUTRAL"
	if score > 0 {
		sentiment = "POSITIVE"
	} else if score < 0 {
		sentiment = "NEGATIVE"
	}
	issues := []any{}
	for _, phrase := range cnlIssues {
		if i := strings.Index(lower, phrase); i >= 0 {
			begin := len([]rune(lower[:i]))
			issues = append(issues, map[string]any{"CharacterOffsets": map[string]any{
				"BeginOffsetChar": begin, "EndOffsetChar": begin + len([]rune(phrase)),
			}})
		}
	}
	beginMs := int(m.offset.Milliseconds())
	// A message's end offset assumes ~50 ms per character of speech-equivalent.
	endMs := beginMs + 50*len([]rune(m.content))
	return map[string]any{
		"Id": m.id, "ParticipantId": m.role, "ParticipantRole": m.role, "Content": m.content,
		"BeginOffsetMillis": beginMs, "EndOffsetMillis": endMs, "Sentiment": sentiment,
		"IssuesDetected": issues, "LanguageCode": "en-US",
	}
}
