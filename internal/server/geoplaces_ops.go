package server

// Amazon Location Service Places V2 operations (AWS CLI 2.42.1 model) sourced from:
// https://docs.aws.amazon.com/location/latest/APIReference/API_Operations_Amazon_Location_Service_Places_V2.html
var geoplacesOperations = []geosaOp{
	{Name: "Autocomplete", Method: "POST", URI: "/v2/autocomplete"},
	{Name: "Geocode", Method: "POST", URI: "/v2/geocode"},
	{Name: "GetPlace", Method: "GET", URI: "/v2/place/{PlaceId}"},
	{Name: "ReverseGeocode", Method: "POST", URI: "/v2/reverse-geocode"},
	{Name: "SearchNearby", Method: "POST", URI: "/v2/search-nearby"},
	{Name: "SearchText", Method: "POST", URI: "/v2/search-text"},
	{Name: "Suggest", Method: "POST", URI: "/v2/suggest"},
}

var geoplacesOperationByName = func() map[string]geosaOp {
	out := make(map[string]geosaOp, len(geoplacesOperations))
	for _, op := range geoplacesOperations {
		out[op.Name] = op
	}
	return out
}()
