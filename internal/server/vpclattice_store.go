package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type vpcLatticeStore struct {
	mu sync.Mutex

	nextID int64

	services                              map[string]map[string]any
	listeners                             map[string]map[string]any
	rules                                 map[string]map[string]any
	serviceNetworks                       map[string]map[string]any
	targetGroups                          map[string]map[string]any
	resourceConfigurations                map[string]map[string]any
	resourceGateways                      map[string]map[string]any
	accessLogSubscriptions                map[string]map[string]any
	domainVerifications                   map[string]map[string]any
	serviceNetworkServiceAssociations     map[string]map[string]any
	serviceNetworkVpcAssociations         map[string]map[string]any
	serviceNetworkResourceAssociations    map[string]map[string]any
	resourceEndpointAssociations          map[string]map[string]any
	tags                                  map[string]map[string]string
	authPolicies                          map[string]string
	resourcePolicies                      map[string]string
	targets                               map[string][]map[string]any
	serviceNetworkVpcEndpointAssociations []map[string]any
}

func newVPCLatticeStore() *vpcLatticeStore {
	now := time.Now().UTC().Format(time.RFC3339)
	svcID := "svc-00000000000000001"
	listenerID := "listener-00000000000000001"
	ruleID := "rule-00000000000000001"
	snID := "sn-00000000000000001"
	tgID := "tg-00000000000000001"
	rcID := "rcfg-00000000000000001"
	rgID := "rgw-00000000000000001"
	alsID := "als-00000000000000001"
	dvID := "dv-00000000000000001"
	snsaID := "snsa-00000000000000001"
	snvaID := "snva-00000000000000001"
	snraID := "snra-00000000000000001"
	reaID := "rea-00000000000000001"

	svcArn := vpclatticeARN("service", svcID)
	listenerArn := vpclatticeARN("listener", listenerID)
	ruleArn := vpclatticeARN("rule", ruleID)
	snArn := vpclatticeARN("servicenetwork", snID)
	tgArn := vpclatticeARN("targetgroup", tgID)
	rcArn := vpclatticeARN("resourceconfiguration", rcID)
	rgArn := vpclatticeARN("resourcegateway", rgID)
	alsArn := vpclatticeARN("accesslogsubscription", alsID)
	dvArn := vpclatticeARN("domainverification", dvID)
	snsaArn := vpclatticeARN("servicenetworkserviceassociation", snsaID)
	snvaArn := vpclatticeARN("servicenetworkvpcassociation", snvaID)
	snraArn := vpclatticeARN("servicenetworkresourceassociation", snraID)
	reaArn := vpclatticeARN("resourceendpointassociation", reaID)

	s := &vpcLatticeStore{
		nextID:                                2,
		services:                              map[string]map[string]any{},
		listeners:                             map[string]map[string]any{},
		rules:                                 map[string]map[string]any{},
		serviceNetworks:                       map[string]map[string]any{},
		targetGroups:                          map[string]map[string]any{},
		resourceConfigurations:                map[string]map[string]any{},
		resourceGateways:                      map[string]map[string]any{},
		accessLogSubscriptions:                map[string]map[string]any{},
		domainVerifications:                   map[string]map[string]any{},
		serviceNetworkServiceAssociations:     map[string]map[string]any{},
		serviceNetworkVpcAssociations:         map[string]map[string]any{},
		serviceNetworkResourceAssociations:    map[string]map[string]any{},
		resourceEndpointAssociations:          map[string]map[string]any{},
		tags:                                  map[string]map[string]string{},
		authPolicies:                          map[string]string{},
		resourcePolicies:                      map[string]string{},
		targets:                               map[string][]map[string]any{},
		serviceNetworkVpcEndpointAssociations: []map[string]any{},
	}

	s.services[svcID] = map[string]any{
		"id":        svcID,
		"arn":       svcArn,
		"name":      "stackyard-service",
		"status":    "ACTIVE",
		"authType":  "AWS_IAM",
		"dnsEntry":  map[string]any{"domainName": "stackyard-service.vpc-lattice.local", "hostedZoneId": "ZSTACKYARD"},
		"createdAt": now,
	}
	s.listeners[listenerID] = map[string]any{
		"id":         listenerID,
		"arn":        listenerArn,
		"name":       "stackyard-listener",
		"protocol":   "HTTP",
		"port":       80,
		"serviceId":  svcID,
		"serviceArn": svcArn,
		"status":     "ACTIVE",
		"defaultAction": map[string]any{
			"fixedResponse": map[string]any{"statusCode": 200},
		},
		"createdAt": now,
	}
	s.rules[ruleID] = map[string]any{
		"id":                 ruleID,
		"arn":                ruleArn,
		"name":               "stackyard-rule",
		"priority":           int64(10),
		"serviceIdentifier":  svcID,
		"listenerIdentifier": listenerID,
		"status":             "ACTIVE",
		"action": map[string]any{
			"fixedResponse": map[string]any{"statusCode": 200},
		},
		"match":     map[string]any{"httpMatch": map[string]any{"pathMatch": map[string]any{"match": map[string]any{"exact": "/"}}}},
		"createdAt": now,
	}
	s.serviceNetworks[snID] = map[string]any{
		"id":        snID,
		"arn":       snArn,
		"name":      "stackyard-service-network",
		"status":    "ACTIVE",
		"authType":  "AWS_IAM",
		"createdAt": now,
	}
	s.targetGroups[tgID] = map[string]any{
		"id":            tgID,
		"arn":           tgArn,
		"name":          "stackyard-target-group",
		"type":          "IP",
		"status":        "ACTIVE",
		"protocol":      "HTTP",
		"port":          80,
		"ipAddressType": "IPV4",
		"healthCheck":   map[string]any{"enabled": true, "protocol": "HTTP", "path": "/", "intervalSeconds": 30},
		"createdAt":     now,
	}
	s.resourceConfigurations[rcID] = map[string]any{
		"id":                        rcID,
		"arn":                       rcArn,
		"name":                      "stackyard-resource-config",
		"resourceConfigurationType": "SINGLE",
		"status":                    "ACTIVE",
		"createdAt":                 now,
	}
	s.resourceGateways[rgID] = map[string]any{
		"id":            rgID,
		"arn":           rgArn,
		"name":          "stackyard-resource-gateway",
		"status":        "ACTIVE",
		"vpcIdentifier": "vpc-00000000000000001",
		"createdAt":     now,
	}
	s.accessLogSubscriptions[alsID] = map[string]any{
		"id":                 alsID,
		"arn":                alsArn,
		"resourceIdentifier": svcID,
		"resourceArn":        svcArn,
		"destinationArn":     "arn:aws:s3:::stackyard-logs",
		"createdAt":          now,
	}
	s.domainVerifications[dvID] = map[string]any{
		"id":                 dvID,
		"arn":                dvArn,
		"domainName":         "stackyard.example.com",
		"status":             "SUCCESSFUL",
		"resourceIdentifier": svcID,
		"createdAt":          now,
	}
	s.serviceNetworkServiceAssociations[snsaID] = map[string]any{
		"id":                       snsaID,
		"arn":                      snsaArn,
		"serviceIdentifier":        svcID,
		"serviceArn":               svcArn,
		"serviceNetworkIdentifier": snID,
		"serviceNetworkArn":        snArn,
		"status":                   "ACTIVE",
		"createdAt":                now,
	}
	s.serviceNetworkVpcAssociations[snvaID] = map[string]any{
		"id":                       snvaID,
		"arn":                      snvaArn,
		"serviceNetworkIdentifier": snID,
		"serviceNetworkArn":        snArn,
		"vpcIdentifier":            "vpc-00000000000000001",
		"status":                   "ACTIVE",
		"createdAt":                now,
	}
	s.serviceNetworkResourceAssociations[snraID] = map[string]any{
		"id":                              snraID,
		"arn":                             snraArn,
		"serviceNetworkIdentifier":        snID,
		"serviceNetworkArn":               snArn,
		"resourceConfigurationIdentifier": rcID,
		"resourceConfigurationArn":        rcArn,
		"status":                          "ACTIVE",
		"createdAt":                       now,
	}
	s.resourceEndpointAssociations[reaID] = map[string]any{
		"id":                              reaID,
		"arn":                             reaArn,
		"resourceConfigurationIdentifier": rcID,
		"resourceConfigurationArn":        rcArn,
		"resourceGatewayIdentifier":       rgID,
		"resourceGatewayArn":              rgArn,
		"status":                          "ACTIVE",
		"createdAt":                       now,
	}

	s.serviceNetworkVpcEndpointAssociations = []map[string]any{
		{
			"serviceNetworkIdentifier": snID,
			"serviceNetworkArn":        snArn,
			"vpcEndpointId":            "vpce-00000000000000001",
			"vpcEndpointOwner":         "123456789012",
		},
	}

	s.targets[tgID] = []map[string]any{{"id": "10.0.0.10", "port": 80, "status": "HEALTHY", "reasonCode": "TARGET_HEALTHY"}}
	s.tags[svcArn] = map[string]string{"seed": "true"}
	s.tags[snArn] = map[string]string{"seed": "true"}
	s.tags[tgArn] = map[string]string{"seed": "true"}
	s.authPolicies[svcID] = `{"Version":"2012-10-17","Statement":[]}`
	s.resourcePolicies[svcArn] = `{"Version":"2012-10-17","Statement":[]}`

	return s
}

