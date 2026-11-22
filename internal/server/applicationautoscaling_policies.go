package server

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	aasPolicyNamePattern    = regexp.MustCompile(`^\P{Cc}+$`)
	aasScheduledNamePattern = regexp.MustCompile(`^[^\x00-\x1f\x7f-\x9f:/|]+$`)
	aasAtSchedule           = regexp.MustCompile(`^at\((\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\)$`)
	aasRateSchedule         = regexp.MustCompile(`^rate\((\d+) (minute|minutes|hour|hours|day|days)\)$`)
	aasCronSchedule         = regexp.MustCompile(`^cron\((.+)\)$`)
	aasCronField            = regexp.MustCompile(`^[0-9A-Za-z*?,/#LW-]+$`)
)

// aasMetricFamilies lists the predefined metric type prefixes valid per
// namespace for target tracking.
var aasMetricFamilies = map[string][]string{
	"ecs": {"ECSService", "ALBRequestCountPerTarget"}, "ec2": {"EC2SpotFleetRequest", "ALBRequestCountPerTarget"},
	"dynamodb": {"DynamoDB"}, "cassandra": {"Cassandra"}, "appstream": {"AppStream"}, "rds": {"RDSReader"},
	"sagemaker": {"SageMaker"}, "comprehend": {"Comprehend"}, "lambda": {"Lambda"}, "kafka": {"Kafka"},
	"elasticache": {"ElastiCache"}, "neptune": {"NeptuneReader"}, "workspaces": {"WorkSpaces"},
}

var aasPredictiveMetrics = map[string]map[string]bool{
	"pair":    {"ECSServiceCPUUtilization": true, "ECSServiceMemoryUtilization": true, "ALBRequestCount": true},
	"scaling": {"ECSServiceAverageCPUUtilization": true, "ECSServiceAverageMemoryUtilization": true, "ALBRequestCountPerTarget": true},
	"load":    {"ECSServiceTotalCPUUtilization": true, "ECSServiceTotalMemoryUtilization": true, "ALBTargetGroupRequestCount": true},
}

func aasCheckPredefined(ns, dim string, m *aasPredefinedMetric) *aasErr {
	typ := m.PredefinedMetricType
	ok := false
	for _, prefix := range aasMetricFamilies[ns] {
		if strings.HasPrefix(typ, prefix) {
			ok = true
		}
	}
	if !ok {
		return aasValidation("Unsupported predefined metric type %q for service namespace %s", typ, ns)
	}
	// Read capacity is tracked with the read metric, write with the write one.
	if (ns == "dynamodb" || ns == "cassandra") && strings.Contains(dim, "ReadCapacity") != strings.Contains(typ, "Read") {
		return aasValidation("Predefined metric type %s doesn't match scalable dimension %s", typ, dim)
	}
	if label := m.ResourceLabel; (typ == "ALBRequestCountPerTarget") != (label != "") {
		return aasValidation("ResourceLabel is required for ALBRequestCountPerTarget and not allowed for other metric types")
	} else if len(label) > 1023 {
		return aasValidation("ResourceLabel must be at most 1023 characters")
	}
	return nil
}

// aasCheckQueries validates metric math: each query has an Id and exactly one
// of Expression or MetricStat, and exactly one query returns data.
func aasCheckQueries(queries []aasMetricQuery) *aasErr {
	if len(queries) == 0 || len(queries) > 10 {
		return aasValidation("metric queries must contain between 1 and 10 entries")
	}
	returning := 0
	for _, q := range queries {
		if q.Id == "" || len(q.Id) > 255 {
			return aasValidation("each metric query needs an Id of 1-255 characters")
		}
		if (q.Expression == "") == (q.MetricStat == nil) {
			return aasValidation("metric query %s must specify exactly one of Expression or MetricStat", q.Id)
		}
		if len(q.Expression) > 2048 {
			return aasValidation("metric query %s Expression must be at most 2048 characters", q.Id)
		}
		if q.MetricStat != nil && q.MetricStat.Stat == "" {
			return aasValidation("metric query %s MetricStat requires Stat", q.Id)
		}
		if q.ReturnData == nil || *q.ReturnData {
			returning++
		}
	}
	if returning != 1 {
		return aasValidation("exactly one metric query must return data")
	}
	return nil
}

