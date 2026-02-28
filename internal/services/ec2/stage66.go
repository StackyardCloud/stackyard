package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type RouteServerAssociation struct {
	RouteServerID string
	VpcID         string
	State         string
}

type RouteServerPropagation struct {
	RouteServerID string
	RouteTableID  string
	State         string
}

type RouteServerRouteInstallationDetail struct {
	RouteTableID                  string
	RouteInstallationStatus       string
	RouteInstallationStatusReason *string
}

type RouteServerRoute struct {
	AsPaths                  []string
	Med                      *int32
	NextHopIP                string
	Prefix                   string
	RouteInstallationDetails []RouteServerRouteInstallationDetail
	RouteServerEndpointID    string
	RouteServerPeerID        string
	RouteStatus              string
}

func (s *Service) AssociateRouteServer(routeServerID, vpcID string) (RouteServerAssociation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	vpcID = strings.TrimSpace(vpcID)
	if routeServerID == "" || vpcID == "" {
		return RouteServerAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.routeServers[routeServerID] == nil {
		return RouteServerAssociation{}, ErrNotFound
	}
	if s.vpcs[vpcID] == nil {
		return RouteServerAssociation{}, ErrNotFound
	}

	key := routeServerAssociationKey(routeServerID, vpcID)
	assoc := s.routeServerAssociations[key]
	if assoc == nil {
		assoc = &RouteServerAssociation{
			RouteServerID: routeServerID,
			VpcID:         vpcID,
			State:         "associated",
		}
		s.routeServerAssociations[key] = assoc
	} else {
		assoc.State = "associated"
	}
	return cloneRouteServerAssociation(assoc), nil
}

func (s *Service) DisassociateRouteServer(routeServerID, vpcID string) (RouteServerAssociation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	vpcID = strings.TrimSpace(vpcID)
	if routeServerID == "" || vpcID == "" {
		return RouteServerAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := routeServerAssociationKey(routeServerID, vpcID)
	assoc := s.routeServerAssociations[key]
	if assoc == nil {
		return RouteServerAssociation{}, ErrNotFound
	}

	out := cloneRouteServerAssociation(assoc)
	out.State = "disassociating"
	delete(s.routeServerAssociations, key)
	for propagationKey, propagation := range s.routeServerPropagations {
		if propagation != nil && propagation.RouteServerID == routeServerID {
			if routeTable := s.routeTables[propagation.RouteTableID]; routeTable != nil && routeTable.VpcID == vpcID {
				delete(s.routeServerPropagations, propagationKey)
			}
		}
	}
	return out, nil
}

func (s *Service) GetRouteServerAssociations(routeServerID string) ([]RouteServerAssociation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	if routeServerID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.routeServers[routeServerID] == nil {
		return nil, ErrNotFound
	}

	out := make([]RouteServerAssociation, 0)
	for _, assoc := range s.routeServerAssociations {
		if assoc == nil || assoc.RouteServerID != routeServerID {
			continue
		}
		out = append(out, cloneRouteServerAssociation(assoc))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RouteServerID == out[j].RouteServerID {
			return out[i].VpcID < out[j].VpcID
		}
		return out[i].RouteServerID < out[j].RouteServerID
	})
	return out, nil
}

func (s *Service) EnableRouteServerPropagation(routeServerID, routeTableID string) (RouteServerPropagation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	routeTableID = strings.TrimSpace(routeTableID)
	if routeServerID == "" || routeTableID == "" {
		return RouteServerPropagation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.routeServers[routeServerID] == nil {
		return RouteServerPropagation{}, ErrNotFound
	}
	routeTable := s.routeTables[routeTableID]
	if routeTable == nil {
		return RouteServerPropagation{}, ErrNotFound
	}

	assocKey := routeServerAssociationKey(routeServerID, routeTable.VpcID)
	if s.routeServerAssociations[assocKey] == nil {
		return RouteServerPropagation{}, ErrConflict
	}

	key := routeServerPropagationKey(routeServerID, routeTableID)
	prop := s.routeServerPropagations[key]
	if prop == nil {
		prop = &RouteServerPropagation{
			RouteServerID: routeServerID,
			RouteTableID:  routeTableID,
			State:         "available",
		}
		s.routeServerPropagations[key] = prop
	} else {
		prop.State = "available"
	}
	return cloneRouteServerPropagation(prop), nil
}

func (s *Service) DisableRouteServerPropagation(routeServerID, routeTableID string) (RouteServerPropagation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	routeTableID = strings.TrimSpace(routeTableID)
	if routeServerID == "" || routeTableID == "" {
		return RouteServerPropagation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := routeServerPropagationKey(routeServerID, routeTableID)
	prop := s.routeServerPropagations[key]
	if prop == nil {
		return RouteServerPropagation{}, ErrNotFound
	}

	out := cloneRouteServerPropagation(prop)
	out.State = "deleting"
	delete(s.routeServerPropagations, key)
	return out, nil
}

func (s *Service) GetRouteServerPropagations(routeServerID string, routeTableID *string) ([]RouteServerPropagation, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	if routeServerID == "" {
		return nil, ErrInvalidParameter
	}

	filterRouteTableID := ""
	if routeTableID != nil {
		filterRouteTableID = strings.TrimSpace(*routeTableID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.routeServers[routeServerID] == nil {
		return nil, ErrNotFound
	}

	out := make([]RouteServerPropagation, 0)
	for _, prop := range s.routeServerPropagations {
		if prop == nil || prop.RouteServerID != routeServerID {
			continue
		}
		if filterRouteTableID != "" && prop.RouteTableID != filterRouteTableID {
			continue
		}
		out = append(out, cloneRouteServerPropagation(prop))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RouteServerID == out[j].RouteServerID {
			return out[i].RouteTableID < out[j].RouteTableID
		}
		return out[i].RouteServerID < out[j].RouteServerID
	})
	return out, nil
}

func (s *Service) ModifyRouteServer(
	routeServerID string,
	persistRoutesAction string,
	persistRoutesDuration *int64,
	snsNotificationsEnabled *bool,
) (RouteServer, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	if routeServerID == "" {
		return RouteServer{}, ErrInvalidParameter
	}

	action := strings.ToLower(strings.TrimSpace(persistRoutesAction))
	if action != "" {
		switch action {
		case "enable", "disable", "reset":
		default:
			return RouteServer{}, ErrInvalidParameter
		}
	}
	if persistRoutesDuration != nil && (*persistRoutesDuration < 1 || *persistRoutesDuration > 5) {
		return RouteServer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeServer := s.routeServers[routeServerID]
	if routeServer == nil {
		return RouteServer{}, ErrNotFound
	}

	if action != "" {
		if action == "enable" && persistRoutesDuration == nil {
			defaultDuration := int64(1)
			persistRoutesDuration = &defaultDuration
		}
		if action != "enable" {
			persistRoutesDuration = nil
		}
		routeServer.PersistRoutesState = routeServerPersistRoutesStateFromAction(action)
		routeServer.PersistRoutesDuration = cloneInt64Pointer(persistRoutesDuration)
	}
	if persistRoutesDuration != nil {
		routeServer.PersistRoutesDuration = cloneInt64Pointer(persistRoutesDuration)
	}
	if snsNotificationsEnabled != nil {
		routeServer.SnsNotificationsEnabled = *snsNotificationsEnabled
		if *snsNotificationsEnabled {
			routeServer.SnsTopicARN = "arn:aws:sns:" + DefaultRegion + ":" + DefaultAccountID + ":stackyard-route-server-events"
		} else {
			routeServer.SnsTopicARN = ""
		}
	}

	routeServer.State = "available"
	return cloneRouteServer(routeServer), nil
}

func (s *Service) GetRouteServerRoutingDatabase(
	routeServerID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) (bool, []RouteServerRoute, *string, error) {
	routeServerID = strings.TrimSpace(routeServerID)
	if routeServerID == "" {
		return false, nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return false, nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return false, nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	peerIDFilterSet := toStringSet(standardFilters["route-server-peer-id"])
	endpointIDFilterSet := toStringSet(standardFilters["route-server-endpoint-id"])
	prefixFilterSet := toStringSet(standardFilters["prefix"])
	nextHopIPFilterSet := toStringSet(standardFilters["next-hop-ip"])
	routeStatusFilterSet := toLowerStringSet(standardFilters["route-status"])
	routeTableIDFilterSet := toStringSet(standardFilters["route-table-id"])
	routeInstallStatusFilterSet := toLowerStringSet(standardFilters["route-installation-status"])
	asPathFilterSet := toStringSet(standardFilters["as-path"])
	medFilterSet := toStringSet(standardFilters["med"])

	s.mu.Lock()
	defer s.mu.Unlock()

	routeServer := s.routeServers[routeServerID]
	if routeServer == nil {
		return false, nil, nil, ErrNotFound
	}

	propagations := make([]RouteServerPropagation, 0)
	for _, prop := range s.routeServerPropagations {
		if prop == nil || prop.RouteServerID != routeServerID {
			continue
		}
		propagations = append(propagations, cloneRouteServerPropagation(prop))
	}
	sort.Slice(propagations, func(i, j int) bool {
		return propagations[i].RouteTableID < propagations[j].RouteTableID
	})

	items := make([]RouteServerRoute, 0)
	for _, peer := range s.routeServerPeers {
		if peer == nil || peer.RouteServerID != routeServerID {
			continue
		}
		if len(peerIDFilterSet) > 0 {
			if _, ok := peerIDFilterSet[peer.ID]; !ok {
				continue
			}
		}
		if len(endpointIDFilterSet) > 0 {
			if _, ok := endpointIDFilterSet[peer.RouteServerEndpointID]; !ok {
				continue
			}
		}
		if len(nextHopIPFilterSet) > 0 {
			if _, ok := nextHopIPFilterSet[peer.PeerAddress]; !ok {
				continue
			}
		}
		if len(asPathFilterSet) > 0 {
			asn := strconv.FormatInt(peer.BgpOptions.PeerASN, 10)
			if _, ok := asPathFilterSet[asn]; !ok {
				continue
			}
		}

		prefix := routeServerPeerPrefix(peer.ID)
		if len(prefixFilterSet) > 0 {
			if _, ok := prefixFilterSet[prefix]; !ok {
				continue
			}
		}

		med := routeServerPeerMED(peer.ID)
		if len(medFilterSet) > 0 {
			if _, ok := medFilterSet[strconv.Itoa(int(med))]; !ok {
				continue
			}
		}

		status := "in-rib"
		if len(propagations) > 0 {
			status = "in-fib"
		}
		if len(routeStatusFilterSet) > 0 {
			if _, ok := routeStatusFilterSet[status]; !ok {
				continue
			}
		}

		installationDetails := make([]RouteServerRouteInstallationDetail, 0, len(propagations))
		for _, prop := range propagations {
			if len(routeTableIDFilterSet) > 0 {
				if _, ok := routeTableIDFilterSet[prop.RouteTableID]; !ok {
					continue
				}
			}
			installationStatus := "installed"
			if len(routeInstallStatusFilterSet) > 0 {
				if _, ok := routeInstallStatusFilterSet[installationStatus]; !ok {
					continue
				}
			}
			installationDetails = append(installationDetails, RouteServerRouteInstallationDetail{
				RouteTableID:            prop.RouteTableID,
				RouteInstallationStatus: installationStatus,
			})
		}
		if len(routeTableIDFilterSet) > 0 && len(installationDetails) == 0 {
			continue
		}
		if len(routeInstallStatusFilterSet) > 0 && len(installationDetails) == 0 {
			continue
		}

		items = append(items, RouteServerRoute{
			AsPaths:                  []string{strconv.FormatInt(peer.BgpOptions.PeerASN, 10)},
			Med:                      &med,
			NextHopIP:                peer.PeerAddress,
			Prefix:                   prefix,
			RouteInstallationDetails: installationDetails,
			RouteServerEndpointID:    peer.RouteServerEndpointID,
			RouteServerPeerID:        peer.ID,
			RouteStatus:              status,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Prefix == items[j].Prefix {
			return items[i].RouteServerPeerID < items[j].RouteServerPeerID
		}
		return items[i].Prefix < items[j].Prefix
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return false, nil, nil, err
	}

	areRoutesPersisted := strings.EqualFold(routeServer.PersistRoutesState, "enabled")
	return areRoutesPersisted, append([]RouteServerRoute(nil), items[start:end]...), outputToken, nil
}

func cloneRouteServerAssociation(in *RouteServerAssociation) RouteServerAssociation {
	return *in
}

func cloneRouteServerPropagation(in *RouteServerPropagation) RouteServerPropagation {
	return *in
}

func routeServerAssociationKey(routeServerID, vpcID string) string {
	return strings.TrimSpace(routeServerID) + "|" + strings.TrimSpace(vpcID)
}

func routeServerPropagationKey(routeServerID, routeTableID string) string {
	return strings.TrimSpace(routeServerID) + "|" + strings.TrimSpace(routeTableID)
}

func routeServerPeerPrefix(peerID string) string {
	trimmed := strings.TrimSpace(peerID)
	if len(trimmed) == 0 {
		return "10.255.255.0/24"
	}
	hash := 0
	for i := 0; i < len(trimmed); i++ {
		hash += int(trimmed[i])
	}
	octet := (hash % 200) + 1
	return "10." + strconv.Itoa(octet) + ".0.0/24"
}

func routeServerPeerMED(peerID string) int32 {
	trimmed := strings.TrimSpace(peerID)
	if trimmed == "" {
		return 100
	}
	hash := 0
	for i := 0; i < len(trimmed); i++ {
		hash += int(trimmed[i])
	}
	return int32((hash % 300) + 1)
}
