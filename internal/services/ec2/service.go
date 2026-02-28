package ec2

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("already exists")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
)

const (
	DefaultRegion        = "us-east-1"
	DefaultAccountID     = "123456789012"
	defaultDHCPOptionsID = "dopt-00000000"
	defaultVPCID         = "vpc-00000001"
	defaultSubnetID      = "subnet-00000001"
	defaultRouteTableID  = "rtb-00000001"
	defaultNetworkACLID  = "acl-00000001"
)

type Tag struct {
	Key   string
	Value string
}

type Instance struct {
	ID                                string
	ImageID                           string
	InstanceType                      string
	StateCode                         int32
	StateName                         string
	LaunchTime                        time.Time
	AvailabilityZone                  string
	PrivateIP                         string
	PublicIP                          string
	SubnetID                          string
	VpcID                             string
	KeyName                           string
	DisableAPITermination             bool
	SourceDestCheck                   bool
	InstanceInitiatedShutdownBehavior string
	UserData                          string
	MonitoringState                   string
	SecurityGroupIDs                  []string
	Tags                              map[string]string
}

type Reservation struct {
	ID          string
	OwnerID     string
	GroupIDs    []string
	InstanceIDs []string
}

type ReservationResult struct {
	Reservation Reservation
	Instances   []Instance
}

type InstanceStateChange struct {
	InstanceID   string
	PreviousCode int32
	PreviousName string
	CurrentCode  int32
	CurrentName  string
}

type InstanceStatus struct {
	InstanceID       string
	AvailabilityZone string
	StateCode        int32
	StateName        string
	SystemStatus     string
	InstanceStatus   string
}

type ResourceTag struct {
	ResourceID   string
	ResourceType string
	Key          string
	Value        string
}

type IPPermission struct {
	Protocol string
	FromPort int32
	ToPort   int32
	CidrIP   string
	// Description is optional rule metadata surfaced by describe/update APIs.
	Description string
}

type SecurityGroup struct {
	ID          string
	Name        string
	Description string
	VpcID       string
	Ingress     []IPPermission
	Egress      []IPPermission
	Tags        map[string]string
}

type VolumeAttachment struct {
	VolumeID   string
	InstanceID string
	Device     string
	State      string
	AttachTime time.Time
}

type Volume struct {
	ID               string
	AvailabilityZone string
	SizeGiB          int32
	SnapshotID       string
	State            string
	VolumeType       string
	Iops             int32
	Throughput       int32
	MultiAttach      bool
	AutoEnableIO     bool
	CreateTime       time.Time
	Attachments      []VolumeAttachment
	Tags             map[string]string
}

type Snapshot struct {
	ID          string
	VolumeID    string
	State       string
	StartTime   time.Time
	Progress    string
	Description string
	VolumeSize  int32
	Tags        map[string]string
}

type Region struct {
	Name     string
	Endpoint string
}

type AvailabilityZone struct {
	Name   string
	Region string
	State  string
	ZoneID string
}

type DedicatedHost struct {
	ID               string
	AvailabilityZone string
	InstanceType     string
	InstanceFamily   string
	AutoPlacement    string
	HostRecovery     string
	HostMaintenance  string
}

type EnclaveCertificateRoleAssociation struct {
	AssociatedRoleArn       string
	CertificateS3BucketName string
	CertificateS3ObjectKey  string
	EncryptionKmsKeyID      string
}

type InstanceEventWindowAssociation struct {
	InstanceEventWindowID string
	DedicatedHostIDs      []string
	InstanceIDs           []string
	InstanceTags          []Tag
	State                 string
}

type IpamResourceDiscoveryAssociation struct {
	IpamARN                             string
	IpamID                              string
	IpamRegion                          string
	IpamResourceDiscoveryAssociationARN string
	IpamResourceDiscoveryAssociationID  string
	IpamResourceDiscoveryID             string
	IsDefault                           bool
	OwnerID                             string
	ResourceDiscoveryStatus             string
	State                               string
	Tags                                []Tag
}

type BundleStorage struct {
	AWSAccessKeyID        string
	Bucket                string
	Prefix                string
	UploadPolicy          string
	UploadPolicySignature string
}

type BundleTaskError struct {
	Code    string
	Message string
}

type BundleTask struct {
	BundleID   string
	InstanceID string
	Progress   string
	StartTime  time.Time
	State      string
	Storage    BundleStorage
	UpdateTime time.Time
	Error      *BundleTaskError
}

