package ec2

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type ActiveVpnTunnelStatus struct {
	IkeVersion                string
	Phase1DHGroup             int32
	Phase1EncryptionAlgorithm string
	Phase1IntegrityAlgorithm  string
	Phase2DHGroup             int32
	Phase2EncryptionAlgorithm string
	Phase2IntegrityAlgorithm  string
	ProvisioningStatus        string
	ProvisioningStatusReason  string
}

type MaintenanceDetails struct {
	LastMaintenanceApplied      *time.Time
	MaintenanceAutoAppliedAfter *time.Time
	PendingMaintenance          string
}

type VpnConnectionDeviceType struct {
	ID       string
	Platform string
	Software string
	Vendor   string
}

type VpnTunnelReplacementStatus struct {
	CustomerGatewayID         string
	MaintenanceDetails        MaintenanceDetails
	TransitGatewayID          string
	VpnConnectionID           string
	VpnGatewayID              string
	VpnTunnelOutsideIPAddress string
}

type ModifyVpnTunnelOptionsRequest struct {
	HasTunnelOptions      bool
	PreSharedKey          *string
	TunnelInsideCidr      *string
	TunnelInsideIpv6Cidr  *string
	PreSharedKeyStorage   *string
	SkipTunnelReplacement *bool
}

var defaultVpnConnectionDeviceTypes = []VpnConnectionDeviceType{
	{ID: "vpn-device-0001", Platform: "ISR", Software: "IOS 15.x", Vendor: "Cisco"},
	{ID: "vpn-device-0002", Platform: "SRX", Software: "Junos 21.x", Vendor: "Juniper"},
	{ID: "vpn-device-0003", Platform: "VM-Series", Software: "PAN-OS 10.x", Vendor: "Palo Alto Networks"},
}

func (s *Service) EnableVgwRoutePropagation(routeTableID, gatewayID string) error {
	routeTableID = strings.TrimSpace(routeTableID)
	gatewayID = strings.TrimSpace(gatewayID)
	if routeTableID == "" || gatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.routeTables[routeTableID]
	if table == nil {
		return ErrNotFound
	}
	gateway := s.vpnGateways[gatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	if gateway.State == "deleted" {
		return ErrConflict
	}
	if !vpnGatewayAttachedToVpcLocked(gateway, table.VpcID) {
		return ErrConflict
	}

	if s.routeTableVgwPropagations[routeTableID] == nil {
		s.routeTableVgwPropagations[routeTableID] = map[string]struct{}{}
	}
	s.routeTableVgwPropagations[routeTableID][gatewayID] = struct{}{}
	return nil
}

func (s *Service) DisableVgwRoutePropagation(routeTableID, gatewayID string) error {
	routeTableID = strings.TrimSpace(routeTableID)
	gatewayID = strings.TrimSpace(gatewayID)
	if routeTableID == "" || gatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.routeTables[routeTableID]
	if table == nil {
		return ErrNotFound
	}
	gateway := s.vpnGateways[gatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	if gateway.State == "deleted" {
		return ErrConflict
	}

	if propagated := s.routeTableVgwPropagations[routeTableID]; propagated != nil {
		delete(propagated, gatewayID)
		if len(propagated) == 0 {
			delete(s.routeTableVgwPropagations, routeTableID)
		}
	}
	return nil
}

func (s *Service) GetActiveVpnTunnelStatus(vpnConnectionID, vpnTunnelOutsideIPAddress string) (ActiveVpnTunnelStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection, outsideIP, err := s.lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress)
	if err != nil {
		return ActiveVpnTunnelStatus{}, err
	}
	status := cloneActiveVpnTunnelStatus(s.ensureActiveVpnTunnelStatusLocked(connection, outsideIP))
	return status, nil
}

func (s *Service) GetVpnConnectionDeviceSampleConfiguration(vpnConnectionID, deviceTypeID, ikeVersion, sampleType string) (string, error) {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	deviceTypeID = strings.TrimSpace(deviceTypeID)
	if vpnConnectionID == "" || deviceTypeID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		s.mu.Unlock()
		return "", ErrNotFound
	}
	if connection.State == "deleted" {
		s.mu.Unlock()
		return "", ErrConflict
	}
	s.mu.Unlock()

	deviceType, ok := vpnConnectionDeviceTypeByID(deviceTypeID)
	if !ok {
		return "", ErrNotFound
	}

	ikeVersion = strings.ToLower(strings.TrimSpace(ikeVersion))
	if ikeVersion == "" {
		ikeVersion = "ikev2"
	}
	sampleType = strings.ToLower(strings.TrimSpace(sampleType))
	if sampleType == "" {
		sampleType = "recommended"
	}

	config := strings.Join([]string{
		"! Stackyard VPN sample configuration",
		"vpn_connection_id=" + vpnConnectionID,
		"vpn_connection_device_type_id=" + deviceType.ID,
		"vendor=" + deviceType.Vendor,
		"platform=" + deviceType.Platform,
		"software=" + deviceType.Software,
		"ike_version=" + ikeVersion,
		"sample_type=" + sampleType,
	}, "\n")
	return config, nil
}

