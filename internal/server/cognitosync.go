package server

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type cognitoSyncError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCognitoSyncRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCognitoSyncRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cognito-sync")
	if !ok {
		respondCognitoSyncError(w, status, code, msg)
		return true
	}

	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 || segments[0] != "identitypools" {
		respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}

	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return true
		}
		s.handleCognitoSyncListIdentityPoolUsage(w, r)
		return true
	}

	identityPoolID, ok := decodeCognitoSyncPathSegment(segments[1])
	if !ok {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "IdentityPoolId is required")
		return true
	}

	if len(segments) == 2 {
		if r.Method != http.MethodGet {
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return true
		}
		s.handleCognitoSyncDescribeIdentityPoolUsage(w, identityPoolID)
		return true
	}

	switch segments[2] {
	case "events":
		if len(segments) == 3 {
			switch r.Method {
			case http.MethodGet:
				s.handleCognitoSyncGetCognitoEvents(w, identityPoolID)
				return true
			case http.MethodPost:
				s.handleCognitoSyncSetCognitoEvents(w, r, identityPoolID)
				return true
			}
		}
	case "configuration":
		if len(segments) == 3 {
			switch r.Method {
			case http.MethodGet:
				s.handleCognitoSyncGetIdentityPoolConfiguration(w, identityPoolID)
				return true
			case http.MethodPost:
				s.handleCognitoSyncSetIdentityPoolConfiguration(w, r, identityPoolID)
				return true
			}
		}
	case "bulkpublish":
		if len(segments) == 3 && r.Method == http.MethodPost {
			s.handleCognitoSyncBulkPublish(w, identityPoolID)
			return true
		}
	case "getBulkPublishDetails":
		if len(segments) == 3 && r.Method == http.MethodPost {
			s.handleCognitoSyncGetBulkPublishDetails(w, identityPoolID)
			return true
		}
	case "identity":
		if len(segments) == 5 && segments[4] == "device" && r.Method == http.MethodPost {
			identityID, ok := decodeCognitoSyncPathSegment(segments[3])
			if !ok {
				respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "IdentityId is required")
				return true
			}
			s.handleCognitoSyncRegisterDevice(w, r, identityPoolID, identityID)
			return true
		}
	case "identities":
		s.handleCognitoSyncIdentityRoutes(w, r, segments, identityPoolID)
		return true
	}

	respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func (s *Server) handleCognitoSyncIdentityRoutes(w http.ResponseWriter, r *http.Request, segments []string, identityPoolID string) {
	if len(segments) < 4 {
		respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}

	identityID, ok := decodeCognitoSyncPathSegment(segments[3])
	if !ok {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "IdentityId is required")
		return
	}

	if len(segments) == 4 {
		if r.Method != http.MethodGet {
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
		s.handleCognitoSyncDescribeIdentityUsage(w, identityPoolID, identityID)
		return
	}

	if segments[4] != "datasets" {
		respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}

	if len(segments) == 5 {
		if r.Method != http.MethodGet {
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
		s.handleCognitoSyncListDatasets(w, r, identityPoolID, identityID)
		return
	}

	datasetName, ok := decodeCognitoSyncPathSegment(segments[5])
	if !ok {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "DatasetName is required")
		return
	}

	if len(segments) == 6 {
		switch r.Method {
		case http.MethodGet:
			s.handleCognitoSyncDescribeDataset(w, identityPoolID, identityID, datasetName)
			return
		case http.MethodPost:
			s.handleCognitoSyncUpdateRecords(w, r, identityPoolID, identityID, datasetName)
			return
		case http.MethodDelete:
			s.handleCognitoSyncDeleteDataset(w, identityPoolID, identityID, datasetName)
			return
		default:
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 7 && segments[6] == "records" && r.Method == http.MethodGet {
		s.handleCognitoSyncListRecords(w, r, identityPoolID, identityID, datasetName)
		return
	}

	if len(segments) == 8 && segments[6] == "subscriptions" {
		deviceID, ok := decodeCognitoSyncPathSegment(segments[7])
		if !ok {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "DeviceId is required")
			return
		}
		switch r.Method {
		case http.MethodPost:
			s.handleCognitoSyncSubscribeToDataset(w, identityPoolID, identityID, datasetName, deviceID)
			return
		case http.MethodDelete:
			s.handleCognitoSyncUnsubscribeFromDataset(w, identityPoolID, identityID, datasetName, deviceID)
			return
		default:
			respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	respondCognitoSyncError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleCognitoSyncDescribeIdentityPoolUsage(w http.ResponseWriter, identityPoolID string) {
	usage, err := s.cognitosync.DescribeIdentityPoolUsage(identityPoolID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{
		"IdentityPoolUsage": cognitoSyncIdentityPoolUsagePayload(usage),
	})
}

func (s *Server) handleCognitoSyncDescribeIdentityUsage(w http.ResponseWriter, identityPoolID, identityID string) {
	usage, err := s.cognitosync.DescribeIdentityUsage(identityPoolID, identityID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{
		"IdentityUsage": cognitoSyncIdentityUsagePayload(usage),
	})
}

func (s *Server) handleCognitoSyncListIdentityPoolUsage(w http.ResponseWriter, r *http.Request) {
	maxResults, err := optionalCognitoSyncIntQuery(r, "maxResults", 10)
	if err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
		return
	}
	items, nextToken, err := s.cognitosync.ListIdentityPoolUsage(maxResults, strings.TrimSpace(r.URL.Query().Get("nextToken")))
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}

	out := map[string]any{
		"IdentityPoolUsages": cognitoSyncIdentityPoolUsageListPayload(items),
		"Count":              len(items),
		"MaxResults":         maxResults,
	}
	if nextToken != "" {
		out["NextToken"] = nextToken
	}
	respondCognitoSyncJSON(w, http.StatusOK, out)
}

func (s *Server) handleCognitoSyncListDatasets(w http.ResponseWriter, r *http.Request, identityPoolID, identityID string) {
	maxResults, err := optionalCognitoSyncIntQuery(r, "maxResults", 10)
	if err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
		return
	}
	items, nextToken, err := s.cognitosync.ListDatasets(identityPoolID, identityID, maxResults, strings.TrimSpace(r.URL.Query().Get("nextToken")))
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}

	out := map[string]any{
		"Datasets": cognitoSyncDatasetListPayload(items),
		"Count":    len(items),
	}
	if nextToken != "" {
		out["NextToken"] = nextToken
	}
	respondCognitoSyncJSON(w, http.StatusOK, out)
}

