package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type lakeFormationStore struct {
	mu sync.Mutex

	nextID int64

	resources           map[string]map[string]any
	permissions         []map[string]any
	lfTags              map[string]map[string]any
	lfTagExpressions    map[string]map[string]any
	resourceLFTags      map[string][]map[string]any
	dataCellsFilters    map[string]map[string]any
	transactions        map[string]map[string]any
	txTokenToID         map[string]string
	queryRuns           map[string]map[string]any
	tableObjects        map[string][]map[string]any
	storageOptimizers   map[string]map[string]any
	lakeFormationOptIns []map[string]any

	identityCenterConfig map[string]any
	dataLakeSettings     map[string]any
}

func newLakeFormationStore() *lakeFormationStore {
	now := time.Now().UTC().Format(time.RFC3339)
	defaultResourceARN := "arn:aws:s3:::stackyard-lakeformation-data"
	defaultResource := map[string]any{
		"ResourceArn":                defaultResourceARN,
		"RoleArn":                    "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
		"HybridAccessEnabled":        false,
		"WithFederation":             false,
		"WithPrivilegedAccess":       true,
		"LastModified":               now,
		"UseServiceLinkedRole":       true,
		"ResourceShareType":          "EXTERNAL",
		"AllowExternalDataFiltering": false,
	}

	return &lakeFormationStore{
		nextID: 2,

		resources: map[string]map[string]any{
			defaultResourceARN: defaultResource,
		},
		permissions: []map[string]any{
			{
				"Principal": map[string]any{
					"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
				},
				"Resource": map[string]any{
					"DataLocation": map[string]any{
						"ResourceArn": defaultResourceARN,
					},
				},
				"Permissions": []any{"DATA_LOCATION_ACCESS"},
			},
		},
		lfTags: map[string]map[string]any{
			"environment": {
				"TagKey":    "environment",
				"TagValues": []any{"dev", "stage", "prod"},
			},
		},
		lfTagExpressions: map[string]map[string]any{
			"stackyard-expression": {
				"Name": "stackyard-expression",
				"Expression": []any{
					map[string]any{
						"TagKey":    "environment",
						"TagValues": []any{"dev"},
					},
				},
			},
		},
		resourceLFTags: map[string][]map[string]any{
			lakeFormationResourceKey(map[string]any{
				"DataLocation": map[string]any{"ResourceArn": defaultResourceARN},
			}): {
				{ // seeded for deterministic read/list behavior
					"TagKey":    "environment",
					"TagValues": []any{"dev"},
				},
			},
		},
		dataCellsFilters: map[string]map[string]any{
			"default_db|default_table|stackyard-filter": {
				"TableCatalogId": "123456789012",
				"DatabaseName":   "default_db",
				"TableName":      "default_table",
				"Name":           "stackyard-filter",
				"VersionId":      "1",
				"RowFilter": map[string]any{
					"FilterExpression": "TRUE",
				},
			},
		},
		transactions: map[string]map[string]any{
			"tx-000001": {
				"TransactionId":     "tx-000001",
				"TransactionStatus": "COMMITTED",
				"Description":       "stackyard-seeded",
				"StartTime":         now,
			},
		},
		txTokenToID: map[string]string{},
		queryRuns: map[string]map[string]any{
			"query-000001": {
				"QueryId": "query-000001",
				"State":   "FINISHED",
			},
		},
		tableObjects: map[string][]map[string]any{
			"default_db|default_table": {
				{
					"Uri":             "s3://stackyard-lakeformation-data/default/default-000001.parquet",
					"ETag":            "etag-000001",
					"Size":            1024,
					"PartitionValues": []any{"2026", "02", "28"},
					"LastModified":    now,
				},
			},
		},
		storageOptimizers: map[string]map[string]any{
			"default_db|default_table|COMPACTION": {
				"DatabaseName":         "default_db",
				"TableName":            "default_table",
				"StorageOptimizerType": "COMPACTION",
				"Enabled":              true,
				"LastUpdatedAt":        now,
			},
		},
		lakeFormationOptIns: []map[string]any{
			{
				"Principal": map[string]any{
					"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
				},
				"Resource": map[string]any{
					"DataLocation": map[string]any{
						"ResourceArn": defaultResourceARN,
					},
				},
				"LastModified": now,
			},
		},
		identityCenterConfig: map[string]any{
			"ApplicationStatus": "ENABLED",
			"CatalogId":         "123456789012",
			"LastUpdatedTime":   now,
		},
		dataLakeSettings: map[string]any{
			"DataLakeAdmins": []any{
				map[string]any{
					"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
				},
			},
			"CreateDatabaseDefaultPermissions": []any{},
			"CreateTableDefaultPermissions":    []any{},
			"Parameters": map[string]any{
				"CROSS_ACCOUNT_VERSION": "4",
			},
		},
	}
}

