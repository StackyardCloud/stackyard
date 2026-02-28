package server

type translateOperation struct {
	Name string
}

// Amazon Translate actions sourced from:
// https://docs.aws.amazon.com/translate/latest/APIReference/API_Operations.html
var translateOperations = []translateOperation{
	{Name: "CreateParallelData"},
	{Name: "DeleteParallelData"},
	{Name: "DeleteTerminology"},
	{Name: "DescribeTextTranslationJob"},
	{Name: "GetParallelData"},
	{Name: "GetTerminology"},
	{Name: "ImportTerminology"},
	{Name: "ListLanguages"},
	{Name: "ListParallelData"},
	{Name: "ListTagsForResource"},
	{Name: "ListTerminologies"},
	{Name: "ListTextTranslationJobs"},
	{Name: "StartTextTranslationJob"},
	{Name: "StopTextTranslationJob"},
	{Name: "TagResource"},
	{Name: "TranslateDocument"},
	{Name: "TranslateText"},
	{Name: "UntagResource"},
	{Name: "UpdateParallelData"},
}

var translateOperationByName = func() map[string]translateOperation {
	out := make(map[string]translateOperation, len(translateOperations))
	for _, op := range translateOperations {
		out[op.Name] = op
	}
	return out
}()
