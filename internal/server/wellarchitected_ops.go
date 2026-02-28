package server

type wellArchitectedOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Well-Architected Tool operations sourced from:
// https://docs.aws.amazon.com/wellarchitected/latest/APIReference/API_Operations.html
var wellArchitectedOperations = []wellArchitectedOperation{
	{Name: "AssociateLenses", Method: "PATCH", URI: "/workloads/{WorkloadId}/associateLenses"},
	{Name: "AssociateProfiles", Method: "PATCH", URI: "/workloads/{WorkloadId}/associateProfiles"},
	{Name: "CreateLensShare", Method: "POST", URI: "/lenses/{LensAlias}/shares"},
	{Name: "CreateLensVersion", Method: "POST", URI: "/lenses/{LensAlias}/versions"},
	{Name: "CreateMilestone", Method: "POST", URI: "/workloads/{WorkloadId}/milestones"},
	{Name: "CreateProfile", Method: "POST", URI: "/profiles"},
	{Name: "CreateProfileShare", Method: "POST", URI: "/profiles/{ProfileArn}/shares"},
	{Name: "CreateReviewTemplate", Method: "POST", URI: "/reviewTemplates"},
	{Name: "CreateTemplateShare", Method: "POST", URI: "/templates/shares/{TemplateArn}"},
	{Name: "CreateWorkload", Method: "POST", URI: "/workloads"},
	{Name: "CreateWorkloadShare", Method: "POST", URI: "/workloads/{WorkloadId}/shares"},
	{Name: "DeleteLens", Method: "DELETE", URI: "/lenses/{LensAlias}"},
	{Name: "DeleteLensShare", Method: "DELETE", URI: "/lenses/{LensAlias}/shares/{ShareId}"},
	{Name: "DeleteProfile", Method: "DELETE", URI: "/profiles/{ProfileArn}"},
	{Name: "DeleteProfileShare", Method: "DELETE", URI: "/profiles/{ProfileArn}/shares/{ShareId}"},
	{Name: "DeleteReviewTemplate", Method: "DELETE", URI: "/reviewTemplates/{TemplateArn}"},
	{Name: "DeleteTemplateShare", Method: "DELETE", URI: "/templates/shares/{TemplateArn}/{ShareId}"},
	{Name: "DeleteWorkload", Method: "DELETE", URI: "/workloads/{WorkloadId}"},
	{Name: "DeleteWorkloadShare", Method: "DELETE", URI: "/workloads/{WorkloadId}/shares/{ShareId}"},
	{Name: "DisassociateLenses", Method: "PATCH", URI: "/workloads/{WorkloadId}/disassociateLenses"},
	{Name: "DisassociateProfiles", Method: "PATCH", URI: "/workloads/{WorkloadId}/disassociateProfiles"},
	{Name: "ExportLens", Method: "GET", URI: "/lenses/{LensAlias}/export"},
	{Name: "GetAnswer", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/answers/{QuestionId}"},
	{Name: "GetConsolidatedReport", Method: "GET", URI: "/consolidatedReport"},
	{Name: "GetGlobalSettings", Method: "GET", URI: "/global-settings"},
	{Name: "GetLens", Method: "GET", URI: "/lenses/{LensAlias}"},
	{Name: "GetLensReview", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}"},
	{Name: "GetLensReviewReport", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/report"},
	{Name: "GetLensVersionDifference", Method: "GET", URI: "/lenses/{LensAlias}/versionDifference"},
	{Name: "GetMilestone", Method: "GET", URI: "/workloads/{WorkloadId}/milestones/{MilestoneNumber}"},
	{Name: "GetProfile", Method: "GET", URI: "/profiles/{ProfileArn}"},
	{Name: "GetProfileTemplate", Method: "GET", URI: "/profileTemplate"},
	{Name: "GetReviewTemplate", Method: "GET", URI: "/reviewTemplates/{TemplateArn}"},
	{Name: "GetReviewTemplateAnswer", Method: "GET", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}/answers/{QuestionId}"},
	{Name: "GetReviewTemplateLensReview", Method: "GET", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}"},
	{Name: "GetWorkload", Method: "GET", URI: "/workloads/{WorkloadId}"},
	{Name: "ImportLens", Method: "PUT", URI: "/importLens"},
	{Name: "ListAnswers", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/answers"},
	{Name: "ListCheckDetails", Method: "POST", URI: "/workloads/{WorkloadId}/checks"},
	{Name: "ListCheckSummaries", Method: "POST", URI: "/workloads/{WorkloadId}/checkSummaries"},
	{Name: "ListLensReviewImprovements", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/improvements"},
	{Name: "ListLensReviews", Method: "GET", URI: "/workloads/{WorkloadId}/lensReviews"},
	{Name: "ListLensShares", Method: "GET", URI: "/lenses/{LensAlias}/shares"},
	{Name: "ListLenses", Method: "GET", URI: "/lenses"},
	{Name: "ListMilestones", Method: "POST", URI: "/workloads/{WorkloadId}/milestonesSummaries"},
	{Name: "ListNotifications", Method: "POST", URI: "/notifications"},
	{Name: "ListProfileNotifications", Method: "GET", URI: "/profileNotifications/"},
	{Name: "ListProfileShares", Method: "GET", URI: "/profiles/{ProfileArn}/shares"},
	{Name: "ListProfiles", Method: "GET", URI: "/profileSummaries"},
	{Name: "ListReviewTemplateAnswers", Method: "GET", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}/answers"},
	{Name: "ListReviewTemplates", Method: "GET", URI: "/reviewTemplates"},
	{Name: "ListShareInvitations", Method: "GET", URI: "/shareInvitations"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{WorkloadArn}"},
	{Name: "ListTemplateShares", Method: "GET", URI: "/templates/shares/{TemplateArn}"},
	{Name: "ListWorkloadShares", Method: "GET", URI: "/workloads/{WorkloadId}/shares"},
	{Name: "ListWorkloads", Method: "POST", URI: "/workloadsSummaries"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{WorkloadArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/WorkloadArn"},
	{Name: "UpdateAnswer", Method: "PATCH", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/answers/{QuestionId}"},
	{Name: "UpdateGlobalSettings", Method: "PATCH", URI: "/global-settings"},
	{Name: "UpdateIntegration", Method: "POST", URI: "/workloads/{WorkloadId}/updateIntegration"},
	{Name: "UpdateLensReview", Method: "PATCH", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}"},
	{Name: "UpdateProfile", Method: "PATCH", URI: "/profiles/{ProfileArn}"},
	{Name: "UpdateReviewTemplate", Method: "PATCH", URI: "/reviewTemplates/{TemplateArn}"},
	{Name: "UpdateReviewTemplateAnswer", Method: "PATCH", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}/answers/{QuestionId}"},
	{Name: "UpdateReviewTemplateLensReview", Method: "PATCH", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}"},
	{Name: "UpdateShareInvitation", Method: "PATCH", URI: "/shareInvitations/{ShareInvitationId}"},
	{Name: "UpdateWorkload", Method: "PATCH", URI: "/workloads/{WorkloadId}"},
	{Name: "UpdateWorkloadShare", Method: "PATCH", URI: "/workloads/{WorkloadId}/shares/{ShareId}"},
	{Name: "UpgradeLensReview", Method: "PUT", URI: "/workloads/{WorkloadId}/lensReviews/{LensAlias}/upgrade"},
	{Name: "UpgradeProfileVersion", Method: "PUT", URI: "/workloads/{WorkloadId}/profiles/{ProfileArn}/upgrade"},
	{Name: "UpgradeReviewTemplateLensReview", Method: "PUT", URI: "/reviewTemplates/{TemplateArn}/lensReviews/{LensAlias}/upgrade"},
}

var wellArchitectedOperationByName = func() map[string]wellArchitectedOperation {
	out := make(map[string]wellArchitectedOperation, len(wellArchitectedOperations))
	for _, op := range wellArchitectedOperations {
		out[op.Name] = op
	}
	return out
}()
