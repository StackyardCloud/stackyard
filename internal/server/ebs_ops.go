package server

type ebsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elastic Block Storage (EBS) actions sourced from:
// https://docs.aws.amazon.com/ebs/latest/APIReference/API_Operations.html
var ebsOperations = []ebsOperation{
	{Name: "CompleteSnapshot", Method: "POST", URI: "/snapshots/completion/{snapshotId}"},
	{Name: "GetSnapshotBlock", Method: "GET", URI: "/snapshots/{snapshotId}/blocks/{blockIndex}"},
	{Name: "ListChangedBlocks", Method: "GET", URI: "/snapshots/{secondSnapshotId}/changedblocks"},
	{Name: "ListSnapshotBlocks", Method: "GET", URI: "/snapshots/{snapshotId}/blocks"},
	{Name: "PutSnapshotBlock", Method: "PUT", URI: "/snapshots/{snapshotId}/blocks/{blockIndex}"},
	{Name: "StartSnapshot", Method: "POST", URI: "/snapshots"},
}

var ebsOperationByName = func() map[string]ebsOperation {
	out := make(map[string]ebsOperation, len(ebsOperations))
	for _, op := range ebsOperations {
		out[op.Name] = op
	}
	return out
}()
