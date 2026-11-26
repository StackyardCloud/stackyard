package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	keyspacessvc "github.com/stackyard/stackyard/internal/services/keyspaces"
)

// Amazon Keyspaces Streams: JSON 1.0, signed cassandra like Keyspaces itself,
// claimed by its X-Amz-Target prefix ahead of the Keyspaces router. Streams
// are the CDC streams of Keyspaces tables (cdcSpecification on CreateTable /
// UpdateTable, internal/services/keyspaces/cdc.go); each stream has one
// shard. Stackyard has no CQL endpoint, so nothing ever writes rows and every
// shard is empty but valid: iterators work, GetRecords returns no records
// and reports AT_TIP. A disabled stream's shard is closed (it has an ending
// sequence number and GetRecords returns no next iterator).
// This service keeps no state of its own: iterators are self-describing.

const (
	kssMaxBody         = 64 << 10
	kssContentType     = "application/x-amz-json-1.0"
	kssTargetPrefix    = "KeyspacesStreams."
	kssIteratorTTL     = 15 * time.Minute
	kssListTokenDigits = 80 // StreamArnToken is at least 80 characters
	kssDefaultMax      = 100
)

type kssErr struct {
	code, msg, errorCode string
}

func kssInvalid(errorCode, format string, args ...any) *kssErr {
	return &kssErr{code: "ValidationException", msg: fmt.Sprintf(format, args...), errorCode: errorCode}
}

func kssNotFound(format string, args ...any) *kssErr {
	return &kssErr{code: "ResourceNotFoundException", msg: fmt.Sprintf(format, args...)}
}

func isKeyspacesStreamsCandidate(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), kssTargetPrefix)
}

func (s *Server) handleKeyspacesStreamsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isKeyspacesStreamsCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "cassandra"); !ok {
		kssRespondStatus(w, status, &kssErr{code: code, msg: msg})
		return true
	}
	op := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), kssTargetPrefix)
	m := awsmodels.KeyspacesStreamsModel()
	shapes, known := m.Operations[op]
	if !known {
		kssRespond(w, &kssErr{code: "UnknownOperationException", msg: "unknown operation " + op})
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > kssMaxBody {
		kssRespond(w, kssInvalid("InvalidFormat", "request body must declare a Content-Length of at most %d bytes", kssMaxBody))
		return true
	}
	in := map[string]any{}
	raw, err := readBodyBytes(r)
	if err != nil {
		kssRespond(w, kssInvalid("InvalidFormat", "unable to read request body"))
		return true
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			kssRespond(w, kssInvalid("InvalidFormat", "request body is not a JSON object"))
			return true
		}
	}
	if problems := m.Validate(shapes.Input, in); len(problems) > 0 {
		kssRespond(w, kssInvalid("InvalidFormat", "%s", strings.Join(problems, "; ")))
		return true
	}
	var out map[string]any
	var kerr *kssErr
	switch op {
	case "ListStreams":
		out, kerr = s.kssListStreams(in)
	case "GetStream":
		out, kerr = s.kssGetStream(in)
	case "GetShardIterator":
		out, kerr = s.kssGetShardIterator(in, time.Now())
	case "GetRecords":
		out, kerr = s.kssGetRecords(in, time.Now())
	}
	if kerr != nil {
		kssRespond(w, kerr)
		return true
	}
	body, _ := json.Marshal(m.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", kssContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func kssRespond(w http.ResponseWriter, e *kssErr) {
	kssRespondStatus(w, http.StatusBadRequest, e)
}

func kssRespondStatus(w http.ResponseWriter, status int, e *kssErr) {
	payload := map[string]string{"__type": e.code, "message": e.msg}
	if e.errorCode != "" {
		payload["errorCode"] = e.errorCode
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", kssContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func kssString(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

// kssMaxResults reads an already-validated maxResults, falling back to def.
func kssMaxResults(in map[string]any, def int) int {
	if n, ok := in["maxResults"].(float64); ok && n >= 1 && n <= float64(def) {
		return int(n)
	}
	return def
}

func (s *Server) kssListStreams(in map[string]any) (map[string]any, *kssErr) {
	ks, table := kssString(in, "keyspaceName"), kssString(in, "tableName")
	if table != "" && ks == "" {
		return nil, kssInvalid("InvalidFormat", "tableName requires keyspaceName")
	}
	streams := s.keyspaces.Streams(ks, table)
	start := 0
	if tok := kssString(in, "nextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || len(tok) != kssListTokenDigits || n < 0 {
			return nil, kssInvalid("InvalidFormat", "invalid nextToken")
		}
		start = min(int(n), len(streams))
	}
	end := min(start+kssMaxResults(in, kssDefaultMax), len(streams))
	list := make([]any, 0, end-start)
	for _, st := range streams[start:end] {
		list = append(list, map[string]any{
			"streamArn": st.ARN, "keyspaceName": st.KeyspaceName, "tableName": st.TableName, "streamLabel": st.Label,
		})
	}
	out := map[string]any{"streams": list}
	if end < len(streams) {
		out["nextToken"] = fmt.Sprintf("%0*d", kssListTokenDigits, end)
	}
	return out, nil
}

func (s *Server) kssStream(arn string) (keyspacessvc.Stream, *kssErr) {
	st, err := s.keyspaces.StreamByARN(arn)
	if err != nil {
		return keyspacessvc.Stream{}, kssNotFound("stream %s not found", arn)
	}
	return st, nil
}

// kssShard describes the single shard of a stream.
type kssShard struct {
	id, start, end string // end is empty while the shard is open
}

func kssShardOf(st keyspacessvc.Stream) kssShard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(st.ARN))
	ms := st.Created.UnixMilli()
	sh := kssShard{
		id:    fmt.Sprintf("shardId-%020d-%08x", ms, h.Sum32()),
		start: fmt.Sprintf("%021d", ms),
	}
	if st.Status != "ENABLED" {
		sh.end = sh.start // no records were ever written
	}
	return sh
}

