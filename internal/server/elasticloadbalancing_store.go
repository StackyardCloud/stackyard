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
	elasticLoadBalancingDefaultLoadBalancerName = "stackyard"
	elasticLoadBalancingDefaultTargetGroupName  = "stackyard-targets"
	elasticLoadBalancingDefaultTrustStoreName   = "stackyard-truststore"
)

type elasticLoadBalancingStore struct {
	mu sync.Mutex

	nextID int64

	loadBalancers   map[string]map[string]any
	listeners       map[string]map[string]any
	rules           map[string]map[string]any
	targetGroups    map[string]map[string]any
	trustStores     map[string]map[string]any
	resourcePolicy  map[string]string
	resourceTags    map[string]map[string]string
	targetGroupTags map[string][]map[string]any
}

func newElasticLoadBalancingStore() *elasticLoadBalancingStore {
	s := &elasticLoadBalancingStore{
		nextID:          1,
		loadBalancers:   map[string]map[string]any{},
		listeners:       map[string]map[string]any{},
		rules:           map[string]map[string]any{},
		targetGroups:    map[string]map[string]any{},
		trustStores:     map[string]map[string]any{},
		resourcePolicy:  map[string]string{},
		resourceTags:    map[string]map[string]string{},
		targetGroupTags: map[string][]map[string]any{},
	}

	lb := s.createLoadBalancerLocked(elasticLoadBalancingDefaultLoadBalancerName)
	tg := s.createTargetGroupLocked(elasticLoadBalancingDefaultTargetGroupName)
	listener := s.createListenerLocked(elasticLoadBalancingDefaultLoadBalancerName, lb["LoadBalancerArn"].(string), tg["TargetGroupArn"].(string))
	s.createRuleLocked(elasticLoadBalancingDefaultLoadBalancerName, listener["ListenerArn"].(string), tg["TargetGroupArn"].(string))
	s.createTrustStoreLocked(elasticLoadBalancingDefaultTrustStoreName)
	return s
}

