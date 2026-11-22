package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const bedrockAgentTestInstruction = "You are a helpful agent that answers questions about Stackyard."

func baObj(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("missing object %q in %v", key, m)
	}
	return v
}

func baWant(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	status, out := bedrockAgentCall(t, ts, method, path, body)
	if status != want {
		t.Fatalf("%s %s: expected %d, got %d: %v", method, path, want, status, out)
	}
	return out
}

// createPreparedAgent makes an agent that can be prepared and versioned.
func createPreparedAgent(t *testing.T, ts *httptest.Server, name, collaboration string) string {
	t.Helper()
	out := baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/", fmt.Sprintf(`{"agentName":%q,"foundationModel":"anthropic.claude-v2","agentResourceRoleArn":"arn:aws:iam::123456789012:role/agent","instruction":%q,"agentCollaboration":%q}`, name, bedrockAgentTestInstruction, collaboration))
	id := baObj(t, out, "agent")["agentId"].(string)
	baWant(t, ts, http.StatusAccepted, http.MethodPost, "/agents/"+id+"/", "")
	return id
}

func TestBedrockAgentStage1AgentLifecycle(t *testing.T) {
	ts := newBedrockAgentTestServer(t)

	out := baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/", `{"agentName":"support","tags":{"team":"core"}}`)
	agent := baObj(t, out, "agent")
	id := agent["agentId"].(string)
	if agent["agentStatus"] != "CREATING" || agent["agentVersion"] != "DRAFT" || agent["idleSessionTTLInSeconds"] != float64(600) {
		t.Fatalf("unexpected agent: %v", agent)
	}
	got := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/agents/"+id+"/", ""), "agent")
	if got["agentStatus"] != "NOT_PREPARED" {
		t.Fatalf("a new agent is NOT_PREPARED, got %v", got["agentStatus"])
	}

	// PrepareAgent needs a model and role.
	baWant(t, ts, http.StatusBadRequest, http.MethodPost, "/agents/"+id+"/", "")
	baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/"+id+"/", fmt.Sprintf(`{"agentName":"support","foundationModel":"anthropic.claude-v2","agentResourceRoleArn":"arn:aws:iam::123456789012:role/agent","instruction":%q}`, bedrockAgentTestInstruction))
	prep := baWant(t, ts, http.StatusAccepted, http.MethodPost, "/agents/"+id+"/", "")
	if prep["agentStatus"] != "PREPARING" || prep["preparedAt"] == nil {
		t.Fatalf("unexpected prepare reply: %v", prep)
	}
	if s := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/agents/"+id+"/", ""), "agent")["agentStatus"]; s != "PREPARED" {
		t.Fatalf("expected PREPARED, got %v", s)
	}

	// An alias without routing snapshots version 1; the test alias is listed too.
	alias := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/"+id+"/agentaliases/", `{"agentAliasName":"live"}`), "agentAlias")
	routes := alias["routingConfiguration"].([]any)
	if routes[0].(map[string]any)["agentVersion"] != "1" {
		t.Fatalf("expected version 1 routing, got %v", routes)
	}
	aliases := baWant(t, ts, http.StatusOK, http.MethodPost, "/agents/"+id+"/agentaliases/", `{}`)["agentAliasSummaries"].([]any)
	if len(aliases) != 2 {
		t.Fatalf("expected test alias plus live, got %v", aliases)
	}
	ver := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/agents/"+id+"/agentversions/1/", ""), "agentVersion")
	if ver["version"] != "1" || ver["agentStatus"] != "PREPARED" {
		t.Fatalf("unexpected version: %v", ver)
	}
	versions := baWant(t, ts, http.StatusOK, http.MethodPost, "/agents/"+id+"/agentversions/", `{}`)["agentVersionSummaries"].([]any)
	if len(versions) != 2 {
		t.Fatalf("expected DRAFT and 1, got %v", versions)
	}
	summaries := baWant(t, ts, http.StatusOK, http.MethodPost, "/agents/", `{}`)["agentSummaries"].([]any)
	if s := summaries[0].(map[string]any); s["latestAgentVersion"] != "1" {
		t.Fatalf("expected latestAgentVersion 1, got %v", s)
	}

	// Editing the draft un-prepares it.
	baWant(t, ts, http.StatusOK, http.MethodPut, "/agents/"+id+"/agentversions/DRAFT/actiongroups/", `{"actionGroupName":"lookup","parentActionGroupSignature":"AMAZON.UserInput"}`)
	if s := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/agents/"+id+"/", ""), "agent")["agentStatus"]; s != "NOT_PREPARED" {
		t.Fatalf("expected NOT_PREPARED after an edit, got %v", s)
	}

	// Tags on the agent ARN.
	arn := url.PathEscape(agent["agentArn"].(string))
	baWant(t, ts, http.StatusOK, http.MethodPost, "/tags/"+arn, `{"tags":{"env":"dev"}}`)
	tags := baWant(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any)
	if tags["team"] != "core" || tags["env"] != "dev" {
		t.Fatalf("unexpected tags: %v", tags)
	}
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/tags/"+arn+"?tagKeys=team", "")
	if tags := baWant(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); len(tags) != 1 {
		t.Fatalf("expected one tag after untag, got %v", tags)
	}

	// Version in use, then agent in use.
	baWant(t, ts, http.StatusConflict, http.MethodDelete, "/agents/"+id+"/agentversions/1/", "")
	baWant(t, ts, http.StatusConflict, http.MethodDelete, "/agents/"+id+"/", "")
	aliasID := alias["agentAliasId"].(string)
	if del := baWant(t, ts, http.StatusAccepted, http.MethodDelete, "/agents/"+id+"/agentaliases/"+aliasID+"/", ""); del["agentAliasStatus"] != "DELETING" {
		t.Fatalf("unexpected alias delete: %v", del)
	}
	baWant(t, ts, http.StatusAccepted, http.MethodDelete, "/agents/"+id+"/agentversions/1/", "")
	if del := baWant(t, ts, http.StatusAccepted, http.MethodDelete, "/agents/"+id+"/", ""); del["agentStatus"] != "DELETING" {
		t.Fatalf("unexpected agent delete: %v", del)
	}
	baWant(t, ts, http.StatusNotFound, http.MethodGet, "/agents/"+id+"/", "")
	baWant(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+arn, "")
}

