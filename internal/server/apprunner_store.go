package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	appRunnerDefaultRegion    = "us-east-1"
	appRunnerDefaultAccountID = "123456789012"
)

type appRunnerStore struct {
	mu sync.Mutex

	nextID int64

	defaultAutoScalingARN string

	services            map[string]map[string]any
	servicesByName      map[string]string
	connections         map[string]map[string]any
	autoScalings        map[string]map[string]any
	autoScalingsByName  map[string]string
	observabilities     map[string]map[string]any
	observabilityByName map[string]string
	vpcConnectors       map[string]map[string]any
	vpcConnectorByName  map[string]string
	vpcIngresses        map[string]map[string]any
	vpcIngressByName    map[string]string
	customDomains       map[string]map[string]map[string]any
	operations          map[string][]map[string]any
	tags                map[string]map[string]string
}

func newAppRunnerStore() *appRunnerStore {
	s := &appRunnerStore{
		nextID:              2,
		services:            map[string]map[string]any{},
		servicesByName:      map[string]string{},
		connections:         map[string]map[string]any{},
		autoScalings:        map[string]map[string]any{},
		autoScalingsByName:  map[string]string{},
		observabilities:     map[string]map[string]any{},
		observabilityByName: map[string]string{},
		vpcConnectors:       map[string]map[string]any{},
		vpcConnectorByName:  map[string]string{},
		vpcIngresses:        map[string]map[string]any{},
		vpcIngressByName:    map[string]string{},
		customDomains:       map[string]map[string]map[string]any{},
		operations:          map[string][]map[string]any{},
		tags:                map[string]map[string]string{},
	}
	now := time.Now().UTC().Format(time.RFC3339)
	auto := s.ensureAutoScalingByNameLocked("stackyard-auto-scaling", now)
	auto["IsDefault"] = true
	s.defaultAutoScalingARN = appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", "")
	_ = s.ensureServiceByNameLocked("stackyard-service", now)
	_ = s.ensureConnectionByNameLocked("stackyard-connection", now)
	_ = s.ensureObservabilityByNameLocked("stackyard-observability", now)
	_ = s.ensureVpcConnectorByNameLocked("stackyard-vpc-connector", now)
	_ = s.ensureVpcIngressByNameLocked("stackyard-vpc-ingress", "", now)
	return s
}

