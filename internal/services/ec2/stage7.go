package ec2

import (
	"fmt"
	"sort"
	"strings"
)

type PeeringConnectionOptionsPatch struct {
	AllowDNSResolutionFromRemoteVPC            *bool
	AllowEgressFromLocalClassicLinkToRemoteVPC *bool
	AllowEgressFromLocalVPCToRemoteClassicLink *bool
}

type NetworkInterfacePermission struct {
	ID                 string
	NetworkInterfaceID string
	Permission         string
	AwsAccountID       string
	AwsService         string
	State              string
	StatusMessage      string
}

type NetworkInterfaceAttributeDescription struct {
	NetworkInterfaceID       string
	AssociatePublicIPAddress bool
	Description              string
	GroupIDs                 []string
	SourceDestCheck          bool
	Attachment               *NetworkInterfaceAttachment
}

func (s *Service) AssignPrivateNatGatewayAddress(gatewayID string, privateIPs []string, privateIPCount int32) ([]NatGatewayAddress, error) {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return nil, ErrInvalidParameter
	}
	if len(privateIPs) > 0 && privateIPCount > 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.natGateways[gatewayID]
	if gateway == nil {
		return nil, ErrNotFound
	}
	if gateway.State == "deleted" {
		return nil, ErrConflict
	}

	trimmed := make([]string, 0, len(privateIPs))
	for _, ip := range privateIPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		trimmed = append(trimmed, ip)
	}
	if len(trimmed) == 0 {
		if privateIPCount <= 0 {
			return nil, ErrInvalidParameter
		}
		for i := int32(0); i < privateIPCount; i++ {
			trimmed = append(trimmed, s.nextNatGatewayPrivateIPLocked(gateway))
		}
	}

	ifaceID := natGatewayNetworkInterfaceID(gateway.ID)
	for _, ip := range trimmed {
		if idx := natGatewayAddressByPrivateIP(gateway, ip); idx >= 0 {
			continue
		}
		gateway.Addresses = append(gateway.Addresses, NatGatewayAddress{
			NetworkInterfaceID: ifaceID,
			PrivateIP:          ip,
			Status:             "succeeded",
		})
	}
	normalizeNatGatewayPrimaryLocked(gateway)

	out := make([]NatGatewayAddress, 0, len(trimmed))
	for _, ip := range trimmed {
		idx := natGatewayAddressByPrivateIP(gateway, ip)
		if idx < 0 {
			continue
		}
		out = append(out, gateway.Addresses[idx])
	}
	return out, nil
}

func (s *Service) AssociateNatGatewayAddress(gatewayID string, allocationIDs, privateIPs []string) ([]NatGatewayAddress, error) {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" || len(allocationIDs) == 0 {
		return nil, ErrInvalidParameter
	}
	if len(privateIPs) > 0 && len(privateIPs) != len(allocationIDs) {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.natGateways[gatewayID]
	if gateway == nil {
		return nil, ErrNotFound
	}
	if gateway.State == "deleted" {
		return nil, ErrConflict
	}

	ifaceID := natGatewayNetworkInterfaceID(gateway.ID)
	out := make([]NatGatewayAddress, 0, len(allocationIDs))
	for i, allocationID := range allocationIDs {
		allocationID = strings.TrimSpace(allocationID)
		if allocationID == "" {
			return nil, ErrInvalidParameter
		}

		if idx := natGatewayAddressByAllocationID(gateway, allocationID); idx >= 0 {
			out = append(out, gateway.Addresses[idx])
			continue
		}

		address := s.addresses[allocationID]
		if address == nil {
			return nil, ErrNotFound
		}
		if address.AssociationID != "" {
			return nil, ErrConflict
		}

		privateIP := ""
		if len(privateIPs) > 0 {
			privateIP = strings.TrimSpace(privateIPs[i])
			if privateIP == "" {
				return nil, ErrInvalidParameter
			}
		} else {
			privateIP = s.nextNatGatewayPrivateIPLocked(gateway)
		}

		if idx := natGatewayAddressByPrivateIP(gateway, privateIP); idx >= 0 {
			if gateway.Addresses[idx].AllocationID != allocationID {
				return nil, ErrConflict
			}
			out = append(out, gateway.Addresses[idx])
			continue
		}

		associationID := s.nextIDLocked("eipassoc")
		address.AssociationID = associationID
		address.InstanceID = ""
		address.NetworkInterfaceID = ifaceID
		address.PrivateIPAddress = privateIP

		natAddress := NatGatewayAddress{
			AllocationID:       allocationID,
			AssociationID:      associationID,
			NetworkInterfaceID: ifaceID,
			PrivateIP:          privateIP,
			PublicIP:           address.PublicIP,
			Status:             "succeeded",
		}
		gateway.Addresses = append(gateway.Addresses, natAddress)
		out = append(out, natAddress)
	}
	normalizeNatGatewayPrimaryLocked(gateway)

	for i := range out {
		if idx := natGatewayAddressByAssociationID(gateway, out[i].AssociationID); idx >= 0 {
			out[i] = gateway.Addresses[idx]
		}
	}
	return out, nil
}