func aasCheckStep(c *aasStepConfig) *aasErr {
	switch c.AdjustmentType {
	case "ChangeInCapacity", "ExactCapacity", "PercentChangeInCapacity":
	default:
		return aasValidation("StepScalingPolicyConfiguration requires an AdjustmentType of ChangeInCapacity, PercentChangeInCapacity or ExactCapacity")
	}
	if c.MinAdjustmentMagnitude != nil && (c.AdjustmentType != "PercentChangeInCapacity" || *c.MinAdjustmentMagnitude < 1) {
		return aasValidation("MinAdjustmentMagnitude must be at least 1 and is only valid with PercentChangeInCapacity")
	}
	if c.Cooldown != nil && *c.Cooldown < 0 {
		return aasValidation("Cooldown must be greater than or equal to 0")
	}
	switch c.MetricAggregationType {
	case "", "Average", "Minimum", "Maximum":
	default:
		return aasValidation("MetricAggregationType must be Average, Minimum or Maximum")
	}
	steps := c.StepAdjustments
	if len(steps) == 0 || len(steps) > 20 {
		return aasValidation("StepAdjustments must contain between 1 and 20 adjustments")
	}
	lower := func(s aasStepAdjustment) float64 {
		if s.MetricIntervalLowerBound == nil {
			return math.Inf(-1)
		}
		return *s.MetricIntervalLowerBound
	}
	upper := func(s aasStepAdjustment) float64 {
		if s.MetricIntervalUpperBound == nil {
			return math.Inf(1)
		}
		return *s.MetricIntervalUpperBound
	}
	sorted := append([]aasStepAdjustment{}, steps...)
	sort.Slice(sorted, func(i, j int) bool { return lower(sorted[i]) < lower(sorted[j]) })
	for i, s := range sorted {
		if s.ScalingAdjustment == nil {
			return aasValidation("each step adjustment requires ScalingAdjustment")
		}
		if lower(s) >= upper(s) {
			return aasValidation("a step adjustment's lower bound must be less than its upper bound")
		}
		if i > 0 && upper(sorted[i-1]) != lower(s) {
			return aasValidation("step adjustment intervals can't overlap or leave a gap")
		}
	}
	return nil
}

func aasCheckTargetTracking(ns, dim string, c *aasTargetTrackingConfig) *aasErr {
	if c.TargetValue == nil || !aasFinite(*c.TargetValue) || *c.TargetValue <= 0 {
		return aasValidation("TargetTrackingScalingPolicyConfiguration requires a positive TargetValue")
	}
	for _, cd := range []*int32{c.ScaleInCooldown, c.ScaleOutCooldown} {
		if cd != nil && *cd < 0 {
			return aasValidation("cooldowns must be greater than or equal to 0")
		}
	}
	switch {
	case (c.PredefinedMetricSpecification == nil) == (c.CustomizedMetricSpecification == nil):
		return aasValidation("specify exactly one of PredefinedMetricSpecification or CustomizedMetricSpecification")
	case c.PredefinedMetricSpecification != nil:
		return aasCheckPredefined(ns, dim, c.PredefinedMetricSpecification)
	}
	m := c.CustomizedMetricSpecification
	if len(m.Metrics) > 0 {
		if m.MetricName != "" || m.Namespace != "" || m.Statistic != "" || len(m.Dimensions) > 0 {
			return aasValidation("a customized metric uses either Metrics or MetricName/Namespace/Statistic, not both")
		}
		return aasCheckQueries(m.Metrics)
	}
	if m.MetricName == "" || m.Namespace == "" || m.Statistic == "" {
		return aasValidation("a customized metric requires MetricName, Namespace and Statistic")
	}
	switch m.Statistic {
	case "Average", "Minimum", "Maximum", "SampleCount", "Sum":
		return nil
	}
	return aasValidation("Statistic must be Average, Minimum, Maximum, SampleCount or Sum")
}

