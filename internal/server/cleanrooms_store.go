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
	cleanRoomsDefaultRegion    = "us-east-1"
	cleanRoomsDefaultAccountID = "123456789012"
)

type cleanRoomsStore struct {
	mu sync.Mutex

	nextID int64

	resources map[string]map[string]map[string]any
	tags      map[string]map[string]string
}

func newCleanRoomsStore() *cleanRoomsStore {
	now := time.Now().UTC()
	s := &cleanRoomsStore{
		nextID:    2,
		resources: map[string]map[string]map[string]any{},
		tags:      map[string]map[string]string{},
	}
	s.ensureCollaborationLocked("col-000001", now)
	s.ensureMembershipLocked("mem-000001", "col-000001", now)
	s.ensureConfiguredTableLocked("tbl-000001", now)
	s.ensureConfiguredTableAssociationLocked("mem-000001", "cta-000001", "tbl-000001", now)
	s.ensureAnalysisTemplateLocked("mem-000001", "at-000001", now)
	s.ensurePrivacyBudgetTemplateLocked("mem-000001", "pbt-000001", now)
	s.ensureProtectedQueryLocked("mem-000001", "pq-000001", now)
	s.ensureProtectedJobLocked("mem-000001", "pj-000001", now)
	s.ensureChangeRequestLocked("col-000001", "cr-000001", now)
	s.ensureConfiguredAudienceModelAssociationLocked("mem-000001", "ama-000001", now)
	s.ensureIDMappingTableLocked("mem-000001", "imt-000001", now)
	s.ensureIDNamespaceAssociationLocked("mem-000001", "ina-000001", now)
	s.ensureMemberLocked("col-000001", "111122223333", now)
	return s
}

