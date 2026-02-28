package dynamodb

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrValidation             = errors.New("validation error")
	ErrResourceNotFound       = errors.New("resource not found")
	ErrResourceInUse          = errors.New("resource in use")
	ErrConditionalCheckFailed = errors.New("conditional check failed")
)

const (
	defaultRegion    = "us-east-1"
	defaultAccountID = "123456789012"
)

type AttributeDefinition struct {
	AttributeName string `json:"AttributeName"`
	AttributeType string `json:"AttributeType"`
}

type KeySchemaElement struct {
	AttributeName string `json:"AttributeName"`
	KeyType       string `json:"KeyType"`
}

type BillingModeSummary struct {
	BillingMode string `json:"BillingMode"`
}

type TableDescription struct {
	AttributeDefinitions []AttributeDefinition `json:"AttributeDefinitions,omitempty"`
	BillingModeSummary   *BillingModeSummary   `json:"BillingModeSummary,omitempty"`
	CreationDateTime     time.Time             `json:"CreationDateTime"`
	ItemCount            int64                 `json:"ItemCount"`
	KeySchema            []KeySchemaElement    `json:"KeySchema,omitempty"`
	TableArn             string                `json:"TableArn"`
	TableName            string                `json:"TableName"`
	TableSizeBytes       int64                 `json:"TableSizeBytes"`
	TableStatus          string                `json:"TableStatus"`
}

type Limits struct {
	AccountMaxReadCapacityUnits  int64 `json:"AccountMaxReadCapacityUnits"`
	AccountMaxWriteCapacityUnits int64 `json:"AccountMaxWriteCapacityUnits"`
	TableMaxReadCapacityUnits    int64 `json:"TableMaxReadCapacityUnits"`
	TableMaxWriteCapacityUnits   int64 `json:"TableMaxWriteCapacityUnits"`
}

type BackupDescription struct {
	BackupArn              string    `json:"BackupArn"`
	BackupName             string    `json:"BackupName"`
	BackupCreationDateTime time.Time `json:"BackupCreationDateTime"`
	BackupStatus           string    `json:"BackupStatus"`
	BackupType             string    `json:"BackupType"`
	SourceTableArn         string    `json:"SourceTableArn"`
	SourceTableName        string    `json:"SourceTableName"`
}

type ReplicaDescription struct {
	RegionName    string `json:"RegionName"`
	ReplicaStatus string `json:"ReplicaStatus"`
}

type GlobalTableDescription struct {
	GlobalTableArn    string               `json:"GlobalTableArn"`
	GlobalTableName   string               `json:"GlobalTableName"`
	GlobalTableStatus string               `json:"GlobalTableStatus"`
	CreationDateTime  time.Time            `json:"CreationDateTime"`
	ReplicationGroup  []ReplicaDescription `json:"ReplicationGroup"`
}

type GlobalTableSettingsDescription struct {
	GlobalTableName string           `json:"GlobalTableName"`
	ReplicaSettings []map[string]any `json:"ReplicaSettings,omitempty"`
}

type ExportDescription struct {
	ExportArn    string    `json:"ExportArn"`
	ExportStatus string    `json:"ExportStatus"`
	ExportFormat string    `json:"ExportFormat"`
	ExportTime   time.Time `json:"ExportTime"`
	ClientToken  string    `json:"ClientToken,omitempty"`
	S3Bucket     string    `json:"S3Bucket,omitempty"`
	TableArn     string    `json:"TableArn"`
}

type ImportDescription struct {
	ImportArn          string    `json:"ImportArn"`
	ImportStatus       string    `json:"ImportStatus"`
	ClientToken        string    `json:"ClientToken,omitempty"`
	InputFormat        string    `json:"InputFormat,omitempty"`
	TableArn           string    `json:"TableArn"`
	ProcessedSizeBytes int64     `json:"ProcessedSizeBytes"`
	ProcessedItemCount int64     `json:"ProcessedItemCount"`
	ImportTime         time.Time `json:"ImportTime"`
}

type tableSnapshot struct {
	Description TableDescription
	HashKey     string
	RangeKey    string
	Items       map[string]map[string]any
}

type backupRecord struct {
	Description BackupDescription
	Snapshot    tableSnapshot
}

type globalTableRecord struct {
	Description GlobalTableDescription
	Settings    GlobalTableSettingsDescription
}

type exportRecord struct {
	Description ExportDescription
}

type importRecord struct {
	Description ImportDescription
}

type resourcePolicyRecord struct {
	Policy     string
	RevisionID string
}

type exportTokenRecord struct {
	ExportArn    string
	TableArn     string
	S3Bucket     string
	ExportFormat string
}

type importTokenRecord struct {
	ImportArn   string
	TableName   string
	InputFormat string
}

type Table struct {
	description TableDescription
	hashKey     string
	rangeKey    string
	items       map[string]map[string]any
}

type QueryInput struct {
	TableName                 string
	KeyConditionExpression    string
	KeyConditions             map[string]any
	ExpressionAttributeNames  map[string]string
	ExpressionAttributeValues map[string]any
	Limit                     int
}

type QueryOutput struct {
	Items        []map[string]any
	Count        int
	ScannedCount int
}

type Service struct {
	mu                  sync.Mutex
	tables              map[string]*Table
	backups             map[string]*backupRecord
	globalTables        map[string]*globalTableRecord
	exports             map[string]*exportRecord
	imports             map[string]*importRecord
	tableAutoScaling    map[string]map[string]any
	kinesisDestinations map[string]map[string]any
	continuousBackups   map[string]map[string]any
	timeToLive          map[string]map[string]any
	contributorInsights map[string]map[string]any
	resourceTags        map[string]map[string]string
	resourcePolicies    map[string]*resourcePolicyRecord
	exportTokenRecords  map[string]*exportTokenRecord
	importTokenRecords  map[string]*importTokenRecord
	backupIDSequence    uint64
	exportIDSequence    uint64
	importIDSequence    uint64
	resourcePolicySeq   uint64
}

func NewService() *Service {
	return &Service{
		tables:              map[string]*Table{},
		backups:             map[string]*backupRecord{},
		globalTables:        map[string]*globalTableRecord{},
		exports:             map[string]*exportRecord{},
		imports:             map[string]*importRecord{},
		tableAutoScaling:    map[string]map[string]any{},
		kinesisDestinations: map[string]map[string]any{},
		continuousBackups:   map[string]map[string]any{},
		timeToLive:          map[string]map[string]any{},
		contributorInsights: map[string]map[string]any{},
		resourceTags:        map[string]map[string]string{},
		resourcePolicies:    map[string]*resourcePolicyRecord{},
		exportTokenRecords:  map[string]*exportTokenRecord{},
		importTokenRecords:  map[string]*importTokenRecord{},
	}
}

func (s *Service) CreateTable(name string, attrs []AttributeDefinition, keys []KeySchemaElement, billingMode string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return TableDescription{}, ErrValidation
	}
	if _, exists := s.tables[name]; exists {
		return TableDescription{}, ErrResourceInUse
	}
	hashKey, rangeKey, err := validateKeySchema(keys)
	if err != nil {
		return TableDescription{}, err
	}
	if strings.TrimSpace(billingMode) == "" {
		billingMode = "PAY_PER_REQUEST"
	}
	desc := TableDescription{
		AttributeDefinitions: cloneAttrDefinitions(attrs),
		BillingModeSummary:   &BillingModeSummary{BillingMode: billingMode},
		CreationDateTime:     time.Now().UTC(),
		ItemCount:            0,
		KeySchema:            cloneKeySchema(keys),
		TableArn:             tableARN(name),
		TableName:            name,
		TableSizeBytes:       0,
		TableStatus:          "ACTIVE",
	}
	s.tables[name] = &Table{description: desc, hashKey: hashKey, rangeKey: rangeKey, items: map[string]map[string]any{}}
	return cloneTableDescription(desc), nil
}

