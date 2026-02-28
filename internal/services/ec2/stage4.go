package ec2

import (
	"fmt"
	"sort"
	"strings"
)

type ElasticAddress struct {
	AllocationID       string
	AssociationID      string
	PublicIP           string
	Domain             string
	PtrRecord          string
	InstanceID         string
	NetworkInterfaceID string
	PrivateIPAddress   string
	NetworkBorderGroup string
	Tags               map[string]string
}

func (s *Service) AllocateAddress(address string, tags []Tag) (ElasticAddress, error) {
	address = strings.TrimSpace(address)

	s.mu.Lock()
	defer s.mu.Unlock()

	if address != "" {
		for _, existing := range s.addresses {
			if existing.PublicIP == address {
				return ElasticAddress{}, ErrAlreadyExists
			}
		}
	} else {
		address = s.nextPublicIPLocked()
	}

	addr := &ElasticAddress{
		AllocationID:       s.nextIDLocked("eipalloc"),
		PublicIP:           address,
		Domain:             "vpc",
		NetworkBorderGroup: "us-east-1",
		Tags:               tagsToMap(tags),
	}
	s.addresses[addr.AllocationID] = addr
	return cloneElasticAddress(addr), nil
}

func (s *Service) DescribeAddresses(allocationIDs, publicIPs, associationIDs, instanceIDs []string) []ElasticAddress {
	s.mu.Lock()
	defer s.mu.Unlock()

	allocationSet := toStringSet(allocationIDs)
	publicIPSet := toStringSet(publicIPs)
	associationSet := toStringSet(associationIDs)
	instanceSet := toStringSet(instanceIDs)

	out := make([]ElasticAddress, 0, len(s.addresses))
	for _, address := range s.addresses {
		if len(allocationSet) > 0 {
			if _, ok := allocationSet[address.AllocationID]; !ok {
				continue
			}
		}
		if len(publicIPSet) > 0 {
			if _, ok := publicIPSet[address.PublicIP]; !ok {
				continue
			}
		}
		if len(associationSet) > 0 {
			if _, ok := associationSet[address.AssociationID]; !ok {
				continue
			}
		}
		if len(instanceSet) > 0 {
			if _, ok := instanceSet[address.InstanceID]; !ok {
				continue
			}
		}
		out = append(out, cloneElasticAddress(address))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AllocationID < out[j].AllocationID })
	return out
}

func (s *Service) AssociateAddress(allocationID, publicIP, instanceID, networkInterfaceID, privateIPAddress string, allowReassociation bool) (string, error) {
	allocationID = strings.TrimSpace(allocationID)
	publicIP = strings.TrimSpace(publicIP)
	instanceID = strings.TrimSpace(instanceID)
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	privateIPAddress = strings.TrimSpace(privateIPAddress)
	if allocationID == "" && publicIP == "" {
		return "", ErrInvalidParameter
	}
	if instanceID == "" && networkInterfaceID == "" {
		return "", ErrInvalidParameter
	}
	if instanceID != "" && networkInterfaceID != "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.resolveElasticAddressLocked(allocationID, publicIP)
	if address == nil {
		return "", ErrNotFound
	}

	targetInstanceID := instanceID
	targetPrivateIP := privateIPAddress
	targetNetworkInterfaceID := networkInterfaceID

	if instanceID != "" {
		instance := s.instances[instanceID]
		if instance == nil {
			return "", ErrNotFound
		}
		if instance.StateName == "terminated" {
			return "", ErrConflict
		}
		if targetPrivateIP == "" {
			targetPrivateIP = instance.PrivateIP
		}
	}
	if networkInterfaceID != "" {
		iface := s.networkInterfaces[networkInterfaceID]
		if iface == nil {
			return "", ErrNotFound
		}
		if targetPrivateIP == "" {
			targetPrivateIP = iface.PrivateIP
		}
		if iface.Attachment != nil {
			targetInstanceID = iface.Attachment.InstanceID
		}
	}

	if address.AssociationID != "" {
		if !allowReassociation {
			return "", ErrConflict
		}
		if address.InstanceID == targetInstanceID &&
			address.NetworkInterfaceID == targetNetworkInterfaceID &&
			address.PrivateIPAddress == targetPrivateIP {
			return address.AssociationID, nil
		}
	}

	if address.AssociationID == "" {
		address.AssociationID = s.nextIDLocked("eipassoc")
	}
	address.InstanceID = targetInstanceID
	address.NetworkInterfaceID = targetNetworkInterfaceID
	address.PrivateIPAddress = targetPrivateIP
	return address.AssociationID, nil
}

