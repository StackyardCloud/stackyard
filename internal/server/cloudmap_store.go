package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudMapStore struct {
	mu                sync.Mutex
	nextID            int64
	namespaces        map[string]map[string]any
	services          map[string]map[string]any
	operations        map[string]map[string]any
	instances         map[string]map[string]map[string]any
	serviceAttributes map[string]map[string]string
	tags              map[string]map[string]string
}

func newCloudMapStore() *cloudMapStore {
	now := cloudMapNow()

	namespaceID := "ns-000000000001"
	namespaceARN := cloudMapNamespaceARN(namespaceID)
	serviceID := "srv-000000000001"
	serviceARN := cloudMapServiceARN(serviceID)
	operationID := "op-000000000001"

	namespace := map[string]any{
		"Id":           namespaceID,
		"Arn":          namespaceARN,
		"Name":         "stackyard.local",
		"Type":         "HTTP",
		"Description":  "stackyard namespace",
		"ServiceCount": 1,
		"Properties": map[string]any{
			"HttpProperties": map[string]any{"HttpName": "stackyard.local"},
		},
		"CreateDate": now,
	}
	service := map[string]any{
		"Id":            serviceID,
		"Arn":           serviceARN,
		"Name":          "stackyard-service",
		"NamespaceId":   namespaceID,
		"Description":   "stackyard service",
		"InstanceCount": 1,
		"DnsConfig": map[string]any{
			"RoutingPolicy": "MULTIVALUE",
			"DnsRecords": []any{
				map[string]any{"Type": "A", "TTL": int64(60)},
			},
		},
		"CreateDate": now,
	}
	operation := map[string]any{
		"Id":         operationID,
		"Type":       "CREATE_NAMESPACE",
		"Status":     "SUCCESS",
		"CreateDate": now,
		"UpdateDate": now,
		"Targets": map[string]any{
			"NAMESPACE": namespaceID,
		},
	}
	instance := map[string]any{
		"Id":               "instance-000001",
		"CreatorRequestId": "stackyard",
		"Attributes": map[string]any{
			"AWS_INSTANCE_IPV4": "10.0.0.10",
		},
		"HealthStatus": "HEALTHY",
	}

	return &cloudMapStore{
		nextID: 2,
		namespaces: map[string]map[string]any{
			namespaceID: namespace,
		},
		services: map[string]map[string]any{
			serviceID: service,
		},
		operations: map[string]map[string]any{
			operationID: operation,
		},
		instances: map[string]map[string]map[string]any{
			serviceID: {
				"instance-000001": instance,
			},
		},
		serviceAttributes: map[string]map[string]string{
			serviceID: {"stage": "dev"},
		},
		tags: map[string]map[string]string{
			namespaceARN: {"stackyard": "true"},
			serviceARN:   {"stackyard": "true"},
		},
	}
}

