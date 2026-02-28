package ec2

import (
	"net/netip"
	"sort"
	"strings"
)

type TransitGatewayMulticastGroupRegistration struct {
	GroupIpAddress                  string
	GroupMember                     bool
	GroupSource                     bool
	MemberType                      string
	NetworkInterfaceID              string
	ResourceID                      string
	ResourceOwnerID                 string
	ResourceType                    string
	SourceType                      string
	SubnetID                        string
	TransitGatewayAttachmentID      string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayMulticastRegisteredGroupMembers struct {
	GroupIpAddress                  string
	RegisteredNetworkInterfaceIDs   []string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayMulticastRegisteredGroupSources struct {
	GroupIpAddress                  string
	RegisteredNetworkInterfaceIDs   []string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayMulticastDeregisteredGroupMembers struct {
	DeregisteredNetworkInterfaceIDs []string
	GroupIpAddress                  string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayMulticastDeregisteredGroupSources struct {
	DeregisteredNetworkInterfaceIDs []string
	GroupIpAddress                  string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayMulticastGroup struct {
	GroupIpAddress             string
	GroupMember                bool
	GroupSource                bool
	MemberType                 string
	NetworkInterfaceID         string
	ResourceID                 string
	ResourceOwnerID            string
	ResourceType               string
	SourceType                 string
	SubnetID                   string
	TransitGatewayAttachmentID string
}

func (s *Service) RegisterTransitGatewayMulticastGroupMembers(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
) (TransitGatewayMulticastRegisteredGroupMembers, error) {
	registered, err := s.registerTransitGatewayMulticastGroupEntries(
		transitGatewayMulticastDomainID,
		groupIPAddress,
		networkInterfaceIDs,
		true,
	)
	if err != nil {
		return TransitGatewayMulticastRegisteredGroupMembers{}, err
	}
	return TransitGatewayMulticastRegisteredGroupMembers{
		GroupIpAddress:                  groupIPAddress,
		RegisteredNetworkInterfaceIDs:   registered,
		TransitGatewayMulticastDomainID: strings.TrimSpace(transitGatewayMulticastDomainID),
	}, nil
}

func (s *Service) RegisterTransitGatewayMulticastGroupSources(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
) (TransitGatewayMulticastRegisteredGroupSources, error) {
	registered, err := s.registerTransitGatewayMulticastGroupEntries(
		transitGatewayMulticastDomainID,
		groupIPAddress,
		networkInterfaceIDs,
		false,
	)
	if err != nil {
		return TransitGatewayMulticastRegisteredGroupSources{}, err
	}
	return TransitGatewayMulticastRegisteredGroupSources{
		GroupIpAddress:                  groupIPAddress,
		RegisteredNetworkInterfaceIDs:   registered,
		TransitGatewayMulticastDomainID: strings.TrimSpace(transitGatewayMulticastDomainID),
	}, nil
}

func (s *Service) DeregisterTransitGatewayMulticastGroupMembers(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
) (TransitGatewayMulticastDeregisteredGroupMembers, error) {
	deregistered, err := s.deregisterTransitGatewayMulticastGroupEntries(
		transitGatewayMulticastDomainID,
		groupIPAddress,
		networkInterfaceIDs,
		true,
	)
	if err != nil {
		return TransitGatewayMulticastDeregisteredGroupMembers{}, err
	}
	return TransitGatewayMulticastDeregisteredGroupMembers{
		DeregisteredNetworkInterfaceIDs: deregistered,
		GroupIpAddress:                  strings.TrimSpace(groupIPAddress),
		TransitGatewayMulticastDomainID: strings.TrimSpace(transitGatewayMulticastDomainID),
	}, nil
}

func (s *Service) DeregisterTransitGatewayMulticastGroupSources(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
) (TransitGatewayMulticastDeregisteredGroupSources, error) {
	deregistered, err := s.deregisterTransitGatewayMulticastGroupEntries(
		transitGatewayMulticastDomainID,
		groupIPAddress,
		networkInterfaceIDs,
		false,
	)
	if err != nil {
		return TransitGatewayMulticastDeregisteredGroupSources{}, err
	}
	return TransitGatewayMulticastDeregisteredGroupSources{
		DeregisteredNetworkInterfaceIDs: deregistered,
		GroupIpAddress:                  strings.TrimSpace(groupIPAddress),
		TransitGatewayMulticastDomainID: strings.TrimSpace(transitGatewayMulticastDomainID),
	}, nil
}

func (s *Service) SearchTransitGatewayMulticastGroups(
	transitGatewayMulticastDomainID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayMulticastGroup, *string, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	if transitGatewayMulticastDomainID == "" {
		return nil, nil, ErrInvalidParameter
	}
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, _, _ := splitEC2Filters(filters)
	groupIPSet := toStringSet(standard["group-ip-address"])
	memberTypeSet := toLowerStringSet(standard["member-type"])
	resourceIDSet := toStringSet(standard["resource-id"])
	resourceTypeSet := toLowerStringSet(standard["resource-type"])
	sourceTypeSet := toLowerStringSet(standard["source-type"])
	subnetIDSet := toStringSet(standard["subnet-id"])
	transitGatewayAttachmentIDSet := toStringSet(standard["transit-gateway-attachment-id"])
	groupMemberFilter, hasGroupMemberFilter, ok := parseEC2BooleanFilterSet(standard["is-group-member"])
	if !ok {
		return nil, nil, ErrInvalidParameter
	}
	groupSourceFilter, hasGroupSourceFilter, ok := parseEC2BooleanFilterSet(standard["is-group-source"])
	if !ok {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayMulticastDomains[transitGatewayMulticastDomainID] == nil {
		return nil, nil, ErrNotFound
	}

	out := make([]TransitGatewayMulticastGroup, 0)
	for _, group := range s.transitGatewayMulticastGroups {
		if group.TransitGatewayMulticastDomainID != transitGatewayMulticastDomainID {
			continue
		}
		if len(groupIPSet) > 0 {
			if _, ok := groupIPSet[group.GroupIpAddress]; !ok {
				continue
			}
		}
		if hasGroupMemberFilter && group.GroupMember != groupMemberFilter {
			continue
		}
		if hasGroupSourceFilter && group.GroupSource != groupSourceFilter {
			continue
		}
		if len(memberTypeSet) > 0 {
			if _, ok := memberTypeSet[strings.ToLower(group.MemberType)]; !ok {
				continue
			}
		}
		if len(resourceIDSet) > 0 {
			if _, ok := resourceIDSet[group.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceTypeSet) > 0 {
			if _, ok := resourceTypeSet[strings.ToLower(group.ResourceType)]; !ok {
				continue
			}
		}
		if len(sourceTypeSet) > 0 {
			if _, ok := sourceTypeSet[strings.ToLower(group.SourceType)]; !ok {
				continue
			}
		}
		if len(subnetIDSet) > 0 {
			if _, ok := subnetIDSet[group.SubnetID]; !ok {
				continue
			}
		}
		if len(transitGatewayAttachmentIDSet) > 0 {
			if _, ok := transitGatewayAttachmentIDSet[group.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}

		out = append(out, TransitGatewayMulticastGroup{
			GroupIpAddress:             group.GroupIpAddress,
			GroupMember:                group.GroupMember,
			GroupSource:                group.GroupSource,
			MemberType:                 group.MemberType,
			NetworkInterfaceID:         group.NetworkInterfaceID,
			ResourceID:                 group.ResourceID,
			ResourceOwnerID:            group.ResourceOwnerID,
			ResourceType:               group.ResourceType,
			SourceType:                 group.SourceType,
			SubnetID:                   group.SubnetID,
			TransitGatewayAttachmentID: group.TransitGatewayAttachmentID,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupIpAddress != out[j].GroupIpAddress {
			return out[i].GroupIpAddress < out[j].GroupIpAddress
		}
		if out[i].NetworkInterfaceID != out[j].NetworkInterfaceID {
			return out[i].NetworkInterfaceID < out[j].NetworkInterfaceID
		}
		if out[i].GroupMember != out[j].GroupMember {
			return out[i].GroupMember
		}
		return out[i].GroupSource
	})

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayMulticastGroup(nil), out[start:end]...), outputToken, nil
}

func (s *Service) registerTransitGatewayMulticastGroupEntries(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
	member bool,
) ([]string, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	groupIPAddress = strings.TrimSpace(groupIPAddress)
	resolvedNetworkInterfaceIDs := dedupeTrimmedStrings(networkInterfaceIDs)
	if transitGatewayMulticastDomainID == "" || groupIPAddress == "" || len(resolvedNetworkInterfaceIDs) == 0 {
		return nil, ErrInvalidParameter
	}
	if _, err := netip.ParseAddr(groupIPAddress); err != nil {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain := s.transitGatewayMulticastDomains[transitGatewayMulticastDomainID]
	if domain == nil {
		return nil, ErrNotFound
	}

	registered := make([]string, 0, len(resolvedNetworkInterfaceIDs))
	for _, networkInterfaceID := range resolvedNetworkInterfaceIDs {
		iface := s.networkInterfaces[networkInterfaceID]
		if iface == nil {
			return nil, ErrNotFound
		}

		key := transitGatewayMulticastGroupKey(transitGatewayMulticastDomainID, groupIPAddress, networkInterfaceID)
		entry := s.transitGatewayMulticastGroups[key]
		if entry == nil {
			entry = &TransitGatewayMulticastGroupRegistration{
				GroupIpAddress:                  groupIPAddress,
				NetworkInterfaceID:              networkInterfaceID,
				ResourceID:                      networkInterfaceID,
				ResourceOwnerID:                 DefaultAccountID,
				ResourceType:                    "vpc",
				SubnetID:                        iface.SubnetID,
				TransitGatewayAttachmentID:      s.resolveTransitGatewayMulticastAttachmentIDLocked(domain.TransitID, iface.SubnetID),
				TransitGatewayMulticastDomainID: transitGatewayMulticastDomainID,
			}
			s.transitGatewayMulticastGroups[key] = entry
		}
		if member {
			entry.GroupMember = true
			entry.MemberType = "static"
		} else {
			entry.GroupSource = true
			entry.SourceType = "static"
		}
		registered = append(registered, networkInterfaceID)
	}

	sort.Strings(registered)
	return registered, nil
}

func (s *Service) deregisterTransitGatewayMulticastGroupEntries(
	transitGatewayMulticastDomainID,
	groupIPAddress string,
	networkInterfaceIDs []string,
	member bool,
) ([]string, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	groupIPAddress = strings.TrimSpace(groupIPAddress)
	resolvedNetworkInterfaceIDs := dedupeTrimmedStrings(networkInterfaceIDs)
	if transitGatewayMulticastDomainID == "" || groupIPAddress == "" || len(resolvedNetworkInterfaceIDs) == 0 {
		return nil, ErrInvalidParameter
	}
	if _, err := netip.ParseAddr(groupIPAddress); err != nil {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayMulticastDomains[transitGatewayMulticastDomainID] == nil {
		return nil, ErrNotFound
	}

	deregistered := make([]string, 0, len(resolvedNetworkInterfaceIDs))
	for _, networkInterfaceID := range resolvedNetworkInterfaceIDs {
		key := transitGatewayMulticastGroupKey(transitGatewayMulticastDomainID, groupIPAddress, networkInterfaceID)
		entry := s.transitGatewayMulticastGroups[key]
		if entry == nil {
			continue
		}

		if member {
			if !entry.GroupMember {
				continue
			}
			entry.GroupMember = false
			entry.MemberType = ""
		} else {
			if !entry.GroupSource {
				continue
			}
			entry.GroupSource = false
			entry.SourceType = ""
		}
		deregistered = append(deregistered, networkInterfaceID)

		if !entry.GroupMember && !entry.GroupSource {
			delete(s.transitGatewayMulticastGroups, key)
		}
	}

	sort.Strings(deregistered)
	return deregistered, nil
}

func (s *Service) resolveTransitGatewayMulticastAttachmentIDLocked(transitGatewayID, subnetID string) string {
	for _, attachment := range s.transitGatewayVpcAttachments {
		if attachment.TransitGatewayID != transitGatewayID {
			continue
		}
		for _, candidateSubnetID := range attachment.SubnetIDs {
			if candidateSubnetID == subnetID {
				return attachment.ID
			}
		}
	}
	return ""
}

func parseEC2BooleanFilterSet(values []string) (bool, bool, bool) {
	if len(values) == 0 {
		return false, false, true
	}
	parsed := false
	set := false
	for _, raw := range values {
		value := strings.TrimSpace(strings.ToLower(raw))
		switch value {
		case "", "false", "0", "no", "off":
			if set && parsed {
				return false, false, false
			}
			parsed = false
			set = true
		case "true", "1", "yes", "on":
			if set && !parsed {
				return false, false, false
			}
			parsed = true
			set = true
		default:
			return false, false, false
		}
	}
	return parsed, set, true
}

func transitGatewayMulticastGroupKey(transitGatewayMulticastDomainID, groupIPAddress, networkInterfaceID string) string {
	return strings.TrimSpace(transitGatewayMulticastDomainID) + "|" + strings.TrimSpace(groupIPAddress) + "|" + strings.TrimSpace(networkInterfaceID)
}

func (s *Service) deleteTransitGatewayMulticastGroupsForDomainLocked(transitGatewayMulticastDomainID string) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	if transitGatewayMulticastDomainID == "" {
		return
	}
	for key, group := range s.transitGatewayMulticastGroups {
		if group.TransitGatewayMulticastDomainID == transitGatewayMulticastDomainID {
			delete(s.transitGatewayMulticastGroups, key)
		}
	}
}

func (s *Service) deleteTransitGatewayMulticastGroupsForNetworkInterfaceLocked(networkInterfaceID string) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	if networkInterfaceID == "" {
		return
	}
	for key, group := range s.transitGatewayMulticastGroups {
		if group.NetworkInterfaceID == networkInterfaceID {
			delete(s.transitGatewayMulticastGroups, key)
		}
	}
}
