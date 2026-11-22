package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (s *Server) handleRDSDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRDSDataRESTCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "rds-data"); !ok {
		respondRDSDataError(w, status, code, msg)
		return true
	}
	op, ok := rdsdataOperationByPath[rawRequestPath(r)]
	if !ok || r.Method != http.MethodPost {
		respondRDSDataError(w, http.StatusNotFound, "UnknownOperationException", "unknown RDS Data API operation "+r.Method+" "+rawRequestPath(r))
		return true
	}
	switch op.Name {
	case "ExecuteStatement":
		s.handleRDSDataExecuteStatement(w, r)
	case "BatchExecuteStatement":
		s.handleRDSDataBatchExecuteStatement(w, r)
	case "BeginTransaction":
		s.handleRDSDataBeginTransaction(w, r)
	case "CommitTransaction", "RollbackTransaction":
		s.handleRDSDataFinishTransaction(w, r, op.Name == "CommitTransaction")
	case "ExecuteSql":
		s.handleRDSDataExecuteSQL(w, r)
	}
	return true
}

func isRDSDataRESTCandidate(r *http.Request) bool {
	if sigV4ServiceHint(r) == "rds-data" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.HasPrefix(host, "rds-data.") || strings.Contains(host, ".rds-data.")
}

type rdsDataHTTPError struct {
	status int
	code   string
	msg    string
}

func (e *rdsDataHTTPError) Error() string { return e.msg }

func rdsDataBadRequest(msg string) error {
	return &rdsDataHTTPError{http.StatusBadRequest, "BadRequestException", msg}
}

// rdsDataTargetInfo is a validated resourceArn/secretArn pair. unavailable is
// set when the cluster exists but can't serve statements (HTTP endpoint off,
// or stopped), which commit and rollback must still clean up after.
type rdsDataTargetInfo struct {
	arn         string
	database    string
	unavailable error
}

func (s *Server) rdsDataResolve(resourceARN, secretARN, database, schema string) (rdsDataTargetInfo, error) {
	resourceARN, secretARN = strings.TrimSpace(resourceARN), strings.TrimSpace(secretARN)
	switch {
	case resourceARN == "":
		return rdsDataTargetInfo{}, rdsDataBadRequest("resourceArn is required")
	case len(resourceARN) < 11 || len(resourceARN) > 570:
		return rdsDataTargetInfo{}, rdsDataBadRequest("resourceArn must be 11 to 570 characters")
	case secretARN == "":
		return rdsDataTargetInfo{}, rdsDataBadRequest("secretArn is required")
	case len(secretARN) < 11 || len(secretARN) > 570:
		return rdsDataTargetInfo{}, rdsDataBadRequest("secretArn must be 11 to 570 characters")
	case len(database) > 64 || len(schema) > 64:
		return rdsDataTargetInfo{}, rdsDataBadRequest("database and schema must be at most 64 characters")
	}
	cluster, err := s.rds.DescribeDBClusterByRef(resourceARN)
	if err != nil || cluster.ARN != resourceARN {
		return rdsDataTargetInfo{}, rdsDataBadRequest("DB cluster " + resourceARN + " not found")
	}
	secret, err := s.secretsmgr.DescribeSecret(secretARN)
	if err != nil || secret.DeletedDate != nil {
		return rdsDataTargetInfo{}, &rdsDataHTTPError{http.StatusBadRequest, "SecretsErrorException", "secret " + secretARN + " can't be found or is scheduled for deletion"}
	}
	info := rdsDataTargetInfo{arn: resourceARN, database: strings.TrimSpace(database)}
	if info.database == "" {
		info.database = cluster.DatabaseName
	}
	switch {
	case !cluster.HTTPEndpointEnabled:
		info.unavailable = &rdsDataHTTPError{http.StatusBadRequest, "HttpEndpointNotEnabledException", "HTTP endpoint is not enabled for " + resourceARN}
	case cluster.Status == "stopped" || cluster.Status == "stopping" || cluster.Status == "starting":
		info.unavailable = &rdsDataHTTPError{http.StatusGatewayTimeout, "DatabaseUnavailableException", "DB cluster " + resourceARN + " is " + cluster.Status}
	}
	return info, nil
}

// rdsDataTarget resolves a target that must be able to run statements.
func (s *Server) rdsDataTarget(resourceARN, secretARN, database, schema string) (string, string, error) {
	info, err := s.rdsDataResolve(resourceARN, secretARN, database, schema)
	if err == nil {
		err = info.unavailable
	}
	if err != nil {
		return "", "", err
	}
	return info.arn, info.database, nil
}

func rdsDataSQL(sql string) error {
	switch {
	case strings.TrimSpace(sql) == "":
		return rdsDataBadRequest("sql is required")
	case len(sql) > 65536:
		return rdsDataBadRequest("sql must be at most 65536 characters")
	}
	return nil
}

func rdsDataTxID(id string) error {
	if len(id) > 192 {
		return rdsDataBadRequest("transactionId must be at most 192 characters")
	}
	return nil
}

func respondRDSDataJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRDSDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

// respondRDSDataErr maps engine, transaction and validation errors to the
// documented RDS Data API error codes.
func respondRDSDataErr(w http.ResponseWriter, err error, finishing bool) {
	var httpErr *rdsDataHTTPError
	var sqlErr *rdsDataSQLError
	switch {
	case errors.As(err, &httpErr):
		respondRDSDataError(w, httpErr.status, httpErr.code, httpErr.msg)
	case errors.As(err, &sqlErr) && sqlErr.Unsupported:
		respondRDSDataError(w, http.StatusBadRequest, "BadRequestException", sqlErr.Msg)
	case errors.As(err, &sqlErr):
		respondRDSDataError(w, http.StatusBadRequest, "DatabaseErrorException", sqlErr.Msg)
	case finishing && (errors.Is(err, errRDSDataTxNotFound) || errors.Is(err, errRDSDataTxResource)):
		respondRDSDataError(w, http.StatusNotFound, "NotFoundException", "transaction can't be found: "+err.Error())
	case errors.Is(err, errRDSDataTxNotFound), errors.Is(err, errRDSDataTxResource):
		respondRDSDataError(w, http.StatusNotFound, "TransactionNotFoundException", "transaction can't be found: "+err.Error())
	default:
		respondRDSDataError(w, http.StatusInternalServerError, "InternalServerErrorException", err.Error())
	}
}

func decodeRDSDataBody(w http.ResponseWriter, r *http.Request, out any) bool {
	body, err := readBodyBytes(r)
	if err == nil && len(strings.TrimSpace(string(body))) > 0 {
		err = json.Unmarshal(body, out)
	}
	if err != nil {
		respondRDSDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return false
	}
	return true
}