func TestBedrockAgentStage1SubResourcesAndVersions(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	helper := createPreparedAgent(t, ts, "helper", "DISABLED")
	id := createPreparedAgent(t, ts, "supervisor", "SUPERVISOR")
	draft := "/agents/" + id + "/agentversions/DRAFT"

	ag := baObj(t, baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/actiongroups/", `{"actionGroupName":"orders","functionSchema":{"functions":[{"name":"get_order"}]}}`), "agentActionGroup")
	agID := ag["actionGroupId"].(string)
	if ag["actionGroupState"] != "ENABLED" {
		t.Fatalf("action groups default to ENABLED: %v", ag)
	}
	baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/actiongroups/"+agID+"/", `{"actionGroupName":"orders","actionGroupState":"DISABLED"}`)

	kb := createKnowledgeBase(t, ts, "docs")
	baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/knowledgebases/", fmt.Sprintf(`{"knowledgeBaseId":%q,"description":"product docs"}`, kb))
	baWant(t, ts, http.StatusConflict, http.MethodPut, draft+"/knowledgebases/", fmt.Sprintf(`{"knowledgeBaseId":%q,"description":"again"}`, kb))

	collab := baObj(t, baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/agentcollaborators/", fmt.Sprintf(`{"agentDescriptor":{"aliasArn":"arn:aws:bedrock:us-east-1:123456789012:agent-alias/%s/TSTALIASID"},"collaboratorName":"helper","collaborationInstruction":"ask the helper"}`, helper)), "agentCollaborator")
	if collab["relayConversationHistory"] != "DISABLED" {
		t.Fatalf("relayConversationHistory defaults to DISABLED: %v", collab)
	}

	// Snapshot version 1 and confirm it froze the sub-resources.
	baWant(t, ts, http.StatusAccepted, http.MethodPost, "/agents/"+id+"/", "")
	baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/"+id+"/agentaliases/", `{"agentAliasName":"v1"}`)
	baWant(t, ts, http.StatusNoContent, http.MethodDelete, draft+"/actiongroups/"+agID+"/", "")
	v1 := "/agents/" + id + "/agentversions/1"
	groups := baWant(t, ts, http.StatusOK, http.MethodPost, v1+"/actiongroups/", `{}`)["actionGroupSummaries"].([]any)
	if len(groups) != 1 || groups[0].(map[string]any)["actionGroupState"] != "DISABLED" {
		t.Fatalf("version 1 should keep its action group: %v", groups)
	}
	if got := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, v1+"/actiongroups/"+agID+"/", ""), "agentActionGroup"); got["agentVersion"] != "1" {
		t.Fatalf("version snapshot should carry agentVersion 1: %v", got)
	}
	if n := len(baWant(t, ts, http.StatusOK, http.MethodPost, draft+"/actiongroups/", `{}`)["actionGroupSummaries"].([]any)); n != 0 {
		t.Fatalf("the draft group was deleted, got %d", n)
	}
	if n := len(baWant(t, ts, http.StatusOK, http.MethodPost, v1+"/knowledgebases/", `{}`)["agentKnowledgeBaseSummaries"].([]any)); n != 1 {
		t.Fatalf("version 1 should keep its knowledge base, got %d", n)
	}
	if n := len(baWant(t, ts, http.StatusOK, http.MethodPost, v1+"/agentcollaborators/", `{}`)["agentCollaboratorSummaries"].([]any)); n != 1 {
		t.Fatalf("version 1 should keep its collaborator, got %d", n)
	}
	// Versions are read-only.
	baWant(t, ts, http.StatusBadRequest, http.MethodPut, v1+"/actiongroups/", `{"actionGroupName":"late"}`)

	baWant(t, ts, http.StatusNoContent, http.MethodDelete, draft+"/knowledgebases/"+kb+"/", "")
	baWant(t, ts, http.StatusNoContent, http.MethodDelete, draft+"/agentcollaborators/"+collab["collaboratorId"].(string)+"/", "")
}

