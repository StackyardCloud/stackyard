package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

// Cloud9 environments. Each EC2 environment launches one instance in the EC2
// store (its subnet must exist there); DescribeEnvironmentStatus reads the
// instance's state back, so stopping it through EC2 shows as "stopped".
// Lock order: the EC2 store's calls are made with c9Store.mu released.

const (
	c9Account     = "123456789012"
	c9Region      = "us-east-1"
	c9MaxEnvs     = 100 // per-account environment quota
	c9MaxMembers  = 8   // members besides the owner
	c9MaxTags     = 50
	c9MaxTokens   = 1000
	c9CreateDelay = 2 * time.Second // "creating" before "ready"
	c9DeleteDelay = 2 * time.Second // "deleting" before the environment is gone
	c9RootOwner   = "arn:aws:iam::" + c9Account + ":root"
)

// c9Images maps the image aliases (and their SSM parameter paths) Cloud9
// accepts to the AMI the instance is launched from.
var c9Images = map[string]string{
	"amazonlinux-2-x86_64":    "ami-0c9a2b2e0f1a00001",
	"amazonlinux-2023-x86_64": "ami-0c9a2b2e0f1a00002",
	"ubuntu-18.04-x86_64":     "ami-0c9a2b2e0f1a00003",
	"ubuntu-22.04-x86_64":     "ami-0c9a2b2e0f1a00004",
}

var c9ArnRe = regexp.MustCompile(`^arn:aws:cloud9:` + c9Region + `:` + c9Account + `:environment:([a-zA-Z0-9]{8,32})$`)

type c9Member struct {
	userArn, userID, permissions string
}

type c9Env struct {
	id, name, description, arn, ownerArn  string
	connectionType, instanceType, imageID string
	subnetID, instanceID, managedCreds    string
	autoStop                              int
	created, deleting                     time.Time
	tags                                  map[string]string
	members                               map[string]*c9Member // by user ARN, owner included
	token                                 string
}

type c9Store struct {
	mu     sync.Mutex
	envs   map[string]*c9Env
	tokens map[string]string // clientRequestToken -> environment ID
}

func newCloud9Store() *c9Store {
	return &c9Store{envs: map[string]*c9Env{}, tokens: map[string]string{}}
}

// purge drops environments whose deletion has finished; callers hold st.mu.
func (st *c9Store) purge(now time.Time) {
	for id, e := range st.envs {
		if !e.deleting.IsZero() && now.Sub(e.deleting) >= c9DeleteDelay {
			delete(st.envs, id)
			if e.token != "" {
				delete(st.tokens, e.token)
			}
		}
	}
}

// live returns an environment that isn't being deleted; callers hold st.mu.
func (st *c9Store) live(id string) (*c9Env, *c9Err) {
	st.purge(time.Now())
	e := st.envs[id]
	if e == nil || !e.deleting.IsZero() {
		return nil, c9NotFound("Environment %s doesn't exist.", id)
	}
	return e, nil
}

func c9UserID(arn string) string {
	if arn == c9RootOwner {
		return c9Account
	}
	sum := sha256.Sum256([]byte(arn))
	return "AIDA" + strings.ToUpper(hex.EncodeToString(sum[:8]))
}

func c9NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *c9Env) member(m *c9Member) map[string]any {
	return map[string]any{"permissions": m.permissions, "userId": m.userID, "userArn": m.userArn, "environmentId": e.id}
}

func (e *c9Env) view(now time.Time) map[string]any {
	status := "CREATED"
	switch {
	case !e.deleting.IsZero():
		status = "DELETING"
	case now.Sub(e.created) < c9CreateDelay:
		status = "CREATING"
	}
	out := map[string]any{
		"id": e.id, "name": e.name, "type": "ec2", "connectionType": e.connectionType, "arn": e.arn,
		"ownerArn": e.ownerArn, "lifecycle": map[string]any{"status": status}, "managedCredentialsStatus": e.managedCreds,
	}
	if e.description != "" {
		out["description"] = e.description
	}
	return out
}

// ---- environments ----