func aasCheckPredictive(ns string, c *aasPredictiveConfig) *aasErr {
	// ponytail: AWS supports predictive scaling for ECS services only.
	if ns != "ecs" {
		return aasValidation("PredictiveScaling policies are supported only for the ecs service namespace")
	}
	if len(c.MetricSpecifications) != 1 {
		return aasValidation("PredictiveScalingPolicyConfiguration requires exactly one metric specification")
	}
	switch c.Mode {
	case "", "ForecastOnly", "ForecastAndScale":
	default:
		return aasValidation("Mode must be ForecastOnly or ForecastAndScale")
	}
	if b := c.SchedulingBufferTime; b != nil && (*b < 0 || *b > 3600) {
		return aasValidation("SchedulingBufferTime must be between 0 and 3600")
	}
	switch c.MaxCapacityBreachBehavior {
	case "", "HonorMaxCapacity":
		if c.MaxCapacityBuffer != nil {
			return aasValidation("MaxCapacityBuffer is only valid with MaxCapacityBreachBehavior IncreaseMaxCapacity")
		}
	case "IncreaseMaxCapacity":
		if c.MaxCapacityBuffer == nil || *c.MaxCapacityBuffer < 0 || *c.MaxCapacityBuffer > 100 {
			return aasValidation("IncreaseMaxCapacity requires a MaxCapacityBuffer between 0 and 100")
		}
	default:
		return aasValidation("MaxCapacityBreachBehavior must be HonorMaxCapacity or IncreaseMaxCapacity")
	}
	s := c.MetricSpecifications[0]
	if s.TargetValue == nil || !aasFinite(*s.TargetValue) || *s.TargetValue <= 0 {
		return aasValidation("a predictive metric specification requires a positive TargetValue")
	}
	check := func(kind string, m *aasPredefinedMetric) *aasErr {
		if m != nil && !aasPredictiveMetrics[kind][m.PredefinedMetricType] {
			return aasValidation("Unsupported predictive %s metric type %q", kind, m.PredefinedMetricType)
		}
		return nil
	}
	for kind, m := range map[string]*aasPredefinedMetric{"pair": s.PredefinedMetricPairSpecification, "scaling": s.PredefinedScalingMetricSpecification, "load": s.PredefinedLoadMetricSpecification} {
		if err := check(kind, m); err != nil {
			return err
		}
	}
	for _, q := range []*aasPredictiveCustomized{s.CustomizedScalingMetricSpecification, s.CustomizedLoadMetricSpecification, s.CustomizedCapacityMetricSpecification} {
		if q != nil {
			if err := aasCheckQueries(q.MetricDataQueries); err != nil {
				return err
			}
		}
	}
	// Either a metric pair, or one scaling and one load metric (predefined
	// or customized each).
	hasScaling := s.PredefinedScalingMetricSpecification != nil || s.CustomizedScalingMetricSpecification != nil
	hasLoad := s.PredefinedLoadMetricSpecification != nil || s.CustomizedLoadMetricSpecification != nil
	both := (s.PredefinedScalingMetricSpecification != nil && s.CustomizedScalingMetricSpecification != nil) ||
		(s.PredefinedLoadMetricSpecification != nil && s.CustomizedLoadMetricSpecification != nil)
	if s.PredefinedMetricPairSpecification != nil {
		if hasScaling || hasLoad {
			return aasValidation("a metric pair can't be combined with separate scaling or load metrics")
		}
		return nil
	}
	if !hasScaling || !hasLoad || both {
		return aasValidation("specify a metric pair, or exactly one scaling metric and one load metric")
	}
	return nil
}

func (st *applicationAutoScalingStore) findPolicyLocked(k aasTargetKey, name string) (int, *aasPolicy) {
	for i, p := range st.policies {
		if p.target == k && p.out.PolicyName == name {
			return i, p
		}
	}
	return -1, nil
}

func (st *applicationAutoScalingStore) dropAlarmsLocked(alarms []aasAlarm) {
	for _, a := range alarms {
		st.deleteAlarm(a.AlarmName)
	}
}