func (s *lakeFormationStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "RegisterResource":
		arn := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "ResourceArn", "resourceArn"),
			"arn:aws:s3:::stackyard-lakeformation-data",
		)
		item := s.ensureResourceLocked(arn, now)
		lakeFormationMergeMap(item, payload)
		item["ResourceArn"] = arn
		item["LastModified"] = now
		return map[string]any{"ResourceArn": arn}

	case "DeregisterResource":
		arn := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "ResourceArn", "resourceArn"),
			s.firstResourceARNLocked(),
		)
		delete(s.resources, arn)
		return map[string]any{}

	case "DescribeResource":
		arn := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "ResourceArn", "resourceArn"),
			s.firstResourceARNLocked(),
		)
		return map[string]any{"ResourceInfo": lakeFormationCloneMap(s.ensureResourceLocked(arn, now))}

	case "UpdateResource":
		arn := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "ResourceArn", "resourceArn"),
			s.firstResourceARNLocked(),
		)
		item := s.ensureResourceLocked(arn, now)
		lakeFormationMergeMap(item, payload)
		item["ResourceArn"] = arn
		item["LastModified"] = now
		return map[string]any{}

	case "ListResources":
		out := make([]any, 0, len(s.resources))
		for _, item := range lakeFormationSortedMaps(s.resources) {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"ResourceInfoList": out, "NextToken": ""}

	case "GetDataLakeSettings":
		return map[string]any{"DataLakeSettings": lakeFormationCloneMap(s.dataLakeSettings)}
	case "PutDataLakeSettings":
		settings := lakeFormationPayloadMap(payload, "DataLakeSettings", "dataLakeSettings")
		if len(settings) == 0 {
			settings = lakeFormationCloneMap(payload)
		}
		if len(settings) == 0 {
			settings = map[string]any{}
		}
		s.dataLakeSettings = lakeFormationCloneMap(settings)
		return map[string]any{}
	case "GetDataLakePrincipal":
		return map[string]any{
			"Identity": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
		}

	case "GrantPermissions":
		s.grantPermissionLocked(payload)
		return map[string]any{}
	case "RevokePermissions":
		s.revokePermissionLocked(payload)
		return map[string]any{}
	case "BatchGrantPermissions":
		for _, entry := range lakeFormationPayloadSlice(payload, "Entries", "entries") {
			if m, ok := entry.(map[string]any); ok {
				s.grantPermissionLocked(m)
			}
		}
		return map[string]any{"Failures": []any{}}
	case "BatchRevokePermissions":
		for _, entry := range lakeFormationPayloadSlice(payload, "Entries", "entries") {
			if m, ok := entry.(map[string]any); ok {
				s.revokePermissionLocked(m)
			}
		}
		return map[string]any{"Failures": []any{}}
	case "ListPermissions":
		out := make([]any, 0, len(s.permissions))
		for _, p := range s.permissions {
			out = append(out, lakeFormationCloneMap(p))
		}
		return map[string]any{"PrincipalResourcePermissions": out, "NextToken": ""}

	case "CreateLFTag":
		tagKey := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "TagKey", "tagKey"),
			"tag-"+s.nextIdentifierLocked(""),
		)
		item := map[string]any{
			"TagKey":    tagKey,
			"TagValues": lakeFormationPayloadStringListAny(payload, "TagValues", "tagValues"),
		}
		if len(item["TagValues"].([]any)) == 0 {
			item["TagValues"] = []any{"value-000001"}
		}
		s.lfTags[tagKey] = item
		return map[string]any{}
	case "GetLFTag":
		tagKey := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "TagKey", "tagKey"),
			s.firstLFTagKeyLocked(),
		)
		item := lakeFormationCloneMap(s.ensureLFTagLocked(tagKey))
		return map[string]any{"TagInfo": item}
	case "UpdateLFTag":
		tagKey := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "TagKey", "tagKey"),
			s.firstLFTagKeyLocked(),
		)
		item := s.ensureLFTagLocked(tagKey)
		toAdd := lakeFormationPayloadStringListAny(payload, "TagValuesToAdd", "tagValuesToAdd")
		toDelete := lakeFormationPayloadStringList(payload, "TagValuesToDelete", "tagValuesToDelete")
		existing := lakeFormationAnyToStringSlice(item["TagValues"])
		filtered := make([]string, 0, len(existing))
		for _, current := range existing {
			shouldDelete := false
			for _, td := range toDelete {
				if current == td {
					shouldDelete = true
					break
				}
			}
			if !shouldDelete {
				filtered = append(filtered, current)
			}
		}
		for _, add := range toAdd {
			val, _ := add.(string)
			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}
			seen := false
			for _, current := range filtered {
				if current == val {
					seen = true
					break
				}
			}
			if !seen {
				filtered = append(filtered, val)
			}
		}
		item["TagValues"] = lakeFormationStringSliceToAny(filtered)
		return map[string]any{}
	case "DeleteLFTag":
		tagKey := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "TagKey", "tagKey"),
			s.firstLFTagKeyLocked(),
		)
		delete(s.lfTags, tagKey)
		return map[string]any{}
	case "ListLFTags":
		out := make([]any, 0, len(s.lfTags))
		for _, item := range lakeFormationSortedMaps(s.lfTags) {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"LFTags": out, "NextToken": ""}

	case "CreateLFTagExpression":
		name := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "Name", "name"),
			"expression-"+s.nextIdentifierLocked(""),
		)
		item := map[string]any{
			"Name":       name,
			"Expression": lakeFormationCloneList(lakeFormationPayloadSlice(payload, "Expression", "expression")),
		}
		if len(item["Expression"].([]any)) == 0 {
			item["Expression"] = []any{map[string]any{"TagKey": "environment", "TagValues": []any{"dev"}}}
		}
		s.lfTagExpressions[name] = item
		return map[string]any{}
	case "GetLFTagExpression":
		name := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "Name", "name"),
			s.firstLFTagExpressionNameLocked(),
		)
		return map[string]any{"LFTagExpression": lakeFormationCloneMap(s.ensureLFTagExpressionLocked(name))}
	case "UpdateLFTagExpression":
		name := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "Name", "name"),
			s.firstLFTagExpressionNameLocked(),
		)
		item := s.ensureLFTagExpressionLocked(name)
		expr := lakeFormationCloneList(lakeFormationPayloadSlice(payload, "Expression", "expression"))
		if len(expr) > 0 {
			item["Expression"] = expr
		}
		return map[string]any{}
	case "DeleteLFTagExpression":
		name := lakeFormationFirstNonEmpty(
			lakeFormationPayloadString(payload, "Name", "name"),
			s.firstLFTagExpressionNameLocked(),
		)
		delete(s.lfTagExpressions, name)
		return map[string]any{}
	case "ListLFTagExpressions":
		out := make([]any, 0, len(s.lfTagExpressions))
		for _, item := range lakeFormationSortedMaps(s.lfTagExpressions) {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"LFTagExpressions": out, "NextToken": ""}

	case "AddLFTagsToResource":
		key := lakeFormationResourceKey(lakeFormationPayloadMap(payload, "Resource", "resource"))
		if key == "" {
			key = lakeFormationResourceKey(map[string]any{"Catalog": map[string]any{}})
		}
		existing := s.resourceLFTags[key]
		for _, rawTag := range lakeFormationPayloadSlice(payload, "LFTags", "lfTags") {
			tagMap, _ := rawTag.(map[string]any)
			tagKey := lakeFormationPayloadString(tagMap, "TagKey", "tagKey")
			if tagKey == "" {
				continue
			}
			tagValues := lakeFormationPayloadStringListAny(tagMap, "TagValues", "tagValues")
			found := false
			for _, current := range existing {
				if lakeFormationPayloadString(current, "TagKey", "tagKey") == tagKey {
					current["TagValues"] = tagValues
					found = true
					break
				}
			}
			if !found {
				existing = append(existing, map[string]any{
					"TagKey":    tagKey,
					"TagValues": tagValues,
				})
			}
		}
		s.resourceLFTags[key] = existing
		return map[string]any{"Failures": []any{}}

	case "RemoveLFTagsFromResource":
		key := lakeFormationResourceKey(lakeFormationPayloadMap(payload, "Resource", "resource"))
		current := s.resourceLFTags[key]
		if len(current) == 0 {
			return map[string]any{"Failures": []any{}}
		}
		toRemove := map[string]struct{}{}
		for _, rawTag := range lakeFormationPayloadSlice(payload, "LFTags", "lfTags") {
			tagMap, _ := rawTag.(map[string]any)
			tagKey := lakeFormationPayloadString(tagMap, "TagKey", "tagKey")
			if tagKey != "" {
				toRemove[tagKey] = struct{}{}
			}
		}
		filtered := make([]map[string]any, 0, len(current))
		for _, tag := range current {
			tagKey := lakeFormationPayloadString(tag, "TagKey", "tagKey")
			if _, drop := toRemove[tagKey]; drop {
				continue
			}
			filtered = append(filtered, tag)
		}
		s.resourceLFTags[key] = filtered
		return map[string]any{"Failures": []any{}}

	case "GetResourceLFTags":
		key := lakeFormationResourceKey(lakeFormationPayloadMap(payload, "Resource", "resource"))
		tags := lakeFormationCloneTagList(s.resourceLFTags[key])
		return map[string]any{
			"LFTagOnDatabase": tags,
			"LFTagOnTable":    tags,
			"LFTagsOnColumns": []any{},
		}

	case "SearchDatabasesByLFTags":
		return map[string]any{
			"DatabaseList": []any{
				map[string]any{
					"Database": map[string]any{
						"CatalogId": "123456789012",
						"Name":      "default_db",
					},
					"LFTags": lakeFormationCloneTagList(s.resourceLFTags[s.firstResourceKeyLocked()]),
				},
			},
			"NextToken": "",
		}
	case "SearchTablesByLFTags":
		return map[string]any{
			"TableList": []any{
				map[string]any{
					"Table": map[string]any{
						"CatalogId":    "123456789012",
						"DatabaseName": "default_db",
						"Name":         "default_table",
					},
					"LFTags": lakeFormationCloneTagList(s.resourceLFTags[s.firstResourceKeyLocked()]),
				},
			},
			"NextToken": "",
		}

	case "CreateDataCellsFilter":
		key := s.dataCellsFilterKeyFromPayload(payload)
		item := map[string]any{
			"TableCatalogId": lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableCatalogId"), "123456789012"),
			"DatabaseName":   lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "DatabaseName", "databaseName"), "default_db"),
			"TableName":      lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableName", "tableName"), "default_table"),
			"Name":           lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "Name", "name"), "filter-"+s.nextIdentifierLocked("")),
			"VersionId":      "1",
		}
		rowFilter := lakeFormationPayloadMap(payload, "RowFilter", "rowFilter")
		if len(rowFilter) == 0 {
			rowFilter = map[string]any{"FilterExpression": "TRUE"}
		}
		item["RowFilter"] = rowFilter
		s.dataCellsFilters[key] = item
		return map[string]any{}

	case "GetDataCellsFilter":
		key := s.dataCellsFilterKeyFromPayload(payload)
		return map[string]any{"DataCellsFilter": lakeFormationCloneMap(s.ensureDataCellsFilterLocked(key))}
	case "UpdateDataCellsFilter":
		key := s.dataCellsFilterKeyFromPayload(payload)
		item := s.ensureDataCellsFilterLocked(key)
		lakeFormationMergeMap(item, payload)
		item["VersionId"] = fmt.Sprintf("%d", s.nextID)
		return map[string]any{}
	case "DeleteDataCellsFilter":
		key := s.dataCellsFilterKeyFromPayload(payload)
		delete(s.dataCellsFilters, key)
		return map[string]any{}
	case "ListDataCellsFilter":
		out := make([]any, 0, len(s.dataCellsFilters))
		for _, item := range lakeFormationSortedMaps(s.dataCellsFilters) {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"DataCellsFilters": out, "NextToken": ""}

	case "GetEffectivePermissionsForPath":
		path := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "ResourceArn", "resourceArn", "Path", "path"), "arn:aws:s3:::stackyard-lakeformation-data")
		return map[string]any{
			"Permissions": []any{
				map[string]any{
					"Principal": map[string]any{
						"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
					},
					"Resource": map[string]any{
						"DataLocation": map[string]any{"ResourceArn": path},
					},
					"Permissions": []any{"DATA_LOCATION_ACCESS"},
				},
			},
			"NextToken": "",
		}

	case "StartTransaction":
		token := lakeFormationPayloadString(payload, "TransactionId", "ClientRequestToken", "clientRequestToken")
		if token != "" {
			if existingID, ok := s.txTokenToID[token]; ok && existingID != "" {
				return map[string]any{"TransactionId": existingID}
			}
		}
		txID := "tx-" + s.nextIdentifierLocked("")
		s.transactions[txID] = map[string]any{
			"TransactionId":     txID,
			"TransactionStatus": "ACTIVE",
			"Description":       lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "Description", "description"), "stackyard-transaction"),
			"StartTime":         now,
		}
		if token != "" {
			s.txTokenToID[token] = txID
		}
		return map[string]any{"TransactionId": txID}

	case "DescribeTransaction":
		txID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TransactionId", "transactionId"), s.firstTransactionIDLocked())
		return map[string]any{"TransactionDescription": lakeFormationCloneMap(s.ensureTransactionLocked(txID, now))}
	case "ExtendTransaction":
		txID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TransactionId", "transactionId"), s.firstTransactionIDLocked())
		item := s.ensureTransactionLocked(txID, now)
		item["TransactionStatus"] = "ACTIVE"
		item["LastExtendedAt"] = now
		return map[string]any{"TransactionId": txID}
	case "CommitTransaction":
		txID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TransactionId", "transactionId"), s.firstTransactionIDLocked())
		item := s.ensureTransactionLocked(txID, now)
		item["TransactionStatus"] = "COMMITTED"
		item["CommittedAt"] = now
		return map[string]any{}
	case "CancelTransaction":
		txID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TransactionId", "transactionId"), s.firstTransactionIDLocked())
		item := s.ensureTransactionLocked(txID, now)
		item["TransactionStatus"] = "ABORTED"
		item["CanceledAt"] = now
		return map[string]any{}
	case "DeleteObjectsOnCancel":
		return map[string]any{}
	case "ListTransactions":
		out := make([]any, 0, len(s.transactions))
		for _, tx := range lakeFormationSortedMaps(s.transactions) {
			out = append(out, lakeFormationCloneMap(tx))
		}
		return map[string]any{"Transactions": out, "NextToken": ""}

	case "StartQueryPlanning":
		queryID := "query-" + s.nextIdentifierLocked("")
		s.queryRuns[queryID] = map[string]any{
			"QueryId": queryID,
			"State":   "FINISHED",
		}
		return map[string]any{"QueryId": queryID}
	case "GetQueryState":
		queryID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "QueryId", "queryId"), s.firstQueryIDLocked())
		query := s.ensureQueryLocked(queryID)
		return map[string]any{"QueryId": queryID, "State": lakeFormationPayloadString(query, "State")}
	case "GetQueryStatistics":
		queryID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "QueryId", "queryId"), s.firstQueryIDLocked())
		return map[string]any{
			"QueryId": queryID,
			"ExecutionStatistics": map[string]any{
				"AverageExecutionTimeMillis": 25,
				"DataScannedBytes":           0,
				"WorkUnitsExecutedCount":     1,
			},
			"PlanningStatistics": map[string]any{
				"EstimatedDataToScanBytes": 0,
				"PlanningTimeMillis":       2,
				"QueueTimeMillis":          1,
				"WorkUnitsGeneratedCount":  1,
			},
		}
	case "GetWorkUnits":
		queryID := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "QueryId", "queryId"), s.firstQueryIDLocked())
		return map[string]any{
			"QueryId": queryID,
			"Token":   "token-000001",
			"WorkUnitRanges": []any{
				map[string]any{
					"WorkUnitIdMin": 0,
					"WorkUnitIdMax": 0,
				},
			},
		}
	case "GetWorkUnitResults":
		return map[string]any{
			"Rows":      []any{},
			"NextToken": "",
		}

	case "GetTemporaryDataLocationCredentials":
		return map[string]any{"TemporaryCredentials": s.temporaryCredentials(now)}
	case "GetTemporaryGluePartitionCredentials":
		return map[string]any{"TemporaryCredentials": s.temporaryCredentials(now)}
	case "GetTemporaryGlueTableCredentials":
		return map[string]any{"TemporaryCredentials": s.temporaryCredentials(now)}

	case "UpdateTableObjects":
		db := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "DatabaseName", "databaseName"), "default_db")
		table := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableName", "tableName"), "default_table")
		key := db + "|" + table
		ops := lakeFormationPayloadSlice(payload, "WriteOperations", "writeOperations")
		existing := s.tableObjects[key]
		if len(ops) == 0 && len(existing) == 0 {
			existing = append(existing, map[string]any{
				"Uri":             "s3://stackyard-lakeformation-data/default/default-000001.parquet",
				"ETag":            "etag-000001",
				"Size":            1024,
				"PartitionValues": []any{"2026", "02", "28"},
				"LastModified":    now,
			})
		}
		for _, rawOp := range ops {
			op, _ := rawOp.(map[string]any)
			add := lakeFormationPayloadMap(op, "AddObject", "addObject")
			if len(add) > 0 {
				if _, ok := add["Uri"]; !ok {
					add["Uri"] = fmt.Sprintf("s3://stackyard-lakeformation-data/%s/%s/object-%s.parquet", db, table, s.nextIdentifierLocked(""))
				}
				existing = append(existing, add)
			}
		}
		s.tableObjects[key] = existing
		return map[string]any{}
	case "GetTableObjects":
		db := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "DatabaseName", "databaseName"), "default_db")
		table := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableName", "tableName"), "default_table")
		key := db + "|" + table
		out := lakeFormationCloneListOfMaps(s.tableObjects[key])
		return map[string]any{"Objects": out, "NextToken": ""}

	case "UpdateTableStorageOptimizer":
		db := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "DatabaseName", "databaseName"), "default_db")
		table := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableName", "tableName"), "default_table")
		optimizerType := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "StorageOptimizerType", "storageOptimizerType"), "COMPACTION")
		key := db + "|" + table + "|" + optimizerType
		item := s.storageOptimizers[key]
		if item == nil {
			item = map[string]any{}
			s.storageOptimizers[key] = item
		}
		item["DatabaseName"] = db
		item["TableName"] = table
		item["StorageOptimizerType"] = optimizerType
		item["Enabled"] = true
		item["LastUpdatedAt"] = now
		config := lakeFormationPayloadMap(payload, "Config", "config")
		if len(config) > 0 {
			item["Config"] = config
		}
		return map[string]any{}
	case "ListTableStorageOptimizers":
		out := make([]any, 0, len(s.storageOptimizers))
		for _, item := range lakeFormationSortedMaps(s.storageOptimizers) {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"StorageOptimizerList": out, "NextToken": ""}

	case "CreateLakeFormationOptIn":
		entry := map[string]any{
			"Principal":    lakeFormationPayloadMap(payload, "Principal", "principal"),
			"Resource":     lakeFormationPayloadMap(payload, "Resource", "resource"),
			"LastModified": now,
		}
		if len(entry["Principal"].(map[string]any)) == 0 {
			entry["Principal"] = map[string]any{"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role"}
		}
		if len(entry["Resource"].(map[string]any)) == 0 {
			entry["Resource"] = map[string]any{"Catalog": map[string]any{}}
		}
		s.lakeFormationOptIns = append(s.lakeFormationOptIns, entry)
		return map[string]any{}
	case "DeleteLakeFormationOptIn":
		if len(s.lakeFormationOptIns) > 0 {
			s.lakeFormationOptIns = s.lakeFormationOptIns[1:]
		}
		return map[string]any{}
	case "ListLakeFormationOptIns":
		out := make([]any, 0, len(s.lakeFormationOptIns))
		for _, item := range s.lakeFormationOptIns {
			out = append(out, lakeFormationCloneMap(item))
		}
		return map[string]any{"LakeFormationOptInsInfoList": out, "NextToken": ""}

	case "CreateLakeFormationIdentityCenterConfiguration":
		cfg := lakeFormationPayloadMap(payload, "LakeFormationIdentityCenterConfiguration", "lakeFormationIdentityCenterConfiguration")
		if len(cfg) == 0 {
			cfg = lakeFormationCloneMap(payload)
		}
		if len(cfg) == 0 {
			cfg = map[string]any{}
		}
		cfg["ApplicationStatus"] = "ENABLED"
		cfg["LastUpdatedTime"] = now
		s.identityCenterConfig = cfg
		return map[string]any{}
	case "DescribeLakeFormationIdentityCenterConfiguration":
		return lakeFormationCloneMap(s.identityCenterConfig)
	case "UpdateLakeFormationIdentityCenterConfiguration":
		cfg := lakeFormationPayloadMap(payload, "LakeFormationIdentityCenterConfiguration", "lakeFormationIdentityCenterConfiguration")
		if len(cfg) == 0 {
			cfg = lakeFormationCloneMap(payload)
		}
		if s.identityCenterConfig == nil {
			s.identityCenterConfig = map[string]any{}
		}
		lakeFormationMergeMap(s.identityCenterConfig, cfg)
		s.identityCenterConfig["LastUpdatedTime"] = now
		return map[string]any{}
	case "DeleteLakeFormationIdentityCenterConfiguration":
		s.identityCenterConfig = map[string]any{}
		return map[string]any{}

	case "AssumeDecoratedRoleWithSAML":
		return map[string]any{
			"Subject": "stackyard-user",
			"AssumedRoleUser": map[string]any{
				"Arn":           "arn:aws:sts::123456789012:assumed-role/stackyard-lakeformation-role/stackyard-user",
				"AssumedRoleId": "AROA123EXAMPLE:stackyard-user",
			},
			"Credentials": s.temporaryCredentials(now),
		}

	default:
		return map[string]any{}
	}
}

