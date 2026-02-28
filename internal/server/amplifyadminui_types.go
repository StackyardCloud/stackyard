package server

type amplifyAdminUIResource struct {
	Name string
}

// AWS Amplify Admin UI resources sourced from:
// https://docs.aws.amazon.com/amplify-admin-ui/latest/APIReference/resources.html
var amplifyAdminUIResources = []amplifyAdminUIResource{
	{Name: "Backend"},
	{Name: "Backend appId Api"},
	{Name: "Backend appId Api backendEnvironmentName"},
	{Name: "Backend appId Api backendEnvironmentName Details"},
	{Name: "Backend appId Api backendEnvironmentName GenerateModels"},
	{Name: "Backend appId Api backendEnvironmentName GetModels"},
	{Name: "Backend appId Api backendEnvironmentName Remove"},
	{Name: "Backend appId Auth"},
	{Name: "Backend appId Auth backendEnvironmentName"},
	{Name: "Backend appId Auth backendEnvironmentName Details"},
	{Name: "Backend appId Auth backendEnvironmentName Import"},
	{Name: "Backend appId Auth backendEnvironmentName Remove"},
	{Name: "Backend appId Challenge"},
	{Name: "Backend appId Challenge sessionId"},
	{Name: "Backend appId Challenge sessionId Remove"},
	{Name: "Backend appId Config"},
	{Name: "Backend appId Config Remove"},
	{Name: "Backend appId Config Update"},
	{Name: "Backend appId Details"},
	{Name: "Backend appId Environments backendEnvironmentName Clone"},
	{Name: "Backend appId Environments backendEnvironmentName Remove"},
	{Name: "Backend appId Job backendEnvironmentName"},
	{Name: "Backend appId Job backendEnvironmentName jobId"},
	{Name: "Backend appId Remove"},
	{Name: "Backend appId Storage"},
	{Name: "Backend appId Storage backendEnvironmentName"},
	{Name: "Backend appId Storage backendEnvironmentName Details"},
	{Name: "Backend appId Storage backendEnvironmentName Import"},
	{Name: "Backend appId Storage backendEnvironmentName Remove"},
	{Name: "S3Buckets"},
}

var amplifyAdminUIResourceByName = func() map[string]amplifyAdminUIResource {
	out := make(map[string]amplifyAdminUIResource, len(amplifyAdminUIResources))
	for _, r := range amplifyAdminUIResources {
		out[r.Name] = r
	}
	return out
}()
