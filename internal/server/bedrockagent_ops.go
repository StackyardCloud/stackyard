package server

type bedrockagentOperation struct {
	Name   string
	Method string
	URI    string
	Status int
}

// Agents for Amazon Bedrock (bedrock-agent) actions sourced from:
// https://docs.aws.amazon.com/bedrock/latest/APIReference/API_Operations_Agents_for_Amazon_Bedrock.html
var bedrockagentOperations = []bedrockagentOperation{
	{Name: "AssociateAgentCollaborator", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/agentcollaborators/", Status: 200},
	{Name: "AssociateAgentKnowledgeBase", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/knowledgebases/", Status: 200},
	{Name: "CreateAgent", Method: "PUT", URI: "/agents/", Status: 202},
	{Name: "CreateAgentActionGroup", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/actiongroups/", Status: 200},
	{Name: "CreateAgentAlias", Method: "PUT", URI: "/agents/{agentId}/agentaliases/", Status: 202},
	{Name: "CreateDataSource", Method: "PUT", URI: "/knowledgebases/{knowledgeBaseId}/datasources/", Status: 200},
	{Name: "CreateFlow", Method: "POST", URI: "/flows/", Status: 201},
	{Name: "CreateFlowAlias", Method: "POST", URI: "/flows/{flowIdentifier}/aliases", Status: 201},
	{Name: "CreateFlowVersion", Method: "POST", URI: "/flows/{flowIdentifier}/versions", Status: 201},
	{Name: "CreateKnowledgeBase", Method: "PUT", URI: "/knowledgebases/", Status: 202},
	{Name: "CreatePrompt", Method: "POST", URI: "/prompts/", Status: 201},
	{Name: "CreatePromptVersion", Method: "POST", URI: "/prompts/{promptIdentifier}/versions", Status: 201},
	{Name: "CreateRetrievalInstruction", Method: "POST", URI: "/retrievalinstructions/", Status: 202},
	{Name: "CreateRetrievalInstructionVersion", Method: "POST", URI: "/retrievalinstructions/{retrievalInstructionId}/versions", Status: 201},
	{Name: "CreateVpcConfiguration", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/vpcconfigurations/", Status: 202},
	{Name: "DeleteAgent", Method: "DELETE", URI: "/agents/{agentId}/", Status: 202},
	{Name: "DeleteAgentActionGroup", Method: "DELETE", URI: "/agents/{agentId}/agentversions/{agentVersion}/actiongroups/{actionGroupId}/", Status: 204},
	{Name: "DeleteAgentAlias", Method: "DELETE", URI: "/agents/{agentId}/agentaliases/{agentAliasId}/", Status: 202},
	{Name: "DeleteAgentVersion", Method: "DELETE", URI: "/agents/{agentId}/agentversions/{agentVersion}/", Status: 202},
	{Name: "DeleteDataSource", Method: "DELETE", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}", Status: 202},
	{Name: "DeleteFlow", Method: "DELETE", URI: "/flows/{flowIdentifier}/", Status: 200},
	{Name: "DeleteFlowAlias", Method: "DELETE", URI: "/flows/{flowIdentifier}/aliases/{aliasIdentifier}", Status: 200},
	{Name: "DeleteFlowVersion", Method: "DELETE", URI: "/flows/{flowIdentifier}/versions/{flowVersion}/", Status: 200},
	{Name: "DeleteKnowledgeBase", Method: "DELETE", URI: "/knowledgebases/{knowledgeBaseId}", Status: 202},
	{Name: "DeleteKnowledgeBaseDocuments", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/documents/deleteDocuments", Status: 202},
	{Name: "DeletePrompt", Method: "DELETE", URI: "/prompts/{promptIdentifier}/", Status: 200},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/resourcepolicy/{resourceArn}", Status: 200},
	{Name: "DeleteRetrievalInstruction", Method: "DELETE", URI: "/retrievalinstructions/{retrievalInstructionId}", Status: 200},
	{Name: "DeleteVpcConfiguration", Method: "DELETE", URI: "/knowledgebases/{knowledgeBaseId}/vpcconfigurations/{vpcConfigurationId}", Status: 202},
	{Name: "DisassociateAgentCollaborator", Method: "DELETE", URI: "/agents/{agentId}/agentversions/{agentVersion}/agentcollaborators/{collaboratorId}/", Status: 204},
	{Name: "DisassociateAgentKnowledgeBase", Method: "DELETE", URI: "/agents/{agentId}/agentversions/{agentVersion}/knowledgebases/{knowledgeBaseId}/", Status: 204},
	{Name: "GetAgent", Method: "GET", URI: "/agents/{agentId}/", Status: 200},
	{Name: "GetAgentActionGroup", Method: "GET", URI: "/agents/{agentId}/agentversions/{agentVersion}/actiongroups/{actionGroupId}/", Status: 200},
	{Name: "GetAgentAlias", Method: "GET", URI: "/agents/{agentId}/agentaliases/{agentAliasId}/", Status: 200},
	{Name: "GetAgentCollaborator", Method: "GET", URI: "/agents/{agentId}/agentversions/{agentVersion}/agentcollaborators/{collaboratorId}/", Status: 200},
	{Name: "GetAgentKnowledgeBase", Method: "GET", URI: "/agents/{agentId}/agentversions/{agentVersion}/knowledgebases/{knowledgeBaseId}/", Status: 200},
	{Name: "GetAgentVersion", Method: "GET", URI: "/agents/{agentId}/agentversions/{agentVersion}/", Status: 200},
	{Name: "GetDataSource", Method: "GET", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}", Status: 200},
	{Name: "GetFlow", Method: "GET", URI: "/flows/{flowIdentifier}/", Status: 200},
	{Name: "GetFlowAlias", Method: "GET", URI: "/flows/{flowIdentifier}/aliases/{aliasIdentifier}", Status: 200},
	{Name: "GetFlowVersion", Method: "GET", URI: "/flows/{flowIdentifier}/versions/{flowVersion}/", Status: 200},
	{Name: "GetIngestionJob", Method: "GET", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/ingestionjobs/{ingestionJobId}", Status: 200},
	{Name: "GetKnowledgeBase", Method: "GET", URI: "/knowledgebases/{knowledgeBaseId}", Status: 200},
	{Name: "GetKnowledgeBaseDocuments", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/documents/getDocuments", Status: 200},
	{Name: "GetPrompt", Method: "GET", URI: "/prompts/{promptIdentifier}/", Status: 200},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/resourcepolicy/{resourceArn}", Status: 200},
	{Name: "GetRetrievalInstruction", Method: "GET", URI: "/retrievalinstructions/{retrievalInstructionId}", Status: 200},
	{Name: "GetVpcConfiguration", Method: "GET", URI: "/knowledgebases/{knowledgeBaseId}/vpcconfigurations/{vpcConfigurationId}", Status: 200},
	{Name: "IngestKnowledgeBaseDocuments", Method: "PUT", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/documents", Status: 202},
	{Name: "ListAgentActionGroups", Method: "POST", URI: "/agents/{agentId}/agentversions/{agentVersion}/actiongroups/", Status: 200},
	{Name: "ListAgentAliases", Method: "POST", URI: "/agents/{agentId}/agentaliases/", Status: 200},
	{Name: "ListAgentCollaborators", Method: "POST", URI: "/agents/{agentId}/agentversions/{agentVersion}/agentcollaborators/", Status: 200},
	{Name: "ListAgentKnowledgeBases", Method: "POST", URI: "/agents/{agentId}/agentversions/{agentVersion}/knowledgebases/", Status: 200},
	{Name: "ListAgentVersions", Method: "POST", URI: "/agents/{agentId}/agentversions/", Status: 200},
	{Name: "ListAgents", Method: "POST", URI: "/agents/", Status: 200},
	{Name: "ListDataSources", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/", Status: 200},
	{Name: "ListFlowAliases", Method: "GET", URI: "/flows/{flowIdentifier}/aliases", Status: 200},
	{Name: "ListFlowVersions", Method: "GET", URI: "/flows/{flowIdentifier}/versions", Status: 200},
	{Name: "ListFlows", Method: "GET", URI: "/flows/", Status: 200},
	{Name: "ListIngestionJobs", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/ingestionjobs/", Status: 200},
	{Name: "ListKnowledgeBaseDocuments", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/documents", Status: 200},
	{Name: "ListKnowledgeBases", Method: "POST", URI: "/knowledgebases/", Status: 200},
	{Name: "ListPrompts", Method: "GET", URI: "/prompts/", Status: 200},
	{Name: "ListRetrievalInstructionVersions", Method: "GET", URI: "/retrievalinstructions/{retrievalInstructionId}/versions", Status: 200},
	{Name: "ListRetrievalInstructions", Method: "GET", URI: "/retrievalinstructions/", Status: 200},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "ListVpcConfigurations", Method: "GET", URI: "/knowledgebases/{knowledgeBaseId}/vpcconfigurations/", Status: 200},
	{Name: "PrepareAgent", Method: "POST", URI: "/agents/{agentId}/", Status: 202},
	{Name: "PrepareFlow", Method: "POST", URI: "/flows/{flowIdentifier}/", Status: 202},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/resourcepolicy/{resourceArn}", Status: 200},
	{Name: "StartIngestionJob", Method: "PUT", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/ingestionjobs/", Status: 202},
	{Name: "StopIngestionJob", Method: "POST", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}/ingestionjobs/{ingestionJobId}/stop", Status: 202},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}", Status: 200},
	{Name: "UpdateAgent", Method: "PUT", URI: "/agents/{agentId}/", Status: 202},
	{Name: "UpdateAgentActionGroup", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/actiongroups/{actionGroupId}/", Status: 200},
	{Name: "UpdateAgentAlias", Method: "PUT", URI: "/agents/{agentId}/agentaliases/{agentAliasId}/", Status: 202},
	{Name: "UpdateAgentCollaborator", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/agentcollaborators/{collaboratorId}/", Status: 200},
	{Name: "UpdateAgentKnowledgeBase", Method: "PUT", URI: "/agents/{agentId}/agentversions/{agentVersion}/knowledgebases/{knowledgeBaseId}/", Status: 200},
	{Name: "UpdateDataSource", Method: "PUT", URI: "/knowledgebases/{knowledgeBaseId}/datasources/{dataSourceId}", Status: 200},
	{Name: "UpdateFlow", Method: "PUT", URI: "/flows/{flowIdentifier}/", Status: 200},
	{Name: "UpdateFlowAlias", Method: "PUT", URI: "/flows/{flowIdentifier}/aliases/{aliasIdentifier}", Status: 200},
	{Name: "UpdateKnowledgeBase", Method: "PUT", URI: "/knowledgebases/{knowledgeBaseId}", Status: 202},
	{Name: "UpdatePrompt", Method: "PUT", URI: "/prompts/{promptIdentifier}/", Status: 200},
	{Name: "UpdateRetrievalInstruction", Method: "PUT", URI: "/retrievalinstructions/{retrievalInstructionId}", Status: 202},
	{Name: "ValidateFlowDefinition", Method: "POST", URI: "/flows/validate-definition", Status: 200},
}

var bedrockagentOperationByName = func() map[string]bedrockagentOperation {
	out := make(map[string]bedrockagentOperation, len(bedrockagentOperations))
	for _, op := range bedrockagentOperations {
		out[op.Name] = op
	}
	return out
}()
