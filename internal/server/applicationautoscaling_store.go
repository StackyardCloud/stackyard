package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	aasAccount = "123456789012"
	aasRegion  = "us-east-1"

	aasMaxTargets             = 3000 // the default per-namespace quota, applied store-wide
	aasMaxPoliciesPerTarget   = 50
	aasMaxScheduledPerTarget  = 200
	aasMaxActivitiesPerTarget = 100
	aasMaxTags                = 50
	aasMaxPageSize            = 50
	aasMaxForecastSpan        = 30 * 24 * time.Hour
)

type aasErr struct{ code, msg string }

func aasValidation(format string, args ...any) *aasErr {
	return &aasErr{"ValidationException", fmt.Sprintf(format, args...)}
}

func aasNotFound(format string, args ...any) *aasErr {
	return &aasErr{"ObjectNotFoundException", fmt.Sprintf(format, args...)}
}

type aasTarget struct {
	key       aasTargetKey
	min, max  int32
	roleARN   string
	created   time.Time
	suspended aasSuspendedStateOut
	arn       string
	tags      map[string]string
}

type aasPolicy struct {
	out    aasPolicyOut // config pointers are replaced wholesale, never mutated
	target aasTargetKey
}

type aasScheduled struct {
	out    aasScheduledActionOut
	target aasTargetKey
}

type aasActivity struct {
	out    aasActivityOut
	target aasTargetKey
}

// applicationAutoScalingStore holds all state under one mutex. Every response
// is marshalled inside run, so nothing reads store memory after unlock.
// ponytail: slices with linear scans; the per-target caps bound them.
type applicationAutoScalingStore struct {
	mu         sync.Mutex
	targets    []*aasTarget
	policies   []*aasPolicy
	scheduled  []*aasScheduled
	activities []*aasActivity
	// alarms creates/deletes the CloudWatch alarms of target tracking policies.
	putAlarm    func(name string)
	deleteAlarm func(name string)
}

func newApplicationAutoScalingStore(putAlarm, deleteAlarm func(string)) *applicationAutoScalingStore {
	return &applicationAutoScalingStore{putAlarm: putAlarm, deleteAlarm: deleteAlarm}
}

func (st *applicationAutoScalingStore) run(fn func() (any, *aasErr)) ([]byte, *aasErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := fn()
	if err != nil {
		return nil, err
	}
	body, mErr := json.Marshal(out)
	if mErr != nil {
		return nil, &aasErr{"InternalServiceException", mErr.Error()}
	}
	return body, nil
}

func aasNow() float64 { return aasEpoch(time.Now()) }

func aasEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func aasID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func aasUUID() string {
	h := aasID()
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// --- validation of namespace, dimension and resource ID ---

var aasNamespaces = map[string]bool{
	"ecs": true, "elasticmapreduce": true, "ec2": true, "appstream": true, "dynamodb": true, "rds": true,
	"sagemaker": true, "custom-resource": true, "comprehend": true, "lambda": true, "cassandra": true,
	"kafka": true, "elasticache": true, "neptune": true, "workspaces": true,
}

// aasResourcePatterns maps each scalable dimension to its ResourceId format.
var aasResourcePatterns = map[string]*regexp.Regexp{
	"ecs:service:DesiredCount":                                      regexp.MustCompile(`^service/[^/]+/[^/]+$`),
	"ec2:spot-fleet-request:TargetCapacity":                         regexp.MustCompile(`^spot-fleet-request/sfr-[0-9a-f-]+$`),
	"elasticmapreduce:instancegroup:InstanceCount":                  regexp.MustCompile(`^instancegroup/j-[A-Za-z0-9]+/ig-[A-Za-z0-9]+$`),
	"appstream:fleet:DesiredCapacity":                               regexp.MustCompile(`^fleet/[^/]+$`),
	"dynamodb:table:ReadCapacityUnits":                              aasDynamoTablePattern,
	"dynamodb:table:WriteCapacityUnits":                             aasDynamoTablePattern,
	"dynamodb:index:ReadCapacityUnits":                              aasDynamoIndexPattern,
	"dynamodb:index:WriteCapacityUnits":                             aasDynamoIndexPattern,
	"rds:cluster:ReadReplicaCount":                                  aasClusterPattern,
	"neptune:cluster:ReadReplicaCount":                              aasClusterPattern,
	"sagemaker:variant:DesiredInstanceCount":                        aasVariantPattern,
	"sagemaker:variant:DesiredProvisionedConcurrency":               aasVariantPattern,
	"sagemaker:inference-component:DesiredCopyCount":                regexp.MustCompile(`^inference-component/[^/]+$`),
	"custom-resource:ResourceType:Property":                         regexp.MustCompile(`^https://\S+$`),
	"comprehend:document-classifier-endpoint:DesiredInferenceUnits": regexp.MustCompile(`^arn:aws:comprehend:[^:]+:\d{12}:document-classifier-endpoint/.+$`),
	"comprehend:entity-recognizer-endpoint:DesiredInferenceUnits":   regexp.MustCompile(`^arn:aws:comprehend:[^:]+:\d{12}:entity-recognizer-endpoint/.+$`),
	"lambda:function:ProvisionedConcurrency":                        regexp.MustCompile(`^function:[^:]+:[^:]+$`),
	"cassandra:table:ReadCapacityUnits":                             aasCassandraPattern,
	"cassandra:table:WriteCapacityUnits":                            aasCassandraPattern,
	"kafka:broker-storage:VolumeSize":                               regexp.MustCompile(`^arn:aws:kafka:[^:]+:\d{12}:cluster/.+$`),
	"elasticache:cache-cluster:Nodes":                               regexp.MustCompile(`^cache-cluster/[^/]+$`),
	"elasticache:replication-group:NodeGroups":                      aasReplicationGroupPattern,
	"elasticache:replication-group:Replicas":                        aasReplicationGroupPattern,
	"workspaces:workspacespool:DesiredUserSessions":                 regexp.MustCompile(`^workspacespool/wspool-[a-z0-9]+$`),
}

var (
	aasDynamoTablePattern      = regexp.MustCompile(`^table/[A-Za-z0-9_.-]{3,255}$`)
	aasDynamoIndexPattern      = regexp.MustCompile(`^table/[A-Za-z0-9_.-]{3,255}/index/[A-Za-z0-9_.-]{3,255}$`)
	aasClusterPattern          = regexp.MustCompile(`^cluster:[A-Za-z][A-Za-z0-9-]*$`)
	aasVariantPattern          = regexp.MustCompile(`^endpoint/[^/]+/variant/[^/]+$`)
	aasCassandraPattern        = regexp.MustCompile(`^keyspace/[^/]+/table/[^/]+$`)
	aasReplicationGroupPattern = regexp.MustCompile(`^replication-group/[^/]+$`)
	aasRoleARNPattern          = regexp.MustCompile(`^arn:aws[a-z-]*:iam::\d{12}:role/.+$`)
	aasTargetARNPattern        = regexp.MustCompile(`^arn:.+:application-autoscaling:.+:[0-9]+:scalable-target/[a-zA-Z0-9-]+$`)
)

// aasServiceLinkedRoles is the role AWS creates when RoleARN is omitted.
var aasServiceLinkedRoles = map[string]string{
	"ecs": "ECSService", "dynamodb": "DynamoDBTable", "lambda": "LambdaConcurrency", "ec2": "EC2SpotFleetRequest",
	"rds": "RDSCluster", "sagemaker": "SageMakerEndpoint", "appstream": "AppStreamFleet", "custom-resource": "CustomResource",
	"comprehend": "ComprehendEndpoint", "cassandra": "CassandraTable", "kafka": "KafkaCluster", "elasticache": "ElastiCacheRG",
	"neptune": "NeptuneCluster", "workspaces": "WorkSpacesPool", "elasticmapreduce": "EMRInstanceGroup",
}

func aasDefaultRole(ns string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/aws-service-role/%s.application-autoscaling.amazonaws.com/AWSServiceRoleForApplicationAutoScaling_%s", aasAccount, ns, aasServiceLinkedRoles[ns])
}

func aasCheckNamespace(ns string) *aasErr {
	if !aasNamespaces[ns] {
		return aasValidation("1 validation error detected: Value '%s' at 'serviceNamespace' failed to satisfy constraint: Member must satisfy enum value set", ns)
	}
	return nil
}

// aasCheckKey validates a full target key (namespace, dimension, resource ID).
func aasCheckKey(k aasTargetKey) *aasErr {
	if err := aasCheckNamespace(k.ServiceNamespace); err != nil {
		return err
	}
	if k.ResourceId == "" || len(k.ResourceId) > 1600 {
		return aasValidation("ResourceId must be between 1 and 1600 characters")
	}
	pattern, ok := aasResourcePatterns[k.ScalableDimension]
	if !ok || !strings.HasPrefix(k.ScalableDimension, k.ServiceNamespace+":") {
		return aasValidation("Unsupported service namespace, resource type or scalable dimension")
	}
	if !pattern.MatchString(k.ResourceId) {
		return aasValidation("Unsupported resource ID format for %s: %s", k.ScalableDimension, k.ResourceId)
	}
	return nil
}

func aasCheckOptionalDimension(ns, dim string) *aasErr {
	if dim == "" {
		return nil
	}
	if _, ok := aasResourcePatterns[dim]; !ok || !strings.HasPrefix(dim, ns+":") {
		return aasValidation("Unsupported service namespace, resource type or scalable dimension")
	}
	return nil
}

func (st *applicationAutoScalingStore) findTargetLocked(k aasTargetKey) *aasTarget {
	for _, t := range st.targets {
		if t.key == k {
			return t
		}
	}
	return nil
}

func aasMissingTarget(k aasTargetKey) *aasErr {
	return aasNotFound("No scalable target registered for service namespace: %s, resource ID: %s, scalable dimension: %s", k.ServiceNamespace, k.ResourceId, k.ScalableDimension)
}

func (t *aasTarget) out() aasTargetOut {
	return aasTargetOut{aasTargetKey: t.key, MinCapacity: t.min, MaxCapacity: t.max, RoleARN: t.roleARN,
		CreationTime: aasEpoch(t.created), SuspendedState: t.suspended, ScalableTargetARN: t.arn}
}

// --- scalable targets ---

func (st *applicationAutoScalingStore) register(in aasRegisterInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	for _, c := range []*int32{in.MinCapacity, in.MaxCapacity} {
		if c != nil && *c < 0 {
			return nil, aasValidation("MinCapacity and MaxCapacity must be greater than or equal to 0")
		}
	}
	if in.RoleARN != "" && !aasRoleARNPattern.MatchString(in.RoleARN) {
		return nil, aasValidation("RoleARN must be an IAM role ARN: %s", in.RoleARN)
	}
	t := st.findTargetLocked(in.aasTargetKey)
	if t == nil {
		if in.MinCapacity == nil || in.MaxCapacity == nil {
			return nil, aasValidation("MinCapacity and MaxCapacity are required to register a new scalable target")
		}
		if len(st.targets) >= aasMaxTargets {
			return nil, &aasErr{"LimitExceededException", fmt.Sprintf("you can register at most %d scalable targets", aasMaxTargets)}
		}
		if err := aasCheckTags(in.Tags, 0); err != nil {
			return nil, err
		}
		t = &aasTarget{key: in.aasTargetKey, created: time.Now(), roleARN: aasDefaultRole(in.ServiceNamespace),
			arn:  fmt.Sprintf("arn:aws:application-autoscaling:%s:%s:scalable-target/%s", aasRegion, aasAccount, aasID()[:28]),
			tags: map[string]string{}}
		for k, v := range in.Tags {
			t.tags[k] = v
		}
		t.min, t.max = *in.MinCapacity, *in.MaxCapacity
		if t.min > t.max {
			return nil, aasValidation("Minimum capacity cannot be greater than maximum capacity")
		}
		st.targets = append(st.targets, t)
	} else {
		// ponytail: Tags apply at creation only, as the API documents; an
		// update ignores them.
		min, max := t.min, t.max
		if in.MinCapacity != nil {
			min = *in.MinCapacity
		}
		if in.MaxCapacity != nil {
			max = *in.MaxCapacity
		}
		if min > max {
			return nil, aasValidation("Minimum capacity cannot be greater than maximum capacity")
		}
		t.min, t.max = min, max
	}
	if in.RoleARN != "" {
		t.roleARN = in.RoleARN
	}
	if s := in.SuspendedState; s != nil {
		if s.DynamicScalingInSuspended != nil {
			t.suspended.DynamicScalingInSuspended = *s.DynamicScalingInSuspended
		}
		if s.DynamicScalingOutSuspended != nil {
			t.suspended.DynamicScalingOutSuspended = *s.DynamicScalingOutSuspended
		}
		if s.ScheduledScalingSuspended != nil {
			t.suspended.ScheduledScalingSuspended = *s.ScheduledScalingSuspended
		}
	}
	if in.MinCapacity != nil || in.MaxCapacity != nil {
		st.recordActivityLocked(t.key, fmt.Sprintf("Setting min capacity to %d and max capacity to %d", t.min, t.max),
			fmt.Sprintf("minimum capacity was set to %d and maximum capacity was set to %d", t.min, t.max))
	}
	return map[string]any{"ScalableTargetARN": t.arn}, nil
}

func (st *applicationAutoScalingStore) recordActivityLocked(k aasTargetKey, desc, cause string) {
	now := aasNow()
	st.activities = append(st.activities, &aasActivity{target: k, out: aasActivityOut{
		ActivityId: aasUUID(), aasTargetKey: k, Description: desc, Cause: cause,
		StartTime: now, EndTime: now, StatusCode: "Successful", StatusMessage: "Successfully set min capacity and max capacity.",
	}})
	n := 0
	for _, a := range st.activities {
		if a.target == k {
			n++
		}
	}
	if n > aasMaxActivitiesPerTarget {
		for i, a := range st.activities {
			if a.target == k {
				st.activities = append(st.activities[:i], st.activities[i+1:]...)
				break
			}
		}
	}
}

func (st *applicationAutoScalingStore) deregister(k aasTargetKey) (any, *aasErr) {
	if err := aasCheckKey(k); err != nil {
		return nil, err
	}
	if st.findTargetLocked(k) == nil {
		return nil, aasMissingTarget(k)
	}
	// Deregistering deletes the target's policies (and their alarms) and
	// scheduled actions, as AWS does.
	keptPolicies := st.policies[:0]
	for _, p := range st.policies {
		if p.target == k {
			st.dropAlarmsLocked(p.out.Alarms)
			continue
		}
		keptPolicies = append(keptPolicies, p)
	}
	st.policies = keptPolicies
	keptScheduled := st.scheduled[:0]
	for _, a := range st.scheduled {
		if a.target != k {
			keptScheduled = append(keptScheduled, a)
		}
	}
	st.scheduled = keptScheduled
	keptTargets := st.targets[:0]
	for _, t := range st.targets {
		if t.key != k {
			keptTargets = append(keptTargets, t)
		}
	}
	st.targets = keptTargets
	// Activities belong to the target too; keeping them would let repeated
	// register/deregister grow the list without bound.
	keptActivities := st.activities[:0]
	for _, a := range st.activities {
		if a.target != k {
			keptActivities = append(keptActivities, a)
		}
	}
	st.activities = keptActivities
	return map[string]any{}, nil
}

// --- pagination ---

func aasPage(in aasDescribeInput, total int) (start, end int, next string, err *aasErr) {
	size := aasMaxPageSize
	if in.MaxResults != nil {
		if *in.MaxResults < 1 || *in.MaxResults > aasMaxPageSize {
			return 0, 0, "", aasValidation("MaxResults must be between 1 and %d", aasMaxPageSize)
		}
		size = int(*in.MaxResults)
	}
	if in.NextToken != nil && *in.NextToken != "" {
		raw, decErr := base64.RawURLEncoding.DecodeString(*in.NextToken)
		n, convErr := strconv.ParseInt(strings.TrimPrefix(string(raw), "aas:"), 10, 32)
		if decErr != nil || convErr != nil || !strings.HasPrefix(string(raw), "aas:") || n < 0 || int(n) > total {
			return 0, 0, "", &aasErr{"InvalidNextTokenException", "The NextToken value is invalid"}
		}
		start = int(n)
	}
	end = min(start+size, total)
	if end < total {
		next = base64.RawURLEncoding.EncodeToString([]byte("aas:" + strconv.Itoa(end)))
	}
	return start, end, next, nil
}

func aasCheckDescribe(in aasDescribeInput) *aasErr {
	if err := aasCheckNamespace(in.ServiceNamespace); err != nil {
		return err
	}
	if err := aasCheckOptionalDimension(in.ServiceNamespace, in.ScalableDimension); err != nil {
		return err
	}
	for _, list := range [][]string{in.ResourceIds, in.PolicyNames, in.ScheduledActionNames} {
		if len(list) > 50 {
			return aasValidation("a filter list can contain at most 50 entries")
		}
	}
	return nil
}

func aasMatches(k aasTargetKey, in aasDescribeInput, ids []string) bool {
	if k.ServiceNamespace != in.ServiceNamespace {
		return false
	}
	if in.ScalableDimension != "" && k.ScalableDimension != in.ScalableDimension {
		return false
	}
	if in.ResourceId != "" && k.ResourceId != in.ResourceId {
		return false
	}
	return len(ids) == 0 || aasContains(ids, k.ResourceId)
}

func aasContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (st *applicationAutoScalingStore) describeTargets(in aasDescribeInput) (any, *aasErr) {
	if err := aasCheckDescribe(in); err != nil {
		return nil, err
	}
	var all []aasTargetOut
	for _, t := range st.targets {
		if aasMatches(t.key, in, in.ResourceIds) {
			all = append(all, t.out())
		}
	}
	start, end, next, err := aasPage(in, len(all))
	if err != nil {
		return nil, err
	}
	return aasPageOut("ScalableTargets", append([]aasTargetOut{}, all[start:end]...), next), nil
}

func aasPageOut(key string, items any, next string) map[string]any {
	out := map[string]any{key: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out
}

func (st *applicationAutoScalingStore) describeActivities(in aasDescribeInput) (any, *aasErr) {
	if err := aasCheckDescribe(in); err != nil {
		return nil, err
	}
	all := []aasActivityOut{}
	for i := len(st.activities) - 1; i >= 0; i-- { // most recent first
		if a := st.activities[i]; aasMatches(a.target, in, nil) {
			all = append(all, a.out)
		}
	}
	start, end, next, err := aasPage(in, len(all))
	if err != nil {
		return nil, err
	}
	return aasPageOut("ScalingActivities", all[start:end], next), nil
}

func (st *applicationAutoScalingStore) describePolicies(in aasDescribeInput) (any, *aasErr) {
	if err := aasCheckDescribe(in); err != nil {
		return nil, err
	}
	all := []aasPolicyOut{}
	for _, p := range st.policies {
		if aasMatches(p.target, in, nil) && (len(in.PolicyNames) == 0 || aasContains(in.PolicyNames, p.out.PolicyName)) {
			out := p.out
			out.Alarms = append([]aasAlarm{}, p.out.Alarms...)
			all = append(all, out)
		}
	}
	start, end, next, err := aasPage(in, len(all))
	if err != nil {
		return nil, err
	}
	return aasPageOut("ScalingPolicies", all[start:end], next), nil
}

func (st *applicationAutoScalingStore) describeScheduled(in aasDescribeInput) (any, *aasErr) {
	if err := aasCheckDescribe(in); err != nil {
		return nil, err
	}
	all := []aasScheduledActionOut{}
	for _, a := range st.scheduled {
		if aasMatches(a.target, in, nil) && (len(in.ScheduledActionNames) == 0 || aasContains(in.ScheduledActionNames, a.out.ScheduledActionName)) {
			all = append(all, a.out)
		}
	}
	start, end, next, err := aasPage(in, len(all))
	if err != nil {
		return nil, err
	}
	return aasPageOut("ScheduledActions", all[start:end], next), nil
}

// --- tags ---

func aasCheckTags(tags map[string]string, existing int) *aasErr {
	for k, v := range tags {
		if k == "" || len(k) > 128 || len(v) > 256 {
			return aasValidation("tag keys must be 1-128 characters and values at most 256")
		}
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return aasValidation("tag keys can't start with aws:")
		}
	}
	if existing+len(tags) > aasMaxTags {
		return &aasErr{"TooManyTagsException", fmt.Sprintf("a scalable target can have at most %d tags", aasMaxTags)}
	}
	return nil
}

func (st *applicationAutoScalingStore) targetByARNLocked(arn string) (*aasTarget, *aasErr) {
	if !aasTargetARNPattern.MatchString(arn) {
		return nil, aasValidation("ResourceARN must be a scalable target ARN: %s", arn)
	}
	for _, t := range st.targets {
		if t.arn == arn {
			return t, nil
		}
	}
	return nil, &aasErr{"ResourceNotFoundException", "No scalable target found for ARN " + arn}
}

func (st *applicationAutoScalingStore) tag(in aasTagInput) (any, *aasErr) {
	t, err := st.targetByARNLocked(in.ResourceARN)
	if err != nil {
		return nil, err
	}
	if len(in.Tags) == 0 {
		return nil, aasValidation("Tags must contain at least one tag")
	}
	added := 0
	for k := range in.Tags {
		if _, ok := t.tags[k]; !ok {
			added++
		}
	}
	if err := aasCheckTags(in.Tags, len(t.tags)+added-len(in.Tags)); err != nil {
		return nil, err
	}
	for k, v := range in.Tags {
		t.tags[k] = v
	}
	return map[string]any{}, nil
}

func (st *applicationAutoScalingStore) untag(in aasTagInput) (any, *aasErr) {
	t, err := st.targetByARNLocked(in.ResourceARN)
	if err != nil {
		return nil, err
	}
	if len(in.TagKeys) > 200 {
		return nil, aasValidation("TagKeys can contain at most 200 keys")
	}
	for _, k := range in.TagKeys {
		delete(t.tags, k)
	}
	return map[string]any{}, nil
}

func (st *applicationAutoScalingStore) listTags(in aasTagInput) (any, *aasErr) {
	t, err := st.targetByARNLocked(in.ResourceARN)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]string, len(t.tags))
	for k, v := range t.tags {
		tags[k] = v
	}
	return map[string]any{"Tags": tags}, nil
}

func aasFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