func (s *cleanRoomsStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	query url.Values,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	collaborationID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "collaborationIdentifier"), "col-000001")
	membershipID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "membershipIdentifier"), "mem-000001")
	configuredTableID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "configuredTableIdentifier"), "tbl-000001")
	configuredTableAssociationID := cleanRoomsDefaultIfEmpty(
		cleanRoomsLookupString(pathParams, payload, query, "configuredTableAssociationIdentifier"),
		"cta-000001",
	)
	analysisTemplateID := cleanRoomsDefaultIfEmpty(
		cleanRoomsLookupString(pathParams, payload, query, "analysisTemplateIdentifier"),
		cleanRoomsLastToken(cleanRoomsLookupString(pathParams, payload, query, "analysisTemplateArn"), "at-000001"),
	)
	protectedQueryID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "protectedQueryIdentifier"), "pq-000001")
	protectedJobID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "protectedJobIdentifier"), "pj-000001")
	changeRequestID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "changeRequestIdentifier"), "cr-000001")
	privacyBudgetTemplateID := cleanRoomsDefaultIfEmpty(
		cleanRoomsLookupString(pathParams, payload, query, "privacyBudgetTemplateIdentifier"),
		"pbt-000001",
	)
	configuredAudienceModelAssociationID := cleanRoomsDefaultIfEmpty(
		cleanRoomsLookupString(pathParams, payload, query, "configuredAudienceModelAssociationIdentifier"),
		"ama-000001",
	)
	idMappingTableID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "idMappingTableIdentifier"), "imt-000001")
	idNamespaceAssociationID := cleanRoomsDefaultIfEmpty(
		cleanRoomsLookupString(pathParams, payload, query, "idNamespaceAssociationIdentifier"),
		"ina-000001",
	)
	analysisRuleType := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "analysisRuleType"), "AGGREGATION")
	schemaName := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "name"), "stackyard_schema")
	schemaType := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "type"), "AGGREGATION")
	accountID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "accountId"), "111122223333")
	resourceARN := cleanRoomsLookupString(pathParams, payload, query, "resourceArn")
	if resourceARN == "" {
		resourceARN = cleanRoomsCollaborationARN(collaborationID)
	}

	switch action {
	case "CreateCollaboration":
		id := cleanRoomsLookupString(pathParams, payload, query, "collaborationIdentifier")
		if id == "" {
			id = s.nextIDLocked("col")
		}
		item := s.ensureCollaborationLocked(id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"collaboration": cleanRoomsCloneMap(item)}
	case "GetCollaboration":
		item := s.ensureCollaborationLocked(collaborationID, now)
		return map[string]any{"collaboration": cleanRoomsCloneMap(item)}
	case "ListCollaborations":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("collaboration"),
			[]string{"id", "arn", "name", "status", "createTime"},
			"collaborationIdentifier",
		)
		return map[string]any{"collaborationList": items, "nextToken": ""}
	case "UpdateCollaboration":
		item := s.ensureCollaborationLocked(collaborationID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"collaboration": cleanRoomsCloneMap(item)}
	case "DeleteCollaboration":
		delete(s.bucketLocked("collaboration"), collaborationID)
		delete(s.tags, cleanRoomsCollaborationARN(collaborationID))
		return map[string]any{}

	case "CreateMembership":
		id := cleanRoomsLookupString(pathParams, payload, query, "membershipIdentifier")
		if id == "" {
			id = s.nextIDLocked("mem")
		}
		collabID := cleanRoomsDefaultIfEmpty(
			cleanRoomsLookupString(pathParams, payload, query, "collaborationIdentifier"),
			collaborationID,
		)
		s.ensureCollaborationLocked(collabID, now)
		item := s.ensureMembershipLocked(id, collabID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"membership": cleanRoomsCloneMap(item)}
	case "GetMembership":
		item := s.ensureMembershipLocked(membershipID, collaborationID, now)
		return map[string]any{"membership": cleanRoomsCloneMap(item)}
	case "ListMemberships":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("membership"),
			[]string{"id", "arn", "status", "collaborationIdentifier", "createTime"},
			"membershipIdentifier",
		)
		return map[string]any{"membershipSummaries": items, "nextToken": ""}
	case "UpdateMembership":
		item := s.ensureMembershipLocked(membershipID, collaborationID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"membership": cleanRoomsCloneMap(item)}
	case "DeleteMembership":
		delete(s.bucketLocked("membership"), membershipID)
		delete(s.tags, cleanRoomsMembershipARN(membershipID))
		return map[string]any{}

	case "CreateConfiguredTable":
		id := cleanRoomsLookupString(pathParams, payload, query, "configuredTableIdentifier")
		if id == "" {
			id = s.nextIDLocked("tbl")
		}
		item := s.ensureConfiguredTableLocked(id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredTable": cleanRoomsCloneMap(item)}
	case "GetConfiguredTable":
		item := s.ensureConfiguredTableLocked(configuredTableID, now)
		return map[string]any{"configuredTable": cleanRoomsCloneMap(item)}
	case "ListConfiguredTables":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("configuredTable"),
			[]string{"id", "arn", "name", "createTime"},
			"configuredTableIdentifier",
		)
		return map[string]any{"configuredTableSummaries": items, "nextToken": ""}
	case "UpdateConfiguredTable":
		item := s.ensureConfiguredTableLocked(configuredTableID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredTable": cleanRoomsCloneMap(item)}
	case "DeleteConfiguredTable":
		delete(s.bucketLocked("configuredTable"), configuredTableID)
		delete(s.tags, cleanRoomsConfiguredTableARN(configuredTableID))
		return map[string]any{}

	case "CreateConfiguredTableAnalysisRule", "GetConfiguredTableAnalysisRule", "UpdateConfiguredTableAnalysisRule":
		return map[string]any{
			"analysisRule": map[string]any{
				"configuredTableIdentifier": configuredTableID,
				"type":                      analysisRuleType,
				"policy":                    map[string]any{},
			},
		}
	case "DeleteConfiguredTableAnalysisRule":
		return map[string]any{}

	case "CreateConfiguredTableAssociation":
		id := cleanRoomsLookupString(pathParams, payload, query, "configuredTableAssociationIdentifier")
		if id == "" {
			id = s.nextIDLocked("cta")
		}
		tableID := cleanRoomsDefaultIfEmpty(cleanRoomsLookupString(pathParams, payload, query, "configuredTableIdentifier"), configuredTableID)
		s.ensureConfiguredTableLocked(tableID, now)
		item := s.ensureConfiguredTableAssociationLocked(membershipID, id, tableID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredTableAssociation": cleanRoomsCloneMap(item)}
	case "GetConfiguredTableAssociation":
		item := s.ensureConfiguredTableAssociationLocked(membershipID, configuredTableAssociationID, configuredTableID, now)
		return map[string]any{"configuredTableAssociation": cleanRoomsCloneMap(item)}
	case "ListConfiguredTableAssociations":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("configuredTableAssociation", membershipID),
			[]string{"configuredTableAssociationIdentifier", "membershipIdentifier", "configuredTableIdentifier", "createTime"},
			"configuredTableAssociationIdentifier",
		)
		return map[string]any{"configuredTableAssociationSummaries": items, "nextToken": ""}
	case "UpdateConfiguredTableAssociation":
		item := s.ensureConfiguredTableAssociationLocked(membershipID, configuredTableAssociationID, configuredTableID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredTableAssociation": cleanRoomsCloneMap(item)}
	case "DeleteConfiguredTableAssociation":
		delete(s.bucketLocked("configuredTableAssociation"), cleanRoomsScopedKey(membershipID, configuredTableAssociationID))
		return map[string]any{}

	case "CreateConfiguredTableAssociationAnalysisRule",
		"GetConfiguredTableAssociationAnalysisRule",
		"UpdateConfiguredTableAssociationAnalysisRule":
		return map[string]any{
			"analysisRule": map[string]any{
				"membershipIdentifier":                 membershipID,
				"configuredTableAssociationIdentifier": configuredTableAssociationID,
				"type":                                 analysisRuleType,
				"policy":                               map[string]any{},
			},
		}
	case "DeleteConfiguredTableAssociationAnalysisRule":
		return map[string]any{}

	case "CreateAnalysisTemplate":
		id := cleanRoomsLookupString(pathParams, payload, query, "analysisTemplateIdentifier")
		if id == "" {
			id = s.nextIDLocked("at")
		}
		item := s.ensureAnalysisTemplateLocked(membershipID, id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"analysisTemplate": cleanRoomsCloneMap(item)}
	case "GetAnalysisTemplate":
		item := s.ensureAnalysisTemplateLocked(membershipID, analysisTemplateID, now)
		return map[string]any{"analysisTemplate": cleanRoomsCloneMap(item)}
	case "ListAnalysisTemplates":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("analysisTemplate", membershipID),
			[]string{"analysisTemplateIdentifier", "membershipIdentifier", "arn", "status", "createTime"},
			"analysisTemplateIdentifier",
		)
		return map[string]any{"analysisTemplateSummaries": items, "nextToken": ""}
	case "UpdateAnalysisTemplate":
		item := s.ensureAnalysisTemplateLocked(membershipID, analysisTemplateID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"analysisTemplate": cleanRoomsCloneMap(item)}
	case "DeleteAnalysisTemplate":
		delete(s.bucketLocked("analysisTemplate"), cleanRoomsScopedKey(membershipID, analysisTemplateID))
		return map[string]any{}
	case "GetCollaborationAnalysisTemplate":
		item := s.ensureAnalysisTemplateLocked(membershipID, analysisTemplateID, now)
		return map[string]any{"collaborationAnalysisTemplate": cleanRoomsCloneMap(item)}
	case "ListCollaborationAnalysisTemplates":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("analysisTemplate"),
			[]string{"analysisTemplateIdentifier", "membershipIdentifier", "arn", "status"},
			"analysisTemplateIdentifier",
		)
		return map[string]any{"collaborationAnalysisTemplateSummaries": items, "nextToken": ""}
	case "BatchGetCollaborationAnalysisTemplate":
		item := s.ensureAnalysisTemplateLocked(membershipID, analysisTemplateID, now)
		return map[string]any{
			"collaborationAnalysisTemplates": []any{cleanRoomsCloneMap(item)},
			"errors":                         []any{},
		}

	case "CreatePrivacyBudgetTemplate":
		id := cleanRoomsLookupString(pathParams, payload, query, "privacyBudgetTemplateIdentifier")
		if id == "" {
			id = s.nextIDLocked("pbt")
		}
		item := s.ensurePrivacyBudgetTemplateLocked(membershipID, id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"privacyBudgetTemplate": cleanRoomsCloneMap(item)}
	case "GetPrivacyBudgetTemplate":
		item := s.ensurePrivacyBudgetTemplateLocked(membershipID, privacyBudgetTemplateID, now)
		return map[string]any{"privacyBudgetTemplate": cleanRoomsCloneMap(item)}
	case "ListPrivacyBudgetTemplates":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("privacyBudgetTemplate", membershipID),
			[]string{"privacyBudgetTemplateIdentifier", "membershipIdentifier", "arn", "status"},
			"privacyBudgetTemplateIdentifier",
		)
		return map[string]any{"privacyBudgetTemplateSummaries": items, "nextToken": ""}
	case "UpdatePrivacyBudgetTemplate":
		item := s.ensurePrivacyBudgetTemplateLocked(membershipID, privacyBudgetTemplateID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"privacyBudgetTemplate": cleanRoomsCloneMap(item)}
	case "DeletePrivacyBudgetTemplate":
		delete(s.bucketLocked("privacyBudgetTemplate"), cleanRoomsScopedKey(membershipID, privacyBudgetTemplateID))
		return map[string]any{}
	case "GetCollaborationPrivacyBudgetTemplate":
		item := s.ensurePrivacyBudgetTemplateLocked(membershipID, privacyBudgetTemplateID, now)
		return map[string]any{"collaborationPrivacyBudgetTemplate": cleanRoomsCloneMap(item)}
	case "ListCollaborationPrivacyBudgetTemplates":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("privacyBudgetTemplate"),
			[]string{"privacyBudgetTemplateIdentifier", "membershipIdentifier", "arn", "status"},
			"privacyBudgetTemplateIdentifier",
		)
		return map[string]any{"collaborationPrivacyBudgetTemplateSummaries": items, "nextToken": ""}
	case "ListPrivacyBudgets", "ListCollaborationPrivacyBudgets":
		budgets := []any{
			map[string]any{
				"privacyBudgetType":    "DIFFERENTIAL_PRIVACY",
				"status":               "ACTIVE",
				"membershipIdentifier": membershipID,
			},
		}
		if action == "ListCollaborationPrivacyBudgets" {
			return map[string]any{"collaborationPrivacyBudgetSummaries": budgets, "nextToken": ""}
		}
		return map[string]any{"privacyBudgetSummaries": budgets, "nextToken": ""}

	case "StartProtectedQuery":
		id := cleanRoomsLookupString(pathParams, payload, query, "protectedQueryIdentifier")
		if id == "" {
			id = s.nextIDLocked("pq")
		}
		item := s.ensureProtectedQueryLocked(membershipID, id, now)
		item["status"] = "STARTED"
		item["updateTime"] = cleanRoomsTime(now)
		return map[string]any{"protectedQuery": cleanRoomsCloneMap(item)}
	case "GetProtectedQuery":
		item := s.ensureProtectedQueryLocked(membershipID, protectedQueryID, now)
		return map[string]any{"protectedQuery": cleanRoomsCloneMap(item)}
	case "ListProtectedQueries":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("protectedQuery", membershipID),
			[]string{"protectedQueryIdentifier", "membershipIdentifier", "status", "createTime"},
			"protectedQueryIdentifier",
		)
		return map[string]any{"protectedQueries": items, "nextToken": ""}
	case "UpdateProtectedQuery":
		item := s.ensureProtectedQueryLocked(membershipID, protectedQueryID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"protectedQuery": cleanRoomsCloneMap(item)}

	case "StartProtectedJob":
		id := cleanRoomsLookupString(pathParams, payload, query, "protectedJobIdentifier")
		if id == "" {
			id = s.nextIDLocked("pj")
		}
		item := s.ensureProtectedJobLocked(membershipID, id, now)
		item["status"] = "STARTED"
		item["updateTime"] = cleanRoomsTime(now)
		return map[string]any{"protectedJob": cleanRoomsCloneMap(item)}
	case "GetProtectedJob":
		item := s.ensureProtectedJobLocked(membershipID, protectedJobID, now)
		return map[string]any{"protectedJob": cleanRoomsCloneMap(item)}
	case "ListProtectedJobs":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("protectedJob", membershipID),
			[]string{"protectedJobIdentifier", "membershipIdentifier", "status", "createTime"},
			"protectedJobIdentifier",
		)
		return map[string]any{"protectedJobs": items, "nextToken": ""}
	case "UpdateProtectedJob":
		item := s.ensureProtectedJobLocked(membershipID, protectedJobID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"protectedJob": cleanRoomsCloneMap(item)}

	case "CreateCollaborationChangeRequest":
		id := cleanRoomsLookupString(pathParams, payload, query, "changeRequestIdentifier")
		if id == "" {
			id = s.nextIDLocked("cr")
		}
		item := s.ensureChangeRequestLocked(collaborationID, id, now)
		item["status"] = "PENDING"
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"collaborationChangeRequest": cleanRoomsCloneMap(item)}
	case "GetCollaborationChangeRequest":
		item := s.ensureChangeRequestLocked(collaborationID, changeRequestID, now)
		return map[string]any{"collaborationChangeRequest": cleanRoomsCloneMap(item)}
	case "ListCollaborationChangeRequests":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("changeRequest", collaborationID),
			[]string{"changeRequestIdentifier", "collaborationIdentifier", "status", "createTime"},
			"changeRequestIdentifier",
		)
		return map[string]any{"collaborationChangeRequests": items, "nextToken": ""}
	case "UpdateCollaborationChangeRequest":
		item := s.ensureChangeRequestLocked(collaborationID, changeRequestID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"collaborationChangeRequest": cleanRoomsCloneMap(item)}

	case "CreateConfiguredAudienceModelAssociation":
		id := cleanRoomsLookupString(pathParams, payload, query, "configuredAudienceModelAssociationIdentifier")
		if id == "" {
			id = s.nextIDLocked("ama")
		}
		item := s.ensureConfiguredAudienceModelAssociationLocked(membershipID, id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredAudienceModelAssociation": cleanRoomsCloneMap(item)}
	case "GetConfiguredAudienceModelAssociation":
		item := s.ensureConfiguredAudienceModelAssociationLocked(membershipID, configuredAudienceModelAssociationID, now)
		return map[string]any{"configuredAudienceModelAssociation": cleanRoomsCloneMap(item)}
	case "ListConfiguredAudienceModelAssociations":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("audienceModelAssociation", membershipID),
			[]string{"configuredAudienceModelAssociationIdentifier", "membershipIdentifier", "status", "createTime"},
			"configuredAudienceModelAssociationIdentifier",
		)
		return map[string]any{"configuredAudienceModelAssociationSummaries": items, "nextToken": ""}
	case "UpdateConfiguredAudienceModelAssociation":
		item := s.ensureConfiguredAudienceModelAssociationLocked(membershipID, configuredAudienceModelAssociationID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"configuredAudienceModelAssociation": cleanRoomsCloneMap(item)}
	case "DeleteConfiguredAudienceModelAssociation":
		delete(s.bucketLocked("audienceModelAssociation"), cleanRoomsScopedKey(membershipID, configuredAudienceModelAssociationID))
		return map[string]any{}
	case "GetCollaborationConfiguredAudienceModelAssociation":
		item := s.ensureConfiguredAudienceModelAssociationLocked(membershipID, configuredAudienceModelAssociationID, now)
		return map[string]any{"collaborationConfiguredAudienceModelAssociation": cleanRoomsCloneMap(item)}
	case "ListCollaborationConfiguredAudienceModelAssociations":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("audienceModelAssociation"),
			[]string{"configuredAudienceModelAssociationIdentifier", "membershipIdentifier", "status", "createTime"},
			"configuredAudienceModelAssociationIdentifier",
		)
		return map[string]any{"collaborationConfiguredAudienceModelAssociationSummaries": items, "nextToken": ""}

	case "CreateIdMappingTable":
		id := cleanRoomsLookupString(pathParams, payload, query, "idMappingTableIdentifier")
		if id == "" {
			id = s.nextIDLocked("imt")
		}
		item := s.ensureIDMappingTableLocked(membershipID, id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"idMappingTable": cleanRoomsCloneMap(item)}
	case "PopulateIdMappingTable":
		item := s.ensureIDMappingTableLocked(membershipID, idMappingTableID, now)
		item["status"] = "ACTIVE"
		item["updateTime"] = cleanRoomsTime(now)
		return map[string]any{"idMappingTable": cleanRoomsCloneMap(item)}
	case "GetIdMappingTable":
		item := s.ensureIDMappingTableLocked(membershipID, idMappingTableID, now)
		return map[string]any{"idMappingTable": cleanRoomsCloneMap(item)}
	case "ListIdMappingTables":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("idMappingTable", membershipID),
			[]string{"idMappingTableIdentifier", "membershipIdentifier", "status", "createTime"},
			"idMappingTableIdentifier",
		)
		return map[string]any{"idMappingTableSummaries": items, "nextToken": ""}
	case "UpdateIdMappingTable":
		item := s.ensureIDMappingTableLocked(membershipID, idMappingTableID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"idMappingTable": cleanRoomsCloneMap(item)}
	case "DeleteIdMappingTable":
		delete(s.bucketLocked("idMappingTable"), cleanRoomsScopedKey(membershipID, idMappingTableID))
		return map[string]any{}

	case "CreateIdNamespaceAssociation":
		id := cleanRoomsLookupString(pathParams, payload, query, "idNamespaceAssociationIdentifier")
		if id == "" {
			id = s.nextIDLocked("ina")
		}
		item := s.ensureIDNamespaceAssociationLocked(membershipID, id, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"idNamespaceAssociation": cleanRoomsCloneMap(item)}
	case "GetIdNamespaceAssociation":
		item := s.ensureIDNamespaceAssociationLocked(membershipID, idNamespaceAssociationID, now)
		return map[string]any{"idNamespaceAssociation": cleanRoomsCloneMap(item)}
	case "ListIdNamespaceAssociations":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("idNamespaceAssociation", membershipID),
			[]string{"idNamespaceAssociationIdentifier", "membershipIdentifier", "status", "createTime"},
			"idNamespaceAssociationIdentifier",
		)
		return map[string]any{"idNamespaceAssociationSummaries": items, "nextToken": ""}
	case "UpdateIdNamespaceAssociation":
		item := s.ensureIDNamespaceAssociationLocked(membershipID, idNamespaceAssociationID, now)
		cleanRoomsApplyMutableFields(item, payload, now)
		return map[string]any{"idNamespaceAssociation": cleanRoomsCloneMap(item)}
	case "DeleteIdNamespaceAssociation":
		delete(s.bucketLocked("idNamespaceAssociation"), cleanRoomsScopedKey(membershipID, idNamespaceAssociationID))
		return map[string]any{}
	case "GetCollaborationIdNamespaceAssociation":
		item := s.ensureIDNamespaceAssociationLocked(membershipID, idNamespaceAssociationID, now)
		return map[string]any{"collaborationIdNamespaceAssociation": cleanRoomsCloneMap(item)}
	case "ListCollaborationIdNamespaceAssociations":
		items := cleanRoomsSummaries(
			s.sortedBucketValuesLocked("idNamespaceAssociation"),
			[]string{"idNamespaceAssociationIdentifier", "membershipIdentifier", "status", "createTime"},
			"idNamespaceAssociationIdentifier",
		)
		return map[string]any{"collaborationIdNamespaceAssociationSummaries": items, "nextToken": ""}

	case "BatchGetSchema":
		names := cleanRoomsLookupStringSlice(payload, "names")
		if len(names) == 0 {
			names = []string{schemaName}
		}
		schemas := make([]any, 0, len(names))
		for _, name := range names {
			schemas = append(schemas, map[string]any{
				"name":       name,
				"type":       "TABLE",
				"createTime": cleanRoomsTime(now),
			})
		}
		return map[string]any{"schemas": schemas, "errors": []any{}}
	case "GetSchema":
		return map[string]any{
			"schema": map[string]any{
				"name":              schemaName,
				"type":              "TABLE",
				"analysisRuleTypes": []any{"AGGREGATION", "LIST"},
			},
		}
	case "ListSchemas":
		return map[string]any{
			"schemaSummaries": []any{
				map[string]any{"name": schemaName, "type": "TABLE"},
			},
			"nextToken": "",
		}
	case "BatchGetSchemaAnalysisRule":
		names := cleanRoomsLookupStringSlice(payload, "names")
		if len(names) == 0 {
			names = []string{schemaName}
		}
		rules := make([]any, 0, len(names))
		for _, name := range names {
			rules = append(rules, map[string]any{
				"name": name,
				"type": schemaType,
				"policy": map[string]any{
					"v1": map[string]any{},
				},
			})
		}
		return map[string]any{"schemaAnalysisRules": rules, "errors": []any{}}
	case "GetSchemaAnalysisRule":
		return map[string]any{
			"analysisRule": map[string]any{
				"name":   schemaName,
				"type":   schemaType,
				"policy": map[string]any{"v1": map[string]any{}},
			},
		}

	case "ListMembers":
		items := cleanRoomsSummaries(
			s.sortedScopedBucketValuesLocked("member", collaborationID),
			[]string{"accountId", "memberStatus", "displayName"},
			"accountId",
		)
		return map[string]any{"memberSummaries": items, "nextToken": ""}
	case "DeleteMember":
		delete(s.bucketLocked("member"), cleanRoomsScopedKey(collaborationID, accountID))
		return map[string]any{}

	case "PreviewPrivacyImpact":
		return map[string]any{
			"privacyImpact": map[string]any{
				"membershipIdentifier": membershipID,
				"status":               "ESTIMATED",
				"privacyBudgetType":    "DIFFERENTIAL_PRIVACY",
			},
		}

	case "TagResource":
		s.upsertTagsLocked(resourceARN, cleanRoomsExtractTags(payload))
		return map[string]any{}
	case "UntagResource":
		s.removeTagsLocked(resourceARN, cleanRoomsTagKeys(payload, query))
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": cleanRoomsCloneStringMap(s.tags[resourceARN])}
	}

	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"nextToken": ""}
	}
	return map[string]any{}
}