func (s *vpcLatticeStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	svcID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "serviceIdentifier"),
		vpclatticeStringAny(payload, "serviceIdentifier", "serviceId", "serviceIdentifier"),
		"svc-00000000000000001",
	)
	listenerID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "listenerIdentifier"),
		vpclatticeStringAny(payload, "listenerIdentifier", "listenerId"),
		"listener-00000000000000001",
	)
	ruleID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "ruleIdentifier"),
		vpclatticeStringAny(payload, "ruleIdentifier", "ruleId"),
		"rule-00000000000000001",
	)
	snID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "serviceNetworkIdentifier"),
		vpclatticeStringAny(payload, "serviceNetworkIdentifier", "serviceNetworkId"),
		"sn-00000000000000001",
	)
	tgID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "targetGroupIdentifier"),
		vpclatticeStringAny(payload, "targetGroupIdentifier", "targetGroupId"),
		"tg-00000000000000001",
	)
	rcID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "resourceConfigurationIdentifier"),
		vpclatticeStringAny(payload, "resourceConfigurationIdentifier", "resourceConfigurationId"),
		"rcfg-00000000000000001",
	)
	rgID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "resourceGatewayIdentifier"),
		vpclatticeStringAny(payload, "resourceGatewayIdentifier", "resourceGatewayId"),
		"rgw-00000000000000001",
	)
	alsID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "accessLogSubscriptionIdentifier"),
		vpclatticeStringAny(payload, "accessLogSubscriptionIdentifier", "accessLogSubscriptionId"),
		"als-00000000000000001",
	)
	dvID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "domainVerificationIdentifier"),
		vpclatticeStringAny(payload, "domainVerificationIdentifier", "domainVerificationId"),
		"dv-00000000000000001",
	)
	snsaID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "serviceNetworkServiceAssociationIdentifier"),
		vpclatticeStringAny(payload, "serviceNetworkServiceAssociationIdentifier", "serviceNetworkServiceAssociationId"),
		"snsa-00000000000000001",
	)
	snvaID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "serviceNetworkVpcAssociationIdentifier"),
		vpclatticeStringAny(payload, "serviceNetworkVpcAssociationIdentifier", "serviceNetworkVpcAssociationId"),
		"snva-00000000000000001",
	)
	snraID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "serviceNetworkResourceAssociationIdentifier"),
		vpclatticeStringAny(payload, "serviceNetworkResourceAssociationIdentifier", "serviceNetworkResourceAssociationId"),
		"snra-00000000000000001",
	)
	reaID := vpclatticeFirstNonEmpty(
		vpclatticePathParam(pathParams, "resourceEndpointAssociationIdentifier"),
		vpclatticeStringAny(payload, "resourceEndpointAssociationIdentifier", "resourceEndpointAssociationId"),
		"rea-00000000000000001",
	)

	svc := s.ensureServiceLocked(svcID)
	listener := s.ensureListenerLocked(listenerID, svcID)
	rule := s.ensureRuleLocked(ruleID, svcID, listenerID)
	sn := s.ensureServiceNetworkLocked(snID)
	tg := s.ensureTargetGroupLocked(tgID)
	rc := s.ensureResourceConfigurationLocked(rcID)
	rg := s.ensureResourceGatewayLocked(rgID)
	als := s.ensureAccessLogSubscriptionLocked(alsID, svcID)
	dv := s.ensureDomainVerificationLocked(dvID, svcID)
	snsa := s.ensureServiceNetworkServiceAssociationLocked(snsaID, snID, svcID)
	snva := s.ensureServiceNetworkVpcAssociationLocked(snvaID, snID)
	snra := s.ensureServiceNetworkResourceAssociationLocked(snraID, snID, rcID)
	rea := s.ensureResourceEndpointAssociationLocked(reaID, rcID, rgID)

	switch action {
	case "BatchUpdateRule":
		rule["updatedAt"] = now
		if updates, ok := payload["rules"].([]any); ok && len(updates) > 0 {
			rule["batchUpdate"] = vpclatticeCloneAny(updates)
		}
		return map[string]any{
			"successful":   []any{vpclatticeCloneMap(rule)},
			"unsuccessful": []any{},
		}

	case "CreateAccessLogSubscription", "UpdateAccessLogSubscription", "GetAccessLogSubscription":
		s.applyMapPayload(als, payload)
		als["updatedAt"] = now
		return vpclatticeCloneMap(als)
	case "DeleteAccessLogSubscription":
		delete(s.accessLogSubscriptions, alsID)
		return map[string]any{}
	case "ListAccessLogSubscriptions":
		return map[string]any{"items": s.listMaps(s.accessLogSubscriptions), "nextToken": ""}

	case "CreateListener", "UpdateListener", "GetListener":
		s.applyMapPayload(listener, payload)
		listener["updatedAt"] = now
		return vpclatticeCloneMap(listener)
	case "DeleteListener":
		delete(s.listeners, listenerID)
		return map[string]any{}
	case "ListListeners":
		return map[string]any{"items": s.listByField(s.listeners, "serviceId", svcID), "nextToken": ""}

	case "CreateResourceConfiguration", "UpdateResourceConfiguration", "GetResourceConfiguration":
		s.applyMapPayload(rc, payload)
		rc["updatedAt"] = now
		return vpclatticeCloneMap(rc)
	case "DeleteResourceConfiguration":
		delete(s.resourceConfigurations, rcID)
		return map[string]any{}
	case "ListResourceConfigurations":
		return map[string]any{"items": s.listMaps(s.resourceConfigurations), "nextToken": ""}

	case "CreateResourceGateway", "UpdateResourceGateway", "GetResourceGateway":
		s.applyMapPayload(rg, payload)
		rg["updatedAt"] = now
		return vpclatticeCloneMap(rg)
	case "DeleteResourceGateway":
		delete(s.resourceGateways, rgID)
		return map[string]any{}
	case "ListResourceGateways":
		return map[string]any{"items": s.listMaps(s.resourceGateways), "nextToken": ""}

	case "CreateRule", "UpdateRule", "GetRule":
		s.applyMapPayload(rule, payload)
		rule["updatedAt"] = now
		return vpclatticeCloneMap(rule)
	case "DeleteRule":
		delete(s.rules, ruleID)
		return map[string]any{}
	case "ListRules":
		return map[string]any{"items": s.listByField(s.rules, "listenerIdentifier", listenerID), "nextToken": ""}

	case "CreateService", "UpdateService", "GetService":
		s.applyMapPayload(svc, payload)
		svc["updatedAt"] = now
		return vpclatticeCloneMap(svc)
	case "DeleteService":
		delete(s.services, svcID)
		return map[string]any{}
	case "ListServices":
		return map[string]any{"items": s.listMaps(s.services), "nextToken": ""}

	case "CreateServiceNetwork", "UpdateServiceNetwork", "GetServiceNetwork":
		s.applyMapPayload(sn, payload)
		sn["updatedAt"] = now
		return vpclatticeCloneMap(sn)
	case "DeleteServiceNetwork":
		delete(s.serviceNetworks, snID)
		return map[string]any{}
	case "ListServiceNetworks":
		return map[string]any{"items": s.listMaps(s.serviceNetworks), "nextToken": ""}

	case "CreateServiceNetworkResourceAssociation", "GetServiceNetworkResourceAssociation":
		s.applyMapPayload(snra, payload)
		snra["updatedAt"] = now
		return vpclatticeCloneMap(snra)
	case "DeleteServiceNetworkResourceAssociation":
		delete(s.serviceNetworkResourceAssociations, snraID)
		return map[string]any{}
	case "ListServiceNetworkResourceAssociations":
		items := s.listMaps(s.serviceNetworkResourceAssociations)
		if filter := vpclatticeFirstNonEmpty(query.Get("serviceNetworkIdentifier"), vpclatticeStringAny(payload, "serviceNetworkIdentifier")); filter != "" {
			items = s.listByField(s.serviceNetworkResourceAssociations, "serviceNetworkIdentifier", filter)
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "CreateServiceNetworkServiceAssociation", "GetServiceNetworkServiceAssociation":
		s.applyMapPayload(snsa, payload)
		snsa["updatedAt"] = now
		return vpclatticeCloneMap(snsa)
	case "DeleteServiceNetworkServiceAssociation":
		delete(s.serviceNetworkServiceAssociations, snsaID)
		return map[string]any{}
	case "ListServiceNetworkServiceAssociations":
		items := s.listMaps(s.serviceNetworkServiceAssociations)
		if filter := vpclatticeFirstNonEmpty(query.Get("serviceIdentifier"), vpclatticeStringAny(payload, "serviceIdentifier")); filter != "" {
			items = s.listByField(s.serviceNetworkServiceAssociations, "serviceIdentifier", filter)
		}
		if filter := vpclatticeFirstNonEmpty(query.Get("serviceNetworkIdentifier"), vpclatticeStringAny(payload, "serviceNetworkIdentifier")); filter != "" {
			items = vpclatticeFilterMaps(items, "serviceNetworkIdentifier", filter)
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "CreateServiceNetworkVpcAssociation", "UpdateServiceNetworkVpcAssociation", "GetServiceNetworkVpcAssociation":
		s.applyMapPayload(snva, payload)
		snva["updatedAt"] = now
		return vpclatticeCloneMap(snva)
	case "DeleteServiceNetworkVpcAssociation":
		delete(s.serviceNetworkVpcAssociations, snvaID)
		return map[string]any{}
	case "ListServiceNetworkVpcAssociations":
		items := s.listMaps(s.serviceNetworkVpcAssociations)
		if filter := vpclatticeFirstNonEmpty(query.Get("serviceNetworkIdentifier"), vpclatticeStringAny(payload, "serviceNetworkIdentifier")); filter != "" {
			items = s.listByField(s.serviceNetworkVpcAssociations, "serviceNetworkIdentifier", filter)
		}
		if filter := vpclatticeFirstNonEmpty(query.Get("vpcIdentifier"), vpclatticeStringAny(payload, "vpcIdentifier")); filter != "" {
			items = vpclatticeFilterMaps(items, "vpcIdentifier", filter)
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "ListServiceNetworkVpcEndpointAssociations":
		items := make([]any, 0, len(s.serviceNetworkVpcEndpointAssociations))
		for _, item := range s.serviceNetworkVpcEndpointAssociations {
			items = append(items, vpclatticeCloneMap(item))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "CreateTargetGroup", "UpdateTargetGroup", "GetTargetGroup":
		s.applyMapPayload(tg, payload)
		tg["updatedAt"] = now
		return vpclatticeCloneMap(tg)
	case "DeleteTargetGroup":
		delete(s.targetGroups, tgID)
		delete(s.targets, tgID)
		return map[string]any{}
	case "ListTargetGroups":
		items := s.listMaps(s.targetGroups)
		if filter := vpclatticeFirstNonEmpty(query.Get("targetGroupType"), vpclatticeStringAny(payload, "targetGroupType", "type")); filter != "" {
			items = vpclatticeFilterMaps(items, "type", filter)
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "DeleteResourceEndpointAssociation":
		delete(s.resourceEndpointAssociations, reaID)
		return map[string]any{}
	case "GetResourceEndpointAssociation":
		s.applyMapPayload(rea, payload)
		rea["updatedAt"] = now
		return vpclatticeCloneMap(rea)
	case "ListResourceEndpointAssociations":
		return map[string]any{"items": s.listMaps(s.resourceEndpointAssociations), "nextToken": ""}

	case "DeleteDomainVerification":
		delete(s.domainVerifications, dvID)
		return map[string]any{}
	case "GetDomainVerification":
		s.applyMapPayload(dv, payload)
		dv["updatedAt"] = now
		return vpclatticeCloneMap(dv)
	case "ListDomainVerifications":
		return map[string]any{"items": s.listMaps(s.domainVerifications), "nextToken": ""}

	case "PutAuthPolicy":
		resourceIdentifier := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceIdentifier"), vpclatticeStringAny(payload, "resourceIdentifier"), svcID)
		policy := vpclatticeFirstNonEmpty(vpclatticeStringAny(payload, "policy", "Policy"), `{"Version":"2012-10-17","Statement":[]}`)
		s.authPolicies[resourceIdentifier] = policy
		return map[string]any{"resourceIdentifier": resourceIdentifier, "policy": policy}
	case "GetAuthPolicy":
		resourceIdentifier := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceIdentifier"), vpclatticeStringAny(payload, "resourceIdentifier"), svcID)
		policy := vpclatticeFirstNonEmpty(s.authPolicies[resourceIdentifier], `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{"resourceIdentifier": resourceIdentifier, "policy": policy}
	case "DeleteAuthPolicy":
		resourceIdentifier := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceIdentifier"), vpclatticeStringAny(payload, "resourceIdentifier"), svcID)
		delete(s.authPolicies, resourceIdentifier)
		return map[string]any{}

	case "PutResourcePolicy":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		policy := vpclatticeFirstNonEmpty(vpclatticeStringAny(payload, "policy", "Policy"), `{"Version":"2012-10-17","Statement":[]}`)
		s.resourcePolicies[resourceArn] = policy
		return map[string]any{"resourceArn": resourceArn, "policy": policy}
	case "GetResourcePolicy":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		policy := vpclatticeFirstNonEmpty(s.resourcePolicies[resourceArn], `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{"resourceArn": resourceArn, "policy": policy}
	case "DeleteResourcePolicy":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		delete(s.resourcePolicies, resourceArn)
		return map[string]any{}

	case "RegisterTargets":
		incoming := vpclatticeTargets(payload["targets"])
		if len(incoming) == 0 {
			incoming = []map[string]any{{"id": "10.0.0.10", "port": 80}}
		}
		s.targets[tgID] = append(s.targets[tgID], incoming...)
		return map[string]any{"successful": vpclatticeCloneAny(incoming), "unsuccessful": []any{}}
	case "DeregisterTargets":
		incoming := vpclatticeTargets(payload["targets"])
		if len(incoming) == 0 {
			incoming = []map[string]any{{"id": "10.0.0.10", "port": 80}}
		}
		return map[string]any{"successful": vpclatticeCloneAny(incoming), "unsuccessful": []any{}}
	case "ListTargets":
		targets := s.targets[tgID]
		if len(targets) == 0 {
			targets = []map[string]any{{"id": "10.0.0.10", "port": 80, "status": "HEALTHY", "reasonCode": "TARGET_HEALTHY"}}
		}
		items := make([]any, 0, len(targets))
		for _, target := range targets {
			items = append(items, vpclatticeCloneMap(target))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "TagResource":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		if s.tags[resourceArn] == nil {
			s.tags[resourceArn] = map[string]string{}
		}
		for k, v := range vpclatticeStringMap(payload["tags"]) {
			s.tags[resourceArn][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		tagKeys := vpclatticeStringList(payload["tagKeys"])
		if len(tagKeys) == 0 {
			tagKeys = vpclatticeStringList(payload["TagKeys"])
		}
		if len(tagKeys) == 0 {
			tagKeys = vpclatticeStringList(query["tagKeys"])
		}
		for _, key := range tagKeys {
			delete(s.tags[resourceArn], key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceArn := vpclatticeFirstNonEmpty(vpclatticePathParam(pathParams, "resourceArn"), vpclatticeStringAny(payload, "resourceArn"), vpclatticeStringAny(svc, "arn"))
		return map[string]any{"tags": vpclatticeCloneStringMap(s.tags[resourceArn])}
	}

	return map[string]any{}
}

func (s *vpcLatticeStore) ensureServiceLocked(id string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "svc-00000000000000001")
	if out := s.services[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":       id,
		"arn":      vpclatticeARN("service", id),
		"name":     fmt.Sprintf("stackyard-service-%s", strings.TrimPrefix(id, "svc-")),
		"status":   "ACTIVE",
		"authType": "AWS_IAM",
		"dnsEntry": map[string]any{"domainName": id + ".vpc-lattice.local", "hostedZoneId": "ZSTACKYARD"},
	}
	s.services[id] = out
	return out
}

func (s *vpcLatticeStore) ensureListenerLocked(id, serviceID string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "listener-00000000000000001")
	if out := s.listeners[id]; out != nil {
		return out
	}
	svc := s.ensureServiceLocked(serviceID)
	out := map[string]any{
		"id":         id,
		"arn":        vpclatticeARN("listener", id),
		"name":       fmt.Sprintf("listener-%s", strings.TrimPrefix(id, "listener-")),
		"serviceId":  vpclatticeStringAny(svc, "id"),
		"serviceArn": vpclatticeStringAny(svc, "arn"),
		"protocol":   "HTTP",
		"port":       int64(80),
		"status":     "ACTIVE",
	}
	s.listeners[id] = out
	return out
}

func (s *vpcLatticeStore) ensureRuleLocked(id, serviceID, listenerID string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "rule-00000000000000001")
	if out := s.rules[id]; out != nil {
		return out
	}
	s.ensureListenerLocked(listenerID, serviceID)
	out := map[string]any{
		"id":                 id,
		"arn":                vpclatticeARN("rule", id),
		"name":               fmt.Sprintf("rule-%s", strings.TrimPrefix(id, "rule-")),
		"serviceIdentifier":  serviceID,
		"listenerIdentifier": listenerID,
		"priority":           int64(10),
		"status":             "ACTIVE",
	}
	s.rules[id] = out
	return out
}

func (s *vpcLatticeStore) ensureServiceNetworkLocked(id string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "sn-00000000000000001")
	if out := s.serviceNetworks[id]; out != nil {
		return out
	}
	out := map[string]any{"id": id, "arn": vpclatticeARN("servicenetwork", id), "name": "stackyard-service-network", "status": "ACTIVE", "authType": "AWS_IAM"}
	s.serviceNetworks[id] = out
	return out
}

func (s *vpcLatticeStore) ensureTargetGroupLocked(id string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "tg-00000000000000001")
	if out := s.targetGroups[id]; out != nil {
		return out
	}
	out := map[string]any{"id": id, "arn": vpclatticeARN("targetgroup", id), "name": "stackyard-target-group", "type": "IP", "status": "ACTIVE", "protocol": "HTTP", "port": int64(80)}
	s.targetGroups[id] = out
	return out
}

func (s *vpcLatticeStore) ensureResourceConfigurationLocked(id string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "rcfg-00000000000000001")
	if out := s.resourceConfigurations[id]; out != nil {
		return out
	}
	out := map[string]any{"id": id, "arn": vpclatticeARN("resourceconfiguration", id), "name": "stackyard-resource-config", "status": "ACTIVE", "resourceConfigurationType": "SINGLE"}
	s.resourceConfigurations[id] = out
	return out
}

func (s *vpcLatticeStore) ensureResourceGatewayLocked(id string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "rgw-00000000000000001")
	if out := s.resourceGateways[id]; out != nil {
		return out
	}
	out := map[string]any{"id": id, "arn": vpclatticeARN("resourcegateway", id), "name": "stackyard-resource-gateway", "status": "ACTIVE", "vpcIdentifier": "vpc-00000000000000001"}
	s.resourceGateways[id] = out
	return out
}

func (s *vpcLatticeStore) ensureAccessLogSubscriptionLocked(id, resourceIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "als-00000000000000001")
	if out := s.accessLogSubscriptions[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                 id,
		"arn":                vpclatticeARN("accesslogsubscription", id),
		"resourceIdentifier": resourceIdentifier,
		"resourceArn":        vpclatticeARN("service", resourceIdentifier),
		"destinationArn":     "arn:aws:s3:::stackyard-logs",
	}
	s.accessLogSubscriptions[id] = out
	return out
}

func (s *vpcLatticeStore) ensureDomainVerificationLocked(id, resourceIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "dv-00000000000000001")
	if out := s.domainVerifications[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                 id,
		"arn":                vpclatticeARN("domainverification", id),
		"domainName":         "stackyard.example.com",
		"status":             "SUCCESSFUL",
		"resourceIdentifier": resourceIdentifier,
	}
	s.domainVerifications[id] = out
	return out
}

func (s *vpcLatticeStore) ensureServiceNetworkServiceAssociationLocked(id, serviceNetworkIdentifier, serviceIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "snsa-00000000000000001")
	if out := s.serviceNetworkServiceAssociations[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                       id,
		"arn":                      vpclatticeARN("servicenetworkserviceassociation", id),
		"serviceNetworkIdentifier": serviceNetworkIdentifier,
		"serviceNetworkArn":        vpclatticeARN("servicenetwork", serviceNetworkIdentifier),
		"serviceIdentifier":        serviceIdentifier,
		"serviceArn":               vpclatticeARN("service", serviceIdentifier),
		"status":                   "ACTIVE",
	}
	s.serviceNetworkServiceAssociations[id] = out
	return out
}

