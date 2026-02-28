package server

type bedrockAgentCoreDataType struct {
	Name string
}

// Amazon Bedrock AgentCore Data Plane data types sourced from:
// https://docs.aws.amazon.com/bedrock-agentcore/latest/APIReference/API_Types.html
var bedrockAgentCoreDataTypes = []bedrockAgentCoreDataType{
	{Name: "ActorSummary"},
	{Name: "AutomationStream"},
	{Name: "AutomationStreamUpdate"},
	{Name: "Branch"},
	{Name: "BranchFilter"},
	{Name: "BrowserExtension"},
	{Name: "BrowserProfileConfiguration"},
	{Name: "BrowserSessionStream"},
	{Name: "BrowserSessionSummary"},
	{Name: "CodeInterpreterResult"},
	{Name: "CodeInterpreterSessionSummary"},
	{Name: "CodeInterpreterStreamOutput"},
	{Name: "Content"},
	{Name: "ContentBlock"},
	{Name: "Context"},
	{Name: "Conversational"},
	{Name: "EvaluationInput"},
	{Name: "EvaluationResultContent"},
	{Name: "EvaluationTarget"},
	{Name: "Event"},
	{Name: "EventMetadataFilterExpression"},
	{Name: "ExtractionJob"},
	{Name: "ExtractionJobFilterInput"},
	{Name: "ExtractionJobMessages"},
	{Name: "ExtractionJobMetadata"},
	{Name: "FilterInput"},
	{Name: "InputContentBlock"},
	{Name: "LeftExpression"},
	{Name: "LiveViewStream"},
	{Name: "MemoryContent"},
	{Name: "MemoryMetadataFilterExpression"},
	{Name: "MemoryRecord"},
	{Name: "MemoryRecordCreateInput"},
	{Name: "MemoryRecordDeleteInput"},
	{Name: "MemoryRecordOutput"},
	{Name: "MemoryRecordSummary"},
	{Name: "MemoryRecordUpdateInput"},
	{Name: "MessageMetadata"},
	{Name: "MetadataValue"},
	{Name: "PayloadType"},
	{Name: "ResourceContent"},
	{Name: "ResourceLocation"},
	{Name: "RightExpression"},
	{Name: "S3Location"},
	{Name: "SearchCriteria"},
	{Name: "SessionSummary"},
	{Name: "SpanContext"},
	{Name: "StreamUpdate"},
	{Name: "TokenUsage"},
	{Name: "ToolArguments"},
	{Name: "ToolResultStructuredContent"},
	{Name: "UpdateBrowserStream"},
	{Name: "UserIdentifier"},
	{Name: "ValidationExceptionField"},
	{Name: "ViewPort"},
}

var bedrockAgentCoreDataTypeByName = func() map[string]bedrockAgentCoreDataType {
	out := make(map[string]bedrockAgentCoreDataType, len(bedrockAgentCoreDataTypes))
	for _, dt := range bedrockAgentCoreDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
