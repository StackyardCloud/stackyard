package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	mstAccount = "123456789012"
	mstRegion  = "us-east-1"
	// ponytail: caps keep the store bounded; set above AWS quotas (100
	// containers) so repeated coverage runs against one server don't hit them.
	mstMaxContainers = 500
	mstMaxTags       = 50
	mstMaxLifecycle  = 10
)

type mstErr struct {
	status    int
	code, msg string
}

func (e *mstErr) Error() string { return e.code + ": " + e.msg }

func mstInvalid(format string, args ...any) *mstErr {
	return &mstErr{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func mstNoContainer(name string) *mstErr {
	return &mstErr{http.StatusBadRequest, "ContainerNotFoundException", fmt.Sprintf("Container %s does not exist", name)}
}

type mstTag struct{ key, value string }

type mstContainer struct {
	name, arn, endpointID string
	created               time.Time
	status                string
	logging               bool
	policy                string
	lifecycle             *string
	cors                  []any          // conformed copy; replaced, never mutated
	metric                map[string]any // conformed copy; replaced, never mutated
	tags                  []mstTag
	objects               map[string]*mstdObject // MediaStore Data objects by path
}

type mstStore struct {
	mu         sync.Mutex
	containers map[string]*mstContainer // by name
	endpoints  map[string]string        // endpoint id -> container name
	// MediaStore Data totals across every container (see mediastoredata.go).
	storedBytes   int64
	storedObjects int
}

func newMediaStoreStore() *mstStore {
	return &mstStore{containers: map[string]*mstContainer{}, endpoints: map[string]string{}}
}

func mstOut(op string, out map[string]any) map[string]any {
	m := awsmodels.MediaStoreModel()
	if shaped, ok := m.Conform(m.Operations[op].Output, out).(map[string]any); ok {
		return shaped
	}
	return map[string]any{}
}

// mstEndpointID is a random host label like the one in a real container
// endpoint (https://<id>.data.mediastore.<region>.amazonaws.com).
func mstEndpointID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 14)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// mstEndpoint is the URL clients send MediaStore Data requests to: a path on
// this server that names the container's endpoint id.
func mstEndpoint(base, id string) string {
	return strings.TrimRight(base, "/") + mstdPathPrefix + id
}

// settle moves a new container to ACTIVE: the create response reports
// CREATING, every later read sees ACTIVE. ponytail: no timers.
func (c *mstContainer) settle() {
	if c.status == "CREATING" {
		c.status = "ACTIVE"
	}
}

func (c *mstContainer) out(base string) map[string]any {
	m := map[string]any{
		"Name": c.name, "ARN": c.arn, "Status": c.status, "CreationTime": float64(c.created.UnixMilli()) / 1000,
		"AccessLoggingEnabled": c.logging,
	}
	if c.status == "ACTIVE" {
		m["Endpoint"] = mstEndpoint(base, c.endpointID)
	}
	return m
}

// get returns an existing container, settled. Callers hold st.mu.
func (st *mstStore) get(name string) (*mstContainer, *mstErr) {
	c := st.containers[name]
	if c == nil {
		return nil, mstNoContainer(name)
	}
	c.settle()
	return c, nil
}

func (st *mstStore) createContainer(name string, tags []mstTag) (map[string]any, *mstErr) {
	if err := mstCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.containers[name] != nil {
		return nil, &mstErr{http.StatusBadRequest, "ContainerInUseException", fmt.Sprintf("Container %s already exists", name)}
	}
	if len(st.containers) >= mstMaxContainers {
		return nil, &mstErr{http.StatusBadRequest, "LimitExceededException", fmt.Sprintf("An account can have at most %d containers", mstMaxContainers)}
	}
	id := mstEndpointID()
	for st.endpoints[id] != "" {
		id = mstEndpointID()
	}
	c := &mstContainer{
		name: name, arn: "arn:aws:mediastore:" + mstRegion + ":" + mstAccount + ":container/" + name,
		endpointID: id, created: time.Now().UTC(), status: "CREATING", tags: mstMergeTags(nil, tags),
		objects: map[string]*mstdObject{},
	}
	st.containers[name], st.endpoints[id] = c, name
	return mstOut("CreateContainer", map[string]any{"Container": c.out("")}), nil
}

func (st *mstStore) describeContainer(name, base string) (map[string]any, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if name == "" {
		// The name is optional; without one AWS describes the first container.
		names := st.sortedNames()
		if len(names) == 0 {
			return nil, &mstErr{http.StatusBadRequest, "ContainerNotFoundException", "No containers exist"}
		}
		name = names[0]
	}
	c, err := st.get(name)
	if err != nil {
		return nil, err
	}
	return mstOut("DescribeContainer", map[string]any{"Container": c.out(base)}), nil
}

func (st *mstStore) sortedNames() []string {
	names := make([]string, 0, len(st.containers))
	for n := range st.containers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (st *mstStore) listContainers(token string, max int, base string) (map[string]any, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := st.sortedNames()
	start := 0
	if token != "" {
		after, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return nil, mstInvalid("NextToken is not valid")
		}
		start = sort.SearchStrings(names, string(after)+"\x00")
	}
	list := []any{}
	out := map[string]any{}
	for i := start; i < len(names); i++ {
		if len(list) == max {
			out["NextToken"] = base64.StdEncoding.EncodeToString([]byte(names[i-1]))
			break
		}
		c := st.containers[names[i]]
		c.settle()
		list = append(list, c.out(base))
	}
	out["Containers"] = list
	return mstOut("ListContainers", out), nil
}

func (st *mstStore) deleteContainer(name string) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.get(name)
	if err != nil {
		return err
	}
	if len(c.objects) > 0 {
		return &mstErr{http.StatusBadRequest, "ContainerInUseException", fmt.Sprintf("Container %s still holds %d objects; delete them first", name, len(c.objects))}
	}
	delete(st.containers, name)
	delete(st.endpoints, c.endpointID)
	return nil
}