func (s *Server) handleCognitoSyncDescribeDataset(w http.ResponseWriter, identityPoolID, identityID, datasetName string) {
	dataset, err := s.cognitosync.DescribeDataset(identityPoolID, identityID, datasetName)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{
		"Dataset": cognitoSyncDatasetPayload(dataset),
	})
}

func (s *Server) handleCognitoSyncListRecords(w http.ResponseWriter, r *http.Request, identityPoolID, identityID, datasetName string) {
	maxResults, err := optionalCognitoSyncIntQuery(r, "maxResults", 1024)
	if err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
		return
	}
	lastSyncCount, err := optionalCognitoSyncInt64Query(r, "lastSyncCount", 0)
	if err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid lastSyncCount")
		return
	}
	syncSessionToken, ok := optionalCognitoSyncDecodedQueryValue(r, "syncSessionToken")
	if !ok {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid syncSessionToken")
		return
	}
	_ = syncSessionToken

	out, err := s.cognitosync.ListRecords(
		identityPoolID,
		identityID,
		datasetName,
		lastSyncCount,
		maxResults,
		strings.TrimSpace(r.URL.Query().Get("nextToken")),
	)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}

	response := map[string]any{
		"Records":                               cognitoSyncRecordListPayload(out.Records),
		"Count":                                 out.Count,
		"DatasetExists":                         out.DatasetExists,
		"DatasetDeletedAfterRequestedSyncCount": out.DatasetDeletedAfterSyncCount,
		"MergedDatasetNames":                    out.MergedDatasetNames,
		"DatasetSyncCount":                      out.DatasetSyncCount,
		"SyncSessionToken":                      out.SyncSessionToken,
		"LastModifiedBy":                        out.LastModifiedBy,
	}
	if out.NextToken != "" {
		response["NextToken"] = out.NextToken
	}
	respondCognitoSyncJSON(w, http.StatusOK, response)
}

func (s *Server) handleCognitoSyncGetCognitoEvents(w http.ResponseWriter, identityPoolID string) {
	events, err := s.cognitosync.GetCognitoEvents(identityPoolID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{"Events": events})
}

func (s *Server) handleCognitoSyncSetCognitoEvents(w http.ResponseWriter, r *http.Request, identityPoolID string) {
	var req struct {
		Events map[string]string `json:"Events"`
	}
	if err := decodeCognitoSyncJSONBody(r, &req); err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
		return
	}
	if req.Events == nil {
		req.Events = map[string]string{}
	}
	events, err := s.cognitosync.SetCognitoEvents(identityPoolID, req.Events)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{"Events": events})
}

