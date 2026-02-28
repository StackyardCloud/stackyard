package ec2

import (
	"sort"
	"strings"
	"time"
)

type CapacityBlock struct {
	AvailabilityZone       string
	AvailabilityZoneID     string
	CapacityBlockID        string
	CapacityReservationIDs []string
	CreateDate             time.Time
	EndDate                time.Time
	StartDate              time.Time
	State                  string
	Tags                   map[string]string
	UltraserverType        string
}

type CapacityReservationBillingInfo struct {
	AvailabilityZone   string
	AvailabilityZoneID string
	InstanceType       string
	Tenancy            string
}

type CapacityReservationBillingRequest struct {
	CapacityReservationID           string
	CapacityReservationInfo         *CapacityReservationBillingInfo
	LastUpdateTime                  time.Time
	RequestedBy                     string
	Status                          string
	StatusMessage                   string
	UnusedReservationBillingOwnerID string
}

type ConversionTask struct {
	ConversionTaskID string
	ExpirationTime   string
	State            string
	StatusMessage    string
	Tags             map[string]string
}

type ElasticGpu struct {
	AvailabilityZone string
	ElasticGpuHealth string
	ElasticGpuID     string
	ElasticGpuState  string
	ElasticGpuType   string
	InstanceID       string
	Tags             map[string]string
}

type ExportTaskS3Location struct {
	S3Bucket string
	S3Prefix string
}

type ExportImageTask struct {
	Description       string
	DiskImageFormat   string
	ExportImageTaskID string
	ImageID           string
	Progress          string
	RoleName          string
	S3ExportLocation  ExportTaskS3Location
	Status            string
	StatusMessage     string
	Tags              map[string]string
}