func (s *Service) DescribeTable(name string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(name)
	if err != nil {
		return TableDescription{}, err
	}
	refreshTableMetrics(table)
	return cloneTableDescription(table.description), nil
}

func (s *Service) ListTables(limit int, exclusiveStartTableName string) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", ErrValidation
	}
	nameList := make([]string, 0, len(s.tables))
	for name := range s.tables {
		nameList = append(nameList, name)
	}
	sort.Strings(nameList)

	start := 0
	exclusiveStartTableName = strings.TrimSpace(exclusiveStartTableName)
	if exclusiveStartTableName != "" {
		found := false
		for i, name := range nameList {
			if name == exclusiveStartTableName {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", ErrValidation
		}
	}
	if start > len(nameList) {
		start = len(nameList)
	}
	end := start + limit
	if end > len(nameList) {
		end = len(nameList)
	}

	out := append([]string(nil), nameList[start:end]...)
	lastEvaluated := ""
	if end < len(nameList) {
		lastEvaluated = nameList[end-1]
	}
	return out, lastEvaluated, nil
}

func (s *Service) UpdateTable(name string, billingMode string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(name)
	if err != nil {
		return TableDescription{}, err
	}
	if strings.TrimSpace(billingMode) != "" {
		table.description.BillingModeSummary = &BillingModeSummary{BillingMode: billingMode}
	}
	table.description.TableStatus = "ACTIVE"
	refreshTableMetrics(table)
	return cloneTableDescription(table.description), nil
}

func (s *Service) DeleteTable(name string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(name)
	if err != nil {
		return TableDescription{}, err
	}
	tableName := table.description.TableName
	tableArn := table.description.TableArn
	deleted := cloneTableDescription(table.description)
	deleted.TableStatus = "DELETING"
	delete(s.tables, strings.TrimSpace(name))
	delete(s.tableAutoScaling, tableName)
	delete(s.kinesisDestinations, tableName)
	delete(s.continuousBackups, tableName)
	delete(s.timeToLive, tableName)
	for key, record := range s.contributorInsights {
		if stringValue(record["TableName"]) == tableName {
			delete(s.contributorInsights, key)
		}
	}
	delete(s.resourceTags, tableArn)
	delete(s.resourcePolicies, tableArn)
	return deleted, nil
}

func (s *Service) DescribeLimits() Limits {
	return Limits{
		AccountMaxReadCapacityUnits:  100000,
		AccountMaxWriteCapacityUnits: 100000,
		TableMaxReadCapacityUnits:    40000,
		TableMaxWriteCapacityUnits:   40000,
	}
}

func (s *Service) PutItem(tableName string, item map[string]any, returnValues string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, ErrValidation
	}
	key, err := table.itemKey(item)
	if err != nil {
		return nil, err
	}
	old := cloneItem(table.items[key])
	table.items[key] = cloneItem(item)
	refreshTableMetrics(table)
	if strings.EqualFold(strings.TrimSpace(returnValues), "ALL_OLD") {
		return old, nil
	}
	return nil, nil
}

func (s *Service) GetItem(tableName string, key map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	compositeKey, err := table.lookupKey(key)
	if err != nil {
		return nil, err
	}
	item, ok := table.items[compositeKey]
	if !ok {
		return nil, nil
	}
	return cloneItem(item), nil
}

func (s *Service) DeleteItem(tableName string, key map[string]any, returnValues string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	compositeKey, err := table.lookupKey(key)
	if err != nil {
		return nil, err
	}
	old := cloneItem(table.items[compositeKey])
	delete(table.items, compositeKey)
	refreshTableMetrics(table)
	if strings.EqualFold(strings.TrimSpace(returnValues), "ALL_OLD") {
		return old, nil
	}
	return nil, nil
}

func (s *Service) UpdateItem(
	tableName string,
	key map[string]any,
	attributeUpdates map[string]any,
	updateExpression string,
	exprNames map[string]string,
	exprValues map[string]any,
	returnValues string,
) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	compositeKey, err := table.lookupKey(key)
	if err != nil {
		return nil, err
	}
	current := cloneItem(table.items[compositeKey])
	if current == nil {
		current = cloneItem(key)
	}
	old := cloneItem(current)

	if len(attributeUpdates) > 0 {
		for rawName, rawUpdate := range attributeUpdates {
			name := strings.TrimSpace(rawName)
			if name == "" {
				continue
			}
			updateMap, _ := rawUpdate.(map[string]any)
			action := strings.ToUpper(strings.TrimSpace(stringValue(updateMap["Action"])))
			if action == "" {
				action = "PUT"
			}
			switch action {
			case "PUT", "ADD":
				current[name] = cloneAny(updateMap["Value"])
			case "DELETE":
				delete(current, name)
			default:
				return nil, ErrValidation
			}
		}
	} else if strings.TrimSpace(updateExpression) != "" {
		if err := applyUpdateExpression(current, updateExpression, exprNames, exprValues); err != nil {
			return nil, err
		}
	}

	table.items[compositeKey] = cloneItem(current)
	refreshTableMetrics(table)

	switch strings.ToUpper(strings.TrimSpace(returnValues)) {
	case "ALL_OLD":
		return old, nil
	case "ALL_NEW", "UPDATED_NEW":
		return cloneItem(current), nil
	default:
		return nil, nil
	}
}

func (s *Service) BatchWriteItem(requestItems map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for tableName, rawRequests := range requestItems {
		table, err := s.requireTable(tableName)
		if err != nil {
			return nil, err
		}
		requests, ok := rawRequests.([]any)
		if !ok {
			return nil, ErrValidation
		}
		for _, rawReq := range requests {
			req, ok := rawReq.(map[string]any)
			if !ok {
				return nil, ErrValidation
			}
			if putReq, ok := req["PutRequest"].(map[string]any); ok {
				item, _ := putReq["Item"].(map[string]any)
				if len(item) == 0 {
					return nil, ErrValidation
				}
				key, err := table.itemKey(item)
				if err != nil {
					return nil, err
				}
				table.items[key] = cloneItem(item)
				continue
			}
			if delReq, ok := req["DeleteRequest"].(map[string]any); ok {
				keySpec, _ := delReq["Key"].(map[string]any)
				lookupKey, err := table.lookupKey(keySpec)
				if err != nil {
					return nil, err
				}
				delete(table.items, lookupKey)
				continue
			}
			return nil, ErrValidation
		}
		refreshTableMetrics(table)
	}
	return map[string]any{}, nil
}

func (s *Service) BatchGetItem(requestItems map[string]any) (map[string][]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	responses := map[string][]map[string]any{}
	for tableName, rawReq := range requestItems {
		table, err := s.requireTable(tableName)
		if err != nil {
			return nil, err
		}
		req, ok := rawReq.(map[string]any)
		if !ok {
			return nil, ErrValidation
		}
		keysRaw, ok := req["Keys"].([]any)
		if !ok {
			return nil, ErrValidation
		}
		items := make([]map[string]any, 0, len(keysRaw))
		for _, entry := range keysRaw {
			keySpec, _ := entry.(map[string]any)
			lookupKey, err := table.lookupKey(keySpec)
			if err != nil {
				return nil, err
			}
			if item := table.items[lookupKey]; item != nil {
				items = append(items, cloneItem(item))
			}
		}
		responses[tableName] = items
	}
	return responses, nil
}

