package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	qBusinessDefaultRegion    = "us-east-1"
	qBusinessDefaultAccountID = "123456789012"
)

type qBusinessStore struct {
	mu sync.Mutex

	nextID int64

	applications            map[string]map[string]any
	indices                 map[string]map[string]any
	dataSources             map[string]map[string]any
	dataAccessors           map[string]map[string]any
	plugins                 map[string]map[string]any
	retrievers              map[string]map[string]any
	users                   map[string]map[string]any
	webExperiences          map[string]map[string]any
	groups                  map[string]map[string]any
	documents               map[string]map[string]any
	chatResponseConfigs     map[string]map[string]any
	subscriptions           map[string]map[string]any
	conversations           map[string]map[string]any
	messages                map[string]map[string]any
	attachments             map[string]map[string]any
	dataSourceSyncJobs      map[string][]map[string]any
	policyStatementsByAppID map[string][]map[string]any
	tags                    map[string]map[string]string
}

func newQBusinessStore() *qBusinessStore {
	now := time.Now().UTC()
	s := &qBusinessStore{
		nextID:                  2,
		applications:            map[string]map[string]any{},
		indices:                 map[string]map[string]any{},
		dataSources:             map[string]map[string]any{},
		dataAccessors:           map[string]map[string]any{},
		plugins:                 map[string]map[string]any{},
		retrievers:              map[string]map[string]any{},
		users:                   map[string]map[string]any{},
		webExperiences:          map[string]map[string]any{},
		groups:                  map[string]map[string]any{},
		documents:               map[string]map[string]any{},
		chatResponseConfigs:     map[string]map[string]any{},
		subscriptions:           map[string]map[string]any{},
		conversations:           map[string]map[string]any{},
		messages:                map[string]map[string]any{},
		attachments:             map[string]map[string]any{},
		dataSourceSyncJobs:      map[string][]map[string]any{},
		policyStatementsByAppID: map[string][]map[string]any{},
		tags:                    map[string]map[string]string{},
	}

	s.ensureApplicationLocked("app-000001", now)
	s.ensureIndexLocked("app-000001", "idx-000001", now)
	s.ensureDataSourceLocked("app-000001", "idx-000001", "ds-000001", now)
	s.ensureDataAccessorLocked("app-000001", "da-000001", now)
	s.ensurePluginLocked("app-000001", "plugin-000001", now)
	s.ensureRetrieverLocked("app-000001", "retriever-000001", now)
	s.ensureUserLocked("app-000001", "user-000001", now)
	s.ensureWebExperienceLocked("app-000001", "webexp-000001", now)
	s.ensureGroupLocked("app-000001", "idx-000001", "engineering", now)
	s.ensureDocumentLocked("app-000001", "idx-000001", "doc-000001", now)
	s.ensureChatResponseConfigurationLocked("app-000001", "crc-000001", now)
	s.ensureSubscriptionLocked("app-000001", "sub-000001", now)
	s.ensureConversationLocked("app-000001", "conv-000001", "user-000001", now)
	s.ensureMessageLocked("app-000001", "conv-000001", "msg-000001", now)
	s.ensureAttachmentLocked("app-000001", "conv-000001", "att-000001", now)
	s.ensureDataSourceSyncJobLocked("app-000001", "idx-000001", "ds-000001", "sync-000001", now)
	s.ensurePolicyStatementLocked("app-000001", "statement-000001")
	return s
}

