package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/athena"
)

type athenaError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

type athenaTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type athenaWorkGroup struct {
	Name         string  `json:"Name"`
	State        string  `json:"State"`
	Description  string  `json:"Description,omitempty"`
	CreationTime float64 `json:"CreationTime,omitempty"`
}

type athenaWorkGroupSummary struct {
	Name         string  `json:"Name"`
	State        string  `json:"State"`
	Description  string  `json:"Description,omitempty"`
	CreationTime float64 `json:"CreationTime,omitempty"`
}

type athenaDataCatalog struct {
	Name        string            `json:"Name"`
	Type        string            `json:"Type"`
	Description string            `json:"Description,omitempty"`
	Parameters  map[string]string `json:"Parameters,omitempty"`
}

type athenaDataCatalogSummary struct {
	CatalogName string `json:"CatalogName"`
	Type        string `json:"Type"`
	Description string `json:"Description,omitempty"`
}

type athenaDatabase struct {
	Name        string            `json:"Name"`
	Description string            `json:"Description,omitempty"`
	Parameters  map[string]string `json:"Parameters,omitempty"`
}

type athenaColumn struct {
	Name string `json:"Name"`
	Type string `json:"Type"`
}

type athenaTableMetadata struct {
	Name        string            `json:"Name"`
	Description string            `json:"Description,omitempty"`
	Columns     []athenaColumn    `json:"Columns,omitempty"`
	Parameters  map[string]string `json:"Parameters,omitempty"`
}

type athenaNamedQuery struct {
	NamedQueryId string `json:"NamedQueryId"`
	Name         string `json:"Name"`
	Description  string `json:"Description,omitempty"`
	Database     string `json:"Database"`
	QueryString  string `json:"QueryString"`
	WorkGroup    string `json:"WorkGroup,omitempty"`
}

type athenaPreparedStatement struct {
	StatementName    string  `json:"StatementName"`
	QueryStatement   string  `json:"QueryStatement"`
	WorkGroup        string  `json:"WorkGroup"`
	Description      string  `json:"Description,omitempty"`
	CreationTime     float64 `json:"CreationTime,omitempty"`
	LastModifiedTime float64 `json:"LastModifiedTime,omitempty"`
}

type athenaQueryExecutionStatus struct {
	State              string  `json:"State"`
	StateChangeReason  string  `json:"StateChangeReason,omitempty"`
	SubmissionDateTime float64 `json:"SubmissionDateTime"`
	CompletionDateTime float64 `json:"CompletionDateTime,omitempty"`
}

type athenaQueryExecution struct {
	QueryExecutionId      string                     `json:"QueryExecutionId"`
	Query                 string                     `json:"Query"`
	WorkGroup             string                     `json:"WorkGroup,omitempty"`
	ResultConfiguration   map[string]any             `json:"ResultConfiguration,omitempty"`
	QueryExecutionContext map[string]string          `json:"QueryExecutionContext,omitempty"`
	Status                athenaQueryExecutionStatus `json:"Status"`
	EngineVersion         map[string]string          `json:"EngineVersion,omitempty"`
}

