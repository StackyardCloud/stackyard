package server

type cloudWatchObservabilityAdminOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudWatch Observability Admin operations sourced from:
// https://docs.aws.amazon.com/cloudwatch/latest/observabilityadmin/API_Operations.html
var cloudWatchObservabilityAdminOperations = []cloudWatchObservabilityAdminOperation{
	{Name: "CreateCentralizationRuleForOrganization", Method: "POST", URI: "/CreateCentralizationRuleForOrganization"},
	{Name: "CreateS3TableIntegration", Method: "POST", URI: "/CreateS3TableIntegration"},
	{Name: "CreateTelemetryPipeline", Method: "POST", URI: "/CreateTelemetryPipeline"},
	{Name: "CreateTelemetryRule", Method: "POST", URI: "/CreateTelemetryRule"},
	{Name: "CreateTelemetryRuleForOrganization", Method: "POST", URI: "/CreateTelemetryRuleForOrganization"},
	{Name: "DeleteCentralizationRuleForOrganization", Method: "POST", URI: "/DeleteCentralizationRuleForOrganization"},
	{Name: "DeleteS3TableIntegration", Method: "POST", URI: "/DeleteS3TableIntegration"},
	{Name: "DeleteTelemetryPipeline", Method: "POST", URI: "/DeleteTelemetryPipeline"},
	{Name: "DeleteTelemetryRule", Method: "POST", URI: "/DeleteTelemetryRule"},
	{Name: "DeleteTelemetryRuleForOrganization", Method: "POST", URI: "/DeleteTelemetryRuleForOrganization"},
	{Name: "GetCentralizationRuleForOrganization", Method: "POST", URI: "/GetCentralizationRuleForOrganization"},
	{Name: "GetS3TableIntegration", Method: "POST", URI: "/GetS3TableIntegration"},
	{Name: "GetTelemetryEnrichmentStatus", Method: "POST", URI: "/GetTelemetryEnrichmentStatus"},
	{Name: "GetTelemetryEvaluationStatus", Method: "POST", URI: "/GetTelemetryEvaluationStatus"},
	{Name: "GetTelemetryEvaluationStatusForOrganization", Method: "POST", URI: "/GetTelemetryEvaluationStatusForOrganization"},
	{Name: "GetTelemetryPipeline", Method: "POST", URI: "/GetTelemetryPipeline"},
	{Name: "GetTelemetryRule", Method: "POST", URI: "/GetTelemetryRule"},
	{Name: "GetTelemetryRuleForOrganization", Method: "POST", URI: "/GetTelemetryRuleForOrganization"},
	{Name: "ListCentralizationRulesForOrganization", Method: "POST", URI: "/ListCentralizationRulesForOrganization"},
	{Name: "ListResourceTelemetry", Method: "POST", URI: "/ListResourceTelemetry"},
	{Name: "ListResourceTelemetryForOrganization", Method: "POST", URI: "/ListResourceTelemetryForOrganization"},
	{Name: "ListS3TableIntegrations", Method: "POST", URI: "/ListS3TableIntegrations"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/ListTagsForResource"},
	{Name: "ListTelemetryPipelines", Method: "POST", URI: "/ListTelemetryPipelines"},
	{Name: "ListTelemetryRules", Method: "POST", URI: "/ListTelemetryRules"},
	{Name: "ListTelemetryRulesForOrganization", Method: "POST", URI: "/ListTelemetryRulesForOrganization"},
	{Name: "StartTelemetryEnrichment", Method: "POST", URI: "/StartTelemetryEnrichment"},
	{Name: "StartTelemetryEvaluation", Method: "POST", URI: "/StartTelemetryEvaluation"},
	{Name: "StartTelemetryEvaluationForOrganization", Method: "POST", URI: "/StartTelemetryEvaluationForOrganization"},
	{Name: "StopTelemetryEnrichment", Method: "POST", URI: "/StopTelemetryEnrichment"},
	{Name: "StopTelemetryEvaluation", Method: "POST", URI: "/StopTelemetryEvaluation"},
	{Name: "StopTelemetryEvaluationForOrganization", Method: "POST", URI: "/StopTelemetryEvaluationForOrganization"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "TestTelemetryPipeline", Method: "POST", URI: "/TestTelemetryPipeline"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
	{Name: "UpdateCentralizationRuleForOrganization", Method: "POST", URI: "/UpdateCentralizationRuleForOrganization"},
	{Name: "UpdateTelemetryPipeline", Method: "POST", URI: "/UpdateTelemetryPipeline"},
	{Name: "UpdateTelemetryRule", Method: "POST", URI: "/UpdateTelemetryRule"},
	{Name: "UpdateTelemetryRuleForOrganization", Method: "POST", URI: "/UpdateTelemetryRuleForOrganization"},
	{Name: "ValidateTelemetryPipelineConfiguration", Method: "POST", URI: "/ValidateTelemetryPipelineConfiguration"},
}

var cloudWatchObservabilityAdminOperationByName = func() map[string]cloudWatchObservabilityAdminOperation {
	out := make(map[string]cloudWatchObservabilityAdminOperation, len(cloudWatchObservabilityAdminOperations))
	for _, op := range cloudWatchObservabilityAdminOperations {
		out[op.Name] = op
	}
	return out
}()
