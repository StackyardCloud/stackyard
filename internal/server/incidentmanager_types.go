package server

type incidentManagerDataType struct {
	Name string
}

// AWS Systems Manager Incident Manager data types sourced from:
// https://docs.aws.amazon.com/incident-manager/latest/APIReference/API_Types.html
var incidentManagerDataTypes = []incidentManagerDataType{
	{Name: "Action"},
	{Name: "AddRegionAction"},
	{Name: "AttributeValueList"},
	{Name: "AutomationExecution"},
	{Name: "BatchGetIncidentFindingsError"},
	{Name: "ChatChannel"},
	{Name: "CloudFormationStackUpdate"},
	{Name: "CodeDeployDeployment"},
	{Name: "Condition"},
	{Name: "DeleteRegionAction"},
	{Name: "DynamicSsmParameterValue"},
	{Name: "EmptyChatChannel"},
	{Name: "EventReference"},
	{Name: "EventSummary"},
	{Name: "Filter"},
	{Name: "Finding"},
	{Name: "FindingDetails"},
	{Name: "FindingSummary"},
	{Name: "IncidentRecord"},
	{Name: "IncidentRecordSource"},
	{Name: "IncidentRecordSummary"},
	{Name: "IncidentTemplate"},
	{Name: "Integration"},
	{Name: "ItemIdentifier"},
	{Name: "ItemValue"},
	{Name: "NotificationTargetItem"},
	{Name: "PagerDutyConfiguration"},
	{Name: "PagerDutyIncidentConfiguration"},
	{Name: "PagerDutyIncidentDetail"},
	{Name: "RegionInfo"},
	{Name: "RegionMapInputValue"},
	{Name: "RelatedItem"},
	{Name: "RelatedItemsUpdate"},
	{Name: "ReplicationSet"},
	{Name: "ResourcePolicy"},
	{Name: "ResponsePlanSummary"},
	{Name: "SsmAutomation"},
	{Name: "TimelineEvent"},
	{Name: "TriggerDetails"},
	{Name: "UpdateReplicationSetAction"},
	{Name: "ChannelTargetInfo"},
	{Name: "Contact"},
	{Name: "ContactChannel"},
	{Name: "ContactChannelAddress"},
	{Name: "ContactTargetInfo"},
	{Name: "CoverageTime"},
	{Name: "DependentEntity"},
	{Name: "Engagement"},
	{Name: "HandOffTime"},
	{Name: "MonthlySetting"},
	{Name: "Page"},
	{Name: "Plan"},
	{Name: "PreviewOverride"},
	{Name: "Receipt"},
	{Name: "RecurrenceSettings"},
	{Name: "ResolutionContact"},
	{Name: "Rotation"},
	{Name: "RotationOverride"},
	{Name: "RotationShift"},
	{Name: "ShiftDetails"},
	{Name: "Stage"},
	{Name: "Tag"},
	{Name: "Target"},
	{Name: "TimeRange"},
	{Name: "ValidationExceptionField"},
	{Name: "WeeklySetting"},
}

var incidentManagerDataTypeByName = func() map[string]incidentManagerDataType {
	out := make(map[string]incidentManagerDataType, len(incidentManagerDataTypes))
	for _, dt := range incidentManagerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
