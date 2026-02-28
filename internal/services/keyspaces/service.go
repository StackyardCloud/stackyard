package keyspaces

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrNotFound         = errors.New("resource not found")
	ErrConflict         = errors.New("resource conflict")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type ReplicationGroupStatus struct {
	Region                    string
	KeyspaceStatus            string
	TablesReplicationProgress float64
}

type ReplicationSpecification struct {
	ReplicationStrategy string
	RegionList          []string
}

type Keyspace struct {
	KeyspaceName             string
	ResourceARN              string
	ReplicationStrategy      string
	ReplicationRegions       []string
	ReplicationGroupStatuses []ReplicationGroupStatus
}

type KeyspaceSummary struct {
	KeyspaceName        string
	ResourceARN         string
	ReplicationStrategy string
	ReplicationRegions  []string
}

type CreateTableInput struct {
	KeyspaceName             string
	TableName                string
	SchemaDefinition         map[string]any
	Comment                  map[string]any
	CapacitySpecification    map[string]any
	EncryptionSpecification  map[string]any
	PointInTimeRecovery      map[string]any
	TTL                      map[string]any
	DefaultTimeToLive        *int
	Tags                     map[string]string
	ClientSideTimestamps     map[string]any
	AutoScalingSpecification map[string]any
	ReplicaSpecifications    []map[string]any
}

type UpdateTableInput struct {
	KeyspaceName             string
	TableName                string
	AddColumns               []map[string]any
	CapacitySpecification    map[string]any
	EncryptionSpecification  map[string]any
	PointInTimeRecovery      map[string]any
	TTL                      map[string]any
	DefaultTimeToLive        *int
	ClientSideTimestamps     map[string]any
	AutoScalingSpecification map[string]any
	ReplicaSpecifications    []map[string]any
}

type RestoreTableInput struct {
	SourceKeyspaceName              string
	SourceTableName                 string
	TargetKeyspaceName              string
	TargetTableName                 string
	RestoreTimestamp                *time.Time
	CapacitySpecificationOverride   map[string]any
	EncryptionSpecificationOverride map[string]any
	PointInTimeRecoveryOverride     map[string]any
	TagsOverride                    map[string]string
	AutoScalingSpecification        map[string]any
	ReplicaSpecifications           []map[string]any
}

type Table struct {
	KeyspaceName             string
	TableName                string
	ResourceARN              string
	CreationTimestamp        time.Time
	Status                   string
	SchemaDefinition         map[string]any
	CapacitySpecification    map[string]any
	EncryptionSpecification  map[string]any
	PointInTimeRecovery      map[string]any
	TTL                      map[string]any
	DefaultTimeToLive        *int
	Comment                  map[string]any
	ClientSideTimestamps     map[string]any
	AutoScalingSpecification map[string]any
	ReplicaSpecifications    []map[string]any
}

type TableSummary struct {
	KeyspaceName string
	TableName    string
	ResourceARN  string
}

type Type struct {
	KeyspaceName          string
	TypeName              string
	FieldDefinitions      []map[string]any
	LastModifiedTimestamp time.Time
	Status                string
	DirectReferringTables []string
	DirectParentTypes     []string
	MaxNestingDepth       int
	KeyspaceARN           string
}

type TableAutoScalingSettings struct {
	KeyspaceName             string
	TableName                string
	ResourceARN              string
	AutoScalingSpecification map[string]any
	ReplicaSpecifications    []map[string]any
}

type Service struct {
	mu           sync.Mutex
	keyspaces    map[string]*Keyspace
	tables       map[string]map[string]*Table
	types        map[string]map[string]*Type
	resourceTags map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		keyspaces:    map[string]*Keyspace{},
		tables:       map[string]map[string]*Table{},
		types:        map[string]map[string]*Type{},
		resourceTags: map[string]map[string]string{},
	}
}

func (s *Service) CreateKeyspace(name string, spec ReplicationSpecification, tags map[string]string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidParameter
	}
	if _, exists := s.keyspaces[name]; exists {
		return "", ErrAlreadyExists
	}

	replicationStrategy, regions := normalizeReplicationSpecification(spec)
	keyspace := &Keyspace{
		KeyspaceName:             name,
		ResourceARN:              keyspaceARN(name),
		ReplicationStrategy:      replicationStrategy,
		ReplicationRegions:       cloneStrings(regions),
		ReplicationGroupStatuses: buildReplicationGroupStatuses(regions),
	}
	s.keyspaces[name] = keyspace
	s.resourceTags[keyspace.ResourceARN] = cloneStringMap(tags)
	return keyspace.ResourceARN, nil
}