func (s *qBusinessStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	appID := qBusinessFirstString(pathParams, payload, []string{"applicationId", "ApplicationId", "applicationID"}, "app-000001")
	indexID := qBusinessFirstString(pathParams, payload, []string{"indexId", "IndexId"}, "idx-000001")
	dataSourceID := qBusinessFirstString(pathParams, payload, []string{"dataSourceId", "DataSourceId"}, "ds-000001")
	dataAccessorID := qBusinessFirstString(pathParams, payload, []string{"dataAccessorId", "DataAccessorId"}, "da-000001")
	pluginID := qBusinessFirstString(pathParams, payload, []string{"pluginId", "PluginId"}, "plugin-000001")
	retrieverID := qBusinessFirstString(pathParams, payload, []string{"retrieverId", "RetrieverId"}, "retriever-000001")
	userID := qBusinessFirstString(pathParams, payload, []string{"userId", "UserId"}, qBusinessQueryString(query, "userId", "user-000001"))
	webExperienceID := qBusinessFirstString(pathParams, payload, []string{"webExperienceId", "WebExperienceId"}, "webexp-000001")
	groupName := qBusinessFirstString(pathParams, payload, []string{"groupName", "GroupName"}, "engineering")
	documentID := qBusinessFirstString(pathParams, payload, []string{"documentId", "DocumentId"}, "doc-000001")
	chatResponseConfigurationID := qBusinessFirstString(pathParams, payload, []string{"chatResponseConfigurationId", "ChatResponseConfigurationId"}, "crc-000001")
	subscriptionID := qBusinessFirstString(pathParams, payload, []string{"subscriptionId", "SubscriptionId"}, "sub-000001")
	conversationID := qBusinessFirstString(pathParams, payload, []string{"conversationId", "ConversationId"}, "conv-000001")
	messageID := qBusinessFirstString(pathParams, payload, []string{"messageId", "MessageId"}, "msg-000001")
	attachmentID := qBusinessFirstString(pathParams, payload, []string{"attachmentId", "AttachmentId"}, "att-000001")
	resourceARN := qBusinessFirstString(pathParams, payload, []string{"resourceARN", "resourceArn", "ResourceArn"}, qBusinessApplicationARN(appID))
	statementID := qBusinessFirstString(pathParams, payload, []string{"statementId", "StatementId"}, "statement-000001")

	app := s.ensureApplicationLocked(appID, now)
	idx := s.ensureIndexLocked(appID, indexID, now)
	dataSource := s.ensureDataSourceLocked(appID, indexID, dataSourceID, now)
	dataAccessor := s.ensureDataAccessorLocked(appID, dataAccessorID, now)
	plugin := s.ensurePluginLocked(appID, pluginID, now)
	retriever := s.ensureRetrieverLocked(appID, retrieverID, now)
	user := s.ensureUserLocked(appID, userID, now)
	webExp := s.ensureWebExperienceLocked(appID, webExperienceID, now)
	group := s.ensureGroupLocked(appID, indexID, groupName, now)
	doc := s.ensureDocumentLocked(appID, indexID, documentID, now)
	chatCfg := s.ensureChatResponseConfigurationLocked(appID, chatResponseConfigurationID, now)
	subscription := s.ensureSubscriptionLocked(appID, subscriptionID, now)
	conversation := s.ensureConversationLocked(appID, conversationID, userID, now)
	msg := s.ensureMessageLocked(appID, conversationID, messageID, now)
	attachment := s.ensureAttachmentLocked(appID, conversationID, attachmentID, now)
	s.ensureDataSourceSyncJobLocked(appID, indexID, dataSourceID, "sync-000001", now)
	s.ensurePolicyStatementLocked(appID, statementID)
	s.ensureTagMapLocked(resourceARN)
	s.applyPayloadUpdatesLocked(payload, app, idx, dataSource, dataAccessor, plugin, retriever, user, webExp, group, doc, chatCfg, subscription, conversation, msg, attachment, now)
	s.applyTagMutationsLocked(action, payload, resourceARN, query)

	switch action {
	case "CreateApplication":
		createdID := qBusinessPayloadString(payload, []string{"applicationId", "applicationID", "ApplicationId"}, s.nextIDLocked("app"))
		created := s.ensureApplicationLocked(createdID, now)
		s.applyPayloadUpdatesLocked(payload, created, created, created, created, created, created, created, created, created, created, created, created, created, created, created, now)
		s.mergeTagsLocked(qBusinessExtractTags(payload), qBusinessAnyString(created, "applicationArn", qBusinessApplicationARN(createdID)))
		return map[string]any{
			"applicationId":  createdID,
			"applicationArn": created["applicationArn"],
			"displayName":    created["displayName"],
			"status":         created["status"],
		}
	case "DeleteApplication":
		s.deleteApplicationLocked(appID)
		return map[string]any{}
	case "GetApplication":
		return qBusinessCloneMap(app)
	case "ListApplications":
		return map[string]any{"applications": qBusinessSortedSummaries(s.applications, "applicationId"), "nextToken": ""}
	case "UpdateApplication":
		return map[string]any{"applicationId": appID}

	case "CreateIndex":
		createdID := qBusinessPayloadString(payload, []string{"indexId", "IndexId"}, s.nextIDLocked("idx"))
		created := s.ensureIndexLocked(appID, createdID, now)
		s.mergeTagsLocked(qBusinessExtractTags(payload), qBusinessAnyString(created, "indexArn", qBusinessIndexARN(appID, createdID)))
		return map[string]any{"indexId": createdID, "indexArn": created["indexArn"]}
	case "DeleteIndex":
		s.deleteByKeyPrefixLocked(s.indices, qBusinessIndexPrefix(appID, indexID))
		return map[string]any{}
	case "GetIndex":
		return qBusinessCloneMap(idx)
	case "ListIndices":
		return map[string]any{"indices": qBusinessFilterSummaries(s.indices, qBusinessAppPrefix(appID), "indexId"), "nextToken": ""}
	case "UpdateIndex":
		return map[string]any{"indexId": indexID}

	case "CreateDataSource":
		createdID := qBusinessPayloadString(payload, []string{"dataSourceId", "DataSourceId"}, s.nextIDLocked("ds"))
		created := s.ensureDataSourceLocked(appID, indexID, createdID, now)
		s.mergeTagsLocked(qBusinessExtractTags(payload), qBusinessAnyString(created, "dataSourceArn", qBusinessDataSourceARN(appID, indexID, createdID)))
		return map[string]any{"dataSourceId": createdID, "dataSourceArn": created["dataSourceArn"]}
	case "DeleteDataSource":
		delete(s.dataSources, qBusinessDataSourceKey(appID, indexID, dataSourceID))
		return map[string]any{}
	case "GetDataSource":
		return qBusinessCloneMap(dataSource)
	case "ListDataSources":
		return map[string]any{"dataSources": qBusinessFilterSummaries(s.dataSources, qBusinessIndexPrefix(appID, indexID), "dataSourceId"), "nextToken": ""}
	case "UpdateDataSource":
		return map[string]any{"dataSourceId": dataSourceID}

	case "StartDataSourceSyncJob":
		jobID := s.nextIDLocked("sync")
		job := s.ensureDataSourceSyncJobLocked(appID, indexID, dataSourceID, jobID, now)
		job["status"] = "SYNCING"
		return map[string]any{"executionId": jobID}
	case "StopDataSourceSyncJob":
		jobs := s.dataSourceSyncJobs[qBusinessDataSourceKey(appID, indexID, dataSourceID)]
		for i := range jobs {
			jobs[i]["status"] = "STOPPED"
		}
		return map[string]any{"executionId": "sync-000001"}
	case "ListDataSourceSyncJobs":
		return map[string]any{"history": qBusinessCloneMapSlice(s.dataSourceSyncJobs[qBusinessDataSourceKey(appID, indexID, dataSourceID)]), "nextToken": ""}

	case "BatchPutDocument":
		docIDs := []string{}
		for _, entry := range qBusinessPayloadSlice(payload, "documents") {
			m, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			id := qBusinessPayloadString(m, []string{"id", "documentId", "DocumentId"}, s.nextIDLocked("doc"))
			docIDs = append(docIDs, id)
			s.ensureDocumentLocked(appID, indexID, id, now)
		}
		if len(docIDs) == 0 {
			docIDs = append(docIDs, documentID)
			s.ensureDocumentLocked(appID, indexID, documentID, now)
		}
		return map[string]any{"failedDocuments": []any{}, "documentIds": docIDs}
	case "BatchDeleteDocument":
		for _, id := range qBusinessPayloadStringSlice(payload, "documentIds") {
			delete(s.documents, qBusinessDocumentKey(appID, indexID, id))
		}
		return map[string]any{"failedDocuments": []any{}}
	case "ListDocuments":
		return map[string]any{"documents": qBusinessFilterSummaries(s.documents, qBusinessIndexPrefix(appID, indexID), "documentId"), "nextToken": ""}
	case "GetDocumentContent":
		return map[string]any{
			"documentId":   documentID,
			"outputFormat": qBusinessQueryString(query, "outputFormat", "MARKDOWN"),
			"url":          "https://stackyard.local/qbusiness/document/" + documentID,
		}
	case "CheckDocumentAccess":
		return map[string]any{"documentId": documentID, "hasAccess": true, "documentAcl": []any{}}

	case "CreateDataAccessor":
		createdID := qBusinessPayloadString(payload, []string{"dataAccessorId", "DataAccessorId"}, s.nextIDLocked("da"))
		created := s.ensureDataAccessorLocked(appID, createdID, now)
		s.mergeTagsLocked(qBusinessExtractTags(payload), qBusinessAnyString(created, "dataAccessorArn", qBusinessDataAccessorARN(appID, createdID)))
		return map[string]any{"dataAccessorId": createdID}
	case "DeleteDataAccessor":
		delete(s.dataAccessors, qBusinessScopedKey(appID, dataAccessorID))
		return map[string]any{}
	case "GetDataAccessor":
		return qBusinessCloneMap(dataAccessor)
	case "ListDataAccessors":
		return map[string]any{"dataAccessors": qBusinessFilterSummaries(s.dataAccessors, qBusinessAppPrefix(appID), "dataAccessorId"), "nextToken": ""}
	case "UpdateDataAccessor":
		return map[string]any{"dataAccessorId": dataAccessorID}

	case "CreatePlugin":
		createdID := qBusinessPayloadString(payload, []string{"pluginId", "PluginId"}, s.nextIDLocked("plugin"))
		created := s.ensurePluginLocked(appID, createdID, now)
		return map[string]any{"pluginId": createdID, "pluginArn": created["pluginArn"]}
	case "DeletePlugin":
		delete(s.plugins, qBusinessScopedKey(appID, pluginID))
		return map[string]any{}
	case "GetPlugin":
		return qBusinessCloneMap(plugin)
	case "ListPlugins":
		return map[string]any{"plugins": qBusinessFilterSummaries(s.plugins, qBusinessAppPrefix(appID), "pluginId"), "nextToken": ""}
	case "UpdatePlugin":
		return map[string]any{"pluginId": pluginID}
	case "ListPluginActions":
		return map[string]any{"pluginActions": []any{map[string]any{"pluginId": pluginID, "actionId": "action-000001"}}, "nextToken": ""}
	case "ListPluginTypeActions":
		return map[string]any{"pluginTypeActions": []any{map[string]any{"pluginType": qBusinessPathString(pathParams, "pluginType", "CUSTOM"), "actionId": "action-000001"}}, "nextToken": ""}
	case "ListPluginTypeMetadata":
		return map[string]any{"pluginTypeMetadataSummaries": []any{map[string]any{"pluginType": "CUSTOM", "displayName": "Custom Plugin"}}}

	case "CreateRetriever":
		createdID := qBusinessPayloadString(payload, []string{"retrieverId", "RetrieverId"}, s.nextIDLocked("retriever"))
		created := s.ensureRetrieverLocked(appID, createdID, now)
		return map[string]any{"retrieverId": createdID, "retrieverArn": created["retrieverArn"]}
	case "DeleteRetriever":
		delete(s.retrievers, qBusinessScopedKey(appID, retrieverID))
		return map[string]any{}
	case "GetRetriever":
		return qBusinessCloneMap(retriever)
	case "ListRetrievers":
		return map[string]any{"retrievers": qBusinessFilterSummaries(s.retrievers, qBusinessAppPrefix(appID), "retrieverId"), "nextToken": ""}
	case "UpdateRetriever":
		return map[string]any{"retrieverId": retrieverID}

	case "CreateUser":
		createdID := qBusinessPayloadString(payload, []string{"userId", "UserId"}, s.nextIDLocked("user"))
		created := s.ensureUserLocked(appID, createdID, now)
		return map[string]any{"userId": createdID, "userArn": created["userArn"]}
	case "DeleteUser":
		delete(s.users, qBusinessScopedKey(appID, userID))
		return map[string]any{}
	case "GetUser":
		return qBusinessCloneMap(user)
	case "UpdateUser":
		return map[string]any{"userId": userID}

	case "PutGroup":
		updated := s.ensureGroupLocked(appID, indexID, groupName, now)
		return map[string]any{"groupName": updated["groupName"], "status": "ACTIVE"}
	case "DeleteGroup":
		delete(s.groups, qBusinessGroupKey(appID, indexID, groupName))
		return map[string]any{}
	case "GetGroup":
		return qBusinessCloneMap(group)
	case "ListGroups":
		return map[string]any{"groups": qBusinessFilterSummaries(s.groups, qBusinessIndexPrefix(appID, indexID), "groupName"), "nextToken": ""}

	case "CreateWebExperience":
		createdID := qBusinessPayloadString(payload, []string{"webExperienceId", "WebExperienceId"}, s.nextIDLocked("webexp"))
		created := s.ensureWebExperienceLocked(appID, createdID, now)
		return map[string]any{"webExperienceId": createdID, "webExperienceArn": created["webExperienceArn"]}
	case "CreateAnonymousWebExperienceUrl":
		return map[string]any{"url": fmt.Sprintf("https://stackyard.local/qbusiness/%s/%s/anonymous", appID, webExperienceID)}
	case "DeleteWebExperience":
		delete(s.webExperiences, qBusinessScopedKey(appID, webExperienceID))
		return map[string]any{}
	case "GetWebExperience":
		return qBusinessCloneMap(webExp)
	case "ListWebExperiences":
		return map[string]any{"webExperiences": qBusinessFilterSummaries(s.webExperiences, qBusinessAppPrefix(appID), "webExperienceId"), "nextToken": ""}
	case "UpdateWebExperience":
		return map[string]any{"webExperienceId": webExperienceID}

	case "CreateChatResponseConfiguration":
		createdID := qBusinessPayloadString(payload, []string{"chatResponseConfigurationId", "ChatResponseConfigurationId"}, s.nextIDLocked("crc"))
		created := s.ensureChatResponseConfigurationLocked(appID, createdID, now)
		return map[string]any{"chatResponseConfigurationId": createdID, "chatResponseConfigurationArn": created["chatResponseConfigurationArn"]}
	case "DeleteChatResponseConfiguration":
		delete(s.chatResponseConfigs, qBusinessScopedKey(appID, chatResponseConfigurationID))
		return map[string]any{}
	case "GetChatResponseConfiguration":
		return qBusinessCloneMap(chatCfg)
	case "ListChatResponseConfigurations":
		return map[string]any{"chatResponseConfigurations": qBusinessFilterSummaries(s.chatResponseConfigs, qBusinessAppPrefix(appID), "chatResponseConfigurationId"), "nextToken": ""}
	case "UpdateChatResponseConfiguration":
		return map[string]any{"chatResponseConfigurationId": chatResponseConfigurationID}

	case "Chat", "ChatSync":
		cid := qBusinessPayloadString(payload, []string{"conversationId", "ConversationId"}, conversationID)
		conv := s.ensureConversationLocked(appID, cid, userID, now)
		mid := s.nextIDLocked("msg")
		message := s.ensureMessageLocked(appID, cid, mid, now)
		message["body"] = qBusinessPayloadString(payload, []string{"userMessage", "UserMessage", "message", "Message"}, "Hello from Stackyard QBusiness")
		return map[string]any{
			"conversationId":  conv["conversationId"],
			"systemMessageId": message["messageId"],
			"sourceAttributions": []any{
				map[string]any{"title": "Stackyard Seed Document", "excerpt": "QBusiness mock response"},
			},
		}
	case "DeleteConversation":
		s.deleteByKeyPrefixLocked(s.messages, qBusinessConversationPrefix(appID, conversationID))
		s.deleteByKeyPrefixLocked(s.attachments, qBusinessConversationPrefix(appID, conversationID))
		delete(s.conversations, qBusinessScopedKey(appID, conversationID))
		return map[string]any{}
	case "ListConversations":
		return map[string]any{"conversations": qBusinessFilterSummaries(s.conversations, qBusinessAppPrefix(appID), "conversationId"), "nextToken": ""}
	case "ListMessages":
		return map[string]any{"messages": qBusinessFilterSummaries(s.messages, qBusinessConversationPrefix(appID, conversationID), "messageId"), "nextToken": ""}
	case "GetMedia":
		return map[string]any{"mediaId": qBusinessPathString(pathParams, "mediaId", "media-000001"), "url": "https://stackyard.local/media/" + qBusinessPathString(pathParams, "mediaId", "media-000001")}
	case "PutFeedback":
		return map[string]any{}

	case "ListAttachments":
		return map[string]any{"attachments": qBusinessFilterSummaries(s.attachments, qBusinessConversationPrefix(appID, conversationID), "attachmentId"), "nextToken": ""}
	case "DeleteAttachment":
		delete(s.attachments, qBusinessAttachmentKey(appID, conversationID, attachmentID))
		return map[string]any{}

	case "CreateSubscription":
		createdID := qBusinessPayloadString(payload, []string{"subscriptionId", "SubscriptionId"}, s.nextIDLocked("sub"))
		created := s.ensureSubscriptionLocked(appID, createdID, now)
		return map[string]any{"subscriptionId": createdID, "subscriptionArn": created["subscriptionArn"]}
	case "CancelSubscription":
		delete(s.subscriptions, qBusinessScopedKey(appID, subscriptionID))
		return map[string]any{}
	case "UpdateSubscription":
		return map[string]any{"subscriptionId": subscriptionID}
	case "ListSubscriptions":
		return map[string]any{"subscriptions": qBusinessFilterSummaries(s.subscriptions, qBusinessAppPrefix(appID), "subscriptionId"), "nextToken": ""}

	case "AssociatePermission":
		s.ensurePolicyStatementLocked(appID, statementID)
		return map[string]any{}
	case "DisassociatePermission":
		s.removePolicyStatementLocked(appID, statementID)
		return map[string]any{}
	case "GetPolicy":
		return map[string]any{"statements": qBusinessCloneMapSlice(s.policyStatementsByAppID[appID])}

	case "GetChatControlsConfiguration":
		return map[string]any{"applicationId": appID, "responseScope": "DEFAULT", "blockedPhrases": []any{}}
	case "UpdateChatControlsConfiguration":
		return map[string]any{"applicationId": appID}

	case "ListTagsForResource":
		return map[string]any{"tags": qBusinessTagsAsList(s.ensureTagMapLocked(resourceARN))}
	case "TagResource", "UntagResource":
		return map[string]any{}

	case "SearchRelevantContent":
		return map[string]any{"relevantContent": []any{map[string]any{"documentId": documentID, "score": 1.0}}}
	}

	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"nextToken": ""}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *qBusinessStore) applyPayloadUpdatesLocked(
	payload map[string]any,
	app, idx, dataSource, dataAccessor, plugin, retriever, user, webExp, group, doc, chatCfg, subscription, conversation, msg, attachment map[string]any,
	now time.Time,
) {
	if displayName := qBusinessPayloadString(payload, []string{"displayName", "DisplayName", "name", "Name"}, ""); displayName != "" {
		app["displayName"] = displayName
		idx["displayName"] = displayName
		dataSource["displayName"] = displayName
		dataAccessor["displayName"] = displayName
		plugin["displayName"] = displayName
		retriever["displayName"] = displayName
		webExp["displayName"] = displayName
		chatCfg["displayName"] = displayName
	}
	if status := qBusinessPayloadString(payload, []string{"status", "Status"}, ""); status != "" {
		app["status"] = status
		idx["status"] = status
		dataSource["status"] = status
		subscription["status"] = status
	}
	if description := qBusinessPayloadString(payload, []string{"description", "Description"}, ""); description != "" {
		app["description"] = description
		idx["description"] = description
		dataSource["description"] = description
		dataAccessor["description"] = description
		plugin["description"] = description
		retriever["description"] = description
		user["description"] = description
		webExp["description"] = description
		group["description"] = description
		doc["description"] = description
		chatCfg["description"] = description
		subscription["description"] = description
		conversation["description"] = description
		msg["description"] = description
		attachment["description"] = description
	}

	ts := qBusinessTime(now)
	app["updatedAt"] = ts
	idx["updatedAt"] = ts
	dataSource["updatedAt"] = ts
	dataAccessor["updatedAt"] = ts
	plugin["updatedAt"] = ts
	retriever["updatedAt"] = ts
	user["updatedAt"] = ts
	webExp["updatedAt"] = ts
	group["updatedAt"] = ts
	doc["updatedAt"] = ts
	chatCfg["updatedAt"] = ts
	subscription["updatedAt"] = ts
	conversation["updatedAt"] = ts
	msg["updatedAt"] = ts
	attachment["updatedAt"] = ts
}