func (s *elasticLoadBalancingStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateLoadBalancer":
		name := elasticLoadBalancingFormString(form, "Name", elasticLoadBalancingDefaultLoadBalancerName)
		lb := s.createLoadBalancerLocked(name)
		return map[string]any{"LoadBalancers": []any{elasticLoadBalancingCloneMap(lb)}}
	case "DescribeLoadBalancers":
		return map[string]any{"LoadBalancers": elasticLoadBalancingSortedMapValues(s.loadBalancers), "NextMarker": ""}
	case "DeleteLoadBalancer":
		arn := elasticLoadBalancingFormString(form, "LoadBalancerArn", s.firstResourceARNLocked(s.loadBalancers))
		delete(s.loadBalancers, arn)
		delete(s.resourceTags, arn)
		return map[string]any{}
	case "DescribeLoadBalancerAttributes", "ModifyLoadBalancerAttributes":
		return map[string]any{
			"Attributes": []any{
				map[string]any{"Key": "deletion_protection.enabled", "Value": "false"},
				map[string]any{"Key": "routing.http2.enabled", "Value": "true"},
			},
		}

	case "CreateTargetGroup":
		name := elasticLoadBalancingFormString(form, "Name", elasticLoadBalancingDefaultTargetGroupName)
		tg := s.createTargetGroupLocked(name)
		return map[string]any{"TargetGroups": []any{elasticLoadBalancingCloneMap(tg)}}
	case "DescribeTargetGroups":
		return map[string]any{"TargetGroups": elasticLoadBalancingSortedMapValues(s.targetGroups), "NextMarker": ""}
	case "ModifyTargetGroup":
		arn := elasticLoadBalancingFormString(form, "TargetGroupArn", s.firstResourceARNLocked(s.targetGroups))
		targetGroup := elasticLoadBalancingCloneMap(s.targetGroups[arn])
		if targetGroup == nil {
			targetGroup = s.createTargetGroupLocked(elasticLoadBalancingDefaultTargetGroupName)
		}
		return map[string]any{"TargetGroups": []any{elasticLoadBalancingCloneMap(targetGroup)}}
	case "DeleteTargetGroup":
		arn := elasticLoadBalancingFormString(form, "TargetGroupArn", s.firstResourceARNLocked(s.targetGroups))
		delete(s.targetGroups, arn)
		delete(s.resourceTags, arn)
		return map[string]any{}
	case "DescribeTargetGroupAttributes", "ModifyTargetGroupAttributes":
		return map[string]any{
			"Attributes": []any{
				map[string]any{"Key": "deregistration_delay.timeout_seconds", "Value": "300"},
				map[string]any{"Key": "stickiness.enabled", "Value": "false"},
			},
		}
	case "DescribeTargetHealth":
		return map[string]any{
			"TargetHealthDescriptions": []any{
				map[string]any{
					"Target":          map[string]any{"Id": "i-0123456789abcdef0", "Port": 80},
					"HealthCheckPort": "80",
					"TargetHealth":    map[string]any{"State": "healthy", "Reason": "", "Description": "stackyard"},
				},
			},
		}
	case "RegisterTargets", "DeregisterTargets":
		return map[string]any{}

	case "CreateListener":
		lbArn := elasticLoadBalancingFormString(form, "LoadBalancerArn", s.firstResourceARNLocked(s.loadBalancers))
		targetGroupArn := elasticLoadBalancingFormString(form, "DefaultActions.member.1.TargetGroupArn", s.firstResourceARNLocked(s.targetGroups))
		if targetGroupArn == "" {
			tg := s.createTargetGroupLocked(elasticLoadBalancingDefaultTargetGroupName)
			targetGroupArn = tg["TargetGroupArn"].(string)
		}
		listener := s.createListenerLocked(elasticLoadBalancingDefaultLoadBalancerName, lbArn, targetGroupArn)
		return map[string]any{"Listeners": []any{elasticLoadBalancingCloneMap(listener)}}
	case "DescribeListeners":
		return map[string]any{"Listeners": elasticLoadBalancingSortedMapValues(s.listeners), "NextMarker": ""}
	case "ModifyListener":
		arn := elasticLoadBalancingFormString(form, "ListenerArn", s.firstResourceARNLocked(s.listeners))
		listener := elasticLoadBalancingCloneMap(s.listeners[arn])
		if listener == nil {
			listener = s.createListenerLocked(elasticLoadBalancingDefaultLoadBalancerName, s.firstResourceARNLocked(s.loadBalancers), s.firstResourceARNLocked(s.targetGroups))
		}
		return map[string]any{"Listeners": []any{listener}}
	case "DeleteListener":
		arn := elasticLoadBalancingFormString(form, "ListenerArn", s.firstResourceARNLocked(s.listeners))
		delete(s.listeners, arn)
		return map[string]any{}
	case "DescribeListenerCertificates":
		return map[string]any{
			"Certificates": []any{
				map[string]any{
					"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stackyard",
					"IsDefault":      true,
				},
			},
			"NextMarker": "",
		}
	case "DescribeListenerAttributes", "ModifyListenerAttributes":
		return map[string]any{
			"Attributes": []any{
				map[string]any{"Key": "routing.http.response.server.enabled", "Value": "true"},
				map[string]any{"Key": "routing.http.request.x_amzn_tls_version.enabled", "Value": "false"},
			},
		}
	case "AddListenerCertificates", "RemoveListenerCertificates":
		return map[string]any{}

	case "CreateRule":
		listenerArn := elasticLoadBalancingFormString(form, "ListenerArn", s.firstResourceARNLocked(s.listeners))
		targetGroupArn := s.firstResourceARNLocked(s.targetGroups)
		rule := s.createRuleLocked(elasticLoadBalancingDefaultLoadBalancerName, listenerArn, targetGroupArn)
		return map[string]any{"Rules": []any{elasticLoadBalancingCloneMap(rule)}}
	case "DescribeRules":
		return map[string]any{"Rules": elasticLoadBalancingSortedMapValues(s.rules), "NextMarker": ""}
	case "ModifyRule":
		arn := elasticLoadBalancingFormString(form, "RuleArn", s.firstResourceARNLocked(s.rules))
		rule := elasticLoadBalancingCloneMap(s.rules[arn])
		if rule == nil {
			rule = s.createRuleLocked(elasticLoadBalancingDefaultLoadBalancerName, s.firstResourceARNLocked(s.listeners), s.firstResourceARNLocked(s.targetGroups))
		}
		return map[string]any{"Rules": []any{rule}}
	case "SetRulePriorities":
		return map[string]any{"Rules": elasticLoadBalancingSortedMapValues(s.rules)}
	case "DeleteRule":
		arn := elasticLoadBalancingFormString(form, "RuleArn", s.firstResourceARNLocked(s.rules))
		delete(s.rules, arn)
		return map[string]any{}

	case "CreateTrustStore":
		name := elasticLoadBalancingFormString(form, "Name", elasticLoadBalancingDefaultTrustStoreName)
		trustStore := s.createTrustStoreLocked(name)
		return map[string]any{"TrustStores": []any{elasticLoadBalancingCloneMap(trustStore)}}
	case "DescribeTrustStores":
		return map[string]any{"TrustStores": elasticLoadBalancingSortedMapValues(s.trustStores), "NextMarker": ""}
	case "ModifyTrustStore":
		arn := elasticLoadBalancingFormString(form, "TrustStoreArn", s.firstResourceARNLocked(s.trustStores))
		trustStore := elasticLoadBalancingCloneMap(s.trustStores[arn])
		if trustStore == nil {
			trustStore = s.createTrustStoreLocked(elasticLoadBalancingDefaultTrustStoreName)
		}
		return map[string]any{"TrustStores": []any{trustStore}}
	case "DeleteTrustStore":
		arn := elasticLoadBalancingFormString(form, "TrustStoreArn", s.firstResourceARNLocked(s.trustStores))
		delete(s.trustStores, arn)
		return map[string]any{}
	case "DescribeTrustStoreAssociations":
		return map[string]any{
			"TrustStoreAssociations": []any{
				map[string]any{
					"ResourceArn":   s.firstResourceARNLocked(s.listeners),
					"TrustStoreArn": s.firstResourceARNLocked(s.trustStores),
				},
			},
			"NextMarker": "",
		}
	case "AddTrustStoreRevocations", "RemoveTrustStoreRevocations", "DeleteSharedTrustStoreAssociation":
		return map[string]any{}
	case "DescribeTrustStoreRevocations":
		return map[string]any{
			"TrustStoreRevocations": []any{
				map[string]any{
					"RevocationId":           1,
					"TrustStoreArn":          s.firstResourceARNLocked(s.trustStores),
					"NumberOfRevokedEntries": 0,
				},
			},
			"NextMarker": "",
		}
	case "GetTrustStoreCaCertificatesBundle":
		return map[string]any{"CaCertificatesBundle": "-----BEGIN CERTIFICATE-----\nSTACKYARD\n-----END CERTIFICATE-----"}
	case "GetTrustStoreRevocationContent":
		return map[string]any{"RevocationContent": "{}"}

	case "AddTags":
		resources := elasticLoadBalancingFormMembers(form, "ResourceArns.member.")
		if len(resources) == 0 {
			resources = []string{s.firstResourceARNLocked(s.loadBalancers)}
		}
		tags := elasticLoadBalancingFormTags(form)
		if len(tags) == 0 {
			tags = map[string]string{"managed-by": "stackyard"}
		}
		for _, resourceARN := range resources {
			if resourceARN == "" {
				continue
			}
			existing := s.resourceTags[resourceARN]
			if existing == nil {
				existing = map[string]string{}
				s.resourceTags[resourceARN] = existing
			}
			for k, v := range tags {
				existing[k] = v
			}
		}
		return map[string]any{}
	case "RemoveTags":
		resources := elasticLoadBalancingFormMembers(form, "ResourceArns.member.")
		keys := elasticLoadBalancingFormMembers(form, "TagKeys.member.")
		if len(resources) == 0 {
			resources = []string{s.firstResourceARNLocked(s.loadBalancers)}
		}
		for _, resourceARN := range resources {
			existing := s.resourceTags[resourceARN]
			for _, tagKey := range keys {
				delete(existing, tagKey)
			}
		}
		return map[string]any{}
	case "DescribeTags":
		resources := elasticLoadBalancingFormMembers(form, "ResourceArns.member.")
		if len(resources) == 0 {
			resources = elasticLoadBalancingSortedKeys(s.resourceTags)
		}
		descriptions := make([]any, 0, len(resources))
		for _, resourceARN := range resources {
			tagMap := s.resourceTags[resourceARN]
			tagList := make([]any, 0, len(tagMap))
			for _, key := range elasticLoadBalancingSortedKeys(tagMap) {
				tagList = append(tagList, map[string]any{"Key": key, "Value": tagMap[key]})
			}
			descriptions = append(descriptions, map[string]any{
				"ResourceArn": resourceARN,
				"Tags":        tagList,
			})
		}
		return map[string]any{"TagDescriptions": descriptions}

	case "DescribeSSLPolicies":
		return map[string]any{
			"SslPolicies": []any{
				map[string]any{
					"Name":         "ELBSecurityPolicy-TLS13-1-2-2021-06",
					"SslProtocols": []any{"TLSv1.2", "TLSv1.3"},
					"Ciphers":      []any{map[string]any{"Name": "TLS_AES_128_GCM_SHA256"}},
				},
			},
			"NextMarker": "",
		}
	case "DescribeAccountLimits":
		return map[string]any{
			"Limits": []any{
				map[string]any{"Name": "application-load-balancers", "Max": "50"},
				map[string]any{"Name": "target-groups", "Max": "3000"},
			},
		}
	case "DescribeCapacityReservation":
		return map[string]any{
			"CapacityReservationState": map[string]any{
				"State":             "provisioned",
				"EffectiveCapacity": map[string]any{"CapacityUnits": 1},
			},
		}
	case "ModifyCapacityReservation":
		return map[string]any{
			"CapacityReservationState": map[string]any{
				"State":             "provisioned",
				"EffectiveCapacity": map[string]any{"CapacityUnits": 1},
			},
		}
	case "ModifyIpPools":
		return map[string]any{}
	case "SetIpAddressType":
		return map[string]any{"IpAddressType": elasticLoadBalancingFormString(form, "IpAddressType", "ipv4")}
	case "SetSecurityGroups":
		return map[string]any{"SecurityGroupIds": []any{"sg-0123456789abcdef0"}}
	case "SetSubnets":
		return map[string]any{
			"AvailabilityZones": []any{
				map[string]any{"ZoneName": "us-east-1a", "SubnetId": "subnet-0123456789abcdef0"},
			},
			"IpAddressType": "ipv4",
		}
	case "GetResourcePolicy":
		arn := elasticLoadBalancingFormString(form, "ResourceArn", s.firstResourceARNLocked(s.loadBalancers))
		policy := s.resourcePolicy[arn]
		if policy == "" {
			policy = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
		}
		return map[string]any{"Policy": policy}
	}

	switch {
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{}
	case strings.HasPrefix(action, "Create"):
		return map[string]any{"ResourceArn": fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:stackyard/%s", strings.ToLower(action))}
	case strings.HasPrefix(action, "Get"):
		return map[string]any{}
	case strings.HasPrefix(action, "Delete"),
		strings.HasPrefix(action, "Modify"),
		strings.HasPrefix(action, "Register"),
		strings.HasPrefix(action, "Deregister"),
		strings.HasPrefix(action, "Add"),
		strings.HasPrefix(action, "Remove"),
		strings.HasPrefix(action, "Set"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *elasticLoadBalancingStore) createLoadBalancerLocked(name string) map[string]any {
	suffix := s.nextTokenLocked(16)
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/%s/%s", name, suffix)
	lb := map[string]any{
		"LoadBalancerArn":       arn,
		"DNSName":               fmt.Sprintf("%s-%s.elb.amazonaws.com", name, suffix),
		"CanonicalHostedZoneId": "Z35SXDOTRQ7X7K",
		"CreatedTime":           time.Now().UTC(),
		"LoadBalancerName":      name,
		"Scheme":                "internet-facing",
		"VpcId":                 "vpc-0123456789abcdef0",
		"State":                 map[string]any{"Code": "active"},
		"Type":                  "application",
		"IpAddressType":         "ipv4",
		"AvailabilityZones": []any{
			map[string]any{"ZoneName": "us-east-1a", "SubnetId": "subnet-0123456789abcdef0"},
			map[string]any{"ZoneName": "us-east-1b", "SubnetId": "subnet-0fedcba9876543210"},
		},
		"SecurityGroups": []any{"sg-0123456789abcdef0"},
	}
	s.loadBalancers[arn] = lb
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{"Name": name}
	}
	return elasticLoadBalancingCloneMap(lb)
}

func (s *elasticLoadBalancingStore) createTargetGroupLocked(name string) map[string]any {
	suffix := s.nextTokenLocked(16)
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/%s/%s", name, suffix)
	targetGroup := map[string]any{
		"TargetGroupArn":             arn,
		"TargetGroupName":            name,
		"Protocol":                   "HTTP",
		"Port":                       80,
		"VpcId":                      "vpc-0123456789abcdef0",
		"HealthCheckProtocol":        "HTTP",
		"HealthCheckPort":            "traffic-port",
		"HealthCheckEnabled":         true,
		"HealthCheckIntervalSeconds": 30,
		"HealthCheckTimeoutSeconds":  5,
		"HealthyThresholdCount":      5,
		"UnhealthyThresholdCount":    2,
		"TargetType":                 "instance",
	}
	s.targetGroups[arn] = targetGroup
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{"Name": name}
	}
	return elasticLoadBalancingCloneMap(targetGroup)
}

