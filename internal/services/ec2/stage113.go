package ec2

import "strings"

type DeleteQueuedReservedInstancesError struct {
	Code    string
	Message string
}

type FailedQueuedPurchaseDeletion struct {
	Error               DeleteQueuedReservedInstancesError
	ReservedInstancesID string
}

type SuccessfulQueuedPurchaseDeletion struct {
	ReservedInstancesID string
}

func (s *Service) DeleteNetworkInsightsAnalysis(networkInsightsAnalysisID string) (string, error) {
	networkInsightsAnalysisID = strings.TrimSpace(networkInsightsAnalysisID)
	if networkInsightsAnalysisID == "" {
		return "", ErrInvalidParameter
	}
	return networkInsightsAnalysisID, nil
}

func (s *Service) DeleteNetworkInsightsPath(networkInsightsPathID string) (string, error) {
	networkInsightsPathID = strings.TrimSpace(networkInsightsPathID)
	if networkInsightsPathID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkInsightsPaths[networkInsightsPathID] == nil {
		return "", ErrNotFound
	}
	delete(s.networkInsightsPaths, networkInsightsPathID)
	return networkInsightsPathID, nil
}

func (s *Service) DeletePublicIpv4Pool(poolID string, networkBorderGroup *string) (bool, error) {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
		return false, ErrInvalidParameter
	}
	networkBorderGroupValue := strings.TrimSpace(derefString(networkBorderGroup))

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.publicIpv4Pools[poolID]
	if pool == nil {
		return false, ErrNotFound
	}
	if networkBorderGroupValue != "" && pool.NetworkBorderGroup != "" && !strings.EqualFold(pool.NetworkBorderGroup, networkBorderGroupValue) {
		return false, ErrInvalidParameter
	}

	delete(s.publicIpv4Pools, poolID)
	return true, nil
}

func (s *Service) DeleteQueuedReservedInstances(reservedInstancesIDs []string) ([]FailedQueuedPurchaseDeletion, []SuccessfulQueuedPurchaseDeletion, error) {
	reservedInstancesIDs = dedupeTrimmedStrings(reservedInstancesIDs)
	if len(reservedInstancesIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	failures := make([]FailedQueuedPurchaseDeletion, 0)
	successes := make([]SuccessfulQueuedPurchaseDeletion, 0)
	for _, reservedInstancesID := range reservedInstancesIDs {
		if !strings.HasPrefix(reservedInstancesID, "ri-") || len(reservedInstancesID) <= len("ri-") {
			failures = append(failures, FailedQueuedPurchaseDeletion{
				Error: DeleteQueuedReservedInstancesError{
					Code:    "reserved-instances-id-invalid",
					Message: "reserved instances id is invalid",
				},
				ReservedInstancesID: reservedInstancesID,
			})
			continue
		}

		listingID := "ril-" + strings.TrimPrefix(reservedInstancesID, "ri-")
		if state, ok := s.reservedInstancesListingStates[listingID]; ok && state != "queued" {
			failures = append(failures, FailedQueuedPurchaseDeletion{
				Error: DeleteQueuedReservedInstancesError{
					Code:    "reserved-instances-not-in-queued-state",
					Message: "reserved instances purchase is not in queued state",
				},
				ReservedInstancesID: reservedInstancesID,
			})
			continue
		}

		successes = append(successes, SuccessfulQueuedPurchaseDeletion{ReservedInstancesID: reservedInstancesID})
	}

	return failures, successes, nil
}

func (s *Service) DeleteSpotDatafeedSubscription() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.spotDatafeedSubscriptions = map[string]*SpotDatafeedSubscription{}
	return nil
}

func (s *Service) DeleteTrafficMirrorFilter(trafficMirrorFilterID string) (string, error) {
	trafficMirrorFilterID = strings.TrimSpace(trafficMirrorFilterID)
	if trafficMirrorFilterID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorFilters[trafficMirrorFilterID] == nil {
		return "", ErrNotFound
	}
	for _, session := range s.trafficMirrorSessions {
		if session != nil && session.TrafficMirrorFilterID == trafficMirrorFilterID {
			return "", ErrConflict
		}
	}

	delete(s.trafficMirrorFilters, trafficMirrorFilterID)
	for ruleID, rule := range s.trafficMirrorFilterRules {
		if rule != nil && rule.TrafficMirrorFilterID == trafficMirrorFilterID {
			delete(s.trafficMirrorFilterRules, ruleID)
		}
	}

	return trafficMirrorFilterID, nil
}

func (s *Service) DeleteTrafficMirrorFilterRule(trafficMirrorFilterRuleID string) (string, error) {
	trafficMirrorFilterRuleID = strings.TrimSpace(trafficMirrorFilterRuleID)
	if trafficMirrorFilterRuleID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorFilterRules[trafficMirrorFilterRuleID] == nil {
		return "", ErrNotFound
	}
	delete(s.trafficMirrorFilterRules, trafficMirrorFilterRuleID)
	return trafficMirrorFilterRuleID, nil
}

func (s *Service) DeleteTrafficMirrorSession(trafficMirrorSessionID string) (string, error) {
	trafficMirrorSessionID = strings.TrimSpace(trafficMirrorSessionID)
	if trafficMirrorSessionID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorSessions[trafficMirrorSessionID] == nil {
		return "", ErrNotFound
	}
	delete(s.trafficMirrorSessions, trafficMirrorSessionID)
	return trafficMirrorSessionID, nil
}

func (s *Service) DeleteTrafficMirrorTarget(trafficMirrorTargetID string) (string, error) {
	trafficMirrorTargetID = strings.TrimSpace(trafficMirrorTargetID)
	if trafficMirrorTargetID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorTargets[trafficMirrorTargetID] == nil {
		return "", ErrNotFound
	}
	for _, session := range s.trafficMirrorSessions {
		if session != nil && session.TrafficMirrorTargetID == trafficMirrorTargetID {
			return "", ErrConflict
		}
	}

	delete(s.trafficMirrorTargets, trafficMirrorTargetID)
	return trafficMirrorTargetID, nil
}

func (s *Service) DeprovisionByoipCidr(cidr string) (ByoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		return ByoipCidr{}, ErrNotFound
	}

	record.State = "deprovisioned"
	record.StatusMessage = "deprovisioned"
	for i := range record.AsnAssociations {
		record.AsnAssociations[i].State = "deprovisioned"
		record.AsnAssociations[i].StatusMessage = "deprovisioned"
	}

	out := cloneByoipCidr(record)
	delete(s.byoipCidrs, cidr)
	return out, nil
}
