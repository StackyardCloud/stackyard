package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bedrockAgentRegion    = "us-east-1"
	bedrockAgentAccount   = "123456789012"
	bedrockAgentDraft     = "DRAFT"
	bedrockAgentTestAlias = "TSTALIASID"
)

var (
	baIDPattern        = regexp.MustCompile(`^[0-9a-zA-Z]{10}$`)
	baNamePattern      = regexp.MustCompile(`^([0-9a-zA-Z][_-]?){1,100}$`)
	baNumericVersion   = regexp.MustCompile(`^[0-9]{1,5}$`)
	baInstructionIDPat = regexp.MustCompile(`^[a-z0-9]{10}$`)
)

// bedrockAgentStore keeps every bedrock-agent resource. Records are the API
// shapes themselves (map[string]any), so responses are the stored record and
// callers' optional members round-trip without a typed copy of each shape.
// ponytail: one mutex for the whole service; split per resource if contention shows.
type bedrockAgentStore struct {
	mu           sync.Mutex
	agents       map[string]*baAgent
	kbs          map[string]*baKB
	flows        map[string]*baVersioned
	prompts      map[string]*baVersioned
	instructions map[string]*baVersioned
	tags         map[string]map[string]string
	policies     map[string]map[string]any
}

// baAgent holds the working DRAFT plus numbered versions. Each version owns
// its action groups, knowledge-base associations and collaborators.
type baAgent struct {
	versions    map[string]*baAgentVersion
	aliases     map[string]map[string]any
	nextVersion int
}

type baAgentVersion struct {
	agent          map[string]any
	actionGroups   map[string]map[string]any
	knowledgeBases map[string]map[string]any
	collaborators  map[string]map[string]any
}

type baKB struct {
	kb          map[string]any
	dataSources map[string]*baDataSource
	vpcs        map[string]map[string]any
}

type baDataSource struct {
	ds   map[string]any
	jobs map[string]map[string]any
	docs map[string]map[string]any
}

// baVersioned is a resource with a DRAFT and immutable numbered versions
// (flows, prompts, retrieval instructions). Flows also have aliases.
type baVersioned struct {
	versions    map[string]map[string]any
	aliases     map[string]map[string]any
	nextVersion int
}

func newBedrockAgentStore() *bedrockAgentStore {
	return &bedrockAgentStore{
		agents:       map[string]*baAgent{},
		kbs:          map[string]*baKB{},
		flows:        map[string]*baVersioned{},
		prompts:      map[string]*baVersioned{},
		instructions: map[string]*baVersioned{},
		tags:         map[string]map[string]string{},
		policies:     map[string]map[string]any{},
	}
}

type baHandler func(st *bedrockAgentStore, in map[string]any, q url.Values) (any, *bedrockAgentError)

var bedrockAgentHandlers map[string]baHandler

func init() {
	bedrockAgentHandlers = map[string]baHandler{
		"TagResource":          (*bedrockAgentStore).tagResource,
		"UntagResource":        (*bedrockAgentStore).untagResource,
		"ListTagsForResource":  (*bedrockAgentStore).listTags,
		"PutResourcePolicy":    (*bedrockAgentStore).putPolicy,
		"GetResourcePolicy":    (*bedrockAgentStore).getPolicy,
		"DeleteResourcePolicy": (*bedrockAgentStore).deletePolicy,
	}
	for name, h := range bedrockAgentAgentHandlers() {
		bedrockAgentHandlers[name] = h
	}
	for name, h := range bedrockAgentKBHandlers() {
		bedrockAgentHandlers[name] = h
	}
	for name, h := range bedrockAgentFlowHandlers() {
		bedrockAgentHandlers[name] = h
	}
}

// handle runs one operation and returns the encoded response. Encoding
// happens under the lock because responses share maps with stored records.
func (st *bedrockAgentStore) handle(op string, in map[string]any, q url.Values) ([]byte, *bedrockAgentError) {
	h, ok := bedrockAgentHandlers[op]
	if !ok {
		return nil, &bedrockAgentError{status: 501, code: "NotImplementedException", msg: op + " is not implemented"}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in, q)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	body, mErr := json.Marshal(out)
	if mErr != nil {
		return nil, &bedrockAgentError{status: 500, code: "InternalServerException", msg: mErr.Error()}
	}
	return body, nil
}

// --- shared helpers ---

func baNow() string { return time.Now().UTC().Format(time.RFC3339Nano) }

const baIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// baNewID returns a random 10-character ID drawn from alphabet.
func baNewID(alphabet string, n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

func baARN(resource string) string {
	return "arn:aws:bedrock:" + bedrockAgentRegion + ":" + bedrockAgentAccount + ":" + resource
}

func baStr(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}

func baRequire(in map[string]any, keys ...string) *bedrockAgentError {
	for _, k := range keys {
		v, ok := in[k]
		if !ok || v == nil {
			return baValidation(k + " is required")
		}
		if s, isStr := v.(string); isStr && strings.TrimSpace(s) == "" {
			return baValidation(k + " must not be empty")
		}
		if l, isList := v.([]any); isList && len(l) == 0 {
			return baValidation(k + " must not be empty")
		}
	}
	return nil
}

func baCheckName(in map[string]any, key string) *bedrockAgentError {
	if err := baRequire(in, key); err != nil {
		return err
	}
	if !baNamePattern.MatchString(baStr(in, key)) {
		return baValidation(key + " must match ([0-9a-zA-Z][_-]?){1,100}")
	}
	return nil
}

func baCheckID(in map[string]any, key string) *bedrockAgentError {
	if !baIDPattern.MatchString(baStr(in, key)) {
		return baValidation(key + " must be 10 alphanumeric characters")
	}
	return nil
}

func baCheckEnum(in map[string]any, key string, allowed ...string) *bedrockAgentError {
	v, ok := in[key]
	if !ok || v == nil {
		return nil
	}
	s, _ := v.(string)
	for _, a := range allowed {
		if s == a {
			return nil
		}
	}
	return baValidation(fmt.Sprintf("%s must be one of %s", key, strings.Join(allowed, ", ")))
}

// baSubStr reads in[outer][inner] as a string.
func baSubStr(in map[string]any, outer, inner string) string {
	m, _ := in[outer].(map[string]any)
	return baStr(m, inner)
}

// baInt reads an integer member that may arrive as a JSON number or a query
// string. Every integer this API bounds fits in 32 bits, so it parses at that
// size and larger values are rejected as invalid.
func baInt(v any) (int, bool) {
	var s string
	switch n := v.(type) {
	case json.Number:
		s = n.String()
	case string:
		s = n
	case float64:
		if n != float64(int32(n)) {
			return 0, false
		}
		return int(int32(n)), true
	default:
		return 0, false
	}
	i, err := strconv.ParseInt(s, 10, 32)
	return int(i), err == nil
}

func baBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true"
	}
	return false
}

// baRecord copies the caller's members into a new record, dropping members that
// are request-only.
func baRecord(in map[string]any, drop ...string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	for _, k := range append(drop, "clientToken", "tags") {
		delete(out, k)
	}
	return out
}

// baMerge copies the members present in in over rec, skipping keys.
func baMerge(rec, in map[string]any, skip ...string) {
	for k, v := range in {
		if k == "clientToken" || k == "tags" || containsString(skip, k) {
			continue
		}
		rec[k] = v
	}
}

// baClone deep-copies a record so a snapshot (a version) can't change later.
func baClone(rec map[string]any) map[string]any {
	raw, _ := json.Marshal(rec)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	out := map[string]any{}
	_ = dec.Decode(&out)
	return out
}

func baCloneAll(in map[string]map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = baClone(v)
	}
	return out
}

// baProject returns only the named members of rec (summary shapes).
func baProject(rec map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := rec[k]; ok {
			out[k] = v
		}
	}
	return out
}

// baSorted returns records ordered by createdAt, then by key, so listings and
// page tokens are stable.
func baSorted(m map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ci, cj := baStr(m[keys[i]], "createdAt"), baStr(m[keys[j]], "createdAt")
		if ci != cj {
			return ci < cj
		}
		return keys[i] < keys[j]
	})
	out := make([]map[string]any, len(keys))
	for i, k := range keys {
		out[i] = m[k]
	}
	return out
}