func (s *cleanRoomsStore) nextIDLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func (s *cleanRoomsStore) ensureCollaborationLocked(id string, now time.Time) map[string]any {
	b := s.bucketLocked("collaboration")
	if item := b[id]; item != nil {
		return item
	}
	item := map[string]any{
		"id":                      id,
		"collaborationIdentifier": id,
		"arn":                     cleanRoomsCollaborationARN(id),
		"name":                    "stackyard-collaboration",
		"description":             "",
		"creatorAccountId":        cleanRoomsDefaultAccountID,
		"status":                  "ACTIVE",
		"createTime":              cleanRoomsTime(now),
		"updateTime":              cleanRoomsTime(now),
	}
	b[id] = item
	s.ensureTagsLocked(cleanRoomsCollaborationARN(id))
	return item
}

func (s *cleanRoomsStore) ensureMembershipLocked(id, collaborationID string, now time.Time) map[string]any {
	b := s.bucketLocked("membership")
	if item := b[id]; item != nil {
		return item
	}
	item := map[string]any{
		"id":                      id,
		"membershipIdentifier":    id,
		"collaborationIdentifier": collaborationID,
		"arn":                     cleanRoomsMembershipARN(id),
		"status":                  "ACTIVE",
		"queryLogStatus":          "ENABLED",
		"createTime":              cleanRoomsTime(now),
		"updateTime":              cleanRoomsTime(now),
	}
	b[id] = item
	s.ensureTagsLocked(cleanRoomsMembershipARN(id))
	return item
}

