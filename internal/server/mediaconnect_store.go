package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type mediaConnectStore struct {
	mu sync.Mutex

	nextFlowID            int64
	nextBridgeID          int64
	nextGatewayID         int64
	nextGatewayInstanceID int64
	nextRouterInputID     int64
	nextRouterOutputID    int64
	nextRouterIfaceID     int64
	nextEntitlementID     int64
	nextOfferingID        int64
	nextReservationID     int64

	flows                map[string]map[string]any
	bridges              map[string]map[string]any
	gateways             map[string]map[string]any
	gatewayInstances     map[string]map[string]any
	routerInputs         map[string]map[string]any
	routerOutputs        map[string]map[string]any
	routerIfaces         map[string]map[string]any
	entitlements         map[string]map[string]any
	offerings            map[string]map[string]any
	reservations         map[string]map[string]any
	resourceTags         map[string]map[string]string
	globalResourceTags   map[string]map[string]string
	flowSubresourceState map[string]map[string]map[string]any
}

func newMediaConnectStore() *mediaConnectStore {
	s := &mediaConnectStore{
		nextFlowID:            2,
		nextBridgeID:          2,
		nextGatewayID:         2,
		nextGatewayInstanceID: 2,
		nextRouterInputID:     2,
		nextRouterOutputID:    2,
		nextRouterIfaceID:     2,
		nextEntitlementID:     2,
		nextOfferingID:        2,
		nextReservationID:     2,
		flows:                 map[string]map[string]any{},
		bridges:               map[string]map[string]any{},
		gateways:              map[string]map[string]any{},
		gatewayInstances:      map[string]map[string]any{},
		routerInputs:          map[string]map[string]any{},
		routerOutputs:         map[string]map[string]any{},
		routerIfaces:          map[string]map[string]any{},
		entitlements:          map[string]map[string]any{},
		offerings:             map[string]map[string]any{},
		reservations:          map[string]map[string]any{},
		resourceTags:          map[string]map[string]string{},
		globalResourceTags:    map[string]map[string]string{},
		flowSubresourceState:  map[string]map[string]map[string]any{},
	}

	flow := s.ensureFlowLocked(mcARN("flow", "flow-00000001"))
	bridge := s.ensureBridgeLocked(mcARN("bridge", "bridge-00000001"))
	gateway := s.ensureGatewayLocked(mcARN("gateway", "gateway-00000001"))
	gatewayInstance := s.ensureGatewayInstanceLocked(mcARN("gateway-instance", "gateway-instance-00000001"))
	routerInput := s.ensureRouterInputLocked(mcARN("router-input", "router-input-00000001"))
	routerOutput := s.ensureRouterOutputLocked(mcARN("router-output", "router-output-00000001"))
	routerIface := s.ensureRouterIfaceLocked(mcARN("router-interface", "router-interface-00000001"))
	entitlement := s.ensureEntitlementLocked(mcARN("entitlement", "entitlement-00000001"), mcStringAny(flow, "FlowArn"))
	offering := s.ensureOfferingLocked(mcARN("offering", "offering-00000001"))
	reservation := s.ensureReservationLocked(mcARN("reservation", "reservation-00000001"))

	s.resourceTags[mcStringAny(flow, "FlowArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(bridge, "BridgeArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(gateway, "GatewayArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(routerInput, "RouterInputArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(routerOutput, "RouterOutputArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(routerIface, "RouterNetworkInterfaceArn")] = map[string]string{"seed": "true"}
	s.resourceTags[mcStringAny(entitlement, "EntitlementArn")] = map[string]string{"seed": "true"}
	s.globalResourceTags[mcStringAny(offering, "OfferingArn")] = map[string]string{"seed": "true"}
	s.globalResourceTags[mcStringAny(reservation, "ReservationArn")] = map[string]string{"seed": "true"}
	s.globalResourceTags[mcStringAny(gatewayInstance, "GatewayInstanceArn")] = map[string]string{"seed": "true"}

	return s
}

func (s *mediaConnectStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	flowArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "FlowArn"),
		mcStringAny(payload, "FlowArn", "flowArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("flow", "flow-00000001"),
	)
	bridgeArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "BridgeArn"),
		mcStringAny(payload, "BridgeArn", "bridgeArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("bridge", "bridge-00000001"),
	)
	gatewayArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "GatewayArn"),
		mcStringAny(payload, "GatewayArn", "gatewayArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("gateway", "gateway-00000001"),
	)
	gatewayInstanceArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "GatewayInstanceArn"),
		mcStringAny(payload, "GatewayInstanceArn", "gatewayInstanceArn"),
		mcARN("gateway-instance", "gateway-instance-00000001"),
	)
	routerInputArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "RouterInputArn"),
		mcStringAny(payload, "RouterInputArn", "routerInputArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("router-input", "router-input-00000001"),
	)
	routerOutputArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "RouterOutputArn"),
		mcStringAny(payload, "RouterOutputArn", "routerOutputArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("router-output", "router-output-00000001"),
	)
	routerIfaceArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "Arn"),
		mcStringAny(payload, "RouterNetworkInterfaceArn", "routerNetworkInterfaceArn"),
		mcStringAny(payload, "Arn", "arn"),
		mcARN("router-interface", "router-interface-00000001"),
	)
	entitlementArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "EntitlementArn"),
		mcStringAny(payload, "EntitlementArn", "entitlementArn"),
		mcARN("entitlement", "entitlement-00000001"),
	)
	offeringArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "OfferingArn"),
		mcStringAny(payload, "OfferingArn", "offeringArn"),
		mcARN("offering", "offering-00000001"),
	)
	reservationArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "ReservationArn"),
		mcStringAny(payload, "ReservationArn", "reservationArn"),
		mcARN("reservation", "reservation-00000001"),
	)
	resourceArn := mcFirstNonEmpty(
		mcPathParam(pathParams, "ResourceArn"),
		mcStringAny(payload, "ResourceArn", "resourceArn"),
		flowArn,
	)
	sourceName := mcFirstNonEmpty(mcPathParam(pathParams, "SourceName"), mcStringAny(payload, "SourceName", "sourceName"), "primary")
	outputArn := mcFirstNonEmpty(mcPathParam(pathParams, "OutputArn"), mcStringAny(payload, "OutputArn", "outputArn"), mcARN("flow-output", "output-00000001"))
	outputName := mcFirstNonEmpty(mcPathParam(pathParams, "OutputName"), mcStringAny(payload, "OutputName", "outputName"), "output-00000001")
	sourceArn := mcFirstNonEmpty(mcPathParam(pathParams, "SourceArn"), mcStringAny(payload, "SourceArn", "sourceArn"), mcARN("flow-source", "source-00000001"))
	mediaStreamName := mcFirstNonEmpty(mcPathParam(pathParams, "MediaStreamName"), mcStringAny(payload, "MediaStreamName", "mediaStreamName"), "stream-00000001")
	vpcInterfaceName := mcFirstNonEmpty(mcPathParam(pathParams, "VpcInterfaceName"), mcStringAny(payload, "VpcInterfaceName", "vpcInterfaceName"), "vpcif-00000001")

	switch action {
	case "CreateFlow":
		flowArn = mcARN("flow", fmt.Sprintf("flow-%08d", s.nextFlowIDLocked()))
		flow := s.ensureFlowLocked(flowArn)
		flow["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-flow-%d", s.nextFlowID-1))
		flow["Status"] = "STANDBY"
		flow["UpdatedAt"] = now
		return map[string]any{"Flow": mcCloneMap(flow), "FlowArn": flowArn}
	case "DescribeFlow":
		return map[string]any{"Flow": mcCloneMap(s.ensureFlowLocked(flowArn))}
	case "ListFlows":
		return map[string]any{"Flows": s.listByTypeLocked("flow"), "NextToken": ""}
	case "UpdateFlow":
		flow := s.ensureFlowLocked(flowArn)
		for k, v := range payload {
			flow[k] = v
		}
		flow["UpdatedAt"] = now
		return map[string]any{"Flow": mcCloneMap(flow)}
	case "DeleteFlow":
		delete(s.flows, flowArn)
		return map[string]any{"FlowArn": flowArn}
	case "StartFlow":
		flow := s.ensureFlowLocked(flowArn)
		flow["Status"] = "ACTIVE"
		flow["UpdatedAt"] = now
		return map[string]any{"FlowArn": flowArn, "Status": "ACTIVE"}
	case "StopFlow":
		flow := s.ensureFlowLocked(flowArn)
		flow["Status"] = "STANDBY"
		flow["UpdatedAt"] = now
		return map[string]any{"FlowArn": flowArn, "Status": "STANDBY"}

	case "CreateBridge":
		bridgeArn = mcARN("bridge", fmt.Sprintf("bridge-%08d", s.nextBridgeIDLocked()))
		bridge := s.ensureBridgeLocked(bridgeArn)
		bridge["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-bridge-%d", s.nextBridgeID-1))
		bridge["EgressGatewayBridge"] = map[string]any{"MaxBitrate": 10000000}
		bridge["UpdatedAt"] = now
		return map[string]any{"Bridge": mcCloneMap(bridge), "BridgeArn": bridgeArn}
	case "DescribeBridge":
		return map[string]any{"Bridge": mcCloneMap(s.ensureBridgeLocked(bridgeArn))}
	case "ListBridges":
		return map[string]any{"Bridges": s.listByTypeLocked("bridge"), "NextToken": ""}
	case "UpdateBridge", "UpdateBridgeOutput", "UpdateBridgeSource":
		bridge := s.ensureBridgeLocked(bridgeArn)
		for k, v := range payload {
			bridge[k] = v
		}
		bridge["UpdatedAt"] = now
		return map[string]any{"Bridge": mcCloneMap(bridge)}
	case "UpdateBridgeState":
		bridge := s.ensureBridgeLocked(bridgeArn)
		desired := mcFirstNonEmpty(mcStringAny(payload, "DesiredState", "desiredState"), "ACTIVE")
		bridge["State"] = desired
		bridge["UpdatedAt"] = now
		return map[string]any{"BridgeArn": bridgeArn, "State": desired}
	case "DeleteBridge":
		delete(s.bridges, bridgeArn)
		return map[string]any{"BridgeArn": bridgeArn}

	case "CreateGateway":
		gatewayArn = mcARN("gateway", fmt.Sprintf("gateway-%08d", s.nextGatewayIDLocked()))
		gateway := s.ensureGatewayLocked(gatewayArn)
		gateway["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-gateway-%d", s.nextGatewayID-1))
		gateway["UpdatedAt"] = now
		return map[string]any{"Gateway": mcCloneMap(gateway), "GatewayArn": gatewayArn}
	case "DescribeGateway":
		return map[string]any{"Gateway": mcCloneMap(s.ensureGatewayLocked(gatewayArn))}
	case "ListGateways":
		return map[string]any{"Gateways": s.listByTypeLocked("gateway"), "NextToken": ""}
	case "DeleteGateway":
		delete(s.gateways, gatewayArn)
		return map[string]any{"GatewayArn": gatewayArn}
	case "DescribeGatewayInstance":
		return map[string]any{"GatewayInstance": mcCloneMap(s.ensureGatewayInstanceLocked(gatewayInstanceArn))}
	case "ListGatewayInstances":
		return map[string]any{"Instances": s.listByTypeLocked("gateway-instance"), "NextToken": ""}
	case "UpdateGatewayInstance":
		gw := s.ensureGatewayInstanceLocked(gatewayInstanceArn)
		for k, v := range payload {
			gw[k] = v
		}
		gw["UpdatedAt"] = now
		return map[string]any{"GatewayInstance": mcCloneMap(gw)}
	case "DeregisterGatewayInstance":
		gw := s.ensureGatewayInstanceLocked(gatewayInstanceArn)
		gw["Status"] = "DEREGISTERED"
		gw["UpdatedAt"] = now
		return map[string]any{"GatewayInstanceArn": gatewayInstanceArn, "Status": "DEREGISTERED"}

	case "CreateRouterInput":
		routerInputArn = mcARN("router-input", fmt.Sprintf("router-input-%08d", s.nextRouterInputIDLocked()))
		ri := s.ensureRouterInputLocked(routerInputArn)
		ri["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-router-input-%d", s.nextRouterInputID-1))
		ri["State"] = "STANDBY"
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInput": mcCloneMap(ri), "RouterInputArn": routerInputArn}
	case "GetRouterInput":
		return map[string]any{"RouterInput": mcCloneMap(s.ensureRouterInputLocked(routerInputArn))}
	case "ListRouterInputs":
		return map[string]any{"RouterInputs": s.listByTypeLocked("router-input"), "NextToken": ""}
	case "UpdateRouterInput":
		ri := s.ensureRouterInputLocked(routerInputArn)
		for k, v := range payload {
			ri[k] = v
		}
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInput": mcCloneMap(ri)}
	case "DeleteRouterInput":
		delete(s.routerInputs, routerInputArn)
		return map[string]any{"RouterInputArn": routerInputArn}
	case "StartRouterInput":
		ri := s.ensureRouterInputLocked(routerInputArn)
		ri["State"] = "ACTIVE"
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInputArn": routerInputArn, "State": "ACTIVE"}
	case "StopRouterInput":
		ri := s.ensureRouterInputLocked(routerInputArn)
		ri["State"] = "STANDBY"
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInputArn": routerInputArn, "State": "STANDBY"}
	case "RestartRouterInput":
		ri := s.ensureRouterInputLocked(routerInputArn)
		ri["State"] = "ACTIVE"
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInputArn": routerInputArn, "State": "ACTIVE"}
	case "TakeRouterInput":
		ri := s.ensureRouterInputLocked(routerInputArn)
		ri["Priority"] = "PRIMARY"
		ri["UpdatedAt"] = now
		return map[string]any{"RouterInputArn": routerInputArn, "Priority": "PRIMARY"}
	case "GetRouterInputSourceMetadata":
		return map[string]any{"SourceMetadata": map[string]any{"RouterInputArn": routerInputArn, "SourceName": sourceName, "Resolution": "1920x1080"}}
	case "GetRouterInputThumbnail":
		return map[string]any{"ThumbnailData": "c3RhY2t5YXJk", "ContentType": "image/jpeg", "RouterInputArn": routerInputArn}
	case "BatchGetRouterInput":
		return map[string]any{"RouterInputs": s.listByTypeLocked("router-input"), "UnprocessedArns": []any{}}

	case "CreateRouterOutput":
		routerOutputArn = mcARN("router-output", fmt.Sprintf("router-output-%08d", s.nextRouterOutputIDLocked()))
		ro := s.ensureRouterOutputLocked(routerOutputArn)
		ro["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-router-output-%d", s.nextRouterOutputID-1))
		ro["State"] = "STANDBY"
		ro["UpdatedAt"] = now
		return map[string]any{"RouterOutput": mcCloneMap(ro), "RouterOutputArn": routerOutputArn}
	case "GetRouterOutput":
		return map[string]any{"RouterOutput": mcCloneMap(s.ensureRouterOutputLocked(routerOutputArn))}
	case "ListRouterOutputs":
		return map[string]any{"RouterOutputs": s.listByTypeLocked("router-output"), "NextToken": ""}
	case "UpdateRouterOutput":
		ro := s.ensureRouterOutputLocked(routerOutputArn)
		for k, v := range payload {
			ro[k] = v
		}
		ro["UpdatedAt"] = now
		return map[string]any{"RouterOutput": mcCloneMap(ro)}
	case "DeleteRouterOutput":
		delete(s.routerOutputs, routerOutputArn)
		return map[string]any{"RouterOutputArn": routerOutputArn}
	case "StartRouterOutput":
		ro := s.ensureRouterOutputLocked(routerOutputArn)
		ro["State"] = "ACTIVE"
		ro["UpdatedAt"] = now
		return map[string]any{"RouterOutputArn": routerOutputArn, "State": "ACTIVE"}
	case "StopRouterOutput":
		ro := s.ensureRouterOutputLocked(routerOutputArn)
		ro["State"] = "STANDBY"
		ro["UpdatedAt"] = now
		return map[string]any{"RouterOutputArn": routerOutputArn, "State": "STANDBY"}
	case "RestartRouterOutput":
		ro := s.ensureRouterOutputLocked(routerOutputArn)
		ro["State"] = "ACTIVE"
		ro["UpdatedAt"] = now
		return map[string]any{"RouterOutputArn": routerOutputArn, "State": "ACTIVE"}
	case "BatchGetRouterOutput":
		return map[string]any{"RouterOutputs": s.listByTypeLocked("router-output"), "UnprocessedArns": []any{}}

	case "CreateRouterNetworkInterface":
		routerIfaceArn = mcARN("router-interface", fmt.Sprintf("router-interface-%08d", s.nextRouterIfaceIDLocked()))
		ri := s.ensureRouterIfaceLocked(routerIfaceArn)
		ri["Name"] = mcFirstNonEmpty(mcStringAny(payload, "Name", "name"), fmt.Sprintf("stackyard-router-interface-%d", s.nextRouterIfaceID-1))
		ri["UpdatedAt"] = now
		return map[string]any{"RouterNetworkInterface": mcCloneMap(ri), "RouterNetworkInterfaceArn": routerIfaceArn}
	case "GetRouterNetworkInterface":
		return map[string]any{"RouterNetworkInterface": mcCloneMap(s.ensureRouterIfaceLocked(routerIfaceArn))}
	case "ListRouterNetworkInterfaces":
		return map[string]any{"RouterNetworkInterfaces": s.listByTypeLocked("router-interface"), "NextToken": ""}
	case "UpdateRouterNetworkInterface":
		ri := s.ensureRouterIfaceLocked(routerIfaceArn)
		for k, v := range payload {
			ri[k] = v
		}
		ri["UpdatedAt"] = now
		return map[string]any{"RouterNetworkInterface": mcCloneMap(ri)}
	case "DeleteRouterNetworkInterface":
		delete(s.routerIfaces, routerIfaceArn)
		return map[string]any{"RouterNetworkInterfaceArn": routerIfaceArn}
	case "BatchGetRouterNetworkInterface":
		return map[string]any{"RouterNetworkInterfaces": s.listByTypeLocked("router-interface"), "UnprocessedArns": []any{}}

	case "DescribeOffering":
		return map[string]any{"Offering": mcCloneMap(s.ensureOfferingLocked(offeringArn))}
	case "ListOfferings":
		return map[string]any{"Offerings": s.listByTypeLocked("offering"), "NextToken": ""}
	case "PurchaseOffering":
		reservationArn = mcARN("reservation", fmt.Sprintf("reservation-%08d", s.nextReservationIDLocked()))
		res := s.ensureReservationLocked(reservationArn)
		res["OfferingArn"] = offeringArn
		res["UpdatedAt"] = now
		return map[string]any{"Reservation": mcCloneMap(res), "ReservationArn": reservationArn}
	case "DescribeReservation":
		return map[string]any{"Reservation": mcCloneMap(s.ensureReservationLocked(reservationArn))}
	case "ListReservations":
		return map[string]any{"Reservations": s.listByTypeLocked("reservation"), "NextToken": ""}

	case "GrantFlowEntitlements":
		entitlementArn = mcARN("entitlement", fmt.Sprintf("entitlement-%08d", s.nextEntitlementIDLocked()))
		ent := s.ensureEntitlementLocked(entitlementArn, flowArn)
		ent["UpdatedAt"] = now
		return map[string]any{"Entitlements": []any{mcCloneMap(ent)}}
	case "RevokeFlowEntitlement":
		delete(s.entitlements, entitlementArn)
		return map[string]any{"EntitlementArn": entitlementArn}
	case "UpdateFlowEntitlement":
		ent := s.ensureEntitlementLocked(entitlementArn, flowArn)
		for k, v := range payload {
			ent[k] = v
		}
		ent["UpdatedAt"] = now
		return map[string]any{"Entitlement": mcCloneMap(ent)}
	case "ListEntitlements":
		return map[string]any{"Entitlements": s.listByTypeLocked("entitlement"), "NextToken": ""}

	case "AddFlowOutputs", "AddFlowSources", "AddFlowMediaStreams", "AddFlowVpcInterfaces",
		"RemoveFlowOutput", "RemoveFlowSource", "RemoveFlowMediaStream", "RemoveFlowVpcInterface",
		"UpdateFlowOutput", "UpdateFlowSource", "UpdateFlowMediaStream":
		s.bumpFlowSubresourceStateLocked(flowArn, action, now)
		return map[string]any{"FlowArn": flowArn, "OutputArn": outputArn, "OutputName": outputName, "SourceArn": sourceArn, "SourceName": sourceName, "MediaStreamName": mediaStreamName, "VpcInterfaceName": vpcInterfaceName}
	case "AddBridgeOutputs", "AddBridgeSources", "RemoveBridgeOutput", "RemoveBridgeSource":
		bridge := s.ensureBridgeLocked(bridgeArn)
		bridge["UpdatedAt"] = now
		return map[string]any{"BridgeArn": bridgeArn, "OutputArn": outputArn, "OutputName": outputName, "SourceArn": sourceArn, "SourceName": sourceName}

	case "DescribeFlowSourceMetadata":
		return map[string]any{"FlowArn": flowArn, "SourceName": sourceName, "Resolution": "1920x1080", "FrameRate": "29.97"}
	case "DescribeFlowSourceThumbnail":
		return map[string]any{"ThumbnailData": "c3RhY2t5YXJk", "ContentType": "image/jpeg", "FlowArn": flowArn, "SourceName": sourceName}

	case "TagResource":
		tags := s.ensureResourceTagsLocked(resourceArn)
		for k, v := range mcStringMapAny(payload, "Tags", "tags") {
			tags[k] = v
		}
		return map[string]any{"ResourceArn": resourceArn, "Tags": mcCloneStringMap(tags)}
	case "UntagResource":
		tags := s.ensureResourceTagsLocked(resourceArn)
		for _, key := range mcStringSliceAny(payload, "TagKeys", "tagKeys") {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			if strings.TrimSpace(key) != "" {
				delete(tags, strings.TrimSpace(key))
			}
		}
		return map[string]any{"ResourceArn": resourceArn, "Tags": mcCloneStringMap(tags)}
	case "ListTagsForResource":
		return map[string]any{"Tags": mcCloneStringMap(s.ensureResourceTagsLocked(resourceArn))}

	case "TagGlobalResource":
		tags := s.ensureGlobalResourceTagsLocked(resourceArn)
		for k, v := range mcStringMapAny(payload, "Tags", "tags") {
			tags[k] = v
		}
		return map[string]any{"ResourceArn": resourceArn, "Tags": mcCloneStringMap(tags)}
	case "UntagGlobalResource":
		tags := s.ensureGlobalResourceTagsLocked(resourceArn)
		for _, key := range mcStringSliceAny(payload, "TagKeys", "tagKeys") {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			if strings.TrimSpace(key) != "" {
				delete(tags, strings.TrimSpace(key))
			}
		}
		return map[string]any{"ResourceArn": resourceArn, "Tags": mcCloneStringMap(tags)}
	case "ListTagsForGlobalResource":
		return map[string]any{"Tags": mcCloneStringMap(s.ensureGlobalResourceTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *mediaConnectStore) nextFlowIDLocked() int64 {
	id := s.nextFlowID
	s.nextFlowID++
	return id
}

func (s *mediaConnectStore) nextBridgeIDLocked() int64 {
	id := s.nextBridgeID
	s.nextBridgeID++
	return id
}

func (s *mediaConnectStore) nextGatewayIDLocked() int64 {
	id := s.nextGatewayID
	s.nextGatewayID++
	return id
}

func (s *mediaConnectStore) nextRouterInputIDLocked() int64 {
	id := s.nextRouterInputID
	s.nextRouterInputID++
	return id
}

func (s *mediaConnectStore) nextRouterOutputIDLocked() int64 {
	id := s.nextRouterOutputID
	s.nextRouterOutputID++
	return id
}

func (s *mediaConnectStore) nextRouterIfaceIDLocked() int64 {
	id := s.nextRouterIfaceID
	s.nextRouterIfaceID++
	return id
}

func (s *mediaConnectStore) nextEntitlementIDLocked() int64 {
	id := s.nextEntitlementID
	s.nextEntitlementID++
	return id
}

func (s *mediaConnectStore) nextOfferingIDLocked() int64 {
	id := s.nextOfferingID
	s.nextOfferingID++
	return id
}

func (s *mediaConnectStore) nextReservationIDLocked() int64 {
	id := s.nextReservationID
	s.nextReservationID++
	return id
}

func mcARN(kind, id string) string {
	return fmt.Sprintf("arn:aws:mediaconnect:us-east-1:123456789012:%s/%s", kind, id)
}

func (s *mediaConnectStore) ensureFlowLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("flow", "flow-00000001"))
	if v, ok := s.flows[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":      "flow",
		"FlowArn":   arn,
		"Name":      "stackyard-flow",
		"Status":    "STANDBY",
		"CreatedAt": now,
		"UpdatedAt": now,
	}
	s.flows[arn] = m
	return m
}

