package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Amazon CloudFront KeyValueStore data plane (REST-JSON, signing name
// cloudfront-keyvaluestore). Real clients sign with SigV4A and send requests
// to {accountId}.cloudfront-kvs.global.api.aws; both are accepted here.

const cfKVSSigningName = "cloudfront-keyvaluestore"

func (s *Server) handleCloudFrontKeyValueStoreRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudFrontKeyValueStoreCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, cfKVSSigningName); !ok {
		respondCFKVSError(w, status, code, msg)
		return true
	}

	arn, key, sub, ok := parseCFKVSPath(rawRequestPath(r))
	if !ok {
		respondCFKVSError(w, http.StatusNotImplemented, "NotImplementedException", "unsupported CloudFront KeyValueStore route")
		return true
	}
	switch {
	case sub == "" && r.Method == http.MethodGet:
		s.cfKVSDescribe(w, arn)
	case sub == "keys" && key == "" && r.Method == http.MethodGet:
		s.cfKVSListKeys(w, r, arn)
	case sub == "keys" && key == "" && r.Method == http.MethodPost:
		s.cfKVSUpdateKeys(w, r, arn)
	case sub == "keys" && key != "" && r.Method == http.MethodGet:
		s.cfKVSGetKey(w, arn, key)
	case sub == "keys" && key != "" && r.Method == http.MethodPut:
		s.cfKVSPutKey(w, r, arn, key)
	case sub == "keys" && key != "" && r.Method == http.MethodDelete:
		stats, err := s.cloudfrontKVS.UpdateKeys(arn, r.Header.Get("If-Match"), nil, []string{key})
		s.respondCFKVSStats(w, stats, err)
	default:
		respondCFKVSError(w, http.StatusNotImplemented, "NotImplementedException", "unsupported CloudFront KeyValueStore route")
	}
	return true
}

func isCloudFrontKeyValueStoreCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == cfKVSSigningName {
		return true
	}
	if service != "" {
		return false
	}
	host := strings.ToLower(r.Host)
	return strings.Contains(host, "cloudfront-kvs") && strings.HasPrefix(rawRequestPath(r), "/key-value-stores/")
}

// parseCFKVSPath splits /key-value-stores/{KvsARN}[/keys[/{Key}]]. Both
// labels are percent-encoded by clients (the ARN contains ':' and '/').
func parseCFKVSPath(path string) (arn, key, sub string, ok bool) {
	rest, found := strings.CutPrefix(path, "/key-value-stores/")
	if !found {
		return "", "", "", false
	}
	segments := strings.Split(rest, "/")
	if len(segments) > 3 || (len(segments) >= 2 && segments[1] != "keys") {
		return "", "", "", false
	}
	arn, err := url.PathUnescape(segments[0])
	if err != nil || arn == "" {
		return "", "", "", false
	}
	if len(segments) >= 2 {
		sub = "keys"
	}
	if len(segments) == 3 {
		if key, err = url.PathUnescape(segments[2]); err != nil || key == "" {
			return "", "", "", false
		}
	}
	return arn, key, sub, true
}

func (s *Server) cfKVSDescribe(w http.ResponseWriter, arn string) {
	kvs, stats, err := s.cloudfrontKVS.DescribeByARN(arn)
	if err != nil {
		respondCFKVSErrorForErr(w, err)
		return
	}
	w.Header().Set("ETag", stats.ETag)
	respondCFKVSJSON(w, http.StatusOK, map[string]any{
		"ItemCount":        stats.ItemCount,
		"TotalSizeInBytes": stats.TotalSize,
		"KvsARN":           kvs.ARN,
		"Created":          awsEpochSeconds(kvs.Created),
		"LastModified":     awsEpochSeconds(kvs.LastModified),
		"Status":           kvs.Status,
	})
}

func (s *Server) cfKVSGetKey(w http.ResponseWriter, arn, key string) {
	value, stats, err := s.cloudfrontKVS.GetKey(arn, key)
	if err != nil {
		respondCFKVSErrorForErr(w, err)
		return
	}
	respondCFKVSJSON(w, http.StatusOK, map[string]any{
		"Key": key, "Value": value, "ItemCount": stats.ItemCount, "TotalSizeInBytes": stats.TotalSize,
	})
}

