package server

type ivsChatMessagingOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon IVS Chat Messaging operations sourced from:
// https://docs.aws.amazon.com/ivs/latest/chatmsgapireference/actions.html
var ivsChatMessagingOperations = []ivsChatMessagingOperation{
	{Name: "DeleteMessage", Method: "POST", URI: "/DeleteMessage"},
	{Name: "DisconnectUser", Method: "POST", URI: "/DisconnectUser"},
	{Name: "SendMessage", Method: "POST", URI: "/SendMessage"},
}

var ivsChatMessagingOperationByName = func() map[string]ivsChatMessagingOperation {
	out := make(map[string]ivsChatMessagingOperation, len(ivsChatMessagingOperations))
	for _, op := range ivsChatMessagingOperations {
		out[op.Name] = op
	}
	return out
}()
