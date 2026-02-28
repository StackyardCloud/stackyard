package server

type workspacesCoreOperation struct {
	Name string
}

// Amazon WorkSpaces Core API actions sourced from:
// https://docs.aws.amazon.com/workspaces/latest/api/workspaces_core.html
var workspacesCoreOperations = []workspacesCoreOperation{
	{Name: "CopyWorkspaceImage"},
	{Name: "CreateTags"},
	{Name: "CreateWorkspaceBundle"},
	{Name: "CreateWorkspaceImage"},
	{Name: "CreateWorkspaces"},
	{Name: "DeleteTags"},
	{Name: "DeleteWorkspaceBundle"},
	{Name: "DeleteWorkspaceImage"},
	{Name: "DeregisterWorkspaceDirectory"},
	{Name: "DescribeAccount"},
	{Name: "DescribeAccountModifications"},
	{Name: "DescribeTags"},
	{Name: "DescribeWorkspaceBundles"},
	{Name: "DescribeWorkspaceDirectories"},
	{Name: "DescribeWorkspaceImagePermissions"},
	{Name: "DescribeWorkspaceImages"},
	{Name: "DescribeWorkspaceSnapshots"},
	{Name: "DescribeWorkspaces"},
	{Name: "ImportWorkspaceImage"},
	{Name: "ListAvailableManagementCidrRanges"},
	{Name: "MigrateWorkspace"},
	{Name: "ModifyAccount"},
	{Name: "ModifyWorkspaceCreationProperties"},
	{Name: "ModifyWorkspaceProperties"},
	{Name: "ModifyWorkspaceState"},
	{Name: "RebootWorkspaces"},
	{Name: "RebuildWorkspaces"},
	{Name: "RegisterWorkspaceDirectory"},
	{Name: "RestoreWorkspace"},
	{Name: "StartWorkspaces"},
	{Name: "StopWorkspaces"},
	{Name: "TerminateWorkspaces"},
	{Name: "UpdateWorkspaceBundle"},
	{Name: "UpdateWorkspaceImagePermission"},
}

var workspacesCoreOperationByName = func() map[string]workspacesCoreOperation {
	out := make(map[string]workspacesCoreOperation, len(workspacesCoreOperations))
	for _, op := range workspacesCoreOperations {
		out[op.Name] = op
	}
	return out
}()