type Service struct {
	mu                                        sync.Mutex
	seq                                       uint64
	ebsEncryptionByDefault                    bool
	ebsDefaultKMSKeyID                        string
	idFormatRoot                              map[string]bool
	idFormatByPrincipal                       map[string]map[string]bool
	instances                                 map[string]*Instance
	reservations                              map[string]*Reservation
	securityGroups                            map[string]*SecurityGroup
	securityGroupNameIndex                    map[string]string
	addresses                                 map[string]*ElasticAddress
	addressTransfers                          map[string]*AddressTransfer
	placementGroups                           map[string]*PlacementGroup
	placementGroupByName                      map[string]string
	customerGateways                          map[string]*CustomerGateway
	vpnGateways                               map[string]*VpnGateway
	vpnConnections                            map[string]*VpnConnection
	dhcpOptions                               map[string]*DHCPOptions
	egressOnlyGateways                        map[string]*EgressOnlyInternetGateway
	natGateways                               map[string]*NatGateway
	vpcPeeringConnections                     map[string]*VpcPeeringConnection
	networkIfacePerms                         map[string]*NetworkInterfacePermission
	keyPairs                                  map[string]*KeyPair
	instanceProfileAssocs                     map[string]*IamInstanceProfileAssociation
	instanceProfileByInst                     map[string]string
	clientVpnEndpoints                        map[string]*ClientVpnEndpoint
	clientVpnRoutes                           map[string]map[string]*ClientVpnRoute
	clientVpnTargetNetworks                   map[string]map[string]*ClientVpnTargetNetwork
	clientVpnAuthorizationRules               map[string]map[string]*ClientVpnAuthorizationRule
	clientVpnConnections                      map[string]map[string]*ClientVpnConnection
	clientVpnCertificateRevocationLists       map[string]string
	clientVpnCertificateRevocationListStatus  map[string]ClientCertificateRevocationListStatus
	routeTableVgwPropagations                 map[string]map[string]struct{}
	vpnActiveTunnelStatuses                   map[string]*ActiveVpnTunnelStatus
	vpnTunnelMaintenanceDetails               map[string]*MaintenanceDetails
	classicLinkAttachments                    map[string]*classicLinkAttachment
	vpcClassicLinkEnabled                     map[string]bool
	vpcClassicLinkDnsSupported                map[string]bool
	images                                    map[string]*Image
	volumes                                   map[string]*Volume
	snapshots                                 map[string]*Snapshot
	lockedSnapshots                           map[string]*LockedSnapshotInfo
	vpcs                                      map[string]*VPC
	subnets                                   map[string]*Subnet
	internetGateways                          map[string]*InternetGateway
	routeTables                               map[string]*RouteTable
	networkAcls                               map[string]*NetworkACL
	networkInterfaces                         map[string]*NetworkInterface
	vpcCidrAssociations                       map[string]*VpcCidrAssociation
	vpcIPv6CidrAssociations                   map[string]*VpcIPv6CidrAssociation
	subnetIPv6CidrAssociations                map[string]*SubnetIPv6CidrAssociation
	subnetCidrReservations                    map[string]*SubnetCidrReservation
	transitGateways                           map[string]*TransitGateway
	transitGatewayVpcAttachments              map[string]*TransitGatewayVpcAttachment
	transitGatewayPeeringAttachments          map[string]*TransitGatewayPeeringAttachment
	transitGatewayConnects                    map[string]*TransitGatewayConnect
	transitGatewayConnectPeers                map[string]*TransitGatewayConnectPeer
	transitGatewayMulticastDomains            map[string]*TransitGatewayMulticastDomain
	transitGatewayMulticastGroups             map[string]*TransitGatewayMulticastGroupRegistration
	transitGatewayPolicyTables                map[string]*TransitGatewayPolicyTable
	transitGatewayPrefixListReferences        map[string]*TransitGatewayPrefixListReference
	transitGatewayPropagations                map[string]*TransitGatewayPropagation
	transitGatewayRoutes                      map[string]*TransitGatewayRoute
	transitGatewayRouteTableAnnouncements     map[string]*TransitGatewayRouteTableAnnouncement
	transitGatewayRouteTables                 map[string]*TransitGatewayRouteTable
	vpcBlockPublicAccessOptions               VpcBlockPublicAccessOptions
	vpcBlockPublicAccessExclusions            map[string]*VpcBlockPublicAccessExclusion
	verifiedAccessInstances                   map[string]*VerifiedAccessInstance
	verifiedAccessGroups                      map[string]*VerifiedAccessGroup
	verifiedAccessEndpoints                   map[string]*VerifiedAccessEndpoint
	verifiedAccessTrustProviders              map[string]*VerifiedAccessTrustProvider
	verifiedAccessGroupPolicies               map[string]*VerifiedAccessPolicy
	verifiedAccessEndpointPolicies            map[string]*VerifiedAccessPolicy
	verifiedAccessInstanceLoggingConfigs      map[string]*VerifiedAccessInstanceLoggingConfiguration
	allowedImagesSettings                     AllowedImagesSettings
	imageBlockPublicAccessState               ImageBlockPublicAccessState
	snapshotBlockPublicAccessState            SnapshotBlockPublicAccessState
	fastLaunchConfigurations                  map[string]*FastLaunchConfiguration
	fastSnapshotRestoreStates                 map[string]map[string]bool
	serialConsoleAccessEnabled                bool
	serialConsoleManagedBy                    string
	ipamOrganizationAdminAccountID            string
	awsNetworkPerformanceMetricSubscriptions  map[string]AwsNetworkPerformanceSubscription
	reachabilityAnalyzerOrganizationSharing   bool
	byoipCidrs                                map[string]*ByoipCidr
	dedicatedHosts                            map[string]*DedicatedHost
	ipamPoolAllocations                       map[string]*IpamPoolAllocation
	capacityReservations                      map[string]*CapacityReservation
	capacityReservationFleets                 map[string]*CapacityReservationFleet
	carrierGateways                           map[string]*CarrierGateway
	coipCidrs                                 map[string]*CoipCidr
	coipPools                                 map[string]*CoipPool
	macModificationTasks                      map[string]*MacModificationTask
	fleets                                    map[string]*Fleet
	flowLogs                                  map[string]*FlowLog
	fpgaImages                                map[string]*FpgaImage
	instanceConnectEndpoints                  map[string]*InstanceConnectEndpoint
	instanceEventWindows                      map[string]*InstanceEventWindow
	instanceExportTasks                       map[string]*InstanceExportTask
	ipams                                     map[string]*Ipam
	ipamExternalResourceVerificationTokens    map[string]*IpamExternalResourceVerificationToken
	ipamPools                                 map[string]*IpamPool
	ipamResourceDiscoveries                   map[string]*IpamResourceDiscovery
	ipamScopes                                map[string]*IpamScope
	launchTemplates                           map[string]*LaunchTemplate
	launchTemplateNameIndex                   map[string]string
	launchTemplateVersions                    map[string]map[int64]*LaunchTemplateVersion
	localGatewayRoutes                        map[string]*LocalGatewayRoute
	localGatewayRouteTables                   map[string]*LocalGatewayRouteTable
	localGatewayRouteTableVifAssociations     map[string]*LocalGatewayRouteTableVirtualInterfaceGroupAssociation
	localGatewayRouteTableVpcAssociations     map[string]*LocalGatewayRouteTableVpcAssociation
	localGatewayVirtualInterfaceGroups        map[string]*LocalGatewayVirtualInterfaceGroup
	localGatewayVirtualInterfaces             map[string]*LocalGatewayVirtualInterface
	managedPrefixLists                        map[string]*ManagedPrefixList
	networkInsightsAccessScopes               map[string]*NetworkInsightsAccessScope
	networkInsightsPaths                      map[string]*NetworkInsightsPath
	publicIpv4Pools                           map[string]*PublicIpv4Pool
	replaceRootVolumeTasks                    map[string]*ReplaceRootVolumeTask
	capacityReservationBillingOwners          map[string]string
	enclaveCertificateRoleAssociations        map[string]map[string]EnclaveCertificateRoleAssociation
	instanceEventWindowAssociations           map[string]InstanceEventWindowAssociation
	ipamResourceDiscoveryAssociations         map[string]*IpamResourceDiscoveryAssociation
	ipamResourceDiscoveryAssociationByPair    map[string]string
	bundleTasks                               map[string]*BundleTask
	cancelledCapacityReservations             map[string]bool
	capacityReservationFleetStates            map[string]string
	defaultCreditSpecifications               map[string]string
	instanceCapacityReservationSpecifications map[string]InstanceCapacityReservationSpecification
	instanceCreditSpecifications              map[string]string
	instanceMaintenanceOptions                map[string]ModifiedInstanceMaintenanceOptions
	instanceMetadataDefaults                  InstanceMetadataDefaults
	instanceMetadataOptions                   map[string]ModifiedInstanceMetadataOptions
	instanceNetworkPerformanceOptions         map[string]string
	instancePlacementOptions                  map[string]instancePlacementOptions
	instanceStatusEvents                      map[string]map[string]ModifiedInstanceEvent
	conversionTaskStates                      map[string]string
	conversionTaskCancelReasons               map[string]string
	cancelledExportTasks                      map[string]bool
	cancelledImageLaunchPermissions           map[string]bool
	importTaskStates                          map[string]string
	importTaskCancelReasons                   map[string]string
	reservedInstancesListingStates            map[string]string
	reservedInstancesListingCreatedAt         map[string]time.Time
	spotFleetRequestStates                    map[string]string
	spotInstanceRequestStates                 map[string]string
	spotDatafeedSubscriptions                 map[string]*SpotDatafeedSubscription
	storeImageTasks                           map[string]*StoreImageTask
	trafficMirrorFilters                      map[string]*TrafficMirrorFilter
	trafficMirrorFilterRules                  map[string]*TrafficMirrorFilterRule
	trafficMirrorSessions                     map[string]*TrafficMirrorSession
	trafficMirrorTargets                      map[string]*TrafficMirrorTarget
	declarativePoliciesReports                map[string]*DeclarativePoliciesReport
	routeServers                              map[string]*RouteServer
	routeServerEndpoints                      map[string]*RouteServerEndpoint
	routeServerPeers                          map[string]*RouteServerPeer
	routeServerAssociations                   map[string]*RouteServerAssociation
	routeServerPropagations                   map[string]*RouteServerPropagation
	vpcEndpoints                              map[string]*VpcEndpoint
	vpcEndpointServiceConfigurations          map[string]*VpcEndpointServiceConfiguration
	vpcEndpointServicePayerResponsibility     map[string]string
	vpcEndpointServicePermissions             map[string]map[string]string
	vpcEndpointConnectionNotifications        map[string]*VpcEndpointConnectionNotification
	transitGatewayMulticastDomainAssocs       map[string]*TransitGatewayMulticastDomainAssociations
	transitGatewayPolicyTableAssocs           map[string]*TransitGatewayPolicyTableAssociation
	transitGatewayRouteTableAssocs            map[string]*TransitGatewayRouteTableAssociation
	trunkInterfaceAssociations                map[string]*TrunkInterfaceAssociation
	securityGroupVpcAssociations              map[string]*SecurityGroupVpcAssociation
	regions                                   []Region
	availabilityZones                         []AvailabilityZone
}