func (s *appRunnerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	serviceArn := appRunnerString(payload, "ServiceArn", "")
	if serviceArn == "" {
		serviceArn = appRunnerString(payload, "ResourceArn", "")
	}
	if serviceArn == "" {
		serviceArn = appRunnerFirstMapValue(s.servicesByName)
	}
	service := s.ensureServiceByARNLocked(serviceArn, now)
	serviceArn = appRunnerStringFromMap(service, "ServiceArn", serviceArn)

	switch action {
	case "CreateService":
		name := appRunnerString(payload, "ServiceName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-service-%06d", s.nextID)
			s.nextID++
		}
		service = s.ensureServiceByNameLocked(name, now)
		autoArn := appRunnerString(payload, "AutoScalingConfigurationArn", s.defaultAutoScalingARN)
		if autoArn != "" {
			auto := s.ensureAutoScalingByARNLocked(autoArn, now)
			autoArn = appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", autoArn)
			auto["HasAssociatedService"] = true
			service["AutoScalingConfigurationSummary"] = map[string]any{"AutoScalingConfigurationArn": autoArn}
		}
		service["Status"] = "RUNNING"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(appRunnerStringFromMap(service, "ServiceArn", ""), "CreateService", now)
		s.upsertTagsLocked(appRunnerStringFromMap(service, "ServiceArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"OperationId": opID, "Service": appRunnerCloneMap(service)}

	case "DeleteService":
		service["Status"] = "DELETED"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(serviceArn, "DeleteService", now)
		return map[string]any{"OperationId": opID, "Service": appRunnerCloneMap(service)}

	case "DescribeService":
		return map[string]any{"Service": appRunnerCloneMap(service)}

	case "ListServices":
		items := make([]any, 0, len(s.services))
		for _, svc := range appRunnerSortedMapValues(s.services, "ServiceName") {
			items = append(items, map[string]any{
				"ServiceName": appRunnerStringFromMap(svc, "ServiceName", ""),
				"ServiceId":   appRunnerStringFromMap(svc, "ServiceId", ""),
				"ServiceArn":  appRunnerStringFromMap(svc, "ServiceArn", ""),
				"ServiceUrl":  appRunnerStringFromMap(svc, "ServiceUrl", ""),
				"Status":      appRunnerStringFromMap(svc, "Status", "RUNNING"),
				"CreatedAt":   appRunnerStringFromMap(svc, "CreatedAt", now),
				"UpdatedAt":   appRunnerStringFromMap(svc, "UpdatedAt", now),
			})
		}
		return map[string]any{"ServiceSummaryList": items, "NextToken": ""}

	case "UpdateService":
		autoArn := appRunnerString(payload, "AutoScalingConfigurationArn", "")
		if autoArn != "" {
			auto := s.ensureAutoScalingByARNLocked(autoArn, now)
			auto["HasAssociatedService"] = true
			service["AutoScalingConfigurationSummary"] = map[string]any{"AutoScalingConfigurationArn": appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", autoArn)}
		}
		service["Status"] = "RUNNING"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(serviceArn, "UpdateService", now)
		return map[string]any{"OperationId": opID, "Service": appRunnerCloneMap(service)}

	case "PauseService":
		service["Status"] = "PAUSED"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(serviceArn, "PauseService", now)
		return map[string]any{"OperationId": opID, "Service": appRunnerCloneMap(service)}

	case "ResumeService":
		service["Status"] = "RUNNING"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(serviceArn, "ResumeService", now)
		return map[string]any{"OperationId": opID, "Service": appRunnerCloneMap(service)}

	case "StartDeployment":
		service["Status"] = "RUNNING"
		service["UpdatedAt"] = now
		opID := s.addOperationLocked(serviceArn, "StartDeployment", now)
		return map[string]any{"OperationId": opID}

	case "ListOperations":
		ops := s.operations[serviceArn]
		if len(ops) == 0 {
			s.addOperationLocked(serviceArn, "ListOperations", now)
			ops = s.operations[serviceArn]
		}
		items := make([]any, 0, len(ops))
		for _, op := range ops {
			items = append(items, appRunnerCloneMap(op))
		}
		return map[string]any{"OperationSummaryList": items, "NextToken": ""}

	case "CreateConnection":
		name := appRunnerString(payload, "ConnectionName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-connection-%06d", s.nextID)
			s.nextID++
		}
		conn := s.ensureConnectionByNameLocked(name, now)
		if provider := appRunnerString(payload, "ProviderType", "GITHUB"); provider != "" {
			conn["ProviderType"] = provider
		}
		s.upsertTagsLocked(appRunnerStringFromMap(conn, "ConnectionArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"Connection": appRunnerCloneMap(conn)}

	case "DeleteConnection":
		arn := appRunnerString(payload, "ConnectionArn", "")
		conn := s.ensureConnectionByARNLocked(arn, now)
		conn["Status"] = "DELETED"
		conn["UpdatedAt"] = now
		return map[string]any{"Connection": appRunnerCloneMap(conn)}

	case "ListConnections":
		items := make([]any, 0, len(s.connections))
		for _, conn := range appRunnerSortedMapValues(s.connections, "ConnectionName") {
			items = append(items, map[string]any{
				"ConnectionName": appRunnerStringFromMap(conn, "ConnectionName", ""),
				"ConnectionArn":  appRunnerStringFromMap(conn, "ConnectionArn", ""),
				"ProviderType":   appRunnerStringFromMap(conn, "ProviderType", "GITHUB"),
				"Status":         appRunnerStringFromMap(conn, "Status", "AVAILABLE"),
				"CreatedAt":      appRunnerStringFromMap(conn, "CreatedAt", now),
			})
		}
		return map[string]any{"ConnectionSummaryList": items, "NextToken": ""}

	case "CreateAutoScalingConfiguration":
		name := appRunnerString(payload, "AutoScalingConfigurationName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-auto-scaling-%06d", s.nextID)
			s.nextID++
		}
		auto := s.ensureAutoScalingByNameLocked(name, now)
		auto["UpdatedAt"] = now
		auto["HasAssociatedService"] = false
		s.upsertTagsLocked(appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"AutoScalingConfiguration": appRunnerCloneMap(auto)}

	case "DescribeAutoScalingConfiguration":
		arn := appRunnerString(payload, "AutoScalingConfigurationArn", s.defaultAutoScalingARN)
		auto := s.ensureAutoScalingByARNLocked(arn, now)
		return map[string]any{"AutoScalingConfiguration": appRunnerCloneMap(auto)}

	case "ListAutoScalingConfigurations":
		items := make([]any, 0, len(s.autoScalings))
		for _, auto := range appRunnerSortedMapValues(s.autoScalings, "AutoScalingConfigurationName") {
			items = append(items, map[string]any{
				"AutoScalingConfigurationName":     appRunnerStringFromMap(auto, "AutoScalingConfigurationName", ""),
				"AutoScalingConfigurationArn":      appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", ""),
				"AutoScalingConfigurationRevision": appRunnerIntFromMap(auto, "AutoScalingConfigurationRevision", 1),
				"Status":                           appRunnerStringFromMap(auto, "Status", "ACTIVE"),
				"HasAssociatedService":             appRunnerBoolFromMap(auto, "HasAssociatedService", false),
				"IsDefault":                        appRunnerBoolFromMap(auto, "IsDefault", false),
			})
		}
		return map[string]any{"AutoScalingConfigurationSummaryList": items, "NextToken": ""}

	case "DeleteAutoScalingConfiguration":
		arn := appRunnerString(payload, "AutoScalingConfigurationArn", s.defaultAutoScalingARN)
		auto := s.ensureAutoScalingByARNLocked(arn, now)
		auto["Status"] = "INACTIVE"
		auto["UpdatedAt"] = now
		return map[string]any{"AutoScalingConfiguration": appRunnerCloneMap(auto)}

	case "UpdateDefaultAutoScalingConfiguration":
		arn := appRunnerString(payload, "AutoScalingConfigurationArn", s.defaultAutoScalingARN)
		auto := s.ensureAutoScalingByARNLocked(arn, now)
		for _, candidate := range s.autoScalings {
			candidate["IsDefault"] = false
		}
		auto["IsDefault"] = true
		auto["Status"] = "ACTIVE"
		s.defaultAutoScalingARN = appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", arn)
		return map[string]any{"AutoScalingConfiguration": appRunnerCloneMap(auto)}

	case "ListServicesForAutoScalingConfiguration":
		autoArn := appRunnerString(payload, "AutoScalingConfigurationArn", s.defaultAutoScalingARN)
		arns := make([]any, 0)
		for _, svc := range s.services {
			summary := appRunnerMapFromAny(svc["AutoScalingConfigurationSummary"])
			if appRunnerStringFromMap(summary, "AutoScalingConfigurationArn", "") == autoArn {
				arns = append(arns, appRunnerStringFromMap(svc, "ServiceArn", ""))
			}
		}
		sort.SliceStable(arns, func(i, j int) bool {
			return fmt.Sprint(arns[i]) < fmt.Sprint(arns[j])
		})
		return map[string]any{"ServiceArnList": arns, "NextToken": ""}

	case "CreateObservabilityConfiguration":
		name := appRunnerString(payload, "ObservabilityConfigurationName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-observability-%06d", s.nextID)
			s.nextID++
		}
		obs := s.ensureObservabilityByNameLocked(name, now)
		if trace := appRunnerMapFromAny(payload["TraceConfiguration"]); len(trace) > 0 {
			obs["TraceConfiguration"] = appRunnerCloneMap(trace)
		}
		s.upsertTagsLocked(appRunnerStringFromMap(obs, "ObservabilityConfigurationArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"ObservabilityConfiguration": appRunnerCloneMap(obs)}

	case "DescribeObservabilityConfiguration":
		arn := appRunnerString(payload, "ObservabilityConfigurationArn", appRunnerFirstMapValue(s.observabilityByName))
		obs := s.ensureObservabilityByARNLocked(arn, now)
		return map[string]any{"ObservabilityConfiguration": appRunnerCloneMap(obs)}

	case "ListObservabilityConfigurations":
		items := make([]any, 0, len(s.observabilities))
		for _, obs := range appRunnerSortedMapValues(s.observabilities, "ObservabilityConfigurationName") {
			items = append(items, map[string]any{
				"ObservabilityConfigurationName":     appRunnerStringFromMap(obs, "ObservabilityConfigurationName", ""),
				"ObservabilityConfigurationArn":      appRunnerStringFromMap(obs, "ObservabilityConfigurationArn", ""),
				"ObservabilityConfigurationRevision": appRunnerIntFromMap(obs, "ObservabilityConfigurationRevision", 1),
				"Status":                             appRunnerStringFromMap(obs, "Status", "ACTIVE"),
			})
		}
		return map[string]any{"ObservabilityConfigurationSummaryList": items, "NextToken": ""}

	case "DeleteObservabilityConfiguration":
		arn := appRunnerString(payload, "ObservabilityConfigurationArn", appRunnerFirstMapValue(s.observabilityByName))
		obs := s.ensureObservabilityByARNLocked(arn, now)
		obs["Status"] = "INACTIVE"
		obs["UpdatedAt"] = now
		return map[string]any{"ObservabilityConfiguration": appRunnerCloneMap(obs)}

	case "CreateVpcConnector":
		name := appRunnerString(payload, "VpcConnectorName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-vpc-connector-%06d", s.nextID)
			s.nextID++
		}
		connector := s.ensureVpcConnectorByNameLocked(name, now)
		if subnets := appRunnerStringSlice(payload["Subnets"]); len(subnets) > 0 {
			connector["Subnets"] = appRunnerStringSliceToAny(subnets)
		}
		if groups := appRunnerStringSlice(payload["SecurityGroups"]); len(groups) > 0 {
			connector["SecurityGroups"] = appRunnerStringSliceToAny(groups)
		}
		s.upsertTagsLocked(appRunnerStringFromMap(connector, "VpcConnectorArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"VpcConnector": appRunnerCloneMap(connector)}

	case "DescribeVpcConnector":
		arn := appRunnerString(payload, "VpcConnectorArn", appRunnerFirstMapValue(s.vpcConnectorByName))
		connector := s.ensureVpcConnectorByARNLocked(arn, now)
		return map[string]any{"VpcConnector": appRunnerCloneMap(connector)}

	case "ListVpcConnectors":
		items := make([]any, 0, len(s.vpcConnectors))
		for _, connector := range appRunnerSortedMapValues(s.vpcConnectors, "VpcConnectorName") {
			items = append(items, appRunnerCloneMap(connector))
		}
		return map[string]any{"VpcConnectors": items, "NextToken": ""}

	case "DeleteVpcConnector":
		arn := appRunnerString(payload, "VpcConnectorArn", appRunnerFirstMapValue(s.vpcConnectorByName))
		connector := s.ensureVpcConnectorByARNLocked(arn, now)
		connector["Status"] = "INACTIVE"
		connector["UpdatedAt"] = now
		return map[string]any{"VpcConnector": appRunnerCloneMap(connector)}

	case "CreateVpcIngressConnection":
		name := appRunnerString(payload, "VpcIngressConnectionName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-vpc-ingress-%06d", s.nextID)
			s.nextID++
		}
		serviceArn := appRunnerString(payload, "ServiceArn", appRunnerStringFromMap(service, "ServiceArn", ""))
		ingress := s.ensureVpcIngressByNameLocked(name, serviceArn, now)
		if cfg := appRunnerMapFromAny(payload["IngressVpcConfiguration"]); len(cfg) > 0 {
			ingress["IngressVpcConfiguration"] = appRunnerCloneMap(cfg)
		}
		s.upsertTagsLocked(appRunnerStringFromMap(ingress, "VpcIngressConnectionArn", ""), appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{"VpcIngressConnection": appRunnerCloneMap(ingress)}

	case "DescribeVpcIngressConnection":
		arn := appRunnerString(payload, "VpcIngressConnectionArn", appRunnerFirstMapValue(s.vpcIngressByName))
		ingress := s.ensureVpcIngressByARNLocked(arn, now)
		return map[string]any{"VpcIngressConnection": appRunnerCloneMap(ingress)}

	case "ListVpcIngressConnections":
		items := make([]any, 0, len(s.vpcIngresses))
		for _, ingress := range appRunnerSortedMapValues(s.vpcIngresses, "VpcIngressConnectionName") {
			items = append(items, map[string]any{
				"VpcIngressConnectionName": appRunnerStringFromMap(ingress, "VpcIngressConnectionName", ""),
				"VpcIngressConnectionArn":  appRunnerStringFromMap(ingress, "VpcIngressConnectionArn", ""),
				"ServiceArn":               appRunnerStringFromMap(ingress, "ServiceArn", ""),
				"Status":                   appRunnerStringFromMap(ingress, "Status", "AVAILABLE"),
				"DomainName":               appRunnerStringFromMap(ingress, "DomainName", ""),
			})
		}
		return map[string]any{"VpcIngressConnectionSummaryList": items, "NextToken": ""}

	case "UpdateVpcIngressConnection":
		arn := appRunnerString(payload, "VpcIngressConnectionArn", appRunnerFirstMapValue(s.vpcIngressByName))
		ingress := s.ensureVpcIngressByARNLocked(arn, now)
		if cfg := appRunnerMapFromAny(payload["IngressVpcConfiguration"]); len(cfg) > 0 {
			ingress["IngressVpcConfiguration"] = appRunnerCloneMap(cfg)
		}
		ingress["UpdatedAt"] = now
		return map[string]any{"VpcIngressConnection": appRunnerCloneMap(ingress)}

	case "DeleteVpcIngressConnection":
		arn := appRunnerString(payload, "VpcIngressConnectionArn", appRunnerFirstMapValue(s.vpcIngressByName))
		ingress := s.ensureVpcIngressByARNLocked(arn, now)
		ingress["Status"] = "DELETED"
		ingress["UpdatedAt"] = now
		return map[string]any{"VpcIngressConnection": appRunnerCloneMap(ingress)}

	case "AssociateCustomDomain":
		serviceArn := appRunnerString(payload, "ServiceArn", appRunnerStringFromMap(service, "ServiceArn", ""))
		domainName := appRunnerString(payload, "DomainName", "example.com")
		domain := s.ensureCustomDomainLocked(serviceArn, domainName, now)
		domain["EnableWWWSubdomain"] = appRunnerBool(payload, "EnableWWWSubdomain", false)
		domain["Status"] = "ACTIVE"
		return map[string]any{
			"ServiceArn":    serviceArn,
			"DNSTarget":     appRunnerServiceDNSTarget(serviceArn),
			"CustomDomain":  appRunnerCloneMap(domain),
			"VpcDNSTargets": []any{map[string]any{"VpcId": "vpc-12345", "DomainName": appRunnerStringFromMap(domain, "DomainName", domainName), "HostedZoneId": "Z12345"}},
		}

	case "DisassociateCustomDomain":
		serviceArn := appRunnerString(payload, "ServiceArn", appRunnerStringFromMap(service, "ServiceArn", ""))
		domainName := appRunnerString(payload, "DomainName", "example.com")
		domain := s.ensureCustomDomainLocked(serviceArn, domainName, now)
		domain["Status"] = "DELETED"
		delete(s.customDomains[serviceArn], domainName)
		return map[string]any{
			"ServiceArn":    serviceArn,
			"DNSTarget":     appRunnerServiceDNSTarget(serviceArn),
			"CustomDomain":  appRunnerCloneMap(domain),
			"VpcDNSTargets": []any{},
		}

	case "DescribeCustomDomains":
		serviceArn := appRunnerString(payload, "ServiceArn", appRunnerStringFromMap(service, "ServiceArn", ""))
		domains := []any{}
		serviceDomains := s.customDomains[serviceArn]
		names := make([]string, 0, len(serviceDomains))
		for name := range serviceDomains {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			domains = append(domains, appRunnerCloneMap(serviceDomains[name]))
		}
		if len(domains) == 0 {
			domains = append(domains, appRunnerCloneMap(s.ensureCustomDomainLocked(serviceArn, "example.com", now)))
		}
		return map[string]any{
			"ServiceArn":    serviceArn,
			"DNSTarget":     appRunnerServiceDNSTarget(serviceArn),
			"CustomDomains": domains,
			"VpcDNSTargets": []any{map[string]any{"VpcId": "vpc-12345", "DomainName": "example.com", "HostedZoneId": "Z12345"}},
			"NextToken":     "",
		}

	case "TagResource":
		resourceArn := appRunnerString(payload, "ResourceArn", serviceArn)
		s.upsertTagsLocked(resourceArn, appRunnerPayloadTags(payload, "Tags"))
		return map[string]any{}

	case "UntagResource":
		resourceArn := appRunnerString(payload, "ResourceArn", serviceArn)
		tagMap := s.ensureTagsLocked(resourceArn)
		for _, key := range appRunnerTagKeys(payload["TagKeys"]) {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceArn := appRunnerString(payload, "ResourceArn", serviceArn)
		tagMap := s.ensureTagsLocked(resourceArn)
		return map[string]any{"Tags": appRunnerTagList(tagMap)}
	}

	return map[string]any{}
}

func (s *appRunnerStore) ensureServiceByNameLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-service"
	}
	if arn, ok := s.servicesByName[name]; ok {
		if svc, ok := s.services[arn]; ok {
			return svc
		}
	}
	serviceID := fmt.Sprintf("%016d", s.nextID)
	s.nextID++
	arn := appRunnerServiceARN(name, serviceID)
	svc := map[string]any{
		"ServiceName": name,
		"ServiceId":   serviceID,
		"ServiceArn":  arn,
		"ServiceUrl":  fmt.Sprintf("https://%s.%s.awsapprunner.com", name, serviceID),
		"Status":      "RUNNING",
		"CreatedAt":   now,
		"UpdatedAt":   now,
		"AutoScalingConfigurationSummary": map[string]any{
			"AutoScalingConfigurationArn": s.defaultAutoScalingARN,
		},
	}
	s.services[arn] = svc
	s.servicesByName[name] = arn
	s.ensureTagsLocked(arn)
	if s.defaultAutoScalingARN != "" {
		auto := s.ensureAutoScalingByARNLocked(s.defaultAutoScalingARN, now)
		auto["HasAssociatedService"] = true
	}
	return svc
}

func (s *appRunnerStore) ensureServiceByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureServiceByNameLocked("stackyard-service", now)
	}
	if svc, ok := s.services[arn]; ok {
		return svc
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-service"
	}
	svc := s.ensureServiceByNameLocked(name, now)
	if appRunnerStringFromMap(svc, "ServiceArn", "") != arn {
		svc = appRunnerCloneMap(svc)
		svc["ServiceArn"] = arn
		s.services[arn] = svc
		s.servicesByName[name] = arn
		s.ensureTagsLocked(arn)
	}
	return s.services[arn]
}

func (s *appRunnerStore) ensureConnectionByNameLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-connection"
	}
	for _, conn := range s.connections {
		if appRunnerStringFromMap(conn, "ConnectionName", "") == name {
			return conn
		}
	}
	id := fmt.Sprintf("%016d", s.nextID)
	s.nextID++
	arn := appRunnerConnectionARN(name, id)
	conn := map[string]any{
		"ConnectionName": name,
		"ConnectionArn":  arn,
		"ProviderType":   "GITHUB",
		"Status":         "AVAILABLE",
		"CreatedAt":      now,
		"UpdatedAt":      now,
	}
	s.connections[arn] = conn
	s.ensureTagsLocked(arn)
	return conn
}

