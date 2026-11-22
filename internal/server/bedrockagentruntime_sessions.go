package server

import (
	"regexp"
	"strings"
)

// Bedrock sessions (CreateSession ... ListInvocationSteps): a session holds
// invocations, and invocations hold steps. A session must be ENDED before it
// can be deleted, and only ACTIVE sessions accept new invocations and steps.

var barTagKey = regexp.MustCompile(`^[a-zA-Z0-9\s._:/=+@-]{1,128}$`)
var barTagValue = regexp.MustCompile(`^[a-zA-Z0-9\s._:/=+@-]{0,256}$`)

type barSession struct {
	rec         map[string]any
	tags        map[string]string
	invocations map[string]map[string]any
	steps       map[string]map[string]any
}

func barSessionKey(identifier string) string {
	if m := barSessionARN.FindStringSubmatch(identifier); m != nil {
		return m[2]
	}
	return identifier
}

func (st *barStore) sessionLocked(in map[string]any) (*barSession, *bedrockAgentError) {
	id := barSessionKey(baStr(in, "sessionIdentifier"))
	if !barUUIDPattern.MatchString(id) {
		return nil, baValidation("sessionIdentifier must be a session ID or ARN")
	}
	sess := st.sessions[id]
	if sess == nil {
		return nil, barNotFound("Session " + id + " was not found")
	}
	return sess, nil
}

func barSessionSummary(sess *barSession) map[string]any {
	return baProject(sess.rec, "sessionId", "sessionArn", "sessionStatus", "createdAt", "lastUpdatedAt")
}

func barCheckMetadata(v any) *bedrockAgentError {
	if m, ok := v.(map[string]any); ok && len(m) > 50 {
		return baValidation("sessionMetadata can have at most 50 entries")
	}
	return nil
}

func barCheckTags(v any) (map[string]string, *bedrockAgentError) {
	m, _ := v.(map[string]any)
	out := make(map[string]string, len(m))
	for k, raw := range m {
		val, _ := raw.(string)
		if !barTagKey.MatchString(k) || !barTagValue.MatchString(val) {
			return nil, baValidation("invalid tag " + k)
		}
		out[k] = val
	}
	return out, nil
}

func (s *Server) barCreateSession(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	if err := barCheckMetadata(in["sessionMetadata"]); err != nil {
		return nil, err
	}
	tags, err := barCheckTags(in["tags"])
	if err != nil {
		return nil, err
	}
	if len(tags) > 200 {
		return nil, baValidation("a session can have at most 200 tags")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.sessions) >= barMaxSessions {
		return nil, barQuota("the account has reached the maximum number of sessions")
	}
	id, now := barNewUUID(), baNow()
	rec := map[string]any{"sessionId": id, "sessionArn": baARN("session/" + id), "sessionStatus": "ACTIVE", "createdAt": now, "lastUpdatedAt": now}
	if m := barMap(in, "sessionMetadata"); m != nil {
		rec["sessionMetadata"] = m
	}
	if k := baStr(in, "encryptionKeyArn"); k != "" {
		rec["encryptionKeyArn"] = k
	}
	st.sessions[id] = &barSession{rec: rec, tags: tags, invocations: map[string]map[string]any{}, steps: map[string]map[string]any{}}
	return baProject(rec, "sessionId", "sessionArn", "sessionStatus", "createdAt"), nil
}

func (s *Server) barGetSession(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.sessionLocked(req.in)
	if err != nil {
		return nil, err
	}
	return baClone(sess.rec), nil
}