func (s *Service) DisassociateAddress(associationID, publicIP string) error {
	associationID = strings.TrimSpace(associationID)
	publicIP = strings.TrimSpace(publicIP)
	if associationID == "" && publicIP == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, address := range s.addresses {
		match := false
		if associationID != "" && address.AssociationID == associationID {
			match = true
		}
		if !match && publicIP != "" && address.PublicIP == publicIP {
			match = true
		}
		if !match {
			continue
		}
		if address.AssociationID == "" {
			return ErrNotFound
		}
		address.AssociationID = ""
		address.InstanceID = ""
		address.NetworkInterfaceID = ""
		address.PrivateIPAddress = ""
		return nil
	}
	return ErrNotFound
}

func (s *Service) ReleaseAddress(allocationID, publicIP string) error {
	allocationID = strings.TrimSpace(allocationID)
	publicIP = strings.TrimSpace(publicIP)
	if allocationID == "" && publicIP == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if allocationID != "" {
		address := s.addresses[allocationID]
		if address == nil {
			return ErrNotFound
		}
		if address.AssociationID != "" {
			return ErrConflict
		}
		delete(s.addresses, allocationID)
		delete(s.addressTransfers, allocationID)
		return nil
	}
	for id, address := range s.addresses {
		if address.PublicIP != publicIP {
			continue
		}
		if address.AssociationID != "" {
			return ErrConflict
		}
		delete(s.addresses, id)
		delete(s.addressTransfers, id)
		return nil
	}
	return ErrNotFound
}

func (s *Service) CreateDefaultVpc() (VPC, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing := s.defaultVPCLocked(); existing != nil {
		return cloneVPC(existing), nil
	}

	vpc := &VPC{
		ID:                 s.nextIDLocked("vpc"),
		CidrBlock:          "172.31.0.0/16",
		State:              "available",
		InstanceTenancy:    "default",
		IsDefault:          true,
		DhcpOptionsID:      defaultDHCPOptionsID,
		EnableDnsSupport:   true,
		EnableDnsHostnames: true,
		Tags:               map[string]string{},
	}
	s.vpcs[vpc.ID] = vpc

	defaultSG := &SecurityGroup{
		ID:          s.nextIDLocked("sg"),
		Name:        "default",
		Description: "default VPC security group",
		VpcID:       vpc.ID,
		Ingress:     []IPPermission{},
		Egress:      []IPPermission{{Protocol: "-1", FromPort: -1, ToPort: -1, CidrIP: "0.0.0.0/0"}},
		Tags:        map[string]string{},
	}
	s.securityGroups[defaultSG.ID] = defaultSG
	s.securityGroupNameIndex[securityGroupNameKey(defaultSG.VpcID, defaultSG.Name)] = defaultSG.ID

	table := &RouteTable{
		ID:    s.nextIDLocked("rtb"),
		VpcID: vpc.ID,
		Routes: []Route{
			{DestinationCIDR: vpc.CidrBlock, GatewayID: "local", State: "active", Origin: "CreateRouteTable"},
		},
		Associations: []RouteTableAssociation{
			{ID: s.nextIDLocked("rtbassoc"), Main: true},
		},
		Tags: map[string]string{},
	}
	s.routeTables[table.ID] = table

	acl := &NetworkACL{
		ID:        s.nextIDLocked("acl"),
		VpcID:     vpc.ID,
		IsDefault: true,
		Entries: []NetworkACLEntry{
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: true, CidrBlock: "0.0.0.0/0"},
		},
		Associations: nil,
		Tags:         map[string]string{},
	}
	s.networkAcls[acl.ID] = acl

	return cloneVPC(vpc), nil
}