func (s *qBusinessStore) applyTagMutationsLocked(action string, payload map[string]any, resourceARN string, query url.Values) {
	switch action {
	case "TagResource":
		s.mergeTagsLocked(qBusinessExtractTags(payload), resourceARN)
	case "UntagResource":
		keys := qBusinessPayloadStringSlice(payload, "tagKeys")
		if len(keys) == 0 {
			keys = qBusinessQuerySlice(query, "tagKeys")
		}
		for _, key := range keys {
			delete(s.ensureTagMapLocked(resourceARN), key)
		}
	}
}

func (s *qBusinessStore) nextIDLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func (s *qBusinessStore) ensureApplicationLocked(appID string, now time.Time) map[string]any {
	appID = qBusinessDefaultString(appID, "app-000001")
	if existing := s.applications[appID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":  appID,
		"applicationArn": qBusinessApplicationARN(appID),
		"displayName":    "stackyard-qbusiness-app",
		"description":    "",
		"status":         "ACTIVE",
		"createdAt":      qBusinessTime(now),
		"updatedAt":      qBusinessTime(now),
	}
	s.applications[appID] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "applicationArn", qBusinessApplicationARN(appID)))
	return item
}

func (s *qBusinessStore) ensureIndexLocked(appID, indexID string, now time.Time) map[string]any {
	key := qBusinessIndexKey(appID, indexID)
	if existing := s.indices[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"indexId":       indexID,
		"indexArn":      qBusinessIndexARN(appID, indexID),
		"displayName":   "stackyard-index",
		"status":        "ACTIVE",
		"createdAt":     qBusinessTime(now),
		"updatedAt":     qBusinessTime(now),
	}
	s.indices[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "indexArn", qBusinessIndexARN(appID, indexID)))
	return item
}