func (s *Server) barUpdateSession(req *barRequest) (any, *bedrockAgentError) {
	if err := barCheckMetadata(req.in["sessionMetadata"]); err != nil {
		return nil, err
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.sessionLocked(req.in)
	if err != nil {
		return nil, err
	}
	if sess.rec["sessionStatus"] != "ACTIVE" {
		return nil, baConflict("only an ACTIVE session can be updated")
	}
	if m, ok := req.in["sessionMetadata"].(map[string]any); ok {
		sess.rec["sessionMetadata"] = m
	}
	sess.rec["lastUpdatedAt"] = baNow()
	return barSessionSummary(sess), nil
}

func (s *Server) barEndSession(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.sessionLocked(req.in)
	if err != nil {
		return nil, err
	}
	if sess.rec["sessionStatus"] != "ACTIVE" {
		return nil, baConflict("the session is already " + baStr(sess.rec, "sessionStatus"))
	}
	sess.rec["sessionStatus"], sess.rec["lastUpdatedAt"] = "ENDED", baNow()
	return baProject(sess.rec, "sessionId", "sessionArn", "sessionStatus"), nil
}

func (s *Server) barDeleteSession(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.sessionLocked(req.in)
	if err != nil {
		return nil, err
	}
	if sess.rec["sessionStatus"] == "ACTIVE" {
		return nil, baConflict("end the session before deleting it")
	}
	delete(st.sessions, baStr(sess.rec, "sessionId"))
	return map[string]any{}, nil
}

func (s *Server) barListSessions(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	all := make(map[string]map[string]any, len(st.sessions))
	for id, sess := range st.sessions {
		all[id] = barSessionSummary(sess)
	}
	st.mu.Unlock()
	page, next, err := baPage(baSorted(all), req.in, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("sessionSummaries", page, next), nil
}

// activeSessionLocked is sessionLocked plus the ACTIVE check writes need.
func (st *barStore) activeSessionLocked(in map[string]any) (*barSession, *bedrockAgentError) {
	sess, err := st.sessionLocked(in)
	if err != nil {
		return nil, err
	}
	if sess.rec["sessionStatus"] != "ACTIVE" {
		return nil, baConflict("the session is " + baStr(sess.rec, "sessionStatus") + "; only an ACTIVE session accepts changes")
	}
	return sess, nil
}

func (s *Server) barCreateInvocation(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	id := baStr(in, "invocationId")
	if id != "" && !barUUIDPattern.MatchString(id) {
		return nil, baValidation("invocationId must be a UUID")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.activeSessionLocked(in)
	if err != nil {
		return nil, err
	}
	if id == "" {
		id = barNewUUID()
	}
	if sess.invocations[id] != nil {
		return nil, baConflict("invocation " + id + " already exists")
	}
	if len(sess.invocations) >= barMaxSessionEntries {
		return nil, barQuota("the session has reached the maximum number of invocations")
	}
	rec := map[string]any{"invocationId": id, "sessionId": sess.rec["sessionId"], "createdAt": baNow()}
	if d := baStr(in, "description"); d != "" {
		rec["description"] = d
	}
	sess.invocations[id] = rec
	sess.rec["lastUpdatedAt"] = baNow()
	return baProject(rec, "invocationId", "sessionId", "createdAt"), nil
}

func (s *Server) barListInvocations(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	sess, err := st.sessionLocked(req.in)
	var items []map[string]any
	if err == nil {
		for _, rec := range baSorted(sess.invocations) {
			items = append(items, baProject(rec, "invocationId", "sessionId", "createdAt"))
		}
	}
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	page, next, err := baPage(items, req.in, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("invocationSummaries", page, next), nil
}

func (s *Server) barPutInvocationStep(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	if err := baRequire(in, "invocationIdentifier", "invocationStepTime", "payload"); err != nil {
		return nil, err
	}
	blocks := barList(barMap(in, "payload"), "contentBlocks")
	if len(blocks) == 0 || len(blocks) > 50 {
		return nil, baValidation("payload.contentBlocks must contain between 1 and 50 blocks")
	}
	for _, b := range blocks {
		m, _ := b.(map[string]any)
		if (m["text"] == nil) == (m["image"] == nil) {
			return nil, baValidation("each content block needs exactly one of text or image")
		}
	}
	stepID := baStr(in, "invocationStepId")
	if stepID != "" && !barUUIDPattern.MatchString(stepID) {
		return nil, baValidation("invocationStepId must be a UUID")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.activeSessionLocked(in)
	if err != nil {
		return nil, err
	}
	invocation := baStr(in, "invocationIdentifier")
	if sess.invocations[invocation] == nil {
		return nil, barNotFound("Invocation " + invocation + " was not found")
	}
	if stepID == "" {
		stepID = barNewUUID()
	}
	if old := sess.steps[stepID]; old != nil && old["invocationId"] != invocation {
		return nil, baConflict("step " + stepID + " belongs to another invocation")
	}
	if sess.steps[stepID] == nil && len(sess.steps) >= barMaxSessionEntries {
		return nil, barQuota("the session has reached the maximum number of invocation steps")
	}
	sess.steps[stepID] = map[string]any{
		"invocationId": invocation, "invocationStepId": stepID, "sessionId": sess.rec["sessionId"],
		"invocationStepTime": in["invocationStepTime"], "payload": in["payload"], "createdAt": baNow(),
	}
	sess.rec["lastUpdatedAt"] = baNow()
	return map[string]any{"invocationStepId": stepID}, nil
}

func (s *Server) barGetInvocationStep(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	if err := baRequire(in, "invocationIdentifier"); err != nil {
		return nil, err
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.sessionLocked(in)
	if err != nil {
		return nil, err
	}
	step := sess.steps[baStr(in, "invocationStepId")]
	if step == nil || step["invocationId"] != baStr(in, "invocationIdentifier") {
		return nil, barNotFound("Invocation step " + baStr(in, "invocationStepId") + " was not found")
	}
	return map[string]any{"invocationStep": baProject(step, "invocationId", "invocationStepId", "sessionId", "invocationStepTime", "payload")}, nil
}

func (s *Server) barListInvocationSteps(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	filter := baStr(in, "invocationIdentifier")
	st := s.bedrockagentruntime
	st.mu.Lock()
	sess, err := st.sessionLocked(in)
	var items []map[string]any
	if err == nil {
		if filter != "" && sess.invocations[filter] == nil {
			err = barNotFound("Invocation " + filter + " was not found")
		}
		for _, step := range baSorted(sess.steps) {
			if filter == "" || step["invocationId"] == filter {
				items = append(items, baProject(step, "invocationId", "invocationStepId", "sessionId", "invocationStepTime"))
			}
		}
	}
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	page, next, err := baPage(items, map[string]any{"maxResults": in["maxResults"]}, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("invocationStepSummaries", page, next), nil
}

// --- tags (sessions are the only taggable resource) ---

func (st *barStore) taggedLocked(in map[string]any) (*barSession, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	m := barSessionARN.FindStringSubmatch(arn)
	if m == nil {
		return nil, baValidation("resourceArn must be a Bedrock session ARN")
	}
	sess := st.sessions[m[2]]
	if sess == nil {
		return nil, barNotFound("Session " + m[2] + " was not found")
	}
	return sess, nil
}

func (s *Server) barTagResource(req *barRequest) (any, *bedrockAgentError) {
	tags, err := barCheckTags(req.in["tags"])
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, baValidation("tags must not be empty")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.taggedLocked(req.in)
	if err != nil {
		return nil, err
	}
	merged := len(sess.tags)
	for k := range tags {
		if _, ok := sess.tags[k]; !ok {
			merged++
		}
	}
	if merged > 200 {
		return nil, barQuota("a session can have at most 200 tags")
	}
	for k, v := range tags {
		sess.tags[k] = v
	}
	return map[string]any{}, nil
}

func (s *Server) barUntagResource(req *barRequest) (any, *bedrockAgentError) {
	keys := req.q["tagKeys"]
	if len(keys) == 0 {
		return nil, baValidation("tagKeys is required")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.taggedLocked(req.in)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		for _, part := range strings.Split(k, ",") {
			delete(sess.tags, part)
		}
	}
	return map[string]any{}, nil
}

func (s *Server) barListTags(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, err := st.taggedLocked(req.in)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(sess.tags))
	for k, v := range sess.tags {
		out[k] = v
	}
	return map[string]any{"tags": out}, nil
}
