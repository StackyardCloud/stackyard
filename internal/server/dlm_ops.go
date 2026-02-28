package server

type dlmOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Data Lifecycle Manager actions sourced from:
// https://docs.aws.amazon.com/dlm/latest/APIReference/API_Operations.html
var dlmOperations = []dlmOperation{
	{Name: "CreateLifecyclePolicy", Method: "POST", URI: "/policies"},
	{Name: "DeleteLifecyclePolicy", Method: "DELETE", URI: "/policies/{policyId}"},
	{Name: "GetLifecyclePolicies", Method: "GET", URI: "/policies?defaultPolicyType={DefaultPolicyType}&policyIds={PolicyIds}&resourceTypes={ResourceTypes}&state={State}&tagsToAdd={TagsToAdd}&targetTags={TargetTags}"},
	{Name: "GetLifecyclePolicy", Method: "GET", URI: "/policies/{policyId}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={TagKeys}"},
	{Name: "UpdateLifecyclePolicy", Method: "PATCH", URI: "/policies/{policyId}"},
}

var dlmOperationByName = func() map[string]dlmOperation {
	out := make(map[string]dlmOperation, len(dlmOperations))
	for _, op := range dlmOperations {
		out[op.Name] = op
	}
	return out
}()