func (s *Server) c9CreateEnvironment(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	image := strings.TrimPrefix(c9Str(in, "imageId"), "resolve:ssm:/aws/service/cloud9/amis/")
	ami, ok := c9Images[image]
	if !ok {
		return nil, c9BadRequest("imageId %q isn't a Cloud9 image alias or SSM path.", c9Str(in, "imageId"))
	}
	subnet := c9Str(in, "subnetId")
	if subnet != "" && len(s.ec2.DescribeSubnets([]string{subnet}, nil)) != 1 {
		return nil, c9NotFound("Subnet %s doesn't exist.", subnet)
	}
	owner := c9Str(in, "ownerArn")
	if owner == "" {
		owner = c9RootOwner
	}
	conn := c9Str(in, "connectionType")
	if conn == "" {
		conn = "CONNECT_SSH"
	}
	tags := map[string]string{}
	for _, t := range c9List(in, "tags") {
		tm, _ := t.(map[string]any)
		tags[c9Str(tm, "Key")] = c9Str(tm, "Value")
	}
	if err := c9CheckTags(tags); err != nil {
		return nil, err
	}
	name, token := c9Str(in, "name"), c9Str(in, "clientRequestToken")
	dryRun, _ := c9Bool(in, "dryRun")

	st := s.cloud9
	st.mu.Lock()
	st.purge(time.Now())
	if id, ok := st.tokens[token]; ok && token != "" {
		st.mu.Unlock()
		return map[string]any{"environmentId": id}, nil
	}
	for _, e := range st.envs {
		if e.name == name && e.ownerArn == owner && e.deleting.IsZero() {
			st.mu.Unlock()
			return nil, c9Conflict("An environment named %s already exists.", name)
		}
	}
	if len(st.envs) >= c9MaxEnvs {
		st.mu.Unlock()
		return nil, c9Limit("The account already has %d environments.", c9MaxEnvs)
	}
	if token != "" && len(st.tokens) >= c9MaxTokens {
		st.mu.Unlock()
		return nil, c9Limit("too many outstanding client request tokens")
	}
	if dryRun {
		st.mu.Unlock()
		return map[string]any{}, nil
	}
	id := c9NewID()
	e := &c9Env{
		id: id, name: name, description: c9Str(in, "description"), ownerArn: owner, connectionType: conn,
		arn: "arn:aws:cloud9:" + c9Region + ":" + c9Account + ":environment:" + id, instanceType: c9Str(in, "instanceType"),
		imageID: ami, subnetID: subnet, autoStop: c9Int(in, "automaticStopTimeMinutes", 30), created: time.Now().UTC(),
		managedCreds: "ENABLED_ON_CREATE", tags: tags, token: token,
		members: map[string]*c9Member{owner: {userArn: owner, userID: c9UserID(owner), permissions: "owner"}},
	}
	st.envs[id] = e
	if token != "" {
		st.tokens[token] = id
	}
	st.mu.Unlock()

	// The instance is launched with the store unlocked (EC2 takes its own lock).
	res, err := s.ec2.RunInstances(ami, e.instanceType, "", subnet, "", nil, 1, 1, []ec2.Tag{
		{Key: "Name", Value: "aws-cloud9-" + name + "-" + id}, {Key: "aws:cloud9:environment", Value: id},
	})
	st.mu.Lock()
	cur := st.envs[id]
	if err != nil || len(res.Instances) != 1 {
		if cur == e {
			delete(st.envs, id)
			delete(st.tokens, token)
		}
		st.mu.Unlock()
		return nil, c9BadRequest("The EC2 instance for environment %s couldn't be launched.", name)
	}
	iid := res.Instances[0].ID
	if cur == e {
		e.instanceID = iid
	}
	st.mu.Unlock()
	if cur != e { // deleted while launching
		_, _ = s.ec2.TerminateInstances([]string{iid})
	}
	return map[string]any{"environmentId": id}, nil
}

func (s *Server) c9DeleteEnvironment(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	e, err := st.live(c9Str(req.in, "environmentId"))
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	e.deleting = time.Now().UTC()
	iid := e.instanceID
	st.mu.Unlock()
	if iid != "" {
		_, _ = s.ec2.TerminateInstances([]string{iid})
	}
	return map[string]any{}, nil
}

