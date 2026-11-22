package server

import (
	"net/url"
	"sort"
)

// Read-only lookups for other services (Bedrock Agent Runtime). Each returns
// deep copies, so callers can't change stored state or race with it.

// bedrockAgentSnapshot is one agent version with its sub-resources.
type bedrockAgentSnapshot struct {
	Agent          map[string]any
	ActionGroups   []map[string]any
	KnowledgeBases []map[string]any
	Collaborators  []map[string]any
}

// LookupAgentAlias resolves an agent alias (TSTALIASID routes to DRAFT) to the
// agent version it serves.
func (st *bedrockAgentStore) LookupAgentAlias(agentID, aliasID string) (bedrockAgentSnapshot, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a := st.agents[agentID]
	if a == nil || a.aliases[aliasID] == nil {
		return bedrockAgentSnapshot{}, false
	}
	routes, _ := a.aliases[aliasID]["routingConfiguration"].([]any)
	ver := bedrockAgentDraft
	if len(routes) > 0 {
		if m, _ := routes[0].(map[string]any); baStr(m, "agentVersion") != "" {
			ver = baStr(m, "agentVersion")
		}
	}
	v := a.versions[ver]
	if v == nil {
		return bedrockAgentSnapshot{}, false
	}
	return bedrockAgentSnapshot{
		Agent:          baClone(v.agent),
		ActionGroups:   baClonedList(v.actionGroups),
		KnowledgeBases: baClonedList(v.knowledgeBases),
		Collaborators:  baClonedList(v.collaborators),
	}, true
}

// LookupKnowledgeBase returns a knowledge base and every document ingested
// into its data sources (each record keeps its "content" for retrieval).
func (st *bedrockAgentStore) LookupKnowledgeBase(id string) (map[string]any, []map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	kb := st.kbs[id]
	if kb == nil {
		return nil, nil, false
	}
	var docs []map[string]any
	for _, ds := range kb.dataSources {
		docs = append(docs, baClonedList(ds.docs)...)
	}
	return baClone(kb.kb), docs, true
}

// LookupFlowAlias resolves a flow alias (ID, ARN or TSTALIASID for DRAFT) to the
// flow version it routes to.
func (st *bedrockAgentStore) LookupFlowAlias(flowIdentifier, aliasIdentifier string) (map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, f, err := st.flow(map[string]any{"flowIdentifier": flowIdentifier})
	if err != nil {
		return nil, false
	}
	ver := bedrockAgentDraft
	if aliasIdentifier != bedrockAgentTestAlias {
		m := baFlowAliasID.FindStringSubmatch(aliasIdentifier)
		if m == nil || f.aliases[m[1]] == nil {
			return nil, false
		}
		routes, _ := f.aliases[m[1]]["routingConfiguration"].([]any)
		if len(routes) > 0 {
			r, _ := routes[0].(map[string]any)
			ver = baStr(r, "flowVersion")
		}
	}
	rec := f.versions[ver]
	if rec == nil {
		return nil, false
	}
	return baClone(rec), true
}

// LookupPrompt resolves a prompt ID or ARN (ARN:version picks a version).
func (st *bedrockAgentStore) LookupPrompt(identifier string) (map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, p, ver, err := st.prompt(map[string]any{"promptIdentifier": identifier}, url.Values{})
	if err != nil {
		return nil, false
	}
	if ver == "" {
		ver = bedrockAgentDraft
	}
	rec := p.versions[ver]
	if rec == nil {
		return nil, false
	}
	return baClone(rec), true
}

func baClonedList(m map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, len(keys))
	for i, k := range keys {
		out[i] = baClone(m[k])
	}
	return out
}