func (s *Server) handleCognitoSyncGetIdentityPoolConfiguration(w http.ResponseWriter, identityPoolID string) {
	pushSync, streams, err := s.cognitosync.GetIdentityPoolConfiguration(identityPoolID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, cognitoSyncIdentityPoolConfigurationPayload(identityPoolID, pushSync, streams))
}

func (s *Server) handleCognitoSyncSetIdentityPoolConfiguration(w http.ResponseWriter, r *http.Request, identityPoolID string) {
	var req struct {
		PushSync *struct {
			ApplicationArns []string `json:"ApplicationArns"`
			RoleArn         string   `json:"RoleArn"`
		} `json:"PushSync"`
		CognitoStreams *struct {
			StreamName      string `json:"StreamName"`
			RoleArn         string `json:"RoleArn"`
			StreamingStatus string `json:"StreamingStatus"`
		} `json:"CognitoStreams"`
	}
	if err := decodeCognitoSyncJSONBody(r, &req); err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
		return
	}

	var pushSync *cognitoSyncPushSync
	if req.PushSync != nil {
		if strings.TrimSpace(req.PushSync.RoleArn) == "" {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "PushSync.RoleArn is required")
			return
		}
		if len(req.PushSync.ApplicationArns) > 10 {
			respondCognitoSyncError(w, http.StatusBadRequest, "LimitExceededException", "PushSync.ApplicationArns exceeds maximum size")
			return
		}
		for _, arn := range req.PushSync.ApplicationArns {
			if strings.TrimSpace(arn) == "" {
				respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "PushSync.ApplicationArns must not contain empty values")
				return
			}
		}
		pushSync = &cognitoSyncPushSync{
			ApplicationArns: append([]string(nil), req.PushSync.ApplicationArns...),
			RoleArn:         strings.TrimSpace(req.PushSync.RoleArn),
		}
	}

	var streams *cognitoSyncCognitoStreams
	if req.CognitoStreams != nil {
		if strings.TrimSpace(req.CognitoStreams.StreamName) == "" {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "CognitoStreams.StreamName is required")
			return
		}
		if strings.TrimSpace(req.CognitoStreams.RoleArn) == "" {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "CognitoStreams.RoleArn is required")
			return
		}
		status := strings.ToUpper(strings.TrimSpace(req.CognitoStreams.StreamingStatus))
		if status != "" && status != "ENABLED" && status != "DISABLED" {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "CognitoStreams.StreamingStatus must be ENABLED or DISABLED")
			return
		}
		streams = &cognitoSyncCognitoStreams{
			StreamName:      strings.TrimSpace(req.CognitoStreams.StreamName),
			RoleArn:         strings.TrimSpace(req.CognitoStreams.RoleArn),
			StreamingStatus: status,
		}
	}

	resolvedPushSync, resolvedStreams, err := s.cognitosync.SetIdentityPoolConfiguration(identityPoolID, pushSync, streams)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, cognitoSyncIdentityPoolConfigurationPayload(identityPoolID, resolvedPushSync, resolvedStreams))
}

func (s *Server) handleCognitoSyncRegisterDevice(w http.ResponseWriter, r *http.Request, identityPoolID, identityID string) {
	var req struct {
		Platform string `json:"Platform"`
		Token    string `json:"Token"`
	}
	if err := decodeCognitoSyncJSONBody(r, &req); err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
		return
	}

	platformFromQuery, ok := optionalCognitoSyncDecodedQueryValue(r, "platform")
	if !ok {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid platform")
		return
	}
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = strings.TrimSpace(platformFromQuery)
	}
	if platform != "" && platformFromQuery != "" && !strings.EqualFold(platform, platformFromQuery) {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "platform value mismatch")
		return
	}

	deviceID, err := s.cognitosync.RegisterDevice(identityPoolID, identityID, platform, req.Token)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{"DeviceId": deviceID})
}