func (s *Server) c9UpdateEnvironment(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.live(c9Str(in, "environmentId"))
	if err != nil {
		return nil, err
	}
	if name := c9Str(in, "name"); name != "" && name != e.name {
		for _, o := range st.envs {
			if o.name == name && o.ownerArn == e.ownerArn && o.deleting.IsZero() {
				return nil, c9Conflict("An environment named %s already exists.", name)
			}
		}
		e.name = name
	}
	if d, ok := in["description"].(string); ok {
		e.description = d
	}
	switch c9Str(in, "managedCredentialsAction") {
	case "ENABLE":
		e.managedCreds = "ENABLED_BY_OWNER"
	case "DISABLE":
		e.managedCreds = "DISABLED_BY_OWNER"
	}
	return map[string]any{}, nil
}

func (s *Server) c9DescribeEnvironments(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	st.purge(now)
	envs := []any{}
	for _, id := range c9Strings(req.in, "environmentIds") {
		if e := st.envs[id]; e != nil {
			envs = append(envs, e.view(now))
		}
	}
	return map[string]any{"environments": envs}, nil
}

func (s *Server) c9ListEnvironments(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	st.purge(time.Now())
	ids := make([]string, 0, len(st.envs))
	for id := range st.envs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := st.envs[ids[i]], st.envs[ids[j]]
		return a.created.Before(b.created) || a.created.Equal(b.created) && a.id < b.id
	})
	limit := c9Int(req.in, "maxResults", 25)
	if limit == 0 {
		limit = 25
	}
	start, end, next, err := c9Page(c9Str(req.in, "nextToken"), len(ids), limit, "BadRequestException")
	if err != nil {
		return nil, err
	}
	return map[string]any{"environmentIds": c9AnyStrings(ids[start:end]), "nextToken": next}, nil
}

// c9DescribeStatus reports creating/deleting from the environment and, once
// created, the state of its EC2 instance.
func (s *Server) c9DescribeStatus(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	now := time.Now()
	st.purge(now)
	e := st.envs[c9Str(req.in, "environmentId")]
	if e == nil {
		st.mu.Unlock()
		return nil, c9NotFound("Environment %s doesn't exist.", c9Str(req.in, "environmentId"))
	}
	deleting, creating, iid := !e.deleting.IsZero(), now.Sub(e.created) < c9CreateDelay, e.instanceID
	st.mu.Unlock()
	switch {
	case deleting:
		return map[string]any{"status": "deleting", "message": "Deleting environment"}, nil
	case creating || iid == "":
		return map[string]any{"status": "creating", "message": "Creating environment"}, nil
	}
	state := ""
	for _, r := range s.ec2.DescribeInstances([]string{iid}) {
		for _, inst := range r.Instances {
			state = inst.StateName
		}
	}
	switch state {
	case "running":
		return map[string]any{"status": "ready", "message": "Environment is ready to use"}, nil
	case "pending":
		return map[string]any{"status": "connecting", "message": "Connecting to the environment's instance"}, nil
	case "stopping", "shutting-down":
		return map[string]any{"status": "stopping", "message": "Environment is stopping"}, nil
	case "stopped":
		return map[string]any{"status": "stopped", "message": "Environment is stopped"}, nil
	}
	return map[string]any{"status": "error", "message": fmt.Sprintf("The environment's instance %s is %s", iid, strings.TrimSpace(state+" or missing"))}, nil
}

// ---- memberships ----

func (s *Server) c9CreateMembership(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.live(c9Str(in, "environmentId"))
	if err != nil {
		return nil, err
	}
	user := c9Str(in, "userArn")
	if e.members[user] != nil {
		return nil, c9Conflict("%s is already a member of environment %s.", user, e.id)
	}
	if len(e.members) > c9MaxMembers {
		return nil, c9Limit("Environment %s already has %d members.", e.id, c9MaxMembers)
	}
	m := &c9Member{userArn: user, userID: c9UserID(user), permissions: c9Str(in, "permissions")}
	e.members[user] = m
	return map[string]any{"membership": e.member(m)}, nil
}

