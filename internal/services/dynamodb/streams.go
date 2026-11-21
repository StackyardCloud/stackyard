package dynamodb

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DynamoDB Streams behaviour, from
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Operations_Amazon_DynamoDB_Streams.html
// and https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Streams.html.

var (
	ErrExpiredIterator   = errors.New("iterator expired")
	ErrTrimmedDataAccess = errors.New("trimmed data access")
)

const (
	streamIteratorTTL = 15 * time.Minute
	streamEnabled     = "ENABLED"
	streamDisabled    = "DISABLED"
)

var streamViewTypes = []string{"KEYS_ONLY", "NEW_IMAGE", "OLD_IMAGE", "NEW_AND_OLD_IMAGES"}

type StreamSpecification struct {
	StreamEnabled  bool   `json:"StreamEnabled"`
	StreamViewType string `json:"StreamViewType,omitempty"`
}

type StreamSummary struct {
	StreamArn   string `json:"StreamArn"`
	StreamLabel string `json:"StreamLabel"`
	TableName   string `json:"TableName"`
}

type SequenceNumberRange struct {
	StartingSequenceNumber string `json:"StartingSequenceNumber"`
	EndingSequenceNumber   string `json:"EndingSequenceNumber,omitempty"`
}

type Shard struct {
	ShardID             string              `json:"ShardId"`
	SequenceNumberRange SequenceNumberRange `json:"SequenceNumberRange"`
}

type StreamDescription struct {
	CreationRequestDateTime float64            `json:"CreationRequestDateTime"`
	KeySchema               []KeySchemaElement `json:"KeySchema"`
	Shards                  []Shard            `json:"Shards"`
	StreamArn               string             `json:"StreamArn"`
	StreamLabel             string             `json:"StreamLabel"`
	StreamStatus            string             `json:"StreamStatus"`
	StreamViewType          string             `json:"StreamViewType"`
	TableName               string             `json:"TableName"`
}

type StreamRecordData struct {
	ApproximateCreationDateTime float64        `json:"ApproximateCreationDateTime"`
	Keys                        map[string]any `json:"Keys"`
	NewImage                    map[string]any `json:"NewImage,omitempty"`
	OldImage                    map[string]any `json:"OldImage,omitempty"`
	SequenceNumber              string         `json:"SequenceNumber"`
	SizeBytes                   int64          `json:"SizeBytes"`
	StreamViewType              string         `json:"StreamViewType"`
}

type StreamRecord struct {
	AwsRegion    string           `json:"awsRegion"`
	Dynamodb     StreamRecordData `json:"dynamodb"`
	EventID      string           `json:"eventID"`
	EventName    string           `json:"eventName"`
	EventSource  string           `json:"eventSource"`
	EventVersion string           `json:"eventVersion"`
}

// ponytail: one shard per stream; real streams split shards over time, add
// child shards (ParentShardId) if a client needs shard lineage.
// ponytail: records are kept for the stream's lifetime, not 24 hours, so
// TrimmedDataAccessException only covers positions before the shard start.
type stream struct {
	arn       string
	label     string
	tableName string
	viewType  string
	keySchema []KeySchemaElement
	hashKey   string
	rangeKey  string
	status    string
	created   time.Time
	shardID   string
	startSeq  uint64
	endSeq    uint64 // 0 while the shard is open
	records   []StreamRecord
}

type shardIterator struct {
	StreamArn string `json:"a"`
	ShardID   string `json:"s"`
	Position  int    `json:"p"`
	Issued    int64  `json:"t"`
}

// ValidateStreamSpecification checks a StreamSpecification request member.
func ValidateStreamSpecification(spec *StreamSpecification) error {
	if spec == nil || !spec.StreamEnabled {
		return nil
	}
	for _, v := range streamViewTypes {
		if spec.StreamViewType == v {
			return nil
		}
	}
	return fmt.Errorf("%w: StreamViewType must be one of %s", ErrValidation, strings.Join(streamViewTypes, ", "))
}

