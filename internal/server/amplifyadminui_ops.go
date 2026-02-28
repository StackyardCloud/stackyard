package server

type amplifyAdminUIOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Amplify Admin UI operations sourced from:
// https://docs.aws.amazon.com/amplify-admin-ui/latest/APIReference/operations.html
var amplifyAdminUIOperations = []amplifyAdminUIOperation{
	{Name: "CloneBackend", Method: "POST", URI: "/prod/backend/{appId}/environments/{backendEnvironmentName}/clone"},
	{Name: "CreateBackend", Method: "POST", URI: "/prod/backend"},
	{Name: "CreateBackendAPI", Method: "POST", URI: "/prod/backend/{appId}/api"},
	{Name: "CreateBackendAuth", Method: "POST", URI: "/prod/backend/{appId}/auth"},
	{Name: "CreateBackendConfig", Method: "POST", URI: "/prod/backend/{appId}/config"},
	{Name: "CreateBackendStorage", Method: "POST", URI: "/prod/backend/{appId}/storage"},
	{Name: "CreateToken", Method: "POST", URI: "/prod/backend/{appId}/challenge"},
	{Name: "DeleteBackend", Method: "POST", URI: "/prod/backend/{appId}/environments/{backendEnvironmentName}/remove"},
	{Name: "DeleteBackendAPI", Method: "POST", URI: "/prod/backend/{appId}/api/{backendEnvironmentName}/remove"},
	{Name: "DeleteBackendAuth", Method: "POST", URI: "/prod/backend/{appId}/auth/{backendEnvironmentName}/remove"},
	{Name: "DeleteBackendStorage", Method: "POST", URI: "/prod/backend/{appId}/storage/{backendEnvironmentName}/remove"},
	{Name: "DeleteToken", Method: "POST", URI: "/prod/backend/{appId}/challenge/{sessionId}/remove"},
	{Name: "GenerateBackendAPIModels", Method: "POST", URI: "/prod/backend/{appId}/api/{backendEnvironmentName}/generateModels"},
	{Name: "GetBackend", Method: "POST", URI: "/prod/backend/{appId}/details"},
	{Name: "GetBackendAPI", Method: "POST", URI: "/prod/backend/{appId}/api/{backendEnvironmentName}/details"},
	{Name: "GetBackendAPIModels", Method: "POST", URI: "/prod/backend/{appId}/api/{backendEnvironmentName}/getModels"},
	{Name: "GetBackendAuth", Method: "POST", URI: "/prod/backend/{appId}/auth/{backendEnvironmentName}/details"},
	{Name: "GetBackendJob", Method: "GET", URI: "/prod/backend/{appId}/job/{backendEnvironmentName}/{jobId}"},
	{Name: "GetBackendStorage", Method: "POST", URI: "/prod/backend/{appId}/storage/{backendEnvironmentName}/details"},
	{Name: "GetToken", Method: "GET", URI: "/prod/backend/{appId}/challenge/{sessionId}"},
	{Name: "ImportBackendAuth", Method: "POST", URI: "/prod/backend/{appId}/auth/{backendEnvironmentName}/import"},
	{Name: "ImportBackendStorage", Method: "POST", URI: "/prod/backend/{appId}/storage/{backendEnvironmentName}/import"},
	{Name: "ListBackendJobs", Method: "POST", URI: "/prod/backend/{appId}/job/{backendEnvironmentName}"},
	{Name: "ListS3Buckets", Method: "POST", URI: "/prod/s3Buckets"},
	{Name: "RemoveAllBackends", Method: "POST", URI: "/prod/backend/{appId}/remove"},
	{Name: "RemoveBackendConfig", Method: "POST", URI: "/prod/backend/{appId}/config/remove"},
	{Name: "UpdateBackendAPI", Method: "POST", URI: "/prod/backend/{appId}/api/{backendEnvironmentName}"},
	{Name: "UpdateBackendAuth", Method: "POST", URI: "/prod/backend/{appId}/auth/{backendEnvironmentName}"},
	{Name: "UpdateBackendConfig", Method: "POST", URI: "/prod/backend/{appId}/config/update"},
	{Name: "UpdateBackendJob", Method: "POST", URI: "/prod/backend/{appId}/job/{backendEnvironmentName}/{jobId}"},
	{Name: "UpdateBackendStorage", Method: "POST", URI: "/prod/backend/{appId}/storage/{backendEnvironmentName}"},
}

var amplifyAdminUIOperationByName = func() map[string]amplifyAdminUIOperation {
	out := make(map[string]amplifyAdminUIOperation, len(amplifyAdminUIOperations))
	for _, op := range amplifyAdminUIOperations {
		out[op.Name] = op
	}
	return out
}()