func NewService() *Service {
	s := &Service{
		seq:                                      1,
		ebsEncryptionByDefault:                   false,
		ebsDefaultKMSKeyID:                       "arn:aws:kms:" + DefaultRegion + ":" + DefaultAccountID + ":alias/aws/ebs",
		idFormatRoot:                             defaultIDFormatState(),
		idFormatByPrincipal:                      map[string]map[string]bool{},
		instances:                                map[string]*Instance{},
		reservations:                             map[string]*Reservation{},
		securityGroups:                           map[string]*SecurityGroup{},
		securityGroupNameIndex:                   map[string]string{},
		addresses:                                map[string]*ElasticAddress{},
		addressTransfers:                         map[string]*AddressTransfer{},
		placementGroups:                          map[string]*PlacementGroup{},
		placementGroupByName:                     map[string]string{},
		customerGateways:                         map[string]*CustomerGateway{},
		vpnGateways:                              map[string]*VpnGateway{},
		vpnConnections:                           map[string]*VpnConnection{},
		dhcpOptions:                              map[string]*DHCPOptions{},
		egressOnlyGateways:                       map[string]*EgressOnlyInternetGateway{},
		natGateways:                              map[string]*NatGateway{},
		vpcPeeringConnections:                    map[string]*VpcPeeringConnection{},
		networkIfacePerms:                        map[string]*NetworkInterfacePermission{},
		keyPairs:                                 map[string]*KeyPair{},
		instanceProfileAssocs:                    map[string]*IamInstanceProfileAssociation{},
		instanceProfileByInst:                    map[string]string{},
		clientVpnEndpoints:                       map[string]*ClientVpnEndpoint{},
		clientVpnRoutes:                          map[string]map[string]*ClientVpnRoute{},
		clientVpnTargetNetworks:                  map[string]map[string]*ClientVpnTargetNetwork{},
		clientVpnAuthorizationRules:              map[string]map[string]*ClientVpnAuthorizationRule{},
		clientVpnConnections:                     map[string]map[string]*ClientVpnConnection{},
		clientVpnCertificateRevocationLists:      map[string]string{},
		clientVpnCertificateRevocationListStatus: map[string]ClientCertificateRevocationListStatus{},
		routeTableVgwPropagations:                map[string]map[string]struct{}{},
		vpnActiveTunnelStatuses:                  map[string]*ActiveVpnTunnelStatus{},
		vpnTunnelMaintenanceDetails:              map[string]*MaintenanceDetails{},
		classicLinkAttachments:                   map[string]*classicLinkAttachment{},
		vpcClassicLinkEnabled:                    map[string]bool{},
		vpcClassicLinkDnsSupported:               map[string]bool{},
		images:                                   map[string]*Image{},
		volumes:                                  map[string]*Volume{},
		snapshots:                                map[string]*Snapshot{},
		lockedSnapshots:                          map[string]*LockedSnapshotInfo{},
		vpcs:                                     map[string]*VPC{},
		subnets:                                  map[string]*Subnet{},
		internetGateways:                         map[string]*InternetGateway{},
		routeTables:                              map[string]*RouteTable{},
		networkAcls:                              map[string]*NetworkACL{},
		networkInterfaces:                        map[string]*NetworkInterface{},
		vpcCidrAssociations:                      map[string]*VpcCidrAssociation{},
		vpcIPv6CidrAssociations:                  map[string]*VpcIPv6CidrAssociation{},
		subnetIPv6CidrAssociations:               map[string]*SubnetIPv6CidrAssociation{},
		subnetCidrReservations:                   map[string]*SubnetCidrReservation{},
		transitGateways:                          map[string]*TransitGateway{},
		transitGatewayVpcAttachments:             map[string]*TransitGatewayVpcAttachment{},
		transitGatewayPeeringAttachments:         map[string]*TransitGatewayPeeringAttachment{},
		transitGatewayConnects:                   map[string]*TransitGatewayConnect{},
		transitGatewayConnectPeers:               map[string]*TransitGatewayConnectPeer{},
		transitGatewayMulticastDomains:           map[string]*TransitGatewayMulticastDomain{},
		transitGatewayMulticastGroups:            map[string]*TransitGatewayMulticastGroupRegistration{},
		transitGatewayPolicyTables:               map[string]*TransitGatewayPolicyTable{},
		transitGatewayPrefixListReferences:       map[string]*TransitGatewayPrefixListReference{},
		transitGatewayPropagations:               map[string]*TransitGatewayPropagation{},
		transitGatewayRoutes:                     map[string]*TransitGatewayRoute{},
		transitGatewayRouteTableAnnouncements:    map[string]*TransitGatewayRouteTableAnnouncement{},
		transitGatewayRouteTables:                map[string]*TransitGatewayRouteTable{},
		transitGatewayMulticastDomainAssocs:      map[string]*TransitGatewayMulticastDomainAssociations{},
		transitGatewayPolicyTableAssocs:          map[string]*TransitGatewayPolicyTableAssociation{},
		transitGatewayRouteTableAssocs:           map[string]*TransitGatewayRouteTableAssociation{},
		trunkInterfaceAssociations:               map[string]*TrunkInterfaceAssociation{},
		securityGroupVpcAssociations:             map[string]*SecurityGroupVpcAssociation{},
		regions: []Region{
			{Name: "us-east-1", Endpoint: "ec2.us-east-1.amazonaws.com"},
			{Name: "us-west-2", Endpoint: "ec2.us-west-2.amazonaws.com"},
			{Name: "eu-west-1", Endpoint: "ec2.eu-west-1.amazonaws.com"},
		},
		availabilityZones: []AvailabilityZone{
			{Name: "us-east-1a", Region: "us-east-1", State: "available", ZoneID: "use1-az1"},
			{Name: "us-east-1b", Region: "us-east-1", State: "available", ZoneID: "use1-az2"},
			{Name: "us-west-2a", Region: "us-west-2", State: "available", ZoneID: "usw2-az1"},
		},
		vpcBlockPublicAccessOptions: VpcBlockPublicAccessOptions{
			AwsAccountID:             DefaultAccountID,
			AwsRegion:                DefaultRegion,
			ExclusionsAllowed:        "allowed",
			InternetGatewayBlockMode: "off",
			LastUpdateTimestamp:      time.Now().UTC(),
			ManagedBy:                "account",
			State:                    "default-state",
		},
		vpcBlockPublicAccessExclusions:       map[string]*VpcBlockPublicAccessExclusion{},
		verifiedAccessInstances:              map[string]*VerifiedAccessInstance{},
		verifiedAccessGroups:                 map[string]*VerifiedAccessGroup{},
		verifiedAccessEndpoints:              map[string]*VerifiedAccessEndpoint{},
		verifiedAccessTrustProviders:         map[string]*VerifiedAccessTrustProvider{},
		verifiedAccessGroupPolicies:          map[string]*VerifiedAccessPolicy{},
		verifiedAccessEndpointPolicies:       map[string]*VerifiedAccessPolicy{},
		verifiedAccessInstanceLoggingConfigs: map[string]*VerifiedAccessInstanceLoggingConfiguration{},
		allowedImagesSettings: AllowedImagesSettings{
			ManagedBy:     "account",
			State:         "disabled",
			ImageCriteria: []AllowedImageCriterion{},
		},
		imageBlockPublicAccessState: ImageBlockPublicAccessState{
			ImageBlockPublicAccessState: "unblocked",
			ManagedBy:                   "account",
		},
		snapshotBlockPublicAccessState: SnapshotBlockPublicAccessState{
			State:     "unblocked",
			ManagedBy: "account",
		},
		fastLaunchConfigurations:                  map[string]*FastLaunchConfiguration{},
		fastSnapshotRestoreStates:                 map[string]map[string]bool{},
		serialConsoleAccessEnabled:                false,
		serialConsoleManagedBy:                    "account",
		ipamOrganizationAdminAccountID:            "",
		awsNetworkPerformanceMetricSubscriptions:  map[string]AwsNetworkPerformanceSubscription{},
		reachabilityAnalyzerOrganizationSharing:   false,
		byoipCidrs:                                map[string]*ByoipCidr{},
		dedicatedHosts:                            map[string]*DedicatedHost{},
		ipamPoolAllocations:                       map[string]*IpamPoolAllocation{},
		capacityReservations:                      map[string]*CapacityReservation{},
		capacityReservationFleets:                 map[string]*CapacityReservationFleet{},
		carrierGateways:                           map[string]*CarrierGateway{},
		coipCidrs:                                 map[string]*CoipCidr{},
		coipPools:                                 map[string]*CoipPool{},
		macModificationTasks:                      map[string]*MacModificationTask{},
		fleets:                                    map[string]*Fleet{},
		flowLogs:                                  map[string]*FlowLog{},
		fpgaImages:                                map[string]*FpgaImage{},
		instanceConnectEndpoints:                  map[string]*InstanceConnectEndpoint{},
		instanceEventWindows:                      map[string]*InstanceEventWindow{},
		instanceExportTasks:                       map[string]*InstanceExportTask{},
		ipams:                                     map[string]*Ipam{},
		ipamExternalResourceVerificationTokens:    map[string]*IpamExternalResourceVerificationToken{},
		ipamPools:                                 map[string]*IpamPool{},
		ipamResourceDiscoveries:                   map[string]*IpamResourceDiscovery{},
		ipamScopes:                                map[string]*IpamScope{},
		launchTemplates:                           map[string]*LaunchTemplate{},
		launchTemplateNameIndex:                   map[string]string{},
		launchTemplateVersions:                    map[string]map[int64]*LaunchTemplateVersion{},
		localGatewayRoutes:                        map[string]*LocalGatewayRoute{},
		localGatewayRouteTables:                   map[string]*LocalGatewayRouteTable{},
		localGatewayRouteTableVifAssociations:     map[string]*LocalGatewayRouteTableVirtualInterfaceGroupAssociation{},
		localGatewayRouteTableVpcAssociations:     map[string]*LocalGatewayRouteTableVpcAssociation{},
		localGatewayVirtualInterfaceGroups:        map[string]*LocalGatewayVirtualInterfaceGroup{},
		localGatewayVirtualInterfaces:             map[string]*LocalGatewayVirtualInterface{},
		managedPrefixLists:                        map[string]*ManagedPrefixList{},
		networkInsightsAccessScopes:               map[string]*NetworkInsightsAccessScope{},
		networkInsightsPaths:                      map[string]*NetworkInsightsPath{},
		publicIpv4Pools:                           map[string]*PublicIpv4Pool{},
		replaceRootVolumeTasks:                    map[string]*ReplaceRootVolumeTask{},
		capacityReservationBillingOwners:          map[string]string{},
		enclaveCertificateRoleAssociations:        map[string]map[string]EnclaveCertificateRoleAssociation{},
		instanceEventWindowAssociations:           map[string]InstanceEventWindowAssociation{},
		ipamResourceDiscoveryAssociations:         map[string]*IpamResourceDiscoveryAssociation{},
		ipamResourceDiscoveryAssociationByPair:    map[string]string{},
		bundleTasks:                               map[string]*BundleTask{},
		cancelledCapacityReservations:             map[string]bool{},
		capacityReservationFleetStates:            map[string]string{},
		defaultCreditSpecifications:               map[string]string{},
		instanceCapacityReservationSpecifications: map[string]InstanceCapacityReservationSpecification{},
		instanceCreditSpecifications:              map[string]string{},
		instanceMaintenanceOptions:                map[string]ModifiedInstanceMaintenanceOptions{},
		instanceMetadataOptions:                   map[string]ModifiedInstanceMetadataOptions{},
		instanceNetworkPerformanceOptions:         map[string]string{},
		instancePlacementOptions:                  map[string]instancePlacementOptions{},
		instanceStatusEvents:                      map[string]map[string]ModifiedInstanceEvent{},
		conversionTaskStates:                      map[string]string{},
		conversionTaskCancelReasons:               map[string]string{},
		cancelledExportTasks:                      map[string]bool{},
		cancelledImageLaunchPermissions:           map[string]bool{},
		importTaskStates:                          map[string]string{},
		importTaskCancelReasons:                   map[string]string{},
		reservedInstancesListingStates:            map[string]string{},
		reservedInstancesListingCreatedAt:         map[string]time.Time{},
		spotFleetRequestStates:                    map[string]string{},
		spotInstanceRequestStates:                 map[string]string{},
		spotDatafeedSubscriptions:                 map[string]*SpotDatafeedSubscription{},
		storeImageTasks:                           map[string]*StoreImageTask{},
		trafficMirrorFilters:                      map[string]*TrafficMirrorFilter{},
		trafficMirrorFilterRules:                  map[string]*TrafficMirrorFilterRule{},
		trafficMirrorSessions:                     map[string]*TrafficMirrorSession{},
		trafficMirrorTargets:                      map[string]*TrafficMirrorTarget{},
		declarativePoliciesReports:                map[string]*DeclarativePoliciesReport{},
		routeServers:                              map[string]*RouteServer{},
		routeServerEndpoints:                      map[string]*RouteServerEndpoint{},
		routeServerPeers:                          map[string]*RouteServerPeer{},
		routeServerAssociations:                   map[string]*RouteServerAssociation{},
		routeServerPropagations:                   map[string]*RouteServerPropagation{},
		vpcEndpoints: map[string]*VpcEndpoint{
			"vpce-00000000": {
				ID:                "vpce-00000000",
				VpcID:             defaultVPCID,
				ServiceName:       "com.amazonaws.us-east-1.s3",
				ServiceRegion:     DefaultRegion,
				State:             "Available",
				OwnerID:           DefaultAccountID,
				VpcEndpointType:   "Gateway",
				RouteTableIDs:     []string{defaultRouteTableID},
				SecurityGroupIDs:  []string{"sg-00000000"},
				SubnetIDs:         []string{defaultSubnetID},
				PolicyDocument:    `{"Version":"2012-10-17","Statement":[]}`,
				PrivateDNSEnabled: true,
				IPAddressType:     "ipv4",
				CreationTimestamp: time.Now().UTC(),
				Tags:              map[string]string{},
			},
		},
		vpcEndpointServiceConfigurations: map[string]*VpcEndpointServiceConfiguration{
			"vpce-svc-00000000": {
				ServiceID:               "vpce-svc-00000000",
				ServiceName:             "com.amazonaws.vpce.us-east-1.vpce-svc-00000000",
				ServiceState:            "Available",
				PayerResponsibility:     "EndpointOwner",
				AcceptanceRequired:      false,
				SupportedIPAddressTypes: []string{"ipv4"},
				SupportedRegions:        []string{DefaultRegion},
			},
		},
		vpcEndpointServicePayerResponsibility: map[string]string{
			"vpce-svc-00000000": "EndpointOwner",
		},
		vpcEndpointServicePermissions: map[string]map[string]string{
			"vpce-svc-00000000": {},
		},
		vpcEndpointConnectionNotifications: map[string]*VpcEndpointConnectionNotification{},
	}
	now := time.Now().UTC()
	s.vpcEndpointConnectionNotifications["vpce-nfn-00000000"] = &VpcEndpointConnectionNotification{
		ConnectionNotificationID:    "vpce-nfn-00000000",
		ConnectionNotificationARN:   "arn:aws:sns:us-east-1:123456789012:stackyard-vpce-notify",
		ConnectionEvents:            []string{"Accept", "Connect"},
		ConnectionNotificationState: "Enabled",
		ConnectionNotificationType:  "Topic",
		ServiceID:                   "vpce-svc-00000000",
		ServiceRegion:               DefaultRegion,
	}
	s.vpcBlockPublicAccessExclusions["vpcbpa-ex-00000000"] = &VpcBlockPublicAccessExclusion{
		ExclusionID:                  "vpcbpa-ex-00000000",
		InternetGatewayExclusionMode: "allow-bidirectional",
		ResourceARN:                  fmt.Sprintf("arn:aws:ec2:%s:%s:vpc/%s", DefaultRegion, DefaultAccountID, defaultVPCID),
		State:                        "create-complete",
		CreationTimestamp:            now,
		LastUpdateTimestamp:          now,
		Tags:                         map[string]string{},
	}
	// default security group in default VPC
	defaultSG := &SecurityGroup{
		ID:          "sg-00000000",
		Name:        "default",
		Description: "default VPC security group",
		VpcID:       defaultVPCID,
		Ingress:     []IPPermission{},
		Egress:      []IPPermission{{Protocol: "-1", FromPort: -1, ToPort: -1, CidrIP: "0.0.0.0/0"}},
		Tags:        map[string]string{},
	}
	s.securityGroups[defaultSG.ID] = defaultSG
	s.securityGroupNameIndex[securityGroupNameKey(defaultSG.VpcID, defaultSG.Name)] = defaultSG.ID
	s.dhcpOptions[defaultDHCPOptionsID] = &DHCPOptions{
		ID: defaultDHCPOptionsID,
		Configurations: []DHCPConfiguration{
			{Key: "domain-name-servers", Values: []string{"AmazonProvidedDNS"}},
		},
		OwnerID: DefaultAccountID,
		Tags:    map[string]string{},
	}
	s.vpcs[defaultVPCID] = &VPC{
		ID:                 defaultVPCID,
		CidrBlock:          "10.0.0.0/16",
		State:              "available",
		InstanceTenancy:    "default",
		IsDefault:          true,
		DhcpOptionsID:      defaultDHCPOptionsID,
		EnableDnsSupport:   true,
		EnableDnsHostnames: true,
		Tags:               map[string]string{},
	}
	s.subnets[defaultSubnetID] = &Subnet{
		ID:                      defaultSubnetID,
		VpcID:                   defaultVPCID,
		CidrBlock:               "10.0.0.0/24",
		AvailabilityZone:        "us-east-1a",
		State:                   "available",
		AvailableIPAddressCount: 251,
		MapPublicIPOnLaunch:     true,
		Tags:                    map[string]string{},
	}
	s.routeTables[defaultRouteTableID] = &RouteTable{
		ID:    defaultRouteTableID,
		VpcID: defaultVPCID,
		Routes: []Route{
			{DestinationCIDR: "10.0.0.0/16", GatewayID: "local", State: "active", Origin: "CreateRouteTable"},
		},
		Associations: []RouteTableAssociation{
			{ID: "rtbassoc-00000001", Main: true},
		},
		Tags: map[string]string{},
	}
	s.networkAcls[defaultNetworkACLID] = &NetworkACL{
		ID:        defaultNetworkACLID,
		VpcID:     defaultVPCID,
		IsDefault: true,
		Entries: []NetworkACLEntry{
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: true, CidrBlock: "0.0.0.0/0"},
		},
		Associations: []NetworkACLAssociation{
			{ID: "aclassoc-00000001", SubnetID: defaultSubnetID},
		},
		Tags: map[string]string{},
	}
	defaultMetadataHopLimit := int32(2)
	s.instanceMetadataDefaults = InstanceMetadataDefaults{
		HttpEndpoint:            "enabled",
		HttpPutResponseHopLimit: &defaultMetadataHopLimit,
		HttpTokens:              "optional",
		InstanceMetadataTags:    "disabled",
		ManagedBy:               "account",
	}
	return s
}