func (s *Service) DescribeCapacityBlocks(capacityBlockIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityBlock, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(capacityBlockIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	idFilterSet := toStringSet(standardFilters["capacity-block-id"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	ultraserverTypeFilterSet := toLowerStringSet(standardFilters["ultraserver-type"])
	createDateFilterSet := toStringSet(standardFilters["create-date"])
	startDateFilterSet := toStringSet(standardFilters["start-date"])
	endDateFilterSet := toStringSet(standardFilters["end-date"])

	now := time.Now().UTC()

	s.mu.Lock()
	itemsByID := map[string]CapacityBlock{}
	reservationIDs := make([]string, 0, len(s.capacityReservations))
	for reservationID := range s.capacityReservations {
		reservationIDs = append(reservationIDs, reservationID)
	}
	sort.Strings(reservationIDs)

	for _, reservationID := range reservationIDs {
		reservation := s.capacityReservations[reservationID]
		if reservation == nil {
			continue
		}
		capacityBlockID := "cb-" + strings.TrimPrefix(reservationID, "cr-")
		state := "active"
		if s.cancelledCapacityReservations[reservationID] {
			state = "cancelled"
		}
		startDate := reservation.CreateDate.UTC()
		if startDate.IsZero() {
			startDate = now
		}
		itemsByID[capacityBlockID] = CapacityBlock{
			AvailabilityZone:       reservation.AvailabilityZone,
			AvailabilityZoneID:     reservation.AvailabilityZoneID,
			CapacityBlockID:        capacityBlockID,
			CapacityReservationIDs: []string{reservationID},
			CreateDate:             startDate,
			EndDate:                startDate.Add(24 * time.Hour),
			StartDate:              startDate,
			State:                  state,
			Tags:                   cloneStringMap(reservation.Tags),
			UltraserverType:        "instances",
		}
	}

	if len(requestedIDs) == 0 && len(itemsByID) == 0 {
		requestedIDs = []string{"cb-0000000000000115"}
	}
	for _, capacityBlockID := range requestedIDs {
		if _, ok := itemsByID[capacityBlockID]; ok {
			continue
		}
		itemsByID[capacityBlockID] = stage115DefaultCapacityBlock(capacityBlockID, now)
	}
	s.mu.Unlock()

	candidateIDs := requestedIDs
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(itemsByID))
		for capacityBlockID := range itemsByID {
			candidateIDs = append(candidateIDs, capacityBlockID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]CapacityBlock, 0, len(candidateIDs))
	for _, capacityBlockID := range candidateIDs {
		item, ok := itemsByID[capacityBlockID]
		if !ok {
			item = stage115DefaultCapacityBlock(capacityBlockID, now)
		}
		if len(idFilterSet) > 0 {
			if _, ok := idFilterSet[item.CapacityBlockID]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if len(ultraserverTypeFilterSet) > 0 {
			if _, ok := ultraserverTypeFilterSet[strings.ToLower(item.UltraserverType)]; !ok {
				continue
			}
		}
		if len(createDateFilterSet) > 0 {
			if _, ok := createDateFilterSet[item.CreateDate.UTC().Format(time.RFC3339)]; !ok {
				continue
			}
		}
		if len(startDateFilterSet) > 0 {
			if _, ok := startDateFilterSet[item.StartDate.UTC().Format(time.RFC3339)]; !ok {
				continue
			}
		}
		if len(endDateFilterSet) > 0 {
			if _, ok := endDateFilterSet[item.EndDate.UTC().Format(time.RFC3339)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		out = append(out, cloneStage115CapacityBlock(item))
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityBlock(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityReservationBillingRequests(role string, capacityReservationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityReservationBillingRequest, *string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "odcr-owner" && role != "unused-reservation-billing-owner" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	statusFilterSet := toLowerStringSet(standardFilters["status"])
	requestedByFilterSet := toStringSet(standardFilters["requested-by"])
	unusedOwnerFilterSet := toStringSet(standardFilters["unused-reservation-billing-owner"])

	requestedIDs := dedupeTrimmedStrings(capacityReservationIDs)

	now := time.Now().UTC()

	s.mu.Lock()
	if len(requestedIDs) == 0 {
		if len(s.capacityReservationBillingOwners) > 0 {
			for reservationID := range s.capacityReservationBillingOwners {
				requestedIDs = append(requestedIDs, reservationID)
			}
		} else {
			for reservationID := range s.capacityReservations {
				requestedIDs = append(requestedIDs, reservationID)
			}
		}
		requestedIDs = dedupeTrimmedStrings(requestedIDs)
		sort.Strings(requestedIDs)
	}
	if len(requestedIDs) == 0 {
		requestedIDs = []string{"cr-0000000000000115"}
	}

	out := make([]CapacityReservationBillingRequest, 0, len(requestedIDs))
	for _, reservationID := range requestedIDs {
		reservation := s.capacityReservations[reservationID]
		unusedOwnerID := strings.TrimSpace(s.capacityReservationBillingOwners[reservationID])
		if unusedOwnerID == "" {
			unusedOwnerID = DefaultAccountID
		}

		if role == "unused-reservation-billing-owner" && unusedOwnerID != DefaultAccountID {
			continue
		}

		info := &CapacityReservationBillingInfo{
			AvailabilityZone:   "us-east-1a",
			AvailabilityZoneID: "use1-az1",
			InstanceType:       "t3.micro",
			Tenancy:            "default",
		}
		if reservation != nil {
			if strings.TrimSpace(reservation.AvailabilityZone) != "" {
				info.AvailabilityZone = reservation.AvailabilityZone
			}
			if strings.TrimSpace(reservation.AvailabilityZoneID) != "" {
				info.AvailabilityZoneID = reservation.AvailabilityZoneID
			}
			if strings.TrimSpace(reservation.InstanceType) != "" {
				info.InstanceType = reservation.InstanceType
			}
			if strings.TrimSpace(reservation.Tenancy) != "" {
				info.Tenancy = reservation.Tenancy
			}
		}

		item := CapacityReservationBillingRequest{
			CapacityReservationID:           reservationID,
			CapacityReservationInfo:         info,
			LastUpdateTime:                  now,
			RequestedBy:                     DefaultAccountID,
			Status:                          "pending",
			StatusMessage:                   "billing assignment pending",
			UnusedReservationBillingOwnerID: unusedOwnerID,
		}

		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(item.Status)]; !ok {
				continue
			}
		}
		if len(requestedByFilterSet) > 0 {
			if _, ok := requestedByFilterSet[item.RequestedBy]; !ok {
				continue
			}
		}
		if len(unusedOwnerFilterSet) > 0 {
			if _, ok := unusedOwnerFilterSet[item.UnusedReservationBillingOwnerID]; !ok {
				continue
			}
		}

		out = append(out, cloneStage115CapacityReservationBillingRequest(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityReservationBillingRequest(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityReservationFleets(capacityReservationFleetIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityReservationFleet, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	fleetIDSet := toStringSet(dedupeTrimmedStrings(capacityReservationFleetIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	instanceMatchCriteriaFilterSet := toLowerStringSet(standardFilters["instance-match-criteria"])
	tenancyFilterSet := toLowerStringSet(standardFilters["tenancy"])
	allocationStrategyFilterSet := toLowerStringSet(standardFilters["allocation-strategy"])

	s.mu.Lock()
	fleetIDs := make([]string, 0, len(s.capacityReservationFleets))
	for fleetID := range s.capacityReservationFleets {
		fleetIDs = append(fleetIDs, fleetID)
	}
	sort.Strings(fleetIDs)

	out := make([]CapacityReservationFleet, 0, len(fleetIDs))
	for _, fleetID := range fleetIDs {
		fleet := s.capacityReservationFleets[fleetID]
		if fleet == nil {
			continue
		}
		if len(fleetIDSet) > 0 {
			if _, ok := fleetIDSet[fleetID]; !ok {
				continue
			}
		}

		item := cloneCapacityReservationFleet(fleet)
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if len(instanceMatchCriteriaFilterSet) > 0 {
			if _, ok := instanceMatchCriteriaFilterSet[strings.ToLower(item.InstanceMatchCriteria)]; !ok {
				continue
			}
		}
		if len(tenancyFilterSet) > 0 {
			if _, ok := tenancyFilterSet[strings.ToLower(item.Tenancy)]; !ok {
				continue
			}
		}
		if len(allocationStrategyFilterSet) > 0 {
			if _, ok := allocationStrategyFilterSet[strings.ToLower(item.AllocationStrategy)]; !ok {
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
	return append([]CapacityReservationFleet(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityReservations(capacityReservationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityReservation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	reservationIDSet := toStringSet(dedupeTrimmedStrings(capacityReservationIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	instancePlatformFilterSet := toLowerStringSet(standardFilters["instance-platform"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	tenancyFilterSet := toLowerStringSet(standardFilters["tenancy"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	instanceMatchCriteriaFilterSet := toLowerStringSet(standardFilters["instance-match-criteria"])

	s.mu.Lock()
	reservationIDs := make([]string, 0, len(s.capacityReservations))
	for reservationID := range s.capacityReservations {
		reservationIDs = append(reservationIDs, reservationID)
	}
	sort.Strings(reservationIDs)

	out := make([]CapacityReservation, 0, len(reservationIDs))
	for _, reservationID := range reservationIDs {
		reservation := s.capacityReservations[reservationID]
		if reservation == nil {
			continue
		}
		if len(reservationIDSet) > 0 {
			if _, ok := reservationIDSet[reservationID]; !ok {
				continue
			}
		}

		item := cloneCapacityReservation(reservation)
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if len(instancePlatformFilterSet) > 0 {
			if _, ok := instancePlatformFilterSet[strings.ToLower(item.InstancePlatform)]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(tenancyFilterSet) > 0 {
			if _, ok := tenancyFilterSet[strings.ToLower(item.Tenancy)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if len(instanceMatchCriteriaFilterSet) > 0 {
			if _, ok := instanceMatchCriteriaFilterSet[strings.ToLower(item.InstanceMatchCriteria)]; !ok {
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
	return append([]CapacityReservation(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCarrierGateways(carrierGatewayIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CarrierGateway, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	carrierGatewayIDSet := toStringSet(dedupeTrimmedStrings(carrierGatewayIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	carrierGatewayIDFilterSet := toStringSet(standardFilters["carrier-gateway-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])

	s.mu.Lock()
	gatewayIDs := make([]string, 0, len(s.carrierGateways))
	for gatewayID := range s.carrierGateways {
		gatewayIDs = append(gatewayIDs, gatewayID)
	}
	sort.Strings(gatewayIDs)

	out := make([]CarrierGateway, 0, len(gatewayIDs))
	for _, gatewayID := range gatewayIDs {
		gateway := s.carrierGateways[gatewayID]
		if gateway == nil {
			continue
		}
		if len(carrierGatewayIDSet) > 0 {
			if _, ok := carrierGatewayIDSet[gatewayID]; !ok {
				continue
			}
		}

		item := cloneCarrierGateway(gateway)
		if len(carrierGatewayIDFilterSet) > 0 {
			if _, ok := carrierGatewayIDFilterSet[item.ID]; !ok {
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
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[item.VpcID]; !ok {
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
	return append([]CarrierGateway(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCoipPools(poolIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CoipPool, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	poolIDSet := toStringSet(dedupeTrimmedStrings(poolIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	poolIDFilterSet := toStringSet(append(standardFilters["coip-pool.pool-id"], standardFilters["pool-id"]...))
	localGatewayRouteTableIDFilterSet := toStringSet(standardFilters["coip-pool.local-gateway-route-table-id"])

	s.mu.Lock()
	coipPoolIDs := make([]string, 0, len(s.coipPools))
	for poolID := range s.coipPools {
		coipPoolIDs = append(coipPoolIDs, poolID)
	}
	sort.Strings(coipPoolIDs)

	out := make([]CoipPool, 0, len(coipPoolIDs))
	for _, poolID := range coipPoolIDs {
		pool := s.coipPools[poolID]
		if pool == nil {
			continue
		}
		if len(poolIDSet) > 0 {
			if _, ok := poolIDSet[poolID]; !ok {
				continue
			}
		}

		item := cloneStage107CoipPool(pool)
		if len(poolIDFilterSet) > 0 {
			if _, ok := poolIDFilterSet[item.PoolID]; !ok {
				continue
			}
		}
		if len(localGatewayRouteTableIDFilterSet) > 0 {
			if _, ok := localGatewayRouteTableIDFilterSet[item.LocalGatewayRouteTableID]; !ok {
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
	return append([]CoipPool(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeConversionTasks(conversionTaskIDs []string) []ConversionTask {
	requestedIDs := dedupeTrimmedStrings(conversionTaskIDs)

	s.mu.Lock()
	if len(requestedIDs) == 0 {
		for conversionTaskID := range s.conversionTaskStates {
			requestedIDs = append(requestedIDs, conversionTaskID)
		}
		requestedIDs = dedupeTrimmedStrings(requestedIDs)
		sort.Strings(requestedIDs)
	}

	out := make([]ConversionTask, 0, len(requestedIDs))
	for _, conversionTaskID := range requestedIDs {
		if conversionTaskID == "" {
			continue
		}
		state := strings.TrimSpace(s.conversionTaskStates[conversionTaskID])
		if state == "" {
			state = "active"
		}
		statusMessage := "conversion task in progress"
		if strings.EqualFold(state, "cancelled") {
			statusMessage = "conversion task cancelled"
		}
		if reason := strings.TrimSpace(s.conversionTaskCancelReasons[conversionTaskID]); reason != "" {
			statusMessage = reason
		}

		out = append(out, ConversionTask{
			ConversionTaskID: conversionTaskID,
			ExpirationTime:   time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			State:            state,
			StatusMessage:    statusMessage,
			Tags:             map[string]string{},
		})
	}
	s.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].ConversionTaskID < out[j].ConversionTaskID
	})
	return out
}

func (s *Service) DescribeElasticGpus(elasticGpuIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ElasticGpu, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	elasticGpuIDFilterSet := toStringSet(standardFilters["elastic-gpu-id"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	healthFilterSet := toLowerStringSet(standardFilters["elastic-gpu-health"])
	stateFilterSet := toLowerStringSet(standardFilters["elastic-gpu-state"])
	typeFilterSet := toLowerStringSet(standardFilters["elastic-gpu-type"])
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])

	requestedIDs := dedupeTrimmedStrings(elasticGpuIDs)
	if len(requestedIDs) == 0 && len(elasticGpuIDFilterSet) > 0 {
		for elasticGpuID := range elasticGpuIDFilterSet {
			requestedIDs = append(requestedIDs, elasticGpuID)
		}
		sort.Strings(requestedIDs)
	}
	if len(requestedIDs) == 0 {
		requestedIDs = []string{"egpu-0000000000000115"}
	}

	instanceID := "i-0000000000000115"
	s.mu.Lock()
	instanceIDs := make([]string, 0, len(s.instances))
	for id := range s.instances {
		instanceIDs = append(instanceIDs, id)
	}
	sort.Strings(instanceIDs)
	if len(instanceIDs) > 0 {
		instanceID = instanceIDs[0]
	}
	s.mu.Unlock()

	out := make([]ElasticGpu, 0, len(requestedIDs))
	for idx, elasticGpuID := range requestedIDs {
		if elasticGpuID == "" {
			continue
		}
		candidateInstanceID := instanceID
		if idx > 0 {
			candidateInstanceID = "i-" + strings.TrimPrefix(elasticGpuID, "egpu-")
		}
		item := ElasticGpu{
			AvailabilityZone: "us-east-1a",
			ElasticGpuHealth: "OK",
			ElasticGpuID:     elasticGpuID,
			ElasticGpuState:  "ATTACHED",
			ElasticGpuType:   "eg1.medium",
			InstanceID:       candidateInstanceID,
			Tags:             map[string]string{"Name": "stackyard-elastic-gpu"},
		}

		if len(elasticGpuIDFilterSet) > 0 {
			if _, ok := elasticGpuIDFilterSet[item.ElasticGpuID]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(healthFilterSet) > 0 {
			if _, ok := healthFilterSet[strings.ToLower(item.ElasticGpuHealth)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.ElasticGpuState)]; !ok {
				continue
			}
		}
		if len(typeFilterSet) > 0 {
			if _, ok := typeFilterSet[strings.ToLower(item.ElasticGpuType)]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage115ElasticGpu(item))
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ElasticGpu(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeExportImageTasks(exportImageTaskIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ExportImageTask, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	exportImageTaskIDFilterSet := toStringSet(standardFilters["export-image-task-id"])
	stateFilterSet := toLowerStringSet(standardFilters["task-state"])

	requestedIDs := dedupeTrimmedStrings(exportImageTaskIDs)
	if len(requestedIDs) == 0 && len(exportImageTaskIDFilterSet) > 0 {
		for exportImageTaskID := range exportImageTaskIDFilterSet {
			requestedIDs = append(requestedIDs, exportImageTaskID)
		}
		sort.Strings(requestedIDs)
	}
	if len(requestedIDs) == 0 {
		requestedIDs = []string{"export-ami-0000000000000115"}
	}

	out := make([]ExportImageTask, 0, len(requestedIDs))
	for _, exportImageTaskID := range requestedIDs {
		if exportImageTaskID == "" {
			continue
		}
		imageID := "ami-" + strings.TrimPrefix(strings.TrimPrefix(exportImageTaskID, "export-"), "ami-")
		if imageID == "ami-" {
			imageID = "ami-0000000000000115"
		}
		item := ExportImageTask{
			Description:       "stackyard export image task",
			ExportImageTaskID: exportImageTaskID,
			ImageID:           imageID,
			Progress:          "100",
			S3ExportLocation: ExportTaskS3Location{
				S3Bucket: "stackyard-export-bucket",
				S3Prefix: "exports/",
			},
			Status:        "completed",
			StatusMessage: "export image task completed",
			Tags:          map[string]string{},
		}

		if len(exportImageTaskIDFilterSet) > 0 {
			if _, ok := exportImageTaskIDFilterSet[item.ExportImageTaskID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.Status)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage115ExportImageTask(item))
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ExportImageTask(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeExportTasks(exportTaskIDs []string, filters map[string][]string) []InstanceExportTask {
	requestedIDs := dedupeTrimmedStrings(exportTaskIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	exportTaskIDFilterSet := toStringSet(standardFilters["export-task-id"])
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	if len(requestedIDs) == 0 {
		for exportTaskID := range s.instanceExportTasks {
			requestedIDs = append(requestedIDs, exportTaskID)
		}
		for exportTaskID := range s.cancelledExportTasks {
			requestedIDs = append(requestedIDs, exportTaskID)
		}
		requestedIDs = dedupeTrimmedStrings(requestedIDs)
		sort.Strings(requestedIDs)
	}

	out := make([]InstanceExportTask, 0, len(requestedIDs))
	for _, exportTaskID := range requestedIDs {
		if exportTaskID == "" {
			continue
		}

		item := InstanceExportTask{
			Description:       "stackyard export task",
			ExportTaskID:      exportTaskID,
			InstanceID:        "i-0000000000000115",
			S3Bucket:          "stackyard-export-bucket",
			S3Key:             "exports/" + exportTaskID + ".vmdk",
			S3Prefix:          "exports",
			State:             "active",
			StatusMessage:     "export task in progress",
			Tags:              map[string]string{},
			TargetEnvironment: "vmware",
			ContainerFormat:   "ova",
			DiskImageFormat:   "vmdk",
		}
		if existing := s.instanceExportTasks[exportTaskID]; existing != nil {
			item = cloneStage107InstanceExportTask(existing)
		}
		if s.cancelledExportTasks[exportTaskID] {
			item.State = "cancelled"
			if !strings.Contains(strings.ToLower(item.StatusMessage), "cancel") {
				item.StatusMessage = "export task cancelled"
			}
		}
		if strings.TrimSpace(item.State) == "" {
			item.State = "active"
		}
		if strings.TrimSpace(item.StatusMessage) == "" {
			item.StatusMessage = "export task in progress"
		}

		if len(exportTaskIDFilterSet) > 0 {
			if _, ok := exportTaskIDFilterSet[item.ExportTaskID]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
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

		out = append(out, cloneStage107InstanceExportTask(&item))
	}
	s.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].ExportTaskID < out[j].ExportTaskID
	})
	return out
}

func stage115DefaultCapacityBlock(capacityBlockID string, now time.Time) CapacityBlock {
	reservationID := "cr-" + strings.TrimPrefix(capacityBlockID, "cb-")
	if reservationID == "cr-" {
		reservationID = "cr-0000000000000115"
	}
	return CapacityBlock{
		AvailabilityZone:       "us-east-1a",
		AvailabilityZoneID:     "use1-az1",
		CapacityBlockID:        capacityBlockID,
		CapacityReservationIDs: []string{reservationID},
		CreateDate:             now,
		EndDate:                now.Add(24 * time.Hour),
		StartDate:              now,
		State:                  "active",
		Tags:                   map[string]string{},
		UltraserverType:        "instances",
	}
}

func cloneStage115CapacityBlock(in CapacityBlock) CapacityBlock {
	return CapacityBlock{
		AvailabilityZone:       in.AvailabilityZone,
		AvailabilityZoneID:     in.AvailabilityZoneID,
		CapacityBlockID:        in.CapacityBlockID,
		CapacityReservationIDs: append([]string(nil), in.CapacityReservationIDs...),
		CreateDate:             in.CreateDate,
		EndDate:                in.EndDate,
		StartDate:              in.StartDate,
		State:                  in.State,
		Tags:                   cloneStringMap(in.Tags),
		UltraserverType:        in.UltraserverType,
	}
}

func cloneStage115CapacityReservationBillingRequest(in CapacityReservationBillingRequest) CapacityReservationBillingRequest {
	out := CapacityReservationBillingRequest{
		CapacityReservationID:           in.CapacityReservationID,
		LastUpdateTime:                  in.LastUpdateTime,
		RequestedBy:                     in.RequestedBy,
		Status:                          in.Status,
		StatusMessage:                   in.StatusMessage,
		UnusedReservationBillingOwnerID: in.UnusedReservationBillingOwnerID,
	}
	if in.CapacityReservationInfo != nil {
		out.CapacityReservationInfo = &CapacityReservationBillingInfo{
			AvailabilityZone:   in.CapacityReservationInfo.AvailabilityZone,
			AvailabilityZoneID: in.CapacityReservationInfo.AvailabilityZoneID,
			InstanceType:       in.CapacityReservationInfo.InstanceType,
			Tenancy:            in.CapacityReservationInfo.Tenancy,
		}
	}
	return out
}

func cloneStage115ElasticGpu(in ElasticGpu) ElasticGpu {
	return ElasticGpu{
		AvailabilityZone: in.AvailabilityZone,
		ElasticGpuHealth: in.ElasticGpuHealth,
		ElasticGpuID:     in.ElasticGpuID,
		ElasticGpuState:  in.ElasticGpuState,
		ElasticGpuType:   in.ElasticGpuType,
		InstanceID:       in.InstanceID,
		Tags:             cloneStringMap(in.Tags),
	}
}

func cloneStage115ExportImageTask(in ExportImageTask) ExportImageTask {
	return ExportImageTask{
		Description:       in.Description,
		ExportImageTaskID: in.ExportImageTaskID,
		ImageID:           in.ImageID,
		Progress:          in.Progress,
		S3ExportLocation: ExportTaskS3Location{
			S3Bucket: in.S3ExportLocation.S3Bucket,
			S3Prefix: in.S3ExportLocation.S3Prefix,
		},
		Status:        in.Status,
		StatusMessage: in.StatusMessage,
		Tags:          cloneStringMap(in.Tags),
	}
}
