package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage130Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyReservedInstances":
		targetConfigurations, hasTargetConfigurations, ok := parseEC2Stage130ReservedInstancesConfigurations(r.Form)
		if !ok || !hasTargetConfigurations {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		modificationID, err := s.ec2.ModifyReservedInstances(
			parseEC2MembersOrItemList(r.Form, "ReservedInstancesId"),
			targetConfigurations,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ModifyReservedInstancesResponse{
			XMLName:                         xml.Name{Local: "ModifyReservedInstancesResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			ReservedInstancesModificationID: modificationID,
		})
		return true
	case "ModifySnapshotAttribute":
		addPermissions, hasAddPermissions, ok := parseEC2Stage130SnapshotCreateVolumePermissionList(r.Form, "CreateVolumePermission.Add.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		removePermissions, hasRemovePermissions, ok := parseEC2Stage130SnapshotCreateVolumePermissionList(r.Form, "CreateVolumePermission.Remove.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if (hasAddPermissions || hasRemovePermissions) && strings.TrimSpace(r.Form.Get("OperationType")) == "" {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		err := s.ec2.ModifySnapshotAttribute(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
			strings.TrimSpace(r.Form.Get("OperationType")),
			parseEC2MembersOrItemList(r.Form, "UserGroup"),
			parseEC2MembersOrItemList(r.Form, "UserId"),
			addPermissions,
			removePermissions,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifySnapshotAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "ModifySnapshotTier":
		snapshotID, tieringStartTime, err := s.ec2.ModifySnapshotTier(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			strings.TrimSpace(r.Form.Get("StorageTier")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ModifySnapshotTierResponse{
			XMLName:          xml.Name{Local: "ModifySnapshotTierResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			SnapshotID:       snapshotID,
			TieringStartTime: tieringStartTime.UTC().Format(time.RFC3339),
		})
		return true
	case "ModifySpotFleetRequest":
		onDemandTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("OnDemandTargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		targetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		okResult, err := s.ec2.ModifySpotFleetRequest(
			strings.TrimSpace(r.Form.Get("SpotFleetRequestId")),
			parseEC2OptionalString(r.Form.Get("Context")),
			strings.TrimSpace(r.Form.Get("ExcessCapacityTerminationPolicy")),
			hasEC2PrefixedField(r.Form, "LaunchTemplateConfig."),
			onDemandTargetCapacity,
			targetCapacity,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifySpotFleetRequestResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyTrafficMirrorFilterNetworkServices":
		filter, err := s.ec2.ModifyTrafficMirrorFilterNetworkServices(
			strings.TrimSpace(r.Form.Get("TrafficMirrorFilterId")),
			parseEC2MembersOrItemList(r.Form, "AddNetworkService"),
			parseEC2MembersOrItemList(r.Form, "RemoveNetworkService"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ModifyTrafficMirrorFilterNetworkServicesResponse{
			XMLName:             xml.Name{Local: "ModifyTrafficMirrorFilterNetworkServicesResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			TrafficMirrorFilter: ec2Stage110TrafficMirrorFilterItemFrom(filter),
		})
		return true
	case "ModifyTrafficMirrorFilterRule":
		destinationPortRange, ok := parseEC2Stage130OptionalTrafficMirrorPortRange(r.Form, "DestinationPortRange.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sourcePortRange, ok := parseEC2Stage130OptionalTrafficMirrorPortRange(r.Form, "SourcePortRange.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		protocol, ok := parseEC2OptionalInt32(r.Form.Get("Protocol"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ruleNumber, ok := parseEC2OptionalInt32(r.Form.Get("RuleNumber"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		rule, err := s.ec2.ModifyTrafficMirrorFilterRule(
			strings.TrimSpace(r.Form.Get("TrafficMirrorFilterRuleId")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			ec2OptionalStringPointerFromForm(r.Form, "DestinationCidrBlock"),
			destinationPortRange,
			protocol,
			parseEC2MembersOrItemList(r.Form, "RemoveField"),
			strings.TrimSpace(r.Form.Get("RuleAction")),
			ruleNumber,
			ec2OptionalStringPointerFromForm(r.Form, "SourceCidrBlock"),
			sourcePortRange,
			strings.TrimSpace(r.Form.Get("TrafficDirection")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ModifyTrafficMirrorFilterRuleResponse{
			XMLName:                 xml.Name{Local: "ModifyTrafficMirrorFilterRuleResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			TrafficMirrorFilterRule: ec2Stage110TrafficMirrorFilterRuleItemFrom(rule),
		})
		return true
	case "ModifyTrafficMirrorSession":
		packetLength, ok := parseEC2OptionalInt32(r.Form.Get("PacketLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sessionNumber, ok := parseEC2OptionalInt32(r.Form.Get("SessionNumber"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		virtualNetworkID, ok := parseEC2OptionalInt32(r.Form.Get("VirtualNetworkId"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		session, err := s.ec2.ModifyTrafficMirrorSession(
			strings.TrimSpace(r.Form.Get("TrafficMirrorSessionId")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			packetLength,
			parseEC2MembersOrItemList(r.Form, "RemoveField"),
			sessionNumber,
			ec2OptionalStringPointerFromForm(r.Form, "TrafficMirrorFilterId"),
			ec2OptionalStringPointerFromForm(r.Form, "TrafficMirrorTargetId"),
			virtualNetworkID,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ModifyTrafficMirrorSessionResponse{
			XMLName:              xml.Name{Local: "ModifyTrafficMirrorSessionResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			TrafficMirrorSession: ec2Stage110TrafficMirrorSessionItemFrom(session),
		})
		return true
	case "MoveByoipCidrToIpam":
		byoipCidr, err := s.ec2.MoveByoipCidrToIpam(
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			strings.TrimSpace(r.Form.Get("IpamPoolOwner")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130MoveByoipCidrToIpamResponse{
			XMLName:   xml.Name{Local: "MoveByoipCidrToIpamResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoipCidr: ec2ByoipCidrItemFrom(byoipCidr),
		})
		return true
	case "MoveCapacityReservationInstances":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok || instanceCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sourceReservation, destinationReservation, movedCount, err := s.ec2.MoveCapacityReservationInstances(
			strings.TrimSpace(r.Form.Get("DestinationCapacityReservationId")),
			*instanceCount,
			strings.TrimSpace(r.Form.Get("SourceCapacityReservationId")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130MoveCapacityReservationInstancesResponse{
			XMLName:                        xml.Name{Local: "MoveCapacityReservationInstancesResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			DestinationCapacityReservation: ec2Stage102CapacityReservationItemFrom(destinationReservation),
			InstanceCount:                  movedCount,
			SourceCapacityReservation:      ec2Stage102CapacityReservationItemFrom(sourceReservation),
		})
		return true
	case "ProvisionByoipCidr":
		multiRegion, ok := parseEC2OptionalBoolValue(r.Form.Get("MultiRegion"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		publiclyAdvertisable, ok := parseEC2OptionalBoolValue(r.Form.Get("PubliclyAdvertisable"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		cidrAuthorizationContext, ok := parseEC2Stage130OptionalCidrAuthorizationContext(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		byoipCidr, err := s.ec2.ProvisionByoipCidr(
			strings.TrimSpace(r.Form.Get("Cidr")),
			cidrAuthorizationContext,
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			multiRegion,
			ec2OptionalStringPointerFromForm(r.Form, "NetworkBorderGroup"),
			parseEC2Stage130PoolTags(r.Form),
			publiclyAdvertisable,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage130ProvisionByoipCidrResponse{
			XMLName:   xml.Name{Local: "ProvisionByoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoipCidr: ec2ByoipCidrItemFrom(byoipCidr),
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage130ReservedInstancesConfigurations(values url.Values) ([]ec2svc.ModifyReservedInstancesTargetConfiguration, bool, bool) {
	const prefix = "ReservedInstancesConfigurationSetItemType."
	if !hasEC2PrefixedField(values, prefix) {
		return nil, false, true
	}

	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		if strings.HasPrefix(rest, "Item.") {
			rest = strings.TrimPrefix(rest, "Item.")
		}
		if strings.HasPrefix(rest, "Member.") {
			rest = strings.TrimPrefix(rest, "Member.")
		}
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	if len(ordered) == 0 {
		return nil, true, false
	}

	configurations := make([]ec2svc.ModifyReservedInstancesTargetConfiguration, 0, len(ordered))
	for _, idx := range ordered {
		instanceCountValue := ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "InstanceCount")
		instanceCount, ok := parseEC2OptionalInt32(instanceCountValue)
		if !ok || instanceCount == nil || *instanceCount <= 0 {
			return nil, true, false
		}
		configurations = append(configurations, ec2svc.ModifyReservedInstancesTargetConfiguration{
			AvailabilityZone:   ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "AvailabilityZone"),
			AvailabilityZoneID: ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "AvailabilityZoneId"),
			InstanceCount:      *instanceCount,
			InstanceType:       ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "InstanceType"),
			Platform:           ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "Platform"),
			Scope:              ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "Scope"),
		})
	}
	return configurations, true, true
}

func parseEC2Stage130SnapshotCreateVolumePermissionList(values url.Values, prefix string) ([]ec2svc.SnapshotCreateVolumePermission, bool, bool) {
	if !hasEC2PrefixedField(values, prefix) {
		return nil, false, true
	}

	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		if strings.HasPrefix(rest, "Item.") {
			rest = strings.TrimPrefix(rest, "Item.")
		}
		if strings.HasPrefix(rest, "Member.") {
			rest = strings.TrimPrefix(rest, "Member.")
		}
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	if len(ordered) == 0 {
		return nil, true, false
	}

	out := make([]ec2svc.SnapshotCreateVolumePermission, 0, len(ordered))
	for _, idx := range ordered {
		group := ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "Group")
		userID := ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "UserId")
		if strings.TrimSpace(group) == "" && strings.TrimSpace(userID) == "" {
			return nil, true, false
		}
		out = append(out, ec2svc.SnapshotCreateVolumePermission{
			Group:  strings.TrimSpace(group),
			UserID: strings.TrimSpace(userID),
		})
	}
	return out, true, true
}

func parseEC2Stage130OptionalTrafficMirrorPortRange(values url.Values, prefix string) (*ec2svc.TrafficMirrorPortRange, bool) {
	fromKey := prefix + "FromPort"
	toKey := prefix + "ToPort"
	if !hasEC2Field(values, fromKey) && !hasEC2Field(values, toKey) {
		return nil, true
	}

	fromPort, fromOK := parseEC2OptionalInt32(values.Get(fromKey))
	toPort, toOK := parseEC2OptionalInt32(values.Get(toKey))
	if !fromOK || !toOK {
		return nil, false
	}
	if fromPort == nil || toPort == nil {
		return nil, false
	}

	return &ec2svc.TrafficMirrorPortRange{FromPort: fromPort, ToPort: toPort}, true
}

func parseEC2Stage130OptionalCidrAuthorizationContext(values url.Values) (*ec2svc.CidrAuthorizationContext, bool) {
	messageKey := "CidrAuthorizationContext.Message"
	signatureKey := "CidrAuthorizationContext.Signature"
	hasMessage := hasEC2Field(values, messageKey)
	hasSignature := hasEC2Field(values, signatureKey)
	if !hasMessage && !hasSignature {
		return nil, true
	}
	if !hasMessage || !hasSignature {
		return nil, false
	}

	message := strings.TrimSpace(values.Get(messageKey))
	signature := strings.TrimSpace(values.Get(signatureKey))
	if message == "" || signature == "" {
		return nil, false
	}

	return &ec2svc.CidrAuthorizationContext{
		Message:   message,
		Signature: signature,
	}, true
}

func parseEC2Stage130PoolTags(values url.Values) []ec2svc.Tag {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "PoolTagSpecification.") || !strings.HasSuffix(key, ".ResourceType") {
			continue
		}
		rest := strings.TrimPrefix(key, "PoolTagSpecification.")
		rest = strings.TrimSuffix(rest, ".ResourceType")
		idx, err := strconv.Atoi(rest)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	for _, idx := range ordered {
		tags := parseEC2Tags(values, "PoolTagSpecification."+strconv.Itoa(idx)+".Tag.")
		if len(tags) > 0 {
			return tags
		}
	}
	return nil
}

type ec2Stage130ModifyReservedInstancesResponse struct {
	XMLName                         xml.Name `xml:"ModifyReservedInstancesResponse"`
	Xmlns                           string   `xml:"xmlns,attr"`
	RequestID                       string   `xml:"requestId"`
	ReservedInstancesModificationID string   `xml:"reservedInstancesModificationId,omitempty"`
}

type ec2Stage130ModifySnapshotTierResponse struct {
	XMLName          xml.Name `xml:"ModifySnapshotTierResponse"`
	Xmlns            string   `xml:"xmlns,attr"`
	RequestID        string   `xml:"requestId"`
	SnapshotID       string   `xml:"snapshotId,omitempty"`
	TieringStartTime string   `xml:"tieringStartTime,omitempty"`
}

type ec2Stage130ModifyTrafficMirrorFilterNetworkServicesResponse struct {
	XMLName             xml.Name                           `xml:"ModifyTrafficMirrorFilterNetworkServicesResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	TrafficMirrorFilter ec2Stage110TrafficMirrorFilterItem `xml:"trafficMirrorFilter"`
}

type ec2Stage130ModifyTrafficMirrorFilterRuleResponse struct {
	XMLName                 xml.Name                               `xml:"ModifyTrafficMirrorFilterRuleResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	TrafficMirrorFilterRule ec2Stage110TrafficMirrorFilterRuleItem `xml:"trafficMirrorFilterRule"`
}

type ec2Stage130ModifyTrafficMirrorSessionResponse struct {
	XMLName              xml.Name                            `xml:"ModifyTrafficMirrorSessionResponse"`
	Xmlns                string                              `xml:"xmlns,attr"`
	RequestID            string                              `xml:"requestId"`
	TrafficMirrorSession ec2Stage110TrafficMirrorSessionItem `xml:"trafficMirrorSession"`
}

type ec2Stage130MoveByoipCidrToIpamResponse struct {
	XMLName   xml.Name         `xml:"MoveByoipCidrToIpamResponse"`
	Xmlns     string           `xml:"xmlns,attr"`
	RequestID string           `xml:"requestId"`
	ByoipCidr ec2ByoipCidrItem `xml:"byoipCidr"`
}

type ec2Stage130MoveCapacityReservationInstancesResponse struct {
	XMLName                        xml.Name                           `xml:"MoveCapacityReservationInstancesResponse"`
	Xmlns                          string                             `xml:"xmlns,attr"`
	RequestID                      string                             `xml:"requestId"`
	DestinationCapacityReservation ec2Stage102CapacityReservationItem `xml:"destinationCapacityReservation"`
	InstanceCount                  int32                              `xml:"instanceCount,omitempty"`
	SourceCapacityReservation      ec2Stage102CapacityReservationItem `xml:"sourceCapacityReservation"`
}

type ec2Stage130ProvisionByoipCidrResponse struct {
	XMLName   xml.Name         `xml:"ProvisionByoipCidrResponse"`
	Xmlns     string           `xml:"xmlns,attr"`
	RequestID string           `xml:"requestId"`
	ByoipCidr ec2ByoipCidrItem `xml:"byoipCidr"`
}