func (s *Service) RunInstances(imageID, instanceType, keyName, subnetID, availabilityZone string, securityGroupIDs []string, minCount, maxCount int32, tags []Tag) (ReservationResult, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ReservationResult{}, ErrInvalidParameter
	}
	if minCount <= 0 || maxCount <= 0 || maxCount < minCount {
		return ReservationResult{}, ErrInvalidParameter
	}
	if strings.TrimSpace(instanceType) == "" {
		instanceType = "t3.micro"
	}
	if strings.TrimSpace(subnetID) == "" {
		subnetID = defaultSubnetID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	subnet := s.subnets[subnetID]
	if subnet == nil {
		return ReservationResult{}, ErrNotFound
	}
	if strings.TrimSpace(availabilityZone) == "" {
		availabilityZone = subnet.AvailabilityZone
	}
	vpcID := subnet.VpcID

	if len(securityGroupIDs) == 0 {
		securityGroupIDs = []string{"sg-00000000"}
	}
	resolvedSGs := make([]string, 0, len(securityGroupIDs))
	for _, id := range securityGroupIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if s.securityGroups[id] == nil {
			return ReservationResult{}, ErrNotFound
		}
		resolvedSGs = append(resolvedSGs, id)
	}
	if len(resolvedSGs) == 0 {
		resolvedSGs = []string{"sg-00000000"}
	}

	reservationID := s.nextIDLocked("r")
	reservation := &Reservation{
		ID:          reservationID,
		OwnerID:     DefaultAccountID,
		GroupIDs:    append([]string(nil), resolvedSGs...),
		InstanceIDs: []string{},
	}

	count := maxCount
	instances := make([]Instance, 0, count)
	for i := int32(0); i < count; i++ {
		instanceID := s.nextIDLocked("i")
		suffix := int(atomic.LoadUint64(&s.seq) % 250)
		if suffix == 0 {
			suffix = 10
		}
		inst := &Instance{
			ID:                                instanceID,
			ImageID:                           imageID,
			InstanceType:                      strings.TrimSpace(instanceType),
			StateCode:                         16,
			StateName:                         "running",
			LaunchTime:                        time.Now().UTC(),
			AvailabilityZone:                  availabilityZone,
			PrivateIP:                         fmt.Sprintf("10.0.0.%d", suffix),
			PublicIP:                          fmt.Sprintf("54.0.0.%d", suffix),
			SubnetID:                          subnetID,
			VpcID:                             vpcID,
			KeyName:                           strings.TrimSpace(keyName),
			DisableAPITermination:             false,
			SourceDestCheck:                   true,
			InstanceInitiatedShutdownBehavior: "stop",
			UserData:                          "",
			MonitoringState:                   "disabled",
			SecurityGroupIDs:                  append([]string(nil), resolvedSGs...),
			Tags:                              tagsToMap(tags),
		}
		s.instances[instanceID] = inst
		reservation.InstanceIDs = append(reservation.InstanceIDs, instanceID)
		instances = append(instances, cloneInstance(inst))
	}
	s.reservations[reservationID] = reservation

	return ReservationResult{Reservation: cloneReservation(reservation), Instances: instances}, nil
}