func (s *qBusinessStore) ensureDataSourceLocked(appID, indexID, dataSourceID string, now time.Time) map[string]any {
	key := qBusinessDataSourceKey(appID, indexID, dataSourceID)
	if existing := s.dataSources[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"indexId":       indexID,
		"dataSourceId":  dataSourceID,
		"dataSourceArn": qBusinessDataSourceARN(appID, indexID, dataSourceID),
		"displayName":   "stackyard-data-source",
		"status":        "ACTIVE",
		"createdAt":     qBusinessTime(now),
		"updatedAt":     qBusinessTime(now),
	}
	s.dataSources[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "dataSourceArn", qBusinessDataSourceARN(appID, indexID, dataSourceID)))
	return item
}

func (s *qBusinessStore) ensureDataAccessorLocked(appID, dataAccessorID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, dataAccessorID)
	if existing := s.dataAccessors[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":   appID,
		"dataAccessorId":  dataAccessorID,
		"dataAccessorArn": qBusinessDataAccessorARN(appID, dataAccessorID),
		"displayName":     "stackyard-data-accessor",
		"status":          "ACTIVE",
		"createdAt":       qBusinessTime(now),
		"updatedAt":       qBusinessTime(now),
	}
	s.dataAccessors[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "dataAccessorArn", qBusinessDataAccessorARN(appID, dataAccessorID)))
	return item
}

