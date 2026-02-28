package ec2

import (
	"fmt"
	"strings"
)

type SpotDatafeedSubscription struct {
	Bucket  string
	OwnerID string
	Prefix  string
	State   string
}

type StoreImageTask struct {
	Bucket       string
	ImageID      string
	ObjectKey    string
	S3ObjectTags map[string]string
}

type TrafficMirrorFilter struct {
	Description           string
	NetworkServices       []string
	Tags                  map[string]string
	TrafficMirrorFilterID string
}

type TrafficMirrorPortRange struct {
	FromPort *int32
	ToPort   *int32
}

type TrafficMirrorFilterRule struct {
	Description               string
	DestinationCidrBlock      string
	DestinationPortRange      TrafficMirrorPortRange
	Protocol                  *int32
	RuleAction                string
	RuleNumber                int32
	SourceCidrBlock           string
	SourcePortRange           TrafficMirrorPortRange
	Tags                      map[string]string
	TrafficDirection          string
	TrafficMirrorFilterID     string
	TrafficMirrorFilterRuleID string
}

type TrafficMirrorSession struct {
	Description            string
	NetworkInterfaceID     string
	OwnerID                string
	PacketLength           *int32
	SessionNumber          int32
	Tags                   map[string]string
	TrafficMirrorFilterID  string
	TrafficMirrorSessionID string
	TrafficMirrorTargetID  string
	VirtualNetworkID       *int32
}

type TrafficMirrorTarget struct {
	Description                   string
	GatewayLoadBalancerEndpointID string
	NetworkInterfaceID            string
	NetworkLoadBalancerARN        string
	OwnerID                       string
	Tags                          map[string]string
	TrafficMirrorTargetID         string
	Type                          string
}

type DeleteFleetSuccess struct {
	CurrentFleetState  string
	FleetID            string
	PreviousFleetState string
}

type DeleteFleetError struct {
	Code    string
	Message string
}

type DeleteFleetErrorItem struct {
	Error   DeleteFleetError
	FleetID string
}

func (s *Service) CreateSpotDatafeedSubscription(bucket string, prefix *string) (SpotDatafeedSubscription, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return SpotDatafeedSubscription{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub := &SpotDatafeedSubscription{
		Bucket:  bucket,
		OwnerID: DefaultAccountID,
		Prefix:  strings.TrimSpace(derefString(prefix)),
		State:   "Active",
	}
	s.spotDatafeedSubscriptions[bucket] = sub
	return cloneStage110SpotDatafeedSubscription(sub), nil
}

func (s *Service) CreateStoreImageTask(bucket string, imageID string, s3ObjectTags []Tag) (string, error) {
	bucket = strings.TrimSpace(bucket)
	imageID = strings.TrimSpace(imageID)
	if bucket == "" || imageID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.images[imageID] == nil {
		return "", ErrNotFound
	}

	objectKey := fmt.Sprintf("stored-images/%s-%s.bin", strings.TrimPrefix(imageID, "ami-"), s.nextIDLocked("sit"))
	s.storeImageTasks[objectKey] = &StoreImageTask{
		Bucket:       bucket,
		ImageID:      imageID,
		ObjectKey:    objectKey,
		S3ObjectTags: tagsToMap(normalizeEC2Tags(s3ObjectTags)),
	}
	return objectKey, nil
}

func (s *Service) CreateTrafficMirrorFilter(
	description *string,
	clientToken *string,
	tags []Tag,
) (string, TrafficMirrorFilter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("tmf-token")
	}

	filterID := s.nextIDLocked("tmf")
	filter := &TrafficMirrorFilter{
		Description:           strings.TrimSpace(derefString(description)),
		Tags:                  tagsToMap(normalizeEC2Tags(tags)),
		TrafficMirrorFilterID: filterID,
	}
	s.trafficMirrorFilters[filterID] = filter
	return token, cloneStage110TrafficMirrorFilter(filter), nil
}

