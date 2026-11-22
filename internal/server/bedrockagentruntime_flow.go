package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Flows run by walking the flow version's definition from its Input node:
// Prompt nodes fill their template with the value, KnowledgeBase nodes
// retrieve, Condition nodes pick a branch, Output nodes emit. Every other node
// type passes its input through unchanged.
// ponytail: one value travels along each connection; multi-input nodes, loops
// and iterators aren't modelled. The walk is capped at barMaxFlowSteps.

const barMaxFlowSteps = 200

// barMaxFlowOutputBytes bounds the total node output of one flow run.
const barMaxFlowOutputBytes = 4 << 20

// barValueSize is the JSON size of a node value (0 if it can't be encoded).
func barValueSize(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}

var barNodeTypes = map[string]string{
	"Input": "FlowInputNode", "Output": "FlowOutputNode", "Prompt": "PromptNode", "KnowledgeBase": "KnowledgeBaseNode",
	"Condition": "ConditionNode", "LambdaFunction": "LambdaFunctionNode", "Lex": "LexNode",
}

var barTemplateVar = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_]+)\s*\}\}`)

// barFlowRun is the result of one walk.
type barFlowRun struct {
	outputs   []map[string]any // flowOutputEvent payloads
	trace     []map[string]any // FlowTrace members (for flowTraceEvent)
	execEvent []map[string]any // FlowExecutionEvent members (for ListFlowExecutionEvents)
}

func barNodeType(t string) string {
	if v := barNodeTypes[t]; v != "" {
		return v
	}
	return t + "Node"
}

func (s *Server) barRunFlow(flow map[string]any, inputs []any) (*barFlowRun, *bedrockAgentError) {
	if len(inputs) != 1 {
		return nil, baValidation("inputs must contain exactly 1 input")
	}
	input, _ := inputs[0].(map[string]any)
	content := barMap(input, "content")
	if content == nil || content["document"] == nil {
		return nil, baValidation("inputs[0].content.document is required")
	}
	def := barMap(flow, "definition")
	nodes := map[string]map[string]any{}
	for _, n := range barList(def, "nodes") {
		m, _ := n.(map[string]any)
		nodes[baStr(m, "name")] = m
	}
	start := nodes[baStr(input, "nodeName")]
	if start == nil || baStr(start, "type") != "Input" {
		return nil, baValidation("inputs[0].nodeName must name the flow's Input node")
	}
	outName := baStr(input, "nodeOutputName")
	if outName == "" {
		outName = "document"
	}

	run := &barFlowRun{}
	now := baNow()
	run.execEvent = append(run.execEvent, map[string]any{"flowInputEvent": map[string]any{
		"nodeName": baStr(start, "name"), "timestamp": now,
		"fields": []map[string]any{{"name": outName, "content": map[string]any{"document": content["document"]}}},
	}})
	type item struct {
		node  string
		value any
	}
	queue := []item{{baStr(start, "name"), content["document"]}}
	produced := 0
	for steps := 0; len(queue) > 0; steps++ {
		if steps >= barMaxFlowSteps {
			return nil, baValidation(fmt.Sprintf("the flow ran more than %d node steps", barMaxFlowSteps))
		}
		cur := queue[0]
		queue = queue[1:]
		node := nodes[cur.node]
		kind := baStr(node, "type")
		run.trace = append(run.trace, map[string]any{"nodeInputTrace": map[string]any{
			"nodeName": cur.node, "timestamp": baNow(), "fields": []map[string]any{{"nodeInputName": "document", "content": map[string]any{"document": cur.value}}},
		}})
		run.execEvent = append(run.execEvent, map[string]any{"nodeInputEvent": map[string]any{
			"nodeName": cur.node, "timestamp": baNow(), "fields": []map[string]any{{"name": "document", "content": map[string]any{"document": cur.value}}},
		}})
		out, branch, err := s.barRunNode(node, cur.value)
		if err != nil {
			return nil, err
		}
		// The step cap alone doesn't bound memory: a Prompt template that
		// repeats its input grows the value every step, and every value is
		// copied into the trace and events. Cap the total produced.
		if produced += barValueSize(out); produced > barMaxFlowOutputBytes {
			return nil, baValidation(fmt.Sprintf("the flow produced more than %d bytes of node output", barMaxFlowOutputBytes))
		}
		if kind == "Output" {
			run.outputs = append(run.outputs, map[string]any{"nodeName": cur.node, "nodeType": "FlowOutputNode", "content": map[string]any{"document": out}})
			run.execEvent = append(run.execEvent, map[string]any{"flowOutputEvent": map[string]any{
				"nodeName": cur.node, "timestamp": baNow(), "fields": []map[string]any{{"name": "document", "content": map[string]any{"document": out}}},
			}})
			continue
		}
		if kind == "Condition" {
			run.trace = append(run.trace, map[string]any{"conditionNodeResultTrace": map[string]any{
				"nodeName": cur.node, "timestamp": baNow(), "satisfiedConditions": []map[string]any{{"conditionName": branch}},
			}})
			run.execEvent = append(run.execEvent, map[string]any{"conditionResultEvent": map[string]any{
				"nodeName": cur.node, "timestamp": baNow(), "satisfiedConditions": []map[string]any{{"conditionName": branch}},
			}})
		} else {
			run.trace = append(run.trace, map[string]any{"nodeOutputTrace": map[string]any{
				"nodeName": cur.node, "timestamp": baNow(), "fields": []map[string]any{{"nodeOutputName": "document", "content": map[string]any{"document": out}}},
			}})
			run.execEvent = append(run.execEvent, map[string]any{"nodeOutputEvent": map[string]any{
				"nodeName": cur.node, "timestamp": baNow(), "fields": []map[string]any{{"name": "document", "content": map[string]any{"document": out}}},
			}})
		}
		for _, c := range barList(def, "connections") {
			conn, _ := c.(map[string]any)
			if baStr(conn, "source") != cur.node {
				continue
			}
			conditional := baStr(conn, "type") == "Conditional"
			if kind == "Condition" {
				if !conditional || barPathStr(conn, "configuration", "conditional", "condition") != branch {
					continue
				}
			} else if conditional {
				continue
			}
			queue = append(queue, item{baStr(conn, "target"), out})
		}
	}
	return run, nil
}

// barRunNode returns a node's output and, for a Condition node, the name of
// the satisfied condition.
func (s *Server) barRunNode(node map[string]any, value any) (any, string, *bedrockAgentError) {
	cfg := barMap(node, "configuration")
	switch baStr(node, "type") {
	case "Prompt":
		src := barMap(barMap(cfg, "prompt"), "sourceConfiguration")
		tmpl := barPathStr(src, "inline", "templateConfiguration", "text", "text")
		if arn := barPathStr(src, "resource", "promptArn"); arn != "" {
			prompt, ok := s.bedrockagent.LookupPrompt(arn)
			if !ok {
				return nil, "", barNotFound("Prompt " + arn + " was not found")
			}
			if variants := barList(prompt, "variants"); len(variants) > 0 {
				v, _ := variants[0].(map[string]any)
				tmpl = barPathStr(v, "templateConfiguration", "text", "text")
			}
		}
		text := barValueText(value)
		if tmpl == "" {
			return text, "", nil
		}
		// Bound the result before building it: every placeholder copies the
		// whole input, so many placeholders times a large input is huge.
		if n := len(barTemplateVar.FindAllStringIndex(tmpl, -1)); n > 0 &&
			(len(tmpl) > barMaxFlowOutputBytes || len(text) > (barMaxFlowOutputBytes-len(tmpl))/n) {
			return nil, "", baValidation(fmt.Sprintf("the Prompt node output would exceed %d bytes", barMaxFlowOutputBytes))
		}
		return barTemplateVar.ReplaceAllLiteralString(tmpl, text), "", nil
	case "KnowledgeBase":
		kb := barMap(cfg, "knowledgeBase")
		n, _, err := barRetrievalConfig(barMap(kb, "retrievalConfiguration"))
		if err != nil {
			return nil, "", err
		}
		hits, err := s.barRetrieveHits(baStr(kb, "knowledgeBaseId"), barValueText(value), n, nil)
		if err != nil {
			return nil, "", err
		}
		if baStr(kb, "modelId") != "" {
			if len(hits) == 0 {
				return barNoAnswer, "", nil
			}
			return hits[0].text, "", nil
		}
		results := make([]map[string]any, len(hits))
		for i, h := range hits {
			results[i] = barRetrievalResult(h)
		}
		return results, "", nil
	case "Condition":
		for _, c := range barList(barMap(cfg, "condition"), "conditions") {
			cond, _ := c.(map[string]any)
			if name := baStr(cond, "name"); name != "default" && barCondition(baStr(cond, "expression"), value) {
				return value, name, nil
			}
		}
		return value, "default", nil
	}
	return value, "", nil
}

func barValueText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

var barConditionExpr = regexp.MustCompile(`^\s*([A-Za-z0-9_.]+)\s*(==|!=|>=|<=|>|<)\s*(.+?)\s*$`)

// barCondition evaluates "<input> <op> <literal>" where the left side is the
// node's input (its name is ignored) and the literal is a number or a quoted
// string. Anything else is false.
func barCondition(expr string, value any) bool {
	m := barConditionExpr.FindStringSubmatch(expr)
	if m == nil {
		return false
	}
	op, lit := m[2], m[3]
	if f, err := strconv.ParseFloat(lit, 64); err == nil {
		v, ok := barNumber(value)
		if !ok {
			if s, isStr := value.(string); isStr {
				if parsed, perr := strconv.ParseFloat(s, 64); perr == nil {
					v, ok = parsed, true
				}
			}
		}
		if !ok {
			return false
		}
		switch op {
		case "==":
			return v == f
		case "!=":
			return v != f
		case ">":
			return v > f
		case "<":
			return v < f
		case ">=":
			return v >= f
		}
		return v <= f
	}
	want := strings.Trim(lit, `"'`)
	got := barValueText(value)
	switch op {
	case "==":
		return got == want
	case "!=":
		return got != want
	}
	return false
}