func (s *qBusinessStore) ensurePluginLocked(appID, pluginID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, pluginID)
	if existing := s.plugins[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"pluginId":      pluginID,
		"pluginArn":     qBusinessPluginARN(appID, pluginID),
		"displayName":   "stackyard-plugin",
		"status":        "ACTIVE",
		"createdAt":     qBusinessTime(now),
		"updatedAt":     qBusinessTime(now),
	}
	s.plugins[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "pluginArn", qBusinessPluginARN(appID, pluginID)))
	return item
}

func (s *qBusinessStore) ensureRetrieverLocked(appID, retrieverID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, retrieverID)
	if existing := s.retrievers[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"retrieverId":   retrieverID,
		"retrieverArn":  qBusinessRetrieverARN(appID, retrieverID),
		"displayName":   "stackyard-retriever",
		"status":        "ACTIVE",
		"createdAt":     qBusinessTime(now),
		"updatedAt":     qBusinessTime(now),
	}
	s.retrievers[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "retrieverArn", qBusinessRetrieverARN(appID, retrieverID)))
	return item
}

func (s *qBusinessStore) ensureUserLocked(appID, userID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, userID)
	if existing := s.users[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"userId":        userID,
		"userArn":       qBusinessUserARN(appID, userID),
		"displayName":   "stackyard-user",
		"status":        "ACTIVE",
		"createdAt":     qBusinessTime(now),
		"updatedAt":     qBusinessTime(now),
	}
	s.users[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "userArn", qBusinessUserARN(appID, userID)))
	return item
}

