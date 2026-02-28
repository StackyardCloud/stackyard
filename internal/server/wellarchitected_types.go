package server

type wellArchitectedDataType struct {
	Name string
}

// AWS Well-Architected Tool data types sourced from:
// https://docs.aws.amazon.com/wellarchitected/latest/APIReference/API_Types.html
var wellArchitectedDataTypes = []wellArchitectedDataType{
	{Name: "AccountJiraConfigurationInput"},
	{Name: "AccountJiraConfigurationOutput"},
	{Name: "AdditionalResources"},
	{Name: "Answer"},
	{Name: "AnswerSummary"},
	{Name: "BestPractice"},
	{Name: "CheckDetail"},
	{Name: "CheckSummary"},
	{Name: "Choice"},
	{Name: "ChoiceAnswer"},
	{Name: "ChoiceAnswerSummary"},
	{Name: "ChoiceContent"},
	{Name: "ChoiceImprovementPlan"},
	{Name: "ChoiceUpdate"},
	{Name: "ConsolidatedReportMetric"},
	{Name: "ImprovementSummary"},
	{Name: "JiraConfiguration"},
	{Name: "JiraSelectedQuestionConfiguration"},
	{Name: "Lens"},
	{Name: "LensMetric"},
	{Name: "LensReview"},
	{Name: "LensReviewReport"},
	{Name: "LensReviewSummary"},
	{Name: "LensShareSummary"},
	{Name: "LensSummary"},
	{Name: "LensUpgradeSummary"},
	{Name: "Milestone"},
	{Name: "MilestoneSummary"},
	{Name: "NotificationSummary"},
	{Name: "PillarDifference"},
	{Name: "PillarMetric"},
	{Name: "PillarReviewSummary"},
	{Name: "Profile"},
	{Name: "ProfileChoice"},
	{Name: "ProfileNotificationSummary"},
	{Name: "ProfileQuestion"},
	{Name: "ProfileQuestionUpdate"},
	{Name: "ProfileShareSummary"},
	{Name: "ProfileSummary"},
	{Name: "ProfileTemplate"},
	{Name: "ProfileTemplateChoice"},
	{Name: "ProfileTemplateQuestion"},
	{Name: "QuestionDifference"},
	{Name: "QuestionMetric"},
	{Name: "ReviewTemplate"},
	{Name: "ReviewTemplateAnswer"},
	{Name: "ReviewTemplateAnswerSummary"},
	{Name: "ReviewTemplateLensReview"},
	{Name: "ReviewTemplatePillarReviewSummary"},
	{Name: "ReviewTemplateSummary"},
	{Name: "SelectedPillar"},
	{Name: "ShareInvitation"},
	{Name: "ShareInvitationSummary"},
	{Name: "TemplateShareSummary"},
	{Name: "UpgradeReviewTemplateLensReview"},
	{Name: "ValidationExceptionField"},
	{Name: "VersionDifferences"},
	{Name: "Workload"},
	{Name: "WorkloadDiscoveryConfig"},
	{Name: "WorkloadJiraConfigurationInput"},
	{Name: "WorkloadJiraConfigurationOutput"},
	{Name: "WorkloadProfile"},
	{Name: "WorkloadShare"},
	{Name: "WorkloadShareSummary"},
	{Name: "WorkloadSummary"},
}

var wellArchitectedDataTypeByName = func() map[string]wellArchitectedDataType {
	out := make(map[string]wellArchitectedDataType, len(wellArchitectedDataTypes))
	for _, t := range wellArchitectedDataTypes {
		out[t.Name] = t
	}
	return out
}()
