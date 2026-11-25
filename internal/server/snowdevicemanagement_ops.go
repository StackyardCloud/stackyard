package server

// AWS Snow Device Management operations sourced from:
// https://docs.aws.amazon.com/snowball/latest/api-reference/API_Operations_AWS_Snow_Device_Management.html
var snowDeviceManagementOperations = []opstRoute{
	{Name: "CancelTask", Method: "POST", URI: "/task/{taskId}/cancel"},
	{Name: "CreateTask", Method: "POST", URI: "/task"},
	{Name: "DescribeDevice", Method: "POST", URI: "/managed-device/{managedDeviceId}/describe"},
	{Name: "DescribeDeviceEc2Instances", Method: "POST", URI: "/managed-device/{managedDeviceId}/resources/ec2/describe"},
	{Name: "DescribeExecution", Method: "POST", URI: "/task/{taskId}/execution/{managedDeviceId}"},
	{Name: "DescribeTask", Method: "POST", URI: "/task/{taskId}"},
	{Name: "ListDeviceResources", Method: "GET", URI: "/managed-device/{managedDeviceId}/resources"},
	{Name: "ListDevices", Method: "GET", URI: "/managed-devices"},
	{Name: "ListExecutions", Method: "GET", URI: "/executions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTasks", Method: "GET", URI: "/tasks"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
}
