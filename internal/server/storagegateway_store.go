package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	storageGatewayDefaultGatewayARN               = "arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-00000001"
	storageGatewayDefaultVolumeARN                = "arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-00000001/volume/vol-0001"
	storageGatewayDefaultFileShareARN             = "arn:aws:storagegateway:us-east-1:123456789012:share/share-0001"
	storageGatewayDefaultTapeARN                  = "arn:aws:storagegateway:us-east-1:123456789012:tape/TEST000001"
	storageGatewayDefaultPoolARN                  = "arn:aws:storagegateway:us-east-1:123456789012:pool/pool-0001"
	storageGatewayDefaultFsAssociationARN         = "arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-00000001/file-system-association/fsa-0001"
	storageGatewayDefaultCacheReportARN           = "arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-00000001/cache-report/cr-0001"
	storageGatewayDefaultTapeRecoveryPointTapeARN = "arn:aws:storagegateway:us-east-1:123456789012:tape/TRP000001"
)

type storageGatewayStore struct {
	mu     sync.Mutex
	nextID int64
	tags   map[string]map[string]string
}

func newStorageGatewayStore() *storageGatewayStore {
	return &storageGatewayStore{
		nextID: 1,
		tags: map[string]map[string]string{
			storageGatewayDefaultGatewayARN: {
				"seed": "true",
			},
		},
	}
}

func (s *storageGatewayStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	template, ok := storageGatewayResponseTemplates[action]
	if !ok {
		return map[string]any{}
	}
	response := storageGatewayCloneMap(template)
	s.applyActionLocked(action, payload, response)
	return response
}

func (s *storageGatewayStore) applyActionLocked(action string, payload map[string]any, response map[string]any) {
	resourceARN := storageGatewayPayloadString(payload, "ResourceARN", storageGatewayDefaultGatewayARN)

	switch action {
	case "ActivateGateway", "StartGateway":
		response["GatewayARN"] = storageGatewayDefaultGatewayARN
	case "CreateCachediSCSIVolume", "CreateStorediSCSIVolume", "AttachVolume":
		response["VolumeARN"] = storageGatewayDefaultVolumeARN
	case "CreateNFSFileShare", "CreateSMBFileShare":
		response["FileShareARN"] = storageGatewayDefaultFileShareARN
	case "CreateSnapshot", "CreateSnapshotFromVolumeRecoveryPoint":
		response["SnapshotId"] = fmt.Sprintf("snap-%08d", s.nextID)
		s.nextID++
		response["VolumeARN"] = storageGatewayDefaultVolumeARN
	case "CreateTapePool":
		response["PoolARN"] = storageGatewayDefaultPoolARN
	case "CreateTapeWithBarcode":
		response["TapeARN"] = storageGatewayDefaultTapeARN
	case "CreateTapes":
		response["TapeARNs"] = []any{storageGatewayDefaultTapeARN}
	case "AssociateFileSystem":
		response["FileSystemAssociationARN"] = storageGatewayDefaultFsAssociationARN
	case "StartCacheReport":
		response["CacheReportARN"] = storageGatewayDefaultCacheReportARN
	case "AddTagsToResource":
		tags := storageGatewayTagsFromAny(payload["Tags"])
		if len(tags) > 0 {
			if s.tags[resourceARN] == nil {
				s.tags[resourceARN] = map[string]string{}
			}
			for k, v := range tags {
				s.tags[resourceARN][k] = v
			}
		}
	case "RemoveTagsFromResource":
		keys := storageGatewayPayloadStringSlice(payload, "TagKeys")
		if existing := s.tags[resourceARN]; len(existing) > 0 {
			for _, key := range keys {
				delete(existing, key)
			}
		}
	case "ListTagsForResource":
		response["Tags"] = storageGatewayTagsToList(s.tags[resourceARN])
	}
}

