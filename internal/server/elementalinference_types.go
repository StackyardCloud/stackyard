package server

type elementalInferenceDataType struct {
	Name string
}

// AWS Elemental Inference data types sourced from:
// https://docs.aws.amazon.com/elemental-inference/latest/APIReference/API_Types.html
var elementalInferenceDataTypes = []elementalInferenceDataType{
	{Name: "ClippingConfig"},
	{Name: "CreateOutput"},
	{Name: "CroppingConfig"},
	{Name: "FeedAssociation"},
	{Name: "FeedSummary"},
	{Name: "GetOutput"},
	{Name: "OutputConfig"},
	{Name: "UpdateFeed"},
	{Name: "UpdateOutput"},
}

var elementalInferenceDataTypeByName = func() map[string]elementalInferenceDataType {
	out := make(map[string]elementalInferenceDataType, len(elementalInferenceDataTypes))
	for _, dt := range elementalInferenceDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
