package server

type amplifyDataType struct {
	Name string
}

// AWS Amplify data types sourced from:
// https://docs.aws.amazon.com/amplify/latest/APIReference/API_Types.html
var amplifyDataTypes = []amplifyDataType{
	{Name: "App"},
	{Name: "Artifact"},
	{Name: "AutoBranchCreationConfig"},
	{Name: "Backend"},
	{Name: "BackendEnvironment"},
	{Name: "Branch"},
	{Name: "CacheConfig"},
	{Name: "Certificate"},
	{Name: "CertificateSettings"},
	{Name: "CustomRule"},
	{Name: "DomainAssociation"},
	{Name: "Job"},
	{Name: "JobConfig"},
	{Name: "JobSummary"},
	{Name: "ProductionBranch"},
	{Name: "Step"},
	{Name: "SubDomain"},
	{Name: "SubDomainSetting"},
	{Name: "WafConfiguration"},
	{Name: "Webhook"},
}

var amplifyDataTypeByName = func() map[string]amplifyDataType {
	out := make(map[string]amplifyDataType, len(amplifyDataTypes))
	for _, dt := range amplifyDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
