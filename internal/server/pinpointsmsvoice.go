package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Pinpoint SMS and Voice (v1): REST JSON under /v1/sms-voice, signed
// sms-voice (shared with SMS Voice v2, which is JSON 1.0 with X-Amz-Target).
// Configuration sets with event destinations, and validated voice messages.
// ponytail: sent messages aren't kept; nothing in the v1 API reads them back.

const (
	ppsvMaxBody      = 1 << 20
	ppsvMaxSets      = 1000
	ppsvMaxDests     = 5
	ppsvMaxText      = 3000 // characters of voice text, as Amazon Polly accepts
	ppsvDefaultPage  = 100
	ppsvContentType  = "application/json"
	ppsvNotFoundCode = "NotFoundException"
)

var ppsvNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var ppsvPhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

type ppsvSet struct {
	created time.Time
	dests   map[string]map[string]any // conformed EventDestinationDefinition
}

type ppsvStore struct {
	mu    sync.Mutex
	model *awsmodels.Model
	sets  map[string]*ppsvSet
}

func newPinpointSMSVoiceStore() *ppsvStore {
	return &ppsvStore{model: awsmodels.PinpointSMSVoiceModel(), sets: map[string]*ppsvSet{}}
}

func (s *Server) handlePinpointSMSVoiceRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/v1/sms-voice/") || r.Header.Get("X-Amz-Target") != "" {
		return false
	}
	hint := sigV4ServiceHint(r)
	if hint != "sms-voice" && (hint != "" || !strings.HasPrefix(strings.ToLower(r.Host), "sms-voice.")) {
		return false
	}
	op, params, ok := matchSESV2OperationIn(pinpointSMSVoiceOperations, r.Method, r.URL.EscapedPath())
	if !ok {
		respondPPSVError(w, http.StatusNotFound, ppsvNotFoundCode, "no such operation")
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > ppsvMaxBody {
		respondPPSVError(w, http.StatusBadRequest, "BadRequestException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "sms-voice"); !ok {
		respondPPSVError(w, status, code, msg)
		return true
	}
	model := awsmodels.PinpointSMSVoiceModel()
	in, problem := awsModelDecodeREST(r, model, model.Operations[op.Name].Input, params)
	if problem != "" {
		respondPPSVError(w, http.StatusBadRequest, "BadRequestException", problem)
		return true
	}
	out, perr := s.pinpointSMSVoice.handle(op.Name, in)
	if perr != nil {
		respondPPSVError(w, perr.status, perr.code, perr.msg)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	body, _ := json.Marshal(model.Conform(model.Operations[op.Name].Output, out))
	w.Header().Set("Content-Type", ppsvContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondPPSVError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"Message": msg})
	w.Header().Set("Content-Type", ppsvContentType)
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func ppsvBadRequest(format string, args ...any) *ppeErr {
	return ppeError(http.StatusBadRequest, "BadRequestException", format, args...)
}

func ppsvNotFound(format string, args ...any) *ppeErr {
	return ppeError(http.StatusNotFound, ppsvNotFoundCode, format, args...)
}

func (st *ppsvStore) handle(op string, in map[string]any) (map[string]any, *ppeErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	name := mmgrStr(in, "ConfigurationSetName")
	switch op {
	case "CreateConfigurationSet":
		if !ppsvNamePattern.MatchString(name) {
			return nil, ppsvBadRequest("ConfigurationSetName must be 1-64 letters, digits, hyphens or underscores.")
		}
		if st.sets[name] != nil {
			return nil, ppeError(http.StatusConflict, "AlreadyExistsException", "Configuration set %s already exists.", name)
		}
		if len(st.sets) >= ppsvMaxSets {
			return nil, ppeError(http.StatusPreconditionFailed, "LimitExceededException", "Stackyard keeps at most %d configuration sets.", ppsvMaxSets)
		}
		st.sets[name] = &ppsvSet{created: time.Now().UTC(), dests: map[string]map[string]any{}}
		return nil, nil
	case "ListConfigurationSets":
		names := make([]string, 0, len(st.sets))
		for n := range st.sets {
			names = append(names, n)
		}
		sort.Strings(names)
		return ppsvPage(names, in)
	case "SendVoiceMessage":
		return st.send(in)
	}
	set := st.sets[name]
	if set == nil {
		return nil, ppsvNotFound("Configuration set %s doesn't exist.", name)
	}
	dest := mmgrStr(in, "EventDestinationName")
	switch op {
	case "DeleteConfigurationSet":
		delete(st.sets, name)
	case "GetConfigurationSetEventDestinations":
		names := make([]string, 0, len(set.dests))
		for n := range set.dests {
			names = append(names, n)
		}
		sort.Strings(names)
		out := make([]any, 0, len(names))
		for _, n := range names {
			entry := map[string]any{"Name": n}
			for k, v := range set.dests[n] {
				entry[k] = v
			}
			out = append(out, st.model.Conform("EventDestination", entry))
		}
		return map[string]any{"EventDestinations": out}, nil
	case "CreateConfigurationSetEventDestination", "UpdateConfigurationSetEventDestination":
		def := mmgrMap(in, "EventDestination")
		if def == nil {
			return nil, ppsvBadRequest("EventDestination is required.")
		}
		if def["CloudWatchLogsDestination"] == nil && def["KinesisFirehoseDestination"] == nil && def["SnsDestination"] == nil {
			return nil, ppsvBadRequest("An event destination needs a CloudWatchLogsDestination, KinesisFirehoseDestination or SnsDestination.")
		}
		_, exists := set.dests[dest]
		if op == "CreateConfigurationSetEventDestination" {
			switch {
			case !ppsvNamePattern.MatchString(dest):
				return nil, ppsvBadRequest("EventDestinationName must be 1-64 letters, digits, hyphens or underscores.")
			case exists:
				return nil, ppeError(http.StatusConflict, "AlreadyExistsException", "Event destination %s already exists.", dest)
			case len(set.dests) >= ppsvMaxDests:
				return nil, ppeError(http.StatusPreconditionFailed, "LimitExceededException", "A configuration set can have at most %d event destinations.", ppsvMaxDests)
			}
		} else if !exists {
			return nil, ppsvNotFound("Event destination %s doesn't exist.", dest)
		}
		set.dests[dest] = def
	case "DeleteConfigurationSetEventDestination":
		if _, ok := set.dests[dest]; !ok {
			return nil, ppsvNotFound("Event destination %s doesn't exist.", dest)
		}
		delete(set.dests, dest)
	}
	return nil, nil
}

func ppsvPage(names []string, in map[string]any) (map[string]any, *ppeErr) {
	size := 0
	if raw := mmgrStr(in, "PageSize"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if size = int(n); err != nil || size < 1 {
			return nil, ppsvBadRequest("PageSize must be a positive integer.")
		}
	}
	start, end, next, ok := mmgrPage(len(names), mmgrStr(in, "NextToken"), size, ppsvDefaultPage)
	if !ok {
		return nil, ppsvBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, n := range names[start:end] {
		items = append(items, n)
	}
	out := map[string]any{"ConfigurationSets": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// send validates a voice message; callers hold the lock.
func (st *ppsvStore) send(in map[string]any) (map[string]any, *ppeErr) {
	dest, orig := mmgrStr(in, "DestinationPhoneNumber"), mmgrStr(in, "OriginationPhoneNumber")
	if !ppsvPhonePattern.MatchString(dest) {
		return nil, ppsvBadRequest("DestinationPhoneNumber must be an E.164 phone number.")
	}
	if !ppsvPhonePattern.MatchString(orig) {
		return nil, ppsvBadRequest("OriginationPhoneNumber must be an E.164 phone number.")
	}
	if cfg := mmgrStr(in, "ConfigurationSetName"); cfg != "" && st.sets[cfg] == nil {
		return nil, ppsvNotFound("Configuration set %s doesn't exist.", cfg)
	}
	content := mmgrMap(in, "Content")
	if len(content) != 1 {
		return nil, ppsvBadRequest("Content sets exactly one of CallInstructionsMessage, PlainTextMessage and SSMLMessage.")
	}
	for kind, v := range content {
		text := mmgrStr(v.(map[string]any), "Text")
		switch {
		case strings.TrimSpace(text) == "":
			return nil, ppsvBadRequest("%s.Text is required.", kind)
		case len([]rune(text)) > ppsvMaxText:
			return nil, ppsvBadRequest("%s.Text can have at most %d characters.", kind, ppsvMaxText)
		case kind == "SSMLMessage" && !(strings.HasPrefix(strings.TrimSpace(text), "<speak") && strings.HasSuffix(strings.TrimSpace(text), "</speak>")):
			return nil, ppsvBadRequest("SSMLMessage.Text must be a <speak> document.")
		}
	}
	return map[string]any{"MessageId": uuid.NewString()}, nil
}
