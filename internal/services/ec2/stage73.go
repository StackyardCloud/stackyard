package ec2

import (
	"sort"
	"strings"
	"time"
)

type FastLaunchLaunchTemplate struct {
	LaunchTemplateID   string
	LaunchTemplateName string
	Version            string
}

type FastLaunchSnapshotConfiguration struct {
	TargetResourceCount int32
}

type FastLaunchConfiguration struct {
	ImageID               string
	LaunchTemplate        *FastLaunchLaunchTemplate
	MaxParallelLaunches   *int32
	OwnerID               string
	ResourceType          string
	SnapshotConfiguration *FastLaunchSnapshotConfiguration
	State                 string
	StateTransitionReason string
	StateTransitionTime   time.Time
}

type FastSnapshotRestoreSuccess struct {
	AvailabilityZone      string
	DisabledTime          *time.Time
	DisablingTime         *time.Time
	EnabledTime           *time.Time
	EnablingTime          *time.Time
	OptimizingTime        *time.Time
	OwnerAlias            string
	OwnerID               string
	SnapshotID            string
	State                 string
	StateTransitionReason string
}

type FastSnapshotRestoreStateError struct {
	AvailabilityZone string
	Code             string
	Message          string
}

type FastSnapshotRestoreError struct {
	SnapshotID                     string
	FastSnapshotRestoreStateErrors []FastSnapshotRestoreStateError
}

func (s *Service) EnableImage(imageID string) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return false, ErrNotFound
	}

	image.State = "available"
	return true, nil
}

func (s *Service) DisableImage(imageID string) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return false, ErrNotFound
	}

	image.State = "disabled"
	return true, nil
}

func (s *Service) EnableFastLaunch(
	imageID string,
	launchTemplate *FastLaunchLaunchTemplate,
	maxParallelLaunches *int32,
	resourceType *string,
	snapshotConfiguration *FastLaunchSnapshotConfiguration,
) (FastLaunchConfiguration, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return FastLaunchConfiguration{}, ErrInvalidParameter
	}
	if maxParallelLaunches != nil && *maxParallelLaunches <= 0 {
		return FastLaunchConfiguration{}, ErrInvalidParameter
	}
	if snapshotConfiguration != nil && snapshotConfiguration.TargetResourceCount <= 0 {
		return FastLaunchConfiguration{}, ErrInvalidParameter
	}

	normalizedResourceType := "snapshot"
	if resourceType != nil {
		normalizedResourceType = strings.ToLower(strings.TrimSpace(*resourceType))
	}
	if normalizedResourceType != "snapshot" {
		return FastLaunchConfiguration{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.images[imageID] == nil {
		return FastLaunchConfiguration{}, ErrNotFound
	}

	now := time.Now().UTC()
	cfg := &FastLaunchConfiguration{
		ImageID:               imageID,
		LaunchTemplate:        cloneFastLaunchLaunchTemplate(launchTemplate),
		MaxParallelLaunches:   cloneInt32Pointer(maxParallelLaunches),
		OwnerID:               DefaultAccountID,
		ResourceType:          normalizedResourceType,
		SnapshotConfiguration: cloneFastLaunchSnapshotConfiguration(snapshotConfiguration),
		State:                 "enabled",
		StateTransitionReason: "Client.UserInitiated",
		StateTransitionTime:   now,
	}
	if cfg.MaxParallelLaunches == nil {
		cfg.MaxParallelLaunches = cloneInt32Pointer(int32Ptr(6))
	}
	if cfg.SnapshotConfiguration == nil {
		cfg.SnapshotConfiguration = &FastLaunchSnapshotConfiguration{TargetResourceCount: 1}
	}

	s.fastLaunchConfigurations[imageID] = cfg
	return cloneFastLaunchConfiguration(cfg), nil
}

func (s *Service) DisableFastLaunch(imageID string, force bool) (FastLaunchConfiguration, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return FastLaunchConfiguration{}, ErrInvalidParameter
	}
	_ = force

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.images[imageID] == nil {
		return FastLaunchConfiguration{}, ErrNotFound
	}

	cfg := s.fastLaunchConfigurations[imageID]
	if cfg == nil {
		cfg = &FastLaunchConfiguration{
			ImageID: imageID,
			OwnerID: DefaultAccountID,
		}
	}
	if cfg.ResourceType == "" {
		cfg.ResourceType = "snapshot"
	}
	if cfg.SnapshotConfiguration == nil {
		cfg.SnapshotConfiguration = &FastLaunchSnapshotConfiguration{TargetResourceCount: 1}
	}
	if cfg.MaxParallelLaunches == nil {
		cfg.MaxParallelLaunches = cloneInt32Pointer(int32Ptr(6))
	}
	cfg.State = "disabling"
	cfg.StateTransitionReason = "Client.UserInitiated"
	cfg.StateTransitionTime = time.Now().UTC()

	s.fastLaunchConfigurations[imageID] = cfg
	return cloneFastLaunchConfiguration(cfg), nil
}

