package server

type routingControlOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Route 53 Application Recovery Controller - Routing Control operations sourced from:
// https://docs.aws.amazon.com/routing-control/latest/APIReference/API_Operations.html
// AWS currently serves this API guide under /latest/api/ resource pages.
var routingControlOperations = []routingControlOperation{
	{Name: "GetRoutingControlState", Method: "POST", URI: "/"},
	{Name: "ListRoutingControls", Method: "POST", URI: "/"},
	{Name: "UpdateRoutingControlState", Method: "POST", URI: "/"},
	{Name: "UpdateRoutingControlStates", Method: "POST", URI: "/"},
}

var routingControlOperationByName = func() map[string]routingControlOperation {
	out := make(map[string]routingControlOperation, len(routingControlOperations))
	for _, op := range routingControlOperations {
		out[op.Name] = op
	}
	return out
}()
