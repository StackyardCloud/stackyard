package server

type cloudWatchInvestigationsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudWatch Investigations operations sourced from:
// https://docs.aws.amazon.com/cloudwatchinvestigations/latest/APIReference/API_Operations.html
var cloudWatchInvestigationsOperations = []cloudWatchInvestigationsOperation{
	{Name: "CreateInvestigationGroup", Method: "POST", URI: "/investigationGroups"},
	{Name: "DeleteInvestigationGroup", Method: "DELETE", URI: "/investigationGroups/{identifier}"},
	{Name: "DeleteInvestigationGroupPolicy", Method: "DELETE", URI: "/investigationGroups/{identifier}/policy"},
	{Name: "GetInvestigationGroup", Method: "GET", URI: "/investigationGroups/{identifier}"},
	{Name: "GetInvestigationGroupPolicy", Method: "GET", URI: "/investigationGroups/{identifier}/policy"},
	{Name: "ListInvestigationGroups", Method: "GET", URI: "/investigationGroups"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutInvestigationGroupPolicy", Method: "POST", URI: "/investigationGroups/{identifier}/policy"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateInvestigationGroup", Method: "PATCH", URI: "/investigationGroups/{identifier}"},
}

var cloudWatchInvestigationsOperationByName = func() map[string]cloudWatchInvestigationsOperation {
	out := make(map[string]cloudWatchInvestigationsOperation, len(cloudWatchInvestigationsOperations))
	for _, op := range cloudWatchInvestigationsOperations {
		out[op.Name] = op
	}
	return out
}()