func (s *Service) GetKeyspace(name string) (Keyspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return Keyspace{}, ErrInvalidParameter
	}
	keyspace, ok := s.keyspaces[name]
	if !ok {
		return Keyspace{}, ErrNotFound
	}
	return cloneKeyspace(keyspace), nil
}

func (s *Service) ListKeyspaces(nextToken string, maxResults int) ([]KeyspaceSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, err := parseOffsetToken(nextToken)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 1 {
		return nil, "", ErrInvalidParameter
	}

	names := make([]string, 0, len(s.keyspaces))
	for name := range s.keyspaces {
		names = append(names, name)
	}
	sort.Strings(names)

	if start > len(names) {
		return []KeyspaceSummary{}, "", nil
	}
	end := start + maxResults
	if end > len(names) {
		end = len(names)
	}

	out := make([]KeyspaceSummary, 0, end-start)
	for _, name := range names[start:end] {
		ks := s.keyspaces[name]
		out = append(out, KeyspaceSummary{
			KeyspaceName:        ks.KeyspaceName,
			ResourceARN:         ks.ResourceARN,
			ReplicationStrategy: ks.ReplicationStrategy,
			ReplicationRegions:  cloneStrings(ks.ReplicationRegions),
		})
	}

	var outToken string
	if end < len(names) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *Service) UpdateKeyspace(name string, spec ReplicationSpecification) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidParameter
	}
	keyspace, ok := s.keyspaces[name]
	if !ok {
		return "", ErrNotFound
	}

	if strings.TrimSpace(spec.ReplicationStrategy) == "" && len(spec.RegionList) == 0 {
		return keyspace.ResourceARN, nil
	}

	replicationStrategy, regions := normalizeReplicationSpecification(spec)
	keyspace.ReplicationStrategy = replicationStrategy
	keyspace.ReplicationRegions = cloneStrings(regions)
	keyspace.ReplicationGroupStatuses = buildReplicationGroupStatuses(regions)
	return keyspace.ResourceARN, nil
}

func (s *Service) DeleteKeyspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	if _, ok := s.keyspaces[name]; !ok {
		return ErrNotFound
	}
	for _, table := range s.tables[name] {
		delete(s.resourceTags, table.ResourceARN)
	}
	keyspaceARN := s.keyspaces[name].ResourceARN
	delete(s.keyspaces, name)
	delete(s.tables, name)
	delete(s.types, name)
	delete(s.resourceTags, keyspaceARN)
	return nil
}

func (s *Service) CreateTable(input CreateTableInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName := strings.TrimSpace(input.KeyspaceName)
	tableName := strings.TrimSpace(input.TableName)
	if keyspaceName == "" || tableName == "" {
		return "", ErrInvalidParameter
	}
	if _, ok := s.keyspaces[keyspaceName]; !ok {
		return "", ErrNotFound
	}
	if _, ok := s.tables[keyspaceName]; !ok {
		s.tables[keyspaceName] = map[string]*Table{}
	}
	if _, exists := s.tables[keyspaceName][tableName]; exists {
		return "", ErrAlreadyExists
	}

	now := time.Now().UTC()
	table := &Table{
		KeyspaceName:             keyspaceName,
		TableName:                tableName,
		ResourceARN:              tableARN(keyspaceName, tableName),
		CreationTimestamp:        now,
		Status:                   "ACTIVE",
		SchemaDefinition:         cloneMap(input.SchemaDefinition),
		CapacitySpecification:    defaultIfNilMap(input.CapacitySpecification, map[string]any{"throughputMode": "PAY_PER_REQUEST"}),
		EncryptionSpecification:  defaultIfNilMap(input.EncryptionSpecification, map[string]any{"type": "AWS_OWNED_KMS_KEY"}),
		PointInTimeRecovery:      defaultIfNilMap(input.PointInTimeRecovery, map[string]any{"status": "DISABLED"}),
		TTL:                      defaultIfNilMap(input.TTL, map[string]any{"status": "DISABLED"}),
		Comment:                  cloneMap(input.Comment),
		ClientSideTimestamps:     defaultIfNilMap(input.ClientSideTimestamps, map[string]any{"status": "DISABLED"}),
		AutoScalingSpecification: cloneMap(input.AutoScalingSpecification),
		ReplicaSpecifications:    cloneSliceOfMaps(input.ReplicaSpecifications),
	}
	if input.DefaultTimeToLive != nil {
		val := *input.DefaultTimeToLive
		table.DefaultTimeToLive = &val
	}

	s.tables[keyspaceName][tableName] = table
	s.resourceTags[table.ResourceARN] = cloneStringMap(input.Tags)
	return table.ResourceARN, nil
}

