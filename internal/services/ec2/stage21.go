package ec2

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type ClientVpnEndpointStatus struct {
	Code    string
	Message string
}

type ClientVpnRouteStatus struct {
	Code    string
	Message string
}

type ClientVpnAssociationStatus struct {
	Code    string
	Message string
}

type ClientVpnAuthorizationRuleStatus struct {
	Code    string
	Message string
}

type ClientVpnConnectionStatus struct {
	Code    string
	Message string
}

type ClientCertificateRevocationListStatus struct {
	Code    string
	Message string
}

type ClientVpnEndpoint struct {
	ID                         string
	ClientCidrBlock            string
	Description                string
	DisconnectOnSessionTimeout bool
	DnsName                    string
	DnsServers                 []string
	SecurityGroupIDs           []string
	SelfServicePortal          string
	SelfServicePortalURL       string
	ServerCertificateARN       string
	SessionTimeoutHours        int32
	SplitTunnel                bool
	Status                     ClientVpnEndpointStatus
	Tags                       map[string]string
	TransportProtocol          string
	VpcID                      string
	VpnPort                    int32
	CreationTime               time.Time
	DeletionTime               *time.Time
}

type ClientVpnRoute struct {
	ClientVpnEndpointID string
	Description         string
	DestinationCIDR     string
	Origin              string
	Status              ClientVpnRouteStatus
	TargetSubnet        string
	Type                string
}

type ClientVpnTargetNetwork struct {
	AssociationID       string
	ClientVpnEndpointID string
	SecurityGroupIDs    []string
	Status              ClientVpnAssociationStatus
	TargetNetworkID     string
	VpcID               string
}

type ClientVpnAuthorizationRule struct {
	AuthorizeAllGroups  bool
	ClientVpnEndpointID string
	Description         string
	DestinationCIDR     string
	GroupID             string
	Status              ClientVpnAuthorizationRuleStatus
}

type ClientVpnConnection struct {
	ClientIP                  string
	ClientVpnEndpointID       string
	CommonName                string
	ConnectionEndTime         *time.Time
	ConnectionEstablishedTime time.Time
	ConnectionID              string
	EgressBytes               string
	EgressPackets             string
	IngressBytes              string
	IngressPackets            string
	PostureComplianceStatuses []string
	Status                    ClientVpnConnectionStatus
	Timestamp                 time.Time
	Username                  string
}

type TerminateClientVpnConnectionStatus struct {
	ConnectionID   string
	CurrentStatus  ClientVpnConnectionStatus
	PreviousStatus ClientVpnConnectionStatus
}

type TerminateClientVpnConnectionsResult struct {
	ClientVpnEndpointID string
	ConnectionStatuses  []TerminateClientVpnConnectionStatus
	Username            string
}