func (s *lakeFormationStore) ensureResourceLocked(arn, now string) map[string]any {
	item := s.resources[arn]
	if item != nil {
		return item
	}
	item = map[string]any{
		"ResourceArn":                arn,
		"RoleArn":                    "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
		"HybridAccessEnabled":        false,
		"WithFederation":             false,
		"WithPrivilegedAccess":       true,
		"UseServiceLinkedRole":       true,
		"ResourceShareType":          "EXTERNAL",
		"AllowExternalDataFiltering": false,
		"LastModified":               now,
	}
	s.resources[arn] = item
	return item
}

func (s *lakeFormationStore) ensureLFTagLocked(tagKey string) map[string]any {
	item := s.lfTags[tagKey]
	if item != nil {
		return item
	}
	item = map[string]any{
		"TagKey":    tagKey,
		"TagValues": []any{"value-000001"},
	}
	s.lfTags[tagKey] = item
	return item
}

func (s *lakeFormationStore) ensureLFTagExpressionLocked(name string) map[string]any {
	item := s.lfTagExpressions[name]
	if item != nil {
		return item
	}
	item = map[string]any{
		"Name":       name,
		"Expression": []any{},
	}
	s.lfTagExpressions[name] = item
	return item
}

func (s *lakeFormationStore) ensureDataCellsFilterLocked(key string) map[string]any {
	item := s.dataCellsFilters[key]
	if item != nil {
		return item
	}
	db, table, name := lakeFormationSplitFilterKey(key)
	item = map[string]any{
		"TableCatalogId": "123456789012",
		"DatabaseName":   db,
		"TableName":      table,
		"Name":           name,
		"VersionId":      "1",
		"RowFilter": map[string]any{
			"FilterExpression": "TRUE",
		},
	}
	s.dataCellsFilters[key] = item
	return item
}

