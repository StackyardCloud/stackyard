package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InvokeAgent and InvokeInlineAgent. The "model" is deterministic:
//  1. results for a pending returnControl are acknowledged;
//  2. else an enabled RETURN_CONTROL action group whose function (or group)
//     name appears in the input gets control back;
//  3. else the best knowledge-base passage answers, with a citation;
//  4. else the agent says what it received.
// ponytail: no model, no orchestration loop; enough to exercise the API shapes.

// barAgentDef is what an invocation needs from a stored or inline agent.
type barAgentDef struct {
	agentID, aliasID, version, name string
	actionGroups                    []map[string]any
	kbIDs                           []string
	memory                          bool
	idleTTL                         time.Duration
}

type barAgentSession struct {
	attrs, promptAttrs map[string]any
	turns              []string
	pendingInvocation  string
	started, lastUsed  time.Time
	ttl                time.Duration
}

func (s *Server) barInvokeAgent(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	in := req.in
	agentID, aliasID, sessionID := baStr(in, "agentId"), baStr(in, "agentAliasId"), baStr(in, "sessionId")
	if !barAgentIDPattern.MatchString(agentID) || !barAgentIDPattern.MatchString(aliasID) {
		return baValidation("agentId and agentAliasId must be 1-10 alphanumeric characters")
	}
	snap, ok := s.bedrockagent.LookupAgentAlias(agentID, aliasID)
	if !ok {
		return barNotFound("Agent " + agentID + " with alias " + aliasID + " was not found")
	}
	if status := baStr(snap.Agent, "agentStatus"); status != "PREPARED" {
		return baConflict("Agent " + agentID + " is " + status + "; prepare the agent before invoking it")
	}
	def := barAgentDef{
		agentID: agentID, aliasID: aliasID, version: baStr(snap.Agent, "agentVersion"), name: baStr(snap.Agent, "agentName"),
		actionGroups: barEnabled(snap.ActionGroups, "actionGroupState"),
		idleTTL:      barTTL(snap.Agent["idleSessionTTLInSeconds"]),
	}
	for _, kb := range barEnabled(snap.KnowledgeBases, "knowledgeBaseState") {
		def.kbIDs = append(def.kbIDs, baStr(kb, "knowledgeBaseId"))
	}
	for _, t := range barList(barMap(snap.Agent, "memoryConfiguration"), "enabledMemoryTypes") {
		def.memory = def.memory || t == "SESSION_SUMMARY"
	}
	return s.barRunAgent(w, req, def, agentID+"/"+aliasID+"/"+sessionID, barMap(in, "sessionState"))
}

func (s *Server) barInvokeInlineAgent(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	in := req.in
	if err := baRequire(in, "foundationModel", "instruction"); err != nil {
		return err
	}
	// The instruction has no documented maximum; the body cap bounds it.
	if err := barCheckLen("instruction", baStr(in, "instruction"), 40, barMaxBody); err != nil {
		return err
	}
	ttl, err := barCount(in["idleSessionTTLInSeconds"], "idleSessionTTLInSeconds", 600, 60, 3600)
	if err != nil {
		return err
	}
	def := barAgentDef{name: "inline-agent", idleTTL: time.Duration(ttl) * time.Second}
	if v := baStr(in, "agentName"); v != "" {
		if !baNamePattern.MatchString(v) {
			return baValidation("agentName must match ([0-9a-zA-Z][_-]?){1,100}")
		}
		def.name = v
	}
	for _, g := range barList(in, "actionGroups") {
		if m, ok := g.(map[string]any); ok {
			def.actionGroups = append(def.actionGroups, m)
		}
	}
	for _, k := range barList(in, "knowledgeBases") {
		m, _ := k.(map[string]any)
		id := baStr(m, "knowledgeBaseId")
		if _, _, ok := s.bedrockagent.LookupKnowledgeBase(id); !ok {
			return barNotFound("Knowledge base " + id + " was not found")
		}
		def.kbIDs = append(def.kbIDs, id)
	}
	return s.barRunAgent(w, req, def, "inline/"+baStr(in, "sessionId"), barMap(in, "inlineSessionState"))
}

func barEnabled(items []map[string]any, stateKey string) []map[string]any {
	var out []map[string]any
	for _, it := range items {
		if baStr(it, stateKey) != "DISABLED" {
			out = append(out, it)
		}
	}
	return out
}

