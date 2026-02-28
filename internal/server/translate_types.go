package server

type translateDataType struct {
	Name string
}

// Amazon Translate data types sourced from:
// https://docs.aws.amazon.com/translate/latest/APIReference/API_Types.html
var translateDataTypes = []translateDataType{
	{Name: "AppliedTerminology"},
	{Name: "Document"},
	{Name: "EncryptionKey"},
	{Name: "InputDataConfig"},
	{Name: "JobDetails"},
	{Name: "Language"},
	{Name: "OutputDataConfig"},
	{Name: "ParallelDataConfig"},
	{Name: "ParallelDataDataLocation"},
	{Name: "ParallelDataProperties"},
	{Name: "Tag"},
	{Name: "Term"},
	{Name: "TerminologyData"},
	{Name: "TerminologyDataLocation"},
	{Name: "TerminologyProperties"},
	{Name: "TextTranslationJobFilter"},
	{Name: "TextTranslationJobProperties"},
	{Name: "TranslatedDocument"},
	{Name: "TranslationSettings"},
}

var translateDataTypeByName = func() map[string]translateDataType {
	out := make(map[string]translateDataType, len(translateDataTypes))
	for _, dt := range translateDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
