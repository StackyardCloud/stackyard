package server

type c9Operation struct{ Name string }

// AWS Cloud9 operations sourced from:
// https://docs.aws.amazon.com/cloud9/latest/APIReference/API_Operations.html
var c9Operations = []c9Operation{
	{Name: "CreateEnvironmentEC2"},
	{Name: "CreateEnvironmentMembership"},
	{Name: "DeleteEnvironment"},
	{Name: "DeleteEnvironmentMembership"},
	{Name: "DescribeEnvironmentMemberships"},
	{Name: "DescribeEnvironmentStatus"},
	{Name: "DescribeEnvironments"},
	{Name: "ListEnvironments"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateEnvironment"},
	{Name: "UpdateEnvironmentMembership"},
}

var c9OperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range c9Operations {
		out[op.Name] = true
	}
	return out
}()