// applyStreamSpecLocked enables or disables the table's stream. creating is
// true for CreateTable, where StreamEnabled=false is a no-op rather than an
// error. It checks before it mutates, so an error leaves the table as it was.
func (s *Service) applyStreamSpecLocked(table *Table, spec *StreamSpecification, creating bool) error {
	if spec == nil {
		return nil
	}
	current := table.stream
	switch {
	case spec.StreamEnabled && current != nil:
		return fmt.Errorf("%w: Table already has an enabled stream", ErrValidation)
	case spec.StreamEnabled:
		s.openStreamLocked(table, spec.StreamViewType)
	case current == nil && !creating:
		return fmt.Errorf("%w: Table does not have an enabled stream", ErrValidation)
	case current != nil:
		s.closeStreamLocked(table)
	}
	return nil
}

func (s *Service) openStreamLocked(table *Table, viewType string) {
	now := s.now().UTC()
	label := now.Format("2006-01-02T15:04:05.000")
	arn := table.description.TableArn + "/stream/" + label
	for s.streams[arn] != nil { // labels are millisecond timestamps; keep them unique
		now = now.Add(time.Millisecond)
		label = now.Format("2006-01-02T15:04:05.000")
		arn = table.description.TableArn + "/stream/" + label
	}
	s.streamSeq++
	st := &stream{
		arn:       arn,
		label:     label,
		tableName: table.description.TableName,
		viewType:  viewType,
		keySchema: cloneKeySchema(table.description.KeySchema),
		hashKey:   table.hashKey,
		rangeKey:  table.rangeKey,
		status:    streamEnabled,
		created:   now,
		shardID:   fmt.Sprintf("shardId-%020d-%08x", now.UnixMilli(), s.streamSeq),
		startSeq:  s.streamSeq,
	}
	s.streams[arn] = st
	table.stream = st
	table.description.StreamSpecification = &StreamSpecification{StreamEnabled: true, StreamViewType: viewType}
	table.description.LatestStreamArn = arn
	table.description.LatestStreamLabel = label
}

func (s *Service) closeStreamLocked(table *Table) {
	st := table.stream
	if st == nil {
		return
	}
	st.status = streamDisabled
	st.endSeq = st.startSeq
	if n := len(st.records); n > 0 {
		st.endSeq, _ = strconv.ParseUint(st.records[n-1].Dynamodb.SequenceNumber, 10, 64)
	}
	table.stream = nil
	table.description.StreamSpecification = nil // LatestStreamArn/Label stay
}

// writeItemLocked stores item under key and records the change on the
// table's stream. A write that changes nothing emits no record.
func (s *Service) writeItemLocked(table *Table, key string, item map[string]any) {
	old := table.items[key]
	stored := cloneItem(item) // normalize first so the comparison sees what is stored
	table.items[key] = stored
	if old == nil {
		s.recordLocked(table, "INSERT", nil, stored)
	} else if !sameItem(old, stored) {
		s.recordLocked(table, "MODIFY", old, stored)
	}
}

// removeItemLocked deletes key and records a REMOVE when an item existed.
func (s *Service) removeItemLocked(table *Table, key string) {
	old, ok := table.items[key]
	if !ok {
		return
	}
	delete(table.items, key)
	s.recordLocked(table, "REMOVE", old, nil)
}

func (s *Service) recordLocked(table *Table, eventName string, oldImage, newImage map[string]any) {
	st := table.stream
	if st == nil {
		return
	}
	source := newImage
	if source == nil {
		source = oldImage
	}
	keys := map[string]any{st.hashKey: cloneAny(source[st.hashKey])}
	if st.rangeKey != "" {
		keys[st.rangeKey] = cloneAny(source[st.rangeKey])
	}
	data := StreamRecordData{
		ApproximateCreationDateTime: float64(s.now().Unix()),
		Keys:                        keys,
		StreamViewType:              st.viewType,
	}
	if newImage != nil && (st.viewType == "NEW_IMAGE" || st.viewType == "NEW_AND_OLD_IMAGES") {
		data.NewImage = cloneItem(newImage)
	}
	if oldImage != nil && (st.viewType == "OLD_IMAGE" || st.viewType == "NEW_AND_OLD_IMAGES") {
		data.OldImage = cloneItem(oldImage)
	}
	if encoded, err := json.Marshal([]any{data.Keys, data.NewImage, data.OldImage}); err == nil {
		data.SizeBytes = int64(len(encoded))
	}
	s.streamSeq++
	data.SequenceNumber = formatSequence(s.streamSeq)
	st.records = append(st.records, StreamRecord{
		AwsRegion:    defaultRegion,
		Dynamodb:     data,
		EventID:      fmt.Sprintf("%032x", s.streamSeq),
		EventName:    eventName,
		EventSource:  "aws:dynamodb",
		EventVersion: "1.1",
	})
}

