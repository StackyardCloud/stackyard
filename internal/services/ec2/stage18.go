package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type VpnGatewayAttachment struct {
	VpcID string
	State string
}

type VpnGateway struct {
	ID               string
	AmazonSideASN    int64
	AvailabilityZone string
	State            string
	Type             string
	Attachments      []VpnGatewayAttachment
	Tags             map[string]string
}

func (s *Service) CreateVpnGateway(vpnType string, amazonSideASN *int64, availabilityZone string, tags []Tag) (VpnGateway, error) {
	vpnType = strings.ToLower(strings.TrimSpace(vpnType))
	availabilityZone = strings.TrimSpace(availabilityZone)
	if vpnType == "" || vpnType != "ipsec.1" {
		return VpnGateway{}, ErrInvalidParameter
	}

	asn := int64(64512)
	if amazonSideASN != nil {
		asn = *amazonSideASN
		if !isValidAmazonSideASN(asn) {
			return VpnGateway{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := &VpnGateway{
		ID:               s.nextIDLocked("vgw"),
		AmazonSideASN:    asn,
		AvailabilityZone: availabilityZone,
		State:            "available",
		Type:             vpnType,
		Attachments:      []VpnGatewayAttachment{},
		Tags:             tagsToMap(tags),
	}
	s.vpnGateways[gateway.ID] = gateway
	return cloneVpnGateway(gateway), nil
}

func (s *Service) DescribeVpnGateways(
	vpnGatewayIDs, filterGatewayIDs, attachmentVpcIDs, attachmentStates, states, types, availabilityZones, amazonSideAsns, tagKeys []string,
	tagValuesByKey map[string][]string,
) []VpnGateway {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(vpnGatewayIDs)
	filterIDSet := toStringSet(filterGatewayIDs)
	vpcIDSet := toStringSet(attachmentVpcIDs)
	asnSet := toStringSet(amazonSideAsns)
	tagKeySet := toStringSet(tagKeys)
	zoneSet := toStringSet(availabilityZones)

	attachmentStateSet := map[string]struct{}{}
	for _, state := range attachmentStates {
		state = strings.ToLower(strings.TrimSpace(state))
		if state == "" {
			continue
		}
		attachmentStateSet[state] = struct{}{}
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

	out := make([]VpnGateway, 0, len(s.vpnGateways))
	for _, gateway := range s.vpnGateways {
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
		if len(vpcIDSet) > 0 {
			matched := false
			for _, attachment := range gateway.Attachments {
				if _, ok := vpcIDSet[attachment.VpcID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if len(attachmentStateSet) > 0 {
			matched := false
			for _, attachment := range gateway.Attachments {
				if _, ok := attachmentStateSet[strings.ToLower(attachment.State)]; ok {
					matched = true
					break
				}
			}
			if !matched {
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
		if len(zoneSet) > 0 {
			if _, ok := zoneSet[gateway.AvailabilityZone]; !ok {
				continue
			}
		}
		if len(asnSet) > 0 {
			if _, ok := asnSet[strconv.FormatInt(gateway.AmazonSideASN, 10)]; !ok {
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
		out = append(out, cloneVpnGateway(gateway))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AttachVpnGateway(vpnGatewayID, vpcID string) (VpnGatewayAttachment, error) {
	vpnGatewayID = strings.TrimSpace(vpnGatewayID)
	vpcID = strings.TrimSpace(vpcID)
	if vpnGatewayID == "" || vpcID == "" {
		return VpnGatewayAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.vpnGateways[vpnGatewayID]
	if gateway == nil {
		return VpnGatewayAttachment{}, ErrNotFound
	}
	if gateway.State == "deleted" {
		return VpnGatewayAttachment{}, ErrConflict
	}
	if s.vpcs[vpcID] == nil {
		return VpnGatewayAttachment{}, ErrNotFound
	}

	for i := range gateway.Attachments {
		attachment := &gateway.Attachments[i]
		if attachment.State == "attached" && attachment.VpcID != vpcID {
			return VpnGatewayAttachment{}, ErrConflict
		}
		if attachment.VpcID == vpcID {
			attachment.State = "attached"
			return *attachment, nil
		}
	}

	attachment := VpnGatewayAttachment{VpcID: vpcID, State: "attached"}
	gateway.Attachments = append(gateway.Attachments, attachment)
	return attachment, nil
}

func (s *Service) DetachVpnGateway(vpnGatewayID, vpcID string) error {
	vpnGatewayID = strings.TrimSpace(vpnGatewayID)
	vpcID = strings.TrimSpace(vpcID)
	if vpnGatewayID == "" || vpcID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.vpnGateways[vpnGatewayID]
	if gateway == nil {
		return ErrNotFound
	}

	for i := range gateway.Attachments {
		attachment := &gateway.Attachments[i]
		if attachment.VpcID != vpcID {
			continue
		}
		if attachment.State == "detached" {
			return nil
		}
		attachment.State = "detached"
		return nil
	}

	return ErrNotFound
}

func (s *Service) DeleteVpnGateway(vpnGatewayID string) error {
	vpnGatewayID = strings.TrimSpace(vpnGatewayID)
	if vpnGatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.vpnGateways[vpnGatewayID]
	if gateway == nil {
		return ErrNotFound
	}
	if gateway.State == "deleted" {
		return nil
	}
	for _, attachment := range gateway.Attachments {
		if attachment.State == "attached" {
			return ErrConflict
		}
	}
	for _, connection := range s.vpnConnections {
		if connection.State == "deleted" {
			continue
		}
		if connection.VpnGatewayID == vpnGatewayID {
			return ErrConflict
		}
	}
	gateway.State = "deleted"
	return nil
}

func isValidAmazonSideASN(asn int64) bool {
	return (asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294)
}

func cloneVpnGateway(in *VpnGateway) VpnGateway {
	out := *in
	out.Attachments = append([]VpnGatewayAttachment(nil), in.Attachments...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}
