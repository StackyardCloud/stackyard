package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type fsxStore struct {
	mu     sync.Mutex
	nextID int64
	tags   map[string]map[string]string
}

func newFSxStore() *fsxStore {
	seedARN := fsxFileSystemARN("fs-00000000000000000")
	return &fsxStore{
		nextID: 1,
		tags: map[string]map[string]string{
			seedARN: {"stackyard": "true"},
		},
	}
}

func (s *fsxStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "DescribeFileSystems":
		return map[string]any{
			"FileSystems": []any{
				map[string]any{
					"FileSystemId":    "fs-00000000000000000",
					"FileSystemType":  "LUSTRE",
					"Lifecycle":       "AVAILABLE",
					"StorageCapacity": 1200,
					"CreationTime":    now,
					"ResourceARN":     fsxFileSystemARN("fs-00000000000000000"),
				},
			},
		}
	case "DescribeBackups":
		return map[string]any{
			"Backups": []any{
				map[string]any{
					"BackupId":     "backup-00000000000000000",
					"Lifecycle":    "AVAILABLE",
					"Type":         "USER_INITIATED",
					"CreationTime": now,
					"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:backup/backup-00000000000000000",
					"FileSystem":   map[string]any{"FileSystemId": "fs-00000000000000000"},
				},
			},
		}
	case "DescribeVolumes":
		return map[string]any{
			"Volumes": []any{
				map[string]any{
					"VolumeId":     "fsvol-00000000000000000",
					"Lifecycle":    "AVAILABLE",
					"CreationTime": now,
					"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:volume/fsvol-00000000000000000",
					"FileSystemId": "fs-00000000000000000",
				},
			},
		}
	case "DescribeStorageVirtualMachines":
		return map[string]any{
			"StorageVirtualMachines": []any{
				map[string]any{
					"StorageVirtualMachineId": "svm-00000000000000000",
					"Lifecycle":               "CREATED",
					"Name":                    "stackyard-svm",
					"ResourceARN":             "arn:aws:fsx:us-east-1:123456789012:storage-virtual-machine/svm-00000000000000000",
					"FileSystemId":            "fs-00000000000000000",
				},
			},
		}
	case "CreateFileSystem", "CreateFileSystemFromBackup":
		fileSystemID := fmt.Sprintf("fs-%017d", s.nextID)
		s.nextID++
		arn := fsxFileSystemARN(fileSystemID)
		s.tags[arn] = map[string]string{"stackyard": "true"}
		return map[string]any{
			"FileSystem": map[string]any{
				"FileSystemId":    fileSystemID,
				"FileSystemType":  fsxPayloadString(payload, "FileSystemType", "LUSTRE"),
				"Lifecycle":       "CREATING",
				"CreationTime":    now,
				"StorageCapacity": fsxPayloadNumber(payload, "StorageCapacity", 1200),
				"ResourceARN":     arn,
			},
		}
	case "CreateBackup":
		backupID := fmt.Sprintf("backup-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"Backup": map[string]any{
				"BackupId":     backupID,
				"Type":         "USER_INITIATED",
				"Lifecycle":    "CREATING",
				"CreationTime": now,
				"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:backup/" + backupID,
			},
		}
	case "CreateSnapshot":
		snapshotID := fmt.Sprintf("snapshot-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"Snapshot": map[string]any{
				"SnapshotId":   snapshotID,
				"Lifecycle":    "PENDING",
				"CreationTime": now,
				"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:snapshot/" + snapshotID,
			},
		}
	case "CreateStorageVirtualMachine":
		id := fmt.Sprintf("svm-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"StorageVirtualMachine": map[string]any{
				"StorageVirtualMachineId": id,
				"Lifecycle":               "CREATED",
				"Name":                    fsxPayloadString(payload, "Name", "stackyard-svm"),
				"ResourceARN":             "arn:aws:fsx:us-east-1:123456789012:storage-virtual-machine/" + id,
			},
		}
	case "CreateVolume", "CreateVolumeFromBackup":
		id := fmt.Sprintf("fsvol-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"Volume": map[string]any{
				"VolumeId":     id,
				"Lifecycle":    "CREATING",
				"CreationTime": now,
				"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:volume/" + id,
			},
		}
	case "CreateFileCache":
		id := fmt.Sprintf("fc-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"FileCache": map[string]any{
				"FileCacheId":     id,
				"Lifecycle":       "CREATING",
				"StorageCapacity": fsxPayloadNumber(payload, "StorageCapacity", 1200),
				"ResourceARN":     "arn:aws:fsx:us-east-1:123456789012:file-cache/" + id,
			},
		}
	case "ListTagsForResource":
		arn := fsxPayloadString(payload, "ResourceARN", fsxPayloadString(payload, "ResourceArn", fsxFileSystemARN("fs-00000000000000000")))
		return map[string]any{"Tags": fsxTagsList(s.tags[arn])}
	case "TagResource":
		arn := fsxPayloadString(payload, "ResourceARN", fsxPayloadString(payload, "ResourceArn", fsxFileSystemARN("fs-00000000000000000")))
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for k, v := range fsxTagsFromAny(payload["Tags"]) {
			s.tags[arn][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		arn := fsxPayloadString(payload, "ResourceARN", fsxPayloadString(payload, "ResourceArn", fsxFileSystemARN("fs-00000000000000000")))
		for _, key := range fsxPayloadStringSlice(payload, "TagKeys") {
			delete(s.tags[arn], key)
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Describe") {
		key := strings.TrimPrefix(action, "Describe")
		if strings.HasSuffix(key, "s") {
			return map[string]any{key: []any{}}
		}
		return map[string]any{key: map[string]any{}}
	}

	if strings.HasPrefix(action, "Create") {
		key := strings.TrimPrefix(action, "Create")
		if key == "" {
			key = "Result"
		}
		return map[string]any{key: map[string]any{}}
	}

	if strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Delete") ||
		strings.HasPrefix(action, "Associate") ||
		strings.HasPrefix(action, "Disassociate") ||
		strings.HasPrefix(action, "Copy") ||
		strings.HasPrefix(action, "Detach") ||
		strings.HasPrefix(action, "Restore") ||
		strings.HasPrefix(action, "Release") ||
		strings.HasPrefix(action, "Cancel") ||
		strings.HasPrefix(action, "Start") {
		return map[string]any{}
	}

	return map[string]any{}
}

func fsxFileSystemARN(id string) string {
	clean := strings.TrimSpace(id)
	if clean == "" {
		clean = "fs-00000000000000000"
	}
	return "arn:aws:fsx:us-east-1:123456789012:file-system/" + clean
}

func fsxPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "%!v(<nil>)" {
			return s
		}
	}
	return fallback
}

func fsxPayloadNumber(payload map[string]any, key string, fallback int64) int64 {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch n := v.(type) {
		case int:
			return int64(n)
		case int32:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		}
	}
	return fallback
}

func fsxPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		items, ok := v.([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" && s != "%!v(<nil>)" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func fsxTagsFromAny(raw any) map[string]string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := fsxPayloadString(m, "Key", "")
		if strings.TrimSpace(key) == "" {
			continue
		}
		out[key] = fsxPayloadString(m, "Value", "")
	}
	return out
}

func fsxTagsList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}
