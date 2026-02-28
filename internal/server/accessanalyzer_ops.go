package server

type accessAnalyzerOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IAM Access Analyzer operations sourced from:
// https://docs.aws.amazon.com/access-analyzer/latest/APIReference/API_Operations.html
var accessAnalyzerOperations = []accessAnalyzerOperation{
	{Name: "ApplyArchiveRule", Method: "PUT", URI: "/archive-rule"},
	{Name: "CancelPolicyGeneration", Method: "PUT", URI: "/policy/generation/{jobId}"},
	{Name: "CheckAccessNotGranted", Method: "POST", URI: "/policy/check-access-not-granted"},
	{Name: "CheckNoNewAccess", Method: "POST", URI: "/policy/check-no-new-access"},
	{Name: "CheckNoPublicAccess", Method: "POST", URI: "/policy/check-no-public-access"},
	{Name: "CreateAccessPreview", Method: "PUT", URI: "/access-preview"},
	{Name: "CreateAnalyzer", Method: "PUT", URI: "/analyzer"},
	{Name: "CreateArchiveRule", Method: "PUT", URI: "/analyzer/{analyzerName}/archive-rule"},
	{Name: "DeleteAnalyzer", Method: "DELETE", URI: "/analyzer/{analyzerName}?clientToken=clientToken"},
	{Name: "DeleteArchiveRule", Method: "DELETE", URI: "/analyzer/{analyzerName}/archive-rule/{ruleName}?clientToken=clientToken"},
	{Name: "GenerateFindingRecommendation", Method: "POST", URI: "/recommendation/{id}?analyzerArn=analyzerArn"},
	{Name: "GetAccessPreview", Method: "GET", URI: "/access-preview/{accessPreviewId}?analyzerArn=analyzerArn"},
	{Name: "GetAnalyzedResource", Method: "GET", URI: "/analyzed-resource?analyzerArn=analyzerArn&resourceArn=resourceArn"},
	{Name: "GetAnalyzer", Method: "GET", URI: "/analyzer/{analyzerName}"},
	{Name: "GetArchiveRule", Method: "GET", URI: "/analyzer/{analyzerName}/archive-rule/{ruleName}"},
	{Name: "GetFinding", Method: "GET", URI: "/finding/{id}?analyzerArn=analyzerArn"},
	{Name: "GetFindingRecommendation", Method: "GET", URI: "/recommendation/{id}?analyzerArn=analyzerArn&maxResults=maxResults&nextToken=nextToken"},
	{Name: "GetFindingsStatistics", Method: "POST", URI: "/analyzer/findings/statistics"},
	{Name: "GetFindingV2", Method: "GET", URI: "/findingv2/{id}?analyzerArn=analyzerArn&maxResults=maxResults&nextToken=nextToken"},
	{Name: "GetGeneratedPolicy", Method: "GET", URI: "/policy/generation/{jobId}?includeResourcePlaceholders=includeResourcePlaceholders&includeServiceLevelTemplate=includeServiceLevelTemplate"},
	{Name: "ListAccessPreviewFindings", Method: "POST", URI: "/access-preview/{accessPreviewId}"},
	{Name: "ListAccessPreviews", Method: "GET", URI: "/access-preview?analyzerArn=analyzerArn&maxResults=maxResults&nextToken=nextToken"},
	{Name: "ListAnalyzedResources", Method: "POST", URI: "/analyzed-resource"},
	{Name: "ListAnalyzers", Method: "GET", URI: "/analyzer?maxResults=maxResults&nextToken=nextToken&type=type"},
	{Name: "ListArchiveRules", Method: "GET", URI: "/analyzer/{analyzerName}/archive-rule?maxResults=maxResults&nextToken=nextToken"},
	{Name: "ListFindings", Method: "POST", URI: "/finding"},
	{Name: "ListFindingsV2", Method: "POST", URI: "/findingv2"},
	{Name: "ListPolicyGenerations", Method: "GET", URI: "/policy/generation?maxResults=maxResults&nextToken=nextToken&principalArn=principalArn"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartPolicyGeneration", Method: "PUT", URI: "/policy/generation"},
	{Name: "StartResourceScan", Method: "POST", URI: "/resource/scan"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys=tagKeys"},
	{Name: "UpdateAnalyzer", Method: "PUT", URI: "/analyzer/{analyzerName}"},
	{Name: "UpdateArchiveRule", Method: "PUT", URI: "/analyzer/{analyzerName}/archive-rule/{ruleName}"},
	{Name: "UpdateFindings", Method: "PUT", URI: "/finding"},
	{Name: "ValidatePolicy", Method: "POST", URI: "/policy/validation?maxResults=maxResults&nextToken=nextToken"},
}

var accessAnalyzerOperationByName = func() map[string]accessAnalyzerOperation {
	out := make(map[string]accessAnalyzerOperation, len(accessAnalyzerOperations))
	for _, op := range accessAnalyzerOperations {
		out[op.Name] = op
	}
	return out
}()
