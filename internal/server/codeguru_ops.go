package server

type codeGuruOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CodeGuru Reviewer operations sourced from:
// https://docs.aws.amazon.com/codeguru/latest/reviewer-api/API_Operations.html
var codeGuruOperations = []codeGuruOperation{
	{Name: "AssociateRepository", Method: "POST", URI: "/associations"},
	{Name: "CreateCodeReview", Method: "POST", URI: "/codereviews"},
	{Name: "CreateCodeReviewInternal", Method: "POST", URI: "/createCodeReviewInternal"},
	{Name: "CreateConnectionToken", Method: "POST", URI: "/token"},
	{Name: "DescribeCodeReview", Method: "GET", URI: "/codereviews/{CodeReviewArn}"},
	{Name: "DescribeRecommendationFeedback", Method: "GET", URI: "/feedback/{CodeReviewArn}"},
	{Name: "DescribeRepositoryAssociation", Method: "GET", URI: "/associations/{AssociationArn}"},
	{Name: "DisassociateRepository", Method: "DELETE", URI: "/associations/{AssociationArn}"},
	{Name: "GetMetricsData", Method: "GET", URI: "/metrics"},
	{Name: "ListCodeReviews", Method: "GET", URI: "/codereviews"},
	{Name: "ListRecommendationFeedback", Method: "GET", URI: "/feedback/{CodeReviewArn}/RecommendationFeedback"},
	{Name: "ListRecommendations", Method: "GET", URI: "/codereviews/{CodeReviewArn}/Recommendations"},
	{Name: "ListRepositoryAssociations", Method: "GET", URI: "/associations"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListThirdPartyRepositories", Method: "GET", URI: "/thirdPartyRepositories?ConnectionToken=ConnectionToken&NextToken=NextToken"},
	{Name: "PutRecommendationFeedback", Method: "PUT", URI: "/feedback"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
}

var codeGuruOperationByName = func() map[string]codeGuruOperation {
	out := make(map[string]codeGuruOperation, len(codeGuruOperations))
	for _, op := range codeGuruOperations {
		out[op.Name] = op
	}
	return out
}()
