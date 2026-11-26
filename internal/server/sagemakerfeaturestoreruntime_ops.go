package server

type sagemakerfeaturestoreruntimeOperation struct{ Name string }

// Operations sourced from:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Feature_Store_Runtime.html
var sagemakerfeaturestoreruntimeOperations = []sagemakerfeaturestoreruntimeOperation{
	{Name: "BatchGetRecord"},
	{Name: "BatchWriteRecord"},
	{Name: "DeleteRecord"},
	{Name: "GetRecord"},
	{Name: "ListRecords"},
	{Name: "PutRecord"},
	{Name: "UpdateRecord"},
}

var sagemakerfeaturestoreruntimeOperationByName = func() map[string]sagemakerfeaturestoreruntimeOperation {
	out := make(map[string]sagemakerfeaturestoreruntimeOperation, len(sagemakerfeaturestoreruntimeOperations))
	for _, op := range sagemakerfeaturestoreruntimeOperations {
		out[op.Name] = op
	}
	return out
}()