func (s *Server) kssGetStream(in map[string]any) (map[string]any, *kssErr) {
	st, kerr := s.kssStream(kssString(in, "streamArn"))
	if kerr != nil {
		return nil, kerr
	}
	if kssString(in, "nextToken") != "" {
		// Every stream has one shard, so no token is ever handed out.
		return nil, kssInvalid("ExpiredNextToken", "the nextToken is not valid for this stream")
	}
	sh := kssShardOf(st)
	shards := []any{}
	if filter, ok := in["shardFilter"].(map[string]any); ok {
		// CHILD_SHARDS: shards never split here, so no shard has children.
		if id, _ := filter["shardId"].(string); id != "" && id != sh.id {
			return nil, kssNotFound("shard %s not found", id)
		}
	} else {
		rng := map[string]any{"startingSequenceNumber": sh.start}
		if sh.end != "" {
			rng["endingSequenceNumber"] = sh.end
		}
		shards = append(shards, map[string]any{"shardId": sh.id, "sequenceNumberRange": rng})
	}
	return map[string]any{
		"streamArn": st.ARN, "streamLabel": st.Label, "streamStatus": st.Status, "streamViewType": st.ViewType,
		"creationRequestDateTime": float64(st.Created.UnixMilli()) / 1000,
		"keyspaceName":            st.KeyspaceName, "tableName": st.TableName, "shards": shards,
	}, nil
}

// kssIterator is the decoded form of a shard iterator.
type kssIterator struct {
	arn, shard, position string
	expires              int64
}

func (it kssIterator) encode() string {
	raw := strings.Join([]string{it.arn, it.shard, it.position, strconv.FormatInt(it.expires, 10)}, "\n")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func kssDecodeIterator(tok string) (kssIterator, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return kssIterator{}, false
	}
	parts := strings.Split(string(raw), "\n")
	if len(parts) != 4 {
		return kssIterator{}, false
	}
	exp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return kssIterator{}, false
	}
	return kssIterator{arn: parts[0], shard: parts[1], position: parts[2], expires: exp}, true
}

func (s *Server) kssGetShardIterator(in map[string]any, now time.Time) (map[string]any, *kssErr) {
	st, kerr := s.kssStream(kssString(in, "streamArn"))
	if kerr != nil {
		return nil, kerr
	}
	sh := kssShardOf(st)
	if id := kssString(in, "shardId"); id != sh.id {
		return nil, kssNotFound("shard %s not found in stream %s", id, st.ARN)
	}
	position := sh.start
	switch typ := kssString(in, "shardIteratorType"); typ {
	case "AT_SEQUENCE_NUMBER", "AFTER_SEQUENCE_NUMBER":
		seq := kssString(in, "sequenceNumber")
		n, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			return nil, kssInvalid("InvalidFormat", "%s requires a numeric sequenceNumber", typ)
		}
		start, _ := strconv.ParseUint(sh.start, 10, 64)
		if n < start {
			return nil, kssInvalid("TrimmedDataAccess", "sequence number %s is before the start of the shard", seq)
		}
		position = fmt.Sprintf("%021d", n)
	}
	it := kssIterator{arn: st.ARN, shard: sh.id, position: position, expires: now.Add(kssIteratorTTL).Unix()}
	return map[string]any{"shardIterator": it.encode()}, nil
}

func (s *Server) kssGetRecords(in map[string]any, now time.Time) (map[string]any, *kssErr) {
	it, ok := kssDecodeIterator(kssString(in, "shardIterator"))
	if !ok {
		return nil, kssInvalid("InvalidFormat", "invalid shard iterator")
	}
	if now.Unix() > it.expires {
		return nil, kssInvalid("ExpiredIterator", "the shard iterator has expired")
	}
	st, kerr := s.kssStream(it.arn)
	if kerr != nil {
		return nil, kerr
	}
	sh := kssShardOf(st)
	if it.shard != sh.id {
		return nil, kssNotFound("shard %s not found in stream %s", it.shard, st.ARN)
	}
	out := map[string]any{
		"changeRecords":       []any{},
		"iteratorDescription": map[string]any{"iteratorPosition": "AT_TIP"},
	}
	if sh.end == "" { // an open shard keeps handing out iterators
		it.expires = now.Add(kssIteratorTTL).Unix()
		out["nextShardIterator"] = it.encode()
	}
	return out, nil
}