func (s *appRunnerStore) ensureConnectionByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureConnectionByNameLocked("stackyard-connection", now)
	}
	if conn, ok := s.connections[arn]; ok {
		return conn
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-connection"
	}
	conn := s.ensureConnectionByNameLocked(name, now)
	if appRunnerStringFromMap(conn, "ConnectionArn", "") != arn {
		conn = appRunnerCloneMap(conn)
		conn["ConnectionArn"] = arn
		s.connections[arn] = conn
		s.ensureTagsLocked(arn)
	}
	return s.connections[arn]
}

func (s *appRunnerStore) ensureAutoScalingByNameLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-auto-scaling"
	}
	if arn, ok := s.autoScalingsByName[name]; ok {
		if auto, ok := s.autoScalings[arn]; ok {
			return auto
		}
	}
	revision := 1
	arn := appRunnerAutoScalingARN(name, revision)
	auto := map[string]any{
		"AutoScalingConfigurationName":     name,
		"AutoScalingConfigurationArn":      arn,
		"AutoScalingConfigurationRevision": revision,
		"Status":                           "ACTIVE",
		"CreatedAt":                        now,
		"UpdatedAt":                        now,
		"HasAssociatedService":             false,
		"IsDefault":                        false,
	}
	s.autoScalings[arn] = auto
	s.autoScalingsByName[name] = arn
	s.ensureTagsLocked(arn)
	return auto
}

