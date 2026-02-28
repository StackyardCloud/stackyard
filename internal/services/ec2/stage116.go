package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type FleetHistoryRecord struct {
	EventDescription string
	EventSubType     string
	EventType        string
	InstanceID       string
	Timestamp        time.Time
}

type FleetActiveInstance struct {
	InstanceHealth        string
	InstanceID            string
	InstanceType          string
	SpotInstanceRequestID string
}

type FpgaImageAttributeView struct {
	Description           string
	FpgaImageID           string
	LoadPermissionUserIDs []string
	Name                  string
	ProductCodes          []string
}

type HostOffering struct {
	CurrencyCode   string
	Duration       int32
	HourlyPrice    string
	InstanceFamily string
	OfferingID     string
	PaymentOption  string
	UpfrontPrice   string
}

type HostReservation struct {
	Count             int32
	CurrencyCode      string
	Duration          int32
	End               time.Time
	HostIDs           []string
	HostReservationID string
	HourlyPrice       string
	InstanceFamily    string
	OfferingID        string
	PaymentOption     string
	Start             time.Time
	State             string
	Tags              map[string]string
	UpfrontPrice      string
}

func (s *Service) DescribeFastLaunchImages(imageIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]FastLaunchConfiguration, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(imageIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	imageIDFilterSet := toStringSet(standardFilters["image-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	resourceTypeFilterSet := toLowerStringSet(standardFilters["resource-type"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for imageID := range s.fastLaunchConfigurations {
			candidateIDs = append(candidateIDs, imageID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]FastLaunchConfiguration, 0, len(candidateIDs))
	for _, imageID := range candidateIDs {
		cfg := s.fastLaunchConfigurations[imageID]
		if cfg == nil {
			if len(requestedIDs) == 0 {
				continue
			}
			cfg = &FastLaunchConfiguration{
				ImageID:      imageID,
				OwnerID:      DefaultAccountID,
				ResourceType: "snapshot",
				State:        "disabled",
			}
		}
		item := cloneFastLaunchConfiguration(cfg)
		if item.OwnerID == "" {
			item.OwnerID = DefaultAccountID
		}
		if item.ResourceType == "" {
			item.ResourceType = "snapshot"
		}
		if item.State == "" {
			item.State = "disabled"
		}

		if len(imageIDFilterSet) > 0 {
			if _, ok := imageIDFilterSet[item.ImageID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if len(resourceTypeFilterSet) > 0 {
			if _, ok := resourceTypeFilterSet[strings.ToLower(item.ResourceType)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}

		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]FastLaunchConfiguration(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeFastSnapshotRestores(filters map[string][]string, maxResults *int32, nextToken *string) ([]FastSnapshotRestoreSuccess, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	snapshotIDFilterSet := toStringSet(standardFilters["snapshot-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	now := time.Now().UTC()

	s.mu.Lock()
	snapshotIDs := make([]string, 0, len(s.fastSnapshotRestoreStates))
	for snapshotID := range s.fastSnapshotRestoreStates {
		snapshotIDs = append(snapshotIDs, snapshotID)
	}
	sort.Strings(snapshotIDs)

	out := make([]FastSnapshotRestoreSuccess, 0)
	for _, snapshotID := range snapshotIDs {
		if len(snapshotIDFilterSet) > 0 {
			if _, ok := snapshotIDFilterSet[snapshotID]; !ok {
				continue
			}
		}

		zoneStates := s.fastSnapshotRestoreStates[snapshotID]
		zoneNames := make([]string, 0, len(zoneStates))
		for zone := range zoneStates {
			zoneNames = append(zoneNames, zone)
		}
		sort.Strings(zoneNames)

		for _, zone := range zoneNames {
			enabled := zoneStates[zone]
			item := FastSnapshotRestoreSuccess{
				AvailabilityZone:      zone,
				OwnerID:               DefaultAccountID,
				SnapshotID:            snapshotID,
				StateTransitionReason: "Client.UserInitiated - Lifecycle state transition",
			}
			if enabled {
				item.State = "enabled"
				item.EnabledTime = cloneTimePointer(&now)
				item.EnablingTime = cloneTimePointer(&now)
			} else {
				item.State = "disabled"
				item.DisabledTime = cloneTimePointer(&now)
				item.DisablingTime = cloneTimePointer(&now)
			}

			if len(availabilityZoneFilterSet) > 0 {
				if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
					continue
				}
			}
			if len(ownerIDFilterSet) > 0 {
				if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
					continue
				}
			}
			if len(stateFilterSet) > 0 {
				if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
					continue
				}
			}

			out = append(out, item)
		}
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]FastSnapshotRestoreSuccess(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeFleetHistory(fleetID string, startTime time.Time, eventType string, maxResults *int32, nextToken *string) (string, []FleetHistoryRecord, *time.Time, *string, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" || startTime.IsZero() {
		return "", nil, nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return "", nil, nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return "", nil, nil, nil, ErrInvalidParameter
	}

	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if eventType == "" {
		eventType = "instance-change"
	}

	base := startTime.UTC()

	s.mu.Lock()
	fleet := s.fleets[fleetID]
	if fleet == nil {
		s.mu.Unlock()
		return "", nil, nil, nil, ErrNotFound
	}

	records := make([]FleetHistoryRecord, 0)
	for _, fleetInstance := range fleet.Instances {
		for _, instanceID := range fleetInstance.InstanceIDs {
			records = append(records, FleetHistoryRecord{
				EventDescription: "Instance launched",
				EventSubType:     "submitted",
				EventType:        "instance-change",
				InstanceID:       instanceID,
				Timestamp:        base,
			})
			base = base.Add(1 * time.Second)
		}
	}
	if len(records) == 0 {
		records = append(records, FleetHistoryRecord{
			EventDescription: "Fleet state changed",
			EventSubType:     "state-change",
			EventType:        "fleet-change",
			Timestamp:        base,
		})
	}
	s.mu.Unlock()

	filtered := make([]FleetHistoryRecord, 0, len(records))
	for _, record := range records {
		if record.Timestamp.Before(startTime.UTC()) {
			continue
		}
		if eventType != "" && !strings.EqualFold(record.EventType, eventType) {
			continue
		}
		filtered = append(filtered, record)
	}

	start, end, outputToken, err := ec2PageWindow(len(filtered), start, maxResults)
	if err != nil {
		return "", nil, nil, nil, ErrInvalidParameter
	}
	page := append([]FleetHistoryRecord(nil), filtered[start:end]...)

	var lastEvaluatedTime *time.Time
	if outputToken == nil && len(page) > 0 {
		last := page[len(page)-1].Timestamp.UTC()
		lastEvaluatedTime = &last
	}

	return fleetID, page, lastEvaluatedTime, outputToken, nil
}

func (s *Service) DescribeFleetInstances(fleetID string, filters map[string][]string, maxResults *int32, nextToken *string) (string, []FleetActiveInstance, *string, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return "", nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return "", nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return "", nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])

	s.mu.Lock()
	fleet := s.fleets[fleetID]
	if fleet == nil {
		s.mu.Unlock()
		return "", nil, nil, ErrNotFound
	}

	out := make([]FleetActiveInstance, 0)
	for _, fleetInstance := range fleet.Instances {
		for _, instanceID := range fleetInstance.InstanceIDs {
			item := FleetActiveInstance{
				InstanceHealth: "healthy",
				InstanceID:     instanceID,
				InstanceType:   fleetInstance.InstanceType,
			}
			if len(instanceTypeFilterSet) > 0 {
				if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
					continue
				}
			}
			out = append(out, item)
		}
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return "", nil, nil, ErrInvalidParameter
	}
	return fleetID, append([]FleetActiveInstance(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeFleets(fleetIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]Fleet, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(fleetIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	fleetIDFilterSet := toStringSet(standardFilters["fleet-id"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for fleetID := range s.fleets {
			candidateIDs = append(candidateIDs, fleetID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]Fleet, 0, len(candidateIDs))
	for _, fleetID := range candidateIDs {
		fleet := s.fleets[fleetID]
		if fleet == nil {
			continue
		}
		if len(fleetIDFilterSet) > 0 {
			if _, ok := fleetIDFilterSet[fleet.FleetID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(fleet.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		out = append(out, cloneStage107Fleet(fleet))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]Fleet(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeFlowLogs(flowLogIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]FlowLog, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(flowLogIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	flowLogIDFilterSet := toStringSet(standardFilters["flow-log-id"])
	resourceIDFilterSet := toStringSet(standardFilters["resource-id"])
	trafficTypeFilterSet := toLowerStringSet(standardFilters["traffic-type"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for flowLogID := range s.flowLogs {
			candidateIDs = append(candidateIDs, flowLogID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]FlowLog, 0, len(candidateIDs))
	for _, flowLogID := range candidateIDs {
		flowLog := s.flowLogs[flowLogID]
		if flowLog == nil {
			if len(requestedIDs) == 0 {
				continue
			}
			flowLog = &FlowLog{
				FlowLogID:    flowLogID,
				ResourceID:   "vpc-00000000116",
				ResourceType: "VPC",
				TrafficType:  "ALL",
				Tags:         map[string]string{},
			}
		}
		item := cloneStage116FlowLog(flowLog)
		if len(flowLogIDFilterSet) > 0 {
			if _, ok := flowLogIDFilterSet[item.FlowLogID]; !ok {
				continue
			}
		}
		if len(resourceIDFilterSet) > 0 {
			if _, ok := resourceIDFilterSet[item.ResourceID]; !ok {
				continue
			}
		}
		if len(trafficTypeFilterSet) > 0 {
			if _, ok := trafficTypeFilterSet[strings.ToLower(item.TrafficType)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]FlowLog(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeFpgaImageAttribute(fpgaImageID, attribute string) (FpgaImageAttributeView, error) {
	fpgaImageID = strings.TrimSpace(fpgaImageID)
	attribute = strings.TrimSpace(attribute)
	if fpgaImageID == "" || attribute == "" {
		return FpgaImageAttributeView{}, ErrInvalidParameter
	}

	normalizedAttribute := strings.ToLower(attribute)
	switch normalizedAttribute {
	case "description", "name", "loadpermission", "productcodes":
	default:
		return FpgaImageAttributeView{}, ErrInvalidParameter
	}

	s.mu.Lock()
	image := s.fpgaImages[fpgaImageID]
	if image == nil {
		s.mu.Unlock()
		return FpgaImageAttributeView{}, ErrNotFound
	}
	out := FpgaImageAttributeView{
		FpgaImageID:           image.FpgaImageID,
		LoadPermissionUserIDs: []string{DefaultAccountID},
		ProductCodes:          []string{"stackyard"},
	}
	switch normalizedAttribute {
	case "description":
		out.Description = image.Description
	case "name":
		out.Name = image.Name
	case "loadpermission":
		// load permissions are returned in LoadPermissionUserIDs.
	case "productcodes":
		// product codes are returned in ProductCodes.
	}
	s.mu.Unlock()

	return out, nil
}

func (s *Service) DescribeFpgaImages(fpgaImageIDs, owners []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]FpgaImage, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(fpgaImageIDs)
	ownerFilterSet := normalizeStage116OwnerFilter(owners)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	fpgaImageIDFilterSet := toStringSet(standardFilters["fpga-image-id"])
	fpgaImageGlobalIDFilterSet := toStringSet(standardFilters["fpga-image-global-id"])
	nameFilterSet := toStringSet(standardFilters["name"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		for fpgaImageID := range s.fpgaImages {
			candidateIDs = append(candidateIDs, fpgaImageID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]FpgaImage, 0, len(candidateIDs))
	for _, fpgaImageID := range candidateIDs {
		image := s.fpgaImages[fpgaImageID]
		if image == nil {
			continue
		}
		item := cloneStage107FpgaImage(image)

		if len(fpgaImageIDFilterSet) > 0 {
			if _, ok := fpgaImageIDFilterSet[item.FpgaImageID]; !ok {
				continue
			}
		}
		if len(fpgaImageGlobalIDFilterSet) > 0 {
			if _, ok := fpgaImageGlobalIDFilterSet[item.FpgaImageGlobalID]; !ok {
				continue
			}
		}
		if len(nameFilterSet) > 0 {
			if _, ok := nameFilterSet[item.Name]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet["available"]; !ok {
				continue
			}
		}
		if len(ownerFilterSet) > 0 {
			if _, ok := ownerFilterSet[DefaultAccountID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[DefaultAccountID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]FpgaImage(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeHostReservationOfferings(offeringID *string, minDuration, maxDuration *int32, filters map[string][]string, maxResults *int32, nextToken *string) ([]HostOffering, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil {
		if *maxResults < 0 || *maxResults > 500 {
			return nil, nil, ErrInvalidParameter
		}
	}
	if minDuration != nil && *minDuration < 0 {
		return nil, nil, ErrInvalidParameter
	}
	if maxDuration != nil && *maxDuration < 0 {
		return nil, nil, ErrInvalidParameter
	}
	if minDuration != nil && maxDuration != nil && *maxDuration < *minDuration {
		return nil, nil, ErrInvalidParameter
	}

	offeringIDValue := strings.TrimSpace(derefString(offeringID))
	standardFilters, _, _ := splitEC2Filters(filters)
	instanceFamilyFilterSet := toLowerStringSet(standardFilters["instance-family"])
	paymentOptionFilterSet := toLowerStringSet(standardFilters["payment-option"])

	s.mu.Lock()
	familySet := map[string]struct{}{}
	for _, host := range s.dedicatedHosts {
		if host == nil {
			continue
		}
		family := strings.TrimSpace(host.InstanceFamily)
		if family == "" {
			family = stage116InstanceFamilyFromType(host.InstanceType)
		}
		if family == "" {
			family = "m5"
		}
		familySet[family] = struct{}{}
	}
	if len(familySet) == 0 {
		familySet["m5"] = struct{}{}
	}
	families := make([]string, 0, len(familySet))
	for family := range familySet {
		families = append(families, family)
	}
	sort.Strings(families)

	out := make([]HostOffering, 0, len(families))
	for idx, family := range families {
		item := HostOffering{
			CurrencyCode:   "USD",
			Duration:       31536000,
			HourlyPrice:    "0.75",
			InstanceFamily: family,
			OfferingID:     fmt.Sprintf("hro-%03d", idx+1),
			PaymentOption:  "NoUpfront",
			UpfrontPrice:   "0.00",
		}
		if offeringIDValue != "" && item.OfferingID != offeringIDValue {
			continue
		}
		if minDuration != nil && item.Duration < *minDuration {
			continue
		}
		if maxDuration != nil && item.Duration > *maxDuration {
			continue
		}
		if len(instanceFamilyFilterSet) > 0 {
			if _, ok := instanceFamilyFilterSet[strings.ToLower(item.InstanceFamily)]; !ok {
				continue
			}
		}
		if len(paymentOptionFilterSet) > 0 {
			if _, ok := paymentOptionFilterSet[strings.ToLower(item.PaymentOption)]; !ok {
				continue
			}
		}
		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]HostOffering(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeHostReservations(hostReservationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]HostReservation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil {
		if *maxResults < 0 || *maxResults > 500 {
			return nil, nil, ErrInvalidParameter
		}
	}

	requestedIDs := dedupeTrimmedStrings(hostReservationIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	instanceFamilyFilterSet := toLowerStringSet(standardFilters["instance-family"])
	paymentOptionFilterSet := toLowerStringSet(standardFilters["payment-option"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	hostReservationIDFilterSet := toStringSet(requestedIDs)

	now := time.Now().UTC()

	s.mu.Lock()
	familyToHostIDs := map[string][]string{}
	for hostID, host := range s.dedicatedHosts {
		if host == nil {
			continue
		}
		family := strings.TrimSpace(host.InstanceFamily)
		if family == "" {
			family = stage116InstanceFamilyFromType(host.InstanceType)
		}
		if family == "" {
			family = "m5"
		}
		familyToHostIDs[family] = append(familyToHostIDs[family], hostID)
	}
	if len(familyToHostIDs) == 0 {
		familyToHostIDs["m5"] = []string{"h-00000000116"}
	}

	families := make([]string, 0, len(familyToHostIDs))
	for family := range familyToHostIDs {
		families = append(families, family)
	}
	sort.Strings(families)

	out := make([]HostReservation, 0, len(families))
	for idx, family := range families {
		hostIDs := append([]string(nil), familyToHostIDs[family]...)
		sort.Strings(hostIDs)
		count := int32(len(hostIDs))
		if count == 0 {
			count = 1
		}
		item := HostReservation{
			Count:             count,
			CurrencyCode:      "USD",
			Duration:          31536000,
			End:               now.AddDate(1, 0, 0),
			HostIDs:           hostIDs,
			HostReservationID: fmt.Sprintf("hr-%012d", idx+116),
			HourlyPrice:       "0.75",
			InstanceFamily:    family,
			OfferingID:        fmt.Sprintf("hro-%03d", idx+1),
			PaymentOption:     "NoUpfront",
			Start:             now,
			State:             "active",
			Tags:              map[string]string{},
			UpfrontPrice:      "0.00",
		}

		if len(hostReservationIDFilterSet) > 0 {
			if _, ok := hostReservationIDFilterSet[item.HostReservationID]; !ok {
				continue
			}
		}
		if len(instanceFamilyFilterSet) > 0 {
			if _, ok := instanceFamilyFilterSet[strings.ToLower(item.InstanceFamily)]; !ok {
				continue
			}
		}
		if len(paymentOptionFilterSet) > 0 {
			if _, ok := paymentOptionFilterSet[strings.ToLower(item.PaymentOption)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]HostReservation(nil), out[start:end]...), outputToken, nil
}

func normalizeStage116OwnerFilter(owners []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, owner := range owners {
		owner = strings.TrimSpace(strings.ToLower(owner))
		switch owner {
		case "":
			continue
		case "self":
			out[DefaultAccountID] = struct{}{}
		default:
			out[owner] = struct{}{}
		}
	}
	return out
}

func stage116InstanceFamilyFromType(instanceType string) string {
	instanceType = strings.TrimSpace(instanceType)
	if instanceType == "" {
		return ""
	}
	parts := strings.SplitN(instanceType, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func cloneStage116FlowLog(in *FlowLog) FlowLog {
	if in == nil {
		return FlowLog{}
	}
	return FlowLog{
		ClientToken:  in.ClientToken,
		FlowLogID:    in.FlowLogID,
		ResourceID:   in.ResourceID,
		ResourceType: in.ResourceType,
		Tags:         cloneStringMap(in.Tags),
		TrafficType:  in.TrafficType,
	}
}