func (s *qBusinessStore) ensureWebExperienceLocked(appID, webExperienceID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, webExperienceID)
	if existing := s.webExperiences[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":    appID,
		"webExperienceId":  webExperienceID,
		"webExperienceArn": qBusinessWebExperienceARN(appID, webExperienceID),
		"displayName":      "stackyard-web-experience",
		"status":           "ACTIVE",
		"createdAt":        qBusinessTime(now),
		"updatedAt":        qBusinessTime(now),
	}
	s.webExperiences[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "webExperienceArn", qBusinessWebExperienceARN(appID, webExperienceID)))
	return item
}

func (s *qBusinessStore) ensureGroupLocked(appID, indexID, groupName string, now time.Time) map[string]any {
	key := qBusinessGroupKey(appID, indexID, groupName)
	if existing := s.groups[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"indexId":       indexID,
		"groupName":     groupName,
		"status":        "ACTIVE",
		"updatedAt":     qBusinessTime(now),
	}
	s.groups[key] = item
	return item
}

func (s *qBusinessStore) ensureDocumentLocked(appID, indexID, documentID string, now time.Time) map[string]any {
	key := qBusinessDocumentKey(appID, indexID, documentID)
	if existing := s.documents[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId": appID,
		"indexId":       indexID,
		"documentId":    documentID,
		"title":         "Stackyard Seed Document",
		"status":        "AVAILABLE",
		"updatedAt":     qBusinessTime(now),
	}
	s.documents[key] = item
	return item
}

func (s *qBusinessStore) ensureChatResponseConfigurationLocked(appID, chatResponseConfigurationID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, chatResponseConfigurationID)
	if existing := s.chatResponseConfigs[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":                appID,
		"chatResponseConfigurationId":  chatResponseConfigurationID,
		"chatResponseConfigurationArn": qBusinessChatResponseConfigurationARN(appID, chatResponseConfigurationID),
		"displayName":                  "stackyard-chat-response-config",
		"status":                       "ACTIVE",
		"createdAt":                    qBusinessTime(now),
		"updatedAt":                    qBusinessTime(now),
	}
	s.chatResponseConfigs[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "chatResponseConfigurationArn", qBusinessChatResponseConfigurationARN(appID, chatResponseConfigurationID)))
	return item
}

func (s *qBusinessStore) ensureSubscriptionLocked(appID, subscriptionID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, subscriptionID)
	if existing := s.subscriptions[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":   appID,
		"subscriptionId":  subscriptionID,
		"subscriptionArn": qBusinessSubscriptionARN(appID, subscriptionID),
		"status":          "ACTIVE",
		"type":            "Q_BUSINESS",
		"createdAt":       qBusinessTime(now),
		"updatedAt":       qBusinessTime(now),
	}
	s.subscriptions[key] = item
	s.ensureTagMapLocked(qBusinessAnyString(item, "subscriptionArn", qBusinessSubscriptionARN(appID, subscriptionID)))
	return item
}

func (s *qBusinessStore) ensureConversationLocked(appID, conversationID, userID string, now time.Time) map[string]any {
	key := qBusinessScopedKey(appID, conversationID)
	if existing := s.conversations[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":  appID,
		"conversationId": conversationID,
		"userId":         userID,
		"title":          "Stackyard Conversation",
		"createdAt":      qBusinessTime(now),
		"updatedAt":      qBusinessTime(now),
	}
	s.conversations[key] = item
	return item
}

func (s *qBusinessStore) ensureMessageLocked(appID, conversationID, messageID string, now time.Time) map[string]any {
	key := qBusinessMessageKey(appID, conversationID, messageID)
	if existing := s.messages[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":  appID,
		"conversationId": conversationID,
		"messageId":      messageID,
		"body":           "Stackyard QBusiness message",
		"type":           "SYSTEM",
		"createdAt":      qBusinessTime(now),
		"updatedAt":      qBusinessTime(now),
	}
	s.messages[key] = item
	return item
}

func (s *qBusinessStore) ensureAttachmentLocked(appID, conversationID, attachmentID string, now time.Time) map[string]any {
	key := qBusinessAttachmentKey(appID, conversationID, attachmentID)
	if existing := s.attachments[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"applicationId":  appID,
		"conversationId": conversationID,
		"attachmentId":   attachmentID,
		"name":           "stackyard.txt",
		"status":         "AVAILABLE",
		"createdAt":      qBusinessTime(now),
		"updatedAt":      qBusinessTime(now),
	}
	s.attachments[key] = item
	return item
}