// policyKind names one of the container's four policies.
type mstPolicyKind int

const (
	mstContainerPolicy mstPolicyKind = iota
	mstCorsPolicy
	mstLifecyclePolicy
	mstMetricPolicy
)

func mstPolicyMissing(kind mstPolicyKind, name string) *mstErr {
	if kind == mstCorsPolicy {
		return &mstErr{http.StatusBadRequest, "CorsPolicyNotFoundException", fmt.Sprintf("Container %s has no CORS policy", name)}
	}
	return &mstErr{http.StatusBadRequest, "PolicyNotFoundException", fmt.Sprintf("Container %s has no such policy", name)}
}

// putPolicy stores an already-validated policy value: a string for the
// container and lifecycle policies, the conformed JSON for CORS and metrics.
func (st *mstStore) putPolicy(name string, kind mstPolicyKind, value any) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.get(name)
	if err != nil {
		return err
	}
	switch kind {
	case mstContainerPolicy:
		c.policy, _ = value.(string)
	case mstLifecyclePolicy:
		s, _ := value.(string)
		c.lifecycle = &s
	case mstCorsPolicy:
		c.cors, _ = value.([]any)
	case mstMetricPolicy:
		c.metric, _ = value.(map[string]any)
	}
	return nil
}

func (st *mstStore) getPolicy(name string, kind mstPolicyKind) (map[string]any, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.get(name)
	if err != nil {
		return nil, err
	}
	switch {
	case kind == mstContainerPolicy && c.policy != "":
		return mstOut("GetContainerPolicy", map[string]any{"Policy": c.policy}), nil
	case kind == mstLifecyclePolicy && c.lifecycle != nil:
		return mstOut("GetLifecyclePolicy", map[string]any{"LifecyclePolicy": *c.lifecycle}), nil
	case kind == mstCorsPolicy && c.cors != nil:
		return mstOut("GetCorsPolicy", map[string]any{"CorsPolicy": c.cors}), nil
	case kind == mstMetricPolicy && c.metric != nil:
		return mstOut("GetMetricPolicy", map[string]any{"MetricPolicy": c.metric}), nil
	}
	return nil, mstPolicyMissing(kind, name)
}

func (st *mstStore) deletePolicy(name string, kind mstPolicyKind) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.get(name)
	if err != nil {
		return err
	}
	switch {
	case kind == mstContainerPolicy && c.policy != "":
		c.policy = ""
	case kind == mstLifecyclePolicy && c.lifecycle != nil:
		c.lifecycle = nil
	case kind == mstCorsPolicy && c.cors != nil:
		c.cors = nil
	case kind == mstMetricPolicy && c.metric != nil:
		c.metric = nil
	default:
		return mstPolicyMissing(kind, name)
	}
	return nil
}

func (st *mstStore) setLogging(name string, on bool) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.get(name)
	if err != nil {
		return err
	}
	c.logging = on
	return nil
}

// byARN resolves a container ARN. Callers hold st.mu.
func (st *mstStore) byARN(arn string) (*mstContainer, *mstErr) {
	name, ok := strings.CutPrefix(arn, "arn:aws:mediastore:"+mstRegion+":"+mstAccount+":container/")
	if !ok {
		return nil, &mstErr{http.StatusBadRequest, "ContainerNotFoundException", fmt.Sprintf("Resource %s is not a container in this account and region", arn)}
	}
	return st.get(name)
}

