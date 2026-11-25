package server

type forecastQueryOperation struct{ Name string }

// Amazon Forecast Query operations sourced from:
// https://docs.aws.amazon.com/forecast/latest/dg/API_Operations_Amazon_Forecast_Query_Service.html
var forecastQueryOperations = []forecastQueryOperation{
	{Name: "QueryForecast"},
	{Name: "QueryWhatIfForecast"},
}

var forecastQueryOperationByName = func() map[string]forecastQueryOperation {
	out := make(map[string]forecastQueryOperation, len(forecastQueryOperations))
	for _, op := range forecastQueryOperations {
		out[op.Name] = op
	}
	return out
}()
