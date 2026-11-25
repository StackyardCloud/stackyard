package server

// Amazon Location Service Routes V2 operations (AWS CLI 2.42.1 model) sourced from:
// https://docs.aws.amazon.com/location/latest/APIReference/API_Operations_Amazon_Location_Service_Routes_V2.html
var georoutesOperations = []geosaOp{
	{Name: "CalculateIsolines", Method: "POST", URI: "/v2/isolines"},
	{Name: "CalculateRouteMatrix", Method: "POST", URI: "/v2/route-matrix"},
	{Name: "CalculateRoutes", Method: "POST", URI: "/v2/routes"},
	{Name: "OptimizeWaypoints", Method: "POST", URI: "/v2/optimize-waypoints"},
	{Name: "SnapToRoads", Method: "POST", URI: "/v2/snap-to-roads"},
}

var georoutesOperationByName = func() map[string]geosaOp {
	out := make(map[string]geosaOp, len(georoutesOperations))
	for _, op := range georoutesOperations {
		out[op.Name] = op
	}
	return out
}()