func (st *applicationAutoScalingStore) putPolicy(in aasPutPolicyInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	if in.PolicyName == "" || len(in.PolicyName) > 256 || !aasPolicyNamePattern.MatchString(in.PolicyName) {
		return nil, aasValidation("PolicyName must be 1-256 printable characters")
	}
	if in.PolicyType == "" {
		in.PolicyType = "StepScaling"
	}
	configs := map[string]bool{
		"StepScaling":           in.StepScalingPolicyConfiguration != nil,
		"TargetTrackingScaling": in.TargetTrackingScalingPolicyConfiguration != nil,
		"PredictiveScaling":     in.PredictiveScalingPolicyConfiguration != nil,
	}
	if _, ok := configs[in.PolicyType]; !ok {
		return nil, aasValidation("PolicyType must be StepScaling, TargetTrackingScaling or PredictiveScaling")
	}
	for typ, set := range configs {
		if set && typ != in.PolicyType {
			return nil, aasValidation("a %s policy can't include a %s configuration", in.PolicyType, typ)
		}
	}
	if !configs[in.PolicyType] {
		return nil, aasValidation("a %s policy requires its %sPolicyConfiguration", in.PolicyType, in.PolicyType)
	}
	var err *aasErr
	switch in.PolicyType {
	case "StepScaling":
		err = aasCheckStep(in.StepScalingPolicyConfiguration)
	case "TargetTrackingScaling":
		err = aasCheckTargetTracking(in.ServiceNamespace, in.ScalableDimension, in.TargetTrackingScalingPolicyConfiguration)
	case "PredictiveScaling":
		err = aasCheckPredictive(in.ServiceNamespace, in.PredictiveScalingPolicyConfiguration)
	}
	if err != nil {
		return nil, err
	}
	if st.findTargetLocked(in.aasTargetKey) == nil {
		return nil, aasMissingTarget(in.aasTargetKey)
	}
	_, existing := st.findPolicyLocked(in.aasTargetKey, in.PolicyName)
	if existing == nil {
		count := 0
		for _, p := range st.policies {
			if p.target == in.aasTargetKey {
				count++
			}
		}
		if count >= aasMaxPoliciesPerTarget {
			return nil, &aasErr{"LimitExceededException", fmt.Sprintf("a scalable target can have at most %d scaling policies", aasMaxPoliciesPerTarget)}
		}
	}
	out := aasPolicyOut{PolicyName: in.PolicyName, aasTargetKey: in.aasTargetKey, PolicyType: in.PolicyType,
		StepScalingPolicyConfiguration:           in.StepScalingPolicyConfiguration,
		TargetTrackingScalingPolicyConfiguration: in.TargetTrackingScalingPolicyConfiguration,
		PredictiveScalingPolicyConfiguration:     in.PredictiveScalingPolicyConfiguration,
		Alarms:                                   []aasAlarm{}, CreationTime: aasNow()}
	if existing != nil {
		// An update keeps the policy's ARN and creation time and replaces
		// its configuration and alarms.
		out.PolicyARN, out.CreationTime = existing.out.PolicyARN, existing.out.CreationTime
		st.dropAlarmsLocked(existing.out.Alarms)
	} else {
		out.PolicyARN = fmt.Sprintf("arn:aws:autoscaling:%s:%s:scalingPolicy:%s:resource/%s/%s:policyName/%s",
			aasRegion, aasAccount, aasUUID(), in.ServiceNamespace, in.ResourceId, in.PolicyName)
	}
	if in.PolicyType == "TargetTrackingScaling" {
		// Target tracking owns a high and a low CloudWatch alarm.
		id := aasUUID()
		for _, side := range []string{"AlarmHigh", "AlarmLow"} {
			name := fmt.Sprintf("TargetTracking-%s-%s-%s", in.ResourceId, side, id)
			st.putAlarm(name)
			out.Alarms = append(out.Alarms, aasAlarm{AlarmName: name,
				AlarmARN: fmt.Sprintf("arn:aws:cloudwatch:%s:%s:alarm:%s", aasRegion, aasAccount, name)})
		}
	}
	if existing != nil {
		existing.out = out
	} else {
		st.policies = append(st.policies, &aasPolicy{out: out, target: in.aasTargetKey})
	}
	return map[string]any{"PolicyARN": out.PolicyARN, "Alarms": append([]aasAlarm{}, out.Alarms...)}, nil
}

func (st *applicationAutoScalingStore) deletePolicy(in aasNamedInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	i, p := st.findPolicyLocked(in.aasTargetKey, in.PolicyName)
	if p == nil {
		return nil, aasNotFound("No scaling policy found for service namespace: %s, resource ID: %s, scalable dimension: %s, policy name: %s", in.ServiceNamespace, in.ResourceId, in.ScalableDimension, in.PolicyName)
	}
	st.dropAlarmsLocked(p.out.Alarms)
	st.policies = append(st.policies[:i], st.policies[i+1:]...)
	return map[string]any{}, nil
}

// --- scheduled actions ---

