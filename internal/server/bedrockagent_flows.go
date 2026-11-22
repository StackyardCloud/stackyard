package server

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	baFlowIdentifier   = regexp.MustCompile(`^(?:arn:aws:bedrock:[a-z0-9-]{1,20}:[0-9]{12}:flow/)?([0-9a-zA-Z]{10})$`)
	baFlowAliasID      = regexp.MustCompile(`^(?:arn:aws:bedrock:[a-z0-9-]{1,20}:[0-9]{12}:flow/[0-9a-zA-Z]{10}/alias/)?([0-9a-zA-Z]{10})$`)
	baPromptIdentifier = regexp.MustCompile(`^(?:arn:aws:bedrock:[a-z0-9-]{1,20}:[0-9]{12}:prompt/)?([0-9a-zA-Z]{10})(?::([0-9]{1,5}))?$`)
	baVersionSelector  = regexp.MustCompile(`^(DRAFT|[0-9]{0,4}[1-9][0-9]{0,4})$`)
)

func bedrockAgentFlowHandlers() map[string]baHandler {
	return map[string]baHandler{
		"CreateFlow":                        (*bedrockAgentStore).createFlow,
		"GetFlow":                           (*bedrockAgentStore).getFlow,
		"UpdateFlow":                        (*bedrockAgentStore).updateFlow,
		"DeleteFlow":                        (*bedrockAgentStore).deleteFlow,
		"ListFlows":                         (*bedrockAgentStore).listFlows,
		"PrepareFlow":                       (*bedrockAgentStore).prepareFlow,
		"ValidateFlowDefinition":            (*bedrockAgentStore).validateFlowDefinition,
		"CreateFlowVersion":                 (*bedrockAgentStore).createFlowVersion,
		"GetFlowVersion":                    (*bedrockAgentStore).getFlowVersion,
		"DeleteFlowVersion":                 (*bedrockAgentStore).deleteFlowVersion,
		"ListFlowVersions":                  (*bedrockAgentStore).listFlowVersions,
		"CreateFlowAlias":                   (*bedrockAgentStore).createFlowAlias,
		"GetFlowAlias":                      (*bedrockAgentStore).getFlowAlias,
		"UpdateFlowAlias":                   (*bedrockAgentStore).updateFlowAlias,
		"DeleteFlowAlias":                   (*bedrockAgentStore).deleteFlowAlias,
		"ListFlowAliases":                   (*bedrockAgentStore).listFlowAliases,
		"CreatePrompt":                      (*bedrockAgentStore).createPrompt,
		"GetPrompt":                         (*bedrockAgentStore).getPrompt,
		"UpdatePrompt":                      (*bedrockAgentStore).updatePrompt,
		"DeletePrompt":                      (*bedrockAgentStore).deletePrompt,
		"ListPrompts":                       (*bedrockAgentStore).listPrompts,
		"CreatePromptVersion":               (*bedrockAgentStore).createPromptVersion,
		"CreateRetrievalInstruction":        (*bedrockAgentStore).createInstruction,
		"GetRetrievalInstruction":           (*bedrockAgentStore).getInstruction,
		"UpdateRetrievalInstruction":        (*bedrockAgentStore).updateInstruction,
		"DeleteRetrievalInstruction":        (*bedrockAgentStore).deleteInstruction,
		"ListRetrievalInstructions":         (*bedrockAgentStore).listInstructions,
		"CreateRetrievalInstructionVersion": (*bedrockAgentStore).createInstructionVersion,
		"ListRetrievalInstructionVersions":  (*bedrockAgentStore).listInstructionVersions,
	}
}

func newBAVersioned(draft map[string]any) *baVersioned {
	return &baVersioned{versions: map[string]map[string]any{bedrockAgentDraft: draft}, aliases: map[string]map[string]any{}}
}

func (v *baVersioned) draft() map[string]any { return v.versions[bedrockAgentDraft] }

// snapshot freezes the DRAFT as the next numbered version.
func (v *baVersioned) snapshot(in map[string]any) map[string]any {
	v.nextVersion++
	ver := strconv.Itoa(v.nextVersion)
	rec := baClone(v.draft())
	rec["version"] = ver
	now := baNow()
	rec["createdAt"], rec["updatedAt"] = now, now
	if d, ok := in["description"].(string); ok {
		rec["description"] = d
	}
	v.versions[ver] = rec
	return rec
}