func (s *mediaConnectStore) ensureBridgeLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("bridge", "bridge-00000001"))
	if v, ok := s.bridges[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":      "bridge",
		"BridgeArn": arn,
		"Name":      "stackyard-bridge",
		"State":     "ACTIVE",
		"CreatedAt": now,
		"UpdatedAt": now,
	}
	s.bridges[arn] = m
	return m
}

func (s *mediaConnectStore) ensureGatewayLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("gateway", "gateway-00000001"))
	if v, ok := s.gateways[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":       "gateway",
		"GatewayArn": arn,
		"Name":       "stackyard-gateway",
		"State":      "ACTIVE",
		"CreatedAt":  now,
		"UpdatedAt":  now,
	}
	s.gateways[arn] = m
	return m
}

func (s *mediaConnectStore) ensureGatewayInstanceLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("gateway-instance", "gateway-instance-00000001"))
	if v, ok := s.gatewayInstances[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":               "gateway-instance",
		"GatewayInstanceArn": arn,
		"Status":             "ACTIVE",
		"CreatedAt":          now,
		"UpdatedAt":          now,
	}
	s.gatewayInstances[arn] = m
	return m
}

func (s *mediaConnectStore) ensureRouterInputLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("router-input", "router-input-00000001"))
	if v, ok := s.routerInputs[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":           "router-input",
		"RouterInputArn": arn,
		"Name":           "stackyard-router-input",
		"State":          "STANDBY",
		"CreatedAt":      now,
		"UpdatedAt":      now,
	}
	s.routerInputs[arn] = m
	return m
}