func aasCheckSchedule(schedule string) *aasErr {
	if len(schedule) > 1600 {
		return aasValidation("Schedule must be at most 1600 characters")
	}
	if m := aasAtSchedule.FindStringSubmatch(schedule); m != nil {
		if _, err := time.Parse("2006-01-02T15:04:05", m[1]); err != nil {
			return aasValidation("invalid at() schedule: %s", schedule)
		}
		return nil
	}
	if m := aasRateSchedule.FindStringSubmatch(schedule); m != nil {
		n, err := strconv.ParseInt(m[1], 10, 32)
		if err != nil || n < 1 || (n == 1) != !strings.HasSuffix(m[2], "s") {
			return aasValidation("invalid rate() schedule: %s (use a positive value, singular units only for 1)", schedule)
		}
		return nil
	}
	if m := aasCronSchedule.FindStringSubmatch(schedule); m != nil {
		fields := strings.Fields(m[1])
		if len(fields) != 6 {
			return aasValidation("a cron() schedule needs 6 fields: minutes hours day-of-month month day-of-week year")
		}
		for _, f := range fields {
			if !aasCronField.MatchString(f) {
				return aasValidation("invalid cron() field %q", f)
			}
		}
		if (fields[2] == "?") == (fields[4] == "?") {
			return aasValidation("a cron() schedule must use ? in exactly one of day-of-month and day-of-week")
		}
		return nil
	}
	return aasValidation("Schedule must be an at(), rate() or cron() expression")
}

func (st *applicationAutoScalingStore) findScheduledLocked(k aasTargetKey, name string) (int, *aasScheduled) {
	for i, a := range st.scheduled {
		if a.target == k && a.out.ScheduledActionName == name {
			return i, a
		}
	}
	return -1, nil
}

func (st *applicationAutoScalingStore) putScheduled(in aasPutScheduledActionInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	name := in.ScheduledActionName
	if name == "" || len(name) > 256 || !aasScheduledNamePattern.MatchString(name) || strings.TrimSpace(name) != name {
		return nil, aasValidation("ScheduledActionName must be 1-256 characters without : / | or leading or trailing spaces")
	}
	if in.Schedule != nil {
		if err := aasCheckSchedule(*in.Schedule); err != nil {
			return nil, err
		}
	}
	if in.Timezone != nil {
		if _, err := time.LoadLocation(*in.Timezone); err != nil || *in.Timezone == "" || len(*in.Timezone) > 1600 {
			return nil, aasValidation("Timezone must be an IANA time zone name: %s", *in.Timezone)
		}
	}
	for _, ts := range []*float64{in.StartTime, in.EndTime} {
		if ts != nil && (!aasFinite(*ts) || *ts < 0) {
			return nil, aasValidation("StartTime and EndTime must be valid timestamps")
		}
	}
	if a := in.ScalableTargetAction; a != nil {
		if (a.MinCapacity != nil && *a.MinCapacity < 0) || (a.MaxCapacity != nil && *a.MaxCapacity < 0) {
			return nil, aasValidation("ScalableTargetAction capacities must be greater than or equal to 0")
		}
	}
	if st.findTargetLocked(in.aasTargetKey) == nil {
		return nil, aasMissingTarget(in.aasTargetKey)
	}
	_, existing := st.findScheduledLocked(in.aasTargetKey, name)
	var out aasScheduledActionOut
	if existing != nil {
		out = existing.out
	} else {
		count := 0
		for _, a := range st.scheduled {
			if a.target == in.aasTargetKey {
				count++
			}
		}
		if count >= aasMaxScheduledPerTarget {
			return nil, &aasErr{"LimitExceededException", fmt.Sprintf("a scalable target can have at most %d scheduled actions", aasMaxScheduledPerTarget)}
		}
		if in.Schedule == nil {
			return nil, aasValidation("Schedule is required to create a scheduled action")
		}
		if in.ScalableTargetAction == nil || (in.ScalableTargetAction.MinCapacity == nil && in.ScalableTargetAction.MaxCapacity == nil) {
			return nil, aasValidation("ScalableTargetAction requires MinCapacity, MaxCapacity or both")
		}
		out = aasScheduledActionOut{ScheduledActionName: name, ServiceNamespace: in.ServiceNamespace, ResourceId: in.ResourceId,
			ScalableDimension: in.ScalableDimension, CreationTime: aasNow(),
			ScheduledActionARN: fmt.Sprintf("arn:aws:autoscaling:%s:%s:scheduledAction:%s:resource/%s/%s:scheduledActionName/%s",
				aasRegion, aasAccount, aasUUID(), in.ServiceNamespace, in.ResourceId, name)}
	}
	if in.Schedule != nil {
		out.Schedule = *in.Schedule
	}
	if in.Timezone != nil {
		out.Timezone = *in.Timezone
	}
	if in.StartTime != nil {
		v := *in.StartTime
		out.StartTime = &v
	}
	if in.EndTime != nil {
		v := *in.EndTime
		out.EndTime = &v
	}
	if a := in.ScalableTargetAction; a != nil {
		merged := aasCapacityAction{}
		if out.ScalableTargetAction != nil {
			merged = *out.ScalableTargetAction
		}
		if a.MinCapacity != nil {
			v := *a.MinCapacity
			merged.MinCapacity = &v
		}
		if a.MaxCapacity != nil {
			v := *a.MaxCapacity
			merged.MaxCapacity = &v
		}
		out.ScalableTargetAction = &merged
	}
	if a := out.ScalableTargetAction; a != nil && a.MinCapacity != nil && a.MaxCapacity != nil && *a.MinCapacity > *a.MaxCapacity {
		return nil, aasValidation("ScalableTargetAction MinCapacity cannot be greater than MaxCapacity")
	}
	if out.StartTime != nil && out.EndTime != nil && *out.StartTime >= *out.EndTime {
		return nil, aasValidation("StartTime must be before EndTime")
	}
	if existing != nil {
		existing.out = out
	} else {
		st.scheduled = append(st.scheduled, &aasScheduled{out: out, target: in.aasTargetKey})
	}
	return map[string]any{}, nil
}

