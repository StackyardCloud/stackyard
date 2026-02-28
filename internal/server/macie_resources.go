package server

type macieResource struct {
	Name string
	Path string
}

// Amazon Macie resources sourced from:
// https://docs.aws.amazon.com/macie/latest/APIReference/resources.html
var macieResources = []macieResource{
	{Name: "Account Administration", Path: "macie.html"},
	{Name: "Administrator", Path: "administrator.html"},
	{Name: "Administrator Disassociation", Path: "administrator-disassociate.html"},
	{Name: "Allow List", Path: "allow-lists-id.html"},
	{Name: "Allow Lists", Path: "allow-lists.html"},
	{Name: "AWS Organizations - Macie Administrator", Path: "admin.html"},
	{Name: "AWS Organizations - Macie Configuration", Path: "admin-configuration.html"},
	{Name: "Automated Sensitive Data Discovery - Accounts", Path: "automated-discovery-accounts.html"},
	{Name: "Automated Sensitive Data Discovery - Configuration", Path: "automated-discovery-configuration.html"},
	{Name: "Classification Job", Path: "jobs-jobid.html"},
	{Name: "Classification Job Creation", Path: "jobs.html"},
	{Name: "Classification Job List", Path: "jobs-list.html"},
	{Name: "Classification Results - Export Configuration", Path: "classification-export-configuration.html"},
	{Name: "Classification Scope", Path: "classification-scopes-id.html"},
	{Name: "Classification Scopes", Path: "classification-scopes.html"},
	{Name: "Custom Data Identifier", Path: "custom-data-identifiers-id.html"},
	{Name: "Custom Data Identifier Creation", Path: "custom-data-identifiers.html"},
	{Name: "Custom Data Identifier Descriptions", Path: "custom-data-identifiers-get.html"},
	{Name: "Custom Data Identifier List", Path: "custom-data-identifiers-list.html"},
	{Name: "Custom Data Identifier Testing", Path: "custom-data-identifiers-test.html"},
	{Name: "Data Sources - Amazon S3", Path: "datasources-s3.html"},
	{Name: "Data Sources - Amazon S3 Statistics", Path: "datasources-s3-statistics.html"},
	{Name: "Data Sources - Search", Path: "datasources-search-resources.html"},
	{Name: "Finding List", Path: "findings.html"},
	{Name: "Finding Samples", Path: "findings-sample.html"},
	{Name: "Finding Statistics", Path: "findings-statistics.html"},
	{Name: "Findings", Path: "findings-describe.html"},
	{Name: "Findings - Publication Configuration", Path: "findings-publication-configuration.html"},
	{Name: "Findings - Reveal Sensitive Data Occurrences", Path: "findings-findingid-reveal.html"},
	{Name: "Findings - Reveal Sensitive Data Occurrences Availability", Path: "findings-findingid-reveal-availability.html"},
	{Name: "Findings - Reveal Sensitive Data Occurrences Configuration", Path: "reveal-configuration.html"},
	{Name: "Findings Filter", Path: "findingsfilters-id.html"},
	{Name: "Findings Filters", Path: "findingsfilters.html"},
	{Name: "Invitation Acceptance", Path: "invitations-accept.html"},
	{Name: "Invitation Count", Path: "invitations-count.html"},
	{Name: "Invitation Decline", Path: "invitations-decline.html"},
	{Name: "Invitation Deletion", Path: "invitations-delete.html"},
	{Name: "Invitation List", Path: "invitations.html"},
	{Name: "Managed Data Identifiers", Path: "managed-data-identifiers-list.html"},
	{Name: "Master Account", Path: "master.html"},
	{Name: "Master Disassociation", Path: "master-disassociate.html"},
	{Name: "Member", Path: "members-id.html"},
	{Name: "Member Disassociation", Path: "members-disassociate-id.html"},
	{Name: "Member Status", Path: "macie-members-id.html"},
	{Name: "Members", Path: "members.html"},
	{Name: "Resource Sensitivity Profile", Path: "resource-profiles.html"},
	{Name: "Resource Sensitivity Profile - Artifacts", Path: "resource-profiles-artifacts.html"},
	{Name: "Resource Sensitivity Profile - Detections", Path: "resource-profiles-detections.html"},
	{Name: "Sensitivity Inspection Template", Path: "templates-sensitivity-inspections-id.html"},
	{Name: "Sensitivity Inspection Templates", Path: "templates-sensitivity-inspections.html"},
	{Name: "Tags", Path: "tags-resourcearn.html"},
	{Name: "Usage Statistics", Path: "usage-statistics.html"},
	{Name: "Usage Totals", Path: "usage.html"},
}

var macieResourceByName = func() map[string]macieResource {
	out := make(map[string]macieResource, len(macieResources))
	for _, resource := range macieResources {
		out[resource.Name] = resource
	}
	return out
}()