func (s *Service) CreateClientVpnEndpoint(
	authenticationType, clientCidrBlock, serverCertificateARN, description string,
	transportProtocol string,
	vpnPort *int32,
	splitTunnel *bool,
	securityGroupIDs []string,
	vpcID string,
	dnsServers []string,
	sessionTimeoutHours *int32,
	disconnectOnSessionTimeout *bool,
	selfServicePortal string,
	tags []Tag,
) (ClientVpnEndpoint, error) {
	authenticationType = strings.TrimSpace(authenticationType)
	clientCidrBlock = strings.TrimSpace(clientCidrBlock)
	serverCertificateARN = strings.TrimSpace(serverCertificateARN)
	description = strings.TrimSpace(description)
	if authenticationType == "" || clientCidrBlock == "" || serverCertificateARN == "" {
		return ClientVpnEndpoint{}, ErrInvalidParameter
	}

	transportProtocol = strings.ToLower(strings.TrimSpace(transportProtocol))
	if transportProtocol == "" {
		transportProtocol = "udp"
	}
	if transportProtocol != "udp" && transportProtocol != "tcp" {
		return ClientVpnEndpoint{}, ErrInvalidParameter
	}

	resolvedVpnPort := int32(443)
	if vpnPort != nil {
		resolvedVpnPort = *vpnPort
	}
	if resolvedVpnPort != 443 && resolvedVpnPort != 1194 {
		return ClientVpnEndpoint{}, ErrInvalidParameter
	}

	resolvedSessionTimeout := int32(24)
	if sessionTimeoutHours != nil {
		resolvedSessionTimeout = *sessionTimeoutHours
	}
	if !isValidClientVpnSessionTimeoutHours(resolvedSessionTimeout) {
		return ClientVpnEndpoint{}, ErrInvalidParameter
	}

	resolvedSplitTunnel := false
	if splitTunnel != nil {
		resolvedSplitTunnel = *splitTunnel
	}

	resolvedDisconnectOnSessionTimeout := true
	if disconnectOnSessionTimeout != nil {
		resolvedDisconnectOnSessionTimeout = *disconnectOnSessionTimeout
	}

	resolvedSelfServicePortal, err := normalizeClientVpnSelfServicePortal(selfServicePortal)
	if err != nil {
		return ClientVpnEndpoint{}, err
	}

	dnsServers = normalizeClientVpnStringList(dnsServers)

	s.mu.Lock()
	defer s.mu.Unlock()

	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		vpcID = defaultVPCID
	}
	if s.vpcs[vpcID] == nil {
		return ClientVpnEndpoint{}, ErrNotFound
	}

	resolvedSecurityGroupIDs, err := s.resolveClientVpnSecurityGroupsLocked(vpcID, securityGroupIDs)
	if err != nil {
		return ClientVpnEndpoint{}, err
	}

	endpointID := s.nextIDLocked("cvpn-endpoint")
	now := time.Now().UTC()
	endpoint := &ClientVpnEndpoint{
		ID:                         endpointID,
		ClientCidrBlock:            clientCidrBlock,
		Description:                description,
		DisconnectOnSessionTimeout: resolvedDisconnectOnSessionTimeout,
		DnsName:                    endpointID + ".prod.clientvpn." + DefaultRegion + ".amazonaws.com",
		DnsServers:                 append([]string(nil), dnsServers...),
		SecurityGroupIDs:           append([]string(nil), resolvedSecurityGroupIDs...),
		SelfServicePortal:          resolvedSelfServicePortal,
		ServerCertificateARN:       serverCertificateARN,
		SessionTimeoutHours:        resolvedSessionTimeout,
		SplitTunnel:                resolvedSplitTunnel,
		Status: ClientVpnEndpointStatus{
			Code:    "pending-associate",
			Message: "pending target network association",
		},
		Tags:              tagsToMap(tags),
		TransportProtocol: transportProtocol,
		VpcID:             vpcID,
		VpnPort:           resolvedVpnPort,
		CreationTime:      now,
	}
	if endpoint.SelfServicePortal == "enabled" {
		endpoint.SelfServicePortalURL = "https://self-service.clientvpn.amazonaws.com/endpoints/" + endpoint.ID
	}

	s.clientVpnEndpoints[endpoint.ID] = endpoint
	s.clientVpnRoutes[endpoint.ID] = map[string]*ClientVpnRoute{}
	s.clientVpnTargetNetworks[endpoint.ID] = map[string]*ClientVpnTargetNetwork{}
	s.clientVpnAuthorizationRules[endpoint.ID] = map[string]*ClientVpnAuthorizationRule{}
	s.clientVpnConnections[endpoint.ID] = map[string]*ClientVpnConnection{}
	s.clientVpnCertificateRevocationLists[endpoint.ID] = defaultClientVpnCertificateRevocationList()
	s.clientVpnCertificateRevocationListStatus[endpoint.ID] = ClientCertificateRevocationListStatus{Code: "active", Message: "active"}

	seedConnectionID := s.nextIDLocked("cvpn-conn")
	s.clientVpnConnections[endpoint.ID][seedConnectionID] = &ClientVpnConnection{
		ClientIP:                  "172.16.0.10",
		ClientVpnEndpointID:       endpoint.ID,
		CommonName:                "stackyard-client",
		ConnectionEstablishedTime: now,
		ConnectionID:              seedConnectionID,
		EgressBytes:               "0",
		EgressPackets:             "0",
		IngressBytes:              "0",
		IngressPackets:            "0",
		PostureComplianceStatuses: []string{},
		Status:                    ClientVpnConnectionStatus{Code: "active", Message: "active"},
		Timestamp:                 now,
		Username:                  "stackyard",
	}

	return cloneClientVpnEndpoint(endpoint), nil
}