func (s *appRunnerStore) ensureAutoScalingByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		if s.defaultAutoScalingARN != "" {
			if auto, ok := s.autoScalings[s.defaultAutoScalingARN]; ok {
				return auto
			}
		}
		return s.ensureAutoScalingByNameLocked("stackyard-auto-scaling", now)
	}
	if auto, ok := s.autoScalings[arn]; ok {
		return auto
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-auto-scaling"
	}
	auto := s.ensureAutoScalingByNameLocked(name, now)
	if appRunnerStringFromMap(auto, "AutoScalingConfigurationArn", "") != arn {
		auto = appRunnerCloneMap(auto)
		auto["AutoScalingConfigurationArn"] = arn
		s.autoScalings[arn] = auto
		s.autoScalingsByName[name] = arn
		s.ensureTagsLocked(arn)
	}
	return s.autoScalings[arn]
}

func (s *appRunnerStore) ensureObservabilityByNameLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-observability"
	}
	if arn, ok := s.observabilityByName[name]; ok {
		if obs, ok := s.observabilities[arn]; ok {
			return obs
		}
	}
	revision := 1
	arn := appRunnerObservabilityARN(name, revision)
	obs := map[string]any{
		"ObservabilityConfigurationName":     name,
		"ObservabilityConfigurationArn":      arn,
		"ObservabilityConfigurationRevision": revision,
		"TraceConfiguration":                 map[string]any{"Vendor": "AWSXRAY"},
		"Status":                             "ACTIVE",
		"CreatedAt":                          now,
		"UpdatedAt":                          now,
	}
	s.observabilities[arn] = obs
	s.observabilityByName[name] = arn
	s.ensureTagsLocked(arn)
	return obs
}