func (s *Server) cfKVSListKeys(w http.ResponseWriter, r *http.Request, arn string) {
	maxResults := cfKVSDefaultListKeys
	if raw := strings.TrimSpace(r.URL.Query().Get("MaxResults")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > cfKVSMaxListKeys {
			respondCFKVSError(w, http.StatusBadRequest, "ValidationException", fmt.Sprintf("MaxResults must be between 1 and %d", cfKVSMaxListKeys))
			return
		}
		maxResults = n
	}
	items, next, err := s.cloudfrontKVS.ListKeys(arn, r.URL.Query().Get("NextToken"), maxResults)
	if err != nil {
		respondCFKVSErrorForErr(w, err)
		return
	}
	out := map[string]any{"Items": items}
	if next != "" {
		out["NextToken"] = next
	}
	respondCFKVSJSON(w, http.StatusOK, out)
}

func (s *Server) cfKVSPutKey(w http.ResponseWriter, r *http.Request, arn, key string) {
	var body struct {
		Value *string `json:"Value"`
	}
	if err := decodeCFKVSBody(r, &body); err != nil {
		respondCFKVSError(w, http.StatusBadRequest, "ValidationException", "invalid request body")
		return
	}
	if body.Value == nil {
		respondCFKVSError(w, http.StatusBadRequest, "ValidationException", "Value is required")
		return
	}
	stats, err := s.cloudfrontKVS.UpdateKeys(arn, r.Header.Get("If-Match"), []cfKVSPut{{Key: key, Value: *body.Value}}, nil)
	s.respondCFKVSStats(w, stats, err)
}

func (s *Server) cfKVSUpdateKeys(w http.ResponseWriter, r *http.Request, arn string) {
	var body struct {
		Puts []struct {
			Key   *string `json:"Key"`
			Value *string `json:"Value"`
		} `json:"Puts"`
		Deletes []struct {
			Key *string `json:"Key"`
		} `json:"Deletes"`
	}
	if err := decodeCFKVSBody(r, &body); err != nil {
		respondCFKVSError(w, http.StatusBadRequest, "ValidationException", "invalid request body")
		return
	}
	puts := make([]cfKVSPut, 0, len(body.Puts))
	for _, p := range body.Puts {
		if p.Key == nil || p.Value == nil {
			respondCFKVSError(w, http.StatusBadRequest, "ValidationException", "each put requires Key and Value")
			return
		}
		puts = append(puts, cfKVSPut{Key: *p.Key, Value: *p.Value})
	}
	deletes := make([]string, 0, len(body.Deletes))
	for _, d := range body.Deletes {
		if d.Key == nil {
			respondCFKVSError(w, http.StatusBadRequest, "ValidationException", "each delete requires Key")
			return
		}
		deletes = append(deletes, *d.Key)
	}
	stats, err := s.cloudfrontKVS.UpdateKeys(arn, r.Header.Get("If-Match"), puts, deletes)
	s.respondCFKVSStats(w, stats, err)
}

func (s *Server) respondCFKVSStats(w http.ResponseWriter, stats cfKVSStats, err error) {
	if err != nil {
		respondCFKVSErrorForErr(w, err)
		return
	}
	w.Header().Set("ETag", stats.ETag)
	respondCFKVSJSON(w, http.StatusOK, map[string]any{"ItemCount": stats.ItemCount, "TotalSizeInBytes": stats.TotalSize})
}

func decodeCFKVSBody(r *http.Request, out any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	return json.Unmarshal(body, out)
}

func awsEpochSeconds(t time.Time) float64 {
	return float64(t.UnixMilli()) / 1000
}

func respondCFKVSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCFKVSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCFKVSJSON(w, status, map[string]string{"__type": code, "Message": msg})
}

func respondCFKVSErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCFKVSNotFound):
		respondCFKVSError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	case errors.Is(err, errCFKVSETagMismatch):
		respondCFKVSError(w, http.StatusConflict, "ConflictException", err.Error())
	case errors.Is(err, errCFKVSQuota):
		respondCFKVSError(w, http.StatusPaymentRequired, "ServiceQuotaExceededException", err.Error())
	case errors.Is(err, errCFKVSInvalid), errors.Is(err, errCFKVSMissingIfMatch):
		respondCFKVSError(w, http.StatusBadRequest, "ValidationException", err.Error())
	default:
		respondCFKVSError(w, http.StatusInternalServerError, "InternalServerException", err.Error())
	}
}
