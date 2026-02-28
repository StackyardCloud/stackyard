package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type DnsEntry struct {
	DnsName      string
	HostedZoneID string
}

type VpcEndpointConnection struct {
	CreationTimestamp       time.Time
	DnsEntries              []DnsEntry
	GatewayLoadBalancerARNs []string
	IPAddressType           string
	NetworkLoadBalancerARNs []string
	ServiceID               string
	Tags                    map[string]string
	VpcEndpointConnectionID string
	VpcEndpointID           string
	VpcEndpointOwner        string
	VpcEndpointRegion       string
	VpcEndpointState        string
}

type VpcEndpointAssociation struct {
	AssociatedResourceAccessibility string
	AssociatedResourceARN           string
	DnsEntry                        *DnsEntry
	FailureCode                     string
	FailureReason                   string
	ID                              string
	PrivateDnsEntry                 *DnsEntry
	ResourceConfigurationGroupARN   string
	ServiceNetworkARN               string
	ServiceNetworkName              string
	Tags                            map[string]string
	VpcEndpointID                   string
}

func (s *Service) AcceptVpcEndpointConnections(serviceID string, vpcEndpointIDs []string) ([]UnsuccessfulItem, error) {
	serviceID = strings.TrimSpace(serviceID)
	vpcEndpointIDs = dedupeTrimmedStrings(vpcEndpointIDs)
	if serviceID == "" || len(vpcEndpointIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.vpcEndpointServiceConfigurations[serviceID]
	if cfg == nil {
		return nil, ErrNotFound
	}

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, endpointID := range vpcEndpointIDs {
		endpoint := s.vpcEndpoints[endpointID]
		if endpoint == nil || endpoint.ServiceName != cfg.ServiceName {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: endpointID,
				Code:       "InvalidVpcEndpointId.NotFound",
				Message:    "vpc endpoint not found for service",
			})
			continue
		}
		endpoint.State = "Available"
	}
	return unsuccessful, nil
}

func (s *Service) RejectVpcEndpointConnections(serviceID string, vpcEndpointIDs []string) ([]UnsuccessfulItem, error) {
	serviceID = strings.TrimSpace(serviceID)
	vpcEndpointIDs = dedupeTrimmedStrings(vpcEndpointIDs)
	if serviceID == "" || len(vpcEndpointIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.vpcEndpointServiceConfigurations[serviceID]
	if cfg == nil {
		return nil, ErrNotFound
	}

	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, endpointID := range vpcEndpointIDs {
		endpoint := s.vpcEndpoints[endpointID]
		if endpoint == nil || endpoint.ServiceName != cfg.ServiceName {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: endpointID,
				Code:       "InvalidVpcEndpointId.NotFound",
				Message:    "vpc endpoint not found for service",
			})
			continue
		}
		endpoint.State = "Rejected"
	}
	return unsuccessful, nil
}

