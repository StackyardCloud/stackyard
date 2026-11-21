package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type rdsDataField struct {
	IsNull       *bool    `json:"isNull,omitempty"`
	BooleanValue *bool    `json:"booleanValue,omitempty"`
	LongValue    *int64   `json:"longValue,omitempty"`
	DoubleValue  *float64 `json:"doubleValue,omitempty"`
	StringValue  *string  `json:"stringValue,omitempty"`
	BlobValue    *[]byte  `json:"blobValue,omitempty"` // pointer so an empty blob still round-trips
	ArrayValue   any      `json:"arrayValue,omitempty"`
}

type rdsDataParameter struct {
	Name     string       `json:"name"`
	Value    rdsDataField `json:"value"`
	TypeHint string       `json:"typeHint"`
}

type rdsDataResultSetOptions struct {
	DecimalReturnType string `json:"decimalReturnType"`
	LongReturnType    string `json:"longReturnType"`
}

type rdsDataExecuteRequest struct {
	ResourceArn           string                  `json:"resourceArn"`
	SecretArn             string                  `json:"secretArn"`
	SQL                   string                  `json:"sql"`
	Database              string                  `json:"database"`
	Schema                string                  `json:"schema"`
	Parameters            []rdsDataParameter      `json:"parameters"`
	ParameterSets         [][]rdsDataParameter    `json:"parameterSets"`
	TransactionID         string                  `json:"transactionId"`
	IncludeResultMetadata bool                    `json:"includeResultMetadata"`
	ResultSetOptions      rdsDataResultSetOptions `json:"resultSetOptions"`
	FormatRecordsAs       string                  `json:"formatRecordsAs"`
}

type rdsDataColumnMetadata struct {
	Name            string `json:"name"`
	Label           string `json:"label"`
	Type            int    `json:"type"`
	TypeName        string `json:"typeName"`
	TableName       string `json:"tableName"`
	SchemaName      string `json:"schemaName"`
	IsAutoIncrement bool   `json:"isAutoIncrement"`
	IsSigned        bool   `json:"isSigned"`
	IsCurrency      bool   `json:"isCurrency"`
	IsCaseSensitive bool   `json:"isCaseSensitive"`
	Nullable        int    `json:"nullable"`
	Precision       int    `json:"precision"`
	Scale           int    `json:"scale"`
}

var rdsDataTypeHints = map[string]bool{"": true, "JSON": true, "UUID": true, "TIMESTAMP": true, "DATE": true, "TIME": true, "DECIMAL": true}

// rdsDataParams converts SQL parameters to engine values.
func rdsDataParams(in []rdsDataParameter) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for _, param := range in {
		name := strings.TrimSpace(param.Name)
		if name == "" {
			return nil, rdsDataBadRequest("every parameter needs a name")
		}
		if !rdsDataTypeHints[param.TypeHint] {
			return nil, rdsDataBadRequest("typeHint must be one of JSON, UUID, TIMESTAMP, DATE, TIME, DECIMAL")
		}
		v := param.Value
		switch {
		case v.ArrayValue != nil:
			return nil, rdsDataBadRequest("array parameters are not supported")
		case v.IsNull != nil && *v.IsNull:
			out[name] = nil
		case v.BooleanValue != nil:
			out[name] = *v.BooleanValue
		case v.LongValue != nil:
			out[name] = *v.LongValue
		case v.DoubleValue != nil:
			out[name] = *v.DoubleValue
		case v.StringValue != nil:
			out[name] = *v.StringValue
		case v.BlobValue != nil:
			out[name] = append([]byte{}, *v.BlobValue...)
		default:
			return nil, rdsDataBadRequest("parameter " + name + " has no value")
		}
	}
	return out, nil
}