func (s *lakeFormationStore) ensureTransactionLocked(txID, now string) map[string]any {
	item := s.transactions[txID]
	if item != nil {
		return item
	}
	item = map[string]any{
		"TransactionId":     txID,
		"TransactionStatus": "ACTIVE",
		"Description":       "stackyard-transaction",
		"StartTime":         now,
	}
	s.transactions[txID] = item
	return item
}

func (s *lakeFormationStore) ensureQueryLocked(queryID string) map[string]any {
	item := s.queryRuns[queryID]
	if item != nil {
		return item
	}
	item = map[string]any{
		"QueryId": queryID,
		"State":   "FINISHED",
	}
	s.queryRuns[queryID] = item
	return item
}

func (s *lakeFormationStore) dataCellsFilterKeyFromPayload(payload map[string]any) string {
	db := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "DatabaseName", "databaseName"), "default_db")
	table := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "TableName", "tableName"), "default_table")
	name := lakeFormationFirstNonEmpty(lakeFormationPayloadString(payload, "Name", "name"), "stackyard-filter")
	return db + "|" + table + "|" + name
}

func (s *lakeFormationStore) grantPermissionLocked(payload map[string]any) {
	permission := map[string]any{
		"Principal":                  lakeFormationPayloadMap(payload, "Principal", "principal"),
		"Resource":                   lakeFormationPayloadMap(payload, "Resource", "resource"),
		"Permissions":                lakeFormationCloneList(lakeFormationPayloadSlice(payload, "Permissions", "permissions")),
		"PermissionsWithGrantOption": lakeFormationCloneList(lakeFormationPayloadSlice(payload, "PermissionsWithGrantOption", "permissionsWithGrantOption")),
	}
	if len(permission["Principal"].(map[string]any)) == 0 {
		permission["Principal"] = map[string]any{
			"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stackyard-lakeformation-role",
		}
	}
	if len(permission["Resource"].(map[string]any)) == 0 {
		permission["Resource"] = map[string]any{
			"Catalog": map[string]any{},
		}
	}
	if len(permission["Permissions"].([]any)) == 0 {
		permission["Permissions"] = []any{"ALL"}
	}
	s.permissions = append(s.permissions, permission)
}

