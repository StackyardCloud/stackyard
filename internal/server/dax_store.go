package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

const (
	daxRegion               = "us-east-1"
	daxAccountID            = "123456789012"
	daxDefaultParamGroup    = "default.dax1.0"
	daxDefaultSubnetGroup   = "default"
	daxMaxNodesPerCluster   = 11 // one primary plus ten replicas
	daxMaxNodesPerCustomer  = 500
	daxMaxClusters          = 200
	daxMaxParamGroups       = 200
	daxMaxSubnetGroups      = 200
	daxMaxSubnetsPerGroup   = 20
	daxMaxSecurityGroups    = 5
	daxMaxTags              = 50
	daxMaxEvents            = 1000
	daxMaxListInput         = 100 // names in a Describe* filter
	daxMaxDescription       = 255
	daxPortPlain            = 8111
	daxPortTLS              = 9111
	daxDefaultMaintenance   = "sun:05:00-sun:06:00"
	daxClusterNameMaxLength = 20
)

// daxNodeTypes are the DAX node types AWS offers.
var daxNodeTypes = map[string]bool{
	"dax.t2.small": true, "dax.t2.medium": true, "dax.t3.small": true, "dax.t3.medium": true,
	"dax.r4.large": true, "dax.r4.xlarge": true, "dax.r4.2xlarge": true, "dax.r4.4xlarge": true, "dax.r4.8xlarge": true, "dax.r4.16xlarge": true,
	"dax.r5.large": true, "dax.r5.xlarge": true, "dax.r5.2xlarge": true, "dax.r5.4xlarge": true, "dax.r5.8xlarge": true, "dax.r5.12xlarge": true, "dax.r5.16xlarge": true, "dax.r5.24xlarge": true,
}

var (
	daxIAMRolePattern  = regexp.MustCompile(`^arn:aws:iam::[0-9]{12}:role/[A-Za-z0-9+=,.@_/-]{1,512}$`)
	daxTopicPattern    = regexp.MustCompile(`^arn:aws:sns:[a-z0-9-]+:[0-9]{12}:[A-Za-z0-9_-]{1,256}$`)
	daxWindowPattern   = regexp.MustCompile(`^(mon|tue|wed|thu|fri|sat|sun):([01][0-9]|2[0-3]):[0-5][0-9]-(mon|tue|wed|thu|fri|sat|sun):([01][0-9]|2[0-3]):[0-5][0-9]$`)
	daxClusterARNRegex = regexp.MustCompile(`^arn:aws:dax:[a-z0-9-]+:[0-9]{12}:cache/(.+)$`)
)

type daxErr struct{ code, msg string }

func daxError(code, format string, args ...any) *daxErr {
	return &daxErr{code: code, msg: fmt.Sprintf(format, args...)}
}

func daxInvalid(format string, args ...any) *daxErr {
	return daxError("InvalidParameterValueException", format, args...)
}

type daxTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value,omitempty"`
}

type daxNode struct {
	id, az, status string
	created        time.Time
}

type daxCluster struct {
	name, description, arn, nodeType, status string
	nodes                                    []*daxNode
	removing                                 []string
	maintenance, topicARN, topicStatus       string
	subnetGroup                              string
	securityGroups                           []string
	iamRole, paramGroup, paramStatus         string
	sse                                      bool
	encryption, network, host                string
	tags                                     []daxTag
}

type daxStore struct {
	mu           sync.Mutex
	ec2          *ec2.Service
	clusters     map[string]*daxCluster
	paramGroups  map[string]*daxParamGroup
	subnetGroups map[string]*daxSubnetGroup
	events       []daxEvent
}

func newDAXStore(ec2Svc *ec2.Service) *daxStore {
	return &daxStore{
		ec2:      ec2Svc,
		clusters: map[string]*daxCluster{},
		paramGroups: map[string]*daxParamGroup{
			daxDefaultParamGroup: {name: daxDefaultParamGroup, description: "Default parameter group for dax1.0", values: map[string]string{}},
		},
		subnetGroups: map[string]*daxSubnetGroup{},
	}
}

