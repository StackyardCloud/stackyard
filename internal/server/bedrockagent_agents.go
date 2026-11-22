package server

import (
	"net/url"
	"strconv"
	"strings"
)

func bedrockAgentAgentHandlers() map[string]baHandler {
	return map[string]baHandler{
		"CreateAgent":                    (*bedrockAgentStore).createAgent,
		"GetAgent":                       (*bedrockAgentStore).getAgent,
		"UpdateAgent":                    (*bedrockAgentStore).updateAgent,
		"DeleteAgent":                    (*bedrockAgentStore).deleteAgent,
		"ListAgents":                     (*bedrockAgentStore).listAgents,
		"PrepareAgent":                   (*bedrockAgentStore).prepareAgent,
		"GetAgentVersion":                (*bedrockAgentStore).getAgentVersion,
		"ListAgentVersions":              (*bedrockAgentStore).listAgentVersions,
		"DeleteAgentVersion":             (*bedrockAgentStore).deleteAgentVersion,
		"CreateAgentAlias":               (*bedrockAgentStore).createAgentAlias,
		"GetAgentAlias":                  (*bedrockAgentStore).getAgentAlias,
		"UpdateAgentAlias":               (*bedrockAgentStore).updateAgentAlias,
		"DeleteAgentAlias":               (*bedrockAgentStore).deleteAgentAlias,
		"ListAgentAliases":               (*bedrockAgentStore).listAgentAliases,
		"CreateAgentActionGroup":         (*bedrockAgentStore).createActionGroup,
		"GetAgentActionGroup":            (*bedrockAgentStore).getActionGroup,
		"UpdateAgentActionGroup":         (*bedrockAgentStore).updateActionGroup,
		"DeleteAgentActionGroup":         (*bedrockAgentStore).deleteActionGroup,
		"ListAgentActionGroups":          (*bedrockAgentStore).listActionGroups,
		"AssociateAgentKnowledgeBase":    (*bedrockAgentStore).associateKnowledgeBase,
		"GetAgentKnowledgeBase":          (*bedrockAgentStore).getAgentKnowledgeBase,
		"UpdateAgentKnowledgeBase":       (*bedrockAgentStore).updateAgentKnowledgeBase,
		"DisassociateAgentKnowledgeBase": (*bedrockAgentStore).disassociateKnowledgeBase,
		"ListAgentKnowledgeBases":        (*bedrockAgentStore).listAgentKnowledgeBases,
		"AssociateAgentCollaborator":     (*bedrockAgentStore).associateCollaborator,
		"GetAgentCollaborator":           (*bedrockAgentStore).getCollaborator,
		"UpdateAgentCollaborator":        (*bedrockAgentStore).updateCollaborator,
		"DisassociateAgentCollaborator":  (*bedrockAgentStore).disassociateCollaborator,
		"ListAgentCollaborators":         (*bedrockAgentStore).listCollaborators,
	}
}

// --- lookups ---

func (st *bedrockAgentStore) agent(in map[string]any) (*baAgent, *bedrockAgentError) {
	if err := baCheckID(in, "agentId"); err != nil {
		return nil, err
	}
	a := st.agents[baStr(in, "agentId")]
	if a == nil {
		return nil, baNotFound("agent", baStr(in, "agentId"))
	}
	return a, nil
}

func (a *baAgent) draft() map[string]any { return a.versions[bedrockAgentDraft].agent }

// agentVersion resolves {agentId, agentVersion}. Writes to sub-resources are
// only allowed on the DRAFT version.
func (st *bedrockAgentStore) agentVersion(in map[string]any, draftOnly bool) (*baAgent, *baAgentVersion, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, nil, err
	}
	ver := baStr(in, "agentVersion")
	if draftOnly && ver != bedrockAgentDraft {
		return nil, nil, baValidation("agentVersion must be DRAFT")
	}
	if ver != bedrockAgentDraft && !baNumericVersion.MatchString(ver) {
		return nil, nil, baValidation("agentVersion must be DRAFT or a version number")
	}
	v := a.versions[ver]
	if v == nil {
		return nil, nil, baNotFound("agent version", ver)
	}
	return a, v, nil
}

