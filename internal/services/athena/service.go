package athena

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrWorkGroupExists           = errors.New("workgroup already exists")
	ErrWorkGroupNotFound         = errors.New("workgroup not found")
	ErrDataCatalogExists         = errors.New("data catalog already exists")
	ErrDataCatalogNotFound       = errors.New("data catalog not found")
	ErrDatabaseExists            = errors.New("database already exists")
	ErrDatabaseNotFound          = errors.New("database not found")
	ErrTableExists               = errors.New("table already exists")
	ErrTableNotFound             = errors.New("table not found")
	ErrNamedQueryNotFound        = errors.New("named query not found")
	ErrPreparedStatementNotFound = errors.New("prepared statement not found")
	ErrQueryExecutionNotFound    = errors.New("query execution not found")
	ErrInvalidParameter          = errors.New("invalid parameter")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type WorkGroup struct {
	Name        string
	Arn         string
	Description string
	State       string
	CreatedAt   time.Time
}

type DataCatalog struct {
	Name        string
	Arn         string
	Type        string
	Description string
	Parameters  map[string]string
}

type Database struct {
	Name        string
	Description string
	Parameters  map[string]string
}

type Table struct {
	Name        string
	Database    string
	Description string
	Columns     []string
	Parameters  map[string]string
}

type NamedQuery struct {
	ID          string
	Name        string
	Description string
	Database    string
	QueryString string
	WorkGroup   string
}