func (s *appRunnerStore) ensureObservabilityByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureObservabilityByNameLocked("stackyard-observability", now)
	}
	if obs, ok := s.observabilities[arn]; ok {
		return obs
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-observability"
	}
	obs := s.ensureObservabilityByNameLocked(name, now)
	if appRunnerStringFromMap(obs, "ObservabilityConfigurationArn", "") != arn {
		obs = appRunnerCloneMap(obs)
		obs["ObservabilityConfigurationArn"] = arn
		s.observabilities[arn] = obs
		s.observabilityByName[name] = arn
		s.ensureTagsLocked(arn)
	}
	return s.observabilities[arn]
}

func (s *appRunnerStore) ensureVpcConnectorByNameLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-vpc-connector"
	}
	if arn, ok := s.vpcConnectorByName[name]; ok {
		if connector, ok := s.vpcConnectors[arn]; ok {
			return connector
		}
	}
	id := fmt.Sprintf("%016d", s.nextID)
	s.nextID++
	arn := appRunnerVpcConnectorARN(name, id)
	connector := map[string]any{
		"VpcConnectorName": name,
		"VpcConnectorArn":  arn,
		"Status":           "ACTIVE",
		"Subnets":          []any{"subnet-12345", "subnet-67890"},
		"SecurityGroups":   []any{"sg-12345"},
		"CreatedAt":        now,
		"UpdatedAt":        now,
	}
	s.vpcConnectors[arn] = connector
	s.vpcConnectorByName[name] = arn
	s.ensureTagsLocked(arn)
	return connector
}