// touchDraft records that the working draft changed and must be prepared again.
func (a *baAgent) touchDraft() {
	d := a.draft()
	d["agentStatus"] = "NOT_PREPARED"
	d["updatedAt"] = baNow()
}

func baWithStatus(rec map[string]any, key, status string) map[string]any {
	out := baClone(rec)
	out[key] = status
	return out
}

// --- agents ---

func (st *bedrockAgentStore) validateAgentInput(in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "agentName"); err != nil {
		return err
	}
	for id, a := range st.agents {
		if id != selfID && baStr(a.draft(), "agentName") == baStr(in, "agentName") {
			return baConflict("an agent named " + baStr(in, "agentName") + " already exists")
		}
	}
	if v, ok := in["idleSessionTTLInSeconds"]; ok {
		n, isInt := baInt(v)
		if !isInt || n < 60 || n > 5400 {
			return baValidation("idleSessionTTLInSeconds must be between 60 and 5400")
		}
	}
	if s, ok := in["instruction"].(string); ok && (len(s) < 40 || len(s) > 20000) {
		return baValidation("instruction must be between 40 and 20000 characters")
	}
	if err := baCheckEnum(in, "agentCollaboration", "SUPERVISOR", "SUPERVISOR_ROUTER", "DISABLED"); err != nil {
		return err
	}
	return baCheckEnum(in, "orchestrationType", "DEFAULT", "CUSTOM_ORCHESTRATION")
}

