package server

type bedrockAgentCoreDataOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Bedrock AgentCore Data Plane actions sourced from:
// https://docs.aws.amazon.com/bedrock-agentcore/latest/APIReference/API_Operations.html
var bedrockAgentCoreDataOperations = []bedrockAgentCoreDataOperation{
	{Name: "BatchCreateMemoryRecords", Method: "POST", URI: "/memories/{memoryId}/memoryRecords/batchCreate"},
	{Name: "BatchDeleteMemoryRecords", Method: "POST", URI: "/memories/{memoryId}/memoryRecords/batchDelete"},
	{Name: "BatchUpdateMemoryRecords", Method: "POST", URI: "/memories/{memoryId}/memoryRecords/batchUpdate"},
	{Name: "CompleteResourceTokenAuth", Method: "POST", URI: "/identities/CompleteResourceTokenAuth"},
	{Name: "CreateEvent", Method: "POST", URI: "/memories/{memoryId}/events"},
	{Name: "DeleteEvent", Method: "DELETE", URI: "/memories/{memoryId}/actor/{actorId}/sessions/{sessionId}/events/{eventId}"},
	{Name: "DeleteMemoryRecord", Method: "DELETE", URI: "/memories/{memoryId}/memoryRecords/{memoryRecordId}"},
	{Name: "Evaluate", Method: "POST", URI: "/evaluations/evaluate/{evaluatorId}"},
	{Name: "GetAgentCard", Method: "GET", URI: "/runtimes/{agentRuntimeArn}/invocations/.well-known/agent-card.json?qualifier={qualifier}"},
	{Name: "GetBrowserSession", Method: "GET", URI: "/browsers/{browserIdentifier}/sessions/get?sessionId={sessionId}"},
	{Name: "GetCodeInterpreterSession", Method: "GET", URI: "/code-interpreters/{codeInterpreterIdentifier}/sessions/get?sessionId={sessionId}"},
	{Name: "GetEvent", Method: "GET", URI: "/memories/{memoryId}/actor/{actorId}/sessions/{sessionId}/events/{eventId}"},
	{Name: "GetMemoryRecord", Method: "GET", URI: "/memories/{memoryId}/memoryRecord/{memoryRecordId}"},
	{Name: "GetResourceApiKey", Method: "POST", URI: "/identities/api-key"},
	{Name: "GetResourceOauth2Token", Method: "POST", URI: "/identities/oauth2/token"},
	{Name: "GetWorkloadAccessToken", Method: "POST", URI: "/identities/GetWorkloadAccessToken"},
	{Name: "GetWorkloadAccessTokenForJWT", Method: "POST", URI: "/identities/GetWorkloadAccessTokenForJWT"},
	{Name: "GetWorkloadAccessTokenForUserId", Method: "POST", URI: "/identities/GetWorkloadAccessTokenForUserId"},
	{Name: "InvokeAgentRuntime", Method: "POST", URI: "/runtimes/{agentRuntimeArn}/invocations?accountId={accountId}&qualifier={qualifier}"},
	{Name: "InvokeCodeInterpreter", Method: "POST", URI: "/code-interpreters/{codeInterpreterIdentifier}/tools/invoke"},
	{Name: "ListActors", Method: "POST", URI: "/memories/{memoryId}/actors"},
	{Name: "ListBrowserSessions", Method: "POST", URI: "/browsers/{browserIdentifier}/sessions/list"},
	{Name: "ListCodeInterpreterSessions", Method: "POST", URI: "/code-interpreters/{codeInterpreterIdentifier}/sessions/list"},
	{Name: "ListEvents", Method: "POST", URI: "/memories/{memoryId}/actor/{actorId}/sessions/{sessionId}"},
	{Name: "ListMemoryExtractionJobs", Method: "POST", URI: "/memories/{memoryId}/extractionJobs"},
	{Name: "ListMemoryRecords", Method: "POST", URI: "/memories/{memoryId}/memoryRecords"},
	{Name: "ListSessions", Method: "POST", URI: "/memories/{memoryId}/actor/{actorId}/sessions"},
	{Name: "RetrieveMemoryRecords", Method: "POST", URI: "/memories/{memoryId}/retrieve"},
	{Name: "SaveBrowserSessionProfile", Method: "PUT", URI: "/browser-profiles/{profileIdentifier}/save"},
	{Name: "StartBrowserSession", Method: "PUT", URI: "/browsers/{browserIdentifier}/sessions/start"},
	{Name: "StartCodeInterpreterSession", Method: "PUT", URI: "/code-interpreters/{codeInterpreterIdentifier}/sessions/start"},
	{Name: "StartMemoryExtractionJob", Method: "POST", URI: "/memories/{memoryId}/extractionJobs/start"},
	{Name: "StopBrowserSession", Method: "PUT", URI: "/browsers/{browserIdentifier}/sessions/stop?sessionId={sessionId}"},
	{Name: "StopCodeInterpreterSession", Method: "PUT", URI: "/code-interpreters/{codeInterpreterIdentifier}/sessions/stop?sessionId={sessionId}"},
	{Name: "StopRuntimeSession", Method: "POST", URI: "/runtimes/{agentRuntimeArn}/stopruntimesession?qualifier={qualifier}"},
	{Name: "UpdateBrowserStream", Method: "PUT", URI: "/browsers/{browserIdentifier}/sessions/streams/update?sessionId={sessionId}"},
}

var bedrockAgentCoreDataOperationByName = func() map[string]bedrockAgentCoreDataOperation {
	out := make(map[string]bedrockAgentCoreDataOperation, len(bedrockAgentCoreDataOperations))
	for _, op := range bedrockAgentCoreDataOperations {
		out[op.Name] = op
	}
	return out
}()