func (s *Service) DescribeClientVpnEndpoints(endpointIDs, filterEndpointIDs, transportProtocols []string) []ClientVpnEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(endpointIDs)
	filterIDSet := toStringSet(filterEndpointIDs)
	transportSet := map[string]struct{}{}
	for _, protocol := range transportProtocols {
		protocol = strings.ToLower(strings.TrimSpace(protocol))
		if protocol == "" {
			continue
		}
		transportSet[protocol] = struct{}{}
	}

	out := make([]ClientVpnEndpoint, 0, len(s.clientVpnEndpoints))
	for _, endpoint := range s.clientVpnEndpoints {
		if len(idSet) > 0 {
			if _, ok := idSet[endpoint.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[endpoint.ID]; !ok {
				continue
			}
		}
		if len(transportSet) > 0 {
			if _, ok := transportSet[strings.ToLower(endpoint.TransportProtocol)]; !ok {
				continue
			}
		}
		out = append(out, cloneClientVpnEndpoint(endpoint))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) ModifyClientVpnEndpoint(
	clientVpnEndpointID string,
	description *string,
	splitTunnel *bool,
	securityGroupIDs []string,
	hasSecurityGroupIDs bool,
	dnsServers []string,
	hasDnsServers bool,
	sessionTimeoutHours *int32,
	disconnectOnSessionTimeout *bool,
	selfServicePortal *string,
	vpcID *string,
	vpnPort *int32,
) (bool, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.clientVpnEndpoints[clientVpnEndpointID]
	if endpoint == nil {
		return false, ErrNotFound
	}
	if endpoint.Status.Code == "deleted" {
		return false, ErrConflict
	}

	if vpcID != nil {
		normalizedVpcID := strings.TrimSpace(*vpcID)
		if normalizedVpcID == "" {
			return false, ErrInvalidParameter
		}
		if s.vpcs[normalizedVpcID] == nil {
			return false, ErrNotFound
		}
		endpoint.VpcID = normalizedVpcID
	}

	if hasSecurityGroupIDs {
		resolvedSecurityGroupIDs, err := s.resolveClientVpnSecurityGroupsLocked(endpoint.VpcID, securityGroupIDs)
		if err != nil {
			return false, err
		}
		endpoint.SecurityGroupIDs = append([]string(nil), resolvedSecurityGroupIDs...)
		for _, targetNetwork := range s.clientVpnTargetNetworks[endpoint.ID] {
			if targetNetwork.Status.Code != "associated" {
				continue
			}
			targetNetwork.SecurityGroupIDs = append([]string(nil), endpoint.SecurityGroupIDs...)
		}
	}

	if hasDnsServers {
		endpoint.DnsServers = normalizeClientVpnStringList(dnsServers)
	}

	if description != nil {
		endpoint.Description = strings.TrimSpace(*description)
	}
	if splitTunnel != nil {
		endpoint.SplitTunnel = *splitTunnel
	}
	if sessionTimeoutHours != nil {
		if !isValidClientVpnSessionTimeoutHours(*sessionTimeoutHours) {
			return false, ErrInvalidParameter
		}
		endpoint.SessionTimeoutHours = *sessionTimeoutHours
	}
	if disconnectOnSessionTimeout != nil {
		endpoint.DisconnectOnSessionTimeout = *disconnectOnSessionTimeout
	}
	if selfServicePortal != nil {
		normalizedSelfServicePortal, err := normalizeClientVpnSelfServicePortal(*selfServicePortal)
		if err != nil {
			return false, err
		}
		endpoint.SelfServicePortal = normalizedSelfServicePortal
		endpoint.SelfServicePortalURL = ""
		if endpoint.SelfServicePortal == "enabled" {
			endpoint.SelfServicePortalURL = "https://self-service.clientvpn.amazonaws.com/endpoints/" + endpoint.ID
		}
	}
	if vpnPort != nil {
		if *vpnPort != 443 && *vpnPort != 1194 {
			return false, ErrInvalidParameter
		}
		endpoint.VpnPort = *vpnPort
	}

	return true, nil
}

func (s *Service) DeleteClientVpnEndpoint(clientVpnEndpointID string) (ClientVpnEndpointStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return ClientVpnEndpointStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.clientVpnEndpoints[clientVpnEndpointID]
	if endpoint == nil {
		return ClientVpnEndpointStatus{}, ErrNotFound
	}
	if endpoint.Status.Code == "deleted" {
		return endpoint.Status, nil
	}

	now := time.Now().UTC()
	endpoint.Status = ClientVpnEndpointStatus{Code: "deleted", Message: "deleted"}
	endpoint.DeletionTime = &now

	for _, route := range s.clientVpnRoutes[endpoint.ID] {
		route.Status = ClientVpnRouteStatus{Code: "deleting", Message: "deleting"}
	}
	for _, targetNetwork := range s.clientVpnTargetNetworks[endpoint.ID] {
		targetNetwork.Status = ClientVpnAssociationStatus{Code: "disassociated", Message: "disassociated"}
	}
	for _, connection := range s.clientVpnConnections[endpoint.ID] {
		if connection.Status.Code == "terminated" {
			continue
		}
		connection.Status = ClientVpnConnectionStatus{Code: "terminated", Message: "terminated"}
		connection.ConnectionEndTime = &now
		connection.Timestamp = now
	}

	return endpoint.Status, nil
}

func (s *Service) CreateClientVpnRoute(clientVpnEndpointID, destinationCIDR, targetVpcSubnetID, description string) (ClientVpnRouteStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	targetVpcSubnetID = strings.TrimSpace(targetVpcSubnetID)
	description = strings.TrimSpace(description)
	if clientVpnEndpointID == "" || destinationCIDR == "" || targetVpcSubnetID == "" {
		return ClientVpnRouteStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnRouteStatus{}, err
	}
	if endpoint == nil {
		return ClientVpnRouteStatus{}, ErrNotFound
	}

	if !strings.EqualFold(targetVpcSubnetID, "local") && !s.hasAssociatedClientVpnTargetSubnetLocked(clientVpnEndpointID, targetVpcSubnetID) {
		return ClientVpnRouteStatus{}, ErrNotFound
	}

	routes := s.clientVpnRoutes[endpoint.ID]
	if routes == nil {
		routes = map[string]*ClientVpnRoute{}
		s.clientVpnRoutes[endpoint.ID] = routes
	}

	routeKey := clientVpnRouteKey(destinationCIDR, targetVpcSubnetID)
	route := routes[routeKey]
	if route == nil {
		route = &ClientVpnRoute{
			ClientVpnEndpointID: endpoint.ID,
			DestinationCIDR:     destinationCIDR,
			Origin:              "add-route",
			TargetSubnet:        targetVpcSubnetID,
			Type:                "nat",
		}
		routes[routeKey] = route
	}
	route.Description = description
	route.Status = ClientVpnRouteStatus{Code: "active", Message: "active"}
	return route.Status, nil
}

func (s *Service) DescribeClientVpnRoutes(clientVpnEndpointID string, destinationCIDRs, origins, targetSubnets []string) ([]ClientVpnRoute, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clientVpnEndpoints[clientVpnEndpointID] == nil {
		return nil, ErrNotFound
	}

	destinationSet := toStringSet(destinationCIDRs)
	originSet := map[string]struct{}{}
	for _, origin := range origins {
		origin = strings.ToLower(strings.TrimSpace(origin))
		if origin == "" {
			continue
		}
		originSet[origin] = struct{}{}
	}
	targetSet := toStringSet(targetSubnets)

	routesByKey := s.clientVpnRoutes[clientVpnEndpointID]
	out := make([]ClientVpnRoute, 0, len(routesByKey))
	for _, route := range routesByKey {
		if len(destinationSet) > 0 {
			if _, ok := destinationSet[route.DestinationCIDR]; !ok {
				continue
			}
		}
		if len(originSet) > 0 {
			if _, ok := originSet[strings.ToLower(route.Origin)]; !ok {
				continue
			}
		}
		if len(targetSet) > 0 {
			if _, ok := targetSet[route.TargetSubnet]; !ok {
				continue
			}
		}
		out = append(out, cloneClientVpnRoute(route))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DestinationCIDR == out[j].DestinationCIDR {
			return out[i].TargetSubnet < out[j].TargetSubnet
		}
		return out[i].DestinationCIDR < out[j].DestinationCIDR
	})
	return out, nil
}