func (st *bedrockAgentStore) createAgent(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	if err := st.validateAgentInput(in, ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["agentId"] = id
	rec["agentArn"] = baARN("agent/" + id)
	rec["agentVersion"] = bedrockAgentDraft
	rec["agentStatus"] = "NOT_PREPARED"
	rec["createdAt"], rec["updatedAt"] = now, now
	for k, def := range map[string]any{"idleSessionTTLInSeconds": 600, "agentResourceRoleArn": "", "orchestrationType": "DEFAULT", "agentCollaboration": "DISABLED"} {
		if _, ok := rec[k]; !ok {
			rec[k] = def
		}
	}
	a := &baAgent{
		versions: map[string]*baAgentVersion{bedrockAgentDraft: newBAAgentVersion(rec)},
		aliases: map[string]map[string]any{bedrockAgentTestAlias: {
			"agentId": id, "agentAliasId": bedrockAgentTestAlias, "agentAliasName": "AgentTestAlias",
			"agentAliasArn": baARN("agent-alias/" + id + "/" + bedrockAgentTestAlias), "description": "Test Alias for Agent",
			"routingConfiguration": []any{map[string]any{"agentVersion": bedrockAgentDraft}},
			"agentAliasStatus":     "PREPARED", "aliasInvocationState": "ACCEPT_INVOCATIONS", "createdAt": now, "updatedAt": now,
		}},
	}
	st.agents[id] = a
	st.setTags(rec["agentArn"].(string), in["tags"])
	return map[string]any{"agent": baWithStatus(rec, "agentStatus", "CREATING")}, nil
}

func newBAAgentVersion(agent map[string]any) *baAgentVersion {
	return &baAgentVersion{agent: agent, actionGroups: map[string]map[string]any{}, knowledgeBases: map[string]map[string]any{}, collaborators: map[string]map[string]any{}}
}

func (st *bedrockAgentStore) getAgent(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agent": a.draft()}, nil
}

func (st *bedrockAgentStore) updateAgent(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	if err := baRequire(in, "foundationModel", "agentResourceRoleArn"); err != nil {
		return nil, err
	}
	if err := st.validateAgentInput(in, baStr(in, "agentId")); err != nil {
		return nil, err
	}
	d := a.draft()
	baMerge(d, in, "agentId")
	a.touchDraft()
	return map[string]any{"agent": baWithStatus(d, "agentStatus", "UPDATING")}, nil
}

func (st *bedrockAgentStore) deleteAgent(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	if !baBool(q.Get("skipResourceInUseCheck")) && len(a.aliases) > 1 {
		return nil, baConflict("agent has aliases; delete them or set skipResourceInUseCheck")
	}
	id := baStr(in, "agentId")
	for _, alias := range a.aliases {
		st.forget(baStr(alias, "agentAliasArn"))
	}
	st.forget(baStr(a.draft(), "agentArn"))
	delete(st.agents, id)
	return map[string]any{"agentId": id, "agentStatus": "DELETING"}, nil
}

// forget drops the tags and resource policy of a deleted resource.
func (st *bedrockAgentStore) forget(arn string) {
	delete(st.tags, arn)
	delete(st.policies, arn)
}

func (a *baAgent) latestVersion() string {
	if a.nextVersion == 0 {
		return bedrockAgentDraft
	}
	for n := a.nextVersion; n > 0; n-- {
		if a.versions[strconv.Itoa(n)] != nil {
			return strconv.Itoa(n)
		}
	}
	return bedrockAgentDraft
}

func (st *bedrockAgentStore) listAgents(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	all := map[string]map[string]any{}
	for id, a := range st.agents {
		s := baProject(a.draft(), "agentId", "agentName", "agentStatus", "description", "updatedAt", "guardrailConfiguration", "createdAt")
		s["latestAgentVersion"] = a.latestVersion()
		all[id] = s
	}
	page, next, err := baPage(baSorted(all), in, q)
	if err != nil {
		return nil, err
	}
	for _, s := range page {
		delete(s, "createdAt")
	}
	return baList("agentSummaries", page, next), nil
}

func (st *bedrockAgentStore) prepareAgent(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	d := a.draft()
	if baStr(d, "foundationModel") == "" || baStr(d, "agentResourceRoleArn") == "" {
		return nil, baValidation("agent needs a foundationModel and an agentResourceRoleArn before it can be prepared")
	}
	now := baNow()
	d["agentStatus"] = "PREPARED"
	d["preparedAt"], d["updatedAt"] = now, now
	return map[string]any{"agentId": d["agentId"], "agentStatus": "PREPARING", "agentVersion": bedrockAgentDraft, "preparedAt": now}, nil
}

// --- versions ---

// snapshotVersion freezes the prepared DRAFT (and its sub-resources) as the next
// numbered version.
func (a *baAgent) snapshotVersion() (string, *bedrockAgentError) {
	draft := a.versions[bedrockAgentDraft]
	if baStr(draft.agent, "agentStatus") != "PREPARED" {
		return "", baValidation("prepare the agent before creating a version")
	}
	a.nextVersion++
	ver := strconv.Itoa(a.nextVersion)
	now := baNow()
	rec := baClone(draft.agent)
	rec["agentVersion"] = ver
	rec["createdAt"], rec["updatedAt"] = now, now
	v := &baAgentVersion{agent: rec, actionGroups: baCloneAll(draft.actionGroups), knowledgeBases: baCloneAll(draft.knowledgeBases), collaborators: baCloneAll(draft.collaborators)}
	for _, sub := range []map[string]map[string]any{v.actionGroups, v.knowledgeBases, v.collaborators} {
		for _, r := range sub {
			r["agentVersion"] = ver
		}
	}
	a.versions[ver] = v
	return ver, nil
}

func baAgentVersionShape(rec map[string]any) map[string]any {
	out := baClone(rec)
	out["version"] = out["agentVersion"]
	delete(out, "agentVersion")
	return out
}

func (st *bedrockAgentStore) getAgentVersion(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, v, err := st.agentVersion(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agentVersion": baAgentVersionShape(v.agent)}, nil
}

func (st *bedrockAgentStore) listAgentVersions(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	all := map[string]map[string]any{}
	for ver, v := range a.versions {
		all[ver] = baProject(v.agent, "agentName", "agentStatus", "agentVersion", "createdAt", "updatedAt", "description", "guardrailConfiguration")
	}
	page, next, err := baPage(baSorted(all), in, q)
	if err != nil {
		return nil, err
	}
	return baList("agentVersionSummaries", page, next), nil
}

func (a *baAgent) versionInUse(ver string) bool {
	for _, alias := range a.aliases {
		routes, _ := alias["routingConfiguration"].([]any)
		for _, r := range routes {
			if m, _ := r.(map[string]any); baStr(m, "agentVersion") == ver {
				return true
			}
		}
	}
	return false
}

func (st *bedrockAgentStore) deleteAgentVersion(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	if baStr(in, "agentVersion") == bedrockAgentDraft {
		return nil, baValidation("the DRAFT version can't be deleted")
	}
	a, _, err := st.agentVersion(in, false)
	if err != nil {
		return nil, err
	}
	ver := baStr(in, "agentVersion")
	if !baBool(q.Get("skipResourceInUseCheck")) && a.versionInUse(ver) {
		return nil, baConflict("agent version " + ver + " is used by an alias")
	}
	delete(a.versions, ver)
	return map[string]any{"agentId": baStr(in, "agentId"), "agentVersion": ver, "agentStatus": "DELETING"}, nil
}

// --- aliases ---

// aliasRouting validates routingConfiguration, or snapshots a new version when
// it is omitted (the API's behaviour).
func (a *baAgent) aliasRouting(in map[string]any) ([]any, *bedrockAgentError) {
	routes, present := in["routingConfiguration"].([]any)
	if !present || len(routes) == 0 {
		ver, err := a.snapshotVersion()
		if err != nil {
			return nil, err
		}
		return []any{map[string]any{"agentVersion": ver}}, nil
	}
	if len(routes) > 1 {
		return nil, baValidation("routingConfiguration accepts one entry")
	}
	m, _ := routes[0].(map[string]any)
	ver := baStr(m, "agentVersion")
	if !baNumericVersion.MatchString(ver) || a.versions[ver] == nil {
		return nil, baValidation("agent version " + ver + " does not exist")
	}
	return routes, nil
}

func (a *baAgent) checkAliasName(in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "agentAliasName"); err != nil {
		return err
	}
	for id, alias := range a.aliases {
		if id != selfID && baStr(alias, "agentAliasName") == baStr(in, "agentAliasName") {
			return baConflict("an alias named " + baStr(in, "agentAliasName") + " already exists")
		}
	}
	return nil
}

func (st *bedrockAgentStore) createAgentAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	if err := a.checkAliasName(in, ""); err != nil {
		return nil, err
	}
	routes, err := a.aliasRouting(in)
	if err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["agentAliasId"] = id
	rec["agentAliasArn"] = baARN("agent-alias/" + baStr(in, "agentId") + "/" + id)
	rec["routingConfiguration"] = routes
	rec["agentAliasStatus"] = "PREPARED"
	rec["aliasInvocationState"] = "ACCEPT_INVOCATIONS"
	rec["agentAliasHistoryEvents"] = []any{map[string]any{"routingConfiguration": routes, "startDate": now}}
	rec["createdAt"], rec["updatedAt"] = now, now
	a.aliases[id] = rec
	st.setTags(rec["agentAliasArn"].(string), in["tags"])
	return map[string]any{"agentAlias": baWithStatus(rec, "agentAliasStatus", "CREATING")}, nil
}