func (s *Service) Query(input QueryInput) (QueryOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(input.TableName)
	if err != nil {
		return QueryOutput{}, err
	}
	filters, err := buildQueryFilters(input, table)
	if err != nil {
		return QueryOutput{}, err
	}
	keys := make([]string, 0, len(table.items))
	for k := range table.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	limit := input.Limit
	if limit <= 0 {
		limit = len(keys)
	}
	items := make([]map[string]any, 0)
	scanned := 0
	for _, k := range keys {
		item := table.items[k]
		scanned++
		if !matchesFilters(item, filters) {
			continue
		}
		items = append(items, cloneItem(item))
		if len(items) >= limit {
			break
		}
	}
	return QueryOutput{Items: items, Count: len(items), ScannedCount: scanned}, nil
}

func (s *Service) Scan(tableName string, limit int) (QueryOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return QueryOutput{}, err
	}
	keys := make([]string, 0, len(table.items))
	for k := range table.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if limit <= 0 {
		limit = len(keys)
	}
	items := make([]map[string]any, 0)
	for _, k := range keys {
		items = append(items, cloneItem(table.items[k]))
		if len(items) >= limit {
			break
		}
	}
	return QueryOutput{Items: items, Count: len(items), ScannedCount: len(keys)}, nil
}

func (s *Service) ExecuteStatement(statement string, _ []any, limit int) (QueryOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return QueryOutput{}, ErrValidation
	}
	op := strings.ToUpper(firstToken(statement))
	switch op {
	case "SELECT":
		tableName := extractStatementTableName(statement, "FROM")
		if tableName == "" {
			return QueryOutput{}, ErrValidation
		}
		table, err := s.requireTable(tableName)
		if err != nil {
			return QueryOutput{}, err
		}
		keys := make([]string, 0, len(table.items))
		for key := range table.items {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if limit <= 0 {
			limit = len(keys)
		}
		items := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, cloneItem(table.items[key]))
			if len(items) >= limit {
				break
			}
		}
		return QueryOutput{Items: items, Count: len(items), ScannedCount: len(keys)}, nil
	case "INSERT", "UPDATE", "DELETE":
		tableName := extractStatementTableName(statement, "INTO")
		if tableName == "" {
			tableName = extractStatementTableName(statement, "FROM")
		}
		if tableName == "" {
			tableName = extractStatementTableName(statement, "UPDATE")
		}
		if tableName == "" {
			return QueryOutput{}, ErrValidation
		}
		if _, err := s.requireTable(tableName); err != nil {
			return QueryOutput{}, err
		}
		return QueryOutput{}, nil
	default:
		return QueryOutput{}, ErrValidation
	}
}

func (s *Service) BatchExecuteStatement(statements []map[string]any) ([]QueryOutput, error) {
	out := make([]QueryOutput, 0, len(statements))
	for _, statement := range statements {
		result, err := s.ExecuteStatement(stringValue(statement["Statement"]), nil, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, result)
	}
	return out, nil
}

func (s *Service) ExecuteTransaction(statements []map[string]any) ([]QueryOutput, error) {
	return s.BatchExecuteStatement(statements)
}

func (s *Service) TransactGetItems(transactItems []map[string]any) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]map[string]any, 0, len(transactItems))
	for _, entry := range transactItems {
		getReq, ok := entry["Get"].(map[string]any)
		if !ok {
			return nil, ErrValidation
		}
		table, err := s.requireTable(stringValue(getReq["TableName"]))
		if err != nil {
			return nil, err
		}
		keySpec, _ := getReq["Key"].(map[string]any)
		compositeKey, err := table.lookupKey(keySpec)
		if err != nil {
			return nil, err
		}
		response := map[string]any{}
		if item := table.items[compositeKey]; item != nil {
			response["Item"] = cloneItem(item)
		}
		out = append(out, response)
	}
	return out, nil
}

func (s *Service) TransactWriteItems(transactItems []map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range transactItems {
		if putReq, ok := entry["Put"].(map[string]any); ok {
			table, err := s.requireTable(stringValue(putReq["TableName"]))
			if err != nil {
				return err
			}
			item, _ := putReq["Item"].(map[string]any)
			if len(item) == 0 {
				return ErrValidation
			}
			key, err := table.itemKey(item)
			if err != nil {
				return err
			}
			table.items[key] = cloneItem(item)
			refreshTableMetrics(table)
			continue
		}
		if deleteReq, ok := entry["Delete"].(map[string]any); ok {
			table, err := s.requireTable(stringValue(deleteReq["TableName"]))
			if err != nil {
				return err
			}
			keySpec, _ := deleteReq["Key"].(map[string]any)
			key, err := table.lookupKey(keySpec)
			if err != nil {
				return err
			}
			delete(table.items, key)
			refreshTableMetrics(table)
			continue
		}
		if updateReq, ok := entry["Update"].(map[string]any); ok {
			table, err := s.requireTable(stringValue(updateReq["TableName"]))
			if err != nil {
				return err
			}
			keySpec, _ := updateReq["Key"].(map[string]any)
			key, err := table.lookupKey(keySpec)
			if err != nil {
				return err
			}
			current := cloneItem(table.items[key])
			if current == nil {
				current = cloneItem(keySpec)
			}
			err = applyUpdateExpression(
				current,
				stringValue(updateReq["UpdateExpression"]),
				toStringMap(updateReq["ExpressionAttributeNames"]),
				toAnyMap(updateReq["ExpressionAttributeValues"]),
			)
			if err != nil {
				return err
			}
			table.items[key] = current
			refreshTableMetrics(table)
			continue
		}
		if conditionReq, ok := entry["ConditionCheck"].(map[string]any); ok {
			table, err := s.requireTable(stringValue(conditionReq["TableName"]))
			if err != nil {
				return err
			}
			keySpec, _ := conditionReq["Key"].(map[string]any)
			key, err := table.lookupKey(keySpec)
			if err != nil {
				return err
			}
			if table.items[key] == nil {
				return ErrConditionalCheckFailed
			}
			continue
		}
		return ErrValidation
	}

	return nil
}

func (s *Service) CreateBackup(tableName, backupName string) (BackupDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTable(tableName)
	if err != nil {
		return BackupDescription{}, err
	}
	s.backupIDSequence++
	if strings.TrimSpace(backupName) == "" {
		backupName = fmt.Sprintf("backup-%06d", s.backupIDSequence)
	}
	backupArn := fmt.Sprintf("%s/backup/%06d", table.description.TableArn, s.backupIDSequence)
	record := &backupRecord{
		Description: BackupDescription{
			BackupArn:              backupArn,
			BackupName:             backupName,
			BackupCreationDateTime: time.Now().UTC(),
			BackupStatus:           "AVAILABLE",
			BackupType:             "USER",
			SourceTableArn:         table.description.TableArn,
			SourceTableName:        table.description.TableName,
		},
		Snapshot: snapshotFromTable(table),
	}
	s.backups[backupArn] = record
	return cloneBackupDescription(record.Description), nil
}

func (s *Service) DescribeBackup(backupArn string) (BackupDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.backups[strings.TrimSpace(backupArn)]
	if record == nil {
		return BackupDescription{}, ErrResourceNotFound
	}
	return cloneBackupDescription(record.Description), nil
}

func (s *Service) DeleteBackup(backupArn string) (BackupDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(backupArn)
	record := s.backups[key]
	if record == nil {
		return BackupDescription{}, ErrResourceNotFound
	}
	deleted := cloneBackupDescription(record.Description)
	deleted.BackupStatus = "DELETED"
	delete(s.backups, key)
	return deleted, nil
}