func daxNow() time.Time { return time.Now().UTC() }

func daxEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// daxCheckName applies the DAX naming rule shared by clusters, parameter
// groups and subnet groups: a letter first, then letters, digits or single
// hyphens, not ending in a hyphen.
func daxCheckName(kind, name string, maxLen int) *daxErr {
	if name == "" || len(name) > maxLen {
		return daxInvalid("%s must be 1 to %d characters", kind, maxLen)
	}
	if c := name[0]; !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return daxInvalid("%s must begin with a letter", kind)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return daxInvalid("%s can contain only letters, digits and hyphens", kind)
		}
	}
	if strings.Contains(name, "--") || strings.HasSuffix(name, "-") {
		return daxInvalid("%s cannot end with a hyphen or contain two consecutive hyphens", kind)
	}
	return nil
}

func daxCheckDescription(d *string) *daxErr {
	if d != nil && len(*d) > daxMaxDescription {
		return daxInvalid("Description must be at most %d characters", daxMaxDescription)
	}
	return nil
}

func daxClusterARN(name string) string {
	return "arn:aws:dax:" + daxRegion + ":" + daxAccountID + ":cache/" + name
}

func daxRandomID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// clusterLocked returns a cluster by name; DAX stores cluster names in
// lowercase, so lookups are case-insensitive.
func (st *daxStore) clusterLocked(name string) (*daxCluster, *daxErr) {
	c, ok := st.clusters[strings.ToLower(name)]
	if !ok {
		return nil, daxError("ClusterNotFoundFault", "Cluster %s not found", name)
	}
	return c, nil
}

func (st *daxStore) requireAvailableLocked(name string) (*daxCluster, *daxErr) {
	c, err := st.clusterLocked(name)
	if err != nil {
		return nil, err
	}
	if c.status != "available" {
		return nil, daxError("InvalidClusterStateFault", "Cluster %s is %s, not available", c.name, c.status)
	}
	return c, nil
}

// settleLocked completes the cluster's pending transitions. It runs when the
// cluster is read (DescribeClusters), so a create or change reports its
// in-progress state once and is finished by the next describe.
func (c *daxCluster) settleLocked() {
	kept := c.nodes[:0]
	for _, n := range c.nodes {
		if n.status == "removing" {
			continue
		}
		n.status = "available"
		kept = append(kept, n)
	}
	c.nodes = kept
	c.removing = nil
	if c.status == "creating" || c.status == "modifying" {
		c.status = "available"
	}
	if c.paramStatus == "applying" {
		c.paramStatus = "in-sync"
	}
}

func (st *daxStore) totalNodesLocked() int {
	n := 0
	for _, c := range st.clusters {
		n += len(c.nodes)
	}
	return n
}

// nextNodeID returns the lowest unused "<cluster>-<letter>" node id.
func (c *daxCluster) nextNodeID() string {
	used := map[string]bool{}
	for _, n := range c.nodes {
		used[n.id] = true
	}
	for l := 'a'; l <= 'z'; l++ {
		if id := c.name + "-" + string(l); !used[id] {
			return id
		}
	}
	return c.name + "-" + daxRandomID()
}

// placeNodesLocked picks an AZ for each of count new nodes: the requested AZs if
// given (checked against the subnet group), else the least-used group AZ.
func (st *daxStore) placeNodesLocked(c *daxCluster, count int, requested []string) ([]string, *daxErr) {
	sg := st.subnetGroups[c.subnetGroup]
	allowed := map[string]bool{}
	var zones []string
	if sg != nil {
		for _, s := range sg.subnets {
			if !allowed[s.az] {
				allowed[s.az] = true
				zones = append(zones, s.az)
			}
		}
	}
	sort.Strings(zones)
	if len(requested) > 0 {
		if len(requested) != count {
			return nil, daxError("InvalidParameterCombinationException", "The number of AvailabilityZones (%d) must equal the number of nodes being added (%d)", len(requested), count)
		}
		for _, az := range requested {
			if !allowed[az] {
				return nil, daxInvalid("Availability Zone %s is not in subnet group %s", az, c.subnetGroup)
			}
		}
		return append([]string(nil), requested...), nil
	}
	if len(zones) == 0 {
		return nil, daxError("InvalidVPCNetworkStateFault", "Subnet group %s has no subnets", c.subnetGroup)
	}
	used := map[string]int{}
	for _, n := range c.nodes {
		if n.status != "removing" {
			used[n.az]++
		}
	}
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		best := zones[0]
		for _, z := range zones[1:] {
			if used[z] < used[best] {
				best = z
			}
		}
		used[best]++
		out = append(out, best)
	}
	return out, nil
}

