package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// bedrockRuntimeStore keeps asynchronous invocations.
type bedrockRuntimeStore struct {
	mu      sync.Mutex
	next    int
	invokes []*brtAsync
}

type brtAsync struct {
	arn, modelArn, token, status, failure, s3URI, bodyKey string
	submit, end                                           time.Time
	output                                                map[string]any
}

func newBedrockRuntimeStore() *bedrockRuntimeStore { return &bedrockRuntimeStore{} }

func (a *brtAsync) summary() map[string]any {
	out := map[string]any{
		"invocationArn":    a.arn,
		"modelArn":         a.modelArn,
		"status":           a.status,
		"submitTime":       a.submit.Format(time.RFC3339Nano),
		"lastModifiedTime": a.end.Format(time.RFC3339Nano),
		"endTime":          a.end.Format(time.RFC3339Nano),
		"outputDataConfig": a.output,
	}
	if a.token != "" {
		out["clientRequestToken"] = a.token
	}
	if a.failure != "" {
		out["failureMessage"] = a.failure
	}
	return out
}

var brtNovaReelTasks = []string{"TEXT_VIDEO", "MULTI_SHOT_AUTOMATED", "MULTI_SHOT_MANUAL"}

func (s *Server) brtStartAsyncInvoke(w http.ResponseWriter, req *brtRequest) *brtError {
	p, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	model, modelArn, e := s.brtResolveModel(bedrockString(p, "modelId"))
	if e != nil {
		return e
	}
	if model.family != brtNovaReel {
		return brtValidation("The provided model doesn't support asynchronous invocation.")
	}
	input, ok := p["modelInput"].(map[string]any)
	if !ok {
		return brtConstraint(nil, "modelInput", "Member must not be null")
	}
	task := bedrockString(input, "taskType")
	if !brtIn(task, brtNovaReelTasks) {
		return brtValidation("Malformed input request: taskType must be one of %s, please reformat your input and try again.", strings.Join(brtNovaReelTasks, ", "))
	}
	if task == "TEXT_VIDEO" {
		params, _ := input["textToVideoParams"].(map[string]any)
		if bedrockString(params, "text") == "" {
			return brtValidation("Malformed input request: #/textToVideoParams: required key [text] not found, please reformat your input and try again.")
		}
	}
	out, _ := p["outputDataConfig"].(map[string]any)
	s3cfg, _ := out["s3OutputDataConfig"].(map[string]any)
	uri := strings.TrimSuffix(bedrockString(s3cfg, "s3Uri"), "/")
	bucket, prefix, _ := strings.Cut(strings.TrimPrefix(uri, "s3://"), "/")
	if !strings.HasPrefix(uri, "s3://") || bucket == "" {
		return brtConstraint(bedrockString(s3cfg, "s3Uri"), "outputDataConfig.s3OutputDataConfig.s3Uri", "Member must satisfy regular expression pattern: s3://...")
	}
	if tags, ok := p["tags"].([]any); ok {
		for i, raw := range tags {
			t, _ := raw.(map[string]any)
			if bedrockString(t, "key") == "" {
				return brtConstraint(nil, fmt.Sprintf("tags.%d.key", i), "Member must not be null")
			}
		}
	}
	token := bedrockString(p, "clientRequestToken")
	body, _ := json.Marshal(p)

	st := s.bedrockruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	if token != "" {
		for _, a := range st.invokes {
			if a.token == token {
				if a.bodyKey != string(body) {
					return &brtError{http.StatusConflict, "ConflictException", "The clientRequestToken was already used with different request parameters."}
				}
				respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"invocationArn": a.arn})
				return nil
			}
		}
	}
	st.next++
	id := fmt.Sprintf("%012d", st.next)
	now := time.Now().UTC()
	a := &brtAsync{
		arn: "arn:aws:bedrock:us-east-1:123456789012:async-invoke/" + id, modelArn: modelArn, token: token,
		s3URI: uri, bodyKey: string(body), submit: now, end: now, status: "Completed", output: map[string]any{"s3OutputDataConfig": s3cfg},
	}
	// ponytail: the "video" is a placeholder written straight to the emulator's
	// S3; the job completes at once.
	key := strings.TrimPrefix(prefix+"/"+id, "/")
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": "1.0", "invocationArn": a.arn, "outputs": []string{uri + "/" + id + "/output.mp4"}})
	if _, err := s.s3.PutObject(bucket, key+"/output.mp4", "video/mp4", []byte("stackyard placeholder video"), nil, "", ""); err != nil {
		a.status, a.failure = "Failed", fmt.Sprintf("Unable to write output to %s: the bucket does not exist or is not accessible.", uri)
	} else {
		_, _ = s.s3.PutObject(bucket, key+"/manifest.json", "application/json", manifest, nil, "", "")
	}
	st.invokes = append(st.invokes, a)
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"invocationArn": a.arn})
	return nil
}

func (s *Server) brtGetAsyncInvoke(w http.ResponseWriter, req *brtRequest) *brtError {
	arn := req.params["invocationArn"]
	st := s.bedrockruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, a := range st.invokes {
		if a.arn == arn {
			respondBedrockRuntimeJSON(w, http.StatusOK, a.summary())
			return nil
		}
	}
	return brtValidation("The invocation %s doesn't exist.", arn)
}