func (s *Service) DeleteClientVpnRoute(clientVpnEndpointID, destinationCIDR, targetVpcSubnetID string) (ClientVpnRouteStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	targetVpcSubnetID = strings.TrimSpace(targetVpcSubnetID)
	if clientVpnEndpointID == "" || destinationCIDR == "" {
		return ClientVpnRouteStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnRouteStatus{}, err
	}
	if endpoint == nil {
		return ClientVpnRouteStatus{}, ErrNotFound
	}

	routes := s.clientVpnRoutes[endpoint.ID]
	if len(routes) == 0 {
		return ClientVpnRouteStatus{}, ErrNotFound
	}

	removeKey := ""
	if targetVpcSubnetID != "" {
		candidateKey := clientVpnRouteKey(destinationCIDR, targetVpcSubnetID)
		if routes[candidateKey] == nil {
			return ClientVpnRouteStatus{}, ErrNotFound
		}
		removeKey = candidateKey
	} else {
		for key, route := range routes {
			if route.DestinationCIDR == destinationCIDR {
				removeKey = key
				break
			}
		}
		if removeKey == "" {
			return ClientVpnRouteStatus{}, ErrNotFound
		}
	}

	delete(routes, removeKey)
	return ClientVpnRouteStatus{Code: "deleting", Message: "deleting"}, nil
}

