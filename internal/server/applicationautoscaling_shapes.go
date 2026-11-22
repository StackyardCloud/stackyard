package server

// Wire shapes for Application Auto Scaling (awsJson 1.1). Requests decode into
// these typed structs, so only model members are ever read or stored: unknown
// request keys are dropped by encoding/json and can't reach store state.
// Integers are int32 (ResourceCapacity, Cooldown, ...), so out-of-range values
// fail decoding instead of being truncated.

type aasTargetKey struct {
	ServiceNamespace  string `json:"ServiceNamespace"`
	ResourceId        string `json:"ResourceId"`
	ScalableDimension string `json:"ScalableDimension"`
}

type aasSuspendedState struct {
	DynamicScalingInSuspended  *bool `json:"DynamicScalingInSuspended,omitempty"`
	DynamicScalingOutSuspended *bool `json:"DynamicScalingOutSuspended,omitempty"`
	ScheduledScalingSuspended  *bool `json:"ScheduledScalingSuspended,omitempty"`
}

type aasRegisterInput struct {
	aasTargetKey
	MinCapacity    *int32             `json:"MinCapacity"`
	MaxCapacity    *int32             `json:"MaxCapacity"`
	RoleARN        string             `json:"RoleARN"`
	SuspendedState *aasSuspendedState `json:"SuspendedState"`
	Tags           map[string]string  `json:"Tags"`
}

type aasStepAdjustment struct {
	MetricIntervalLowerBound *float64 `json:"MetricIntervalLowerBound,omitempty"`
	MetricIntervalUpperBound *float64 `json:"MetricIntervalUpperBound,omitempty"`
	ScalingAdjustment        *int32   `json:"ScalingAdjustment"`
}

type aasStepConfig struct {
	AdjustmentType         string              `json:"AdjustmentType,omitempty"`
	StepAdjustments        []aasStepAdjustment `json:"StepAdjustments,omitempty"`
	MinAdjustmentMagnitude *int32              `json:"MinAdjustmentMagnitude,omitempty"`
	Cooldown               *int32              `json:"Cooldown,omitempty"`
	MetricAggregationType  string              `json:"MetricAggregationType,omitempty"`
}

type aasPredefinedMetric struct {
	PredefinedMetricType string `json:"PredefinedMetricType"`
	ResourceLabel        string `json:"ResourceLabel,omitempty"`
}

type aasMetricDimension struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type aasMetric struct {
	Dimensions []aasMetricDimension `json:"Dimensions,omitempty"`
	MetricName string               `json:"MetricName,omitempty"`
	Namespace  string               `json:"Namespace,omitempty"`
}

type aasMetricStat struct {
	Metric aasMetric `json:"Metric"`
	Stat   string    `json:"Stat"`
	Unit   string    `json:"Unit,omitempty"`
}

type aasMetricQuery struct {
	Expression string         `json:"Expression,omitempty"`
	Id         string         `json:"Id"`
	Label      string         `json:"Label,omitempty"`
	MetricStat *aasMetricStat `json:"MetricStat,omitempty"`
	ReturnData *bool          `json:"ReturnData,omitempty"`
}

type aasCustomizedMetric struct {
	MetricName string               `json:"MetricName,omitempty"`
	Namespace  string               `json:"Namespace,omitempty"`
	Dimensions []aasMetricDimension `json:"Dimensions,omitempty"`
	Statistic  string               `json:"Statistic,omitempty"`
	Unit       string               `json:"Unit,omitempty"`
	Metrics    []aasMetricQuery     `json:"Metrics,omitempty"`
}

type aasTargetTrackingConfig struct {
	TargetValue                   *float64             `json:"TargetValue"`
	PredefinedMetricSpecification *aasPredefinedMetric `json:"PredefinedMetricSpecification,omitempty"`
	CustomizedMetricSpecification *aasCustomizedMetric `json:"CustomizedMetricSpecification,omitempty"`
	ScaleOutCooldown              *int32               `json:"ScaleOutCooldown,omitempty"`
	ScaleInCooldown               *int32               `json:"ScaleInCooldown,omitempty"`
	DisableScaleIn                *bool                `json:"DisableScaleIn,omitempty"`
}

type aasPredictiveCustomized struct {
	MetricDataQueries []aasMetricQuery `json:"MetricDataQueries"`
}

type aasPredictiveMetricSpec struct {
	TargetValue                           *float64                 `json:"TargetValue"`
	PredefinedMetricPairSpecification     *aasPredefinedMetric     `json:"PredefinedMetricPairSpecification,omitempty"`
	PredefinedScalingMetricSpecification  *aasPredefinedMetric     `json:"PredefinedScalingMetricSpecification,omitempty"`
	PredefinedLoadMetricSpecification     *aasPredefinedMetric     `json:"PredefinedLoadMetricSpecification,omitempty"`
	CustomizedScalingMetricSpecification  *aasPredictiveCustomized `json:"CustomizedScalingMetricSpecification,omitempty"`
	CustomizedLoadMetricSpecification     *aasPredictiveCustomized `json:"CustomizedLoadMetricSpecification,omitempty"`
	CustomizedCapacityMetricSpecification *aasPredictiveCustomized `json:"CustomizedCapacityMetricSpecification,omitempty"`
}

type aasPredictiveConfig struct {
	MetricSpecifications      []aasPredictiveMetricSpec `json:"MetricSpecifications"`
	Mode                      string                    `json:"Mode,omitempty"`
	SchedulingBufferTime      *int32                    `json:"SchedulingBufferTime,omitempty"`
	MaxCapacityBreachBehavior string                    `json:"MaxCapacityBreachBehavior,omitempty"`
	MaxCapacityBuffer         *int32                    `json:"MaxCapacityBuffer,omitempty"`
}

