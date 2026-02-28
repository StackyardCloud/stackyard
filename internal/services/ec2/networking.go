package ec2

import (
	"sort"
	"strings"
	"time"
)

type VPC struct {
	ID                 string
	CidrBlock          string
	State              string
	InstanceTenancy    string
	IsDefault          bool
	DhcpOptionsID      string
	EnableDnsSupport   bool
	EnableDnsHostnames bool
	Tags               map[string]string
}

type Subnet struct {
	ID                      string
	VpcID                   string
	CidrBlock               string
	AvailabilityZone        string
	State                   string
	AvailableIPAddressCount int32
	MapPublicIPOnLaunch     bool
	Tags                    map[string]string
}

type InternetGatewayAttachment struct {
	VpcID string
	State string
}

type InternetGateway struct {
	ID          string
	Attachments []InternetGatewayAttachment
	Tags        map[string]string
}

type Route struct {
	DestinationCIDR string
	GatewayID       string
	State           string
	Origin          string
}

type RouteTableAssociation struct {
	ID       string
	SubnetID string
	Main     bool
}

type RouteTable struct {
	ID           string
	VpcID        string
	Routes       []Route
	Associations []RouteTableAssociation
	Tags         map[string]string
}

type NetworkACLEntry struct {
	RuleNumber int32
	Protocol   string
	RuleAction string
	Egress     bool
	CidrBlock  string
}

type NetworkACLAssociation struct {
	ID       string
	SubnetID string
}

type NetworkACL struct {
	ID           string
	VpcID        string
	IsDefault    bool
	Entries      []NetworkACLEntry
	Associations []NetworkACLAssociation
	Tags         map[string]string
}

type NetworkInterfaceAttachment struct {
	ID          string
	InstanceID  string
	DeviceIndex int32
	Status      string
	AttachTime  time.Time
}

type NetworkInterface struct {
	ID                  string
	SubnetID            string
	VpcID               string
	Description         string
	PrivateIP           string
	SecondaryPrivateIPs []string
	IPv4Prefixes        []string
	IPv6Addresses       []string
	IPv6Prefixes        []string
	Status              string
	SourceDestCheck     bool
	GroupIDs            []string
	Attachment          *NetworkInterfaceAttachment
	Tags                map[string]string
}

