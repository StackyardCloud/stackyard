package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type CustomerGateway struct {
	ID             string
	BgpASN         string
	BgpASNExtended string
	CertificateARN string
	DeviceName     string
	IPAddress      string
	State          string
	Type           string
	Tags           map[string]string
}

func (s *Service) CreateCustomerGateway(
	vpnType, ipAddress string,
	bgpASN, bgpASNExtended *int64,
	certificateARN, deviceName string,
	tags []Tag,
) (CustomerGateway, error) {
	vpnType = strings.ToLower(strings.TrimSpace(vpnType))
	ipAddress = strings.TrimSpace(ipAddress)
	certificateARN = strings.TrimSpace(certificateARN)
	deviceName = strings.TrimSpace(deviceName)
	if vpnType == "" || vpnType != "ipsec.1" || ipAddress == "" {
		return CustomerGateway{}, ErrInvalidParameter
	}
	if bgpASN != nil && bgpASNExtended != nil {
		return CustomerGateway{}, ErrInvalidParameter
	}

	asn := int64(65000)
	asnExtended := int64(0)
	if bgpASN != nil {
		asn = *bgpASN
		if asn < 1 || asn > 2147483647 {
			return CustomerGateway{}, ErrInvalidParameter
		}
	}
	if bgpASNExtended != nil {
		asnExtended = *bgpASNExtended
		if asnExtended < 2147483648 || asnExtended > 4294967295 {
			return CustomerGateway{}, ErrInvalidParameter
		}
	}

	asnText := ""
	asnExtendedText := ""
	if bgpASNExtended != nil {
		asnExtendedText = strconv.FormatInt(asnExtended, 10)
	} else {
		asnText = strconv.FormatInt(asn, 10)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, gateway := range s.customerGateways {
		if gateway.State == "deleted" {
			continue
		}
		if gateway.Type == vpnType &&
			gateway.IPAddress == ipAddress &&
			gateway.BgpASN == asnText &&
			gateway.BgpASNExtended == asnExtendedText &&
			gateway.CertificateARN == certificateARN &&
			gateway.DeviceName == deviceName {
			return cloneCustomerGateway(gateway), nil
		}
	}

	gateway := &CustomerGateway{
		ID:             s.nextIDLocked("cgw"),
		BgpASN:         asnText,
		BgpASNExtended: asnExtendedText,
		CertificateARN: certificateARN,
		DeviceName:     deviceName,
		IPAddress:      ipAddress,
		State:          "available",
		Type:           vpnType,
		Tags:           tagsToMap(tags),
	}
	s.customerGateways[gateway.ID] = gateway
	return cloneCustomerGateway(gateway), nil
}

func (s *Service) DescribeCustomerGateways(
	customerGatewayIDs, filterGatewayIDs, ipAddresses, states, types, bgpAsns, tagKeys []string,
	tagValuesByKey map[string][]string,
) []CustomerGateway {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(customerGatewayIDs)
	filterIDSet := toStringSet(filterGatewayIDs)
	ipSet := toStringSet(ipAddresses)
	bgpASNSet := toStringSet(bgpAsns)
	tagKeySet := toStringSet(tagKeys)

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

	out := make([]CustomerGateway, 0, len(s.customerGateways))
	for _, gateway := range s.customerGateways {
		if len(idSet) > 0 {
			if _, ok := idSet[gateway.ID]; !ok {
				continue
			}
		}
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[gateway.ID]; !ok {
				continue
			}
		}
		if len(ipSet) > 0 {
			if _, ok := ipSet[gateway.IPAddress]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(gateway.State)]; !ok {
				continue
			}
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[strings.ToLower(gateway.Type)]; !ok {
				continue
			}
		}
		if len(bgpASNSet) > 0 {
			matched := false
			if gateway.BgpASN != "" {
				if _, ok := bgpASNSet[gateway.BgpASN]; ok {
					matched = true
				}
			}
			if gateway.BgpASNExtended != "" {
				if _, ok := bgpASNSet[gateway.BgpASNExtended]; ok {
					matched = true
				}
			}
			if !matched {
				continue
			}
		}
		if len(tagKeySet) > 0 {
			matched := false
			for key := range tagKeySet {
				if _, ok := gateway.Tags[key]; ok {
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
				value, ok := gateway.Tags[key]
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
		out = append(out, cloneCustomerGateway(gateway))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteCustomerGateway(customerGatewayID string) error {
	customerGatewayID = strings.TrimSpace(customerGatewayID)
	if customerGatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.customerGateways[customerGatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	for _, connection := range s.vpnConnections {
		if connection.State == "deleted" {
			continue
		}
		if connection.CustomerGatewayID == customerGatewayID {
			return ErrConflict
		}
	}
	gateway.State = "deleted"
	return nil
}

func cloneCustomerGateway(in *CustomerGateway) CustomerGateway {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
