package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage110Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateSpotDatafeedSubscription":
		subscription, err := s.ec2.CreateSpotDatafeedSubscription(
			strings.TrimSpace(r.Form.Get("Bucket")),
			parseEC2OptionalString(r.Form.Get("Prefix")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateSpotDatafeedSubscriptionResponse{
			XMLName:                  xml.Name{Local: "CreateSpotDatafeedSubscriptionResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			SpotDatafeedSubscription: ec2Stage110SpotDatafeedSubscriptionItemFrom(subscription),
		})
		return true
	case "CreateStoreImageTask":
		objectKey, err := s.ec2.CreateStoreImageTask(
			strings.TrimSpace(r.Form.Get("Bucket")),
			strings.TrimSpace(r.Form.Get("ImageId")),
			parseEC2Tags(r.Form, "S3ObjectTag."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateStoreImageTaskResponse{
			XMLName:   xml.Name{Local: "CreateStoreImageTaskResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ObjectKey: objectKey,
		})
		return true
	case "CreateTrafficMirrorFilter":
		clientToken, filter, err := s.ec2.CreateTrafficMirrorFilter(
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "traffic-mirror-filter"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateTrafficMirrorFilterResponse{
			XMLName:             xml.Name{Local: "CreateTrafficMirrorFilterResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ClientToken:         clientToken,
			TrafficMirrorFilter: ec2Stage110TrafficMirrorFilterItemFrom(filter),
		})
		return true
	case "CreateTrafficMirrorFilterRule":
		ruleNumber, ok := parseEC2OptionalInt32(r.Form.Get("RuleNumber"))
		if !ok || ruleNumber == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		protocol, ok := parseEC2OptionalInt32(r.Form.Get("Protocol"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		destinationFromPort, ok := parseEC2OptionalInt32(r.Form.Get("DestinationPortRange.FromPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		destinationToPort, ok := parseEC2OptionalInt32(r.Form.Get("DestinationPortRange.ToPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sourceFromPort, ok := parseEC2OptionalInt32(r.Form.Get("SourcePortRange.FromPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sourceToPort, ok := parseEC2OptionalInt32(r.Form.Get("SourcePortRange.ToPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		clientToken, rule, err := s.ec2.CreateTrafficMirrorFilterRule(
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			strings.TrimSpace(r.Form.Get("RuleAction")),
			*ruleNumber,
			strings.TrimSpace(r.Form.Get("SourceCidrBlock")),
			strings.TrimSpace(r.Form.Get("TrafficDirection")),
			strings.TrimSpace(r.Form.Get("TrafficMirrorFilterId")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2OptionalString(r.Form.Get("Description")),
			ec2svc.TrafficMirrorPortRange{FromPort: destinationFromPort, ToPort: destinationToPort},
			protocol,
			ec2svc.TrafficMirrorPortRange{FromPort: sourceFromPort, ToPort: sourceToPort},
			parseEC2TagSpecificationsForResource(r.Form, "traffic-mirror-filter-rule"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateTrafficMirrorFilterRuleResponse{
			XMLName:                 xml.Name{Local: "CreateTrafficMirrorFilterRuleResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			ClientToken:             clientToken,
			TrafficMirrorFilterRule: ec2Stage110TrafficMirrorFilterRuleItemFrom(rule),
		})
		return true
	case "CreateTrafficMirrorSession":
		sessionNumber, ok := parseEC2OptionalInt32(r.Form.Get("SessionNumber"))
		if !ok || sessionNumber == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		packetLength, ok := parseEC2OptionalInt32(r.Form.Get("PacketLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		virtualNetworkID, ok := parseEC2OptionalInt32(r.Form.Get("VirtualNetworkId"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		clientToken, session, err := s.ec2.CreateTrafficMirrorSession(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			*sessionNumber,
			strings.TrimSpace(r.Form.Get("TrafficMirrorFilterId")),
			strings.TrimSpace(r.Form.Get("TrafficMirrorTargetId")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2OptionalString(r.Form.Get("Description")),
			packetLength,
			virtualNetworkID,
			parseEC2TagSpecificationsForResource(r.Form, "traffic-mirror-session"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateTrafficMirrorSessionResponse{
			XMLName:              xml.Name{Local: "CreateTrafficMirrorSessionResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			ClientToken:          clientToken,
			TrafficMirrorSession: ec2Stage110TrafficMirrorSessionItemFrom(session),
		})
		return true
	case "CreateTrafficMirrorTarget":
		clientToken, target, err := s.ec2.CreateTrafficMirrorTarget(
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("GatewayLoadBalancerEndpointId")),
			parseEC2OptionalString(r.Form.Get("NetworkInterfaceId")),
			parseEC2OptionalString(r.Form.Get("NetworkLoadBalancerArn")),
			parseEC2TagSpecificationsForResource(r.Form, "traffic-mirror-target"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110CreateTrafficMirrorTargetResponse{
			XMLName:             xml.Name{Local: "CreateTrafficMirrorTargetResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ClientToken:         clientToken,
			TrafficMirrorTarget: ec2Stage110TrafficMirrorTargetItemFrom(target),
		})
		return true
	case "DeleteCarrierGateway":
		gateway, err := s.ec2.DeleteCarrierGateway(strings.TrimSpace(r.Form.Get("CarrierGatewayId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110DeleteCarrierGatewayResponse{
			XMLName:   xml.Name{Local: "DeleteCarrierGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CarrierGateway: ec2Stage105CarrierGatewayItem{
				CarrierGatewayID: gateway.ID,
				OwnerID:          gateway.OwnerID,
				State:            gateway.State,
				TagSet:           ec2TagSet{Items: ec2TagItemsFromMap(gateway.Tags)},
				VpcID:            gateway.VpcID,
			},
		})
		return true
	case "DeleteCoipCidr":
		coipCidr, err := s.ec2.DeleteCoipCidr(
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("CoipPoolId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110DeleteCoipCidrResponse{
			XMLName:   xml.Name{Local: "DeleteCoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CoipCidr:  ec2Stage106CoipCidrItem{Cidr: coipCidr.Cidr, CoipPoolID: coipCidr.CoipPoolID, LocalGatewayRouteTableID: coipCidr.LocalGatewayRouteTableID},
		})
		return true
	case "DeleteCoipPool":
		coipPool, err := s.ec2.DeleteCoipPool(strings.TrimSpace(r.Form.Get("CoipPoolId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110DeleteCoipPoolResponse{
			XMLName:   xml.Name{Local: "DeleteCoipPoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CoipPool:  ec2Stage107CoipPoolItemFrom(coipPool),
		})
		return true
	case "DeleteFleets":
		terminateInstances, hasTerminateInstances, ok := ec2OptionalBoolFromForm(r.Form, "TerminateInstances")
		if !ok || !hasTerminateInstances || terminateInstances == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		successes, errors, err := s.ec2.DeleteFleets(
			parseEC2MembersOrItemList(r.Form, "FleetId"),
			*terminateInstances,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage110DeleteFleetsResponse{
			XMLName:                    xml.Name{Local: "DeleteFleetsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			SuccessfulFleetDeletions:   ec2Stage110DeleteFleetSuccessSet{Items: ec2Stage110DeleteFleetSuccessItemsFrom(successes)},
			UnsuccessfulFleetDeletions: ec2Stage110DeleteFleetErrorSet{Items: ec2Stage110DeleteFleetErrorItemsFrom(errors)},
		})
		return true
	default:
		return false
	}
}

func ec2Stage110SpotDatafeedSubscriptionItemFrom(in ec2svc.SpotDatafeedSubscription) ec2Stage110SpotDatafeedSubscriptionItem {
	return ec2Stage110SpotDatafeedSubscriptionItem{
		Bucket:  in.Bucket,
		OwnerID: in.OwnerID,
		Prefix:  in.Prefix,
		State:   in.State,
	}
}

func ec2Stage110TrafficMirrorFilterItemFrom(in ec2svc.TrafficMirrorFilter) ec2Stage110TrafficMirrorFilterItem {
	return ec2Stage110TrafficMirrorFilterItem{
		Description:           in.Description,
		NetworkServiceSet:     ec2ValueStringSet{Items: append([]string(nil), in.NetworkServices...)},
		TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TrafficMirrorFilterID: in.TrafficMirrorFilterID,
	}
}

func ec2Stage110TrafficMirrorFilterRuleItemFrom(in ec2svc.TrafficMirrorFilterRule) ec2Stage110TrafficMirrorFilterRuleItem {
	return ec2Stage110TrafficMirrorFilterRuleItem{
		Description:               in.Description,
		DestinationCidrBlock:      in.DestinationCidrBlock,
		DestinationPortRange:      ec2Stage110TrafficMirrorPortRangeItem{FromPort: in.DestinationPortRange.FromPort, ToPort: in.DestinationPortRange.ToPort},
		Protocol:                  in.Protocol,
		RuleAction:                in.RuleAction,
		RuleNumber:                &in.RuleNumber,
		SourceCidrBlock:           in.SourceCidrBlock,
		SourcePortRange:           ec2Stage110TrafficMirrorPortRangeItem{FromPort: in.SourcePortRange.FromPort, ToPort: in.SourcePortRange.ToPort},
		TagSet:                    ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TrafficDirection:          in.TrafficDirection,
		TrafficMirrorFilterID:     in.TrafficMirrorFilterID,
		TrafficMirrorFilterRuleID: in.TrafficMirrorFilterRuleID,
	}
}

func ec2Stage110TrafficMirrorSessionItemFrom(in ec2svc.TrafficMirrorSession) ec2Stage110TrafficMirrorSessionItem {
	return ec2Stage110TrafficMirrorSessionItem{
		Description:            in.Description,
		NetworkInterfaceID:     in.NetworkInterfaceID,
		OwnerID:                in.OwnerID,
		PacketLength:           in.PacketLength,
		SessionNumber:          &in.SessionNumber,
		TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TrafficMirrorFilterID:  in.TrafficMirrorFilterID,
		TrafficMirrorSessionID: in.TrafficMirrorSessionID,
		TrafficMirrorTargetID:  in.TrafficMirrorTargetID,
		VirtualNetworkID:       in.VirtualNetworkID,
	}
}

func ec2Stage110TrafficMirrorTargetItemFrom(in ec2svc.TrafficMirrorTarget) ec2Stage110TrafficMirrorTargetItem {
	return ec2Stage110TrafficMirrorTargetItem{
		Description:                   in.Description,
		GatewayLoadBalancerEndpointID: in.GatewayLoadBalancerEndpointID,
		NetworkInterfaceID:            in.NetworkInterfaceID,
		NetworkLoadBalancerARN:        in.NetworkLoadBalancerARN,
		OwnerID:                       in.OwnerID,
		TagSet:                        ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TrafficMirrorTargetID:         in.TrafficMirrorTargetID,
		Type:                          in.Type,
	}
}

func ec2Stage110DeleteFleetSuccessItemsFrom(in []ec2svc.DeleteFleetSuccess) []ec2Stage110DeleteFleetSuccessItem {
	out := make([]ec2Stage110DeleteFleetSuccessItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110DeleteFleetSuccessItem{
			CurrentFleetState:  item.CurrentFleetState,
			FleetID:            item.FleetID,
			PreviousFleetState: item.PreviousFleetState,
		})
	}
	return out
}

func ec2Stage110DeleteFleetErrorItemsFrom(in []ec2svc.DeleteFleetErrorItem) []ec2Stage110DeleteFleetErrorItem {
	out := make([]ec2Stage110DeleteFleetErrorItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110DeleteFleetErrorItem{
			Error: ec2Stage110DeleteFleetError{
				Code:    item.Error.Code,
				Message: item.Error.Message,
			},
			FleetID: item.FleetID,
		})
	}
	return out
}

type ec2Stage110CreateSpotDatafeedSubscriptionResponse struct {
	XMLName                  xml.Name                                `xml:"CreateSpotDatafeedSubscriptionResponse"`
	Xmlns                    string                                  `xml:"xmlns,attr"`
	RequestID                string                                  `xml:"requestId"`
	SpotDatafeedSubscription ec2Stage110SpotDatafeedSubscriptionItem `xml:"spotDatafeedSubscription"`
}

type ec2Stage110CreateStoreImageTaskResponse struct {
	XMLName   xml.Name `xml:"CreateStoreImageTaskResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	ObjectKey string   `xml:"objectKey,omitempty"`
}

type ec2Stage110CreateTrafficMirrorFilterResponse struct {
	XMLName             xml.Name                           `xml:"CreateTrafficMirrorFilterResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	ClientToken         string                             `xml:"clientToken,omitempty"`
	TrafficMirrorFilter ec2Stage110TrafficMirrorFilterItem `xml:"trafficMirrorFilter"`
}

type ec2Stage110CreateTrafficMirrorFilterRuleResponse struct {
	XMLName                 xml.Name                               `xml:"CreateTrafficMirrorFilterRuleResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	ClientToken             string                                 `xml:"clientToken,omitempty"`
	TrafficMirrorFilterRule ec2Stage110TrafficMirrorFilterRuleItem `xml:"trafficMirrorFilterRule"`
}

type ec2Stage110CreateTrafficMirrorSessionResponse struct {
	XMLName              xml.Name                            `xml:"CreateTrafficMirrorSessionResponse"`
	Xmlns                string                              `xml:"xmlns,attr"`
	RequestID            string                              `xml:"requestId"`
	ClientToken          string                              `xml:"clientToken,omitempty"`
	TrafficMirrorSession ec2Stage110TrafficMirrorSessionItem `xml:"trafficMirrorSession"`
}

type ec2Stage110CreateTrafficMirrorTargetResponse struct {
	XMLName             xml.Name                           `xml:"CreateTrafficMirrorTargetResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	ClientToken         string                             `xml:"clientToken,omitempty"`
	TrafficMirrorTarget ec2Stage110TrafficMirrorTargetItem `xml:"trafficMirrorTarget"`
}

type ec2Stage110DeleteCarrierGatewayResponse struct {
	XMLName        xml.Name                      `xml:"DeleteCarrierGatewayResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	CarrierGateway ec2Stage105CarrierGatewayItem `xml:"carrierGateway"`
}

type ec2Stage110DeleteCoipCidrResponse struct {
	XMLName   xml.Name                `xml:"DeleteCoipCidrResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	CoipCidr  ec2Stage106CoipCidrItem `xml:"coipCidr"`
}

type ec2Stage110DeleteCoipPoolResponse struct {
	XMLName   xml.Name                `xml:"DeleteCoipPoolResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	CoipPool  ec2Stage107CoipPoolItem `xml:"coipPool"`
}

type ec2Stage110DeleteFleetsResponse struct {
	XMLName                    xml.Name                         `xml:"DeleteFleetsResponse"`
	Xmlns                      string                           `xml:"xmlns,attr"`
	RequestID                  string                           `xml:"requestId"`
	SuccessfulFleetDeletions   ec2Stage110DeleteFleetSuccessSet `xml:"successfulFleetDeletionSet"`
	UnsuccessfulFleetDeletions ec2Stage110DeleteFleetErrorSet   `xml:"unsuccessfulFleetDeletionSet"`
}

type ec2Stage110SpotDatafeedSubscriptionItem struct {
	Bucket  string `xml:"bucket,omitempty"`
	OwnerID string `xml:"ownerId,omitempty"`
	Prefix  string `xml:"prefix,omitempty"`
	State   string `xml:"state,omitempty"`
}

type ec2Stage110TrafficMirrorFilterItem struct {
	Description           string            `xml:"description,omitempty"`
	NetworkServiceSet     ec2ValueStringSet `xml:"networkServiceSet"`
	TagSet                ec2TagSet         `xml:"tagSet"`
	TrafficMirrorFilterID string            `xml:"trafficMirrorFilterId,omitempty"`
}

type ec2Stage110TrafficMirrorPortRangeItem struct {
	FromPort *int32 `xml:"fromPort,omitempty"`
	ToPort   *int32 `xml:"toPort,omitempty"`
}

type ec2Stage110TrafficMirrorFilterRuleItem struct {
	Description               string                                `xml:"description,omitempty"`
	DestinationCidrBlock      string                                `xml:"destinationCidrBlock,omitempty"`
	DestinationPortRange      ec2Stage110TrafficMirrorPortRangeItem `xml:"destinationPortRange"`
	Protocol                  *int32                                `xml:"protocol,omitempty"`
	RuleAction                string                                `xml:"ruleAction,omitempty"`
	RuleNumber                *int32                                `xml:"ruleNumber,omitempty"`
	SourceCidrBlock           string                                `xml:"sourceCidrBlock,omitempty"`
	SourcePortRange           ec2Stage110TrafficMirrorPortRangeItem `xml:"sourcePortRange"`
	TagSet                    ec2TagSet                             `xml:"tagSet"`
	TrafficDirection          string                                `xml:"trafficDirection,omitempty"`
	TrafficMirrorFilterID     string                                `xml:"trafficMirrorFilterId,omitempty"`
	TrafficMirrorFilterRuleID string                                `xml:"trafficMirrorFilterRuleId,omitempty"`
}

type ec2Stage110TrafficMirrorSessionItem struct {
	Description            string    `xml:"description,omitempty"`
	NetworkInterfaceID     string    `xml:"networkInterfaceId,omitempty"`
	OwnerID                string    `xml:"ownerId,omitempty"`
	PacketLength           *int32    `xml:"packetLength,omitempty"`
	SessionNumber          *int32    `xml:"sessionNumber,omitempty"`
	TagSet                 ec2TagSet `xml:"tagSet"`
	TrafficMirrorFilterID  string    `xml:"trafficMirrorFilterId,omitempty"`
	TrafficMirrorSessionID string    `xml:"trafficMirrorSessionId,omitempty"`
	TrafficMirrorTargetID  string    `xml:"trafficMirrorTargetId,omitempty"`
	VirtualNetworkID       *int32    `xml:"virtualNetworkId,omitempty"`
}

type ec2Stage110TrafficMirrorTargetItem struct {
	Description                   string    `xml:"description,omitempty"`
	GatewayLoadBalancerEndpointID string    `xml:"gatewayLoadBalancerEndpointId,omitempty"`
	NetworkInterfaceID            string    `xml:"networkInterfaceId,omitempty"`
	NetworkLoadBalancerARN        string    `xml:"networkLoadBalancerArn,omitempty"`
	OwnerID                       string    `xml:"ownerId,omitempty"`
	TagSet                        ec2TagSet `xml:"tagSet"`
	TrafficMirrorTargetID         string    `xml:"trafficMirrorTargetId,omitempty"`
	Type                          string    `xml:"type,omitempty"`
}

type ec2Stage110DeleteFleetSuccessSet struct {
	Items []ec2Stage110DeleteFleetSuccessItem `xml:"item"`
}

type ec2Stage110DeleteFleetSuccessItem struct {
	CurrentFleetState  string `xml:"currentFleetState,omitempty"`
	FleetID            string `xml:"fleetId,omitempty"`
	PreviousFleetState string `xml:"previousFleetState,omitempty"`
}

type ec2Stage110DeleteFleetErrorSet struct {
	Items []ec2Stage110DeleteFleetErrorItem `xml:"item"`
}

type ec2Stage110DeleteFleetErrorItem struct {
	Error   ec2Stage110DeleteFleetError `xml:"error"`
	FleetID string                      `xml:"fleetId,omitempty"`
}

type ec2Stage110DeleteFleetError struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}