func (s *Service) AssociateClientVpnTargetNetwork(clientVpnEndpointID, subnetID string) (ClientVpnTargetNetwork, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	subnetID = strings.TrimSpace(subnetID)
	if clientVpnEndpointID == "" || subnetID == "" {
		return ClientVpnTargetNetwork{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnTargetNetwork{}, err
	}
	if endpoint == nil {
		return ClientVpnTargetNetwork{}, ErrNotFound
	}

	subnet := s.subnets[subnetID]
	if subnet == nil {
		return ClientVpnTargetNetwork{}, ErrNotFound
	}

	targetNetworks := s.clientVpnTargetNetworks[endpoint.ID]
	if targetNetworks == nil {
		targetNetworks = map[string]*ClientVpnTargetNetwork{}
		s.clientVpnTargetNetworks[endpoint.ID] = targetNetworks
	}

	for _, targetNetwork := range targetNetworks {
		if targetNetwork.TargetNetworkID == subnetID && targetNetwork.Status.Code == "associated" {
			return cloneClientVpnTargetNetwork(targetNetwork), nil
		}
	}

	associationID := s.nextIDLocked("cvpn-assoc")
	targetNetwork := &ClientVpnTargetNetwork{
		AssociationID:       associationID,
		ClientVpnEndpointID: endpoint.ID,
		SecurityGroupIDs:    append([]string(nil), endpoint.SecurityGroupIDs...),
		Status:              ClientVpnAssociationStatus{Code: "associated", Message: "associated"},
		TargetNetworkID:     subnetID,
		VpcID:               subnet.VpcID,
	}
	targetNetworks[associationID] = targetNetwork

	endpoint.Status = ClientVpnEndpointStatus{Code: "available", Message: "available"}

	routes := s.clientVpnRoutes[endpoint.ID]
	if routes == nil {
		routes = map[string]*ClientVpnRoute{}
		s.clientVpnRoutes[endpoint.ID] = routes
	}
	associateRouteKey := clientVpnRouteKey(subnet.CidrBlock, subnetID)
	routes[associateRouteKey] = &ClientVpnRoute{
		ClientVpnEndpointID: endpoint.ID,
		Description:         "",
		DestinationCIDR:     subnet.CidrBlock,
		Origin:              "associate",
		Status:              ClientVpnRouteStatus{Code: "active", Message: "active"},
		TargetSubnet:        subnetID,
		Type:                "nat",
	}

	return cloneClientVpnTargetNetwork(targetNetwork), nil
}

func (s *Service) DisassociateClientVpnTargetNetwork(clientVpnEndpointID, associationID string) (ClientVpnTargetNetwork, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	associationID = strings.TrimSpace(associationID)
	if clientVpnEndpointID == "" || associationID == "" {
		return ClientVpnTargetNetwork{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnTargetNetwork{}, err
	}
	if endpoint == nil {
		return ClientVpnTargetNetwork{}, ErrNotFound
	}

	targetNetwork := s.clientVpnTargetNetworks[endpoint.ID][associationID]
	if targetNetwork == nil {
		return ClientVpnTargetNetwork{}, ErrNotFound
	}
	if targetNetwork.Status.Code != "disassociated" {
		targetNetwork.Status = ClientVpnAssociationStatus{Code: "disassociated", Message: "disassociated"}
	}

	routes := s.clientVpnRoutes[endpoint.ID]
	for key, route := range routes {
		if route.Origin == "associate" && route.TargetSubnet == targetNetwork.TargetNetworkID {
			delete(routes, key)
		}
	}

	if !s.hasAssociatedClientVpnTargetNetworkLocked(endpoint.ID) {
		endpoint.Status = ClientVpnEndpointStatus{Code: "pending-associate", Message: "pending target network association"}
	}

	return cloneClientVpnTargetNetwork(targetNetwork), nil
}

func (s *Service) ApplySecurityGroupsToClientVpnTargetNetwork(clientVpnEndpointID, vpcID string, securityGroupIDs []string) ([]string, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	vpcID = strings.TrimSpace(vpcID)
	if clientVpnEndpointID == "" || vpcID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		return nil, ErrNotFound
	}
	if s.vpcs[vpcID] == nil {
		return nil, ErrNotFound
	}

	resolvedSecurityGroupIDs, err := s.resolveClientVpnSecurityGroupsLocked(vpcID, securityGroupIDs)
	if err != nil {
		return nil, err
	}
	endpoint.SecurityGroupIDs = append([]string(nil), resolvedSecurityGroupIDs...)

	for _, targetNetwork := range s.clientVpnTargetNetworks[endpoint.ID] {
		if targetNetwork.VpcID != vpcID || targetNetwork.Status.Code != "associated" {
			continue
		}
		targetNetwork.SecurityGroupIDs = append([]string(nil), resolvedSecurityGroupIDs...)
	}

	return append([]string(nil), resolvedSecurityGroupIDs...), nil
}

func (s *Service) AuthorizeClientVpnIngress(clientVpnEndpointID, targetNetworkCIDR, accessGroupID string, authorizeAllGroups bool, description string) (ClientVpnAuthorizationRuleStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	targetNetworkCIDR = strings.TrimSpace(targetNetworkCIDR)
	accessGroupID = strings.TrimSpace(accessGroupID)
	description = strings.TrimSpace(description)
	if clientVpnEndpointID == "" || targetNetworkCIDR == "" {
		return ClientVpnAuthorizationRuleStatus{}, ErrInvalidParameter
	}
	if !authorizeAllGroups && accessGroupID == "" {
		return ClientVpnAuthorizationRuleStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnAuthorizationRuleStatus{}, err
	}
	if endpoint == nil {
		return ClientVpnAuthorizationRuleStatus{}, ErrNotFound
	}

	rules := s.clientVpnAuthorizationRules[endpoint.ID]
	if rules == nil {
		rules = map[string]*ClientVpnAuthorizationRule{}
		s.clientVpnAuthorizationRules[endpoint.ID] = rules
	}

	ruleKey := clientVpnAuthorizationRuleKey(targetNetworkCIDR, accessGroupID, authorizeAllGroups)
	rule := rules[ruleKey]
	if rule == nil {
		rule = &ClientVpnAuthorizationRule{
			AuthorizeAllGroups:  authorizeAllGroups,
			ClientVpnEndpointID: endpoint.ID,
			DestinationCIDR:     targetNetworkCIDR,
			GroupID:             accessGroupID,
		}
		rules[ruleKey] = rule
	}
	rule.Description = description
	rule.Status = ClientVpnAuthorizationRuleStatus{Code: "active", Message: "active"}
	return rule.Status, nil
}

func (s *Service) RevokeClientVpnIngress(clientVpnEndpointID, targetNetworkCIDR, accessGroupID string, revokeAllGroups bool) (ClientVpnAuthorizationRuleStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	targetNetworkCIDR = strings.TrimSpace(targetNetworkCIDR)
	accessGroupID = strings.TrimSpace(accessGroupID)
	if clientVpnEndpointID == "" || targetNetworkCIDR == "" {
		return ClientVpnAuthorizationRuleStatus{}, ErrInvalidParameter
	}
	if !revokeAllGroups && accessGroupID == "" {
		return ClientVpnAuthorizationRuleStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return ClientVpnAuthorizationRuleStatus{}, err
	}
	if endpoint == nil {
		return ClientVpnAuthorizationRuleStatus{}, ErrNotFound
	}

	rules := s.clientVpnAuthorizationRules[endpoint.ID]
	if len(rules) == 0 {
		return ClientVpnAuthorizationRuleStatus{}, ErrNotFound
	}

	if revokeAllGroups {
		removed := false
		for key, rule := range rules {
			if rule.DestinationCIDR == targetNetworkCIDR && rule.AuthorizeAllGroups {
				delete(rules, key)
				removed = true
			}
		}
		if !removed {
			return ClientVpnAuthorizationRuleStatus{}, ErrNotFound
		}
		return ClientVpnAuthorizationRuleStatus{Code: "revoking", Message: "revoking"}, nil
	}

	ruleKey := clientVpnAuthorizationRuleKey(targetNetworkCIDR, accessGroupID, false)
	if rules[ruleKey] == nil {
		return ClientVpnAuthorizationRuleStatus{}, ErrNotFound
	}
	delete(rules, ruleKey)
	return ClientVpnAuthorizationRuleStatus{Code: "revoking", Message: "revoking"}, nil
}

func (s *Service) DescribeClientVpnAuthorizationRules(clientVpnEndpointID string, descriptions, destinationCIDRs, groupIDs []string) ([]ClientVpnAuthorizationRule, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clientVpnEndpoints[clientVpnEndpointID] == nil {
		return nil, ErrNotFound
	}

	descriptionSet := toStringSet(descriptions)
	destinationSet := toStringSet(destinationCIDRs)
	groupIDSet := toStringSet(groupIDs)

	rules := s.clientVpnAuthorizationRules[clientVpnEndpointID]
	out := make([]ClientVpnAuthorizationRule, 0, len(rules))
	for _, rule := range rules {
		if len(descriptionSet) > 0 {
			if _, ok := descriptionSet[rule.Description]; !ok {
				continue
			}
		}
		if len(destinationSet) > 0 {
			if _, ok := destinationSet[rule.DestinationCIDR]; !ok {
				continue
			}
		}
		if len(groupIDSet) > 0 {
			if _, ok := groupIDSet[rule.GroupID]; !ok {
				continue
			}
		}
		out = append(out, cloneClientVpnAuthorizationRule(rule))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DestinationCIDR == out[j].DestinationCIDR {
			return out[i].GroupID < out[j].GroupID
		}
		return out[i].DestinationCIDR < out[j].DestinationCIDR
	})
	return out, nil
}