func (s *Server) barLookupFlow(in map[string]any) (map[string]any, *bedrockAgentError) {
	flowID, aliasID := baStr(in, "flowIdentifier"), baStr(in, "flowAliasIdentifier")
	flow, ok := s.bedrockagent.LookupFlowAlias(flowID, aliasID)
	if !ok {
		return nil, barNotFound("Flow " + flowID + " with alias " + aliasID + " was not found")
	}
	if baStr(flow, "status") != "" && baStr(flow, "status") != "Prepared" {
		return nil, baConflict("Flow " + flowID + " is " + baStr(flow, "status") + "; prepare the flow before invoking it")
	}
	return flow, nil
}

func (s *Server) barInvokeFlow(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	flow, err := s.barLookupFlow(req.in)
	if err != nil {
		return err
	}
	execID := baStr(req.in, "executionId")
	if execID != "" && !barExecutionID.MatchString(execID) {
		return baValidation("executionId must be 1-36 letters, digits or hyphens")
	}
	run, err := s.barRunFlow(flow, barList(req.in, "inputs"))
	if err != nil {
		return err
	}
	if execID == "" {
		execID = barNewUUID()
	}
	w.Header().Set("x-amz-bedrock-flow-execution-id", execID)
	st := brtStartStream(w)
	if baBool(req.in["enableTrace"]) {
		for _, t := range run.trace {
			st.event("flowTraceEvent", map[string]any{"trace": t})
		}
	}
	for _, o := range run.outputs {
		st.event("flowOutputEvent", o)
	}
	st.event("flowCompletionEvent", map[string]any{"completionReason": "SUCCESS"})
	return nil
}

