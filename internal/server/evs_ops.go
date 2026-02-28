package server

type evsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Elastic VMware Service operations sourced from:
// https://docs.aws.amazon.com/evs/latest/APIReference/API_Operations.html
var evsOperations = []evsOperation{
	{Name: "AssociateEipToVlan", Method: "POST", URI: "/environments/{environmentId}/vlans/{vlanId}/eip-associations"},
	{Name: "CreateEnvironment", Method: "POST", URI: "/environments"},
	{Name: "CreateEnvironmentHost", Method: "POST", URI: "/environments/{environmentId}/hosts"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environments/{environmentId}"},
	{Name: "DeleteEnvironmentHost", Method: "DELETE", URI: "/environments/{environmentId}/hosts/{hostId}"},
	{Name: "DisassociateEipFromVlan", Method: "DELETE", URI: "/environments/{environmentId}/vlans/{vlanId}/eip-associations/{allocationId}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environments/{environmentId}"},
	{Name: "GetVersions", Method: "GET", URI: "/versions"},
	{Name: "ListEnvironmentHosts", Method: "GET", URI: "/environments/{environmentId}/hosts"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environments"},
	{Name: "ListEnvironmentVlans", Method: "GET", URI: "/environments/{environmentId}/vlans"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
}

var evsOperationByName = func() map[string]evsOperation {
	out := make(map[string]evsOperation, len(evsOperations))
	for _, op := range evsOperations {
		out[op.Name] = op
	}
	return out
}()
