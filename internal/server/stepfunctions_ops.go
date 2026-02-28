package server

type stepFunctionsOperation struct {
	Name string
}

// AWS Step Functions actions sourced from:
// https://docs.aws.amazon.com/step-functions/latest/apireference/API_Operations.html
var stepFunctionsOperations = []stepFunctionsOperation{
	{Name: "CreateActivity"},
	{Name: "CreateStateMachine"},
	{Name: "CreateStateMachineAlias"},
	{Name: "DeleteActivity"},
	{Name: "DeleteStateMachine"},
	{Name: "DeleteStateMachineAlias"},
	{Name: "DeleteStateMachineVersion"},
	{Name: "DescribeActivity"},
	{Name: "DescribeExecution"},
	{Name: "DescribeMapRun"},
	{Name: "DescribeStateMachine"},
	{Name: "DescribeStateMachineAlias"},
	{Name: "DescribeStateMachineForExecution"},
	{Name: "GetActivityTask"},
	{Name: "GetExecutionHistory"},
	{Name: "ListActivities"},
	{Name: "ListExecutions"},
	{Name: "ListMapRuns"},
	{Name: "ListStateMachineAliases"},
	{Name: "ListStateMachines"},
	{Name: "ListStateMachineVersions"},
	{Name: "ListTagsForResource"},
	{Name: "PublishStateMachineVersion"},
	{Name: "RedriveExecution"},
	{Name: "SendTaskFailure"},
	{Name: "SendTaskHeartbeat"},
	{Name: "SendTaskSuccess"},
	{Name: "StartExecution"},
	{Name: "StartSyncExecution"},
	{Name: "StopExecution"},
	{Name: "TagResource"},
	{Name: "TestState"},
	{Name: "UntagResource"},
	{Name: "UpdateMapRun"},
	{Name: "UpdateStateMachine"},
	{Name: "UpdateStateMachineAlias"},
	{Name: "ValidateStateMachineDefinition"},
}

var stepFunctionsOperationByName = func() map[string]stepFunctionsOperation {
	out := make(map[string]stepFunctionsOperation, len(stepFunctionsOperations))
	for _, op := range stepFunctionsOperations {
		out[op.Name] = op
	}
	return out
}()