func (s *Service) DescribeInstances(instanceIDs []string) []ReservationResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(instanceIDs)
	reservationIDs := make([]string, 0, len(s.reservations))
	for id := range s.reservations {
		reservationIDs = append(reservationIDs, id)
	}
	sort.Strings(reservationIDs)

	out := make([]ReservationResult, 0, len(reservationIDs))
	for _, reservationID := range reservationIDs {
		reservation := s.reservations[reservationID]
		instances := make([]Instance, 0, len(reservation.InstanceIDs))
		for _, instanceID := range reservation.InstanceIDs {
			inst := s.instances[instanceID]
			if inst == nil {
				continue
			}
			if len(idSet) > 0 {
				if _, ok := idSet[inst.ID]; !ok {
					continue
				}
			}
			instances = append(instances, cloneInstance(inst))
		}
		if len(idSet) > 0 && len(instances) == 0 {
			continue
		}
		out = append(out, ReservationResult{Reservation: cloneReservation(reservation), Instances: instances})
	}
	return out
}

func (s *Service) StartInstances(instanceIDs []string) ([]InstanceStateChange, error) {
	return s.transitionInstances(instanceIDs, 16, "running")
}

func (s *Service) StopInstances(instanceIDs []string) ([]InstanceStateChange, error) {
	return s.transitionInstances(instanceIDs, 80, "stopped")
}

