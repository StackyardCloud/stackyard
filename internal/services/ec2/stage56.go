package ec2

import (
	"strings"
	"time"
)

func (s *Service) CreateVpcEndpoint(
	vpcID, serviceName, serviceRegion, vpcEndpointType, ipAddressType string,
	routeTableIDs, securityGroupIDs, subnetIDs, subnetConfigurationSubnetIDs []string,
	policyDocument *string,
	privateDNSEnabled *bool,
	resourceConfigurationARN, serviceNetworkARN *string,
	tags []Tag,
	clientToken *string,
) (VpcEndpoint, *string, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return VpcEndpoint{}, nil, ErrInvalidParameter
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return VpcEndpoint{}, nil, ErrInvalidParameter
	}
	serviceRegion = strings.TrimSpace(serviceRegion)
	if serviceRegion == "" {
		serviceRegion = DefaultRegion
	}
	vpcEndpointType = strings.TrimSpace(vpcEndpointType)
	if vpcEndpointType == "" {
		vpcEndpointType = "Gateway"
	}
	ipAddressType = strings.TrimSpace(ipAddressType)
	if ipAddressType == "" {
		ipAddressType = "ipv4"
	}

	policy := `{"Version":"2012-10-17","Statement":[]}`
	if policyDocument != nil {
		policy = strings.TrimSpace(*policyDocument)
		if policy == "" {
			return VpcEndpoint{}, nil, ErrInvalidParameter
		}
	}
	var privateDNS bool
	if privateDNSEnabled != nil {
		privateDNS = *privateDNSEnabled
	}

	routeTableIDs = dedupeTrimmedStrings(routeTableIDs)
	securityGroupIDs = dedupeTrimmedStrings(securityGroupIDs)
	subnetIDs = dedupeTrimmedStrings(subnetIDs)
	subnetConfigurationSubnetIDs = dedupeTrimmedStrings(subnetConfigurationSubnetIDs)

	var effectiveClientToken *string
	if clientToken != nil {
		token := strings.TrimSpace(*clientToken)
		if token != "" {
			effectiveClientToken = &token
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[vpcID]; !ok {
		return VpcEndpoint{}, nil, ErrNotFound
	}
	for _, routeTableID := range routeTableIDs {
		if _, ok := s.routeTables[routeTableID]; !ok {
			return VpcEndpoint{}, nil, ErrNotFound
		}
	}
	for _, securityGroupID := range securityGroupIDs {
		if _, ok := s.securityGroups[securityGroupID]; !ok {
			return VpcEndpoint{}, nil, ErrNotFound
		}
	}
	for _, subnetID := range subnetIDs {
		if _, ok := s.subnets[subnetID]; !ok {
			return VpcEndpoint{}, nil, ErrNotFound
		}
	}
	for _, subnetID := range subnetConfigurationSubnetIDs {
		if _, ok := s.subnets[subnetID]; !ok {
			return VpcEndpoint{}, nil, ErrNotFound
		}
	}

	combinedSubnets := append([]string(nil), subnetIDs...)
	combinedSubnets = addToEC2StringSet(combinedSubnets, subnetConfigurationSubnetIDs)
	if strings.EqualFold(vpcEndpointType, "Gateway") && len(routeTableIDs) == 0 {
		routeTableIDs = []string{defaultRouteTableID}
	}

	endpointID := s.nextIDLocked("vpce")
	endpoint := VpcEndpoint{
		ID:                endpointID,
		VpcID:             vpcID,
		ServiceName:       serviceName,
		ServiceRegion:     serviceRegion,
		State:             "Available",
		OwnerID:           DefaultAccountID,
		VpcEndpointType:   vpcEndpointType,
		RouteTableIDs:     routeTableIDs,
		SecurityGroupIDs:  securityGroupIDs,
		SubnetIDs:         combinedSubnets,
		PolicyDocument:    policy,
		PrivateDNSEnabled: privateDNS,
		IPAddressType:     ipAddressType,
		CreationTimestamp: time.Now().UTC(),
		ResourceConfigARN: derefTrimmedString(resourceConfigurationARN),
		ServiceNetworkARN: derefTrimmedString(serviceNetworkARN),
		Tags:              tagsToMap(tags),
	}
	s.vpcEndpoints[endpointID] = &endpoint
	return endpoint, effectiveClientToken, nil
}

func derefTrimmedString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