func (s *elasticLoadBalancingStore) createListenerLocked(name, loadBalancerARN, targetGroupARN string) map[string]any {
	if loadBalancerARN == "" {
		loadBalancerARN = s.firstResourceARNLocked(s.loadBalancers)
	}
	if targetGroupARN == "" {
		targetGroupARN = s.firstResourceARNLocked(s.targetGroups)
	}
	suffix := s.nextTokenLocked(16)
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/%s/%s/%s", name, s.nextTokenLocked(12), suffix)
	listener := map[string]any{
		"ListenerArn":     arn,
		"LoadBalancerArn": loadBalancerARN,
		"Port":            443,
		"Protocol":        "HTTPS",
		"SslPolicy":       "ELBSecurityPolicy-TLS13-1-2-2021-06",
		"Certificates": []any{
			map[string]any{"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stackyard"},
		},
		"DefaultActions": []any{
			map[string]any{"Type": "forward", "TargetGroupArn": targetGroupARN},
		},
	}
	s.listeners[arn] = listener
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{"Name": name}
	}
	return elasticLoadBalancingCloneMap(listener)
}

func (s *elasticLoadBalancingStore) createRuleLocked(name, listenerARN, targetGroupARN string) map[string]any {
	if listenerARN == "" {
		listenerARN = s.firstResourceARNLocked(s.listeners)
	}
	if targetGroupARN == "" {
		targetGroupARN = s.firstResourceARNLocked(s.targetGroups)
	}
	suffix := s.nextTokenLocked(16)
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:listener-rule/app/%s/%s", name, suffix)
	rule := map[string]any{
		"RuleArn":     arn,
		"ListenerArn": listenerARN,
		"Priority":    "1",
		"Conditions":  []any{map[string]any{"Field": "path-pattern", "Values": []any{"/*"}}},
		"Actions":     []any{map[string]any{"Type": "forward", "TargetGroupArn": targetGroupARN}},
		"IsDefault":   false,
	}
	s.rules[arn] = rule
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{"Name": name}
	}
	return elasticLoadBalancingCloneMap(rule)
}