func (s *Service) CreateTrafficMirrorFilterRule(
	destinationCidrBlock string,
	ruleAction string,
	ruleNumber int32,
	sourceCidrBlock string,
	trafficDirection string,
	trafficMirrorFilterID string,
	clientToken *string,
	description *string,
	destinationPortRange TrafficMirrorPortRange,
	protocol *int32,
	sourcePortRange TrafficMirrorPortRange,
	tags []Tag,
) (string, TrafficMirrorFilterRule, error) {
	destinationCidrBlock = strings.TrimSpace(destinationCidrBlock)
	ruleAction = strings.TrimSpace(ruleAction)
	sourceCidrBlock = strings.TrimSpace(sourceCidrBlock)
	trafficDirection = strings.TrimSpace(trafficDirection)
	trafficMirrorFilterID = strings.TrimSpace(trafficMirrorFilterID)
	if destinationCidrBlock == "" || ruleAction == "" || ruleNumber <= 0 || sourceCidrBlock == "" || trafficDirection == "" || trafficMirrorFilterID == "" {
		return "", TrafficMirrorFilterRule{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorFilters[trafficMirrorFilterID] == nil {
		return "", TrafficMirrorFilterRule{}, ErrNotFound
	}

	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("tmfr-token")
	}

	ruleID := s.nextIDLocked("tmfr")
	rule := &TrafficMirrorFilterRule{
		Description:               strings.TrimSpace(derefString(description)),
		DestinationCidrBlock:      destinationCidrBlock,
		DestinationPortRange:      cloneStage110TrafficMirrorPortRange(destinationPortRange),
		Protocol:                  cloneInt32Pointer(protocol),
		RuleAction:                strings.ToLower(ruleAction),
		RuleNumber:                ruleNumber,
		SourceCidrBlock:           sourceCidrBlock,
		SourcePortRange:           cloneStage110TrafficMirrorPortRange(sourcePortRange),
		Tags:                      tagsToMap(normalizeEC2Tags(tags)),
		TrafficDirection:          strings.ToLower(trafficDirection),
		TrafficMirrorFilterID:     trafficMirrorFilterID,
		TrafficMirrorFilterRuleID: ruleID,
	}
	s.trafficMirrorFilterRules[ruleID] = rule
	return token, cloneStage110TrafficMirrorFilterRule(rule), nil
}

func (s *Service) CreateTrafficMirrorSession(
	networkInterfaceID string,
	sessionNumber int32,
	trafficMirrorFilterID string,
	trafficMirrorTargetID string,
	clientToken *string,
	description *string,
	packetLength *int32,
	virtualNetworkID *int32,
	tags []Tag,
) (string, TrafficMirrorSession, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	trafficMirrorFilterID = strings.TrimSpace(trafficMirrorFilterID)
	trafficMirrorTargetID = strings.TrimSpace(trafficMirrorTargetID)
	if networkInterfaceID == "" || trafficMirrorFilterID == "" || trafficMirrorTargetID == "" || sessionNumber <= 0 {
		return "", TrafficMirrorSession{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trafficMirrorFilters[trafficMirrorFilterID] == nil || s.trafficMirrorTargets[trafficMirrorTargetID] == nil {
		return "", TrafficMirrorSession{}, ErrNotFound
	}

	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("tms-token")
	}

	sessionID := s.nextIDLocked("tms")
	session := &TrafficMirrorSession{
		Description:            strings.TrimSpace(derefString(description)),
		NetworkInterfaceID:     networkInterfaceID,
		OwnerID:                DefaultAccountID,
		PacketLength:           cloneInt32Pointer(packetLength),
		SessionNumber:          sessionNumber,
		Tags:                   tagsToMap(normalizeEC2Tags(tags)),
		TrafficMirrorFilterID:  trafficMirrorFilterID,
		TrafficMirrorSessionID: sessionID,
		TrafficMirrorTargetID:  trafficMirrorTargetID,
		VirtualNetworkID:       cloneInt32Pointer(virtualNetworkID),
	}
	s.trafficMirrorSessions[sessionID] = session
	return token, cloneStage110TrafficMirrorSession(session), nil
}

func (s *Service) CreateTrafficMirrorTarget(
	clientToken *string,
	description *string,
	gatewayLoadBalancerEndpointID *string,
	networkInterfaceID *string,
	networkLoadBalancerARN *string,
	tags []Tag,
) (string, TrafficMirrorTarget, error) {
	gatewayLoadBalancerEndpointIDValue := strings.TrimSpace(derefString(gatewayLoadBalancerEndpointID))
	networkInterfaceIDValue := strings.TrimSpace(derefString(networkInterfaceID))
	networkLoadBalancerARNValue := strings.TrimSpace(derefString(networkLoadBalancerARN))
	if gatewayLoadBalancerEndpointIDValue == "" && networkInterfaceIDValue == "" && networkLoadBalancerARNValue == "" {
		return "", TrafficMirrorTarget{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("tmt-token")
	}

	targetType := "network-interface"
	if networkLoadBalancerARNValue != "" {
		targetType = "network-load-balancer"
	} else if gatewayLoadBalancerEndpointIDValue != "" {
		targetType = "gateway-load-balancer-endpoint"
	}

	targetID := s.nextIDLocked("tmt")
	target := &TrafficMirrorTarget{
		Description:                   strings.TrimSpace(derefString(description)),
		GatewayLoadBalancerEndpointID: gatewayLoadBalancerEndpointIDValue,
		NetworkInterfaceID:            networkInterfaceIDValue,
		NetworkLoadBalancerARN:        networkLoadBalancerARNValue,
		OwnerID:                       DefaultAccountID,
		Tags:                          tagsToMap(normalizeEC2Tags(tags)),
		TrafficMirrorTargetID:         targetID,
		Type:                          targetType,
	}
	s.trafficMirrorTargets[targetID] = target
	return token, cloneStage110TrafficMirrorTarget(target), nil
}

func (s *Service) DeleteCarrierGateway(carrierGatewayID string) (CarrierGateway, error) {
	carrierGatewayID = strings.TrimSpace(carrierGatewayID)
	if carrierGatewayID == "" {
		return CarrierGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.carrierGateways[carrierGatewayID]
	if gateway == nil {
		return CarrierGateway{}, ErrNotFound
	}
	out := cloneCarrierGateway(gateway)
	delete(s.carrierGateways, carrierGatewayID)
	return out, nil
}

func (s *Service) DeleteCoipCidr(cidr string, coipPoolID string) (CoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	coipPoolID = strings.TrimSpace(coipPoolID)
	if cidr == "" || coipPoolID == "" {
		return CoipCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := coipPoolID + "|" + cidr
	coipCidr := s.coipCidrs[key]
	if coipCidr == nil {
		return CoipCidr{}, ErrNotFound
	}
	out := cloneCoipCidr(coipCidr)
	delete(s.coipCidrs, key)
	return out, nil
}

func (s *Service) DeleteCoipPool(coipPoolID string) (CoipPool, error) {
	coipPoolID = strings.TrimSpace(coipPoolID)
	if coipPoolID == "" {
		return CoipPool{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.coipPools[coipPoolID]
	if pool == nil {
		return CoipPool{}, ErrNotFound
	}
	out := cloneStage107CoipPool(pool)
	delete(s.coipPools, coipPoolID)
	prefix := coipPoolID + "|"
	for key := range s.coipCidrs {
		if strings.HasPrefix(key, prefix) {
			delete(s.coipCidrs, key)
		}
	}
	return out, nil
}

func (s *Service) DeleteFleets(fleetIDs []string, terminateInstances bool) ([]DeleteFleetSuccess, []DeleteFleetErrorItem, error) {
	if len(fleetIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}
	_ = terminateInstances

	s.mu.Lock()
	defer s.mu.Unlock()

	successes := make([]DeleteFleetSuccess, 0, len(fleetIDs))
	errors := make([]DeleteFleetErrorItem, 0)
	for _, fleetID := range fleetIDs {
		fleetID = strings.TrimSpace(fleetID)
		if fleetID == "" {
			errors = append(errors, DeleteFleetErrorItem{
				Error:   DeleteFleetError{Code: "fleetIdMalformed", Message: "fleet id is required"},
				FleetID: fleetID,
			})
			continue
		}

		if s.fleets[fleetID] == nil {
			errors = append(errors, DeleteFleetErrorItem{
				Error:   DeleteFleetError{Code: "fleetIdDoesNotExist", Message: "fleet not found"},
				FleetID: fleetID,
			})
			continue
		}

		delete(s.fleets, fleetID)
		successes = append(successes, DeleteFleetSuccess{
			CurrentFleetState:  "deleted",
			FleetID:            fleetID,
			PreviousFleetState: "active",
		})
	}
	return successes, errors, nil
}

func cloneStage110SpotDatafeedSubscription(in *SpotDatafeedSubscription) SpotDatafeedSubscription {
	if in == nil {
		return SpotDatafeedSubscription{}
	}
	return SpotDatafeedSubscription{
		Bucket:  in.Bucket,
		OwnerID: in.OwnerID,
		Prefix:  in.Prefix,
		State:   in.State,
	}
}

func cloneStage110TrafficMirrorFilter(in *TrafficMirrorFilter) TrafficMirrorFilter {
	if in == nil {
		return TrafficMirrorFilter{}
	}
	return TrafficMirrorFilter{
		Description:           in.Description,
		NetworkServices:       append([]string(nil), in.NetworkServices...),
		Tags:                  cloneStringMap(in.Tags),
		TrafficMirrorFilterID: in.TrafficMirrorFilterID,
	}
}

func cloneStage110TrafficMirrorPortRange(in TrafficMirrorPortRange) TrafficMirrorPortRange {
	return TrafficMirrorPortRange{
		FromPort: cloneInt32Pointer(in.FromPort),
		ToPort:   cloneInt32Pointer(in.ToPort),
	}
}

func cloneStage110TrafficMirrorFilterRule(in *TrafficMirrorFilterRule) TrafficMirrorFilterRule {
	if in == nil {
		return TrafficMirrorFilterRule{}
	}
	return TrafficMirrorFilterRule{
		Description:               in.Description,
		DestinationCidrBlock:      in.DestinationCidrBlock,
		DestinationPortRange:      cloneStage110TrafficMirrorPortRange(in.DestinationPortRange),
		Protocol:                  cloneInt32Pointer(in.Protocol),
		RuleAction:                in.RuleAction,
		RuleNumber:                in.RuleNumber,
		SourceCidrBlock:           in.SourceCidrBlock,
		SourcePortRange:           cloneStage110TrafficMirrorPortRange(in.SourcePortRange),
		Tags:                      cloneStringMap(in.Tags),
		TrafficDirection:          in.TrafficDirection,
		TrafficMirrorFilterID:     in.TrafficMirrorFilterID,
		TrafficMirrorFilterRuleID: in.TrafficMirrorFilterRuleID,
	}
}

func cloneStage110TrafficMirrorSession(in *TrafficMirrorSession) TrafficMirrorSession {
	if in == nil {
		return TrafficMirrorSession{}
	}
	return TrafficMirrorSession{
		Description:            in.Description,
		NetworkInterfaceID:     in.NetworkInterfaceID,
		OwnerID:                in.OwnerID,
		PacketLength:           cloneInt32Pointer(in.PacketLength),
		SessionNumber:          in.SessionNumber,
		Tags:                   cloneStringMap(in.Tags),
		TrafficMirrorFilterID:  in.TrafficMirrorFilterID,
		TrafficMirrorSessionID: in.TrafficMirrorSessionID,
		TrafficMirrorTargetID:  in.TrafficMirrorTargetID,
		VirtualNetworkID:       cloneInt32Pointer(in.VirtualNetworkID),
	}
}

func cloneStage110TrafficMirrorTarget(in *TrafficMirrorTarget) TrafficMirrorTarget {
	if in == nil {
		return TrafficMirrorTarget{}
	}
	return TrafficMirrorTarget{
		Description:                   in.Description,
		GatewayLoadBalancerEndpointID: in.GatewayLoadBalancerEndpointID,
		NetworkInterfaceID:            in.NetworkInterfaceID,
		NetworkLoadBalancerARN:        in.NetworkLoadBalancerARN,
		OwnerID:                       in.OwnerID,
		Tags:                          cloneStringMap(in.Tags),
		TrafficMirrorTargetID:         in.TrafficMirrorTargetID,
		Type:                          in.Type,
	}
}
