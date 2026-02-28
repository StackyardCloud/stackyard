package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type LocalGatewayVirtualInterfaceGroup struct {
	ConfigurationState                   string
	LocalBgpASN                          *int32
	LocalBgpASNExtended                  *int64
	LocalGatewayID                       string
	LocalGatewayVirtualInterfaceGroupARN string
	LocalGatewayVirtualInterfaceGroupID  string
	LocalGatewayVirtualInterfaceIDs      []string
	OwnerID                              string
	Tags                                 map[string]string
}

type ManagedPrefixList struct {
	AddressFamily  string
	MaxEntries     int32
	OwnerID        string
	PrefixListARN  string
	PrefixListID   string
	PrefixListName string
	State          string
	StateMessage   string
	Tags           map[string]string
	Version        int64
}

type ManagedPrefixListEntry struct {
	CIDR        string
	Description string
}

type NetworkInsightsAccessScope struct {
	CreatedDate                   time.Time
	NetworkInsightsAccessScopeARN string
	NetworkInsightsAccessScopeID  string
	Tags                          map[string]string
	UpdatedDate                   time.Time
}

type NetworkInsightsPath struct {
	CreatedDate            time.Time
	Destination            string
	DestinationIP          string
	DestinationPort        *int32
	NetworkInsightsPathARN string
	NetworkInsightsPathID  string
	Protocol               string
	Source                 string
	SourceIP               string
	Tags                   map[string]string
}

type PublicIpv4Pool struct {
	NetworkBorderGroup string
	PoolID             string
	Tags               map[string]string
}

type ReplaceRootVolumeTask struct {
	CompleteTime             string
	DeleteReplacedRootVolume *bool
	ImageID                  string
	InstanceID               string
	ReplaceRootVolumeTaskID  string
	SnapshotID               string
	StartTime                string
	Tags                     map[string]string
	TaskState                string
}

type ReservedInstancesPriceSchedule struct {
	CurrencyCode string
	Price        float64
	Term         int64
}

func (s *Service) CreateLocalGatewayVirtualInterfaceGroup(
	localGatewayID string,
	localBgpASN *int32,
	localBgpASNExtended *int64,
	tags []Tag,
) (LocalGatewayVirtualInterfaceGroup, error) {
	localGatewayID = strings.TrimSpace(localGatewayID)
	if localGatewayID == "" {
		return LocalGatewayVirtualInterfaceGroup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("lgw-vifgrp")
	group := &LocalGatewayVirtualInterfaceGroup{
		ConfigurationState:                   "available",
		LocalBgpASN:                          cloneInt32Pointer(localBgpASN),
		LocalBgpASNExtended:                  cloneInt64Pointer(localBgpASNExtended),
		LocalGatewayID:                       localGatewayID,
		LocalGatewayVirtualInterfaceGroupARN: fmt.Sprintf("arn:aws:ec2:%s:%s:local-gateway-virtual-interface-group/%s", DefaultRegion, DefaultAccountID, id),
		LocalGatewayVirtualInterfaceGroupID:  id,
		LocalGatewayVirtualInterfaceIDs:      []string{},
		OwnerID:                              DefaultAccountID,
		Tags:                                 tagsToMap(normalizeEC2Tags(tags)),
	}
	s.localGatewayVirtualInterfaceGroups[id] = group
	return cloneStage109LocalGatewayVirtualInterfaceGroup(group), nil
}

func (s *Service) CreateMacSystemIntegrityProtectionModificationTask(
	instanceID string,
	status string,
	clientToken *string,
	macCredentials *string,
	tags []Tag,
) (MacModificationTask, error) {
	instanceID = strings.TrimSpace(instanceID)
	status = strings.ToLower(strings.TrimSpace(status))
	if instanceID == "" || status == "" {
		return MacModificationTask{}, ErrInvalidParameter
	}
	if status != "enabled" && status != "disabled" {
		return MacModificationTask{}, ErrInvalidParameter
	}

	_ = clientToken
	_ = macCredentials

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return MacModificationTask{}, ErrNotFound
	}

	taskID := s.nextIDLocked("mmt")
	task := &MacModificationTask{
		InstanceID:            instanceID,
		MacModificationTaskID: taskID,
		StartTime:             time.Now().UTC(),
		Tags:                  tagsToMap(normalizeEC2Tags(tags)),
		TaskState:             "completed",
		TaskType:              "sip-modification",
	}
	s.macModificationTasks[taskID] = task
	return cloneStage107MacModificationTask(task), nil
}