func baUniqueDraftName(all map[string]*baVersioned, name, selfID string) *bedrockAgentError {
	for id, v := range all {
		if id != selfID && baStr(v.draft(), "name") == name {
			return baConflict("a resource named " + name + " already exists")
		}
	}
	return nil
}

// --- flows ---

func (st *bedrockAgentStore) flow(in map[string]any) (string, *baVersioned, *bedrockAgentError) {
	m := baFlowIdentifier.FindStringSubmatch(baStr(in, "flowIdentifier"))
	if m == nil {
		return "", nil, baValidation("flowIdentifier must be a flow ID or ARN")
	}
	f := st.flows[m[1]]
	if f == nil {
		return "", nil, baNotFound("flow", m[1])
	}
	return m[1], f, nil
}

func (st *bedrockAgentStore) validateFlowInput(in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "name"); err != nil {
		return err
	}
	if err := baRequire(in, "executionRoleArn"); err != nil {
		return err
	}
	return baUniqueDraftName(st.flows, baStr(in, "name"), selfID)
}

func (st *bedrockAgentStore) createFlow(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	if err := st.validateFlowInput(in, ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["id"] = id
	rec["arn"] = baARN("flow/" + id)
	rec["status"] = "NotPrepared"
	rec["version"] = bedrockAgentDraft
	rec["createdAt"], rec["updatedAt"] = now, now
	st.flows[id] = newBAVersioned(rec)
	st.setTags(rec["arn"].(string), in["tags"])
	return rec, nil
}

func (st *bedrockAgentStore) getFlow(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	out := baClone(f.draft())
	if def, ok := out["definition"].(map[string]any); ok {
		if v := baFlowValidations(def); len(v) > 0 {
			out["validations"] = v
		}
	}
	return out, nil
}

func (st *bedrockAgentStore) updateFlow(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	id, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	if err := st.validateFlowInput(in, id); err != nil {
		return nil, err
	}
	d := f.draft()
	baMerge(d, in, "flowIdentifier")
	d["status"] = "NotPrepared"
	d["updatedAt"] = baNow()
	return d, nil
}

func (st *bedrockAgentStore) deleteFlow(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	id, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	if !baBool(q.Get("skipResourceInUseCheck")) && len(f.aliases) > 0 {
		return nil, baConflict("flow has aliases; delete them or set skipResourceInUseCheck")
	}
	for _, a := range f.aliases {
		st.forget(baStr(a, "arn"))
	}
	st.forget(baStr(f.draft(), "arn"))
	delete(st.flows, id)
	return map[string]any{"id": id}, nil
}

func (st *bedrockAgentStore) listFlows(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	all := map[string]map[string]any{}
	for id, f := range st.flows {
		all[id] = f.draft()
	}
	return baListProjected(all, in, q, "flowSummaries", "name", "description", "id", "arn", "status", "createdAt", "updatedAt", "version")
}

func (st *bedrockAgentStore) prepareFlow(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	id, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	d := f.draft()
	def, _ := d["definition"].(map[string]any)
	if def == nil {
		return nil, baValidation("the flow has no definition to prepare")
	}
	for _, v := range baFlowValidations(def) {
		if v["severity"] == "Error" {
			return nil, baValidation("flow definition is invalid: " + baStr(v, "message"))
		}
	}
	d["status"] = "Prepared"
	d["updatedAt"] = baNow()
	return map[string]any{"id": id, "status": "Preparing"}, nil
}

func (st *bedrockAgentStore) validateFlowDefinition(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	def, ok := in["definition"].(map[string]any)
	if !ok {
		return nil, baValidation("definition is required")
	}
	validations := baFlowValidations(def)
	if validations == nil {
		validations = []map[string]any{}
	}
	return map[string]any{"validations": validations}, nil
}

// baFlowValidations checks the structural rules a flow must meet before it
// can be prepared: one Input and one Output node, and connections that name
// real nodes. Nodes other than Input with nothing flowing in are a warning.
func baFlowValidations(def map[string]any) []map[string]any {
	nodes, _ := def["nodes"].([]any)
	conns, _ := def["connections"].([]any)
	types := map[string]string{}
	for _, n := range nodes {
		nm, _ := n.(map[string]any)
		types[baStr(nm, "name")] = baStr(nm, "type")
	}
	var out []map[string]any
	add := func(severity, kind, msg string, details map[string]any) {
		out = append(out, map[string]any{"severity": severity, "type": kind, "message": msg, "details": details})
	}
	hasType := func(t string) bool {
		for _, v := range types {
			if v == t {
				return true
			}
		}
		return false
	}
	if !hasType("Input") {
		add("Error", "MissingStartingNodes", "the flow needs an Input node", map[string]any{"missingStartingNodes": map[string]any{}})
	}
	if !hasType("Output") {
		add("Error", "MissingEndingNodes", "the flow needs an Output node", map[string]any{"missingEndingNodes": map[string]any{}})
	}
	targets := map[string]bool{}
	for _, c := range conns {
		cm, _ := c.(map[string]any)
		name := baStr(cm, "name")
		if _, ok := types[baStr(cm, "source")]; !ok {
			add("Error", "UnknownConnectionSource", "connection "+name+" has an unknown source", map[string]any{"unknownConnectionSource": map[string]any{"connection": name}})
		}
		if _, ok := types[baStr(cm, "target")]; !ok {
			add("Error", "UnknownConnectionTarget", "connection "+name+" has an unknown target", map[string]any{"unknownConnectionTarget": map[string]any{"connection": name}})
		}
		targets[baStr(cm, "target")] = true
	}
	for name, t := range types {
		if t != "Input" && !targets[name] {
			add("Warning", "UnreachableNode", "node "+name+" is unreachable", map[string]any{"unreachableNode": map[string]any{"node": name}})
		}
	}
	return out
}

func (st *bedrockAgentStore) createFlowVersion(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	if f.draft()["status"] != "Prepared" {
		return nil, baValidation("prepare the flow before creating a version")
	}
	return f.snapshot(in), nil
}

func (st *bedrockAgentStore) flowVersion(in map[string]any) (*baVersioned, map[string]any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, nil, err
	}
	ver := baStr(in, "flowVersion")
	if !baNumericVersion.MatchString(ver) {
		return nil, nil, baValidation("flowVersion must be a version number")
	}
	rec := f.versions[ver]
	if rec == nil {
		return nil, nil, baNotFound("flow version", ver)
	}
	return f, rec, nil
}