func (s *Service) CreateDefaultSubnet(availabilityZone, availabilityZoneID string) (Subnet, error) {
	availabilityZone = strings.TrimSpace(availabilityZone)
	availabilityZoneID = strings.TrimSpace(availabilityZoneID)
	if availabilityZone != "" && availabilityZoneID != "" {
		return Subnet{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	defaultVPC := s.defaultVPCLocked()
	if defaultVPC == nil {
		return Subnet{}, ErrNotFound
	}

	zoneName := s.resolveZoneNameLocked(availabilityZone, availabilityZoneID)
	if zoneName == "" {
		return Subnet{}, ErrInvalidParameter
	}

	for _, subnet := range s.subnets {
		if subnet.VpcID == defaultVPC.ID && subnet.AvailabilityZone == zoneName {
			return cloneSubnet(subnet), nil
		}
	}

	subnet := &Subnet{
		ID:                      s.nextIDLocked("subnet"),
		VpcID:                   defaultVPC.ID,
		CidrBlock:               s.nextDefaultSubnetCIDRLocked(defaultVPC.ID),
		AvailabilityZone:        zoneName,
		State:                   "available",
		AvailableIPAddressCount: 4091,
		MapPublicIPOnLaunch:     true,
		Tags:                    map[string]string{},
	}
	s.subnets[subnet.ID] = subnet

	if table := s.mainRouteTableLocked(defaultVPC.ID); table != nil {
		table.Associations = append(table.Associations, RouteTableAssociation{
			ID:       s.nextIDLocked("rtbassoc"),
			SubnetID: subnet.ID,
			Main:     false,
		})
	}
	if acl := s.defaultNetworkACLLocked(defaultVPC.ID); acl != nil {
		acl.Associations = append(acl.Associations, NetworkACLAssociation{
			ID:       s.nextIDLocked("aclassoc"),
			SubnetID: subnet.ID,
		})
	}

	return cloneSubnet(subnet), nil
}

func (s *Service) resolveElasticAddressLocked(allocationID, publicIP string) *ElasticAddress {
	if allocationID != "" {
		return s.addresses[allocationID]
	}
	if publicIP != "" {
		for _, address := range s.addresses {
			if address.PublicIP == publicIP {
				return address
			}
		}
	}
	return nil
}

func (s *Service) nextPublicIPLocked() string {
	for i := 10; i < 250; i++ {
		candidate := fmt.Sprintf("203.0.113.%d", i)
		inUse := false
		for _, address := range s.addresses {
			if address.PublicIP == candidate {
				inUse = true
				break
			}
		}
		if !inUse {
			return candidate
		}
	}
	return fmt.Sprintf("203.0.113.%d", len(s.addresses)+10)
}

func (s *Service) defaultVPCLocked() *VPC {
	for _, vpc := range s.vpcs {
		if vpc.IsDefault {
			return vpc
		}
	}
	return nil
}

func (s *Service) resolveZoneNameLocked(zoneName, zoneID string) string {
	if zoneName != "" {
		return zoneName
	}
	for _, zone := range s.availabilityZones {
		if zone.ZoneID == zoneID {
			return zone.Name
		}
	}
	return ""
}

func (s *Service) nextDefaultSubnetCIDRLocked(vpcID string) string {
	used := map[string]struct{}{}
	for _, subnet := range s.subnets {
		if subnet.VpcID != vpcID {
			continue
		}
		used[subnet.CidrBlock] = struct{}{}
	}
	for i := 0; i < 16; i++ {
		candidate := fmt.Sprintf("172.31.%d.0/20", i*16)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
	return "172.31.240.0/20"
}

func cloneElasticAddress(in *ElasticAddress) ElasticAddress {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
