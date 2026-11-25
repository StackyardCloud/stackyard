package server

type machineLearningOperation struct{ Name string }

// Amazon Machine Learning operations sourced from:
// https://docs.aws.amazon.com/machine-learning/latest/APIReference/API_Operations.html
var machineLearningOperations = []machineLearningOperation{
	{Name: "AddTags"},
	{Name: "CreateBatchPrediction"},
	{Name: "CreateDataSourceFromRDS"},
	{Name: "CreateDataSourceFromRedshift"},
	{Name: "CreateDataSourceFromS3"},
	{Name: "CreateEvaluation"},
	{Name: "CreateMLModel"},
	{Name: "CreateRealtimeEndpoint"},
	{Name: "DeleteBatchPrediction"},
	{Name: "DeleteDataSource"},
	{Name: "DeleteEvaluation"},
	{Name: "DeleteMLModel"},
	{Name: "DeleteRealtimeEndpoint"},
	{Name: "DeleteTags"},
	{Name: "DescribeBatchPredictions"},
	{Name: "DescribeDataSources"},
	{Name: "DescribeEvaluations"},
	{Name: "DescribeMLModels"},
	{Name: "DescribeTags"},
	{Name: "GetBatchPrediction"},
	{Name: "GetDataSource"},
	{Name: "GetEvaluation"},
	{Name: "GetMLModel"},
	{Name: "Predict"},
	{Name: "UpdateBatchPrediction"},
	{Name: "UpdateDataSource"},
	{Name: "UpdateEvaluation"},
	{Name: "UpdateMLModel"},
}
