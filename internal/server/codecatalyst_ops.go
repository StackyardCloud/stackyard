package server

type codeCatalystOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CodeCatalyst actions sourced from:
// https://docs.aws.amazon.com/codecatalyst/latest/APIReference/API_Operations.html
var codeCatalystOperations = []codeCatalystOperation{
	{Name: "CreateAccessToken", Method: "PUT", URI: "/v1/accessTokens"},
	{Name: "CreateDevEnvironment", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments"},
	{Name: "CreateProject", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects"},
	{Name: "CreateSourceRepository", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{name}"},
	{Name: "CreateSourceRepositoryBranch", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{sourceRepositoryName}/branches/{name}"},
	{Name: "DeleteAccessToken", Method: "DELETE", URI: "/v1/accessTokens/{id}"},
	{Name: "DeleteDevEnvironment", Method: "DELETE", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}"},
	{Name: "DeleteProject", Method: "DELETE", URI: "/v1/spaces/{spaceName}/projects/{name}"},
	{Name: "DeleteSourceRepository", Method: "DELETE", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{name}"},
	{Name: "DeleteSpace", Method: "DELETE", URI: "/v1/spaces/{name}"},
	{Name: "GetDevEnvironment", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}"},
	{Name: "GetProject", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{name}"},
	{Name: "GetSourceRepository", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{name}"},
	{Name: "GetSourceRepositoryCloneUrls", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{sourceRepositoryName}/cloneUrls"},
	{Name: "GetSpace", Method: "GET", URI: "/v1/spaces/{name}"},
	{Name: "GetSubscription", Method: "GET", URI: "/v1/spaces/{spaceName}/subscription"},
	{Name: "GetUserDetails", Method: "GET", URI: "/userDetails"},
	{Name: "GetWorkflow", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{projectName}/workflows/{id}"},
	{Name: "GetWorkflowRun", Method: "GET", URI: "/v1/spaces/{spaceName}/projects/{projectName}/workflowRuns/{id}"},
	{Name: "ListAccessTokens", Method: "POST", URI: "/v1/accessTokens"},
	{Name: "ListDevEnvironmentSessions", Method: "POST", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{devEnvironmentId}/sessions"},
	{Name: "ListDevEnvironments", Method: "POST", URI: "/v1/spaces/{spaceName}/devEnvironments"},
	{Name: "ListEventLogs", Method: "POST", URI: "/v1/spaces/{spaceName}/eventLogs"},
	{Name: "ListProjects", Method: "POST", URI: "/v1/spaces/{spaceName}/projects"},
	{Name: "ListSourceRepositories", Method: "POST", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories"},
	{Name: "ListSourceRepositoryBranches", Method: "POST", URI: "/v1/spaces/{spaceName}/projects/{projectName}/sourceRepositories/{sourceRepositoryName}/branches"},
	{Name: "ListSpaces", Method: "POST", URI: "/v1/spaces"},
	{Name: "ListWorkflowRuns", Method: "POST", URI: "/v1/spaces/{spaceName}/projects/{projectName}/workflowRuns"},
	{Name: "ListWorkflows", Method: "POST", URI: "/v1/spaces/{spaceName}/projects/{projectName}/workflows"},
	{Name: "StartDevEnvironment", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}/start"},
	{Name: "StartDevEnvironmentSession", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}/session"},
	{Name: "StartWorkflowRun", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/workflowRuns"},
	{Name: "StopDevEnvironment", Method: "PUT", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}/stop"},
	{Name: "StopDevEnvironmentSession", Method: "DELETE", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}/session/{sessionId}"},
	{Name: "UpdateDevEnvironment", Method: "PATCH", URI: "/v1/spaces/{spaceName}/projects/{projectName}/devEnvironments/{id}"},
	{Name: "UpdateProject", Method: "PATCH", URI: "/v1/spaces/{spaceName}/projects/{name}"},
	{Name: "UpdateSpace", Method: "PATCH", URI: "/v1/spaces/{name}"},
	{Name: "VerifySession", Method: "GET", URI: "/session"},
}

var codeCatalystOperationByName = func() map[string]codeCatalystOperation {
	out := make(map[string]codeCatalystOperation, len(codeCatalystOperations))
	for _, op := range codeCatalystOperations {
		out[op.Name] = op
	}
	return out
}()
