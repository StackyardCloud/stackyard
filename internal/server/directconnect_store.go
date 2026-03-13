package server

import (
	"fmt"
	"strings"
	"sync"
)

type directConnectStore struct {
	mu     sync.Mutex
	nextID int64
}

func newDirectConnectStore() *directConnectStore {
	return &directConnectStore{nextID: 1}
}

func (s *directConnectStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "DescribeConnections":
		return map[string]any{"connections": []any{s.connectionPayload("dxcon-00000001")}}
	case "DescribeConnectionsOnInterconnect":
		return map[string]any{"connections": []any{s.connectionPayload("dxcon-00000001")}}
	case "DescribeHostedConnections":
		return map[string]any{"connections": []any{s.connectionPayload("dxcon-00000002")}}
	case "DescribeInterconnects":
		return map[string]any{"interconnects": []any{s.interconnectPayload("dxi-00000001")}}
	case "DescribeLags":
		return map[string]any{"lags": []any{s.lagPayload("dxlag-00000001")}}
	case "DescribeLocations":
		return map[string]any{"locations": []any{s.locationPayload()}}
	case "DescribeVirtualInterfaces":
		return map[string]any{"virtualInterfaces": []any{s.virtualInterfacePayload("dxvif-00000001")}}
	case "DescribeVirtualGateways":
		return map[string]any{"virtualGateways": []any{s.virtualGatewayPayload()}}
	case "DescribeDirectConnectGateways":
		return map[string]any{"directConnectGateways": []any{s.directConnectGatewayPayload("dxgw-00000001")}}
	case "DescribeDirectConnectGatewayAssociations":
		return map[string]any{"directConnectGatewayAssociations": []any{s.gatewayAssociationPayload("dxgwassoc-00000001")}}
	case "DescribeDirectConnectGatewayAssociationProposals":
		return map[string]any{"directConnectGatewayAssociationProposals": []any{s.gatewayAssociationProposalPayload("dxgwassocprop-00000001")}}
	case "DescribeDirectConnectGatewayAttachments":
		return map[string]any{"directConnectGatewayAttachments": []any{s.gatewayAttachmentPayload("dxgwattach-00000001")}}
	case "DescribeTags":
		arn := directConnectPayloadString(payload, "resourceArn", directConnectConnectionARN("dxcon-00000001"))
		return map[string]any{"resourceTags": []any{map[string]any{"resourceArn": arn, "tags": []any{map[string]any{"key": "stackyard", "value": "true"}}}}}
	case "DescribeRouterConfiguration":
		return map[string]any{"customerRouterConfig": "router bgp 65000\n neighbor 169.254.0.1 remote-as 7224"}
	case "DescribeConnectionLoa", "DescribeInterconnectLoa":
		return map[string]any{"loa": map[string]any{"loaContent": "c3RhY2t5YXJk", "loaContentType": "application/pdf"}}
	case "DescribeLoa":
		return map[string]any{"loaContent": "c3RhY2t5YXJk", "loaContentType": "application/pdf"}
	case "DescribeCustomerMetadata":
		return map[string]any{"agreements": []any{map[string]any{"status": "SIGNED", "agreementName": "AWS Customer Agreement"}}}

	case "CreateConnection", "AllocateConnectionOnInterconnect", "AllocateHostedConnection", "AssociateConnectionWithLag", "DisassociateConnectionFromLag", "UpdateConnection":
		id := directConnectPayloadString(payload, "connectionId", "")
		if id == "" || strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Allocate") {
			id = s.nextTokenLocked("dxcon", 8)
		}
		return s.connectionPayload(id)
	case "DeleteConnection":
		id := directConnectPayloadString(payload, "connectionId", "dxcon-00000001")
		conn := s.connectionPayload(id)
		conn["connectionState"] = "deleted"
		return conn
	case "ConfirmConnection":
		return map[string]any{"connectionState": "available"}

	case "CreateInterconnect":
		id := s.nextTokenLocked("dxi", 8)
		return s.interconnectPayload(id)
	case "DeleteInterconnect":
		return map[string]any{"interconnectState": "deleted"}

	case "CreateLag", "UpdateLag":
		id := directConnectPayloadString(payload, "lagId", "")
		if id == "" || action == "CreateLag" {
			id = s.nextTokenLocked("dxlag", 8)
		}
		return s.lagPayload(id)
	case "DeleteLag":
		id := directConnectPayloadString(payload, "lagId", "dxlag-00000001")
		lag := s.lagPayload(id)
		lag["lagState"] = "deleted"
		return lag

	case "CreatePrivateVirtualInterface", "CreatePublicVirtualInterface",
		"AllocatePrivateVirtualInterface", "AllocatePublicVirtualInterface",
		"AssociateVirtualInterface", "UpdateVirtualInterfaceAttributes", "CreateBGPPeer", "DeleteBGPPeer":
		id := directConnectPayloadString(payload, "virtualInterfaceId", "")
		if id == "" {
			id = s.nextTokenLocked("dxvif", 8)
		}
		return s.virtualInterfacePayload(id)
	case "CreateTransitVirtualInterface", "AllocateTransitVirtualInterface":
		id := directConnectPayloadString(payload, "virtualInterfaceId", "")
		if id == "" {
			id = s.nextTokenLocked("dxvif", 8)
		}
		return map[string]any{"virtualInterface": s.virtualInterfacePayloadWithType(id, "transit")}
	case "DeleteVirtualInterface":
		return map[string]any{"virtualInterfaceState": "deleted"}
	case "ConfirmPrivateVirtualInterface", "ConfirmPublicVirtualInterface", "ConfirmTransitVirtualInterface":
		return map[string]any{"virtualInterfaceState": "available"}
	case "ConfirmCustomerAgreement":
		return map[string]any{"status": "signed"}

	case "CreateDirectConnectGateway", "UpdateDirectConnectGateway":
		id := directConnectPayloadString(payload, "directConnectGatewayId", "")
		if id == "" || action == "CreateDirectConnectGateway" {
			id = s.nextTokenLocked("dxgw", 8)
		}
		return map[string]any{"directConnectGateway": s.directConnectGatewayPayload(id)}
	case "DeleteDirectConnectGateway":
		id := directConnectPayloadString(payload, "directConnectGatewayId", "dxgw-00000001")
		gw := s.directConnectGatewayPayload(id)
		gw["directConnectGatewayState"] = "deleted"
		return map[string]any{"directConnectGateway": gw}

	case "CreateDirectConnectGatewayAssociation", "AcceptDirectConnectGatewayAssociationProposal", "UpdateDirectConnectGatewayAssociation":
		id := s.nextTokenLocked("dxgwassoc", 8)
		return map[string]any{"directConnectGatewayAssociation": s.gatewayAssociationPayload(id)}
	case "DeleteDirectConnectGatewayAssociation":
		id := directConnectPayloadString(payload, "associationId", s.nextTokenLocked("dxgwassoc", 8))
		assoc := s.gatewayAssociationPayload(id)
		assoc["associationState"] = "disassociated"
		return map[string]any{"directConnectGatewayAssociation": assoc}

	case "CreateDirectConnectGatewayAssociationProposal":
		id := s.nextTokenLocked("dxgwassocprop", 8)
		return map[string]any{"directConnectGatewayAssociationProposal": s.gatewayAssociationProposalPayload(id)}
	case "DeleteDirectConnectGatewayAssociationProposal":
		id := directConnectPayloadString(payload, "proposalId", s.nextTokenLocked("dxgwassocprop", 8))
		proposal := s.gatewayAssociationProposalPayload(id)
		proposal["proposalState"] = "deleted"
		return map[string]any{"directConnectGatewayAssociationProposal": proposal}

	case "ListVirtualInterfaceTestHistory":
		return map[string]any{"virtualInterfaceTestHistory": []any{s.virtualInterfaceTestPayload("dxvif-00000001")}, "nextToken": ""}
	case "StartBgpFailoverTest", "StopBgpFailoverTest":
		return map[string]any{"virtualInterfaceTest": s.virtualInterfaceTestPayload(directConnectPayloadString(payload, "virtualInterfaceId", "dxvif-00000001"))}

	case "AssociateHostedConnection":
		return map[string]any{
			"connectionId":       directConnectPayloadString(payload, "connectionId", "dxcon-00000002"),
			"virtualGatewayId":   directConnectPayloadString(payload, "virtualGatewayId", "vgw-00000001"),
			"virtualInterfaceId": directConnectPayloadString(payload, "virtualInterfaceId", "dxvif-00000002"),
		}
	case "AssociateMacSecKey", "DisassociateMacSecKey":
		return map[string]any{"connectionId": directConnectPayloadString(payload, "connectionId", "dxcon-00000001"), "macSecKeys": []any{}}

	case "TagResource", "UntagResource":
		return map[string]any{}
	}

	switch {
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{}
	case strings.HasPrefix(action, "List"):
		return map[string]any{"nextToken": ""}
	case strings.HasPrefix(action, "Create"), strings.HasPrefix(action, "Allocate"), strings.HasPrefix(action, "Associate"), strings.HasPrefix(action, "Disassociate"), strings.HasPrefix(action, "Confirm"), strings.HasPrefix(action, "Update"), strings.HasPrefix(action, "Delete"), strings.HasPrefix(action, "Start"), strings.HasPrefix(action, "Stop"), strings.HasPrefix(action, "Tag"), strings.HasPrefix(action, "Untag"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *directConnectStore) nextTokenLocked(prefix string, width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func (s *directConnectStore) connectionPayload(id string) map[string]any {
	if id == "" {
		id = "dxcon-00000001"
	}
	return map[string]any{
		"connectionId":    id,
		"connectionName":  "stackyard-directconnect-connection",
		"connectionState": "available",
		"location":        "EqSV5",
		"bandwidth":       "1Gbps",
		"ownerAccount":    "123456789012",
		"region":          "us-east-1",
		"connectionArn":   directConnectConnectionARN(id),
	}
}

func (s *directConnectStore) interconnectPayload(id string) map[string]any {
	if id == "" {
		id = "dxi-00000001"
	}
	return map[string]any{
		"interconnectId":    id,
		"interconnectName":  "stackyard-directconnect-interconnect",
		"interconnectState": "available",
		"bandwidth":         "10Gbps",
		"location":          "EqSV5",
		"region":            "us-east-1",
	}
}

func (s *directConnectStore) lagPayload(id string) map[string]any {
	if id == "" {
		id = "dxlag-00000001"
	}
	return map[string]any{
		"lagId":            id,
		"lagName":          "stackyard-directconnect-lag",
		"lagState":         "available",
		"location":         "EqSV5",
		"connections":      []any{s.connectionPayload("dxcon-00000001")},
		"connectionsLagId": id,
	}
}

func (s *directConnectStore) virtualInterfacePayload(id string) map[string]any {
	return s.virtualInterfacePayloadWithType(id, "private")
}

func (s *directConnectStore) virtualInterfacePayloadWithType(id string, vifType string) map[string]any {
	if id == "" {
		id = "dxvif-00000001"
	}
	if vifType == "" {
		vifType = "private"
	}
	virtualGatewayID := "vgw-00000001"
	if vifType == "transit" {
		virtualGatewayID = ""
	}
	return map[string]any{
		"virtualInterfaceId":     id,
		"virtualInterfaceName":   "stackyard-directconnect-vif",
		"virtualInterfaceType":   vifType,
		"virtualInterfaceState":  "available",
		"connectionId":           "dxcon-00000001",
		"ownerAccount":           "123456789012",
		"vlan":                   101,
		"asn":                    64512,
		"amazonAddress":          "169.254.0.1/30",
		"customerAddress":        "169.254.0.2/30",
		"virtualGatewayId":       virtualGatewayID,
		"directConnectGatewayId": "dxgw-00000001",
	}
}

func (s *directConnectStore) directConnectGatewayPayload(id string) map[string]any {
	if id == "" {
		id = "dxgw-00000001"
	}
	return map[string]any{
		"directConnectGatewayId":    id,
		"directConnectGatewayName":  "stackyard-directconnect-gateway",
		"directConnectGatewayState": "available",
		"ownerAccount":              "123456789012",
		"directConnectGatewayArn":   directConnectGatewayARN(id),
	}
}

func (s *directConnectStore) gatewayAssociationPayload(id string) map[string]any {
	if id == "" {
		id = "dxgwassoc-00000001"
	}
	return map[string]any{
		"associationId":                         id,
		"associationState":                      "associated",
		"directConnectGatewayId":                "dxgw-00000001",
		"associatedGateway":                     map[string]any{"id": "vgw-00000001", "type": "virtualPrivateGateway", "ownerAccount": "123456789012", "region": "us-east-1"},
		"allowedPrefixesToDirectConnectGateway": []any{},
	}
}

func (s *directConnectStore) gatewayAssociationProposalPayload(id string) map[string]any {
	if id == "" {
		id = "dxgwassocprop-00000001"
	}
	return map[string]any{
		"proposalId":                            id,
		"proposalState":                         "requested",
		"directConnectGatewayId":                "dxgw-00000001",
		"associatedGateway":                     map[string]any{"id": "vgw-00000001", "type": "virtualPrivateGateway", "ownerAccount": "123456789012", "region": "us-east-1"},
		"allowedPrefixesToDirectConnectGateway": []any{},
	}
}

func (s *directConnectStore) gatewayAttachmentPayload(id string) map[string]any {
	if id == "" {
		id = "dxgwattach-00000001"
	}
	return map[string]any{
		"directConnectGatewayAttachmentId": id,
		"directConnectGatewayId":           "dxgw-00000001",
		"state":                            "attached",
		"virtualInterfaceOwner":            "123456789012",
		"virtualInterfaceRegion":           "us-east-1",
	}
}

func (s *directConnectStore) locationPayload() map[string]any {
	return map[string]any{
		"locationCode":        "EqSV5",
		"locationName":        "Seattle, WA",
		"region":              "us-west-2",
		"availablePortSpeeds": []any{"1Gbps", "10Gbps"},
	}
}

func (s *directConnectStore) virtualGatewayPayload() map[string]any {
	return map[string]any{
		"virtualGatewayId":    "vgw-00000001",
		"virtualGatewayState": "available",
	}
}

func (s *directConnectStore) virtualInterfaceTestPayload(virtualInterfaceID string) map[string]any {
	if virtualInterfaceID == "" {
		virtualInterfaceID = "dxvif-00000001"
	}
	return map[string]any{
		"virtualInterfaceId": virtualInterfaceID,
		"testId":             s.nextTokenLocked("dxviftest", 8),
		"bgpPeers":           []any{},
		"status":             "running",
	}
}

func directConnectPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" {
				return s
			}
			break
		}
	}
	return fallback
}

func directConnectConnectionARN(connectionID string) string {
	return "arn:aws:directconnect:us-east-1:123456789012:dxcon/" + connectionID
}

func directConnectGatewayARN(gatewayID string) string {
	return "arn:aws:directconnect::123456789012:dx-gateway/" + gatewayID
}