func (s *vpcLatticeStore) ensureServiceNetworkVpcAssociationLocked(id, serviceNetworkIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "snva-00000000000000001")
	if out := s.serviceNetworkVpcAssociations[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                       id,
		"arn":                      vpclatticeARN("servicenetworkvpcassociation", id),
		"serviceNetworkIdentifier": serviceNetworkIdentifier,
		"serviceNetworkArn":        vpclatticeARN("servicenetwork", serviceNetworkIdentifier),
		"vpcIdentifier":            "vpc-00000000000000001",
		"status":                   "ACTIVE",
	}
	s.serviceNetworkVpcAssociations[id] = out
	return out
}

func (s *vpcLatticeStore) ensureServiceNetworkResourceAssociationLocked(id, serviceNetworkIdentifier, resourceConfigurationIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "snra-00000000000000001")
	if out := s.serviceNetworkResourceAssociations[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                              id,
		"arn":                             vpclatticeARN("servicenetworkresourceassociation", id),
		"serviceNetworkIdentifier":        serviceNetworkIdentifier,
		"serviceNetworkArn":               vpclatticeARN("servicenetwork", serviceNetworkIdentifier),
		"resourceConfigurationIdentifier": resourceConfigurationIdentifier,
		"resourceConfigurationArn":        vpclatticeARN("resourceconfiguration", resourceConfigurationIdentifier),
		"status":                          "ACTIVE",
	}
	s.serviceNetworkResourceAssociations[id] = out
	return out
}

