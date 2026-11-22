package server

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// A location keeps its validated create input (model members only) in attrs;
// Describe returns attrs plus the ARN, URI and creation time, shaped to the
// Describe output so secrets (passwords, keys, SAS tokens) never come back.

type dsyLocation struct {
	seq     int
	arn     string
	kind    string // the CreateLocation<kind> suffix, e.g. "S3"
	uri     string
	created time.Time
	attrs   map[string]any
	tags    map[string]string
}

type dsyLocationKind struct {
	check func(s *Server, a map[string]any) *dsyErr
	uri   func(a map[string]any) string
	// extra adds derived Describe members.
	extra func(a map[string]any, out map[string]any)
}

var (
	dsyS3BucketARNPattern = regexp.MustCompile(`^arn:aws[-a-z]*:s3:::([a-z0-9][a-z0-9.\-]{1,61}[a-z0-9])$`)
	dsyS3OtherARNPattern  = regexp.MustCompile(`^arn:aws[-a-z]*:s3(-outposts)?:[a-z0-9-]+:\d{12}:(accesspoint|outpost)/[A-Za-z0-9./\-]+$`)
	dsyRoleARNPattern     = regexp.MustCompile(`^arn:aws[-a-z]*:iam::\d{12}:role/[A-Za-z0-9+=,.@/_-]+$`)
	dsyEFSARNPattern      = regexp.MustCompile(`^arn:aws[-a-z]*:elasticfilesystem:[a-z0-9-]+:\d{12}:file-system/(fs-[0-9a-f]{8,40})$`)
	dsyEFSAPARNPattern    = regexp.MustCompile(`^arn:aws[-a-z]*:elasticfilesystem:[a-z0-9-]+:\d{12}:access-point/fsap-[0-9a-f]{8,40}$`)
	dsyFSxARNPattern      = regexp.MustCompile(`^arn:aws[-a-z]*:fsx:[a-z0-9-]+:\d{12}:file-system/(fs-[0-9a-f]{8,21})$`)
	dsySVMARNPattern      = regexp.MustCompile(`^arn:aws[-a-z]*:fsx:[a-z0-9-]+:\d{12}:storage-virtual-machine/(fs-[0-9a-f]{8,21})/(svm-[0-9a-f]{17})$`)
	dsyAgentARNPattern    = regexp.MustCompile(`^arn:aws[-a-z]*:datasync:[a-z0-9-]+:\d{12}:agent/agent-[0-9a-z]{17}$`)
	dsyHostPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\-:\[\]]{0,254}$`)
	dsyAzureURLPattern    = regexp.MustCompile(`^https://([a-z0-9]{3,24})\.blob\.core\.windows\.net/([a-z0-9][a-z0-9\-]{1,61}[a-z0-9])/?$`)
)