func (s *Service) CreateManagedPrefixList(
	addressFamily string,
	maxEntries int32,
	prefixListName string,
	entries []ManagedPrefixListEntry,
	clientToken *string,
	tags []Tag,
) (ManagedPrefixList, error) {
	addressFamily = strings.ToLower(strings.TrimSpace(addressFamily))
	prefixListName = strings.TrimSpace(prefixListName)
	if addressFamily == "" || prefixListName == "" || maxEntries <= 0 {
		return ManagedPrefixList{}, ErrInvalidParameter
	}
	if addressFamily != "ipv4" && addressFamily != "ipv6" {
		return ManagedPrefixList{}, ErrInvalidParameter
	}
	if len(entries) > int(maxEntries) {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	_ = clientToken

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("pl")
	list := &ManagedPrefixList{
		AddressFamily:  addressFamily,
		MaxEntries:     maxEntries,
		OwnerID:        DefaultAccountID,
		PrefixListARN:  fmt.Sprintf("arn:aws:ec2:%s:%s:prefix-list/%s", DefaultRegion, DefaultAccountID, id),
		PrefixListID:   id,
		PrefixListName: prefixListName,
		State:          "create-complete",
		StateMessage:   "",
		Tags:           tagsToMap(normalizeEC2Tags(tags)),
		Version:        1,
	}
	s.managedPrefixLists[id] = list
	return cloneStage109ManagedPrefixList(list), nil
}

func (s *Service) CreateNetworkInsightsAccessScope(
	clientToken string,
	tags []Tag,
) (NetworkInsightsAccessScope, error) {
	clientToken = strings.TrimSpace(clientToken)
	if clientToken == "" {
		return NetworkInsightsAccessScope{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	id := s.nextIDLocked("nias")
	scope := &NetworkInsightsAccessScope{
		CreatedDate:                   now,
		NetworkInsightsAccessScopeARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-access-scope/%s", DefaultRegion, DefaultAccountID, id),
		NetworkInsightsAccessScopeID:  id,
		Tags:                          tagsToMap(normalizeEC2Tags(tags)),
		UpdatedDate:                   now,
	}
	s.networkInsightsAccessScopes[id] = scope
	return cloneStage109NetworkInsightsAccessScope(scope), nil
}

func (s *Service) CreateNetworkInsightsPath(
	clientToken string,
	protocol string,
	source string,
	destination *string,
	destinationIP *string,
	destinationPort *int32,
	sourceIP *string,
	tags []Tag,
) (NetworkInsightsPath, error) {
	clientToken = strings.TrimSpace(clientToken)
	protocol = strings.TrimSpace(protocol)
	source = strings.TrimSpace(source)
	if clientToken == "" || protocol == "" || source == "" {
		return NetworkInsightsPath{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("nip")
	now := time.Now().UTC()
	path := &NetworkInsightsPath{
		CreatedDate:            now,
		Destination:            strings.TrimSpace(derefString(destination)),
		DestinationIP:          strings.TrimSpace(derefString(destinationIP)),
		DestinationPort:        cloneInt32Pointer(destinationPort),
		NetworkInsightsPathARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-path/%s", DefaultRegion, DefaultAccountID, id),
		NetworkInsightsPathID:  id,
		Protocol:               strings.ToLower(protocol),
		Source:                 source,
		SourceIP:               strings.TrimSpace(derefString(sourceIP)),
		Tags:                   tagsToMap(normalizeEC2Tags(tags)),
	}
	s.networkInsightsPaths[id] = path
	return cloneStage109NetworkInsightsPath(path), nil
}

func (s *Service) CreatePublicIpv4Pool(networkBorderGroup *string, tags []Tag) (PublicIpv4Pool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("ipv4pool")
	pool := &PublicIpv4Pool{
		NetworkBorderGroup: strings.TrimSpace(derefString(networkBorderGroup)),
		PoolID:             id,
		Tags:               tagsToMap(normalizeEC2Tags(tags)),
	}
	s.publicIpv4Pools[id] = pool
	return cloneStage109PublicIpv4Pool(pool), nil
}

func (s *Service) CreateReplaceRootVolumeTask(
	instanceID string,
	deleteReplacedRootVolume *bool,
	imageID *string,
	snapshotID *string,
	clientToken *string,
	volumeInitializationRate *int64,
	tags []Tag,
) (ReplaceRootVolumeTask, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return ReplaceRootVolumeTask{}, ErrInvalidParameter
	}

	_ = clientToken
	_ = volumeInitializationRate

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return ReplaceRootVolumeTask{}, ErrNotFound
	}

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := s.nextIDLocked("replacevol")
	task := &ReplaceRootVolumeTask{
		CompleteTime:             now,
		DeleteReplacedRootVolume: deleteReplacedRootVolume,
		ImageID:                  strings.TrimSpace(derefString(imageID)),
		InstanceID:               instanceID,
		ReplaceRootVolumeTaskID:  taskID,
		SnapshotID:               strings.TrimSpace(derefString(snapshotID)),
		StartTime:                now,
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
		TaskState:                "succeeded",
	}
	s.replaceRootVolumeTasks[taskID] = task
	return cloneStage109ReplaceRootVolumeTask(task), nil
}

func (s *Service) CreateReservedInstancesListing(
	clientToken string,
	instanceCount int32,
	reservedInstancesID string,
	priceSchedules []ReservedInstancesPriceSchedule,
	tags []Tag,
) ([]ReservedInstancesListing, error) {
	clientToken = strings.TrimSpace(clientToken)
	reservedInstancesID = strings.TrimSpace(reservedInstancesID)
	if clientToken == "" || reservedInstancesID == "" || instanceCount <= 0 || len(priceSchedules) == 0 {
		return nil, ErrInvalidParameter
	}
	_ = tags

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	listingID := s.nextIDLocked("ril")
	s.reservedInstancesListingStates[listingID] = "active"
	s.reservedInstancesListingCreatedAt[listingID] = now
	listing := ReservedInstancesListing{
		ClientToken:                clientToken,
		CreateDate:                 now,
		ReservedInstancesID:        reservedInstancesID,
		ReservedInstancesListingID: listingID,
		Status:                     "active",
		StatusMessage:              "listing created",
		UpdateDate:                 now,
	}
	return []ReservedInstancesListing{listing}, nil
}

func (s *Service) CreateRestoreImageTask(
	bucket string,
	objectKey string,
	name *string,
	tags []Tag,
) (string, error) {
	bucket = strings.TrimSpace(bucket)
	objectKey = strings.TrimSpace(objectKey)
	if bucket == "" || objectKey == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	imageName := strings.TrimSpace(derefString(name))
	if imageName == "" {
		imageName = objectKey
	}

	id := s.nextIDLocked("ami")
	image := &Image{
		ID:                       id,
		Name:                     imageName,
		Description:              "",
		DeprecationTime:          nil,
		DeregistrationProtection: "disabled",
		State:                    "available",
		OwnerID:                  DefaultAccountID,
		ImageLocation:            DefaultAccountID + "/" + imageName,
		Architecture:             "x86_64",
		ImageType:                "machine",
		RootDeviceType:           "ebs",
		RootDeviceName:           "/dev/sda1",
		VirtualizationType:       "hvm",
		CreationDate:             time.Now().UTC(),
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
		LaunchPermissions:        nil,
	}
	s.images[id] = image
	return id, nil
}

func (s *Service) CreateSnapshots(
	instanceID string,
	description *string,
	copyTagsFromSource string,
	tags []Tag,
) ([]Snapshot, error) {
	instanceID = strings.TrimSpace(instanceID)
	copyTagsFromSource = strings.ToLower(strings.TrimSpace(copyTagsFromSource))
	if instanceID == "" {
		return nil, ErrInvalidParameter
	}
	if copyTagsFromSource != "" && copyTagsFromSource != "volume" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return nil, ErrNotFound
	}

	volumeIDs := make([]string, 0)
	for volumeID, volume := range s.volumes {
		for _, attachment := range volume.Attachments {
			if attachment.InstanceID == instanceID {
				volumeIDs = append(volumeIDs, volumeID)
				break
			}
		}
	}
	sort.Strings(volumeIDs)
	if len(volumeIDs) == 0 {
		return nil, ErrNotFound
	}

	snapshotDescription := strings.TrimSpace(derefString(description))
	requestTags := tagsToMap(normalizeEC2Tags(tags))
	out := make([]Snapshot, 0, len(volumeIDs))
	for _, volumeID := range volumeIDs {
		volume := s.volumes[volumeID]
		snapshotTags := cloneStringMap(requestTags)
		if copyTagsFromSource == "volume" {
			applyTagsToMap(snapshotTags, mapToTags(volume.Tags))
			applyTagsToMap(snapshotTags, mapToTags(requestTags))
		}

		snapshot := &Snapshot{
			ID:          s.nextIDLocked("snap"),
			VolumeID:    volumeID,
			State:       "completed",
			StartTime:   time.Now().UTC(),
			Progress:    "100%",
			Description: snapshotDescription,
			VolumeSize:  volume.SizeGiB,
			Tags:        snapshotTags,
		}
		s.snapshots[snapshot.ID] = snapshot
		out = append(out, cloneSnapshot(snapshot))
	}
	return out, nil
}

func cloneStage109LocalGatewayVirtualInterfaceGroup(in *LocalGatewayVirtualInterfaceGroup) LocalGatewayVirtualInterfaceGroup {
	if in == nil {
		return LocalGatewayVirtualInterfaceGroup{}
	}
	return LocalGatewayVirtualInterfaceGroup{
		ConfigurationState:                   in.ConfigurationState,
		LocalBgpASN:                          cloneInt32Pointer(in.LocalBgpASN),
		LocalBgpASNExtended:                  cloneInt64Pointer(in.LocalBgpASNExtended),
		LocalGatewayID:                       in.LocalGatewayID,
		LocalGatewayVirtualInterfaceGroupARN: in.LocalGatewayVirtualInterfaceGroupARN,
		LocalGatewayVirtualInterfaceGroupID:  in.LocalGatewayVirtualInterfaceGroupID,
		LocalGatewayVirtualInterfaceIDs:      append([]string(nil), in.LocalGatewayVirtualInterfaceIDs...),
		OwnerID:                              in.OwnerID,
		Tags:                                 cloneStringMap(in.Tags),
	}
}

func cloneStage109ManagedPrefixList(in *ManagedPrefixList) ManagedPrefixList {
	if in == nil {
		return ManagedPrefixList{}
	}
	return ManagedPrefixList{
		AddressFamily:  in.AddressFamily,
		MaxEntries:     in.MaxEntries,
		OwnerID:        in.OwnerID,
		PrefixListARN:  in.PrefixListARN,
		PrefixListID:   in.PrefixListID,
		PrefixListName: in.PrefixListName,
		State:          in.State,
		StateMessage:   in.StateMessage,
		Tags:           cloneStringMap(in.Tags),
		Version:        in.Version,
	}
}

func cloneStage109NetworkInsightsAccessScope(in *NetworkInsightsAccessScope) NetworkInsightsAccessScope {
	if in == nil {
		return NetworkInsightsAccessScope{}
	}
	return NetworkInsightsAccessScope{
		CreatedDate:                   in.CreatedDate,
		NetworkInsightsAccessScopeARN: in.NetworkInsightsAccessScopeARN,
		NetworkInsightsAccessScopeID:  in.NetworkInsightsAccessScopeID,
		Tags:                          cloneStringMap(in.Tags),
		UpdatedDate:                   in.UpdatedDate,
	}
}

func cloneStage109NetworkInsightsPath(in *NetworkInsightsPath) NetworkInsightsPath {
	if in == nil {
		return NetworkInsightsPath{}
	}
	return NetworkInsightsPath{
		CreatedDate:            in.CreatedDate,
		Destination:            in.Destination,
		DestinationIP:          in.DestinationIP,
		DestinationPort:        cloneInt32Pointer(in.DestinationPort),
		NetworkInsightsPathARN: in.NetworkInsightsPathARN,
		NetworkInsightsPathID:  in.NetworkInsightsPathID,
		Protocol:               in.Protocol,
		Source:                 in.Source,
		SourceIP:               in.SourceIP,
		Tags:                   cloneStringMap(in.Tags),
	}
}

func cloneStage109PublicIpv4Pool(in *PublicIpv4Pool) PublicIpv4Pool {
	if in == nil {
		return PublicIpv4Pool{}
	}
	return PublicIpv4Pool{
		NetworkBorderGroup: in.NetworkBorderGroup,
		PoolID:             in.PoolID,
		Tags:               cloneStringMap(in.Tags),
	}
}

func cloneStage109ReplaceRootVolumeTask(in *ReplaceRootVolumeTask) ReplaceRootVolumeTask {
	if in == nil {
		return ReplaceRootVolumeTask{}
	}
	return ReplaceRootVolumeTask{
		CompleteTime:             in.CompleteTime,
		DeleteReplacedRootVolume: cloneBoolPointer(in.DeleteReplacedRootVolume),
		ImageID:                  in.ImageID,
		InstanceID:               in.InstanceID,
		ReplaceRootVolumeTaskID:  in.ReplaceRootVolumeTaskID,
		SnapshotID:               in.SnapshotID,
		StartTime:                in.StartTime,
		Tags:                     cloneStringMap(in.Tags),
		TaskState:                in.TaskState,
	}
}

func mapToTags(in map[string]string) []Tag {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Tag, 0, len(keys))
	for _, key := range keys {
		out = append(out, Tag{Key: key, Value: in[key]})
	}
	return out
}