func (st *bedrockAgentStore) alias(in map[string]any) (*baAgent, map[string]any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, nil, err
	}
	id := baStr(in, "agentAliasId")
	if len(id) != 10 {
		return nil, nil, baValidation("agentAliasId must be 10 characters")
	}
	rec := a.aliases[id]
	if rec == nil {
		return nil, nil, baNotFound("agent alias", id)
	}
	return a, rec, nil
}

func (st *bedrockAgentStore) getAgentAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, rec, err := st.alias(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agentAlias": rec}, nil
}

func (st *bedrockAgentStore) updateAgentAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, rec, err := st.alias(in)
	if err != nil {
		return nil, err
	}
	if baStr(in, "agentAliasId") == bedrockAgentTestAlias {
		return nil, baValidation("the test alias can't be updated")
	}
	if err := a.checkAliasName(in, baStr(in, "agentAliasId")); err != nil {
		return nil, err
	}
	routes, err := a.aliasRouting(in)
	if err != nil {
		return nil, err
	}
	now := baNow()
	baMerge(rec, in, "agentId", "agentAliasId")
	rec["routingConfiguration"] = routes
	history, _ := rec["agentAliasHistoryEvents"].([]any)
	if n := len(history); n > 0 {
		if last, ok := history[n-1].(map[string]any); ok {
			last["endDate"] = now
		}
	}
	rec["agentAliasHistoryEvents"] = append(history, map[string]any{"routingConfiguration": routes, "startDate": now})
	rec["updatedAt"] = now
	return map[string]any{"agentAlias": baWithStatus(rec, "agentAliasStatus", "UPDATING")}, nil
}