func (s *lakeFormationStore) revokePermissionLocked(_ map[string]any) {
	if len(s.permissions) == 0 {
		return
	}
	s.permissions = s.permissions[1:]
}

func (s *lakeFormationStore) nextIdentifierLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s%06d", prefix, id)
}

func (s *lakeFormationStore) temporaryCredentials(now string) map[string]any {
	return map[string]any{
		"AccessKeyId":     "ASIAXAMPLESTACKYARD",
		"SecretAccessKey": "stackyard-secret-access-key",
		"SessionToken":    "stackyard-session-token",
		"Expiration":      now,
	}
}

func (s *lakeFormationStore) firstResourceARNLocked() string {
	for arn := range s.resources {
		return arn
	}
	return "arn:aws:s3:::stackyard-lakeformation-data"
}

func (s *lakeFormationStore) firstLFTagKeyLocked() string {
	for key := range s.lfTags {
		return key
	}
	return "environment"
}

func (s *lakeFormationStore) firstLFTagExpressionNameLocked() string {
	for name := range s.lfTagExpressions {
		return name
	}
	return "stackyard-expression"
}

func (s *lakeFormationStore) firstTransactionIDLocked() string {
	for txID := range s.transactions {
		return txID
	}
	return "tx-000001"
}

func (s *lakeFormationStore) firstQueryIDLocked() string {
	for queryID := range s.queryRuns {
		return queryID
	}
	return "query-000001"
}

