package ec2

import (
	"strings"
	"time"
)

type VolumeModification struct {
	VolumeID                   string
	ModificationState          string
	StartTime                  time.Time
	EndTime                    time.Time
	Progress                   int64
	StatusMessage              string
	OriginalSize               int32
	TargetSize                 int32
	OriginalVolumeType         string
	TargetVolumeType           string
	OriginalIops               int32
	TargetIops                 int32
	OriginalThroughput         int32
	TargetThroughput           int32
	OriginalMultiAttachEnabled bool
	TargetMultiAttachEnabled   bool
}

func (s *Service) ModifyVolume(
	volumeID string,
	size,
	iops,
	throughput *int32,
	volumeType *string,
	multiAttachEnabled *bool,
) (VolumeModification, error) {
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" {
		return VolumeModification{}, ErrInvalidParameter
	}

	if size != nil {
		if *size <= 0 {
			return VolumeModification{}, ErrInvalidParameter
		}
	}
	if iops != nil {
		if *iops <= 0 {
			return VolumeModification{}, ErrInvalidParameter
		}
	}
	if throughput != nil {
		if *throughput <= 0 {
			return VolumeModification{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	volume := s.volumes[volumeID]
	if volume == nil {
		return VolumeModification{}, ErrNotFound
	}

	if size != nil && *size < volume.SizeGiB {
		return VolumeModification{}, ErrInvalidParameter
	}

	targetVolumeType := volume.VolumeType
	if volumeType != nil {
		trimmed := strings.TrimSpace(*volumeType)
		if trimmed == "" {
			return VolumeModification{}, ErrInvalidParameter
		}
		targetVolumeType = trimmed
	}

	targetSize := volume.SizeGiB
	if size != nil {
		targetSize = *size
	}
	targetIops := volume.Iops
	if iops != nil {
		targetIops = *iops
	}
	targetThroughput := volume.Throughput
	if throughput != nil {
		targetThroughput = *throughput
	}
	targetMultiAttach := volume.MultiAttach
	if multiAttachEnabled != nil {
		targetMultiAttach = *multiAttachEnabled
	}

	now := time.Now().UTC()
	modification := VolumeModification{
		VolumeID:                   volume.ID,
		ModificationState:          "completed",
		StartTime:                  now,
		EndTime:                    now,
		Progress:                   100,
		StatusMessage:              "completed",
		OriginalSize:               volume.SizeGiB,
		TargetSize:                 targetSize,
		OriginalVolumeType:         volume.VolumeType,
		TargetVolumeType:           targetVolumeType,
		OriginalIops:               volume.Iops,
		TargetIops:                 targetIops,
		OriginalThroughput:         volume.Throughput,
		TargetThroughput:           targetThroughput,
		OriginalMultiAttachEnabled: volume.MultiAttach,
		TargetMultiAttachEnabled:   targetMultiAttach,
	}

	volume.SizeGiB = targetSize
	volume.VolumeType = targetVolumeType
	volume.Iops = targetIops
	volume.Throughput = targetThroughput
	volume.MultiAttach = targetMultiAttach

	return modification, nil
}

func (s *Service) ModifyVolumeAttribute(volumeID string, autoEnableIO *bool) error {
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" || autoEnableIO == nil {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	volume := s.volumes[volumeID]
	if volume == nil {
		return ErrNotFound
	}

	volume.AutoEnableIO = *autoEnableIO
	return nil
}
