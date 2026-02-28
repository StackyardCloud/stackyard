package server

type rekognitionOperation struct {
	Name string
}

// Amazon Rekognition actions sourced from:
// https://docs.aws.amazon.com/rekognition/latest/APIReference/API_Operations.html
var rekognitionOperations = []rekognitionOperation{
	{Name: "AssociateFaces"},
	{Name: "CompareFaces"},
	{Name: "CopyProjectVersion"},
	{Name: "CreateCollection"},
	{Name: "CreateDataset"},
	{Name: "CreateFaceLivenessSession"},
	{Name: "CreateProject"},
	{Name: "CreateProjectVersion"},
	{Name: "CreateStreamProcessor"},
	{Name: "CreateUser"},
	{Name: "DeleteCollection"},
	{Name: "DeleteDataset"},
	{Name: "DeleteFaces"},
	{Name: "DeleteProject"},
	{Name: "DeleteProjectPolicy"},
	{Name: "DeleteProjectVersion"},
	{Name: "DeleteStreamProcessor"},
	{Name: "DeleteUser"},
	{Name: "DescribeCollection"},
	{Name: "DescribeDataset"},
	{Name: "DescribeProjectVersions"},
	{Name: "DescribeProjects"},
	{Name: "DescribeStreamProcessor"},
	{Name: "DetectCustomLabels"},
	{Name: "DetectFaces"},
	{Name: "DetectLabels"},
	{Name: "DetectModerationLabels"},
	{Name: "DetectProtectiveEquipment"},
	{Name: "DetectText"},
	{Name: "DisassociateFaces"},
	{Name: "DistributeDatasetEntries"},
	{Name: "GetCelebrityInfo"},
	{Name: "GetCelebrityRecognition"},
	{Name: "GetContentModeration"},
	{Name: "GetFaceDetection"},
	{Name: "GetFaceLivenessSessionResults"},
	{Name: "GetFaceSearch"},
	{Name: "GetLabelDetection"},
	{Name: "GetMediaAnalysisJob"},
	{Name: "GetPersonTracking"},
	{Name: "GetSegmentDetection"},
	{Name: "GetTextDetection"},
	{Name: "IndexFaces"},
	{Name: "ListCollections"},
	{Name: "ListDatasetEntries"},
	{Name: "ListDatasetLabels"},
	{Name: "ListFaces"},
	{Name: "ListMediaAnalysisJobs"},
	{Name: "ListProjectPolicies"},
	{Name: "ListStreamProcessors"},
	{Name: "ListTagsForResource"},
	{Name: "ListUsers"},
	{Name: "PutProjectPolicy"},
	{Name: "RecognizeCelebrities"},
	{Name: "SearchFaces"},
	{Name: "SearchFacesByImage"},
	{Name: "SearchUsers"},
	{Name: "SearchUsersByImage"},
	{Name: "StartCelebrityRecognition"},
	{Name: "StartContentModeration"},
	{Name: "StartFaceDetection"},
	{Name: "StartFaceSearch"},
	{Name: "StartLabelDetection"},
	{Name: "StartMediaAnalysisJob"},
	{Name: "StartPersonTracking"},
	{Name: "StartProjectVersion"},
	{Name: "StartSegmentDetection"},
	{Name: "StartStreamProcessor"},
	{Name: "StartTextDetection"},
	{Name: "StopProjectVersion"},
	{Name: "StopStreamProcessor"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDatasetEntries"},
	{Name: "UpdateStreamProcessor"},
}

var rekognitionOperationByName = func() map[string]rekognitionOperation {
	out := make(map[string]rekognitionOperation, len(rekognitionOperations))
	for _, op := range rekognitionOperations {
		out[op.Name] = op
	}
	return out
}()
