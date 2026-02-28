package server

type routingControlDataType struct {
	Name string
}

// Amazon Route 53 Application Recovery Controller - Routing Control data types sourced from:
// https://docs.aws.amazon.com/routing-control/latest/APIReference/API_Types.html
// AWS currently serves this API guide under /latest/api/ resource pages.
var routingControlDataTypes = []routingControlDataType{
	{Name: "RoutingControl"},
	{Name: "UpdateRoutingControlStateEntry"},
	{Name: "ValidationExceptionField"},
}

var routingControlDataTypeByName = func() map[string]routingControlDataType {
	out := make(map[string]routingControlDataType, len(routingControlDataTypes))
	for _, dt := range routingControlDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
