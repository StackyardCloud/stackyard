package server

type incidentManagerOperation struct {
	Name string
}

// AWS Systems Manager Incident Manager operations sourced from:
// https://docs.aws.amazon.com/incident-manager/latest/APIReference/API_Operations.html
var incidentManagerOperations = []incidentManagerOperation{
	{Name: "BatchGetIncidentFindings"},
	{Name: "CreateReplicationSet"},
	{Name: "CreateResponsePlan"},
	{Name: "CreateTimelineEvent"},
	{Name: "DeleteIncidentRecord"},
	{Name: "DeleteReplicationSet"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteResponsePlan"},
	{Name: "DeleteTimelineEvent"},
	{Name: "GetIncidentRecord"},
	{Name: "GetReplicationSet"},
	{Name: "GetResourcePolicies"},
	{Name: "GetResponsePlan"},
	{Name: "GetTimelineEvent"},
	{Name: "ListIncidentFindings"},
	{Name: "ListIncidentRecords"},
	{Name: "ListRelatedItems"},
	{Name: "ListReplicationSets"},
	{Name: "ListResponsePlans"},
	{Name: "ListTagsForResource"},
	{Name: "ListTimelineEvents"},
	{Name: "PutResourcePolicy"},
	{Name: "StartIncident"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDeletionProtection"},
	{Name: "UpdateIncidentRecord"},
	{Name: "UpdateRelatedItems"},
	{Name: "UpdateReplicationSet"},
	{Name: "UpdateResponsePlan"},
	{Name: "UpdateTimelineEvent"},
	{Name: "AcceptPage"},
	{Name: "ActivateContactChannel"},
	{Name: "CreateContact"},
	{Name: "CreateContactChannel"},
	{Name: "CreateRotation"},
	{Name: "CreateRotationOverride"},
	{Name: "DeactivateContactChannel"},
	{Name: "DeleteContact"},
	{Name: "DeleteContactChannel"},
	{Name: "DeleteRotation"},
	{Name: "DeleteRotationOverride"},
	{Name: "DescribeEngagement"},
	{Name: "DescribePage"},
	{Name: "GetContact"},
	{Name: "GetContactChannel"},
	{Name: "GetContactPolicy"},
	{Name: "GetRotation"},
	{Name: "GetRotationOverride"},
	{Name: "ListContactChannels"},
	{Name: "ListContacts"},
	{Name: "ListEngagements"},
	{Name: "ListPageReceipts"},
	{Name: "ListPageResolutions"},
	{Name: "ListPagesByContact"},
	{Name: "ListPagesByEngagement"},
	{Name: "ListPreviewRotationShifts"},
	{Name: "ListRotationOverrides"},
	{Name: "ListRotations"},
	{Name: "ListRotationShifts"},
	{Name: "PutContactPolicy"},
	{Name: "SendActivationCode"},
	{Name: "StartEngagement"},
	{Name: "StopEngagement"},
	{Name: "UpdateContact"},
	{Name: "UpdateContactChannel"},
	{Name: "UpdateRotation"},
}

var incidentManagerOperationByName = func() map[string]incidentManagerOperation {
	out := make(map[string]incidentManagerOperation, len(incidentManagerOperations))
	for _, op := range incidentManagerOperations {
		out[op.Name] = op
	}
	return out
}()
