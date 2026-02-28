package server

type managedServicesCMDataType struct {
	Name string
}

// AWS Managed Services Change Management data types sourced from:
// https://docs.aws.amazon.com/managedservices/latest/ApiReference-cm/API_Types.html
var managedServicesCMDataTypes = []managedServicesCMDataType{
	{Name: "AutomationStatus"},
	{Name: "ChangeTypeAccessLevel"},
	{Name: "ChangeTypeApprovalCondition"},
	{Name: "ChangeTypeApprovalRequirement"},
	{Name: "ChangeTypeAutomationStatus"},
	{Name: "ChangeTypeClassificationSummary"},
	{Name: "ChangeTypeOperationSummary"},
	{Name: "ChangeTypeVersion"},
	{Name: "ChangeTypeVersionSummary"},
	{Name: "Email"},
	{Name: "Filter"},
	{Name: "LinkedAttachment"},
	{Name: "RestrictedExecutionTime"},
	{Name: "Rfc"},
	{Name: "RfcActionState"},
	{Name: "RfcApprovalState"},
	{Name: "RfcApprovalStatus"},
	{Name: "RfcAttachment"},
	{Name: "RfcAttachmentSummary"},
	{Name: "RfcCorrespondence"},
	{Name: "RfcNotification"},
	{Name: "RfcRestrictedExecutionTimesOverride"},
	{Name: "RfcStatus"},
	{Name: "RfcSummary"},
	{Name: "TimeRange"},
	{Name: "UpdateRfc"},
	{Name: "UserType"},
}

var managedServicesCMDataTypeByName = func() map[string]managedServicesCMDataType {
	out := make(map[string]managedServicesCMDataType, len(managedServicesCMDataTypes))
	for _, t := range managedServicesCMDataTypes {
		out[t.Name] = t
	}
	return out
}()