type PreparedStatement struct {
	Name        string
	Description string
	Query       string
	WorkGroup   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type QueryExecutionStatus struct {
	State             string
	StateChangeReason string
	SubmissionTime    time.Time
	CompletionTime    time.Time
}

type QueryExecution struct {
	ID               string
	QueryString      string
	Database         string
	Catalog          string
	WorkGroup        string
	OutputLocation   string
	Status           QueryExecutionStatus
	EngineVersion    string
	ResultRows       [][]string
	ResultColumnInfo []string
}

type CapacityReservation struct {
	Name          string
	Status        string
	TargetDpus    int
	AllocatedDpus int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CapacityAssignment struct {
	WorkGroupNames []string
}

type Notebook struct {
	ID         string
	Name       string
	WorkGroup  string
	Type       string
	Payload    string
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type CalculationExecution struct {
	ID          string
	SessionID   string
	Description string
	CodeBlock   string
	State       string
	SubmittedAt time.Time
	CompletedAt time.Time
}

type Session struct {
	ID            string
	Description   string
	WorkGroup     string
	State         string
	EngineVersion string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Service struct {
	mu                   sync.Mutex
	seq                  uint64
	workGroups           map[string]*WorkGroup
	dataCatalogs         map[string]*DataCatalog
	databases            map[string]map[string]*Database
	tables               map[string]map[string]map[string]*Table
	namedQueries         map[string]*NamedQuery
	preparedStatements   map[string]map[string]*PreparedStatement
	queryExecutions      map[string]*QueryExecution
	capacityReservations map[string]*CapacityReservation
	capacityAssignments  map[string][]CapacityAssignment
	notebooks            map[string]*Notebook
	calculations         map[string]*CalculationExecution
	sessions             map[string]*Session
	resourceTags         map[string]map[string]string
}

func NewService() *Service {
	s := &Service{
		workGroups:           make(map[string]*WorkGroup),
		dataCatalogs:         make(map[string]*DataCatalog),
		databases:            make(map[string]map[string]*Database),
		tables:               make(map[string]map[string]map[string]*Table),
		namedQueries:         make(map[string]*NamedQuery),
		preparedStatements:   make(map[string]map[string]*PreparedStatement),
		queryExecutions:      make(map[string]*QueryExecution),
		capacityReservations: make(map[string]*CapacityReservation),
		capacityAssignments:  make(map[string][]CapacityAssignment),
		notebooks:            make(map[string]*Notebook),
		calculations:         make(map[string]*CalculationExecution),
		sessions:             make(map[string]*Session),
		resourceTags:         make(map[string]map[string]string),
	}

	_ = s.CreateWorkGroup("primary", "Default workgroup", "ENABLED")
	_ = s.CreateDataCatalog("AwsDataCatalog", "GLUE", "Default catalog", nil)
	_ = s.CreateDatabase("AwsDataCatalog", "default", "Default database", nil)
	_ = s.CreateTable("AwsDataCatalog", "default", "sample_table", "Sample table", []string{"result"}, nil)
	return s
}

func (s *Service) nextID(prefix string) string {
	id := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("%s-%d", prefix, id)
}

func workGroupArn(name string) string {
	return fmt.Sprintf("arn:aws:athena:%s:%s:workgroup/%s", DefaultRegion, DefaultAccountID, name)
}

func catalogArn(name string) string {
	return fmt.Sprintf("arn:aws:athena:%s:%s:datacatalog/%s", DefaultRegion, DefaultAccountID, name)
}

func (s *Service) CreateWorkGroup(name, description, state string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	if state == "" {
		state = "ENABLED"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workGroups[name]; ok {
		return ErrWorkGroupExists
	}
	s.workGroups[name] = &WorkGroup{
		Name:        name,
		Arn:         workGroupArn(name),
		Description: description,
		State:       state,
		CreatedAt:   time.Now().UTC(),
	}
	return nil
}

func (s *Service) GetWorkGroup(name string) (WorkGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wg, ok := s.workGroups[name]
	if !ok {
		return WorkGroup{}, ErrWorkGroupNotFound
	}
	return *wg, nil
}

func (s *Service) ListWorkGroups() []WorkGroup {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]WorkGroup, 0, len(s.workGroups))
	for _, wg := range s.workGroups {
		out = append(out, *wg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateWorkGroup(name, description, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	wg, ok := s.workGroups[name]
	if !ok {
		return ErrWorkGroupNotFound
	}
	if description != "" {
		wg.Description = description
	}
	if state != "" {
		wg.State = state
	}
	return nil
}

func (s *Service) DeleteWorkGroup(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workGroups[name]; !ok {
		return ErrWorkGroupNotFound
	}
	delete(s.workGroups, name)
	return nil
}

func (s *Service) CreateDataCatalog(name, typ, description string, params map[string]string) error {
	name = strings.TrimSpace(name)
	if name == "" || typ == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataCatalogs[name]; ok {
		return ErrDataCatalogExists
	}
	s.dataCatalogs[name] = &DataCatalog{
		Name:        name,
		Arn:         catalogArn(name),
		Type:        typ,
		Description: description,
		Parameters:  cloneStringMap(params),
	}
	return nil
}

func (s *Service) GetDataCatalog(name string) (DataCatalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cat, ok := s.dataCatalogs[name]
	if !ok {
		return DataCatalog{}, ErrDataCatalogNotFound
	}
	return *cat, nil
}

func (s *Service) ListDataCatalogs() []DataCatalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DataCatalog, 0, len(s.dataCatalogs))
	for _, cat := range s.dataCatalogs {
		out = append(out, *cat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateDataCatalog(name, typ, description string, params map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cat, ok := s.dataCatalogs[name]
	if !ok {
		return ErrDataCatalogNotFound
	}
	if typ != "" {
		cat.Type = typ
	}
	if description != "" {
		cat.Description = description
	}
	if params != nil {
		cat.Parameters = cloneStringMap(params)
	}
	return nil
}

func (s *Service) DeleteDataCatalog(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataCatalogs[name]; !ok {
		return ErrDataCatalogNotFound
	}
	delete(s.dataCatalogs, name)
	return nil
}

func (s *Service) CreateDatabase(catalog, name, description string, params map[string]string) error {
	catalog = defaultCatalog(catalog)
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[catalog]; !ok {
		s.databases[catalog] = make(map[string]*Database)
	}
	if _, ok := s.databases[catalog][name]; ok {
		return ErrDatabaseExists
	}
	s.databases[catalog][name] = &Database{
		Name:        name,
		Description: description,
		Parameters:  cloneStringMap(params),
	}
	return nil
}

func (s *Service) GetDatabase(catalog, name string) (Database, error) {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	dbs, ok := s.databases[catalog]
	if !ok {
		return Database{}, ErrDatabaseNotFound
	}
	db, ok := dbs[name]
	if !ok {
		return Database{}, ErrDatabaseNotFound
	}
	return *db, nil
}

func (s *Service) ListDatabases(catalog string) []Database {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	dbs := s.databases[catalog]
	out := make([]Database, 0, len(dbs))
	for _, db := range dbs {
		out = append(out, *db)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateDatabase(catalog, name, description string, params map[string]string) error {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	dbs, ok := s.databases[catalog]
	if !ok {
		return ErrDatabaseNotFound
	}
	db, ok := dbs[name]
	if !ok {
		return ErrDatabaseNotFound
	}
	if description != "" {
		db.Description = description
	}
	if params != nil {
		db.Parameters = cloneStringMap(params)
	}
	return nil
}

func (s *Service) DeleteDatabase(catalog, name string) error {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	dbs, ok := s.databases[catalog]
	if !ok {
		return ErrDatabaseNotFound
	}
	if _, ok := dbs[name]; !ok {
		return ErrDatabaseNotFound
	}
	delete(dbs, name)
	return nil
}

func (s *Service) CreateTable(catalog, dbName, name, description string, columns []string, params map[string]string) error {
	catalog = defaultCatalog(catalog)
	if name == "" || dbName == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tables[catalog]; !ok {
		s.tables[catalog] = make(map[string]map[string]*Table)
	}
	if _, ok := s.tables[catalog][dbName]; !ok {
		s.tables[catalog][dbName] = make(map[string]*Table)
	}
	if _, ok := s.tables[catalog][dbName][name]; ok {
		return ErrTableExists
	}
	s.tables[catalog][dbName][name] = &Table{
		Name:        name,
		Database:    dbName,
		Description: description,
		Columns:     append([]string(nil), columns...),
		Parameters:  cloneStringMap(params),
	}
	return nil
}

func (s *Service) GetTable(catalog, dbName, name string) (Table, error) {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tables[catalog]; !ok {
		return Table{}, ErrTableNotFound
	}
	if _, ok := s.tables[catalog][dbName]; !ok {
		return Table{}, ErrTableNotFound
	}
	tbl, ok := s.tables[catalog][dbName][name]
	if !ok {
		return Table{}, ErrTableNotFound
	}
	return *tbl, nil
}

func (s *Service) ListTables(catalog, dbName string) []Table {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	dbTables := s.tables[catalog][dbName]
	out := make([]Table, 0, len(dbTables))
	for _, tbl := range dbTables {
		out = append(out, *tbl)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateTable(catalog, dbName, name, description string, columns []string, params map[string]string) error {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	tbl, err := s.GetTableLocked(catalog, dbName, name)
	if err != nil {
		return err
	}
	if description != "" {
		tbl.Description = description
	}
	if columns != nil {
		tbl.Columns = append([]string(nil), columns...)
	}
	if params != nil {
		tbl.Parameters = cloneStringMap(params)
	}
	return nil
}

func (s *Service) DeleteTable(catalog, dbName, name string) error {
	catalog = defaultCatalog(catalog)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tables[catalog]; !ok {
		return ErrTableNotFound
	}
	if _, ok := s.tables[catalog][dbName]; !ok {
		return ErrTableNotFound
	}
	if _, ok := s.tables[catalog][dbName][name]; !ok {
		return ErrTableNotFound
	}
	delete(s.tables[catalog][dbName], name)
	return nil
}

func (s *Service) GetTableLocked(catalog, dbName, name string) (*Table, error) {
	if _, ok := s.tables[catalog]; !ok {
		return nil, ErrTableNotFound
	}
	if _, ok := s.tables[catalog][dbName]; !ok {
		return nil, ErrTableNotFound
	}
	tbl, ok := s.tables[catalog][dbName][name]
	if !ok {
		return nil, ErrTableNotFound
	}
	return tbl, nil
}

func (s *Service) CreateNamedQuery(name, database, query, description, workGroup string) (NamedQuery, error) {
	name = strings.TrimSpace(name)
	if name == "" || database == "" || query == "" {
		return NamedQuery{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID("named")
	nq := &NamedQuery{
		ID:          id,
		Name:        name,
		Description: description,
		Database:    database,
		QueryString: query,
		WorkGroup:   workGroup,
	}
	s.namedQueries[id] = nq
	return *nq, nil
}

func (s *Service) GetNamedQuery(id string) (NamedQuery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nq, ok := s.namedQueries[id]
	if !ok {
		return NamedQuery{}, ErrNamedQueryNotFound
	}
	return *nq, nil
}

func (s *Service) ListNamedQueries(workGroup string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.namedQueries))
	for id, nq := range s.namedQueries {
		if workGroup != "" && nq.WorkGroup != workGroup {
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *Service) UpdateNamedQuery(id, name, description, query, database string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nq, ok := s.namedQueries[id]
	if !ok {
		return ErrNamedQueryNotFound
	}
	if name != "" {
		nq.Name = name
	}
	if description != "" {
		nq.Description = description
	}
	if query != "" {
		nq.QueryString = query
	}
	if database != "" {
		nq.Database = database
	}
	return nil
}

func (s *Service) DeleteNamedQuery(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namedQueries[id]; !ok {
		return ErrNamedQueryNotFound
	}
	delete(s.namedQueries, id)
	return nil
}

func (s *Service) CreatePreparedStatement(workGroup, name, query, description string) (PreparedStatement, error) {
	if workGroup == "" || name == "" || query == "" {
		return PreparedStatement{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.preparedStatements[workGroup]; !ok {
		s.preparedStatements[workGroup] = make(map[string]*PreparedStatement)
	}
	if _, ok := s.preparedStatements[workGroup][name]; ok {
		return PreparedStatement{}, ErrInvalidParameter
	}
	ps := &PreparedStatement{
		Name:        name,
		Description: description,
		Query:       query,
		WorkGroup:   workGroup,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	s.preparedStatements[workGroup][name] = ps
	return *ps, nil
}

func (s *Service) GetPreparedStatement(workGroup, name string) (PreparedStatement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.preparedStatements[workGroup]; !ok {
		return PreparedStatement{}, ErrPreparedStatementNotFound
	}
	ps, ok := s.preparedStatements[workGroup][name]
	if !ok {
		return PreparedStatement{}, ErrPreparedStatementNotFound
	}
	return *ps, nil
}

func (s *Service) ListPreparedStatements(workGroup string) []PreparedStatement {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.preparedStatements[workGroup]
	out := make([]PreparedStatement, 0, len(list))
	for _, ps := range list {
		out = append(out, *ps)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdatePreparedStatement(workGroup, name, query, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.preparedStatements[workGroup]
	if list == nil {
		return ErrPreparedStatementNotFound
	}
	ps, ok := list[name]
	if !ok {
		return ErrPreparedStatementNotFound
	}
	if query != "" {
		ps.Query = query
	}
	if description != "" {
		ps.Description = description
	}
	ps.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) DeletePreparedStatement(workGroup, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.preparedStatements[workGroup]; !ok {
		return ErrPreparedStatementNotFound
	}
	if _, ok := s.preparedStatements[workGroup][name]; !ok {
		return ErrPreparedStatementNotFound
	}
	delete(s.preparedStatements[workGroup], name)
	return nil
}

func (s *Service) StartQueryExecution(query, database, catalog, workGroup, outputLocation string) (QueryExecution, error) {
	if query == "" {
		return QueryExecution{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID("query")
	status := QueryExecutionStatus{
		State:          "SUCCEEDED",
		SubmissionTime: time.Now().UTC(),
		CompletionTime: time.Now().UTC(),
	}
	qe := &QueryExecution{
		ID:               id,
		QueryString:      query,
		Database:         database,
		Catalog:          defaultCatalog(catalog),
		WorkGroup:        workGroup,
		OutputLocation:   outputLocation,
		Status:           status,
		EngineVersion:    "Athena engine version 3",
		ResultColumnInfo: []string{"result"},
		ResultRows:       [][]string{{"ok"}},
	}
	s.queryExecutions[id] = qe
	return *qe, nil
}

func (s *Service) GetQueryExecution(id string) (QueryExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	qe, ok := s.queryExecutions[id]
	if !ok {
		return QueryExecution{}, ErrQueryExecutionNotFound
	}
	return *qe, nil
}

func (s *Service) ListQueryExecutions(workGroup string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.queryExecutions))
	for id, qe := range s.queryExecutions {
		if workGroup != "" && qe.WorkGroup != workGroup {
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *Service) StopQueryExecution(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	qe, ok := s.queryExecutions[id]
	if !ok {
		return ErrQueryExecutionNotFound
	}
	qe.Status.State = "CANCELLED"
	qe.Status.StateChangeReason = "cancelled"
	qe.Status.CompletionTime = time.Now().UTC()
	return nil
}

func (s *Service) TagResource(resourceArn string, tags map[string]string) {
	if resourceArn == "" || len(tags) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resourceTags[resourceArn]; !ok {
		s.resourceTags[resourceArn] = make(map[string]string)
	}
	for k, v := range tags {
		s.resourceTags[resourceArn][k] = v
	}
}

func (s *Service) UntagResource(resourceArn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tagMap := s.resourceTags[resourceArn]
	if tagMap == nil {
		return
	}
	for _, k := range keys {
		delete(tagMap, k)
	}
}

func (s *Service) ListTags(resourceArn string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneStringMap(s.resourceTags[resourceArn])
}

func (s *Service) CreateCapacityReservation(name string, targetDpus int) (CapacityReservation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	if targetDpus < 24 {
		targetDpus = 24
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cr, ok := s.capacityReservations[name]; ok {
		return *cr, nil
	}
	now := time.Now().UTC()
	cr := &CapacityReservation{
		Name:          name,
		Status:        "ACTIVE",
		TargetDpus:    targetDpus,
		AllocatedDpus: targetDpus,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.capacityReservations[name] = cr
	return *cr, nil
}

func (s *Service) UpdateCapacityReservation(name string, targetDpus int) (CapacityReservation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	if targetDpus < 24 {
		targetDpus = 24
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cr, ok := s.capacityReservations[name]
	if !ok {
		now := time.Now().UTC()
		cr = &CapacityReservation{
			Name:          name,
			Status:        "ACTIVE",
			CreatedAt:     now,
			UpdatedAt:     now,
			TargetDpus:    targetDpus,
			AllocatedDpus: targetDpus,
		}
		s.capacityReservations[name] = cr
		return *cr, nil
	}
	cr.TargetDpus = targetDpus
	cr.AllocatedDpus = targetDpus
	cr.UpdatedAt = time.Now().UTC()
	return *cr, nil
}

func (s *Service) GetCapacityReservation(name string) (CapacityReservation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cr, ok := s.capacityReservations[name]
	if !ok {
		return CapacityReservation{}, ErrInvalidParameter
	}
	return *cr, nil
}

func (s *Service) ListCapacityReservations() []CapacityReservation {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CapacityReservation, 0, len(s.capacityReservations))
	for _, cr := range s.capacityReservations {
		out = append(out, *cr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CancelCapacityReservation(name string) (CapacityReservation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cr, ok := s.capacityReservations[name]
	if !ok {
		now := time.Now().UTC()
		cr = &CapacityReservation{
			Name:          name,
			Status:        "CANCELLED",
			TargetDpus:    24,
			AllocatedDpus: 0,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.capacityReservations[name] = cr
		return *cr, nil
	}
	cr.Status = "CANCELLED"
	cr.AllocatedDpus = 0
	cr.UpdatedAt = time.Now().UTC()
	return *cr, nil
}

func (s *Service) DeleteCapacityReservation(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.capacityReservations, name)
	delete(s.capacityAssignments, name)
	return nil
}

func (s *Service) PutCapacityAssignmentConfiguration(name string, assignments []CapacityAssignment) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capacityAssignments[name] = append([]CapacityAssignment(nil), assignments...)
}

func (s *Service) GetCapacityAssignmentConfiguration(name string) []CapacityAssignment {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	assignments := s.capacityAssignments[name]
	out := make([]CapacityAssignment, 0, len(assignments))
	out = append(out, assignments...)
	return out
}

func (s *Service) CreateNotebook(workGroup, name string) (Notebook, error) {
	workGroup = strings.TrimSpace(workGroup)
	name = strings.TrimSpace(name)
	if workGroup == "" {
		workGroup = "primary"
	}
	if name == "" {
		return Notebook{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if nb, ok := s.notebooks[name]; ok {
		return *nb, nil
	}
	now := time.Now().UTC()
	nb := &Notebook{
		ID:         s.nextID("notebook"),
		Name:       name,
		WorkGroup:  workGroup,
		Type:       "IPYNB",
		Payload:    "",
		CreatedAt:  now,
		ModifiedAt: now,
	}
	s.notebooks[name] = nb
	return *nb, nil
}

func (s *Service) ImportNotebook(workGroup, name, payload, notebookType string) (Notebook, error) {
	nb, err := s.CreateNotebook(workGroup, name)
	if err != nil {
		return Notebook{}, err
	}
	return s.UpdateNotebook(nb.ID, payload, notebookType)
}

func (s *Service) findNotebookLocked(idOrName string) *Notebook {
	if nb, ok := s.notebooks[idOrName]; ok {
		return nb
	}
	for _, nb := range s.notebooks {
		if nb.ID == idOrName {
			return nb
		}
	}
	return nil
}

func (s *Service) UpdateNotebook(idOrName, payload, notebookType string) (Notebook, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return Notebook{}, ErrInvalidParameter
	}
	if notebookType == "" {
		notebookType = "IPYNB"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	nb := s.findNotebookLocked(idOrName)
	if nb == nil {
		now := time.Now().UTC()
		nb = &Notebook{
			ID:         s.nextID("notebook"),
			Name:       idOrName,
			WorkGroup:  "primary",
			Type:       notebookType,
			Payload:    payload,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		s.notebooks[nb.Name] = nb
		return *nb, nil
	}
	nb.Payload = payload
	nb.Type = notebookType
	nb.ModifiedAt = time.Now().UTC()
	return *nb, nil
}

func (s *Service) UpdateNotebookMetadata(idOrName, name string) (Notebook, error) {
	idOrName = strings.TrimSpace(idOrName)
	name = strings.TrimSpace(name)
	if idOrName == "" {
		return Notebook{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	nb := s.findNotebookLocked(idOrName)
	if nb == nil {
		now := time.Now().UTC()
		nb = &Notebook{
			ID:         s.nextID("notebook"),
			Name:       firstNonEmpty(name, idOrName),
			WorkGroup:  "primary",
			Type:       "IPYNB",
			CreatedAt:  now,
			ModifiedAt: now,
		}
		s.notebooks[nb.Name] = nb
		return *nb, nil
	}
	if name != "" && name != nb.Name {
		delete(s.notebooks, nb.Name)
		nb.Name = name
		s.notebooks[nb.Name] = nb
	}
	nb.ModifiedAt = time.Now().UTC()
	return *nb, nil
}

func (s *Service) GetNotebookMetadata(idOrName string) (Notebook, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return Notebook{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	nb := s.findNotebookLocked(idOrName)
	if nb == nil {
		return Notebook{}, ErrInvalidParameter
	}
	return *nb, nil
}

func (s *Service) ListNotebookMetadata(workGroup, nameFilter string) []Notebook {
	workGroup = strings.TrimSpace(workGroup)
	nameFilter = strings.TrimSpace(nameFilter)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Notebook, 0, len(s.notebooks))
	for _, nb := range s.notebooks {
		if workGroup != "" && nb.WorkGroup != workGroup {
			continue
		}
		if nameFilter != "" && !strings.Contains(nb.Name, nameFilter) {
			continue
		}
		out = append(out, *nb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteNotebook(idOrName string) error {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if nb, ok := s.notebooks[idOrName]; ok {
		delete(s.notebooks, nb.Name)
		return nil
	}
	for key, nb := range s.notebooks {
		if nb.ID == idOrName {
			delete(s.notebooks, key)
			return nil
		}
	}
	return nil
}

func (s *Service) ExportNotebook(idOrName string) (Notebook, error) {
	nb, err := s.GetNotebookMetadata(idOrName)
	if err != nil {
		now := time.Now().UTC()
		return Notebook{
			ID:         strings.TrimSpace(idOrName),
			Name:       strings.TrimSpace(idOrName),
			WorkGroup:  "primary",
			Type:       "IPYNB",
			Payload:    "",
			CreatedAt:  now,
			ModifiedAt: now,
		}, nil
	}
	return nb, nil
}

func (s *Service) StartCalculationExecution(sessionID, description, codeBlock string) (CalculationExecution, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "stackyard"
	}
	now := time.Now().UTC()
	calc := CalculationExecution{
		ID:          s.nextID("calc"),
		SessionID:   sessionID,
		Description: description,
		CodeBlock:   codeBlock,
		State:       "COMPLETED",
		SubmittedAt: now,
		CompletedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calculations[calc.ID] = &calc
	return calc, nil
}

func (s *Service) GetCalculationExecution(id string) (CalculationExecution, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CalculationExecution{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if calc, ok := s.calculations[id]; ok {
		return *calc, nil
	}
	now := time.Now().UTC()
	calc := &CalculationExecution{
		ID:          id,
		SessionID:   "stackyard",
		CodeBlock:   "",
		State:       "COMPLETED",
		SubmittedAt: now,
		CompletedAt: now,
	}
	s.calculations[id] = calc
	return *calc, nil
}

func (s *Service) ListCalculationExecutions(sessionID, stateFilter string) []CalculationExecution {
	sessionID = strings.TrimSpace(sessionID)
	stateFilter = strings.TrimSpace(stateFilter)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CalculationExecution, 0, len(s.calculations))
	for _, calc := range s.calculations {
		if sessionID != "" && calc.SessionID != sessionID {
			continue
		}
		if stateFilter != "" && calc.State != stateFilter {
			continue
		}
		out = append(out, *calc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) StopCalculationExecution(id string) (CalculationExecution, error) {
	calc, err := s.GetCalculationExecution(id)
	if err != nil {
		return CalculationExecution{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.calculations[calc.ID]
	c.State = "CANCELLED"
	c.CompletedAt = time.Now().UTC()
	return *c, nil
}

func (s *Service) StartSession(workGroup, description string) (Session, error) {
	workGroup = strings.TrimSpace(workGroup)
	if workGroup == "" {
		workGroup = "primary"
	}
	now := time.Now().UTC()
	sess := Session{
		ID:            s.nextID("session"),
		Description:   description,
		WorkGroup:     workGroup,
		State:         "IDLE",
		EngineVersion: "Athena engine version 3",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = &sess
	return sess, nil
}

func (s *Service) GetSession(id string) (Session, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Session{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		return *sess, nil
	}
	now := time.Now().UTC()
	sess := &Session{
		ID:            id,
		WorkGroup:     "primary",
		State:         "IDLE",
		EngineVersion: "Athena engine version 3",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.sessions[id] = sess
	return *sess, nil
}

func (s *Service) GetSessionStatus(id string) (Session, error) {
	return s.GetSession(id)
}

func (s *Service) ListSessions(workGroup, stateFilter string) []Session {
	workGroup = strings.TrimSpace(workGroup)
	stateFilter = strings.TrimSpace(stateFilter)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		if workGroup != "" && sess.WorkGroup != workGroup {
			continue
		}
		if stateFilter != "" && sess.State != stateFilter {
			continue
		}
		out = append(out, *sess)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) TerminateSession(id string) (Session, error) {
	sess, err := s.GetSession(id)
	if err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.sessions[sess.ID]
	cur.State = "TERMINATED"
	cur.UpdatedAt = time.Now().UTC()
	return *cur, nil
}

func (s *Service) ListExecutors(sessionID, stateFilter string) []string {
	_, _ = s.GetSession(sessionID)
	if strings.TrimSpace(stateFilter) != "" && strings.TrimSpace(stateFilter) != "IDLE" {
		return nil
	}
	return []string{"executor-1"}
}

func (s *Service) BatchGetPreparedStatement(workGroup string, names []string) ([]PreparedStatement, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.preparedStatements[workGroup]
	out := make([]PreparedStatement, 0, len(names))
	missing := make([]string, 0)
	for _, name := range names {
		ps, ok := list[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		out = append(out, *ps)
	}
	sort.Strings(missing)
	return out, missing
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func defaultCatalog(name string) string {
	if strings.TrimSpace(name) == "" {
		return "AwsDataCatalog"
	}
	return name
}
