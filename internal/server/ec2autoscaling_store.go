package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ec2AutoScalingDefaultGroupName       = "stackyard-asg"
	ec2AutoScalingDefaultLaunchConfig    = "stackyard-lc"
	ec2AutoScalingDefaultLifecycleHook   = "stackyard-lifecycle-hook"
	ec2AutoScalingDefaultPolicyName      = "stackyard-scaling-policy"
	ec2AutoScalingDefaultScheduledAction = "stackyard-scheduled-action"
)

type ec2AutoScalingStore struct {
	mu sync.Mutex

	nextID int64

	groups               map[string]map[string]any
	launchConfigurations map[string]map[string]any
	lifecycleHooks       map[string]map[string]map[string]any
	scalingPolicies      map[string]map[string]map[string]any
	scheduledActions     map[string]map[string]map[string]any
	warmPools            map[string]map[string]any
	loadBalancers        map[string]map[string]struct{}
	targetGroups         map[string]map[string]struct{}
	trafficSources       map[string]map[string]struct{}
	suspendedProcesses   map[string]map[string]struct{}
	instanceRefreshes    map[string][]map[string]any
	tags                 map[string]map[string]string
	scalingActivities    []map[string]any
}

func newEC2AutoScalingStore() *ec2AutoScalingStore {
	s := &ec2AutoScalingStore{
		nextID:               2,
		groups:               map[string]map[string]any{},
		launchConfigurations: map[string]map[string]any{},
		lifecycleHooks:       map[string]map[string]map[string]any{},
		scalingPolicies:      map[string]map[string]map[string]any{},
		scheduledActions:     map[string]map[string]map[string]any{},
		warmPools:            map[string]map[string]any{},
		loadBalancers:        map[string]map[string]struct{}{},
		targetGroups:         map[string]map[string]struct{}{},
		trafficSources:       map[string]map[string]struct{}{},
		suspendedProcesses:   map[string]map[string]struct{}{},
		instanceRefreshes:    map[string][]map[string]any{},
		tags:                 map[string]map[string]string{},
		scalingActivities:    []map[string]any{},
	}

	s.launchConfigurations[ec2AutoScalingDefaultLaunchConfig] = map[string]any{
		"LaunchConfigurationName": ec2AutoScalingDefaultLaunchConfig,
		"ImageId":                 "ami-0123456789abcdef0",
		"InstanceType":            "t3.micro",
		"CreatedTime":             time.Now().UTC(),
	}
	_ = s.ensureGroupLocked(ec2AutoScalingDefaultGroupName)
	return s
}