func (s *vpcLatticeStore) ensureResourceEndpointAssociationLocked(id, resourceConfigurationIdentifier, resourceGatewayIdentifier string) map[string]any {
	id = vpclatticeFirstNonEmpty(strings.TrimSpace(id), "rea-00000000000000001")
	if out := s.resourceEndpointAssociations[id]; out != nil {
		return out
	}
	out := map[string]any{
		"id":                              id,
		"arn":                             vpclatticeARN("resourceendpointassociation", id),
		"resourceConfigurationIdentifier": resourceConfigurationIdentifier,
		"resourceConfigurationArn":        vpclatticeARN("resourceconfiguration", resourceConfigurationIdentifier),
		"resourceGatewayIdentifier":       resourceGatewayIdentifier,
		"resourceGatewayArn":              vpclatticeARN("resourcegateway", resourceGatewayIdentifier),
		"status":                          "ACTIVE",
	}
	s.resourceEndpointAssociations[id] = out
	return out
}

func (s *vpcLatticeStore) applyMapPayload(dst map[string]any, payload map[string]any) {
	for k, v := range payload {
		dst[k] = vpclatticeCloneAny(v)
	}
}

func (s *vpcLatticeStore) listMaps(in map[string]map[string]any) []any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, vpclatticeCloneMap(in[key]))
	}
	return out
}

