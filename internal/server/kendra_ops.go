package server

type kendraOperation struct {
	Name string
}

// Amazon Kendra actions sourced from:
// https://docs.aws.amazon.com/kendra/latest/APIReference/API_Operations.html
var kendraOperations = []kendraOperation{
	{Name: "AssociateEntitiesToExperience"},
	{Name: "AssociatePersonasToEntities"},
	{Name: "BatchDeleteDocument"},
	{Name: "BatchDeleteFeaturedResultsSet"},
	{Name: "BatchGetDocumentStatus"},
	{Name: "BatchPutDocument"},
	{Name: "ClearQuerySuggestions"},
	{Name: "CreateAccessControlConfiguration"},
	{Name: "CreateDataSource"},
	{Name: "CreateExperience"},
	{Name: "CreateFaq"},
	{Name: "CreateFeaturedResultsSet"},
	{Name: "CreateIndex"},
	{Name: "CreateQuerySuggestionsBlockList"},
	{Name: "CreateThesaurus"},
	{Name: "DeleteAccessControlConfiguration"},
	{Name: "DeleteDataSource"},
	{Name: "DeleteExperience"},
	{Name: "DeleteFaq"},
	{Name: "DeleteIndex"},
	{Name: "DeletePrincipalMapping"},
	{Name: "DeleteQuerySuggestionsBlockList"},
	{Name: "DeleteThesaurus"},
	{Name: "DescribeAccessControlConfiguration"},
	{Name: "DescribeDataSource"},
	{Name: "DescribeExperience"},
	{Name: "DescribeFaq"},
	{Name: "DescribeFeaturedResultsSet"},
	{Name: "DescribeIndex"},
	{Name: "DescribePrincipalMapping"},
	{Name: "DescribeQuerySuggestionsBlockList"},
	{Name: "DescribeQuerySuggestionsConfig"},
	{Name: "DescribeThesaurus"},
	{Name: "DisassociateEntitiesFromExperience"},
	{Name: "DisassociatePersonasFromEntities"},
	{Name: "GetQuerySuggestions"},
	{Name: "GetSnapshots"},
	{Name: "ListAccessControlConfigurations"},
	{Name: "ListDataSources"},
	{Name: "ListDataSourceSyncJobs"},
	{Name: "ListEntityPersonas"},
	{Name: "ListExperienceEntities"},
	{Name: "ListExperiences"},
	{Name: "ListFaqs"},
	{Name: "ListFeaturedResultsSets"},
	{Name: "ListGroupsOlderThanOrderingId"},
	{Name: "ListIndices"},
	{Name: "ListQuerySuggestionsBlockLists"},
	{Name: "ListTagsForResource"},
	{Name: "ListThesauri"},
	{Name: "PutPrincipalMapping"},
	{Name: "Query"},
	{Name: "Retrieve"},
	{Name: "StartDataSourceSyncJob"},
	{Name: "StopDataSourceSyncJob"},
	{Name: "SubmitFeedback"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAccessControlConfiguration"},
	{Name: "UpdateDataSource"},
	{Name: "UpdateExperience"},
	{Name: "UpdateFeaturedResultsSet"},
	{Name: "UpdateIndex"},
	{Name: "UpdateQuerySuggestionsBlockList"},
	{Name: "UpdateQuerySuggestionsConfig"},
	{Name: "UpdateThesaurus"},
}

var kendraOperationByName = func() map[string]kendraOperation {
	out := make(map[string]kendraOperation, len(kendraOperations))
	for _, op := range kendraOperations {
		out[op.Name] = op
	}
	return out
}()