func (s *Server) handleCognitoSyncSubscribeToDataset(w http.ResponseWriter, identityPoolID, identityID, datasetName, deviceID string) {
	if err := s.cognitosync.SubscribeToDataset(identityPoolID, identityID, datasetName, deviceID); err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleCognitoSyncUnsubscribeFromDataset(w http.ResponseWriter, identityPoolID, identityID, datasetName, deviceID string) {
	if err := s.cognitosync.UnsubscribeFromDataset(identityPoolID, identityID, datasetName, deviceID); err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleCognitoSyncUpdateRecords(w http.ResponseWriter, r *http.Request, identityPoolID, identityID, datasetName string) {
	var req struct {
		DeviceID         string `json:"DeviceId"`
		SyncSessionToken string `json:"SyncSessionToken"`
		ClientContext    string `json:"ClientContext"`
		RecordPatches    []struct {
			Op                     string          `json:"Op"`
			Key                    string          `json:"Key"`
			Value                  string          `json:"Value"`
			SyncCount              *int64          `json:"SyncCount"`
			DeviceLastModifiedDate json.RawMessage `json:"DeviceLastModifiedDate"`
		} `json:"RecordPatches"`
	}
	if err := decodeCognitoSyncJSONBody(r, &req); err != nil {
		respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
		return
	}

	patches := make([]cognitoSyncRecordPatchInput, 0, len(req.RecordPatches))
	for _, patch := range req.RecordPatches {
		if patch.SyncCount == nil {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "RecordPatch.SyncCount is required")
			return
		}
		deviceLastModified, err := parseCognitoSyncTimeRaw(patch.DeviceLastModifiedDate)
		if err != nil {
			respondCognitoSyncError(w, http.StatusBadRequest, "InvalidParameterException", "RecordPatch.DeviceLastModifiedDate is invalid")
			return
		}
		patches = append(patches, cognitoSyncRecordPatchInput{
			Op:                     patch.Op,
			Key:                    patch.Key,
			Value:                  patch.Value,
			SyncCount:              *patch.SyncCount,
			DeviceLastModifiedDate: deviceLastModified,
		})
	}

	records, err := s.cognitosync.UpdateRecords(identityPoolID, identityID, datasetName, req.DeviceID, req.SyncSessionToken, patches)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{
		"Records": cognitoSyncRecordListPayload(records),
	})
}

func (s *Server) handleCognitoSyncDeleteDataset(w http.ResponseWriter, identityPoolID, identityID, datasetName string) {
	dataset, err := s.cognitosync.DeleteDataset(identityPoolID, identityID, datasetName)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{
		"Dataset": cognitoSyncDatasetPayload(dataset),
	})
}

func (s *Server) handleCognitoSyncBulkPublish(w http.ResponseWriter, identityPoolID string) {
	out, err := s.cognitosync.BulkPublish(identityPoolID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	respondCognitoSyncJSON(w, http.StatusOK, map[string]any{"IdentityPoolId": out})
}

func (s *Server) handleCognitoSyncGetBulkPublishDetails(w http.ResponseWriter, identityPoolID string) {
	details, err := s.cognitosync.GetBulkPublishDetails(identityPoolID)
	if err != nil {
		respondCognitoSyncErrorForErr(w, err)
		return
	}
	response := map[string]any{
		"IdentityPoolId":    details.IdentityPoolID,
		"BulkPublishStatus": details.BulkPublishStatus,
		"FailureMessage":    details.FailureMessage,
	}
	if !details.BulkPublishStart.IsZero() {
		response["BulkPublishStartTime"] = cognitoSyncTimestamp(details.BulkPublishStart)
	}
	if !details.BulkPublishComplete.IsZero() {
		response["BulkPublishCompleteTime"] = cognitoSyncTimestamp(details.BulkPublishComplete)
	}
	respondCognitoSyncJSON(w, http.StatusOK, response)
}

func isCognitoSyncRESTCandidate(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(sigV4ServiceHint(r)), "cognito-sync") {
		return true
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(r.Host)), "cognito-sync") {
		return true
	}
	return strings.HasPrefix(rawRequestPath(r), "/identitypools")
}

func decodeCognitoSyncPathSegment(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func optionalCognitoSyncIntQuery(r *http.Request, key string, def int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def, nil
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strings.TrimSpace(decoded))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func optionalCognitoSyncInt64Query(r *http.Request, key string, def int64) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def, nil
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(decoded), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func optionalCognitoSyncDecodedQueryValue(r *http.Request, key string) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return "", true
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(decoded), true
}

func decodeCognitoSyncJSONBody(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return strconv.ErrSyntax
		}
		return err
	}
	return nil
}

