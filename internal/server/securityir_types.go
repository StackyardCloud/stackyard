package server

type securityIRType struct {
	Name string
}

// AWS Security Incident Response data types sourced from:
// https://docs.aws.amazon.com/security-ir/latest/APIReference/API_Types.html
var securityIRTypes = []securityIRType{
	{Name: "CaseAttachmentAttributes"},
	{Name: "CaseEditItem"},
	{Name: "CaseMetadataEntry"},
	{Name: "GetMembershipAccountDetailError"},
	{Name: "GetMembershipAccountDetailItem"},
	{Name: "ImpactedAwsRegion"},
	{Name: "IncidentResponder"},
	{Name: "InvestigationAction"},
	{Name: "InvestigationFeedback"},
	{Name: "ListCasesItem"},
	{Name: "ListCommentsItem"},
	{Name: "ListMembershipItem"},
	{Name: "MembershipAccountsConfigurations"},
	{Name: "MembershipAccountsConfigurationsUpdate"},
	{Name: "OptInFeature"},
	{Name: "ThreatActorIp"},
	{Name: "UpdateResolverType"},
	{Name: "ValidationExceptionField"},
	{Name: "Watcher"},
}

var securityIRTypeByName = func() map[string]securityIRType {
	out := make(map[string]securityIRType, len(securityIRTypes))
	for _, dt := range securityIRTypes {
		out[dt.Name] = dt
	}
	return out
}()