func (s *Server) handleAthenaJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAthenaJSONCandidate(r) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "athena")
	if !ok {
		respondAthenaJSONError(w, status, code, msg)
		return true
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	action := parseAthenaTarget(target)
	if action == "" {
		respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "missing X-Amz-Target")
		return true
	}

	body, err := readBodyBytes(r)
	if err != nil {
		respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "unable to read request body")
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}

	switch action {
	case "CreateWorkGroup":
		var input struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
			State       string `json:"State"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.CreateWorkGroup(input.Name, input.Description, input.State); err != nil {
			switch err {
			case athena.ErrWorkGroupExists:
				respondAthenaJSONError(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
			default:
				respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			}
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetWorkGroup":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		wg, err := s.athena.GetWorkGroup(strings.TrimSpace(input.WorkGroup))
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"WorkGroup": athenaWorkGroup{
				Name:         wg.Name,
				State:        wg.State,
				Description:  wg.Description,
				CreationTime: athenaTimestamp(wg.CreatedAt),
			},
		})
		return true
	case "ListWorkGroups":
		workGroups := s.athena.ListWorkGroups()
		out := make([]athenaWorkGroupSummary, 0, len(workGroups))
		for _, wg := range workGroups {
			out = append(out, athenaWorkGroupSummary{
				Name:         wg.Name,
				State:        wg.State,
				Description:  wg.Description,
				CreationTime: athenaTimestamp(wg.CreatedAt),
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"WorkGroups": out})
		return true
	case "UpdateWorkGroup":
		var input struct {
			WorkGroup   string `json:"WorkGroup"`
			Description string `json:"Description"`
			State       string `json:"State"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.UpdateWorkGroup(strings.TrimSpace(input.WorkGroup), input.Description, input.State); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteWorkGroup":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteWorkGroup(strings.TrimSpace(input.WorkGroup)); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateDataCatalog":
		var input struct {
			Name        string            `json:"Name"`
			Type        string            `json:"Type"`
			Description string            `json:"Description"`
			Parameters  map[string]string `json:"Parameters"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.CreateDataCatalog(input.Name, input.Type, input.Description, input.Parameters); err != nil {
			switch err {
			case athena.ErrDataCatalogExists:
				respondAthenaJSONError(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
			default:
				respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			}
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetDataCatalog":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		cat, err := s.athena.GetDataCatalog(strings.TrimSpace(input.Name))
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"DataCatalog": athenaDataCatalog{
				Name:        cat.Name,
				Type:        cat.Type,
				Description: cat.Description,
				Parameters:  cat.Parameters,
			},
		})
		return true
	case "ListDataCatalogs":
		catalogs := s.athena.ListDataCatalogs()
		out := make([]athenaDataCatalogSummary, 0, len(catalogs))
		for _, cat := range catalogs {
			out = append(out, athenaDataCatalogSummary{
				CatalogName: cat.Name,
				Type:        cat.Type,
				Description: cat.Description,
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"DataCatalogsSummary": out})
		return true
	case "UpdateDataCatalog":
		var input struct {
			Name        string            `json:"Name"`
			Type        string            `json:"Type"`
			Description string            `json:"Description"`
			Parameters  map[string]string `json:"Parameters"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.UpdateDataCatalog(strings.TrimSpace(input.Name), input.Type, input.Description, input.Parameters); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteDataCatalog":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteDataCatalog(strings.TrimSpace(input.Name)); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateDatabase":
		var input struct {
			CatalogName   string `json:"CatalogName"`
			DatabaseInput struct {
				Name        string            `json:"Name"`
				Description string            `json:"Description"`
				Parameters  map[string]string `json:"Parameters"`
			} `json:"DatabaseInput"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.CreateDatabase(input.CatalogName, input.DatabaseInput.Name, input.DatabaseInput.Description, input.DatabaseInput.Parameters); err != nil {
			switch err {
			case athena.ErrDatabaseExists:
				respondAthenaJSONError(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
			default:
				respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			}
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetDatabase":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		db, err := s.athena.GetDatabase(input.CatalogName, input.DatabaseName)
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"Database": athenaDatabase{
				Name:        db.Name,
				Description: db.Description,
				Parameters:  db.Parameters,
			},
		})
		return true
	case "ListDatabases":
		var input struct {
			CatalogName string `json:"CatalogName"`
		}
		_ = json.Unmarshal(body, &input)
		databases := s.athena.ListDatabases(input.CatalogName)
		out := make([]athenaDatabase, 0, len(databases))
		for _, db := range databases {
			out = append(out, athenaDatabase{
				Name:        db.Name,
				Description: db.Description,
				Parameters:  db.Parameters,
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"DatabaseList": out})
		return true
	case "UpdateDatabase":
		var input struct {
			CatalogName   string            `json:"CatalogName"`
			DatabaseName  string            `json:"DatabaseName"`
			Description   string            `json:"Description"`
			Parameters    map[string]string `json:"Parameters"`
			DatabaseInput *struct {
				Description string            `json:"Description"`
				Parameters  map[string]string `json:"Parameters"`
			} `json:"DatabaseInput"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		desc := input.Description
		params := input.Parameters
		if input.DatabaseInput != nil {
			if desc == "" {
				desc = input.DatabaseInput.Description
			}
			if params == nil {
				params = input.DatabaseInput.Parameters
			}
		}
		if err := s.athena.UpdateDatabase(input.CatalogName, input.DatabaseName, desc, params); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteDatabase":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteDatabase(input.CatalogName, input.DatabaseName); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateTable":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
			TableInput   struct {
				Name              string            `json:"Name"`
				Description       string            `json:"Description"`
				Parameters        map[string]string `json:"Parameters"`
				StorageDescriptor struct {
					Columns []struct {
						Name string `json:"Name"`
					} `json:"Columns"`
				} `json:"StorageDescriptor"`
			} `json:"TableInput"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		cols := make([]string, 0, len(input.TableInput.StorageDescriptor.Columns))
		for _, col := range input.TableInput.StorageDescriptor.Columns {
			if col.Name != "" {
				cols = append(cols, col.Name)
			}
		}
		if err := s.athena.CreateTable(input.CatalogName, input.DatabaseName, input.TableInput.Name, input.TableInput.Description, cols, input.TableInput.Parameters); err != nil {
			switch err {
			case athena.ErrTableExists:
				respondAthenaJSONError(w, http.StatusBadRequest, "AlreadyExistsException", err.Error())
			default:
				respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			}
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetTableMetadata":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
			TableName    string `json:"TableName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		tbl, err := s.athena.GetTable(input.CatalogName, input.DatabaseName, input.TableName)
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		meta := athenaTableMetadata{
			Name:        tbl.Name,
			Description: tbl.Description,
			Parameters:  tbl.Parameters,
			Columns:     makeAthenaColumns(tbl.Columns),
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"TableMetadata": meta})
		return true
	case "ListTableMetadata":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		tables := s.athena.ListTables(input.CatalogName, input.DatabaseName)
		out := make([]athenaTableMetadata, 0, len(tables))
		for _, tbl := range tables {
			out = append(out, athenaTableMetadata{
				Name:        tbl.Name,
				Description: tbl.Description,
				Parameters:  tbl.Parameters,
				Columns:     makeAthenaColumns(tbl.Columns),
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"TableMetadataList": out})
		return true
	case "UpdateTable":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
			TableInput   struct {
				Name              string            `json:"Name"`
				Description       string            `json:"Description"`
				Parameters        map[string]string `json:"Parameters"`
				StorageDescriptor struct {
					Columns []struct {
						Name string `json:"Name"`
					} `json:"Columns"`
				} `json:"StorageDescriptor"`
			} `json:"TableInput"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		cols := make([]string, 0, len(input.TableInput.StorageDescriptor.Columns))
		for _, col := range input.TableInput.StorageDescriptor.Columns {
			if col.Name != "" {
				cols = append(cols, col.Name)
			}
		}
		if err := s.athena.UpdateTable(input.CatalogName, input.DatabaseName, input.TableInput.Name, input.TableInput.Description, cols, input.TableInput.Parameters); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteTable":
		var input struct {
			CatalogName  string `json:"CatalogName"`
			DatabaseName string `json:"DatabaseName"`
			TableName    string `json:"TableName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteTable(input.CatalogName, input.DatabaseName, input.TableName); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateNamedQuery":
		var input struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
			Database    string `json:"Database"`
			QueryString string `json:"QueryString"`
			WorkGroup   string `json:"WorkGroup"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nq, err := s.athena.CreateNamedQuery(input.Name, input.Database, input.QueryString, input.Description, input.WorkGroup)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NamedQueryId": nq.ID})
		return true
	case "GetNamedQuery":
		var input struct {
			NamedQueryId string `json:"NamedQueryId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nq, err := s.athena.GetNamedQuery(strings.TrimSpace(input.NamedQueryId))
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NamedQuery": athenaNamedQuery{
			NamedQueryId: nq.ID,
			Name:         nq.Name,
			Description:  nq.Description,
			Database:     nq.Database,
			QueryString:  nq.QueryString,
			WorkGroup:    nq.WorkGroup,
		}})
		return true
	case "ListNamedQueries":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
		}
		_ = json.Unmarshal(body, &input)
		ids := s.athena.ListNamedQueries(input.WorkGroup)
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NamedQueryIds": ids})
		return true
	case "UpdateNamedQuery":
		var input struct {
			NamedQueryId string `json:"NamedQueryId"`
			Name         string `json:"Name"`
			Description  string `json:"Description"`
			QueryString  string `json:"QueryString"`
			Database     string `json:"Database"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.UpdateNamedQuery(input.NamedQueryId, input.Name, input.Description, input.QueryString, input.Database); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteNamedQuery":
		var input struct {
			NamedQueryId string `json:"NamedQueryId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteNamedQuery(strings.TrimSpace(input.NamedQueryId)); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreatePreparedStatement":
		var input struct {
			WorkGroup      string `json:"WorkGroup"`
			StatementName  string `json:"StatementName"`
			QueryStatement string `json:"QueryStatement"`
			Description    string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.CreatePreparedStatement(input.WorkGroup, input.StatementName, input.QueryStatement, input.Description); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetPreparedStatement":
		var input struct {
			WorkGroup     string `json:"WorkGroup"`
			StatementName string `json:"StatementName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		ps, err := s.athena.GetPreparedStatement(input.WorkGroup, input.StatementName)
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"PreparedStatement": athenaPreparedStatement{
			StatementName:    ps.Name,
			QueryStatement:   ps.Query,
			WorkGroup:        ps.WorkGroup,
			Description:      ps.Description,
			CreationTime:     athenaTimestamp(ps.CreatedAt),
			LastModifiedTime: athenaTimestamp(ps.UpdatedAt),
		}})
		return true
	case "ListPreparedStatements":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		list := s.athena.ListPreparedStatements(input.WorkGroup)
		out := make([]athenaPreparedStatement, 0, len(list))
		for _, ps := range list {
			out = append(out, athenaPreparedStatement{
				StatementName:    ps.Name,
				QueryStatement:   ps.Query,
				WorkGroup:        ps.WorkGroup,
				Description:      ps.Description,
				CreationTime:     athenaTimestamp(ps.CreatedAt),
				LastModifiedTime: athenaTimestamp(ps.UpdatedAt),
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"PreparedStatements": out})
		return true
	case "UpdatePreparedStatement":
		var input struct {
			WorkGroup      string `json:"WorkGroup"`
			StatementName  string `json:"StatementName"`
			QueryStatement string `json:"QueryStatement"`
			Description    string `json:"Description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.UpdatePreparedStatement(input.WorkGroup, input.StatementName, input.QueryStatement, input.Description); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeletePreparedStatement":
		var input struct {
			WorkGroup     string `json:"WorkGroup"`
			StatementName string `json:"StatementName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeletePreparedStatement(input.WorkGroup, input.StatementName); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "StartQueryExecution":
		var input struct {
			QueryString           string `json:"QueryString"`
			WorkGroup             string `json:"WorkGroup"`
			QueryExecutionContext struct {
				Database string `json:"Database"`
				Catalog  string `json:"Catalog"`
			} `json:"QueryExecutionContext"`
			ResultConfiguration struct {
				OutputLocation string `json:"OutputLocation"`
			} `json:"ResultConfiguration"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		qe, err := s.athena.StartQueryExecution(input.QueryString, input.QueryExecutionContext.Database, input.QueryExecutionContext.Catalog, input.WorkGroup, input.ResultConfiguration.OutputLocation)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"QueryExecutionId": qe.ID})
		return true
	case "GetQueryExecution":
		var input struct {
			QueryExecutionId string `json:"QueryExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		qe, err := s.athena.GetQueryExecution(input.QueryExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"QueryExecution": athenaQueryExecution{
				QueryExecutionId: qe.ID,
				Query:            qe.QueryString,
				WorkGroup:        qe.WorkGroup,
				ResultConfiguration: map[string]any{
					"OutputLocation": qe.OutputLocation,
				},
				QueryExecutionContext: map[string]string{
					"Database": qe.Database,
					"Catalog":  qe.Catalog,
				},
				Status: athenaQueryExecutionStatus{
					State:              qe.Status.State,
					StateChangeReason:  qe.Status.StateChangeReason,
					SubmissionDateTime: athenaTimestamp(qe.Status.SubmissionTime),
					CompletionDateTime: athenaTimestamp(qe.Status.CompletionTime),
				},
				EngineVersion: map[string]string{
					"SelectedEngineVersion": qe.EngineVersion,
				},
			},
		})
		return true
	case "ListQueryExecutions":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
		}
		_ = json.Unmarshal(body, &input)
		ids := s.athena.ListQueryExecutions(input.WorkGroup)
		respondAthenaJSON(w, http.StatusOK, map[string]any{"QueryExecutionIds": ids})
		return true
	case "StopQueryExecution":
		var input struct {
			QueryExecutionId string `json:"QueryExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.StopQueryExecution(input.QueryExecutionId); err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetQueryResults":
		var input struct {
			QueryExecutionId string `json:"QueryExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		qe, err := s.athena.GetQueryExecution(input.QueryExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
			return true
		}
		columnInfo := make([]map[string]string, 0, len(qe.ResultColumnInfo))
		for _, col := range qe.ResultColumnInfo {
			columnInfo = append(columnInfo, map[string]string{"Name": col, "Type": "varchar"})
		}
		rows := make([]map[string]any, 0, len(qe.ResultRows))
		for _, row := range qe.ResultRows {
			data := make([]map[string]string, 0, len(row))
			for _, val := range row {
				data = append(data, map[string]string{"VarCharValue": val})
			}
			rows = append(rows, map[string]any{"Data": data})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"UpdateCount": 0,
			"ResultSet": map[string]any{
				"ResultSetMetadata": map[string]any{"ColumnInfo": columnInfo},
				"Rows":              rows,
			},
		})
		return true
	case "BatchGetQueryExecution":
		var input struct {
			QueryExecutionIds []string `json:"QueryExecutionIds"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		executions := make([]athenaQueryExecution, 0, len(input.QueryExecutionIds))
		unprocessed := make([]map[string]string, 0)
		for _, id := range input.QueryExecutionIds {
			qe, err := s.athena.GetQueryExecution(id)
			if err != nil {
				unprocessed = append(unprocessed, map[string]string{
					"QueryExecutionId": id,
					"ErrorCode":        "INVALID_INPUT",
					"ErrorMessage":     "query execution not found",
				})
				continue
			}
			executions = append(executions, athenaQueryExecution{
				QueryExecutionId: qe.ID,
				Query:            qe.QueryString,
				WorkGroup:        qe.WorkGroup,
				Status: athenaQueryExecutionStatus{
					State:              qe.Status.State,
					StateChangeReason:  qe.Status.StateChangeReason,
					SubmissionDateTime: athenaTimestamp(qe.Status.SubmissionTime),
					CompletionDateTime: athenaTimestamp(qe.Status.CompletionTime),
				},
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"QueryExecutions":              executions,
			"UnprocessedQueryExecutionIds": unprocessed,
		})
		return true
	case "BatchGetNamedQuery":
		var input struct {
			NamedQueryIds []string `json:"NamedQueryIds"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		namedQueries := make([]athenaNamedQuery, 0, len(input.NamedQueryIds))
		unprocessed := make([]map[string]string, 0)
		for _, id := range input.NamedQueryIds {
			nq, err := s.athena.GetNamedQuery(id)
			if err != nil {
				unprocessed = append(unprocessed, map[string]string{
					"NamedQueryId": id,
					"ErrorCode":    "INVALID_INPUT",
					"ErrorMessage": "named query not found",
				})
				continue
			}
			namedQueries = append(namedQueries, athenaNamedQuery{
				NamedQueryId: nq.ID,
				Name:         nq.Name,
				Description:  nq.Description,
				Database:     nq.Database,
				QueryString:  nq.QueryString,
				WorkGroup:    nq.WorkGroup,
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"NamedQueries":             namedQueries,
			"UnprocessedNamedQueryIds": unprocessed,
		})
		return true
	case "CreateCapacityReservation":
		var input struct {
			Name       string `json:"Name"`
			TargetDpus int    `json:"TargetDpus"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.CreateCapacityReservation(input.Name, input.TargetDpus); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UpdateCapacityReservation":
		var input struct {
			Name       string `json:"Name"`
			TargetDpus int    `json:"TargetDpus"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.UpdateCapacityReservation(input.Name, input.TargetDpus); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetCapacityReservation":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		cr, err := s.athena.GetCapacityReservation(input.Name)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"CapacityReservation": map[string]any{
				"Name":          cr.Name,
				"Status":        cr.Status,
				"TargetDpus":    cr.TargetDpus,
				"AllocatedDpus": cr.AllocatedDpus,
				"CreationTime":  athenaTimestamp(cr.CreatedAt),
				"LastAllocation": map[string]any{
					"Status":                cr.Status,
					"RequestTime":           athenaTimestamp(cr.UpdatedAt),
					"RequestCompletionTime": athenaTimestamp(cr.UpdatedAt),
				},
			},
		})
		return true
	case "ListCapacityReservations":
		list := s.athena.ListCapacityReservations()
		out := make([]map[string]any, 0, len(list))
		for _, cr := range list {
			out = append(out, map[string]any{
				"Name":          cr.Name,
				"Status":        cr.Status,
				"TargetDpus":    cr.TargetDpus,
				"AllocatedDpus": cr.AllocatedDpus,
				"CreationTime":  athenaTimestamp(cr.CreatedAt),
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"CapacityReservations": out})
		return true
	case "CancelCapacityReservation":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.CancelCapacityReservation(input.Name); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteCapacityReservation":
		var input struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if err := s.athena.DeleteCapacityReservation(input.Name); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "PutCapacityAssignmentConfiguration":
		var input struct {
			CapacityReservationName string `json:"CapacityReservationName"`
			CapacityAssignments     []struct {
				WorkGroupNames []string `json:"WorkGroupNames"`
			} `json:"CapacityAssignments"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		assignments := make([]athena.CapacityAssignment, 0, len(input.CapacityAssignments))
		for _, item := range input.CapacityAssignments {
			assignments = append(assignments, athena.CapacityAssignment{WorkGroupNames: append([]string(nil), item.WorkGroupNames...)})
		}
		s.athena.PutCapacityAssignmentConfiguration(input.CapacityReservationName, assignments)
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetCapacityAssignmentConfiguration":
		var input struct {
			CapacityReservationName string `json:"CapacityReservationName"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		assignments := s.athena.GetCapacityAssignmentConfiguration(input.CapacityReservationName)
		out := make([]map[string]any, 0, len(assignments))
		for _, item := range assignments {
			out = append(out, map[string]any{"WorkGroupNames": item.WorkGroupNames})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"CapacityAssignmentConfiguration": map[string]any{
				"CapacityReservationName": input.CapacityReservationName,
				"CapacityAssignments":     out,
			},
		})
		return true
	case "CreateNotebook":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
			Name      string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nb, err := s.athena.CreateNotebook(input.WorkGroup, input.Name)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NotebookId": nb.ID})
		return true
	case "ImportNotebook":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
			Name      string `json:"Name"`
			Payload   string `json:"Payload"`
			Type      string `json:"Type"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nb, err := s.athena.ImportNotebook(input.WorkGroup, input.Name, input.Payload, input.Type)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NotebookId": nb.ID})
		return true
	case "UpdateNotebook":
		var input struct {
			NotebookId string `json:"NotebookId"`
			Payload    string `json:"Payload"`
			Type       string `json:"Type"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.UpdateNotebook(input.NotebookId, input.Payload, input.Type); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UpdateNotebookMetadata":
		var input struct {
			NotebookId string `json:"NotebookId"`
			Name       string `json:"Name"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if _, err := s.athena.UpdateNotebookMetadata(input.NotebookId, input.Name); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetNotebookMetadata":
		var input struct {
			NotebookId string `json:"NotebookId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nb, err := s.athena.GetNotebookMetadata(input.NotebookId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"NotebookMetadata": map[string]any{
				"NotebookId":       nb.ID,
				"Name":             nb.Name,
				"WorkGroup":        nb.WorkGroup,
				"CreationTime":     athenaTimestamp(nb.CreatedAt),
				"Type":             nb.Type,
				"LastModifiedTime": athenaTimestamp(nb.ModifiedAt),
			},
		})
		return true
	case "ListNotebookMetadata":
		var input struct {
			WorkGroup string `json:"WorkGroup"`
			Filters   struct {
				Name string `json:"Name"`
			} `json:"Filters"`
		}
		_ = json.Unmarshal(body, &input)
		list := s.athena.ListNotebookMetadata(input.WorkGroup, input.Filters.Name)
		out := make([]map[string]any, 0, len(list))
		for _, nb := range list {
			out = append(out, map[string]any{
				"NotebookId":       nb.ID,
				"Name":             nb.Name,
				"WorkGroup":        nb.WorkGroup,
				"CreationTime":     athenaTimestamp(nb.CreatedAt),
				"Type":             nb.Type,
				"LastModifiedTime": athenaTimestamp(nb.ModifiedAt),
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"NotebookMetadataList": out})
		return true
	case "DeleteNotebook":
		var input struct {
			NotebookId string `json:"NotebookId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		_ = s.athena.DeleteNotebook(input.NotebookId)
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ExportNotebook":
		var input struct {
			NotebookId string `json:"NotebookId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		nb, err := s.athena.ExportNotebook(input.NotebookId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"NotebookMetadata": map[string]any{
				"NotebookId":       nb.ID,
				"Name":             nb.Name,
				"WorkGroup":        nb.WorkGroup,
				"CreationTime":     athenaTimestamp(nb.CreatedAt),
				"Type":             nb.Type,
				"LastModifiedTime": athenaTimestamp(nb.ModifiedAt),
			},
			"Payload": nb.Payload,
		})
		return true
	case "CreatePresignedNotebookUrl":
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"NotebookUrl":             "http://localhost:4566/athena/notebook",
			"AuthToken":               "stackyard-token",
			"AuthTokenExpirationTime": time.Now().UTC().Add(15 * time.Minute).Unix(),
		})
		return true
	case "StartCalculationExecution":
		var input struct {
			SessionId                string `json:"SessionId"`
			Description              string `json:"Description"`
			CodeBlock                string `json:"CodeBlock"`
			CalculationConfiguration struct {
				CodeBlock string `json:"CodeBlock"`
			} `json:"CalculationConfiguration"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		code := input.CodeBlock
		if strings.TrimSpace(code) == "" {
			code = input.CalculationConfiguration.CodeBlock
		}
		calc, err := s.athena.StartCalculationExecution(input.SessionId, input.Description, code)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"CalculationExecutionId": calc.ID,
			"State":                  calc.State,
		})
		return true
	case "GetCalculationExecution":
		var input struct {
			CalculationExecutionId string `json:"CalculationExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		calc, err := s.athena.GetCalculationExecution(input.CalculationExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"CalculationExecutionId": calc.ID,
			"SessionId":              calc.SessionID,
			"Description":            calc.Description,
			"Status": map[string]any{
				"SubmissionDateTime": athenaTimestamp(calc.SubmittedAt),
				"CompletionDateTime": athenaTimestamp(calc.CompletedAt),
				"State":              calc.State,
			},
			"Statistics": map[string]any{
				"DpuExecutionInMillis": 1,
				"Progress":             "100",
			},
			"Result": map[string]any{
				"ResultType": "TEXT",
			},
		})
		return true
	case "GetCalculationExecutionCode":
		var input struct {
			CalculationExecutionId string `json:"CalculationExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		calc, err := s.athena.GetCalculationExecution(input.CalculationExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"CodeBlock": calc.CodeBlock})
		return true
	case "GetCalculationExecutionStatus":
		var input struct {
			CalculationExecutionId string `json:"CalculationExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		calc, err := s.athena.GetCalculationExecution(input.CalculationExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"Status": map[string]any{
				"SubmissionDateTime": athenaTimestamp(calc.SubmittedAt),
				"CompletionDateTime": athenaTimestamp(calc.CompletedAt),
				"State":              calc.State,
			},
			"Statistics": map[string]any{
				"DpuExecutionInMillis": 1,
				"Progress":             "100",
			},
		})
		return true
	case "ListCalculationExecutions":
		var input struct {
			SessionId   string `json:"SessionId"`
			StateFilter string `json:"StateFilter"`
		}
		_ = json.Unmarshal(body, &input)
		list := s.athena.ListCalculationExecutions(input.SessionId, input.StateFilter)
		out := make([]map[string]any, 0, len(list))
		for _, calc := range list {
			out = append(out, map[string]any{
				"CalculationExecutionId": calc.ID,
				"Description":            calc.Description,
				"Status": map[string]any{
					"SubmissionDateTime": athenaTimestamp(calc.SubmittedAt),
					"CompletionDateTime": athenaTimestamp(calc.CompletedAt),
					"State":              calc.State,
				},
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"Calculations": out})
		return true
	case "StopCalculationExecution":
		var input struct {
			CalculationExecutionId string `json:"CalculationExecutionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		calc, err := s.athena.StopCalculationExecution(input.CalculationExecutionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"State": calc.State})
		return true
	case "StartSession":
		var input struct {
			Description string `json:"Description"`
			WorkGroup   string `json:"WorkGroup"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		sess, err := s.athena.StartSession(input.WorkGroup, input.Description)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"SessionId": sess.ID, "State": sess.State})
		return true
	case "GetSession":
		var input struct {
			SessionId string `json:"SessionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		sess, err := s.athena.GetSession(input.SessionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"SessionId":     sess.ID,
			"Description":   sess.Description,
			"WorkGroup":     sess.WorkGroup,
			"EngineVersion": sess.EngineVersion,
			"Status": map[string]any{
				"StartDateTime":        athenaTimestamp(sess.CreatedAt),
				"LastModifiedDateTime": athenaTimestamp(sess.UpdatedAt),
				"IdleSinceDateTime":    athenaTimestamp(sess.UpdatedAt),
				"State":                sess.State,
			},
			"EngineConfiguration": map[string]any{
				"CoordinatorDpuSize":     1,
				"MaxConcurrentDpus":      2,
				"DefaultExecutorDpuSize": 1,
			},
			"SessionConfiguration": map[string]any{
				"ExecutionRole": "arn:aws:iam::123456789012:role/stackyard-athena",
			},
		})
		return true
	case "GetSessionStatus":
		var input struct {
			SessionId string `json:"SessionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		sess, err := s.athena.GetSessionStatus(input.SessionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"SessionId": sess.ID,
			"Status": map[string]any{
				"StartDateTime":        athenaTimestamp(sess.CreatedAt),
				"LastModifiedDateTime": athenaTimestamp(sess.UpdatedAt),
				"IdleSinceDateTime":    athenaTimestamp(sess.UpdatedAt),
				"State":                sess.State,
			},
		})
		return true
	case "ListSessions":
		var input struct {
			WorkGroup   string `json:"WorkGroup"`
			StateFilter string `json:"StateFilter"`
		}
		_ = json.Unmarshal(body, &input)
		list := s.athena.ListSessions(input.WorkGroup, input.StateFilter)
		out := make([]map[string]any, 0, len(list))
		for _, sess := range list {
			out = append(out, map[string]any{
				"SessionId":     sess.ID,
				"Description":   sess.Description,
				"EngineVersion": map[string]any{"SelectedEngineVersion": sess.EngineVersion, "EffectiveEngineVersion": sess.EngineVersion},
				"Status": map[string]any{
					"StartDateTime":        athenaTimestamp(sess.CreatedAt),
					"LastModifiedDateTime": athenaTimestamp(sess.UpdatedAt),
					"IdleSinceDateTime":    athenaTimestamp(sess.UpdatedAt),
					"State":                sess.State,
				},
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"Sessions": out})
		return true
	case "TerminateSession":
		var input struct {
			SessionId string `json:"SessionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		sess, err := s.athena.TerminateSession(input.SessionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"State": sess.State})
		return true
	case "GetSessionEndpoint":
		var input struct {
			SessionId string `json:"SessionId"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		sess, err := s.athena.GetSession(input.SessionId)
		if err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", err.Error())
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"AuthToken":               "stackyard-session-token",
			"AuthTokenExpirationTime": time.Now().UTC().Add(15 * time.Minute).Unix(),
			"EndpointUrl":             "http://localhost:4566/athena/session/" + sess.ID,
		})
		return true
	case "GetResourceDashboard":
		var input struct {
			ResourceARN string `json:"ResourceARN"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceARN) == "" {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "ResourceARN is required")
			return true
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"Url": "http://localhost:4566/athena/dashboard",
		})
		return true
	case "ListExecutors":
		var input struct {
			SessionId           string `json:"SessionId"`
			ExecutorStateFilter string `json:"ExecutorStateFilter"`
		}
		_ = json.Unmarshal(body, &input)
		ids := s.athena.ListExecutors(input.SessionId, input.ExecutorStateFilter)
		out := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, map[string]any{
				"ExecutorId":    id,
				"ExecutorType":  "COORDINATOR",
				"StartDateTime": time.Now().UTC().Unix(),
				"ExecutorState": "IDLE",
				"ExecutorSize":  1,
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{"SessionId": input.SessionId, "ExecutorsSummary": out})
		return true
	case "ListNotebookSessions":
		var input struct {
			NotebookId string `json:"NotebookId"`
		}
		_ = json.Unmarshal(body, &input)
		sess, _ := s.athena.GetSession("stackyard")
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"NotebookSessionsList": []map[string]any{{"SessionId": sess.ID, "CreationTime": athenaTimestamp(sess.CreatedAt)}},
		})
		return true
	case "GetQueryRuntimeStatistics":
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"QueryRuntimeStatistics": map[string]any{
				"Timeline": map[string]any{
					"QueryQueueTimeInMillis":           1,
					"ServicePreProcessingTimeInMillis": 1,
					"QueryPlanningTimeInMillis":        1,
					"EngineExecutionTimeInMillis":      1,
					"ServiceProcessingTimeInMillis":    1,
					"TotalExecutionTimeInMillis":       1,
				},
				"Rows": map[string]any{
					"InputRows":   1,
					"InputBytes":  1,
					"OutputBytes": 1,
					"OutputRows":  1,
				},
			},
		})
		return true
	case "ListApplicationDPUSizes", "ListApplicationDpuSizes":
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"ApplicationDPUSizes": []map[string]any{{
				"ApplicationRuntimeId": "AthenaNotebook",
				"SupportedDPUSizes":    []int{1, 2, 4},
			}},
		})
		return true
	case "ListEngineVersions":
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"EngineVersions": []map[string]any{{
				"SelectedEngineVersion":  "Athena engine version 3",
				"EffectiveEngineVersion": "Athena engine version 3",
			}},
		})
		return true
	case "BatchGetPreparedStatement":
		var input struct {
			WorkGroup              string   `json:"WorkGroup"`
			PreparedStatementNames []string `json:"PreparedStatementNames"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		statements, missing := s.athena.BatchGetPreparedStatement(input.WorkGroup, input.PreparedStatementNames)
		outStatements := make([]map[string]any, 0, len(statements))
		for _, ps := range statements {
			outStatements = append(outStatements, map[string]any{
				"StatementName":    ps.Name,
				"QueryStatement":   ps.Query,
				"WorkGroupName":    ps.WorkGroup,
				"Description":      ps.Description,
				"LastModifiedTime": athenaTimestamp(ps.UpdatedAt),
			})
		}
		outMissing := make([]map[string]any, 0, len(missing))
		for _, name := range missing {
			outMissing = append(outMissing, map[string]any{
				"StatementName": name,
				"ErrorCode":     "INVALID_INPUT",
				"ErrorMessage":  "prepared statement not found",
			})
		}
		respondAthenaJSON(w, http.StatusOK, map[string]any{
			"PreparedStatements":                outStatements,
			"UnprocessedPreparedStatementNames": outMissing,
		})
		return true
	case "TagResource":
		var input struct {
			ResourceARN string      `json:"ResourceARN"`
			Tags        []athenaTag `json:"Tags"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		tags := map[string]string{}
		for _, tag := range input.Tags {
			if tag.Key != "" {
				tags[tag.Key] = tag.Value
			}
		}
		s.athena.TagResource(input.ResourceARN, tags)
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		var input struct {
			ResourceARN string   `json:"ResourceARN"`
			TagKeys     []string `json:"TagKeys"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		s.athena.UntagResource(input.ResourceARN, input.TagKeys)
		respondAthenaJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		var input struct {
			ResourceARN string `json:"ResourceARN"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondAthenaJSONError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON")
			return true
		}
		tags := s.athena.ListTags(input.ResourceARN)
		out := make([]athenaTag, 0, len(tags))
		for k, v := range tags {
			out = append(out, athenaTag{Key: k, Value: v})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
		respondAthenaJSON(w, http.StatusOK, map[string]any{"Tags": out})
		return true
	default:
		respondAthenaJSONError(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func athenaTimestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / float64(time.Second)
}

func makeAthenaColumns(columns []string) []athenaColumn {
	if len(columns) == 0 {
		return nil
	}
	out := make([]athenaColumn, 0, len(columns))
	for _, col := range columns {
		out = append(out, athenaColumn{Name: col, Type: "string"})
	}
	return out
}

func respondAthenaJSON(w http.ResponseWriter, status int, body any) {
	respondJSON(w, status, body)
}

func respondAthenaJSONError(w http.ResponseWriter, status int, code, msg string) {
	respondAthenaJSON(w, status, athenaError{Type: code, Message: msg})
}

func isAthenaJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AmazonAthena") {
		return true
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AmazonAthena")
	}
	return false
}

func parseAthenaTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonAthena.") {
		return strings.TrimPrefix(target, "AmazonAthena.")
	}
	if strings.HasPrefix(target, "AmazonAthena_2017-05-18.") {
		return strings.TrimPrefix(target, "AmazonAthena_2017-05-18.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}
