package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---- paging ----

// daxPage slices n items: MaxResults, when given, must be 20-100 as DAX
// requires; NextToken is the decimal offset this server handed out.
func daxPage(n int, max *int32, token *string) (start, end int, next string, err *daxErr) {
	size := 100
	if max != nil {
		if *max < 20 || *max > 100 {
			return 0, 0, "", daxInvalid("MaxResults must be between 20 and 100")
		}
		size = int(*max)
	}
	if token != nil && *token != "" {
		v, perr := strconv.ParseInt(*token, 10, 32)
		if perr != nil || v < 0 || int(v) > n {
			return 0, 0, "", daxInvalid("NextToken is not valid")
		}
		start = int(v)
	}
	end = min(start+size, n)
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func daxWithToken(out map[string]any, next string) map[string]any {
	if next != "" {
		out["NextToken"] = next
	}
	return out
}

// ---- parameter groups ----

type daxParamGroup struct {
	name, description string
	values            map[string]string // user-set values by parameter name
}

type daxParamDef struct{ name, description, def string }

// daxParamCatalog is the set of parameters a DAX parameter group holds.
var daxParamCatalog = []daxParamDef{
	{"query-ttl-millis", "Duration in milliseconds for queries to remain cached", "300000"},
	{"record-ttl-millis", "Duration in milliseconds for records to remain valid in cache (Default: 0 = infinite)", "300000"},
}

type daxParameterOut struct {
	ParameterName          string `json:"ParameterName"`
	ParameterType          string `json:"ParameterType"`
	ParameterValue         string `json:"ParameterValue"`
	NodeTypeSpecificValues []any  `json:"NodeTypeSpecificValues"`
	Description            string `json:"Description"`
	Source                 string `json:"Source"`
	DataType               string `json:"DataType"`
	AllowedValues          string `json:"AllowedValues"`
	IsModifiable           string `json:"IsModifiable"`
	ChangeType             string `json:"ChangeType"`
}

func daxParameters(values map[string]string, source string) []daxParameterOut {
	out := []daxParameterOut{}
	for _, p := range daxParamCatalog {
		value, src := p.def, "system"
		if v, ok := values[p.name]; ok {
			value, src = v, "user"
		}
		if source != "" && source != src {
			continue
		}
		out = append(out, daxParameterOut{
			ParameterName: p.name, ParameterType: "DEFAULT", ParameterValue: value, NodeTypeSpecificValues: []any{},
			Description: p.description, Source: src, DataType: "integer", AllowedValues: "0-",
			IsModifiable: "TRUE", ChangeType: "IMMEDIATE",
		})
	}
	return out
}

type daxParamGroupOut struct {
	ParameterGroupName string `json:"ParameterGroupName"`
	Description        string `json:"Description,omitempty"`
}

func (g *daxParamGroup) out() daxParamGroupOut {
	return daxParamGroupOut{ParameterGroupName: g.name, Description: g.description}
}

type daxCreateParamGroupIn struct {
	ParameterGroupName string  `json:"ParameterGroupName"`
	Description        *string `json:"Description"`
}

func (st *daxStore) createParamGroup(in daxCreateParamGroupIn) (any, *daxErr) {
	if err := daxCheckName("ParameterGroupName", in.ParameterGroupName, 255); err != nil {
		return nil, err
	}
	if err := daxCheckDescription(in.Description); err != nil {
		return nil, err
	}
	name := strings.ToLower(in.ParameterGroupName)
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.paramGroups[name]; ok {
		return nil, daxError("ParameterGroupAlreadyExistsFault", "Parameter group %s already exists", name)
	}
	if len(st.paramGroups) >= daxMaxParamGroups {
		return nil, daxError("ParameterGroupQuotaExceededFault", "You have reached the maximum of %d parameter groups", daxMaxParamGroups)
	}
	g := &daxParamGroup{name: name, description: daxDeref(in.Description, ""), values: map[string]string{}}
	st.paramGroups[name] = g
	st.recordLocked(name, "PARAMETER_GROUP", "Parameter group created")
	return map[string]any{"ParameterGroup": g.out()}, nil
}

func (st *daxStore) paramGroupLocked(name string) (*daxParamGroup, *daxErr) {
	g, ok := st.paramGroups[strings.ToLower(name)]
	if !ok {
		return nil, daxError("ParameterGroupNotFoundFault", "Parameter group %s not found", name)
	}
	return g, nil
}

type daxNameIn struct {
	ParameterGroupName string `json:"ParameterGroupName"`
	SubnetGroupName    string `json:"SubnetGroupName"`
}

func (st *daxStore) deleteParamGroup(in daxNameIn) (any, *daxErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.paramGroupLocked(in.ParameterGroupName)
	if err != nil {
		return nil, err
	}
	if g.name == daxDefaultParamGroup {
		return nil, daxError("InvalidParameterGroupStateFault", "The default parameter group cannot be deleted")
	}
	for _, c := range st.clusters {
		if c.paramGroup == g.name {
			return nil, daxError("InvalidParameterGroupStateFault", "Parameter group %s is in use by cluster %s", g.name, c.name)
		}
	}
	delete(st.paramGroups, g.name)
	st.recordLocked(g.name, "PARAMETER_GROUP", "Parameter group deleted")
	return map[string]any{"DeletionMessage": "Parameter group " + g.name + " has been deleted."}, nil
}

type daxParamGroupsPageIn struct {
	Names      []string `json:"ParameterGroupNames"`
	MaxResults *int32   `json:"MaxResults"`
	NextToken  *string  `json:"NextToken"`
}

func (st *daxStore) describeParamGroups(in daxParamGroupsPageIn) (any, *daxErr) {
	if len(in.Names) > daxMaxListInput {
		return nil, daxInvalid("At most %d parameter group names are allowed", daxMaxListInput)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var picked []*daxParamGroup
	if len(in.Names) > 0 {
		for _, n := range in.Names {
			g, err := st.paramGroupLocked(n)
			if err != nil {
				return nil, err
			}
			picked = append(picked, g)
		}
	} else {
		for _, g := range st.paramGroups {
			picked = append(picked, g)
		}
		sort.Slice(picked, func(i, j int) bool { return picked[i].name < picked[j].name })
	}
	start, end, next, err := daxPage(len(picked), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	out := []daxParamGroupOut{}
	for _, g := range picked[start:end] {
		out = append(out, g.out())
	}
	return daxWithToken(map[string]any{"ParameterGroups": out}, next), nil
}

type daxDescribeParamsIn struct {
	ParameterGroupName string  `json:"ParameterGroupName"`
	Source             *string `json:"Source"`
	MaxResults         *int32  `json:"MaxResults"`
	NextToken          *string `json:"NextToken"`
}

func (st *daxStore) describeParameters(in daxDescribeParamsIn) (any, *daxErr) {
	source := daxDeref(in.Source, "")
	if source != "" && source != "user" && source != "system" {
		return nil, daxInvalid("Source must be user or system")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.paramGroupLocked(in.ParameterGroupName)
	if err != nil {
		return nil, err
	}
	params := daxParameters(g.values, source)
	start, end, next, err := daxPage(len(params), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	return daxWithToken(map[string]any{"Parameters": params[start:end]}, next), nil
}

type daxPageIn struct {
	MaxResults *int32  `json:"MaxResults"`
	NextToken  *string `json:"NextToken"`
}

func (st *daxStore) describeDefaultParameters(in daxPageIn) (any, *daxErr) {
	params := daxParameters(nil, "")
	start, end, next, err := daxPage(len(params), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	return daxWithToken(map[string]any{"Parameters": params[start:end]}, next), nil
}

type daxUpdateParamGroupIn struct {
	ParameterGroupName  string `json:"ParameterGroupName"`
	ParameterNameValues []struct {
		ParameterName  *string `json:"ParameterName"`
		ParameterValue *string `json:"ParameterValue"`
	} `json:"ParameterNameValues"`
}

func (st *daxStore) updateParamGroup(in daxUpdateParamGroupIn) (any, *daxErr) {
	if len(in.ParameterNameValues) == 0 || len(in.ParameterNameValues) > len(daxParamCatalog) {
		return nil, daxInvalid("ParameterNameValues must list 1 to %d parameters", len(daxParamCatalog))
	}
	known := map[string]bool{}
	for _, p := range daxParamCatalog {
		known[p.name] = true
	}
	set := map[string]string{}
	for _, nv := range in.ParameterNameValues {
		name, value := daxDeref(nv.ParameterName, ""), daxDeref(nv.ParameterValue, "")
		if !known[name] {
			return nil, daxInvalid("Unknown parameter %q", name)
		}
		if v, err := strconv.ParseInt(value, 10, 64); err != nil || v < 0 {
			return nil, daxInvalid("Parameter %s must be a non-negative integer", name)
		}
		set[name] = value
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.paramGroupLocked(in.ParameterGroupName)
	if err != nil {
		return nil, err
	}
	if g.name == daxDefaultParamGroup {
		return nil, daxError("InvalidParameterGroupStateFault", "The default parameter group cannot be modified")
	}
	for k, v := range set {
		g.values[k] = v
	}
	for _, c := range st.clusters {
		if c.paramGroup == g.name {
			c.paramStatus = "applying"
		}
	}
	st.recordLocked(g.name, "PARAMETER_GROUP", "Parameter group updated")
	return map[string]any{"ParameterGroup": g.out()}, nil
}

// ---- subnet groups ----

type daxSubnet struct{ id, az string }

type daxSubnetGroup struct {
	name, description, vpcID string
	subnets                  []daxSubnet
}

type daxSubnetOut struct {
	SubnetIdentifier       string   `json:"SubnetIdentifier"`
	SubnetAvailabilityZone string   `json:"SubnetAvailabilityZone"`
	SupportedNetworkTypes  []string `json:"SupportedNetworkTypes"`
}

type daxSubnetGroupOut struct {
	SubnetGroupName       string         `json:"SubnetGroupName"`
	Description           string         `json:"Description,omitempty"`
	VpcID                 string         `json:"VpcId"`
	Subnets               []daxSubnetOut `json:"Subnets"`
	SupportedNetworkTypes []string       `json:"SupportedNetworkTypes"`
}

func (g *daxSubnetGroup) out() daxSubnetGroupOut {
	o := daxSubnetGroupOut{SubnetGroupName: g.name, Description: g.description, VpcID: g.vpcID, Subnets: []daxSubnetOut{}, SupportedNetworkTypes: []string{"ipv4"}}
	for _, s := range g.subnets {
		o.Subnets = append(o.Subnets, daxSubnetOut{SubnetIdentifier: s.id, SubnetAvailabilityZone: s.az, SupportedNetworkTypes: []string{"ipv4"}})
	}
	return o
}

// resolveSubnetsLocked checks subnet ids against the EC2 store: they must
// exist and share one VPC.
func (st *daxStore) resolveSubnetsLocked(ids []string) ([]daxSubnet, string, *daxErr) {
	if len(ids) == 0 {
		return nil, "", daxInvalid("SubnetIds must list at least one subnet")
	}
	if len(ids) > daxMaxSubnetsPerGroup {
		return nil, "", daxError("SubnetQuotaExceededFault", "A subnet group can have at most %d subnets", daxMaxSubnetsPerGroup)
	}
	byID := map[string]daxSubnet{}
	vpcOf := map[string]string{}
	for _, s := range st.ec2.DescribeSubnets(ids, nil) {
		byID[s.ID] = daxSubnet{id: s.ID, az: s.AvailabilityZone}
		vpcOf[s.ID] = s.VpcID
	}
	var out []daxSubnet
	vpc := ""
	seen := map[string]bool{}
	for _, id := range ids {
		s, ok := byID[id]
		if !ok {
			return nil, "", daxError("InvalidSubnet", "Subnet %s does not exist", id)
		}
		if vpc == "" {
			vpc = vpcOf[id]
		} else if vpcOf[id] != vpc {
			return nil, "", daxError("InvalidSubnet", "All subnets must be in the same VPC; %s is in %s, not %s", id, vpcOf[id], vpc)
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, s)
		}
	}
	return out, vpc, nil
}

// subnetGroupForClusterLocked returns the named group, creating the
// "default" group from the EC2 default VPC's subnets on first use.
func (st *daxStore) subnetGroupForClusterLocked(name string) (*daxSubnetGroup, *daxErr) {
	if g, ok := st.subnetGroups[strings.ToLower(name)]; ok {
		return g, nil
	}
	if strings.ToLower(name) != daxDefaultSubnetGroup {
		return nil, daxError("SubnetGroupNotFoundFault", "Subnet group %s not found", name)
	}
	var ids []string
	for _, s := range st.ec2.DescribeSubnets(nil, []string{"vpc-00000001"}) {
		ids = append(ids, s.ID)
		if len(ids) == daxMaxSubnetsPerGroup {
			break
		}
	}
	subnets, vpc, err := st.resolveSubnetsLocked(ids)
	if err != nil {
		return nil, daxError("InvalidVPCNetworkStateFault", "The default VPC has no subnets for the default subnet group")
	}
	g := &daxSubnetGroup{name: daxDefaultSubnetGroup, description: "Default subnet group", vpcID: vpc, subnets: subnets}
	st.subnetGroups[g.name] = g
	return g, nil
}

type daxSubnetGroupIn struct {
	SubnetGroupName string   `json:"SubnetGroupName"`
	Description     *string  `json:"Description"`
	SubnetIds       []string `json:"SubnetIds"`
}

func (st *daxStore) createSubnetGroup(in daxSubnetGroupIn) (any, *daxErr) {
	if err := daxCheckName("SubnetGroupName", in.SubnetGroupName, 255); err != nil {
		return nil, err
	}
	if err := daxCheckDescription(in.Description); err != nil {
		return nil, err
	}
	name := strings.ToLower(in.SubnetGroupName)
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.subnetGroups[name]; ok {
		return nil, daxError("SubnetGroupAlreadyExistsFault", "Subnet group %s already exists", name)
	}
	if len(st.subnetGroups) >= daxMaxSubnetGroups {
		return nil, daxError("SubnetGroupQuotaExceededFault", "You have reached the maximum of %d subnet groups", daxMaxSubnetGroups)
	}
	subnets, vpc, err := st.resolveSubnetsLocked(in.SubnetIds)
	if err != nil {
		return nil, err
	}
	g := &daxSubnetGroup{name: name, description: daxDeref(in.Description, ""), vpcID: vpc, subnets: subnets}
	st.subnetGroups[name] = g
	st.recordLocked(name, "SUBNET_GROUP", "Subnet group created")
	return map[string]any{"SubnetGroup": g.out()}, nil
}

func (st *daxStore) subnetGroupLocked(name string) (*daxSubnetGroup, *daxErr) {
	g, ok := st.subnetGroups[strings.ToLower(name)]
	if !ok {
		return nil, daxError("SubnetGroupNotFoundFault", "Subnet group %s not found", name)
	}
	return g, nil
}

func (st *daxStore) updateSubnetGroup(in daxSubnetGroupIn) (any, *daxErr) {
	if err := daxCheckDescription(in.Description); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.subnetGroupLocked(in.SubnetGroupName)
	if err != nil {
		return nil, err
	}
	subnets, vpc := g.subnets, g.vpcID
	if in.SubnetIds != nil {
		if subnets, vpc, err = st.resolveSubnetsLocked(in.SubnetIds); err != nil {
			return nil, err
		}
		// A subnet whose AZ still hosts a cluster node can't be dropped, and
		// clusters can't move VPC.
		zones := map[string]bool{}
		for _, s := range subnets {
			zones[s.az] = true
		}
		for _, c := range st.clusters {
			if c.subnetGroup != g.name {
				continue
			}
			if vpc != g.vpcID {
				return nil, daxError("SubnetInUse", "Subnet group %s is used by cluster %s in VPC %s", g.name, c.name, g.vpcID)
			}
			for _, n := range c.nodes {
				if !zones[n.az] {
					return nil, daxError("SubnetInUse", "Cluster %s has node %s in %s, which the new subnets don't cover", c.name, n.id, n.az)
				}
			}
		}
	}
	if in.Description != nil {
		g.description = *in.Description
	}
	g.subnets, g.vpcID = subnets, vpc
	st.recordLocked(g.name, "SUBNET_GROUP", "Subnet group updated")
	return map[string]any{"SubnetGroup": g.out()}, nil
}

func (st *daxStore) deleteSubnetGroup(in daxNameIn) (any, *daxErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.subnetGroupLocked(in.SubnetGroupName)
	if err != nil {
		return nil, err
	}
	for _, c := range st.clusters {
		if c.subnetGroup == g.name {
			return nil, daxError("SubnetGroupInUseFault", "Subnet group %s is in use by cluster %s", g.name, c.name)
		}
	}
	delete(st.subnetGroups, g.name)
	st.recordLocked(g.name, "SUBNET_GROUP", "Subnet group deleted")
	return map[string]any{"DeletionMessage": "Subnet group " + g.name + " has been deleted."}, nil
}

type daxSubnetGroupsPageIn struct {
	Names      []string `json:"SubnetGroupNames"`
	MaxResults *int32   `json:"MaxResults"`
	NextToken  *string  `json:"NextToken"`
}

func (st *daxStore) describeSubnetGroups(in daxSubnetGroupsPageIn) (any, *daxErr) {
	if len(in.Names) > daxMaxListInput {
		return nil, daxInvalid("At most %d subnet group names are allowed", daxMaxListInput)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var picked []*daxSubnetGroup
	if len(in.Names) > 0 {
		for _, n := range in.Names {
			g, err := st.subnetGroupLocked(n)
			if err != nil {
				return nil, err
			}
			picked = append(picked, g)
		}
	} else {
		for _, g := range st.subnetGroups {
			picked = append(picked, g)
		}
		sort.Slice(picked, func(i, j int) bool { return picked[i].name < picked[j].name })
	}
	start, end, next, err := daxPage(len(picked), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	out := []daxSubnetGroupOut{}
	for _, g := range picked[start:end] {
		out = append(out, g.out())
	}
	return daxWithToken(map[string]any{"SubnetGroups": out}, next), nil
}

// ---- events ----

type daxEvent struct {
	SourceName, SourceType, Message string
	Date                            time.Time
}

type daxEventsIn struct {
	SourceName *string  `json:"SourceName"`
	SourceType *string  `json:"SourceType"`
	StartTime  *float64 `json:"StartTime"`
	EndTime    *float64 `json:"EndTime"`
	Duration   *int32   `json:"Duration"`
	MaxResults *int32   `json:"MaxResults"`
	NextToken  *string  `json:"NextToken"`
}

func daxFromEpoch(v float64) time.Time {
	return time.UnixMilli(int64(v * 1000)).UTC()
}

func (st *daxStore) describeEvents(in daxEventsIn) (any, *daxErr) {
	if t := daxDeref(in.SourceType, ""); t != "" && t != "CLUSTER" && t != "PARAMETER_GROUP" && t != "SUBNET_GROUP" {
		return nil, daxInvalid("SourceType must be CLUSTER, PARAMETER_GROUP or SUBNET_GROUP")
	}
	if in.Duration != nil && (*in.Duration < 0 || *in.Duration > 20160) {
		return nil, daxInvalid("Duration must be between 0 and 20160 minutes")
	}
	for _, v := range []*float64{in.StartTime, in.EndTime} {
		if v != nil && (*v < 0 || *v > 1e11) {
			return nil, daxInvalid("StartTime and EndTime must be valid timestamps")
		}
	}
	now := daxNow()
	end := now
	if in.EndTime != nil {
		end = daxFromEpoch(*in.EndTime)
	}
	// AWS returns the last hour by default.
	start := end.Add(-time.Hour)
	if in.Duration != nil {
		start = end.Add(-time.Duration(*in.Duration) * time.Minute)
	}
	if in.StartTime != nil {
		start = daxFromEpoch(*in.StartTime)
	}
	if start.After(end) {
		return nil, daxError("InvalidParameterCombinationException", "StartTime must be before EndTime")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	type eventOut struct {
		SourceName string  `json:"SourceName"`
		SourceType string  `json:"SourceType"`
		Message    string  `json:"Message"`
		Date       float64 `json:"Date"`
	}
	var matched []eventOut
	for i := len(st.events) - 1; i >= 0; i-- { // newest first
		e := st.events[i]
		if e.Date.Before(start) || e.Date.After(end) {
			continue
		}
		if in.SourceName != nil && *in.SourceName != e.SourceName {
			continue
		}
		if in.SourceType != nil && *in.SourceType != e.SourceType {
			continue
		}
		matched = append(matched, eventOut{e.SourceName, e.SourceType, e.Message, daxEpoch(e.Date)})
	}
	pstart, pend, next, err := daxPage(len(matched), in.MaxResults, in.NextToken)
	if err != nil {
		return nil, err
	}
	out := append([]eventOut{}, matched[pstart:pend]...)
	return daxWithToken(map[string]any{"Events": out}, next), nil
}

// ---- tags (clusters only, as in DAX) ----

func daxCheckTags(tags []daxTag) *daxErr {
	if len(tags) > daxMaxTags {
		return daxError("TagQuotaPerResourceExceeded", "A resource can have at most %d tags", daxMaxTags)
	}
	for _, t := range tags {
		if t.Key == "" || len(t.Key) > 128 || len(t.Value) > 256 {
			return daxInvalid("Tag keys must be 1-128 characters and values at most 256")
		}
		if strings.HasPrefix(strings.ToLower(t.Key), "aws:") {
			return daxInvalid("Tag keys cannot start with aws:")
		}
	}
	return nil
}

func (st *daxStore) taggedClusterLocked(arn string) (*daxCluster, *daxErr) {
	m := daxClusterARNRegex.FindStringSubmatch(arn)
	if m == nil {
		return nil, daxError("InvalidARNFault", "%q is not a valid DAX cluster ARN", arn)
	}
	return st.clusterLocked(m[1])
}

func daxTagsOut(tags []daxTag) []daxTag {
	return append([]daxTag{}, tags...)
}

type daxTagIn struct {
	ResourceName string   `json:"ResourceName"`
	Tags         []daxTag `json:"Tags"`
	TagKeys      []string `json:"TagKeys"`
	NextToken    *string  `json:"NextToken"`
}

func (st *daxStore) tagResource(in daxTagIn) (any, *daxErr) {
	if err := daxCheckTags(in.Tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.taggedClusterLocked(in.ResourceName)
	if err != nil {
		return nil, err
	}
	merged := append([]daxTag(nil), c.tags...)
	for _, t := range in.Tags {
		replaced := false
		for i := range merged {
			if merged[i].Key == t.Key {
				merged[i].Value, replaced = t.Value, true
			}
		}
		if !replaced {
			merged = append(merged, t)
		}
	}
	if len(merged) > daxMaxTags {
		return nil, daxError("TagQuotaPerResourceExceeded", "A resource can have at most %d tags", daxMaxTags)
	}
	c.tags = merged
	return map[string]any{"Tags": daxTagsOut(c.tags)}, nil
}

func (st *daxStore) untagResource(in daxTagIn) (any, *daxErr) {
	if len(in.TagKeys) == 0 || len(in.TagKeys) > daxMaxTags {
		return nil, daxInvalid("TagKeys must list 1 to %d keys", daxMaxTags)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.taggedClusterLocked(in.ResourceName)
	if err != nil {
		return nil, err
	}
	drop := map[string]bool{}
	for _, k := range in.TagKeys {
		found := false
		for _, t := range c.tags {
			found = found || t.Key == k
		}
		if !found {
			return nil, daxError("TagNotFoundFault", "Tag %s not found on %s", k, c.arn)
		}
		drop[k] = true
	}
	var kept []daxTag
	for _, t := range c.tags {
		if !drop[t.Key] {
			kept = append(kept, t)
		}
	}
	c.tags = kept
	return map[string]any{"Tags": daxTagsOut(c.tags)}, nil
}

func (st *daxStore) listTags(in daxTagIn) (any, *daxErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.taggedClusterLocked(in.ResourceName)
	if err != nil {
		return nil, err
	}
	// A cluster holds at most 50 tags, so one page always covers them.
	return map[string]any{"Tags": daxTagsOut(c.tags)}, nil
}
