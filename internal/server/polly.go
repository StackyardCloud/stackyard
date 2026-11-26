package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// pollyMaxBody bounds a request body. CreateVoice carries base64 recordings
// (up to 5 MiB each in the model), so it gets the most room.
const pollyMaxBody = 8 << 20

var pollyPathRoots = map[string]bool{
	"voices": true, "lexicons": true, "speech": true, "synthesisTasks": true, "synthesisStream": true,
	"listTagsForResource": true, "tagResource": true, "untagResource": true,
}

// pollyError is a Polly error with its HTTP status and modelled extras.
type pollyError struct {
	status int
	code   string
	msg    string
	extra  map[string]any
}

func pollyErr(status int, code, format string, args ...any) *pollyError {
	return &pollyError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func pollyBad(code, format string, args ...any) *pollyError {
	return pollyErr(http.StatusBadRequest, code, format, args...)
}

const pollyServicePrincipal = "polly.amazonaws.com"

func pollyAccessDenied(bucket, key string) *pollyError {
	return pollyErr(http.StatusForbidden, "AccessDeniedException", "Access denied for s3://%s/%s", bucket, key)
}

func pollyValidation(format string, args ...any) *pollyError {
	e := pollyBad("ValidationException", format, args...)
	e.extra = map[string]any{"reason": "fieldValidationFailed"}
	return e
}

func pollyNotFound(resourceType, id string) *pollyError {
	e := pollyErr(http.StatusNotFound, "ResourceNotFoundException", "%s %s does not exist.", resourceType, id)
	e.extra = map[string]any{"resourceId": id, "resourceType": resourceType}
	return e
}

func isPollyRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "polly" && !strings.HasPrefix(host, "polly.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	return len(seg) >= 2 && seg[0] == "v1" && pollyPathRoots[seg[1]]
}

func (s *Server) handlePollyRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPollyRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > pollyMaxBody {
		respondPollyError(w, pollyValidation("request body must declare a Content-Length of at most 8 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "polly"); !ok {
		respondPollyError(w, &pollyError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))[1:]
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondPollyError(w, pollyValidation("invalid path segment"))
			return true
		}
		seg[i] = u
	}
	if err := s.pollyRoute(w, r, seg); err != nil {
		respondPollyError(w, err)
	}
	return true
}