// rdsDataFieldOf renders a stored value, honouring resultSetOptions: longs as
// strings when longReturnType is STRING, and DECIMAL/NUMERIC text as a string
// unless decimalReturnType is DOUBLE_OR_LONG (STRING is the default).
func rdsDataFieldOf(col rdsDataColumn, v any, opts rdsDataResultSetOptions) rdsDataField {
	switch x := v.(type) {
	case nil:
		t := true
		return rdsDataField{IsNull: &t}
	case bool:
		return rdsDataField{BooleanValue: &x}
	case int64:
		if opts.LongReturnType == "STRING" {
			s := strconv.FormatInt(x, 10)
			return rdsDataField{StringValue: &s}
		}
		return rdsDataField{LongValue: &x}
	case float64:
		return rdsDataField{DoubleValue: &x}
	case []byte:
		return rdsDataField{BlobValue: &x}
	case string:
		if col.Kind == "decimal" && opts.DecimalReturnType == "DOUBLE_OR_LONG" {
			if n, err := strconv.ParseInt(x, 10, 64); err == nil {
				return rdsDataField{LongValue: &n}
			}
			if f, ok := rdsDataFloat(x); ok {
				return rdsDataField{DoubleValue: &f}
			}
		}
		return rdsDataField{StringValue: &x}
	}
	return rdsDataField{}
}

// rdsDataJDBCType returns the java.sql.Types code the Data API reports.
func rdsDataJDBCType(typeName string) int {
	switch typeName {
	case "INT", "INTEGER", "SERIAL", "MEDIUMINT", "INT4":
		return 4
	case "BIGINT", "BIGSERIAL", "INT8":
		return -5
	case "SMALLINT", "SMALLSERIAL", "INT2":
		return 5
	case "TINYINT":
		return -6
	case "DOUBLE", "FLOAT8":
		return 8
	case "FLOAT", "REAL", "FLOAT4":
		return 7
	case "DECIMAL":
		return 3
	case "NUMERIC":
		return 2
	case "VARCHAR":
		return 12
	case "BOOL", "BOOLEAN":
		return 16
	case "BIT":
		return -7
	case "BLOB", "BYTEA", "BINARY", "VARBINARY", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB":
		return -2
	case "CHAR":
		return 1
	case "TEXT", "LONGTEXT", "MEDIUMTEXT":
		return -1
	}
	return 12
}

func rdsDataMetadata(cols []rdsDataColumn) []rdsDataColumnMetadata {
	out := make([]rdsDataColumnMetadata, 0, len(cols))
	for _, col := range cols {
		out = append(out, rdsDataColumnMetadata{
			Name:            col.Name,
			Label:           col.Name,
			Type:            rdsDataJDBCType(col.TypeName),
			TypeName:        col.TypeName,
			TableName:       col.Table,
			IsAutoIncrement: col.AutoInc,
			IsSigned:        col.Kind == "long" || col.Kind == "double" || col.Kind == "decimal",
			IsCaseSensitive: col.Kind == "string",
			Nullable:        1,
		})
	}
	return out
}

func rdsDataGenerated(result rdsDataResult) []rdsDataField {
	out := make([]rdsDataField, 0, len(result.Generated))
	for _, v := range result.Generated {
		out = append(out, rdsDataFieldOf(rdsDataColumn{}, v, rdsDataResultSetOptions{}))
	}
	return out
}

// rdsDataFormatted renders a SELECT result as the JSON string returned when
// formatRecordsAs is JSON: an array of objects keyed by column name.
func rdsDataFormatted(result rdsDataResult) (string, error) {
	rows := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		obj := make(map[string]any, len(row))
		for i, v := range row {
			if b, ok := v.([]byte); ok {
				v = base64.StdEncoding.EncodeToString(b)
			}
			obj[result.Columns[i].Name] = v
		}
		rows = append(rows, obj)
	}
	out, err := json.Marshal(rows)
	return string(out), err
}