func (s *lakeFormationStore) firstResourceKeyLocked() string {
	for key := range s.resourceLFTags {
		return key
	}
	return lakeFormationResourceKey(map[string]any{"Catalog": map[string]any{}})
}

func lakeFormationResourceKey(resource map[string]any) string {
	if len(resource) == 0 {
		return ""
	}
	raw, err := json.Marshal(resource)
	if err != nil {
		return ""
	}
	return string(raw)
}

func lakeFormationSplitFilterKey(key string) (string, string, string) {
	parts := strings.SplitN(key, "|", 3)
	if len(parts) != 3 {
		return "default_db", "default_table", "stackyard-filter"
	}
	return parts[0], parts[1], parts[2]
}

func lakeFormationSortedMaps(src map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, src[key])
	}
	return out
}

func lakeFormationPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := payload[key]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t)
			}
		case fmt.Stringer:
			text := strings.TrimSpace(t.String())
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func lakeFormationPayloadMap(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		v, ok := payload[key]
		if !ok {
			continue
		}
		if m, ok := v.(map[string]any); ok && len(m) >= 0 {
			return lakeFormationCloneMap(m)
		}
	}
	return map[string]any{}
}

func lakeFormationPayloadSlice(payload map[string]any, keys ...string) []any {
	for _, key := range keys {
		v, ok := payload[key]
		if !ok {
			continue
		}
		if items, ok := v.([]any); ok {
			return lakeFormationCloneList(items)
		}
	}
	return nil
}