func (st *bedrockAgentStore) getFlowVersion(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, rec, err := st.flowVersion(in)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (v *baVersioned) flowVersionInUse(ver string) bool {
	for _, a := range v.aliases {
		routes, _ := a["routingConfiguration"].([]any)
		for _, r := range routes {
			if m, _ := r.(map[string]any); baStr(m, "flowVersion") == ver {
				return true
			}
		}
	}
	return false
}

func (st *bedrockAgentStore) deleteFlowVersion(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	f, rec, err := st.flowVersion(in)
	if err != nil {
		return nil, err
	}
	ver := baStr(in, "flowVersion")
	if !baBool(q.Get("skipResourceInUseCheck")) && f.flowVersionInUse(ver) {
		return nil, baConflict("flow version " + ver + " is used by an alias")
	}
	delete(f.versions, ver)
	return map[string]any{"id": rec["id"], "version": ver}, nil
}

func (st *bedrockAgentStore) listFlowVersions(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	return baListProjected(f.versions, in, q, "flowVersionSummaries", "id", "arn", "status", "createdAt", "version")
}

// --- flow aliases ---

func (st *bedrockAgentStore) validateFlowAlias(f *baVersioned, in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "name"); err != nil {
		return err
	}
	for id, a := range f.aliases {
		if id != selfID && baStr(a, "name") == baStr(in, "name") {
			return baConflict("a flow alias named " + baStr(in, "name") + " already exists")
		}
	}
	routes, _ := in["routingConfiguration"].([]any)
	if len(routes) != 1 {
		return baValidation("routingConfiguration must contain exactly one entry")
	}
	m, _ := routes[0].(map[string]any)
	ver := baStr(m, "flowVersion")
	if !baVersionSelector.MatchString(ver) || f.versions[ver] == nil {
		return baValidation("flow version " + ver + " does not exist")
	}
	if cc, ok := in["concurrencyConfiguration"].(map[string]any); ok {
		if err := baCheckEnum(cc, "type", "Automatic", "Manual"); err != nil || baStr(cc, "type") == "" {
			return baValidation("concurrencyConfiguration.type must be Automatic or Manual")
		}
		if n, has := baInt(cc["maxConcurrency"]); baStr(cc, "type") == "Manual" && (!has || n < 1 || n > 100) {
			return baValidation("concurrencyConfiguration.maxConcurrency must be between 1 and 100 for Manual concurrency")
		}
	}
	return nil
}

