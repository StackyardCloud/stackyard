package server

// bedrockagentruntimeOperation is one Agents for Amazon Bedrock Runtime operation
// with its REST route and success status, taken from the botocore model
// (bedrock-agent-runtime 2023-07-26).
type bedrockagentruntimeOperation struct {
	Name   string
	Method string
	URI    string
	Status int
}

// Agents for Amazon Bedrock Runtime operations sourced from:
// https://docs.aws.amazon.com/bedrock/latest/APIReference/API_Operations_Agents_for_Amazon_Bedrock_Runtime.html
var bedrockagentruntimeOperations = []bedrockagentruntimeOperation{
	{Name: "AgenticRetrieveStream", Method: "POST", URI: "/agenticRetrieveStream", Status: 200},
	{Name: "CheckIngestedDocumentAcl", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/check-ingested-document-acl", Status: 200},
	{Name: "CreateInvocation", Method: "PUT", URI: "/sessions/{sessionIdentifier}/invocations/", Status: 201},
	{Name: "CreateSession", Method: "PUT", URI: "/sessions/", Status: 201},
	{Name: "DeleteAgentMemory", Method: "DELETE", URI: "/agents/{agentId}/agentAliases/{agentAliasId}/memories", Status: 202},
	{Name: "DeleteSession", Method: "DELETE", URI: "/sessions/{sessionIdentifier}/", Status: 200},
	{Name: "EndSession", Method: "PATCH", URI: "/sessions/{sessionIdentifier}", Status: 200},
	{Name: "GenerateQuery", Method: "POST", URI: "/generateQuery", Status: 200},
	{Name: "GetAgentMemory", Method: "GET", URI: "/agents/{agentId}/agentAliases/{agentAliasId}/memories", Status: 200},
	{Name: "GetDocumentContent", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/documents/{documentId}/content", Status: 200},
	{Name: "GetExecutionFlowSnapshot", Method: "GET", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}/executions/{executionIdentifier}/flowsnapshot", Status: 200},
	{Name: "GetFlowExecution", Method: "GET", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}/executions/{executionIdentifier}", Status: 200},
	{Name: "GetIngestedDocumentAcl", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/get-ingested-document-acl", Status: 200},
	{Name: "GetInvocationStep", Method: "POST", URI: "/sessions/{sessionIdentifier}/invocationSteps/{invocationStepId}", Status: 200},
	{Name: "GetSession", Method: "GET", URI: "/sessions/{sessionIdentifier}/", Status: 200},
	{Name: "InvokeAgent", Method: "POST", URI: "/agents/{agentId}/agentAliases/{agentAliasId}/sessions/{sessionId}/text", Status: 200},
	{Name: "InvokeFlow", Method: "POST", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}", Status: 200},
	{Name: "InvokeInlineAgent", Method: "POST", URI: "/agents/{sessionId}", Status: 200},
	{Name: "ListFlowExecutionEvents", Method: "GET", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}/executions/{executionIdentifier}/events", Status: 200},
	{Name: "ListFlowExecutions", Method: "GET", URI: "/flows/{flowIdentifier}/executions", Status: 200},
	{Name: "ListInvocationSteps", Method: "POST", URI: "/sessions/{sessionIdentifier}/invocationSteps/", Status: 200},
	{Name: "ListInvocations", Method: "POST", URI: "/sessions/{sessionIdentifier}/invocations/", Status: 200},
	{Name: "ListSessions", Method: "POST", URI: "/sessions/", Status: 200},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "OptimizePrompt", Method: "POST", URI: "/optimize-prompt", Status: 200},
	{Name: "PutInvocationStep", Method: "PUT", URI: "/sessions/{sessionIdentifier}/invocationSteps/", Status: 201},
	{Name: "Rerank", Method: "POST", URI: "/rerank", Status: 200},
	{Name: "Retrieve", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/retrieve", Status: 200},
	{Name: "RetrieveAndGenerate", Method: "POST", URI: "/retrieveAndGenerate", Status: 200},
	{Name: "RetrieveAndGenerateStream", Method: "POST", URI: "/retrieveAndGenerateStream", Status: 200},
	{Name: "StartFlowExecution", Method: "POST", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}/executions", Status: 200},
	{Name: "StopFlowExecution", Method: "POST", URI: "/flows/{flowIdentifier}/aliases/{flowAliasIdentifier}/executions/{executionIdentifier}/stop", Status: 200},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "UpdateSession", Method: "PUT", URI: "/sessions/{sessionIdentifier}/", Status: 200},
}

var bedrockagentruntimeOperationByName = func() map[string]bedrockagentruntimeOperation {
	out := make(map[string]bedrockagentruntimeOperation, len(bedrockagentruntimeOperations))
	for _, op := range bedrockagentruntimeOperations {
		out[op.Name] = op
	}
	return out
}()
