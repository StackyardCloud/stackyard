package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Agents for Amazon Bedrock Runtime (bedrock-agent-runtime). It signs as
// "bedrock" like the Bedrock control plane, Bedrock Runtime and the
// bedrock-agent control plane, so requests are claimed only by matching one
// of this service's URI templates. It reads agents, knowledge bases, flows and
// prompts from the bedrock-agent store through its Lookup* methods.

const (
	// barMaxBody bounds a request body. InvokeAgent's inputText may be large,
	// but nothing here needs more than a few MiB.
	barMaxBody = 8 << 20
	// Per-store caps on caller-created state.
	barMaxSessions       = 1000
	barMaxSessionEntries = 1000 // invocations and steps per session
	barMaxAgentSessions  = 1000
	barMaxExecutions     = 1000
	barMaxDocTokens      = 1000
	barMaxMemories       = 100 // session summaries kept per memory ID
	barDocumentPath      = "/_stackyard/bedrockagentruntime/documents/"
)

var (
	barAgentIDPattern   = regexp.MustCompile(`^[0-9a-zA-Z]{1,10}$`)
	barSessionIDPattern = regexp.MustCompile(`^[0-9a-zA-Z._:-]{2,100}$`)
	barUUIDPattern      = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	barSessionARN       = regexp.MustCompile(`^arn:aws(-[^:]+)?:bedrock:[a-z0-9-]+:[0-9]{12}:session/([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`)
	barExecutionID      = regexp.MustCompile(`^(?:arn:aws:bedrock:[a-z0-9-]{1,20}:[0-9]{12}:flow/[0-9a-zA-Z]{10}/alias/[0-9a-zA-Z]{10}/execution/)?([a-zA-Z0-9-]{1,36})$`)
	barPathPrefixes     = []string{"/agents/", "/knowledgebases/", "/flows/", "/sessions/", "/tags/", "/retrieveAndGenerate/", "/retrieveAndGenerateStream/", "/rerank/", "/generateQuery/", "/optimize-prompt/", "/agenticRetrieveStream/"}
)

// barStore keeps runtime state: conversation sessions for agents, Bedrock
// sessions (CreateSession and friends), RetrieveAndGenerate sessions, agent
// memory, flow executions and document-content links.
// ponytail: one mutex for the whole service, like the bedrock-agent store.
type barStore struct {
	mu            sync.Mutex
	agentSessions map[string]*barAgentSession
	memories      map[string][]map[string]any // agentId/aliasId/memoryId -> summaries
	ragSessions   map[string]barRAGSession
	sessions      map[string]*barSession
	executions    map[string]*barExecution // flowId/executionId
	docTokens     map[string]barDocToken
}

func newBARStore() *barStore {
	return &barStore{
		agentSessions: map[string]*barAgentSession{},
		memories:      map[string][]map[string]any{},
		ragSessions:   map[string]barRAGSession{},
		sessions:      map[string]*barSession{},
		executions:    map[string]*barExecution{},
		docTokens:     map[string]barDocToken{},
	}
}

// barRequest is one routed call: the decoded body with path parameters merged
// in, plus the query string.
type barRequest struct {
	r  *http.Request
	in map[string]any
	q  url.Values
}

func (s *Server) handleBedrockAgentRuntimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet && strings.HasPrefix(rawRequestPath(r), barDocumentPath) {
		s.barServeDocument(w, strings.TrimPrefix(rawRequestPath(r), barDocumentPath))
		return true
	}
	if !isBedrockAgentRuntimeCandidate(r) {
		return false
	}
	op, params := matchBedrockAgentRuntimeRoute(r.Method, rawRequestPath(r))
	if op == nil {
		return false
	}
	if r.ContentLength > barMaxBody {
		respondBedrockAgentError(w, baValidation("request body exceeds the maximum size"))
		return true
	}
	// ContentLength is absent for chunked bodies, so also bound the bytes
	// actually read (by the SigV4 check and the decoder).
	r.Body = http.MaxBytesReader(w, r.Body, barMaxBody)
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock"); !ok {
		respondBedrockAgentError(w, &bedrockAgentError{status: status, code: code, msg: msg})
		return true
	}
	in, err := decodeBedrockAgentBody(r)
	if err != nil {
		respondBedrockAgentError(w, baValidation("invalid JSON body"))
		return true
	}
	for k, v := range params {
		in[k] = v
	}
	req := &barRequest{r: r, in: in, q: r.URL.Query()}

	// Streaming operations write their own response.
	var streamErr *bedrockAgentError
	switch op.Name {
	case "InvokeAgent":
		streamErr = s.barInvokeAgent(w, req)
	case "InvokeInlineAgent":
		streamErr = s.barInvokeInlineAgent(w, req)
	case "InvokeFlow":
		streamErr = s.barInvokeFlow(w, req)
	case "RetrieveAndGenerateStream":
		streamErr = s.barRetrieveAndGenerateStream(w, req)
	case "OptimizePrompt":
		streamErr = s.barOptimizePrompt(w, req)
	case "AgenticRetrieveStream":
		streamErr = s.barAgenticRetrieve(w, req)
	default:
		handler := barHandlers[op.Name]
		if handler == nil {
			respondBedrockAgentError(w, &bedrockAgentError{status: http.StatusNotImplemented, code: "NotImplementedException", msg: op.Name + " is not implemented"})
			return true
		}
		out, apiErr := handler(s, req)
		if apiErr != nil {
			respondBedrockAgentError(w, apiErr)
			return true
		}
		body, _ := json.Marshal(out)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(op.Status)
		_, _ = w.Write(body)
		return true
	}
	if streamErr != nil {
		respondBedrockAgentError(w, streamErr)
	}
	return true
}