func (s *Service) CreateVpc(cidrBlock string, tags []Tag) (VPC, error) {
	cidrBlock = strings.TrimSpace(cidrBlock)
	if cidrBlock == "" {
		return VPC{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := &VPC{
		ID:                 s.nextIDLocked("vpc"),
		CidrBlock:          cidrBlock,
		State:              "available",
		InstanceTenancy:    "default",
		IsDefault:          false,
		DhcpOptionsID:      defaultDHCPOptionsID,
		EnableDnsSupport:   true,
		EnableDnsHostnames: false,
		Tags:               tagsToMap(tags),
	}
	s.vpcs[vpc.ID] = vpc

	table := &RouteTable{
		ID:    s.nextIDLocked("rtb"),
		VpcID: vpc.ID,
		Routes: []Route{
			{DestinationCIDR: vpc.CidrBlock, GatewayID: "local", State: "active", Origin: "CreateRouteTable"},
		},
		Associations: nil,
		Tags:         map[string]string{},
	}
	s.routeTables[table.ID] = table

	acl := &NetworkACL{
		ID:           s.nextIDLocked("acl"),
		VpcID:        vpc.ID,
		IsDefault:    true,
		Entries:      nil,
		Associations: nil,
		Tags:         map[string]string{},
	}
	s.networkAcls[acl.ID] = acl

	return cloneVPC(vpc), nil
}

func (s *Service) DescribeVpcs(vpcIDs []string) []VPC {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(vpcIDs)
	out := make([]VPC, 0, len(s.vpcs))
	for _, vpc := range s.vpcs {
		if len(idSet) > 0 {
			if _, ok := idSet[vpc.ID]; !ok {
				continue
			}
		}
		out = append(out, cloneVPC(vpc))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteVpc(vpcID string) error {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return ErrNotFound
	}
	if vpc.IsDefault {
		return ErrConflict
	}
	for _, subnet := range s.subnets {
		if subnet.VpcID == vpcID {
			return ErrConflict
		}
	}
	for _, gateway := range s.internetGateways {
		for _, attachment := range gateway.Attachments {
			if attachment.VpcID == vpcID {
				return ErrConflict
			}
		}
	}
	for _, gateway := range s.egressOnlyGateways {
		for _, attachment := range gateway.Attachments {
			if attachment.VpcID == vpcID {
				return ErrConflict
			}
		}
	}
	for _, gateway := range s.natGateways {
		if gateway.State == "deleted" {
			continue
		}
		if gateway.VpcID == vpcID {
			return ErrConflict
		}
	}
	for _, peering := range s.vpcPeeringConnections {
		if peering.StatusCode == "deleted" {
			continue
		}
		if peering.RequesterVpcID == vpcID || peering.AccepterVpcID == vpcID {
			return ErrConflict
		}
	}
	for _, gateway := range s.vpnGateways {
		for _, attachment := range gateway.Attachments {
			if attachment.State != "attached" {
				continue
			}
			if attachment.VpcID == vpcID {
				return ErrConflict
			}
		}
	}
	for _, instance := range s.instances {
		if instance.VpcID == vpcID && instance.StateName != "terminated" {
			return ErrConflict
		}
	}
	for _, attachment := range s.classicLinkAttachments {
		if attachment.VpcID == vpcID {
			return ErrConflict
		}
	}
	for id, group := range s.securityGroups {
		if group.VpcID != vpcID {
			continue
		}
		delete(s.securityGroupNameIndex, securityGroupNameKey(group.VpcID, group.Name))
		delete(s.securityGroups, id)
	}
	for id, table := range s.routeTables {
		if table.VpcID == vpcID {
			delete(s.routeTables, id)
		}
	}
	for id, acl := range s.networkAcls {
		if acl.VpcID == vpcID {
			delete(s.networkAcls, id)
		}
	}
	delete(s.vpcClassicLinkEnabled, vpcID)
	delete(s.vpcClassicLinkDnsSupported, vpcID)
	delete(s.vpcs, vpcID)
	return nil
}

func (s *Service) CreateSubnet(vpcID, cidrBlock, availabilityZone string, tags []Tag) (Subnet, error) {
	vpcID = strings.TrimSpace(vpcID)
	cidrBlock = strings.TrimSpace(cidrBlock)
	availabilityZone = strings.TrimSpace(availabilityZone)
	if vpcID == "" || cidrBlock == "" {
		return Subnet{}, ErrInvalidParameter
	}
	if availabilityZone == "" {
		availabilityZone = "us-east-1a"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.vpcs[vpcID] == nil {
		return Subnet{}, ErrNotFound
	}

	subnet := &Subnet{
		ID:                      s.nextIDLocked("subnet"),
		VpcID:                   vpcID,
		CidrBlock:               cidrBlock,
		AvailabilityZone:        availabilityZone,
		State:                   "available",
		AvailableIPAddressCount: 251,
		MapPublicIPOnLaunch:     true,
		Tags:                    tagsToMap(tags),
	}
	s.subnets[subnet.ID] = subnet

	if table := s.mainRouteTableLocked(vpcID); table != nil {
		table.Associations = append(table.Associations, RouteTableAssociation{
			ID:       s.nextIDLocked("rtbassoc"),
			SubnetID: subnet.ID,
			Main:     false,
		})
	}
	if acl := s.defaultNetworkACLLocked(vpcID); acl != nil {
		acl.Associations = append(acl.Associations, NetworkACLAssociation{
			ID:       s.nextIDLocked("aclassoc"),
			SubnetID: subnet.ID,
		})
	}

	return cloneSubnet(subnet), nil
}

func (s *Service) DescribeSubnets(subnetIDs, vpcIDs []string) []Subnet {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(subnetIDs)
	vpcSet := toStringSet(vpcIDs)
	out := make([]Subnet, 0, len(s.subnets))
	for _, subnet := range s.subnets {
		if len(idSet) > 0 {
			if _, ok := idSet[subnet.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			if _, ok := vpcSet[subnet.VpcID]; !ok {
				continue
			}
		}
		out = append(out, cloneSubnet(subnet))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteSubnet(subnetID string) error {
	subnetID = strings.TrimSpace(subnetID)
	if subnetID == "" {
		return ErrInvalidParameter
	}
	if subnetID == defaultSubnetID {
		return ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	subnet := s.subnets[subnetID]
	if subnet == nil {
		return ErrNotFound
	}
	for _, instance := range s.instances {
		if instance.SubnetID == subnetID && instance.StateName != "terminated" {
			return ErrConflict
		}
	}
	for _, iface := range s.networkInterfaces {
		if iface.SubnetID == subnetID {
			return ErrConflict
		}
	}
	for _, table := range s.routeTables {
		filtered := make([]RouteTableAssociation, 0, len(table.Associations))
		for _, assoc := range table.Associations {
			if assoc.SubnetID == subnetID {
				continue
			}
			filtered = append(filtered, assoc)
		}
		table.Associations = filtered
	}
	for _, acl := range s.networkAcls {
		filtered := make([]NetworkACLAssociation, 0, len(acl.Associations))
		for _, assoc := range acl.Associations {
			if assoc.SubnetID == subnetID {
				continue
			}
			filtered = append(filtered, assoc)
		}
		acl.Associations = filtered
	}
	delete(s.subnets, subnetID)
	_ = subnet
	return nil
}

func (s *Service) CreateInternetGateway(tags []Tag) (InternetGateway, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gateway := &InternetGateway{
		ID:          s.nextIDLocked("igw"),
		Attachments: nil,
		Tags:        tagsToMap(tags),
	}
	s.internetGateways[gateway.ID] = gateway
	return cloneInternetGateway(gateway), nil
}

func (s *Service) DescribeInternetGateways(gatewayIDs, vpcIDs []string) []InternetGateway {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(gatewayIDs)
	vpcSet := toStringSet(vpcIDs)
	out := make([]InternetGateway, 0, len(s.internetGateways))
	for _, gateway := range s.internetGateways {
		if len(idSet) > 0 {
			if _, ok := idSet[gateway.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			matched := false
			for _, attachment := range gateway.Attachments {
				if _, ok := vpcSet[attachment.VpcID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cloneInternetGateway(gateway))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AttachInternetGateway(gatewayID, vpcID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	vpcID = strings.TrimSpace(vpcID)
	if gatewayID == "" || vpcID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	gateway := s.internetGateways[gatewayID]
	if gateway == nil || s.vpcs[vpcID] == nil {
		return ErrNotFound
	}
	for _, attachment := range gateway.Attachments {
		if attachment.VpcID == vpcID {
			return nil
		}
	}
	gateway.Attachments = append(gateway.Attachments, InternetGatewayAttachment{VpcID: vpcID, State: "available"})
	return nil
}

func (s *Service) DetachInternetGateway(gatewayID, vpcID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	vpcID = strings.TrimSpace(vpcID)
	if gatewayID == "" || vpcID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	gateway := s.internetGateways[gatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	filtered := make([]InternetGatewayAttachment, 0, len(gateway.Attachments))
	removed := false
	for _, attachment := range gateway.Attachments {
		if attachment.VpcID == vpcID {
			removed = true
			continue
		}
		filtered = append(filtered, attachment)
	}
	if !removed {
		return ErrNotFound
	}
	gateway.Attachments = filtered
	return nil
}

func (s *Service) DeleteInternetGateway(gatewayID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	gateway := s.internetGateways[gatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	if len(gateway.Attachments) > 0 {
		return ErrConflict
	}
	delete(s.internetGateways, gatewayID)
	return nil
}

func (s *Service) CreateRouteTable(vpcID string, tags []Tag) (RouteTable, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return RouteTable{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return RouteTable{}, ErrNotFound
	}
	table := &RouteTable{
		ID:    s.nextIDLocked("rtb"),
		VpcID: vpcID,
		Routes: []Route{
			{DestinationCIDR: vpc.CidrBlock, GatewayID: "local", State: "active", Origin: "CreateRouteTable"},
		},
		Associations: nil,
		Tags:         tagsToMap(tags),
	}
	s.routeTables[table.ID] = table
	return cloneRouteTable(table), nil
}

func (s *Service) DescribeRouteTables(tableIDs, vpcIDs, subnetIDs []string) []RouteTable {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(tableIDs)
	vpcSet := toStringSet(vpcIDs)
	subnetSet := toStringSet(subnetIDs)
	out := make([]RouteTable, 0, len(s.routeTables))
	for _, table := range s.routeTables {
		if len(idSet) > 0 {
			if _, ok := idSet[table.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			if _, ok := vpcSet[table.VpcID]; !ok {
				continue
			}
		}
		if len(subnetSet) > 0 {
			matched := false
			for _, assoc := range table.Associations {
				if _, ok := subnetSet[assoc.SubnetID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cloneRouteTable(table))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AssociateRouteTable(routeTableID, subnetID string) (RouteTableAssociation, error) {
	routeTableID = strings.TrimSpace(routeTableID)
	subnetID = strings.TrimSpace(subnetID)
	if routeTableID == "" || subnetID == "" {
		return RouteTableAssociation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	table := s.routeTables[routeTableID]
	subnet := s.subnets[subnetID]
	if table == nil || subnet == nil {
		return RouteTableAssociation{}, ErrNotFound
	}
	if subnet.VpcID != table.VpcID {
		return RouteTableAssociation{}, ErrConflict
	}
	for _, existing := range table.Associations {
		if existing.SubnetID == subnetID {
			return existing, nil
		}
	}
	for _, other := range s.routeTables {
		filtered := make([]RouteTableAssociation, 0, len(other.Associations))
		for _, assoc := range other.Associations {
			if assoc.SubnetID == subnetID && !assoc.Main {
				continue
			}
			filtered = append(filtered, assoc)
		}
		other.Associations = filtered
	}
	assoc := RouteTableAssociation{
		ID:       s.nextIDLocked("rtbassoc"),
		SubnetID: subnetID,
		Main:     false,
	}
	table.Associations = append(table.Associations, assoc)
	return assoc, nil
}

func (s *Service) DisassociateRouteTable(associationID string) error {
	associationID = strings.TrimSpace(associationID)
	if associationID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, table := range s.routeTables {
		filtered := make([]RouteTableAssociation, 0, len(table.Associations))
		removed := false
		for _, assoc := range table.Associations {
			if assoc.ID == associationID {
				if assoc.Main {
					return ErrConflict
				}
				removed = true
				continue
			}
			filtered = append(filtered, assoc)
		}
		if removed {
			table.Associations = filtered
			return nil
		}
	}
	return ErrNotFound
}

func (s *Service) CreateRoute(routeTableID, destinationCIDR, gatewayID string) error {
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
	route := Route{
		DestinationCIDR: destinationCIDR,
		GatewayID:       gatewayID,
		State:           "active",
		Origin:          "CreateRoute",
	}
	for i, existing := range table.Routes {
		if existing.DestinationCIDR == destinationCIDR {
			table.Routes[i] = route
			return nil
		}
	}
	table.Routes = append(table.Routes, route)
	return nil
}

func (s *Service) DeleteRoute(routeTableID, destinationCIDR string) error {
	routeTableID = strings.TrimSpace(routeTableID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	if routeTableID == "" || destinationCIDR == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	table := s.routeTables[routeTableID]
	if table == nil {
		return ErrNotFound
	}
	filtered := make([]Route, 0, len(table.Routes))
	removed := false
	for _, route := range table.Routes {
		if route.DestinationCIDR == destinationCIDR && route.GatewayID != "local" {
			removed = true
			continue
		}
		filtered = append(filtered, route)
	}
	if !removed {
		return ErrNotFound
	}
	table.Routes = filtered
	return nil
}

func (s *Service) DeleteRouteTable(routeTableID string) error {
	routeTableID = strings.TrimSpace(routeTableID)
	if routeTableID == "" {
		return ErrInvalidParameter
	}
	if routeTableID == defaultRouteTableID {
		return ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	table := s.routeTables[routeTableID]
	if table == nil {
		return ErrNotFound
	}
	if len(table.Associations) > 0 {
		return ErrConflict
	}
	delete(s.routeTables, routeTableID)
	return nil
}

func (s *Service) CreateNetworkACL(vpcID string, tags []Tag) (NetworkACL, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return NetworkACL{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.vpcs[vpcID] == nil {
		return NetworkACL{}, ErrNotFound
	}
	acl := &NetworkACL{
		ID:           s.nextIDLocked("acl"),
		VpcID:        vpcID,
		IsDefault:    false,
		Entries:      nil,
		Associations: nil,
		Tags:         tagsToMap(tags),
	}
	s.networkAcls[acl.ID] = acl
	return cloneNetworkACL(acl), nil
}

func (s *Service) DescribeNetworkACLs(aclIDs, vpcIDs []string) []NetworkACL {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(aclIDs)
	vpcSet := toStringSet(vpcIDs)
	out := make([]NetworkACL, 0, len(s.networkAcls))
	for _, acl := range s.networkAcls {
		if len(idSet) > 0 {
			if _, ok := idSet[acl.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			if _, ok := vpcSet[acl.VpcID]; !ok {
				continue
			}
		}
		out = append(out, cloneNetworkACL(acl))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteNetworkACL(aclID string) error {
	aclID = strings.TrimSpace(aclID)
	if aclID == "" {
		return ErrInvalidParameter
	}
	if aclID == defaultNetworkACLID {
		return ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acl := s.networkAcls[aclID]
	if acl == nil {
		return ErrNotFound
	}
	if acl.IsDefault || len(acl.Associations) > 0 {
		return ErrConflict
	}
	delete(s.networkAcls, aclID)
	return nil
}

func (s *Service) CreateNetworkInterface(subnetID, description, privateIP string, groupIDs []string, tags []Tag) (NetworkInterface, error) {
	subnetID = strings.TrimSpace(subnetID)
	description = strings.TrimSpace(description)
	privateIP = strings.TrimSpace(privateIP)
	if subnetID == "" {
		return NetworkInterface{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	subnet := s.subnets[subnetID]
	if subnet == nil {
		return NetworkInterface{}, ErrNotFound
	}
	if privateIP == "" {
		privateIP = "10.0.0.10"
	}
	if len(groupIDs) == 0 {
		groupIDs = []string{"sg-00000000"}
	}
	resolved := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			continue
		}
		group := s.securityGroups[groupID]
		if group == nil {
			return NetworkInterface{}, ErrNotFound
		}
		if group.VpcID != subnet.VpcID {
			return NetworkInterface{}, ErrConflict
		}
		resolved = append(resolved, groupID)
	}
	if len(resolved) == 0 {
		resolved = []string{"sg-00000000"}
	}
	iface := &NetworkInterface{
		ID:                  s.nextIDLocked("eni"),
		SubnetID:            subnet.ID,
		VpcID:               subnet.VpcID,
		Description:         description,
		PrivateIP:           privateIP,
		SecondaryPrivateIPs: []string{},
		IPv4Prefixes:        []string{},
		IPv6Addresses:       []string{},
		IPv6Prefixes:        []string{},
		Status:              "available",
		SourceDestCheck:     true,
		GroupIDs:            append([]string(nil), resolved...),
		Attachment:          nil,
		Tags:                tagsToMap(tags),
	}
	s.networkInterfaces[iface.ID] = iface
	return cloneNetworkInterface(iface), nil
}

func (s *Service) DescribeNetworkInterfaces(interfaceIDs []string, subnetID, vpcID string) []NetworkInterface {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(interfaceIDs)
	subnetID = strings.TrimSpace(subnetID)
	vpcID = strings.TrimSpace(vpcID)
	out := make([]NetworkInterface, 0, len(s.networkInterfaces))
	for _, iface := range s.networkInterfaces {
		if len(idSet) > 0 {
			if _, ok := idSet[iface.ID]; !ok {
				continue
			}
		}
		if subnetID != "" && iface.SubnetID != subnetID {
			continue
		}
		if vpcID != "" && iface.VpcID != vpcID {
			continue
		}
		out = append(out, cloneNetworkInterface(iface))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AttachNetworkInterface(interfaceID, instanceID string, deviceIndex int32) (NetworkInterfaceAttachment, error) {
	interfaceID = strings.TrimSpace(interfaceID)
	instanceID = strings.TrimSpace(instanceID)
	if interfaceID == "" || instanceID == "" {
		return NetworkInterfaceAttachment{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	iface := s.networkInterfaces[interfaceID]
	instance := s.instances[instanceID]
	if iface == nil || instance == nil {
		return NetworkInterfaceAttachment{}, ErrNotFound
	}
	if iface.Attachment != nil {
		return NetworkInterfaceAttachment{}, ErrConflict
	}
	if instance.StateName == "terminated" {
		return NetworkInterfaceAttachment{}, ErrConflict
	}
	attachment := &NetworkInterfaceAttachment{
		ID:          s.nextIDLocked("eniattach"),
		InstanceID:  instanceID,
		DeviceIndex: deviceIndex,
		Status:      "attached",
		AttachTime:  time.Now().UTC(),
	}
	iface.Attachment = attachment
	iface.Status = "in-use"
	return *attachment, nil
}

func (s *Service) DetachNetworkInterface(attachmentID string, force bool) error {
	_ = force
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, iface := range s.networkInterfaces {
		if iface.Attachment == nil {
			continue
		}
		if iface.Attachment.ID != attachmentID {
			continue
		}
		iface.Attachment = nil
		iface.Status = "available"
		return nil
	}
	return ErrNotFound
}

func (s *Service) DeleteNetworkInterface(interfaceID string) error {
	interfaceID = strings.TrimSpace(interfaceID)
	if interfaceID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	iface := s.networkInterfaces[interfaceID]
	if iface == nil {
		return ErrNotFound
	}
	if iface.Attachment != nil {
		return ErrConflict
	}
	for id, perm := range s.networkIfacePerms {
		if perm.NetworkInterfaceID == interfaceID {
			delete(s.networkIfacePerms, id)
		}
	}
	s.deleteTransitGatewayMulticastGroupsForNetworkInterfaceLocked(interfaceID)
	delete(s.networkInterfaces, interfaceID)
	return nil
}

func (s *Service) mainRouteTableLocked(vpcID string) *RouteTable {
	for _, table := range s.routeTables {
		if table.VpcID != vpcID {
			continue
		}
		for _, assoc := range table.Associations {
			if assoc.Main {
				return table
			}
		}
	}
	return nil
}

func (s *Service) defaultNetworkACLLocked(vpcID string) *NetworkACL {
	for _, acl := range s.networkAcls {
		if acl.VpcID == vpcID && acl.IsDefault {
			return acl
		}
	}
	return nil
}

func cloneVPC(in *VPC) VPC {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneSubnet(in *Subnet) Subnet {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneInternetGateway(in *InternetGateway) InternetGateway {
	out := *in
	out.Attachments = append([]InternetGatewayAttachment(nil), in.Attachments...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneRouteTable(in *RouteTable) RouteTable {
	out := *in
	out.Routes = append([]Route(nil), in.Routes...)
	out.Associations = append([]RouteTableAssociation(nil), in.Associations...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneNetworkACL(in *NetworkACL) NetworkACL {
	out := *in
	out.Entries = append([]NetworkACLEntry(nil), in.Entries...)
	out.Associations = append([]NetworkACLAssociation(nil), in.Associations...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneNetworkInterface(in *NetworkInterface) NetworkInterface {
	out := *in
	out.GroupIDs = append([]string(nil), in.GroupIDs...)
	out.SecondaryPrivateIPs = append([]string(nil), in.SecondaryPrivateIPs...)
	out.IPv4Prefixes = append([]string(nil), in.IPv4Prefixes...)
	out.IPv6Addresses = append([]string(nil), in.IPv6Addresses...)
	out.IPv6Prefixes = append([]string(nil), in.IPv6Prefixes...)
	if in.Attachment != nil {
		attachment := *in.Attachment
		out.Attachment = &attachment
	}
	out.Tags = cloneStringMap(in.Tags)
	return out
}