func (s *Service) DescribeClientVpnConnections(clientVpnEndpointID string, connectionIDs, usernames []string) ([]ClientVpnConnection, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clientVpnEndpoints[clientVpnEndpointID] == nil {
		return nil, ErrNotFound
	}

	connectionIDSet := toStringSet(connectionIDs)
	usernameSet := toStringSet(usernames)

	connections := s.clientVpnConnections[clientVpnEndpointID]
	out := make([]ClientVpnConnection, 0, len(connections))
	for _, connection := range connections {
		if len(connectionIDSet) > 0 {
			if _, ok := connectionIDSet[connection.ConnectionID]; !ok {
				continue
			}
		}
		if len(usernameSet) > 0 {
			if _, ok := usernameSet[connection.Username]; !ok {
				continue
			}
		}
		out = append(out, cloneClientVpnConnection(connection))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ConnectionID < out[j].ConnectionID })
	return out, nil
}

func (s *Service) TerminateClientVpnConnections(clientVpnEndpointID, connectionID, username string) (TerminateClientVpnConnectionsResult, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	connectionID = strings.TrimSpace(connectionID)
	username = strings.TrimSpace(username)
	if clientVpnEndpointID == "" {
		return TerminateClientVpnConnectionsResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return TerminateClientVpnConnectionsResult{}, err
	}
	if endpoint == nil {
		return TerminateClientVpnConnectionsResult{}, ErrNotFound
	}

	connections := s.clientVpnConnections[endpoint.ID]
	targetIDs := make([]string, 0)
	if connectionID != "" {
		if connection := connections[connectionID]; connection != nil {
			targetIDs = append(targetIDs, connectionID)
		}
	} else if username != "" {
		for id, connection := range connections {
			if connection.Username == username {
				targetIDs = append(targetIDs, id)
			}
		}
	} else {
		for id, connection := range connections {
			if connection.Status.Code == "active" {
				targetIDs = append(targetIDs, id)
			}
		}
	}
	sort.Strings(targetIDs)

	now := time.Now().UTC()
	statuses := make([]TerminateClientVpnConnectionStatus, 0, len(targetIDs))
	for _, id := range targetIDs {
		connection := connections[id]
		if connection == nil {
			continue
		}
		previous := connection.Status
		connection.Status = ClientVpnConnectionStatus{Code: "terminated", Message: "terminated"}
		connection.ConnectionEndTime = &now
		connection.Timestamp = now
		statuses = append(statuses, TerminateClientVpnConnectionStatus{
			ConnectionID:   id,
			CurrentStatus:  connection.Status,
			PreviousStatus: previous,
		})
	}

	result := TerminateClientVpnConnectionsResult{
		ClientVpnEndpointID: endpoint.ID,
		ConnectionStatuses:  statuses,
		Username:            username,
	}
	if result.Username == "" && connectionID != "" {
		if connection := connections[connectionID]; connection != nil {
			result.Username = connection.Username
		}
	}

	return result, nil
}