type aasPutPolicyInput struct {
	aasTargetKey
	PolicyName                               string                   `json:"PolicyName"`
	PolicyType                               string                   `json:"PolicyType"`
	StepScalingPolicyConfiguration           *aasStepConfig           `json:"StepScalingPolicyConfiguration"`
	TargetTrackingScalingPolicyConfiguration *aasTargetTrackingConfig `json:"TargetTrackingScalingPolicyConfiguration"`
	PredictiveScalingPolicyConfiguration     *aasPredictiveConfig     `json:"PredictiveScalingPolicyConfiguration"`
}

type aasCapacityAction struct {
	MinCapacity *int32 `json:"MinCapacity,omitempty"`
	MaxCapacity *int32 `json:"MaxCapacity,omitempty"`
}

type aasPutScheduledActionInput struct {
	aasTargetKey
	ScheduledActionName  string             `json:"ScheduledActionName"`
	Schedule             *string            `json:"Schedule"`
	Timezone             *string            `json:"Timezone"`
	StartTime            *float64           `json:"StartTime"`
	EndTime              *float64           `json:"EndTime"`
	ScalableTargetAction *aasCapacityAction `json:"ScalableTargetAction"`
}

type aasNamedInput struct {
	aasTargetKey
	PolicyName          string `json:"PolicyName"`
	ScheduledActionName string `json:"ScheduledActionName"`
}

type aasDescribeInput struct {
	ServiceNamespace           string   `json:"ServiceNamespace"`
	ResourceId                 string   `json:"ResourceId"`
	ResourceIds                []string `json:"ResourceIds"`
	ScalableDimension          string   `json:"ScalableDimension"`
	PolicyNames                []string `json:"PolicyNames"`
	ScheduledActionNames       []string `json:"ScheduledActionNames"`
	IncludeNotScaledActivities bool     `json:"IncludeNotScaledActivities"`
	MaxResults                 *int32   `json:"MaxResults"`
	NextToken                  *string  `json:"NextToken"`
}

type aasForecastInput struct {
	aasTargetKey
	PolicyName string   `json:"PolicyName"`
	StartTime  *float64 `json:"StartTime"`
	EndTime    *float64 `json:"EndTime"`
}

type aasTagInput struct {
	ResourceARN string            `json:"ResourceARN"`
	Tags        map[string]string `json:"Tags"`
	TagKeys     []string          `json:"TagKeys"`
}

// Output shapes. Timestamps are epoch seconds, as the json protocol expects.

type aasSuspendedStateOut struct {
	DynamicScalingInSuspended  bool `json:"DynamicScalingInSuspended"`
	DynamicScalingOutSuspended bool `json:"DynamicScalingOutSuspended"`
	ScheduledScalingSuspended  bool `json:"ScheduledScalingSuspended"`
}

type aasTargetOut struct {
	aasTargetKey
	MinCapacity       int32                `json:"MinCapacity"`
	MaxCapacity       int32                `json:"MaxCapacity"`
	RoleARN           string               `json:"RoleARN"`
	CreationTime      float64              `json:"CreationTime"`
	SuspendedState    aasSuspendedStateOut `json:"SuspendedState"`
	ScalableTargetARN string               `json:"ScalableTargetARN"`
}

type aasAlarm struct {
	AlarmName string `json:"AlarmName"`
	AlarmARN  string `json:"AlarmARN"`
}

type aasPolicyOut struct {
	PolicyARN  string `json:"PolicyARN"`
	PolicyName string `json:"PolicyName"`
	aasTargetKey
	PolicyType                               string                   `json:"PolicyType"`
	StepScalingPolicyConfiguration           *aasStepConfig           `json:"StepScalingPolicyConfiguration,omitempty"`
	TargetTrackingScalingPolicyConfiguration *aasTargetTrackingConfig `json:"TargetTrackingScalingPolicyConfiguration,omitempty"`
	PredictiveScalingPolicyConfiguration     *aasPredictiveConfig     `json:"PredictiveScalingPolicyConfiguration,omitempty"`
	Alarms                                   []aasAlarm               `json:"Alarms"`
	CreationTime                             float64                  `json:"CreationTime"`
}

type aasScheduledActionOut struct {
	ScheduledActionName  string             `json:"ScheduledActionName"`
	ScheduledActionARN   string             `json:"ScheduledActionARN"`
	ServiceNamespace     string             `json:"ServiceNamespace"`
	Schedule             string             `json:"Schedule"`
	Timezone             string             `json:"Timezone,omitempty"`
	ResourceId           string             `json:"ResourceId"`
	ScalableDimension    string             `json:"ScalableDimension,omitempty"`
	StartTime            *float64           `json:"StartTime,omitempty"`
	EndTime              *float64           `json:"EndTime,omitempty"`
	ScalableTargetAction *aasCapacityAction `json:"ScalableTargetAction,omitempty"`
	CreationTime         float64            `json:"CreationTime"`
}

type aasNotScaledReason struct {
	Code            string `json:"Code"`
	MaxCapacity     *int32 `json:"MaxCapacity,omitempty"`
	MinCapacity     *int32 `json:"MinCapacity,omitempty"`
	CurrentCapacity *int32 `json:"CurrentCapacity,omitempty"`
}

type aasActivityOut struct {
	ActivityId string `json:"ActivityId"`
	aasTargetKey
	Description      string               `json:"Description"`
	Cause            string               `json:"Cause"`
	StartTime        float64              `json:"StartTime"`
	EndTime          float64              `json:"EndTime,omitempty"`
	StatusCode       string               `json:"StatusCode"`
	StatusMessage    string               `json:"StatusMessage,omitempty"`
	NotScaledReasons []aasNotScaledReason `json:"NotScaledReasons,omitempty"`
}
