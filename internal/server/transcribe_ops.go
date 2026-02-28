package server

type transcribeOperation struct {
	Name string
}

// Amazon Transcribe actions sourced from:
// https://docs.aws.amazon.com/transcribe/latest/APIReference/API_Operations.html
var transcribeOperations = []transcribeOperation{
	{Name: "CreateCallAnalyticsCategory"},
	{Name: "CreateLanguageModel"},
	{Name: "CreateMedicalVocabulary"},
	{Name: "CreateVocabulary"},
	{Name: "CreateVocabularyFilter"},
	{Name: "DeleteCallAnalyticsCategory"},
	{Name: "DeleteCallAnalyticsJob"},
	{Name: "DeleteLanguageModel"},
	{Name: "DeleteMedicalScribeJob"},
	{Name: "DeleteMedicalTranscriptionJob"},
	{Name: "DeleteMedicalVocabulary"},
	{Name: "DeleteTranscriptionJob"},
	{Name: "DeleteVocabulary"},
	{Name: "DeleteVocabularyFilter"},
	{Name: "DescribeLanguageModel"},
	{Name: "GetCallAnalyticsCategory"},
	{Name: "GetCallAnalyticsJob"},
	{Name: "GetMedicalScribeJob"},
	{Name: "GetMedicalTranscriptionJob"},
	{Name: "GetMedicalVocabulary"},
	{Name: "GetTranscriptionJob"},
	{Name: "GetVocabulary"},
	{Name: "GetVocabularyFilter"},
	{Name: "ListCallAnalyticsCategories"},
	{Name: "ListCallAnalyticsJobs"},
	{Name: "ListLanguageModels"},
	{Name: "ListMedicalScribeJobs"},
	{Name: "ListMedicalTranscriptionJobs"},
	{Name: "ListMedicalVocabularies"},
	{Name: "ListTagsForResource"},
	{Name: "ListTranscriptionJobs"},
	{Name: "ListVocabularies"},
	{Name: "ListVocabularyFilters"},
	{Name: "StartCallAnalyticsJob"},
	{Name: "StartMedicalScribeJob"},
	{Name: "StartMedicalTranscriptionJob"},
	{Name: "StartTranscriptionJob"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateCallAnalyticsCategory"},
	{Name: "UpdateMedicalVocabulary"},
	{Name: "UpdateVocabulary"},
	{Name: "UpdateVocabularyFilter"},
}

var transcribeOperationByName = func() map[string]transcribeOperation {
	out := make(map[string]transcribeOperation, len(transcribeOperations))
	for _, op := range transcribeOperations {
		out[op.Name] = op
	}
	return out
}()
