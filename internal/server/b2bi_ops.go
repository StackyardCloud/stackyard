package server

type b2biOperation struct {
	Name string
}

// AWS B2B Data Interchange actions sourced from:
// https://docs.aws.amazon.com/b2bi/latest/APIReference/API_Operations.html
var b2biOperations = []b2biOperation{
	{Name: "CreateCapability"},
	{Name: "CreatePartnership"},
	{Name: "CreateProfile"},
	{Name: "CreateStarterMappingTemplate"},
	{Name: "CreateTransformer"},
	{Name: "DeleteCapability"},
	{Name: "DeletePartnership"},
	{Name: "DeleteProfile"},
	{Name: "DeleteTransformer"},
	{Name: "GenerateMapping"},
	{Name: "GetCapability"},
	{Name: "GetPartnership"},
	{Name: "GetProfile"},
	{Name: "GetTransformer"},
	{Name: "GetTransformerJob"},
	{Name: "ListCapabilities"},
	{Name: "ListPartnerships"},
	{Name: "ListProfiles"},
	{Name: "ListTagsForResource"},
	{Name: "ListTransformers"},
	{Name: "StartTransformerJob"},
	{Name: "TagResource"},
	{Name: "TestConversion"},
	{Name: "TestMapping"},
	{Name: "TestParsing"},
	{Name: "UntagResource"},
	{Name: "UpdateCapability"},
	{Name: "UpdatePartnership"},
	{Name: "UpdateProfile"},
	{Name: "UpdateTransformer"},
}

var b2biOperationByName = func() map[string]b2biOperation {
	out := make(map[string]b2biOperation, len(b2biOperations))
	for _, op := range b2biOperations {
		out[op.Name] = op
	}
	return out
}()
