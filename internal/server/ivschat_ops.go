package server

type ivsChatOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon IVS Chat operations sourced from:
// https://docs.aws.amazon.com/ivs/latest/ChatAPIReference/API_Operations.html
var ivsChatOperations = []ivsChatOperation{
	{Name: "CreateChatToken", Method: "POST", URI: "/CreateChatToken"},
	{Name: "CreateLoggingConfiguration", Method: "POST", URI: "/CreateLoggingConfiguration"},
	{Name: "CreateRoom", Method: "POST", URI: "/CreateRoom"},
	{Name: "DeleteLoggingConfiguration", Method: "POST", URI: "/DeleteLoggingConfiguration"},
	{Name: "DeleteMessage", Method: "POST", URI: "/DeleteMessage"},
	{Name: "DeleteRoom", Method: "POST", URI: "/DeleteRoom"},
	{Name: "DisconnectUser", Method: "POST", URI: "/DisconnectUser"},
	{Name: "GetLoggingConfiguration", Method: "POST", URI: "/GetLoggingConfiguration"},
	{Name: "GetRoom", Method: "POST", URI: "/GetRoom"},
	{Name: "ListLoggingConfigurations", Method: "POST", URI: "/ListLoggingConfigurations"},
	{Name: "ListRooms", Method: "POST", URI: "/ListRooms"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "SendEvent", Method: "POST", URI: "/SendEvent"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateLoggingConfiguration", Method: "POST", URI: "/UpdateLoggingConfiguration"},
	{Name: "UpdateRoom", Method: "POST", URI: "/UpdateRoom"},
}

var ivsChatOperationByName = func() map[string]ivsChatOperation {
	out := make(map[string]ivsChatOperation, len(ivsChatOperations))
	for _, op := range ivsChatOperations {
		out[op.Name] = op
	}
	return out
}()
