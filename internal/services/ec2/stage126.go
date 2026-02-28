package ec2

import (
	"sort"
	"strings"
	"time"
)

type AccessScopeAnalysisFinding struct {
	FindingID                            string
	NetworkInsightsAccessScopeAnalysisID string
	NetworkInsightsAccessScopeID         string
}

type NetworkInsightsAccessScopeContent struct {
	NetworkInsightsAccessScopeID string
}

type ReservationValue struct {
	HourlyPrice           string
	RemainingTotalValue   string
	RemainingUpfrontValue string
}

type ReservedInstanceReservationValue struct {
	ReservedInstanceID string
	ReservationValue   ReservationValue
}

type TargetReservationValue struct {
	ReservationValue ReservationValue
}

type ReservedInstancesExchangeQuote struct {
	CurrencyCode                        string
	IsValidExchange                     bool
	OutputReservedInstancesWillExpireAt *time.Time
	PaymentDue                          string
	ReservedInstanceValueRollup         ReservationValue
	ReservedInstanceValueSet            []ReservedInstanceReservationValue
	TargetConfigurationValueRollup      ReservationValue
	TargetConfigurationValueSet         []TargetReservationValue
	ValidationFailureReason             string
}

type SpotPlacementScore struct {
	AvailabilityZoneID string
	Region             string
	Score              int32
}

type ImportImageResult struct {
	Architecture  string
	Description   string
	Encrypted     *bool
	Hypervisor    string
	ImageID       string
	ImportTaskID  string
	KmsKeyID      string
	LicenseType   string
	Platform      string
	Progress      string
	Status        string
	StatusMessage string
}

type ImportSnapshotResult struct {
	Description        string
	ImportTaskID       string
	SnapshotTaskDetail ImportSnapshotTaskDetail
	Tags               []Tag
}

type ImageRecycleBinInfo struct {
	Description         string
	ImageID             string
	Name                string
	RecycleBinEnterTime *time.Time
	RecycleBinExitTime  *time.Time
}