func (s *Server) c9UpdateMembership(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, m, err := st.nonOwner(c9Str(in, "environmentId"), c9Str(in, "userArn"))
	if err != nil {
		return nil, err
	}
	m.permissions = c9Str(in, "permissions")
	return map[string]any{"membership": e.member(m)}, nil
}

func (s *Server) c9DeleteMembership(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, m, err := st.nonOwner(c9Str(req.in, "environmentId"), c9Str(req.in, "userArn"))
	if err != nil {
		return nil, err
	}
	delete(e.members, m.userArn)
	return map[string]any{}, nil
}

// nonOwner finds a member other than the owner; callers hold st.mu.
func (st *c9Store) nonOwner(envID, user string) (*c9Env, *c9Member, *c9Err) {
	e, err := st.live(envID)
	if err != nil {
		return nil, nil, err
	}
	m := e.members[user]
	if m == nil {
		return nil, nil, c9NotFound("%s isn't a member of environment %s.", user, envID)
	}
	if m.permissions == "owner" {
		return nil, nil, c9BadRequest("The owner's membership of environment %s can't be changed or removed.", envID)
	}
	return e, m, nil
}

func (s *Server) c9DescribeMemberships(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	st.purge(time.Now())
	envID, user := c9Str(in, "environmentId"), c9Str(in, "userArn")
	if envID != "" {
		if _, err := st.live(envID); err != nil {
			return nil, err
		}
	}
	perms := map[string]bool{}
	for _, p := range c9Strings(in, "permissions") {
		perms[p] = true
	}
	var items []map[string]any
	for _, e := range st.envs {
		if (envID != "" && e.id != envID) || !e.deleting.IsZero() {
			continue
		}
		for _, m := range e.members {
			if (user == "" || m.userArn == user) && (len(perms) == 0 || perms[m.permissions]) {
				items = append(items, e.member(m))
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a["environmentId"] != b["environmentId"] {
			return a["environmentId"].(string) < b["environmentId"].(string)
		}
		return a["userArn"].(string) < b["userArn"].(string)
	})
	limit := c9Int(in, "maxResults", 25)
	if limit == 0 {
		limit = 25
	}
	start, end, next, err := c9Page(c9Str(in, "nextToken"), len(items), limit, "BadRequestException")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, m := range items[start:end] {
		out = append(out, m)
	}
	return map[string]any{"memberships": out, "nextToken": next}, nil
}

// ---- tags ----

func c9CheckTags(tags map[string]string) *c9Err {
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return c9BadRequest("Tag keys starting with aws: are reserved.")
		}
	}
	if len(tags) > c9MaxTags {
		return c9BadRequest("A resource can have at most %d tags.", c9MaxTags)
	}
	return nil
}

// byArn resolves an environment ARN; callers hold st.mu.
func (st *c9Store) byArn(arn string) (*c9Env, *c9Err) {
	m := c9ArnRe.FindStringSubmatch(arn)
	if m == nil {
		return nil, c9NotFound("Resource %s doesn't exist.", arn)
	}
	return st.live(m[1])
}

func (s *Server) c9TagResource(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.byArn(c9Str(req.in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(e.tags))
	for k, v := range e.tags {
		next[k] = v
	}
	for _, t := range c9List(req.in, "Tags") {
		tm, _ := t.(map[string]any)
		next[c9Str(tm, "Key")] = c9Str(tm, "Value")
	}
	if err := c9CheckTags(next); err != nil {
		return nil, err
	}
	e.tags = next
	return map[string]any{}, nil
}

func (s *Server) c9UntagResource(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.byArn(c9Str(req.in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(e.tags))
	for k, v := range e.tags {
		next[k] = v
	}
	for _, k := range c9Strings(req.in, "TagKeys") {
		delete(next, k)
	}
	e.tags = next
	return map[string]any{}, nil
}

func (s *Server) c9ListTags(req c9Request) (map[string]any, *c9Err) {
	st := s.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.byArn(c9Str(req.in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Tags": c9TagList(e.tags, "Key", "Value")}, nil
}

// c9TagList renders a tag map as a sorted list of key/value objects.
func c9TagList(tags map[string]string, keyName, valueName string) []any {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{keyName: k, valueName: tags[k]})
	}
	return out
}