// pollyRoute dispatches one request. Handlers that answer with JSON return
// through respondPollyJSON; synthesis writes its own audio response.
func (s *Server) pollyRoute(w http.ResponseWriter, r *http.Request, seg []string) *pollyError {
	q := r.URL.Query()
	m, n := r.Method, len(seg)
	reply := func(op string) func(map[string]any, *pollyError) *pollyError {
		return func(out map[string]any, err *pollyError) *pollyError {
			if err != nil {
				return err
			}
			respondPollyJSON(w, out, op)
			return nil
		}
	}
	switch {
	case seg[0] == "speech" && n == 1 && m == http.MethodPost:
		in, err := pollyDecode(r, "SynthesizeSpeech")
		if err != nil {
			return err
		}
		res, err := s.pollySynthesize(in, pollySpeechLimits)
		if err != nil {
			return err
		}
		writePollyAudio(w, res.format, res.chars, res.audio)
		return nil
	case seg[0] == "synthesisStream" && n == 1 && m == http.MethodPost:
		return s.pollyStream(w, r)

	case seg[0] == "voices" && n == 1 && m == http.MethodGet:
		return reply("DescribeVoices")(s.pollyDescribeVoices(q))
	case seg[0] == "voices" && n == 1 && m == http.MethodPost:
		in, err := pollyDecode(r, "CreateVoice")
		if err != nil {
			return err
		}
		return reply("CreateVoice")(s.pollyCreateVoice(r, in))
	case seg[0] == "voices" && n == 2 && m == http.MethodGet:
		return reply("GetVoice")(s.polly.getVoice(seg[1]))
	case seg[0] == "voices" && n == 2 && m == http.MethodDelete:
		return reply("DeleteVoice")(s.polly.deleteVoice(seg[1]))

	case seg[0] == "lexicons" && n == 1 && m == http.MethodGet:
		start, err := pollyToken(q.Get("NextToken"))
		if err != nil {
			return err
		}
		return reply("ListLexicons")(s.polly.listLexicons(start))
	case seg[0] == "lexicons" && n == 2 && m == http.MethodPut:
		if err := pollyCheck("LexiconName", "Name", seg[1]); err != nil {
			return err
		}
		in, err := pollyDecode(r, "PutLexicon")
		if err != nil {
			return err
		}
		tags, _ := in["Tags"].(map[string]any)
		return reply("PutLexicon")(s.polly.putLexicon(seg[1], pollyStr(in, "Content"), tags))
	case seg[0] == "lexicons" && n == 2 && m == http.MethodGet:
		return reply("GetLexicon")(s.polly.getLexicon(seg[1]))
	case seg[0] == "lexicons" && n == 2 && m == http.MethodDelete:
		return reply("DeleteLexicon")(s.polly.deleteLexicon(seg[1]))

	case seg[0] == "synthesisTasks" && n == 1 && m == http.MethodPost:
		in, err := pollyDecode(r, "StartSpeechSynthesisTask")
		if err != nil {
			return err
		}
		return reply("StartSpeechSynthesisTask")(s.pollyStartTask(r, in))
	case seg[0] == "synthesisTasks" && n == 1 && m == http.MethodGet:
		return reply("ListSpeechSynthesisTasks")(s.pollyListTasks(q))
	case seg[0] == "synthesisTasks" && n == 2 && m == http.MethodGet:
		if err := pollyCheck("TaskId", "TaskId", seg[1]); err != nil {
			return pollyBad("InvalidTaskIdException", "The provided Task ID is not valid.")
		}
		return reply("GetSpeechSynthesisTask")(s.polly.getTask(seg[1]))

	case seg[0] == "listTagsForResource" && m == http.MethodGet:
		arn := q.Get("arn")
		if arn == "" {
			return pollyValidation("ResourceArn is required.")
		}
		return reply("ListTagsForResource")(s.polly.listTags(arn))
	case seg[0] == "tagResource" && n == 1 && m == http.MethodPost:
		in, err := pollyDecode(r, "TagResource")
		if err != nil {
			return err
		}
		tags, _ := in["Tags"].(map[string]any)
		return reply("TagResource")(s.polly.tagResource(pollyStr(in, "ResourceArn"), tags))
	case seg[0] == "untagResource" && n == 1 && m == http.MethodPost:
		in, err := pollyDecode(r, "UntagResource")
		if err != nil {
			return err
		}
		keys, _ := in["TagKeys"].([]any)
		return reply("UntagResource")(s.polly.untagResource(pollyStr(in, "ResourceArn"), keys))
	}
	return pollyErr(http.StatusNotFound, "UnknownOperationException", "no Amazon Polly operation matches %s %s", m, "/v1/"+strings.Join(seg, "/"))
}

// pollyDecode reads a JSON body and validates it against the operation's
// input shape (URI, query and header members are skipped by the validator).
func pollyDecode(r *http.Request, op string) (map[string]any, *pollyError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, pollyValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, pollyValidation("request body is not a JSON object")
		}
	}
	m := awsmodels.PollyModel()
	if problems := m.Validate(m.Operations[op].Input, in); len(problems) > 0 {
		return nil, pollyValidation("%s", strings.Join(problems, "; "))
	}
	return in, nil
}

// pollyCheck validates one URI, query or header value against its shape.
func pollyCheck(shape, field, v string) *pollyError {
	if problems := awsmodels.PollyModel().Validate(shape, v); len(problems) > 0 {
		_, reason, _ := strings.Cut(problems[0], ": ")
		return pollyValidation("%s %s.", field, reason)
	}
	return nil
}

// pollyToken parses an offset page token.
func pollyToken(raw string) (int, *pollyError) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return 0, pollyBad("InvalidNextTokenException", "The NextToken is invalid.")
	}
	return int(n), nil
}

func pollyStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func pollyStrings(in map[string]any, key string) []string {
	items, _ := in[key].([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if v, ok := it.(string); ok {
			out = append(out, v)
		}
	}
	return out
}

func respondPollyJSON(w http.ResponseWriter, body map[string]any, op string) {
	m := awsmodels.PollyModel()
	if shape := m.Operations[op].Output; shape != "" && body != nil {
		body, _ = m.Conform(shape, body).(map[string]any)
	}
	if body == nil {
		body = map[string]any{}
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func respondPollyError(w http.ResponseWriter, e *pollyError) {
	body := map[string]any{"message": e.msg}
	for k, v := range e.extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