func (s *Server) brtListAsyncInvokes(w http.ResponseWriter, req *brtRequest) *brtError {
	q := req.r.URL.Query()
	status := q.Get("statusEquals")
	if status != "" && !brtIn(status, []string{"InProgress", "Completed", "Failed"}) {
		return brtConstraint(status, "statusEquals", "Member must satisfy enum value set: [InProgress, Completed, Failed]")
	}
	if v := q.Get("sortBy"); v != "" && v != "SubmissionTime" {
		return brtConstraint(v, "sortBy", "Member must satisfy enum value set: [SubmissionTime]")
	}
	order := q.Get("sortOrder")
	if order != "" && order != "Ascending" && order != "Descending" {
		return brtConstraint(order, "sortOrder", "Member must satisfy enum value set: [Ascending, Descending]")
	}
	limit := 1000
	if v := q.Get("maxResults"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			return brtConstraint(v, "maxResults", "Member must have value between 1 and 1000")
		}
		limit = n
	}
	var after, before time.Time
	for key, dst := range map[string]*time.Time{"submitTimeAfter": &after, "submitTimeBefore": &before} {
		if v := q.Get(key); v != "" {
			t, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return brtConstraint(v, key, "Member must be an ISO 8601 timestamp")
			}
			*dst = t
		}
	}
	start := 0
	if v := q.Get("nextToken"); v != "" {
		raw, err := base64.RawURLEncoding.DecodeString(v)
		n, convErr := strconv.Atoi(string(raw))
		if err != nil || convErr != nil || n < 0 {
			return brtValidation("The nextToken is invalid.")
		}
		start = n
	}

	st := s.bedrockruntime
	st.mu.Lock()
	var matches []map[string]any
	var times []time.Time
	for _, a := range st.invokes {
		if (status == "" || a.status == status) && (after.IsZero() || a.submit.After(after)) && (before.IsZero() || a.submit.Before(before)) {
			matches = append(matches, a.summary())
			times = append(times, a.submit)
		}
	}
	st.mu.Unlock()
	idx := make([]int, len(matches))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		if order == "Ascending" {
			return times[idx[i]].Before(times[idx[j]])
		}
		return times[idx[i]].After(times[idx[j]])
	})
	out := map[string]any{"asyncInvokeSummaries": []any{}}
	page := []any{}
	for i := start; i < len(idx) && len(page) < limit; i++ {
		page = append(page, matches[idx[i]])
	}
	out["asyncInvokeSummaries"] = page
	if end := start + len(page); end < len(idx) {
		out["nextToken"] = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	respondBedrockRuntimeJSON(w, http.StatusOK, out)
	return nil
}

// brtBidirectional answers an InvokeModelWithBidirectionalStream session.
// ponytail: the whole input stream is read before replying (HTTP/1.1); a real
// session interleaves input and output over HTTP/2.
func (s *Server) brtBidirectional(w http.ResponseWriter, req *brtRequest) *brtError {
	model, _, e := s.brtResolveModel(req.params["modelId"])
	if e != nil {
		return e
	}
	if model.family != brtNovaSonic {
		return brtValidation("The provided model doesn't support InvokeModelWithBidirectionalStream.")
	}
	frames, err := brtReadEvents(req.body)
	if err != nil {
		return brtValidation("The request body must be an event stream of chunk events.")
	}
	var events []map[string]any
	for i, f := range frames {
		if t := brtHeaderString(f, ":event-type"); t != "chunk" {
			return brtValidation("Input event %d has type %q; only chunk events are accepted.", i, t)
		}
		var part struct {
			Bytes string `json:"bytes"`
		}
		raw, decErr := []byte(nil), json.Unmarshal(f.Payload, &part)
		if decErr == nil {
			raw, decErr = base64.StdEncoding.DecodeString(part.Bytes)
		}
		var ev map[string]any
		if decErr != nil || json.Unmarshal(raw, &ev) != nil {
			return brtValidation("Input event %d is not a base64-encoded JSON chunk.", i)
		}
		inner, _ := ev["event"].(map[string]any)
		if len(inner) != 1 {
			return brtValidation("Input event %d must contain exactly one event.", i)
		}
		events = append(events, inner)
	}
	if _, ok := events[0]["sessionStart"]; !ok {
		return brtValidation("The first input event must be sessionStart.")
	}
	if _, ok := events[len(events)-1]["sessionEnd"]; !ok {
		return brtValidation("The last input event must be sessionEnd.")
	}
	prompt, promptName := "", ""
	for _, ev := range events {
		if ps, ok := ev["promptStart"].(map[string]any); ok {
			promptName = bedrockString(ps, "promptName")
		}
		if ti, ok := ev["textInput"].(map[string]any); ok {
			prompt += " " + bedrockString(ti, "content")
		}
	}
	reply := brtGenerate(prompt, 0, nil).text
	completion := brtID("", prompt, "completion")
	outEvents := []map[string]any{
		{"completionStart": map[string]any{"sessionId": brtID("", prompt), "promptName": promptName, "completionId": completion}},
		{"contentStart": map[string]any{"promptName": promptName, "completionId": completion, "contentId": completion + "-text", "type": "TEXT", "role": "ASSISTANT"}},
		{"textOutput": map[string]any{"promptName": promptName, "completionId": completion, "contentId": completion + "-text", "content": reply, "role": "ASSISTANT"}},
		{"contentEnd": map[string]any{"promptName": promptName, "completionId": completion, "contentId": completion + "-text", "stopReason": "END_TURN", "type": "TEXT"}},
		{"usageEvent": map[string]any{"promptName": promptName, "completionId": completion, "totalInputTokens": brtTokens(prompt), "totalOutputTokens": brtTokens(reply), "totalTokens": brtTokens(prompt) + brtTokens(reply)}},
		{"completionEnd": map[string]any{"promptName": promptName, "completionId": completion, "stopReason": "END_TURN"}},
	}
	st := brtStartStream(w)
	for _, ev := range outEvents {
		raw, _ := json.Marshal(map[string]any{"event": ev})
		st.event("chunk", map[string]any{"bytes": base64.StdEncoding.EncodeToString(raw)})
	}
	return nil
}
