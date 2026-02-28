package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ebsDefaultBlockSize = 512 * 1024

type ebsStoreResponse struct {
	Status       int
	JSON         map[string]any
	Body         []byte
	Headers      map[string]string
	ErrorCode    string
	ErrorMessage string
	ErrorReason  string
}

type ebsStore struct {
	mu sync.Mutex

	nextSnapshot uint64
	snapshots    map[string]*ebsSnapshot
}

type ebsSnapshot struct {
	ID              string
	Description     string
	OwnerID         string
	Status          string
	StartTime       time.Time
	VolumeSize      int64
	BlockSize       int64
	ParentSnapshot  string
	KmsKeyArn       string
	Tags            []map[string]string
	Blocks          map[int64]*ebsBlock
	CompletedBlocks int64
}

type ebsBlock struct {
	Index             int64
	Token             string
	Data              []byte
	Checksum          string
	ChecksumAlgorithm string
}

func newEBSStore() *ebsStore {
	s := &ebsStore{
		nextSnapshot: 2,
		snapshots:    map[string]*ebsSnapshot{},
	}

	seedID := "snap-00000000000000001"
	snapshot := &ebsSnapshot{
		ID:          seedID,
		Description: "stackyard seeded snapshot",
		OwnerID:     "123456789012",
		Status:      "pending",
		StartTime:   time.Now().UTC(),
		VolumeSize:  1,
		BlockSize:   ebsDefaultBlockSize,
		Blocks:      map[int64]*ebsBlock{},
		Tags: []map[string]string{
			{"Key": "seed", "Value": "true"},
		},
	}
	snapshot.Blocks[0] = s.newBlockLocked(seedID, 0, []byte("stackyard-ebs-seed-block"))
	s.snapshots[seedID] = snapshot
	return s
}