func (st *bedrockAgentStore) createFlowAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	flowID, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	if err := st.validateFlowAlias(f, in, ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in, "flowIdentifier")
	rec["id"] = id
	rec["flowId"] = flowID
	rec["arn"] = baARN("flow/" + flowID + "/alias/" + id)
	rec["createdAt"], rec["updatedAt"] = now, now
	f.aliases[id] = rec
	st.setTags(rec["arn"].(string), in["tags"])
	return rec, nil
}

func (st *bedrockAgentStore) flowAlias(in map[string]any) (*baVersioned, map[string]any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, nil, err
	}
	m := baFlowAliasID.FindStringSubmatch(baStr(in, "aliasIdentifier"))
	if m == nil {
		return nil, nil, baValidation("aliasIdentifier must be a flow alias ID or ARN")
	}
	rec := f.aliases[m[1]]
	if rec == nil {
		return nil, nil, baNotFound("flow alias", m[1])
	}
	return f, rec, nil
}

func (st *bedrockAgentStore) getFlowAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, rec, err := st.flowAlias(in)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (st *bedrockAgentStore) updateFlowAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	f, rec, err := st.flowAlias(in)
	if err != nil {
		return nil, err
	}
	if err := st.validateFlowAlias(f, in, baStr(rec, "id")); err != nil {
		return nil, err
	}
	baMerge(rec, in, "flowIdentifier", "aliasIdentifier")
	rec["updatedAt"] = baNow()
	return rec, nil
}

func (st *bedrockAgentStore) deleteFlowAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	f, rec, err := st.flowAlias(in)
	if err != nil {
		return nil, err
	}
	st.forget(baStr(rec, "arn"))
	delete(f.aliases, baStr(rec, "id"))
	return map[string]any{"flowId": rec["flowId"], "id": rec["id"]}, nil
}

func (st *bedrockAgentStore) listFlowAliases(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, f, err := st.flow(in)
	if err != nil {
		return nil, err
	}
	return baListProjected(f.aliases, in, q, "flowAliasSummaries", "name", "description", "routingConfiguration", "concurrencyConfiguration", "flowId", "id", "arn", "createdAt", "updatedAt")
}

// --- prompts ---

// prompt resolves promptIdentifier (ID or ARN, optionally ARN:version) and the
// requested version (promptVersion query wins over the ARN suffix).
func (st *bedrockAgentStore) prompt(in map[string]any, q url.Values) (string, *baVersioned, string, *bedrockAgentError) {
	m := baPromptIdentifier.FindStringSubmatch(baStr(in, "promptIdentifier"))
	if m == nil {
		return "", nil, "", baValidation("promptIdentifier must be a prompt ID or ARN")
	}
	p := st.prompts[m[1]]
	if p == nil {
		return "", nil, "", baNotFound("prompt", m[1])
	}
	ver := m[2]
	if qv := q.Get("promptVersion"); qv != "" {
		if !baVersionSelector.MatchString(qv) {
			return "", nil, "", baValidation("promptVersion must be DRAFT or a version number")
		}
		ver = qv
	}
	return m[1], p, ver, nil
}

func (st *bedrockAgentStore) validatePromptInput(in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "name"); err != nil {
		return err
	}
	if err := baUniqueDraftName(st.prompts, baStr(in, "name"), selfID); err != nil {
		return err
	}
	variants, _ := in["variants"].([]any)
	if len(variants) > 1 {
		return baValidation("variants accepts at most one variant")
	}
	names := map[string]bool{}
	for _, v := range variants {
		vm, _ := v.(map[string]any)
		if err := baRequire(vm, "name", "templateType", "templateConfiguration"); err != nil {
			return baValidation("each variant needs a name, templateType and templateConfiguration")
		}
		if err := baCheckEnum(vm, "templateType", "TEXT", "CHAT"); err != nil {
			return err
		}
		names[baStr(vm, "name")] = true
	}
	if dv := baStr(in, "defaultVariant"); dv != "" && !names[dv] {
		return baValidation("defaultVariant must name one of the variants")
	}
	return nil
}

