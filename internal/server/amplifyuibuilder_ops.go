package server

type amplifyUIBuilderOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Amplify UI Builder operations sourced from:
// https://docs.aws.amazon.com/amplifyuibuilder/latest/APIReference/API_Operations.html
var amplifyUIBuilderOperations = []amplifyUIBuilderOperation{
	{Name: "CreateComponent", Method: "POST", URI: "/app/{appId}/environment/{environmentName}/components?clientToken={clientToken}"},
	{Name: "CreateForm", Method: "POST", URI: "/app/{appId}/environment/{environmentName}/forms?clientToken={clientToken}"},
	{Name: "CreateTheme", Method: "POST", URI: "/app/{appId}/environment/{environmentName}/themes?clientToken={clientToken}"},
	{Name: "DeleteComponent", Method: "DELETE", URI: "/app/{appId}/environment/{environmentName}/components/{id}"},
	{Name: "DeleteForm", Method: "DELETE", URI: "/app/{appId}/environment/{environmentName}/forms/{id}"},
	{Name: "DeleteTheme", Method: "DELETE", URI: "/app/{appId}/environment/{environmentName}/themes/{id}"},
	{Name: "ExchangeCodeForToken", Method: "POST", URI: "/tokens/{provider}"},
	{Name: "ExportComponents", Method: "GET", URI: "/export/app/{appId}/environment/{environmentName}/components?nextToken={nextToken}"},
	{Name: "ExportForms", Method: "GET", URI: "/export/app/{appId}/environment/{environmentName}/forms?nextToken={nextToken}"},
	{Name: "ExportThemes", Method: "GET", URI: "/export/app/{appId}/environment/{environmentName}/themes?nextToken={nextToken}"},
	{Name: "GetCodegenJob", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/codegen-jobs/{id}"},
	{Name: "GetComponent", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/components/{id}"},
	{Name: "GetForm", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/forms/{id}"},
	{Name: "GetMetadata", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/metadata"},
	{Name: "GetTheme", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/themes/{id}"},
	{Name: "ListCodegenJobs", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/codegen-jobs?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListComponents", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/components?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListForms", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/forms?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListThemes", Method: "GET", URI: "/app/{appId}/environment/{environmentName}/themes?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "PutMetadataFlag", Method: "PUT", URI: "/app/{appId}/environment/{environmentName}/metadata/features/{featureName}"},
	{Name: "RefreshToken", Method: "POST", URI: "/tokens/{provider}/refresh"},
	{Name: "StartCodegenJob", Method: "POST", URI: "/app/{appId}/environment/{environmentName}/codegen-jobs?clientToken={clientToken}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateComponent", Method: "PATCH", URI: "/app/{appId}/environment/{environmentName}/components/{id}?clientToken={clientToken}"},
	{Name: "UpdateForm", Method: "PATCH", URI: "/app/{appId}/environment/{environmentName}/forms/{id}?clientToken={clientToken}"},
	{Name: "UpdateTheme", Method: "PATCH", URI: "/app/{appId}/environment/{environmentName}/themes/{id}?clientToken={clientToken}"},
}

var amplifyUIBuilderOperationByName = func() map[string]amplifyUIBuilderOperation {
	out := make(map[string]amplifyUIBuilderOperation, len(amplifyUIBuilderOperations))
	for _, op := range amplifyUIBuilderOperations {
		out[op.Name] = op
	}
	return out
}()
