package server

type controlCatalogOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Control Catalog actions sourced from:
// https://docs.aws.amazon.com/controlcatalog/latest/APIReference/API_Operations.html
var controlCatalogOperations = []controlCatalogOperation{
	{Name: "GetControl", Method: "POST", URI: "/get-control"},
	{Name: "ListCommonControls", Method: "POST", URI: "/common-controls"},
	{Name: "ListControlMappings", Method: "POST", URI: "/list-control-mappings"},
	{Name: "ListControls", Method: "POST", URI: "/list-controls"},
	{Name: "ListDomains", Method: "POST", URI: "/domains"},
	{Name: "ListObjectives", Method: "POST", URI: "/objectives"},
}

var controlCatalogOperationByName = func() map[string]controlCatalogOperation {
	out := make(map[string]controlCatalogOperation, len(controlCatalogOperations))
	for _, op := range controlCatalogOperations {
		out[op.Name] = op
	}
	return out
}()
