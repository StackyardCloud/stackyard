package server

// Amazon Location Service Maps V2 operations (AWS CLI 2.42.1 model) sourced from:
// https://docs.aws.amazon.com/location/latest/APIReference/API_Operations_Amazon_Location_Service_Maps_V2.html
var geomapsOperations = []geosaOp{
	{Name: "GetGlyphs", Method: "GET", URI: "/v2/glyphs/{FontStack}/{FontUnicodeRange}"},
	{Name: "GetSprites", Method: "GET", URI: "/v2/styles/{Style}/{ColorScheme}/{Variant}/sprites/{FileName}"},
	{Name: "GetStaticMap", Method: "GET", URI: "/v2/static/{FileName}"},
	{Name: "GetStyleDescriptor", Method: "GET", URI: "/v2/styles/{Style}/descriptor"},
	{Name: "GetTile", Method: "GET", URI: "/v2/tiles/{Tileset}/{Z}/{X}/{Y}"},
}

var geomapsOperationByName = func() map[string]geosaOp {
	out := make(map[string]geosaOp, len(geomapsOperations))
	for _, op := range geomapsOperations {
		out[op.Name] = op
	}
	return out
}()