func (s *Service) ListBackups(tableName string, limit int, exclusiveStartBackupArn string) ([]BackupDescription, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", ErrValidation
	}

	filterTable := strings.TrimSpace(tableName)
	arns := make([]string, 0, len(s.backups))
	for arn, record := range s.backups {
		if filterTable != "" && record.Description.SourceTableName != filterTable {
			continue
		}
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	start := 0
	exclusiveStartBackupArn = strings.TrimSpace(exclusiveStartBackupArn)
	if exclusiveStartBackupArn != "" {
		found := false
		for i, arn := range arns {
			if arn == exclusiveStartBackupArn {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", ErrValidation
		}
	}
	if start > len(arns) {
		start = len(arns)
	}
	end := start + limit
	if end > len(arns) {
		end = len(arns)
	}
	out := make([]BackupDescription, 0, end-start)
	for _, arn := range arns[start:end] {
		out = append(out, cloneBackupDescription(s.backups[arn].Description))
	}
	next := ""
	if end < len(arns) {
		next = arns[end-1]
	}
	return out, next, nil
}

func (s *Service) RestoreTableFromBackup(targetTableName, backupArn string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetTableName = strings.TrimSpace(targetTableName)
	if targetTableName == "" {
		return TableDescription{}, ErrValidation
	}
	if _, exists := s.tables[targetTableName]; exists {
		return TableDescription{}, ErrResourceInUse
	}
	record := s.backups[strings.TrimSpace(backupArn)]
	if record == nil {
		return TableDescription{}, ErrResourceNotFound
	}
	table := restoreTableFromSnapshot(targetTableName, record.Snapshot)
	s.tables[targetTableName] = table
	return cloneTableDescription(table.description), nil
}

func (s *Service) RestoreTableToPointInTime(sourceTableName, targetTableName string) (TableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceTable, err := s.requireTable(sourceTableName)
	if err != nil {
		return TableDescription{}, err
	}
	targetTableName = strings.TrimSpace(targetTableName)
	if targetTableName == "" {
		return TableDescription{}, ErrValidation
	}
	if _, exists := s.tables[targetTableName]; exists {
		return TableDescription{}, ErrResourceInUse
	}
	table := restoreTableFromSnapshot(targetTableName, snapshotFromTable(sourceTable))
	s.tables[targetTableName] = table
	return cloneTableDescription(table.description), nil
}

func (s *Service) CreateGlobalTable(name string, regions []string) (GlobalTableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return GlobalTableDescription{}, ErrValidation
	}
	if _, exists := s.globalTables[name]; exists {
		return GlobalTableDescription{}, ErrResourceInUse
	}
	replicas := normalizedReplicas(regions)
	desc := GlobalTableDescription{
		GlobalTableArn:    globalTableARN(name),
		GlobalTableName:   name,
		GlobalTableStatus: "ACTIVE",
		CreationDateTime:  time.Now().UTC(),
		ReplicationGroup:  replicas,
	}
	settings := GlobalTableSettingsDescription{
		GlobalTableName: name,
		ReplicaSettings: defaultReplicaSettings(replicas),
	}
	s.globalTables[name] = &globalTableRecord{
		Description: desc,
		Settings:    settings,
	}
	return cloneGlobalTableDescription(desc), nil
}

func (s *Service) DescribeGlobalTable(name string) (GlobalTableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.globalTables[strings.TrimSpace(name)]
	if record == nil {
		return GlobalTableDescription{}, ErrResourceNotFound
	}
	return cloneGlobalTableDescription(record.Description), nil
}

func (s *Service) ListGlobalTables(limit int, exclusiveStartGlobalTableName string) ([]GlobalTableDescription, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", ErrValidation
	}

	names := make([]string, 0, len(s.globalTables))
	for name := range s.globalTables {
		names = append(names, name)
	}
	sort.Strings(names)
	start := 0
	exclusiveStartGlobalTableName = strings.TrimSpace(exclusiveStartGlobalTableName)
	if exclusiveStartGlobalTableName != "" {
		found := false
		for i, name := range names {
			if name == exclusiveStartGlobalTableName {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", ErrValidation
		}
	}
	if start > len(names) {
		start = len(names)
	}
	end := start + limit
	if end > len(names) {
		end = len(names)
	}

	out := make([]GlobalTableDescription, 0, end-start)
	for _, name := range names[start:end] {
		out = append(out, cloneGlobalTableDescription(s.globalTables[name].Description))
	}
	next := ""
	if end < len(names) {
		next = names[end-1]
	}
	return out, next, nil
}

func (s *Service) UpdateGlobalTable(name string, addRegions []string, deleteRegions []string) (GlobalTableDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.globalTables[strings.TrimSpace(name)]
	if record == nil {
		return GlobalTableDescription{}, ErrResourceNotFound
	}

	existing := map[string]ReplicaDescription{}
	for _, replica := range record.Description.ReplicationGroup {
		existing[replica.RegionName] = replica
	}
	for _, region := range addRegions {
		trimmed := strings.TrimSpace(region)
		if trimmed == "" {
			continue
		}
		existing[trimmed] = ReplicaDescription{RegionName: trimmed, ReplicaStatus: "ACTIVE"}
	}
	for _, region := range deleteRegions {
		delete(existing, strings.TrimSpace(region))
	}
	if len(existing) == 0 {
		existing[defaultRegion] = ReplicaDescription{RegionName: defaultRegion, ReplicaStatus: "ACTIVE"}
	}
	replicas := make([]ReplicaDescription, 0, len(existing))
	for _, replica := range existing {
		replicas = append(replicas, replica)
	}
	sort.Slice(replicas, func(i, j int) bool { return replicas[i].RegionName < replicas[j].RegionName })
	record.Description.ReplicationGroup = replicas
	record.Settings.ReplicaSettings = defaultReplicaSettings(replicas)
	return cloneGlobalTableDescription(record.Description), nil
}

func (s *Service) DescribeGlobalTableSettings(name string) (GlobalTableSettingsDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.globalTables[strings.TrimSpace(name)]
	if record == nil {
		return GlobalTableSettingsDescription{}, ErrResourceNotFound
	}
	return cloneGlobalTableSettings(record.Settings), nil
}

func (s *Service) UpdateGlobalTableSettings(name string, replicaSettings []map[string]any) (GlobalTableSettingsDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.globalTables[strings.TrimSpace(name)]
	if record == nil {
		return GlobalTableSettingsDescription{}, ErrResourceNotFound
	}
	if len(replicaSettings) > 0 {
		record.Settings.ReplicaSettings = cloneReplicaSettings(replicaSettings)
	}
	return cloneGlobalTableSettings(record.Settings), nil
}

func (s *Service) DescribeTableReplicaAutoScaling(tableName string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	auto := s.tableAutoScaling[table.description.TableName]
	if auto == nil {
		auto = map[string]any{
			"TableName":   table.description.TableName,
			"TableStatus": table.description.TableStatus,
			"Replicas": []map[string]any{
				{
					"RegionName":    defaultRegion,
					"ReplicaStatus": "ACTIVE",
				},
			},
		}
		s.tableAutoScaling[table.description.TableName] = auto
	}
	return cloneAnyMap(auto), nil
}

func (s *Service) UpdateTableReplicaAutoScaling(tableName string, updates map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	auto := s.tableAutoScaling[table.description.TableName]
	if auto == nil {
		auto = map[string]any{
			"TableName":   table.description.TableName,
			"TableStatus": table.description.TableStatus,
			"Replicas": []map[string]any{
				{
					"RegionName":    defaultRegion,
					"ReplicaStatus": "ACTIVE",
				},
			},
		}
	}
	for key, value := range updates {
		auto[key] = cloneAny(value)
	}
	auto["TableName"] = table.description.TableName
	s.tableAutoScaling[table.description.TableName] = auto
	return cloneAnyMap(auto), nil
}

