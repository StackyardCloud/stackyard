package ec2

import "strings"

func (s *Service) CopySnapshot(
	sourceSnapshotID string,
	sourceRegion string,
	completionDurationMinutes *int32,
	description *string,
	encrypted *bool,
	kmsKeyID *string,
	tags []Tag,
) (Snapshot, error) {
	sourceSnapshotID = strings.TrimSpace(sourceSnapshotID)
	sourceRegion = strings.TrimSpace(sourceRegion)
	if sourceSnapshotID == "" || sourceRegion == "" {
		return Snapshot{}, ErrInvalidParameter
	}
	if completionDurationMinutes != nil && *completionDurationMinutes <= 0 {
		return Snapshot{}, ErrInvalidParameter
	}
	if encrypted != nil && !*encrypted {
		return Snapshot{}, ErrInvalidParameter
	}
	if kmsKeyID != nil && strings.TrimSpace(*kmsKeyID) != "" && (encrypted == nil || !*encrypted) {
		return Snapshot{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	source := s.snapshots[sourceSnapshotID]
	if source == nil {
		return Snapshot{}, ErrNotFound
	}

	copiedDescription := source.Description
	if description != nil {
		copiedDescription = strings.TrimSpace(*description)
	}

	copied := &Snapshot{
		ID:          s.nextIDLocked("snap"),
		VolumeID:    source.VolumeID,
		State:       "completed",
		StartTime:   source.StartTime,
		Progress:    "100%",
		Description: copiedDescription,
		VolumeSize:  source.VolumeSize,
		Tags:        tagsToMap(tags),
	}
	s.snapshots[copied.ID] = copied
	return cloneSnapshot(copied), nil
}