func formatSequence(n uint64) string {
	return fmt.Sprintf("%021d", n)
}

// ListStreams lists streams in ARN order, optionally for one table.
func (s *Service) ListStreams(tableName string, limit int, exclusiveStartStreamArn string) ([]StreamSummary, string, error) {
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return nil, "", fmt.Errorf("%w: Limit must be between 1 and 100", ErrValidation)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tableName = strings.TrimSpace(tableName)
	arns := make([]string, 0, len(s.streams))
	for arn, st := range s.streams {
		if tableName == "" || st.tableName == tableName {
			arns = append(arns, arn)
		}
	}
	// A deleted table's streams stay listed; only an unknown name with no
	// streams is an error.
	if _, exists := s.tables[tableName]; tableName != "" && !exists && len(arns) == 0 {
		return nil, "", fmt.Errorf("%w: Requested resource not found: Table: %s not found", ErrResourceNotFound, tableName)
	}
	sort.Strings(arns)
	start := 0
	if exclusiveStartStreamArn != "" {
		start = sort.SearchStrings(arns, exclusiveStartStreamArn)
		if start < len(arns) && arns[start] == exclusiveStartStreamArn {
			start++
		}
	}
	end := min(start+limit, len(arns))
	out := make([]StreamSummary, 0, end-start)
	for _, arn := range arns[start:end] {
		st := s.streams[arn]
		out = append(out, StreamSummary{StreamArn: st.arn, StreamLabel: st.label, TableName: st.tableName})
	}
	last := ""
	if end < len(arns) {
		last = arns[end-1]
	}
	return out, last, nil
}

