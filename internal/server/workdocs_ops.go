package server

// wdocOperation is one WorkDocs REST operation and its success status.
type wdocOperation struct {
	Name    string
	Method  string
	Pattern string
	Status  int
}

// Amazon WorkDocs operations sourced from the AWS SDK for Go v2 operation
// files (the WorkDocs API reference has been retired), e.g.:
// https://github.com/aws/aws-sdk-go-v2/blob/main/service/workdocs/api_op_CreateFolder.go
var workDocsOperations = []wdocOperation{
	{Name: "AbortDocumentVersionUpload", Method: "DELETE", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}", Status: 204},
	{Name: "ActivateUser", Method: "POST", Pattern: "/api/v1/users/{UserId}/activation", Status: 200},
	{Name: "AddResourcePermissions", Method: "POST", Pattern: "/api/v1/resources/{ResourceId}/permissions", Status: 201},
	{Name: "CreateComment", Method: "POST", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}/comment", Status: 201},
	{Name: "CreateCustomMetadata", Method: "PUT", Pattern: "/api/v1/resources/{ResourceId}/customMetadata", Status: 200},
	{Name: "CreateFolder", Method: "POST", Pattern: "/api/v1/folders", Status: 201},
	{Name: "CreateLabels", Method: "PUT", Pattern: "/api/v1/resources/{ResourceId}/labels", Status: 200},
	{Name: "CreateNotificationSubscription", Method: "POST", Pattern: "/api/v1/organizations/{OrganizationId}/subscriptions", Status: 200},
	{Name: "CreateUser", Method: "POST", Pattern: "/api/v1/users", Status: 201},
	{Name: "DeactivateUser", Method: "DELETE", Pattern: "/api/v1/users/{UserId}/activation", Status: 204},
	{Name: "DeleteComment", Method: "DELETE", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}/comment/{CommentId}", Status: 204},
	{Name: "DeleteCustomMetadata", Method: "DELETE", Pattern: "/api/v1/resources/{ResourceId}/customMetadata", Status: 200},
	{Name: "DeleteDocument", Method: "DELETE", Pattern: "/api/v1/documents/{DocumentId}", Status: 204},
	{Name: "DeleteDocumentVersion", Method: "DELETE", Pattern: "/api/v1/documentVersions/{DocumentId}/versions/{VersionId}", Status: 204},
	{Name: "DeleteFolder", Method: "DELETE", Pattern: "/api/v1/folders/{FolderId}", Status: 204},
	{Name: "DeleteFolderContents", Method: "DELETE", Pattern: "/api/v1/folders/{FolderId}/contents", Status: 204},
	{Name: "DeleteLabels", Method: "DELETE", Pattern: "/api/v1/resources/{ResourceId}/labels", Status: 200},
	{Name: "DeleteNotificationSubscription", Method: "DELETE", Pattern: "/api/v1/organizations/{OrganizationId}/subscriptions/{SubscriptionId}", Status: 200},
	{Name: "DeleteUser", Method: "DELETE", Pattern: "/api/v1/users/{UserId}", Status: 204},
	{Name: "DescribeActivities", Method: "GET", Pattern: "/api/v1/activities", Status: 200},
	{Name: "DescribeComments", Method: "GET", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}/comments", Status: 200},
	{Name: "DescribeDocumentVersions", Method: "GET", Pattern: "/api/v1/documents/{DocumentId}/versions", Status: 200},
	{Name: "DescribeFolderContents", Method: "GET", Pattern: "/api/v1/folders/{FolderId}/contents", Status: 200},
	{Name: "DescribeGroups", Method: "GET", Pattern: "/api/v1/groups", Status: 200},
	{Name: "DescribeNotificationSubscriptions", Method: "GET", Pattern: "/api/v1/organizations/{OrganizationId}/subscriptions", Status: 200},
	{Name: "DescribeResourcePermissions", Method: "GET", Pattern: "/api/v1/resources/{ResourceId}/permissions", Status: 200},
	{Name: "DescribeRootFolders", Method: "GET", Pattern: "/api/v1/me/root", Status: 200},
	{Name: "DescribeUsers", Method: "GET", Pattern: "/api/v1/users", Status: 200},
	{Name: "GetCurrentUser", Method: "GET", Pattern: "/api/v1/me", Status: 200},
	{Name: "GetDocument", Method: "GET", Pattern: "/api/v1/documents/{DocumentId}", Status: 200},
	{Name: "GetDocumentPath", Method: "GET", Pattern: "/api/v1/documents/{DocumentId}/path", Status: 200},
	{Name: "GetDocumentVersion", Method: "GET", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}", Status: 200},
	{Name: "GetFolder", Method: "GET", Pattern: "/api/v1/folders/{FolderId}", Status: 200},
	{Name: "GetFolderPath", Method: "GET", Pattern: "/api/v1/folders/{FolderId}/path", Status: 200},
	{Name: "GetResources", Method: "GET", Pattern: "/api/v1/resources", Status: 200},
	{Name: "InitiateDocumentVersionUpload", Method: "POST", Pattern: "/api/v1/documents", Status: 201},
	{Name: "RemoveAllResourcePermissions", Method: "DELETE", Pattern: "/api/v1/resources/{ResourceId}/permissions", Status: 204},
	{Name: "RemoveResourcePermission", Method: "DELETE", Pattern: "/api/v1/resources/{ResourceId}/permissions/{PrincipalId}", Status: 204},
	{Name: "RestoreDocumentVersions", Method: "POST", Pattern: "/api/v1/documentVersions/restore/{DocumentId}", Status: 204},
	{Name: "SearchResources", Method: "POST", Pattern: "/api/v1/search", Status: 200},
	{Name: "UpdateDocument", Method: "PATCH", Pattern: "/api/v1/documents/{DocumentId}", Status: 200},
	{Name: "UpdateDocumentVersion", Method: "PATCH", Pattern: "/api/v1/documents/{DocumentId}/versions/{VersionId}", Status: 200},
	{Name: "UpdateFolder", Method: "PATCH", Pattern: "/api/v1/folders/{FolderId}", Status: 200},
	{Name: "UpdateUser", Method: "PATCH", Pattern: "/api/v1/users/{UserId}", Status: 200},
}
