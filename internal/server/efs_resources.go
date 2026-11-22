package server

import (
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"time"
)

type efsMountTarget struct {
	seq            int
	ID             string
	FileSystemID   string
	SubnetID       string
	VpcID          string
	AZ             string
	IP             string
	ENI            string
	SecurityGroups []string
}

type efsPosixUser struct {
	Uid           *int64  `json:"Uid"`
	Gid           *int64  `json:"Gid"`
	SecondaryGids []int64 `json:"SecondaryGids,omitempty"`
}

type efsCreationInfo struct {
	OwnerUid    *int64  `json:"OwnerUid"`
	OwnerGid    *int64  `json:"OwnerGid"`
	Permissions *string `json:"Permissions"`
}

type efsRootDirectory struct {
	Path         *string          `json:"Path,omitempty"`
	CreationInfo *efsCreationInfo `json:"CreationInfo,omitempty"`
}

type efsAccessPoint struct {
	seq          int
	ID           string
	ARN          string
	FileSystemID string
	ClientToken  string
	Tags         []efsTag
	PosixUser    *efsPosixUser
	Root         efsRootDirectory
}

type efsReplication struct {
	SourceID     string
	CreationTime time.Time
	DestID       string
	DestRegion   string
	RoleArn      string
	announced    bool
}

// efsSubnet is the slice of an EC2 subnet a mount target needs. The handler
// reads it from the EC2 store before calling in, so the two locks never nest.
type efsSubnet struct {
	ID, VpcID, CIDR, AZ string
	// SecurityGroups maps each requested (or default) group to its VPC; a
	// missing entry means the group doesn't exist.
	SecurityGroups map[string]string
	DefaultGroup   string
}

const efsMaxUnixID = 4294967295

// ---- tags ----

func efsValidateTags(tags []efsTag) ([]efsTag, *efsError) {
	if len(tags) > efsMaxTags {
		return nil, efsBadRequest("A resource can have at most %d tags.", efsMaxTags)
	}
	seen := map[string]bool{}
	out := make([]efsTag, 0, len(tags))
	for _, tag := range tags {
		if tag.Key == "" || len(tag.Key) > 128 || len(tag.Value) > 256 {
			return nil, efsBadRequest("Tag keys must be 1-128 characters and values at most 256.")
		}
		if strings.HasPrefix(strings.ToLower(tag.Key), "aws:") {
			return nil, efsBadRequest("Tag keys starting with 'aws:' are reserved.")
		}
		if seen[tag.Key] {
			return nil, efsBadRequest("Tag key %q is duplicated.", tag.Key)
		}
		seen[tag.Key] = true
		out = append(out, tag)
	}
	return out, nil
}

func efsMergeTags(existing, add []efsTag) ([]efsTag, *efsError) {
	merged := efsCopyTags(existing)
	for _, tag := range add {
		replaced := false
		for i := range merged {
			if merged[i].Key == tag.Key {
				merged[i].Value, replaced = tag.Value, true
			}
		}
		if !replaced {
			merged = append(merged, tag)
		}
	}
	if len(merged) > efsMaxTags {
		return nil, efsBadRequest("A resource can have at most %d tags.", efsMaxTags)
	}
	return merged, nil
}

func efsRemoveTags(existing []efsTag, keys []string) []efsTag {
	drop := efsSet(keys...)
	out := make([]efsTag, 0, len(existing))
	for _, tag := range existing {
		if !drop[tag.Key] {
			out = append(out, tag)
		}
	}
	return out
}

func efsCopyTags(tags []efsTag) []efsTag {
	return append(make([]efsTag, 0, len(tags)), tags...)
}

func efsTagValue(tags []efsTag, key string) (string, bool) {
	for _, tag := range tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}

// resourceTagsLocked returns a pointer to the tag list of a file system or
// access point named by ID or ARN.
func (s *efsStore) resourceTagsLocked(idOrARN string) (*[]efsTag, *efsError) {
	if efsAccessPointIDPattern.MatchString(idOrARN) {
		ap, ok := s.accessPoints[efsResourceID(idOrARN)]
		if !ok {
			return nil, efsErr(http.StatusNotFound, "AccessPointNotFound", "Access point '%s' does not exist.", efsResourceID(idOrARN))
		}
		return &ap.Tags, nil
	}
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	return &fs.Tags, nil
}

