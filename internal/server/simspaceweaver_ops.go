package server

type simSpaceWeaverOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon SimSpace Weaver actions sourced from:
// https://docs.aws.amazon.com/simspaceweaver/latest/APIReference/API_Operations.html
var simSpaceWeaverOperations = []simSpaceWeaverOperation{
	{Name: "CreateSnapshot", Method: "POST", URI: "/createsnapshot"},
	{Name: "DeleteApp", Method: "DELETE", URI: "/deleteapp?app={app}&domain={domain}&simulation={simulation}"},
	{Name: "DeleteSimulation", Method: "DELETE", URI: "/deletesimulation?simulation={simulation}"},
	{Name: "DescribeApp", Method: "GET", URI: "/describeapp?app={app}&domain={domain}&simulation={simulation}"},
	{Name: "DescribeSimulation", Method: "GET", URI: "/describesimulation?simulation={simulation}"},
	{Name: "ListApps", Method: "GET", URI: "/listapps?domain={domain}&maxResults={maxResults}&nextToken={nextToken}&simulation={simulation}"},
	{Name: "ListSimulations", Method: "GET", URI: "/listsimulations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "StartApp", Method: "POST", URI: "/startapp"},
	{Name: "StartClock", Method: "POST", URI: "/startclock"},
	{Name: "StartSimulation", Method: "POST", URI: "/startsimulation"},
	{Name: "StopApp", Method: "POST", URI: "/stopapp"},
	{Name: "StopClock", Method: "POST", URI: "/stopclock"},
	{Name: "StopSimulation", Method: "POST", URI: "/stopsimulation"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}?tagKeys={tagKeys}"},
}

var simSpaceWeaverOperationByName = func() map[string]simSpaceWeaverOperation {
	out := make(map[string]simSpaceWeaverOperation, len(simSpaceWeaverOperations))
	for _, op := range simSpaceWeaverOperations {
		out[op.Name] = op
	}
	return out
}()
