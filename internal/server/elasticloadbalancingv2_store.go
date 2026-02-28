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

const elasticLoadBalancingV2DefaultLoadBalancerName = "stackyard-classic-elb"

type elasticLoadBalancingV2Store struct {
	mu sync.Mutex

	nextID int64

	loadBalancers map[string]map[string]any
	attributes    map[string]map[string]any
	healthChecks  map[string]map[string]any
	instances     map[string]map[string]struct{}
	tags          map[string]map[string]string
	policies      map[string]map[string]any
}

func newElasticLoadBalancingV2Store() *elasticLoadBalancingV2Store {
	s := &elasticLoadBalancingV2Store{
		nextID:        1,
		loadBalancers: map[string]map[string]any{},
		attributes:    map[string]map[string]any{},
		healthChecks:  map[string]map[string]any{},
		instances:     map[string]map[string]struct{}{},
		tags:          map[string]map[string]string{},
		policies:      map[string]map[string]any{},
	}

	s.ensureLoadBalancerLocked(elasticLoadBalancingV2DefaultLoadBalancerName)
	return s
}

func (s *elasticLoadBalancingV2Store) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		return map[string]any{"DNSName": fmt.Sprintf("%v", lb["DNSName"])}
	case "DeleteLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		delete(s.loadBalancers, name)
		delete(s.attributes, name)
		delete(s.healthChecks, name)
		delete(s.instances, name)
		delete(s.tags, name)
		return map[string]any{}
	case "DescribeLoadBalancers":
		names := elasticLoadBalancingFormMembers(form, "LoadBalancerNames.member.")
		out := make([]any, 0, len(s.loadBalancers))
		if len(names) == 0 {
			for _, name := range elasticLoadBalancingSortedKeys(s.loadBalancers) {
				out = append(out, elasticLoadBalancingCloneMap(s.loadBalancers[name]))
			}
		} else {
			for _, name := range names {
				out = append(out, elasticLoadBalancingCloneMap(s.ensureLoadBalancerLocked(name)))
			}
		}
		return map[string]any{"LoadBalancerDescriptions": out}
	case "DescribeLoadBalancerAttributes":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		return map[string]any{"LoadBalancerAttributes": elasticLoadBalancingCloneMap(s.attributes[name])}
	case "ModifyLoadBalancerAttributes":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		return map[string]any{"LoadBalancerAttributes": elasticLoadBalancingCloneMap(s.attributes[name])}
	case "DescribeAccountLimits":
		return map[string]any{
			"Limits": []any{
				map[string]any{"Name": "classic-load-balancers", "Max": "20"},
				map[string]any{"Name": "classic-load-balancer-listeners", "Max": "100"},
			},
		}
	case "DescribeTags":
		names := elasticLoadBalancingFormMembers(form, "LoadBalancerNames.member.")
		if len(names) == 0 {
			names = elasticLoadBalancingSortedKeys(s.loadBalancers)
		}
		out := make([]any, 0, len(names))
		for _, name := range names {
			s.ensureLoadBalancerLocked(name)
			tagMap := s.tags[name]
			tagList := make([]any, 0, len(tagMap))
			for _, key := range elasticLoadBalancingSortedKeys(tagMap) {
				tagList = append(tagList, map[string]any{"Key": key, "Value": tagMap[key]})
			}
			out = append(out, map[string]any{
				"LoadBalancerName": name,
				"Tags":             tagList,
			})
		}
		return map[string]any{"TagDescriptions": out}
	case "AddTags":
		names := elasticLoadBalancingFormMembers(form, "LoadBalancerNames.member.")
		if len(names) == 0 {
			names = []string{elasticLoadBalancingV2DefaultLoadBalancerName}
		}
		tags := elasticLoadBalancingFormTags(form)
		if len(tags) == 0 {
			tags = map[string]string{"managed-by": "stackyard"}
		}
		for _, name := range names {
			s.ensureLoadBalancerLocked(name)
			existing := s.tags[name]
			for k, v := range tags {
				existing[k] = v
			}
		}
		return map[string]any{}
	case "RemoveTags":
		names := elasticLoadBalancingFormMembers(form, "LoadBalancerNames.member.")
		if len(names) == 0 {
			names = []string{elasticLoadBalancingV2DefaultLoadBalancerName}
		}
		keys := elasticLoadBalancingFormMembers(form, "Tags.member.")
		for _, name := range names {
			s.ensureLoadBalancerLocked(name)
			existing := s.tags[name]
			for _, key := range keys {
				delete(existing, key)
			}
		}
		return map[string]any{}

	case "ConfigureHealthCheck":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		healthCheck := map[string]any{
			"Target":             elasticLoadBalancingFormString(form, "HealthCheck.Target", "HTTP:80/"),
			"Interval":           elasticLoadBalancingV2ParseInt(elasticLoadBalancingFormString(form, "HealthCheck.Interval", "30"), 30),
			"Timeout":            elasticLoadBalancingV2ParseInt(elasticLoadBalancingFormString(form, "HealthCheck.Timeout", "5"), 5),
			"UnhealthyThreshold": elasticLoadBalancingV2ParseInt(elasticLoadBalancingFormString(form, "HealthCheck.UnhealthyThreshold", "2"), 2),
			"HealthyThreshold":   elasticLoadBalancingV2ParseInt(elasticLoadBalancingFormString(form, "HealthCheck.HealthyThreshold", "10"), 10),
		}
		s.healthChecks[name] = healthCheck
		return map[string]any{"HealthCheck": elasticLoadBalancingCloneMap(healthCheck)}
	case "RegisterInstancesWithLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		instanceSet := s.instances[name]
		for _, instanceID := range elasticLoadBalancingV2FormInstanceIDs(form) {
			instanceSet[instanceID] = struct{}{}
		}
		if len(instanceSet) == 0 {
			instanceSet["i-0123456789abcdef0"] = struct{}{}
		}
		return map[string]any{"Instances": elasticLoadBalancingV2InstancesFromSet(instanceSet)}
	case "DeregisterInstancesFromLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		instanceSet := s.instances[name]
		for _, instanceID := range elasticLoadBalancingV2FormInstanceIDs(form) {
			delete(instanceSet, instanceID)
		}
		return map[string]any{"Instances": elasticLoadBalancingV2InstancesFromSet(instanceSet)}
	case "DescribeInstanceHealth":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		s.ensureLoadBalancerLocked(name)
		states := make([]any, 0)
		for _, instance := range elasticLoadBalancingV2InstancesFromSet(s.instances[name]) {
			if m, ok := instance.(map[string]any); ok {
				states = append(states, map[string]any{
					"InstanceId":  m["InstanceId"],
					"State":       "InService",
					"ReasonCode":  "N/A",
					"Description": "stackyard",
				})
			}
		}
		return map[string]any{"InstanceStates": states}

	case "CreateLoadBalancerListeners", "DeleteLoadBalancerListeners", "SetLoadBalancerListenerSSLCertificate", "SetLoadBalancerPoliciesForBackendServer", "SetLoadBalancerPoliciesOfListener":
		return map[string]any{}

	case "CreateAppCookieStickinessPolicy":
		name := elasticLoadBalancingFormString(form, "PolicyName", "stackyard-app-cookie-policy")
		s.policies[name] = map[string]any{
			"PolicyName":     name,
			"PolicyTypeName": "AppCookieStickinessPolicyType",
			"PolicyAttributeDescriptions": []any{
				map[string]any{"AttributeName": "CookieName", "AttributeValue": elasticLoadBalancingFormString(form, "CookieName", "stackyard")},
			},
		}
		return map[string]any{}
	case "CreateLBCookieStickinessPolicy":
		name := elasticLoadBalancingFormString(form, "PolicyName", "stackyard-lb-cookie-policy")
		s.policies[name] = map[string]any{
			"PolicyName":     name,
			"PolicyTypeName": "LBCookieStickinessPolicyType",
			"PolicyAttributeDescriptions": []any{
				map[string]any{"AttributeName": "CookieExpirationPeriod", "AttributeValue": elasticLoadBalancingFormString(form, "CookieExpirationPeriod", "60")},
			},
		}
		return map[string]any{}
	case "CreateLoadBalancerPolicy":
		name := elasticLoadBalancingFormString(form, "PolicyName", "stackyard-custom-policy")
		s.policies[name] = map[string]any{
			"PolicyName":                  name,
			"PolicyTypeName":              elasticLoadBalancingFormString(form, "PolicyTypeName", "SSLNegotiationPolicyType"),
			"PolicyAttributeDescriptions": []any{},
		}
		return map[string]any{}
	case "DeleteLoadBalancerPolicy":
		name := elasticLoadBalancingFormString(form, "PolicyName", "")
		delete(s.policies, name)
		return map[string]any{}
	case "DescribeLoadBalancerPolicies":
		names := elasticLoadBalancingFormMembers(form, "PolicyNames.member.")
		out := make([]any, 0)
		if len(names) == 0 {
			for _, name := range elasticLoadBalancingSortedKeys(s.policies) {
				out = append(out, elasticLoadBalancingCloneMap(s.policies[name]))
			}
		} else {
			for _, name := range names {
				if p, ok := s.policies[name]; ok {
					out = append(out, elasticLoadBalancingCloneMap(p))
				}
			}
		}
		return map[string]any{"PolicyDescriptions": out}
	case "DescribeLoadBalancerPolicyTypes":
		return map[string]any{
			"PolicyTypeDescriptions": []any{
				map[string]any{
					"PolicyTypeName": "SSLNegotiationPolicyType",
					"Description":    "Stackyard generated policy type",
					"PolicyAttributeTypeDescriptions": []any{
						map[string]any{"AttributeName": "Protocol-TLSv1.2", "AttributeType": "Boolean", "Cardinality": "ZERO_OR_ONE"},
					},
				},
			},
		}

	case "ApplySecurityGroupsToLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		securityGroups := elasticLoadBalancingFormMembers(form, "SecurityGroups.member.")
		if len(securityGroups) == 0 {
			securityGroups = []string{"sg-0123456789abcdef0"}
		}
		lb["SecurityGroups"] = elasticLoadBalancingV2StringsToAny(securityGroups)
		return map[string]any{"SecurityGroups": elasticLoadBalancingV2StringsToAny(securityGroups)}
	case "AttachLoadBalancerToSubnets":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		set := elasticLoadBalancingV2StringSetFromAny(lb["Subnets"])
		for _, subnet := range elasticLoadBalancingFormMembers(form, "Subnets.member.") {
			set[subnet] = struct{}{}
		}
		subnets := elasticLoadBalancingV2SortedSet(set)
		lb["Subnets"] = elasticLoadBalancingV2StringsToAny(subnets)
		return map[string]any{"Subnets": elasticLoadBalancingV2StringsToAny(subnets)}
	case "DetachLoadBalancerFromSubnets":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		set := elasticLoadBalancingV2StringSetFromAny(lb["Subnets"])
		for _, subnet := range elasticLoadBalancingFormMembers(form, "Subnets.member.") {
			delete(set, subnet)
		}
		subnets := elasticLoadBalancingV2SortedSet(set)
		lb["Subnets"] = elasticLoadBalancingV2StringsToAny(subnets)
		return map[string]any{"Subnets": elasticLoadBalancingV2StringsToAny(subnets)}
	case "EnableAvailabilityZonesForLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		set := elasticLoadBalancingV2StringSetFromAny(lb["AvailabilityZones"])
		for _, zone := range elasticLoadBalancingFormMembers(form, "AvailabilityZones.member.") {
			set[zone] = struct{}{}
		}
		zones := elasticLoadBalancingV2SortedSet(set)
		lb["AvailabilityZones"] = elasticLoadBalancingV2StringsToAny(zones)
		return map[string]any{"AvailabilityZones": elasticLoadBalancingV2StringsToAny(zones)}
	case "DisableAvailabilityZonesForLoadBalancer":
		name := elasticLoadBalancingFormString(form, "LoadBalancerName", elasticLoadBalancingV2DefaultLoadBalancerName)
		lb := s.ensureLoadBalancerLocked(name)
		set := elasticLoadBalancingV2StringSetFromAny(lb["AvailabilityZones"])
		for _, zone := range elasticLoadBalancingFormMembers(form, "AvailabilityZones.member.") {
			delete(set, zone)
		}
		if len(set) == 0 {
			set["us-east-1a"] = struct{}{}
		}
		zones := elasticLoadBalancingV2SortedSet(set)
		lb["AvailabilityZones"] = elasticLoadBalancingV2StringsToAny(zones)
		return map[string]any{"AvailabilityZones": elasticLoadBalancingV2StringsToAny(zones)}
	}

	switch {
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{}
	case strings.HasPrefix(action, "Create"):
		return map[string]any{}
	case strings.HasPrefix(action, "Delete"),
		strings.HasPrefix(action, "Modify"),
		strings.HasPrefix(action, "Configure"),
		strings.HasPrefix(action, "Register"),
		strings.HasPrefix(action, "Deregister"),
		strings.HasPrefix(action, "Apply"),
		strings.HasPrefix(action, "Attach"),
		strings.HasPrefix(action, "Detach"),
		strings.HasPrefix(action, "Enable"),
		strings.HasPrefix(action, "Disable"),
		strings.HasPrefix(action, "Add"),
		strings.HasPrefix(action, "Remove"),
		strings.HasPrefix(action, "Set"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *elasticLoadBalancingV2Store) ensureLoadBalancerLocked(name string) map[string]any {
	if name == "" {
		name = elasticLoadBalancingV2DefaultLoadBalancerName
	}
	if lb, ok := s.loadBalancers[name]; ok {
		return lb
	}

	suffix := s.nextTokenLocked(12)
	lb := map[string]any{
		"LoadBalancerName":          name,
		"DNSName":                   fmt.Sprintf("%s-%s.us-east-1.elb.amazonaws.com", name, suffix),
		"CanonicalHostedZoneName":   fmt.Sprintf("%s-%s.us-east-1.elb.amazonaws.com", name, suffix),
		"CanonicalHostedZoneNameID": "Z35SXDOTRQ7X7K",
		"CreatedTime":               time.Now().UTC(),
		"Scheme":                    "internet-facing",
		"AvailabilityZones":         []any{"us-east-1a", "us-east-1b"},
		"Subnets":                   []any{"subnet-0123456789abcdef0", "subnet-0fedcba9876543210"},
		"SecurityGroups":            []any{"sg-0123456789abcdef0"},
		"Instances":                 []any{map[string]any{"InstanceId": "i-0123456789abcdef0"}},
		"ListenerDescriptions": []any{
			map[string]any{
				"Listener": map[string]any{
					"Protocol":         "HTTP",
					"LoadBalancerPort": 80,
					"InstanceProtocol": "HTTP",
					"InstancePort":     80,
				},
				"PolicyNames": []any{},
			},
		},
		"Policies": map[string]any{
			"AppCookieStickinessPolicies": []any{},
			"LBCookieStickinessPolicies":  []any{},
			"OtherPolicies":               []any{},
		},
		"SourceSecurityGroup": map[string]any{
			"OwnerAlias": "123456789012",
			"GroupName":  "default",
		},
	}
	s.loadBalancers[name] = lb
	s.instances[name] = map[string]struct{}{"i-0123456789abcdef0": {}}
	s.tags[name] = map[string]string{"Name": name}
	s.healthChecks[name] = map[string]any{
		"Target":             "HTTP:80/",
		"Interval":           30,
		"Timeout":            5,
		"UnhealthyThreshold": 2,
		"HealthyThreshold":   10,
	}
	s.attributes[name] = map[string]any{
		"CrossZoneLoadBalancing": map[string]any{"Enabled": true},
		"ConnectionDraining":     map[string]any{"Enabled": false, "Timeout": 300},
		"ConnectionSettings":     map[string]any{"IdleTimeout": 60},
		"AccessLog":              map[string]any{"Enabled": false, "EmitInterval": 60},
		"AdditionalAttributes":   []any{},
	}
	return lb
}

func (s *elasticLoadBalancingV2Store) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%0*x", width, id)
}