func (s *Service) RebootInstances(instanceIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(instanceIDs) == 0 {
		return ErrInvalidParameter
	}
	for _, id := range instanceIDs {
		inst := s.instances[strings.TrimSpace(id)]
		if inst == nil {
			return ErrNotFound
		}
		if inst.StateName == "terminated" {
			return ErrConflict
		}
	}
	return nil
}

func (s *Service) TerminateInstances(instanceIDs []string) ([]InstanceStateChange, error) {
	return s.transitionInstances(instanceIDs, 48, "terminated")
}

func (s *Service) DescribeInstanceStatus(instanceIDs []string, includeAll bool) []InstanceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(instanceIDs)
	out := make([]InstanceStatus, 0, len(s.instances))
	for _, inst := range s.instances {
		if len(idSet) > 0 {
			if _, ok := idSet[inst.ID]; !ok {
				continue
			}
		}
		if !includeAll && inst.StateName != "running" {
			continue
		}
		out = append(out, InstanceStatus{
			InstanceID:       inst.ID,
			AvailabilityZone: inst.AvailabilityZone,
			StateCode:        inst.StateCode,
			StateName:        inst.StateName,
			SystemStatus:     "ok",
			InstanceStatus:   "ok",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
	return out
}

func (s *Service) CreateTags(resourceIDs []string, tags []Tag) error {
	if len(resourceIDs) == 0 || len(tags) == 0 {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, resourceID := range resourceIDs {
		resourceID = strings.TrimSpace(resourceID)
		if resourceID == "" {
			continue
		}
		if err := s.applyTagsLocked(resourceID, tags); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteTags(resourceIDs []string, tags []Tag) error {
	if len(resourceIDs) == 0 {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, resourceID := range resourceIDs {
		resourceID = strings.TrimSpace(resourceID)
		if resourceID == "" {
			continue
		}
		if err := s.removeTagsLocked(resourceID, tags); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DescribeTags(resourceIDs []string) []ResourceTag {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(resourceIDs)
	out := make([]ResourceTag, 0)
	appendTags := func(resourceID, resourceType string, tags map[string]string) {
		if len(idSet) > 0 {
			if _, ok := idSet[resourceID]; !ok {
				return
			}
		}
		for k, v := range tags {
			out = append(out, ResourceTag{ResourceID: resourceID, ResourceType: resourceType, Key: k, Value: v})
		}
	}
	for _, inst := range s.instances {
		appendTags(inst.ID, "instance", inst.Tags)
	}
	for _, sg := range s.securityGroups {
		appendTags(sg.ID, "security-group", sg.Tags)
	}
	for _, volume := range s.volumes {
		appendTags(volume.ID, "volume", volume.Tags)
	}
	for _, image := range s.images {
		appendTags(image.ID, "image", image.Tags)
	}
	for _, address := range s.addresses {
		appendTags(address.AllocationID, "elastic-ip", address.Tags)
	}
	for _, options := range s.dhcpOptions {
		appendTags(options.ID, "dhcp-options", options.Tags)
	}
	for _, gateway := range s.egressOnlyGateways {
		appendTags(gateway.ID, "egress-only-internet-gateway", gateway.Tags)
	}
	for _, gateway := range s.natGateways {
		appendTags(gateway.ID, "natgateway", gateway.Tags)
	}
	for _, peering := range s.vpcPeeringConnections {
		appendTags(peering.ID, "vpc-peering-connection", peering.Tags)
	}
	for _, snapshot := range s.snapshots {
		appendTags(snapshot.ID, "snapshot", snapshot.Tags)
	}
	for _, vpc := range s.vpcs {
		appendTags(vpc.ID, "vpc", vpc.Tags)
	}
	for _, subnet := range s.subnets {
		appendTags(subnet.ID, "subnet", subnet.Tags)
	}
	for _, gateway := range s.internetGateways {
		appendTags(gateway.ID, "internet-gateway", gateway.Tags)
	}
	for _, table := range s.routeTables {
		appendTags(table.ID, "route-table", table.Tags)
	}
	for _, acl := range s.networkAcls {
		appendTags(acl.ID, "network-acl", acl.Tags)
	}
	for _, iface := range s.networkInterfaces {
		appendTags(iface.ID, "network-interface", iface.Tags)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ResourceID == out[j].ResourceID {
			return out[i].Key < out[j].Key
		}
		return out[i].ResourceID < out[j].ResourceID
	})
	return out
}

func (s *Service) DescribeRegions(regionNames []string) []Region {
	nameSet := toStringSet(regionNames)
	out := make([]Region, 0, len(s.regions))
	for _, region := range s.regions {
		if len(nameSet) > 0 {
			if _, ok := nameSet[region.Name]; !ok {
				continue
			}
		}
		out = append(out, region)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DescribeAvailabilityZones(zoneNames []string) []AvailabilityZone {
	nameSet := toStringSet(zoneNames)
	out := make([]AvailabilityZone, 0, len(s.availabilityZones))
	for _, zone := range s.availabilityZones {
		if len(nameSet) > 0 {
			if _, ok := nameSet[zone.Name]; !ok {
				continue
			}
		}
		out = append(out, zone)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreateSecurityGroup(name, description, vpcID string, tags []Tag) (SecurityGroup, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	vpcID = strings.TrimSpace(vpcID)
	if name == "" || description == "" {
		return SecurityGroup{}, ErrInvalidParameter
	}
	if vpcID == "" {
		vpcID = defaultVPCID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.vpcs[vpcID] == nil {
		return SecurityGroup{}, ErrNotFound
	}

	key := securityGroupNameKey(vpcID, name)
	if _, exists := s.securityGroupNameIndex[key]; exists {
		return SecurityGroup{}, ErrAlreadyExists
	}

	group := &SecurityGroup{
		ID:          s.nextIDLocked("sg"),
		Name:        name,
		Description: description,
		VpcID:       vpcID,
		Ingress:     []IPPermission{},
		Egress:      []IPPermission{},
		Tags:        tagsToMap(tags),
	}
	s.securityGroups[group.ID] = group
	s.securityGroupNameIndex[key] = group.ID
	return cloneSecurityGroup(group), nil
}

func (s *Service) DescribeSecurityGroups(groupIDs, groupNames []string, vpcID string) []SecurityGroup {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(groupIDs)
	nameSet := toStringSet(groupNames)
	vpcID = strings.TrimSpace(vpcID)

	out := make([]SecurityGroup, 0, len(s.securityGroups))
	for _, group := range s.securityGroups {
		if len(idSet) > 0 {
			if _, ok := idSet[group.ID]; !ok {
				continue
			}
		}
		if len(nameSet) > 0 {
			if _, ok := nameSet[group.Name]; !ok {
				continue
			}
		}
		if vpcID != "" && group.VpcID != vpcID {
			continue
		}
		out = append(out, cloneSecurityGroup(group))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteSecurityGroup(groupID, groupName, vpcID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return ErrNotFound
	}
	if group.ID == "sg-00000000" {
		return ErrConflict
	}
	for _, inst := range s.instances {
		if inst.StateName == "terminated" {
			continue
		}
		for _, sgID := range inst.SecurityGroupIDs {
			if sgID == group.ID {
				return ErrConflict
			}
		}
	}
	for _, iface := range s.networkInterfaces {
		for _, sgID := range iface.GroupIDs {
			if sgID == group.ID {
				return ErrConflict
			}
		}
	}
	delete(s.securityGroups, group.ID)
	delete(s.securityGroupNameIndex, securityGroupNameKey(group.VpcID, group.Name))
	return nil
}

func (s *Service) AuthorizeSecurityGroupIngress(groupID, groupName, vpcID string, perms []IPPermission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return ErrNotFound
	}
	for _, perm := range perms {
		perm = normalizePermission(perm)
		if perm.Protocol == "" {
			return ErrInvalidParameter
		}
		group.Ingress = upsertPermission(group.Ingress, perm)
	}
	return nil
}

func (s *Service) RevokeSecurityGroupIngress(groupID, groupName, vpcID string, perms []IPPermission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return ErrNotFound
	}
	if len(perms) == 0 {
		group.Ingress = nil
		return nil
	}
	for _, perm := range perms {
		perm = normalizePermission(perm)
		group.Ingress = deletePermission(group.Ingress, perm)
	}
	return nil
}

func (s *Service) AuthorizeSecurityGroupEgress(groupID, groupName, vpcID string, perms []IPPermission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return ErrNotFound
	}
	for _, perm := range perms {
		perm = normalizePermission(perm)
		if perm.Protocol == "" {
			return ErrInvalidParameter
		}
		group.Egress = upsertPermission(group.Egress, perm)
	}
	return nil
}

func (s *Service) RevokeSecurityGroupEgress(groupID, groupName, vpcID string, perms []IPPermission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return ErrNotFound
	}
	if len(perms) == 0 {
		group.Egress = nil
		return nil
	}
	for _, perm := range perms {
		perm = normalizePermission(perm)
		group.Egress = deletePermission(group.Egress, perm)
	}
	return nil
}

func (s *Service) CreateVolume(size int32, availabilityZone, volumeType, snapshotID string, tags []Tag) (Volume, error) {
	if size <= 0 {
		size = 8
	}
	availabilityZone = strings.TrimSpace(availabilityZone)
	if availabilityZone == "" {
		availabilityZone = "us-east-1a"
	}
	volumeType = strings.TrimSpace(volumeType)
	if volumeType == "" {
		volumeType = "gp3"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if snapshotID != "" && s.snapshots[strings.TrimSpace(snapshotID)] == nil {
		return Volume{}, ErrNotFound
	}
	volume := &Volume{
		ID:               s.nextIDLocked("vol"),
		AvailabilityZone: availabilityZone,
		SizeGiB:          size,
		SnapshotID:       strings.TrimSpace(snapshotID),
		State:            "available",
		VolumeType:       volumeType,
		AutoEnableIO:     true,
		CreateTime:       time.Now().UTC(),
		Attachments:      []VolumeAttachment{},
		Tags:             tagsToMap(tags),
	}
	switch strings.ToLower(volumeType) {
	case "gp3":
		volume.Iops = 3000
		volume.Throughput = 125
	case "io1", "io2":
		volume.Iops = 100
	}
	s.volumes[volume.ID] = volume
	return cloneVolume(volume), nil
}

func (s *Service) DescribeVolumes(volumeIDs []string) []Volume {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(volumeIDs)
	out := make([]Volume, 0, len(s.volumes))
	for _, volume := range s.volumes {
		if len(idSet) > 0 {
			if _, ok := idSet[volume.ID]; !ok {
				continue
			}
		}
		out = append(out, cloneVolume(volume))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AttachVolume(volumeID, instanceID, device string) (VolumeAttachment, error) {
	volumeID = strings.TrimSpace(volumeID)
	instanceID = strings.TrimSpace(instanceID)
	device = strings.TrimSpace(device)
	if volumeID == "" || instanceID == "" || device == "" {
		return VolumeAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	volume := s.volumes[volumeID]
	if volume == nil {
		return VolumeAttachment{}, ErrNotFound
	}
	if s.instances[instanceID] == nil {
		return VolumeAttachment{}, ErrNotFound
	}
	if len(volume.Attachments) > 0 {
		return VolumeAttachment{}, ErrConflict
	}
	attachment := VolumeAttachment{
		VolumeID:   volumeID,
		InstanceID: instanceID,
		Device:     device,
		State:      "attached",
		AttachTime: time.Now().UTC(),
	}
	volume.Attachments = []VolumeAttachment{attachment}
	volume.State = "in-use"
	return attachment, nil
}

func (s *Service) DetachVolume(volumeID, instanceID, device string, force bool) (VolumeAttachment, error) {
	_ = force
	volumeID = strings.TrimSpace(volumeID)
	instanceID = strings.TrimSpace(instanceID)
	device = strings.TrimSpace(device)
	if volumeID == "" {
		return VolumeAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	volume := s.volumes[volumeID]
	if volume == nil {
		return VolumeAttachment{}, ErrNotFound
	}
	if len(volume.Attachments) == 0 {
		return VolumeAttachment{}, ErrConflict
	}
	attachment := volume.Attachments[0]
	if instanceID != "" && attachment.InstanceID != instanceID {
		return VolumeAttachment{}, ErrConflict
	}
	if device != "" && attachment.Device != device {
		return VolumeAttachment{}, ErrConflict
	}
	attachment.State = "detached"
	volume.Attachments = nil
	volume.State = "available"
	return attachment, nil
}

func (s *Service) DeleteVolume(volumeID string) error {
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	volume := s.volumes[volumeID]
	if volume == nil {
		return ErrNotFound
	}
	if len(volume.Attachments) > 0 {
		return ErrConflict
	}
	delete(s.volumes, volumeID)
	return nil
}

func (s *Service) CreateSnapshot(volumeID, description string, tags []Tag) (Snapshot, error) {
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" {
		return Snapshot{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	volume := s.volumes[volumeID]
	if volume == nil {
		return Snapshot{}, ErrNotFound
	}
	snapshot := &Snapshot{
		ID:          s.nextIDLocked("snap"),
		VolumeID:    volumeID,
		State:       "completed",
		StartTime:   time.Now().UTC(),
		Progress:    "100%",
		Description: strings.TrimSpace(description),
		VolumeSize:  volume.SizeGiB,
		Tags:        tagsToMap(tags),
	}
	s.snapshots[snapshot.ID] = snapshot
	return cloneSnapshot(snapshot), nil
}

func (s *Service) DescribeSnapshots(snapshotIDs []string) []Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	idSet := toStringSet(snapshotIDs)
	out := make([]Snapshot, 0, len(s.snapshots))
	for _, snapshot := range s.snapshots {
		if len(idSet) > 0 {
			if _, ok := idSet[snapshot.ID]; !ok {
				continue
			}
		}
		out = append(out, cloneSnapshot(snapshot))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteSnapshot(snapshotID string) error {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshots[snapshotID] == nil {
		return ErrNotFound
	}
	delete(s.snapshots, snapshotID)
	return nil
}

func (s *Service) transitionInstances(instanceIDs []string, stateCode int32, stateName string) ([]InstanceStateChange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(instanceIDs) == 0 {
		return nil, ErrInvalidParameter
	}
	changes := make([]InstanceStateChange, 0, len(instanceIDs))
	for _, id := range instanceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		inst := s.instances[id]
		if inst == nil {
			return nil, ErrNotFound
		}
		change := InstanceStateChange{
			InstanceID:   id,
			PreviousCode: inst.StateCode,
			PreviousName: inst.StateName,
			CurrentCode:  stateCode,
			CurrentName:  stateName,
		}
		inst.StateCode = stateCode
		inst.StateName = stateName
		if stateName == "stopped" || stateName == "terminated" {
			delete(s.classicLinkAttachments, id)
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func (s *Service) applyTagsLocked(resourceID string, tags []Tag) error {
	if inst := s.instances[resourceID]; inst != nil {
		applyTagsToMap(inst.Tags, tags)
		return nil
	}
	if group := s.securityGroups[resourceID]; group != nil {
		applyTagsToMap(group.Tags, tags)
		return nil
	}
	if volume := s.volumes[resourceID]; volume != nil {
		applyTagsToMap(volume.Tags, tags)
		return nil
	}
	if image := s.images[resourceID]; image != nil {
		applyTagsToMap(image.Tags, tags)
		return nil
	}
	if address := s.addresses[resourceID]; address != nil {
		applyTagsToMap(address.Tags, tags)
		return nil
	}
	if options := s.dhcpOptions[resourceID]; options != nil {
		applyTagsToMap(options.Tags, tags)
		return nil
	}
	if gateway := s.egressOnlyGateways[resourceID]; gateway != nil {
		applyTagsToMap(gateway.Tags, tags)
		return nil
	}
	if gateway := s.natGateways[resourceID]; gateway != nil {
		applyTagsToMap(gateway.Tags, tags)
		return nil
	}
	if peering := s.vpcPeeringConnections[resourceID]; peering != nil {
		applyTagsToMap(peering.Tags, tags)
		return nil
	}
	if snapshot := s.snapshots[resourceID]; snapshot != nil {
		applyTagsToMap(snapshot.Tags, tags)
		return nil
	}
	if vpc := s.vpcs[resourceID]; vpc != nil {
		applyTagsToMap(vpc.Tags, tags)
		return nil
	}
	if subnet := s.subnets[resourceID]; subnet != nil {
		applyTagsToMap(subnet.Tags, tags)
		return nil
	}
	if gateway := s.internetGateways[resourceID]; gateway != nil {
		applyTagsToMap(gateway.Tags, tags)
		return nil
	}
	if table := s.routeTables[resourceID]; table != nil {
		applyTagsToMap(table.Tags, tags)
		return nil
	}
	if acl := s.networkAcls[resourceID]; acl != nil {
		applyTagsToMap(acl.Tags, tags)
		return nil
	}
	if iface := s.networkInterfaces[resourceID]; iface != nil {
		applyTagsToMap(iface.Tags, tags)
		return nil
	}
	return ErrNotFound
}

func (s *Service) removeTagsLocked(resourceID string, tags []Tag) error {
	remove := func(m map[string]string) {
		if len(tags) == 0 {
			for key := range m {
				delete(m, key)
			}
			return
		}
		for _, tag := range tags {
			key := strings.TrimSpace(tag.Key)
			if key == "" {
				continue
			}
			if tag.Value == "" {
				delete(m, key)
				continue
			}
			if m[key] == strings.TrimSpace(tag.Value) {
				delete(m, key)
			}
		}
	}
	if inst := s.instances[resourceID]; inst != nil {
		remove(inst.Tags)
		return nil
	}
	if group := s.securityGroups[resourceID]; group != nil {
		remove(group.Tags)
		return nil
	}
	if volume := s.volumes[resourceID]; volume != nil {
		remove(volume.Tags)
		return nil
	}
	if image := s.images[resourceID]; image != nil {
		remove(image.Tags)
		return nil
	}
	if address := s.addresses[resourceID]; address != nil {
		remove(address.Tags)
		return nil
	}
	if options := s.dhcpOptions[resourceID]; options != nil {
		remove(options.Tags)
		return nil
	}
	if gateway := s.egressOnlyGateways[resourceID]; gateway != nil {
		remove(gateway.Tags)
		return nil
	}
	if gateway := s.natGateways[resourceID]; gateway != nil {
		remove(gateway.Tags)
		return nil
	}
	if peering := s.vpcPeeringConnections[resourceID]; peering != nil {
		remove(peering.Tags)
		return nil
	}
	if snapshot := s.snapshots[resourceID]; snapshot != nil {
		remove(snapshot.Tags)
		return nil
	}
	if vpc := s.vpcs[resourceID]; vpc != nil {
		remove(vpc.Tags)
		return nil
	}
	if subnet := s.subnets[resourceID]; subnet != nil {
		remove(subnet.Tags)
		return nil
	}
	if gateway := s.internetGateways[resourceID]; gateway != nil {
		remove(gateway.Tags)
		return nil
	}
	if table := s.routeTables[resourceID]; table != nil {
		remove(table.Tags)
		return nil
	}
	if acl := s.networkAcls[resourceID]; acl != nil {
		remove(acl.Tags)
		return nil
	}
	if iface := s.networkInterfaces[resourceID]; iface != nil {
		remove(iface.Tags)
		return nil
	}
	return ErrNotFound
}

func (s *Service) resolveSecurityGroupLocked(groupID, groupName, vpcID string) *SecurityGroup {
	groupID = strings.TrimSpace(groupID)
	groupName = strings.TrimSpace(groupName)
	vpcID = strings.TrimSpace(vpcID)
	if groupID != "" {
		return s.securityGroups[groupID]
	}
	if groupName != "" {
		if vpcID == "" {
			vpcID = defaultVPCID
		}
		if id := s.securityGroupNameIndex[securityGroupNameKey(vpcID, groupName)]; id != "" {
			return s.securityGroups[id]
		}
	}
	return nil
}

func (s *Service) nextIDLocked(prefix string) string {
	seq := atomic.AddUint64(&s.seq, 1)
	switch prefix {
	case "i", "r", "h":
		return fmt.Sprintf("%s-%08x", prefix, seq)
	case "sg", "vol", "snap", "vpc", "subnet", "igw", "rtb", "rtbassoc", "acl", "aclassoc", "eni", "eniattach", "eni-perm", "key", "iip-assoc", "eipalloc", "eipassoc", "dopt", "eigw", "nat", "pcx", "ami", "cgw", "vgw", "vpn", "cvpn-endpoint", "cvpn-assoc", "cvpn-conn", "vai", "vag", "vae", "vatp":
		return fmt.Sprintf("%s-%08x", prefix, seq)
	default:
		return prefix + "-" + strconv.FormatUint(seq, 16)
	}
}

func tagsToMap(tags []Tag) map[string]string {
	out := map[string]string{}
	applyTagsToMap(out, tags)
	return out
}

func applyTagsToMap(out map[string]string, tags []Tag) {
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(tag.Value)
	}
}

func securityGroupNameKey(vpcID, name string) string {
	return strings.TrimSpace(vpcID) + "|" + strings.TrimSpace(name)
}

func normalizePermission(perm IPPermission) IPPermission {
	perm.Protocol = strings.TrimSpace(perm.Protocol)
	perm.CidrIP = strings.TrimSpace(perm.CidrIP)
	if perm.CidrIP == "" {
		perm.CidrIP = "0.0.0.0/0"
	}
	perm.Description = strings.TrimSpace(perm.Description)
	return perm
}

func upsertPermission(existing []IPPermission, perm IPPermission) []IPPermission {
	for i := range existing {
		if samePermissionIdentity(existing[i], perm) {
			return existing
		}
	}
	return append(existing, perm)
}

func deletePermission(existing []IPPermission, perm IPPermission) []IPPermission {
	out := make([]IPPermission, 0, len(existing))
	for _, current := range existing {
		if samePermissionIdentity(current, perm) {
			continue
		}
		out = append(out, current)
	}
	return out
}

func samePermissionIdentity(a, b IPPermission) bool {
	a = normalizePermission(a)
	b = normalizePermission(b)
	return a.Protocol == b.Protocol &&
		a.FromPort == b.FromPort &&
		a.ToPort == b.ToPort &&
		a.CidrIP == b.CidrIP
}

func toStringSet(in []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func cloneInstance(in *Instance) Instance {
	out := *in
	out.SecurityGroupIDs = append([]string(nil), in.SecurityGroupIDs...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneReservation(in *Reservation) Reservation {
	out := *in
	out.GroupIDs = append([]string(nil), in.GroupIDs...)
	out.InstanceIDs = append([]string(nil), in.InstanceIDs...)
	return out
}

func cloneSecurityGroup(in *SecurityGroup) SecurityGroup {
	out := *in
	out.Ingress = append([]IPPermission(nil), in.Ingress...)
	out.Egress = append([]IPPermission(nil), in.Egress...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneVolume(in *Volume) Volume {
	out := *in
	out.Attachments = append([]VolumeAttachment(nil), in.Attachments...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneSnapshot(in *Snapshot) Snapshot {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
