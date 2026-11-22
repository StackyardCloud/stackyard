package server

type schedulerOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon EventBridge Scheduler operations sourced from:
// https://docs.aws.amazon.com/scheduler/latest/APIReference/API_Operations.html
var schedulerOperations = []schedulerOperation{
	{Name: "CreateSchedule", Method: "POST", URI: "/schedules/{Name}"},
	{Name: "CreateScheduleGroup", Method: "POST", URI: "/schedule-groups/{Name}"},
	{Name: "DeleteSchedule", Method: "DELETE", URI: "/schedules/{Name}"},
	{Name: "DeleteScheduleGroup", Method: "DELETE", URI: "/schedule-groups/{Name}"},
	{Name: "GetSchedule", Method: "GET", URI: "/schedules/{Name}"},
	{Name: "GetScheduleGroup", Method: "GET", URI: "/schedule-groups/{Name}"},
	{Name: "ListScheduleGroups", Method: "GET", URI: "/schedule-groups"},
	{Name: "ListSchedules", Method: "GET", URI: "/schedules"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateSchedule", Method: "PUT", URI: "/schedules/{Name}"},
}

var schedulerOperationByName = func() map[string]schedulerOperation {
	out := make(map[string]schedulerOperation, len(schedulerOperations))
	for _, op := range schedulerOperations {
		out[op.Name] = op
	}
	return out
}()