func (s *cloudMapStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateHttpNamespace":
		return map[string]any{"OperationId": s.createNamespaceLocked(payload, "HTTP")}
	case "CreatePrivateDnsNamespace":
		return map[string]any{"OperationId": s.createNamespaceLocked(payload, "DNS_PRIVATE")}
	case "CreatePublicDnsNamespace":
		return map[string]any{"OperationId": s.createNamespaceLocked(payload, "DNS_PUBLIC")}
	case "CreateService":
		return map[string]any{"Service": s.createServiceLocked(payload)}
	case "DeleteNamespace":
		namespaceID := cloudMapPayloadString(payload, "Id", s.firstNamespaceIDLocked())
		delete(s.namespaces, namespaceID)
		for serviceID, service := range s.services {
			if strings.TrimSpace(fmt.Sprintf("%v", service["NamespaceId"])) == namespaceID {
				delete(s.services, serviceID)
				delete(s.instances, serviceID)
				delete(s.serviceAttributes, serviceID)
				delete(s.tags, cloudMapServiceARN(serviceID))
			}
		}
		s.refreshNamespaceServiceCountsLocked()
		return map[string]any{"OperationId": s.createOperationLocked("DELETE_NAMESPACE", map[string]string{"NAMESPACE": namespaceID})}
	case "DeleteService":
		serviceID := cloudMapPayloadString(payload, "Id", s.firstServiceIDLocked())
		delete(s.services, serviceID)
		delete(s.instances, serviceID)
		delete(s.serviceAttributes, serviceID)
		delete(s.tags, cloudMapServiceARN(serviceID))
		s.refreshNamespaceServiceCountsLocked()
		return map[string]any{}
	case "DeleteServiceAttributes":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		attrs := s.ensureServiceAttributesLocked(serviceID)
		for _, key := range cloudMapPayloadStringSlice(payload, "Attributes") {
			delete(attrs, key)
		}
		return map[string]any{}
	case "DeregisterInstance":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		instanceID := cloudMapPayloadString(payload, "InstanceId", s.firstInstanceIDLocked(serviceID))
		if s.instances[serviceID] != nil {
			delete(s.instances[serviceID], instanceID)
		}
		s.refreshServiceInstanceCountsLocked()
		return map[string]any{"OperationId": s.createOperationLocked("DEREGISTER_INSTANCE", map[string]string{"SERVICE": serviceID})}
	case "DiscoverInstances":
		namespaceName := cloudMapPayloadString(payload, "NamespaceName", "stackyard.local")
		serviceName := cloudMapPayloadString(payload, "ServiceName", "stackyard-service")
		serviceID := s.findServiceByNameLocked(serviceName)
		if serviceID == "" {
			serviceID = s.firstServiceIDLocked()
		}
		out := make([]any, 0)
		for _, item := range s.sortedInstancesLocked(serviceID) {
			m, _ := item.(map[string]any)
			out = append(out, map[string]any{
				"NamespaceName": namespaceName,
				"ServiceName":   serviceName,
				"InstanceId":    cloudMapPayloadString(m, "Id", ""),
				"Attributes":    cloudMapCloneAny(m["Attributes"]),
			})
		}
		return map[string]any{"Instances": out}
	case "DiscoverInstancesRevision":
		return map[string]any{"InstancesRevision": int64(1)}
	case "GetInstance":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		instanceID := cloudMapPayloadString(payload, "InstanceId", s.firstInstanceIDLocked(serviceID))
		return map[string]any{"Instance": cloudMapCloneMap(s.ensureInstanceLocked(serviceID, instanceID))}
	case "GetInstancesHealthStatus":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		status := map[string]any{}
		requested := cloudMapPayloadStringSlice(payload, "Instances")
		if len(requested) == 0 {
			requested = []string{s.firstInstanceIDLocked(serviceID)}
		}
		for _, instanceID := range requested {
			instance := s.ensureInstanceLocked(serviceID, instanceID)
			health := cloudMapPayloadString(instance, "HealthStatus", "HEALTHY")
			status[instanceID] = health
		}
		return map[string]any{"Status": status, "NextToken": ""}
	case "GetNamespace":
		namespaceID := cloudMapPayloadString(payload, "Id", s.firstNamespaceIDLocked())
		return map[string]any{"Namespace": cloudMapCloneMap(s.ensureNamespaceLocked(namespaceID))}
	case "GetOperation":
		operationID := cloudMapPayloadString(payload, "OperationId", s.firstOperationIDLocked())
		return map[string]any{"Operation": cloudMapCloneMap(s.ensureOperationLocked(operationID))}
	case "GetService":
		serviceID := cloudMapPayloadString(payload, "Id", s.firstServiceIDLocked())
		return map[string]any{"Service": cloudMapCloneMap(s.ensureServiceLocked(serviceID))}
	case "GetServiceAttributes":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		return map[string]any{"ServiceAttributes": cloudMapCloneStringMap(s.ensureServiceAttributesLocked(serviceID))}
	case "ListInstances":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		return map[string]any{"Instances": s.sortedInstancesLocked(serviceID), "NextToken": ""}
	case "ListNamespaces":
		return map[string]any{"Namespaces": s.sortedNamespacesLocked(), "NextToken": ""}
	case "ListOperations":
		return map[string]any{"Operations": s.sortedOperationsLocked(), "NextToken": ""}
	case "ListServices":
		return map[string]any{"Services": s.sortedServicesLocked(), "NextToken": ""}
	case "ListTagsForResource":
		resourceARN := cloudMapPayloadString(payload, "ResourceARN", cloudMapNamespaceARN(s.firstNamespaceIDLocked()))
		return map[string]any{"Tags": s.tagsListLocked(resourceARN)}
	case "RegisterInstance":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		instanceID := cloudMapPayloadString(payload, "InstanceId", "instance-000001")
		instance := s.ensureInstanceLocked(serviceID, instanceID)
		instance["CreatorRequestId"] = cloudMapPayloadString(payload, "CreatorRequestId", "stackyard")
		instance["Attributes"] = cloudMapPayloadStringMapAny(payload, "Attributes", map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.10"})
		instance["HealthStatus"] = cloudMapPayloadString(instance, "HealthStatus", "HEALTHY")
		s.refreshServiceInstanceCountsLocked()
		return map[string]any{"OperationId": s.createOperationLocked("REGISTER_INSTANCE", map[string]string{"SERVICE": serviceID})}
	case "TagResource":
		resourceARN := cloudMapPayloadString(payload, "ResourceARN", cloudMapNamespaceARN(s.firstNamespaceIDLocked()))
		s.applyTagsLocked(resourceARN, payload)
		return map[string]any{}
	case "UntagResource":
		resourceARN := cloudMapPayloadString(payload, "ResourceARN", cloudMapNamespaceARN(s.firstNamespaceIDLocked()))
		s.removeTagsLocked(resourceARN, payload)
		return map[string]any{}
	case "UpdateHttpNamespace":
		return map[string]any{"OperationId": s.updateNamespaceLocked(payload, "HTTP")}
	case "UpdateInstanceCustomHealthStatus":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		instanceID := cloudMapPayloadString(payload, "InstanceId", s.firstInstanceIDLocked(serviceID))
		instance := s.ensureInstanceLocked(serviceID, instanceID)
		instance["HealthStatus"] = cloudMapPayloadString(payload, "Status", "HEALTHY")
		return map[string]any{}
	case "UpdatePrivateDnsNamespace":
		return map[string]any{"OperationId": s.updateNamespaceLocked(payload, "DNS_PRIVATE")}
	case "UpdatePublicDnsNamespace":
		return map[string]any{"OperationId": s.updateNamespaceLocked(payload, "DNS_PUBLIC")}
	case "UpdateService":
		serviceID := cloudMapPayloadString(payload, "Id", s.firstServiceIDLocked())
		service := s.ensureServiceLocked(serviceID)
		change := cloudMapPayloadMap(payload, "Service")
		if desc := cloudMapPayloadString(change, "Description", ""); desc != "" {
			service["Description"] = desc
		}
		if dns := cloudMapPayloadMap(change, "DnsConfig"); len(dns) > 0 {
			service["DnsConfig"] = cloudMapCloneMap(dns)
		}
		return map[string]any{"OperationId": s.createOperationLocked("UPDATE_SERVICE", map[string]string{"SERVICE": serviceID})}
	case "UpdateServiceAttributes":
		serviceID := cloudMapPayloadString(payload, "ServiceId", s.firstServiceIDLocked())
		attrs := s.ensureServiceAttributesLocked(serviceID)
		for key, value := range cloudMapPayloadStringMap(payload, "Attributes") {
			attrs[key] = value
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *cloudMapStore) createNamespaceLocked(payload map[string]any, namespaceType string) string {
	namespaceID := s.nextIdentifierLocked("ns")
	name := cloudMapPayloadString(payload, "Name", "stackyard.local")
	namespace := map[string]any{
		"Id":           namespaceID,
		"Arn":          cloudMapNamespaceARN(namespaceID),
		"Name":         name,
		"Type":         namespaceType,
		"Description":  cloudMapPayloadString(payload, "Description", ""),
		"ServiceCount": 0,
		"CreateDate":   cloudMapNow(),
	}
	if namespaceType == "HTTP" {
		namespace["Properties"] = map[string]any{
			"HttpProperties": map[string]any{"HttpName": name},
		}
	} else {
		namespace["Properties"] = map[string]any{
			"DnsProperties": map[string]any{
				"HostedZoneId": "Z0000000000000000000",
			},
		}
	}
	s.namespaces[namespaceID] = namespace
	s.tags[cloudMapNamespaceARN(namespaceID)] = cloudMapTagsFromPayload(payload, "Tags")
	return s.createOperationLocked("CREATE_NAMESPACE", map[string]string{"NAMESPACE": namespaceID})
}

func (s *cloudMapStore) updateNamespaceLocked(payload map[string]any, namespaceType string) string {
	namespaceID := cloudMapPayloadString(payload, "Id", s.firstNamespaceIDLocked())
	namespace := s.ensureNamespaceLocked(namespaceID)
	namespace["Type"] = namespaceType
	change := cloudMapPayloadMap(payload, "Namespace")
	if desc := cloudMapPayloadString(change, "Description", ""); desc != "" {
		namespace["Description"] = desc
	}
	return s.createOperationLocked("UPDATE_NAMESPACE", map[string]string{"NAMESPACE": namespaceID})
}

func (s *cloudMapStore) createServiceLocked(payload map[string]any) map[string]any {
	serviceID := s.nextIdentifierLocked("srv")
	namespaceID := cloudMapPayloadString(payload, "NamespaceId", s.firstNamespaceIDLocked())
	service := map[string]any{
		"Id":            serviceID,
		"Arn":           cloudMapServiceARN(serviceID),
		"Name":          cloudMapPayloadString(payload, "Name", "stackyard-service"),
		"NamespaceId":   namespaceID,
		"Description":   cloudMapPayloadString(payload, "Description", ""),
		"InstanceCount": 0,
		"CreateDate":    cloudMapNow(),
	}
	dnsConfig := cloudMapPayloadMap(payload, "DnsConfig")
	if len(dnsConfig) > 0 {
		service["DnsConfig"] = cloudMapCloneMap(dnsConfig)
	} else {
		service["DnsConfig"] = map[string]any{
			"RoutingPolicy": "MULTIVALUE",
			"DnsRecords": []any{
				map[string]any{"Type": "A", "TTL": int64(60)},
			},
		}
	}
	s.services[serviceID] = service
	s.serviceAttributes[serviceID] = cloudMapPayloadStringMap(payload, "Attributes")
	s.tags[cloudMapServiceARN(serviceID)] = cloudMapTagsFromPayload(payload, "Tags")
	s.refreshNamespaceServiceCountsLocked()
	return cloudMapCloneMap(service)
}

func (s *cloudMapStore) createOperationLocked(opType string, targets map[string]string) string {
	operationID := s.nextIdentifierLocked("op")
	now := cloudMapNow()
	targetMap := map[string]any{}
	for key, value := range targets {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		targetMap[key] = value
	}
	s.operations[operationID] = map[string]any{
		"Id":         operationID,
		"Type":       opType,
		"Status":     "SUCCESS",
		"CreateDate": now,
		"UpdateDate": now,
		"Targets":    targetMap,
	}
	return operationID
}

func (s *cloudMapStore) nextIdentifierLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%012d", strings.TrimSpace(prefix), id)
}

func (s *cloudMapStore) firstNamespaceIDLocked() string {
	if len(s.namespaces) == 0 {
		return "ns-000000000001"
	}
	keys := make([]string, 0, len(s.namespaces))
	for id := range s.namespaces {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudMapStore) firstServiceIDLocked() string {
	if len(s.services) == 0 {
		return "srv-000000000001"
	}
	keys := make([]string, 0, len(s.services))
	for id := range s.services {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudMapStore) firstOperationIDLocked() string {
	if len(s.operations) == 0 {
		return "op-000000000001"
	}
	keys := make([]string, 0, len(s.operations))
	for id := range s.operations {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudMapStore) firstInstanceIDLocked(serviceID string) string {
	instances := s.instances[strings.TrimSpace(serviceID)]
	if len(instances) == 0 {
		return "instance-000001"
	}
	keys := make([]string, 0, len(instances))
	for id := range instances {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudMapStore) ensureNamespaceLocked(namespaceID string) map[string]any {
	namespaceID = strings.TrimSpace(namespaceID)
	if namespaceID == "" {
		namespaceID = s.firstNamespaceIDLocked()
	}
	if namespace := s.namespaces[namespaceID]; namespace != nil {
		return namespace
	}
	namespace := map[string]any{
		"Id":           namespaceID,
		"Arn":          cloudMapNamespaceARN(namespaceID),
		"Name":         "stackyard.local",
		"Type":         "HTTP",
		"Description":  "",
		"ServiceCount": 0,
		"Properties": map[string]any{
			"HttpProperties": map[string]any{"HttpName": "stackyard.local"},
		},
		"CreateDate": cloudMapNow(),
	}
	s.namespaces[namespaceID] = namespace
	return namespace
}

func (s *cloudMapStore) ensureServiceLocked(serviceID string) map[string]any {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		serviceID = s.firstServiceIDLocked()
	}
	if service := s.services[serviceID]; service != nil {
		return service
	}
	namespaceID := s.firstNamespaceIDLocked()
	service := map[string]any{
		"Id":            serviceID,
		"Arn":           cloudMapServiceARN(serviceID),
		"Name":          "stackyard-service",
		"NamespaceId":   namespaceID,
		"Description":   "",
		"InstanceCount": 0,
		"DnsConfig": map[string]any{
			"RoutingPolicy": "MULTIVALUE",
			"DnsRecords": []any{
				map[string]any{"Type": "A", "TTL": int64(60)},
			},
		},
		"CreateDate": cloudMapNow(),
	}
	s.services[serviceID] = service
	if s.instances[serviceID] == nil {
		s.instances[serviceID] = map[string]map[string]any{}
	}
	if s.serviceAttributes[serviceID] == nil {
		s.serviceAttributes[serviceID] = map[string]string{}
	}
	return service
}

func (s *cloudMapStore) ensureOperationLocked(operationID string) map[string]any {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		operationID = s.firstOperationIDLocked()
	}
	if operation := s.operations[operationID]; operation != nil {
		return operation
	}
	now := cloudMapNow()
	operation := map[string]any{
		"Id":         operationID,
		"Type":       "UPDATE_SERVICE",
		"Status":     "SUCCESS",
		"CreateDate": now,
		"UpdateDate": now,
		"Targets":    map[string]any{},
	}
	s.operations[operationID] = operation
	return operation
}

func (s *cloudMapStore) ensureInstanceLocked(serviceID, instanceID string) map[string]any {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		serviceID = s.firstServiceIDLocked()
	}
	_ = s.ensureServiceLocked(serviceID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = s.firstInstanceIDLocked(serviceID)
	}
	if s.instances[serviceID] == nil {
		s.instances[serviceID] = map[string]map[string]any{}
	}
	if instance := s.instances[serviceID][instanceID]; instance != nil {
		return instance
	}
	instance := map[string]any{
		"Id":               instanceID,
		"CreatorRequestId": "stackyard",
		"Attributes": map[string]any{
			"AWS_INSTANCE_IPV4": "10.0.0.10",
		},
		"HealthStatus": "HEALTHY",
	}
	s.instances[serviceID][instanceID] = instance
	return instance
}

func (s *cloudMapStore) ensureServiceAttributesLocked(serviceID string) map[string]string {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		serviceID = s.firstServiceIDLocked()
	}
	if s.serviceAttributes[serviceID] == nil {
		s.serviceAttributes[serviceID] = map[string]string{}
	}
	return s.serviceAttributes[serviceID]
}

func (s *cloudMapStore) sortedNamespacesLocked() []any {
	keys := make([]string, 0, len(s.namespaces))
	for id := range s.namespaces {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		ns := s.ensureNamespaceLocked(id)
		out = append(out, map[string]any{
			"Id":           ns["Id"],
			"Arn":          ns["Arn"],
			"Name":         ns["Name"],
			"Type":         ns["Type"],
			"Description":  ns["Description"],
			"ServiceCount": ns["ServiceCount"],
			"CreateDate":   ns["CreateDate"],
		})
	}
	return out
}

func (s *cloudMapStore) sortedServicesLocked() []any {
	keys := make([]string, 0, len(s.services))
	for id := range s.services {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		service := s.ensureServiceLocked(id)
		out = append(out, map[string]any{
			"Id":            service["Id"],
			"Arn":           service["Arn"],
			"Name":          service["Name"],
			"Description":   service["Description"],
			"InstanceCount": service["InstanceCount"],
			"CreateDate":    service["CreateDate"],
		})
	}
	return out
}

func (s *cloudMapStore) sortedOperationsLocked() []any {
	keys := make([]string, 0, len(s.operations))
	for id := range s.operations {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		op := s.ensureOperationLocked(id)
		out = append(out, map[string]any{
			"Id":         op["Id"],
			"Status":     op["Status"],
			"CreateDate": op["CreateDate"],
			"UpdateDate": op["UpdateDate"],
		})
	}
	return out
}

func (s *cloudMapStore) sortedInstancesLocked(serviceID string) []any {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		serviceID = s.firstServiceIDLocked()
	}
	if s.instances[serviceID] == nil {
		s.instances[serviceID] = map[string]map[string]any{}
	}
	keys := make([]string, 0, len(s.instances[serviceID]))
	for id := range s.instances[serviceID] {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		instance := s.instances[serviceID][id]
		out = append(out, map[string]any{
			"Id":         instance["Id"],
			"Attributes": cloudMapCloneAny(instance["Attributes"]),
		})
	}
	return out
}