func (st *bedrockAgentStore) deleteAgentAlias(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, rec, err := st.alias(in)
	if err != nil {
		return nil, err
	}
	id := baStr(in, "agentAliasId")
	if id == bedrockAgentTestAlias {
		return nil, baValidation("the test alias can't be deleted")
	}
	st.forget(baStr(rec, "agentAliasArn"))
	delete(a.aliases, id)
	return map[string]any{"agentId": baStr(in, "agentId"), "agentAliasId": id, "agentAliasStatus": "DELETING"}, nil
}

func (st *bedrockAgentStore) listAgentAliases(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	a, err := st.agent(in)
	if err != nil {
		return nil, err
	}
	all := map[string]map[string]any{}
	for id, alias := range a.aliases {
		all[id] = baProject(alias, "agentAliasId", "agentAliasName", "description", "routingConfiguration", "agentAliasStatus", "createdAt", "updatedAt", "aliasInvocationState")
	}
	page, next, err := baPage(baSorted(all), in, q)
	if err != nil {
		return nil, err
	}
	return baList("agentAliasSummaries", page, next), nil
}

// --- action groups ---

func baUniqueName(records map[string]map[string]any, key, name, selfID string) *bedrockAgentError {
	for id, r := range records {
		if id != selfID && baStr(r, key) == name {
			return baConflict(key + " " + name + " already exists")
		}
	}
	return nil
}

func validateActionGroup(in map[string]any) *bedrockAgentError {
	if err := baCheckName(in, "actionGroupName"); err != nil {
		return err
	}
	if in["apiSchema"] != nil && in["functionSchema"] != nil {
		return baValidation("specify apiSchema or functionSchema, not both")
	}
	if err := baCheckEnum(in, "actionGroupState", "ENABLED", "DISABLED"); err != nil {
		return err
	}
	return baCheckEnum(in, "parentActionGroupSignature", "AMAZON.UserInput", "AMAZON.CodeInterpreter", "ANTHROPIC.Computer", "ANTHROPIC.Bash", "ANTHROPIC.TextEditor")
}

func (st *bedrockAgentStore) createActionGroup(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, true)
	if err != nil {
		return nil, err
	}
	if err := validateActionGroup(in); err != nil {
		return nil, err
	}
	if err := baUniqueName(v.actionGroups, "actionGroupName", baStr(in, "actionGroupName"), ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["actionGroupId"] = id
	if rec["actionGroupState"] == nil {
		rec["actionGroupState"] = "ENABLED"
	}
	rec["createdAt"], rec["updatedAt"] = now, now
	v.actionGroups[id] = rec
	a.touchDraft()
	return map[string]any{"agentActionGroup": rec}, nil
}

func (st *bedrockAgentStore) actionGroup(in map[string]any, draftOnly bool) (*baAgent, *baAgentVersion, map[string]any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, draftOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := baCheckID(in, "actionGroupId"); err != nil {
		return nil, nil, nil, err
	}
	rec := v.actionGroups[baStr(in, "actionGroupId")]
	if rec == nil {
		return nil, nil, nil, baNotFound("action group", baStr(in, "actionGroupId"))
	}
	return a, v, rec, nil
}

