package server

type cloudWatchSyntheticsDataType struct {
	Name string
}

// Amazon CloudWatch Synthetics data types sourced from:
// https://docs.aws.amazon.com/AmazonSynthetics/latest/APIReference/API_Types.html
var cloudWatchSyntheticsDataTypes = []cloudWatchSyntheticsDataType{
	{Name: "ArtifactConfigInput"},
	{Name: "ArtifactConfigOutput"},
	{Name: "BaseScreenshot"},
	{Name: "BrowserConfig"},
	{Name: "Canary"},
	{Name: "CanaryCodeInput"},
	{Name: "CanaryCodeOutput"},
	{Name: "CanaryDryRunConfigOutput"},
	{Name: "CanaryLastRun"},
	{Name: "CanaryRun"},
	{Name: "CanaryRunConfigInput"},
	{Name: "CanaryRunConfigOutput"},
	{Name: "CanaryRunStatus"},
	{Name: "CanaryRunTimeline"},
	{Name: "CanaryScheduleInput"},
	{Name: "CanaryScheduleOutput"},
	{Name: "CanaryStatus"},
	{Name: "CanaryTimeline"},
	{Name: "Dependency"},
	{Name: "DryRunConfigOutput"},
	{Name: "EngineConfig"},
	{Name: "Group"},
	{Name: "GroupSummary"},
	{Name: "RetryConfigInput"},
	{Name: "RetryConfigOutput"},
	{Name: "RuntimeVersion"},
	{Name: "S3EncryptionConfig"},
	{Name: "UpdateCanary"},
	{Name: "VisualReferenceInput"},
	{Name: "VisualReferenceOutput"},
	{Name: "VpcConfigInput"},
	{Name: "VpcConfigOutput"},
}

var cloudWatchSyntheticsDataTypeByName = func() map[string]cloudWatchSyntheticsDataType {
	out := make(map[string]cloudWatchSyntheticsDataType, len(cloudWatchSyntheticsDataTypes))
	for _, dt := range cloudWatchSyntheticsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