func (s *mediaConnectStore) ensureRouterOutputLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("router-output", "router-output-00000001"))
	if v, ok := s.routerOutputs[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":            "router-output",
		"RouterOutputArn": arn,
		"Name":            "stackyard-router-output",
		"State":           "STANDBY",
		"CreatedAt":       now,
		"UpdatedAt":       now,
	}
	s.routerOutputs[arn] = m
	return m
}

func (s *mediaConnectStore) ensureRouterIfaceLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("router-interface", "router-interface-00000001"))
	if v, ok := s.routerIfaces[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":                      "router-interface",
		"RouterNetworkInterfaceArn": arn,
		"Name":                      "stackyard-router-interface",
		"CreatedAt":                 now,
		"UpdatedAt":                 now,
	}
	s.routerIfaces[arn] = m
	return m
}

func (s *mediaConnectStore) ensureEntitlementLocked(arn, flowArn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("entitlement", "entitlement-00000001"))
	flowArn = mcFirstNonEmpty(strings.TrimSpace(flowArn), mcARN("flow", "flow-00000001"))
	if v, ok := s.entitlements[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":           "entitlement",
		"EntitlementArn": arn,
		"FlowArn":        flowArn,
		"Name":           "stackyard-entitlement",
		"CreatedAt":      now,
		"UpdatedAt":      now,
	}
	s.entitlements[arn] = m
	return m
}

