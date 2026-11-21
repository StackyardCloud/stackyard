package server

type bedrockRuntimeOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Bedrock Runtime operations sourced from:
// https://docs.aws.amazon.com/bedrock/latest/APIReference/API_Operations_Amazon_Bedrock_Runtime.html
var bedrockRuntimeOperations = []bedrockRuntimeOperation{
	{Name: "ApplyGuardrail", Method: "POST", URI: "/guardrail/{guardrailIdentifier}/version/{guardrailVersion}/apply"},
	{Name: "Converse", Method: "POST", URI: "/model/{modelId}/converse"},
	{Name: "ConverseStream", Method: "POST", URI: "/model/{modelId}/converse-stream"},
	{Name: "CountTokens", Method: "POST", URI: "/model/{modelId}/count-tokens"},
	{Name: "GetAsyncInvoke", Method: "GET", URI: "/async-invoke/{invocationArn}"},
	{Name: "InvokeGuardrailChecks", Method: "POST", URI: "/guardrail-checks/invoke"},
	{Name: "InvokeModel", Method: "POST", URI: "/model/{modelId}/invoke"},
	{Name: "InvokeModelWithBidirectionalStream", Method: "POST", URI: "/model/{modelId}/invoke-with-bidirectional-stream"},
	{Name: "InvokeModelWithResponseStream", Method: "POST", URI: "/model/{modelId}/invoke-with-response-stream"},
	{Name: "ListAsyncInvokes", Method: "GET", URI: "/async-invoke"},
	{Name: "StartAsyncInvoke", Method: "POST", URI: "/async-invoke"},
}

var bedrockRuntimeOperationByName = func() map[string]bedrockRuntimeOperation {
	out := make(map[string]bedrockRuntimeOperation, len(bedrockRuntimeOperations))
	for _, op := range bedrockRuntimeOperations {
		out[op.Name] = op
	}
	return out
}()
