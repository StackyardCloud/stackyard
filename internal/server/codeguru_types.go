package server

type codeGuruDataType struct {
	Name string
}

// Amazon CodeGuru Reviewer data types sourced from:
// https://docs.aws.amazon.com/codeguru/latest/reviewer-api/API_Types.html
var codeGuruDataTypes = []codeGuruDataType{
	{Name: "AuthorizationToken"},
	{Name: "BranchDiffSourceCodeType"},
	{Name: "CodeArtifacts"},
	{Name: "CodeCommitRepository"},
	{Name: "CodeReview"},
	{Name: "CodeReviewSummary"},
	{Name: "CodeReviewType"},
	{Name: "CommitDiffSourceCodeType"},
	{Name: "EventInfo"},
	{Name: "FindingsMetricsData"},
	{Name: "GitHubRepository"},
	{Name: "KMSKeyDetails"},
	{Name: "Metrics"},
	{Name: "MetricsSummary"},
	{Name: "RecommendationFeedback"},
	{Name: "RecommendationFeedbackSummary"},
	{Name: "RecommendationSummary"},
	{Name: "Repository"},
	{Name: "RepositoryAnalysis"},
	{Name: "RepositoryAssociation"},
	{Name: "RepositoryAssociationSummary"},
	{Name: "RepositoryHeadSourceCodeType"},
	{Name: "RequestMetadata"},
	{Name: "RuleMetadata"},
	{Name: "S3BucketRepository"},
	{Name: "S3Repository"},
	{Name: "S3RepositoryDetails"},
	{Name: "SourceCodeType"},
	{Name: "ThirdPartyRepository"},
	{Name: "ThirdPartySourceRepository"},
}

var codeGuruDataTypeByName = func() map[string]codeGuruDataType {
	out := make(map[string]codeGuruDataType, len(codeGuruDataTypes))
	for _, dt := range codeGuruDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
