package server

type iotJobsDataPlaneOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Jobs Data Plane operations sourced from:
// https://docs.aws.amazon.com/iot/latest/apireference/API_Operations_AWS_IoT_Jobs_Data_Plane.html
var iotJobsDataPlaneOperations = []iotJobsDataPlaneOperation{
	{Name: "DescribeJobExecution", Method: "GET", URI: "/things/{thingName}/jobs/{jobId}"},
	{Name: "GetPendingJobExecutions", Method: "GET", URI: "/things/{thingName}/jobs"},
	{Name: "StartCommandExecution", Method: "POST", URI: "/command-executions"},
	{Name: "StartNextPendingJobExecution", Method: "PUT", URI: "/things/{thingName}/jobs/$next"},
	{Name: "UpdateJobExecution", Method: "POST", URI: "/things/{thingName}/jobs/{jobId}"},
}
