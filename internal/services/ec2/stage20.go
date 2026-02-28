package ec2

import "strings"

func (s *Service) ModifyVpnConnection(vpnConnectionID, customerGatewayID, vpnGatewayID, transitGatewayID string) (VpnConnection, error) {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	customerGatewayID = strings.TrimSpace(customerGatewayID)
	vpnGatewayID = strings.TrimSpace(vpnGatewayID)
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if vpnConnectionID == "" {
		return VpnConnection{}, ErrInvalidParameter
	}
	if vpnGatewayID != "" && transitGatewayID != "" {
		return VpnConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return VpnConnection{}, ErrNotFound
	}
	if connection.State == "deleted" {
		return VpnConnection{}, ErrConflict
	}

	if customerGatewayID != "" {
		customerGateway := s.customerGateways[customerGatewayID]
		if customerGateway == nil {
			return VpnConnection{}, ErrNotFound
		}
		if customerGateway.State == "deleted" {
			return VpnConnection{}, ErrConflict
		}
		connection.CustomerGatewayID = customerGatewayID
		connection.CustomerGatewayConfiguration = buildCustomerGatewayConfiguration(
			connection.CustomerGatewayID,
			connection.VpnGatewayID,
			connection.TransitGatewayID,
		)
		if len(connection.VgwTelemetry) > 0 {
			connection.VgwTelemetry[0].OutsideIPAddress = customerGateway.IPAddress
		}
	}

	if vpnGatewayID != "" {
		vpnGateway := s.vpnGateways[vpnGatewayID]
		if vpnGateway == nil {
			return VpnConnection{}, ErrNotFound
		}
		if vpnGateway.State == "deleted" {
			return VpnConnection{}, ErrConflict
		}
		connection.VpnGatewayID = vpnGatewayID
		connection.TransitGatewayID = ""
		connection.GatewayAssociationState = "associated"
		connection.CustomerGatewayConfiguration = buildCustomerGatewayConfiguration(
			connection.CustomerGatewayID,
			connection.VpnGatewayID,
			connection.TransitGatewayID,
		)
	}

	if transitGatewayID != "" {
		connection.TransitGatewayID = transitGatewayID
		connection.VpnGatewayID = ""
		connection.GatewayAssociationState = "associated"
		connection.CustomerGatewayConfiguration = buildCustomerGatewayConfiguration(
			connection.CustomerGatewayID,
			connection.VpnGatewayID,
			connection.TransitGatewayID,
		)
	}

	return cloneVpnConnection(connection), nil
}

func (s *Service) ModifyVpnConnectionOptions(
	vpnConnectionID string,
	localIpv4NetworkCidr, localIpv6NetworkCidr, remoteIpv4NetworkCidr, remoteIpv6NetworkCidr *string,
) (VpnConnection, error) {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	if vpnConnectionID == "" {
		return VpnConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return VpnConnection{}, ErrNotFound
	}
	if connection.State == "deleted" {
		return VpnConnection{}, ErrConflict
	}

	if localIpv4NetworkCidr != nil {
		connection.Options.LocalIpv4NetworkCidr = strings.TrimSpace(*localIpv4NetworkCidr)
	}
	if localIpv6NetworkCidr != nil {
		connection.Options.LocalIpv6NetworkCidr = strings.TrimSpace(*localIpv6NetworkCidr)
	}
	if remoteIpv4NetworkCidr != nil {
		connection.Options.RemoteIpv4NetworkCidr = strings.TrimSpace(*remoteIpv4NetworkCidr)
	}
	if remoteIpv6NetworkCidr != nil {
		connection.Options.RemoteIpv6NetworkCidr = strings.TrimSpace(*remoteIpv6NetworkCidr)
	}

	return cloneVpnConnection(connection), nil
}
