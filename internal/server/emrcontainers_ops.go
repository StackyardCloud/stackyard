package server

type emrContainersOperation struct {
	Name string
}

// Amazon EMR on EKS actions sourced from:
// https://docs.aws.amazon.com/emr-on-eks/latest/APIReference/API_Operations.html
var emrContainersOperations = []emrContainersOperation{
	{Name: "CancelJobRun"},
	{Name: "CreateJobTemplate"},
	{Name: "CreateManagedEndpoint"},
	{Name: "CreateSecurityConfiguration"},
	{Name: "CreateVirtualCluster"},
	{Name: "DeleteJobTemplate"},
	{Name: "DeleteManagedEndpoint"},
	{Name: "DeleteVirtualCluster"},
	{Name: "DescribeJobRun"},
	{Name: "DescribeJobTemplate"},
	{Name: "DescribeManagedEndpoint"},
	{Name: "DescribeSecurityConfiguration"},
	{Name: "DescribeVirtualCluster"},
	{Name: "GetManagedEndpointSessionCredentials"},
	{Name: "ListJobRuns"},
	{Name: "ListJobTemplates"},
	{Name: "ListManagedEndpoints"},
	{Name: "ListSecurityConfigurations"},
	{Name: "ListTagsForResource"},
	{Name: "ListVirtualClusters"},
	{Name: "StartJobRun"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var emrContainersOperationByName = func() map[string]emrContainersOperation {
	out := make(map[string]emrContainersOperation, len(emrContainersOperations))
	for _, op := range emrContainersOperations {
		out[op.Name] = op
	}
	return out
}()
