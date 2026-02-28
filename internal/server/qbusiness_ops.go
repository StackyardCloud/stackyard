package server

type qBusinessOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Q Business actions sourced from:
// https://docs.aws.amazon.com/amazonq/latest/api-reference/API_Operations.html
var qBusinessOperations = []qBusinessOperation{
	{Name: "AssociatePermission", Method: "POST", URI: "/applications/{applicationId}/policy"},
	{Name: "BatchDeleteDocument", Method: "POST", URI: "/applications/{applicationId}/indices/{indexId}/documents/delete"},
	{Name: "BatchPutDocument", Method: "POST", URI: "/applications/{applicationId}/indices/{indexId}/documents"},
	{Name: "CancelSubscription", Method: "DELETE", URI: "/applications/{applicationId}/subscriptions/{subscriptionId}"},
	{Name: "Chat", Method: "POST", URI: "/applications/{applicationId}/conversations?clientToken={clientToken}&conversationId={conversationId}&parentMessageId={parentMessageId}&userGroups={userGroups}&userId={userId}"},
	{Name: "ChatSync", Method: "POST", URI: "/applications/{applicationId}/conversations?sync&userGroups={userGroups}&userId={userId}"},
	{Name: "CheckDocumentAccess", Method: "GET", URI: "/applications/{applicationId}/index/{indexId}/users/{userId}/documents/{documentId}/check-document-access?dataSourceId={dataSourceId}"},
	{Name: "CreateAnonymousWebExperienceUrl", Method: "POST", URI: "/applications/{applicationId}/experiences/{webExperienceId}/anonymous-url"},
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateChatResponseConfiguration", Method: "POST", URI: "/applications/{applicationId}/chatresponseconfigurations"},
	{Name: "CreateDataAccessor", Method: "POST", URI: "/applications/{applicationId}/dataaccessors"},
	{Name: "CreateDataSource", Method: "POST", URI: "/applications/{applicationId}/indices/{indexId}/datasources"},
	{Name: "CreateIndex", Method: "POST", URI: "/applications/{applicationId}/indices"},
	{Name: "CreatePlugin", Method: "POST", URI: "/applications/{applicationId}/plugins"},
	{Name: "CreateRetriever", Method: "POST", URI: "/applications/{applicationId}/retrievers"},
	{Name: "CreateSubscription", Method: "POST", URI: "/applications/{applicationId}/subscriptions"},
	{Name: "CreateUser", Method: "POST", URI: "/applications/{applicationId}/users"},
	{Name: "CreateWebExperience", Method: "POST", URI: "/applications/{applicationId}/experiences"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{applicationId}"},
	{Name: "DeleteAttachment", Method: "DELETE", URI: "/applications/{applicationId}/conversations/{conversationId}/attachments/{attachmentId}?userId={userId}"},
	{Name: "DeleteChatControlsConfiguration", Method: "DELETE", URI: "/applications/{applicationId}/chatcontrols"},
	{Name: "DeleteChatResponseConfiguration", Method: "DELETE", URI: "/applications/{applicationId}/chatresponseconfigurations/{chatResponseConfigurationId}"},
	{Name: "DeleteConversation", Method: "DELETE", URI: "/applications/{applicationId}/conversations/{conversationId}?userId={userId}"},
	{Name: "DeleteDataAccessor", Method: "DELETE", URI: "/applications/{applicationId}/dataaccessors/{dataAccessorId}"},
	{Name: "DeleteDataSource", Method: "DELETE", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}"},
	{Name: "DeleteGroup", Method: "DELETE", URI: "/applications/{applicationId}/indices/{indexId}/groups/{groupName}?dataSourceId={dataSourceId}"},
	{Name: "DeleteIndex", Method: "DELETE", URI: "/applications/{applicationId}/indices/{indexId}"},
	{Name: "DeletePlugin", Method: "DELETE", URI: "/applications/{applicationId}/plugins/{pluginId}"},
	{Name: "DeleteRetriever", Method: "DELETE", URI: "/applications/{applicationId}/retrievers/{retrieverId}"},
	{Name: "DeleteUser", Method: "DELETE", URI: "/applications/{applicationId}/users/{userId}"},
	{Name: "DeleteWebExperience", Method: "DELETE", URI: "/applications/{applicationId}/experiences/{webExperienceId}"},
	{Name: "DisassociatePermission", Method: "DELETE", URI: "/applications/{applicationId}/policy/{statementId}"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{applicationId}"},
	{Name: "GetChatControlsConfiguration", Method: "GET", URI: "/applications/{applicationId}/chatcontrols?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "GetChatResponseConfiguration", Method: "GET", URI: "/applications/{applicationId}/chatresponseconfigurations/{chatResponseConfigurationId}"},
	{Name: "GetDataAccessor", Method: "GET", URI: "/applications/{applicationId}/dataaccessors/{dataAccessorId}"},
	{Name: "GetDataSource", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}"},
	{Name: "GetDocumentContent", Method: "GET", URI: "/applications/{applicationId}/index/{indexId}/documents/{documentId}/content?dataSourceId={dataSourceId}&outputFormat={outputFormat}"},
	{Name: "GetGroup", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}/groups/{groupName}?dataSourceId={dataSourceId}"},
	{Name: "GetIndex", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}"},
	{Name: "GetMedia", Method: "GET", URI: "/applications/{applicationId}/conversations/{conversationId}/messages/{messageId}/media/{mediaId}"},
	{Name: "GetPlugin", Method: "GET", URI: "/applications/{applicationId}/plugins/{pluginId}"},
	{Name: "GetPolicy", Method: "GET", URI: "/applications/{applicationId}/policy"},
	{Name: "GetRetriever", Method: "GET", URI: "/applications/{applicationId}/retrievers/{retrieverId}"},
	{Name: "GetUser", Method: "GET", URI: "/applications/{applicationId}/users/{userId}"},
	{Name: "GetWebExperience", Method: "GET", URI: "/applications/{applicationId}/experiences/{webExperienceId}"},
	{Name: "ListApplications", Method: "GET", URI: "/applications?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListAttachments", Method: "GET", URI: "/applications/{applicationId}/attachments?conversationId={conversationId}&maxResults={maxResults}&nextToken={nextToken}&userId={userId}"},
	{Name: "ListChatResponseConfigurations", Method: "GET", URI: "/applications/{applicationId}/chatresponseconfigurations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListConversations", Method: "GET", URI: "/applications/{applicationId}/conversations?maxResults={maxResults}&nextToken={nextToken}&userId={userId}"},
	{Name: "ListDataAccessors", Method: "GET", URI: "/applications/{applicationId}/dataaccessors?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataSources", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}/datasources?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataSourceSyncJobs", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}/syncjobs?endTime={endTime}&maxResults={maxResults}&nextToken={nextToken}&startTime={startTime}&syncStatus={statusFilter}"},
	{Name: "ListDocuments", Method: "GET", URI: "/applications/{applicationId}/index/{indexId}/documents?dataSourceIds={dataSourceIds}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListGroups", Method: "GET", URI: "/applications/{applicationId}/indices/{indexId}/groups?dataSourceId={dataSourceId}&maxResults={maxResults}&nextToken={nextToken}&updatedEarlierThan={updatedEarlierThan}"},
	{Name: "ListIndices", Method: "GET", URI: "/applications/{applicationId}/indices?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListMessages", Method: "GET", URI: "/applications/{applicationId}/conversations/{conversationId}?maxResults={maxResults}&nextToken={nextToken}&userId={userId}"},
	{Name: "ListPluginActions", Method: "GET", URI: "/applications/{applicationId}/plugins/{pluginId}/actions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListPlugins", Method: "GET", URI: "/applications/{applicationId}/plugins?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListPluginTypeActions", Method: "GET", URI: "/pluginTypes/{pluginType}/actions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListPluginTypeMetadata", Method: "GET", URI: "/pluginTypeMetadata?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListRetrievers", Method: "GET", URI: "/applications/{applicationId}/retrievers?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListSubscriptions", Method: "GET", URI: "/applications/{applicationId}/subscriptions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceARN}"},
	{Name: "ListWebExperiences", Method: "GET", URI: "/applications/{applicationId}/experiences?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "PutFeedback", Method: "POST", URI: "/applications/{applicationId}/conversations/{conversationId}/messages/{messageId}/feedback?userId={userId}"},
	{Name: "PutGroup", Method: "PUT", URI: "/applications/{applicationId}/indices/{indexId}/groups"},
	{Name: "SearchRelevantContent", Method: "POST", URI: "/applications/{applicationId}/relevant-content"},
	{Name: "StartDataSourceSyncJob", Method: "POST", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}/startsync"},
	{Name: "StopDataSourceSyncJob", Method: "POST", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}/stopsync"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceARN}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceARN}?tagKeys={tagKeys}"},
	{Name: "UpdateApplication", Method: "PUT", URI: "/applications/{applicationId}"},
	{Name: "UpdateChatControlsConfiguration", Method: "PATCH", URI: "/applications/{applicationId}/chatcontrols"},
	{Name: "UpdateChatResponseConfiguration", Method: "PUT", URI: "/applications/{applicationId}/chatresponseconfigurations/{chatResponseConfigurationId}"},
	{Name: "UpdateDataAccessor", Method: "PUT", URI: "/applications/{applicationId}/dataaccessors/{dataAccessorId}"},
	{Name: "UpdateDataSource", Method: "PUT", URI: "/applications/{applicationId}/indices/{indexId}/datasources/{dataSourceId}"},
	{Name: "UpdateIndex", Method: "PUT", URI: "/applications/{applicationId}/indices/{indexId}"},
	{Name: "UpdatePlugin", Method: "PUT", URI: "/applications/{applicationId}/plugins/{pluginId}"},
	{Name: "UpdateRetriever", Method: "PUT", URI: "/applications/{applicationId}/retrievers/{retrieverId}"},
	{Name: "UpdateSubscription", Method: "PUT", URI: "/applications/{applicationId}/subscriptions/{subscriptionId}"},
	{Name: "UpdateUser", Method: "PUT", URI: "/applications/{applicationId}/users/{userId}"},
	{Name: "UpdateWebExperience", Method: "PUT", URI: "/applications/{applicationId}/experiences/{webExperienceId}"},
}

var qBusinessOperationByName = func() map[string]qBusinessOperation {
	out := make(map[string]qBusinessOperation, len(qBusinessOperations))
	for _, op := range qBusinessOperations {
		out[op.Name] = op
	}
	return out
}()