func (s *cloudMapStore) refreshNamespaceServiceCountsLocked() {
	counts := map[string]int{}
	for namespaceID := range s.namespaces {
		counts[namespaceID] = 0
	}
	for _, service := range s.services {
		namespaceID := strings.TrimSpace(fmt.Sprintf("%v", service["NamespaceId"]))
		if namespaceID == "" {
			continue
		}
		counts[namespaceID]++
	}
	for namespaceID, namespace := range s.namespaces {
		namespace["ServiceCount"] = counts[namespaceID]
	}
}

func (s *cloudMapStore) refreshServiceInstanceCountsLocked() {
	for serviceID, service := range s.services {
		service["InstanceCount"] = len(s.instances[serviceID])
	}
}

func (s *cloudMapStore) findServiceByNameLocked(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for serviceID, service := range s.services {
		if strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", service["Name"])), name) {
			return serviceID
		}
	}
	return ""
}

func (s *cloudMapStore) applyTagsLocked(resourceARN string, payload map[string]any) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	for key, value := range cloudMapTagsFromPayload(payload, "Tags") {
		s.tags[resourceARN][key] = value
	}
}

func (s *cloudMapStore) removeTagsLocked(resourceARN string, payload map[string]any) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	for _, key := range cloudMapPayloadStringSlice(payload, "TagKeys") {
		delete(s.tags[resourceARN], key)
	}
}