func (s *Service) GetVpnConnectionDeviceTypes(maxResults *int32, nextToken *string) ([]VpnConnectionDeviceType, *string, error) {
	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}

	items := append([]VpnConnectionDeviceType(nil), defaultVpnConnectionDeviceTypes...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if start > len(items) {
		return nil, nil, ErrInvalidParameter
	}

	end := len(items)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(items) {
				end = len(items)
			}
		}
	}

	out := append([]VpnConnectionDeviceType(nil), items[start:end]...)
	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func (s *Service) GetVpnTunnelReplacementStatus(vpnConnectionID, vpnTunnelOutsideIPAddress string) (VpnTunnelReplacementStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection, outsideIP, err := s.lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress)
	if err != nil {
		return VpnTunnelReplacementStatus{}, err
	}
	details := cloneMaintenanceDetails(s.ensureVpnTunnelMaintenanceDetailsLocked(connection, outsideIP))
	return VpnTunnelReplacementStatus{
		CustomerGatewayID:         connection.CustomerGatewayID,
		MaintenanceDetails:        details,
		TransitGatewayID:          connection.TransitGatewayID,
		VpnConnectionID:           connection.ID,
		VpnGatewayID:              connection.VpnGatewayID,
		VpnTunnelOutsideIPAddress: outsideIP,
	}, nil
}

func (s *Service) ModifyVpnTunnelCertificate(vpnConnectionID, vpnTunnelOutsideIPAddress string) (VpnConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection, outsideIP, err := s.lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress)
	if err != nil {
		return VpnConnection{}, err
	}

	now := time.Now().UTC()
	certificateARN := "arn:aws:acm:" + DefaultRegion + ":" + DefaultAccountID + ":certificate/vpn-tunnel-" + connection.ID
	applied := false
	for i := range connection.VgwTelemetry {
		telemetry := &connection.VgwTelemetry[i]
		if strings.TrimSpace(telemetry.OutsideIPAddress) != outsideIP {
			continue
		}
		telemetry.CertificateARN = certificateARN
		telemetry.LastStatusChange = &now
		telemetry.Status = "UP"
		telemetry.StatusMessage = "CERTIFICATE UPDATED"
		applied = true
		break
	}
	if !applied {
		connection.VgwTelemetry = append(connection.VgwTelemetry, VpnConnectionTelemetry{
			AcceptedRouteCount: 0,
			CertificateARN:     certificateARN,
			LastStatusChange:   &now,
			OutsideIPAddress:   outsideIP,
			Status:             "UP",
			StatusMessage:      "CERTIFICATE UPDATED",
		})
	}

	status := s.ensureActiveVpnTunnelStatusLocked(connection, outsideIP)
	status.ProvisioningStatus = "available"
	status.ProvisioningStatusReason = "certificate updated"

	return cloneVpnConnection(connection), nil
}