func (s *ec2AutoScalingStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateLaunchConfiguration":
		name := ec2AutoScalingFormString(form, "LaunchConfigurationName", ec2AutoScalingDefaultLaunchConfig)
		s.launchConfigurations[name] = map[string]any{
			"LaunchConfigurationName": name,
			"ImageId":                 ec2AutoScalingFormString(form, "ImageId", "ami-0123456789abcdef0"),
			"InstanceType":            ec2AutoScalingFormString(form, "InstanceType", "t3.micro"),
			"CreatedTime":             time.Now().UTC(),
		}
		return map[string]any{}
	case "DeleteLaunchConfiguration":
		delete(s.launchConfigurations, ec2AutoScalingFormString(form, "LaunchConfigurationName", ec2AutoScalingDefaultLaunchConfig))
		return map[string]any{}
	case "DescribeLaunchConfigurations":
		items := make([]any, 0, len(s.launchConfigurations))
		keys := make([]string, 0, len(s.launchConfigurations))
		for name := range s.launchConfigurations {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			items = append(items, ec2AutoScalingCloneMap(s.launchConfigurations[name]))
		}
		return map[string]any{"LaunchConfigurations": items, "NextToken": ""}

	case "CreateAutoScalingGroup":
		name := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		group := s.ensureGroupLocked(name)
		if lc := ec2AutoScalingFormString(form, "LaunchConfigurationName", ""); lc != "" {
			group["LaunchConfigurationName"] = lc
		}
		group["MinSize"] = ec2AutoScalingFormInt(form, "MinSize", ec2AutoScalingFormIntFromAny(group["MinSize"], 1))
		group["MaxSize"] = ec2AutoScalingFormInt(form, "MaxSize", ec2AutoScalingFormIntFromAny(group["MaxSize"], 3))
		group["DesiredCapacity"] = ec2AutoScalingFormInt(form, "DesiredCapacity", ec2AutoScalingFormIntFromAny(group["DesiredCapacity"], 1))
		s.addActivityLocked(name, "Created Auto Scaling group")
		return map[string]any{}
	case "UpdateAutoScalingGroup":
		name := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		group := s.ensureGroupLocked(name)
		if v := strings.TrimSpace(form.Get("LaunchConfigurationName")); v != "" {
			group["LaunchConfigurationName"] = v
		}
		if v := strings.TrimSpace(form.Get("MinSize")); v != "" {
			group["MinSize"] = ec2AutoScalingParseInt(v, ec2AutoScalingFormIntFromAny(group["MinSize"], 1))
		}
		if v := strings.TrimSpace(form.Get("MaxSize")); v != "" {
			group["MaxSize"] = ec2AutoScalingParseInt(v, ec2AutoScalingFormIntFromAny(group["MaxSize"], 3))
		}
		if v := strings.TrimSpace(form.Get("DesiredCapacity")); v != "" {
			group["DesiredCapacity"] = ec2AutoScalingParseInt(v, ec2AutoScalingFormIntFromAny(group["DesiredCapacity"], 1))
		}
		s.addActivityLocked(name, "Updated Auto Scaling group")
		return map[string]any{}
	case "DeleteAutoScalingGroup":
		name := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		delete(s.groups, name)
		delete(s.lifecycleHooks, name)
		delete(s.scalingPolicies, name)
		delete(s.scheduledActions, name)
		delete(s.warmPools, name)
		delete(s.loadBalancers, name)
		delete(s.targetGroups, name)
		delete(s.trafficSources, name)
		delete(s.suspendedProcesses, name)
		delete(s.instanceRefreshes, name)
		s.addActivityLocked(name, "Deleted Auto Scaling group")
		return map[string]any{}
	case "DescribeAutoScalingGroups":
		items := make([]any, 0, len(s.groups))
		keys := make([]string, 0, len(s.groups))
		for name := range s.groups {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			items = append(items, ec2AutoScalingCloneMap(s.groups[name]))
		}
		return map[string]any{"AutoScalingGroups": items, "NextToken": ""}
	case "DescribeAutoScalingInstances":
		items := make([]any, 0)
		for _, group := range s.groups {
			instances, _ := group["Instances"].([]any)
			for _, inst := range instances {
				if entry, ok := inst.(map[string]any); ok {
					items = append(items, ec2AutoScalingCloneMap(entry))
				}
			}
		}
		return map[string]any{"AutoScalingInstances": items, "NextToken": ""}
	case "SetDesiredCapacity":
		name := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		group := s.ensureGroupLocked(name)
		desired := ec2AutoScalingFormInt(form, "DesiredCapacity", ec2AutoScalingFormIntFromAny(group["DesiredCapacity"], 1))
		group["DesiredCapacity"] = desired
		s.reconcileInstancesLocked(group)
		s.addActivityLocked(name, fmt.Sprintf("Set desired capacity to %d", desired))
		return map[string]any{}
	case "AttachInstances", "DetachInstances", "EnterStandby", "ExitStandby", "SetInstanceHealth", "SetInstanceProtection", "LaunchInstances", "TerminateInstanceInAutoScalingGroup":
		name := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		group := s.ensureGroupLocked(name)
		s.reconcileInstancesLocked(group)
		s.addActivityLocked(name, action)
		if action == "TerminateInstanceInAutoScalingGroup" {
			return map[string]any{"Activity": map[string]any{"Description": "Terminated instance", "StatusCode": "Successful"}}
		}
		return map[string]any{}

	case "PutLifecycleHook":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		hookName := ec2AutoScalingFormString(form, "LifecycleHookName", ec2AutoScalingDefaultLifecycleHook)
		hooks := s.ensureLifecycleHooksLocked(groupName)
		hooks[hookName] = map[string]any{
			"AutoScalingGroupName": groupName,
			"LifecycleHookName":    hookName,
			"LifecycleTransition":  ec2AutoScalingFormString(form, "LifecycleTransition", "autoscaling:EC2_INSTANCE_LAUNCHING"),
		}
		return map[string]any{}
	case "DeleteLifecycleHook":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		hookName := ec2AutoScalingFormString(form, "LifecycleHookName", ec2AutoScalingDefaultLifecycleHook)
		if hooks := s.lifecycleHooks[groupName]; hooks != nil {
			delete(hooks, hookName)
		}
		return map[string]any{}
	case "DescribeLifecycleHooks":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		hooks := s.ensureLifecycleHooksLocked(groupName)
		out := make([]any, 0, len(hooks))
		keys := make([]string, 0, len(hooks))
		for name := range hooks {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			out = append(out, ec2AutoScalingCloneMap(hooks[name]))
		}
		return map[string]any{"LifecycleHooks": out}
	case "DescribeLifecycleHookTypes":
		return map[string]any{"LifecycleHookTypes": []any{"autoscaling:EC2_INSTANCE_LAUNCHING", "autoscaling:EC2_INSTANCE_TERMINATING"}}
	case "CompleteLifecycleAction", "RecordLifecycleActionHeartbeat":
		return map[string]any{}

	case "PutScalingPolicy":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		policyName := ec2AutoScalingFormString(form, "PolicyName", ec2AutoScalingDefaultPolicyName)
		policies := s.ensureScalingPoliciesLocked(groupName)
		policyARN := fmt.Sprintf("arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:%s:autoScalingGroupName/%s:policyName/%s", s.nextIdentifierLocked("policy"), groupName, policyName)
		policies[policyName] = map[string]any{"AutoScalingGroupName": groupName, "PolicyName": policyName, "PolicyARN": policyARN}
		return map[string]any{"PolicyARN": policyARN, "Alarms": []any{}}
	case "DeletePolicy":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		policyName := ec2AutoScalingFormString(form, "PolicyName", ec2AutoScalingDefaultPolicyName)
		if policies := s.scalingPolicies[groupName]; policies != nil {
			delete(policies, policyName)
		}
		return map[string]any{}
	case "DescribePolicies":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		policies := s.ensureScalingPoliciesLocked(groupName)
		out := make([]any, 0, len(policies))
		keys := make([]string, 0, len(policies))
		for name := range policies {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			out = append(out, ec2AutoScalingCloneMap(policies[name]))
		}
		return map[string]any{"ScalingPolicies": out, "NextToken": ""}
	case "ExecutePolicy":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		s.addActivityLocked(groupName, "Executed scaling policy")
		return map[string]any{}

	case "PutScheduledUpdateGroupAction", "BatchPutScheduledUpdateGroupAction":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		actionName := ec2AutoScalingFormString(form, "ScheduledActionName", ec2AutoScalingDefaultScheduledAction)
		actions := s.ensureScheduledActionsLocked(groupName)
		actions[actionName] = map[string]any{"AutoScalingGroupName": groupName, "ScheduledActionName": actionName}
		if action == "BatchPutScheduledUpdateGroupAction" {
			return map[string]any{"FailedScheduledUpdateGroupActions": []any{}}
		}
		return map[string]any{}
	case "DeleteScheduledAction", "BatchDeleteScheduledAction":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		actionName := ec2AutoScalingFormString(form, "ScheduledActionName", ec2AutoScalingDefaultScheduledAction)
		if actions := s.scheduledActions[groupName]; actions != nil {
			delete(actions, actionName)
		}
		if action == "BatchDeleteScheduledAction" {
			return map[string]any{"FailedScheduledActions": []any{}}
		}
		return map[string]any{}
	case "DescribeScheduledActions":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		actions := s.ensureScheduledActionsLocked(groupName)
		out := make([]any, 0, len(actions))
		keys := make([]string, 0, len(actions))
		for name := range actions {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			out = append(out, ec2AutoScalingCloneMap(actions[name]))
		}
		return map[string]any{"ScheduledUpdateGroupActions": out, "NextToken": ""}

	case "PutWarmPool":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		s.warmPools[groupName] = map[string]any{
			"AutoScalingGroupName": groupName,
			"MinSize":              ec2AutoScalingFormInt(form, "MinSize", 0),
			"PoolState":            ec2AutoScalingFormString(form, "PoolState", "Stopped"),
		}
		return map[string]any{}
	case "DeleteWarmPool":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		delete(s.warmPools, groupName)
		return map[string]any{}
	case "DescribeWarmPool":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		pool := s.warmPools[groupName]
		if pool == nil {
			pool = map[string]any{"AutoScalingGroupName": groupName, "PoolState": "Stopped", "MinSize": 0}
		}
		return map[string]any{"WarmPoolConfiguration": pool, "Instances": []any{}, "NextToken": ""}

	case "AttachLoadBalancers":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.loadBalancers, groupName)
		for _, lb := range ec2AutoScalingFormSlice(form, "LoadBalancerNames.member") {
			set[lb] = struct{}{}
		}
		return map[string]any{}
	case "DetachLoadBalancers":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.loadBalancers, groupName)
		for _, lb := range ec2AutoScalingFormSlice(form, "LoadBalancerNames.member") {
			delete(set, lb)
		}
		return map[string]any{}
	case "DescribeLoadBalancers":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.loadBalancers, groupName)
		out := make([]any, 0, len(set))
		for _, name := range ec2AutoScalingSortedSet(set) {
			out = append(out, map[string]any{"LoadBalancerName": name, "State": "Added"})
		}
		return map[string]any{"LoadBalancers": out, "NextToken": ""}

	case "AttachLoadBalancerTargetGroups":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.targetGroups, groupName)
		for _, arn := range ec2AutoScalingFormSlice(form, "TargetGroupARNs.member") {
			set[arn] = struct{}{}
		}
		return map[string]any{}
	case "DetachLoadBalancerTargetGroups":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.targetGroups, groupName)
		for _, arn := range ec2AutoScalingFormSlice(form, "TargetGroupARNs.member") {
			delete(set, arn)
		}
		return map[string]any{}
	case "DescribeLoadBalancerTargetGroups":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.targetGroups, groupName)
		out := make([]any, 0, len(set))
		for _, arn := range ec2AutoScalingSortedSet(set) {
			out = append(out, map[string]any{"LoadBalancerTargetGroupARN": arn, "State": "Added"})
		}
		return map[string]any{"LoadBalancerTargetGroups": out, "NextToken": ""}

	case "AttachTrafficSources":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.trafficSources, groupName)
		for _, src := range ec2AutoScalingFormSlice(form, "TrafficSources.member") {
			set[src] = struct{}{}
		}
		return map[string]any{}
	case "DetachTrafficSources":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.trafficSources, groupName)
		for _, src := range ec2AutoScalingFormSlice(form, "TrafficSources.member") {
			delete(set, src)
		}
		return map[string]any{}
	case "DescribeTrafficSources":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.trafficSources, groupName)
		out := make([]any, 0, len(set))
		for _, src := range ec2AutoScalingSortedSet(set) {
			out = append(out, map[string]any{"Identifier": src, "State": "Added"})
		}
		return map[string]any{"TrafficSources": out, "NextToken": ""}

	case "SuspendProcesses":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.suspendedProcesses, groupName)
		for _, proc := range ec2AutoScalingFormSlice(form, "ScalingProcesses.member") {
			set[proc] = struct{}{}
		}
		return map[string]any{}
	case "ResumeProcesses":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		set := s.ensureSetLocked(s.suspendedProcesses, groupName)
		for _, proc := range ec2AutoScalingFormSlice(form, "ScalingProcesses.member") {
			delete(set, proc)
		}
		return map[string]any{}
	case "DescribeScalingProcessTypes":
		return map[string]any{"Processes": []any{map[string]any{"ProcessName": "Launch"}, map[string]any{"ProcessName": "Terminate"}, map[string]any{"ProcessName": "HealthCheck"}}}

	case "StartInstanceRefresh":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		refreshID := s.nextIdentifierLocked("refresh")
		entry := map[string]any{
			"AutoScalingGroupName": groupName,
			"InstanceRefreshId":    refreshID,
			"Status":               "InProgress",
			"StartTime":            time.Now().UTC(),
		}
		s.instanceRefreshes[groupName] = append(s.instanceRefreshes[groupName], entry)
		return map[string]any{"InstanceRefreshId": refreshID}
	case "DescribeInstanceRefreshes":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		items := make([]any, 0, len(s.instanceRefreshes[groupName]))
		for _, r := range s.instanceRefreshes[groupName] {
			items = append(items, ec2AutoScalingCloneMap(r))
		}
		return map[string]any{"InstanceRefreshes": items, "NextToken": ""}
	case "CancelInstanceRefresh", "RollbackInstanceRefresh":
		groupName := ec2AutoScalingFormString(form, "AutoScalingGroupName", ec2AutoScalingDefaultGroupName)
		refreshes := s.instanceRefreshes[groupName]
		if len(refreshes) > 0 {
			refreshes[len(refreshes)-1]["Status"] = "Cancelled"
		}
		return map[string]any{}

	case "CreateOrUpdateTags":
		for idx := 1; idx <= 100; idx++ {
			keyBase := fmt.Sprintf("Tags.member.%d.", idx)
			groupName := strings.TrimSpace(form.Get(keyBase + "ResourceId"))
			if groupName == "" {
				continue
			}
			tagKey := strings.TrimSpace(form.Get(keyBase + "Key"))
			if tagKey == "" {
				continue
			}
			tagValue := strings.TrimSpace(form.Get(keyBase + "Value"))
			tags := s.ensureTagsLocked(groupName)
			tags[tagKey] = tagValue
		}
		return map[string]any{}
	case "DeleteTags":
		for idx := 1; idx <= 100; idx++ {
			keyBase := fmt.Sprintf("Tags.member.%d.", idx)
			groupName := strings.TrimSpace(form.Get(keyBase + "ResourceId"))
			if groupName == "" {
				continue
			}
			tagKey := strings.TrimSpace(form.Get(keyBase + "Key"))
			if tagKey == "" {
				continue
			}
			tags := s.ensureTagsLocked(groupName)
			delete(tags, tagKey)
		}
		return map[string]any{}
	case "DescribeTags":
		out := make([]any, 0)
		groupNames := make([]string, 0, len(s.tags))
		for groupName := range s.tags {
			groupNames = append(groupNames, groupName)
		}
		sort.Strings(groupNames)
		for _, groupName := range groupNames {
			tags := s.tags[groupName]
			keys := make([]string, 0, len(tags))
			for k := range tags {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				out = append(out, map[string]any{
					"ResourceId":   groupName,
					"ResourceType": "auto-scaling-group",
					"Key":          k,
					"Value":        tags[k],
				})
			}
		}
		return map[string]any{"Tags": out, "NextToken": ""}

	case "DescribeScalingActivities":
		items := make([]any, 0, len(s.scalingActivities))
		for _, a := range s.scalingActivities {
			items = append(items, ec2AutoScalingCloneMap(a))
		}
		return map[string]any{"Activities": items, "NextToken": ""}
	case "DescribeMetricCollectionTypes":
		return map[string]any{"Metrics": []any{map[string]any{"Metric": "GroupDesiredCapacity"}, map[string]any{"Metric": "GroupInServiceInstances"}}, "Granularities": []any{map[string]any{"Granularity": "1Minute"}}}
	case "DescribeAccountLimits":
		return map[string]any{"MaxNumberOfAutoScalingGroups": 200, "MaxNumberOfLaunchConfigurations": 200}
	case "DescribeAdjustmentTypes":
		return map[string]any{"AdjustmentTypes": []any{map[string]any{"AdjustmentType": "ChangeInCapacity"}, map[string]any{"AdjustmentType": "PercentChangeInCapacity"}}}
	case "DescribeAutoScalingNotificationTypes":
		return map[string]any{"AutoScalingNotificationTypes": []any{"autoscaling:EC2_INSTANCE_LAUNCH", "autoscaling:EC2_INSTANCE_TERMINATE"}}
	case "DescribeNotificationConfigurations":
		return map[string]any{"NotificationConfigurations": []any{}, "NextToken": ""}
	case "DescribeTerminationPolicyTypes":
		return map[string]any{"TerminationPolicyTypes": []any{"Default", "OldestInstance", "NewestInstance"}}
	case "PutNotificationConfiguration", "DeleteNotificationConfiguration":
		return map[string]any{}
	case "EnableMetricsCollection", "DisableMetricsCollection":
		return map[string]any{}
	case "GetPredictiveScalingForecast":
		return map[string]any{"LoadForecast": []any{}, "CapacityForecast": []any{}, "UpdateTime": time.Now().UTC()}
	}

	switch {
	case strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get"):
		return map[string]any{"NextToken": ""}
	case strings.HasPrefix(action, "List"):
		return map[string]any{"NextToken": ""}
	default:
		return map[string]any{}
	}
}