func (s *Service) GetTable(keyspaceName, tableName string) (Table, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	tableName = strings.TrimSpace(tableName)
	if keyspaceName == "" || tableName == "" {
		return Table{}, ErrInvalidParameter
	}
	keyspaceTables, ok := s.tables[keyspaceName]
	if !ok {
		return Table{}, ErrNotFound
	}
	table, ok := keyspaceTables[tableName]
	if !ok {
		return Table{}, ErrNotFound
	}
	return cloneTable(table), nil
}

func (s *Service) ListTables(keyspaceName, nextToken string, maxResults int) ([]TableSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, err := parseOffsetToken(nextToken)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 1 {
		return nil, "", ErrInvalidParameter
	}

	keyspaceName = strings.TrimSpace(keyspaceName)
	keyspaceNames := make([]string, 0)
	if keyspaceName != "" {
		if _, exists := s.keyspaces[keyspaceName]; !exists {
			return nil, "", ErrNotFound
		}
		keyspaceNames = append(keyspaceNames, keyspaceName)
	} else {
		for name := range s.keyspaces {
			keyspaceNames = append(keyspaceNames, name)
		}
		sort.Strings(keyspaceNames)
	}

	all := make([]TableSummary, 0)
	for _, ksName := range keyspaceNames {
		keyspaceTables := s.tables[ksName]
		names := make([]string, 0, len(keyspaceTables))
		for tableName := range keyspaceTables {
			names = append(names, tableName)
		}
		sort.Strings(names)
		for _, tableName := range names {
			table := keyspaceTables[tableName]
			all = append(all, TableSummary{
				KeyspaceName: table.KeyspaceName,
				TableName:    table.TableName,
				ResourceARN:  table.ResourceARN,
			})
		}
	}

	if start > len(all) {
		return []TableSummary{}, "", nil
	}
	end := start + maxResults
	if end > len(all) {
		end = len(all)
	}
	out := make([]TableSummary, 0, end-start)
	out = append(out, all[start:end]...)

	var outToken string
	if end < len(all) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *Service) UpdateTable(input UpdateTableInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName := strings.TrimSpace(input.KeyspaceName)
	tableName := strings.TrimSpace(input.TableName)
	if keyspaceName == "" || tableName == "" {
		return "", ErrInvalidParameter
	}
	keyspaceTables, ok := s.tables[keyspaceName]
	if !ok {
		return "", ErrNotFound
	}
	table, ok := keyspaceTables[tableName]
	if !ok {
		return "", ErrNotFound
	}

	if len(input.AddColumns) > 0 {
		existing := map[string]struct{}{}
		allColumns := make([]any, 0)
		if rawColumns, ok := table.SchemaDefinition["allColumns"]; ok {
			switch cols := rawColumns.(type) {
			case []any:
				allColumns = append(allColumns, cols...)
			case []map[string]any:
				for _, col := range cols {
					allColumns = append(allColumns, col)
				}
			}
		}
		for _, col := range allColumns {
			colMap, _ := col.(map[string]any)
			name, _ := colMap["name"].(string)
			name = strings.TrimSpace(name)
			if name != "" {
				existing[name] = struct{}{}
			}
		}
		for _, col := range input.AddColumns {
			name, _ := col["name"].(string)
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := existing[name]; ok {
				continue
			}
			existing[name] = struct{}{}
			allColumns = append(allColumns, cloneMap(col))
		}
		if table.SchemaDefinition == nil {
			table.SchemaDefinition = map[string]any{}
		}
		table.SchemaDefinition["allColumns"] = allColumns
	}

	if input.CapacitySpecification != nil {
		table.CapacitySpecification = cloneMap(input.CapacitySpecification)
	}
	if input.EncryptionSpecification != nil {
		table.EncryptionSpecification = cloneMap(input.EncryptionSpecification)
	}
	if input.PointInTimeRecovery != nil {
		table.PointInTimeRecovery = cloneMap(input.PointInTimeRecovery)
	}
	if input.TTL != nil {
		table.TTL = cloneMap(input.TTL)
	}
	if input.DefaultTimeToLive != nil {
		val := *input.DefaultTimeToLive
		table.DefaultTimeToLive = &val
	}
	if input.ClientSideTimestamps != nil {
		table.ClientSideTimestamps = cloneMap(input.ClientSideTimestamps)
	}
	if input.AutoScalingSpecification != nil {
		table.AutoScalingSpecification = cloneMap(input.AutoScalingSpecification)
	}
	if input.ReplicaSpecifications != nil {
		table.ReplicaSpecifications = cloneSliceOfMaps(input.ReplicaSpecifications)
	}
	table.Status = "ACTIVE"
	return table.ResourceARN, nil
}

