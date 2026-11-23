package server

type dataBrewOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Glue DataBrew operations sourced from:
// https://docs.aws.amazon.com/databrew/latest/dg/API_Operations.html
var dataBrewOperations = []dataBrewOperation{
	{Name: "BatchDeleteRecipeVersion", Method: "POST", URI: "/recipes/{name}/batchDeleteRecipeVersion"},
	{Name: "CreateDataset", Method: "POST", URI: "/datasets"},
	{Name: "CreateProfileJob", Method: "POST", URI: "/profileJobs"},
	{Name: "CreateProject", Method: "POST", URI: "/projects"},
	{Name: "CreateRecipe", Method: "POST", URI: "/recipes"},
	{Name: "CreateRecipeJob", Method: "POST", URI: "/recipeJobs"},
	{Name: "CreateRuleset", Method: "POST", URI: "/rulesets"},
	{Name: "CreateSchedule", Method: "POST", URI: "/schedules"},
	{Name: "DeleteDataset", Method: "DELETE", URI: "/datasets/{name}"},
	{Name: "DeleteJob", Method: "DELETE", URI: "/jobs/{name}"},
	{Name: "DeleteProject", Method: "DELETE", URI: "/projects/{name}"},
	{Name: "DeleteRecipeVersion", Method: "DELETE", URI: "/recipes/{name}/recipeVersion/{recipeVersion}"},
	{Name: "DeleteRuleset", Method: "DELETE", URI: "/rulesets/{name}"},
	{Name: "DeleteSchedule", Method: "DELETE", URI: "/schedules/{name}"},
	{Name: "DescribeDataset", Method: "GET", URI: "/datasets/{name}"},
	{Name: "DescribeJob", Method: "GET", URI: "/jobs/{name}"},
	{Name: "DescribeJobRun", Method: "GET", URI: "/jobs/{name}/jobRun/{runId}"},
	{Name: "DescribeProject", Method: "GET", URI: "/projects/{name}"},
	{Name: "DescribeRecipe", Method: "GET", URI: "/recipes/{name}"},
	{Name: "DescribeRuleset", Method: "GET", URI: "/rulesets/{name}"},
	{Name: "DescribeSchedule", Method: "GET", URI: "/schedules/{name}"},
	{Name: "ListDatasets", Method: "GET", URI: "/datasets"},
	{Name: "ListJobRuns", Method: "GET", URI: "/jobs/{name}/jobRuns"},
	{Name: "ListJobs", Method: "GET", URI: "/jobs"},
	{Name: "ListProjects", Method: "GET", URI: "/projects"},
	{Name: "ListRecipeVersions", Method: "GET", URI: "/recipeVersions"},
	{Name: "ListRecipes", Method: "GET", URI: "/recipes"},
	{Name: "ListRulesets", Method: "GET", URI: "/rulesets"},
	{Name: "ListSchedules", Method: "GET", URI: "/schedules"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PublishRecipe", Method: "POST", URI: "/recipes/{name}/publishRecipe"},
	{Name: "SendProjectSessionAction", Method: "PUT", URI: "/projects/{name}/sendProjectSessionAction"},
	{Name: "StartJobRun", Method: "POST", URI: "/jobs/{name}/startJobRun"},
	{Name: "StartProjectSession", Method: "PUT", URI: "/projects/{name}/startProjectSession"},
	{Name: "StopJobRun", Method: "POST", URI: "/jobs/{name}/jobRun/{runId}/stopJobRun"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateDataset", Method: "PUT", URI: "/datasets/{name}"},
	{Name: "UpdateProfileJob", Method: "PUT", URI: "/profileJobs/{name}"},
	{Name: "UpdateProject", Method: "PUT", URI: "/projects/{name}"},
	{Name: "UpdateRecipe", Method: "PUT", URI: "/recipes/{name}"},
	{Name: "UpdateRecipeJob", Method: "PUT", URI: "/recipeJobs/{name}"},
	{Name: "UpdateRuleset", Method: "PUT", URI: "/rulesets/{name}"},
	{Name: "UpdateSchedule", Method: "PUT", URI: "/schedules/{name}"},
}