func (st *bedrockAgentStore) getActionGroup(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, _, rec, err := st.actionGroup(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agentActionGroup": rec}, nil
}

func (st *bedrockAgentStore) updateActionGroup(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, rec, err := st.actionGroup(in, true)
	if err != nil {
		return nil, err
	}
	if err := validateActionGroup(in); err != nil {
		return nil, err
	}
	if err := baUniqueName(v.actionGroups, "actionGroupName", baStr(in, "actionGroupName"), baStr(in, "actionGroupId")); err != nil {
		return nil, err
	}
	baMerge(rec, in)
	rec["updatedAt"] = baNow()
	a.touchDraft()
	return map[string]any{"agentActionGroup": rec}, nil
}

func (st *bedrockAgentStore) deleteActionGroup(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, _, err := st.actionGroup(in, true)
	if err != nil {
		return nil, err
	}
	delete(v.actionGroups, baStr(in, "actionGroupId"))
	a.touchDraft()
	return map[string]any{}, nil
}

func (st *bedrockAgentStore) listActionGroups(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, v, err := st.agentVersion(in, false)
	if err != nil {
		return nil, err
	}
	return baListProjected(v.actionGroups, in, q, "actionGroupSummaries", "actionGroupId", "actionGroupName", "actionGroupState", "description", "updatedAt")
}

// baListProjected pages records (ordered by createdAt) and projects each to a summary.
func baListProjected(records map[string]map[string]any, in map[string]any, q url.Values, key string, fields ...string) (any, *bedrockAgentError) {
	page, next, err := baPage(baSorted(records), in, q)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(page))
	for i, r := range page {
		out[i] = baProject(r, fields...)
	}
	return baList(key, out, next), nil
}

// --- knowledge-base associations ---

func (st *bedrockAgentStore) associateKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, true)
	if err != nil {
		return nil, err
	}
	if err := baRequire(in, "description"); err != nil {
		return nil, err
	}
	if err := baCheckID(in, "knowledgeBaseId"); err != nil {
		return nil, err
	}
	if err := baCheckEnum(in, "knowledgeBaseState", "ENABLED", "DISABLED"); err != nil {
		return nil, err
	}
	kbID := baStr(in, "knowledgeBaseId")
	if st.kbs[kbID] == nil {
		return nil, baNotFound("knowledge base", kbID)
	}
	if v.knowledgeBases[kbID] != nil {
		return nil, baConflict("knowledge base " + kbID + " is already associated with the agent")
	}
	now := baNow()
	rec := baRecord(in)
	if rec["knowledgeBaseState"] == nil {
		rec["knowledgeBaseState"] = "ENABLED"
	}
	rec["createdAt"], rec["updatedAt"] = now, now
	v.knowledgeBases[kbID] = rec
	a.touchDraft()
	return map[string]any{"agentKnowledgeBase": rec}, nil
}

func (st *bedrockAgentStore) agentKnowledgeBase(in map[string]any, draftOnly bool) (*baAgent, *baAgentVersion, map[string]any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, draftOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := baCheckID(in, "knowledgeBaseId"); err != nil {
		return nil, nil, nil, err
	}
	rec := v.knowledgeBases[baStr(in, "knowledgeBaseId")]
	if rec == nil {
		return nil, nil, nil, baNotFound("agent knowledge base", baStr(in, "knowledgeBaseId"))
	}
	return a, v, rec, nil
}

func (st *bedrockAgentStore) getAgentKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, _, rec, err := st.agentKnowledgeBase(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agentKnowledgeBase": rec}, nil
}

func (st *bedrockAgentStore) updateAgentKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, _, rec, err := st.agentKnowledgeBase(in, true)
	if err != nil {
		return nil, err
	}
	if err := baCheckEnum(in, "knowledgeBaseState", "ENABLED", "DISABLED"); err != nil {
		return nil, err
	}
	baMerge(rec, in)
	rec["updatedAt"] = baNow()
	a.touchDraft()
	return map[string]any{"agentKnowledgeBase": rec}, nil
}

func (st *bedrockAgentStore) disassociateKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, _, err := st.agentKnowledgeBase(in, true)
	if err != nil {
		return nil, err
	}
	delete(v.knowledgeBases, baStr(in, "knowledgeBaseId"))
	a.touchDraft()
	return map[string]any{}, nil
}

func (st *bedrockAgentStore) listAgentKnowledgeBases(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, v, err := st.agentVersion(in, false)
	if err != nil {
		return nil, err
	}
	return baListProjected(v.knowledgeBases, in, q, "agentKnowledgeBaseSummaries", "knowledgeBaseId", "description", "knowledgeBaseState", "updatedAt")
}