func (s *Service) DeleteTable(keyspaceName, tableName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	tableName = strings.TrimSpace(tableName)
	if keyspaceName == "" || tableName == "" {
		return ErrInvalidParameter
	}
	keyspaceTables, ok := s.tables[keyspaceName]
	if !ok {
		return ErrNotFound
	}
	if _, ok := keyspaceTables[tableName]; !ok {
		return ErrNotFound
	}
	tableARN := keyspaceTables[tableName].ResourceARN
	delete(keyspaceTables, tableName)
	delete(s.resourceTags, tableARN)
	return nil
}

func (s *Service) RestoreTable(input RestoreTableInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceKeyspaceName := strings.TrimSpace(input.SourceKeyspaceName)
	sourceTableName := strings.TrimSpace(input.SourceTableName)
	targetKeyspaceName := strings.TrimSpace(input.TargetKeyspaceName)
	targetTableName := strings.TrimSpace(input.TargetTableName)
	if sourceKeyspaceName == "" || sourceTableName == "" || targetKeyspaceName == "" || targetTableName == "" {
		return "", ErrInvalidParameter
	}
	if input.RestoreTimestamp != nil && input.RestoreTimestamp.After(time.Now().UTC()) {
		return "", ErrInvalidParameter
	}

	sourceTables, ok := s.tables[sourceKeyspaceName]
	if !ok {
		return "", ErrNotFound
	}
	sourceTable, ok := sourceTables[sourceTableName]
	if !ok {
		return "", ErrNotFound
	}
	if _, ok := s.keyspaces[targetKeyspaceName]; !ok {
		return "", ErrNotFound
	}
	if _, ok := s.tables[targetKeyspaceName]; !ok {
		s.tables[targetKeyspaceName] = map[string]*Table{}
	}
	if _, exists := s.tables[targetKeyspaceName][targetTableName]; exists {
		return "", ErrAlreadyExists
	}

	copied := cloneTable(sourceTable)
	copied.KeyspaceName = targetKeyspaceName
	copied.TableName = targetTableName
	copied.ResourceARN = tableARN(targetKeyspaceName, targetTableName)
	copied.CreationTimestamp = time.Now().UTC()
	copied.Status = "ACTIVE"

	if input.CapacitySpecificationOverride != nil {
		copied.CapacitySpecification = cloneMap(input.CapacitySpecificationOverride)
	}
	if input.EncryptionSpecificationOverride != nil {
		copied.EncryptionSpecification = cloneMap(input.EncryptionSpecificationOverride)
	}
	if input.PointInTimeRecoveryOverride != nil {
		copied.PointInTimeRecovery = cloneMap(input.PointInTimeRecoveryOverride)
	}
	if input.AutoScalingSpecification != nil {
		copied.AutoScalingSpecification = cloneMap(input.AutoScalingSpecification)
	}
	if input.ReplicaSpecifications != nil {
		copied.ReplicaSpecifications = cloneSliceOfMaps(input.ReplicaSpecifications)
	}

	s.tables[targetKeyspaceName][targetTableName] = &copied
	if input.TagsOverride != nil {
		s.resourceTags[copied.ResourceARN] = cloneStringMap(input.TagsOverride)
	} else {
		s.resourceTags[copied.ResourceARN] = cloneStringMap(s.resourceTags[sourceTable.ResourceARN])
	}

	return copied.ResourceARN, nil
}

