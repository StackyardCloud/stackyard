package server

type gameliftStreamsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon GameLift Streams API actions sourced from:
// https://docs.aws.amazon.com/gameliftstreams/latest/apireference/API_Operations.html
var gameliftStreamsOperations = []gameliftStreamsOperation{
	{Name: "AddStreamGroupLocations", Method: "POST", URI: "/streamgroups/{identifier}/locations"},
	{Name: "AssociateApplications", Method: "POST", URI: "/streamgroups/{identifier}/associations"},
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateStreamGroup", Method: "POST", URI: "/streamgroups"},
	{Name: "CreateStreamSessionConnection", Method: "POST", URI: "/streamgroups/{identifier}/streamsessions/{streamSessionIdentifier}/connections"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{identifier}"},
	{Name: "DeleteStreamGroup", Method: "DELETE", URI: "/streamgroups/{identifier}"},
	{Name: "DisassociateApplications", Method: "POST", URI: "/streamgroups/{identifier}/disassociations"},
	{Name: "ExportStreamSessionFiles", Method: "PUT", URI: "/streamgroups/{identifier}/streamsessions/{streamSessionIdentifier}/exportfiles"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{identifier}"},
	{Name: "GetStreamGroup", Method: "GET", URI: "/streamgroups/{identifier}"},
	{Name: "GetStreamSession", Method: "GET", URI: "/streamgroups/{identifier}/streamsessions/{streamSessionIdentifier}"},
	{Name: "ListApplications", Method: "GET", URI: "/applications"},
	{Name: "ListStreamGroups", Method: "GET", URI: "/streamgroups"},
	{Name: "ListStreamSessions", Method: "GET", URI: "/streamgroups/{identifier}/streamsessions"},
	{Name: "ListStreamSessionsByAccount", Method: "GET", URI: "/streamsessions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RemoveStreamGroupLocations", Method: "DELETE", URI: "/streamgroups/{identifier}/locations"},
	{Name: "StartStreamSession", Method: "POST", URI: "/streamgroups/{identifier}/streamsessions"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "TerminateStreamSession", Method: "DELETE", URI: "/streamgroups/{identifier}/streamsessions/{streamSessionIdentifier}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{identifier}"},
	{Name: "UpdateStreamGroup", Method: "PATCH", URI: "/streamgroups/{identifier}"},
}

var gameliftStreamsOperationByName = func() map[string]gameliftStreamsOperation {
	out := make(map[string]gameliftStreamsOperation, len(gameliftStreamsOperations))
	for _, op := range gameliftStreamsOperations {
		out[op.Name] = op
	}
	return out
}()
