package ec2

import (
	"fmt"
	"strings"
	"time"
)

type CoipPool struct {
	LocalGatewayRouteTableID string
	PoolARN                  string
	PoolCIDRs                []string
	PoolID                   string
	Tags                     map[string]string
}

type MacModificationTask struct {
	InstanceID            string
	MacModificationTaskID string
	StartTime             time.Time
	Tags                  map[string]string
	TaskState             string
	TaskType              string
}

type Fleet struct {
	FleetID   string
	Instances []FleetInstance
	Tags      map[string]string
}

type FleetInstance struct {
	InstanceIDs  []string
	InstanceType string
	Lifecycle    string
}

type FlowLog struct {
	ClientToken  string
	FlowLogID    string
	ResourceID   string
	ResourceType string
	Tags         map[string]string
	TrafficType  string
}

type FlowLogsCreation struct {
	ClientToken string
	FlowLogIDs  []string
}

type FpgaImage struct {
	Description       string
	FpgaImageGlobalID string
	FpgaImageID       string
	InputBucket       string
	InputKey          string
	Name              string
	Tags              map[string]string
}

type InstanceConnectEndpoint struct {
	AvailabilityZone           string
	ClientToken                string
	CreatedAt                  time.Time
	DNSName                    string
	FipsDNSName                string
	InstanceConnectEndpointARN string
	InstanceConnectEndpointID  string
	IPAddressType              string
	OwnerID                    string
	PreserveClientIP           bool
	SecurityGroupIDs           []string
	State                      string
	StateMessage               string
	SubnetID                   string
	Tags                       map[string]string
	VpcID                      string
}

type InstanceEventWindow struct {
	CronExpression        string
	InstanceEventWindowID string
	Name                  string
	State                 string
	Tags                  map[string]string
	TimeRanges            []InstanceEventWindowTimeRange
}

type InstanceEventWindowTimeRange struct {
	EndHour      *int32
	EndWeekDay   string
	StartHour    *int32
	StartWeekDay string
}

type InstanceExportTask struct {
	Description       string
	ExportTaskID      string
	InstanceID        string
	S3Bucket          string
	S3Key             string
	S3Prefix          string
	State             string
	StatusMessage     string
	Tags              map[string]string
	TargetEnvironment string
	ContainerFormat   string
	DiskImageFormat   string
}

type Ipam struct {
	Description      string
	EnablePrivateGua bool
	IpamARN          string
	IpamID           string
	IpamRegion       string
	MeteredAccount   string
	OperatingRegions []string
	OwnerID          string
	State            string
	Tags             map[string]string
	Tier             string
}

type IpamExternalResourceVerificationToken struct {
	IpamARN                                  string
	IpamExternalResourceVerificationTokenARN string
	IpamExternalResourceVerificationTokenID  string
	IpamID                                   string
	IpamRegion                               string
	NotAfter                                 time.Time
	State                                    string
	Status                                   string
	Tags                                     map[string]string
	TokenName                                string
}

