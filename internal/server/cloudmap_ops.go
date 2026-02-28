package server

type cloudMapOperation struct {
	Name string
}

// AWS Cloud Map operations sourced from:
// https://docs.aws.amazon.com/cloud-map/latest/api/API_Operations.html
var cloudMapOperations = []cloudMapOperation{
	{Name: "CreateHttpNamespace"},
	{Name: "CreatePrivateDnsNamespace"},
	{Name: "CreatePublicDnsNamespace"},
	{Name: "CreateService"},
	{Name: "DeleteNamespace"},
	{Name: "DeleteService"},
	{Name: "DeleteServiceAttributes"},
	{Name: "DeregisterInstance"},
	{Name: "DiscoverInstances"},
	{Name: "DiscoverInstancesRevision"},
	{Name: "GetInstance"},
	{Name: "GetInstancesHealthStatus"},
	{Name: "GetNamespace"},
	{Name: "GetOperation"},
	{Name: "GetService"},
	{Name: "GetServiceAttributes"},
	{Name: "ListInstances"},
	{Name: "ListNamespaces"},
	{Name: "ListOperations"},
	{Name: "ListServices"},
	{Name: "ListTagsForResource"},
	{Name: "RegisterInstance"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateHttpNamespace"},
	{Name: "UpdateInstanceCustomHealthStatus"},
	{Name: "UpdatePrivateDnsNamespace"},
	{Name: "UpdatePublicDnsNamespace"},
	{Name: "UpdateService"},
	{Name: "UpdateServiceAttributes"},
}

var cloudMapOperationByName = func() map[string]cloudMapOperation {
	out := make(map[string]cloudMapOperation, len(cloudMapOperations))
	for _, op := range cloudMapOperations {
		out[op.Name] = op
	}
	return out
}()
