package server

type iotDataPlaneOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Data Plane operations sourced from:
// https://docs.aws.amazon.com/iot/latest/apireference/API_Operations_AWS_IoT_Data_Plane.html
var iotDataPlaneOperations = []iotDataPlaneOperation{
	{Name: "DeleteConnection", Method: "DELETE", URI: "/connections/{clientId}"},
	{Name: "DeleteThingShadow", Method: "DELETE", URI: "/things/{thingName}/shadow"},
	{Name: "GetConnection", Method: "GET", URI: "/connections/{clientId}"},
	{Name: "GetRetainedMessage", Method: "GET", URI: "/retainedMessage/{topic}"},
	{Name: "GetThingShadow", Method: "GET", URI: "/things/{thingName}/shadow"},
	{Name: "ListNamedShadowsForThing", Method: "GET", URI: "/api/things/shadow/ListNamedShadowsForThing/{thingName}"},
	{Name: "ListRetainedMessages", Method: "GET", URI: "/retainedMessage"},
	{Name: "ListSubscriptions", Method: "GET", URI: "/connections/{clientId}/subscriptions"},
	{Name: "Publish", Method: "POST", URI: "/topics/{topic}"},
	{Name: "SendDirectMessage", Method: "POST", URI: "/connections/{clientId}/messages"},
	{Name: "UpdateThingShadow", Method: "POST", URI: "/things/{thingName}/shadow"},
}