func (c *daxCluster) addNodes(zones []string, now time.Time) {
	for _, az := range zones {
		c.nodes = append(c.nodes, &daxNode{id: c.nextNodeID(), az: az, status: "creating", created: now})
	}
}

func (c *daxCluster) liveNodes() int {
	n := 0
	for _, node := range c.nodes {
		if node.status != "removing" {
			n++
		}
	}
	return n
}

// checkSecurityGroupsLocked validates security group ids against the EC2
// store and the subnet group's VPC, defaulting to the VPC's default group.
// The EC2 store has its own lock and never calls back into DAX.
func (st *daxStore) checkSecurityGroupsLocked(ids []string, vpcID string) ([]string, *daxErr) {
	if len(ids) > daxMaxSecurityGroups {
		return nil, daxInvalid("At most %d security groups are allowed", daxMaxSecurityGroups)
	}
	if len(ids) == 0 {
		for _, g := range st.ec2.DescribeSecurityGroups(nil, []string{"default"}, vpcID) {
			return []string{g.ID}, nil
		}
		return nil, daxError("InvalidVPCNetworkStateFault", "VPC %s has no default security group", vpcID)
	}
	found := map[string]bool{}
	for _, g := range st.ec2.DescribeSecurityGroups(ids, nil, "") {
		if g.VpcID == vpcID {
			found[g.ID] = true
		}
	}
	for _, id := range ids {
		if !found[id] {
			return nil, daxInvalid("Security group %s does not exist in VPC %s", id, vpcID)
		}
	}
	return append([]string(nil), ids...), nil
}

func (st *daxStore) recordLocked(sourceName, sourceType, message string) {
	st.events = append(st.events, daxEvent{SourceName: sourceName, SourceType: sourceType, Message: message, Date: daxNow()})
	if len(st.events) > daxMaxEvents {
		st.events = append([]daxEvent(nil), st.events[len(st.events)-daxMaxEvents:]...)
	}
}

// ---- output shapes (built fresh under the lock; they never alias state) ----

type daxEndpointOut struct {
	Address string `json:"Address"`
	Port    int32  `json:"Port"`
	URL     string `json:"URL"`
}

type daxNodeOut struct {
	NodeID               string         `json:"NodeId"`
	Endpoint             daxEndpointOut `json:"Endpoint"`
	NodeCreateTime       float64        `json:"NodeCreateTime"`
	AvailabilityZone     string         `json:"AvailabilityZone"`
	NodeStatus           string         `json:"NodeStatus"`
	ParameterGroupStatus string         `json:"ParameterGroupStatus"`
}

type daxClusterOut struct {
	ClusterName              string          `json:"ClusterName"`
	Description              string          `json:"Description,omitempty"`
	ClusterArn               string          `json:"ClusterArn"`
	TotalNodes               int32           `json:"TotalNodes"`
	ActiveNodes              int32           `json:"ActiveNodes"`
	NodeType                 string          `json:"NodeType"`
	Status                   string          `json:"Status"`
	ClusterDiscoveryEndpoint daxEndpointOut  `json:"ClusterDiscoveryEndpoint"`
	NodeIdsToRemove          []string        `json:"NodeIdsToRemove,omitempty"`
	Nodes                    []daxNodeOut    `json:"Nodes"`
	PreferredMaintenance     string          `json:"PreferredMaintenanceWindow"`
	NotificationConfig       *daxNotifyOut   `json:"NotificationConfiguration,omitempty"`
	SubnetGroup              string          `json:"SubnetGroup"`
	SecurityGroups           []daxSGOut      `json:"SecurityGroups"`
	IamRoleArn               string          `json:"IamRoleArn"`
	ParameterGroup           daxParamStatOut `json:"ParameterGroup"`
	SSEDescription           daxSSEOut       `json:"SSEDescription"`
	EncryptionType           string          `json:"ClusterEndpointEncryptionType"`
	NetworkType              string          `json:"NetworkType"`
}

