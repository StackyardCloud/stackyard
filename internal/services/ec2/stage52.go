package ec2

import (
	"strings"
	"time"
)

type VpcEndpoint struct {
	ID                string
	VpcID             string
	ServiceName       string
	ServiceRegion     string
	State             string
	OwnerID           string
	VpcEndpointType   string
	RouteTableIDs     []string
	SecurityGroupIDs  []string
	SubnetIDs         []string
	PolicyDocument    string
	PrivateDNSEnabled bool
	IPAddressType     string
	CreationTimestamp time.Time
	ResourceConfigARN string
	ServiceNetworkARN string
	Tags              map[string]string
}

func (s *Service) ModifyVpcEndpoint(
	vpcEndpointID string,
	addRouteTableIDs []string,
	addSecurityGroupIDs []string,
	addSubnetIDs []string,
	ipAddressType string,
	policyDocument *string,
	privateDNSEnabled *bool,
	removeRouteTableIDs []string,
	removeSecurityGroupIDs []string,
	removeSubnetIDs []string,
	resetPolicy *bool,
	subnetConfigurationSubnetIDs []string,
) (bool, error) {
	vpcEndpointID = strings.TrimSpace(vpcEndpointID)
	if vpcEndpointID == "" {
		return false, ErrInvalidParameter
	}
	if policyDocument != nil && strings.TrimSpace(*policyDocument) == "" {
		return false, ErrInvalidParameter
	}
	if policyDocument != nil && resetPolicy != nil && *resetPolicy {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.vpcEndpoints[vpcEndpointID]
	if endpoint == nil {
		return false, ErrNotFound
	}

	endpoint.RouteTableIDs = addToEC2StringSet(endpoint.RouteTableIDs, addRouteTableIDs)
	endpoint.RouteTableIDs = removeFromEC2StringSet(endpoint.RouteTableIDs, removeRouteTableIDs)
	endpoint.SecurityGroupIDs = addToEC2StringSet(endpoint.SecurityGroupIDs, addSecurityGroupIDs)
	endpoint.SecurityGroupIDs = removeFromEC2StringSet(endpoint.SecurityGroupIDs, removeSecurityGroupIDs)
	endpoint.SubnetIDs = addToEC2StringSet(endpoint.SubnetIDs, addSubnetIDs)
	endpoint.SubnetIDs = addToEC2StringSet(endpoint.SubnetIDs, subnetConfigurationSubnetIDs)
	endpoint.SubnetIDs = removeFromEC2StringSet(endpoint.SubnetIDs, removeSubnetIDs)

	if policyDocument != nil {
		endpoint.PolicyDocument = strings.TrimSpace(*policyDocument)
	}
	if resetPolicy != nil && *resetPolicy {
		endpoint.PolicyDocument = `{"Version":"2012-10-17","Statement":[]}`
	}
	if privateDNSEnabled != nil {
		endpoint.PrivateDNSEnabled = *privateDNSEnabled
	}
	if trimmedIPAddressType := strings.TrimSpace(ipAddressType); trimmedIPAddressType != "" {
		endpoint.IPAddressType = trimmedIPAddressType
	}

	return true, nil
}