func (s *Server) handleRDSDataExecuteStatement(w http.ResponseWriter, r *http.Request) {
	var req rdsDataExecuteRequest
	if !decodeRDSDataBody(w, r, &req) {
		return
	}
	resourceARN, database, err := s.rdsDataTarget(req.ResourceArn, req.SecretArn, req.Database, req.Schema)
	if err == nil {
		err = rdsDataExecuteChecks(req)
	}
	var params map[string]any
	if err == nil {
		params, err = rdsDataParams(req.Parameters)
	}
	var results []rdsDataResult
	if err == nil {
		results, err = s.rdsData.run(resourceARN, database, req.TransactionID, []rdsDataStatement{{req.SQL, params}})
	}
	if err != nil {
		respondRDSDataErr(w, err, false)
		return
	}
	result := results[0]
	out := map[string]any{
		"numberOfRecordsUpdated": result.Updated,
		"generatedFields":        rdsDataGenerated(result),
	}
	if result.Query {
		if req.IncludeResultMetadata {
			out["columnMetadata"] = rdsDataMetadata(result.Columns)
		}
		if req.FormatRecordsAs == "JSON" {
			formatted, err := rdsDataFormatted(result)
			if err != nil {
				respondRDSDataErr(w, err, false)
				return
			}
			out["formattedRecords"] = formatted
		} else {
			records := make([][]rdsDataField, 0, len(result.Rows))
			for _, row := range result.Rows {
				fields := make([]rdsDataField, 0, len(row))
				for i, v := range row {
					fields = append(fields, rdsDataFieldOf(result.Columns[i], v, req.ResultSetOptions))
				}
				records = append(records, fields)
			}
			out["records"] = records
		}
	}
	respondRDSDataJSON(w, out)
}

func rdsDataExecuteChecks(req rdsDataExecuteRequest) error {
	if err := rdsDataSQL(req.SQL); err != nil {
		return err
	}
	if err := rdsDataTxID(req.TransactionID); err != nil {
		return err
	}
	switch req.FormatRecordsAs {
	case "", "NONE", "JSON":
	default:
		return rdsDataBadRequest("formatRecordsAs must be NONE or JSON")
	}
	switch req.ResultSetOptions.DecimalReturnType {
	case "", "STRING", "DOUBLE_OR_LONG":
	default:
		return rdsDataBadRequest("resultSetOptions.decimalReturnType must be STRING or DOUBLE_OR_LONG")
	}
	switch req.ResultSetOptions.LongReturnType {
	case "", "STRING", "LONG":
	default:
		return rdsDataBadRequest("resultSetOptions.longReturnType must be STRING or LONG")
	}
	return nil
}

func (s *Server) handleRDSDataBatchExecuteStatement(w http.ResponseWriter, r *http.Request) {
	var req rdsDataExecuteRequest
	if !decodeRDSDataBody(w, r, &req) {
		return
	}
	resourceARN, database, err := s.rdsDataTarget(req.ResourceArn, req.SecretArn, req.Database, req.Schema)
	if err == nil {
		err = rdsDataSQL(req.SQL)
	}
	if err == nil {
		err = rdsDataTxID(req.TransactionID)
	}
	sets := req.ParameterSets
	if len(sets) == 0 {
		sets = [][]rdsDataParameter{nil}
	}
	stmts := make([]rdsDataStatement, 0, len(sets))
	for _, set := range sets {
		if err != nil {
			break
		}
		var params map[string]any
		params, err = rdsDataParams(set)
		stmts = append(stmts, rdsDataStatement{req.SQL, params})
	}
	var results []rdsDataResult
	if err == nil {
		results, err = s.rdsData.run(resourceARN, database, req.TransactionID, stmts)
	}
	if err != nil {
		respondRDSDataErr(w, err, false)
		return
	}
	updates := make([]map[string]any, 0, len(results))
	if len(req.ParameterSets) > 0 {
		for _, result := range results {
			updates = append(updates, map[string]any{"generatedFields": rdsDataGenerated(result)})
		}
	}
	respondRDSDataJSON(w, map[string]any{"updateResults": updates})
}

func (s *Server) handleRDSDataBeginTransaction(w http.ResponseWriter, r *http.Request) {
	var req rdsDataExecuteRequest
	if !decodeRDSDataBody(w, r, &req) {
		return
	}
	resourceARN, database, err := s.rdsDataTarget(req.ResourceArn, req.SecretArn, req.Database, req.Schema)
	if err != nil {
		respondRDSDataErr(w, err, false)
		return
	}
	respondRDSDataJSON(w, map[string]string{"transactionId": s.rdsData.begin(resourceARN, database)})
}