func (s *mediaConnectStore) ensureOfferingLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("offering", "offering-00000001"))
	if v, ok := s.offerings[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":        "offering",
		"OfferingArn": arn,
		"Name":        "stackyard-offering",
		"CreatedAt":   now,
		"UpdatedAt":   now,
	}
	s.offerings[arn] = m
	return m
}

func (s *mediaConnectStore) ensureReservationLocked(arn string) map[string]any {
	arn = mcFirstNonEmpty(strings.TrimSpace(arn), mcARN("reservation", "reservation-00000001"))
	if v, ok := s.reservations[arn]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := map[string]any{
		"type":           "reservation",
		"ReservationArn": arn,
		"Name":           "stackyard-reservation",
		"State":          "ACTIVE",
		"CreatedAt":      now,
		"UpdatedAt":      now,
	}
	s.reservations[arn] = m
	return m
}

func (s *mediaConnectStore) listByTypeLocked(resourceType string) []any {
	collect := []map[string]any{}
	appendMap := func(in map[string]map[string]any) {
		for _, v := range in {
			if strings.EqualFold(mcStringAny(v, "type"), resourceType) {
				collect = append(collect, mcCloneMap(v))
			}
		}
	}

	switch resourceType {
	case "flow":
		appendMap(s.flows)
	case "bridge":
		appendMap(s.bridges)
	case "gateway":
		appendMap(s.gateways)
	case "gateway-instance":
		appendMap(s.gatewayInstances)
	case "router-input":
		appendMap(s.routerInputs)
	case "router-output":
		appendMap(s.routerOutputs)
	case "router-interface":
		appendMap(s.routerIfaces)
	case "entitlement":
		appendMap(s.entitlements)
	case "offering":
		appendMap(s.offerings)
	case "reservation":
		appendMap(s.reservations)
	}

	sort.SliceStable(collect, func(i, j int) bool {
		ai := mcFirstNonEmpty(
			mcStringAny(collect[i], "FlowArn"),
			mcStringAny(collect[i], "BridgeArn"),
			mcStringAny(collect[i], "GatewayArn"),
			mcStringAny(collect[i], "GatewayInstanceArn"),
			mcStringAny(collect[i], "RouterInputArn"),
			mcStringAny(collect[i], "RouterOutputArn"),
			mcStringAny(collect[i], "RouterNetworkInterfaceArn"),
			mcStringAny(collect[i], "EntitlementArn"),
			mcStringAny(collect[i], "OfferingArn"),
			mcStringAny(collect[i], "ReservationArn"),
		)
		aj := mcFirstNonEmpty(
			mcStringAny(collect[j], "FlowArn"),
			mcStringAny(collect[j], "BridgeArn"),
			mcStringAny(collect[j], "GatewayArn"),
			mcStringAny(collect[j], "GatewayInstanceArn"),
			mcStringAny(collect[j], "RouterInputArn"),
			mcStringAny(collect[j], "RouterOutputArn"),
			mcStringAny(collect[j], "RouterNetworkInterfaceArn"),
			mcStringAny(collect[j], "EntitlementArn"),
			mcStringAny(collect[j], "OfferingArn"),
			mcStringAny(collect[j], "ReservationArn"),
		)
		return ai < aj
	})

	out := make([]any, 0, len(collect))
	for _, v := range collect {
		out = append(out, v)
	}
	return out
}