func (s *Service) GetTableAutoScalingSettings(keyspaceName, tableName string) (TableAutoScalingSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	tableName = strings.TrimSpace(tableName)
	if keyspaceName == "" || tableName == "" {
		return TableAutoScalingSettings{}, ErrInvalidParameter
	}
	keyspaceTables, ok := s.tables[keyspaceName]
	if !ok {
		return TableAutoScalingSettings{}, ErrNotFound
	}
	table, ok := keyspaceTables[tableName]
	if !ok {
		return TableAutoScalingSettings{}, ErrNotFound
	}

	replicaSpecs := make([]map[string]any, 0)
	for _, replica := range table.ReplicaSpecifications {
		region, _ := replica["region"].(string)
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		entry := map[string]any{"region": region}

		if autoScalingSpec, ok := replica["autoScalingSpecification"].(map[string]any); ok && autoScalingSpec != nil {
			entry["autoScalingSpecification"] = cloneMap(autoScalingSpec)
			replicaSpecs = append(replicaSpecs, entry)
			continue
		}

		legacyAutoScaling := map[string]any{}
		if readCapacityAutoScaling, ok := replica["readCapacityAutoScaling"].(map[string]any); ok && readCapacityAutoScaling != nil {
			legacyAutoScaling["readCapacityAutoScaling"] = cloneMap(readCapacityAutoScaling)
		}
		if writeCapacityAutoScaling, ok := replica["writeCapacityAutoScaling"].(map[string]any); ok && writeCapacityAutoScaling != nil {
			legacyAutoScaling["writeCapacityAutoScaling"] = cloneMap(writeCapacityAutoScaling)
		}
		if len(legacyAutoScaling) > 0 {
			entry["autoScalingSpecification"] = legacyAutoScaling
		}
		replicaSpecs = append(replicaSpecs, entry)
	}

	return TableAutoScalingSettings{
		KeyspaceName:             table.KeyspaceName,
		TableName:                table.TableName,
		ResourceARN:              table.ResourceARN,
		AutoScalingSpecification: cloneMap(table.AutoScalingSpecification),
		ReplicaSpecifications:    replicaSpecs,
	}, nil
}

func (s *Service) CreateType(keyspaceName, typeName string, fieldDefinitions []map[string]any) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	typeName = strings.TrimSpace(typeName)
	if keyspaceName == "" || typeName == "" {
		return "", "", ErrInvalidParameter
	}
	if len(fieldDefinitions) == 0 {
		return "", "", ErrInvalidParameter
	}
	for _, field := range fieldDefinitions {
		fieldName, _ := field["name"].(string)
		fieldType, _ := field["type"].(string)
		if strings.TrimSpace(fieldName) == "" || strings.TrimSpace(fieldType) == "" {
			return "", "", ErrInvalidParameter
		}
	}
	keyspace, ok := s.keyspaces[keyspaceName]
	if !ok {
		return "", "", ErrNotFound
	}
	if _, ok := s.types[keyspaceName]; !ok {
		s.types[keyspaceName] = map[string]*Type{}
	}
	if _, exists := s.types[keyspaceName][typeName]; exists {
		return "", "", ErrAlreadyExists
	}

	entry := &Type{
		KeyspaceName:          keyspaceName,
		TypeName:              typeName,
		FieldDefinitions:      cloneSliceOfMaps(fieldDefinitions),
		LastModifiedTimestamp: time.Now().UTC(),
		Status:                "ACTIVE",
		DirectReferringTables: nil,
		DirectParentTypes:     nil,
		MaxNestingDepth:       1,
		KeyspaceARN:           keyspace.ResourceARN,
	}
	s.types[keyspaceName][typeName] = entry
	return keyspace.ResourceARN, typeName, nil
}

func (s *Service) GetType(keyspaceName, typeName string) (Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	typeName = strings.TrimSpace(typeName)
	if keyspaceName == "" || typeName == "" {
		return Type{}, ErrInvalidParameter
	}
	keyspaceTypes, ok := s.types[keyspaceName]
	if !ok {
		return Type{}, ErrNotFound
	}
	entry, ok := keyspaceTypes[typeName]
	if !ok {
		return Type{}, ErrNotFound
	}

	out := cloneType(entry)
	out.DirectReferringTables = s.typeDirectReferringTablesLocked(keyspaceName, typeName)
	out.DirectParentTypes = s.typeDirectParentTypesLocked(keyspaceName, typeName)
	out.MaxNestingDepth = s.typeMaxNestingDepthLocked(keyspaceName, typeName, map[string]struct{}{})
	return out, nil
}