func (s *cleanRoomsStore) ensureConfiguredTableLocked(id string, now time.Time) map[string]any {
	b := s.bucketLocked("configuredTable")
	if item := b[id]; item != nil {
		return item
	}
	item := map[string]any{
		"id":                        id,
		"configuredTableIdentifier": id,
		"arn":                       cleanRoomsConfiguredTableARN(id),
		"name":                      "stackyard-configured-table",
		"description":               "",
		"createTime":                cleanRoomsTime(now),
		"updateTime":                cleanRoomsTime(now),
		"tableReference": map[string]any{
			"glue": map[string]any{
				"databaseName": "default",
				"tableName":    "stackyard_table",
			},
		},
	}
	b[id] = item
	s.ensureTagsLocked(cleanRoomsConfiguredTableARN(id))
	return item
}

func (s *cleanRoomsStore) ensureConfiguredTableAssociationLocked(
	membershipID, associationID, configuredTableID string,
	now time.Time,
) map[string]any {
	key := cleanRoomsScopedKey(membershipID, associationID)
	b := s.bucketLocked("configuredTableAssociation")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"configuredTableAssociationIdentifier": associationID,
		"membershipIdentifier":                 membershipID,
		"configuredTableIdentifier":            configuredTableID,
		"name":                                 "stackyard-configured-table-association",
		"description":                          "",
		"createTime":                           cleanRoomsTime(now),
		"updateTime":                           cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureAnalysisTemplateLocked(membershipID, templateID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, templateID)
	b := s.bucketLocked("analysisTemplate")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"analysisTemplateIdentifier": templateID,
		"membershipIdentifier":       membershipID,
		"arn":                        cleanRoomsAnalysisTemplateARN(membershipID, templateID),
		"name":                       "stackyard-analysis-template",
		"description":                "",
		"status":                     "ACTIVE",
		"createTime":                 cleanRoomsTime(now),
		"updateTime":                 cleanRoomsTime(now),
	}
	b[key] = item
	s.ensureTagsLocked(cleanRoomsAnalysisTemplateARN(membershipID, templateID))
	return item
}

