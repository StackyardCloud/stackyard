package server

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	dsyMaxAgents    = 1000
	dsyMaxLocations = 2000
	dsyMaxTasks     = 1000
	// dsyMaxExecHistory bounds the finished executions kept per task; older
	// ones are dropped when a new execution starts.
	dsyMaxExecHistory = 100
)

type dsyStore struct {
	mu        sync.Mutex
	seq       int
	agents    map[string]*dsyAgent
	locations map[string]*dsyLocation
	tasks     map[string]*dsyTask
	execs     map[string]*dsyExec
	keysUsed  map[string]bool // activation keys already exchanged for an agent
}

func newDataSyncStore() *dsyStore {
	return &dsyStore{
		agents:    map[string]*dsyAgent{},
		locations: map[string]*dsyLocation{},
		tasks:     map[string]*dsyTask{},
		execs:     map[string]*dsyExec{},
		keysUsed:  map[string]bool{},
	}
}

func (st *dsyStore) next() int {
	st.seq++
	return st.seq
}

type dsyAgent struct {
	seq      int
	arn      string
	name     string
	created  time.Time
	vpce     string
	subnets  []string
	sgs      []string
	tags     map[string]string
	platform string
}

var (
	dsyActivationKeyPattern = regexp.MustCompile(`^[A-Z0-9]{5}(-[A-Z0-9]{5}){4}$`)
	dsySubnetARNPattern     = regexp.MustCompile(`^arn:aws[-a-z]*:ec2:[a-z0-9-]+:\d{12}:subnet/subnet-[0-9a-f]{8,17}$`)
	dsySGARNPattern         = regexp.MustCompile(`^arn:aws[-a-z]*:ec2:[a-z0-9-]+:\d{12}:security-group/sg-[0-9a-f]{8,17}$`)
	dsyVPCEPattern          = regexp.MustCompile(`^vpce-[0-9a-f]{17}$`)
)

func init() {
	dsyHandlers["CreateAgent"] = dsyCreateAgent
	dsyHandlers["DescribeAgent"] = dsyDescribeAgent
	dsyHandlers["UpdateAgent"] = dsyUpdateAgent
	dsyHandlers["DeleteAgent"] = dsyDeleteAgent
	dsyHandlers["ListAgents"] = dsyListAgents
	dsyHandlers["TagResource"] = dsyTagResource
	dsyHandlers["UntagResource"] = dsyUntagResource
	dsyHandlers["ListTagsForResource"] = dsyListTags
}

// dsyCheckARNs requires every value to match pattern.
func dsyCheckARNs(label string, values []string, pattern *regexp.Regexp) *dsyErr {
	for _, v := range values {
		if !pattern.MatchString(v) {
			return dsyInvalid("%s %q is not valid", label, v)
		}
	}
	return nil
}

func dsyCreateAgent(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	key := dsyStr(in, "ActivationKey")
	if !dsyActivationKeyPattern.MatchString(key) {
		return nil, dsyInvalid("ActivationKey %q is not a valid activation key", key)
	}
	if st.keysUsed[key] {
		return nil, dsyInvalid("activation key %s has already been used", key)
	}
	if len(st.agents) >= dsyMaxAgents {
		return nil, dsyInvalid("an account can have at most %d agents", dsyMaxAgents)
	}
	a := &dsyAgent{name: dsyStr(in, "AgentName"), vpce: dsyStr(in, "VpcEndpointId"), subnets: dsyStrings(in, "SubnetArns"), sgs: dsyStrings(in, "SecurityGroupArns"), tags: map[string]string{}, platform: "2"}
	if a.vpce != "" {
		if !dsyVPCEPattern.MatchString(a.vpce) {
			return nil, dsyInvalid("VpcEndpointId %q is not valid", a.vpce)
		}
		if len(a.subnets) == 0 || len(a.sgs) == 0 {
			return nil, dsyInvalid("a VPC endpoint agent needs SubnetArns and SecurityGroupArns")
		}
	} else if len(a.subnets) > 0 || len(a.sgs) > 0 {
		return nil, dsyInvalid("SubnetArns and SecurityGroupArns need a VpcEndpointId")
	}
	if err := dsyCheckARNs("SubnetArn", a.subnets, dsySubnetARNPattern); err != nil {
		return nil, err
	}
	if err := dsyCheckARNs("SecurityGroupArn", a.sgs, dsySGARNPattern); err != nil {
		return nil, err
	}
	if err := dsyTagsFrom(in, "Tags", a.tags); err != nil {
		return nil, err
	}
	a.seq, a.arn, a.created = st.next(), dsyARN("agent/"+dsyID("agent-")), time.Now().UTC()
	st.agents[a.arn] = a
	st.keysUsed[key] = true
	return map[string]any{"AgentArn": a.arn}, nil
}

