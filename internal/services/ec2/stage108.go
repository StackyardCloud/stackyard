package ec2

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type IpamPool struct {
	AddressFamily                  string
	AllocationDefaultNetmaskLength *int32
	AllocationMaxNetmaskLength     *int32
	AllocationMinNetmaskLength     *int32
	AutoImport                     bool
	AwsService                     string
	Description                    string
	IpamARN                        string
	IpamPoolARN                    string
	IpamPoolID                     string
	IpamRegion                     string
	IpamScopeARN                   string
	IpamScopeID                    string
	IpamScopeType                  string
	Locale                         string
	OwnerID                        string
	PublicIpSource                 string
	PubliclyAdvertisable           bool
	SourceIpamPoolID               string
	State                          string
	Tags                           map[string]string
}

type IpamResourceDiscovery struct {
	Description              string
	IpamResourceDiscoveryARN string
	IpamResourceDiscoveryID  string
	IpamResourceRegion       string
	IsDefault                bool
	OperatingRegions         []string
	OwnerID                  string
	State                    string
	Tags                     map[string]string
}

type IpamScope struct {
	Description   string
	IpamARN       string
	IpamID        string
	IpamRegion    string
	IpamScopeARN  string
	IpamScopeID   string
	IpamScopeType string
	IsDefault     bool
	OwnerID       string
	PoolCount     int32
	State         string
	Tags          map[string]string
}

type LaunchTemplate struct {
	CreatedBy            string
	CreateTime           time.Time
	DefaultVersionNumber int64
	LatestVersionNumber  int64
	LaunchTemplateID     string
	LaunchTemplateName   string
	Tags                 map[string]string
}

type LaunchTemplateVersion struct {
	CreatedBy          string
	CreateTime         time.Time
	DefaultVersion     bool
	LaunchTemplateID   string
	LaunchTemplateName string
	VersionDescription string
	VersionNumber      int64
}

type LocalGatewayRoute struct {
	CoipPoolID                          string
	DestinationCidrBlock                string
	DestinationPrefixListID             string
	LocalGatewayRouteTableARN           string
	LocalGatewayRouteTableID            string
	LocalGatewayVirtualInterfaceGroupID string
	NetworkInterfaceID                  string
	OwnerID                             string
	State                               string
	SubnetID                            string
	Type                                string
}

type LocalGatewayRouteTable struct {
	LocalGatewayID            string
	LocalGatewayRouteTableARN string
	LocalGatewayRouteTableID  string
	Mode                      string
	OutpostARN                string
	OwnerID                   string
	State                     string
	Tags                      map[string]string
}

type LocalGatewayRouteTableVirtualInterfaceGroupAssociation struct {
	LocalGatewayID                                           string
	LocalGatewayRouteTableARN                                string
	LocalGatewayRouteTableID                                 string
	LocalGatewayRouteTableVirtualInterfaceGroupAssociationID string
	LocalGatewayVirtualInterfaceGroupID                      string
	OwnerID                                                  string
	State                                                    string
	Tags                                                     map[string]string
}

type LocalGatewayRouteTableVpcAssociation struct {
	LocalGatewayID                         string
	LocalGatewayRouteTableARN              string
	LocalGatewayRouteTableID               string
	LocalGatewayRouteTableVpcAssociationID string
	OwnerID                                string
	State                                  string
	Tags                                   map[string]string
	VpcID                                  string
}

type LocalGatewayVirtualInterface struct {
	ConfigurationState                  string
	LocalAddress                        string
	LocalBgpASN                         *int32
	LocalGatewayID                      string
	LocalGatewayVirtualInterfaceARN     string
	LocalGatewayVirtualInterfaceGroupID string
	LocalGatewayVirtualInterfaceID      string
	OutpostLagID                        string
	OwnerID                             string
	PeerAddress                         string
	PeerBgpASN                          *int32
	PeerBgpASNExtended                  *int64
	Tags                                map[string]string
	VLAN                                *int32
}