func (s *Service) DisassociateNatGatewayAddress(gatewayID string, associationIDs []string) ([]NatGatewayAddress, error) {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" || len(associationIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.natGateways[gatewayID]
	if gateway == nil {
		return nil, ErrNotFound
	}

	associationSet := toStringSet(associationIDs)
	out := make([]NatGatewayAddress, 0, len(associationIDs))
	kept := make([]NatGatewayAddress, 0, len(gateway.Addresses))
	for _, addr := range gateway.Addresses {
		if addr.AssociationID == "" {
			kept = append(kept, addr)
			continue
		}
		if _, ok := associationSet[addr.AssociationID]; !ok {
			kept = append(kept, addr)
			continue
		}
		if elastic := s.addresses[addr.AllocationID]; elastic != nil {
			elastic.AssociationID = ""
			elastic.InstanceID = ""
			elastic.NetworkInterfaceID = ""
			elastic.PrivateIPAddress = ""
		}
		out = append(out, addr)
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	gateway.Addresses = kept
	normalizeNatGatewayPrimaryLocked(gateway)
	return out, nil
}

func (s *Service) UnassignPrivateNatGatewayAddress(gatewayID string, privateIPs []string) ([]NatGatewayAddress, error) {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" || len(privateIPs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.natGateways[gatewayID]
	if gateway == nil {
		return nil, ErrNotFound
	}

	ipSet := toStringSet(privateIPs)
	out := make([]NatGatewayAddress, 0, len(privateIPs))
	kept := make([]NatGatewayAddress, 0, len(gateway.Addresses))
	for _, addr := range gateway.Addresses {
		if _, ok := ipSet[addr.PrivateIP]; !ok {
			kept = append(kept, addr)
			continue
		}
		if addr.AllocationID != "" {
			kept = append(kept, addr)
			continue
		}
		out = append(out, addr)
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	gateway.Addresses = kept
	normalizeNatGatewayPrimaryLocked(gateway)
	return out, nil
}

func (s *Service) ModifyVpcPeeringConnectionOptions(connectionID string, accepter, requester *PeeringConnectionOptionsPatch) (PeeringConnectionOptions, PeeringConnectionOptions, error) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return PeeringConnectionOptions{}, PeeringConnectionOptions{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpcPeeringConnections[connectionID]
	if connection == nil {
		return PeeringConnectionOptions{}, PeeringConnectionOptions{}, ErrNotFound
	}
	if connection.StatusCode == "deleted" {
		return PeeringConnectionOptions{}, PeeringConnectionOptions{}, ErrConflict
	}

	if accepter != nil {
		if accepter.AllowDNSResolutionFromRemoteVPC != nil {
			connection.AccepterOptions.AllowDNSResolutionFromRemoteVPC = *accepter.AllowDNSResolutionFromRemoteVPC
		}
		if accepter.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			connection.AccepterOptions.AllowEgressFromLocalClassicLinkToRemoteVPC = *accepter.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if accepter.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			connection.AccepterOptions.AllowEgressFromLocalVPCToRemoteClassicLink = *accepter.AllowEgressFromLocalVPCToRemoteClassicLink
		}
	}
	if requester != nil {
		if requester.AllowDNSResolutionFromRemoteVPC != nil {
			connection.RequesterOptions.AllowDNSResolutionFromRemoteVPC = *requester.AllowDNSResolutionFromRemoteVPC
		}
		if requester.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			connection.RequesterOptions.AllowEgressFromLocalClassicLinkToRemoteVPC = *requester.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if requester.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			connection.RequesterOptions.AllowEgressFromLocalVPCToRemoteClassicLink = *requester.AllowEgressFromLocalVPCToRemoteClassicLink
		}
	}

	return connection.AccepterOptions, connection.RequesterOptions, nil
}