func (s *elasticLoadBalancingStore) createTrustStoreLocked(name string) map[string]any {
	suffix := s.nextTokenLocked(16)
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:truststore/%s/%s", name, suffix)
	trustStore := map[string]any{
		"TrustStoreArn":          arn,
		"Name":                   name,
		"Status":                 "ACTIVE",
		"NumberOfCaCertificates": 1,
		"TotalRevokedEntries":    0,
	}
	s.trustStores[arn] = trustStore
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{"Name": name}
	}
	return elasticLoadBalancingCloneMap(trustStore)
}

func (s *elasticLoadBalancingStore) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%0*x", width, id)
}

func (s *elasticLoadBalancingStore) firstResourceARNLocked(resources map[string]map[string]any) string {
	if len(resources) == 0 {
		return ""
	}
	keys := elasticLoadBalancingSortedKeys(resources)
	return keys[0]
}

func elasticLoadBalancingCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func elasticLoadBalancingSortedMapValues(resources map[string]map[string]any) []any {
	out := make([]any, 0, len(resources))
	for _, arn := range elasticLoadBalancingSortedKeys(resources) {
		out = append(out, elasticLoadBalancingCloneMap(resources[arn]))
	}
	return out
}

func elasticLoadBalancingSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func elasticLoadBalancingFormString(form url.Values, key, fallback string) string {
	value := strings.TrimSpace(form.Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func elasticLoadBalancingFormMembers(form url.Values, prefix string) []string {
	indexed := map[int]string{}
	for key, values := range form {
		if !strings.HasPrefix(key, prefix) || len(values) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		idx, err := strconv.Atoi(rest)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(values[0])
		if value == "" {
			continue
		}
		indexed[idx] = value
	}
	indices := make([]int, 0, len(indexed))
	for idx := range indexed {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]string, 0, len(indices))
	for _, idx := range indices {
		out = append(out, indexed[idx])
	}
	return out
}

func elasticLoadBalancingFormTags(form url.Values) map[string]string {
	keys := map[int]string{}
	values := map[int]string{}
	for formKey, formValues := range form {
		if len(formValues) == 0 {
			continue
		}
		const prefix = "Tags.member."
		if !strings.HasPrefix(formKey, prefix) {
			continue
		}
		rest := strings.TrimPrefix(formKey, prefix)
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		value := strings.TrimSpace(formValues[0])
		switch parts[1] {
		case "Key":
			keys[idx] = value
		case "Value":
			values[idx] = value
		}
	}
	out := map[string]string{}
	for idx, key := range keys {
		if key == "" {
			continue
		}
		out[key] = values[idx]
	}
	return out
}