func (s *cleanRoomsStore) ensurePrivacyBudgetTemplateLocked(membershipID, templateID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, templateID)
	b := s.bucketLocked("privacyBudgetTemplate")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"privacyBudgetTemplateIdentifier": templateID,
		"membershipIdentifier":            membershipID,
		"arn":                             cleanRoomsPrivacyBudgetTemplateARN(membershipID, templateID),
		"name":                            "stackyard-privacy-budget-template",
		"description":                     "",
		"status":                          "ACTIVE",
		"createTime":                      cleanRoomsTime(now),
		"updateTime":                      cleanRoomsTime(now),
	}
	b[key] = item
	s.ensureTagsLocked(cleanRoomsPrivacyBudgetTemplateARN(membershipID, templateID))
	return item
}

func (s *cleanRoomsStore) ensureProtectedQueryLocked(membershipID, queryID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, queryID)
	b := s.bucketLocked("protectedQuery")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"protectedQueryIdentifier": queryID,
		"membershipIdentifier":     membershipID,
		"status":                   "SUBMITTED",
		"createTime":               cleanRoomsTime(now),
		"updateTime":               cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureProtectedJobLocked(membershipID, jobID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, jobID)
	b := s.bucketLocked("protectedJob")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"protectedJobIdentifier": jobID,
		"membershipIdentifier":   membershipID,
		"status":                 "SUBMITTED",
		"createTime":             cleanRoomsTime(now),
		"updateTime":             cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureChangeRequestLocked(collaborationID, changeRequestID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(collaborationID, changeRequestID)
	b := s.bucketLocked("changeRequest")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"changeRequestIdentifier": changeRequestID,
		"collaborationIdentifier": collaborationID,
		"status":                  "PENDING",
		"createTime":              cleanRoomsTime(now),
		"updateTime":              cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureConfiguredAudienceModelAssociationLocked(
	membershipID, associationID string,
	now time.Time,
) map[string]any {
	key := cleanRoomsScopedKey(membershipID, associationID)
	b := s.bucketLocked("audienceModelAssociation")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"configuredAudienceModelAssociationIdentifier": associationID,
		"membershipIdentifier":                         membershipID,
		"status":                                       "ACTIVE",
		"createTime":                                   cleanRoomsTime(now),
		"updateTime":                                   cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureIDMappingTableLocked(membershipID, tableID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, tableID)
	b := s.bucketLocked("idMappingTable")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"idMappingTableIdentifier": tableID,
		"membershipIdentifier":     membershipID,
		"status":                   "CREATED",
		"createTime":               cleanRoomsTime(now),
		"updateTime":               cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureIDNamespaceAssociationLocked(membershipID, associationID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(membershipID, associationID)
	b := s.bucketLocked("idNamespaceAssociation")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"idNamespaceAssociationIdentifier": associationID,
		"membershipIdentifier":             membershipID,
		"status":                           "ACTIVE",
		"createTime":                       cleanRoomsTime(now),
		"updateTime":                       cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) ensureMemberLocked(collaborationID, accountID string, now time.Time) map[string]any {
	key := cleanRoomsScopedKey(collaborationID, accountID)
	b := s.bucketLocked("member")
	if item := b[key]; item != nil {
		return item
	}
	item := map[string]any{
		"collaborationIdentifier": collaborationID,
		"accountId":               accountID,
		"displayName":             "stackyard-member",
		"memberStatus":            "ACTIVE",
		"createTime":              cleanRoomsTime(now),
		"updateTime":              cleanRoomsTime(now),
	}
	b[key] = item
	return item
}

func (s *cleanRoomsStore) bucketLocked(kind string) map[string]map[string]any {
	b := s.resources[kind]
	if b == nil {
		b = map[string]map[string]any{}
		s.resources[kind] = b
	}
	return b
}

func (s *cleanRoomsStore) sortedBucketValuesLocked(kind string) []map[string]any {
	return cleanRoomsSortedValues(s.bucketLocked(kind))
}

func (s *cleanRoomsStore) sortedScopedBucketValuesLocked(kind, scope string) []map[string]any {
	values := make([]map[string]any, 0)
	prefix := cleanRoomsScopedKey(scope)
	for key, item := range s.bucketLocked(kind) {
		if strings.HasPrefix(key, prefix) {
			values = append(values, item)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return cleanRoomsStableID(values[i]) < cleanRoomsStableID(values[j])
	})
	return values
}

func (s *cleanRoomsStore) ensureTagsLocked(resourceARN string) map[string]string {
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func (s *cleanRoomsStore) upsertTagsLocked(resourceARN string, incoming map[string]string) {
	tags := s.ensureTagsLocked(resourceARN)
	for k, v := range incoming {
		if strings.TrimSpace(k) == "" {
			continue
		}
		tags[k] = v
	}
}

func (s *cleanRoomsStore) removeTagsLocked(resourceARN string, keys []string) {
	tags := s.ensureTagsLocked(resourceARN)
	for _, key := range keys {
		delete(tags, key)
	}
}

func cleanRoomsLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		for k, v := range pathParams {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				if out := strings.TrimSpace(v); out != "" {
					return out
				}
			}
		}
		for k, v := range payload {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				if out := strings.TrimSpace(cleanRoomsAnyToString(v)); out != "" {
					return out
				}
			}
		}
		for qk, values := range query {
			if !strings.EqualFold(strings.TrimSpace(qk), key) {
				continue
			}
			for _, qv := range values {
				if out := strings.TrimSpace(qv); out != "" {
					return out
				}
			}
		}
	}
	return ""
}