func (s *appRunnerStore) ensureVpcConnectorByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureVpcConnectorByNameLocked("stackyard-vpc-connector", now)
	}
	if connector, ok := s.vpcConnectors[arn]; ok {
		return connector
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-vpc-connector"
	}
	connector := s.ensureVpcConnectorByNameLocked(name, now)
	if appRunnerStringFromMap(connector, "VpcConnectorArn", "") != arn {
		connector = appRunnerCloneMap(connector)
		connector["VpcConnectorArn"] = arn
		s.vpcConnectors[arn] = connector
		s.vpcConnectorByName[name] = arn
		s.ensureTagsLocked(arn)
	}
	return s.vpcConnectors[arn]
}

func (s *appRunnerStore) ensureVpcIngressByNameLocked(name, serviceArn, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-vpc-ingress"
	}
	if arn, ok := s.vpcIngressByName[name]; ok {
		if ingress, ok := s.vpcIngresses[arn]; ok {
			if serviceArn != "" {
				ingress["ServiceArn"] = serviceArn
			}
			return ingress
		}
	}
	id := fmt.Sprintf("%016d", s.nextID)
	s.nextID++
	arn := appRunnerVpcIngressARN(name, id)
	if serviceArn == "" {
		service := s.ensureServiceByNameLocked("stackyard-service", now)
		serviceArn = appRunnerStringFromMap(service, "ServiceArn", "")
	}
	ingress := map[string]any{
		"VpcIngressConnectionName": name,
		"VpcIngressConnectionArn":  arn,
		"ServiceArn":               serviceArn,
		"Status":                   "AVAILABLE",
		"DomainName":               fmt.Sprintf("%s.example.com", name),
		"IngressVpcConfiguration": map[string]any{
			"VpcId":         "vpc-12345",
			"VpcEndpointId": "vpce-12345",
		},
		"CreatedAt": now,
		"UpdatedAt": now,
	}
	s.vpcIngresses[arn] = ingress
	s.vpcIngressByName[name] = arn
	s.ensureTagsLocked(arn)
	return ingress
}