func (s *Service) ExportTableToPointInTime(tableArn, s3Bucket, exportFormat, clientToken string) (ExportDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTableByARN(tableArn)
	if err != nil {
		return ExportDescription{}, err
	}
	tableArn = table.description.TableArn
	s3Bucket = strings.TrimSpace(s3Bucket)
	exportFormat = strings.TrimSpace(exportFormat)
	clientToken = strings.TrimSpace(clientToken)
	if strings.TrimSpace(exportFormat) == "" {
		exportFormat = "DYNAMODB_JSON"
	}
	if clientToken != "" {
		if record := s.exportTokenRecords[clientToken]; record != nil {
			if record.TableArn != tableArn || record.S3Bucket != s3Bucket || record.ExportFormat != exportFormat {
				return ExportDescription{}, ErrValidation
			}
			existing := s.exports[record.ExportArn]
			if existing != nil {
				return cloneExportDescription(existing.Description), nil
			}
		}
	}
	s.exportIDSequence++
	desc := ExportDescription{
		ExportArn:    fmt.Sprintf("%s/export/%06d", table.description.TableArn, s.exportIDSequence),
		ExportStatus: "COMPLETED",
		ExportFormat: exportFormat,
		ExportTime:   time.Now().UTC(),
		ClientToken:  clientToken,
		S3Bucket:     s3Bucket,
		TableArn:     table.description.TableArn,
	}
	s.exports[desc.ExportArn] = &exportRecord{Description: desc}
	if clientToken != "" {
		s.exportTokenRecords[clientToken] = &exportTokenRecord{
			ExportArn:    desc.ExportArn,
			TableArn:     tableArn,
			S3Bucket:     s3Bucket,
			ExportFormat: exportFormat,
		}
	}
	return cloneExportDescription(desc), nil
}

func (s *Service) DescribeExport(exportArn string) (ExportDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.exports[strings.TrimSpace(exportArn)]
	if record == nil {
		return ExportDescription{}, ErrResourceNotFound
	}
	return cloneExportDescription(record.Description), nil
}

func (s *Service) ListExports(tableArn string, limit int, exclusiveStartExportArn string) ([]ExportDescription, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", ErrValidation
	}

	filterArn := strings.TrimSpace(tableArn)
	arns := make([]string, 0, len(s.exports))
	for arn, record := range s.exports {
		if filterArn != "" && record.Description.TableArn != filterArn {
			continue
		}
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	start := 0
	exclusiveStartExportArn = strings.TrimSpace(exclusiveStartExportArn)
	if exclusiveStartExportArn != "" {
		found := false
		for i, arn := range arns {
			if arn == exclusiveStartExportArn {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", ErrValidation
		}
	}
	if start > len(arns) {
		start = len(arns)
	}
	end := start + limit
	if end > len(arns) {
		end = len(arns)
	}
	out := make([]ExportDescription, 0, end-start)
	for _, arn := range arns[start:end] {
		out = append(out, cloneExportDescription(s.exports[arn].Description))
	}
	next := ""
	if end < len(arns) {
		next = arns[end-1]
	}
	return out, next, nil
}

func (s *Service) ImportTable(tableName, inputFormat, clientToken string) (ImportDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tableName = strings.TrimSpace(tableName)
	inputFormat = strings.TrimSpace(inputFormat)
	clientToken = strings.TrimSpace(clientToken)
	if inputFormat == "" {
		inputFormat = "DYNAMODB_JSON"
	}
	if clientToken != "" {
		if tokenRecord := s.importTokenRecords[clientToken]; tokenRecord != nil {
			if tokenRecord.TableName != tableName || tokenRecord.InputFormat != inputFormat {
				return ImportDescription{}, ErrValidation
			}
			existing := s.imports[tokenRecord.ImportArn]
			if existing != nil {
				return cloneImportDescription(existing.Description), nil
			}
		}
	}
	if tableName == "" {
		s.importIDSequence++
		tableName = fmt.Sprintf("imported-table-%06d", s.importIDSequence)
	}
	if _, exists := s.tables[tableName]; exists {
		return ImportDescription{}, ErrResourceInUse
	}
	desc := TableDescription{
		AttributeDefinitions: []AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		BillingModeSummary:   &BillingModeSummary{BillingMode: "PAY_PER_REQUEST"},
		CreationDateTime:     time.Now().UTC(),
		ItemCount:            0,
		KeySchema:            []KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		TableArn:             tableARN(tableName),
		TableName:            tableName,
		TableSizeBytes:       0,
		TableStatus:          "ACTIVE",
	}
	s.tables[tableName] = &Table{description: desc, hashKey: "pk", items: map[string]map[string]any{}}

	s.importIDSequence++
	importDesc := ImportDescription{
		ImportArn:          fmt.Sprintf("%s/import/%06d", desc.TableArn, s.importIDSequence),
		ImportStatus:       "COMPLETED",
		ClientToken:        clientToken,
		InputFormat:        inputFormat,
		TableArn:           desc.TableArn,
		ProcessedSizeBytes: 0,
		ProcessedItemCount: 0,
		ImportTime:         time.Now().UTC(),
	}
	s.imports[importDesc.ImportArn] = &importRecord{Description: importDesc}
	if clientToken != "" {
		s.importTokenRecords[clientToken] = &importTokenRecord{
			ImportArn:   importDesc.ImportArn,
			TableName:   tableName,
			InputFormat: inputFormat,
		}
	}
	return cloneImportDescription(importDesc), nil
}

func (s *Service) DescribeImport(importArn string) (ImportDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.imports[strings.TrimSpace(importArn)]
	if record == nil {
		return ImportDescription{}, ErrResourceNotFound
	}
	return cloneImportDescription(record.Description), nil
}

func (s *Service) ListImports(tableArn string, limit int, exclusiveStartImportArn string) ([]ImportDescription, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", ErrValidation
	}
	filterArn := strings.TrimSpace(tableArn)
	arns := make([]string, 0, len(s.imports))
	for arn, record := range s.imports {
		if filterArn != "" && record.Description.TableArn != filterArn {
			continue
		}
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	start := 0
	exclusiveStartImportArn = strings.TrimSpace(exclusiveStartImportArn)
	if exclusiveStartImportArn != "" {
		found := false
		for i, arn := range arns {
			if arn == exclusiveStartImportArn {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", ErrValidation
		}
	}
	if start > len(arns) {
		start = len(arns)
	}
	end := start + limit
	if end > len(arns) {
		end = len(arns)
	}
	out := make([]ImportDescription, 0, end-start)
	for _, arn := range arns[start:end] {
		out = append(out, cloneImportDescription(s.imports[arn].Description))
	}
	next := ""
	if end < len(arns) {
		next = arns[end-1]
	}
	return out, next, nil
}

func (s *Service) EnableKinesisStreamingDestination(tableNameOrARN, streamArn string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTableByNameOrARN(tableNameOrARN)
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"TableName":                            table.description.TableName,
		"TableArn":                             table.description.TableArn,
		"StreamArn":                            strings.TrimSpace(streamArn),
		"DestinationStatus":                    "ACTIVE",
		"ApproximateCreationDateTimePrecision": "MICROSECOND",
	}
	s.kinesisDestinations[table.description.TableName] = config
	return cloneAnyMap(config), nil
}

func (s *Service) DisableKinesisStreamingDestination(tableNameOrARN, streamArn string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTableByNameOrARN(tableNameOrARN)
	if err != nil {
		return nil, err
	}
	config := s.kinesisDestinations[table.description.TableName]
	if config == nil {
		config = map[string]any{
			"TableName": table.description.TableName,
			"TableArn":  table.description.TableArn,
			"StreamArn": strings.TrimSpace(streamArn),
		}
	}
	config["DestinationStatus"] = "DISABLED"
	if strings.TrimSpace(streamArn) != "" {
		config["StreamArn"] = strings.TrimSpace(streamArn)
	}
	s.kinesisDestinations[table.description.TableName] = config
	return cloneAnyMap(config), nil
}

func (s *Service) UpdateKinesisStreamingDestination(tableNameOrARN, streamArn string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTableByNameOrARN(tableNameOrARN)
	if err != nil {
		return nil, err
	}
	config := s.kinesisDestinations[table.description.TableName]
	if config == nil {
		config = map[string]any{
			"TableName": table.description.TableName,
			"TableArn":  table.description.TableArn,
		}
	}
	if strings.TrimSpace(streamArn) != "" {
		config["StreamArn"] = strings.TrimSpace(streamArn)
	}
	config["DestinationStatus"] = "ACTIVE"
	s.kinesisDestinations[table.description.TableName] = config
	return cloneAnyMap(config), nil
}

func (s *Service) DescribeKinesisStreamingDestination(tableNameOrARN string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, err := s.requireTableByNameOrARN(tableNameOrARN)
	if err != nil {
		return nil, err
	}
	config := s.kinesisDestinations[table.description.TableName]
	if config == nil {
		config = map[string]any{
			"TableName":         table.description.TableName,
			"TableArn":          table.description.TableArn,
			"DestinationStatus": "DISABLED",
		}
	}
	return map[string]any{
		"TableName":                     table.description.TableName,
		"KinesisDataStreamDestinations": []map[string]any{cloneAnyMap(config)},
	}, nil
}

func (s *Service) DescribeEndpoints() map[string]any {
	return map[string]any{
		"Endpoints": []map[string]any{
			{
				"Address":              "dynamodb.us-east-1.amazonaws.com",
				"CachePeriodInMinutes": int64(60),
			},
		},
	}
}

func (s *Service) UpdateContinuousBackups(tableName string, enabled bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	status := "DISABLED"
	if enabled {
		status = "ENABLED"
	}
	description := map[string]any{
		"TableName":               table.description.TableName,
		"ContinuousBackupsStatus": status,
		"PointInTimeRecoveryDescription": map[string]any{
			"PointInTimeRecoveryStatus": status,
		},
	}
	s.continuousBackups[table.description.TableName] = description
	return cloneAnyMap(description), nil
}

func (s *Service) DescribeContinuousBackups(tableName string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	description := s.continuousBackups[table.description.TableName]
	if description == nil {
		description = map[string]any{
			"TableName":               table.description.TableName,
			"ContinuousBackupsStatus": "DISABLED",
			"PointInTimeRecoveryDescription": map[string]any{
				"PointInTimeRecoveryStatus": "DISABLED",
			},
		}
		s.continuousBackups[table.description.TableName] = description
	}
	return cloneAnyMap(description), nil
}

func (s *Service) UpdateTimeToLive(tableName, attributeName string, enabled bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	spec := map[string]any{
		"AttributeName": strings.TrimSpace(attributeName),
		"Enabled":       enabled,
	}
	s.timeToLive[table.description.TableName] = spec
	return cloneAnyMap(spec), nil
}

func (s *Service) DescribeTimeToLive(tableName string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	spec := s.timeToLive[table.description.TableName]
	if spec == nil {
		spec = map[string]any{
			"AttributeName": "",
			"Enabled":       false,
		}
		s.timeToLive[table.description.TableName] = spec
	}
	status := "DISABLED"
	if boolValue(spec["Enabled"]) {
		status = "ENABLED"
	}
	return map[string]any{
		"AttributeName":    stringValue(spec["AttributeName"]),
		"TimeToLiveStatus": status,
	}, nil
}

func (s *Service) UpdateContributorInsights(tableName, indexName string, enabled bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	status := "DISABLED"
	if enabled {
		status = "ENABLED"
	}
	key := contributorInsightsKey(table.description.TableName, indexName)
	description := map[string]any{
		"TableName":                 table.description.TableName,
		"IndexName":                 strings.TrimSpace(indexName),
		"ContributorInsightsStatus": status,
	}
	s.contributorInsights[key] = description
	return cloneAnyMap(description), nil
}

func (s *Service) DescribeContributorInsights(tableName, indexName string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table, err := s.requireTable(tableName)
	if err != nil {
		return nil, err
	}
	key := contributorInsightsKey(table.description.TableName, indexName)
	description := s.contributorInsights[key]
	if description == nil {
		description = map[string]any{
			"TableName":                 table.description.TableName,
			"IndexName":                 strings.TrimSpace(indexName),
			"ContributorInsightsStatus": "DISABLED",
		}
		s.contributorInsights[key] = description
	}
	return cloneAnyMap(description), nil
}

func (s *Service) ListContributorInsights(tableName string) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tableName = strings.TrimSpace(tableName)
	if tableName != "" {
		if _, err := s.requireTable(tableName); err != nil {
			return nil, err
		}
	}
	keys := make([]string, 0, len(s.contributorInsights))
	for key := range s.contributorInsights {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		record := s.contributorInsights[key]
		if tableName != "" && stringValue(record["TableName"]) != tableName {
			continue
		}
		out = append(out, cloneAnyMap(record))
	}
	return out, nil
}