func (s *ec2AutoScalingStore) ensureGroupLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = ec2AutoScalingDefaultGroupName
	}
	if group := s.groups[name]; group != nil {
		return group
	}
	group := map[string]any{
		"AutoScalingGroupName":    name,
		"AutoScalingGroupARN":     fmt.Sprintf("arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:%s:autoScalingGroupName/%s", s.nextIdentifierLocked("asg"), name),
		"LaunchConfigurationName": ec2AutoScalingDefaultLaunchConfig,
		"MinSize":                 1,
		"MaxSize":                 3,
		"DesiredCapacity":         1,
		"AvailabilityZones":       []any{"us-east-1a"},
		"CreatedTime":             time.Now().UTC(),
		"Instances": []any{map[string]any{
			"InstanceId":              "i-00000000000000001",
			"AutoScalingGroupName":    name,
			"AvailabilityZone":        "us-east-1a",
			"LifecycleState":          "InService",
			"HealthStatus":            "Healthy",
			"ProtectedFromScaleIn":    false,
			"WeightedCapacity":        "1",
			"LaunchTemplate":          map[string]any{},
			"InstanceType":            "t3.micro",
			"AvailabilityZoneId":      "use1-az1",
			"HealthStatusTimestamp":   time.Now().UTC(),
			"LaunchTemplateOverrides": []any{},
		}},
	}
	s.groups[name] = group
	return group
}

