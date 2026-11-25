package server

import (
	"crypto/sha256"
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

// ponytail: invocations are kept in memory and the oldest are dropped past
// this cap; a real deployment would expire them by age.
const brtMaxAsyncInvokes = 1000

const brtAccountID = "123456789012"

// bedrockRuntimeStore keeps asynchronous invocations in submission order,
// indexed by ARN and client token.
type bedrockRuntimeStore struct {
	mu      sync.Mutex
	next    int
	invokes []*brtAsync
	byARN   map[string]*brtAsync
	byToken map[string]*brtAsync
}

type brtAsync struct {
	arn, modelArn, token, status, failure string
	bodyHash                              [32]byte
	submit, end                           time.Time
	output                                map[string]any
}

func newBedrockRuntimeStore() *bedrockRuntimeStore {
	return &bedrockRuntimeStore{byARN: map[string]*brtAsync{}, byToken: map[string]*brtAsync{}}
}

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

// replayLocked answers a repeated clientRequestToken: the same invocation, or
// a conflict when the parameters differ.
func (st *bedrockRuntimeStore) replayLocked(w http.ResponseWriter, token string, hash [32]byte) (bool, *brtError) {
	a := st.byToken[token]
	if token == "" || a == nil {
		return false, nil
	}
	if a.bodyHash != hash {
		return true, &brtError{http.StatusConflict, "ConflictException", "The clientRequestToken was already used with different request parameters."}
	}
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"invocationArn": a.arn})
	return true, nil
}

func (st *bedrockRuntimeStore) insertLocked(a *brtAsync) {
	st.invokes = append(st.invokes, a)
	st.byARN[a.arn] = a
	if a.token != "" {
		st.byToken[a.token] = a
	}
	for len(st.invokes) > brtMaxAsyncInvokes {
		old := st.invokes[0]
		st.invokes = st.invokes[1:]
		delete(st.byARN, old.arn)
		if st.byToken[old.token] == old {
			delete(st.byToken, old.token)
		}
	}
}

var brtNovaReelTasks = []string{"TEXT_VIDEO", "MULTI_SHOT_AUTOMATED", "MULTI_SHOT_MANUAL"}

func (s *Server) brtStartAsyncInvoke(w http.ResponseWriter, req *brtRequest) *brtError {
	p, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	modelID := bedrockString(p, "modelId")
	model, modelArn, e := s.brtResolveModel(modelID)
	if e != nil {
		return e
	}
	if model.family != brtNovaReel {
		return brtValidation("The provided model doesn't support asynchronous invocation.")
	}
	region := cognitoIdentityRegionFromRequest(req.r)
	if !strings.HasPrefix(modelID, "arn:") {
		modelArn = fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/%s", region, model.id)
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
	hash := sha256.Sum256(body)

	st := s.bedrockruntime
	st.mu.Lock()
	if done, e := st.replayLocked(w, token, hash); done {
		st.mu.Unlock()
		return e
	}
	st.next++
	id := fmt.Sprintf("%012d", st.next)
	st.mu.Unlock()

	now := time.Now().UTC()
	a := &brtAsync{
		arn: fmt.Sprintf("arn:aws:bedrock:%s:%s:async-invoke/%s", region, brtAccountID, id), modelArn: modelArn, token: token,
		bodyHash: hash, submit: now, end: now, status: "Completed", output: map[string]any{"s3OutputDataConfig": s3cfg},
	}
	// ponytail: the "video" is a placeholder written straight to the emulator's
	// S3; the job completes at once.
	key := strings.TrimPrefix(prefix+"/"+id, "/")
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": "1.0", "invocationArn": a.arn, "outputs": []string{uri + "/" + id + "/output.mp4"}})
	if !s.authorizeServiceS3(req.r, "bedrock.amazonaws.com", bucket, key+"/output.mp4", "s3:PutObject") ||
		!s.authorizeServiceS3(req.r, "bedrock.amazonaws.com", bucket, key+"/manifest.json", "s3:PutObject") {
		a.status, a.failure = "Failed", fmt.Sprintf("Access denied writing output to %s.", uri)
	} else if _, err := s.s3.PutObject(bucket, key+"/output.mp4", "video/mp4", []byte("stackyard placeholder video"), nil, "", ""); err != nil {
		a.status, a.failure = "Failed", fmt.Sprintf("Unable to write output to %s: the bucket does not exist or is not accessible.", uri)
	} else {
		_, _ = s.s3.PutObject(bucket, key+"/manifest.json", "application/json", manifest, nil, "", "")
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	// A concurrent request with the same token may have won while S3 was
	// written; answer as its replay.
	if done, e := st.replayLocked(w, token, hash); done {
		return e
	}
	st.insertLocked(a)
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"invocationArn": a.arn})
	return nil
}

func (s *Server) brtGetAsyncInvoke(w http.ResponseWriter, req *brtRequest) *brtError {
	arn := req.params["invocationArn"]
	st := s.bedrockruntime
	st.mu.Lock()
	a := st.byARN[arn]
	var out map[string]any
	if a != nil {
		out = a.summary()
	}
	st.mu.Unlock()
	if a == nil {
		return brtValidation("The invocation %s doesn't exist.", arn)
	}
	respondBedrockRuntimeJSON(w, http.StatusOK, out)
	return nil
}

// brtCursor orders invocations by (submitTime, ARN).
type brtCursor struct {
	submit int64
	arn    string
}

func (c brtCursor) before(o brtCursor) bool {
	if c.submit != o.submit {
		return c.submit < o.submit
	}
	return c.arn < o.arn
}

func (c brtCursor) token() string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.submit, 10) + "|" + c.arn))
}

func brtParseCursor(token string) (brtCursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	nanos, arn, ok := strings.Cut(string(raw), "|")
	n, convErr := strconv.ParseInt(nanos, 10, 64)
	if err != nil || !ok || convErr != nil || arn == "" {
		return brtCursor{}, false
	}
	return brtCursor{n, arn}, true
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
	var cursor *brtCursor
	if v := q.Get("nextToken"); v != "" {
		c, ok := brtParseCursor(v)
		if !ok {
			return brtValidation("The nextToken is invalid.")
		}
		cursor = &c
	}
	asc := order == "Ascending"

	type entry struct {
		key     brtCursor
		summary map[string]any
	}
	st := s.bedrockruntime
	st.mu.Lock()
	var matches []entry
	for _, a := range st.invokes {
		if (status == "" || a.status == status) && (after.IsZero() || a.submit.After(after)) && (before.IsZero() || a.submit.Before(before)) {
			matches = append(matches, entry{brtCursor{a.submit.UnixNano(), a.arn}, a.summary()})
		}
	}
	st.mu.Unlock()
	sort.Slice(matches, func(i, j int) bool {
		if asc {
			return matches[i].key.before(matches[j].key)
		}
		return matches[j].key.before(matches[i].key)
	})
	page := []any{}
	var last brtCursor
	more := false
	for _, m := range matches {
		// Skip everything up to and including the cursor in this sort order.
		if cursor != nil && ((asc && !cursor.before(m.key)) || (!asc && !m.key.before(*cursor))) {
			continue
		}
		if len(page) == limit {
			more = true
			break
		}
		page = append(page, m.summary)
		last = m.key
	}
	out := map[string]any{"asyncInvokeSummaries": page}
	if more {
		out["nextToken"] = last.token()
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