func (s *Service) TagResource(resourceArn string, tags []map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return ErrResourceNotFound
	}
	existing := s.resourceTags[resourceArn]
	if existing == nil {
		existing = map[string]string{}
	}
	for _, tag := range tags {
		key := strings.TrimSpace(tag["Key"])
		if key == "" {
			continue
		}
		existing[key] = strings.TrimSpace(tag["Value"])
	}
	s.resourceTags[resourceArn] = existing
	return nil
}

func (s *Service) UntagResource(resourceArn string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return ErrResourceNotFound
	}
	existing := s.resourceTags[resourceArn]
	if existing == nil {
		return nil
	}
	for _, key := range tagKeys {
		delete(existing, strings.TrimSpace(key))
	}
	return nil
}

func (s *Service) ListTagsOfResource(resourceArn string) ([]map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return nil, ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return nil, ErrResourceNotFound
	}
	existing := s.resourceTags[resourceArn]
	if existing == nil {
		return []map[string]string{}, nil
	}
	keys := make([]string, 0, len(existing))
	for key := range existing {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{
			"Key":   key,
			"Value": existing[key],
		})
	}
	return out, nil
}

func (s *Service) PutResourcePolicy(resourceArn, policy, expectedRevisionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceArn = strings.TrimSpace(resourceArn)
	policy = strings.TrimSpace(policy)
	expectedRevisionID = strings.TrimSpace(expectedRevisionID)
	if resourceArn == "" || policy == "" {
		return "", ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return "", ErrResourceNotFound
	}
	var policyJSON any
	if err := json.Unmarshal([]byte(policy), &policyJSON); err != nil {
		return "", ErrValidation
	}

	record := s.resourcePolicies[resourceArn]
	if expectedRevisionID != "" {
		if record == nil || record.RevisionID != expectedRevisionID {
			return "", ErrConditionalCheckFailed
		}
	}
	s.resourcePolicySeq++
	revisionID := fmt.Sprintf("r-%06d", s.resourcePolicySeq)
	s.resourcePolicies[resourceArn] = &resourcePolicyRecord{
		Policy:     policy,
		RevisionID: revisionID,
	}
	return revisionID, nil
}