func (s *mediaConnectStore) ensureResourceTagsLocked(resourceArn string) map[string]string {
	resourceArn = mcFirstNonEmpty(strings.TrimSpace(resourceArn), mcARN("flow", "flow-00000001"))
	if tags, ok := s.resourceTags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{}
	s.resourceTags[resourceArn] = tags
	return tags
}

func (s *mediaConnectStore) ensureGlobalResourceTagsLocked(resourceArn string) map[string]string {
	resourceArn = mcFirstNonEmpty(strings.TrimSpace(resourceArn), mcARN("gateway-instance", "gateway-instance-00000001"))
	if tags, ok := s.globalResourceTags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{}
	s.globalResourceTags[resourceArn] = tags
	return tags
}

func (s *mediaConnectStore) bumpFlowSubresourceStateLocked(flowArn, action, updatedAt string) {
	flowArn = mcFirstNonEmpty(flowArn, mcARN("flow", "flow-00000001"))
	state, ok := s.flowSubresourceState[flowArn]
	if !ok {
		state = map[string]map[string]any{}
		s.flowSubresourceState[flowArn] = state
	}
	entry := map[string]any{
		"FlowArn":    flowArn,
		"Action":     action,
		"UpdatedAt":  updatedAt,
		"Invocation": time.Now().UTC().UnixNano(),
	}
	state[action] = entry
}

