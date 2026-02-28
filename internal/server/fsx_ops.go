package server

type fsxOperation struct {
	Name string
}

// Amazon FSx operations sourced from:
// https://docs.aws.amazon.com/fsx/latest/APIReference/API_Operations.html
var fsxOperations = []fsxOperation{
	{Name: "AssociateFileSystemAliases"},
	{Name: "CancelDataRepositoryTask"},
	{Name: "CopyBackup"},
	{Name: "CopySnapshotAndUpdateVolume"},
	{Name: "CreateAndAttachS3AccessPoint"},
	{Name: "CreateBackup"},
	{Name: "CreateDataRepositoryAssociation"},
	{Name: "CreateDataRepositoryTask"},
	{Name: "CreateFileCache"},
	{Name: "CreateFileSystem"},
	{Name: "CreateFileSystemFromBackup"},
	{Name: "CreateSnapshot"},
	{Name: "CreateStorageVirtualMachine"},
	{Name: "CreateVolume"},
	{Name: "CreateVolumeFromBackup"},
	{Name: "DeleteBackup"},
	{Name: "DeleteDataRepositoryAssociation"},
	{Name: "DeleteFileCache"},
	{Name: "DeleteFileSystem"},
	{Name: "DeleteSnapshot"},
	{Name: "DeleteStorageVirtualMachine"},
	{Name: "DeleteVolume"},
	{Name: "DescribeBackups"},
	{Name: "DescribeDataRepositoryAssociations"},
	{Name: "DescribeDataRepositoryTasks"},
	{Name: "DescribeFileCaches"},
	{Name: "DescribeFileSystemAliases"},
	{Name: "DescribeFileSystems"},
	{Name: "DescribeS3AccessPointAttachments"},
	{Name: "DescribeSharedVpcConfiguration"},
	{Name: "DescribeSnapshots"},
	{Name: "DescribeStorageVirtualMachines"},
	{Name: "DescribeVolumes"},
	{Name: "DetachAndDeleteS3AccessPoint"},
	{Name: "DisassociateFileSystemAliases"},
	{Name: "ListTagsForResource"},
	{Name: "ReleaseFileSystemNfsV3Locks"},
	{Name: "RestoreVolumeFromSnapshot"},
	{Name: "StartMisconfiguredStateRecovery"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDataRepositoryAssociation"},
	{Name: "UpdateFileCache"},
	{Name: "UpdateFileSystem"},
	{Name: "UpdateSharedVpcConfiguration"},
	{Name: "UpdateSnapshot"},
	{Name: "UpdateStorageVirtualMachine"},
	{Name: "UpdateVolume"},
}

var fsxOperationByName = func() map[string]fsxOperation {
	out := make(map[string]fsxOperation, len(fsxOperations))
	for _, op := range fsxOperations {
		out[op.Name] = op
	}
	return out
}()
