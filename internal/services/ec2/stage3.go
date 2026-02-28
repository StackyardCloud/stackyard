package ec2

import (
	"sort"
	"strings"
)

type AccountAttribute struct {
	Name   string
	Values []string
}

type VPCAttribute struct {
	VpcID     string
	Attribute string
	Value     bool
}

func (s *Service) CreateNetworkACLEntry(networkACLID string, ruleNumber int32, protocol, ruleAction string, egress bool, cidrBlock string) error {
	networkACLID = strings.TrimSpace(networkACLID)
	protocol = strings.TrimSpace(protocol)
	ruleAction = strings.ToLower(strings.TrimSpace(ruleAction))
	cidrBlock = strings.TrimSpace(cidrBlock)
	if networkACLID == "" || ruleNumber <= 0 || protocol == "" || (ruleAction != "allow" && ruleAction != "deny") {
		return ErrInvalidParameter
	}
	if cidrBlock == "" {
		cidrBlock = "0.0.0.0/0"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acl := s.networkAcls[networkACLID]
	if acl == nil {
		return ErrNotFound
	}
	for _, entry := range acl.Entries {
		if entry.RuleNumber == ruleNumber && entry.Egress == egress {
			return ErrAlreadyExists
		}
	}
	acl.Entries = append(acl.Entries, NetworkACLEntry{
		RuleNumber: ruleNumber,
		Protocol:   protocol,
		RuleAction: ruleAction,
		Egress:     egress,
		CidrBlock:  cidrBlock,
	})
	sortNetworkACLEntries(acl.Entries)
	return nil
}

func (s *Service) ReplaceNetworkACLEntry(networkACLID string, ruleNumber int32, protocol, ruleAction string, egress bool, cidrBlock string) error {
	networkACLID = strings.TrimSpace(networkACLID)
	protocol = strings.TrimSpace(protocol)
	ruleAction = strings.ToLower(strings.TrimSpace(ruleAction))
	cidrBlock = strings.TrimSpace(cidrBlock)
	if networkACLID == "" || ruleNumber <= 0 || protocol == "" || (ruleAction != "allow" && ruleAction != "deny") {
		return ErrInvalidParameter
	}
	if cidrBlock == "" {
		cidrBlock = "0.0.0.0/0"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acl := s.networkAcls[networkACLID]
	if acl == nil {
		return ErrNotFound
	}
	for i, entry := range acl.Entries {
		if entry.RuleNumber == ruleNumber && entry.Egress == egress {
			acl.Entries[i] = NetworkACLEntry{
				RuleNumber: ruleNumber,
				Protocol:   protocol,
				RuleAction: ruleAction,
				Egress:     egress,
				CidrBlock:  cidrBlock,
			}
			sortNetworkACLEntries(acl.Entries)
			return nil
		}
	}
	return ErrNotFound
}

func (s *Service) DeleteNetworkACLEntry(networkACLID string, ruleNumber int32, egress bool) error {
	networkACLID = strings.TrimSpace(networkACLID)
	if networkACLID == "" || ruleNumber <= 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acl := s.networkAcls[networkACLID]
	if acl == nil {
		return ErrNotFound
	}
	next := make([]NetworkACLEntry, 0, len(acl.Entries))
	removed := false
	for _, entry := range acl.Entries {
		if entry.RuleNumber == ruleNumber && entry.Egress == egress {
			removed = true
			continue
		}
		next = append(next, entry)
	}
	if !removed {
		return ErrNotFound
	}
	acl.Entries = next
	return nil
}

func (s *Service) ReplaceRoute(routeTableID, destinationCIDR, gatewayID string) error {
	routeTableID = strings.TrimSpace(routeTableID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	gatewayID = strings.TrimSpace(gatewayID)
	if routeTableID == "" || destinationCIDR == "" || gatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.routeTables[routeTableID]
	if table == nil {
		return ErrNotFound
	}
	if gatewayID != "local" && s.internetGateways[gatewayID] == nil {
		return ErrNotFound
	}
	for i, route := range table.Routes {
		if route.DestinationCIDR != destinationCIDR {
			continue
		}
		if route.GatewayID == "local" {
			return ErrConflict
		}
		table.Routes[i].GatewayID = gatewayID
		table.Routes[i].State = "active"
		table.Routes[i].Origin = "CreateRoute"
		return nil
	}
	return ErrNotFound
}

func (s *Service) ReplaceRouteTableAssociation(associationID, routeTableID string) (RouteTableAssociation, error) {
	associationID = strings.TrimSpace(associationID)
	routeTableID = strings.TrimSpace(routeTableID)
	if associationID == "" || routeTableID == "" {
		return RouteTableAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.routeTables[routeTableID]
	if target == nil {
		return RouteTableAssociation{}, ErrNotFound
	}
	var source *RouteTable
	var current RouteTableAssociation
	found := false
	for _, table := range s.routeTables {
		for _, assoc := range table.Associations {
			if assoc.ID == associationID {
				source = table
				current = assoc
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return RouteTableAssociation{}, ErrNotFound
	}
	if current.Main || current.SubnetID == "" {
		return RouteTableAssociation{}, ErrConflict
	}
	if source.VpcID != target.VpcID {
		return RouteTableAssociation{}, ErrConflict
	}

	filtered := make([]RouteTableAssociation, 0, len(source.Associations))
	for _, assoc := range source.Associations {
		if assoc.ID == associationID {
			continue
		}
		filtered = append(filtered, assoc)
	}
	source.Associations = filtered

	for _, assoc := range target.Associations {
		if assoc.SubnetID == current.SubnetID && !assoc.Main {
			return assoc, nil
		}
	}
	newAssoc := RouteTableAssociation{
		ID:       s.nextIDLocked("rtbassoc"),
		SubnetID: current.SubnetID,
		Main:     false,
	}
	target.Associations = append(target.Associations, newAssoc)
	return newAssoc, nil
}

func (s *Service) ModifyNetworkInterfaceAttribute(interfaceID string, description *string, sourceDestCheck *bool, groupIDs []string) error {
	interfaceID = strings.TrimSpace(interfaceID)
	if interfaceID == "" {
		return ErrInvalidParameter
	}
	if description == nil && sourceDestCheck == nil && len(groupIDs) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	iface := s.networkInterfaces[interfaceID]
	if iface == nil {
		return ErrNotFound
	}
	if description != nil {
		iface.Description = strings.TrimSpace(*description)
	}
	if sourceDestCheck != nil {
		iface.SourceDestCheck = *sourceDestCheck
	}
	if len(groupIDs) > 0 {
		next := make([]string, 0, len(groupIDs))
		for _, id := range groupIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			group := s.securityGroups[id]
			if group == nil {
				return ErrNotFound
			}
			if group.VpcID != iface.VpcID {
				return ErrConflict
			}
			next = append(next, id)
		}
		if len(next) == 0 {
			return ErrInvalidParameter
		}
		iface.GroupIDs = next
	}
	return nil
}

func (s *Service) ModifySubnetAttribute(subnetID string, mapPublicIPOnLaunch *bool) error {
	subnetID = strings.TrimSpace(subnetID)
	if subnetID == "" || mapPublicIPOnLaunch == nil {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subnet := s.subnets[subnetID]
	if subnet == nil {
		return ErrNotFound
	}
	subnet.MapPublicIPOnLaunch = *mapPublicIPOnLaunch
	return nil
}

func (s *Service) ModifyVpcAttribute(vpcID string, enableDnsSupport, enableDnsHostnames *bool) error {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return ErrInvalidParameter
	}
	if enableDnsSupport == nil && enableDnsHostnames == nil {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return ErrNotFound
	}
	if enableDnsSupport != nil {
		vpc.EnableDnsSupport = *enableDnsSupport
	}
	if enableDnsHostnames != nil {
		vpc.EnableDnsHostnames = *enableDnsHostnames
	}
	return nil
}

func (s *Service) DescribeVpcAttribute(vpcID, attribute string) (VPCAttribute, error) {
	vpcID = strings.TrimSpace(vpcID)
	attribute = strings.TrimSpace(attribute)
	if vpcID == "" || attribute == "" {
		return VPCAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return VPCAttribute{}, ErrNotFound
	}
	switch strings.ToLower(attribute) {
	case "enablednssupport":
		return VPCAttribute{VpcID: vpcID, Attribute: "enableDnsSupport", Value: vpc.EnableDnsSupport}, nil
	case "enablednshostnames":
		return VPCAttribute{VpcID: vpcID, Attribute: "enableDnsHostnames", Value: vpc.EnableDnsHostnames}, nil
	default:
		return VPCAttribute{}, ErrInvalidParameter
	}
}

func (s *Service) DescribeAccountAttributes(attributeNames []string) []AccountAttribute {
	requested := toStringSet(attributeNames)
	all := []AccountAttribute{
		{Name: "default-vpc", Values: []string{defaultVPCID}},
		{Name: "supported-platforms", Values: []string{"VPC"}},
	}
	if len(requested) == 0 {
		return all
	}
	out := make([]AccountAttribute, 0, len(all))
	for _, attr := range all {
		if _, ok := requested[attr.Name]; ok {
			out = append(out, attr)
		}
	}
	return out
}

func sortNetworkACLEntries(entries []NetworkACLEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Egress == entries[j].Egress {
			return entries[i].RuleNumber < entries[j].RuleNumber
		}
		if entries[i].Egress {
			return false
		}
		return true
	})
}