func (s *Service) ListTypes(keyspaceName, nextToken string, maxResults int) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, err := parseOffsetToken(nextToken)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 1 {
		return nil, "", ErrInvalidParameter
	}

	keyspaceName = strings.TrimSpace(keyspaceName)
	if keyspaceName == "" {
		return nil, "", ErrInvalidParameter
	}
	if _, ok := s.keyspaces[keyspaceName]; !ok {
		return nil, "", ErrNotFound
	}

	types := make([]string, 0)
	for name := range s.types[keyspaceName] {
		types = append(types, name)
	}
	sort.Strings(types)
	if start > len(types) {
		return []string{}, "", nil
	}
	end := start + maxResults
	if end > len(types) {
		end = len(types)
	}
	out := append([]string(nil), types[start:end]...)
	outToken := ""
	if end < len(types) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func (s *Service) DeleteType(keyspaceName, typeName string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyspaceName = strings.TrimSpace(keyspaceName)
	typeName = strings.TrimSpace(typeName)
	if keyspaceName == "" || typeName == "" {
		return "", "", ErrInvalidParameter
	}
	keyspace, ok := s.keyspaces[keyspaceName]
	if !ok {
		return "", "", ErrNotFound
	}
	keyspaceTypes, ok := s.types[keyspaceName]
	if !ok {
		return "", "", ErrNotFound
	}
	if _, ok := keyspaceTypes[typeName]; !ok {
		return "", "", ErrNotFound
	}
	if parents := s.typeDirectParentTypesLocked(keyspaceName, typeName); len(parents) > 0 {
		return "", "", ErrConflict
	}
	if tables := s.typeDirectReferringTablesLocked(keyspaceName, typeName); len(tables) > 0 {
		return "", "", ErrConflict
	}
	delete(keyspaceTypes, typeName)
	return keyspace.ResourceARN, typeName, nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}
	if !s.resourceExists(resourceARN) {
		return ErrNotFound
	}
	if len(tags) == 0 {
		return ErrInvalidParameter
	}
	resourceTags := s.resourceTags[resourceARN]
	if resourceTags == nil {
		resourceTags = map[string]string{}
		s.resourceTags[resourceARN] = resourceTags
	}
	added := 0
	for k, v := range tags {
		if strings.TrimSpace(k) == "" {
			continue
		}
		resourceTags[k] = v
		added++
	}
	if added == 0 {
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}
	if !s.resourceExists(resourceARN) {
		return ErrNotFound
	}
	if len(tagKeys) == 0 {
		return ErrInvalidParameter
	}
	resourceTags := s.resourceTags[resourceARN]
	removed := 0
	for _, key := range tagKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		delete(resourceTags, trimmed)
		removed++
	}
	if removed == 0 {
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) ListTagsForResource(resourceARN, nextToken string, maxResults int) (map[string]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, "", ErrInvalidParameter
	}
	if !s.resourceExists(resourceARN) {
		return nil, "", ErrNotFound
	}
	start, err := parseOffsetToken(nextToken)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 1 {
		return nil, "", ErrInvalidParameter
	}

	keys := make([]string, 0, len(s.resourceTags[resourceARN]))
	for k := range s.resourceTags[resourceARN] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if start > len(keys) {
		return map[string]string{}, "", nil
	}
	end := start + maxResults
	if end > len(keys) {
		end = len(keys)
	}
	out := map[string]string{}
	for _, k := range keys[start:end] {
		out[k] = s.resourceTags[resourceARN][k]
	}
	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return out, outToken, nil
}

func keyspaceARN(name string) string {
	return fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s", DefaultRegion, DefaultAccountID, name)
}

func tableARN(keyspaceName, tableName string) string {
	return fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s/table/%s", DefaultRegion, DefaultAccountID, keyspaceName, tableName)
}

func parseOffsetToken(nextToken string) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(nextToken)
	if err != nil || start < 0 {
		return 0, ErrInvalidParameter
	}
	return start, nil
}