func (s *ebsStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	query url.Values,
	headers http.Header,
	body []byte,
) ebsStoreResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID := ebsFirstNonEmpty(pathParams["snapshotId"], ebsStringAny(payload, "SnapshotId"), "snap-00000000000000001")
	secondSnapshotID := ebsFirstNonEmpty(pathParams["secondSnapshotId"], ebsStringAny(payload, "SecondSnapshotId"), snapshotID)
	firstSnapshotID := ebsFirstNonEmpty(query.Get("firstSnapshotId"), ebsStringAny(payload, "FirstSnapshotId"))
	blockIndex := ebsFirstInt64(pathParams["blockIndex"], ebsInt64Any(payload, "BlockIndex"), 0)

	switch action {
	case "StartSnapshot":
		volumeSize := ebsInt64Any(payload, "VolumeSize")
		if volumeSize <= 0 {
			volumeSize = 1
		}

		snapshotID = s.nextSnapshotIDLocked()
		snapshot := &ebsSnapshot{
			ID:             snapshotID,
			Description:    ebsStringAny(payload, "Description"),
			OwnerID:        "123456789012",
			Status:         "pending",
			StartTime:      time.Now().UTC(),
			VolumeSize:     volumeSize,
			BlockSize:      ebsDefaultBlockSize,
			ParentSnapshot: ebsStringAny(payload, "ParentSnapshotId"),
			KmsKeyArn:      ebsStringAny(payload, "KmsKeyArn"),
			Tags:           ebsTagsFromPayload(payload),
			Blocks:         map[int64]*ebsBlock{},
		}
		snapshot.Blocks[0] = s.newBlockLocked(snapshotID, 0, []byte("stackyard-ebs-seed-block"))
		s.snapshots[snapshotID] = snapshot

		return ebsStoreResponse{
			Status: http.StatusCreated,
			JSON: map[string]any{
				"Description":      snapshot.Description,
				"SnapshotId":       snapshot.ID,
				"OwnerId":          snapshot.OwnerID,
				"Status":           snapshot.Status,
				"StartTime":        snapshot.StartTime.UTC().Format(time.RFC3339),
				"VolumeSize":       snapshot.VolumeSize,
				"BlockSize":        snapshot.BlockSize,
				"Tags":             ebsTagsToAny(snapshot.Tags),
				"ParentSnapshotId": snapshot.ParentSnapshot,
				"KmsKeyArn":        snapshot.KmsKeyArn,
				"SseType":          "",
			},
		}

	case "PutSnapshotBlock":
		snapshot := s.ensureSnapshotLocked(snapshotID)
		if len(body) == 0 {
			body = []byte("stackyard-ebs-block")
		}
		algo := strings.ToUpper(strings.TrimSpace(headers.Get("x-amz-checksum-algorithm")))
		if algo == "" {
			algo = strings.ToUpper(strings.TrimSpace(ebsStringAny(payload, "ChecksumAlgorithm")))
		}
		if algo == "" {
			algo = "SHA256"
		}
		block := s.newBlockLocked(snapshot.ID, blockIndex, body)
		block.ChecksumAlgorithm = algo
		snapshot.Blocks[blockIndex] = block

		return ebsStoreResponse{
			Status: http.StatusCreated,
			Headers: map[string]string{
				"x-amz-Checksum":           block.Checksum,
				"x-amz-Checksum-Algorithm": block.ChecksumAlgorithm,
			},
		}

	case "GetSnapshotBlock":
		snapshot := s.ensureSnapshotLocked(snapshotID)
		block := snapshot.Blocks[blockIndex]
		if block == nil {
			block = s.newBlockLocked(snapshot.ID, blockIndex, []byte("stackyard-ebs-block"))
			snapshot.Blocks[blockIndex] = block
		}
		if len(block.Data) == 0 {
			block.Data = []byte("stackyard-ebs-block")
			block.Checksum = ebsBase64SHA256(block.Data)
			if block.ChecksumAlgorithm == "" {
				block.ChecksumAlgorithm = "SHA256"
			}
		}

		return ebsStoreResponse{
			Status: http.StatusOK,
			Body:   append([]byte(nil), block.Data...),
			Headers: map[string]string{
				"x-amz-Data-Length":        strconv.Itoa(len(block.Data)),
				"x-amz-Checksum":           block.Checksum,
				"x-amz-Checksum-Algorithm": ebsFirstNonEmpty(block.ChecksumAlgorithm, "SHA256"),
			},
		}

	case "ListSnapshotBlocks":
		snapshot := s.ensureSnapshotLocked(snapshotID)
		return s.listSnapshotBlocksLocked(snapshot, query)

	case "ListChangedBlocks":
		second := s.ensureSnapshotLocked(secondSnapshotID)
		var first *ebsSnapshot
		if strings.TrimSpace(firstSnapshotID) != "" {
			first = s.ensureSnapshotLocked(firstSnapshotID)
		}
		return s.listChangedBlocksLocked(first, second, query)

	case "CompleteSnapshot":
		snapshot := s.ensureSnapshotLocked(snapshotID)
		changedCount := ebsInt64FromString(headers.Get("x-amz-ChangedBlocksCount"), 0)
		if changedCount <= 0 {
			changedCount = ebsInt64Any(payload, "ChangedBlocksCount")
		}
		if changedCount <= 0 {
			changedCount = int64(len(snapshot.Blocks))
		}
		snapshot.CompletedBlocks = changedCount
		snapshot.Status = "completed"

		return ebsStoreResponse{
			Status: http.StatusAccepted,
			JSON: map[string]any{
				"Status": snapshot.Status,
			},
		}

	default:
		return ebsStoreResponse{
			Status:       http.StatusBadRequest,
			ErrorCode:    "ValidationException",
			ErrorMessage: "unknown action",
		}
	}
}

