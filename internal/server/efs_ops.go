package server

type efsOperation struct{ Name string }

// Amazon EFS operations (API version 2015-02-01) sourced from:
// https://docs.aws.amazon.com/efs/latest/ug/API_Operations.html
var efsOperations = []efsOperation{
	{Name: "CreateAccessPoint"},
	{Name: "CreateFileSystem"},
	{Name: "CreateMountTarget"},
	{Name: "CreateReplicationConfiguration"},
	{Name: "CreateTags"},
	{Name: "DeleteAccessPoint"},
	{Name: "DeleteFileSystem"},
	{Name: "DeleteFileSystemPolicy"},
	{Name: "DeleteMountTarget"},
	{Name: "DeleteReplicationConfiguration"},
	{Name: "DeleteTags"},
	{Name: "DescribeAccessPoints"},
	{Name: "DescribeAccountPreferences"},
	{Name: "DescribeBackupPolicy"},
	{Name: "DescribeFileSystemPolicy"},
	{Name: "DescribeFileSystems"},
	{Name: "DescribeLifecycleConfiguration"},
	{Name: "DescribeMountTargetSecurityGroups"},
	{Name: "DescribeMountTargets"},
	{Name: "DescribeReplicationConfigurations"},
	{Name: "DescribeTags"},
	{Name: "ListTagsForResource"},
	{Name: "ModifyMountTargetSecurityGroups"},
	{Name: "PutAccountPreferences"},
	{Name: "PutBackupPolicy"},
	{Name: "PutFileSystemPolicy"},
	{Name: "PutLifecycleConfiguration"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateFileSystem"},
	{Name: "UpdateFileSystemProtection"},
}

var efsOperationByName = func() map[string]efsOperation {
	out := make(map[string]efsOperation, len(efsOperations))
	for _, op := range efsOperations {
		out[op.Name] = op
	}
	return out
}()