func (s *cloudMapStore) tagsListLocked(resourceARN string) []any {
	resourceARN = strings.TrimSpace(resourceARN)
	tags := s.tags[resourceARN]
	if len(tags) == 0 {
		return []any{}
	}
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

func cloudMapPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for payloadKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(payloadKey), strings.TrimSpace(key)) {
			if out := strings.TrimSpace(fmt.Sprintf("%v", value)); out != "" && out != "%!v(<nil>)" {
				return out
			}
		}
	}
	return fallback
}

func cloudMapPayloadMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	for payloadKey, value := range payload {
		if !strings.EqualFold(strings.TrimSpace(payloadKey), strings.TrimSpace(key)) {
			continue
		}
		if out, ok := value.(map[string]any); ok && out != nil {
			return out
		}
	}
	return map[string]any{}
}

func cloudMapPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return []string{}
	}
	for payloadKey, value := range payload {
		if !strings.EqualFold(strings.TrimSpace(payloadKey), strings.TrimSpace(key)) {
			continue
		}
		switch typed := value.(type) {
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				itemString := strings.TrimSpace(fmt.Sprintf("%v", item))
				if itemString != "" && itemString != "%!v(<nil>)" {
					out = append(out, itemString)
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				itemString := strings.TrimSpace(item)
				if itemString != "" {
					out = append(out, itemString)
				}
			}
			return out
		}
	}
	return []string{}
}