func (s *qBusinessStore) ensureDataSourceSyncJobLocked(appID, indexID, dataSourceID, syncJobID string, now time.Time) map[string]any {
	key := qBusinessDataSourceKey(appID, indexID, dataSourceID)
	if s.dataSourceSyncJobs[key] == nil {
		s.dataSourceSyncJobs[key] = []map[string]any{}
	}
	for _, existing := range s.dataSourceSyncJobs[key] {
		if qBusinessAnyString(existing, "executionId", "") == syncJobID {
			return existing
		}
	}
	job := map[string]any{
		"executionId": syncJobID,
		"status":      "SUCCEEDED",
		"startTime":   qBusinessTime(now),
		"endTime":     qBusinessTime(now),
	}
	s.dataSourceSyncJobs[key] = append(s.dataSourceSyncJobs[key], job)
	return job
}

func (s *qBusinessStore) ensurePolicyStatementLocked(appID, statementID string) {
	statements := s.policyStatementsByAppID[appID]
	for _, statement := range statements {
		if qBusinessAnyString(statement, "statementId", "") == statementID {
			return
		}
	}
	s.policyStatementsByAppID[appID] = append(s.policyStatementsByAppID[appID], map[string]any{
		"statementId": statementID,
		"effect":      "Allow",
		"principal":   map[string]any{"user": "user-000001"},
		"action":      []any{"qbusiness:Chat"},
		"resource":    qBusinessApplicationARN(appID),
	})
}

func (s *qBusinessStore) removePolicyStatementLocked(appID, statementID string) {
	statements := s.policyStatementsByAppID[appID]
	out := make([]map[string]any, 0, len(statements))
	for _, statement := range statements {
		if qBusinessAnyString(statement, "statementId", "") == statementID {
			continue
		}
		out = append(out, statement)
	}
	s.policyStatementsByAppID[appID] = out
}

func (s *qBusinessStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = qBusinessApplicationARN("app-000001")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	s.tags[resourceARN] = map[string]string{"service": "qbusiness", "seed": "true"}
	return s.tags[resourceARN]
}

func (s *qBusinessStore) mergeTagsLocked(tags map[string]string, resourceARN string) {
	if len(tags) == 0 {
		return
	}
	dest := s.ensureTagMapLocked(resourceARN)
	for key, value := range tags {
		dest[key] = value
	}
}

func (s *qBusinessStore) deleteApplicationLocked(appID string) {
	delete(s.applications, appID)
	s.deleteByKeyPrefixLocked(s.indices, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.dataSources, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.dataAccessors, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.plugins, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.retrievers, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.users, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.webExperiences, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.groups, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.documents, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.chatResponseConfigs, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.subscriptions, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.conversations, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.messages, qBusinessAppPrefix(appID))
	s.deleteByKeyPrefixLocked(s.attachments, qBusinessAppPrefix(appID))
	delete(s.policyStatementsByAppID, appID)
}

func (s *qBusinessStore) deleteByKeyPrefixLocked(store map[string]map[string]any, prefix string) {
	for key := range store {
		if strings.HasPrefix(key, prefix) {
			delete(store, key)
		}
	}
}

func qBusinessSortedSummaries(store map[string]map[string]any, idKey string) []any {
	keys := make([]string, 0, len(store))
	for key := range store {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, qBusinessSummary(store[key], idKey))
	}
	return out
}

func qBusinessFilterSummaries(store map[string]map[string]any, prefix, idKey string) []any {
	keys := make([]string, 0, len(store))
	for key := range store {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, qBusinessSummary(store[key], idKey))
	}
	return out
}

func qBusinessSummary(item map[string]any, idKey string) map[string]any {
	return map[string]any{
		idKey:       qBusinessAnyString(item, idKey, "id-000001"),
		"status":    qBusinessAnyString(item, "status", "ACTIVE"),
		"updatedAt": qBusinessAnyString(item, "updatedAt", qBusinessTime(time.Now().UTC())),
	}
}

func qBusinessCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func qBusinessCloneMapSlice(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, qBusinessCloneMap(item))
	}
	return out
}

func qBusinessCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func qBusinessFirstString(pathParams map[string]string, payload map[string]any, names []string, def string) string {
	for _, name := range names {
		if value := qBusinessPathString(pathParams, name, ""); value != "" {
			return value
		}
	}
	if payload != nil {
		if value := qBusinessPayloadString(payload, names, ""); value != "" {
			return value
		}
	}
	return def
}

func qBusinessPathString(pathParams map[string]string, name, def string) string {
	if pathParams == nil {
		return def
	}
	for key, value := range pathParams {
		if strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(name)) {
			clean := strings.TrimSpace(value)
			if clean != "" {
				return clean
			}
		}
	}
	return def
}

func qBusinessPayloadString(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		for existingKey, value := range payload {
			if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
				continue
			}
			switch typed := value.(type) {
			case string:
				if clean := strings.TrimSpace(typed); clean != "" {
					return clean
				}
			case fmt.Stringer:
				if clean := strings.TrimSpace(typed.String()); clean != "" {
					return clean
				}
			default:
				if clean := strings.TrimSpace(fmt.Sprintf("%v", value)); clean != "" && clean != "<nil>" {
					return clean
				}
			}
		}
	}
	return def
}

func qBusinessPayloadSlice(payload map[string]any, key string) []any {
	for existingKey, value := range payload {
		if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			continue
		}
		if out, ok := value.([]any); ok {
			return out
		}
	}
	return nil
}