func (s *Service) CreateIpamPool(
	addressFamily string,
	ipamScopeID string,
	allocationDefaultNetmaskLength *int32,
	allocationMaxNetmaskLength *int32,
	allocationMinNetmaskLength *int32,
	autoImport *bool,
	awsService string,
	description *string,
	locale *string,
	publicIPSource string,
	publiclyAdvertisable *bool,
	sourceIpamPoolID *string,
	tags []Tag,
) (IpamPool, error) {
	addressFamily = strings.ToLower(strings.TrimSpace(addressFamily))
	ipamScopeID = strings.TrimSpace(ipamScopeID)
	awsService = strings.TrimSpace(awsService)
	publicIPSource = strings.TrimSpace(publicIPSource)
	if addressFamily == "" || ipamScopeID == "" {
		return IpamPool{}, ErrInvalidParameter
	}
	if addressFamily != "ipv4" && addressFamily != "ipv6" {
		return IpamPool{}, ErrInvalidParameter
	}
	if !validStage108Netmask(allocationDefaultNetmaskLength, addressFamily) ||
		!validStage108Netmask(allocationMaxNetmaskLength, addressFamily) ||
		!validStage108Netmask(allocationMinNetmaskLength, addressFamily) {
		return IpamPool{}, ErrInvalidParameter
	}
	if allocationMaxNetmaskLength != nil && allocationMinNetmaskLength != nil && *allocationMinNetmaskLength > *allocationMaxNetmaskLength {
		return IpamPool{}, ErrInvalidParameter
	}
	if publicIPSource == "" {
		publicIPSource = "byoip"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope := s.ipamScopes[ipamScopeID]
	if scope == nil {
		return IpamPool{}, ErrNotFound
	}

	sourcePoolID := strings.TrimSpace(derefString(sourceIpamPoolID))
	if sourcePoolID != "" && s.ipamPools[sourcePoolID] == nil {
		return IpamPool{}, ErrNotFound
	}

	poolID := s.nextIDLocked("ipam-pool")
	autoImportValue := false
	if autoImport != nil {
		autoImportValue = *autoImport
	}
	publiclyAdvertisableValue := false
	if publiclyAdvertisable != nil {
		publiclyAdvertisableValue = *publiclyAdvertisable
	}
	pool := &IpamPool{
		AddressFamily:                  addressFamily,
		AllocationDefaultNetmaskLength: cloneInt32Pointer(allocationDefaultNetmaskLength),
		AllocationMaxNetmaskLength:     cloneInt32Pointer(allocationMaxNetmaskLength),
		AllocationMinNetmaskLength:     cloneInt32Pointer(allocationMinNetmaskLength),
		AutoImport:                     autoImportValue,
		AwsService:                     awsService,
		Description:                    strings.TrimSpace(derefString(description)),
		IpamARN:                        scope.IpamARN,
		IpamPoolARN:                    fmt.Sprintf("arn:aws:ec2:%s:%s:ipam-pool/%s", DefaultRegion, DefaultAccountID, poolID),
		IpamPoolID:                     poolID,
		IpamRegion:                     scope.IpamRegion,
		IpamScopeARN:                   scope.IpamScopeARN,
		IpamScopeID:                    scope.IpamScopeID,
		IpamScopeType:                  scope.IpamScopeType,
		Locale:                         strings.TrimSpace(derefString(locale)),
		OwnerID:                        DefaultAccountID,
		PublicIpSource:                 publicIPSource,
		PubliclyAdvertisable:           publiclyAdvertisableValue,
		SourceIpamPoolID:               sourcePoolID,
		State:                          "create-complete",
		Tags:                           tagsToMap(normalizeEC2Tags(tags)),
	}
	s.ipamPools[poolID] = pool
	scope.PoolCount++
	return cloneStage108IpamPool(pool), nil
}

func (s *Service) CreateIpamResourceDiscovery(
	description *string,
	operatingRegions []string,
	tags []Tag,
) (IpamResourceDiscovery, error) {
	operatingRegions = dedupeTrimmedStrings(operatingRegions)
	if len(operatingRegions) == 0 {
		operatingRegions = []string{DefaultRegion}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("ipam-rd")
	discovery := &IpamResourceDiscovery{
		Description:              strings.TrimSpace(derefString(description)),
		IpamResourceDiscoveryARN: fmt.Sprintf("arn:aws:ec2:%s:%s:ipam-resource-discovery/%s", DefaultRegion, DefaultAccountID, id),
		IpamResourceDiscoveryID:  id,
		IpamResourceRegion:       DefaultRegion,
		IsDefault:                false,
		OperatingRegions:         append([]string(nil), operatingRegions...),
		OwnerID:                  DefaultAccountID,
		State:                    "create-complete",
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
	}
	s.ipamResourceDiscoveries[id] = discovery
	return cloneStage108IpamResourceDiscovery(discovery), nil
}

func (s *Service) CreateIpamScope(
	ipamID string,
	description *string,
	tags []Tag,
) (IpamScope, error) {
	ipamID = strings.TrimSpace(ipamID)
	if ipamID == "" {
		return IpamScope{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ipam := s.ipams[ipamID]
	if ipam == nil {
		return IpamScope{}, ErrNotFound
	}

	scopeID := s.nextIDLocked("ipam-scope")
	scope := &IpamScope{
		Description:   strings.TrimSpace(derefString(description)),
		IpamARN:       ipam.IpamARN,
		IpamID:        ipamID,
		IpamRegion:    ipam.IpamRegion,
		IpamScopeARN:  fmt.Sprintf("arn:aws:ec2:%s:%s:ipam-scope/%s", DefaultRegion, DefaultAccountID, scopeID),
		IpamScopeID:   scopeID,
		IpamScopeType: "private",
		IsDefault:     false,
		OwnerID:       DefaultAccountID,
		PoolCount:     0,
		State:         "create-complete",
		Tags:          tagsToMap(normalizeEC2Tags(tags)),
	}
	s.ipamScopes[scopeID] = scope
	return cloneStage108IpamScope(scope), nil
}

func (s *Service) CreateLaunchTemplate(
	launchTemplateName string,
	hasLaunchTemplateData bool,
	clientToken *string,
	versionDescription *string,
	tags []Tag,
) (LaunchTemplate, LaunchTemplateVersion, error) {
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	if launchTemplateName == "" || !hasLaunchTemplateData {
		return LaunchTemplate{}, LaunchTemplateVersion{}, ErrInvalidParameter
	}
	_ = clientToken

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.launchTemplateNameIndex[launchTemplateName]; exists {
		return LaunchTemplate{}, LaunchTemplateVersion{}, ErrAlreadyExists
	}

	launchTemplateID := s.nextIDLocked("lt")
	now := time.Now().UTC()
	createdBy := fmt.Sprintf("arn:aws:iam::%s:root", DefaultAccountID)
	template := &LaunchTemplate{
		CreatedBy:            createdBy,
		CreateTime:           now,
		DefaultVersionNumber: 1,
		LatestVersionNumber:  1,
		LaunchTemplateID:     launchTemplateID,
		LaunchTemplateName:   launchTemplateName,
		Tags:                 tagsToMap(normalizeEC2Tags(tags)),
	}
	version := &LaunchTemplateVersion{
		CreatedBy:          createdBy,
		CreateTime:         now,
		DefaultVersion:     true,
		LaunchTemplateID:   launchTemplateID,
		LaunchTemplateName: launchTemplateName,
		VersionDescription: strings.TrimSpace(derefString(versionDescription)),
		VersionNumber:      1,
	}

	s.launchTemplates[launchTemplateID] = template
	s.launchTemplateNameIndex[launchTemplateName] = launchTemplateID
	s.launchTemplateVersions[launchTemplateID] = map[int64]*LaunchTemplateVersion{1: version}
	return cloneStage108LaunchTemplate(template), cloneStage108LaunchTemplateVersion(version), nil
}

func (s *Service) CreateLaunchTemplateVersion(
	launchTemplateID string,
	launchTemplateName string,
	hasLaunchTemplateData bool,
	sourceVersion *string,
	versionDescription *string,
	resolveAlias *bool,
	clientToken *string,
) (LaunchTemplateVersion, error) {
	launchTemplateID = strings.TrimSpace(launchTemplateID)
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	if !hasLaunchTemplateData {
		return LaunchTemplateVersion{}, ErrInvalidParameter
	}
	rawSourceVersion := strings.TrimSpace(derefString(sourceVersion))
	if rawSourceVersion != "" && rawSourceVersion != "$Latest" && rawSourceVersion != "$Default" {
		if _, err := strconv.ParseInt(rawSourceVersion, 10, 64); err != nil {
			return LaunchTemplateVersion{}, ErrInvalidParameter
		}
	}
	_ = resolveAlias
	_ = clientToken
	if launchTemplateID == "" && launchTemplateName == "" {
		return LaunchTemplateVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if launchTemplateID == "" {
		launchTemplateID = s.launchTemplateNameIndex[launchTemplateName]
	}
	template := s.launchTemplates[launchTemplateID]
	if template == nil {
		return LaunchTemplateVersion{}, ErrNotFound
	}
	if launchTemplateName != "" && template.LaunchTemplateName != launchTemplateName {
		return LaunchTemplateVersion{}, ErrInvalidParameter
	}

	versionNumber := template.LatestVersionNumber + 1
	version := &LaunchTemplateVersion{
		CreatedBy:          template.CreatedBy,
		CreateTime:         time.Now().UTC(),
		DefaultVersion:     false,
		LaunchTemplateID:   template.LaunchTemplateID,
		LaunchTemplateName: template.LaunchTemplateName,
		VersionDescription: strings.TrimSpace(derefString(versionDescription)),
		VersionNumber:      versionNumber,
	}
	template.LatestVersionNumber = versionNumber
	if s.launchTemplateVersions[launchTemplateID] == nil {
		s.launchTemplateVersions[launchTemplateID] = map[int64]*LaunchTemplateVersion{}
	}
	s.launchTemplateVersions[launchTemplateID][versionNumber] = version
	return cloneStage108LaunchTemplateVersion(version), nil
}

func (s *Service) CreateLocalGatewayRoute(
	localGatewayRouteTableID string,
	destinationCidrBlock *string,
	destinationPrefixListID *string,
	localGatewayVirtualInterfaceGroupID *string,
	networkInterfaceID *string,
) (LocalGatewayRoute, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	if localGatewayRouteTableID == "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}

	destinationCIDR := strings.TrimSpace(derefString(destinationCidrBlock))
	destinationPrefix := strings.TrimSpace(derefString(destinationPrefixListID))
	if destinationCIDR == "" && destinationPrefix == "" {
		destinationCIDR = "0.0.0.0/0"
	}
	if destinationCIDR != "" && destinationPrefix != "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	virtualInterfaceGroupID := strings.TrimSpace(derefString(localGatewayVirtualInterfaceGroupID))
	networkIF := strings.TrimSpace(derefString(networkInterfaceID))

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTable := s.localGatewayRouteTables[localGatewayRouteTableID]
	if routeTable == nil {
		return LocalGatewayRoute{}, ErrNotFound
	}

	key := strings.Join([]string{localGatewayRouteTableID, destinationCIDR, destinationPrefix, virtualInterfaceGroupID, networkIF}, "|")
	if _, exists := s.localGatewayRoutes[key]; exists {
		return LocalGatewayRoute{}, ErrAlreadyExists
	}
	route := &LocalGatewayRoute{
		DestinationCidrBlock:                destinationCIDR,
		DestinationPrefixListID:             destinationPrefix,
		LocalGatewayRouteTableARN:           routeTable.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:            routeTable.LocalGatewayRouteTableID,
		LocalGatewayVirtualInterfaceGroupID: virtualInterfaceGroupID,
		NetworkInterfaceID:                  networkIF,
		OwnerID:                             DefaultAccountID,
		State:                               "active",
		Type:                                "static",
	}
	s.localGatewayRoutes[key] = route
	return cloneStage108LocalGatewayRoute(route), nil
}

func (s *Service) CreateLocalGatewayRouteTable(
	localGatewayID string,
	mode string,
	tags []Tag,
) (LocalGatewayRouteTable, error) {
	localGatewayID = strings.TrimSpace(localGatewayID)
	mode = strings.TrimSpace(mode)
	if localGatewayID == "" {
		return LocalGatewayRouteTable{}, ErrInvalidParameter
	}
	if mode == "" {
		mode = "direct-vpc-routing"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTableID := s.nextIDLocked("lgw-rtb")
	routeTable := &LocalGatewayRouteTable{
		LocalGatewayID:            localGatewayID,
		LocalGatewayRouteTableARN: fmt.Sprintf("arn:aws:ec2:%s:%s:local-gateway-route-table/%s", DefaultRegion, DefaultAccountID, routeTableID),
		LocalGatewayRouteTableID:  routeTableID,
		Mode:                      mode,
		OutpostARN:                fmt.Sprintf("arn:aws:outposts:%s:%s:outpost/op-00000000", DefaultRegion, DefaultAccountID),
		OwnerID:                   DefaultAccountID,
		State:                     "available",
		Tags:                      tagsToMap(normalizeEC2Tags(tags)),
	}
	s.localGatewayRouteTables[routeTableID] = routeTable
	return cloneStage108LocalGatewayRouteTable(routeTable), nil
}

func (s *Service) CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation(
	localGatewayRouteTableID string,
	localGatewayVirtualInterfaceGroupID string,
	tags []Tag,
) (LocalGatewayRouteTableVirtualInterfaceGroupAssociation, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	localGatewayVirtualInterfaceGroupID = strings.TrimSpace(localGatewayVirtualInterfaceGroupID)
	if localGatewayRouteTableID == "" || localGatewayVirtualInterfaceGroupID == "" {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTable := s.localGatewayRouteTables[localGatewayRouteTableID]
	if routeTable == nil {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}, ErrNotFound
	}

	key := localGatewayRouteTableID + "|" + localGatewayVirtualInterfaceGroupID
	if _, exists := s.localGatewayRouteTableVifAssociations[key]; exists {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}, ErrAlreadyExists
	}
	associationID := s.nextIDLocked("lgw-vif-assoc")
	association := &LocalGatewayRouteTableVirtualInterfaceGroupAssociation{
		LocalGatewayID:            routeTable.LocalGatewayID,
		LocalGatewayRouteTableARN: routeTable.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:  routeTable.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVirtualInterfaceGroupAssociationID: associationID,
		LocalGatewayVirtualInterfaceGroupID:                      localGatewayVirtualInterfaceGroupID,
		OwnerID:                                                  DefaultAccountID,
		State:                                                    "associated",
		Tags:                                                     tagsToMap(normalizeEC2Tags(tags)),
	}
	s.localGatewayRouteTableVifAssociations[key] = association
	return cloneStage108LocalGatewayRouteTableVirtualInterfaceGroupAssociation(association), nil
}

func (s *Service) CreateLocalGatewayRouteTableVpcAssociation(
	localGatewayRouteTableID string,
	vpcID string,
	tags []Tag,
) (LocalGatewayRouteTableVpcAssociation, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	vpcID = strings.TrimSpace(vpcID)
	if localGatewayRouteTableID == "" || vpcID == "" {
		return LocalGatewayRouteTableVpcAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTable := s.localGatewayRouteTables[localGatewayRouteTableID]
	if routeTable == nil {
		return LocalGatewayRouteTableVpcAssociation{}, ErrNotFound
	}
	if s.vpcs[vpcID] == nil {
		return LocalGatewayRouteTableVpcAssociation{}, ErrNotFound
	}

	key := localGatewayRouteTableID + "|" + vpcID
	if _, exists := s.localGatewayRouteTableVpcAssociations[key]; exists {
		return LocalGatewayRouteTableVpcAssociation{}, ErrAlreadyExists
	}
	associationID := s.nextIDLocked("lgw-vpc-assoc")
	association := &LocalGatewayRouteTableVpcAssociation{
		LocalGatewayID:                         routeTable.LocalGatewayID,
		LocalGatewayRouteTableARN:              routeTable.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:               routeTable.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVpcAssociationID: associationID,
		OwnerID:                                DefaultAccountID,
		State:                                  "associated",
		Tags:                                   tagsToMap(normalizeEC2Tags(tags)),
		VpcID:                                  vpcID,
	}
	s.localGatewayRouteTableVpcAssociations[key] = association
	return cloneStage108LocalGatewayRouteTableVpcAssociation(association), nil
}

func (s *Service) CreateLocalGatewayVirtualInterface(
	localAddress string,
	localGatewayVirtualInterfaceGroupID string,
	outpostLagID string,
	peerAddress string,
	vlan int32,
	peerBgpASN *int32,
	peerBgpASNExtended *int64,
	tags []Tag,
) (LocalGatewayVirtualInterface, error) {
	localAddress = strings.TrimSpace(localAddress)
	localGatewayVirtualInterfaceGroupID = strings.TrimSpace(localGatewayVirtualInterfaceGroupID)
	outpostLagID = strings.TrimSpace(outpostLagID)
	peerAddress = strings.TrimSpace(peerAddress)
	if localAddress == "" || localGatewayVirtualInterfaceGroupID == "" || outpostLagID == "" || peerAddress == "" || vlan <= 0 {
		return LocalGatewayVirtualInterface{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	virtualInterfaceID := s.nextIDLocked("lgw-vif")
	localASN := int32(64512)
	peerASN := int32(64513)
	if peerBgpASN != nil {
		peerASN = *peerBgpASN
	}
	virtualInterface := &LocalGatewayVirtualInterface{
		ConfigurationState:                  "pending",
		LocalAddress:                        localAddress,
		LocalBgpASN:                         &localASN,
		LocalGatewayID:                      "lgw-00000000",
		LocalGatewayVirtualInterfaceARN:     fmt.Sprintf("arn:aws:ec2:%s:%s:local-gateway-virtual-interface/%s", DefaultRegion, DefaultAccountID, virtualInterfaceID),
		LocalGatewayVirtualInterfaceGroupID: localGatewayVirtualInterfaceGroupID,
		LocalGatewayVirtualInterfaceID:      virtualInterfaceID,
		OutpostLagID:                        outpostLagID,
		OwnerID:                             DefaultAccountID,
		PeerAddress:                         peerAddress,
		PeerBgpASN:                          &peerASN,
		PeerBgpASNExtended:                  cloneInt64Pointer(peerBgpASNExtended),
		Tags:                                tagsToMap(normalizeEC2Tags(tags)),
		VLAN:                                &vlan,
	}
	s.localGatewayVirtualInterfaces[virtualInterfaceID] = virtualInterface
	return cloneStage108LocalGatewayVirtualInterface(virtualInterface), nil
}

func cloneStage108IpamPool(in *IpamPool) IpamPool {
	if in == nil {
		return IpamPool{}
	}
	return IpamPool{
		AddressFamily:                  in.AddressFamily,
		AllocationDefaultNetmaskLength: cloneInt32Pointer(in.AllocationDefaultNetmaskLength),
		AllocationMaxNetmaskLength:     cloneInt32Pointer(in.AllocationMaxNetmaskLength),
		AllocationMinNetmaskLength:     cloneInt32Pointer(in.AllocationMinNetmaskLength),
		AutoImport:                     in.AutoImport,
		AwsService:                     in.AwsService,
		Description:                    in.Description,
		IpamARN:                        in.IpamARN,
		IpamPoolARN:                    in.IpamPoolARN,
		IpamPoolID:                     in.IpamPoolID,
		IpamRegion:                     in.IpamRegion,
		IpamScopeARN:                   in.IpamScopeARN,
		IpamScopeID:                    in.IpamScopeID,
		IpamScopeType:                  in.IpamScopeType,
		Locale:                         in.Locale,
		OwnerID:                        in.OwnerID,
		PublicIpSource:                 in.PublicIpSource,
		PubliclyAdvertisable:           in.PubliclyAdvertisable,
		SourceIpamPoolID:               in.SourceIpamPoolID,
		State:                          in.State,
		Tags:                           cloneStringMap(in.Tags),
	}
}

func cloneStage108IpamResourceDiscovery(in *IpamResourceDiscovery) IpamResourceDiscovery {
	if in == nil {
		return IpamResourceDiscovery{}
	}
	return IpamResourceDiscovery{
		Description:              in.Description,
		IpamResourceDiscoveryARN: in.IpamResourceDiscoveryARN,
		IpamResourceDiscoveryID:  in.IpamResourceDiscoveryID,
		IpamResourceRegion:       in.IpamResourceRegion,
		IsDefault:                in.IsDefault,
		OperatingRegions:         append([]string(nil), in.OperatingRegions...),
		OwnerID:                  in.OwnerID,
		State:                    in.State,
		Tags:                     cloneStringMap(in.Tags),
	}
}

func cloneStage108IpamScope(in *IpamScope) IpamScope {
	if in == nil {
		return IpamScope{}
	}
	return IpamScope{
		Description:   in.Description,
		IpamARN:       in.IpamARN,
		IpamID:        in.IpamID,
		IpamRegion:    in.IpamRegion,
		IpamScopeARN:  in.IpamScopeARN,
		IpamScopeID:   in.IpamScopeID,
		IpamScopeType: in.IpamScopeType,
		IsDefault:     in.IsDefault,
		OwnerID:       in.OwnerID,
		PoolCount:     in.PoolCount,
		State:         in.State,
		Tags:          cloneStringMap(in.Tags),
	}
}

func cloneStage108LaunchTemplate(in *LaunchTemplate) LaunchTemplate {
	if in == nil {
		return LaunchTemplate{}
	}
	return LaunchTemplate{
		CreatedBy:            in.CreatedBy,
		CreateTime:           in.CreateTime,
		DefaultVersionNumber: in.DefaultVersionNumber,
		LatestVersionNumber:  in.LatestVersionNumber,
		LaunchTemplateID:     in.LaunchTemplateID,
		LaunchTemplateName:   in.LaunchTemplateName,
		Tags:                 cloneStringMap(in.Tags),
	}
}

func cloneStage108LaunchTemplateVersion(in *LaunchTemplateVersion) LaunchTemplateVersion {
	if in == nil {
		return LaunchTemplateVersion{}
	}
	return LaunchTemplateVersion{
		CreatedBy:          in.CreatedBy,
		CreateTime:         in.CreateTime,
		DefaultVersion:     in.DefaultVersion,
		LaunchTemplateID:   in.LaunchTemplateID,
		LaunchTemplateName: in.LaunchTemplateName,
		VersionDescription: in.VersionDescription,
		VersionNumber:      in.VersionNumber,
	}
}

func cloneStage108LocalGatewayRoute(in *LocalGatewayRoute) LocalGatewayRoute {
	if in == nil {
		return LocalGatewayRoute{}
	}
	return LocalGatewayRoute{
		CoipPoolID:                          in.CoipPoolID,
		DestinationCidrBlock:                in.DestinationCidrBlock,
		DestinationPrefixListID:             in.DestinationPrefixListID,
		LocalGatewayRouteTableARN:           in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:            in.LocalGatewayRouteTableID,
		LocalGatewayVirtualInterfaceGroupID: in.LocalGatewayVirtualInterfaceGroupID,
		NetworkInterfaceID:                  in.NetworkInterfaceID,
		OwnerID:                             in.OwnerID,
		State:                               in.State,
		SubnetID:                            in.SubnetID,
		Type:                                in.Type,
	}
}

func cloneStage108LocalGatewayRouteTable(in *LocalGatewayRouteTable) LocalGatewayRouteTable {
	if in == nil {
		return LocalGatewayRouteTable{}
	}
	return LocalGatewayRouteTable{
		LocalGatewayID:            in.LocalGatewayID,
		LocalGatewayRouteTableARN: in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:  in.LocalGatewayRouteTableID,
		Mode:                      in.Mode,
		OutpostARN:                in.OutpostARN,
		OwnerID:                   in.OwnerID,
		State:                     in.State,
		Tags:                      cloneStringMap(in.Tags),
	}
}

func cloneStage108LocalGatewayRouteTableVirtualInterfaceGroupAssociation(in *LocalGatewayRouteTableVirtualInterfaceGroupAssociation) LocalGatewayRouteTableVirtualInterfaceGroupAssociation {
	if in == nil {
		return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{}
	}
	return LocalGatewayRouteTableVirtualInterfaceGroupAssociation{
		LocalGatewayID:            in.LocalGatewayID,
		LocalGatewayRouteTableARN: in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:  in.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVirtualInterfaceGroupAssociationID: in.LocalGatewayRouteTableVirtualInterfaceGroupAssociationID,
		LocalGatewayVirtualInterfaceGroupID:                      in.LocalGatewayVirtualInterfaceGroupID,
		OwnerID:                                                  in.OwnerID,
		State:                                                    in.State,
		Tags:                                                     cloneStringMap(in.Tags),
	}
}

func cloneStage108LocalGatewayRouteTableVpcAssociation(in *LocalGatewayRouteTableVpcAssociation) LocalGatewayRouteTableVpcAssociation {
	if in == nil {
		return LocalGatewayRouteTableVpcAssociation{}
	}
	return LocalGatewayRouteTableVpcAssociation{
		LocalGatewayID:                         in.LocalGatewayID,
		LocalGatewayRouteTableARN:              in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:               in.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVpcAssociationID: in.LocalGatewayRouteTableVpcAssociationID,
		OwnerID:                                in.OwnerID,
		State:                                  in.State,
		Tags:                                   cloneStringMap(in.Tags),
		VpcID:                                  in.VpcID,
	}
}

func cloneStage108LocalGatewayVirtualInterface(in *LocalGatewayVirtualInterface) LocalGatewayVirtualInterface {
	if in == nil {
		return LocalGatewayVirtualInterface{}
	}
	return LocalGatewayVirtualInterface{
		ConfigurationState:                  in.ConfigurationState,
		LocalAddress:                        in.LocalAddress,
		LocalBgpASN:                         cloneInt32Pointer(in.LocalBgpASN),
		LocalGatewayID:                      in.LocalGatewayID,
		LocalGatewayVirtualInterfaceARN:     in.LocalGatewayVirtualInterfaceARN,
		LocalGatewayVirtualInterfaceGroupID: in.LocalGatewayVirtualInterfaceGroupID,
		LocalGatewayVirtualInterfaceID:      in.LocalGatewayVirtualInterfaceID,
		OutpostLagID:                        in.OutpostLagID,
		OwnerID:                             in.OwnerID,
		PeerAddress:                         in.PeerAddress,
		PeerBgpASN:                          cloneInt32Pointer(in.PeerBgpASN),
		PeerBgpASNExtended:                  cloneInt64Pointer(in.PeerBgpASNExtended),
		Tags:                                cloneStringMap(in.Tags),
		VLAN:                                cloneInt32Pointer(in.VLAN),
	}
}

func validStage108Netmask(value *int32, addressFamily string) bool {
	if value == nil {
		return true
	}
	if *value < 0 {
		return false
	}
	limit := int32(128)
	if strings.EqualFold(addressFamily, "ipv4") {
		limit = 32
	}
	return *value <= limit
}