func normalizeReplicationSpecification(spec ReplicationSpecification) (string, []string) {
	strategy := strings.ToUpper(strings.TrimSpace(spec.ReplicationStrategy))
	if strategy == "" {
		strategy = "SINGLE_REGION"
	}
	regions := make([]string, 0, len(spec.RegionList))
	seen := map[string]struct{}{}
	for _, region := range spec.RegionList {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	if len(regions) == 0 {
		regions = []string{DefaultRegion}
	}
	if strategy == "MULTI_REGION" && len(regions) < 2 {
		if regions[0] != DefaultRegion {
			regions = append([]string{DefaultRegion}, regions...)
		}
		if len(regions) < 2 {
			regions = append(regions, "us-west-2")
		}
	}
	sort.Strings(regions)
	return strategy, regions
}

func buildReplicationGroupStatuses(regions []string) []ReplicationGroupStatus {
	if len(regions) == 0 {
		return nil
	}
	out := make([]ReplicationGroupStatus, 0, len(regions))
	for _, region := range regions {
		out = append(out, ReplicationGroupStatus{
			Region:                    region,
			KeyspaceStatus:            "ACTIVE",
			TablesReplicationProgress: 100,
		})
	}
	return out
}

func cloneKeyspace(in *Keyspace) Keyspace {
	out := Keyspace{
		KeyspaceName:        in.KeyspaceName,
		ResourceARN:         in.ResourceARN,
		ReplicationStrategy: in.ReplicationStrategy,
		ReplicationRegions:  cloneStrings(in.ReplicationRegions),
	}
	if len(in.ReplicationGroupStatuses) > 0 {
		out.ReplicationGroupStatuses = make([]ReplicationGroupStatus, 0, len(in.ReplicationGroupStatuses))
		out.ReplicationGroupStatuses = append(out.ReplicationGroupStatuses, in.ReplicationGroupStatuses...)
	}
	return out
}

func cloneTable(in *Table) Table {
	out := Table{
		KeyspaceName:             in.KeyspaceName,
		TableName:                in.TableName,
		ResourceARN:              in.ResourceARN,
		CreationTimestamp:        in.CreationTimestamp,
		Status:                   in.Status,
		SchemaDefinition:         cloneMap(in.SchemaDefinition),
		CapacitySpecification:    cloneMap(in.CapacitySpecification),
		EncryptionSpecification:  cloneMap(in.EncryptionSpecification),
		PointInTimeRecovery:      cloneMap(in.PointInTimeRecovery),
		TTL:                      cloneMap(in.TTL),
		Comment:                  cloneMap(in.Comment),
		ClientSideTimestamps:     cloneMap(in.ClientSideTimestamps),
		AutoScalingSpecification: cloneMap(in.AutoScalingSpecification),
		ReplicaSpecifications:    cloneSliceOfMaps(in.ReplicaSpecifications),
	}
	if in.DefaultTimeToLive != nil {
		val := *in.DefaultTimeToLive
		out.DefaultTimeToLive = &val
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	return cloneAny(in).(map[string]any)
}

func cloneType(in *Type) Type {
	out := Type{
		KeyspaceName:          in.KeyspaceName,
		TypeName:              in.TypeName,
		FieldDefinitions:      cloneSliceOfMaps(in.FieldDefinitions),
		LastModifiedTimestamp: in.LastModifiedTimestamp,
		Status:                in.Status,
		DirectReferringTables: cloneStrings(in.DirectReferringTables),
		DirectParentTypes:     cloneStrings(in.DirectParentTypes),
		MaxNestingDepth:       in.MaxNestingDepth,
		KeyspaceARN:           in.KeyspaceARN,
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *Service) resourceExists(resourceARN string) bool {
	for _, keyspace := range s.keyspaces {
		if keyspace.ResourceARN == resourceARN {
			return true
		}
	}
	for _, keyspaceTables := range s.tables {
		for _, table := range keyspaceTables {
			if table.ResourceARN == resourceARN {
				return true
			}
		}
	}
	return false
}

func (s *Service) typeDirectReferringTablesLocked(keyspaceName, typeName string) []string {
	target := strings.TrimSpace(typeName)
	if target == "" {
		return nil
	}
	keyspaceTables := s.tables[keyspaceName]
	if len(keyspaceTables) == 0 {
		return nil
	}

	out := make([]string, 0)
	for tableName, table := range keyspaceTables {
		if table == nil || table.SchemaDefinition == nil {
			continue
		}
		if !schemaReferencesType(table.SchemaDefinition, target) {
			continue
		}
		out = append(out, tableName)
	}
	sort.Strings(out)
	return out
}

func (s *Service) typeDirectParentTypesLocked(keyspaceName, typeName string) []string {
	target := strings.TrimSpace(typeName)
	if target == "" {
		return nil
	}
	keyspaceTypes := s.types[keyspaceName]
	if len(keyspaceTypes) == 0 {
		return nil
	}

	out := make([]string, 0)
	for parentName, parent := range keyspaceTypes {
		if parent == nil {
			continue
		}
		if parentName == target {
			continue
		}
		if !fieldDefinitionsReferenceType(parent.FieldDefinitions, target) {
			continue
		}
		out = append(out, parentName)
	}
	sort.Strings(out)
	return out
}

func (s *Service) typeMaxNestingDepthLocked(keyspaceName, typeName string, seen map[string]struct{}) int {
	if _, ok := seen[typeName]; ok {
		return 1
	}
	seen[typeName] = struct{}{}

	current := s.types[keyspaceName][typeName]
	if current == nil {
		return 1
	}

	maxDepth := 1
	children := s.childTypesForTypeLocked(keyspaceName, current)
	for _, child := range children {
		nextSeen := cloneStringSet(seen)
		depth := 1 + s.typeMaxNestingDepthLocked(keyspaceName, child, nextSeen)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func (s *Service) childTypesForTypeLocked(keyspaceName string, current *Type) []string {
	if current == nil || len(current.FieldDefinitions) == 0 {
		return nil
	}
	keyspaceTypes := s.types[keyspaceName]
	if len(keyspaceTypes) == 0 {
		return nil
	}

	children := make([]string, 0)
	seen := map[string]struct{}{}
	for candidate := range keyspaceTypes {
		if candidate == current.TypeName {
			continue
		}
		if !fieldDefinitionsReferenceType(current.FieldDefinitions, candidate) {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		children = append(children, candidate)
	}
	sort.Strings(children)
	return children
}

func schemaReferencesType(schema map[string]any, targetType string) bool {
	rawColumns, ok := schema["allColumns"]
	if !ok {
		return false
	}
	switch cols := rawColumns.(type) {
	case []any:
		for _, col := range cols {
			colMap, _ := col.(map[string]any)
			colType, _ := colMap["type"].(string)
			if typeReferencesType(colType, targetType) {
				return true
			}
		}
	case []map[string]any:
		for _, colMap := range cols {
			colType, _ := colMap["type"].(string)
			if typeReferencesType(colType, targetType) {
				return true
			}
		}
	}
	return false
}

func fieldDefinitionsReferenceType(fieldDefinitions []map[string]any, targetType string) bool {
	for _, field := range fieldDefinitions {
		fieldType, _ := field["type"].(string)
		if typeReferencesType(fieldType, targetType) {
			return true
		}
	}
	return false
}

func typeReferencesType(fieldType, targetType string) bool {
	target := strings.ToLower(strings.TrimSpace(targetType))
	field := strings.ToLower(strings.TrimSpace(fieldType))
	if target == "" || field == "" {
		return false
	}
	if field == target {
		return true
	}

	start := 0
	for {
		offset := strings.Index(field[start:], target)
		if offset < 0 {
			return false
		}
		offset += start

		beforeOK := offset == 0 || !isTypeIdentifierRune(rune(field[offset-1]))
		afterIndex := offset + len(target)
		afterOK := afterIndex >= len(field) || !isTypeIdentifierRune(rune(field[afterIndex]))
		if beforeOK && afterOK {
			return true
		}
		start = offset + 1
		if start >= len(field) {
			return false
		}
	}
}

func isTypeIdentifierRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
}

func cloneStringSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func defaultIfNilMap(in map[string]any, defaults map[string]any) map[string]any {
	if in == nil {
		return cloneMap(defaults)
	}
	return cloneMap(in)
}

func cloneSliceOfMaps(in []map[string]any) []map[string]any {
	if in == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		out = append(out, cloneMap(m))
	}
	return out
}

func cloneAny(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, inner := range value {
			out[k] = cloneAny(inner)
		}
		return out
	case []any:
		out := make([]any, 0, len(value))
		for _, inner := range value {
			out = append(out, cloneAny(inner))
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, 0, len(value))
		for _, inner := range value {
			out = append(out, cloneMap(inner))
		}
		return out
	default:
		return value
	}
}