func cloudMapPayloadStringMap(payload map[string]any, key string) map[string]string {
	return cloudMapStringMapFromAny(cloudMapPayloadValue(payload, key))
}

func cloudMapPayloadStringMapAny(payload map[string]any, key string, fallback map[string]any) map[string]any {
	value := cloudMapPayloadValue(payload, key)
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if len(out) > 0 {
			return out
		}
	case map[string]string:
		out := map[string]any{}
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(v)
		}
		if len(out) > 0 {
			return out
		}
	}
	return cloudMapCloneMap(fallback)
}

func cloudMapPayloadValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	for payloadKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(payloadKey), strings.TrimSpace(key)) {
			return value
		}
	}
	return nil
}

func cloudMapTagsFromPayload(payload map[string]any, key string) map[string]string {
	value := cloudMapPayloadValue(payload, key)
	tags := map[string]string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			tagKey := cloudMapPayloadString(tag, "Key", "")
			if tagKey == "" {
				continue
			}
			tags[tagKey] = cloudMapPayloadString(tag, "Value", "")
		}
	case map[string]any:
		for tagKey, tagValue := range typed {
			tagKey = strings.TrimSpace(tagKey)
			if tagKey == "" {
				continue
			}
			tags[tagKey] = strings.TrimSpace(fmt.Sprintf("%v", tagValue))
		}
	case map[string]string:
		for tagKey, tagValue := range typed {
			tagKey = strings.TrimSpace(tagKey)
			if tagKey == "" {
				continue
			}
			tags[tagKey] = strings.TrimSpace(tagValue)
		}
	}
	return tags
}

func cloudMapStringMapFromAny(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	case map[string]string:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(value)
		}
	}
	return out
}

func cloudMapCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloudMapCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloudMapCloneAny(value)
	}
	return out
}

func cloudMapCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloudMapCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloudMapCloneAny(item))
		}
		return out
	case map[string]string:
		return cloudMapCloneStringMap(typed)
	default:
		return typed
	}
}

func cloudMapNamespaceARN(namespaceID string) string {
	return "arn:aws:servicediscovery:us-east-1:123456789012:namespace/" + strings.TrimSpace(namespaceID)
}

func cloudMapServiceARN(serviceID string) string {
	return "arn:aws:servicediscovery:us-east-1:123456789012:service/" + strings.TrimSpace(serviceID)
}

func cloudMapNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
