package server

type batchOperation struct {
	Name    string
	Method  string
	Pattern string
}

var batchOperations = []batchOperation{
	{Name: "CancelJob", Method: "POST", Pattern: "/v1/canceljob"},
	{Name: "CreateComputeEnvironment", Method: "POST", Pattern: "/v1/createcomputeenvironment"},
	{Name: "CreateConsumableResource", Method: "POST", Pattern: "/v1/createconsumableresource"},
	{Name: "CreateJobQueue", Method: "POST", Pattern: "/v1/createjobqueue"},
	{Name: "CreateSchedulingPolicy", Method: "POST", Pattern: "/v1/createschedulingpolicy"},
	{Name: "CreateServiceEnvironment", Method: "POST", Pattern: "/v1/createserviceenvironment"},
	{Name: "DeleteComputeEnvironment", Method: "POST", Pattern: "/v1/deletecomputeenvironment"},
	{Name: "DeleteConsumableResource", Method: "POST", Pattern: "/v1/deleteconsumableresource"},
	{Name: "DeleteJobQueue", Method: "POST", Pattern: "/v1/deletejobqueue"},
	{Name: "DeleteSchedulingPolicy", Method: "POST", Pattern: "/v1/deleteschedulingpolicy"},
	{Name: "DeleteServiceEnvironment", Method: "POST", Pattern: "/v1/deleteserviceenvironment"},
	{Name: "DeregisterJobDefinition", Method: "POST", Pattern: "/v1/deregisterjobdefinition"},
	{Name: "DescribeComputeEnvironments", Method: "POST", Pattern: "/v1/describecomputeenvironments"},
	{Name: "DescribeConsumableResource", Method: "POST", Pattern: "/v1/describeconsumableresource"},
	{Name: "DescribeJobDefinitions", Method: "POST", Pattern: "/v1/describejobdefinitions"},
	{Name: "DescribeJobQueues", Method: "POST", Pattern: "/v1/describejobqueues"},
	{Name: "DescribeJobs", Method: "POST", Pattern: "/v1/describejobs"},
	{Name: "DescribeSchedulingPolicies", Method: "POST", Pattern: "/v1/describeschedulingpolicies"},
	{Name: "DescribeServiceEnvironments", Method: "POST", Pattern: "/v1/describeserviceenvironments"},
	{Name: "DescribeServiceJob", Method: "POST", Pattern: "/v1/describeservicejob"},
	{Name: "GetJobQueueSnapshot", Method: "POST", Pattern: "/v1/getjobqueuesnapshot"},
	{Name: "ListConsumableResources", Method: "POST", Pattern: "/v1/listconsumableresources"},
	{Name: "ListJobs", Method: "POST", Pattern: "/v1/listjobs"},
	{Name: "ListJobsByConsumableResource", Method: "POST", Pattern: "/v1/listjobsbyconsumableresource"},
	{Name: "ListSchedulingPolicies", Method: "POST", Pattern: "/v1/listschedulingpolicies"},
	{Name: "ListServiceJobs", Method: "POST", Pattern: "/v1/listservicejobs"},
	{Name: "ListTagsForResource", Method: "GET", Pattern: "/v1/tags/{resourceArn}"},
	{Name: "RegisterJobDefinition", Method: "POST", Pattern: "/v1/registerjobdefinition"},
	{Name: "SubmitJob", Method: "POST", Pattern: "/v1/submitjob"},
	{Name: "SubmitServiceJob", Method: "POST", Pattern: "/v1/submitservicejob"},
	{Name: "TagResource", Method: "POST", Pattern: "/v1/tags/{resourceArn}"},
	{Name: "TerminateJob", Method: "POST", Pattern: "/v1/terminatejob"},
	{Name: "TerminateServiceJob", Method: "POST", Pattern: "/v1/terminateservicejob"},
	{Name: "UntagResource", Method: "DELETE", Pattern: "/v1/tags/{resourceArn}"},
	{Name: "UpdateComputeEnvironment", Method: "POST", Pattern: "/v1/updatecomputeenvironment"},
	{Name: "UpdateConsumableResource", Method: "POST", Pattern: "/v1/updateconsumableresource"},
	{Name: "UpdateJobQueue", Method: "POST", Pattern: "/v1/updatejobqueue"},
	{Name: "UpdateSchedulingPolicy", Method: "POST", Pattern: "/v1/updateschedulingpolicy"},
	{Name: "UpdateServiceEnvironment", Method: "POST", Pattern: "/v1/updateserviceenvironment"},
}

var batchOperationByName = func() map[string]batchOperation {
	out := make(map[string]batchOperation, len(batchOperations))
	for _, op := range batchOperations {
		out[op.Name] = op
	}
	return out
}()