func initStorageGatewayResponseTemplates() map[string]map[string]any {
	out := make(map[string]map[string]any, len(storageGatewayOperations))
	for _, op := range storageGatewayOperations {
		out[op.Name] = map[string]any{}
	}

	out["ListGateways"] = map[string]any{
		"Gateways": []any{
			map[string]any{
				"GatewayARN":  storageGatewayDefaultGatewayARN,
				"GatewayType": "VTL",
			},
		},
	}
	out["ListLocalDisks"] = map[string]any{
		"Disks": []any{
			map[string]any{
				"DiskId":          "pci-0000:00:1f.2-scsi-0:0:0:0",
				"DiskSizeInBytes": 107374182400,
			},
		},
	}
	out["ListFileShares"] = map[string]any{"FileShareInfoList": []any{map[string]any{"FileShareARN": storageGatewayDefaultFileShareARN, "FileShareId": "share-0001", "FileShareType": "NFS"}}}
	out["ListFileSystemAssociations"] = map[string]any{"FileSystemAssociationSummaryList": []any{map[string]any{"FileSystemAssociationARN": storageGatewayDefaultFsAssociationARN}}}
	out["ListTapePools"] = map[string]any{"PoolInfos": []any{map[string]any{"PoolARN": storageGatewayDefaultPoolARN, "PoolName": "StackyardPool", "StorageClass": "GLACIER"}}}
	out["ListTapes"] = map[string]any{"TapeInfos": []any{map[string]any{"TapeARN": storageGatewayDefaultTapeARN, "TapeBarcode": "TEST000001"}}}
	out["ListVolumeInitiators"] = map[string]any{"Initiators": []any{"iqn.1993-08.org.debian:01:stackyard"}}
	out["ListVolumeRecoveryPoints"] = map[string]any{"VolumeRecoveryPointInfos": []any{map[string]any{"VolumeARN": storageGatewayDefaultVolumeARN}}}
	out["ListVolumes"] = map[string]any{"VolumeInfos": []any{map[string]any{"VolumeARN": storageGatewayDefaultVolumeARN, "VolumeType": "STORED"}}}
	out["DescribeGatewayInformation"] = map[string]any{"GatewayARN": storageGatewayDefaultGatewayARN, "GatewayName": "stackyard-gateway", "GatewayState": "RUNNING"}
	out["DescribeNFSFileShares"] = map[string]any{"NFSFileShareInfoList": []any{map[string]any{"FileShareARN": storageGatewayDefaultFileShareARN}}}
	out["DescribeSMBFileShares"] = map[string]any{"SMBFileShareInfoList": []any{map[string]any{"FileShareARN": storageGatewayDefaultFileShareARN}}}
	out["DescribeStorediSCSIVolumes"] = map[string]any{"StorediSCSIVolumes": []any{map[string]any{"VolumeARN": storageGatewayDefaultVolumeARN}}}
	out["DescribeCachediSCSIVolumes"] = map[string]any{"CachediSCSIVolumes": []any{map[string]any{"VolumeARN": storageGatewayDefaultVolumeARN}}}
	out["DescribeTapeArchives"] = map[string]any{"TapeArchives": []any{map[string]any{"TapeARN": storageGatewayDefaultTapeARN}}}
	out["DescribeTapeRecoveryPoints"] = map[string]any{"TapeRecoveryPointInfos": []any{map[string]any{"TapeARN": storageGatewayDefaultTapeRecoveryPointTapeARN}}}
	out["DescribeTapes"] = map[string]any{"Tapes": []any{map[string]any{"TapeARN": storageGatewayDefaultTapeARN}}}
	out["DescribeVTLDevices"] = map[string]any{"VTLDevices": []any{map[string]any{"VTLDeviceARN": "arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-00000001/device/vtl-dev-0001"}}}
	out["DescribeFileSystemAssociations"] = map[string]any{"FileSystemAssociationInfoList": []any{map[string]any{"FileSystemAssociationARN": storageGatewayDefaultFsAssociationARN}}}
	out["DescribeCacheReport"] = map[string]any{"CacheReport": map[string]any{"CacheReportARN": storageGatewayDefaultCacheReportARN, "ReportStatus": "COMPLETED"}}
	out["DescribeAvailabilityMonitorTest"] = map[string]any{"GatewayARN": storageGatewayDefaultGatewayARN, "Status": "COMPLETE", "StartTime": "2026-01-01T00:00:00Z"}
	out["ListAutomaticTapeCreationPolicies"] = map[string]any{"AutomaticTapeCreationPolicyInfos": []any{}}
	out["ListCacheReports"] = map[string]any{"CacheReportList": []any{}}
	out["DescribeSMBSettings"] = map[string]any{"SMBGuestPasswordSet": true, "SMBSecurityStrategy": "ClientSpecified"}
	out["DescribeSnapshotSchedule"] = map[string]any{"Description": "Default", "RecurrenceInHours": 24, "StartAt": 0, "Timezone": "GMT"}
	out["DescribeBandwidthRateLimit"] = map[string]any{"AverageUploadRateLimitInBitsPerSec": 0, "AverageDownloadRateLimitInBitsPerSec": 0}
	out["DescribeBandwidthRateLimitSchedule"] = map[string]any{"BandwidthRateLimitIntervals": []any{}}
	out["DescribeMaintenanceStartTime"] = map[string]any{"DayOfWeek": 1, "HourOfDay": 0, "MinuteOfHour": 0, "Timezone": "GMT"}
	out["DescribeCache"] = map[string]any{"GatewayARN": storageGatewayDefaultGatewayARN, "DiskIds": []any{"pci-0000:00:1f.2-scsi-0:0:0:0"}}
	out["DescribeUploadBuffer"] = map[string]any{"GatewayARN": storageGatewayDefaultGatewayARN, "DiskIds": []any{"pci-0000:00:1f.2-scsi-0:0:0:0"}}
	out["DescribeWorkingStorage"] = map[string]any{"GatewayARN": storageGatewayDefaultGatewayARN, "DiskIds": []any{"pci-0000:00:1f.2-scsi-0:0:0:0"}}
	out["DescribeChapCredentials"] = map[string]any{"ChapCredentials": []any{map[string]any{"TargetARN": storageGatewayDefaultVolumeARN}}}

	return out
}

var storageGatewayResponseTemplates = initStorageGatewayResponseTemplates()

func storageGatewayPayloadString(payload map[string]any, key, fallback string) string {
	if payload != nil {
		if value, ok := payload[key]; ok {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}
	return fallback
}

func storageGatewayPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func storageGatewayTagsFromAny(raw any) map[string]string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k, _ := m["Key"].(string)
		v, _ := m["Value"].(string)
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func storageGatewayTagsToList(in map[string]string) []any {
	if len(in) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": in[k]})
	}
	return out
}

func storageGatewayCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = storageGatewayCloneAny(v)
	}
	return out
}

func storageGatewayCloneAny(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return storageGatewayCloneMap(tv)
	case []any:
		out := make([]any, len(tv))
		for i := range tv {
			out[i] = storageGatewayCloneAny(tv[i])
		}
		return out
	default:
		return tv
	}
}
