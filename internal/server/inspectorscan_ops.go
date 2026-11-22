package server

type inspectorScanOperation struct{ Name string }

// Amazon Inspector Scan operations sourced from:
// https://docs.aws.amazon.com/inspector/v2/APIReference/API_Operations_Inspector_Scan.html
var inspectorScanOperations = []inspectorScanOperation{
	{Name: "ScanSbom"},
}