func lakeFormationPayloadStringList(payload map[string]any, keys ...string) []string {
	raw := lakeFormationPayloadSlice(payload, keys...)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func lakeFormationPayloadStringListAny(payload map[string]any, keys ...string) []any {
	values := lakeFormationPayloadStringList(payload, keys...)
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func lakeFormationAnyToStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func lakeFormationStringSliceToAny(items []string) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func lakeFormationFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func lakeFormationMergeMap(dst, src map[string]any) {
	for key, value := range src {
		dst[key] = lakeFormationCloneAny(value)
	}
}

func lakeFormationCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = lakeFormationCloneAny(value)
	}
	return out
}

func lakeFormationCloneList(src []any) []any {
	if src == nil {
		return nil
	}
	out := make([]any, len(src))
	for i := range src {
		out[i] = lakeFormationCloneAny(src[i])
	}
	return out
}

func lakeFormationCloneTagList(src []map[string]any) []any {
	if src == nil {
		return []any{}
	}
	out := make([]any, 0, len(src))
	for _, item := range src {
		out = append(out, lakeFormationCloneMap(item))
	}
	return out
}

func lakeFormationCloneListOfMaps(src []map[string]any) []any {
	if src == nil {
		return []any{}
	}
	out := make([]any, 0, len(src))
	for _, item := range src {
		out = append(out, lakeFormationCloneMap(item))
	}
	return out
}

func lakeFormationCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return lakeFormationCloneMap(t)
	case []any:
		return lakeFormationCloneList(t)
	case []map[string]any:
		return lakeFormationCloneListOfMaps(t)
	default:
		return t
	}
}