func (st *bedrockAgentStore) createPrompt(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	if err := st.validatePromptInput(in, ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["id"] = id
	rec["arn"] = baARN("prompt/" + id)
	rec["version"] = bedrockAgentDraft
	rec["createdAt"], rec["updatedAt"] = now, now
	st.prompts[id] = newBAVersioned(rec)
	st.setTags(rec["arn"].(string), in["tags"])
	return rec, nil
}

func (st *bedrockAgentStore) getPrompt(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, p, ver, err := st.prompt(in, q)
	if err != nil {
		return nil, err
	}
	if ver == "" {
		ver = bedrockAgentDraft
	}
	rec := p.versions[ver]
	if rec == nil {
		return nil, baNotFound("prompt version", ver)
	}
	return rec, nil
}

func (st *bedrockAgentStore) updatePrompt(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	id, p, _, err := st.prompt(in, q)
	if err != nil {
		return nil, err
	}
	if err := st.validatePromptInput(in, id); err != nil {
		return nil, err
	}
	d := p.draft()
	baMerge(d, in, "promptIdentifier")
	d["updatedAt"] = baNow()
	return d, nil
}

func (st *bedrockAgentStore) deletePrompt(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	id, p, ver, err := st.prompt(in, q)
	if err != nil {
		return nil, err
	}
	if ver == "" {
		for v, rec := range p.versions {
			if v != bedrockAgentDraft {
				st.forget(baStr(rec, "arn"))
			}
		}
		st.forget(baStr(p.draft(), "arn"))
		delete(st.prompts, id)
		return map[string]any{"id": id}, nil
	}
	if ver == bedrockAgentDraft {
		return nil, baValidation("delete the prompt (without promptVersion) to remove its DRAFT")
	}
	rec := p.versions[ver]
	if rec == nil {
		return nil, baNotFound("prompt version", ver)
	}
	st.forget(baStr(rec, "arn"))
	delete(p.versions, ver)
	return map[string]any{"id": id, "version": ver}, nil
}

func (st *bedrockAgentStore) listPrompts(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	records := map[string]map[string]any{}
	if q.Get("promptIdentifier") != "" {
		_, p, _, err := st.prompt(map[string]any{"promptIdentifier": q.Get("promptIdentifier")}, url.Values{})
		if err != nil {
			return nil, err
		}
		records = p.versions
	} else {
		for id, p := range st.prompts {
			records[id] = p.draft()
		}
	}
	return baListProjected(records, in, q, "promptSummaries", "name", "description", "id", "arn", "version", "createdAt", "updatedAt")
}

func (st *bedrockAgentStore) createPromptVersion(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, p, _, err := st.prompt(in, q)
	if err != nil {
		return nil, err
	}
	rec := p.snapshot(in)
	rec["arn"] = baStr(p.draft(), "arn") + ":" + baStr(rec, "version")
	st.setTags(rec["arn"].(string), in["tags"])
	return rec, nil
}

// --- retrieval instructions ---

func (st *bedrockAgentStore) instruction(in map[string]any) (string, *baVersioned, *bedrockAgentError) {
	id := baStr(in, "retrievalInstructionId")
	if !baInstructionIDPat.MatchString(id) {
		return "", nil, baValidation("retrievalInstructionId must be 10 lowercase alphanumeric characters")
	}
	ri := st.instructions[id]
	if ri == nil {
		return "", nil, baNotFound("retrieval instruction", id)
	}
	return id, ri, nil
}

func validateInstructionContent(in map[string]any) *bedrockAgentError {
	content, ok := in["instruction"].(map[string]any)
	if !ok {
		return baValidation("instruction is required")
	}
	for _, k := range []string{"retrievalGuidance", "persona", "answerFormat", "answerPolicy"} {
		if strings.TrimSpace(baStr(content, k)) != "" {
			return nil
		}
	}
	return baValidation("instruction needs at least one of retrievalGuidance, persona, answerFormat or answerPolicy")
}

func (ri *baVersioned) latest() string {
	if ri.nextVersion == 0 {
		return ""
	}
	for n := ri.nextVersion; n > 0; n-- {
		if ri.versions[strconv.Itoa(n)] != nil {
			return strconv.Itoa(n)
		}
	}
	return ""
}

func (st *bedrockAgentStore) createInstruction(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	if err := baCheckName(in, "name"); err != nil {
		return nil, err
	}
	if err := validateInstructionContent(in); err != nil {
		return nil, err
	}
	if err := baUniqueDraftName(st.instructions, baStr(in, "name"), ""); err != nil {
		return nil, err
	}
	id := baNewID("abcdefghijklmnopqrstuvwxyz0123456789", 10)
	now := baNow()
	rec := baRecord(in)
	rec["retrievalInstructionId"] = id
	rec["retrievalInstructionArn"] = baARN("retrieval-instruction/" + id)
	rec["status"] = "ACTIVE"
	rec["lastTransitionStatus"] = "CREATE_SUCCESSFUL"
	rec["version"] = bedrockAgentDraft
	rec["createdAt"], rec["updatedAt"] = now, now
	st.instructions[id] = newBAVersioned(rec)
	st.setTags(rec["retrievalInstructionArn"].(string), in["tags"])
	return baInstructionSummary(baWithStatus(rec, "lastTransitionStatus", "CREATING")), nil
}

func baInstructionSummary(rec map[string]any) map[string]any {
	return baProject(rec, "retrievalInstructionId", "retrievalInstructionArn", "name", "description", "status", "lastTransitionStatus", "createdAt", "updatedAt")
}

func (st *bedrockAgentStore) getInstruction(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, ri, err := st.instruction(in)
	if err != nil {
		return nil, err
	}
	ver := q.Get("retrievalInstructionVersion")
	if ver == "" {
		ver = bedrockAgentDraft
	}
	rec := ri.versions[ver]
	if rec == nil {
		return nil, baNotFound("retrieval instruction version", ver)
	}
	out := baClone(rec)
	if latest := ri.latest(); latest != "" {
		out["latestVersion"] = latest
	}
	return out, nil
}

func (st *bedrockAgentStore) updateInstruction(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ri, err := st.instruction(in)
	if err != nil {
		return nil, err
	}
	if err := validateInstructionContent(in); err != nil {
		return nil, err
	}
	d := ri.draft()
	baMerge(d, in, "retrievalInstructionId")
	d["lastTransitionStatus"] = "UPDATE_SUCCESSFUL"
	d["updatedAt"] = baNow()
	return baInstructionSummary(baWithStatus(d, "lastTransitionStatus", "UPDATING")), nil
}

func (st *bedrockAgentStore) deleteInstruction(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	id, ri, err := st.instruction(in)
	if err != nil {
		return nil, err
	}
	arn := baStr(ri.draft(), "retrievalInstructionArn")
	ver := q.Get("retrievalInstructionVersion")
	if ver == "" || ver == bedrockAgentDraft {
		st.forget(arn)
		delete(st.instructions, id)
		return map[string]any{"retrievalInstructionId": id, "retrievalInstructionArn": arn}, nil
	}
	if ri.versions[ver] == nil {
		return nil, baNotFound("retrieval instruction version", ver)
	}
	delete(ri.versions, ver)
	return map[string]any{"retrievalInstructionId": id, "retrievalInstructionArn": arn, "version": ver}, nil
}

func (st *bedrockAgentStore) listInstructions(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	status := q.Get("status")
	all := map[string]map[string]any{}
	for id, ri := range st.instructions {
		if status == "" || ri.draft()["status"] == status {
			all[id] = ri.draft()
		}
	}
	return baListProjected(all, in, q, "retrievalInstructionSummaries", "retrievalInstructionId", "retrievalInstructionArn", "name", "description", "status", "lastTransitionStatus", "createdAt", "updatedAt")
}

func (st *bedrockAgentStore) createInstructionVersion(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ri, err := st.instruction(in)
	if err != nil {
		return nil, err
	}
	rec := ri.snapshot(in)
	return baProject(rec, "retrievalInstructionId", "retrievalInstructionArn", "version", "createdAt", "description"), nil
}

func (st *bedrockAgentStore) listInstructionVersions(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, ri, err := st.instruction(in)
	if err != nil {
		return nil, err
	}
	numbered := map[string]map[string]any{}
	for v, rec := range ri.versions {
		if v != bedrockAgentDraft {
			numbered[v] = rec
		}
	}
	return baListProjected(numbered, in, q, "retrievalInstructionVersionSummaries", "version", "createdAt", "description")
}
