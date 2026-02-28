package server

type entityResolutionDataType struct {
	Name string
}

// AWS Entity Resolution data types sourced from:
// https://docs.aws.amazon.com/entityresolution/latest/apireference/API_Types.html
var entityResolutionDataTypes = []entityResolutionDataType{
	{Name: "CustomerProfilesIntegrationConfig"},
	{Name: "DeleteUniqueIdError"},
	{Name: "DeletedUniqueId"},
	{Name: "ErrorDetails"},
	{Name: "FailedRecord"},
	{Name: "IdMappingIncrementalRunConfig"},
	{Name: "IdMappingJobMetrics"},
	{Name: "IdMappingJobOutputSource"},
	{Name: "IdMappingRuleBasedProperties"},
	{Name: "IdMappingTechniques"},
	{Name: "IdMappingWorkflowInputSource"},
	{Name: "IdMappingWorkflowOutputSource"},
	{Name: "IdMappingWorkflowSummary"},
	{Name: "IdNamespaceIdMappingWorkflowMetadata"},
	{Name: "IdNamespaceIdMappingWorkflowProperties"},
	{Name: "IdNamespaceInputSource"},
	{Name: "IdNamespaceSummary"},
	{Name: "IncrementalRunConfig"},
	{Name: "InputSource"},
	{Name: "IntermediateSourceConfiguration"},
	{Name: "JobMetrics"},
	{Name: "JobOutputSource"},
	{Name: "JobSummary"},
	{Name: "MatchGroup"},
	{Name: "MatchedRecord"},
	{Name: "MatchingWorkflowSummary"},
	{Name: "NamespaceProviderProperties"},
	{Name: "NamespaceRuleBasedProperties"},
	{Name: "OutputAttribute"},
	{Name: "OutputSource"},
	{Name: "ProviderComponentSchema"},
	{Name: "ProviderEndpointConfiguration"},
	{Name: "ProviderIdNameSpaceConfiguration"},
	{Name: "ProviderIntermediateDataAccessConfiguration"},
	{Name: "ProviderMarketplaceConfiguration"},
	{Name: "ProviderProperties"},
	{Name: "ProviderSchemaAttribute"},
	{Name: "ProviderServiceSummary"},
	{Name: "Record"},
	{Name: "ResolutionTechniques"},
	{Name: "Rule"},
	{Name: "RuleBasedProperties"},
	{Name: "RuleCondition"},
	{Name: "RuleConditionProperties"},
	{Name: "SchemaInputAttribute"},
	{Name: "SchemaMappingSummary"},
	{Name: "UpdateSchemaMapping"},
}

var entityResolutionDataTypeByName = func() map[string]entityResolutionDataType {
	out := make(map[string]entityResolutionDataType, len(entityResolutionDataTypes))
	for _, dt := range entityResolutionDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