func (s *ebsStore) listSnapshotBlocksLocked(snapshot *ebsSnapshot, query url.Values) ebsStoreResponse {
	indices := make([]int64, 0, len(snapshot.Blocks))
	for idx := range snapshot.Blocks {
		indices = append(indices, idx)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

	startingBlock := ebsInt64FromString(query.Get("startingBlockIndex"), 0)
	maxResults := int(ebsInt64FromString(query.Get("maxResults"), 1000))
	if maxResults <= 0 {
		maxResults = 1000
	}

	filtered := make([]int64, 0, len(indices))
	for _, idx := range indices {
		if idx >= startingBlock {
			filtered = append(filtered, idx)
		}
	}

	offset := int(ebsInt64FromString(query.Get("pageToken"), 0))
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + maxResults
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[offset:end]
	nextToken := ""
	if end < len(filtered) {
		nextToken = strconv.Itoa(end)
	}

	blocks := make([]any, 0, len(page))
	for _, idx := range page {
		block := snapshot.Blocks[idx]
		blocks = append(blocks, map[string]any{
			"BlockIndex": idx,
			"BlockToken": block.Token,
		})
	}

	return ebsStoreResponse{
		Status: http.StatusOK,
		JSON: map[string]any{
			"Blocks":     blocks,
			"ExpiryTime": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
			"VolumeSize": snapshot.VolumeSize,
			"BlockSize":  snapshot.BlockSize,
			"NextToken":  nextToken,
		},
	}
}

func (s *ebsStore) listChangedBlocksLocked(first, second *ebsSnapshot, query url.Values) ebsStoreResponse {
	indices := make([]int64, 0, len(second.Blocks))
	for idx := range second.Blocks {
		indices = append(indices, idx)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

	startingBlock := ebsInt64FromString(query.Get("startingBlockIndex"), 0)
	maxResults := int(ebsInt64FromString(query.Get("maxResults"), 1000))
	if maxResults <= 0 {
		maxResults = 1000
	}

	filtered := make([]int64, 0, len(indices))
	for _, idx := range indices {
		if idx >= startingBlock {
			filtered = append(filtered, idx)
		}
	}

	offset := int(ebsInt64FromString(query.Get("pageToken"), 0))
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + maxResults
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[offset:end]
	nextToken := ""
	if end < len(filtered) {
		nextToken = strconv.Itoa(end)
	}

	changed := make([]any, 0, len(page))
	for _, idx := range page {
		secondBlock := second.Blocks[idx]
		entry := map[string]any{
			"BlockIndex":       idx,
			"SecondBlockToken": secondBlock.Token,
		}
		if first != nil {
			if firstBlock := first.Blocks[idx]; firstBlock != nil {
				entry["FirstBlockToken"] = firstBlock.Token
			}
		}
		changed = append(changed, entry)
	}

	return ebsStoreResponse{
		Status: http.StatusOK,
		JSON: map[string]any{
			"ChangedBlocks": changed,
			"ExpiryTime":    time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
			"VolumeSize":    second.VolumeSize,
			"BlockSize":     second.BlockSize,
			"NextToken":     nextToken,
		},
	}
}

func (s *ebsStore) ensureSnapshotLocked(snapshotID string) *ebsSnapshot {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		snapshotID = "snap-00000000000000001"
	}
	if snapshot, ok := s.snapshots[snapshotID]; ok {
		return snapshot
	}

	snapshot := &ebsSnapshot{
		ID:          snapshotID,
		Description: "stackyard synthetic snapshot",
		OwnerID:     "123456789012",
		Status:      "pending",
		StartTime:   time.Now().UTC(),
		VolumeSize:  1,
		BlockSize:   ebsDefaultBlockSize,
		Blocks:      map[int64]*ebsBlock{},
		Tags: []map[string]string{
			{"Key": "managed-by", "Value": "stackyard"},
		},
	}
	snapshot.Blocks[0] = s.newBlockLocked(snapshotID, 0, []byte("stackyard-ebs-seed-block"))
	s.snapshots[snapshotID] = snapshot
	return snapshot
}

func (s *ebsStore) newBlockLocked(snapshotID string, index int64, data []byte) *ebsBlock {
	if len(data) == 0 {
		data = []byte("stackyard-ebs-block")
	}
	copyData := append([]byte(nil), data...)
	return &ebsBlock{
		Index:             index,
		Token:             fmt.Sprintf("token-%s-%d-%d", snapshotID, index, time.Now().UnixNano()),
		Data:              copyData,
		Checksum:          ebsBase64SHA256(copyData),
		ChecksumAlgorithm: "SHA256",
	}
}

func (s *ebsStore) nextSnapshotIDLocked() string {
	id := fmt.Sprintf("snap-%017x", s.nextSnapshot)
	s.nextSnapshot++
	return id
}

func ebsBase64SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func ebsStringAny(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	for k, v := range m {
		if !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		default:
			return strings.TrimSpace(fmt.Sprintf("%v", t))
		}
	}
	return ""
}

func ebsInt64Any(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	for k, v := range m {
		if !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		switch t := v.(type) {
		case int:
			return int64(t)
		case int64:
			return t
		case float64:
			return int64(t)
		case json.Number:
			n, _ := strconv.ParseInt(string(t), 10, 64)
			return n
		case string:
			n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
			return n
		}
	}
	return 0
}

func ebsInt64FromString(raw string, fallback int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func ebsFirstInt64(pathValue string, payloadValue int64, fallback int64) int64 {
	if n := ebsInt64FromString(pathValue, fallback); strings.TrimSpace(pathValue) != "" {
		return n
	}
	if payloadValue != 0 {
		return payloadValue
	}
	return fallback
}

func ebsFirstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func ebsTagsFromPayload(payload map[string]any) []map[string]string {
	if payload == nil {
		return nil
	}
	var raw any
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), "Tags") {
			raw = v
			break
		}
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := ebsStringAny(m, "Key")
		if key == "" {
			continue
		}
		out = append(out, map[string]string{
			"Key":   key,
			"Value": ebsStringAny(m, "Value"),
		})
	}
	return out
}

func ebsTagsToAny(tags []map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]any{
			"Key":   strings.TrimSpace(tag["Key"]),
			"Value": strings.TrimSpace(tag["Value"]),
		})
	}
	return out
}
