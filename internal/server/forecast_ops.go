package server

type forecastOperation struct{ Name string }

// Amazon Forecast operations sourced from:
// https://docs.aws.amazon.com/forecast/latest/dg/API_Operations_Amazon_Forecast_Service.html
var forecastOperations = []forecastOperation{
	{Name: "CreateAutoPredictor"},
	{Name: "CreateDataset"},
	{Name: "CreateDatasetGroup"},
	{Name: "CreateDatasetImportJob"},
	{Name: "CreateExplainability"},
	{Name: "CreateExplainabilityExport"},
	{Name: "CreateForecast"},
	{Name: "CreateForecastExportJob"},
	{Name: "CreateMonitor"},
	{Name: "CreatePredictor"},
	{Name: "CreatePredictorBacktestExportJob"},
	{Name: "CreateWhatIfAnalysis"},
	{Name: "CreateWhatIfForecast"},
	{Name: "CreateWhatIfForecastExport"},
	{Name: "DeleteDataset"},
	{Name: "DeleteDatasetGroup"},
	{Name: "DeleteDatasetImportJob"},
	{Name: "DeleteExplainability"},
	{Name: "DeleteExplainabilityExport"},
	{Name: "DeleteForecast"},
	{Name: "DeleteForecastExportJob"},
	{Name: "DeleteMonitor"},
	{Name: "DeletePredictor"},
	{Name: "DeletePredictorBacktestExportJob"},
	{Name: "DeleteResourceTree"},
	{Name: "DeleteWhatIfAnalysis"},
	{Name: "DeleteWhatIfForecast"},
	{Name: "DeleteWhatIfForecastExport"},
	{Name: "DescribeAutoPredictor"},
	{Name: "DescribeDataset"},
	{Name: "DescribeDatasetGroup"},
	{Name: "DescribeDatasetImportJob"},
	{Name: "DescribeExplainability"},
	{Name: "DescribeExplainabilityExport"},
	{Name: "DescribeForecast"},
	{Name: "DescribeForecastExportJob"},
	{Name: "DescribeMonitor"},
	{Name: "DescribePredictor"},
	{Name: "DescribePredictorBacktestExportJob"},
	{Name: "DescribeWhatIfAnalysis"},
	{Name: "DescribeWhatIfForecast"},
	{Name: "DescribeWhatIfForecastExport"},
	{Name: "GetAccuracyMetrics"},
	{Name: "ListDatasetGroups"},
	{Name: "ListDatasetImportJobs"},
	{Name: "ListDatasets"},
	{Name: "ListExplainabilities"},
	{Name: "ListExplainabilityExports"},
	{Name: "ListForecastExportJobs"},
	{Name: "ListForecasts"},
	{Name: "ListMonitorEvaluations"},
	{Name: "ListMonitors"},
	{Name: "ListPredictorBacktestExportJobs"},
	{Name: "ListPredictors"},
	{Name: "ListTagsForResource"},
	{Name: "ListWhatIfAnalyses"},
	{Name: "ListWhatIfForecastExports"},
	{Name: "ListWhatIfForecasts"},
	{Name: "ResumeResource"},
	{Name: "StopResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDatasetGroup"},
}

var forecastOperationByName = func() map[string]forecastOperation {
	out := make(map[string]forecastOperation, len(forecastOperations))
	for _, op := range forecastOperations {
		out[op.Name] = op
	}
	return out
}()
