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
	fileSystemID := fsxPayloadString(payload, "FileSystemId", "fs-00000000000000000")
	backupID := fsxPayloadString(payload, "BackupId", "backup-00000000000000000")
	fileCacheID := fsxPayloadString(payload, "FileCacheId", "fc-00000000000000000")
	snapshotID := fsxPayloadString(payload, "SnapshotId", "snapshot-00000000000000000")
	volumeID := fsxPayloadString(payload, "VolumeId", "fsvol-00000000000000000")
	svmID := fsxPayloadString(payload, "StorageVirtualMachineId", "svm-00000000000000000")
	associationID := fsxPayloadString(payload, "AssociationId", "dra-00000000000000000")
	taskID := fsxPayloadString(payload, "TaskId", "drt-00000000000000000")

	switch action {
	case "AssociateFileSystemAliases":
		return map[string]any{"Aliases": fsxAliasesFromPayload(payload, "CREATING")}
	case "DescribeFileSystems":
		return map[string]any{
			"FileSystems": []any{
				fsxDefaultFileSystem("fs-00000000000000000", now, "AVAILABLE"),
			},
		}
	case "DescribeBackups":
		return map[string]any{
			"Backups": []any{
				fsxDefaultBackup("backup-00000000000000000", "fs-00000000000000000", now, "AVAILABLE"),
			},
		}
	case "DescribeDataRepositoryAssociations":
		return map[string]any{
			"Associations": []any{
				fsxDefaultDataRepositoryAssociation(associationID, fileSystemID, fileCacheID, now, "AVAILABLE"),
			},
		}
	case "DescribeFileSystemAliases":
		return map[string]any{"Aliases": fsxAliasesFromPayload(payload, "AVAILABLE")}
	case "DescribeSharedVpcConfiguration":
		return map[string]any{
			"EnableFsxRouteTableUpdatesFromParticipantAccounts": fsxPayloadBoolString(
				payload,
				"EnableFsxRouteTableUpdatesFromParticipantAccounts",
				true,
			),
		}
	case "DescribeVolumes":
		return map[string]any{
			"Volumes": []any{
				fsxDefaultVolume("fsvol-00000000000000000", "fs-00000000000000000", now, "AVAILABLE"),
			},
		}
	case "DescribeStorageVirtualMachines":
		return map[string]any{
			"StorageVirtualMachines": []any{
				fsxDefaultStorageVirtualMachine("svm-00000000000000000", "fs-00000000000000000", now, "CREATED"),
			},
		}
	case "CreateAndAttachS3AccessPoint":
		return map[string]any{
			"S3AccessPointAttachment": fsxDefaultS3AccessPointAttachment(payload, volumeID, now),
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
			"Backup": fsxDefaultBackup(backupID, fileSystemID, now, "CREATING"),
		}
	case "CreateDataRepositoryAssociation":
		return map[string]any{
			"Association": fsxDefaultDataRepositoryAssociation(associationID, fileSystemID, fileCacheID, now, "CREATING"),
		}
	case "CreateDataRepositoryTask":
		return map[string]any{
			"DataRepositoryTask": fsxDefaultDataRepositoryTask(taskID, fileSystemID, fileCacheID, now, "EXECUTING"),
		}
	case "CreateSnapshot":
		snapshotID := fmt.Sprintf("snapshot-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"Snapshot": fsxDefaultSnapshot(snapshotID, volumeID, now, "PENDING"),
		}
	case "CreateStorageVirtualMachine":
		id := fmt.Sprintf("svm-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"StorageVirtualMachine": fsxDefaultStorageVirtualMachine(id, fileSystemID, now, "CREATED"),
		}
	case "CreateVolume", "CreateVolumeFromBackup":
		id := fmt.Sprintf("fsvol-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"Volume": fsxDefaultVolume(id, fileSystemID, now, "CREATING"),
		}
	case "CreateFileCache":
		id := fmt.Sprintf("fc-%017d", s.nextID)
		s.nextID++
		return map[string]any{
			"FileCache": fsxDefaultFileCache(id, now, "CREATING"),
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
	case "StartMisconfiguredStateRecovery":
		return map[string]any{"FileSystem": fsxDefaultFileSystem(fileSystemID, now, "UPDATING")}
	case "RestoreVolumeFromSnapshot":
		return map[string]any{
			"VolumeId":              volumeID,
			"Lifecycle":             "CREATING",
			"AdministrativeActions": []any{},
		}
	case "UpdateDataRepositoryAssociation":
		return map[string]any{
			"Association": fsxDefaultDataRepositoryAssociation(associationID, fileSystemID, fileCacheID, now, "UPDATING"),
		}
	case "UpdateFileCache":
		return map[string]any{"FileCache": fsxDefaultFileCache(fileCacheID, now, "UPDATING")}
	case "UpdateFileSystem":
		return map[string]any{"FileSystem": fsxDefaultFileSystem(fileSystemID, now, "UPDATING")}
	case "UpdateSharedVpcConfiguration":
		return map[string]any{
			"EnableFsxRouteTableUpdatesFromParticipantAccounts": fsxPayloadBoolString(
				payload,
				"EnableFsxRouteTableUpdatesFromParticipantAccounts",
				true,
			),
		}
	case "UpdateSnapshot":
		return map[string]any{"Snapshot": fsxDefaultSnapshot(snapshotID, volumeID, now, "AVAILABLE")}
	case "UpdateStorageVirtualMachine":
		return map[string]any{
			"StorageVirtualMachine": fsxDefaultStorageVirtualMachine(svmID, fileSystemID, now, "CREATED"),
		}
	case "UpdateVolume":
		return map[string]any{"Volume": fsxDefaultVolume(volumeID, fileSystemID, now, "UPDATING")}
	case "DisassociateFileSystemAliases":
		return map[string]any{"Aliases": fsxAliasesFromPayload(payload, "DELETING")}
	case "CancelDataRepositoryTask":
		return map[string]any{"TaskId": taskID, "Lifecycle": "CANCELING"}
	case "DeleteBackup":
		return map[string]any{"BackupId": backupID, "Lifecycle": "DELETED"}
	case "DeleteDataRepositoryAssociation":
		return map[string]any{
			"AssociationId":          associationID,
			"Lifecycle":              "DELETING",
			"DeleteDataInFileSystem": fsxPayloadBool(payload, "DeleteDataInFileSystem", false),
		}
	case "DeleteFileCache":
		return map[string]any{"FileCacheId": fileCacheID, "Lifecycle": "DELETING"}
	case "DeleteFileSystem":
		return map[string]any{"FileSystemId": fileSystemID, "Lifecycle": "DELETING"}
	case "DeleteSnapshot":
		return map[string]any{"SnapshotId": snapshotID, "Lifecycle": "DELETING"}
	case "DeleteStorageVirtualMachine":
		return map[string]any{"StorageVirtualMachineId": svmID, "Lifecycle": "DELETING"}
	case "DeleteVolume":
		return map[string]any{"VolumeId": volumeID, "Lifecycle": "DELETING"}
	case "ReleaseFileSystemNfsV3Locks":
		return map[string]any{"FileSystem": fsxDefaultFileSystem(fileSystemID, now, "AVAILABLE")}
	case "CopyBackup":
		return map[string]any{"Backup": fsxDefaultBackup(backupID, fileSystemID, now, "COPYING")}
	case "CopySnapshotAndUpdateVolume":
		return map[string]any{
			"VolumeId":              volumeID,
			"Lifecycle":             "UPDATING",
			"AdministrativeActions": []any{},
		}
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

func fsxPayloadBool(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		b, ok := v.(bool)
		if ok {
			return b
		}
	}
	return fallback
}

func fsxPayloadBoolString(payload map[string]any, key string, fallback bool) string {
	if payload == nil {
		return fmt.Sprintf("%t", fallback)
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch value := v.(type) {
		case bool:
			return fmt.Sprintf("%t", value)
		case string:
			clean := strings.TrimSpace(value)
			if clean != "" {
				return clean
			}
		}
	}
	return fmt.Sprintf("%t", fallback)
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

func fsxDefaultFileSystem(id, now, lifecycle string) map[string]any {
	return map[string]any{
		"FileSystemId":    id,
		"FileSystemType":  "LUSTRE",
		"Lifecycle":       lifecycle,
		"StorageCapacity": 1200,
		"CreationTime":    now,
		"ResourceARN":     fsxFileSystemARN(id),
	}
}

func fsxDefaultBackup(id, fileSystemID, now, lifecycle string) map[string]any {
	return map[string]any{
		"BackupId":     id,
		"Type":         "USER_INITIATED",
		"Lifecycle":    lifecycle,
		"CreationTime": now,
		"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:backup/" + id,
		"FileSystem":   fsxDefaultFileSystem(fileSystemID, now, "AVAILABLE"),
	}
}

func fsxDefaultDataRepositoryAssociation(id, fileSystemID, fileCacheID, now, lifecycle string) map[string]any {
	return map[string]any{
		"AssociationId":      id,
		"ResourceARN":        "arn:aws:fsx:us-east-1:123456789012:data-repository-association/" + id,
		"FileSystemId":       fileSystemID,
		"FileCacheId":        fileCacheID,
		"Lifecycle":          lifecycle,
		"FileSystemPath":     "/ns1/",
		"DataRepositoryPath": "s3://stackyard-bucket/fsx/",
		"CreationTime":       now,
	}
}

func fsxDefaultDataRepositoryTask(id, fileSystemID, fileCacheID, now, lifecycle string) map[string]any {
	return map[string]any{
		"TaskId":       id,
		"Lifecycle":    lifecycle,
		"Type":         "EXPORT_TO_REPOSITORY",
		"CreationTime": now,
		"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:data-repository-task/" + id,
		"FileSystemId": fileSystemID,
		"FileCacheId":  fileCacheID,
		"Paths":        []any{"/"},
	}
}

func fsxDefaultFileCache(id, now, lifecycle string) map[string]any {
	return map[string]any{
		"FileCacheId":     id,
		"Lifecycle":       lifecycle,
		"CreationTime":    now,
		"StorageCapacity": 1200,
		"FileCacheType":   "LUSTRE",
		"ResourceARN":     "arn:aws:fsx:us-east-1:123456789012:file-cache/" + id,
	}
}

func fsxDefaultSnapshot(id, volumeID, now, lifecycle string) map[string]any {
	return map[string]any{
		"SnapshotId":   id,
		"Name":         "stackyard-snapshot",
		"VolumeId":     volumeID,
		"Lifecycle":    lifecycle,
		"CreationTime": now,
		"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:snapshot/" + id,
	}
}

func fsxDefaultStorageVirtualMachine(id, fileSystemID, now, lifecycle string) map[string]any {
	return map[string]any{
		"StorageVirtualMachineId": id,
		"Lifecycle":               lifecycle,
		"Name":                    "stackyard-svm",
		"CreationTime":            now,
		"ResourceARN":             "arn:aws:fsx:us-east-1:123456789012:storage-virtual-machine/" + id,
		"FileSystemId":            fileSystemID,
	}
}

func fsxDefaultVolume(id, fileSystemID, now, lifecycle string) map[string]any {
	return map[string]any{
		"VolumeId":     id,
		"Lifecycle":    lifecycle,
		"CreationTime": now,
		"ResourceARN":  "arn:aws:fsx:us-east-1:123456789012:volume/" + id,
		"FileSystemId": fileSystemID,
		"Name":         "stackyard-volume",
		"VolumeType":   "ONTAP",
	}
}

func fsxAliasesFromPayload(payload map[string]any, lifecycle string) []any {
	names := fsxPayloadStringSlice(payload, "Aliases")
	if len(names) == 0 {
		names = []string{"files.example.com"}
	}
	aliases := make([]any, 0, len(names))
	for _, name := range names {
		aliases = append(aliases, map[string]any{"Name": name, "Lifecycle": lifecycle})
	}
	return aliases
}

func fsxDefaultS3AccessPointAttachment(payload map[string]any, volumeID, now string) map[string]any {
	name := fsxPayloadString(payload, "Name", "stackyard-s3-access-point")
	s3AccessPoint := map[string]any{
		"Name": name,
		"VpcConfiguration": map[string]any{
			"VpcId": "vpc-12345678",
		},
	}
	return map[string]any{
		"Name":          name,
		"Type":          "OPENZFS",
		"Lifecycle":     "CREATING",
		"CreationTime":  now,
		"S3AccessPoint": s3AccessPoint,
		"OpenZFSConfiguration": map[string]any{
			"VolumeId": volumeID,
			"FileSystemIdentity": map[string]any{
				"PosixUser": map[string]any{
					"Uid": 0,
					"Gid": 0,
				},
			},
		},
	}
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