func (s *appRunnerStore) ensureVpcIngressByARNLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return s.ensureVpcIngressByNameLocked("stackyard-vpc-ingress", "", now)
	}
	if ingress, ok := s.vpcIngresses[arn]; ok {
		return ingress
	}
	name := appRunnerResourceNameFromARN(arn)
	if name == "" {
		name = "stackyard-vpc-ingress"
	}
	ingress := s.ensureVpcIngressByNameLocked(name, "", now)
	if appRunnerStringFromMap(ingress, "VpcIngressConnectionArn", "") != arn {
		ingress = appRunnerCloneMap(ingress)
		ingress["VpcIngressConnectionArn"] = arn
		s.vpcIngresses[arn] = ingress
		s.vpcIngressByName[name] = arn
		s.ensureTagsLocked(arn)
	}
	return s.vpcIngresses[arn]
}

func (s *appRunnerStore) ensureCustomDomainLocked(serviceArn, domainName, now string) map[string]any {
	serviceArn = strings.TrimSpace(serviceArn)
	if serviceArn == "" {
		service := s.ensureServiceByNameLocked("stackyard-service", now)
		serviceArn = appRunnerStringFromMap(service, "ServiceArn", "")
	}
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		domainName = "example.com"
	}
	if _, ok := s.customDomains[serviceArn]; !ok {
		s.customDomains[serviceArn] = map[string]map[string]any{}
	}
	if domain, ok := s.customDomains[serviceArn][domainName]; ok {
		return domain
	}
	domain := map[string]any{
		"DomainName":                   domainName,
		"EnableWWWSubdomain":           false,
		"Status":                       "ACTIVE",
		"CertificateValidationRecords": []any{map[string]any{"Name": fmt.Sprintf("_acme-challenge.%s", domainName), "Type": "CNAME", "Value": "validation-token", "Status": "SUCCESS"}},
	}
	s.customDomains[serviceArn][domainName] = domain
	return domain
}

func (s *appRunnerStore) addOperationLocked(serviceArn, opType, now string) string {
	if strings.TrimSpace(serviceArn) == "" {
		service := s.ensureServiceByNameLocked("stackyard-service", now)
		serviceArn = appRunnerStringFromMap(service, "ServiceArn", "")
	}
	id := fmt.Sprintf("op-%06d", s.nextID)
	s.nextID++
	op := map[string]any{
		"Id":        id,
		"Type":      opType,
		"Status":    "SUCCEEDED",
		"TargetArn": serviceArn,
		"StartedAt": now,
		"EndedAt":   now,
		"UpdatedAt": now,
	}
	s.operations[serviceArn] = append(s.operations[serviceArn], op)
	return id
}

func (s *appRunnerStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = appRunnerServiceARN("stackyard-service", "0000000000000001")
	}
	tags, ok := s.tags[resourceArn]
	if !ok {
		tags = map[string]string{"stackyard": "true"}
		s.tags[resourceArn] = tags
	}
	return tags
}

func (s *appRunnerStore) upsertTagsLocked(resourceArn string, incoming map[string]string) {
	if len(incoming) == 0 {
		return
	}
	tags := s.ensureTagsLocked(resourceArn)
	for key, value := range incoming {
		tags[key] = value
	}
}

func appRunnerPayloadTags(payload map[string]any, key string) map[string]string {
	raw, ok := payload[key]
	if !ok {
		return map[string]string{}
	}
	result := map[string]string{}
	switch tv := raw.(type) {
	case map[string]any:
		for k, v := range tv {
			result[strings.TrimSpace(k)] = appRunnerValueString(v)
		}
	case []any:
		for _, item := range tv {
			entry := appRunnerMapFromAny(item)
			k := appRunnerStringFromMap(entry, "Key", "")
			if k == "" {
				continue
			}
			result[k] = appRunnerStringFromMap(entry, "Value", "")
		}
	}
	return result
}

