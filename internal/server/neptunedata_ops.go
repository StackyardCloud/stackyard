package server

type neptuneDataOperation struct {
	Name string
}

// Amazon Neptune Data API operations sourced from:
// https://docs.aws.amazon.com/neptune/latest/data-api/API_Operations.html
var neptuneDataOperations = []neptuneDataOperation{
	{Name: "CancelGremlinQuery"},
	{Name: "CancelLoaderJob"},
	{Name: "CancelMLDataProcessingJob"},
	{Name: "CancelMLModelTrainingJob"},
	{Name: "CancelMLModelTransformJob"},
	{Name: "CancelOpenCypherQuery"},
	{Name: "CreateMLEndpoint"},
	{Name: "DeleteMLEndpoint"},
	{Name: "DeletePropertygraphStatistics"},
	{Name: "DeleteSparqlStatistics"},
	{Name: "ExecuteFastReset"},
	{Name: "ExecuteGremlinExplainQuery"},
	{Name: "ExecuteGremlinProfileQuery"},
	{Name: "ExecuteGremlinQuery"},
	{Name: "ExecuteOpenCypherExplainQuery"},
	{Name: "ExecuteOpenCypherQuery"},
	{Name: "GetEngineStatus"},
	{Name: "GetGremlinQueryStatus"},
	{Name: "GetLoaderJobStatus"},
	{Name: "GetMLDataProcessingJob"},
	{Name: "GetMLEndpoint"},
	{Name: "GetMLModelTrainingJob"},
	{Name: "GetMLModelTransformJob"},
	{Name: "GetOpenCypherQueryStatus"},
	{Name: "GetPropertygraphStatistics"},
	{Name: "GetPropertygraphStream"},
	{Name: "GetPropertygraphSummary"},
	{Name: "GetRDFGraphSummary"},
	{Name: "GetSparqlStatistics"},
	{Name: "GetSparqlStream"},
	{Name: "ListGremlinQueries"},
	{Name: "ListLoaderJobs"},
	{Name: "ListMLDataProcessingJobs"},
	{Name: "ListMLEndpoints"},
	{Name: "ListMLModelTrainingJobs"},
	{Name: "ListMLModelTransformJobs"},
	{Name: "ListOpenCypherQueries"},
	{Name: "ManagePropertygraphStatistics"},
	{Name: "ManageSparqlStatistics"},
	{Name: "StartLoaderJob"},
	{Name: "StartMLDataProcessingJob"},
	{Name: "StartMLModelTrainingJob"},
	{Name: "StartMLModelTransformJob"},
}

var neptuneDataOperationByName = func() map[string]neptuneDataOperation {
	out := make(map[string]neptuneDataOperation, len(neptuneDataOperations))
	for _, op := range neptuneDataOperations {
		out[op.Name] = op
	}
	return out
}()