func (s *Service) DescribeClientVpnTargetNetworks(clientVpnEndpointID string, associationIDs, filterAssociationIDs, targetNetworkIDs, vpcIDs []string) ([]ClientVpnTargetNetwork, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clientVpnEndpoints[clientVpnEndpointID] == nil {
		return nil, ErrNotFound
	}

	associationIDSet := toStringSet(associationIDs)
	filterAssociationIDSet := toStringSet(filterAssociationIDs)
	targetNetworkIDSet := toStringSet(targetNetworkIDs)
	vpcIDSet := toStringSet(vpcIDs)

	targetNetworks := s.clientVpnTargetNetworks[clientVpnEndpointID]
	out := make([]ClientVpnTargetNetwork, 0, len(targetNetworks))
	for _, targetNetwork := range targetNetworks {
		if len(associationIDSet) > 0 {
			if _, ok := associationIDSet[targetNetwork.AssociationID]; !ok {
				continue
			}
		}
		if len(filterAssociationIDSet) > 0 {
			if _, ok := filterAssociationIDSet[targetNetwork.AssociationID]; !ok {
				continue
			}
		}
		if len(targetNetworkIDSet) > 0 {
			if _, ok := targetNetworkIDSet[targetNetwork.TargetNetworkID]; !ok {
				continue
			}
		}
		if len(vpcIDSet) > 0 {
			if _, ok := vpcIDSet[targetNetwork.VpcID]; !ok {
				continue
			}
		}
		out = append(out, cloneClientVpnTargetNetwork(targetNetwork))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].AssociationID < out[j].AssociationID })
	return out, nil
}