func (s *Service) ModifyVpnTunnelOptions(
	vpnConnectionID, vpnTunnelOutsideIPAddress string,
	options ModifyVpnTunnelOptionsRequest,
) (VpnConnection, error) {
	if !options.HasTunnelOptions {
		return VpnConnection{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connection, outsideIP, err := s.lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress)
	if err != nil {
		return VpnConnection{}, err
	}

	if options.PreSharedKeyStorage != nil {
		storage := strings.ToLower(strings.TrimSpace(*options.PreSharedKeyStorage))
		if storage != "" && storage != "standard" && storage != "secretsmanager" {
			return VpnConnection{}, ErrInvalidParameter
		}
	}
	if options.PreSharedKey != nil {
		psk := strings.TrimSpace(*options.PreSharedKey)
		if psk == "" {
			return VpnConnection{}, ErrInvalidParameter
		}
	}
	if options.TunnelInsideCidr != nil {
		connection.Options.LocalIpv4NetworkCidr = strings.TrimSpace(*options.TunnelInsideCidr)
	}
	if options.TunnelInsideIpv6Cidr != nil {
		connection.Options.LocalIpv6NetworkCidr = strings.TrimSpace(*options.TunnelInsideIpv6Cidr)
	}

	status := s.ensureActiveVpnTunnelStatusLocked(connection, outsideIP)
	status.ProvisioningStatus = "available"
	status.ProvisioningStatusReason = "tunnel options updated"

	if options.SkipTunnelReplacement != nil && !*options.SkipTunnelReplacement {
		details := s.ensureVpnTunnelMaintenanceDetailsLocked(connection, outsideIP)
		details.PendingMaintenance = "tunnel replacement required"
		auto := time.Now().UTC().Add(2 * time.Hour)
		details.MaintenanceAutoAppliedAfter = &auto
	}

	return cloneVpnConnection(connection), nil
}

func (s *Service) ReplaceVpnTunnel(vpnConnectionID, vpnTunnelOutsideIPAddress string, applyPendingMaintenance bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection, outsideIP, err := s.lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	details := s.ensureVpnTunnelMaintenanceDetailsLocked(connection, outsideIP)
	if applyPendingMaintenance || strings.TrimSpace(details.PendingMaintenance) != "" {
		details.LastMaintenanceApplied = &now
	}
	details.PendingMaintenance = ""
	auto := now.Add(24 * time.Hour)
	details.MaintenanceAutoAppliedAfter = &auto

	for i := range connection.VgwTelemetry {
		telemetry := &connection.VgwTelemetry[i]
		if strings.TrimSpace(telemetry.OutsideIPAddress) != outsideIP {
			continue
		}
		telemetry.LastStatusChange = &now
		telemetry.Status = "UP"
		telemetry.StatusMessage = "TUNNEL REPLACED"
		break
	}

	status := s.ensureActiveVpnTunnelStatusLocked(connection, outsideIP)
	status.ProvisioningStatus = "available"
	status.ProvisioningStatusReason = "tunnel replaced"

	return true, nil
}

func (s *Service) lookupVpnConnectionTunnelLocked(vpnConnectionID, vpnTunnelOutsideIPAddress string) (*VpnConnection, string, error) {
	vpnConnectionID = strings.TrimSpace(vpnConnectionID)
	vpnTunnelOutsideIPAddress = strings.TrimSpace(vpnTunnelOutsideIPAddress)
	if vpnConnectionID == "" || vpnTunnelOutsideIPAddress == "" {
		return nil, "", ErrInvalidParameter
	}

	connection := s.vpnConnections[vpnConnectionID]
	if connection == nil {
		return nil, "", ErrNotFound
	}
	if connection.State == "deleted" {
		return nil, "", ErrConflict
	}

	hasOutsideIP := false
	matched := false
	for _, telemetry := range connection.VgwTelemetry {
		outside := strings.TrimSpace(telemetry.OutsideIPAddress)
		if outside == "" {
			continue
		}
		hasOutsideIP = true
		if outside == vpnTunnelOutsideIPAddress {
			matched = true
			break
		}
	}
	if hasOutsideIP && !matched {
		return nil, "", ErrNotFound
	}
	return connection, vpnTunnelOutsideIPAddress, nil
}

func vpnGatewayAttachedToVpcLocked(gateway *VpnGateway, vpcID string) bool {
	for _, attachment := range gateway.Attachments {
		if attachment.VpcID == vpcID && attachment.State == "attached" {
			return true
		}
	}
	return false
}

func vpnTunnelStateKey(vpnConnectionID, vpnTunnelOutsideIPAddress string) string {
	return vpnConnectionID + "|" + vpnTunnelOutsideIPAddress
}

func (s *Service) ensureActiveVpnTunnelStatusLocked(connection *VpnConnection, outsideIP string) *ActiveVpnTunnelStatus {
	key := vpnTunnelStateKey(connection.ID, outsideIP)
	status := s.vpnActiveTunnelStatuses[key]
	if status != nil {
		return status
	}
	status = &ActiveVpnTunnelStatus{
		IkeVersion:                "ikev2",
		Phase1DHGroup:             14,
		Phase1EncryptionAlgorithm: "AES256",
		Phase1IntegrityAlgorithm:  "SHA2-256",
		Phase2DHGroup:             14,
		Phase2EncryptionAlgorithm: "AES256",
		Phase2IntegrityAlgorithm:  "SHA2-256",
		ProvisioningStatus:        "available",
		ProvisioningStatusReason:  "active",
	}
	s.vpnActiveTunnelStatuses[key] = status
	return status
}

func (s *Service) ensureVpnTunnelMaintenanceDetailsLocked(connection *VpnConnection, outsideIP string) *MaintenanceDetails {
	key := vpnTunnelStateKey(connection.ID, outsideIP)
	details := s.vpnTunnelMaintenanceDetails[key]
	if details != nil {
		return details
	}
	auto := time.Now().UTC().Add(24 * time.Hour)
	details = &MaintenanceDetails{
		MaintenanceAutoAppliedAfter: &auto,
		PendingMaintenance:          "scheduled-maintenance",
	}
	s.vpnTunnelMaintenanceDetails[key] = details
	return details
}

func cloneActiveVpnTunnelStatus(in *ActiveVpnTunnelStatus) ActiveVpnTunnelStatus {
	if in == nil {
		return ActiveVpnTunnelStatus{}
	}
	return *in
}

func cloneMaintenanceDetails(in *MaintenanceDetails) MaintenanceDetails {
	if in == nil {
		return MaintenanceDetails{}
	}
	out := *in
	if in.LastMaintenanceApplied != nil {
		applied := *in.LastMaintenanceApplied
		out.LastMaintenanceApplied = &applied
	}
	if in.MaintenanceAutoAppliedAfter != nil {
		auto := *in.MaintenanceAutoAppliedAfter
		out.MaintenanceAutoAppliedAfter = &auto
	}
	return out
}

func vpnConnectionDeviceTypeByID(deviceTypeID string) (VpnConnectionDeviceType, bool) {
	deviceTypeID = strings.TrimSpace(deviceTypeID)
	for _, deviceType := range defaultVpnConnectionDeviceTypes {
		if deviceType.ID == deviceTypeID {
			return deviceType, true
		}
	}
	return VpnConnectionDeviceType{}, false
}