func (s *Service) ReplaceNetworkACLAssociation(associationID, networkACLID string) (NetworkACLAssociation, error) {
	associationID = strings.TrimSpace(associationID)
	networkACLID = strings.TrimSpace(networkACLID)
	if associationID == "" || networkACLID == "" {
		return NetworkACLAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newACL := s.networkAcls[networkACLID]
	if newACL == nil {
		return NetworkACLAssociation{}, ErrNotFound
	}

	var sourceACL *NetworkACL
	var sourceAssoc *NetworkACLAssociation
	for _, acl := range s.networkAcls {
		for i := range acl.Associations {
			if acl.Associations[i].ID != associationID {
				continue
			}
			sourceACL = acl
			sourceAssoc = &acl.Associations[i]
			break
		}
		if sourceAssoc != nil {
			break
		}
	}
	if sourceAssoc == nil || sourceACL == nil {
		return NetworkACLAssociation{}, ErrNotFound
	}
	if sourceACL.VpcID != newACL.VpcID {
		return NetworkACLAssociation{}, ErrInvalidParameter
	}

	subnetID := sourceAssoc.SubnetID
	filtered := make([]NetworkACLAssociation, 0, len(sourceACL.Associations))
	for _, assoc := range sourceACL.Associations {
		if assoc.ID == associationID {
			continue
		}
		filtered = append(filtered, assoc)
	}
	sourceACL.Associations = filtered

	replacement := NetworkACLAssociation{
		ID:       s.nextIDLocked("aclassoc"),
		SubnetID: subnetID,
	}
	newACL.Associations = append(newACL.Associations, replacement)
	return replacement, nil
}

func (s *Service) CreateNetworkInterfacePermission(networkInterfaceID, permission, awsAccountID, awsService string) (NetworkInterfacePermission, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	permission = strings.TrimSpace(strings.ToUpper(permission))
	awsAccountID = strings.TrimSpace(awsAccountID)
	awsService = strings.TrimSpace(awsService)
	if networkInterfaceID == "" || permission == "" {
		return NetworkInterfacePermission{}, ErrInvalidParameter
	}
	if permission != "INSTANCE-ATTACH" && permission != "EIP-ASSOCIATE" {
		return NetworkInterfacePermission{}, ErrInvalidParameter
	}
	if awsAccountID == "" && awsService == "" {
		return NetworkInterfacePermission{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkInterfaces[networkInterfaceID] == nil {
		return NetworkInterfacePermission{}, ErrNotFound
	}

	perm := &NetworkInterfacePermission{
		ID:                 s.nextIDLocked("eni-perm"),
		NetworkInterfaceID: networkInterfaceID,
		Permission:         permission,
		AwsAccountID:       awsAccountID,
		AwsService:         awsService,
		State:              "granted",
		StatusMessage:      "granted",
	}
	s.networkIfacePerms[perm.ID] = perm
	return cloneNetworkInterfacePermission(perm), nil
}

func (s *Service) DeleteNetworkInterfacePermission(permissionID string) error {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkIfacePerms[permissionID] == nil {
		return ErrNotFound
	}
	delete(s.networkIfacePerms, permissionID)
	return nil
}

func (s *Service) DescribeNetworkInterfacePermissions(permissionIDs, networkInterfaceIDs, awsAccountIDs, awsServices, permissions []string) []NetworkInterfacePermission {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(permissionIDs)
	ifaceSet := toStringSet(networkInterfaceIDs)
	accountSet := toStringSet(awsAccountIDs)
	serviceSet := toStringSet(awsServices)
	permissionSet := map[string]struct{}{}
	for _, permission := range permissions {
		permission = strings.TrimSpace(strings.ToUpper(permission))
		if permission == "" {
			continue
		}
		permissionSet[permission] = struct{}{}
	}

	out := make([]NetworkInterfacePermission, 0, len(s.networkIfacePerms))
	for _, permission := range s.networkIfacePerms {
		if len(idSet) > 0 {
			if _, ok := idSet[permission.ID]; !ok {
				continue
			}
		}
		if len(ifaceSet) > 0 {
			if _, ok := ifaceSet[permission.NetworkInterfaceID]; !ok {
				continue
			}
		}
		if len(accountSet) > 0 {
			if _, ok := accountSet[permission.AwsAccountID]; !ok {
				continue
			}
		}
		if len(serviceSet) > 0 {
			if _, ok := serviceSet[permission.AwsService]; !ok {
				continue
			}
		}
		if len(permissionSet) > 0 {
			if _, ok := permissionSet[permission.Permission]; !ok {
				continue
			}
		}
		out = append(out, cloneNetworkInterfacePermission(permission))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DescribeNetworkInterfaceAttribute(networkInterfaceID, attribute string) (NetworkInterfaceAttributeDescription, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	attribute = strings.TrimSpace(attribute)
	if networkInterfaceID == "" {
		return NetworkInterfaceAttributeDescription{}, ErrInvalidParameter
	}
	if attribute != "" {
		switch attribute {
		case "description", "groupSet", "sourceDestCheck", "attachment", "associatePublicIpAddress":
		default:
			return NetworkInterfaceAttributeDescription{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return NetworkInterfaceAttributeDescription{}, ErrNotFound
	}

	associatePublic := false
	if subnet := s.subnets[iface.SubnetID]; subnet != nil {
		associatePublic = subnet.MapPublicIPOnLaunch
	}
	var attachment *NetworkInterfaceAttachment
	if iface.Attachment != nil {
		clone := *iface.Attachment
		attachment = &clone
	}

	return NetworkInterfaceAttributeDescription{
		NetworkInterfaceID:       iface.ID,
		AssociatePublicIPAddress: associatePublic,
		Description:              iface.Description,
		GroupIDs:                 append([]string(nil), iface.GroupIDs...),
		SourceDestCheck:          iface.SourceDestCheck,
		Attachment:               attachment,
	}, nil
}

func (s *Service) ResetNetworkInterfaceAttribute(networkInterfaceID, sourceDestCheck string) error {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	sourceDestCheck = strings.TrimSpace(sourceDestCheck)
	if networkInterfaceID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[networkInterfaceID]
	if iface == nil {
		return ErrNotFound
	}
	if sourceDestCheck != "" && !strings.EqualFold(sourceDestCheck, "sourceDestCheck") {
		return ErrInvalidParameter
	}
	iface.SourceDestCheck = true
	return nil
}

func natGatewayAddressByPrivateIP(gateway *NatGateway, privateIP string) int {
	for i := range gateway.Addresses {
		if gateway.Addresses[i].PrivateIP == privateIP {
			return i
		}
	}
	return -1
}

func natGatewayAddressByAllocationID(gateway *NatGateway, allocationID string) int {
	for i := range gateway.Addresses {
		if gateway.Addresses[i].AllocationID == allocationID {
			return i
		}
	}
	return -1
}

func natGatewayAddressByAssociationID(gateway *NatGateway, associationID string) int {
	for i := range gateway.Addresses {
		if gateway.Addresses[i].AssociationID == associationID {
			return i
		}
	}
	return -1
}

func natGatewayNetworkInterfaceID(gatewayID string) string {
	return fmt.Sprintf("eni-%s", strings.TrimPrefix(gatewayID, "nat-"))
}

func normalizeNatGatewayPrimaryLocked(gateway *NatGateway) {
	for i := range gateway.Addresses {
		gateway.Addresses[i].IsPrimary = false
	}
	if len(gateway.Addresses) > 0 {
		gateway.Addresses[0].IsPrimary = true
	}
}

func (s *Service) nextNatGatewayPrivateIPLocked(gateway *NatGateway) string {
	used := map[string]struct{}{}
	for _, address := range gateway.Addresses {
		if address.PrivateIP == "" {
			continue
		}
		used[address.PrivateIP] = struct{}{}
	}
	for i := 10; i < 250; i++ {
		candidate := fmt.Sprintf("10.0.1.%d", i)
		if _, ok := used[candidate]; ok {
			continue
		}
		return candidate
	}
	return fmt.Sprintf("10.0.1.%d", int(s.seq%250)+1)
}

func cloneNetworkInterfacePermission(in *NetworkInterfacePermission) NetworkInterfacePermission {
	if in == nil {
		return NetworkInterfacePermission{}
	}
	return NetworkInterfacePermission{
		ID:                 in.ID,
		NetworkInterfaceID: in.NetworkInterfaceID,
		Permission:         in.Permission,
		AwsAccountID:       in.AwsAccountID,
		AwsService:         in.AwsService,
		State:              in.State,
		StatusMessage:      in.StatusMessage,
	}
}
