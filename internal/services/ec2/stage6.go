package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type NatGatewayAddress struct {
	AllocationID       string
	AssociationID      string
	NetworkInterfaceID string
	PrivateIP          string
	PublicIP           string
	Status             string
	IsPrimary          bool
}

type NatGateway struct {
	ID               string
	VpcID            string
	SubnetID         string
	State            string
	ConnectivityType string
	CreateTime       time.Time
	DeleteTime       *time.Time
	Addresses        []NatGatewayAddress
	Tags             map[string]string
}

type PeeringConnectionOptions struct {
	AllowDNSResolutionFromRemoteVPC            bool
	AllowEgressFromLocalClassicLinkToRemoteVPC bool
	AllowEgressFromLocalVPCToRemoteClassicLink bool
}

type VpcPeeringConnection struct {
	ID                 string
	RequesterVpcID     string
	RequesterCidrBlock string
	AccepterVpcID      string
	AccepterCidrBlock  string
	RequesterOptions   PeeringConnectionOptions
	AccepterOptions    PeeringConnectionOptions
	StatusCode         string
	StatusMessage      string
	Tags               map[string]string
}

func (s *Service) CreateNatGateway(subnetID, allocationID, connectivityType string, tags []Tag) (NatGateway, error) {
	subnetID = strings.TrimSpace(subnetID)
	allocationID = strings.TrimSpace(allocationID)
	connectivityType = strings.ToLower(strings.TrimSpace(connectivityType))
	if subnetID == "" {
		return NatGateway{}, ErrInvalidParameter
	}
	if connectivityType == "" {
		connectivityType = "public"
	}
	if connectivityType != "public" && connectivityType != "private" {
		return NatGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subnet := s.subnets[subnetID]
	if subnet == nil {
		return NatGateway{}, ErrNotFound
	}

	gateway := &NatGateway{
		ID:               s.nextIDLocked("nat"),
		VpcID:            subnet.VpcID,
		SubnetID:         subnet.ID,
		State:            "available",
		ConnectivityType: connectivityType,
		CreateTime:       time.Now().UTC(),
		Addresses:        []NatGatewayAddress{},
		Tags:             tagsToMap(tags),
	}

	if connectivityType == "public" {
		if allocationID == "" {
			return NatGateway{}, ErrInvalidParameter
		}
		address := s.addresses[allocationID]
		if address == nil {
			return NatGateway{}, ErrNotFound
		}
		if address.AssociationID != "" {
			return NatGateway{}, ErrConflict
		}
		associationID := s.nextIDLocked("eipassoc")
		ifaceID := fmt.Sprintf("eni-%s", strings.TrimPrefix(gateway.ID, "nat-"))
		privateIP := fmt.Sprintf("10.0.1.%d", int(s.seq%250)+1)
		address.AssociationID = associationID
		address.NetworkInterfaceID = ifaceID
		address.PrivateIPAddress = privateIP
		address.InstanceID = ""
		gateway.Addresses = append(gateway.Addresses, NatGatewayAddress{
			AllocationID:       address.AllocationID,
			AssociationID:      associationID,
			NetworkInterfaceID: ifaceID,
			PrivateIP:          privateIP,
			PublicIP:           address.PublicIP,
			Status:             "succeeded",
			IsPrimary:          true,
		})
	}

	s.natGateways[gateway.ID] = gateway
	return cloneNatGateway(gateway), nil
}

func (s *Service) DescribeNatGateways(gatewayIDs, vpcIDs, subnetIDs []string) []NatGateway {
	s.mu.Lock()
	defer s.mu.Unlock()

	gatewaySet := toStringSet(gatewayIDs)
	vpcSet := toStringSet(vpcIDs)
	subnetSet := toStringSet(subnetIDs)

	out := make([]NatGateway, 0, len(s.natGateways))
	for _, gateway := range s.natGateways {
		if len(gatewaySet) > 0 {
			if _, ok := gatewaySet[gateway.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			if _, ok := vpcSet[gateway.VpcID]; !ok {
				continue
			}
		}
		if len(subnetSet) > 0 {
			if _, ok := subnetSet[gateway.SubnetID]; !ok {
				continue
			}
		}
		out = append(out, cloneNatGateway(gateway))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteNatGateway(gatewayID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.natGateways[gatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	if gateway.State == "deleted" {
		return nil
	}
	now := time.Now().UTC()
	gateway.State = "deleted"
	gateway.DeleteTime = &now
	for _, ref := range gateway.Addresses {
		address := s.addresses[ref.AllocationID]
		if address == nil {
			continue
		}
		if ref.AssociationID != "" && address.AssociationID != ref.AssociationID {
			continue
		}
		address.AssociationID = ""
		address.InstanceID = ""
		address.NetworkInterfaceID = ""
		address.PrivateIPAddress = ""
	}
	return nil
}

func (s *Service) CreateVpcPeeringConnection(vpcID, peerVpcID string, tags []Tag) (VpcPeeringConnection, error) {
	vpcID = strings.TrimSpace(vpcID)
	peerVpcID = strings.TrimSpace(peerVpcID)
	if vpcID == "" || peerVpcID == "" || vpcID == peerVpcID {
		return VpcPeeringConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	requester := s.vpcs[vpcID]
	accepter := s.vpcs[peerVpcID]
	if requester == nil || accepter == nil {
		return VpcPeeringConnection{}, ErrNotFound
	}

	connection := &VpcPeeringConnection{
		ID:                 s.nextIDLocked("pcx"),
		RequesterVpcID:     requester.ID,
		RequesterCidrBlock: requester.CidrBlock,
		AccepterVpcID:      accepter.ID,
		AccepterCidrBlock:  accepter.CidrBlock,
		RequesterOptions:   PeeringConnectionOptions{},
		AccepterOptions:    PeeringConnectionOptions{},
		StatusCode:         "pending-acceptance",
		StatusMessage:      "Pending Acceptance",
		Tags:               tagsToMap(tags),
	}
	s.vpcPeeringConnections[connection.ID] = connection
	return cloneVpcPeeringConnection(connection), nil
}

func (s *Service) DescribeVpcPeeringConnections(connectionIDs, requesterVpcIDs, accepterVpcIDs, statusCodes []string) []VpcPeeringConnection {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(connectionIDs)
	requesterSet := toStringSet(requesterVpcIDs)
	accepterSet := toStringSet(accepterVpcIDs)
	statusSet := map[string]struct{}{}
	for _, status := range statusCodes {
		status = strings.ToLower(strings.TrimSpace(status))
		if status == "" {
			continue
		}
		statusSet[status] = struct{}{}
	}

	out := make([]VpcPeeringConnection, 0, len(s.vpcPeeringConnections))
	for _, connection := range s.vpcPeeringConnections {
		if len(idSet) > 0 {
			if _, ok := idSet[connection.ID]; !ok {
				continue
			}
		}
		if len(requesterSet) > 0 {
			if _, ok := requesterSet[connection.RequesterVpcID]; !ok {
				continue
			}
		}
		if len(accepterSet) > 0 {
			if _, ok := accepterSet[connection.AccepterVpcID]; !ok {
				continue
			}
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[strings.ToLower(connection.StatusCode)]; !ok {
				continue
			}
		}
		out = append(out, cloneVpcPeeringConnection(connection))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AcceptVpcPeeringConnection(connectionID string) (VpcPeeringConnection, error) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return VpcPeeringConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpcPeeringConnections[connectionID]
	if connection == nil {
		return VpcPeeringConnection{}, ErrNotFound
	}
	if connection.StatusCode == "deleted" {
		return VpcPeeringConnection{}, ErrConflict
	}
	if connection.StatusCode == "rejected" {
		return VpcPeeringConnection{}, ErrConflict
	}
	connection.StatusCode = "active"
	connection.StatusMessage = "Active"
	return cloneVpcPeeringConnection(connection), nil
}

func (s *Service) RejectVpcPeeringConnection(connectionID string) (VpcPeeringConnection, error) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return VpcPeeringConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpcPeeringConnections[connectionID]
	if connection == nil {
		return VpcPeeringConnection{}, ErrNotFound
	}
	if connection.StatusCode == "deleted" {
		return VpcPeeringConnection{}, ErrConflict
	}
	if connection.StatusCode == "active" {
		return VpcPeeringConnection{}, ErrConflict
	}
	connection.StatusCode = "rejected"
	connection.StatusMessage = "Rejected"
	return cloneVpcPeeringConnection(connection), nil
}

func (s *Service) DeleteVpcPeeringConnection(connectionID string) (VpcPeeringConnection, error) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return VpcPeeringConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpcPeeringConnections[connectionID]
	if connection == nil {
		return VpcPeeringConnection{}, ErrNotFound
	}
	connection.StatusCode = "deleted"
	connection.StatusMessage = "Deleted"
	return cloneVpcPeeringConnection(connection), nil
}

func cloneNatGateway(in *NatGateway) NatGateway {
	if in == nil {
		return NatGateway{}
	}
	addresses := make([]NatGatewayAddress, 0, len(in.Addresses))
	for _, address := range in.Addresses {
		addresses = append(addresses, address)
	}
	tags := map[string]string{}
	for key, value := range in.Tags {
		tags[key] = value
	}
	var deleteTime *time.Time
	if in.DeleteTime != nil {
		clone := *in.DeleteTime
		deleteTime = &clone
	}
	return NatGateway{
		ID:               in.ID,
		VpcID:            in.VpcID,
		SubnetID:         in.SubnetID,
		State:            in.State,
		ConnectivityType: in.ConnectivityType,
		CreateTime:       in.CreateTime,
		DeleteTime:       deleteTime,
		Addresses:        addresses,
		Tags:             tags,
	}
}

func cloneVpcPeeringConnection(in *VpcPeeringConnection) VpcPeeringConnection {
	if in == nil {
		return VpcPeeringConnection{}
	}
	tags := map[string]string{}
	for key, value := range in.Tags {
		tags[key] = value
	}
	return VpcPeeringConnection{
		ID:                 in.ID,
		RequesterVpcID:     in.RequesterVpcID,
		RequesterCidrBlock: in.RequesterCidrBlock,
		AccepterVpcID:      in.AccepterVpcID,
		AccepterCidrBlock:  in.AccepterCidrBlock,
		RequesterOptions:   in.RequesterOptions,
		AccepterOptions:    in.AccepterOptions,
		StatusCode:         in.StatusCode,
		StatusMessage:      in.StatusMessage,
		Tags:               tags,
	}
}