func (s *efsStore) tagResource(idOrARN string, tags []efsTag) *efsError {
	tags, err := efsValidateTags(tags)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return efsBadRequest("Tags must contain at least one tag.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.resourceTagsLocked(idOrARN)
	if err != nil {
		return err
	}
	merged, err := efsMergeTags(*current, tags)
	if err != nil {
		return err
	}
	*current = merged
	return nil
}

func (s *efsStore) untagResource(idOrARN string, keys []string) *efsError {
	if len(keys) == 0 {
		return efsBadRequest("TagKeys must contain at least one key.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.resourceTagsLocked(idOrARN)
	if err != nil {
		return err
	}
	*current = efsRemoveTags(*current, keys)
	return nil
}

// listTags pages a resource's tags; legacy selects the Marker/NextMarker
// shape of DescribeTags over the NextToken shape of ListTagsForResource.
func (s *efsStore) listTags(idOrARN, marker string, limit int, legacy bool) (map[string]any, *efsError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var current *[]efsTag
	var err *efsError
	if legacy {
		var fs *efsFileSystem
		if fs, err = s.fileSystemLocked(idOrARN); err == nil {
			current = &fs.Tags
		}
	} else {
		current, err = s.resourceTagsLocked(idOrARN)
	}
	if err != nil {
		return nil, err
	}
	page, next, err := efsPage(efsCopyTags(*current), func(t efsTag) string { return t.Key }, marker, efsDefaultLimit(limit, efsDefaultPage))
	if err != nil {
		return nil, err
	}
	if legacy {
		return efsMarkerResponse("Tags", page, marker, next), nil
	}
	return efsTokenResponse("Tags", page, next), nil
}

func efsTokenResponse(key string, items any, next string) map[string]any {
	out := map[string]any{key: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out
}

// ---- mount targets ----

type efsCreateMountTargetInput struct {
	FileSystemId   *string  `json:"FileSystemId"`
	SubnetId       *string  `json:"SubnetId"`
	IpAddress      *string  `json:"IpAddress"`
	Ipv6Address    *string  `json:"Ipv6Address"`
	IpAddressType  *string  `json:"IpAddressType"`
	SecurityGroups []string `json:"SecurityGroups"`
}

func efsDescribeMountTarget(mt *efsMountTarget, state string) map[string]any {
	out := map[string]any{
		"OwnerId": efsAccountID, "MountTargetId": mt.ID, "FileSystemId": mt.FileSystemID,
		"SubnetId": mt.SubnetID, "LifeCycleState": state, "IpAddress": mt.IP,
		"NetworkInterfaceId": mt.ENI, "AvailabilityZoneName": mt.AZ, "VpcId": mt.VpcID,
	}
	if id := efsAZID(mt.AZ); id != "" {
		out["AvailabilityZoneId"] = id
	}
	return out
}

// efsCheckSecurityGroups dedupes the groups (defaulting to the VPC's default
// group) and checks each exists in the subnet's VPC.
func efsCheckSecurityGroups(groups []string, subnet efsSubnet) ([]string, *efsError) {
	if len(groups) == 0 && subnet.DefaultGroup != "" {
		groups = []string{subnet.DefaultGroup}
	}
	out := make([]string, 0, len(groups))
	seen := map[string]bool{}
	for _, g := range groups {
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	if len(out) > efsMaxSecurityGroups {
		return nil, efsErr(http.StatusBadRequest, "SecurityGroupLimitExceeded", "A mount target can have at most %d security groups.", efsMaxSecurityGroups)
	}
	for _, g := range out {
		vpc, ok := subnet.SecurityGroups[g]
		if !ok {
			return nil, efsErr(http.StatusBadRequest, "SecurityGroupNotFound", "Security group '%s' does not exist.", g)
		}
		if vpc != subnet.VpcID {
			return nil, efsErr(http.StatusBadRequest, "SecurityGroupNotFound", "Security group '%s' is not in VPC '%s'.", g, subnet.VpcID)
		}
	}
	return out, nil
}

// pickIPLocked validates a requested address or allocates the lowest free one.
// AWS reserves the first four addresses and the last address of a subnet.
// ponytail: only EFS mount targets are tracked as address users, not EC2 ENIs.
func (s *efsStore) pickIPLocked(subnet efsSubnet, requested string) (string, *efsError) {
	prefix, err := netip.ParsePrefix(subnet.CIDR)
	if err != nil || !prefix.Addr().Is4() {
		return "", efsBadRequest("Subnet '%s' has no IPv4 CIDR block.", subnet.ID)
	}
	prefix = prefix.Masked()
	used := map[string]bool{}
	for _, mt := range s.mountTargets {
		if mt.SubnetID == subnet.ID {
			used[mt.IP] = true
		}
	}
	first := prefix.Addr()
	for range 4 {
		first = first.Next()
	}
	reserved := func(a netip.Addr) bool {
		return a.Less(first) || !prefix.Contains(a.Next())
	}
	if requested != "" {
		addr, err := netip.ParseAddr(requested)
		if err != nil || !addr.Is4() || !prefix.Contains(addr) || reserved(addr) {
			return "", efsBadRequest("IpAddress '%s' is not a usable address in subnet '%s' (%s).", requested, subnet.ID, subnet.CIDR)
		}
		if used[requested] {
			return "", efsErr(http.StatusConflict, "IpAddressInUse", "IP address '%s' is already in use in subnet '%s'.", requested, subnet.ID)
		}
		return requested, nil
	}
	for a := first; prefix.Contains(a) && !reserved(a); a = a.Next() {
		if !used[a.String()] {
			return a.String(), nil
		}
	}
	return "", efsErr(http.StatusConflict, "NoFreeAddressesInSubnet", "Subnet '%s' has no free addresses.", subnet.ID)
}

func (s *efsStore) createMountTarget(in efsCreateMountTargetInput, subnet *efsSubnet) (map[string]any, *efsError) {
	switch efsDefault(in.IpAddressType, "IPV4_ONLY") {
	case "IPV4_ONLY":
		if efsDeref(in.Ipv6Address) != "" {
			return nil, efsBadRequest("Ipv6Address requires IpAddressType IPV6_ONLY or DUAL_STACK.")
		}
	case "IPV6_ONLY", "DUAL_STACK":
		// ponytail: the EC2 store has no subnet IPv6 CIDRs, so IPv6 mount
		// targets can't be placed; add when EC2 subnets carry IPv6 blocks.
		return nil, efsBadRequest("Subnet '%s' has no IPv6 CIDR block.", efsDeref(in.SubnetId))
	default:
		return nil, efsBadRequest("IpAddressType must be IPV4_ONLY, IPV6_ONLY or DUAL_STACK.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(efsDeref(in.FileSystemId))
	if err != nil {
		return nil, err
	}
	if subnet == nil {
		return nil, efsErr(http.StatusBadRequest, "SubnetNotFound", "Subnet '%s' does not exist.", efsDeref(in.SubnetId))
	}
	groups, err := efsCheckSecurityGroups(in.SecurityGroups, *subnet)
	if err != nil {
		return nil, err
	}
	if fs.AvailabilityZoneName != "" && fs.AvailabilityZoneName != subnet.AZ {
		return nil, efsErr(http.StatusBadRequest, "AvailabilityZonesMismatch", "One Zone file system '%s' is in %s, but subnet '%s' is in %s.", fs.ID, fs.AvailabilityZoneName, subnet.ID, subnet.AZ)
	}
	for _, mt := range s.mountTargets {
		if mt.FileSystemID != fs.ID {
			continue
		}
		if mt.VpcID != subnet.VpcID {
			return nil, efsErr(http.StatusConflict, "MountTargetConflict", "File system '%s' already has mount targets in VPC '%s'.", fs.ID, mt.VpcID)
		}
		if mt.AZ == subnet.AZ {
			return nil, efsErr(http.StatusConflict, "MountTargetConflict", "File system '%s' already has a mount target in %s.", fs.ID, subnet.AZ)
		}
	}
	ip, err := s.pickIPLocked(*subnet, efsDeref(in.IpAddress))
	if err != nil {
		return nil, err
	}
	s.seq++
	mt := &efsMountTarget{
		seq: s.seq, ID: s.newIDLocked("fsmt", true), FileSystemID: fs.ID, SubnetID: subnet.ID,
		VpcID: subnet.VpcID, AZ: subnet.AZ, IP: ip, ENI: "eni-" + efsRandomHex(17), SecurityGroups: groups,
	}
	s.mountTargets[mt.ID] = mt
	return efsDescribeMountTarget(mt, "creating"), nil
}

func (s *efsStore) mountTargetLocked(id string) (*efsMountTarget, *efsError) {
	if !efsMountTargetIDPattern.MatchString(id) {
		return nil, efsBadRequest("MountTargetId %q is not a valid mount target ID.", id)
	}
	mt, ok := s.mountTargets[id]
	if !ok {
		return nil, efsErr(http.StatusNotFound, "MountTargetNotFound", "Mount target '%s' does not exist.", id)
	}
	return mt, nil
}

func (s *efsStore) deleteMountTarget(id string) *efsError {
	s.mu.Lock()
	defer s.mu.Unlock()
	mt, err := s.mountTargetLocked(id)
	if err != nil {
		return err
	}
	delete(s.mountTargets, mt.ID)
	return nil
}

func (s *efsStore) describeMountTargets(fsID, mtID, apID, marker string, limit int) (map[string]any, *efsError) {
	set := 0
	for _, v := range []string{fsID, mtID, apID} {
		if v != "" {
			set++
		}
	}
	if set != 1 {
		return nil, efsBadRequest("Specify exactly one of FileSystemId, MountTargetId or AccessPointId.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*efsMountTarget
	switch {
	case mtID != "":
		mt, err := s.mountTargetLocked(mtID)
		if err != nil {
			return nil, err
		}
		list = []*efsMountTarget{mt}
	default:
		if apID != "" {
			ap, err := s.accessPointLocked(apID)
			if err != nil {
				return nil, err
			}
			fsID = ap.FileSystemID
		}
		fs, err := s.fileSystemLocked(fsID)
		if err != nil {
			return nil, err
		}
		for _, mt := range s.mountTargets {
			if mt.FileSystemID == fs.ID {
				list = append(list, mt)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	}
	page, next, err := efsPage(list, func(mt *efsMountTarget) string { return mt.ID }, marker, efsDefaultLimit(limit, efsDefaultMountTargetPage))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(page))
	for _, mt := range page {
		items = append(items, efsDescribeMountTarget(mt, efsLifeCycleAvailable))
	}
	return efsMarkerResponse("MountTargets", items, marker, next), nil
}

func (s *efsStore) mountTargetSubnetID(id string) (string, *efsError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mt, err := s.mountTargetLocked(id)
	if err != nil {
		return "", err
	}
	return mt.SubnetID, nil
}

// mountTargetSecurityGroups reads, or with subnet set replaces, the groups.
func (s *efsStore) mountTargetSecurityGroups(id string, groups []string, subnet *efsSubnet) ([]string, *efsError) {
	if subnet != nil {
		if len(groups) == 0 {
			return nil, efsBadRequest("SecurityGroups must contain at least one security group.")
		}
		var err *efsError
		if groups, err = efsCheckSecurityGroups(groups, *subnet); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mt, err := s.mountTargetLocked(id)
	if err != nil {
		return nil, err
	}
	if subnet != nil {
		mt.SecurityGroups = groups
	}
	return append([]string{}, mt.SecurityGroups...), nil
}

// ---- access points ----

type efsCreateAccessPointInput struct {
	ClientToken   *string           `json:"ClientToken"`
	Tags          []efsTag          `json:"Tags"`
	FileSystemId  *string           `json:"FileSystemId"`
	PosixUser     *efsPosixUser     `json:"PosixUser"`
	RootDirectory *efsRootDirectory `json:"RootDirectory"`
}

func efsValidUnixID(v *int64) bool { return v != nil && *v >= 0 && *v <= efsMaxUnixID }

func efsValidateAccessPoint(in efsCreateAccessPointInput) (efsRootDirectory, *efsError) {
	if p := in.PosixUser; p != nil {
		if !efsValidUnixID(p.Uid) || !efsValidUnixID(p.Gid) {
			return efsRootDirectory{}, efsBadRequest("PosixUser requires Uid and Gid between 0 and %d.", efsMaxUnixID)
		}
		if len(p.SecondaryGids) > 16 {
			return efsRootDirectory{}, efsBadRequest("PosixUser.SecondaryGids can hold at most 16 IDs.")
		}
		for i := range p.SecondaryGids {
			if !efsValidUnixID(&p.SecondaryGids[i]) {
				return efsRootDirectory{}, efsBadRequest("SecondaryGids must be between 0 and %d.", efsMaxUnixID)
			}
		}
	}
	path := "/"
	root := efsRootDirectory{}
	if in.RootDirectory != nil {
		root = *in.RootDirectory
		if root.Path != nil {
			path = *root.Path
		}
		if c := root.CreationInfo; c != nil {
			if !efsValidUnixID(c.OwnerUid) || !efsValidUnixID(c.OwnerGid) || c.Permissions == nil || !efsPermissionsPattern.MatchString(*c.Permissions) {
				return efsRootDirectory{}, efsBadRequest("CreationInfo requires OwnerUid, OwnerGid and octal Permissions (e.g. 755).")
			}
		}
	}
	if !strings.HasPrefix(path, "/") || len(path) > 100 || strings.Contains(path, "//") {
		return efsRootDirectory{}, efsBadRequest("RootDirectory.Path must be an absolute path of at most 100 characters.")
	}
	root.Path = &path
	return root, nil
}

func (s *efsStore) describeAccessPointLocked(ap *efsAccessPoint, state string) map[string]any {
	out := map[string]any{
		"ClientToken": ap.ClientToken, "Tags": efsCopyTags(ap.Tags), "AccessPointId": ap.ID,
		"AccessPointArn": ap.ARN, "FileSystemId": ap.FileSystemID, "RootDirectory": ap.Root,
		"OwnerId": efsAccountID, "LifeCycleState": state,
	}
	if ap.PosixUser != nil {
		out["PosixUser"] = ap.PosixUser
	}
	if name, ok := efsTagValue(ap.Tags, "Name"); ok {
		out["Name"] = name
	}
	return out
}

func (s *efsStore) createAccessPoint(in efsCreateAccessPointInput) (map[string]any, *efsError) {
	token := efsDeref(in.ClientToken)
	if token == "" || len(token) > 64 {
		return nil, efsBadRequest("ClientToken is required and must be 1-64 characters.")
	}
	root, err := efsValidateAccessPoint(in)
	if err != nil {
		return nil, err
	}
	tags, err := efsValidateTags(in.Tags)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(efsDeref(in.FileSystemId))
	if err != nil {
		return nil, err
	}
	count := 0
	for _, ap := range s.accessPoints {
		if ap.ClientToken == token {
			e := efsErr(http.StatusConflict, "AccessPointAlreadyExists", "Access point '%s' already exists with client token '%s'.", ap.ID, token)
			e.extra = map[string]string{"AccessPointId": ap.ID}
			return nil, e
		}
		if ap.FileSystemID == fs.ID {
			count++
		}
	}
	if count >= efsMaxAccessPointsPerFS {
		return nil, efsErr(http.StatusForbidden, "AccessPointLimitExceeded", "File system '%s' has the maximum of %d access points.", fs.ID, efsMaxAccessPointsPerFS)
	}
	s.seq++
	id := s.newIDLocked("fsap", false)
	ap := &efsAccessPoint{seq: s.seq, ID: id, ARN: efsAccessPointARN(id), FileSystemID: fs.ID, ClientToken: token, Tags: tags, PosixUser: in.PosixUser, Root: root}
	s.accessPoints[id] = ap
	return s.describeAccessPointLocked(ap, "creating"), nil
}

func (s *efsStore) accessPointLocked(idOrARN string) (*efsAccessPoint, *efsError) {
	if !efsAccessPointIDPattern.MatchString(idOrARN) {
		return nil, efsBadRequest("AccessPointId %q is not a valid access point ID or ARN.", idOrARN)
	}
	ap, ok := s.accessPoints[efsResourceID(idOrARN)]
	if !ok {
		return nil, efsErr(http.StatusNotFound, "AccessPointNotFound", "Access point '%s' does not exist.", efsResourceID(idOrARN))
	}
	return ap, nil
}

func (s *efsStore) deleteAccessPoint(idOrARN string) *efsError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ap, err := s.accessPointLocked(idOrARN)
	if err != nil {
		return err
	}
	delete(s.accessPoints, ap.ID)
	return nil
}

func (s *efsStore) describeAccessPoints(apID, fsID, token string, limit int) (map[string]any, *efsError) {
	if apID != "" && fsID != "" {
		return nil, efsBadRequest("Specify either AccessPointId or FileSystemId, not both.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*efsAccessPoint
	switch {
	case apID != "":
		ap, err := s.accessPointLocked(apID)
		if err != nil {
			return nil, err
		}
		list = []*efsAccessPoint{ap}
	default:
		if fsID != "" {
			fs, err := s.fileSystemLocked(fsID)
			if err != nil {
				return nil, err
			}
			fsID = fs.ID
		}
		for _, ap := range s.accessPoints {
			if fsID == "" || ap.FileSystemID == fsID {
				list = append(list, ap)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	}
	page, next, err := efsPage(list, func(ap *efsAccessPoint) string { return ap.ID }, token, efsDefaultLimit(limit, efsDefaultPage))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(page))
	for _, ap := range page {
		items = append(items, s.describeAccessPointLocked(ap, efsLifeCycleAvailable))
	}
	return efsTokenResponse("AccessPoints", items, next), nil
}

// ---- replication ----

type efsDestinationInput struct {
	Region               *string `json:"Region"`
	AvailabilityZoneName *string `json:"AvailabilityZoneName"`
	KmsKeyId             *string `json:"KmsKeyId"`
	FileSystemId         *string `json:"FileSystemId"`
	RoleArn              *string `json:"RoleArn"`
}

func (s *efsStore) describeReplicationLocked(rep *efsReplication) map[string]any {
	status := efsReplicationEnabled
	if !rep.announced {
		status, rep.announced = efsReplicationEnabling, true
	}
	dest := map[string]any{"Status": status, "FileSystemId": rep.DestID, "Region": rep.DestRegion, "OwnerId": efsAccountID}
	if rep.RoleArn != "" {
		dest["RoleArn"] = rep.RoleArn
	}
	arn := efsFileSystemARN(rep.SourceID)
	return map[string]any{
		"SourceFileSystemId": rep.SourceID, "SourceFileSystemRegion": efsRegion,
		"SourceFileSystemArn": arn, "OriginalSourceFileSystemArn": arn,
		"CreationTime": efsTime(rep.CreationTime), "Destinations": []map[string]any{dest},
		"SourceFileSystemOwnerId": efsAccountID,
	}
}

// createReplication sets up replication to one destination, creating the
// destination file system unless an existing one is named.
// ponytail: destination file systems live in this same (single-region)
// store; the Region is recorded but not partitioned.
func (s *efsStore) createReplication(sourceIDOrARN string, dests []efsDestinationInput) (map[string]any, *efsError) {
	if len(dests) != 1 {
		return nil, efsErr(http.StatusBadRequest, "ValidationException", "Destinations must contain exactly one destination.")
	}
	d := dests[0]
	region := efsDefault(d.Region, efsRegion)
	az := efsDeref(d.AvailabilityZoneName)
	kms := efsDeref(d.KmsKeyId)
	if kms != "" && !efsKMSKeyIDPattern.MatchString(kms) {
		return nil, efsBadRequest("KmsKeyId %q is not a valid KMS key ID, ARN or alias.", kms)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.fileSystemLocked(sourceIDOrARN)
	if err != nil {
		return nil, err
	}
	if src.ReplicationSource != "" {
		return nil, efsErr(http.StatusBadRequest, "ValidationException", "File system '%s' is already part of a replication configuration.", src.ID)
	}
	var dest *efsFileSystem
	if destID := efsDeref(d.FileSystemId); destID != "" {
		if dest, err = s.fileSystemLocked(destID); err != nil {
			return nil, err
		}
		if dest.ID == src.ID || dest.ReplicationSource != "" {
			return nil, efsErr(http.StatusBadRequest, "ValidationException", "File system '%s' can't be the destination.", dest.ID)
		}
		if dest.Protection != efsProtectionDisabled {
			return nil, efsErr(http.StatusConflict, "ConflictException", "Destination '%s' must have replication overwrite protection DISABLED.", dest.ID)
		}
	} else {
		if az != "" && efsAZID(az) == "" {
			return nil, efsErr(http.StatusBadRequest, "UnsupportedAvailabilityZone", "Availability Zone '%s' is not supported.", az)
		}
		if len(s.fileSystems) >= efsMaxFileSystems {
			return nil, efsErr(http.StatusForbidden, "FileSystemLimitExceeded", "You have reached the maximum number of file systems allowed for your account.")
		}
		encrypted := src.Encrypted || kms != ""
		dest = s.addFileSystemLocked("replica-"+src.ID+"-"+efsRandomHex(8), src.PerformanceMode, efsThroughputElastic, nil, encrypted, kms, az, nil)
		if az != "" {
			dest.BackupStatus = efsBackupEnabled
		}
	}
	dest.Protection = efsProtectionReplicating
	src.ReplicationSource, dest.ReplicationSource = src.ID, src.ID
	rep := &efsReplication{SourceID: src.ID, CreationTime: time.Now().UTC(), DestID: dest.ID, DestRegion: region, RoleArn: efsDeref(d.RoleArn)}
	s.replications[src.ID] = rep
	return s.describeReplicationLocked(rep), nil
}

func (s *efsStore) replicationForLocked(fs *efsFileSystem) (*efsReplication, *efsError) {
	rep, ok := s.replications[fs.ReplicationSource]
	if fs.ReplicationSource == "" || !ok {
		return nil, efsErr(http.StatusNotFound, "ReplicationNotFound", "File system '%s' has no replication configuration.", fs.ID)
	}
	return rep, nil
}

func (s *efsStore) describeReplications(fsID, token string, limit int) (map[string]any, *efsError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*efsReplication
	if fsID != "" {
		fs, err := s.fileSystemLocked(fsID)
		if err != nil {
			return nil, err
		}
		rep, err := s.replicationForLocked(fs)
		if err != nil {
			return nil, err
		}
		list = []*efsReplication{rep}
	} else {
		for _, rep := range s.replications {
			list = append(list, rep)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].SourceID < list[j].SourceID })
	}
	page, next, err := efsPage(list, func(r *efsReplication) string { return r.SourceID }, token, efsDefaultLimit(limit, efsDefaultPage))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(page))
	for _, rep := range page {
		items = append(items, s.describeReplicationLocked(rep))
	}
	return efsTokenResponse("Replications", items, next), nil
}

// deleteReplication removes the replication named by its source or
// destination; the destination becomes writable with protection ENABLED.
func (s *efsStore) deleteReplication(fsID string) *efsError {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(fsID)
	if err != nil {
		return err
	}
	rep, err := s.replicationForLocked(fs)
	if err != nil {
		return err
	}
	if src, ok := s.fileSystems[rep.SourceID]; ok {
		src.ReplicationSource = ""
	}
	if dest, ok := s.fileSystems[rep.DestID]; ok {
		dest.ReplicationSource, dest.Protection = "", efsProtectionEnabled
	}
	delete(s.replications, rep.SourceID)
	return nil
}