func (s *Server) handleRDSDataFinishTransaction(w http.ResponseWriter, r *http.Request, commit bool) {
	var req rdsDataExecuteRequest
	if !decodeRDSDataBody(w, r, &req) {
		return
	}
	// Commit and rollback still clean up when the endpoint is disabled or the
	// cluster is stopped: rollback succeeds, commit discards the transaction
	// and reports why it couldn't commit.
	info, err := s.rdsDataResolve(req.ResourceArn, req.SecretArn, "", "")
	if err == nil && strings.TrimSpace(req.TransactionID) == "" {
		err = rdsDataBadRequest("transactionId is required")
	}
	if err == nil {
		err = rdsDataTxID(req.TransactionID)
	}
	if err == nil {
		err = s.rdsData.finish(req.TransactionID, info.arn, commit && info.unavailable == nil)
		if err == nil && commit && info.unavailable != nil {
			err = info.unavailable
		}
	}
	if err != nil {
		respondRDSDataErr(w, err, true)
		return
	}
	status := "Rollback Complete"
	if commit {
		status = "Transaction Committed"
	}
	respondRDSDataJSON(w, map[string]string{"transactionStatus": status})
}

// handleRDSDataExecuteSQL serves the deprecated ExecuteSql operation: several
// semicolon-separated statements run in order, atomically.
// ponytail: one result per statement in the older Value shape; per-statement
// details the real service adds (for example update counts on DDL) are approximated.
func (s *Server) handleRDSDataExecuteSQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DBClusterOrInstanceArn string `json:"dbClusterOrInstanceArn"`
		AwsSecretStoreArn      string `json:"awsSecretStoreArn"`
		SQLStatements          string `json:"sqlStatements"`
		Database               string `json:"database"`
		Schema                 string `json:"schema"`
	}
	if !decodeRDSDataBody(w, r, &req) {
		return
	}
	resourceARN, database, err := s.rdsDataTarget(req.DBClusterOrInstanceArn, req.AwsSecretStoreArn, req.Database, req.Schema)
	if err == nil && strings.TrimSpace(req.SQLStatements) == "" {
		err = rdsDataBadRequest("sqlStatements is required")
	}
	var results []rdsDataResult
	if err == nil {
		var stmts []rdsDataStatement
		for _, sql := range rdsDataSplitStatements(req.SQLStatements) {
			stmts = append(stmts, rdsDataStatement{sql: sql})
		}
		results, err = s.rdsData.run(resourceARN, database, "", stmts)
	}
	if err != nil {
		respondRDSDataErr(w, err, false)
		return
	}
	out := make([]map[string]any, 0, len(results))
	for _, result := range results {
		item := map[string]any{"numberOfRecordsUpdated": result.Updated}
		if result.Query {
			records := make([]map[string]any, 0, len(result.Rows))
			for _, row := range result.Rows {
				values := make([]map[string]any, 0, len(row))
				for _, v := range row {
					values = append(values, rdsDataLegacyValue(v))
				}
				records = append(records, map[string]any{"values": values})
			}
			item["resultFrame"] = map[string]any{
				"resultSetMetadata": map[string]any{"columnCount": len(result.Columns), "columnMetadata": rdsDataMetadata(result.Columns)},
				"records":           records,
			}
		}
		out = append(out, item)
	}
	respondRDSDataJSON(w, map[string]any{"sqlStatementResults": out})
}

// rdsDataLegacyValue renders a value in ExecuteSql's older Value shape.
func rdsDataLegacyValue(v any) map[string]any {
	switch x := v.(type) {
	case nil:
		return map[string]any{"isNull": true}
	case bool:
		return map[string]any{"bitValue": x}
	case int64:
		return map[string]any{"bigIntValue": x}
	case float64:
		return map[string]any{"doubleValue": x}
	case []byte:
		return map[string]any{"blobValue": x}
	}
	return map[string]any{"stringValue": v}
}