func barTTL(v any) time.Duration {
	if n, ok := baInt(v); ok && n >= 60 && n <= 5400 {
		return time.Duration(n) * time.Second
	}
	return 10 * time.Minute
}

// barRunAgent is the shared invocation: validate, update the session, decide
// the answer and stream it.
func (s *Server) barRunAgent(w http.ResponseWriter, req *barRequest, def barAgentDef, key string, state map[string]any) *bedrockAgentError {
	in := req.in
	sessionID := baStr(in, "sessionId")
	if !barSessionIDPattern.MatchString(sessionID) {
		return baValidation("sessionId must match [0-9a-zA-Z._:-]{2,100}")
	}
	memoryID := baStr(in, "memoryId")
	if memoryID != "" && !barSessionIDPattern.MatchString(memoryID) {
		return baValidation("memoryId must match [0-9a-zA-Z._:-]{2,100}")
	}
	input := baStr(in, "inputText")
	if err := barCheckLen("inputText", input, 0, 25000000); err != nil {
		return err
	}
	results := barList(state, "returnControlInvocationResults")
	if strings.TrimSpace(input) == "" && len(results) == 0 {
		return baValidation("inputText is required unless sessionState.returnControlInvocationResults is set")
	}

	st := s.bedrockagentruntime
	st.mu.Lock()
	sess := st.agentSessionLocked(key, def.idleTTL)
	if len(results) > 0 && sess.pendingInvocation == "" {
		st.mu.Unlock()
		return baValidation("returnControlInvocationResults were sent, but no action is waiting for results in this session")
	}
	if id := baStr(state, "invocationId"); len(results) > 0 && id != "" && id != sess.pendingInvocation {
		st.mu.Unlock()
		return baValidation("sessionState.invocationId does not match the pending invocation " + sess.pendingInvocation)
	}
	for k, v := range barMap(state, "sessionAttributes") {
		sess.attrs[k] = v
	}
	for k, v := range barMap(state, "promptSessionAttributes") {
		sess.promptAttrs[k] = v
	}

	// The decision runs under the store lock so two calls on one session can't
	// both answer the same pending invocation. Retrieval takes only the
	// bedrock-agent store's lock, which never waits on this one.
	var answer string
	var hits []barHit
	var control map[string]any
	switch {
	case len(results) > 0:
		answer = barSummarizeResults(results)
	default:
		if control = barReturnControl(def, input); control == nil {
			for _, kbID := range def.kbIDs {
				found, err := s.barRetrieveHits(kbID, input, 3, nil)
				if err != nil {
					st.mu.Unlock()
					return err
				}
				hits = append(hits, found...)
			}
			if len(hits) > 0 {
				answer = hits[0].text
			} else {
				answer = def.name + " received your request: " + input
			}
		}
	}

	if control != nil {
		sess.pendingInvocation = baStr(control, "invocationId")
	} else {
		sess.pendingInvocation = ""
	}
	if input != "" {
		sess.turns = append(sess.turns, input)
		if len(sess.turns) > 50 {
			sess.turns = sess.turns[len(sess.turns)-50:]
		}
	}
	sess.lastUsed = barNowTime()
	if baBool(in["endSession"]) {
		if def.memory && memoryID != "" {
			st.rememberLocked(def, memoryID, sessionID, sess)
		}
		delete(st.agentSessions, key)
	}
	st.mu.Unlock()

	w.Header().Set("x-amzn-bedrock-agent-content-type", "application/json")
	w.Header().Set("x-amz-bedrock-agent-session-id", sessionID)
	if memoryID != "" {
		w.Header().Set("x-amz-bedrock-agent-memory-id", memoryID)
	}
	stream := brtStartStream(w)
	traceBase := map[string]any{"sessionId": sessionID, "eventTime": barNowTime().Format(time.RFC3339Nano)}
	if def.agentID != "" {
		traceBase["agentId"], traceBase["agentAliasId"], traceBase["agentVersion"] = def.agentID, def.aliasID, def.version
	}
	trace := func(t map[string]any) {
		if baBool(in["enableTrace"]) {
			ev := map[string]any{"trace": t}
			for k, v := range traceBase {
				ev[k] = v
			}
			stream.event("trace", ev)
		}
	}
	traceID := barNewUUID()
	trace(map[string]any{"orchestrationTrace": map[string]any{"rationale": map[string]any{"traceId": traceID, "text": barRationale(control, hits)}}})
	if control != nil {
		stream.event("returnControl", control)
		return nil
	}
	if len(hits) > 0 {
		refs := make([]map[string]any, len(hits))
		for i, h := range hits {
			refs[i] = barReference(h)
		}
		trace(map[string]any{"orchestrationTrace": map[string]any{"observation": map[string]any{
			"traceId": traceID, "type": "KNOWLEDGE_BASE", "knowledgeBaseLookupOutput": map[string]any{"retrievedReferences": refs},
		}}})
	}
	trace(map[string]any{"orchestrationTrace": map[string]any{"observation": map[string]any{
		"traceId": traceID, "type": "FINISH", "finalResponse": map[string]any{"text": answer},
	}}})
	chunk := map[string]any{"bytes": []byte(answer)}
	if len(hits) > 0 {
		chunk["attribution"] = map[string]any{"citations": barCitations(answer, hits)}
	}
	stream.event("chunk", chunk)
	return nil
}