func (st *mstStore) tagResource(arn string, tags []mstTag) *mstErr {
	if err := mstCheckTags(tags); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.byARN(arn)
	if err != nil {
		return err
	}
	merged := mstMergeTags(c.tags, tags)
	if len(merged) > mstMaxTags {
		return mstInvalid("A container can have at most %d tags", mstMaxTags)
	}
	c.tags = merged
	return nil
}

func (st *mstStore) untagResource(arn string, keys []string) *mstErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.byARN(arn)
	if err != nil {
		return err
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	kept := make([]mstTag, 0, len(c.tags))
	for _, t := range c.tags {
		if !drop[t.key] {
			kept = append(kept, t)
		}
	}
	c.tags = kept
	return nil
}

func (st *mstStore) listTags(arn string) (map[string]any, *mstErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.byARN(arn)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(c.tags) > 0 {
		list := make([]any, 0, len(c.tags))
		for _, t := range c.tags {
			list = append(list, map[string]any{"Key": t.key, "Value": t.value})
		}
		out["Tags"] = list
	}
	return mstOut("ListTagsForResource", out), nil
}

func mstCheckTags(tags []mstTag) *mstErr {
	if len(tags) > mstMaxTags {
		return mstInvalid("A container can have at most %d tags", mstMaxTags)
	}
	for _, t := range tags {
		if strings.HasPrefix(strings.ToLower(t.key), "aws:") {
			return mstInvalid("Tag keys can't start with aws: (%s)", t.key)
		}
	}
	return nil
}

// mstMergeTags returns a new list: old tags, with values from add replacing
// those of equal keys and new keys appended in order.
func mstMergeTags(old, add []mstTag) []mstTag {
	out := append([]mstTag(nil), old...)
	for _, t := range add {
		replaced := false
		for i := range out {
			if out[i].key == t.key {
				out[i].value, replaced = t.value, true
			}
		}
		if !replaced {
			out = append(out, t)
		}
	}
	return out
}

// mstCheckPolicyDocument checks a container policy is an IAM-style JSON
// document with at least one statement.
func mstCheckPolicyDocument(doc string) *mstErr {
	var p struct {
		Version   string
		Statement json.RawMessage
	}
	if err := json.Unmarshal([]byte(doc), &p); err != nil {
		return mstInvalid("Policy is not valid JSON")
	}
	var stmts []map[string]any
	if err := json.Unmarshal(p.Statement, &stmts); err != nil {
		var one map[string]any
		if json.Unmarshal(p.Statement, &one) != nil || one == nil {
			return mstInvalid("Policy needs a Statement")
		}
		stmts = []map[string]any{one}
	}
	if len(stmts) == 0 {
		return mstInvalid("Policy needs at least one Statement")
	}
	for _, s := range stmts {
		if e, _ := s["Effect"].(string); e != "Allow" && e != "Deny" {
			return mstInvalid("Every policy statement needs an Effect of Allow or Deny")
		}
		if s["Action"] == nil {
			return mstInvalid("Every policy statement needs an Action")
		}
	}
	return nil
}

// mstCheckLifecyclePolicy checks the lifecycle policy JSON: 1 to 10 rules,
// each with a definition object and an EXPIRE or ARCHIVE action.
// ponytail: rules are stored and returned, not applied to objects.
func mstCheckLifecyclePolicy(doc string) *mstErr {
	var p struct {
		Rules []struct {
			Definition map[string]any `json:"definition"`
			Action     string         `json:"action"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(doc), &p); err != nil {
		return mstInvalid("LifecyclePolicy is not valid JSON")
	}
	if len(p.Rules) == 0 || len(p.Rules) > mstMaxLifecycle {
		return mstInvalid("LifecyclePolicy needs 1 to %d rules", mstMaxLifecycle)
	}
	for i, r := range p.Rules {
		if len(r.Definition) == 0 {
			return mstInvalid("Lifecycle rule %d needs a definition", i+1)
		}
		if r.Action != "EXPIRE" && r.Action != "ARCHIVE" {
			return mstInvalid("Lifecycle rule %d action must be EXPIRE or ARCHIVE", i+1)
		}
	}
	return nil
}

// mstCheckMetricPolicy requires unique object group names.
func mstCheckMetricPolicy(policy map[string]any) *mstErr {
	rules, _ := policy["MetricPolicyRules"].([]any)
	seen := map[string]bool{}
	for _, r := range rules {
		rule, _ := r.(map[string]any)
		name, _ := rule["ObjectGroupName"].(string)
		if seen[name] {
			return mstInvalid("ObjectGroupName %s is used by more than one rule", name)
		}
		seen[name] = true
	}
	return nil
}