type daxNotifyOut struct {
	TopicArn    string `json:"TopicArn"`
	TopicStatus string `json:"TopicStatus"`
}

type daxSGOut struct {
	SecurityGroupIdentifier string `json:"SecurityGroupIdentifier"`
	Status                  string `json:"Status"`
}

type daxParamStatOut struct {
	ParameterGroupName   string   `json:"ParameterGroupName"`
	ParameterApplyStatus string   `json:"ParameterApplyStatus"`
	NodeIdsToReboot      []string `json:"NodeIdsToReboot"`
}

type daxSSEOut struct {
	Status string `json:"Status"`
}

func (c *daxCluster) endpoint(prefix string) daxEndpointOut {
	port, scheme := int32(daxPortPlain), "dax"
	if c.encryption == "TLS" {
		port, scheme = daxPortTLS, "daxs"
	}
	addr := prefix + c.host
	return daxEndpointOut{Address: addr, Port: port, URL: fmt.Sprintf("%s://%s:%d", scheme, addr, port)}
}

func (c *daxCluster) out() daxClusterOut {
	o := daxClusterOut{
		ClusterName: c.name, Description: c.description, ClusterArn: c.arn, NodeType: c.nodeType, Status: c.status,
		ClusterDiscoveryEndpoint: c.endpoint(""), PreferredMaintenance: c.maintenance, SubnetGroup: c.subnetGroup,
		IamRoleArn: c.iamRole, EncryptionType: c.encryption, NetworkType: c.network,
		NodeIdsToRemove: append([]string(nil), c.removing...),
		Nodes:           []daxNodeOut{}, SecurityGroups: []daxSGOut{},
		ParameterGroup: daxParamStatOut{ParameterGroupName: c.paramGroup, ParameterApplyStatus: c.paramStatus, NodeIdsToReboot: []string{}},
		SSEDescription: daxSSEOut{Status: "DISABLED"},
	}
	if c.sse {
		o.SSEDescription.Status = "ENABLED"
	}
	for _, n := range c.nodes {
		if n.status != "removing" {
			o.TotalNodes++
		}
		if n.status == "available" {
			o.ActiveNodes++
		}
		o.Nodes = append(o.Nodes, daxNodeOut{
			NodeID: n.id, Endpoint: c.endpoint(n.id + "."), NodeCreateTime: daxEpoch(n.created),
			AvailabilityZone: n.az, NodeStatus: n.status, ParameterGroupStatus: c.paramStatus,
		})
	}
	for _, g := range c.securityGroups {
		o.SecurityGroups = append(o.SecurityGroups, daxSGOut{SecurityGroupIdentifier: g, Status: "active"})
	}
	if c.topicARN != "" {
		o.NotificationConfig = &daxNotifyOut{TopicArn: c.topicARN, TopicStatus: c.topicStatus}
	}
	return o
}

// ---- cluster operations ----