func barRationale(control map[string]any, hits []barHit) string {
	switch {
	case control != nil:
		return "The request needs an action the caller performs, so control returns to the caller."
	case len(hits) > 0:
		return "The knowledge base has relevant passages, so the answer comes from them."
	}
	return "No tool or knowledge base applies, so the agent answers directly."
}

// agentSessionLocked returns the live session for key, starting a new one if
// it doesn't exist or has been idle past its TTL.
func (st *barStore) agentSessionLocked(key string, ttl time.Duration) *barAgentSession {
	now := barNowTime()
	if sess := st.agentSessions[key]; sess != nil && now.Sub(sess.lastUsed) <= sess.ttl {
		return sess
	}
	if len(st.agentSessions) >= barMaxAgentSessions {
		oldest := ""
		for k, s := range st.agentSessions {
			if oldest == "" || s.lastUsed.Before(st.agentSessions[oldest].lastUsed) {
				oldest = k
			}
		}
		delete(st.agentSessions, oldest)
	}
	sess := &barAgentSession{attrs: map[string]any{}, promptAttrs: map[string]any{}, started: now, lastUsed: now, ttl: ttl}
	st.agentSessions[key] = sess
	return sess
}

func barSummarizeResults(results []any) string {
	var parts []string
	for _, r := range results {
		m, _ := r.(map[string]any)
		res := barMap(m, "functionResult")
		name := baStr(res, "function")
		if res == nil {
			res = barMap(m, "apiResult")
			name = baStr(res, "httpMethod") + " " + baStr(res, "apiPath")
		}
		body := ""
		for _, content := range barMap(res, "responseBody") {
			if cm, ok := content.(map[string]any); ok {
				body = baStr(cm, "body")
			}
			break
		}
		parts = append(parts, strings.TrimSpace(baStr(res, "actionGroup")+" "+name+": "+body))
	}
	return "Action results received. " + strings.Join(parts, "; ")
}

// barReturnControl picks the first RETURN_CONTROL function (or API operation)
// whose name appears in the input, and fills parameters given as name=value.
func barReturnControl(def barAgentDef, input string) map[string]any {
	lower := strings.ToLower(input)
	for _, g := range def.actionGroups {
		if barPathStr(g, "actionGroupExecutor", "customControl") != "RETURN_CONTROL" {
			continue
		}
		group := baStr(g, "actionGroupName")
		for _, f := range barList(barMap(g, "functionSchema"), "functions") {
			fn, _ := f.(map[string]any)
			name := baStr(fn, "name")
			if name == "" || !barMentions(lower, name) {
				continue
			}
			params := []map[string]any{}
			for pname, p := range barMap(fn, "parameters") {
				pd, _ := p.(map[string]any)
				if v, ok := barParam(input, pname); ok {
					params = append(params, map[string]any{"name": pname, "type": baStr(pd, "type"), "value": v})
				}
			}
			call := map[string]any{"actionGroup": group, "function": name, "parameters": params, "actionInvocationType": "RESULT"}
			if def.agentID != "" {
				call["agentId"] = def.agentID
			}
			return map[string]any{"invocationId": barNewUUID(), "invocationInputs": []map[string]any{{"functionInvocationInput": call}}}
		}
		if path, method := barAPIOperation(g); path != "" && barMentions(lower, group) {
			call := map[string]any{"actionGroup": group, "apiPath": path, "httpMethod": method, "parameters": []map[string]any{}, "actionInvocationType": "RESULT"}
			if def.agentID != "" {
				call["agentId"] = def.agentID
			}
			return map[string]any{"invocationId": barNewUUID(), "invocationInputs": []map[string]any{{"apiInvocationInput": call}}}
		}
	}
	return nil
}

