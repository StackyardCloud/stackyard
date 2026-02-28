package server

type transcribeDataType struct {
	Name string
}

// Amazon Transcribe data types sourced from:
// https://docs.aws.amazon.com/transcribe/latest/APIReference/API_Types.html
var transcribeDataTypes = []transcribeDataType{
	{Name: "AbsoluteTimeRange"},
	{Name: "CallAnalyticsJob"},
	{Name: "CallAnalyticsJobDetails"},
	{Name: "CallAnalyticsJobSettings"},
	{Name: "CallAnalyticsJobSummary"},
	{Name: "CallAnalyticsSkippedFeature"},
	{Name: "CategoryProperties"},
	{Name: "ChannelDefinition"},
	{Name: "ClinicalNoteGenerationSettings"},
	{Name: "ContentRedaction"},
	{Name: "InputDataConfig"},
	{Name: "InterruptionFilter"},
	{Name: "JobExecutionSettings"},
	{Name: "LanguageCodeItem"},
	{Name: "LanguageIdSettings"},
	{Name: "LanguageModel"},
	{Name: "Media"},
	{Name: "MedicalScribeChannelDefinition"},
	{Name: "MedicalScribeContext"},
	{Name: "MedicalScribeJob"},
	{Name: "MedicalScribeJobSummary"},
	{Name: "MedicalScribeOutput"},
	{Name: "MedicalScribePatientContext"},
	{Name: "MedicalScribeSettings"},
	{Name: "MedicalTranscript"},
	{Name: "MedicalTranscriptionJob"},
	{Name: "MedicalTranscriptionJobSummary"},
	{Name: "MedicalTranscriptionSetting"},
	{Name: "ModelSettings"},
	{Name: "NonTalkTimeFilter"},
	{Name: "RelativeTimeRange"},
	{Name: "Rule"},
	{Name: "SentimentFilter"},
	{Name: "Settings"},
	{Name: "Subtitles"},
	{Name: "SubtitlesOutput"},
	{Name: "Summarization"},
	{Name: "Tag"},
	{Name: "ToxicityDetectionSettings"},
	{Name: "Transcript"},
	{Name: "TranscriptFilter"},
	{Name: "TranscriptionJob"},
	{Name: "TranscriptionJobSummary"},
	{Name: "VocabularyFilterInfo"},
	{Name: "VocabularyInfo"},
}

var transcribeDataTypeByName = func() map[string]transcribeDataType {
	out := make(map[string]transcribeDataType, len(transcribeDataTypes))
	for _, dt := range transcribeDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