func appRunnerTagKeys(raw any) []string {
	switch tv := raw.(type) {
	case []string:
		out := make([]string, 0, len(tv))
		for _, k := range tv {
			k = strings.TrimSpace(k)
			if k != "" {
				out = append(out, k)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(tv))
		for _, item := range tv {
			k := strings.TrimSpace(appRunnerValueString(item))
			if k != "" {
				out = append(out, k)
			}
		}
		return out
	case string:
		trimmed := strings.TrimSpace(tv)
		if trimmed == "" {
			return nil
		}
		if strings.Contains(trimmed, ",") {
			parts := strings.Split(trimmed, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return []string{trimmed}
	default:
		return nil
	}
}

func appRunnerTagList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func appRunnerSortedMapValues(m map[string]map[string]any, sortKey string) []map[string]any {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		left := appRunnerStringFromMap(m[keys[i]], sortKey, keys[i])
		right := appRunnerStringFromMap(m[keys[j]], sortKey, keys[j])
		if left == right {
			return keys[i] < keys[j]
		}
		return left < right
	})
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m[key])
	}
	return out
}

func appRunnerString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if val, ok := payload[key]; ok {
		if s := strings.TrimSpace(appRunnerValueString(val)); s != "" {
			return s
		}
	}
	return fallback
}

func appRunnerBool(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	val, ok := payload[key]
	if !ok {
		return fallback
	}
	switch tv := val.(type) {
	case bool:
		return tv
	case string:
		s := strings.TrimSpace(strings.ToLower(tv))
		if s == "true" {
			return true
		}
		if s == "false" {
			return false
		}
	}
	return fallback
}

func appRunnerMapFromAny(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func appRunnerStringSlice(v any) []string {
	switch tv := v.(type) {
	case []string:
		out := make([]string, 0, len(tv))
		for _, item := range tv {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(tv))
		for _, item := range tv {
			s := strings.TrimSpace(appRunnerValueString(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(tv)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

func appRunnerStringSliceToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func appRunnerStringFromMap(m map[string]any, key, fallback string) string {
	if val, ok := m[key]; ok {
		if s := strings.TrimSpace(appRunnerValueString(val)); s != "" {
			return s
		}
	}
	return fallback
}

func appRunnerBoolFromMap(m map[string]any, key string, fallback bool) bool {
	val, ok := m[key]
	if !ok {
		return fallback
	}
	switch tv := val.(type) {
	case bool:
		return tv
	case string:
		s := strings.TrimSpace(strings.ToLower(tv))
		if s == "true" {
			return true
		}
		if s == "false" {
			return false
		}
	}
	return fallback
}

func appRunnerIntFromMap(m map[string]any, key string, fallback int) int {
	val, ok := m[key]
	if !ok {
		return fallback
	}
	switch tv := val.(type) {
	case int:
		return tv
	case int32:
		return int(tv)
	case int64:
		return int(tv)
	case float64:
		return int(tv)
	case float32:
		return int(tv)
	}
	return fallback
}

func appRunnerCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = appRunnerCloneValue(value)
	}
	return out
}

func appRunnerCloneValue(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return appRunnerCloneMap(tv)
	case []any:
		out := make([]any, 0, len(tv))
		for _, item := range tv {
			out = append(out, appRunnerCloneValue(item))
		}
		return out
	default:
		return tv
	}
}

func appRunnerValueString(v any) string {
	switch tv := v.(type) {
	case string:
		return tv
	case fmt.Stringer:
		return tv.String()
	case int:
		return fmt.Sprintf("%d", tv)
	case int32:
		return fmt.Sprintf("%d", tv)
	case int64:
		return fmt.Sprintf("%d", tv)
	case float64:
		return strconv.FormatFloat(tv, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(tv), 'f', -1, 32)
	case bool:
		if tv {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", tv)
	}
}

func appRunnerFirstMapValue(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return m[keys[0]]
}

func appRunnerServiceDNSTarget(serviceArn string) string {
	name := appRunnerResourceNameFromARN(serviceArn)
	if name == "" {
		name = "stackyard-service"
	}
	return fmt.Sprintf("%s.%s.apprunner.amazonaws.com", name, appRunnerDefaultRegion)
}

func appRunnerResourceNameFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func appRunnerServiceARN(name, id string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:service/%s/%s", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, id)
}

func appRunnerConnectionARN(name, id string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:connection/%s/%s", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, id)
}

func appRunnerAutoScalingARN(name string, revision int) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:autoscalingconfiguration/%s/%d", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, revision)
}

func appRunnerObservabilityARN(name string, revision int) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:observabilityconfiguration/%s/%d", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, revision)
}

func appRunnerVpcConnectorARN(name, id string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:vpcconnector/%s/%s", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, id)
}

func appRunnerVpcIngressARN(name, id string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:vpcingressconnection/%s/%s", appRunnerDefaultRegion, appRunnerDefaultAccountID, name, id)
}