func parseCognitoSyncTimeRaw(raw json.RawMessage) (*time.Time, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var floatVal float64
	if err := json.Unmarshal(raw, &floatVal); err == nil {
		if math.IsNaN(floatVal) || math.IsInf(floatVal, 0) {
			return nil, strconv.ErrSyntax
		}
		sec := math.Floor(floatVal)
		nsec := (floatVal - sec) * float64(time.Second)
		t := time.Unix(int64(sec), int64(nsec)).UTC()
		return &t, nil
	}
	var stringVal string
	if err := json.Unmarshal(raw, &stringVal); err != nil {
		return nil, err
	}
	stringVal = strings.TrimSpace(stringVal)
	if stringVal == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, stringVal); err == nil {
		t := parsed.UTC()
		return &t, nil
	}
	if parsedFloat, err := strconv.ParseFloat(stringVal, 64); err == nil {
		sec := math.Floor(parsedFloat)
		nsec := (parsedFloat - sec) * float64(time.Second)
		t := time.Unix(int64(sec), int64(nsec)).UTC()
		return &t, nil
	}
	return nil, strconv.ErrSyntax
}

func cognitoSyncTimestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixMilli()) / 1000.0
}

func cognitoSyncIdentityPoolUsagePayload(usage cognitoSyncIdentityPoolUsage) map[string]any {
	return map[string]any{
		"IdentityPoolId":    usage.IdentityPoolID,
		"SyncSessionsCount": usage.SyncSessions,
		"DataStorage":       usage.DataStorage,
		"LastModifiedDate":  cognitoSyncTimestamp(usage.LastModifiedDate),
	}
}

func cognitoSyncIdentityPoolUsageListPayload(items []cognitoSyncIdentityPoolUsage) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, cognitoSyncIdentityPoolUsagePayload(item))
	}
	return out
}

func cognitoSyncIdentityUsagePayload(usage cognitoSyncIdentityUsage) map[string]any {
	return map[string]any{
		"IdentityId":       usage.IdentityID,
		"IdentityPoolId":   usage.IdentityPoolID,
		"DatasetCount":     usage.DatasetCount,
		"DataStorage":      usage.DataStorage,
		"LastModifiedDate": cognitoSyncTimestamp(usage.LastModifiedDate),
	}
}

func cognitoSyncDatasetPayload(dataset cognitoSyncDatasetSummary) map[string]any {
	return map[string]any{
		"IdentityId":       dataset.IdentityID,
		"DatasetName":      dataset.Name,
		"CreationDate":     cognitoSyncTimestamp(dataset.CreationDate),
		"LastModifiedDate": cognitoSyncTimestamp(dataset.LastModifiedDate),
		"LastModifiedBy":   dataset.LastModifiedBy,
		"DataStorage":      dataset.DataStorage,
		"NumRecords":       dataset.NumRecords,
	}
}

func cognitoSyncDatasetListPayload(items []cognitoSyncDatasetSummary) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, cognitoSyncDatasetPayload(item))
	}
	return out
}

func cognitoSyncRecordPayload(record cognitoSyncRecordSummary) map[string]any {
	return map[string]any{
		"Key":                    record.Key,
		"Value":                  record.Value,
		"SyncCount":              record.SyncCount,
		"LastModifiedDate":       cognitoSyncTimestamp(record.LastModifiedDate),
		"LastModifiedBy":         record.LastModifiedBy,
		"DeviceLastModifiedDate": cognitoSyncTimestamp(record.DeviceLastModifiedDate),
	}
}

func cognitoSyncRecordListPayload(records []cognitoSyncRecordSummary) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoSyncRecordPayload(record))
	}
	return out
}

func cognitoSyncIdentityPoolConfigurationPayload(identityPoolID string, pushSync *cognitoSyncPushSync, streams *cognitoSyncCognitoStreams) map[string]any {
	out := map[string]any{
		"IdentityPoolId": identityPoolID,
	}
	if pushSync != nil {
		out["PushSync"] = map[string]any{
			"ApplicationArns": append([]string(nil), pushSync.ApplicationArns...),
			"RoleArn":         pushSync.RoleArn,
		}
	}
	if streams != nil {
		out["CognitoStreams"] = map[string]any{
			"StreamName":      streams.StreamName,
			"RoleArn":         streams.RoleArn,
			"StreamingStatus": streams.StreamingStatus,
		}
	}
	return out
}

func respondCognitoSyncJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func respondCognitoSyncError(w http.ResponseWriter, status int, code, msg string) {
	respondCognitoSyncJSON(w, status, cognitoSyncError{Type: code, Message: msg})
}

func respondCognitoSyncErrorForErr(w http.ResponseWriter, err error) {
	if apiErr := asCognitoSyncAPIError(err); apiErr != nil {
		respondCognitoSyncError(w, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	respondCognitoSyncError(w, http.StatusInternalServerError, "InternalErrorException", err.Error())
}