func cleanRoomsLookupStringSlice(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		for pk, raw := range payload {
			if !strings.EqualFold(strings.TrimSpace(pk), key) {
				continue
			}
			switch v := raw.(type) {
			case []any:
				out := make([]string, 0, len(v))
				for _, item := range v {
					if s := strings.TrimSpace(cleanRoomsAnyToString(item)); s != "" {
						out = append(out, s)
					}
				}
				return out
			case []string:
				out := make([]string, 0, len(v))
				for _, item := range v {
					if s := strings.TrimSpace(item); s != "" {
						out = append(out, s)
					}
				}
				return out
			case string:
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					return []string{trimmed}
				}
			}
		}
	}
	return nil
}

func cleanRoomsApplyMutableFields(item map[string]any, payload map[string]any, now time.Time) {
	for key, value := range payload {
		switch {
		case strings.EqualFold(key, "name"):
			if str := strings.TrimSpace(cleanRoomsAnyToString(value)); str != "" {
				item["name"] = str
			}
		case strings.EqualFold(key, "description"):
			item["description"] = cleanRoomsAnyToString(value)
		case strings.EqualFold(key, "status"):
			if str := strings.TrimSpace(cleanRoomsAnyToString(value)); str != "" {
				item["status"] = str
			}
		}
	}
	item["updateTime"] = cleanRoomsTime(now)
}

func cleanRoomsExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for key, raw := range payload {
		if !strings.EqualFold(strings.TrimSpace(key), "tags") {
			continue
		}
		switch v := raw.(type) {
		case map[string]any:
			for tagKey, tagVal := range v {
				out[strings.TrimSpace(tagKey)] = cleanRoomsAnyToString(tagVal)
			}
		case map[string]string:
			for tagKey, tagVal := range v {
				out[strings.TrimSpace(tagKey)] = strings.TrimSpace(tagVal)
			}
		case []any:
			for _, entry := range v {
				m, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				tagKey := strings.TrimSpace(cleanRoomsLookupNestedString(m, "key", "Key", "tagKey"))
				if tagKey == "" {
					continue
				}
				out[tagKey] = cleanRoomsLookupNestedString(m, "value", "Value", "tagValue")
			}
		}
	}
	return out
}

func cleanRoomsTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	for key, raw := range payload {
		if !strings.EqualFold(strings.TrimSpace(key), "tagKeys") {
			continue
		}
		switch v := raw.(type) {
		case string:
			for _, part := range strings.Split(v, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					keys = append(keys, trimmed)
				}
			}
		case []any:
			for _, entry := range v {
				if trimmed := strings.TrimSpace(cleanRoomsAnyToString(entry)); trimmed != "" {
					keys = append(keys, trimmed)
				}
			}
		case []string:
			for _, entry := range v {
				if trimmed := strings.TrimSpace(entry); trimmed != "" {
					keys = append(keys, trimmed)
				}
			}
		}
	}
	for queryKey, vals := range query {
		if !strings.EqualFold(strings.TrimSpace(queryKey), "tagKeys") {
			continue
		}
		for _, value := range vals {
			for _, part := range strings.Split(value, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					keys = append(keys, trimmed)
				}
			}
		}
	}
	return keys
}