type daxCreateClusterIn struct {
	ClusterName                string   `json:"ClusterName"`
	NodeType                   string   `json:"NodeType"`
	Description                *string  `json:"Description"`
	ReplicationFactor          int32    `json:"ReplicationFactor"`
	AvailabilityZones          []string `json:"AvailabilityZones"`
	SubnetGroupName            *string  `json:"SubnetGroupName"`
	SecurityGroupIds           []string `json:"SecurityGroupIds"`
	PreferredMaintenanceWindow *string  `json:"PreferredMaintenanceWindow"`
	NotificationTopicArn       *string  `json:"NotificationTopicArn"`
	IamRoleArn                 string   `json:"IamRoleArn"`
	ParameterGroupName         *string  `json:"ParameterGroupName"`
	Tags                       []daxTag `json:"Tags"`
	SSESpecification           *struct {
		Enabled *bool `json:"Enabled"`
	} `json:"SSESpecification"`
	ClusterEndpointEncryptionType *string `json:"ClusterEndpointEncryptionType"`
	NetworkType                   *string `json:"NetworkType"`
}

func (st *daxStore) createCluster(in daxCreateClusterIn, host string) (any, *daxErr) {
	if err := daxCheckName("ClusterName", in.ClusterName, daxClusterNameMaxLength); err != nil {
		return nil, err
	}
	if !daxNodeTypes[in.NodeType] {
		return nil, daxInvalid("NodeType %q is not a valid DAX node type", in.NodeType)
	}
	if in.ReplicationFactor < 1 {
		return nil, daxInvalid("ReplicationFactor must be at least 1")
	}
	if in.ReplicationFactor > daxMaxNodesPerCluster {
		return nil, daxError("NodeQuotaForClusterExceededFault", "A cluster can have at most %d nodes", daxMaxNodesPerCluster)
	}
	if !daxIAMRolePattern.MatchString(in.IamRoleArn) {
		return nil, daxInvalid("IamRoleArn %q is not a valid IAM role ARN", in.IamRoleArn)
	}
	if err := daxCheckDescription(in.Description); err != nil {
		return nil, err
	}
	window := daxDefaultMaintenance
	if in.PreferredMaintenanceWindow != nil {
		window = strings.ToLower(*in.PreferredMaintenanceWindow)
		if !daxWindowPattern.MatchString(window) {
			return nil, daxInvalid("PreferredMaintenanceWindow must have the format ddd:hh24:mi-ddd:hh24:mi")
		}
	}
	if in.NotificationTopicArn != nil && !daxTopicPattern.MatchString(*in.NotificationTopicArn) {
		return nil, daxInvalid("NotificationTopicArn %q is not a valid SNS topic ARN", *in.NotificationTopicArn)
	}
	encryption := daxDeref(in.ClusterEndpointEncryptionType, "NONE")
	if encryption != "NONE" && encryption != "TLS" {
		return nil, daxInvalid("ClusterEndpointEncryptionType must be NONE or TLS")
	}
	network := daxDeref(in.NetworkType, "ipv4")
	if network != "ipv4" && network != "ipv6" && network != "dual_stack" {
		return nil, daxInvalid("NetworkType must be ipv4, ipv6 or dual_stack")
	}
	if in.SSESpecification != nil && in.SSESpecification.Enabled == nil {
		return nil, daxInvalid("SSESpecification.Enabled is required")
	}
	if err := daxCheckTags(in.Tags); err != nil {
		return nil, err
	}
	name := strings.ToLower(in.ClusterName)

	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.clusters[name]; ok {
		return nil, daxError("ClusterAlreadyExistsFault", "Cluster %s already exists", name)
	}
	if len(st.clusters) >= daxMaxClusters {
		return nil, daxError("ClusterQuotaForCustomerExceededFault", "You have reached the maximum of %d clusters", daxMaxClusters)
	}
	if st.totalNodesLocked()+int(in.ReplicationFactor) > daxMaxNodesPerCustomer {
		return nil, daxError("NodeQuotaForCustomerExceededFault", "You have reached the maximum of %d nodes", daxMaxNodesPerCustomer)
	}
	pg, err := st.paramGroupLocked(daxDeref(in.ParameterGroupName, daxDefaultParamGroup))
	if err != nil {
		return nil, err
	}
	group, err := st.subnetGroupForClusterLocked(daxDeref(in.SubnetGroupName, daxDefaultSubnetGroup))
	if err != nil {
		return nil, err
	}
	if network != "ipv4" {
		return nil, daxError("InvalidParameterCombinationException", "Subnet group %s supports only the ipv4 network type", group.name)
	}
	groups, err := st.checkSecurityGroupsLocked(in.SecurityGroupIds, group.vpcID)
	if err != nil {
		return nil, err
	}
	c := &daxCluster{
		name: name, description: daxDeref(in.Description, ""), arn: daxClusterARN(name), nodeType: in.NodeType,
		status: "creating", maintenance: window, subnetGroup: group.name, securityGroups: groups,
		iamRole: in.IamRoleArn, paramGroup: pg.name, paramStatus: "in-sync",
		sse: in.SSESpecification != nil && *in.SSESpecification.Enabled, encryption: encryption, network: network,
		host: host, tags: append([]daxTag(nil), in.Tags...),
	}
	if in.NotificationTopicArn != nil {
		c.topicARN, c.topicStatus = *in.NotificationTopicArn, "active"
	}
	zones, err := st.placeNodesLocked(c, int(in.ReplicationFactor), in.AvailabilityZones)
	if err != nil {
		return nil, err
	}
	c.addNodes(zones, daxNow())
	st.clusters[name] = c
	st.recordLocked(name, "CLUSTER", "Cluster created")
	return map[string]any{"Cluster": c.out()}, nil
}

