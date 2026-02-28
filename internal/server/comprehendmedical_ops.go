package server

type comprehendMedicalOperation struct {
	Name string
}

// Amazon Comprehend Medical operations sourced from:
// https://docs.aws.amazon.com/comprehend-medical/latest/api/API_Operations.html
var comprehendMedicalOperations = []comprehendMedicalOperation{
	{Name: "DescribeEntitiesDetectionV2Job"},
	{Name: "DescribeICD10CMInferenceJob"},
	{Name: "DescribePHIDetectionJob"},
	{Name: "DescribeRxNormInferenceJob"},
	{Name: "DescribeSNOMEDCTInferenceJob"},
	{Name: "DetectEntitiesV2"},
	{Name: "DetectPHI"},
	{Name: "InferICD10CM"},
	{Name: "InferRxNorm"},
	{Name: "InferSNOMEDCT"},
	{Name: "ListEntitiesDetectionV2Jobs"},
	{Name: "ListICD10CMInferenceJobs"},
	{Name: "ListPHIDetectionJobs"},
	{Name: "ListRxNormInferenceJobs"},
	{Name: "ListSNOMEDCTInferenceJobs"},
	{Name: "StartEntitiesDetectionV2Job"},
	{Name: "StartICD10CMInferenceJob"},
	{Name: "StartPHIDetectionJob"},
	{Name: "StartRxNormInferenceJob"},
	{Name: "StartSNOMEDCTInferenceJob"},
	{Name: "StopEntitiesDetectionV2Job"},
	{Name: "StopICD10CMInferenceJob"},
	{Name: "StopPHIDetectionJob"},
	{Name: "StopRxNormInferenceJob"},
	{Name: "StopSNOMEDCTInferenceJob"},
}

var comprehendMedicalOperationByName = func() map[string]comprehendMedicalOperation {
	out := make(map[string]comprehendMedicalOperation, len(comprehendMedicalOperations))
	for _, op := range comprehendMedicalOperations {
		out[op.Name] = op
	}
	return out
}()