func (s *ec2AutoScalingStore) reconcileInstancesLocked(group map[string]any) {
	desired := ec2AutoScalingFormIntFromAny(group["DesiredCapacity"], 1)
	if desired < 0 {
		desired = 0
	}
	name, _ := group["AutoScalingGroupName"].(string)
	instances := make([]any, 0, desired)
	for i := 0; i < desired; i++ {
		instances = append(instances, map[string]any{
			"InstanceId":           fmt.Sprintf("i-%017d", i+1),
			"AutoScalingGroupName": name,
			"AvailabilityZone":     "us-east-1a",
			"LifecycleState":       "InService",
			"HealthStatus":         "Healthy",
			"ProtectedFromScaleIn": false,
		})
	}
	group["Instances"] = instances
}

func (s *ec2AutoScalingStore) ensureLifecycleHooksLocked(groupName string) map[string]map[string]any {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = ec2AutoScalingDefaultGroupName
	}
	if hooks := s.lifecycleHooks[groupName]; hooks != nil {
		return hooks
	}
	hooks := map[string]map[string]any{}
	s.lifecycleHooks[groupName] = hooks
	return hooks
}

func (s *ec2AutoScalingStore) ensureScalingPoliciesLocked(groupName string) map[string]map[string]any {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = ec2AutoScalingDefaultGroupName
	}
	if policies := s.scalingPolicies[groupName]; policies != nil {
		return policies
	}
	policies := map[string]map[string]any{}
	s.scalingPolicies[groupName] = policies
	return policies
}

