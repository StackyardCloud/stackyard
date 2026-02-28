package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type RouteServer struct {
	ID                      string
	AmazonSideASN           int64
	PersistRoutesState      string
	PersistRoutesDuration   *int64
	SnsNotificationsEnabled bool
	SnsTopicARN             string
	State                   string
	Tags                    map[string]string
}

type RouteServerEndpoint struct {
	ID            string
	RouteServerID string
	SubnetID      string
	VpcID         string
	EniID         string
	EniAddress    string
	FailureReason string
	State         string
	Tags          map[string]string
}

type RouteServerBgpOptions struct {
	PeerASN               int64
	PeerLivenessDetection string
}

type RouteServerBgpOptionsRequest struct {
	PeerASN               int64
	PeerLivenessDetection string
}

type RouteServerPeer struct {
	ID                    string
	RouteServerID         string
	RouteServerEndpointID string
	PeerAddress           string
	BgpOptions            RouteServerBgpOptions
	BgpStatus             string
	BfdStatus             string
	EndpointEniID         string
	EndpointEniAddress    string
	SubnetID              string
	VpcID                 string
	FailureReason         string
	State                 string
	Tags                  map[string]string
}

func (s *Service) CreateRouteServer(
	amazonSideASN int64,
	persistRoutesAction string,
	persistRoutesDuration *int64,
	snsNotificationsEnabled *bool,
	tags []Tag,
) (RouteServer, error) {
	if amazonSideASN <= 0 {
		return RouteServer{}, ErrInvalidParameter
	}

	action := strings.ToLower(strings.TrimSpace(persistRoutesAction))
	if action == "" {
		action = "disable"
	}
	switch action {
	case "enable", "disable", "reset":
	default:
		return RouteServer{}, ErrInvalidParameter
	}

	if persistRoutesDuration != nil && (*persistRoutesDuration < 1 || *persistRoutesDuration > 5) {
		return RouteServer{}, ErrInvalidParameter
	}
	if action == "enable" && persistRoutesDuration == nil {
		defaultDuration := int64(1)
		persistRoutesDuration = &defaultDuration
	}
	if action != "enable" {
		persistRoutesDuration = nil
	}

	snsEnabled := false
	if snsNotificationsEnabled != nil {
		snsEnabled = *snsNotificationsEnabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeServer := &RouteServer{
		ID:                      s.nextIDLocked("rsrv"),
		AmazonSideASN:           amazonSideASN,
		PersistRoutesState:      routeServerPersistRoutesStateFromAction(action),
		PersistRoutesDuration:   cloneInt64Pointer(persistRoutesDuration),
		SnsNotificationsEnabled: snsEnabled,
		State:                   "available",
		Tags:                    tagsToMap(tags),
	}
	if snsEnabled {
		routeServer.SnsTopicARN = fmt.Sprintf("arn:aws:sns:%s:%s:stackyard-route-server-events", DefaultRegion, DefaultAccountID)
	}
	s.routeServers[routeServer.ID] = routeServer
	return cloneRouteServer(routeServer), nil
}

func (s *Service) DeleteRouteServer(routeServerID string) (RouteServer, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	if routeServerID == "" {
		return RouteServer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeServer := s.routeServers[routeServerID]
	if routeServer == nil {
		return RouteServer{}, ErrNotFound
	}
	for _, assoc := range s.routeServerAssociations {
		if assoc != nil && assoc.RouteServerID == routeServerID {
			return RouteServer{}, ErrConflict
		}
	}
	for _, propagation := range s.routeServerPropagations {
		if propagation != nil && propagation.RouteServerID == routeServerID {
			return RouteServer{}, ErrConflict
		}
	}
	for _, endpoint := range s.routeServerEndpoints {
		if endpoint != nil && endpoint.RouteServerID == routeServerID {
			return RouteServer{}, ErrConflict
		}
	}

	out := cloneRouteServer(routeServer)
	out.State = "deleted"
	delete(s.routeServers, routeServerID)
	return out, nil
}

func (s *Service) DescribeRouteServers(
	routeServerIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]RouteServer, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(routeServerIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["route-server-id"])
	filterAmazonASNSet := toStringSet(standardFilters["amazon-side-asn"])
	filterPersistStateSet := toLowerStringSet(standardFilters["persist-routes-state"])
	filterStateSet := toLowerStringSet(standardFilters["state"])
	filterSNSEnabledSet := toLowerStringSet(standardFilters["sns-notifications-enabled"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]RouteServer, 0, len(s.routeServers))
	for _, routeServer := range s.routeServers {
		if routeServer == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[routeServer.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[routeServer.ID]; !ok {
				continue
			}
		}
		if len(filterAmazonASNSet) > 0 {
			if _, ok := filterAmazonASNSet[strconv.FormatInt(routeServer.AmazonSideASN, 10)]; !ok {
				continue
			}
		}
		if len(filterPersistStateSet) > 0 {
			if _, ok := filterPersistStateSet[strings.ToLower(routeServer.PersistRoutesState)]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(routeServer.State)]; !ok {
				continue
			}
		}
		if len(filterSNSEnabledSet) > 0 {
			if _, ok := filterSNSEnabledSet[strconv.FormatBool(routeServer.SnsNotificationsEnabled)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(routeServer.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneRouteServer(routeServer))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]RouteServer(nil), items[start:end]...), outputToken, nil
}

func (s *Service) CreateRouteServerEndpoint(routeServerID, subnetID string, tags []Tag) (RouteServerEndpoint, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	subnetID = strings.TrimSpace(subnetID)
	if routeServerID == "" || subnetID == "" {
		return RouteServerEndpoint{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.routeServers[routeServerID] == nil {
		return RouteServerEndpoint{}, ErrNotFound
	}
	subnet := s.subnets[subnetID]
	if subnet == nil {
		return RouteServerEndpoint{}, ErrNotFound
	}

	endpointID := s.nextIDLocked("rse")
	eniID := s.nextIDLocked("eni")
	hostOctet := int(s.seq%250) + 4
	endpoint := &RouteServerEndpoint{
		ID:            endpointID,
		RouteServerID: routeServerID,
		SubnetID:      subnetID,
		VpcID:         subnet.VpcID,
		EniID:         eniID,
		EniAddress:    fmt.Sprintf("10.0.0.%d", hostOctet),
		State:         "available",
		Tags:          tagsToMap(tags),
	}
	s.routeServerEndpoints[endpoint.ID] = endpoint
	return cloneRouteServerEndpoint(endpoint), nil
}

func (s *Service) DeleteRouteServerEndpoint(routeServerEndpointID string) (RouteServerEndpoint, error) {
	routeServerEndpointID = strings.TrimSpace(routeServerEndpointID)
	if routeServerEndpointID == "" {
		return RouteServerEndpoint{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.routeServerEndpoints[routeServerEndpointID]
	if endpoint == nil {
		return RouteServerEndpoint{}, ErrNotFound
	}
	for _, peer := range s.routeServerPeers {
		if peer != nil && peer.RouteServerEndpointID == routeServerEndpointID {
			return RouteServerEndpoint{}, ErrConflict
		}
	}

	out := cloneRouteServerEndpoint(endpoint)
	out.State = "deleted"
	delete(s.routeServerEndpoints, routeServerEndpointID)
	return out, nil
}

func (s *Service) DescribeRouteServerEndpoints(
	routeServerEndpointIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]RouteServerEndpoint, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(routeServerEndpointIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["route-server-endpoint-id"])
	filterRouteServerIDSet := toStringSet(standardFilters["route-server-id"])
	filterSubnetIDSet := toStringSet(standardFilters["subnet-id"])
	filterVpcIDSet := toStringSet(standardFilters["vpc-id"])
	filterEniIDSet := toStringSet(standardFilters["eni-id"])
	filterEniAddressSet := toStringSet(standardFilters["eni-address"])
	filterStateSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]RouteServerEndpoint, 0, len(s.routeServerEndpoints))
	for _, endpoint := range s.routeServerEndpoints {
		if endpoint == nil {
			continue
		}
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
		if len(filterRouteServerIDSet) > 0 {
			if _, ok := filterRouteServerIDSet[endpoint.RouteServerID]; !ok {
				continue
			}
		}
		if len(filterSubnetIDSet) > 0 {
			if _, ok := filterSubnetIDSet[endpoint.SubnetID]; !ok {
				continue
			}
		}
		if len(filterVpcIDSet) > 0 {
			if _, ok := filterVpcIDSet[endpoint.VpcID]; !ok {
				continue
			}
		}
		if len(filterEniIDSet) > 0 {
			if _, ok := filterEniIDSet[endpoint.EniID]; !ok {
				continue
			}
		}
		if len(filterEniAddressSet) > 0 {
			if _, ok := filterEniAddressSet[endpoint.EniAddress]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(endpoint.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(endpoint.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneRouteServerEndpoint(endpoint))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]RouteServerEndpoint(nil), items[start:end]...), outputToken, nil
}

func (s *Service) CreateRouteServerPeer(
	routeServerEndpointID string,
	peerAddress string,
	bgpOptions RouteServerBgpOptionsRequest,
	tags []Tag,
) (RouteServerPeer, error) {
	routeServerEndpointID = strings.TrimSpace(routeServerEndpointID)
	peerAddress = strings.TrimSpace(peerAddress)
	if routeServerEndpointID == "" || peerAddress == "" || bgpOptions.PeerASN <= 0 {
		return RouteServerPeer{}, ErrInvalidParameter
	}

	peerLivenessDetection := strings.ToLower(strings.TrimSpace(bgpOptions.PeerLivenessDetection))
	if peerLivenessDetection == "" {
		peerLivenessDetection = "bgp-keepalive"
	}
	switch peerLivenessDetection {
	case "bgp-keepalive", "bfd":
	default:
		return RouteServerPeer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.routeServerEndpoints[routeServerEndpointID]
	if endpoint == nil {
		return RouteServerPeer{}, ErrNotFound
	}
	if s.routeServers[endpoint.RouteServerID] == nil {
		return RouteServerPeer{}, ErrNotFound
	}

	bfdStatus := "down"
	if peerLivenessDetection == "bfd" {
		bfdStatus = "up"
	}
	peer := &RouteServerPeer{
		ID:                    s.nextIDLocked("rsp"),
		RouteServerID:         endpoint.RouteServerID,
		RouteServerEndpointID: routeServerEndpointID,
		PeerAddress:           peerAddress,
		BgpOptions: RouteServerBgpOptions{
			PeerASN:               bgpOptions.PeerASN,
			PeerLivenessDetection: peerLivenessDetection,
		},
		BgpStatus:          "up",
		BfdStatus:          bfdStatus,
		EndpointEniID:      endpoint.EniID,
		EndpointEniAddress: endpoint.EniAddress,
		SubnetID:           endpoint.SubnetID,
		VpcID:              endpoint.VpcID,
		State:              "available",
		Tags:               tagsToMap(tags),
	}
	s.routeServerPeers[peer.ID] = peer
	return cloneRouteServerPeer(peer), nil
}

func (s *Service) DeleteRouteServerPeer(routeServerPeerID string) (RouteServerPeer, error) {
	routeServerPeerID = strings.TrimSpace(routeServerPeerID)
	if routeServerPeerID == "" {
		return RouteServerPeer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	peer := s.routeServerPeers[routeServerPeerID]
	if peer == nil {
		return RouteServerPeer{}, ErrNotFound
	}

	out := cloneRouteServerPeer(peer)
	out.State = "deleted"
	delete(s.routeServerPeers, routeServerPeerID)
	return out, nil
}

func (s *Service) DescribeRouteServerPeers(
	routeServerPeerIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]RouteServerPeer, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(routeServerPeerIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDSet := toStringSet(standardFilters["route-server-peer-id"])
	filterRouteServerIDSet := toStringSet(standardFilters["route-server-id"])
	filterRouteServerEndpointIDSet := toStringSet(standardFilters["route-server-endpoint-id"])
	filterPeerAddressSet := toStringSet(standardFilters["peer-address"])
	filterSubnetIDSet := toStringSet(standardFilters["subnet-id"])
	filterVpcIDSet := toStringSet(standardFilters["vpc-id"])
	filterEndpointEniIDSet := toStringSet(standardFilters["endpoint-eni-id"])
	filterEndpointEniAddressSet := toStringSet(standardFilters["endpoint-eni-address"])
	filterStateSet := toLowerStringSet(standardFilters["state"])
	filterBgpStateSet := toLowerStringSet(
		append(append(append([]string{}, standardFilters["bgp-session-state"]...), standardFilters["bgp-state"]...), standardFilters["bgp-status"]...),
	)
	filterBfdStateSet := toLowerStringSet(
		append(append(append([]string{}, standardFilters["bfd-session-state"]...), standardFilters["bfd-state"]...), standardFilters["bfd-status"]...),
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]RouteServerPeer, 0, len(s.routeServerPeers))
	for _, peer := range s.routeServerPeers {
		if peer == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[peer.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[peer.ID]; !ok {
				continue
			}
		}
		if len(filterRouteServerIDSet) > 0 {
			if _, ok := filterRouteServerIDSet[peer.RouteServerID]; !ok {
				continue
			}
		}
		if len(filterRouteServerEndpointIDSet) > 0 {
			if _, ok := filterRouteServerEndpointIDSet[peer.RouteServerEndpointID]; !ok {
				continue
			}
		}
		if len(filterPeerAddressSet) > 0 {
			if _, ok := filterPeerAddressSet[peer.PeerAddress]; !ok {
				continue
			}
		}
		if len(filterSubnetIDSet) > 0 {
			if _, ok := filterSubnetIDSet[peer.SubnetID]; !ok {
				continue
			}
		}
		if len(filterVpcIDSet) > 0 {
			if _, ok := filterVpcIDSet[peer.VpcID]; !ok {
				continue
			}
		}
		if len(filterEndpointEniIDSet) > 0 {
			if _, ok := filterEndpointEniIDSet[peer.EndpointEniID]; !ok {
				continue
			}
		}
		if len(filterEndpointEniAddressSet) > 0 {
			if _, ok := filterEndpointEniAddressSet[peer.EndpointEniAddress]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(peer.State)]; !ok {
				continue
			}
		}
		if len(filterBgpStateSet) > 0 {
			if _, ok := filterBgpStateSet[strings.ToLower(peer.BgpStatus)]; !ok {
				continue
			}
		}
		if len(filterBfdStateSet) > 0 {
			if _, ok := filterBfdStateSet[strings.ToLower(peer.BfdStatus)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(peer.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneRouteServerPeer(peer))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]RouteServerPeer(nil), items[start:end]...), outputToken, nil
}

func routeServerPersistRoutesStateFromAction(action string) string {
	switch action {
	case "enable":
		return "enabled"
	case "reset":
		return "resetting"
	default:
		return "disabled"
	}
}

func cloneRouteServer(in *RouteServer) RouteServer {
	out := *in
	out.PersistRoutesDuration = cloneInt64Pointer(in.PersistRoutesDuration)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneRouteServerEndpoint(in *RouteServerEndpoint) RouteServerEndpoint {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneRouteServerPeer(in *RouteServerPeer) RouteServerPeer {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneInt64Pointer(in *int64) *int64 {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}
