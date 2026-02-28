package server

type ivsChatMessagingDataType struct {
	Name string
}

// Amazon IVS Chat Messaging data types sourced from message schemas on:
// https://docs.aws.amazon.com/ivs/latest/chatmsgapireference/actions.html
var ivsChatMessagingDataTypes = []ivsChatMessagingDataType{
	{Name: "Event"},
	{Name: "Message"},
}

var ivsChatMessagingDataTypeByName = func() map[string]ivsChatMessagingDataType {
	out := make(map[string]ivsChatMessagingDataType, len(ivsChatMessagingDataTypes))
	for _, dt := range ivsChatMessagingDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