func (s *vpcLatticeStore) listByField(in map[string]map[string]any, field, value string) []any {
	items := s.listMaps(in)
	if value == "" {
		return items
	}
	return vpclatticeFilterMaps(items, field, value)
}

func vpclatticeFilterMaps(items []any, field, value string) []any {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(vpclatticeStringAny(m, field)) == value {
			out = append(out, vpclatticeCloneMap(m))
		}
	}
	return out
}

func vpclatticeARN(kind, id string) string {
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	if kind == "" {
		kind = "resource"
	}
	if id == "" {
		id = "unknown"
	}
	return fmt.Sprintf("arn:aws:vpc-lattice:us-east-1:123456789012:%s/%s", kind, id)
}

func vpclatticeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func vpclatticePathParam(from map[string]string, key string) string {
	if from == nil {
		return ""
	}
	return strings.TrimSpace(from[key])
}

func vpclatticeStringAny(from map[string]any, keys ...string) string {
	if from == nil {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		for k, value := range from {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
				switch v := value.(type) {
				case string:
					if strings.TrimSpace(v) != "" {
						return strings.TrimSpace(v)
					}
				case fmt.Stringer:
					if strings.TrimSpace(v.String()) != "" {
						return strings.TrimSpace(v.String())
					}
				}
			}
		}
	}
	return ""
}

func vpclatticeStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for k, val := range v {
			out[strings.TrimSpace(k)] = strings.TrimSpace(val)
		}
	case map[string]any:
		for k, val := range v {
			out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}
	return out
}

func vpclatticeStringList(value any) []string {
	out := []string{}
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	case []any:
		for _, item := range v {
			itemStr := strings.TrimSpace(fmt.Sprintf("%v", item))
			if itemStr != "" {
				out = append(out, itemStr)
			}
		}
	case string:
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func vpclatticeTargets(value any) []map[string]any {
	out := []map[string]any{}
	switch v := value.(type) {
	case []map[string]any:
		for _, item := range v {
			out = append(out, vpclatticeCloneMap(item))
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, vpclatticeCloneMap(m))
			}
		}
	}
	return out
}

func vpclatticeCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return vpclatticeCloneMap(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, vpclatticeCloneAny(item))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, vpclatticeCloneMap(item))
		}
		return out
	default:
		return v
	}
}

func vpclatticeCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = vpclatticeCloneAny(value)
	}
	return out
}

func vpclatticeCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