func elasticLoadBalancingV2ParseInt(raw string, fallback int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func elasticLoadBalancingV2FormInstanceIDs(form url.Values) []string {
	values := map[int]string{}
	for key, entries := range form {
		if len(entries) == 0 || !strings.HasPrefix(key, "Instances.member.") || !strings.HasSuffix(key, ".InstanceId") {
			continue
		}
		rest := strings.TrimSuffix(strings.TrimPrefix(key, "Instances.member."), ".InstanceId")
		idx, err := strconv.Atoi(rest)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(entries[0])
		if value == "" {
			continue
		}
		values[idx] = value
	}
	indices := make([]int, 0, len(values))
	for idx := range values {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]string, 0, len(indices))
	for _, idx := range indices {
		out = append(out, values[idx])
	}
	return out
}

func elasticLoadBalancingV2InstancesFromSet(set map[string]struct{}) []any {
	out := make([]any, 0, len(set))
	for _, instanceID := range elasticLoadBalancingV2SortedSet(set) {
		out = append(out, map[string]any{"InstanceId": instanceID})
	}
	return out
}

func elasticLoadBalancingV2StringSetFromAny(v any) map[string]struct{} {
	out := map[string]struct{}{}
	switch values := v.(type) {
	case []any:
		for _, item := range values {
			value := strings.TrimSpace(fmt.Sprintf("%v", item))
			if value != "" {
				out[value] = struct{}{}
			}
		}
	case []string:
		for _, item := range values {
			value := strings.TrimSpace(item)
			if value != "" {
				out[value] = struct{}{}
			}
		}
	}
	return out
}

func elasticLoadBalancingV2SortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func elasticLoadBalancingV2StringsToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