func daxDeref(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

type daxNamesPageIn struct {
	Names      []string `json:"ClusterNames"`
	MaxResults *int32   `json:"MaxResults"`
	NextToken  *string  `json:"NextToken"`
}

func (st *daxStore) describeClusters(in daxNamesPageIn) (any, *daxErr) {
	if len(in.Names) > daxMaxListInput {
		return nil, daxInvalid("At most %d cluster names are allowed", daxMaxListInput)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var picked []*daxCluster
	if len(in.Names) > 0 {
		for _, n := range in.Names {
			c, err := st.clusterLocked(n)
			if err != nil {
				return nil, err
			}
			picked = append(picked, c)
		}
	} else {
		for _, c := range st.clusters {
			picked = append(picked, c)
		}
		sort.Slice(picked, func(i, j int) bool { return picked[i].name < picked[j].name })
	}
	start, end, next, err := daxPage(len(picked), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	out := []daxClusterOut{}
	for _, c := range picked[start:end] {
		c.settleLocked()
		out = append(out, c.out())
	}
	return daxWithToken(map[string]any{"Clusters": out}, next), nil
}

type daxUpdateClusterIn struct {
	ClusterName                string   `json:"ClusterName"`
	Description                *string  `json:"Description"`
	PreferredMaintenanceWindow *string  `json:"PreferredMaintenanceWindow"`
	NotificationTopicArn       *string  `json:"NotificationTopicArn"`
	NotificationTopicStatus    *string  `json:"NotificationTopicStatus"`
	ParameterGroupName         *string  `json:"ParameterGroupName"`
	SecurityGroupIds           []string `json:"SecurityGroupIds"`
}

func (st *daxStore) updateCluster(in daxUpdateClusterIn) (any, *daxErr) {
	if err := daxCheckDescription(in.Description); err != nil {
		return nil, err
	}
	if in.PreferredMaintenanceWindow != nil && !daxWindowPattern.MatchString(strings.ToLower(*in.PreferredMaintenanceWindow)) {
		return nil, daxInvalid("PreferredMaintenanceWindow must have the format ddd:hh24:mi-ddd:hh24:mi")
	}
	if in.NotificationTopicArn != nil && *in.NotificationTopicArn != "" && !daxTopicPattern.MatchString(*in.NotificationTopicArn) {
		return nil, daxInvalid("NotificationTopicArn %q is not a valid SNS topic ARN", *in.NotificationTopicArn)
	}
	if in.NotificationTopicStatus != nil && *in.NotificationTopicStatus != "active" && *in.NotificationTopicStatus != "inactive" {
		return nil, daxInvalid("NotificationTopicStatus must be active or inactive")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.requireAvailableLocked(in.ClusterName)
	if err != nil {
		return nil, err
	}
	topic := c.topicARN
	if in.NotificationTopicArn != nil {
		topic = *in.NotificationTopicArn
	}
	if in.NotificationTopicStatus != nil && topic == "" {
		return nil, daxError("InvalidParameterCombinationException", "NotificationTopicStatus needs a notification topic")
	}
	newPG := c.paramGroup
	if in.ParameterGroupName != nil {
		pg, err := st.paramGroupLocked(*in.ParameterGroupName)
		if err != nil {
			return nil, err
		}
		newPG = pg.name
	}
	var groups []string
	if in.SecurityGroupIds != nil {
		vpc := ""
		if g := st.subnetGroups[c.subnetGroup]; g != nil {
			vpc = g.vpcID
		}
		if groups, err = st.checkSecurityGroupsLocked(in.SecurityGroupIds, vpc); err != nil {
			return nil, err
		}
	}
	// Everything is validated; apply.
	if in.Description != nil {
		c.description = *in.Description
	}
	if in.PreferredMaintenanceWindow != nil {
		c.maintenance = strings.ToLower(*in.PreferredMaintenanceWindow)
	}
	if in.NotificationTopicArn != nil {
		c.topicARN, c.topicStatus = topic, "active"
		if topic == "" {
			c.topicStatus = ""
		}
	}
	if in.NotificationTopicStatus != nil {
		c.topicStatus = *in.NotificationTopicStatus
	}
	if newPG != c.paramGroup {
		c.paramGroup, c.paramStatus = newPG, "applying"
		c.status = "modifying"
	}
	if groups != nil {
		c.securityGroups = groups
		c.status = "modifying"
	}
	st.recordLocked(c.name, "CLUSTER", "Cluster updated")
	return map[string]any{"Cluster": c.out()}, nil
}

type daxDeleteClusterIn struct {
	ClusterName string `json:"ClusterName"`
}

func (st *daxStore) deleteCluster(in daxDeleteClusterIn) (any, *daxErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(in.ClusterName)
	if err != nil {
		return nil, err
	}
	// ponytail: removed at once (AWS keeps it "deleting" for minutes); the
	// response still reports the deleting state.
	delete(st.clusters, c.name)
	c.status = "deleting"
	for _, n := range c.nodes {
		n.status = "deleting"
	}
	st.recordLocked(c.name, "CLUSTER", "Cluster deleted")
	return map[string]any{"Cluster": c.out()}, nil
}

type daxReplicationIn struct {
	ClusterName          string   `json:"ClusterName"`
	NewReplicationFactor int32    `json:"NewReplicationFactor"`
	AvailabilityZones    []string `json:"AvailabilityZones"`
	NodeIdsToRemove      []string `json:"NodeIdsToRemove"`
}

func (st *daxStore) increaseReplication(in daxReplicationIn) (any, *daxErr) {
	if len(in.AvailabilityZones) > daxMaxNodesPerCluster {
		return nil, daxInvalid("At most %d Availability Zones are allowed", daxMaxNodesPerCluster)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.requireAvailableLocked(in.ClusterName)
	if err != nil {
		return nil, err
	}
	current := c.liveNodes()
	if int(in.NewReplicationFactor) <= current {
		return nil, daxInvalid("NewReplicationFactor (%d) must be greater than the current replication factor (%d)", in.NewReplicationFactor, current)
	}
	if in.NewReplicationFactor > daxMaxNodesPerCluster {
		return nil, daxError("NodeQuotaForClusterExceededFault", "A cluster can have at most %d nodes", daxMaxNodesPerCluster)
	}
	add := int(in.NewReplicationFactor) - current
	if st.totalNodesLocked()+add > daxMaxNodesPerCustomer {
		return nil, daxError("NodeQuotaForCustomerExceededFault", "You have reached the maximum of %d nodes", daxMaxNodesPerCustomer)
	}
	zones, err := st.placeNodesLocked(c, add, in.AvailabilityZones)
	if err != nil {
		return nil, err
	}
	c.addNodes(zones, daxNow())
	c.status = "modifying"
	st.recordLocked(c.name, "CLUSTER", fmt.Sprintf("Increasing replication factor to %d", in.NewReplicationFactor))
	return map[string]any{"Cluster": c.out()}, nil
}

func (st *daxStore) decreaseReplication(in daxReplicationIn) (any, *daxErr) {
	if len(in.AvailabilityZones) > daxMaxNodesPerCluster || len(in.NodeIdsToRemove) > daxMaxNodesPerCluster {
		return nil, daxInvalid("At most %d Availability Zones or node ids are allowed", daxMaxNodesPerCluster)
	}
	if len(in.AvailabilityZones) > 0 && len(in.NodeIdsToRemove) > 0 {
		return nil, daxError("InvalidParameterCombinationException", "Specify AvailabilityZones or NodeIdsToRemove, not both")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.requireAvailableLocked(in.ClusterName)
	if err != nil {
		return nil, err
	}
	current := c.liveNodes()
	if in.NewReplicationFactor < 1 || int(in.NewReplicationFactor) >= current {
		return nil, daxInvalid("NewReplicationFactor must be at least 1 and less than the current replication factor (%d)", current)
	}
	remove := current - int(in.NewReplicationFactor)
	var victims []*daxNode
	switch {
	case len(in.NodeIdsToRemove) > 0:
		if len(in.NodeIdsToRemove) != remove {
			return nil, daxError("InvalidParameterCombinationException", "NodeIdsToRemove must name exactly %d nodes", remove)
		}
		seen := map[string]bool{}
		for _, id := range in.NodeIdsToRemove {
			n := c.node(id)
			if n == nil || seen[id] {
				return nil, daxError("NodeNotFoundFault", "Node %s not found in cluster %s", id, c.name)
			}
			seen[id] = true
			victims = append(victims, n)
		}
	case len(in.AvailabilityZones) > 0:
		if len(in.AvailabilityZones) != remove {
			return nil, daxError("InvalidParameterCombinationException", "AvailabilityZones must list exactly %d zones", remove)
		}
		taken := map[*daxNode]bool{}
		for _, az := range in.AvailabilityZones {
			var pick *daxNode
			for i := len(c.nodes) - 1; i >= 0; i-- {
				if n := c.nodes[i]; n.az == az && !taken[n] {
					pick = n
					break
				}
			}
			if pick == nil {
				return nil, daxInvalid("Cluster %s has no node left to remove in %s", c.name, az)
			}
			taken[pick] = true
			victims = append(victims, pick)
		}
	default:
		for i := len(c.nodes) - 1; i >= 0 && len(victims) < remove; i-- {
			victims = append(victims, c.nodes[i])
		}
	}
	for _, n := range victims {
		n.status = "removing"
		c.removing = append(c.removing, n.id)
	}
	c.status = "modifying"
	st.recordLocked(c.name, "CLUSTER", fmt.Sprintf("Decreasing replication factor to %d", in.NewReplicationFactor))
	return map[string]any{"Cluster": c.out()}, nil
}

func (c *daxCluster) node(id string) *daxNode {
	for _, n := range c.nodes {
		if n.id == id {
			return n
		}
	}
	return nil
}

type daxRebootIn struct {
	ClusterName string `json:"ClusterName"`
	NodeID      string `json:"NodeId"`
}

func (st *daxStore) rebootNode(in daxRebootIn) (any, *daxErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.requireAvailableLocked(in.ClusterName)
	if err != nil {
		return nil, err
	}
	n := c.node(in.NodeID)
	if n == nil {
		return nil, daxError("NodeNotFoundFault", "Node %s not found in cluster %s", in.NodeID, c.name)
	}
	if n.status != "available" {
		return nil, daxError("InvalidClusterStateFault", "Node %s is %s, not available", n.id, n.status)
	}
	n.status = "rebooting"
	st.recordLocked(c.name, "CLUSTER", "Rebooting node "+n.id)
	return map[string]any{"Cluster": c.out()}, nil
}
