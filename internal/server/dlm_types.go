package server

type dlmDataType struct {
	Name string
}

// Amazon Data Lifecycle Manager data types sourced from:
// https://docs.aws.amazon.com/dlm/latest/APIReference/API_Types.html
var dlmDataTypes = []dlmDataType{
	{Name: "Action"},
	{Name: "ArchiveRetainRule"},
	{Name: "ArchiveRule"},
	{Name: "CreateRule"},
	{Name: "CrossRegionCopyAction"},
	{Name: "CrossRegionCopyDeprecateRule"},
	{Name: "CrossRegionCopyRetainRule"},
	{Name: "CrossRegionCopyRule"},
	{Name: "CrossRegionCopyTarget"},
	{Name: "DeprecateRule"},
	{Name: "EncryptionConfiguration"},
	{Name: "EventParameters"},
	{Name: "EventSource"},
	{Name: "Exclusions"},
	{Name: "FastRestoreRule"},
	{Name: "LifecyclePolicy"},
	{Name: "LifecyclePolicySummary"},
	{Name: "Parameters"},
	{Name: "PolicyDetails"},
	{Name: "RetainRule"},
	{Name: "RetentionArchiveTier"},
	{Name: "Schedule"},
	{Name: "Script"},
	{Name: "ShareRule"},
	{Name: "Tag"},
	{Name: "UpdateLifecyclePolicy"},
}

var dlmDataTypeByName = func() map[string]dlmDataType {
	out := make(map[string]dlmDataType, len(dlmDataTypes))
	for _, dt := range dlmDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