// --- collaborators ---

func (st *bedrockAgentStore) validateCollaborator(a *baAgent, in map[string]any) *bedrockAgentError {
	if err := baRequire(in, "agentDescriptor", "collaborationInstruction"); err != nil {
		return err
	}
	if err := baCheckName(in, "collaboratorName"); err != nil {
		return err
	}
	if err := baCheckEnum(in, "relayConversationHistory", "TO_COLLABORATOR", "DISABLED"); err != nil {
		return err
	}
	switch baStr(a.draft(), "agentCollaboration") {
	case "SUPERVISOR", "SUPERVISOR_ROUTER":
	default:
		return baValidation("agentCollaboration must be SUPERVISOR or SUPERVISOR_ROUTER to add collaborators")
	}
	aliasArn := baSubStr(in, "agentDescriptor", "aliasArn")
	if aliasArn == "" {
		return baValidation("agentDescriptor.aliasArn is required")
	}
	_, rest, _ := strings.Cut(aliasArn, ":agent-alias/")
	agentID, aliasID, _ := strings.Cut(rest, "/")
	if agentID == baStr(in, "agentId") {
		return baValidation("an agent can't collaborate with itself")
	}
	if target := st.agents[agentID]; target == nil || target.aliases[aliasID] == nil {
		return baNotFound("agent alias", aliasArn)
	}
	return nil
}

func (st *bedrockAgentStore) associateCollaborator(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, true)
	if err != nil {
		return nil, err
	}
	if err := st.validateCollaborator(a, in); err != nil {
		return nil, err
	}
	if err := baUniqueName(v.collaborators, "collaboratorName", baStr(in, "collaboratorName"), ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["collaboratorId"] = id
	if rec["relayConversationHistory"] == nil {
		rec["relayConversationHistory"] = "DISABLED"
	}
	rec["createdAt"], rec["lastUpdatedAt"] = now, now
	v.collaborators[id] = rec
	a.touchDraft()
	return map[string]any{"agentCollaborator": rec}, nil
}

func (st *bedrockAgentStore) collaborator(in map[string]any, draftOnly bool) (*baAgent, *baAgentVersion, map[string]any, *bedrockAgentError) {
	a, v, err := st.agentVersion(in, draftOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := baCheckID(in, "collaboratorId"); err != nil {
		return nil, nil, nil, err
	}
	rec := v.collaborators[baStr(in, "collaboratorId")]
	if rec == nil {
		return nil, nil, nil, baNotFound("agent collaborator", baStr(in, "collaboratorId"))
	}
	return a, v, rec, nil
}

func (st *bedrockAgentStore) getCollaborator(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, _, rec, err := st.collaborator(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"agentCollaborator": rec}, nil
}

func (st *bedrockAgentStore) updateCollaborator(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, rec, err := st.collaborator(in, true)
	if err != nil {
		return nil, err
	}
	if err := st.validateCollaborator(a, in); err != nil {
		return nil, err
	}
	if err := baUniqueName(v.collaborators, "collaboratorName", baStr(in, "collaboratorName"), baStr(in, "collaboratorId")); err != nil {
		return nil, err
	}
	baMerge(rec, in)
	rec["lastUpdatedAt"] = baNow()
	a.touchDraft()
	return map[string]any{"agentCollaborator": rec}, nil
}

func (st *bedrockAgentStore) disassociateCollaborator(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	a, v, _, err := st.collaborator(in, true)
	if err != nil {
		return nil, err
	}
	delete(v.collaborators, baStr(in, "collaboratorId"))
	a.touchDraft()
	return map[string]any{}, nil
}

func (st *bedrockAgentStore) listCollaborators(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, v, err := st.agentVersion(in, false)
	if err != nil {
		return nil, err
	}
	return baListProjected(v.collaborators, in, q, "agentCollaboratorSummaries", "agentId", "agentVersion", "collaboratorId", "agentDescriptor", "collaborationInstruction", "relayConversationHistory", "collaboratorName", "createdAt", "lastUpdatedAt")
}