// --- asynchronous executions ---

// barExecution is a StartFlowExecution. It reports Running until it's first
// read, so StopFlowExecution has something to stop.
type barExecution struct {
	rec    map[string]any
	run    *barFlowRun
	flow   map[string]any
	seen   bool
	flowID string
}

func barAliasID(identifier string) string {
	return identifier[strings.LastIndex(identifier, "/")+1:]
}

func barFlowID(identifier string) string {
	if i := strings.Index(identifier, ":flow/"); i >= 0 {
		rest := identifier[i+len(":flow/"):]
		if j := strings.Index(rest, "/"); j >= 0 {
			rest = rest[:j]
		}
		return rest
	}
	return identifier
}

func (s *Server) barStartFlowExecution(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	flow, err := s.barLookupFlow(in)
	if err != nil {
		return nil, err
	}
	id := baStr(in, "flowExecutionName")
	if id != "" && !regexp.MustCompile(`^[a-zA-Z0-9-]{1,36}$`).MatchString(id) {
		return nil, baValidation("flowExecutionName must be 1-36 letters, digits or hyphens")
	}
	run, err := s.barRunFlow(flow, barList(in, "inputs"))
	if err != nil {
		return nil, err
	}
	if id == "" {
		id = barNewUUID()
	}
	flowID, aliasID := barFlowID(baStr(in, "flowIdentifier")), barAliasID(baStr(in, "flowAliasIdentifier"))
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	key := flowID + "/" + id
	if st.executions[key] != nil {
		return nil, baConflict("an execution named " + id + " already exists")
	}
	if len(st.executions) >= barMaxExecutions {
		return nil, barQuota("the account has reached the maximum number of flow executions")
	}
	now := baNow()
	rec := map[string]any{
		"executionArn": baARN("flow/" + flowID + "/alias/" + aliasID + "/execution/" + id), "flowIdentifier": flowID,
		"flowAliasIdentifier": aliasID, "flowVersion": baStr(flow, "version"), "status": "Running", "startedAt": now, "createdAt": now,
	}
	if rec["flowVersion"] == "" {
		rec["flowVersion"] = bedrockAgentDraft
	}
	st.executions[key] = &barExecution{rec: rec, run: run, flow: flow, flowID: flowID}
	return map[string]any{"executionArn": rec["executionArn"]}, nil
}

