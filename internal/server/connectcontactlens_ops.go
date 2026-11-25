package server

type connectContactLensOperation struct{ Name string }

// Amazon Connect Contact Lens operations sourced from:
// https://docs.aws.amazon.com/connect/latest/APIReference/API_Operations_Amazon_Connect_Contact_Lens.html
var connectContactLensOperations = []connectContactLensOperation{
	{Name: "ListRealtimeContactAnalysisSegments"},
}
