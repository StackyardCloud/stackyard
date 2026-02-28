package server

type fisOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Fault Injection Service actions sourced from:
// https://docs.aws.amazon.com/fis/latest/APIReference/API_Operations.html
var fisOperations = []fisOperation{
	{Name: "CreateExperimentTemplate", Method: "POST", URI: "/experimentTemplates"},
	{Name: "CreateTargetAccountConfiguration", Method: "POST", URI: "/experimentTemplates/{experimentTemplateId}/targetAccountConfigurations"},
	{Name: "DeleteExperimentTemplate", Method: "DELETE", URI: "/experimentTemplates/{id}"},
	{Name: "DeleteTargetAccountConfiguration", Method: "DELETE", URI: "/experimentTemplates/{experimentTemplateId}/targetAccountConfigurations/{accountId}"},
	{Name: "GetAction", Method: "GET", URI: "/actions/{id}"},
	{Name: "GetExperiment", Method: "GET", URI: "/experiments/{id}"},
	{Name: "GetExperimentTargetAccountConfiguration", Method: "GET", URI: "/experiments/{experimentId}/targetAccountConfigurations/{accountId}"},
	{Name: "GetExperimentTemplate", Method: "GET", URI: "/experimentTemplates/{id}"},
	{Name: "GetSafetyLever", Method: "GET", URI: "/safetyLevers/{id}"},
	{Name: "GetTargetAccountConfiguration", Method: "GET", URI: "/experimentTemplates/{experimentTemplateId}/targetAccountConfigurations/{accountId}"},
	{Name: "GetTargetResourceType", Method: "GET", URI: "/targetResourceTypes/{id}"},
	{Name: "ListActions", Method: "GET", URI: "/actions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListExperimentResolvedTargets", Method: "GET", URI: "/experiments/{id}/resolvedTargets?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListExperimentTargetAccountConfigurations", Method: "GET", URI: "/experiments/{experimentId}/targetAccountConfigurations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListExperimentTemplates", Method: "GET", URI: "/experimentTemplates?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListExperiments", Method: "GET", URI: "/experiments?experimentTemplateId={experimentTemplateId}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn+}"},
	{Name: "ListTargetAccountConfigurations", Method: "GET", URI: "/experimentTemplates/{experimentTemplateId}/targetAccountConfigurations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTargetResourceTypes", Method: "GET", URI: "/targetResourceTypes?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "StartExperiment", Method: "POST", URI: "/experiments"},
	{Name: "StopExperiment", Method: "DELETE", URI: "/experiments/{id}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn+}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn+}?tagKeys={tagKeys}"},
	{Name: "UpdateExperimentTemplate", Method: "PATCH", URI: "/experimentTemplates/{id}"},
	{Name: "UpdateSafetyLeverState", Method: "PATCH", URI: "/safetyLevers/{id}/state"},
	{Name: "UpdateTargetAccountConfiguration", Method: "PATCH", URI: "/experimentTemplates/{experimentTemplateId}/targetAccountConfigurations/{accountId}"},
}

var fisOperationByName = func() map[string]fisOperation {
	out := make(map[string]fisOperation, len(fisOperations))
	for _, op := range fisOperations {
		out[op.Name] = op
	}
	return out
}()