func (s *Service) GetResourcePolicy(resourceArn string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return "", "", ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return "", "", ErrResourceNotFound
	}
	record := s.resourcePolicies[resourceArn]
	if record == nil {
		return "", "", ErrResourceNotFound
	}
	return record.Policy, record.RevisionID, nil
}

func (s *Service) DeleteResourcePolicy(resourceArn, expectedRevisionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceArn = strings.TrimSpace(resourceArn)
	expectedRevisionID = strings.TrimSpace(expectedRevisionID)
	if resourceArn == "" {
		return "", ErrValidation
	}
	if !s.resourceExists(resourceArn) {
		return "", ErrResourceNotFound
	}
	record := s.resourcePolicies[resourceArn]
	if record == nil {
		return "", ErrResourceNotFound
	}
	if expectedRevisionID != "" && record.RevisionID != expectedRevisionID {
		return "", ErrConditionalCheckFailed
	}
	delete(s.resourcePolicies, resourceArn)
	return record.RevisionID, nil
}

func (s *Service) requireTable(name string) (*Table, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrValidation
	}
	table := s.tables[name]
	if table == nil {
		return nil, ErrResourceNotFound
	}
	return table, nil
}

func (s *Service) requireTableByARN(tableArn string) (*Table, error) {
	tableName := tableNameFromARN(tableArn)
	if tableName == "" {
		return nil, ErrValidation
	}
	return s.requireTable(tableName)
}

func (s *Service) requireTableByNameOrARN(nameOrArn string) (*Table, error) {
	value := strings.TrimSpace(nameOrArn)
	if value == "" {
		return nil, ErrValidation
	}
	if strings.HasPrefix(value, "arn:") {
		return s.requireTableByARN(value)
	}
	return s.requireTable(value)
}

func (s *Service) resourceExists(resourceArn string) bool {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return false
	}
	if tableName := tableNameFromARN(resourceArn); tableName != "" {
		if _, ok := s.tables[tableName]; ok {
			return true
		}
	}
	if strings.Contains(resourceArn, ":global-table/") {
		name := strings.TrimSpace(resourceArn[strings.LastIndex(resourceArn, "/")+1:])
		_, ok := s.globalTables[name]
		return ok
	}
	if _, ok := s.backups[resourceArn]; ok {
		return true
	}
	if _, ok := s.exports[resourceArn]; ok {
		return true
	}
	if _, ok := s.imports[resourceArn]; ok {
		return true
	}
	return false
}

func (t *Table) itemKey(item map[string]any) (string, error) {
	hashPart, ok := attrCanonical(item[t.hashKey])
	if !ok {
		return "", ErrValidation
	}
	parts := []string{"H=" + hashPart}
	if t.rangeKey != "" {
		rangePart, ok := attrCanonical(item[t.rangeKey])
		if !ok {
			return "", ErrValidation
		}
		parts = append(parts, "R="+rangePart)
	}
	return strings.Join(parts, "|") + "|", nil
}

func (t *Table) lookupKey(keySpec map[string]any) (string, error) {
	if len(keySpec) == 0 {
		return "", ErrValidation
	}
	hashPart, ok := attrCanonical(keySpec[t.hashKey])
	if !ok {
		return "", ErrValidation
	}
	parts := []string{"H=" + hashPart}
	if t.rangeKey != "" {
		rangePart, ok := attrCanonical(keySpec[t.rangeKey])
		if !ok {
			return "", ErrValidation
		}
		parts = append(parts, "R="+rangePart)
	}
	return strings.Join(parts, "|") + "|", nil
}

func validateKeySchema(keys []KeySchemaElement) (string, string, error) {
	var hashKey string
	var rangeKey string
	for _, key := range keys {
		attr := strings.TrimSpace(key.AttributeName)
		typ := strings.ToUpper(strings.TrimSpace(key.KeyType))
		if attr == "" || typ == "" {
			return "", "", ErrValidation
		}
		switch typ {
		case "HASH":
			if hashKey != "" {
				return "", "", ErrValidation
			}
			hashKey = attr
		case "RANGE":
			if rangeKey != "" {
				return "", "", ErrValidation
			}
			rangeKey = attr
		default:
			return "", "", ErrValidation
		}
	}
	if hashKey == "" {
		return "", "", ErrValidation
	}
	return hashKey, rangeKey, nil
}

func buildQueryFilters(input QueryInput, table *Table) ([]queryFilter, error) {
	if len(input.KeyConditions) > 0 {
		filters := make([]queryFilter, 0, len(input.KeyConditions))
		for rawName, rawCond := range input.KeyConditions {
			condMap, ok := rawCond.(map[string]any)
			if !ok {
				return nil, ErrValidation
			}
			operator := strings.ToUpper(strings.TrimSpace(stringValue(condMap["ComparisonOperator"])))
			if operator != "" && operator != "EQ" {
				return nil, ErrValidation
			}
			list, ok := condMap["AttributeValueList"].([]any)
			if !ok || len(list) == 0 {
				return nil, ErrValidation
			}
			filters = append(filters, queryFilter{name: strings.TrimSpace(rawName), expected: cloneAny(list[0])})
		}
		return filters, nil
	}
	expr := strings.TrimSpace(input.KeyConditionExpression)
	if expr == "" {
		return nil, ErrValidation
	}
	parts := strings.Split(expr, "AND")
	filters := make([]queryFilter, 0, len(parts))
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		tokens := strings.Split(segment, "=")
		if len(tokens) != 2 {
			return nil, ErrValidation
		}
		left := strings.TrimSpace(tokens[0])
		right := strings.TrimSpace(tokens[1])
		if strings.HasPrefix(left, "#") {
			left = strings.TrimSpace(input.ExpressionAttributeNames[left])
		}
		if !strings.HasPrefix(right, ":") {
			return nil, ErrValidation
		}
		expected := input.ExpressionAttributeValues[right]
		if strings.TrimSpace(left) == "" || expected == nil {
			return nil, ErrValidation
		}
		filters = append(filters, queryFilter{name: left, expected: cloneAny(expected)})
	}
	if len(filters) == 0 {
		return nil, ErrValidation
	}
	return filters, nil
}

type queryFilter struct {
	name     string
	expected any
}

func matchesFilters(item map[string]any, filters []queryFilter) bool {
	for _, filter := range filters {
		value, ok := item[filter.name]
		if !ok {
			return false
		}
		left, ok := attrCanonical(value)
		if !ok {
			return false
		}
		right, ok := attrCanonical(filter.expected)
		if !ok {
			return false
		}
		if left != right {
			return false
		}
	}
	return true
}

func applyUpdateExpression(item map[string]any, expr string, names map[string]string, values map[string]any) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}
	upper := strings.ToUpper(expr)
	setIndex := strings.Index(upper, "SET ")
	removeIndex := strings.Index(upper, " REMOVE ")

	if setIndex >= 0 {
		setBody := ""
		if removeIndex > setIndex {
			setBody = strings.TrimSpace(expr[setIndex+4 : removeIndex])
		} else {
			setBody = strings.TrimSpace(expr[setIndex+4:])
		}
		if setBody != "" {
			for _, assignment := range strings.Split(setBody, ",") {
				pair := strings.Split(strings.TrimSpace(assignment), "=")
				if len(pair) != 2 {
					return ErrValidation
				}
				name := strings.TrimSpace(pair[0])
				if strings.HasPrefix(name, "#") {
					name = strings.TrimSpace(names[name])
				}
				valueToken := strings.TrimSpace(pair[1])
				value := values[valueToken]
				if name == "" || value == nil {
					return ErrValidation
				}
				item[name] = cloneAny(value)
			}
		}
	}
	if removeIndex >= 0 {
		removeBody := strings.TrimSpace(expr[removeIndex+8:])
		for _, rawName := range strings.Split(removeBody, ",") {
			name := strings.TrimSpace(rawName)
			if strings.HasPrefix(name, "#") {
				name = strings.TrimSpace(names[name])
			}
			if name == "" {
				return ErrValidation
			}
			delete(item, name)
		}
	}
	if setIndex < 0 && removeIndex < 0 {
		return ErrValidation
	}
	return nil
}

