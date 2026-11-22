package server

type glacierOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon S3 Glacier operations sourced from:
// https://docs.aws.amazon.com/amazonglacier/latest/dev/amazon-glacier-api.html
var glacierOperations = []glacierOperation{
	{Name: "AbortMultipartUpload", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads/{uploadId}"},
	{Name: "AbortVaultLock", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}/lock-policy"},
	{Name: "AddTagsToVault", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/tags?operation=add"},
	{Name: "CompleteMultipartUpload", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads/{uploadId}"},
	{Name: "CompleteVaultLock", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/lock-policy/{lockId}"},
	{Name: "CreateVault", Method: "PUT", URI: "/{accountId}/vaults/{vaultName}"},
	{Name: "DeleteArchive", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}/archives/{archiveId}"},
	{Name: "DeleteVault", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}"},
	{Name: "DeleteVaultAccessPolicy", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}/access-policy"},
	{Name: "DeleteVaultNotifications", Method: "DELETE", URI: "/{accountId}/vaults/{vaultName}/notification-configuration"},
	{Name: "DescribeJob", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/jobs/{jobId}"},
	{Name: "DescribeVault", Method: "GET", URI: "/{accountId}/vaults/{vaultName}"},
	{Name: "GetDataRetrievalPolicy", Method: "GET", URI: "/{accountId}/policies/data-retrieval"},
	{Name: "GetJobOutput", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/jobs/{jobId}/output"},
	{Name: "GetVaultAccessPolicy", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/access-policy"},
	{Name: "GetVaultLock", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/lock-policy"},
	{Name: "GetVaultNotifications", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/notification-configuration"},
	{Name: "InitiateJob", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/jobs"},
	{Name: "InitiateMultipartUpload", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads"},
	{Name: "InitiateVaultLock", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/lock-policy"},
	{Name: "ListJobs", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/jobs"},
	{Name: "ListMultipartUploads", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads"},
	{Name: "ListParts", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads/{uploadId}"},
	{Name: "ListProvisionedCapacity", Method: "GET", URI: "/{accountId}/provisioned-capacity"},
	{Name: "ListTagsForVault", Method: "GET", URI: "/{accountId}/vaults/{vaultName}/tags"},
	{Name: "ListVaults", Method: "GET", URI: "/{accountId}/vaults"},
	{Name: "PurchaseProvisionedCapacity", Method: "POST", URI: "/{accountId}/provisioned-capacity"},
	{Name: "RemoveTagsFromVault", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/tags?operation=remove"},
	{Name: "SetDataRetrievalPolicy", Method: "PUT", URI: "/{accountId}/policies/data-retrieval"},
	{Name: "SetVaultAccessPolicy", Method: "PUT", URI: "/{accountId}/vaults/{vaultName}/access-policy"},
	{Name: "SetVaultNotifications", Method: "PUT", URI: "/{accountId}/vaults/{vaultName}/notification-configuration"},
	{Name: "UploadArchive", Method: "POST", URI: "/{accountId}/vaults/{vaultName}/archives"},
	{Name: "UploadMultipartPart", Method: "PUT", URI: "/{accountId}/vaults/{vaultName}/multipart-uploads/{uploadId}"},
}

var glacierOperationByName = func() map[string]glacierOperation {
	out := make(map[string]glacierOperation, len(glacierOperations))
	for _, op := range glacierOperations {
		out[op.Name] = op
	}
	return out
}()