func (s *ec2AutoScalingStore) ensureScheduledActionsLocked(groupName string) map[string]map[string]any {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = ec2AutoScalingDefaultGroupName
	}
	if actions := s.scheduledActions[groupName]; actions != nil {
		return actions
	}
	actions := map[string]map[string]any{}
	s.scheduledActions[groupName] = actions
	return actions
}

func (s *ec2AutoScalingStore) ensureTagsLocked(groupName string) map[string]string {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = ec2AutoScalingDefaultGroupName
	}
	if tags := s.tags[groupName]; tags != nil {
		return tags
	}
	tags := map[string]string{}
	s.tags[groupName] = tags
	return tags
}

func (s *ec2AutoScalingStore) ensureSetLocked(target map[string]map[string]struct{}, groupName string) map[string]struct{} {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = ec2AutoScalingDefaultGroupName
	}
	if set := target[groupName]; set != nil {
		return set
	}
	set := map[string]struct{}{}
	target[groupName] = set
	return set
}

func (s *ec2AutoScalingStore) addActivityLocked(groupName, description string) {
	activity := map[string]any{
		"ActivityId":           s.nextIdentifierLocked("activity"),
		"AutoScalingGroupName": groupName,
		"Description":          description,
		"Cause":                "At(" + time.Now().UTC().Format(time.RFC3339) + ")",
		"StartTime":            time.Now().UTC(),
		"EndTime":              time.Now().UTC(),
		"StatusCode":           "Successful",
	}
	s.scalingActivities = append([]map[string]any{activity}, s.scalingActivities...)
	if len(s.scalingActivities) > 100 {
		s.scalingActivities = s.scalingActivities[:100]
	}
}