// DescribeStream returns a stream and its single shard.
func (s *Service) DescribeStream(streamArn string, limit int, exclusiveStartShardID string) (StreamDescription, error) {
	if limit < 0 || limit > 100 {
		return StreamDescription{}, fmt.Errorf("%w: Limit must be between 1 and 100", ErrValidation)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.requireStream(streamArn)
	if err != nil {
		return StreamDescription{}, err
	}
	shards := []Shard{}
	if exclusiveStartShardID != st.shardID {
		shard := Shard{ShardID: st.shardID, SequenceNumberRange: SequenceNumberRange{StartingSequenceNumber: formatSequence(st.startSeq)}}
		if st.endSeq != 0 {
			shard.SequenceNumberRange.EndingSequenceNumber = formatSequence(st.endSeq)
		}
		shards = append(shards, shard)
	}
	return StreamDescription{
		CreationRequestDateTime: float64(st.created.Unix()),
		KeySchema:               cloneKeySchema(st.keySchema),
		Shards:                  shards,
		StreamArn:               st.arn,
		StreamLabel:             st.label,
		StreamStatus:            st.status,
		StreamViewType:          st.viewType,
		TableName:               st.tableName,
	}, nil
}

// GetShardIterator positions an iterator in a stream's shard.
func (s *Service) GetShardIterator(streamArn, shardID, iteratorType, sequenceNumber string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.requireStream(streamArn)
	if err != nil {
		return "", err
	}
	if shardID != st.shardID {
		return "", fmt.Errorf("%w: shard %s does not exist in stream", ErrResourceNotFound, shardID)
	}
	position := 0
	switch iteratorType {
	case "TRIM_HORIZON":
	case "LATEST":
		position = len(st.records)
	case "AT_SEQUENCE_NUMBER", "AFTER_SEQUENCE_NUMBER":
		if sequenceNumber == "" {
			return "", fmt.Errorf("%w: SequenceNumber is required for %s", ErrValidation, iteratorType)
		}
		position, err = st.positionOf(sequenceNumber, iteratorType == "AFTER_SEQUENCE_NUMBER")
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("%w: ShardIteratorType must be TRIM_HORIZON, LATEST, AT_SEQUENCE_NUMBER or AFTER_SEQUENCE_NUMBER", ErrValidation)
	}
	if iteratorType != "AT_SEQUENCE_NUMBER" && iteratorType != "AFTER_SEQUENCE_NUMBER" && sequenceNumber != "" {
		return "", fmt.Errorf("%w: SequenceNumber is only valid with AT_SEQUENCE_NUMBER or AFTER_SEQUENCE_NUMBER", ErrValidation)
	}
	return s.encodeIterator(st, position), nil
}

// positionOf finds the record index for a sequence number in the shard. The
// shard's StartingSequenceNumber carries no record, so it maps to the start.
func (st *stream) positionOf(sequenceNumber string, after bool) (int, error) {
	n, err := strconv.ParseUint(sequenceNumber, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid SequenceNumber", ErrValidation)
	}
	if n < st.startSeq {
		return 0, fmt.Errorf("%w: sequence number %s is before the start of the shard", ErrTrimmedDataAccess, sequenceNumber)
	}
	if n == st.startSeq {
		return 0, nil
	}
	for i, rec := range st.records {
		if rec.Dynamodb.SequenceNumber == formatSequence(n) {
			if after {
				return i + 1, nil
			}
			return i, nil
		}
	}
	return 0, fmt.Errorf("%w: sequence number %s is not in the shard", ErrValidation, sequenceNumber)
}

func (s *Service) encodeIterator(st *stream, position int) string {
	raw, _ := json.Marshal(shardIterator{StreamArn: st.arn, ShardID: st.shardID, Position: position, Issued: s.now().UnixNano()})
	return st.arn + "|" + base64.RawURLEncoding.EncodeToString(raw)
}

// GetRecords reads up to limit records from the iterator's position. The next
// iterator is empty once a closed shard has been read to its end.
func (s *Service) GetRecords(iterator string, limit int) ([]StreamRecord, string, error) {
	if limit == 0 {
		limit = 1000
	}
	if limit < 1 || limit > 1000 {
		return nil, "", fmt.Errorf("%w: Limit must be between 1 and 1000", ErrValidation)
	}
	_, encoded, ok := strings.Cut(iterator, "|")
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	var it shardIterator
	if !ok || err != nil || json.Unmarshal(raw, &it) != nil {
		return nil, "", fmt.Errorf("%w: invalid ShardIterator", ErrValidation)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now().Sub(time.Unix(0, it.Issued)) > streamIteratorTTL {
		return nil, "", fmt.Errorf("%w: Iterator expired. The iterator was created more than 15 minutes ago", ErrExpiredIterator)
	}
	st, err := s.requireStream(it.StreamArn)
	if err != nil {
		return nil, "", err
	}
	if it.ShardID != st.shardID || it.Position < 0 || it.Position > len(st.records) {
		return nil, "", fmt.Errorf("%w: invalid ShardIterator", ErrValidation)
	}
	end := min(it.Position+limit, len(st.records))
	out := make([]StreamRecord, 0, end-it.Position)
	for _, rec := range st.records[it.Position:end] {
		rec.Dynamodb.Keys = cloneItem(rec.Dynamodb.Keys)
		rec.Dynamodb.NewImage = cloneItem(rec.Dynamodb.NewImage)
		rec.Dynamodb.OldImage = cloneItem(rec.Dynamodb.OldImage)
		out = append(out, rec)
	}
	if st.endSeq != 0 && end == len(st.records) {
		return out, "", nil
	}
	return out, s.encodeIterator(st, end), nil
}

func (s *Service) requireStream(streamArn string) (*stream, error) {
	st := s.streams[strings.TrimSpace(streamArn)]
	if st == nil {
		return nil, fmt.Errorf("%w: Requested resource not found: Stream: %s not found", ErrResourceNotFound, streamArn)
	}
	return st, nil
}