func refreshTableMetrics(table *Table) {
	table.description.ItemCount = int64(len(table.items))
	var total int64
	for _, item := range table.items {
		if encoded, err := json.Marshal(item); err == nil {
			total += int64(len(encoded))
		}
	}
	table.description.TableSizeBytes = total
}

func tableARN(name string) string {
	return fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", defaultRegion, defaultAccountID, name)
}

func cloneTableDescription(in TableDescription) TableDescription {
	out := in
	out.AttributeDefinitions = cloneAttrDefinitions(in.AttributeDefinitions)
	out.KeySchema = cloneKeySchema(in.KeySchema)
	if in.BillingModeSummary != nil {
		bm := *in.BillingModeSummary
		out.BillingModeSummary = &bm
	}
	return out
}

func cloneAttrDefinitions(in []AttributeDefinition) []AttributeDefinition {
	if len(in) == 0 {
		return nil
	}
	out := make([]AttributeDefinition, 0, len(in))
	for _, item := range in {
		name := strings.TrimSpace(item.AttributeName)
		typ := strings.TrimSpace(item.AttributeType)
		if name == "" || typ == "" {
			continue
		}
		out = append(out, AttributeDefinition{AttributeName: name, AttributeType: typ})
	}
	return out
}

func cloneKeySchema(in []KeySchemaElement) []KeySchemaElement {
	if len(in) == 0 {
		return nil
	}
	out := make([]KeySchemaElement, 0, len(in))
	for _, item := range in {
		name := strings.TrimSpace(item.AttributeName)
		typ := strings.TrimSpace(item.KeyType)
		if name == "" || typ == "" {
			continue
		}
		out = append(out, KeySchemaElement{AttributeName: name, KeyType: typ})
	}
	return out
}

func cloneItem(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneAny(v)
	}
	return out
}

func cloneAny(in any) any {
	if in == nil {
		return nil
	}
	encoded, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out any
	if err := json.Unmarshal(encoded, &out); err != nil {
		return in
	}
	return out
}

func attrCanonical(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return "", false
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		raw := m[k]
		switch value := raw.(type) {
		case string:
			if value == "" {
				continue
			}
			return k + ":" + value, true
		case []any:
			encoded, _ := json.Marshal(value)
			if string(encoded) == "[]" {
				continue
			}
			return k + ":" + string(encoded), true
		case map[string]any:
			encoded, _ := json.Marshal(value)
			if string(encoded) == "{}" {
				continue
			}
			return k + ":" + string(encoded), true
		case bool:
			if !value {
				continue
			}
			return k + ":true", true
		default:
			if raw == nil {
				continue
			}
			encoded, _ := json.Marshal(raw)
			return k + ":" + string(encoded), true
		}
	}
	return "", false
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolValue(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func tableNameFromARN(tableArn string) string {
	tableArn = strings.TrimSpace(tableArn)
	if tableArn == "" {
		return ""
	}
	marker := ":table/"
	idx := strings.Index(tableArn, marker)
	if idx < 0 {
		return ""
	}
	rest := tableArn[idx+len(marker):]
	if rest == "" {
		return ""
	}
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	return strings.TrimSpace(rest)
}

func contributorInsightsKey(tableName, indexName string) string {
	return strings.TrimSpace(tableName) + "|" + strings.TrimSpace(indexName)
}

func firstToken(statement string) string {
	fields := strings.Fields(strings.TrimSpace(statement))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func extractStatementTableName(statement, keyword string) string {
	fields := strings.Fields(statement)
	if len(fields) == 0 {
		return ""
	}
	upperKeyword := strings.ToUpper(strings.TrimSpace(keyword))
	for i, field := range fields {
		if strings.ToUpper(strings.TrimSpace(field)) != upperKeyword {
			continue
		}
		if i+1 >= len(fields) {
			return ""
		}
		raw := strings.TrimSpace(fields[i+1])
		raw = strings.Trim(raw, ",")
		raw = strings.Trim(raw, "\"`")
		return strings.TrimSpace(raw)
	}
	return ""
}

func toStringMap(v any) map[string]string {
	in, _ := v.(map[string]any)
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		s, _ := value.(string)
		out[key] = strings.TrimSpace(s)
	}
	return out
}

func toAnyMap(v any) map[string]any {
	in, _ := v.(map[string]any)
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAny(value)
	}
	return out
}

func snapshotFromTable(table *Table) tableSnapshot {
	return tableSnapshot{
		Description: cloneTableDescription(table.description),
		HashKey:     table.hashKey,
		RangeKey:    table.rangeKey,
		Items:       cloneItemsMap(table.items),
	}
}

func restoreTableFromSnapshot(targetName string, snapshot tableSnapshot) *Table {
	desc := cloneTableDescription(snapshot.Description)
	desc.TableName = targetName
	desc.TableArn = tableARN(targetName)
	desc.CreationDateTime = time.Now().UTC()
	desc.TableStatus = "ACTIVE"
	table := &Table{
		description: desc,
		hashKey:     snapshot.HashKey,
		rangeKey:    snapshot.RangeKey,
		items:       cloneItemsMap(snapshot.Items),
	}
	refreshTableMetrics(table)
	return table
}

func cloneItemsMap(in map[string]map[string]any) map[string]map[string]any {
	if len(in) == 0 {
		return map[string]map[string]any{}
	}
	out := make(map[string]map[string]any, len(in))
	for key, item := range in {
		out[key] = cloneItem(item)
	}
	return out
}

func cloneBackupDescription(in BackupDescription) BackupDescription {
	return in
}

func cloneExportDescription(in ExportDescription) ExportDescription {
	return in
}

func cloneImportDescription(in ImportDescription) ImportDescription {
	return in
}

func globalTableARN(name string) string {
	return fmt.Sprintf("arn:aws:dynamodb:%s:%s:global-table/%s", defaultRegion, defaultAccountID, name)
}

func normalizedReplicas(regions []string) []ReplicaDescription {
	seen := map[string]struct{}{}
	out := make([]ReplicaDescription, 0, len(regions))
	for _, region := range regions {
		trimmed := strings.TrimSpace(region)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, ReplicaDescription{RegionName: trimmed, ReplicaStatus: "ACTIVE"})
	}
	if len(out) == 0 {
		out = append(out, ReplicaDescription{RegionName: defaultRegion, ReplicaStatus: "ACTIVE"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RegionName < out[j].RegionName })
	return out
}

func defaultReplicaSettings(replicas []ReplicaDescription) []map[string]any {
	out := make([]map[string]any, 0, len(replicas))
	for _, replica := range replicas {
		out = append(out, map[string]any{
			"RegionName": replica.RegionName,
		})
	}
	return out
}

func cloneGlobalTableDescription(in GlobalTableDescription) GlobalTableDescription {
	out := in
	if len(in.ReplicationGroup) > 0 {
		out.ReplicationGroup = append([]ReplicaDescription(nil), in.ReplicationGroup...)
	}
	return out
}

func cloneGlobalTableSettings(in GlobalTableSettingsDescription) GlobalTableSettingsDescription {
	out := in
	out.ReplicaSettings = cloneReplicaSettings(in.ReplicaSettings)
	return out
}

func cloneReplicaSettings(in []map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneAnyMap(item))
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAny(value)
	}
	return out
}