// baPage applies maxResults/nextToken (from the body or the query string). The
// token is the offset of the next item.
func baPage(items []map[string]any, in map[string]any, q url.Values) ([]map[string]any, string, *bedrockAgentError) {
	limit := 1000
	if v := baQueryOrBody(in, q, "maxResults"); v != nil {
		n, ok := baInt(v)
		if !ok || n < 1 || n > 1000 {
			return nil, "", baValidation("maxResults must be between 1 and 1000")
		}
		limit = n
	}
	start := 0
	if v, _ := baQueryOrBody(in, q, "nextToken").(string); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", baValidation("invalid nextToken")
		}
		start = int(n)
	}
	end := len(items)
	if end-start > limit {
		end = start + limit
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func baList(key string, items []map[string]any, next string) map[string]any {
	out := map[string]any{key: items}
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

// --- tags and resource policies ---

// baResourceExists reports whether a taggable ARN names a live resource.
func (st *bedrockAgentStore) baResourceExists(arn string) bool {
	_, rest, ok := strings.Cut(arn, ":"+bedrockAgentAccount+":")
	if !ok {
		return false
	}
	kind, id, _ := strings.Cut(rest, "/")
	id, _, _ = strings.Cut(id, ":")
	switch kind {
	case "agent":
		_, ok = st.agents[id]
	case "agent-alias":
		agentID, aliasID, _ := strings.Cut(id, "/")
		if a := st.agents[agentID]; a != nil {
			_, ok = a.aliases[aliasID]
		} else {
			ok = false
		}
	case "knowledge-base":
		_, ok = st.kbs[id]
	case "flow":
		flowID, aliasPart, hasAlias := strings.Cut(id, "/alias/")
		f := st.flows[flowID]
		ok = f != nil
		if ok && hasAlias {
			_, ok = f.aliases[aliasPart]
		}
	case "prompt":
		_, ok = st.prompts[id]
	case "retrieval-instruction":
		_, ok = st.instructions[id]
	default:
		ok = false
	}
	return ok
}

func (st *bedrockAgentStore) setTags(arn string, tags any) {
	m, _ := tags.(map[string]any)
	if len(m) == 0 {
		return
	}
	if st.tags[arn] == nil {
		st.tags[arn] = map[string]string{}
	}
	for k, v := range m {
		st.tags[arn][k], _ = v.(string)
	}
}

func (st *bedrockAgentStore) tagResource(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	if err := baRequire(in, "tags"); err != nil {
		return nil, err
	}
	if _, ok := in["tags"].(map[string]any); !ok {
		return nil, baValidation("tags must be a map")
	}
	if !st.baResourceExists(arn) {
		return nil, baNotFound("resource", arn)
	}
	st.setTags(arn, in["tags"])
	return map[string]any{}, nil
}

func (st *bedrockAgentStore) untagResource(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	keys := q["tagKeys"]
	if len(keys) == 0 {
		return nil, baValidation("tagKeys is required")
	}
	if !st.baResourceExists(arn) {
		return nil, baNotFound("resource", arn)
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return map[string]any{}, nil
}

func (st *bedrockAgentStore) listTags(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	if !st.baResourceExists(arn) {
		return nil, baNotFound("resource", arn)
	}
	tags := map[string]string{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"tags": tags}, nil
}

func (st *bedrockAgentStore) putPolicy(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	if err := baRequire(in, "policy"); err != nil {
		return nil, err
	}
	if !json.Valid([]byte(baStr(in, "policy"))) {
		return nil, baValidation("policy must be a JSON document")
	}
	if !st.baResourceExists(arn) {
		return nil, baNotFound("resource", arn)
	}
	if err := st.checkRevision(arn, baQueryOrBody(in, q, "expectedRevisionId")); err != nil {
		return nil, err
	}
	rev := baNewID(baIDAlphabet, 12)
	st.policies[arn] = map[string]any{"resourceArn": arn, "policy": baStr(in, "policy"), "revisionId": rev}
	return map[string]any{"resourceArn": arn, "revisionId": rev}, nil
}

func (st *bedrockAgentStore) checkRevision(arn string, expected any) *bedrockAgentError {
	want, _ := expected.(string)
	if want == "" {
		return nil
	}
	if cur := st.policies[arn]; cur == nil || cur["revisionId"] != want {
		return baConflict("expectedRevisionId does not match the current policy revision")
	}
	return nil
}

func (st *bedrockAgentStore) getPolicy(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	p := st.policies[arn]
	if p == nil {
		return nil, baNotFound("resource policy for", arn)
	}
	return p, nil
}

func (st *bedrockAgentStore) deletePolicy(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	arn := baStr(in, "resourceArn")
	p := st.policies[arn]
	if p == nil {
		return nil, baNotFound("resource policy for", arn)
	}
	if err := st.checkRevision(arn, q.Get("expectedRevisionId")); err != nil {
		return nil, err
	}
	delete(st.policies, arn)
	return map[string]any{"resourceArn": arn, "revisionId": p["revisionId"]}, nil
}