type barHandler func(*Server, *barRequest) (any, *bedrockAgentError)

var barHandlers map[string]barHandler

func init() {
	barHandlers = map[string]barHandler{
		"Retrieve":                 (*Server).barRetrieveOp,
		"RetrieveAndGenerate":      (*Server).barRetrieveAndGenerateOp,
		"Rerank":                   (*Server).barRerank,
		"GenerateQuery":            (*Server).barGenerateQuery,
		"GetDocumentContent":       (*Server).barGetDocumentContent,
		"CheckIngestedDocumentAcl": (*Server).barCheckDocumentAcl,
		"GetIngestedDocumentAcl":   (*Server).barGetDocumentAcl,
		"GetAgentMemory":           (*Server).barGetAgentMemory,
		"DeleteAgentMemory":        (*Server).barDeleteAgentMemory,
		"CreateSession":            (*Server).barCreateSession,
		"GetSession":               (*Server).barGetSession,
		"UpdateSession":            (*Server).barUpdateSession,
		"EndSession":               (*Server).barEndSession,
		"DeleteSession":            (*Server).barDeleteSession,
		"ListSessions":             (*Server).barListSessions,
		"CreateInvocation":         (*Server).barCreateInvocation,
		"ListInvocations":          (*Server).barListInvocations,
		"PutInvocationStep":        (*Server).barPutInvocationStep,
		"GetInvocationStep":        (*Server).barGetInvocationStep,
		"ListInvocationSteps":      (*Server).barListInvocationSteps,
		"TagResource":              (*Server).barTagResource,
		"UntagResource":            (*Server).barUntagResource,
		"ListTagsForResource":      (*Server).barListTags,
		"StartFlowExecution":       (*Server).barStartFlowExecution,
		"StopFlowExecution":        (*Server).barStopFlowExecution,
		"GetFlowExecution":         (*Server).barGetFlowExecution,
		"ListFlowExecutions":       (*Server).barListFlowExecutions,
		"ListFlowExecutionEvents":  (*Server).barListFlowExecutionEvents,
		"GetExecutionFlowSnapshot": (*Server).barGetFlowSnapshot,
	}
}

func isBedrockAgentRuntimeCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
	default:
		return false
	}
	if !hasAnyPrefix(rawRequestPath(r)+"/", barPathPrefixes) {
		return false
	}
	if sigV4ServiceHint(r) == "bedrock" {
		return true
	}
	return isBedrockAgentRuntimeHost(r)
}

func isBedrockAgentRuntimeHost(r *http.Request) bool {
	host := strings.ToLower(r.Host)
	return strings.HasPrefix(host, "bedrock-agent-runtime.") || strings.Contains(host, ".bedrock-agent-runtime.")
}

// matchBedrockAgentRuntimeRoute finds the operation whose method and URI
// template match; fewer path variables win, as for bedrock-agent.
func matchBedrockAgentRuntimeRoute(method, path string) (*bedrockagentruntimeOperation, map[string]string) {
	var best *bedrockagentruntimeOperation
	var bestParams map[string]string
	for i := range bedrockagentruntimeOperations {
		op := &bedrockagentruntimeOperations[i]
		if op.Method != method {
			continue
		}
		params, ok := bedrockAgentCoreControlPathParams(op.URI, path)
		if !ok {
			continue
		}
		if best == nil || len(params) < len(bestParams) {
			best, bestParams = op, params
		}
	}
	return best, bestParams
}

// --- small shared helpers ---

func barNewUUID() string { return uuid.NewString() }

func barNotFound(msg string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: msg}
}

func barQuota(msg string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusBadRequest, code: "ServiceQuotaExceededException", msg: msg}
}

// barMap reads in[key] as an object (nil when absent or not an object).
func barMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func barList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}

// barPath walks nested objects: barPath(in, "a", "b", "c") is in.a.b.c.
func barPath(in map[string]any, keys ...string) any {
	var cur any = in
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

func barPathStr(in map[string]any, keys ...string) string {
	s, _ := barPath(in, keys...).(string)
	return s
}

// barCheckLen enforces a string member's length bounds (in characters).
func barCheckLen(name, v string, least, most int) *bedrockAgentError {
	n := len([]rune(v))
	if n < least || n > most {
		return baValidation(name + " length must be between " + strconv.Itoa(least) + " and " + strconv.Itoa(most))
	}
	return nil
}

// barCount reads an optional integer member and checks its bounds.
func barCount(v any, name string, def, least, most int) (int, *bedrockAgentError) {
	if v == nil {
		return def, nil
	}
	n, ok := baInt(v)
	if !ok || n < least || n > most {
		return 0, baValidation(name + " must be between " + strconv.Itoa(least) + " and " + strconv.Itoa(most))
	}
	return n, nil
}

func barNowTime() time.Time { return time.Now().UTC() }

// barListOut is baList with an empty list (not null) when there are no items,
// since the list members are required in the output shapes.
func barListOut(key string, items []map[string]any, next string) map[string]any {
	if items == nil {
		items = []map[string]any{}
	}
	return baList(key, items, next)
}