// executionLocked finds an execution and, on its first read, completes it.
func (st *barStore) executionLocked(in map[string]any) (*barExecution, *bedrockAgentError) {
	m := barExecutionID.FindStringSubmatch(baStr(in, "executionIdentifier"))
	if m == nil {
		return nil, baValidation("executionIdentifier must be an execution ID or ARN")
	}
	ex := st.executions[barFlowID(baStr(in, "flowIdentifier"))+"/"+m[1]]
	if ex == nil || (baStr(in, "flowAliasIdentifier") != "" && ex.rec["flowAliasIdentifier"] != barAliasID(baStr(in, "flowAliasIdentifier"))) {
		return nil, barNotFound("Flow execution " + m[1] + " was not found")
	}
	return ex, nil
}

func (ex *barExecution) settle() {
	if !ex.seen && ex.rec["status"] == "Running" {
		ex.rec["status"], ex.rec["endedAt"] = "Succeeded", baNow()
	}
	ex.seen = true
}

func (s *Server) barStopFlowExecution(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	ex, err := st.executionLocked(req.in)
	if err != nil {
		return nil, err
	}
	if ex.rec["status"] != "Running" {
		return nil, baConflict("the execution is already " + baStr(ex.rec, "status"))
	}
	ex.rec["status"], ex.rec["endedAt"], ex.seen = "Aborted", baNow(), true
	return map[string]any{"executionArn": ex.rec["executionArn"], "status": "Aborted"}, nil
}

func (s *Server) barGetFlowExecution(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	ex, err := st.executionLocked(req.in)
	if err != nil {
		return nil, err
	}
	ex.settle()
	return baProject(ex.rec, "executionArn", "flowIdentifier", "flowAliasIdentifier", "flowVersion", "status", "startedAt", "endedAt", "errors"), nil
}

func (s *Server) barListFlowExecutions(req *barRequest) (any, *bedrockAgentError) {
	flowID := barFlowID(baStr(req.in, "flowIdentifier"))
	if _, ok := s.bedrockagent.LookupFlowAlias(flowID, bedrockAgentTestAlias); !ok {
		return nil, barNotFound("Flow " + flowID + " was not found")
	}
	alias := req.q.Get("flowAliasIdentifier")
	st := s.bedrockagentruntime
	st.mu.Lock()
	var items []map[string]any
	for _, ex := range st.executions {
		if ex.flowID != flowID || (alias != "" && ex.rec["flowAliasIdentifier"] != barAliasID(alias)) {
			continue
		}
		ex.settle()
		items = append(items, baProject(ex.rec, "executionArn", "flowIdentifier", "flowAliasIdentifier", "flowVersion", "status", "createdAt", "endedAt"))
	}
	st.mu.Unlock()
	sort.SliceStable(items, func(i, j int) bool { return baStr(items[i], "createdAt") > baStr(items[j], "createdAt") })
	page, next, err := baPage(items, map[string]any{}, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("flowExecutionSummaries", page, next), nil
}

func (s *Server) barListFlowExecutionEvents(req *barRequest) (any, *bedrockAgentError) {
	kind := req.q.Get("eventType")
	if kind != "Node" && kind != "Flow" {
		return nil, baValidation("eventType must be Node or Flow")
	}
	st := s.bedrockagentruntime
	st.mu.Lock()
	ex, err := st.executionLocked(req.in)
	var events []map[string]any
	if err == nil {
		ex.settle()
		for _, e := range ex.run.execEvent {
			_, isFlow := e["flowInputEvent"]
			if _, out := e["flowOutputEvent"]; out {
				isFlow = true
			}
			if isFlow == (kind == "Flow") {
				events = append(events, e)
			}
		}
	}
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	page, next, err := baPage(events, map[string]any{}, req.q)
	if err != nil {
		return nil, err
	}
	return barListOut("flowExecutionEvents", page, next), nil
}

func (s *Server) barGetFlowSnapshot(req *barRequest) (any, *bedrockAgentError) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	defer st.mu.Unlock()
	ex, err := st.executionLocked(req.in)
	if err != nil {
		return nil, err
	}
	ex.settle()
	def, _ := json.Marshal(ex.flow["definition"])
	out := map[string]any{
		"definition": string(def), "executionRoleArn": baStr(ex.flow, "executionRoleArn"),
		"flowIdentifier": ex.flowID, "flowAliasIdentifier": ex.rec["flowAliasIdentifier"], "flowVersion": ex.rec["flowVersion"],
	}
	if k := baStr(ex.flow, "customerEncryptionKeyArn"); k != "" {
		out["customerEncryptionKeyArn"] = k
	}
	return out, nil
}