func (s *Service) EnableFastSnapshotRestores(
	sourceSnapshotIDs []string,
	availabilityZones []string,
) ([]FastSnapshotRestoreSuccess, []FastSnapshotRestoreError, error) {
	snapshotIDs := dedupeTrimmedStrings(sourceSnapshotIDs)
	zones := dedupeTrimmedStrings(availabilityZones)
	if len(snapshotIDs) == 0 || len(zones) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]FastSnapshotRestoreSuccess, 0)
	unsuccessful := make([]FastSnapshotRestoreError, 0)
	now := time.Now().UTC()

	for _, snapshotID := range snapshotIDs {
		if s.snapshots[snapshotID] == nil {
			unsuccessful = append(unsuccessful, fastSnapshotRestoreError(snapshotID, zones, "InvalidSnapshot.NotFound", "snapshot not found"))
			continue
		}
		zoneState := s.fastSnapshotRestoreStates[snapshotID]
		if zoneState == nil {
			zoneState = map[string]bool{}
			s.fastSnapshotRestoreStates[snapshotID] = zoneState
		}
		for _, zone := range zones {
			zoneState[zone] = true
			successful = append(successful, FastSnapshotRestoreSuccess{
				AvailabilityZone:      zone,
				EnabledTime:           cloneTimePointer(&now),
				EnablingTime:          cloneTimePointer(&now),
				OwnerID:               DefaultAccountID,
				SnapshotID:            snapshotID,
				State:                 "enabled",
				StateTransitionReason: "Client.UserInitiated - Lifecycle state transition",
			})
		}
	}

	sortFastSnapshotRestoreSuccess(successful)
	sortFastSnapshotRestoreErrors(unsuccessful)
	return successful, unsuccessful, nil
}

func (s *Service) DisableFastSnapshotRestores(
	sourceSnapshotIDs []string,
	availabilityZones []string,
) ([]FastSnapshotRestoreSuccess, []FastSnapshotRestoreError, error) {
	snapshotIDs := dedupeTrimmedStrings(sourceSnapshotIDs)
	zones := dedupeTrimmedStrings(availabilityZones)
	if len(snapshotIDs) == 0 || len(zones) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]FastSnapshotRestoreSuccess, 0)
	unsuccessful := make([]FastSnapshotRestoreError, 0)
	now := time.Now().UTC()

	for _, snapshotID := range snapshotIDs {
		if s.snapshots[snapshotID] == nil {
			unsuccessful = append(unsuccessful, fastSnapshotRestoreError(snapshotID, zones, "InvalidSnapshot.NotFound", "snapshot not found"))
			continue
		}
		zoneState := s.fastSnapshotRestoreStates[snapshotID]
		if zoneState == nil {
			zoneState = map[string]bool{}
			s.fastSnapshotRestoreStates[snapshotID] = zoneState
		}
		for _, zone := range zones {
			zoneState[zone] = false
			successful = append(successful, FastSnapshotRestoreSuccess{
				AvailabilityZone:      zone,
				DisabledTime:          cloneTimePointer(&now),
				DisablingTime:         cloneTimePointer(&now),
				OwnerID:               DefaultAccountID,
				SnapshotID:            snapshotID,
				State:                 "disabled",
				StateTransitionReason: "Client.UserInitiated - Lifecycle state transition",
			})
		}
	}

	sortFastSnapshotRestoreSuccess(successful)
	sortFastSnapshotRestoreErrors(unsuccessful)
	return successful, unsuccessful, nil
}

func fastSnapshotRestoreError(snapshotID string, zones []string, code, message string) FastSnapshotRestoreError {
	errs := make([]FastSnapshotRestoreStateError, 0, len(zones))
	for _, zone := range zones {
		errs = append(errs, FastSnapshotRestoreStateError{
			AvailabilityZone: zone,
			Code:             code,
			Message:          message,
		})
	}
	return FastSnapshotRestoreError{
		SnapshotID:                     snapshotID,
		FastSnapshotRestoreStateErrors: errs,
	}
}

func sortFastSnapshotRestoreSuccess(in []FastSnapshotRestoreSuccess) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].SnapshotID != in[j].SnapshotID {
			return in[i].SnapshotID < in[j].SnapshotID
		}
		return in[i].AvailabilityZone < in[j].AvailabilityZone
	})
}

func sortFastSnapshotRestoreErrors(in []FastSnapshotRestoreError) {
	sort.Slice(in, func(i, j int) bool { return in[i].SnapshotID < in[j].SnapshotID })
	for i := range in {
		sort.Slice(in[i].FastSnapshotRestoreStateErrors, func(a, b int) bool {
			return in[i].FastSnapshotRestoreStateErrors[a].AvailabilityZone < in[i].FastSnapshotRestoreStateErrors[b].AvailabilityZone
		})
	}
}

func cloneFastLaunchConfiguration(in *FastLaunchConfiguration) FastLaunchConfiguration {
	if in == nil {
		return FastLaunchConfiguration{}
	}
	return FastLaunchConfiguration{
		ImageID:               in.ImageID,
		LaunchTemplate:        cloneFastLaunchLaunchTemplate(in.LaunchTemplate),
		MaxParallelLaunches:   cloneInt32Pointer(in.MaxParallelLaunches),
		OwnerID:               in.OwnerID,
		ResourceType:          in.ResourceType,
		SnapshotConfiguration: cloneFastLaunchSnapshotConfiguration(in.SnapshotConfiguration),
		State:                 in.State,
		StateTransitionReason: in.StateTransitionReason,
		StateTransitionTime:   in.StateTransitionTime,
	}
}

func cloneFastLaunchLaunchTemplate(in *FastLaunchLaunchTemplate) *FastLaunchLaunchTemplate {
	if in == nil {
		return nil
	}
	return &FastLaunchLaunchTemplate{
		LaunchTemplateID:   in.LaunchTemplateID,
		LaunchTemplateName: in.LaunchTemplateName,
		Version:            in.Version,
	}
}

func cloneFastLaunchSnapshotConfiguration(in *FastLaunchSnapshotConfiguration) *FastLaunchSnapshotConfiguration {
	if in == nil {
		return nil
	}
	return &FastLaunchSnapshotConfiguration{
		TargetResourceCount: in.TargetResourceCount,
	}
}

func int32Ptr(v int32) *int32 {
	return &v
}