func (st *applicationAutoScalingStore) deleteScheduled(in aasNamedInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	i, a := st.findScheduledLocked(in.aasTargetKey, in.ScheduledActionName)
	if a == nil {
		return nil, aasNotFound("No scheduled action found for service namespace: %s, resource ID: %s, scalable dimension: %s, scheduled action name: %s", in.ServiceNamespace, in.ResourceId, in.ScalableDimension, in.ScheduledActionName)
	}
	st.scheduled = append(st.scheduled[:i], st.scheduled[i+1:]...)
	return map[string]any{}, nil
}

// --- predictive forecast ---

// forecast returns hourly points between StartTime and EndTime.
// ponytail: there is no metric history, so the load forecast is flat at the
// policy's target value and the capacity forecast is MinCapacity+1 clamped to
// the target's range.
func (st *applicationAutoScalingStore) forecast(in aasForecastInput) (any, *aasErr) {
	if err := aasCheckKey(in.aasTargetKey); err != nil {
		return nil, err
	}
	if in.StartTime == nil || in.EndTime == nil || !aasFinite(*in.StartTime) || !aasFinite(*in.EndTime) {
		return nil, aasValidation("StartTime and EndTime are required")
	}
	start, end := *in.StartTime, *in.EndTime
	if start >= end {
		return nil, aasValidation("StartTime must be before EndTime")
	}
	if end-start > aasMaxForecastSpan.Seconds() {
		return nil, aasValidation("the forecast window can be at most 30 days")
	}
	t := st.findTargetLocked(in.aasTargetKey)
	if t == nil {
		return nil, aasMissingTarget(in.aasTargetKey)
	}
	_, p := st.findPolicyLocked(in.aasTargetKey, in.PolicyName)
	if p == nil || p.out.PolicyType != "PredictiveScaling" {
		return nil, aasValidation("No predictive scaling policy named %s for resource %s", in.PolicyName, in.ResourceId)
	}
	spec := p.out.PredictiveScalingPolicyConfiguration.MetricSpecifications[0]
	capacity := float64(min(max(t.min+1, t.min), t.max))
	var stamps, load, capacities []float64
	for ts := math.Ceil(start/3600) * 3600; ts < end; ts += 3600 {
		stamps = append(stamps, ts)
		load = append(load, *spec.TargetValue)
		capacities = append(capacities, capacity)
	}
	if stamps == nil {
		stamps, load, capacities = []float64{}, []float64{}, []float64{}
	}
	return map[string]any{
		"LoadForecast":     []any{map[string]any{"Timestamps": stamps, "Values": load, "MetricSpecification": spec}},
		"CapacityForecast": map[string]any{"Timestamps": stamps, "Values": capacities},
		"UpdateTime":       aasNow(),
	}, nil
}
