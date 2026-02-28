package server

type rolesAnywhereDataType struct {
	Name string
}

// AWS IAM Roles Anywhere data types sourced from:
// https://docs.aws.amazon.com/rolesanywhere/latest/APIReference/API_Types.html
var rolesAnywhereDataTypes = []rolesAnywhereDataType{
	{Name: "AttributeMapping"},
	{Name: "CredentialSummary"},
	{Name: "CrlDetail"},
	{Name: "InstanceProperty"},
	{Name: "MappingRule"},
	{Name: "NotificationSetting"},
	{Name: "NotificationSettingDetail"},
	{Name: "NotificationSettingKey"},
	{Name: "ProfileDetail"},
	{Name: "Source"},
	{Name: "SourceData"},
	{Name: "SubjectDetail"},
	{Name: "SubjectSummary"},
	{Name: "Tag"},
	{Name: "TrustAnchorDetail"},
}

var rolesAnywhereDataTypeByName = func() map[string]rolesAnywhereDataType {
	out := make(map[string]rolesAnywhereDataType, len(rolesAnywhereDataTypes))
	for _, dt := range rolesAnywhereDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