// barMentions reports whether a function name like get_weather or getWeather
// appears in the lowercased input, as written or as words.
func barMentions(lower, name string) bool {
	n := strings.ToLower(name)
	if strings.Contains(lower, n) {
		return true
	}
	words := strings.FieldsFunc(n, func(r rune) bool { return r == '_' || r == '-' })
	return len(words) > 1 && strings.Contains(lower, strings.Join(words, " "))
}

func barParam(input, name string) (string, bool) {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\s*[:=]\s*"?([^",\s]+)`)
	m := re.FindStringSubmatch(input)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// barAPIOperation returns the first path and method of an inline OpenAPI schema.
func barAPIOperation(g map[string]any) (string, string) {
	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if json.Unmarshal([]byte(barPathStr(g, "apiSchema", "payload")), &doc) != nil {
		return "", ""
	}
	best := ""
	for p := range doc.Paths {
		if best == "" || p < best {
			best = p
		}
	}
	for _, m := range []string{"get", "post", "put", "patch", "delete"} {
		if _, ok := doc.Paths[best][m]; ok {
			return best, strings.ToUpper(m)
		}
	}
	return "", ""
}

// --- agent memory ---

func (st *barStore) rememberLocked(def barAgentDef, memoryID, sessionID string, sess *barAgentSession) {
	key := def.agentID + "/" + def.aliasID + "/" + memoryID
	summary := "The user asked: " + strings.Join(sess.turns, " | ")
	if r := []rune(summary); len(r) > 1000 {
		summary = string(r[:1000])
	}
	list := append(st.memories[key], map[string]any{"sessionSummary": map[string]any{
		"memoryId": memoryID, "sessionId": sessionID, "summaryText": summary,
		"sessionStartTime": sess.started.Format(time.RFC3339Nano), "sessionExpiryTime": barNowTime().Add(30 * 24 * time.Hour).Format(time.RFC3339Nano),
	}})
	if len(list) > barMaxMemories {
		list = list[len(list)-barMaxMemories:]
	}
	st.memories[key] = list
}

func (s *Server) barMemoryAgent(in map[string]any) (string, *bedrockAgentError) {
	agentID, aliasID := baStr(in, "agentId"), baStr(in, "agentAliasId")
	if _, ok := s.bedrockagent.LookupAgentAlias(agentID, aliasID); !ok {
		return "", barNotFound("Agent " + agentID + " with alias " + aliasID + " was not found")
	}
	return agentID + "/" + aliasID + "/", nil
}

func (s *Server) barGetAgentMemory(req *barRequest) (any, *bedrockAgentError) {
	prefix, err := s.barMemoryAgent(req.in)
	if err != nil {
		return nil, err
	}
	memoryID := req.q.Get("memoryId")
	if !barSessionIDPattern.MatchString(memoryID) {
		return nil, baValidation("memoryId must match [0-9a-zA-Z._:-]{2,100}")
	}
	if req.q.Get("memoryType") != "SESSION_SUMMARY" {
		return nil, baValidation("memoryType must be SESSION_SUMMARY")
	}
	limit, err := barCount(barQueryValue(req.q, "maxItems"), "maxItems", 1000, 1, 1000)
	if err != nil {
		return nil, err
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	all := append([]map[string]any(nil), st.memories[prefix+memoryID]...)
	st.mu.Unlock()
	page, next, err := baPage(all, map[string]any{"maxResults": strconv.Itoa(limit)}, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("memoryContents", page, next), nil
}

func (s *Server) barDeleteAgentMemory(req *barRequest) (any, *bedrockAgentError) {
	prefix, err := s.barMemoryAgent(req.in)
	if err != nil {
		return nil, err
	}
	memoryID, sessionID := req.q.Get("memoryId"), req.q.Get("sessionId")
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	for key, list := range st.memories {
		if !strings.HasPrefix(key, prefix) || (memoryID != "" && key != prefix+memoryID) {
			continue
		}
		kept := list[:0]
		for _, m := range list {
			if sessionID != "" && barPathStr(m, "sessionSummary", "sessionId") != sessionID {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			delete(st.memories, key)
		} else {
			st.memories[key] = kept
		}
	}
	return map[string]any{}, nil
}

func barQueryValue(q map[string][]string, key string) any {
	if v := q[key]; len(v) > 0 && v[0] != "" {
		return v[0]
	}
	return nil
}
