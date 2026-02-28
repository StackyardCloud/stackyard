package ec2

import (
	"sort"
	"strings"
	"time"
)

type VpnConnectionTelemetry struct {
	AcceptedRouteCount int32
	CertificateARN     string
	LastStatusChange   *time.Time
	OutsideIPAddress   string
	Status             string
	StatusMessage      string
}

type VpnConnectionRoute struct {
	DestinationCIDRBlock string
	Source               string
	State                string
}

type VpnConnectionOptions struct {
	StaticRoutesOnly      bool
	LocalIpv4NetworkCidr  string
	LocalIpv6NetworkCidr  string
	RemoteIpv4NetworkCidr string
	RemoteIpv6NetworkCidr string
}

type VpnConnection struct {
	ID                           string
	CustomerGatewayConfiguration string
	CustomerGatewayID            string
	GatewayAssociationState      string
	Options                      VpnConnectionOptions
	Routes                       []VpnConnectionRoute
	State                        string
	Tags                         map[string]string
	TransitGatewayID             string
	Type                         string
	VgwTelemetry                 []VpnConnectionTelemetry
	VpnGatewayID                 string
}

func (s *Service) CreateVpnConnection(
	customerGatewayID, vpnType, vpnGatewayID, transitGatewayID string,
	staticRoutesOnly *bool,
	tags []Tag,
) (VpnConnection, error) {
	customerGatewayID = strings.TrimSpace(customerGatewayID)
	vpnType = strings.ToLower(strings.TrimSpace(vpnType))
	vpnGatewayID = strings.TrimSpace(vpnGatewayID)
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if customerGatewayID == "" || vpnType != "ipsec.1" {
		return VpnConnection{}, ErrInvalidParameter
	}
	if (vpnGatewayID == "" && transitGatewayID == "") || (vpnGatewayID != "" && transitGatewayID != "") {
		return VpnConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	customerGateway := s.customerGateways[customerGatewayID]
	if customerGateway == nil {
		return VpnConnection{}, ErrNotFound
	}
	if customerGateway.State == "deleted" {
		return VpnConnection{}, ErrConflict
	}
	if vpnGatewayID != "" {
		vpnGateway := s.vpnGateways[vpnGatewayID]
		if vpnGateway == nil {
			return VpnConnection{}, ErrNotFound
		}
		if vpnGateway.State == "deleted" {
			return VpnConnection{}, ErrConflict
		}
	}

	for _, connection := range s.vpnConnections {
		if connection.State == "deleted" {
			continue
		}
		if connection.CustomerGatewayID == customerGatewayID &&
			connection.Type == vpnType &&
			connection.VpnGatewayID == vpnGatewayID &&
			connection.TransitGatewayID == transitGatewayID {
			return cloneVpnConnection(connection), nil
		}
	}

	routeMode := false
	if staticRoutesOnly != nil {
		routeMode = *staticRoutesOnly
	}

	now := time.Now().UTC()
	telemetry := []VpnConnectionTelemetry{
		{
			AcceptedRouteCount: 0,
			LastStatusChange:   &now,
			OutsideIPAddress:   customerGateway.IPAddress,
			Status:             "UP",
			StatusMessage:      "IPSEC IS UP",
		},
	}

	associationState := "not-associated"
	if vpnGatewayID != "" {
		associationState = "associated"
	}

	connection := &VpnConnection{
		ID:                           s.nextIDLocked("vpn"),
		CustomerGatewayConfiguration: buildCustomerGatewayConfiguration(customerGatewayID, vpnGatewayID, transitGatewayID),
		CustomerGatewayID:            customerGatewayID,
		GatewayAssociationState:      associationState,
		Options: VpnConnectionOptions{
			StaticRoutesOnly:      routeMode,
			LocalIpv4NetworkCidr:  "0.0.0.0/0",
			LocalIpv6NetworkCidr:  "::/0",
			RemoteIpv4NetworkCidr: "0.0.0.0/0",
			RemoteIpv6NetworkCidr: "::/0",
		},
		Routes:           []VpnConnectionRoute{},
		State:            "available",
		Tags:             tagsToMap(tags),
		TransitGatewayID: transitGatewayID,
		Type:             vpnType,
		VgwTelemetry:     telemetry,
		VpnGatewayID:     vpnGatewayID,
	}
	s.vpnConnections[connection.ID] = connection
	return cloneVpnConnection(connection), nil
}

func (s *Service) DescribeVpnConnections(
	vpnConnectionIDs, filterConnectionIDs, customerGatewayIDs, states, types, vpnGatewayIDs, transitGatewayIDs, customerGatewayConfigurations, routeDestinationCIDRs, bgpAsns, tagKeys []string,
	staticRoutesOnly []bool,
	tagValuesByKey map[string][]string,
) []VpnConnection {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(vpnConnectionIDs)
	filterIDSet := toStringSet(filterConnectionIDs)
	customerGatewayIDSet := toStringSet(customerGatewayIDs)
	vpnGatewayIDSet := toStringSet(vpnGatewayIDs)
	transitGatewayIDSet := toStringSet(transitGatewayIDs)
	customerGatewayConfigSet := toStringSet(customerGatewayConfigurations)
	routeDestinationSet := toStringSet(routeDestinationCIDRs)
	bgpASNSet := toStringSet(bgpAsns)
	tagKeySet := toStringSet(tagKeys)

	staticRouteOnlySet := map[bool]struct{}{}
	for _, value := range staticRoutesOnly {
		staticRouteOnlySet[value] = struct{}{}
	}

	stateSet := map[string]struct{}{}
	for _, state := range states {
		state = strings.ToLower(strings.TrimSpace(state))
		if state == "" {
			continue
		}
		stateSet[state] = struct{}{}
	}

	typeSet := map[string]struct{}{}
	for _, vpnType := range types {
		vpnType = strings.ToLower(strings.TrimSpace(vpnType))
		if vpnType == "" {
			continue
		}
		typeSet[vpnType] = struct{}{}
	}

	tagValueFilters := make(map[string]map[string]struct{}, len(tagValuesByKey))
	for key, values := range tagValuesByKey {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		valueSet := map[string]struct{}{}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			valueSet[value] = struct{}{}
		}
		if len(valueSet) == 0 {
			continue
		}
		tagValueFilters[key] = valueSet
	}

	out := make([]VpnConnection, 0, len(s.vpnConnections))
	for _, connection := range s.vpnConnections {
		if len(idSet) > 0 {
			if _, ok := idSet[connection.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[connection.ID]; !ok {
				continue
			}
		}
		if len(customerGatewayIDSet) > 0 {
			if _, ok := customerGatewayIDSet[connection.CustomerGatewayID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(connection.State)]; !ok {
				continue
			}
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[strings.ToLower(connection.Type)]; !ok {
				continue
			}
		}
		if len(vpnGatewayIDSet) > 0 {
			if _, ok := vpnGatewayIDSet[connection.VpnGatewayID]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[connection.TransitGatewayID]; !ok {
				continue
			}
		}
		if len(customerGatewayConfigSet) > 0 {
			if _, ok := customerGatewayConfigSet[connection.CustomerGatewayConfiguration]; !ok {
				continue
			}
		}
		if len(routeDestinationSet) > 0 {
			matched := false
			for _, route := range connection.Routes {
				if _, ok := routeDestinationSet[route.DestinationCIDRBlock]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if len(bgpASNSet) > 0 {
			matched := false
			customerGateway := s.customerGateways[connection.CustomerGatewayID]
			if customerGateway != nil {
				if customerGateway.BgpASN != "" {
					if _, ok := bgpASNSet[customerGateway.BgpASN]; ok {
						matched = true
					}
				}
				if customerGateway.BgpASNExtended != "" {
					if _, ok := bgpASNSet[customerGateway.BgpASNExtended]; ok {
						matched = true
					}
				}
			}
			if !matched {
				continue
			}
		}
		if len(staticRouteOnlySet) > 0 {
			if _, ok := staticRouteOnlySet[connection.Options.StaticRoutesOnly]; !ok {
				continue
			}
		}
		if len(tagKeySet) > 0 {
			matched := false
			for key := range tagKeySet {
				if _, ok := connection.Tags[key]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if len(tagValueFilters) > 0 {
			matched := true
			for key, valueSet := range tagValueFilters {
				value, ok := connection.Tags[key]
				if !ok {
					matched = false
					break
				}
				if _, ok := valueSet[value]; !ok {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cloneVpnConnection(connection))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteVpnConnection(vpnConnectionID string) error {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	if vpnConnectionID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return ErrNotFound
	}
	if connection.State == "deleted" {
		return nil
	}
	connection.State = "deleted"
	connection.GatewayAssociationState = "not-associated"
	return nil
}

func (s *Service) CreateVpnConnectionRoute(vpnConnectionID, destinationCIDR string) error {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	if vpnConnectionID == "" || destinationCIDR == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return ErrNotFound
	}
	if connection.State == "deleted" {
		return ErrConflict
	}

	for i := range connection.Routes {
		route := &connection.Routes[i]
		if route.DestinationCIDRBlock != destinationCIDR {
			continue
		}
		route.Source = "Static"
		route.State = "available"
		s.refreshVpnConnectionTelemetryLocked(connection)
		return nil
	}

	connection.Routes = append(connection.Routes, VpnConnectionRoute{
		DestinationCIDRBlock: destinationCIDR,
		Source:               "Static",
		State:                "available",
	})
	s.refreshVpnConnectionTelemetryLocked(connection)
	return nil
}

func (s *Service) DeleteVpnConnectionRoute(vpnConnectionID, destinationCIDR string) error {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	if vpnConnectionID == "" || destinationCIDR == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return ErrNotFound
	}
	if connection.State == "deleted" {
		return ErrConflict
	}

	removed := false
	out := make([]VpnConnectionRoute, 0, len(connection.Routes))
	for _, route := range connection.Routes {
		if route.DestinationCIDRBlock == destinationCIDR {
			removed = true
			continue
		}
		out = append(out, route)
	}
	if !removed {
		return ErrNotFound
	}
	connection.Routes = out
	s.refreshVpnConnectionTelemetryLocked(connection)
	return nil
}

func (s *Service) refreshVpnConnectionTelemetryLocked(connection *VpnConnection) {
	if len(connection.VgwTelemetry) == 0 {
		now := time.Now().UTC()
		connection.VgwTelemetry = append(connection.VgwTelemetry, VpnConnectionTelemetry{
			LastStatusChange: &now,
			Status:           "UP",
			StatusMessage:    "IPSEC IS UP",
		})
	}
	if len(connection.VgwTelemetry) == 0 {
		return
	}
	count := int32(0)
	for _, route := range connection.Routes {
		if strings.EqualFold(route.State, "available") {
			count++
		}
	}
	connection.VgwTelemetry[0].AcceptedRouteCount = count
}

func buildCustomerGatewayConfiguration(customerGatewayID, vpnGatewayID, transitGatewayID string) string {
	target := vpnGatewayID
	if target == "" {
		target = transitGatewayID
	}
	if target == "" {
		target = "gateway"
	}
	return "<vpn_connection><customer_gateway_id>" + customerGatewayID + "</customer_gateway_id><target_gateway_id>" + target + "</target_gateway_id></vpn_connection>"
}

func parseBoolFilterValues(values []string) []bool {
	out := make([]bool, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		switch value {
		case "1", "true", "yes", "on":
			out = append(out, true)
		case "0", "false", "no", "off":
			out = append(out, false)
		}
	}
	return out
}

func cloneVpnConnection(in *VpnConnection) VpnConnection {
	out := *in
	out.Routes = append([]VpnConnectionRoute(nil), in.Routes...)
	out.VgwTelemetry = append([]VpnConnectionTelemetry(nil), in.VgwTelemetry...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}