func createKnowledgeBase(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	out := baWant(t, ts, http.StatusAccepted, http.MethodPut, "/knowledgebases/", fmt.Sprintf(`{"name":%q,"roleArn":"arn:aws:iam::123456789012:role/kb","knowledgeBaseConfiguration":{"type":"VECTOR","vectorKnowledgeBaseConfiguration":{"embeddingModelArn":"arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"}},"storageConfiguration":{"type":"S3_VECTORS"}}`, name))
	kb := baObj(t, out, "knowledgeBase")
	if kb["status"] != "CREATING" {
		t.Fatalf("CreateKnowledgeBase replies CREATING: %v", kb)
	}
	return kb["knowledgeBaseId"].(string)
}

func TestBedrockAgentStage1KnowledgeBaseIngestion(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	kb := createKnowledgeBase(t, ts, "manuals")
	if s := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/knowledgebases/"+kb, ""), "knowledgeBase")["status"]; s != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %v", s)
	}
	ds := baObj(t, baWant(t, ts, http.StatusOK, http.MethodPut, "/knowledgebases/"+kb+"/datasources/", `{"name":"inline","dataSourceConfiguration":{"type":"CUSTOM"}}`), "dataSource")
	dsPath := "/knowledgebases/" + kb + "/datasources/" + ds["dataSourceId"].(string)

	ingest := baWant(t, ts, http.StatusAccepted, http.MethodPut, dsPath+"/documents", `{"documents":[{"content":{"dataSourceType":"CUSTOM","custom":{"customDocumentIdentifier":{"id":"doc1"},"sourceType":"IN_LINE","inlineContent":{"type":"TEXT","textContent":{"data":"hello"}}}}}]}`)
	if d := ingest["documentDetails"].([]any)[0].(map[string]any); d["status"] != "STARTING" {
		t.Fatalf("unexpected ingest reply: %v", d)
	}
	got := baWant(t, ts, http.StatusOK, http.MethodPost, dsPath+"/documents/getDocuments", `{"documentIdentifiers":[{"dataSourceType":"CUSTOM","custom":{"id":"doc1"}},{"dataSourceType":"CUSTOM","custom":{"id":"missing"}}]}`)["documentDetails"].([]any)
	if got[0].(map[string]any)["status"] != "INDEXED" || got[1].(map[string]any)["status"] != "NOT_FOUND" {
		t.Fatalf("unexpected document details: %v", got)
	}
	// An S3 document can't go into a CUSTOM data source.
	baWant(t, ts, http.StatusBadRequest, http.MethodPut, dsPath+"/documents", `{"documents":[{"content":{"dataSourceType":"S3","s3":{"s3Location":{"uri":"s3://b/k"}}}}]}`)

	// Jobs run until they are next read, so a fresh job can be stopped.
	job := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, dsPath+"/ingestionjobs/", `{"description":"first"}`), "ingestionJob")
	if job["status"] != "STARTING" {
		t.Fatalf("StartIngestionJob replies STARTING: %v", job)
	}
	jobPath := dsPath + "/ingestionjobs/" + job["ingestionJobId"].(string)
	done := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, jobPath, ""), "ingestionJob")
	if done["status"] != "COMPLETE" {
		t.Fatalf("expected COMPLETE, got %v", done)
	}
	baWant(t, ts, http.StatusConflict, http.MethodPost, jobPath+"/stop", "")
	second := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, dsPath+"/ingestionjobs/", `{}`), "ingestionJob")
	stopped := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPost, dsPath+"/ingestionjobs/"+second["ingestionJobId"].(string)+"/stop", ""), "ingestionJob")
	if stopped["status"] != "STOPPING" {
		t.Fatalf("StopIngestionJob replies STOPPING: %v", stopped)
	}
	only := baWant(t, ts, http.StatusOK, http.MethodPost, dsPath+"/ingestionjobs/", `{"filters":[{"attribute":"STATUS","operator":"EQ","values":["STOPPED"]}]}`)["ingestionJobSummaries"].([]any)
	if len(only) != 1 {
		t.Fatalf("expected one STOPPED job, got %v", only)
	}

	page := baWant(t, ts, http.StatusOK, http.MethodPost, dsPath+"/ingestionjobs/", `{"maxResults":1}`)
	if page["nextToken"] == nil || len(page["ingestionJobSummaries"].([]any)) != 1 {
		t.Fatalf("expected a one-item page with a token: %v", page)
	}

	vpc := baWant(t, ts, http.StatusAccepted, http.MethodPost, "/knowledgebases/"+kb+"/vpcconfigurations/", `{"vpcId":"vpc-1","subnetIds":["subnet-1"],"resourceTarget":"db.internal","port":443,"protocol":"HTTPS","resolutionMode":"IN_VPC"}`)
	vpcID := vpc["vpcConfigurationId"].(string)
	if len(vpcID) != 32 || vpc["status"] != "CREATING" {
		t.Fatalf("unexpected VPC configuration: %v", vpc)
	}
	if s := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/knowledgebases/"+kb+"/vpcconfigurations/"+vpcID, ""), "vpcConfiguration")["status"]; s != "CREATED" {
		t.Fatalf("expected CREATED, got %v", s)
	}
	baWant(t, ts, http.StatusAccepted, http.MethodDelete, "/knowledgebases/"+kb+"/vpcconfigurations/"+vpcID, "")

	baWant(t, ts, http.StatusAccepted, http.MethodPost, dsPath+"/documents/deleteDocuments", `{"documentIdentifiers":[{"dataSourceType":"CUSTOM","custom":{"id":"doc1"}}]}`)
	if docs := baWant(t, ts, http.StatusOK, http.MethodPost, dsPath+"/documents", `{}`)["documentDetails"].([]any); len(docs) != 0 {
		t.Fatalf("expected no documents, got %v", docs)
	}
	baWant(t, ts, http.StatusAccepted, http.MethodDelete, dsPath, "")
	baWant(t, ts, http.StatusAccepted, http.MethodDelete, "/knowledgebases/"+kb, "")
	baWant(t, ts, http.StatusNotFound, http.MethodGet, "/knowledgebases/"+kb, "")
}