var dsyLocationKinds = map[string]dsyLocationKind{
	"S3": {check: dsyCheckS3, uri: func(a map[string]any) string {
		bucket := dsyStr(a, "S3BucketArn")
		if m := dsyS3BucketARNPattern.FindStringSubmatch(bucket); m != nil {
			bucket = m[1]
		}
		return "s3://" + bucket + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
	"Efs": {check: dsyCheckEFS, uri: func(a map[string]any) string {
		id := dsyEFSARNPattern.FindStringSubmatch(dsyStr(a, "EfsFilesystemArn"))
		return "efs://" + dsyRegion + "." + id[1] + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
	"FsxLustre":  {check: dsyCheckFSx(false, false), uri: dsyFSxURI("fsxl")},
	"FsxWindows": {check: dsyCheckFSx(false, false), uri: dsyFSxURI("fsxw")},
	"FsxOpenZfs": {check: dsyCheckFSx(true, true), uri: dsyFSxURI("fsxz")},
	"FsxOntap": {check: dsyCheckOntap, uri: func(a map[string]any) string {
		m := dsySVMARNPattern.FindStringSubmatch(dsyStr(a, "StorageVirtualMachineArn"))
		return "fsxn://" + dsyRegion + "." + m[2] + dsySubdir(dsyStr(a, "Subdirectory"))
	}, extra: func(a map[string]any, out map[string]any) {
		m := dsySVMARNPattern.FindStringSubmatch(dsyStr(a, "StorageVirtualMachineArn"))
		out["FsxFilesystemArn"] = fmt.Sprintf("arn:aws:fsx:%s:%s:file-system/%s", dsyRegion, dsyAccountID, m[1])
	}},
	"Nfs": {check: dsyCheckNFS, uri: func(a map[string]any) string {
		return "nfs://" + dsyStr(a, "ServerHostname") + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
	"Smb": {check: dsyCheckSMB, uri: func(a map[string]any) string {
		return "smb://" + dsyStr(a, "ServerHostname") + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
	"Hdfs": {check: dsyCheckHDFS, uri: func(a map[string]any) string {
		nodes, _ := a["NameNodes"].([]any)
		first, _ := nodes[0].(map[string]any)
		port, _ := first["Port"].(float64)
		return fmt.Sprintf("hdfs://%s:%d%s", dsyStr(first, "Hostname"), int(port), dsySubdir(dsyStr(a, "Subdirectory")))
	}},
	"ObjectStorage": {check: dsyCheckObjectStorage, uri: func(a map[string]any) string {
		return "object-storage://" + dsyStr(a, "ServerHostname") + "/" + dsyStr(a, "BucketName") + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
	"AzureBlob": {check: dsyCheckAzure, uri: func(a map[string]any) string {
		m := dsyAzureURLPattern.FindStringSubmatch(dsyStr(a, "ContainerUrl"))
		return "azure-blob://" + m[1] + ".blob.core.windows.net/" + m[2] + dsySubdir(dsyStr(a, "Subdirectory"))
	}},
}

func init() {
	for kind := range dsyLocationKinds {
		kind := kind
		dsyHandlers["CreateLocation"+kind] = func(s *Server, in map[string]any) (map[string]any, *dsyErr) { return dsyCreateLocation(s, kind, in) }
		dsyHandlers["DescribeLocation"+kind] = func(s *Server, in map[string]any) (map[string]any, *dsyErr) { return dsyDescribeLocation(s, kind, in) }
		dsyHandlers["UpdateLocation"+kind] = func(s *Server, in map[string]any) (map[string]any, *dsyErr) { return dsyUpdateLocation(s, kind, in) }
	}
	dsyHandlers["DeleteLocation"] = dsyDeleteLocation
	dsyHandlers["ListLocations"] = dsyListLocations
}

func dsyCreateLocation(s *Server, kind string, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	if len(st.locations) >= dsyMaxLocations {
		return nil, dsyInvalid("an account can have at most %d locations", dsyMaxLocations)
	}
	attrs := dsyCopyMap(in)
	delete(attrs, "Tags")
	spec := dsyLocationKinds[kind]
	if err := spec.check(s, attrs); err != nil {
		return nil, err
	}
	loc := &dsyLocation{kind: kind, attrs: attrs, tags: map[string]string{}}
	if err := dsyTagsFrom(in, "Tags", loc.tags); err != nil {
		return nil, err
	}
	loc.seq, loc.arn, loc.created, loc.uri = st.next(), dsyARN("location/"+dsyID("loc-")), time.Now().UTC(), spec.uri(attrs)
	st.locations[loc.arn] = loc
	return map[string]any{"LocationArn": loc.arn}, nil
}

func (st *dsyStore) location(arn, kind string) (*dsyLocation, *dsyErr) {
	loc, ok := st.locations[arn]
	if !ok {
		return nil, dsyNotFound("Location", arn)
	}
	if kind != "" && loc.kind != kind {
		return nil, dsyInvalid("location %s is a %s location, not %s", arn, loc.kind, kind)
	}
	return loc, nil
}

func dsyDescribeLocation(s *Server, kind string, in map[string]any) (map[string]any, *dsyErr) {
	loc, err := s.datasync.location(dsyStr(in, "LocationArn"), kind)
	if err != nil {
		return nil, err
	}
	out := dsyCopyMap(loc.attrs)
	if spec := dsyLocationKinds[kind]; spec.extra != nil {
		spec.extra(loc.attrs, out)
	}
	out["LocationArn"], out["LocationUri"], out["CreationTime"] = loc.arn, loc.uri, loc.created
	return out, nil
}

func dsyUpdateLocation(s *Server, kind string, in map[string]any) (map[string]any, *dsyErr) {
	loc, err := s.datasync.location(dsyStr(in, "LocationArn"), kind)
	if err != nil {
		return nil, err
	}
	merged := dsyCopyMap(loc.attrs)
	for k, v := range dsyCopyMap(in) {
		if k != "LocationArn" {
			merged[k] = v
		}
	}
	spec := dsyLocationKinds[kind]
	if err := spec.check(s, merged); err != nil {
		return nil, err
	}
	loc.attrs, loc.uri = merged, spec.uri(merged)
	return map[string]any{}, nil
}

func dsyDeleteLocation(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	loc, err := st.location(dsyStr(in, "LocationArn"), "")
	if err != nil {
		return nil, err
	}
	for _, t := range st.tasks {
		if t.src == loc.arn || t.dst == loc.arn {
			return nil, dsyInvalid("location %s is used by task %s; delete the task first", loc.arn, t.arn)
		}
	}
	delete(st.locations, loc.arn)
	return map[string]any{}, nil
}

// dsyLocationType is the ListLocations LocationType value: the URI scheme.
func dsyLocationType(loc *dsyLocation) string {
	scheme, _, _ := strings.Cut(loc.uri, "://")
	return strings.ToUpper(strings.ReplaceAll(scheme, "-", "_"))
}

func dsyListLocations(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	locs := make([]*dsyLocation, 0, len(s.datasync.locations))
	for _, loc := range s.datasync.locations {
		if dsyLocationMatches(loc, in) {
			locs = append(locs, loc)
		}
	}
	sort.Slice(locs, func(i, j int) bool { return locs[i].seq < locs[j].seq })
	start, end, next, err := dsyPage(in, len(locs))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, loc := range locs[start:end] {
		list = append(list, map[string]any{"LocationArn": loc.arn, "LocationUri": loc.uri})
	}
	return map[string]any{"Locations": list, "NextToken": dsyOptional(next)}, nil
}

func dsyLocationMatches(loc *dsyLocation, in map[string]any) bool {
	filters, _ := in["Filters"].([]any)
	for _, raw := range filters {
		f, _ := raw.(map[string]any)
		var value string
		switch dsyStr(f, "Name") {
		case "LocationUri":
			value = loc.uri
		case "LocationType":
			value = dsyLocationType(loc)
		case "CreationTime":
			value = fmt.Sprint(loc.created.Unix())
		}
		if !dsyFilterMatch(dsyStr(f, "Operator"), value, dsyStrings(f, "Values")) {
			return false
		}
	}
	return true
}

// --- per-kind validation -------------------------------------------------

func dsyCheckAgents(s *Server, label string, arns []string, required bool) *dsyErr {
	if required && len(arns) == 0 {
		return dsyInvalid("%s must name at least one agent", label)
	}
	for _, arn := range arns {
		if !dsyAgentARNPattern.MatchString(arn) {
			return dsyInvalid("%s %q is not an agent ARN", label, arn)
		}
		if _, err := s.datasync.agent(arn); err != nil {
			return err
		}
	}
	return nil
}

func dsyCheckS3(s *Server, a map[string]any) *dsyErr {
	arn := dsyStr(a, "S3BucketArn")
	if m := dsyS3BucketARNPattern.FindStringSubmatch(arn); m != nil {
		if _, err := s.s3.GetBucket(m[1]); err != nil {
			return dsyInvalid("S3 bucket %s does not exist", m[1])
		}
	} else if !dsyS3OtherARNPattern.MatchString(arn) {
		// ponytail: access point and Outposts ARNs are accepted by format;
		// only plain bucket ARNs are checked against the S3 store.
		return dsyInvalid("S3BucketArn %q is not a bucket, access point or Outposts ARN", arn)
	}
	if _, ok := a["S3StorageClass"]; !ok {
		a["S3StorageClass"] = "STANDARD"
	}
	if role := dsyStr(dsyMap(a, "S3Config"), "BucketAccessRoleArn"); !dsyRoleARNPattern.MatchString(role) {
		return dsyInvalid("S3Config.BucketAccessRoleArn %q is not an IAM role ARN", role)
	}
	return dsyCheckAgents(s, "AgentArns", dsyStrings(a, "AgentArns"), false)
}

func dsyCheckEFS(s *Server, a map[string]any) *dsyErr {
	m := dsyEFSARNPattern.FindStringSubmatch(dsyStr(a, "EfsFilesystemArn"))
	if m == nil {
		return dsyInvalid("EfsFilesystemArn %q is not an EFS file system ARN", dsyStr(a, "EfsFilesystemArn"))
	}
	s.efs.mu.Lock()
	_, exists := s.efs.fileSystems[m[1]]
	s.efs.mu.Unlock()
	if !exists {
		return dsyInvalid("EFS file system %s does not exist", m[1])
	}
	ec2 := dsyMap(a, "Ec2Config")
	if subnet := dsyStr(ec2, "SubnetArn"); !dsySubnetARNPattern.MatchString(subnet) {
		return dsyInvalid("Ec2Config.SubnetArn %q is not a subnet ARN", subnet)
	}
	if err := dsyCheckARNs("Ec2Config.SecurityGroupArns", dsyStrings(ec2, "SecurityGroupArns"), dsySGARNPattern); err != nil {
		return err
	}
	ap, role := dsyStr(a, "AccessPointArn"), dsyStr(a, "FileSystemAccessRoleArn")
	if ap != "" && !dsyEFSAPARNPattern.MatchString(ap) {
		return dsyInvalid("AccessPointArn %q is not an EFS access point ARN", ap)
	}
	if role != "" && !dsyRoleARNPattern.MatchString(role) {
		return dsyInvalid("FileSystemAccessRoleArn %q is not an IAM role ARN", role)
	}
	// DataSync only mounts through an access point or with an IAM role over TLS.
	if (ap != "" || role != "") && dsyStr(a, "InTransitEncryption") != "TLS1_2" {
		return dsyInvalid("an access point or file system access role needs InTransitEncryption TLS1_2")
	}
	return nil
}

// dsyCheckFSx validates the FSx Lustre/Windows/OpenZFS shapes. ponytail: FSx
// file systems are checked by ARN format, not against an FSx store.
func dsyCheckFSx(protocol, nfsOnly bool) func(s *Server, a map[string]any) *dsyErr {
	return func(s *Server, a map[string]any) *dsyErr {
		if arn := dsyStr(a, "FsxFilesystemArn"); !dsyFSxARNPattern.MatchString(arn) {
			return dsyInvalid("FsxFilesystemArn %q is not an FSx file system ARN", arn)
		}
		if err := dsyCheckARNs("SecurityGroupArns", dsyStrings(a, "SecurityGroupArns"), dsySGARNPattern); err != nil {
			return err
		}
		if protocol {
			p := dsyMap(a, "Protocol")
			if nfsOnly && (p["NFS"] == nil || p["SMB"] != nil) {
				return dsyInvalid("FSx for OpenZFS locations support the NFS protocol only")
			}
		}
		return nil
	}
}

func dsyFSxURI(scheme string) func(a map[string]any) string {
	return func(a map[string]any) string {
		m := dsyFSxARNPattern.FindStringSubmatch(dsyStr(a, "FsxFilesystemArn"))
		return scheme + "://" + dsyRegion + "." + m[1] + dsySubdir(dsyStr(a, "Subdirectory"))
	}
}

func dsyCheckOntap(s *Server, a map[string]any) *dsyErr {
	if arn := dsyStr(a, "StorageVirtualMachineArn"); !dsySVMARNPattern.MatchString(arn) {
		return dsyInvalid("StorageVirtualMachineArn %q is not an FSx for ONTAP SVM ARN", arn)
	}
	if err := dsyCheckARNs("SecurityGroupArns", dsyStrings(a, "SecurityGroupArns"), dsySGARNPattern); err != nil {
		return err
	}
	p := dsyMap(a, "Protocol")
	if (p["NFS"] == nil) == (p["SMB"] == nil) {
		return dsyInvalid("Protocol must configure exactly one of NFS or SMB")
	}
	return nil
}

func dsyCheckHost(label, host string) *dsyErr {
	if !dsyHostPattern.MatchString(host) {
		return dsyInvalid("%s %q is not a valid host name or IP address", label, host)
	}
	return nil
}

func dsyCheckNFS(s *Server, a map[string]any) *dsyErr {
	if err := dsyCheckHost("ServerHostname", dsyStr(a, "ServerHostname")); err != nil {
		return err
	}
	return dsyCheckAgents(s, "OnPremConfig.AgentArns", dsyStrings(dsyMap(a, "OnPremConfig"), "AgentArns"), true)
}

func dsyCheckSMB(s *Server, a map[string]any) *dsyErr {
	if err := dsyCheckHost("ServerHostname", dsyStr(a, "ServerHostname")); err != nil {
		return err
	}
	if dsyStr(a, "AuthenticationType") == "KERBEROS" {
		if dsyStr(a, "KerberosPrincipal") == "" || dsyStr(a, "KerberosKeytab") == "" || dsyStr(a, "KerberosKrb5Conf") == "" {
			return dsyInvalid("Kerberos authentication needs KerberosPrincipal, KerberosKeytab and KerberosKrb5Conf")
		}
	} else if dsyStr(a, "User") == "" {
		return dsyInvalid("NTLM authentication needs a User")
	}
	return dsyCheckAgents(s, "AgentArns", dsyStrings(a, "AgentArns"), true)
}

func dsyCheckHDFS(s *Server, a map[string]any) *dsyErr {
	nodes, _ := a["NameNodes"].([]any)
	if len(nodes) == 0 {
		return dsyInvalid("NameNodes must name at least one NameNode")
	}
	for _, raw := range nodes {
		n, _ := raw.(map[string]any)
		if err := dsyCheckHost("NameNodes.Hostname", dsyStr(n, "Hostname")); err != nil {
			return err
		}
	}
	switch dsyStr(a, "AuthenticationType") {
	case "SIMPLE":
		if dsyStr(a, "SimpleUser") == "" {
			return dsyInvalid("SIMPLE authentication needs a SimpleUser")
		}
	case "KERBEROS":
		if dsyStr(a, "KerberosPrincipal") == "" || dsyStr(a, "KerberosKeytab") == "" || dsyStr(a, "KerberosKrb5Conf") == "" {
			return dsyInvalid("Kerberos authentication needs KerberosPrincipal, KerberosKeytab and KerberosKrb5Conf")
		}
	}
	return dsyCheckAgents(s, "AgentArns", dsyStrings(a, "AgentArns"), true)
}

func dsyCheckObjectStorage(s *Server, a map[string]any) *dsyErr {
	if err := dsyCheckHost("ServerHostname", dsyStr(a, "ServerHostname")); err != nil {
		return err
	}
	if _, ok := a["ServerProtocol"]; !ok {
		a["ServerProtocol"] = "HTTPS"
	}
	if _, ok := a["ServerPort"]; !ok {
		a["ServerPort"] = float64(443)
		if a["ServerProtocol"] == "HTTP" {
			a["ServerPort"] = float64(80)
		}
	}
	if (dsyStr(a, "AccessKey") == "") != (dsyStr(a, "SecretKey") == "") {
		return dsyInvalid("AccessKey and SecretKey must be given together")
	}
	return dsyCheckAgents(s, "AgentArns", dsyStrings(a, "AgentArns"), false)
}

func dsyCheckAzure(s *Server, a map[string]any) *dsyErr {
	if url := dsyStr(a, "ContainerUrl"); !dsyAzureURLPattern.MatchString(url) {
		return dsyInvalid("ContainerUrl %q is not an Azure Blob container URL", url)
	}
	if dsyStr(a, "AuthenticationType") == "SAS" && dsyStr(dsyMap(a, "SasConfiguration"), "Token") == "" {
		return dsyInvalid("SAS authentication needs SasConfiguration.Token")
	}
	return dsyCheckAgents(s, "AgentArns", dsyStrings(a, "AgentArns"), false)
}