func mcPathParam(pathParams map[string]string, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := pathParams[key]; ok {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func mcStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		for existing, value := range m {
			if !strings.EqualFold(existing, key) {
				continue
			}
			if s, ok := value.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func mcFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func mcCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = mcCloneMap(typed)
		case map[string]string:
			cp := make(map[string]string, len(typed))
			for kk, vv := range typed {
				cp[kk] = vv
			}
			out[k] = cp
		case []any:
			cp := make([]any, 0, len(typed))
			for _, item := range typed {
				if m, ok := item.(map[string]any); ok {
					cp = append(cp, mcCloneMap(m))
					continue
				}
				cp = append(cp, item)
			}
			out[k] = cp
		default:
			out[k] = v
		}
	}
	return out
}

func mcStringMapAny(payload map[string]any, keys ...string) map[string]string {
	out := map[string]string{}
	for _, key := range keys {
		for existing, value := range payload {
			if !strings.EqualFold(existing, key) {
				continue
			}
			if m, ok := value.(map[string]any); ok {
				for kk, vv := range m {
					if s, ok := vv.(string); ok && strings.TrimSpace(kk) != "" {
						out[strings.TrimSpace(kk)] = strings.TrimSpace(s)
					}
				}
			}
			if m, ok := value.(map[string]string); ok {
				for kk, vv := range m {
					if strings.TrimSpace(kk) != "" {
						out[strings.TrimSpace(kk)] = strings.TrimSpace(vv)
					}
				}
			}
		}
	}
	return out
}

func mcStringSliceAny(payload map[string]any, keys ...string) []string {
	out := []string{}
	for _, key := range keys {
		for existing, value := range payload {
			if !strings.EqualFold(existing, key) {
				continue
			}
			if items, ok := value.([]any); ok {
				for _, item := range items {
					if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
						out = append(out, strings.TrimSpace(s))
					}
				}
			}
			if items, ok := value.([]string); ok {
				for _, item := range items {
					item = strings.TrimSpace(item)
					if item != "" {
						out = append(out, item)
					}
				}
			}
		}
	}
	return out
}

func mcCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