func (s *Service) GetManagedPrefixListEntries(prefixListID string, targetVersion *int64, maxResults *int32, nextToken *string) ([]ManagedPrefixListEntry, *string, error) {
	prefixListID = strings.TrimSpace(prefixListID)
	if prefixListID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	prefixList := s.managedPrefixLists[prefixListID]
	if prefixList == nil {
		s.mu.Unlock()
		return nil, nil, ErrNotFound
	}
	if targetVersion != nil {
		if *targetVersion <= 0 || *targetVersion > prefixList.Version {
			s.mu.Unlock()
			return nil, nil, ErrInvalidParameter
		}
	}
	entry := ManagedPrefixListEntry{
		CIDR:        stage126DefaultPrefixListCIDR(prefixList.AddressFamily),
		Description: "entry for " + prefixList.PrefixListName,
	}
	s.mu.Unlock()

	items := []ManagedPrefixListEntry{entry}
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ManagedPrefixListEntry(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetNetworkInsightsAccessScopeAnalysisFindings(networkInsightsAccessScopeAnalysisID string, maxResults *int32, nextToken *string) ([]AccessScopeAnalysisFinding, string, *string, error) {
	networkInsightsAccessScopeAnalysisID = strings.TrimSpace(networkInsightsAccessScopeAnalysisID)
	if networkInsightsAccessScopeAnalysisID == "" {
		return nil, "", nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, "", nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	scopeID := stage126ScopeIDFromAnalysisIDLocked(s.networkInsightsAccessScopes, networkInsightsAccessScopeAnalysisID)
	s.mu.Unlock()
	if scopeID == "" {
		return nil, "", nil, ErrNotFound
	}

	findings := []AccessScopeAnalysisFinding{
		{
			FindingID:                            "finding-" + strings.TrimPrefix(networkInsightsAccessScopeAnalysisID, "niasa-"),
			NetworkInsightsAccessScopeAnalysisID: networkInsightsAccessScopeAnalysisID,
			NetworkInsightsAccessScopeID:         scopeID,
		},
	}

	start, end, outputToken, err := ec2PageWindow(len(findings), start, maxResults)
	if err != nil {
		return nil, "", nil, ErrInvalidParameter
	}
	return append([]AccessScopeAnalysisFinding(nil), findings[start:end]...), "succeeded", outputToken, nil
}

func (s *Service) GetNetworkInsightsAccessScopeContent(networkInsightsAccessScopeID string) (NetworkInsightsAccessScopeContent, error) {
	networkInsightsAccessScopeID = strings.TrimSpace(networkInsightsAccessScopeID)
	if networkInsightsAccessScopeID == "" {
		return NetworkInsightsAccessScopeContent{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkInsightsAccessScopes[networkInsightsAccessScopeID] == nil {
		return NetworkInsightsAccessScopeContent{}, ErrNotFound
	}
	return NetworkInsightsAccessScopeContent{NetworkInsightsAccessScopeID: networkInsightsAccessScopeID}, nil
}

func (s *Service) GetReservedInstancesExchangeQuote(reservedInstanceIDs []string) (ReservedInstancesExchangeQuote, error) {
	reservedInstanceIDs = dedupeTrimmedStrings(reservedInstanceIDs)
	if len(reservedInstanceIDs) == 0 {
		return ReservedInstancesExchangeQuote{}, ErrInvalidParameter
	}

	sort.Strings(reservedInstanceIDs)

	rollup := ReservationValue{
		HourlyPrice:           "0.0000",
		RemainingTotalValue:   "0.0000",
		RemainingUpfrontValue: "0.0000",
	}
	valueSet := make([]ReservedInstanceReservationValue, 0, len(reservedInstanceIDs))
	for _, reservedInstanceID := range reservedInstanceIDs {
		valueSet = append(valueSet, ReservedInstanceReservationValue{
			ReservedInstanceID: reservedInstanceID,
			ReservationValue:   rollup,
		})
	}

	expiration := time.Now().UTC().Add(365 * 24 * time.Hour)
	return ReservedInstancesExchangeQuote{
		CurrencyCode:                        "USD",
		IsValidExchange:                     true,
		OutputReservedInstancesWillExpireAt: cloneTimePointer(&expiration),
		PaymentDue:                          "0.0000",
		ReservedInstanceValueRollup:         rollup,
		ReservedInstanceValueSet:            valueSet,
		TargetConfigurationValueRollup:      rollup,
		TargetConfigurationValueSet:         []TargetReservationValue{{ReservationValue: rollup}},
		ValidationFailureReason:             "",
	}, nil
}

func (s *Service) GetSpotPlacementScores(targetCapacity int32, instanceTypes []string, regionNames []string, singleAvailabilityZone *bool, targetCapacityUnitType string, maxResults *int32, nextToken *string) ([]SpotPlacementScore, *string, error) {
	if targetCapacity <= 0 {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	_ = strings.TrimSpace(targetCapacityUnitType)
	instanceTypes = dedupeTrimmedStrings(instanceTypes)
	regionNames = dedupeTrimmedStrings(regionNames)

	s.mu.Lock()
	if len(regionNames) == 0 {
		regionNames = make([]string, 0, len(s.regions))
		for _, region := range s.regions {
			if name := strings.TrimSpace(region.Name); name != "" {
				regionNames = append(regionNames, name)
			}
		}
	}
	if len(regionNames) == 0 {
		regionNames = []string{DefaultRegion}
	}
	sort.Strings(regionNames)

	baseScore := int32(10)
	switch len(instanceTypes) {
	case 1:
		baseScore = 6
	case 2:
		baseScore = 8
	}
	if targetCapacity >= 100 {
		baseScore--
	}
	if targetCapacity >= 500 {
		baseScore--
	}
	if baseScore < 1 {
		baseScore = 1
	}

	items := make([]SpotPlacementScore, 0, len(regionNames))
	for idx, regionName := range regionNames {
		score := baseScore - int32(idx%3)
		if score < 1 {
			score = 1
		}
		if score > 10 {
			score = 10
		}
		item := SpotPlacementScore{
			Region: regionName,
			Score:  score,
		}
		if singleAvailabilityZone != nil && *singleAvailabilityZone {
			item.AvailabilityZoneID = stage126AvailabilityZoneIDForRegionLocked(s.availabilityZones, regionName)
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SpotPlacementScore(nil), items[start:end]...), outputToken, nil
}

func (s *Service) ImportImage(architecture string, description string, hypervisor string, kmsKeyID string, licenseType string, platform string, roleName string, encrypted *bool, tags []Tag) (ImportImageResult, error) {
	architecture = strings.TrimSpace(architecture)
	description = strings.TrimSpace(description)
	hypervisor = strings.TrimSpace(hypervisor)
	kmsKeyID = strings.TrimSpace(kmsKeyID)
	licenseType = strings.TrimSpace(licenseType)
	platform = strings.TrimSpace(platform)
	_ = strings.TrimSpace(roleName)

	if architecture == "" {
		architecture = "x86_64"
	}
	if hypervisor == "" {
		hypervisor = "xen"
	}
	if description == "" {
		description = "import image task"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	importTaskID := s.nextIDLocked("import-ami")
	s.importTaskStates[importTaskID] = "active"
	delete(s.importTaskCancelReasons, importTaskID)

	imageID := stage117ImageIDFromImportTaskID(importTaskID)
	s.images[imageID] = &Image{
		ID:                       imageID,
		Name:                     "imported-" + imageID,
		Description:              description,
		DeprecationTime:          nil,
		DeregistrationProtection: "disabled",
		State:                    "available",
		OwnerID:                  DefaultAccountID,
		ImageLocation:            DefaultAccountID + "/" + imageID,
		Architecture:             architecture,
		ImageType:                "machine",
		RootDeviceType:           "ebs",
		RootDeviceName:           "/dev/sda1",
		VirtualizationType:       "hvm",
		CreationDate:             time.Now().UTC(),
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
	}

	statusMessage := "import task active"
	if reason := strings.TrimSpace(s.importTaskCancelReasons[importTaskID]); reason != "" {
		statusMessage = reason
	}

	return ImportImageResult{
		Architecture:  architecture,
		Description:   description,
		Encrypted:     cloneBoolPointer(encrypted),
		Hypervisor:    hypervisor,
		ImageID:       imageID,
		ImportTaskID:  importTaskID,
		KmsKeyID:      kmsKeyID,
		LicenseType:   licenseType,
		Platform:      platform,
		Progress:      "0",
		Status:        "active",
		StatusMessage: statusMessage,
	}, nil
}

func (s *Service) ImportInstance(description string, platform string) (ConversionTask, error) {
	platform = strings.TrimSpace(platform)
	description = strings.TrimSpace(description)
	if platform == "" {
		return ConversionTask{}, ErrInvalidParameter
	}
	if description == "" {
		description = "import instance task"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	conversionTaskID := s.nextIDLocked("import-i")
	s.conversionTaskStates[conversionTaskID] = "active"
	delete(s.conversionTaskCancelReasons, conversionTaskID)

	expiration := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	return ConversionTask{
		ConversionTaskID: conversionTaskID,
		ExpirationTime:   expiration,
		State:            "active",
		StatusMessage:    description,
		Tags:             map[string]string{},
	}, nil
}

func (s *Service) ImportSnapshot(description string, tags []Tag) (ImportSnapshotResult, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		description = "import snapshot task"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	importTaskID := s.nextIDLocked("import-snap")
	s.importTaskStates[importTaskID] = "active"
	delete(s.importTaskCancelReasons, importTaskID)

	snapshotID := stage117SnapshotIDFromImportTaskID(importTaskID)
	s.snapshots[snapshotID] = &Snapshot{
		ID:          snapshotID,
		VolumeID:    "",
		State:       "completed",
		StartTime:   time.Now().UTC(),
		Progress:    "100%",
		Description: description,
		VolumeSize:  8,
		Tags:        tagsToMap(normalizeEC2Tags(tags)),
	}

	return ImportSnapshotResult{
		Description:  description,
		ImportTaskID: importTaskID,
		SnapshotTaskDetail: ImportSnapshotTaskDetail{
			Description:   description,
			Progress:      "0",
			SnapshotID:    snapshotID,
			Status:        "active",
			StatusMessage: "import task active",
		},
		Tags: cloneEC2Tags(tags),
	}, nil
}

func (s *Service) ImportVolume(description string, availabilityZone string, availabilityZoneID string) (ConversionTask, error) {
	description = strings.TrimSpace(description)
	availabilityZone = strings.TrimSpace(availabilityZone)
	availabilityZoneID = strings.TrimSpace(availabilityZoneID)
	if availabilityZone != "" && availabilityZoneID != "" {
		return ConversionTask{}, ErrInvalidParameter
	}
	if description == "" {
		description = "import volume task"
	}
	if availabilityZone == "" {
		availabilityZone = "us-east-1a"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	conversionTaskID := s.nextIDLocked("import-vol")
	s.conversionTaskStates[conversionTaskID] = "active"
	delete(s.conversionTaskCancelReasons, conversionTaskID)

	volumeID := stage126VolumeIDFromConversionTaskID(conversionTaskID)
	s.volumes[volumeID] = &Volume{
		ID:               volumeID,
		AvailabilityZone: availabilityZone,
		SizeGiB:          8,
		State:            "available",
		VolumeType:       "gp3",
		Iops:             3000,
		Throughput:       125,
		CreateTime:       time.Now().UTC(),
		Tags:             map[string]string{},
	}

	expiration := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	return ConversionTask{
		ConversionTaskID: conversionTaskID,
		ExpirationTime:   expiration,
		State:            "active",
		StatusMessage:    description,
		Tags:             map[string]string{},
	}, nil
}

func (s *Service) ListImagesInRecycleBin(imageIDs []string, maxResults *int32, nextToken *string) ([]ImageRecycleBinInfo, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedImageIDs := dedupeTrimmedStrings(imageIDs)

	s.mu.Lock()
	candidateImageIDs := append([]string(nil), requestedImageIDs...)
	if len(candidateImageIDs) == 0 {
		candidateImageIDs = make([]string, 0, len(s.images))
		for imageID := range s.images {
			candidateImageIDs = append(candidateImageIDs, imageID)
		}
		sort.Strings(candidateImageIDs)
	}

	items := make([]ImageRecycleBinInfo, 0, len(candidateImageIDs))
	for _, imageID := range candidateImageIDs {
		image := s.images[imageID]
		if image == nil {
			continue
		}
		enter := image.CreationDate.UTC()
		if enter.IsZero() {
			enter = time.Now().UTC().Add(-1 * time.Hour)
		}
		exit := enter.Add(30 * 24 * time.Hour)
		items = append(items, ImageRecycleBinInfo{
			Description:         firstNonEmptyString(image.Description, "recycled image"),
			ImageID:             imageID,
			Name:                firstNonEmptyString(image.Name, imageID),
			RecycleBinEnterTime: cloneTimePointer(&enter),
			RecycleBinExitTime:  cloneTimePointer(&exit),
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ImageRecycleBinInfo(nil), items[start:end]...), outputToken, nil
}

func stage126DefaultPrefixListCIDR(addressFamily string) string {
	if strings.EqualFold(strings.TrimSpace(addressFamily), "ipv6") {
		return "2001:db8::/64"
	}
	return "10.0.0.0/24"
}

func stage126ScopeIDFromAnalysisIDLocked(scopes map[string]*NetworkInsightsAccessScope, analysisID string) string {
	analysisID = strings.TrimSpace(analysisID)
	if analysisID == "" {
		return ""
	}
	scopeIDs := make([]string, 0, len(scopes))
	for scopeID := range scopes {
		scopeIDs = append(scopeIDs, scopeID)
	}
	sort.Strings(scopeIDs)
	for _, scopeID := range scopeIDs {
		if stage120NetworkInsightsAccessScopeAnalysisID(scopeID) == analysisID {
			return scopeID
		}
	}
	return ""
}

func stage126AvailabilityZoneIDForRegionLocked(availabilityZones []AvailabilityZone, regionName string) string {
	regionName = strings.TrimSpace(regionName)
	if regionName == "" {
		return ""
	}
	for _, zone := range availabilityZones {
		if strings.EqualFold(strings.TrimSpace(zone.Region), regionName) {
			return strings.TrimSpace(zone.ZoneID)
		}
	}
	return ""
}

func stage126VolumeIDFromConversionTaskID(conversionTaskID string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(conversionTaskID), "import-vol-")
	if suffix == "" || suffix == conversionTaskID {
		suffix = strings.ReplaceAll(strings.TrimSpace(conversionTaskID), "-", "")
	}
	if suffix == "" {
		suffix = "000000000000"
	}
	return "vol-" + suffix
}
