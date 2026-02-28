package server

type augmentedAIDataType struct {
	Name string
}

// Amazon Augmented AI Runtime data types sourced from:
// https://docs.aws.amazon.com/augmented-ai/2019-11-07/APIReference/API_Types.html
var augmentedAIDataTypes = []augmentedAIDataType{
	{Name: "HumanLoopDataAttributes"},
	{Name: "HumanLoopInput"},
	{Name: "HumanLoopOutput"},
	{Name: "HumanLoopSummary"},
}

var augmentedAIDataTypeByName = func() map[string]augmentedAIDataType {
	out := make(map[string]augmentedAIDataType, len(augmentedAIDataTypes))
	for _, dt := range augmentedAIDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
