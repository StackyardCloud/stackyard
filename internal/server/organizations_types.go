package server

type organizationsDataType struct {
	Name string
}

// AWS Organizations data types sourced from:
// https://docs.aws.amazon.com/organizations/latest/APIReference/API_Types.html
var organizationsDataTypes = []organizationsDataType{
	{Name: "Account"},
	{Name: "Child"},
	{Name: "CreateAccountStatus"},
	{Name: "DelegatedAdministrator"},
	{Name: "DelegatedService"},
	{Name: "EffectivePolicy"},
	{Name: "EffectivePolicyValidationError"},
	{Name: "EnabledServicePrincipal"},
	{Name: "Handshake"},
	{Name: "HandshakeFilter"},
	{Name: "HandshakeParty"},
	{Name: "HandshakeResource"},
	{Name: "Organization"},
	{Name: "OrganizationalUnit"},
	{Name: "Parent"},
	{Name: "Policy"},
	{Name: "PolicySummary"},
	{Name: "PolicyTargetSummary"},
	{Name: "PolicyTypeSummary"},
	{Name: "ResourcePolicy"},
	{Name: "ResourcePolicySummary"},
	{Name: "ResponsibilityTransfer"},
	{Name: "Root"},
	{Name: "Tag"},
	{Name: "TransferParticipant"},
}

var organizationsDataTypeByName = func() map[string]organizationsDataType {
	out := make(map[string]organizationsDataType, len(organizationsDataTypes))
	for _, dt := range organizationsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
