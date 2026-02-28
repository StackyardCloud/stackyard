package server

type stepFunctionsDataType struct {
	Name string
}

// AWS Step Functions data types sourced from:
// https://docs.aws.amazon.com/step-functions/latest/apireference/API_Types.html
var stepFunctionsDataTypes = []stepFunctionsDataType{
	{Name: "ActivityFailedEventDetails"},
	{Name: "ActivityListItem"},
	{Name: "ActivityScheduledEventDetails"},
	{Name: "ActivityScheduleFailedEventDetails"},
	{Name: "ActivityStartedEventDetails"},
	{Name: "ActivitySucceededEventDetails"},
	{Name: "ActivityTimedOutEventDetails"},
	{Name: "AssignedVariablesDetails"},
	{Name: "BillingDetails"},
	{Name: "CloudWatchEventsExecutionDataDetails"},
	{Name: "CloudWatchLogsLogGroup"},
	{Name: "EncryptionConfiguration"},
	{Name: "EvaluationFailedEventDetails"},
	{Name: "ExecutionAbortedEventDetails"},
	{Name: "ExecutionFailedEventDetails"},
	{Name: "ExecutionListItem"},
	{Name: "ExecutionRedrivenEventDetails"},
	{Name: "ExecutionStartedEventDetails"},
	{Name: "ExecutionSucceededEventDetails"},
	{Name: "ExecutionTimedOutEventDetails"},
	{Name: "HistoryEvent"},
	{Name: "HistoryEventExecutionDataDetails"},
	{Name: "InspectionData"},
	{Name: "InspectionDataRequest"},
	{Name: "InspectionDataResponse"},
	{Name: "InspectionErrorDetails"},
	{Name: "LambdaFunctionFailedEventDetails"},
	{Name: "LambdaFunctionScheduledEventDetails"},
	{Name: "LambdaFunctionScheduleFailedEventDetails"},
	{Name: "LambdaFunctionStartFailedEventDetails"},
	{Name: "LambdaFunctionSucceededEventDetails"},
	{Name: "LambdaFunctionTimedOutEventDetails"},
	{Name: "LogDestination"},
	{Name: "LoggingConfiguration"},
	{Name: "MapIterationEventDetails"},
	{Name: "MapRunExecutionCounts"},
	{Name: "MapRunFailedEventDetails"},
	{Name: "MapRunItemCounts"},
	{Name: "MapRunListItem"},
	{Name: "MapRunRedrivenEventDetails"},
	{Name: "MapRunStartedEventDetails"},
	{Name: "MapStateStartedEventDetails"},
	{Name: "MockErrorOutput"},
	{Name: "MockInput"},
	{Name: "RoutingConfigurationListItem"},
	{Name: "StateEnteredEventDetails"},
	{Name: "StateExitedEventDetails"},
	{Name: "StateMachineAliasListItem"},
	{Name: "StateMachineListItem"},
	{Name: "StateMachineVersionListItem"},
	{Name: "Tag"},
	{Name: "TaskCredentials"},
	{Name: "TaskFailedEventDetails"},
	{Name: "TaskScheduledEventDetails"},
	{Name: "TaskStartedEventDetails"},
	{Name: "TaskStartFailedEventDetails"},
	{Name: "TaskSubmitFailedEventDetails"},
	{Name: "TaskSubmittedEventDetails"},
	{Name: "TaskSucceededEventDetails"},
	{Name: "TaskTimedOutEventDetails"},
	{Name: "TestStateConfiguration"},
	{Name: "TracingConfiguration"},
	{Name: "ValidateStateMachineDefinitionDiagnostic"},
}

var stepFunctionsDataTypeByName = func() map[string]stepFunctionsDataType {
	out := make(map[string]stepFunctionsDataType, len(stepFunctionsDataTypes))
	for _, dt := range stepFunctionsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