func (s *ec2AutoScalingStore) nextIdentifierLocked(prefix string) string {
	prefix = strings.Trim(strings.ToLower(prefix), "- ")
	if prefix == "" {
		prefix = "resource"
	}
	id := fmt.Sprintf("stackyard-%s-%06d", prefix, s.nextID)
	s.nextID++
	return id
}

func ec2AutoScalingFormString(form url.Values, key, fallback string) string {
	value := strings.TrimSpace(form.Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func ec2AutoScalingFormInt(form url.Values, key string, fallback int) int {
	value := strings.TrimSpace(form.Get(key))
	if value == "" {
		return fallback
	}
	return ec2AutoScalingParseInt(value, fallback)
}

func ec2AutoScalingFormIntFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		return ec2AutoScalingParseInt(v, fallback)
	default:
		return fallback
	}
}

func ec2AutoScalingParseInt(value string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return n
}

func ec2AutoScalingFormSlice(form url.Values, prefix string) []string {
	values := []string{}
	for key, list := range form {
		if key == prefix {
			for _, item := range list {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
			continue
		}
		if !strings.HasPrefix(key, prefix+".") || len(list) == 0 {
			continue
		}
		trimmed := strings.TrimSpace(list[0])
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func ec2AutoScalingCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = ec2AutoScalingCloneAny(v)
	}
	return out
}

func ec2AutoScalingCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return ec2AutoScalingCloneMap(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, ec2AutoScalingCloneAny(item))
		}
		return out
	default:
		return v
	}
}

func ec2AutoScalingSortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