const bedrockAgentTestFlow = `{"nodes":[{"name":"in","type":"Input"},{"name":"out","type":"Output"}],"connections":[{"name":"c","type":"Data","source":"in","target":"out"}]}`

func TestBedrockAgentStage1FlowsPromptsAndInstructions(t *testing.T) {
	ts := newBedrockAgentTestServer(t)

	bad := baWant(t, ts, http.StatusOK, http.MethodPost, "/flows/validate-definition", `{"definition":{"nodes":[{"name":"in","type":"Input"}],"connections":[{"name":"c","type":"Data","source":"in","target":"nowhere"}]}}`)["validations"].([]any)
	if len(bad) != 2 {
		t.Fatalf("expected missing-output and unknown-target errors, got %v", bad)
	}
	good := baWant(t, ts, http.StatusOK, http.MethodPost, "/flows/validate-definition", `{"definition":`+bedrockAgentTestFlow+`}`)["validations"].([]any)
	if len(good) != 0 {
		t.Fatalf("a valid flow has no validations, got %v", good)
	}

	flow := baWant(t, ts, http.StatusCreated, http.MethodPost, "/flows/", `{"name":"echo","executionRoleArn":"arn:aws:iam::123456789012:role/flow"}`)
	flowID := flow["id"].(string)
	baWant(t, ts, http.StatusBadRequest, http.MethodPost, "/flows/"+flowID+"/", "")
	baWant(t, ts, http.StatusBadRequest, http.MethodPost, "/flows/"+flowID+"/versions", `{}`)
	baWant(t, ts, http.StatusOK, http.MethodPut, "/flows/"+flowID+"/", `{"name":"echo","executionRoleArn":"arn:aws:iam::123456789012:role/flow","definition":`+bedrockAgentTestFlow+`}`)
	if p := baWant(t, ts, http.StatusAccepted, http.MethodPost, "/flows/"+url.PathEscape(flow["arn"].(string))+"/", ""); p["status"] != "Preparing" {
		t.Fatalf("unexpected PrepareFlow reply: %v", p)
	}
	v := baWant(t, ts, http.StatusCreated, http.MethodPost, "/flows/"+flowID+"/versions", `{"description":"first"}`)
	if v["version"] != "1" || v["status"] != "Prepared" {
		t.Fatalf("unexpected flow version: %v", v)
	}
	alias := baWant(t, ts, http.StatusCreated, http.MethodPost, "/flows/"+flowID+"/aliases", `{"name":"prod","routingConfiguration":[{"flowVersion":"1"}]}`)
	aliasID := alias["id"].(string)
	if alias["flowId"] != flowID {
		t.Fatalf("unexpected alias: %v", alias)
	}
	baWant(t, ts, http.StatusBadRequest, http.MethodPut, "/flows/"+flowID+"/aliases/"+aliasID, `{"name":"prod","routingConfiguration":[{"flowVersion":"9"}]}`)
	baWant(t, ts, http.StatusConflict, http.MethodDelete, "/flows/"+flowID+"/versions/1/", "")
	baWant(t, ts, http.StatusConflict, http.MethodDelete, "/flows/"+flowID+"/", "")
	if n := len(baWant(t, ts, http.StatusOK, http.MethodGet, "/flows/"+flowID+"/versions", "")["flowVersionSummaries"].([]any)); n != 2 {
		t.Fatalf("expected DRAFT and 1, got %d", n)
	}
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/flows/"+flowID+"/aliases/"+aliasID, "")
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/flows/"+flowID+"/", "")

	prompt := baWant(t, ts, http.StatusCreated, http.MethodPost, "/prompts/", `{"name":"greeting","defaultVariant":"v","variants":[{"name":"v","templateType":"TEXT","templateConfiguration":{"text":{"text":"Hi {{name}}"}}}]}`)
	promptID := prompt["id"].(string)
	pv := baWant(t, ts, http.StatusCreated, http.MethodPost, "/prompts/"+promptID+"/versions", `{}`)
	if pv["version"] != "1" || pv["arn"] != prompt["arn"].(string)+":1" {
		t.Fatalf("unexpected prompt version: %v", pv)
	}
	if got := baWant(t, ts, http.StatusOK, http.MethodGet, "/prompts/"+url.PathEscape(pv["arn"].(string))+"/", ""); got["version"] != "1" {
		t.Fatalf("an ARN:version identifier selects that version: %v", got)
	}
	if n := len(baWant(t, ts, http.StatusOK, http.MethodGet, "/prompts/?promptIdentifier="+promptID, "")["promptSummaries"].([]any)); n != 2 {
		t.Fatalf("expected DRAFT and 1, got %d", n)
	}
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/prompts/"+promptID+"/?promptVersion=1", "")
	baWant(t, ts, http.StatusNotFound, http.MethodGet, "/prompts/"+promptID+"/?promptVersion=1", "")

	ri := baWant(t, ts, http.StatusAccepted, http.MethodPost, "/retrievalinstructions/", `{"name":"concise","instruction":{"answerFormat":"Two sentences."}}`)
	riID := ri["retrievalInstructionId"].(string)
	if ri["status"] != "ACTIVE" || ri["lastTransitionStatus"] != "CREATING" {
		t.Fatalf("unexpected retrieval instruction: %v", ri)
	}
	baWant(t, ts, http.StatusCreated, http.MethodPost, "/retrievalinstructions/"+riID+"/versions", `{}`)
	if got := baWant(t, ts, http.StatusOK, http.MethodGet, "/retrievalinstructions/"+riID, ""); got["latestVersion"] != "1" {
		t.Fatalf("expected latestVersion 1: %v", got)
	}
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/retrievalinstructions/"+riID, "")

	// Resource policy with optimistic revisions.
	arn := url.PathEscape(prompt["arn"].(string))
	put := baWant(t, ts, http.StatusOK, http.MethodPut, "/resourcepolicy/"+arn, `{"policy":"{\"Version\":\"2012-10-17\"}"}`)
	baWant(t, ts, http.StatusConflict, http.MethodPut, "/resourcepolicy/"+arn, `{"policy":"{}","expectedRevisionId":"stale"}`)
	baWant(t, ts, http.StatusOK, http.MethodPut, "/resourcepolicy/"+arn, fmt.Sprintf(`{"policy":"{}","expectedRevisionId":%q}`, put["revisionId"]))
	baWant(t, ts, http.StatusOK, http.MethodGet, "/resourcepolicy/"+arn, "")
	baWant(t, ts, http.StatusOK, http.MethodDelete, "/resourcepolicy/"+arn, "")
	baWant(t, ts, http.StatusNotFound, http.MethodGet, "/resourcepolicy/"+arn, "")
}