func (st *dsyStore) agent(arn string) (*dsyAgent, *dsyErr) {
	if a, ok := st.agents[arn]; ok {
		return a, nil
	}
	return nil, dsyNotFound("Agent", arn)
}

func dsyDescribeAgent(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	a, err := s.datasync.agent(dsyStr(in, "AgentArn"))
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"AgentArn": a.arn, "Name": a.name, "Status": "ONLINE", "CreationTime": a.created,
		"LastConnectionTime": time.Now().UTC(), "EndpointType": "PUBLIC", "Platform": map[string]any{"Version": a.platform},
	}
	if a.vpce != "" {
		out["EndpointType"] = "PRIVATE_LINK"
		out["PrivateLinkConfig"] = map[string]any{
			"VpcEndpointId": a.vpce, "PrivateLinkEndpoint": "10.0.0.10",
			"SubnetArns": toAnySlice(a.subnets), "SecurityGroupArns": toAnySlice(a.sgs),
		}
	}
	return out, nil
}

func dsyUpdateAgent(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	a, err := s.datasync.agent(dsyStr(in, "AgentArn"))
	if err != nil {
		return nil, err
	}
	if _, ok := in["Name"]; ok {
		a.name = dsyStr(in, "Name")
	}
	return map[string]any{}, nil
}

func dsyDeleteAgent(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	a, err := st.agent(dsyStr(in, "AgentArn"))
	if err != nil {
		return nil, err
	}
	delete(st.agents, a.arn)
	return map[string]any{}, nil
}

func dsyListAgents(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	agents := make([]*dsyAgent, 0, len(s.datasync.agents))
	for _, a := range s.datasync.agents {
		agents = append(agents, a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].seq < agents[j].seq })
	start, end, next, err := dsyPage(in, len(agents))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, a := range agents[start:end] {
		list = append(list, map[string]any{"AgentArn": a.arn, "Name": a.name, "Status": "ONLINE", "Platform": map[string]any{"Version": a.platform}})
	}
	return map[string]any{"Agents": list, "NextToken": dsyOptional(next)}, nil
}

// dsyOptional turns "" into nil so the member is left out of the response.
func dsyOptional(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func toAnySlice(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// tagsOf finds the tag map of any DataSync resource by ARN.
func (st *dsyStore) tagsOf(arn string) (map[string]string, *dsyErr) {
	switch {
	case st.agents[arn] != nil:
		return st.agents[arn].tags, nil
	case st.locations[arn] != nil:
		return st.locations[arn].tags, nil
	case st.tasks[arn] != nil:
		return st.tasks[arn].tags, nil
	case st.execs[arn] != nil:
		return st.execs[arn].tags, nil
	}
	return nil, dsyNotFound("Resource", arn)
}

func dsyTagResource(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	tags, err := s.datasync.tagsOf(dsyStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	merged := make(map[string]string, len(tags))
	for k, v := range tags {
		merged[k] = v
	}
	if err := dsyTagsFrom(in, "Tags", merged); err != nil {
		return nil, err
	}
	for k, v := range merged {
		tags[k] = v
	}
	return map[string]any{}, nil
}

func dsyUntagResource(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	tags, err := s.datasync.tagsOf(dsyStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	for _, k := range dsyStrings(in, "Keys") {
		delete(tags, k)
	}
	return map[string]any{}, nil
}

func dsyListTags(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	tags, err := s.datasync.tagsOf(dsyStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	all := dsyTagList(tags)
	start, end, next, perr := dsyPage(in, len(all))
	if perr != nil {
		return nil, perr
	}
	return map[string]any{"Tags": all[start:end], "NextToken": dsyOptional(next)}, nil
}

// dsySubdir normalises a subdirectory to "/a/b/" form ("/" when empty).
func dsySubdir(sub string) string {
	sub = strings.Trim(sub, "/")
	if sub == "" {
		return "/"
	}
	return "/" + sub + "/"
}