func (s *Service) DescribeVpcEndpoints(
	vpcEndpointIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpoint, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(vpcEndpointIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterEndpointIDSet := toStringSet(standardFilters["vpc-endpoint-id"])
	filterServiceNameSet := toStringSet(standardFilters["service-name"])
	filterServiceRegionSet := toLowerStringSet(standardFilters["service-region"])
	filterIPAddressTypeSet := toLowerStringSet(standardFilters["ip-address-type"])
	filterVpcIDSet := toStringSet(standardFilters["vpc-id"])
	filterStateSet := toLowerStringSet(standardFilters["vpc-endpoint-state"])
	filterTypeSet := toLowerStringSet(standardFilters["vpc-endpoint-type"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcEndpoint, 0, len(s.vpcEndpoints))
	for _, endpoint := range s.vpcEndpoints {
		if endpoint == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[endpoint.ID]; !ok {
				continue
			}
		}
		if len(filterEndpointIDSet) > 0 {
			if _, ok := filterEndpointIDSet[endpoint.ID]; !ok {
				continue
			}
		}
		if len(filterServiceNameSet) > 0 {
			if _, ok := filterServiceNameSet[endpoint.ServiceName]; !ok {
				continue
			}
		}
		if len(filterServiceRegionSet) > 0 {
			if _, ok := filterServiceRegionSet[strings.ToLower(endpoint.ServiceRegion)]; !ok {
				continue
			}
		}
		if len(filterIPAddressTypeSet) > 0 {
			if _, ok := filterIPAddressTypeSet[strings.ToLower(endpoint.IPAddressType)]; !ok {
				continue
			}
		}
		if len(filterVpcIDSet) > 0 {
			if _, ok := filterVpcIDSet[endpoint.VpcID]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(endpoint.State)]; !ok {
				continue
			}
		}
		if len(filterTypeSet) > 0 {
			if _, ok := filterTypeSet[strings.ToLower(endpoint.VpcEndpointType)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(endpoint.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVpcEndpoint(endpoint))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcEndpoint(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeVpcEndpointConnections(
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointConnection, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	filterEndpointIDSet := toStringSet(standardFilters["vpc-endpoint-id"])
	filterServiceIDSet := toStringSet(standardFilters["service-id"])
	filterOwnerSet := toStringSet(standardFilters["vpc-endpoint-owner"])
	filterRegionSet := toLowerStringSet(standardFilters["vpc-endpoint-region"])
	filterStateSet := toLowerStringSet(standardFilters["vpc-endpoint-state"])
	filterIPAddressTypeSet := toLowerStringSet(standardFilters["ip-address-type"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcEndpointConnection, 0, len(s.vpcEndpoints))
	for _, endpoint := range s.vpcEndpoints {
		if endpoint == nil {
			continue
		}
		serviceID, cfg := s.lookupVpcEndpointServiceConfigByNameLocked(endpoint.ServiceName)
		if cfg == nil {
			continue
		}
		conn := vpcEndpointConnectionFromEndpoint(endpoint, serviceID, cfg)
		if len(filterEndpointIDSet) > 0 {
			if _, ok := filterEndpointIDSet[conn.VpcEndpointID]; !ok {
				continue
			}
		}
		if len(filterServiceIDSet) > 0 {
			if _, ok := filterServiceIDSet[conn.ServiceID]; !ok {
				continue
			}
		}
		if len(filterOwnerSet) > 0 {
			if _, ok := filterOwnerSet[conn.VpcEndpointOwner]; !ok {
				continue
			}
		}
		if len(filterRegionSet) > 0 {
			if _, ok := filterRegionSet[strings.ToLower(conn.VpcEndpointRegion)]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(conn.VpcEndpointState)]; !ok {
				continue
			}
		}
		if len(filterIPAddressTypeSet) > 0 {
			if _, ok := filterIPAddressTypeSet[strings.ToLower(conn.IPAddressType)]; !ok {
				continue
			}
		}
		items = append(items, conn)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].VpcEndpointConnectionID < items[j].VpcEndpointConnectionID
	})
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcEndpointConnection(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeVpcEndpointAssociations(
	vpcEndpointIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointAssociation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(vpcEndpointIDs))
	standardFilters, _, _ := splitEC2Filters(filters)
	filterEndpointIDSet := toStringSet(standardFilters["vpc-endpoint-id"])
	filterAccessibilitySet := toLowerStringSet(standardFilters["associated-resource-accessibility"])
	filterAssociationIDSet := toStringSet(standardFilters["association-id"])
	filterAssociatedResourceIDSet := toStringSet(standardFilters["associated-resource-id"])
	filterServiceNetworkARNSet := toStringSet(standardFilters["service-network-arn"])
	filterResourceConfigGroupARNSet := toStringSet(standardFilters["resource-configuration-group-arn"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcEndpointAssociation, 0, len(s.vpcEndpoints))
	for _, endpoint := range s.vpcEndpoints {
		if endpoint == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[endpoint.ID]; !ok {
				continue
			}
		}
		association := vpcEndpointAssociationFromEndpoint(endpoint)
		if len(filterEndpointIDSet) > 0 {
			if _, ok := filterEndpointIDSet[association.VpcEndpointID]; !ok {
				continue
			}
		}
		if len(filterAccessibilitySet) > 0 && !matchesAssociationAccessibility(association.AssociatedResourceAccessibility, filterAccessibilitySet) {
			continue
		}
		if len(filterAssociationIDSet) > 0 {
			if _, ok := filterAssociationIDSet[association.ID]; !ok {
				continue
			}
		}
		if len(filterAssociatedResourceIDSet) > 0 {
			resourceID := vpcEndpointAssociatedResourceID(association.AssociatedResourceARN)
			if _, ok := filterAssociatedResourceIDSet[resourceID]; !ok {
				continue
			}
		}
		if len(filterServiceNetworkARNSet) > 0 {
			if _, ok := filterServiceNetworkARNSet[association.ServiceNetworkARN]; !ok {
				continue
			}
		}
		if len(filterResourceConfigGroupARNSet) > 0 {
			if _, ok := filterResourceConfigGroupARNSet[association.ResourceConfigurationGroupARN]; !ok {
				continue
			}
		}
		items = append(items, association)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcEndpointAssociation(nil), items[start:end]...), outputToken, nil
}

func cloneVpcEndpoint(in *VpcEndpoint) VpcEndpoint {
	out := *in
	out.RouteTableIDs = append([]string(nil), in.RouteTableIDs...)
	out.SecurityGroupIDs = append([]string(nil), in.SecurityGroupIDs...)
	out.SubnetIDs = append([]string(nil), in.SubnetIDs...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func (s *Service) lookupVpcEndpointServiceConfigByNameLocked(serviceName string) (string, *VpcEndpointServiceConfiguration) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return "", nil
	}
	var selectedID string
	var selected *VpcEndpointServiceConfiguration
	for serviceID, cfg := range s.vpcEndpointServiceConfigurations {
		if cfg == nil || cfg.ServiceName != serviceName {
			continue
		}
		if selected == nil || serviceID < selectedID {
			selectedID = serviceID
			selected = cfg
		}
	}
	return selectedID, selected
}

func vpcEndpointConnectionFromEndpoint(
	endpoint *VpcEndpoint,
	serviceID string,
	cfg *VpcEndpointServiceConfiguration,
) VpcEndpointConnection {
	connectionID := strings.Replace(endpoint.ID, "vpce-", "vpceconn-", 1)
	if connectionID == endpoint.ID {
		connectionID = "vpceconn-" + endpoint.ID
	}
	return VpcEndpointConnection{
		CreationTimestamp:       endpoint.CreationTimestamp,
		DnsEntries:              vpcEndpointDNSEntries(endpoint),
		GatewayLoadBalancerARNs: append([]string(nil), cfg.GatewayLoadBalancerARNs...),
		IPAddressType:           endpoint.IPAddressType,
		NetworkLoadBalancerARNs: append([]string(nil), cfg.NetworkLoadBalancerARNs...),
		ServiceID:               serviceID,
		Tags:                    cloneStringMap(endpoint.Tags),
		VpcEndpointConnectionID: connectionID,
		VpcEndpointID:           endpoint.ID,
		VpcEndpointOwner:        firstNonEmptyString(endpoint.OwnerID, DefaultAccountID),
		VpcEndpointRegion:       firstNonEmptyString(endpoint.ServiceRegion, DefaultRegion),
		VpcEndpointState:        endpoint.State,
	}
}

func vpcEndpointAssociationFromEndpoint(endpoint *VpcEndpoint) VpcEndpointAssociation {
	associationID := strings.Replace(endpoint.ID, "vpce-", "vpce-assoc-", 1)
	if associationID == endpoint.ID {
		associationID = "vpce-assoc-" + endpoint.ID
	}

	associatedResourceARN := firstNonEmptyString(
		endpoint.ResourceConfigARN,
		endpoint.ServiceNetworkARN,
		fmt.Sprintf("arn:aws:ec2:%s:%s:vpc-endpoint/%s", DefaultRegion, DefaultAccountID, endpoint.ID),
	)

	accessibility := "PENDING"
	if strings.EqualFold(endpoint.State, "available") {
		accessibility = "AVAILABLE"
	}

	entry := vpcEndpointDNSEntries(endpoint)
	var dnsEntry *DnsEntry
	var privateDnsEntry *DnsEntry
	if len(entry) > 0 {
		e := entry[0]
		dnsEntry = &e
		if endpoint.PrivateDNSEnabled {
			p := entry[0]
			privateDnsEntry = &p
		}
	}

	return VpcEndpointAssociation{
		AssociatedResourceAccessibility: accessibility,
		AssociatedResourceARN:           associatedResourceARN,
		DnsEntry:                        dnsEntry,
		ID:                              associationID,
		PrivateDnsEntry:                 privateDnsEntry,
		ResourceConfigurationGroupARN:   endpoint.ResourceConfigARN,
		ServiceNetworkARN:               endpoint.ServiceNetworkARN,
		ServiceNetworkName:              vpcEndpointServiceNetworkName(endpoint.ServiceNetworkARN),
		Tags:                            cloneStringMap(endpoint.Tags),
		VpcEndpointID:                   endpoint.ID,
	}
}

func vpcEndpointDNSEntries(endpoint *VpcEndpoint) []DnsEntry {
	region := firstNonEmptyString(endpoint.ServiceRegion, DefaultRegion)
	return []DnsEntry{
		{
			DnsName:      fmt.Sprintf("%s.%s.vpce.stackyard.local", endpoint.ID, region),
			HostedZoneID: "ZVPCE000000",
		},
	}
}

func vpcEndpointAssociatedResourceID(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	slash := strings.LastIndex(arn, "/")
	colon := strings.LastIndex(arn, ":")
	if slash > colon {
		return arn[slash+1:]
	}
	return arn[colon+1:]
}

func vpcEndpointServiceNetworkName(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	slash := strings.LastIndex(arn, "/")
	if slash >= 0 && slash < len(arn)-1 {
		return arn[slash+1:]
	}
	colon := strings.LastIndex(arn, ":")
	if colon >= 0 && colon < len(arn)-1 {
		return arn[colon+1:]
	}
	return arn
}

func matchesAssociationAccessibility(current string, filters map[string]struct{}) bool {
	if len(filters) == 0 {
		return true
	}
	currentLower := strings.ToLower(strings.TrimSpace(current))
	for value := range filters {
		switch value {
		case "accessible":
			if currentLower == "available" {
				return true
			}
		case "inaccessible":
			if currentLower != "available" {
				return true
			}
		default:
			if currentLower == value {
				return true
			}
		}
	}
	return false
}