func qBusinessPayloadStringSlice(payload map[string]any, key string) []string {
	out := []string{}
	for _, entry := range qBusinessPayloadSlice(payload, key) {
		clean := strings.TrimSpace(fmt.Sprintf("%v", entry))
		if clean != "" && clean != "<nil>" {
			out = append(out, clean)
		}
	}
	return out
}

func qBusinessQueryString(query url.Values, key, def string) string {
	for existingKey, values := range query {
		if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			continue
		}
		for _, value := range values {
			clean := strings.TrimSpace(value)
			if clean != "" {
				return clean
			}
		}
	}
	return def
}

func qBusinessQuerySlice(query url.Values, key string) []string {
	out := []string{}
	for existingKey, values := range query {
		if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			continue
		}
		for _, value := range values {
			clean := strings.TrimSpace(value)
			if clean != "" {
				out = append(out, clean)
			}
		}
	}
	return out
}

func qBusinessExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for key, value := range payload {
		if !strings.EqualFold(strings.TrimSpace(key), "tags") {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			for tagKey, tagValue := range typed {
				out[strings.TrimSpace(tagKey)] = strings.TrimSpace(fmt.Sprintf("%v", tagValue))
			}
		case []any:
			for _, entry := range typed {
				tagMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				tagKey := qBusinessPayloadString(tagMap, []string{"key", "Key"}, "")
				if tagKey == "" {
					continue
				}
				tagValue := qBusinessPayloadString(tagMap, []string{"value", "Value"}, "")
				out[tagKey] = tagValue
			}
		}
	}
	return out
}

func qBusinessTagsAsList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"key": key, "value": tags[key]})
	}
	return out
}

func qBusinessAnyString(item map[string]any, key, def string) string {
	if item == nil {
		return def
	}
	for existingKey, value := range item {
		if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			continue
		}
		clean := strings.TrimSpace(fmt.Sprintf("%v", value))
		if clean != "" && clean != "<nil>" {
			return clean
		}
	}
	return def
}

func qBusinessDefaultString(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func qBusinessTime(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}

func qBusinessAppPrefix(appID string) string {
	return qBusinessDefaultString(appID, "app-000001") + "::"
}

func qBusinessScopedKey(appID, id string) string {
	return qBusinessAppPrefix(appID) + qBusinessDefaultString(id, "id-000001")
}

func qBusinessIndexKey(appID, indexID string) string {
	return qBusinessScopedKey(appID, qBusinessDefaultString(indexID, "idx-000001"))
}

func qBusinessIndexPrefix(appID, indexID string) string {
	return qBusinessIndexKey(appID, indexID) + "::"
}

func qBusinessDataSourceKey(appID, indexID, dataSourceID string) string {
	return qBusinessIndexPrefix(appID, indexID) + qBusinessDefaultString(dataSourceID, "ds-000001")
}

func qBusinessGroupKey(appID, indexID, groupName string) string {
	return qBusinessIndexPrefix(appID, indexID) + qBusinessDefaultString(groupName, "engineering")
}

func qBusinessDocumentKey(appID, indexID, documentID string) string {
	return qBusinessIndexPrefix(appID, indexID) + qBusinessDefaultString(documentID, "doc-000001")
}

func qBusinessConversationPrefix(appID, conversationID string) string {
	return qBusinessScopedKey(appID, conversationID) + "::"
}

func qBusinessMessageKey(appID, conversationID, messageID string) string {
	return qBusinessConversationPrefix(appID, conversationID) + qBusinessDefaultString(messageID, "msg-000001")
}

func qBusinessAttachmentKey(appID, conversationID, attachmentID string) string {
	return qBusinessConversationPrefix(appID, conversationID) + qBusinessDefaultString(attachmentID, "att-000001")
}

func qBusinessApplicationARN(appID string) string {
	return "arn:aws:qbusiness:" + qBusinessDefaultRegion + ":" + qBusinessDefaultAccountID + ":application/" + qBusinessDefaultString(appID, "app-000001")
}

func qBusinessIndexARN(appID, indexID string) string {
	return qBusinessApplicationARN(appID) + "/index/" + qBusinessDefaultString(indexID, "idx-000001")
}

func qBusinessDataSourceARN(appID, indexID, dataSourceID string) string {
	return qBusinessIndexARN(appID, indexID) + "/datasource/" + qBusinessDefaultString(dataSourceID, "ds-000001")
}

func qBusinessDataAccessorARN(appID, dataAccessorID string) string {
	return qBusinessApplicationARN(appID) + "/dataaccessor/" + qBusinessDefaultString(dataAccessorID, "da-000001")
}

func qBusinessPluginARN(appID, pluginID string) string {
	return qBusinessApplicationARN(appID) + "/plugin/" + qBusinessDefaultString(pluginID, "plugin-000001")
}

func qBusinessRetrieverARN(appID, retrieverID string) string {
	return qBusinessApplicationARN(appID) + "/retriever/" + qBusinessDefaultString(retrieverID, "retriever-000001")
}

func qBusinessUserARN(appID, userID string) string {
	return qBusinessApplicationARN(appID) + "/user/" + qBusinessDefaultString(userID, "user-000001")
}

func qBusinessWebExperienceARN(appID, webExperienceID string) string {
	return qBusinessApplicationARN(appID) + "/webexperience/" + qBusinessDefaultString(webExperienceID, "webexp-000001")
}

func qBusinessChatResponseConfigurationARN(appID, configID string) string {
	return qBusinessApplicationARN(appID) + "/chatresponseconfiguration/" + qBusinessDefaultString(configID, "crc-000001")
}

func qBusinessSubscriptionARN(appID, subscriptionID string) string {
	return qBusinessApplicationARN(appID) + "/subscription/" + qBusinessDefaultString(subscriptionID, "sub-000001")
}