func cleanRoomsLookupNestedString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		for mk, mv := range m {
			if strings.EqualFold(strings.TrimSpace(mk), key) {
				return strings.TrimSpace(cleanRoomsAnyToString(mv))
			}
		}
	}
	return ""
}

func cleanRoomsAnyToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func cleanRoomsSummaries(items []map[string]any, fields []string, idField string) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		summary := map[string]any{}
		for _, field := range fields {
			for key, value := range item {
				if strings.EqualFold(strings.TrimSpace(key), field) {
					summary[field] = cleanRoomsCloneAny(value)
				}
			}
		}
		if summary[idField] == nil {
			if v := cleanRoomsStableID(item); v != "" {
				summary[idField] = v
			}
		}
		out = append(out, summary)
	}
	return out
}

func cleanRoomsSortedValues(bucket map[string]map[string]any) []map[string]any {
	values := make([]map[string]any, 0, len(bucket))
	for _, item := range bucket {
		values = append(values, item)
	}
	sort.Slice(values, func(i, j int) bool {
		return cleanRoomsStableID(values[i]) < cleanRoomsStableID(values[j])
	})
	return values
}

func cleanRoomsStableID(item map[string]any) string {
	candidates := []string{
		cleanRoomsLookupNestedString(item, "id"),
		cleanRoomsLookupNestedString(item, "collaborationIdentifier"),
		cleanRoomsLookupNestedString(item, "membershipIdentifier"),
		cleanRoomsLookupNestedString(item, "configuredTableIdentifier"),
		cleanRoomsLookupNestedString(item, "configuredTableAssociationIdentifier"),
		cleanRoomsLookupNestedString(item, "analysisTemplateIdentifier"),
		cleanRoomsLookupNestedString(item, "privacyBudgetTemplateIdentifier"),
		cleanRoomsLookupNestedString(item, "protectedQueryIdentifier"),
		cleanRoomsLookupNestedString(item, "protectedJobIdentifier"),
		cleanRoomsLookupNestedString(item, "changeRequestIdentifier"),
		cleanRoomsLookupNestedString(item, "configuredAudienceModelAssociationIdentifier"),
		cleanRoomsLookupNestedString(item, "idMappingTableIdentifier"),
		cleanRoomsLookupNestedString(item, "idNamespaceAssociationIdentifier"),
		cleanRoomsLookupNestedString(item, "accountId"),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func cleanRoomsScopedKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			cleaned = append(cleaned, v)
		}
	}
	return strings.Join(cleaned, "|")
}

func cleanRoomsDefaultIfEmpty(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func cleanRoomsLastToken(value, def string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return def
	}
	parts := strings.Split(trimmed, "/")
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return def
	}
	return last
}

func cleanRoomsTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func cleanRoomsCollaborationARN(id string) string {
	return fmt.Sprintf("arn:aws:cleanrooms:%s:%s:collaboration/%s", cleanRoomsDefaultRegion, cleanRoomsDefaultAccountID, id)
}

func cleanRoomsMembershipARN(id string) string {
	return fmt.Sprintf("arn:aws:cleanrooms:%s:%s:membership/%s", cleanRoomsDefaultRegion, cleanRoomsDefaultAccountID, id)
}

func cleanRoomsConfiguredTableARN(id string) string {
	return fmt.Sprintf("arn:aws:cleanrooms:%s:%s:configuredtable/%s", cleanRoomsDefaultRegion, cleanRoomsDefaultAccountID, id)
}

func cleanRoomsAnalysisTemplateARN(membershipID, id string) string {
	return fmt.Sprintf(
		"arn:aws:cleanrooms:%s:%s:membership/%s/analysistemplate/%s",
		cleanRoomsDefaultRegion,
		cleanRoomsDefaultAccountID,
		membershipID,
		id,
	)
}

func cleanRoomsPrivacyBudgetTemplateARN(membershipID, id string) string {
	return fmt.Sprintf(
		"arn:aws:cleanrooms:%s:%s:membership/%s/privacybudgettemplate/%s",
		cleanRoomsDefaultRegion,
		cleanRoomsDefaultAccountID,
		membershipID,
		id,
	)
}

func cleanRoomsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cleanRoomsCloneAny(value)
	}
	return out
}

func cleanRoomsCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cleanRoomsCloneMap(v)
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, vv := range v {
			out[k] = vv
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, cleanRoomsCloneAny(item))
		}
		return out
	case []string:
		out := make([]string, len(v))
		copy(out, v)
		return out
	default:
		return v
	}
}

func cleanRoomsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
