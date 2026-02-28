package server

type ebsDataType struct {
	Name string
}

// AWS Elastic Block Storage (EBS) data types sourced from:
// https://docs.aws.amazon.com/ebs/latest/APIReference/API_Types.html
var ebsDataTypes = []ebsDataType{
	{Name: "AccessDeniedException"},
	{Name: "Block"},
	{Name: "ChangedBlock"},
	{Name: "CompleteSnapshotRequest"},
	{Name: "CompleteSnapshotResponse"},
	{Name: "ConcurrentLimitExceededException"},
	{Name: "ConflictException"},
	{Name: "GetSnapshotBlockRequest"},
	{Name: "GetSnapshotBlockResponse"},
	{Name: "InternalServerException"},
	{Name: "ListChangedBlocksRequest"},
	{Name: "ListChangedBlocksResponse"},
	{Name: "ListSnapshotBlocksRequest"},
	{Name: "ListSnapshotBlocksResponse"},
	{Name: "PutSnapshotBlockRequest"},
	{Name: "PutSnapshotBlockResponse"},
	{Name: "RequestThrottledException"},
	{Name: "ResourceNotFoundException"},
	{Name: "ServiceQuotaExceededException"},
	{Name: "StartSnapshotRequest"},
	{Name: "StartSnapshotResponse"},
	{Name: "Tag"},
	{Name: "ValidationException"},
}

var ebsDataTypeByName = func() map[string]ebsDataType {
	out := make(map[string]ebsDataType, len(ebsDataTypes))
	for _, dt := range ebsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