func (s *Service) CreateCoipPool(localGatewayRouteTableID string, tags []Tag) (CoipPool, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	if localGatewayRouteTableID == "" {
		return CoipPool{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	poolID := s.nextIDLocked("coip-pool")
	pool := &CoipPool{
		LocalGatewayRouteTableID: localGatewayRouteTableID,
		PoolARN:                  fmt.Sprintf("arn:aws:ec2:%s:%s:coip-pool/%s", DefaultRegion, DefaultAccountID, poolID),
		PoolCIDRs:                []string{},
		PoolID:                   poolID,
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
	}
	s.coipPools[poolID] = pool
	return cloneStage107CoipPool(pool), nil
}

func (s *Service) CreateDelegateMacVolumeOwnershipTask(
	instanceID string,
	macCredentials string,
	clientToken *string,
	tags []Tag,
) (MacModificationTask, error) {
	instanceID = strings.TrimSpace(instanceID)
	macCredentials = strings.TrimSpace(macCredentials)
	if instanceID == "" || macCredentials == "" {
		return MacModificationTask{}, ErrInvalidParameter
	}

	_ = clientToken

	s.mu.Lock()
	defer s.mu.Unlock()

	taskID := s.nextIDLocked("mmt")
	task := &MacModificationTask{
		InstanceID:            instanceID,
		MacModificationTaskID: taskID,
		StartTime:             time.Now().UTC(),
		Tags:                  tagsToMap(normalizeEC2Tags(tags)),
		TaskState:             "completed",
		TaskType:              "restore-volume-permissions",
	}
	s.macModificationTasks[taskID] = task
	return cloneStage107MacModificationTask(task), nil
}

func (s *Service) CreateFleet(
	hasLaunchTemplateConfigs bool,
	totalTargetCapacity int32,
	instanceType string,
	tags []Tag,
) (Fleet, error) {
	if !hasLaunchTemplateConfigs || totalTargetCapacity <= 0 {
		return Fleet{}, ErrInvalidParameter
	}
	instanceType = strings.TrimSpace(instanceType)
	if instanceType == "" {
		instanceType = "m5.large"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fleetID := s.nextIDLocked("fleet")
	instanceIDs := make([]string, 0, totalTargetCapacity)
	for i := int32(0); i < totalTargetCapacity; i++ {
		instanceIDs = append(instanceIDs, s.nextIDLocked("i"))
	}
	fleet := &Fleet{
		FleetID: fleetID,
		Instances: []FleetInstance{
			{
				InstanceIDs:  instanceIDs,
				InstanceType: instanceType,
				Lifecycle:    "on-demand",
			},
		},
		Tags: tagsToMap(normalizeEC2Tags(tags)),
	}
	s.fleets[fleetID] = fleet
	return cloneStage107Fleet(fleet), nil
}

func (s *Service) CreateFlowLogs(
	resourceIDs []string,
	resourceType string,
	trafficType string,
	clientToken *string,
	tags []Tag,
) (FlowLogsCreation, error) {
	resourceIDs = dedupeTrimmedStrings(resourceIDs)
	resourceType = strings.TrimSpace(resourceType)
	trafficType = strings.TrimSpace(trafficType)
	if len(resourceIDs) == 0 || resourceType == "" {
		return FlowLogsCreation{}, ErrInvalidParameter
	}
	if trafficType == "" {
		trafficType = "ALL"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("flt")
	}
	flowLogIDs := make([]string, 0, len(resourceIDs))
	tagMap := tagsToMap(normalizeEC2Tags(tags))
	for _, resourceID := range resourceIDs {
		flowLogID := s.nextIDLocked("fl")
		s.flowLogs[flowLogID] = &FlowLog{
			ClientToken:  token,
			FlowLogID:    flowLogID,
			ResourceID:   resourceID,
			ResourceType: resourceType,
			Tags:         cloneStringMap(tagMap),
			TrafficType:  trafficType,
		}
		flowLogIDs = append(flowLogIDs, flowLogID)
	}
	return FlowLogsCreation{
		ClientToken: token,
		FlowLogIDs:  flowLogIDs,
	}, nil
}

func (s *Service) CreateFpgaImage(
	inputBucket string,
	inputKey string,
	name *string,
	description *string,
	tags []Tag,
) (FpgaImage, error) {
	inputBucket = strings.TrimSpace(inputBucket)
	inputKey = strings.TrimSpace(inputKey)
	if inputBucket == "" || inputKey == "" {
		return FpgaImage{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fpgaImageID := s.nextIDLocked("afi")
	fpgaImageGlobalID := s.nextIDLocked("agfi")
	image := &FpgaImage{
		Description:       strings.TrimSpace(derefString(description)),
		FpgaImageGlobalID: fpgaImageGlobalID,
		FpgaImageID:       fpgaImageID,
		InputBucket:       inputBucket,
		InputKey:          inputKey,
		Name:              strings.TrimSpace(derefString(name)),
		Tags:              tagsToMap(normalizeEC2Tags(tags)),
	}
	s.fpgaImages[fpgaImageID] = image
	return cloneStage107FpgaImage(image), nil
}

func (s *Service) CreateInstanceConnectEndpoint(
	subnetID string,
	securityGroupIDs []string,
	ipAddressType string,
	preserveClientIP *bool,
	clientToken *string,
	tags []Tag,
) (InstanceConnectEndpoint, error) {
	subnetID = strings.TrimSpace(subnetID)
	if subnetID == "" {
		return InstanceConnectEndpoint{}, ErrInvalidParameter
	}
	securityGroupIDs = dedupeTrimmedStrings(securityGroupIDs)
	ipAddressType = strings.TrimSpace(ipAddressType)
	if ipAddressType == "" {
		ipAddressType = "ipv4"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subnet := s.subnets[subnetID]
	if subnet == nil {
		return InstanceConnectEndpoint{}, ErrNotFound
	}
	if len(securityGroupIDs) == 0 {
		securityGroupIDs = []string{"sg-00000000"}
	}

	endpointID := s.nextIDLocked("eice")
	token := strings.TrimSpace(derefString(clientToken))
	if token == "" {
		token = s.nextIDLocked("eice-token")
	}
	preserve := false
	if preserveClientIP != nil {
		preserve = *preserveClientIP
	}
	dnsName := fmt.Sprintf("%s.ec2-instance-connect.%s.amazonaws.com", endpointID, DefaultRegion)
	endpoint := &InstanceConnectEndpoint{
		AvailabilityZone:           subnet.AvailabilityZone,
		ClientToken:                token,
		CreatedAt:                  time.Now().UTC(),
		DNSName:                    dnsName,
		FipsDNSName:                "fips." + dnsName,
		InstanceConnectEndpointARN: fmt.Sprintf("arn:aws:ec2:%s:%s:instance-connect-endpoint/%s", DefaultRegion, DefaultAccountID, endpointID),
		InstanceConnectEndpointID:  endpointID,
		IPAddressType:              ipAddressType,
		OwnerID:                    DefaultAccountID,
		PreserveClientIP:           preserve,
		SecurityGroupIDs:           append([]string(nil), securityGroupIDs...),
		State:                      "create-complete",
		StateMessage:               "created",
		SubnetID:                   subnet.ID,
		Tags:                       tagsToMap(normalizeEC2Tags(tags)),
		VpcID:                      subnet.VpcID,
	}
	s.instanceConnectEndpoints[endpointID] = endpoint
	return cloneStage107InstanceConnectEndpoint(endpoint), nil
}

func (s *Service) CreateInstanceEventWindow(
	name string,
	cronExpression string,
	timeRanges []InstanceEventWindowTimeRange,
	tags []Tag,
) (InstanceEventWindow, error) {
	name = strings.TrimSpace(name)
	cronExpression = strings.TrimSpace(cronExpression)
	if cronExpression == "" {
		cronExpression = "cron(0 0 ? * SUN *)"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	windowID := s.nextIDLocked("iew")
	if name == "" {
		name = "event-window-" + strings.TrimPrefix(windowID, "iew-")
	}
	window := &InstanceEventWindow{
		CronExpression:        cronExpression,
		InstanceEventWindowID: windowID,
		Name:                  name,
		State:                 "active",
		Tags:                  tagsToMap(normalizeEC2Tags(tags)),
		TimeRanges:            cloneStage107EventWindowTimeRanges(timeRanges),
	}
	s.instanceEventWindows[windowID] = window
	return cloneStage107InstanceEventWindow(window), nil
}

func (s *Service) CreateInstanceExportTask(
	description *string,
	instanceID string,
	targetEnvironment string,
	s3Bucket string,
	s3Prefix string,
	containerFormat string,
	diskImageFormat string,
	tags []Tag,
) (InstanceExportTask, error) {
	instanceID = strings.TrimSpace(instanceID)
	targetEnvironment = strings.TrimSpace(targetEnvironment)
	s3Bucket = strings.TrimSpace(s3Bucket)
	s3Prefix = strings.TrimSpace(s3Prefix)
	containerFormat = strings.TrimSpace(containerFormat)
	diskImageFormat = strings.TrimSpace(diskImageFormat)
	if instanceID == "" || targetEnvironment == "" || s3Bucket == "" {
		return InstanceExportTask{}, ErrInvalidParameter
	}
	if containerFormat == "" {
		containerFormat = "ova"
	}
	if diskImageFormat == "" {
		diskImageFormat = "vmdk"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exportTaskID := "export-" + s.nextIDLocked("i")
	s3Key := strings.TrimPrefix(s3Prefix+"/"+exportTaskID+"."+diskImageFormat, "/")
	task := &InstanceExportTask{
		Description:       strings.TrimSpace(derefString(description)),
		ExportTaskID:      exportTaskID,
		InstanceID:        instanceID,
		S3Bucket:          s3Bucket,
		S3Key:             s3Key,
		S3Prefix:          s3Prefix,
		State:             "active",
		StatusMessage:     "export task in progress",
		Tags:              tagsToMap(normalizeEC2Tags(tags)),
		TargetEnvironment: targetEnvironment,
		ContainerFormat:   containerFormat,
		DiskImageFormat:   diskImageFormat,
	}
	s.instanceExportTasks[exportTaskID] = task
	return cloneStage107InstanceExportTask(task), nil
}

func (s *Service) CreateIpam(
	description *string,
	enablePrivateGua *bool,
	meteredAccount string,
	operatingRegions []string,
	tier string,
	tags []Tag,
) (Ipam, error) {
	meteredAccount = strings.TrimSpace(meteredAccount)
	tier = strings.TrimSpace(tier)
	if meteredAccount == "" {
		meteredAccount = "disabled"
	}
	if tier == "" {
		tier = "free"
	}
	operatingRegions = dedupeTrimmedStrings(operatingRegions)
	if len(operatingRegions) == 0 {
		operatingRegions = []string{DefaultRegion}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ipamID := s.nextIDLocked("ipam")
	enablePrivate := false
	if enablePrivateGua != nil {
		enablePrivate = *enablePrivateGua
	}
	ipam := &Ipam{
		Description:      strings.TrimSpace(derefString(description)),
		EnablePrivateGua: enablePrivate,
		IpamARN:          fmt.Sprintf("arn:aws:ec2:%s:%s:ipam/%s", DefaultRegion, DefaultAccountID, ipamID),
		IpamID:           ipamID,
		IpamRegion:       DefaultRegion,
		MeteredAccount:   meteredAccount,
		OperatingRegions: append([]string(nil), operatingRegions...),
		OwnerID:          DefaultAccountID,
		State:            "create-complete",
		Tags:             tagsToMap(normalizeEC2Tags(tags)),
		Tier:             tier,
	}
	s.ipams[ipamID] = ipam
	return cloneStage107Ipam(ipam), nil
}

func (s *Service) CreateIpamExternalResourceVerificationToken(
	ipamID string,
	tokenName *string,
	tags []Tag,
) (IpamExternalResourceVerificationToken, error) {
	ipamID = strings.TrimSpace(ipamID)
	if ipamID == "" {
		return IpamExternalResourceVerificationToken{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ipam := s.ipams[ipamID]
	if ipam == nil {
		return IpamExternalResourceVerificationToken{}, ErrNotFound
	}

	tokenID := s.nextIDLocked("ipam-ervt")
	name := strings.TrimSpace(derefString(tokenName))
	if name == "" {
		name = "token-" + strings.TrimPrefix(tokenID, "ipam-ervt-")
	}
	token := &IpamExternalResourceVerificationToken{
		IpamARN:                                  ipam.IpamARN,
		IpamExternalResourceVerificationTokenARN: fmt.Sprintf("arn:aws:ec2:%s:%s:ipam-external-resource-verification-token/%s", DefaultRegion, DefaultAccountID, tokenID),
		IpamExternalResourceVerificationTokenID:  tokenID,
		IpamID:                                   ipamID,
		IpamRegion:                               ipam.IpamRegion,
		NotAfter:                                 time.Now().UTC().Add(24 * time.Hour),
		State:                                    "active",
		Status:                                   "pending",
		Tags:                                     tagsToMap(normalizeEC2Tags(tags)),
		TokenName:                                name,
	}
	s.ipamExternalResourceVerificationTokens[tokenID] = token
	return cloneStage107IpamExternalResourceVerificationToken(token), nil
}

func cloneStage107CoipPool(in *CoipPool) CoipPool {
	if in == nil {
		return CoipPool{}
	}
	return CoipPool{
		LocalGatewayRouteTableID: in.LocalGatewayRouteTableID,
		PoolARN:                  in.PoolARN,
		PoolCIDRs:                append([]string(nil), in.PoolCIDRs...),
		PoolID:                   in.PoolID,
		Tags:                     cloneStringMap(in.Tags),
	}
}

func cloneStage107MacModificationTask(in *MacModificationTask) MacModificationTask {
	if in == nil {
		return MacModificationTask{}
	}
	return MacModificationTask{
		InstanceID:            in.InstanceID,
		MacModificationTaskID: in.MacModificationTaskID,
		StartTime:             in.StartTime,
		Tags:                  cloneStringMap(in.Tags),
		TaskState:             in.TaskState,
		TaskType:              in.TaskType,
	}
}

func cloneStage107Fleet(in *Fleet) Fleet {
	if in == nil {
		return Fleet{}
	}
	out := Fleet{
		FleetID: in.FleetID,
		Tags:    cloneStringMap(in.Tags),
	}
	out.Instances = make([]FleetInstance, 0, len(in.Instances))
	for _, item := range in.Instances {
		out.Instances = append(out.Instances, FleetInstance{
			InstanceIDs:  append([]string(nil), item.InstanceIDs...),
			InstanceType: item.InstanceType,
			Lifecycle:    item.Lifecycle,
		})
	}
	return out
}

func cloneStage107FpgaImage(in *FpgaImage) FpgaImage {
	if in == nil {
		return FpgaImage{}
	}
	return FpgaImage{
		Description:       in.Description,
		FpgaImageGlobalID: in.FpgaImageGlobalID,
		FpgaImageID:       in.FpgaImageID,
		InputBucket:       in.InputBucket,
		InputKey:          in.InputKey,
		Name:              in.Name,
		Tags:              cloneStringMap(in.Tags),
	}
}

func cloneStage107InstanceConnectEndpoint(in *InstanceConnectEndpoint) InstanceConnectEndpoint {
	if in == nil {
		return InstanceConnectEndpoint{}
	}
	return InstanceConnectEndpoint{
		AvailabilityZone:           in.AvailabilityZone,
		ClientToken:                in.ClientToken,
		CreatedAt:                  in.CreatedAt,
		DNSName:                    in.DNSName,
		FipsDNSName:                in.FipsDNSName,
		InstanceConnectEndpointARN: in.InstanceConnectEndpointARN,
		InstanceConnectEndpointID:  in.InstanceConnectEndpointID,
		IPAddressType:              in.IPAddressType,
		OwnerID:                    in.OwnerID,
		PreserveClientIP:           in.PreserveClientIP,
		SecurityGroupIDs:           append([]string(nil), in.SecurityGroupIDs...),
		State:                      in.State,
		StateMessage:               in.StateMessage,
		SubnetID:                   in.SubnetID,
		Tags:                       cloneStringMap(in.Tags),
		VpcID:                      in.VpcID,
	}
}

func cloneStage107InstanceEventWindow(in *InstanceEventWindow) InstanceEventWindow {
	if in == nil {
		return InstanceEventWindow{}
	}
	return InstanceEventWindow{
		CronExpression:        in.CronExpression,
		InstanceEventWindowID: in.InstanceEventWindowID,
		Name:                  in.Name,
		State:                 in.State,
		Tags:                  cloneStringMap(in.Tags),
		TimeRanges:            cloneStage107EventWindowTimeRanges(in.TimeRanges),
	}
}

func cloneStage107EventWindowTimeRanges(in []InstanceEventWindowTimeRange) []InstanceEventWindowTimeRange {
	out := make([]InstanceEventWindowTimeRange, 0, len(in))
	for _, item := range in {
		out = append(out, InstanceEventWindowTimeRange{
			EndHour:      cloneInt32Pointer(item.EndHour),
			EndWeekDay:   item.EndWeekDay,
			StartHour:    cloneInt32Pointer(item.StartHour),
			StartWeekDay: item.StartWeekDay,
		})
	}
	return out
}

func cloneStage107InstanceExportTask(in *InstanceExportTask) InstanceExportTask {
	if in == nil {
		return InstanceExportTask{}
	}
	return InstanceExportTask{
		Description:       in.Description,
		ExportTaskID:      in.ExportTaskID,
		InstanceID:        in.InstanceID,
		S3Bucket:          in.S3Bucket,
		S3Key:             in.S3Key,
		S3Prefix:          in.S3Prefix,
		State:             in.State,
		StatusMessage:     in.StatusMessage,
		Tags:              cloneStringMap(in.Tags),
		TargetEnvironment: in.TargetEnvironment,
		ContainerFormat:   in.ContainerFormat,
		DiskImageFormat:   in.DiskImageFormat,
	}
}

func cloneStage107Ipam(in *Ipam) Ipam {
	if in == nil {
		return Ipam{}
	}
	return Ipam{
		Description:      in.Description,
		EnablePrivateGua: in.EnablePrivateGua,
		IpamARN:          in.IpamARN,
		IpamID:           in.IpamID,
		IpamRegion:       in.IpamRegion,
		MeteredAccount:   in.MeteredAccount,
		OperatingRegions: append([]string(nil), in.OperatingRegions...),
		OwnerID:          in.OwnerID,
		State:            in.State,
		Tags:             cloneStringMap(in.Tags),
		Tier:             in.Tier,
	}
}

func cloneStage107IpamExternalResourceVerificationToken(in *IpamExternalResourceVerificationToken) IpamExternalResourceVerificationToken {
	if in == nil {
		return IpamExternalResourceVerificationToken{}
	}
	return IpamExternalResourceVerificationToken{
		IpamARN:                                  in.IpamARN,
		IpamExternalResourceVerificationTokenARN: in.IpamExternalResourceVerificationTokenARN,
		IpamExternalResourceVerificationTokenID:  in.IpamExternalResourceVerificationTokenID,
		IpamID:                                   in.IpamID,
		IpamRegion:                               in.IpamRegion,
		NotAfter:                                 in.NotAfter,
		State:                                    in.State,
		Status:                                   in.Status,
		Tags:                                     cloneStringMap(in.Tags),
		TokenName:                                in.TokenName,
	}
}

func derefString(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}
