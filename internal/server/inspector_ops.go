package server

type inspectorOperation struct{ Name string }

// Amazon Inspector Classic operations sourced from:
// https://docs.aws.amazon.com/inspector/v1/APIReference/API_Operations.html
var inspectorOperations = []inspectorOperation{
	{Name: "AddAttributesToFindings"},
	{Name: "CreateAssessmentTarget"},
	{Name: "CreateAssessmentTemplate"},
	{Name: "CreateExclusionsPreview"},
	{Name: "CreateResourceGroup"},
	{Name: "DeleteAssessmentRun"},
	{Name: "DeleteAssessmentTarget"},
	{Name: "DeleteAssessmentTemplate"},
	{Name: "DescribeAssessmentRuns"},
	{Name: "DescribeAssessmentTargets"},
	{Name: "DescribeAssessmentTemplates"},
	{Name: "DescribeCrossAccountAccessRole"},
	{Name: "DescribeExclusions"},
	{Name: "DescribeFindings"},
	{Name: "DescribeResourceGroups"},
	{Name: "DescribeRulesPackages"},
	{Name: "GetAssessmentReport"},
	{Name: "GetExclusionsPreview"},
	{Name: "GetTelemetryMetadata"},
	{Name: "ListAssessmentRunAgents"},
	{Name: "ListAssessmentRuns"},
	{Name: "ListAssessmentTargets"},
	{Name: "ListAssessmentTemplates"},
	{Name: "ListEventSubscriptions"},
	{Name: "ListExclusions"},
	{Name: "ListFindings"},
	{Name: "ListRulesPackages"},
	{Name: "ListTagsForResource"},
	{Name: "PreviewAgents"},
	{Name: "RegisterCrossAccountAccessRole"},
	{Name: "RemoveAttributesFromFindings"},
	{Name: "SetTagsForResource"},
	{Name: "StartAssessmentRun"},
	{Name: "StopAssessmentRun"},
	{Name: "SubscribeToEvent"},
	{Name: "UnsubscribeFromEvent"},
	{Name: "UpdateAssessmentTarget"},
}

var inspectorOperationByName = func() map[string]inspectorOperation {
	out := make(map[string]inspectorOperation, len(inspectorOperations))
	for _, op := range inspectorOperations {
		out[op.Name] = op
	}
	return out
}()
