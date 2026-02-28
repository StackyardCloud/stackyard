package server

type amplifyOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Amplify actions sourced from:
// https://docs.aws.amazon.com/amplify/latest/APIReference/API_Operations.html
var amplifyOperations = []amplifyOperation{
	{Name: "CreateApp", Method: "POST", URI: "/apps"},
	{Name: "CreateBackendEnvironment", Method: "POST", URI: "/apps/{appId}/backendenvironments"},
	{Name: "CreateBranch", Method: "POST", URI: "/apps/{appId}/branches"},
	{Name: "CreateDeployment", Method: "POST", URI: "/apps/{appId}/branches/{branchName}/deployments"},
	{Name: "CreateDomainAssociation", Method: "POST", URI: "/apps/{appId}/domains"},
	{Name: "CreateWebhook", Method: "POST", URI: "/apps/{appId}/webhooks"},
	{Name: "DeleteApp", Method: "DELETE", URI: "/apps/{appId}"},
	{Name: "DeleteBackendEnvironment", Method: "DELETE", URI: "/apps/{appId}/backendenvironments/{environmentName}"},
	{Name: "DeleteBranch", Method: "DELETE", URI: "/apps/{appId}/branches/{branchName}"},
	{Name: "DeleteDomainAssociation", Method: "DELETE", URI: "/apps/{appId}/domains/{domainName}"},
	{Name: "DeleteJob", Method: "DELETE", URI: "/apps/{appId}/branches/{branchName}/jobs/{jobId}"},
	{Name: "DeleteWebhook", Method: "DELETE", URI: "/webhooks/{webhookId}"},
	{Name: "GenerateAccessLogs", Method: "POST", URI: "/apps/{appId}/accesslogs"},
	{Name: "GetApp", Method: "GET", URI: "/apps/{appId}"},
	{Name: "GetArtifactUrl", Method: "GET", URI: "/artifacts/{artifactId}"},
	{Name: "GetBackendEnvironment", Method: "GET", URI: "/apps/{appId}/backendenvironments/{environmentName}"},
	{Name: "GetBranch", Method: "GET", URI: "/apps/{appId}/branches/{branchName}"},
	{Name: "GetDomainAssociation", Method: "GET", URI: "/apps/{appId}/domains/{domainName}"},
	{Name: "GetJob", Method: "GET", URI: "/apps/{appId}/branches/{branchName}/jobs/{jobId}"},
	{Name: "GetWebhook", Method: "GET", URI: "/webhooks/{webhookId}"},
	{Name: "ListApps", Method: "GET", URI: "/apps?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListArtifacts", Method: "GET", URI: "/apps/{appId}/branches/{branchName}/jobs/{jobId}/artifacts?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListBackendEnvironments", Method: "GET", URI: "/apps/{appId}/backendenvironments?environmentName={environmentName}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListBranches", Method: "GET", URI: "/apps/{appId}/branches?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDomainAssociations", Method: "GET", URI: "/apps/{appId}/domains?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListJobs", Method: "GET", URI: "/apps/{appId}/branches/{branchName}/jobs?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListWebhooks", Method: "GET", URI: "/apps/{appId}/webhooks?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "StartDeployment", Method: "POST", URI: "/apps/{appId}/branches/{branchName}/deployments/start"},
	{Name: "StartJob", Method: "POST", URI: "/apps/{appId}/branches/{branchName}/jobs"},
	{Name: "StopJob", Method: "DELETE", URI: "/apps/{appId}/branches/{branchName}/jobs/{jobId}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateApp", Method: "POST", URI: "/apps/{appId}"},
	{Name: "UpdateBranch", Method: "POST", URI: "/apps/{appId}/branches/{branchName}"},
	{Name: "UpdateDomainAssociation", Method: "POST", URI: "/apps/{appId}/domains/{domainName}"},
	{Name: "UpdateWebhook", Method: "POST", URI: "/webhooks/{webhookId}"},
}

var amplifyOperationByName = func() map[string]amplifyOperation {
	out := make(map[string]amplifyOperation, len(amplifyOperations))
	for _, op := range amplifyOperations {
		out[op.Name] = op
	}
	return out
}()