func (s *Service) ExportClientVpnClientConfiguration(clientVpnEndpointID string) (string, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.clientVpnEndpoints[clientVpnEndpointID]
	if endpoint == nil {
		return "", ErrNotFound
	}

	lines := []string{
		"client",
		"dev tun",
		"proto " + endpoint.TransportProtocol,
		"remote " + endpoint.DnsName + " " + strconv.FormatInt(int64(endpoint.VpnPort), 10),
		"resolv-retry infinite",
		"nobind",
		"persist-key",
		"persist-tun",
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func (s *Service) ExportClientVpnClientCertificateRevocationList(clientVpnEndpointID string) (string, ClientCertificateRevocationListStatus, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	if clientVpnEndpointID == "" {
		return "", ClientCertificateRevocationListStatus{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clientVpnEndpoints[clientVpnEndpointID] == nil {
		return "", ClientCertificateRevocationListStatus{}, ErrNotFound
	}

	status := s.clientVpnCertificateRevocationListStatus[clientVpnEndpointID]
	if status.Code == "" {
		status = ClientCertificateRevocationListStatus{Code: "active", Message: "active"}
	}
	crl := s.clientVpnCertificateRevocationLists[clientVpnEndpointID]
	if strings.TrimSpace(crl) == "" {
		crl = defaultClientVpnCertificateRevocationList()
	}
	return crl, status, nil
}

func (s *Service) ImportClientVpnClientCertificateRevocationList(clientVpnEndpointID, certificateRevocationList string) (bool, error) {
	clientVpnEndpointID = strings.TrimSpace(clientVpnEndpointID)
	certificateRevocationList = strings.TrimSpace(certificateRevocationList)
	if clientVpnEndpointID == "" || certificateRevocationList == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.clientVpnEndpointForMutationLocked(clientVpnEndpointID)
	if err != nil {
		return false, err
	}

	s.clientVpnCertificateRevocationLists[clientVpnEndpointID] = certificateRevocationList
	s.clientVpnCertificateRevocationListStatus[clientVpnEndpointID] = ClientCertificateRevocationListStatus{Code: "active", Message: "active"}
	return true, nil
}

func (s *Service) clientVpnEndpointForMutationLocked(clientVpnEndpointID string) (*ClientVpnEndpoint, error) {
	endpoint := s.clientVpnEndpoints[clientVpnEndpointID]
	if endpoint == nil {
		return nil, ErrNotFound
	}
	if endpoint.Status.Code == "deleted" {
		return nil, ErrConflict
	}
	return endpoint, nil
}

func (s *Service) resolveClientVpnSecurityGroupsLocked(vpcID string, securityGroupIDs []string) ([]string, error) {
	securityGroupIDs = normalizeClientVpnStringList(securityGroupIDs)
	if len(securityGroupIDs) == 0 {
		defaultGroupID := s.defaultSecurityGroupIDForVPCLocked(vpcID)
		if defaultGroupID == "" {
			return []string{}, nil
		}
		return []string{defaultGroupID}, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(securityGroupIDs))
	for _, securityGroupID := range securityGroupIDs {
		group := s.securityGroups[securityGroupID]
		if group == nil {
			return nil, ErrNotFound
		}
		if vpcID != "" && group.VpcID != "" && group.VpcID != vpcID {
			return nil, ErrInvalidParameter
		}
		if _, ok := seen[securityGroupID]; ok {
			continue
		}
		seen[securityGroupID] = struct{}{}
		out = append(out, securityGroupID)
	}
	return out, nil
}

func (s *Service) defaultSecurityGroupIDForVPCLocked(vpcID string) string {
	if vpcID == "" {
		return ""
	}
	if id := s.securityGroupNameIndex[securityGroupNameKey(vpcID, "default")]; id != "" {
		if s.securityGroups[id] != nil {
			return id
		}
	}
	for id, group := range s.securityGroups {
		if group.VpcID == vpcID && group.Name == "default" {
			return id
		}
	}
	return ""
}

func (s *Service) hasAssociatedClientVpnTargetSubnetLocked(clientVpnEndpointID, subnetID string) bool {
	for _, targetNetwork := range s.clientVpnTargetNetworks[clientVpnEndpointID] {
		if targetNetwork.TargetNetworkID == subnetID && targetNetwork.Status.Code == "associated" {
			return true
		}
	}
	return false
}

func (s *Service) hasAssociatedClientVpnTargetNetworkLocked(clientVpnEndpointID string) bool {
	for _, targetNetwork := range s.clientVpnTargetNetworks[clientVpnEndpointID] {
		if targetNetwork.Status.Code == "associated" {
			return true
		}
	}
	return false
}

func isValidClientVpnSessionTimeoutHours(value int32) bool {
	switch value {
	case 8, 10, 12, 24:
		return true
	default:
		return false
	}
}

func normalizeClientVpnSelfServicePortal(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "enabled"
	}
	if normalized != "enabled" && normalized != "disabled" {
		return "", ErrInvalidParameter
	}
	return normalized, nil
}

func normalizeClientVpnStringList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func clientVpnRouteKey(destinationCIDR, targetSubnet string) string {
	return strings.TrimSpace(destinationCIDR) + "|" + strings.TrimSpace(targetSubnet)
}

func clientVpnAuthorizationRuleKey(destinationCIDR, groupID string, authorizeAllGroups bool) string {
	if authorizeAllGroups {
		groupID = "*"
	}
	return strings.TrimSpace(destinationCIDR) + "|" + strings.TrimSpace(groupID)
}

func defaultClientVpnCertificateRevocationList() string {
	return strings.Join([]string{
		"-----BEGIN X509 CRL-----",
		"U1RBQ0tZQVJELUNSTC1QTEFDRUhPTERFUg==",
		"-----END X509 CRL-----",
	}, "\n")
}

func cloneClientVpnEndpoint(in *ClientVpnEndpoint) ClientVpnEndpoint {
	out := *in
	out.DnsServers = append([]string(nil), in.DnsServers...)
	out.SecurityGroupIDs = append([]string(nil), in.SecurityGroupIDs...)
	out.Tags = cloneStringMap(in.Tags)
	if in.DeletionTime != nil {
		timestamp := *in.DeletionTime
		out.DeletionTime = &timestamp
	}
	return out
}

func cloneClientVpnRoute(in *ClientVpnRoute) ClientVpnRoute {
	out := *in
	return out
}

func cloneClientVpnTargetNetwork(in *ClientVpnTargetNetwork) ClientVpnTargetNetwork {
	out := *in
	out.SecurityGroupIDs = append([]string(nil), in.SecurityGroupIDs...)
	return out
}

func cloneClientVpnAuthorizationRule(in *ClientVpnAuthorizationRule) ClientVpnAuthorizationRule {
	out := *in
	return out
}

func cloneClientVpnConnection(in *ClientVpnConnection) ClientVpnConnection {
	out := *in
	out.PostureComplianceStatuses = append([]string(nil), in.PostureComplianceStatuses...)
	if in.ConnectionEndTime != nil {
		timestamp := *in.ConnectionEndTime
		out.ConnectionEndTime = &timestamp
	}
	return out
}
