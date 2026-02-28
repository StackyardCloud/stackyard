package server

type macieOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Macie operations sourced from:
// https://docs.aws.amazon.com/macie/latest/APIReference/operations.html
var macieOperations = []macieOperation{
	{Name: "AcceptInvitation", Method: "POST", URI: "/invitations/accept"},
	{Name: "BatchGetCustomDataIdentifiers", Method: "POST", URI: "/custom-data-identifiers/get"},
	{Name: "BatchUpdateAutomatedDiscoveryAccounts", Method: "PATCH", URI: "/automated-discovery/accounts"},
	{Name: "CreateAllowList", Method: "POST", URI: "/allow-lists"},
	{Name: "CreateClassificationJob", Method: "POST", URI: "/jobs"},
	{Name: "CreateCustomDataIdentifier", Method: "POST", URI: "/custom-data-identifiers"},
	{Name: "CreateFindingsFilter", Method: "POST", URI: "/findingsfilters"},
	{Name: "CreateInvitations", Method: "POST", URI: "/invitations"},
	{Name: "CreateMember", Method: "POST", URI: "/members"},
	{Name: "CreateSampleFindings", Method: "POST", URI: "/findings/sample"},
	{Name: "DeclineInvitations", Method: "POST", URI: "/invitations/decline"},
	{Name: "DeleteAllowList", Method: "DELETE", URI: "/allow-lists/{id}"},
	{Name: "DeleteCustomDataIdentifier", Method: "DELETE", URI: "/custom-data-identifiers/{id}"},
	{Name: "DeleteFindingsFilter", Method: "DELETE", URI: "/findingsfilters/{id}"},
	{Name: "DeleteInvitations", Method: "POST", URI: "/invitations/delete"},
	{Name: "DeleteMember", Method: "DELETE", URI: "/members/{id}"},
	{Name: "DescribeBuckets", Method: "POST", URI: "/datasources/s3"},
	{Name: "DescribeClassificationJob", Method: "GET", URI: "/jobs/{jobId}"},
	{Name: "DescribeOrganizationConfiguration", Method: "GET", URI: "/admin/configuration"},
	{Name: "DisableMacie", Method: "DELETE", URI: "/macie"},
	{Name: "DisableOrganizationAdminAccount", Method: "DELETE", URI: "/admin"},
	{Name: "DisassociateFromAdministratorAccount", Method: "POST", URI: "/administrator/disassociate"},
	{Name: "DisassociateFromMasterAccount", Method: "POST", URI: "/master/disassociate"},
	{Name: "DisassociateMember", Method: "POST", URI: "/members/disassociate/{id}"},
	{Name: "EnableMacie", Method: "POST", URI: "/macie"},
	{Name: "EnableOrganizationAdminAccount", Method: "POST", URI: "/admin"},
	{Name: "GetAdministratorAccount", Method: "GET", URI: "/administrator"},
	{Name: "GetAllowList", Method: "GET", URI: "/allow-lists/{id}"},
	{Name: "GetAutomatedDiscoveryConfiguration", Method: "GET", URI: "/automated-discovery/configuration"},
	{Name: "GetBucketStatistics", Method: "POST", URI: "/datasources/s3/statistics"},
	{Name: "GetClassificationExportConfiguration", Method: "GET", URI: "/classification-export-configuration"},
	{Name: "GetClassificationScope", Method: "GET", URI: "/classification-scopes/{id}"},
	{Name: "GetCustomDataIdentifier", Method: "GET", URI: "/custom-data-identifiers/{id}"},
	{Name: "GetFindingStatistics", Method: "POST", URI: "/findings/statistics"},
	{Name: "GetFindings", Method: "POST", URI: "/findings/describe"},
	{Name: "GetFindingsFilter", Method: "GET", URI: "/findingsfilters/{id}"},
	{Name: "GetFindingsPublicationConfiguration", Method: "GET", URI: "/findings-publication-configuration"},
	{Name: "GetInvitationsCount", Method: "GET", URI: "/invitations/count"},
	{Name: "GetMacieSession", Method: "GET", URI: "/macie"},
	{Name: "GetMasterAccount", Method: "GET", URI: "/master"},
	{Name: "GetMember", Method: "GET", URI: "/members/{id}"},
	{Name: "GetResourceProfile", Method: "GET", URI: "/resource-profiles"},
	{Name: "GetRevealConfiguration", Method: "GET", URI: "/reveal-configuration"},
	{Name: "GetSensitiveDataOccurrences", Method: "GET", URI: "/findings/{findingId}/reveal"},
	{Name: "GetSensitiveDataOccurrencesAvailability", Method: "GET", URI: "/findings/{findingId}/reveal/availability"},
	{Name: "GetSensitivityInspectionTemplate", Method: "GET", URI: "/templates/sensitivity-inspections/{id}"},
	{Name: "GetUsageStatistics", Method: "POST", URI: "/usage/statistics"},
	{Name: "GetUsageTotals", Method: "GET", URI: "/usage"},
	{Name: "ListAllowLists", Method: "GET", URI: "/allow-lists"},
	{Name: "ListAutomatedDiscoveryAccounts", Method: "GET", URI: "/automated-discovery/accounts"},
	{Name: "ListClassificationJobs", Method: "POST", URI: "/jobs/list"},
	{Name: "ListClassificationScopes", Method: "GET", URI: "/classification-scopes"},
	{Name: "ListCustomDataIdentifiers", Method: "POST", URI: "/custom-data-identifiers/list"},
	{Name: "ListFindings", Method: "POST", URI: "/findings"},
	{Name: "ListFindingsFilters", Method: "GET", URI: "/findingsfilters"},
	{Name: "ListInvitations", Method: "GET", URI: "/invitations"},
	{Name: "ListManagedDataIdentifiers", Method: "POST", URI: "/managed-data-identifiers/list"},
	{Name: "ListMembers", Method: "GET", URI: "/members"},
	{Name: "ListOrganizationAdminAccounts", Method: "GET", URI: "/admin"},
	{Name: "ListResourceProfileArtifacts", Method: "GET", URI: "/resource-profiles/artifacts"},
	{Name: "ListResourceProfileDetections", Method: "GET", URI: "/resource-profiles/detections"},
	{Name: "ListSensitivityInspectionTemplates", Method: "GET", URI: "/templates/sensitivity-inspections"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutClassificationExportConfiguration", Method: "PUT", URI: "/classification-export-configuration"},
	{Name: "PutFindingsPublicationConfiguration", Method: "PUT", URI: "/findings-publication-configuration"},
	{Name: "SearchResources", Method: "POST", URI: "/datasources/search-resources"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "TestCustomDataIdentifier", Method: "POST", URI: "/custom-data-identifiers/test"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateAllowList", Method: "PUT", URI: "/allow-lists/{id}"},
	{Name: "UpdateAutomatedDiscoveryConfiguration", Method: "PUT", URI: "/automated-discovery/configuration"},
	{Name: "UpdateClassificationJob", Method: "PATCH", URI: "/jobs/{jobId}"},
	{Name: "UpdateClassificationScope", Method: "PATCH", URI: "/classification-scopes/{id}"},
	{Name: "UpdateFindingsFilter", Method: "PATCH", URI: "/findingsfilters/{id}"},
	{Name: "UpdateMacieSession", Method: "PATCH", URI: "/macie"},
	{Name: "UpdateMemberSession", Method: "PATCH", URI: "/macie/members/{id}"},
	{Name: "UpdateOrganizationConfiguration", Method: "PATCH", URI: "/admin/configuration"},
	{Name: "UpdateResourceProfile", Method: "PATCH", URI: "/resource-profiles"},
	{Name: "UpdateResourceProfileDetections", Method: "PATCH", URI: "/resource-profiles/detections"},
	{Name: "UpdateRevealConfiguration", Method: "PUT", URI: "/reveal-configuration"},
	{Name: "UpdateSensitivityInspectionTemplate", Method: "PUT", URI: "/templates/sensitivity-inspections/{id}"},
}

var macieOperationByName = func() map[string]macieOperation {
	out := make(map[string]macieOperation, len(macieOperations))
	for _, op := range macieOperations {
		out[op.Name] = op
	}
	return out
}()
